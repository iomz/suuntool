package auth

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

// Golden vector from `python3 handoff/reference/secret_check.py`.
// Update this when the embedded key constants rotate.
const expectedLoginSecretHex = "77764342455243417a3730794252723964531c7e2b5803454d394c5564065765451d774e5c4b666f2059584252402d002e01efbfbdefbfbdefbfbd5f12633667570c5e7a2e505515245a2d4d204a230c577f6e253437050c57791376"

func TestDeriveLoginSecret_MatchesPythonGolden(t *testing.T) {
	got := DeriveLoginSecret()
	gotHex := hex.EncodeToString([]byte(got))
	require.Equal(t, expectedLoginSecretHex, gotHex,
		"Go login_secret must be byte-identical to the Python reference (handoff/reference/secret_check.py)")
}

func TestKeyObfuscator_XORsAgainstRepeatingPackageBytes(t *testing.T) {
	// Trivial sanity: same input twice with same pkg → same output; XOR is its own inverse.
	in := "hello world"
	pkg := "com.stt.android.suunto"
	once := keyObfuscator(in, pkg)
	twice := keyObfuscator(string(once), pkg)
	require.Equal(t, in, string(twice), "applying obfuscator twice should round-trip")
}
