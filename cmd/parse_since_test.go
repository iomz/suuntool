package cmd

import (
	"testing"
	"time"
)

func TestParseSince(t *testing.T) {
	t.Run("empty returns 0", func(t *testing.T) {
		got, err := parseSince("")
		if err != nil || got != 0 {
			t.Fatalf("got (%d, %v); want (0, nil)", got, err)
		}
	})

	t.Run("integer unix ms passthrough", func(t *testing.T) {
		got, err := parseSince("1730000000000")
		if err != nil || got != 1730000000000 {
			t.Fatalf("got (%d, %v); want (1730000000000, nil)", got, err)
		}
	})

	t.Run("RFC3339", func(t *testing.T) {
		got, err := parseSince("2026-01-01T00:00:00Z")
		want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		if err != nil || got != want {
			t.Fatalf("got (%d, %v); want (%d, nil)", got, err, want)
		}
	})

	t.Run("YYYY-MM-DD", func(t *testing.T) {
		got, err := parseSince("2026-01-01")
		want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()
		if err != nil || got != want {
			t.Fatalf("got (%d, %v); want (%d, nil)", got, err, want)
		}
	})

	t.Run("duration days", func(t *testing.T) {
		before := time.Now()
		got, err := parseSince("7d")
		after := time.Now()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		min := before.Add(-7 * 24 * time.Hour).UnixMilli()
		max := after.Add(-7 * 24 * time.Hour).UnixMilli()
		if got < min || got > max {
			t.Fatalf("got %d; want in [%d, %d]", got, min, max)
		}
	})

	t.Run("duration hours go-native", func(t *testing.T) {
		before := time.Now()
		got, err := parseSince("2h30m")
		after := time.Now()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		min := before.Add(-(2*time.Hour + 30*time.Minute)).UnixMilli()
		max := after.Add(-(2*time.Hour + 30*time.Minute)).UnixMilli()
		if got < min || got > max {
			t.Fatalf("got %d; want in [%d, %d]", got, min, max)
		}
	})

	t.Run("keyword last-week", func(t *testing.T) {
		before := time.Now()
		got, err := parseSince("last-week")
		after := time.Now()
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		min := before.AddDate(0, 0, -7).UnixMilli()
		max := after.AddDate(0, 0, -7).UnixMilli()
		if got < min-1 || got > max+1 {
			t.Fatalf("got %d; want in [%d, %d]", got, min, max)
		}
	})

	t.Run("keyword today is midnight", func(t *testing.T) {
		got, err := parseSince("today")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		now := time.Now()
		y, m, d := now.Date()
		want := time.Date(y, m, d, 0, 0, 0, 0, now.Location()).UnixMilli()
		if got != want {
			t.Fatalf("got %d; want %d", got, want)
		}
	})

	t.Run("garbage rejected", func(t *testing.T) {
		if _, err := parseSince("not-a-time"); err == nil {
			t.Fatalf("expected error")
		}
	})
}
