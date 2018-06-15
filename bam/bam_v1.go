package bam
// Provides BAM V1 specific functionality.

import (
  "bytes"
  "errors"
  "fmt"
  "image/color"

  "github.com/InfinityTools/go-logging"
  "github.com/InfinityTools/go-ietools"
  "github.com/InfinityTools/go-ietools/buffers"
)

// BAM V1 only: GetColorKey returns the color value that should be treated as transparent.
// It is only effective if GetColorKeyEnabled() is true. Operation is skipped if error state is set.
func (bam *BamFile) GetColorKey() color.Color {
  if bam.err != nil { return color.Transparent }
  return bam.bamV1.colorKey
}


// BAM V1 only: SetColorKey assigns a color value that should be treated as transparent.
// It is only effective if GetColorKeyEnabled() is true. Operation is skipped if error state is set.
func (bam *BamFile) SetColorKey(col color.Color) {
  if bam.err != nil { return }
  if col == nil { bam.err = fmt.Errorf("SetColorKey: Color is undefined"); return }
  r, g, b, a := col.RGBA()
  bam.bamV1.colorKey = color.RGBA{ byte(r), byte(g), byte(b), byte(a) }
}


// BAM V1 only: GetColorKeyEnabled returns whether a the color defined by SetColorKey should be treated as
// transparent when encoding BAM V1. Operation is skipped if error state is set.
func (bam *BamFile) GetColorKeyEnabled() bool {
  if bam.err != nil { return false }
  return bam.bamV1.colorKeyEnabled
}


// BAM V1 only: SetColorKeyEnabled indicates whether a the color defined by SetColorKey should be treated as
// transparent when encoding BAM V1. Operation is skipped if error state is set.
func (bam *BamFile) SetColorKeyEnabled(set bool) {
  if bam.err != nil { return }
  bam.bamV1.colorKeyEnabled = set
}


// BAM V1 only: GetTransparentColor returns the palette index that is used for RLE frame compression.
// Operation is skipped if error state is set.
func (bam *BamFile) GetTransparentColor() byte {
  if bam.err != nil { return 0 }
  return bam.bamV1.transColor
}


// BAM V1 only: SetTransparentColor assigns a new color index that is used for RLE frame compression.
// Operation is skipped if error state is set.
func (bam *BamFile) SetTransparentColor(index byte) {
  if bam.err != nil { return }
  bam.bamV1.transColor = index
}


// BAM V1 only: GetCompression returns whether the BAM structure is exported as compressed BAMC.
// Operation is skipped if error state is set.
func (bam *BamFile) GetCompression() bool {
  if bam.err != nil { return false }
  return bam.bamV1.compress
}


// BAM V1 only: SetCompression defines whether to export the BAM structure as compressed BAMC.
// Operation is skipped if error state is set.
func (bam *BamFile) SetCompression(set bool) {
  if bam.err != nil { return }
  bam.bamV1.compress = set
}


// BAM V1 only: GetRle returns whether RLE encoding is applied to individual BAM frames.
// Use one of the constants RLE_OFF, RLE_ON or RLE_AUTO to turn RLE off, turn RLE on or use best compression ratio.
// Operation is skipped if error state is set.
func (bam *BamFile) GetRle() int {
  if bam.err != nil { return 0 }
  return bam.bamV1.rle
}


// BAM V1 only: SetRle defines whether RLE encoding is applied to individual BAM frames.
// Specify either RLE_OFF, RLE_ON or RLE_AUTO. Operation is skipped if error state is set.
func (bam *BamFile) SetRle(set int) {
  if bam.err != nil { return }
  switch {
    case set < 0: bam.bamV1.rle = RLE_AUTO
    case set > 0: bam.bamV1.rle = RLE_ON
    default:      bam.bamV1.rle = RLE_OFF
  }
}


// BAM V1 only: GetDiscardAlpha returns whether alpha color information from the palette will be discarded on export.
// This can be useful to retain compatibility with classic IE games.
// Operation is skipped if error state is set.
func (bam *BamFile) GetDiscardAlpha() bool {
  if bam.err != nil { return false }
  return bam.bamV1.discardAlpha
}


// BAM V1 only: SetDiscardAlpha defines whether alpha color information from the palette will be discarded on export.
// Operation is skipped if error state is set.
func (bam *BamFile) SetDiscardAlpha(set bool) {
  if bam.err != nil { return }
  bam.bamV1.discardAlpha = set
}


// BAM V1 only: GetQuality returns the minimum and maximum quality bounds used to generate the output palette.
// Operation is skipped if error state is set.
func (bam *BamFile) GetQuality() (min, max int) {
  if bam.err != nil { return }
  min = bam.bamV1.qualityMin
  max = bam.bamV1.qualityMax
  return
}


// BAM V1 only: SetQuality sets the minimum and maximum bounds of accepted quality used to generate the output palette.
// Values must be in range [0, 100] where 0 is worst quality and 100 is best quality. Operation is skipped if error
// state is set.
func (bam *BamFile) SetQuality(min, max int) {
  if bam.err != nil { return }
  if min < 0 || min > 100 { bam.err = fmt.Errorf("SetQuality: MinQuality out of range (%d)", min); return }
  if max < 0 || max > 100 { bam.err = fmt.Errorf("SetQuality: MaxQuality out of range (%d)", max); return }
  if max < min { min, max = max, min }
  bam.bamV1.qualityMin = min
  bam.bamV1.qualityMax = max
}


// BAM V1 only: GetSpeed returns the accuracy at which the generated palette will be applied to the output image.
// Operation is skipped if error state is set.
func (bam *BamFile) GetSpeed() int {
  if bam.err != nil { return 0 }
  return bam.bamV1.speed
}


// BAM V1 only: SetSpeed sets the accuracy at which the generated palette will be applied to the output image.
// Values must be in range [1, 10].
// High speed combined with high quality makes it less likely that minimum required quality is reached. Default: 3
// Operation is skipped if error state is set.
func (bam *BamFile) SetSpeed(value int) {
  if bam.err != nil { return }
  if value < 1 || value > 10 { bam.err = fmt.Errorf("SetSpeed: Value out of range (%d)", value); return }
  bam.bamV1.speed = value
}


// BAM V1 only: GetDither returns the amount of dither to be applied when generating palettized output frames.
// Operation is skipped if error state is set.
func (bam *BamFile) GetDither() float32 {
  if bam.err != nil { return 0.0 }
  return bam.bamV1.dither
}


// BAM V1 only: SetDither sets the amount of dither to be applied when generating palettized output frames. Values
// must be in range [0.0, 1.0]
// where 0.0 is no dither and 1.0 is strongest dither. Operation is skipped if error state is set.
func (bam *BamFile) SetDither(value float32) {
  if bam.err != nil { return }
  if value < 0.0 || value > 1.0 { bam.err = fmt.Errorf("SetDither: Value out of range (%v)", value); return }
  bam.bamV1.dither = value
}


// BAM V1 only: GetPaletteSortFlags returns the currently defined type and options for sorting BAM palette entries.
// See palette/sort.go for type and flags constants. Operation is skipped if error state is set.
func (bam *BamFile) GetPaletteSortFlags() int {
  if bam.err != nil { return 0 }
  return bam.bamV1.sortFlags
}

// BAM V1 only: SetPaletteSortFlags defines type and options for sorting BAM palette entries. See palette/sort.go for
// type and flags constants. Operation is skipped if error state is set.
func (bam *BamFile) SetPaletteSortFlags(flags int) {
  if bam.err != nil { return }
  bam.bamV1.sortFlags = flags
}


// BAM V1 only: GetFixedColorLength returns the current number of predefined colors.
// Operation is skipped if error state is set.
func (bam *BamFile) GetFixedColorLength() int {
  if bam.err != nil { return 0 }
  return len(bam.bamV1.fixedColors)
}


// BAM V1 only: GetFixedColor returns the predefined color at the specified index.
// Operation is skipped if error state is set.
func (bam *BamFile) GetFixedColor(index int) color.Color {
  if bam.err != nil { return nil }
  if index < 0 || index >= len(bam.bamV1.fixedColors) { bam.err = fmt.Errorf("GetFixedColor: Index out of range (%d)", index); return nil }
  return bam.bamV1.fixedColors[index]
}


// BAM V1 only: SetFixedColor replaces the fixed color at the given index with the specified color definition.
// Fully transparent color definitions will be silently skipped. Operation is skipped if error state is set.
func (bam *BamFile) SetFixedColor(index int, col color.Color) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.bamV1.fixedColors) { bam.err = fmt.Errorf("SetFixedColor: Index out of range (%d)", index); return }
  _, _, _, a := col.RGBA()
  if a == 0 { return }

  bam.bamV1.fixedColors[index] = col
}


// BAM V1 only: DeleteFixedColor removes the predefined color definition at the specified index.
// Operation is skipped if error state is set.
func (bam *BamFile) DeleteFixedColor(index int) {
  if bam.err != nil { return }
  if index < 0 || index >= len(bam.bamV1.fixedColors) { bam.err = fmt.Errorf("DeleteFixedColor: Index out of range (%d)", index); return }

  for idx := index + 1; idx < len(bam.bamV1.fixedColors); idx++ {
    bam.bamV1.fixedColors[idx - 1] = bam.bamV1.fixedColors[idx]
  }
  bam.bamV1.fixedColors = bam.bamV1.fixedColors[:len(bam.bamV1.fixedColors) - 1]
}


// BAM V1 only: InsertFixedColor inserts a new predefined color definition at the specified index.
// Fully transparent color definitions will be silently skipped. Operation is skipped if error state is set.
func (bam *BamFile) InsertFixedColor(index int, col color.Color) {
  if bam.err != nil { return }
  if index < 0 || index > len(bam.bamV1.fixedColors) { bam.err = fmt.Errorf("InsertFixedColor: Index out of range (%d)", index); return }
  _, _, _, a := col.RGBA()
  if a == 0 { return }

  bam.bamV1.fixedColors = append(bam.bamV1.fixedColors, make([]color.Color, 1)...)
  for idx := len(bam.bamV1.fixedColors) - 1; idx > index; idx-- {
    bam.bamV1.fixedColors[idx] = bam.bamV1.fixedColors[idx - 1]
  }
  bam.SetFixedColor(index, col)
}

// BAM V1 only: AddFixedColor appends a new predefined color definition to the list of predefined colors.
// Returns the index of the added color definition. Operation is skipped if error state is set.
func (bam *BamFile) AddFixedColor(col color.Color) int {
  if bam.err != nil { return 0 }
  retVal := len(bam.bamV1.fixedColors)
  bam.InsertFixedColor(retVal, col)
  return retVal
}


// BAM V1 only: SetPalette assigns a fixed palette that is used in place of color quantization.
//
// Palette order will be preserved as best as possible. However, duplicate color entries will be discarded and color
// at palette index 0 will always be treated as fully transparent. Operation is skipped if error state is set.
func (bam *BamFile) SetPalette(pal color.Palette) {
  if bam.err != nil { return }
  if pal == nil { bam.ClearPalette(); return }

  exists := make(map[color.Color]bool)
  bam.bamV1.palette = make(color.Palette, 256)

  // index 0 is always transparent
  bam.bamV1.palette[0] = color.RGBA{0, 0, 0, 0}
  exists[bam.bamV1.palette[0]] = true
  exists[color.RGBA{0, 255, 0, 255}] = true // green is equivalent to transparent
  numColors := 1

  for _, col := range pal {
    if _, ok := exists[col]; !ok {
      bam.bamV1.palette[numColors] = col
      exists[col] = true
      numColors++
      if numColors == 256 { break }
    }
  }
  if numColors < 256 {
    bam.bamV1.palette = bam.bamV1.palette[:numColors]
  }
}

// BAM V1 only: GetPalette returns a copy of an external palette that may be used by the BAM conversion.
// Returns nil if no external palette is available. Operation is skipped if error state is set.
func (bam *BamFile) GetPalette() color.Palette {
  if bam.err != nil { return nil }

  var retVal color.Palette = nil
  if bam.bamV1.palette != nil {
    retVal = make(color.Palette, len(bam.bamV1.palette))
    copy(retVal, bam.bamV1.palette)
  }
  return retVal
}

// BAM V1 only: ClearPalette removes an assigned external palette and re-enables color quantization.
// Operation is skipped if error state is set.
func (bam *BamFile) ClearPalette() {
  if bam.err != nil { return }

  bam.bamV1.palette = nil
}


// BAM V1 only: DecompressRLEFrame is a helper function that decodes RLE-compressed frame data.
// width and height specify the frame dimension in pixels. transColor indicates the pixel value
// to decompress. data contains the compressed pixel data. Returns nil on error.
func DecompressRLEFrame(width, height int, transColor byte, data []byte) []byte {
  if width <= 0 || height <= 0 || data == nil || len(data) == 0 { return nil }

  comp := buffers.Load(bytes.NewBuffer(data))
  uncomp, err := getFrameBuffer(comp, width, height, 0, true, transColor)
  if err != nil { return nil }
  return uncomp
}


// BAM V1 only: CompressRLEFrame is a helper function that encodes uncompressed frame data.
// width and height specify the frame dimension in pixels. transColor indicates the pixel value
// to compress. data contains the uncompressed pixel data. Returns nil on error.
func CompressRLEFrame(width, height int, transColor byte, data[]byte) []byte {
  if width <= 0 || height <= 0 || data == nil || len(data) < width*height { return nil }

  uncomp := buffers.Load(bytes.NewBuffer(data))
  comp, err := encodeFrame(uncomp, width, height, 0, transColor)
  if err != nil { return nil}
  return comp
}


// Used internally. Parses bytes stream into a BAM structure. Supports both BAM V1 and BAMC V1.
func (bam *BamFile) decodeBamV1(buf *buffers.Buffer) {
  logging.Logln("Decoding BAM V1")
  if buf.BufferLength() < 0x18 { bam.err = errors.New("BAM structure size too small"); return }

  s := buf.GetString(0, 8, false)
  if buf.Error() != nil { bam.err = buf.Error(); return }

  // decompress if needed
  if s == sig_bamc + ver_v1 {
    uncSize := int(buf.GetInt32(8))
    if buf.DecompressReplace(12, buf.BufferLength() - 12) != uncSize { bam.err = errors.New("BAMC buffer size mismatch"); return }
    if buf.Error() != nil { bam.err = buf.Error(); return }
    buf.DeleteBytes(0, 12)
    bam.bamV1.compress = true
    s = buf.GetString(0, 8, false)
  }

  // consistency checks
  if s != sig_bam + ver_v1 { bam.err = errors.New("Invalid BAM signature"); return }
  bam.bamVersion = BAM_V1
  numFrames := int(buf.GetUint16(0x08))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  if numFrames == 0 { bam.err = errors.New("Empty frames list"); return }
  numCycles := int(buf.GetUint8(0x0a))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  if numFrames == 0 { bam.err = errors.New("Empty cycles list"); return }
  bam.bamV1.transColor = buf.GetUint8(0x0b)
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ofsFrames := int(buf.GetUint32(0x0c))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ofsCycles := ofsFrames + numFrames * 0x0c
  ofsPal := int(buf.GetUint32(0x10))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  ofsLookup := int(buf.GetUint32(0x14))
  if buf.Error() != nil { bam.err = buf.Error(); return }
  size := numFrames * 0x0c + numCycles * 0x04
  if ofsPal - ofsFrames < size { bam.err = errors.New("Invalid frame or cycle definitions"); return }
  if ofsLookup - ofsPal < 0x04 { bam.err = errors.New("Palette too small"); return }

  // preparing palette array
  size = ofsLookup - ofsPal
  numColors := (size & ^3) >> 2
  palette := make([]uint32, numColors)
  for i := 0; i < numColors; i++ {
    ofs := ofsPal + i << 2
    palette[i] =  buf.GetUint32(ofs)
    if buf.Error() != nil { bam.err = buf.Error(); return }
    if i == 0 {
      palette[i] &= ^MASK_BYTE4   // palette entry 0 is always transparent
    } else {
      if palette[i] & MASK_BYTE4 == 0 { palette[i] |= MASK_BYTE4 }  // special case: alpha=0 treated as fully opaque
    }
  }

  // importing frames
  logging.Log("Importing BAM frames")
  bam.frames = make([]BamFrame, numFrames)
  for i := 0; i < numFrames; i++ {
    logging.LogProgressDot(i, numFrames, 79 - 20)  // 20 is length of prefixed string
    ofs := ofsFrames + i * 0x0c
    w := int(buf.GetUint16(ofs))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    h := int(buf.GetUint16(ofs + 2))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    cx := int(buf.GetUint16(ofs + 4))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    cy := int(buf.GetUint16(ofs + 6))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    ofsData := buf.GetUint32(ofs + 8)
    if buf.Error() != nil { bam.err = buf.Error(); return }
    rle := ofsData & uint32(ietools.BIT31) == 0
    data, err := getFrameBuffer(buf, w, h, int(ofsData & ^uint32(ietools.BIT31)), rle, bam.bamV1.transColor)
    if err != nil { bam.err = err; return }
    if buf.Error() != nil { bam.err = buf.Error(); return }
    bam.frames[i] = BamFrame { cx, cy, createPaletteImage(w, h, data, palette) }
  }
  logging.OverridePrefix(false, false, false).Logln("")

  // importing cycles
  bam.cycles = make([]BamCycle, numCycles)
  for i := 0; i < numCycles; i++ {
    ofs := ofsCycles + i * 4
    count := int(buf.GetUint16(ofs))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    index := int(buf.GetUint16(ofs + 2))
    if buf.Error() != nil { bam.err = buf.Error(); return }
    ofs = ofsLookup + index * 2
    cycle := make([]uint16, count)
    for j := 0; j < count; j++ {
      cycle[j] = buf.GetUint16(ofs + j * 2)
      if buf.Error() != nil { bam.err = buf.Error(); return }
    }
    bam.cycles[i] = cycle
  }
  logging.Logln("Finished decoding BAM V1")
}


// Used internally. Encodes current BAM V1 structure into a byte buffer.
func (bam *BamFile) encodeBamV1() []byte {
  logging.Logln("Encoding BAM V1")
  // checking consistency
  if len(bam.frames) == 0 { bam.err = errors.New("No frames defined"); return nil }
  if len(bam.frames) > 65535 { bam.err = fmt.Errorf("Too many frames defined: %d", len(bam.frames)); return nil }
  if len(bam.cycles) == 0 { bam.err = errors.New("No cycles defined"); return nil }
  if len(bam.cycles) > 255 { bam.err = fmt.Errorf("Too many cycles defined: %d", len(bam.cycles)); return nil }
  for i := 0; i < len(bam.cycles); i++ {
    if len(bam.cycles[i]) == 0 { bam.err = fmt.Errorf("Empty cycle: %d", i); return nil }
    if len(bam.cycles[i]) > 65535 { bam.err = fmt.Errorf("Too many entries in cycle %d: %d", i, len(bam.cycles[i])); return nil }
    for j := 0; j < len(bam.cycles[i]); j++ {
      if int(bam.cycles[i][j]) >= len(bam.frames) { bam.err = fmt.Errorf("Cycle %d: frame index out of range at index=%d", i, j); return nil }
    }
  }

  // Applying optimizations
  frames, cycles := bam.optimize()
  if bam.err != nil { return nil }

  numFrames := len(frames)
  numCycles := len(cycles)
  ofsFrames := 0x18
  ofsCycles := ofsFrames + numFrames * 0x0c
  ofsPalette := ofsCycles + numCycles * 0x04
  ofsLookup := ofsPalette + 256*4
  ofsFrameData := ofsLookup
  for i := 0; i < numCycles; i++ {
    ofsFrameData += len(cycles[i]) * 2
  }

  // pre-allocating and setting buffer up to frame pixel data
  out := buffers.Create()
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.InsertBytes(0, ofsFrameData)
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutString(0x00, 4, sig_bam)
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutString(0x04, 4, ver_v1)
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint8(0x0b, bam.bamV1.transColor)
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint16(0x08, uint16(numFrames))
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint8(0x0a, uint8(numCycles))
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint32(0x0c, uint32(ofsFrames))
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint32(0x10, uint32(ofsPalette))
  if out.Error() != nil { bam.err = out.Error(); return nil }
  out.PutUint32(0x14, uint32(ofsLookup))
  if out.Error() != nil { bam.err = out.Error(); return nil }

  // writing cycle entries
  var idxLookup uint16 = 0
  for i := 0; i < numCycles; i++ {
    cycle := cycles[i]

    // writing cycle entry
    ofs := ofsCycles + i * 0x04
    out.PutUint16(ofs, uint16(len(cycle)))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutUint16(ofs + 2, idxLookup)
    if out.Error() != nil { bam.err = out.Error(); return nil }

    // writing frame lookup sequence
    ofs = ofsLookup + int(idxLookup) * 2
    for j := 0; j < len(cycle); j++ {
      out.PutUint16(ofs, cycle[j])
      ofs += 2
    }

    idxLookup += uint16(len(cycle))
  }

  // applying filters
  frames, bam.err = bam.applyFilters(frames)
  if bam.err != nil { return nil }

  // writing palette
  palImages, palette, err := bam.generatePalette(frames)
  if err != nil { bam.err = err; return nil }
  for i := 0; i < len(palette); i++ {
    r, g, b, a := NRGBA(palette[i])
    if i == 0 { r, g, b, a = 0, 255, 0, 0 } // palette #0 is always "green"
    if bam.bamV1.discardAlpha { a = 0 }
    var color uint32 = uint32(a) << 24 | uint32(r) << 16 | uint32(g) << 8 | uint32(b)
    out.PutUint32(ofsPalette + i << 2, color)
    if out.Error() != nil { bam.err = out.Error(); return nil }
  }

  // writing frame entries
  logging.Log("Exporting BAM frames")
  curOfsData := ofsFrameData
  for i := 0; i < numFrames; i++ {
    logging.LogProgressDot(i, numFrames, 79 - 20)  // 20 is length of prefix string
    frame := frames[i]
    img := palImages[i]

    // writing frame data
    frameBuf := palettedToBytes(img)
    width, height := frame.img.Bounds().Dx(), frame.img.Bounds().Dy()
    rleFlag := uint32(ietools.BIT31)  // initially uncompressed
    if bam.bamV1.rle != RLE_OFF {
      var rleBuf []byte
      rleBuf, bam.err = encodeFrame(buffers.Load(bytes.NewBuffer(frameBuf)), width, height, 0, bam.bamV1.transColor)
      if bam.err != nil { return nil }
      if bam.bamV1.rle == RLE_ON || len(rleBuf) < len(frameBuf) {
        frameBuf = rleBuf
        rleFlag = 0 // set to RLE compressed
      }
    }
    out.InsertBytes(curOfsData, len(frameBuf))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutBuffer(curOfsData, frameBuf)
    if out.Error() != nil { bam.err = out.Error(); return nil }

    // writing frame entry
    ofs := ofsFrames + i * 0x0c
    out.PutUint16(ofs, uint16(width))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutUint16(ofs + 0x02, uint16(height))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutInt16(ofs + 0x04, int16(frame.cx))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutInt16(ofs + 0x06, int16(frame.cy))
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutUint32(ofs + 0x08, uint32(curOfsData) | rleFlag)
    if out.Error() != nil { bam.err = out.Error(); return nil }

    curOfsData += len(frameBuf)
  }
  logging.OverridePrefix(false, false, false).Logln("")

  // Compressing to BAMC?
  if bam.bamV1.compress {
    logging.Logln("Compressing BAM")
    uncSize := out.BufferLength()
    out.CompressReplace(0, uncSize, -1)
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.InsertBytes(0, 12)
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutString(0, 4, sig_bamc)
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutString(4, 4, ver_v1)
    if out.Error() != nil { bam.err = out.Error(); return nil }
    out.PutUint32(8, uint32(uncSize))
    if out.Error() != nil { bam.err = out.Error(); return nil }
  }

  logging.Logln("Finished encoding BAM V1")
  return out.Bytes()
}


// Used internally. Decodes frame data (uncompressed or RLE-encoded).
func getFrameBuffer(buf *buffers.Buffer, width, height, ofs int, rle bool, transColor byte) (data []byte, err error) {
  data = nil
  err = nil
  uncSize := width*height
  if rle {
    data = make([]byte, uncSize)
    for size := 0; size < uncSize; {
      b := buf.GetUint8(ofs)
      ofs++
      if b == transColor {
        c := uint(buf.GetUint8(ofs))
        ofs++
        for i := uint(0); i <= c; i++ {
          if size >= uncSize { err = errors.New("Buffer overflow while decoding frame data"); return }
          data[size] = b
          size++
        }
      } else {
        data[size] = b
        size++
      }
    }
  } else {
    data = buf.GetBuffer(ofs, uncSize)
  }

  return
}


// Used internally. Applying RLE-compression to specified buffer data.
func encodeFrame(buf *buffers.Buffer, width, height, ofs int, transColor byte) (data []byte, err error) {
  if width < 1 || height < 1 { err = fmt.Errorf("EncodeFrame: Frame dimension out of range (w=%d, h=%d)", width, height); return }
  if buf.BufferLength() < ofs + width*height { err = fmt.Errorf("EncodeFrame: Buffer too small"); return }
  dstBuf := make([]byte, width*height)
  dstOfs := 0

  totalSize := width * height
  count := 0
  var b byte
  for curOfs := ofs; curOfs < ofs + totalSize; curOfs++ {
    b = buf.GetUint8(curOfs)
    if b == transColor && count < 256 {
      // registering transparent pixel
      count++
    } else {
      if dstOfs + 2 > len(dstBuf) {
        dstBuf = append(dstBuf, make([]byte, width*height)...)
      }

      // encoding pending pixels
      if count > 0 {
        dstBuf[dstOfs] = transColor
        dstBuf[dstOfs + 1] = uint8(count-1)
        dstOfs += 2
        count = 0
      }

      // encoding current pixel
      if b != transColor {
        dstBuf[dstOfs] = b
        dstOfs++
      } else {
        count++
      }
    }
  }

  // handling pending pixels
  if count > 0 {
    if dstOfs + 2 > len(dstBuf) {
      dstBuf = append(dstBuf, make([]byte, 2)...)
    }
    dstBuf[dstOfs] = b
    dstBuf[dstOfs + 1] = uint8(count-1)
    dstOfs += 2
    count = 0
  }

  if dstOfs < len(dstBuf) {
    dstBuf = dstBuf[:dstOfs]
  }

  err = nil
  data = dstBuf
  return
}
