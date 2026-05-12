package mcp

import (
	"context"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api/endpoints"
)

// --- destructive-tier arg structs ---

type workoutsDeleteArgs struct {
	Key string `json:"key" jsonschema:"workout key to permanently delete"`
}

type workoutsUncommentArgs struct {
	CommentKey string `json:"comment_key" jsonschema:"comment key (NOT the workout key) to delete"`
}

type workoutsUnreactArgs struct {
	Key string `json:"key" jsonschema:"workout key to remove the calling user's reaction from"`
}

// destructiveRegistrars returns the tierDestructive tool registrars.
func destructiveRegistrars() []toolRegistrar {
	return []toolRegistrar{
		// workouts_delete
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_delete",
				Description: "Permanently delete a workout (DELETE /v1/workouts/{key}/delete). Requires both --allow-write AND --allow-destructive.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsDeleteArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				if err := endpoints.DeleteWorkout(ctx, d.client, a.Key); err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"ok": true, "key": a.Key}, nil
			})
		},

		// workouts_uncomment
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_uncomment",
				Description: "Delete a comment by its comment key (DELETE /v1/workouts/comment/{commentKey}). Requires both --allow-write AND --allow-destructive.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsUncommentArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				if err := endpoints.DeleteComment(ctx, d.client, a.CommentKey); err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"ok": true, "comment_key": a.CommentKey}, nil
			})
		},

		// workouts_unreact
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_unreact",
				Description: "Remove the caller's reaction from a workout (DELETE /v1/workouts/reaction/{key}). Requires both --allow-write AND --allow-destructive.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsUnreactArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				if err := endpoints.RemoveReaction(ctx, d.client, a.Key); err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"ok": true, "key": a.Key}, nil
			})
		},
	}
}
