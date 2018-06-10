package config
// Parse functionality for JSON structures.

import (
  "encoding/json"
  "fmt"
  "regexp"
  "strconv"
  "strings"

  "github.com/InfinityTools/go-logging"
)

// Used internally by json.Unmarshal to store output settings.
type JsonOutput struct {
  Version       int64
  File          string
  PvrzPath      string
}

// Used internally by json.Unmarshal to store file input sequences.
type JsonInputSequence struct {
  Path          string
  Prefix        string
  SuffixStart   int64
  SuffixEnd     int64
  SuffixLength  int64
  Ext           string
}

// Used internally by json.Unmarshal to store input settings.
type JsonInput struct {
  Static        bool
  Files         []string
  FileSequence  JsonInputSequence
  Search        []string
}

// Used internally by json.Unmarshal to store bam settings.
type JsonSettings struct {
  Center        [][]int64
  Sequence      [][]int64
}

// Used internally by json.Unmarshal to store bam v1 settings.
type JsonBamV1 struct {
  Compress      bool
  Rle           int64
  Alpha         bool
  QualityMin    int64
  QualityMax    int64
  Speed         int64
  Dither        float64
  Color         []string
  TransColor    string
  UseTransColor bool
  SortBy        string
  UsePalette    bool
  Palette       string
}

// Used internally by json.Unmarshal to store bam v2 settings.
type JsonBamV2 struct {
  StartIndex    int64
  Encoding      int64
  Threshold     float64
  Quality       int64
  WeightAlpha   bool
  UseMetric     bool
}

// Used internally by json.Unmarshal to store filter settings.
type JsonFilterOptions struct {
  Key           string
  Value         string
}

// Used internally by json.Unmarshal to store filter options.
type JsonFilter struct {
  Name          string
  Options       []JsonFilterOptions
}

// Used internally by json.Unmarshal to store configuration data from JSON scripts.
type JsonGenerator struct {
  Output        JsonOutput
  Input         JsonInput
  Settings      JsonSettings
  BamV1         JsonBamV1
  BamV2         JsonBamV2
  Filters       []JsonFilter
}

// Used internally. Parses JSON source into intermediate structures.
func importJson(buffer []byte) (config *BamConfig, err error) {
  jsonGenerator := JsonGenerator{}
  err = json.Unmarshal(buffer, &jsonGenerator)
  if err != nil { return }

  config, err = processConfigJson(&jsonGenerator)
  return
}


// Used internally. Converts parsed JSON input into useful data types, taking defaults into account for omitted input.
func processConfigJson(input *JsonGenerator) (config *BamConfig, err error) {
  bam := make(BamConfig)
  config = &bam
  logging.Logln("Processing output settings")
  err = processConfigJsonOutput(input, config)
  if err != nil { return }
  logging.Logln("Processing input settings")
  err = processConfigJsonInput(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM settings")
  err = processConfigJsonSettings(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM V1 settings")
  err = processConfigJsonBamV1(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM V2 settings")
  err = processConfigJsonBamV2(input, config)
  if err != nil { return }
  logging.Logln("Processing filter settings")
  err = processConfigJsonFilters(input, config)
  return
}

// Used internally. Process "output" section.
func processConfigJsonOutput(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_OUTPUT] = make(BamMap)

  var intVal int64
  intVal = input.Output.Version
  if intVal < 1 || intVal > 2 { return fmt.Errorf("Output>Version: Invalid BAM version specified: %d", intVal) }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_VERSION] = Int{intVal}

  var textVal string
  textVal = fixPath(strings.TrimSpace(input.Output.File))
  if len(textVal) == 0 { textVal = "default.bam" }
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_PATH] = Text{textVal}

  textVal = fixPath(strings.TrimSpace(input.Output.PvrzPath))
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_PVRZ_PATH] = Text{textVal}

  return nil
}

// Used internally. Process "input" section.
func processConfigJsonInput(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_INPUT] = make(BamMap)

  static := input.Input.Static
  (*config)[SECTION_INPUT][KEY_INPUT_STATIC] = Bool{static}

  var size int
  size = len(input.Input.Files)
  textSeq := make([]string, size)
  for i := 0; i < size; i++ {
    textSeq[i] = strings.TrimSpace(input.Input.Files[i])
  }
  (*config)[SECTION_INPUT][KEY_INPUT_FILES] = TextArray{textSeq}

  var textVal string
  textVal = fixPath(strings.TrimSpace(input.Input.FileSequence.Path))
  if len(textVal) == 0 { textVal = "." }
  for len(textVal) > 1 && (textVal[len(textVal)-1:] == "/" || textVal[len(textVal)-1:] == "\\") { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_INPUT][KEY_INPUT_PATH] = Text{textVal}

  textVal = strings.TrimSpace(input.Input.FileSequence.Prefix)
  (*config)[SECTION_INPUT][KEY_INPUT_PREFIX] = Text{textVal}

  var intVal int64
  intVal = input.Input.FileSequence.SuffixStart
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_START] = Int{intVal}

  intVal = input.Input.FileSequence.SuffixEnd
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_END] = Int{intVal}

  intVal = input.Input.FileSequence.SuffixLength
  if intVal < 1 || intVal > 16 { return fmt.Errorf("Input>FileSequence>SuffixLength not in range [1,16]: %d", intVal) }
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_LEN] = Int{intVal}

  textVal = strings.TrimSpace(input.Input.FileSequence.Ext)
  for len(textVal) > 0 && textVal[0:1] == "." { textVal = textVal[1:] }
  (*config)[SECTION_INPUT][KEY_INPUT_EXT] = Text{textVal}

  size = len(input.Input.Search)
  textSeq = make([]string, size)
  for i := 0; i < size; i++ {
    textSeq[i] = strings.TrimSpace(input.Input.Search[i])
  }
  (*config)[SECTION_INPUT][KEY_INPUT_SEARCH] = TextArray{textSeq}

  return nil
}

// Used internally. Process "settings" section.
func processConfigJsonSettings(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_SETTINGS] = make(BamMap)

  var size int
  size = len(input.Settings.Center)
  intSeq := make([]int64, size)
  for i := 0; i < size; i++ {
    center := input.Settings.Center[i]
    intVal := int64(0)
    if len(center) > 1 { intVal |= center[1] & 0xffff }
    intVal <<= 16
    if len(center) > 0 { intVal |= center[0] & 0xffff }
    intSeq[i] = intVal
  }
  (*config)[SECTION_SETTINGS][KEY_CENTERS] = IntArray{intSeq}

  size = len(input.Settings.Sequence)
  intSeq2 := make([][]int64, size)
  for i := 0; i < size; i++ {
    intSeq2[i] = input.Settings.Sequence[i]
    if len(intSeq2[i]) == 0 { intSeq2[i] = []int64{0} }
  }
  (*config)[SECTION_SETTINGS][KEY_SEQUENCES] = IntMultiArray{intSeq2}

  return nil
}

// Used internally. Process "bamv1" section.
func processConfigJsonBamV1(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_BAMV1] = make(BamMap)

  var boolVal bool
  boolVal = input.BamV1.Compress
  (*config)[SECTION_BAMV1][KEY_V1_COMPRESS] = Bool{boolVal}

  var intVal int64
  intVal = input.BamV1.Rle
  (*config)[SECTION_BAMV1][KEY_V1_RLE] = Int{intVal}

  boolVal = input.BamV1.Alpha
  (*config)[SECTION_BAMV1][KEY_V1_ALPHA] = Bool{boolVal}

  intVal = input.BamV1.QualityMin
  if intVal < 0 || intVal > 100 { return fmt.Errorf("BamV1>QualityMin not in range [0, 100]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_QUALITY_MIN] = Int{intVal}

  intVal = input.BamV1.QualityMax
  if intVal < 0 || intVal > 100 { return fmt.Errorf("BamV1>QualityMax not in range [0, 100]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_QUALITY_MAX] = Int{intVal}

  intVal = input.BamV1.Speed
  if intVal < 1 || intVal > 10 { return fmt.Errorf("BamV1>Speed not in range [1, 10]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_SPEED] = Int{intVal}

  var floatVal float64
  floatVal = input.BamV1.Dither
  if floatVal < 0.0 || floatVal > 1.0 { return fmt.Errorf("BamV1>Dither not in range [0.0, 1.0]: %f", floatVal) }
  (*config)[SECTION_BAMV1][KEY_V1_DITHER] = Float{floatVal}

  var textVal string
  textVal = input.BamV1.TransColor
  reg := regexp.MustCompile("[ \t]*,[ \t]*")
  seq := reg.Split(textVal, -1)
  if len(seq) > 1 {
    // color sequence?
    intVal = 0
    if len(seq) > 3 { intVal |= tryParseUInt(seq[3], 255) & 0xff } else if len(seq) > 2 { intVal |= 0xff }
    intVal <<= 8
    if len(seq) > 2 { intVal |= tryParseUInt(seq[2], 0) & 0xff }
    intVal <<= 8
    if len(seq) > 1 { intVal |= tryParseUInt(seq[1], 0) & 0xff }
    intVal <<= 8
    intVal |= tryParseUInt(seq[0], 0) & 0xff
  } else {
    // color value?
    intVal = tryParseUInt(seq[0], 0xff00ff00)
  }
  (*config)[SECTION_BAMV1][KEY_V1_TRANS_COLOR] = Int{intVal}

  boolVal = input.BamV1.UseTransColor
  (*config)[SECTION_BAMV1][KEY_V1_USE_TRANS_COLOR] = Bool{boolVal}

  textVal = input.BamV1.SortBy
  if len(textVal) == 0 { textVal = "none" }
  (*config)[SECTION_BAMV1][KEY_V1_SORT_BY] = Text{textVal}

  boolVal = input.BamV1.UsePalette
  (*config)[SECTION_BAMV1][KEY_V1_USE_PALETTE] = Bool{boolVal}

  textVal = fixPath(strings.TrimSpace(input.BamV1.Palette))
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_BAMV1][KEY_V1_PALETTE] = Text{textVal}

  // Palette entry section may consist of single ARGB values or color sequences
  var size int
  size = len(input.BamV1.Color)
  intSeq := make([]int64, size)
  for i := 0; i < size; i++ {
    s := strings.ToLower(strings.TrimSpace(input.BamV1.Color[i]))
    reg := regexp.MustCompile("[ \t]*,[ \t]*")
    seq := reg.Split(s, -1)
    intVal = 0
    if len(seq) > 1 {
      // color sequence?
      if len(seq) > 3 { intVal |= tryParseUInt(seq[3], 255) & 0xff } else if len(seq) > 2 { intVal |= 0xff }
      intVal <<= 8
      if len(seq) > 2 { intVal |= tryParseUInt(seq[2], 0) & 0xff }
      intVal <<= 8
      if len(seq) > 1 { intVal |= tryParseUInt(seq[1], 0) & 0xff }
      intVal <<= 8
      intVal |= tryParseUInt(seq[0], 0) & 0xff
    } else if len(seq) > 0 {
      // color value?
      intVal = tryParseUInt(seq[0], 0)
    }
    intSeq[i] = intVal
  }
  (*config)[SECTION_BAMV1][KEY_V1_FIXED_COLORS] = IntArray{intSeq}

  return nil
}

// Used internally. Process "bamv2" section.
func processConfigJsonBamV2(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_BAMV2] = make(BamMap)

  var intVal int64
  intVal = input.BamV2.StartIndex
  if intVal < 0 || intVal > 99999 { return fmt.Errorf("BamV2>StartIndex not in range [0, 99999]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_START_INDEX] = Int{intVal}

  intVal = input.BamV2.Encoding
  if intVal < 0 || intVal > 3 { return fmt.Errorf("BamV2>Encoding not in range [0, 3]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_ENCODING] = Int{intVal}

  var floatVal float64
  floatVal = input.BamV2.Threshold
  if floatVal < 0.0 || floatVal > 100.0 { return fmt.Errorf("BamV2>Threshold not in range [0.0, 100.0]: %f", floatVal) }
  (*config)[SECTION_BAMV2][KEY_V2_THRESHOLD] = Float{floatVal}

  intVal = input.BamV2.Quality
  if intVal < 0 || intVal > 2 { return fmt.Errorf("BamV2>Quality not in range [0, 2]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_QUALITY] = Int{intVal}

  var boolVal bool
  boolVal = input.BamV2.WeightAlpha
  (*config)[SECTION_BAMV2][KEY_V2_WEIGHT_ALPHA] = Bool{boolVal}

  boolVal = input.BamV2.UseMetric
  (*config)[SECTION_BAMV2][KEY_V2_USE_METRIC] = Bool{boolVal}

  return nil
}

func processConfigJsonFilters(input *JsonGenerator, config *BamConfig) error {
  (*config)[SECTION_FILTERS] = make(BamMap)

  // process filters sequentially
  for index, filter := range input.Filters {
    f := Filter{ Name: filter.Name, Options: make(map[string]string) }
    for _, option := range filter.Options {
      f.Options[option.Key] = option.Value
    }
    (*config)[SECTION_FILTERS][strconv.Itoa(index)] = f
  }

  return nil
}
