package midterm

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/muesli/termenv"
)

func (vt *Terminal) MarshalBinary() (data []byte, err error) {
	vt.mut.Lock()
	defer vt.mut.Unlock()

	return vt.marshalBinary()
}

func (s *Screen) marshalBinary() (data []byte, err error) {
	var buffer bytes.Buffer
	for i := 0; i < s.Height; i++ {
		if i > 0 {
			_, _ = io.WriteString(&buffer, "\r\n")
		}
		var line []byte
		line, err = s.marshalLine(i)
		if err != nil {
			return
		}
		buffer.Write(line)
	}

	var cursor []byte
	cursor, err = s.Cursor.MarshalBinary()
	if err != nil {
		return
	}
	_, err = buffer.Write(cursor)
	if err != nil {
		return
	}

	if s.CursorVisible {
		_, err = buffer.WriteString(termenv.CSI + termenv.ShowCursorSeq)
	} else {
		_, err = buffer.WriteString(termenv.CSI + termenv.HideCursorSeq)
	}
	if err != nil {
		return
	}

	data = buffer.Bytes()
	return
}

func (s *Screen) marshalLine(row int) (data []byte, err error) {
	if row >= len(s.Content) {
		err = fmt.Errorf("line %d exceeds content height", row)
		return
	}

	var buffer bytes.Buffer
	lastFormat := EmptyFormat
	format := func(f Format) {
		if lastFormat != f {
			// TODO: this is probably a sane thing to do, but it makes picky tests
			// fail; what if the last format set Italic? we need to reset it if the
			// new format doesn't also set it.
			// if lastFormat != EmptyFormat {
			// 	fmt.Fprint(w, resetSeq)
			// }
			data, _ := f.MarshalBinary()
			_, _ = buffer.Write(data)
			lastFormat = f
		}
	}

	var pos int
	for region := range s.Format.Regions(row) {
		format(region.F)
		content := string(s.Content[row][pos : pos+region.Size])
		_, err = buffer.WriteString(content)
		if err != nil {
			return
		}

		pos += region.Size
	}

	_, err = buffer.WriteString(resetSeq)
	if err != nil {
		return
	}

	data = buffer.Bytes()
	return
}

func (f Format) MarshalBinary() (data []byte, err error) {
	var styles []string

	if f.IsBold() {
		styles = append(styles, termenv.BoldSeq)
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
	if f.IsReset() || f == EmptyFormat {
		res = resetSeq
	}
	if len(styles) > 0 {
		res += fmt.Sprintf("%s%sm", termenv.CSI, strings.Join(styles, ";"))
	}
	return []byte(res), nil
}

func (c Cursor) MarshalBinary() (data []byte, err error) {
	var buffer bytes.Buffer
	//Move cursor into position
	_, err = io.WriteString(&buffer, termenv.CSI)
	if err != nil {
		return
	}
	_, err = fmt.Fprintf(&buffer, termenv.CursorPositionSeq, c.Y+1, c.X+1)
	if err != nil {
		return
	}
	//Set the format
	var format []byte
	format, err = c.F.MarshalBinary()
	if err != nil {
		return
	}
	_, err = buffer.Write(format)
	if err != nil {
		return
	}

	data = buffer.Bytes()
	return
}
