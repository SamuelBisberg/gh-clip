package main

import (
	"os"

	"github.com/SamuelBisberg/gh-clip/cmd"
	"github.com/SamuelBisberg/gh-clip/pkg/config"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
// (cli/gh-extension-precompile does this automatically on release).
var version = "dev"

func main() {
	cmd.Version = version

	if err := cmd.NewRootCmd().Execute(); err != nil {
		cfg, cfgErr := config.Load()
		if cfgErr != nil {
			cfg = &config.Config{}
		}
		tui.New(cfg).Errorf("%v", err)
		os.Exit(1)
	}
}
