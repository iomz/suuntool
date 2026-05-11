package mcp

import (
	"context"
	"testing"
)

func TestToolDefValidate(t *testing.T) {
	h := func(context.Context, *deps, []byte) (any, error) { return nil, nil }
	valid := toolDef{Name: "whoami", Description: "x", Handler: h}
	if err := valid.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := (toolDef{}).validate(); err == nil {
		t.Fatal("expected error on empty toolDef")
	}
}
