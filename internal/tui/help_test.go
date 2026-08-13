package tui

import (
	"strings"
	"testing"
)

// TestRenderHelpLinePadsBeforeStyling is a regression test: padRight was
// being applied to the already-styled (ANSI-wrapped) key text, so padRight's
// len() check counted escape-code bytes and any key of ~8+ visible
// characters (e.g. "Ctrl+R") already exceeded helpKeyWidth in raw byte
// length — padRight added zero padding and the description ran straight
// into the key with no gap. Found via visual verification, not a unit test
// failure, which is exactly why this test exists now.
func TestRenderHelpLinePadsBeforeStyling(t *testing.T) {
	line := renderHelpLine(helpLine{keys: "Ctrl+R", desc: "Reload spec"})
	plain := stripANSI(line)
	if !strings.Contains(plain, "Ctrl+R      Reload spec") {
		t.Errorf("expected the key column padded to helpKeyWidth before the description, got %q", plain)
	}
}

func TestRenderHelpLineShortKeyStillPadded(t *testing.T) {
	line := renderHelpLine(helpLine{keys: "q", desc: "Quit"})
	plain := stripANSI(line)
	if !strings.HasPrefix(plain, "q") || !strings.Contains(plain, "Quit") {
		t.Errorf("got %q", plain)
	}
	// The description must start exactly helpKeyWidth characters in.
	if len(plain) < helpKeyWidth || plain[helpKeyWidth:helpKeyWidth+4] != "Quit" {
		t.Errorf("expected description to start at column %d, got %q", helpKeyWidth, plain)
	}
}

// TestHelpSectionsMatchClaudeMdForManualRequest is a regression test for a
// real drift caught during a Phase 7 cross-check: help.go's "MANUAL
// REQUEST (m)" section still had the pre-Phase-4 keybindings (an 'a' key
// that was never implemented, no 'p'/'m') long after CLAUDE.md's own
// "Manual Request (m):" table was updated to match what actually shipped.
func TestHelpSectionsMatchClaudeMdForManualRequest(t *testing.T) {
	var section *helpSection
	for i := range helpSections {
		if helpSections[i].title == "MANUAL REQUEST  (m)" {
			section = &helpSections[i]
		}
	}
	if section == nil {
		t.Fatalf("expected a MANUAL REQUEST section")
	}
	keys := make(map[string]bool)
	for _, e := range section.entries {
		keys[e.keys] = true
	}
	for _, want := range []string{"Tab", "p", "m", "e", "s", "d", "Esc"} {
		if !keys[want] {
			t.Errorf("expected MANUAL REQUEST section to document %q (matches CLAUDE.md's Manual Request table)", want)
		}
	}
	if keys["a"] {
		t.Errorf("expected the stale 'a' entry (never implemented) to be gone")
	}
}

func TestHelpSectionsIncludeCustomTagRenameDelete(t *testing.T) {
	var section *helpSection
	for i := range helpSections {
		if helpSections[i].title == "LEFT PANEL" {
			section = &helpSections[i]
		}
	}
	if section == nil {
		t.Fatalf("expected a LEFT PANEL section")
	}
	keys := make(map[string]bool)
	for _, e := range section.entries {
		keys[e.keys] = true
	}
	if !keys["R"] || !keys["D"] {
		t.Errorf("expected LEFT PANEL to document R (rename) and D (delete) for custom tags, matching CLAUDE.md's Left Panel table")
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
