/*
Package bam deals with BAM file creation or manipulation.

BAM Creator is released under the BSD 2-clause license. See LICENSE in the project's root folder for more details.
*/
package bam

import (
  "errors"
  "fmt"
  "hash/fnv"
  "image"
  "image/color"
  "io"
  "os"
  "runtime"
  "sync"
  // "time"

  "github.com/InfinityTools/go-logging"
  "github.com/InfinityTools/go-ietools/buffers"
  "github.com/InfinityTools/go-ietools/pvrz"
  "github.com/pbenner/threadpool"
)

const (
  // Supported BAM output formats
  BAM_V1    = 1
  BAM_V2    = 2

  // RLE frame encoding modes (BAM V1)
  RLE_OFF   = 0     // never encode
  RLE_ON    = 1     // always encode
  RLE_AUTO  = -1    // use encoding only if frame size is smaller

  // Available PVRZ encoding types (based on PVR pixel format codes)
  PVRZ_AUTO       = -1    // determine DXT1 or DXT5 compression from input data
  PVRZ_DXT1       = 7
  PVRZ_DXT3       = 9     // currently not supported by the games, only added for completeness.
  PVRZ_DXT5       = 11
  // PVRZ_PVRTC_2BPP = 1
  // PVRZ_PVRTC_4BPP = 3
  // PVRZ_ETC1       = 6

  // Available pixel encoding quality modes (BAM V2)
  QUALITY_LOW         = 0   // Encode with lowest possible quality
  QUALITY_DEFAULT     = 1   // Encode with a sensible quality/speed ratio
  QUALITY_HIGH        = 2   // Encoed with highest possiblle quality

  // Helps addressing individual bytes in an int32 value
  MASK_BYTE1  = uint32(0x000000ff)
  MASK_BYTE2  = uint32(0x0000ff00)
  MASK_BYTE3  = uint32(0x00ff0000)
  MASK_BYTE4  = uint32(0xff000000)

  // Used internally. Available BAM signature and version strings
  sig_bam   = "BAM "
  sig_bamc  = "BAMC"
  ver_v1    = "V1  "
  ver_v2    = "V2  "
)

// Stores BAM V1 specific settings
type BamV1Config struct {
  // BAM V1 options
  compress      bool          // whether to create a zlib-compressed BAMC file
  rle           int           // whether to apply RLE compression to transparent color indices in frames (0: no, 1: yes, -1: only if smaller)
  discardAlpha  bool          // whether to discard alpha channel from palette entries
  transColor    byte          // the transparent color index for RLE frame encoding (usually 0)
  // Quantization options
  qualityMin    int           // imagequant mimimum quality [0, 100]
  qualityMax    int           // imagequant maxmimum quality [0, 100]
  speed         int           // imagequant speed setting [1, 10]
  dither        float32       // dither value [0.0, 1.0]
  fixedColors   []color.Color // list of fixed color entries
  sortFlags     int           // Palette sorting type and options
  colorKey      color.Color   // the color to treat as transparent if colorKeyEnabled == true
  colorKeyEnabled bool        // whether colorKey is treated as transparent
  palette       color.Palette // an optional external palette that is used in place of color quantization
}

// Stores BAM V2 specific settings
type BamV2Config struct {
  pvrzStart       int       // PVRZ start index
  pvrzType        int       // PVRZ pixel encoding (see PVRZ_xxx constants)
  quality         int       // 0: lowest, 1: average, 2: highest
  weightAlpha     bool      // whether to weight pixels by alpha (for better quality)
  useMetric       bool      // use perceptual metric
  threshold       float32   // alpha threshold (in percent) to exceed before using DXT5 compression (if pvrzType == PVRZ_AUTO)
}

// Defines a single BAM cycle structure
type BamCycle []uint16

// Defines a single BAM frame structure. Buffer data is always uncompressed.
type BamFrame struct {
  cx, cy int                // center x and center y of the frame
  img image.Image           // the BAM frame pixel data
}

// Main BAM structure
type BamFile struct {
  // part of the BAM structure
  frames        []BamFrame    // list of frame definitions
  cycles        []BamCycle    // list of cycle definitions
  filters       []BamFilter   // chain of filters that will be applied on export (see bam_filter.go)

  // internal settings
  err           error     // stores error from last BAM-related operation
  optimization  int       // optimization level
  bamVersion    int       // 1: BAM V1, 2: BAM V2
  bamV1         BamV1Config
  bamV2         BamV2Config
}


// CreateNew initializes an empty BAM structure and returns a pointer to it.
func CreateNew() *BamFile {
  bam := BamFile{ frames: make([]BamFrame, 0),
                  cycles: make([]BamCycle, 0),
                  err: nil,
                  optimization: 0,
                  bamVersion: BAM_V1,
                  bamV1: BamV1Config { compress: false, rle: RLE_AUTO, discardAlpha: false, transColor: 0,
                                       qualityMin: 80, qualityMax: 100, speed: 3, dither: 0.0, fixedColors: make([]color.Color, 0),
                                       sortFlags: 0, colorKey: color.RGBA{0, 255, 0, 255}, colorKeyEnabled: false, palette: nil},
                  bamV2: BamV2Config { pvrzStart: 1000, pvrzType: PVRZ_AUTO, quality: QUALITY_DEFAULT, weightAlpha: true, useMetric: false, threshold: 0.0 },
                }
  return &bam
}


// Import imports data from the source BAM pointed to by the Reader. Returns a fully initialized BAM structure when
// successful.
//
// If BAM V2 type is detected the function will search the current directory for associated pvrz files.
// Use function Error() to check if Import returned successfully.
func Import(r io.Reader) *BamFile {
  path, err := os.Getwd()
  if err != nil { path = "." }  // should never fail
  return ImportEx(r, []string{path})
}

// ImportEx imports data from the source BAM pointed to by the Reader and looks for associated pvrz files in the
// specified search paths.
//
// Use function Error() to check if Import returned successfully.
func ImportEx(r io.Reader, searchPaths []string) *BamFile {
  logging.Logln("Importing BAM")
  bam := CreateNew()

  buf := buffers.Load(r)
  if buf.Error() != nil { bam.err = buf.Error(); return bam }

  // decoding BAM structures
  bam.decodeBam(buf, searchPaths)
  logging.Logln("Finished importing BAM")
  return bam
}


// Export writes the current BAM structure to the buffer addressed by the given Writer object.
// Pvrz files associated with BAM V2 type will be created in the current directory.
// Does nothing if the BamFile is in an invalid state (see Error() function).
func (bam *BamFile) Export(w io.Writer) {
  path, err := os.Getwd()
  if err != nil { path = "." }  // should never fail
  bam.ExportEx(w, path)
}

// ExportEx writes the current BAM structure to the buffer addressed by the given Writer object and associated pvrz files
// to the given specified path. Does nothing if the BamFile is in an invalid state (see Error() function).
func (bam *BamFile) ExportEx(w io.Writer, outPath string) {
  if bam.err != nil { return }
  logging.Logln("Exporting BAM")

  // Writing bam file
  buf, pvrMap := bam.encodeBam(outPath)
  if bam.err != nil { return }
  _, bam.err = w.Write(buf)

  // Writing pvrz files
  bam.exportPvrz(pvrMap, outPath, GetMultiThreaded())

  logging.Logln("Finished exporting BAM")
}


// Error returns the error state of the most recent operation on the BamFile. Use ClearError() function to clear the
// current error state.
func (bam *BamFile) Error() error {
  return bam.err
}


// ClearError clears the error state from the last BamFile operation. This function must be called for subsequent
// operations to work correctly.
//
// Use this function with care. Functions may leave the BamFile object in an incomplete state after returning
// unsuccessfully.
func (bam *BamFile) ClearError() {
  bam.err = nil
}


// GetOptimization returns the optimization level to use when generating the output BAM file. Operation is skipped if
// error state is set.
func (bam *BamFile) GetOptimization() int {
  if bam.err != nil { return 0 }
  return bam.optimization
}

// SetOptimization sets the optimization level to use when generating the output BAM file. Operation is skipped if
// error state is set. Available optimization levels:
//   0    Apply no further optimizations.
//   1    Remove unreferenced frames.
//   2    Remove duplicate frames and update cycle lists.
//   3    Remove similar frames and update cycle lists.
// Levels are cumulative.
func (bam *BamFile) SetOptimization(level int) {
  if bam.err != nil { return }
  if level < 0 { level = 0 }
  bam.optimization = level
}


// GetBamVersion returns the desired BAM version to create when calling Export(). Supported versions: BAM_V1 and BAM_V2.
// Operation is skipped if error state is set.
func (bam *BamFile) GetBamVersion() int {
  if bam.err != nil { return 0 }
  return bam.bamVersion
}


// SetBamVersion sets the desired BAM version to create when calling Export(). Only BAM_V1 and BAM_V2 are supported.
// Operation is skipped if error state is set.
func (bam *BamFile) SetBamVersion(version int) {
  if bam.err != nil { return }
  if version != BAM_V1 && version != BAM_V2 { bam.err = fmt.Errorf("Unsupported BAM version: %d", version); return }
  bam.bamVersion = version
}


// GetFrameLength returns the number of frames in the current BAM structure. Operation is skipped if error state is set.
func (bam *BamFile) GetFrameLength() int {
  if bam.err != nil { return 0 }
  return len(bam.frames)
}


// GetFrameWidth returns the height of the frame at given index. Operation is skipped if error state is set.
func (bam *BamFile) GetFrameWidth(index int) int {
  if bam.err != nil { return 0 }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("GetFrameWidth: Index out of range (%d)", index); return 0 }
  return bam.frames[index].img.Bounds().Dx()
}


// GetFrameHeight returns the height of the frame at given index. Operation is skipped if error state is set.
func (bam *BamFile) GetFrameHeight(index int) int {
  if bam.err != nil { return 0 }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("GetFrameHeight: Index out of range (%d)", index); return 0 }
  return bam.frames[index].img.Bounds().Dy()
}


// GetFrameCenterX returns the horizontal center position of the frame at given index. Operation is skipped if error
// state is set.
func (bam *BamFile) GetFrameCenterX(index int) int {
  if bam.err != nil { return 0 }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("GetFrameCenterX: Index out of range (%d)", index); return 0 }
  return bam.frames[index].cx
}


// SetFrameCenterX sets a new horizontal center position for the specified frame. Operation is skipped if error state
// is set.
func (bam *BamFile) SetFrameCenterX(index int, value int) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("SetFrameCenterX: Index out of range (%d)", index); return }
  bam.frames[index].cx = value
}


// GetFrameCenterY returns the vertical center position of the frame at given index. Operation is skipped if error
// state is set.
func (bam *BamFile) GetFrameCenterY(index int) int {
  if bam.err != nil { return 0 }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("GetFrameCenterY: Index out of range (%d)", index); return 0 }
  return bam.frames[index].cy
}


// SetFrameCenterX sets a new vertical center position for the specified frame. Operation is skipped if error state is
// set.
func (bam *BamFile) SetFrameCenterY(index int, value int) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("SetFrameCenterY: Index out of range (%d)", index); return }
  bam.frames[index].cy = value
}


// GetFrameImage returns the image object attached to the frame at the given index. Operation is skipped if error
// state is set.
func (bam *BamFile) GetFrameImage(index int) image.Image {
  if bam.err != nil { return nil }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("GetFrameImage: Index out of range (%d)", index); return nil }
  return bam.frames[index].img
}


// SetFrame replaces the frame at the given frame index with the provided data. Operation is skipped if error state is
// set.
func (bam *BamFile) SetFrame(index, centerX, centerY int, img image.Image) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("SetFrame: Index out of range (%d)", index); return }
  if img == nil { bam.err = fmt.Errorf("SetFrame: Frame is undefined"); return }
  if img.Bounds().Dx() > 65535 || img.Bounds().Dy() > 65535 { bam.err = fmt.Errorf("SetFrame %d: Size too big (w=%d,h=%d)", index, img.Bounds().Dx(), img.Bounds().Dy()); return }
  if centerX < -32768 || centerX > 32767 || centerY < -32768 || centerY > 32767 { bam.err = fmt.Errorf("SetFrame %d: Center out of range (x=%d,y=%d)", index, centerX, centerY); return }

  bam.frames[index].cx = centerX
  bam.frames[index].cy = centerY
  bam.frames[index].img = img
}


// DeleteFrame removes the frame entry at the given index.
// Set "adjust" to update cycle lists accordingly, i.e. remove references to the deleted frame and adjust higher
// frame indices.
// Note: The adjustment may result in empty cycle list. Make sure to fill or remove them before exporting the BAM
// structure.
// Operation is skipped if error state is set.
func (bam *BamFile) DeleteFrame(index int, adjust bool) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.frames) { bam.err = fmt.Errorf("DeleteFrame: Index out of range (%d)", index); return }

  for idx := index + 1; idx < len(bam.frames); idx++ {
    bam.frames[idx - 1] = bam.frames[idx]
  }
  bam.frames = bam.frames[:len(bam.frames) - 1]

  if adjust {
    bam.adjustCycles(index, -1)
  }
}


// InsertFrame inserts a new frame entry at the given position and assigns it the specified frame data.
// Set "adjust" to update cycle lists accordingly, i.e. adjust frame indices of same or higher index.
// Operation is skipped if error state is set.
func (bam *BamFile) InsertFrame(index, centerX, centerY int, img image.Image, adjust bool) {
  if bam.err != nil { return }
  if index < 0 || index > len(bam.frames) { bam.err = fmt.Errorf("InsertFrame: Index out of range (%d)", index); return }

  bam.frames = append(bam.frames, make([]BamFrame, 1)...)
  for idx := len(bam.frames) - 1; idx > index; idx-- {
    bam.frames[idx] = bam.frames[idx - 1]
  }
  bam.SetFrame(index, centerX, centerY, img)
  if bam.err != nil {
    err := bam.err
    bam.err = nil
    bam.DeleteFrame(index, false)
    bam.err = err
    return
  }
  if adjust {
    bam.adjustCycles(index, 1)
  }
}


// AddFrame appends a new frame entry to the list of frames. Returns the index of the added frame.
// Operation is skipped if error state is set.
func (bam *BamFile) AddFrame(centerX, centerY int, img image.Image) int {
  if bam.err != nil { return 0 }
  retVal := len(bam.frames)
  bam.InsertFrame(retVal, centerX, centerY, img, false)
  return retVal
}


// GetFrameLength returns the number of cycles in the current BAM structure. Operation is skipped if error state is set.
func (bam *BamFile) GetCycleLength() int {
  if bam.err != nil { return 0 }
  return len(bam.cycles)
}


// GetCycle returns a copy of the array of indices of the specified cycle. Operation is skipped if error state is set.
func (bam *BamFile) GetCycle(index int) []uint16 {
  if bam.err != nil { return make([]uint16, 0) }
  if index < 0 || index >= len(bam.cycles) { bam.err = fmt.Errorf("GetCycle: Index out of range (%d)", index); return make([]uint16, 0) }

  retVal := make([]uint16, len(bam.cycles[index]))
  copy(retVal, bam.cycles[index])
  return retVal
}


// SetCycle replaces the cycle array at the given cycle index with the provided data. Operation is skipped if error
// state is set.
// Notes: Cycle array must contain at least one frame index.
func (bam *BamFile) SetCycle(index int, cycle []uint16) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.cycles) { bam.err = fmt.Errorf("SetCycle: Cycle index out of range (%d)", index); return }
  if cycle == nil || len(cycle) == 0 { bam.err = fmt.Errorf("SetCycle: Cycle entry index out of range (%d)", cycle); return }

  bam.cycles[index] = make([]uint16, len(cycle))
  copy(bam.cycles[index], cycle)
}


// DeleteCycle removes the cycle at the given position. Operation is skipped if error state is set.
func (bam *BamFile) DeleteCycle(index int) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.cycles) { bam.err = fmt.Errorf("DeleteCycle: Index out of range (%d)", index); return }

  for idx := index + 1; idx < len(bam.cycles); idx++ {
    bam.cycles[idx - 1] = bam.cycles[idx]
  }
  bam.cycles = bam.cycles[:len(bam.cycles) - 1]
}


// InsertCycle insert a new cycle entry at the given position and assigns to it the specified cycle array.
// index must be in range [0, NumberOfCycles]. Cycle array must contain at least one frame index.
// Operation is skipped if error state is set.
func (bam *BamFile) InsertCycle(index int, cycle []uint16) {
  if bam.err != nil { return }
  if index < 0 || index > len(bam.cycles) { bam.err = fmt.Errorf("InsertCycle: Cycle index out of range (%d)", index); return }
  if cycle == nil || len(cycle) == 0 { bam.err = fmt.Errorf("InsertCycle: Cycle entry index out of range (%d)", cycle); return }

  bam.cycles = append(bam.cycles, make([]BamCycle, 1)...)
  for idx := len(bam.cycles) - 1; idx > index; idx-- {
    bam.cycles[idx] = bam.cycles[idx - 1]
  }
  bam.cycles[index] = make([]uint16, len(cycle))
  copy(bam.cycles[index], cycle)
}


// AddCycle appends a new cycle entry entry to the list of cycles. Returns the index of the added cycle array.
// Operation is skipped if error state is set.
func (bam *BamFile) AddCycle(cycle []uint16) int {
  if bam.err != nil { return 0 }
  retVal := len(bam.cycles)
  bam.InsertCycle(retVal, cycle)
  return retVal
}


// GetFilterLength returns the number of explicitly stored filter entries. Operation is skipped if error state is set.
func (bam *BamFile) GetFilterLength() int {
  if bam.err != nil { return 0 }
  return len(bam.filters)
}


// GetFilter returns the filter at the specified index. Operation is skipped if error state is set.
func (bam *BamFile) GetFilter(index int) BamFilter {
  if bam.err != nil { return nil }
  if index < 0 || index >= len(bam.filters) { bam.err = fmt.Errorf("GetFilter: Index out of range (%d)", index); return nil }

  return bam.filters[index]
}


// SetFilter replaces the filter at the given index with the specified filter.
// Operation is skipped if error state is set.
func (bam *BamFile) SetFilter(index int, filter BamFilter) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.filters) { bam.err = fmt.Errorf("SetFilter: Index out of range (%d)", index); return }
  if filter == nil { bam.err = fmt.Errorf("SetFilter: Filter is undefined"); return }

  bam.filters[index] = filter
}


// DeleteFilter removes the filter entry at the given index. Operation is skipped if error state is set.
func (bam *BamFile) DeleteFilter(index int) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.filters) { bam.err = fmt.Errorf("DeleteFilter: Index out of range (%d)", index); return }

  for idx := index + 1; idx < len(bam.filters); idx++ {
    bam.filters[idx - 1] = bam.filters[idx]
  }
  bam.filters = bam.filters[:len(bam.filters) - 1]
}


// InsertFilter inserts a new filter at the given index. index must be in range [0, GetFilterLength()].
// Operation is skipped if error state is set.
func (bam *BamFile) InsertFilter(index int, filter BamFilter) {
  if bam.err != nil { return }
  if filter == nil || index < 0 || index > len(bam.filters) { bam.err = fmt.Errorf("InsertFilter: Index out of range (%d)", index); return }

  bam.filters = append(bam.filters, nil)
  for idx := len(bam.filters) - 1; idx > index; idx-- {
    bam.filters[idx] = bam.filters[idx - 1]
  }
  bam.filters[index] = filter
}


// AddFilter appends the specified filter to the filter list and returns filter index. Operation is skipped if error
// state is set.
func (bam *BamFile) AddFilter(filter BamFilter) int {
  if bam.err != nil { return -1 }
  if filter == nil { return -1 }

  retVal := len(bam.filters)
  bam.InsertFilter(retVal, filter)
  return retVal
}


// Used internally. Delegates BAM import to more specialized functions.
func (bam *BamFile) decodeBam(buf *buffers.Buffer, searchPaths []string) {
  if buf.BufferLength() < 8 { bam.err = errors.New("BAM structure size too small"); return }

  // determining BAM type
  sig := buf.GetString(0, 4, false)
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ver := buf.GetString(4, 4, false)
  if buf.Error() != nil { bam.err = buf.Error(); return }

  if (sig == sig_bamc || sig == sig_bam) && ver == ver_v1 {
    bam.decodeBamV1(buf)
  } else if sig == sig_bam && ver == ver_v2 {
    bam.decodeBamV2(buf, searchPaths)
  } else {
    bam.err = errors.New("Invalid BAM signature")
  }
}


// Used internally. Delegates BAM export to more specialized functions.
func (bam *BamFile) encodeBam(outPath string) ([]byte, map[int]*pvrz.Pvr) {
  switch bam.bamVersion {
    case BAM_V1:
      return bam.encodeBamV1(), nil
    case BAM_V2:
      return bam.encodeBamV2(outPath)
    default:
      bam.err = errors.New("Unsupported BAM type specified")
      return nil, nil
  }
}


// Used internally. Adjusts cycle lists to account for a single inserted or removed frame entry.
func (bam *BamFile) adjustCycles(index, shift int) {
  if index < 0 || index >= len(bam.frames) { return }
  if shift != -1 && shift != 1 { return }

  if shift < 0 {
    // removal
    for i := 0; i < len(bam.cycles); i++ {
      cycle := bam.cycles[i]
      for j := 0; j < len(cycle); j++ {
        if cycle[j] == uint16(index) {
          for k := j + 1; k < len(cycle); k++ {
            cycle[k - 1] = cycle[k]
          }
          cycle = cycle[:len(cycle) - 1]
          j-- // ensure next pass doesn't skip a frame index
        } else if cycle[j] > uint16(index) {
          cycle[j]--
        }
      }
      bam.cycles[i] = cycle
    }
  } else {
    // insertion
    for i := 0; i < len(bam.cycles); i++ {
      cycle := bam.cycles[i]
      for j := 0; j < len(cycle); j++ {
        if cycle[j] >= uint16(index) {
          cycle[j]++
        }
      }
    }
  }
}


// Used internally. Creates pvrz files from pvr objects in pvrMap.
func (bam *BamFile) exportPvrz(pvrMap bamV2TextureMap, outPath string, multithreaded bool) {
  if pvrMap != nil {
    progressIdx, progressMax := 0, len(pvrMap)

    // t0 := time.Now()
    logging.Log("Writing PVRZ files")
    if multithreaded {
      // Multi-threaded operation
      numThreads := runtime.NumCPU()
      pool := threadpool.NewThreadPool(numThreads, len(pvrMap))
      g := pool.NewJobGroup()
      var m sync.Mutex
      for page, pvr := range pvrMap {
        pvrzPath := generatePvrzFileName(outPath, page)
        pvrLocal := pvr
        if len(pvrzPath) == 0 { bam.err = fmt.Errorf("Cannot export PVRZ file: %q", pvrzPath); break }
        if err := pool.AddJob(g, func(pool threadpool.ThreadPool, erf func() error) error {
          if erf() != nil { return nil }
          fout, err := os.Create(pvrzPath)
          if err != nil { bam.err = err; return err }
          defer fout.Close()
          pvrLocal.Save(fout, true)
          if pvrLocal.Error() != nil { err = pvrLocal.Error(); return err }
          func() {
            m.Lock()
            defer m.Unlock()
            logging.LogProgressDot(progressIdx, progressMax, 79 - 18)    // 18 is length of prefix string above
            progressIdx++
          }()
          return nil
        }); err != nil { break }
      }
      if err := pool.Wait(g); err != nil { bam.err = err; return }
      if bam.err != nil { return }
      pool.Stop()
    } else {
      // Single-threaded operation
      for page, pvr := range pvrMap {
        pvrzPath := generatePvrzFileName(outPath, page)
        if len(pvrzPath) == 0 { bam.err = fmt.Errorf("Cannot export PVRZ file: %q", pvrzPath); return }
        fout, err := os.Create(pvrzPath)
        if err != nil { bam.err = err; return }
        defer fout.Close()
        pvr.Save(fout, true)
        if pvr.Error() != nil { err = pvr.Error(); return }
        logging.LogProgressDot(progressIdx, progressMax, 79 - 18)    // 18 is length of prefix string
        progressIdx++
      }
    }
    logging.OverridePrefix(false, false, false).Logln("")
    // t1 := time.Now()
    // fmt.Printf("DEBUG: ExportEx() timing = %v\n", t1.Sub(t0))
  }
}


// Used internally. Applies optimizations based on current optimization level.
func (bam *BamFile) optimize() (frames []BamFrame, cycles []BamCycle) {
  if bam.optimization == 0 {
    frames, cycles = bam.frames, bam.cycles
    return
  }

  // Making deep copy of frames and cycles
  frames = make([]BamFrame, len(bam.frames))
  copy(frames, bam.frames)
  cycles = make([]BamCycle, len(bam.cycles))
  for i, cycle := range bam.cycles {
    cycles[i] = make(BamCycle, len(cycle))
    copy(cycles[i], cycle)
  }

  // Remove unused frames
  // 1. Registering frame references
  cycleSet := make(map[uint16]bool)
  for _, cycle := range cycles {
    for _, ref := range cycle {
      cycleSet[ref] = true
    }
  }

  // 2. Removing frames not referenced in cycles
  for idx := len(frames) - 1; idx >= 0; idx-- {
    if _, ok := cycleSet[uint16(idx)]; !ok {
      // Frame not references -> remove frame
      logging.Logf("Removing unreferenced frame: %d\n", idx)
      if idx < len(frames) - 1 { copy(frames[idx:], frames[idx+1:]) }
      frames = frames[:len(frames) - 1]
      // Updating cycles
      for _, cycle := range cycles {
        for i := 0; i < len(cycle); i++ {
          if cycle[i] > uint16(idx) { cycle[i]-- }
        }
      }
    }
  }
  if bam.optimization == 1 { return }


  // Remove duplicate frames
  // 1. Generating list of hashes
  hashes := make([]uint64, len(frames))
  for i, frame := range frames { hashes[i] = getFrameHash(frame) }

  // 2. Removing frames with identical hashes
  for i := 0; i < len(frames); i++ {
    v1 := hashes[i]
    for j := len(frames) - 1; j > i; j-- {
      if v1 == hashes[j] &&
         frames[i].cx == frames[j].cx &&
         frames[i].cy == frames[j].cy &&
         frames[i].img.Bounds().Eq(frames[j].img.Bounds()) {
        // Removing frame data
        logging.Logf("Duplicate frames: %d and %d. Removing frame %d\n", i, j, j)
        if j < len(frames) - 1 {
          copy(frames[j:], frames[j+1:])
          copy(hashes[j:], hashes[j+1:])
        }
        frames = frames[:len(frames) - 1]
        hashes = hashes[:len(hashes) - 1]

        // Updating cycles
        for _, cycle := range cycles {
          for k := 0; k < len(cycle); k++ {
            if cycle[k] == uint16(j) {
              cycle[k] = uint16(i)
            } else if cycle[k] > uint16(j) {
              cycle[k]--
            }
          }
        }
      }
    }
  }
  if bam.optimization == 2 { return }


  // Removing similar frames
  const threshold = 5.0   // similarity threshold
  for i := 0; i < len(frames); i++ {
    for j := len(frames) - 1; j > i; j-- {
      if frames[i].cx == frames[j].cx &&
         frames[i].cy == frames[j].cy &&
         computeMSE(frames[i].img, frames[j].img, true) < threshold {
        // Removing frame data
        logging.Logf("Similar frames: %d and %d. Removing frame %d\n", i, j, j)
        if j < len(frames) - 1 {
          copy(frames[j:], frames[j+1:])
        }
        frames = frames[:len(frames) - 1]

        // Updating cycles
        for _, cycle := range cycles {
          for k := 0; k < len(cycle); k++ {
            if cycle[k] == uint16(j) {
              cycle[k] = uint16(i)
            } else if cycle[k] > uint16(j) {
              cycle[k]--
            }
          }
        }
      }
    }
  }

  return
}


// Used internally. Returns the hash of the given BAM frame.
func getFrameHash(frame BamFrame) uint64 {
  hash := fnv.New64a()
  buf := make([]byte, 4)

  // Hashing center location
  for i := uint(0); i < 4; i++ {
    buf[i] = byte(frame.cx >> i*8)
  }
  hash.Write(buf)
  for i := uint(0); i < 4; i++ {
    buf[i] = byte(frame.cy >> i*8)
  }
  hash.Write(buf)

  // Hashing image
  x0, x1 := frame.img.Bounds().Min.X, frame.img.Bounds().Max.X
  y0, y1 := frame.img.Bounds().Min.Y, frame.img.Bounds().Max.Y
  for y := y0; y < y1; y++ {
    for x := x0; x < x1; x++ {
      r, g, b, a := frame.img.At(x, y).RGBA()
      buf[0] = byte(r >> 8)
      buf[1] = byte(g >> 8)
      buf[2] = byte(b >> 8)
      buf[3] = byte(a >> 8)
      hash.Write(buf)
    }
  }

  return hash.Sum64()
}


// Used internally. Calculates the signal/noise ratio (in dB) from the given MSE value.
// Return value is clamped to (0, 100].
// func toPSNR(mse float64) float64 {
  // if mse < 0.000000001 { return 100.0 }
  // return 20.0 * math.Log10(255.0 / math.Sqrt(mse))
// }

// Used internally. Calculates the MSE (mean squared error) between the two specified images.
// Returns a weighted value for combined color and alpha parts. If exactMatch is true, returns 16777216.0 when both
// images differ in dimension.
// A value < 5 can be considered a close match, 7 - 10 is still similar. 20 or higher indicates noticeable differences.
func computeMSE(img1, img2 image.Image, exactMatch bool) float64 {
  cmse, amse := 0.0, 0.0
  if img1 == nil || img2 == nil { return 0.0 }
  if exactMatch &&
     (img1.Bounds().Dx() != img2.Bounds().Dx() ||
      img1.Bounds().Dy() != img2.Bounds().Dy()) {
    return 16777216.0
  }

  rgba1 := make([]byte, 4)
  rgba2 := make([]byte, 4)
  width1 := img1.Bounds().Dx()
  width2 := img2.Bounds().Dx()
  width := width1; if width2 < width { width = width2 }
  height1 := img1.Bounds().Dy()
  height2 := img2.Bounds().Dy()
  height := height1; if height2 < height { height = height2 }

  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {
      r, g, b, a := img1.At(img1.Bounds().Min.X + x, img1.Bounds().Min.Y + y).RGBA()
      rgba1[0] = byte(r >> 8)
      rgba1[1] = byte(g >> 8)
      rgba1[2] = byte(b >> 8)
      rgba1[3] = byte(a >> 8)

      r, g, b, a = img2.At(img2.Bounds().Min.X + x, img2.Bounds().Min.Y + y).RGBA()
      rgba2[0] = byte(r >> 8)
      rgba2[1] = byte(g >> 8)
      rgba2[2] = byte(b >> 8)
      rgba2[3] = byte(a >> 8)

      pc, pa := computeMSEPixel(rgba1, rgba2)
      cmse += pc
      amse += pa
    }
  }
  cmse /= float64(width*height*3)
  amse /= float64(width*height)

  return (3.0*cmse + amse) / 4.0
}

func computeMSEPixel(rgba1, rgba2 []byte) (cmse, amse float64) {
  cmse, amse = 0.0, 0.0
  if len(rgba1) < 4 || len(rgba2) < 4 { return }

  // Computing color MSE
  for i := 0; i < 3; i++ {
    cmse += ErrorSq(float64(rgba1[i]), float64(rgba2[i]))
  }
  // Computing alpha MSE
  amse = ErrorSq(float64(rgba1[3]), float64(rgba2[3]))

  return
}

func ErrorSq(x, y float64) float64 {
  return (x - y) * (x - y)
}
