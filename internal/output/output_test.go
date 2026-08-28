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
