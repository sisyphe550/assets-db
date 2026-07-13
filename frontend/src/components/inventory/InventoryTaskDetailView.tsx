import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from 'react';
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
import InventoryConflictPanel from '@/components/inventory/InventoryConflictPanel';
import {
  useArchiveTaskMutation,
  useCompareTaskMutation,
  useGetConflictsQuery,
  useGetExpectedAssetsQuery,
  useGetDraftsQuery,
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
  buildRowsFromExpected,
  buildSubmitItems,
  mergeDraftTimestamps,
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
  const [compareError, setCompareError] = useState<string | null>(null);
  const compareTriggered = useRef(false);
  const hydratedRef = useRef(false);
  const [rowsReady, setRowsReady] = useState(false);
  const [recordsPage, setRecordsPage] = useState(1);
  const [recordsPageSize, setRecordsPageSize] = useState(50);

  const { data: task, isLoading, isError, error, refetch } = useGetTaskQuery(taskIdNum, {
    skip: !taskIdNum,
    pollingInterval: pollComparing ? 2000 : 0,
  });
  const { data: expectedData, isLoading: expectedLoading } = useGetExpectedAssetsQuery(
    taskIdNum,
    { skip: !taskIdNum || task?.status !== 1, refetchOnFocus: false },
  );
  const {
    data: draftsData,
    isLoading: draftsLoading,
    refetch: refetchDrafts,
  } = useGetDraftsQuery(taskIdNum, {
    skip: !taskIdNum || task?.status !== 1,
    refetchOnFocus: false,
  });
  const { data: recordsData } = useGetRecordsQuery(
    { taskId: taskIdNum, page: recordsPage, pageSize: recordsPageSize },
    { skip: !taskIdNum || task?.status !== 3 },
  );
  const shouldLoadAdminConflicts = Boolean(
    taskIdNum && showArchive && task?.status === 2,
  );
  const {
    data: conflictsData,
    isLoading: conflictsLoading,
    isError: conflictsError,
    refetch: refetchConflicts,
  } = useGetConflictsQuery(taskIdNum, {
    skip: !shouldLoadAdminConflicts,
  });

  const [rows, setRows] = useState<SpreadsheetRow[]>([]);
  const [submitRecords, { isLoading: submitting }] = useSubmitRecordsMutation();
  const [archiveTask, { isLoading: archiving }] = useArchiveTaskMutation();
  const [compareTask, { isLoading: comparing }] = useCompareTaskMutation();

  // 冲突列表接口是比对阶段的权威来源。任务详情中的计数可能因缓存、
  // 旧服务版本或短暂查询失败而滞后，不能用它作为是否请求冲突的门禁。
  const pendingConflicts = showArchive
    ? (conflictsData?.pendingCount ?? task?.pendingConflictCount ?? 0)
    : (task?.pendingConflictCount ?? 0);
  const adminConflictsReady =
    !showArchive || (!conflictsLoading && !conflictsError && conflictsData !== undefined);

  const triggerCompare = useCallback(() => {
    if (!taskIdNum) return;
    compareTriggered.current = true;
    setCompareError(null);
    setPollComparing(true);
    compareTask(taskIdNum)
      .unwrap()
      .then(() => refetch())
      .catch((err: unknown) => {
        compareTriggered.current = false;
        setPollComparing(false);
        const msg = err instanceof Error ? err.message : '比对失败，请稍后重试';
        setCompareError(msg);
      });
  }, [taskIdNum, compareTask, refetch]);

  useEffect(() => {
    if (task?.status !== 2 || compareTriggered.current) return;
    if (!adminConflictsReady || pendingConflicts > 0) return;
    triggerCompare();
  }, [task?.status, adminConflictsReady, pendingConflicts, taskIdNum, triggerCompare]);

  useEffect(() => {
    if (task?.status === 3) {
      setPollComparing(false);
      compareTriggered.current = false;
      setCompareError(null);
    }
  }, [task?.status]);

  const readOnly = !task || task.status !== 1;

  const assigneeDenied = useMemo(() => {
    if (basePath !== '/user' || !user || !task) return false;
    return !task.assigneeIds.includes(user.id);
  }, [basePath, user, task]);

  useEffect(() => {
    hydratedRef.current = false;
    setRows([]);
    setRowsReady(false);
    compareTriggered.current = false;
    setPollComparing(false);
    setCompareError(null);
    setRecordsPage(1);
  }, [taskIdNum]);

  useEffect(() => {
    if (!taskIdNum || task?.status !== 1) return;
    if (expectedLoading || draftsLoading) return;
    if (expectedData === undefined) return;
    if (hydratedRef.current) return;

    setRows(
      buildRowsFromExpected(expectedData.list ?? [], draftsData?.list ?? [], user?.id),
    );
    setRowsReady(true);
    hydratedRef.current = true;
  }, [
    taskIdNum,
    task?.status,
    expectedData,
    draftsData?.list,
    expectedLoading,
    draftsLoading,
    user?.id,
  ]);

  const conflictMessages = useMemo(
    () => [
      ...new Set(
        rows.filter((r) => r.rowState === 'conflict').map((r) => r.rowMessage).filter(Boolean),
      ),
    ],
    [rows],
  );

  const handleSubmit = async () => {
    const items = buildSubmitItems(rows);
    if (!items.length) {
      message.warning('请先修改实际位置、备注或填写盘盈行信息后再提交');
      return;
    }
    const invalidSurplus = items.find((item) => {
      const row = rows.find((r) => r.assetNo.trim() === item.assetNo);
      return (
        row?.isSurplus &&
        !item.modifiedCells.found_name?.trim() &&
        !item.modifiedCells.actual_location?.trim()
      );
    });
    if (invalidSurplus) {
      message.warning('盘盈行请至少填写资产名称或实际位置');
      return;
    }
    try {
      const result = await submitRecords({ taskId: taskIdNum, items }).unwrap();
      setRows((prev) => applySubmitResult(prev, result));
      const { data: freshDrafts } = await refetchDrafts();
      if (freshDrafts?.list) {
        setRows((prev) => mergeDraftTimestamps(prev, freshDrafts.list, user?.id));
      }
      const conflicts = result.conflicts ?? [];
      const failures = result.failures ?? [];
      const success = result.success ?? [];
      const total = items.length;
      if (conflicts.length) {
        message.error(`有 ${conflicts.length} 条冲突，请检查标红行`);
      } else if (failures.length) {
        const detail = failures.map((f) => f.message).join('；');
        message.warning(`有 ${failures.length} 条提交失败：${detail}`);
      } else if (success.length < total) {
        message.warning(`成功提交 ${success.length}/${total} 条，请检查未保存的行`);
      } else {
        message.success(`成功提交 ${success.length} 条`);
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
          const result = await archiveTask({ id: task!.id }).unwrap();
          compareTriggered.current = false;
          setCompareError(null);
          if ((result.pendingConflictCount ?? 0) > 0) {
            setPollComparing(false);
            message.warning(
              `已归档，有 ${result.pendingConflictCount} 条盘点员冲突待裁决`,
            );
          } else {
            setPollComparing(true);
            message.success('已归档，正在比对…');
          }
          refetch();
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
        bookLocation: '',
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

  const hasSpreadsheetRows = rows.length > 0;

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
        <>
          {showArchive && (
            <InventoryConflictPanel
              taskId={taskIdNum}
              onResolved={() => {
                refetchConflicts();
                refetch();
              }}
            />
          )}
          {showArchive && conflictsError ? (
            <Card>
              <Result
                status="error"
                title="裁决条目加载失败"
                subTitle="无法确认是否存在待裁决冲突，系统不会自动进入下一阶段。"
                extra={
                  <Button type="primary" onClick={() => refetchConflicts()}>
                    重新加载
                  </Button>
                }
              />
            </Card>
          ) : showArchive && (!adminConflictsReady || pendingConflicts > 0) ? null : compareError ? (
            <Card>
              <Result
                status="error"
                title="比对失败"
                subTitle={compareError}
                extra={
                  <Button type="primary" loading={comparing} onClick={triggerCompare}>
                    重新比对
                  </Button>
                }
              />
            </Card>
          ) : pendingConflicts > 0 && !showArchive ? (
            <Card>
              <Typography.Paragraph style={{ textAlign: 'center', padding: 48 }}>
                存在盘点员冲突，等待管理员裁决…
              </Typography.Paragraph>
            </Card>
          ) : (
            <Card>
              <div style={{ textAlign: 'center', padding: 48 }}>
                <Spin size="large" />
                <Typography.Paragraph style={{ marginTop: 16 }}>
                  比对中，请稍候…
                </Typography.Paragraph>
              </div>
            </Card>
          )}
        </>
      )}

      {task.status === 3 && (
        <Card title="差异报告">
          <Table
            rowKey="assetNo"
            dataSource={recordsData?.list ?? []}
            pagination={{
              current: recordsPage,
              pageSize: recordsPageSize,
              total: recordsData?.total ?? 0,
              showSizeChanger: true,
              onChange: (page, pageSize) => {
                setRecordsPage(page);
                setRecordsPageSize(pageSize);
              },
            }}
            locale={{ emptyText: '暂无差异记录' }}
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
      )}

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
          {expectedLoading || draftsLoading || !rowsReady ? (
            <Skeleton active paragraph={{ rows: 10 }} />
          ) : !hasSpreadsheetRows ? (
            <Empty description="暂无应盘资产，可添加盘盈行后提交" />
          ) : (
            <Suspense fallback={<Skeleton active paragraph={{ rows: 10 }} />}>
              <InventorySpreadsheet
                key={taskIdNum}
                rows={rows}
                readOnly={readOnly}
                onChange={setRows}
              />
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
