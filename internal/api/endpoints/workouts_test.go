package endpoints_test

import (
	"context"
	"io"
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

func TestStats_DecodesEnvelope(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":null,"payload":{
			"totalDistanceSum":100000.0,
			"totalTimeSum":36000.0,
			"totalEnergyConsumptionSum":50000.0,
			"totalNumberOfWorkoutsSum":42,
			"totalDays":10,
			"allStats":[
				{"activityId":1,"count":20,"distance":50000.0,"duration":18000.0,"energy":25000.0},
				{"activityId":3,"count":22,"distance":50000.0,"duration":18000.0,"energy":25000.0}
			]
		}}`))
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	ws, err := endpoints.Stats(context.Background(), client, "alice")
	require.NoError(t, err)
	require.NotNil(t, ws)
	assert.Equal(t, 42, ws.TotalNumberOfWorkoutsSum)
	assert.Len(t, ws.AllStats, 2)
	assert.True(t, strings.Contains(capturedPath, "/workouts/alice/stats"),
		"request URL should contain /workouts/alice/stats, got: %s", capturedPath)
}

func TestFetchSML_StreamsBody(t *testing.T) {
	// 256 bytes of fake JSON-ish payload.
	fakeBody := make([]byte, 256)
	for i := range fakeBody {
		fakeBody[i] = byte(i % 128) // printable-ish bytes, not gzip magic
	}
	// Ensure not accidentally gzip magic (0x1f 0x8b).
	fakeBody[0] = '{'
	fakeBody[1] = '"'

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/workouts/wk1/sml", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeBody)
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	rc, err := endpoints.FetchSML(context.Background(), client, "wk1")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, fakeBody, got, "streamed bytes must be byte-identical to server response")
}

func TestFetchFIT_UsesSingularWorkoutPath(t *testing.T) {
	// 32 bytes of fake binary content (not gzip magic).
	fakeBody := []byte("FIT\x00fake-binary-fit-data-padding-xx")
	var capturedPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeBody)
	}))
	defer srv.Close()

	client := api.NewClient(srv.URL+"/v1/", "SK", 0)
	rc, err := endpoints.FetchFIT(context.Background(), client, "wk1")
	require.NoError(t, err)
	require.NotNil(t, rc)
	defer rc.Close()

	// The path MUST be singular "workout/" not "workouts/".
	assert.Equal(t, "/v1/workout/exportFit/wk1", capturedPath,
		"FIT export must use singular /workout/exportFit/{key} path")

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, fakeBody, got, "streamed bytes must be byte-identical to server response")
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

func TestDeleteWorkout_TrailingDeletePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "DELETE", r.Method)
		require.Equal(t, "/v1/workouts/wk1/delete", r.URL.Path)
		_, _ = w.Write([]byte(`{"error":null,"payload":true}`))
	}))
	defer srv.Close()
	c := api.NewClient(srv.URL+"/v1/", "SK", 0)
	require.NoError(t, endpoints.DeleteWorkout(context.Background(), c, "wk1"))
}
