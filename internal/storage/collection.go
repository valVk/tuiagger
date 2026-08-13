package storage

import (
	"os"
	"path/filepath"
	"strings"
)

var internalFiles = map[string]bool{
	authFile:          true,
	overridesFile:     true,
	savedRequestsFile: true,
	environmentsFile:  true,
}

type CollectionConfig struct {
	Name   string
	Source string
	Path   string
}

// BaseDir is ~/.tuiagger, where named collections live.
func BaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, storageDirName), nil
}

// ResolveCollection maps a CLI argument to a spec source, matching
// collectionResolver.ts's resolveCollection: URLs and anything that looks
// like a path (contains a separator or a spec extension) are used directly;
// anything else is looked up as a named collection under ~/.tuiagger. A nil
// result (no error) means "not found" — mirrors the TS function returning
// null rather than throwing.
func ResolveCollection(nameOrPath string) (*CollectionConfig, error) {
	if strings.HasPrefix(nameOrPath, "http://") || strings.HasPrefix(nameOrPath, "https://") {
		return &CollectionConfig{Name: "Remote", Source: nameOrPath, Path: nameOrPath}, nil
	}

	if strings.ContainsAny(nameOrPath, "/\\") ||
		strings.HasSuffix(nameOrPath, ".json") ||
		strings.HasSuffix(nameOrPath, ".yaml") ||
		strings.HasSuffix(nameOrPath, ".yml") {
		return &CollectionConfig{Name: "Local", Source: nameOrPath, Path: nameOrPath}, nil
	}

	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	collectionDir := filepath.Join(base, nameOrPath)

	if _, err := os.Stat(collectionDir); err != nil {
		return nil, nil
	}

	entries, err := os.ReadDir(collectionDir)
	if err != nil {
		return nil, nil
	}

	var specFile string
	for _, e := range entries {
		name := e.Name()
		isSpecExt := strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
		if isSpecExt && !internalFiles[name] {
			specFile = name
			break
		}
	}
	if specFile == "" {
		return nil, nil
	}

	return &CollectionConfig{
		Name:   nameOrPath,
		Source: filepath.Join(collectionDir, specFile),
		Path:   collectionDir,
	}, nil
}

// ListCollections lists subdirectories of ~/.tuiagger.
func ListCollections() ([]string, error) {
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return []string{}, nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
