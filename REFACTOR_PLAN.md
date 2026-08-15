# Component-system redesign for internal/tui

## Context

The just-completed refactor (5 commits, merged via PR #4) split `tryit.go`/
`manual.go` into per-concern files and extracted shared sub-widgets
(`headertable.go`, `paramtable.go`), but the architecture review that
followed found the same "one big file, three tangled concerns" shape still
present in `infopopup.go`, and the root `Model`/`View()` still mixing
generic Elm-loop plumbing with one specific mode's (browse's) logic.

The user wants to go further than fixing those two spots: separate
**representation** (rendering) from **business logic** (state transitions)
throughout `internal/tui`, and decompose the flat `Model` into small,
independently-owned components — one state + one update-logic + one render
each, composed into a tree, the way a Vue SFC tree composes. Communication
between components goes through `tea.Msg`/`tea.Cmd` (Bubbletea's own
message-passing idiom), not raw goroutine channels — channels would fight
Bubbletea's single-threaded `Update`/`View` loop, confirmed via exploration
that the codebase already treats `tea.Cmd`-returning closures as its only
async primitive (HTTP execution, clipboard yank, yank-expiry timers all go
through this pattern already; no goroutines/channels exist directly in
`internal/tui`).

## Component contract

Bubbletea's own `tea.Model` interface is:

```go
type Model interface {
    Init() Cmd
    Update(Msg) (Model, Cmd)
    View() string
}
```

Implementing this *literally* for every sub-component forces `Update` to
return the `tea.Model` interface, which means every call site needs a type
assertion to get a concrete type back — Bubbletea's own bundled components
(`bubbles/textinput`, `bubbles/textarea`, already used throughout this
codebase) don't do this either. They use the same three method names with a
**concrete** return type: `Update(tea.Msg) (textinput.Model, tea.Cmd)`. This
plan follows that existing, already-in-use convention — a documented pattern,
not a compiler-checked interface:

```go
func (c T) Init() tea.Cmd
func (c T) Update(msg tea.Msg) (T, tea.Cmd)
func (c T) View() string
```

This applies to every **top-level, Mode-routed** component (the ones the root
`Model` dispatches `tea.Msg` to directly). Nested sub-widgets that are only
ever driven by their owning component — `headerTableState`, `paramTable`
inside TryIt/Manual — keep their current richer signatures (they take
slices of their parent's owned data each call, the way a Vue child receives
props). That split already matches this repo's own composition philosophy
(`.claude/skills/tuiagger-dev`: "a parent can only pass its child a message
or read its child's exported fields") — it doesn't need to change, just be
named consistently at the top level.

## Target component tree

Root `Model` keeps: chrome (`Spec`, `Width`/`Height`, `Quitting`,
`SpecLoading`/`SpecError`, `Source`, `CollectionName`), `Mode`, DI services
(`HTTPClient`, `Store`), and one field per top-level component:

| Component | New/existing file(s) | Replaces |
|---|---|---|
| `ResponseViewer` | responseviewer.go | Same file, reshaped: owns `Response`/`Curl` internally (currently passed in on every `render` call from the root `Model`); `handleKey`→`Update`, `render`→`View`; consumes `yankExpiredMsg` itself instead of the root `Model.Update` special-casing it |
| `ServersPanel`, `AuthPanel`, `EnvironmentsPanel` + thin `InfoPopup` parent | infopopup_servers.go, infopopup_auth.go, infopopup_env.go, infopopup.go | infopopup.go (656 lines, 3 tangled concerns) |
| `HelpPopup` | help.go | Same file, method rename only (already self-contained) |
| `SaveDialog` | savedialog.go | Same file, method rename only (already self-contained; stays nested under Manual since it's only ever entered from there) |
| `RenameTagPopup` | renametag.go | Same file, method rename only |
| `TryIt` | tryit_state.go / tryit_keys.go / tryit_render.go / tryit_execute.go | Same files, thin `Init`/`Update`/`View` facade added; **captures its `*openapi.ParsedEndpoint` once at `Init`** instead of the root re-deriving `m.selectedItem()` on every keystroke — makes `Update` self-sufficient from `msg` alone |
| `Manual` | manual_state.go / manual_keys.go / manual_render.go / manual_execute.go | Same files, thin facade only (already self-contained, no left-panel dependency) |
| `LeftPanel` | flatlist.go + new leftpanel_keys.go/leftpanel_render.go | `handleLeftPanelKey` (model.go), the left-panel slice of view.go |
| `Browse` | new browse_render.go | The endpoint-doc-display branch currently inlined in `renderRightPanel` (view.go) |

Root `Model.Update` becomes: the true cross-cutting guards (quit,
reload-error, window resize, `responseMsg`/`reloadMsg` — since `Spec` is
root-owned, reload has to stay root-level) plus a dispatch of `tea.Msg` to
whichever component `Mode` selects. Root `Model.View` becomes: header +
active component's `.View()` + status bar, replacing the current `switch`
of bespoke `renderXPopup`/`renderXPanel` calls.

## Staging (task check after every stage, same discipline as the last refactor)

Branch: `refactor/tui-component-system`, based on `master` (post-merge of
the tryit/manual split via PR #4, so this branch has `tryit_state.go`/
`manual_state.go`/etc. to build on). Commit this plan to the repo as
`REFACTOR_PLAN.md` in the first commit, checklist-updated per stage, deleted
in the final stage — same pattern as last time.

- [x] Stage 0 — commit this plan
- [x] Stage 1 — ResponseViewer → real component. `Response`/`Curl` moved
      from `Model` into `responseViewer` (populated from `responseMsg`
      inside its new `Update`, consumed by its new `View`). Root `Model`
      forwards `responseMsg`/`yankExpiredMsg`/gated `tea.KeyMsg` into
      `Update` instead of special-casing `yankExpiredMsg` itself. Kept the
      existing, directly-tested `handleKey(keyStr string)`/
      `render(resp, curl, active, width)`/`yankCurl(curl string)` as the
      implementation `Update`/`View` wrap, rather than rewriting
      already-well-tested internals just to match the naming convention —
      `Update`/`View` are the new component-contract entry points, the
      lower-level methods are unchanged plumbing underneath.
- [x] Stage 2 — Split infopopup.go into `serversPanelState`/
      `authPanelState`/`environmentsPanelState` (infopopup_servers.go/
      infopopup_auth.go/infopopup_env.go), each with its own state +
      Update-shaped methods + View. `infoPopupState` (infopopup.go) is the
      thin parent — owns which section is active, dispatches to whichever
      one. `handleInfoKey` is the real Mode-routed Update; the three panels
      underneath keep richer per-section signatures (string key + explicit
      context args), the same convention headerTableState already
      established. `activeEnvName` (used by the header bar independent of
      the popup) moved to view.go rather than into environmentsPanelState,
      since it doesn't touch that component's own state.
- [x] Stage 3 — `HelpPopup`/`SaveDialog`/`RenameTagPopup` get `Update`/
      `View` methods. Skipped adding a no-op `Init()` where a component has
      no setup beyond a zero-value literal (`helpPopupState{}`,
      `saveDialogState` built by `newSaveDialogState`, `renameTagState`
      built by `enterRenameTag`) — YAGNI, matches this project's own
      "don't add for scenarios that can't happen." `helpPopupState` is new
      (state used to be flat `HelpScroll` on `Model`); `saveDialogState`/
      `renameTagState` already existed, just gained `Update`/`View`.
      `Update` on all three returns a result signal (mirrors
      `headerTableState`'s convention) for the Model-owned side effects
      (closing the popup, persisting via `Store`) the component can't do
      itself.
- [x] Stage 4 — TryIt/Manual. Added `tryItState.Endpoint`, captured once
      by `enterTryIt`; `handleTryItKey`/`exitTryIt`/`handleBodyFocusedKey`
      now read `m.TryIt.Endpoint` instead of re-deriving it via
      `m.selectedItem()` on every keystroke (safe: left-panel navigation
      is unreachable while `Mode == ModeTryIt`, so the selection can't
      change mid-session). No literal `Init`/`Update`/`View` renames on
      `handleTryItKey`/`renderTryItLines`: they're already Mode-dispatch
      entry points called from `Model.Update`/`Model.View`, the same
      shape `handleInfoKey`/`renderInfoPopup` kept in Stage 2 rather than
      being renamed — the real component boundary in this codebase is
      "does the root call it as `m.X.Update(...)` on an independent
      value" (ResponseViewer, HelpPopup, SaveDialog, RenameTag, the info
      popup's 3 panels), not every Mode branch. Manual needed no
      changes — already self-contained, no left-panel dependency to
      remove.
- [x] Stage 5 — LeftPanel extraction: `handleLeftPanelKey`/
      `safeLeftIndex`/`selectedItem`/`toggleTag`/`allExpanded`
      (model.go) + `renderLeftPanel`/`renderListRow` (view.go) moved into
      new leftpanel.go, alongside flatlist.go's existing data builder.
      Kept as Model methods, not a value-type component: LeftIndex/
      FlatList/ExpandedTags are root-shared (RightScroll/ResponseTab
      resets belong to the right panel; FlatList is rebuilt from Spec/
      Store-derived data every mode reads) — same call as TryIt/Manual in
      Stage 4, file-locality is the real win here, not an owned-value
      boundary that would just duplicate root state.
- [x] Stage 6 — Browse extraction: `handleRightPanelKey`/
      `sortedResponseCodes` (model.go) and `renderTagLines`/
      `renderEndpointLines` (view.go) moved into new browse.go — browse
      mode's key handling and content rendering, giving it its own file
      alongside every other mode. `renderRightPanel` (the bordered-box +
      scroll-clamp chrome shared by browse *and* try-it-out) stays in
      view.go as shared panel infrastructure, same reasoning as keeping
      `paramEditWidgets`/`renderCustomParamRow` shared rather than
      duplicated per mode.
- [ ] Stage 7 — Root Model becomes a thin dispatcher. `Update`/`View`
      shrink to chrome guards + component dispatch; delete now-dead
      bespoke `handleXKey`/`renderXLines` wrapper functions in
      model.go/view.go that the facades superseded. Delete
      `REFACTOR_PLAN.md` in this final commit.

## Verification

- `task check` (fmt-check + vet + test) after every stage — must stay green;
  stop and diagnose on red before continuing, same bar as the last refactor.
- Full `go test ./... -v` skim at the end to confirm no assertion was
  silently dropped while chasing green (existing 213 tests are the behavior
  contract; renamed methods need only mechanical call-site updates in
  `*_test.go`, not new test logic — a logic change needed to stay green
  signals a real behavior slip, not a safe refactor).
- Manual `task smoke` pass by the user at the end for real-terminal
  confirmation (this environment can't drive a TTY) — same keyboard-driven
  checklist as before: HEADERS/PARAMETERS nav, try-it-out, manual builder
  save/edit/delete, info popup (servers/auth/environments), help, response
  viewer visual-select/yank.
