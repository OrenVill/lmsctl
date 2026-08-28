package cmd

import (
	"bytes"
	"strings"
	"testing"

	"lmsctl/internal/config"
)

func TestConfigSetHost_WritesHostToConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"config", "set-host", "192.168.1.50:1234"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "192.168.1.50:1234") {
		t.Errorf("output = %q, want it to mention the host", out.String())
	}
}

func TestConfigShow_RedactsToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := config.Save(config.Config{Host: "host:1234", Token: "super-secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out.String(), "super-secret") {
		t.Errorf("output leaked the token: %q", out.String())
	}
	if !strings.Contains(out.String(), "host:1234") {
		t.Errorf("output = %q, want it to contain the host", out.String())
	}
}
