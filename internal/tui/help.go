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
		{"R", "Rename custom tag"},
		{"D", "Delete custom tag (confirm if non-empty)"},
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
		{"j / k", "Also work as J/K while selecting"},
		{"g / G", "Jump to top / bottom"},
		{"v", "Toggle visual selection"},
		{"y", "Yank selection (or full body)"},
		{"C", "Yank the curl command"},
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
		{"Tab", "Next field (path / params / body)"},
		{"p", "Edit path"},
		{"m", "Cycle HTTP method"},
		{"e", "Execute request"},
		{"s", "Save request (name/tag dialog)"},
		{"d", "Delete (only while editing a saved request)"},
		{"Esc", "Close (discards unsaved draft)"},
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

// helpPopupState backs the '?' overlay. A nested widget the root Model
// dispatches into once ShowHelp is true (ShowHelp itself stays root-level,
// same as InfoPopup's ShowInfo) — see infopopup.go's doc comment on why
// "is this popup open at all" and "the popup's own state" are kept
// separate.
type helpPopupState struct {
	Scroll int
}

// Update moves the scroll position, or reports that '?'/Esc should close
// the popup — ShowHelp is root-owned, so this can't just clear it itself
// (same "report the change" shape as serversPanelState.Update).
func (h helpPopupState) Update(key string, viewHeight int) (next helpPopupState, closePopup bool) {
	maxScroll := max(helpContentHeight-viewHeight, 0)
	next = h
	switch key {
	case "?", "esc":
		return next, true
	case "j", "down":
		next.Scroll = min(next.Scroll+1, maxScroll)
	case "k", "up":
		next.Scroll = max(next.Scroll-1, 0)
	case "G":
		next.Scroll = maxScroll
	case "g":
		next.Scroll = 0
	}
	return next, false
}

func (m Model) handleHelpKey(key string) (tea.Model, tea.Cmd) {
	viewHeight := max(m.Height-6-3, 1) // matches View's viewHeight below
	var closePopup bool
	m.Help, closePopup = m.Help.Update(key, viewHeight)
	if closePopup {
		m.ShowHelp = false
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
	// Pad the plain key text to width *before* styling it, not after —
	// padRight measures len(), which counts ANSI escape bytes once a style
	// is applied. Styling first made any key of ~8+ visible chars (e.g.
	// "Ctrl+R") already exceed helpKeyWidth in raw byte length, so padRight
	// added zero padding and the description ran straight into the key
	// with no gap at all. Found while visually verifying this cheatsheet
	// through the pty harness — the same class of styling-vs-layout-math
	// bug as this session's other fixes, just manifesting as misalignment
	// instead of overflow.
	return yellowStyle.Render(padRight(truncate(l.keys, helpKeyWidth), helpKeyWidth)) +
		dimStyle.Render(truncate(l.desc, helpDescWidth))
}

// View matches HelpPopup.tsx: a double-bordered cyan box replacing the
// two-panel body (Header/StatusBar stay visible around it), two scrollable
// columns of flattened sections.
func (h helpPopupState) View(height, width int) string {
	viewHeight := max(height-3, 1) // border top+bottom (2) + title row (1)
	maxScroll := max(helpContentHeight-viewHeight, 0)
	scroll := min(h.Scroll, maxScroll)

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
		Width(width-2).
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
