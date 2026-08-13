package main

import (
	"bytes"
	"strings"
	"testing"
)

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

func TestLocalSpecPathLoadsAndSummarizes(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"../../internal/openapi/testdata/petstore.json"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d, stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "endpoint(s)") {
		t.Errorf("expected summary output, got %q", out.String())
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
