# suuntool

> **Own your Suunto data. From the terminal. With one command.**

`suuntool` is a fast, scriptable, agent-friendly CLI for the Suunto / Sports-Tracker API — the same backend the Suunto mobile app talks to. Log in, pull your profile, pipe it into `jq`, automate it, hand it to an LLM. No browser, no app, no clicks.

```bash
$ suuntool login --email you@example.com
Password:
Logged in as alice (you@example.com). Session saved to ~/.config/suuntool/session.json.

$ suuntool whoami
username     : alice
email        : you@example.com
userKey      : k1
country      : FI

$ suuntool profile follow --format json
{
  "followers": 12,
  "followings": 8,
  "blocked": 0,
  "blockedBy": 0
}
```

## Why

- **Agent-ready.** Stable exit codes, JSON-on-pipe defaults, machine-readable error envelopes, and `suuntool endpoints --format json` so an LLM can map intents to commands without scraping help text.
- **Scriptable.** One static binary. No Python, no node, no app sandbox.
- **Honest.** Every endpoint is reverse-engineered from the shipping app and the signing logic is locked to golden vectors, so when the contract drifts, the tool fails loudly instead of pretending.

## Install

```bash
brew install tajchert/tap/suuntool
```

Or from source:

```bash
go install github.com/tajchert/suuntool@latest
```

## Quickstart

```bash
suuntool login --email you@example.com              # interactive password prompt
echo 'hunter2' | suuntool login \
   --email you@example.com --password-stdin         # non-interactive (CI, agents)

# Profile
suuntool whoami                                     # current session user
suuntool profile settings                           # full user settings DTO
suuntool profile follow                             # follower/following counts
suuntool profile user alice                         # look up any user by handle

# Workouts
suuntool workouts list --limit 5                    # most recent 5 workouts
suuntool workouts get wk_abc123                     # one workout's metadata
suuntool workouts stats                             # aggregate stats for you
suuntool workouts count                             # workout count
suuntool workouts sml wk_abc123 -o wk.sml.json      # full ~5MB sample data
suuntool workouts fit wk_abc123 -o wk.fit           # binary .fit export

# 24/7 wellness (gzipped NDJSON, decoded on the fly)
suuntool wellness sleep      --since 0 -o sleep.ndjson
suuntool wellness activity   --since 0 | jq '.entryData.stepCount'
suuntool wellness recovery   --out ./wellness
suuntool wellness sleepstages --out ./wellness

# Other reads
suuntool partner-connections                        # Strava/TrainingPeaks/… links
suuntool gear list                                  # paired gear
suuntool maps library --device-serial SN123         # offline-map regions

suuntool doctor                                     # connectivity + session check
suuntool logout
```

## Output

`suuntool` picks a sensible default and gets out of the way:

| Flag | Effect |
|------|--------|
| `--format auto` (default) | Pretty key/value on a TTY, JSON when piped or redirected |
| `--format json` | Force JSON (2-space indent) |
| `--format pretty` | Force pretty rendering |
| `-o, --output <path>` | Write to a file instead of stdout — format inferred from extension |
| `--no-color` | Disable ANSI styling (also honors `NO_COLOR`) |
| `--quiet`, `--verbose` | Tune log verbosity on stderr |
| `--timeout <dur>` | HTTP timeout, e.g. `45s` |

```bash
suuntool profile settings -o settings.json          # save to file
suuntool whoami --format json | jq .username        # pipe into jq
NO_COLOR=1 suuntool whoami                          # plain-text TTY
```

## Commands

### Session

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `login --email <e>` | `POST /v1/login2` | no | Reads password from a no-echo TTY prompt or `--password-stdin` |
| `logout` | `GET /v1/logout` | yes | Invalidates server-side, clears local session |
| `whoami` | `GET /v1/user` | yes | Current session user |
| `doctor` | `GET /v1/servertime` + session check | no | Probe connectivity and session validity |

### Profile

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `profile settings` | `GET /v1/user/settings` | yes | Full user settings DTO (passed through) |
| `profile follow` | `GET /v1/user/follow` | yes | Follower / following / blocked counts |
| `profile user <name>` | `GET /v1/user/name/{name}` | yes | Look up a user by username |

### Workouts

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `workouts list` | `GET /v1/workouts` | yes | Paginated list with `--since`, `--limit`, `--offset`; returns cursor `until` |
| `workouts get <key>` | `GET /v1/workouts/{key}` | yes | One workout's metadata |
| `workouts stats [user]` | `GET /v1/workouts/{user}/stats` | yes | Aggregate totals + per-activity breakdown |
| `workouts count` | `GET /v1/workouts/count` | yes | Requires `until` + `sharingFlags` server-side (defaults handled) |
| `workouts sml <key>` | `GET /v1/workouts/{key}/sml` | yes | Full ~5MB sample-by-sample JSON. Raw passthrough — use `-o` |
| `workouts fit <key>` | `GET /v1/workout/exportFit/{key}` | yes | Binary `.fit`. Raw passthrough — use `-o` |

### Wellness (24/7 health timeline)

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `wellness sleep` | `GET 247.../v1/sleep/export` | yes | NDJSON; `--since <ms>`, `--out <dir>` |
| `wellness activity` | `GET 247.../v1/activity/export` | yes | Per-15-min `hr`/`steps`/`energy`. **`hr` is in Hz** — ×60 for BPM |
| `wellness recovery` | `GET 247.../v1/recovery/export` | yes | `balance` ∈ 0..1 = "wake-up resources" |
| `wellness sleepstages` | `GET 247.../v1/sleepstages/export` | yes | Stage timeline (light/deep/REM/…) |

### Other reads

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `partner-connections` | `GET /v1/partnerconnection` | yes | Linked OAuth partners (Strava, TrainingPeaks, …) |
| `gear list` | `GET /v1/gear` | yes | Gear paired to the account |
| `maps library --device-serial <sn>` | `GET /v1/maps/library` | yes | Offline-map regions. Serial is `Source: "suunto-<sn>"` in `/sml` data |

### Discovery / meta

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `endpoints` | — | no | Stable command → method/path table for agents (`--format json`) |
| `version` | — | no | Build version |

Run `suuntool <command> --help` for the full reference of any command. The mapping above is also available as a JSON document via `suuntool endpoints --format json`.

> **Raw passthrough.** `workouts sml`, `workouts fit`, and the four `wellness` subcommands stream their response body straight to stdout (or `-o`) without going through the `--format` pretty-printer. The body is already in its final shape (large JSON / binary `.fit` / NDJSON) and reformatting it would waste memory and lose fidelity.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Generic error |
| 2 | Bad usage (flags/args) |
| 3 | Network (timeout, DNS, connection) |
| 4 | Auth — session missing or expired (run `suuntool login`) |
| 5 | Server error (5xx or malformed response) |
| 6 | Not found |
| 7 | Forbidden |

In `--format json` mode, errors are emitted to stderr as:

```json
{ "error": { "code": "AUTH_EXPIRED", "message": "...", "hint": "Run: suuntool login" } }
```

## Environment

| Variable | Purpose |
|----------|---------|
| `SUUNTOOL_SESSION_FILE` | Override session storage path (default `$XDG_CONFIG_HOME/suuntool/session.json`) |
| `SUUNTOOL_FORMAT` | Default output format |
| `SUUNTOOL_TIMEOUT` | Default HTTP timeout |
| `SUUNTOOL_BASE_URL` | Override API base URL (for testing) |
| `NO_COLOR` | Disable ANSI styling |

## Security & privacy

- Your session key is persisted to `$XDG_CONFIG_HOME/suuntool/session.json` with mode `0600`. Don't put it in your dotfiles repo.
- Passwords are read from a no-echo TTY prompt or stdin. `suuntool` never accepts `--password` on the command line.
- No telemetry, no analytics, no phone-home. The binary talks to Suunto's servers and nowhere else.

## ⚠️ Disclaimers

**This is an unofficial, experimental tool. Use at your own risk.**

- `suuntool` is **not affiliated with, endorsed by, or supported by Suunto Oy, Amer Sports, or Sports-Tracker**. All trademarks belong to their respective owners.
- The Suunto API is **not public**. This client is reverse-engineered from the shipping Android app and the wire contract can change without notice. A future app release may break this tool with no warning.
- Using this tool may **violate Suunto's Terms of Service**. Read them before you run anything. Heavy or abusive usage may get your account flagged or banned. The author accepts no responsibility for account actions taken against you.
- Provided **"as is", without warranty of any kind**, express or implied, including but not limited to merchantability, fitness for a particular purpose, and noninfringement. In no event shall the authors or copyright holders be liable for any claim, damages, or other liability arising from your use of the software.
- For **your own data only.** Do not use this tool to scrape, harvest, or aggregate other users' data — you'll get your account banned, and you may be breaking the law.
- The author is **not** a lawyer. Nothing here is legal advice.

If any of the above makes you uncomfortable, use the official Suunto app instead.

## License

MIT.
