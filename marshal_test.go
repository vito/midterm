package midterm_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
)

func TestMarshalBinary(t *testing.T) {
	ents, err := os.ReadDir(filepath.Join("testdata", "vhs"))
	require.NoError(t, err)

	for _, ent := range ents {
		t.Run(ent.Name(), func(t *testing.T) {
			marshalBinaryTest(t, ent.Name())
		})
	}
}

func marshalBinaryTest(t *testing.T, name string) {
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

		success := t.Run(fmt.Sprintf("frame %d", frame), func(t *testing.T) {
			t.Log(frameLogs.String())

			clone := midterm.NewTerminal(24, 120)
			reproducer, err := vt.MarshalBinary()
			require.NoError(t, err)
			n, err = clone.Write(reproducer)
			require.NoError(t, err)
			require.Equal(t, len(reproducer), n)

			// Smooth over state that currently can't be encoded/decoded into identical state
			reconcileScreen(vt.Screen, clone.Screen)
			if vt.Alt != nil && clone.Alt != nil {
				reconcileScreen(vt.Alt, clone.Alt)
			}
			clone.Decoder = vt.Decoder

			require.Equal(t, vt, clone)
		})

		if !success {
			t.FailNow()
		}
	})
	require.NoError(t, err)
}

func reconcileScreen(expected *midterm.Screen, actual *midterm.Screen) {
	actual.Changes = expected.Changes
	if actual.MaxY < expected.MaxY {
		actual.MaxY = expected.MaxY
	}
	if actual.MaxX < expected.MaxX {
		actual.MaxX = expected.MaxX
	}

	reconcileFormat(&expected.Cursor.F, &actual.Cursor.F)

	var expectedRegion *midterm.Region
	for row := 0; row < len(expected.Format.Rows); row++ {
		expectedRegion = expected.Format.Rows[row]
		actualRegion := actual.Format.Rows[row]
		pos := 0
		for expectedRegion != nil && actualRegion != nil {
			reconcileFormat(&expectedRegion.F, &actualRegion.F)
			if expectedRegion.Size < actualRegion.Size {
				//Split the region into two
				actualRegion.Next = &midterm.Region{
					F:    actualRegion.F,
					Size: actualRegion.Size - expectedRegion.Size,
					Next: actualRegion.Next,
				}
				actualRegion.Size = expectedRegion.Size
			} else if actualRegion.Size < expectedRegion.Size {
				//Try to merge a chunk of the next region into the current one as long as the content is just spaces
				for actualRegion.Size < expectedRegion.Size && actualRegion.Next != nil {
					if actual.Content[row][pos+actualRegion.Size] != ' ' {
						break
					}
					actualRegion.Size += 1
					actualRegion.Next.Size -= 1
					if actualRegion.Next.Size == 0 {
						//We've fully merged the region. Clean it up.
						if actualRegion.Next.Next != nil && actualRegion.Next.Next.F.Properties&midterm.ResetBit == 0 {
							//It seems regions don't fully encapsulate their state and rely on the regions before them.
							forward := actualRegion.Next.Next
							backward := actualRegion.Next
							if forward.F.Fg == nil {
								forward.F.Fg = backward.F.Fg
							}
							if forward.F.Bg == nil {
								forward.F.Bg = backward.F.Bg
							}
							forward.F.Properties |= backward.F.Properties
						}
						actualRegion.Next = actualRegion.Next.Next
					}
				}
			}
			pos += expectedRegion.Size
			if pos > expected.MaxX {
				//Regions greater than max x don't get serialized so just copy them over from the expected value
				actualRegion.Next = expectedRegion.Next
				break
			}

			expectedRegion = expectedRegion.Next
			actualRegion = actualRegion.Next
		}
	}
}

func reconcileFormat(expected *midterm.Format, actual *midterm.Format) {
	//Force reset bit to match
	actual.Properties = (actual.Properties & ^midterm.ResetBit) | (expected.Properties & midterm.ResetBit)
}
