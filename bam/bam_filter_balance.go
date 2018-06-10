package bam
/*
Implements filter "balance":
Options:
- red: int [-255, 255]
- green: int [-255, 255]
- blue: int [-255, 255]
- alpha: int [-255, 255]
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameBalance = "balance"
)

type FilterBalance struct {
  options     optionsMap
  opt_red, opt_green, opt_blue, opt_alpha string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameBalance, NewFilterBalance)
}


// Creates a new Balance filter.
func NewFilterBalance() BamFilter {
  f := FilterBalance{options: make(optionsMap),
                     opt_red: "red", opt_green: "green",
                     opt_blue: "blue", opt_alpha: "alpha"}
  f.SetOption(f.opt_red, "0")
  f.SetOption(f.opt_green, "0")
  f.SetOption(f.opt_blue, "0")
  f.SetOption(f.opt_alpha, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterBalance) GetName() string {
  return filterNameBalance
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterBalance) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterBalance) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_red, f.opt_green, f.opt_blue, f.opt_alpha:
      v, err := parseIntRange(value, -255, 255)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterBalance) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies balance effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterBalance) apply(img image.Image) error {
  options := []float64{ float64(f.GetOption(f.opt_red).(int)) / 255.0,
                        float64(f.GetOption(f.opt_green).(int)) / 255.0,
                        float64(f.GetOption(f.opt_blue).(int)) / 255.0,
                        float64(f.GetOption(f.opt_alpha).(int)) / 255.0 }
  if options[0] == 0.0 && options[1] == 0.0 && options[2] == 0.0 && options[3] == 0.0 { return nil }

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


// Applies balance to given color value
func (f *FilterBalance) applyColor(col color.Color, options []float64) color.Color {
  r, g, b, a := NRGBA(col)
  if options[0] != 0.0 {
    fr := float64(r) / 255.0
    fr += options[0]
    if fr < 0.0 { fr = 0.0 }
    if fr > 1.0 { fr = 1.0 }
    r = byte(fr * 255.0 + 0.5)
  }
  if options[1] != 0.0 {
    fg := float64(g) / 255.0
    fg += options[1]
    if fg < 0.0 { fg = 0.0 }
    if fg > 1.0 { fg = 1.0 }
    g = byte(fg * 255.0 + 0.5)
  }
  if options[2] != 0.0 {
    fb := float64(b) / 255.0
    fb += options[2]
    if fb < 0.0 { fb = 0.0 }
    if fb > 1.0 { fb = 1.0 }
    b = byte(fb * 255.0 + 0.5)
  }
  if options[3] != 0.0 {
    fa := float64(a) / 255.0
    fa += options[3]
    if fa < 0.0 { fa = 0.0 }
    if fa > 1.0 { fa = 1.0 }
    a = byte(fa * 255.0 + 0.5)
  }
  return color.NRGBA{r, g, b, a}
}

// Applies balance to given slice[0:4] of premultiplied RGBA values
func (f *FilterBalance) applyRGBA(slice []byte, options []float64) {
  fa2s := float64(slice[3])   // scaled final alpha [0, 255]
  fa := fa2s / 255.0          // normalized initial alpha [0, 1]
  if options[3] != 0.0 {
    fa2 := fa + options[3]
    if fa2 < 0.0 { fa2 = 0.0 }
    if fa2 > 1.0 { fa2 = 1.0 }
    fa2s = fa2 * 255.0
    slice[3] = byte(fa2s)
  }

  if fa2s > 0.0 {
    fr, fg, fb := 0.0, 0.0, 0.0
    if fa > 0.0 {
      fr /= fa
      fg /= fa
      fb /= fa
    }
    fr += options[0]
    fg += options[1]
    fb += options[2]
    if fr < 0.0 { fr = 0.0 }
    if fg < 0.0 { fg = 0.0 }
    if fb < 0.0 { fb = 0.0 }
    if fr > 1.0 { fr = 1.0 }
    if fg > 1.0 { fg = 1.0 }
    if fb > 1.0 { fb = 1.0 }
    slice[0] = byte(fr * fa2s + 0.5)
    slice[1] = byte(fg * fa2s + 0.5)
    slice[2] = byte(fb * fa2s + 0.5)
  } else {
    slice[0], slice[1], slice[2] = 0, 0, 0
  }
}
