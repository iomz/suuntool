package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
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

// WorkoutList wraps a page of workouts with the cursor metadata so JSON
// consumers see the pagination state.
type WorkoutList struct {
	Items []RemoteSyncedWorkout `json:"items"`
	Until int64                 `json:"until"` // metadata.until from the envelope
}

// WorkoutSummary is an aggregate over a set of workouts. Distance is meters
// and time is seconds (matching the wire types); Pretty() formats them.
type WorkoutSummary struct {
	Count         int                      `json:"count"`
	TotalDistance float64                  `json:"totalDistance"` // meters
	TotalTime     float64                  `json:"totalTime"`     // seconds
	TotalAscent   float64                  `json:"totalAscent"`
	TotalDescent  float64                  `json:"totalDescent"`
	ByActivity    map[int]PerActivityStats `json:"byActivity,omitempty"`
}

// Summary computes aggregate totals over the items in the list.
func (l WorkoutList) Summary() WorkoutSummary {
	s := WorkoutSummary{Count: len(l.Items), ByActivity: map[int]PerActivityStats{}}
	for _, w := range l.Items {
		s.TotalDistance += w.TotalDistance
		s.TotalTime += w.TotalTime
		s.TotalAscent += w.TotalAscent
		s.TotalDescent += w.TotalDescent
		a := s.ByActivity[w.ActivityID]
		a.ActivityID = w.ActivityID
		a.Count++
		a.Distance += w.TotalDistance
		a.Duration += w.TotalTime
		s.ByActivity[w.ActivityID] = a
	}
	if len(s.ByActivity) == 0 {
		s.ByActivity = nil
	}
	return s
}

// Pretty returns a multi-line summary of the aggregate, with the per-activity
// breakdown rendered as an aligned table.
func (s WorkoutSummary) Pretty() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "workouts:  %d\n", s.Count)
	fmt.Fprintf(&sb, "distance:  %s\n", formatKm(s.TotalDistance))
	fmt.Fprintf(&sb, "time:      %s\n", formatDuration(s.TotalTime))
	fmt.Fprintf(&sb, "ascent:    %.0f m\n", s.TotalAscent)
	fmt.Fprintf(&sb, "descent:   %.0f m", s.TotalDescent)
	if len(s.ByActivity) > 0 {
		ids := make([]int, 0, len(s.ByActivity))
		for id := range s.ByActivity {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		rows := make([][]string, 0, len(ids))
		for _, id := range ids {
			a := s.ByActivity[id]
			rows = append(rows, []string{
				fmt.Sprintf("%d", a.ActivityID),
				fmt.Sprintf("%d", a.Count),
				formatKm(a.Distance),
				formatDuration(a.Duration),
			})
		}
		sb.WriteString("\n\nPer activity:\n")
		sb.WriteString(renderTable([]string{"Act", "Count", "Distance", "Duration"}, rows))
	}
	return sb.String()
}

// Pretty renders the workout page as a fixed-width table with an aggregate
// footer (count, distance, time) so the human render is self-summarizing.
// Empty list still emits a header row + footer so the output is recognisable.
func (l WorkoutList) Pretty() string {
	headers := []string{"Date", "Act", "Distance", "Duration", "Ascent", "Key"}
	rows := make([][]string, 0, len(l.Items))
	for _, w := range l.Items {
		rows = append(rows, []string{
			time.Unix(0, w.StartTime*int64(time.Millisecond)).Local().Format("2006-01-02 15:04"),
			fmt.Sprintf("%d", w.ActivityID),
			fmt.Sprintf("%.2f km", w.TotalDistance/1000.0),
			formatDuration(w.TotalTime),
			fmt.Sprintf("%.0f m", w.TotalAscent),
			w.Key,
		})
	}
	s := l.Summary()
	footer := fmt.Sprintf("\n%d %s  %s  %s",
		s.Count, pluralWorkout(s.Count), formatKm(s.TotalDistance), formatDuration(s.TotalTime))
	if l.Until > 0 {
		footer += fmt.Sprintf("  (until=%d)", l.Until)
	}
	return renderTable(headers, rows) + footer
}

func pluralWorkout(n int) string {
	if n == 1 {
		return "workout"
	}
	return "workouts"
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

// Pretty returns a multi-line summary of the aggregate stats, with per-activity
// rows rendered as an aligned table.
func (s WorkoutStats) Pretty() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "workouts:  %d\n", s.TotalNumberOfWorkoutsSum)
	fmt.Fprintf(&sb, "distance:  %s\n", formatKm(s.TotalDistanceSum))
	fmt.Fprintf(&sb, "time:      %s\n", formatDuration(s.TotalTimeSum))
	fmt.Fprintf(&sb, "energy:    %.0f kcal\n", s.TotalEnergyConsumptionSum)
	fmt.Fprintf(&sb, "days:      %d", s.TotalDays)
	if len(s.AllStats) > 0 {
		rows := make([][]string, 0, len(s.AllStats))
		for _, a := range s.AllStats {
			rows = append(rows, []string{
				fmt.Sprintf("%d", a.ActivityID),
				fmt.Sprintf("%d", a.Count),
				formatKm(a.Distance),
				formatDuration(a.Duration),
				fmt.Sprintf("%.0f kcal", a.Energy),
			})
		}
		sb.WriteString("\n\nPer activity:\n")
		sb.WriteString(renderTable([]string{"Act", "Count", "Distance", "Duration", "Energy"}, rows))
	}
	return sb.String()
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

// FetchSML returns the full /v1/workouts/{key}/sml body as a stream. Despite the
// path name, the body is application/json (~5MB per workout). Caller MUST Close
// the reader.
func FetchSML(ctx context.Context, c *api.Client, key string) (io.ReadCloser, error) {
	return c.DoStream(ctx, "GET", "workouts/"+key+"/sml", nil, nil)
}

// FetchFIT returns the binary .fit export from /v1/workout/exportFit/{key}.
// Note the singular "workout/" prefix per handoff §6.19. application/octet-stream.
// Caller MUST Close.
func FetchFIT(ctx context.Context, c *api.Client, key string) (io.ReadCloser, error) {
	return c.DoStream(ctx, "GET", "workout/exportFit/"+key, nil, nil)
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

// DeleteWorkout permanently removes a workout. Returns nil on success.
// Note the path has a trailing /delete (not just /workouts/{key}) — see handoff §6.4.
func DeleteWorkout(ctx context.Context, c *api.Client, workoutKey string) error {
	_, err := c.Do(ctx, "DELETE", "workouts/"+workoutKey+"/delete", nil, nil)
	return err
}
