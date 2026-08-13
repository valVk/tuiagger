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

// TestYankCurlSetsDistinctFlagFromBodyYank is a Go-only addition (not a TS
// port): 'C' copies the generated curl command independent of whatever
// selection/tab state the body viewer is in — YankedCurl is a separate flag
// from Yanked so the two "[...]" indicators never fight over one timer.
func TestYankCurlSetsDistinctFlagFromBodyYank(t *testing.T) {
	v := newResponseViewer("a\nb\nc")
	v = stepViewer(v, "v", "J") // start a body selection first
	next, cmd := v.yankCurl("curl -X GET https://example.com")
	if !next.YankedCurl {
		t.Errorf("expected YankedCurl set")
	}
	if next.Yanked {
		t.Errorf("expected the unrelated body-yank flag to stay false")
	}
	if !next.Selecting {
		t.Errorf("expected yanking curl to leave an in-progress body selection untouched")
	}
	if cmd == nil {
		t.Errorf("expected yankCurl to return a batched clipboard+timer command")
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
	respOut := stripANSI(strings.Join(v.render(resp, "curl -X GET x", true, 80), "\n"))
	if !strings.Contains(respOut, "curl -X GET x") {
		t.Errorf("expected curl on response tab, got:\n%s", respOut)
	}

	v = stepViewer(v, `\`)
	reqOut := stripANSI(strings.Join(v.render(resp, "curl -X GET x", true, 80), "\n"))
	if strings.Contains(reqOut, "curl -X GET x") {
		t.Errorf("expected no curl on request tab, got:\n%s", reqOut)
	}
}

// TestResponseViewerRenderCurlNoEmbeddedNewlines is a regression test:
// GenerateCurl (curl.go) joins its parts with " \\\n", so a curl command
// with headers or a body is always a multi-line string. Appending it as one
// []string element (as the code briefly did) under-counts its real height
// in renderRightPanel's scroll/pad math, overflowing the terminal — the
// same bug class fixed for renderTryItBodySection's box and the manual
// builder's BODY box, just a third occurrence that was missed the first
// time. Every returned element must be exactly one row.
func TestResponseViewerRenderCurlNoEmbeddedNewlines(t *testing.T) {
	v := newResponseViewer("hello")
	resp := &request.Response{Status: 200}
	curl := "curl -X 'POST' \\\n  'http://example.com' \\\n  -H 'Accept: application/json' \\\n  -d '{\n  \"a\": 1\n}'"
	lines := v.render(resp, curl, true, 80)
	for i, l := range lines {
		if strings.Contains(l, "\n") {
			t.Fatalf("line %d contains an embedded newline (renders as %d real rows, not 1): %q", i, strings.Count(l, "\n")+1, l)
		}
	}
}

// TestResponseViewerRenderCurlContentPreservedInOrder guards against a
// sloppy fix for the bug above: splitting the curl string into separate
// lines must not lose or reorder any of its content (e.g. an off-by-one on
// the "curl: " prefix, or strings.Split silently dropping an empty line).
func TestResponseViewerRenderCurlContentPreservedInOrder(t *testing.T) {
	v := newResponseViewer("body")
	resp := &request.Response{Status: 200}
	curl := "curl -X 'POST' \\\n  'http://example.com' \\\n  -H 'Accept: application/json' \\\n  -d '{\n  \"a\": 1\n}'"
	lines := v.render(resp, curl, true, 80)
	for i, l := range lines {
		lines[i] = stripANSI(l)
	}

	want := strings.Split(curl, "\n")
	var got []string
	for _, l := range lines {
		for _, w := range want {
			if strings.Contains(l, w) {
				got = append(got, w)
				break
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("expected all %d curl fragments present in order, found %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fragment %d out of order: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestResponseViewerRenderCurlSectionLayout matches a user request: curl
// gets its own section — a blank separator, a "CURL" heading (with a
// "C: yank curl" hint, matching the RESPONSE/REQUEST status line's own
// pattern), a rule, then the syntax-colored command — instead of a single
// dim "curl: ..." line glued directly to whatever precedes it.
func TestResponseViewerRenderCurlSectionLayout(t *testing.T) {
	v := newResponseViewer("hello")
	resp := &request.Response{Status: 200}
	lines := v.render(resp, "curl -X GET https://example.com", true, 80)
	for i, l := range lines {
		lines[i] = stripANSI(l)
	}

	headingIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "CURL") {
			headingIdx = i
			break
		}
	}
	if headingIdx <= 0 {
		t.Fatalf("expected a CURL heading after at least one preceding line, got index %d in:\n%s", headingIdx, strings.Join(lines, "\n"))
	}
	if lines[headingIdx-1] != "" {
		t.Errorf("expected a blank line directly before the CURL heading, got %q", lines[headingIdx-1])
	}
	if !strings.Contains(lines[headingIdx], "C: yank curl") {
		t.Errorf("expected the CURL heading to show the yank-curl hint, got %q", lines[headingIdx])
	}
	if !strings.Contains(lines[headingIdx+1], "───") {
		t.Errorf("expected a rule directly under the CURL heading, got %q", lines[headingIdx+1])
	}
	if !strings.Contains(lines[headingIdx+2], "curl -X GET https://example.com") {
		t.Errorf("expected the curl command directly under the rule, got %q", lines[headingIdx+2])
	}
}

// TestResponseViewerRenderCurlHeadingShowsYankedIndicator matches the same
// "[curl yanked]" pattern the main RESPONSE/REQUEST status line uses for
// body yanks.
func TestResponseViewerRenderCurlHeadingShowsYankedIndicator(t *testing.T) {
	v := newResponseViewer("hello")
	v.YankedCurl = true
	resp := &request.Response{Status: 200}
	out := stripANSI(strings.Join(v.render(resp, "curl -X GET x", true, 80), "\n"))
	if !strings.Contains(out, "CURL") || !strings.Contains(out, "[curl yanked]") {
		t.Errorf("expected the CURL heading to show [curl yanked], got:\n%s", out)
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

// TestLowercaseJKExtendsSelectionInBrowseMode is a regression test: a user
// reported that after pressing 'v' to start a selection, reaching for the
// muscle-memory lowercase 'j'/'k' (instead of the shifted 'J'/'K' the hint
// text asks for) just scrolled the panel out from under the selection
// instead of extending it — "I just move the viewport up and down." While
// actively selecting, lowercase j/k must drive the response cursor, not
// m.RightScroll.
func TestLowercaseJKExtendsSelectionInBrowseMode(t *testing.T) {
	m := firstEndpointModel(t)
	m = step(m, "l") // focus right panel
	next, _ := m.Update(responseMsg{response: &request.Response{Status: 200, Body: "l0\nl1\nl2\nl3\nl4"}})
	m = next.(Model)

	m = step(m, "v")
	beforeScroll := m.RightScroll
	m = step(m, "j", "j")
	if m.RightScroll != beforeScroll {
		t.Errorf("expected lowercase j to extend the selection, not scroll the panel (RightScroll changed from %d to %d)", beforeScroll, m.RightScroll)
	}
	if m.Viewer.Cursor != 2 {
		t.Errorf("expected lowercase j to move the response cursor to 2, got %d", m.Viewer.Cursor)
	}
	if got := m.Viewer.selectedText(); got != "l0\nl1\nl2" {
		t.Errorf("expected 3-line selection via lowercase j, got %q", got)
	}

	// Outside of an active selection, lowercase j must still scroll the
	// outer panel as before — this fix should not touch normal browsing.
	m = step(m, "esc") // cancel selection
	m.RightScroll = 0
	m = step(m, "j")
	if m.RightScroll != 1 {
		t.Errorf("expected lowercase j to scroll the panel when not selecting, got RightScroll=%d", m.RightScroll)
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
