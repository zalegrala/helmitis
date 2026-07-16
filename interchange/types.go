package interchange

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type Document struct {
	Chart      Chart                `json:"chart"`
	Components map[string]Component `json:"components"`
	Resources  []Resource           `json:"resources"`
}

type Chart struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion,omitempty"`
	Description string `json:"description,omitempty"`
	KubeVersion string `json:"kubeVersion,omitempty"`
}

type Component struct {
	Enabled  bool   `json:"enabled"`
	Workload string `json:"workload,omitempty"`
}

type Resource struct {
	File      string                 `json:"file"`
	Component string                 `json:"component,omitempty"`
	GVK       string                 `json:"gvk,omitempty"`
	Gate      string                 `json:"gate,omitempty"`
	// GateExpr is a verbatim Helm boolean expression wrapping the whole resource
	// in {{- if <expr> }}. Unlike Gate (a values path, prefixed with .Values.),
	// GateExpr is used as-is — for capability/version gates like
	// `.Capabilities.APIVersions.Has "policy/v1/PodDisruptionBudget"`. Takes
	// precedence over Gate when set.
	GateExpr string                 `json:"gateExpr,omitempty"`
	Manifest map[string]interface{} `json:"manifest"`
	Holes    []Hole                 `json:"holes,omitempty"`
}

// Hole marks a variable point in a manifest.
// Render is one of "scalar", "block", "quote", "with", "raw"; empty means infer
// from Default's type (object/array -> block, otherwise scalar).
type Hole struct {
	Path     string          `json:"path"`
	Pointer  string          `json:"pointer"`
	Default  interface{}     `json:"default,omitempty"`
	Schema   json.RawMessage `json:"schema,omitempty"`
	Render   string          `json:"render,omitempty"`
	Raw      string          `json:"raw,omitempty"`
	Required bool            `json:"required,omitempty"`
}

func Decode(data []byte) (Document, error) {
	var doc Document
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&doc); err != nil {
		return Document{}, fmt.Errorf("decode interchange: %w", err)
	}
	return doc, nil
}
