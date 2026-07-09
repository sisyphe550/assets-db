import {
  BooleanNumber,
  LocaleType,
  type ICellData,
  type IObjectMatrixPrimitiveType,
  type IWorkbookData,
} from '@univerjs/core';
import type { SpreadsheetRow } from '@/utils/inventory';

const SHEET_ID = 'inventory-sheet';
const HEADERS = ['资产编号', '资产名称', '账面位置', '实际位置', '备注'];

export function buildInventoryWorkbook(rows: SpreadsheetRow[]): Partial<IWorkbookData> {
  const cellData: IObjectMatrixPrimitiveType<ICellData> = {};
  HEADERS.forEach((header, col) => {
    cellData[0] = cellData[0] ?? {};
    cellData[0][col] = { v: header };
  });

  rows.forEach((row, index) => {
    const rowIndex = index + 1;
    cellData[rowIndex] = {
      0: { v: row.assetNo },
      1: { v: row.name || '' },
      2: { v: row.bookLocation || '' },
      3: { v: row.actualLocation || '' },
      4: { v: row.notes || '' },
    };
  });

  const rowCount = Math.max(rows.length + 20, 30);

  return {
    id: 'inventory-workbook',
    name: '盘点表',
    appVersion: '0.5.5',
    locale: LocaleType.ZH_CN,
    styles: {},
    sheetOrder: [SHEET_ID],
    sheets: {
      [SHEET_ID]: {
        id: SHEET_ID,
        name: '盘点表',
        tabColor: '',
        hidden: BooleanNumber.FALSE,
        rowCount,
        columnCount: 8,
        freeze: { xSplit: 0, ySplit: 1, startRow: 1, startColumn: 0 },
        cellData,
        rowData: {},
        columnData: {
          0: { w: 160 },
          1: { w: 140 },
          2: { w: 160 },
          3: { w: 180 },
          4: { w: 160 },
        },
        defaultRowHeight: 28,
        defaultColumnWidth: 120,
        mergeData: [],
        showGridlines: BooleanNumber.TRUE,
        rowHeader: { width: 46 },
        columnHeader: { height: 20 },
        rightToLeft: BooleanNumber.FALSE,
      },
    },
  };
}

export function readRowsFromWorkbook(
  rows: SpreadsheetRow[],
  getCellValue: (row: number, col: number) => string | number | boolean | null | undefined,
): SpreadsheetRow[] {
  return rows.map((row, index) => {
    const sheetRow = index + 1;
    const assetNo = String(getCellValue(sheetRow, 0) ?? '');
    const name = String(getCellValue(sheetRow, 1) ?? '');
    const bookLocation = String(getCellValue(sheetRow, 2) ?? '');
    const actualLocation = String(getCellValue(sheetRow, 3) ?? '');
    const notes = String(getCellValue(sheetRow, 4) ?? '');
    const next: SpreadsheetRow = {
      ...row,
      assetNo,
      name,
      bookLocation,
      actualLocation,
      notes,
    };
    if (row.isSurplus) {
      next.foundName = name;
    }
    return next;
  });
}
