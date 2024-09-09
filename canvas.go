package midterm

import (
	"fmt"
	"iter"
	"strings"
)

type Canvas struct {
	rows []*Row // Each row is composed of Regions
}

// Region represents a segment of a row with a specific format.
type Region struct {
	// Format that applies to this region.
	F *Format

	// Size is the number of characters to which the format applies.
	Size int
}

type Chunk struct {
	X, Y   int
	Region Region
}

// func (canvas *Canvas) Resize(h, w int) {
// 	// Handle height adjustment
// 	if h < len(canvas.Lines) {
// 		// Truncate rows if height is reduced
// 		canvas.Lines = canvas.Lines[:h]
// 	} else if h > len(canvas.Lines) {
// 		// Add new empty rows if height is increased
// 		for i := len(canvas.Lines); i < h; i++ {
// 			canvas.Lines = append(canvas.Lines, []Region{})
// 		}
// 	}
//
// 	// Handle width adjustment
// 	for y := 0; y < len(canvas.Lines); y++ {
// 		row := canvas.Lines[y]
// 		rowWidth := 0
// 		for _, region := range row {
// 			rowWidth += region.Size
// 		}
//
// 		if rowWidth > w {
// 			// Truncate regions to fit the new width
// 			newRow := []Region{}
// 			remainingWidth := w
//
// 			for _, region := range row {
// 				if region.Size <= remainingWidth {
// 					newRow = append(newRow, region)
// 					remainingWidth -= region.Size
// 				} else {
// 					// Truncate the last region to fit the remaining width
// 					newRow = append(newRow, Region{
// 						F:    region.F,
// 						Size: remainingWidth,
// 					})
// 					break
// 				}
// 			}
// 			canvas.Lines[y] = newRow
// 		}
// 		// No need to do anything if rowWidth <= w (expanding the width)
// 	}
// }

func (canvas *Canvas) Resize(h, w int) {
	// Handle height adjustment
	if h < len(canvas.rows) {
		// Truncate rows if height is reduced
		canvas.rows = canvas.rows[:h]
	} else if h > len(canvas.rows) {
		// Add new empty rows if height is increased
		for i := len(canvas.rows); i < h; i++ {
			canvas.rows = append(canvas.rows, &Row{}) // Append new empty rows
		}
	}
	return

	// Handle width adjustment
	for y := 0; y < len(canvas.rows); y++ {
		row := canvas.rows[y]
		current := row.Head
		previous := (*RegionNode)(nil)
		position := 0

		// Traverse the row and check if the row width exceeds the new width
		for current != nil {
			next := current.Next
			if position+current.Region.Size > w {
				// If the current region exceeds the new width, we truncate or remove it
				if position < w {
					// Partially truncate the region to fit within the new width
					current.Region.Size = w - position
					current.Next = nil // Truncate the rest of the row
					row.Tail = current
				} else {
					// The region starts beyond the new width, so remove it
					if previous != nil {
						previous.Next = nil
						row.Tail = previous
					} else {
						// If there's no previous node, the row becomes empty
						row.Head = nil
						row.Tail = nil
					}
				}
				break
			}
			// Move to the next region in the row
			position += current.Region.Size
			previous = current
			current = next
		}
	}
}

// Linked list node for Region
type RegionNode struct {
	Region *Region
	Next   *RegionNode
}

// Row will now be represented as a linked list of RegionNodes
type Row struct {
	Head *RegionNode
	Tail *RegionNode
}

func (canvas *Canvas) Height() int {
	return len(canvas.rows)
}

func (canvas *Canvas) Regions(row int) iter.Seq[*Region] {
	return func(yield func(*Region) bool) {
		// Check if the requested row exists
		if row >= len(canvas.rows) || row < 0 {
			return
		}
		canvas.rows[row].Regions(yield)
	}
}

func (row *Row) Regions(yield func(*Region) bool) {
	// Get the head of the linked list for the row
	current := row.Head

	// Iterate over the linked list and yield each Region
	for current != nil {
		// Call the yield function with the current Region
		if !yield(current.Region) {
			// Stop iteration if yield returns false
			break
		}
		current = current.Next
	}
}

func (r *Row) String() string {
	rs := []string{}
	for region := range r.Regions {
		rs = append(rs, region.String())
	}
	return strings.Join(rs, ";")
}

func (r Region) String() string {
	return fmt.Sprintf("%s:%d", r.F.Render(), r.Size)
}

func (canvas *Canvas) Region(row, col int) *Region {
	// Ensure the row exists
	if row >= len(canvas.rows) || row < 0 {
		return nil // Row does not exist
	}

	rowRegions := canvas.rows[row]
	current := rowRegions.Head
	position := 0

	// Traverse through the regions in the specified row
	for current != nil {
		// Check if the column falls within the current region
		if position <= col && col < position+current.Region.Size {
			return current.Region // Found the region containing the column
		}

		// Move to the next region
		position += current.Region.Size
		current = current.Next
	}

	// If we reach here, the column is out of bounds
	return nil
}

func (canvas *Canvas) Insert(cursor Cursor, n int) {
	// Ensure the row exists
	for len(canvas.rows) <= cursor.Y {
		canvas.rows = append(canvas.rows, &Row{})
	}

	row := canvas.rows[cursor.Y]
	cursorX := cursor.X
	position := 0
	current := row.Head
	var previous *RegionNode

	// Traverse until we find where the cursor is or the list ends
	for current != nil && position+current.Region.Size <= cursorX {
		position += current.Region.Size
		previous = current
		current = current.Next
	}

	// Case 1: Cursor inside or at the boundary of an existing region
	if current != nil && position <= cursorX && cursorX < position+current.Region.Size {
		// Split the current region if necessary
		if position < cursorX {
			// Create a new region for the part before the cursor
			newRegionBefore := &RegionNode{
				Region: &Region{
					F:    current.Region.F,
					Size: cursorX - position,
				},
			}

			if previous != nil {
				previous.Next = newRegionBefore
			} else {
				row.Head = newRegionBefore
			}

			previous = newRegionBefore
			newRegionBefore.Next = current
		}

		// Adjust the current region for the part after the cursor
		remainingRegionSize := (position + current.Region.Size) - cursorX
		if remainingRegionSize > 0 {
			// Move the current region after the inserted region
			newRegionAfter := &RegionNode{
				Region: &Region{
					F:    current.Region.F,
					Size: remainingRegionSize,
				},
			}

			current.Region.Size = 0 // The current part before the split is now handled by the new region
			newRegionAfter.Next = current.Next
			current = newRegionAfter
		} else {
			// If nothing is left in the current region, remove it
			current = current.Next
		}
	}

	// Case 2: Insert the new region at the cursor position
	newFormatRegion := &RegionNode{
		Region: &Region{
			F:    cursor.F,
			Size: n,
		},
	}

	if previous != nil {
		previous.Next = newFormatRegion
	} else {
		// This is the new head
		row.Head = newFormatRegion
	}

	newFormatRegion.Next = current

	// Update the tail if the current is nil
	if current == nil {
		row.Tail = newFormatRegion
	}
}

func (canvas *Canvas) Delete(cursor Cursor, n int) {
	// Ensure the row exists
	if cursor.Y >= len(canvas.rows) {
		return // Row does not exist, nothing to delete
	}

	row := canvas.rows[cursor.Y]
	cursorX := cursor.X
	position := 0
	current := row.Head
	var previous *RegionNode

	// Traverse until we find where the cursor is or the list ends
	for current != nil && position+current.Region.Size <= cursorX {
		position += current.Region.Size
		previous = current
		current = current.Next
	}

	// If cursor is beyond the current content, nothing to delete
	if current == nil {
		return
	}

	// Case 1: Cursor inside or at the boundary of an existing region
	if position <= cursorX && cursorX < position+current.Region.Size {
		// Calculate how many cells to delete from the current region
		deleteCount := min(n, (position+current.Region.Size)-cursorX)

		// If the current region has more cells than we're deleting, just reduce the size
		if deleteCount < current.Region.Size {
			current.Region.Size -= deleteCount
		} else {
			// Remove the current region if it's fully consumed
			if previous != nil {
				previous.Next = current.Next
			} else {
				// If there's no previous, we are removing the head
				row.Head = current.Next
			}

			// Update the tail if necessary
			if current.Next == nil {
				row.Tail = previous
			}
		}

		// Reduce the number of cells left to delete
		n -= deleteCount
		current = current.Next
	}

	// Case 2: Continue deleting from subsequent regions if `n > 0`
	for n > 0 && current != nil {
		if current.Region.Size <= n {
			// If the entire region is deleted, remove it
			n -= current.Region.Size
			if previous != nil {
				previous.Next = current.Next
			} else {
				row.Head = current.Next
			}

			// Update the tail if necessary
			if current.Next == nil {
				row.Tail = previous
			}

			current = current.Next
		} else {
			// Partially delete this region
			current.Region.Size -= n
			n = 0
		}
	}
}

func (canvas *Canvas) Paint(cursor Cursor) {
	// Ensure the row exists
	for len(canvas.rows) <= cursor.Y {
		canvas.rows = append(canvas.rows, &Row{})
	}

	row := canvas.rows[cursor.Y]
	cursorX := cursor.X
	remainingWidth := 1 // The cursor paints one character at a time

	// Traverse the linked list to find where the cursor is
	current := row.Head
	var previous *RegionNode
	position := 0

	// Traverse until we find where the cursor is or the list ends
	for current != nil && position+current.Region.Size <= cursorX {
		position += current.Region.Size
		previous = current
		current = current.Next
	}

	// Case 1: Painting beyond the end (or at the boundary of the last region)
	if current == nil && position <= cursorX {
		// If the cursor is beyond the last region, we are appending
		if previous != nil && previous.Region.F.Fg == cursor.F.Fg && previous.Region.F.Bg == cursor.F.Bg && previous.Region.F.Properties == cursor.F.Properties {
			// Extend the last region if the format is the same
			previous.Region.Size += remainingWidth
		} else {
			// Otherwise, create a new region with the cursor's format
			newFormatRegion := &RegionNode{
				Region: &Region{
					F:    cursor.F,
					Size: remainingWidth,
				},
			}

			if previous != nil {
				previous.Next = newFormatRegion
			} else {
				// If no previous node, this is the first region in the row
				row.Head = newFormatRegion
			}

			row.Tail = newFormatRegion // Update the tail
		}
		return
	}

	// Case 2: Cursor inside an existing region
	if current != nil && position <= cursorX && cursorX < position+current.Region.Size {
		// Cursor is inside this region, but the format might be different
		if current.Region.F.Fg == cursor.F.Fg && current.Region.F.Bg == cursor.F.Bg && current.Region.F.Properties == cursor.F.Properties {
			// Same format: No-op (nothing to change)
			return
		}

		// Different format: we need to split this region
		// 1. Create a new region before the cursor (if any content exists before the cursor)
		if position < cursorX {
			newRegionBefore := &RegionNode{
				Region: &Region{
					F:    current.Region.F,
					Size: cursorX - position,
				},
			}
			if previous != nil {
				previous.Next = newRegionBefore
			} else {
				// If there's no previous, this is the new head
				row.Head = newRegionBefore
			}
			newRegionBefore.Next = current
			previous = newRegionBefore
		}

		// 2. Insert a new region with the cursor's format
		newFormatRegion := &RegionNode{
			Region: &Region{
				F:    cursor.F,
				Size: remainingWidth,
			},
		}
		newFormatRegion.Next = current
		if previous != nil {
			previous.Next = newFormatRegion
		} else {
			// If no previous node, this is the new head
			row.Head = newFormatRegion
		}

		// 3. Adjust the size of the current region (the part after the cursor)
		remainingRegionSize := (position + current.Region.Size) - (cursorX + remainingWidth)
		if remainingRegionSize > 0 {
			// Truncate the current region and move it after the new format
			current.Region.Size = remainingRegionSize
			newFormatRegion.Next = current
		} else {
			// If there's nothing left, remove the current region
			newFormatRegion.Next = current.Next
		}

		// Update the tail if necessary
		if current.Next == nil && newFormatRegion.Next == nil {
			row.Tail = newFormatRegion
		}
		return
	}
}

// func (canvas *Canvas) Paint(cursor Cursor) {
// 	// Ensure the row exists
// 	for len(canvas.rows) <= cursor.Y {
// 		canvas.rows = append(canvas.rows, &Row{})
// 	}
//
// 	row := canvas.rows[cursor.Y]
// 	cursorX := cursor.X
// 	remainingWidth := 1 // The cursor paints one character at a time
//
// 	// Traverse the linked list to find where the cursor is
// 	current := row.Head
// 	var previous *RegionNode
// 	position := 0
//
// 	// Traverse until we find where the cursor is or the list ends
// 	for current != nil && position+current.Region.Size <= cursorX {
// 		position += current.Region.Size
// 		previous = current
// 		current = current.Next
// 	}
//
// 	// Now we are at the correct position; let's handle painting
//
// 	// Case 1: Cursor inside an existing region
// 	if current != nil && position <= cursorX && cursorX < position+current.Region.Size {
// 		// Cursor is inside this region, but the format might be different
// 		if current.Region.F.Fg == cursor.F.Fg && current.Region.F.Bg == cursor.F.Bg && current.Region.F.Properties == cursor.F.Properties {
// 			// Same format: No-op (nothing to change)
// 			return
// 		}
//
// 		// Different format: we need to split this region
// 		// 1. Create a new region before the cursor (if any content exists before the cursor)
// 		if position < cursorX {
// 			newRegionBefore := &RegionNode{
// 				Region: &Region{
// 					F:    current.Region.F,
// 					Size: cursorX - position,
// 				},
// 			}
// 			if previous != nil {
// 				previous.Next = newRegionBefore
// 			} else {
// 				// If there's no previous, this is the new head
// 				row.Head = newRegionBefore
// 			}
// 			newRegionBefore.Next = current
// 			previous = newRegionBefore
// 		}
//
// 		// 2. Insert a new region with the cursor's format
// 		newFormatRegion := &RegionNode{
// 			Region: &Region{
// 				F:    cursor.F,
// 				Size: remainingWidth,
// 			},
// 		}
// 		newFormatRegion.Next = current
// 		if previous != nil {
// 			previous.Next = newFormatRegion
// 		} else {
// 			// If no previous, this is the new head
// 			row.Head = newFormatRegion
// 		}
//
// 		// 3. Adjust the size of the current region (the part after the cursor)
// 		remainingRegionSize := (position + current.Region.Size) - (cursorX + remainingWidth)
// 		if remainingRegionSize > 0 {
// 			// Truncate the current region and move it after the new format
// 			current.Region.Size = remainingRegionSize
// 			newFormatRegion.Next = current
// 		} else {
// 			// If there's nothing left, remove the current region
// 			newFormatRegion.Next = current.Next
// 		}
//
// 		// Update the tail if necessary
// 		if current.Next == nil && newFormatRegion.Next == nil {
// 			row.Tail = newFormatRegion
// 		}
// 		return
// 	}
//
// 	// Case 2: Insert at the boundary of an existing region or beyond current content
//
// 	// If cursorX is beyond the end of the row, append an empty region to fill the gap
// 	if current == nil && position < cursorX {
// 		// Add an empty region up to cursorX
// 		emptyRegion := &RegionNode{
// 			Region: &Region{
// 				F:    EmptyFormat,
// 				Size: cursorX - position,
// 			},
// 		}
//
// 		if previous != nil {
// 			previous.Next = emptyRegion
// 		} else {
// 			// If there was no previous, this is the first region in the row
// 			row.Head = emptyRegion
// 		}
// 		row.Tail = emptyRegion
// 		previous = emptyRegion
// 		position = cursorX
// 	}
//
// 	// Case 3: Insert a new region if formats are different
// 	newFormatRegion := &RegionNode{
// 		Region: &Region{
// 			F:    cursor.F,
// 			Size: remainingWidth,
// 		},
// 	}
//
// 	if previous != nil {
// 		previous.Next = newFormatRegion
// 	} else {
// 		// If no previous node, the new region is the head
// 		row.Head = newFormatRegion
// 	}
// 	newFormatRegion.Next = current
//
// 	// If there's no `current` (we're at the end of the row), set the new region as the tail
// 	if current == nil {
// 		row.Tail = newFormatRegion
// 	}
// }

// func (canvas *Canvas) Paint2(cursor Cursor) {
// 	defer log.Println("PAINTED", cursor, canvas.rows)
// 	// Ensure the row exists
// 	for len(canvas.rows) <= cursor.Y {
// 		canvas.rows = append(canvas.rows, &Row{})
// 	}
//
// 	row := canvas.rows[cursor.Y]
// 	cursorX := cursor.X
// 	remainingWidth := 1 // The cursor paints one character at a time
//
// 	// Traverse the linked list to find where the cursor is
// 	current := row.Head
// 	var previous *RegionNode
// 	position := 0
//
// 	// Traverse until we find where the cursor is or the list ends
// 	for current != nil && position+current.Region.Size <= cursorX {
// 		position += current.Region.Size
// 		previous = current
// 		current = current.Next
// 	}
//
// 	// If cursorX is beyond the end of the row, append an empty region to fill the gap
// 	if current == nil && position < cursorX {
// 		// Add an empty region up to cursorX
// 		emptyRegion := &RegionNode{
// 			Region: &Region{
// 				F:    EmptyFormat,
// 				Size: cursorX - position,
// 			},
// 		}
//
// 		if previous != nil {
// 			previous.Next = emptyRegion
// 		} else {
// 			// If there was no previous, this is the first region in the row
// 			row.Head = emptyRegion
// 		}
// 		row.Tail = emptyRegion
// 		previous = emptyRegion
// 		position = cursorX
// 	}
//
// 	// Now we are at the correct position; let's handle painting
//
// 	// Case 1: Grow the current region if the format is the same
// 	if current != nil && current.Region.F.Fg == cursor.F.Fg && current.Region.F.Bg == cursor.F.Bg && current.Region.F.Properties == cursor.F.Properties {
// 		// The current region's format matches the cursor's format, so we extend it
// 		current.Region.Size += remainingWidth
// 		return
// 	}
//
// 	// Case 2: Extend the previous region if the format matches and we're directly adjacent
// 	if previous != nil && previous.Region.F.Fg == cursor.F.Fg && previous.Region.F.Bg == cursor.F.Bg && previous.Region.F.Properties == cursor.F.Properties {
// 		// The previous region has the same format, so we extend it
// 		previous.Region.Size += remainingWidth
// 		return
// 	}
//
// 	// Case 3: Insert a new region if formats are different
// 	newFormatRegion := &RegionNode{
// 		Region: &Region{
// 			F:    cursor.F,
// 			Size: remainingWidth,
// 		},
// 	}
//
// 	if previous != nil {
// 		previous.Next = newFormatRegion
// 	} else {
// 		// If no previous node, the new region is the head
// 		row.Head = newFormatRegion
// 	}
// 	newFormatRegion.Next = current
//
// 	// If there's no `current` (we're at the end of the row), set the new region as the tail
// 	if current == nil {
// 		row.Tail = newFormatRegion
// 	}
// }

// func (canvas *Canvas) Paint(cursor Cursor) {
// 	defer log.Println("PAINTED", cursor, canvas.rows)
// 	// Ensure the row exists
// 	for len(canvas.rows) <= cursor.Y {
// 		canvas.rows = append(canvas.rows, &Row{})
// 	}
//
// 	row := canvas.rows[cursor.Y]
// 	cursorX := cursor.X
// 	remainingWidth := 1 // The cursor paints one character
//
// 	// Traverse the linked list to find where the cursor is
// 	current := row.Head
// 	position := 0
//
// 	for {
// 		isInRegion := position > cursorX && cursorX <= position+current.Region.Size // TODO: off by one potential
// 		if isInRegion {
// 			if current.Region.F == cursor.F {
// 				// if we're within the region and it has the same format,
// 				// there's nothing to do
// 				return
// 			}
// 			offset := cursorX - position
// 			if offset == 0 {
// 				newRegion := &RegionNode{
// 					Region: &Region{
// 						F:    cursor.F,
// 						Size: 1,
// 					},
// 					Next:
// 				}
// 				current.Region.Size = offset
// 				current.Next = newRegion
// 			}
// 		}
// 		position += current.Region.Size
// 		current = current.Next
// 	}
//
// 	for current != nil && position+current.Region.Size <= cursorX {
// 		position += current.Region.Size
// 		current = current.Next
// 	}
//
// 	// Handle case where row is empty or cursor is beyond the end of the row
// 	if current == nil {
// 		log.Println("NIL CURRENT")
// 		// Add a new empty region up to cursorX if needed
// 		if row.Tail == nil {
// 			row.Head = &RegionNode{
// 				Region: &Region{
// 					F:    EmptyFormat,
// 					Size: cursorX,
// 				},
// 			}
// 			row.Tail = row.Head
// 		} else {
// 			// Tail exists, append the empty region
// 			newRegion := &RegionNode{
// 				Region: &Region{
// 					F:    EmptyFormat,
// 					Size: cursorX - position,
// 				},
// 			}
// 			row.Tail.Next = newRegion
// 			newRegion.Prev = row.Tail
// 			row.Tail = newRegion
// 		}
//
// 		// Append the new cursor's format region after the empty region
// 		newFormatRegion := &RegionNode{
// 			Region: &Region{
// 				F:    cursor.F,
// 				Size: remainingWidth,
// 			},
// 		}
// 		row.Tail.Next = newFormatRegion
// 		newFormatRegion.Prev = row.Tail
// 		row.Tail = newFormatRegion
//
// 		return
// 	}
//
// 	// We're now in the region where the cursor lands
//
// 	// Check if the format is the same; no-op if so
// 	if current.Region.F.Fg == cursor.F.Fg && current.Region.F.Bg == cursor.F.Bg && current.Region.F.Properties == cursor.F.Properties {
// 		log.Println("SAME FORMAT", position, cursorX)
// 		if position == cursorX {
// 			current.Region.Size++
// 			if current.Next != nil {
// 				current.Next.Region.Size--
// 				if current.Next.Region.Size == 0 {
// 					// Remove the next region if it's empty
// 					if current.Next.Next != nil {
// 						current.Next.Next.Prev = current
// 					}
// 					current.Next = current.Next.Next
// 				}
// 			}
// 		}
// 		return
// 	}
//
// 	// Split the current region if necessary
// 	if position < cursorX {
// 		// Create a new region for the portion before the cursor
// 		newRegion := &RegionNode{
// 			Region: &Region{
// 				F:    current.Region.F,
// 				Size: cursorX - position,
// 			},
// 		}
// 		// Insert the new region before the current one
// 		if current.Prev != nil {
// 			current.Prev.Next = newRegion
// 			newRegion.Prev = current.Prev
// 		} else {
// 			// We're at the head of the list
// 			row.Head = newRegion
// 		}
// 		newRegion.Next = current
// 		current.Prev = newRegion
// 	}
//
// 	// Insert the new format for the cursor's format
// 	newFormatRegion := &RegionNode{
// 		Region: &Region{
// 			F:    cursor.F,
// 			Size: remainingWidth,
// 		},
// 	}
// 	// Insert between the previous part and current
// 	newFormatRegion.Next = current
// 	newFormatRegion.Prev = current.Prev
// 	if current.Prev != nil {
// 		current.Prev.Next = newFormatRegion
// 	} else {
// 		// New region is now the head
// 		row.Head = newFormatRegion
// 	}
// 	current.Prev = newFormatRegion
//
// 	// Adjust the remaining size of the current region
// 	current.Region.Size -= remainingWidth
// 	if current.Region.Size <= 0 {
// 		// If the current region is now empty, remove it from the list
// 		if current.Prev != nil {
// 			current.Prev.Next = current.Next
// 		}
// 		if current.Next != nil {
// 			current.Next.Prev = current.Prev
// 		}
// 		if row.Head == current {
// 			row.Head = current.Next
// 		}
// 		if row.Tail == current {
// 			row.Tail = current.Prev
// 		}
// 	}
// }

// func (canvas *Canvas) Paint(cursor Cursor) {
// 	// Ensure the row exists
// 	for len(canvas.Lines) <= cursor.Y {
// 		canvas.Lines = append(canvas.Lines, []Region{})
// 	}
//
// 	row := canvas.Lines[cursor.Y]
// 	cursorX := cursor.X
// 	remainingWidth := 1 // The cursor is only painting one character at a time
//
// 	var lastRegion *Region
// 	totalWidth := 0
//
// 	// Handle cases where the cursor's X position is beyond the current row's length
// 	for i, r := range row {
// 		r := r
// 		totalWidth += r.Size
// 		if totalWidth > cursorX {
// 			break
// 		}
// 		newRow = append(newRow, r)
// 		lastRegion = &r
// 	}
//
// 	// If the row is too short, allocate an empty region up to the cursor's X position
// 	if totalWidth < cursorX {
// 		lastRegion = &Region{
// 			F:    EmptyFormat,
// 			Size: cursorX - totalWidth,
// 		}
// 		newRow = append(newRow, *lastRegion)
// 		totalWidth = cursorX
// 	}
//
// 	// Now insert or update the region at the cursor's position
// 	if lastRegion != nil && lastRegion.F == cursor.F {
// 		// If the last region has the same format as the cursor's format, extend it
// 		newRow[len(newRow)-1].Size += remainingWidth
// 	} else {
// 		// Otherwise, insert a new region for the cursor's format
// 		newRow = append(newRow, Region{
// 			F:    cursor.F,
// 			Size: remainingWidth,
// 		})
// 	}
//
// 	// Append any remaining content of the row after the cursor
// 	rowWidth := 0
// 	for _, r := range row {
// 		rowWidth += r.Size
// 	}
//
// 	if rowWidth > cursorX+remainingWidth {
// 		remainingRow := row[rowWidth-(cursorX+remainingWidth):]
// 		newRow = append(newRow, remainingRow...)
// 	}
//
// 	// Replace the original row with the newly constructed one
// 	canvas.Lines[cursor.Y] = newRow
// }
