package bam
// Provides BAM V2 specific functionality.

import (
  "errors"
  "fmt"
  "hash/fnv"
  "image"
  "image/draw"
  "os"

  "github.com/InfinityTools/go-binpack2d"
  "github.com/InfinityTools/go-logging"
  "github.com/InfinityTools/go-ietools"
  "github.com/InfinityTools/go-ietools/buffers"
  "github.com/InfinityTools/go-ietools/pvrz"
)

// Cache for pvr texture data
type bamV2TextureMap map[int]*pvrz.Pvr

// Holds information for a single BAM V2 data block
type bamV2Block struct {
  page int
  sx, sy int
  w, h int
  dx, dy int
}

// Holds information about a single BAM V2 frame entry
type bamV2FrameEntry struct {
  w, h, cx, cy  int           // width, height, center x/y of frame (include in hash calculation)
  blocks        []bamV2Block  // list of frame data blocks (exclude from hash calculation)
  hash uint64                 // required for identification purposes
}

// Maps bamV2FrameEntry entry to bam.frame entry
type bamV2FrameRef []int


// BAM V2 only: GetPvrzStartIndex returns the start index for PVRZ file generation.
// Operation is skipped if error state is set.
func (bam *BamFile) GetPvrzStartIndex() int {
  if bam.err != nil { return 0 }
  return bam.bamV2.pvrzStart
}


// BAM V2 only: SetPvrzStartIndex defines the start index for PVRZ file generation.
// "start" must be in range [0, 99999]. The upper range limit is absolute and applies to all PVRZ files
// generated during the BAM export operation. Operation is skipped if error state is set.
func (bam *BamFile) SetPvrzStartIndex(start int) {
  if bam.err != nil { return }
  if start < 0 || start > 99999 { bam.err = fmt.Errorf("Index out of range: %d", start); return }
  bam.bamV2.pvrzStart = start
}


// BAM V2 only: GetPvrzType returns the compression type for generated PVRZ files.
// Returns one of the PVRZ_xxx constants. Operation is skipped if error state is set.
func (bam *BamFile) GetPvrzType() int {
  if bam.err != nil { return 0 }
  return bam.bamV2.pvrzType
}


// BAM V2 only: SetPvrzType defines the PVRZ compression type. Use one of the PVRZ_xxx constants.
// Operation is skipped if error state is set.
func (bam *BamFile) SetPvrzType(pvrzType int) {
  if bam.err != nil { return }
  switch pvrzType {
  case PVRZ_AUTO, PVRZ_DXT1, PVRZ_DXT3, PVRZ_DXT5:
    bam.bamV2.pvrzType = pvrzType
  default:
    bam.err = fmt.Errorf("PVRZ type not supported: %d", pvrzType)
  }
}


// BAM V2 only: GetPvrzQuality returns the quality setting for compressing pixel data (see QUALITY_xxx constants).
// Operation is skipped if error state is set.
func (bam *BamFile) GetPvrzQuality() int {
  if bam.err != nil { return 0 }
  return bam.bamV2.quality
}


// BAM V2 only: SetPvrzQuality defines the quality setting applied when compressing pixel data. Use one of the
// QUALITY_xxx constants. Operation is skipped if error state is set.
func (bam *BamFile) SetPvrzQuality(value int) {
  if bam.err != nil { return }
  if value < QUALITY_LOW { value = QUALITY_LOW }
  if value > QUALITY_HIGH { value = QUALITY_HIGH }
  bam.bamV2.quality = value
}


// GetPvrzWeightByAlpha returns whether pixels are weighted by alpha during the encoding process to improve quality.
// Operation is skipped if error state is set.
func (bam *BamFile) GetPvrzWeightByAlpha() bool {
  if bam.err != nil { return false }
  return bam.bamV2.weightAlpha
}


// SetPvrzWeightByAlpha defines whether pixels are weighted by alpha during the encoding process to improve quality.
// Operation is skipped if error state is set.
func (bam *BamFile) SetPvrzWeightByAlpha(set bool) {
  if bam.err != nil { return }
  bam.bamV2.weightAlpha = set
}


// BAM V2 only: GetPvrzUseMetric indicates whether a perceptual metric is applied when compressing pixels.
// Operation is skipped if error state is set.
func (bam *BamFile) GetPvrzUseMetric() bool {
  if bam.err != nil { return false }
  return bam.bamV2.useMetric
}


// BAM V2 only: SetPvrzUseMetric defines whether a perceptual metric should be applied when compressing pixels.
// Operation is skipped if error state is set.
func (bam *BamFile) SetPvrzUseMetric(set bool) {
  if bam.err != nil { return }
  bam.bamV2.useMetric = set
}


// GetPvrzAlphaThreshold returns the alpha threshold in range [0.0, 100.0].
//
// It is used to determine whether to use a pixel encoding type that explicitly supports alpha. This option is only
// relevant if pvrz type is set to PVRZ_AUTO.
func (bam *BamFile) GetPvrzAlphaThreshold() float32 {
  if bam.err != nil { return 0.0 }
  return bam.bamV2.threshold
}

// SetPvrzAlphaThreshold sets the alpha threshold in range [0.0, 100.0], with 0.0 as lowest threshold (i.e. always
// consider non-transparent alpha) and 100.0 as highest threshold (i.e. ignore non-transparent alpha completely).
//
// It is used to determine whether to use a pixel encoding type that explicitly supports alpha. This option is only
// relevant if pvrz type is set to PVRZ_AUTO.
func (bam *BamFile) SetPvrzAlphaThreshold(value float32) {
  if bam.err != nil { return }
  if value < 0.0 { value = 0.0 }
  if value > 100.0 { value = 100.0 }
  bam.bamV2.threshold = value
}


// Used internally. Parses bytes stream into a BAM structure. Supports both BAM V1 and BAMC V1.
func (bam *BamFile) decodeBamV2(buf *buffers.Buffer, searchPaths []string) {
  logging.Logln("Decoding BAM V2")
  if buf.BufferLength() < 0x20 { bam.err = errors.New("BAM structure size too small"); return }

  // consistency checks
  bam.bamVersion = BAM_V2
  numFrames := int(buf.GetUint32(0x08))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  if numFrames == 0 { bam.err = errors.New("Empty frames list"); return }
  numCycles := int(buf.GetUint32(0x0c))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  if numFrames == 0 { bam.err = errors.New("Empty cycles list"); return }
  numBlocks := int(buf.GetUint32(0x10))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  if numBlocks == 0 { bam.err = errors.New("Empty data block list"); return }
  ofsFrames := int(buf.GetUint32(0x14))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ofsCycles := int(buf.GetUint32(0x18))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ofsBlocks := int(buf.GetUint32(0x1c))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  size := 0x20 + numFrames * 0x0c + numCycles + 0x04 + numBlocks * 0x1c
  if buf.BufferLength() < size { bam.err = errors.New("BAM size mismatch"); return }

  // importing frames
  logging.Log("Importing BAM frames")
  pvrzCache := make(bamV2TextureMap)    // cache for texture data
  hashCache := make(map[uint64]int)     // caches hash -> frame entry index references
  bam.frames = make([]BamFrame, numFrames)
  framesUsed := 0   // tracks number of entries in bam.frames
  frameEntries := make([]bamV2FrameEntry, numFrames)  // stores individual frame entry structures
  frameRef := make(bamV2FrameRef, numFrames)          // translates frame entries to unique frames (in bam.frames)
  for i := 0; i < numFrames; i++ {
    logging.LogProgressDot(i, numFrames, 79 - 20)  // 20 is length of prefixed string
    frameEntry := bamV2FrameEntry{}
    ofs := ofsFrames + i * 0x0c
    frameEntry.w = int(buf.GetUint16(ofs))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    frameEntry.h = int(buf.GetUint16(ofs + 0x02))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    frameEntry.cx = int(buf.GetUint16(ofs + 0x04))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    frameEntry.cy = int(buf.GetUint16(ofs + 0x06))
    if buf.Error() != nil { bam.err = buf.Error(); return }

    // assembling frame block data
    bidx := int(buf.GetUint16(ofs + 0x08))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    bcnt := int(buf.GetUint16(ofs + 0x0a))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    frameEntry.blocks = make([]bamV2Block, bcnt)
    for j := 0; j < bcnt; j++ {
      ofs2 := ofsBlocks + (bidx + j) * 0x1c
      frameEntry.blocks[j] = bamV2Block{}
      frameEntry.blocks[j].page = int(buf.GetUint32(ofs2))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].sx = int(buf.GetUint32(ofs2 + 0x04))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].sy = int(buf.GetUint32(ofs2 + 0x08))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].w = int(buf.GetUint32(ofs2 + 0x0c))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].h = int(buf.GetUint32(ofs2 + 0x10))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].dx = int(buf.GetUint32(ofs2 + 0x14))
      if buf.Error() != nil { bam.err = buf.Error(); return }
      frameEntry.blocks[j].dy = int(buf.GetUint32(ofs2 + 0x18))
      if buf.Error() != nil { bam.err = buf.Error(); return }
    }

    // generating frame image
    img, err := createTexture(frameEntry, searchPaths, pvrzCache)
    if err != nil { bam.err = err; return }
    frameEntry.hash = generateBamV2FrameHash(frameEntry, img)
    frameEntries[i] = frameEntry

    // checking for identical frames
    idx, ok := hashCache[frameEntry.hash]
    if ok {
      // match: just update frame reference
      frameRef[i] = frameRef[idx]
    } else {
      // no match: add frame to bam.frame and update frame reference
      hashCache[frameEntry.hash] = i
      frameRef[i] = framesUsed
      bam.frames[framesUsed] = BamFrame { frameEntry.cx, frameEntry.cy, img }
      framesUsed++
    }
  }
  bam.frames = bam.frames[:framesUsed]
  logging.OverridePrefix(false, false, false).Logln("")

  // importing cycles
  bam.cycles = make([]BamCycle, numCycles)
  for i := 0; i < numCycles; i++ {
    ofs := ofsCycles + i * 4
    count := int(buf.GetUint16(ofs))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    index := int(buf.GetUint16(ofs + 2))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    bam.cycles[i] = make([]uint16, count)
    for j := 0; j < count; j++ {
      bam.cycles[i][j] = uint16(frameRef[index+j])
    }
  }

  logging.Logln("Finished decoding BAM V2")
}


// Used internally. Encodes current BAM V2 structure into a byte buffer
func (bam *BamFile) encodeBamV2(outPath string) (bamOut []byte, pvrMap bamV2TextureMap) {
  logging.Logln("Encoding BAM V2")
  // checking consistency
  if len(bam.frames) == 0 { bam.err = errors.New("No frames defined"); return }
  if len(bam.cycles) == 0 { bam.err = errors.New("No cycles defined"); return }
  for i := 0; i < len(bam.cycles); i++ {
    if len(bam.cycles[i]) == 0 { bam.err = fmt.Errorf("Empty cycle: %d", i); return }
    if len(bam.cycles[i]) > 65535 { bam.err = fmt.Errorf("Too many entries in cycle %d: %d", i, len(bam.cycles[i])); return }
    for j := 0; j < len(bam.cycles[i]); j++ {
      if int(bam.cycles[i][j]) >= len(bam.frames) { bam.err = fmt.Errorf("Cycle %d: frame index out of range at index=%d", i, j); return }
    }
  }

  pvrzIndex := bam.bamV2.pvrzStart
  numFrames := 0                            // to be updated
  numCycles := len(bam.cycles)
  numBlocks := 0                            // to be updated
  ofsFrames := 0x20
  ofsCycles := ofsFrames                    // to be updated
  ofsBlocks := ofsCycles + numCycles * 0x04 // to be updated

  // applying filters
  filteredFrames, err := bam.applyFilters()
  if err != nil { bam.err = err; return }

  // preparing pvrz
  frames, pvrs := bam.packBamFrames(filteredFrames)
  if bam.err != nil { return }

  // determine size of frame and block lists
  for _, cycle := range bam.cycles {
    numFrames += len(cycle)
    for _, frameIdx := range cycle {
      numBlocks += len(frames[frameIdx].blocks)
    }
  }
  ofsCycles += numFrames * 0x0c
  ofsBlocks += numFrames * 0x0c

  outSize := ofsBlocks + numBlocks * 0x1c
  out := buffers.Create()
  if out.Error() != nil { bam.err = out.Error(); return }
  out.InsertBytes(0, outSize)
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutString(0x00, 4, sig_bam)
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutString(0x04, 4, ver_v2)
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x08, uint32(numFrames))
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x0c, uint32(numCycles))
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x10, uint32(numBlocks))
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x14, uint32(ofsFrames))
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x18, uint32(ofsCycles))
  if out.Error() != nil { bam.err = out.Error(); return }
  out.PutUint32(0x1c, uint32(ofsBlocks))
  if out.Error() != nil { bam.err = out.Error(); return }

  // writing bam structures
  frameIdx, blockIdx := 0, 0
  for cycleIdx, cycle := range bam.cycles {
    // writing cycle entry
    ofs := ofsCycles + cycleIdx * 0x04
    out.PutUint16(ofs, uint16(len(cycle)))
    if out.Error() != nil { bam.err = out.Error(); return }
    out.PutUint16(ofs + 2, uint16(frameIdx))
    if out.Error() != nil { bam.err = out.Error(); return }

    for _, fidx := range cycle {
      // writing frame entries
      e := frames[fidx]
      ofs = ofsFrames + frameIdx * 0x0c
      out.PutUint16(ofs, uint16(e.w))
      if out.Error() != nil { bam.err = out.Error(); return }
      out.PutUint16(ofs + 0x02, uint16(e.h))
      if out.Error() != nil { bam.err = out.Error(); return }
      out.PutInt16(ofs + 0x04, int16(e.cx))
      if out.Error() != nil { bam.err = out.Error(); return }
      out.PutInt16(ofs + 0x06, int16(e.cy))
      if out.Error() != nil { bam.err = out.Error(); return }
      out.PutInt16(ofs + 0x08, int16(blockIdx))
      if out.Error() != nil { bam.err = out.Error(); return }
      out.PutInt16(ofs + 0x0a, int16(len(e.blocks)))
      if out.Error() != nil { bam.err = out.Error(); return }
      frameIdx++

      // writing frame blocks
      for _, block := range e.blocks {
        ofs = ofsBlocks + blockIdx * 0x1c
        out.PutUint32(ofs, uint32(pvrzIndex + block.page))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x04, uint32(block.sx))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x08, uint32(block.sy))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x0c, uint32(block.w))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x10, uint32(block.h))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x14, uint32(block.dx))
        if out.Error() != nil { bam.err = out.Error(); return }
        out.PutUint32(ofs + 0x18, uint32(block.dy))
        if out.Error() != nil { bam.err = out.Error(); return }
        blockIdx++
      }
    }
  }

  bamOut = out.Bytes()
  pvrMap = make(bamV2TextureMap)
  for idx, pvr := range pvrs {
    pvrMap[pvrzIndex + idx] = pvr
  }

  logging.Logln("Finished encoding BAM V2")
  return
}

// Used internally. Distributes available source frames over Pvr textures.
// Returns frame information as a master bamV2FrameEntry list for each unique frame and Pvr textures as pvr list.
func (bam *BamFile) packBamFrames(frames []BamFrame) (frameInfo []bamV2FrameEntry, pvrs []*pvrz.Pvr) {
  const texSize = 1024    // maximum texture width and height
  const binRule = binpack2d.RULE_BEST_LONG_SIDE_FIT
  threshold := byte(255.0 - (bam.bamV2.threshold * 2.55))
  hasAlpha := (bam.bamV2.pvrzType >= PVRZ_DXT3)

  frameInfo = make([]bamV2FrameEntry, 0)
  pvrs = make([]*pvrz.Pvr, 0)
  packers := make([]*binpack2d.Packer, 0)

  logging.Log("Packing input frames")
  for frameIdx, frame := range frames {
    logging.LogProgressDot(frameIdx, len(frames), 79 - 20)   // 20 is length of prefix string
    frameEntry := bamV2FrameEntry{ w: frame.img.Bounds().Dx(),
                                   h: frame.img.Bounds().Dy(),
                                   cx: frame.cx, cy: frame.cy,
                                   blocks: make([]bamV2Block, 0), hash: 0 }

    // Testing alpha threshold
    if !hasAlpha {
      hasAlpha = imageHasAlpha(frame.img, threshold)
    }

    // Try to fit each frame block onto a texture
    frameBlocks := splitBamFrame(frame.img, texSize, texSize)
    for _, rectBlock := range frameBlocks {
      packerIdx := -1
      rect := image.Rectangle{}
      // Does it fit on existing packers?
      for idx, packer := range packers {
        r, ok := packer.Insert(rectBlock.Dx(), rectBlock.Dy(), binRule)
        if ok {
          // Yes! Adding.
          packerIdx = idx
          rect.Min.X, rect.Min.Y = r.X, r.Y
          rect.Max.X, rect.Max.Y = r.X + r.W, r.Y + r.H
          break
        }
      }

      if packerIdx < 0 {
        // No. Creating new packer and texture.
        pvrs = append(pvrs, pvrz.CreateNew(texSize, texSize, pvrz.TYPE_BC1))
        packers = append(packers, binpack2d.Create(texSize, texSize))
        r, ok := packers[len(packers) - 1].Insert(rectBlock.Dx(), rectBlock.Dy(), binRule)
        if ok {
          packerIdx = len(packers) - 1
          rect.Min.X, rect.Min.Y = r.X, r.Y
          rect.Max.X, rect.Max.Y = r.X + r.W, r.Y + r.H
        }
      }

      // Should never happen
      if packerIdx < 0 {
        bam.err = fmt.Errorf("Error exporting frame %d to PVRZ", frameIdx)
        logging.OverridePrefix(false, false, false).Logln("")
        return
      }

      // Finalizing frame block
      pvrs[packerIdx].SetImageRect(frame.img, image.Rectangle{rectBlock.Min, rectBlock.Min.Add(rectBlock.Size())}, rect.Min)
      block := bamV2Block{page: packerIdx,
                          sx: rect.Min.X, sy: rect.Min.Y,
                          w: rect.Dx(), h: rect.Dy(),
                          dx: rectBlock.Min.X, dy: rectBlock.Min.Y}
      frameEntry.blocks = append(frameEntry.blocks, block)
    }

    frameInfo = append(frameInfo, frameEntry)
  }
  logging.OverridePrefix(false, false, false).Logln("")

  // Finalizing packer and pvr objects
  for idx, pvr := range pvrs {
    // Don't waste empty texture space
    packers[idx].ShrinkBin(true)
    w, h := packers[idx].GetWidth(), packers[idx].GetHeight()
    if w < texSize || h < texSize {
      pvr.SetDimension(w, h, true)
    }
    // Setting remaining pvrz options
    if hasAlpha && (bam.bamV2.pvrzType != PVRZ_DXT1) {
      pvr.SetPixelType(pvrz.TYPE_BC3)
    }
    switch bam.bamV2.quality {
      case QUALITY_LOW:   pvr.SetQuality(pvrz.QUALITY_LOW)
      case QUALITY_HIGH:  pvr.SetQuality(pvrz.QUALITY_HIGH)
    }
    pvr.SetWeightByAlpha(bam.bamV2.weightAlpha)
    pvr.SetPerceptiveMetric(bam.bamV2.useMetric)
  }

  return
}


// Used internally. Checks if given image contains alpha values exceeding threshold.
func imageHasAlpha(img image.Image, threshold byte) bool {
  retVal := false
  var buf []uint8 = nil
  var stride int
  switch img.(type) {
    case *image.NRGBA:
      buf = img.(*image.NRGBA).Pix
      stride = img.(*image.NRGBA).Stride
    case *image.RGBA:
      buf = img.(*image.RGBA).Pix
      stride = img.(*image.RGBA).Stride
  }
  if buf != nil {
    for y := 0; !retVal && y < img.Bounds().Dy(); y++ {
      ofs := y * stride + 3
      for x := 0; !retVal && x < img.Bounds().Dx(); x++ {
        retVal = (buf[ofs] != 0 && buf[ofs] < threshold)
        ofs += 4
      }
    }
  } else {
    for y := img.Bounds().Min.Y; !retVal && y < img.Bounds().Max.Y; y++ {
      for x := img.Bounds().Min.X; !retVal && x < img.Bounds().Max.X; x++ {
        _, _, _, a := img.At(x, y).RGBA()
        retVal = (a != 0 && byte(a) < threshold)
      }
    }
  }
  return retVal
}


// Used internally. Returns a list of image regions that are guaranteed to meet the specified size restrictions.
func splitBamFrame(img image.Image, maxWidth, maxHeight int) []image.Rectangle {
  retVal := make([]image.Rectangle, 0)
  if maxWidth < 1 || maxHeight < 1 { return retVal }

  w, h := img.Bounds().Dx(), img.Bounds().Dy()
  for y := 0; y < h; {
    bh := maxHeight
    if h - y < bh { bh = h - y }
    for x := 0; x < w; {
      bw := maxWidth
      if w - x < bw { bw = w - x }
      retVal = append(retVal, image.Rect(x, y, x + bw, y + bh))
      x += bw
    }
    y += bh
  }
  return retVal
}


// Used internally. Creates a correctly formatted pvrz filename in the form "[dir]/mosXXXX.pvrz" where XXXX specifies
// the pvrz index. Returns an empty string if the index is out of range.
func generatePvrzFileName(dir string, index int) string {
  if index < 0 || index > 99999 { return "" }

  path := ""
  if len(dir) > 0 && dir[:1] != "/" && dir[:1] != "\\" {
    path += dir
    if path[len(path)-1:] != "/" && path[len(path)-1:] != "\\" {
      path += ietools.PATH_SEPARATOR
    }
  }
  path += fmt.Sprintf("mos%04d.pvrz", index)

  return path
}


// Used internally. Calculates a hash based on the given bamV2FrameEntry data.
func generateBamV2FrameHash(e bamV2FrameEntry, img image.Image) uint64 {
  hash := fnv.New64()

  // adding frame header data
  buf := make([]byte, 8)
  buf[0] = byte(e.w); buf[1] = byte(e.w >> 8)
  buf[2] = byte(e.h); buf[3] = byte(e.h >> 8)
  buf[4] = byte(e.cx); buf[5] = byte(e.cx >> 8)
  buf[6] = byte(e.cy); buf[7] = byte(e.cy >> 8)
  hash.Write(buf)

  // adding frame image data
  x0 := img.Bounds().Min.X
  y0 := img.Bounds().Min.Y
  width := img.Bounds().Max.X - x0
  height := img.Bounds().Max.Y - y0
  buf = make([]byte, width*height*4)
  ofs := 0
  for y := 0; y < height; y++ {
    for x := 0; x < width; x++ {
      r, g, b, a := img.At(x0 + x, y0 + y).RGBA()
      buf[ofs], buf[ofs+1], buf[ofs+2], buf[ofs+3] = byte(r), byte(g), byte(b), byte(a)
      ofs += 4
    }
  }
  hash.Write(buf)

  return hash.Sum64()
}


// Used internally. Composes a new image from the graphics blocks defined by block. searchPath specifies one or more
// search paths for pvrz files. pvrz is a cache for previously used pvrz files.
func createTexture(entry bamV2FrameEntry, searchPaths []string, pvrzCache bamV2TextureMap) (imgOut image.Image, err error) {
  imgDst := image.NewRGBA(image.Rect(0, 0, entry.w, entry.h))
  imgOut = imgDst
  for _, block := range entry.blocks {
    pvr, ok := pvrzCache[block.page]
    if !ok {
      // Pvrz not cached: initialize a new Pvr object
      for _, path := range searchPaths {
        pvrzFile := generatePvrzFileName(path, block.page)
        var fi os.FileInfo
        fi, err = os.Stat(pvrzFile)
        if err != nil { return }
        if fi.Mode().IsRegular() {
          // File found
          var fin *os.File
          fin, err = os.Open(pvrzFile)
          if err != nil { return }
          defer fin.Close()
          pvr = pvrz.Load(fin)
          if pvr.Error() != nil { err = pvr.Error(); return }
          pvrzCache[block.page] = pvr
          break
        }
      }
    }
    if pvr == nil { err = fmt.Errorf("Could not find PVRZ file: %d", block.page); return }

    // assembling pixel data
    sp := image.Pt(block.sx, block.sy)
    dr := image.Rect(block.dx, block.dy, block.dx + block.w, block.dy + block.h)
    draw.Draw(imgDst, dr, pvr.GetImage(), sp, draw.Src)
  }

  return
}
