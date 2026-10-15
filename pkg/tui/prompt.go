package tui

import (
	"github.com/charmbracelet/huh"
)

// ErrAborted is returned by prompts when the user cancels (e.g. Ctrl+C).
var ErrAborted = huh.ErrUserAborted

// PromptSecret asks for a masked value such as an access token.
func PromptSecret(title string) (string, error) {
	var v string
	err := huh.NewInput().
		Title(title).
		EchoMode(huh.EchoModePassword).
		Value(&v).
		Run()
	return v, err
}
