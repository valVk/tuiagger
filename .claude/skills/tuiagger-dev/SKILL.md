---
name: tuiagger-dev
description: Development guide for tuiagger (Go/Bubbletea, Elm-architecture TUI for OpenAPI specs) — how Model/Update/View composition, tea.Cmd, and persistence wire together, and the recipe for adding a new keyboard-driven feature. Use when working in cmd/tuiagger or internal/.
---

# tuiagger dev guide

`CLAUDE.md` at repo root is the source of truth for feature scope and
keybindings.

## Architecture

Bubbletea's Elm architecture: there is exactly one place a key becomes a
state transition — the root `Update(tea.Msg, Model) (Model, tea.Cmd)` — and
it's a plain function, directly unit-testable with no terminal mocking.

## Adding a new keyboard-driven feature

1. **Model owns state, Update owns transitions, View owns rendering.**
   Never mutate state inside `View()`. Never call I/O inside `Update()`
   directly — return a `tea.Cmd` (a `func() tea.Msg`) and handle the result
   as a message on the next `Update` call. This is what makes `Update`
   testable with plain structs and no goroutine/IO mocking.

2. **Compose, don't duck-type.** Each panel/section is its own small
   `Model` with its own `Update`/`View`. The root `Model` holds child models
   as typed fields and delegates: `m.viewer, cmd = m.viewer.Update(msg)`. A
   parent can only pass its child a message or read its child's exported
   fields — there is no "reach into a sibling's setter" by construction.

3. **Mode as a sum type.** Model a mode as a small `type Mode int` enum or a
   tagged struct, defined once, imported everywhere it's switched on.

4. **Async work (HTTP execution, file I/O) is always a `tea.Cmd`.** Never
   block inside `Update`. Pattern: `Update` returns a `tea.Cmd` closure that
   does the work and returns a result `tea.Msg`; a later `Update` call
   handles that message.

5. **Persistence keeps the atomic-write discipline** (write to `.tmp` then
   rename) for every stored file — overrides, saved requests, auth,
   environments.

6. **Write the `Update` unit test before wiring the real `tea.Program`.**
   Since `Update` is a pure function, test it directly:
   `m, cmd := model.Update(tea.KeyMsg{...})`, assert on `m`'s fields. Reserve
   manual smoke-testing (`go run ./cmd/tuiagger`) for confirming the `View()`
   render and real terminal behavior — `Update` correctness should already
   be proven by the time you run it.

7. **Update `CLAUDE.md`** keybinding tables and feature bullets to match.

## Dependencies are vendored locally, not committed

`vendor/` exists on disk (Go auto-uses it — no `-mod=vendor` flag needed —
as long as it stays consistent with `go.mod`/`go.sum`) but is
`.gitignore`d, not committed. `go.sum` alone is sufficient for reproducible
builds; `go build`/`go test` resolve dependencies from the local module
cache or proxy when `vendor/` is stale or absent. A fresh clone doesn't need
`vendor/` at all — plain `go build` works. Vendoring here is purely a local
build-speed/offline convenience, not a reproducibility requirement.

**After any `go get` or manual `go.mod` edit, run `task vendor`** (tidy +
vendor) locally to keep your own `vendor/` consistent with `go.mod`/`go.sum`
— otherwise a `-mod=vendor` build (if you ever force one) will fail with a
vendor-inconsistency error. There's nothing to stage for `vendor/` itself
since it's gitignored; do still stage `go.mod`/`go.sum`.

## Verifying changes

- `task check` (fmt-check + vet + test) after every edit round. See
  `Taskfile.yml` at repo root for the full list (`build`, `run`, `test`,
  `vet`, `fmt`, `vendor`, `clean`); run `task` with no args to list them.
- Manual smoke test: `go run ./cmd/tuiagger <collection>` against the
  Petstore v3 fixture (`https://petstore3.swagger.io/api/v3/openapi.json`)
  — full-screen Bubbletea apps have no real `/dev/tty` in a sandboxed shell,
  so ask the user to smoke-test interactively rather than trying to drive it
  with `expect`/`script`.

## Key dependencies

- `github.com/pb33f/libopenapi` — OpenAPI 3.0.x and 3.1.x parsing.
- `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` — no Bubbles
  component exists for the visual-select response viewer or the multi-section
  key-value editor; both are custom `Model`s built on the primitives.
- `atotto/clipboard` — yank target for response body / curl.
- `brianvoe/gofakeit` — `{{faker.*}}` interpolation.
