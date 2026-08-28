# lmsctl — `info` command and expanded coloring

**Date:** 2026-08-28
**Status:** Approved

## Purpose

Two additions to `lmsctl`, the LM Studio management CLI:

1. `lmsctl info <model>` — a details view showing everything already returned
   by LM Studio's model-listing API for one model, but not currently
   displayed anywhere (publisher, architecture, quantization bits-per-weight,
   max context length, format, and — for loaded instances — the actual
   config it was loaded with).
2. Expanded terminal coloring across all commands, replacing the current
   narrow scope (status/state words and success confirmations only) with
   color on nearly every element: table columns, labels, identifiers, and
   errors.

This is a follow-up to the completed v1 implementation
(`docs/superpowers/plans/2026-08-28-lmsctl-implementation.md`) and to the
first coloring pass (which added `internal/color` with `Green`/`Dim`/`Bold`,
scoped to status/state words and success confirmations only).

## Scope

**In scope:**
- `lmsctl info <model>` command, with `--json` support.
- New `Cyan`, `Yellow`, `Red` colors added to `internal/color.Palette`.
- `models` table stops using `text/tabwriter` in favor of manual,
  color-safe column alignment, so KEY/SIZE/QUANTIZATION can be colored
  (previously only the last, tab-unterminated STATE column was safe to
  color under `tabwriter`).
- Color extended to labels ("Loaded models:", "host:"/"token:"), model
  keys/hosts/instance IDs, and error output (`Error:` prefix + message, in
  `cmd/root.go`'s `Execute()`).

**Explicitly out of scope** (separate design to follow):
- Live/interactive tokens-per-second session (start a session, send one or
  more prompts, watch live throughput, stop it, get a summary). This needs
  its own design — it requires LM Studio's chat/completion endpoint,
  streaming, an interactive REPL, and signal handling, none of which exist
  in `lmsctl` today.

## `lmsctl info <model>`

**Files:**
- `cmd/info.go` (new)

Same shape as every other command: `infoCmd` (`Use: "info <model>"`,
`Args: cobra.ExactArgs(1)`, a `--json` local flag) with a thin `RunE` that
resolves the client via `newClient()` and delegates to a testable
`runInfo(cmd, client, model, jsonOut)`.

**Data source:** `runInfo` calls `client.ListModels(cmd.Context())` — the
same call every other command already makes. No new API endpoint is
introduced. It then searches `resp.Models` for a `Model` whose `Key`
matches the given argument.

**Not found:** returns `&lmstudio.ErrModelNotFound{Model: model}` — the
same typed error `load`/`unload` already return for an unrecognized model
key, so the message and `errors.As` behavior are already established and
tested elsewhere.

**Found, `--json`:** `output.JSON(cmd.OutOrStdout(), matchedModel)` — dumps
the matched `lmstudio.Model` struct directly, the same convention
`models --json` uses for the full list.

**Found, human-readable:** prints the static fields that are non-nil
(`Model.Architecture`, `Format`, `ParamsString` are all `*string` and may
be nil — printed as `-` when nil, matching the `models` table's existing
convention for a nil `Quantization`), then either:
- `Not currently loaded.` (dim), if `len(LoadedInstances) == 0`, or
- One block per entry in `LoadedInstances`, showing that instance's `ID`
  and its `InstanceConfig` fields (`ContextLength`, `FlashAttention`,
  `OffloadKVCacheToGPU`, `Parallel`, `EvalBatchSize` — `NumExperts` is
  omitted from display when it's `0`, since that's the zero-value for a
  non-MoE model and printing "Num experts: 0" for every dense model would
  be noise).

Example output:

```
$ lmsctl info openai/gpt-oss-20b
Key:           openai/gpt-oss-20b
Publisher:     openai
Display name:  GPT OSS 20B
Architecture:  llama
Format:        gguf
Quantization:  MXFP4 (4.25 bits/weight)
Size:          11.3GiB
Max context:   131072

Loaded instances:
  Instance:            inst-1
  Context length:      16384
  Flash attention:     true
  Offload KV to GPU:   false
  Parallel:            1
  Eval batch size:     512
```

## Expanded coloring

### New palette colors

`internal/color/color.go` gains three more `Palette` methods, following the
exact pattern of the existing `Green`/`Dim`/`Bold`:

```go
func (p Palette) Cyan(s string) string   { return p.wrap("\033[36m", s) }
func (p Palette) Yellow(s string) string { return p.wrap("\033[33m", s) }
func (p Palette) Red(s string) string    { return p.wrap("\033[31m", s) }
```

Color meaning, applied consistently across every command:
- **Cyan** — identifiers the user would reference elsewhere: model keys,
  instance IDs, hosts.
- **Yellow** — secondary/quantitative values: sizes, quantization names.
- **Green** — positive/active state (`reachable`, `loaded`, success verbs)
  — unchanged from the first coloring pass.
- **Dim** — inactive/neutral state (`not-loaded`, "nothing to do" messages)
  — unchanged.
- **Bold** — section labels and table headers — unchanged in meaning, now
  applied more widely.
- **Red** — errors.

### `models` table: manual alignment instead of `tabwriter`

The current implementation writes rows through `output.NewTable`
(`text/tabwriter`), which only ever let the *last* column (STATE) be
colored safely — `tabwriter` computes column width from the raw byte
length of what's written into it, and doesn't know ANSI escape sequences
are invisible, so coloring an earlier column would corrupt every later
column's alignment.

`cmd/models.go` will compute each column's width from the **plain** cell
text first (`len()` on the uncolored string), pad using that plain width,
and only wrap the already-correctly-positioned text in color afterward.
Concretely, a small helper:

```go
// PadRight returns colored, right-padded with spaces so the total VISIBLE
// width (measured from plain, before any color codes were added) is at
// least width. Safe to use with ANSI-colored strings since padding is
// computed from the pre-color length, not len(colored).
func PadRight(plain, colored string, width int) string {
	if n := width - len(plain); n > 0 {
		return colored + strings.Repeat(" ", n)
	}
	return colored
}
```

This is a general-purpose alignment helper, not tied to any one command, so
it belongs in `internal/output` (alongside `JSON`/`NewTable`) rather than
`cmd/models.go`, as `internal/output.PadRight`. `NewTable`/`text/tabwriter`
remains available for any future command that needs simple alignment
without color (nothing currently uses `NewTable` besides `models`, but
removing it isn't part of this change).

The header row becomes bold; KEY cyan; SIZE yellow; QUANTIZATION dim;
STATE unchanged (green `loaded` / dim `not-loaded`). Two literal spaces
separate columns (matching `tabwriter`'s previous `padding: 2` setting, so
the visual spacing is unchanged).

### Labels and errors

- `status`: `"LM Studio at "` label plain, host cyan, `": reachable"` with
  `reachable` green (unchanged). `"Loaded models:"` bold. Bullet lines:
  model key cyan, instance ID cyan.
- `load`: `"Loaded"` green (unchanged), model key cyan, instance ID cyan.
- `unload`: `"Unloaded"` green (unchanged) / `"Instance ... already
  unloaded"` dim (unchanged), instance ID cyan in both.
- `config show`: `"host:"`/`"token:"` labels bold, the host value cyan
  (token stays as the literal `(set)`/`(not set)` marker, never colored
  differently, since coloring it could be read as implying something about
  the token's validity, which `lmsctl` has no way to know).
- `config set-host`: `"Default host set to"` green (unchanged), host value
  cyan.
- `info`: field labels (`"Key:"`, `"Publisher:"`, etc.) bold, values cyan
  where they're identifiers (key, instance ID), yellow where they're
  quantitative (size, quantization, context length), plain otherwise
  (architecture, format, publisher — free-text values with no clear color
  role).
- **Errors:** `cmd/root.go`'s `Execute()` changes from
  `fmt.Fprintln(os.Stderr, "Error:", err)` to coloring the `"Error:"`
  prefix and the error text red. Errors go to stderr, and stderr can be
  redirected independently of stdout (`lmsctl status 2>&1 | less` still
  wants color on the piped-through terminal, but `lmsctl status
  2>err.log` should not write ANSI codes into the log file) — so this
  needs its own palette checked against `os.Stderr`, not the existing
  `palette` var (which is bound to `os.Stdout`). `cmd/root.go` gains a
  second package-level var: `errPalette = color.New(os.Stderr)`, used
  only inside `Execute()`.

## Testing

- `runInfo`: table-driven tests via `lmstudiotest.Fake`, covering
  found-and-loaded, found-and-not-loaded, not-found (`ErrModelNotFound`),
  and `--json`, following the exact pattern of `runModels`'s tests.
- `internal/output.PadRight`: unit tests confirming padding is computed
  from the plain string's length regardless of what's passed as `colored`
  (i.e. that passing a color-wrapped string doesn't change the padding
  amount), plus a case where `plain` is already `>= width` (no padding
  added).
- `internal/color.Palette`: extend the existing table-driven color tests
  (`TestPalette_EnabledWrapsWithAnsiCodesAndReset`,
  `TestPalette_DisabledReturnsPlainText`) with cases for `Cyan`/`Yellow`/
  `Red`.
- A manual pty-based check (as done for the first coloring pass) verifying
  the `models` table stays aligned with all four columns colored.
