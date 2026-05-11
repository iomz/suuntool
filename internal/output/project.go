package output

import (
	"encoding/json"
)

// projectJSON keeps only the requested fields from a JSON-encoded value.
//
// Shape rules:
//   - Top-level JSON array: project each element (each must be a JSON object;
//     non-object elements are passed through unchanged).
//   - Top-level JSON object with an "items" array: project each item, leave
//     the wrapper (e.g. "until" pagination cursor) intact.
//   - Top-level JSON object without "items": project the object itself.
//
// Unknown fields in `fields` are silently dropped — they simply don't appear
// in the output. This mirrors how `jq '{key, total}'` would behave.
func projectJSON(in []byte, fields []string) ([]byte, error) {
	if len(fields) == 0 {
		return in, nil
	}
	var v any
	if err := json.Unmarshal(in, &v); err != nil {
		return nil, err
	}
	projected := projectValue(v, fields)
	out, err := json.MarshalIndent(projected, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func projectValue(v any, fields []string) any {
	switch x := v.(type) {
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = projectObject(item, fields)
		}
		return out
	case map[string]any:
		if items, ok := x["items"].([]any); ok {
			out := make(map[string]any, len(x))
			for k, val := range x {
				out[k] = val
			}
			projected := make([]any, len(items))
			for i, item := range items {
				projected[i] = projectObject(item, fields)
			}
			out["items"] = projected
			return out
		}
		return projectObject(x, fields)
	default:
		return v
	}
}

// projectObject keeps only `fields` keys when v is a JSON object; otherwise
// returns v unchanged.
func projectObject(v any, fields []string) any {
	obj, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := make(map[string]any, len(fields))
	for _, f := range fields {
		if val, ok := obj[f]; ok {
			out[f] = val
		}
	}
	return out
}
