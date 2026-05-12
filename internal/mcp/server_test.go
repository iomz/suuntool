package mcp

import (
	"context"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestServer_ListsDoctorTool spins up Run() against an in-memory transport
// pair, connects a client, calls tools/list, and asserts the doctor tool is
// registered. This proves the read-tier registrar wiring works end-to-end.
func TestServer_ListsDoctorTool(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientT, serverT := sdkmcp.NewInMemoryTransports()

	// Run the server in a goroutine; it returns once ctx is cancelled or
	// the client closes the transport.
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, Opts{Transport: serverT})
	}()

	c := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := c.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	var found bool
	for _, tt := range res.Tools {
		if tt.Name == "doctor" {
			found = true
			if tt.Description == "" {
				t.Error("doctor tool: missing description")
			}
			break
		}
	}
	if !found {
		t.Fatalf("doctor tool not listed; got %d tools", len(res.Tools))
	}
}
