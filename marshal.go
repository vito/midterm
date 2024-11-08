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

	var buffer bytes.Buffer
	var bytez []byte
	// The current screen must be serialized last so that we can hack
	// the wrap flag into the correct state.
	if vt.Alt != nil {
		if !vt.IsAlt {
			buffer.WriteString(termenv.CSI + termenv.AltScreenSeq)
		}
		bytez, err = vt.Alt.marshalBinary()
		if err != nil {
			return
		}
		buffer.Write(bytez)
		if vt.IsAlt {
			buffer.WriteString(termenv.CSI + termenv.AltScreenSeq)
		} else {
			buffer.WriteString(termenv.CSI + termenv.ExitAltScreenSeq)
		}
	}
	bytez, err = vt.Screen.marshalBinary()
	if err != nil {
		return
	}
	buffer.Write(bytez)

	if vt.Title != "" {
		_, err = fmt.Fprintf(&buffer, termenv.OSC+termenv.SetWindowTitleSeq, vt.Title)
		if err != nil {
			return
		}
	}

	if vt.wrap { // Hack to force wrap flag into correct state
		row := vt.Screen.Cursor.Y
		col := vt.Screen.Cursor.X

		_, _ = fmt.Fprintf(&buffer, termenv.CSI+termenv.CursorPositionSeq, row+1, col+1)

		var region *Region
		for region = vt.Screen.Format.Rows[row]; region.Next != nil; region = region.Next {
			//Seek to the last region since we're always targeting the end of the line
		}

		bytez, err = region.F.MarshalBinary()
		if err != nil {
			return
		}
		content := vt.Screen.Content[row][col]
		_, err = buffer.WriteRune(content)
		if err != nil {
			return
		}

		bytez, err = vt.Screen.Cursor.F.MarshalBinary()
		if err != nil {
			return
		}
		buffer.Write(bytez)
	}

	data = buffer.Bytes()
	return
}

func (s *Screen) marshalBinary() (data []byte, err error) {
	var buffer bytes.Buffer
	for i := 0; i <= s.MaxY; i++ {
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
		end := min(pos+region.Size, s.MaxX+1)
		content := string(s.Content[row][pos:end])
		_, err = buffer.WriteString(content)
		if err != nil {
			return
		}

		if end > s.MaxX {
			//Don't write the rest of the line if the screen doesn't extend that far
			break
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

	// Handling the foreground and background down here is a hack to compensate for this bug:
	// https://github.com/danielgatis/go-ansicode/issues/4
	if f.Fg != nil {
		res += termenv.CSI + f.Fg.Sequence(false) + "m"
		//styles = append(styles, f.Fg.Sequence(false))
	}
	if f.Bg != nil {
		res += termenv.CSI + f.Bg.Sequence(true) + "m"
		//styles = append(styles, f.Bg.Sequence(true))
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

	_, err = fmt.Fprintf(&buffer, termenv.CSI+"%d q", c.S+1)
	if err != nil {
		return
	}

	data = buffer.Bytes()
	return
}
