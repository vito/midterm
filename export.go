package vt100

import (
	"html/template"
	"net/http"
)

// Handler is a type that knows how to serve the VT100 as an HTML
// page. This is useful as a way to debug problems, or display
// the state of the terminal to a user.
//
// TODO(jaguilar): move me to a subpackage, then export to the
// default mux automatically. This seems to be the way it's done
// in the std lib.
type Handler struct {
	*VT100
}

var termTemplate = template.Must(template.New("vt100_html").Parse(`
	<html><head><style>
	.mono {
		font-family: "monospace";
	}</style>
	<body>
	<p>VT100 Terminal
	<p>Dimensions {{.Height}}x{{.Width}}
	<p><span class="mono">{{.ConsoleHTML}}</span>
	</span>
	</html>
`))

// ServeHTTP is part of the http.Handler interface.
func (v Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock() // TODO(jaguilar): bad software engineering.
	termTemplate.Execute(w, struct {
		Height, Width int
		ConsoleHTML   template.HTML
	}{v.Height, v.Width, template.HTML(v.HTML())})
	v.mu.Unlock()
}
