# Shared Chronograph Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a production-ready, installable web service for anonymous, server-authoritative shared stopwatches and countdown timers with separate control/view links.

**Architecture:** A Go modular monolith owns time, state transitions, persistence, SSE fan-out, retention, and exports. PostgreSQL stores an append-only event log plus current projections. A static SvelteKit/TypeScript PWA uses HTTP commands and SSE state updates; its build is embedded into the Go binary for one-container deployment.

**Tech Stack:** Go 1.24-compatible source (built with Go 1.26 in Docker), PostgreSQL 18, pgx v5, SvelteKit/Svelte 5, TypeScript, Vitest, Playwright, Server-Sent Events, Docker Compose, Caddy-compatible reverse-proxy deployment.

---

## File map

- `cmd/chronograph/main.go`: configuration, dependency wiring, lifecycle, HTTP server.
- `internal/clock/model.go`: clock/event types and pure projection calculations.
- `internal/clock/transition.go`: validated stopwatch/timer state machine and undo compensation.
- `internal/clock/*_test.go`: deterministic domain tests.
- `internal/store/store.go`: persistence interface and shared errors.
- `internal/store/memory.go`: concurrency-safe in-memory adapter used by unit/server tests and optional development.
- `internal/store/postgres.go`: PostgreSQL commands, snapshots, replay, retention, and export queries.
- `internal/store/migrations/*.sql`: versioned database schema.
- `internal/realtime/hub.go`: bounded instance subscriptions and non-blocking fan-out.
- `internal/httpapi/router.go`: routes, security headers, static delivery, and health endpoints.
- `internal/httpapi/handlers.go`: lazy instance creation, snapshots, commands, labels/order/highlight, deletion.
- `internal/httpapi/sse.go`: snapshot/replay SSE stream and heartbeats.
- `internal/httpapi/export.go`: streaming JSON and CSV exports.
- `internal/retention/worker.go`: free/premium archive and purge scheduling.
- `web/src/lib/*`: typed API client, SSE store, display-time projection, and reusable clock components.
- `web/src/routes/+page.svelte`: lazy local stopwatch landing view.
- `web/src/routes/c/[token]/+page.svelte`: Race Control deck.
- `web/src/routes/v/[token]/+page.svelte`: public board.
- `web/static/*`: PWA manifest/icons.
- `Dockerfile`, `compose.yaml`, `.env.example`: reproducible deployment.
- `README.md`: local development, deployment, operations, and product limitations.

### Task 1: Repository and toolchain bootstrap

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `Makefile`
- Create: `web/package.json`
- Create: `web/svelte.config.js`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/src/app.html`

- [ ] **Step 1: Initialize Git and create the isolated implementation branch/worktree**

Run the `using-git-worktrees` workflow, initialize a repository if necessary, commit the approved spec and this plan, and create branch `feat/shared-chronograph` in an isolated worktree.

- [ ] **Step 2: Add toolchain configuration**

Create a Go module named `chronograph`, a SvelteKit package with scripts `check`, `test`, `build`, and `test:e2e`, and Make targets `test`, `check`, `build`, and `verify`. Keep Go source compatible with the locally available Go 1.24 toolchain.

- [ ] **Step 3: Install pinned dependencies**

Run:

```bash
go get github.com/jackc/pgx/v5 github.com/jackc/pgx/v5/pgxpool
cd web && npm install
```

Expected: `go.sum` and `web/package-lock.json` are created without audit errors that block installation.

- [ ] **Step 4: Verify empty toolchains**

Run:

```bash
go test ./...
cd web && npm run check && npm test -- --run
```

Expected: commands exit 0 with no source-test failures.

- [ ] **Step 5: Commit**

```bash
git add .gitignore Makefile go.mod go.sum web
git commit -m "chore: bootstrap chronograph toolchains"
```

### Task 2: Clock domain through TDD

**Files:**
- Create: `internal/clock/model.go`
- Create: `internal/clock/transition.go`
- Test: `internal/clock/transition_test.go`

- [ ] **Step 1: Write failing state-machine tests**

Define table tests against the wished-for API:

```go
next, event, err := clock.Apply(current, clock.Command{ID: "cmd-1", Type: clock.Start}, now)
```

Cover stopwatch Start/Pause/Resume/Lap/Reset, timer Start/Pause/Resume/Reset/Expire, invalid Lap on timer, invalid repeated commands, centisecond/millisecond projection inputs, reset clearing visible laps, and ten-second Undo eligibility.

- [ ] **Step 2: Verify RED**

Run `go test ./internal/clock -run TestApply -v`.

Expected: compilation fails because `clock.Apply` and domain types do not exist.

- [ ] **Step 3: Implement minimal immutable transitions**

Create explicit `ClockType`, `State`, `CommandType`, `EventType`, `Clock`, `Command`, `Event`, `Lap`, and `Projection` types. Implement `Apply`, `ElapsedAt`, `RemainingAt`, and compensating Undo without timers or goroutines in the domain package.

- [ ] **Step 4: Verify GREEN and refactor**

Run `go test ./internal/clock -race -v`.

Expected: all state-machine tests pass with the race detector.

- [ ] **Step 5: Commit**

```bash
git add internal/clock
git commit -m "feat: add authoritative clock state machine"
```

### Task 3: Store contract and concurrency-safe memory adapter

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/memory.go`
- Test: `internal/store/memory_test.go`

- [ ] **Step 1: Write failing contract tests**

Use a reusable store suite that creates an instance, resolves distinct control/view capabilities, applies concurrent commands, deduplicates command IDs, updates labels/order/highlight, replays events after a sequence, archives, deletes, and produces an export snapshot.

- [ ] **Step 2: Verify RED**

Run `go test ./internal/store -run TestMemoryStore -v`.

Expected: compilation fails because `Store` and `NewMemory` do not exist.

- [ ] **Step 3: Implement the store boundary**

Define methods around domain concepts rather than SQL: `CreateInstance`, `ResolveCapability`, `Snapshot`, `ApplyCommand`, `UpdateClock`, `AddClock`, `RemoveClock`, `SetHighlight`, `EventsAfter`, `ArchiveDue`, `PurgeDue`, and `DeleteInstance`. Hash raw capability tokens with SHA-256 and serialize mutations per clock in the adapter.

- [ ] **Step 4: Verify GREEN under concurrency**

Run `go test ./internal/store -race -count=10`.

Expected: all repetitions pass and the race detector reports no races.

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/memory.go internal/store/memory_test.go
git commit -m "feat: add clock store contract and memory adapter"
```

### Task 4: HTTP API, lazy creation, and capability security

**Files:**
- Create: `internal/httpapi/router.go`
- Create: `internal/httpapi/handlers.go`
- Test: `internal/httpapi/router_test.go`

- [ ] **Step 1: Write failing HTTP behavior tests**

Test `POST /api/v1/instances` returning different control/view URLs, `GET /api/v1/control/{token}`, `GET /api/v1/view/{token}`, command submission with `Idempotency-Key`, forbidden mutation via view token, unknown token behavior, limits, conflict responses, no-store/referrer/CSP headers, and root GET creating no instance.

- [ ] **Step 2: Verify RED**

Run `go test ./internal/httpapi -run TestRouter -v`.

Expected: compilation fails because `httpapi.NewRouter` does not exist.

- [ ] **Step 3: Implement minimal handlers and typed errors**

Use `http.ServeMux`, strict JSON decoding, bounded bodies, cryptographic capability generation, consistent JSON errors, request IDs, recovery middleware, and handler timeouts that exclude streaming routes. Return authoritative snapshots with `server_time`.

- [ ] **Step 4: Verify GREEN**

Run `go test ./internal/httpapi -race -v`.

Expected: all API/security tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi
git commit -m "feat: expose secure internal clock API"
```

### Task 5: Realtime SSE delivery

**Files:**
- Create: `internal/realtime/hub.go`
- Create: `internal/realtime/hub_test.go`
- Create: `internal/httpapi/sse.go`
- Modify: `internal/httpapi/router.go`
- Test: `internal/httpapi/sse_test.go`

- [ ] **Step 1: Write failing hub and SSE tests**

Test bounded subscribe/unsubscribe, non-blocking eviction of slow consumers, monotonic SSE IDs, replay from `Last-Event-ID`, snapshot fallback, heartbeat comments, and cancellation on client disconnect.

- [ ] **Step 2: Verify RED**

Run `go test ./internal/realtime ./internal/httpapi -run 'TestHub|TestSSE' -v`.

Expected: compilation fails because Hub and SSE routes do not exist.

- [ ] **Step 3: Implement commit-after-broadcast wiring**

Create a per-instance hub whose publish operation cannot block command acknowledgement. Add `/api/v1/view/{token}/events` and control equivalent, sending a snapshot first and committed updates afterward. Set `text/event-stream`, `Cache-Control: no-cache, no-transform`, and proxy-buffering prevention headers.

- [ ] **Step 4: Verify GREEN**

Run `go test ./internal/realtime ./internal/httpapi -race -v`.

Expected: all realtime and HTTP tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/realtime internal/httpapi
git commit -m "feat: stream committed clock updates with SSE"
```

### Task 6: PostgreSQL persistence and migrations

**Files:**
- Create: `internal/store/migrations/001_init.sql`
- Create: `internal/store/postgres.go`
- Create: `internal/store/postgres_test.go`
- Create: `internal/store/migrate.go`

- [ ] **Step 1: Write failing PostgreSQL contract test**

Run the same store contract suite against a disposable PostgreSQL database when `TEST_DATABASE_URL` is set. Add explicit tests for row-lock serialization, unique command IDs, hashed tokens, transaction rollback, and event/projection atomicity.

- [ ] **Step 2: Verify RED**

Start PostgreSQL with Docker, export `TEST_DATABASE_URL`, and run `go test ./internal/store -run TestPostgresStore -v`.

Expected: compilation fails because `NewPostgres` does not exist.

- [ ] **Step 3: Implement schema and adapter**

Create normalized `instances`, `capabilities`, `clocks`, `clock_events`, and `laps` tables with foreign keys, constraints, and retention indexes. Use `SELECT ... FOR UPDATE`, append event and projection update in one transaction, advisory migration lock, and embedded forward-only migrations.

- [ ] **Step 4: Verify GREEN and parity**

Run `go test ./internal/store -race -count=3` with `TEST_DATABASE_URL` set.

Expected: memory and PostgreSQL adapters pass the shared suite.

- [ ] **Step 5: Commit**

```bash
git add internal/store
git commit -m "feat: persist clock events in PostgreSQL"
```

### Task 7: Retention and export

**Files:**
- Create: `internal/retention/worker.go`
- Test: `internal/retention/worker_test.go`
- Create: `internal/httpapi/export.go`
- Test: `internal/httpapi/export_test.go`

- [ ] **Step 1: Write failing retention/export tests**

Use a fake clock to verify free archive at 24 hours and purge seven days later, premium archive at 30 days and purge one year later, view traffic not extending deadlines, control activity renewing deadlines, archived mutation rejection, archived export/delete allowance, CSV quoting, and versioned JSON completeness.

- [ ] **Step 2: Verify RED**

Run `go test ./internal/retention ./internal/httpapi -run 'TestRetention|TestExport' -v`.

Expected: compilation fails because worker and export handlers do not exist.

- [ ] **Step 3: Implement worker and streaming exporters**

Run bounded archive/purge batches on a cancellable ticker. Generate `application/json` schema version 1 and a ZIP containing `clocks.csv`, `laps.csv`, and `events.csv` for CSV export without persisting temporary files.

- [ ] **Step 4: Verify GREEN**

Run `go test ./internal/retention ./internal/httpapi -race -v`.

Expected: retention and byte-level export tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/retention internal/httpapi
git commit -m "feat: add retention lifecycle and exports"
```

### Task 8: Svelte domain client and local time projection

**Files:**
- Create: `web/src/lib/types.ts`
- Create: `web/src/lib/api.ts`
- Create: `web/src/lib/clock.ts`
- Create: `web/src/lib/session.svelte.ts`
- Test: `web/src/lib/clock.test.ts`
- Test: `web/src/lib/session.test.ts`

- [ ] **Step 1: Write failing frontend unit tests**

Test stopwatch centiseconds, stopped millisecond detail, timer whole-second ceiling and zero clamp, server-anchor projection, SSE snapshot/update/reconnect reduction, optimistic `SENDING` state, conflict replacement, and no offline command queue.

- [ ] **Step 2: Verify RED**

Run `cd web && npm test -- --run src/lib`.

Expected: tests fail because projection and session modules do not exist.

- [ ] **Step 3: Implement minimal typed client**

Add fetch wrappers with request IDs/idempotency keys, browser device ID storage, EventSource lifecycle, immutable snapshot reduction, and pure formatting/projection helpers. Never treat local display interpolation as authoritative state.

- [ ] **Step 4: Verify GREEN**

Run `cd web && npm test -- --run src/lib && npm run check`.

Expected: unit tests and Svelte/TypeScript checks pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib
git commit -m "feat: add synchronized web client state"
```

### Task 9: Race Control and Public Board UI

**Files:**
- Create: `web/src/app.css`
- Create: `web/src/lib/ClockCard.svelte`
- Create: `web/src/lib/ConnectionBadge.svelte`
- Create: `web/src/lib/AddClock.svelte`
- Create: `web/src/lib/SharePanel.svelte`
- Create: `web/src/routes/+layout.svelte`
- Create: `web/src/routes/+page.svelte`
- Create: `web/src/routes/c/[token]/+page.svelte`
- Create: `web/src/routes/v/[token]/+page.svelte`
- Test: `web/src/lib/ClockCard.test.ts`
- Test: `web/src/routes/routes.test.ts`

- [ ] **Step 1: Write failing component/route tests**

Test lazy creation only on mutation, Start/Pause/Resume, stopwatch-only Lap, timer presets, independent Reset, ten-second Undo, label editing, add/remove/reorder/highlight, distinct share links, public read-only rendering, public grid/highlight, expiry audio opt-in, keyboard guards, accessible names, and non-color state text.

- [ ] **Step 2: Verify RED**

Run `cd web && npm test -- --run`.

Expected: tests fail because components and routes do not exist.

- [ ] **Step 3: Implement Race Control brutalist UI**

Use semantic HTML, CSS custom properties, responsive grid/container queries, square 4px borders, high-contrast system/monospace fonts, minimum 48px controls, reduced-motion support, outdoor-readable status, focus-visible outlines, and no decorative dependency. Keep public and control layouts separate while sharing ClockCard display logic.

- [ ] **Step 4: Verify GREEN and accessibility checks**

Run `cd web && npm test -- --run && npm run check`.

Expected: all component tests pass with no TypeScript or accessibility warnings.

- [ ] **Step 5: Commit**

```bash
git add web/src
git commit -m "feat: build brutalist race control interface"
```

### Task 10: PWA, embedded assets, and application wiring

**Files:**
- Create: `web/static/manifest.webmanifest`
- Create: `web/static/sw.js`
- Create: `web/static/icon.svg`
- Create: `web/src/service-worker.ts`
- Create: `cmd/chronograph/main.go`
- Create: `internal/httpapi/static.go`
- Create: `internal/httpapi/dist/.gitkeep`
- Test: `internal/httpapi/static_test.go`

- [ ] **Step 1: Write failing static/PWA tests**

Test SPA route fallback, immutable asset caching, no-store capability pages, manifest headers, service-worker shell-only caching, offline-control warning hook, health/readiness, graceful cancellation, and static embedding.

- [ ] **Step 2: Verify RED**

Run `go test ./cmd/chronograph ./internal/httpapi -run 'TestStatic|TestHealth' -v`.

Expected: compilation fails because application wiring/static host do not exist.

- [ ] **Step 3: Implement wiring and PWA shell**

Load environment configuration, connect/migrate PostgreSQL, start retention worker, serve API/SSE/static files, set server timeouts, handle SIGINT/SIGTERM, and register a cache-first strategy only for hashed assets/app shell. Build web assets into the embed directory as part of `make build`.

- [ ] **Step 4: Verify GREEN and production build**

Run:

```bash
make build
go test ./... -race
```

Expected: frontend builds, assets embed, Go binary builds, and all Go tests pass.

- [ ] **Step 5: Commit**

```bash
git add cmd internal/httpapi web/static web/src/service-worker.ts Makefile
git commit -m "feat: package installable chronograph service"
```

### Task 11: Deployment, end-to-end tests, and documentation

**Files:**
- Create: `Dockerfile`
- Create: `compose.yaml`
- Create: `.dockerignore`
- Create: `.env.example`
- Create: `web/playwright.config.ts`
- Create: `web/e2e/chronograph.spec.ts`
- Create: `README.md`
- Create: `scripts/smoke.sh`

- [ ] **Step 1: Write failing end-to-end tests**

Test browser-to-browser lazy creation, shared Start/Pause/Lap/Reset, simultaneous controller convergence, timer expiry, public-board highlight, refresh/reconnect, export downloads, public mutation denial, and offline command rejection.

- [ ] **Step 2: Verify RED**

Run `cd web && npm run test:e2e` against the not-yet-wired compose service.

Expected: tests fail because the deployable stack is not running/configured.

- [ ] **Step 3: Implement deployment and operator documentation**

Create a non-root multi-stage image, PostgreSQL health-gated Compose stack, persistent volume, secrets-by-environment configuration, resource-aware defaults, and smoke script. Document setup, HTTPS reverse proxy requirements, backup/restore, NTP/clock monitoring, upgrades, retention, limits, exports, security, and the network-latency limitation.

- [ ] **Step 4: Verify GREEN end to end**

Run:

```bash
docker compose up -d --build
./scripts/smoke.sh
cd web && npm run test:e2e
docker compose down
```

Expected: smoke and Playwright suites pass; services stop cleanly while the database volume remains.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile compose.yaml .dockerignore .env.example README.md scripts web/e2e web/playwright.config.ts
git commit -m "chore: add reproducible deployment and e2e coverage"
```

### Task 12: Full verification, security review, and delivery

**Files:**
- Modify: only files required by failures found during verification

- [ ] **Step 1: Run the complete verification matrix**

```bash
make verify
docker compose build
docker compose up -d
./scripts/smoke.sh
cd web && npm run test:e2e
cd .. && docker compose down
```

Expected: all unit, race, frontend, type, build, integration, smoke, and E2E checks exit 0.

- [ ] **Step 2: Review requirements line by line**

Map every acceptance criterion in `outputs/2026-07-14-shared-chronograph-design.md` to an automated test or a documented manual operational check. Add a failing regression test before fixing any discovered gap.

- [ ] **Step 3: Review security-sensitive output**

Confirm raw capability tokens do not appear in database rows, structured logs, analytics, referrers, public API payloads, or Docker health output. Confirm CSP, `noindex`, `Referrer-Policy: no-referrer`, and cache rules using smoke assertions.

- [ ] **Step 4: Commit verification fixes**

```bash
git add -A
git commit -m "test: complete chronograph verification"
```

- [ ] **Step 5: Invoke completion workflow**

Use `verification-before-completion`, then `finishing-a-development-branch`; merge the feature branch into the local main branch because the user pre-approved all recommendations and requested autonomous completion.

