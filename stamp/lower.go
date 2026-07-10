package stamp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zalegrala/chartwright/interchange"
)

// holeMarkerKey is the reserved single key that marks a manifest node as a hole.
const holeMarkerKey = "__cw_hole__"

// escapePointerToken escapes a token per RFC 6901 (~ -> ~0, / -> ~1).
func escapePointerToken(s string) string {
	s = strings.ReplaceAll(s, "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	return s
}

// asMarker returns the marker payload if v is a hole-marker object, else nil.
func asMarker(v interface{}) map[string]interface{} {
	m, ok := v.(map[string]interface{})
	if !ok || len(m) != 1 {
		return nil
	}
	payload, ok := m[holeMarkerKey].(map[string]interface{})
	if !ok {
		return nil
	}
	return payload
}

// markerToHole builds a Hole from a marker payload at the given pointer, and
// returns the placeholder value to leave in the clean manifest (the default).
func markerToHole(payload map[string]interface{}, pointer string) (interchange.Hole, interface{}, error) {
	h := interchange.Hole{Pointer: pointer}
	path, ok := payload["path"].(string)
	if !ok || path == "" {
		return h, nil, fmt.Errorf("hole at %s: missing string \"path\"", pointer)
	}
	h.Path = path
	if r, ok := payload["render"].(string); ok {
		h.Render = r
	}
	if raw, ok := payload["raw"].(string); ok {
		h.Raw = raw
	}
	if req, ok := payload["required"].(bool); ok {
		h.Required = req
	}
	if def, ok := payload["default"]; ok {
		h.Default = def
	}
	if sch, ok := payload["schema"]; ok {
		b, err := json.Marshal(sch)
		if err != nil {
			return h, nil, fmt.Errorf("hole %s: schema: %w", path, err)
		}
		h.Schema = b
	}
	return h, h.Default, nil
}

// lowerManifest walks a manifest, extracting every hole marker into a Hole with
// its RFC 6901 pointer and replacing the marker node with a placeholder value.
// Holes are returned in sorted-pointer order for determinism.
func lowerManifest(m map[string]interface{}) (map[string]interface{}, []interchange.Hole, error) {
	var holes []interchange.Hole
	cleaned, err := walk(m, "", &holes)
	if err != nil {
		return nil, nil, err
	}
	clean, _ := cleaned.(map[string]interface{})
	sort.SliceStable(holes, func(i, j int) bool { return holes[i].Pointer < holes[j].Pointer })
	return clean, holes, nil
}

// Lower applies lowerManifest to every resource: it replaces each resource's
// manifest with the cleaned form and appends the extracted holes to any holes
// the resource already declared. Resources with no markers are unchanged.
func Lower(doc interchange.Document) (interchange.Document, error) {
	for i := range doc.Resources {
		clean, holes, err := lowerManifest(doc.Resources[i].Manifest)
		if err != nil {
			return interchange.Document{}, fmt.Errorf("resource %s: %w", doc.Resources[i].File, err)
		}
		doc.Resources[i].Manifest = clean
		doc.Resources[i].Holes = append(doc.Resources[i].Holes, holes...)
	}
	return doc, nil
}

// walk recursively cleans a value, collecting holes into *out.
func walk(v interface{}, pointer string, out *[]interchange.Hole) (interface{}, error) {
	if payload := asMarker(v); payload != nil {
		h, placeholder, err := markerToHole(payload, pointer)
		if err != nil {
			return nil, err
		}
		*out = append(*out, h)
		return placeholder, nil
	}
	switch node := v.(type) {
	case map[string]interface{}:
		clean := make(map[string]interface{}, len(node))
		keys := make([]string, 0, len(node))
		for k := range node {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child, err := walk(node[k], pointer+"/"+escapePointerToken(k), out)
			if err != nil {
				return nil, err
			}
			clean[k] = child
		}
		return clean, nil
	case []interface{}:
		clean := make([]interface{}, len(node))
		for i, item := range node {
			child, err := walk(item, pointer+"/"+strconv.Itoa(i), out)
			if err != nil {
				return nil, err
			}
			clean[i] = child
		}
		return clean, nil
	default:
		return v, nil
	}
}
