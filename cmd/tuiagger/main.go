// Command tuiagger is a terminal UI for browsing and exercising OpenAPI specs.
package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/valVK/tuiagger/internal/openapi"
	"github.com/valVK/tuiagger/internal/storage"
)

const version = "v0.1.0-go"

const helpText = `
tuiagger - TUI Swagger/OpenAPI Documentation Viewer

Usage:
  tuiagger <collection>           Load from ~/.tuiagger/<collection>/
  tuiagger <spec-path-or-url>     Load from file path or URL

Examples:
  tuiagger PetStore
  tuiagger ./openapi.json
  tuiagger https://petstore3.swagger.io/api/v3/openapi.json

Options:
  --help, -h     Show this help message
  --version, -v  Show version number
  --list, -l     List available collections
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run contains all argument handling and exits nowhere itself, so it's
// testable without spawning a subprocess.
func run(args []string, stdout, stderr io.Writer) int {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	if hasFlag(args, "--version", "-v") {
		fmt.Fprintln(stdout, "tuiagger", version)
		return 0
	}
	if hasFlag(args, "--list", "-l") {
		printCollections(stdout)
		return 0
	}

	if len(args) == 0 || args[0] == "" {
		fmt.Fprintln(stderr, "Error: Please provide a collection name, path, or URL")
		fmt.Fprint(stderr, helpText)
		return 1
	}

	input := args[0]
	collection, err := storage.ResolveCollection(input)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	if collection == nil {
		fmt.Fprintf(stderr, "Error: Collection %q not found in ~/.tuiagger/\n\n", input)
		fmt.Fprintln(stderr, "Make sure the directory exists and contains an OpenAPI spec file (JSON/YAML).")
		printCollections(stderr)
		return 1
	}

	parsed, err := openapi.ParseOpenAPISpec(collection.Source)
	if err != nil {
		fmt.Fprintf(stderr, "Error loading spec: %v\n", err)
		return 1
	}

	// Phase 2 replaces this with the Bubbletea program launch; Phase 1 only
	// proves the foundation (resolve -> parse -> persistence wiring) works
	// end to end.
	fmt.Fprintf(stdout, "%s v%s — %d endpoint(s) across %d tag(s)\n",
		parsed.Spec.Info.Title, parsed.Spec.Info.Version, len(parsed.Endpoints), len(parsed.Tags))
	return 0
}

func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		if slices.Contains(names, a) {
			return true
		}
	}
	return false
}

func printCollections(w io.Writer) {
	collections, err := storage.ListCollections()
	if err != nil || len(collections) == 0 {
		fmt.Fprintln(w, "No collections found in ~/.tuiagger/")
		fmt.Fprintln(w, "\nTo create a collection:")
		fmt.Fprintln(w, "  mkdir -p ~/.tuiagger/MyAPI")
		fmt.Fprintln(w, "  cp openapi.json ~/.tuiagger/MyAPI/")
		return
	}
	fmt.Fprintln(w, "Available collections:")
	fmt.Fprintln(w)
	for _, name := range collections {
		fmt.Fprintln(w, " ", name)
	}
}
