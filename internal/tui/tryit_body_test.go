package tui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

// focusBody presses 'j' enough times to walk off the last parameter row and
// land on the BODY section, matching the real navigation path a user takes.
func focusBody(m Model) Model {
	params := sortedParameters(m.selectedItem().Endpoint.Operation.Parameters)
	for range len(params) + 1 {
		m = step(m, "j")
	}
	return m
}

func TestEnterTryItScaffoldsBodyWithFakeData(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")

	if m.TryIt.Body == "" {
		t.Fatalf("expected body auto-scaffolded on entering try-it-out")
	}
	var decoded any
	if err := json.Unmarshal([]byte(m.TryIt.Body), &decoded); err != nil {
		t.Errorf("expected valid JSON, got %q: %v", m.TryIt.Body, err)
	}
}

func TestEnterTryItPrefersSavedOverrideBody(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	ep := m.selectedItem().Endpoint
	if err := m.Store.SaveEndpointOverride(string(ep.Method), ep.Path, storage.EndpointOverride{
		Params: map[string]string{}, CustomParams: []storage.CustomParameter{}, DisabledParams: []string{},
		Body: `{"saved":"value"}`,
	}); err != nil {
		t.Fatal(err)
	}

	m = step(m, "t")
	if m.TryIt.Body != `{"saved":"value"}` {
		t.Errorf("expected saved override body to win over auto-scaffold, got %q", m.TryIt.Body)
	}
}

func TestJPastLastParamFocusesBody(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	if !m.TryIt.BodyFocused {
		t.Fatalf("expected BodyFocused after walking off the last param row")
	}
}

func TestBodyFocusIEntersEditMode(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	if !m.TryIt.EditingBody {
		t.Fatalf("expected EditingBody after 'i'")
	}
}

func TestBodyEditingTypesIntoTextarea(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	before := m.TryIt.Body
	m = typeText(m, "X")
	if m.TryIt.Body == before {
		t.Errorf("expected typed character to change the body")
	}
	if !strings.Contains(m.TryIt.BodyInput.Value(), "X") {
		t.Errorf("expected textarea to contain typed character, got %q", m.TryIt.BodyInput.Value())
	}
}

func TestBodyEditEscStopsEditingWithoutExitingTryIt(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	m = step(m, "esc")
	if m.TryIt.EditingBody {
		t.Errorf("expected editing to stop")
	}
	if m.Mode != ModeTryIt {
		t.Errorf("expected to remain in try-it-out after Esc from body editing, got mode %v", m.Mode)
	}
}

func TestBodyFocusEscExitsTryItEntirely(t *testing.T) {
	// A real TS quirk (see handleBodyFocusedKey's doc comment): Esc while
	// body-focused but not editing exits the whole try-it-out session, not
	// just the body focus.
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "esc")
	if m.Mode != ModeBrowse {
		t.Errorf("expected Esc from body-focused (not editing) to exit try-it-out, got mode %v", m.Mode)
	}
}

func TestBodyFocusKUnfocusesBackToParams(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "k")
	if m.TryIt.BodyFocused {
		t.Errorf("expected 'k' to unfocus the body")
	}
	if m.Mode != ModeTryIt {
		t.Errorf("expected to remain in try-it-out, got mode %v", m.Mode)
	}
}

func TestExitTryItPersistsBodyWithoutExecuting(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	ep := m.selectedItem().Endpoint
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	m = typeText(m, "X")
	m = step(m, "esc") // stop editing (still focused on body)
	m = step(m, "esc") // now exits try-it-out entirely (see quirk test above)

	if m.Mode != ModeBrowse {
		t.Fatalf("expected browse mode after second Esc")
	}
	override := m.Store.GetEndpointOverride(string(ep.Method), ep.Path)
	if override == nil || !strings.Contains(override.Body, "X") {
		t.Errorf("expected body persisted to the override on exit without executing, got %+v", override)
	}
}

func TestResetOverrideClearsBody(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	if m.TryIt.Body == "" {
		t.Fatalf("setup: expected a scaffolded body")
	}
	m = step(m, "r", "y")
	if m.TryIt.Body != "" {
		t.Errorf("expected reset to clear the body, got %q", m.TryIt.Body)
	}
}

func TestExecuteSendsTheEditedBodyVerbatim(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := endpointWithBodyModel(t)
	m.Spec.Spec.Servers = []openapi.Server{{URL: srv.URL}}
	m = m.WithServices(srv.Client(), newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	m.TryIt.BodyInput.SetValue(`{"hand":"edited"}`)
	m.TryIt.Body = m.TryIt.BodyInput.Value()
	m = step(m, "esc") // stop editing, body stays focused

	next, cmd := m.Update(key("e"))
	if cmd == nil {
		t.Fatalf("expected an execute command")
	}
	m = next.(Model)
	msg := cmd()
	m2, _ := m.Update(msg)
	m = m2.(Model)
	if m.Viewer.Response == nil || m.Viewer.Response.Status != 200 {
		t.Fatalf("expected a successful response, got %+v", m.Viewer.Response)
	}
	if gotBody != `{"hand":"edited"}` {
		t.Errorf("expected the hand-edited body sent verbatim, got %q", gotBody)
	}
}

func TestQuitGuardedWhileEditingBody(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")
	m = typeText(m, "q")
	if !m.TryIt.EditingBody || m.Quitting {
		t.Errorf("expected 'q' to type into the body, not quit")
	}
}

// TestRenderTryItLinesEveryElementIsOneTerminalRow is a regression test for
// the real bug behind "pressing 't' jumps the content to the bottom and it
// can't be scrolled back": the BODY section's bordered box is a single
// lipgloss.Render() call that returns one string containing many embedded
// newlines. Appending that whole string as ONE element of the flat
// []string that renderRightPanel's scroll/pad math treats as "one element
// == one terminal row" silently under-counted the real rendered height —
// the actual output written to the terminal ended up taller than the
// panel's row budget, so the real terminal (not any app-level scroll
// offset) scrolled natively, pushing the top of the frame off screen with
// no way back. Every element returned here must be exactly one row.
func TestRenderTryItLinesEveryElementIsOneTerminalRow(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	if m.TryIt.Body == "" {
		t.Fatalf("setup: expected a scaffolded body")
	}

	lines, _ := m.renderTryItLines(m.selectedItem().Endpoint, 100)
	for i, l := range lines {
		if strings.Contains(l, "\n") {
			t.Fatalf("line %d contains an embedded newline (renders as %d real rows, not 1): %q", i, strings.Count(l, "\n")+1, l)
		}
	}
}

// TestBodyEditingShowsDoneHint is a regression test: the "Enter: newline
// Esc: done" hint below the body textarea was missing entirely while
// editing (RightPanel.tsx shows an equivalent hint under its own TextArea
// whenever editingBody is true — this rewrite had the border/content but
// dropped the hint line).
func TestBodyEditingShowsDoneHint(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	m = focusBody(m)
	m = step(m, "i")

	lines, _ := m.renderTryItLines(m.selectedItem().Endpoint, 150)
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "Esc: done") {
		t.Errorf("expected an 'Esc: done' hint while editing the body, got:\n%s", out)
	}
}

// TestBodyEditingHintNotShownWhenIdle matches TS: no hint renders once the
// body is populated and not being edited (RightPanel.tsx's hint text only
// exists inside the `!editingBody && !body` branch).
func TestBodyEditingHintNotShownWhenIdle(t *testing.T) {
	m := endpointWithBodyModel(t)
	m = m.WithServices(nil, newTestStore(t))
	m = step(m, "t")
	if m.TryIt.Body == "" {
		t.Fatalf("setup: expected a scaffolded body")
	}

	lines, _ := m.renderTryItLines(m.selectedItem().Endpoint, 150)
	out := strings.Join(lines, "\n")
	if strings.Contains(out, "Esc: done") || strings.Contains(out, "j: focus") {
		t.Errorf("expected no body hint while idle with a populated body, got:\n%s", out)
	}
}
