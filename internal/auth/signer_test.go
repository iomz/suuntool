package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Golden vector from `python3 handoff/reference/secret_check.py`.
const expectedSampleSignature = "Sf8mC1rPA6rrZh0uHpdwh-TkSlQLO0hkKs4S_6vdBqo"

func TestSignParams_FooBarSampleMatchesPythonGolden(t *testing.T) {
	got := SignParams("login", []Param{
		{"l", "foo@bar.com"},
		{"p", "Pass123"},
	})
	require.Equal(t, expectedSampleSignature, got)
}

func TestSignParams_RoundTrip(t *testing.T) {
	// Same call twice must produce the same deterministic output.
	a := SignParams("login", []Param{{"l", "user@example.com"}, {"p", "secret"}})
	b := SignParams("login", []Param{{"l", "user@example.com"}, {"p", "secret"}})
	require.Equal(t, a, b)
}
