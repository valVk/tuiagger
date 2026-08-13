package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCollectionURL(t *testing.T) {
	cfg, err := ResolveCollection("https://petstore3.swagger.io/api/v3/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Name != "Remote" || cfg.Source != cfg.Path {
		t.Errorf("unexpected: %+v", cfg)
	}
}

func TestResolveCollectionLocalPath(t *testing.T) {
	for _, in := range []string{"./openapi.json", "spec.yaml", "spec.yml", "/abs/path.json"} {
		cfg, err := ResolveCollection(in)
		if err != nil {
			t.Fatal(err)
		}
		if cfg == nil || cfg.Name != "Local" {
			t.Errorf("expected Local for %q, got %+v", in, cfg)
		}
	}
}

func TestResolveCollectionNamedNotFound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := ResolveCollection("DefinitelyNotARealCollectionName12345")
	if err != nil {
		t.Fatal(err)
	}
	if cfg != nil {
		t.Errorf("expected nil for missing collection, got %+v", cfg)
	}
}

func TestResolveCollectionNamedFindsSpecIgnoresInternalFiles(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	base, err := BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	collectionDir := filepath.Join(base, "TestCollectionXYZ")
	if err := os.MkdirAll(collectionDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Internal files should never be picked as the spec file, even though
	// they match the extension filter.
	for _, f := range []string{"overrides.json", "auth.json", "saved-requests.json", "environments.json"} {
		if err := os.WriteFile(filepath.Join(collectionDir, f), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(collectionDir, "petstore.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ResolveCollection("TestCollectionXYZ")
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatalf("expected collection to resolve")
	}
	if cfg.Source != filepath.Join(collectionDir, "petstore.json") {
		t.Errorf("expected petstore.json as spec, got %q", cfg.Source)
	}
	if cfg.Path != collectionDir {
		t.Errorf("expected path to be collection dir, got %q", cfg.Path)
	}
}

func TestListCollections(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	base, err := BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(base, "TestListCollectionsXYZ")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	names, err := ListCollections()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range names {
		if n == "TestListCollectionsXYZ" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected TestListCollectionsXYZ in %v", names)
	}
}
