package stamp

import (
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

// block-string embeds a structured value as a YAML literal block string, e.g. a
// whole config object under a ConfigMap data key.
func TestSubstituteBlockString(t *testing.T) {
	yamlText := "data:\n  tempo.yaml: CWTOK\n"
	holes := []interchange.Hole{{
		Path:    "web.configs.config",
		Render:  "block-string",
		Default: map[string]interface{}{"a": "b"},
	}}
	tokens := map[int]string{0: "CWTOK"}
	out, err := substitute(yamlText, holes, tokens)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"  tempo.yaml: |",
		"    {{- toYaml .Values.web.configs.config | nindent 4 }}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}
