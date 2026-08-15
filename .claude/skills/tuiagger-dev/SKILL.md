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

`internal/tui` is a tree of these Elm-shaped units, not one flat `Model`.
Two different shapes coexist by design — know which one a new piece of
state actually is before reaching for either:

- **Independently-owned components** — `responseViewer`, `helpPopupState`,
  `saveDialogState`, `renameTagState`, and the info popup's three sections
  (`serversPanelState`/`authPanelState`/`environmentsPanelState`). Each is
  its own value type with `Update`/`View` methods (or, for a nested widget
  embedded inside a bigger one — `headerTableState`, the shared PARAMETERS
  row editor in `paramtable.go` — a richer signature than bare
  `Update(tea.Msg)`, since the parent already has the extra context to pass
  in and there's no value in forcing a uniform signature nothing calls
  uniformly). The root calls these as `m.viewer.Update(msg)`,
  `m.Help.View(...)`, etc. — a parent can only pass its child a message /
  extra args, or read its child's exported fields; there's no "reach into a
  sibling's internals" by construction.
- **Mode-routed sections that stay Model methods** — `TryIt`, `Manual`,
  `LeftPanel` (`leftpanel.go`), `Browse` (`browse.go`). These *look* like
  components (each gets its own state/keys/render/execute files) but their
  key handling and rendering stay `Model` methods
  (`handleTryItKey`, `renderTryItLines`, `handleBrowseKey`, ...) rather than
  methods on an owned value type. Don't force these into the first shape:
  their state (`FlatList`, `RightScroll`, `Store`, `Spec`) is genuinely
  root-shared, not something a self-contained value could own without
  either duplicating it or threading half the `Model` through as
  parameters anyway. The real signal for which shape a new piece of state
  wants: can it be constructed once and mutated only through its own
  methods, using nothing else off `Model`? If yes, first shape. If it
  fundamentally needs `Store`/`Spec`/sibling state on every call, second
  shape — and that's fine, don't manufacture a component for its own sake.

Components in the first shape that can't perform a side effect
themselves (closing a popup, persisting via `Store`, reading `Spec`) return
an extra result value instead — the parent applies it. This is the
established convention, not a one-off: `serversPanelState.Update` returns
`(next, selected int, closePopup bool)`, `saveDialogState.Update` /
`renameTagState.Update` return a small result enum
(`saveDialogSaved`/`saveDialogCancelled`/...), `headerTableState`'s
handlers return a merged param slice or `nil` for "unchanged". Follow this
shape for any new component that needs the same thing, rather than
inventing a different one per case.

## Adding a new keyboard-driven feature

1. **Model owns state, Update owns transitions, View owns rendering.**
   Never mutate state inside `View()`. Never call I/O inside `Update()`
   directly — return a `tea.Cmd` (a `func() tea.Msg`) and handle the result
   as a message on the next `Update` call. This is what makes `Update`
   testable with plain structs and no goroutine/IO mocking.

2. **Compose, don't duck-type** — see the Architecture section above for
   which of the two shapes a new piece of state actually is.

3. **Mode as a sum type.** Model a mode as a small `type Mode int` enum or a
   tagged struct, defined once, imported everywhere it's switched on.

4. **Async work (HTTP execution, file I/O) is always a `tea.Cmd`.** Never
   block inside `Update`. Pattern: `Update` returns a `tea.Cmd` closure that
   does the work and returns a result `tea.Msg`; a later `Update` call
   handles that message. `buildRequestSpec` (`execute.go`) is the shared
   baseURL/env/collector/`request.Spec` construction both try-it-out's and
   the manual builder's execute `tea.Cmd`s call into — extend that, don't
   re-derive it a third time.

5. **A component that needs data from elsewhere on every keystroke should
   capture its own snapshot once, not re-derive it from root state every
   `Update`.** `tryItState.Endpoint` is captured once by `enterTryIt`
   (instead of `handleTryItKey` re-deriving it from the left panel's
   current selection on every keystroke) — safe here because left-panel
   navigation is unreachable while `Mode == ModeTryIt`, so the selection
   genuinely can't change mid-session. Confirm the equivalent invariant
   holds before copying this pattern elsewhere.

6. **Panel auto-scroll: `scrollToShow` is for a moving single-row cursor,
   not a stationary focused region.** It pins a line bidirectionally (snaps
   back up if scroll drifts below it), which is right for PARAMETERS row
   navigation but was wrong for the BODY box: pinning its first line from
   above made it impossible to ever scroll past it once focused. Use the
   one-directional `scrollToShowBelow` for a stationary multi-line focused
   block instead (nudges into view once, never snaps back once the user
   scrolls past it deliberately).

7. **Any raw `RightScroll++`/`--` in a key handler must be clamped
   immediately via `clampRightScroll()`, not left to render-time
   clamping.** Render already clamps what it displays, but if the
   underlying field itself isn't also bounded at `Update` time, repeatedly
   scrolling past the content's end accumulates a hidden offset that then
   silently eats an equal number of presses in the other direction before
   the view visibly moves. `clampRightScroll` replicates the exact layout
   math (`rightPanelLayout`/`rightPanelLineCount`) the next render will
   use, so it stays in sync without needing an actual render.

8. **Syntax highlighting: hand-roll a small tokenizer + `lipgloss`, don't
   add a colorizer dependency.** `jsoncolor.go` (JSON/schema-outline) and
   `colorizeCurlLine`/`colorizeCurlFlag` in `responseviewer.go` (curl) are
   the existing examples — both regex-based, both reuse the app's existing
   palette (cyan = keys/names, green = values/body content, yellow =
   types, red = required/error, dim = punctuation/structure) rather than
   inventing new colors per feature. Follow that palette for any new
   highlighted text.

9. **Persistence keeps the atomic-write discipline** (write to `.tmp` then
   rename) for every stored file — overrides, saved requests, auth,
   environments.

10. **Write the `Update` unit test before wiring the real `tea.Program`.**
    Since `Update` is a pure function, test it directly:
    `m, cmd := model.Update(tea.KeyMsg{...})`, assert on `m`'s fields.

11. **Update `CLAUDE.md`** keybinding tables, feature bullets, and the
    project-structure file tree to match.

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
  `vet`, `fmt`, `vendor`, `clean`, `smoke`, `smoke-auto`); run `task` with
  no args to list them.
- **`task smoke-auto` (`scripts/smoke_test.py`) drives the real compiled
  binary through an actual pty, unattended** — stdlib `os.forkpty()` only,
  no third-party dependency (pip installs are blocked in a sandboxed shell
  anyway). It walks a representative slice of the keyboard surface (left
  panel nav/expand, endpoint select, try-it-out enter/exit, manual builder,
  info popup + section switch, help, clean quit) and asserts on
  ANSI-stripped screen content after each keystroke. Run this after any
  `internal/tui` change instead of assuming a sandboxed shell can't verify
  UI behavior at all — it can, just not interactively. Two things it had to
  handle that a naive pty harness won't: bubbletea/lipgloss send terminal
  capability queries (OSC 11 background-color, DSR cursor-position) on
  startup and stall ~5s waiting for a reply — the harness answers them like
  a real terminal would; and `HOME` is redirected to a temp dir per run so
  the isolated `PetStore` collection it seeds never touches the real
  `~/.tuiagger`.
- **`task smoke`** still exists for an actual human eyeball pass (real
  terminal, `~/.tuiagger/PetStore`) — use it for anything the automated
  walk doesn't cover (visual layout/color judgment calls, response-body
  visual-select/yank, anything needing a live network response).
- Prefer a scratch `_test.go` file (`internal/tui/zz_*_test.go`, deleted
  before committing) over ad hoc `fmt.Println` debugging when you need to
  see actual rendered/ANSI output while iterating — build the real
  `Model`, call the render function directly, print `stripANSI(...)` (or
  the raw string when checking escape codes themselves, e.g. confirming a
  background/foreground code is or isn't present). Faster feedback loop
  than round-tripping through the pty smoke test for a single render
  question, and doesn't require committing scratch assertions.

## Key dependencies

- `github.com/pb33f/libopenapi` — OpenAPI 3.0.x and 3.1.x parsing.
- `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` — no Bubbles
  component exists for the visual-select response viewer, the multi-section
  key-value editor, or JSON/schema syntax highlighting; all three are
  custom, built on the primitives (`bubbles/textarea` itself has no
  highlighting hook — see jsoncolor.go's doc comment).
- `atotto/clipboard` — yank target for response body / curl.
- `brianvoe/gofakeit` — `{{faker.*}}` interpolation.
