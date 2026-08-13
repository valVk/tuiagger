package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain forces ANSI color output for the whole package's tests. go test
// runs with stdout not attached to a tty, so lipgloss would otherwise
// silently strip all styling and any test asserting on rendered color
// (e.g. distinguishing a hint's key from its label) would pass vacuously
// regardless of whether the styling code is actually correct.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.ANSI)
	os.Exit(m.Run())
}
