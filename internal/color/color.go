// Package color provides minimal ANSI color helpers for lmsctl's
// human-readable output.
package color

import "os"

// Palette applies ANSI styling when Enabled is true, and is a no-op
// otherwise (so callers can use it unconditionally without branching).
type Palette struct {
	Enabled bool
}

// New returns a Palette that enables color only when w looks like an
// interactive terminal and the NO_COLOR environment variable is not set
// (see https://no-color.org/).
func New(w *os.File) Palette {
	if os.Getenv("NO_COLOR") != "" {
		return Palette{Enabled: false}
	}
	stat, err := w.Stat()
	if err != nil {
		return Palette{Enabled: false}
	}
	return Palette{Enabled: stat.Mode()&os.ModeCharDevice != 0}
}

const reset = "\033[0m"

func (p Palette) wrap(code, s string) string {
	if !p.Enabled {
		return s
	}
	return code + s + reset
}

// Green styles s for a positive/active state (e.g. "reachable", "loaded").
func (p Palette) Green(s string) string { return p.wrap("\033[32m", s) }

// Dim styles s for an inactive/neutral state (e.g. "not-loaded").
func (p Palette) Dim(s string) string { return p.wrap("\033[2m", s) }

// Bold styles s for emphasis (e.g. table headers).
func (p Palette) Bold(s string) string { return p.wrap("\033[1m", s) }

// Cyan styles s as an identifier the user would reference elsewhere (a
// model key, instance ID, or host).
func (p Palette) Cyan(s string) string { return p.wrap("\033[36m", s) }

// Yellow styles s as a secondary/quantitative value (a size, a
// quantization name).
func (p Palette) Yellow(s string) string { return p.wrap("\033[33m", s) }

// Red styles s as an error.
func (p Palette) Red(s string) string { return p.wrap("\033[31m", s) }
