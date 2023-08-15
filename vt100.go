// package vt100 implements a quick-and-dirty programmable ANSI terminal emulator.
//
// You could, for example, use it to run a program like nethack that expects
// a terminal as a subprocess. It tracks the position of the cursor,
// colors, and various other aspects of the terminal's state, and
// allows you to inspect them.
//
// We do very much mean the dirty part. It's not that we think it might have
// bugs. It's that we're SURE it does. Currently, we only handle raw mode, with no
// cooked mode features like scrolling. We also misinterpret some of the control
// codes, which may or may not matter for your purpose.
package vt100

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sync"
)

// VT100 represents a simplified, raw VT100 terminal.
type VT100 struct {
	// Height and Width are the dimensions of the terminal.
	Height, Width int

	// Content is the text in the terminal.
	Content [][]rune

	// Format is the display properties of each cell.
	Format [][]Format

	// Cursor is the current state of the cursor.
	Cursor Cursor

	// AutoResizeY indicates whether the terminal should automatically resize
	// when the content exceeds its maximum height.
	AutoResizeY bool

	// AutoResizeX indicates whether the terminal should automatically resize
	// when the content exceeds its maximum width.
	AutoResizeX bool

	// ScrollRegion is the region of the terminal that is scrollable. If it is
	// nil, the entire terminal is scrollable.
	//
	// This value is set by the CSI ; Ps ; Ps r command.
	ScrollRegion *ScrollRegion

	// DebugLogs is a location to print ANSI parse errors and other debugging
	// information.
	DebugLogs io.Writer

	// savedCursor is the state of the cursor last time save() was called.
	savedCursor Cursor

	unparsed []byte

	// maxY is the maximum vertical offset that a character was printed
	maxY int

	// for synchronizing e.g. writes and async resizing
	mut sync.Mutex
}

// Cursor represents both the position and text type of the cursor.
type Cursor struct {
	// Y and X are the coordinates.
	Y, X int

	// F is the format that will be displayed.
	F Format
}

// ScrollRegion represents a region of the terminal that is
// scrollable.
type ScrollRegion struct {
	Start, End int
}

// NewVT100 creates a new VT100 object with the specified dimensions. y and x
// must both be greater than zero.
//
// Each cell is set to contain a ' ' rune, and all formats are left as the
// default.
func NewVT100(rows, cols int) *VT100 {
	if rows <= 0 || cols <= 0 {
		panic(fmt.Errorf("invalid dim (%d, %d)", rows, cols))
	}

	v := &VT100{
		Height: rows,
		Width:  cols,

		// start at -1 so there's no "used" height until first write
		maxY: -1,
	}

	v.Reset()

	return v
}

func (v *VT100) Reset() {
	v.Content = make([][]rune, v.Height)
	v.Format = make([][]Format, v.Height)
	for row := 0; row < v.Height; row++ {
		v.Content[row] = make([]rune, v.Width)
		v.Format[row] = make([]Format, v.Width)
		for col := 0; col < v.Width; col++ {
			v.Content[row][col] = ' '
		}
	}
	v.Cursor.X = 0
	v.Cursor.Y = 0
}

func (v *VT100) UsedHeight() int {
	v.mut.Lock()
	defer v.mut.Unlock()
	return v.maxY + 1
}

func (v *VT100) Resize(h, w int) {
	v.mut.Lock()
	defer v.mut.Unlock()
	v.resize(h, w)
}

func (v *VT100) resize(h, w int) {
	if h > v.Height {
		n := h - v.Height
		for row := 0; row < n; row++ {
			v.Content = append(v.Content, make([]rune, v.Width))
			v.Format = append(v.Format, make([]Format, v.Width))
			for col := 0; col < v.Width; col++ {
				v.clear(v.Height+row, col, Format{})
			}
		}
		v.Height = h
	} else if h < v.Height {
		v.Content = v.Content[:h]
		v.Format = v.Format[:h]
		v.Height = h
	}

	if h < v.maxY {
		v.maxY = h - 1
	}

	if w > v.Width {
		for i := range v.Content {
			row := make([]rune, w)
			copy(row, v.Content[i])
			v.Content[i] = row
			format := make([]Format, w)
			copy(format, v.Format[i])
			v.Format[i] = format
			for j := v.Width; j < w; j++ {
				v.clear(i, j, Format{})
			}
		}
		v.Width = w
	} else if w < v.Width {
		for i := range v.Content {
			v.Content[i] = v.Content[i][:w]
			v.Format[i] = v.Format[i][:w]
		}
		v.Width = w
	}

	if v.Cursor.X >= v.Width {
		v.Cursor.X = v.Width - 1
	}
}

func (v *VT100) Write(dt []byte) (int, error) {
	v.mut.Lock()
	defer v.mut.Unlock()

	n := len(dt)
	if len(v.unparsed) > 0 {
		dt = append(v.unparsed, dt...)
		v.unparsed = nil
	}

	buf := bytes.NewBuffer(dt)
	for buf.Len() > 0 {
		cmd, unparsed, err := Decode(buf)
		if err != nil {
			log.Printf("!!! LEAVING UNPARSED: %q", string(unparsed))
			v.unparsed = []byte(string(unparsed))
			break
		}

		log.Println("DISPLAY", cmd)

		if err := cmd.display(v); err != nil {
			if v.DebugLogs != nil {
				fmt.Fprintln(v.DebugLogs, err)
			}
		}
	}

	return n, nil
}

// Process handles a single ANSI terminal command, updating the terminal
// appropriately.
//
// One special kind of error that this can return is an UnsupportedError. It's
// probably best to check for these and skip, because they are likely recoverable.
// Support errors are exported as expvars, so it is possibly not necessary to log
// them. If you want to check what's failed, start a debug http server and examine
// the vt100-unsupported-commands field in /debug/vars.
func (v *VT100) Process(c Command) error {
	v.mut.Lock()
	defer v.mut.Unlock()

	return c.display(v)
}

// HTML renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader.
func (v *VT100) HTML() string {
	v.mut.Lock()
	defer v.mut.Unlock()

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
	if v.Cursor.Y > v.maxY {
		// track max character offset for UsedHeight()
		v.maxY = v.Cursor.Y
	}

	v.scrollOrResizeYIfNeeded()
	v.resizeXIfNeeded()
	row := v.Content[v.Cursor.Y]
	row[v.Cursor.X] = r
	rowF := v.Format[v.Cursor.Y]
	rowF[v.Cursor.X] = v.Cursor.F
	v.advance()
}

// advance advances the cursor, wrapping to the next line if need be.
func (v *VT100) advance() {
	v.Cursor.X++
	if v.Cursor.X >= v.Width && !v.AutoResizeX {
		v.Cursor.X = 0
		v.Cursor.Y++
	}
}

func (v *VT100) resizeXIfNeeded() {
	if v.AutoResizeX && v.Cursor.X+1 >= v.Width {
		v.resize(v.Height, v.Cursor.X+1)
	}
}

func (v *VT100) scrollOrResizeYIfNeeded() {
	if v.Cursor.Y >= v.Height {
		if v.AutoResizeY {
			v.resize(v.Cursor.Y+1, v.Width)
		} else {
			// log.Println("SCROLLONE", v.Cursor.Y, v.Height)
			v.scrollOne()
		}
	}
}

func scrollUp[T any](arr [][]T, positions, start, end int, empty T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		panic("invalid scrollUp inputs")
		return // handle invalid inputs
	}

	for i := start; i < end-positions; i++ {
		arr[i] = make([]T, len(arr[i+positions]))
		copy(arr[i], arr[i+positions])
	}

	// Fill the newly scrolled lines with blank runes
	for i := end - positions; i < end; i++ {
		arr[i] = make([]T, len(arr[i]))
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func scrollDown[T any](arr [][]T, positions, start, end int, empty T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		panic("invalid scrollDown inputs")
		return // handle invalid inputs
	}

	for i := end - 1; i >= start+positions; i-- {
		arr[i] = make([]T, len(arr[i-positions]))
		copy(arr[i], arr[i-positions])
	}

	// Fill the newly scrolled lines with blank runes
	for i := start; i < start+positions; i++ {
		arr[i] = make([]T, len(arr[i]))
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func insertLines[T any](arr [][]T, start, ps int, empty T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Shift lines down by Ps positions starting from the start position
	for i := len(arr) - 1; i >= start+ps; i-- {
		arr[i] = arr[i-ps]
	}

	// Fill the newly inserted lines with the empty value
	for i := start; i < start+ps; i++ {
		arr[i] = make([]T, len(arr[i]))
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func deleteLines[T any](arr [][]T, start, ps int, empty T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Delete Ps lines starting from the start position
	copy(arr[start:], arr[start+ps:])

	// Fill the end lines with the empty value
	for i := len(arr) - ps; i < len(arr); i++ {
		arr[i] = make([]T, len(arr[i]))
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func eraseCharacters[T any](arr [][]T, row, col, ps int, empty T) {
	if row < 0 || row >= len(arr) || col < 0 || col+ps > len(arr[row]) {
		return // handle invalid inputs
	}

	if ps <= 0 {
		ps = 1 // if Ps is 0 or negative, erase one character
	}

	// Replace Ps characters with the empty value starting from the given position
	for i := col; i < col+ps; i++ {
		arr[row][i] = empty
	}
}

func repeatPrecedingCharacter[T any](arr [][]T, row, col, ps int) {
	if row < 0 || row >= len(arr) || col <= 0 || col-1 >= len(arr[row]) || ps < 0 {
		return // handle invalid inputs
	}

	charToRepeat := arr[row][col-1]

	if ps == 0 {
		ps = 1 // if Ps is 0, repeat the character once
	}

	// Repeat the preceding character Ps times starting from the current column
	for i := 0; i < ps && col+i < len(arr[row]); i++ {
		arr[row][col+i] = charToRepeat
	}
}

func deleteCharacters[T any](arr [][]T, row, col, ps int, empty T) {
	if row < 0 || row >= len(arr) || col < 0 || col >= len(arr[row]) || ps < 0 {
		return // handle invalid inputs
	}

	// Calculate the actual number of characters to delete, so it doesn't exceed the available space
	actualPs := ps
	if actualPs == 0 {
		actualPs = 1 // if Ps is 0, delete one character
	}
	if col+actualPs > len(arr[row]) {
		actualPs = len(arr[row]) - col
	}

	// Shift characters to the left by Ps positions starting from the given column
	copy(arr[row][col:], arr[row][col+actualPs:])

	// Fill the end characters with the empty value
	for i := len(arr[row]) - ps; i < len(arr[row]); i++ {
		arr[row][i] = empty
	}
}

func (v *VT100) deleteCharacters(n int) {
	deleteCharacters(v.Content, v.Cursor.Y, v.Cursor.X, n, ' ')
	deleteCharacters(v.Format, v.Cursor.Y, v.Cursor.X, n, Format{})
}

func (v *VT100) repeatPrecedingCharacter(n int) {
	repeatPrecedingCharacter(v.Content, v.Cursor.Y, v.Cursor.X, n)
	repeatPrecedingCharacter(v.Format, v.Cursor.Y, v.Cursor.X, n)
}

func (v *VT100) eraseCharacters(n int) {
	eraseCharacters(v.Content, v.Cursor.Y, v.Cursor.X, n, ' ')
	eraseCharacters(v.Format, v.Cursor.Y, v.Cursor.X, n, Format{})
}

func (v *VT100) insertLines(n int) {
	insertLines(v.Content, v.Cursor.Y, n, ' ')
	insertLines(v.Format, v.Cursor.Y, n, Format{})
}

func (v *VT100) deleteLines(n int) {
	deleteLines(v.Content, v.Cursor.Y, n, ' ')
	deleteLines(v.Format, v.Cursor.Y, n, Format{})
}

func (v *VT100) scrollDownN(n int) {
	start, end := v.scrollRegion()
	scrollDown(v.Content, n, start, end, ' ')
	scrollDown(v.Format, n, start, end, Format{})
}

func (v *VT100) scrollUpN(n int) {
	start, end := v.scrollRegion()
	scrollUp(v.Content, n, start, end, ' ')
	scrollUp(v.Format, n, start, end, Format{})
}

func (v *VT100) scrollRegion() (int, int) {
	if v.ScrollRegion == nil {
		return 0, v.Height
	} else {
		return v.ScrollRegion.Start, v.ScrollRegion.End
	}
}

func (v *VT100) scrollOne() {
	v.scrollUpN(1)
	v.Cursor.Y = v.Height - 1
}

// home moves the cursor to the coordinates y x. If y x are out of bounds, v.Err
// is set.
func (v *VT100) home(y, x int) {
	v.Cursor.Y, v.Cursor.X = y, x
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

// eraseLines erases lines from the current terminal.
func (v *VT100) eraseLines(d eraseDirection) {
	x, y := v.Cursor.X, v.Cursor.Y // Alias for simplicity.
	switch d {
	case eraseBack:
		v.eraseRegion(0, 0, y, x)
	case eraseForward:
		v.eraseRegion(y, x, v.Height-1, v.Width-1)
	case eraseAll:
		v.eraseRegion(0, 0, v.Height-1, v.Width-1)
	}
}

func (v *VT100) eraseRegion(y1, x1, y2, x2 int) {
	// Do not sanitize or bounds-check these coordinates, since they come from the
	// programmer (me). We should panic if any of them are out of bounds.
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}

	col := v.Cursor.X - 1
	if col < 0 {
		col = 0
	}
	f := v.Format[v.Cursor.Y][col]

	for y := y1; y <= y2; y++ {
		for x := x1; x <= x2; x++ {
			v.clear(y, x, f)
		}
	}
}

func (v *VT100) clear(y, x int, format Format) {
	if y >= len(v.Content) || x >= len(v.Content[0]) {
		return
	}
	v.Content[y][x] = ' '
	v.Format[y][x] = format
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
