package stamp

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/zalegrala/chartwright/interchange"
	"sigs.k8s.io/yaml"
)

// renderResource produces the final template-file content for one resource:
// sentinel insertion -> deterministic YAML marshal -> hole substitution -> gate.
func renderResource(r interchange.Resource) (string, error) {
	m, err := deepCopy(r.Manifest)
	if err != nil {
		return "", fmt.Errorf("resource %s: %w", r.File, err)
	}
	// Per-resource nonce derived from the file path makes the sentinel
	// effectively impossible to collide with real manifest content, while
	// staying deterministic (same input -> same output). Hex + letters only,
	// so it survives YAML marshaling unquoted.
	nonce := fmt.Sprintf("%x", sha256.Sum256([]byte(r.File)))[:16]
	tokens := make(map[int]string, len(r.Holes))
	for i, h := range r.Holes {
		tok := fmt.Sprintf("CWHOLE%s%dEND", nonce, i)
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
	// GateExpr (verbatim) wins over Gate (a values path). Either wraps the whole
	// resource in {{- if <expr> }} ... {{- end }}.
	gateExpr := ""
	if r.GateExpr != "" {
		gateExpr = r.GateExpr
	} else if r.Gate != "" {
		gateExpr = ".Values." + r.Gate
	}
	if gateExpr != "" {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text = "{{- if " + gateExpr + " }}\n" + text + "{{- end }}\n"
	}
	return text, nil
}
