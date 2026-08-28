// Package output renders command results as either human-readable tables
// or machine-readable JSON.
package output

import (
	"bytes"
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

// Table is a tabwriter that aligns every column, including the last one in
// each row. text/tabwriter only pads tab-terminated cells, so a plain
// tabwriter.Writer leaves the final column of each row unpadded; Table
// inserts a trailing tab before every newline to bring it into the aligned
// column set as well.
type Table struct {
	tw *tabwriter.Writer
}

// NewTable returns a table writer configured for aligned column output.
// Write tab-separated rows to it and call Flush when done.
func NewTable(w io.Writer) *Table {
	return &Table{tw: tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)}
}

// Write implements io.Writer.
func (t *Table) Write(p []byte) (int, error) {
	if _, err := t.tw.Write(bytes.ReplaceAll(p, []byte("\n"), []byte("\t\n"))); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Flush flushes any buffered output to the underlying writer.
func (t *Table) Flush() error {
	return t.tw.Flush()
}
