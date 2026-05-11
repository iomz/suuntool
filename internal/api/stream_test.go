package api

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write(data)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestDoStream_PassesThroughPlainBody(t *testing.T) {
	payload := []byte(`{"ok":true}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "SK123", time.Second)
	rc, err := c.DoStream(context.Background(), "GET", "test", nil, nil)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestDoStream_TransparentlyGunzipsWithHeader(t *testing.T) {
	original := []byte("line1\nline2\nline3\n")
	compressed := gzipBytes(t, original)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(compressed)
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "SK123", time.Second)
	rc, err := c.DoStream(context.Background(), "GET", "test", nil, nil)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, original, got)
}

func TestDoStream_TransparentlyGunzipsWithoutHeader(t *testing.T) {
	original := []byte("line1\nline2\nline3\n")
	compressed := gzipBytes(t, original)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Intentionally no Content-Encoding header — magic-bytes path.
		_, _ = w.Write(compressed)
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "SK123", time.Second)
	rc, err := c.DoStream(context.Background(), "GET", "test", nil, nil)
	require.NoError(t, err)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, original, got)
}

func TestDoStream_MapsStatusToTypedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(srv.URL+"/", "SK123", time.Second)
	rc, err := c.DoStream(context.Background(), "GET", "missing", nil, nil)
	require.Nil(t, rc)
	require.Error(t, err)

	e, ok := err.(*Error)
	require.True(t, ok)
	require.Equal(t, "NOT_FOUND", e.Code)
	require.Equal(t, 6, e.Exit)
}
