package session

import "testing"

func TestTOTPHeaders(t *testing.T) {
	s := &Session{Email: "you@example.com", OffsetMS: 0}
	got := TOTPHeaders(s)
	if v, ok := got["x-totp"]; !ok || len(v) != 6 {
		t.Fatalf("expected 6-digit x-totp header, got %q", v)
	}
}
