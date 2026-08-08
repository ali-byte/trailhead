// Owns POST /api/bookmarks and every inline (400/409) error presentation —
// see AddBar.test.tsx for the locked contract and docs/issues/06-frontend-
// board-add-bar.md "UI Contract" (Add bar optimism: "no card until 201").
// Never touches board state itself — calls onCreated with the created
// domain.Bookmark so App can patch local state.
import { useId, useState } from 'react';
import type { FormEvent, MouseEvent } from 'react';

import { highlightExistingCard } from '../lib/highlightExistingCard';
import type { Bookmark, CreateBookmarkRequest, DuplicateErrorEnvelope, ErrorEnvelope } from '../api/types';

export interface AddBarProps {
  onCreated: (bookmark: Bookmark) => void;
  onTransientError: (message: string) => void;
  /** Optional external disable — App sets this true while the initial
   * board is still loading (loadState !== 'ready'), so a 201 here can
   * never race a still-in-flight GET /api/board (code-review fix: that
   * GET's now-stale response, arriving after the 201, would otherwise
   * wholesale-clobber the just-added card via setBoard(data)). Defaults
   * to false so every existing call site — including the locked
   * AddBar.test.tsx, which never passes this prop — is unaffected. */
  disabled?: boolean;
}

interface InlineError {
  message: string;
  /** Present only for the 409 duplicate case — the existing card to link to. */
  existingId?: string;
}

export function AddBar({ onCreated, onTransientError, disabled = false }: AddBarProps) {
  const [url, setUrl] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [inlineError, setInlineError] = useState<InlineError | null>(null);

  const inputId = useId();
  const errorId = useId();

  async function submit(): Promise<void> {
    // Belt-and-suspenders alongside the submit button's own `disabled`
    // attribute — a disabled submit button should already stop Enter-key
    // form submission in a real browser, but this guard doesn't depend on
    // that browser nuance holding in every environment.
    if (disabled) return;

    setSubmitting(true);
    setInlineError(null);

    try {
      const body: CreateBookmarkRequest = { url, title: null };
      const response = await fetch('/api/bookmarks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (response.status === 201) {
        const created = (await response.json()) as Bookmark;
        onCreated(created);
        setUrl('');
        return;
      }

      if (response.status === 409) {
        const envelope = (await response.json()) as DuplicateErrorEnvelope;
        setInlineError({ message: 'This is already on your board.', existingId: envelope.existing.id });
        // Automatic on the 409 itself — UI Contract "409 'link to the
        // existing card'": scroll to AND highlight it, not only when the
        // "Show it" link below is clicked.
        highlightExistingCard(envelope.existing.id);
        return;
      }

      if (response.status === 400) {
        const envelope = (await response.json()) as ErrorEnvelope;
        const message =
          envelope.error === 'invalid_url'
            ? "That doesn't look like a web address."
            : 'Paste a link before saving.';
        setInlineError({ message });
        return;
      }

      // 413 / 500 / anything else unexpected — a transient failure the App
      // surfaces as a toast, not an inline field message (UI Contract).
      // Must contain the literal phrase "Try again" (AddBar.test.tsx
      // asserts on that substring, not an exact string).
      onTransientError("Couldn't save that link. Try again.");
    } catch {
      onTransientError("Couldn't save that link. Try again.");
    } finally {
      setSubmitting(false);
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>): void {
    event.preventDefault();
    void submit();
  }

  // The href stays a real `#bookmark-{id}` fragment link (native
  // Cmd/Ctrl-click, middle-click, right-click all still work per web-
  // design-guidelines "Links use <a>"), but a plain click is handled here
  // instead of falling through to the browser's native hash-jump — the
  // highlight is no longer :target-driven (code-review fix: :target only
  // applied once a user clicked, and the highlight lingered indefinitely
  // rather than being time-boxed), so this reuses the exact same
  // scroll+highlight the 409 already triggers automatically.
  function handleShowExistingClick(event: MouseEvent<HTMLAnchorElement>, existingId: string): void {
    event.preventDefault();
    highlightExistingCard(existingId);
  }

  return (
    <form onSubmit={handleSubmit} className="flex w-full max-w-lg items-stretch gap-2">
      <label htmlFor={inputId} className="sr-only">
        Add a bookmark by URL
      </label>
      <input
        id={inputId}
        type="text"
        value={url}
        onChange={(event) => setUrl(event.target.value)}
        placeholder="Paste a link to save it…"
        aria-invalid={inlineError ? 'true' : undefined}
        aria-describedby={inlineError ? errorId : undefined}
        className="min-w-0 flex-1 rounded-md border border-border bg-surface px-3 py-2 text-base text-text placeholder:text-text-dim focus-visible:border-border-hi"
      />
      <button
        type="submit"
        disabled={submitting || disabled}
        className="min-h-btn shrink-0 rounded-md bg-primary px-4 text-base font-medium text-white transition-colors hover:bg-primary-dim disabled:cursor-default disabled:bg-surface-2 disabled:text-text-dim"
      >
        {submitting ? 'Saving…' : 'Save'}
      </button>
      {inlineError && (
        <p id={errorId} role="alert" className="basis-full text-sm text-error">
          {inlineError.message}
          {inlineError.existingId && (
            <>
              {' '}
              <a
                href={`#bookmark-${inlineError.existingId}`}
                onClick={(event) => handleShowExistingClick(event, inlineError.existingId as string)}
                className="font-medium underline"
              >
                Show it
              </a>
            </>
          )}
        </p>
      )}
    </form>
  );
}
