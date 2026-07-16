# Chronoflag Native Flutter App Design

## Goal

Deliver a native Flutter client for Android, iOS, and macOS that gives event
operators the same server-authoritative timing capabilities as the Chronoflag
web app. The application is anonymous and capability-link based: no account,
analytics, or third-party telemetry is used.

## Users and access

Operators create and control instances. A control capability grants complete,
equal operator access. View capabilities are unlisted and read-only. Both are
credentials and are never logged or included in telemetry.

The home screen stores opened capabilities only on the local device. It lists
recent sessions with their title, role, lifecycle, and last use. Removing a
session removes only the local capability. A shared control or view URL uses
Android App Links and Apple Universal Links to open the native app when it is
installed and otherwise continues to work in the web app.

Each session has a server-stored title that is shared with every operator and
viewer. Operators can regenerate the control capability and the view
capability independently; regeneration invalidates the preceding capability.

## Primary flows

1. On first launch the user sees an empty recent-sessions home and can create
   a session in one tap, open a copied link, or scan a QR code.
2. Creating a session creates the same ready, stopped stopwatch supplied by
   the existing public Chronoflag API and opens control mode.
3. Control mode presents a responsive board of independent clocks. Operators
   start, pause, resume, reset, record stopwatch laps, rename, reorder,
   focus, add, and delete clocks. A ten-second undo is shown after a command.
4. Adding a timer provides current minute presets and a custom
   hours/minutes/seconds duration. Reset and deletion require confirmation;
   Start, Pause, and Resume are immediate and provide platform feedback.
5. The Share surface defaults to the safe view capability. The operator
   capability sits behind a clearly labelled warning and confirmation. Both
   can be copied, sent through the native share surface, or presented as QR
   codes.
6. View mode renders the live board, focused clock, labels, and lap history
   without control affordances.
7. The session menu exposes export through the existing JSON/CSV endpoint,
   lifecycle/retention information, capability regeneration, and local
   removal.

## Synchronization and reliability

Chronoflag's server remains authoritative. The client projects running time
from server anchors and maintains a live SSE connection while foregrounded.
It includes an idempotency key and device identifier on each command. If
disconnected, the app displays the connection state, keeps rendering the last
known clock projection, and disables commands; it does not queue or simulate
offline operations.

The app schedules best-effort local expiration notifications. Android offers
an explicit `Event in progress` mode with a foreground-service notification
while the operator keeps the mode active. iOS and macOS do not promise live
background synchronization or cancellation of notifications after remote
changes.

## Platform design

Android uses Material Design 3. iOS and macOS use a Liquid Glass treatment on
iOS 26 and macOS 26, falling back to conventional Cupertino surfaces on older
supported Apple releases. Every platform follows the system light/dark theme;
there is no in-app theme override and no brutalist UI.

Phones use a compact vertical board, tablet and desktop use adaptive grids,
and focus mode is a large, high-contrast adaptive presentation in either
portrait or landscape. macOS supports independent windows for individual
session/mode combinations. The desktop navigation uses a sidebar while phone
navigation is a simple home-to-board flow without a permanent tab bar.

The app supports Italian and English from the system locale, Dynamic Type,
platform contrast and reduced-motion preferences, and screen-reader semantics.
Time displays may compress only enough to avoid clipping. Physical external
keyboards and controllers support Space for primary action, L for lap,
Command-Z for undo where available, and arrow selection. The application does
not remap hardware volume, power, silent, camera, or similar system keys.

Camera permission is requested only when scanning a QR code. Notification
permission is requested only when an operator starts the first countdown.

## Server changes

The Chronoflag service remains the only backend. Public API additions provide
session-title updates, independent control/view capability regeneration, and
native-client-friendly absolute API and export handling where the existing
web-relative API cannot be used. Existing capability endpoints remain
compatible with the web app.

## Scope boundaries

The release contains no account system, billing interface, analytics,
crash-reporting SDK, push-notification infrastructure, or background command
queue. A Free/Premium badge may be displayed solely to communicate the
existing server retention policy; native clients do not sell or unlock tiers.

## Verification

Unit tests cover clock projection, capability parsing, API serialization,
local-session storage, command and regeneration behavior, and Flutter UI
state. Integration tests cover the new Go HTTP routes and preserve web API
compatibility. Platform analyzer/build checks run for Flutter plus the
repository's existing Go and web verification suite.
