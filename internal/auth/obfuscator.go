package auth

import (
	"encoding/base64"
	"unicode/utf8"
)

// keyObfuscator ports com.stt.android.billing.KeyObfuscator.a: byte-wise XOR
// of UTF-8 bytes against the repeating bytes of pkg.
func keyObfuscator(s, pkg string) []byte {
	k := []byte(s)
	p := []byte(pkg)
	out := make([]byte, len(k))
	for i, b := range k {
		out[i] = b ^ p[i%len(p)]
	}
	return out
}

// utf8Replace mimics Java `new String(bytes, UTF-8)` followed by .getBytes(UTF-8):
// any invalid UTF-8 byte sequence becomes U+FFFD (EF BF BD). Valid sequences pass through.
func utf8Replace(b []byte) string {
	var out []byte
	for i := 0; i < len(b); {
		r, size := utf8.DecodeRune(b[i:])
		if r == utf8.RuneError && size == 1 {
			out = append(out, 0xEF, 0xBF, 0xBD)
			i++
			continue
		}
		out = append(out, b[i:i+size]...)
		i += size
	}
	return string(out)
}

func deriveObfuscatedSecret(parts []string, pkg string) string {
	joined := ""
	for _, p := range parts {
		joined += p
	}
	raw, err := base64.StdEncoding.DecodeString(joined)
	if err != nil {
		// Base64 is well-formed for the embedded constants; an error here means
		// the constants got corrupted, which is a programming error.
		panic("auth: base64 decode of embedded key parts failed: " + err.Error())
	}
	mid := utf8Replace(raw)
	xored := keyObfuscator(mid, pkg)
	return utf8Replace(xored)
}

// DeriveLoginSecret returns the secret used to sign /login2 form submissions.
func DeriveLoginSecret() string {
	return deriveObfuscatedSecret([]string{loginKeyPart1, loginKeyPart2, loginKeyPart3}, PackageName)
}

// DeriveTOTPMasterSecret returns the PBKDF2 password for per-user TOTP generation.
func DeriveTOTPMasterSecret() string {
	return deriveObfuscatedSecret([]string{totpKeyPart1, totpKeyPart2}, totpObfuscationKey)
}
