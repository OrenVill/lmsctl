package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the on-disk shape of ~/.config/lmsctl/config.yaml.
type Config struct {
	Host  string `yaml:"host"`
	Token string `yaml:"token,omitempty"`
}

// Path returns the path to the config file, honoring XDG_CONFIG_HOME.
func Path() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("determining home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lmsctl", "config.yaml"), nil
}

// Load reads the config file. A missing file is not an error; it returns a
// zero-value Config.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("setting permissions on config file %s: %w", path, err)
	}
	return nil
}

// Effective holds the fully-resolved settings for one command invocation.
type Effective struct {
	Host  string
	Token string
}

// ErrNoHost is returned by Resolve when no host is configured anywhere.
var ErrNoHost = errors.New("no host configured: run 'lmsctl config set-host <host:port>', set LMSCTL_HOST, or pass --host")

// Resolve applies the precedence flag > env var > config file to produce
// the effective settings for one invocation. Empty strings mean "not set"
// at that level.
func Resolve(flagHost, flagToken, envHost, envToken string, fileCfg Config) (Effective, error) {
	host := firstNonEmpty(flagHost, envHost, fileCfg.Host)
	if host == "" {
		return Effective{}, ErrNoHost
	}
	token := firstNonEmpty(flagToken, envToken, fileCfg.Token)
	return Effective{Host: host, Token: token}, nil
}

// EffectiveDisplay resolves host/token the same way Resolve does, but never
// errors on a missing host: an empty result means "not configured
// anywhere", which callers like `config show` display as such rather than
// treating as a hard failure (unlike Resolve, which real commands need to
// fail fast on).
func EffectiveDisplay(flagHost, flagToken, envHost, envToken string, fileCfg Config) Effective {
	return Effective{
		Host:  firstNonEmpty(flagHost, envHost, fileCfg.Host),
		Token: firstNonEmpty(flagToken, envToken, fileCfg.Token),
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
