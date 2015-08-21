package vt100

import (
	"errors"
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

const (
	// When the terminal asks that the color be set to a number,
	// that number less the base is the vt100.Color type we should
	// set for the corresponding region (background or foreground)
	fgBase = 30
	bgBase = 40
)

// command is a type of object that knows how to display itself
// to the terminal.
type command interface {
	display(v *VT100)
}

// runeCommand is a simple command that just writes a rune
// to the current cell and advances the cursor.
type runeCommand rune

func (r runeCommand) display(v *VT100) {
	v.put(rune(r))
}

// escapeCommand is a control sequence command. It includes a variety
// of control and escape sequences that move and modify the cursor
// or the terminal.
type escapeCommand struct {
	cmd  rune
	args string
}

type intHandler func(*VT100, []int)

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
		'K': eraseColumns,
		'J': eraseLines,
		'H': home,
		'f': home,
		'm': updateAttributes,
	}
)

func save(v *VT100, _ []int) {
	v.save()
}

func unsave(v *VT100, _ []int) {
	v.unsave()
}

var (
	codeColors = []color.RGBA{
		Black,
		Red,
		Green,
		Yellow,
		Blue,
		Magenta,
		Cyan,
		White,
	}
)

// A command to update the attributes of the cursor based on the arg list.
func updateAttributes(v *VT100, args []int) {
	f := &v.Cursor.F

	for _, x := range args {
		switch x {
		case 0:
			*f = Format{}
		case 1:
			f.Intensity = Bright
		case 2:
			f.Intensity = Dim
		case 22:
			f.Intensity = Normal
		case 4:
			f.Underscore = true
		case 24:
			f.Underscore = false
		case 5, 6:
			f.Blink = true // We don't distinguish between blink speeds.
		case 25:
			f.Blink = false
		case 7:
			f.Inverse = true
		case 27:
			f.Inverse = false
		case 8:
			f.Conceal = true
		case 28:
			f.Conceal = false
		case 30, 31, 32, 33, 34, 35, 36, 37:
			f.Fg = codeColors[x-30]
		case 39:
			f.Fg = DefaultColor
		case 40, 41, 42, 43, 44, 45, 46, 47:
			f.Bg = codeColors[x-40]
		case 49:
			f.Bg = DefaultColor
			// 38 and 48 not supported. Maybe someday.
		}
	}
}

func relativeMove(y, x int) func(*VT100, []int) {
	return func(v *VT100, args []int) {
		c := 1
		if len(args) >= 1 {
			c = args[0]
		}
		v.move(y*c, x*c)
	}
}

func eraseColumns(v *VT100, args []int) {
	d := eraseForward
	if len(args) > 0 {
		d = eraseDirection(args[0])
	}
	v.eraseColumns(d)
}

func eraseLines(v *VT100, args []int) {
	d := eraseForward
	if len(args) > 0 {
		d = eraseDirection(args[0])
	}
	v.eraseLines(d)
}

func home(v *VT100, args []int) {
	var y, x int
	if len(args) >= 2 {
		y, x = args[0]-1, args[1]-1 // home args are 1-indexed.
	}
	v.home(y, x)
}

func (c escapeCommand) display(v *VT100) {
	f, ok := intHandlers[c.cmd]
	if !ok {
		c.log(errors.New("unsupported command"))
		return
	}

	args, err := c.argInts()
	if err != nil {
		err = fmt.Errorf("while parsing int args: %v", err)
		c.log(err)
		v.Err = err
	}

	f(v, args)
}

// log logs a problem with a escapeCommand at the warning level. Generally speaking,
// only parse errors and unsupported commands will be logged. The idea here
// is that, through logs, we'll be able to figure out what codes are unsupported
// but actually used that we need to implement.
func (c escapeCommand) log(e error) {
	glog.Warningf("[%v] %s", e)
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

// newEscapeCommand makes a new control sequence command from cmd and args.
func newEscapeCommand(cmd rune, args string) escapeCommand {
	return escapeCommand{cmd, args}
}

type controlCommand rune

const (
	backspace      controlCommand = '\b'
	_horizontalTab                = '\t'
	linefeed                      = '\n'
	_verticalTab                  = '\v'
	_formfeed                     = '\f'
	carriageReturn                = '\r'
)

func (c controlCommand) display(v *VT100) {
	switch c {
	case backspace:
		v.backspace()
	case linefeed:
		v.Cursor.Y++
		v.Cursor.X = 0
	case carriageReturn:
		v.Cursor.X = 0
	default:
		v.Err = fmt.Errorf("control code not implemented %0x", uint(c))
		glog.Warning(v.Err)
	}

}
