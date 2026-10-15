// Package tui provides gh-clip's color-coded terminal output, spinners, and
// interactive prompts, all built on lipgloss/huh with no external processes.
package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
)

// Theme is a small, consistent stylesheet used for every status message
// gh-clip prints: green for success, red for errors, yellow for warnings,
// and cyan for informational/data output.
type Theme struct {
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style
	Bold    lipgloss.Style
	Faint   lipgloss.Style
}

// New builds a Theme honoring the user's ui.color setting (auto/always/never).
func New(cfg *config.Config) *Theme {
	r := lipgloss.NewRenderer(os.Stdout)
	switch cfg.UI.Color {
	case config.ColorAlways:
		r.SetColorProfile(termenv.TrueColor)
	case config.ColorNever:
		r.SetColorProfile(termenv.Ascii)
	}

	return &Theme{
		Success: r.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		Error:   r.NewStyle().Foreground(lipgloss.Color("204")).Bold(true),
		Warning: r.NewStyle().Foreground(lipgloss.Color("214")).Bold(true),
		Info:    r.NewStyle().Foreground(lipgloss.Color("39")),
		Bold:    r.NewStyle().Bold(true),
		Faint:   r.NewStyle().Faint(true),
	}
}

// Successf prints a "✓"-prefixed success message to stdout.
func (t *Theme) Successf(format string, a ...any) {
	fmt.Println(t.Success.Render("✓") + " " + fmt.Sprintf(format, a...))
}

// Errorf prints a "✗"-prefixed error message to stderr.
func (t *Theme) Errorf(format string, a ...any) {
	fmt.Fprintln(os.Stderr, t.Error.Render("✗")+" "+fmt.Sprintf(format, a...))
}

// Warningf prints a "⚠"-prefixed warning message to stdout.
func (t *Theme) Warningf(format string, a ...any) {
	fmt.Println(t.Warning.Render("⚠") + " " + fmt.Sprintf(format, a...))
}

// Infof prints an "ℹ"-prefixed informational message to stdout.
func (t *Theme) Infof(format string, a ...any) {
	fmt.Println(t.Info.Render("ℹ") + " " + fmt.Sprintf(format, a...))
}
