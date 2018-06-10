/*
Package graphics provides functions for loading various single- or multi-image graphics resources
without having to take care of the details.
*/
package graphics

import (
  "bytes"
  "errors"
  "image"
  "image/draw"
  "image/gif"
  "image/jpeg"
  "image/png"
  "io"
  // "time"

  "github.com/InfinityTools/bamcreator/bam"
  "github.com/InfinityTools/go-logging"
  "golang.org/x/image/bmp"
)

// Can be used to identifiy the imported image format
const (
  TYPE_UNKNOWN = -1
  TYPE_BAM  = iota
  TYPE_BMP
  TYPE_GIF
  TYPE_JPG
  TYPE_PNG
)

type Frame struct {
  img     image.Image
  cx, cy  int
}

// The main graphics structure.
type Graphics struct {
  frames  []Frame   // one or more frames imported from the graphics resource
  format  int       // see TYPE_xxx constants
  err     error
}


// Import imports a graphics resources pointed to by the ReadSeeker interface.
//
// Optional list of search paths is used for input files that are asociated with auxiliary files
// (e.g. BAM V2 and PVRZ). Specify nil to ignore or fall back to current directory.
// Use function Error() to check if Import returned successfully.
func Import(rs io.ReadSeeker, searchPaths []string) *Graphics {
  g := Graphics{frames: make([]Frame, 0), format: TYPE_UNKNOWN, err: nil}
  if rs == nil { g.err = errors.New("No source specified"); return &g }

  if searchPaths == nil {
    searchPaths = make([]string, 0)
  }
  if len(searchPaths) == 0 {
    searchPaths = append(searchPaths, "")
  }

  (&g).importImage(rs, searchPaths)

  return &g
}


// Error returns the error state of the most recent operation on the Graphics. Use ClearError() function to clear the
// current error state.
func (g *Graphics) Error() error {
  return g.err
}


// ClearError clears the error state from the last Graphics operation. This function must be called for subsequent
// operations to work correctly.
//
// Use this function with care. Several functions may leave the Graphics object in an incomplete state after
// returning unsuccessfully.
func (g *Graphics) ClearError() {
  g.err = nil
}


// GetImageLength returns the number of available images.
func (g *Graphics) GetImageLength() int {
  if g.err != nil { return 0 }

  return len(g.frames)
}


// GetImageType returns the format of the imported image. See TYPE_xxx constants.
func (g *Graphics) GetImageType() int {
  if g.err != nil { return TYPE_UNKNOWN }
  return g.format
}


// GetImage returns the image at the specified index.
//
// For BMP, JPG and PNG only index=0 is valid. GIF and BAM may contain multiple images.
// The returned image is guaranteed to be of either Image.Paletted or Image.RGBA format.
func (g *Graphics) GetImage(index int) image.Image {
  if g.err != nil { return nil }
  if index < 0 || index > g.GetImageLength() { return nil }

  var imgOut image.Image = g.frames[index].img
  if img, ok := g.frames[index].img.(*image.Paletted); ok {
    imgOut = img
  } else if _, ok := g.frames[index].img.(*image.RGBA); !ok {
    rgba := image.NewRGBA(image.Rect(0, 0, g.frames[index].img.Bounds().Dx(), g.frames[index].img.Bounds().Dy()))
    draw.Draw(rgba, rgba.Bounds(), g.frames[index].img, image.ZP, draw.Src)
    imgOut = rgba
  }
  return imgOut
}


// GetCenter returns the center position of the specified frame. This is only meaningful when importing BAM frames.
// Other formats will always return {0, 0}.
func (g *Graphics) GetCenter(index int) (x, y int) {
  if g.err != nil { return }
  if index < 0 || index > g.GetImageLength() { return }

  x, y = g.frames[index].cx, g.frames[index].cy
  return
}


// Used internally. Delegates import to more specialized functions.
func (g *Graphics) importImage(rs io.ReadSeeker, searchPaths []string) {
  hdr := make([]byte, 4)
  _, err := rs.Read(hdr)
  if err != nil { g.err = err; return }
  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { g.err = err; return }

  if string(hdr) == "BAM " || string(hdr) == "BAMC" {
    g.importImageBAM(rs, searchPaths)
  } else if string(hdr[:2]) == "BM" {
    g.importImageBMP(rs)
  } else if string(hdr[:3]) == "GIF" {
    g.importImageGIF(rs)
  } else if bytes.Equal(hdr, []byte{0xff, 0xd8, 0xff, 0xe0}) {
    g.importImageJPG(rs)
  } else if string(hdr[1:4]) == "PNG" {
    g.importImagePNG(rs)
  } else {
    // unsupported
    g.err = errors.New("Unrecognized input format")
  }
}


// Used internally. Imports a BAM resource.
func (g *Graphics) importImageBAM(r io.Reader, searchPaths []string) {
  b := bam.ImportEx(r, searchPaths)
  if b.Error() != nil { g.err = b.Error(); return }

  g.frames = make([]Frame, b.GetFrameLength())
  for idx := 0; idx < len(g.frames); idx++ {
    g.frames[idx].img = b.GetFrameImage(idx)
    g.frames[idx].cx = b.GetFrameCenterX(idx)
    g.frames[idx].cy = b.GetFrameCenterY(idx)
  }

  g.format = TYPE_BAM
}


// Used internally. Imports a BMP resource.
func (g *Graphics) importImageBMP(r io.Reader) {
  g.frames = make([]Frame, 1)
  g.frames[0].img, g.err = bmp.Decode(r)
  if g.err != nil { return }
  g.frames[0].cx, g.frames[0].cy = 0, 0

  g.format = TYPE_BMP
}


// Used internally. Imports a GIF resource.
func (g *Graphics) importImageGIF(r io.Reader) {
  data, err := gif.DecodeAll(r)
  if err != nil { g.err = err; return }

  // t0 := time.Now()
  isAnim := len(data.Image) > 1
  if isAnim { logging.Log("Decoding GIF frames") }
  numFrames := len(data.Image)
  g.frames = make([]Frame, numFrames)

  // Creating master image with global canvas size for all frames
  imgMain := image.NewRGBA(image.Rect(0, 0, data.Config.Width, data.Config.Height))

  for idx := 0; idx < numFrames; idx++ {
    imgCur := data.Image[idx]
    mode := data.Disposal[idx]

    // Backing up current frame content for later
    var imgBackup *image.RGBA = nil
    if mode == gif.DisposalPrevious {
      imgBackup = image.NewRGBA(imgMain.Bounds())
      draw.Draw(imgBackup, imgBackup.Bounds(), imgMain, image.ZP, draw.Src)
    }

    // Rendering frame
    draw.Draw(imgMain, imgCur.Bounds(), imgCur, imgCur.Bounds().Min, draw.Over)
    img := image.NewRGBA(imgMain.Bounds())
    draw.Draw(img, img.Bounds(), imgMain, image.ZP, draw.Src)
    g.frames[idx].img = img
    g.frames[idx].cx, g.frames[idx].cy = 0, 0

    // Cleaning up frame
    switch mode {
      case gif.DisposalBackground:
        // Restore current frame region to background color
        draw.Draw(imgMain, imgCur.Bounds(), image.Transparent, image.ZP, draw.Src)
      case gif.DisposalPrevious:
        // Restore content of previous frame
        draw.Draw(imgMain, imgMain.Bounds(), imgBackup, image.ZP, draw.Src)
      default:  // Don't clear content from previous frame(s)
    }

    if isAnim { logging.LogProgressDot(idx, numFrames, 79 - 19) }  // 19 is length of prefixed string
  }
  if isAnim { logging.OverridePrefix(false, false, false).Logln("") }
  // t1 := time.Now()
  // logging.Logf("DEBUG: importImageGIF() timing = %v\n", t1.Sub(t0))

  g.format = TYPE_GIF
}


// Used internally. Imports a JPG resource.
func (g *Graphics) importImageJPG(r io.Reader) {
  g.frames = make([]Frame, 1)
  g.frames[0].img, g.err = jpeg.Decode(r)
  if g.err != nil { return }
  g.frames[0].cx, g.frames[0].cy = 0, 0

  g.format = TYPE_JPG
}


// Used internally. Imports a PNG resource.
func (g *Graphics) importImagePNG(r io.Reader) {
  g.frames = make([]Frame, 1)
  g.frames[0].img, g.err = png.Decode(r)
  if g.err != nil { return }
  g.frames[0].cx, g.frames[0].cy = 0, 0

  g.format = TYPE_PNG
}
