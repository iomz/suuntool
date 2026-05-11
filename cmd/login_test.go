package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCmd_PersistsSession_AgainstFakeServer(t *testing.T) {
	// Start a fake login server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/login2") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"sessionkey":"SK","username":"alice","email":"user@example.com","userKey":"k1"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// Set up temp session file and base URL
	tmpFile := t.TempDir() + "/session.json"
	t.Setenv("SUUNTOOL_SESSION_FILE", tmpFile)
	t.Setenv("SUUNTOOL_BASE_URL", srv.URL+"/")

	// Pipe "hunter2\n" into stdin via a pipe
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, _ = io.WriteString(w, "hunter2\n")
	w.Close()
	os.Stdin = r
	defer func() {
		os.Stdin = origStdin
		r.Close()
	}()

	rootCmd.SetArgs([]string{
		"login",
		"--email", "user@example.com",
		"--password-stdin",
		"--format", "json",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)

	// Assert session file contains sessionkey
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"sessionkey": "SK"`)
}
