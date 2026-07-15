<script lang="ts">
  import { onDestroy, tick } from 'svelte';

  export let open = false;
  export let title: string;
  export let eyebrow = 'Chronoflag';
  export let onclose: () => void = () => {};

  let dialog: HTMLDialogElement;
  let closeButton: HTMLButtonElement;
  let previousFocus: HTMLElement | null = null;
  $: titleId = `modal-${title.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`;

  async function showDialog() {
    previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    if (typeof dialog.showModal === 'function') dialog.showModal();
    else dialog.setAttribute('open', '');
    await tick();
    if (open && dialog.open) closeButton?.focus();
  }

  function hideDialog() {
    if (typeof dialog.close === 'function') dialog.close();
    else dialog.removeAttribute('open');
    restoreFocus();
  }

  function restoreFocus() {
    previousFocus?.focus();
    previousFocus = null;
  }

  function handleKeydown(event: KeyboardEvent) {
    if (!open || event.key !== 'Escape') return;
    event.preventDefault();
    onclose();
  }

  function handleCancel(event: Event) {
    event.preventDefault();
  }

  function handleBackdrop(event: MouseEvent) {
    if (event.target === dialog) onclose();
  }

  $: if (dialog) {
    if (open && !dialog.open) void showDialog();
    if (!open && dialog.open) hideDialog();
  }

  onDestroy(() => {
    if (dialog?.open) hideDialog();
  });
</script>

<svelte:window on:keydown={handleKeydown} />

<dialog
  bind:this={dialog}
  aria-labelledby={titleId}
  on:cancel={handleCancel}
  on:click={handleBackdrop}
  on:close={restoreFocus}
>
  <div class="modal-card">
    <header class="modal-header">
      <div>
        <span class="eyebrow">{eyebrow}</span>
        <h2 id={titleId}>{title}</h2>
      </div>
      <button
        bind:this={closeButton}
        class="icon-button"
        aria-label={`Close ${title}`}
        on:click={onclose}
      >×</button>
    </header>
    <div class="modal-body"><slot /></div>
  </div>
</dialog>
