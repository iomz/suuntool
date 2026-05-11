package cmd

import (
	"github.com/spf13/cobra"
)

type endpointRow struct {
	Command    string `json:"command"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	AuthNeeded bool   `json:"authNeeded"`
}

var endpointTable = []endpointRow{
	{"login", "POST", "/v1/login2", false},
	{"logout", "GET", "/v1/logout", true},
	{"whoami", "GET", "/v1/user", true},
	{"profile settings", "GET", "/v1/user/settings", true},
	{"profile follow", "GET", "/v1/user/follow", true},
	{"profile user <username>", "GET", "/v1/user/name/{username}", true},
	{"doctor", "GET", "/v1/servertime", false},
	{"workouts list", "GET", "/v1/workouts", true},
	{"workouts get <key>", "GET", "/v1/workouts/{key}", true},
	{"workouts count", "GET", "/v1/workouts/count", true},
	{"workouts stats [username]", "GET", "/v1/workouts/{username}/stats", true},
	{"workouts sml <key>", "GET", "/v1/workouts/{key}/sml", true},
	{"workouts fit <key>", "GET", "/v1/workout/exportFit/{key}", true},
	{"wellness sleep", "GET", "https://247.sports-tracker.com/v1/sleep/export", true},
	{"wellness activity", "GET", "https://247.sports-tracker.com/v1/activity/export", true},
	{"wellness recovery", "GET", "https://247.sports-tracker.com/v1/recovery/export", true},
	{"wellness sleepstages", "GET", "https://247.sports-tracker.com/v1/sleepstages/export", true},
	{"partner-connections", "GET", "/v1/partnerconnection", true},
	{"gear list", "GET", "/v1/gear", true},
	{"maps library", "GET", "/v1/maps/library", true},
	{"workouts comments <key>", "GET", "/v1/workouts/comments/{key}", true},
	{"workouts comment <key> [text]", "POST", "/v1/workouts/comment/{key}", true},
	{"workouts uncomment <comment-key>", "DELETE", "/v1/workouts/comment/{commentKey}", true},
}

var endpointsCmd = &cobra.Command{
	Use:    "endpoints",
	Short:  "Print the command→endpoint mapping (machine-readable with --format=json)",
	Hidden: true, // discoverable via `suuntool help endpoints`
	RunE: func(cmd *cobra.Command, args []string) error {
		return emit(endpointTable)
	},
}

func init() { rootCmd.AddCommand(endpointsCmd) }
