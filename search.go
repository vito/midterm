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

// searchState holds cached state from the previous Search() call so that
// subsequent calls with the same query can skip unchanged rows.
type searchState struct {
	query      string   // lowercased query from last search
	queryRunes int      // rune length of query
	changes    []uint64 // snapshot of Changes[] at last search
	maxY       int      // MaxY at last search
}

// Search finds all case-insensitive occurrences of query in Content
// and populates SearchHighlights. If the query is the same as the
// previous call, only rows that have changed since then are re-scanned.
// Returns the total match count.
func (vt *Terminal) Search(query string) int {
	if query == "" {
		vt.SearchClear()
		return 0
	}

	lowerQuery := strings.ToLower(query)
	queryRuneLen := utf8.RuneCountInString(lowerQuery)

	// Check if we can do an incremental update.
	if vt.searchCache != nil &&
		vt.searchCache.query == lowerQuery &&
		vt.SearchHighlights != nil {
		return vt.searchIncremental(lowerQuery, queryRuneLen)
	}

	// Full search.
	return vt.searchFull(lowerQuery, queryRuneLen)
}

// searchFull does a complete search from scratch.
func (vt *Terminal) searchFull(lowerQuery string, queryRuneLen int) int {
	vt.SearchMatches = vt.SearchMatches[:0]
	if vt.SearchHighlights == nil {
		vt.SearchHighlights = make(map[int][]SearchHighlight)
	} else {
		clear(vt.SearchHighlights)
	}

	used := vt.MaxY + 1
	for row := 0; row < used && row < len(vt.Content); row++ {
		vt.searchRow(row, lowerQuery, queryRuneLen)
	}

	vt.snapshotSearchState(lowerQuery, queryRuneLen)
	return len(vt.SearchMatches)
}

// searchIncremental re-scans only rows that have changed since the last
// search, plus any new rows that appeared.
func (vt *Terminal) searchIncremental(lowerQuery string, queryRuneLen int) int {
	cache := vt.searchCache
	used := vt.MaxY + 1

	// Collect rows that need re-scanning.
	var dirtyRows []int
	limit := min(used, len(vt.Changes))
	cachedLimit := min(limit, len(cache.changes))
	for row := 0; row < cachedLimit; row++ {
		if vt.Changes[row] != cache.changes[row] {
			dirtyRows = append(dirtyRows, row)
		}
	}
	// New rows beyond what we last saw.
	for row := cache.maxY + 1; row < limit; row++ {
		dirtyRows = append(dirtyRows, row)
	}

	if len(dirtyRows) == 0 {
		return len(vt.SearchMatches)
	}

	// Remove old matches/highlights for dirty rows.
	dirtySet := make(map[int]bool, len(dirtyRows))
	for _, r := range dirtyRows {
		dirtySet[r] = true
		delete(vt.SearchHighlights, r)
	}

	// Filter out stale matches from SearchMatches.
	n := 0
	for _, m := range vt.SearchMatches {
		if !dirtySet[m.Row] {
			vt.SearchMatches[n] = m
			n++
		}
	}
	vt.SearchMatches = vt.SearchMatches[:n]

	// Re-scan dirty rows and collect new matches.
	var newMatches []SearchMatch
	for _, row := range dirtyRows {
		if row >= len(vt.Content) {
			continue
		}
		before := len(newMatches)
		newMatches = vt.searchRowInto(row, lowerQuery, queryRuneLen, newMatches)
		// Also populate highlights.
		for _, m := range newMatches[before:] {
			vt.SearchHighlights[row] = append(vt.SearchHighlights[row], SearchHighlight{
				Col: m.Col,
				End: m.End,
			})
		}
	}

	// Merge new matches into SearchMatches in row order.
	if len(newMatches) > 0 {
		vt.SearchMatches = mergeMatches(vt.SearchMatches, newMatches)
	}

	vt.snapshotSearchState(lowerQuery, queryRuneLen)
	return len(vt.SearchMatches)
}

// searchRow scans a single row and appends matches to SearchMatches and
// SearchHighlights.
func (vt *Terminal) searchRow(row int, lowerQuery string, queryRuneLen int) {
	line := vt.Content[row]
	lineStr := strings.ToLower(string(line))
	searchFrom := 0
	for {
		idx := strings.Index(lineStr[searchFrom:], lowerQuery)
		if idx < 0 {
			break
		}
		col := utf8.RuneCountInString(lineStr[:searchFrom+idx])
		end := col + queryRuneLen
		vt.SearchHighlights[row] = append(vt.SearchHighlights[row], SearchHighlight{
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

// searchRowInto is like searchRow but appends to an external slice instead
// of vt.SearchMatches (used during incremental merge).
func (vt *Terminal) searchRowInto(row int, lowerQuery string, queryRuneLen int, dst []SearchMatch) []SearchMatch {
	line := vt.Content[row]
	lineStr := strings.ToLower(string(line))
	searchFrom := 0
	for {
		idx := strings.Index(lineStr[searchFrom:], lowerQuery)
		if idx < 0 {
			break
		}
		col := utf8.RuneCountInString(lineStr[:searchFrom+idx])
		end := col + queryRuneLen
		dst = append(dst, SearchMatch{
			Row: row,
			Col: col,
			End: end,
		})
		searchFrom += idx + len(lowerQuery)
	}
	return dst
}

// snapshotSearchState captures the current Changes[] and MaxY so the next
// Search() call can detect which rows are dirty.
func (vt *Terminal) snapshotSearchState(lowerQuery string, queryRuneLen int) {
	used := vt.MaxY + 1
	limit := min(used, len(vt.Changes))
	changes := make([]uint64, limit)
	copy(changes, vt.Changes[:limit])
	vt.searchCache = &searchState{
		query:      lowerQuery,
		queryRunes: queryRuneLen,
		changes:    changes,
		maxY:       vt.MaxY,
	}
}

// mergeMatches merges two sorted-by-row match slices into one.
func mergeMatches(a, b []SearchMatch) []SearchMatch {
	result := make([]SearchMatch, 0, len(a)+len(b))
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i].Row < b[j].Row || (a[i].Row == b[j].Row && a[i].Col <= b[j].Col) {
			result = append(result, a[i])
			i++
		} else {
			result = append(result, b[j])
			j++
		}
	}
	result = append(result, a[i:]...)
	result = append(result, b[j:]...)
	return result
}

// SearchClear removes all search highlights, matches, and cached state.
func (vt *Terminal) SearchClear() {
	vt.SearchHighlights = nil
	vt.SearchMatches = nil
	vt.searchCache = nil
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
