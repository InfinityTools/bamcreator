package bam

import (
  "image"
  "image/color"
)

type FilterResizeNearest struct {
  parent    *FilterResize
  frame     *BamFrame
  dw, dh    int
  center    bool
}

func (f *FilterResize) newResizeNearest(frame *BamFrame, dw, dh int, updateCenter bool) (*FilterResizeNearest, error) {
  var err error = nil
  frn := FilterResizeNearest{parent: f, frame: frame, dw: dw, dh: dh, center: updateCenter}
  return &frn, err
}

func (fr *FilterResizeNearest) GetSize() (int, int) { return fr.dw, fr.dh }

func (fr *FilterResizeNearest) SetSize(dw, dh int) error {
  fr.dw = dw
  fr.dh = dh
  return nil
}

func (fr *FilterResizeNearest) QuerySize(dw, dh int) (int, int) { return dw, dh }

func (fr *FilterResizeNearest) GetFrame() *BamFrame { return fr.frame }

// Resizes frame image to dw/dh with nearest neighbor algorithm. Updates center position if needed.
func (fr *FilterResizeNearest) Resize() error {
  sw := fr.frame.img.Bounds().Dx()
  sh := fr.frame.img.Bounds().Dy()

  var imgOut image.Image = nil
  var err error = nil
  if imgPal, ok := fr.frame.img.(*image.Paletted); ok {
    imgOut, err = fr.applyNearestPaletted(imgPal)
  } else if imgRGBA, ok := fr.frame.img.(*image.RGBA); ok {
    imgOut, err = fr.applyNearestRGBA(imgRGBA)
  }
  if err != nil { return err }
  fr.frame.img = imgOut

  if fr.center {
    dx := (fr.dw - sw) / 2
    dy := (fr.dh - sh) / 2
    fr.frame.cx += dx
    fr.frame.cy += dy
  }

  return nil
}


func (fr *FilterResizeNearest) applyNearestPaletted(imgIn *image.Paletted) (imgOut *image.Paletted, err error) {
  pal := make(color.Palette, len(imgIn.Palette))
  copy(pal, imgIn.Palette)
  imgOut = image.NewPaletted(image.Rect(0, 0, fr.dw, fr.dh), pal)

  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  srw, drw := fr.getRatio(sw, fr.dw)
  srh, drh := fr.getRatio(sh, fr.dh)

  // resize
  sofs := y0 * imgIn.Stride + x0
  dofs := 0
  for stepY, countY := 0, 0; countY < fr.dh; {
    // resizing y
    if stepY < drh {
      sx, dx := 0, 0
      for stepX, countX := 0, 0; countX < fr.dw; {
        // resizing x
        if stepX < drw {
          imgOut.Pix[dofs+dx] = imgIn.Pix[sofs+sx]
          dx++
          countX++
          stepX += srw
        } else {
          sx++
          stepX -= drw
        }
      }
      dofs += imgOut.Stride
      countY++
      stepY += srh
    } else {
      sofs += imgIn.Stride
      stepY -= drh
    }
  }

  return
}

func (fr *FilterResizeNearest) applyNearestRGBA(imgIn *image.RGBA) (imgOut *image.RGBA, err error) {
  imgOut = image.NewRGBA(image.Rect(0, 0, fr.dw, fr.dh))

  x0, x1 := imgIn.Bounds().Min.X, imgIn.Bounds().Max.X
  y0, y1 := imgIn.Bounds().Min.Y, imgIn.Bounds().Max.Y
  sw := x1 - x0
  sh := y1 - y0
  srw, drw := fr.getRatio(sw, fr.dw)
  srh, drh := fr.getRatio(sh, fr.dh)

  // resize
  sofs := y0 * imgIn.Stride + x0*4
  dofs := 0
  for stepY, countY := 0, 0; countY < fr.dh; {
    // resizing y
    if stepY < drh {
      sx, dx := 0, 0
      for stepX, countX := 0, 0; countX < fr.dw; {
        // resizing x
        if stepX < drw {
          imgOut.Pix[dofs+dx] = imgIn.Pix[sofs+sx]
          imgOut.Pix[dofs+dx+1] = imgIn.Pix[sofs+sx+1]
          imgOut.Pix[dofs+dx+2] = imgIn.Pix[sofs+sx+2]
          imgOut.Pix[dofs+dx+3] = imgIn.Pix[sofs+sx+3]
          dx += 4
          countX++
          stepX += srw
        } else {
          sx += 4
          stepX -= drw
        }
      }
      dofs += imgOut.Stride
      countY++
      stepY += srh
    } else {
      sofs += imgIn.Stride
      stepY -= drh
    }
  }

  return
}

// Returns optimized source/dest ratio values
func (fr *FilterResizeNearest) getRatio(s, d int) (sr, dr int) {
  sr, dr = s, d
  for (sr & 1) == 0 && (dr & 1) == 0 {
    sr >>= 1
    dr >>= 1
  }
  return
}
