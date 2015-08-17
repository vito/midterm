package vt100

import (
	"strings"
	"unicode/utf8"
)

// fromLines generates a VT100 from content text.
// Each line must have the same number of runes.
func fromLines(s string) *VT100 {
	return fromLinesAndFormats(s, nil)
}

// fromLinesAndFormats generates a *VT100 whose state is set according
// to s (for content) and a (for attributes).
//
// Dimensions are set to the width of s' first line and the height of the
// number of lines in s.
//
// If a is nil, the default attributes are used.
func fromLinesAndFormats(s string, a [][]Format) *VT100 {
	lines := strings.Split(s, "\n")
	v := NewVT100(len(lines), utf8.RuneCountInString(lines[0]))
	for y := 0; y < v.Height; y++ {
		x := 0
		for _, r := range lines[y] {
			v.Content[y][x] = r
			if a != nil {
				v.Format[y][x] = a[y][x]
			}
			x++
		}
	}
	return v
}
