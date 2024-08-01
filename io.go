package grouter

import (
	"bytes"
	"fmt"
)

type TemplateValues map[string]any

type BufferedResponse struct {
	buf          bytes.Buffer
	templateName string
	values       TemplateValues
	end          bool
}

func (r *BufferedResponse) RenderTemplate(name string, values TemplateValues, end bool) {
	r.templateName = name
	r.values = values
	r.end = end
	r.buf.Reset()
}

func (r *BufferedResponse) RenderContinueTemplate(name string, values TemplateValues) {
	r.RenderTemplate(name, values, false)
}

func (r *BufferedResponse) RenderEndTemplate(name string, values TemplateValues) {
	r.RenderTemplate(name, values, false)
}

func (r *BufferedResponse) Write(p []byte) (int, error) {
	r.templateName = ""
	return r.buf.Write(p)
}

func (r *BufferedResponse) Printf(format string, args ...any) {
	_, _ = fmt.Fprintf(&r.buf, format, args...)
}
