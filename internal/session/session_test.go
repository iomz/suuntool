package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSaveLoadRoundTrip_EnforcesPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	t.Setenv("SUUNTOOL_SESSION_FILE", path)

	in := &Session{SessionKey: "abc", Username: "u", Email: "e@x", OffsetMS: 123}
	require.NoError(t, Save(in))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())

	got, err := Load()
	require.NoError(t, err)
	require.Equal(t, in.SessionKey, got.SessionKey)
	require.Equal(t, in.Username, got.Username)
	require.Equal(t, in.OffsetMS, got.OffsetMS)

	require.NoError(t, Clear())
	_, err = Load()
	require.ErrorIs(t, err, ErrNoSession)
}
