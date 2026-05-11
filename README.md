# suuntool

Unofficial Go CLI for the Suunto / Sports-Tracker API. Designed to be used by
humans **and** by LLM agents driving the terminal.

> ⚠️ The Suunto API is not public. This client is reverse-engineered from the
> shipping Android APK and the wire contract is fragile. See `handoff/` (local
> reference materials) for the full API surface.

## Install

```bash
go install github.com/tajchert/suuntool@latest
```

## Quickstart

```bash
suuntool login --email you@example.com         # prompts for password
suuntool whoami
suuntool profile settings
suuntool profile follow
suuntool profile user michal --format json
suuntool profile user michal -o user.json
suuntool logout
```

## Output

- `--format auto` (default): pretty on TTY, JSON otherwise.
- `--format json` / `--format pretty`: force a mode.
- `-o, --output <path>`: write to file; format inferred from extension.
- Honors `NO_COLOR`.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | OK |
| 1 | Generic error |
| 2 | Bad usage |
| 3 | Network |
| 4 | Auth (run `suuntool login`) |
| 5 | Server error |
| 6 | Not found |
| 7 | Forbidden |

## Environment

| Var | Purpose |
|-----|---------|
| `SUUNTOOL_SESSION_FILE` | Override session storage path |
| `SUUNTOOL_FORMAT` | Default output format |
| `SUUNTOOL_TIMEOUT` | Default HTTP timeout |
| `SUUNTOOL_BASE_URL` | Override API base URL (for tests) |
| `NO_COLOR` | Disable ANSI styling |

## For agents

`suuntool endpoints --format json` prints a stable `command → method+path`
table — use it to map intents to commands without scraping help text.
