// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"encoding/json"
	"io"
	"text/tabwriter"
)

// JSON writes v to w as indented JSON.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// NewTable returns a tabwriter configured for aligned column output. Write
// tab-separated rows to it and call Flush when done.
func NewTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}
