package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct{ keys, desc string }

type helpSection struct {
	title   string
	entries []helpEntry
}

// helpSections is a verbatim port of HelpPopup.tsx's SECTIONS — the
// authoritative shortcut inventory for the whole app. Kept in the same
// order/grouping/wording as the TS source, including entries for keys not
// yet wired in this rewrite (manual request builder, environments) — the
// cheatsheet documents the target UX, not just what's implemented today.
var helpSections = []helpSection{
	{"GLOBAL", []helpEntry{
		{"q", "Quit"},
		{"Ctrl+R", "Reload spec"},
		{"i", "Info panel (servers / auth / envs)"},
		{"[", "Toggle left panel width"},
		{"?", "Toggle this help"},
	}},
	{"NAVIGATION", []helpEntry{
		{"h / ←", "Focus left panel"},
		{"l / →", "Focus right panel"},
	}},
	{"LEFT PANEL", []helpEntry{
		{"j / k", "Move up / down"},
		{"Enter", "Expand / collapse tag"},
		{"g / G", "First / last item"},
		{"c / x", "Collapse / expand all tags"},
	}},
	{"RIGHT PANEL (browse)", []helpEntry{
		{"j / k", "Scroll up / down"},
		{"g", "Scroll to top"},
		{"t", "Enter try-it-out mode"},
		{"e", "Quick execute (uses saved params)"},
		{"m", "New manual request"},
		{"E", "Edit saved request"},
		{"D", "Delete saved request"},
		{"\\", "Toggle request / response tab"},
		{"/", "Cycle response status tabs"},
	}},
	{"TRY IT OUT / MANUAL", []helpEntry{
		{"e", "Execute request"},
		{"p", "Edit path"},
		{"m", "Cycle HTTP method"},
		{"s", "Save request (manual mode)"},
		{"d", "Delete request (manual edit)"},
		{"r", "Reset overrides (tryit only)"},
		{"Esc", "Exit (NORMAL mode only)"},
	}},
	{"PARAMETERS / HEADERS", []helpEntry{
		{"j / k", "Navigate rows"},
		{"i", "Edit value"},
		{"← / →", "Cycle enum values"},
		{"d", "Toggle enable / disable"},
		{"x", "Delete custom param / header"},
		{"c", "Cycle param type (query / path)"},
		{"Tab", "Move to next section"},
	}},
	{"BODY", []helpEntry{
		{"j", "Focus body"},
		{"i", "Edit body (scaffolds JSON if empty)"},
		{"Esc", "Finish editing / unfocus"},
	}},
	{"RESPONSE BODY", []helpEntry{
		{"J / K", "Scroll down / up"},
		{"g / G", "Jump to top / bottom"},
		{"v", "Toggle visual selection"},
		{"y", "Yank selection (or full body)"},
		{"Esc", "Cancel visual mode"},
	}},
	{"INFO PANEL  (i)", []helpEntry{
		{"Tab", "Switch section (Servers / Auth / Envs)"},
		{"j / k", "Navigate items"},
		{"Enter", "Select server / activate env"},
		{"Esc", "Close panel"},
	}},
	{"ENVIRONMENTS  (in Info Panel)", []helpEntry{
		{"n", "New environment"},
		{"e", "Edit variables"},
		{"x", "Delete environment"},
		{"i", "Add / edit variable"},
		{"Esc", "Back to env list"},
	}},
	{"MANUAL REQUEST  (m)", []helpEntry{
		{"Tab", "Next field"},
		{"a", "Add query / header row"},
		{"d", "Delete selected row"},
		{"e", "Execute request"},
		{"s", "Save request"},
		{"Esc", "Close"},
	}},
}

type helpLine struct {
	isHeader bool
	isSpacer bool
	keys     string
	desc     string
	title    string
}

func flattenHelpSections(sections []helpSection) []helpLine {
	var lines []helpLine
	for _, s := range sections {
		lines = append(lines, helpLine{isHeader: true, title: s.title})
		for _, e := range s.entries {
			lines = append(lines, helpLine{keys: e.keys, desc: e.desc})
		}
		lines = append(lines, helpLine{isSpacer: true})
	}
	return lines
}

var (
	helpLeftLines, helpRightLines []helpLine
	helpContentHeight             int
)

func init() {
	mid := (len(helpSections) + 1) / 2
	helpLeftLines = flattenHelpSections(helpSections[:mid])
	helpRightLines = flattenHelpSections(helpSections[mid:])
	helpContentHeight = max(len(helpLeftLines), len(helpRightLines))
}

func (m Model) handleHelpKey(key string) (tea.Model, tea.Cmd) {
	viewHeight := max(m.Height-6-3, 1) // matches renderHelpPopup's viewHeight below
	maxScroll := max(helpContentHeight-viewHeight, 0)

	switch key {
	case "?", "esc":
		m.ShowHelp = false
	case "j", "down":
		m.HelpScroll = min(m.HelpScroll+1, maxScroll)
	case "k", "up":
		m.HelpScroll = max(m.HelpScroll-1, 0)
	case "G":
		m.HelpScroll = maxScroll
	case "g":
		m.HelpScroll = 0
	}
	return m, nil
}

const (
	helpKeyWidth  = 12
	helpDescWidth = 40
	helpColGap    = 4
)

func renderHelpLine(l helpLine) string {
	if l.isSpacer {
		return ""
	}
	if l.isHeader {
		return cyanStyle.Bold(true).Render(truncate(l.title, helpKeyWidth+helpDescWidth))
	}
	return padRight(yellowStyle.Render(truncate(l.keys, helpKeyWidth)), helpKeyWidth) +
		dimStyle.Render(truncate(l.desc, helpDescWidth))
}

// renderHelpPopup matches HelpPopup.tsx: a double-bordered cyan box
// replacing the two-panel body (Header/StatusBar stay visible around it),
// two scrollable columns of flattened sections.
func (m Model) renderHelpPopup(height, width int) string {
	viewHeight := max(height-3, 1) // border top+bottom (2) + title row (1)
	maxScroll := max(helpContentHeight-viewHeight, 0)
	scroll := min(m.HelpScroll, maxScroll)

	leftSlice := sliceHelpLines(helpLeftLines, scroll, viewHeight)
	rightSlice := sliceHelpLines(helpRightLines, scroll, viewHeight)

	var leftCol, rightCol []string
	for _, l := range leftSlice {
		leftCol = append(leftCol, renderHelpLine(l))
	}
	for _, l := range rightSlice {
		rightCol = append(rightCol, renderHelpLine(l))
	}

	colWidth := helpKeyWidth + helpDescWidth
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(colWidth).Render(strings.Join(leftCol, "\n")),
		strings.Repeat(" ", helpColGap),
		lipgloss.NewStyle().Width(colWidth).Render(strings.Join(rightCol, "\n")),
	)

	titleRight := "?/Esc: close"
	if maxScroll > 0 {
		titleRight = "j/k: scroll  g/G: top/bottom  " + titleRight
	}
	titleGap := max(width-4-lipgloss.Width("KEYBOARD SHORTCUTS")-lipgloss.Width(titleRight), 1)
	title := boldStyle.Render("KEYBOARD SHORTCUTS") + strings.Repeat(" ", titleGap) + dimStyle.Render(titleRight)

	content := title + "\n" + body

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(0, 1).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(activeBorderColor).
		Render(content)
}

func sliceHelpLines(lines []helpLine, start, count int) []helpLine {
	end := min(start+count, len(lines))
	if start >= end {
		return nil
	}
	return lines[start:end]
}
