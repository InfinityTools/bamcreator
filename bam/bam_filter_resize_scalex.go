package bam

import (
  "fmt"
  "image"
  "image/color"
)

type FilterResizeScaleX struct {
  parent      *FilterResize
  frame       *BamFrame
  dw, dh      int
  center      bool
  transIndex  byte
  factors     []int
}

func (f *FilterResize) newResizeScaleX(frame *BamFrame, dw, dh int, transIndex int, updateCenter bool) (frn *FilterResizeScaleX, err error) {
  frn = &FilterResizeScaleX{parent: f, frame: frame, center: updateCenter, transIndex: byte(transIndex), factors: make([]int, 0)}
  err = frn.SetSize(dw, dh)
  return frn, err
}

func (fr *FilterResizeScaleX) GetSize() (int, int) { return fr.dw, fr.dh }

func (fr *FilterResizeScaleX) SetSize(dw, dh int) error {
  // determining scale factors
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()
  fw := dw / sw
  fh := dh / sh
  if fw != fh { return fmt.Errorf("Scale factors are not equal (width=%d <> height=%d)", fw, fh) }

  fr.dw = fw*sw
  fr.dh = fh*sh
  if fw == 1 {
    fr.factors = make([]int, 0)
    return nil
  }

  var ok bool
  fr.factors, ok = fr.parent.factorize(fw, []int{3, 2})
  if !ok { return fmt.Errorf("Scale factor not a multiple of 2 or 3: %d", fw) }

  return nil
}

func (fr *FilterResizeScaleX) QuerySize(dw, dh int) (int, int) {
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()
  if sw <= 0 || sh <= 0 { return 0, 0 }
  fw := dw / sw
  fh := dh / sh
  if fw != fh { return 0, 0 }
  dw = fw*sw
  dh = fh*sh
  if fw == 1 || fw == 2 || fw == 3 || fw == 4 { return dw, dh }
  _, ok := fr.parent.factorize(fw, []int{3, 2})
  if !ok { return 0, 0 }

  return dw, dh
}

func (fr *FilterResizeScaleX) GetFrame() *BamFrame { return fr.frame }

// Resizes frame image to dw/dh with ScaleX algorithm. Updates center position if needed.
func (fr *FilterResizeScaleX) Resize() error {
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()
  curFactor := fr.dw / sw
  var err error = nil
  for _, factor := range fr.factors {
    switch factor {
      case 3:
        err = fr.applyScale3X()
      case 2:
        err = fr.applyScale2X()
      default:
        err = fmt.Errorf("Unexpected: scale factor = %d", factor)
    }
    if err != nil { return err }
    curFactor /= factor
  }

  if fr.center {
    dw := fr.frame.img.Bounds().Dx()
    dh := fr.frame.img.Bounds().Dy()
    fr.frame.cx += (dw - sw) / 2
    fr.frame.cy += (dh - sh) / 2
  }

  return nil
}


func (fr *FilterResizeScaleX) applyScale2X() error {
  var err error = nil
  if imgPal, ok := fr.frame.img.(*image.Paletted); ok {
    err = fr.applyScale2XPaletted(imgPal)
  } else if imgRGBA, ok := fr.frame.img.(*image.RGBA); ok {
    err = fr.applyScale2XRGBA(imgRGBA)
  } else {
    err = fmt.Errorf("Unsupported image format")
  }
  return err
}

func (fr *FilterResizeScaleX) applyScale2XPaletted(imgIn *image.Paletted) error {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw := sw * 2
  dh := sh * 2

  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut := image.NewPaletted(image.Rect(0, 0, dw, dh), pal)

  sgap := imgIn.Stride - sw
  dgap := imgOut.Stride - dw
  sofs := y0 * imgIn.Stride + x0
  dofs := 0
  for y := y0; y < y1; y, sofs, dofs = y+1, sofs+sgap, dofs+imgOut.Stride+dgap {
    for x := x0; x < x1; x, sofs, dofs = x+1, sofs+1, dofs+2 {
      p := imgIn.Pix[sofs]
      a, b, c, d := fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex
      if y > y0 { a = imgIn.Pix[sofs-imgIn.Stride] }
      if x+1 < x1 { b = imgIn.Pix[sofs+1] }
      if x > x0 { c = imgIn.Pix[sofs-1] }
      if y+1 < y1 { d = imgIn.Pix[sofs+imgIn.Stride] }
      t1, t2, t3, t4 := p, p, p, p
      if c == a && c != d && a != b { t1 = a }
      if a == b && a != c && b != d { t2 = b }
      if b == d && b != a && d != c { t4 = d }
      if d == c && d != b && c != a { t3 = c }
      imgOut.Pix[dofs] = t1
      imgOut.Pix[dofs+1] = t2
      imgOut.Pix[dofs+imgOut.Stride] = t3
      imgOut.Pix[dofs+imgOut.Stride+1] = t4
    }
  }

  fr.frame.img = imgOut
  return nil
}

func (fr *FilterResizeScaleX) applyScale2XRGBA(imgIn *image.RGBA) error {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw := sw * 2
  dh := sh * 2

  var bgColor uint32 = 0
  imgOut := image.NewRGBA(image.Rect(0, 0, dw, dh))

  sgap := imgIn.Stride - sw*4
  dgap := imgOut.Stride - dw*4
  sofs := y0 * imgIn.Stride + x0*4
  dofs := 0
  for y := y0; y < y1; y, sofs, dofs = y+1, sofs+sgap, dofs+imgOut.Stride+dgap {
    for x := x0; x < x1; x, sofs, dofs = x+1, sofs+4, dofs+8 {
      p := fr.parent.getRGBAInt(imgIn.Pix[sofs:])
      a, b, c, d := bgColor, bgColor, bgColor, bgColor
      if y > y0 { a = fr.parent.getRGBAInt(imgIn.Pix[sofs-imgIn.Stride:]) }
      if x+1 < x1 { b = fr.parent.getRGBAInt(imgIn.Pix[sofs+4:]) }
      if x > x0 { c = fr.parent.getRGBAInt(imgIn.Pix[sofs-4:]) }
      if y+1 < y1 { d = fr.parent.getRGBAInt(imgIn.Pix[sofs+imgIn.Stride:]) }
      t1, t2, t3, t4 := p, p, p, p
      if c == a && c != d && a != b { t1 = a }
      if a == b && a != c && b != d { t2 = b }
      if b == d && b != a && d != c { t4 = d }
      if d == c && d != b && c != a { t3 = c }
      fr.parent.putRGBAInt(imgOut.Pix[dofs:], t1)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+4:], t2)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride:], t3)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+4:], t4)
    }
  }

  fr.frame.img = imgOut
  return nil
}


func (fr *FilterResizeScaleX) applyScale3X() error {
  var err error = nil
  if imgPal, ok := fr.frame.img.(*image.Paletted); ok {
    err = fr.applyScale3XPaletted(imgPal)
  } else if imgRGBA, ok := fr.frame.img.(*image.RGBA); ok {
    err = fr.applyScale3XRGBA(imgRGBA)
  } else {
    err = fmt.Errorf("Unsupported image format")
  }
  return err
}

func (fr *FilterResizeScaleX) applyScale3XPaletted(imgIn *image.Paletted) error {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw := sw * 3
  dh := sh * 3

  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut := image.NewPaletted(image.Rect(0, 0, dw, dh), pal)

  sgap := imgIn.Stride - sw
  dgap := imgOut.Stride - dw
  sofs := y0 * imgIn.Stride + x0
  dofs := 0
  for y := y0; y < y1; y, sofs, dofs = y+1, sofs+sgap, dofs+imgOut.Stride+imgOut.Stride+dgap {
    for x := x0; x < x1; x, sofs, dofs = x+1, sofs+1, dofs+3 {
      e := imgIn.Pix[sofs]
      a, b, c, d, f, g, h, i := fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex, fr.transIndex
      if x > x0 && y > y0 { a = imgIn.Pix[sofs-imgIn.Stride-1] }
      if y > y0 { b = imgIn.Pix[sofs-imgIn.Stride] }
      if x+1 < x1 && y > y0 { c = imgIn.Pix[sofs-imgIn.Stride+1] }
      if x > x0 { d = imgIn.Pix[sofs-1] }
      if x+1 < x1 { f = imgIn.Pix[sofs+1] }
      if x > x0 && y+1 < y1 { g = imgIn.Pix[sofs+imgIn.Stride-1] }
      if y+1 < y1 { h = imgIn.Pix[sofs+imgIn.Stride] }
      if x+1 < x1 && y+1 < y1 { i = imgIn.Pix[sofs+imgIn.Stride+1] }
      t1, t2, t3, t4, t5, t6, t7, t8, t9 := e, e, e, e, e, e, e, e, e
      if d == b && d != h && b != f { t1 = d }
      if (d == b && d != h && b != f && e != c) || (b == f && b != d && f != h && e != a) { t2 = b }
      if b == f && b != d && f != h { t3 = f }
      if (h == d && h != f && d != b && e != a) || (d == b && d != h && b != f && e != g) { t4 = d }
      if (b == f && b != d && f != h && e != i) || (f == h && f != b && h != d && e != c) { t6 = f }
      if h == d && h != f && d != b { t7 = d }
      if (f == h && f != b && h != d && e != g) || (h == d && h != f && d != b && e != i) { t8 = h }
      if f == h && f != b && h != d { t9 = f }
      imgOut.Pix[dofs] = t1
      imgOut.Pix[dofs+1] = t2
      imgOut.Pix[dofs+2] = t3
      imgOut.Pix[dofs+imgOut.Stride] = t4
      imgOut.Pix[dofs+imgOut.Stride+1] = t5
      imgOut.Pix[dofs+imgOut.Stride+2] = t6
      imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride] = t7
      imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride+1] = t8
      imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride+2] = t9
    }
  }

  fr.frame.img = imgOut
  return nil
}

func (fr *FilterResizeScaleX) applyScale3XRGBA(imgIn *image.RGBA) error {
  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  dw := sw * 3
  dh := sh * 3

  var bgColor uint32 = 0
  imgOut := image.NewRGBA(image.Rect(0, 0, dw, dh))

  sgap := imgIn.Stride - sw*4
  dgap := imgOut.Stride - dw*4
  sofs := y0 * imgIn.Stride + x0*4
  dofs := 0
  for y := y0; y < y1; y, sofs, dofs = y+1, sofs+sgap, dofs+imgOut.Stride+imgOut.Stride+dgap {
    for x := x0; x < x1; x, sofs, dofs = x+1, sofs+4, dofs+12 {
      e := fr.parent.getRGBAInt(imgIn.Pix[sofs:])
      a, b, c, d, f, g, h, i := bgColor, bgColor, bgColor, bgColor, bgColor, bgColor, bgColor, bgColor
      if x > x0 && y > y0 { a = fr.parent.getRGBAInt(imgIn.Pix[sofs-imgIn.Stride-4:]) }
      if y > y0 { b = fr.parent.getRGBAInt(imgIn.Pix[sofs-imgIn.Stride:]) }
      if x+1 < x1 && y > y0 { c = fr.parent.getRGBAInt(imgIn.Pix[sofs-imgIn.Stride+4:]) }
      if x > x0 { d = fr.parent.getRGBAInt(imgIn.Pix[sofs-4:]) }
      if x+1 < x1 { f = fr.parent.getRGBAInt(imgIn.Pix[sofs+4:]) }
      if x > x0 && y+1 < y1 { g = fr.parent.getRGBAInt(imgIn.Pix[sofs+imgIn.Stride-4:]) }
      if y+1 < y1 { h = fr.parent.getRGBAInt(imgIn.Pix[sofs+imgIn.Stride:]) }
      if x+1 < x1 && y+1 < y1 { i = fr.parent.getRGBAInt(imgIn.Pix[sofs+imgIn.Stride+4:]) }
      t1, t2, t3, t4, t5, t6, t7, t8, t9 := e, e, e, e, e, e, e, e, e
      if d == b && d != h && b != f { t1 = d }
      if (d == b && d != h && b != f && e != c) || (b == f && b != d && f != h && e != a) { t2 = b }
      if b == f && b != d && f != h { t3 = f }
      if (h == d && h != f && d != b && e != a) || (d == b && d != h && b != f && e != g) { t4 = d }
      if (b == f && b != d && f != h && e != i) || (f == h && f != b && h != d && e != c) { t6 = f }
      if h == d && h != f && d != b { t7 = d }
      if (f == h && f != b && h != d && e != g) || (h == d && h != f && d != b && e != i) { t8 = h }
      if f == h && f != b && h != d { t9 = f }
      fr.parent.putRGBAInt(imgOut.Pix[dofs:], t1)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+4:], t2)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+8:], t3)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride:], t4)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+4:], t5)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+8:], t6)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride:], t7)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride+4:], t8)
      fr.parent.putRGBAInt(imgOut.Pix[dofs+imgOut.Stride+imgOut.Stride+8:], t9)
    }
  }

  fr.frame.img = imgOut
  return nil
}

