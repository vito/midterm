package vt100

import (
	"fmt"
	"io"
	"strings"

	"github.com/muesli/termenv"
)

func (vt *VT100) RenderLine(w io.Writer, row int) error {
	var lastFormat Format

	line := vt.Content[row]
	for col, r := range line {
		f := vt.Format[row][col]

		if row == vt.Cursor.Y && col == vt.Cursor.X {
			f.Reverse = !vt.PagerMode
		}

		if f != lastFormat {
			lastFormat = f
			_, err := w.Write([]byte(f.Render()))
			if err != nil {
				return err
			}
		}

		_, err := w.Write([]byte(string(r)))
		if err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w, reset)
	return err
}

const reset = termenv.CSI + termenv.ResetSeq + "m"

func (f Format) Render() string {
	styles := []string{}
	if f.Fg != nil {
		styles = append(styles, f.Fg.Sequence(false))
	}
	if f.Bg != nil {
		styles = append(styles, f.Bg.Sequence(true))
	}

	switch f.Intensity {
	case Bold:
		styles = append(styles, termenv.BoldSeq)
	case Faint:
		styles = append(styles, termenv.FaintSeq)
	}

	if f.Italic {
		styles = append(styles, termenv.ItalicSeq)
	}

	if f.Underline {
		styles = append(styles, termenv.UnderlineSeq)
	}

	if f.Blink {
		styles = append(styles, termenv.BlinkSeq)
	}

	if f.Reverse {
		styles = append(styles, termenv.ReverseSeq)
	}

	if f.Conceal {
		styles = append(styles, "8")
	}

	if f.CrossOut {
		styles = append(styles, termenv.CrossOutSeq)
	}

	if f.Overline {
		styles = append(styles, termenv.OverlineSeq)
	}

	var res string
	if f.Reset || f == (Format{}) {
		res = reset
	}
	if len(styles) > 0 {
		res += fmt.Sprintf("%s%sm", termenv.CSI, strings.Join(styles, ";"))
	}
	return res
}
