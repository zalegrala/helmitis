package stamp

import (
	"fmt"
	"strconv"

	"github.com/zalegrala/helmitis/interchange"
)

func helmPath(dotted string) string { return ".Values." + dotted }

// renderMode returns the explicit Render, or infers from the default's type.
func renderMode(h interchange.Hole) string {
	if h.Render != "" {
		return h.Render
	}
	switch h.Default.(type) {
	case map[string]interface{}, []interface{}:
		return "block"
	default:
		return "scalar"
	}
}

// helmLiteral renders a Go/JSON value as a Helm template literal.
func helmLiteral(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strconv.Quote(t)
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

// inlineExpr builds the single-line {{ }} expression for scalar/quote holes.
func inlineExpr(h interchange.Hole) string {
	base := helmPath(h.Path)
	var expr string
	switch {
	case h.Required:
		expr = fmt.Sprintf("required %q %s", h.Path+" is required", base)
	case h.Default != nil:
		expr = fmt.Sprintf("%s | default %s", base, helmLiteral(h.Default))
	default:
		expr = base
	}
	if renderMode(h) == "quote" {
		expr += " | quote"
	}
	return "{{ " + expr + " }}"
}
