<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { createInstance } from '$lib/api';

  let error = '';

  async function prepare() {
    error = '';
    try {
      const made = await createInstance();
      await goto(made.control_url, { replaceState: true });
    } catch (reason) {
      error = reason instanceof Error ? reason.message : 'Could not prepare a clock.';
    }
  }

  onMount(() => {
    void prepare();
  });
</script>

<svelte:head>
  <title>Preparing your clock — Chronoflag</title>
  <meta name="description" content="Preparing a shared server-synchronized stopwatch." />
</svelte:head>

<main class="loading" aria-busy={!error}>
  {#if error}
    <span role="alert">{error}</span>
    <button on:click={prepare}>Retry</button>
  {:else}
    <span>Preparing your clock</span>
    <b aria-hidden="true">00:00.00</b>
  {/if}
</main>
