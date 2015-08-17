package vt100

import (
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

func esc(s string) command {
	cmd, err := newScanner(strings.NewReader("\u001b" + s)).next()
	if err != nil {
		panic(err)
	}
	return cmd
}

func TestPutRune(t *testing.T) {
	v := fromLines("abc\ndef\nghi")
	v.Cursor.Y = 1
	v.Cursor.X = 1
	putRuneCommand('z').display(v)

	assert.Nil(t, v.Err)
	assert.Equal(t, v.Content, splitLines("abc\ndzf\nghi"))
	assert.Equal(t, v.Cursor.X, 2)
	assert.Equal(t, v.Cursor.Y, 1)
}

func TestMoveCursor(t *testing.T) {
	v := fromLines("abc\ndef\nghi")
	esc("[2;0H").display(v)
	assert.Equal(t, v.Cursor, Cursor{Y: 2, X: 0})
}

func TestCursorDirections(t *testing.T) {
	v := fromLines("abc\ndef\nghi")

	for _, x := range []string {
		"[1B",  // down
		"[2C",  // right right
		"[A",  // up -- no argument = distance 1.
		"[1D"
	}
}
