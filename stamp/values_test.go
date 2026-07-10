package stamp

import (
	"reflect"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

func TestSetDottedNested(t *testing.T) {
	root := map[string]interface{}{}
	if err := setDotted(root, "web.vpa.maxAllowed", "x"); err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"web": map[string]interface{}{
			"vpa": map[string]interface{}{"maxAllowed": "x"},
		},
	}
	if !reflect.DeepEqual(root, want) {
		t.Errorf("got %#v", root)
	}
}

func TestBuildValues(t *testing.T) {
	doc := interchange.Document{
		Components: map[string]interchange.Component{
			"web": {Enabled: true, Workload: "Deployment"},
		},
		Resources: []interchange.Resource{
			{Holes: []interchange.Hole{
				{Path: "web.replicas", Default: float64(3)},
				{Path: "web.vpa.maxAllowed", Default: map[string]interface{}{"cpu": "2"}},
			}},
		},
	}
	vals, err := buildValues(doc)
	if err != nil {
		t.Fatal(err)
	}
	web := vals["web"].(map[string]interface{})
	if web["enabled"] != true {
		t.Errorf("enabled = %v", web["enabled"])
	}
	if web["replicas"] != float64(3) {
		t.Errorf("replicas = %v", web["replicas"])
	}
	if web["vpa"].(map[string]interface{})["maxAllowed"].(map[string]interface{})["cpu"] != "2" {
		t.Errorf("vpa = %v", web["vpa"])
	}
}

func TestBuildValuesTwoComponentsStable(t *testing.T) {
	doc := interchange.Document{
		Components: map[string]interchange.Component{
			"web":      {Enabled: true, Workload: "Deployment"},
			"ingester": {Enabled: false, Workload: "StatefulSet"},
		},
	}
	a, err := buildValues(doc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := buildValues(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("buildValues not stable:\n a=%#v\n b=%#v", a, b)
	}
	if a["web"].(map[string]interface{})["enabled"] != true {
		t.Errorf("web.enabled = %v", a["web"])
	}
	if a["ingester"].(map[string]interface{})["enabled"] != false {
		t.Errorf("ingester.enabled = %v", a["ingester"])
	}
}
