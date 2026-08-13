---
name: tuiagger-dev-go
description: Development guide for the Go/Bubbletea rewrite of tuiagger (Elm-architecture TUI for OpenAPI specs) — how Model/Update/View composition, tea.Cmd, and persistence wire together, and the recipe for adding a new keyboard-driven feature. Use when working in cmd/tuiagger or internal/ (Go code), or orienting to the Go rewrite for the first time. For the legacy Ink/TS app in src/, use tuiagger-dev instead.
---

# tuiagger Go rewrite — dev guide

Full plan: `~/.claude/plans/purrfect-drifting-newt.md`. Progress/status: `HANDOFF.md`
at repo root — **read that first**, it says which phase is done and what's next.

This is the Go rewrite of the Ink/TS app in `src/` (see `tuiagger-dev` skill for
that codebase — it's the behavior reference until Phase 8 cutover deletes it).
`CLAUDE.md` at repo root is still the source of truth for *feature scope and
keybindings*, regardless of implementation language.

## Why the architecture differs from the TS version

The architecture review that preceded this rewrite found the TS app's core
weakness: 9 keyboard hooks each reimplement the same dispatch idiom
independently, `useAppKeyboard` duck-types into 4 other hooks' setters, and
nothing is unit-testable because Ink's `useInput` lives inside React hooks.

Bubbletea's Elm architecture removes the *category* of bug, not just the
instance: there is exactly one place a key becomes a state transition — the
root `Update(tea.Msg, Model) (Model, tea.Cmd)` — and it's a plain function.
The Phase 0 spike (`internal/spike/viewer/model.go`) proved this: the
response-viewer visual-select/yank state machine, the single hardest widget
in the TS app to test, became 5 plain `go test` cases with no terminal
mocking.

## Adding a new keyboard-driven feature

1. **Model owns state, Update owns transitions, View owns rendering.**
   Never mutate state inside `View()`. Never call I/O inside `Update()`
   directly — return a `tea.Cmd` (a `func() tea.Msg`) and handle the result
   as a message on the next `Update` call. This is what makes `Update`
   testable with plain structs and no goroutine/IO mocking.

2. **Compose, don't duck-type.** Each panel/section is its own small
   `Model` (see `internal/spike/viewer.Model` as the template) with its own
   `Update`/`View`. The root `Model` holds child models as typed fields and
   delegates: `m.viewer, cmd = m.viewer.Update(msg)`. This directly replaces
   the TS app's pattern of `useAppKeyboard` reaching into `panelNav.setActivePanel`
   etc. — there is no equivalent "reach into a sibling's setter" here by
   construction; a parent can only pass its child a message or read its
   child's exported fields.

3. **Mode as a sum type, not a duck-typed union.** Where the TS app used a
   TypeScript union duplicated between `App.tsx` and `useAppKeyboard.ts`
   (kept in sync manually), Go's type system enforces this for free — model
   a mode as a small `type Mode int` enum or a tagged struct, defined once,
   imported everywhere it's switched on.

4. **Async work (HTTP execution, file I/O) is always a `tea.Cmd`.** Never
   block inside `Update`. Pattern: `Update` returns a `tea.Cmd` closure that
   does the work and returns a result `tea.Msg`; a later `Update` call
   handles that message. This maps to the TS app's `useRequest.ts` /
   `useStorage.ts` async hooks — same shape, but the return-a-command
   discipline is enforced by the framework rather than by hook convention.

5. **Persistence keeps the atomic-write discipline from Phase 1**, including
   for saved-requests (the TS app's `saveSavedRequests` skipped atomic write —
   don't repeat that in the Go port; see architecture review Candidate D and
   `HANDOFF.md`'s Phase 1 next-steps).

6. **Write the `Update` unit test before wiring the real `tea.Program`.**
   Since `Update` is a pure function, test it directly:
   `m, cmd := model.Update(tea.KeyMsg{...})`, assert on `m`'s fields. Reserve
   manual smoke-testing (`go run ./cmd/tuiagger`) for confirming the `View()`
   render and real terminal behavior — `Update` correctness should already
   be proven by the time you run it.

7. **Update `CLAUDE.md`** keybinding tables and feature bullets to match, same
   as the TS workflow — it's the shared source of truth for both codebases
   until cutover.

## Verifying changes

- `go build ./...`, `go vet ./...`, `go test ./...` after every edit round —
  fast, and (unlike the TS app) actually exercises behavior, not just types.
- Manual smoke test: `go run ./cmd/tuiagger <collection>` against the
  Petstore v3 fixture (`https://petstore3.swagger.io/api/v3/openapi.json`,
  same fixture the TS app and the Phase 0 spike used) — full-screen Bubbletea
  apps have the same sandboxed-shell limitation Ink did (no real `/dev/tty`),
  so ask the user to smoke-test interactively rather than trying to drive it
  with `expect`/`script`.
- Cross-check behavior against the running TS app (`npm start` in `src/`)
  when in doubt about exact parity — it's still the reference until Phase 8.

## Confirmed dependencies (Phase 0 spike)

- `github.com/pb33f/libopenapi` — OpenAPI 3.0.x and 3.1.x parsing, confirmed
  against real Petstore fixture and a hand-built 3.1 fixture (webhooks,
  nullable-via-type-array). Not `kin-openapi`.
- `github.com/charmbracelet/bubbletea`, `bubbles`, `lipgloss` — no Bubbles
  component exists for the visual-select response viewer or the multi-section
  key-value editor; both are custom `Model`s built on the primitives, not
  drop-in reuse.
- Not yet spiked: `atotto/clipboard` (yank target), a Go faker library
  (`brianvoe/gofakeit` is the leading candidate) for `{{faker.*}}`
  interpolation parity — see Phase 6 in the plan.
