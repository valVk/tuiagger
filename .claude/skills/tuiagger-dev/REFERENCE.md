# tuiagger Architecture Reference

Companion to SKILL.md. `CLAUDE.md` at the repo root has the authoritative file
map and keybinding table — this file explains how the pieces move data around.

## State ownership

`App.tsx` is the single owner of interaction state. Nothing below it holds
state that another component needs to read — everything is props down,
callbacks up.

- `appState: AppState` — the mode machine: `browse | tryit | manual |
  renameTag`. Only one mode is active at a time; each mode's `useInput` blocks
  gate on `appState.mode`.
- `panelNav` (`usePanelNavigation`) — which panel is focused, left-list
  selection index, scroll offset, expanded tags, and the derived `flatList`
  (tags/endpoints/savedRequests flattened for rendering) and `selectedItem`.
- Per-endpoint editing state (`parameterValues`, `customParams`,
  `disabledParams`, `body`, `overridePath`, `overrideMethod`) — reset/reloaded
  in a `useEffect` keyed on `selectedItem?.id`, pulling from
  `overrides.getOverride()` if one exists.
- Data hooks (`useSavedRequests`, `useOverrides`, `useAuth`,
  `useEnvironments`) — each wraps `useStorage`, which loads once on mount and
  writes through on every `setStore` call. There is no explicit "save" step;
  mutating the returned store persists immediately (atomically, via
  `persistence.ts`'s temp-file-then-rename).

## Data flow: spec vs. saved requests

Two parallel object systems get merged into one flat list for the left panel:

- **Spec-derived**: `useOpenAPI` parses the OpenAPI file once
  (`utils/parser.ts`) into `endpoints` + `tags`. Endpoints without a spec
  `tags` array fall into the literal tag `'default'`.
- **App-level (saved requests)**: `useSavedRequests` owns `requests[]` and
  `customTags[]`, persisted to `saved-requests.json`. A request's `tag` field
  is just a string — there's no foreign key, no id-based grouping. Tags are
  identified purely by name equality.

`App.tsx` merges them: `allTags = savedRequests.getAllTags(spec.tags)`, which
unions spec tags, `customTags` names, and *any tag name actually referenced by
a request* (this last part matters — it's how a request can carry a tag that
isn't in `customTags`, e.g. `'default'`, and still show up as a group).
`usePanelNavigation.buildFlatList` then interleaves tag rows with their
endpoints and saved requests when expanded.

**Custom vs. spec vs. reserved tags** — this distinction gates which
mutations are legal:
- Spec tags (from the OpenAPI file) — read-only, editing means editing the spec file (out of scope).
- `'default'` — reserved fallback bucket, always selectable when saving a
  request, never renamable/deletable.
- Custom tags (`savedRequests.customTags`) — the only tags a user can rename
  or delete from the UI. Always check membership in `customTags` (by name)
  before allowing a tag-level mutation.

## Component tree (interaction-relevant parts)

```
App.tsx
├── Header               (title, server, active env)
├── HelpPopup / InfoPopup (overlay, mutually exclusive with main layout)
├── ManualSaveDialog      (overlay when appState.manual.showSaveDialog)
├── LeftPanel             (presentational; renders panelNav.flatList)
├── RightPanel            (browse/tryit details + tag preview + rename/delete UI)
│   └── ManualRequestPanel (swapped in instead of RightPanel when appState.mode === 'manual')
└── StatusBar             (mode-driven shortcut hints, bottom bar)
```

`RightPanel` and `LeftPanel` never call hooks that own app state — they take
everything as props (including callbacks for the few interactive bits they
render, like the tag-rename `TextInput` and the y/n confirm box).

## Persistence files (per collection)

Written by `utils/persistence.ts`, all atomic (write `.tmp`, then rename):

| File | Hook | Notes |
|---|---|---|
| `saved-requests.json` | `useSavedRequests` | `requests[]`, `customTags[]` |
| `overrides.json` | `useOverrides` | per-endpoint param/body overrides, keyed `METHOD /path` |
| `auth.json` | `useAuth` | credentials keyed by security scheme name |
| `environments.json` | `useEnvironments` | named variable sets + active index |

None of these files are required to exist — every loader falls back to an
empty/default store on read failure, so a fresh collection with just
`openapi.json` works.
