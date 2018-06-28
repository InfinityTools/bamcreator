/*
Package bamcreator provides definitions that are shared by the more specialized tools.

BAM Creator is released under the BSD 2-clause license. See LICENSE in the project's root folder for more details.
*/
package bamcreator

import (
  "fmt"
  "runtime"
)

// Version number of the whole bamcreator package.
const (
  VERSION_MAJOR = 0
  VERSION_MINOR = 3
  VERSION_PATCH = 0
)

// PrintVersion prints the current version of the bamcreator package to standard output,
// prefixed by the specified tool name.
func PrintVersion(toolName string) {
  fmt.Printf("%s version %d.%d.%d (binary: %s, %s)\n",
             toolName,
             VERSION_MAJOR, VERSION_MINOR, VERSION_PATCH,
             runtime.GOOS, runtime.GOARCH)
}
