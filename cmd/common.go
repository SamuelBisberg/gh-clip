package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/atotto/clipboard"
	"github.com/pkg/browser"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
	"github.com/SamuelBisberg/gh-clip/pkg/gh"
	"github.com/SamuelBisberg/gh-clip/pkg/store"
	"github.com/SamuelBisberg/gh-clip/pkg/tui"
)

// Seams for the outside world, swapped out in tests.
var (
	newREST         = gh.NewREST
	copyToClipboard = clipboard.WriteAll
	openBrowser     = browser.OpenURL
	stdinIsPipe     = func() bool {
		fi, err := os.Stdin.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice == 0
	}
	stdoutIsTTY = func() bool {
		fi, err := os.Stdout.Stat()
		return err == nil && fi.Mode()&os.ModeCharDevice != 0
	}
)

// loadTheme loads the persisted config and builds the matching color theme
// - the first thing every subcommand needs before it can print anything.
func loadTheme() (*config.Config, *tui.Theme, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, tui.New(cfg), nil
}

// session is an authenticated GitHub client plus the identity it acts as.
type session struct {
	rest  gh.REST
	owner string
}

func newSession() (*session, error) {
	rest, err := newREST()
	if err != nil {
		return nil, err
	}
	owner, err := gh.Login(rest)
	if err != nil {
		return nil, err
	}
	return &session{rest: rest, owner: owner}, nil
}

func (s *session) store(cfg *config.Config) *store.Store {
	return store.New(s.rest, s.owner, cfg.Storage.Repo)
}

// errNotInitialized is returned when the storage repo doesn't exist yet.
var errNotInitialized = errors.New("storage repo not found - run `gh clip init` first")

// hintInit rewrites a "repo missing" API failure into a pointer at
// `gh clip init`; other errors pass through untouched.
func hintInit(err error) error {
	if gh.IsStatus(err, http.StatusNotFound) {
		return fmt.Errorf("%w: %w", errNotInitialized, err)
	}
	return err
}
