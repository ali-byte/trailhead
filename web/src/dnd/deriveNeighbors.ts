// See web/src/dnd/deriveNeighbors.test.ts for the full locked contract and
// docs/issues/06-frontend-board-add-bar.md "DnD -> API mapping" for the
// UI Contract this implements.
//
// targetColumnIds is the target column's bookmark id array with the
// dragged bookmark already removed by the caller. dropIndex is the
// 0-based index the dragged card should land at; out-of-range values
// clamp into [0, targetColumnIds.length]. before is the PREDECESSOR of the
// (clamped) drop position — the id the moved card lands immediately AFTER
// — and after is the SUCCESSOR — the id the moved card lands immediately
// BEFORE. Pure: does not mutate targetColumnIds.
export interface Neighbors {
  before: string | null;
  after: string | null;
}

export function deriveNeighbors(targetColumnIds: string[], dropIndex: number): Neighbors {
  const length = targetColumnIds.length;
  const clampedIndex = Math.max(0, Math.min(dropIndex, length));

  const before = clampedIndex > 0 ? targetColumnIds[clampedIndex - 1] : null;
  const after = clampedIndex < length ? targetColumnIds[clampedIndex] : null;

  return { before, after };
}
