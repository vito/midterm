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

func (canvas *Canvas) Region(row, col int) *Region {
	// Check if the row is out of bounds
	if row >= len(canvas.Rows) || row < 0 {
		return nil
	}

	current := canvas.Rows[row]
	pos := 0

	// Traverse the regions in the row to find the region containing 'col'
	for current != nil {
		end := pos + current.Size
		if pos <= col && col < end {
			// The column is within the current region
			return current
		}

		// Move to the next region
		pos = end
		current = current.Next
	}

	// If no region is found, return nil
	return nil
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
				if region.Size <= 0 {
					panic(region.Size)
				}
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
	if prev.Next.Size <= 0 {
		panic("wha? 4")
	}

	// handle empty initial region
	if prev.Size == 0 {
		*prev = *prev.Next
	}
}

// TODO: untested
func (canvas *Canvas) Insert(cursor Cursor, n int) {
	for len(canvas.Rows) <= cursor.Y {
		// initialize empty regions up to the cursor row
		canvas.Rows = append(canvas.Rows, &Region{Size: canvas.Width})
	}

	var pos int
	var prev *Region
	for region := range canvas.Regions(cursor.Y) {
		next := region.Next
		end := pos + region.Size
		if end == cursor.X {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size += n
				region.F = cursor.F
				return
			}
			if cursor.F == region.F {
				// same format; grow existing region
				region.Size += n
				return
			} else if next != nil && cursor.F == next.F {
				// next region already has same format; grow it
				next.Size += n
				return
			} else {
				// insert before the next region
				region.Next = &Region{
					F:    cursor.F,
					Size: n,
					Next: region.Next,
				}
				return
			}
		} else if cursor.X == pos {
			if region.Size == 0 {
				// empty row; bootstrap it
				region.Size++
				region.F = cursor.F
				return
			}
			cp := *region
			*region = Region{
				F:    cursor.F,
				Size: n,
				Next: &cp,
			}
			return
		} else if end > cursor.X {
			if cursor.F == region.F {
				// grow the current region
				region.Size += n
				return
			} else {
				// split the region
				region.Size = cursor.X - pos
				if region.Size <= 0 {
					panic(region.Size)
				}
				origNext := region.Next
				region.Next = &Region{
					F:    cursor.F,
					Size: n,
					Next: &Region{
						F:    region.F,
						Size: end - cursor.X,
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
		Size: cursor.X - pos,
		Next: &Region{
			F:    cursor.F,
			Size: n,
		},
	}

	// handle empty initial region
	if prev.Size == 0 {
		panic("TESTME")
		*prev = *prev.Next
	}
}

// TODO: untested
func (canvas *Canvas) Delete(cursor Cursor, n int) {
	if cursor.Y >= len(canvas.Rows) {
		return // Row doesn't exist, nothing to delete
	}

	var pos int
	var prev *Region
	for region := range canvas.Regions(cursor.Y) {
		end := pos + region.Size

		if end > cursor.X {
			// We're in the region to start deleting
			if cursor.X > pos {
				// Split the region if the cursor is in the middle
				region.Size = cursor.X - pos
				origNext := region.Next
				remainder := end - cursor.X
				region.Next = &Region{
					F:    region.F,
					Size: remainder,
					Next: origNext,
				}
				region = region.Next
			}

			// Delete the specified number of characters
			for n > 0 && region != nil {
				if n >= region.Size {
					// Fully delete this region
					n -= region.Size
					if prev != nil {
						prev.Next = region.Next
					} else {
						// Head of the row being deleted
						canvas.Rows[cursor.Y] = region.Next
					}
					region = region.Next
				} else {
					// Partially delete from this region
					region.Size -= n
					n = 0
				}
			}
			return
		}

		pos = end
		prev = region
	}
}

func (canvas *Canvas) Resize(h, w int) {
	// Handle height adjustment
	if h < len(canvas.Rows) {
		// Truncate rows if the new height is less than the current height
		canvas.Rows = canvas.Rows[:h]
	} else if h > len(canvas.Rows) {
		// Add new empty rows if the new height is greater than the current height
		for i := len(canvas.Rows); i < h; i++ {
			canvas.Rows = append(canvas.Rows, nil)
		}
	}

	// Handle width adjustment
	for y := 0; y < len(canvas.Rows); y++ {
		row := canvas.Rows[y]
		if row == nil {
			continue
		}

		current := row
		position := 0
		var previous *Region

		// Traverse the row to find regions that exceed the new width
		for current != nil {
			if position+current.Size > w {
				// Case 1: The current region exceeds the new width, so truncate it
				if position < w {
					current.Size = w - position
					current.Next = nil // Remove the rest of the row
				} else {
					// Case 2: The entire region is beyond the new width, so remove it
					if previous != nil {
						previous.Next = nil
					} else {
						// If this was the first region, the row becomes empty
						canvas.Rows[y] = nil
					}
				}
				break
			}

			position += current.Size
			previous = current
			current = current.Next
		}
	}
}

func (region *Region) String() string {
	return fmt.Sprintf("%s:%d", region.F.Render(), region.Size)
}

func (region *Region) consumeNext() {
	next := region.Next
	if next != nil {
		next.Size--
		if next.Size < 0 {
			panic("wha? 1")
		}
		if next.Size == 0 {
			region.Next = nil
			if next.Next != nil {
				region.Next = next.Next
			}
		}
	}
}
