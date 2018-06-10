package bam
/*
Implements filter "invert":
Options:
- red: bool (true)
- green: bool (true)
- blue: bool (true)
- alpha: bool (false)
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameInvert = "invert"
)

type FilterInvert struct {
  options   optionsMap
  opt_red, opt_green, opt_blue, opt_alpha string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameInvert, NewFilterInvert)
}


// Creates a new Invert filter.
func NewFilterInvert() BamFilter {
  f := FilterInvert{options: make(optionsMap),
                    opt_red: "red", opt_green: "green",
                    opt_blue: "blue", opt_alpha: "alpha"}
  f.SetOption(f.opt_red, "true")
  f.SetOption(f.opt_green, "true")
  f.SetOption(f.opt_blue, "true")
  f.SetOption(f.opt_alpha, "false")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterInvert) GetName() string {
  return filterNameInvert
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterInvert) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterInvert) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_red, f.opt_green, f.opt_blue, f.opt_alpha:
      v, err := parseBool(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterInvert) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies invert effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterInvert) apply(img image.Image) error {
  options := []bool{ f.GetOption(f.opt_red).(bool), f.GetOption(f.opt_green).(bool),
                     f.GetOption(f.opt_blue).(bool), f.GetOption(f.opt_alpha).(bool) }
  if !(options[0] || options[1] || options[2] || options[3]) { return nil }

  if imgPal, ok := img.(*image.Paletted); ok {
    // Apply to palette only
    for idx, size := 0, len(imgPal.Palette); idx < size; idx++ {
      imgPal.Palette[idx] = f.applyColor(imgPal.Palette[idx], options)
    }
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    // apply to RGBA pixels
    x0, x1 := imgRGBA.Bounds().Min.X, imgRGBA.Bounds().Max.X
    y0, y1 := imgRGBA.Bounds().Min.Y, imgRGBA.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * imgRGBA.Stride + x0 * 4
      for x := x0; x < x1; x++ {
        f.applyRGBA(imgRGBA.Pix[ofs:ofs+4], options)
        ofs += 4
      }
    }
  }
  return nil
}


// Applies invert to given color value
func (f *FilterInvert) applyColor(col color.Color, options []bool) color.Color {
  if nrgba, ok := col.(color.NRGBA); ok {
    retVal := color.NRGBA{nrgba.R, nrgba.G, nrgba.B, nrgba.A}
    if options[0] { retVal.R = 255 - retVal.R }
    if options[1] { retVal.G = 255 - retVal.G }
    if options[2] { retVal.B = 255 - retVal.B }
    if options[3] { retVal.A = 255 - retVal.A }
    return retVal
  } else {
    r, g, b, a := col.RGBA()
    if a > 0 {
      if options[0] { r = a - (r & 0xff) }
      if options[1] { g = a - (g & 0xff) }
      if options[2] { b = a - (b & 0xff) }
      if options[3] {
        a2 := 255 - (a & 0xff)
        if options[0] { r = r * a2 / a }
        if options[0] { g = g * a2 / a }
        if options[0] { b = b * a2 / a }
        a = a2
      }
    }
    return color.RGBA{byte(r), byte(g), byte(b), byte(a)}
  }
}

// Applies invert to given slice[0:4] of premultiplied RGBA values
func (f *FilterInvert) applyRGBA(slice []byte, options []bool) {
  a := slice[3]
  if a > 0 {
    r, g, b := slice[0], slice[1], slice[2]
    if options[0] { r = a - r }
    if options[1] { g = a - g }
    if options[2] { b = a - b }
    if options[3] {
      a2 := 255 - a
      if options[0] { r = r * a2 / a }
      if options[1] { g = g * a2 / a }
      if options[2] { b = b * a2 / a }
      a = a2
    }
    slice[0], slice[1], slice[2] = r, g, b
  } else {
    if options[3] { a = 255 - a }
  }
  slice[3] = a
}
