package midterm

import (
	"fmt"
	"image/color"
	"time"

	"github.com/danielgatis/go-ansicode"
	"github.com/muesli/termenv"
)

// Backspace moves the cursor one position to the left.
func (v *Terminal) Backspace() {
	v.changed(v.Cursor.Y, true)
	v.moveRel(0, -1)
}

// Bell rings the bell.
func (v *Terminal) Bell() {
	// no-op
}

// CarriageReturn moves the cursor to the beginning of the line.
func (v *Terminal) CarriageReturn() {
	v.home(v.Cursor.Y, 0)
}

// ClearLine clears the line.
func (v *Terminal) ClearLine(mode ansicode.LineClearMode) {
	dbg.Println("ClearLine", mode)
	y, x, w := v.Cursor.Y, v.Cursor.X, v.Width
	switch mode {
	case ansicode.LineClearModeRight:
		v.eraseRegion(y, x, y, w-1)
	case ansicode.LineClearModeLeft:
		v.eraseRegion(y, 0, y, x)
	case ansicode.LineClearModeAll:
		v.eraseRegion(y, 0, y, w-1)
	}
}

// ClearScreen clears the screen.
func (v *Terminal) ClearScreen(mode ansicode.ClearMode) {
	dbg.Println("ClearScreen", mode)
	y, x, w, h := v.Cursor.Y, v.Cursor.X, v.Width, v.Height
	switch mode {
	case ansicode.ClearModeBelow:
		v.eraseRegion(y, x, h-1, w-1)
		if y < h-1 {
			v.eraseRegion(y+1, 0, h-1, w-1)
		}
	case ansicode.ClearModeAbove:
		v.eraseRegion(0, 0, y, x)
		if y > 0 {
			v.eraseRegion(0, 0, y-1, w-1)
		}
	case ansicode.ClearModeAll:
		v.eraseRegion(0, 0, h-1, w-1)
	case ansicode.ClearModeSaved:
		dbg.Println("TODO: ClearModeSaved")
	}
}

// ClearTabs clears the tab stops.
func (v *Terminal) ClearTabs(mode ansicode.TabulationClearMode) {
	dbg.Println("TODO: ClearTabs", mode)
}

// ClipboardLoad loads data from the clipboard.
func (v *Terminal) ClipboardLoad(clipboard byte, terminator string) {
	dbg.Printf("TODO: ClipboardLoad: clipboard=%d, terminator=%s\n", clipboard, terminator)
}

// ClipboardStore stores data in the clipboard.
func (v *Terminal) ClipboardStore(clipboard byte, data []byte) {
	dbg.Printf("TODO: ClipboardStore: clipboard=%d, data=%s\n", clipboard, string(data))
}

// ConfigureCharset configures the charset.
func (v *Terminal) ConfigureCharset(index ansicode.CharsetIndex, charset ansicode.Charset) {
	if v.ForwardRequests == nil {
		dbg.Printf("ConfigureCharset: index=%d, charset=%v (ignored)\n", index, charset)
		return
	}
	dbg.Printf("ConfigureCharset: index=%d, charset=%v (forwarding)\n", index, charset)
	var renderedIndex rune
	switch index {
	case ansicode.CharsetIndexG0:
		renderedIndex = '('
	case ansicode.CharsetIndexG1:
		renderedIndex = ')'
	case ansicode.CharsetIndexG2:
		renderedIndex = '*'
	case ansicode.CharsetIndexG3:
		renderedIndex = '+'
	}
	var renderedCharset rune
	switch charset {
	case ansicode.CharsetASCII:
		renderedCharset = 'B'
	case ansicode.CharsetLineDrawing:
		renderedCharset = '0'
	}
	fmt.Fprintf(v.ForwardRequests, "\x1b%c%c", renderedIndex, renderedCharset)
}

// Decaln runs the DECALN command.
func (v *Terminal) Decaln() {
	dbg.Println("TODO: Decaln")
}

// DeleteChars deletes n characters.
func (v *Terminal) DeleteChars(n int) {
	dbg.Printf("DeleteChars: n=%d\n", n)
	v.deleteCharacters(n)
}

// DeleteLines deletes n lines.
func (v *Terminal) DeleteLines(n int) {
	dbg.Printf("DeleteLines: n=%d\n", n)
	v.deleteLines(n)
}

// DeviceStatus reports the device status.
func (v *Terminal) DeviceStatus(n int) {
	dbg.Printf("DeviceStatus: n=%d\n", n)
	if v.ForwardResponses == nil {
		dbg.Println("NO RESPONSE CHANNEL FOR DEVICE STATUS QUERY", n)
		return
	}
	switch n {
	case 5:
		fmt.Fprint(v.ForwardResponses, termenv.CSI+"0n")
	case 6:
		fmt.Fprintf(v.ForwardResponses, "%s%d;%dR", termenv.CSI, v.Cursor.Y+1, v.Cursor.X+1)
	default:
		dbg.Println("UNKNOWN DEVICE STATUS QUERY", n)
	}
}

// EraseChars erases n characters.
func (v *Terminal) EraseChars(n int) {
	dbg.Printf("EraseChars: n=%d\n", n)
	v.eraseCharacters(n)
}

// Goto moves the cursor to the specified position.
func (v *Terminal) Goto(y int, x int) {
	dbg.Printf("Goto: y=%d, x=%d\n", y, x)
	if y == 65535 {
		// BUG: somehow this is what \e[H is being parsed as
		y = 0
	}
	if y >= v.Height && !v.AutoResizeY {
		y = v.Height - 1
	}
	if x >= v.Width && !v.AutoResizeX {
		x = v.Width - 1
	}
	v.home(y, x)
}

// GotoCol moves the cursor to the specified column.
func (v *Terminal) GotoCol(n int) {
	dbg.Printf("GotoCol: n=%d\n", n)
	v.home(v.Cursor.Y, n)
}

// GotoLine moves the cursor to the specified line.
func (v *Terminal) GotoLine(n int) {
	dbg.Printf("GotoLine: n=%d\n", n)
	v.home(n, v.Cursor.X)
}

// HorizontalTab sets the current position as a tab stop.
func (v *Terminal) HorizontalTabSet() {
	dbg.Println("TODO: HorizontalTabSet")
}

// IdentifyTerminal identifies the terminal.
func (v *Terminal) IdentifyTerminal(b byte) {
	dbg.Printf("IdentifyTerminal: b=%d\n", b)
	if v.ForwardResponses == nil {
		dbg.Println("IdentifyTerminal: NO RESPONSE CHANNEL")
		return
	}
	dbg.Println("IdentifyTerminal: RESPONDING VT102")
	fmt.Fprint(v.ForwardResponses, termenv.CSI+"?62;22c") // VT220 + ANSI
}

// Input inputs a rune to be displayed.
func (v *Terminal) Input(r rune) {
	dbg.Printf("Input: %c\n", r)
	v.put(r)
}

// InsertBlank inserts n blank characters.
func (v *Terminal) InsertBlank(n int) {
	dbg.Printf("InsertBlank: n=%d\n", n)
	v.insertCharacters(n)
}

// InsertBlankLines inserts n blank lines.
func (v *Terminal) InsertBlankLines(n int) {
	dbg.Printf("InsertBlankLines: n=%d\n", n)
	v.insertLines(n)
}

// LineFeed moves the cursor to the next line.
func (v *Terminal) LineFeed() {
	dbg.Println("LineFeed")
	if !v.Raw {
		// in "cooked" mode, commonly used for displaying logs, \n implies \r\n
		v.Cursor.X = 0
	}
	v.moveDown()
}

// MoveBackward moves the cursor backward n columns.
func (v *Terminal) MoveBackward(n int) {
	dbg.Printf("MoveBackward: n=%d\n", n)
	v.home(v.Cursor.Y, v.Cursor.X-n)
}

// MoveBackwardTabs moves the cursor backward n tab stops.
func (v *Terminal) MoveBackwardTabs(n int) {
	dbg.Printf("TODO: MoveBackwardTabs: n=%d\n", n)
}

// MoveDown moves the cursor down n lines.
func (v *Terminal) MoveDown(n int) {
	dbg.Printf("MoveDown: n=%d\n", n)
	for i := 0; i < n; i++ {
		v.moveDown()
	}
}

// MoveDownCr moves the cursor down n lines and to the beginning of the line.
func (v *Terminal) MoveDownCr(n int) {
	dbg.Printf("TODO: MoveDownCr: n=%d\n", n)
}

// MoveForward moves the cursor forward n columns.
func (v *Terminal) MoveForward(n int) {
	dbg.Printf("MoveForward: n=%d\n", n)
	v.home(v.Cursor.Y, v.Cursor.X+n)
}

// MoveForwardTabs moves the cursor forward n tab stops.
func (v *Terminal) MoveForwardTabs(n int) {
	dbg.Printf("TODO: MoveForwardTabs: n=%d\n", n)
}

// MoveUp moves the cursor up n lines.
func (v *Terminal) MoveUp(n int) {
	dbg.Printf("MoveUp: n=%d\n", n)
	for i := 0; i < n; i++ {
		v.moveUp()
	}
}

// MoveUpCr moves the cursor up n lines and to the beginning of the line.
func (v *Terminal) MoveUpCr(n int) {
	dbg.Printf("TODO: MoveUpCr: n=%d\n", n)
}

// PopKeyboardMode pops the given amount n of keyboard modes from the stack.
func (v *Terminal) PopKeyboardMode(n int) {
	dbg.Printf("TODO: PopKeyboardMode: n=%d\n", n)
}

// PopTitle pops the title from the stack.
func (v *Terminal) PopTitle() {
	dbg.Println("PopTitle (ignored)")
}

// PushKeyboardMode pushes the given keyboard mode to the stack.
func (v *Terminal) PushKeyboardMode(mode ansicode.KeyboardMode) {
	dbg.Println("TODO: PushKeyboardMode", mode)
}

// PushTitle pushes the given title to the stack.
func (v *Terminal) PushTitle() {
	dbg.Println("PushTitle (ignored)")
}

// ReportKeyboardMode reports the keyboard mode.
func (v *Terminal) ReportKeyboardMode() {
	if v.ForwardResponses == nil {
		dbg.Println("ReportKeyboardMode (ignored)")
		return
	}
	dbg.Println("ReportKeyboardMode (forwarding)")
	fmt.Fprint(v.ForwardResponses, termenv.CSI+"?0u")
}

// ReportModifyOtherKeys reports the modify other keys mode. (XTERM)
func (v *Terminal) ReportModifyOtherKeys() {
	dbg.Println("TODO: ReportModifyOtherKeys")
}

// ResetColor resets the color at the given index.
func (v *Terminal) ResetColor(i int) {
	dbg.Printf("ResetColor: i=%d (ignored)\n", i)
}

// ResetState resets the terminal state.
func (v *Terminal) ResetState() {
	dbg.Println("TODO: ResetState")
}

// RestoreCursorPosition restores the cursor position.
func (v *Terminal) RestoreCursorPosition() {
	dbg.Println("RestoreCursorPosition")
	v.unsave()
}

// ReverseIndex moves the active position to the same horizontal position on the preceding line.
func (v *Terminal) ReverseIndex() {
	dbg.Println("ReverseIndex")
	v.moveUp()
}

// SaveCursorPosition saves the cursor position.
func (v *Terminal) SaveCursorPosition() {
	dbg.Println("SaveCursorPosition")
	v.save()
}

// ScrollDown scrolls the screen down n lines.
func (v *Terminal) ScrollDown(n int) {
	dbg.Printf("ScrollDown: n=%d\n", n)
	v.scrollDownN(n)
}

// ScrollUp scrolls the screen up n lines.
func (v *Terminal) ScrollUp(n int) {
	dbg.Printf("ScrollUp: n=%d\n", n)
	v.scrollUpN(n)
}

// SetActiveCharset sets the active charset.
func (v *Terminal) SetActiveCharset(n int) {
	dbg.Printf("TODO: SetActiveCharset: n=%d\n", n)
}

// SetColor sets the color at the given index.
func (v *Terminal) SetColor(index int, color color.Color) {
	dbg.Printf("TODO: SetColor: index=%d, color=%v\n", index, color)
}

// SetCursorStyle sets the cursor style.
func (v *Terminal) SetCursorStyle(style ansicode.CursorStyle) {
	dbg.Println("SetCursorStyle", style)
	v.Cursor.S = style
}

// SetDynamicColor sets the dynamic color at the given index.
func (v *Terminal) SetDynamicColor(prefix string, index int, terminator string) {
	dbg.Printf("SetDynamicColor: prefix=%s, index=%d, terminator=%s (ignored)\n", prefix, index, terminator)
}

// SetHyperlink sets the hyperlink.
func (v *Terminal) SetHyperlink(hyperlink *ansicode.Hyperlink) {
	dbg.Println("TODO: SetHyperlink", hyperlink)
}

// SetKeyboardMode sets the keyboard mode.
func (v *Terminal) SetKeyboardMode(mode ansicode.KeyboardMode, behavior ansicode.KeyboardModeBehavior) {
	dbg.Printf("TODO: SetKeyboardMode: mode=%v, behavior=%v\n", mode, behavior)
}

// SetKeypadApplicationMode sets keypad to applications mode.
func (v *Terminal) SetKeypadApplicationMode() {
	dbg.Println("SetKeypadApplicationMode (ignored)")
}

// SetMode sets the given mode.
func (v *Terminal) SetMode(mode ansicode.TerminalMode) {
	dbg.Println("SetMode", mode)
	var forward bool
	switch mode {
	case ansicode.TerminalModeCursorKeys:
		forward = true
	case ansicode.TerminalModeLineWrap:
	case ansicode.TerminalModeBlinkingCursor:
		epoch := time.Now()
		v.CursorBlinkEpoch = &epoch
	case ansicode.TerminalModeShowCursor:
		v.CursorVisible = true
	case ansicode.TerminalModeReportMouseClicks, // basic
		ansicode.TerminalModeReportCellMouseMotion, // drag
		ansicode.TerminalModeReportAllMouseMotion,  // all mouse controls
		ansicode.TerminalModeSGRMouse:              // extended mouse coords
		forward = true
	case ansicode.TerminalModeReportFocusInOut: // window focus
		dbg.Println("SET WINDOW FOCUS TRACKING MODE", mode)
		forward = true
	case ansicode.TerminalModeSwapScreenAndSetRestoreCursor:
		dbg.Println("SET ALT SCREEN")
		if v.IsAlt {
			dbg.Println("ALREADY ALT")
		} else {
			dbg.Println("SWITCHING TO ALT")
			if v.Alt == nil {
				dbg.Println("ALLOCATING ALT SCREEN")
				v.Alt = newScreen(v.Height, v.Width)
			}
			v.swapAlt()
		}
	case ansicode.TerminalModeBracketedPaste:
		dbg.Println("SET BRACKETED PASTE")
		forward = true
	default:
		dbg.Println("SET UNKNOWN MODE", mode)
	}
	if forward && v.ForwardRequests != nil {
		fmt.Fprintf(v.ForwardRequests, "\x1b[%dh", mode)
	}
}

// SetModifyOtherKeys sets the modify other keys mode. (XTERM)
func (v *Terminal) SetModifyOtherKeys(modify ansicode.ModifyOtherKeys) {
	dbg.Println("SetModifyOtherKeys", modify)
	if v.ForwardRequests != nil {
		fmt.Fprintf(v.ForwardRequests, "%s%dm", termenv.CSI, modify)
	}
}

// SetScrollingRegion sets the scrolling region.
func (v *Terminal) SetScrollingRegion(top int, bottom int) {
	if v.AppendOnly {
		dbg.Printf("SetScrollingRegion: top=%d, bottom=%d (ignored)\n", top, bottom)
		return
	}

	if bottom < top {
		dbg.Printf("SetScrollingRegion: top=%d, bottom=%d (insane)\n", top, bottom)
		// sanity check
		return
	}

	dbg.Printf("SetScrollingRegion: top=%d, bottom=%d\n", top, bottom)

	if top == 1 && bottom == v.Height ||
		top == 1 && bottom == 1 {
		// equivalent to just resetting
		v.ScrollRegion = nil
	} else {
		v.ScrollRegion = &ScrollRegion{
			Start: top - 1,
			End:   bottom - 1,
		}
	}
	// Reset cursor position and wrap state
	// TODO: respect origin mode
	v.home(0, 0)
}

// SetTerminalCharAttribute sets the terminal char attribute.
func (v *Terminal) SetTerminalCharAttribute(attr ansicode.TerminalCharAttribute) {
	dbg.Println("SetTerminalCharAttribute", attr)
	switch attr.Attr {
	case ansicode.CharAttributeReset:
		dbg.Println("RESET CHAR ATTRIBUTES")
		v.Cursor.F = Reset
	case ansicode.CharAttributeBold:
		v.Cursor.F.SetBold(true)
	case ansicode.CharAttributeDim:
		v.Cursor.F.SetFaint(true)
	case ansicode.CharAttributeItalic:
		v.Cursor.F.SetItalic(true)
	case ansicode.CharAttributeUnderline:
		dbg.Println("UNDERLINING")
		v.Cursor.F.SetUnderline(true)
	case ansicode.CharAttributeDoubleUnderline:
		dbg.Println("TODO: CharAttributeDoubleUnderline")
		v.Cursor.F.SetUnderline(true)
	case ansicode.CharAttributeCurlyUnderline:
		dbg.Println("TODO: CharAttributeCurlyUnderline")
		v.Cursor.F.SetUnderline(true)
	case ansicode.CharAttributeDottedUnderline:
		dbg.Println("TODO: CharAttributeDottedUnderline")
		v.Cursor.F.SetUnderline(true)
	case ansicode.CharAttributeDashedUnderline:
		dbg.Println("TODO: CharAttributeDashedUnderline")
		v.Cursor.F.SetUnderline(true)
	case ansicode.CharAttributeBlinkSlow:
		dbg.Println("TODO: CharAttributeBlinkSlow")
		v.Cursor.F.SetBlink(false)
	case ansicode.CharAttributeBlinkFast:
		dbg.Println("TODO: CharAttributeBlinkFast")
		v.Cursor.F.SetBlink(false)
	case ansicode.CharAttributeReverse:
		v.Cursor.F.SetReverse(true)
	case ansicode.CharAttributeHidden:
		v.Cursor.F.SetConceal(true)
	case ansicode.CharAttributeStrike:
		dbg.Println("TODO: CharAttributeStrike")
	case ansicode.CharAttributeCancelBold:
		v.Cursor.F.SetBold(false)
	case ansicode.CharAttributeCancelBoldDim:
		v.Cursor.F.SetBold(false)
		v.Cursor.F.SetFaint(false)
	case ansicode.CharAttributeCancelItalic:
		v.Cursor.F.SetItalic(false)
	case ansicode.CharAttributeCancelUnderline:
		v.Cursor.F.SetUnderline(false)
	case ansicode.CharAttributeCancelBlink:
		v.Cursor.F.SetBlink(false)
	case ansicode.CharAttributeCancelReverse:
		v.Cursor.F.SetReverse(false)
	case ansicode.CharAttributeCancelHidden:
		v.Cursor.F.SetConceal(false)
	case ansicode.CharAttributeCancelStrike:
		dbg.Println("TODO: CharAttributeCancelStrike")
	case ansicode.CharAttributeForeground:
		v.Cursor.F.Fg = attrColor(attr)
	case ansicode.CharAttributeBackground:
		v.Cursor.F.Bg = attrColor(attr)
	case ansicode.CharAttributeUnderlineColor:
		dbg.Println("TODO: CharAttributeUnderlineColor")
	default:
		dbg.Println("UNKNOWN CHAR ATTRIBUTE:", attr.Attr)
	}
}

// RGBColor Lossless alternative to what termenv uses since its floats have rounding issues
type RGBColor struct {
	R, G, B uint8
}

func (f RGBColor) Sequence(bg bool) string {
	prefix := termenv.Foreground
	if bg {
		prefix = termenv.Background
	}
	return fmt.Sprintf("%s;2;%d;%d;%d", prefix, f.R, f.G, f.B)
}

var _ termenv.Color = RGBColor{}

func attrColor(attr ansicode.TerminalCharAttribute) termenv.Color {
	switch {
	case attr.NamedColor != nil:
		dbg.Printf("NAMED COLOR: %d\n", *attr.NamedColor)
		num := *attr.NamedColor
		switch {
		case num >= 0 && num <= 15:
			return termenv.ANSIColor(num)
		case num <= 255:
			return termenv.ANSI256Color(num)
		case num == ansicode.NamedColorForeground:
			// might make sense to actually track the foreground/background color,
			// but for now we just represent it as unset (nil), leaving it to the
			// frontend to style however it wants (i.e. with CSS)
			return nil
		case num == ansicode.NamedColorBackground:
			// see above
			return nil
		case num == ansicode.NamedColorCursor:
			dbg.Println("TODO: NamedColorCursor")
			return nil
		default:
			dbg.Println("UNKNOWN NAMED COLOR:", num)
		}
	case attr.IndexedColor != nil:
		dbg.Println("INDEXED COLOR:", attr.IndexedColor.Index)
		return termenv.ANSI256Color(attr.IndexedColor.Index)
	case attr.RGBColor != nil:
		dbg.Println("RGB COLOR:", attr.RGBColor)
		return RGBColor{
			R: attr.RGBColor.R,
			G: attr.RGBColor.G,
			B: attr.RGBColor.B,
		}
	default:
		dbg.Println("UNKNOWN FOREGROUND COLOR:", attr)
	}
	// noisy fallback
	return termenv.ANSIBrightMagenta
}

// SetTitle sets the window title.
func (v *Terminal) SetTitle(title string) {
	dbg.Printf("SetTitle: title=%s\n", title)
	v.Title = title
}

// Substitute replaces the character under the cursor.
func (v *Terminal) Substitute() {
	dbg.Println("TODO: Substitute")
}

const tabWidth = 8

// Tab moves the cursor to the next tab stop.
func (v *Terminal) Tab(n int) {
	target := ((v.Cursor.X / tabWidth) + 1) * tabWidth
	if !v.AutoResizeX && target >= v.Width {
		target = v.Width - 1
	}
	format := v.Cursor.F
	for x := v.Cursor.X; x < target; x++ {
		if v.AutoResizeX {
			v.put(' ')
		} else {
			v.clear(v.Cursor.Y, x, format)
		}
	}
	v.Cursor.X = target
}

// TextAreaSizeChars reports the text area size in characters.
func (v *Terminal) TextAreaSizeChars() {
	dbg.Println("TODO: TextAreaSizeChars")
}

// TextAreaSizePixels reports the text area size in pixels.
func (v *Terminal) TextAreaSizePixels() {
	dbg.Println("TODO: TextAreaSizePixels")
}

// UnsetKeypadApplicationMode sets the keypad to numeric mode.
func (v *Terminal) UnsetKeypadApplicationMode() {
	dbg.Println("UnsetKeypadApplicationMode (ignored)")
}

// UnsetMode unsets the given mode.
func (v *Terminal) UnsetMode(mode ansicode.TerminalMode) {
	dbg.Println("UnsetMode", mode)
	var forward bool
	switch mode {
	case ansicode.TerminalModeCursorKeys:
		forward = true
	case ansicode.TerminalModeLineWrap:
	case ansicode.TerminalModeBlinkingCursor:
		v.CursorBlinkEpoch = nil
	case ansicode.TerminalModeShowCursor:
		v.CursorVisible = false
	case ansicode.TerminalModeReportMouseClicks, // basic
		ansicode.TerminalModeReportCellMouseMotion, // drag
		ansicode.TerminalModeReportAllMouseMotion,  // all mouse controls
		ansicode.TerminalModeSGRMouse:              // extended mouse coords
		forward = true
	case ansicode.TerminalModeReportFocusInOut: // window focus
		dbg.Println("UNSET WINDOW FOCUS TRACKING MODE", mode)
		forward = true
	case ansicode.TerminalModeSwapScreenAndSetRestoreCursor:
		dbg.Println("UNSET ALT SCREEN")
		if !v.IsAlt {
			dbg.Println("ALREADY NOT ALT")
		} else {
			dbg.Println("RESTORING MAIN SCREEN")
			v.swapAlt()
		}
	case ansicode.TerminalModeBracketedPaste:
		dbg.Println("UNSET BRACKETED PASTE")
		forward = true
	default:
		dbg.Println("UNSET UNKNOWN MODE", mode)
	}
	if forward && v.ForwardRequests != nil {
		fmt.Fprintf(v.ForwardRequests, "\x1b[?%dl", mode)
	}
}
