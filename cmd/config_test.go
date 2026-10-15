package cmd

import (
	"testing"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
)

func TestConfigSetGet(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := runConfigSet("share.ttl", "2h"); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := cfg.Get("share.ttl"); got != "2h" {
		t.Errorf("share.ttl = %q", got)
	}
	if err := runConfigSet("share.ttl", "never"); err == nil {
		t.Error("expected validation error")
	}
	if err := runConfigGet("nope"); err == nil {
		t.Error("expected unknown key error")
	}
}
