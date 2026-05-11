package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
	"github.com/tajchert/suuntool/internal/auth"
	"github.com/tajchert/suuntool/internal/output"
)

var workoutsCmd = &cobra.Command{
	Use:   "workouts",
	Short: "Workout commands (list, get, count)",
	Long:  `Read-only workout commands. Requires an active session (run 'suuntool login' first).`,
}

// parseSince converts an empty string, an integer string, or an RFC3339
// string to a unix millisecond timestamp. Returns 0 for an empty string.
func parseSince(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0, fmt.Errorf("--since: cannot parse %q as integer or RFC3339 timestamp", s)
	}
	return t.UnixMilli(), nil
}

var (
	workoutsListLimit  int
	workoutsListSince  string
	workoutsListOffset int
)

var workoutsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your workouts (paginated)",
	Long: `List your synced workouts. Results are paginated; use --limit and --since
to control the window. If --limit exceeds one page (100), the command
automatically fetches subsequent pages using the server-returned cursor.`,
	Example: `  suuntool workouts list --limit 5
  suuntool workouts list --since 2026-01-01T00:00:00Z --format json
  suuntool workouts list -o workouts.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if workoutsListLimit > 100 {
			return fmt.Errorf("--limit must be <= 100 (server maximum per page); got %d", workoutsListLimit)
		}
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()

		since, err := parseSince(workoutsListSince)
		if err != nil {
			return err
		}

		limit := workoutsListLimit
		if limit == 0 {
			limit = 20
		}

		// Fetch pages, advancing the Since cursor, until we have enough items
		// or the server returns fewer items than requested (last page).
		var all []endpoints.RemoteSyncedWorkout
		var lastUntil int64
		offset := workoutsListOffset
		curSince := since

		for {
			remaining := limit - len(all)
			if remaining <= 0 {
				break
			}
			pageLimit := remaining
			if pageLimit > 100 {
				pageLimit = 100
			}

			page, err := endpoints.ListWorkouts(ctx, c, endpoints.ListWorkoutsOpts{
				Since:  curSince,
				Limit:  pageLimit,
				Offset: offset,
			})
			if err != nil {
				return err
			}

			all = append(all, page.Items...)
			lastUntil = page.Until

			// Stop if we got a partial page (last page) or nothing.
			if len(page.Items) < pageLimit {
				break
			}
			// Advance cursor for next page. Reset offset after first page.
			curSince = page.Until
			offset = 0
		}

		return emit(&endpoints.WorkoutList{Items: all, Until: lastUntil})
	},
}

var workoutsGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Fetch a single workout by key",
	Args:  cobra.ExactArgs(1),
	Example: `  suuntool workouts get abc123
  suuntool workouts get abc123 --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		w, err := endpoints.GetWorkout(ctx, c, args[0])
		if err != nil {
			return err
		}
		return emit(w)
	},
}

var (
	workoutsCountUntil        string
	workoutsCountSharingFlags int
)

var workoutsCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Count your workouts",
	Long: `Return the count and totalCount of your synced workouts from the server.
Both --until and --sharing-flags are required server-side; this command
defaults them so you can invoke with no flags. Pass --until as an RFC3339
timestamp or unix milliseconds; omitting it uses the current time.`,
	Example: `  suuntool workouts count
  suuntool workouts count --until 2026-01-01T00:00:00Z
  suuntool workouts count --sharing-flags 1 --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()

		untilMS, err := parseSince(workoutsCountUntil)
		if err != nil {
			return err
		}
		if untilMS <= 0 {
			untilMS = auth.NowMS()
		}

		wc, err := endpoints.CountWorkouts(ctx, c, untilMS, workoutsCountSharingFlags)
		if err != nil {
			return err
		}
		return emit(wc)
	},
}

var workoutsStatsCmd = &cobra.Command{
	Use:   "stats [username]",
	Short: "Fetch aggregate workout stats for a user",
	Long: `Fetch aggregate workout statistics from the server for a given user.
If no username is provided, the currently logged-in user's stats are returned.`,
	Args: cobra.MaximumNArgs(1),
	Example: `  suuntool workouts stats
  suuntool workouts stats alice
  suuntool workouts stats --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, s, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()

		username := s.Username
		if len(args) == 1 {
			username = args[0]
		}

		ws, err := endpoints.Stats(ctx, c, username)
		if err != nil {
			return err
		}
		return emit(ws)
	},
}

var workoutsSMLCmd = &cobra.Command{
	Use:   "sml <key>",
	Short: "Download the full SML data for a workout",
	Long: `Download the full SML payload for a workout from /v1/workouts/{key}/sml.
Output is JSON despite the path name. Default writes to stdout; use -o to save to a file.

For large workouts (~5MB) it is strongly recommended to use -o <file> rather than
piping stdout.`,
	Args: cobra.ExactArgs(1),
	Example: `  suuntool workouts sml wk1 -o wk1.sml.json
  suuntool workouts sml wk1 | jq '.Data.Samples | length'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()

		rc, err := endpoints.FetchSML(ctx, c, args[0])
		if err != nil {
			return err
		}
		defer rc.Close()

		// Raw passthrough — sanctioned bypass of emit() per CLAUDE.md / plan P4.
		// SML responses are ~5MB JSON; rendering through Render would waste memory
		// and the user almost always wants -o <file>.
		if flagOutput != "" {
			return output.WriteRaw(flagOutput, rc)
		}
		return output.WriteRawStdout(rc)
	},
}

var workoutsFITCmd = &cobra.Command{
	Use:   "fit <key>",
	Short: "Download the binary .fit export for a workout",
	Long: `Download the binary .fit export for a workout from /v1/workout/exportFit/{key}.
Returns a binary .fit file. Use -o to save; piping to stdout is binary-unsafe on some terminals.`,
	Args: cobra.ExactArgs(1),
	Example: `  suuntool workouts fit wk1 -o wk1.fit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()

		rc, err := endpoints.FetchFIT(ctx, c, args[0])
		if err != nil {
			return err
		}
		defer rc.Close()

		// Raw passthrough — sanctioned bypass of emit() per CLAUDE.md / plan P4.
		// SML responses are ~5MB JSON; rendering through Render would waste memory
		// and the user almost always wants -o <file>.
		if flagOutput != "" {
			return output.WriteRaw(flagOutput, rc)
		}
		return output.WriteRawStdout(rc)
	},
}

// workouts comments <key>
var workoutsCommentsCmd = &cobra.Command{
	Use:   "comments <key>",
	Short: "List comments on a workout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		list, err := endpoints.ListComments(ctx, c, args[0])
		if err != nil {
			return err
		}
		return emit(list)
	},
	Example: `  suuntool workouts comments wk_abc123
  suuntool workouts comments wk_abc123 --format json`,
}

// workouts comment <key> [text]
var flagCommentStdin bool

var workoutsCommentCmd = &cobra.Command{
	Use:   "comment <key> [text]",
	Short: "Post a comment on a workout (requires x-totp)",
	Long: `Post a comment on a workout. The comment body is sent as text/plain.

Requires an x-totp header (auto-generated from the session).
Quotas are conservative — don't batch-spam comments.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		var text string
		switch {
		case flagCommentStdin:
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			text = strings.TrimRight(string(b), "\r\n")
		case len(args) == 2:
			text = args[1]
		default:
			return &api.Error{Code: "USAGE", Message: "comment text required (pass as arg or use --stdin)", Exit: ExitUsage}
		}
		if strings.TrimSpace(text) == "" {
			return &api.Error{Code: "USAGE", Message: "refusing to post empty comment", Exit: ExitUsage}
		}
		c, sess, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		raw, err := endpoints.PostComment(ctx, c, key, text, totpHeaders(sess))
		if err != nil {
			return err
		}
		return emit(raw)
	},
	Example: `  suuntool workouts comment wk_abc123 "great run"
  echo "multi-line\nrun report" | suuntool workouts comment wk_abc123 --stdin`,
}

// workouts react <key>
var flagReactType string

var workoutsReactCmd = &cobra.Command{
	Use:   "react <key>",
	Short: "Add a reaction to a workout (requires x-totp)",
	Long: `Add a reaction (currently only --reaction=like is supported) to a workout.

The server validates x-totp; the CLI auto-generates one from the session.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagReactType != string(endpoints.ReactionLike) {
			return &api.Error{
				Code:    "USAGE",
				Message: "unknown --reaction value " + flagReactType + " (supported: like)",
				Hint:    "Pass --reaction like",
				Exit:    ExitUsage,
			}
		}
		c, sess, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		raw, err := endpoints.AddReaction(ctx, c, args[0], endpoints.Reaction(flagReactType), totpHeaders(sess))
		if err != nil {
			return err
		}
		return emit(raw)
	},
	Example: `  suuntool workouts react wk_abc123
  suuntool workouts react wk_abc123 --reaction like --format json`,
}

// workouts unreact <key>
var workoutsUnreactCmd = &cobra.Command{
	Use:   "unreact <key>",
	Short: "Remove the caller's reaction from a workout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		if err := endpoints.RemoveReaction(ctx, c, args[0]); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintln(os.Stderr, "Removed reaction from", args[0])
		}
		return nil
	},
	Example: `  suuntool workouts unreact wk_abc123`,
}

// workouts edit <key>
var flagEditSet []string // repeatable --set field=<json-literal>

var workoutsEditCmd = &cobra.Command{
	Use:   "edit <key>",
	Short: "Edit workout attributes (partial PUT)",
	Long: `Apply a partial update to a workout's attributes via PUT /v1/workouts/{key}/attributes.

Each --set value is parsed as field=<json-literal>:
  --set totalAscent=100         # number
  --set "name=\"Morning run\""  # string (note quoting)
  --set isPublic=true           # bool
  --set notes=null              # explicit null

Edits are non-destructive (no confirmation prompt). Unknown fields are rejected
by the server — refer to handoff §6.4 if you see a 5xx.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(flagEditSet) == 0 {
			return &api.Error{Code: "USAGE", Message: "at least one --set required", Hint: "Pass --set field=<json>", Exit: ExitUsage}
		}
		patch, err := parseSetFlags(flagEditSet)
		if err != nil {
			return err
		}
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		raw, err := endpoints.EditWorkout(ctx, c, args[0], patch)
		if err != nil {
			return err
		}
		return emit(raw)
	},
	Example: `  suuntool workouts edit wk_abc123 --set totalAscent=120
  suuntool workouts edit wk_abc123 --set "name=\"Long ride\"" --set isPublic=true`,
}

// parseSetFlags converts ["field=value", ...] into a JSON-typed map. The value
// part is parsed as a JSON literal (so 100 → float64, "x" → string, true → bool,
// null → nil). On parse error, fall back to treating the value as a raw string
// — this lets users write `--set name=Morning` without quoting numerics.
func parseSetFlags(in []string) (map[string]any, error) {
	out := make(map[string]any, len(in))
	for _, raw := range in {
		idx := strings.Index(raw, "=")
		if idx < 0 {
			return nil, &api.Error{Code: "USAGE", Message: "expected field=value in --set, got " + raw, Exit: ExitUsage}
		}
		key, val := raw[:idx], raw[idx+1:]
		if key == "" {
			return nil, &api.Error{Code: "USAGE", Message: "empty field in --set " + raw, Exit: ExitUsage}
		}
		var v any
		if err := json.Unmarshal([]byte(val), &v); err == nil {
			out[key] = v
		} else {
			out[key] = val // raw string fallback
		}
	}
	return out, nil
}

// workouts batch-update
var flagBatchFile string

var workoutsBatchUpdateCmd = &cobra.Command{
	Use:   "batch-update",
	Short: "Apply a batch of workout updates (POST /v1/workouts/batchUpdate)",
	Long: `Post a JSON array of update entries. Each entry must include a "key" field
plus the fields to update. Read from a file via --file <path>, or from stdin
when --file is "-" or omitted.

Example entries file:
  [
    {"key":"wk_abc123","totalAscent":120},
    {"key":"wk_xyz789","name":"Recovery jog"}
  ]`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var src io.Reader
		switch {
		case flagBatchFile == "" || flagBatchFile == "-":
			src = os.Stdin
		default:
			f, err := os.Open(flagBatchFile)
			if err != nil {
				return &api.Error{Code: "USAGE", Message: err.Error(), Exit: ExitUsage}
			}
			defer f.Close()
			src = f
		}
		data, err := io.ReadAll(src)
		if err != nil {
			return err
		}
		var entries []map[string]any
		if err := json.Unmarshal(data, &entries); err != nil {
			return &api.Error{Code: "USAGE", Message: "invalid JSON: " + err.Error(), Hint: "Body must be a JSON array of update objects", Exit: ExitUsage}
		}
		if len(entries) == 0 {
			return &api.Error{Code: "USAGE", Message: "empty entries array", Exit: ExitUsage}
		}
		for i, e := range entries {
			if _, ok := e["key"]; !ok {
				return &api.Error{Code: "USAGE", Message: fmt.Sprintf("entry %d missing \"key\"", i), Exit: ExitUsage}
			}
		}
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		raw, err := endpoints.BatchUpdate(ctx, c, entries)
		if err != nil {
			return err
		}
		return emit(raw)
	},
	Example: `  suuntool workouts batch-update --file edits.json
  cat edits.json | suuntool workouts batch-update`,
}

// workouts share <key>
var flagShareFormat string

var workoutsShareCmd = &cobra.Command{
	Use:   "share <key>",
	Short: "Get a signed GPX share URL for a workout",
	Long: `Generate a signed GPX share URL. Format must be 'gpx-route' or 'gpx-track'.
Sends header Brand: suuntoapp as required by the server.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var format endpoints.ShareFormat
		switch flagShareFormat {
		case string(endpoints.ShareGPXRoute):
			format = endpoints.ShareGPXRoute
		case string(endpoints.ShareGPXTrack):
			format = endpoints.ShareGPXTrack
		default:
			return &api.Error{Code: "USAGE", Message: "--as must be gpx-route or gpx-track", Exit: ExitUsage}
		}
		c, sess, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		url, err := endpoints.ShareWorkout(ctx, c, sess.Username, args[0], format)
		if err != nil {
			return err
		}
		return emit(shareResult{URL: url})
	},
	Example: `  suuntool workouts share wk_abc123 --as gpx-track
  suuntool workouts share wk_abc123 --as gpx-route --format json`,
}

type shareResult struct {
	URL string `json:"url"`
}

func (s shareResult) Pretty() string { return s.URL }

// workouts extensions <key>
var flagExtTypes []string

var workoutsExtensionsCmd = &cobra.Command{
	Use:   "extensions <key>",
	Short: "Fetch workout extensions (Fitness/Intensity/Ski/…)",
	Long: `Fetches extensions for a workout via POST /v1/workout/extensions/{key}.
Despite the verb, this is a read — the body is the *filter* list of extension
types you want. Server returns whatever the workout actually has from that list.

With no --types, the full default set is requested (matches Android app behaviour).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		raw, err := endpoints.FetchExtensions(ctx, c, args[0], flagExtTypes)
		if err != nil {
			return err
		}
		return emit(raw)
	},
	Example: `  suuntool workouts extensions wk_abc123
  suuntool workouts extensions wk_abc123 --types FitnessExtension,IntensityExtension`,
}

// workouts upload
var (
	flagUploadSML        string
	flagUploadExtensions string
)

var workoutsUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload an existing SML workout file (multipart)",
	Long: `Upload a pre-built SML workout file to the server. The body is sent as
multipart/form-data with parts:

  filePart                — the SML XML (required, --sml)
  workoutExtensionsPart   — optional extensions JSON (--extensions)

This command does NOT generate SML from raw GPS/HR. Producing a valid SML
container (with header, service header, legacy workout sections, delta-chain
GPS encoding) is a non-trivial undertaking — see handoff/WORKOUT_BINARY_FORMAT.md
for the format spec. Use a Suunto watch, an existing export, or a third-party
tool to create the SML.

On success the server returns the newly assigned workout (with polyline,
recoveryTime, etc.). Save the 'key' if you need it for follow-up calls.

(Picture and video uploads via PUT /v1/workouts/{key}/image and POST /v1/workouts/{key}/video are deferred to a future release.)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagUploadSML == "" {
			return &api.Error{Code: "USAGE", Message: "--sml <path> required", Exit: ExitUsage}
		}
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		wkt, err := endpoints.UploadWorkout(ctx, c, flagUploadSML, flagUploadExtensions)
		if err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintf(os.Stderr, "Uploaded workout key=%s (%s)\n", wkt.Key, wkt.Username)
		}
		return emit(wkt)
	},
	Example: `  suuntool workouts upload --sml ./wk.sml
  suuntool workouts upload --sml ./wk.sml --extensions ./ext.json
  suuntool workouts upload --sml ./wk.sml --format json -o response.json`,
}

// workouts delete <key>
var flagDeleteYes bool

var workoutsDeleteCmd = &cobra.Command{
	Use:   "delete <key>",
	Short: "Permanently delete a workout (destructive)",
	Long: `Permanently delete a workout. THIS CANNOT BE UNDONE.

By default, asks for interactive confirmation on a TTY. In non-TTY contexts
(scripts, agents, CI) you MUST pass --yes; otherwise the command exits with
code 2 (USAGE) without making any HTTP call.

Server endpoint: DELETE /v1/workouts/{key}/delete  (note trailing /delete).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		ok, err := confirm("Really delete workout "+key+"? This cannot be undone.", flagDeleteYes)
		if err != nil {
			return err
		}
		if !ok {
			if !flagQuiet {
				fmt.Fprintln(os.Stderr, "Aborted.")
			}
			return nil
		}
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		if err := endpoints.DeleteWorkout(ctx, c, key); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintln(os.Stderr, "Deleted workout", key)
		}
		return nil
	},
	Example: `  suuntool workouts delete wk_abc123          # interactive prompt on TTY
  suuntool workouts delete wk_abc123 --yes    # non-interactive (scripts/agents)`,
}

// workouts uncomment <comment-key>
var workoutsUncommentCmd = &cobra.Command{
	Use:   "uncomment <comment-key>",
	Short: "Delete a comment (by comment key, not workout key)",
	Long: `Delete a comment. NOTE: the argument is the comment key (returned by
'workouts comments <workout-key>'), NOT the workout key.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		if err := endpoints.DeleteComment(ctx, c, args[0]); err != nil {
			return err
		}
		if !flagQuiet {
			fmt.Fprintln(os.Stderr, "Deleted comment", args[0])
		}
		return nil
	},
	Example: `  suuntool workouts uncomment c_xyz789`,
}

func init() {
	workoutsListCmd.Flags().IntVar(&workoutsListLimit, "limit", 20, "Number of workouts to fetch (max 100 per server page)")
	workoutsListCmd.Flags().StringVar(&workoutsListSince, "since", "", "Only fetch workouts after this time (RFC3339 or unix ms)")
	workoutsListCmd.Flags().IntVar(&workoutsListOffset, "offset", 0, "Page offset for first request")

	workoutsCountCmd.Flags().StringVar(&workoutsCountUntil, "until", "", "Upper bound timestamp (RFC3339 or unix ms; default = now)")
	workoutsCountCmd.Flags().IntVar(&workoutsCountSharingFlags, "sharing-flags", 0, "Sharing flags filter (server-required param)")

	workoutsCommentCmd.Flags().BoolVar(&flagCommentStdin, "stdin", false, "Read comment text from stdin (for multi-line or piped input)")

	workoutsReactCmd.Flags().StringVar(&flagReactType, "reaction", "like", "Reaction type (currently only 'like')")

	workoutsEditCmd.Flags().StringArrayVar(&flagEditSet, "set", nil, "field=<json> attribute update (repeatable, required)")
	_ = workoutsEditCmd.MarkFlagRequired("set")

	workoutsBatchUpdateCmd.Flags().StringVar(&flagBatchFile, "file", "", "Path to JSON array of update entries (default: stdin)")

	workoutsShareCmd.Flags().StringVar(&flagShareFormat, "as", "gpx-track", "Share format: gpx-route or gpx-track")
	workoutsExtensionsCmd.Flags().StringSliceVar(&flagExtTypes, "types", nil, "Extension types to request (comma-separated; empty = full default set)")

	workoutsUploadCmd.Flags().StringVar(&flagUploadSML, "sml", "", "Path to the SML file (required)")
	workoutsUploadCmd.Flags().StringVar(&flagUploadExtensions, "extensions", "", "Path to optional extensions JSON")
	_ = workoutsUploadCmd.MarkFlagRequired("sml")

	workoutsDeleteCmd.Flags().BoolVar(&flagDeleteYes, "yes", false, "Skip the confirmation prompt (required for non-TTY)")

	workoutsCmd.AddCommand(workoutsListCmd, workoutsGetCmd, workoutsCountCmd, workoutsStatsCmd, workoutsSMLCmd, workoutsFITCmd,
		workoutsCommentsCmd, workoutsCommentCmd, workoutsUncommentCmd,
		workoutsReactCmd, workoutsUnreactCmd,
		workoutsEditCmd, workoutsBatchUpdateCmd,
		workoutsShareCmd, workoutsExtensionsCmd,
		workoutsUploadCmd, workoutsDeleteCmd)
	rootCmd.AddCommand(workoutsCmd)
}
