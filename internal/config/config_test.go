package config

import (
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

func TestResolve_NoHostAnywhereReturnsErrNoHost(t *testing.T) {
	_, err := Resolve("", "", "", "", Config{})
	if err != ErrNoHost {
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
