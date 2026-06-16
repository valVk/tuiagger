# Twagger - TUI Swagger/OpenAPI Documentation Viewer

A terminal-based user interface for viewing and interacting with OpenAPI/Swagger documentation.

## Project Overview

Twagger is a CLI application that renders OpenAPI specifications in an interactive terminal interface. It uses a two-panel layout optimized for terminal navigation, with tag-based grouping and a fully functional "Try it out" feature for executing API requests directly from the terminal.

## Tech Stack

### Core Framework
- **Ink** (`ink` v5.x) - React-based library for building CLI applications
- **React 18+** with hooks (useState, useEffect, useMemo)
- **@inkjs/ui** - Pre-built UI components (Select, Spinner, TextInput)

### OpenAPI Parsing
- **@readme/openapi-parser** - Parse and validate OpenAPI specs (3.0.x and 3.1.x)

### HTTP Client
- **Native fetch** - For "Try it out" API execution

## Project Structure

```
twagger/
├── src/
│   ├── index.tsx                 # Entry point, CLI argument handling
│   ├── App.tsx                   # Main application component, two-panel layout
│   ├── components/
│   │   ├── Header.tsx            # API title, version, server selector
│   │   ├── LeftPanel.tsx         # Scrollable endpoints list panel
│   │   ├── RightPanel.tsx        # Details/editor panel container
│   │   ├── StatusBar.tsx         # Bottom keyboard shortcuts bar
│   │   ├── MethodBadge.tsx       # Colored HTTP method label
│   │   ├── ParametersSection.tsx # Parameters display
│   │   ├── ResponsesSection.tsx  # Response codes display
│   │   ├── KeyValueEditor.tsx    # Key-value pair editor
│   │   ├── ManualRequest.tsx     # Manual request builder
│   │   ├── ServerResponse.tsx    # Response display
│   │   └── Spinner.tsx           # Loading indicator
│   ├── hooks/
│   │   ├── useOpenAPI.ts         # OpenAPI spec loading/parsing
│   │   ├── usePanelNavigation.ts # Two-panel focus management
│   │   ├── useRequest.ts         # HTTP request execution
│   │   └── useSavedRequests.ts   # Saved requests persistence
│   ├── utils/
│   │   ├── parser.ts             # OpenAPI parsing utilities
│   │   ├── urlBuilder.ts         # URL construction with params
│   │   ├── curlGenerator.ts      # Curl command generation
│   │   ├── colors.ts             # HTTP method color mapping
│   │   └── storage.ts            # Saved requests file operations
│   └── types/
│       ├── openapi.ts            # OpenAPI spec type definitions
│       ├── request.ts            # Request/response types
│       └── index.ts              # Type exports
├── package.json
├── tsconfig.json
└── CLAUDE.md
```

## UI Layout - Two-Panel Design

The UI uses a two-panel layout optimized for terminal navigation. The left panel contains a scrollable list of tags and endpoints, while the right panel shows details for the selected endpoint.

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
│                            │                                                │
│                            │                          [ Try it out (t) ]    │
├────────────────────────────┴────────────────────────────────────────────────┤
│  j/k: navigate | h/l: panels | Enter: expand tag | t: try it | q: quit      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Layout Principles

1. **Left Panel (Endpoints List)** - 30% width, scrollable list of tags/endpoints
2. **Right Panel (Details)** - 70% width, shows selected endpoint details
3. **Panel Navigation** - `h`/`l` switch focus, active panel has highlighted border

## Keyboard Shortcuts

```
Panel Navigation:
  h / Left      - Focus left panel (endpoints list)
  l / Right     - Focus right panel (details)

Left Panel:
  j / Down      - Move to next endpoint/tag
  k / Up        - Move to previous endpoint/tag
  Enter         - Expand/collapse tag
  g / G         - Go to first/last item
  c / x         - Collapse/expand all tags

Right Panel:
  j / k         - Scroll content
  g             - Scroll to top

Actions:
  t             - Toggle "Try it out" mode
  e             - Execute request (in try-it-out mode)
  Esc           - Cancel / go back
  m             - Open manual request mode
  s             - Save request (in manual request mode)

Application:
  q             - Quit application
  Ctrl+r        - Reload spec
```

## Features

### Try It Out Mode
- Press `t` on an endpoint to enter edit mode
- Fill in parameters (path, query, header)
- Press `e` to execute the request
- View response with status, body, and curl command

### Manual Request Builder
- Press `m` to create custom requests not in the spec
- Assign to existing or custom tags
- Save for reuse (stored in `.twagger/saved-requests.json`)
- Saved requests marked with `*` in the left panel

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
# Load from collection (stored in ~/.twagger/<name>/)
twagger <collection-name>

# Load from file path or URL
twagger <spec-path-or-url>

# List available collections
twagger --list

# Examples
twagger PetStore                                           # Uses ~/.twagger/PetStore/openapi.json
twagger ./openapi.json                                     # Local file
twagger https://petstore3.swagger.io/api/v3/openapi.json  # URL
```

## Collections

Collections are stored in `~/.twagger/<name>/` directories. Each collection directory should contain an OpenAPI spec file (JSON or YAML).

```bash
# Create a collection
mkdir -p ~/.twagger/MyAPI
cp openapi.json ~/.twagger/MyAPI/

# List collections
twagger --list
```

## Test Data

Use the Petstore API for development and testing:
```
https://petstore3.swagger.io/api/v3/openapi.json
```

## Scope (v1)

**In Scope:**
- JSON OpenAPI specs (3.0.x, 3.1.x)
- Path, query, header parameters
- JSON request/response bodies
- Manual request builder with save/edit/delete
- Local persistence

**Out of Scope:**
- YAML spec support
- File uploads / FormData
- OAuth/API key authentication flows
- Request history
- Environment variables
