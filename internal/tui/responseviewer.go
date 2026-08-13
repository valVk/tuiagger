package tui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// responseViewer is the visual-select-and-yank response body viewer,
// promoted from the Phase 0 spike (internal/spike/viewer) into real use.
// Same Update/View shape, proven by that spike's unit tests; the only
// addition here is wiring 'y' to the OS clipboard via a tea.Cmd rather than
// writing to a field, since a real yank is a side effect.
type responseViewer struct {
	Lines     []string
	Cursor    int
	Selecting bool
	SelStart  int
	Height    int
	Offset    int
}

func newResponseViewer(body string, height int) responseViewer {
	return responseViewer{Lines: strings.Split(body, "\n"), Height: height}
}

// handleKey processes J/K/g/G/v/y/Esc — the visual-select keyset only,
// distinct from the lowercase j/k/g generic scroll keys so both can coexist
// on the same screen without conflict (matches the TS app's key layout).
func (v responseViewer) handleKey(keyStr string) (responseViewer, tea.Cmd) {
	switch keyStr {
	case "J", "down":
		if v.Cursor < len(v.Lines)-1 {
			v.Cursor++
		}
	case "K", "up":
		if v.Cursor > 0 {
			v.Cursor--
		}
	case "g":
		v.Cursor = 0
	case "G":
		v.Cursor = len(v.Lines) - 1
	case "v":
		if v.Selecting {
			v.Selecting = false
		} else {
			v.Selecting = true
			v.SelStart = v.Cursor
		}
	case "esc":
		v.Selecting = false
	case "y":
		text := v.selectedText()
		v.Selecting = false
		v.clampScroll()
		return v, yankCmd(text)
	default:
		return v, nil
	}
	v.clampScroll()
	return v, nil
}

func (v responseViewer) selectedText() string {
	if !v.Selecting {
		return strings.Join(v.Lines, "\n")
	}
	lo, hi := v.SelStart, v.Cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return strings.Join(v.Lines[lo:hi+1], "\n")
}

func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.WriteAll(text)
		return nil
	}
}

func (v *responseViewer) clampScroll() {
	v.Offset = min(v.Offset, v.Cursor)
	if v.Cursor >= v.Offset+v.Height {
		v.Offset = v.Cursor - v.Height + 1
	}
}

// render renders every line with cursor/selection highlighting. It does not
// window by v.Height/v.Offset — the caller (the right panel) already
// scrolls its full content via line-slicing, so the viewer only needs to
// own cursor position and selection state, not its own scroll window.
func (v responseViewer) render(width int) []string {
	lo, hi := -1, -1
	if v.Selecting {
		lo, hi = v.SelStart, v.Cursor
		if lo > hi {
			lo, hi = hi, lo
		}
	}
	lines := make([]string, len(v.Lines))
	for i, raw := range v.Lines {
		line := truncate(raw, width)
		switch {
		case i >= lo && i <= hi:
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		case i == v.Cursor:
			line = cyanStyle.Render(line)
		default:
			line = dimStyle.Render(line)
		}
		lines[i] = line
	}
	return lines
}
