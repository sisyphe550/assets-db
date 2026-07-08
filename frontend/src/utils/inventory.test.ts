import { describe, it, expect } from 'vitest';
import {
  applySubmitResult,
  buildSubmitItems,
  type SpreadsheetRow,
} from '@/utils/inventory';

const baseRow: SpreadsheetRow = {
  key: 'EQUIP-2026-0001',
  assetNo: 'EQUIP-2026-0001',
  name: '激光切割机',
  bookLocation: '101',
  actualLocation: '101',
  notes: '正常',
  foundName: '',
  isSurplus: false,
};

describe('inventory utils', () => {
  it('builds submit items from edited rows', () => {
    const items = buildSubmitItems([baseRow]);
    expect(items).toHaveLength(1);
    expect(items[0].modifiedCells.actual_location).toBe('101');
  });

  it('marks conflict and success rows', () => {
    const next = applySubmitResult([baseRow], {
      success: ['EQUIP-2026-0001'],
      conflicts: [],
      failures: [],
    });
    expect(next[0].rowState).toBe('success');

    const conflict = applySubmitResult([baseRow], {
      success: [],
      conflicts: [{ assetNo: 'EQUIP-2026-0001', message: '正在被他人盘点' }],
      failures: [],
    });
    expect(conflict[0].rowState).toBe('conflict');
  });
});
