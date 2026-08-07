import { setupServer } from 'msw/node';

import { handlers } from './handlers';

// Node-side MSW server for Vitest — see src/test/setup.ts for lifecycle
// wiring (listen/resetHandlers/close).
export const server = setupServer(...handlers);
