package config

import (
  "fmt"
  "strings"
)

// Generic variant type used to represent various datatypes used in the config package.
type Variant interface { ToString() string }
// Variant of type bool
type VarBool interface { ToBool() bool }
// Variant of type int64
type VarInt interface { ToInt() int64 }
// Variant of type float64
type VarFloat interface { ToFloat() float64 }
// Variant of type []int64
type VarIntArray interface { ToIntArray() []int64 }
// Variant of type []float64
type VarFloatArray interface { ToFloatArray() []float64 }
// Variant of type []string
type VarTextArray interface { ToTextArray() []string }
// Variant of type [][]int64
type VarIntMultiArray interface { ToIntMultiArray() [][]int64 }
// Variant of a custom filter structure
type VarFilterMap interface {
  GetName() string
  GetOptions() [][]string
}

type Text struct { Value string }
type Bool struct { Value bool }
type Int struct { Value int64 }
type Float struct { Value float64 }
type IntArray struct { Value []int64 }
type FloatArray struct { Value []float64 }
type TextArray struct { Value []string }
type IntMultiArray struct { Value [][]int64 }
type Filter struct {
  Name      string
  Options   map[string]string
}


func (t Text) ToString() string { return t.Value }

func (b Bool) ToString() string { return fmt.Sprintf("%v", b.Value) }
func (b Bool) ToBool() bool { return b.Value }

func (i Int) ToString() string { return fmt.Sprintf("%d", i.Value) }
func (i Int) ToInt() int64 { return i.Value }

func (f Float) ToString() string { return fmt.Sprintf("%f", f.Value) }
func (f Float) ToFloat() float64 { return f.Value }

func (ia IntArray) ToString() string { return fmt.Sprintf("%v", ia.Value) }
func (ia IntArray) ToIntArray() []int64 { return ia.Value }

func (fa FloatArray) ToString() string { return fmt.Sprintf("%v", fa.Value) }
func (fa FloatArray) ToFloatArray() []float64 { return fa.Value }

func (ta TextArray) ToString() string { return fmt.Sprintf("%v", ta.Value) }
func (ta TextArray) ToTextArray() []string { return ta.Value }

func (ima IntMultiArray) ToString() string { return fmt.Sprintf("%v", ima.Value) }
func (ima IntMultiArray) ToIntMultiArray() [][]int64 { return ima.Value }

// ToString returns summary of filter name and options.
func (f Filter) ToString() string {
  var sb strings.Builder
  sb.WriteString(fmt.Sprintf("{name:%s}", f.Name))
  for key, value := range f.Options {
    sb.WriteString(fmt.Sprintf(",{%s:%s}", key, value))
  }
  return sb.String()
}

// GetName returns the filter name.
func (f Filter) GetName() string { return f.Name }

// GetOptions returns all options as an array of key/value pairs.
func (f Filter) GetOptions() [][]string {
  retVal := make([][]string, 0, len(f.Options))
  for key, value := range f.Options {
    retVal = append(retVal, []string{key, value})
  }
  return retVal
}
