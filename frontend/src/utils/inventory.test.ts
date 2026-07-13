import { describe, it, expect } from 'vitest';
import {
  applySubmitResult,
  buildRowsFromExpected,
  buildSubmitItems,
  mergeDraftTimestamps,
  type SpreadsheetRow,
} from '@/utils/inventory';

const baseRow: SpreadsheetRow = {
  key: 'EQUIP-2026-0001',
  assetNo: 'EQUIP-2026-0001',
  name: '激光切割机',
  bookLocation: '101',
  actualLocation: '102',
  notes: '正常',
  foundName: '',
  isSurplus: false,
};

describe('inventory utils', () => {
  it('builds submit items only for edited rows', () => {
    const items = buildSubmitItems([baseRow]);
    expect(items).toHaveLength(1);
    expect(items[0].modifiedCells.actual_location).toBe('102');
  });

  it('skips rows with empty actual location', () => {
    const empty = { ...baseRow, actualLocation: '', notes: '' };
    expect(buildSubmitItems([empty])).toHaveLength(0);
  });

  it('skips rows where actual location matches book location', () => {
    const unchanged = { ...baseRow, actualLocation: '101', notes: '' };
    expect(buildSubmitItems([unchanged])).toHaveLength(1);
    expect(buildSubmitItems([unchanged])[0].modifiedCells.actual_location).toBe('101');
  });

  it('builds submit items for multiple edited rows', () => {
    const rows: SpreadsheetRow[] = [
      { ...baseRow, actualLocation: '102' },
      {
        ...baseRow,
        key: 'EQUIP-2026-0002',
        assetNo: 'EQUIP-2026-0002',
        actualLocation: '201',
      },
      {
        ...baseRow,
        key: 'EQUIP-2026-0003',
        assetNo: 'EQUIP-2026-0003',
        actualLocation: '301',
      },
    ];
    const items = buildSubmitItems(rows);
    expect(items).toHaveLength(3);
    expect(items.map((i) => i.assetNo)).toEqual([
      'EQUIP-2026-0001',
      'EQUIP-2026-0002',
      'EQUIP-2026-0003',
    ]);
  });

  it('includes surplus rows with name or location', () => {
    const surplus: SpreadsheetRow = {
      ...baseRow,
      assetNo: 'UNKNOWN-1',
      key: 'UNKNOWN-1',
      isSurplus: true,
      name: '未登记投影仪',
      foundName: '未登记投影仪',
      actualLocation: '',
      bookLocation: '',
    };
    expect(buildSubmitItems([surplus])).toHaveLength(1);
    expect(buildSubmitItems([surplus])[0].modifiedCells.found_name).toBe('未登记投影仪');
  });

  it('skips empty surplus rows', () => {
    const surplus: SpreadsheetRow = {
      ...baseRow,
      assetNo: 'UNKNOWN-1',
      key: 'UNKNOWN-1',
      isSurplus: true,
      name: '',
      actualLocation: '',
      bookLocation: '',
      notes: '',
    };
    expect(buildSubmitItems([surplus])).toHaveLength(0);
  });

  it('submits surplus book_location in modified cells', () => {
    const surplus: SpreadsheetRow = {
      ...baseRow,
      assetNo: 'CUSTOM-001',
      key: 'CUSTOM-001',
      isSurplus: true,
      name: '备用显示器',
      foundName: '备用显示器',
      bookLocation: '无',
      actualLocation: '仓库A',
    };
    const items = buildSubmitItems([surplus]);
    expect(items[0].modifiedCells.book_location).toBe('无');
    expect(items[0].assetNo).toBe('CUSTOM-001');
  });

  it('handles null submit result arrays from API', () => {
    const next = applySubmitResult([baseRow], {
      success: ['EQUIP-2026-0001'],
      conflicts: null,
      failures: null,
    });
    expect(next[0].rowState).toBe('success');
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

  it('merges expected assets with saved drafts', () => {
    const rows = buildRowsFromExpected(
      [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '101',
        },
      ],
      [
        {
          assetNo: 'EQUIP-2026-0001',
          modifiedCells: { actual_location: '205', temp_notes: '已搬' },
          updatedAt: '2026-07-08T10:00:00Z',
        },
      ],
    );
    expect(rows).toHaveLength(1);
    expect(rows[0].actualLocation).toBe('205');
    expect(rows[0].notes).toBe('已搬');
    expect(rows[0].expectedUpdatedAt).toBe('2026-07-08T10:00:00Z');
  });

  it('restores surplus rows from drafts not in expected list', () => {
    const rows = buildRowsFromExpected(
      [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '101',
        },
      ],
      [
        {
          assetNo: 'UNKNOWN-123',
          modifiedCells: {
            actual_location: '仓库',
            found_name: '未知设备',
            book_location: '账面无',
            temp_notes: '盘盈',
          },
          updatedAt: '2026-07-08T11:00:00Z',
        },
      ],
    );
    expect(rows).toHaveLength(2);
    const surplus = rows.find((r) => r.assetNo === 'UNKNOWN-123');
    expect(surplus?.isSurplus).toBe(true);
    expect(surplus?.name).toBe('未知设备');
    expect(surplus?.actualLocation).toBe('仓库');
    expect(surplus?.bookLocation).toBe('账面无');
  });

  it('merges draft timestamps without overwriting row content', () => {
    const rows: SpreadsheetRow[] = [
      {
        ...baseRow,
        actualLocation: '本地未保存编辑',
        expectedUpdatedAt: null,
      },
    ];
    const merged = mergeDraftTimestamps(rows, [
      {
        assetNo: 'EQUIP-2026-0001',
        modifiedCells: { actual_location: '服务端值' },
        updatedAt: '2026-07-09T04:00:00.000Z',
      },
    ]);
    expect(merged[0].actualLocation).toBe('本地未保存编辑');
    expect(merged[0].expectedUpdatedAt).toBe('2026-07-09T04:00:00.000Z');
  });

  it('does not show another operator draft cell values', () => {
    const rows = buildRowsFromExpected(
      [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '101',
        },
      ],
      [
        {
          assetNo: 'EQUIP-2026-0001',
          operatorId: 10003,
          modifiedCells: { actual_location: '学生填写' },
          updatedAt: '2026-07-09T04:00:00.000Z',
        },
      ],
      10002,
    );
    expect(rows[0].actualLocation).toBe('');
    expect(rows[0].expectedUpdatedAt).toBeNull();
  });

  it('uses own operator draft for CAS timestamp', () => {
    const rows = buildRowsFromExpected(
      [
        {
          assetId: 1,
          assetNo: 'EQUIP-2026-0001',
          name: '激光切割机',
          bookLocation: '101',
        },
      ],
      [
        {
          assetNo: 'EQUIP-2026-0001',
          operatorId: 10002,
          modifiedCells: { actual_location: '院管填写' },
          updatedAt: '2026-07-09T05:00:00.000Z',
        },
      ],
      10002,
    );
    expect(rows[0].expectedUpdatedAt).toBe('2026-07-09T05:00:00.000Z');
  });
});
