package midterm

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
)

// Decode decodes one ANSI terminal command from s.
//
// s should be connected to a client program that expects an
// ANSI terminal on the other end. It will push bytes to us that we are meant
// to intepret as terminal control codes, or text to place onto the terminal.
//
// This Command alone does not actually update the terminal. You need to pass
// it to VT100.Process().
//
// You should not share s with any other reader, because it could leave
// the stream in an invalid state.
func Decode(s io.RuneScanner) (Command, []rune, error) {
	r, size, err := s.ReadRune()
	if err != nil {
		return nil, nil, err
	}

	if r == unicode.ReplacementChar && size == 1 {
		return nil, nil, fmt.Errorf("non-utf8 data from reader")
	}

	if r == escape || r == monogramCsi { // At beginning of escape sequence.
		s.UnreadRune()
		return scanEscapeCommand(s)
	}

	if unicode.IsControl(r) {
		return controlCommand(r), nil, nil
	}

	return runeCommand(r), nil, nil
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
func scanEscapeCommand(s io.RuneScanner) (Command, []rune, error) {
	csi := false

	esc, _, err := s.ReadRune()
	if err != nil {
		return nil, nil, err
	}
	if esc != escape && esc != monogramCsi {
		return nil, nil, fmt.Errorf("invalid content")
	}
	if esc == monogramCsi {
		csi = true
	}

	// pessimistically keep track of everything we read
	unparsed := []rune{esc}

	var args bytes.Buffer
	quote := false
	for i := 0; ; i++ {
		r, _, err := s.ReadRune()
		if err != nil {
			return nil, unparsed, err
		}
		unparsed = append(unparsed, r)
		if i == 0 {
			switch r {
			case '[':
				csi = true
				continue

			case '(', ')', '*', '+', '-', '.', '/':
				// character sets
				l, _, err := s.ReadRune()
				if err != nil {
					return nil, unparsed, err
				}
				// typical value is B, or USASCII
				return escCommand{r, string(l)}, nil, nil

			case ']':
				// Operating System Command
				osc := ""
				for {
					ch, _, err := s.ReadRune()
					if err != nil {
						return nil, unparsed, err
					}
					unparsed = append(unparsed, ch)
					switch ch {
					case '\x07', '\x9c': // BEL, ST
						return escCommand{r, osc}, nil, nil
					case '\x1b': // possibly ST (alternate form, e.g. notcurses)
						ch, _, err := s.ReadRune()
						if err != nil {
							return nil, unparsed, err
						}
						unparsed = append(unparsed, ch)
						if ch == '\\' { // ST (\x1b\\)
							return escCommand{r, osc}, nil, nil
						}
					default:
						osc += string(ch)
					}
				}

			case '=', '>', '7', '8', 'D', 'M', 'c':
				// non-CSI; pass through
				return escCommand{r, ""}, nil, nil
			}
		}

		if !csi {
			// TODO
			dbg.Println("UNKNOWN NON CSI CMD: " + string(r))
			return csiCommand{r, ""}, nil, nil
		} else if !quote && unicode.Is(csEnd, r) {
			return csiCommand{r, args.String()}, nil, nil
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

type commandFunc func(v *Terminal) error

func (f commandFunc) display(v *Terminal) error {
	return f(v)
}
