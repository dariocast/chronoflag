<script lang="ts">
  import { goto } from '$app/navigation';
  import { createInstance, getSnapshot, command } from '$lib/api';

  let busy = false;
  let error = '';

  async function start() {
    busy = true;
    error = '';
    try {
      const made = await createInstance();
      const token = made.control_url.split('/').pop()!;
      const snapshot = await getSnapshot('control', token);
      await command(token, snapshot.clocks[0].id, 'start');
      await goto(made.control_url);
    } catch (reason) {
      error = reason instanceof Error ? reason.message : 'Could not start a clock.';
      busy = false;
    }
  }
</script>

<svelte:head>
  <title>Chronograph — Shared time, instantly</title>
  <meta name="description" content="A server-synchronized stopwatch and timer you can share instantly." />
</svelte:head>

<main class="landing">
  <header class="landing-top">
    <p class="eyebrow"><span class="registration-dot" aria-hidden="true"></span> Cloud-synchronized time</p>
    <div class="landing-mark" aria-label="Chronograph">C/</div>
  </header>

  <div class="landing-layout">
    <section class="landing-display" aria-label="Instant shared timing">
      <div class="display-strip"><span>Server clock</span><span>Ready / 001</span></div>
      <div class="preview">00:00.00</div>
      <div class="display-strip bottom"><span>Precision first</span><span>EU cloud</span></div>
    </section>

    <section class="landing-copy">
      <div class="poster-index" aria-hidden="true">№ 01</div>
      <h1><span>Open.</span><span>Start.</span><span>Share.</span></h1>
      <p class="landing-lede">One tap creates a live stopwatch. Your operators control it; everyone else follows the same server time.</p>
      <button class="launch" disabled={busy} on:click={start}>
        <span>{busy ? 'Connecting…' : 'Start now'}</span>
        <b aria-hidden="true">→</b>
      </button>
      {#if error}<p class="landing-error" role="alert">{error}</p>{/if}
    </section>
  </div>

  <footer class="landing-footer">
    <ul class="promise" aria-label="Service benefits">
      <li>No account</li>
      <li>No setup</li>
      <li>Server time</li>
    </ul>
    <div class="landing-stamp" aria-hidden="true"><span>Live</span><b>SYNC</b></div>
  </footer>

  <div class="poster-dots" aria-hidden="true"></div>
  <div class="poster-cross cross-one" aria-hidden="true">＋</div>
  <div class="poster-cross cross-two" aria-hidden="true">＋</div>
</main>
