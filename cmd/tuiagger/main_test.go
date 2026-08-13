package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/valVK/tuiagger/internal/tui"
)

// stubTUI replaces launchTUI for the duration of a test so tests never take
// over the real terminal; it restores the real implementation on cleanup and
// returns a pointer to the built Model (nil until launchTUI runs).
func stubTUI(t *testing.T) *struct {
	model  tui.Model
	called bool
} {
	t.Helper()
	got := &struct {
		model  tui.Model
		called bool
	}{}
	original := launchTUI
	launchTUI = func(model tui.Model) error {
		got.model = model
		got.called = true
		return nil
	}
	t.Cleanup(func() { launchTUI = original })
	return got
}

func TestHelpFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Usage:") {
		t.Errorf("expected usage text, got %q", out.String())
	}
}

func TestVersionFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"-v"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), version) {
		t.Errorf("expected version in output, got %q", out.String())
	}
}

func TestListFlagNoCollections(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := run([]string{"--list"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if !strings.Contains(out.String(), "No collections found") {
		t.Errorf("unexpected output: %q", out.String())
	}
}

func TestNoArgsShowsErrorAndHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "Please provide") {
		t.Errorf("expected prompt for input, got %q", errOut.String())
	}
}

func TestUnknownNamedCollectionErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var out, errOut bytes.Buffer
	code := run([]string{"NoSuchCollection"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "not found") {
		t.Errorf("expected not-found error, got %q", errOut.String())
	}
}

func TestLocalSpecPathLaunchesTUI(t *testing.T) {
	got := stubTUI(t)
	var out, errOut bytes.Buffer
	code := run([]string{"../../internal/openapi/testdata/petstore.json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%q", code, errOut.String())
	}
	if !got.called {
		t.Fatalf("expected launchTUI to be called")
	}
	if got.model.Spec == nil || got.model.Spec.Spec.Info.Title == "" {
		t.Errorf("expected a parsed spec to be passed through, got %+v", got.model.Spec)
	}
	// Local file paths have no meaningful collection name.
	if got.model.CollectionName != "" {
		t.Errorf("expected empty collection name for a local path, got %q", got.model.CollectionName)
	}
}

func TestNamedCollectionPassesCollectionName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	collectionDir := home + "/.tuiagger/TestCol"
	if err := os.MkdirAll(collectionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec, err := os.ReadFile("../../internal/openapi/testdata/petstore.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(collectionDir+"/petstore.json", spec, 0o644); err != nil {
		t.Fatal(err)
	}

	got := stubTUI(t)
	var out, errOut bytes.Buffer
	code := run([]string{"TestCol"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%q", code, errOut.String())
	}
	if !got.called {
		t.Fatalf("expected launchTUI to be called")
	}
	if got.model.CollectionName != "TestCol" {
		t.Errorf("expected collection name TestCol, got %q", got.model.CollectionName)
	}
	if got.model.Store == nil {
		t.Errorf("expected a Store to be wired for a named collection")
	}
	if got.model.HTTPClient == nil {
		t.Errorf("expected an HTTPClient to be wired")
	}
}

func TestMissingSpecFileErrors(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"./does-not-exist.json"}, &out, &errOut)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "Error loading spec") {
		t.Errorf("expected load error, got %q", errOut.String())
	}
}
