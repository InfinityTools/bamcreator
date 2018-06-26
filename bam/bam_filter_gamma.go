package bam
/*
Implements filter "gamma":
Options:
- level: float [0.0001, 5.0]
*/

import (
  "fmt"
  "image"
  "image/color"
  "math"
  "strings"
)

const (
  filterNameGamma = "gamma"
)

type FilterGamma struct {
  options     optionsMap
  opt_level   string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameGamma, NewFilterGamma)
}


// Creates a new Gamma filter.
func NewFilterGamma() BamFilter {
  f := FilterGamma{options: make(optionsMap), opt_level: "level"}
  f.SetOption(f.opt_level, "1.0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterGamma) GetName() string {
  return filterNameGamma
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterGamma) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterGamma) SetOption(key, value string) error {
  key = strings.ToLower(key)
  if key == f.opt_level {
    v, err := parseFloatRange(value, 0.0001, 5.0)
    if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
    f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterGamma) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies gamma effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterGamma) apply(img image.Image) error {
  level := f.GetOption(f.opt_level).(float64)
  if level == 1.0 { return nil }
  level = 1.0 / level

  if imgPal, ok := img.(*image.Paletted); ok {
    // Apply to palette only
    for idx, size := 0, len(imgPal.Palette); idx < size; idx++ {
      imgPal.Palette[idx] = f.applyColor(imgPal.Palette[idx], level)
    }
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    // apply to RGBA pixels
    x0, x1 := imgRGBA.Bounds().Min.X, imgRGBA.Bounds().Max.X
    y0, y1 := imgRGBA.Bounds().Min.Y, imgRGBA.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * imgRGBA.Stride + x0 * 4
      for x := x0; x < x1; x++ {
        f.applyRGBA(imgRGBA.Pix[ofs:ofs+4], level)
        ofs += 4
      }
    }
  }
  return nil
}


// Applies gamma to given color value
func (f *FilterGamma) applyColor(col color.Color, level float64) color.Color {
  r, g, b, a := NRGBA(col)
  if a > 0 {
    fr, fg, fb := float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
    fr = math.Pow(fr, level)
    if fr < 0.0 { fr = 0.0 }
    if fr > 1.0 { fr = 1.0 }
    fg = math.Pow(fg, level)
    if fg < 0.0 { fg = 0.0 }
    if fg > 1.0 { fg = 1.0 }
    fb = math.Pow(fb, level)
    if fb < 0.0 { fb = 0.0 }
    if fb > 1.0 { fb = 1.0 }
    r, g, b = byte(fr * 255.0 + 0.5), byte(fg * 255.0 + 0.5), byte(fb * 255.0 + 0.5)
    return color.NRGBA{r, g, b, a}
  } else {
    return col
  }
}

// Applies gamma to given slice[0:4] of premultiplied RGBA values
func (f *FilterGamma) applyRGBA(slice []byte, level float64) {
  r, g, b, a := slice[0], slice[1], slice[2], slice[3]
  a &= 0xff
  if a > 0 {
    fa := float64(a)
    fr, fg, fb := float64(r) / fa, float64(g) / fa, float64(b) / fa
    fr = math.Pow(fr, level)
    if fr < 0.0 { fr = 0.0 }
    if fr > fa { fr = fa }
    fg = math.Pow(fg, level)
    if fg < 0.0 { fg = 0.0 }
    if fg > fa { fg = fa }
    fb = math.Pow(fb, level)
    if fb < 0.0 { fb = 0.0 }
    if fb > fa { fb = fa }
    r, g, b = byte(fr * fa + 0.5), byte(fg * fa + 0.5), byte(fb * fa + 0.5)
    slice[0], slice[1], slice[2] = r, g, b
  }
}
