package stamp

import (
	"reflect"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
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

func TestLowerManifestDeterministicOrder(t *testing.T) {
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
