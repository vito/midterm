// package vt100 acts as a vt100 terminal emulator. For example,
// you could use it to run a program like nethack that expects
// a terminal as a subprocess. It tracks the position of the cursor,
// colors, and various other aspects of the terminal's state, and
// allows you to inspect them.
//
// This package's terminal does not support completion, readline,
// or any of the other things you might expect if you were trying
// to use it, say, to run a shell. It could run a subshell, but
// mostly you'd use it for programs that operate in raw mode.
package vt100

import (
	"bytes"
	"fmt"
	"image/color"
	"strings"
)

// +gen stringer
type Color int

// +gen stringer
type Attribute uint8

const (
	Default Color = iota
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

const (
	Normal     Attribute = 0
	Bright               = 1
	Dim                  = 2
	Underscore           = 4
	Blink                = 5
	Reverse              = 7 //TODO(jaguilar): not displayed in html.
	Hidden               = 8
)

const (
	alphaNormal uint8 = 170
	alphaBright uint8 = 255
	alphaDim    uint8 = 85
)

var (
	// Mapping from terminal color to rgb. The alpha values are overridden below.
	colorMap = map[Color]color.RGBA{
		Default: {0, 0, 0, 0},
		Black:   {0, 0, 0, 0},
		Red:     {255, 0, 0, 0},
		Green:   {0, 255, 0, 0},
		Yellow:  {255, 255, 0, 0},
		Blue:    {0, 0, 255, 0},
		Magenta: {255, 0, 255, 0},
		Cyan:    {0, 255, 255, 0},
		White:   {255, 255, 255, 0},
	}
)

type Format struct {
	Fg, Bg Color
	Att    []Attribute
}

func (f Format) contains(a Attribute) bool {
	for _, aa := range f.Att {
		if a == aa {
			return true
		}
	}
	return false
}

// cssColor returns an color.Color to display to the generated html page.
func (f Format) cssColor(c Color) color.RGBA {
	rgb := colorMap[c]
	switch {
	case f.contains(Bright):
		rgb.A = alphaBright
	case f.contains(Dim):
		rgb.A = alphaDim
	default:
		rgb.A = alphaNormal
	}
	return rgb
}

func colorString(c color.RGBA) string {
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}

func (f Format) css() string {
	parts := make([]string, 0)
	if f.Fg != Default {
		parts = append(parts, "color:"+colorString(f.cssColor(f.Fg)))
	}
	if f.Bg != Default {
		parts = append(parts, "background-color:"+colorString(f.cssColor(f.Fg)))
	}
	if f.contains(Underscore) {
		parts = append(parts, "text-decoration:underline")
	}
	if f.contains(Hidden) {
		parts = append(parts, "display:none")
	}
	if f.contains(Blink) {
		parts = append(parts, "text-decoration:blink")
	}
	return strings.Join(parts, ";")
}

type Cursor struct {
	// The position of the cursor.
	Y, X int

	// The format of the text that will be emitted.
	F Format
}

type VT100 struct {
	// The width and height of the terminal. Modification only via SetDim.
	Height, Width int

	// The textual content of the terminal. Only UTF-8 encoding is currently
	// supported.
	Content [][]rune

	// The text format of each cell.
	Format [][]Format

	// Cursor is the current state of the cursor.
	Cursor Cursor

	// Err is the latest error seen while parsing the input stream.
	// This will be set, e.g., when we encounter an unknown operation,
	// or if the command stream is malformed. You may read it and quit,
	// or continue. Given the nature of terminal updates, it is possible
	// that continuing will return you to a valid state -- for example,
	// if there is a screen wipe, or through the normal course of updating
	// the screen.
	//
	// If you choose to continue, we recommend that you clear this field
	// by setting it to nil after you are done with it.
	Err error

	savedCursor Cursor
}

func NewVT100(y, x int) *VT100 {
	if y == 0 || x == 0 {
		panic(fmt.Errorf("invalid dim (%d, %d)", y, x))
	}

	v := &VT100{
		Height:  y,
		Width:   x,
		Content: make([][]rune, y),
		Format:  make([][]Format, y),
	}

	for row := 0; row < y; row++ {
		v.Content[row] = make([]rune, x)
		v.Format[row] = make([]Format, x)
	}
	return v
}

// Html renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader. We also use it in the tests.
func (v *VT100) Html() string {
	var buf bytes.Buffer

	// Iterate each row. When the css changes, close the previous span, and open
	// a new one. No need to close a span when the css is empty, we won't have
	// opened one in the past.
	css := ""
	for y, row := range v.Content {
		for x, r := range row {
			newCss := v.Format[y][x].css()
			if newCss != css {
				if css != "" {
					buf.WriteString("</span>")
				}
				if newCss != "" {
					buf.WriteString(`<span style="` + newCss + `">`)
				}
				css = newCss
			}

			if s, escapeNeeded := maybeEscapeRune(r); escapeNeeded {
				buf.WriteString(s)
			} else {
				buf.WriteRune(r)
			}
		}
		buf.WriteString("<br>")
	}
	return buf.String()
}

// maybeEscapeRune potentially escapes a rune for display in an html document.
// It only escapes the things that html.EscapeString does, but it works without allocating
// a string to hold r.
func maybeEscapeRune(r rune) (string, bool) {
	switch r {
	case '&':
		return "&amp;", true
	case '\'':
		return "&#39;", true
	case '<':
		return "&lt;", true
	case '>':
		return "&gt;", true
	case '"':
		return "&quot;", true
	}
	return "", false
}

// put puts r onto the current cursor's position, then advances the cursor.
func (v *VT100) put(r rune) {
	v.Content[v.Cursor.Y][v.Cursor.X] = r
	v.Format[v.Cursor.Y][v.Cursor.X] = v.Cursor.F
	v.advance()
}

// advance advances the cursor, wrapping to the next line if need be.
func (v *VT100) advance() {
	v.Cursor.X++
	if v.Cursor.X >= v.Width {
		v.Cursor.X = 0
		v.Cursor.Y++
	}
	if v.Cursor.Y >= v.Height {
		v.Cursor.Y = 0 // TODO(jaguilar): is this right?
	}
}

// home moves the cursor to the coordinates y x.
func (v *VT100) home(y, x int) {
	v.bounds(y, x)
	v.Cursor.Y, v.Cursor.X = y, x
}

func (v *VT100) bounds(y, x int) {
	if y < 0 || y >= v.Height || x < 0 || x >= v.Width {
		panic(fmt.Errorf("out of bounds (%d, %d)", y, x))
	}
}

// move moves the cursor according to the vector y x.
func (v *VT100) move(yy, xx int) {
	y, x := v.Cursor.Y+yy, v.Cursor.X+xx
	v.home(y, x)
}

type eraseDirection int

const (
	eraseRight eraseDirection = iota
	eraseLeft
	eraseLine
	eraseDown
	eraseUp
	eraseAll
)

// eraseTo erases from the cursor to the provided coordinate, inclusively.
// (All runes in the region are set to ' ' and all colors are returned to
// the default.)
func (v *VT100) eraseDirection(sel eraseDirection) {
	y, x := v.Cursor.Y, v.Cursor.X // Aliases for simplicity.
	maxY, maxX := v.Height-1, v.Width-1
	switch sel {
	case eraseLeft:
		v.eraseRegion(y, 0, y, x)
	case eraseRight:
		v.eraseRegion(y, x, y, maxX)
	case eraseLine:
		v.eraseRegion(y, 0, y, maxX)
	case eraseUp:
		v.eraseRegion(0, 0, y, maxX)
	case eraseDown:
		v.eraseRegion(y, 0, maxY, maxX)
	case eraseAll:
		v.eraseRegion(0, 0, maxY, maxX)
	}
}

func (v *VT100) eraseRegion(y1, x1, y2, x2 int) {
	v.bounds(y1, x1)
	v.bounds(y2, x2)

	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}

	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			v.clear(y, x)
		}
	}
}

func (v *VT100) clear(y, x int) {
	v.Content[y][x] = ' '
	v.Format[y][x] = Format{}
}

func (v *VT100) save() {
	v.savedCursor = v.Cursor
}

func (v *VT100) unsave() {
	v.Cursor = v.savedCursor
}
