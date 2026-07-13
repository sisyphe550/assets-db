import type { Dispatch, SetStateAction } from 'react';
import { Input, Table } from 'antd';
import type { SpreadsheetRow } from '@/utils/inventory';

interface InventorySpreadsheetProps {
  rows: SpreadsheetRow[];
  readOnly: boolean;
  onChange: Dispatch<SetStateAction<SpreadsheetRow[]>>;
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
    onChange((prev) =>
      prev.map((row) => (row.key === key ? { ...row, ...patch } : row)),
    );
  };

  const surplusInput = (
    value: string,
    record: SpreadsheetRow,
    field: keyof Pick<SpreadsheetRow, 'assetNo' | 'name' | 'bookLocation'>,
    placeholder?: string,
  ) => (
    <Input
      value={value}
      placeholder={placeholder}
      onChange={(e) => {
        const next = e.target.value;
        if (field === 'assetNo') {
          updateRow(record.key, { assetNo: next, key: next.trim() || record.key });
          return;
        }
        if (field === 'name') {
          updateRow(record.key, { name: next, foundName: next });
          return;
        }
        updateRow(record.key, { [field]: next });
      }}
    />
  );

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
          {
            title: '资产编号',
            dataIndex: 'assetNo',
            width: 160,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly || !record.isSurplus ? (
                v
              ) : (
                surplusInput(v, record, 'assetNo', '未登记编号')
              ),
          },
          {
            title: '资产名称',
            dataIndex: 'name',
            width: 140,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly || !record.isSurplus ? (
                v || '-'
              ) : (
                surplusInput(v, record, 'name', '盘盈资产名称')
              ),
          },
          {
            title: '账面位置',
            dataIndex: 'bookLocation',
            width: 160,
            render: (v: string, record: SpreadsheetRow) =>
              readOnly || !record.isSurplus ? (
                v
              ) : (
                surplusInput(v, record, 'bookLocation', '账面无记录可填「-」')
              ),
          },
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
        ]}
      />
    </>
  );
}
