package bam
/*
Implements filter "contrast":
Options:
- level: int [-255, 255]
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameContrast = "contrast"
)

type FilterContrast struct {
  options     optionsMap
  opt_level   string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameContrast, NewFilterContrast)
}


// Creates a new Contrast filter.
func NewFilterContrast() BamFilter {
  f := FilterContrast{options: make(optionsMap), opt_level: "level"}
  f.SetOption(f.opt_level, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterContrast) GetName() string {
  return filterNameContrast
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterContrast) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterContrast) SetOption(key, value string) error {
  key = strings.ToLower(key)
  if key == f.opt_level {
    v, err := parseIntRange(value, -255, 255)
    if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
    f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterContrast) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies contrast effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterContrast) apply(img image.Image) error {
  level := float64(f.GetOption(f.opt_level).(int)) / 255.0
  if level == 0.0 { return nil }

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


// Applies contrast to given color value
func (f *FilterContrast) applyColor(col color.Color, level float64) color.Color {
  r, g, b, a := NRGBA(col)
  if a > 0 {
    fr, fg, fb := float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
    fr -= 0.5
    fr *= level
    fr += 0.5
    if fr < 0.0 { fr = 0.0 }
    if fr > 1.0 { fr = 1.0 }
    fg -= 0.5
    fg *= level
    fg += 0.5
    if fg < 0.0 { fg = 0.0 }
    if fg > 1.0 { fg = 1.0 }
    fb -= 0.5
    fb *= level
    fb += 0.5
    if fb < 0.0 { fb = 0.0 }
    if fb > 1.0 { fb = 1.0 }
    r, g, b = byte(fr * 255.0 + 0.5), byte(fg * 255.0 + 0.5), byte(fb * 255.0 + 0.5)
    return color.NRGBA{r, g, b, a}
  } else {
    return col
  }
}

// Applies contrast to given slice[0:4] of premultiplied RGBA values
func (f *FilterContrast) applyRGBA(slice []byte, level float64) {
  r, g, b, a := slice[0], slice[1], slice[2], slice[3]
  a &= 0xff
  if a > 0 {
    fa := float64(a)
    fa2 := fa / 2.0
    fr, fg, fb := float64(r) / fa, float64(g) / fa, float64(b) / fa
    fr -= fa2
    fr *= level
    fr += fa2
    if fr < 0.0 { fr = 0.0 }
    if fr > fa { fr = fa }
    fg -= fa2
    fg *= level
    fg += fa2
    if fg < 0.0 { fg = 0.0 }
    if fg > fa { fg = fa }
    fb -= fa2
    fb *= level
    fb += fa2
    if fb < 0.0 { fb = 0.0 }
    if fb > fa { fb = fa }
    r, g, b = byte(fr * fa + 0.5), byte(fg * fa + 0.5), byte(fb * fa + 0.5)
    slice[0], slice[1], slice[2] = r, g, b
  }
}
