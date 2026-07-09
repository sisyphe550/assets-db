import { lazy, Suspense, useMemo, useState } from 'react';
import { Card, Col, Empty, Result, Row, Select, Spin, Tabs } from 'antd';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import type { CategoryStatItem, DeptStatItem, InventoryRecord } from '@/types/api';
import { useGetRecordsQuery, useGetTasksQuery } from '@/store/api/inventoryApi';
import {
  useGetAssetsByCategoryQuery,
  useGetAssetsByDeptQuery,
  useGetInventoryDiffQuery,
} from '@/store/api/reportApi';
import { useGetDeptTreeQuery } from '@/store/api/userApi';
import DiffSummary from '@/components/report/DiffSummary';
import ChartBox from '@/components/report/ChartBox';
import StatusTag from '@/components/common/StatusTag';
import {
  CHART_COLORS,
  enrichDeptStats,
  filterDeptStatsByIds,
  findDeptNode,
  collectSubtreeIds,
  flattenDeptTree,
} from '@/utils/report';
import { formatPrice } from '@/utils/format';

const Column = lazy(() => import('@ant-design/charts').then((m) => ({ default: m.Column })));
const Pie = lazy(() => import('@ant-design/charts').then((m) => ({ default: m.Pie })));

export type ReportTab = 'dept' | 'category' | 'diff';

interface ReportChartsProps {
  activeTab: ReportTab;
  onTabChange: (tab: ReportTab) => void;
  departmentId?: number;
  restrictToSubtree?: boolean;
}

const chartFallback = (
  <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
    <Spin />
  </div>
);

export default function ReportCharts({
  activeTab,
  onTabChange,
  departmentId,
  restrictToSubtree,
}: ReportChartsProps) {
  const [taskId, setTaskId] = useState<number | undefined>();
  const { data: deptTree } = useGetDeptTreeQuery();
  const { data: deptData, isLoading: deptLoading, isError: deptError } = useGetAssetsByDeptQuery();
  const { data: categoryData, isLoading: categoryLoading } = useGetAssetsByCategoryQuery(
    restrictToSubtree && departmentId ? { departmentId } : undefined,
  );
  const { data: tasksData } = useGetTasksQuery({ page: 1, pageSize: 100, status: 3 });
  const { data: diffData, isLoading: diffLoading } = useGetInventoryDiffQuery(taskId!, {
    skip: !taskId || activeTab !== 'diff',
  });
  const { data: recordsData, isLoading: recordsLoading } = useGetRecordsQuery(
    { taskId: taskId!, page: 1, pageSize: 100 },
    { skip: !taskId || activeTab !== 'diff' },
  );

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

  const deptItems: DeptStatItem[] = useMemo(() => {
    const enriched = enrichDeptStats(deptData?.items ?? [], deptMap);
    if (subtreeIds) {
      return filterDeptStatsByIds(enriched, subtreeIds);
    }
    return enriched;
  }, [deptData, deptMap, subtreeIds]);

  const categoryItems: CategoryStatItem[] = categoryData?.items ?? [];

  const diffRecords = useMemo(
    () => (recordsData?.list ?? []).filter((r) => r.diffStatus === 2 || r.diffStatus === 3),
    [recordsData?.list],
  );

  const deptColumns: ProColumns<DeptStatItem>[] = [
    { title: '部门', dataIndex: 'departmentName', width: 180 },
    { title: '总数', dataIndex: 'totalCount', width: 80 },
    { title: '在库', dataIndex: 'inStockCount', width: 80 },
    { title: '领用中', dataIndex: 'inUseCount', width: 80 },
    {
      title: '总价值',
      dataIndex: 'totalValue',
      render: (_, r) => formatPrice(r.totalValue),
    },
  ];

  const categoryColumns: ProColumns<CategoryStatItem>[] = [
    { title: '类别', dataIndex: 'category', width: 120 },
    { title: '数量', dataIndex: 'count', width: 80 },
    {
      title: '总价值',
      dataIndex: 'totalValue',
      render: (_, r) => formatPrice(r.totalValue),
    },
  ];

  const diffRecordColumns: ProColumns<InventoryRecord>[] = [
    { title: '资产编号', dataIndex: 'assetNo', width: 160 },
    { title: '名称', dataIndex: 'name', width: 140 },
    { title: '账面位置', dataIndex: 'bookLocation', width: 140 },
    { title: '实际位置', dataIndex: 'actualLocation', width: 140 },
    {
      title: '差异',
      dataIndex: 'diffStatus',
      width: 100,
      render: (_, r) => <StatusTag type="inventoryDiff" value={r.diffStatus} />,
    },
  ];

  const deptChartData = deptItems.map((item) => ({
    dept: item.departmentName ?? `#${item.departmentId}`,
    count: item.totalCount,
  }));

  const categoryChartData = categoryItems.map((item) => ({
    type: item.category,
    value: item.count,
  }));

  const tabItems = [
    {
      key: 'dept',
      label: '按部门',
      children: deptError ? (
        <Result status="error" title="部门统计加载失败" />
      ) : (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={10}>
            <Card title="各学院资产数量" loading={deptLoading}>
              {deptChartData.length === 0 ? (
                <Empty description="暂无数据" />
              ) : (
                <ChartBox height={300}>
                  {(width) => (
                    <Suspense fallback={chartFallback}>
                      <Column
                        key={`dept-col-${width}`}
                        width={width}
                        height={300}
                        data={deptChartData}
                        xField="dept"
                        yField="count"
                        color="#1677FF"
                        animation={{ appear: { animation: 'fade-in', duration: 500 } }}
                        legend={false}
                        axis={{ x: { labelAutoRotate: true } }}
                      />
                    </Suspense>
                  )}
                </ChartBox>
              )}
            </Card>
          </Col>
          <Col xs={24} lg={14}>
            <Card title="部门资产明细" styles={{ body: { paddingTop: 0 } }}>
              <ProTable<DeptStatItem>
                rowKey="departmentId"
                search={false}
                options={false}
                pagination={false}
                loading={deptLoading}
                dataSource={deptItems}
                columns={deptColumns}
                scroll={{ x: 520 }}
              />
            </Card>
          </Col>
        </Row>
      ),
    },
    {
      key: 'category',
      label: '按类别',
      children: (
        <Row gutter={[16, 16]}>
          <Col xs={24} lg={10}>
            <Card title="资产类型分布" loading={categoryLoading}>
              {categoryChartData.length === 0 ? (
                <Empty description="暂无数据" />
              ) : (
                <ChartBox height={300}>
                  {(width) => (
                    <Suspense fallback={chartFallback}>
                      <Pie
                        key={`cat-pie-${width}`}
                        width={width}
                        height={300}
                        data={categoryChartData}
                        angleField="value"
                        colorField="type"
                        color={CHART_COLORS}
                        animation={{ appear: { animation: 'fade-in', duration: 500 } }}
                        legend={{ position: 'bottom' }}
                        label={false}
                      />
                    </Suspense>
                  )}
                </ChartBox>
              )}
            </Card>
          </Col>
          <Col xs={24} lg={14}>
            <Card title="类别明细" styles={{ body: { paddingTop: 0 } }}>
              <ProTable<CategoryStatItem>
                rowKey="category"
                search={false}
                options={false}
                pagination={false}
                loading={categoryLoading}
                dataSource={categoryItems}
                columns={categoryColumns}
              />
            </Card>
          </Col>
        </Row>
      ),
    },
    {
      key: 'diff',
      label: '盘点差异',
      children: (
        <Card>
          <Select
            placeholder="选择已完成的盘点任务"
            style={{ width: 320, marginBottom: 16 }}
            value={taskId}
            onChange={setTaskId}
            options={(tasksData?.list ?? []).map((t) => ({
              value: t.id,
              label: t.taskName,
            }))}
            allowClear
          />
          {!taskId ? (
            <Empty description="请选择盘点任务" />
          ) : (
            <>
              <DiffSummary
                matchCount={diffData?.match ?? 0}
                surplusCount={diffData?.surplus ?? 0}
                deficitCount={diffData?.loss ?? 0}
                loading={diffLoading}
              />
              <Card type="inner" title="差异明细" style={{ marginTop: 16 }}>
                <ProTable<InventoryRecord>
                  rowKey="assetNo"
                  search={false}
                  options={false}
                  pagination={false}
                  loading={recordsLoading}
                  dataSource={diffRecords}
                  columns={diffRecordColumns}
                  scroll={{ x: 720 }}
                />
              </Card>
            </>
          )}
        </Card>
      ),
    },
  ];

  return (
    <Tabs
      activeKey={activeTab}
      onChange={(key) => onTabChange(key as ReportTab)}
      destroyInactiveTabPane
      items={tabItems}
    />
  );
}
