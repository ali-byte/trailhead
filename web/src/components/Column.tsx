// One board column — not part of the locked module map (App.tsx owns the
// DndContext itself; this is Dispatch's own decomposition choice), but
// every element it renders satisfies the UI Contract's DOM/accessibility
// contract and design.md's "Layout Rules" / "Cards" / "Empty states" /
// "Loading states".
import { useDroppable } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import type { CSSProperties } from 'react';

import { Card } from './Card';
import { columnContainerId, COLUMN_DISPLAY_NAME } from '../dnd/boardDnd';
import { usePrefersReducedMotion } from '../hooks/usePrefersReducedMotion';
import type { Bookmark, Status } from '../api/types';

// Verbatim from design.md "Empty states" — per-column calm empty-state copy.
const EMPTY_STATE_COPY: Record<Status, string> = {
  inbox: 'Nothing saved yet. Paste a link above to start your board.',
  reading: 'Drag a card here when you start reading it.',
  done: 'Finished something? Drag it here to check it off.',
};

// design.md "Status column markers": "quiet per-column identity — used
// ONLY as a small dot / thin header underline, never as a column
// background fill." Written as a literal lookup (not a template string)
// so Tailwind's content scanner can see each class name — see
// tailwind.config.js's `status-*` colors, themselves CSS-variable-backed.
const STATUS_MARKER_CLASS: Record<Status, string> = {
  inbox: 'bg-status-inbox',
  reading: 'bg-status-reading',
  done: 'bg-status-done',
};

function SkeletonCard() {
  return (
    <div
      className="h-[92px] animate-skeleton-pulse rounded-lg border border-border bg-surface-2"
      aria-hidden="true"
    />
  );
}

interface SortableCardProps {
  bookmark: Bookmark;
}

// The dnd-kit sortable ref/listeners/attributes are attached directly to
// Card's own root div (the data-testid="card-{id}" node), not to this <li>
// — the <li> stays a plain native list item (implicit listitem role, no
// role="..." override) per the UI Contract's DOM/accessibility contract.
function SortableCard({ bookmark }: SortableCardProps) {
  const { setNodeRef, attributes, listeners, transform, transition, isDragging } = useSortable({
    id: bookmark.id,
  });
  const prefersReducedMotion = usePrefersReducedMotion();

  // design.md "Reduced-motion rule": "the neighbor reflow transition" is
  // decorative motion to remove under reduced-motion — "dragging still
  // works — the card repositions instantly" — so the transition string is
  // dropped entirely rather than shortened; the transform itself (the
  // card's actual position) is never skipped.
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition: prefersReducedMotion ? undefined : transition,
  };

  return (
    // aria-label — the ARIA "listitem" role's nameFrom is 'author' only
    // (no name-from-content, per the ARIA role spec), so an accessible
    // name must be supplied explicitly here for
    // getByRole('listitem', { name: ... }) to resolve; the <li> itself
    // still carries no explicit role (implicit listitem, per the DOM
    // contract).
    <li aria-label={bookmark.title}>
      <Card
        ref={setNodeRef}
        bookmark={bookmark}
        style={style}
        {...attributes}
        {...listeners}
        className={isDragging ? 'border-dashed opacity-40' : undefined}
      />
    </li>
  );
}

export interface ColumnProps {
  status: Status;
  bookmarks: Bookmark[];
  loading: boolean;
}

export function Column({ status, bookmarks, loading }: ColumnProps) {
  const headingId = `column-heading-${status}`;
  const { setNodeRef } = useDroppable({ id: columnContainerId(status) });
  const ids = bookmarks.map((b) => b.id);

  return (
    <section className="flex min-h-0 flex-1 flex-col rounded-xl bg-surface-2 p-4">
      <h2
        id={headingId}
        className="flex items-center gap-2 font-display text-xl font-medium text-text-bright"
      >
        <span aria-hidden="true" className={`h-2 w-2 rounded-full ${STATUS_MARKER_CLASS[status]}`} />
        {COLUMN_DISPLAY_NAME[status]}
      </h2>
      <div
        ref={setNodeRef}
        data-testid={columnContainerId(status)}
        className="mt-3 flex flex-1 flex-col gap-2"
      >
        {loading ? (
          <div data-testid="column-skeleton" className="flex flex-col gap-2">
            <SkeletonCard />
            <SkeletonCard />
          </div>
        ) : (
          <>
            <SortableContext items={ids} strategy={verticalListSortingStrategy}>
              <ul aria-labelledby={headingId} className="flex flex-col gap-2">
                {bookmarks.map((bookmark) => (
                  <SortableCard key={bookmark.id} bookmark={bookmark} />
                ))}
              </ul>
            </SortableContext>
            {bookmarks.length === 0 && (
              <p className="mt-6 px-2 text-center text-sm text-text-dim">{EMPTY_STATE_COPY[status]}</p>
            )}
          </>
        )}
      </div>
    </section>
  );
}
