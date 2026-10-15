package tui

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name  string
		color string
	}{
		{name: "auto", color: config.ColorAuto},
		{name: "always", color: config.ColorAlways},
		{name: "never", color: config.ColorNever},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.UI.Color = tt.color
			if got := New(cfg); got == nil {
				t.Error("New() = nil, want a Theme")
			}
		})
	}

	t.Run("never disables ANSI codes", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.UI.Color = config.ColorNever
		theme := New(cfg)
		if got := theme.Bold.Render("x"); got != "x" {
			t.Errorf("Bold.Render(%q) = %q, want plain text with no ANSI codes", "x", got)
		}
	})

	t.Run("always forces ANSI codes", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.UI.Color = config.ColorAlways
		theme := New(cfg)
		if got := theme.Success.Render("x"); !strings.Contains(got, "\x1b[") {
			t.Errorf("Success.Render(%q) = %q, want it to contain an ANSI escape sequence", "x", got)
		}
	})
}

// captureStdout redirects os.Stdout for the duration of fn and returns
// whatever was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	return captureFD(t, &os.Stdout, fn)
}

// captureStderr redirects os.Stderr for the duration of fn and returns
// whatever was written to it.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	return captureFD(t, &os.Stderr, fn)
}

func captureFD(t *testing.T, target **os.File, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	orig := *target
	*target = w
	defer func() { *target = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading pipe: %v", err)
	}
	return string(out)
}

func noColorTheme() *Theme {
	cfg := &config.Config{}
	cfg.UI.Color = config.ColorNever
	return New(cfg)
}

func TestTheme_Successf(t *testing.T) {
	theme := noColorTheme()
	out := captureStdout(t, func() { theme.Successf("done %s", "thing") })
	if !strings.Contains(out, "✓") || !strings.Contains(out, "done thing") {
		t.Errorf("Successf() wrote %q, want it to contain %q and the formatted message", out, "✓")
	}
}

func TestTheme_Errorf(t *testing.T) {
	theme := noColorTheme()
	out := captureStderr(t, func() { theme.Errorf("failed: %s", "oops") })
	if !strings.Contains(out, "✗") || !strings.Contains(out, "failed: oops") {
		t.Errorf("Errorf() wrote %q, want it to contain %q and the formatted message", out, "✗")
	}
}

func TestTheme_Warningf(t *testing.T) {
	theme := noColorTheme()
	out := captureStdout(t, func() { theme.Warningf("careful: %s", "thing") })
	if !strings.Contains(out, "⚠") || !strings.Contains(out, "careful: thing") {
		t.Errorf("Warningf() wrote %q, want it to contain %q and the formatted message", out, "⚠")
	}
}

func TestTheme_Infof(t *testing.T) {
	theme := noColorTheme()
	out := captureStdout(t, func() { theme.Infof("fyi: %s", "thing") })
	if !strings.Contains(out, "ℹ") || !strings.Contains(out, "fyi: thing") {
		t.Errorf("Infof() wrote %q, want it to contain %q and the formatted message", out, "ℹ")
	}
}
