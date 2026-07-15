package stamp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zalegrala/chartwright/interchange"
)

// substitute replaces each hole's sentinel token in the marshaled YAML text
// with the Helm expression for that hole's render mode.
func substitute(yamlText string, holes []interchange.Hole, tokens map[int]string) (string, error) {
	lines := strings.Split(yamlText, "\n")
	for i, h := range holes {
		tok := tokens[i]
		switch renderMode(h) {
		case "scalar", "quote":
			if err := replaceTokenInline(lines, tok, inlineExpr(h)); err != nil {
				return "", fmt.Errorf("hole %q: %w", h.Path, err)
			}
		case "raw":
			if err := replaceTokenInline(lines, tok, h.Raw); err != nil {
				return "", fmt.Errorf("hole %q: %w", h.Path, err)
			}
		case "block", "with", "block-string":
			var err error
			lines, err = replaceTokenBlock(lines, tok, h)
			if err != nil {
				return "", err
			}
		default:
			return "", fmt.Errorf("hole %q: unknown render mode %q", h.Path, renderMode(h))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func replaceTokenInline(lines []string, tok, expr string) error {
	for i, line := range lines {
		if strings.Contains(line, tok) {
			lines[i] = strings.Replace(line, tok, expr, 1)
			return nil
		}
	}
	return fmt.Errorf("sentinel %q not found in rendered YAML", tok)
}

// replaceTokenBlock replaces a whole "key: TOKEN" line with a block-rendered
// hole (toYaml | nindent). Modes:
//   - "with":         guarded by {{- with }} so the key vanishes when empty
//   - "block-string": the value is a YAML literal block (`key: |`), i.e. the
//     structured value is embedded as a string (e.g. a config in a ConfigMap)
//   - "block":        the value is a nested YAML mapping/sequence
func replaceTokenBlock(lines []string, tok string, h interchange.Hole) ([]string, error) {
	mode := renderMode(h)
	for i, line := range lines {
		pos := strings.Index(line, tok)
		if pos < 0 {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		pad := strings.Repeat(" ", indent)
		cpad := strings.Repeat(" ", indent+2)
		childIndent := strconv.Itoa(indent + 2)
		// Preserve the key prefix verbatim (everything before the sentinel value),
		// so keys containing colons or requiring quotes survive untouched.
		keyLine := strings.TrimRight(line[:pos], " ")
		if mode == "block-string" {
			keyLine += " |" // YAML literal block scalar: value embedded as a string
		}
		path := helmPath(h.Path)

		var repl []string
		if mode == "with" {
			repl = []string{
				pad + "{{- with " + path + " }}",
				keyLine,
				cpad + "{{- toYaml . | nindent " + childIndent + " }}",
				pad + "{{- end }}",
			}
		} else {
			repl = []string{
				keyLine,
				cpad + "{{- toYaml " + path + " | nindent " + childIndent + " }}",
			}
		}
		out := make([]string, 0, len(lines)+len(repl))
		out = append(out, lines[:i]...)
		out = append(out, repl...)
		out = append(out, lines[i+1:]...)
		return out, nil
	}
	return nil, fmt.Errorf("hole %q: sentinel %q not found in rendered YAML", h.Path, tok)
}
