// Locked wire-contract types for the internal/api REST/JSON surface — see
// docs/issues/03-api-create-board.md and docs/issues/05-api-move.md "Wire
// Contract", and internal/domain/bookmark.go's struct tags (the source of
// truth these mirror). Field names are the exact JSON keys the Go handlers
// emit/accept — do not rename to camelCase.
//
// This file is Pre-Phase F scaffold, not an app component: it is the
// frontend analog of internal/adapter/ports.go — a locked contract the
// Phase F build implements against, not a UI element itself.

export type Status = 'inbox' | 'reading' | 'done';

export interface Bookmark {
  id: string;
  original_url: string;
  canonical_url: string;
  identity_hash: string;
  title: string;
  tags: string[];
  status: Status;
  position: string;
  /** ISO-8601 / RFC3339 timestamp, or null iff status !== 'done'. */
  finished_at: string | null;
  author: string | null;
  created_at: string;
  updated_at: string;
}

export interface Board {
  inbox: Bookmark[];
  reading: Bookmark[];
  done: Bookmark[];
}

/** Request body for POST /api/bookmarks. title is always null from the Add
 * bar in this issue — see UI Contract "Add bar fields (URL-only)". */
export interface CreateBookmarkRequest {
  url: string;
  title: string | null;
}

/** Request body for POST /api/bookmarks/{id}/move. */
export interface MoveBookmarkRequest {
  target_status: Status;
  before: string | null;
  after: string | null;
}

/** The locked {"error": string, "message": string} envelope for every
 * error response except 409, which additionally carries "existing". */
export interface ErrorEnvelope {
  error: 'bad_request' | 'invalid_url' | 'not_found' | 'method_not_allowed' | 'payload_too_large' | 'internal';
  message: string;
}

export interface DuplicateErrorEnvelope {
  error: 'duplicate';
  existing: Bookmark;
}
