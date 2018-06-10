package bam
/*
Implements filter "canvas": Applies options that affect the canvas around the image
Options:
- trim: bool (false) - remove regions of transparent pixels around the frame
- borderleft: int (0) - adds a border of fixed size on the left side of the frame
- bordertop: int (0) - adds a border of fixed size at the top of the frame
- borderright: int (0) - adds a border of fixed size on the right side of the frame
- borderbottom: int (0) - adds a border of fixed size at the bottom of the frame
- minwidth: int (0) - ensures that the frame is at least "minwidth" wide
- minheight: int (0) - ensures that the frame is at least "minheight" high
- horizontalalign: string (center) - for minwidth: align left (left), align center (center), align right (right)
- verticalalign: string (center) - for minheight: align top (top), align center (center), align bottom (bottom)
- updatecenter: bool (true) - whether to update center position
*/

import (
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "strings"
)

const (
  filterNameCanvas = "canvas"
)

// Available alignment options for filter "canvas".
const (
  ALIGN_CENTER  = "center"
  ALIGN_LEFT    = "left"
  ALIGN_RIGHT   = "right"
  ALIGN_TOP     = "top"
  ALIGN_BOTTOM  = "bottom"
)

type FilterCanvas struct {
  options     optionsMap
  opt_trim, opt_minwidth, opt_minheight string
  opt_horizontalalign, opt_verticalalign string
  opt_borderleft, opt_bordertop, opt_borderright, opt_borderbottom string
  opt_updatecenter string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameCanvas, NewFilterCanvas)
}


// Creates a new Canvas filter.
func NewFilterCanvas() BamFilter {
  f := FilterCanvas{options: make(optionsMap),
                    opt_trim: "trim",
                    opt_minwidth: "minwidth", opt_minheight: "minheight",
                    opt_horizontalalign: "horizontalalign", opt_verticalalign: "verticalalign",
                    opt_borderleft: "borderleft", opt_bordertop: "bordertop",
                    opt_borderright: "borderright", opt_borderbottom: "borderbottom",
                    opt_updatecenter: "updatecenter"}
  f.SetOption(f.opt_trim, "false")
  f.SetOption(f.opt_minwidth, "0")
  f.SetOption(f.opt_minheight, "0")
  f.SetOption(f.opt_horizontalalign, ALIGN_CENTER)
  f.SetOption(f.opt_verticalalign, ALIGN_CENTER)
  f.SetOption(f.opt_borderleft, "0")
  f.SetOption(f.opt_bordertop, "0")
  f.SetOption(f.opt_borderright, "0")
  f.SetOption(f.opt_borderbottom, "0")
  f.SetOption(f.opt_updatecenter, "true")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterCanvas) GetName() string {
  return filterNameCanvas
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterCanvas) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterCanvas) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_trim, f.opt_updatecenter:
      v, err := parseBool(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_minwidth, f.opt_minheight, f.opt_borderleft,
         f.opt_bordertop, f.opt_borderright, f.opt_borderbottom:
      v, err := parseIntRange(value, 0, 65535)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_horizontalalign:
      value = strings.ToLower(value)
      switch value {
        case ALIGN_LEFT, ALIGN_CENTER, ALIGN_RIGHT:
          f.options[key] = value
        default:
          return fmt.Errorf("Option %s: %v", key, value)
      }
    case f.opt_verticalalign:
      value = strings.ToLower(value)
      switch value {
        case ALIGN_TOP, ALIGN_CENTER, ALIGN_BOTTOM:
          f.options[key] = value
        default:
          return fmt.Errorf("Option %s: %v", key, value)
      }
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterCanvas) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut)
  return frameOut, err
}


// Used internally. Applies canvas effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterCanvas) apply(frame *BamFrame) error {
  trim := f.GetOption(f.opt_trim).(bool)
  center := f.GetOption(f.opt_updatecenter).(bool)
  border := make([]int, 4)
  border[0] = f.GetOption(f.opt_borderleft).(int)
  border[1] = f.GetOption(f.opt_bordertop).(int)
  border[2] = f.GetOption(f.opt_borderright).(int)
  border[3] = f.GetOption(f.opt_borderbottom).(int)
  dim := make([]int, 2)
  dim[0] = f.GetOption(f.opt_minwidth).(int)
  dim[1] = f.GetOption(f.opt_minheight).(int)
  align := make([]string, 2)
  align[0] = f.GetOption(f.opt_horizontalalign).(string)
  align[1] = f.GetOption(f.opt_verticalalign).(string)
  if !trim && dim[0] < 2 && dim[1] < 2 &&
     border[0] == 0 && border[1] == 0 &&
     border[2] == 0 && border[3] == 0 { return nil }

  // Check availability of transparent palette entry
  var transIndex int = -1
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    for idx, col := range imgPal.Palette {
      _, _, _, a := col.RGBA()
      if a == 0 {
        transIndex = idx
        break
      }
    }
    if transIndex < 0 {
      // Not found: convert to RGBA image?
      if border[0] > 0 || border[1] > 0 || border[2] > 0 || border[3] > 0 ||
         dim[0] > imgPal.Bounds().Dx() || dim[1] > imgPal.Bounds().Dy() {
        img := image.NewRGBA(imgPal.Bounds())
        draw.Draw(img, imgPal.Bounds(), imgPal, imgPal.Bounds().Min, draw.Src)
        frame.img = img
      }
    }
  }

  // Trimming excess canvas regions
  if trim {
    var left, top int
    frame.img, left, top, _, _ = canvasTrim(frame.img, byte(transIndex))
    if center {
      frame.cx -= left
      frame.cy -= top
    }
  }

  // Adding border canvas
  frame.img = canvasAddBorder(frame.img, border[0], border[1], border[2], border[3], byte(transIndex))
  if center {
    frame.cx += border[0]
    frame.cy += border[1]
  }

  // Enforcing minimum image dimensions
  dw := dim[0] - frame.img.Bounds().Dx()
  dh := dim[1] - frame.img.Bounds().Dy()
  if dw > 0 && dh > 0 {
    left, top, right, bottom := 0, 0, 0, 0
    if dw > 0 {
      switch align[0] {
        case ALIGN_LEFT:
          right += dw
        case ALIGN_RIGHT:
          left += dw
        default:
          dw2 := dw / 2
          left += dw2
          right += dw - dw2
      }
    }
    if dh > 0 {
      switch align[1] {
        case ALIGN_TOP:
          bottom += dh
        case ALIGN_BOTTOM:
          top += dh
        default:
          dh2 := dh / 2
          top += dh2
          bottom += dh - dh2
      }
    }
    frame.img = canvasAddBorder(frame.img, left, top, right, bottom, byte(transIndex))
    if center {
      frame.cx += left
      frame.cy += top
    }
  }

  return nil
}


// Used internally. Based on list of given BAM frames, calculates the space to add to each side of the given center,
// so that each frame in the list would fit without overlapping. It can be used to avoid jittering center frames for
// resized BAM frames. Implementation is based on algorithm used in the tool "bamresize", by Avenger_teambg and Sam.
func getGlobalCanvas(width, height, cx, cy int, frames []BamFrame) (left, top, right, bottom int) {
  // fmt.Printf("DEBUG: getGlobalCanvas(%d, %d, %d, %d, frames)\n", width, height, cx, cy)
  maxCenterX, maxCenterY := 0, 0
  for _, curFrame := range frames {
    if curFrame.cx > maxCenterX { maxCenterX = curFrame.cx }
    if curFrame.cy > maxCenterY { maxCenterY = curFrame.cy }
  }
  maxWidth, maxHeight := 0, 0
  for _, curFrame := range frames {
    l := maxCenterX - curFrame.cx
    t := maxCenterY - curFrame.cy
    if w := curFrame.img.Bounds().Dx() + l; w > maxWidth { maxWidth = w }
    if h := curFrame.img.Bounds().Dy() + t; h > maxHeight { maxHeight = h }
  }
  left = maxCenterX - cx
  top = maxCenterY - cy
  width += left
  height += top
  right = maxWidth - width
  bottom = maxHeight - height
  return
}


// Used internally. Adds or removes borders around the frame, depending on specified parameters.
// Use positive values to add space. Use negative values to remove space regardless of content.
func canvasAddBorder(img image.Image, left, top, right, bottom int, transIndex byte) image.Image {
  // fmt.Printf("DEBUG: canvasAddBorder(img, %d, %d, %d, %d, %d)\n", left, top, right, bottom, transIndex)
  imgOut := img

  rc := imgOut.Bounds()
  rc.Min.X -= left
  rc.Min.Y -= top
  rc.Max.X += right
  rc.Max.Y += bottom
  isEmpty := false
  if rc.Empty() {
    isEmpty = true
    rc = image.Rect(0, 0, 1, 1)
  }

  if !rc.Eq(imgOut.Bounds()) {
    var imgNew draw.Image
    if imgPal, ok := imgOut.(*image.Paletted); ok {
      imgPalNew := image.NewPaletted(image.Rect(0, 0, rc.Dx(), rc.Dy()), make(color.Palette, len(imgPal.Palette)))
      copy(imgPalNew.Palette, imgPal.Palette)
      if transIndex > 0 {
        for i, _ := range imgPalNew.Pix { imgPalNew.Pix[i] = transIndex }
      }
      imgNew = imgPalNew
    } else {
      imgNew = image.NewRGBA(image.Rect(0, 0, rc.Dx(), rc.Dy()))
    }

    if !isEmpty {
      rcX := rc.Intersect(imgOut.Bounds())
      if !rcX.Empty() {
        pt := image.Point{-left, -top}
        if pt.X < 0 { pt.X = 0 }
        if pt.Y < 0 { pt.Y = 0 }
        draw.Draw(imgNew, rcX.Sub(rc.Min), imgOut, pt, draw.Src)
      }
    }

    imgOut = imgNew
  }

  return imgOut
}

// Used internally. Remove regions of transparent pixels around the frame (for paletted: given transIndex, for RGBA: transparent color).
func canvasTrim(img image.Image, transIndex byte) (imgOut image.Image, left, top, right, bottom int) {
  imgOut = img

  x0, x1 := img.Bounds().Min.X, img.Bounds().Max.X
  y0, y1 := img.Bounds().Min.Y, img.Bounds().Max.Y
  var lineStride, pixelStride int
  if imgPal, ok := img.(*image.Paletted); ok {
    lineStride = imgPal.Stride
    pixelStride = 1;
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    lineStride = imgRGBA.Stride
    pixelStride = 4;
    transIndex = 0
  }
  left = getTrimSize(img, x0, x1, pixelStride, y0, y1, lineStride, transIndex)
  x0 += left
  top = getTrimSize(img, y0, y1, lineStride, x0, x1, pixelStride, transIndex)
  y0 += top
  right = getTrimSize(img, x0, x1, -pixelStride, y0, y1, lineStride, transIndex)
  x1 -= right
  bottom = getTrimSize(img, y0, y1, -lineStride, x0, x1, pixelStride, transIndex)
  y1 -= bottom

  imgOut = canvasAddBorder(img, -left, -top, -right, -bottom, transIndex)
  return
}


// Used internally. A generalized implementation to find region to trim for either side of the image. Both paletted and RGBA images are supported.
// outer0/1 specifies outer loop to iterate. OuterStride is the distance to the start offset for the next pass. InnerXXX is the same for the inner loop.
// Use negative values for xxxStride to loop backwards. outer0/inner0 must always be less than outer1/inner1.
func getTrimSize(img image.Image, outer0, outer1, outerStride, inner0, inner1, innerStride int, transValue byte) int {
  // fmt.Printf("DEBUG: getTrimSize(img, %d, %d, %d, %d, %d, %d, %d)\n", outer0, outer1, outerStride, inner0, inner1, innerStride, transValue)
  trim := 0   // width of canvas to trim
  if outer0 >= outer1 || inner0 >= inner1 { return trim }

  // initializations
  var pix []byte
  var ofsA int
  if imgPal, ok := img.(*image.Paletted); ok {
    pix = imgPal.Pix
    ofsA = 0
  } else if imgRGBA, ok := img.(*image.RGBA); ok {
    pix = imgRGBA.Pix
    ofsA = 3
  }

  // scanning image
  if pix != nil {
    dirty := false
    for outer := outer0; outer < outer1; outer++ {
      var ofs int
      if outerStride > 0 {
        ofs = outer*outerStride
      } else {
        ofs = (outer1 + outer0 - outer - 1)*-outerStride
      }
      if innerStride > 0 {
        ofs += inner0*innerStride
      } else {
        ofs += (inner1 + inner0 - 1)*-innerStride
      }
      for inner := inner0; inner < inner1; inner++ {
        if pix[ofs+ofsA] != transValue { dirty = true; break }
        ofs += innerStride
      }
      if dirty { break }
      trim++
    }
  }

  return trim
}
