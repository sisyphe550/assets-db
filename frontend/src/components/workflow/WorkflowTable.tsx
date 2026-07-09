import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Empty, Result } from 'antd';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { WorkflowListParams, WorkflowRequest } from '@/types/api';
import { useGetRequestsQuery } from '@/store/api/workflowApi';
import StatusTag from '@/components/common/StatusTag';
import WorkflowDetail from '@/components/workflow/WorkflowDetail';
import { formatDateTime } from '@/utils/format';

interface WorkflowTableProps {
  scope: 'my' | 'todo' | 'all';
  title: string;
  assetBasePath?: '/admin' | '/college' | '/user';
  emptyDescription?: string;
  showCreateLink?: boolean;
}

export default function WorkflowTable({
  scope,
  title,
  assetBasePath = '/admin',
  emptyDescription,
  showCreateLink = false,
}: WorkflowTableProps) {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [params, setParams] = useState<WorkflowListParams>({ page: 1, pageSize: 20, scope });
  const [detailId, setDetailId] = useState<number | null>(null);

  const { data, isLoading, isError, refetch } = useGetRequestsQuery(params);

  useEffect(() => {
    const highlight = searchParams.get('highlight');
    if (highlight) {
      const id = Number(highlight);
      if (id > 0) {
        setDetailId(id);
        searchParams.delete('highlight');
        setSearchParams(searchParams, { replace: true });
      }
    }
  }, [searchParams, setSearchParams]);

  const columns: ProColumns<WorkflowRequest>[] = useMemo(
    () => [
      {
        title: '工单号',
        dataIndex: 'id',
        width: 90,
        render: (_, r) => `#${r.id}`,
      },
      {
        title: '资产ID',
        dataIndex: 'assetId',
        width: 90,
        render: (_, r) =>
          assetBasePath === '/user' ? (
            r.assetId
          ) : (
            <Button
              type="link"
              size="small"
              onClick={() => navigate(`${assetBasePath}/assets/${r.assetId}`)}
            >
              {r.assetId}
            </Button>
          ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 80,
        render: (_, r) => <StatusTag type="workflowType" value={r.type} />,
      },
      {
        title: '申请原因',
        dataIndex: 'reason',
        width: 200,
        ellipsis: true,
      },
      {
        title: '申请人',
        dataIndex: 'requesterName',
        width: 100,
        render: (_, r) => r.requesterName ?? `#${r.requesterId}`,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (_, r) => <StatusTag type="workflow" value={r.status} />,
      },
      {
        title: '当前阶段',
        dataIndex: 'currentStage',
        width: 120,
        render: (_, r) => <StatusTag type="workflowStage" value={r.currentStage} />,
      },
      {
        title: '提交时间',
        dataIndex: 'createdAt',
        width: 160,
        render: (_, r) => formatDateTime(r.createdAt),
      },
      {
        title: '操作',
        width: 80,
        render: (_, r) => (
          <Button type="link" onClick={() => setDetailId(r.id)}>
            详情
          </Button>
        ),
      },
    ],
    [assetBasePath, navigate],
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

  const emptyText =
    emptyDescription ??
    (scope === 'todo' ? '暂无待审批工单' : scope === 'my' ? '暂无申请记录' : '暂无工单');

  return (
    <>
      <Card
        title={title}
        extra={
          showCreateLink ? (
            <Button type="primary" onClick={() => navigate('/user/workflow/create')}>
              新建申请
            </Button>
          ) : null
        }
      >
        {!isLoading && !data?.list?.length ? (
          <Empty description={emptyText}>
            {showCreateLink && (
              <Button type="primary" onClick={() => navigate('/user/workflow/create')}>
                新建申请
              </Button>
            )}
          </Empty>
        ) : (
          <ProTable<WorkflowRequest>
            rowKey="id"
            search={false}
            options={false}
            loading={isLoading}
            columns={columns}
            dataSource={data?.list ?? []}
            pagination={{
              current: data?.page ?? 1,
              pageSize: data?.pageSize ?? 20,
              total: data?.total ?? 0,
              onChange: (page, pageSize) => setParams((prev) => ({ ...prev, page, pageSize })),
            }}
            locale={{ emptyText }}
          />
        )}
      </Card>

      {detailId !== null && (
        <WorkflowDetail
          requestId={detailId}
          open
          assetBasePath={assetBasePath}
          onClose={() => setDetailId(null)}
        />
      )}
    </>
  );
}
