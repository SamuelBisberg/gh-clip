package tui

import (
	"errors"
	"strings"
	"testing"
)

// RunWithSpinner's animated branch only runs when os.Stdout is a terminal.
// captureStdout (styles_test.go) redirects it to an os.Pipe, which is never
// a terminal, so these tests exercise its non-animated fallback: print
// "label..." once and just run fn.
func TestTheme_RunWithSpinner(t *testing.T) {
	theme := noColorTheme()

	t.Run("runs fn and prints the label", func(t *testing.T) {
		called := false
		out := captureStdout(t, func() {
			err := theme.RunWithSpinner("doing work", func() error {
				called = true
				return nil
			})
			if err != nil {
				t.Errorf("RunWithSpinner() error = %v", err)
			}
		})
		if !called {
			t.Error("RunWithSpinner() did not call fn")
		}
		if !strings.Contains(out, "doing work...") {
			t.Errorf("RunWithSpinner() wrote %q, want it to contain %q", out, "doing work...")
		}
	})

	t.Run("propagates fn's error", func(t *testing.T) {
		wantErr := errors.New("boom")
		var gotErr error
		captureStdout(t, func() {
			gotErr = theme.RunWithSpinner("doing work", func() error {
				return wantErr
			})
		})
		if !errors.Is(gotErr, wantErr) {
			t.Errorf("RunWithSpinner() error = %v, want %v", gotErr, wantErr)
		}
	})
}
