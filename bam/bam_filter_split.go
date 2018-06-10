package bam
/*
Implements filter "split": Allows you to split the BAM into multiple parts.
Options:
- splitW: int (0) - split how many times horizontally [0, 7]
- splitH: int (0) - split how many times vertically [0, 7]
- segmentX: int (0) - zero-based column index of the returned BAM segment
- segmentY: int (0) - zero-based row index of the returned BAM segment
*/

import (
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "strings"
)

const (
  filterNameSplit = "split"
)

type FilterSplit struct {
  options     optionsMap
  opt_splitw, opt_splith, opt_segmentx, opt_segmenty string
}


// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameSplit, NewFilterSplit)
}


// Creates a new Split filter.
func NewFilterSplit() BamFilter {
  f := FilterSplit{options: make(optionsMap),
                   opt_splitw: "splitw",
                   opt_splith: "splith",
                   opt_segmentx: "segmentx",
                   opt_segmenty: "segmenty"}
  f.SetOption(f.opt_splitw, "0")
  f.SetOption(f.opt_splith, "0")
  f.SetOption(f.opt_segmentx, "0")
  f.SetOption(f.opt_segmenty, "0")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterSplit) GetName() string {
  return filterNameSplit
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterSplit) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterSplit) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_splitw, f.opt_splith:
      v, err := parseIntRange(value, 0, 7)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_segmentx, f.opt_segmenty:
      v, err := parseIntRange(value, 0, 7)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterSplit) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut, inFrames)
  return frameOut, err
}


// Used internally. Applies split effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterSplit) apply(frame *BamFrame, inFrames []BamFrame) error {
  divX := f.GetOption(f.opt_splitw).(int) + 1   // store number of resulting segments per axis
  divY := f.GetOption(f.opt_splith).(int) + 1
  segmentX := f.GetOption(f.opt_segmentx).(int)
  segmentY := f.GetOption(f.opt_segmenty).(int)
  if segmentX >= divX { return fmt.Errorf("Column index %d out of range [0, %d]", segmentX, divX - 1) }
  if segmentY >= divY { return fmt.Errorf("Row index %d out of range [0, %d]", segmentY, divY - 1) }
  if divX == 0 && divY == 0 { return nil }
  var transIndex byte = 0

  // Applying global frame size
  sleft, stop, sright, sbottom := getGlobalCanvas(frame.img.Bounds().Dx(), frame.img.Bounds().Dy(), frame.cx, frame.cy, inFrames)
  frame.img = canvasAddBorder(frame.img, sleft, stop, sright, sbottom, transIndex)
  frame.cx += sleft
  frame.cy += stop
  swidth := frame.img.Bounds().Dx()
  sheight := frame.img.Bounds().Dy()
  if divX > swidth || divY > sheight { return fmt.Errorf("Frame dimension too small for split operation.") }

  // Calculating target segment rectangle and center
  segRect := image.Rectangle{}
  segRect.Min.X = segmentX * swidth / divX
  segRect.Min.Y = segmentY * sheight / divY
  segRect.Max.X = (segmentX + 1) * swidth / divX
  segRect.Max.Y = (segmentY + 1) * sheight / divY
  segCenterX := frame.cx - segRect.Min.X
  segCenterY := frame.cy - segRect.Min.Y

  // Trimming excess canvas from segment
  dleft := sleft - segRect.Min.X
  if dleft < 0 { dleft = 0 }
  dtop := stop - segRect.Min.Y
  if dtop < 0 { dtop = 0 }
  dright := segRect.Max.X - swidth + sright
  if dright < 0 { dright = 0 }
  dbottom := segRect.Max.Y - sheight + sbottom
  if dbottom < 0 { dbottom = 0 }
  dwidth := segRect.Dx() - dleft - dright
  dheight := segRect.Dy() - dtop - dbottom
  isEmpty := dwidth <= 0 || dheight <= 0

  // Drawing segment
  var imgDest image.Image = nil
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    pal := make(color.Palette, len(imgPal.Palette))
    copy(pal, imgPal.Palette)
    if isEmpty {
      imgDest = image.NewPaletted(image.Rect(0, 0, 1, 1), pal)
    } else {
      imgDestPal := image.NewPaletted(image.Rect(0, 0, dwidth, dheight), pal)
      draw.Draw(imgDestPal, imgDestPal.Bounds(), imgPal, segRect.Min.Add(image.Pt(dleft, dtop)), draw.Src)
      imgDest = imgDestPal
    }
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    if isEmpty {
      imgDest = image.NewRGBA(image.Rect(0, 0, 1, 1))
    } else {
      imgDestRGBA := image.NewRGBA(image.Rect(0, 0, dwidth, dheight))
      draw.Draw(imgDestRGBA, imgDestRGBA.Bounds(), imgRGBA, segRect.Min.Add(image.Pt(dleft, dtop)), draw.Src)
      imgDest = imgDestRGBA
    }
  }
  segCenterX -= dleft
  segCenterY -= dtop

  frame.img = imgDest
  canvasAddBorder(frame.img, -dleft, -dtop, -dright, -dbottom, transIndex)
  frame.cx = segCenterX
  frame.cy = segCenterY

  return nil
}
