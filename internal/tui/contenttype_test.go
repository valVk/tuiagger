package tui

import "testing"

func TestContentTypeCycleSelectedWrapsIndex(t *testing.T) {
	c := contentTypeCycle{types: []string{"a", "b", "c"}}
	cases := map[int]string{0: "a", 1: "b", 2: "c", 3: "a", -1: "c"}
	for tab, want := range cases {
		if got := c.Selected(tab); got != want {
			t.Errorf("Selected(%d) = %q, want %q", tab, got, want)
		}
	}
}

func TestContentTypeCycleSelectedEmptyFallsBackToJSON(t *testing.T) {
	c := contentTypeCycle{}
	if got := c.Selected(0); got != "application/json" {
		t.Errorf("expected application/json for an empty cycle, got %q", got)
	}
}

func TestContentTypeCycleNextWrapsAndNoOpsOnSingleEntry(t *testing.T) {
	c := contentTypeCycle{types: []string{"a", "b"}}
	if got := c.Next(0); got != 1 {
		t.Errorf("Next(0) = %d, want 1", got)
	}
	if got := c.Next(1); got != 0 {
		t.Errorf("Next(1) = %d, want 0 (wrap)", got)
	}
	single := contentTypeCycle{types: []string{"a"}}
	if got := single.Next(0); got != 0 {
		t.Errorf("Next on a single-entry cycle should be a no-op, got %d", got)
	}
}

func TestContentTypeCycleIndexOf(t *testing.T) {
	c := contentTypeCycle{types: []string{"a", "b", "c"}}
	if got := c.IndexOf("c"); got != 2 {
		t.Errorf("IndexOf(%q) = %d, want 2", "c", got)
	}
	if got := c.IndexOf("not-there"); got != 0 {
		t.Errorf("expected IndexOf to fall back to 0 for an unknown type, got %d", got)
	}
}
