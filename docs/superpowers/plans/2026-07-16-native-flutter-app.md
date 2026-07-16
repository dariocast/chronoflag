# Native Flutter App Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a Flutter operator and viewer client backed by Chronoflag's public capability API.

**Architecture:** Extend the Go domain snapshot with a title and capability rotation, exposing small authenticated control routes. Add a standalone `app/` Flutter package with pure domain/API/storage layers, then compose adaptive Material 3/Cupertino-Liquid-Glass presentation around them. Keep all time and mutation authority in the existing service.

**Tech Stack:** Go 1.24, existing memory/PostgreSQL stores and net/http router; Flutter/Dart, Material 3, Cupertino, `http`, `shared_preferences`, `qr_flutter`, `mobile_scanner`, `share_plus`, `flutter_local_notifications`.

## Global Constraints

- No accounts, analytics, crash-reporting SDKs, command queue, or intermediary backend.
- Android supports API 26+. iOS 26/macOS 26 receive Liquid Glass styling with Cupertino fallback.
- API tokens are capability credentials and must never be logged or persisted outside local session storage.
- All mutable actions carry an idempotency key and only work in control mode while an active snapshot is synchronized.
- User-facing copy is English/Italian through the platform locale.

---

### Task 1: Extend the server domain and capability lifecycle

**Files:**
- Modify: `internal/store/store.go`, `internal/store/memory.go`, `internal/store/postgres.go`
- Modify: `internal/httpapi/router.go`, `internal/httpapi/router_test.go`
- Test: `internal/store/memory_test.go`, `internal/httpapi/router_test.go`

**Interfaces:**
- Produces `InstanceSnapshot.Title string`, `Store.UpdateInstance(context.Context, Capability, InstancePatch, time.Time)`, and `Store.RotateCapability(context.Context, Capability, Scope) (CreatedInstance, error)`.
- Produces `PATCH /api/v1/control/{token}` accepting `{"title":"..."}` and `POST /api/v1/control/{token}/capabilities/{scope}/rotate`.

- [ ] Write failing memory-store tests for a title change, control-token rotation, view-token rotation, and invalidation of the old token.
- [ ] Run `go test ./internal/store -run 'Test.*(Title|Rotate)'` and observe failure because the API does not exist.
- [ ] Add the `Title` field, `InstancePatch`, store interface methods, memory implementation, and PostgreSQL transactional implementation. Validate trimmed titles are at most 120 runes; only a control capability may mutate or rotate.
- [ ] Add HTTP tests asserting title is in snapshots, old capabilities return not-found after rotation, and new `/c/` or `/v/` URLs work.
- [ ] Register the PATCH and rotation routes, decode strict request bodies, publish changed snapshots, and return relative replacement URLs.
- [ ] Run `go test ./internal/store ./internal/httpapi` and commit `feat: add native session metadata APIs`.

### Task 2: Scaffold and test the Flutter core package

**Files:**
- Create: `app/pubspec.yaml`, `app/analysis_options.yaml`, `app/lib/main.dart`
- Create: `app/lib/src/domain/models.dart`, `app/lib/src/domain/clock_projection.dart`
- Create: `app/test/clock_projection_test.dart`, `app/test/capability_uri_test.dart`
- Modify: `.gitignore`, `README.md`, `Makefile`

**Interfaces:**
- Produces immutable `ChronoflagSession`, `InstanceSnapshot`, `Clock`, `Capability`, and `ClockProjection` Dart models.
- Produces `Capability.parse(Uri)` and `ClockProjection.at(Clock, DateTime)`.

- [ ] Write tests that parse `/c/{token}` and `/v/{token}`, reject unrelated URLs, project running stopwatches, and clamp expired timers.
- [ ] Run `cd app && flutter test test/clock_projection_test.dart test/capability_uri_test.dart`; observe unresolved imports/types.
- [ ] Create the Flutter application and implement pure models/projection with JSON conversion matching the Go envelope's nanosecond durations and ISO timestamps.
- [ ] Add dependency declarations, lint configuration, ignored generated directories, and Makefile targets `app-test`, `app-analyze`, and `app-build`.
- [ ] Run the focused tests and commit `feat: scaffold Flutter chronoflag client`.

### Task 3: Implement HTTP, SSE, and local recent-session services

**Files:**
- Create: `app/lib/src/data/chronoflag_api.dart`, `app/lib/src/data/sse_client.dart`, `app/lib/src/data/session_repository.dart`
- Create: `app/test/chronoflag_api_test.dart`, `app/test/session_repository_test.dart`

**Interfaces:**
- Produces `ChronoflagApi` methods `create`, `snapshot`, `command`, `undo`, `addClock`, `updateClock`, `deleteClock`, `export`, `updateTitle`, and `rotateCapability`.
- Produces `SessionRepository.load/save/remove` and `Stream<InstanceSnapshot> events(Capability)`.

- [ ] Write API tests with a local `HttpServer` asserting absolute API URL construction, command idempotency headers, JSON payloads, and error mapping.
- [ ] Run `cd app && flutter test test/chronoflag_api_test.dart`; observe missing client.
- [ ] Implement `ChronoflagApi`, including UUID command IDs and a line-oriented SSE decoder that accepts `snapshot` and `update` events.
- [ ] Write repository tests proving sessions persist role/title/last-used but never duplicate a capability and can be removed locally.
- [ ] Implement shared-preferences persistence with a versioned JSON list, and run all Task 3 tests.
- [ ] Commit `feat: add Flutter API and local session services`.

### Task 4: Build adaptive home, routing, and synchronization state

**Files:**
- Create: `app/lib/src/app.dart`, `app/lib/src/features/home/home_screen.dart`, `app/lib/src/features/board/board_controller.dart`, `app/lib/src/features/board/board_screen.dart`
- Create: `app/test/board_controller_test.dart`, `app/test/home_screen_test.dart`

**Interfaces:**
- Produces `BoardController.open(Capability)`, `BoardController.perform(ClockCommand)`, and `BoardController.connection`.
- Produces `ChronoflagApp` routing for home, control board, and view board.

- [ ] Write controller tests showing an initial snapshot is fetched, SSE updates replace it, and mutation is refused while offline or archived.
- [ ] Run controller tests and observe the missing controller.
- [ ] Implement a `ChangeNotifier` controller with foreground connection lifecycle, explicit connecting/synced/offline/error state, ten-second undo expiry, and local-session upsert.
- [ ] Write widget tests for an empty home, a saved control session, and a saved viewer session.
- [ ] Implement Material 3 home and adaptive Cupertino shell: phone home-to-board navigation, desktop sidebar, locale-aware strings, and deep-link ingestion.
- [ ] Run Task 4 tests and commit `feat: add native session navigation and live sync`.

### Task 5: Implement board controls, focus, and accessibility

**Files:**
- Create: `app/lib/src/features/board/clock_card.dart`, `app/lib/src/features/board/focus_clock_screen.dart`, `app/lib/src/features/board/add_clock_sheet.dart`
- Create: `app/test/clock_card_test.dart`, `app/test/add_clock_sheet_test.dart`

**Interfaces:**
- Consumes `BoardController` and `ClockProjection`.
- Produces interactive control/read-only cards and custom timer duration selection.

- [ ] Write widget tests for control versus view cards, primary action state, lap availability, reset confirmation, custom duration, and semantic labels.
- [ ] Run the tests and observe the missing widgets.
- [ ] Implement responsive list/grid cards, Material/Cupertino adaptive controls, labels, laps, reorder/focus/delete controls, haptic feedback, and confirmation dialogs.
- [ ] Implement focus presentation with portrait/landscape layouts and keyboard/controller shortcuts that do not consume platform system buttons.
- [ ] Run Task 5 tests and commit `feat: add adaptive timing board controls`.

### Task 6: Implement sharing, scanning, export, notifications, and platform integration

**Files:**
- Create: `app/lib/src/features/share/share_sheet.dart`, `app/lib/src/platform/notifications.dart`
- Modify: `app/lib/src/features/home/home_screen.dart`, `app/lib/src/features/board/board_screen.dart`
- Modify: `app/android/app/src/main/AndroidManifest.xml`, `app/ios/Runner/Info.plist`, `app/macos/Runner/Info.plist`
- Create: `app/test/share_sheet_test.dart`, `app/test/notifications_test.dart`

**Interfaces:**
- Produces safe-default view sharing, warned control sharing, QR rendering/scanning, platform share export, and best-effort local timer alerts.

- [ ] Write tests for a view-first share model, control-warning acknowledgement, capability rotation replacing stored URLs, and notifications scheduled only for running timers.
- [ ] Run the tests and observe missing services/widgets.
- [ ] Implement QR/share UI, camera request only at scanner invocation, native export sharing, and notification scheduling/cancellation from board snapshots.
- [ ] Add verified app/universal-link host declarations, Android notification/camera permissions, and Android foreground-service event mode behind an explicit user action.
- [ ] Run Task 6 tests and commit `feat: add native sharing and event alerts`.

### Task 7: End-to-end validation and documentation

**Files:**
- Modify: `README.md`, `docs/superpowers/specs/2026-07-16-flutter-native-app-design.md`
- Test: existing Go/web suites and Flutter test/analyze/build commands

- [ ] Run `go test ./...`, `cd web && npm test -- --run`, `cd app && flutter test`, `cd app && flutter analyze`, and platform builds for Android, iOS, and macOS where the host SDK permits them.
- [ ] Run `make verify` to confirm the existing backend and web application remain compatible.
- [ ] Document local Flutter setup, public API/deep-link hosting files, and any platform build limitation observed on the host.
- [ ] Review the spec and this plan against implementation; remove only completed checklist markers and commit `docs: document native app development`.

## Plan review

All approved product requirements map to Tasks 1–6. The plan contains no placeholder work, preserves the current web API, and keeps platform-only behavior behind small interfaces that are unit-testable without a device.
