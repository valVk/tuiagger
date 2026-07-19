---
name: tuiagger-dev
description: Development guide for the tuiagger codebase (Ink/React TUI for OpenAPI specs) — how state, keyboard hooks, and panels wire together, and the recipe for adding a new keyboard-driven feature. Use when adding/modifying keybindings, panel behavior, or app-level state in this repo, or when orienting to its architecture for the first time.
---

# tuiagger Dev Guide

Full file/feature map lives in `CLAUDE.md` at the repo root — read that first for
project layout, keybinding list, and feature scope. This skill covers the
*pattern* for wiring changes into that architecture. Deeper data-flow notes are
in [REFERENCE.md](REFERENCE.md).

## Adding a new keyboard-driven feature

This is the recipe used for tag rename/delete (`R`/`D` on a tag row) — follow it
for any new interaction gated on app mode or selection state.

1. **Decide where the key lives.** Three keyboard layers, each with its own
   `useInput` and `isActive` gate:
   - `usePanelNavigation.ts` — left/right panel focus, scroll, tag expand (`j/k/g/G/Enter/c/x`)
   - `useAppKeyboard.ts` — mode transitions (`m`, `t`, `e`, `E`, `D` on saved requests) and manual/tryit-mode actions
   - Popup-local hooks (`useServersKeyboard`, `useAuthKeyboard`, etc.) — scoped to `InfoPopup` sections
   Pick the layer whose `isActive`/mode guard already matches when your key should fire. Don't add a fourth global listener — extend an existing gated block.

2. **If the feature needs a new mode** (multi-key input, not a single action),
   add a variant to the `AppState` union — **in both** `src/App.tsx` and
   `src/hooks/useAppKeyboard.ts` (the type is duplicated between them; keep them
   in sync manually, there's no shared import). Example: `RenameTagState =
   { mode: 'renameTag'; tagName: string; value: string }`.

3. **Gate other handlers.** Any `useInput` block that must NOT fire while your
   new mode/confirm is active needs an explicit early return (see the
   `tagDeleteConfirm` early-return in `useAppKeyboard.ts` — it blocks `m`/`t`/`e`
   etc. while a y/n prompt is pending). Forgetting this lets stray keystrokes
   leak through to the wrong handler.

4. **Text input inside a mode**: use `ink-text-input`'s `TextInput` with
   `focus={true}`, controlled value from `AppState`, `onSubmit` to commit. Only
   handle `Esc` yourself in a small scoped `useInput` — don't hand-roll
   character-by-character capture (see `ManualSaveDialog.tsx`'s new-tag field or
   the tag-rename input in `RightPanel.tsx`).

5. **Wire props down, don't reach into hooks from components.** `App.tsx` owns
   all state and passes callbacks/data as props into `LeftPanel`/`RightPanel`/
   `ManualRequestPanel`. Components stay presentational.

6. **Confirm/destructive actions**: reuse the inline yellow double-border y/n
   prompt pattern already in `RightPanel.tsx` (see `showResetConfirm` and
   `tagDeleteConfirm`) rather than introducing a new dialog style.

7. **Update `StatusBar.tsx`** dynamic shortcuts only if the hint isn't already
   shown elsewhere — avoid duplicating a hint that's already visible in the
   right panel for the focused item (this was reverted once already for tag
   rename/delete: the per-item hint in `RightPanel.tsx` was kept, the StatusBar
   entry was removed).

8. **Update `CLAUDE.md`** keybinding tables and feature bullets to match —
   it's the source of truth other sessions read first.

## Verifying changes

`npm run build` (tsc) is the fastest correctness check and catches prop/type
drift across the App.tsx ↔ hook ↔ component boundary described above — run it
after every edit round.

The full-screen Ink UI generally **cannot be driven interactively in a
sandboxed shell** (no real `/dev/tty`; `expect`/`script` pty wrappers fail to
render the alt-screen buffer). Don't burn time trying — rely on `tsc`, a
careful read-through of the diff, and ask the user to smoke-test interactively
before treating a UI change as verified. (Plain readline/CLI code outside the
Ink tree — like `src/init.ts` — *can* be tested with `expect` since it's not
full-screen.)
