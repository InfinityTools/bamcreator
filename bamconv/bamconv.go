/*
BAM Converter (bamconv) is a supplementary tool that can be used to help generate configuration data from
BAM input files.

BAM Converter is part of the BAM Creator package. BAM Creator is released under the BSD 2-clause license.
See LICENSE in the project's root folder for more details.
*/
package main

import (
  "fmt"
  "os"

  "github.com/InfinityTools/bamcreator"
  "github.com/InfinityTools/go-logging"
)

const TOOL_NAME = "BAM Converter"

func main() {
  err := loadArgs(os.Args)
  if err != nil {
    logging.Errorf("Error: %v\n", err)
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

  // Logger should not interfere with configuration data when writing to stdout
  if _, x := argsOutput(); !x {
    logging.SetOutput(logging.LOG, os.Stderr)
    logging.SetOutput(logging.INFO, os.Stderr)
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
    bamcreator.PrintVersion(TOOL_NAME)
  } else if _, x := argsHelp(); x {
    printHelp()
  } else if argsExtraLength() == 0 {
    printHelp()
  } else {
    logging.Infoln("Starting configuration generation")
    err = generate()
    if err != nil {
      logging.Errorf("Error: %v\n", err)
      logging.Infoln("Configuration generation failed.")
      os.Exit(1)
    }
    logging.Infoln("Configuration generation finished successfully.")
  }
}


func generate() error {
  outType := "xml"
  if s, x := argsOutputType(); x {
    outType = s
  }

  var err error = nil
  output := os.Stdout
  if s, x := argsOutput(); x {
    output, err = os.Create(s)
    if err != nil { return fmt.Errorf("Output file: %v", err) }
    defer output.Close()
  }

  var compact bool = false
  if b, x := argsCompact(); x {
    compact = b
  }

  if outType == "json" {
    err = generateJson(output, compact)
  } else {
    err = generateXml(output, compact)
  }

  return err
}


func printHelp() {
  fmt.Printf("Usage: %s [options] bamfile [bamfile2 ...]\n", os.Args[0])
  const helpText = "A supplementary tool for BAM Creator that generates configurations derived from\n" +
                   "specified BAM files.\n" +
                   "\n" +
                // "...............................................................................\n" +
                   "Options:\n" +
                   "  --verbose                 Show additional log messages. All messages will be\n" +
                   "                            redirected to standard error if configuration is\n" +
                   "                            written to standard output.\n" +
                   "  --silent                  Suppress any log messages. Useful when writing\n" +
                   "                            configuration data to standard output.\n" +
                   "  --log-style               Print log messages in log style, complete with\n" +
                   "                            timestamp and log level.\n" +
                   "  --compact                 Generate compact configuration data (without\n" +
                   "                            indentations, line breaks, etc.). Default: generate\n" +
                   "                            preformatted configuration data\n" +
                   "  --output-type type        Specify configuration type to output. Can be xml or\n" +
                   "                            json. Default: xml\n" +
                   "  --output configfile       Specify a filename where configuration data should\n" +
                   "                            be written to. By default configuration data is\n" +
                   "                            written to standard output.\n" +
                   "  --bam-version version     Set BAM output version. Can be 1 for BAM V1 or\n" +
                   "                            2 for BAM V2. Default: 1\n" +
                   "  --bam-output file         Set BAM output file.\n" +
                   "  --bam-pvrz-path path      Set PVRZ output path for BAM V2 output.\n" +
                   "                            Default: same folder as output BAM\n" +
                   "  --bam-search-path path    Search path for pvrz files associated with BAM V2\n" +
                   "                            input files. Default: same path as input BAM.\n" +
                   "  --bam-center              Specify to generate explicit center position\n" +
                   "                            entries for each individual BAM frame.\n" +
                   "  --bamv1-compress          Enable BAM V1 compression. Used by default.\n" +
                   "  --bamv1-no-compress       Disable BAM V1 compression.\n" +
                   "  --bamv1-rle type          Set RLE frame compression type. Allowed types:\n" +
                   "                               -1     Decide based on resulting frame size\n" +
                   "                                0     Always disable RLE encoding\n" +
                   "                                1     Always enable RLE encoding\n" +
                   "                            Default: -1\n" +
                   "  --bamv1-alpha             Preserve alpha in BAM V1 palette. Used by default.\n" +
                   "  --bamv1-no-alpha          Discard alpha in BAM V1 palette.\n" +
                   "  --bamv1-quality-min qmin  Set minimum quality for BAM V1 color quantization.\n" +
                   "                            Allowed range: [0, 100]. Default: 80\n" +
                   "  --bamv1-quality-max qmax  Set maximum quality for BAM V1 color quantization.\n" +
                   "                            Allowed range: [0, 100]. Default: 100\n" +
                   "  --bamv1-speed value       Set speed for palette generation. Default: 3\n" +
                   "  --bamv1-dither value      Set dither strength for output graphics. Value must\n" +
                   "                            be in range [0.0, 1.0]. Set to 0.0 to disable.\n" +
                   "                            Default: 0.0\n" +
                   "  --bamv1-sort type         Sort palette by the specified type. The following\n" +
                   "                            types are recognized: none, lightness, saturation,\n" +
                   "                            hue, red, green, blue, alpha. Append _reversed to\n" +
                   "                            reverse the sort order. Default: none\n" +
                   "  --bamv1-palette file      Specify an external palette. Default: empty\n" +
                   "  --bamv2-start-index idx   Set start index for PVRZ files generated by BAM V2.\n" +
                   "                            Allowed range: [0, 99999]. Default: 1000\n" +
                   "  --bamv2-encoding type     PVRZ pixel encoding type. Available types:\n" +
                   "                                0     Determine automatically\n" +
                   "                                1     Enforce DXT1 (BC1)\n" +
                   "                                2     Enforce DXT3 (BC2) [unsupported by games]\n" +
                   "                                3     Enforce DXT5 (BC3)\n" +
                   "                            Default: 0\n" +
                   "  --bamv2-threshold value   Percentage threshold for determining PVRZ pixel\n" +
                   "                            encoding type. Allowed range: [0.0, 100.0].\n" +
                   "                            Default: 5.0\n" +
                   "  --bamv2-quality value     Quality of PVRZ pixel encoding. in range [0, 2],\n" +
                   "                            where 0 is lowest quality and 2 is highest quality.\n" +
                   "                            Default: 1\n" +
                   "  --bamv2-weight-alpha      Weight pixels by alpha. May improve visual quality\n" +
                   "                            for alpha-blended pixels. Used by default.\n" +
                   "  --bamv2-no-weight-alpha   Don't weight pixels by alpha.\n" +
                   "  --bamv2-use-metric        Apply perceptual metric to encoded pixels. May\n" +
                   "                            improve perceived quality.\n" +
                   "  --bamv2-no-use-metric     Don't apply perceptual metric to encoded pixels.\n" +
                   "                            Used by default.\n" +
                   "  --filter name[:key=value[;key=value[;more options...]]]\n" +
                   "                            Define one or more filters complete with options.\n" +
                   "                            'name' is the filter name, separated by colon from\n" +
                   "                            zero, one or more options. Each option is defined\n" +
                   "                            by the option name 'key' and the respective option\n" +
                   "                            value. Semicolon is used to separate multiple\n" +
                   "                            options. Wrap the whole definition in quotes if it\n" +
                   "                            contains spaces. Add multiple --filter instances to\n" +
                   "                            define multiple filters. Filters are added in order\n" +
                   "                            of appearance.\n" +
                   "  --help                    Print this help and terminate.\n" +
                   "  --version                 Print version information and terminate.\n" +
                   "\n" +
                   "BAM files:\n" +
                   "Configuration data is derived from frame and cycle information of the specified\n" +
                   "BAM file. Additional BAM files will be appended in the order of appearance.\n" +
                   "Cycle definitions are adjusted accordingly."
  fmt.Println(helpText)
}
