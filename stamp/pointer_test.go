package stamp

import "testing"

func TestSetAtPointerMap(t *testing.T) {
	m := map[string]interface{}{"spec": map[string]interface{}{"replicas": float64(0)}}
	if err := setAtPointer(m, "/spec/replicas", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	got := m["spec"].(map[string]interface{})["replicas"]
	if got != "TOKEN" {
		t.Errorf("got %v, want TOKEN", got)
	}
}

func TestSetAtPointerSlice(t *testing.T) {
	m := map[string]interface{}{
		"spec": map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"image": "x"},
			},
		},
	}
	if err := setAtPointer(m, "/spec/containers/0/image", "TOKEN"); err != nil {
		t.Fatal(err)
	}
	c := m["spec"].(map[string]interface{})["containers"].([]interface{})
	if c[0].(map[string]interface{})["image"] != "TOKEN" {
		t.Errorf("got %v", c[0])
	}
}

func TestSetAtPointerMissing(t *testing.T) {
	m := map[string]interface{}{"spec": map[string]interface{}{}}
	if err := setAtPointer(m, "/spec/nope/x", "TOKEN"); err == nil {
		t.Fatal("expected error for missing path")
	}
}

func TestDeepCopyIsolates(t *testing.T) {
	orig := map[string]interface{}{"a": map[string]interface{}{"b": float64(1)}}
	cp, err := deepCopy(orig)
	if err != nil {
		t.Fatal(err)
	}
	cp["a"].(map[string]interface{})["b"] = float64(2)
	if orig["a"].(map[string]interface{})["b"] != float64(1) {
		t.Error("deepCopy did not isolate nested map")
	}
}
