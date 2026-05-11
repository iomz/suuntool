package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSleepPretty_TableAndAverages(t *testing.T) {
	ndjson := strings.Join([]string{
		// Night sleep: duration 8h, HR 1.1 Hz = 66 BPM, quality 87%
		`{"timestamp":"2026-01-05T23:30:00.000+01:00","entryData":{"duration":28800,"deepSleepDuration":7200,"lightSleepDuration":14400,"remSleepDuration":7200,"hrAvg":1.1,"hrMin":1.0,"quality":0.87,"maxSpo2":0.98,"avgHrv":40,"isNap":false}}`,
		// Daytime nap: 30 min
		`{"timestamp":"2026-01-05T14:00:00.000+01:00","entryData":{"duration":1800,"deepSleepDuration":0,"lightSleepDuration":1800,"remSleepDuration":0,"hrAvg":1.0,"hrMin":0.95,"quality":0,"maxSpo2":0,"avgHrv":0,"isNap":true}}`,
	}, "\n")

	var buf bytes.Buffer
	require.NoError(t, renderSleepPretty(&buf, strings.NewReader(ndjson)))
	out := buf.String()

	// Header counts both entries
	require.Contains(t, out, "Sleep — 2 entries")

	// Both types rendered
	require.Contains(t, out, "night")
	require.Contains(t, out, "nap")

	// Unit conversions applied
	require.Contains(t, out, "66")    // hrAvg 1.1 Hz → 66 BPM
	require.Contains(t, out, "87%")   // quality 0.87 → 87%
	require.Contains(t, out, "98%")   // maxSpo2 0.98 → 98%
	require.Contains(t, out, "40 ms") // HRV

	// Durations as h:mm
	require.Contains(t, out, "8h 00m") // night sleep total
	require.Contains(t, out, "30m")    // nap

	// REM as fraction of total: 7200/28800 = 25%
	require.Contains(t, out, "25%")

	// Footer averages over nights only (1 night)
	require.Contains(t, out, "Averages over 1 night sleep")
}

func TestRenderSleepPretty_EmptyInput(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, renderSleepPretty(&buf, strings.NewReader("")))
	require.Equal(t, "(no sleep entries)\n", buf.String())
}

func TestRenderSleepPretty_GracefulOnMissingFields(t *testing.T) {
	// Entry with zero values for quality/HRV/SpO2 — should render "—" not "0%".
	ndjson := `{"timestamp":"2026-01-05T23:30:00.000+01:00","entryData":{"duration":7200,"deepSleepDuration":0,"lightSleepDuration":7200,"remSleepDuration":0,"hrAvg":0,"hrMin":0,"quality":0,"maxSpo2":0,"avgHrv":0,"isNap":false}}`
	var buf bytes.Buffer
	require.NoError(t, renderSleepPretty(&buf, strings.NewReader(ndjson)))
	out := buf.String()
	require.Contains(t, out, "—") // some placeholder is present
}
