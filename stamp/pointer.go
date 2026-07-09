package stamp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// setAtPointer replaces the value at an RFC 6901 JSON Pointer within a nested
// map[string]interface{} / []interface{} tree. The path must already exist.
func setAtPointer(root interface{}, pointer string, val interface{}) error {
	if !strings.HasPrefix(pointer, "/") {
		return fmt.Errorf("pointer %q must start with /", pointer)
	}
	tokens := strings.Split(pointer[1:], "/")
	cur := root
	for i, tok := range tokens {
		tok = strings.ReplaceAll(strings.ReplaceAll(tok, "~1", "/"), "~0", "~")
		last := i == len(tokens)-1
		switch node := cur.(type) {
		case map[string]interface{}:
			if last {
				if _, ok := node[tok]; !ok {
					return fmt.Errorf("pointer %q: key %q not found", pointer, tok)
				}
				node[tok] = val
				return nil
			}
			next, ok := node[tok]
			if !ok {
				return fmt.Errorf("pointer %q: key %q not found", pointer, tok)
			}
			cur = next
		case []interface{}:
			idx, err := strconv.Atoi(tok)
			if err != nil || idx < 0 || idx >= len(node) {
				return fmt.Errorf("pointer %q: bad index %q", pointer, tok)
			}
			if last {
				node[idx] = val
				return nil
			}
			cur = node[idx]
		default:
			return fmt.Errorf("pointer %q: cannot descend into %T at %q", pointer, cur, tok)
		}
	}
	return nil
}

// deepCopy clones a JSON-shaped value via round-trip marshaling.
func deepCopy(v map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("deepCopy marshal: %w", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("deepCopy unmarshal: %w", err)
	}
	return out, nil
}
