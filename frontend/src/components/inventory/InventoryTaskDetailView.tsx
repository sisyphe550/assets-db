import { lazy, Suspense, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Empty,
  Modal,
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
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import { getApiErrorCode } from '@/utils/api';
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
  const user = useAppSelector(selectCurrentUser);
  const [pollComparing, setPollComparing] = useState(false);

  const { data: task, isLoading, isError, error, refetch } = useGetTaskQuery(taskIdNum, {
    skip: !taskIdNum,
    pollingInterval: pollComparing ? 2000 : 0,
  });
  const { data: expectedData, isLoading: expectedLoading } = useGetExpectedAssetsQuery(
    taskIdNum,
    { skip: !taskIdNum || task?.status !== 1 },
  );
  const { data: recordsData } = useGetRecordsQuery(
    { taskId: taskIdNum, page: 1, pageSize: 50 },
    { skip: !taskIdNum || task?.status !== 3 },
  );

  const [rows, setRows] = useState<SpreadsheetRow[]>([]);
  const [submitRecords, { isLoading: submitting }] = useSubmitRecordsMutation();
  const [archiveTask, { isLoading: archiving }] = useArchiveTaskMutation();

  useEffect(() => {
    if (task?.status === 2) {
      setPollComparing(true);
    }
    if (task?.status === 3) {
      setPollComparing(false);
    }
  }, [task?.status]);

  const readOnly = !task || task.status !== 1;

  const assigneeDenied = useMemo(() => {
    if (basePath !== '/user' || !user || !task) return false;
    return !task.assigneeIds.includes(user.id);
  }, [basePath, user, task]);

  useEffect(() => {
    if (expectedData?.list) {
      setRows(
        expectedData.list.map((item) => ({
          key: item.assetNo,
          assetNo: item.assetNo,
          name: item.name,
          bookLocation: item.bookLocation,
          actualLocation: '',
          notes: '',
          foundName: '',
          isSurplus: false,
          expectedUpdatedAt: item.expectedUpdatedAt ?? null,
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
      message.warning('请先修改实际位置、备注或添加盘盈行后再提交');
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

  const handleArchive = () => {
    Modal.confirm({
      title: '确认归档',
      content: '归档后将进入比对阶段，师生将无法继续提交草稿。确定归档吗？',
      okText: '归档',
      cancelText: '取消',
      onOk: async () => {
        try {
          await archiveTask({ id: task!.id }).unwrap();
          setPollComparing(true);
          message.success('已归档，正在比对…');
        } catch (err: unknown) {
          const e = err as { message?: string };
          message.error(e.message ?? '归档失败');
        }
      },
    });
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

  const errCode = getApiErrorCode(error);
  if (isError || !task) {
    if (basePath === '/user' && (errCode === 40302 || errCode === 40303)) {
      return (
        <Result
          status="403"
          title="无权参与该盘点任务"
          subTitle="您未被指派为该任务的盘点员"
          extra={
            <Button type="primary" onClick={() => navigate('/user/assets')}>
              返回我的资产
            </Button>
          }
        />
      );
    }
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

  if (assigneeDenied) {
    return (
      <Result
        status="403"
        title="无权参与该盘点任务"
        subTitle="您未被指派为该任务的盘点员"
        extra={
          <Button type="primary" onClick={() => navigate('/user/assets')}>
            返回我的资产
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
            <Button danger loading={archiving} onClick={handleArchive}>
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

      {task.status === 2 && (
        <Card>
          <div style={{ textAlign: 'center', padding: 48 }}>
            <Spin size="large" />
            <Typography.Paragraph style={{ marginTop: 16 }}>比对中，请稍候…</Typography.Paragraph>
          </div>
        </Card>
      )}

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

      {task.status === 1 && (
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
