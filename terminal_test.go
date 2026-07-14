package midterm_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielgatis/go-ansicode"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
)

func mustFprintln(t testing.TB, w io.Writer, a ...any) {
	t.Helper()

	_, err := fmt.Fprintln(w, a...)
	require.NoError(t, err)
}

func mustFprintf(t testing.TB, w io.Writer, format string, a ...any) {
	t.Helper()

	_, err := fmt.Fprintf(w, format, a...)
	require.NoError(t, err)
}

func TestGolden(t *testing.T) {
	ents, err := os.ReadDir(filepath.Join("testdata", "vhs"))
	require.NoError(t, err)

	for _, ent := range ents {
		t.Run(ent.Name(), func(t *testing.T) {
			goldenTest(t, ent.Name())
		})
	}
}

func TestAutoResize(t *testing.T) {
	t.Run("with an initial width", func(t *testing.T) {
		vt := midterm.NewTerminal(0, 5)
		vt.AutoResizeX = true
		vt.AutoResizeY = true
		mustFprintln(t, vt, "hey")
		mustFprintln(t, vt, "yo")
		mustFprintln(t, vt, "im a grower")

		buf := new(bytes.Buffer)
		err := vt.Render(buf)
		require.NoError(t, err)
		require.Equal(t, "hey  \x1b[0m\nyo   \x1b[0m\nim a grower\x1b[0m", buf.String())
	})

	t.Run("without an initial width", func(t *testing.T) {
		vt := midterm.NewAutoResizingTerminal()
		mustFprintln(t, vt, "hey")
		mustFprintln(t, vt, "yo")
		mustFprintln(t, vt, "im a grower")

		buf := new(bytes.Buffer)
		err := vt.Render(buf)
		require.NoError(t, err)
		require.Equal(t, "hey\x1b[0m\nyo\x1b[0m\nim a grower\x1b[0m", buf.String())
	})

	type action struct {
		name   string
		fn     func(*midterm.Terminal)
		height int
	}

	for _, example := range []action{
		{"Backspace", (*midterm.Terminal).Backspace, 1},
		{"CarriageReturn", (*midterm.Terminal).CarriageReturn, 1},
		{"ClearLine right", func(vt *midterm.Terminal) { vt.ClearLine(ansicode.LineClearModeRight) }, 1},
		{"ClearLine left", func(vt *midterm.Terminal) { vt.ClearLine(ansicode.LineClearModeLeft) }, 1},
		{"ClearLine all", func(vt *midterm.Terminal) { vt.ClearLine(ansicode.LineClearModeAll) }, 1},
		{"ClearScreen", func(vt *midterm.Terminal) { vt.ClearScreen(ansicode.ClearModeAll) }, 1},
		{"DeleteChars", func(vt *midterm.Terminal) { vt.DeleteChars(1) }, 1},
		{"DeleteLines", func(vt *midterm.Terminal) { vt.DeleteLines(1) }, 1},
		{"EraseChars", func(vt *midterm.Terminal) { vt.EraseChars(1) }, 1},
		{"Goto", func(vt *midterm.Terminal) { vt.Goto(3, 5) }, 4},
		{"GotoCol", func(vt *midterm.Terminal) { vt.GotoCol(5) }, 1},
		{"GotoLine", func(vt *midterm.Terminal) { vt.GotoLine(5) }, 6},
		{"InsertBlank", func(vt *midterm.Terminal) { vt.InsertBlank(5) }, 1},
		{"InsertBlankLines", func(vt *midterm.Terminal) { vt.InsertBlankLines(5) }, 6},
		{"LineFeed", (*midterm.Terminal).LineFeed, 2},
		{"MoveBackward", func(vt *midterm.Terminal) { vt.MoveBackward(5) }, 1},
		{"MoveDown", func(vt *midterm.Terminal) { vt.MoveDown(5) }, 6},
		{"MoveForward", func(vt *midterm.Terminal) { vt.MoveForward(5) }, 1},
		{"MoveUp", func(vt *midterm.Terminal) { vt.MoveUp(5) }, 1},
		{"ReverseIndex", (*midterm.Terminal).ReverseIndex, 1},
		{"ScrollDown", func(vt *midterm.Terminal) { vt.ScrollDown(5) }, 1},
		{"ScrollUp", func(vt *midterm.Terminal) { vt.ScrollUp(5) }, 1},
		{"SetScrollingRegion", func(vt *midterm.Terminal) { vt.SetScrollingRegion(5, 8) }, 1},
		{"Tab", func(vt *midterm.Terminal) { vt.Tab(3) }, 1},
	} {
		t.Run("handling "+example.name, func(t *testing.T) {
			vt := midterm.NewAutoResizingTerminal()
			example.fn(vt)
			vt.Input('.')
			require.Equal(t, example.height, vt.Height)
		})
	}
}

func goldenTest(t *testing.T, name string) {
	t.Helper()

	file := filepath.Join("testdata", "vhs", name)
	rawOutput, err := os.ReadFile(file)
	require.NoError(t, err)

	buf := bytes.NewBuffer(rawOutput)
	escs := bytes.Count(buf.Bytes(), []byte("\x1b"))
	const targetFrames = 1000
	skipFrames := escs / targetFrames
	if skipFrames < 1 {
		skipFrames = 1
	}

	g := goldie.New(t)

	vt := midterm.NewTerminal(24, 120)
	eachNthFrame(buf, skipFrames, func(frame int, segment []byte) {
		frameLogs := new(bytes.Buffer)
		midterm.DebugLogsTo(frameLogs)

		if testing.Verbose() {
			t.Logf("frame: %d, writing: %q", frame, string(segment))
		}

		n, err := vt.Write(segment)
		require.NoError(t, err)
		require.Equal(t, len(segment), n)

		buf := new(bytes.Buffer)
		err = vt.Render(buf)
		require.NoError(t, err)

		for _, l := range strings.Split(frameLogs.String(), "\n") {
			if strings.Contains(l, "TODO") {
				t.Error(l)
			} else if testing.Verbose() {
				t.Log(l)
			}
		}

		t.Run(fmt.Sprintf("frame %d", frame), func(t *testing.T) {
			t.Log(frameLogs.String())

			framePath := filepath.Join("frames", name, fmt.Sprintf("%05d", frame))
			g.Assert(t, framePath, buf.Bytes())
			expected, err := os.ReadFile(filepath.Join("testdata", framePath) + ".golden")
			require.NoError(t, err)
			if t.Failed() {
				t.Log("expected:")
				t.Log("\n" + string(expected))
				t.Log("actual:")
				t.Log("\n" + buf.String())

				t.Logf("frame mismatch: %d", frame)
				t.Logf("after writing: %q", string(segment))

				eRows := strings.Split(string(expected), "\n")
				aRows := strings.Split(buf.String(), "\n")
				for i := 0; i < len(eRows); i++ {
					if i >= len(aRows) {
						t.Logf("expected: %q", eRows[i])
						t.Logf("actual: nothing")
						break
					}
					if eRows[i] != aRows[i] {
						t.Logf("expected: %q", eRows[i])
						t.Logf("actual  : %q", aRows[i])
					}
				}
			}
		})
	})
	require.NoError(t, err)
}

func TestResizeSmallerWidth(t *testing.T) {
	vt := midterm.NewTerminal(24, 200)

	for i := range 24 {
		mustFprintf(t, vt, "Line %d with some content\n", i)
	}

	// Resize to a much smaller width - this should not panic
	vt.Resize(24, 100)

	buf := new(bytes.Buffer)
	err := vt.Render(buf)
	require.NoError(t, err)
}

func TestResizeGrowingHeightThenShrinkWidth(t *testing.T) {
	vt := midterm.NewTerminal(20, 200)

	for i := range 20 {
		mustFprintf(t, vt, "Line %d with content\n", i)
	}

	// Resize: grow height, shrink width
	// This triggers resizeY to grow, then resizeX to shrink
	vt.Resize(30, 188)

	buf := new(bytes.Buffer)
	err := vt.Render(buf)
	require.NoError(t, err)
}

// TestResizeGrowingHeightDoesNotBloatCanvas pins the size invariants
// after a Resize() that grows the height. Before the fix to resizeY's
// grow path, the inner clear() call mutated v.Height while the outer
// loop was still using it as the offset for the next row, so each
// new row compounded the height by row+1 per column. Concretely,
// NewTerminal(24, 40) + Resize(29, 50) ended with Height=624 and
// len(Content)=1219 instead of 29 and 29. Embedders (terminal
// multiplexers, agent dashboards) saw the side-effect as their host
// renderer clipping content off-screen.
func TestResizeGrowingHeightDoesNotBloatCanvas(t *testing.T) {
	cases := []struct {
		startRows, startCols int
		targetRows           int
		targetCols           int
	}{
		// Original reproduction: small initial size, modest grow, both axes.
		{24, 40, 29, 50},
		// Grow only height.
		{20, 80, 50, 80},
		// Grow only width (control — was never buggy, included for completeness).
		{30, 60, 30, 120},
		// Grow then grow again from a state that previously couldn't survive
		// a single Resize without bloat.
		{24, 40, 100, 100},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("from_%dx%d_to_%dx%d", tc.startRows, tc.startCols, tc.targetRows, tc.targetCols), func(t *testing.T) {
			vt := midterm.NewTerminal(tc.startRows, tc.startCols)
			vt.Resize(tc.targetRows, tc.targetCols)

			require.Equal(t, tc.targetRows, vt.Height, "vt.Height should equal target rows")
			require.Equal(t, tc.targetCols, vt.Width, "vt.Width should equal target cols")
			require.Equal(t, tc.targetRows, len(vt.Content), "len(vt.Content) should equal target rows")
			require.Equal(t, tc.targetRows, len(vt.Changes), "len(vt.Changes) should equal target rows")
			for i, row := range vt.Content {
				require.Equal(t, tc.targetCols, len(row), "Content row %d width should equal target cols", i)
			}

			// Render must produce exactly target-rows lines so embedders
			// don't see phantom rows past the visible area.
			buf := new(bytes.Buffer)
			require.NoError(t, vt.Render(buf))
			require.Equal(t, tc.targetRows, strings.Count(buf.String(), "\n")+1, "rendered line count should equal target rows")
		})
	}
}

// TestResizeNarrowThenRedrawDoesNotPanic guards Canvas.ResizeX against
// leaving canvas.Width stale after a width shrink. canvas.Width is the
// seed size for every later row (re)init — Paint/Insert bootstrap,
// ResizeY grow, and ClearRow (ESC[2J). If it stays at the old, wider
// value, the next row re-init builds a format region wider than the
// now-narrower content row, and renderLine's line[pos:pos+region.Size]
// slice overruns it (panic: slice bounds out of range).
func TestResizeNarrowThenRedrawDoesNotPanic(t *testing.T) {
	// Wide initial grid: canvas.Width is seeded at 201.
	vt := midterm.NewTerminal(24, 201)

	// Shrink to a narrower width, as a window resize would.
	vt.Resize(24, 95)

	// Full-screen redraw after the resize: clear screen (ESC[2J re-inits
	// each row at canvas.Width), home, then paint.
	mustFprintf(t, vt, "\x1b[2J\x1b[Hhello")

	buf := new(bytes.Buffer)
	require.NoError(t, vt.Render(buf))

	// Growing taller after the shrink also appends rows at canvas.Width;
	// clear and repaint, then re-render the whole (now taller) grid.
	vt.Resize(40, 95)
	mustFprintf(t, vt, "\x1b[2J\x1b[40;1Hbottom")
	require.NoError(t, vt.Render(buf))
}

// TestOnScrollback verifies the OnScrollback hook fires for each line
// pushed off the top of the main screen - in order, with its content and
// formatting - and stays silent on the alt screen, which has no
// scrollback.
func TestOnScrollback(t *testing.T) {
	t.Run("fires for lines scrolled off the main screen", func(t *testing.T) {
		vt := midterm.NewTerminal(3, 20)

		var got []string
		vt.OnScrollback(func(line midterm.Line) {
			got = append(got, strings.TrimRight(string(line.Content), " "))
		})

		// Five lines into a three-row screen: the first two fall into history.
		mustFprintf(t, vt, "line 1\r\nline 2\r\nline 3\r\nline 4\r\nline 5")

		require.Equal(t, []string{"line 1", "line 2"}, got)
	})

	t.Run("preserves the evicted line's formatting", func(t *testing.T) {
		vt := midterm.NewTerminal(2, 10)

		var got []midterm.Line
		vt.OnScrollback(func(line midterm.Line) {
			got = append(got, line)
		})

		// Bold the first line, then push it off the two-row screen.
		mustFprintf(t, vt, "\x1b[1mAB\x1b[0m\r\nplain\r\nx")

		require.Len(t, got, 1)
		require.Len(t, got[0].Format, len(got[0].Content))
		require.True(t, got[0].Format[0].IsBold())
		require.True(t, got[0].Format[1].IsBold())
	})

	t.Run("stays silent on the alt screen", func(t *testing.T) {
		vt := midterm.NewTerminal(3, 20)

		var calls int
		vt.OnScrollback(func(midterm.Line) { calls++ })

		// Enter the alt screen (DECSET 1049), then overflow it.
		mustFprintf(t, vt, "\x1b[?1049h")
		require.True(t, vt.IsAlt)
		mustFprintf(t, vt, "line 1\r\nline 2\r\nline 3\r\nline 4\r\nline 5")

		require.Zero(t, calls)
	})

	t.Run("stays silent for a bounded scroll region", func(t *testing.T) {
		vt := midterm.NewTerminal(5, 20)

		var calls int
		vt.OnScrollback(func(midterm.Line) { calls++ })

		// A scroll region smaller than the screen isn't history-producing.
		vt.SetScrollingRegion(2, 4)
		vt.ScrollUp(3)

		require.Zero(t, calls)
	})
}

// TestRenderResetsClearedAttributes verifies renderLine resets SGR state
// between regions only when the next region drops an attribute the previous
// one set - so attributes don't bleed forward, without resetting needlessly.
func TestRenderResetsClearedAttributes(t *testing.T) {
	const reset = "\x1b[0m"

	// between returns the bytes emitted between the "AB" and "CD" runs.
	between := func(out string) string {
		return out[strings.Index(out, "AB")+2 : strings.Index(out, "CD")]
	}

	t.Run("resets when a region drops an attribute", func(t *testing.T) {
		vt := midterm.NewTerminal(1, 4)
		// reverse+red "AB", then plain red "CD" (SGR 27 cancels reverse) -
		// reverse must not bleed into CD.
		mustFprintf(t, vt, "\x1b[7;31mAB\x1b[27mCD")

		var buf bytes.Buffer
		require.NoError(t, vt.Render(&buf))
		require.Contains(t, between(buf.String()), reset)
	})

	t.Run("does not reset when a region only adds attributes", func(t *testing.T) {
		vt := midterm.NewTerminal(1, 4)
		// red "AB", then reverse+red "CD" - red carries over, no reset needed.
		mustFprintf(t, vt, "\x1b[31mAB\x1b[7mCD")

		var buf bytes.Buffer
		require.NoError(t, vt.Render(&buf))
		require.NotContains(t, between(buf.String()), reset)
	})
}

func TestInsertModePreservesShiftedContentAcrossLines(t *testing.T) {
	term := midterm.NewTerminal(24, 80)
	term.Raw = true

	// Seed the two-line prompt before the redraw:
	//
	// Please answer: first
	// (▼ for other options)
	_, err := io.WriteString(term, "\r\x1b[JPlease answer: first \r\n(▼ for other options)")
	require.NoError(t, err)

	// Replace "first" with "second",
	// and insert "▲" and "▼" on the next line
	// so the existing space shifts:
	//
	// Please answer: second
	// (▲▼ for other options)
	_, err = io.WriteString(term, "\x1b[A\x1b[6Dsecond\x1b[4h \x1b[4l\r\n\x1b[C▲\x1b[4h▼\x1b[4l")
	require.NoError(t, err)

	require.Equal(t, "Please answer: second", strings.TrimRight(string(term.Content[0]), " "))
	require.Equal(t, "(▲▼ for other options)", strings.TrimRight(string(term.Content[1]), " "))
}

func TestInsertModeShiftsSingleLineContent(t *testing.T) {
	term := midterm.NewTerminal(1, 8)
	term.Raw = true

	// Start with the cursor after the final "e":
	//
	// abcde^
	_, err := io.WriteString(term, "abcde")
	require.NoError(t, err)

	// Return to column 3, enable insert mode, and insert a space:
	//
	// abc^de
	// abc ^de
	_, err = io.WriteString(term, "\r\x1b[3C\x1b[4h \x1b[4l")
	require.NoError(t, err)

	require.Equal(t, "abc de", strings.TrimRight(string(term.Content[0]), " "))
}

func TestUnsetInsertModeRestoresReplaceMode(t *testing.T) {
	term := midterm.NewTerminal(1, 8)
	term.Raw = true

	// Start with the cursor after the final "e":
	//
	// abcde^
	_, err := io.WriteString(term, "abcde")
	require.NoError(t, err)

	// Insert a space at column 3, disable insert mode, then overwrite the "d"
	// with "Z" at the next cursor position:
	//
	// abc^de
	// abc ^de
	// abc Z^e
	_, err = io.WriteString(term, "\r\x1b[3C\x1b[4h \x1b[4lZ")
	require.NoError(t, err)

	require.Equal(t, "abc Ze", strings.TrimRight(string(term.Content[0]), " "))
}

func eachNthFrame(r io.Reader, n int, callback func(frame int, segment []byte)) {
	const esc = 0x1b

	var frame int
	var segment []byte

	maybeCall := func() {
		frame++
		if frame%n == 0 {
			callback(frame, segment)
			segment = segment[:0]
		}
	}

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return
		}

		for i := 0; i < n; i++ {
			if buf[i] == esc {
				maybeCall()
			}

			segment = append(segment, buf[i])
		}

		if err == io.EOF {
			break
		}
	}

	if len(segment) > 0 {
		maybeCall()
	}
}
