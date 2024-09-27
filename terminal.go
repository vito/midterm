package midterm

import (
	"bytes"
	"io"
	"log"
	"runtime/debug"
	"sync"
)

// Terminal represents a raw terminal capable of handling VT100 and VT102 ANSI
// escape sequences, some of which are handled by forwarding them to a local or
// remote session (e.g. OSC52 copy/paste).
type Terminal struct {
	// Screen is the current screen, embedded so that Terminal is a pass-through.
	*Screen

	// Alt is either the alternate screen (if !IsAlt) or the main screen (if
	// IsAlt).
	Alt *Screen

	// IsAlt indicates whether the alt screen is active.
	IsAlt bool

	// AutoResizeY indicates whether the terminal should automatically resize
	// when the content exceeds its maximum height.
	AutoResizeY bool

	// AutoResizeX indicates that the terminal has no defined width - instead,
	// columns are dynamically allocated as text is printed, and not all rows
	// will be the same width.
	AutoResizeX bool

	// ForwardRequests is the writer to which we send requests to forward
	// to the terminal.
	ForwardRequests io.Writer

	// ForwardResponses is the writer to which we send responses to CSI/OSC queries.
	ForwardResponses io.Writer

	// Enable "raw" mode. Line endings do not imply a carriage return.
	Raw bool

	// wrap indicates that we've reached the end of the screen and need to wrap
	// to the next line if another character is printed.
	wrap bool

	// unparsed is the bytes that we have not yet parsed. It typically contains a
	// partial escape sequence.
	unparsed []byte

	// onResize is a hook called every time the terminal resizes.
	onResize OnResizeFunc

	// onScrollack is a hook called every time a line is about to be pushed out
	// of the visible screen region.
	onScrollback OnScrollbackFunc

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

// NewAutoResizingTerminal creates a new Terminal object with small initial
// dimensions, configured to automatically resize width and height as needed.
//
// This may be useful for applications that want to display dynamically sized
// content.
func NewAutoResizingTerminal() *Terminal {
	term := NewTerminal(1, 80)
	term.AutoResizeX = true
	term.AutoResizeY = true
	return term
}

// NewTerminal creates a new Terminal object with the specified dimensions. y
// and x must both be greater than zero.
//
// Each cell is set to contain a ' ' rune, and all formats are left as the
// default.
func NewTerminal(rows, cols int) *Terminal {
	v := &Terminal{
		Screen: newScreen(rows, cols),
	}

	v.reset()

	return v
}

func (v *Terminal) Reset() {
	v.mut.Lock()
	defer v.mut.Unlock()
	v.reset()
}

func (v *Terminal) UsedHeight() int {
	v.mut.Lock()
	defer v.mut.Unlock()
	return v.MaxY + 1
}

func (v *Terminal) UsedWidth() int {
	v.mut.Lock()
	defer v.mut.Unlock()
	return v.MaxX + 1
}

// Resize sets the terminal height and width to rows and cols and disables
// auto-resizing on both axes.
func (v *Terminal) Resize(rows, cols int) {
	v.mut.Lock()
	v.resize(rows, cols)

	// disable auto-resize upon manually resizing. what's the point if the new
	// size won't be respected?
	v.AutoResizeX = false
	v.AutoResizeY = false

	f := v.onResize
	v.mut.Unlock()
	if f != nil {
		f(rows, cols)
	}
}

// Resize sets the terminal width to cols and disables auto-resizing width.
func (v *Terminal) ResizeX(cols int) {
	v.mut.Lock()
	v.resize(v.Height, cols)

	// disable auto-resize upon manually resizing. what's the point if the new
	// size won't be respected?
	v.AutoResizeX = false

	f := v.onResize
	v.mut.Unlock()
	if f != nil {
		f(v.Height, cols)
	}
}

// Resize sets the terminal height to rows rows and disables auto-resizing
// height.
func (v *Terminal) ResizeY(rows int) {
	v.mut.Lock()
	v.resize(rows, v.Width)

	// disable auto-resize upon manually resizing. what's the point if the new
	// size won't be respected?
	v.AutoResizeY = false

	f := v.onResize
	v.mut.Unlock()
	if f != nil {
		f(rows, v.Width)
	}
}

type OnScrollbackFunc func(line Line)

// OnScrollback sets a hook that is called every time a line is about to be
// pushed out of the visible screen region.
func (v *Terminal) OnScrollback(f OnScrollbackFunc) {
	v.mut.Lock()
	v.onScrollback = f
	v.mut.Unlock()
}

type OnResizeFunc func(rows, cols int)

// OnResize sets a hook that is called every time the terminal resizes.
func (v *Terminal) OnResize(f OnResizeFunc) {
	f(v.Height, v.Width)
	v.mut.Lock()
	v.onResize = f
	v.mut.Unlock()
}

func (v *Terminal) resize(h, w int) {
	v.Screen.resize(h, w)
	if v.Alt != nil {
		v.Alt.resize(h, w)
	}
}

func (v *Terminal) Write(dt []byte) (n int, rerr error) {
	n = len(dt)

	if trace != nil {
		trace.Write(dt)
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("RECOVERED WRITE PANIC FOR %q: %v", string(dt), err)
			log.Writer().Write(debug.Stack())
			dbg.Printf("RECOVERED WRITE PANIC FOR %q: %v", string(dt), err)
			dbg.Writer().Write(debug.Stack())
		}
	}()

	v.mut.Lock()
	defer v.mut.Unlock()

	if len(v.unparsed) > 0 {
		dt = append(v.unparsed, dt...)
		v.unparsed = nil
	}

	buf := bytes.NewBuffer(dt)
	for buf.Len() > 0 {
		// nextLine := bytes.IndexByte(buf.Bytes(), '\n')
		// if nextLine == -1 {
		// 	nextLine = buf.Len()
		// }
		//
		cmd, unparsed, err := Decode(buf)
		if err != nil {
			dbg.Printf("LEAVING UNPARSED: %q FOR ERR: %s", string(unparsed), err)
			v.unparsed = unparsed
			break
		}

		dbg.Printf("PARSED: %+v", cmd)

		// grow before handling every command. this is a little unintuitive, but
		// the root desire is to avoid leaving a trailing blank line when ending
		// with "\n" since it wastes a row of output, but we also need to make
		// sure we actually advance before we perform any other update to avoid
		// bounds related panics.
		v.scrollOrResizeYIfNeeded()
		v.resizeXIfNeeded()

		if err := cmd.display(v); err != nil {
			dbg.Printf("DISPLAY ERR FOR %s: %v", cmd, err)
		}
	}

	return n, nil
}

// Process handles a single ANSI terminal command, updating the terminal
// appropriately.
//
// One special kind of error that this can return is an UnsupportedError. It's
// probably best to check for these and skip, because they are likely recoverable.
func (v *Terminal) Process(c Command) error {
	v.mut.Lock()
	defer v.mut.Unlock()

	return c.display(v)
}

// put puts r onto the current cursor's position, then advances the cursor.
func (v *Terminal) put(r rune) {
	if v.wrap {
		v.Changes[v.Cursor.Y]++
		v.Cursor.X = 0
		v.Cursor.Y++
		v.scrollOrResizeYIfNeeded()
		v.wrap = false
	}
	x, y := v.Cursor.X, v.Cursor.Y
	v.Content[y][x] = r
	v.Format.Paint(v.Cursor)
	if y > v.MaxY {
		// track max character offset for UsedHeight()
		v.MaxY = y
	}
	if x > v.MaxX {
		// track max character offset for UsedHeight()
		v.MaxX = x
	}
	v.advance()
}

// advance advances the cursor, wrapping to the next line if need be.
func (v *Terminal) advance() {
	if !v.AutoResizeX && v.Cursor.X == v.Width-1 {
		v.wrap = true
	} else {
		v.Cursor.X++
		v.Changes[v.Cursor.Y]++
	}
}

func (v *Terminal) resizeXIfNeeded() {
	// +1 because printing advances the cursor, and this is called before the print
	target := v.Cursor.X + 1
	row := v.Cursor.Y
	content := v.Screen.Content[row]
	if v.AutoResizeX && target > len(content) {
		spaces := make([]rune, target-len(content))
		for i := range spaces {
			spaces[i] = ' '
			v.Format.Paint(Cursor{
				X: v.Cursor.X + (i + 1),
				Y: row,
				F: v.Cursor.F,
			})
		}
		v.Screen.Content[row] = append(content, spaces...)
		v.Changes[row]++
	}
}

func (v *Terminal) scrollOrResizeYIfNeeded() {
	if v.Cursor.Y >= v.Height {
		if v.AutoResizeY {
			dbg.Println("RESIZING Y NEEDED", v.Cursor.Y, v.Height)
			v.resize(v.Cursor.Y+1, v.Width)
		} else {
			dbg.Println("SCROLLING NEEDED", v.Cursor.Y, v.Height)
			v.scrollOne()
		}
	}
}

func scrollUp[T any](arr [][]T, positions, start, end int, empty T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		return // handle invalid inputs
	}

	// for i := start; i < end-positions; i++ {
	for i := start; i < (end+1)-positions; i++ { // +1 fixes weird stuff when shell scrollback exceeds window height
		arr[i] = make([]T, len(arr[i+positions]))
		copy(arr[i], arr[i+positions])
	}

	// Fill the newly scrolled lines with blank runes
	for i := end - positions + 1; i <= end; i++ { // +1 and <= fixes last line not being cleared
		arr[i] = make([]T, len(arr[i]))
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func scrollUpShallow[T any](arr []T, positions, start, end int, init func() T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		return // handle invalid inputs
	}

	// for i := start; i < end-positions; i++ {
	for i := start; i < (end+1)-positions; i++ { // +1 fixes weird stuff when shell scrollback exceeds window height
		arr[i] = arr[i+positions]
	}

	// Fill the newly scrolled lines with blank runes
	for i := end - positions + 1; i <= end; i++ { // +1 and <= fixes last line not being cleared
		arr[i] = init()
	}
}

func scrollDown[T any](arr [][]T, positions, start, end int, empty T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		return // handle invalid inputs
	}

	// Shift all rows down
	for i := end; i >= start+positions; i-- {
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

func scrollDownShallow[T any](arr []T, positions, start, end int, init func() T) {
	if start < 0 || end > len(arr) || start >= end || positions <= 0 {
		return // handle invalid inputs
	}

	// Shift all rows down
	for i := end; i >= start+positions; i-- {
		arr[i] = arr[i-positions]
	}

	// Fill the newly scrolled lines with blank runes
	for i := start; i < start+positions; i++ {
		arr[i] = init()
	}
}

func insertLines[T any](arr [][]T, start, ps, scrollStart, scrollEnd int, empty T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Shift lines down by Ps positions starting from the start position
	for i := scrollEnd; i >= start+ps; i-- {
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

func insertLinesShallow[T any](arr []T, start, ps, scrollStart, scrollEnd int, init func() T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Shift lines down by Ps positions starting from the start position
	for i := scrollEnd; i >= start+ps; i-- {
		arr[i] = arr[i-ps]
	}

	// Fill the newly inserted lines with the empty value
	for i := start; i < start+ps; i++ {
		arr[i] = init()
	}
}

func deleteLines[T any](arr [][]T, start, ps, scrollStart, scrollEnd int, empty T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Delete Ps lines starting from the start position
	copy(
		arr[start:scrollEnd],
		arr[start+ps:],
	)

	// Fill the end lines with the empty value
	fillStart := scrollEnd - ps
	for i := fillStart + 1; i < scrollEnd+1; i++ {
		arr[i] = make([]T, len(arr[start])) // Assume all lines have the same length as the start line
		for j := range arr[i] {
			arr[i][j] = empty
		}
	}
}

func deleteLinesShallow[T any](arr []T, start, ps, scrollStart, scrollEnd int, init func() T) {
	if start < 0 || start+ps > len(arr) || ps <= 0 {
		return // handle invalid inputs
	}

	// Delete Ps lines starting from the start position
	copy(
		arr[start:scrollEnd],
		arr[start+ps:],
	)

	// Fill the end lines with the empty value
	fillStart := scrollEnd - ps
	for i := fillStart + 1; i < scrollEnd+1; i++ {
		arr[i] = init()
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

func insertEmpties[T any](arr [][]T, row, col, ps int, empty T) {
	if row < 0 || row >= len(arr) || col < 0 || col > len(arr[row]) || ps <= 0 {
		return // Return the original array if the inputs are out of bounds or invalid
	}

	// Create a slice with ps empty elements
	empties := make([]T, ps)
	for i := range empties {
		empties[i] = empty
	}

	// Insert the empties at the specified row and column
	inserted := append(arr[row][:col], append(empties, arr[row][col:]...)...)

	// clip the row to the length of the original array
	//
	// NB: we don't actually need to handle wrapping. sh for example handles that
	// automatically, by manually writing the next row and moving the cursor back
	// up
	arr[row] = inserted[:len(arr[row])]
}

func (v *Terminal) insertCharacters(n int) {
	insertEmpties(v.Content, v.Cursor.Y, v.Cursor.X, n, ' ')
	v.Format.Insert(v.Cursor, n)
	v.Changes[v.Cursor.Y]++
}

func (v *Terminal) deleteCharacters(n int) {
	v.wrap = false // delete characters resets the wrap state.
	deleteCharacters(v.Content, v.Cursor.Y, v.Cursor.X, n, ' ')
	v.Format.Delete(v.Cursor, n)
	v.Changes[v.Cursor.Y]++
}

func (v *Terminal) repeatPrecedingCharacter(n int) {
	repeatPrecedingCharacter(v.Content, v.Cursor.Y, v.Cursor.X, n)
	region := v.Format.Region(v.Cursor.Y, v.Cursor.X-1)
	if region != nil {
		for i := 0; i < n; i++ {
			v.Format.Paint(Cursor{
				Y: v.Cursor.Y,
				X: v.Cursor.X + i,
				F: region.F,
			})
		}
	}
	v.Changes[v.Cursor.Y]++
}

func (v *Terminal) eraseCharacters(n int) {
	v.wrap = false // erase characters resets the wrap state.
	eraseCharacters(v.Content, v.Cursor.Y, v.Cursor.X, n, ' ')
	for i := 0; i < n; i++ {
		v.Format.Paint(Cursor{
			Y: v.Cursor.Y,
			X: v.Cursor.X + i,
			F: v.Cursor.F,
		})
	}
	v.Changes[v.Cursor.Y]++
}

func (v *Terminal) insertLines(n int) {
	start, end := v.scrollRegion()
	if v.Cursor.Y < start || v.Cursor.Y > end {
		return
	}
	v.wrap = false
	insertLines(v.Content, v.Cursor.Y, n, start, end, ' ')
	insertLinesShallow(v.Format.Rows, v.Cursor.Y, n, start, end, func() *Region {
		return &Region{Size: v.Width, F: v.Cursor.F}
	})
	insertLinesShallow(v.Changes, v.Cursor.Y, n, start, end, func() uint64 {
		return 0
	})
}

func (v *Terminal) deleteLines(n int) {
	start, end := v.scrollRegion()
	if v.Cursor.Y < start || v.Cursor.Y > end {
		return
	}
	v.wrap = false // delete lines resets the wrap state.
	deleteLines(v.Content, v.Cursor.Y, n, start, end, ' ')
	deleteLinesShallow(v.Format.Rows, v.Cursor.Y, n, start, end, func() *Region {
		return &Region{Size: v.Width, F: v.Cursor.F}
	})
	deleteLinesShallow(v.Changes, v.Cursor.Y, n, start, end, func() uint64 {
		return 0
	})
}

func (v *Terminal) scrollDownN(n int) {
	v.wrap = false // scroll down resets the wrap state.
	start, end := v.scrollRegion()
	scrollDown(v.Content, n, start, end, ' ')
	scrollDownShallow(v.Format.Rows, n, start, end, func() *Region {
		return &Region{Size: v.Width, F: v.Cursor.F}
	})
	scrollDownShallow(v.Changes, n, start, end, func() uint64 {
		return 0
	})
}

func (v *Terminal) scrollUpN(n int) {
	if v.onScrollback != nil {
		for i := 0; i < n; i++ {
			// v.onScrollback(Line{v.Content[i], v.Format[i]})
		}
	}
	// v.wrap = false // scroll up does NOT reset the wrap state.
	start, end := v.scrollRegion()
	scrollUp(v.Content, n, start, end, ' ')
	scrollUpShallow(v.Format.Rows, n, start, end, func() *Region {
		return &Region{Size: v.Width, F: v.Cursor.F}
	})
	scrollUpShallow(v.Changes, n, start, end, func() uint64 {
		return 0
	})
}

func (v *Terminal) scrollRegion() (int, int) {
	if v.ScrollRegion == nil {
		return 0, v.Height - 1
	} else {
		return v.ScrollRegion.Start, v.ScrollRegion.End
	}
}

func (v *Terminal) scrollOne() {
	v.scrollUpN(1)
	v.Cursor.Y = v.Height - 1
}

// home moves the cursor to the coordinates y x. If y x are out of bounds, v.Err
// is set.
func (v *Terminal) home(y, x int) {
	v.Changes[v.Cursor.Y]++
	v.wrap = false // cursor movement always resets the wrap state.
	v.Cursor.Y, v.Cursor.X = y, x
	v.Changes[v.Cursor.Y]++
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
func (v *Terminal) eraseColumns(d eraseDirection) {
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
func (v *Terminal) eraseLines(d eraseDirection) {
	x, y := v.Cursor.X, v.Cursor.Y // Alias for simplicity.
	switch d {
	case eraseBack:
		v.eraseRegion(0, 0, y, x)
		if y > 0 {
			v.eraseRegion(0, 0, y-1, v.Width-1)
		}
	case eraseForward:
		v.eraseRegion(y, x, v.Height-1, v.Width-1)
		if y < v.Height-1 {
			v.eraseRegion(y+1, 0, v.Height-1, v.Width-1)
		}
	case eraseAll:
		v.eraseRegion(0, 0, v.Height-1, v.Width-1)
	}
}

func (v *Terminal) eraseRegion(y1, x1, y2, x2 int) {
	// Erasing lines and columns clears the wrap state.
	v.wrap = false
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	if x1 > x2 && x2 > 0 {
		x1, x2 = x2, x1
	}
	f := v.Cursor.F
	for y := y1; y <= y2; y++ {
		rowX2 := x2
		if rowX2 < 0 {
			// to handle dynamic width, assume "clear to end"
			rowX2 = len(v.Content[y]) - 1
		}
		for x := x1; x <= rowX2; x++ {
			v.clear(y, x, f)
		}
	}
}

func (v *Terminal) backspace() {
	v.Changes[v.Cursor.Y]++
	v.Cursor.X--
	if v.Cursor.X < 0 {
		if v.Cursor.Y == 0 {
			v.Cursor.X = 0
		} else {
			v.Cursor.Y--
			if v.AutoResizeX {
				v.Cursor.X = len(v.Content[v.Cursor.Y]) - 1
			} else {
				v.Cursor.X = v.Width - 1
			}
			v.Changes[v.Cursor.Y]++
		}
	}
}

func (v *Terminal) moveDown() {
	if v.ScrollRegion != nil && v.Cursor.Y == v.ScrollRegion.End {
		// if we're at the bottom of the scroll region, scroll it instead of
		// moving the cursor
		v.scrollUpN(1)
	} else {
		v.Changes[v.Cursor.Y]++
		v.Cursor.Y++
		v.scrollOrResizeYIfNeeded()
		v.Changes[v.Cursor.Y]++
	}
}

func (v *Terminal) moveUp() {
	if v.ScrollRegion != nil && v.Cursor.Y == v.ScrollRegion.Start {
		// if we're at the bottom of the scroll region, scroll it instead of
		// moving the cursor
		v.scrollDownN(1)
	} else {
		v.Changes[v.Cursor.Y]++
		v.Cursor.Y--
		v.Changes[v.Cursor.Y]++
	}
}

func (v *Terminal) save() {
	v.SavedCursor = v.Cursor
}

func (v *Terminal) unsave() {
	v.Cursor = v.SavedCursor
}
