// Shared reduced-motion check — design.md's "Reduced-motion rule (non-
// negotiable)" applies to every animation, including ones driven by
// inline styles a CSS media query can't reach on its own (dnd-kit's
// useSortable `transition` string is applied via a React inline style,
// not a CSS class, so the global @media (prefers-reduced-motion: reduce)
// rule in index.css cannot strip it — this hook is how JS-driven
// transitions opt out instead).
import { useEffect, useState } from 'react';

const QUERY = '(prefers-reduced-motion: reduce)';

// window.matchMedia is absent in jsdom (the locked Vitest/RTL test
// environment) unless a test explicitly polyfills it — guard its
// existence rather than assuming a browser environment, so this hook
// degrades to "no reduced-motion preference" instead of throwing.
function matchMediaSupported(): boolean {
  return typeof window !== 'undefined' && typeof window.matchMedia === 'function';
}

export function usePrefersReducedMotion(): boolean {
  const [reduced, setReduced] = useState(() => matchMediaSupported() && window.matchMedia(QUERY).matches);

  useEffect(() => {
    if (!matchMediaSupported()) return;
    const mediaQuery = window.matchMedia(QUERY);
    const onChange = (): void => setReduced(mediaQuery.matches);
    mediaQuery.addEventListener('change', onChange);
    return () => mediaQuery.removeEventListener('change', onChange);
  }, []);

  return reduced;
}
