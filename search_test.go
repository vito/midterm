package midterm

import (
	"strings"
	"testing"
)

func TestSearchBasic(t *testing.T) {
	vt := NewTerminal(10, 40)
	vt.Write([]byte("hello world\r\nfoo bar hello\r\nbaz\r\n"))

	count := vt.Search("hello")
	if count != 2 {
		t.Fatalf("expected 2 matches, got %d", count)
	}

	// Check highlights exist for the right rows
	if len(vt.SearchHighlights[0]) != 1 {
		t.Fatalf("expected 1 highlight on row 0, got %d", len(vt.SearchHighlights[0]))
	}
	if len(vt.SearchHighlights[1]) != 1 {
		t.Fatalf("expected 1 highlight on row 1, got %d", len(vt.SearchHighlights[1]))
	}

	// Row 0: "hello" at cols [0, 5)
	hl := vt.SearchHighlights[0][0]
	if hl.Col != 0 || hl.End != 5 {
		t.Fatalf("row 0 highlight: got [%d, %d), want [0, 5)", hl.Col, hl.End)
	}

	// Row 1: "foo bar hello" → "hello" at col 8
	hl = vt.SearchHighlights[1][0]
	if hl.Col != 8 || hl.End != 13 {
		t.Fatalf("row 1 highlight: got [%d, %d), want [8, 13)", hl.Col, hl.End)
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	vt := NewTerminal(10, 40)
	vt.Write([]byte("Hello HELLO hElLo\r\n"))

	count := vt.Search("hello")
	if count != 3 {
		t.Fatalf("expected 3 matches, got %d", count)
	}
}

func TestSearchSetCurrent(t *testing.T) {
	vt := NewTerminal(10, 40)
	vt.Write([]byte("aaa\r\naaa\r\naaa\r\n"))

	vt.Search("aaa")

	row, col := vt.SearchSetCurrent(1)
	if row != 1 || col != 0 {
		t.Fatalf("expected (1, 0), got (%d, %d)", row, col)
	}

	// Check that match 1 is marked current
	hl := vt.SearchHighlights[1][0]
	if !hl.Current {
		t.Fatal("expected match 1 to be current")
	}
	// And match 0 is not
	hl = vt.SearchHighlights[0][0]
	if hl.Current {
		t.Fatal("expected match 0 to not be current")
	}

	// Move current to match 0
	row, col = vt.SearchSetCurrent(0)
	if row != 0 || col != 0 {
		t.Fatalf("expected (0, 0), got (%d, %d)", row, col)
	}
	if !vt.SearchHighlights[0][0].Current {
		t.Fatal("expected match 0 to be current after SetCurrent(0)")
	}
	if vt.SearchHighlights[1][0].Current {
		t.Fatal("expected match 1 to not be current after SetCurrent(0)")
	}
}

func TestSearchClear(t *testing.T) {
	vt := NewTerminal(10, 40)
	vt.Write([]byte("hello\r\n"))
	vt.Search("hello")
	if len(vt.SearchHighlights) == 0 {
		t.Fatal("expected highlights after search")
	}
	vt.SearchClear()
	if vt.SearchHighlights != nil {
		t.Fatal("expected nil highlights after clear")
	}
}

func TestSearchRenderHighlight(t *testing.T) {
	vt := NewTerminal(5, 20)
	vt.Write([]byte("hello world\r\n"))

	vt.Search("world")

	var buf strings.Builder
	err := vt.RenderLine(&buf, 0)
	if err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()
	// The rendered output should contain the highlight styling.
	// SearchMatchStyle uses ANSIYellow bg + ANSIBlack fg.
	// Just check that the output differs from a non-search render.
	vt.SearchClear()
	var buf2 strings.Builder
	vt.RenderLine(&buf2, 0)
	plain := buf2.String()

	if rendered == plain {
		t.Fatal("expected rendered output with search highlights to differ from plain")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	vt := NewTerminal(5, 20)
	vt.Write([]byte("hello\r\n"))

	count := vt.Search("")
	if count != 0 {
		t.Fatalf("expected 0 matches for empty query, got %d", count)
	}
	if vt.SearchHighlights != nil {
		t.Fatal("expected nil highlights for empty query")
	}
}

func TestSearchNoMatches(t *testing.T) {
	vt := NewTerminal(5, 20)
	vt.Write([]byte("hello world\r\n"))

	count := vt.Search("xyz")
	if count != 0 {
		t.Fatalf("expected 0 matches, got %d", count)
	}
}

func TestSearchMultipleOnSameLine(t *testing.T) {
	vt := NewTerminal(5, 40)
	vt.Write([]byte("abcabcabc\r\n"))

	count := vt.Search("abc")
	if count != 3 {
		t.Fatalf("expected 3 matches, got %d", count)
	}

	hls := vt.SearchHighlights[0]
	if len(hls) != 3 {
		t.Fatalf("expected 3 highlights on row 0, got %d", len(hls))
	}
	// Check positions
	expected := [][2]int{{0, 3}, {3, 6}, {6, 9}}
	for i, e := range expected {
		if hls[i].Col != e[0] || hls[i].End != e[1] {
			t.Fatalf("highlight %d: got [%d, %d), want [%d, %d)", i, hls[i].Col, hls[i].End, e[0], e[1])
		}
	}
}
