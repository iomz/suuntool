package endpoints_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tajchert/suuntool/internal/api"
	"github.com/tajchert/suuntool/internal/api/endpoints"
)

func TestPartners_DecodesAndPretty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/partnerconnection", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":[{"name":"Strava","connected":true},{"name":"TrainingPeaks","connected":false}]}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	p, err := endpoints.Partners(context.Background(), client)
	require.NoError(t, err)
	require.NotNil(t, p)

	pretty := p.Pretty()
	assert.True(t, strings.Contains(pretty, "Strava"), "Pretty should contain Strava")
	assert.True(t, strings.Contains(pretty, "connected=true"), "Pretty should contain connected=true")
	assert.True(t, strings.Contains(pretty, "TrainingPeaks"), "Pretty should contain TrainingPeaks")
	assert.True(t, strings.Contains(pretty, "connected=false"), "Pretty should contain connected=false")

	// JSON marshal should round-trip as the array.
	out, err := json.Marshal(p)
	require.NoError(t, err)
	var rows []map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &rows))
	assert.Len(t, rows, 2)
	assert.Equal(t, "Strava", rows[0]["name"])
}
