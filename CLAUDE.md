# CLAUDE.md — developer orientation

Audience: senior engineers (and LLM agents) joining this repo. Read this once, then read the code.

## What this is

`suuntool` is a Go CLI for the **unofficial, reverse-engineered** Suunto / Sports-Tracker HTTP API. The wire contract was extracted from the shipping Android APK (`com.stt.android.suunto`). It is meant to be used by humans and by agents driving a terminal — every UX decision flows from that dual audience.

Reverse-engineering notes (endpoint inventory, signing scheme, Python reference client, golden-vector diagnostics) live in `handoff/`. That directory is **excluded via `.git/info/exclude`** — not `.gitignore` — so it stays local but is never committed. Treat it as load-bearing reference material; don't delete it.

## Layering — strict, one-way

```
main.go
└── cmd/                 Cobra commands. Knows about session+api+output, nothing else.
    ├── root.go          persistent flags, exit codes, helpers (baseURL, authedClient, emit, pickTimeout)
    ├── login.go logout.go whoami.go profile.go doctor.go endpoints.go version.go
    └── login_test.go    end-to-end via httptest
internal/
├── auth/                signing pipeline. Zero net/http imports.
│   ├── keys.go          embedded APK constants (login parts, TOTP parts, package name, user-agent)
│   ├── obfuscator.go    KeyObfuscator XOR + lossy-UTF-8 replace; DeriveLoginSecret / DeriveTOTPMasterSecret
│   ├── totp.go          PBKDF2-HmacSHA1 + RFC 6238 HOTP → GenerateTOTP(salt, offsetMS)
│   └── signer.go        SignParams(path, []Param) → base64url(SHA-256), RandomSalt, NowMS
├── api/                 HTTP transport.
│   ├── client.go        Client + Do() — injects STTAuthorization + User-Agent, maps HTTP status → typed errors
│   ├── envelope.go      generic AskoResponse[T] + DecodeAsko[T]
│   ├── errors.go        *Error{Code, Message, Hint, HTTP, Exit} — ExitCode() drives os.Exit
│   └── endpoints/       per-resource wrappers — add a file here for each new resource
│       ├── session.go   Login (POST /login2), Logout, RemoteUserSession + Pretty()
│       └── user.go      Whoami, Settings, Follow, UserByName + Prettier impls
├── session/             session.go — XDG-aware persistence, 0600 perms, ErrNoSession
└── output/              the single render boundary
    ├── output.go        Render / RenderToFile, Opts, Prettier interface, resolveFormat
    └── tty.go           IsStdoutTTY (respects NO_COLOR)
```

Dependency direction is strict: **cmd → api(/endpoints) → auth**, and **cmd → output**, never the reverse. `internal/output` does not import any Suunto type. `internal/auth` does not import `net/http`.

## Patterns to follow

- **One verb per file in `cmd/`.** A new command = new file, registered via that file's `init()`.
- **Endpoints stay thin.** `internal/api/endpoints/*` files return typed structs that implement `output.Prettier`. Don't put business logic there. For list-shaped responses (workouts, comments, per-activity stats), `Pretty()` should render an aligned table via `renderTable(headers, rows)` in `endpoints/table.go`; single-record responses stay as key/value lines. Streaming endpoints (e.g. wellness sleep NDJSON) keep their TTY-only table renderers in `cmd/`.
- **Always go through `emit(v)` from `cmd/root.go:emit`.** It handles `--output` vs stdout and `--format auto|json|pretty`. Commands never call `json.Marshal` or `fmt.Println` for primary output. Status/log lines go to stderr.
- **Errors are typed.** Anything you return from `cmd/*` `RunE` that should set a non-zero exit code must be (or wrap) a `*api.Error`. `cmd.Execute` reads `ExitCode()` off it. Codes 3/4/5/6/7 are stable and documented in `--help`.
- **TDD with golden vectors for crypto.** Anything in `internal/auth` is locked to byte-identical output from the Python reference (`handoff/reference/secret_check.py`, `sign_check.py`, plus the inline TOTP capture script in `docs/superpowers/plans/2026-05-11-suuntool-v1.md` Task 3). Don't tweak the signer/TOTP without re-running those diagnostics first.
- **HTTP tests use `httptest.Server`.** No live API calls in CI. See `internal/api/client_test.go` and `cmd/login_test.go` for the pattern (env-vars `SUUNTOOL_BASE_URL` and `SUUNTOOL_SESSION_FILE` are the only knobs you need).
- **Cobra `Example:` block on every user-facing command.** Agents read help text — keep examples concrete and copy-pasteable.

## Don'ts

- **Never commit private data.** No real emails, usernames, sessionkeys, userKeys, passwords, workout IDs, GPS coordinates, or device serial numbers — not in tests, fixtures, README snippets, commit messages, or sample outputs. Use placeholders: `you@example.com`, `alice`, `SK123`, `k1`, `hunter2`. If you capture a real response for fixture work, scrub it before saving.
- **Don't commit `handoff/` or `docs/`.** They're local-only by design. Use targeted `git add <files>`; never `git add .` or `git add -A`.
- **Don't introduce another output path.** No `fmt.Println(json…)` in commands. If `emit` doesn't do what you need, extend `internal/output`, don't bypass it.
- **Don't touch git config.** Email + name are set per-repo already.
- **Don't add a feature-flag layer or "future-proof" abstraction "just in case".** Add the file when you add the endpoint.
- **Don't widen `internal/auth`'s public surface** without good reason. Helpers like `keyObfuscator`, `utf8Replace`, `pbkdf2KeyForSalt`, `hotp6` are unexported on purpose — promoting them invites parallel implementations.
- **Don't paper over a 401.** Surface it as `*api.Error{Code:"AUTH_EXPIRED", Exit:4}` with `Hint:"Run: suuntool login"`. No silent re-auth.
- **Don't run write-side smoke tests in CI.** Anything that touches `POST`/`PUT`/`DELETE` on Suunto's real backend must be exercised manually against a personal account (round-trip: do → verify → undo) and never from automation. CI runs `httptest`-backed unit tests only.
- **Don't compute x-totp inside endpoint wrappers.** The wrappers in `internal/api/endpoints/` stay session-agnostic; the cmd layer calls `totpHeaders(sess)` and passes the header in. Same rule for any header derived from session state.
- **Don't bypass `confirm()` on destructive commands.** `workouts delete` requires `--yes` on non-TTY. New destructive commands (future bulk-deletes, etc.) must follow the same pattern. The helper returns `*api.Error{Code:"USAGE", Exit:2}` when stdin is not a TTY and `--yes` is unset — propagate, don't catch.

## Key code to read first (in this order)

1. `internal/auth/obfuscator.go` — `DeriveLoginSecret`, `utf8Replace`. The whole signing scheme hinges on matching Java's lossy `new String(bytes, UTF-8)` semantics; the test in `obfuscator_test.go` proves it.
2. `internal/auth/signer.go` — `SignParams`. Build string verbatim, no URL-encoding, SHA-256, base64url no-padding. Same shape as `SessionRemoteApi.Companion.d()` in the APK.
3. `internal/auth/totp.go` — `GenerateTOTP`. Java PBEKeySpec quirk: only the low byte of each password char is used.
4. `internal/api/client.go` — `Do()` is the only place HTTP status codes are mapped to exit codes; if you're surprised by a CLI exit code, start here.
5. `internal/api/endpoints/session.go` — `Login`. Note that `/login2` returns the session at the top level, NOT in an AskoResponse envelope — every other endpoint does.
6. `cmd/root.go` (bottom half) — `baseURL`, `authedClient`, `renderOpts`, `emit`, `pickTimeout`. Every other command leans on these four.
7. `cmd/login_test.go` — the end-to-end pattern (env-var overrides + httptest + piped stdin password).

## Adding a new endpoint — the recipe

1. New file under `internal/api/endpoints/<resource>.go` with the typed struct + a `Pretty()` method + a function `Foo(ctx, *api.Client, args…) (*T, error)` that calls `c.Do` and `api.DecodeAsko[T]`.
2. New file under `cmd/<verb>.go` that calls `authedClient()` → endpoint → `emit(v)`.
3. Add a row to `endpointTable` in `cmd/endpoints.go` so agents discover it.
4. Test the endpoint wrapper with `httptest.Server` (mirror `endpoints/session_test.go`).
5. If the endpoint takes a `x-totp` header (reactions, comments, settings-safe, email/phone change), generate it with `auth.GenerateTOTP(session.Email, session.OffsetMS)` and pass via the `headers` map of `client.Do`.

## Signing-key rotation

Suunto rotates the embedded keys on major app version bumps. See `CONTRIBUTING.md` for the procedure — short version: re-decompile the APK with `jadx`, update the five constants in `internal/auth/keys.go`, regenerate goldens from the Python diagnostics, and update the `expected*` constants in `internal/auth/*_test.go`. The test suite is the canary.
