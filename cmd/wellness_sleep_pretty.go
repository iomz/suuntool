package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// sleepEntry mirrors the NDJSON shape from /v1/sleep/export. Fields are only
// what we render — additional fields in the wire payload are ignored.
//
// Unit quirks (preserved as wire values; the renderer converts):
//   - hrAvg, hrMin    → Hz (×60 = BPM)
//   - quality, maxSpo2 → 0..1 fraction (×100 = %)
//   - durations       → seconds
type sleepEntry struct {
	Timestamp string `json:"timestamp"`
	EntryData struct {
		Duration              float64 `json:"duration"`
		DeepSleepDuration     float64 `json:"deepSleepDuration"`
		LightSleepDuration    float64 `json:"lightSleepDuration"`
		RemSleepDuration      float64 `json:"remSleepDuration"`
		HrAvg                 float64 `json:"hrAvg"`
		HrMin                 float64 `json:"hrMin"`
		Quality               float64 `json:"quality"`
		MaxSpo2               float64 `json:"maxSpo2"`
		AvgHrv                float64 `json:"avgHrv"`
		IsNap                 bool    `json:"isNap"`
		SleepID               int64   `json:"sleepId"`
	} `json:"entryData"`
}

// renderSleepPretty parses NDJSON from r and writes a human-readable table to w.
// Buffers all entries in memory; sleep records are bounded (typically <1000
// across all-history exports) so this is fine.
func renderSleepPretty(w io.Writer, r io.Reader) error {
	entries, err := parseSleepNDJSON(r)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		_, err := fmt.Fprintln(w, "(no sleep entries)")
		return err
	}
	entries = dedupeSleep(entries)
	// Newest first — Suunto returns chronological, flip for readability.
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Timestamp > entries[j].Timestamp
	})

	headers := []string{"Date", "Type", "Duration", "REM", "Deep", "Light", "HR avg", "HR min", "HRV", "Qual", "SpO2"}
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []string{
			formatSleepDate(e.Timestamp),
			sleepKind(e.EntryData.IsNap),
			formatDurHM(e.EntryData.Duration),
			formatPctOfTotal(e.EntryData.RemSleepDuration, e.EntryData.Duration),
			formatPctOfTotal(e.EntryData.DeepSleepDuration, e.EntryData.Duration),
			formatPctOfTotal(e.EntryData.LightSleepDuration, e.EntryData.Duration),
			formatHzToBPM(e.EntryData.HrAvg),
			formatHzToBPM(e.EntryData.HrMin),
			formatHRV(e.EntryData.AvgHrv),
			formatPct01(e.EntryData.Quality),
			formatPct01(e.EntryData.MaxSpo2),
		})
	}
	if _, err := fmt.Fprintf(w, "Sleep — %d %s\n\n", len(entries), pluralize("entry", "entries", len(entries))); err != nil {
		return err
	}
	if err := writeTable(w, headers, rows); err != nil {
		return err
	}
	return writeSleepFooter(w, entries)
}

func parseSleepNDJSON(r io.Reader) ([]sleepEntry, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1 MiB per line
	var out []sleepEntry
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var e sleepEntry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parse sleep ndjson: %w", err)
		}
		out = append(out, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func sleepKind(isNap bool) string {
	if isNap {
		return "nap"
	}
	return "night"
}

func formatSleepDate(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("2006-01-02 15:04")
}

// formatDurHM renders seconds as "5h 30m" (rounded). Zero/negative → "—".
func formatDurHM(seconds float64) string {
	if seconds <= 0 {
		return "—"
	}
	total := int(math.Round(seconds / 60.0))
	h, m := total/60, total%60
	if h == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh %02dm", h, m)
}

func formatPctOfTotal(part, total float64) string {
	if total <= 0 || part <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", (part/total)*100)
}

// formatPct01 takes a 0..1 fraction → "87%". 0 → "—" (Suunto returns 0 for "unknown").
func formatPct01(v float64) string {
	if v <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", v*100)
}

// formatHzToBPM converts Hz → BPM. Sub-threshold values → "—".
func formatHzToBPM(hz float64) string {
	if hz <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", int(math.Round(hz*60)))
}

func formatHRV(ms float64) string {
	if ms <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d ms", int(math.Round(ms)))
}

func pluralize(singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}

// dedupeSleep collapses Suunto's repeated records. The 247 service emits an
// append-only stream where the same sleep is updated multiple times as data
// refines; we keep one row per (sleepId, isNap) with the longest captured
// duration (= the most complete version). Entries without a sleepId are kept
// as-is.
func dedupeSleep(in []sleepEntry) []sleepEntry {
	type k struct {
		id  int64
		nap bool
	}
	best := make(map[k]int, len(in))
	out := make([]sleepEntry, 0, len(in))
	for _, e := range in {
		if e.EntryData.SleepID == 0 {
			out = append(out, e)
			continue
		}
		key := k{e.EntryData.SleepID, e.EntryData.IsNap}
		if idx, ok := best[key]; ok {
			if e.EntryData.Duration > out[idx].EntryData.Duration {
				out[idx] = e
			}
			continue
		}
		best[key] = len(out)
		out = append(out, e)
	}
	return out
}

// writeTable renders a fixed-width text table. Aligns each column to its
// widest cell (measured in runes, not bytes — important for "—"); one space
// between columns. No ANSI styling so plain pipes work.
func writeTable(w io.Writer, headers []string, rows [][]string) error {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = utf8.RuneCountInString(h)
	}
	for _, r := range rows {
		for i, c := range r {
			n := utf8.RuneCountInString(c)
			if i < len(widths) && n > widths[i] {
				widths[i] = n
			}
		}
	}
	var sb strings.Builder
	writeRow := func(cells []string) {
		for i, c := range cells {
			if i > 0 {
				sb.WriteString("  ")
			}
			sb.WriteString(c)
			pad := widths[i] - utf8.RuneCountInString(c)
			if pad > 0 {
				sb.WriteString(strings.Repeat(" ", pad))
			}
		}
		sb.WriteByte('\n')
	}
	writeRow(headers)
	for i, wd := range widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("─", wd))
	}
	sb.WriteByte('\n')
	for _, r := range rows {
		writeRow(r)
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

// writeSleepFooter prints summary stats over night sleeps only.
func writeSleepFooter(w io.Writer, entries []sleepEntry) error {
	var nights int
	var dur, rem, deep, qual, hr float64
	var hrn int
	for _, e := range entries {
		if e.EntryData.IsNap {
			continue
		}
		nights++
		dur += e.EntryData.Duration
		rem += e.EntryData.RemSleepDuration
		deep += e.EntryData.DeepSleepDuration
		if e.EntryData.Quality > 0 {
			qual += e.EntryData.Quality
		}
		if e.EntryData.HrAvg > 0 {
			hr += e.EntryData.HrAvg
			hrn++
		}
	}
	if nights == 0 {
		return nil
	}
	avgDur := dur / float64(nights)
	avgRem := "—"
	avgDeep := "—"
	if dur > 0 {
		avgRem = fmt.Sprintf("%.0f%%", (rem/dur)*100)
		avgDeep = fmt.Sprintf("%.0f%%", (deep/dur)*100)
	}
	avgHR := "—"
	if hrn > 0 {
		avgHR = fmt.Sprintf("%d bpm", int(math.Round((hr/float64(hrn))*60)))
	}
	avgQ := "—"
	if qual > 0 {
		avgQ = fmt.Sprintf("%.0f%%", (qual/float64(nights))*100)
	}
	_, err := fmt.Fprintf(w,
		"\nAverages over %d night %s: %s sleep, %s REM, %s deep, HR %s, quality %s\n",
		nights, pluralize("sleep", "sleeps", nights),
		formatDurHM(avgDur), avgRem, avgDeep, avgHR, avgQ)
	return err
}
