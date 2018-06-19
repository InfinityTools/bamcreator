/*
Package config translates BAM generation configurations from XML or JSON structures into a preprocessed map structure
for quick access.

BAM Creator is released under the BSD 2-clause license. See LICENSE in the project's root folder for more details.
*/
package config

import (
  "bytes"
  "errors"
  "io"
  "strconv"
  "strings"

  "github.com/InfinityTools/go-logging"
)


// Available BAM configuration section names
const (
  SECTION_OUTPUT    = "output"
  SECTION_INPUT     = "input"
  SECTION_SETTINGS  = "settings"
  SECTION_BAMV1     = "bamv1"
  SECTION_BAMV2     = "bamv2"
  SECTION_FILTERS   = "filters"
)

// Available BAM configuration key names
const (
  KEY_OUTPUT_VERSION      = "output_version"
  KEY_OUTPUT_PATH         = "output_path"
  KEY_OUTPUT_PVRZ_PATH    = "output_pvrz_path"
  KEY_INPUT_STATIC        = "input_static"
  KEY_INPUT_PATH          = "input_path"
  KEY_INPUT_PREFIX        = "input_prefix"
  KEY_INPUT_SUFFIX_START  = "input_suffix_start"
  KEY_INPUT_SUFFIX_END    = "input_suffix_end"
  KEY_INPUT_SUFFIX_LEN    = "input_suffix_len"
  KEY_INPUT_EXT           = "input_ext"
  KEY_INPUT_FILES         = "input_files"
  KEY_INPUT_SEARCH        = "input_search"
  KEY_CENTERS             = "center"
  KEY_SEQUENCES           = "sequence"
  KEY_V1_COMPRESS         = "v1_compress"
  KEY_V1_RLE              = "v1_rle"
  KEY_V1_ALPHA            = "v1_alpha"
  KEY_V1_QUALITY_MIN      = "v1_quality_min"
  KEY_V1_QUALITY_MAX      = "v1_quality_max"
  KEY_V1_SPEED            = "v1_speed"
  KEY_V1_DITHER           = "v1_dither"
  KEY_V1_FIXED_COLORS     = "v1_fixed_colors"
  KEY_V1_TRANS_COLOR      = "v1_trans_color"
  KEY_V1_USE_TRANS_COLOR  = "v1_use_trans_color"
  KEY_V1_SORT_BY          = "v1_sort_by"
  KEY_V1_USE_PALETTE      = "v1_use_palette"
  KEY_V1_PALETTE          = "v1_palette"
  KEY_V2_START_INDEX      = "v2_start_index"
  KEY_V2_ENCODING         = "v2_encoding"
  KEY_V2_THRESHOLD        = "v2_threshold"
  KEY_V2_QUALITY          = "v2_quality"
  KEY_V2_WEIGHT_ALPHA     = "v2_weight_alpha"
  KEY_V2_USE_METRIC       = "v2_use_metric"
  KEY_FILTERS             = "filter"
)

// Internally used to determine assigned value type.
type ID int

// BamMap maps key => value associations.
type BamMap map[string]Variant

// BamConfig maps section => key => value.
type BamConfig map[string]BamMap


// ImportConfig constructs a BamConfig object from configuration data found in the source wrapped by the Reader object.
func ImportConfig(r io.Reader) (config *BamConfig, err error) {
  // reading XML data into byte buffer
  logging.Logln("Loading configuration data")
  buffer := make([]byte, 1024)
  totalRead := 0
  for {
    bytesRead, err := r.Read(buffer[totalRead:]);
    if err != nil { break }
    totalRead += bytesRead
    if totalRead == len(buffer) {
      buffer = append(buffer, make([]byte, len(buffer))...)
    }
  }
  if err != nil && err != io.EOF { return }
  if totalRead < len(buffer) {
    buffer = buffer[:totalRead]
  }

  // try to determine input format
  isXml := true
  ofs := 0
  whiteSpace := []byte{0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x20}
  for ofs < len(buffer) {
    if bytes.IndexByte(whiteSpace, buffer[ofs]) < 0 {
      if buffer[ofs] == '<' {
        isXml = true
      } else if buffer[ofs] == '{' {
        isXml = false
      } else {
        err = errors.New("Configuration: Unrecognized format")
      }
      break
    }
    ofs++
  }
  if err != nil { return }

  // parsing source into intermediate structures
  if isXml {
    config, err = importXml(buffer)
  } else {
    config, err = importJson(buffer)
  }
  if err != nil { return }

  logging.Logln("Finished loading configuration data")
  return
}

// GetConfigValueBool returns the boolean value assigned to the specified section => key location. ok returns whether
// the value is available.
func (bam *BamConfig) GetConfigValueBool(section, key string) (retVal bool, ok bool) {
  value, ok := (*bam)[section][key].(VarBool)
  if !ok { return }
  retVal = value.ToBool()
  return
}

// GetConfigValueInt returns the numeric value assigned to the specified section => key location. ok returns whether
// the value is available.
func (bam *BamConfig) GetConfigValueInt(section, key string) (retVal int64, ok bool) {
  value, ok := (*bam)[section][key].(VarInt)
  if !ok { return }
  retVal = value.ToInt()
  return
}

// GetConfigValueFloat returns the floating point value assigned to the specified section => key location. ok returns
// whether the value is available.
func (bam *BamConfig) GetConfigValueFloat(section, key string) (retVal float64, ok bool) {
  value, ok := (*bam)[section][key].(VarFloat)
  if !ok { return }
  retVal = value.ToFloat()
  return
}

// GetConfigValueText returns the string value assigned to the specified section => key location. ok returns whether
// the value is available.
func (bam *BamConfig) GetConfigValueText(section, key string) (retVal string, ok bool) {
  value, ok := (*bam)[section][key].(Variant)
  if !ok { return }
  retVal = value.ToString()
  return
}

// GetConfigValueIntSeq returns the numeric array assigned to the specified section => key location. ok returns whether
// the value is available.
func (bam *BamConfig) GetConfigValueIntSeq(section, key string) (retVal []int64, ok bool) {
  value, ok := (*bam)[section][key].(VarIntArray)
  if !ok { return }
  retVal = value.ToIntArray()
  return
}

// GetConfigValueIntSeq2 returns the two-dimensional numeric array assigned to the specified section => key location.
// ok returns whether the value is available.
func (bam *BamConfig) GetConfigValueIntSeq2(section, key string) (retVal [][]int64, ok bool) {
  value, ok := (*bam)[section][key].(VarIntMultiArray)
  if !ok { return }
  retVal = value.ToIntMultiArray()
  return
}

// GetConfigValueFloatSeq returns the array of floating point values assigned to the specified section => key location.
// ok returns whether the value is available.
func (bam *BamConfig) GetConfigValueFloatSeq(section, key string) (retVal []float64, ok bool) {
  value, ok := (*bam)[section][key].(VarFloatArray)
  if !ok { return }
  retVal = value.ToFloatArray()
  return
}

// GetConfigValueTextSeq returns the array of strings assigned to the specified section => key location. ok returns
// whether the value is available.
func (bam *BamConfig) GetConfigValueTextSeq(section, key string) (retVal []string, ok bool) {
  value, ok := (*bam)[section][key].(VarTextArray)
  if !ok { return }
  retVal = value.ToTextArray()
  return
}

// GetConfigFilterLength returns the number of available filter definitions.
func (bam *BamConfig) GetConfigFilterLength() int {
  return len((*bam)[SECTION_FILTERS])
}

// GetConfigFilterName returns the name of the filter at the specified index. ok returns whether the filter is available.
func (bam *BamConfig) GetConfigFilterName(index int) (retVal string, ok bool) {
  var option VarFilterMap
  if option, ok = (*bam)[SECTION_FILTERS][strconv.Itoa(index)].(VarFilterMap); ok {
    retVal = option.GetName()
  }
  return
}

// GetConfigFilterOptions returns the options of the specified filter as multi-array. First item of each entry contains
// key, second item contains value. ok returns whether the filter is available.
func (bam *BamConfig) GetConfigFilterOptions(index int) (retVal [][]string, ok bool) {
  var filter VarFilterMap
  if filter, ok = (*bam)[SECTION_FILTERS][strconv.Itoa(index)].(VarFilterMap); ok {
    retVal = filter.GetOptions()
  } else {
    retVal = make([][]string, 0)
  }
  return
}


// Used internally. Attempts to convert the content of s into a boolean value. Failing that the function will return
// the specified default value. Both numeric (decimal/hexadecimal) and true/false string values are detected.
func tryParseBool(s string, defValue bool) bool {
  // try true/false first
  if strings.ToLower(s) == "true" {
    return true
  } else if strings.ToLower(s) == "false" {
    return false
  }
  // try numeric value second
  def := 0
  if defValue { def = 1 }
  return (tryParseInt(s, def) != 0)
}

// Used internally. Attempts to convert the content of s into a signed numeric value. Failing that the function will
// return the specified default value. Both decimal and hexadecimal (with prefix "0x") are detected.
func tryParseInt(s string, defValue int) int64 {
  s = strings.ToLower(strings.TrimSpace(s))

  var value int64
  var err error
  if len(s) > 2 && s[:2] == "0x" {
    // hex value?
    value, err = strconv.ParseInt(s[2:], 16, 32)
  } else {
    // dec value?
    value, err = strconv.ParseInt(s, 10, 32)
  }
  if err != nil { value = int64(defValue) }

  return value
}

// Used internally. Attempts to convert the content of s into an unsigned numeric value. Failing that the function
// will return the specified default value. Both decimal and hexadecimal (with prefix "0x") are detected.
func tryParseUInt(s string, defValue uint) int64 {
  s = strings.ToLower(strings.TrimSpace(s))

  var value uint64
  var err error
  if len(s) > 2 && s[:2] == "0x" {
    // hex value?
    value, err = strconv.ParseUint(s[2:], 16, 32)
  } else {
    // dec value?
    value, err = strconv.ParseUint(s, 10, 32)
  }
  if err != nil { value = uint64(defValue) }

  return int64(value)
}

// Used internally. Attempts to convert the content of s into a floating point value. Failing that the function will
// return the specified default value.
func tryParseFloat(s string, defValue float64) float64 {
  s = strings.ToLower(strings.TrimSpace(s))

  var value float64
  var err error
  value, err = strconv.ParseFloat(s, 64)
  if err != nil { value = defValue }

  return value
}

// Used internally. Attempts to convert the content of s into a sequence of signed numeric values. Invalid elements
// will be replaced by the provided default value. The returned array may contain zero, one or more items.
func tryParseIntSeq(s string, defValue int) []int64 {
  items := strings.Split(s, ",")
  retVal := make([]int64, len(items))
  for idx, val := range items {
    retVal[idx] = tryParseInt(val, defValue)
  }

  return retVal
}

// Used internally. Attempts to convert the content of s into a sequence of unsigned numeric values. Invalid elements
// will be replaced by the provided default value. The returned array may contain zero, one or more items.
func tryParseUIntSeq(s string, defValue uint) []int64 {
  items := strings.Split(s, ",")
  retVal := make([]int64, len(items))
  for idx, val := range items {
    retVal[idx] = tryParseUInt(val, defValue)
  }

  return retVal
}

// Used internally. Attempts to convert the content of s into a sequence of floating point values. Invalid elements
// will be replaced by the provided default value. The returned array may contain zero, one or more items.
func tryParseFloatSeq(s string, defValue float64) []float64 {
  items := strings.Split(s, ",")
  retVal := make([]float64, len(items))
  for idx, val := range items {
    retVal[idx] = tryParseFloat(val, defValue)
  }

  return retVal
}

// Used internally. Attempts to convert the content of s into a sequence of string values. The returned array may
// contain zero, one or more items.
func tryParseTextSeq(s string) []string {
  items := strings.Split(s, ",")
  retVal := make([]string, len(items))
  for idx, val := range items {
    retVal[idx] = strings.TrimSpace(val)
  }

  return retVal
}

// Used internally. Fixes Windows-specific path separater characters.
func fixPath(s string) string {
  if PATH_SEPARATOR == "\\" {
    s = strings.Replace(s, PATH_SEPARATOR, "/", -1)
  }
  return s
}
