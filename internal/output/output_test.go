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

func TestPadRight_PadsUsingPlainLengthNotColoredLength(t *testing.T) {
	// "colored" simulates an ANSI-wrapped string that's longer in bytes
	// than the plain text it wraps -- PadRight must pad based on plain's
	// length, not colored's, or alignment breaks when color is enabled.
	got := PadRight("ab", "\033[36mab\033[0m", 5)
	want := "\033[36mab\033[0m   "
	if got != want {
		t.Errorf("PadRight = %q, want %q", got, want)
	}
}

func TestPadRight_NoPaddingWhenPlainAlreadyMeetsWidth(t *testing.T) {
	got := PadRight("hello", "hello", 3)
	if got != "hello" {
		t.Errorf("PadRight = %q, want %q (no padding, and no truncation)", got, "hello")
	}
}

func TestPadRight_PlainAndColoredCanDiffer(t *testing.T) {
	// The common real usage: colored is plain wrapped in ANSI codes, but
	// PadRight doesn't require that relationship -- it trusts plain's
	// length and appends colored verbatim.
	got := PadRight("KEY", "COLORED-KEY", 6)
	want := "COLORED-KEY   "
	if got != want {
		t.Errorf("PadRight = %q, want %q", got, want)
	}
}
