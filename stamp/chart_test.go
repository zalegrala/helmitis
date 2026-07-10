package stamp

import (
	"strings"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

func TestChartYAML(t *testing.T) {
	c := interchange.Chart{Name: "demo", Version: "0.1.0", AppVersion: "2.6.0"}
	out, err := chartYAML(c)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"apiVersion: v2", "name: demo", "version: 0.1.0", "appVersion: 2.6.0", "type: application"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
}

func TestHelpersTplStable(t *testing.T) {
	if !strings.Contains(helpersTpl, "{{- define") {
		t.Error("helpers should define named templates")
	}
}
