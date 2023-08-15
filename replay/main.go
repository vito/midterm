package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/creack/pty"
	"github.com/vito/vt100"
	"golang.org/x/term"
)

const cols = 120
const rows = 24

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: vt100 <record|repro> <output>")
		os.Exit(1)
	}
	cmd, out := os.Args[1], os.Args[2]
	switch cmd {
	case "record":
		if err := record(out); err != nil {
			log.Fatal(err)
		}
	case "repro":
		if err := repro(out); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("Usage: vt100 <record|repro> <output>")
		os.Exit(1)
	}
}

func repro(out string) error {
	vt := vt100.NewVT100(rows, cols)

	content, err := os.ReadFile(out)
	if err != nil {
		return err
	}

	// vt.Write(content)
	fields := bytes.SplitAfter(content, []byte("\n"))

	for _, f := range fields {
		before := new(bytes.Buffer)
		renderVt(before, vt)
		vt.Write(f)
		after := new(bytes.Buffer)
		renderVt(after, vt)
		fmt.Println("-------------------------------------------------------------------------------------------")
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

	prog := tea.NewProgram(&vtModel{vt}, tea.WithInput(nil))

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go io.Copy(ptmx, os.Stdin)

	raw, err := os.Create(out)
	if err != nil {
		return err
	}

	go func() {
		io.Copy(io.MultiWriter(vt, raw), ptmx)
		prog.Quit()
	}()

	m, err := prog.Run()
	if err != nil {
		return err
	}

	fmt.Print(m.View())
	return nil
}

type vtModel struct {
	vt *vt100.VT100
}

func (m vtModel) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(100, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type tickMsg time.Time

type exitMsg error

func (m vtModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		return m, tick()
	case exitMsg:
		return m, tea.Quit
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
		fmt.Fprintln(w)
	}
}
