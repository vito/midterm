package midterm

import (
	"time"
)

type Screen struct {
	// Height and Width are the dimensions of the terminal.
	Height, Width int

	// Content is the text in the terminal.
	Content [][]rune

	// Format contains the display properties of each cell.
	Format *Canvas

	// Changes counts the number of times each row has been modified so that the
	// UI can know which content needs to be redrawn.
	//
	// Note that this includes changes to the cursor position, in case the cursor
	// is visible: if the cursor moves from row 0 to row 3, both rows will be
	// incremented.
	Changes []uint64

	// Cursor is the current state of the cursor.
	Cursor Cursor

	// ScrollRegion is the region of the terminal that is scrollable. If it is
	// nil, the entire terminal is scrollable.
	//
	// This value is set by the CSI ; Ps ; Ps r command.
	ScrollRegion *ScrollRegion

	// CursorVisible indicates whether the cursor is visible.
	//
	// This value is set by CSI ? 25 h and unset by CSI ? 25 l.
	CursorVisible bool

	// CursorBlinking indicates whether the cursor is blinking, and the start of
	// the blinking interval.
	CursorBlinkEpoch *time.Time

	// SavedCursor is the state of the cursor last time save() was called.
	SavedCursor Cursor

	// MaxY is the maximum vertical offset that a character has been printed.
	MaxY int
	// MaxX is the maximum horizontal offset that a character has been printed.
	MaxX int
}

func newScreen(h, w int) *Screen {
	s := &Screen{
		Height: h,
		Width:  w,

		// start at -1 so there's no "used" height until first write
		MaxY: -1,
	}
	s.reset()
	return s
}

func (s *Screen) reset() {
	s.Content = make([][]rune, s.Height)
	s.Format = new(Canvas)
	s.Changes = make([]uint64, s.Height)
	for row := 0; row < s.Height; row++ {
		s.Content[row] = make([]rune, s.Width)
		for col := 0; col < s.Width; col++ {
			s.Content[row][col] = ' '
			s.Format.Paint(Cursor{
				X: col,
				Y: row,
				F: EmptyFormat,
			})
		}
	}
	s.Cursor.X = 0
	s.Cursor.Y = 0
}

func (v *Screen) resize(h, w int) {
	if h < v.MaxY {
		v.MaxY = h - 1
	}

	v.Format.Resize(h, w)

	if h > v.Height {
		n := h - v.Height
		for row := 0; row < n; row++ {
			v.Content = append(v.Content, make([]rune, v.Width))
			v.Changes = append(v.Changes, 0)
			for col := 0; col < v.Width; col++ {
				v.clear(v.Height+row, col, EmptyFormat)
			}
		}
	} else if h < v.Height {
		v.Content = v.Content[:h]
		v.Changes = v.Changes[:h]
	}

	if w > v.Width {
		for i := range v.Content {
			row := make([]rune, w)
			copy(row, v.Content[i])
			v.Content[i] = row
			for j := v.Width; j < w; j++ {
				v.clear(i, j, Format{})
			}
		}
	} else if w < v.Width {
		for i := range v.Content {
			v.Content[i] = v.Content[i][:w]
			v.Changes[i]++
		}
	}

	v.Height = h
	v.Width = w

	if v.Width != 0 && v.Cursor.X >= v.Width {
		v.Cursor.X = v.Width - 1
	}
}

func (v *Screen) clear(y, x int, format Format) {
	v.Content[y][x] = ' '
	v.Format.Paint(Cursor{
		X: x,
		Y: y,
		F: format,
	})
	v.Changes[y]++
}
