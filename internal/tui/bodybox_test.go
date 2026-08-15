package tui

import (
	"strings"
	"testing"
)

func TestRenderBodyHeadingShowsCycleHintOnlyWhenMultipleTypesAndFocused(t *testing.T) {
	types := []string{"application/json", "application/xml"}
	unfocused := strings.Join(renderBodyHeading(types, "application/json", false, false), "\n")
	if strings.Contains(unfocused, "c:cycle") {
		t.Errorf("expected no cycle hint while unfocused, got %q", unfocused)
	}
	focused := strings.Join(renderBodyHeading(types, "application/json", true, false), "\n")
	if !strings.Contains(focused, "c:cycle") {
		t.Errorf("expected a cycle hint while focused with >1 type, got %q", focused)
	}
	single := strings.Join(renderBodyHeading([]string{"application/json"}, "application/json", true, false), "\n")
	if strings.Contains(single, "c:cycle") {
		t.Errorf("expected no cycle hint with only one type, got %q", single)
	}
}

func TestRenderBodyHeadingMarksRequired(t *testing.T) {
	got := strings.Join(renderBodyHeading(nil, "", false, true), "\n")
	if !strings.Contains(got, "*") {
		t.Errorf("expected a required marker, got %q", got)
	}
}

func TestRenderBodyBoxUnfocusedEmptyShowsPlaceholderAndUnfocusedHint(t *testing.T) {
	lines := renderBodyBox(bodyBoxState{
		Width:              40,
		EmptyLines:         []string{"placeholder"},
		EmptyHintUnfocused: "j: focus",
		EmptyHintFocused:   "i: edit",
	})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "placeholder") {
		t.Errorf("expected the placeholder content rendered, got %q", joined)
	}
	if !strings.Contains(joined, "j: focus") || strings.Contains(joined, "i: edit") {
		t.Errorf("expected the unfocused hint, not the focused one, got %q", joined)
	}
}

func TestRenderBodyBoxFocusedEmptyShowsFocusedHint(t *testing.T) {
	lines := renderBodyBox(bodyBoxState{
		Width:              40,
		Focused:            true,
		EmptyHintUnfocused: "j: focus",
		EmptyHintFocused:   "i: edit",
	})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "i: edit") || strings.Contains(joined, "j: focus") {
		t.Errorf("expected the focused hint, not the unfocused one, got %q", joined)
	}
}

func TestRenderBodyBoxFocusedWithContentShowsEditHint(t *testing.T) {
	lines := renderBodyBox(bodyBoxState{Width: 40, Focused: true, Body: `{"a":1}`})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, `{"a":1}`) || !strings.Contains(joined, "i: edit") {
		t.Errorf("expected the body content plus an edit hint, got %q", joined)
	}
}

func TestRenderBodyBoxEditingShowsTextareaAndDoneHint(t *testing.T) {
	ta := newBodyTextarea()
	ta.SetValue("hello")
	lines := renderBodyBox(bodyBoxState{Width: 40, Editing: true, BodyInput: ta})
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hello") {
		t.Errorf("expected the textarea's content rendered, got %q", joined)
	}
	if !strings.Contains(joined, "Enter: newline") {
		t.Errorf("expected the editing hint, got %q", joined)
	}
}

func TestRenderBodyBoxUnfocusedWithContentShowsNoHint(t *testing.T) {
	lines := renderBodyBox(bodyBoxState{Width: 40, Body: `{"a":1}`})
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "i: edit") {
		t.Errorf("expected no edit hint while neither focused nor editing, got %q", joined)
	}
}

func TestRenderBodyBoxSplitsIntoOneLinePerRow(t *testing.T) {
	lines := renderBodyBox(bodyBoxState{Width: 40, Body: "line1\nline2"})
	for _, l := range lines {
		if strings.Contains(l, "\n") {
			t.Errorf("expected every returned element to be exactly one terminal row, got %q", l)
		}
	}
}
