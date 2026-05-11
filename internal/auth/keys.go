// Package auth implements the reverse-engineered Suunto login signing pipeline:
// KeyObfuscator XOR, secret derivation, RFC 6238 TOTP, and SHA-256 signatures.
//
// Key material is extracted from APK com.stt.android.suunto v6.8.13. On a
// Suunto app major version bump the constants below may need to be refreshed
// from the new APK; see CONTRIBUTING.md "Key rotation".
package auth

const (
	AppVersionCode = "6008013"
	PackageName    = "com.stt.android.suunto"
	UserAgent      = PackageName + "/" + AppVersionCode

	loginKeyPart1 = "FBkubDYmN28bWVQLLTsWFxcmaRB"
	loginKeyPart2 = "fN2AqIBc/IRAoNgshbxgnOGUVGlU3LC0xL0AuXXXXMXY"
	loginKeyPart3 = "RWQ4zIi0PWz4hekc1QGNTPlciNhEKV1teYSIkDGYY"

	totpKeyPart1       = "FBkubDYmN28bWVQLLTsWWhI+NAtILCNlPQc5Y"
	totpKeyPart2       = "BgiMRYjKA99Jj4HHFIqLmomOFttBQchNzcZU0QrODcDWz4hekc1QGNTPlciNhEKGl5GPDkzFyVX"
	totpObfuscationKey = "Bh8nsTyCeC0Ql2drMen78awk84AE3ZxW"

	// TOTPDummySalt is used when no email is known yet; mirrors GenerateOTPUseCaseImpl.a().
	TOTPDummySalt = "totp.validation.dummy.email@suunto.com"
)
