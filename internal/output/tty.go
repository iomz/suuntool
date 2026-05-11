package output

import (
	"os"

	"golang.org/x/term"
)

// IsStdoutTTY reports whether stdout is an interactive terminal.
// Respects NO_COLOR by returning false when set.
func IsStdoutTTY() bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}
