package midterm

import (
	"fmt"
)

func swapAlt(v *Terminal) {
	v.IsAlt = !v.IsAlt
	v.Screen, v.Alt = v.Alt, v.Screen
}

var Reset = Format{Properties: ResetBit}
var EmptyFormat = Format{}

func sanitize(v *Terminal, y, x int) (int, int, error) {
	var err error
	if y < 0 || y >= v.Height || x < 0 || (!v.AutoResizeX && x >= v.Width) {
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
	if !v.AutoResizeX && x >= v.Width {
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
