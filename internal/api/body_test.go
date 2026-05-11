package api

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONBody_RoundTripsValue(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}
	in := payload{Name: "alice", Value: 42}

	r, headers, err := JSONBody(in)
	require.NoError(t, err)
	require.NotNil(t, r)
	require.Contains(t, headers, "Content-Type")

	b, err := io.ReadAll(r)
	require.NoError(t, err)

	var out payload
	require.NoError(t, json.Unmarshal(b, &out))
	require.Equal(t, in, out)
}

func TestTextBody_PreservesBytesAndContentType(t *testing.T) {
	text := "hello"
	r, headers := TextBody(text)

	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, text, string(b))
	require.Contains(t, headers, "Content-Type")
	require.True(t, strings.Contains(headers["Content-Type"], "text/plain"),
		"Content-Type should contain text/plain, got %q", headers["Content-Type"])
}
