package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/output"
	"github.com/tajchert/suuntool/internal/session"
)

// Exit codes — stable, documented in --help.
const (
	ExitOK        = 0
	ExitGeneric   = 1
	ExitUsage     = 2
	ExitNetwork   = 3
	ExitAuth      = 4
	ExitServer    = 5
	ExitNotFound  = 6
	ExitForbidden = 7
)

var (
	flagOutput  string
	flagFormat  string
	flagQuiet   bool
	flagVerbose bool
	flagNoColor bool
	flagTimeout time.Duration
	flagConfig  string
)

var rootCmd = &cobra.Command{
	Use:   "suuntool",
	Short: "Unofficial CLI for the Suunto / Sports-Tracker API",
	Long: `suuntool is a command-line client for the (unofficial, reverse-engineered)
Suunto / Sports-Tracker API. v1 covers login and profile read-side endpoints.

Environment variables:
  SUUNTOOL_SESSION_FILE  Override session storage path
  SUUNTOOL_FORMAT        Default output format (json|pretty|auto)
  SUUNTOOL_TIMEOUT       Default HTTP timeout (e.g. 30s)
  NO_COLOR               Disable ANSI styling

Exit codes:
  0 ok   1 generic   2 usage   3 network   4 auth
  5 server   6 not-found   7 forbidden`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		code := ExitGeneric
		if c, ok := err.(interface{ ExitCode() int }); ok {
			code = c.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVarP(&flagOutput, "output", "o", "", "Write to file (format from extension or --format)")
	pf.StringVarP(&flagFormat, "format", "f", "auto", "Output format: json|pretty|auto")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress non-error logs")
	pf.BoolVarP(&flagVerbose, "verbose", "v", false, "Verbose (debug) logs to stderr")
	pf.BoolVar(&flagNoColor, "no-color", false, "Disable ANSI styling")
	pf.DurationVar(&flagTimeout, "timeout", 30*time.Second, "HTTP timeout")
	pf.StringVar(&flagConfig, "config", "", "Path to config file")

	_ = viper.BindPFlag("format", pf.Lookup("format"))
	_ = viper.BindPFlag("timeout", pf.Lookup("timeout"))
	viper.SetEnvPrefix("SUUNTOOL")
	viper.AutomaticEnv()
}

// baseURL returns the API base URL, honoring SUUNTOOL_BASE_URL env var.
func baseURL() string {
	if u := os.Getenv("SUUNTOOL_BASE_URL"); u != "" {
		return u
	}
	return api.DefaultBaseURL
}

// authedClient loads the saved session and returns an authenticated client.
// Returns *api.Error{Code:"AUTH_EXPIRED"} when no session is on disk.
func authedClient() (*api.Client, *session.Session, error) {
	s, err := session.Load()
	if err != nil {
		if errors.Is(err, session.ErrNoSession) {
			return nil, nil, &api.Error{
				Code:    "AUTH_EXPIRED",
				Message: "no saved session",
				Hint:    "Run: suuntool login",
				Exit:    ExitAuth,
			}
		}
		return nil, nil, err
	}
	c := api.NewClient(baseURL(), s.SessionKey, flagTimeout)
	return c, s, nil
}

// renderOpts builds output.Opts from current flags.
func renderOpts() output.Opts {
	return output.Opts{
		Format: flagFormat,
		IsTTY:  output.IsStdoutTTY(),
	}
}

// emit writes v to the output file (if --output is set) or stdout.
func emit(v any) error {
	if flagOutput != "" {
		return output.RenderToFile(flagOutput, v, renderOpts())
	}
	return output.Render(os.Stdout, v, renderOpts())
}

// pickTimeout returns flagTimeout if >0, else 30 seconds.
func pickTimeout() time.Duration {
	if flagTimeout > 0 {
		return flagTimeout
	}
	return 30 * time.Second
}
