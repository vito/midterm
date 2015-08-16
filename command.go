package vt100

import (
	"fmt"
	"github.com/golang/glog"
	"regexp"
	"strconv"
	"strings"
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

// putRuneCommand is a simple command that just writes a rune
// to the current cell and advances the cursor.
type putRuneCommand rune

func (r putRuneCommand) display(v *VT100) {
	v.put(rune(r))
}

// csCommand is a control sequence command. It includes a variety
// of control and escape sequences that move and modify the cursor
// or the terminal.
type csCommand struct {
	cmd  rune
	args string
}

type nullaryHandler func(*VT100)
type unaryHandler func(*VT100, int)
type binaryHandler func(*VT100, int, int)
type naryHandler func(*VT100, ...int)

var (
	nullaryCommands = map[rune]nullaryHandler{
		's': (*VT100).save,
		'7': (*VT100).save,
		'u': (*VT100).unsave,
		'8': (*VT100).unsave,
	}
	unaryCommands = map[rune]unaryHandler{
		'A': relativeMove(-1, 0),
		'B': relativeMove(1, 0),
		'C': relativeMove(0, 1),
		'D': relativeMove(0, -1),
		'K': func(v *VT100, lineDir int) {
			v.eraseDirection(eraseDirection(lineDir))
		},
		'J': func(v *VT100, sel int) {
			v.eraseDirection(eraseDirection(int(eraseDown) + sel))
		},
	}
	binaryCommands = map[rune]binaryHandler{
		'H': (*VT100).home,
		'f': (*VT100).home,
	}
	naryCommands = map[rune]naryHandler{}
)

func relativeMove(y, x int) func(*VT100, int) {
	return func(v *VT100, count int) {
		v.move(y*count, x*count)
	}
}

func (c csCommand) getHandler() interface{} {
	if f, ok := nullaryCommands[c.cmd]; ok {
		return f
	} else if f, ok := unaryCommands[c.cmd]; ok {
		return f
	} else if f, ok := binaryCommands[c.cmd]; ok {
		return f
	} else if f, ok := naryCommands[c.cmd]; ok {
		return f
	} else {
		return nil
	}
}

func (c csCommand) display(v *VT100) {
	var err error
	// Declaring this here rather than using short assignment allows us to
	// share the error handling.
	var a []int

	handler := c.getHandler()
	switch f := handler.(type) {
	case nullaryHandler:
		f(v)
	case unaryHandler:
		a, err = c.argInts(1)
		if err == nil {
			f(v, a[0])
		}
	case binaryHandler:
		a, err = c.argInts(2)
		if err == nil {
			f(v, a[0], a[1])
		}
	case naryHandler:
		a, err = c.argInts(0)
		if err == nil {
			f(v, a...)
		}
	case nil:
		c.log("unsupported command")
	}
	if err != nil {
		c.log("while parsing args: %v", err)
		v.Err = err
	}
}

// log logs a problem with a csCommand at the warning level. Generally speaking,
// only parse errors and unsupported commands will be logged. The idea here
// is that, through logs, we'll be able to figure out what codes are unsupported
// but actually used that we need to implement.
func (c csCommand) log(format string, x ...interface{}) {
	glog.Warningf("[%v] %s", fmt.Sprintf(format, x...))
}

var csArgsRe = regexp.MustCompile("^([^0-9]*)(.*)$")

// argInts parses c.args as a slice of at least arity ints. If the number
// of ; separated arguments is less than arity, the remaining elements of
// the result will be zero. errors only on integer parsing failure.
func (c csCommand) argInts(arity int) ([]int, error) {
	args := strings.Split(c.args, ";")
	if arity < len(args) {
		arity = len(args)
	}
	out := make([]int, arity)
	for i, s := range args {
		x, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return nil, err
		}
		out[i] = int(x)
	}
	return out, nil
}

// newCSCommand makes a new control sequence command from cmd and args.
func newCSCommand(cmd rune, args string) csCommand {
	return csCommand{cmd, args}
}
