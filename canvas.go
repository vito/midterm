package midterm

import (
	"fmt"
	"iter"
)

type Canvas struct {
	Width int
	Rows  []*Region
}

// Region represents a segment of a row with a specific format.
type Region struct {
	// Format that applies to this region.
	F Format

	// Size is the number of characters to which the format applies.
	Size int

	// Next is the next region in the row.
	Next *Region
}

func (canvas *Canvas) Height() int {
	return len(canvas.Rows)
}

func (canvas *Canvas) Regions(row int) iter.Seq[*Region] {
	return func(yield func(*Region) bool) {
		// Check if the requested row exists
		if row >= len(canvas.Rows) || row < 0 {
			return
		}
		for r := canvas.Rows[row]; r != nil; r = r.Next {
			if !yield(r) {
				break
			}
		}
	}
}

func (canvas *Canvas) Paint(row, col int, format Format) {
	// dbg.Printf("PAINTING %d:%d: %q", row, col, format.Render())
	for len(canvas.Rows) <= row {
		// initialize empty regions up to the cursor row
		canvas.Rows = append(canvas.Rows, &Region{Size: canvas.Width})
	}

	var pos int
	var prev *Region
	for region := range canvas.Regions(row) {
		next := region.Next
		end := pos + region.Size
		if end == col {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size++
				region.F = format
				region.consumeNext()
				return
			}
			if format == region.F {
				// same format; grow existing region
				region.Size++
				region.consumeNext()
				return
			} else if next != nil && format == next.F {
				// next region already has same format; nothing to do
				return
			} else {
				// eat into the next region
				region.Next = &Region{
					F:    format,
					Size: 1,
					Next: region.Next,
				}
				region.Next.consumeNext()
				return
			}
		} else if col == pos {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size++
				region.F = format
				region.consumeNext()
				return
			}
			cp := *region
			*region = Region{
				F:    format,
				Size: 1,
				Next: &cp,
			}
			region.consumeNext()
			return
		} else if end > col {
			if format == region.F {
				// nothing to do
				return
			} else {
				// split the region
				region.Size = col - pos
				origNext := region.Next
				region.Next = &Region{
					F:    format,
					Size: 1,
				}
				remainder := end - col - 1
				if remainder > 0 {
					// add remainder, followed by original next
					region.Next.Next = &Region{
						F:    region.F,
						Size: remainder,
						Next: origNext,
					}
				} else {
					// clipped the end; restore original next
					region.Next.Next = origNext
				}
				return
			}
		}
		pos = end
		prev = region
	}

	// painting beyond the end of the row; insert a blank gap followed by the
	// cursor.
	prev.Next = &Region{
		F:    EmptyFormat,
		Size: col - pos,
		Next: &Region{
			F:    format,
			Size: 1,
		},
	}

	// handle empty initial region
	if prev.Size == 0 {
		*prev = *prev.Next
	}
}

func (canvas *Canvas) Insert(row, col int, f Format, n int) {
	for len(canvas.Rows) <= row {
		// initialize empty regions up to the cursor row
		canvas.Rows = append(canvas.Rows, &Region{Size: canvas.Width})
	}

	var pos int
	var prev *Region
	for region := range canvas.Regions(row) {
		next := region.Next
		end := pos + region.Size
		if end == col {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size += n
				region.F = f
				return
			}
			if f == region.F {
				// same format; grow existing region
				region.Size += n
				return
			} else if next != nil && f == next.F {
				// next region already has same format; grow it
				next.Size += n
				return
			} else {
				// insert before the next region
				region.Next = &Region{
					F:    f,
					Size: n,
					Next: region.Next,
				}
				return
			}
		} else if col == pos {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size++
				region.F = f
				return
			}
			if f == region.F {
				// same format; grow existing region
				region.Size += n
				return
			}
			cp := *region
			*region = Region{
				F:    f,
				Size: n,
				Next: &cp,
			}
			return
		} else if end > col {
			if f == region.F {
				// grow the current region
				region.Size += n
				return
			} else {
				// split the region
				region.Size = col - pos
				origNext := region.Next
				region.Next = &Region{
					F:    f,
					Size: n,
					Next: &Region{
						F:    region.F,
						Size: end - col,
						Next: origNext,
					},
				}
				return
			}
		}
		pos = end
		prev = region
	}

	// painting beyond the end of the row; insert a blank gap followed by the
	// cursor.
	prev.Next = &Region{
		F:    EmptyFormat,
		Size: col - pos,
		Next: &Region{
			F:    f,
			Size: n,
		},
	}

	// handle empty initial region
	if prev.Size == 0 {
		*prev = *prev.Next
	}
}

func (canvas *Canvas) Delete(row, col, n int) {
	if row < 0 || row >= len(canvas.Rows) || col < 0 || n <= 0 {
		return
	}

	position := 0
	var previous *Region
	link := &canvas.Rows[row]
	for *link != nil {
		region := *link
		end := position + region.Size
		if end <= col {
			position = end
			previous = region
			link = &region.Next
			continue
		}

		if offset := col - position; offset > 0 {
			tail := &Region{F: region.F, Size: region.Size - offset, Next: region.Next}
			region.Size = offset
			region.Next = tail
			previous = region
			link = &region.Next
		}

		for n > 0 && *link != nil {
			region = *link
			if n < region.Size {
				region.Size -= n
				n = 0
			} else {
				n -= region.Size
				*link = region.Next
			}
		}

		if previous != nil && previous.Next != nil && previous.F == previous.Next.F {
			previous.Size += previous.Next.Size
			previous.Next = previous.Next.Next
		}
		if canvas.Rows[row] == nil {
			canvas.Rows[row] = &Region{}
		}
		return
	}
}

func (canvas *Canvas) Resize(h, w int) {
	canvas.ResizeY(h)
	canvas.ResizeX(w)
}

func (canvas *Canvas) ResizeY(h int) {
	// Handle height adjustment
	if h < len(canvas.Rows) {
		// Truncate rows if the new height is less than the current height
		canvas.Rows = canvas.Rows[:h]
	} else if h > len(canvas.Rows) {
		// Add new empty rows if the new height is greater than the current height
		for i := len(canvas.Rows); i < h; i++ {
			canvas.Rows = append(canvas.Rows, &Region{Size: canvas.Width})
		}
	}
}

func (canvas *Canvas) ResizeX(w int) {
	// Track the new width; row (re)inits seed their region size from it.
	canvas.Width = w

	// Handle width adjustment
	for y := 0; y < len(canvas.Rows); y++ {
		canvas.truncateRow(y, w)
	}
}

func (canvas *Canvas) truncateRow(row, width int) {
	if row < 0 || row >= len(canvas.Rows) || canvas.Rows[row] == nil {
		return
	}

	current := canvas.Rows[row]
	position := 0
	var previous *Region

	for current != nil {
		if position+current.Size > width {
			if position < width {
				current.Size = width - position
				current.Next = nil
			} else if previous != nil {
				previous.Next = nil
			} else {
				canvas.Rows[row] = nil
			}
			return
		}

		position += current.Size
		previous = current
		current = current.Next
	}
}

func (region *Region) String() string {
	return fmt.Sprintf("%s:%d", region.F.Render(), region.Size)
}

// ClearRow resets a row to a single region with EmptyFormat.
func (canvas *Canvas) ClearRow(row int, format Format) {
	if row >= len(canvas.Rows) {
		return
	}
	canvas.Rows[row] = &Region{F: format, Size: canvas.Width}
}

func (region *Region) consumeNext() {
	next := region.Next
	if next != nil {
		next.Size--
		if next.Size == 0 {
			region.Next = nil
			if next.Next != nil {
				region.Next = next.Next
			}
		}
	}
}
