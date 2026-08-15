package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

// handleBrowseKey is browse mode's Update — kept as a Model method rather
// than a value-type component (same call as TryIt/Manual/LeftPanel, see
// their doc comments): TagDeleteConfirm/ActivePanel/Viewer are root-shared
// state, not something a self-contained Browse value could own without
// duplicating it. Only reached once every other mode's own Update has
// already returned (see handleKey's Mode switch) — everything below is
// unconditionally browse-only, matching useAppKeyboard.ts's browse-mode
// handler.
func (m Model) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Matches useAppKeyboard.ts's tagDeleteConfirm intercept: takes over
	// input entirely until y/n/Esc resolves it.
	if m.TagDeleteConfirm != "" {
		switch key {
		case "y":
			if m.Store != nil {
				m.Store.DeleteCustomTag(m.TagDeleteConfirm)
			}
			m.TagDeleteConfirm = ""
			return m.refreshSavedRequests(), nil
		case "n", "esc":
			m.TagDeleteConfirm = ""
			return m, nil
		}
		return m, nil
	}

	// R/D on a custom tag, E/D on a saved request, and 'e' quick-execute all
	// work regardless of which panel is active, matching
	// useAppKeyboard.ts's browse-mode handler (checked ahead of h/l/j/k
	// navigation) — 'e' in particular is bound in a useInput with
	// isActive: mode === 'browse' only, no panel check, so it must work
	// from the left panel too, not just after pressing 'l' first.
	if item := m.selectedItem(); item != nil {
		switch {
		case key == "R" && item.Type == ItemTag && m.isCustomTag(item.TagName):
			return m.enterRenameTag(item.TagName), nil
		case key == "D" && item.Type == ItemTag && m.isCustomTag(item.TagName):
			m.TagDeleteConfirm = item.TagName
			return m, nil
		case key == "E" && item.Type == ItemSavedRequest:
			return m.enterManualEdit(item.SavedRequest), nil
		case key == "D" && item.Type == ItemSavedRequest:
			if m.Store != nil {
				m.Store.DeleteSavedRequest(item.SavedRequest.ID)
			}
			return m.refreshSavedRequests(), nil
		case key == "e" && item.Type == ItemEndpoint:
			cmd := m.quickExecuteCmd(item.Endpoint)
			m.Loading = true
			return m, cmd
		case key == "e" && item.Type == ItemSavedRequest:
			cmd := m.savedRequestExecuteCmd(item.SavedRequest)
			m.Loading = true
			return m, cmd
		}
	}

	switch key {
	case "ctrl+r":
		m.SpecLoading = true
		return m, m.reloadCmd()
	case "?":
		m.ShowHelp = true
		m.Help = helpPopupState{}
		return m, nil
	case "i":
		return m.enterInfo(), nil
	case "h", "left":
		m.ActivePanel = PanelLeft
		return m, nil
	case "l", "right":
		m.ActivePanel = PanelRight
		return m, nil
	case "[":
		m.LeftExpanded = !m.LeftExpanded
		return m, nil
	case "t":
		if item := m.selectedItem(); item != nil && item.Type == ItemEndpoint {
			return m.enterTryIt(), nil
		}
		return m, nil
	case "m":
		return m.enterManualNew(), nil
	}

	if m.ActivePanel == PanelLeft {
		return m.handleLeftPanelKey(key)
	}

	// Response-viewer keys (J/K/G/v/y/Esc/\) take priority over generic
	// scroll (j/k/g) when a response is present — distinct key casing means
	// both coexist without a mode flag, matching the TS app. Lowercase 'g'
	// is bound by both ResponseViewer.tsx (jump response cursor to top) and
	// usePanelNavigation.ts (reset panel scroll) as two independent Ink
	// input handlers that both fire on the same keypress — replicated here
	// by routing to the viewer and then still falling through to the
	// generic handler below, rather than picking one.
	if m.Viewer.Response != nil {
		switch key {
		case "J", "K", "G", "v", "y", "esc", `\`:
			var cmd tea.Cmd
			m.Viewer, cmd = m.Viewer.handleKey(key)
			return m, cmd
		case "C":
			// Go-only addition, not a TS port — yanks the generated curl
			// command to the clipboard, independent of tab/selection state.
			if m.Viewer.Curl != "" {
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.yankCurl(m.Viewer.Curl)
				return m, cmd
			}
		case "j", "k":
			// While actively visual-selecting, lowercase j/k also drive the
			// response cursor instead of the outer panel scroll — found via
			// a user report: without this, pressing 'v' then reaching for
			// the muscle-memory 'j'/'k' (rather than the shifted 'J'/'K'
			// the hint text actually asks for) just scrolls the panel out
			// from under the selection, which looks exactly like "can't
			// expand the selection, it just moves the viewport." Only
			// active during a selection — outside of one, lowercase j/k
			// keeps its normal job of reaching content that might be
			// scrolled out of view above the response section.
			if m.Viewer.Selecting {
				viewerKey := "J"
				if key == "k" {
					viewerKey = "K"
				}
				var cmd tea.Cmd
				m.Viewer, cmd = m.Viewer.handleKey(viewerKey)
				return m, cmd
			}
		case "g":
			m.Viewer, _ = m.Viewer.handleKey(key)
		}
	}

	return m.handleRightPanelKey(key)
}

// handleRightPanelKey handles plain scroll/tab-cycle keys once nothing
// above (item actions, response-viewer keys) has already claimed the
// keystroke.
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
					lines = append(lines, colorizeSchemaLine(l))
				}
			}
		}
	}

	lines = append(lines, renderResponseTabs(op, m.ResponseTab, active)...)
	lines = append(lines, m.renderResponseBlock(width)...)

	return lines
}
