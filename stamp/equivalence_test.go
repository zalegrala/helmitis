package stamp

import (
	"os"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

// TestMarkerFixtureEqualsLevel1 proves the lowering pass is correct end-to-end:
// a manifest with inline __cw_hole__ markers builds to a byte-identical chart as
// the equivalent hand-written Level-1 interchange (holes out-of-band).
func TestMarkerFixtureEqualsLevel1(t *testing.T) {
	load := func(path string) interchange.Document {
		t.Helper()
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
