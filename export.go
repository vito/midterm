package vt100

import (
	"html/template"
	"net/http"
)

type VT100Handler struct {
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

func (v VT100Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.mu.Lock() // TODO(jaguilar): bad software engineering.
	termTemplate.Execute(w, struct {
		Height, Width int
		ConsoleHTML   template.HTML
	}{v.Height, v.Width, template.HTML(v.HTML())})
	v.mu.Unlock()
}
