package stamp

import (
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

func TestRenderResourceGateExpr(t *testing.T) {
	r := interchange.Resource{
		File:     "templates/web/pdb.yaml",
		GateExpr: `.Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget"`,
		Gate:     "web.enabled", // should be ignored in favor of GateExpr
		Manifest: map[string]interface{}{"apiVersion": "policy/v1", "kind": "PodDisruptionBudget"},
	}
	out, err := renderResource(r)
	if err != nil {
		t.Fatal(err)
	}
	want := `{{- if .Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget" }}`
	if !strings.HasPrefix(out, want+"\n") {
		t.Errorf("gate expr not used verbatim:\n%s", out)
	}
	if strings.Contains(out, ".Values..Capabilities") {
		t.Errorf(".Values. was wrongly prefixed to a raw gate expr:\n%s", out)
	}
	if !strings.HasSuffix(out, "{{- end }}\n") {
		t.Errorf("missing gate end:\n%s", out)
	}
}

func TestRenderResourceGatePathStillWorks(t *testing.T) {
	r := interchange.Resource{
		File:     "templates/web/x.yaml",
		Gate:     "web.enabled",
		Manifest: map[string]interface{}{"kind": "ConfigMap"},
	}
	out, err := renderResource(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "{{- if .Values.web.enabled }}\n") {
		t.Errorf("simple values-path gate regressed:\n%s", out)
	}
}
