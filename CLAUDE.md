# Tuiagger - TUI Swagger/OpenAPI Documentation Viewer

A terminal-based user interface for viewing and interacting with OpenAPI/Swagger documentation.

## Project Overview

Tuiagger is a CLI application that renders OpenAPI specifications in an interactive terminal interface. It uses a two-panel layout optimized for terminal navigation, with tag-based grouping, a fully functional "Try it out" feature, environments, auth, and a manual request builder.

## Tech Stack

### Core Framework
- **Bubbletea** (`github.com/charmbracelet/bubbletea`) - Elm-architecture TUI framework
- **Lipgloss** - terminal styling
- **Bubbles** - text input / textarea components

### OpenAPI Parsing
- **pb33f/libopenapi** - Parse and validate OpenAPI specs (3.0.x and 3.1.x)

### Test Data Generation
- **gofakeit** - Realistic test data via `{{faker.*.*()}}` interpolation

### HTTP Client
- **Native net/http** - For "Try it out" API execution

## Project Structure

```
tuiagger/
├── cmd/
│   └── tuiagger/
│       └── main.go                    # Entry point, CLI argument handling
├── internal/
│   ├── tui/
│   │   ├── model.go                   # Root Model, Update, chrome guards + Mode dispatch
│   │   ├── view.go                    # Root View, two-panel layout, shared render helpers
│   │   ├── colors.go                  # HTTP method / status color mapping
│   │   ├── jsoncolor.go               # Hand-rolled JSON/schema-outline syntax highlighting
│   │   ├── flatlist.go                # Left-panel tag/endpoint/saved-request data model
│   │   ├── leftpanel.go               # Left panel: key handling + render
│   │   ├── browse.go                  # Browse mode: key handling + render
│   │   ├── tryit_state.go             # Try-it-out: state struct, enter/exit, scaffoldFor
│   │   ├── tryit_keys.go              # Try-it-out: key handling
│   │   ├── tryit_render.go            # Try-it-out: rendering
│   │   ├── tryit_execute.go           # Try-it-out: request execution
│   │   ├── manual_state.go            # Manual builder: state struct, enter/exit
│   │   ├── manual_keys.go             # Manual builder: key handling
│   │   ├── manual_render.go           # Manual builder: rendering
│   │   ├── manual_execute.go          # Manual builder: request execution
│   │   ├── contenttype.go             # contentTypeCycle — shared content-type cycling (tryit + manual)
│   │   ├── bodybox.go                 # Shared BODY box render (tryit + manual)
│   │   ├── execute.go                 # buildRequestSpec — shared by tryit/manual execute
│   │   ├── headertable.go             # Shared HEADERS table sub-widget (tryit + manual)
│   │   ├── paramtable.go              # Shared custom/add-new PARAMETERS row sub-widget
│   │   ├── responseviewer.go          # Response body + curl viewer (visual select/yank)
│   │   ├── infopopup.go               # Info popup: thin parent, section dispatch
│   │   ├── infopopup_servers.go       # Info popup: SERVERS section
│   │   ├── infopopup_auth.go          # Info popup: AUTH section
│   │   ├── infopopup_env.go           # Info popup: ENVIRONMENTS section
│   │   ├── savedialog.go              # Save dialog for manual requests
│   │   ├── renametag.go               # Rename-tag overlay
│   │   └── help.go                    # Interactive keyboard shortcut cheatsheet
│   ├── bodyformat/
│   │   └── bodyformat.go              # Encode/WireEncode — JSON/form-urlencoded/XML body serialization
│   ├── openapi/
│   │   ├── parser.go                  # OpenAPI spec loading/parsing (libopenapi)
│   │   ├── schema.go                  # Schema traversal helpers
│   │   └── scaffold_fake.go           # Auto-generate request body from schema
│   ├── request/
│   │   ├── builder.go                 # Build full request from spec + overrides
│   │   ├── collector.go               # Collect/merge parameters from spec
│   │   ├── curl.go                    # Curl command generation
│   │   ├── executor.go                # HTTP request execution
│   │   ├── faker.go                   # {{faker.*}} interpolation
│   │   ├── interpolate.go             # {{env}} variable interpolation
│   │   ├── urlbuilder.go              # URL construction with params
│   │   └── client.go                  # HttpClient interface + net/http adapter
│   └── storage/
│       ├── collection.go              # Collection name -> spec path resolution
│       ├── persistence.go             # Atomic file I/O for all stored data
│       └── types.go                   # Override/saved-request/auth/env types
├── scripts/
│   └── smoke_test.py                  # Unattended pty-driven smoke test (task smoke-auto)
├── go.mod
└── CLAUDE.md
```

## UI Layout - Two-Panel Design

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Swagger Petstore v3.0                    Server: [petstore3.swagger.io ▼]  │
├────────────────────────────┬────────────────────────────────────────────────┤
│  ENDPOINTS                 │  GET /pet/findByStatus                         │
│                            │  Find Pets by status                           │
│  ▼ pet (8)                 ├────────────────────────────────────────────────┤
│    PUT    /pet             │                                                │
│    POST   /pet             │  Multiple status values can be provided with   │
│  > GET    /pet/findByStat  │  comma separated strings                       │
│    GET    /pet/findByTags  │                                                │
│    GET    /pet/{petId}     │  PARAMETERS                                    │
│    POST   /pet/{petId}     │  ┌────────────────────────────────────────────┐│
│    DELETE /pet/{petId}     │  │ status * (query)              string       ││
│    POST   /pet/{petId}/..  │  │ Status values for filter                   ││
│                            │  │ Enum: available | pending | sold           ││
│  ▶ store (4)               │  └────────────────────────────────────────────┘│
│                            │                                                │
│  ▶ user (8)                │  RESPONSES                                     │
│                            │  ┌──────┬─────────────────────────────────────┐│
│                            │  │ 200  │ successful operation                ││
│                            │  │ 400  │ Invalid status value                ││
│                            │  └──────┴─────────────────────────────────────┘│
│                            │                          [ Try it out (t) ]    │
├────────────────────────────┴────────────────────────────────────────────────┤
│  q:quit  i:info  ?:help  Ctrl+r:reload          h/l:panels  j/k:scroll     │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layout Principles

1. **Left Panel** - 30% width, scrollable list of tags/endpoints
2. **Right Panel** - 70% width, selected endpoint details or manual request builder
3. **Panel Navigation** - `h`/`l` switch focus, active panel has highlighted border
4. **Overlays** - InfoPopup (`i`) and HelpPopup (`?`) render over both panels

## Keyboard Shortcuts

```
Global:
  q             - Quit
  Ctrl+r        - Reload spec
  i             - Toggle info panel (servers / auth / environments)
  ?             - Toggle help cheatsheet
  [             - Toggle left panel width

Panel Navigation:
  h / Left      - Focus left panel
  l / Right     - Focus right panel

Left Panel:
  j / k         - Move down / up
  Enter         - Expand/collapse tag
  g / G         - First / last item
  c / x         - Collapse / expand all tags
  R             - Rename tag (custom tags only)
  D             - Delete tag, with confirm if non-empty (custom tags only)

Right Panel (browse):
  j / k         - Scroll content
  g             - Scroll to top
  t             - Enter try-it-out mode
  e             - Quick execute (reuses saved overrides)
  m             - Open manual request builder
  E             - Edit selected saved request
  D             - Delete selected saved request
  \             - Toggle request / response tab
  /             - Cycle response status tabs

Try It Out:
  e             - Execute request
  p             - Edit path override
  m             - Cycle HTTP method
  r             - Reset overrides
  Esc           - Exit try-it-out
  k (at first PARAMETERS row) - Focus the HEADERS section
  j (past last param) - Focus the BODY section
  i (body focused)     - Edit body (scaffolds realistic data if empty)
  c (body focused)     - Cycle the request body's content type (json / form / xml)
  k / Esc (body focused) - Back to parameters / exit try-it-out

Parameters / Headers:
  j / k         - Navigate rows
  i             - Edit value
  Left / Right  - Cycle enum values
  d             - Toggle enable / disable
  x             - Delete custom row
  c             - Cycle param type (query / path)
  Tab           - Move to next section

Response Body:
  J / K         - Scroll down / up
  j / k         - Also work as J/K while a visual selection is active
  g / G         - Jump to top / bottom
  v             - Toggle visual selection mode
  y             - Yank selection (or full body) to clipboard
  C             - Yank the generated curl command to clipboard
  Esc           - Cancel visual mode

Info Panel (i):
  Tab           - Switch section (Servers / Auth / Environments)
  j / k         - Navigate items
  Enter         - Select server / activate environment
  Esc           - Close panel

Environments:
  n             - New environment
  e             - Edit variables
  x             - Delete environment
  i             - Add / edit variable
  Esc           - Back to environment list

Manual Request (m):
  p             - Edit path
  m             - Cycle HTTP method
  e             - Execute request
  s             - Save request (opens name/tag dialog)
  d             - Delete request (only while editing a saved request via 'E')
  Esc           - Close (discards an unsaved draft)
  k (at first PARAMETERS row) - Focus the HEADERS section
  j (past last PARAMETERS row, write methods only) - Focus the BODY section
  c (body focused)     - Cycle the request body's content type (json / form / xml)
  (row editing inside PARAMETERS/HEADERS follows the Parameters / Headers
  table above: j/k move, i edit/add, x delete, d toggle enable, c cycle
  query/path type — PARAMETERS only, HEADERS entries are always type
  "header")

Save Dialog (s):
  Tab           - Switch field (name / tag)
  Left / Right  - Cycle existing tags / create a new one
  Enter         - Confirm field, then save
  Esc           - Cancel (back to the builder, back out of new-tag entry)
```

## Features

### Try It Out
- Press `t` on any endpoint to enter edit mode
- Fill in path, query, header parameters; the request body auto-fills with realistic generated data and is fully editable (`j` past the last parameter row to focus it, `i` to edit)
- When an endpoint declares more than one request body content type (`application/json`, `application/x-www-form-urlencoded`, `application/xml`), cycle between them with `c` while BODY is focused — the body is re-scaffolded and the `Content-Type` header sent on execute follows the selected tab
- `application/x-www-form-urlencoded` bodies are shown/edited as plain `key=value` lines (one field per line, unescaped; a plain array repeats its key, e.g. `tags=a` / `tags=b`), not the actual percent-encoded wire text — percent-encoding happens once, automatically, right before the request is sent
- Press `e` to execute; view response with status, headers, body, and curl command
- Parameter values, body, and the selected content type persist per-endpoint as overrides in `.tuiagger/overrides.json`

### Manual Request Builder
- Press `m` to create custom requests not defined in the spec
- Cycle the request body's content type (`application/json` / `application/x-www-form-urlencoded` / `application/xml`) with `c` while BODY is focused — no spec to read declared types from, so this is always the same fixed 3-way choice
- Assign to existing or new custom tags, or leave untagged (goes to the `default` tag)
- Save for reuse (stored in `.tuiagger/saved-requests.json`)
- Saved requests appear with `*` in the left panel
- Custom tags can be renamed (`R`) or deleted (`D`, with confirmation) from the left panel; spec-derived tags and `default` are read-only

### Environments
- Create named variable sets (e.g. `dev`, `staging`, `prod`) in the info panel
- Reference variables with `{{variableName}}` in parameter values, headers, and bodies
- Active environment shown in the header bar
- Stored per-collection in `environments.json`

### Auth
- Configure Bearer token, Basic auth, or API key in the info panel (`i` → Auth)
- Applied automatically to executed requests
- Stored per-collection in `auth.json`

### Faker Interpolation
- Use `{{faker.internet.email()}}`, `{{faker.person.fullName()}}`, etc. in any field
- Body scaffolding auto-generates realistic values from the response schema

### External Editor
- Edit request body in `$EDITOR` (falls back to `vi`)
- Temp file named `tuiagger-body-<timestamp>.json`

### Response Viewer
- Visual selection mode (`v`) to select lines, `y` to yank to clipboard
- `C` yanks the generated curl command to clipboard, independent of the current tab/selection
- Scroll large bodies with `J`/`K`

### Method Badge Colors

| Method  | Color   |
|---------|---------|
| GET     | Blue    |
| POST    | Green   |
| PUT     | Yellow  |
| DELETE  | Red     |
| PATCH   | Cyan    |
| HEAD    | Magenta |
| OPTIONS | Gray    |

## CLI Usage

```bash
tuiagger <collection>                                      # ~/.tuiagger/<collection>/
tuiagger <spec-path-or-url>                                # local file or URL
tuiagger --list                                            # list collections
tuiagger --help
tuiagger --version

# Examples
tuiagger PetStore
tuiagger ./openapi.json
tuiagger https://petstore3.swagger.io/api/v3/openapi.json
```

## Collections

Stored in `~/.tuiagger/<name>/`. Each directory holds an OpenAPI spec plus per-collection data files written by the app:

```
~/.tuiagger/MyAPI/
├── openapi.json          # your spec (bring your own)
├── overrides.json        # saved try-it-out parameter values
├── saved-requests.json   # manual requests
├── auth.json             # auth credentials
└── environments.json     # named environment variable sets
```

```bash
mkdir -p ~/.tuiagger/MyAPI
cp openapi.json ~/.tuiagger/MyAPI/
tuiagger MyAPI
```

## Persistence

All state is written to the collection directory (or `.tuiagger/` in cwd for non-collection loads). Writes are atomic (write to `.tmp` then rename) to avoid corruption.

## Services / DI

`internal/request` defines an `HttpClient` interface. The production adapter uses native `net/http`. This seam exists to allow the HTTP layer to be swapped (e.g. for tests) without touching request logic.

## Test Data

```
https://petstore3.swagger.io/api/v3/openapi.json
```

## Scope

**In Scope:**
- JSON and YAML OpenAPI specs (3.0.x, 3.1.x)
- Path, query, header parameters
- Request bodies: JSON, `application/x-www-form-urlencoded`, XML (scaffolding, editing, and sending); response bodies stay a read-only display, format-agnostic
- Manual request builder with save/edit/delete
- Environments with variable interpolation
- Auth (Bearer, Basic, API key)
- Faker interpolation in parameter values and bodies
- Body scaffolding from schema
- External editor (`$EDITOR`) for request bodies
- Local persistence per collection

**Out of Scope:**
- File uploads / `multipart/form-data` (`application/x-www-form-urlencoded` is supported; multipart needs a file-attach UI and binary content this app doesn't have)
- OAuth flows
- Request history
- WebSocket / streaming responses
