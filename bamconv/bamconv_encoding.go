package main
// Handles configuration encoding details.

import (
  "encoding/xml"
  "encoding/json"
  "fmt"
  "io"
  "os"
  "strconv"
  "strings"

  "github.com/InfinityTools/bamcreator/config"
  "github.com/InfinityTools/bamcreator/bam"
  "github.com/InfinityTools/go-logging"
)


// Returns a pointer to a XmlGenerator structure initialized with the default values.
func getDefaultXml() *config.XmlGenerator {
  xmlData := config.XmlGenerator{}
  xmlData.Output.Version = "1"
  xmlData.Output.File = "default.bam"
  xmlData.Output.PvrzPath = ""
  xmlData.Input.Static = "true"
  xmlData.Input.Files = make([]string, 0)
  xmlData.Input.Sequence.Path = ""
  xmlData.Input.Sequence.Prefix = ""
  xmlData.Input.Sequence.SuffixStart = "0"
  xmlData.Input.Sequence.SuffixEnd = "0"
  xmlData.Input.Sequence.SuffixLength = "1"
  xmlData.Input.Sequence.Ext = ""
  xmlData.Input.Search = make([]string, 0)
  xmlData.Settings.Center = make([]string, 0)
  xmlData.Settings.Sequence = make([]string, 0)
  xmlData.BamV1.Compress = "true"
  xmlData.BamV1.Rle = "-1"
  xmlData.BamV1.Alpha = "true"
  xmlData.BamV1.QualityMin = "80"
  xmlData.BamV1.QualityMax = "100"
  xmlData.BamV1.Speed = "3"
  xmlData.BamV1.Dither = "0.0"
  xmlData.BamV1.FixedColors = make([]string, 0)
  xmlData.BamV1.TransColor = "0xff00ff00"
  xmlData.BamV1.UseTransColor = "false"
  xmlData.BamV1.SortBy = "none"
  xmlData.BamV1.UsePalette = "false"
  xmlData.BamV1.Palette = ""
  xmlData.BamV2.StartIndex = "1000"
  xmlData.BamV2.Encoding = "0"
  xmlData.BamV2.Threshold = "5.0"
  xmlData.BamV2.Quality = "1"
  xmlData.BamV2.WeightAlpha = "true"
  xmlData.BamV2.UseMetric = "false"
  xmlData.Filters = make([]config.XmlFilter, 0)
  return &xmlData
}

// Returns a pointer to a JsonGenerator structure initialized with the default values.
func getDefaultJson() *config.JsonGenerator {
  jsonData := config.JsonGenerator{}
  jsonData.Output.Version = 1
  jsonData.Output.File = "default.bam"
  jsonData.Output.PvrzPath = ""
  jsonData.Input.Static = true
  jsonData.Input.Files = make([]string, 0)
  jsonData.Input.Sequence.Path = ""
  jsonData.Input.Sequence.Prefix = ""
  jsonData.Input.Sequence.SuffixStart = 0
  jsonData.Input.Sequence.SuffixEnd = 0
  jsonData.Input.Sequence.SuffixLength = 1
  jsonData.Input.Sequence.Ext = ""
  jsonData.Input.Search = make([]string, 0)
  jsonData.Settings.Center = make([][]int64, 0)
  jsonData.Settings.Sequence = make([][]int64, 0)
  jsonData.BamV1.Compress = true
  jsonData.BamV1.Rle = -1
  jsonData.BamV1.Alpha = true
  jsonData.BamV1.QualityMin = 80
  jsonData.BamV1.QualityMax = 100
  jsonData.BamV1.Speed = 3
  jsonData.BamV1.Dither = 0.0
  jsonData.BamV1.FixedColors = make([]string, 0)
  jsonData.BamV1.TransColor = "0xff00ff00"
  jsonData.BamV1.UseTransColor = false
  jsonData.BamV1.SortBy = "none"
  jsonData.BamV1.UsePalette = false
  jsonData.BamV1.Palette = ""
  jsonData.BamV2.StartIndex = 1000
  jsonData.BamV2.Encoding = 0
  jsonData.BamV2.Threshold = 5.0
  jsonData.BamV2.Quality = 1
  jsonData.BamV2.WeightAlpha = true
  jsonData.BamV2.UseMetric = false
  jsonData.Filters = make([]config.JsonFilter, 0)
  return &jsonData
}


// Handles creation and output of XML configuration.
func generateXml(w io.Writer, compact bool) error {
  if w == nil { w = os.Stdout }

  data := getDefaultXml()

  // Adding options
  if v, x := argsBamVersion(); x {
    data.Output.Version = strconv.Itoa(v)
  }
  if s, x := argsBamOutput(); x {
    data.Output.File = s
  }
  if s, x := argsBamPvrzPath(); x {
    data.Output.PvrzPath = s
  }
  if s, x := argsBamSearchPath(); x {
    data.Input.Search = append(data.Input.Search, s)
  }
  if b, x := argsBamV1Compress(); x {
    data.BamV1.Compress = strconv.FormatBool(b)
  }
  if v, x := argsBamV1Rle(); x {
    data.BamV1.Rle = strconv.Itoa(v)
  }
  if b, x := argsBamV1Alpha(); x {
    data.BamV1.Alpha = strconv.FormatBool(b)
  }
  if v, x := argsBamV1QualityMin(); x {
    data.BamV1.QualityMin = strconv.Itoa(v)
  }
  if v, x := argsBamV1QualityMax(); x {
    data.BamV1.QualityMax = strconv.Itoa(v)
  }
  if v, x := argsBamV1Speed(); x {
    data.BamV1.Speed = strconv.Itoa(v)
  }
  if f, x := argsBamV1Dither(); x {
    data.BamV1.Dither = strconv.FormatFloat(float64(f), 'f', -1, 32)
  }
  if s, x := argsBamV1SortBy(); x {
    data.BamV1.SortBy = s
  }
  if s, x := argsBamV1Palette(); x {
    data.BamV1.UsePalette = "true"
    data.BamV1.Palette = s
  }
  if v, x := argsBamV2StartIndex(); x {
    data.BamV2.StartIndex = strconv.Itoa(v)
  }
  if v, x := argsBamV2Encoding(); x {
    data.BamV2.Encoding = strconv.Itoa(v)
  }
  if f, x := argsBamV2Threshold(); x {
    data.BamV2.Threshold = strconv.FormatFloat(float64(f), 'f', -1, 32)
  }
  if v, x := argsBamV2Quality(); x {
    data.BamV2.Quality = strconv.Itoa(v)
  }
  if b, x := argsBamV2WeightAlpha(); x {
    data.BamV2.WeightAlpha = strconv.FormatBool(b)
  }
  if b, x := argsBamV2UseMetric(); x {
    data.BamV2.UseMetric = strconv.FormatBool(b)
  }

  // Adding filter definitions
  if numFilters := argsFilterLength(); numFilters > 0 {
    for i := 0; i < numFilters; i++ {
      if filter, x := argsFilter(i); x {
        xmlFilter := config.XmlFilter{Name: filter.name, Options: make([]config.XmlFilterOption, 0)}
        for _, option := range filter.options {
          xmlFilter.Options = append(xmlFilter.Options, config.XmlFilterOption{Key: option.key, Value: option.value})
        }
        data.Filters = append(data.Filters, xmlFilter)
      } else {
        logging.Logf("Filter %d not defined. Skipping.\n", i)
      }
    }
  }

  var center bool = false
  if b, x := argsBamCenter(); x {
    center = b
  }

  // Processing BAM input files
  if numBams := argsExtraLength(); numBams > 0 {
    // Preparing BAM input
    logging.Log("Parsing BAM files")
    bams := make([]*bam.BamFile, 0)
    search := data.Input.Search
    for i := 0; i < numBams; i++ {
      fileName := argsExtra(i)
      fin, err := os.Open(fileName)
      if err != nil { return fmt.Errorf("BAM file %q: %v", fileName, err) }
      defer fin.Close()

      data.Input.Files = append(data.Input.Files, fileName)
      var b *bam.BamFile = nil
      if len(search) > 0 {
        b = bam.ImportEx(fin, search)
      } else {
        b = bam.Import(fin)
      }
      if b == nil { return fmt.Errorf("BAM file %q: Could not import data", fileName) }
      if b.Error() != nil { return fmt.Errorf("BAM file %q: %v", fileName, b.Error()) }
      if b.GetFrameLength() == 0 { return fmt.Errorf("BAM file %q: Does not contain any frames", fileName) }

      bams = append(bams, b)
      logging.LogProgressDot(i, numBams, 79 - 17)    // 17 is length of prefix string above
    }
    logging.OverridePrefix(false, false, false).Logln("")

    // Processing BAM structures (centers, cycles)
    frameOffset := 0   // offset to first frame index of BAM
    for _, b := range bams {
      // Adding centers
      if center {
        for i, cnt := 0, b.GetFrameLength(); i < cnt; i++ {
          data.Settings.Center = append(data.Settings.Center, fmt.Sprintf("%d,%d", b.GetFrameCenterX(i), b.GetFrameCenterY(i)))
        }
      }

      // Adding cycles
      for i, cnt := 0, b.GetCycleLength(); i < cnt; i++ {
        cycle := b.GetCycle(i)
        sb := strings.Builder{}
        for _, j := range cycle {
          if sb.Len() > 0 { sb.WriteString(",") }
          sb.WriteString(strconv.Itoa(int(j)+frameOffset))
        }
        if sb.Len() == 0 { sb.WriteString("0") }
        data.Settings.Sequence = append(data.Settings.Sequence, sb.String())
      }

      // Adjusting frame index offset for next BAM
      frameOffset += b.GetFrameLength()
    }
  }

  // Writing data to output
  logging.Logln("Generating XML configuration data")
  var err error
  var buf []byte = nil
  if compact {
    buf, err = xml.Marshal(data)
  } else {
    buf, err = xml.MarshalIndent(data, "", "    ")
  }
  if err != nil { return fmt.Errorf("Encoding XML data: %v", err) }

  _, err = w.Write([]byte(xml.Header))
  if err != nil { return fmt.Errorf("Writing XML data: %v", err) }
  _, err = w.Write(buf)
  if err != nil { return fmt.Errorf("Writing XML data: %v", err) }

  return nil
}


// Handles creation and output of JSON configuration.
func generateJson(w io.Writer, compact bool) error {
  if w == nil { w = os.Stdout }

  data := getDefaultJson()

  // Adding options
  if v, x := argsBamVersion(); x {
    data.Output.Version = int64(v)
  }
  if s, x := argsBamOutput(); x {
    data.Output.File = s
  }
  if s, x := argsBamPvrzPath(); x {
    data.Output.PvrzPath = s
  }
  if s, x := argsBamSearchPath(); x {
    data.Input.Search = append(data.Input.Search, s)
  }
  if b, x := argsBamV1Compress(); x {
    data.BamV1.Compress = b
  }
  if v, x := argsBamV1Rle(); x {
    data.BamV1.Rle = int64(v)
  }
  if b, x := argsBamV1Alpha(); x {
    data.BamV1.Alpha = b
  }
  if v, x := argsBamV1QualityMin(); x {
    data.BamV1.QualityMin = int64(v)
  }
  if v, x := argsBamV1QualityMax(); x {
    data.BamV1.QualityMax = int64(v)
  }
  if v, x := argsBamV1Speed(); x {
    data.BamV1.Speed = int64(v)
  }
  if f, x := argsBamV1Dither(); x {
    data.BamV1.Dither = float64(f)
  }
  if s, x := argsBamV1SortBy(); x {
    data.BamV1.SortBy = s
  }
  if s, x := argsBamV1Palette(); x {
    data.BamV1.UsePalette = true
    data.BamV1.Palette = s
  }
  if v, x := argsBamV2StartIndex(); x {
    data.BamV2.StartIndex = int64(v)
  }
  if v, x := argsBamV2Encoding(); x {
    data.BamV2.Encoding = int64(v)
  }
  if f, x := argsBamV2Threshold(); x {
    data.BamV2.Threshold = float64(f)
  }
  if v, x := argsBamV2Quality(); x {
    data.BamV2.Quality = int64(v)
  }
  if b, x := argsBamV2WeightAlpha(); x {
    data.BamV2.WeightAlpha = b
  }
  if b, x := argsBamV2UseMetric(); x {
    data.BamV2.UseMetric = b
  }

  // Adding filter definitions
  if numFilters := argsFilterLength(); numFilters > 0 {
    for i := 0; i < numFilters; i++ {
      if filter, x := argsFilter(i); x {
        jsonFilter := config.JsonFilter{Name: filter.name, Options: make([]config.JsonFilterOption, 0)}
        for _, option := range filter.options {
          jsonFilter.Options = append(jsonFilter.Options, config.JsonFilterOption{Key: option.key, Value: option.value})
        }
        data.Filters = append(data.Filters, jsonFilter)
      } else {
        logging.Logf("Filter %d not defined. Skipping.\n", i)
      }
    }
  }

  var center bool = false
  if b, x := argsBamCenter(); x {
    center = b
  }

  // Processing BAM input files
  if numBams := argsExtraLength(); numBams > 0 {
    // Preparing BAM input
    logging.Log("Parsing BAM files")
    bams := make([]*bam.BamFile, 0)
    search := data.Input.Search
    for i := 0; i < numBams; i++ {
      fileName := argsExtra(i)
      fin, err := os.Open(fileName)
      if err != nil { return fmt.Errorf("BAM file %q: %v", fileName, err) }
      defer fin.Close()

      data.Input.Files = append(data.Input.Files, fileName)
      var b *bam.BamFile = nil
      if len(search) > 0 {
        b = bam.ImportEx(fin, search)
      } else {
        b = bam.Import(fin)
      }
      if b == nil { return fmt.Errorf("BAM file %q: Could not import data", fileName) }
      if b.Error() != nil { return fmt.Errorf("BAM file %q: %v", fileName, b.Error()) }
      if b.GetFrameLength() == 0 { return fmt.Errorf("BAM file %q: Does not contain any frames", fileName) }

      bams = append(bams, b)
      logging.LogProgressDot(i, numBams, 79 - 17)    // 17 is length of prefix string above
    }
    logging.OverridePrefix(false, false, false).Logln("")

    // Processing BAM structures (centers, cycles)
    frameOffset := 0   // offset to first frame index of BAM
    for _, b := range bams {
      // Adding centers
      if center {
        for i, cnt := 0, b.GetFrameLength(); i < cnt; i++ {
          data.Settings.Center = append(data.Settings.Center, []int64{int64(b.GetFrameCenterX(i)), int64(b.GetFrameCenterY(i))})
        }
      }

      // Adding cycles
      for i, cnt := 0, b.GetCycleLength(); i < cnt; i++ {
        cycle := b.GetCycle(i)
        c := make([]int64, 0)
        for _, j := range cycle {
          c = append(c, int64(int(j)+frameOffset))
        }
        if len(c) == 0 { c = append(c, 0) }
        data.Settings.Sequence = append(data.Settings.Sequence, c)
      }

      // Adjusting frame index offset for next BAM
      frameOffset += b.GetFrameLength()
    }
  }

  // Writing data to output
  logging.Logln("Generating JSON configuration data")
  var err error
  var buf []byte = nil
  if compact {
    buf, err = json.Marshal(data)
  } else {
    buf, err = json.MarshalIndent(data, "", "    ")
  }
  if err != nil { return fmt.Errorf("Encoding JSON data: %v", err) }

  _, err = w.Write(buf)
  if err != nil { return fmt.Errorf("Writing JSON data: %v", err) }

  return nil
}
