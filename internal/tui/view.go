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
	greenBoldStyle      = lipgloss.NewStyle().Foreground(color2xx).Bold(true)
)

// hint is one "key:label" pair rendered with a cyan key and a dim label,
// matching StatusBar.tsx's ShortcutList (a flat dim string loses the
// key/label distinction that makes the hint scannable).
type hint struct{ key, label string }

func renderHints(hints []hint) string {
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = cyanStyle.Render(h.key) + dimStyle.Render(":"+h.label)
	}
	return strings.Join(parts, dimStyle.Render("  "))
}

// button is one bracketed action hint, e.g. "[ Execute (e) ]", matching the
// TS RightPanel's inline "[ Try it out (t) ][ Execute (e) ]" banner.
type button struct {
	text  string
	style lipgloss.Style
}

func renderButtons(buttons []button) string {
	parts := make([]string, len(buttons))
	for i, b := range buttons {
		parts[i] = dimStyle.Render("[ ") + b.style.Render(b.text) + dimStyle.Render(" ]")
	}
	return strings.Join(parts, " ")
}

// panelBorderStyle matches every focusable Box in the TS app
// (LeftPanel/RightPanel/ManualRequestPanel): `borderStyle={isActive ? 'bold'
// : 'single'} borderColor={isActive ? 'cyan' : 'gray'}` — the active state
// changes the border's *weight* (thick line chars), not just its color.
func panelBorderStyle(active bool) (lipgloss.Border, lipgloss.Color) {
	if active {
		return lipgloss.ThickBorder(), activeBorderColor
	}
	return lipgloss.NormalBorder(), inactiveBorderColor
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	if m.Width == 0 {
		return "loading..."
	}

	// Matches App.tsx: loading/error replace the entire UI (no
	// Header/panels/StatusBar), not just the content area.
	if m.SpecLoading {
		return lipgloss.NewStyle().Padding(2, 2).Render(cyanStyle.Render("Loading OpenAPI specification..."))
	}
	if m.SpecError != "" {
		errStyle := lipgloss.NewStyle().Foreground(color5xx)
		return lipgloss.NewStyle().Padding(2, 2).Render(strings.Join([]string{
			errStyle.Bold(true).Render("Error loading OpenAPI specification"),
			errStyle.Render(m.SpecError),
			"",
			dimStyle.Render("Source: " + m.Source),
			dimStyle.Render("Press Ctrl+R to retry or q to quit"),
		}, "\n"))
	}

	// Header and footer are each full 3-row boxes (border top + content +
	// border bottom, matching Header.tsx/StatusBar.tsx's default 4-sided
	// Ink Box), plus the panel row's own top+bottom border — 3+3+2 = 8.
	contentHeight := max(m.Height-8, 10)
	leftWidthPct := 30
	if m.LeftExpanded {
		leftWidthPct = 50
	}
	leftWidth := max(m.Width*leftWidthPct/100, 20)
	rightWidth := max(m.Width-leftWidth-2, 20)

	var body string
	switch {
	case m.ShowHelp:
		body = m.renderHelpPopup(contentHeight, m.Width)
	case m.ShowInfo:
		body = m.renderInfoPopup(contentHeight, m.Width)
	case m.Mode == ModeManual && m.Manual.ShowSaveDialog:
		body = m.renderSaveDialogOverlay(contentHeight, m.Width)
	case m.Mode == ModeRenameTag:
		body = m.renderRenameTagOverlay(contentHeight, m.Width)
	default:
		left := m.renderLeftPanel(contentHeight, leftWidth)
		var right string
		if m.Mode == ModeManual {
			right = m.renderManualPanel(contentHeight, rightWidth)
		} else {
			right = m.renderRightPanel(contentHeight, rightWidth)
		}
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

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
	if name := m.activeEnvName(); name != "" {
		right += yellowStyle.Render("env: " + name + "  ")
	}
	if m.Spec != nil && len(m.Spec.Spec.Servers) > m.SelectedServer && m.SelectedServer >= 0 {
		right += dimStyle.Render("server: ") + cyanStyle.Render(m.Spec.Spec.Servers[m.SelectedServer].URL)
	}

	// Header.tsx's Box has a default (4-sided) border, not just a bottom
	// rule — matches every other focusable/framed Box in the TS app.
	style := lipgloss.NewStyle().
		Width(m.Width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(inactiveBorderColor)

	gap := max(m.Width-4-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return style.Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) renderStatusBar() string {
	staticHints := []hint{{"q", "quit"}, {"i", "info"}, {"?", "help"}, {"Ctrl+r", "reload"}}

	// Matches StatusBar.tsx's getDynamicShortcuts exactly — a curated
	// subset, not every bound key (e.g. no 'd'/'c'/'x'/'g' here; those
	// hints live next to the row they act on instead, per that file's own
	// "avoid duplicating a hint that's already visible" comment).
	var dynamicHints []hint
	switch {
	case m.Mode == ModeManual:
		// Matches StatusBar.tsx's manual-mode hints, minus its stale 'a:add
		// param' entry — no 'a' key exists in either app (help.go dropped
		// the same dead hint earlier this session; see HANDOFF.md).
		dynamicHints = []hint{
			{"p", "path"}, {"m", "method"}, {"e", "execute"},
			{"s", "save"}, {"d", "delete"}, {"Esc", "cancel"},
		}
	case m.Mode == ModeRenameTag:
		dynamicHints = []hint{{"Enter", "save"}, {"Esc", "cancel"}}
	case m.Mode == ModeTryIt:
		dynamicHints = []hint{
			{"j/k", "navigate"}, {"i", "edit"}, {"Esc", "done/cancel"},
			{"e", "execute"}, {"m", "method"}, {"p", "path"}, {"r", "reset"},
		}
	case m.ActivePanel == PanelRight:
		dynamicHints = []hint{
			{"h/l", "panels"}, {"j/k", "scroll"}, {"[", "wide"}, {"t", "try it"}, {"m", "manual"},
		}
	default:
		dynamicHints = []hint{
			{"h/l", "panels"}, {"j/k", "navigate"}, {"Enter", "expand tag"}, {"[", "wide"}, {"t", "try it"}, {"m", "manual"},
		}
	}

	position := fmt.Sprintf("%d/%d", m.safeLeftIndex()+1, len(m.FlatList))

	// StatusBar.tsx's Box also has a default (4-sided) border, and always
	// appends a "{cols}x{rows}" size indicator after the position.
	style := lipgloss.NewStyle().
		Width(m.Width-2).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(inactiveBorderColor)

	left := renderHints(staticHints)
	right := renderHints(dynamicHints) + dimStyle.Render(fmt.Sprintf("  %s  %dx%d", position, m.Width, m.Height))
	gap := max(m.Width-4-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return style.Render(left + strings.Repeat(" ", gap) + right)
}

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

func (m Model) renderRightPanel(height, width int) string {
	active := m.ActivePanel == PanelRight
	borderStyle, borderColor := panelBorderStyle(active)

	// Available columns inside the box: width minus the border (1 each
	// side) minus Padding(0,1) (1 each side) — Width() below already
	// counts padding internally (lipgloss), so only the border needs
	// subtracting from the outer width passed in.
	inner := max(width-4, 1)

	var lines []string
	// cursorLine, when set (try-it-out only), drives an auto-scroll-into-
	// view instead of the manual m.RightScroll offset — see
	// renderTryItLines's doc comment for why.
	cursorLine := -1
	item := m.selectedItem()
	switch {
	case m.Loading:
		lines = []string{cyanStyle.Render("Executing request...")}
	case item == nil:
		lines = []string{dimStyle.Render("Select an endpoint from the left panel (j/k to navigate)")}
	case item.Type == ItemTag:
		lines = m.renderTagLines(item.TagName, inner)
	case item.Type == ItemEndpoint && m.Mode == ModeTryIt:
		lines, cursorLine = m.renderTryItLines(item.Endpoint, inner)
	case item.Type == ItemEndpoint:
		lines = m.renderEndpointLines(item.Endpoint, active, inner)
	}

	visibleHeight := max(height-2, 1)
	start := min(m.RightScroll, max(len(lines)-1, 0))
	if cursorLine >= 0 {
		start = scrollToShow(cursorLine, start, visibleHeight, len(lines))
	}
	end := min(start+visibleHeight, len(lines))
	visible := lines[start:end]
	for len(visible) < visibleHeight {
		visible = append(visible, "")
	}

	return lipgloss.NewStyle().
		Width(width-2).
		Height(height).
		Padding(0, 1).
		BorderStyle(borderStyle).
		BorderForeground(borderColor).
		Render(strings.Join(visible, "\n"))
}

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

const (
	paramCursorWidth = 3
	paramNameWidth   = 25
	paramValueWidth  = 25
	// 16 rather than TS's 12: TS stacks "in" above "type" on two lines so
	// a 12-wide box never runs out of room; this port combines them on one
	// line ("in:type") to avoid doubling every row's height (see
	// renderParamRow's doc comment), so the column needs enough width for
	// the longest realistic combination ("header:integer", 14 chars) plus
	// a visible gap before DESCRIPTION.
	paramTypeWidth = 16
)

// paramTableHeader matches ParametersSection.tsx's column header row (a
// blank cursor column, then NAME/VALUE/TYPE/DESCRIPTION).
func paramTableHeader() string {
	return strings.Repeat(" ", paramCursorWidth) +
		dimStyle.Bold(true).Render(padRight("NAME", paramNameWidth)+padRight("VALUE", paramValueWidth)+padRight("TYPE", paramTypeWidth)+"DESCRIPTION")
}

// paramRowState is everything renderParamRow needs to reproduce
// SpecParamRow.tsx's row exactly — selected/editing/disabled state plus the
// live edit widget's rendered view (textinput.View(), when editing a
// non-enum value).
type paramRowState struct {
	param       openapi.Parameter
	value       string
	selected    bool
	editing     bool
	disabled    bool
	editingView string
}

// renderParamRow renders one PARAMETERS row for both browse (read-only,
// selected/editing always false) and try-it-out (editable) modes — the two
// share this one row renderer rather than each maintaining their own copy.
// Matches SpecParamRow.tsx column-for-column, with one deliberate
// adaptation: TS stacks the parameter's "in" (query/path/header) above its
// type in a two-line-tall flexbox cell; here, where each row is one line in
// a flat, line-sliced buffer, they're combined as "in:type" on one line
// instead of doubling every row's height.
func renderParamRow(s paramRowState, width int) []string {
	p := s.param

	// Cursor cell is padRight'd to paramCursorWidth (3), matching
	// SpecParamRow.tsx's `<Box width={3}>{'> '|'* '|'  '}</Box>` — the
	// literal 2-char marker plus Ink's own auto-pad to the box width.
	cursor := strings.Repeat(" ", paramCursorWidth)
	switch {
	case s.selected:
		cursor = cyanStyle.Render(padRight("> ", paramCursorWidth))
	case p.Required:
		cursor = lipgloss.NewStyle().Foreground(color5xx).Render(padRight("* ", paramCursorWidth))
	}

	// Every cell below pads its PLAIN text to the column width first, then
	// styles the padded result — not the other way around. Styling first
	// and padding second (the original approach) measures the ANSI-escaped
	// string's byte length, which is always >= the target width, so
	// padRight's len(s)>=width early-return fires and adds no real padding
	// at all — the same class of bug fixed once already in help.go's
	// renderHelpLine ("measure before you style, not after").
	name := p.Name
	nameStyle := lipgloss.NewStyle()
	switch {
	case s.selected:
		nameStyle = cyanStyle
	case s.disabled:
		nameStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
	}
	nameCell := nameStyle.Strikethrough(s.disabled).Render(padRight(truncateTS(name, paramNameWidth), paramNameWidth))

	enumOpts := enumValues(p)
	placeholder := paramPlaceholder(p)

	var valueCell string
	switch {
	case s.editing && !s.disabled && len(enumOpts) > 0:
		current := s.value
		if current == "" {
			current = enumOpts[0]
		}
		plain := "< " + current + " >"
		pad := strings.Repeat(" ", max(paramValueWidth-len(plain), 0))
		valueCell = dimStyle.Render("< ") + cyanStyle.Render(current) + dimStyle.Render(" >") + pad
	case s.editing && !s.disabled:
		valueCell = padRight(truncateTS(s.editingView, paramValueWidth), paramValueWidth)
	default:
		valueStyle := lipgloss.NewStyle().Foreground(color2xx) // green
		switch {
		case s.disabled:
			valueStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
		case s.selected:
			valueStyle = cyanStyle
		}
		display := s.value
		if display == "" {
			display = placeholder
		}
		if display == "" {
			display = "-"
		}
		valueCell = valueStyle.Render(padRight(truncateTS(display, paramValueWidth), paramValueWidth))
	}

	typeStr := "string"
	if p.Schema != nil && len(p.Schema.Type) > 0 {
		typeStr = p.Schema.Type[0]
	}
	typeStyle := yellowStyle
	if s.disabled {
		typeStyle = lipgloss.NewStyle().Foreground(inactiveBorderColor)
	}
	inPrefix := p.In + ":"
	typePlain := padRight(truncateTS(inPrefix+typeStr, paramTypeWidth), paramTypeWidth)
	prefixLen := min(len(inPrefix), len(typePlain))
	typeCell := dimStyle.Render(typePlain[:prefixLen]) + typeStyle.Render(typePlain[prefixLen:])

	descWidth := max(width-paramCursorWidth-paramNameWidth-paramValueWidth-paramTypeWidth, 4)
	desc := dimStyle.Render(truncate(p.Description, descWidth))

	lines := []string{cursor + nameCell + valueCell + typeCell + desc}

	if len(enumOpts) > 0 && !s.disabled {
		hint := lipgloss.NewStyle().Foreground(activeBorderColor).Faint(true).
			Render("Allowed: " + strings.Join(enumOpts, " | "))
		lines = append(lines, strings.Repeat(" ", paramCursorWidth+paramNameWidth)+hint)
	}

	return lines
}

// truncateTS matches SpecParamRow.tsx's local truncate: text shorter than
// maxLen-1 passes through untouched; otherwise it's cut to maxLen-2 chars
// plus a 3-dot ellipsis (TS parity — distinct from this file's other
// truncate, which uses a 2-dot ellipsis and a maxLen-0 threshold).
func truncateTS(text string, maxLen int) string {
	if len(text) <= maxLen-1 {
		return text
	}
	if maxLen <= 2 {
		return text[:max(maxLen, 0)]
	}
	return text[:maxLen-2] + "..."
}

func paramPlaceholder(p openapi.Parameter) string {
	if p.Schema != nil && p.Schema.Default != nil {
		return toStr(p.Schema.Default)
	}
	return toStr(p.Example)
}

// renderResponseBlock delegates to responseViewer.render, matching
// ResponseViewer.tsx exactly (status/tab header, request/response tab
// content, curl). "active" mirrors TS's `isActive={isActive && !isTryItMode}`
// — the viewer's interactive hints (position indicator, tab-toggle hint,
// selection hint) only show in browse mode with the right panel focused.
func (m Model) renderResponseBlock(width int) []string {
	if m.Viewer.Response == nil {
		return nil
	}
	// Matches the try-it-out response-viewer keys now being wired up in
	// tryit.go (see its doc comment): the interactive hints (position
	// indicator, v/y hint, \:toggle) show whenever those keys actually
	// work, not just in browse mode.
	active := m.Mode == ModeTryIt || (m.Mode == ModeBrowse && m.ActivePanel == PanelRight)
	return append([]string{""}, m.Viewer.View(active, width)...)
}

// renderResponseTabs matches ResponsesSection.tsx: "Responses" (not
// "RESPONSES" — TS itself is inconsistent about heading case across
// sections, replicated as-is), a "/:next" hint when active with more than
// one status code, and — matching the TS ternary exactly — only the
// *active* tab gets its status color; inactive tabs render in the default
// terminal foreground, not dimmed or status-colored.
func renderResponseTabs(op *openapi.Operation, activeTab int, active bool) []string {
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

	heading := boldStyle.Render("Responses")
	if active && len(codes) > 1 {
		heading += dimStyle.Render(" /:next")
	}

	var tabLine strings.Builder
	for i, code := range codes {
		style := lipgloss.NewStyle()
		if i == safeTab {
			if statusNum, ok := parseStatus(code); ok {
				style = style.Foreground(StatusColor(statusNum))
			}
			style = style.Reverse(true).Bold(true)
		}
		tabLine.WriteString(style.Render(" " + code + " "))
		tabLine.WriteString(" ")
	}

	resp := byCode[codes[safeTab]]
	desc := resp.Description
	if len(resp.Content) > 0 {
		desc += " (" + contentTypesOf(resp.Content) + ")"
	}
	lines := []string{"", heading, tabLine.String(), " " + dimStyle.Render(desc)}
	schema := firstSchema(resp.Content)
	if schema != nil {
		for l := range strings.SplitSeq(openapi.FormatSchema(schema, 0), "\n") {
			lines = append(lines, " "+dimStyle.Render(l))
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

// scrollToShow returns the minimal-motion scroll offset that brings line
// into the visible window [start, start+visibleHeight) — matches
// usePanelNavigation/useRightPanelKeyboard's scrollToParamRow intent
// (keep the focused row on screen without needlessly re-centering it).
func scrollToShow(line, start, visibleHeight, total int) int {
	switch {
	case line < start:
		start = line
	case line >= start+visibleHeight:
		start = line - visibleHeight + 1
	}
	return min(max(start, 0), max(total-visibleHeight, 0))
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
