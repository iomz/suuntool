package auth

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// pbkdf2KeyForSalt mirrors Java PBEKeySpec semantics: only the low 8 bits of
// each char of the master secret are used as the password bytes.
func pbkdf2KeyForSalt(salt string) []byte {
	master := DeriveTOTPMasterSecret()
	pwd := make([]byte, 0, len(master))
	for _, r := range master {
		pwd = append(pwd, byte(r&0xff))
	}
	return pbkdf2.Key(pwd, []byte(salt), 100, 32, sha1.New)
}

// hotp6 returns a 6-digit RFC 4226 HOTP code for the given key and counter.
func hotp6(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	d := mac.Sum(nil)
	off := d[len(d)-1] & 0x0f
	code := (uint32(d[off]&0x7f) << 24) |
		(uint32(d[off+1]) << 16) |
		(uint32(d[off+2]) << 8) |
		uint32(d[off+3])
	return fmt.Sprintf("%06d", code%1_000_000)
}

// GenerateTOTP returns the current 6-digit TOTP for the given salt (email/username).
// offset adjusts the wall clock — pass the server-time offset in milliseconds.
func GenerateTOTP(salt string, offsetMS int64) string {
	key := pbkdf2KeyForSalt(salt)
	now := time.Now().UnixMilli() + offsetMS
	counter := uint64(now / 30000)
	return hotp6(key, counter)
}
