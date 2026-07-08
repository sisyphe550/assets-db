import type { SubmitItem } from '@/types/api';

export interface SpreadsheetRow {
  key: string;
  assetNo: string;
  name: string;
  bookLocation: string;
  actualLocation: string;
  notes: string;
  foundName: string;
  isSurplus: boolean;
  rowState?: 'success' | 'conflict' | 'failure';
  rowMessage?: string;
}

export function buildSubmitItems(rows: SpreadsheetRow[]): SubmitItem[] {
  return rows
    .filter(
      (row) =>
        row.isSurplus ||
        row.actualLocation.trim() ||
        row.notes.trim() ||
        row.foundName.trim(),
    )
    .map((row) => ({
      assetNo: row.assetNo,
      modifiedCells: {
        actual_location: row.actualLocation,
        temp_notes: row.notes,
        found_name: row.foundName,
      },
      expectedUpdatedAt: null,
    }));
}

export function applySubmitResult(
  rows: SpreadsheetRow[],
  result: {
    success: string[];
    conflicts: { assetNo: string; message: string }[];
    failures: { assetNo: string; message: string }[];
  },
): SpreadsheetRow[] {
  const conflictMap = new Map(result.conflicts.map((c) => [c.assetNo, c.message]));
  const failureMap = new Map(result.failures.map((f) => [f.assetNo, f.message]));
  const successSet = new Set(result.success);

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
