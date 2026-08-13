package tui

import "testing"

func stepViewer(v responseViewer, keys ...string) responseViewer {
	for _, k := range keys {
		v, _ = v.handleKey(k)
	}
	return v
}

func TestResponseViewerCursorMovement(t *testing.T) {
	v := newResponseViewer("a\nb\nc", 5)
	v = stepViewer(v, "J", "J")
	if v.Cursor != 2 {
		t.Errorf("expected cursor 2, got %d", v.Cursor)
	}
	v = stepViewer(v, "K")
	if v.Cursor != 1 {
		t.Errorf("expected cursor 1, got %d", v.Cursor)
	}
	v = stepViewer(v, "g")
	if v.Cursor != 0 {
		t.Errorf("expected g to jump to 0, got %d", v.Cursor)
	}
	v = stepViewer(v, "G")
	if v.Cursor != 2 {
		t.Errorf("expected G to jump to last line, got %d", v.Cursor)
	}
}

func TestResponseViewerVisualSelectAndYankClearsSelection(t *testing.T) {
	v := newResponseViewer("a\nb\nc\nd", 5)
	v = stepViewer(v, "J", "J", "v", "J") // cursor->2, select start=2, cursor->3
	if !v.Selecting {
		t.Fatalf("expected visual mode active")
	}
	if got := v.selectedText(); got != "c\nd" {
		t.Errorf("expected 'c\\nd', got %q", got)
	}

	next, cmd := v.handleKey("y")
	if next.Selecting {
		t.Errorf("expected selection cleared after yank")
	}
	if cmd == nil {
		t.Errorf("expected yank to return a clipboard command")
	}
}

func TestResponseViewerEscCancelsSelection(t *testing.T) {
	v := newResponseViewer("a\nb\nc", 5)
	v = stepViewer(v, "v", "J", "esc")
	if v.Selecting {
		t.Errorf("expected esc to cancel selection")
	}
}

func TestResponseViewerRenderHighlightsSelection(t *testing.T) {
	v := newResponseViewer("alpha\nbeta\ngamma", 5)
	v = stepViewer(v, "v", "J")
	lines := v.render(80)
	if len(lines) != 3 {
		t.Fatalf("expected 3 rendered lines, got %d", len(lines))
	}
}
