package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTool_WorkoutsComment exercises POST /v1/workouts/comment/{key} and asserts
// the x-totp header is propagated through to the upstream request.
func TestTool_WorkoutsComment(t *testing.T) {
	var sawTOTP string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/workouts/comment/w1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		sawTOTP = r.Header.Get("x-totp")
		body, _ := io.ReadAll(r.Body)
		if string(body) != "nice ride" {
			t.Fatalf("body: %q", body)
		}
		_, _ = w.Write([]byte(`{"payload":{"key":"c1","comment":"nice ride"},"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	res := callTool(t, cs, "workouts_comment", map[string]any{"key": "w1", "text": "nice ride"})
	mustOK(t, res)
	if sawTOTP == "" {
		t.Fatal("expected x-totp header on POST /workouts/comment")
	}
}

func TestTool_WorkoutsReact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/workouts/reaction/w1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-totp") == "" {
			t.Fatal("missing TOTP")
		}
		_, _ = w.Write([]byte(`{"payload":"r1","error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_react", map[string]any{"key": "w1"}))
}

func TestTool_WorkoutsEdit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/v1/workouts/w1/attributes" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var got map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if got["description"] != "new" {
			t.Fatalf("body: %+v", got)
		}
		_, _ = w.Write([]byte(`{"payload":{"key":"w1","description":"new"},"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_edit", map[string]any{
		"key":   "w1",
		"patch": map[string]any{"description": "new"},
	}))
}

func TestTool_WorkoutsBatchUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/workouts/batchUpdate" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var got []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&got)
		if len(got) != 1 || got[0]["key"] != "w1" {
			t.Fatalf("entries: %+v", got)
		}
		_, _ = w.Write([]byte(`{"payload":[{"key":"w1","ok":true}],"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_batch_update", map[string]any{
		"entries": []any{map[string]any{"key": "w1", "description": "x"}},
	}))
}

func TestTool_WorkoutsShare(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" || r.URL.Path != "/v1/workouts/alice/w1/share/gpx-route" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"payload":"https://share.example/gpx/abc","error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	res := callTool(t, cs, "workouts_share", map[string]any{"key": "w1"})
	mustOK(t, res)
	sc := res.StructuredContent.(map[string]any)
	if sc["url"] != "https://share.example/gpx/abc" {
		t.Fatalf("url: %v", sc["url"])
	}
}

func TestTool_WorkoutsExtensions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/workout/extensions/w1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		var got []string
		_ = json.NewDecoder(r.Body).Decode(&got)
		if len(got) == 0 {
			t.Fatal("expected non-empty types body")
		}
		_, _ = w.Write([]byte(`{"payload":{"FitnessExtension":{}},"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_extensions", map[string]any{"key": "w1"}))
}

func TestTool_WorkoutsUpload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v1/workout" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("content-type: %s", ct)
		}
		_, _ = w.Write([]byte(`{"payload":{"key":"wk_new","username":"alice","activityId":1,"startTime":1700000000000,"totalDistance":5000.0,"totalTime":1800.0},"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	smlB64 := base64.StdEncoding.EncodeToString([]byte("<sml>fake</sml>"))
	res := callTool(t, cs, "workouts_upload", map[string]any{"sml_base64": smlB64})
	mustOK(t, res)
}

// --- destructive tier ---

func TestTool_WorkoutsDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/workouts/w1/delete" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"payload":null,"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	res := callTool(t, cs, "workouts_delete", map[string]any{"key": "w1"})
	mustOK(t, res)
	sc := res.StructuredContent.(map[string]any)
	if sc["ok"] != true || sc["key"] != "w1" {
		t.Fatalf("payload: %+v", sc)
	}
}

func TestTool_WorkoutsUncomment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/workouts/comment/c1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"payload":null,"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_uncomment", map[string]any{"comment_key": "c1"}))
}

func TestTool_WorkoutsUnreact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/workouts/reaction/w1" {
			t.Fatalf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"payload":null,"error":null,"metadata":null}`))
	}))
	defer srv.Close()
	cs := startTestServer(t, srv.URL+"/v1/", "", authSession())
	mustOK(t, callTool(t, cs, "workouts_unreact", map[string]any{"key": "w1"}))
}

// TestDestructive_Gating verifies that when allowDestructive=false the
// destructive tools are NOT advertised, but the write tools still are.
func TestDestructive_Gating(t *testing.T) {
	srv := newSuuntoStub(t, map[string]string{})
	defer srv.Close()
	cs := startTestServerGated(t, srv.URL+"/v1/", "", authSession(), true, false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	names := map[string]bool{}
	for _, tt := range res.Tools {
		names[tt.Name] = true
	}
	if !names["workouts_comment"] {
		t.Error("expected workouts_comment (write) to be listed")
	}
	for _, n := range []string{"workouts_delete", "workouts_uncomment", "workouts_unreact"} {
		if names[n] {
			t.Errorf("destructive tool %q must NOT be listed when allowDestructive=false", n)
		}
	}
}
