package stamp

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zalegrala/chartwright/interchange"
	"sigs.k8s.io/yaml"
)

// setDotted writes val into root following a dotted path, creating intermediate
// maps as needed. Errors if a segment is already occupied by a non-map.
func setDotted(root map[string]interface{}, dotted string, val interface{}) error {
	parts := strings.Split(dotted, ".")
	cur := root
	for i, p := range parts {
		if i == len(parts)-1 {
			cur[p] = val
			return nil
		}
		switch next := cur[p].(type) {
		case nil:
			m := map[string]interface{}{}
			cur[p] = m
			cur = m
		case map[string]interface{}:
			cur = next
		default:
			return fmt.Errorf("path %q: segment %q occupied by %T", dotted, p, next)
		}
	}
	return nil
}

// marshalValues serializes a values map to YAML bytes.
func marshalValues(v map[string]interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

// buildValues folds component gates and hole defaults into a nested values map.
func buildValues(doc interchange.Document) (map[string]interface{}, error) {
	root := map[string]interface{}{}
	names := make([]string, 0, len(doc.Components))
	for name := range doc.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := doc.Components[name]
		if err := setDotted(root, name+".enabled", c.Enabled); err != nil {
			return nil, err
		}
	}
	for _, r := range doc.Resources {
		for _, h := range r.Holes {
			if h.Default == nil {
				continue
			}
			if err := setDotted(root, h.Path, h.Default); err != nil {
				return nil, err
			}
		}
	}
	return root, nil
}
