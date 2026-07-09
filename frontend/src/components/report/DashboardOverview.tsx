import { lazy, Suspense, useMemo } from 'react';
import { Card, Col, Empty, Result, Row, Spin } from 'antd';
import {
  DatabaseOutlined,
  AuditOutlined,
  InboxOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import type { WorkflowRequest } from '@/types/api';
import StatCard from '@/components/report/StatCard';
import ChartBox from '@/components/report/ChartBox';
import { useGetAssetsByCategoryQuery, useGetAssetsByDeptQuery } from '@/store/api/reportApi';
import { useGetRequestsQuery } from '@/store/api/workflowApi';
import { useGetDeptTreeQuery } from '@/store/api/userApi';
import StatusTag from '@/components/common/StatusTag';
import {
  CHART_COLORS,
  collectSubtreeIds,
  enrichDeptStats,
  filterDeptStatsByIds,
  findDeptNode,
  flattenDeptTree,
  formatWan,
} from '@/utils/report';
import { formatDateTime } from '@/utils/format';
import { WORKFLOW_TYPE_MAP } from '@/utils/constants';

const Pie = lazy(() => import('@ant-design/charts').then((m) => ({ default: m.Pie })));

interface DashboardOverviewProps {
  roleLevel: 1 | 2;
  basePath: '/admin' | '/college';
  departmentId?: number;
}

export default function DashboardOverview({
  roleLevel,
  basePath,
  departmentId,
}: DashboardOverviewProps) {
  const navigate = useNavigate();
  const restrictToSubtree = roleLevel === 2;
  const { data: deptTree } = useGetDeptTreeQuery();
  const { data: deptData, isLoading: deptLoading } = useGetAssetsByDeptQuery();
  const { data: categoryData, isLoading: categoryLoading } = useGetAssetsByCategoryQuery(
    restrictToSubtree && departmentId ? { departmentId } : undefined,
  );
  const { data: todoData, isLoading: todoLoading } = useGetRequestsQuery({
    page: 1,
    pageSize: 10,
    scope: 'todo',
    status: 1,
  });
  const { data: recentData, isLoading: recentLoading } = useGetRequestsQuery({
    page: 1,
    pageSize: 10,
    scope: 'all',
  });

  const deptMap = useMemo(() => {
    const map = new Map<number, string>();
    if (deptTree) {
      for (const item of flattenDeptTree(deptTree)) {
        map.set(item.id, item.name);
      }
    }
    return map;
  }, [deptTree]);

  const subtreeIds = useMemo(() => {
    if (!restrictToSubtree || !departmentId || !deptTree) return null;
    const root = findDeptNode(deptTree, departmentId);
    return root ? collectSubtreeIds(root) : [departmentId];
  }, [restrictToSubtree, departmentId, deptTree]);

  const deptItems = useMemo(() => {
    const enriched = enrichDeptStats(deptData?.items ?? [], deptMap);
    if (subtreeIds) {
      return filterDeptStatsByIds(enriched, subtreeIds);
    }
    return enriched;
  }, [deptData, deptMap, subtreeIds]);

  const totals = useMemo(
    () =>
      deptItems.reduce(
        (acc, item) => ({
          total: acc.total + item.totalCount,
          inStock: acc.inStock + item.inStockCount,
          inUse: acc.inUse + item.inUseCount,
          value: acc.value + item.totalValue,
        }),
        { total: 0, inStock: 0, inUse: 0, value: 0 },
      ),
    [deptItems],
  );

  const categoryItems = categoryData?.items ?? [];
  const pieData = categoryItems.map((item) => ({ type: item.category, value: item.count }));

  const workflowColumns: ProColumns<WorkflowRequest>[] = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '类型',
      dataIndex: 'type',
      width: 80,
      render: (_, r) => WORKFLOW_TYPE_MAP[r.type] ?? r.type,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (_, r) => <StatusTag type="workflow" value={r.status} />,
    },
    {
      title: '提交时间',
      dataIndex: 'createdAt',
      render: (_, r) => formatDateTime(r.createdAt),
    },
    {
      title: '操作',
      width: 80,
      render: (_, r) => (
        <a onClick={() => navigate(`${basePath}/workflow/todo?highlight=${r.id}`)}>查看</a>
      ),
    },
  ];

  const chartFallback = (
    <div style={{ height: 280, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Spin />
    </div>
  );

  return (
    <>
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="资产总数"
            value={totals.total}
            prefix={<DatabaseOutlined />}
            loading={deptLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="在库"
            value={totals.inStock}
            prefix={<InboxOutlined />}
            loading={deptLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="领用中"
            value={totals.inUse}
            prefix={<TeamOutlined />}
            loading={deptLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            title="待审批"
            value={todoData?.total ?? 0}
            prefix={<AuditOutlined />}
            loading={todoLoading}
          />
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card title="资产总价值" loading={deptLoading}>
            <Result
              icon={null}
              title={formatWan(totals.value)}
              subTitle="基于部门统计快照"
              style={{ padding: '12px 0' }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="按类别资产分布" loading={categoryLoading}>
            {pieData.length === 0 ? (
              <Empty description="暂无数据" />
            ) : (
              <ChartBox height={280}>
                {(width) => (
                  <Suspense fallback={chartFallback}>
                    <Pie
                      key={`dash-pie-${width}`}
                      width={width}
                      height={280}
                      data={pieData}
                      angleField="value"
                      colorField="type"
                      color={CHART_COLORS}
                      legend={{ position: 'bottom' }}
                      label={false}
                    />
                  </Suspense>
                )}
              </ChartBox>
            )}
          </Card>
        </Col>
      </Row>

      <Card title="最近工单" style={{ marginTop: 16 }} loading={recentLoading}>
        <ProTable<WorkflowRequest>
          rowKey="id"
          search={false}
          options={false}
          pagination={false}
          dataSource={recentData?.list ?? []}
          columns={workflowColumns}
        />
      </Card>
    </>
  );
}
