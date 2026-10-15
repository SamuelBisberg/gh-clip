// Package config manages gh-clip's persistent user settings.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Color values for the "ui.color" setting: whether to force, suppress, or
// auto-detect ANSI color output.
const (
	ColorAuto   = "auto"
	ColorAlways = "always"
	ColorNever  = "never"
)

const (
	// DefaultRepo is the private repository clips are stored in.
	DefaultRepo = "gh-clip-storage"
	// ViewerRepo is the public repository hosting the mobile "tap to copy"
	// page on GitHub Pages.
	ViewerRepo = "gh-clip-viewer"
	// DefaultShareTTL is how long a shared Gist lives before the sweeper
	// removes it.
	DefaultShareTTL = "1h"
)

// Config holds all persisted gh-clip settings.
type Config struct {
	Storage struct {
		Repo string `json:"repo"`
	} `json:"storage"`
	Share struct {
		TTL string `json:"ttl"`
	} `json:"share"`
	UI struct {
		Color string `json:"color"`
	} `json:"ui"`
}

func defaults() *Config {
	c := &Config{}
	c.Storage.Repo = DefaultRepo
	c.Share.TTL = DefaultShareTTL
	c.UI.Color = ColorAuto
	return c
}

// ShareTTL returns the parsed share.ttl setting, falling back to the
// default if the stored value is somehow unparsable.
func (c *Config) ShareTTL() time.Duration {
	d, err := time.ParseDuration(c.Share.TTL)
	if err != nil || d <= 0 {
		d, _ = time.ParseDuration(DefaultShareTTL)
	}
	return d
}

// Path returns the on-disk location of the config file, honoring
// XDG_CONFIG_HOME / os.UserConfigDir so it resolves to ~/.config/gh/clip.json
// on Linux and the platform-appropriate equivalent elsewhere.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gh", "clip.json"), nil
}

// Load reads the config file, returning defaults if none exists yet.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaults(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config at %s: %w", path, err)
	}
	return cfg, nil
}

// Save writes the config file, creating its parent directory if needed.
func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config at %s: %w", path, err)
	}
	return nil
}

// settingField is one entry in the config schema: a dotted key plus how to
// read and validate/write it. Get, Set, and Keys all derive from the single
// fields table below, so adding a setting means adding one entry.
type settingField struct {
	key string
	get func(*Config) string
	set func(*Config, string) error
}

var fields = []settingField{
	{
		key: "storage.repo",
		get: func(c *Config) string { return c.Storage.Repo },
		set: func(c *Config, v string) error {
			if v == "" {
				return fmt.Errorf("storage.repo must not be empty")
			}
			c.Storage.Repo = v
			return nil
		},
	},
	{
		key: "share.ttl",
		get: func(c *Config) string { return c.Share.TTL },
		set: func(c *Config, v string) error {
			d, err := time.ParseDuration(v)
			if err != nil || d <= 0 {
				return fmt.Errorf("invalid share.ttl %q (want a positive duration such as 30m, 1h, 24h)", v)
			}
			c.Share.TTL = v
			return nil
		},
	},
	{
		key: "ui.color",
		get: func(c *Config) string { return c.UI.Color },
		set: func(c *Config, v string) error {
			switch v {
			case ColorAuto, ColorAlways, ColorNever:
				c.UI.Color = v
				return nil
			default:
				return fmt.Errorf("invalid ui.color %q (want one of: auto, always, never)", v)
			}
		},
	},
}

func lookupField(key string) (settingField, error) {
	for _, f := range fields {
		if f.key == key {
			return f, nil
		}
	}
	return settingField{}, fmt.Errorf("unknown config key %q", key)
}

// Get returns the string value stored at a dotted key, e.g. "share.ttl".
func (c *Config) Get(key string) (string, error) {
	f, err := lookupField(key)
	if err != nil {
		return "", err
	}
	return f.get(c), nil
}

// Set validates and assigns a value to a dotted key, e.g. "share.ttl".
func (c *Config) Set(key, value string) error {
	f, err := lookupField(key)
	if err != nil {
		return err
	}
	return f.set(c, value)
}

// Keys lists every known dotted config key, in stable display order.
func Keys() []string {
	keys := make([]string, len(fields))
	for i, f := range fields {
		keys[i] = f.key
	}
	return keys
}
