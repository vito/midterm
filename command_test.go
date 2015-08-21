package vt100

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func splitLines(s string) [][]rune {
	ss := strings.Split(s, "\n")
	r := make([][]rune, len(ss))
	for i, line := range ss {
		r[i] = []rune(line)
	}
	return r
}

func esc(s string) string {
	return "\u001b" + s
}

func cmd(s string) command {
	cmd, err := newScanner(strings.NewReader(s)).next()
	if err != nil {
		panic(err)
	}
	return cmd
}

func TestPutRune(t *testing.T) {
	v := fromLines("abc\ndef\nghi")
	v.Cursor.Y = 1
	v.Cursor.X = 1
	runeCommand('z').display(v)

	assert.Nil(t, v.Err)
	assert.Equal(t, splitLines("abc\ndzf\nghi"), v.Content)
	assert.Equal(t, 2, v.Cursor.X)
	assert.Equal(t, 1, v.Cursor.Y)
}

func TestMoveCursor(t *testing.T) {
	v := fromLines("abc\ndef\nghi")
	cmd(esc("[3;1H")).display(v)
	assert.Equal(t, Cursor{Y: 2, X: 0}, v.Cursor)
}

func TestCursorDirections(t *testing.T) {
	v := fromLines("abc\ndef\nghi")

	moves := strings.Join([]string{
		esc("[2B"), // down down
		esc("[2C"), // right right
		esc("[A"),  // up (no args = 1)
		esc("[1D"), // left
	}, "") // End state: +1, +1
	s := newScanner(strings.NewReader(moves))

	want := []Cursor{
		{Y: 2, X: 0},
		{Y: 2, X: 2},
		{Y: 1, X: 2},
		{Y: 1, X: 1},
	}
	got := make([]Cursor, 0)

	cmd, err := s.next()
	for err == nil {
		cmd.display(v)
		got = append(got, v.Cursor)
		cmd, err = s.next()
	}
	if assert.Equal(t, err, io.EOF) {
		assert.Equal(t, want, got)
	}
}

func TestErase(t *testing.T) {
	c := Format{Fg: Yellow, Intensity: Bright}
	d := DefaultFormat
	for _, tc := range []struct {
		command command
		want    *VT100
	}{
		{cmd(esc("[K")), fromLinesAndFormats("abcd\nef  \nijkl", [][]Format{
			{c, c, c, c},
			{c, c, d, d},
			{c, c, c, c},
		})},
		{cmd(esc("[1K")), fromLinesAndFormats("abcd\n   h\nijkl", [][]Format{
			{c, c, c, c},
			{d, d, d, c},
			{c, c, c, c},
		})},
		{cmd(esc("[2K")), fromLinesAndFormats("abcd\n    \nijkl", [][]Format{
			{c, c, c, c},
			{d, d, d, d},
			{c, c, c, c},
		})},
		{cmd(esc("[J")), fromLinesAndFormats("abcd\n    \n    ", [][]Format{
			{c, c, c, c},
			{d, d, d, d},
			{d, d, d, d},
		})},
		{cmd(esc("[1J")), fromLinesAndFormats("    \n    \nijkl", [][]Format{
			{d, d, d, d},
			{d, d, d, d},
			{c, c, c, c},
		})},
		{cmd(esc("[2J")), fromLinesAndFormats("    \n    \n    ", [][]Format{
			{d, d, d, d},
			{d, d, d, d},
			{d, d, d, d},
		})},
	} {
		v := fromLinesAndFormats(
			"abcd\nefgh\nijkl", [][]Format{
				{c, c, c, c},
				{c, c, c, c},
				{c, c, c, c},
			})
		v.Cursor = Cursor{Y: 1, X: 2}
		beforeCursor := v.Cursor

		tc.command.display(v)

		assert.Equal(t, tc.want.Content, v.Content, "while evaluating ", tc.command)
		assert.Equal(t, tc.want.Format, v.Format, "while evaluating ", tc.command)
		// Check the cursor separately. We don't set it on any of the test cases
		// so we cannot expect it to be equal. It's not meant to change.
		assert.Equal(t, beforeCursor, v.Cursor)
	}
}

func TestBackspace(t *testing.T) {
	v := fromLines("BA..")
	v.Cursor.Y, v.Cursor.X = 0, 2

	controlCommand(backspace).display(v)
	assert.Equal(t, fromLines("B ..").Content, v.Content)

	v.Cursor.X = 0
	controlCommand(backspace).display(v)
	assert.Equal(t, 0, v.Cursor.X)

	v = fromLines("..\n..")
	v.Cursor.Y, v.Cursor.X = 1, 0
	controlCommand(backspace).display(v)
	assert.Equal(t, 0, v.Cursor.Y)
	assert.Equal(t, 1, v.Cursor.X)
}

func TestLineFeed(t *testing.T) {
	v := fromLines("AA\n..")
	v.Cursor.X = 1
	controlCommand(linefeed).display(v)
	runeCommand('b').display(v)
	assert.Equal(t, fromLines("AA\nb.").Content, v.Content)
}

func TestCarriageReturn(t *testing.T) {
	v := fromLines("AA\n..")
	v.Cursor.X = 1
	controlCommand(carriageReturn).display(v)
	runeCommand('b').display(v)
	assert.Equal(t, fromLines("bA\n..").Content, v.Content)
}

func TestAttributes(t *testing.T) {
	v := fromLines("....")
	s := newScanner(strings.NewReader(
		esc("[2ma") + esc("[5;22;31mb") + esc("[0mc") + esc("[4;46md")))
	cmd, err := s.next()
	for err == nil {
		cmd.display(v)
		cmd, err = s.next()
	}
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, []rune("abcd"), v.Content[0])
	assert.Equal(t, []Format{
		{Intensity: Dim}, {Blink: true, Fg: Red}, {}, {Underscore: true, Bg: Cyan},
	}, v.Format[0])
}
