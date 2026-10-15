package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"

	"github.com/SamuelBisberg/gh-clip/pkg/store"
)

// ListActions are the side effects the clip list can trigger. They run as
// bubbletea commands, off the UI goroutine.
type ListActions struct {
	Delete func(store.Clip) error
	// Share publishes a clip as an expiring Gist and returns the viewer URL
	// that the QR code points at.
	Share func(store.Clip) (string, error)
}

// ListResult reports what the user did when the list exited.
type ListResult struct {
	// Selected is the clip chosen with Enter; nil if the user quit.
	Selected *store.Clip
}

type listMode int

const (
	modeList listMode = iota
	modeQR
	modeSharing
	modeConfirmDelete
)

type deletedMsg struct {
	name string
	err  error
}

type sharedMsg struct {
	url string
	err error
}

type listModel struct {
	theme   *Theme
	actions ListActions
	clips   []store.Clip
	cursor  int
	offset  int
	height  int
	width   int
	mode    listMode
	qr      string
	qrURL   string
	status  string
	result  ListResult
}

func newListModel(theme *Theme, clips []store.Clip, actions ListActions) *listModel {
	return &listModel{theme: theme, clips: clips, actions: actions, height: 15, width: 80}
}

// RunList shows the interactive clip picker and returns the user's choice.
func RunList(theme *Theme, clips []store.Clip, actions ListActions) (ListResult, error) {
	m := newListModel(theme, clips, actions)
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return ListResult{}, err
	}
	return final.(*listModel).result, nil
}

func (m *listModel) Init() tea.Cmd { return nil }

func (m *listModel) visibleRows() int {
	// Leave room for the title, the help line, and the status line.
	return max(m.height-5, 3)
}

func (m *listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height, m.width = msg.Height, msg.Width
		m.clamp()
	case deletedMsg:
		m.mode = modeList
		if msg.err != nil {
			m.status = m.theme.Error.Render("✗ " + msg.err.Error())
			return m, nil
		}
		for i, c := range m.clips {
			if c.Name == msg.name {
				m.clips = append(m.clips[:i], m.clips[i+1:]...)
				break
			}
		}
		m.status = m.theme.Success.Render("✓") + " Deleted"
		m.clamp()
	case sharedMsg:
		if m.mode != modeSharing {
			return m, nil
		}
		m.mode = modeList
		if msg.err != nil {
			m.status = m.theme.Error.Render("✗ " + msg.err.Error())
			return m, nil
		}
		qr, err := QRString(msg.url)
		if err != nil {
			m.status = m.theme.Warning.Render("⚠ " + err.Error())
			return m, nil
		}
		m.qr, m.qrURL = qr, msg.url
		m.mode = modeQR
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *listModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "ctrl+c" {
		return m, tea.Quit
	}

	switch m.mode {
	case modeQR:
		m.mode = modeList
		return m, nil
	case modeSharing:
		if key == "esc" {
			m.mode = modeList
		}
		return m, nil
	case modeConfirmDelete:
		if key == "y" && len(m.clips) > 0 {
			clip := m.clips[m.cursor]
			del := m.actions.Delete
			return m, func() tea.Msg { return deletedMsg{name: clip.Name, err: del(clip)} }
		}
		m.mode = modeList
		m.status = ""
		return m, nil
	}

	m.status = ""
	switch key {
	case "esc":
		return m, tea.Quit
	case "up", "k":
		m.cursor--
	case "down", "j":
		m.cursor++
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = len(m.clips) - 1
	case "enter":
		if len(m.clips) > 0 {
			c := m.clips[m.cursor]
			m.result.Selected = &c
		}
		return m, tea.Quit
	case "q":
		if len(m.clips) == 0 {
			return m, tea.Quit
		}
		if m.actions.Share == nil {
			return m, nil
		}
		clip := m.clips[m.cursor]
		share := m.actions.Share
		m.mode = modeSharing
		return m, func() tea.Msg {
			url, err := share(clip)
			return sharedMsg{url: url, err: err}
		}
	case "d":
		if len(m.clips) > 0 {
			m.mode = modeConfirmDelete
		}
	}
	m.clamp()
	return m, nil
}

func (m *listModel) clamp() {
	m.cursor = max(0, min(m.cursor, len(m.clips)-1))
	rows := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	m.offset = max(0, m.offset)
}

// preview collapses a clip to a single truncated line for the list.
func preview(content string, width int) string {
	line := strings.Join(strings.Fields(content), " ")
	if width < 4 {
		width = 4
	}
	r := []rune(line)
	if len(r) > width {
		return string(r[:width-1]) + "…"
	}
	return line
}

func (m *listModel) View() string {
	var b strings.Builder

	if m.mode == modeQR {
		b.WriteString(m.theme.Bold.Render("QR code") + "\n\n")
		b.WriteString(m.qr)
		b.WriteString("\n" + m.qrURL + "\n")
		b.WriteString("\n" + m.theme.Faint.Render("press any key to go back") + "\n")
		return b.String()
	}

	b.WriteString(m.theme.Bold.Render("Select a clip to copy") + "\n\n")
	if len(m.clips) == 0 {
		b.WriteString(m.theme.Faint.Render("No clips yet - try: echo hello | gh clip") + "\n")
	}

	end := min(m.offset+m.visibleRows(), len(m.clips))
	for i := m.offset; i < end; i++ {
		c := m.clips[i]
		age := humanize.Time(c.Created)
		line := fmt.Sprintf("%-14s  %s", age, preview(c.Content, m.width-22))
		if i == m.cursor {
			b.WriteString(m.theme.Info.Render("▸ "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n")
	switch {
	case m.mode == modeConfirmDelete:
		b.WriteString(m.theme.Warning.Render("Delete this clip? press y to confirm, any other key cancels") + "\n")
	case m.mode == modeSharing:
		b.WriteString(m.theme.Faint.Render("Creating secret gist... (esc to cancel)") + "\n")
	case m.status != "":
		b.WriteString(m.status + "\n")
	default:
		b.WriteString(m.theme.Faint.Render("↑/↓ move • enter copy • q qr • d delete • esc quit") + "\n")
	}
	return b.String()
}
