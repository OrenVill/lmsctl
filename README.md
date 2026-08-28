# lmsctl

A CLI for managing an LM Studio instance running on another machine on your
local network — check status, list models, view a model's details, and
load/unload them — over LM Studio's `/api/v1` REST API.

## Setup

On the remote machine, in LM Studio's Developer settings, enable
**"Serve on Local Network"** (and set an API token there if you want
authenticated access).

## Install

```bash
go build -o lmsctl .
```

Move the resulting `lmsctl` binary onto your `$PATH`.

## Configure

```bash
lmsctl config set-host 192.168.1.50:1234
```

This is stored in `~/.config/lmsctl/config.yaml`. Override it per-command
with `--host`, or set `LMSCTL_HOST` in your shell profile. If the remote
server requires a token, set it the same way with `--token` /
`LMSCTL_TOKEN`, or add it directly to the config file's `token:` field.

## Usage

```bash
lmsctl status                          # is it up, what's loaded
lmsctl models                          # list downloaded models (alias: ls)
lmsctl info openai/gpt-oss-20b         # full details for one model
lmsctl load openai/gpt-oss-20b         # load a model
lmsctl load openai/gpt-oss-20b --context-length 16384 --flash-attention
lmsctl unload openai/gpt-oss-20b       # unload one model
lmsctl unload --all                    # unload everything loaded
lmsctl config show                     # see effective config (token redacted)
```

Add `--json` to `status`, `models`, `info`, or `load` for machine-readable
output. (`unload` and `config` are plain-text only — there's nothing in
their output that benefits from a JSON form.)

Output is colored automatically when connected to a terminal, and disabled
automatically when piped/redirected or when the `NO_COLOR` environment
variable is set.

## Notes

- `status` uses `GET /api/v1/models` as its reachability check — LM Studio
  has no dedicated health endpoint.
- `unload` resolves the model key you give it to its currently loaded
  instance ID(s) via a fresh model list, then unloads each one. If an
  instance turns out to already be gone by the time the unload request
  lands (e.g. LM Studio's own idle eviction, or unloading it yourself in
  the LM Studio UI), `lmsctl` treats that as success rather than an error.
- Every error message is meant to be actionable on its own — if something
  fails, read it before assuming `lmsctl` is broken.
