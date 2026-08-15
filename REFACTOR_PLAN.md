# Refactor: try-it-out / manual-request-builder architecture

## Context

`internal/tui/tryit.go` (1260 lines) and `internal/tui/manual.go` (805 lines)
are the two largest files in `internal/tui` by a wide margin (next is
`view.go` at 772). Both grew this way because Go's Elm-architecture model
(state + `Update` + `View` per concern, per `.claude/skills/tuiagger-dev`)
was applied at the *mode* level (try-it-out is one big state/handler/render
blob, manual builder is another) instead of also being applied to the
sub-widgets those two modes share.

Concretely, both modes independently implement:
- A HEADERS table (focus navigation, row edit, enable/disable toggle,
  delete) — `handleHeadersFocusedKey`/`handleHeaderEditKey` in `tryit.go`
  are near-line-for-line duplicates of `handleManualHeadersFocusedKey`/
  `handleManualHeaderEditKey` in `manual.go`, differing only in reading
  `m.TryIt.*` vs `m.Manual.*` fields.
- Custom/add-new PARAMETERS row editing — `handleCustomParamEditKey`'s
  non-spec branch and `handleManualParamEditKey` are the same shape.
- Request execution plumbing (`executeWithOverride` in `tryit.go`,
  `runRequestCmd` in `manual.go`) — both build a `request.Spec` from
  servers/auth/env/params/body the same way, with executeWithOverride
  additionally handling override persistence.

`manual.go`'s own comment on `handleManualHeadersFocusedKey` explicitly
notes this was a known tradeoff at the time ("kept as separate functions...
no cheap way to factor that out in Go without an interface neither call
site would otherwise need") — composition via an embedded sub-state (which
the TS app's own hook-per-concern split couldn't do, but Go structs can) is
the missing piece, and is exactly the pattern the project's own dev skill
already prescribes ("Compose, don't duck-type... a parent can only pass its
child a message or read its child's exported fields").

Goal: apply that composition pattern to eliminate the duplication (DRY),
split each mode's state/key-handling/rendering/execution into separate
files (SRP, addresses "huge module"), and do it as a pure refactor — no
behavior change, no new features (YAGNI), tests green at every step.

## Non-goals

- No change to keybindings, rendering output, or persistence format.
- No broader "Clean Architecture" layering rewrite (introducing use-case/
  interactor layers, etc.) — disproportionate for this app; the existing
  Elm-style state/update/view split is already the right shape, this
  refactor just applies it one level deeper.
- No dependency/library changes.

## Approach

Work happens on a new branch off `master` (confirm branch name / whether to
open a PR at the end). After **every** step below: run `task check`
(fmt-check + vet + test). If anything goes red, stop and diagnose the root
cause before continuing — do not proceed past a red test, and do not modify
tests to make them pass without first understanding whether the refactor
introduced a real behavior change.

### [x] Step 0 — Persist this plan in the repo

On the new branch, commit this document verbatim as `REFACTOR_PLAN.md` at
the repo root, before any code changes. Any agent picking up work on this
branch reads that file first for full context (why, scope, non-goals, and
the exact step sequence below) rather than re-deriving it. Update its
checklist status inline as each step below completes (e.g. `- [x] Step 2 —
...`) so progress is visible to whoever picks the branch up next, including
across separate sessions/agents. Delete `REFACTOR_PLAN.md` in the final
commit once Step 6 is done and the branch is ready for review — it's a
working document for the refactor itself, not permanent project
documentation (`CLAUDE.md` already covers the shipped architecture; update
that too if the refactor changes anything user-facing enough to warrant it,
though this refactor shouldn't).

### [x] Step 1 — Baseline

Run `task check` on the current branch tip and record the result (test
count, pass/fail) as the reference point every later step is compared
against.

Result: `task check` green (fmt-check, vet, test all pass). `go test
./... -v`: 213 PASS, 0 FAIL. Reference point for Steps 2-6.

### [x] Step 2 — Extract shared HEADERS-table component

New file `internal/tui/headertable.go`. Define a `headerTableState` struct
holding what's currently duplicated across `tryItState`/`manualState`:
`Focused bool`, `Cursor int`, `Editing bool`, plus the shared editing
widgets (`ParamField string`, `NameInput`, `ValueInput textinput.Model`).

Give it methods operating on itself plus the two split `CustomParameter`
slices it needs (headers being edited, the sibling "others" slice, merged
via the existing `mergeCustomParams` helper — reuse it, don't duplicate):
- `handleFocusedKey(msg, headers, others) (headerTableState, []storage.CustomParameter, tea.Cmd)`
- `handleEditKey(msg, headers, others) (headerTableState, []storage.CustomParameter, tea.Cmd)`

Body is the current `handleHeadersFocusedKey`/`handleHeaderEditKey` logic
verbatim (they're already identical to the `Manual` versions modulo field
names) — this is a mechanical extraction, not a rewrite.

Update `tryItState`/`manualState` (in `tryit.go`/`manual.go` — or their new
homes per Step 4) to embed `HeaderTable headerTableState` in place of their
separate `HeadersFocused`/`HeaderCursor`/`HeaderEditing` fields. This is the
step with the widest blast radius: ~60 existing references to
`TryIt.Headers*`/`Manual.Header*` across `tryit.go`, `manual.go`,
`view.go`, `savedialog.go`, and the test files need mechanical renaming to
`TryIt.HeaderTable.*`/`Manual.HeaderTable.*`. Do this rename with a search
across the package, not ad hoc — grep for the exact field names first to
get a complete list before editing.

Delete `handleHeadersFocusedKey`, `handleManualHeadersFocusedKey`,
`handleHeaderEditKey`, `handleManualHeaderEditKey` once call sites route
through the shared methods instead.

Run `task check`.

### [x] Step 3 — Extract shared custom/add-new PARAMETERS row editor

Similar composition for the non-spec-param editing path. New type in
`internal/tui/paramtable.go` (or add to `headertable.go` if the shapes end
up close enough to share — decide during implementation, don't force it)
covering the `enter*ParamEdit`/`handle*ParamEditKey` logic for custom rows
and the always-present add-new row — the part of `handleCustomParamEditKey`
(tryit.go) that's cursor-past-spec-params, and all of
`handleManualParamEditKey` (manual.go).

Try-it-out's spec-parameter editing (enum cycling, disabled-row guard) has
no manual-builder equivalent (manual requests have no OpenAPI spec to
source enums/required-ness from) — that part stays in `tryit.go`, only the
custom/add-new-row logic moves into the shared component.

Run `task check`.

### [x] Step 4 — Split each mode into files by responsibility

Once the shared components exist, split the remaining mode-specific code
by concern, matching the pattern `responseviewer.go` already uses (state +
`Update`-equivalent + render in one cohesive file per widget is fine when a
file is that size; it's the *mixing of four unrelated concerns inside 1200
lines* that's the actual problem, not file count for its own sake):

- `internal/tui/tryit_state.go` — `tryItState`, `enterTryIt`, `exitTryIt`,
  `resetOverride`, `isEmptyOverride`, small pure helpers
  (`tryItTotalRows`, `splitCustomParams`, `mergeCustomParams`,
  `applicationJSONSchema`, `enumValues`, `toStr`).
- `internal/tui/tryit_keys.go` — `handleTryItKey` and its sub-handlers
  (`handleParamEditKey`, `handleSpecParamEditKey`, `handleBodyFocusedKey`,
  `handleBodyEditKey`, `handlePathEditKey`).
- `internal/tui/tryit_render.go` — `renderTryItLines`,
  `renderTryItBodySection`, `renderHeadersSection` (or keep
  `renderHeadersSection` next to `headerTableState` in `headertable.go`
  since it renders that component specifically — decide based on which
  reads cleaner once written).
- `internal/tui/tryit_execute.go` — `executeCmd`, `quickExecuteCmd`,
  `executeWithOverride`, `loadEnvAndAuth` (shared with manual, could also
  live in a new `internal/tui/execute.go` — see Step 5).

Mirror the same split for `manual.go` →
`manual_state.go`/`manual_keys.go`/`manual_render.go`/`manual_execute.go`,
keeping `renameTagState`/`enterRenameTag`/`handleRenameTagKey` (unrelated
to the manual builder proper, just colocated today) wherever they fit best
— likely their own small file since they're a distinct concern.

This step is a pure move (`git mv`-then-edit-imports style), not a logic
change — no function bodies change here beyond what Steps 2–3 already
did. Run `task check` after the split.

### [x] Step 5 — Unify request-building duplication

`executeWithOverride` (tryit.go) and `runRequestCmd` (manual.go) both:
build `baseURL` from servers/selected index, call `loadEnvAndAuth`,
construct a `request.ParameterCollector`, and build a `request.Spec` with
the same field mapping. `executeWithOverride` additionally persists an
override first and has a body-scaffold fallback.

Extract the shared baseURL/collector/spec-construction into one helper
(e.g. `buildRequestSpec(...)` in `internal/tui/execute.go`), called by
both `executeWithOverride` and `runRequestCmd`/`manualExecuteCmd`. Keep
override-persistence and body-scaffolding as call-site-specific logic
layered on top, not forced into the shared helper (YAGNI — don't add
parameters to the shared function for behavior only one caller needs).

Run `task check`.

### [ ] Step 6 — Final verification

- `gofmt -l`, `go vet ./...`, full `go test ./...` (or `task check`) one
  more time end-to-end.
- Ask the user to run `task smoke` for a real-terminal pass (sandboxed
  shell here can't drive a TTY) — behavior must look identical to before
  the refactor: HEADERS/PARAMETERS navigation, edit/toggle/delete, body
  focus, execute, and the manual builder's save/edit/delete flow.
- Do not merge into `master` automatically — leave the branch (and PR, if
  one is opened) for the user's review.

## Key files

- `internal/tui/tryit.go` — being split (Steps 2–4)
- `internal/tui/manual.go` — being split (Steps 2–4)
- `internal/tui/view.go`, `internal/tui/savedialog.go` — read field
  references to `TryIt.Header*`/`Manual.Header*` that need renaming
  alongside Step 2
- `internal/tui/tryit_test.go`, `internal/tui/tryit_body_test.go`,
  `internal/tui/tryit_customparams_test.go`, `internal/tui/manual_test.go`
  — existing behavior contract; these should need only field-name updates
  (not new test logic) if the refactor is truly behavior-preserving. If a
  test needs a logic change (not just a rename) to stay green, stop — that
  signals an actual behavior change slipped in, not a safe refactor.
- Existing shared helpers to reuse, not re-implement: `mergeCustomParams`,
  `splitCustomParams`, `cycleQueryPath`, `renderCustomParamRow`,
  `renderAddParamRow`, `paramEditWidgets` (already shared between the two
  modes today — a working example of the composition pattern this plan
  extends).

## Verification

- `task check` after every step (fmt-check + vet + test) — must stay
  green throughout; red at any point means stop and diagnose before
  continuing.
- Final full `go test ./... -v` skim to confirm no test was silently
  weakened (e.g. an assertion dropped) while chasing green.
- Manual `task smoke` pass by the user for real-terminal confirmation,
  since this environment can't drive an interactive TTY.
