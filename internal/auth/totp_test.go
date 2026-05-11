package auth

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	// Captured from Python (see plan Task 3 step 1).
	expectedPBKDF2HexForAlice            = "15d70cdb1125776ce85e91af922c2ba6db8917d994bf2a283fc5b249acf72e8e"
	expectedHOTPAtCounter1000000ForAlice = "215251"
)

func TestPBKDF2Key_AliceMatchesPython(t *testing.T) {
	key := pbkdf2KeyForSalt("alice@example.com")
	require.Equal(t, expectedPBKDF2HexForAlice, hex.EncodeToString(key))
}

func TestHOTP_AliceCounter1000000(t *testing.T) {
	key := pbkdf2KeyForSalt("alice@example.com")
	got := hotp6(key, 1000000)
	require.Equal(t, expectedHOTPAtCounter1000000ForAlice, got)
}
