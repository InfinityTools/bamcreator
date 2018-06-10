/*
Package sort provides functionality for sorting color tables by a selected characteristic.
*/
package sort

import (
  "image/color"
  "math"
  "sort"
)

// Available Sort types and flags.
const (
  // Don't sort colors. Only useful in combination with sort flags.
  SORT_BY_NONE        = 0x00
  // Sort by the perceived lightness aspect of a color.
  SORT_BY_LIGHTNESS   = 0x01
  // Sort by the saturation aspect of a color.
  SORT_BY_SATURATION  = 0x02
  // Sort by the hue aspect of a color.
  SORT_BY_HUE         = 0x03
  // Sort by red color component.
  SORT_BY_RED         = 0x04
  // Sort by green color component.
  SORT_BY_GREEN       = 0x05
  // Sort by blue color component.
  SORT_BY_BLUE        = 0x06
  // Sort by alpha component.
  SORT_BY_ALPHA       = 0x07

  // Sort colors in reversed order
  SORT_REVERSED       = 0x100
)

// ColorMapping maps input to output palette indices.
type ColorMapping map[int]int

type sortEntry struct {
  index int       // the original palette index
  value float64   // the value to sort by
}

type sortTable []sortEntry

// Sort performs a sort operation over the specified color table according to sortFlags.
// sortFlags may be composed of one of the SORT_BY_xxx constants and optional SORT_xxx flags.
// Color entries at startIndex and higher are considered for sorting. Specify 0 to sort all color entries.
// Returns nil in both values on error.
func Sort(pal color.Palette, sortFlags int, startIndex int) (palOut color.Palette, remap ColorMapping) {
  if pal == nil { return }
  if startIndex < 0 { startIndex = 0 }

  // Sorting not needed for small palette regions
  if len(pal) - startIndex < 2 || sortFlags == SORT_BY_NONE {
    palOut = make(color.Palette, len(pal))
    copy(palOut, pal)
    remap = make(ColorMapping)
    for i := 0; i < len(palOut); i++ { remap[i] = i }
    return
  }


  stype := sortFlags & 0xff
  reversed := (sortFlags & SORT_REVERSED != 0)
  var stable sortTable = nil
  switch stype {
    case SORT_BY_LIGHTNESS:
      stable = createSortTable(pal[startIndex:], lightness)
    case SORT_BY_SATURATION:
      stable = createSortTable(pal[startIndex:], saturation)
    case SORT_BY_HUE:
      stable = createSortTable(pal[startIndex:], hue)
    case SORT_BY_RED:
      stable = createSortTable(pal[startIndex:], red)
    case SORT_BY_GREEN:
      stable = createSortTable(pal[startIndex:], green)
    case SORT_BY_BLUE:
      stable = createSortTable(pal[startIndex:], blue)
    case SORT_BY_ALPHA:
      stable = createSortTable(pal[startIndex:], alpha)
    default:
      stable = createSortTable(pal[startIndex:], none)
  }
  if stable == nil { return }
  stable.Sort()
  if reversed { reverseTable(stable) }

  palOut = make(color.Palette, len(pal))
  remap = make(ColorMapping)

  // Preserve fixed palette entries
  for i := 0; i < startIndex; i++ {
    remap[i] = i
    palOut[i] = pal[i]
  }

  // Write mappings to return values
  for i := 0; i < len(stable); i++ {
    palOut[i + startIndex] = pal[stable[i].index + startIndex]
    remap[stable[i].index + startIndex] = i + startIndex
  }

  return
}


// Returns perceived lightness value of the color, mapped to range [0.0, 1.0]
func lightness(col color.Color) float64 {
  r, g, b, _ := normalizeColor(col)
  r *= r * 0.299
  g *= g * 0.587
  b *= b * 0.114
  return math.Sqrt(r + g + b)
}

// Returns saturation value of the color, mapped to range [0.0, 1.0]
func saturation(col color.Color) float64 {
  r, g, b, _ := normalizeColor(col)
  cmin := r; if g < cmin { cmin = g }; if b < cmin { cmin = b }
  cmax := r; if g > cmax { cmax = g }; if b > cmax { cmax = b }
  csum := cmax + cmin
  cdelta := cmax - cmin

  var s float64 = 0.0
  if cdelta != 0.0 {
    csum2 := csum / 2.0
    if csum2 < 0.5 {
      s = cdelta / csum
    } else {
      s = cdelta / (2.0 - csum)
    }
  }
  return s
}

// Returns hue value of the color, mapped to range [0.0, 1.0]
func hue(col color.Color) float64 {
  r, g, b, _ := normalizeColor(col)
  cmin := r; if g < cmin { cmin = g }; if b < cmin { cmin = b }
  cmax := r; if g > cmax { cmax = g }; if b > cmax { cmax = b }
  cdelta := cmax - cmin
  cdelta2 := cdelta / 2.0

  var h float64 = 0.0
  if cdelta != 0.0 {
    dr := ((cmax - r) / 6.0 + cdelta2) / cdelta
    dg := ((cmax - g) / 6.0 + cdelta2) / cdelta
    db := ((cmax - b) / 6.0 + cdelta2) / cdelta
    switch cmax {
      case r:  h = db - dg
      case g:  h = 1.0/3.0 + dr - db
      default:  h = 2.0/3.0 + dg - dr
    }
    if h < 0.0 { h += 1.0 }
    if h > 1.0 { h -= 1.0 }
  }
  return h
}

// Returns red color channel, mapped to range [0.0, 1.0]
func red(col color.Color) float64 {
  r, _, _, _ := normalizeColor(col)
  return r
}

// Returns green color channel, mapped to range [0.0, 1.0]
func green(col color.Color) float64 {
  _, g, _, _ := normalizeColor(col)
  return g
}

// Returns blue color channel, mapped to range [0.0, 1.0]
func blue(col color.Color) float64 {
  _, _, b, _ := normalizeColor(col)
  return b
}

// Returns alpha channel, mapped to range [0.0, 1.0]
func alpha(col color.Color) float64 {
  _, _, _, a := normalizeColor(col)
  return a
}

// No-op function to keep original palette order after performing sort.
func none(col color.Color) float64 {
  return 0.0
}

// Returns each color component in range [0.0, 1.0]
func normalizeColor(col color.Color) (r, g, b, a float64) {
  nr, ng, nb, na := col.RGBA()
  na >>= 8
  if na > 0 {
    a = float64(na)
    r = float64(nr >> 8) / a
    g = float64(ng >> 8) / a
    b = float64(nb >> 8) / a
    a /= 255.0
  }
  return
}


// Reverses order of the given table.
func reverseTable(table sortTable) {
  if table != nil && len(table) > 1 {
    for i, j := 0, len(table) - 1; i < j; i, j = i+1, j-1 {
      table[i], table[j] = table[j], table[i]
    }
  }
}

// Converts given palette into an array of sort entries.
func createSortTable(pal color.Palette, f func(color.Color) float64) sortTable {
  st := make(sortTable, len(pal))
  for i := 0; i < len(pal); i++ {
    st[i] = sortEntry{index: i, value: f(pal[i])}
  }
  return st
}

// Performs a stable sort of the table data.
func (s sortTable) Sort() {
  sort.Stable(s)
}

func (s sortTable) Len() int {
  return len(s)
}

func (s sortTable) Less(i, j int) bool {
  return s[i].value < s[j].value
}

func (s sortTable) Swap(i, j int) {
  s[i], s[j] = s[j], s[i]
}
