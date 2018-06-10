package config
// Parse functionality for XML structures.

import (
  "encoding/xml"
  "fmt"
  "strconv"
  "strings"

  "github.com/InfinityTools/go-logging"
)

// Used internally by xml.Unmarshal to store output settings.
type XmlOutput struct {
  Version       string      `xml:"version"`
  Path          string      `xml:"file"`
  PvrzPath      string      `xml:"pvrzpath"`
}

// Used internally by xml.Unmarshal to store input file sequences settings.
type XmlInputSequence struct {
  Path          string      `xml:"path"`
  Prefix        string      `xml:"prefix"`
  SuffixStart   string      `xml:"suffixstart"`
  SuffixEnd     string      `xml:"suffixend"`
  SuffixLength  string      `xml:"suffixlength"`
  Ext           string      `xml:"ext"`
}

// Used internally by xml.Unmarshal to store input settings.
type XmlInput struct {
  Static        string            `xml:"static"`
  Sequence      XmlInputSequence  `xml:"filesequence"`
  Files         []string          `xml:"files>path"`
  Search        []string          `xml:"search>path"`
}

// Used internally by xml.Unmarshal to store bam settings settings.
type XmlSettings struct {
  Center        []string    `xml:"center"`
  Sequence      []string    `xml:"sequence"`
}

// Used internally by xml.Unmarshal to store bam v1 settings.
type XmlBamV1 struct {
  Compress      string      `xml:"compress"`
  Rle           string      `xml:"rle"`
  Alpha         string      `xml:"alpha"`
  QualityMin    string      `xml:"qualitymin"`
  QualityMax    string      `xml:"qualitymax"`
  Speed         string      `xml:"speed"`
  Dither        string      `xml:"dither"`
  FixedColors   []string    `xml:"color>entry"`
  TransColor    string      `xml:"transcolor"`
  UseTransColor string      `xml:"usetranscolor"`
  SortBy        string      `xml:"sortby"`
  UsePalette    string      `xml:"usepalette"`
  Palette       string      `xml:"palette"`
}

// Used internally by xml.Unmarshal to store bam v2 settings.
type XmlBamV2 struct {
  StartIndex    string      `xml:"startindex"`
  Encoding      string      `xml:"encoding"`
  Threshold     string      `xml:"threshold"`
  Quality       string      `xml:"quality"`
  WeightAlpha   string      `xml:"weightalpha"`
  UseMetric     string      `xml:"usemetric"`
}

// Used internally by xml.Unmarshal to store filter settings.
type XmlBamFilterOption struct {
  Key           string      `xml:"key"`
  Value         string      `xml:"value"`
}

// Used internally by xml.Unmarshal to store filter options.
type XmlBamFilter struct {
  Name          string                `xml:"name"`
  Options       []XmlBamFilterOption  `xml:"option"`
}

// Used internally by xml.Unmarshal to store configuration data from XML scripts.
type XmlGenerator struct {
  XMLName       xml.Name        `xml:"generator"`
  Output        XmlOutput       `xml:"output"`
  Input         XmlInput        `xml:"input"`
  Settings      XmlSettings     `xml:"settings"`
  BamV1         XmlBamV1        `xml:"bamv1"`
  BamV2         XmlBamV2        `xml:"bamv2"`
  Filters       []XmlBamFilter  `xml:"filters>filter"`
}


// Used internally. Parses XML source into intermediate structures.
func importXml(buffer []byte) (config *BamConfig, err error) {
  xmlGenerator := XmlGenerator{}
  err = xml.Unmarshal(buffer, &xmlGenerator)
  if err != nil { return }

  config, err = processConfigXml(&xmlGenerator)
  return
}


// Used internally. Converts parsed XML input into useful data types, taking defaults into account for omitted input.
func processConfigXml(input *XmlGenerator) (config *BamConfig, err error) {
  bam := make(BamConfig)
  config = &bam
  logging.Logln("Processing output settings")
  err = processConfigXmlOutput(input, config)
  if err != nil { return }
  logging.Logln("Processing input settings")
  err = processConfigXmlInput(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM settings")
  err = processConfigXmlSettings(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM V1 settings")
  err = processConfigXmlBamV1(input, config)
  if err != nil { return }
  logging.Logln("Processing BAM V2 settings")
  err = processConfigXmlBamV2(input, config)
  if err != nil { return }
  logging.Logln("Processing filter settings")
  err = processConfigXmlFilters(input, config)
  return
}

// Used internally. Process "output" section.
func processConfigXmlOutput(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_OUTPUT] = make(BamMap)

  var intVal int64
  intVal = tryParseInt(input.Output.Version, 1)
  if intVal < 1 || intVal > 2 { return fmt.Errorf("Output>Version: Invalid BAM version specified: %d", intVal) }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_VERSION] = Int{intVal}

  var textVal string
  textVal = fixPath(strings.TrimSpace(input.Output.Path))
  if len(textVal) == 0 { textVal = "default.bam" }
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_PATH] = Text{textVal}

  textVal = fixPath(strings.TrimSpace(input.Output.PvrzPath))
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_OUTPUT][KEY_OUTPUT_PVRZ_PATH] = Text{textVal}

  return nil
}

// Used internally. Process "input" section.
func processConfigXmlInput(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_INPUT] = make(BamMap)

  var static bool
  static = tryParseBool(input.Input.Static, true)
  (*config)[SECTION_INPUT][KEY_INPUT_STATIC] = Bool{static}

  var size int
  size = len(input.Input.Files)
  textSeq := make([]string, size)
  for i := 0; i < size; i++ {
    textSeq[i] = strings.TrimSpace(input.Input.Files[i])
  }
  (*config)[SECTION_INPUT][KEY_INPUT_FILES] = TextArray{textSeq}

  var textVal string
  textVal = fixPath(strings.TrimSpace(input.Input.Sequence.Path))
  if len(textVal) == 0 { textVal = "." }
  for len(textVal) > 1 && (textVal[len(textVal)-1:] == "/" || textVal[len(textVal)-1:] == "\\") { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_INPUT][KEY_INPUT_PATH] = Text{textVal}

  textVal = strings.TrimSpace(input.Input.Sequence.Prefix)
  (*config)[SECTION_INPUT][KEY_INPUT_PREFIX] = Text{textVal}

  var intVal int64
  intVal = tryParseInt(input.Input.Sequence.SuffixStart, 0)
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_START] = Int{intVal}

  intVal = tryParseInt(input.Input.Sequence.SuffixEnd, 0)
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_END] = Int{intVal}

  intVal = tryParseInt(input.Input.Sequence.SuffixLength, 1)
  if intVal < 1 || intVal > 16 { return fmt.Errorf("Input>FileSequence>SuffixLength not in range [1,16]: %d", intVal) }
  (*config)[SECTION_INPUT][KEY_INPUT_SUFFIX_LEN] = Int{intVal}

  textVal = strings.TrimSpace(input.Input.Sequence.Ext)
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
func processConfigXmlSettings(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_SETTINGS] = make(BamMap)

  var size int
  size = len(input.Settings.Center)
  intSeq := make([]int64, size)
  for i := 0; i < size; i++ {
    intVal := int64(0)
    seq := tryParseIntSeq(input.Settings.Center[i], 0)
    if len(seq) > 1 { intVal |= seq[1] & 0xffff }
    intVal <<= 16
    if len(seq) > 0 { intVal |= seq[0] & 0xffff }
    intSeq[i] = intVal
  }
  (*config)[SECTION_SETTINGS][KEY_CENTERS] = IntArray{intSeq}

  size = len(input.Settings.Sequence)
  intSeq2 := make([][]int64, size)
  for i := 0; i < size; i++ {
    seq := tryParseUIntSeq(input.Settings.Sequence[i], 0)
    if len(seq) == 0 { seq = []int64{0} }
    intSeq2[i] = seq
  }
  (*config)[SECTION_SETTINGS][KEY_SEQUENCES] = IntMultiArray{intSeq2}

  return nil
}

// Used internally. Process "bamv1" section.
func processConfigXmlBamV1(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_BAMV1] = make(BamMap)

  var boolVal bool
  boolVal = tryParseBool(input.BamV1.Compress, true)
  (*config)[SECTION_BAMV1][KEY_V1_COMPRESS] = Bool{boolVal}

  var intVal int64
  intVal = tryParseInt(input.BamV1.Rle, -1)
  (*config)[SECTION_BAMV1][KEY_V1_RLE] = Int{intVal}

  boolVal = tryParseBool(input.BamV1.Alpha, true)
  (*config)[SECTION_BAMV1][KEY_V1_ALPHA] = Bool{boolVal}

  intVal = tryParseInt(input.BamV1.QualityMin, 80)
  if intVal < 0 || intVal > 100 { return fmt.Errorf("BamV1>QualityMin not in range [0, 100]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_QUALITY_MIN] = Int{intVal}

  intVal = tryParseInt(input.BamV1.QualityMax, 100)
  if intVal < 0 || intVal > 100 { return fmt.Errorf("BamV1>QualityMax not in range [0, 100]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_QUALITY_MAX] = Int{intVal}

  intVal = tryParseInt(input.BamV1.Speed, 3)
  if intVal < 1 || intVal > 10 { return fmt.Errorf("BamV1>Speed not in range [1, 10]: %d", intVal) }
  (*config)[SECTION_BAMV1][KEY_V1_SPEED] = Int{intVal}

  var floatVal float64
  floatVal = tryParseFloat(input.BamV1.Dither, 0.0)
  if floatVal < 0.0 || floatVal > 1.0 { return fmt.Errorf("BamV1>Dither not in range [0.0, 1.0]: %f", floatVal) }
  (*config)[SECTION_BAMV1][KEY_V1_DITHER] = Float{floatVal}

  var textVal string
  textVal = input.BamV1.TransColor
  if strings.Index(textVal, ",") >= 0 {
    // color sequence?
    intVal = 0
    seq := tryParseUIntSeq(textVal, 0)
    if len(seq) > 3 { intVal |= seq[3] & 0xff } else { intVal |= 0xff } // assume opaque color
    intVal <<= 8
    if len(seq) > 2 { intVal |= seq[2] & 0xff }
    intVal <<= 8
    if len(seq) > 1 { intVal |= seq[1] & 0xff }
    intVal <<= 8
    if len(seq) > 0 { intVal |= seq[0] & 0xff }
  } else {
    // color value?
    intVal = tryParseUInt(textVal, 0xff00ff00)
  }
  (*config)[SECTION_BAMV1][KEY_V1_TRANS_COLOR] = Int{intVal}

  boolVal = tryParseBool(input.BamV1.UseTransColor, false)
  (*config)[SECTION_BAMV1][KEY_V1_USE_TRANS_COLOR] = Bool{boolVal}

  textVal = input.BamV1.SortBy
  if len(textVal) == 0 { textVal = "none" }
  (*config)[SECTION_BAMV1][KEY_V1_SORT_BY] = Text{textVal}

  boolVal = tryParseBool(input.BamV1.UsePalette, false)
  (*config)[SECTION_BAMV1][KEY_V1_USE_PALETTE] = Bool{boolVal}

  textVal = fixPath(strings.TrimSpace(input.BamV1.Palette))
  for len(textVal) > 1 && textVal[len(textVal)-1:] == "/" { textVal = textVal[:len(textVal)-1] }
  (*config)[SECTION_BAMV1][KEY_V1_PALETTE] = Text{textVal}

  // Palette entry section may consist of single ARGB values or color sequences
  var size int
  size = len(input.BamV1.FixedColors)
  intSeq := make([]int64, size)
  for i := 0; i < size; i++ {
    s := input.BamV1.FixedColors[i]
    intVal = 0
    if strings.Index(s, ",") >= 0 {
      // color sequence?
      seq := tryParseUIntSeq(s, 0)
      if len(seq) > 3 { intVal |= seq[3] & 0xff } else { intVal |= 0xff } // assume opaque color
      intVal <<= 8
      if len(seq) > 2 { intVal |= seq[2] & 0xff }
      intVal <<= 8
      if len(seq) > 1 { intVal |= seq[1] & 0xff }
      intVal <<= 8
      if len(seq) > 0 { intVal |= seq[0] & 0xff }
    } else {
      // color value?
      intVal = tryParseUInt(s, 0)
    }
    intSeq[i] = intVal
  }
  (*config)[SECTION_BAMV1][KEY_V1_FIXED_COLORS] = IntArray{intSeq}

  return nil
}

// Used internally. Process "bamv2" section.
func processConfigXmlBamV2(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_BAMV2] = make(BamMap)

  var intVal int64
  intVal = tryParseInt(input.BamV2.StartIndex, 1000)
  if intVal < 0 || intVal > 99999 { return fmt.Errorf("BamV2>StartIndex not in range [0, 99999]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_START_INDEX] = Int{intVal}

  intVal = tryParseInt(input.BamV2.Encoding, 0)
  if intVal < 0 || intVal > 3 { return fmt.Errorf("BamV2>Encoding not in range [0, 3]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_ENCODING] = Int{intVal}

  var floatVal float64
  floatVal = tryParseFloat(input.BamV2.Threshold, 5.0)
  if floatVal < 0.0 || floatVal > 100.0 { return fmt.Errorf("BamV2>Threshold not in range [0.0, 100.0]: %f", floatVal) }
  (*config)[SECTION_BAMV2][KEY_V2_THRESHOLD] = Float{floatVal}

  intVal = tryParseInt(input.BamV2.Quality, 1)
  if intVal < 0 || intVal > 2 { return fmt.Errorf("BamV2>Quality not in range [0, 2]: %d", intVal) }
  (*config)[SECTION_BAMV2][KEY_V2_QUALITY] = Int{intVal}

  var boolVal bool
  boolVal = tryParseBool(input.BamV2.WeightAlpha, true)
  (*config)[SECTION_BAMV2][KEY_V2_WEIGHT_ALPHA] = Bool{boolVal}

  boolVal = tryParseBool(input.BamV2.UseMetric, false)
  (*config)[SECTION_BAMV2][KEY_V2_USE_METRIC] = Bool{boolVal}

  return nil
}


func processConfigXmlFilters(input *XmlGenerator, config *BamConfig) error {
  (*config)[SECTION_FILTERS] = make(BamMap)

  // process filters sequentially
  for index, filter := range input.Filters {
    f := Filter{ Name: filter.Name, Options: make(map[string]string) }
    for i := 0; i < len(filter.Options); i++ {
      key, value := strings.TrimSpace(filter.Options[i].Key), strings.TrimSpace(filter.Options[i].Value)
      f.Options[key] = value
    }
    (*config)[SECTION_FILTERS][strconv.Itoa(index)] = f
  }

  return nil
}
