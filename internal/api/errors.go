package api

import "fmt"

// Error is a typed API error with an exit code, used by cmd/ to set os.Exit.
type Error struct {
	Code    string // stable token: AUTH_EXPIRED, FORBIDDEN, NOT_FOUND, SERVER, NETWORK, BAD_ENVELOPE
	Message string
	Hint    string
	HTTP    int // 0 if not HTTP-derived
	Exit    int // process exit code (cmd.Exit*)
}

func (e *Error) Error() string {
	if e.HTTP != 0 {
		return fmt.Sprintf("%s (HTTP %d): %s", e.Code, e.HTTP, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) ExitCode() int { return e.Exit }
