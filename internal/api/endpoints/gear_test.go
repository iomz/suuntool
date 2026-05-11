package endpoints_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
)

func TestListGear_DecodesAndPretty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/gear", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":[{"serial":"GEAR001","nickname":"My Shoes","type":"shoes","distance":150000.0,"duration":54000.0,"manufacturer":"Nike"}]}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	list, err := endpoints.ListGear(context.Background(), client)
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "GEAR001", list.Items[0].Serial)

	pretty := list.Pretty()
	assert.True(t, strings.Contains(pretty, "My Shoes"), "Pretty should contain nickname")
	assert.True(t, strings.Contains(pretty, "GEAR001"), "Pretty should contain serial")
}

func TestListGear_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":[]}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	list, err := endpoints.ListGear(context.Background(), client)
	require.NoError(t, err)
	require.NotNil(t, list)
	assert.Len(t, list.Items, 0)
	assert.Equal(t, "(no gear)", list.Pretty())
}
