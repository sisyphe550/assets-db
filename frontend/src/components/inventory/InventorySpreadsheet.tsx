import { Input, Table } from 'antd';
import type { SpreadsheetRow } from '@/utils/inventory';

interface InventorySpreadsheetProps {
  rows: SpreadsheetRow[];
  readOnly: boolean;
  onChange: (rows: SpreadsheetRow[]) => void;
}

function rowClassName(record: SpreadsheetRow) {
  if (record.rowState === 'conflict') return 'inventory-row-conflict';
  if (record.rowState === 'success') return 'inventory-row-success';
  if (record.rowState === 'failure') return 'inventory-row-failure';
  return '';
}

export default function InventorySpreadsheet({
  rows,
  readOnly,
  onChange,
}: InventorySpreadsheetProps) {
  const updateRow = (key: string, patch: Partial<SpreadsheetRow>) => {
    onChange(rows.map((row) => (row.key === key ? { ...row, ...patch } : row)));
  };

  return (
    <>
      <style>{`
        .inventory-row-conflict td { background: #fff2f0 !important; border-left: 3px solid #ff4d4f; }
        .inventory-row-success td { background: #f6ffed !important; border-left: 3px solid #52c41a; }
        .inventory-row-failure td { background: #fffbe6 !important; }
      `}</style>
      <Table
        rowKey="key"
        dataSource={rows}
        pagination={false}
        scroll={{ x: 900, y: 480 }}
        rowClassName={rowClassName}
        columns={[
          { title: '资产编号', dataIndex: 'assetNo', width: 160 },
          { title: '资产名称', dataIndex: 'name', width: 140 },
          { title: '账面位置', dataIndex: 'bookLocation', width: 160 },
          {
            title: '实际位置',
            dataIndex: 'actualLocation',
            width: 180,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly ? (
                v
              ) : (
                <Input
                  value={v}
                  onChange={(e) => updateRow(record.key, { actualLocation: e.target.value })}
                />
              ),
          },
          {
            title: '备注',
            dataIndex: 'notes',
            width: 160,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly ? (
                v
              ) : (
                <Input
                  value={v}
                  onChange={(e) => updateRow(record.key, { notes: e.target.value })}
                />
              ),
          },
          {
            title: '盘盈名称',
            dataIndex: 'foundName',
            width: 160,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly || !record.isSurplus ? (
                v || '-'
              ) : (
                <Input
                  value={v}
                  placeholder="未登记资产名称"
                  onChange={(e) => updateRow(record.key, { foundName: e.target.value })}
                />
              ),
          },
        ]}
      />
    </>
  );
}
