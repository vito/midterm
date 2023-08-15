package vt100

import (
	"sort"
	"strings"

	"github.com/muesli/termenv"
)

// Format represents the display format of text on a terminal.
type Format struct {
	// Reset inidcates that the format should be reset prior to applying any of
	// the other fields.
	Reset bool
	// Fg is the foreground color.
	Fg termenv.Color
	// Bg is the background color.
	Bg termenv.Color
	// Intensity is the text intensity (bright, normal, dim).
	Intensity Intensity
	// Various text properties.
	Italic, Underline, Blink, Reverse, Conceal, CrossOut, Overline bool
}

func toCss(c termenv.Color) string {
	return termenv.ConvertToRGB(c).Hex()
}

func (f Format) css() string {
	parts := make([]string, 0)
	fg, bg := f.Fg, f.Bg
	if f.Reverse {
		bg, fg = fg, bg
	}

	parts = append(parts, "color:"+toCss(fg))
	parts = append(parts, "background-color:"+toCss(bg))
	switch f.Intensity {
	case Bold:
		parts = append(parts, "font-weight:bold")
	case Normal:
	case Faint:
		parts = append(parts, "opacity:0.33")
	}
	if f.Underline {
		parts = append(parts, "text-decoration:underline")
	}
	if f.Conceal {
		parts = append(parts, "display:none")
	}
	if f.Blink {
		parts = append(parts, "text-decoration:blink")
	}

	// We're not in performance sensitive code. Although this sort
	// isn't strictly necessary, it gives us the nice property that
	// the style of a particular set of attributes will always be
	// generated the same way. As a result, we can use the html
	// output in tests.
	sort.StringSlice(parts).Sort()

	return strings.Join(parts, ";")
}

type Intensity int

const (
	Normal Intensity = 0
	Bold   Intensity = 1
	Faint  Intensity = 2
	// TODO(jaguilar): Should this be in a subpackage, since the names are pretty collide-y?
)

func (i Intensity) alpha() uint8 {
	switch i {
	case Bold:
		return 255
	case Normal:
		return 170
	case Faint:
		return 85
	default:
		return 170
	}
}
