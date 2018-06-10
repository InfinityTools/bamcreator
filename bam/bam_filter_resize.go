package bam
/*
Implements filter "resize":
(see https://en.wikipedia.org/wiki/Pixel-art_scaling_algorithms )
Features: Implemented algorithm for reducing center point jittering after resize (based on bamresize by by Avenger_teambg and Sam.)
Options:
- type: Keyword from a list: (Default: nearest)
  - General-purpose scaling algorithms: Can be freely resized
    - nearest: nearest neighbor
    - bilinear: bilinear
    - bicubic: bicubic
  - Pixel-art scaling algorithms: Don't support fractional scaling factors. They are most effective on images with clear structures.
    - scalex: Scaling factors for width/height must be identical and be multiple of 2 or 3. Supports both paletted and rgba sources.
- scalewidth:  float (1.0)
- scaleheight: float (1.0)
- background: rgba value (0/transparent) - used by some resize methods
- updatecenter: bool (true)
*/

import (
  "fmt"
  "image"
  "image/color"
  "strings"
)

const (
  filterNameResize = "resize"
)

// Available resize filter types.
const (
  FILTER_NEAREST    = "nearest"
  FILTER_BILINEAR   = "bilinear"
  FILTER_BICUBIC    = "bicubic"
  FILTER_SCALEX     = "scalex"
)

type FilterResize struct {
  options     optionsMap
  opt_type, opt_scalewidth, opt_scaleheight, opt_background, opt_updatecenter string
}

// Handles resize operations
type FilterResizeKernel interface {
  // Applies resize operation to frame
  Resize() error
  // Returns resulting frame dimension. May differ from requested frame dimension for selected scaler types.
  GetSize() (int, int)
  // Sets new target width and height. Values may be corrected to conform to resize type restrictions.
  SetSize(int, int) error
  // Returns corrected frame dimensions of dw and dh. Returns 0 on error.
  QuerySize(int, int) (int, int)
  // Returns current state of BamFrame
  GetFrame() *BamFrame
}


// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameResize, NewFilterResize)
}


// Creates a new Resize filter.
func NewFilterResize() BamFilter {
  f := FilterResize{options: make(optionsMap),
                    opt_type: "type",
                    opt_scalewidth: "scalewidth",
                    opt_scaleheight: "scaleheight",
                    opt_background: "background",
                    opt_updatecenter: "updatecenter"}
  f.SetOption(f.opt_type, "nearest")
  f.SetOption(f.opt_scalewidth, "1.0")
  f.SetOption(f.opt_scaleheight, "1.0")
  f.SetOption(f.opt_background, "0")
  f.SetOption(f.opt_updatecenter, "true")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterResize) GetName() string {
  return filterNameResize
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterResize) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterResize) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_updatecenter:
      v, err := parseBool(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_scalewidth, f.opt_scaleheight:
      v, err := parseFloatRange(value, 0.00001, 256.0)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_background:
      v, err := parseInt(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_type:
      value = strings.ToLower(strings.TrimSpace(value))
      switch value {
        case FILTER_NEAREST, FILTER_BILINEAR, FILTER_BICUBIC, FILTER_SCALEX:
          f.options[key] = value
        default:
          return fmt.Errorf("Option %s: unsupported: %q", key, value)
      }
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterResize) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut, inFrames)
  return frameOut, err
}


// Used internally. Applies resize effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterResize) apply(frame *BamFrame, inFrames []BamFrame) error {
  scaler := f.GetOption(f.opt_type).(string)
  scalex := f.GetOption(f.opt_scalewidth).(float64)
  scaley := f.GetOption(f.opt_scaleheight).(float64)
  bgcolor := uint32(f.GetOption(f.opt_background).(int))
  center := f.GetOption(f.opt_updatecenter).(bool)
  if scalex == 1.0 && scaley == 1.0 { return nil }

  // Getting palette index of background color
  transIndex := 0
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    transIndex = f.getColorIndexOf(imgPal.Palette, bgcolor, transIndex)
  }

  // Initial target size calculation
  sw := frame.img.Bounds().Dx()
  sh := frame.img.Bounds().Dy()
  dw := int(float64(sw) * scalex)
  if dw < 1 { dw = 1 }
  if dw > 65535 { return fmt.Errorf("Scaled width too big: %d", dw) }
  dh := int(float64(sh) * scaley)
  if dh < 1 { dh = 1 }
  if dw > 65535 { return fmt.Errorf("Scaled height too big: %d", dh) }

  var kernel FilterResizeKernel = nil
  var err error = nil
  switch scaler {
    case FILTER_BILINEAR:
      kernel, err = f.newResizeBilinear(frame, dw, dh, center)
    case FILTER_BICUBIC:
      kernel, err = f.newResizeBicubic(frame, dw, dh, center)
    case FILTER_SCALEX:
      kernel, err = f.newResizeScaleX(frame, dw, dh, transIndex, center)
    default:
      kernel, err = f.newResizeNearest(frame, dw, dh, center)
  }
  if err != nil { return err }


  // To reduce jittering center positions, canvas is increased to be fixed for all available BAM frames
  origWidth, origHeight := kernel.GetSize()
  left, top, right, bottom := getGlobalCanvas(frame.img.Bounds().Dx(), frame.img.Bounds().Dy(), frame.cx, frame.cy, inFrames)
  frame.img = canvasAddBorder(frame.img, left, top, right, bottom, byte(transIndex))
  frame.cx += left
  frame.cy += top

  // Repeated target size calculation
  sw = frame.img.Bounds().Dx()
  sh = frame.img.Bounds().Dy()
  dw = int(float64(sw) * scalex)
  dh = int(float64(sh) * scaley)
  err = kernel.SetSize(dw, dh)
  if err != nil { return err }

  err = kernel.Resize()
  if err != nil { return err }

  // Jitter reduction, part 2: trimming excess canvas
  dw = origWidth - frame.img.Bounds().Dx()
  left = dw / 2
  right = dw - left
  dh = origHeight - frame.img.Bounds().Dy()
  top = dh / 2
  bottom = dh - top
  if left < 0 { left = 0 }
  if right < 0 { right = 0 }
  if top < 0 { top = 0 }
  if bottom < 0 { bottom = 0 }
  canvasAddBorder(frame.img, -left, -top, -right, -bottom, byte(transIndex))
  frame.cx -= left
  frame.cy -= top

  return err
}


// Returns a sequence of r,g,b,a items as composite ARGB value.
func (f *FilterResize) getRGBAInt(imgBuf []byte) uint32 {
  ret := uint32(imgBuf[3])
  ret <<= 8
  ret |= uint32(imgBuf[0])
  ret <<= 8
  ret |= uint32(imgBuf[1])
  ret <<= 8
  ret |= uint32(imgBuf[2])
  return ret
}

// Splits the given ARGB value into r,g,b,a items and writes them into the specified slice.
func (f *FilterResize) putRGBAInt(imgBuf []byte, argb uint32) {
  imgBuf[0] = byte(argb >> 16)
  imgBuf[1] = byte(argb >> 8)
  imgBuf[2] = byte(argb)
  imgBuf[3] = byte(argb >> 24)
}


// Returns the palette index of the entry that matches "color". Returns defColor if not found.
func (f *FilterResize) getColorIndexOf(pal color.Palette, color uint32, defIndex int) int {
  retVal := defIndex

  // reference color -> premultiplied color components
  var sr, sg, sb, sa uint32
  sa = (color >> 24) & 0xff
  if sa > 0 {
    sr = ((color >> 16) & 0xff) * sa / 255
    sg = ((color >> 8) & 0xff) * sa / 255
    sb = (color & 0xff) * sa / 255
  } else {
    sr, sg, sb = 0, 0, 0
  }

  for idx, col := range pal {
    r, g, b, a := col.RGBA()
    if a == 0 && a == sa {
      retVal = idx
      break
    }
    r, g, b, a = r & 0xff, g & 0xff, b & 0xff, a & 0xff
    if r == sr && g == sg && b == sb && a == sa {
      retVal = idx
      break
    }
  }

  return retVal
}


// Attempts to factorize the given value only with the factors listed in the specified sequence.
// Returns whether factorization was successful. Resulting factors are sorted by appearance in the "factors" argument.
func (f *FilterResize) factorize(value int, factors []int) (ret []int, ok bool) {
  ret = nil
  ok = false
  if value < 1 || factors == nil || len(factors) == 0 { return }

  ret = factorize0(value, factors, make([]int, 0, 10))

  if len(ret) > 0 && ret[len(ret)-1] == 0 {
    ok = true
    ret = ret[0:len(ret)-1]
  }
  return
}

// Used internally. Returns a zero-terminated array of factors.
func factorize0(value int, factors []int, result []int) []int {
  for _, f := range factors {
    if value == 1 { result = append(result, 0); break }
    if value % f == 0 {
      result = append(result, f)
      result = factorize0(value / f, factors, result)
      if len(result) > 0 && result[len(result)-1] == 0 { break }
    }
  }
  return result
}
