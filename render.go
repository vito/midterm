package midterm

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

func (vt *Terminal) Render(w io.Writer) error {
	return vt.RenderFgBg(w, nil, nil)
}

func (vt *Terminal) RenderFgBg(w io.Writer, fg, bg termenv.Color) error {
	vt.mut.Lock()
	defer vt.mut.Unlock()
	for i := range vt.Height {
		if i > 0 {
			fmt.Fprintln(w)
		}
		err := vt.renderLine(w, i, fg, bg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vt *Terminal) RenderLine(w io.Writer, row int) error {
	return vt.RenderLineFgBg(w, row, nil, nil)
}

func (vt *Terminal) RenderLineFgBg(w io.Writer, row int, fg, bg termenv.Color) error {
	vt.mut.Lock()
	defer vt.mut.Unlock()
	return vt.renderLine(w, row, fg, bg)
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
			out += f.Render(nil, nil)
		}
		out += string(r)
	}
	return out
}

var ReverseFormat = Format{Properties: ReverseBit}

func (vt *Terminal) renderLine(w io.Writer, row int, fg, bg termenv.Color) error {
	if row >= len(vt.Content) {
		return fmt.Errorf("line %d exceeds content height", row)
	}

	var pos int
	lastFormat := EmptyFormat
	format := func(f Format) {
		if lastFormat != f {
			// TODO: this is probably a sane thing to do, but it makes picky tests
			// fail; what if the last format set Italic? we need to reset it if the
			// new format doesn't also set it.
			// if lastFormat != EmptyFormat {
			// 	fmt.Fprint(w, resetSeq)
			// }
			fmt.Fprint(w, f.Render(fg, bg))
			lastFormat = f
		}
	}

	if fg != nil || bg != nil {
		format(Format{Fg: fg, Bg: bg})
	}

	for region := range vt.Format.Regions(row) {
		line := vt.Content[row]

		showCursor := vt.CursorVisible &&
			row == vt.Cursor.Y &&
			vt.Cursor.X >= pos &&
			vt.Cursor.X < pos+region.Size &&
			(vt.CursorBlinkEpoch == nil ||
				int(time.Since(*vt.CursorBlinkEpoch).Seconds())%2 == 0)

		if showCursor {
			before := string(line[pos:vt.Cursor.X])
			cursor := string(line[vt.Cursor.X])
			after := string(line[vt.Cursor.X+1 : pos+region.Size])

			if len(before) > 0 {
				format(region.F)
				fmt.Fprint(w, before)
			}

			invert := region.F
			invert.SetReverse(!region.F.IsReverse())
			format(invert)
			fmt.Fprint(w, cursor)

			if len(after) > 0 {
				format(region.F)
				fmt.Fprint(w, after)
			}
		} else {
			format(region.F)
			content := string(line[pos : pos+region.Size])
			fmt.Fprint(w, content)
		}

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

func (f Format) Render(fg, bg termenv.Color) string {
	styles := []string{}

	if f.IsBold() {
		styles = append(styles, termenv.BoldSeq)
		f.Fg = brighten(f.Fg)
	} else if f.IsFaint() {
		styles = append(styles, termenv.FaintSeq)
	}

	if f.Fg != nil {
		styles = append(styles, f.Fg.Sequence(false))
	} else if fg != nil {
		styles = append(styles, fg.Sequence(false))
	}
	if f.Bg != nil {
		styles = append(styles, f.Bg.Sequence(true))
	} else if bg != nil {
		styles = append(styles, bg.Sequence(true))
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
