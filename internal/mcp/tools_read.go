package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
)

// emptyArgs is the shared no-input args struct.
type emptyArgs struct{}

// doctorArgs has no input — the doctor probe is parameterless.
type doctorArgs struct{}

type profileUserArgs struct {
	Username string `json:"username" jsonschema:"the Suunto/Sports-Tracker username to look up"`
}

type workoutsListArgs struct {
	SinceMS int64 `json:"since_ms,omitempty" jsonschema:"workouts modified after this unix-millisecond timestamp (0 = all)"`
	Limit   int   `json:"limit,omitempty" jsonschema:"page size (default 20, server max 100)"`
	Offset  int   `json:"offset,omitempty" jsonschema:"pagination offset"`
}

type workoutKeyArgs struct {
	Key string `json:"key" jsonschema:"the workout key (e.g. 6634ab12cd34ef5678901234)"`
}

type workoutsCountArgs struct {
	UntilMS      int64 `json:"until_ms,omitempty" jsonschema:"upper bound timestamp (unix ms); 0 means now"`
	SharingFlags int   `json:"sharing_flags,omitempty" jsonschema:"sharing-flag bitmask required by the server (use 0 for default)"`
}

type workoutsStatsArgs struct {
	Username string `json:"username,omitempty" jsonschema:"username to fetch stats for; empty = authenticated user"`
}

type wellnessArgs struct {
	SinceMS int64 `json:"since_ms,omitempty" jsonschema:"unix-millisecond cursor; 0 = all history"`
	Limit   int   `json:"limit,omitempty" jsonschema:"max NDJSON entries to return (0 = no limit)"`
}

// authGate returns AUTH_EXPIRED when no session is loaded. Returns nil if ok.
func authGate(d *deps) *sdkmcp.CallToolResult {
	if d.session == nil {
		return mapErrorToCallToolResult(&api.Error{
			Code: "AUTH_EXPIRED", Message: "no session", Hint: "Run: suuntool login", HTTP: 401, Exit: 4,
		})
	}
	return nil
}

// readNDJSON decodes one JSON object per line from r, optionally bounded by limit.
func readNDJSON(r io.ReadCloser, limit int) ([]map[string]any, error) {
	defer r.Close()
	dec := json.NewDecoder(r)
	var out []map[string]any
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// readRegistrars returns the read-only (tierRead) tool registrars.
func readRegistrars() []toolRegistrar {
	return []toolRegistrar{
		// doctor — unauthed health probe.
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

		// whoami
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "whoami",
				Description: "Return the authenticated user's profile (GET /v1/user).",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.Whoami(ctx, d.client)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},

		// profile_settings
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "profile_settings",
				Description: "Return the authenticated user's settings (GET /v1/user/settings).",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				raw, err := endpoints.Settings(ctx, d.client)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				var v map[string]any
				if err := json.Unmarshal(raw, &v); err != nil {
					return mapErrorToCallToolResult(&api.Error{Code: "BAD_ENVELOPE", Message: err.Error(), Exit: 5}), nil, nil
				}
				return nil, v, nil
			})
		},

		// profile_follow
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "profile_follow",
				Description: "Return social follow/block counts for the authenticated user (GET /v1/user/follow).",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, _ emptyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.Follow(ctx, d.client)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},

		// profile_user — lookup by username
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "profile_user",
				Description: "Look up a user profile by username (GET /v1/user/name/{username}).",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args profileUserArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.UserByName(ctx, d.client, args.Username)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},

		// workouts_list
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_list",
				Description: "List workouts (GET /v1/workouts). Paginated by since/limit/offset. Each item is enriched with activityName alongside the numeric activityId.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutsListArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.ListWorkouts(ctx, d.client, endpoints.ListWorkoutsOpts{
					Since: args.SinceMS, Limit: args.Limit, Offset: args.Offset,
				})
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, enrichWorkoutList(v), nil
			})
		},

		// workouts_get
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_get",
				Description: "Fetch a single workout summary by key (GET /v1/workouts/{key}). Response includes activityName alongside the numeric activityId.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutKeyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.GetWorkout(ctx, d.client, args.Key)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				if v == nil {
					return nil, nil, nil
				}
				enriched := enrichWorkout(*v)
				return nil, enriched, nil
			})
		},

		// workouts_count
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_count",
				Description: "Return workout counts (GET /v1/workouts/count). Both until and sharingFlags are required by the server.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutsCountArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.CountWorkouts(ctx, d.client, args.UntilMS, args.SharingFlags)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},

		// workouts_stats
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_stats",
				Description: "Per-activity totals for a user (GET /v1/workouts/{username}/stats). Empty username defaults to the authenticated user. Each allStats entry is enriched with activityName.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutsStatsArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				username := args.Username
				if username == "" {
					username = d.session.Username
				}
				if username == "" {
					return mapErrorToCallToolResult(&api.Error{Code: "USAGE", Message: "username required (session has no username)", Exit: 2}), nil, nil
				}
				v, err := endpoints.Stats(ctx, d.client, username)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, enrichWorkoutStats(v), nil
			})
		},

		// workouts_sml
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_sml",
				Description: "Fetch the full per-workout SML JSON blob (GET /v1/workouts/{key}/sml) and return it base64-encoded.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutKeyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				rc, err := endpoints.FetchSML(ctx, d.client, args.Key)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				defer rc.Close()
				b, err := io.ReadAll(rc)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": args.Key, "base64": base64.StdEncoding.EncodeToString(b)}, nil
			})
		},

		// workouts_fit
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_fit",
				Description: "Fetch the binary FIT export for a workout (GET /v1/workout/exportFit/{key}) base64-encoded.",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutKeyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				rc, err := endpoints.FetchFIT(ctx, d.client, args.Key)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				defer rc.Close()
				b, err := io.ReadAll(rc)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"key": args.Key, "base64": base64.StdEncoding.EncodeToString(b)}, nil
			})
		},

		// workouts_comments
		func(s *sdkmcp.Server, d *deps) {
			sdkmcp.AddTool(s, &sdkmcp.Tool{
				Name:        "workouts_comments",
				Description: "List comments on a workout (GET /v1/workouts/{key}/comments).",
			}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, args workoutKeyArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				v, err := endpoints.ListComments(ctx, d.client, args.Key)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, v, nil
			})
		},

		// wellness_sleep
		makeWellnessTool("wellness_sleep", "Stream 24/7 sleep entries as JSON objects (GET /v1/sleep/export).", endpoints.StreamSleep),
		// wellness_activity
		makeWellnessTool("wellness_activity", "Stream 24/7 activity entries as JSON objects (GET /v1/activity/export).", endpoints.StreamActivity),
		// wellness_recovery
		makeWellnessTool("wellness_recovery", "Stream 24/7 recovery entries as JSON objects (GET /v1/recovery/export).", endpoints.StreamRecovery),
		// wellness_sleepstages
		makeWellnessTool("wellness_sleepstages", "Stream 24/7 sleep-stages entries as JSON objects (GET /v1/sleepstages/export).", endpoints.StreamSleepStages),

		// activity_type_name (unauthed lookup; uses the embedded ActivityType table)
		registerActivityNameTool,
	}
}

func makeWellnessTool(name, desc string, stream endpoints.WellnessStream) toolRegistrar {
	return func(s *sdkmcp.Server, d *deps) {
		sdkmcp.AddTool(s, &sdkmcp.Tool{Name: name, Description: desc},
			func(ctx context.Context, _ *sdkmcp.CallToolRequest, args wellnessArgs) (*sdkmcp.CallToolResult, any, error) {
				if e := authGate(d); e != nil {
					return e, nil, nil
				}
				rc, err := endpoints.FetchWellness(ctx, d.timelineClient, stream, args.SinceMS)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				items, err := readNDJSON(rc, args.Limit)
				if err != nil {
					return mapErrorToCallToolResult(err), nil, nil
				}
				return nil, map[string]any{"items": items}, nil
			})
	}
}
