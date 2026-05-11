package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/tajchert/suuntool/internal/api/endpoints"
)

var gearCmd = &cobra.Command{
	Use:   "gear",
	Short: "Manage gear",
	Long:  "Read-side only in v0.2 — list gear paired to your account.",
}

var gearListCmd = &cobra.Command{
	Use:   "list",
	Short: "List gear paired to your account",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		list, err := endpoints.ListGear(ctx, c)
		if err != nil {
			return err
		}
		return emit(list)
	},
	Example: `  suuntool gear list
  suuntool gear list --format json -o gear.json`,
}

func init() {
	gearCmd.AddCommand(gearListCmd)
	rootCmd.AddCommand(gearCmd)
}
