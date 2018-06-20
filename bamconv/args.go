package main
// Handles command line arguments for bamconv.

import (
  "errors"
  "fmt"
  "os"
  "strings"

  "github.com/InfinityTools/go-cmdargs"
  "github.com/InfinityTools/go-logging"
)

const (
  CMDOPT_HELP = "help"
  CMDOPT_VERSION = "version"
  CMDOPT_VERBOSE = "verbose"
  CMDOPT_SILENT = "silent"
  CMDOPT_LOG_STYLE = "log-style"
  CMDOPT_COMPACT = "compact"
  CMDOPT_OUTPUT_TYPE = "output-type"
  CMDOPT_OUTPUT = "output"
  CMDOPT_BAM_VERSION = "bam-version"
  CMDOPT_BAM_OUTPUT = "bam-output"
  CMDOPT_BAM_PVRZ_PATH = "bam-pvrz-path"
  CMDOPT_BAM_SEARCH_PATH = "bam-search-path"
  CMDOPT_BAM_CENTER = "bam-center"
  CMDOPT_BAMV1_COMPRESS = "bamv1-compress"
  CMDOPT_BAMV1_NO_COMPRESS = "bamv1-no-compress"
  CMDOPT_BAMV1_RLE = "bamv1-rle"
  CMDOPT_BAMV1_ALPHA = "bamv1-alpha"
  CMDOPT_BAMV1_NO_ALPHA = "bamv1-no-alpha"
  CMDOPT_BAMV1_QUALITY_MIN = "bamv1-quality-min"
  CMDOPT_BAMV1_QUALITY_MAX = "bamv1-quality-max"
  CMDOPT_BAMV1_SPEED = "bamv1-speed"
  CMDOPT_BAMV1_DITHER = "bamv1-dither"
  CMDOPT_BAMV1_SORT_BY = "bamv1-sort"
  CMDOPT_BAMV1_PALETTE = "bamv1-palette"
  CMDOPT_BAMV2_START_INDEX = "bamv2-start-index"
  CMDOPT_BAMV2_ENCODING = "bamv2-encoding"
  CMDOPT_BAMV2_THRESHOLD = "bamv2-threshold"
  CMDOPT_BAMV2_QUALITY = "bamv2-quality"
  CMDOPT_BAMV2_WEIGHT_ALPHA = "bamv2-weight-alpha"
  CMDOPT_BAMV2_NO_WEIGHT_ALPHA = "bamv2-no-weight-alpha"
  CMDOPT_BAMV2_USE_METRIC = "bamv2-use-metric"
  CMDOPT_BAMV2_NO_USE_METRIC = "bamv2-no-use-metric"
  CMDOPT_FILTER = "filter"
)

type OptBool struct { value bool; set bool }
type OptInt struct { value int; set bool }
type OptFloat struct { value float32; set bool }
type OptText struct { value string; set bool }
type OptFilterOption struct { key, value string }
type OptFilter struct { name string; options []OptFilterOption }

type CmdOptions struct {
  help                OptBool
  version             OptBool
  verbose             OptBool
  logStyle            OptBool
  compact             OptBool
  outputType          OptText
  output              OptText
  bamVersion          OptInt
  bamOutput           OptText
  bamPvrzPath         OptText
  bamSearchPath       OptText
  bamCenter           OptBool
  bamv1Compress       OptBool
  bamv1Rle            OptInt
  bamv1Alpha          OptBool
  bamv1QualityMin     OptInt
  bamv1QualityMax     OptInt
  bamv1Speed          OptInt
  bamv1Dither         OptFloat
  bamv1SortBy         OptText
  bamv1Palette        OptText
  bamv2StartIndex     OptInt
  bamv2Encoding       OptInt
  bamv2Threshold      OptFloat
  bamv2Quality        OptInt
  bamv2WeightAlpha    OptBool
  bamv2UseMetric      OptBool
  filters             []OptFilter
  optionsLength       int
  argSelf             string
  argsExtra           []string
}

var cmdOptions  CmdOptions


func loadArgs(args []string) error {
  params := cmdargs.Create()
  params.AddParameter(CMDOPT_HELP, nil, 0)
  params.AddParameter(CMDOPT_VERSION, nil, 0)
  params.AddParameter(CMDOPT_VERBOSE, nil, 0)
  params.AddParameter(CMDOPT_SILENT, nil, 0)
  params.AddParameter(CMDOPT_LOG_STYLE, nil, 0)
  params.AddParameter(CMDOPT_COMPACT, nil, 0)
  params.AddParameter(CMDOPT_OUTPUT_TYPE, nil, 1)
  params.AddParameter(CMDOPT_OUTPUT, nil, 1)
  params.AddParameter(CMDOPT_BAM_VERSION, nil, 1)
  params.AddParameter(CMDOPT_BAM_OUTPUT, nil, 1)
  params.AddParameter(CMDOPT_BAM_PVRZ_PATH, nil, 1)
  params.AddParameter(CMDOPT_BAM_SEARCH_PATH, nil, 1)
  params.AddParameter(CMDOPT_BAM_CENTER, nil, 0)
  params.AddParameter(CMDOPT_BAMV1_COMPRESS, nil, 0)
  params.AddParameter(CMDOPT_BAMV1_NO_COMPRESS, nil, 0)
  params.AddParameter(CMDOPT_BAMV1_RLE, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_ALPHA, nil, 0)
  params.AddParameter(CMDOPT_BAMV1_NO_ALPHA, nil, 0)
  params.AddParameter(CMDOPT_BAMV1_QUALITY_MIN, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_QUALITY_MAX, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_SPEED, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_DITHER, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_SORT_BY, nil, 1)
  params.AddParameter(CMDOPT_BAMV1_PALETTE, nil, 1)
  params.AddParameter(CMDOPT_BAMV2_START_INDEX, nil, 1)
  params.AddParameter(CMDOPT_BAMV2_ENCODING, nil, 1)
  params.AddParameter(CMDOPT_BAMV2_THRESHOLD, nil, 1)
  params.AddParameter(CMDOPT_BAMV2_QUALITY, nil, 1)
  params.AddParameter(CMDOPT_BAMV2_WEIGHT_ALPHA, nil, 0)
  params.AddParameter(CMDOPT_BAMV2_NO_WEIGHT_ALPHA, nil, 0)
  params.AddParameter(CMDOPT_BAMV2_USE_METRIC, nil, 0)
  params.AddParameter(CMDOPT_BAMV2_NO_USE_METRIC, nil, 0)
  params.AddParameter(CMDOPT_FILTER, nil, 1)

  err := params.Evaluate(args)
  if err != nil { return err }

  // validating extra arguments
  cmdOptions.argSelf = params.GetArgSelf()
  cmdOptions.argsExtra = make([]string, 0)
  for i := 0; i < params.GetArgExtraLength(); i++ {
    s := params.GetArgExtra(i).ToString()
    // Expanding wildcard
    expanded := params.GetExpandedArgExtra(i)
    if len(expanded) == 0 { expanded = []string{s} }  // falling back to check directly
    for _, name := range expanded {
      fi, err := os.Stat(name)
      if err != nil { return fmt.Errorf("BAM file at %d: %v", len(cmdOptions.argsExtra), err) }
      if !fi.Mode().IsRegular() { return fmt.Errorf("BAM file does not exist: %q", name) }
      cmdOptions.argsExtra = append(cmdOptions.argsExtra, name)
    }
  }

  // validating options
  cmdOptions.filters = make([]OptFilter, 0)
  cmdOptions.optionsLength = 0
  for idx := 0; idx < params.GetArgLength(); idx++ {
    arg, err := params.GetArgAt(idx)
    if err != nil {
      logging.Warnf("Could not parse command line option at index %d. Skipping...\n", idx)
      continue
    }
    switch arg.Name {
      case CMDOPT_HELP:
        if !cmdOptions.help.set { cmdOptions.optionsLength++ }
        cmdOptions.help = OptBool{true, true}
        return nil
      case CMDOPT_VERSION:
        if !cmdOptions.version.set { cmdOptions.optionsLength++ }
        cmdOptions.version = OptBool{true, true}
        return nil
      case CMDOPT_VERBOSE:
        if !cmdOptions.verbose.set { cmdOptions.optionsLength++ }
        cmdOptions.verbose = OptBool{true, true}
      case CMDOPT_SILENT:
        if !cmdOptions.verbose.set { cmdOptions.optionsLength++ }
        cmdOptions.verbose = OptBool{false, true}
      case CMDOPT_LOG_STYLE:
        if !cmdOptions.logStyle.set { cmdOptions.optionsLength++ }
        cmdOptions.logStyle = OptBool{true, true}
      case CMDOPT_COMPACT:
        if !cmdOptions.compact.set { cmdOptions.optionsLength++ }
        cmdOptions.compact = OptBool{true, true}
      case CMDOPT_OUTPUT_TYPE:
        if !cmdOptions.outputType.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          s := strings.ToLower(arg.Arguments[0].ToString())
          switch s {
            case "xml", "json":
            default:
              return fmt.Errorf("Option %q: Unrecognized output type %q", arg.Name, arg.Arguments[0].ToString())
          }
          cmdOptions.outputType = OptText{s, true}
        }
      case CMDOPT_OUTPUT:
        if !cmdOptions.output.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          s := arg.Arguments[0].ToString()
          if len(s) == 0 { return fmt.Errorf("Option %q: No configuration output file specified", arg.Name) }
          cmdOptions.output = OptText{s, true}
        }
      case CMDOPT_BAM_VERSION:
        if !cmdOptions.bamVersion.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && (i == 1 || i == 2) {
            cmdOptions.bamVersion = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAM_OUTPUT:
        if !cmdOptions.bamOutput.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          s := arg.Arguments[0].ToString()
          if len(s) == 0 { return fmt.Errorf("Option %q: No BAM output file specified", arg.Name) }
          cmdOptions.bamOutput = OptText{s, true}
        }
      case CMDOPT_BAM_PVRZ_PATH:
        if !cmdOptions.bamPvrzPath.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          cmdOptions.bamPvrzPath = OptText{arg.Arguments[0].ToString(), true}
        }
      case CMDOPT_BAM_SEARCH_PATH:
        if !cmdOptions.bamSearchPath.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          cmdOptions.bamSearchPath = OptText{arg.Arguments[0].ToString(), true}
        }
      case CMDOPT_BAM_CENTER:
        if !cmdOptions.bamCenter.set { cmdOptions.optionsLength++ }
        cmdOptions.bamCenter = OptBool{true, true}
      case CMDOPT_BAMV1_COMPRESS:
        if !cmdOptions.bamv1Compress.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv1Compress = OptBool{true, true}
      case CMDOPT_BAMV1_NO_COMPRESS:
        if !cmdOptions.bamv1Compress.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv1Compress = OptBool{false, true}
      case CMDOPT_BAMV1_RLE:
        if !cmdOptions.bamv1Rle.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= -1 && i <= 1 {
            cmdOptions.bamv1Rle = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV1_ALPHA:
        if !cmdOptions.bamv1Alpha.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv1Alpha = OptBool{true, true}
      case CMDOPT_BAMV1_NO_ALPHA:
        if !cmdOptions.bamv1Alpha.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv1Alpha = OptBool{false, true}
      case CMDOPT_BAMV1_QUALITY_MIN:
        if !cmdOptions.bamv1QualityMin.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 0 && i <= 100 {
            cmdOptions.bamv1QualityMin = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV1_QUALITY_MAX:
        if !cmdOptions.bamv1QualityMax.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 0 && i <= 100 {
            cmdOptions.bamv1QualityMax = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV1_SPEED:
        if !cmdOptions.bamv1Speed.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 1 && i <= 10 {
            cmdOptions.bamv1Speed = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV1_DITHER:
        if !cmdOptions.bamv1Dither.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if f, x := arg.Arguments[0].Float(); x && f >= 0.0 && f <= 1.0 {
            cmdOptions.bamv1Dither = OptFloat{float32(f), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV1_SORT_BY:
        if !cmdOptions.bamv1SortBy.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          cmdOptions.bamv1SortBy = OptText{arg.Arguments[0].ToString(), true}
        }
      case CMDOPT_BAMV1_PALETTE:
        if !cmdOptions.bamv1Palette.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          cmdOptions.bamv1Palette = OptText{arg.Arguments[0].ToString(), true}
        }
      case CMDOPT_BAMV2_START_INDEX:
        if !cmdOptions.bamv2StartIndex.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 0 && i <= 99999 {
            cmdOptions.bamv2StartIndex = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV2_ENCODING:
        if !cmdOptions.bamv2Encoding.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 0 && i <= 3 {
            cmdOptions.bamv2Encoding = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV2_THRESHOLD:
        if !cmdOptions.bamv2Threshold.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if f, x := arg.Arguments[0].Float(); x && f >= 0.0 && f <= 100.0 {
            cmdOptions.bamv2Threshold = OptFloat{float32(f), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV2_QUALITY:
        if !cmdOptions.bamv2Quality.set { cmdOptions.optionsLength++ }
        if len(arg.Arguments) > 0 {
          if i, x := arg.Arguments[0].Int(); x && i >= 0 && i <= 2 {
            cmdOptions.bamv2Quality = OptInt{int(i), true}
          } else {
            return fmt.Errorf("Option %q: Invalid argument %v", arg.Name, arg.Arguments[0])
          }
        }
      case CMDOPT_BAMV2_WEIGHT_ALPHA:
        if !cmdOptions.bamv2WeightAlpha.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv2WeightAlpha = OptBool{true, true}
      case CMDOPT_BAMV2_NO_WEIGHT_ALPHA:
        if !cmdOptions.bamv2WeightAlpha.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv2WeightAlpha = OptBool{false, true}
      case CMDOPT_BAMV2_USE_METRIC:
        if !cmdOptions.bamv2UseMetric.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv2UseMetric = OptBool{true, true}
      case CMDOPT_BAMV2_NO_USE_METRIC:
        if !cmdOptions.bamv2UseMetric.set { cmdOptions.optionsLength++ }
        cmdOptions.bamv2UseMetric = OptBool{false, true}
      case CMDOPT_FILTER:
        if len(arg.Arguments) > 0 {
          cmdOptions.optionsLength++
          filter, err := parseFilter(arg.Arguments[0].ToString())
          if err != nil { return fmt.Errorf("Option %q: Invalid filter argument %v (%v)", arg.Name, arg.Arguments[0], err) }
          cmdOptions.filters = append(cmdOptions.filters, filter)
        }
      default:
        return fmt.Errorf("Unrecognized option: %q", arg.Name)
    }
  }

  // Invalid combination: Options, but no BAM files
  if len(cmdOptions.argsExtra) == 0 && cmdOptions.optionsLength > 0 {
    return errors.New("No BAM file specified")
  }

  return nil
}


func argsExtraLength() int {
  if cmdOptions.argsExtra == nil { return 0 }
  return len(cmdOptions.argsExtra)
}

func argsExtra(index int) string {
  if cmdOptions.argsExtra == nil { return "" }
  if index < 0 || index > len(cmdOptions.argsExtra) { return "" }
  return cmdOptions.argsExtra[index]
}

func argsLength() int {
  return cmdOptions.optionsLength
}

func argsHelp() (bool, bool) {
  return cmdOptions.help.value, cmdOptions.help.set
}

func argsVersion() (bool, bool) {
  return cmdOptions.version.value, cmdOptions.version.set
}

func argsVerbose() (bool, bool) {
  return cmdOptions.verbose.value, cmdOptions.verbose.set
}

func argsLogStyle() (bool, bool) {
  return cmdOptions.logStyle.value, cmdOptions.logStyle.set
}

func argsCompact() (bool, bool) {
  return cmdOptions.compact.value, cmdOptions.compact.set
}

func argsOutputType() (string, bool) {
  return cmdOptions.outputType.value, cmdOptions.outputType.set
}

func argsOutput() (string, bool) {
  return cmdOptions.output.value, cmdOptions.output.set
}

func argsBamVersion() (int, bool) {
  return cmdOptions.bamVersion.value, cmdOptions.bamVersion.set
}

func argsBamOutput() (string, bool) {
  return cmdOptions.bamOutput.value, cmdOptions.bamOutput.set
}

func argsBamPvrzPath() (string, bool) {
  return cmdOptions.bamPvrzPath.value, cmdOptions.bamPvrzPath.set
}

func argsBamSearchPath() (string, bool) {
  return cmdOptions.bamSearchPath.value, cmdOptions.bamSearchPath.set
}

func argsBamCenter() (bool, bool) {
  return cmdOptions.bamCenter.value, cmdOptions.bamCenter.set
}

func argsBamV1Compress() (bool, bool) {
  return cmdOptions.bamv1Compress.value, cmdOptions.bamv1Compress.set
}

func argsBamV1Rle() (int, bool) {
  return cmdOptions.bamv1Rle.value, cmdOptions.bamv1Rle.set
}

func argsBamV1Alpha() (bool, bool) {
  return cmdOptions.bamv1Alpha.value, cmdOptions.bamv1Alpha.set
}

func argsBamV1QualityMin() (int, bool) {
  return cmdOptions.bamv1QualityMin.value, cmdOptions.bamv1QualityMin.set
}

func argsBamV1QualityMax() (int, bool) {
  return cmdOptions.bamv1QualityMax.value, cmdOptions.bamv1QualityMax.set
}

func argsBamV1Speed() (int, bool) {
  return cmdOptions.bamv1Speed.value, cmdOptions.bamv1Speed.set
}

func argsBamV1Dither() (float32, bool) {
  return cmdOptions.bamv1Dither.value, cmdOptions.bamv1Dither.set
}

func argsBamV1SortBy() (string, bool) {
  return cmdOptions.bamv1SortBy.value, cmdOptions.bamv1SortBy.set
}

func argsBamV1Palette() (string, bool) {
  return cmdOptions.bamv1Palette.value, cmdOptions.bamv1Palette.set
}

func argsBamV2StartIndex() (int, bool) {
  return cmdOptions.bamv2StartIndex.value, cmdOptions.bamv2StartIndex.set
}

func argsBamV2Encoding() (int, bool) {
  return cmdOptions.bamv2Encoding.value, cmdOptions.bamv2Encoding.set
}

func argsBamV2Threshold() (float32, bool) {
  return cmdOptions.bamv2Threshold.value, cmdOptions.bamv2Threshold.set
}

func argsBamV2Quality() (int, bool) {
  return cmdOptions.bamv2Quality.value, cmdOptions.bamv2Quality.set
}

func argsBamV2WeightAlpha() (bool, bool) {
  return cmdOptions.bamv2WeightAlpha.value, cmdOptions.bamv2WeightAlpha.set
}

func argsBamV2UseMetric() (bool, bool) {
  return cmdOptions.bamv2UseMetric.value, cmdOptions.bamv2UseMetric.set
}

// Returns number of filter definitions.
func argsFilterLength() int {
  if cmdOptions.filters != nil {
    return len(cmdOptions.filters)
  }
  return 0
}

// Returns filter structure at specified index.
func argsFilter(index int) (OptFilter, bool) {
  if cmdOptions.filters != nil {
    if index >= 0 && index < len(cmdOptions.filters) {
      return cmdOptions.filters[index], true
    }
  }
  return OptFilter{}, false
}


// Used internally. Parses and returns a filter definition.
func parseFilter(param string) (filter OptFilter, err error) {
  filter = OptFilter{name: "", options: make([]OptFilterOption, 0)}

  param = strings.TrimSpace(param)
  if len(param) == 0 { err = fmt.Errorf("No filter name found"); return }

  // parsing filter name
  idx := strings.Index(param, ":")
  if idx < 0 {
    filter.name = strings.TrimSpace(param)
    param = param[len(param):]
  } else {
    filter.name = strings.TrimSpace(param[:idx])
    param = param[idx+1:]
  }
  if len(filter.name) == 0 { err = fmt.Errorf("Empty filter name"); return }

  // parsing filter options
  for len(param) > 0 {
    var option string
    idx = strings.Index(param, ";")
    if idx < 0 {
      option = param
      param = param[len(param):]
    } else {
      option = param[:idx]
      param = param[idx+1:]
    }
    option = strings.TrimSpace(option)
    if len(option) > 0 {
      idx = strings.Index(option, "=")
      if idx < 0 { err = fmt.Errorf("Invalid syntax in filter option: %q", option); return }
      key := strings.TrimSpace(option[:idx])
      value := strings.TrimSpace(option[idx+1:])
      filter.options = append(filter.options, OptFilterOption{key: key, value: value})
    }
  }

  return
}
