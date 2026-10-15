package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SamuelBisberg/gh-clip/pkg/config"
	"github.com/SamuelBisberg/gh-clip/pkg/store"
)

func testClips() []store.Clip {
	now := time.Now()
	return []store.Clip{
		{Name: "3-c.txt", Content: "third", Created: now},
		{Name: "2-b.txt", Content: "second\nline", Created: now},
		{Name: "1-a.txt", Content: strings.Repeat("x", 5000), Created: now},
	}
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newTestModel(actions ListActions) *listModel {
	return newListModel(New(&config.Config{}), testClips(), actions)
}

func TestListNavigationAndSelect(t *testing.T) {
	m := newTestModel(ListActions{})
	m.Update(key("down"))
	m.Update(key("j"))
	m.Update(key("j")) // clamped at the last item
	if m.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", m.cursor)
	}
	_, cmd := m.Update(key("enter"))
	if cmd == nil || m.result.Selected == nil || m.result.Selected.Name != "1-a.txt" {
		t.Errorf("selected = %+v", m.result.Selected)
	}
}

func TestListEscQuitsWithoutSelection(t *testing.T) {
	m := newTestModel(ListActions{})
	_, cmd := m.Update(key("esc"))
	if cmd == nil || m.result.Selected != nil {
		t.Errorf("esc: cmd=%v selected=%v", cmd, m.result.Selected)
	}
}

func TestListQR(t *testing.T) {
	var shared []string
	m := newTestModel(ListActions{Share: func(c store.Clip) (string, error) {
		shared = append(shared, c.Name)
		return "https://me.github.io/gh-clip-viewer/?gist=abc", nil
	}})
	_, cmd := m.Update(key("q"))
	if m.mode != modeSharing || cmd == nil {
		t.Fatalf("mode=%v cmd=%v", m.mode, cmd)
	}
	m.Update(cmd())
	if len(shared) != 1 || m.mode != modeQR || m.qr == "" || !strings.Contains(m.View(), "gist=abc") {
		t.Fatalf("shared=%v mode=%v", shared, m.mode)
	}
	m.Update(key("x"))
	if m.mode != modeList {
		t.Error("any key should leave the QR view")
	}

	// A large clip is shared by link too, so its size doesn't matter.
	m.cursor = 2
	_, cmd = m.Update(key("q"))
	if cmd == nil || m.mode != modeSharing {
		t.Errorf("mode=%v", m.mode)
	}
}

func TestListQRShareError(t *testing.T) {
	m := newTestModel(ListActions{Share: func(store.Clip) (string, error) { return "", errors.New("boom") }})
	_, cmd := m.Update(key("q"))
	m.Update(cmd())
	if m.mode != modeList || !strings.Contains(m.status, "boom") {
		t.Errorf("mode=%v status=%q", m.mode, m.status)
	}
}

func TestListDelete(t *testing.T) {
	var deleted []string
	m := newTestModel(ListActions{Delete: func(c store.Clip) error { deleted = append(deleted, c.Name); return nil }})

	m.Update(key("d"))
	if m.mode != modeConfirmDelete {
		t.Fatal("d should ask for confirmation")
	}
	_, cmd := m.Update(key("y"))
	if cmd == nil {
		t.Fatal("y should return a delete command")
	}
	m.Update(cmd())
	if len(m.clips) != 2 || deleted[0] != "3-c.txt" {
		t.Errorf("clips=%d deleted=%v", len(m.clips), deleted)
	}
}

func TestListDeleteCancelAndError(t *testing.T) {
	m := newTestModel(ListActions{Delete: func(store.Clip) error { return errors.New("boom") }})
	m.Update(key("d"))
	m.Update(key("n"))
	if m.mode != modeList || len(m.clips) != 3 {
		t.Error("non-y key should cancel")
	}

	m.Update(key("d"))
	_, cmd := m.Update(key("y"))
	m.Update(cmd())
	if len(m.clips) != 3 || !strings.Contains(m.status, "boom") {
		t.Errorf("clips=%d status=%q", len(m.clips), m.status)
	}
}

func TestListView(t *testing.T) {
	out := newTestModel(ListActions{}).View()
	for _, want := range []string{"third", "second line", "enter copy"} {
		if !strings.Contains(out, want) {
			t.Errorf("View missing %q:\n%s", want, out)
		}
	}
	empty := newListModel(New(&config.Config{}), nil, ListActions{}).View()
	if !strings.Contains(empty, "No clips yet") {
		t.Errorf("empty View = %q", empty)
	}
}

func TestPreview(t *testing.T) {
	if got := preview("a\n  b\tc", 20); got != "a b c" {
		t.Errorf("preview = %q", got)
	}
	if got := preview("abcdefghij", 5); got != "abcd…" {
		t.Errorf("preview = %q", got)
	}
}

func TestQRString(t *testing.T) {
	if s, err := QRString("hello"); err != nil || s == "" {
		t.Errorf("QRString = %q, %v", s, err)
	}
	if _, err := QRString(strings.Repeat("x", MaxQRBytes+1)); err == nil {
		t.Error("expected too-long error")
	}
}
