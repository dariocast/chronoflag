import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
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
    trigger.textContent = 'Open';
    document.body.append(trigger);
    trigger.focus();
    const view = render(Modal, { props: { open: true, title: 'Add clock' } });

    await view.rerender({ open: false, title: 'Add clock' });

    await waitFor(() => expect(trigger).toHaveFocus());
    trigger.remove();
  });
});
