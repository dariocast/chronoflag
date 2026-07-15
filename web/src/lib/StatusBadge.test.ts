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
