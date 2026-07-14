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
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
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
			out += f.Render()
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

	write := func(a ...any) error {
		_, err := fmt.Fprint(w, a...)
		return err
	}

	var pos int
	lastFormat := EmptyFormat
	format := func(f Format) error {
		if lastFormat != f {
			// RenderFgBg emits only "on" sequences; if f drops an attribute or
			// color the previous format set, reset first so it doesn't bleed in.
			if leaksInto(lastFormat, f, fg, bg) {
				if err := write(resetSeq); err != nil {
					return err
				}
			}
			if err := write(f.RenderFgBg(fg, bg)); err != nil {
				return err
			}
			lastFormat = f
		}
		return nil
	}

	if fg != nil || bg != nil {
		if err := format(Format{Fg: fg, Bg: bg}); err != nil {
			return err
		}
	}

	// Pre-fetch search highlights for this row (if any).
	var searchHL []SearchHighlight
	if vt.SearchHighlights != nil {
		searchHL = vt.SearchHighlights[row]
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
				if err := format(region.F); err != nil {
					return err
				}
				if err := write(before); err != nil {
					return err
				}
			}

			invert := region.F
			invert.SetReverse(!region.F.IsReverse())
			if err := format(invert); err != nil {
				return err
			}
			if err := write(cursor); err != nil {
				return err
			}

			if len(after) > 0 {
				if err := format(region.F); err != nil {
					return err
				}
				if err := write(after); err != nil {
					return err
				}
			}
		} else if len(searchHL) > 0 {
			// Render character-by-character, overriding format for highlighted cols.
			for col := pos; col < pos+region.Size; col++ {
				f := region.F
				if hlF, ok := vt.searchHighlightAt(searchHL, col); ok {
					f = hlF
				}
				if err := format(f); err != nil {
					return err
				}
				if err := write(string(line[col])); err != nil {
					return err
				}
			}
		} else {
			if err := format(region.F); err != nil {
				return err
			}
			content := string(line[pos : pos+region.Size])
			if err := write(content); err != nil {
				return err
			}
		}

		pos += region.Size
	}

	return write(resetSeq)
}

// leaksInto reports whether emitting f right after prev needs an explicit
// reset first: RenderFgBg emits only "on" sequences, so any attribute or color
// prev set that f drops would otherwise bleed through. fg and bg are
// RenderFgBg's fallback colors for unset sides.
func leaksInto(prev, f Format, fg, bg termenv.Color) bool {
	// the empty and reset formats already render from a clean slate.
	if f.IsReset() || f == (Format{}) {
		return false
	}
	const attrs = BoldBit | FaintBit | ItalicBit | UnderlineBit | BlinkBit | ReverseBit | ConcealBit
	if (prev.Properties&attrs)&^f.Properties != 0 {
		return true
	}
	if orColor(prev.Fg, fg) != nil && orColor(f.Fg, fg) == nil {
		return true
	}
	return orColor(prev.Bg, bg) != nil && orColor(f.Bg, bg) == nil
}

func orColor(c, fallback termenv.Color) termenv.Color {
	if c != nil {
		return c
	}
	return fallback
}

// searchHighlightAt checks if col falls within any search highlight range
// and returns the appropriate format override.
func (vt *Terminal) searchHighlightAt(highlights []SearchHighlight, col int) (Format, bool) {
	for _, hl := range highlights {
		if col >= hl.Col && col < hl.End {
			if hl.Current {
				return vt.SearchCurrentStyle, true
			}
			return vt.SearchMatchStyle, true
		}
	}
	return Format{}, false
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
	return f.RenderFgBg(nil, nil)
}

func (f Format) RenderFgBg(fg, bg termenv.Color) string {
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
