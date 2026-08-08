// Scrolls to and briefly highlights an existing bookmark card — UI
// Contract "409 'link to the existing card'": on a duplicate response the
// app scrolls to AND highlights the existing card automatically, not only
// when a user clicks a link (code-review fix — the original :target/click
// -driven implementation didn't fire until the user clicked "Show it").
// A plain DOM utility, not React state: the target card lives in a wholly
// different part of the component tree than AddBar (which "never touches
// board state itself" — that's about React state, not this transient
// visual effect), and the highlight duration is intentionally NOT modeled
// as React state (no re-render should be needed to clear it).
const HIGHLIGHT_CLASS = 'card-root--highlight';
const HIGHLIGHT_DURATION_MS = 1500;

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );
}

/** No-ops if the card isn't currently rendered (id not found) — safe to
 * call unconditionally on every 409 response. */
export function highlightExistingCard(id: string): void {
  if (typeof document === 'undefined') return;
  const el = document.getElementById(`bookmark-${id}`);
  if (!el) return;

  // Reduced motion: instant scroll, no animated highlight fade — the
  // fade itself is handled by the .card-root--highlight CSS transition,
  // already neutralized under reduced-motion by index.css's global
  // `transition-duration` override; only the scroll behavior needs an
  // explicit branch here (scrollIntoView's `behavior` isn't CSS-driven).
  el.scrollIntoView({ behavior: prefersReducedMotion() ? 'auto' : 'smooth', block: 'center' });

  el.classList.add(HIGHLIGHT_CLASS);
  window.setTimeout(() => el.classList.remove(HIGHLIGHT_CLASS), HIGHLIGHT_DURATION_MS);
}
