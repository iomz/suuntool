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

func TestListMaps_CapturesSerialAndDecodesRegion(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":[{"regionId":"europe-west","name":"Western Europe","status":"downloaded","size":104857600}]}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	lib, err := endpoints.ListMaps(context.Background(), client, "SN123")
	require.NoError(t, err)
	require.NotNil(t, lib)
	require.Len(t, lib.Items, 1)
	assert.Equal(t, "europe-west", lib.Items[0].RegionID)

	assert.True(t, strings.Contains(capturedURL, "deviceSerialNumber=SN123"),
		"request URL should contain deviceSerialNumber=SN123, got: %s", capturedURL)
}

func TestListMaps_EmptySerialReturnsError(t *testing.T) {
	client := api.NewClient("http://localhost/v1/", "SK", 0)
	_, err := endpoints.ListMaps(context.Background(), client, "")
	require.Error(t, err)
	var apiErr *api.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "USAGE", apiErr.Code)
	assert.Equal(t, 2, apiErr.Exit)
}
