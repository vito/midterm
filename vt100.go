// package vt100 acts as a vt100 terminal emulator. For example,
// you could use it to run a program like nethack that expects
// a terminal as a subprocess. It tracks the position of the cursor,
// colors, and various other aspects of the terminal's state, and
// allows you to inspect them. Currently, we only handle raw mode,
// no scrolling or prompt features.
package vt100

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"sort"
	"strings"
	"sync"
)

type Intensity int

const (
	Normal Intensity = 0
	Bright           = 1
	Dim              = 2
	// TODO(jaguilar): Should this be in a subpackage, since the names are pretty collide-y?
)

var (
	// Technically RGBAs are supposed to be premultiplied. But CSS doesn't expect them
	// that way, so we won't do it in this file.
	DefaultColor = color.RGBA{0, 0, 0, 0}
	// Our black has 255 alpha, so it will compare negatively with DefaultColor.
	Black   = color.RGBA{0, 0, 0, 255}
	Red     = color.RGBA{255, 0, 0, 255}
	Green   = color.RGBA{0, 255, 0, 255}
	Yellow  = color.RGBA{255, 255, 0, 255}
	Blue    = color.RGBA{0, 0, 255, 255}
	Magenta = color.RGBA{255, 0, 255, 255}
	Cyan    = color.RGBA{0, 255, 255, 255}
	White   = color.RGBA{255, 255, 255, 255}
)

func (i Intensity) alpha() uint8 {
	switch i {
	case Bright:
		return 255
	case Normal:
		return 170
	case Dim:
		return 85
	default:
		panic(fmt.Errorf("unknown intensity: %d", uint8(i)))
	}
}

// Format represents the display format of text on a terminal.
type Format struct {
	// Fg is the foreground color.
	Fg color.RGBA
	// Bg is the background color.
	Bg color.RGBA
	// Intensity is the text intensity (bright, normal, dim).
	Intensity Intensity
	// Various text properties.
	Underscore, Conceal, Negative, Blink, Inverse bool
}

func toCss(c color.RGBA) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %f)", c.R, c.G, c.B, float32(c.A)/255)
}

func (f Format) css() string {
	parts := make([]string, 0)
	fg, bg := f.Fg, f.Bg
	if f.Inverse {
		bg, fg = fg, bg
	}

	if f.Intensity != Normal {
		// Intensity only applies to the text -- i.e., the foreground.
		fg.A = f.Intensity.alpha()
	}

	if fg != DefaultColor {
		parts = append(parts, "color:"+toCss(fg))
	}
	if bg != DefaultColor {
		parts = append(parts, "background-color:"+toCss(bg))
	}
	if f.Underscore {
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

// Cursor represents both the position and text type of the cursor.
type Cursor struct {
	// Y and X are the coordinates.
	Y, X int

	// F is the format that will be displayed.
	F Format
}

// VT100 represents a simplified, raw VT100 terminal.
type VT100 struct {
	// Height and Width are the dimensions of the terminal.
	Height, Width int

	// Content is the text in the terminal.
	Content [][]rune

	// Format is the display properties of each cell.
	Format [][]Format

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

	// savedCursor is the state of the cursor last time save() was called.
	savedCursor Cursor

	// TODO(jaguilar): remove.
	mu sync.Mutex
}

// NewVT100 creates a new VT100 object with the specified dimensions. y and x
// must both be greater than zero.
//
// Each cell is set to contain a ' ' rune, and all formats are left as the
// default.
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

		for col := 0; col < x; col++ {
			v.clear(row, col)
		}
	}
	return v
}

// UpdateFrom reads r for updates until EOF or other error. If the error
// is not io.EOF, v.Err will be set.
func (v *VT100) UpdateFrom(r io.Reader) {
	// TODO(jaguilar): Figure out what interface we really want here. There
	// will need to be some concept of "idleness" for the terminal for our purposes.
	// not sure if that should be handled here or outside.
	s := newScanner(r)
	for {
		cmd, err := s.next()
		if err != nil {
			if err != io.EOF {
				v.Err = err
			}
			return
		}

		v.mu.Lock() // TODO(jaguilar): remove.
		cmd.display(v)
		v.mu.Unlock()
	}
}

// HTML renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader.
func (v *VT100) HTML() string {
	var buf bytes.Buffer
	buf.WriteString(`<pre style="color:white;background-color:black;">`)

	// Iterate each row. When the css changes, close the previous span, and open
	// a new one. No need to close a span when the css is empty, we won't have
	// opened one in the past.
	var lastFormat Format
	for y, row := range v.Content {
		for x, r := range row {
			f := v.Format[y][x]
			if f != lastFormat {
				if lastFormat != (Format{}) {
					buf.WriteString("</span>")
				}
				if f != (Format{}) {
					buf.WriteString(`<span style="` + f.css() + `">`)
				}
				lastFormat = f
			}
			if s := maybeEscapeRune(r); s != "" {
				buf.WriteString(s)
			} else {
				buf.WriteRune(r)
			}
		}
		buf.WriteRune('\n')
	}
	buf.WriteString("</pre>")

	return buf.String()
}

// maybeEscapeRune potentially escapes a rune for display in an html document.
// It only escapes the things that html.EscapeString does, but it works without allocating
// a string to hold r. Returns an empty string if there is no need to escape.
func maybeEscapeRune(r rune) string {
	switch r {
	case '&':
		return "&amp;"
	case '\'':
		return "&#39;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&quot;"
	}
	return ""
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
		// TODO(jaguilar): if we implement scroll, this should probably scroll.
		v.Cursor.Y = 0
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

// eraseDirection is the logical direction in which an erase command happens,
// from the cursor. For both erase commands, forward is 0, backward is 1,
// and everything is 2.
type eraseDirection int

const (
	// From the cursor to the end, inclusive.
	eraseForward eraseDirection = iota

	// From the beginning to the cursor, inclusive.
	eraseBack

	// Everything.
	eraseAll
)

// eraseColumns erases columns from the current line.
func (v *VT100) eraseColumns(d eraseDirection) {
	y, x := v.Cursor.Y, v.Cursor.X // Aliases for simplicity.
	switch d {
	case eraseBack:
		v.eraseRegion(y, 0, y, x)
	case eraseForward:
		v.eraseRegion(y, x, y, v.Width-1)
	case eraseAll:
		v.eraseRegion(y, 0, y, v.Width-1)
	}
}

// eraseLines erases lines from the current terminal. Note that
// no matter what is selected, the entire current line is erased.
func (v *VT100) eraseLines(d eraseDirection) {
	y := v.Cursor.Y // Alias for simplicity.
	switch d {
	case eraseBack:
		v.eraseRegion(0, 0, y, v.Width-1)
	case eraseForward:
		v.eraseRegion(y, 0, v.Height-1, v.Width-1)
	case eraseAll:
		v.eraseRegion(0, 0, v.Height-1, v.Width-1)
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

func (v *VT100) backspace() {
	v.Cursor.X--
	if v.Cursor.X < 0 {
		if v.Cursor.Y == 0 {
			v.Cursor.X = 0
		} else {
			v.Cursor.Y--
			v.Cursor.X = v.Width - 1
		}
	}
}

func (v *VT100) save() {
	v.savedCursor = v.Cursor
}

func (v *VT100) unsave() {
	v.Cursor = v.savedCursor
}
