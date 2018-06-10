package bam
/*
Implements filter "replace": Replaces a specific color (palette or rgba) by a given color.
Options:
- match: number or rgba quadruplet (0xff00ff00) - the color value to replace
- color: number or rgba quadruplet (0x00000000) - the replacement color
*/

import (
)

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameReplace = "replace"
)

type FilterReplace struct {
  options   optionsMap
  opt_match, opt_color  string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameReplace, NewFilterReplace)
}


// Creates a new Replace filter.
func NewFilterReplace() BamFilter {
  f := FilterReplace{options: make(optionsMap), opt_match: "match", opt_color: "color"}
  f.SetOption(f.opt_match, "0xff00ff00")
  f.SetOption(f.opt_color, "0x00000000")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterReplace) GetName() string {
  return filterNameReplace
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterReplace) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterReplace) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_match, f.opt_color:
      v, err := parseInt(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterReplace) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(imgOut)
  return frameOut, err
}


// Used internally. Applies color replacement effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterReplace) apply(img image.Image) error {
  v1 := f.GetOption(f.opt_match).(int)
  v2 := f.GetOption(f.opt_color).(int)
  if v1 == v2 { return nil }
  match := []byte{ byte(v1 >> 16), byte(v1 >> 8), byte(v1), byte(v1 >> 24) }
  replace := []byte{ byte(v2 >> 16), byte(v2 >> 8), byte(v2), byte(v2 >> 24) }
  f.premultiply(match)
  f.premultiply(replace)

  if imgPal, ok := img.(*image.Paletted); ok {
    // Apply to palette only
    for idx, size := 0, len(imgPal.Palette); idx < size; idx++ {
      imgPal.Palette[idx] = f.applyColor(imgPal.Palette[idx], match, replace)
    }
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    // apply to RGBA pixels
    x0, x1 := imgRGBA.Bounds().Min.X, imgRGBA.Bounds().Max.X
    y0, y1 := imgRGBA.Bounds().Min.Y, imgRGBA.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * imgRGBA.Stride + x0 * 4
      for x := x0; x < x1; x++ {
        f.applyRGBA(imgRGBA.Pix[ofs:ofs+4], match, replace)
        ofs += 4
      }
    }
  }

  return nil
}


// Applies replace to given color value
func (f *FilterReplace) applyColor(col color.Color, match, replace []byte) color.Color {
  r, g, b, a := col.RGBA()
  if a > 0 {
    if byte(r) == match[0] && byte(g) == match[1] && byte(b) == match[2] && byte(a) == match[3] {
      return color.NRGBA{replace[0], replace[1], replace[2], replace[3] }
    }
  } else {
    if byte(a) == match[3] {
      return color.NRGBA{replace[0], replace[1], replace[2], replace[3] }
    }
  }
  return col
}

// Applies replace to given slice[0:4] of premultiplied RGBA values
func (f *FilterReplace) applyRGBA(slice, match, replace []byte) {
  if slice[3] > 0 {
    if slice[0] == match[0] && slice[1] == match[1] && slice[2] == match[2] && slice[3] == match[3] {
      slice[0], slice[1], slice[2], slice[3] = replace[0], replace[1], replace[2], replace[3]
    }
  } else {
    if slice[3] == match[3] {
      slice[0], slice[1], slice[2], slice[3] = replace[0], replace[1], replace[2], replace[3]
    }
  }
}


// Converts normalized RGBA into premultiplied RGBA
func (f *FilterReplace) premultiply(slice []byte) {
  a := uint(slice[3])
  if a > 0 {
    r, g, b := uint(slice[0]), uint(slice[1]), uint(slice[2])
    r = r * a / 255
    g = g * a / 255
    b = b * a / 255
    slice[0], slice[1], slice[2] = byte(r), byte(g), byte(b)
  } else {
    slice[0], slice[1], slice[2] = 0, 0, 0
  }
}

