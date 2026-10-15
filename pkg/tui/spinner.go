package tui

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")

// RunWithSpinner animates an inline spinner labeled label while fn runs,
// so network/git operations (fetch, push, AI summary generation) don't
// leave the terminal looking frozen. When stdout isn't a TTY (e.g. CI logs)
// it prints a single "label..." line instead of animating.
func (t *Theme) RunWithSpinner(label string, fn func() error) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Println(label + "...")
		return fn()
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			fmt.Printf("\r%s %s", t.Info.Render(string(spinnerFrames[i%len(spinnerFrames)])), label)
			i++
			select {
			case <-done:
				return
			case <-ticker.C:
			}
		}
	}()

	defer func() {
		close(done)
		<-stopped
		fmt.Print("\r\033[K")
	}()
	return fn()
}
