package cmd

import (
	"context"
	"io"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
	"github.com/tajchert/suuntool/internal/output"
	"github.com/tajchert/suuntool/internal/session"
)

var (
	flagWellnessSinceMS int64
	flagWellnessOutDir  string
)

var wellnessCmd = &cobra.Command{
	Use:   "wellness",
	Short: "24/7 wellness exports (sleep, activity, recovery, sleepstages)",
	Long: `Stream 24/7 wellness data from the timeline service at 247.sports-tracker.com.

Each subcommand emits gzipped NDJSON which the client decodes on the fly to plain NDJSON
(one JSON object per line). Use --since to limit the cursor (unix ms; 0 = all history).

Unit quirks (pass-through, NOT normalized — see handoff §5):
  - hrAvg, hrMin are in Hz (beats per second). Multiply by 60 for BPM.
  - durations are in seconds (float).
  - quality, maxSpo2, balance are 0..1 fractions. Multiply by 100 for percent.`,
}

func newWellnessStreamCmd(stream endpoints.WellnessStream) *cobra.Command {
	return &cobra.Command{
		Use:   string(stream),
		Short: "Export " + string(stream) + " entries as NDJSON",
		Example: "  suuntool wellness " + string(stream) + " > " + string(stream) + ".ndjson\n" +
			"  suuntool wellness " + string(stream) + " --since 1730000000000 -o " + string(stream) + ".ndjson\n" +
			"  suuntool wellness " + string(stream) + " --out ./export",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := session.Load()
			if err != nil {
				return &api.Error{Code: "AUTH_EXPIRED", Message: "no saved session", Hint: "Run: suuntool login", Exit: 4}
			}
			c := api.NewTimelineClient(s.SessionKey, pickTimeout())
			ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
			defer cancel()
			body, err := endpoints.FetchWellness(ctx, c, stream, flagWellnessSinceMS)
			if err != nil {
				return err
			}
			defer body.Close()
			return writeWellness(string(stream), body)
		},
	}
}

// writeWellness picks the right sink:
//   - --out <dir>: write to <dir>/<stream>.ndjson
//   - --output / -o <path>: write to that path
//   - else: stdout
//
// Raw passthrough — sanctioned bypass of emit() per CLAUDE.md / plan P4.
func writeWellness(stream string, body io.Reader) error {
	if flagWellnessOutDir != "" {
		return output.WriteRaw(filepath.Join(flagWellnessOutDir, stream+".ndjson"), body)
	}
	if flagOutput != "" {
		return output.WriteRaw(flagOutput, body)
	}
	return output.WriteRawStdout(body)
}

func init() {
	wellnessCmd.PersistentFlags().Int64Var(&flagWellnessSinceMS, "since", 0,
		"Unix ms cursor (0 = all history)")
	wellnessCmd.PersistentFlags().StringVar(&flagWellnessOutDir, "out", "",
		"Write to <dir>/<stream>.ndjson instead of stdout")
	for _, s := range []endpoints.WellnessStream{
		endpoints.StreamSleep,
		endpoints.StreamActivity,
		endpoints.StreamRecovery,
		endpoints.StreamSleepStages,
	} {
		wellnessCmd.AddCommand(newWellnessStreamCmd(s))
	}
	rootCmd.AddCommand(wellnessCmd)
}
