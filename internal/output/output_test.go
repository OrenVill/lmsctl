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
