package midterm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
)

func BenchmarkGolden(t *testing.B) {
	ents, err := os.ReadDir(filepath.Join("testdata", "vhs"))
	require.NoError(t, err)

	for _, ent := range ents {
		file := filepath.Join("testdata", "vhs", ent.Name())
		rawOutput, err := os.ReadFile(file)
		require.NoError(t, err)
		t.Run(ent.Name(), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				vt := midterm.NewTerminal(24, 120)
				vt.Write(rawOutput)
			}
		})
	}
}
