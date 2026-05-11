# Contributing to suuntool

## Local layout

- `cmd/` — Cobra commands. One verb per file.
- `internal/api/` — HTTP client + AskoResponse envelope + typed errors.
- `internal/api/endpoints/` — thin per-resource wrappers. Add files here for new endpoints.
- `internal/auth/` — KeyObfuscator, TOTP, signature builder. Golden-vector tested against `handoff/reference/`.
- `internal/output/` — single boundary between commands and stdout/files.
- `internal/session/` — session.json persistence (0600).

## Handoff materials

The `handoff/` directory contains reverse-engineering notes and a Python reference
client. It is **excluded locally via `.git/info/exclude`** — NOT via `.gitignore`.
If you re-clone, you'll need to re-add the exclude or restore the files manually.

## Signing-key rotation

Suunto rotates the embedded signing keys on major app version bumps. When this
happens:

1. Decompile the new APK with `jadx` (see `handoff/SUUNTO_API.md` §11).
2. Update `internal/auth/keys.go` constants from:
   - `R.string.signing_key_p1` → `loginKeyPart1`
   - `SignInRemoteModule.java` → `loginKeyPart2`, `loginKeyPart3`
   - `R.string.signing_key_totp_p1` → `totpKeyPart1`
   - `RemoteModule.z()` → `totpKeyPart2`, `totpObfuscationKey`
3. Regenerate golden vectors via `python3 handoff/reference/secret_check.py` (after also updating the constants there).
4. Update the corresponding `expected*` constants in `internal/auth/*_test.go`.
5. Bump `AppVersionCode` to match the new APK.

## Testing

```bash
go test ./...
```

CI must pass without any live API calls. The HTTP-touching tests use `httptest.Server`.
A manual smoke test against a real account lives at `scripts/smoke.sh` (not yet written).
