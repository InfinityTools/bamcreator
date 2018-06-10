package bam

import (
  "image"
  "image/color"
)

type FilterResizeBicubic struct {
  parent      *FilterResize
  frame     *BamFrame
  dw, dh    int
  center    bool
}

func (f *FilterResize) newResizeBicubic(frame *BamFrame, dw, dh int, updateCenter bool) (*FilterResizeBicubic, error) {
  var err error = nil
  frn := FilterResizeBicubic{parent: f, frame: frame, dw: dw, dh: dh, center: updateCenter}
  return &frn, err
}

func (fr *FilterResizeBicubic) GetSize() (int, int) { return fr.dw, fr.dh }

func (fr *FilterResizeBicubic) SetSize(dw, dh int) error {
  fr.dw = dw
  fr.dh = dh
  return nil
}

func (fr *FilterResizeBicubic) QuerySize(dw, dh int) (int, int) { return dw, dh }

func (fr *FilterResizeBicubic) GetFrame() *BamFrame { return fr.frame }

// Resizes frame image to dw/dh with bicubic algorithm. Updates center position if needed.
func (fr *FilterResizeBicubic) Resize() error {
  // doesn't support paletted image
  imgOut := image.NewRGBA(image.Rect(0, 0, fr.dw, fr.dh))
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()
  rmat := make([][]float32, 4)
  gmat := make([][]float32, 4)
  bmat := make([][]float32, 4)
  amat := make([][]float32, 4)
  for i := 0; i < 4; i++ {
    rmat[i] = make([]float32, 4)
    gmat[i] = make([]float32, 4)
    bmat[i] = make([]float32, 4)
    amat[i] = make([]float32, 4)
  }
  frx := float32(fr.dw) / float32(sw - 1)
  fry := float32(fr.dh) / float32(sh - 1)
  for x, y := 0, 0; y < fr.dh; x++ {
    if x > fr.dw { x = 0; y++ }
    fx := float32(x) / frx
    fy := float32(y) / fry
    fxi := int(fx)
    fyi := int(fy)
    for i := 0; i < 16; i++ {
      ix, iy := i & 3, i >> 2
      r, g, b, a := fr.frame.img.At(fxi+ix-1, fyi+iy-1).RGBA()
      rmat[ix][iy] = float32(r & 0xff)
      gmat[ix][iy] = float32(g & 0xff)
      bmat[ix][iy] = float32(b & 0xff)
      amat[ix][iy] = float32(a & 0xff)
    }
    a := fr.bicubicInterpolate(amat, fx - float32(fxi), fy - float32(fyi))
    if a < 0 { a =  0 }
    if a > 255 { a = 255 }
    r := fr.bicubicInterpolate(rmat, fx - float32(fxi), fy - float32(fyi))
    if r < 0 { r = 0 }
    if r > a { r = a }
    g := fr.bicubicInterpolate(gmat, fx - float32(fxi), fy - float32(fyi))
    if g < 0 { g = 0 }
    if g > a { g = a }
    b := fr.bicubicInterpolate(bmat, fx - float32(fxi), fy - float32(fyi))
    if b < 0 { b = 0 }
    if b > a { b = a }
    imgOut.Set(x, y, color.RGBA{byte(r), byte(g), byte(b), byte(a)})
  }
  fr.frame.img = imgOut

  if fr.center {
    dx := (fr.dw - sw) / 2
    dy := (fr.dh - sh) / 2
    fr.frame.cx += dx
    fr.frame.cy += dy
  }

  return nil
}


func (fr *FilterResizeBicubic) cubicInterpolate(c []float32, x float32) float32 {
  return c[1] + 0.5*x*(c[2] - c[0] + x*(2.0*c[0] - 5.0*c[1] + 4.0*c[2] - c[3] + x*(3.0*(c[1] - c[2]) + c[3] - c[0])))
}

func (fr *FilterResizeBicubic) bicubicInterpolate(c [][]float32, x, y float32) float32 {
  arr := make([]float32, 4)
  arr[0] = fr.cubicInterpolate(c[0], y)
  arr[1] = fr.cubicInterpolate(c[1], y)
  arr[2] = fr.cubicInterpolate(c[2], y)
  arr[3] = fr.cubicInterpolate(c[3], y)
  return fr.cubicInterpolate(arr, x)
}
