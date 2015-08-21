package vt100

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
)

// scanner is a type that takes a stream of bytes destined for a terminal
// and turns it into commands that update our terminal screen reader.
type scanner struct {
	io.RuneScanner
}

// next advances to the next command in the stream. It will return the command
// if it finds one. If it receives any error, that error will be returned instead.
// It is also an error if the input stream contains non-UTF8 data.
func readOneCommand(s io.RuneScanner) (command, error) {
	r, size, err := s.ReadRune()
	if err != nil {
		return nil, err
	}

	if r == unicode.ReplacementChar && size == 1 {
		return nil, fmt.Errorf("non-utf8 data from reader")
	}

	if r == escape || r == monogramCsi { // At beginning of escape sequence.
		s.UnreadRune()
		return scanEscapeCommand(s)
	}

	if unicode.IsControl(r) {
		return controlCommand(r), nil
	}

	return runeCommand(r), nil
}

const (
	// There are two ways to begin an escape sequence. One is to put the escape byte.
	// The other is to put the single-rune control sequence indicator, which is equivalent
	// to putting "\u001b[".
	escape      = '\u001b'
	monogramCsi = '\u009b'
)

var (
	csEnd = &unicode.RangeTable{R16: []unicode.Range16{{Lo: 64, Hi: 126, Stride: 1}}}
)

// scanEscapeCommand scans to the end of the current escape sequence. The scanner
// must be positioned at an escape rune (esc or the unicode CSI).
func scanEscapeCommand(s io.RuneScanner) (command, error) {
	csi := false
	esc, _, err := s.ReadRune()
	if err != nil {
		return nil, err
	}
	if esc != escape && esc != monogramCsi {
		panic(fmt.Errorf("not positioned at beginning of escape sequence, saw: %v", esc))
	}
	if esc == monogramCsi {
		csi = true
	}

	var args bytes.Buffer
	quote := false
	for i := 0; ; i++ {
		r, _, err := s.ReadRune()
		if err != nil {
			return nil, err
		}
		if i == 0 && r == '[' {
			csi = true
			continue
		}

		if !csi {
			return escapeCommand{r, ""}, nil
		} else if quote == false && unicode.Is(csEnd, r) {
			return escapeCommand{r, args.String()}, nil
		}

		if r == '"' {
			quote = !quote
		}

		// Otherwise, we're still in the args, and this rune is one of those args.
		if _, err := args.WriteRune(r); err != nil {
			panic(err) // WriteRune cannot return an error from bytes.Buffer.
		}
	}
}
