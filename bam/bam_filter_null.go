package bam
/*
Implements filter "null": Simply returns a copy of the input frame entry.
*/

import (
)

const (
  filterNameNull = "null"
)

type FilterNull struct {}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameNull, NewFilterNull)
}


// Creates a new Null filter.
func NewFilterNull() BamFilter {
  f := FilterNull{}
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterNull) GetName() string {
  return filterNameNull
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterNull) GetOption(key string) interface{} {
  // Doesn't have any options
  return nil
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterNull) SetOption(key, value string) error {
  // Doesn't have options
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterNull) Process(index int, frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  return frameOut, nil
}
