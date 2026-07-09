package stamp

import (
	"os"
	"testing"

	"github.com/zalegrala/helmitis/interchange"
)

func loadMinimal(t *testing.T) interchange.Document {
	t.Helper()
	data, err := os.ReadFile("../testdata/minimal.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := interchange.DecodeAndValidate(data)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestBuildProducesExpectedFiles(t *testing.T) {
	files, err := Build(loadMinimal(t))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Path] = true
	}
	for _, want := range []string{
		"Chart.yaml", "values.yaml", "values.schema.json",
		"templates/_helpers.tpl", "templates/web/deployment.yaml",
	} {
		if !got[want] {
			t.Errorf("missing file %q", want)
		}
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	doc := loadMinimal(t)
	a, err := Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Build(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) {
		t.Fatalf("file count differs: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Path != b[i].Path {
			t.Fatalf("path order differs at %d: %q vs %q", i, a[i].Path, b[i].Path)
		}
		if string(a[i].Content) != string(b[i].Content) {
			t.Fatalf("content differs for %q", a[i].Path)
		}
	}
}
