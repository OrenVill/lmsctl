package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"lmsctl/internal/config"
)

func resetRootCmd(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})
}

func TestConfigSetHost_WritesHostToConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	resetRootCmd(t)

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"config", "set-host", "192.168.1.50:1234"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "192.168.1.50:1234") {
		t.Errorf("output = %q, want it to mention the host", out.String())
	}

	got, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Host != "192.168.1.50:1234" {
		t.Errorf("saved host = %q, want %q", got.Host, "192.168.1.50:1234")
	}
}

func TestConfigShow_RedactsToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	resetRootCmd(t)

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
	if !strings.Contains(out.String(), "(set)") {
		t.Errorf("output = %q, want it to report the token as (set)", out.String())
	}
}

func TestConfigShow_ReportsTokenNotSetWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	resetRootCmd(t)

	if err := config.Save(config.Config{Host: "host:1234"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "(not set)") {
		t.Errorf("output = %q, want it to report the token as (not set)", out.String())
	}
}

func TestConfigShow_ReflectsHostFlagOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	resetRootCmd(t)

	if err := config.Save(config.Config{Host: "file-host:1234"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"--host", "flag-host:5678", "config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "flag-host:5678") {
		t.Errorf("output = %q, want it to show the --host override, not the file value", out.String())
	}
	if strings.Contains(out.String(), "file-host:1234") {
		t.Errorf("output = %q, should not show the file's host once overridden", out.String())
	}
}

func TestConfigShow_ShowsNotSetWhenNoHostConfigured(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	resetRootCmd(t)

	out := &bytes.Buffer{}
	rootCmd.SetOut(out)
	rootCmd.SetArgs([]string{"config", "show"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	hostLine := strings.SplitN(out.String(), "\n", 2)[0]
	if !strings.Contains(hostLine, "host:  ") || !strings.Contains(hostLine, "(not set)") {
		t.Errorf("host line = %q, want host to show (not set) when nothing is configured", hostLine)
	}
}
