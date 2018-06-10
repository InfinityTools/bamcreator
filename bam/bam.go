/*
Package bam deals with BAM file creation or manipulation.
*/
package bam

import (
  "errors"
  "fmt"
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
  bamVersion    int       // 1: BAM V1, 2: BAM V2
  bamV1         BamV1Config
  bamV2         BamV2Config
}


// CreateNew initializes an empty BAM structure and returns a pointer to it.
func CreateNew() *BamFile {
  bam := BamFile{ frames: make([]BamFrame, 0),
                  cycles: make([]BamCycle, 0),
                  err: nil,
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
