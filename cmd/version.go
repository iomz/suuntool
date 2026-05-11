package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time; defaults to "dev".
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("suuntool", Version)
	},
}

func init() { rootCmd.AddCommand(versionCmd) }
