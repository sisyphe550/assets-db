import { useState } from 'react';
import { Button, Card, Empty, Result, Space, Spin, Table, Tabs, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import type { Asset } from '@/types/api';
import { useGetAssetsQuery, useGetSharedAssetsQuery } from '@/store/api/assetApi';
import StatusTag from '@/components/common/StatusTag';
import { formatPrice } from '@/utils/format';

export default function MyAssetsPage() {
  const navigate = useNavigate();
  const [page, setPage] = useState(1);
  const [sharedPage, setSharedPage] = useState(1);
  const pageSize = 20;

  const {
    data: myData,
    isLoading: myLoading,
    isError: myError,
    refetch: refetchMy,
  } = useGetAssetsQuery({ page, pageSize, scope: 'my' });

  const {
    data: sharedData,
    isLoading: sharedLoading,
    isError: sharedError,
    refetch: refetchShared,
  } = useGetSharedAssetsQuery({ page: sharedPage, pageSize });

  const columns = [
    {
      title: '资产编号',
      dataIndex: 'assetNo',
      render: (v: string) => <Typography.Text copyable>{v}</Typography.Text>,
    },
    { title: '名称', dataIndex: 'name' },
    { title: '地点', dataIndex: 'location' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status: number) => <StatusTag type="asset" value={status} />,
    },
    {
      title: '价格',
      dataIndex: 'price',
      render: (price: number) => formatPrice(price),
    },
    {
      title: '操作',
      render: (_: unknown, record: Asset) => (
        <Space>
          {record.status === 2 && (
            <>
              <Button
                type="link"
                size="small"
                onClick={() =>
                  navigate(`/user/workflow/create?type=2&assetId=${record.id}`)
                }
              >
                申请归还
              </Button>
              <Button
                type="link"
                size="small"
                onClick={() =>
                  navigate(`/user/workflow/create?type=3&assetId=${record.id}`)
                }
              >
                申请报修
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  const sharedColumns = [
    {
      title: '资产编号',
      dataIndex: 'assetNo',
      render: (v: string) => <Typography.Text copyable>{v}</Typography.Text>,
    },
    { title: '名称', dataIndex: 'name' },
    { title: '地点', dataIndex: 'location' },
    {
      title: '状态',
      dataIndex: 'status',
      render: (status: number) => <StatusTag type="asset" value={status} />,
    },
  ];

  const renderTable = (
    loading: boolean,
    isError: boolean,
    refetch: () => void,
    data: typeof myData,
    currentPage: number,
    onPageChange: (p: number) => void,
    cols: typeof columns,
  ) => {
    if (loading) return <Spin style={{ display: 'block', margin: '40px auto' }} />;
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
    if (!data?.list?.length) return <Empty description="暂无数据" />;
    return (
      <Table
        rowKey="id"
        columns={cols}
        dataSource={data.list}
        pagination={{
          current: currentPage,
          pageSize,
          total: data.total,
          onChange: onPageChange,
        }}
      />
    );
  };

  return (
    <Card title="我的资产">
      <Tabs
        items={[
          {
            key: 'my',
            label: '已领用',
            children: renderTable(
              myLoading,
              myError,
              refetchMy,
              myData,
              page,
              setPage,
              columns,
            ),
          },
          {
            key: 'shared',
            label: '学院共享',
            children: renderTable(
              sharedLoading,
              sharedError,
              refetchShared,
              sharedData,
              sharedPage,
              setSharedPage,
              sharedColumns,
            ),
          },
        ]}
      />
    </Card>
  );
}
