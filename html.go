package midterm

import (
	"bytes"
	"html"
)

// HTML renders v as an HTML fragment. One idea for how to use this is to debug
// the current state of the screen reader.
func (v *Terminal) HTML() string {
	v.mut.Lock()
	defer v.mut.Unlock()

	var buf bytes.Buffer
	buf.WriteString(`<pre style="color:white;background-color:black;">`)

	for y := 0; y < v.Format.Height(); y++ {
		var x int
		for region := range v.Format.Regions(y) {
			buf.WriteString(`<span style="` + region.F.css() + `">`)
			buf.WriteString(html.EscapeString(string(v.Content[y][x : x+region.Size])))
			buf.WriteString("</span>")
			x += region.Size
		}
		buf.WriteRune('\n')
	}
	buf.WriteString("</pre>")

	return buf.String()
}
