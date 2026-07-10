# Plan 2a — Hole-marker Lowering Pass

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development. Steps use `- [ ]`.

**Goal:** Let the stamper accept manifests with *inline hole markers* and lower them to the Level-1 interchange (clean manifest + `holes[]` with JSON Pointers) it already renders.

**Architecture:** A hole marker is a JSON object with exactly one key, `__cw_hole__`, whose value carries `{path, default?, schema?, render?, raw?, required?}`. A new `Lower` pass walks each resource's manifest, extracts every marker into a `holes[]` entry (computing its RFC 6901 pointer), and replaces the marker node with a placeholder value. `Lower` runs at the top of `Build`, so hand-written Level-1 docs (no markers) pass through unchanged while jsonnet-produced Level-0 docs get lowered transparently. This is the seam the Plan 2b jsonnet library targets.

**Tech Stack:** Go 1.23; existing `interchange` + `stamp` packages. Reuses `helm`/`kubeconform` for the acceptance gate.

**Context for the engineer:** Read `DESIGN.md` §9 (interchange) and `stamp/render.go` (how holes+pointers are consumed). The stamper today: `Build(doc)` → per resource `renderResource` deep-copies the manifest, inserts a sentinel at each hole's `Pointer`, marshals, substitutes. Lowering must produce pointers that address the *clean* manifest it also produces — consistent by construction. Determinism is a first-class property: walk map keys in sorted order so `holes[]` ordering is stable.

---

## Task 1: `lowerManifest` — walk a manifest, extract markers

**Files:** Create `stamp/lower.go`; Test `stamp/lower_test.go`.

- [ ] **Step 1: Write failing tests**

```go
package stamp

import (
	"reflect"
	"testing"
)

func TestLowerManifestScalar(t *testing.T) {
	m := map[string]interface{}{
		"apiVersion": "apps/v1",
		"spec": map[string]interface{}{
			"replicas": map[string]interface{}{
				"__cw_hole__": map[string]interface{}{
					"path":    "web.replicas",
					"default": float64(3),
					"schema":  map[string]interface{}{"type": "integer"},
				},
			},
		},
	}
	clean, holes, err := lowerManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	// marker replaced by its default in the clean manifest
	if clean["spec"].(map[string]interface{})["replicas"] != float64(3) {
		t.Errorf("placeholder = %v, want 3", clean["spec"].(map[string]interface{})["replicas"])
	}
	if len(holes) != 1 {
		t.Fatalf("holes = %d, want 1", len(holes))
	}
	if holes[0].Path != "web.replicas" || holes[0].Pointer != "/spec/replicas" {
		t.Errorf("hole = %+v", holes[0])
	}
	if holes[0].Default != float64(3) {
		t.Errorf("default = %v", holes[0].Default)
	}
	if len(holes[0].Schema) == 0 {
		t.Errorf("schema not carried through")
	}
}

func TestLowerManifestArrayAndNested(t *testing.T) {
	m := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name": "web",
					"image": map[string]interface{}{
						"__cw_hole__": map[string]interface{}{
							"path": "web.image", "default": "x:1", "render": "quote",
						},
					},
				},
			},
		},
	}
	clean, holes, err := lowerManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(holes) != 1 || holes[0].Pointer != "/spec/containers/0/image" {
		t.Fatalf("hole = %+v", holes)
	}
	if holes[0].Render != "quote" {
		t.Errorf("render = %q", holes[0].Render)
	}
	got := clean["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})["image"]
	if got != "x:1" {
		t.Errorf("placeholder = %v", got)
	}
}

func TestLowerManifestBlockDefault(t *testing.T) {
	m := map[string]interface{}{
		"spec": map[string]interface{}{
			"resources": map[string]interface{}{
				"__cw_hole__": map[string]interface{}{
					"path":    "web.resources",
					"default": map[string]interface{}{"limits": map[string]interface{}{"cpu": "1"}},
				},
			},
		},
	}
	clean, holes, err := lowerManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(holes) != 1 || holes[0].Pointer != "/spec/resources" {
		t.Fatalf("hole = %+v", holes)
	}
	// object default is preserved as the placeholder
	want := map[string]interface{}{"limits": map[string]interface{}{"cpu": "1"}}
	if !reflect.DeepEqual(clean["spec"].(map[string]interface{})["resources"], want) {
		t.Errorf("placeholder = %v", clean["spec"].(map[string]interface{})["resources"])
	}
}

func TestLowerManifestNoMarkers(t *testing.T) {
	m := map[string]interface{}{"kind": "Service", "spec": map[string]interface{}{"x": float64(1)}}
	clean, holes, err := lowerManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(holes) != 0 {
		t.Errorf("expected no holes, got %d", len(holes))
	}
	if !reflect.DeepEqual(clean, m) {
		t.Errorf("manifest changed: %v", clean)
	}
}

func TestLowerManifestDeterministicOrder(t *testing.T) {
	// two markers at sibling keys must come out in sorted-pointer order every time
	mk := func(path string) map[string]interface{} {
		return map[string]interface{}{"__cw_hole__": map[string]interface{}{"path": path, "default": float64(1)}}
	}
	m := map[string]interface{}{"spec": map[string]interface{}{"zeta": mk("z"), "alpha": mk("a")}}
	_, holes, err := lowerManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(holes) != 2 || holes[0].Pointer != "/spec/alpha" || holes[1].Pointer != "/spec/zeta" {
		t.Fatalf("holes not in sorted-pointer order: %+v", holes)
	}
}
```

- [ ] **Step 2: Run, verify fail** — `go test ./stamp/ -run TestLower` → FAIL `undefined: lowerManifest`.

- [ ] **Step 3: Implement** — Create `stamp/lower.go`:

```go
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

// markerToHole builds a Hole from a marker payload at the given pointer.
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
	// The placeholder left in the clean manifest is the default (or null).
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

// walk recursively cleans a value, collecting holes into *out.
func walk(v interface{}, pointer string, out *[]interchange.Hole) (interface{}, error) {
	if payload := asMarker(v); payload != nil {
		h, placeholder, err := markerToHole(payload, pointer)
		if err != nil {
			return nil, err
		}
		*out = append(*out, h)
		// a placeholder that is itself a marker is not possible; return as-is
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
```

- [ ] **Step 4: Run, verify pass** — `go test ./stamp/ -run TestLower` → PASS.

- [ ] **Step 5: Commit**

```bash
git add stamp/lower.go stamp/lower_test.go
git commit -m "feat(stamp): lowerManifest — extract inline hole markers to holes[]"
```

---

## Task 2: `Lower` document pass + wire into `Build`

**Files:** Modify `stamp/lower.go` (add `Lower`); Modify `stamp/stamp.go` (call it in `Build`); Test `stamp/lower_test.go`.

- [ ] **Step 1: Write failing test** — append to `stamp/lower_test.go`:

```go
import "github.com/zalegrala/chartwright/interchange" // ensure imported

func TestLowerDocumentAppendsHoles(t *testing.T) {
	doc := interchange.Document{
		Chart: interchange.Chart{Name: "d", Version: "0.1.0"},
		Resources: []interchange.Resource{{
			File: "templates/x.yaml",
			Manifest: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": map[string]interface{}{
						"__cw_hole__": map[string]interface{}{"path": "x.replicas", "default": float64(2)},
					},
				},
			},
		}},
	}
	out, err := Lower(doc)
	if err != nil {
		t.Fatal(err)
	}
	r := out.Resources[0]
	if len(r.Holes) != 1 || r.Holes[0].Pointer != "/spec/replicas" {
		t.Fatalf("holes = %+v", r.Holes)
	}
	if r.Manifest["spec"].(map[string]interface{})["replicas"] != float64(2) {
		t.Errorf("manifest not cleaned")
	}
}
```

- [ ] **Step 2: Run, verify fail** — `go test ./stamp/ -run TestLowerDocument` → FAIL `undefined: Lower`.

- [ ] **Step 3: Implement** — add to `stamp/lower.go`:

```go
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
```

Then wire it into `Build` in `stamp/stamp.go`. Change the top of `Build`:

```go
func Build(doc interchange.Document) ([]File, error) {
	doc, err := Lower(doc)
	if err != nil {
		return nil, err
	}
	var files []File
	// ... rest unchanged (note: the existing `vals, err :=` / `schema, err :=`
	//     lines already declare err; change the FIRST such line from `:=` to `=`
	//     if the compiler complains about redeclaration, or keep `err` declared here.)
```

> IMPORTANT compile detail: `Build` currently uses `vals, err := buildValues(doc)` etc. Introducing `doc, err := Lower(doc)` at the top means `err` is already declared, so later `:=` uses that also declare a *new* variable in the same scope are fine only if they introduce at least one new name (they do: `vals`, `valsYAML`, `schema`, `chartFile`). No change needed to those lines. Verify with `go build ./...`.

- [ ] **Step 4: Run, verify pass** — `go test ./...` → PASS (Build now lowers; existing Level-1 fixtures have no markers so are unchanged).

- [ ] **Step 5: Commit**

```bash
git add stamp/lower.go stamp/stamp.go stamp/lower_test.go
git commit -m "feat(stamp): Lower document pass wired into Build"
```

---

## Task 3: End-to-end equivalence + acceptance gate for a marker fixture

**Files:** Create `testdata/installable-markers.json`; Test `stamp/lower_test.go` (add).

- [ ] **Step 1: Create the Level-0 fixture** — `testdata/installable-markers.json`: identical to `testdata/installable.json` but with the two holes expressed as INLINE MARKERS inside the manifest and NO `holes[]` arrays. The Deployment's `spec.replicas` becomes:

```json
"replicas": { "__cw_hole__": { "path": "web.replicas", "default": 1, "schema": { "type": "integer", "minimum": 1 } } }
```

and the container image becomes:

```json
"image": { "__cw_hole__": { "path": "web.image", "default": "grafana/tempo:2.6.0", "render": "quote" } }
```

Keep `chart`, `components`, `file`, `component`, `gvk`, `gate` exactly as in `installable.json`. Remove the `"holes": [...]` arrays.

- [ ] **Step 2: Write failing test** — append to `stamp/lower_test.go`:

```go
import "os" // ensure imported

func TestMarkerFixtureEqualsLevel1(t *testing.T) {
	load := func(path string) interchange.Document {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		d, err := interchange.DecodeAndValidate(data)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	level1, err := Build(load("../testdata/installable.json"))
	if err != nil {
		t.Fatal(err)
	}
	markers, err := Build(load("../testdata/installable-markers.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(level1) != len(markers) {
		t.Fatalf("file count: level1=%d markers=%d", len(level1), len(markers))
	}
	for i := range level1 {
		if level1[i].Path != markers[i].Path || string(level1[i].Content) != string(markers[i].Content) {
			t.Fatalf("mismatch for %s:\n--- level1 ---\n%s\n--- markers ---\n%s",
				level1[i].Path, level1[i].Content, markers[i].Content)
		}
	}
}
```

- [ ] **Step 3: Run, verify fail then pass** — `go test ./stamp/ -run TestMarkerFixtureEqualsLevel1`. If it fails on content mismatch, inspect the diff: the two must render byte-identically. (Both go through the same `Build`; the only difference is marker-vs-holes input, which `Lower` normalizes.)

> Note: `installable.json` (Level-1) is NOT re-lowered destructively — `Lower` finds no markers in it, so its pre-declared `holes[]` remain and its manifest is unchanged. The marker fixture yields the same holes via extraction. Pointers must match: `/spec/replicas` and `/spec/template/spec/containers/0/image` — confirm the hand-written Level-1 pointers equal what `lowerManifest` computes.

- [ ] **Step 4: Verify the marker fixture is installable** — reuse the gate manually:

```bash
go run ./cmd/stamp --in testdata/installable-markers.json --out /tmp/markers-chart --no-validate-output
helm lint /tmp/markers-chart
helm template acc /tmp/markers-chart | kubeconform -strict -summary -
```
Expected: lint clean, 2/2 resources valid — identical to the Level-1 fixture.

- [ ] **Step 5: Commit**

```bash
git add testdata/installable-markers.json stamp/lower_test.go
git commit -m "test(stamp): marker fixture lowers to the same installable chart as Level-1"
```

---

## Done criteria for Plan 2a

- `go test ./...` passes.
- A manifest with inline `__cw_hole__` markers produces the identical chart to the equivalent hand-written Level-1 interchange.
- The marker fixture passes the installability gate (`helm lint` + `kubeconform`).
- Hand-written Level-1 input is unaffected (no markers → no change).

This is the seam **Plan 2b** targets: the jsonnet `helm.value()` helper emits exactly these `__cw_hole__` markers, and generators build normal k8s objects containing them.

## Self-review notes

- **Spec coverage:** marker detection + pointer computation + placeholder (Task 1); document-level pass + Build integration (Task 2); equivalence to Level-1 + installability (Task 3). Determinism via sorted keys/pointers (Task 1 test).
- **Type consistency:** `lowerManifest(map[string]interface{}) (map[string]interface{}, []interchange.Hole, error)`, `Lower(interchange.Document) (interchange.Document, error)`, marker key `__cw_hole__`, payload fields `path/default/schema/render/raw/required` used consistently and matching `interchange.Hole`.
- **Placeholder scan:** complete code in every step; the one prose note (Build `err` shadowing) is a compile caveat with the fix stated.
