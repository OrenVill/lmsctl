package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSON_WritesIndentedJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, map[string]string{"key": "value"}); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(buf.String(), "\"key\": \"value\"") {
		t.Errorf("output = %q, want indented JSON containing the key/value", buf.String())
	}
}

func TestNewTable_AlignsColumns(t *testing.T) {
	var buf bytes.Buffer
	tw := NewTable(&buf)
	tw.Write([]byte("A\tBB\n"))
	tw.Write([]byte("CC\tD\n"))
	if err := tw.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}
	// Equal line length is a poor proxy for "aligned" (text/tabwriter doesn't
	// pad an untabbed last cell, so it isn't achievable here anyway, and
	// achieving it via a wrapper adds trailing whitespace to every row plus
	// edge cases for tab-free lines). Check what "aligned" actually means:
	// the second column starts at the same offset on both lines.
	if strings.Index(lines[0], "BB") != strings.Index(lines[1], "D") {
		t.Errorf("columns not aligned: %q vs %q", lines[0], lines[1])
	}
}
