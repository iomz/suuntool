package mcp

import (
	"context"
	"encoding/base64"
	"os"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
	"github.com/tajchert/suuntool/internal/session"
)

// --- write-tier arg structs ---

type workoutsCommentArgs struct {
	Key  string `json:"key" jsonschema:"the workout key to comment on"`
	Text string `json:"text" jsonschema:"comment body (plain text)"`
}

type workoutsReactArgs struct {
	Key string `json:"key" jsonschema:"the workout key to react on"`
}

type workoutsEditArgs struct {
	Key   string         `json:"key" jsonschema:"workout key to update"`
	Patch map[string]any `json:"patch" jsonschema:"attribute changes to apply (server-shaped JSON object)"`
}

type workoutsBatchUpdateArgs struct {
	Entries []map[string]any `json:"entries" jsonschema:"array of partial-update entries (each must include a 'key' field)"`
}

type workoutsShareArgs struct {
	Username string `json:"username,omitempty" jsonschema:"username that owns the workout; empty = authenticated user"`
	Key      string `json:"key" jsonschema:"workout key to share"`
	Format   string `json:"format,omitempty" jsonschema:"share format; one of gpx-route (default) or gpx-track"`
}

type workoutsExtensionsArgs struct {
	Key   string   `json:"key" jsonschema:"workout key"`
	Types []string `json:"types,omitempty" jsonschema:"extension types to request; empty = default Android set"`
}

type workoutsUploadArgs struct {
	SMLBase64        string `json:"sml_base64" jsonschema:"base64-encoded .sml file body"`
	ExtensionsBase64 string `json:"extensions_base64,omitempty" jsonschema:"optional base64-encoded extensions.json"`
}

// writeRegistrars returns the tierWrite tool registrars.
func writeRegistrars() []toolRegistrar {
	return []toolRegistrar{
		// workouts_comment
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_comment",
				Description: "Post a comment on a workout (POST /v1/workouts/comment/{key}). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsCommentArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.PostComment(ctx, d.client, a.Key, a.Text, session.TOTPHeaders(d.session))
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": a.Key, "payload": v}, nil
			})
		},

		// workouts_react
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_react",
				Description: "Add a 'like' reaction to a workout (POST /v1/workouts/reaction/{key}). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsReactArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.AddReaction(ctx, d.client, a.Key, endpoints.ReactionLike, session.TOTPHeaders(d.session))
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": a.Key, "payload": v}, nil
			})
		},

		// workouts_edit
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_edit",
				Description: "Patch workout attributes (PUT /v1/workouts/{key}/attributes). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsEditArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				if a.Key == "" {
					return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "key required", Exit: 2}), nil, nil
				}
				v, err := endpoints.EditWorkout(ctx, d.client, a.Key, a.Patch)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": a.Key, "payload": v}, nil
			})
		},

		// workouts_batch_update
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_batch_update",
				Description: "Apply multiple partial workout updates (POST /v1/workouts/batchUpdate). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsBatchUpdateArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.BatchUpdate(ctx, d.client, a.Entries)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"payload": v}, nil
			})
		},

		// workouts_share
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_share",
				Description: "Request a signed GPX share URL for a workout (PUT /v1/workouts/{user}/{key}/share/{format}). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsShareArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				username := a.Username
				if username == "" {
					username = d.session.Username
				}
				if username == "" {
					return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "username required (session has none)", Exit: 2}), nil, nil
				}
				format := endpoints.ShareFormat(a.Format)
				if format == "" {
					format = endpoints.ShareGPXRoute
				}
				url, err := endpoints.ShareWorkout(ctx, d.client, username, a.Key, format)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": a.Key, "url": url}, nil
			})
		},

		// workouts_extensions
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_extensions",
				Description: "Fetch per-workout extension payloads (POST /v1/workout/extensions/{key}). Despite POST, this is read-shaped — gated as write because cmd parity treats it under the write tier.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsExtensionsArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.FetchExtensions(ctx, d.client, a.Key, a.Types)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": a.Key, "payload": v}, nil
			})
		},

		// workouts_upload — accepts base64, materializes temp files, calls UploadWorkout.
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_upload",
				Description: "Upload a new workout from base64-encoded SML (and optional extensions JSON) (POST /v1/workout). Requires --allow-write.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, a workoutsUploadArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				if a.SMLBase64 == "" {
					return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "sml_base64 required", Exit: 2}), nil, nil
				}
				smlBytes, err := base64.StdEncoding.DecodeString(a.SMLBase64)
				if err != nil {
					return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "sml_base64 decode: " + err.Error(), Exit: 2}), nil, nil
				}
				smlFile, err := os.CreateTemp("", "suuntool-mcp-*.sml")
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				smlPath := smlFile.Name()
				defer os.Remove(smlPath)
				if _, err := smlFile.Write(smlBytes); err != nil {
					_ = smlFile.Close()
					return mapErrorToCallToolResult(err), nil, nil
				}
				if err := smlFile.Close(); err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}

				extPath := ""
				if a.ExtensionsBase64 != "" {
					extBytes, err := base64.StdEncoding.DecodeString(a.ExtensionsBase64)
					if err != nil {
						return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "extensions_base64 decode: " + err.Error(), Exit: 2}), nil, nil
					}
					extFile, err := os.CreateTemp("", "suuntool-mcp-*.json")
					if err != nil {
						return mapErrorToCallToolResult(err), nil, nil
					}
					extPath = extFile.Name()
					defer os.Remove(extPath)
					if _, err := extFile.Write(extBytes); err != nil {
						_ = extFile.Close()
						return mapErrorToCallToolResult(err), nil, nil
					}
					if err := extFile.Close(); err != nil {
						return mapErrorToCallToolResult(err), nil, nil
					}
				}

				wkt, err := endpoints.UploadWorkout(ctx, d.client, smlPath, extPath)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, wkt, nil
			})
		},
	}
}
