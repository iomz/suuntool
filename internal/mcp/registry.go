package mcp

import (
	"context"
	"errors"
)

// tier is the gating bucket for a tool.
type tier int

const (
	tierRead tier = iota
	tierWrite
	tierDestructive
)

// deps holds runtime dependencies handler functions need. Populated by
// Run() in server.go (Task 4). Declared here so registry.go can compile
// independently.
type deps struct{}

// toolHandler is the signature every MCP tool handler implements.
type toolHandler func(ctx context.Context, d *deps, rawArgs []byte) (any, error)

// toolDef is the registry entry for a single MCP tool.
type toolDef struct {
	Name        string
	Description string
	Tier        tier
	InputSchema map[string]any
	Handler     toolHandler
}

func (t toolDef) validate() error {
	if t.Name == "" {
		return errors.New("toolDef: missing Name")
	}
	if t.Description == "" {
		return errors.New("toolDef: missing Description")
	}
	if t.Handler == nil {
		return errors.New("toolDef: missing Handler")
	}
	return nil
}
