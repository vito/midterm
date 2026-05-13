package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/vito/midterm"
	"golang.org/x/term"
)

const cols = 120
const rows = 24

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <record|replay> <output> [cmd...]\n", os.Args[0])
		os.Exit(1)
	}
	sub, out, cmd := os.Args[1], os.Args[2], os.Args[3:]
	switch sub {
	case "record":
		if err := record(out, cmd); err != nil {
			log.Fatal(err)
		}
	case "replay":
		if err := replay(out); err != nil {
			log.Fatal(err)
		}
	case "replay-logs":
		if err := replayLogs(out); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Printf("Usage: %s <record|replay> <output>\n", os.Args[0])
		os.Exit(1)
	}
}

func record(out string, cmd []string) error {
	if len(cmd) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "sh"
		}
		cmd = []string{shell}
	}

	// Create arbitrary command.
	c := exec.Command(cmd[0], cmd[1:]...)

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

	vt := midterm.NewTerminal(rows, cols)
	vt.Raw = true
	vt.CursorVisible = true
	vt.ForwardResponses = ptmx
	vt.ForwardRequests = os.Stdout

	prog := tea.NewProgram(&vtModel{vt}, tea.WithInput(nil))

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	raw, err := os.Create(out)
	if err != nil {
		return err
	}
	defer func() { _ = raw.Close() }() // Best effort.

	go func() {
		for {
			buf := make([]byte, 32*1024)
			n, err := ptmx.Read(buf)
			if err != nil {
				break
			}
			if _, err := raw.Write(buf[:n]); err != nil {
				prog.Send(exitMsg(err))
				break
			}
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

func replay(out string) error {
	vt := midterm.NewTerminal(rows, cols)
	vt.Raw = true
	vt.CursorVisible = true

	content, err := os.ReadFile(out)
	if err != nil {
		return err
	}

	// vt.Write(content)
	fields := splitBefore(content, []byte("\x1b"))

	for _, f := range fields {
		before := new(bytes.Buffer)
		if err := renderVt(before, vt); err != nil {
			return err
		}
		prev := vt.Cursor
		if _, err := vt.Write(f); err != nil {
			return err
		}
		after := new(bytes.Buffer)
		if err := renderVt(after, vt); err != nil {
			return err
		}
		fmt.Printf("------------------------------------------------------------------------------------------- (%d,%d) %q\n", prev.X, prev.Y, f)
		if err := renderVt(os.Stdout, vt); err != nil {
			return err
		}
		// if regexp.MustCompile("\x1b" + `\[.*[TS]`).Match(after.Bytes()) {
		// 	fmt.Print(before.String())
		// 	fmt.Printf("Wrote %q\n", string(f))
		// 	fmt.Print(after.String())
		// }
	}

	if err := renderVt(os.Stdout, vt); err != nil {
		return err
	}

	return nil
}

func replayLogs(out string) error {
	vt := midterm.NewAutoResizingTerminal()
	vt.AppendOnly = true
	// midterm.DebugLogsTo(os.Stderr)

	f, err := os.Open(out)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // Best effort.

	scan := bufio.NewScanner(f)
	lines := 0
	for scan.Scan() {
		lines++
		// fmt.Fprintf(os.Stderr, "%d WRITING: %q\n", lines, scan.Text())
		if _, err := vt.Write(scan.Bytes()); err != nil {
			return err
		}
		vt.LineFeed()
	}
	if err := scan.Err(); err != nil {
		return err
	}

	if err := renderVt(os.Stdout, vt); err != nil {
		return err
	}

	return nil
}

type vtModel struct {
	vt *midterm.Terminal
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
		if _, err := m.vt.Write(x); err != nil {
			return m, func() tea.Msg { return exitMsg(err) }
		}
	}
	return m, nil
}

func (m vtModel) View() string {
	buf := new(bytes.Buffer)
	if err := renderVt(buf, m.vt); err != nil {
		return err.Error()
	}
	return buf.String()
}

func renderVt(w io.Writer, vt *midterm.Terminal) error {
	for i := 0; i < vt.Height; i++ {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if err := vt.RenderLine(w, i); err != nil {
			return err
		}
	}
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
