package tui

import (
	"regexp"
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

// responseViewer is the visual-select-and-yank response body viewer: a
// request/response tab toggle, its own scroll-follow viewport, and a
// transient yank message.
// Curl is rendered as its own separate section below the body (see render)
// — not part of Lines/Cursor/the visual-selection scroll, since it's a
// fixed, always-fully-shown block, not something to navigate line by line.
type responseViewer struct {
	Lines      []string
	Cursor     int
	Selecting  bool
	SelStart   int
	Offset     int
	Tab        respTab
	Yanked     bool
	YankedCurl bool // distinct from Yanked so the "[...]" feedback names what was actually copied
}

func newResponseViewer(body string) responseViewer {
	return responseViewer{Lines: strings.Split(body, "\n")}
}

// yankExpiredMsg clears the transient "[yanked]"/"[curl yanked]" indicator,
// matching TS's setTimeout(..., 1500). Like the TS version, a second yank of
// the *same kind* within the window doesn't reset the timer — the first
// timer to fire always clears it, even if a newer yank message should still
// be showing. Curl (a Go-only addition) is deliberately kept a fully
// separate indicator/timer from the response-body yank rather than sharing
// one flag, since yanking the curl command has nothing to do with whatever
// visual selection or body content is currently showing.
type yankExpiredMsg struct{ curl bool }

func clearYankAfter() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return yankExpiredMsg{} })
}

func clearCurlYankAfter() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(time.Time) tea.Msg { return yankExpiredMsg{curl: true} })
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

// yankCurl copies the generated curl command to the clipboard — a Go-only
// addition, not a TS port: ResponseViewer.tsx renders the curl command as
// plain read-only text with no way to copy it except manual terminal
// selection. Works regardless of which tab (request/response) is active or
// whether a visual selection is in progress, since the curl command
// represents the whole request, not the body being viewed.
func (v responseViewer) yankCurl(curl string) (responseViewer, tea.Cmd) {
	v.YankedCurl = true
	return v, tea.Batch(yankCmd(curl), clearCurlYankAfter())
}

func yankCmd(text string) tea.Cmd {
	return func() tea.Msg {
		_ = clipboard.WriteAll(text)
		return nil
	}
}

// curlHeadingLine matches the RESPONSE/REQUEST status line's own pattern —
// a bold section label plus a right-side hint that becomes "[curl yanked]"
// right after 'C' is pressed, same as the main status line's "[yanked]".
func curlHeadingLine(active, yankedCurl bool) string {
	left := boldStyle.Render("CURL")
	switch {
	case yankedCurl:
		left += lipgloss.NewStyle().Foreground(color2xx).Bold(true).Render("  [curl yanked]")
	case active:
		left += dimStyle.Render("  C: yank curl")
	}
	return left
}

var (
	curlFlagRe    = regexp.MustCompile(`^(\s*)(curl|--?[A-Za-z][A-Za-z-]*)`)
	curlStringRe  = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	curlFlagStyle = cyanStyle
	// yellowStyle reused for quoted strings — the app's existing ANSI-16
	// palette (see colors.go) doesn't have a dedicated "string" color, and
	// yellow renders close enough to the amber/orange typical editor
	// syntax themes use for strings.
	curlStringStyle = yellowStyle
)

// colorizeCurlLine gives the generated curl command basic syntax
// highlighting instead of rendering it all dim/gray: quoted string
// arguments (URLs, header values, JSON bodies) in one color, the leading
// "curl"/flag token (curl, -X, -H, -d, --data, ...) in another. A Go-only
// addition, not a TS port — TS renders curl as a single plain dim line.
func colorizeCurlLine(line string) string {
	var b strings.Builder
	last := 0
	for _, m := range curlStringRe.FindAllStringIndex(line, -1) {
		b.WriteString(colorizeCurlFlag(line[last:m[0]]))
		b.WriteString(curlStringStyle.Render(line[m[0]:m[1]]))
		last = m[1]
	}
	b.WriteString(colorizeCurlFlag(line[last:]))
	return b.String()
}

func colorizeCurlFlag(s string) string {
	loc := curlFlagRe.FindStringSubmatchIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[4]] + curlFlagStyle.Render(s[loc[4]:loc[5]]) + s[loc[5]:]
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
// text, then a separate CURL section with its own heading/rule, syntax-
// colored). active gates the interactive hints (position indicator,
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
	if v.YankedCurl {
		statusText += lipgloss.NewStyle().Foreground(color2xx).Bold(true).Render("  [curl yanked]")
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
		default:
			// Always documents J/K/g/G here, not just when the body is
			// longer than one viewport — found via a user report that they
			// "lost" the J/K hint (it used to only show once content
			// overflowed, easy to miss that it's there at all otherwise).
			lines = append(lines, dimStyle.Render("J/K: move  g/G: top/bottom  v: visual  y: yank"))
		}
	}

	if curl != "" {
		lines = append(lines, "", curlHeadingLine(active, v.YankedCurl), strings.Repeat("─", max(width, 0)))
		for l := range strings.SplitSeq(curl, "\n") {
			lines = append(lines, colorizeCurlLine(truncate(l, width)))
		}
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
