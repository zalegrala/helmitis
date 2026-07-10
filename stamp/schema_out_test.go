package stamp

import (
	"encoding/json"
	"testing"

	"github.com/zalegrala/chartwright/interchange"
)

func TestBuildValuesSchema(t *testing.T) {
	doc := interchange.Document{
		Resources: []interchange.Resource{
			{Holes: []interchange.Hole{
				{Path: "web.replicas", Schema: json.RawMessage(`{"type":"integer","minimum":1}`)},
			}},
		},
	}
	out, err := buildValuesSchema(doc)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	props := parsed["properties"].(map[string]interface{})
	web := props["web"].(map[string]interface{})
	webProps := web["properties"].(map[string]interface{})
	replicas := webProps["replicas"].(map[string]interface{})
	if replicas["type"] != "integer" {
		t.Errorf("replicas schema = %v", replicas)
	}
}
