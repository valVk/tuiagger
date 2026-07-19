# Tuiagger - TUI Swagger/OpenAPI Documentation Viewer

A terminal-based user interface for viewing and interacting with OpenAPI/Swagger documentation.

## Project Overview

Tuiagger is a CLI application that renders OpenAPI specifications in an interactive terminal interface. It uses a two-panel layout optimized for terminal navigation, with tag-based grouping, a fully functional "Try it out" feature, environments, auth, and a manual request builder.

## Tech Stack

### Core Framework
- **Ink** (`ink` v7.x) - React-based library for building CLI applications
- **React 19+** with hooks (useState, useEffect, useMemo)
- **@inkjs/ui** v2 - Pre-built UI components (Select, Spinner, TextInput)

### OpenAPI Parsing
- **@readme/openapi-parser** v5 - Parse and validate OpenAPI specs (3.0.x and 3.1.x)

### Test Data Generation
- **@faker-js/faker** - Realistic test data via `{{faker.*.*()}}` interpolation

### HTTP Client
- **Native fetch** - For "Try it out" API execution

## Project Structure

```
tuiagger/
├── src/
│   ├── index.tsx                      # Entry point, CLI argument handling
│   ├── App.tsx                        # Root component, two-panel layout
│   ├── components/
│   │   ├── index.ts                   # Barrel exports
│   │   ├── Header.tsx                 # API title, version, active environment badge
│   │   ├── LeftPanel.tsx              # Scrollable tags/endpoints list
│   │   ├── RightPanel.tsx             # Details/editor panel container
│   │   ├── StatusBar.tsx              # Bottom keyboard shortcuts bar
│   │   ├── MethodBadge.tsx            # Colored HTTP method label
│   │   ├── ParametersSection.tsx      # Spec parameters display
│   │   ├── ResponsesSection.tsx       # Response codes display
│   │   ├── ResponseViewer.tsx         # Response body with visual select/yank
│   │   ├── KeyValueEditor.tsx         # Key-value pair editor (params/headers)
│   │   ├── SpecParamRow.tsx           # Single spec parameter row
│   │   ├── CustomParamRow.tsx         # Single custom/override parameter row
│   │   ├── AddNewParamRow.tsx         # Row for adding new custom parameters
│   │   ├── HeadersSection.tsx         # Request headers editor
│   │   ├── ManualRequestPanel.tsx     # Manual request builder panel
│   │   ├── ManualSaveDialog.tsx       # Save dialog for manual requests
│   │   ├── InfoPopup.tsx              # Info overlay (servers / auth / environments)
│   │   ├── ServersSection.tsx         # Server selector within InfoPopup
│   │   ├── AuthSection.tsx            # Auth config (Bearer / Basic / API key)
│   │   ├── EnvironmentsSection.tsx    # Named environment variable sets
│   │   ├── HelpPopup.tsx              # Interactive keyboard shortcut cheatsheet
│   │   └── Spinner.tsx                # Loading indicator
│   ├── hooks/
│   │   ├── index.ts                   # Barrel exports
│   │   ├── useAppKeyboard.ts          # Global keyboard handler
│   │   ├── usePanelNavigation.ts      # Two-panel focus management
│   │   ├── useRightPanelKeyboard.ts   # Right panel keyboard handler
│   │   ├── useManualPanelKeyboard.ts  # Manual request panel keyboard handler
│   │   ├── useParamNavigation.ts      # Parameter row navigation
│   │   ├── useHeadersNavigation.ts    # Headers section navigation
│   │   ├── useServersKeyboard.ts      # Server selector keyboard handler
│   │   ├── useAuthKeyboard.ts         # Auth section keyboard handler
│   │   ├── useEnvironmentsKeyboard.ts # Environments section keyboard handler
│   │   ├── useOpenAPI.ts              # OpenAPI spec loading/parsing
│   │   ├── useRequest.ts              # HTTP request execution
│   │   ├── useSavedRequests.ts        # Saved manual requests state
│   │   ├── useOverrides.ts            # Per-endpoint parameter overrides
│   │   ├── useAuth.ts                 # Auth credentials state
│   │   ├── useEnvironments.ts         # Environments state
│   │   └── useStorage.ts              # Generic async storage primitive
│   ├── utils/
│   │   ├── index.ts                   # Barrel exports
│   │   ├── collectionResolver.ts      # Resolve collection name → spec path
│   │   ├── parser.ts                  # OpenAPI parsing utilities
│   │   ├── urlBuilder.ts              # URL construction with params
│   │   ├── requestBuilder.ts          # Build full request from spec + overrides
│   │   ├── parameterCollector.ts      # Collect/merge parameters from spec
│   │   ├── curlGenerator.ts           # Curl command generation
│   │   ├── interpolate.ts             # {{faker.*}} and {{env}} interpolation
│   │   ├── scaffoldBody.ts            # Auto-generate request body from schema
│   │   ├── externalEditor.ts          # Open body in $EDITOR
│   │   ├── persistence.ts             # File I/O for all stored data
│   │   └── colors.ts                  # HTTP method color mapping
│   ├── contexts/
│   │   └── ServicesContext.tsx        # HttpClient DI context (fetch adapter)
│   └── types/
│       ├── index.ts                   # Type exports
│       ├── openapi.ts                 # OpenAPI spec type definitions
│       ├── request.ts                 # Request/response types
│       └── services.ts                # HttpClient interface + fetch adapter
├── package.json
├── tsconfig.json
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
  \             - Toggle request / response tab
  /             - Cycle response status tabs

Try It Out:
  e             - Execute request
  p             - Edit path override
  m             - Cycle HTTP method
  r             - Reset overrides
  Esc           - Exit try-it-out

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
  g / G         - Jump to top / bottom
  v             - Toggle visual selection mode
  y             - Yank selection (or full body) to clipboard
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
  Tab           - Next field
  a             - Add query / header row
  d             - Delete selected row
  e             - Execute request
  s             - Save request
  Esc           - Close
```

## Features

### Try It Out
- Press `t` on any endpoint to enter edit mode
- Fill in path, query, header parameters; edit request body
- Press `e` to execute; view response with status, headers, body, and curl command
- Parameter values and body persist per-endpoint as overrides in `.tuiagger/overrides.json`

### Manual Request Builder
- Press `m` to create custom requests not defined in the spec
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

### Response Viewer
- Visual selection mode (`v`) to select lines, `y` to yank to clipboard
- Scroll large bodies with `J`/`K`

### External Editor
- Edit request body in `$EDITOR` (falls back to `vi`)
- Temp file named `tuiagger-body-<timestamp>.json`

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

`ServicesContext` provides an `HttpClient` interface to the component tree. The production adapter uses native `fetch`. This seam exists to allow the HTTP layer to be swapped without touching request logic.

## Test Data

```
https://petstore3.swagger.io/api/v3/openapi.json
```

## Scope

**In Scope:**
- JSON and YAML OpenAPI specs (3.0.x, 3.1.x)
- Path, query, header parameters
- JSON request/response bodies
- Manual request builder with save/edit/delete
- Environments with variable interpolation
- Auth (Bearer, Basic, API key)
- Faker interpolation in parameter values and bodies
- Body scaffolding from schema
- External editor (`$EDITOR`) for request bodies
- Local persistence per collection

**Out of Scope:**
- File uploads / FormData
- OAuth flows
- Request history
- WebSocket / streaming responses
