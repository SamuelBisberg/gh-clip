package config

import (
	"path/filepath"
	"testing"
	"time"
)

// withConfigHome points os.UserConfigDir at a fresh temp directory so tests
// never touch the real user's config.
func withConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestPath(t *testing.T) {
	dir := withConfigHome(t)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	want := filepath.Join(dir, "gh", "clip.json")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoadDefaults(t *testing.T) {
	withConfigHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Storage.Repo != DefaultRepo {
		t.Errorf("Storage.Repo = %q, want %q", cfg.Storage.Repo, DefaultRepo)
	}
	if cfg.ShareTTL() != time.Hour {
		t.Errorf("ShareTTL() = %v, want 1h", cfg.ShareTTL())
	}
	if cfg.UI.Color != ColorAuto {
		t.Errorf("UI.Color = %q, want %q", cfg.UI.Color, ColorAuto)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	withConfigHome(t)

	cfg := defaults()
	if err := cfg.Set("share.ttl", "30m"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Set("storage.repo", "my-clips"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Storage.Repo != "my-clips" || got.ShareTTL() != 30*time.Minute {
		t.Errorf("round trip lost values: %+v", got)
	}
}

func TestSetValidation(t *testing.T) {
	tests := []struct {
		key, value string
		wantErr    bool
	}{
		{"ui.color", "always", false},
		{"ui.color", "rainbow", true},
		{"share.ttl", "2h", false},
		{"share.ttl", "soon", true},
		{"share.ttl", "-1h", true},
		{"storage.repo", "", true},
		{"nope", "x", true},
	}
	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			err := defaults().Set(tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Set(%q, %q) error = %v, wantErr %v", tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestKeysAreGettable(t *testing.T) {
	cfg := defaults()
	for _, k := range Keys() {
		if _, err := cfg.Get(k); err != nil {
			t.Errorf("Get(%q) error = %v", k, err)
		}
	}
}
