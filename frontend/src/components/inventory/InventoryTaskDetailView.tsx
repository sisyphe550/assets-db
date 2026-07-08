import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Empty,
  Result,
  Skeleton,
  Space,
  Spin,
  Table,
  Typography,
  message,
} from 'antd';
import { ArrowLeftOutlined, PlusOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import StatusTag from '@/components/common/StatusTag';
import {
  useArchiveTaskMutation,
  useGetExpectedAssetsQuery,
  useGetRecordsQuery,
  useGetTaskQuery,
  useSubmitRecordsMutation,
} from '@/store/api/inventoryApi';
import { formatDateTime } from '@/utils/format';
import {
  applySubmitResult,
  buildSubmitItems,
  type SpreadsheetRow,
} from '@/utils/inventory';

const InventorySpreadsheet = lazy(() => import('@/components/inventory/InventorySpreadsheet'));

interface InventoryTaskDetailViewProps {
  basePath: '/admin' | '/college' | '/user';
  showArchive?: boolean;
}

export default function InventoryTaskDetailView({
  basePath,
  showArchive = false,
}: InventoryTaskDetailViewProps) {
  const { id, taskId } = useParams();
  const taskIdNum = Number(id ?? taskId);
  const navigate = useNavigate();

  const { data: task, isLoading, isError, refetch } = useGetTaskQuery(taskIdNum, {
    skip: !taskIdNum,
  });
  const { data: expectedData, isLoading: expectedLoading } = useGetExpectedAssetsQuery(
    taskIdNum,
    { skip: !taskIdNum },
  );
  const { data: recordsData } = useGetRecordsQuery(
    { taskId: taskIdNum, page: 1, pageSize: 50 },
    { skip: !taskIdNum || task?.status !== 3 },
  );

  const [rows, setRows] = useState<SpreadsheetRow[]>([]);
  const [submitRecords, { isLoading: submitting }] = useSubmitRecordsMutation();
  const [archiveTask, { isLoading: archiving }] = useArchiveTaskMutation();

  const readOnly = !task || task.status !== 1;

  useEffect(() => {
    if (expectedData?.list) {
      setRows(
        expectedData.list.map((item) => ({
          key: item.assetNo,
          assetNo: item.assetNo,
          name: item.name,
          bookLocation: item.bookLocation,
          actualLocation: item.bookLocation,
          notes: '',
          foundName: '',
          isSurplus: false,
        })),
      );
    }
  }, [expectedData?.list]);

  const conflictMessages = useMemo(
    () => rows.filter((r) => r.rowState === 'conflict').map((r) => r.rowMessage).filter(Boolean),
    [rows],
  );

  const handleSubmit = async () => {
    const items = buildSubmitItems(rows);
    if (!items.length) {
      message.warning('请先填写盘点数据');
      return;
    }
    try {
      const result = await submitRecords({ taskId: taskIdNum, items }).unwrap();
      setRows((prev) => applySubmitResult(prev, result));
      if (result.conflicts.length) {
        message.error(`有 ${result.conflicts.length} 条冲突，请检查标红行`);
      } else if (result.failures.length) {
        message.warning(`有 ${result.failures.length} 条提交失败`);
      } else {
        message.success(`成功提交 ${result.success.length} 条`);
      }
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '提交失败');
    }
  };

  const addSurplusRow = () => {
    const no = `UNKNOWN-${Date.now()}`;
    setRows((prev) => [
      ...prev,
      {
        key: no,
        assetNo: no,
        name: '',
        bookLocation: '-',
        actualLocation: '',
        notes: '',
        foundName: '',
        isSurplus: true,
      },
    ]);
  };

  if (isLoading) {
    return <Spin size="large" style={{ display: 'block', margin: '40vh auto' }} />;
  }

  if (isError || !task) {
    return (
      <Result
        status="error"
        title="任务加载失败"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  const listPath =
    basePath === '/user' ? '/user/assets' : `${basePath}/inventory/tasks`;

  return (
    <Space direction="vertical" size="large" style={{ width: '100%' }}>
      <Card
        title={
          <Space>
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate(listPath)} />
            {task.taskName}
            <StatusTag type="inventory" value={task.status} />
          </Space>
        }
        extra={
          showArchive && task.status === 1 ? (
            <Button
              danger
              loading={archiving}
              onClick={async () => {
                try {
                  await archiveTask({ id: task.id }).unwrap();
                  message.success('已归档');
                } catch (err: unknown) {
                  const e = err as { message?: string };
                  message.error(e.message ?? '归档失败');
                }
              }}
            >
              归档
            </Button>
          ) : null
        }
      >
        <Typography.Paragraph type="secondary">
          时间：{formatDateTime(task.startTime)} ~ {formatDateTime(task.endTime)} | 进度：已提交{' '}
          {task.submittedCount ?? 0} / 应盘 {task.expectedAssetCount ?? 0}
        </Typography.Paragraph>
      </Card>

      {task.status === 3 && recordsData?.list?.length ? (
        <Card title="差异报告">
          <Table
            rowKey="assetNo"
            dataSource={recordsData.list}
            pagination={false}
            columns={[
              { title: '资产编号', dataIndex: 'assetNo' },
              { title: '名称', dataIndex: 'name' },
              { title: '账面位置', dataIndex: 'bookLocation' },
              { title: '实际位置', dataIndex: 'actualLocation' },
              {
                title: '差异',
                dataIndex: 'diffStatus',
                render: (v: number) => <StatusTag type="inventoryDiff" value={v} />,
              },
            ]}
          />
        </Card>
      ) : null}

      {task.status !== 3 && (
        <Card
          title="盘点表格"
          extra={
            !readOnly && (
              <Space>
                <Button icon={<PlusOutlined />} onClick={addSurplusRow}>
                  添加盘盈行
                </Button>
                <Button type="primary" loading={submitting} onClick={handleSubmit}>
                  保存草稿
                </Button>
              </Space>
            )
          }
        >
          {expectedLoading ? (
            <Skeleton active paragraph={{ rows: 10 }} />
          ) : !expectedData?.list?.length ? (
            <Empty description="暂无应盘资产" />
          ) : (
            <Suspense fallback={<Skeleton active paragraph={{ rows: 10 }} />}>
              <InventorySpreadsheet rows={rows} readOnly={readOnly} onChange={setRows} />
            </Suspense>
          )}
          {conflictMessages.length > 0 && (
            <Alert
              type="error"
              showIcon
              style={{ marginTop: 16 }}
              message="提交冲突"
              description={conflictMessages.join('；')}
            />
          )}
        </Card>
      )}
    </Space>
  );
}
