package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_PrecedenceFlagWinsOverEnvAndFile(t *testing.T) {
	eff, err := Resolve("flag-host:1234", "", "env-host:1234", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "flag-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "flag-host:1234")
	}
}

func TestResolve_EnvWinsOverFile(t *testing.T) {
	eff, err := Resolve("", "", "env-host:1234", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "env-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "env-host:1234")
	}
}

func TestResolve_FallsBackToFile(t *testing.T) {
	eff, err := Resolve("", "", "", "", Config{Host: "file-host:1234"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Host != "file-host:1234" {
		t.Errorf("Host = %q, want %q", eff.Host, "file-host:1234")
	}
}

func TestResolve_TokenFollowsSamePrecedence(t *testing.T) {
	eff, err := Resolve("host:1234", "flag-token", "", "env-token", Config{Token: "file-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Token != "flag-token" {
		t.Errorf("Token = %q, want %q", eff.Token, "flag-token")
	}
}

func TestResolve_TokenEnvWinsOverFile(t *testing.T) {
	eff, err := Resolve("host:1234", "", "", "env-token", Config{Token: "file-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eff.Token != "env-token" {
		t.Errorf("Token = %q, want %q", eff.Token, "env-token")
	}
}

func TestResolve_NoHostAnywhereReturnsErrNoHost(t *testing.T) {
	_, err := Resolve("", "", "", "", Config{})
	if !errors.Is(err, ErrNoHost) {
		t.Errorf("err = %v, want ErrNoHost", err)
	}
}

func TestSaveAndLoad_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := Config{Host: "192.168.1.50:1234", Token: "secret"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}

	path, _ := Path()
	if filepath.Dir(path) != filepath.Join(dir, "lmsctl") {
		t.Errorf("unexpected config dir: %s", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestSave_TightensPermissionsOnPreExistingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("host: old\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := Save(Config{Host: "new-host:1234", Token: "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode = %v, want 0600 (Save must tighten permissions on an existing file)", info.Mode().Perm())
	}
}

func TestLoad_MissingFileReturnsZeroValue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != (Config{}) {
		t.Errorf("Load() = %+v, want zero value", got)
	}
}

func TestLoad_CorruptYAMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("host: [unclosed"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for corrupt YAML, got nil")
	}
}

func TestPath_FallsBackToHomeConfigWhenXDGUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(home, ".config", "lmsctl", "config.yaml")
	if path != want {
		t.Errorf("Path() = %q, want %q", path, want)
	}
}
