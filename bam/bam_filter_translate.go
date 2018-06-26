package bam
/*
Implements filter "translate": Moves center position by the specified amount.
*/

import (
  "fmt"
  "strings"
)

const (
  filterNameTranslate = "translate"
)

type FilterTranslate struct {
  options     optionsMap
  opt_x, opt_y string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameTranslate, NewFilterTranslate)
}


// Creates a new Translate filter.
func NewFilterTranslate() BamFilter {
  f := FilterTranslate{options: make(optionsMap), opt_x: "x", opt_y: "y"}
  f.SetOption(f.opt_x, "0")
  f.SetOption(f.opt_y, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterTranslate) GetName() string {
  return filterNameTranslate
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterTranslate) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterTranslate) SetOption(key, value string) error {
  switch key {
    case f.opt_x, f.opt_y:
      v, err := parseInt(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterTranslate) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut)
  return frameOut, err
}

func (f *FilterTranslate) apply(frame *BamFrame) error {
  x := f.GetOption(f.opt_x).(int)
  y := f.GetOption(f.opt_y).(int)
  if x == 0 && y == 0 { return nil }

  frame.cx += x
  frame.cy += y

  return nil
}
