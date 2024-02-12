package midterm

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
	display(v *Terminal) error
}

// runeCommand is a simple command that just writes a rune
// to the current cell and advances the cursor.
type runeCommand rune

func (c runeCommand) String() string {
	return fmt.Sprintf("%T(%q)", c, string(c))
}

func (c runeCommand) display(v *Terminal) error {
	v.put(rune(c))
	return nil
}

func setMode(v *Terminal, args []string) error {
	if len(args) == 0 {
		dbg.Println("EMPTY SET MODE")
		return nil
	}

	mode := args[0]

	var forward bool
	switch mode {
	case "4":
		dbg.Println("SET RESET MODE")
	case "?1":
		dbg.Println("SET APP CURSOR MODE")
		forward = true
	case "?7":
		dbg.Println("SET WRAP MODE")
	case "?12":
		dbg.Println("SET BLINK CURSOR MODE")
		epoch := time.Now()
		v.CursorBlinkEpoch = &epoch
	case "?25":
		dbg.Println("SET CURSOR VISIBLE")
		v.CursorVisible = true
	case "?1000", // basic
		"?1002", // drag
		"?1003", // all mouse controls
		"?1006": // extended mouse coords
		dbg.Println("SET MOUSE TRACKING MODE", mode)
		forward = true
	case "?1004": // window focus
		dbg.Println("SET WINDOW FOCUS TRACKING MODE", mode)
		forward = true
	case "?1049":
		dbg.Println("SET ALT SCREEN")
		if v.IsAlt {
			dbg.Println("ALREADY ALT")
		} else {
			dbg.Println("SWITCHING TO ALT")
			if v.Alt == nil {
				dbg.Println("ALLOCATING ALT SCREEN")
				v.Alt = newScreen(v.Height, v.Width)
			}
			swapAlt(v)
		}
	case "?2004":
		dbg.Println("SET BRACKETED PASTE")
		forward = true
	default:
		dbg.Println("SET UNKNOWN MODE", mode)
	}

	if forward && v.ForwardRequests != nil {
		fmt.Fprintf(v.ForwardRequests, "\x1b[%sh", mode)
	}

	return nil
}

func swapAlt(v *Terminal) {
	v.IsAlt = !v.IsAlt
	v.Screen, v.Alt = v.Alt, v.Screen
}

func unsetMode(v *Terminal, args []string) error {
	if len(args) == 0 {
		dbg.Println("EMPTY UNSET MODE")
		return nil
	}
	mode := args[0]
	var forward bool
	switch mode {
	case "4":
		dbg.Println("UNSET RESET MODE")
	case "?1":
		dbg.Println("UNSET APP CURSOR MODE")
		forward = true
	case "?7":
		dbg.Println("UNSET WRAP MODE")
	case "?12":
		dbg.Println("UNSET BLINK CURSOR MODE")
		v.CursorBlinkEpoch = nil
	case "?25":
		dbg.Println("UNSET CURSOR VISIBLE")
		v.CursorVisible = false
	case "?1000", // basic
		"?1002", // drag
		"?1003", // all mouse controls
		"?1006": // extended mouse coords
		dbg.Println("UNSET MOUSE TRACKING MODE", mode)
		forward = true
	case "?1004": // window focus
		dbg.Println("UNSET WINDOW FOCUS TRACKING MODE", mode)
		forward = true
	case "?1049":
		dbg.Println("UNSET ALT SCREEN")
		if !v.IsAlt {
			dbg.Println("ALREADY NOT ALT")
		} else {
			dbg.Println("RESTORING MAIN SCREEN")
			swapAlt(v)
		}
	case "?2004":
		dbg.Println("UNSET BRACKETED PASTE")
		forward = true
	default:
		dbg.Println("UNSET UNKNOWN MODE", mode)
	}

	if forward && v.ForwardRequests != nil {
		fmt.Fprintf(v.ForwardRequests, "\x1b[%sl", mode)
	}

	return nil
}

type escHandler func(*Terminal, string) error

type intHandler func(*Terminal, []int) error

type strHandler func(*Terminal, []string) error

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
		']': func(v *Terminal, arg string) error { // OSC (OS Command)
			args := strings.Split(arg, ";")
			if len(args) == 0 {
				dbg.Println("EMPTY OSC")
				return nil
			}
			switch args[0] {
			case "52":
				dbg.Println("FORWARDING OSC 52 REQUEST", arg)
				fmt.Fprintf(v.ForwardRequests, "\x1b]%s\x07", arg)
			case "112":
				dbg.Println("IGNORING OSC RESET CURSOR COLOR")
			default:
				dbg.Println("IGNORING UNKNOWN OSC", arg)
			}
			return nil
		},
		'7': func(v *Terminal, _ string) error {
			v.save()
			return nil
		},
		'8': func(v *Terminal, _ string) error {
			v.unsave()
			return nil
		},
		'D': func(v *Terminal, arg string) error {
			v.moveDown()
			return nil
		},
		'M': func(v *Terminal, arg string) error {
			v.moveUp()
			return nil
		},
		'c': func(v *Terminal, arg string) error {
			v.reset()
			return nil
		},
	}

	csiStrHandlers = map[rune]strHandler{
		'h': setMode,
		'l': unsetMode,
		's': save,
		'u': unsave,           // NB: vim prints \e[?u on start - a bit of a mystery
		'q': noopStr,          // Set Cursor Style; vim prints \e[2 q which is another mystery
		'm': updateAttributes, // NB: 'm' usually has int args, except for CSI > Pp m and CSI ? Pp m
		'r': setScrollRegion,
	}

	csiIntHandlers = map[rune]intHandler{
		'c': func(v *Terminal, args []int) error { // Request terminal attributes
			dbg.Println("SENDING DEVICE ATTRIBUTES", args)
			if v.ForwardResponses == nil {
				dbg.Println("NO RESPONSE CHANNEL", args)
				return nil
			}
			dbg.Println("RESPONDING VT102")
			fmt.Fprint(v.ForwardResponses, termenv.CSI+"?6c")
			return nil
		},
		'A': relativeMove(-1, 0),
		'B': relativeMove(1, 0),
		'C': relativeMove(0, 1),
		'D': relativeMove(0, -1),
		'G': absoluteMove,
		'H': home,
		'J': eraseLines,
		'K': eraseColumns,
		'f': home,
		't': noopInt, // Window manipulation
		'd': func(v *Terminal, args []int) error {
			y := 1
			if len(args) >= 1 {
				y = args[0]
			}

			// NB: home is 1-indexed, hence the +1.
			return home(v, []int{y, v.Cursor.X + 1})
		},
		// scroll down N times
		'T': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.scrollDownN(n)

			return nil
		},
		'S': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.scrollUpN(n)

			return nil
		},
		'L': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.insertLines(n)

			return nil
		},
		'M': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.deleteLines(n)

			return nil
		},
		'n': func(v *Terminal, args []int) error { // Device Status Report
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
		'X': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.eraseCharacters(n)

			return nil
		},
		'b': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.repeatPrecedingCharacter(n)

			return nil
		},
		'P': func(v *Terminal, args []int) error {
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.deleteCharacters(n)

			return nil
		},
		'@': func(v *Terminal, args []int) error {
			dbg.Println("INSERTING", args)
			n := 1
			if len(args) >= 1 {
				n = args[0]
			}

			v.insertCharacters(n)

			return nil
		},
	}
)

func setScrollRegion(v *Terminal, args []string) error {
	switch len(args) {
	case 0:
		dbg.Println("RESETTING SCROLL REGION", args)
		v.ScrollRegion = nil
	case 1:
		dbg.Println("UNKNOWN CSI r", args)
	case 2:
		dbg.Println("SETTING SCROLL REGION r", args)

		var start, end int

		if args[0] == "" {
			start = 1
		} else {
			line, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			start = line
		}

		if args[1] == "" {
			end = v.Height
		} else {
			line, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			end = line
		}

		if start == 1 && end == v.Height {
			// equivalent to just resetting
			v.ScrollRegion = nil
		} else {
			v.ScrollRegion = &ScrollRegion{
				Start: start - 1,
				End:   end - 1,
			}
		}
	}
	// Reset cursor position and wrap state
	// TODO: respect origin mode
	v.home(0, 0)
	return nil
}

type noopCommand struct{}

func (noopCommand) String() string {
	return "<noop>"
}

func (noopCommand) display(v *Terminal) error {
	return nil
}

func noopInt(v *Terminal, args []int) error {
	return nil
}

func noopStr(v *Terminal, args []string) error {
	return nil
}

func noopEsc(v *Terminal, arg string) error {
	return nil
}

func save(v *Terminal, _ []string) error {
	v.save()
	return nil
}

func unsave(v *Terminal, _ []string) error {
	v.unsave()
	return nil
}

// A command to update the attributes of the cursor based on the arg list.
func updateAttributes(v *Terminal, args []string) error {
	f := &v.Cursor.F
	if len(args) == 0 {
		*f = Format{Reset: true}
		return nil
	}

	// forward modifier key requests, which are confusingly _also_ the 'm' CSI
	// command, just with a different prefix. c'mon man
	if strings.HasPrefix(args[0], ">") || strings.HasPrefix(args[0], "?") {
		if v.ForwardRequests != nil {
			fmt.Fprintf(v.ForwardRequests, "%s%sm", termenv.CSI, strings.Join(args, ";"))
		}
		return nil
	}

	var unsupported []int
	i := 0
	for i < len(args) {
		x, err := strconv.Atoi(args[i])
		if err != nil {
			return err
		}

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

			type_, err := strconv.Atoi(args[i])
			if err != nil {
				return err
			}

			i++

			var color termenv.Color
			switch type_ {
			case 5: // 256-color
				if len(args) < 3 {
					return fmt.Errorf("malformed 8- or 24-bit flags: %q", args)
				}

				num, err := strconv.Atoi(args[i])
				if err != nil {
					return err
				}

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

				r, err := strconv.Atoi(args[i])
				if err != nil {
					return err
				}
				i++
				g, err := strconv.Atoi(args[i])
				if err != nil {
					return err
				}
				i++
				b, err := strconv.Atoi(args[i])
				if err != nil {
					return err
				}
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

func relativeMove(y, x int) func(*Terminal, []int) error {
	return func(v *Terminal, args []int) error {
		c := 1
		if len(args) >= 1 {
			c = args[0]
		}
		// home is 1-indexed, because that's what the terminal sends us. We want to
		// reuse its sanitization scheme, so we'll just modify our args by that amount.
		return home(v, []int{(v.Cursor.Y + 1) + y*c, (v.Cursor.X + 1) + x*c})
	}
}

func absoluteMove(v *Terminal, args []int) error {
	x := 1
	if len(args) >= 1 {
		x = args[0]
	}

	// NB: home is 1-indexed, hence the +1.
	return home(v, []int{v.Cursor.Y + 1, x})
}

func eraseColumns(v *Terminal, args []int) error {
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

func eraseLines(v *Terminal, args []int) error {
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

func sanitize(v *Terminal, y, x int) (int, int, error) {
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

func home(v *Terminal, args []int) error {
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

func (c escCommand) display(v *Terminal) error {
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

func (c csiCommand) display(v *Terminal) error {
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
		return []int{}, nil
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
	if len(c.args) == 0 {
		return []string{}
	}
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

func (c controlCommand) display(v *Terminal) error {
	switch c {
	case backspace:
		v.backspace()
	case carriageReturn:
		v.Cursor.X = 0
		v.wrap = false
	case linefeed:
		if !v.Raw {
			// in "cooked" mode, commonly used for displaying logs, \n implies \r\n
			v.Cursor.X = 0
		}
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
	}
	return nil
}
