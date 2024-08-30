package midterm

import (
	"sort"
	"strings"

	"github.com/muesli/termenv"
)

// Constants for property bit positions
const (
	ResetBit uint8 = 1 << iota
	BoldBit
	FaintBit
	ItalicBit
	UnderlineBit
	BlinkBit
	ReverseBit
	ConcealBit
	// not worth the memory footprint
	// CrossOutBit
	// OverlineBit
)

// Format represents the text formatting options.
type Format struct {
	// Fg and Bg are the foreground and background colors.
	Fg, Bg termenv.Color

	// Properties packed into a single byte.
	Properties uint8
}

// Helper methods to set properties

func (f *Format) SetReset(value bool) {
	f.setProperty(ResetBit, value)
}

func (f *Format) IsReset() bool {
	return f.hasProperty(ResetBit)
}

func (f *Format) SetBold(value bool) {
	f.setProperty(BoldBit, value)
}

func (f *Format) IsBold() bool {
	return f.hasProperty(BoldBit)
}

func (f *Format) SetFaint(value bool) {
	f.setProperty(FaintBit, value)
}

func (f *Format) IsFaint() bool {
	return f.hasProperty(FaintBit)
}

func (f *Format) SetItalic(value bool) {
	f.setProperty(ItalicBit, value)
}

func (f *Format) IsItalic() bool {
	return f.hasProperty(ItalicBit)
}

func (f *Format) SetUnderline(value bool) {
	f.setProperty(UnderlineBit, value)
}

func (f *Format) IsUnderline() bool {
	return f.hasProperty(UnderlineBit)
}

func (f *Format) SetBlink(value bool) {
	f.setProperty(BlinkBit, value)
}

func (f *Format) IsBlink() bool {
	return f.hasProperty(BlinkBit)
}

func (f *Format) SetReverse(value bool) {
	f.setProperty(ReverseBit, value)
}

func (f *Format) IsReverse() bool {
	return f.hasProperty(ReverseBit)
}

func (f *Format) SetConceal(value bool) {
	f.setProperty(ConcealBit, value)
}

func (f *Format) IsConceal() bool {
	return f.hasProperty(ConcealBit)
}

//	func (f *Format) SetCrossOut(value bool) {
//		f.setProperty(CrossOutBit, value)
//	}
//
//	func (f *Format) IsCrossOut() bool {
//		return f.hasProperty(CrossOutBit)
//	}
//
//	func (f *Format) SetOverline(value bool) {
//		f.setProperty(OverlineBit, value)
//	}
//
//	func (f *Format) IsOverline() bool {
//		return f.hasProperty(OverlineBit)
//	}

// Helper method to set a property bit
func (f *Format) setProperty(bit uint8, value bool) {
	if value {
		f.Properties |= bit
	} else {
		f.Properties &^= bit
	}
}

// Helper method to check if a property bit is set
func (f *Format) hasProperty(bit uint8) bool {
	return f.Properties&bit != 0
}

func toCss(c termenv.Color) string {
	return termenv.ConvertToRGB(c).Hex()
}

func (f Format) css() string {
	parts := make([]string, 0)
	fg, bg := f.Fg, f.Bg
	if f.IsReverse() {
		bg, fg = fg, bg
	}

	parts = append(parts, "color:"+toCss(fg))
	parts = append(parts, "background-color:"+toCss(bg))
	if f.IsBold() {
		parts = append(parts, "font-weight:bold")
	}
	if f.IsFaint() {
		parts = append(parts, "opacity:0.33")
	}
	if f.IsUnderline() {
		parts = append(parts, "text-decoration:underline")
	}
	if f.IsConceal() {
		parts = append(parts, "display:none")
	}
	if f.IsBlink() {
		parts = append(parts, "text-decoration:blink")
	}

	// We're not in performance sensitive code. Although this sort
	// isn't strictly necessary, it gives us the nice property that
	// the style of a particular set of attributes will always be
	// generated the same way. As a result, we can use the html
	// output in tests.
	sort.StringSlice(parts).Sort()

	return strings.Join(parts, ";")
}
