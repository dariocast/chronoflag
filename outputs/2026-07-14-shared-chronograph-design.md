# Shared Chronograph — Product and Technical Design

**Status:** approved design  
**Date:** 2026-07-14  
**Scope:** web/PWA MVP for shared, server-authoritative stopwatches and countdown timers

## 1. Product intent

Build the fastest possible way to create and share one or more synchronized stopwatches or timers. A user opens the site and can start immediately: no account, setup form, naming step, or installation is required.

The product is deliberately generic. Amateur race officials are an important scenario, but competitors, bibs, lanes, and sporting rules are not part of the core domain. The same primitives must also serve workshops, classrooms, events, broadcasts, embedded integrations, and future native applications.

The system is not certified sports-timing equipment. Server receipt time is authoritative, so network latency is part of manual Start, Stop, and Lap measurements.

## 2. MVP goals

- Open the site and operate a ready stopwatch immediately.
- Create the server instance lazily on the first mutating action.
- Provide separate, unguessable control and public-view links.
- Synchronize multiple controllers and viewers in real time.
- Support multiple independent stopwatches and countdown timers per instance.
- Preserve an auditable event history while keeping the visible interface simple.
- Work as an installable PWA, while requiring connectivity for control commands.
- Export useful data before retention expiry.
- Run as a simple, containerized service on a single European Hetzner server.

## 3. Explicit non-goals

- No public API, SDK, webhook, API-key management, or embedded-device support in the MVP.
- No offline command capture or retroactive timestamps.
- No participant, lane, race, ranking, or checkpoint model.
- No certified timing or promise of network-independent precision.
- No push notifications.
- No multi-region active-active deployment.
- No password-protected public links or granular controller roles.

An internal versioned HTTP API remains a hard architectural boundary so a public API and native clients can be added later without rewriting the domain.

## 4. Primary flows

### 4.1 Creation

1. The root page renders a local, zeroed stopwatch immediately.
2. The first Start, add-clock, or other mutation creates an instance on the server.
3. The browser URL is replaced with the secret control URL without a page reload.
4. The control view exposes `COPY CONTROL LINK` and `COPY PUBLIC LINK`.
5. Crawlers, link previews, and abandoned root-page visits create no server records.

### 4.2 Shared control

- Anyone holding the control link has equal authority in the MVP.
- Valid commands are serialized per clock; the first valid command committed wins.
- Duplicate requests are idempotent.
- Every accepted transition receives a server timestamp and monotonically increasing sequence number.
- Every connected controller receives the resulting state.
- An anonymous device identifier records which controller submitted an action; it is not an identity or account.

### 4.3 Public tracking

- Anyone holding the public link can view without authentication.
- The page is unlisted, non-enumerable, and excluded from search indexing.
- All clocks appear in an adaptive grid.
- One clock can optionally be highlighted by a controller while the others remain visible.
- The controller-defined ordering is shared with all viewers.

## 5. Clock behavior

### 5.1 Stopwatch

Commands: Start, Pause, Resume, Lap, Reset.

- Live display uses centiseconds.
- A stopped value and exports retain millisecond precision.
- Lap is valid only while running.
- Each lap stores cumulative elapsed time and delta from the preceding lap.
- A lap receives an automatic number and may receive a plain-text label later.
- Reset returns that stopwatch to zero and clears its visible lap list.
- Reset preserves the clock object, its label, order, and display settings.
- Pre-reset events and laps remain in the technical history and exports until data expiry.

### 5.2 Countdown timer

Commands: Start, Pause, Resume, Reset.

- The user creates a timer by choosing a quick duration preset or entering a custom duration.
- Live display uses whole seconds.
- Lap is not available.
- At zero, the timer stops and the server records one `expired` transition.
- Overtime is excluded from the MVP.
- Reset restores the timer's configured starting duration.

### 5.3 Multiple clocks

- Each instance starts conceptually with one stopwatch.
- Every clock has independent state and controls.
- Adding another clock never changes existing clocks.
- The MVP has no global Start, Pause, or Reset.
- Internal command contracts should not prevent future atomic group commands.

## 6. Undo and deletion

- Undo is available for ten seconds after the latest operation on an individual clock.
- It can compensate for Start, Pause/Resume, Lap, and Reset.
- Any controller may invoke it.
- A subsequent accepted operation on the same clock invalidates the prior undo opportunity.
- Undo appends a compensating event; it never rewrites or removes history.
- Removing an entire clock is a distinct action requiring explicit confirmation.

## 7. Labels and metadata

- Instance, clock, and stopwatch-lap labels are optional and editable after creation.
- Labels are short plain text; HTML and active content are never accepted.
- The core schema reserves versioned JSON metadata for future integrations, but the MVP UI does not expose arbitrary metadata editing.
- Race-specific concepts must later be built through metadata or higher-level resources, not by changing base clock semantics.

## 8. Visual and interaction design

### 8.1 Direction: Race Control brutalism

- Rigid grid, thick borders, square corners, high contrast, and oversized monospaced numerals.
- Minimal animation, used only to communicate state changes and acknowledgement.
- Red is reserved for errors and disconnection, green for server confirmation, amber for pending state.
- Each clock may use one accent color, but status meaning always takes precedence.
- State is communicated with text and shape as well as color.
- Controls are large enough for outdoor, one-handed, and high-pressure use.

### 8.2 Control Deck

Each clock card contains:

- dominant time;
- explicit `RUNNING`, `PAUSED`, `EXPIRED`, `PENDING`, or `OFFLINE` state;
- large primary Start/Pause/Resume action;
- Lap only for stopwatches;
- visually separated Reset;
- transient server acknowledgement;
- ten-second Undo affordance when eligible;
- label editing, ordering, highlight, and removal controls.

A persistent `+` adds a stopwatch or timer. Stopwatch is the default. Timer creation offers `1`, `3`, `5`, `10`, `15`, `30`, and `60` minute presets plus a duration keypad.

Desktop shortcuts may include Space for the primary action, `L` for Lap, `R` for Reset, and number keys for clock selection. Shortcuts are disabled during text entry and destructive shortcuts require deliberate confirmation.

### 8.3 Public Board

- With one clock, the time occupies nearly the entire viewport.
- With multiple clocks, the optional highlighted clock occupies approximately two-thirds of the useful area and the remainder form compact cards.
- No administrative navigation or interactive controls appear.
- The view remains readable at distance and adapts to portrait, landscape, and full-screen displays.
- Public audio is off by default and can be enabled locally.

### 8.4 Alerts

- After the first user interaction, controller devices use sound and vibration for timer expiry when supported.
- Public viewers are silent by default and may opt in locally.
- The server owns the single `expired` state transition; devices own presentation of the alert.

## 9. Synchronization model

The server stores transitions, not ticking counters. Each clock projection contains its type, lifecycle state, accumulated duration, last transition time, timer duration if applicable, and current version.

For a command, the server:

1. authenticates the control capability;
2. validates and deduplicates the command ID;
3. enters the per-clock serialization boundary;
4. checks the command against current state;
5. assigns authoritative time and sequence;
6. appends the event and updates the current projection in one database transaction;
7. commits before acknowledging success;
8. broadcasts the committed state to subscribers.

Clients receive a base elapsed/remaining value and server time anchor, then animate locally. They do not receive periodic timer ticks. A lightweight clock-offset estimate improves display smoothness but never changes authoritative event times.

## 10. Conflict and connection behavior

- The first valid serialized command wins.
- A command invalidated by concurrent state receives a conflict response containing the latest state.
- Retried commands reuse their command ID and cannot apply twice.
- The UI displays `SENDING` until the server acknowledges.
- A delayed request produces a visible connection warning.
- A request the server did not receive is not recorded.
- Commands are never queued offline or backdated in the MVP.
- On reconnect, the client requests events after its last SSE event ID.
- If replay is unavailable, the client replaces local state with a fresh snapshot.
- Control actions take priority over fan-out work for public viewers.

## 11. Architecture

### 11.1 Stack

- **Backend:** Go 1.26, using `net/http` and a small routing/data-access surface.
- **Frontend:** SvelteKit with TypeScript, compiled to static assets.
- **Database:** PostgreSQL 18 or the stable version selected at implementation time.
- **Real time:** Server-Sent Events over HTTP/2 for committed state updates.
- **Commands:** versioned internal JSON HTTP endpoints.
- **Packaging:** static frontend embedded into the Go binary; OCI/Docker container.
- **Edge:** TLS reverse proxy with HTTP/2, compression for static assets, and correct no-buffering configuration for SSE.

SSE matches the asymmetric traffic pattern: clients send occasional commands and receive state broadcasts. Native EventSource reconnection and event IDs simplify recovery. WebSockets are unnecessary for the MVP.

### 11.2 Backend components

- **Capability gateway:** resolves control and view tokens without exposing internal IDs.
- **Instance service:** creates instances lazily, manages ordering, highlighting, retention, and tier.
- **Clock domain:** validates state transitions and computes projections.
- **Event store:** atomically appends events and updates projections.
- **Realtime hub:** fans committed changes to local SSE subscribers.
- **Export service:** generates versioned JSON and spreadsheet-friendly CSV.
- **Retention worker:** archives and purges expired data in bounded batches.
- **Static/PWA host:** serves the application shell, manifest, icons, and service worker.

These components begin in one Go process with explicit interfaces. No message broker or cache is required for the single-process MVP.

### 11.3 Data model

Core records:

- `instances`: internal ID, tier, lifecycle, created/last-control/archive/purge times, highlighted clock, version;
- `capabilities`: hashed control/view token, instance ID, scope, revoked time;
- `clocks`: instance ID, type, label, order, lifecycle state, duration fields, version;
- `clock_events`: instance/clock ID, sequence, command ID, event type, server time, anonymous device ID, payload;
- `laps`: clock ID, event ID, lap number, cumulative and split duration, optional label;
- `accounts` and `subscriptions`: introduced only for premium claiming and lifecycle management;
- `exports`: optional short-lived generated artifacts or streaming export audit.

Raw capability tokens are never stored; only secure hashes are persisted.

## 12. Access, security, and abuse controls

- Control and public URLs use distinct, cryptographically random tokens with at least 128 bits of entropy.
- A public token cannot be transformed into a control token or internal ID.
- Capability tokens are excluded from logs, analytics payloads, error reports, and referrer leakage.
- Pages set `noindex`, a strict referrer policy, a restrictive Content Security Policy, and appropriate cache headers.
- Mutations accept only the control capability and require same-origin protections appropriate to token-bearing URLs.
- Label length, clock count, command rate, export size, and SSE connections are bounded.
- MVP target per instance: 10 controllers, 1,000 concurrent viewers, and 100 clocks.
- Premium account owners will later be able to rotate control links; anonymous free instances have no recovery guarantee if the control link is lost.

## 13. Retention and premium

Retention is renewed only by an accepted control action. Viewer connections do not extend it.

### Free

- Editable until 24 hours after the latest accepted control action.
- Then archived read-only for seven additional days.
- Purged after the read-only window.

### Premium

- The user starts anonymously like everyone else.
- Upgrade and account claiming occur only after the instance already exists.
- Editable until 30 days after the latest accepted control action.
- Then archived read-only for up to one additional year.
- Account management permits recovery and later lifecycle features without affecting link-based controllers.

Before archival, the Control Deck shows a clear expiry notice and immediate export action. During read-only retention, the control capability may still export or delete but cannot mutate clocks.

## 14. Export

- **CSV:** separate logical tables or files for clocks, laps, and event history, with ISO timestamps and millisecond durations.
- **Versioned JSON:** complete portable snapshot including instance, clocks, current projections, events, laps, labels, and schema version.
- Export is initiated from the control view.
- Public viewers cannot export in the MVP.
- Generation must stream where possible and enforce size limits.

## 15. PWA behavior

- Installable manifest, dedicated icons, theme colors, and standalone display.
- The service worker caches only the application shell and immutable static assets.
- Opening the shell offline clearly shows that control is unavailable.
- Instance state is not treated as authoritative offline.
- No background command queue or sync is registered.

## 16. Operations and observability

- Structured logs exclude URL tokens and label contents by default.
- Metrics cover command latency, transaction conflicts, SSE connections, reconnects, fan-out delay, rejected commands, instance counts, and retention jobs.
- Health endpoints distinguish process liveness from database readiness.
- Host time must be continuously synchronized; clock-offset monitoring raises an alert when drift exceeds the operational threshold.
- PostgreSQL backups and restore drills protect data inside its promised retention period.
- Graceful shutdown stops new mutations, completes in-flight transactions, and lets SSE clients reconnect.

## 17. Testing strategy

### Domain tests

- Exhaustive state-transition tables for stopwatch and timer.
- Reset, expiry, lap numbering, undo compensation, and invalid-command cases.
- Deterministic fake clock for every time calculation.

### Persistence and concurrency tests

- Event append and projection update are atomic.
- Concurrent Start/Stop/Lap requests produce one valid sequence.
- Duplicate command IDs are idempotent.
- Retention boundaries and premium/free transitions use controlled time.

### Realtime tests

- Snapshot plus SSE replay converges to the same state.
- Reconnection with and without a valid last-event ID.
- Slow viewers cannot delay command acknowledgement or healthy viewers.

### End-to-end tests

- Root page to lazy creation, URL replacement, and link sharing.
- Multiple controller race conditions.
- Public-board ordering and highlight updates.
- Offline, timeout, retry, expiry alert, Reset, and Undo behavior.
- CSV and JSON exports reconstruct the expected history.
- PWA installation and cached-shell offline warning.

### Quality targets

- Test with 10 controllers and 1,000 viewers on one instance.
- Normal committed updates should be perceived by viewers within roughly 250 ms under the target load.
- Accessibility checks cover keyboard operation, focus, reduced motion, contrast, non-color status cues, and screen-reader labels.

## 18. Future-compatible extensions

After the MVP proves the shared-clock core:

- public API with instance-scoped capabilities and premium account keys;
- restricted device tokens such as Lap-only or Stop-only;
- native applications using the same command and event contracts;
- embedded timing gates and automatic systems;
- atomic grouped clock commands;
- competitors, checkpoints, rankings, and race-oriented projections;
- password-protected boards, granular roles, token rotation, and audit identities;
- webhooks and long-term external archives;
- regional instance placement if geographic expansion requires it.

None of these extensions should alter the MVP's base event semantics.

## 19. Acceptance criteria

The MVP is successful when:

1. A new user can open the site and start a shared stopwatch without configuration or authentication.
2. Separate control and public links synchronize the same committed server state.
3. Concurrent control commands resolve deterministically and never apply twice.
4. Multiple independent stopwatches and timers behave according to their defined state machines.
5. Reconnecting clients converge without manual refresh.
6. Reset, Lap, Undo, timer expiry, ordering, highlighting, labels, and exports match this design.
7. The PWA remains honest about connectivity and never invents offline timings.
8. Free and premium retention policies execute from the latest accepted control action.
9. The target instance load remains usable and command processing retains priority.
10. No capability token leaks through storage, logs, analytics, referrers, or public URLs.

