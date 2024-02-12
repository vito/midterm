package main_test

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
)

func TestGhosttyValidations(t *testing.T) {
	ghostDir := os.Getenv("GHOSTTY_DIR")
	if ghostDir == "" {
		t.Skip("GHOSTTY_DIR not set")
	}
	docsDir := filepath.Join(ghostDir, "website", "app", "vt")

	mdx, err := filepath.Glob(filepath.Join(docsDir, "**", "*.mdx"))
	require.NoError(t, err)

	for _, doc := range mdx {
		doc := doc
		dir := filepath.Base(filepath.Dir(doc))
		content, err := os.ReadFile(doc)
		require.NoError(t, err)

		docs, tests := parseTests(t, content)

		if len(tests) == 0 {
			continue
		}

		md, err := glamour.Render(docs, "auto")
		require.NoError(t, err)

		t.Run(dir, func(t *testing.T) {
			for _, test := range tests {
				test := test
				t.Run(strings.ReplaceAll(test.Name, "/", "-"), func(t *testing.T) {
					t.Log("docs:\n" + strings.TrimSpace(md))

					lines := strings.Split(test.Out, "\n")
					rows := 10                // arbitrary
					cols := len(lines[0]) - 2 // account for '|' on either side
					term := midterm.NewTerminal(rows, cols)
					term.AutoResizeY = true

					t.Logf("running:\n%s", test.Bash)

					cmd := exec.Command("bash", "-c", test.Bash)
					cmd.Stdout = term
					cmd.Stderr = term
					require.NoError(t, cmd.Run())

					actual := ""
					for r, row := range term.Content {
						if r+1 >= len(lines) {
							// there isn't a fixed height, so we just compare whatever lines
							// the output expects
							break
						}
						actual += "|"
						for c, cell := range row {
							if cell == ' ' {
								if c == term.Cursor.X && r == term.Cursor.Y {
									actual += "c"
								} else {
									actual += "_"
								}
							} else {
								actual += string(cell)
							}
						}
						actual += "|"
						actual += "\n"
					}
					require.Equal(t, test.Out, actual)
					// require.Equal(t, test.Out, string(out))
				})
			}
		})
	}
}

type Test struct {
	Name string
	Bash string
	Out  string
}

func parseTests(t *testing.T, content []byte) (string, []Test) {
	tests := []Test{}

	scan := bufio.NewScanner(bytes.NewBuffer(content))

	docs := ""

	var inValidations bool
	var inBash bool
	var inOut bool
	var currentTest Test
	for scan.Scan() {
		switch {
		case scan.Text() == "## Validation":
			// t.Logf("enter validations")
			inValidations = true
		case !inValidations:
			docs += scan.Text() + "\n"
			continue
		case currentTest.Name == "":
			_, name, ok := strings.Cut(scan.Text(), "### ")
			if !ok {
				continue
			}
			// t.Logf("got name: %q", name)
			currentTest.Name = name
		case currentTest.Bash == "" && !inBash:
			if scan.Text() == "```bash" {
				// t.Logf("enter bash")
				inBash = true
				continue
			}
		case inBash:
			if scan.Text() == "```" {
				// t.Logf("exit bash")
				inBash = false
				continue
			}
			// t.Logf("got bash: %q", scan.Text())
			currentTest.Bash += scan.Text() + "\n"
		case currentTest.Out == "" && !inOut:
			if scan.Text() == "```" {
				// t.Logf("enter out")
				inOut = true
				continue
			}
		case inOut:
			if scan.Text() == "```" {
				// t.Logf("exit out, done with test: %+v", currentTest)
				inOut = false
				tests = append(tests, currentTest)
				currentTest = Test{}
				continue
			}
			// t.Logf("got out: %q", scan.Text())
			currentTest.Out += scan.Text() + "\n"
		}
	}

	return docs, tests
}
