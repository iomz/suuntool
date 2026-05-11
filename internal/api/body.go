package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// JSONBody marshals v as JSON and returns a reader + the matching Content-Type
// header (always "application/json;charset=UTF-8" — the format Suunto uses).
// Errors are returned as-is (callers wrap into *Error if they want exit codes).
func JSONBody(v any) (io.Reader, map[string]string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, nil, err
	}
	return bytes.NewReader(b), map[string]string{
		"Content-Type": "application/json;charset=UTF-8",
	}, nil
}

// TextBody wraps a plain-text body. Used by /v1/workouts/comment/{key}.
func TextBody(s string) (io.Reader, map[string]string) {
	return strings.NewReader(s), map[string]string{
		"Content-Type": "text/plain;charset=UTF-8",
	}
}
