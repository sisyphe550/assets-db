import { useMemo, useState } from 'react';
import { Button, Card, Modal, Result, Select, Space, message } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import type { InventoryTask } from '@/types/api';
import { useArchiveTaskMutation, useGetTasksQuery } from '@/store/api/inventoryApi';
import StatusTag from '@/components/common/StatusTag';
import { formatDateTime } from '@/utils/format';

interface InventoryTaskTableProps {
  basePath: '/admin' | '/college';
}

export default function InventoryTaskTable({ basePath }: InventoryTaskTableProps) {
  const navigate = useNavigate();
  const [status, setStatus] = useState<number | undefined>();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);

  const { data, isLoading, isError, refetch } = useGetTasksQuery({
    page,
    pageSize,
    status,
  });
  const [archiveTask, { isLoading: archiving }] = useArchiveTaskMutation();

  const columns: ProColumns<InventoryTask>[] = useMemo(
    () => [
      { title: '任务名称', dataIndex: 'taskName', width: 220, ellipsis: true },
      { title: '范围部门ID', dataIndex: 'scopeDeptId', width: 110 },
      {
        title: '时间窗',
        width: 280,
        render: (_, r) => `${formatDateTime(r.startTime)} ~ ${formatDateTime(r.endTime)}`,
      },
      {
        title: '进度',
        width: 120,
        render: (_, r) =>
          `已提交 ${r.submittedCount ?? 0} / 应盘 ${r.expectedAssetCount ?? 0}`,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (_, r) => <StatusTag type="inventory" value={r.status} />,
      },
      {
        title: '操作',
        width: 180,
        render: (_, r) => (
          <Space>
            <Button
              type="link"
              onClick={() => navigate(`${basePath}/inventory/tasks/${r.id}`)}
            >
              {r.status === 0 ? '配置' : r.status === 1 ? '进入盘点' : '查看'}
            </Button>
            {r.status === 1 && (
              <Button
                type="link"
                danger
                loading={archiving}
                onClick={() => {
                  Modal.confirm({
                    title: '确认归档？',
                    content: '归档后盘点员将无法继续提交',
                    onOk: async () => {
                      try {
                        await archiveTask({ id: r.id }).unwrap();
                        message.success('已归档，进入比对阶段');
                      } catch (err: unknown) {
                        const e = err as { message?: string };
                        message.error(e.message ?? '归档失败');
                      }
                    },
                  });
                }}
              >
                归档
              </Button>
            )}
          </Space>
        ),
      },
    ],
    [archiveTask, archiving, basePath, navigate],
  );

  if (isError) {
    return (
      <Result
        status="error"
        title="加载失败"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  return (
    <Card
      title="盘点管理"
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate(`${basePath}/inventory/tasks/create`)}
        >
          创建任务
        </Button>
      }
    >
      <Space style={{ marginBottom: 16 }}>
        <Select
          allowClear
          placeholder="任务状态"
          style={{ width: 140 }}
          options={[
            { label: '待发布', value: 0 },
            { label: '进行中', value: 1 },
            { label: '比对中', value: 2 },
            { label: '已完成', value: 3 },
          ]}
          onChange={(v) => {
            setStatus(v);
            setPage(1);
          }}
        />
      </Space>

      <ProTable<InventoryTask>
        rowKey="id"
        search={false}
        options={false}
        loading={isLoading}
        columns={columns}
        dataSource={data?.list ?? []}
        pagination={{
          current: page,
          pageSize,
          total: data?.total ?? 0,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
        locale={{ emptyText: '暂无盘点任务' }}
      />
    </Card>
  );
}
