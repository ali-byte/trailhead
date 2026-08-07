// MSW request handlers reflecting the ACTUAL internal/api behavior and
// JSON shapes (internal/api/handlers.go, response.go — read at Pre-Phase F,
// not guessed) — see docs/issues/03-api-create-board.md and
// docs/issues/05-api-move.md "Wire Contract". Default handlers cover the
// happy path for every endpoint; error scenarios (409/400/413/500/404) are
// exported as factories individual tests layer in via `server.use(...)`.
import { http, HttpResponse } from 'msw';

import type { Board, Bookmark, CreateBookmarkRequest, MoveBookmarkRequest } from '../api/types';

let idCounter = 0;
function nextID(): string {
  idCounter += 1;
  return `00000000-0000-0000-0000-${String(idCounter).padStart(12, '0')}`;
}

/** Resets the id counter between test files — call from a test's own
 * beforeEach if a test depends on exact generated ids across cases. */
export function resetMockIDs(): void {
  idCounter = 0;
}

const FIXED_NOW = '2026-08-06T12:00:00Z';

export function makeBookmark(overrides: Partial<Bookmark> = {}): Bookmark {
  return {
    id: nextID(),
    original_url: 'https://example.com/article',
    canonical_url: 'https://example.com/article',
    identity_hash: 'deadbeef',
    title: 'Example Article',
    tags: [],
    status: 'inbox',
    position: '00000000',
    finished_at: null,
    author: null,
    created_at: FIXED_NOW,
    updated_at: FIXED_NOW,
    ...overrides,
  };
}

export const emptyBoard: Board = { inbox: [], reading: [], done: [] };

/** Default (happy-path) handlers — the base server is configured with
 * these; individual tests override via `server.use(errorHandlers.x())`. */
export const handlers = [
  http.get('/api/board', () => HttpResponse.json(emptyBoard)),

  http.post('/api/bookmarks', async ({ request }) => {
    const body = (await request.json()) as CreateBookmarkRequest;
    return HttpResponse.json(
      makeBookmark({
        original_url: body.url,
        canonical_url: body.url,
        title: body.title ?? 'Example Article',
      }),
      { status: 201 },
    );
  }),

  http.post('/api/bookmarks/:id/move', async ({ params, request }) => {
    const body = (await request.json()) as MoveBookmarkRequest;
    const isDone = body.target_status === 'done';
    return HttpResponse.json(
      makeBookmark({
        id: params.id as string,
        status: body.target_status,
        finished_at: isDone ? FIXED_NOW : null,
      }),
      { status: 200 },
    );
  }),
];

/** Error-scenario handler factories — see Wire Contract error envelopes.
 * `error`/`message` copy is the exact constant text handlers.go emits. */
export const errorHandlers = {
  createDuplicate: (existing: Bookmark) =>
    http.post('/api/bookmarks', () => HttpResponse.json({ error: 'duplicate', existing }, { status: 409 })),

  createInvalidUrl: () =>
    http.post('/api/bookmarks', () =>
      HttpResponse.json({ error: 'invalid_url', message: 'the submitted url is not valid' }, { status: 400 }),
    ),

  createBadRequest: (message = 'url is required') =>
    http.post('/api/bookmarks', () => HttpResponse.json({ error: 'bad_request', message }, { status: 400 })),

  createPayloadTooLarge: () =>
    http.post('/api/bookmarks', () =>
      HttpResponse.json({ error: 'payload_too_large', message: 'request body exceeds the maximum size' }, { status: 413 }),
    ),

  createInternalError: () =>
    http.post('/api/bookmarks', () =>
      HttpResponse.json({ error: 'internal', message: 'internal server error' }, { status: 500 }),
    ),

  boardOk: (board: Board) => http.get('/api/board', () => HttpResponse.json(board)),

  boardInternalError: () =>
    http.get('/api/board', () => HttpResponse.json({ error: 'internal', message: 'internal server error' }, { status: 500 })),

  moveNotFound: () =>
    http.post('/api/bookmarks/:id/move', () =>
      HttpResponse.json({ error: 'not_found', message: 'resource not found' }, { status: 404 }),
    ),

  moveInternalError: () =>
    http.post('/api/bookmarks/:id/move', () =>
      HttpResponse.json({ error: 'internal', message: 'internal server error' }, { status: 500 }),
    ),
};
