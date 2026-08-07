// Playwright pointer-drag + DOM-reading helpers shared across
// web/e2e/*.spec.ts. Not a spec file itself.
//
// UI Contract element ids these helpers rely on (locked — the Phase F
// build must render exactly these):
//   - a card's root element: [data-testid="card-{id}"], containing a
//     single <a> whose text is the bookmark's title (the same link
//     Card.test.tsx locks via getByRole('link', { name: title })).
//   - a column's droppable container: [data-testid="column-{status}"]
//     (status = inbox | reading | done).
import type { Locator, Page } from '@playwright/test';

/** Drags `source` onto `target` via real, multi-step pointer events.
 * Playwright's built-in locator.dragTo() fires too few pointermove events
 * for dnd-kit's PointerSensor to register the movement past its
 * activation constraint and run collision detection along the way — this
 * helper steps the pointer so dnd-kit sees the drag the way a real mouse
 * would produce it. */
export async function dragCardTo(page: Page, source: Locator, target: Locator): Promise<void> {
  const sourceBox = await source.boundingBox();
  const targetBox = await target.boundingBox();
  if (!sourceBox || !targetBox) {
    throw new Error('dragCardTo: source or target element has no bounding box (not visible/rendered)');
  }

  const startX = sourceBox.x + sourceBox.width / 2;
  const startY = sourceBox.y + sourceBox.height / 2;
  const endX = targetBox.x + targetBox.width / 2;
  const endY = targetBox.y + targetBox.height / 2;

  await page.mouse.move(startX, startY);
  await page.mouse.down();

  const steps = 10;
  for (let i = 1; i <= steps; i += 1) {
    await page.mouse.move(startX + ((endX - startX) * i) / steps, startY + ((endY - startY) * i) / steps, {
      steps: 5,
    });
  }

  await page.mouse.up();
}

/** Reads the card titles currently rendered in a column, in DOM order —
 * used to assert reorder/persist behavior without depending on the fake's
 * or the real adapter's internal Position string representation. */
export async function getColumnOrder(page: Page, status: 'inbox' | 'reading' | 'done'): Promise<string[]> {
  const cards = page.getByTestId(`column-${status}`).locator('[data-testid^="card-"]');
  return cards.evaluateAll((elements) => elements.map((el) => el.querySelector('a')?.textContent?.trim() ?? ''));
}
