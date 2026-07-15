# Race Control UI/UX Redesign

**Date:** 2026-07-15
**Status:** Approved
**Baseline:** `v0.1.0-mvp` (`f312dfb`)

## Objective

Turn the existing functional interface into a distinctive, mobile-first race-control product that can be operated quickly under pressure and read clearly from a distance. Preserve the zero-configuration promise: opening the service creates a server-authoritative stopwatch immediately, while sharing and secondary organization remain available after the clock is running.

The redesign changes presentation and client-side interaction only. Clock synchronization, command semantics, persistence, URLs, retention, and backend APIs remain unchanged.

## Design Direction

The chosen direction is **Race Control with Brutalist Poster accents**:

- Control views prioritize speed, state awareness, large touch targets, and prevention of accidental destructive actions.
- The landing page and public board carry the expressive editorial identity.
- Neo-brutalist construction stays physical and direct: four-pixel borders, hard shadows, flat color, sharp corners, monospace instrumentation, and fast mechanical button presses.
- Organized asymmetry and small graphic patterns add character without competing with the running time.

The alternative “Brutalist Poster everywhere” was rejected because it would add visual noise to operational controls. The alternative “Minimal Broadcast” was rejected because it would erase too much of the product’s personality.

## Visual System

### Tokens

- Canvas: warm cream `#FFFDF5`.
- Ink: near-black `#111111`.
- Primary action/highlight: yellow `#FFD93D`.
- Critical/error/destructive: coral `#FF6B6B`.
- Secondary accent/focused clock: violet `#C4B5FD`.
- Confirmed/live: green `#15803D`, always paired with a text label or icon.
- Focus indicator: blue `#2563EB`, selected for strong contrast against every surface.
- Border: 4px solid ink for primary surfaces; 2px for internal separators.
- Shadow: hard 6px on mobile and 8px on wider screens; no blur, opacity, or gradients.
- Radius: zero except circular status dots.

### Typography

- Interface and headlines use `Space Grotesk`, with a robust system fallback.
- Time values and instrumentation use a tabular monospace stack.
- Headlines are uppercase, compact, and heavy; support text is sentence case for easier reading.
- Time is always the largest element in a clock card and scales with `clamp()` without horizontal overflow.

### Motion

- Buttons move into their shadow on press.
- Cards may lift by two pixels on hover only where hover is available.
- Panels enter with a short, direct translation; no spring or decorative looping animation.
- `prefers-reduced-motion` removes all nonessential movement.

## Information Architecture

### Landing

The first screen contains one decision only: start. It uses a poster-like composition with a live-looking zero time, a concise promise, a large yellow launch button, and three trust statements: no account, no setup, server time. Decorative registration marks and a restrained dot/grid pattern fill dead space without adding content.

Starting retains the current behavior: create an instance, start its initial stopwatch, and navigate to its control URL.

### Control Deck

The sticky command bar contains:

1. compact brand/home link;
2. explicit synchronization state (`Live`, `Sending`, `Offline`, `Connecting`);
3. Share action;
4. Add clock action.

The clock grid follows the current order. A focused clock spans the available width on desktop and gains the violet accent; on mobile it remains a single column but is visually dominant.

Each stopwatch card contains:

1. type and state strip;
2. editable label with a clear placeholder;
3. server-projected time and millisecond detail;
4. one dominant context-aware command (`Start`, `Pause`, `Resume`, or `Reset` after expiry);
5. lap and reset actions;
6. a compact management row for undo, ordering, focus, and deletion;
7. lap history, newest first, with lap number, elapsed value, and split.

Primary timing commands remain visible. Management actions are quieter so the operating path is not buried in controls. Reset and delete are visually separated from the primary action. Delete keeps the current confirmation requirement.

### Public Board

The public view is a broadcast surface, not a disabled copy of the control deck:

- a compact `LIVE BOARD` header and synchronization badge;
- no controls or editable fields;
- a larger time hierarchy and generous viewing distance;
- focused clocks dominate the grid;
- labels and lap history remain readable but secondary;
- multi-clock layouts adapt from one column on phones to an auto-fit grid on wide displays.

## Panels and Feedback

Share and Add Clock become semantic modal dialogs with:

- a solid patterned backdrop rather than blur;
- visible title and close button;
- Escape-to-close behavior;
- focus moved into the dialog on open and restored to the trigger on close;
- background interaction prevented while open;
- minimum 44px touch targets.

The Share dialog separates the secret control link from the public viewer link. Copy actions show a short inline `Copied` confirmation and use explicit labels.

The Add Clock dialog gives stopwatch and timer choices clear hierarchy while retaining the existing timer presets and no-configuration default.

Network and command errors appear as a persistent high-contrast alert with a retry action. Connectivity state never relies on color alone. Short action confirmations use an `aria-live="polite"` status region; the continuously changing time remains excluded from screen-reader announcements.

## Responsive Behavior

- Mobile is the control baseline, including a 390px viewport.
- All operating controls are at least 44px high; primary controls target 56px.
- The command bar may use two rows on narrow screens, with brand/status first and actions second.
- Clock actions stack into a full-width primary row and a balanced secondary row.
- Time typography shrinks before wrapping and never clips.
- From 768px, the grid can show multiple cards; focused cards span all columns.
- Landscape phones use reduced vertical padding and preserve the timing controls above the fold.
- Public boards use viewport-aware minimum card heights but avoid large empty areas when only one clock exists.

## Accessibility

- Semantic `header`, `main`, `article`, `dialog`, `button`, `label`, `ol`, and status/alert regions.
- Every icon-only action has an accessible name; most actions retain visible text.
- Logical tab order follows visual order.
- Thick, offset focus states are visible against all backgrounds.
- Disabled actions retain legible contrast and expose disabled semantics.
- State is expressed with words and shape in addition to color.
- Dialog keyboard behavior is tested.
- Motion reduction and coarse pointer media queries are respected.

## Component Structure

- `Board.svelte` owns snapshot/event state, command dispatch, connectivity feedback, and composition of the command bar, dialogs, and grid.
- `ClockCard.svelte` remains responsible for a single stopwatch/timer presentation and emits operations to the board.
- `Modal.svelte` provides accessible dialog lifecycle, Escape handling, focus management, and backdrop behavior.
- `StatusBadge.svelte` maps internal connection states to stable user-facing labels and accessible status semantics.
- `app.css` becomes a readable tokenized stylesheet organized by foundation, landing, shell, clocks, dialogs, feedback, and responsive rules.

No component library or icon dependency is added. Small graphic marks use CSS or inline SVG to keep the production bundle lean.

## Verification Criteria

1. Landing still creates and starts a stopwatch with one activation.
2. Control mode exposes share, add, primary commands, lap, reset, focus, ordering, undo, and delete where applicable.
3. Public mode exposes no modifying controls.
4. Connection status has clear text for connecting, sending, live, and offline states.
5. Share and add dialogs have accessible names, close with Escape, and return focus to their trigger.
6. Copying either link produces visible and assistive-technology feedback.
7. The layout has no horizontal overflow at 390px, 768px, 1440px, or landscape-phone dimensions.
8. All existing Go, Svelte, and Playwright tests pass; new interaction tests cover the new client behavior.
9. Svelte type checking, production build, Go race tests, and fresh desktop/mobile browser smoke tests pass.

## Out of Scope

- Public API exposure.
- Authentication, accounts, billing, or retention changes.
- Backend command or synchronization changes.
- Custom timer-duration input beyond existing presets.
- Native or embedded clients.
- Localization and theme switching.
