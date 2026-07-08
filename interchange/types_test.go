package interchange

import (
	"os"
	"testing"
)

func TestDecodeMinimal(t *testing.T) {
	data, err := os.ReadFile("../testdata/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Chart.Name != "demo" {
		t.Errorf("chart name = %q, want demo", doc.Chart.Name)
	}
	if len(doc.Resources) != 1 {
		t.Fatalf("resources = %d, want 1", len(doc.Resources))
	}
	r := doc.Resources[0]
	if r.Gate != "web.enabled" {
		t.Errorf("gate = %q", r.Gate)
	}
	if len(r.Holes) != 1 || r.Holes[0].Path != "web.replicas" {
		t.Errorf("holes = %+v", r.Holes)
	}
	if got, ok := r.Holes[0].Default.(float64); !ok || got != 3 {
		t.Errorf("default = %v (%T), want 3", r.Holes[0].Default, r.Holes[0].Default)
	}
}
