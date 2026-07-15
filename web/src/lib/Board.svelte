<script lang="ts">
  import { onMount } from 'svelte';
  import ClockCard from './ClockCard.svelte';
  import Modal from './Modal.svelte';
  import StatusBadge from './StatusBadge.svelte';
  import type { Snapshot, ClockType } from './types';
  import { elapsedAt } from './clock';
  import * as api from './api';

  export let token: string;
  export let mode: 'control' | 'view';

  let snapshot: Snapshot | null = null;
  let status = 'connecting';
  let now = new Date();
  let error = '';
  let feedback = '';
  let activePanel: 'share' | 'add' | null = null;
  let undoUntil: Record<string, number> = {};
  const expiring = new Set<string>();

  $: control = mode === 'control';
  $: publicURL = snapshot && typeof location !== 'undefined'
    ? `${location.origin}/v/${snapshot.id}`
    : '';

  onMount(() => {
    api.getSnapshot(mode, token).then((value) => snapshot = value).catch(reportError);
    const stop = api.events(
      mode,
      token,
      (value) => {
        const previous = snapshot;
        snapshot = value;
        if (value.clocks.some((clock) =>
          clock.state === 'expired' &&
          previous?.clocks.find((other) => other.id === clock.id)?.state !== 'expired'
        )) navigator.vibrate?.(250);
      },
      (value) => status = value
    );
    const tick = setInterval(() => {
      now = new Date();
      if (!control || !snapshot) return;
      for (const clock of snapshot.clocks) {
        if (
          clock.type === 'timer' &&
          clock.state === 'running' &&
          elapsedAt(clock, now) >= clock.duration &&
          !expiring.has(clock.id)
        ) {
          expiring.add(clock.id);
          void act(clock.id, 'expire').finally(() => expiring.delete(clock.id));
        }
      }
    }, 20);
    return () => {
      stop();
      clearInterval(tick);
    };
  });

  function reportError(reason: unknown) {
    error = reason instanceof Error ? reason.message : 'Something went wrong.';
    status = 'offline';
  }

  function commandFeedback(type: string) {
    const messages: Record<string, string> = {
      start: 'Clock started',
      pause: 'Clock paused',
      resume: 'Clock resumed',
      lap: 'Lap recorded',
      reset: 'Clock reset',
      undo: 'Last action undone',
      expire: 'Timer expired'
    };
    feedback = messages[type] ?? '';
  }

  async function act(id: string, type: string) {
    try {
      error = '';
      status = 'sending';
      snapshot = type === 'undo'
        ? await api.undo(token, id)
        : await api.command(token, id, type);
      if (type === 'undo') delete undoUntil[id];
      else undoUntil = { ...undoUntil, [id]: Date.now() + 10_000 };
      commandFeedback(type);
      status = 'synced';
    } catch (reason) {
      reportError(reason);
    }
  }

  async function add(type: ClockType, durationMS = 0) {
    try {
      error = '';
      snapshot = await api.addClock(token, type, durationMS);
      activePanel = null;
      feedback = type === 'stopwatch' ? 'Stopwatch added' : 'Timer added';
    } catch (reason) {
      reportError(reason);
    }
  }

  async function patch(id: string, body: object) {
    try {
      error = '';
      snapshot = await api.patchClock(token, id, body);
    } catch (reason) {
      reportError(reason);
    }
  }

  async function remove(id: string) {
    if (!confirm('Delete this clock? This cannot be undone.')) return;
    try {
      error = '';
      snapshot = await api.removeClock(token, id);
      feedback = 'Clock deleted';
    } catch (reason) {
      reportError(reason);
    }
  }

  async function copyLink(value: string, kind: 'Control' | 'Public') {
    try {
      await navigator.clipboard.writeText(value);
      feedback = `${kind} link copied`;
    } catch {
      error = 'Could not copy the link. Select and copy it manually.';
    }
  }
</script>

<svelte:head><title>{control ? 'Race Control' : 'Live Board'} — Chronoflag</title></svelte:head>

<div class:public-shell={!control} class="app-shell">
  <header class="command-bar">
    <a href="/" class="brand" aria-label="Chronoflag home"><span>C/</span> Chronoflag</a>
    {#if !control}<span class="view-badge">Live board</span>{/if}
    <StatusBadge state={status} />
    {#if control}
      <nav class="command-actions" aria-label="Board actions">
        <button class="secondary" on:click={() => activePanel = 'share'}>Share</button>
        <button class="add" on:click={() => activePanel = 'add'}><span aria-hidden="true">＋</span> Add clock</button>
      </nav>
    {/if}
  </header>

  <div class="sr-only" role="status" aria-live="polite">{feedback}</div>

  {#if error}
    <div class="error-banner" role="alert">
      <div><strong>Connection problem</strong><span>{error}</span></div>
      <button on:click={() => location.reload()}>Retry</button>
    </div>
  {/if}

  {#if snapshot}
    <section class="board-intro" aria-labelledby="board-title">
      <div>
        <p class="eyebrow">{control ? 'Server-synced command centre' : 'Server-synced broadcast'}</p>
        <h1 id="board-title">{control ? 'Control deck' : 'Live board'}</h1>
      </div>
      <p class="board-count">{snapshot.clocks.length} {snapshot.clocks.length === 1 ? 'clock' : 'clocks'}</p>
    </section>

    <main class:public-board={!control} class="board">
      {#each snapshot.clocks as clock, index (clock.id)}
        <ClockCard
          {clock}
          {control}
          {now}
          {index}
          total={snapshot.clocks.length}
          canUndo={(undoUntil[clock.id] ?? 0) > now.getTime()}
          highlighted={snapshot.highlighted_clock_id === clock.id}
          onaction={(type) => act(clock.id, type)}
          onlabel={(label) => patch(clock.id, { label })}
          onmove={(order) => patch(clock.id, { order })}
          onhighlight={() => patch(clock.id, { highlighted: snapshot?.highlighted_clock_id !== clock.id })}
          onremove={() => remove(clock.id)}
        />
      {/each}
    </main>
  {:else}
    <main class="loading" aria-busy="true"><span>Connecting</span><b>00:00.00</b></main>
  {/if}
</div>

<Modal open={activePanel === 'share'} title="Share links" onclose={() => activePanel = null}>
  {#if snapshot}
    <p class="modal-lede">Send the public link to spectators. Keep the control link between operators.</p>
    <div class="link-block secret-link">
      <div class="link-heading"><strong>Control link</strong><span>Secret · can operate clocks</span></div>
      <label>
        <span class="sr-only">Control operator link</span>
        <input aria-label="Control operator link" readonly value={location.href} />
      </label>
      <button on:click={() => copyLink(location.href, 'Control')}>Copy control link</button>
    </div>
    <div class="link-block public-link">
      <div class="link-heading"><strong>Public link</strong><span>Safe to share · view only</span></div>
      <label>
        <span class="sr-only">Public viewer link</span>
        <input aria-label="Public viewer link" readonly value={publicURL} />
      </label>
      <button class="primary" on:click={() => copyLink(publicURL, 'Public')}>Copy public link</button>
    </div>
    {#if feedback.includes('link copied')}<p class="inline-feedback" role="status">{feedback}</p>{/if}
  {/if}
</Modal>

<Modal open={activePanel === 'add'} title="Add clock" eyebrow="Independent timing" onclose={() => activePanel = null}>
  <p class="modal-lede">Every clock runs independently. Add one now and name it whenever you are ready.</p>
  <button class="clock-choice" on:click={() => add('stopwatch')}>
    <span class="choice-icon" aria-hidden="true">＋</span>
    <span><strong>New stopwatch</strong><small>Count up with lap recording</small></span>
  </button>
  <div class="timer-choices">
    <div class="link-heading"><strong>Countdown timer</strong><span>Stops automatically at zero</span></div>
    <div class="presets">
      {#each [1, 3, 5, 10, 15, 30, 60] as minutes}
        <button on:click={() => add('timer', minutes * 60_000)} aria-label={`${minutes} ${minutes === 1 ? 'minute' : 'minutes'}`}>
          <strong>{minutes}</strong><span>min</span>
        </button>
      {/each}
    </div>
  </div>
</Modal>
