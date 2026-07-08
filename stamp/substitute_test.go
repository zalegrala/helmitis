package stamp

import (
	"strings"
	"testing"

	"github.com/zalegrala/helmitis/interchange"
)

func TestSubstituteScalar(t *testing.T) {
	yamlText := "spec:\n  replicas: HOLESENTINEL0END\n"
	holes := []interchange.Hole{{Path: "web.replicas", Default: float64(3)}}
	tokens := map[int]string{0: "HOLESENTINEL0END"}
	out, err := substitute(yamlText, holes, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "replicas: {{ .Values.web.replicas | default 3 }}") {
		t.Errorf("got:\n%s", out)
	}
}

func TestSubstituteRaw(t *testing.T) {
	yamlText := "image: HOLESENTINEL0END\n"
	holes := []interchange.Hole{{Path: "x", Render: "raw",
		Raw: `{{ .Values.repo }}:{{ .Values.tag }}`}}
	tokens := map[int]string{0: "HOLESENTINEL0END"}
	out, _ := substitute(yamlText, holes, tokens)
	if !strings.Contains(out, "image: {{ .Values.repo }}:{{ .Values.tag }}") {
		t.Errorf("got:\n%s", out)
	}
}

func TestSubstituteBlock(t *testing.T) {
	yamlText := "spec:\n  resources: HOLESENTINEL0END\n"
	holes := []interchange.Hole{{Path: "web.resources",
		Default: map[string]interface{}{"limits": map[string]interface{}{"cpu": "1"}}}}
	tokens := map[int]string{0: "HOLESENTINEL0END"}
	out, err := substitute(yamlText, holes, tokens)
	if err != nil {
		t.Fatal(err)
	}
	want := "  resources:\n    {{- toYaml .Values.web.resources | nindent 4 }}"
	if !strings.Contains(out, want) {
		t.Errorf("got:\n%s\nwant substring:\n%s", out, want)
	}
}

func TestSubstituteWith(t *testing.T) {
	yamlText := "spec:\n  nodeSelector: HOLESENTINEL0END\n"
	holes := []interchange.Hole{{Path: "web.nodeSelector", Render: "with",
		Default: map[string]interface{}{}}}
	tokens := map[int]string{0: "HOLESENTINEL0END"}
	out, _ := substitute(yamlText, holes, tokens)
	for _, want := range []string{
		"  {{- with .Values.web.nodeSelector }}",
		"  nodeSelector:",
		"    {{- toYaml . | nindent 4 }}",
		"  {{- end }}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
