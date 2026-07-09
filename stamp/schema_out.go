package stamp

import (
	"encoding/json"
	"strings"

	"github.com/zalegrala/helmitis/interchange"
)

// buildValuesSchema produces a values.schema.json: an object schema whose
// nested "properties" mirror the dotted hole paths, with each leaf carrying the
// hole's provided JSON Schema (or an empty schema if none).
func buildValuesSchema(doc interchange.Document) ([]byte, error) {
	root := map[string]interface{}{
		"$schema":    "https://json-schema.org/draft/2020-12/schema",
		"type":       "object",
		"properties": map[string]interface{}{},
	}
	for _, r := range doc.Resources {
		for _, h := range r.Holes {
			var leaf interface{}
			if len(h.Schema) > 0 {
				if err := json.Unmarshal(h.Schema, &leaf); err != nil {
					return nil, err
				}
			} else {
				leaf = map[string]interface{}{}
			}
			insertSchemaLeaf(root["properties"].(map[string]interface{}), strings.Split(h.Path, "."), leaf)
		}
	}
	return json.MarshalIndent(root, "", "  ")
}

func insertSchemaLeaf(props map[string]interface{}, parts []string, leaf interface{}) {
	head := parts[0]
	if len(parts) == 1 {
		props[head] = leaf
		return
	}
	node, ok := props[head].(map[string]interface{})
	if !ok {
		node = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		props[head] = node
	}
	childProps, ok := node["properties"].(map[string]interface{})
	if !ok {
		childProps = map[string]interface{}{}
		node["properties"] = childProps
	}
	insertSchemaLeaf(childProps, parts[1:], leaf)
}
