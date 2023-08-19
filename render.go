package vt100

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

func (vt *VT100) Render(w io.Writer) error {
	for i := 0; i < vt.Height; i++ {
		err := vt.RenderLine(w, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vt *VT100) RenderLine(w io.Writer, row int) error {
	var lastFormat Format

	line := vt.Content[row]
	for col, r := range line {
		f := vt.Format[row][col]

		if vt.CursorVisible && row == vt.Cursor.Y && col == vt.Cursor.X {
			f.Reverse = vt.CursorBlinkEpoch == nil ||
				int(time.Since(*vt.CursorBlinkEpoch).Seconds())%2 == 0
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

	_, err := fmt.Fprintln(w, resetSeq)
	return err
}

const resetSeq = termenv.CSI + termenv.ResetSeq + "m"

func brighten(color termenv.Color) termenv.Color {
	if ansi, ok := color.(termenv.ANSIColor); ok && ansi < termenv.ANSIBrightBlack {
		return ansi + termenv.ANSIBrightBlack
	} else {
		return color
	}
}

func (f Format) Render() string {
	styles := []string{}

	switch f.Intensity {
	case Bold:
		styles = append(styles, termenv.BoldSeq)
		f.Fg = brighten(f.Fg)
	case Faint:
		styles = append(styles, termenv.FaintSeq)
	}

	if f.Fg != nil {
		styles = append(styles, f.Fg.Sequence(false))
	}
	if f.Bg != nil {
		styles = append(styles, f.Bg.Sequence(true))
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
		res = resetSeq
	}
	if len(styles) > 0 {
		res += fmt.Sprintf("%s%sm", termenv.CSI, strings.Join(styles, ";"))
	}
	return res
}
