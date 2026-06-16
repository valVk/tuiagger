---
name: Twagger missing features and improvements
description: Gaps vs classic Swagger UI and UX bugs — prioritized backlog for future implementation
type: project
originSessionId: 5d298355-e5e7-4cef-85d4-50447ecf8342
---

## Done

1. ~~**Multi-line body editor**~~ — `react-ink-textarea` in `RightPanel.tsx`.
2. ~~**Server switcher**~~ — Header + InfoPopup (`i`) with j/k navigation.
3. ~~**Enum value cycling**~~ — `i` seeds first value, `←/→` cycles.
4. ~~**Response headers display**~~ — `\` toggles request/response tab.
5. ~~**Request body schema display**~~ — schema in browse, scaffold placeholder in try-it.
6. ~~**Response schema display**~~ — status code tabs, `/` cycles.
7. ~~**Authentication**~~ — Bearer/apiKey in `i` panel, persisted per-collection in `auth.json`.
9. ~~**Body scaffold (faker.js)**~~ — `src/utils/scaffoldBody.ts`, triggered on `t`, preserves overrides.
13. ~~**Faker interpolation**~~ — `{{faker.internet.email()}}` syntax in body/params/headers, fresh on every send.
- ~~**Dead code removed**~~ — `EndpointCard`, `TagGroup`, `ServerResponse` deleted.

## Must-have for release

8. **Search/filter** — `f` key to filter endpoint list. Left panel unusable on large specs (50+ endpoints).

10. **Response body viewer (vim-like)** — can't read long responses. Need independent scroll + `v/V/y/gg/G/h/l`. Plan already written.

11. **Per-request custom headers** — dedicated Headers section in try-it (like Postman). Mechanism exists via `customParams in: 'header'` but UX is buried.

## Nice to have

12. **Environment/variable switcher** — named envs (`dev`, `staging`, `prod`) with `{{variableName}}` substitution at execute time. Storage: `~/.twagger/<collection>/environments.json`.
