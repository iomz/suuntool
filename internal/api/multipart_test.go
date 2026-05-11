package api

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o600))
	return p
}

func TestWorkoutMultipart_FilePartOnly(t *testing.T) {
	smlPath := writeTempFile(t, "workout.sml", `<sml>hello</sml>`)
	body, headers, err := WorkoutMultipart(smlPath, "")
	require.NoError(t, err)
	defer body.Close()

	ct, ok := headers["Content-Type"]
	require.True(t, ok)
	mediatype, params, err := mime.ParseMediaType(ct)
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediatype)
	require.NotEmpty(t, params["boundary"])

	raw, err := io.ReadAll(body)
	require.NoError(t, err)
	mr := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
	part, err := mr.NextPart()
	require.NoError(t, err)
	require.Equal(t, "filePart", part.FormName())
	require.Equal(t, "workout.sml", part.FileName())
	got, _ := io.ReadAll(part)
	require.Equal(t, "<sml>hello</sml>", string(got))
	_, err = mr.NextPart()
	require.ErrorIs(t, err, io.EOF)
}

func TestWorkoutMultipart_WithExtensions(t *testing.T) {
	smlPath := writeTempFile(t, "wk.sml", "sml-body")
	extPath := writeTempFile(t, "ext.json", `{"FitnessExtension":{}}`)

	body, headers, err := WorkoutMultipart(smlPath, extPath)
	require.NoError(t, err)
	defer body.Close()

	_, params, err := mime.ParseMediaType(headers["Content-Type"])
	require.NoError(t, err)

	raw, err := io.ReadAll(body)
	require.NoError(t, err)
	mr := multipart.NewReader(bytes.NewReader(raw), params["boundary"])

	names := []string{}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		names = append(names, p.FormName())
		_, _ = io.Copy(io.Discard, p)
	}
	require.Equal(t, []string{"filePart", "workoutExtensionsPart"}, names)
}

func TestWorkoutMultipart_RejectsMissingSML(t *testing.T) {
	_, _, err := WorkoutMultipart("/nope/missing.sml", "")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "stat sml"))
}

func TestWorkoutMultipart_StreamingNoBuffering(t *testing.T) {
	// Sanity: a 1 MB SML file should produce > 1 MB of multipart body without OOM.
	// We just check we can read it all without errors.
	big := strings.Repeat("X", 1<<20)
	smlPath := writeTempFile(t, "big.sml", big)
	body, _, err := WorkoutMultipart(smlPath, "")
	require.NoError(t, err)
	defer body.Close()
	n, err := io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.Greater(t, n, int64(1<<20))
}
