package mcp

import (
	"context"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/session"
)

// Opts configures the MCP server. AllowWrite/AllowDestructive gate which
// tool tiers get registered. Transport defaults to StdioTransport when nil
// (production CLI). Tests pass an InMemoryTransport.
type Opts struct {
	AllowWrite       bool
	AllowDestructive bool
	BaseURL          string
	Timeout          time.Duration
	Transport        sdkmcp.Transport
}

// Run starts the MCP server and blocks until the context is cancelled or the
// transport closes. Session is loaded lazily; if absent, authenticated tools
// surface AUTH_EXPIRED at call-time.
func Run(ctx context.Context, o Opts) error {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sess, _ := session.Load() // may be nil — surfaced per-tool.
	sessionKey := ""
	if sess != nil {
		sessionKey = sess.SessionKey
	}
	cl := api.NewClient(o.BaseURL, sessionKey, timeout)
	d := &deps{client: cl, session: sess}

	s := sdkmcp.NewServer(&sdkmcp.Implementation{Name: "suuntool", Version: "0"}, nil)
	for _, r := range readRegistrars() {
		r(s, d)
	}
	if o.AllowWrite {
		for _, r := range writeRegistrars() {
			r(s, d)
		}
	}
	if o.AllowWrite && o.AllowDestructive {
		for _, r := range destructiveRegistrars() {
			r(s, d)
		}
	}

	t := o.Transport
	if t == nil {
		t = &sdkmcp.StdioTransport{}
	}
	return s.Run(ctx, t)
}
