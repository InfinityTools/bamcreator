package bam
// Provides functionality for palette and color operations.

import (
  "fmt"
  "image"
  "image/color"

  "github.com/InfinityTools/bamcreator/palette/sort"
  "github.com/InfinityTools/go-imagequant"
  "github.com/InfinityTools/go-logging"
)


// NRGBA converts a premultiplied color back to a normalized color with each component in range [0, 255].
func NRGBA(col color.Color) (r, g, b, a byte) {
  if nrgba, ok := col.(color.NRGBA); ok {
    r, g, b, a = nrgba.R, nrgba.G, nrgba.B, nrgba.A
  } else {
    pr, pg, pb, pa := col.RGBA()
    pa >>= 8
    if pa > 0 {
      pr >>= 8
      pr *= 0xff
      pr /= pa
      pg >>= 8
      pg *= 0xff
      pg /= pa
      pb >>= 8
      pb *= 0xff
      pb /= pa
    }
    r = byte(pr)
    g = byte(pg)
    b = byte(pb)
    a = byte(pa)
  }
  return
}


// Used internally. Defers paletted image generation to more specialized functions.
func (bam *BamFile) generatePalette(frames []BamFrame) (imgList []*image.Paletted, pal color.Palette, err error) {
  logging.Logln("Starting palette generation")

  if bam.bamV1.palette != nil {
    imgList, pal, err = bam.remapPalette(frames, bam.bamV1.palette)
  } else {
    imgList, pal, err = bam.quantizePalette(frames)
  }
  if err != nil { return }

  logging.Logln("Finished palette generation")
  return
}


// Used internally. Generate palette and paletted output images from available input images.
func (bam *BamFile) quantizePalette(frames []BamFrame) (imgList []*image.Paletted, pal color.Palette, err error) {
  att := imagequant.CreateAttributes()
  defer att.Release()

  // Initial quantization settings
  err = att.SetMaxColors(256)
  if err != nil { return }
  err = att.SetQuality(bam.bamV1.qualityMin, bam.bamV1.qualityMax)
  if err != nil { return }
  err = att.SetSpeed(bam.bamV1.speed)
  if err != nil { return }

  // Quantization may fail if minimum quality is too high. Retrying with updated quality settings if needed.
  var qimgList []*imagequant.Image = nil
  var res *imagequant.Result = nil
  for {
    // Adding fixed color entries to histogram (transparent "green" and optional custom entries)
    hist := att.CreateHistogram()
    att.AddColorsToHistogram(hist, []imagequant.HistogramEntry{ imagequant.HistogramEntry{Color: color.RGBA{0, 0, 0, 0}, Count: 256} }, 0.0)
    for _, col := range bam.bamV1.fixedColors {
      att.AddColorsToHistogram(hist, []imagequant.HistogramEntry{ imagequant.HistogramEntry{Color: col, Count: 256} }, 0.0)
    }

    // Preparing input frames
    logging.Log("Preparing input frames")
    qimgList = make([]*imagequant.Image, len(frames))
    for i, f := range frames {
      logging.LogProgressDot(i, len(frames), 79-22)  // 22 is length of prefixed string
      qimgList[i] = att.CreateImage(f.img, 0.0)
      if qimgList[i] == nil { err = fmt.Errorf("Unable to process input frame #%d", i); return }

      // Adding image to histogram
      err = att.AddImageToHistogram(hist, qimgList[i])
      if err != nil { return }
    }
    logging.OverridePrefix(false, false, false).Logln("")

    logging.Logf("Calculating output palette%s\n", logging.ProgressDot(0, 1, 79 - 26))

    res, err = att.QuantizeHistogram(hist)
    if qmin, qmax := att.GetQuality(); err == imagequant.ErrQualityTooLow && qmin > 0 {
      if qspeed := att.GetSpeed(); qspeed > 1 {
        att.SetSpeed(qspeed / 2)
      }
      if qmin >= 5 {
        qmin -= 5
      } else {
        qmin = 0
      }
      att.SetQuality(qmin, qmax)
      logging.Warnf("Quantization failed. Trying again with reduced quality: %d\n", qmin)
    } else {
      break
    }
  }
  if err != nil { return }

  // Making final adjustments
  err = att.SetDitheringLevel(res, bam.bamV1.dither)
  if err != nil { return }

  // Creating paletted output frames
  logging.Log("Generating output frames")
  imgList = make([]*image.Paletted, len(qimgList))
  palSrc := att.GetPalette(res)
  if len(palSrc) == 0 { logging.Logln(""); err = fmt.Errorf("Error generating output palette"); return }

  // Reordering palette to satisfy BAM standards
  pal = bam.getNormalizedPalette(palSrc)

  // Sorting palette
  fixedCols := len(bam.bamV1.fixedColors) + 1
  pal, _ = sort.Sort(pal, bam.bamV1.sortFlags, fixedCols)

  remap := bam.remapColors(palSrc, pal, nil)
  for i, qimg := range qimgList {
    logging.LogProgressDot(i, len(frames), 79-24)  // 24 is length of prefixed string
    var img image.Image
    img, err = att.WriteRemappedImage(res, qimg)
    if err != nil { logging.Logln(""); return }
    imgList[i] = bam.getRemappedImage(img, remap)
  }
  logging.OverridePrefix(false, false, false).Logln("")

  return
}


// Used internally. Remap input frames directly to specified color table.
func (bam *BamFile) remapPalette(frames []BamFrame, palIn color.Palette) (imgList []*image.Paletted, palOut color.Palette, err error) {
  palOut = palIn
  att := imagequant.CreateAttributes()
  defer att.Release()

  // Initial quantization settings
  err = att.SetMaxColors(len(palOut))
  if err != nil { return }
  err = att.SetQuality(0, 100)  // Setting minquality to 0 to ensure a successful quantization
  if err != nil { return }

  hist := att.CreateHistogram()
  histEntries := make([]imagequant.HistogramEntry, len(palOut))
  for i := 0; i < len(palOut); i++ {
    histEntries[i] = imagequant.HistogramEntry{Color: palOut[i], Count: 256}
  }
  att.AddColorsToHistogram(hist, histEntries, 0.0)

  // Preparing input frames
  logging.Log("Preparing input frames")
  qimgList := make([]*imagequant.Image, len(frames))
  for i, f := range frames {
    logging.LogProgressDot(i, len(frames), 79-22)  // 22 is length of prefixed string
    qimgList[i] = att.CreateImage(f.img, 0.0)
    if qimgList[i] == nil { err = fmt.Errorf("Unable to process input frame #%d", i); return }
  }
  logging.OverridePrefix(false, false, false).Logln("")

  var res *imagequant.Result = nil
  res, err = att.QuantizeHistogram(hist)
  if err != nil { return }

  // Making final adjustments
  err = att.SetDitheringLevel(res, bam.bamV1.dither)
  if err != nil { return }

  // Creating paletted output frames
  logging.Log("Generating output frames")
  palSrc := att.GetPalette(res)
  if len(palSrc) == 0 { logging.Logln(""); err = fmt.Errorf("Error generating output palette"); return }

  // Reordering palette to match external palette
  remap := bam.remapColors(palSrc, palOut, nil)

  imgList = make([]*image.Paletted, len(qimgList))
  for i, qimg := range qimgList {
    logging.LogProgressDot(i, len(frames), 79-24)  // 24 is length of prefixed string
    var img image.Image
    img, err = att.WriteRemappedImage(res, qimg)
    if err != nil { logging.Logln(""); return }
    imgList[i] = bam.getRemappedImage(img, remap)
  }
  logging.OverridePrefix(false, false, false).Logln("")

  return
}

// Used internally. Returns a structure that remaps color order of palSrc to palDst.
// Specify a remap structure if palSrc has already been remapped via getRemappedPalette().
func (bam *BamFile) remapColors(palSrc, palDst color.Palette, remap sort.ColorMapping) sort.ColorMapping {
  if palSrc == nil { return nil }

  remapOut := make(sort.ColorMapping)
  if remap != nil {
    for k, v := range remap {
      remapOut[k] = v
    }
  } else {
    for i := 0; i < len(palSrc); i++ {
      remapOut[i] = i
    }
  }

  if palDst == nil { return remapOut }

  for i := 0; i < len(palSrc); i++ {
    if idx, ok := remapOut[i]; ok {
      newIdx := palDst.Index(palSrc[idx])
      remapOut[i] = newIdx
    }
  }

  return remapOut
}

// Used internally. Applies remap structure to given palette. Returns remapped palette.
func (bam *BamFile) getRemappedPalette(pal color.Palette, remap sort.ColorMapping) color.Palette {
  if pal == nil { return nil }
  if remap == nil { return pal }

  palOut := make(color.Palette, 256)
  for i, col := range pal {
    if idx, ok := remap[i]; ok {
      if idx >= 0 && idx < 256 {
        palOut[idx] = col
      }
    }
  }

  return palOut
}

// Used internally. Applies remap structure to palette and pixels of the given image. Returns the update image.
func (bam *BamFile) getRemappedImage(img image.Image, remap sort.ColorMapping) *image.Paletted {
  if imgPal, ok := img.(*image.Paletted); ok {
    imgPal.Palette = bam.getRemappedPalette(imgPal.Palette, remap)
    x0, x1 := imgPal.Bounds().Min.X, imgPal.Bounds().Max.X
    y0, y1 := imgPal.Bounds().Min.Y, imgPal.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * imgPal.Stride
      for x := x0; x < x1; x++ {
        px := int(imgPal.Pix[ofs+x])
        if idx, ok := remap[px]; ok && idx >= 0 && idx < 256 {
          px = idx
        }
        imgPal.Pix[ofs+x] = byte(px)
      }
    }
    return imgPal
  }
  return nil
}


// Used internally. Returns a reordered palette where index 0 contains transparent entry and index 1 and up contain
// fixed colors if available. This palette can be used for remapColors() to generate a remap structure.
func (bam *BamFile) getNormalizedPalette(pal color.Palette) color.Palette {
  if pal == nil { return pal }
  palOut := make(color.Palette, len(pal))
  copy(palOut, pal)

  // Remapping transparent color entry
  idx := palOut.Index(color.RGBA{0, 0, 0, 0})
  if idx != 0 {
    palOut[0], palOut[idx] = palOut[idx], palOut[0]
  }

  // Remapping fixed color entries
  for i := 0; i < len(bam.bamV1.fixedColors); i++ {
    idx := palOut[i+1:].Index(bam.bamV1.fixedColors[i])
    if idx > 0 {
      idx += i+1
      palOut[i+1], palOut[idx] = palOut[idx], palOut[i+1]
    }
  }

  return palOut
}


// Used internally. A helper function that creates an Image object with palette out of raw pixel data for use in the
// BAM frame functions.
func createPaletteImage(width, height int, data[]byte, palette[]uint32) image.Image {
  if width < 1 || height < 1 { return nil }
  if data == nil || len(data) < width*height || palette == nil { return nil }

  // preparing palette
  cm := make(color.Palette, 256)
  size := len(palette)
  if size > 256 { size = 256 }
  for i := 0; i < size; i++ {
    // reshuffling components: 0xAARRGGBB -> R, G, B, A
    cm[i] = color.RGBA{ byte(palette[i] >> 16), byte(palette[i] >> 8), byte(palette[i]), byte(palette[i] >> 24) }
  }
  for i := size; i < 256; i++ { cm[i] = color.RGBA{0, 0, 0, 0} }
  img := image.NewPaletted(image.Rect(0, 0, width, height), cm)

  // preparing pixel data
  ofsSrc, ofsDst := 0, 0
  for y := 0; y < height; y++ {
    copy(img.Pix[ofsDst:ofsDst+img.Stride], data[ofsSrc:ofsSrc+width])
    ofsSrc += width
    ofsDst += img.Stride
  }

  return img
}


// Used internally. Converts a paletted image into a byte buffer.
func palettedToBytes(img *image.Paletted) []byte {
  w := img.Bounds().Max.X - img.Bounds().Min.X
  h := img.Bounds().Max.Y - img.Bounds().Min.Y
  buffer := make([]byte, w * h)

  ofsSrc, ofsDst := 0, 0
  for y := 0; y < h; y++ {
    copy(buffer[ofsDst:ofsDst+w], img.Pix[ofsSrc:ofsSrc+img.Stride])
    ofsSrc += img.Stride
    ofsDst += w
  }

  return buffer
}
