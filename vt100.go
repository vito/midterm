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
	"fmt"
)

type Color int

const (
	Black Color = iota
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

type Attribute int

const (
	_                = iota
	Bright Attribute = iota
	Dim
	_
	Underscore
	Blink
	_
	Reverse
	Hidden
)

type Cursor struct {
	// The position of the cursor.
	Y, X int

	// The foreground and background colors.
	Fg, Bg Color
}

type VT100 struct {
	// The width and height of the terminal. Modification only via SetDim.
	Height, Width int

	// The textual content of the terminal. Only UTF-8 encoding is currently
	// supported.
	Content [][]rune

	// The color of the terminal. TODO(jaguilar): This defaults to white and black, but is that right?
	Foreground [][]Color
	Background [][]Color

	// Cursor is the current state of the cursor.
	Cursor Cursor

	// Err is the latest error seen while parsing the input stream.
	Err error

	savedCursor Cursor
}

// put puts r onto the current cursor's position, then advances the cursor.
func (v *VT100) put(r rune) {
	v.Content[v.Cursor.Y][v.Cursor.X] = r
	v.Foreground[v.Cursor.Y][v.Cursor.X] = v.Cursor.Fg
	v.Background[v.Cursor.Y][v.Cursor.X] = v.Cursor.Bg
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
	v.Foreground[y][x] = White
	v.Background[y][x] = Black
}

func (v *VT100) save() {
	v.savedCursor = v.Cursor
}

func (v *VT100) unsave() {
	v.Cursor = v.savedCursor
}
