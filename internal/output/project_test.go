package output

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestProjectJSON_ArrayRoot(t *testing.T) {
	in := []byte(`[{"a":1,"b":2,"c":3},{"a":10,"b":20,"c":30}]`)
	out, err := projectJSON(in, []string{"a", "c"})
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	want := []map[string]any{
		{"a": float64(1), "c": float64(3)},
		{"a": float64(10), "c": float64(30)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestProjectJSON_ItemsWrapper(t *testing.T) {
	in := []byte(`{"items":[{"a":1,"b":2}],"until":99}`)
	out, err := projectJSON(in, []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got["until"] != float64(99) {
		t.Errorf("wrapper key 'until' lost: %v", got)
	}
	items, ok := got["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items missing or wrong: %v", got["items"])
	}
	item := items[0].(map[string]any)
	if _, hasB := item["b"]; hasB {
		t.Errorf("expected 'b' to be dropped: %v", item)
	}
	if item["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", item)
	}
}

func TestProjectJSON_ObjectRoot(t *testing.T) {
	in := []byte(`{"key":"abc","name":"X","total":42}`)
	out, err := projectJSON(in, []string{"key", "total"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got["key"] != "abc" || got["total"] != float64(42) {
		t.Errorf("missing fields: %v", got)
	}
	if _, hasName := got["name"]; hasName {
		t.Errorf("'name' should be dropped: %v", got)
	}
}

func TestProjectJSON_EmptyFieldsIsPassthrough(t *testing.T) {
	in := []byte(`{"a":1}`)
	out, err := projectJSON(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(in) {
		t.Errorf("expected passthrough, got %s", out)
	}
}

func TestProjectJSON_UnknownFieldsSilentlyDropped(t *testing.T) {
	in := []byte(`[{"a":1}]`)
	out, err := projectJSON(in, []string{"a", "nope"})
	if err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["a"] != float64(1) || len(got[0]) != 1 {
		t.Errorf("expected only 'a':1, got %v", got)
	}
}
