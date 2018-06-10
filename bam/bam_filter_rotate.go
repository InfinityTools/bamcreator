package bam
/*
Implements filter "rotate":
Options:
- angle: float (0.0) - rotation amount in clockwise direction, using degrees in range [0, 360)
- interpolate: bool (false) - whether to interpolate resulting pixels (ignored when rotating by multiples of 90 deg)
- background: rgba value (0/transparent) - the color value for pixels that lie outside source image bounds after rotation
- updatecenter: bool (true)
*/

import (
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "math"
  "strings"
)

const (
  filterNameRotate = "rotate"
)

type FilterRotate struct {
  options     optionsMap
  opt_angle, opt_interpolate, opt_background, opt_updatecenter string
}

// Register filter for use in bam creator.
func init() {
  registerFilter(filterNameRotate, NewFilterRotate)
}


// Creates a new Rotate filter.
func NewFilterRotate() BamFilter {
  f := FilterRotate{options: make(optionsMap),
                    opt_angle: "angle",
                    opt_interpolate: "interpolate",
                    opt_background: "background",
                    opt_updatecenter: "updatecenter"}
  f.SetOption(f.opt_angle, "0")
  f.SetOption(f.opt_interpolate, "false")
  f.SetOption(f.opt_background, "0")
  f.SetOption(f.opt_updatecenter, "true")
  return &f
}

// GetName returns the name of the filter for identification purposes.
func (f *FilterRotate) GetName() string {
  return filterNameRotate
}

// GetOption returns the option of given name. Content of return value is filter specific.
func (f *FilterRotate) GetOption(key string) interface{} {
  v, ok := f.options[strings.ToLower(key)]
  if !ok { return nil }
  return v
}

// SetOption adds or updates an option of the given key to the filter.
func (f *FilterRotate) SetOption(key, value string) error {
  key = strings.ToLower(key)
  switch key {
    case f.opt_interpolate, f.opt_updatecenter:
      v, err := parseBool(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_angle:
      v, err := parseFloat(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
    case f.opt_background:
      v, err := parseInt(value)
      if err != nil { return fmt.Errorf("Option %s: %v", key, err) }
      f.options[key] = v
  }
  return nil
}

// Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
func (f *FilterRotate) Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error) {
  frameOut := BamFrame{cx: frame.cx, cy: frame.cy, img: nil}
  imgOut := cloneImage(frame.img, false)
  frameOut.img = imgOut
  err := f.apply(&frameOut, inFrames)
  return frameOut, err
}


// Used internally. Applies rotate effect. Assumes source image is of type image.RGBA or image.Paletted
func (f *FilterRotate) apply(frame *BamFrame, inFrames []BamFrame) error {
  angle := math.Mod(f.GetOption(f.opt_angle).(float64), 360.0)
  if angle == 0.0 { return nil }
  if angle < 0.0 { angle += 360.0 }
  ortho := 0    // indicates a rotation by multiple of 90 deg
  switch (angle) {
    case 90.0:  ortho = 1
    case 180.0: ortho = 2
    case 270.0: ortho = 3
  }
  angle = angle * math.Pi / 180.0
  interpolate := f.GetOption(f.opt_interpolate).(bool)
  bgcolor := uint32(f.GetOption(f.opt_background).(int))
  center := f.GetOption(f.opt_updatecenter).(bool)

  // Check availability of bgcolor
  bgIndex := -1
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    if !interpolate {
      // reference color -> premultiplied color components
      sr, sg, sb, sa := (bgcolor >> 16) & 0xff, (bgcolor >> 8) & 0xff, (bgcolor) & 0xff, (bgcolor >> 24) & 0xff
      if sa > 0 {
        sr = sr * sa / 255
        sg = sg * sa / 255
        sb = sb * sa / 255
      } else {
        sr, sg, sb = 0, 0, 0
      }
      for idx, col := range imgPal.Palette {
        r, g, b, a := col.RGBA()
        if a == 0 && a == sa {
          bgIndex = idx
          break
        }
        r, g, b, a = r & 0xff, g & 0xff, b & 0xff, a & 0xff
        if r == sr && g == sg && b == sb && a == sa {
          bgIndex = idx
          break
        }
      }
    }
    if interpolate || bgIndex < 0 {
      // convert to RGBA image
      img := image.NewRGBA(imgPal.Bounds())
      draw.Draw(img, imgPal.Bounds(), imgPal, imgPal.Bounds().Min, draw.Src)
      frame.img = img
    }
  }

  var err error = nil
  switch ortho {
    case 1:   err = f.rotate90(frame, center)
    case 2:   err = f.rotate180(frame, center)
    case 3:   err = f.rotate270(frame, center)
    default:  err = f.rotate(frame, angle, byte(bgIndex), interpolate, center, inFrames)
  }
  return err
}


func (f *FilterRotate) rotate90(frame *BamFrame, updateCenter bool) error {
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    imgOut, err := f.rotatePal90(imgPal, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    imgOut, err := f.rotateRGBA90(imgRGBA, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  }

  if updateCenter {
    frame.cx, frame.cy = frame.cy, frame.img.Bounds().Dx() - 1 - frame.cx
  }

  return nil
}

func (f *FilterRotate) rotatePal90(imgIn *image.Paletted, updateCenter bool) (imgOut *image.Paletted, err error) {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw, dh := sh, sw
  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut = image.NewPaletted(image.Rect(0, 0, dw, dh), pal)

  sgap := imgIn.Stride - sw
  sofs := y0 * imgIn.Stride + x0
  for y := 0; y < sh; y, sofs = y+1, sofs+sgap {
    dofs := dw - y - 1
    for x := 0; x < sw; x, sofs, dofs = x+1, sofs+1, dofs+imgOut.Stride {
      imgOut.Pix[dofs] = imgIn.Pix[sofs]
    }
  }

  return
}

func (f *FilterRotate) rotateRGBA90(imgIn *image.RGBA, updateCenter bool) (imgOut *image.RGBA, err error) {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw, dh := sh, sw
  imgOut = image.NewRGBA(image.Rect(0, 0, dw, dh))

  sgap := imgIn.Stride - sw*4
  sofs := y0 * imgIn.Stride + x0*4
  for y := 0; y < sh; y, sofs = y+4, sofs+sgap {
    dofs := (dw - y - 1)*4
    for x := 0; x < sw; x, sofs, dofs = x+1, sofs+4, dofs+imgOut.Stride {
      imgOut.Pix[dofs] = imgIn.Pix[sofs]
      imgOut.Pix[dofs+1] = imgIn.Pix[sofs+1]
      imgOut.Pix[dofs+2] = imgIn.Pix[sofs+2]
      imgOut.Pix[dofs+3] = imgIn.Pix[sofs+3]
    }
  }

  return
}


func (f *FilterRotate) rotate180(frame *BamFrame, updateCenter bool) error {
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    imgOut, err := f.rotatePal180(imgPal, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    imgOut, err := f.rotateRGBA180(imgRGBA, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  }

  if updateCenter {
    frame.cx = frame.img.Bounds().Dx() - 1 - frame.cx
    frame.cy = frame.img.Bounds().Dy() - 1 - frame.cy
  }

  return nil
}

func (f *FilterRotate) rotatePal180(imgIn *image.Paletted, updateCenter bool) (imgOut *image.Paletted, err error) {
  // attempting in-place rotation
  imgOut = imgIn
  x0, x1 := imgOut.Bounds().Min.X, imgOut.Bounds().Max.X
  y0, y1 := imgOut.Bounds().Min.Y, imgOut.Bounds().Max.Y
  w := x1 - x0

  gap := imgOut.Stride - w
  sofs := y0 * imgOut.Stride + x0
  dofs := (y1 - 1) * imgOut.Stride + (x1-1)
  for ; sofs < dofs; sofs, dofs = sofs+gap, dofs-gap {
    for x := 0; x < w && sofs < dofs; x, sofs, dofs = x+1, sofs+1, dofs-1 {
      imgOut.Pix[sofs], imgOut.Pix[dofs] = imgOut.Pix[dofs], imgOut.Pix[sofs]
    }
  }

  return
}

func (f *FilterRotate) rotateRGBA180(imgIn *image.RGBA, updateCenter bool) (imgOut *image.RGBA, err error) {
  // attempting in-place rotation
  imgOut = imgIn
  x0, x1 := imgOut.Bounds().Min.X, imgOut.Bounds().Max.X
  y0, y1 := imgOut.Bounds().Min.Y, imgOut.Bounds().Max.Y
  w := x1 - x0

  gap := imgOut.Stride - w*4
  sofs := y0 * imgOut.Stride + x0*4
  dofs := (y1 - 1) * imgOut.Stride + (x1 - 1)*4
  for ; sofs < dofs; sofs, dofs = sofs+gap, dofs-gap {
    for x := 0; x < w && sofs < dofs; x, sofs, dofs = x+1, sofs+4, dofs-4 {
      imgOut.Pix[sofs], imgOut.Pix[dofs] = imgOut.Pix[dofs], imgOut.Pix[sofs]
      imgOut.Pix[sofs+1], imgOut.Pix[dofs+1] = imgOut.Pix[dofs+1], imgOut.Pix[sofs+1]
      imgOut.Pix[sofs+2], imgOut.Pix[dofs+2] = imgOut.Pix[dofs+2], imgOut.Pix[sofs+2]
      imgOut.Pix[sofs+3], imgOut.Pix[dofs+3] = imgOut.Pix[dofs+3], imgOut.Pix[sofs+3]
    }
  }

  return
}


func (f *FilterRotate) rotate270(frame *BamFrame, updateCenter bool) error {
  if imgPal, ok := frame.img.(*image.Paletted); ok {
    imgOut, err := f.rotatePal270(imgPal, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    imgOut, err := f.rotateRGBA270(imgRGBA, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  }

  if updateCenter {
    frame.cx, frame.cy = frame.cy, frame.img.Bounds().Dy() - 1 - frame.cx
  }

  return nil
}

func (f *FilterRotate) rotatePal270(imgIn *image.Paletted, updateCenter bool) (imgOut *image.Paletted, err error) {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw, dh := sh, sw
  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut = image.NewPaletted(image.Rect(0, 0, dw, dh), pal)

  sgap := imgIn.Stride - sw
  sofs := y0 * imgIn.Stride + x0
  for y := 0; y < sh; y, sofs = y+1, sofs+sgap {
    dofs := (dh - 1) * imgOut.Stride + y
    for x := 0; x < sw; x, sofs, dofs = x+1, sofs+1, dofs-imgOut.Stride {
      imgOut.Pix[dofs] = imgIn.Pix[sofs]
    }
  }

  return
}

func (f *FilterRotate) rotateRGBA270(imgIn *image.RGBA, updateCenter bool) (imgOut *image.RGBA, err error) {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw, dh := sh, sw
  imgOut = image.NewRGBA(image.Rect(0, 0, dw, dh))

  sgap := imgIn.Stride - sw*4
  sofs := y0 * imgIn.Stride + x0*4
  for y := 0; y < sh; y, sofs = y+1, sofs+sgap {
    dofs := (dh - 1) * imgOut.Stride + y*4
    for x := 0; x < sw; x, sofs, dofs = x+1, sofs+4, dofs-imgOut.Stride {
      imgOut.Pix[dofs] = imgIn.Pix[sofs]
      imgOut.Pix[dofs+1] = imgIn.Pix[sofs+1]
      imgOut.Pix[dofs+2] = imgIn.Pix[sofs+2]
      imgOut.Pix[dofs+3] = imgIn.Pix[sofs+3]
    }
  }

  return
}


func (f *FilterRotate) rotate(frame *BamFrame, angle float64, bgIndex byte, interpolate, updateCenter bool, inFrames []BamFrame) error {
  // To reduce jittering center positions, canvas is increased to be fixed for all available BAM frames
  var bgColor byte = 0  // use bgIndex for paletted, transparent color for RGBA
  if _, ok := frame.img.(*image.Paletted); ok { bgColor = bgIndex }
  left, top, right, bottom := getGlobalCanvas(frame.img.Bounds().Dx(), frame.img.Bounds().Dy(), frame.cx, frame.cy, inFrames)
  frame.img = canvasAddBorder(frame.img, left, top, right, bottom, bgColor)
  frame.cx += left
  frame.cy += top

  sw, sh := frame.img.Bounds().Dx(), frame.img.Bounds().Dy()

  if imgPal, ok := frame.img.(*image.Paletted); ok {
    imgOut, err := f.rotatePal(imgPal, angle, bgIndex, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  } else if imgRGBA, ok := frame.img.(*image.RGBA); ok {
    imgOut, err := f.rotateRGBA(imgRGBA, angle, interpolate, updateCenter)
    if err != nil { return err }
    frame.img = imgOut
  }

  if updateCenter {
    dw, dh := frame.img.Bounds().Dx(), frame.img.Bounds().Dy()
    dx, dy := (dw - sw) / 2, (dh - sh) / 2
    frame.cx, frame.cy = f.getRotatedPoint(frame.cx + dx, frame.cy + dy, dw, dh, angle)
  }

  // Jitter reduction, part 2: trimming excess canvas
  frame.img, left, top, _, _ = canvasTrim(frame.img, bgColor)
  frame.cx -= left
  frame.cy -= top

  return nil
}

func (f *FilterRotate) rotatePal(imgIn *image.Paletted, angle float64, bgIndex byte, updateCenter bool) (imgOut *image.Paletted, err error) {
  rectOut := f.getRotatedRect(imgIn.Bounds(), angle)
  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut = image.NewPaletted(rectOut, pal)

  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  scx := sw / 2
  scy := sh / 2
  swm2 := sw - 2;
  shm2 := sh - 2;
  dw := imgOut.Bounds().Dx()
  dh := imgOut.Bounds().Dy()
  dcx := dw / 2
  dcy := dh / 2
  sina := math.Sin(angle)
  cosa := math.Cos(angle)

  // applying rotation
  for y := 0; y < dh; y++ {
    ydiff := float64(dcy - y)
    dofs := y * imgOut.Stride
    for x := 0; x < dw; x, dofs = x+1, dofs+1 {
      xdiff := float64(dcx - x)
      xpm := int(-xdiff*cosa - ydiff*sina)
      ypm := int(-ydiff*cosa + xdiff*sina)
      xp, yp := scx + xpm, scy + ypm

      // set bgIndex for out of bounds pixels
      if xp < 0 || yp < 0 || xp > swm2 || yp > shm2 {
        imgOut.Pix[dofs] = bgIndex
        continue
      }

      // no interpolation necessary for paletted pixels
      sofs := (y0 + yp) * imgIn.Stride + (x0 + xp)
      imgOut.Pix[dofs] = imgIn.Pix[sofs]
    }
  }

  return
}

func (f *FilterRotate) rotateRGBA(imgIn *image.RGBA, angle float64, interpolate, updateCenter bool) (imgOut *image.RGBA, err error) {
  rectOut := f.getRotatedRect(imgIn.Bounds(), angle)
  imgOut = image.NewRGBA(rectOut)

  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  scx := sw / 2
  scy := sh / 2
  swm2 := sw - 2;
  shm2 := sh - 2;
  dw := imgOut.Bounds().Dx()
  dh := imgOut.Bounds().Dy()
  dcx := dw / 2
  dcy := dh / 2
  sina := 16.0 * math.Sin(angle)
  cosa := 16.0 * math.Cos(angle)

  // applying rotation
  for y := 0; y < dh; y++ {
    ydiff := float64(dcy - y)
    dofs := y * imgOut.Stride
    for x := 0; x < dw; x, dofs = x+1, dofs+4 {
      xdiff := float64(dcx - x)
      xpm := int(-xdiff*cosa - ydiff*sina)
      ypm := int(-ydiff*cosa + xdiff*sina)
      xp, yp := scx + (xpm >> 4), scy + (ypm >> 4)
      xf, yf := xpm & 0x0f, ypm & 0x0f

      // set bgIndex for out of bounds pixels
      if xp < 0 || yp < 0 || xp > swm2 || yp > shm2 {
        imgOut.Pix[dofs], imgOut.Pix[dofs+1], imgOut.Pix[dofs+2], imgOut.Pix[dofs+3] = 0, 0, 0, 0
        continue
      }

      var r, g, b, a int
      if interpolate {
        r00, g00, b00, a00 := imgIn.At(xp, yp).RGBA()
        r10, g10, b10, a10 := imgIn.At(xp + 1, yp).RGBA()
        r01, g01, b01, a01 := imgIn.At(xp, yp + 1).RGBA()
        r11, g11, b11, a11 := imgIn.At(xp + 1, yp + 1).RGBA()
        r = ((16 - xf)*(16 - yf)*int(r00) + xf*(16 - yf)*int(r10) + (16 - xf)*yf*int(r01) + xf*yf*int(r11)) >> 16
        g = ((16 - xf)*(16 - yf)*int(g00) + xf*(16 - yf)*int(g10) + (16 - xf)*yf*int(g01) + xf*yf*int(g11)) >> 16
        b = ((16 - xf)*(16 - yf)*int(b00) + xf*(16 - yf)*int(b10) + (16 - xf)*yf*int(b01) + xf*yf*int(b11)) >> 16
        a = ((16 - xf)*(16 - yf)*int(a00) + xf*(16 - yf)*int(a10) + (16 - xf)*yf*int(a01) + xf*yf*int(a11)) >> 16
      } else {
        r0, g0, b0, a0 := imgIn.At(xp, yp).RGBA()
        r, g, b, a = int(r0), int(g0), int(b0), int(a0)
      }
      imgOut.Pix[dofs] = byte(r)
      imgOut.Pix[dofs+1] = byte(g)
      imgOut.Pix[dofs+2] = byte(b)
      imgOut.Pix[dofs+3] = byte(a)
    }
  }

  return
}


// Calculate output image dimensions. Specify angle in Radian.
func (f *FilterRotate) getRotatedRect(rectIn image.Rectangle, angle float64) image.Rectangle {
  cx, cy := float64(rectIn.Dx() / 2), float64(rectIn.Dy() / 2)
  w, h := float64(rectIn.Dx()), float64(rectIn.Dy())
  bounds := [][]float64{{0., 0.}, {w-1., 0.}, {w-1., h-1.}, {0., h-1.}}
  for idx, p := range bounds {
    x, y := p[0], p[1]
    x2 := (x - cx)*math.Cos(angle) - (cy - y)*math.Sin(angle) + cx
    y2 := (cy - y)*math.Cos(angle) - (x - cx)*math.Sin(angle) - cy
    bounds[idx][0], bounds[idx][1] = x2, y2
  }
  minX, maxX := 99999.0, -99999.0
  minY, maxY := 99999.0, -99999.0
  for _, p := range bounds {
    if p[0] < minX { minX = p[0] }
    if p[0] > maxX { maxX = p[0] }
    if p[1] < minY { minY = p[1] }
    if p[1] > maxY { maxY = p[1] }
  }
  return image.Rect(0, 0, int(maxX - minX + 1), int(maxY - minY + 1))
}

// Calculate new position based on given arguments
func (f *FilterRotate) getRotatedPoint(x, y, width, height int, angle float64) (x2, y2 int) {
  fcx, fcy := float64(width / 2), float64(height / 2)
  fx := (float64(x) - fcx)*math.Cos(-angle) - (fcy - float64(y))*math.Sin(-angle)
  fy := (float64(y) - fcy)*math.Cos(-angle) + (fcx - float64(x))*math.Sin(-angle)

  x2 = int(fcx + fx)
  y2 = int(fcy + fy)
  return
}
