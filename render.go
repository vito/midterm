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

var ReverseFormat = Format{Properties: ReverseBit}

func (vt *Terminal) renderLine(w io.Writer, row int) error {
	if row >= len(vt.Content) {
		return fmt.Errorf("line %d exceeds content height", row)
	}

	var pos int
	lastFormat := EmptyFormat
	for region := range vt.Format.Regions(row) {
		if region.F != lastFormat {
			lastFormat = region.F
			fmt.Fprint(w, region.F.Render())
		}
		line := vt.Content[row]
		if vt.CursorVisible && row == vt.Cursor.Y && vt.Cursor.X >= pos && vt.Cursor.X < pos+region.Size &&
			(vt.CursorBlinkEpoch == nil || int(time.Since(*vt.CursorBlinkEpoch).Seconds())%2 == 0) {
			fmt.Fprint(w, string(line[:vt.Cursor.X-pos]))
			fmt.Fprint(w, "\x1b[7m")
			fmt.Fprint(w, string(line[vt.Cursor.X]))
			fmt.Fprint(w, "\x1b[27m")
			fmt.Fprint(w, string(line[vt.Cursor.X+1:pos+region.Size]))
		} else {
			content := string(line[pos : pos+region.Size])
			fmt.Fprint(w, content)
		}
		// fmt.Fprintf(w, "region: %q", string(vt.Content[row][x:x+region.Size]))
		pos += region.Size
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
