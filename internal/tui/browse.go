package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

// handleRightPanelKey is browse mode's Update — kept as a Model method
// rather than a value-type component (same call as TryIt/Manual/
// LeftPanel, see their doc comments): RightScroll/ResponseTab are shared
// with try-it-out's own right-panel scroll handling, and the selected
// item comes from the root-owned FlatList, not state a self-contained
// Browse value could own without duplicating it.
func (m Model) handleRightPanelKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.RightScroll++
		return m, nil
	case "k", "up":
		m.RightScroll = max(0, m.RightScroll-1)
		return m, nil
	case "g":
		m.RightScroll = 0
		return m, nil
	case "/":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			codes := sortedResponseCodes(item.Endpoint)
			if len(codes) > 0 {
				m.ResponseTab = (m.ResponseTab + 1) % len(codes)
			}
		}
		return m, nil
	}
	return m, nil
}

func sortedResponseCodes(ep *openapi.ParsedEndpoint) []string {
	var codes []string
	for _, r := range ep.Operation.Responses {
		codes = append(codes, r.Status)
	}
	sort.Strings(codes)
	return codes
}

// renderTagLines is browse mode's View for a selected tag row: its
// endpoint list, matching RightPanel.tsx's tag-selected branch.
func (m Model) renderTagLines(tagName string, width int) []string {
	desc := " — Enter: expand/collapse"
	if m.isCustomTag(tagName) {
		desc += "  R: rename  D: delete"
	}
	lines := []string{boldStyle.Render(tagName), dimStyle.Render(desc), ""}
	for _, ep := range m.EndpointsByTag[tagName] {
		methodStyle := lipgloss.NewStyle().Foreground(MethodColor(string(ep.Method))).Bold(true)
		lines = append(lines, methodStyle.Render(padRight(strings.ToUpper(string(ep.Method)), 7))+truncate(ep.Path, max(width-8, 4)))
	}
	return lines
}

// renderEndpointLines is browse mode's View for a selected endpoint: the
// read-only doc display (summary/description/parameters/body/responses),
// matching RightPanel.tsx's default (non-try-it) branch.
func (m Model) renderEndpointLines(ep *openapi.ParsedEndpoint, active bool, width int) []string {
	op := ep.Operation
	var lines []string

	lines = append(lines, MethodBadge(string(ep.Method))+" "+boldStyle.Render(ep.Path))
	lines = append(lines, dimStyle.Render(strings.Repeat("─", width)))

	// No blank line before this banner — RightPanel.tsx's Box has no
	// marginTop, sitting directly under the heading's border-bottom (same
	// fix as the try-it-out button row, see renderTryItLines).
	saved := ""
	if m.Store != nil {
		if o := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); o != nil {
			saved = yellowStyle.Render(" *saved params")
		}
	}
	// Matches RightPanel.tsx's hand-composed banner exactly — brackets
	// touch directly ("][", no gap between the two buttons, unlike every
	// other button row in the app which goes through renderButtons' " "
	// join) and "*saved params" sits inside the second bracket.
	banner := dimStyle.Render("[ ") + cyanStyle.Render("Try it out (t)") + dimStyle.Render(" ][ ") +
		greenBoldStyle.Render("Quick execute (e)") + saved + dimStyle.Render(" ]")
	lines = append(lines, lipgloss.NewStyle().Width(width).Align(lipgloss.Right).Render(banner))

	if op.Summary != "" {
		lines = append(lines, boldStyle.Render(op.Summary))
	}
	if op.Description != "" {
		lines = append(lines, wrapLines(openapi.HTMLToPlainText(op.Description), width)...)
	}
	if op.Deprecated {
		lines = append(lines, yellowStyle.Bold(true).Render("DEPRECATED"))
	}

	if len(op.Parameters) > 0 {
		lines = append(lines, "", boldStyle.Render("PARAMETERS"))
		lines = append(lines, paramTableHeader())
		lines = append(lines, dimStyle.Render(strings.Repeat("─", width)))
		for _, p := range sortedParameters(op.Parameters) {
			lines = append(lines, renderParamRow(paramRowState{param: p}, width)...)
		}
	}

	if op.RequestBody != nil {
		heading := boldStyle.Render("BODY") + " " + dimStyle.Render(contentTypesOf(op.RequestBody.Content))
		if op.RequestBody.Required {
			heading += lipgloss.NewStyle().Foreground(color5xx).Render(" *")
		}
		lines = append(lines, "", heading)
		// Matches RightPanel.tsx's non-try-it BODY block: a saved override's
		// actual body (green, 1-space indent) takes priority over the plain
		// schema shape (dim) — a leftover try-it-out session's body is
		// visible from browse mode too, not just while try-it is open.
		savedBody := ""
		if m.Store != nil {
			if o := m.Store.GetEndpointOverride(string(ep.Method), ep.Path); o != nil {
				savedBody = o.Body
			}
		}
		bodyStyle := lipgloss.NewStyle().Foreground(color2xx)
		switch {
		case savedBody != "":
			for l := range strings.SplitSeq(savedBody, "\n") {
				lines = append(lines, " "+bodyStyle.Render(l))
			}
		default:
			if schema := firstSchema(op.RequestBody.Content); schema != nil {
				for l := range strings.SplitSeq(openapi.FormatSchema(schema, 0), "\n") {
					lines = append(lines, dimStyle.Render(l))
				}
			}
		}
	}

	lines = append(lines, renderResponseTabs(op, m.ResponseTab, active)...)
	lines = append(lines, m.renderResponseBlock(width)...)

	return lines
}
