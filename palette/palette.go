/*
Package palette provides functions for loading color sequences from various input formats without having to
take care of the details.

BAM Creator is released under the BSD 2-clause license. See LICENSE in the project's root folder for more details.
*/
package palette

import (
  "encoding/binary"
  "errors"
  "image"
  "image/color"
  "image/gif"
  "image/png"
  "io"

  "github.com/InfinityTools/bamcreator/bam"
  "golang.org/x/image/bmp"
)


// Import imports the palette from the graphics or palette file pointed to by the ReadSeeker parameter.
//
// Returns a palette on success, or a non-nil error value otherwise.
func Import(rs io.ReadSeeker) (color.Palette, error) {
  if rs == nil { return nil, errors.New("No source specified") }
  pal, err := importPalette(rs)
  return pal, err
}


// Used internally. Delegates import to more specialized functions.
func importPalette(rs io.ReadSeeker) (color.Palette, error) {
  hdr := make([]byte, 4)
  _, err := rs.Read(hdr)
  if err != nil { return nil, err }
  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }

  if string(hdr) == "BAM " || string(hdr) == "BAMC" {
    return importPaletteBAM(rs)
  } else if string(hdr[:2]) == "BM" {
    return importPaletteBMP(rs)
  } else if string(hdr[:3]) == "GIF" {
    return importPaletteGIF(rs)
  } else if string(hdr[1:4]) == "PNG" {
    return importPalettePNG(rs)
  } else if string(hdr) == "RIFF" {
    return importPalettePAL(rs)
  } else {
    // assume Adobe Color Table
    return importPaletteACT(rs)
  }

  return nil, errors.New("Unrecognized graphics or palette file format")
}

// Used internally. Imports a BAM palette.
func importPaletteBAM(rs io.ReadSeeker) (color.Palette, error) {
  hdr := make([]byte, 4)
  _, err := rs.Seek(4, io.SeekStart)
  if err != nil { return nil, err }
  _, err = rs.Read(hdr)
  if err != nil { return nil, err }
  if string(hdr) == "V2  " { return nil, errors.New("BAM V2 does not contain palette data") }
  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }

  b := bam.Import(rs)
  if b.Error() != nil { return nil, b.Error() }

  if b.GetFrameLength() == 0 { return nil, errors.New("BAM does not contain frame data") }
  img := b.GetFrameImage(0)
  if b.Error() != nil { return nil, b.Error() }

  return getImagePalette(img)
}

// Used internally. Imports a BMP palette.
func importPaletteBMP(rs io.ReadSeeker) (color.Palette, error) {
  cfg, err := bmp.DecodeConfig(rs)
  if err != nil { return nil, err }
  if pal, err := getConfigPalette(cfg); err == nil {
    return pal, nil
  }

  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }
  img, err := bmp.Decode(rs)
  if err != nil { return nil, err }
  return getImagePalette(img)
}

// Used internally. Imports a GIF palette.
func importPaletteGIF(rs io.ReadSeeker) (color.Palette, error) {
  cfg, err := gif.DecodeConfig(rs)
  if err != nil { return nil, err }
  if pal, err := getConfigPalette(cfg); err == nil {
    return pal, nil
  }

  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }
  img, err := gif.Decode(rs)
  if err != nil { return nil, err }
  return getImagePalette(img)
}

// Used internally. Imports a PNG palette.
func importPalettePNG(rs io.ReadSeeker) (color.Palette, error) {
  cfg, err := png.DecodeConfig(rs)
  if err != nil { return nil, err }
  if pal, err := getConfigPalette(cfg); err == nil {
    return pal, nil
  }

  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }
  img, err := png.Decode(rs)
  if err != nil { return nil, err }
  return getImagePalette(img)
}

// Used internally. Imports a Windows Palette.
func importPalettePAL(rs io.ReadSeeker) (color.Palette, error) {
  hdr := make([]byte, 12)

  _, err := rs.Read(hdr)
  if err != nil { return nil, err }
  if string(hdr[8:]) != "PAL " { return nil, errors.New("Not a Windows Palette file") }

  // looking for palette data chunk
  for {
    _, err = rs.Read(hdr[:8])
    if err == io.EOF { return nil, errors.New("No palette data found") }
    if err != nil { return nil, err }
    if string(hdr[:4]) == "data" { break }
    size := binary.LittleEndian.Uint32(hdr[4:]) - 4
    _, err := rs.Seek(int64(size), io.SeekCurrent)
    if err != nil { return nil, err }
  }

  size := int(binary.LittleEndian.Uint32(hdr[4:]))
  if size <= 8 { return nil, errors.New("Invalid palette file") }
  size -= 8   // adjust to palette data size

  _, err = rs.Read(hdr[:4])
  if err != nil { return nil, err }
  numCols := int(binary.LittleEndian.Uint16(hdr[2:]))
  if numCols > 256 { numCols = 256 }
  if size < numCols * 4 { return nil, errors.New("Corrupted palette header") }

  buf := make([]byte, numCols*4)
  _, err = rs.Read(buf)
  if err != nil { return nil, err }

  pal := make(color.Palette, 256)
  for i := 0; i < len(pal); i++ {
    if i < numCols {
      ofs := i * 4
      pal[i] = color.RGBA{buf[ofs], buf[ofs+1], buf[ofs+2], 255}
    } else {
      pal[i] = color.RGBA{0, 0, 0, 255}
    }
  }

  return pal, nil
}

// Used internally. Imports a Adobe Color Table.
func importPaletteACT(rs io.ReadSeeker) (color.Palette, error) {
  offset, err := rs.Seek(0, io.SeekEnd)
  if err != nil { return nil, err }
  if offset != 768 && offset != 772 {
    return nil, errors.New("Unrecognized graphics or palette file format")
  }
  _, err = rs.Seek(0, io.SeekStart)
  if err != nil { return nil, err }

  buf := make([]byte, int(offset))
  _, err = rs.Read(buf)
  if err != nil { return nil, err }

  var numCols, transIndex int = 256, -1
  if len(buf) == 772 {
    numCols = int(binary.BigEndian.Uint16(buf[768:]))
    transIndex = int(binary.BigEndian.Uint16(buf[770:]))
  }

  pal := make(color.Palette, 256)
  for i := 0; i < len(pal); i++ {
    if i < numCols {
      ofs := i * 3
      alpha := byte(255)
      if i == transIndex { alpha = 0 }
      pal[i] = color.RGBA{buf[ofs], buf[ofs+1], buf[ofs+2], alpha}
    } else {
      pal[i] = color.RGBA{0, 0, 0, 255}
    }
  }

  return pal, nil
}


// Used internally. Returns a copy of the image palette if available.
func getImagePalette(img image.Image) (color.Palette, error) {
  if img != nil {
    if imgPal, ok := img.(*image.Paletted); ok {
      pal := make(color.Palette, len(imgPal.Palette))
      copy(pal, imgPal.Palette)
      return pal, nil
    }
  }
  return nil, errors.New("No palette data available")
}


// Used internally. Returns the global palette stored in the Config structure if available.
func getConfigPalette(cfg image.Config) (color.Palette, error) {
  if pal, ok := cfg.ColorModel.(color.Palette); ok {
    return pal, nil
  } else {
    return nil, errors.New("No palatte data available")
  }
}
