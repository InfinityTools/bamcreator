package config

import (
  "fmt"
)

// AssembleFilePath assembles a fully qualified file path string from the specified arguments.
// ok returns false if mustExist is set and file doesn't exist.
func AssembleFilePath(path, prefix, ext string, index, indexWidth int64) string {
  file := path
  for len(file) > 1 && (file[len(file)-1:] == "/" || file[len(file)-1:] == "\\") { file = file[:len(file)-1] }
  if len(file) > 0 && file[len(file)-1:] != "/" { file += "/" }
  if len(prefix) > 0 { file += prefix }

  // generating a prefixed index string
  neg := ""
  if index < 0 { neg = "-"; index = -index; indexWidth-- }
  if indexWidth < 0 { indexWidth = 0 }
  if len(ext) > 0 && ext[:1] != "." { ext = "." + ext }
  fmtString := fmt.Sprintf("%s%s%%0%dd%s", file, neg, indexWidth, ext)
  file = fmt.Sprintf(fmtString, index)

  return file
}
