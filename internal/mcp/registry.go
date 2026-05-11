package mcp

import (
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/session"
)

// tier is the gating bucket for a tool.
type tier int

const (
	tierRead tier = iota
	tierWrite
	tierDestructive
)

// deps holds runtime dependencies tool handlers close over.
// Populated by Run() in server.go.
type deps struct {
	client  *api.Client
	session *session.Session
}

// toolRegistrar registers one MCP tool on the given server, closing over d.
// Each tool has its own typed args struct, so registrars are not uniform —
// they wrap mcp.AddTool internally.
type toolRegistrar func(s *sdkmcp.Server, d *deps)
