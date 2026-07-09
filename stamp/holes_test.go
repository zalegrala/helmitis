package stamp

import (
	"testing"

	"github.com/zalegrala/helmitis/interchange"
)

func TestRenderModeInference(t *testing.T) {
	cases := []struct {
		def  interface{}
		want string
	}{
		{float64(3), "scalar"},
		{"hello", "scalar"},
		{true, "scalar"},
		{map[string]interface{}{"a": 1}, "block"},
		{[]interface{}{1, 2}, "block"},
	}
	for _, c := range cases {
		h := interchange.Hole{Default: c.def}
		if got := renderMode(h); got != c.want {
			t.Errorf("renderMode(%T) = %q, want %q", c.def, got, c.want)
		}
	}
}

func TestRenderModeExplicit(t *testing.T) {
	h := interchange.Hole{Default: float64(3), Render: "quote"}
	if renderMode(h) != "quote" {
		t.Errorf("explicit render should win")
	}
}

func TestHelmLiteral(t *testing.T) {
	cases := []struct {
		v    interface{}
		want string
	}{
		{float64(3), "3"},
		{float64(3.5), "3.5"},
		{true, "true"},
		{"hi", `"hi"`},
	}
	for _, c := range cases {
		if got := helmLiteral(c.v); got != c.want {
			t.Errorf("helmLiteral(%v) = %q, want %q", c.v, got, c.want)
		}
	}
}

func TestInlineExpr(t *testing.T) {
	cases := []struct {
		h    interchange.Hole
		want string
	}{
		{interchange.Hole{Path: "web.replicas", Default: float64(3)},
			"{{ .Values.web.replicas | default 3 }}"},
		{interchange.Hole{Path: "img.tag", Default: "latest", Render: "quote"},
			`{{ .Values.img.tag | default "latest" | quote }}`},
		{interchange.Hole{Path: "web.name", Required: true},
			`{{ required "web.name is required" .Values.web.name }}`},
		{interchange.Hole{Path: "web.x"},
			"{{ .Values.web.x }}"},
	}
	for _, c := range cases {
		if got := inlineExpr(c.h); got != c.want {
			t.Errorf("inlineExpr(%+v) = %q, want %q", c.h, got, c.want)
		}
	}
}
