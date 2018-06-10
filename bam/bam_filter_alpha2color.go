package bam
/*
Implements filter "alpha2color":
Options: n/a
*/

import (
  "image"
  "image/color"
)

const (
  filterNameAlpha2Color = "alpha2color"
)

type FilterAlpha2Color struct {
  options     optionsMap
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameAlpha2Color, NewFilterAlpha2Color)
}


// Creates a new Alpha2Color filter.
func NewFilterAlpha2Color() BamFilter {
  f := FilterAlpha2Color{options: make(optionsMap)}
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterAlpha2Color) GetName() string {
  return filterNameAlpha2Color
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterAlpha2Color) GetOption(key string) interface{} {
  return nil
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterAlpha2Color) SetOption(key, value string) error {
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterAlpha2Color) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies Alpha2Color effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterAlpha2Color) apply(img image.Image) error {
  if imgPal, ok := img.(*image.Paletted); ok {
    // Apply to palette only
    for idx, size := 0, len(imgPal.Palette); idx < size; idx++ {
      imgPal.Palette[idx] = f.applyColor(imgPal.Palette[idx])
    }
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    // apply to RGBA pixels
    x0, x1 := imgRGBA.Bounds().Min.X, imgRGBA.Bounds().Max.X
    y0, y1 := imgRGBA.Bounds().Min.Y, imgRGBA.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * imgRGBA.Stride + x0 * 4
      for x := x0; x < x1; x++ {
        r, g, b, a := f.applyColor(imgRGBA.At(x, y)).RGBA()
        imgRGBA.Pix[ofs] = byte(r >> 8)
        imgRGBA.Pix[ofs+1] = byte(g >> 8)
        imgRGBA.Pix[ofs+2] = byte(b >> 8)
        imgRGBA.Pix[ofs+3] = byte(a >> 8)
        ofs += 4
      }
    }
  }
  return nil
}


// Applies conversion to given color value
func (f *FilterAlpha2Color) applyColor(col color.Color) color.Color {
  r, g, b, a := col.RGBA()
  if a > 0 { a = 255 }
  return color.RGBA{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a)}
}
