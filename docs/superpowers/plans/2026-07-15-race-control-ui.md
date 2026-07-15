# Race Control UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver an accessible, mobile-first neo-brutalist race-control interface for Chronograph without changing server synchronization or API behavior.

**Architecture:** Keep `Board.svelte` as the network/state coordinator and `ClockCard.svelte` as the clock presentation boundary. Extract dialog focus lifecycle and connection-state presentation into small reusable Svelte components, then rebuild the stylesheet around semantic tokens and validate the complete flows with component and Playwright tests.

**Tech Stack:** SvelteKit 2, Svelte 5 legacy component syntax, TypeScript, Vitest, Testing Library, Playwright, CSS custom properties, Go static embedding.

---

## File Map

- Create `web/src/lib/Modal.svelte`: native dialog lifecycle, Escape close, initial focus, and focus restoration.
- Create `web/src/lib/Modal.test.ts`: dialog accessibility and keyboard/focus behavior.
- Create `web/src/lib/StatusBadge.svelte`: stable mapping from transport state to visible status.
- Create `web/src/lib/StatusBadge.test.ts`: visible and accessible status mapping.
- Modify `web/src/lib/Board.svelte`: control/public command bars, accessible dialogs, copy feedback, and semantic page shell.
- Modify `web/src/lib/ClockCard.svelte`: clearer timing hierarchy and management semantics.
- Modify `web/src/lib/ClockCard.test.ts`: control/public behavior and accessible labels.
- Modify `web/src/routes/+page.svelte`: poster landing structure while preserving one-click creation.
- Replace `web/src/app.css`: tokenized neo-brutalist system and responsive layouts.
- Modify `web/e2e/chronograph.spec.ts`: dialog, public view, and horizontal-overflow coverage.

### Task 1: Connection Status Badge

**Files:**
- Create: `web/src/lib/StatusBadge.test.ts`
- Create: `web/src/lib/StatusBadge.svelte`

- [ ] **Step 1: Write the failing component test**

```ts
import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import StatusBadge from './StatusBadge.svelte';

describe('StatusBadge', () => {
  it.each([
    ['connecting', 'Connecting'],
    ['sending', 'Sending'],
    ['synced', 'Live'],
    ['offline', 'Offline']
  ])('maps %s to %s', (state, label) => {
    render(StatusBadge, { props: { state } });
    expect(screen.getByRole('status')).toHaveTextContent(label);
  });
});
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/lib/StatusBadge.test.ts`  
Expected: FAIL because `StatusBadge.svelte` does not exist.

- [ ] **Step 3: Implement the minimal badge**

```svelte
<script lang="ts">
  export let state = 'connecting';
  const labels: Record<string, string> = {
    connecting: 'Connecting', sending: 'Sending', synced: 'Live', offline: 'Offline'
  };
  $: label = labels[state] ?? 'Connecting';
</script>

<span class="status-badge" data-state={state} role="status" aria-live="polite">
  <span class="status-dot" aria-hidden="true"></span>{label}
</span>
```

- [ ] **Step 4: Run the test and verify GREEN**

Run: `cd web && npm test -- --run src/lib/StatusBadge.test.ts`  
Expected: 4 cases pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/StatusBadge.svelte web/src/lib/StatusBadge.test.ts
git commit -m "feat: add explicit connection status badge"
```

### Task 2: Accessible Modal Primitive

**Files:**
- Create: `web/src/lib/Modal.test.ts`
- Create: `web/src/lib/Modal.svelte`

- [ ] **Step 1: Write failing dialog tests**

```ts
import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import Modal from './Modal.svelte';

describe('Modal', () => {
  it('exposes a named dialog and closes on Escape', async () => {
    const onclose = vi.fn();
    render(Modal, { props: { open: true, title: 'Share links', onclose } });
    expect(screen.getByRole('dialog', { name: 'Share links' })).toBeInTheDocument();
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('returns focus to the trigger after closing', async () => {
    const trigger = document.createElement('button');
    document.body.append(trigger);
    trigger.focus();
    const view = render(Modal, { props: { open: true, title: 'Add clock' } });
    await view.rerender({ open: false, title: 'Add clock' });
    expect(trigger).toHaveFocus();
    trigger.remove();
  });
});
```

- [ ] **Step 2: Run the tests and verify RED**

Run: `cd web && npm test -- --run src/lib/Modal.test.ts`  
Expected: FAIL because `Modal.svelte` does not exist.

- [ ] **Step 3: Implement semantic dialog lifecycle**

Implement a native `<dialog>` with `aria-labelledby`, a close button using the supplied `onclose`, a `window` Escape listener active only while open, and reactive open/close synchronization. Store `document.activeElement` before opening, focus the close button after opening, call `showModal()` when supported, and restore the stored element when closing. Use an `on:cancel` handler with `preventDefault()` so Escape follows the same callback path.

```svelte
<dialog bind:this={dialog} aria-labelledby={titleId} on:cancel={cancel} on:close={restore}>
  <div class="modal-card">
    <header class="modal-header">
      <div><span class="eyebrow">Chronograph</span><h2 id={titleId}>{title}</h2></div>
      <button bind:this={closeButton} class="icon-button" aria-label={`Close ${title}`} on:click={onclose}>×</button>
    </header>
    <div class="modal-body"><slot /></div>
  </div>
</dialog>
```

- [ ] **Step 4: Run the tests and verify GREEN**

Run: `cd web && npm test -- --run src/lib/Modal.test.ts`  
Expected: both tests pass with no console errors.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/Modal.svelte web/src/lib/Modal.test.ts
git commit -m "feat: add accessible modal primitive"
```

### Task 3: Clock Card Operating Hierarchy

**Files:**
- Modify: `web/src/lib/ClockCard.test.ts`
- Modify: `web/src/lib/ClockCard.svelte`

- [ ] **Step 1: Add failing behavior assertions**

Add tests proving that a controller card has an accessible label derived from its label/type, exposes a visible `Clock tools` group, labels reset as `Reset clock`, and that the public card has no form controls or buttons.

```ts
expect(screen.getByRole('article', { name: 'Untitled stopwatch' })).toBeInTheDocument();
expect(screen.getByRole('group', { name: 'Clock tools' })).toBeInTheDocument();
expect(screen.getByRole('button', { name: 'Reset clock' })).toBeInTheDocument();
expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd web && npm test -- --run src/lib/ClockCard.test.ts`  
Expected: FAIL because the current article and tool group are unnamed and reset is labeled only `Reset`.

- [ ] **Step 3: Rebuild card semantics and hierarchy**

Use an `aria-label` on the article, a state dot plus visible state word, a `time-display` wrapper, a `role="group"` primary action row, and a separate `role="group" aria-label="Clock tools"` row. Preserve all callbacks and existing command strings. Keep the time `aria-live="off"`. In public mode render the label as a heading, or the visible text `Untitled` when absent.

- [ ] **Step 4: Run the focused tests and verify GREEN**

Run: `cd web && npm test -- --run src/lib/ClockCard.test.ts`  
Expected: all ClockCard tests pass.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/ClockCard.svelte web/src/lib/ClockCard.test.ts
git commit -m "feat: clarify clock operating hierarchy"
```

### Task 4: Control Deck Dialogs and Feedback

**Files:**
- Modify: `web/e2e/chronograph.spec.ts`
- Modify: `web/src/lib/Board.svelte`

- [ ] **Step 1: Add failing end-to-end dialog assertions**

Extend the stopwatch flow to assert a dialog named `Share links`, close it with Escape, verify focus returns to Share, open `Add clock`, and verify its dialog name. Add clipboard permission and assert that copying the public link shows `Public link copied`.

```ts
await context.grantPermissions(['clipboard-read', 'clipboard-write']);
await page.getByRole('button', { name: 'Share' }).click();
await expect(page.getByRole('dialog', { name: 'Share links' })).toBeVisible();
await page.getByRole('button', { name: 'Copy public link' }).click();
await expect(page.getByRole('status')).toContainText('Public link copied');
await page.keyboard.press('Escape');
await expect(page.getByRole('button', { name: 'Share' })).toBeFocused();
```

- [ ] **Step 2: Run Playwright and verify RED**

Run: `cd web && npm run test:e2e -- --grep "creates, controls"`  
Expected: FAIL because the current panels are asides, lack dialog names, lack copy feedback, and do not restore focus.

- [ ] **Step 3: Integrate `Modal` and `StatusBadge`**

Restructure `Board.svelte` into `app-shell`, semantic `command-bar`, `board-heading`, and clock grid. Replace both asides with `Modal`, store the active panel as `'share' | 'add' | null`, and expose trigger references for focus restoration through normal DOM focus. Add a `feedback` string in a polite status region. Implement:

```ts
async function copyLink(value: string, kind: 'Control' | 'Public') {
  try {
    await navigator.clipboard.writeText(value);
    feedback = `${kind} link copied`;
  } catch {
    error = 'Could not copy the link. Select and copy it manually.';
  }
}
```

Retain event subscription, 20ms display projection, timer expiry, undo window, API calls, and all existing control/public capability boundaries.

- [ ] **Step 4: Run Playwright and verify GREEN**

Run: `cd web && npm run test:e2e -- --grep "creates, controls"`  
Expected: the complete stopwatch/share/public synchronization flow passes.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/Board.svelte web/e2e/chronograph.spec.ts
git commit -m "feat: redesign control deck dialogs and feedback"
```

### Task 5: Poster Landing and Tokenized Visual System

**Files:**
- Modify: `web/src/routes/+page.svelte`
- Modify: `web/src/app.css`
- Modify: `web/e2e/chronograph.spec.ts`

- [ ] **Step 1: Add failing responsive assertions**

Add a Playwright test covering 390×844, 844×390, 768×1024, and 1440×900 viewports. At each size load the landing and a newly created control board and assert `document.documentElement.scrollWidth <= window.innerWidth`. Also assert that all visible control buttons have bounding-box height at least 44px.

- [ ] **Step 2: Run the responsive test and verify RED**

Run: `cd web && npm run test:e2e -- --grep "fits supported viewports"`  
Expected: FAIL on at least the existing compact tool buttons, which are below 44px.

- [ ] **Step 3: Rebuild landing markup**

Keep `start()` unchanged and wrap the page in a poster composition with a top registration row, framed time, stacked headline, yellow launch button, trust list, and decorative CSS-only marks hidden from assistive technology. Keep the button accessible name `Start now` and expose busy/error states.

- [ ] **Step 4: Replace CSS with the tokenized system**

Define the approved colors, spacing, border, shadow, typography, and motion tokens in `:root`. Organize selectors under foundation, landing, shell, status, clock, lap, modal, feedback, and media-query sections. Use a local/system fallback stack with a `Space Grotesk` web-font import, tabular monospace time values, no gradients or blur, minimum 44px controls, 56px primary controls, 390px-safe time scaling, two-row mobile command bar, multi-column desktop grid, public-board sizing, landscape compaction, hover-capability guards, and reduced-motion overrides.

- [ ] **Step 5: Run the responsive test and verify GREEN**

Run: `cd web && npm run test:e2e -- --grep "fits supported viewports"`  
Expected: all viewport and touch-target assertions pass.

- [ ] **Step 6: Run component tests and type checking**

Run: `cd web && npm test -- --run && npm run check`  
Expected: all tests pass and Svelte reports 0 errors and 0 warnings.

- [ ] **Step 7: Commit**

```bash
git add web/src/routes/+page.svelte web/src/app.css web/e2e/chronograph.spec.ts
git commit -m "feat: apply race control visual system"
```

### Task 6: Integrated Verification and Visual Audit

**Files:**
- Modify only if a failing verification produces a regression test and corresponding fix.

- [ ] **Step 1: Run the full repository gate**

Run: `make verify`  
Expected: Go tests, Vitest, `go vet`, Svelte check, production web build, Go build, and Go race tests all exit 0.

- [ ] **Step 2: Run complete browser flows**

Run: `cd web && npm run test:e2e`  
Expected: stopwatch/public tracking, independent timer, dialogs, and viewport tests all pass.

- [ ] **Step 3: Inspect fresh browser screenshots**

Build and run the current image with Docker Compose, then capture landing, control, share dialog, and public board at 390×844 and 1440×900. Verify no clipping, collision, illegible state, accidental horizontal scrolling, or offscreen primary action.

- [ ] **Step 4: Check repository state and diff quality**

Run: `git diff v0.1.0-mvp --check && git status --short && git log --oneline v0.1.0-mvp..HEAD`  
Expected: no whitespace errors, only intentional files are changed, and commits map to the plan.

- [ ] **Step 5: Commit any verification-driven fix**

For every discovered issue, first add a failing regression test, observe RED, implement the fix, observe GREEN, rerun `make verify` and Playwright, then commit only the related files with a specific `fix:` message. If no issue is found, do not create an empty commit.
