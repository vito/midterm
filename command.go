package vt100

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/muesli/termenv"
)

func init() {
	f, _ := os.Create("/tmp/vt100.log")
	log.SetOutput(f)
}

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

// escapeCommand is a control sequence command. It includes a variety
// of control and escape sequences that move and modify the cursor
// or the terminal.
type escapeCommand struct {
	cmd  rune
	args string
}

func (c escapeCommand) String() string {
	return fmt.Sprintf("%T[%q %U](%v)", c, c.cmd, c.cmd, c.args)
}

type intHandler func(*VT100, []int) error

var (
	// intHandlers are handlers for which all arguments are numbers.
	// This is most of them -- all the ones that we process. Eventually,
	// we may add handlers that support non-int args. Those handlers
	// will instead receive []string, and they'll have to choose on their
	// own how they might be parsed.
	intHandlers = map[rune]intHandler{
		's': save,
		'7': save,
		'u': unsave,
		'8': unsave,
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
		'h': noop, // TODO DECSET
		'l': noop, // TODO DECSET
		't': noop, // TODO unknown, htop uses it. save xterm window/icon? https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h4-Functions-using-CSI-_-ordered-by-the-final-character-lparen-s-rparen:CSI-Ps;Ps;Ps-t.1EB0
		'r': func(v *VT100, args []int) error {
			log.Println("SET SCROLL REGION", args)
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
		'=': noop, // TODO keypad??? htop again
		'(': noop, // TODO unknown
		')': noop, // TODO unknown
		'*': noop, // TODO unknown
		'+': noop, // TODO unknown
		'-': noop, // TODO unknown
		'.': noop, // TODO unknown
		'/': noop, // TODO unknown
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
		'>': noop, // TODO unknown
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
		'n': noop, // TODO query?
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
		']': noop, // TODO OS Command
		'c': noop, // TODO Device Attributes
		'q': noop, // TODO Set Cursor Style
	}
)

type resetCommand struct{}

func (resetCommand) String() string {
	return "<reset>"
}

func (resetCommand) display(v *VT100) error {
	v.Reset()
	return nil
}

type noopCommand struct{}

func (noopCommand) String() string {
	return "<noop>"
}

func (noopCommand) display(v *VT100) error {
	return nil
}

func noop(v *VT100, args []int) error {
	// TODO?
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
			f.Fg = termenv.ANSIColor(x - 90 + 8)
		case 40, 41, 42, 43, 44, 45, 46, 47:
			f.Bg = termenv.ANSIColor(x - 40)
		case 49:
			f.Bg = nil
		case 100, 101, 102, 103, 104, 105, 106, 107:
			f.Bg = termenv.ANSIColor(x - 100 + 8)
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

func (c escapeCommand) display(v *VT100) error {
	f, ok := intHandlers[c.cmd]
	if !ok {
		log.Println("UNSUPPORTED COMMAND", c)
		return supportError(c.err(errors.New("unsupported command")))
	}

	args, err := c.argInts()
	if err != nil {
		return c.err(fmt.Errorf("while parsing int args: %v", err))
	}

	return f(v, args)
}

// err enhances e with information about the current escape command
func (c escapeCommand) err(e error) error {
	return fmt.Errorf("%s: %s", c, e)
}

var csArgsRe = regexp.MustCompile("^([^0-9]*)(.*)$")

// argInts parses c.args as a slice of at least arity ints. If the number
// of ; separated arguments is less than arity, the remaining elements of
// the result will be zero. errors only on integer parsing failure.
func (c escapeCommand) argInts() ([]int, error) {
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

type controlCommand rune

const (
	backspace      controlCommand = '\b'
	horizontalTab  controlCommand = '\t'
	linefeed       controlCommand = '\n'
	_verticalTab   controlCommand = '\v'
	_formfeed      controlCommand = '\f'
	carriageReturn controlCommand = '\r'
)

const tabWidth = 4

func (c controlCommand) String() string {
	return fmt.Sprintf("%T(%q)", c, string(c))
}

func (c controlCommand) display(v *VT100) error {
	switch c {
	case backspace:
		v.backspace()
	case linefeed:
		v.Cursor.X = 0

		if v.ScrollRegion != nil && v.Cursor.Y == v.ScrollRegion.End {
			// if we're at the bottom of the scroll region, scroll it instead of
			// moving the cursor
			v.scrollUpN(1)
			return nil
		}

		v.Cursor.Y++
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
