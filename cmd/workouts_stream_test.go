package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fakes /workouts with a deterministic pageable corpus so we can confirm
// --stream walks every 'until' cursor and stops on the short final page.
func TestWorkoutsList_Stream_AutoPaginates(t *testing.T) {
	const total = 250 // 3 pages: 100, 100, 50
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/workouts") {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		since, _ := strconv.Atoi(q.Get("since"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit == 0 {
			limit = 100
		}
		// startTime descends from total → 1; "since" is an exclusive upper bound.
		start := total
		if since > 0 {
			start = since - 1
		}
		items := make([]map[string]any, 0, limit)
		for i := 0; i < limit && start-i >= 1; i++ {
			st := start - i
			items = append(items, map[string]any{
				"workoutKey": fmt.Sprintf("wk_%d", st),
				"startTime":  st,
			})
		}
		var nextUntil int
		if len(items) > 0 {
			nextUntil = items[len(items)-1]["startTime"].(int)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload":  items,
			"metadata": map[string]any{"until": nextUntil},
		})
	}))
	defer srv.Close()

	// Session + base URL plumbing.
	tmp := t.TempDir()
	sessionFile := tmp + "/session.json"
	require.NoError(t, os.WriteFile(sessionFile, []byte(
		`{"sessionkey":"SK","username":"alice","email":"user@example.com","userKey":"k1"}`), 0600))
	t.Setenv("SUUNTOOL_SESSION_FILE", sessionFile)
	t.Setenv("SUUNTOOL_BASE_URL", srv.URL+"/")

	out := tmp + "/out.ndjson"
	rootCmd.SetArgs([]string{"workouts", "list", "--stream", "-o", out})
	require.NoError(t, rootCmd.Execute())

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	assert.Len(t, lines, total, "expected one NDJSON line per workout across all pages")

	// First line is highest startTime, last line lowest — verifies cursor walk.
	var first, last map[string]any
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	require.NoError(t, json.Unmarshal([]byte(lines[len(lines)-1]), &last))
	assert.Equal(t, float64(total), first["startTime"])
	assert.Equal(t, float64(1), last["startTime"])
}

// --stream --limit N caps the stream after N items even with more available.
func TestWorkoutsList_Stream_RespectsLimitCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return a full page of 100 — never the short page.
		items := make([]map[string]any, 100)
		since, _ := strconv.Atoi(r.URL.Query().Get("since"))
		start := 100000
		if since > 0 {
			start = since - 1
		}
		for i := range items {
			st := start - i
			items[i] = map[string]any{"workoutKey": fmt.Sprintf("wk_%d", st), "startTime": st}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload":  items,
			"metadata": map[string]any{"until": items[len(items)-1]["startTime"]},
		})
	}))
	defer srv.Close()

	tmp := t.TempDir()
	sessionFile := tmp + "/session.json"
	require.NoError(t, os.WriteFile(sessionFile, []byte(
		`{"sessionkey":"SK","username":"alice","email":"user@example.com","userKey":"k1"}`), 0600))
	t.Setenv("SUUNTOOL_SESSION_FILE", sessionFile)
	t.Setenv("SUUNTOOL_BASE_URL", srv.URL+"/")

	// Capture stdout.
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	rootCmd.SetArgs([]string{"workouts", "list", "--stream", "--limit", "230", "-o", ""})
	require.NoError(t, rootCmd.Execute())
	w.Close()
	got, _ := io.ReadAll(r)
	lines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")
	assert.Len(t, lines, 230)
}
