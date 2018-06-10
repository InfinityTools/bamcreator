package bam
// Provides general-purpose functionality.

import (
  "runtime"
)

var multithreaded bool = runtime.NumCPU() > 1


// GetMultiThreaded returns whether multithreading should be used for selected operations.
func GetMultiThreaded() bool {
  return multithreaded
}


// SetMultiThreaded sets whether multithreading should be used for selected operations.
func SetMultiThreaded(set bool) {
  multithreaded = set
}
