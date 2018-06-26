package bam
/*
Implements filter "saturation":
Options:
- level: int [-100, 100]
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameSaturation = "saturation"
)

type FilterSaturation struct {
  options     optionsMap
  opt_level   string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameSaturation, NewFilterSaturation)
}


// Creates a new Saturation filter.
func NewFilterSaturation() BamFilter {
  f := FilterSaturation{options: make(optionsMap), opt_level: "level"}
  f.SetOption(f.opt_level, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterSaturation) GetName() string {
  return filterNameSaturation
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterSaturation) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterSaturation) SetOption(key, value string) error {
  key = strings.ToLower(key)
  if key == f.opt_level {
    v, err := parseIntRange(value, -100, 100)
    if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
    f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterSaturation) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies saturation effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterSaturation) apply(img image.Image) error {
  level := float64(f.GetOption(f.opt_level).(int)) / 100.0
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


// Applies saturation to given color value
func (f *FilterSaturation) applyColor(col color.Color, level float64) color.Color {
  _, _, _, a := col.RGBA()
  if a > 0 {
    h, s, l := colorToHSL(col)
    s += level
    if s < 0.0 { s = 0.0 }
    if s > 1.0 { s = 1.0 }
    fr, fg, fb := hslToRGB(h, s, l)
    return color.NRGBA{byte(fr * 255.0 + 0.5), byte(fg * 255.0 + 0.5), byte(fb * 255.0 + 0.5), byte(a)}
  } else {
    return col
  }
}

// Applies saturation to given slice[0:4] of premultiplied RGBA values
func (f *FilterSaturation) applyRGBA(slice []byte, level float64) {
  if slice[3] > 0 {
    h, s, l := rgbaToHSL(slice)
    s += level
    if s < 0.0 { s = 0.0 }
    if s > 1.0 { s = 1.0 }
    fa := float64(slice[3])
    fr, fg, fb := hslToRGB(h, s, l)
    slice[0], slice[1], slice[2] = byte(fr * fa + 0.5), byte(fg * fa + 0.5), byte(fb * fa + 0.5)
  }
}
