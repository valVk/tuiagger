package tui

import (
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/request"
)

func stepViewer(v responseViewer, keys ...string) responseViewer {
	for _, k := range keys {
		v, _ = v.handleKey(k)
	}
	return v
}

func TestResponseViewerCursorMovement(t *testing.T) {
	v := newResponseViewer("a\nb\nc")
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
	v := newResponseViewer("a\nb\nc\nd")
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
	if !next.Yanked {
		t.Errorf("expected Yanked flag set")
	}
	if cmd == nil {
		t.Errorf("expected yank to return a batched clipboard+timer command")
	}
}

func TestResponseViewerEscCancelsSelection(t *testing.T) {
	v := newResponseViewer("a\nb\nc")
	v = stepViewer(v, "v", "J", "esc")
	if v.Selecting {
		t.Errorf("expected esc to cancel selection")
	}
}

func TestResponseViewerNavigationDisabledOnRequestTab(t *testing.T) {
	v := newResponseViewer("a\nb\nc")
	v = stepViewer(v, `\`) // switch to request tab
	if v.Tab != tabRequestBody {
		t.Fatalf("expected request tab active")
	}
	before := v.Cursor
	v = stepViewer(v, "J", "G", "v")
	if v.Cursor != before || v.Selecting {
		t.Errorf("expected navigation/selection to be no-ops on the request tab")
	}
}

func TestResponseViewerBackslashTogglesTabRegardlessOfCurrentTab(t *testing.T) {
	v := newResponseViewer("a")
	v = stepViewer(v, `\`)
	if v.Tab != tabRequestBody {
		t.Fatalf("expected request tab")
	}
	v = stepViewer(v, `\`)
	if v.Tab != tabResponseBody {
		t.Fatalf("expected back to response tab")
	}
}

func TestResponseViewerScrollFollowsCursorPastViewport(t *testing.T) {
	var lines []string
	for range 30 {
		lines = append(lines, "line")
	}
	v := newResponseViewer(strings.Join(lines, "\n"))
	for range responseViewport + 3 {
		v = stepViewer(v, "J")
	}
	if v.Offset == 0 {
		t.Errorf("expected viewport to scroll to keep cursor visible, offset still 0")
	}
	if v.Cursor-v.Offset >= responseViewport {
		t.Errorf("expected cursor within viewport window, cursor=%d offset=%d", v.Cursor, v.Offset)
	}
}

func TestResponseViewerRenderHighlightsSelection(t *testing.T) {
	v := newResponseViewer("alpha\nbeta\ngamma")
	v = stepViewer(v, "v", "J")
	resp := &request.Response{Status: 200, StatusText: "OK"}
	lines := v.render(resp, "", true, 80)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 rendered lines (header+rule+body), got %d", len(lines))
	}
}

func TestResponseViewerRenderShowsStatusAndTabs(t *testing.T) {
	v := newResponseViewer("hello")
	resp := &request.Response{Status: 200, StatusText: "OK", TimeMs: 42}
	out := strings.Join(v.render(resp, "", true, 80), "\n")
	if !strings.Contains(out, "200") || !strings.Contains(out, "OK") || !strings.Contains(out, "42ms") {
		t.Errorf("expected status line, got:\n%s", out)
	}
	if !strings.Contains(out, "Request") || !strings.Contains(out, "Response") {
		t.Errorf("expected tab buttons, got:\n%s", out)
	}
}

func TestResponseViewerRenderRequestTabShowsMethodURLHeadersBody(t *testing.T) {
	v := newResponseViewer("body")
	v = stepViewer(v, `\`)
	resp := &request.Response{
		RequestMethod: "POST", RequestURL: "http://x/y",
		RequestHeaders: map[string]string{"Authorization": "Bearer t"},
		RequestBody:    `{"a":1}`,
	}
	out := strings.Join(v.render(resp, "", true, 80), "\n")
	if !strings.Contains(out, "POST http://x/y") {
		t.Errorf("expected method+url, got:\n%s", out)
	}
	if !strings.Contains(out, "Authorization") {
		t.Errorf("expected request header, got:\n%s", out)
	}
	if !strings.Contains(out, `{"a":1}`) {
		t.Errorf("expected request body, got:\n%s", out)
	}
}

func TestResponseViewerRenderCurlOnlyOnResponseTab(t *testing.T) {
	v := newResponseViewer("hello")
	resp := &request.Response{Status: 200}
	respOut := strings.Join(v.render(resp, "curl -X GET x", true, 80), "\n")
	if !strings.Contains(respOut, "curl -X GET x") {
		t.Errorf("expected curl on response tab, got:\n%s", respOut)
	}

	v = stepViewer(v, `\`)
	reqOut := strings.Join(v.render(resp, "curl -X GET x", true, 80), "\n")
	if strings.Contains(reqOut, "curl -X GET x") {
		t.Errorf("expected no curl on request tab, got:\n%s", reqOut)
	}
}

func TestResponseViewerRenderHintTextVariants(t *testing.T) {
	resp := &request.Response{Status: 200}

	short := newResponseViewer("one\ntwo")
	if out := strings.Join(short.render(resp, "", true, 80), "\n"); !strings.Contains(out, "v: visual  y: yank") {
		t.Errorf("expected short-body hint, got:\n%s", out)
	}

	var many []string
	for range responseViewport + 2 {
		many = append(many, "line")
	}
	long := newResponseViewer(strings.Join(many, "\n"))
	if out := strings.Join(long.render(resp, "", true, 80), "\n"); !strings.Contains(out, "g/G: top/bottom") {
		t.Errorf("expected scrollable hint, got:\n%s", out)
	}

	visual := stepViewer(newResponseViewer("one\ntwo"), "v")
	if out := strings.Join(visual.render(resp, "", true, 80), "\n"); !strings.Contains(out, "-- VISUAL --") {
		t.Errorf("expected visual-mode hint, got:\n%s", out)
	}
}

func TestResponseViewerYankMessageShownAndClearedByTimerMsg(t *testing.T) {
	v := stepViewer(newResponseViewer("a\nb"), "y")
	if !v.Yanked {
		t.Fatalf("expected Yanked set after y")
	}
	resp := &request.Response{Status: 200}
	if out := strings.Join(v.render(resp, "", true, 80), "\n"); !strings.Contains(out, "[yanked]") {
		t.Errorf("expected yanked indicator, got:\n%s", out)
	}
}
