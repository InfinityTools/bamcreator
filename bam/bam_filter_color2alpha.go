package bam
/*
Implements filter "color2alpha":
Options: n/a
*/

import (
  "image"
  "image/color"
)

const (
  filterNameColor2alpha = "color2alpha"
)

type FilterColor2alpha struct {
  options     optionsMap
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameColor2alpha, NewFilterColor2alpha)
}


// Creates a new Color2alpha filter.
func NewFilterColor2alpha() BamFilter {
  f := FilterColor2alpha{options: make(optionsMap)}
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterColor2alpha) GetName() string {
  return filterNameColor2alpha
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterColor2alpha) GetOption(key string) interface{} {
  return nil
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterColor2alpha) SetOption(key, value string) error {
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterColor2alpha) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies Color2alpha effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterColor2alpha) apply(img image.Image) error {
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
func (f *FilterColor2alpha) applyColor(col color.Color) color.Color {
  r, g, b, a := col.RGBA()
  a >>= 8
  if a == 255 {
    r >>= 8
    g >>= 8
    b >>= 8
    a = r
    if g > a { a = g }
    if b > a { a = b }
    return color.RGBA{byte(r), byte(g), byte(b), byte(a)}
  }
  return col
}
