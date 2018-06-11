/*
BAM Creator is a command line tool for generating BAM V1/V2 files from scripts.

BAM Creator is released under the BSD 2-clause license. See LICENSE in the project's root folder for more details.
*/
package main

import (
  "errors"
  "fmt"
  "image/color"
  "io"
  "os"
  "path/filepath"
  "regexp"
  "strconv"
  "strings"

  "github.com/InfinityTools/bamcreator/config"
  "github.com/InfinityTools/bamcreator/bam"
  "github.com/InfinityTools/bamcreator/graphics"
  "github.com/InfinityTools/bamcreator/palette"
  "github.com/InfinityTools/bamcreator/palette/sort"
  "github.com/InfinityTools/go-logging"
)


const TOOL_NAME     = "BAM Creator"
const VERSION_MAJOR = 0
const VERSION_MINOR = 1


func main() {
  err := loadArgs(os.Args)
  if err != nil {
    fmt.Printf("%v\n", err)
    os.Exit(1)
  }

  // Setting global options
  if b, x := argsVerbose(); x {
    if b {
      logging.SetVerbosity(logging.LOG)
    } else {
      logging.SetVerbosity(logging.ERROR)
    }
  }
  logging.SetPrefixCaller(false)
  if b, x := argsLogStyle(); x && b {
    logging.SetPrefixTimestamp(true)
    logging.SetPrefixLevel(true)
  } else {
    logging.SetPrefixTimestamp(false)
    logging.SetPrefixLevel(false)
  }

  if _, x := argsVersion(); x {
    printVersion()
  } else if _, x := argsHelp(); x {
    printHelp()
  } else if argsExtraLength() == 0 {
    printHelp()
  } else {
    logging.Infoln("Starting BAM conversion")
    err = convert()
    if err != nil {
      logging.Errorf("%v\n", err)
      os.Exit(1)
    }
    logging.Infoln("BAM conversion finished successfully.")
  }
}


func convert() error {
  length := argsExtraLength()
  for idx := 0; idx < length; idx++ {
    configFile := argsExtra(idx)
    if len(configFile) == 0 { continue }  // should not happen
    if configFile == "-" {
      logging.Infof("Starting job %d: (standard input)\n", idx)
    } else {
      logging.Infof("Starting job %d: %s\n", idx, configFile)
    }
    err := convertJob(configFile)
    if err != nil { return fmt.Errorf("Job %d: %v", idx, err) }
    logging.Infof("Finished job %d\n", idx)
  }

  return nil
}


func convertJob(configFile string) error {
  // consistency checks
  isStdIn := configFile == "-"
  if !isStdIn {
    fi, err := os.Stat(configFile)
    if err != nil { return err }
    if !fi.Mode().IsRegular() { return fmt.Errorf("File not found: %q", configFile) }
  }

  var r io.Reader = nil
  if isStdIn {
    r = os.Stdin
  } else {
    fin, err := os.Open(configFile)
    if err != nil { return fmt.Errorf("Cannot open %q: %v", configFile, err) }
    defer fin.Close()
    r = fin
  }
  cfg, err := config.ImportConfig(r)
  if err != nil { return fmt.Errorf("Error parsing configuration: %v", err) }

  err = generateBAM(cfg)
  if err != nil { return err }

  return nil
}


func generateBAM(cfg *config.BamConfig) error {
  if cfg == nil { return errors.New("No configuration data found") }

  logging.Logln("Generating BAM file")
  bamOut := bam.CreateNew()

  // setting up general options
  if b, x := argsThreaded(); x { bam.SetMultiThreaded(b) }
  bamVersion, _ := cfg.GetConfigValueInt(config.SECTION_OUTPUT, config.KEY_OUTPUT_VERSION)
  if i, x := argsBamVersion(); x { bamVersion = int64(i) }
  bamOut.SetBamVersion(int(bamVersion))
  if bamOut.Error() != nil { return bamOut.Error() }

  // setting up BAM version-specific options
  if bamVersion == 1 {
    err := bamSetupV1(cfg, bamOut)
    if err != nil { return err }
  } else if bamVersion == 2 {
    err := bamSetupV2(cfg, bamOut)
    if err != nil { return err }
  }

  // setting up output options
  bamOutFile, _ := cfg.GetConfigValueText(config.SECTION_OUTPUT, config.KEY_OUTPUT_PATH)
  if s, x := argsBamOutput(); x { bamOutFile = s }
  if dir := filepath.Dir(bamOutFile); !directoryExists(dir) {
    err := os.MkdirAll(dir, 0755)
    if err != nil { return fmt.Errorf("Cannot create output path %q: %v", dir, err) }
  }

  pvrzOutPath := "."
  if bamVersion == 2 {
    pvrzOutPath, _ = cfg.GetConfigValueText(config.SECTION_OUTPUT, config.KEY_OUTPUT_PVRZ_PATH)
    if s, x := argsBamPvrzPath(); x { pvrzOutPath = s }
    if len(pvrzOutPath) == 0 {
      // special: use path of BAM output file
      pvrzOutPath = filepath.Dir(bamOutFile)
    }
    if dir, err := filepath.Abs(pvrzOutPath); err != nil || !directoryExists(dir) {
      if err != nil {
        err = os.MkdirAll(pvrzOutPath, 0755)
        if err != nil { return fmt.Errorf("Cannot create output path %q: %v", dir, err) }
      } else {
        err = os.MkdirAll(dir, 0755)
        if err != nil { return fmt.Errorf("Cannot create output path %q: %v", dir, err) }
      }
    }
  }

  // setting up filters
  err := bamSetupFilters(cfg, bamOut)
  if err != nil { return err }

  // printing a summary of current BAM export options (INFO level)
  var sb strings.Builder
  sb.WriteString("Options: ")
  sb.WriteString(fmt.Sprintf("verbose: %v", logging.GetVerbosity() < logging.INFO))
  sb.WriteString(fmt.Sprintf(", threading: %v", bam.GetMultiThreaded()))
  sb.WriteString(fmt.Sprintf(", BAM version:= BAM V%d", bamVersion))
  sb.WriteString(fmt.Sprintf(", BAM output: %q", bamOutFile))
  if bamVersion == 1 {
    sb.WriteString(fmt.Sprintf(", compress: %v", bamOut.GetCompression()))
    sb.WriteString(fmt.Sprintf(", rle: %s", []string{"auto", "off", "on"}[bamOut.GetRle()+1]))
    sb.WriteString(fmt.Sprintf(", alpha: %v", !bamOut.GetDiscardAlpha()))
    i, j := bamOut.GetQuality()
    sb.WriteString(fmt.Sprintf(", quality: (%d, %d)", i, j))
    sb.WriteString(fmt.Sprintf(", speed: %d", bamOut.GetSpeed()))
    sb.WriteString(fmt.Sprintf(", dither: %v", bamOut.GetDither()))
    sb.WriteString(fmt.Sprintf(", # fixed colors: %d", bamOut.GetFixedColorLength()))
    b := bamOut.GetPalette() != nil
    sb.WriteString(fmt.Sprintf(", external palette: %v", b))
  } else {
    sb.WriteString(fmt.Sprintf(", PVRZ output path: %q", pvrzOutPath))
    sb.WriteString(fmt.Sprintf(", PVRZ start index: %d", bamOut.GetPvrzStartIndex()))
    var s string
    switch bamOut.GetPvrzType() {
      case bam.PVRZ_DXT1: s = "DXT1"
      case bam.PVRZ_DXT3: s = "DXT3"
      case bam.PVRZ_DXT5: s = "DXT5"
      default:            s = "auto"
    }
    sb.WriteString(fmt.Sprintf(", PVRZ type: %s", s))
    if bamOut.GetPvrzType() == bam.PVRZ_AUTO {
      sb.WriteString(fmt.Sprintf(", threshold: %v", bamOut.GetPvrzAlphaThreshold()))
    }
    sb.WriteString(fmt.Sprintf(", quality: %d", bamOut.GetPvrzQuality()))
    sb.WriteString(fmt.Sprintf(", weight by alpha: %v", bamOut.GetPvrzWeightByAlpha()))
    sb.WriteString(fmt.Sprintf(", use metric: %v", bamOut.GetPvrzUseMetric()))
  }
  sb.WriteString(fmt.Sprintf(", filters: %d", bamOut.GetFilterLength()))
  logging.Infoln(sb.String())

  // setting up bam frames
  if err := bamLoadFrames(cfg, bamOut); err != nil { return err }

  // setting up bam cycles
  if err := bamLoadCycles(cfg, bamOut); err != nil { return err }

  fout, err := os.Create(bamOutFile)
  if err != nil { return fmt.Errorf("Cannot create %q: %v", bamOutFile, err) }
  defer fout.Close()

  bamOut.ExportEx(fout, pvrzOutPath)

  logging.Logln("Finished generating BAM file")
  return bamOut.Error()
}


func bamSetupV1(cfg *config.BamConfig, bamOut *bam.BamFile) error {
  // Setting BAM V1 options
  bVal, _ := cfg.GetConfigValueBool(config.SECTION_BAMV1, config.KEY_V1_COMPRESS)
  if b, x := argsBamV1Compress(); x { bVal = b }
  bamOut.SetCompression(bVal)

  iVal, _ := cfg.GetConfigValueInt(config.SECTION_BAMV1, config.KEY_V1_RLE)
  if i, x := argsBamV1Rle(); x { iVal = int64(i) }
  bamOut.SetRle(int(iVal))

  bVal, _ = cfg.GetConfigValueBool(config.SECTION_BAMV1, config.KEY_V1_ALPHA)
  if b, x := argsBamV1Alpha(); x { bVal = b }
  bamOut.SetDiscardAlpha(!bVal)

  iVal, _ = cfg.GetConfigValueInt(config.SECTION_BAMV1, config.KEY_V1_QUALITY_MIN)
  if i, x := argsBamV1QualityMin(); x { iVal = int64(i) }
  iVal2, _ := cfg.GetConfigValueInt(config.SECTION_BAMV1, config.KEY_V1_QUALITY_MAX)
  if i, x := argsBamV1QualityMax(); x { iVal2 = int64(i) }
  bamOut.SetQuality(int(iVal), int(iVal2))

  iVal, _ = cfg.GetConfigValueInt(config.SECTION_BAMV1, config.KEY_V1_SPEED)
  if i, x := argsBamV1Speed(); x { iVal = int64(i) }
  bamOut.SetSpeed(int(iVal))

  fVal, _ := cfg.GetConfigValueFloat(config.SECTION_BAMV1, config.KEY_V1_DITHER)
  if f, x := argsBamV1Dither(); x { fVal = float64(f) }
  bamOut.SetDither(float32(fVal))

  iVal, _ = cfg.GetConfigValueInt(config.SECTION_BAMV1, config.KEY_V1_TRANS_COLOR)
  b, g, r, a := byte(iVal), byte(iVal >> 8), byte(iVal >> 16), byte(iVal >> 24)
  bamOut.SetColorKey(color.NRGBA{r, g, b, a})

  bVal, _ = cfg.GetConfigValueBool(config.SECTION_BAMV1, config.KEY_V1_USE_TRANS_COLOR)
  if b, x := argsBamV1UseTransColor(); x { bVal = b }
  bamOut.SetColorKeyEnabled(bVal)

  var sVal string
  sVal, bVal = argsBamV1SortBy()
  if !bVal {
    sVal, _ = cfg.GetConfigValueText(config.SECTION_BAMV1, config.KEY_V1_SORT_BY)
  }
  if len(sVal) > 0 {
    sVal = strings.ToLower(sVal)
    bVal = false
    if idx := strings.LastIndex(sVal, "_reversed"); idx == len(sVal) - len("_reversed") {
      bVal = true
      sVal = sVal[:idx]
    }
    stype := 0
    switch sVal {
      case "lightness":
        stype = sort.SORT_BY_LIGHTNESS
      case "saturation":
        stype = sort.SORT_BY_SATURATION
      case "hue":
        stype = sort.SORT_BY_HUE
      case "red":
        stype = sort.SORT_BY_RED
      case "green":
        stype = sort.SORT_BY_GREEN
      case "blue":
        stype = sort.SORT_BY_BLUE
      case "alpha":
        stype = sort.SORT_BY_ALPHA
      default:
        if sVal != "none" { logging.Warnf("Unrecognized color sort type: %q. Defaulting to \"none\".\n", sVal) }
        stype = sort.SORT_BY_NONE
    }
    if bVal { stype |= sort.SORT_REVERSED }
    bamOut.SetPaletteSortFlags(stype)
  }

  sVal, bVal = argsBamV1Palette()
  if !bVal {
    bVal, _ = cfg.GetConfigValueBool(config.SECTION_BAMV1, config.KEY_V1_USE_PALETTE)
    if bVal {
      sVal, _ = cfg.GetConfigValueText(config.SECTION_BAMV1, config.KEY_V1_PALETTE)
    }
  }
  if bVal && len(sVal) > 0 {
    fin, err := os.Open(sVal)
    if err != nil { return fmt.Errorf("External palette: %v", err) }
    defer fin.Close()
    var pal color.Palette
    pal, err = palette.Import(fin)
    if err != nil { return fmt.Errorf("External palette: %v", err) }
    bamOut.SetPalette(pal)
  } else {
    bamOut.ClearPalette()
  }

  seqVal, ok := cfg.GetConfigValueIntSeq(config.SECTION_BAMV1, config.KEY_V1_FIXED_COLORS)
  if ok {
    for _, col := range seqVal {
      b, g, r, a := byte(col), byte(col >> 8), byte(col >> 16), byte(col >> 24)
      bamOut.AddFixedColor(color.NRGBA{r, g, b, a})
    }
  }

  return bamOut.Error()
}


func bamSetupV2(cfg *config.BamConfig, bamOut *bam.BamFile) error {
  // Setting BAM V2 options
  iVal, _ := cfg.GetConfigValueInt(config.SECTION_BAMV2, config.KEY_V2_START_INDEX)
  if i, x := argsBamV2StartIndex(); x { iVal = int64(i) }
  bamOut.SetPvrzStartIndex(int(iVal))

  iVal, _ = cfg.GetConfigValueInt(config.SECTION_BAMV2, config.KEY_V2_ENCODING)
  if i, x := argsBamV2Encoding(); x { iVal = int64(i) }
  switch iVal {
    case 1: iVal = bam.PVRZ_DXT1
    case 2: iVal = bam.PVRZ_DXT3
    case 3: iVal = bam.PVRZ_DXT5
    default: iVal = bam.PVRZ_AUTO
  }
  bamOut.SetPvrzType(int(iVal))

  fVal, _ := cfg.GetConfigValueFloat(config.SECTION_BAMV2, config.KEY_V2_THRESHOLD)
  if f, x := argsBamV2Threshold(); x { fVal = float64(f) }
  bamOut.SetPvrzAlphaThreshold(float32(fVal))

  iVal, _ = cfg.GetConfigValueInt(config.SECTION_BAMV2, config.KEY_V2_QUALITY)
  if i, x := argsBamV2Quality(); x { iVal = int64(i) }
  switch iVal {
    case 0: iVal = bam.QUALITY_LOW
    case 2: iVal = bam.QUALITY_HIGH
    default: iVal = bam.QUALITY_DEFAULT
  }
  bamOut.SetPvrzQuality(int(iVal))

  bVal, _ := cfg.GetConfigValueBool(config.SECTION_BAMV2, config.KEY_V2_WEIGHT_ALPHA)
  if b, x := argsBamV2WeightAlpha(); x { bVal = b }
  bamOut.SetPvrzWeightByAlpha(bVal)

  bVal, _ = cfg.GetConfigValueBool(config.SECTION_BAMV2, config.KEY_V2_USE_METRIC)
  if b, x := argsBamV2UseMetric(); x { bVal = b }
  bamOut.SetPvrzUseMetric(bVal)

  return bamOut.Error()
}


func bamSetupFilters(cfg *config.BamConfig, bamOut *bam.BamFile) error {
  // initializing filters
  numFilters := cfg.GetConfigFilterLength()
  for idx := 0; idx < numFilters; idx++ {
    name, ok := cfg.GetConfigFilterName(idx)
    if !ok { return fmt.Errorf("Empty filter at index=%d", idx) }
    options, ok := cfg.GetConfigFilterOptions(idx)
    if !ok { return fmt.Errorf("Could not evaluate filter %q at index=%d", name, idx) }
    f := bamOut.CreateFilter(name)
    if f == nil { return fmt.Errorf("Could not create filter: %s", name) }
    for idx2, option := range options {
      if option == nil || len(option) < 2 { return fmt.Errorf("Could not evaluate option %d of filter %q (index=%d)", idx2, name, idx) }
      err := f.SetOption(option[0], option[1])
      if err != nil { return fmt.Errorf("Filter %q (index=%d), option %q: %v", name, idx, option[0], err) }
    }
    bamOut.AddFilter(f)
  }
  if bamOut.Error() != nil { return bamOut.Error() }

  // applying override options
  if options, x := argsFilterOptions(); x {
    reg := regexp.MustCompile("(0|[1-9][0-9]*):([^=]+)=(.*)")
    for _, option := range options {
      values := reg.FindStringSubmatch(option)  // should return []string{"full-string", "idx", "key", "value"}
      if values == nil || len(values) < 4 { return fmt.Errorf("Invalid filter option: %s", option) }
      index, err := strconv.Atoi(strings.TrimSpace(values[1]))
      if err != nil { return fmt.Errorf("Invalid filter index: %s", values[1]) }
      key, value := strings.TrimSpace(values[2]), strings.TrimSpace(values[3])
      if index < 0 || index >= bamOut.GetFilterLength() {
        logging.Warnf("Filter index out of bounds: %d. Skipping option...\n", index)
        continue
      }
      filter := bamOut.GetFilter(index)
      logging.Logf("Filter #%d (%s): Overriding option %s = %s\n", index, filter.GetName(), key, value)
      err = filter.SetOption(key, value)
      if err != nil {
        logging.Warnf("Filter #%d (%s): Could not set option %s = %s: %v\n", index, filter.GetName(), key, value, err)
      }
    }
  }

  return nil
}


func bamLoadFrames(cfg *config.BamConfig, bamOut *bam.BamFile) error {
  useStatic, _ := cfg.GetConfigValueBool(config.SECTION_INPUT, config.KEY_INPUT_STATIC)

  // setting up pvrz search paths
  searchPaths := make([]string, 0)
  sp, _ := cfg.GetConfigValueTextSeq(config.SECTION_INPUT, config.KEY_INPUT_SEARCH)
  for _, path := range sp {
    if path != "" {
      if !directoryExists(path) { return fmt.Errorf("Input search path does not exist: %q", path) }
    }
    searchPaths = append(searchPaths, path)
  }
  if len(searchPaths) == 0 { searchPaths = append(searchPaths, "") }

  // preparing center entries
  centers, _ := cfg.GetConfigValueIntSeq(config.SECTION_SETTINGS, config.KEY_CENTERS)

  // importing frames
  logging.Logln("Importing input graphics files")
  var err error = nil
  if useStatic {
    err = bamLoadFramesStatic(cfg, bamOut, searchPaths, centers)
  } else {
    err = bamLoadFramesSequence(cfg, bamOut, searchPaths, centers)
  }
  if err != nil { return err }
  logging.Logln("Finished importing input graphics files")

  return nil
}


// Adds frames and center positions from a static list of file path entries
func bamLoadFramesStatic(cfg *config.BamConfig, bamOut *bam.BamFile, searchPaths []string, centers []int64) error {
  entries, _ := cfg.GetConfigValueTextSeq(config.SECTION_INPUT, config.KEY_INPUT_FILES)
  if len(entries) == 0 { return fmt.Errorf("No input files defined") }

  for eidx, entry := range entries {
    // min, max values of -1 indicate to use input image defaults
    fileName, min, max, err := parseInputFile(entry)
    logging.Logf("Importing %s\n", fileName)
    if err != nil { return err }
    if !fileExists(fileName) { return fmt.Errorf("Input file %d does not exist: %q", eidx, fileName) }
    sp := expandSearchPaths(searchPaths, fileName)
    g, err := loadGraphics(fileName, sp)
    if err != nil { return err }

    // expand last known center entry to remaining frames if needed
    prevX, prevY := 0, 0
    if len(centers) > 0 { prevX, prevY = getCenter(centers[len(centers) - 1]) }

    length := g.GetImageLength()
    if min < 0 { min = 0 }
    if max < 0 { max = length }
    if min >= length || max > length { return fmt.Errorf("Frame range of input file %d is out of bounds: have=[%d,%d], need=[%d,%d]", eidx, min, max, 0, length) }
    if min == max { logging.Warnf("Frame range of input file %d is empty. Skipping.\n", eidx) }
    for imgIdx := min; imgIdx < max; imgIdx++ {
      img := g.GetImage(imgIdx)
      x, y := g.GetCenter(imgIdx)

      // override by config centers
      if bamOut.GetFrameLength() < len(centers) {
        x, y = getCenter(centers[bamOut.GetFrameLength()])
      } else if g.GetImageType() != graphics.TYPE_BAM {
        x, y = prevX, prevY
      }

      bamOut.AddFrame(x, y, img)
    }
  }

  return nil
}


// Adds frames and center positions from a file sequence generated by parameters
func bamLoadFramesSequence(cfg *config.BamConfig, bamOut *bam.BamFile, searchPaths []string, centers []int64) error {
  path, _ := cfg.GetConfigValueText(config.SECTION_INPUT, config.KEY_INPUT_PATH)
  prefix, _ := cfg.GetConfigValueText(config.SECTION_INPUT, config.KEY_INPUT_PREFIX)
  ext, _ := cfg.GetConfigValueText(config.SECTION_INPUT, config.KEY_INPUT_EXT)
  suffixStart, _ := cfg.GetConfigValueInt(config.SECTION_INPUT, config.KEY_INPUT_SUFFIX_START)
  suffixEnd, _ := cfg.GetConfigValueInt(config.SECTION_INPUT, config.KEY_INPUT_SUFFIX_END)
  suffixLen, _ := cfg.GetConfigValueInt(config.SECTION_INPUT, config.KEY_INPUT_SUFFIX_LEN)

  // sequence may be incremented or decremented
  var inc int64
  if suffixEnd < suffixStart { inc = -1; suffixEnd-- } else { inc = 1; suffixEnd++ }
  for index := suffixStart; index != suffixEnd; index += inc {
    fileName := config.AssembleFilePath(path, prefix, ext, index, suffixLen)
    if !fileExists(fileName) { return fmt.Errorf("Input file does not exist: %q", fileName) }
    logging.Logf("Importing %s\n", fileName)
    sp := expandSearchPaths(searchPaths, fileName)
    g, err := loadGraphics(fileName, sp)
    if err != nil { return err }

    // expand last known center entry to remaining frames if needed
    prevX, prevY := 0, 0
    if len(centers) > 0 { prevX, prevY = getCenter(centers[len(centers) - 1]) }

    length := g.GetImageLength()
    for imgIdx := 0; imgIdx < length; imgIdx++ {
      img := g.GetImage(imgIdx)
      x, y := g.GetCenter(imgIdx)

      // override by config centers
      if len(centers) > bamOut.GetFrameLength() {
        x, y = getCenter(centers[bamOut.GetFrameLength()])
      } else if g.GetImageType() != graphics.TYPE_BAM {
        x, y = prevX, prevY
      }

      bamOut.AddFrame(x, y, img)
    }
  }

  return nil
}


// Adds cycle definitions to the bam object
func bamLoadCycles(cfg *config.BamConfig, bamOut *bam.BamFile) error {
  cycleSeqs, ok := cfg.GetConfigValueIntSeq2(config.SECTION_SETTINGS, config.KEY_SEQUENCES)
  if !ok { return fmt.Errorf("No BAM cycle definitions found") }
  if len(cycleSeqs) == 0 { cycleSeqs = append(cycleSeqs, []int64{0}) }

  for idx, cycle := range cycleSeqs {
    if len(cycle) == 0 { return fmt.Errorf("Cycle %d is empty", idx) }
    c := make([]uint16, len(cycle))
    for idx2 := 0; idx2 < len(cycle); idx2++ {
      c[idx2] = uint16(cycle[idx2])
    }
    bamOut.AddCycle(c)
    if bamOut.Error() != nil { return bamOut.Error() }
  }

  return nil
}


// Splits combined center into x and y.
func getCenter(value int64) (x, y int) {
  x = int(int16(value))
  y = int(int16(value >> 16))
  return
}

// Returns file path and frame index range of min (inclusive) and max (exclusive).
func parseInputFile(entry string) (path string, min, max int, err error) {
  path = entry
  min, max = -1, -1
  err = nil

  regRange := regexp.MustCompile("(:[0-9]+){1,2}$")
  regSplit := regexp.MustCompile(":")
  indices := regRange.FindStringIndex(entry)
  if indices != nil {
    path = entry[:indices[0]]
    items := regSplit.Split(entry[indices[0]+1:indices[1]], -1) // first character in range would cause to produce an empty item, skipping
    if len(items) > 0 {
      min, err = strconv.Atoi(items[0])
      if err != nil { err = fmt.Errorf("Input entry %q: invalid frame index=%s", entry, items[0]); return }
      if min < 0 { min = 0 }
      if len(items) > 1 {
        max, err = strconv.Atoi(items[1])
        if err != nil { err = fmt.Errorf("Input entry %q: invalid frame index=%s", entry, items[1]); return }
        if max >= 0 && max < min { min, max = max, min }
      }
    }
  }
  return
}


// Replaces empty search path entries by the path of the specified file.
func expandSearchPaths(paths []string, file string) []string {
  dir := filepath.Dir(file)
  retVal := make([]string, 0)
  for idx := 0; idx < len(paths); idx++ {
    if len(paths[idx]) == 0 {
      retVal = append(retVal, dir)
    } else {
      retVal = append(retVal, paths[idx])
    }
  }
  return retVal
}


// Loads graphics file with optional pvrz search paths
func loadGraphics(fileName string, searchPaths []string) (*graphics.Graphics, error) {
  fin, err := os.Open(fileName)
  if err != nil { return nil, fmt.Errorf("Could not open %q: %v", fileName, err) }
  defer fin.Close()

  if searchPaths == nil { searchPaths = []string{filepath.Dir(fileName)} }
  retVal := graphics.Import(fin, searchPaths)
  return retVal, retVal.Error()
}


func printHelp() {
  fmt.Printf("Usage: %s [options] configfile [configfile2 ...]\n", os.Args[0])
  const helpText = "Allows you to build BAM V1 or BAM V2 files based on settings defined in configuration files.\n" +
                   "\n" +
                // "...............................................................................\n" +
                   "Options:\n" +
                   "  --verbose                 Show additional log messages during the conversion\n" +
                   "                            process.\n" +
                   "  --silent                  Suppress any log messages during the conversion\n" +
                   "                            process except for errors.\n" +
                   "  --log-style               Print log messages in log style, complete with\n" +
                   "                            timestamp and log level.\n" +
                   "  --threaded                Enable multithreading for BAM conversion. May speed\n" +
                   "                            up the conversion process on multi-core systems.\n" +
                   "                            Enabled by default if multiple CPU cores are\n" +
                   "                            detected.\n" +
                   "  --no-threaded             Disable multithreading for BAM conversion.\n" +
                   "  --bam-version version     Set BAM output version. Can be 1 for BAM V1 or\n" +
                   "                            2 for BAM V2. Overrides setting in the config file.\n" +
                   "  --bam-output file         Set BAM output file. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bam-pvrz-path path      Set PVRZ output path for BAM V2 output. Overrides\n" +
                   "                            setting in the config file.\n" +
                   "  --bamv1-compress          Enable BAM V1 compression. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv1-no-compress       Disable BAM V1 compression. Overrides setting in\n" +
                   "                            the config file.\n" +
                   "  --bamv1-rle type          Set RLE frame compression type. Allowed types:\n" +
                   "                               -1     Decide based on resulting frame size\n" +
                   "                                0     Always disable RLE encoding\n" +
                   "                                1     Always enable RLE encoding\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --bamv1-alpha             Preserve alpha in BAM V1 palette. Overrides setting\n" +
                   "                            in the config file.\n" +
                   "  --bamv1-no-alpha          Discard alpha in BAM V1 palette. Overrides setting\n" +
                   "                            in the config file.\n" +
                   "  --bamv1-quality-min qmin  Set minimum quality for BAM V1 color quantization.\n" +
                   "                            Allowed range: [0, 100]. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv1-quality-max qmax  Set maximum quality for BAM V1 color quantization.\n" +
                   "                            Allowed range: [0, 100]. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv1-speed value       Set speed for palette generation. Allowed range:\n" +
                   "                            [1, 10]. Overrides setting in the config file.\n" +
                   "  --bamv1-dither value      Set dither strength for output graphics. Value must\n" +
                   "                            be in range [0.0, 1.0]. Set to 0 to disable.\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --bamv1-transcolor        Enable to treat the color defined in the config\n" +
                   "                            file as transparent. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv1-no-transcolor     Don't treat the color defined in the config file as\n" +
                   "                            transparent. Overrides setting in the config file.\n" +
                   "  --bamv1-sort type         Sort palette by the specified type. The following\n" +
                   "                            types are recognized: none, lightness, saturation,\n" +
                   "                            hue, red, green, blue, alpha. Append _reversed to\n" +
                   "                            reverse the sort order. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv1-palette file      Specify an external palette. Overrides settings in\n" +
                   "                            the config file.\n" +
                   "  --bamv2-start-index idx   Set start index for PVRZ files generated by BAM V2.\n" +
                   "                            Allowed range: [0, 99999]. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv2-encoding type     PVRZ pixel encoding type. Available types:\n" +
                   "                                0     Determine automatically\n" +
                   "                                1     Enforce DXT1 (BC1)\n" +
                   "                                2     Enforce DXT3 (BC2) [unsupported by games]\n" +
                   "                                3     Enforce DXT5 (BC3)\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --bamv2-threshold value   Percentage threshold for determining PVRZ pixel\n" +
                   "                            encoding type. Allowed range: [0.0, 100.0].\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --bamv2-quality value     Quality of PVRZ pixel encoding. in range [0, 2],\n" +
                   "                            where 0 is lowest quality and 2 is highest quality.\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --bamv2-weight-alpha      Weight pixels by alpha. May improve visual quality\n" +
                   "                            for alpha-blended pixels. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv2-no-weight-alpha   Don't weight pixels by alpha. Overrides setting in\n" +
                   "                            the config file.\n" +
                   "  --bamv2-use-metric        Apply perceptual metric to encoded pixels. May\n" +
                   "                            improve perceived quality. Overrides setting in the\n" +
                   "                            config file.\n" +
                   "  --bamv2-no-use-metric     Don't apply perceptual metric to encoded pixels.\n" +
                   "                            Overrides setting in the config file.\n" +
                   "  --filter idx:key=value    Set or override a filter option. 'idx' indicates\n" +
                   "                            the filter index in the list of filters, starting\n" +
                   "                            at index 0. 'key' and 'value' define a single\n" +
                   "                            filter option key and value pair. Wrap the whole\n" +
                   "                            definition in quotes if it contains spaces.\n" +
                   "                            Add multiple --filter instances to set or override\n" +
                   "                            multiple filter options.\n" +
                   "  --help                    Print this help and terminate.\n" +
                   "  --version                 Print version information and terminate.\n" +
                   "\n" +
                   "Note: Use minus sign (-) in place of configfile to read configuration data\n" +
                   "      from standard input."
  fmt.Println(helpText)
}


func printVersion() {
  fmt.Printf("%s version %d.%d\n", TOOL_NAME, VERSION_MAJOR, VERSION_MINOR)
}


// Used internally. Returns whether the specified filename points to a regular existing file.
func fileExists(file string) bool {
  if len(file) == 0 { return false }
  fi, err := os.Stat(file)
  if err != nil { return false }
  return fi.Mode().IsRegular()
}

// Used internally. Returns whether the specified path points to an existing directory.
func directoryExists(dir string) bool {
  if len(dir) == 0 { return true }  // special
  fi, err := os.Stat(dir)
  if err != nil { return false }
  return fi.Mode().IsDir()
}

