package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/openapi"
)

var (
	activeBorderColor   = lipgloss.Color("6") // cyan
	inactiveBorderColor = lipgloss.Color("8") // gray
	dimStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle           = lipgloss.NewStyle().Bold(true)
	cyanStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	yellowStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	if m.Width == 0 {
		return "loading..."
	}

	contentHeight := max(m.Height-6, 10)
	leftWidth := max(m.Width*30/100, 20)
	rightWidth := max(m.Width-leftWidth-2, 20)

	left := m.renderLeftPanel(contentHeight, leftWidth)
	right := m.renderRightPanel(contentHeight, rightWidth)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return lipgloss.JoinVertical(lipgloss.Left, m.renderHeader(), body, m.renderStatusBar())
}

func (m Model) renderHeader() string {
	title := "tuiagger"
	version := ""
	if m.Spec != nil {
		title = m.Spec.Spec.Info.Title
		version = m.Spec.Spec.Info.Version
	}
	left := boldStyle.Render(title)
	if version != "" {
		left += dimStyle.Render(" v" + version)
	}
	if m.CollectionName != "" {
		left = yellowStyle.Render("["+m.CollectionName+"] ") + left
	}

	right := ""
	if m.Spec != nil && len(m.Spec.Spec.Servers) > m.SelectedServer && m.SelectedServer >= 0 {
		right = dimStyle.Render("server: ") + cyanStyle.Render(m.Spec.Spec.Servers[m.SelectedServer].URL)
	}

	style := lipgloss.NewStyle().
		Width(m.Width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(inactiveBorderColor)

	gap := max(m.Width-2-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	return style.Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderStatusBar() string {
	staticHints := "q:quit  i:info  ?:help  Ctrl+r:reload"
	dynamicHints := "h/l:panels  j/k:navigate  Enter:expand  c/x:collapse/expand  t:try it"
	switch {
	case m.Mode == ModeTryIt:
		dynamicHints = "j/k:params  i:edit  d:disable  m:method  p:path  e:execute  r:reset  Esc:cancel"
	case m.ActivePanel == PanelRight:
		dynamicHints = "h/l:panels  j/k:scroll  g:top  /:next response  t:try it  e:quick execute"
	}
	position := fmt.Sprintf("%d/%d", m.safeLeftIndex()+1, len(m.FlatList))

	style := lipgloss.NewStyle().
		Width(m.Width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderForeground(inactiveBorderColor)

	left := cyanStyle.Render(staticHints)
	right := dimStyle.Render(dynamicHints + "  " + position)
	gap := max(m.Width-2-lipgloss.Width(left)-lipgloss.Width(right)-2, 1)
	return style.Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderLeftPanel(height, width int) string {
	active := m.ActivePanel == PanelLeft
	borderColor := inactiveBorderColor
	if active {
		borderColor = activeBorderColor
	}

	title := "ENDPOINTS"
	if m.Spec != nil {
		title = m.Spec.Spec.Info.Title
	}

	visibleHeight := max(height-4, 1)
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
		lines = append(lines, renderListRow(m.FlatList[i], i == selected, m.ExpandedTags, width-2))
	}
	for len(lines) < visibleHeight {
		lines = append(lines, "")
	}

	footer := ""
	if len(m.FlatList) > visibleHeight {
		footer = dimStyle.Render(fmt.Sprintf("%d/%d", selected+1, len(m.FlatList)))
	}

	content := boldStyle.Render(truncate(title, width-2)) + "\n" + strings.Join(lines, "\n") + "\n" + footer

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(content)
}

func renderListRow(item FlatListItem, selected bool, expanded map[string]bool, width int) string {
	rowStyle := lipgloss.NewStyle()
	if selected {
		rowStyle = rowStyle.Reverse(true)
	}

	if item.Type == ItemTag {
		arrow := "▶"
		if expanded[item.TagName] {
			arrow = "▼"
		}
		return rowStyle.Bold(true).Render(truncate(arrow+" "+item.TagName, width))
	}

	ep := item.Endpoint
	method := padRight(strings.ToUpper(string(ep.Method)), 6)
	methodStyle := lipgloss.NewStyle().Foreground(MethodColor(string(ep.Method))).Bold(true)
	path := truncate(ep.Path, max(width-8, 4))
	return methodStyle.Render(method) + " " + rowStyle.Render(path)
}

func (m Model) renderRightPanel(height, width int) string {
	active := m.ActivePanel == PanelRight
	borderColor := inactiveBorderColor
	if active {
		borderColor = activeBorderColor
	}

	var lines []string
	item := m.selectedItem()
	switch {
	case m.Loading:
		lines = []string{cyanStyle.Render("Executing request...")}
	case item == nil:
		lines = []string{dimStyle.Render("Select an endpoint from the left panel (j/k to navigate)")}
	case item.Type == ItemTag:
		lines = m.renderTagLines(item.TagName, width-2)
	case item.Type == ItemEndpoint && m.Mode == ModeTryIt:
		lines = m.renderTryItLines(item.Endpoint, width-2)
	case item.Type == ItemEndpoint:
		lines = m.renderEndpointLines(item.Endpoint, width-2)
	}

	visibleHeight := max(height-2, 1)
	start := min(m.RightScroll, max(len(lines)-1, 0))
	end := min(start+visibleHeight, len(lines))
	visible := lines[start:end]
	for len(visible) < visibleHeight {
		visible = append(visible, "")
	}

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(borderColor).
		Render(strings.Join(visible, "\n"))
}

func (m Model) renderTagLines(tagName string, width int) []string {
	lines := []string{boldStyle.Render(tagName), dimStyle.Render("Enter: expand/collapse"), ""}
	for _, ep := range m.EndpointsByTag[tagName] {
		methodStyle := lipgloss.NewStyle().Foreground(MethodColor(string(ep.Method))).Bold(true)
		lines = append(lines, methodStyle.Render(padRight(strings.ToUpper(string(ep.Method)), 7))+truncate(ep.Path, max(width-8, 4)))
	}
	return lines
}

func (m Model) renderEndpointLines(ep *openapi.ParsedEndpoint, width int) []string {
	op := ep.Operation
	var lines []string

	lines = append(lines, MethodBadge(string(ep.Method))+" "+boldStyle.Render(ep.Path))
	lines = append(lines, "")

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
		lines = append(lines, dimStyle.Render(padRight("NAME", 25)+padRight("VALUE", 20)+padRight("TYPE", 10)+"DESCRIPTION"))
		for _, p := range sortedParameters(op.Parameters) {
			lines = append(lines, renderParamRow(p, "", false, false, false, "", width))
		}
	}

	if op.RequestBody != nil {
		lines = append(lines, "", boldStyle.Render("BODY")+" "+dimStyle.Render(contentTypesOf(op.RequestBody.Content)))
		schema := firstSchema(op.RequestBody.Content)
		if schema != nil {
			for l := range strings.SplitSeq(openapi.FormatSchema(schema, 0), "\n") {
				lines = append(lines, dimStyle.Render(l))
			}
		}
	}

	lines = append(lines, "", boldStyle.Render("RESPONSES"))
	lines = append(lines, renderResponseTabs(op, m.ResponseTab)...)
	lines = append(lines, m.renderResponseBlock(width)...)

	return lines
}

// renderParamRow renders one PARAMETERS row for both browse (read-only,
// value always "") and try-it-out (editable) modes — the two modes share
// this one row renderer rather than each maintaining their own copy.
func renderParamRow(p openapi.Parameter, value string, selected, editing, disabled bool, editingView string, width int) string {
	name := p.Name
	if p.Required {
		name += " *"
	}
	typeStr := ""
	if p.Schema != nil && len(p.Schema.Type) > 0 {
		typeStr = p.Schema.Type[0]
	}
	desc := truncate(p.Description, max(width-59, 4))

	valueCell := truncate(value, 19)
	if editing {
		valueCell = truncate(editingView, 19)
	}
	if disabled {
		valueCell = "(disabled)"
	}

	row := padRight(truncate(name, 24), 25) + padRight(valueCell, 20) + padRight(truncate(typeStr, 9), 10) + desc
	if disabled {
		row = dimStyle.Render(row)
	}
	if selected {
		row = lipgloss.NewStyle().Reverse(true).Render(row)
	}
	return row
}

// renderResponseBlock renders the curl command and response body (via the
// visual-select viewer) shared between browse and try-it-out modes.
func (m Model) renderResponseBlock(width int) []string {
	if m.Response == nil {
		return nil
	}
	var lines []string
	lines = append(lines, "", boldStyle.Render("RESPONSE"))
	if m.Response.Error != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(color5xx).Render("Error: "+m.Response.Error))
	} else {
		statusLine := lipgloss.NewStyle().Foreground(StatusColor(m.Response.Status)).Bold(true).
			Render(fmt.Sprintf("%d %s", m.Response.Status, m.Response.StatusText))
		lines = append(lines, statusLine+dimStyle.Render(fmt.Sprintf("  %dms", m.Response.TimeMs)))
	}
	if m.Curl != "" {
		lines = append(lines, "", dimStyle.Render("curl:"))
		for l := range strings.SplitSeq(m.Curl, "\n") {
			lines = append(lines, dimStyle.Render(l))
		}
	}
	if m.Response.Body != "" {
		lines = append(lines, "", boldStyle.Render("BODY")+dimStyle.Render("  J/K:move v:select y:yank"))
		lines = append(lines, m.Viewer.render(width)...)
	}
	return lines
}

func renderResponseTabs(op *openapi.Operation, activeTab int) []string {
	codes := make([]string, len(op.Responses))
	byCode := make(map[string]openapi.Response, len(op.Responses))
	for i, r := range op.Responses {
		codes[i] = r.Status
		byCode[r.Status] = r.Response
	}
	sort.Strings(codes)
	if len(codes) == 0 {
		return nil
	}
	safeTab := activeTab % len(codes)

	var tabLine strings.Builder
	for i, code := range codes {
		style := lipgloss.NewStyle()
		if statusNum, ok := parseStatus(code); ok {
			style = style.Foreground(StatusColor(statusNum))
		}
		if i == safeTab {
			style = style.Reverse(true).Bold(true)
		}
		tabLine.WriteString(style.Render(" " + code + " "))
		tabLine.WriteString(" ")
	}

	resp := byCode[codes[safeTab]]
	lines := []string{tabLine.String(), dimStyle.Render(resp.Description)}
	schema := firstSchema(resp.Content)
	if schema != nil {
		for l := range strings.SplitSeq(openapi.FormatSchema(schema, 0), "\n") {
			lines = append(lines, dimStyle.Render(l))
		}
	}
	return lines
}

func sortedParameters(params []openapi.Parameter) []openapi.Parameter {
	out := make([]openapi.Parameter, 0, len(params))
	for _, p := range params {
		if p.Required {
			out = append(out, p)
		}
	}
	for _, p := range params {
		if !p.Required {
			out = append(out, p)
		}
	}
	return out
}

func firstSchema(content map[string]openapi.MediaType) *openapi.Schema {
	for _, mt := range content {
		if mt.Schema != nil {
			return mt.Schema
		}
	}
	return nil
}

func contentTypesOf(content map[string]openapi.MediaType) string {
	types := make([]string, 0, len(content))
	for k := range content {
		types = append(types, k)
	}
	sort.Strings(types)
	return strings.Join(types, ", ")
}

func parseStatus(code string) (int, bool) {
	n := 0
	for _, c := range code {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

func truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 2 {
		return s[:width]
	}
	return s[:width-2] + ".."
}

func wrapLines(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	for raw := range strings.SplitSeq(s, "\n") {
		for len(raw) > width {
			lines = append(lines, dimStyle.Render(raw[:width]))
			raw = raw[width:]
		}
		lines = append(lines, dimStyle.Render(raw))
	}
	return lines
}
