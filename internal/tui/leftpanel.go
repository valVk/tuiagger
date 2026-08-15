package tui

import (
	"fmt"
	"maps"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// safeLeftIndex clamps LeftIndex to a valid FlatList index — the flat list
// can shrink out from under a stale index (tag collapse, saved-request
// delete), so every read goes through here rather than LeftIndex directly.
func (m Model) safeLeftIndex() int {
	if len(m.FlatList) == 0 {
		return 0
	}
	if m.LeftIndex >= len(m.FlatList) {
		return len(m.FlatList) - 1
	}
	return m.LeftIndex
}

func (m Model) selectedItem() *FlatListItem {
	if len(m.FlatList) == 0 {
		return nil
	}
	return &m.FlatList[m.safeLeftIndex()]
}

// handleLeftPanelKey is the left panel's Update — kept as a Model method
// rather than a value-type component (same call as TryIt/Manual, see
// tryit_keys.go's doc comment): LeftIndex/FlatList/ExpandedTags are
// root-shared state (RightScroll/ResponseTab reset here belong to the
// right panel, FlatList is rebuilt from Spec/Store-derived data every
// mode reads), not state a self-contained LeftPanel value could own
// without duplicating it.
func (m Model) handleLeftPanelKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.LeftIndex = min(m.safeLeftIndex()+1, len(m.FlatList)-1)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "k", "up":
		m.LeftIndex = max(m.safeLeftIndex()-1, 0)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "g":
		m.LeftIndex = 0
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "G":
		m.LeftIndex = max(0, len(m.FlatList)-1)
		m.RightScroll = 0
		m.ResponseTab = 0
		return m, nil
	case "enter":
		if item := m.selectedItem(); item != nil && item.Type == ItemTag {
			m.toggleTag(item.TagName)
		}
		return m, nil
	case "c":
		m.ExpandedTags = make(map[string]bool)
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
		m.LeftIndex = 0
		return m, nil
	case "x":
		m.ExpandedTags = allExpanded(m.AllTags)
		m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
		return m, nil
	}
	return m, nil
}

func (m *Model) toggleTag(tagName string) {
	next := make(map[string]bool, len(m.ExpandedTags))
	maps.Copy(next, m.ExpandedTags)
	next[tagName] = !next[tagName]
	m.ExpandedTags = next
	m.FlatList = buildFlatList(m.AllTags, m.EndpointsByTag, m.SavedRequestsByTag, m.ExpandedTags)
}

func allExpanded(tags []string) map[string]bool {
	m := make(map[string]bool, len(tags))
	for _, t := range tags {
		m[t] = true
	}
	return m
}

// renderLeftPanel is the left panel's View — matches LeftPanel.tsx: a
// bordered box with title/rule, a scrollable window of flattened tag/
// endpoint/saved-request rows centered on the selection, and a position
// footer once the list overflows the box.
func (m Model) renderLeftPanel(height, width int) string {
	active := m.ActivePanel == PanelLeft
	borderStyle, borderColor := panelBorderStyle(active)

	// boxInner is the box's full interior width (border only, no padding);
	// inner further subtracts the 1-space margin Padding(0,1) adds — the
	// budget most row renderers below actually receive. The title's rule
	// line is built to span boxInner exactly (edge-to-edge, no margin),
	// matching LeftPanel.tsx's border-bottom Box, which spans full-width
	// while the title/rows above and below it each carry their own
	// paddingX={1} — one flat Padding() on this whole box can't reproduce
	// that (it pads every line uniformly), so the title/rule pair is
	// hand-built here instead of flowing through the outer Style's Padding.
	boxInner := max(width-2, 1)
	inner := max(width-4, 1)

	suffix := ""
	if active {
		suffix = dimStyle.Render("(i)")
	}
	titleText := boldStyle.Render("ENDPOINTS")
	if m.Spec != nil {
		titleText = boldStyle.Render(truncate(m.Spec.Spec.Info.Title, inner))
	}
	gap := max(inner-lipgloss.Width(titleText)-lipgloss.Width(suffix), 0)
	titleLine := " " + titleText + strings.Repeat(" ", gap) + suffix + " "
	titleRule := dimStyle.Render(strings.Repeat("─", boxInner))

	visibleHeight := max(height-5, 1)
	selected := m.safeLeftIndex()
	startIndex := 0
	if len(m.FlatList) > visibleHeight {
		half := visibleHeight / 2
		switch {
		case selected < half:
			startIndex = 0
		case selected > len(m.FlatList)-half-1:
			startIndex = len(m.FlatList) - visibleHeight
		default:
			startIndex = selected - half
		}
	}
	end := min(startIndex+visibleHeight, len(m.FlatList))

	var lines []string
	for i := startIndex; i < end; i++ {
		lines = append(lines, m.renderListRow(m.FlatList[i], i == selected, inner))
	}
	for len(lines) < visibleHeight {
		lines = append(lines, "")
	}

	footer := ""
	if len(m.FlatList) > visibleHeight {
		footer = dimStyle.Render(fmt.Sprintf("%d/%d", selected+1, len(m.FlatList)))
	}

	rowsBlock := lipgloss.NewStyle().Width(boxInner).Padding(0, 1).
		Render(strings.Join(lines, "\n") + "\n" + footer)

	content := titleLine + "\n" + titleRule + "\n" + rowsBlock

	return lipgloss.NewStyle().
		Width(boxInner).
		Height(height).
		BorderStyle(borderStyle).
		BorderForeground(borderColor).
		Render(content)
}

// renderListRow matches LeftPanel.tsx's row rendering: tag rows show an
// expand/collapse arrow + item count; endpoint rows prefix a '~' when a
// saved override exists for that endpoint (path/method/body/params), and
// highlight the path (not the whole row) when selected.
func (m Model) renderListRow(item FlatListItem, selected bool, width int) string {
	if item.Type == ItemTag {
		arrow := "▶"
		if m.ExpandedTags[item.TagName] {
			arrow = "▼"
		}
		count := len(m.EndpointsByTag[item.TagName]) + len(m.SavedRequestsByTag[item.TagName])
		style := boldStyle
		if selected {
			style = style.Foreground(lipgloss.Color("6")).Reverse(true)
		}
		return style.Render(truncate(arrow+" "+item.TagName, width)) + dimStyle.Render(fmt.Sprintf(" (%d)", count))
	}

	if item.Type == ItemSavedRequest {
		sr := item.SavedRequest
		method := padRight(strings.ToUpper(sr.Method), 6)
		methodStyle := lipgloss.NewStyle().Foreground(MethodColor(sr.Method)).Bold(true)
		name := truncate(sr.Name+"*", max(width-9, 4))
		nameStyle := lipgloss.NewStyle()
		if selected {
			nameStyle = nameStyle.Reverse(true).Foreground(lipgloss.Color("6"))
		}
		return "  " + methodStyle.Render(method) + " " + nameStyle.Render(name)
	}

	ep := item.Endpoint
	cursor := "  "
	if m.Store != nil && m.Store.GetEndpointOverride(string(ep.Method), ep.Path) != nil {
		cursor = "~ "
	}
	method := padRight(strings.ToUpper(string(ep.Method)), 6)
	methodStyle := lipgloss.NewStyle().Foreground(MethodColor(string(ep.Method))).Bold(true)
	path := truncate(ep.Path, max(width-len(cursor)-7, 4))
	pathStyle := lipgloss.NewStyle()
	if selected {
		pathStyle = pathStyle.Reverse(true).Foreground(lipgloss.Color("6"))
	}
	return cursor + methodStyle.Render(method) + " " + pathStyle.Render(path)
}
