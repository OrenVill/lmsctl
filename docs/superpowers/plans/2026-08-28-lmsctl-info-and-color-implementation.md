# lmsctl info command and expanded coloring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `lmsctl info <model>` (details view for one model) and extend terminal coloring to nearly every command's output, including the `models` table's other columns and error messages.

**Architecture:** Three new `Palette` colors (`Cyan`/`Yellow`/`Red`) and a `PadRight` alignment helper (in `internal/output`) let commands compose colored, correctly-aligned output without `text/tabwriter` (which can't be combined with ANSI color safely — it measures raw byte length, not visible width). `cmd/models.go`'s table switches from `tabwriter` to manual `PadRight`-based alignment. `cmd/info.go` is a new command following the same `RunE` → testable `runXxx` pattern as every other command, using data already returned by `ListModels()` — no new API endpoint.

**Tech Stack:** Go, existing `internal/color`/`internal/output`/`internal/lmstudio` packages, no new dependencies.

---

### Task 1: Extend `internal/color` with Cyan, Yellow, Red

**Files:**
- Modify: `internal/color/color.go`
- Modify: `internal/color/color_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/color/color_test.go`:

```go
func TestPalette_EnabledWrapsCyanYellowRed(t *testing.T) {
	p := Palette{Enabled: true}
	if got, want := p.Cyan("x"), "\033[36mx\033[0m"; got != want {
		t.Errorf("Cyan(%q) = %q, want %q", "x", got, want)
	}
	if got, want := p.Yellow("x"), "\033[33mx\033[0m"; got != want {
		t.Errorf("Yellow(%q) = %q, want %q", "x", got, want)
	}
	if got, want := p.Red("x"), "\033[31mx\033[0m"; got != want {
		t.Errorf("Red(%q) = %q, want %q", "x", got, want)
	}
}

func TestPalette_DisabledReturnsPlainTextForCyanYellowRed(t *testing.T) {
	p := Palette{Enabled: false}
	if got := p.Cyan("x"); got != "x" {
		t.Errorf("Cyan(%q) = %q, want unmodified %q", "x", got, "x")
	}
	if got := p.Yellow("x"); got != "x" {
		t.Errorf("Yellow(%q) = %q, want unmodified %q", "x", got, "x")
	}
	if got := p.Red("x"); got != "x" {
		t.Errorf("Red(%q) = %q, want unmodified %q", "x", got, "x")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/color/... -v`
Expected: FAIL — `p.Cyan`/`p.Yellow`/`p.Red` undefined (compile error).

- [ ] **Step 3: Add the three methods**

Append to `internal/color/color.go`, after the existing `Bold` method:

```go
// Cyan styles s as an identifier the user would reference elsewhere (a
// model key, instance ID, or host).
func (p Palette) Cyan(s string) string { return p.wrap("\033[36m", s) }

// Yellow styles s as a secondary/quantitative value (a size, a
// quantization name).
func (p Palette) Yellow(s string) string { return p.wrap("\033[33m", s) }

// Red styles s as an error.
func (p Palette) Red(s string) string { return p.wrap("\033[31m", s) }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/color/... -v`
Expected: PASS (all 6 tests: the 4 from before plus these 2)

- [ ] **Step 5: Commit**

```bash
git add internal/color/color.go internal/color/color_test.go
git commit -m "Add Cyan/Yellow/Red to internal/color.Palette"
```

---

### Task 2: Add `internal/output.PadRight`

**Files:**
- Modify: `internal/output/output.go`
- Modify: `internal/output/output_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `internal/output/output_test.go`:

`PadRight` takes the plain string plus a *styling function* (e.g.
`palette.Cyan`, which has exactly the right `func(string) string` shape as
a method value) rather than taking the plain text and the already-colored
text as two separate strings. The original two-string design was reviewed
and rejected: it requires every caller to keep two arguments in sync by
hand (`PadRight(r.key, palette.Cyan(r.key), keyW)`), and a mismatch (e.g.
accidentally styling the wrong field) is invisible in tests, since `go
test` always runs with color disabled — the bug would only be visible to a
real user with color enabled, as a few misplaced spaces. Taking a styling
function instead makes that class of mistake impossible: there's only one
string to get right, and the styling is applied to it directly.

```go
func TestPadRight_PadsToWidthAfterApplyingStyle(t *testing.T) {
	got := PadRight("ab", 5, func(s string) string { return "\033[36m" + s + "\033[0m" })
	want := "\033[36mab\033[0m   "
	if got != want {
		t.Errorf("PadRight = %q, want %q", got, want)
	}
}

func TestPadRight_NoPaddingWhenAlreadyAtOrOverWidth(t *testing.T) {
	got := PadRight("hello", 3, func(s string) string { return s })
	if got != "hello" {
		t.Errorf("PadRight = %q, want %q (no padding, and no truncation)", got, "hello")
	}
}

func TestPadRight_ZeroAndNegativeWidthReturnStyledUnpadded(t *testing.T) {
	for _, width := range []int{0, -3} {
		got := PadRight("ab", width, func(s string) string { return s })
		if got != "ab" {
			t.Errorf("PadRight(%q, %d, ...) = %q, want %q", "ab", width, got, "ab")
		}
	}
}

func TestPadRight_EmptyStringPadsToFullWidth(t *testing.T) {
	got := PadRight("", 5, func(s string) string { return s })
	if got != "     " {
		t.Errorf("PadRight = %q, want 5 spaces", got)
	}
}

func TestPadRight_IdentityStyleIsANoOp(t *testing.T) {
	got := PadRight("KEY", 6, func(s string) string { return s })
	if got != "KEY   " {
		t.Errorf("PadRight = %q, want %q", got, "KEY   ")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/output/... -v`
Expected: FAIL — `PadRight` undefined (compile error).

- [ ] **Step 3: Add `PadRight`**

`internal/output/output.go` needs a new import. The current top of the file is:

```go
// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"encoding/json"
	"io"
	"text/tabwriter"
)
```

Change the import block to add `"strings"`:

```go
// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"encoding/json"
	"io"
	"strings"
	"text/tabwriter"
)
```

Append this function at the end of the file:

```go
// PadRight applies style to s and right-pads with spaces so the total
// VISIBLE width -- measured from s's byte length before styling -- is at
// least width. Assumes single-width characters (true for lmsctl's ASCII
// model keys, sizes, and quantization names); a multi-byte value would
// under-pad rather than corrupt anything. Callers computing column widths
// must also use len() on the plain string, to match.
func PadRight(s string, width int, style func(string) string) string {
	styled := style(s)
	if n := width - len(s); n > 0 {
		return styled + strings.Repeat(" ", n)
	}
	return styled
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/output/... -v`
Expected: PASS (all 7 tests: the 2 from before plus these 5)

- [ ] **Step 5: Commit**

```bash
git add internal/output/output.go internal/output/output_test.go
git commit -m "Add output.PadRight for color-safe manual column alignment"
```

---

### Task 3: Replace `models` table's `tabwriter` with colored manual alignment

**Files:**
- Modify: `cmd/models.go`
- Modify: `cmd/models_test.go`

Current `cmd/models.go` (for reference — you're replacing the `runModels` function body and its imports, `formatBytes` and `init()` stay as-is):

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"ls"},
	Short:   "List downloaded models and whether each is loaded",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runModels(cmd, client, jsonOut)
	},
}

func runModels(cmd *cobra.Command, client lmstudio.Client, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOut {
		models := resp.Models
		if models == nil {
			models = []lmstudio.Model{}
		}
		return output.JSON(cmd.OutOrStdout(), models)
	}

	tw := output.NewTable(cmd.OutOrStdout())
	fmt.Fprintln(tw, "KEY\tSIZE\tQUANTIZATION\tSTATE")
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		state := "not-loaded"
		if len(m.LoadedInstances) > 0 {
			state = "loaded"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.Key, formatBytes(m.SizeBytes), quant, state)
	}
	return tw.Flush()
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	modelsCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(modelsCmd)
}
```

- [ ] **Step 1: Write the failing test**

`cmd/models_test.go`'s existing `TestRunModels_TableOutputShowsStateAndSize` doesn't check for the header row. Add that assertion — find this test and add the new `if` block right after the existing two:

```go
func TestRunModels_TableOutputShowsStateAndSize(t *testing.T) {
	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{
				Key:             "openai/gpt-oss-20b",
				SizeBytes:       12884901888, // 12 GiB
				Quantization:    &lmstudio.Quantization{Name: "Q4_K_M"},
				LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}},
			},
			{
				Key:       "embed/model",
				SizeBytes: 500,
			},
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, false); err != nil {
		t.Fatalf("runModels: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "KEY") || !strings.Contains(got, "SIZE") || !strings.Contains(got, "QUANTIZATION") || !strings.Contains(got, "STATE") {
		t.Errorf("output missing expected header row: %q", got)
	}
	if !strings.Contains(got, "openai/gpt-oss-20b") || !strings.Contains(got, "12.0GiB") || !strings.Contains(got, "loaded") {
		t.Errorf("output missing expected loaded model row: %q", got)
	}
	if !strings.Contains(got, "embed/model") || !strings.Contains(got, "not-loaded") {
		t.Errorf("output missing expected not-loaded model row: %q", got)
	}
}
```

(Only the new header-row `if` block was added; the rest of the test function is unchanged from what's already in the file.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunModels_TableOutputShowsStateAndSize' -v`
Expected: this specific test should still PASS even before Step 3's changes (the header row already exists via `tabwriter`) — this step is really about confirming your understanding of the current output, not a red/green TDD cycle. Skip straight to Step 3.

- [ ] **Step 3: Replace `runModels` and the imports**

Replace the full contents of `cmd/models.go` with:

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var modelsCmd = &cobra.Command{
	Use:     "models",
	Aliases: []string{"ls"},
	Short:   "List downloaded models and whether each is loaded",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runModels(cmd, client, jsonOut)
	},
}

type modelRow struct {
	key, size, quant, state string
	loaded                  bool
}

func runModels(cmd *cobra.Command, client lmstudio.Client, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	if jsonOut {
		models := resp.Models
		if models == nil {
			models = []lmstudio.Model{}
		}
		return output.JSON(cmd.OutOrStdout(), models)
	}

	rows := make([]modelRow, 0, len(resp.Models))
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		loaded := len(m.LoadedInstances) > 0
		state := "not-loaded"
		if loaded {
			state = "loaded"
		}
		rows = append(rows, modelRow{key: m.Key, size: formatBytes(m.SizeBytes), quant: quant, state: state, loaded: loaded})
	}

	const headerKey, headerSize, headerQuant, headerState = "KEY", "SIZE", "QUANTIZATION", "STATE"
	keyW, sizeW, quantW := len(headerKey), len(headerSize), len(headerQuant)
	for _, r := range rows {
		if len(r.key) > keyW {
			keyW = len(r.key)
		}
		if len(r.size) > sizeW {
			sizeW = len(r.size)
		}
		if len(r.quant) > quantW {
			quantW = len(r.quant)
		}
	}

	w := cmd.OutOrStdout()
	header := output.PadRight(headerKey, keyW, palette.Bold) + "  " +
		output.PadRight(headerSize, sizeW, palette.Bold) + "  " +
		output.PadRight(headerQuant, quantW, palette.Bold) + "  " +
		palette.Bold(headerState)
	fmt.Fprintln(w, header)

	for _, r := range rows {
		stateStyle := palette.Dim
		if r.loaded {
			stateStyle = palette.Green
		}
		line := output.PadRight(r.key, keyW, palette.Cyan) + "  " +
			output.PadRight(r.size, sizeW, palette.Yellow) + "  " +
			output.PadRight(r.quant, quantW, palette.Dim) + "  " +
			stateStyle(r.state)
		fmt.Fprintln(w, line)
	}
	return nil
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	modelsCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(modelsCmd)
}
```

Note `output.NewTable` (`text/tabwriter`) is no longer called anywhere, but stays exported in `internal/output` for any future command that needs plain alignment without color — don't delete it.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunModels' -v`
Expected: PASS (all `TestRunModels_*` tests, including the strengthened header-row check)

Then run the full `cmd` package suite to confirm nothing else regressed:

Run: `go test ./cmd/... -v`
Expected: PASS (all tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/models.go cmd/models_test.go
git commit -m "Replace models table's tabwriter with color-safe manual alignment"
```

**Follow-up fix (after code review):** the review found the entire point of
this task — that color doesn't break alignment — had no automated coverage.
`go test` always runs with `palette.Enabled == false` (stdout is a pipe, not
a terminal, under the test runner), so the ANSI-vs-`tabwriter` regression
this task exists to prevent was structurally uncatchable by the suite. It
also flagged `modelRow`'s `state string` field as redundant with `loaded
bool` (both derived from the same check, stored separately, and `state`
never actually participates in the width computation) — a latent
desync risk for a future edit, worth removing while the file is open.

Simplify `modelRow` in `cmd/models.go` (remove the `state` field):

```go
type modelRow struct {
	key, size, quant string
	loaded           bool
}
```

Update the row-building loop to stop setting a `state` field:

```go
	rows := make([]modelRow, 0, len(resp.Models))
	for _, m := range resp.Models {
		quant := "-"
		if m.Quantization != nil {
			quant = m.Quantization.Name
		}
		rows = append(rows, modelRow{key: m.Key, size: formatBytes(m.SizeBytes), quant: quant, loaded: len(m.LoadedInstances) > 0})
	}
```

Update the row-rendering loop to compute `state` inline instead of reading `r.state`:

```go
	for _, r := range rows {
		state, stateStyle := "not-loaded", palette.Dim
		if r.loaded {
			state, stateStyle = "loaded", palette.Green
		}
		line := output.PadRight(r.key, keyW, palette.Cyan) + "  " +
			output.PadRight(r.size, sizeW, palette.Yellow) + "  " +
			output.PadRight(r.quant, quantW, palette.Dim) + "  " +
			stateStyle(state)
		fmt.Fprintln(w, line)
	}
```

Add a real alignment-under-color test to `cmd/models_test.go`. It temporarily
forces `palette.Enabled = true` (restoring it afterward), strips ANSI codes
with a regexp, and asserts that the SIZE and QUANTIZATION columns start at
the same character offset on the header row and both data rows — using
deliberately varying-length values so both the "row widens a column past
the header" and "header stays the widest" cases are exercised in one test.
It also asserts the raw output actually contains an ANSI escape sequence
first, so the test can't pass vacuously if the `palette` mutation silently
didn't take effect:

```go
func TestRunModels_ColumnsStayAlignedWithColorEnabled(t *testing.T) {
	old := palette
	palette = color.Palette{Enabled: true}
	defer func() { palette = old }()

	fake := &lmstudiotest.Fake{
		ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
			{
				Key:             "openai/gpt-oss-20b", // longer than the "KEY" header
				SizeBytes:       12884901888,           // "12.0GiB", longer than "SIZE" header
				Quantization:    &lmstudio.Quantization{Name: "Q4_K_M"},
				LoadedInstances: []lmstudio.LoadedInstance{{ID: "inst-1"}},
			},
			{Key: "b", SizeBytes: 500}, // shorter than the header on every column
		}},
	}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runModels(cmd, fake, false); err != nil {
		t.Fatalf("runModels: %v", err)
	}

	raw := out.String()
	if !strings.Contains(raw, "\x1b[") {
		t.Fatalf("output has no ANSI escape codes; palette.Enabled wasn't honored, so this test can't prove anything: %q", raw)
	}

	ansi := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3 (header + 2 rows): %q", len(lines), raw)
	}
	plainHeader := ansi.ReplaceAllString(lines[0], "")
	plainRow1 := ansi.ReplaceAllString(lines[1], "")
	plainRow2 := ansi.ReplaceAllString(lines[2], "")

	sizeIdx := strings.Index(plainHeader, "SIZE")
	if idx := strings.Index(plainRow1, "12.0GiB"); idx != sizeIdx {
		t.Errorf("SIZE column misaligned: header at %d, row1 (\"12.0GiB\") at %d\nheader: %q\nrow1:   %q", sizeIdx, idx, plainHeader, plainRow1)
	}
	if idx := strings.Index(plainRow2, "500B"); idx != sizeIdx {
		t.Errorf("SIZE column misaligned: header at %d, row2 (\"500B\") at %d\nheader: %q\nrow2:   %q", sizeIdx, idx, plainHeader, plainRow2)
	}

	quantIdx := strings.Index(plainHeader, "QUANTIZATION")
	if idx := strings.Index(plainRow1, "Q4_K_M"); idx != quantIdx {
		t.Errorf("QUANTIZATION column misaligned: header at %d, row1 at %d\nheader: %q\nrow1:   %q", quantIdx, idx, plainHeader, plainRow1)
	}
	if idx := strings.Index(plainRow2, "-"); idx != quantIdx {
		t.Errorf("QUANTIZATION column misaligned: header at %d, row2 (\"-\") at %d\nheader: %q\nrow2:   %q", quantIdx, idx, plainHeader, plainRow2)
	}
}
```

This needs two new imports in `cmd/models_test.go`: `"regexp"` and
`"lmsctl/internal/color"` (for the `color.Palette{Enabled: true}` literal).

Run: `go test ./cmd/... -run 'TestRunModels' -v`
Expected: PASS (all 5 subtests, including the new alignment test)

Commit:

```bash
git add cmd/models.go cmd/models_test.go
git commit -m "Add real alignment-under-color test; drop modelRow's redundant state field"
```

---

### Task 4: Color `status`

**Files:**
- Modify: `cmd/status.go`

Current relevant lines in `runStatus` (the human-readable branch, at the end of the function):

```go
	fmt.Fprintf(w, "LM Studio at %s: reachable\n", host)
	if len(loaded) == 0 {
		fmt.Fprintln(w, "No models currently loaded.")
		return nil
	}
	fmt.Fprintln(w, "Loaded models:")
	for _, l := range loaded {
		fmt.Fprintf(w, "  - %s (%s)\n", l.Key, l.InstanceID)
	}
	return nil
```

Wait — check the actual current file first; the coloring pass before this one already changed `"reachable"` and `"No models currently loaded."`. The current content is:

```go
	fmt.Fprintf(w, "LM Studio at %s: %s\n", host, palette.Green("reachable"))
	if len(loaded) == 0 {
		fmt.Fprintln(w, palette.Dim("No models currently loaded."))
		return nil
	}
	fmt.Fprintln(w, "Loaded models:")
	for _, l := range loaded {
		fmt.Fprintf(w, "  - %s (%s)\n", l.Key, l.InstanceID)
	}
	return nil
```

- [ ] **Step 1: Make the edit**

Replace that block with:

```go
	fmt.Fprintf(w, "LM Studio at %s: %s\n", palette.Cyan(host), palette.Green("reachable"))
	if len(loaded) == 0 {
		fmt.Fprintln(w, palette.Dim("No models currently loaded."))
		return nil
	}
	fmt.Fprintln(w, palette.Bold("Loaded models:"))
	for _, l := range loaded {
		fmt.Fprintf(w, "  - %s (%s)\n", palette.Cyan(l.Key), palette.Cyan(l.InstanceID))
	}
	return nil
```

**Important:** never pass an already-colored string as the argument to another `Palette` method (e.g. `palette.Dim(palette.Cyan(s))`), and never build one colored string by embedding another colored string inside the text passed to a wrap — each `Palette` method appends its own full reset code (`\033[0m`), so nesting one wrap's *output* inside another wrap's *input* causes the inner reset to cancel the outer color partway through the line. Every colored piece above is a separate, sibling `palette.X(...)` call passed as its own `%s` argument to `Fprintf`/string concatenation — never one wrapped inside another. Keep that pattern in every remaining task.

- [ ] **Step 2: Verify**

Run: `go test ./cmd/... -run 'TestRunStatus' -v`
Expected: PASS (all `TestRunStatus_*` tests — they use `strings.Contains` on plain substrings like `"reachable"`/`"host"`, which still match since color is disabled during `go test`, per `internal/color.New`'s terminal detection)

- [ ] **Step 3: Commit**

```bash
git add cmd/status.go
git commit -m "Color host and labels in lmsctl status output"
```

---

### Task 5: Color `load`

**Files:**
- Modify: `cmd/load.go`

Current line in `runLoad`:

```go
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s as instance %s (%.1fs)\n", palette.Green("Loaded"), model, resp.InstanceID, resp.LoadTimeSeconds)
```

- [ ] **Step 1: Make the edit**

Replace with:

```go
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s as instance %s (%.1fs)\n", palette.Green("Loaded"), palette.Cyan(model), palette.Cyan(resp.InstanceID), resp.LoadTimeSeconds)
```

- [ ] **Step 2: Verify**

Run: `go test ./cmd/... -run 'TestRunLoad' -v`
Expected: PASS (all `TestRunLoad_*` and `TestBuildLoadRequest_*` tests)

- [ ] **Step 3: Commit**

```bash
git add cmd/load.go
git commit -m "Color model key and instance ID in lmsctl load output"
```

---

### Task 6: Color `unload`

**Files:**
- Modify: `cmd/unload.go`

Current relevant block in `runUnload`:

```go
	for _, id := range toUnload {
		if err := client.UnloadModel(cmd.Context(), id); err != nil {
			var notFound *lmstudio.ErrInstanceNotFound
			if errors.As(err, &notFound) {
				fmt.Fprintln(cmd.OutOrStdout(), palette.Dim(fmt.Sprintf("Instance %s was already unloaded", id)))
				continue
			}
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s instance %s\n", palette.Green("Unloaded"), id)
	}
```

- [ ] **Step 1: Make the edit**

Replace with:

```go
	for _, id := range toUnload {
		if err := client.UnloadModel(cmd.Context(), id); err != nil {
			var notFound *lmstudio.ErrInstanceNotFound
			if errors.As(err, &notFound) {
				fmt.Fprintln(cmd.OutOrStdout(), palette.Dim(fmt.Sprintf("Instance %s was already unloaded", id)))
				continue
			}
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s instance %s\n", palette.Green("Unloaded"), palette.Cyan(id))
	}
```

Only the last line changes (`id` → `palette.Cyan(id)`). The "already unloaded" line is deliberately left as one single `Dim(...)` wrap around the whole formatted string (not split into sibling colored pieces) — coloring `id` differently inside that line would require nesting a `Cyan` wrap inside the `Dim` wrap's input string, which is exactly the nesting pattern Task 4 said to avoid.

- [ ] **Step 2: Verify**

Run: `go test ./cmd/... -run 'TestRunUnload' -v`
Expected: PASS (all `TestRunUnload_*` tests)

- [ ] **Step 3: Commit**

```bash
git add cmd/unload.go
git commit -m "Color instance ID in lmsctl unload output"
```

---

### Task 7: Color `config`

**Files:**
- Modify: `cmd/config.go`

Current file has two relevant lines. In `configSetHostCmd`:

```go
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", palette.Green("Default host set to"), args[0])
```

In `configShowCmd`:

```go
		fmt.Fprintf(cmd.OutOrStdout(), "host:  %s\ntoken: %s\n", host, token)
```

- [ ] **Step 1: Make the edits**

Replace the `configSetHostCmd` line with:

```go
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", palette.Green("Default host set to"), palette.Cyan(args[0]))
```

Replace the `configShowCmd` line with:

```go
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n%s%s\n", palette.Bold("host:  "), palette.Cyan(host), palette.Bold("token: "), token)
```

Note the exact spacing inside the bold labels: `"host:  "` has two trailing spaces, `"token: "` has one — this preserves the original `"host:  %s\ntoken: %s\n"` column alignment (both labels are 7 characters wide including trailing spaces) now that the labels carry their own trailing whitespace instead of it living in the format string. `token`'s value (`"(set)"`/`"(not set)"`) is deliberately left uncolored — see the design spec's rationale (coloring it could imply something about the token's validity, which `lmsctl` has no way to know).

**Before verifying, fix a test-coupling issue this edit would otherwise introduce.** Task 4's code review found that `cmd/status_test.go` had an assertion checking a substring that spanned two separately-colored pieces (`"openai/gpt-oss-20b (inst-1)"`, contiguous only when color happens to be disabled) and fixed it by splitting into two `strings.Contains` checks. `cmd/config_test.go`'s `TestConfigShow_ShowsNotSetWhenNoHostConfigured` has the exact same shape: it currently checks the literal substring `"host:  (not set)"`, which spans the `Bold("host:  ")` piece and the plain `"(not set)"` piece you're about to introduce — contiguous today only because color is disabled during `go test`. Fix it the same way Task 4 did, before it becomes a latent trap: find this line in `cmd/config_test.go`

```go
	if !strings.Contains(out.String(), "host:  (not set)") {
```

**Splitting it into two independent `Contains` checks is not enough** — a
code review on this exact fix found that `"(not set)"` also appears on the
line below (the token line prints the same literal when no token is
configured), so two independent checks pass even if the HOST line's
fallback text is mutated away from `"(not set)"` entirely, as long as the
token line still says it. Scope both checks to just the host line (the
first line of output) instead:

```go
	hostLine := strings.SplitN(out.String(), "\n", 2)[0]
	if !strings.Contains(hostLine, "host:  ") || !strings.Contains(hostLine, "(not set)") {
```

(and update the `t.Errorf` message's variable reference from `out.String()` to `hostLine` to match)

- [ ] **Step 2: Verify**

Run: `go test ./cmd/... -run 'TestConfig' -v`
Expected: PASS (all `TestConfig*` tests)

- [ ] **Step 3: Commit**

```bash
git add cmd/config.go cmd/config_test.go
git commit -m "Color labels and values in lmsctl config output"
```

---

### Task 8: Color error output

**Files:**
- Modify: `cmd/root.go`

Current `cmd/root.go` has:

```go
var palette = color.New(os.Stdout)
```

and

```go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 1: Make the edits**

Add a second palette bound to stderr (errors go to stderr, which can be redirected independently of stdout: `lmsctl status | cat` — stdout piped, stderr left on the terminal — should still color the error, and `lmsctl status 2>err.log` should not write ANSI codes into the log file. Note `2>&1 | less` does NOT demonstrate this: that redirects stderr onto the same pipe as stdout, so both are non-terminal and color is correctly disabled either way — a code-quality review on this task verified that distinction empirically before this section was corrected). Change:

```go
var palette = color.New(os.Stdout)
```

to:

```go
var (
	palette    = color.New(os.Stdout)
	errPalette = color.New(os.Stderr)
)
```

Change `Execute()` to:

```go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, errPalette.Red(fmt.Sprintf("Error: %v", err)))
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify**

Run: `go build ./...`
Expected: no errors.

Run: `go test ./... -count=1`
Expected: PASS across all packages (nothing in the test suite calls `Execute()` directly — all command tests call `runXxx` functions or drive `rootCmd.Execute()` in `cmd/config_test.go`'s tests, none of which trigger an error path through `Execute()`'s top-level error printing, so this change has no test surface to break).

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go
git commit -m "Color the Error: prefix and message in lmsctl's top-level error output"
```

---

### Task 9: `lmsctl info <model>`

**Files:**
- Create: `cmd/info.go`
- Test: `cmd/info_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/lmstudio/lmstudiotest"
)

func TestRunInfo_NotFoundReturnsErrModelNotFound(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "other/model"},
	}}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	err := runInfo(cmd, fake, "missing/model", false)
	var notFound *lmstudio.ErrModelNotFound
	if !errors.As(err, &notFound) {
		t.Fatalf("err = %v, want *lmstudio.ErrModelNotFound", err)
	}
}

func TestRunInfo_ShowsStaticFieldsWhenNotLoaded(t *testing.T) {
	arch := "llama"
	format := "gguf"
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{
			Key:              "openai/gpt-oss-20b",
			Publisher:        "openai",
			DisplayName:      "GPT OSS 20B",
			Architecture:     &arch,
			Format:           &format,
			Quantization:     &lmstudio.Quantization{Name: "MXFP4", BitsPerWeight: 4.25},
			SizeBytes:        12137712025,
			MaxContextLength: 131072,
		},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runInfo: %v", err)
	}

	got := out.String()
	for _, want := range []string{"openai/gpt-oss-20b", "openai", "GPT OSS 20B", "llama", "gguf", "MXFP4", "131072", "Not currently loaded."} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
}

func TestRunInfo_ShowsLoadedInstanceConfig(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{
			Key: "openai/gpt-oss-20b",
			LoadedInstances: []lmstudio.LoadedInstance{
				{ID: "inst-1", Config: lmstudio.InstanceConfig{ContextLength: 16384, FlashAttention: true, OffloadKVCacheToGPU: false, Parallel: 1, EvalBatchSize: 512}},
			},
		},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", false); err != nil {
		t.Fatalf("runInfo: %v", err)
	}

	got := out.String()
	for _, want := range []string{"inst-1", "16384", "true", "false", "512"} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "Not currently loaded") {
		t.Errorf("output = %q, should not say not-loaded when an instance exists", got)
	}
}

func TestRunInfo_JSONOutput(t *testing.T) {
	fake := &lmstudiotest.Fake{ModelsResponse: &lmstudio.ModelsResponse{Models: []lmstudio.Model{
		{Key: "openai/gpt-oss-20b", Publisher: "openai"},
	}}}
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := runInfo(cmd, fake, "openai/gpt-oss-20b", true); err != nil {
		t.Fatalf("runInfo: %v", err)
	}
	if !strings.Contains(out.String(), `"key": "openai/gpt-oss-20b"`) {
		t.Errorf("output = %q, want JSON containing the model key", out.String())
	}
}

func TestRunInfo_PropagatesClientError(t *testing.T) {
	fake := &lmstudiotest.Fake{ListModelsErr: &lmstudio.ErrUnreachable{Host: "http://host:1234"}}
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})

	if err := runInfo(cmd, fake, "any/model", false); err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestRunInfo' -v`
Expected: FAIL — `runInfo`/`infoCmd` undefined (compile error).

- [ ] **Step 3: Write `cmd/info.go`**

```go
package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"lmsctl/internal/lmstudio"
	"lmsctl/internal/output"
)

var infoCmd = &cobra.Command{
	Use:   "info <model>",
	Short: "Show details for one downloaded model",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := newClient()
		if err != nil {
			return err
		}
		jsonOut, _ := cmd.Flags().GetBool("json")
		return runInfo(cmd, client, args[0], jsonOut)
	},
}

// fieldValue is one "Label: value" line in runInfo's output. yellow marks
// values that are quantitative (sizes, counts, context lengths) rather
// than free text (publisher, architecture, format).
type fieldValue struct {
	label, value string
	yellow       bool
}

func runInfo(cmd *cobra.Command, client lmstudio.Client, model string, jsonOut bool) error {
	resp, err := client.ListModels(cmd.Context())
	if err != nil {
		return err
	}

	var match *lmstudio.Model
	for i := range resp.Models {
		if resp.Models[i].Key == model {
			match = &resp.Models[i]
			break
		}
	}
	if match == nil {
		return &lmstudio.ErrModelNotFound{Model: model}
	}

	if jsonOut {
		return output.JSON(cmd.OutOrStdout(), match)
	}

	w := cmd.OutOrStdout()

	quant := "-"
	if match.Quantization != nil {
		quant = fmt.Sprintf("%s (%.2f bits/weight)", match.Quantization.Name, match.Quantization.BitsPerWeight)
	}

	printFields(w, []fieldValue{
		{"Key", match.Key, false},
		{"Publisher", match.Publisher, false},
		{"Display name", match.DisplayName, false},
		{"Architecture", derefOr(match.Architecture, "-"), false},
		{"Format", derefOr(match.Format, "-"), false},
		{"Quantization", quant, false},
		{"Size", formatBytes(match.SizeBytes), true},
		{"Max context", fmt.Sprintf("%d", match.MaxContextLength), true},
	})

	fmt.Fprintln(w)
	if len(match.LoadedInstances) == 0 {
		fmt.Fprintln(w, palette.Dim("Not currently loaded."))
		return nil
	}
	fmt.Fprintln(w, palette.Bold("Loaded instances:"))
	for _, inst := range match.LoadedInstances {
		fmt.Fprintln(w)
		fields := []fieldValue{
			{"Instance", inst.ID, false},
			{"Context length", fmt.Sprintf("%d", inst.Config.ContextLength), true},
			{"Flash attention", fmt.Sprintf("%t", inst.Config.FlashAttention), false},
			{"Offload KV to GPU", fmt.Sprintf("%t", inst.Config.OffloadKVCacheToGPU), false},
			{"Parallel", fmt.Sprintf("%d", inst.Config.Parallel), true},
			{"Eval batch size", fmt.Sprintf("%d", inst.Config.EvalBatchSize), true},
		}
		if inst.Config.NumExperts != 0 {
			fields = append(fields, fieldValue{"Num experts", fmt.Sprintf("%d", inst.Config.NumExperts), true})
		}
		printFields(w, fields)
	}
	return nil
}

// printFields prints label-aligned "Label: value" lines: labels are
// right-padded to the widest label in fields and bolded; "Key"/"Instance"
// values are colored cyan (they're identifiers), quantitative values
// yellow, everything else plain.
func printFields(w io.Writer, fields []fieldValue) {
	width := 0
	for _, f := range fields {
		if n := len(f.label) + 1; n > width { // +1 accounts for the trailing colon
			width = n
		}
	}
	for _, f := range fields {
		label := f.label + ":"
		value := f.value
		switch {
		case f.label == "Key" || f.label == "Instance":
			value = palette.Cyan(value)
		case f.yellow:
			value = palette.Yellow(value)
		}
		fmt.Fprintln(w, output.PadRight(label, width, palette.Bold)+" "+value)
	}
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func init() {
	infoCmd.Flags().Bool("json", false, "output machine-readable JSON")
	rootCmd.AddCommand(infoCmd)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -run 'TestRunInfo' -v`
Expected: PASS (all 5 `TestRunInfo_*` tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/info.go cmd/info_test.go
git commit -m "Add lmsctl info command"
```

---

### Task 10: Full suite, README, manual color verification

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Run the entire test suite**

Run: `go vet ./... && go test ./... -v`
Expected: PASS across every package, no `go vet` warnings.

- [ ] **Step 2: Build and check help output**

Run: `go build -o /tmp/lmsctl . && /tmp/lmsctl --help`
Expected: `info` now appears in the command list alongside `config`/`load`/`models`/`status`/`unload`.

Run: `/tmp/lmsctl info --help`
Expected: usage showing `info <model>` and a `--json` flag.

- [ ] **Step 3: Update `README.md`**

**Read the current `README.md` first rather than trusting the "find this
block" text below verbatim** — a code review on Task 9 found this plan
section had drifted out of sync with the actual file (the file's `--json`
paragraph was already corrected independently, earlier in this plan's
history, to not claim `--json` is a silent no-op on `unload`/`config` —
it actually errors with `unknown flag: --json` on those, since Task 8 made
it a per-command local flag). If the file doesn't match the "find" block
exactly, edit based on what's actually there instead of forcing a literal
match — the intent (add `info` to the usage list and mention it under
`--json`; add a color-detection note; leave the accurate `unload`/`config`
wording alone) matters more than an exact diff.

Find this block:

```markdown
## Usage

```bash
lmsctl status                          # is it up, what's loaded
lmsctl models                          # list downloaded models (alias: ls)
lmsctl load openai/gpt-oss-20b         # load a model
lmsctl load openai/gpt-oss-20b --context-length 16384 --flash-attention
lmsctl unload openai/gpt-oss-20b       # unload one model
lmsctl unload --all                    # unload everything loaded
lmsctl config show                     # see effective config (token redacted)
```

Add `--json` to `status`, `models`, or `load` for machine-readable output.
(`unload` and `config` are plain-text only — there's nothing in their
output that benefits from a JSON form.)
```

Replace it with:

```markdown
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
```

Also update the one-line project description near the top of the file
(currently "check status, list models, and load/unload them") to mention
the details view too, e.g. "check status, list models, view a model's
details, and load/unload them" — Task 9 added a whole command this
description doesn't reflect.

- [ ] **Step 4: Manual color verification**

This can't be automated in a non-interactive shell (color auto-disables when stdout isn't a terminal), so verify by hand: run `/tmp/lmsctl models --host <any reachable LM Studio host>` (or against a local `httptest` stub server if you don't have one handy) directly in a real terminal, and confirm:
- The `models` table's columns (KEY cyan, SIZE yellow, QUANTIZATION dim, STATE green/dim, headers bold) are still vertically aligned.
- `lmsctl info <model>` against a model with multiple loaded instances — each instance's "Label:" lines line up within that instance's block (this is covered by an automated test now, `TestRunInfo_ColumnsStayAlignedWithColorEnabledAndMultipleInstances`, but a quick visual look confirms the real terminal rendering matches).
- `lmsctl status`, `lmsctl load <model>`, `lmsctl unload <model>`, `lmsctl config show`, `lmsctl info <model>` all show color.
- `lmsctl status --host 127.0.0.1:1` (a dead port) shows the error in red.
- Piping output (e.g. `lmsctl models | cat`) shows NO color codes (confirms auto-disable works).
- `lmsctl status --host 127.0.0.1:1 2>err.log` (stdout still a terminal, stderr redirected to a file) — inspect `err.log` (e.g. `xxd err.log | head`) and confirm it contains NO `1b` (ESC) bytes. This is the one check that actually distinguishes `errPalette` (bound to stderr) from a bug that reused the stdout-bound `palette` for errors instead — both would pass the two checks above, but only the correct binding passes this one.

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "Document lmsctl info and automatic color detection in README"
```

---

## Plan self-review notes

- **Spec coverage:** `lmsctl info` → Task 9; new palette colors → Task 1; `PadRight` → Task 2; `models` table color-safe alignment → Task 3; labels/identifiers colored across `status`/`load`/`unload`/`config` → Tasks 4-7; error coloring → Task 8; README update → Task 10.
- **Nesting hazard called out explicitly** in Task 4 (the first task that composes multiple colors in one line) and referenced again in Task 6 where it determines a deliberate non-obvious choice (why the "already unloaded" line stays a single `Dim(...)` wrap instead of also coloring the instance ID).
- **Out of scope confirmed:** no live/interactive tokens-per-second session — that's a separate, not-yet-designed sub-project per the spec.
