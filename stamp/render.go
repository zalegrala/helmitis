package stamp

import (
	"fmt"
	"strings"

	"github.com/zalegrala/helmitis/interchange"
	"sigs.k8s.io/yaml"
)

// renderResource produces the final template-file content for one resource:
// sentinel insertion -> deterministic YAML marshal -> hole substitution -> gate.
func renderResource(r interchange.Resource) (string, error) {
	m := deepCopy(r.Manifest)
	tokens := make(map[int]string, len(r.Holes))
	for i, h := range r.Holes {
		tok := fmt.Sprintf("HOLESENTINEL%dEND", i)
		tokens[i] = tok
		if err := setAtPointer(m, h.Pointer, tok); err != nil {
			return "", fmt.Errorf("resource %s: %w", r.File, err)
		}
	}
	y, err := yaml.Marshal(m) // sigs.k8s.io/yaml sorts keys via encoding/json
	if err != nil {
		return "", fmt.Errorf("resource %s: marshal: %w", r.File, err)
	}
	text, err := substitute(string(y), r.Holes, tokens)
	if err != nil {
		return "", err
	}
	if r.Gate != "" {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text = "{{- if .Values." + r.Gate + " }}\n" + text + "{{- end }}\n"
	}
	return text, nil
}
