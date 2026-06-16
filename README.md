# Twagger

A terminal-based UI for viewing and interacting with OpenAPI/Swagger documentation. Navigate endpoints, execute requests, and manage API collections - all without leaving the terminal.

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
│  ▶ user (8)                │                                                │
│                            │                          [ Try it out (t) ]    │
├────────────────────────────┴────────────────────────────────────────────────┤
│  q:quit  i:info  ?:help  Ctrl+r:reload          h/l:panels  j/k:scroll     │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Features

- **Two-panel layout** - scrollable endpoint list on the left, details on the right
- **Try it out** - execute requests directly from the terminal with live responses
- **Manual request builder** - create and save custom requests not in the spec
- **Collections** - store named API specs in `~/.twagger/` for quick access
- **Environments** - named variable sets (`dev`, `staging`, `prod`) with `{{variable}}` interpolation
- **Faker interpolation** - generate realistic test data with `{{faker.internet.email()}}` syntax
- **Auth support** - configure Bearer token, Basic auth, or API key in the info panel
- **Server switching** - select from servers defined in the spec
- **Visual response selection** - enter visual mode to select and yank response body lines
- **Curl generation** - every executed request shows its equivalent curl command

## Installation

### Homebrew (macOS)

```bash
brew tap valVK/twagger
brew install twagger
```

### Uninstall

```bash
brew uninstall twagger
brew untap valVK/twagger
```

### From source

```bash
git clone https://github.com/valVK/twagger
cd twagger
npm install
npm run build
npm link        # makes `twagger` available globally
```

## Usage

```bash
# Load a saved collection
twagger PetStore

# Load from a local file
twagger ./openapi.json

# Load from a URL
twagger https://petstore3.swagger.io/api/v3/openapi.json

# List saved collections
twagger --list
```

## Collections

Collections are directories under `~/.twagger/<name>/` containing an OpenAPI spec file.

```bash
# Create a collection
mkdir -p ~/.twagger/MyAPI
cp openapi.json ~/.twagger/MyAPI/

# Open it
twagger MyAPI
```

Saved manual requests and auth/environment config are stored per-collection.

## Keyboard Shortcuts

Press `?` inside twagger to open the full interactive cheatsheet.

### Global

| Key | Action |
|-----|--------|
| `q` | Quit |
| `Ctrl+R` | Reload spec |
| `i` | Open info panel (servers / auth / environments) |
| `[` | Toggle left panel width |
| `?` | Toggle help cheatsheet |

### Navigation

| Key | Action |
|-----|--------|
| `h` / `←` | Focus left panel |
| `l` / `→` | Focus right panel |

### Left Panel

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `Enter` | Expand / collapse tag |
| `g` / `G` | First / last item |
| `c` / `x` | Collapse / expand all tags |

### Right Panel (browse)

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll down / up |
| `g` | Scroll to top |
| `t` | Enter try-it-out mode |
| `e` | Quick execute (reuses saved params) |
| `m` | Open manual request builder |
| `\` | Toggle request / response tab |
| `/` | Cycle response status tabs |

### Try It Out

| Key | Action |
|-----|--------|
| `e` | Execute request |
| `p` | Edit path |
| `m` | Cycle HTTP method |
| `r` | Reset overrides |
| `Esc` | Exit try-it-out |

### Parameters / Headers

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate rows |
| `i` | Edit value |
| `←` / `→` | Cycle enum values |
| `d` | Toggle enable / disable |
| `x` | Delete custom row |
| `c` | Cycle param type (query / path) |
| `Tab` | Move to next section |

### Response Body

| Key | Action |
|-----|--------|
| `J` / `K` | Scroll down / up |
| `g` / `G` | Jump to top / bottom |
| `v` | Toggle visual selection |
| `y` | Yank selection (or full body) to clipboard |
| `Esc` | Cancel visual mode |

### Info Panel (`i`)

| Key | Action |
|-----|--------|
| `Tab` | Switch section (Servers / Auth / Environments) |
| `j` / `k` | Navigate items |
| `Enter` | Select server / activate environment |
| `Esc` | Close panel |

### Environments

| Key | Action |
|-----|--------|
| `n` | New environment |
| `e` | Edit variables |
| `x` | Delete environment |
| `i` | Add / edit variable |
| `Esc` | Back to environment list |

### Manual Request (`m`)

| Key | Action |
|-----|--------|
| `Tab` | Next field |
| `a` | Add query / header row |
| `d` | Delete selected row |
| `e` | Execute request |
| `s` | Save request |
| `Esc` | Close |

## Faker Interpolation

Use `{{faker.*.*()}}` syntax in parameter values and request bodies to generate realistic test data:

```
{{faker.internet.email()}}
{{faker.person.fullName()}}
{{faker.string.uuid()}}
{{faker.number.int()}}
```

## Environment Variables

Create environments in the info panel (`i` → Tab to Environments) and reference variables with `{{variableName}}` in parameter values, headers, and request bodies:

```
base_url  =  https://staging.api.example.com
api_key   =  sk-staging-abc123
```

Then use `{{base_url}}` or `{{api_key}}` anywhere in your request. The active environment is shown in the header bar.

## Development

```bash
npm run build       # compile TypeScript
npm run dev         # watch mode
npm start -- PetStore   # run against a collection
```

Test with the Petstore API:
```
https://petstore3.swagger.io/api/v3/openapi.json
```

## Tech Stack

- [Ink](https://github.com/vadimdemedes/ink) - React for CLI
- [@readme/openapi-parser](https://github.com/readmeio/openapi-parser) - OpenAPI 3.x parsing
- [@faker-js/faker](https://fakerjs.dev/) - test data generation
- TypeScript

## License

MIT
