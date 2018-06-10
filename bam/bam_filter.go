package bam
// Provides base functionality for processing BAM filters.

import (
  "fmt"
  "image"
  "image/color"
  "image/draw"
  "runtime"
  "strconv"
  "strings"
  "sync"
  // "time"

  "github.com/InfinityTools/go-logging"
  "github.com/pbenner/threadpool"
)

// BamFilter provides functions for applying color or transform effects to individual frames.
type BamFilter interface {
  // GetName returns the name of the filter for identification purposes.
  GetName() string
  // GetOption returns the option of given name. Content of return value is filter specific.
  GetOption(key string) interface{}
  // SetOption adds or updates an option of the given key to the filter. Return value indicates whether option is valid.
  SetOption(key, value string) error
  // Process applies the filter effect to the specified BAM frame and returns the transformed BAM frame.
  // inFrames contains list of frames from a previous filter pass or initial unfiltered frames. It can be used
  // by filters that gather statistical data from multiple frames in the BAM.
  Process(frame BamFrame, inFrames []BamFrame) (BamFrame, error)
}

type optionsMap map[string]interface{}

type filterType struct {
  name    string
  create  func() BamFilter
}

type filterMap map[string]filterType


var filterTypes filterMap = make(filterMap)


// CreateFilter creates a new filter of the given type. Returns nil if the does not exist or cannot be created.
func (bam *BamFile) CreateFilter(filterName string) BamFilter {
  f, ok := filterTypes[filterName]
  if !ok { return nil }
  return f.create()
}


// Used internally. Applies the chain of filters to input frames and returns the result.
func (bam *BamFile) applyFilters() (out []BamFrame, err error) {
  // Preparing output frame list
  tmp := make([]BamFrame, len(bam.frames)) // working array of frames
  copy(tmp, bam.frames)
  out = make([]BamFrame, len(bam.frames)) // updated with resulting frames after each filter
  copy(out, bam.frames)

  // Preparing filter chain
  filters := make([]BamFilter, 0, len(bam.filters) + 1)

  // Applying transColor config option (BAM V1 option)
  if col := bam.GetColorKey(); bam.GetBamVersion() == BAM_V1 &&
                               bam.GetColorKeyEnabled() {
    f := bam.CreateFilter("replace")
    r, g, b, a := NRGBA(col)
    f.SetOption("match", fmt.Sprintf("0x%02x%02x%02x%02x", a, r, g, b))
    filters = append(filters, f)
  }

  // Adding remaining filters
  filters = append(filters, bam.filters...)

  // applying filter chain
  // t0 := time.Now()
  for filterIdx, filter := range filters {
    msg := fmt.Sprintf("Applying filter %q", filter.GetName())
    logging.Log(msg)
    if GetMultiThreaded() {
      // Multi-threaded operation
      numThreads := runtime.NumCPU()
      pool := threadpool.NewThreadPool(numThreads, len(tmp))
      g := pool.NewJobGroup()
      var m sync.Mutex
      globalFilterIdx := 0
      for frameIdx, inFrame := range tmp {
        idx := frameIdx
        frm := inFrame
        err = pool.AddJob(g, func(pool threadpool.ThreadPool, erf func() error) error {
          if erf() != nil { return nil }
          outFrame, err := filter.Process(frm, out)
          if err != nil {
            err = fmt.Errorf("Filter #%d (%s) at frame %d: %v", filterIdx, filter.GetName(), idx, err);
            return err
          }
          tmp[idx] = outFrame
          func() {
            m.Lock()
            defer m.Unlock()
            logging.LogProgressDot(globalFilterIdx, len(tmp), 79 - len(msg))
            globalFilterIdx++
          }()
          return nil
        })
        if err != nil { break }
      }
      if err2 := pool.Wait(g); err2 != nil && err == nil { err = err2 }
      if err != nil {
        logging.OverridePrefix(false, false, false).Logln("")
        err = fmt.Errorf("Filter #%d (%s) at frame %d: %v", filterIdx, filter.GetName(), globalFilterIdx, err)
        return
      }
    } else {
      // Single-threaded operation
      for frameIdx, inFrame := range tmp {
        var outFrame BamFrame
        outFrame, err = filter.Process(inFrame, out)
        if err != nil {
          logging.OverridePrefix(false, false, false).Logln("")
          err = fmt.Errorf("Filter #%d (%s) at frame %d: %v", filterIdx, filter.GetName(), frameIdx, err)
          return
        }
        tmp[frameIdx] = outFrame
        logging.LogProgressDot(frameIdx, len(tmp), 79 - len(msg))
      }
    }
    logging.OverridePrefix(false, false, false).Logln("")
    copy(out, tmp)
  }
  // t1 := time.Now()
  // fmt.Printf("DEBUG: Filter timing = %v\n", t1.Sub(t0))

  return
}


// registerFilter registers a BamFilter for use by the converter. It must be called by each filter once.
func registerFilter(name string, create func() BamFilter) {
  filterTypes[name] = filterType{name, create}
}

// cloneImage is a helper function that creates a copy of the specified image and returns it as image.RGBA or
// image.Paletted depending on source image format and flags.
// Set forceRGBA to always return an image of type RGBA. Otherwise paletted images will be cloned as paletted image.
// Always returns a valid Image object.
func cloneImage(img image.Image, forceRGBA bool) image.Image {
  if img == nil { return image.NewRGBA(image.Rect(0, 0, 1, 1)) }

  var imgOut draw.Image = nil
  b := img.Bounds()
  if imgPal, ok := img.(*image.Paletted); ok && !forceRGBA {
    imgPalNew := image.NewPaletted(b, make(color.Palette, len(imgPal.Palette)))
    copy(imgPalNew.Palette, imgPal.Palette)
    imgOut = imgPalNew
  } else {
    imgOut = image.NewRGBA(b)
  }
  draw.Draw(imgOut, b, img, b.Min, draw.Src)

  return imgOut
}

// A more specialized helper function that converts the given image into an image of image.RGBA type.
func ToRGBA(img image.Image) *image.RGBA {
  if img == nil { return nil }
  if imgRGBA, ok := img.(*image.RGBA); ok { return imgRGBA }
  imgOut := image.NewRGBA(img.Bounds())
  draw.Draw(imgOut, imgOut.Bounds(), img, image.ZP, draw.Src)
  return imgOut
}


// Converts string (oct/dec/hex) without range restrictions.
func parseInt(value string) (int, error) {
  ret, err := strconv.ParseInt(value, 0, 0)
  if err != nil { return 0, fmt.Errorf("not an int: %s", value) }
  return int(ret), nil
}

// Converts string (oct/dec/hex) into int in range [min, max] (both inclusive).
func parseIntRange(value string, min, max int) (int, error) {
  if max < min { min, max = max, min }
  ret, err := strconv.ParseInt(value, 0, 0)
  if err != nil { return 0, fmt.Errorf("not an int: %s", value) }
  if int(ret) < min || int(ret) > max { return 0, fmt.Errorf("not in range [%d, %d]: %s", min, max, value) }
  return int(ret), nil
}

// Converts string (oct/dec/hex) into unsigned int without range restrictions.
func parseUint(value string) (uint, error) {
  ret, err := strconv.ParseUint(value, 0, 0)
  if err != nil { return 0, fmt.Errorf("not an uint: %s", value) }
  return uint(ret), nil
}

// Converts string (oct/dec/hex) into unsigned int in range [min, max] (both inclusive).
func parseUintRange(value string, min, max uint) (uint, error) {
  if max < min { min, max = max, min }
  ret, err := strconv.ParseUint(value, 0, 0)
  if err != nil { return 0, fmt.Errorf("not an uint: %s", value) }
  if uint(ret) < min || uint(ret) > max { return 0, fmt.Errorf("not in range [%d, %d]: %s", min, max, value) }
  return uint(ret), nil
}

// Converts string into float without range restrictions.
func parseFloat(value string) (float64, error) {
  ret, err := strconv.ParseFloat(value, 64)
  if err != nil { return 0, fmt.Errorf("not a float: %s", value) }
  return ret, nil
}

// Converts string into float in range [min, max] (both inclusive).
func parseFloatRange(value string, min, max float64) (float64, error) {
  if max < min { min, max = max, min }
  ret, err := strconv.ParseFloat(value, 64)
  if err != nil { return 0, fmt.Errorf("not a float: %s", value) }
  if ret < min || ret > max { return 0, fmt.Errorf("not in range [%v, %v]: %s", min, max, value) }
  return ret, nil
}

// Converts string into bool.
func parseBool(value string) (bool, error) {
  ret, err := strconv.ParseBool(value)
  if err != nil {
    n, err := strconv.ParseInt(value, 0, 0)
    if err != nil { return false, fmt.Errorf("not a boolean: %s", value) }
    ret = n != 0
  }
  return ret, nil
}

// Converts string with comma-separated values into sequence of ints.
func parseIntSeq(value string) ([]int, error) {
  seq := make([]int, 0)
  s := strings.Split(value, ",")
  for idx, item := range s {
    item = strings.TrimSpace(item)
    n, err := strconv.ParseInt(item, 0, 0)
    if err != nil { return seq, fmt.Errorf("item %d not an int: %s", idx, item) }
    seq = append(seq, int(n))
  }
  return seq, nil
}

// Converts string with comma-separated values into sequence of floats.
func parseFloatSeq(value string) ([]float64, error) {
  seq := make([]float64, 0)
  s := strings.Split(value, ",")
  for idx, item := range s {
    item = strings.TrimSpace(item)
    f, err := strconv.ParseFloat(item, 0)
    if err != nil { return seq, fmt.Errorf("item %d not a float: %s", idx, item) }
    seq = append(seq, f)
  }
  return seq, nil
}

// Converts string with comma-separated values into sequence of floats.
func parseStringSeq(value string) ([]string, error) {
  seq := make([]string, 0)
  s := strings.Split(value, ",")
  for _, item := range s {
    item = strings.TrimSpace(item)
    seq = append(seq, item)
  }
  return seq, nil
}


// Converts a Color into hue, saturation and lightness
func colorToHSL(col color.Color) (h, s, l float64) {
  r, g, b, a := NRGBA(col)
  if a > 0 {
    fr, fg, fb := float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0
    cmin := fr; if fg < cmin { cmin = fg }; if fb < cmin { cmin = fb }
    cmax := fr; if fg > cmax { cmax = fg }; if fb > cmax { cmax = fb }
    csum := cmax + cmin
    cdelta := cmax - cmin
    cdelta2 := cdelta / 2.0

    l = csum / 2.0

    if cdelta != 0.0 {
      if l < 0.5 {
        s = cdelta / csum
      } else {
        s = cdelta / (2.0 - csum)
      }

      dr := ((cmax - fr) / 6.0 + cdelta2) / cdelta
      dg := ((cmax - fg) / 6.0 + cdelta2) / cdelta
      db := ((cmax - fb) / 6.0 + cdelta2) / cdelta

      switch cmax {
        case fr:  h = db - dg
        case fg:  h = 1.0/3.0 + dr - db
        default:  h = 2.0/3.0 + dg - dr
      }

      if h < 0.0 {
        h += 1.0
      }
      if h > 1.0 {
        h -= 1.0
      }
    }
  }
  return
}

// Converts given slice[0:4] of premultiplied RGBA values into hue, saturation and lightness
func rgbaToHSL(slice []byte) (h, s, l float64) {
  if slice[3] > 0 {
    fa := float64(slice[3])
    fr, fg, fb := float64(slice[0]) / fa, float64(slice[1]) / fa, float64(slice[2]) / fa
    cmin := fr; if fg < cmin { cmin = fg }; if fb < cmin { cmin = fb }
    cmax := fr; if fg > cmax { cmax = fg }; if fb > cmax { cmax = fb }
    csum := cmax + cmin
    cdelta := cmax - cmin
    cdelta2 := cdelta / 2.0

    l = csum / 2.0

    if cdelta != 0.0 {
      if l < 0.5 {
        s = cdelta / csum
      } else {
        s = cdelta / (2.0 - csum)
      }

      dr := ((cmax - fr) / 6.0 + cdelta2) / cdelta
      dg := ((cmax - fg) / 6.0 + cdelta2) / cdelta
      db := ((cmax - fb) / 6.0 + cdelta2) / cdelta

      switch cmax {
        case fr:  h = db - dg
        case fg:  h = 1.0/3.0 + dr - db
        default:  h = 2.0/3.0 + dg - dr
      }

      if h < 0.0 {
        h += 1.0
      }
      if h > 1.0 {
        h -= 1.0
      }
    }
  }
  return
}

// Converts HSL values back to RGB values in range [0, 1]
func hslToRGB(h, s, l float64) (r, g, b float64) {
  if s == 0.0 {
    r, g, b = l, l, l
  } else {
    var f2 float64
    if l < 0.5 {
      f2 = l * (1.0 + s)
    } else {
      f2 = (l + s) - (s * l)
    }
    f1 := 2.0 * l - f2
    f21sub := f2 - f1

    // red
    t := h + 1.0/3.0
    if t < 0.0 { t += 1.0 }
    if t > 1.0 { t -= 1.0 }
    switch {
      case 6.0 * t < 1.0: r = f1 + f21sub * 6.0 * t
      case 2.0 * t < 1.0: r = f2
      case 3.0 * t < 2.0: r = f1 + f21sub * (2.0/3.0 - t) * 6.0
      default:            r = f1
    }
    if r < 0.0 { r = 0.0 }
    if r > 1.0 { r = 1.0 }

    // green
    t = h
    switch {
      case 6.0 * t < 1.0: g = f1 + f21sub * 6.0 * t
      case 2.0 * t < 1.0: g = f2
      case 3.0 * t < 2.0: g = f1 + f21sub * (2.0/3.0 - t) * 6.0
      default:            g = f1
    }
    if g < 0.0 { g = 0.0 }
    if g > 1.0 { g = 1.0 }

    // blue
    t = h - 1.0/3.0
    if t < 0.0 { t += 1.0 }
    if t > 1.0 { t -= 1.0 }
    switch {
      case 6.0 * t < 1.0: b = f1 + f21sub * 6.0 * t
      case 2.0 * t < 1.0: b = f2
      case 3.0 * t < 2.0: b = f1 + f21sub * (2.0/3.0 - t) * 6.0
      default:            b = f1
    }
    if b < 0.0 { b = 0.0 }
    if b > 1.0 { b = 1.0 }
  }
  return
}
