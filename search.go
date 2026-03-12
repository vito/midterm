package midterm

import (
	"strings"
	"unicode/utf8"
)

// SearchHighlight represents a highlighted range on a single row.
type SearchHighlight struct {
	Col, End int  // column range [Col, End)
	Current  bool // true = current match (distinct style)
}

// SearchMatch stores an individual match location for indexed navigation.
type SearchMatch struct {
	Row, Col, End int
}

// Search finds all case-insensitive occurrences of query in Content
// and populates SearchHighlights. Returns the total match count.
// The caller must hold vt.mut.
func (vt *Terminal) Search(query string) int {
	vt.SearchMatches = nil
	vt.SearchHighlights = nil

	if query == "" {
		return 0
	}

	lowerQuery := strings.ToLower(query)
	queryRuneLen := utf8.RuneCountInString(lowerQuery)

	used := vt.MaxY + 1
	highlights := make(map[int][]SearchHighlight)

	for row := 0; row < used; row++ {
		if row >= len(vt.Content) {
			break
		}
		line := vt.Content[row]
		lineStr := strings.ToLower(string(line))
		// Trim trailing spaces for matching, but use rune indices into the
		// original line for highlight positions.
		searchFrom := 0
		for {
			idx := strings.Index(lineStr[searchFrom:], lowerQuery)
			if idx < 0 {
				break
			}
			// idx is byte offset in lineStr[searchFrom:]; convert to rune col
			col := utf8.RuneCountInString(lineStr[:searchFrom+idx])
			end := col + queryRuneLen
			highlights[row] = append(highlights[row], SearchHighlight{
				Col: col,
				End: end,
			})
			vt.SearchMatches = append(vt.SearchMatches, SearchMatch{
				Row: row,
				Col: col,
				End: end,
			})
			searchFrom += idx + len(lowerQuery)
		}
	}

	vt.SearchHighlights = highlights
	return len(vt.SearchMatches)
}

// SearchClear removes all search highlights and matches.
func (vt *Terminal) SearchClear() {
	vt.SearchHighlights = nil
	vt.SearchMatches = nil
}

// SearchSetCurrent marks the match at the given index as "current"
// (receives CurrentStyle). Returns the (row, col) of that match.
// If idx is out of range, clears any current highlight.
func (vt *Terminal) SearchSetCurrent(idx int) (row, col int) {
	// Clear all Current flags first.
	for r, hls := range vt.SearchHighlights {
		for i := range hls {
			if hls[i].Current {
				hls[i].Current = false
				vt.SearchHighlights[r] = hls
			}
		}
	}

	if idx < 0 || idx >= len(vt.SearchMatches) {
		return -1, -1
	}

	m := vt.SearchMatches[idx]
	hls := vt.SearchHighlights[m.Row]
	for i := range hls {
		if hls[i].Col == m.Col && hls[i].End == m.End {
			hls[i].Current = true
			vt.SearchHighlights[m.Row] = hls
			break
		}
	}
	return m.Row, m.Col
}

// SearchMatchCount returns the number of matches from the last Search call.
func (vt *Terminal) SearchMatchCount() int {
	return len(vt.SearchMatches)
}

// SearchMatchRows returns the distinct row indices that have matches,
// in ascending order.
func (vt *Terminal) SearchMatchRows() []int {
	if len(vt.SearchMatches) == 0 {
		return nil
	}
	var rows []int
	lastRow := -1
	for _, m := range vt.SearchMatches {
		if m.Row != lastRow {
			rows = append(rows, m.Row)
			lastRow = m.Row
		}
	}
	return rows
}
