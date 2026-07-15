<script lang="ts">
  import type { Clock } from './types';
  import { displayTime, detailTime } from './clock';

  export let clock: Clock;
  export let control = false;
  export let now = new Date();
  export let highlighted = false;
  export let canUndo = false;
  export let index = 0;
  export let total = 1;
  export let onaction: (type: string) => void = () => {};
  export let onlabel: (label: string) => void = () => {};
  export let onhighlight: () => void = () => {};
  export let onremove: () => void = () => {};
  export let onmove: (target: number) => void = () => {};

  $: primary = clock.state === 'idle'
    ? 'Start'
    : clock.state === 'running'
      ? 'Pause'
      : clock.state === 'paused'
        ? 'Resume'
        : 'Reset';
  $: primaryCommand = primary.toLowerCase();
  $: displayLabel = clock.label?.trim() || 'Untitled';
  $: cardLabel = `${displayLabel} ${clock.type}`;
</script>

<article
  class:hero={highlighted}
  class="clock-card"
  data-state={clock.state}
  aria-label={cardLabel}
>
  <header class="clock-meta">
    <span class="kind">{clock.type === 'timer' ? 'Timer' : 'Stopwatch'}</span>
    <span class="clock-state">
      <span class="state-dot" aria-hidden="true"></span>
      {clock.state}
    </span>
  </header>

  {#if control}
    <label class="clock-label-editor">
      <span class="sr-only">Clock label</span>
      <input
        class="label"
        aria-label="Clock label"
        maxlength="80"
        value={clock.label ?? ''}
        placeholder="Untitled"
        on:change={(event) => onlabel(event.currentTarget.value)}
      />
    </label>
  {:else}
    <h2 class="public-label">{displayLabel}</h2>
  {/if}

  <div class="time-display">
    <div class="time" aria-live="off">{displayTime(clock, now)}</div>
    {#if clock.state !== 'running' && clock.type === 'stopwatch'}
      <div class="detail">Precision · {detailTime(clock.accumulated)}</div>
    {/if}
  </div>

  {#if control}
    <div class="controls" role="group" aria-label="Timing controls">
      <button class="primary" on:click={() => onaction(primaryCommand)}>{primary}</button>
      {#if clock.type === 'stopwatch'}
        <button disabled={clock.state !== 'running'} on:click={() => onaction('lap')}>Lap</button>
      {/if}
      <button class="quiet" aria-label="Reset clock" on:click={() => onaction('reset')}>Reset</button>
    </div>

    <div class="tools" role="group" aria-label="Clock tools">
      {#if canUndo}<button on:click={() => onaction('undo')}>Undo</button>{/if}
      {#if total > 1}
        <button disabled={index === 0} aria-label="Move clock up" on:click={() => onmove(index - 1)}>↑ Up</button>
        <button disabled={index === total - 1} aria-label="Move clock down" on:click={() => onmove(index + 1)}>↓ Down</button>
      {/if}
      <button aria-pressed={highlighted} on:click={onhighlight}>{highlighted ? 'Unfocus' : 'Focus'}</button>
      {#if total > 1}<button class="danger" on:click={onremove}>Delete</button>{/if}
    </div>
  {/if}

  {#if clock.type === 'stopwatch' && clock.laps?.length}
    <section class="lap-section" aria-labelledby={`laps-${clock.id}`}>
      <div class="lap-heading">
        <h3 id={`laps-${clock.id}`}>Lap history</h3>
        <span>{clock.laps.length} total</span>
      </div>
      <ol class="laps">
        {#each [...clock.laps].reverse() as lap}
          <li>
            <b>#{lap.number}</b>
            <span>{detailTime(lap.elapsed)}</span>
            <small>+{detailTime(lap.split)}</small>
          </li>
        {/each}
      </ol>
    </section>
  {/if}
</article>
