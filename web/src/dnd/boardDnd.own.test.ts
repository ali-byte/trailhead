// Dispatch's own regression coverage (NOT a locked test file) — added per
// code-review: the surgical-rollback fix to App.tsx's performMove isn't
// exercised by any locked test file, so it's locked in here instead,
// directly against the pure helpers App.tsx composes (captureMoveOrigin /
// undoMove / deriveMove). See App.tsx's handleDragEnd/performMove for how
// these are wired into the actual failure/retry path.
import { describe, expect, it } from 'vitest';

import { MoveGenerationTracker, captureMoveOrigin, deriveMove, undoMove } from './boardDnd';
import type { Board, Bookmark } from '../api/types';

function bookmark(id: string, overrides: Partial<Bookmark> = {}): Bookmark {
  return {
    id,
    original_url: `https://example.com/${id}`,
    canonical_url: `https://example.com/${id}`,
    identity_hash: id,
    title: id,
    tags: [],
    status: 'inbox',
    position: id,
    finished_at: null,
    author: null,
    created_at: '2026-08-01T00:00:00Z',
    updated_at: '2026-08-01T00:00:00Z',
    ...overrides,
  };
}

describe('captureMoveOrigin', () => {
  it('records the container and the id of the preceding card', () => {
    const board: Board = {
      inbox: [bookmark('a'), bookmark('b'), bookmark('c')],
      reading: [],
      done: [],
    };
    expect(captureMoveOrigin(board, 'b')).toEqual({ container: 'inbox', beforeId: 'a' });
  });

  it('beforeId is null when the card is first in its column', () => {
    const board: Board = { inbox: [bookmark('a'), bookmark('b')], reading: [], done: [] };
    expect(captureMoveOrigin(board, 'a')).toEqual({ container: 'inbox', beforeId: null });
  });

  it('returns null when the card does not exist anywhere', () => {
    const board: Board = { inbox: [], reading: [], done: [] };
    expect(captureMoveOrigin(board, 'missing')).toBeNull();
  });
});

describe('undoMove — surgical rollback', () => {
  it('restores the card to its origin column/position WITHOUT discarding an unrelated card added while the move was in flight', () => {
    // Starting state: Inbox = [A, B], Reading = [].
    const preDropBoard: Board = {
      inbox: [bookmark('a'), bookmark('b')],
      reading: [],
      done: [],
    };

    // A is dragged to Reading — capture its origin BEFORE applying the
    // optimistic move (mirrors App.tsx's handleDragEnd ordering).
    const origin = captureMoveOrigin(preDropBoard, 'a');
    expect(origin).toEqual({ container: 'inbox', beforeId: null });

    // Optimistic apply: A moves to Reading. Inbox = [B], Reading = [A].
    const afterOptimisticApply: Board = {
      inbox: [bookmark('b')],
      reading: [bookmark('a', { status: 'reading' })],
      done: [],
    };

    // While the Move request for A is still in flight, an UNRELATED card
    // C is added via the Add bar (prepended to Inbox, per the Add bar's
    // own optimism) — this is the state the rollback must run against.
    const boardWithConcurrentAdd: Board = {
      inbox: [bookmark('c'), ...afterOptimisticApply.inbox],
      reading: afterOptimisticApply.reading,
      done: [],
    };

    // The Move for A fails. A wholesale rollback to preDropBoard would
    // silently erase C. undoMove must not.
    const rolledBack = undoMove(boardWithConcurrentAdd, 'a', origin!);

    expect(rolledBack.reading).toEqual([]);
    expect(rolledBack.inbox.map((b) => b.id)).toEqual(['a', 'c', 'b']);
    // A's status is restored to its origin container.
    expect(rolledBack.inbox.find((b) => b.id === 'a')?.status).toBe('inbox');
    // C — added after the optimistic apply, during the failed request —
    // survives the rollback untouched.
    expect(rolledBack.inbox.some((b) => b.id === 'c')).toBe(true);
  });

  it('restores a card that had a predecessor to immediately after that predecessor, even if other cards were inserted between them meanwhile', () => {
    // Inbox = [A, B, C]; B is dragged out to Done.
    const preDropBoard: Board = {
      inbox: [bookmark('a'), bookmark('b'), bookmark('c')],
      reading: [],
      done: [],
    };
    const origin = captureMoveOrigin(preDropBoard, 'b');
    expect(origin).toEqual({ container: 'inbox', beforeId: 'a' });

    // While the Move is in flight: B already optimistically left Inbox,
    // and a concurrent reorder/add put a new card D between A and C.
    const boardDuringFlight: Board = {
      inbox: [bookmark('a'), bookmark('d'), bookmark('c')],
      reading: [],
      done: [bookmark('b', { status: 'done' })],
    };

    const rolledBack = undoMove(boardDuringFlight, 'b', origin!);

    expect(rolledBack.done).toEqual([]);
    // B lands right after A (its recorded predecessor), ahead of D and C
    // — both of which are preserved from the concurrent change.
    expect(rolledBack.inbox.map((b) => b.id)).toEqual(['a', 'b', 'd', 'c']);
  });

  it('is a no-op when the card no longer exists anywhere (e.g. deleted while the move was in flight)', () => {
    const board: Board = { inbox: [bookmark('a')], reading: [], done: [] };
    const result = undoMove(board, 'gone', { container: 'inbox', beforeId: null });
    expect(result).toBe(board);
  });
});

describe('deriveMove', () => {
  it('re-derives a fresh result from whatever board is passed in — not a captured snapshot', () => {
    const boardAtDropTime: Board = {
      inbox: [bookmark('a'), bookmark('b')],
      reading: [],
      done: [],
    };
    const firstDerived = deriveMove(boardAtDropTime, 'a', 'column-reading');
    expect(firstDerived?.finalOrder.container).toBe('reading');
    expect(firstDerived?.before).toBeNull();
    expect(firstDerived?.after).toBeNull();

    // Board changes shape by the time of a Retry (a is already gone from
    // inbox — some other flow moved it, or state simply advanced) —
    // deriveMove must reflect the board it's given, not the first call's
    // result.
    const boardAtRetryTime: Board = {
      inbox: [bookmark('b')],
      reading: [bookmark('c', { status: 'reading' })],
      done: [],
    };
    const retryDerived = deriveMove(boardAtRetryTime, 'a', 'column-reading');
    // 'a' no longer exists in boardAtRetryTime, so there is nothing valid
    // to derive — deriveMove reports that rather than silently reusing
    // stale data.
    expect(retryDerived).toBeNull();
  });
});

// Regression coverage for the code-review fix: two in-flight moves of the
// SAME card, where a stale response (an earlier move failing OR
// succeeding after a later move has already landed) must never apply —
// only the latest attempt for a card may ever reconcile or roll back.
// App.tsx's performMove is the actual caller (next() at drag-end/retry,
// isCurrent() immediately before every rollback/reconcile setBoard call);
// this tests the exact generation-comparison logic that fix relies on.
describe('MoveGenerationTracker', () => {
  it('a later next() for the same id supersedes an earlier one', () => {
    const tracker = new MoveGenerationTracker();
    const firstGen = tracker.next('a');
    expect(tracker.isCurrent('a', firstGen)).toBe(true);

    const secondGen = tracker.next('a');
    expect(tracker.isCurrent('a', firstGen)).toBe(false);
    expect(tracker.isCurrent('a', secondGen)).toBe(true);
  });

  it('tracks generations independently per card id', () => {
    const tracker = new MoveGenerationTracker();
    const genA = tracker.next('a');
    const genB = tracker.next('b');
    expect(tracker.isCurrent('a', genA)).toBe(true);
    expect(tracker.isCurrent('b', genB)).toBe(true);

    // A third card's move must not affect a's or b's generations.
    tracker.next('c');
    expect(tracker.isCurrent('a', genA)).toBe(true);
    expect(tracker.isCurrent('b', genB)).toBe(true);
  });

  it('move A twice: the second succeeds, the first then fails — the stale failure must be ignored (A stays where the second move put it)', () => {
    const tracker = new MoveGenerationTracker();

    // Drag 1 starts (handleDragEnd calls next('a')).
    const firstMoveGen = tracker.next('a');

    // Before drag 1's request resolves, the card is dragged again — drag
    // 2 starts, superseding drag 1.
    const secondMoveGen = tracker.next('a');

    // Drag 2's response lands first and succeeds — performMove checks
    // isCurrent before reconciling.
    expect(tracker.isCurrent('a', secondMoveGen)).toBe(true);
    // (caller would call setBoard(prev => reconcileMove(prev, updated)) here)

    // Drag 1's response lands afterward and fails — performMove checks
    // isCurrent before rolling back, per the SAME gate.
    expect(tracker.isCurrent('a', firstMoveGen)).toBe(false);
    // Because firstMoveGen is no longer current, the caller must skip
    // undoMove entirely — the card is left exactly where drag 2 (the
    // still-current, already-applied move) put it. There is nothing
    // further to assert against board state here: the fix IS that no
    // setBoard call happens for the stale response at all.
  });

  it('a retried move gets its own new generation, distinct from the original failed attempt', () => {
    const tracker = new MoveGenerationTracker();
    const originalGen = tracker.next('a');
    // Original attempt fails; its rollback is applied (isCurrent is still
    // true at this point — nothing else has superseded it yet).
    expect(tracker.isCurrent('a', originalGen)).toBe(true);

    // User clicks Retry — App.tsx calls next('a') again for the retry.
    const retryGen = tracker.next('a');
    expect(retryGen).not.toBe(originalGen);
    expect(tracker.isCurrent('a', originalGen)).toBe(false);
    expect(tracker.isCurrent('a', retryGen)).toBe(true);
  });
});
