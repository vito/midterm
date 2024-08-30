package midterm

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

func (vt *Terminal) Render(w io.Writer) error {
	vt.mut.Lock()
	defer vt.mut.Unlock()
	for i := 0; i < vt.Height; i++ {
		if i > 0 {
			fmt.Fprintln(w)
		}
		err := vt.renderLine(w, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vt *Terminal) RenderLine(w io.Writer, row int) error {
	vt.mut.Lock()
	defer vt.mut.Unlock()
	return vt.renderLine(w, row)
}

type Line struct {
	Content []rune
	Format  []Format
}

func (line Line) Display() string {
	out := ""
	var lastFormat Format
	for col, r := range line.Content {
		f := line.Format[col]
		if f != lastFormat {
			lastFormat = f
			out += f.Render()
		}
		out += string(r)
	}
	return out
}

func (vt *Terminal) renderLine(w io.Writer, row int) error {
	if row >= len(vt.Content) {
		return fmt.Errorf("line %d exceeds content height", row)
	}

	var lastFormat Format
	line := vt.Content[row]
	for col, r := range line {
		f := vt.Format[row][col]

		if vt.CursorVisible && row == vt.Cursor.Y && col == vt.Cursor.X {
			f.SetReverse(vt.CursorBlinkEpoch == nil ||
				int(time.Since(*vt.CursorBlinkEpoch).Seconds())%2 == 0)
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

	_, err := fmt.Fprint(w, resetSeq)
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

	if f.IsBold() {
		styles = append(styles, termenv.BoldSeq)
		f.Fg = brighten(f.Fg)
	} else if f.IsFaint() {
		styles = append(styles, termenv.FaintSeq)
	}

	if f.Fg != nil {
		styles = append(styles, f.Fg.Sequence(false))
	}
	if f.Bg != nil {
		styles = append(styles, f.Bg.Sequence(true))
	}

	if f.IsItalic() {
		styles = append(styles, termenv.ItalicSeq)
	}

	if f.IsUnderline() {
		styles = append(styles, termenv.UnderlineSeq)
	}

	if f.IsBlink() {
		styles = append(styles, termenv.BlinkSeq)
	}

	if f.IsReverse() {
		styles = append(styles, termenv.ReverseSeq)
	}

	if f.IsConceal() {
		styles = append(styles, "8")
	}

	// if f.IsCrossOut() {
	// 	styles = append(styles, termenv.CrossOutSeq)
	// }
	//
	// if f.IsOverline() {
	// 	styles = append(styles, termenv.OverlineSeq)
	// }
	//
	var res string
	if f.IsReset() || f == (Format{}) {
		res = resetSeq
	}
	if len(styles) > 0 {
		res += fmt.Sprintf("%s%sm", termenv.CSI, strings.Join(styles, ";"))
	}
	return res
}
