package bam
/*
Implements filter "posterize":
Options:
- level: int [0, 7] (0)
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNamePosterize = "posterize"
)

type FilterPosterize struct {
  options       optionsMap
  opt_level  string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNamePosterize, NewFilterPosterize)
}


// Creates a new Posterize filter.
func NewFilterPosterize() BamFilter {
  f := FilterPosterize{options: make(optionsMap), opt_level: "level"}
  f.SetOption(f.opt_level, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterPosterize) GetName() string {
  return filterNamePosterize
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterPosterize) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterPosterize) SetOption(key, value string) error {
  key = strings.ToLower(key)
  if key == f.opt_level {
    v, err := parseIntRange(value, 0, 7)
    if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
    f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterPosterize) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies Posterize effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterPosterize) apply(img image.Image) error {
  level := uint(f.GetOption(f.opt_level).(int))
  if level == 0 { return nil }

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
        r, g, b, a := f.applyColor(imgRGBA.At(x, y), level).RGBA()
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


// Applies Posterize to given color value
func (f *FilterPosterize) applyColor(col color.Color, level uint) color.Color {
  mask := byte((1 << level) - 1)
  imask := ^mask
  r, g, b, a := NRGBA(col)
  return color.NRGBA{r & imask, g & imask, b & imask, a | mask}
}
