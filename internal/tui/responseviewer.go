package tui

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/valVK/tuiagger/internal/request"
)

// responseViewport matches ResponseViewer.tsx's RESPONSE_VIEWPORT: a fixed
// 15-line window that auto-scrolls to keep the cursor visible, independent
// of the outer panel's own scroll (distinct J/K vs j/k keys, so both
// coexist without a focus-mode flag).
const responseViewport = 15

type respTab int

const (
	tabResponseBody respTab = iota
	tabRequestBody
)

// responseViewer is the visual-select-and-yank response body viewer,
// promoted from the Phase 0 spike (internal/spike/viewer) and rebuilt in
// Phase 3's UX-parity pass to match ResponseViewer.tsx: a request/response
// tab toggle, its own scroll-follow viewport, and a transient yank message.
type responseViewer struct {
	Lines     []string
	Cursor    int
	Selecting bool
	SelStart  int
	Offset    int
	Tab       respTab
	Yanked    bool
}

func newResponseViewer(body string) responseViewer {
	return responseViewer{Lines: strings.Split(body, "\n")}
}

// yankExpiredMsg clears the transient "[yanked]" indicator, matching TS's
// setTimeout(..., 1500). Like the TS version, a second yank within the
// window doesn't reset the timer — the first timer to fire always clears
// it, even if a newer yank message should still be showing.
type yankExpiredMsg struct{}

func clearYankAfter() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return yankExpiredMsg{} })
}

// handleKey processes '\' (tab toggle, always active) plus J/K/g/G/v/y/Esc
// (response-tab only — matches ResponseViewer.tsx's `if (responseTab !==
// 'response') return`, so navigating/yanking is a no-op while viewing the
// request tab).
func (v responseViewer) handleKey(keyStr string) (responseViewer, tea.Cmd) {
	if keyStr == `\` {
		if v.Tab == tabResponseBody {
			v.Tab = tabRequestBody
		} else {
			v.Tab = tabResponseBody
		}
		return v, nil
	}
	if v.Tab != tabResponseBody {
		return v, nil
	}

	switch keyStr {
	case "esc":
		v.Selecting = false
	case "v":
		if v.Selecting {
			v.Selecting = false
		} else {
			v.Selecting = true
			v.SelStart = v.Cursor
		}
	case "J", "down":
		v.moveCursor(v.Cursor + 1)
	case "K", "up":
		v.moveCursor(v.Cursor - 1)
	case "G":
		v.moveCursor(len(v.Lines) - 1)
	case "g":
		v.moveCursor(0)
	case "y":
		text := v.selectedText()
		v.Selecting = false
		v.Yanked = true
		return v, tea.Batch(yankCmd(text), clearYankAfter())
	}
	return v, nil
}

func (v *responseViewer) moveCursor(next int) {
	v.Cursor = max(0, min(next, len(v.Lines)-1))
	v.Offset = min(v.Offset, v.Cursor)
	if v.Cursor >= v.Offset+responseViewport {
		v.Offset = v.Cursor - responseViewport + 1
	}
}

func (v responseViewer) selectedText() string {
	if !v.Selecting {
		return strings.Join(v.Lines, "\n")
	}
	lo, hi := v.SelStart, v.Cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	return strings.Join(v.Lines[lo:hi+1], "\n")
}

func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.WriteAll(text)
		return nil
	}
}

// statusColor matches ResponseViewer.tsx's own inline 3-way color — a
// distinct scale from colors.ts's 4-way getStatusColor used for the
// RESPONSES section's status tabs. TS applies this literally to whatever
// status is set, including 0 (network failure), which is < 300 and so
// renders green — an existing TS quirk, replicated here rather than fixed.
func responseStatusColor(status int) lipgloss.Color {
	switch {
	case status < 300:
		return color2xx
	case status < 400:
		return color4xx
	default:
		return color5xx
	}
}

// render matches ResponseViewer.tsx line-for-line: a status/tab header row,
// then either the REQUEST tab (method+url, headers, body) or the RESPONSE
// tab (headers, body with cursor/selection highlighting, contextual hint
// text, curl). active gates the interactive hints (position indicator,
// tab-toggle hint, selection hint) exactly like the TS `isActive` prop.
func (v responseViewer) render(resp *request.Response, curl string, active bool, width int) []string {
	var lines []string

	tabLabel := "RESPONSE "
	if v.Tab == tabRequestBody {
		tabLabel = "REQUEST  "
	}
	statusText := boldStyle.Render(tabLabel) +
		lipgloss.NewStyle().Foreground(responseStatusColor(resp.Status)).Bold(true).Render(strconv.Itoa(resp.Status)+" "+resp.StatusText) +
		dimStyle.Render(" "+strconv.FormatInt(resp.TimeMs, 10)+"ms")
	if v.Yanked {
		statusText += lipgloss.NewStyle().Foreground(color2xx).Bold(true).Render("  [yanked]")
	}

	right := ""
	if active && v.Tab == tabResponseBody {
		total := len(v.Lines)
		if total > responseViewport {
			right += dimStyle.Render("[" + strconv.Itoa(v.Cursor+1) + "/" + strconv.Itoa(total) + "]  ")
		}
	}
	requestTabStyle, responseTabStyle := dimStyle, dimStyle
	if v.Tab == tabRequestBody {
		requestTabStyle = cyanStyle
	} else {
		responseTabStyle = cyanStyle
	}
	right += requestTabStyle.Render("[ Request  ]") + responseTabStyle.Render("[ Response ]")
	if active {
		right += dimStyle.Render(` \:toggle`)
	}

	gap := max(width-lipgloss.Width(statusText)-lipgloss.Width(right), 1)
	lines = append(lines, statusText+strings.Repeat(" ", gap)+right)
	lines = append(lines, strings.Repeat("─", max(width, 0)))

	if v.Tab == tabRequestBody {
		lines = append(lines, dimStyle.Bold(true).Render(resp.RequestMethod+" "+resp.RequestURL))
		for _, k := range sortedKeys(resp.RequestHeaders) {
			lines = append(lines, dimStyle.Render(k+": ")+resp.RequestHeaders[k])
		}
		if resp.RequestBody != "" {
			lines = append(lines, "", dimStyle.Bold(true).Render("BODY"))
			for l := range strings.SplitSeq(resp.RequestBody, "\n") {
				lines = append(lines, l)
			}
		}
		return lines
	}

	if len(resp.Headers) > 0 {
		for _, k := range sortedKeys(resp.Headers) {
			lines = append(lines, dimStyle.Render(k+": ")+resp.Headers[k])
		}
		lines = append(lines, strings.Repeat("─", max(width, 0)))
	}

	lo, hi := -1, -1
	if v.Selecting {
		lo, hi = v.SelStart, v.Cursor
		if lo > hi {
			lo, hi = hi, lo
		}
	}
	end := min(v.Offset+responseViewport, len(v.Lines))
	for i := v.Offset; i < end; i++ {
		text := v.Lines[i]
		if text == "" {
			text = " "
		}
		line := truncate(text, width)
		switch {
		case v.Selecting && i >= lo && i <= hi:
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		case i == v.Cursor:
			line = cyanStyle.Render(line)
		}
		lines = append(lines, line)
	}

	if active {
		switch {
		case v.Selecting:
			lines = append(lines, cyanStyle.Render("-- VISUAL --  y: yank selection  Esc: cancel"))
		case len(v.Lines) > responseViewport:
			lines = append(lines, dimStyle.Render("J/K: move  g/G: top/bottom  v: visual  y: yank all"))
		default:
			lines = append(lines, dimStyle.Render("v: visual  y: yank"))
		}
	}

	if curl != "" {
		lines = append(lines, dimStyle.Render("curl: "+curl))
	}

	return lines
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
