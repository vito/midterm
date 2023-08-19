package vt100_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
	"github.com/vito/vt100"
)

func TestGolden(t *testing.T) {
	ents, err := os.ReadDir(filepath.Join("testdata", "vhs"))
	require.NoError(t, err)

	for _, ent := range ents {
		t.Run(ent.Name(), func(t *testing.T) {
			goldenTest(t, ent.Name())
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

	vt := vt100.NewVT100(24, 120)
	err = eachNthFrame(buf, skipFrames, func(frame int, segment []byte) error {
		n, err := vt.Write(segment)
		require.NoError(t, err)
		require.Equal(t, len(segment), n)

		buf := new(bytes.Buffer)
		err = vt.Render(buf)
		require.NoError(t, err)

		framePath := filepath.Join("frames", name, fmt.Sprintf("%05d", frame))
		g.Assert(t, framePath, buf.Bytes())
		if t.Failed() {
			t.FailNow()
		}

		return nil
	})
	require.NoError(t, err)
}

func eachFrame(r io.Reader, callback func(frame int, segment []byte) error) error {
	return eachNthFrame(r, 1, callback)
}

func eachNthFrame(r io.Reader, n int, callback func(frame int, segment []byte) error) error {
	const esc = 0x1b

	var frame int
	var segment []byte

	maybeCall := func() error {
		frame++
		if frame%n == 0 {
			if err := callback(frame, segment); err != nil {
				return err
			}

			segment = segment[:0]
		}
		return nil
	}

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}

		for i := 0; i < n; i++ {
			if buf[i] == esc {
				if err := maybeCall(); err != nil {
					return err
				}
			}

			segment = append(segment, buf[i])
		}

		if err == io.EOF {
			break
		}
	}

	if len(segment) > 0 {
		if err := maybeCall(); err != nil {
			return err
		}
	}

	return nil
}
