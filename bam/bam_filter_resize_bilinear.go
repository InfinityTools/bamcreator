package bam

import (
  "image"
  "image/color"
)

type FilterResizeBilinear struct {
  parent    *FilterResize
  frame     *BamFrame
  dw, dh    int
  center    bool
}

func (f *FilterResize) newResizeBilinear(frame *BamFrame, dw, dh int, updateCenter bool) (*FilterResizeBilinear, error) {
  var err error = nil
  frn := FilterResizeBilinear{parent: f, frame: frame, dw: dw, dh: dh, center: updateCenter}
  return &frn, err
}

func (fr *FilterResizeBilinear) GetSize() (int, int) { return fr.dw, fr.dh }

func (fr *FilterResizeBilinear) SetSize(dw, dh int) error {
  fr.dw = dw
  fr.dh = dh
  return nil
}

func (fr *FilterResizeBilinear) QuerySize(dw, dh int) (int, int) { return dw, dh }

func (fr *FilterResizeBilinear) GetFrame() *BamFrame { return fr.frame }

// Resizes frame image to dw/dh with bilinear algorithm. Updates center position if needed.
func (fr *FilterResizeBilinear) Resize() error {
  // doesn't support paletted image
  imgOut := image.NewRGBA(image.Rect(0, 0, fr.dw, fr.dh))
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()
  frx := float32(fr.dw) / float32(sw - 1)
  fry := float32(fr.dh) / float32(sh - 1)
  for x, y := 0, 0; y < fr.dh; x++ {
    if x > fr.dw { x = 0; y++ }
    fx := float32(x) / frx
    fy := float32(y) / fry
    fxi := int(fx)
    fyi := int(fy)
    r00, g00, b00, a00 := fr.frame.img.At(fxi, fyi).RGBA()
    r10, g10, b10, a10 := fr.frame.img.At(fxi + 1, fyi).RGBA()
    r01, g01, b01, a01 := fr.frame.img.At(fxi, fyi + 1).RGBA()
    r11, g11, b11, a11 := fr.frame.img.At(fxi + 1, fyi + 1).RGBA()
    r := uint32(fr.blerp(float32(r00), float32(r10), float32(r01), float32(r11), fx - float32(fxi), fy - float32(fyi))) >> 8
    g := uint32(fr.blerp(float32(g00), float32(g10), float32(g01), float32(g11), fx - float32(fxi), fy - float32(fyi))) >> 8
    b := uint32(fr.blerp(float32(b00), float32(b10), float32(b01), float32(b11), fx - float32(fxi), fy - float32(fyi))) >> 8
    a := uint32(fr.blerp(float32(a00), float32(a10), float32(a01), float32(a11), fx - float32(fxi), fy - float32(fyi))) >> 8
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


func (f *FilterResizeBilinear) lerp(s, e, t float32) float32 {
  return s + (e - s)*t
}

func (f *FilterResizeBilinear) blerp(c00, c10, c01, c11, tx, ty float32) float32 {
  return f.lerp(f.lerp(c00, c10, tx), f.lerp(c01, c11, tx), ty)
}
