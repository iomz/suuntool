package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tajchert/suuntool/internal/api"
)

// LatLon is a latitude/longitude pair as returned by the workout API.
type LatLon struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// RemoteSyncedWorkout is the shape returned by /v1/workouts, /v1/workouts/{key},
// and /v1/workouts/{username}/public. Only fields confirmed live in handoff §5
// are included.
type RemoteSyncedWorkout struct {
	Key            string  `json:"key"`
	Username       string  `json:"username"`
	ActivityID     int     `json:"activityId"`
	StartTime      int64   `json:"startTime"`      // unix ms
	StopTime       int64   `json:"stopTime"`       // unix ms
	TotalTime      float64 `json:"totalTime"`      // seconds
	TotalDistance  float64 `json:"totalDistance"`  // meters
	TotalAscent    float64 `json:"totalAscent"`
	TotalDescent   float64 `json:"totalDescent"`
	MaxSpeed       float64 `json:"maxSpeed,omitempty"`
	Polyline       string  `json:"polyline,omitempty"`
	StepCount      int     `json:"stepCount,omitempty"`
	RecoveryTime   int     `json:"recoveryTime,omitempty"`
	StartPosition  *LatLon `json:"startPosition,omitempty"`
	StopPosition   *LatLon `json:"stopPosition,omitempty"`
	CenterPosition *LatLon `json:"centerPosition,omitempty"`
}

// Pretty returns a single summary line for the workout.
func (w RemoteSyncedWorkout) Pretty() string {
	start := time.Unix(0, w.StartTime*int64(time.Millisecond)).UTC().Format(time.RFC3339)
	km := w.TotalDistance / 1000.0
	dur := formatDuration(w.TotalTime)
	return fmt.Sprintf("%s  %s  act=%d  %.2fkm  %s", w.Key, start, w.ActivityID, km, dur)
}

// formatDuration formats seconds as h:mm:ss.
func formatDuration(secs float64) string {
	total := int(secs)
	h := total / 3600
	m := (total % 3600) / 60
	s := total % 60
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}

// WorkoutList wraps a page of workouts with the cursor metadata so JSON
// consumers see the pagination state.
type WorkoutList struct {
	Items []RemoteSyncedWorkout `json:"items"`
	Until int64                 `json:"until"` // metadata.until from the envelope
}

// Pretty returns one line per workout and a footer with the total count.
func (l WorkoutList) Pretty() string {
	var sb strings.Builder
	for _, w := range l.Items {
		sb.WriteString(w.Pretty())
		sb.WriteByte('\n')
	}
	fmt.Fprintf(&sb, "%d workouts", len(l.Items))
	return sb.String()
}

// ListWorkoutsOpts controls pagination for ListWorkouts.
type ListWorkoutsOpts struct {
	Since  int64 // unix ms; 0 = all
	Limit  int   // server max 100; if 0, default 20
	Offset int
}

// ListWorkouts fetches a single page of workouts. The caller paginates by
// reading Until and re-calling with Since=Until (or Offset+=Limit).
func ListWorkouts(ctx context.Context, c *api.Client, opts ListWorkoutsOpts) (*WorkoutList, error) {
	limit := opts.Limit
	if limit == 0 {
		limit = 20
	}

	path := fmt.Sprintf("workouts?since=%d&limit=%d&offset=%d", opts.Since, limit, opts.Offset)

	body, err := c.Do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	items, rawMeta, err := api.DecodeAskoWithMeta[[]RemoteSyncedWorkout](body)
	if err != nil {
		return nil, err
	}

	var until int64
	if len(rawMeta) > 0 {
		var meta struct {
			Until int64 `json:"until"`
		}
		_ = json.Unmarshal(rawMeta, &meta)
		until = meta.Until
	}

	if items == nil {
		items = []RemoteSyncedWorkout{}
	}

	return &WorkoutList{Items: items, Until: until}, nil
}

// GetWorkout fetches a single workout by key.
func GetWorkout(ctx context.Context, c *api.Client, key string) (*RemoteSyncedWorkout, error) {
	body, err := c.Do(ctx, "GET", "workouts/"+key, nil, nil)
	if err != nil {
		return nil, err
	}
	w, err := api.DecodeAsko[RemoteSyncedWorkout](body)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// WorkoutCount is returned by /v1/workouts/count.
type WorkoutCount struct {
	Count      int `json:"count"`
	TotalCount int `json:"totalCount"`
}

// Pretty returns a two-line key/value representation.
func (wc WorkoutCount) Pretty() string {
	return fmt.Sprintf("count:       %d\ntotalCount:  %d", wc.Count, wc.TotalCount)
}

// PerActivityStats is one row of WorkoutStats.AllStats.
type PerActivityStats struct {
	ActivityID int     `json:"activityId"`
	Count      int     `json:"count"`
	Distance   float64 `json:"distance"`
	Duration   float64 `json:"duration"`
	Energy     float64 `json:"energy"`
}

// WorkoutStats is the response from /v1/workouts/{username}/stats.
// Field names follow what Suunto returns; numerics are kept verbatim.
type WorkoutStats struct {
	TotalDistanceSum          float64            `json:"totalDistanceSum"`
	TotalTimeSum              float64            `json:"totalTimeSum"`
	TotalEnergyConsumptionSum float64            `json:"totalEnergyConsumptionSum"`
	TotalNumberOfWorkoutsSum  int                `json:"totalNumberOfWorkoutsSum"`
	TotalDays                 int                `json:"totalDays"`
	AllStats                  []PerActivityStats `json:"allStats"`
}

// formatKm formats meters as a km string with two decimal places.
func formatKm(meters float64) string {
	return fmt.Sprintf("%.2fkm", meters/1000.0)
}

// Pretty returns a multi-line summary of the aggregate stats.
func (s WorkoutStats) Pretty() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "workouts:  %d\n", s.TotalNumberOfWorkoutsSum)
	fmt.Fprintf(&sb, "distance:  %s\n", formatKm(s.TotalDistanceSum))
	fmt.Fprintf(&sb, "time:      %s\n", formatDuration(s.TotalTimeSum))
	fmt.Fprintf(&sb, "energy:    %.0f kcal\n", s.TotalEnergyConsumptionSum)
	fmt.Fprintf(&sb, "days:      %d\n", s.TotalDays)
	if len(s.AllStats) > 0 {
		fmt.Fprintf(&sb, "Per activity:\n")
		for _, a := range s.AllStats {
			fmt.Fprintf(&sb, "  %d:  %dx  %s  %s  %.0f kcal\n",
				a.ActivityID, a.Count,
				formatKm(a.Distance),
				formatDuration(a.Duration),
				a.Energy,
			)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Stats fetches /v1/workouts/{username}/stats. Empty username is rejected at the
// caller (cmd layer falls back to the session username).
func Stats(ctx context.Context, c *api.Client, username string) (*WorkoutStats, error) {
	path := fmt.Sprintf("workouts/%s/stats", username)
	body, err := c.Do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	ws, err := api.DecodeAsko[WorkoutStats](body)
	if err != nil {
		return nil, err
	}
	return &ws, nil
}

// CountWorkouts hits /v1/workouts/count. Both until and sharingFlags are
// required by the server (handoff §5 quirks). If untilMS <= 0, the caller
// should pass auth.NowMS().
func CountWorkouts(ctx context.Context, c *api.Client, untilMS int64, sharingFlags int) (*WorkoutCount, error) {
	path := fmt.Sprintf("workouts/count?until=%d&sharingFlags=%d", untilMS, sharingFlags)
	body, err := c.Do(ctx, "GET", path, nil, nil)
	if err != nil {
		return nil, err
	}
	wc, err := api.DecodeAsko[WorkoutCount](body)
	if err != nil {
		return nil, err
	}
	return &wc, nil
}
