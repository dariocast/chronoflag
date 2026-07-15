<script lang="ts">
  export let state = 'connecting';

  const labels: Record<string, string> = {
    connecting: 'Connecting',
    sending: 'Sending',
    synced: 'Live',
    offline: 'Offline'
  };

  $: label = labels[state] ?? labels.connecting;
</script>

<span class="status-badge" data-state={state} role="status" aria-live="polite">
  <span class="status-dot" aria-hidden="true"></span>
  {label}
</span>
