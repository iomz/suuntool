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
suuntool workouts list --limit 100 --summary        # totals over a window (km/time/ascent + per-activity)
suuntool workouts list --since 7d                   # last 7 days (also: 12h, 2w, last-week, 2026-01-01)
suuntool workouts get wk_abc123                     # one workout's metadata
suuntool workouts stats                             # aggregate stats for you
suuntool workouts count                             # workout count
suuntool workouts sml wk_abc123 -o wk.sml.json      # full ~5MB sample data
suuntool workouts fit wk_abc123 -o wk.fit           # binary .fit export

# 24/7 wellness (gzipped NDJSON, decoded on the fly)
suuntool wellness sleep                             # pretty table on TTY, raw NDJSON when piped
suuntool wellness sleep --since 7d -o sleep.ndjson  # last 7 days (also: 2026-01-01, last-week)
suuntool wellness activity   --since last-month | jq '.entryData.stepCount'
suuntool wellness recovery   --out ./wellness
suuntool wellness sleepstages --out ./wellness

# Workout interactions
suuntool workouts comments wk_abc123                # list comments
suuntool workouts comment wk_abc123 "great run"     # post a comment (x-totp)
suuntool workouts react wk_abc123                   # like (x-totp)
suuntool workouts edit wk_abc123 --set totalAscent=120
suuntool workouts share wk_abc123 --as gpx-track    # signed GPX URL
suuntool workouts extensions wk_abc123              # Fitness/Intensity/…
suuntool workouts upload --sml ./wk.sml             # multipart upload
suuntool workouts delete wk_abc123 --yes            # destructive — needs --yes off-TTY

suuntool doctor                                     # connectivity + session check
suuntool logout
```

## Output

`suuntool` picks a sensible default and gets out of the way:

| Flag | Effect |
|------|--------|
| `--format auto` (default) | Pretty on a TTY, JSON when piped or redirected |
| `--format json` | Force JSON (2-space indent) |
| `--format pretty` | Force pretty rendering — aligned tables for list responses (`workouts list`, `workouts stats`, `workouts comments`, `wellness sleep`); key/value blocks for single records |
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
| `workouts list` | `GET /v1/workouts` | yes | Paginated list with `--since`, `--limit`, `--offset`; returns cursor `until`. `--summary` collapses the window into totals (count, distance, time, ascent, descent, per-activity) |
| `workouts get <key>` | `GET /v1/workouts/{key}` | yes | One workout's metadata |
| `workouts stats [user]` | `GET /v1/workouts/{user}/stats` | yes | Aggregate totals + per-activity breakdown |
| `workouts count` | `GET /v1/workouts/count` | yes | Requires `until` + `sharingFlags` server-side (defaults handled) |
| `workouts sml <key>` | `GET /v1/workouts/{key}/sml` | yes | Full ~5MB sample-by-sample JSON. Raw passthrough — use `-o` |
| `workouts fit <key>` | `GET /v1/workout/exportFit/{key}` | yes | Binary `.fit`. Raw passthrough — use `-o` |

### Wellness (24/7 health timeline)

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `wellness sleep` | `GET 247.../v1/sleep/export` | yes | **Pretty table on TTY** (dedup'd by `sleepId`, units converted: Hz→BPM, fractions→%); raw NDJSON when piped or with `-o`/`--out` |
| `wellness activity` | `GET 247.../v1/activity/export` | yes | Per-15-min `hr`/`steps`/`energy`. **`hr` is in Hz** — ×60 for BPM |
| `wellness recovery` | `GET 247.../v1/recovery/export` | yes | `balance` ∈ 0..1 = "wake-up resources" |
| `wellness sleepstages` | `GET 247.../v1/sleepstages/export` | yes | Stage timeline (light/deep/REM/…) |

### Workout interactions & writes ⚠️

These mutate server state. The `delete` command requires `--yes` in non-TTY contexts.

| Command | Endpoint | Auth | Notes |
|---------|----------|------|-------|
| `workouts comments <key>` | `GET /v1/workouts/comments/{key}` | yes | List comments on a workout (note plural `comments/`) |
| `workouts comment <key> [text]` | `POST /v1/workouts/comment/{key}` | yes + **x-totp** | Post a comment; `--stdin` for multi-line |
| `workouts uncomment <comment-key>` | `DELETE /v1/workouts/comment/{commentKey}` | yes | Delete a comment by comment-key (NOT workout-key) |
| `workouts react <key>` | `POST /v1/workouts/reaction/{key}` | yes + **x-totp** | `--reaction like` (only supported value) |
| `workouts unreact <key>` | `DELETE /v1/workouts/reaction/{key}` | yes | Remove your reaction |
| `workouts edit <key> --set field=<json>` | `PUT /v1/workouts/{key}/attributes` | yes | Partial attribute update; values parsed as JSON literals |
| `workouts batch-update --file <json>` | `POST /v1/workouts/batchUpdate` | yes | Bulk updates from a JSON array |
| `workouts share <key> --as gpx-route\|gpx-track` | `PUT /v1/workouts/{user}/{key}/share/{format}` | yes | Returns a signed GPX URL |
| `workouts extensions <key>` | `POST /v1/workout/extensions/{key}` | yes | Despite POST, this is a fetch — body is the filter list |
| `workouts upload --sml <path> [--extensions <path>]` | `POST /v1/workout` (multipart) | yes | Upload a pre-built SML file. Streams via `io.Pipe`. **Does NOT generate SML from raw GPS** |
| `workouts delete <key>` | `DELETE /v1/workouts/{key}/delete` | yes | **Destructive.** TTY confirmation prompt; pass `--yes` in scripts/agents |

> **`x-totp` writes.** `comment` and `react` require a fresh 6-digit TOTP. `suuntool` auto-derives one from your session — you don't need to do anything. Codes rotate every 30s, so commands generate per call.
>
> **`--yes` and destructive operations.** `workouts delete` refuses to run on a non-TTY without `--yes` (exit code 2). This prevents agents and CI scripts from accidentally bypassing the confirmation by piping nothing into stdin.
>
> **Rate limiting.** Suunto's quotas are conservative (a few QPS). Don't batch-spam comments or reactions — your account can be flagged.

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
