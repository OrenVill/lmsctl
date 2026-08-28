# lmsctl — LM Studio remote management CLI

**Date:** 2026-08-28
**Status:** Approved

## Purpose

A personal command-line tool to manage an LM Studio instance running on a
different device on the local network: check whether it's up, see what
models are available and loaded, and load/unload models — without opening
the LM Studio desktop app or SSHing in.

## Scope

**In scope (v1):**
- Model lifecycle: list downloaded models, load a model, unload a model.
- Status/monitoring: server reachability, LM Studio version (if exposed),
  which model(s) are currently loaded.

**Explicitly out of scope for v1** (may be added later without restructuring):
- Sending chat/completion prompts from the CLI.
- Searching or triggering downloads from the model catalog.

## Connection method

Talks directly to LM Studio's native `/api/v1` REST API (LM Studio 0.4.0+)
over HTTP, e.g. `http://<host>:1234`. No SSH, no agent installed on the
remote box — the only remote-side requirement is that LM Studio has
"Serve on Local Network" enabled (and, if the user sets one, an API token
configured in LM Studio matching the one `lmsctl` sends).

Relevant endpoints (confirmed against LM Studio's developer docs,
2026-08-28):
- `GET /api/v1/models` — list models (downloaded + loaded state).
- `POST /api/v1/models/load` — load a model, with tuning params such as
  context length / GPU offload.
- `POST /api/v1/models/unload` — unload a model instance
  (`{"instance_id": "..."}`).
- Auth: optional `Authorization: Bearer <token>` header.

Exact request/response field names beyond the above will be confirmed
against a live server (or the fuller docs) during implementation, since the
public docs excerpt didn't give the complete `load`/`list` schemas.

## Language & tooling

- **Go**, built as a single static binary (`go build`), no runtime deps to
  install elsewhere.
- CLI framework: `spf13/cobra` for command/flag structure.
- Config file: YAML, parsed with `gopkg.in/yaml.v3`.

## Architecture

- `internal/lmstudio` package: a small Go interface (e.g. `Client`) wrapping
  the LM Studio REST API — `Status()`, `ListModels()`, `LoadModel(...)`,
  `UnloadModel(...)`. All HTTP/JSON details live here.
- `cmd/` (cobra commands): each subcommand calls the `lmstudio.Client`
  interface, never `net/http` directly. This keeps commands unit-testable
  against a fake client and isolates them from API changes.
- `internal/config` package: loads/saves `~/.config/lmsctl/config.yaml`,
  resolves the effective host/token from flag > env var > config file.

## Commands

| Command | Description |
|---|---|
| `lmsctl status` | Reachability check; reports LM Studio version (if available) and currently loaded model(s). |
| `lmsctl models` (alias `ls`) | Lists downloaded models: name, size, quantization, loaded/not-loaded. |
| `lmsctl load <model>` | Loads a model. Flags: `--context-length`, `--gpu-offload`. |
| `lmsctl unload <model>` | Unloads a model. Flag: `--all` to unload everything loaded. |
| `lmsctl config set-host <host:port>` | Writes the default remote host to the config file. |
| `lmsctl config show` | Prints the effective config (token redacted). |

Global flags on every command: `--host`, `--token`, `--json` (machine-readable
output instead of a human table).

## Config resolution order

1. Command-line flag (`--host`, `--token`)
2. Environment variable (`LMSCTL_HOST`, `LMSCTL_TOKEN`)
3. Config file (`~/.config/lmsctl/config.yaml`)
4. No built-in default — if nothing is set, `lmsctl` tells the user to run
   `lmsctl config set-host <host:port>` rather than silently trying
   `localhost`.

The token is never printed in `--json` output, logs, or `config show`.

## Error handling

Distinct, actionable error messages (no raw Go stack traces) for:
- **Connection refused / timeout** — likely LM Studio isn't running or
  "Serve on Local Network" isn't enabled on the remote box; message says so.
- **401 Unauthorized** — bad or missing token; points at `--token` /
  `LMSCTL_TOKEN` / `config set-host`.
- **404 / model not found** — lists the closest known model names if
  feasible, otherwise says to check `lmsctl models`.

## Output format

Default: human-readable tables/text. `--json` flag on every command switches
to structured JSON for scripting.

## Testing approach

- Unit tests against the `lmstudio.Client` interface using a hand-written
  fake (table-driven tests), covering command logic without real network
  calls.
- `httptest`-backed tests in `internal/lmstudio` verifying actual HTTP
  request shaping (method, path, headers, body) and response parsing.
- Manual smoke test against the user's real remote LM Studio instance, since
  that's the real integration target and the exact API field names need
  confirming live.

## Open items for the implementation plan

- Confirm exact JSON field names for `POST /api/v1/models/load` request and
  the `GET /api/v1/models` response by checking a live LM Studio server (or
  the full docs pages not captured in this design pass).
