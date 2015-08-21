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

	"github.com/golang/glog"
)

// +gen stringer
type Intensity int

const (
	Normal Intensity = iota
	Bright           = 1
	Dim              = 2
)

// Technically RGBAs are supposed to be premultiplied. But CSS doesn't expect them
// that way, so we won't do it in this file.
var (
	BgDefault = color.RGBA{0, 0, 0, 255}
	FgDefault = color.RGBA{255, 255, 255, Normal.alpha()}
	Black     = color.RGBA{0, 0, 0, 255}
	Red       = color.RGBA{255, 0, 0, 255}
	Green     = color.RGBA{0, 255, 0, 255}
	Yellow    = color.RGBA{255, 255, 0, 255}
	Blue      = color.RGBA{0, 0, 255, 255}
	Magenta   = color.RGBA{255, 0, 255, 255}
	Cyan      = color.RGBA{0, 255, 255, 255}
	White     = color.RGBA{255, 255, 255, 255}
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

type Format struct {
	// Fg is the foreground color.
	Fg color.RGBA
	// Bg is the background color.
	Bg color.RGBA
	// Intensity is the text intensity (bright, normal, dim).
	Intensity Intensity
	// Various text properties.
	Underscore, Conceal, Negative, Blink bool
}

var zeroColor = color.RGBA{0, 0, 0, 0}

func (f Format) FgColor() color.RGBA {
	c := FgDefault
	if f.Fg != zeroColor {
		c = f.Fg
	}
	c.A = f.Intensity.alpha()
	return c
}

func toCss(c color.RGBA) string {
	return fmt.Sprintf("rgba(%d, %d, %d, %f)", c.R, c.G, c.B, float32(c.A)/255)
}

func (f Format) css() string {
	parts := make([]string, 0)
	if f.Fg != zeroColor || f.Intensity != Normal {
		parts = append(parts, "color:"+toCss(f.FgColor()))
	}
	if f.Bg != zeroColor {
		// There is no intensity funny business with the bg color. We can emit
		// it directly, since all the colors default to full opacity, and the background
		// is opaque in terminals.
		parts = append(parts, "background-color:"+toCss(f.Bg))
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

	mu sync.Mutex
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

// Updates v from r until
func (v *VT100) UpdateFrom(r io.Reader) {
	// TODO(jaguilar): Figure out what interface we really want here.

	s := newScanner(r)
	for {
		cmd, err := s.next()
		if err != nil {
			if err != io.EOF {
				glog.Info(err)
			}
			return
		}

		v.mu.Lock()
		cmd.display(v)
		v.mu.Unlock()
	}
}

// Html renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader. We also use it in the tests.
func (v *VT100) HTML() string {
	var buf bytes.Buffer
	buf.WriteString(`<pre style="color:white;background-color:black;">`)

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
		buf.WriteRune('\n')
	}
	buf.WriteString("</pre>")

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

func (v *VT100) save() {
	v.savedCursor = v.Cursor
}

func (v *VT100) unsave() {
	v.Cursor = v.savedCursor
}
