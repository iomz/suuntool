package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/tajchert/suuntool/internal/api/endpoints"
)

var (
	flagMapsDeviceSerial string
)

var mapsCmd = &cobra.Command{
	Use:   "maps",
	Short: "Map offline regions",
	Long:  "Commands for managing offline map regions on Suunto devices.",
}

var mapsLibraryCmd = &cobra.Command{
	Use:   "library",
	Short: "List offline-map regions associated with a watch",
	Long: `List the offline map regions associated with a Suunto watch.

Requires the device serial number. Find it in /sml responses as Source: "suunto-<sn>".
Requires an active session (run 'suuntool login' first).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, _, err := authedClient()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), pickTimeout())
		defer cancel()
		lib, err := endpoints.ListMaps(ctx, c, flagMapsDeviceSerial)
		if err != nil {
			return err
		}
		return emit(lib)
	},
	Example: `  suuntool maps library --device-serial SN123
  suuntool maps library --device-serial SN123 --format json`,
}

func init() {
	mapsLibraryCmd.Flags().StringVar(&flagMapsDeviceSerial, "device-serial", "", "Suunto device serial (required)")
	_ = mapsLibraryCmd.MarkFlagRequired("device-serial")
	mapsCmd.AddCommand(mapsLibraryCmd)
	rootCmd.AddCommand(mapsCmd)
}
