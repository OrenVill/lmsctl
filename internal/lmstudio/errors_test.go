package lmstudio

import (
	"errors"
	"strings"
	"testing"
)

func TestErrUnreachable_MessageMentionsHostAndLocalNetworkSetting(t *testing.T) {
	err := &ErrUnreachable{Host: "http://192.168.1.50:1234", Err: errors.New("connection refused")}
	msg := err.Error()
	if !strings.Contains(msg, "192.168.1.50:1234") {
		t.Errorf("message = %q, want it to mention the host", msg)
	}
	if !strings.Contains(msg, "Serve on Local Network") {
		t.Errorf("message = %q, want it to mention the LM Studio setting", msg)
	}
}

func TestErrUnauthorized_MessageMentionsToken(t *testing.T) {
	msg := (&ErrUnauthorized{}).Error()
	if !strings.Contains(msg, "token") {
		t.Errorf("message = %q, want it to mention the token", msg)
	}
}

func TestErrModelNotFound_MessageMentionsModelKey(t *testing.T) {
	msg := (&ErrModelNotFound{Model: "nonexistent/model"}).Error()
	if !strings.Contains(msg, "nonexistent/model") {
		t.Errorf("message = %q, want it to mention the model key", msg)
	}
}

func TestErrModelNotLoaded_MessageMentionsModelKey(t *testing.T) {
	msg := (&ErrModelNotLoaded{Model: "openai/gpt-oss-20b"}).Error()
	if !strings.Contains(msg, "openai/gpt-oss-20b") {
		t.Errorf("message = %q, want it to mention the model key", msg)
	}
}
