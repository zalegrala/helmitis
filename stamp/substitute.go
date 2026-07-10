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
		case "block", "with":
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
// hole (toYaml | nindent), optionally guarded by {{- with }}.
func replaceTokenBlock(lines []string, tok string, h interchange.Hole) ([]string, error) {
	for i, line := range lines {
		if !strings.Contains(line, tok) {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		pad := strings.Repeat(" ", indent)
		cpad := strings.Repeat(" ", indent+2)
		childIndent := strconv.Itoa(indent + 2)
		key := strings.TrimSpace(strings.SplitN(strings.TrimSpace(line), ":", 2)[0])
		path := helmPath(h.Path)

		var repl []string
		if renderMode(h) == "with" {
			repl = []string{
				pad + "{{- with " + path + " }}",
				pad + key + ":",
				cpad + "{{- toYaml . | nindent " + childIndent + " }}",
				pad + "{{- end }}",
			}
		} else {
			repl = []string{
				pad + key + ":",
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
