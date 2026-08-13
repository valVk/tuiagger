// Package viewer spikes the response-viewer visual-select-and-yank widget:
// no Bubbles component covers per-line visual selection, so this proves the
// Update/View split works as a plain, unit-testable state machine before
// Phase 3 builds the real thing on top of bubbles/viewport.
package viewer

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	Lines     []string
	Cursor    int
	Selecting bool
	SelStart  int
	Height    int
	Offset    int
	Yanked    string
}

func New(body string, height int) Model {
	return Model{Lines: strings.Split(body, "\n"), Height: height}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "j", "down":
		if m.Cursor < len(m.Lines)-1 {
			m.Cursor++
		}
	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "g":
		m.Cursor = 0
	case "G":
		m.Cursor = len(m.Lines) - 1
	case "v":
		if m.Selecting {
			m.Selecting = false
		} else {
			m.Selecting = true
			m.SelStart = m.Cursor
		}
	case "esc":
		m.Selecting = false
	case "y":
		if m.Selecting {
			m.Yanked = m.selectedText()
			m.Selecting = false
		} else {
			m.Yanked = strings.Join(m.Lines, "\n")
		}
	}
	m.clampScroll()
	return m, nil
}

func (m Model) selectedText() string {
	lo, hi := m.SelStart, m.Cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return strings.Join(m.Lines[lo:hi+1], "\n")
}

func (m *Model) clampScroll() {
	m.Offset = min(m.Offset, m.Cursor)
	if m.Cursor >= m.Offset+m.Height {
		m.Offset = m.Cursor - m.Height + 1
	}
}

var selectedStyle = lipgloss.NewStyle().Reverse(true)

func (m Model) View() string {
	lo, hi := -1, -1
	if m.Selecting {
		lo, hi = m.SelStart, m.Cursor
		if lo > hi {
			lo, hi = hi, lo
		}
	}
	end := min(m.Offset+m.Height, len(m.Lines))
	var b strings.Builder
	for i := m.Offset; i < end; i++ {
		line := m.Lines[i]
		if i >= lo && i <= hi {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
