# Project Rules

Desktop app: Wails v3 (Go backend in `main.go` + `internal/`) with a Svelte 5 + Vite
frontend in `frontend/`. Bun is the JS toolchain.

## Go

- Go 1.26. Keep logic in `internal/` (`app`, `deps`, `download`, `config`); `main.go`
  only wires services, events, and the window.
- New service methods exposed to the frontend must live on the services registered in
  `main.go` (`app.Service`, `app.DepsService`).
- Register every event type with `application.RegisterEvent[...]` in `main.go` — this
  is what makes the binding generator emit typed frontend APIs.
- `frontend/bindings/` is generator-owned. Never hand-edit it. Regenerate bindings
  (`wails3 generate bindings` / `wails3 build`) after changing service APIs or events.

## TypeScript / Svelte

- **No `any`**. Use `unknown` + type guard, or a proper interface/type.
- **No inline object types** in function signatures or callbacks. Extract to a named
  `interface` or `type` alias.
- Use `interface` for object shapes (preferred), `type` for unions/utility types.
- Svelte 5 runes only (`$state`, `$derived`, `$effect`...). No `svelte/store`, no
  legacy `export let` reactivity.
- Shared reactive state lives in `src/lib/store.svelte.ts`. Components stay
  presentational: read through `$derived`/function calls from the store, and send
  mutations back through exported store functions — don't talk to services directly
  from components.
- Backend calls go through `bindings/mldy/...` modules only. No raw `fetch`/HTTP to
  the backend; event updates arrive via `Events` from `@wailsio/runtime`.
- Use `@/` for everything under `src/` and `@bindings/` for Wails bindings; never
  use relative `./...` or `../...` import paths. Biome's organize-imports owns
  ordering (`:ALIAS:` group sits above `:PATH:`).

## Style

- Biome is the formatter and linter: single quotes, no semicolons, 2-space indent,
  100 width. Run `bun run lint` (which formats) — never fight it by hand.
- Go: standard `gofmt`.

## Process

- Before considering done, from `frontend/` run:
  - `bun run typecheck`
  - `bun run check` (svelte-check)
  - `bun run lint`
  and from the repo root: `go build ./...`.
- Never use browser testing unless explicitly asked. Verify UI behavior through
  type checks and non-browser tests.
- Never start the app (`wails3 dev`, `task run`, or `bin/mldy`) during verification.
  It's a desktop GUI; the developer runs application services.
