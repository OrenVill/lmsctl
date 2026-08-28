// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"encoding/json"
	"io"
	"strings"
)

// JSON writes v to w as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

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
