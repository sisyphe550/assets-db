import { Button, Card, Input, Radio, Space, Table, Typography, message } from 'antd';
import { useMemo, useState } from 'react';
import {
  useGetConflictsQuery,
  useResolveConflictMutation,
} from '@/store/api/inventoryApi';
import type { InventoryConflict } from '@/types/api';

interface InventoryConflictPanelProps {
  taskId: number;
  onResolved?: () => void;
}

type ResolveMode = { type: 'assignee'; operatorId: number } | { type: 'custom' };

export default function InventoryConflictPanel({
  taskId,
  onResolved,
}: InventoryConflictPanelProps) {
  const { data, isLoading, refetch } = useGetConflictsQuery(taskId);
  const [resolveConflict, { isLoading: resolving }] = useResolveConflictMutation();
  const [selection, setSelection] = useState<Record<string, ResolveMode>>({});
  const [customValues, setCustomValues] = useState<
    Record<string, { actualLocation: string; notes: string }>
  >({});

  const conflicts = data?.list ?? [];

  const columns = useMemo(
    () => [
      { title: '资产编号', dataIndex: 'assetNo', width: 160 },
      { title: '名称', dataIndex: 'name', width: 140, render: (v: string) => v ?? '-' },
      {
        title: '账面位置',
        dataIndex: 'bookLocation',
        width: 160,
        render: (v: string) => v ?? '-',
      },
      {
        title: '裁决',
        render: (_: unknown, row: InventoryConflict) => {
          const sel = selection[row.assetNo];
          const radioValue =
            sel?.type === 'assignee'
              ? `assignee:${sel.operatorId}`
              : sel?.type === 'custom'
                ? 'custom'
                : undefined;
          return (
          <Radio.Group
            value={radioValue}
            onChange={(e) => {
              const val = e.target.value as string;
              if (val === 'custom') {
                setSelection((prev) => ({ ...prev, [row.assetNo]: { type: 'custom' } }));
                return;
              }
              if (val.startsWith('assignee:')) {
                const operatorId = Number(val.slice('assignee:'.length));
                setSelection((prev) => ({
                  ...prev,
                  [row.assetNo]: { type: 'assignee', operatorId },
                }));
              }
            }}
          >
            <Space direction="vertical">
              {row.candidates.map((c) => (
                <Radio key={c.operatorId} value={`assignee:${c.operatorId}`}>
                  {c.operatorName ?? `#${c.operatorId}`}：{c.actualLocation || '（空）'}
                  {c.notes ? `（${c.notes}）` : ''}
                </Radio>
              ))}
              <Radio value="custom">管理员自定义</Radio>
            </Space>
          </Radio.Group>
          );
        },
      },
      {
        title: '自定义实际位置',
        width: 200,
        render: (_: unknown, row: InventoryConflict) =>
          selection[row.assetNo]?.type === 'custom' ? (
            <Input
              placeholder="实际位置"
              value={customValues[row.assetNo]?.actualLocation ?? ''}
              onChange={(e) =>
                setCustomValues((prev) => ({
                  ...prev,
                  [row.assetNo]: {
                    actualLocation: e.target.value,
                    notes: prev[row.assetNo]?.notes ?? '',
                  },
                }))
              }
            />
          ) : (
            '-'
          ),
      },
    ],
    [conflicts, customValues, selection],
  );

  const handleResolve = async (row: InventoryConflict) => {
    const mode = selection[row.assetNo];
    if (!mode) {
      message.warning('请先选择采纳哪条记录或自定义');
      return;
    }
    try {
      const body =
        mode.type === 'assignee'
          ? { source: 'assignee' as const, operatorId: mode.operatorId }
          : {
              source: 'custom' as const,
              actualLocation: customValues[row.assetNo]?.actualLocation?.trim() ?? '',
              notes: customValues[row.assetNo]?.notes?.trim() ?? '',
            };
      const result = await resolveConflict({
        taskId,
        assetNo: row.assetNo,
        body,
      }).unwrap();
      message.success(
        result.allResolved ? '已全部裁决，正在账实比对…' : `已裁决 ${row.assetNo}`,
      );
      await refetch();
      onResolved?.();
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '裁决失败');
    }
  };

  if (!conflicts.length && !isLoading) {
    return null;
  }

  return (
    <Card title="盘点员冲突裁决" loading={isLoading}>
      <Typography.Paragraph type="secondary">
        多名盘点员对同一资产填写不一致。请逐条选择采纳的记录，或由管理员填写新的实际位置；全部裁决完成后系统将自动进行账实比对。
      </Typography.Paragraph>
      <Table
        rowKey="assetNo"
        dataSource={conflicts}
        columns={[
          ...columns,
          {
            title: '操作',
            width: 100,
            render: (_: unknown, row: InventoryConflict) => (
              <Button
                type="primary"
                size="small"
                loading={resolving}
                onClick={() => handleResolve(row)}
              >
                确认裁决
              </Button>
            ),
          },
        ]}
        pagination={false}
      />
    </Card>
  );
}
