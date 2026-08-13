package viewer

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(s string) tea.KeyMsg {
	switch s {
	case "j", "k", "g", "G", "v", "y":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
	panic("unhandled key " + s)
}

func step(m Model, keys ...string) Model {
	for _, k := range keys {
		m, _ = m.Update(key(k))
	}
	return m
}

func TestVisualSelectYankRange(t *testing.T) {
	m := New("a\nb\nc\nd\ne", 5)
	m = step(m, "j", "j", "v", "j", "y") // cursor->2, select start=2, cursor->3, yank
	if m.Yanked != "c\nd" {
		t.Fatalf("expected yank 'c\\nd', got %q", m.Yanked)
	}
	if m.Selecting {
		t.Fatalf("expected selection cleared after yank")
	}
}

func TestYankWholeBodyWithoutSelection(t *testing.T) {
	m := New("a\nb\nc", 5)
	m = step(m, "y")
	if m.Yanked != "a\nb\nc" {
		t.Fatalf("expected full body yank, got %q", m.Yanked)
	}
}

func TestEscCancelsSelection(t *testing.T) {
	m := New("a\nb\nc", 5)
	m = step(m, "v", "j", "esc")
	if m.Selecting {
		t.Fatalf("expected selection cancelled by esc")
	}
	if m.Yanked != "" {
		t.Fatalf("esc must not yank, got %q", m.Yanked)
	}
}

func TestSelectionOrderIndependentOfDirection(t *testing.T) {
	m := New("a\nb\nc\nd", 5)
	m = step(m, "j", "j", "j", "v", "k", "k", "y") // cursor 3, select start=3, cursor->1
	if m.Yanked != "b\nc\nd" {
		t.Fatalf("expected 'b\\nc\\nd' regardless of selection direction, got %q", m.Yanked)
	}
}

func TestScrollClampsToCursor(t *testing.T) {
	m := New("1\n2\n3\n4\n5\n6\n7\n8", 3)
	m = step(m, "G")
	if m.Offset != len(m.Lines)-m.Height {
		t.Fatalf("expected offset to follow cursor to bottom, got offset=%d", m.Offset)
	}
}
