// Pure presentational bookmark card — see Card.test.tsx for the locked
// contract and design.md "Cards" for every style decision below (surface,
// border, radius, padding, type). No status label anywhere on the card —
// the column conveys status.
//
// Accepts (and spreads) standard div props so a drag wrapper (see
// components/Column.tsx's SortableCard, not locked) can attach dnd-kit's
// ref/role/tabIndex/aria-*/pointer-listeners directly onto the same DOM
// node the UI Contract's `data-testid="card-{id}"` targets — the wrapping
// `<li>` stays a plain, un-augmented native list item (the DOM contract's
// "implicit listitem role — no role='...' needed" is about that `<li>`,
// not about this component). Card.test.tsx never passes any of these
// extra props, so this is additive and does not change its own contract.
import { forwardRef } from 'react';
import type { HTMLAttributes } from 'react';

import type { Bookmark } from '../api/types';

function hostFromUrl(url: string): string {
  try {
    return new URL(url).hostname;
  } catch {
    // Not expected in practice — original_url is always a validated
    // absolute URL by the time a Bookmark reaches the client — but a
    // presentational component must not throw on malformed input.
    return url;
  }
}

// Explicit UTC per the UI Contract's "Finished-date formatting" — avoids
// the TZ-dependent-test class already scarred once on this project
// (trailhead-rules Rule 3 / FakeBookmarkRepository's UTC clock fix).
const finishedDateFormatter = new Intl.DateTimeFormat('en-US', {
  month: 'short',
  day: 'numeric',
  year: 'numeric',
  timeZone: 'UTC',
});

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  bookmark: Bookmark;
}

export const Card = forwardRef<HTMLDivElement, CardProps>(function Card(
  { bookmark, className, ...rest },
  ref,
) {
  const host = hostFromUrl(bookmark.original_url);
  const isDone = bookmark.status === 'done' && bookmark.finished_at !== null;

  return (
    <div
      ref={ref}
      id={`bookmark-${bookmark.id}`}
      data-testid={`card-${bookmark.id}`}
      className={`card-root cursor-grab rounded-lg border border-border bg-surface p-card transition-[box-shadow,border-color] hover:border-border-hi/60 active:cursor-grabbing ${className ?? ''}`}
      {...rest}
    >
      <a
        href={bookmark.original_url}
        className="font-display text-lg font-semibold text-text hover:text-primary"
      >
        {bookmark.title}
      </a>
      <p className="mt-1 text-sm text-text-dim">{host}</p>
      {bookmark.tags.length > 0 && (
        <ul className="mt-2 flex flex-wrap gap-1.5">
          {bookmark.tags.map((tag) => (
            <li
              key={tag}
              data-testid="tag-pill"
              className="rounded-full bg-surface-2 px-2.5 py-0.5 text-xs font-medium text-text-dim"
            >
              {tag}
            </li>
          ))}
        </ul>
      )}
      {isDone && (
        <p className="mt-2 text-sm text-text-dim">
          finished {finishedDateFormatter.format(new Date(bookmark.finished_at as string))}
        </p>
      )}
    </div>
  );
});
