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
func (f *FilterContrast) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies contrast effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterContrast) apply(img image.Image) error {
  level := float64(f.GetOption(f.opt_level).(int)) / 255.0 + 1.0
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
  r, g, b, a := col.RGBA()
  if a > 0 {
    slice := []byte{byte(r >> 8), byte(g >> 8), byte(b >> 8), byte(a >> 8)}
    f.applyRGBA(slice, level)
    return color.RGBA{slice[0], slice[1], slice[2], slice[3]}
  }
  return col
}

// Applies contrast to given slice[0:4] of premultiplied RGBA values
func (f *FilterContrast) applyRGBA(slice []byte, level float64) {
  a := slice[3]
  if a > 0 {
    fa := float64(a)
    fr, fg, fb := float64(slice[0]) / fa, float64(slice[1]) / fa, float64(slice[2]) / fa
    fr -= 0.5
    fr *= level
    fr += 0.5
    if fr < 0.0 { fr = 0.0 }
    if fr > 1.0 { fr = 1.0 }
    slice[0] = byte(fr * fa + 0.5)
    fg -= 0.5
    fg *= level
    fg += 0.5
    if fg < 0.0 { fg = 0.0 }
    if fg > 1.0 { fg = 1.0 }
    slice[1] = byte(fg * fa + 0.5)
    fb -= 0.5
    fb *= level
    fb += 0.5
    if fb < 0.0 { fb = 0.0 }
    if fb > 1.0 { fb = 1.0 }
    slice[2] = byte(fb * fa + 0.5)
  }
}
