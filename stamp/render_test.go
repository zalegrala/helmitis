package stamp

import (
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

func TestRenderResource(t *testing.T) {
	r := interchange.Resource{
		File: "templates/web/deployment.yaml",
		Gate: "web.enabled",
		Manifest: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"spec":       map[string]interface{}{"replicas": float64(0)},
		},
		Holes: []interchange.Hole{
			{Path: "web.replicas", Pointer: "/spec/replicas", Default: float64(3)},
		},
	}
	out, err := renderResource(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "{{- if .Values.web.enabled }}\n") {
		t.Errorf("missing gate prefix:\n%s", out)
	}
	if !strings.Contains(out, "replicas: {{ .Values.web.replicas | default 3 }}") {
		t.Errorf("hole not substituted:\n%s", out)
	}
	if !strings.HasSuffix(out, "{{- end }}\n") {
		t.Errorf("missing gate suffix:\n%s", out)
	}
	// Original manifest must be untouched (deepCopy isolation).
	if r.Manifest["spec"].(map[string]interface{})["replicas"] != float64(0) {
		t.Error("renderResource mutated the input manifest")
	}
}
