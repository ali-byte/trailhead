// Board-level drag-and-drop helpers used by App.tsx — NOT part of the
// locked module map (only deriveNeighbors.ts is locked/tested directly);
// this module exists to keep App.tsx's DndContext wiring readable. See
// docs/issues/06-frontend-board-add-bar.md "UI Contract" (DnD -> API
// mapping, Optimistic UI + rollback, Keyboard / drag announcements) for
// the behavior this implements.
import { KeyboardCode } from '@dnd-kit/core';
import type { KeyboardCoordinateGetter } from '@dnd-kit/core';

import { deriveNeighbors } from './deriveNeighbors';
import type { Board, Bookmark, Status } from '../api/types';

export const COLUMN_STATUSES: Status[] = ['inbox', 'reading', 'done'];

export const COLUMN_DISPLAY_NAME: Record<Status, string> = {
  inbox: 'Inbox',
  reading: 'Reading',
  done: 'Done',
};

export function columnContainerId(status: Status): string {
  return `column-${status}`;
}

function statusFromColumnContainerId(id: string): Status | null {
  return COLUMN_STATUSES.find((status) => columnContainerId(status) === id) ?? null;
}

export function findBookmark(board: Board, id: string): Bookmark | null {
  for (const status of COLUMN_STATUSES) {
    const found = board[status].find((b) => b.id === id);
    if (found) return found;
  }
  return null;
}

/** Which column currently contains id — id may be a bookmark id, or a
 * column container id (`column-{status}`) when the pointer/keyboard target
 * is empty space in a column rather than a specific card. */
export function containerOf(board: Board, id: string): Status | null {
  const asColumn = statusFromColumnContainerId(id);
  if (asColumn) return asColumn;
  for (const status of COLUMN_STATUSES) {
    if (board[status].some((b) => b.id === id)) return status;
  }
  return null;
}

export interface FinalOrder {
  container: Status;
  items: Bookmark[];
}

/** Computes the resulting order of the target column after dropping
 * activeId onto whatever overId identifies (a specific card — lands
 * immediately before it — or a column container id, meaning empty space /
 * an empty column — lands at the end). Pure — does not mutate board. */
export function computeFinalOrder(board: Board, activeId: string, overId: string): FinalOrder | null {
  const activeContainer = containerOf(board, activeId);
  if (!activeContainer) return null;

  const overContainer = containerOf(board, overId) ?? activeContainer;
  const activeBookmark = findBookmark(board, activeId);
  if (!activeBookmark) return null;

  // Same technique for both same-container reorder and cross-container
  // move: remove active first, then insert it immediately before overId
  // in the resulting (post-removal) array. Using arrayMove's raw
  // (activeIndex, overIndex) pair for the same-container case instead
  // would land active AT overId's old slot (pushing overId back by one)
  // rather than immediately before it, which is dnd-kit's own canonical
  // sortable recipe but NOT what this project's locked reorder contract
  // wants (deriveNeighbors.test.ts / reorder.spec.ts: dropping onto a
  // card's slot means "insert immediately before it" — see the neighbor
  // semantics deriveNeighbors.ts itself derives from this same order).
  const withoutActive = board[overContainer].filter((b) => b.id !== activeId);
  let insertIndex = withoutActive.findIndex((b) => b.id === overId);
  if (insertIndex === -1) insertIndex = withoutActive.length;
  const items = [...withoutActive.slice(0, insertIndex), activeBookmark, ...withoutActive.slice(insertIndex)];
  return { container: overContainer, items };
}

/** Applies a completed drop to board optimistically — moves activeId into
 * finalOrder.container at finalOrder.items' order and updates its Status.
 * Position/FinishedAt/UpdatedAt are left as-is until the server response
 * reconciles them (UI Contract "Drag/move optimism": "only its Status/
 * Position change" is known client-side; the rest reconciles on 200). */
export function applyFinalOrder(board: Board, activeId: string, finalOrder: FinalOrder): Board {
  const sourceContainer = containerOf(board, activeId);
  const movedBookmark = finalOrder.items.find((b) => b.id === activeId);
  if (!movedBookmark) return board;

  const updatedItems = finalOrder.items.map((b) =>
    b.id === activeId ? { ...b, status: finalOrder.container } : b,
  );

  if (sourceContainer && sourceContainer !== finalOrder.container) {
    return {
      ...board,
      [sourceContainer]: board[sourceContainer].filter((b) => b.id !== activeId),
      [finalOrder.container]: updatedItems,
    };
  }

  return { ...board, [finalOrder.container]: updatedItems };
}

/** Reconciles board with the authoritative Bookmark returned by
 * POST /api/bookmarks/{id}/move — replaces the matching row in its
 * (now-current) column in place, preserving order — UI Contract "reconcile
 * the moved card with the server's authoritative response". */
export function reconcileMove(board: Board, updated: Bookmark): Board {
  const status = updated.status;
  return {
    ...board,
    [status]: board[status].map((b) => (b.id === updated.id ? updated : b)),
  };
}

export interface DerivedMove {
  finalOrder: FinalOrder;
  before: string | null;
  after: string | null;
}

/** Composes computeFinalOrder + deriveNeighbors into the one thing a move
 * attempt needs: where the card ends up, and the before/after neighbor
 * ids to send. Pulled out so both the initial optimistic apply and a
 * later Retry (re-derived against whatever board looks like AT RETRY
 * TIME, not a stale snapshot from the original drop) go through the
 * exact same derivation — see performMove/handleDragEnd in App.tsx. */
export function deriveMove(board: Board, activeId: string, overId: string): DerivedMove | null {
  const finalOrder = computeFinalOrder(board, activeId, overId);
  if (!finalOrder) return null;
  const targetIds = finalOrder.items.filter((b) => b.id !== activeId).map((b) => b.id);
  const dropIndex = finalOrder.items.findIndex((b) => b.id === activeId);
  const { before, after } = deriveNeighbors(targetIds, dropIndex);
  return { finalOrder, before, after };
}

/** Where a card sat immediately before an optimistic move was applied —
 * captured at drag-end so a failed move can be undone surgically (see
 * undoMove) instead of restoring an entire stale board snapshot. */
export interface MoveOrigin {
  container: Status;
  /** The id of the card immediately before this one in its origin
   * column, or null if it was first. */
  beforeId: string | null;
}

/** Computes activeId's MoveOrigin from board — call this BEFORE applying
 * the optimistic move (board must still contain activeId in its
 * pre-move position). */
export function captureMoveOrigin(board: Board, activeId: string): MoveOrigin | null {
  const container = containerOf(board, activeId);
  if (!container) return null;
  const items = board[container];
  const index = items.findIndex((b) => b.id === activeId);
  if (index === -1) return null;
  return { container, beforeId: index > 0 ? items[index - 1].id : null };
}

/** Surgically undoes a failed move: removes activeId from wherever it
 * CURRENTLY sits (which may differ from where the optimistic apply put
 * it, if something else has changed board state since) and reinserts it
 * into its origin column at approximately its original position (right
 * after origin.beforeId, or at the front if it was first) — WITHOUT
 * touching any other card. Unlike restoring a captured pre-move board
 * snapshot wholesale, this preserves any unrelated change (a new card
 * added via the Add bar, a different move completing) that landed while
 * this move was in flight — see App.tsx's performMove failure path.
 * A no-op if activeId no longer exists anywhere (deleted meanwhile). */
export function undoMove(board: Board, activeId: string, origin: MoveOrigin): Board {
  const bookmark = findBookmark(board, activeId);
  if (!bookmark) return board;

  const currentContainer = containerOf(board, activeId);
  const withoutCard: Board = currentContainer
    ? { ...board, [currentContainer]: board[currentContainer].filter((b) => b.id !== activeId) }
    : board;

  const targetArray = withoutCard[origin.container];
  const insertIndex = origin.beforeId
    ? Math.max(0, targetArray.findIndex((b) => b.id === origin.beforeId) + 1)
    : 0;
  const restored: Bookmark = { ...bookmark, status: origin.container };
  const newArray = [...targetArray.slice(0, insertIndex), restored, ...targetArray.slice(insertIndex)];

  return { ...withoutCard, [origin.container]: newArray };
}

/** Custom KeyboardSensor coordinateGetter — UI Contract "Keyboard / drag
 * announcements": ArrowRight/Left move the lifted card to the end of the
 * next/previous column; ArrowDown/Up move it later/earlier within its
 * current column's order. Space/Escape are dnd-kit's own default
 * KeyboardSensor behavior (pick-up/drop, cancel) and need no custom
 * handling here — only arrow-key movement needs a custom coordinateGetter
 * (dnd-kit's default only handles same-container reordering).
 *
 * getBoard is a getter (not a snapshot) so the sensor — constructed once
 * and held for the sensor's lifetime — always reads the current board via
 * a ref, not whatever board looked like when the sensor was configured.
 *
 * Returns a synthetic coordinate inside the target's rect so dnd-kit's own
 * collision detection resolves `over` to that target — the same mechanism
 * the pointer sensor uses, just driven by the keyboard.
 *
 * Rect lookup prefers a live DOM measurement (getRectByTestId) over
 * dnd-kit's own context.droppableRects cache: that cache is populated by
 * dnd-kit's MeasuringStrategy and observed empirically to occasionally
 * still be stale immediately after pickup (Space then an arrow key in
 * quick succession — an ArrowRight/Left fired before the very first
 * re-measurement lands would otherwise silently no-op). A live
 * getBoundingClientRect() read has no such lag. */
interface Rect {
  left: number;
  top: number;
  width: number;
  height: number;
  bottom: number;
}

function getRectByTestId(testId: string): Rect | null {
  if (typeof document === 'undefined') return null;
  const el = document.querySelector(`[data-testid="${testId}"]`);
  if (!el) return null;
  return el.getBoundingClientRect();
}

export function createBoardCoordinateGetter(getBoard: () => Board): KeyboardCoordinateGetter {
  return (event, { active, context }) => {
    const board = getBoard();
    const activeId = String(active);
    const activeContainer = containerOf(board, activeId);
    if (!activeContainer) return undefined;

    if (event.code === KeyboardCode.Right || event.code === KeyboardCode.Left) {
      event.preventDefault();
      const currentIndex = COLUMN_STATUSES.indexOf(activeContainer);
      const nextIndex = event.code === KeyboardCode.Right ? currentIndex + 1 : currentIndex - 1;
      if (nextIndex < 0 || nextIndex >= COLUMN_STATUSES.length) return undefined;
      const targetId = columnContainerId(COLUMN_STATUSES[nextIndex]);
      const rect = getRectByTestId(targetId) ?? context.droppableRects.get(targetId);
      if (!rect) return undefined;
      return { x: rect.left + rect.width / 2, y: rect.bottom - Math.min(8, rect.height / 4) };
    }

    if (event.code === KeyboardCode.Down || event.code === KeyboardCode.Up) {
      event.preventDefault();
      const items = board[activeContainer].map((b) => b.id);
      const currentIndex = items.indexOf(activeId);
      if (currentIndex === -1) return undefined;
      const targetIndex = event.code === KeyboardCode.Down ? currentIndex + 1 : currentIndex - 1;
      if (targetIndex < 0 || targetIndex >= items.length) return undefined;
      const targetItemId = items[targetIndex];
      const rect = getRectByTestId(`card-${targetItemId}`) ?? context.droppableRects.get(targetItemId);
      if (!rect) return undefined;
      return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
    }

    return undefined;
  };
}
