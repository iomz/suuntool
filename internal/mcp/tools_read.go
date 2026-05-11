package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api/endpoints"
)

// doctorArgs has no input — the doctor probe is parameterless.
type doctorArgs struct{}

// readRegistrars returns the read-only (tierRead) tool registrars.
func readRegistrars() []toolRegistrar {
	return []toolRegistrar{
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "doctor",
				Description: "Suunto server health probe (GET /v1/servertime). Unauthenticated.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ doctorArgs) (*sdkmcp.CallToolResult, any, error) {
				v, err := endpoints.FetchServerTime(ctx, d.client)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},
	}
}
