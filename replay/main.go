package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/vito/vt100"
	"golang.org/x/term"
)

const cols = 120
const rows = 24

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: vt100 <record|replay> <output>")
		os.Exit(1)
	}
	cmd, out := os.Args[1], os.Args[2]
	switch cmd {
	case "record":
		if err := record(out); err != nil {
			log.Fatal(err)
		}
	case "replay":
		if err := replay(out); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("Usage: vt100 <record|replay> <output>")
		os.Exit(1)
	}
}

func replay(out string) error {
	vt := vt100.NewVT100(rows, cols)
	vt.CursorVisible = true

	content, err := os.ReadFile(out)
	if err != nil {
		return err
	}

	// vt.Write(content)
	fields := splitBefore(content, []byte("\x1b"))

	for _, f := range fields {
		before := new(bytes.Buffer)
		renderVt(before, vt)
		vt.Write(f)
		after := new(bytes.Buffer)
		renderVt(after, vt)
		fmt.Printf("------------------------------------------------------------------------------------------- %q\n", f)
		renderVt(os.Stdout, vt)
		// if regexp.MustCompile("\x1b" + `\[.*[TS]`).Match(after.Bytes()) {
		// 	fmt.Print(before.String())
		// 	fmt.Printf("Wrote %q\n", string(f))
		// 	fmt.Print(after.String())
		// }
	}

	renderVt(os.Stdout, vt)

	return nil
}

func splitBefore(data []byte, delim []byte) [][]byte {
	if len(delim) == 0 {
		return [][]byte{data}
	}
	var result [][]byte
	start := 0
	for i := 0; i+len(delim) <= len(data); i++ {
		if bytes.Equal(data[i:i+len(delim)], delim) {
			result = append(result, data[start:i])
			start = i
		}
	}
	result = append(result, data[start:])
	return result
}

func record(out string) error {
	// Create arbitrary command.
	c := exec.Command("bash")

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Start the command with a pty.
	ptmx, err := pty.StartWithSize(c, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() { _ = ptmx.Close() }() // Best effort.

	vt := vt100.NewVT100(rows, cols)
	vt.CursorVisible = true
	vt.ForwardResponses = ptmx
	vt.ForwardRequests = os.Stdout

	prog := tea.NewProgram(&vtModel{vt}, tea.WithInput(nil))

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go io.Copy(ptmx, os.Stdin)

	raw, err := os.Create(out)
	if err != nil {
		return err
	}

	go func() {
		for {
			buf := make([]byte, 32*1024)
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			raw.Write(buf[:n])
			prog.Send(ptyMsg(buf[:n]))
		}

		prog.Quit()
	}()

	_, err = prog.Run()
	if err != nil {
		return err
	}

	return nil
}

type vtModel struct {
	vt *vt100.VT100
}

func (m vtModel) Init() tea.Cmd {
	return nil
}

type exitMsg error

type ptyMsg []byte

func (m vtModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case exitMsg:
		return m, tea.Quit
	case ptyMsg:
		m.vt.Write(x)
	}
	return m, nil
}

func (m vtModel) View() string {
	buf := new(bytes.Buffer)
	renderVt(buf, m.vt)
	return buf.String()
}

func renderVt(w io.Writer, vt *vt100.VT100) {
	for i := 0; i < vt.Height; i++ {
		vt.RenderLine(w, i)
	}
}
