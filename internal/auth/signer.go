package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"
)

// Param is an ordered key/value pair. Order matters for the signature.
type Param struct {
	Key, Value string
}

// SignParams returns the base64url(no-padding) SHA-256 signature of:
//
//	"POST&" + path + ("&" + k + "=" + v)... + "&secret=" + DeriveLoginSecret()
//
// No URL-encoding is applied — values are concatenated verbatim, matching
// SessionRemoteApi.Companion.d() in the APK.
func SignParams(path string, params []Param) string {
	var sb strings.Builder
	sb.WriteString("POST&")
	sb.WriteString(path)
	for _, p := range params {
		sb.WriteByte('&')
		sb.WriteString(p.Key)
		sb.WriteByte('=')
		sb.WriteString(p.Value)
	}
	sb.WriteString("&secret=")
	sb.WriteString(DeriveLoginSecret())
	sum := sha256.Sum256([]byte(sb.String()))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// RandomSalt returns 16 random bytes as base64url-no-padding (matches Python random_salt()).
func RandomSalt() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

// NowMS returns the current unix timestamp in milliseconds.
func NowMS() int64 { return time.Now().UnixMilli() }
