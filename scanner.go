package vt100

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"unicode"
)

var (
	_ = log.Print
)

// scanner is a type that helps us turn a stream of terminal commands into
// command objects that can act on our terminal.
type scanner struct {
	reader *bufio.Reader
}

func newScanner(r io.Reader) scanner {
	return scanner{reader: bufio.NewReaderSize(r, 256)}
}

// next advances to the next command in the stream. It will return the command
// if it finds one. If it reaches eof, the command will be nil, as will the error.
// Only if an actual error condition is observed will error be set.
func (s *scanner) next() (command, error) {
	r, size, err := s.reader.ReadRune()
	if err != nil {
		return nil, err
	}

	if r == unicode.ReplacementChar && size == 1 {
		return nil, fmt.Errorf("non-utf8 data from reader")
	}

	if r == escape || r == unicodeCsi { // At beginning of escape sequence.
		s.reader.UnreadRune()
		return s.scanEscapeSequence()
	}

	// TODO(jaguilar): handle other control codes.

	return putRuneCommand(r), nil
}

const (
	escape     = '\u001b'
	unicodeCsi = '\u009b'
)

var (
	csEnd = &unicode.RangeTable{R16: []unicode.Range16{{Lo: 64, Hi: 126, Stride: 1}}}
)

// scanEscapeSequence scans to the end of the current escape sequence. The first
// character
func (s *scanner) scanEscapeSequence() (command, error) {
	csi := false
	esc, _, err := s.reader.ReadRune()
	if err != nil {
		return nil, err
	}
	if esc != escape && esc != unicodeCsi {
		panic(fmt.Errorf("not positioned at beginning of escape sequence, saw: %r", esc))
	}
	if esc == unicodeCsi {
		csi = true
	}

	var cmd bytes.Buffer
	quote := false
	for i := 0; ; i++ {
		r, _, err := s.reader.ReadRune()
		if err != nil {
			return nil, err
		}
		if i == 0 && r == '[' {
			csi = true
			continue
		}

		if !csi {
			return newCSCommand(r, ""), nil
		} else if quote == false && unicode.Is(csEnd, r) {
			return newCSCommand(r, cmd.String()), nil
		}

		if r == '"' {
			quote = !quote
		}

		// Otherwise, we're still in the args of the command, and this rune is one of
		// those args.
		if _, err := cmd.WriteRune(r); err != nil {
			panic(err) // WriteRune cannot return an error from bytes.Buffer.
		}
	}
}
