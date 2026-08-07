import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './index.css';

// Pre-Phase F scaffold (issue #6) — a compiling placeholder only, so `vite
// build`/`vite dev` work before App.tsx exists. Phase F builds App.tsx (the
// board + Add bar) and wires it in here — see
// docs/issues/06-frontend-board-add-bar.md "UI Contract". Mirrors the
// backend precedent: cmd/trailhead/main.go was left as a minimal,
// compiling placeholder at the Phase B gate for the same reason.
const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('main.tsx: #root element not found in index.html');
}

createRoot(rootElement).render(
  <StrictMode>
    <div>Trailhead — build pending (Phase F)</div>
  </StrictMode>,
);
