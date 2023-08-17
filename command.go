package vt100

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

// UnsupportedError indicates that we parsed an operation that this
// terminal does not implement. Such errors indicate that the client
// program asked us to perform an action that we don't know how to.
// It MAY be safe to continue trying to do additional operations.
// This is a distinct category of errors from things we do know how
// to do, but are badly encoded, or errors from the underlying io.RuneScanner
// that we're reading commands from.
type UnsupportedError struct {
	error
}

func supportError(e error) error {
	return UnsupportedError{e}
}

// Command is a type of object that the terminal can process to perform
// an update.
type Command interface {
	display(v *VT100) error
}

// runeCommand is a simple command that just writes a rune
// to the current cell and advances the cursor.
type runeCommand rune

func (c runeCommand) String() string {
	return fmt.Sprintf("%T(%q)", c, string(c))
}

func (c runeCommand) display(v *VT100) error {
	v.put(rune(c))
	return nil
}

func setMode(v *VT100, mode []string) error {
	if len(mode) == 0 {
		dbg.Println("EMPTY SET MODE")
		return nil
	}
	switch mode[0] {
	case "4":
		dbg.Println("SET RESET MODE")
	case "?1":
		dbg.Println("SET APP CURSOR MODE")
		if v.ForwardRequests != nil {
			fmt.Fprintf(v.ForwardRequests, "\x1b[?1h")
		}
	case "?7":
		dbg.Println("SET WRAP MODE")
	case "?12":
		epoch := time.Now()
		v.CursorBlinkEpoch = &epoch
	case "?25":
		v.CursorVisible = true
	case "?1000":
		dbg.Println("SET MOUSE TRACKING MODE")
	case "?1049":
		dbg.Println("SET ALT SCREEN")
	case "?2004":
		dbg.Println("SET BRACKETED PASTE")
	default:
		dbg.Println("SET UNKNOWN MODE", mode)
	}
	return nil
}

func unsetMode(v *VT100, mode []string) error {
	if len(mode) == 0 {
		dbg.Println("EMPTY UNSET MODE")
		return nil
	}
	switch mode[0] {
	case "4":
		dbg.Println("UNSET RESET MODE")
	case "?1":
		if v.ForwardRequests != nil {
			fmt.Fprintf(v.ForwardRequests, "\x1b[?1l")
		}
	case "?7":
		dbg.Println("UNSET WRAP MODE")
	case "?12":
		v.CursorBlinkEpoch = nil
	case "?25":
		v.CursorVisible = false
	case "?1000":
		dbg.Println("UNSET MOUSE TRACKING MODE")
	case "?1049":
		dbg.Println("UNSET ALT SCREEN")
	case "?2004":
		dbg.Println("UNSET BRACKETED PASTE")
	default:
		dbg.Println("UNSET UNKNOWN MODE", mode)
	}
	return nil
}

type escHandler func(*VT100, string) error

type intHandler func(*VT100, []int) error

type strHandler func(*VT100, []string) error

var (
	escHandlers = map[rune]escHandler{
		'(': noopEsc, // Character sets
		')': noopEsc, // Character sets
		'*': noopEsc, // Character sets
		'+': noopEsc, // Character sets
		'-': noopEsc, // Character sets
		'.': noopEsc, // Character sets
		'/': noopEsc, // Character sets
		'=': noopEsc, // Application keypad
		'>': noopEsc, // Normal keypad
		']': func(v *VT100, arg string) error { // OSC (OS Command)
			args := strings.Split(arg, ";")
			if len(args) == 0 {
				dbg.Println("EMPTY OSC")
				return nil
			}
			switch args[0] {
			case "52":
				// forward along
				if strings.HasSuffix(arg, "?") {
					dbg.Println("FORWARDING OSC 52 REQUEST", arg)
					fmt.Fprintf(v.ForwardRequests, "\x1b]%s\x07", args)
				} else {
					dbg.Println("FORWARDING OSC 52 RESPONSE", arg)
					fmt.Fprintf(v.ForwardResponses, "\x1b]%s\x07", args)
				}
			}
			return nil
		},
		'7': func(v *VT100, _ string) error {
			v.save()
			return nil
		},
		'8': func(v *VT100, _ string) error {
			v.unsave()
			return nil
		},
		'D': func(v *VT100, arg string) error {
			v.moveDown()
			return nil
		},
		'M': func(v *VT100, arg string) error {
			v.moveUp()
			return nil
		},
		'c': func(v *VT100, arg string) error {
			v.Reset()
			return nil
		},
	}

	csiStrHandlers = map[rune]strHandler{
		'h': setMode,
		'l': unsetMode,
	}

	csiIntHandlers = map[rune]intHandler{
		's': save,
		'u': unsave,
		'A': relativeMove(-1, 0),
		'B': relativeMove(1, 0),
		'C': relativeMove(0, 1),
		'D': relativeMove(0, -1),
		'G': absoluteMove,
		'H': home,
		'J': eraseLines,
		'K': eraseColumns,
		'f': home,
		'm': updateAttributes,
		't': noopInt, // Window manipulation
		'r': func(v *VT100, args []int) error {
			switch len(args) {
			case 0:
				v.ScrollRegion = nil
			case 1: // TODO: handle \e[;10r and \e[10;r
			case 2:
				start, end := args[0]-1, args[1]-1
				if start == 0 && end == v.Height-1 {
					// equivalent to just resetting
					v.ScrollRegion = nil
				} else {
					v.ScrollRegion = &ScrollRegion{
						Start: start,
						End:   end,
					}
				}
			}
			return nil
		},
		'd': func(v *VT100, args []int) error {
			y := 1
			if len(args) >= 1 {
				y = args[0]
			}

			// NB: home is 1-indexed, hence the +1.
			return home(v, []int{y, v.Cursor.X - 1})
		},
		// scroll down N times
		'T': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.scrollDownN(n)

			return nil
		},
		'S': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.scrollUpN(n)

			return nil
		},
		'L': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.insertLines(n)

			return nil
		},
		'M': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.deleteLines(n)

			return nil
		},
		'n': func(v *VT100, args []int) error { // Device Status Report
			dbg.Println("QUERYING", args)
			if len(args) == 0 {
				dbg.Println("EMPTY QUERY?", args)
				return nil
			}
			if v.ForwardResponses == nil {
				dbg.Println("NO RESPONSE CHANNEL", args)
				return nil
			}
			switch args[0] {
			case 5:
				fmt.Fprint(v.ForwardResponses, termenv.CSI+"0n")
			case 6:
				fmt.Fprintf(v.ForwardResponses, "%s%d;%dR", termenv.CSI, v.Cursor.Y+1, v.Cursor.X+1)
			default:
				dbg.Println("UNKNOWN QUERY", args)
			}
			return nil
		},
		'X': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.eraseCharacters(n)

			return nil
		},
		'b': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.repeatPrecedingCharacter(n)

			return nil
		},
		'P': func(v *VT100, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.deleteCharacters(n)

			return nil
		},
		// 'c': noop, // TODO Device Attributes
		// 'q': noop, // TODO Set Cursor Style
	}
)

type noopCommand struct{}

func (noopCommand) String() string {
	return "<noop>"
}

func (noopCommand) display(v *VT100) error {
	return nil
}

func noopInt(v *VT100, args []int) error {
	return nil
}

func noopStr(v *VT100, args []string) error {
	return nil
}

func noopEsc(v *VT100, arg string) error {
	return nil
}

func save(v *VT100, _ []int) error {
	v.save()
	return nil
}

func unsave(v *VT100, _ []int) error {
	v.unsave()
	return nil
}

// A command to update the attributes of the cursor based on the arg list.
func updateAttributes(v *VT100, args []int) error {
	f := &v.Cursor.F
	if len(args) == 0 {
		*f = Format{Reset: true}
		return nil
	}

	var unsupported []int
	i := 0
	for i < len(args) {
		x := args[i]
		i++

		switch x {
		case 0:
			*f = Format{Reset: true}
		case 1:
			f.Intensity = Bold
		case 2:
			f.Intensity = Faint
		case 3:
			f.Italic = true
		case 22:
			f.Intensity = Normal
		case 4:
			f.Underline = true
		case 24:
			f.Underline = false
		case 5, 6:
			f.Blink = true // We don't distinguish between blink speeds.
		case 25:
			f.Blink = false
		case 7:
			f.Reverse = true
		case 27:
			f.Reverse = false
		case 8:
			f.Conceal = true
		case 28:
			f.Conceal = false
		case 30, 31, 32, 33, 34, 35, 36, 37:
			f.Fg = termenv.ANSIColor(x - 30)
		case 39:
			f.Fg = nil
		case 90, 91, 92, 93, 94, 95, 96, 97:
			f.Fg = termenv.ANSIColor(x-90) + termenv.ANSIBrightBlack
		case 40, 41, 42, 43, 44, 45, 46, 47:
			f.Bg = termenv.ANSIColor(x - 40)
		case 49:
			f.Bg = nil
		case 100, 101, 102, 103, 104, 105, 106, 107:
			f.Bg = termenv.ANSIColor(x-100) + termenv.ANSIBrightBlack
		case 38, 48: // 256-color foreground/background
			bg := x == 48

			if len(args) < 2 {
				return fmt.Errorf("malformed 8- or 24-bit flags: %q", args)
			}

			type_ := args[i]
			i++

			var color termenv.Color
			switch type_ {
			case 5: // 256-color
				if len(args) < 3 {
					return fmt.Errorf("malformed 8- or 24-bit flags: %q", args)
				}

				num := args[i]
				i++
				switch {
				case num < 16:
					color = termenv.ANSIColor(num)
				default:
					color = termenv.ANSI256Color(num)
				}
			case 2: // 24-bit
				if len(args) < 5 {
					return fmt.Errorf("malformed 8- or 24-bit flags: %q", args)
				}

				r := args[i]
				i++
				g := args[i]
				i++
				b := args[i]
				i++

				color = termenv.RGBColor(fmt.Sprintf("#%02x%02x%02x", r, g, b))
			}

			if bg {
				f.Bg = color
			} else {
				f.Fg = color
			}
		default:
			unsupported = append(unsupported, x)
		}
	}

	if unsupported != nil {
		return supportError(fmt.Errorf("unknown attributes: %v", unsupported))
	}
	return nil
}

func relativeMove(y, x int) func(*VT100, []int) error {
	return func(v *VT100, args []int) error {
		c := 1
		if len(args) >= 1 {
			c = args[0]
		}
		// home is 1-indexed, because that's what the terminal sends us. We want to
		// reuse its sanitization scheme, so we'll just modify our args by that amount.
		return home(v, []int{v.Cursor.Y + y*c + 1, v.Cursor.X + x*c + 1})
	}
}

func absoluteMove(v *VT100, args []int) error {
	x := 1
	if len(args) >= 1 {
		x = args[0]
	}

	// NB: home is 1-indexed, hence the +1.
	return home(v, []int{v.Cursor.Y + 1, x})
}

func eraseColumns(v *VT100, args []int) error {
	d := eraseForward
	if len(args) > 0 {
		d = eraseDirection(args[0])
	}
	if d > eraseAll {
		return fmt.Errorf("unknown erase direction: %d", d)
	}
	v.eraseColumns(d)
	return nil
}

func eraseLines(v *VT100, args []int) error {
	d := eraseForward
	if len(args) > 0 {
		d = eraseDirection(args[0])
	}
	if d > eraseAll {
		return fmt.Errorf("unknown erase direction: %d", d)
	}
	v.eraseLines(d)
	return nil
}

func sanitize(v *VT100, y, x int) (int, int, error) {
	var err error
	if y < 0 || y >= v.Height || x < 0 || x >= v.Width {
		err = fmt.Errorf("out of bounds (%d, %d)", y, x)
	} else {
		return y, x, nil
	}

	if y < 0 {
		y = 0
	}
	if y >= v.Height {
		y = v.Height - 1
	}
	if x < 0 {
		x = 0
	}
	if x >= v.Width {
		x = v.Width - 1
	}
	return y, x, err
}

func home(v *VT100, args []int) error {
	var y, x int
	if len(args) >= 2 {
		y, x = args[0]-1, args[1]-1 // home args are 1-indexed.
	}
	y, x, err := sanitize(v, y, x) // Clamp y and x to the bounds of the terminal.
	v.home(y, x)                   // Try to do something like what the client asked.
	return err
}

// escCommand is an escape sequence command. It includes a variety
type escCommand struct {
	cmd rune
	arg string
}

func (c escCommand) String() string {
	return fmt.Sprintf("%T[%q %U](%v)", c, c.cmd, c.cmd, c.arg)
}

func (c escCommand) display(v *VT100) error {
	f, ok := escHandlers[c.cmd]
	if !ok {
		dbg.Println("UNSUPPORTED COMMAND", c)
		return fmt.Errorf("unsupported escape sequence: %q (%s)", c.cmd, c)
	}

	return f(v, c.arg)
}

// csiCommand is a control sequence command. It includes a variety
// of control and escape sequences that move and modify the cursor
// or the terminal.
type csiCommand struct {
	cmd  rune
	args string
}

func (c csiCommand) String() string {
	return fmt.Sprintf("%T[%q %U](%v)", c, c.cmd, c.cmd, c.args)
}

func (c csiCommand) display(v *VT100) error {
	strF, ok := csiStrHandlers[c.cmd]
	if ok {
		return strF(v, c.argStrs())
	}

	f, ok := csiIntHandlers[c.cmd]
	if !ok {
		dbg.Println("UNSUPPORTED COMMAND", c)
		return supportError(c.err(errors.New("unsupported command")))
	}

	args, err := c.argInts()
	if err != nil {
		return c.err(fmt.Errorf("while parsing int args: %v", err))
	}

	return f(v, args)
}

// err enhances e with information about the current escape command
func (c csiCommand) err(e error) error {
	return fmt.Errorf("%s: %s", c, e)
}

func (c csiCommand) argInts() ([]int, error) {
	if len(c.args) == 0 {
		return make([]int, 0), nil
	}
	args := strings.Split(c.args, ";")
	out := make([]int, len(args))
	for i, s := range args {
		x, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return nil, err
		}
		out[i] = int(x)
	}
	return out, nil
}

func (c csiCommand) argStrs() []string {
	return strings.Split(c.args, ";")
}

type controlCommand rune

const (
	backspace      controlCommand = '\b'
	horizontalTab  controlCommand = '\t'
	linefeed       controlCommand = '\n'
	_verticalTab   controlCommand = '\v'
	_formfeed      controlCommand = '\f'
	carriageReturn controlCommand = '\r'
)

const tabWidth = 8

func (c controlCommand) String() string {
	return fmt.Sprintf("%T(%q)", c, string(c))
}

func (c controlCommand) display(v *VT100) error {
	switch c {
	case backspace:
		v.backspace()
	case linefeed:
		v.Cursor.X = 0 // TODO is this right? seems like we're ignoring raw/cooked
		v.moveDown()
	case horizontalTab:
		target := ((v.Cursor.X / tabWidth) + 1) * tabWidth
		if target >= v.Width {
			target = v.Width - 1
		}
		formatY := v.Format[v.Cursor.Y]
		format := formatY[v.Cursor.X]
		for x := v.Cursor.X; x < target; x++ {
			v.clear(v.Cursor.Y, x, format)
		}
		v.Cursor.X = target
	case carriageReturn:
		v.Cursor.X = 0
	}
	return nil
}
