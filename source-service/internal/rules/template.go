package rules

import (
	"bytes"
	"encoding/json"
	"text/template"
)

// templateFuncs is available to every rule template.
//
// `json` is the important one: run_task templates produce JSON, and Event
// fields (message text, in particular) can contain quotes/newlines that
// would break the JSON if interpolated raw inside a quoted string — use
// {{.Text | json}} (which itself includes the surrounding quotes) instead
// of "{{.Text}}". send_text templates don't need it since they produce
// plain text, not JSON.
var templateFuncs = template.FuncMap{
	"json": func(v any) (string, error) {
		b, err := json.Marshal(v)
		return string(b), err
	},
}

// renderTemplate executes a Go text/template against ev, so actions can
// reference the triggering event, e.g. `{{.Sender}}` or `{{.Text | json}}`.
// See the package doc comment for why this is deliberately not a scripting
// language.
func renderTemplate(tmplText string, ev Event) ([]byte, error) {
	t, err := template.New("action").Funcs(templateFuncs).Parse(tmplText)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, ev); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
