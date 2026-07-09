import type { ExpectedAsset, InventoryDraft, SubmitItem } from '@/types/api';

export interface SpreadsheetRow {
  key: string;
  assetNo: string;
  name: string;
  bookLocation: string;
  actualLocation: string;
  notes: string;
  foundName: string;
  isSurplus: boolean;
  expectedUpdatedAt?: string | null;
  rowState?: 'success' | 'conflict' | 'failure';
  rowMessage?: string;
}

function draftCell(cells: Record<string, string> | undefined, key: string): string {
  const v = cells?.[key];
  return v == null ? '' : String(v);
}

export function buildRowsFromExpected(
  expected: ExpectedAsset[],
  drafts: InventoryDraft[],
): SpreadsheetRow[] {
  const draftMap = new Map(drafts.map((d) => [d.assetNo, d]));
  const expectedNos = new Set(expected.map((e) => e.assetNo));

  const rows: SpreadsheetRow[] = expected.map((item) => {
    const draft = draftMap.get(item.assetNo);
    const cells = draft?.modifiedCells;
    return {
      key: item.assetNo,
      assetNo: item.assetNo,
      name: item.name,
      bookLocation: item.bookLocation,
      actualLocation: draftCell(cells, 'actual_location'),
      notes: draftCell(cells, 'temp_notes'),
      foundName: draftCell(cells, 'found_name'),
      isSurplus: false,
      expectedUpdatedAt: draft?.updatedAt ?? item.expectedUpdatedAt ?? null,
    };
  });

  for (const draft of drafts) {
    if (expectedNos.has(draft.assetNo)) continue;
    const cells = draft.modifiedCells;
    const name = draftCell(cells, 'found_name');
    rows.push({
      key: draft.assetNo,
      assetNo: draft.assetNo,
      name,
      bookLocation: draftCell(cells, 'book_location') || '-',
      actualLocation: draftCell(cells, 'actual_location'),
      notes: draftCell(cells, 'temp_notes'),
      foundName: name,
      isSurplus: true,
      expectedUpdatedAt: draft.updatedAt,
    });
  }

  return rows;
}

export function surplusHasEdits(row: SpreadsheetRow): boolean {
  if (row.name.trim() || row.actualLocation.trim() || row.notes.trim()) return true;
  const book = row.bookLocation.trim();
  return book !== '' && book !== '-';
}

function rowHasEdits(row: SpreadsheetRow): boolean {
  if (row.isSurplus) return surplusHasEdits(row);
  if (row.notes.trim() || row.foundName.trim()) return true;
  const actual = row.actualLocation.trim();
  if (!actual) return false;
  return actual !== row.bookLocation.trim();
}

export function buildSubmitItems(rows: SpreadsheetRow[]): SubmitItem[] {
  return rows.filter(rowHasEdits).map((row) => {
    const cells: Record<string, string> = {
      actual_location: row.actualLocation,
      temp_notes: row.notes,
      found_name: row.isSurplus ? row.name : row.foundName,
    };
    if (row.isSurplus) {
      cells.book_location = row.bookLocation;
    }
    return {
      assetNo: row.assetNo.trim(),
      modifiedCells: cells,
      expectedUpdatedAt: row.expectedUpdatedAt ?? null,
    };
  });
}

export function mergeDraftTimestamps(
  rows: SpreadsheetRow[],
  drafts: InventoryDraft[],
): SpreadsheetRow[] {
  const draftMap = new Map(drafts.map((d) => [d.assetNo, d.updatedAt]));
  return rows.map((row) => {
    const updatedAt = draftMap.get(row.assetNo);
    return updatedAt ? { ...row, expectedUpdatedAt: updatedAt } : row;
  });
}

export function applySubmitResult(
  rows: SpreadsheetRow[],
  result: {
    success?: string[] | null;
    conflicts?: { assetNo: string; message: string }[] | null;
    failures?: { assetNo: string; message: string }[] | null;
  },
): SpreadsheetRow[] {
  const success = result.success ?? [];
  const conflicts = result.conflicts ?? [];
  const failures = result.failures ?? [];

  const conflictMap = new Map(conflicts.map((c) => [c.assetNo, c.message]));
  const failureMap = new Map(failures.map((f) => [f.assetNo, f.message]));
  const successSet = new Set(success);

  return rows.map((row) => {
    if (conflictMap.has(row.assetNo)) {
      return { ...row, rowState: 'conflict', rowMessage: conflictMap.get(row.assetNo) };
    }
    if (failureMap.has(row.assetNo)) {
      return { ...row, rowState: 'failure', rowMessage: failureMap.get(row.assetNo) };
    }
    if (successSet.has(row.assetNo)) {
      return { ...row, rowState: 'success', rowMessage: undefined };
    }
    return { ...row, rowState: undefined, rowMessage: undefined };
  });
}
