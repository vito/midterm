package midterm

import "bytes"

// HTML renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader.
func (v *Terminal) HTML() string {
	v.mut.Lock()
	defer v.mut.Unlock()

	var buf bytes.Buffer
	buf.WriteString(`<pre style="color:white;background-color:black;">`)

	// Iterate each row. When the css changes, close the previous span, and open
	// a new one. No need to close a span when the css is empty, we won't have
	// opened one in the past.
	var lastFormat Format
	for y, row := range v.Content {
		for x, r := range row {
			f := v.Format[y][x]
			if f != lastFormat {
				if lastFormat != (Format{}) {
					buf.WriteString("</span>")
				}
				if f != (Format{}) {
					buf.WriteString(`<span style="` + f.css() + `">`)
				}
				lastFormat = f
			}
			if s := maybeEscapeRune(r); s != "" {
				buf.WriteString(s)
			} else {
				buf.WriteRune(r)
			}
		}
		buf.WriteRune('\n')
	}
	buf.WriteString("</pre>")

	return buf.String()
}

// maybeEscapeRune potentially escapes a rune for display in an html document.
// It only escapes the things that html.EscapeString does, but it works without allocating
// a string to hold r. Returns an empty string if there is no need to escape.
func maybeEscapeRune(r rune) string {
	switch r {
	case '&':
		return "&amp;"
	case '\'':
		return "&#39;"
	case '<':
		return "&lt;"
	case '>':
		return "&gt;"
	case '"':
		return "&quot;"
	}
	return ""
}
