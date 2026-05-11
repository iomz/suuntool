package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/tajchert/suuntool/internal/api/endpoints"
)

var partnerCmd = &cobra.Command{
	Use:   "partner-connections",
	Short: "Show linked partner accounts (Strava, TrainingPeaks, …)",
	Long:  `List all partner accounts linked to your Suunto profile. Requires an active session (run 'suuntool login' first).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		p, err := endpoints.Partners(ctx, c)
		if err != nil {
			return err
		}
		return emit(p)
	},
	Example: `  suuntool partner-connections
  suuntool partner-connections --format json`,
}

func init() { rootCmd.AddCommand(partnerCmd) }
