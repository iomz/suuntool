package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkoutsExportCmd_WritesBundle stands up a fake API that returns
// canned bodies for every endpoint workouts-export hits, then asserts every
// artifact lands in the bundle dir with the expected content.
func TestWorkoutsExportCmd_WritesBundle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/workouts/wk1/sml"):
			_, _ = w.Write([]byte(`{"Data":{"Samples":[]}}`))
		case strings.HasSuffix(r.URL.Path, "/workout/exportFit/wk1"):
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("FITBIN"))
		case strings.HasSuffix(r.URL.Path, "/workout/extensions/wk1"):
			_, _ = w.Write([]byte(`{"payload":[{"type":"FitnessExtension"}],"metadata":{}}`))
		case strings.HasSuffix(r.URL.Path, "/workouts/comments/wk1"):
			_, _ = w.Write([]byte(`{"payload":[{"key":"c1","comment":"nice","username":"alice","timestamp":1700000000000}],"metadata":{}}`))
		case strings.HasSuffix(r.URL.Path, "/workouts/wk1"):
			_, _ = w.Write([]byte(`{"payload":{"key":"wk1","username":"alice","activityId":1,"startTime":1700000000000,"stopTime":1700003600000,"totalTime":3600,"totalDistance":10000},"metadata":{}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	sessFile := filepath.Join(tmpDir, "session.json")
	require.NoError(t, os.WriteFile(sessFile, []byte(`{"sessionkey":"SK","username":"alice","email":"user@example.com","userKey":"k1"}`), 0o600))

	t.Setenv("SUUNTOOL_SESSION_FILE", sessFile)
	t.Setenv("SUUNTOOL_BASE_URL", srv.URL+"/")

	bundleDir := filepath.Join(tmpDir, "bundle")

	rootCmd.SetArgs([]string{
		"workouts", "export", "wk1",
		"--bundle", bundleDir,
		"--format", "json",
	})
	require.NoError(t, rootCmd.Execute())

	// All five files present
	for _, name := range []string{"workout.json", "workout.sml.json", "workout.fit", "extensions.json", "comments.json"} {
		_, err := os.Stat(filepath.Join(bundleDir, name))
		assert.NoErrorf(t, err, "expected %s in bundle", name)
	}

	// metadata is a valid JSON file
	b, err := os.ReadFile(filepath.Join(bundleDir, "workout.json"))
	require.NoError(t, err)
	var meta map[string]any
	require.NoError(t, json.Unmarshal(b, &meta))
	assert.Equal(t, "wk1", meta["key"])

	// FIT file is the raw bytes (binary passthrough, NOT JSON)
	fit, err := os.ReadFile(filepath.Join(bundleDir, "workout.fit"))
	require.NoError(t, err)
	assert.Equal(t, "FITBIN", string(fit))
}

func TestWorkoutsExportCmd_RefusesNonEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	bundleDir := filepath.Join(tmpDir, "bundle")
	require.NoError(t, os.MkdirAll(bundleDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(bundleDir, "junk"), []byte("x"), 0o644))

	sessFile := filepath.Join(tmpDir, "session.json")
	require.NoError(t, os.WriteFile(sessFile, []byte(`{"sessionkey":"SK","username":"alice","email":"user@example.com","userKey":"k1"}`), 0o600))
	t.Setenv("SUUNTOOL_SESSION_FILE", sessFile)
	t.Setenv("SUUNTOOL_BASE_URL", "http://127.0.0.1:1/")

	rootCmd.SetArgs([]string{"workouts", "export", "wk1", "--bundle", bundleDir})
	err := rootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not empty")
}
