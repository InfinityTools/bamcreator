package bam
/*
Implements filter "mirror":
Options:
- horizontal: bool (false) - mirror horizontally
- vertical: bool (false) - mirror vertically
- updatecenter: bool (true) - whether to update center position
*/

import (
  "fmt"
  "image"
  "strings"
)

const (
  filterNameMirror = "mirror"
)

type FilterMirror struct {
  options     optionsMap
  opt_horizontal, opt_vertical, opt_updatecenter string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameMirror, NewFilterMirror)
}


// Creates a new Mirror filter.
func NewFilterMirror() BamFilter {
  f := FilterMirror{options: make(optionsMap), opt_horizontal: "horizontal", opt_vertical: "vertical", opt_updatecenter: "updatecenter"}
  f.SetOption(f.opt_horizontal, "false")
  f.SetOption(f.opt_vertical, "false")
  f.SetOption(f.opt_updatecenter, "true")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterMirror) GetName() string {
  return filterNameMirror
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterMirror) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterMirror) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_horizontal, f.opt_vertical, f.opt_updatecenter:
      v, err := parseBool(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterMirror) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut)
  return frameOut, err
}


// Used internally. Applies mirror effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterMirror) apply(frame *BamFrame) error {
  options := []bool{f.GetOption(f.opt_horizontal).(bool), f.GetOption(f.opt_vertical).(bool), f.GetOption(f.opt_updatecenter).(bool)}
  if !(options[0] || options[1]) { return nil }

  if imgPal, ok := frame.img.(*image.Paletted); ok {
    frame.cx, frame.cy = f.applyPaletted(imgPal, frame.cx, frame.cy, options)
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    frame.cx, frame.cy = f.applyRGBA(imgRGBA, frame.cx, frame.cy, options)
  }

  return nil
}


// Applies mirror to paletted image. Returns updated center positions.
func (f *FilterMirror) applyPaletted(img *image.Paletted, cx, cy int, options []bool) (cxNew, cyNew int) {
  cxNew, cyNew = cx, cy

  if options[0] { // mirror horizontally
    x0, x1 := img.Bounds().Min.X, img.Bounds().Max.X
    y0, y1 := img.Bounds().Min.Y, img.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * img.Stride + x0
      for left, right := x0, x1-1; left < right; {
        img.Pix[ofs+left], img.Pix[ofs+right] = img.Pix[ofs+right], img.Pix[ofs+left]
        left++
        right--
      }
    }
    if options[2] { // update center
      cxNew = img.Bounds().Dx() - cxNew
    }
  }

  if options[1] { // mirror vertically
    x0, x1 := img.Bounds().Min.X, img.Bounds().Max.X
    y0, y1 := img.Bounds().Min.Y, img.Bounds().Max.Y
    size := x1 - x0
    buf := make([]byte, size)
    ofsTop := y0 * img.Stride + x0
    ofsBottom := (y1 - 1) * img.Stride + x0
    for ofsTop < ofsBottom {
      copy(buf, img.Pix[ofsTop:ofsTop+size])
      copy(img.Pix[ofsTop:ofsTop+size], img.Pix[ofsBottom:ofsBottom+size])
      copy(img.Pix[ofsBottom:ofsBottom+size], buf)
      ofsTop += img.Stride
      ofsBottom -= img.Stride
    }
    if options[2] { // update center
      cyNew = img.Bounds().Dy() - cyNew
    }
  }

  return
}

// Applies mirror to paletted image. Returns updated center positions.
func (f *FilterMirror) applyRGBA(img *image.RGBA, cx, cy int, options []bool) (cxNew, cyNew int) {
  cxNew, cyNew = cx, cy

  if options[0] { // mirror horizontally
    x0, x1 := img.Bounds().Min.X, img.Bounds().Max.X
    y0, y1 := img.Bounds().Min.Y, img.Bounds().Max.Y
    for y := y0; y < y1; y++ {
      ofs := y * img.Stride + (x0 * 4)
      for left, right := x0*4, (x1-1)*4; left < right; {
        img.Pix[ofs+left], img.Pix[ofs+right] = img.Pix[ofs+right], img.Pix[ofs+left]
        img.Pix[ofs+left+1], img.Pix[ofs+right+1] = img.Pix[ofs+right+1], img.Pix[ofs+left+1]
        img.Pix[ofs+left+2], img.Pix[ofs+right+2] = img.Pix[ofs+right+2], img.Pix[ofs+left+2]
        img.Pix[ofs+left+3], img.Pix[ofs+right+3] = img.Pix[ofs+right+3], img.Pix[ofs+left+3]
        left += 4
        right -= 4
      }
    }
    if options[2] { // update center
      cxNew = img.Bounds().Dx() - cxNew
    }
  }

  if options[1] { // mirror vertically
    x0, x1 := img.Bounds().Min.X, img.Bounds().Max.X
    y0, y1 := img.Bounds().Min.Y, img.Bounds().Max.Y
    size := (x1 - x0) * 4
    buf := make([]byte, size)
    ofsTop := y0 * img.Stride + (x0 * 4)
    ofsBottom := (y1 - 1) * img.Stride + (x0 * 4)
    for ofsTop < ofsBottom {
      copy(buf, img.Pix[ofsTop:ofsTop+size])
      copy(img.Pix[ofsTop:ofsTop+size], img.Pix[ofsBottom:ofsBottom+size])
      copy(img.Pix[ofsBottom:ofsBottom+size], buf)
      ofsTop += img.Stride
      ofsBottom -= img.Stride
    }
    if options[2] { // update center
      cyNew = img.Bounds().Dy() - cyNew
    }
  }

  return
}
