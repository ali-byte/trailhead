// Dispatch's own regression coverage (NOT a locked test file) — added per
// code-review: the Add-bar-vs-initial-load race fix isn't exercised by
// any locked test file (App.test.tsx never submits the Add bar while
// GET /api/board is still in flight), so it's locked in here instead.
//
// The race: the Add bar was always interactable, even while the initial
// GET /api/board was still pending. A 201 landing before that GET
// resolved got clobbered when the GET's now-stale response arrived and
// wholesale-replaced board state via setBoard(data) — the just-added
// card would silently vanish from the UI even though it existed on the
// server. Fixed by disabling the Add bar while loadState === 'loading'
// (App.tsx), which makes the race structurally impossible; this test
// locks that guarantee in.
import { delay, http, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';

import { App } from './App';
import { server } from './mocks/server';
import type { Board } from './api/types';

describe('App — Add bar vs. initial board load race', () => {
  it('disables the Add bar submit button while GET /api/board is in flight, re-enabling once it resolves', async () => {
    server.use(
      http.get('/api/board', async () => {
        await delay(50);
        return HttpResponse.json({ inbox: [], reading: [], done: [] } satisfies Board);
      }),
    );

    render(<App />);

    const button = screen.getByRole('button', { name: /save/i });
    expect(button).toBeDisabled();

    await waitFor(() => expect(button).toBeEnabled());
  });

  it('the Add bar is enabled immediately once a fast GET /api/board has already resolved', async () => {
    render(<App />);
    await screen.findAllByRole('heading', { level: 2 });
    expect(screen.getByRole('button', { name: /save/i })).toBeEnabled();
  });
});
