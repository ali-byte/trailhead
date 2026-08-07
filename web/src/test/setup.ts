import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterAll, afterEach, beforeAll } from 'vitest';

import { server } from '../mocks/server';

// onUnhandledRequest: 'error' — an un-mocked fetch is a test bug (a
// component calling an endpoint the locked Wire Contract doesn't define),
// not something to silently pass through.
beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));

afterEach(() => {
  server.resetHandlers();
  cleanup();
});

afterAll(() => server.close());
