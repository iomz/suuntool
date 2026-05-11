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

func TestListWorkouts_DecodesEnvelopeAndCursor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/workouts", r.URL.Path)
		assert.Equal(t, "SK", r.Header.Get("STTAuthorization"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"error": null,
			"payload": [{"key":"wk1","username":"alice","activityId":1,"startTime":1700000000000,"totalDistance":5000.0,"totalTime":1800.0}],
			"metadata": {"until":1700001000000}
		}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	list, err := endpoints.ListWorkouts(context.Background(), client, endpoints.ListWorkoutsOpts{})
	require.NoError(t, err)
	require.NotNil(t, list)
	require.Len(t, list.Items, 1)
	assert.Equal(t, "wk1", list.Items[0].Key)
	assert.Equal(t, "alice", list.Items[0].Username)
	assert.Equal(t, int64(1700001000000), list.Until)
}

func TestGetWorkout_ParsesSingle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/workouts/wk1", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"error": null,
			"payload": {"key":"wk1","username":"alice","activityId":1,"startTime":1700000000000,"totalDistance":5000.0,"totalTime":1800.0}
		}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	w, err := endpoints.GetWorkout(context.Background(), client, "wk1")
	require.NoError(t, err)
	require.NotNil(t, w)
	assert.Equal(t, "wk1", w.Key)
	assert.Equal(t, "alice", w.Username)
	assert.Equal(t, 1, w.ActivityID)
	assert.InDelta(t, 5000.0, w.TotalDistance, 0.001)
}

func TestCountWorkouts_RequiresBothParams(t *testing.T) {
	var capturedQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":{"count":42,"totalCount":100}}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	wc, err := endpoints.CountWorkouts(context.Background(), client, 1700000000000, 0)
	require.NoError(t, err)
	require.NotNil(t, wc)
	assert.Equal(t, 42, wc.Count)
	assert.Equal(t, 100, wc.TotalCount)

	// Both params must appear in the request URL.
	assert.True(t, strings.Contains(capturedQuery, "until="), "query should contain until= param, got: %s", capturedQuery)
	assert.True(t, strings.Contains(capturedQuery, "sharingFlags="), "query should contain sharingFlags= param, got: %s", capturedQuery)
}
