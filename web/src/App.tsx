// The top-level component — see App.test.tsx for the locked contract and
// docs/issues/06-frontend-board-add-bar.md "UI Contract" for the full DOM/
// accessibility contract and DnD wiring this implements. Owns GET
// /api/board, loading/error/board state, the header (h1 + AddBar), the
// three columns, and dnd-kit's DndContext.
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  MeasuringStrategy,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import type { Announcements, DragCancelEvent, DragEndEvent, DragStartEvent } from '@dnd-kit/core';

import { AddBar } from './components/AddBar';
import { Card } from './components/Card';
import { Column } from './components/Column';
import { Toast } from './components/Toast';
import {
  COLUMN_DISPLAY_NAME,
  COLUMN_STATUSES,
  applyFinalOrder,
  captureMoveOrigin,
  computeFinalOrder,
  containerOf,
  createBoardCoordinateGetter,
  deriveMove,
  findBookmark,
  reconcileMove,
  undoMove,
} from './dnd/boardDnd';
import type { MoveOrigin } from './dnd/boardDnd';
import type { Board, Bookmark, MoveBookmarkRequest } from './api/types';

const EMPTY_BOARD: Board = { inbox: [], reading: [], done: [] };

type LoadState = 'loading' | 'ready' | 'error';

interface ToastState {
  message: string;
  onRetry?: () => void;
}

export function App() {
  const [board, setBoard] = useState<Board>(EMPTY_BOARD);
  const [loadState, setLoadState] = useState<LoadState>('loading');
  const [toast, setToast] = useState<ToastState | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);

  // A ref mirror of `board` so the keyboard coordinateGetter and the
  // announcements callbacks (both created once, held by dnd-kit for the
  // sensor/context's lifetime) always read the CURRENT board rather than
  // whatever it was when they were constructed.
  const boardRef = useRef(board);
  useEffect(() => {
    boardRef.current = board;
  }, [board]);

  const loadBoard = useCallback(async (): Promise<void> => {
    setLoadState('loading');
    try {
      const response = await fetch('/api/board');
      if (!response.ok) throw new Error('board fetch failed');
      const data = (await response.json()) as Board;
      setBoard(data);
      setLoadState('ready');
    } catch {
      setLoadState('error');
    }
  }, []);

  useEffect(() => {
    void loadBoard();
  }, [loadBoard]);

  const coordinateGetter = useMemo(() => createBoardCoordinateGetter(() => boardRef.current), []);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter }),
  );

  // dnd-kit's default MeasuringStrategy only measures droppable rects once,
  // at drag start — a keyboard ArrowRight/Left fired immediately after
  // pickup (Space) can race that initial measurement, so the custom
  // coordinateGetter's droppableRects.get(columnContainerId) sometimes
  // returns stale/undefined data and the keypress silently does nothing
  // (observed empirically as an intermittent e2e flake). Measuring
  // continuously during the drag removes that race.
  const measuring = useMemo(
    () => ({ droppable: { strategy: MeasuringStrategy.Always } }),
    [],
  );

  function handleCreated(bookmark: Bookmark): void {
    // Add bar optimism: "no card until 201" — patch local state directly
    // from the 201 body, no GET /api/board refetch (UI Contract).
    setBoard((prev) => ({ ...prev, inbox: [bookmark, ...prev.inbox] }));
  }

  function handleTransientError(message: string): void {
    setToast({ message });
  }

  function handleDragStart(event: DragStartEvent): void {
    setActiveId(String(event.active.id));
  }

  function handleDragCancel(_event: DragCancelEvent): void {
    setActiveId(null);
  }

  function handleDragEnd(event: DragEndEvent): void {
    setActiveId(null);
    const { active, over } = event;
    if (!over) return;

    const activeId = String(active.id);
    const overId = String(over.id);
    if (activeId === overId) return;

    const preDropBoard = boardRef.current;

    // Captured BEFORE the optimistic apply below, from the card's actual
    // pre-move position — this (not a whole-board snapshot) is what a
    // failed move rolls back to, so a rollback only ever touches this one
    // card. See undoMove's doc comment.
    const origin = captureMoveOrigin(preDropBoard, activeId);
    if (!origin) return;

    const derived = deriveMove(preDropBoard, activeId, overId);
    if (!derived) return;

    // True pre-response optimism (design.md "apply immediately,
    // reconcile") — the reorder/status change applies to local state the
    // instant the drop resolves, before the network call returns.
    setBoard(applyFinalOrder(preDropBoard, activeId, derived.finalOrder));
    void performMove(activeId, overId, origin);
  }

  async function performMove(id: string, overId: string, origin: MoveOrigin): Promise<void> {
    // Re-derived from the CURRENT board (boardRef.current), not a
    // snapshot captured back at drag-end — on the initial call this is
    // the same board the optimistic apply just used (nothing else has
    // run yet); on a Retry (well after the failure, board may have moved
    // on) this recomputes fresh so the retried request reflects whatever
    // the board looks like now, not what it looked like at the original
    // drop.
    const derived = deriveMove(boardRef.current, id, overId);
    if (!derived) return;
    const { finalOrder, before, after } = derived;

    try {
      const body: MoveBookmarkRequest = { target_status: finalOrder.container, before, after };
      const response = await fetch(`/api/bookmarks/${id}/move`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error('move failed');
      const updated = (await response.json()) as Bookmark;
      setBoard((prev) => reconcileMove(prev, updated));
    } catch {
      // Surgical rollback — undo ONLY this card's move against whatever
      // the CURRENT board is (a functional update), not a wholesale
      // restore of a stale pre-move snapshot. A wholesale restore would
      // also silently discard any add or other move that landed on the
      // board while this request was in flight; undoMove touches only
      // this one card. Design.md's own literal example copy for the
      // toast (UI Contract "Drag/move optimism").
      setBoard((prev) => undoMove(prev, id, origin));
      setToast({
        message: "Couldn't save that move — put it back?",
        onRetry: () => {
          setToast(null);
          const retryDerived = deriveMove(boardRef.current, id, overId);
          if (retryDerived) {
            setBoard((prev) => applyFinalOrder(prev, id, retryDerived.finalOrder));
          }
          void performMove(id, overId, origin);
        },
      });
    }
  }

  const announcements = useMemo<Announcements>(
    () => ({
      onDragStart({ active }) {
        const board = boardRef.current;
        const activeIdStr = String(active.id);
        const bookmark = findBookmark(board, activeIdStr);
        if (!bookmark) return undefined;
        // Include the source column using the UI's own name (never the
        // raw Status code value) — design.md Accessibility Baseline: drag
        // announcements use the UI's column names throughout, and a
        // pick-up announcement with no origin is ambiguous when more than
        // one card shares a title.
        const container = containerOf(board, activeIdStr);
        const from = container ? ` from ${COLUMN_DISPLAY_NAME[container]}` : '';
        return `Picked up ${bookmark.title}${from}.`;
      },
      onDragOver({ active, over }) {
        if (!over) return undefined;
        const board = boardRef.current;
        const activeIdStr = String(active.id);
        const overIdStr = String(over.id);
        const currentContainer = containerOf(board, activeIdStr);
        const currentIndex = currentContainer
          ? board[currentContainer].findIndex((b) => b.id === activeIdStr)
          : -1;

        const result = computeFinalOrder(board, activeIdStr, overIdStr);
        if (!result) return undefined;
        const newIndex = result.items.findIndex((b) => b.id === active.id);
        if (newIndex === -1) return undefined;

        // dnd-kit fires an initial onDragOver at pickup even before any
        // arrow key/pointer movement — collision detection may resolve
        // `over` to the active item's own id OR its own enclosing column
        // container, both no-ops. Skip announcing any hover that would
        // leave the card exactly where it already is, so it doesn't
        // overwrite onDragStart's "Picked up {title}." announcement.
        if (result.container === currentContainer && newIndex === currentIndex) return undefined;

        return `Moved to ${COLUMN_DISPLAY_NAME[result.container]}, position ${newIndex + 1} of ${result.items.length}.`;
      },
      onDragEnd() {
        return 'Dropped.';
      },
      onDragCancel() {
        return 'Movement cancelled.';
      },
    }),
    [],
  );

  const activeBookmark = activeId ? findBookmark(board, activeId) : null;

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      accessibility={{ announcements }}
      measuring={measuring}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      onDragCancel={handleDragCancel}
    >
      <div className="min-h-screen bg-bg">
        <header className="sticky top-0 z-10 border-b border-border bg-bg/95 px-4 py-3 backdrop-blur-sm sm:px-6 lg:px-8">
          <div className="mx-auto flex max-w-board flex-col gap-3 board:flex-row board:items-center board:gap-6">
            <h1 className="font-display text-3xl font-semibold text-text-bright">Trailhead</h1>
            <AddBar
              onCreated={handleCreated}
              onTransientError={handleTransientError}
              disabled={loadState === 'loading'}
            />
          </div>
        </header>

        <main className="mx-auto max-w-board px-4 py-6 sm:px-6 lg:px-8">
          {loadState === 'error' ? (
            <div className="flex flex-col items-center gap-4 py-24 text-center">
              <p className="max-w-sm text-text-dim">
                Couldn't load your board. Check your connection and try again.
              </p>
              <button
                type="button"
                onClick={() => void loadBoard()}
                className="min-h-btn rounded-md bg-primary px-4 text-base font-medium text-white hover:bg-primary-dim"
              >
                Try again
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-5 board:grid-cols-3">
              {COLUMN_STATUSES.map((status) => (
                <Column
                  key={status}
                  status={status}
                  bookmarks={board[status]}
                  loading={loadState === 'loading'}
                />
              ))}
            </div>
          )}
        </main>
      </div>

      {/* dropAnimation={null}: dnd-kit's default drop animation clones the
          overlay's last-rendered DOM node and animates it independently of
          React, outside our activeId-driven unmount — in practice its
          teardown did not reliably fire (a stale clone with data-testid
          duplicating the real, reconciled card, breaking Playwright's
          testid strict-mode uniqueness — e2e/cross-column-drag.spec.ts).
          Disabling it makes the overlay disappear the instant activeId
          resets to null, matching every locked test's expectation that
          `card-{id}` is unique in the DOM once a drag ends. */}
      <DragOverlay dropAnimation={null}>
        {activeBookmark ? <Card bookmark={activeBookmark} className="scale-[1.02] shadow-drag" /> : null}
      </DragOverlay>

      {toast && <Toast message={toast.message} onRetry={toast.onRetry} onDismiss={() => setToast(null)} />}
    </DndContext>
  );
}
