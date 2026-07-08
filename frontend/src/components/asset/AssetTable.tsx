import { useMemo, useState } from 'react';
import { Button, Card, Input, Result, Select, Space, Typography } from 'antd';
import { ProTable, type ProColumns } from '@ant-design/pro-components';
import { PlusOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { Asset, AssetListParams } from '@/types/api';
import { useGetAssetsQuery } from '@/store/api/assetApi';
import { ASSET_CATEGORIES } from '@/utils/constants';
import { formatPrice } from '@/utils/format';
import StatusTag from '@/components/common/StatusTag';

interface AssetTableProps {
  basePath: '/admin' | '/college';
  roleLevel: 1 | 2;
  showCreate?: boolean;
}

export default function AssetTable({ basePath, roleLevel, showCreate = true }: AssetTableProps) {
  const navigate = useNavigate();
  const [params, setParams] = useState<AssetListParams>({ page: 1, pageSize: 20 });
  const [keyword, setKeyword] = useState('');
  const { data, isLoading, isError, refetch } = useGetAssetsQuery(params);

  const columns: ProColumns<Asset>[] = useMemo(
    () => [
      {
        title: '资产编号',
        dataIndex: 'assetNo',
        width: 180,
        render: (_, record) => <Typography.Text copyable>{record.assetNo}</Typography.Text>,
      },
      { title: '名称', dataIndex: 'name', width: 150 },
      {
        title: '类别',
        dataIndex: 'category',
        width: 100,
        render: (v) => <Typography.Text>{v}</Typography.Text>,
      },
      {
        title: '价格',
        dataIndex: 'price',
        width: 120,
        render: (_, record) => formatPrice(record.price),
      },
      { title: '地点', dataIndex: 'location', width: 150, ellipsis: true },
      ...(roleLevel === 1
        ? [{ title: '部门ID', dataIndex: 'departmentId', width: 90 } satisfies ProColumns<Asset>]
        : []),
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        render: (_, record) => <StatusTag type="asset" value={record.status} />,
      },
      {
        title: '领用人ID',
        dataIndex: 'userId',
        width: 100,
        render: (_, record) => record.userId ?? '-',
      },
      {
        title: '操作',
        width: 120,
        render: (_, record) => (
          <Button type="link" onClick={() => navigate(`${basePath}/assets/${record.id}`)}>
            详情
          </Button>
        ),
      },
    ],
    [basePath, navigate, roleLevel],
  );

  const onSearch = () => {
    setParams((prev) => ({ ...prev, page: 1, keyword: keyword || undefined }));
  };

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
      title="资产管理"
      extra={
        showCreate && roleLevel <= 2 ? (
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate(`${basePath}/assets/create`)}
          >
            新增资产
          </Button>
        ) : null
      }
    >
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          allowClear
          placeholder="类别"
          style={{ width: 140 }}
          options={ASSET_CATEGORIES.map((c) => ({ label: c, value: c }))}
          onChange={(category) => setParams((prev) => ({ ...prev, page: 1, category }))}
        />
        <Select
          allowClear
          placeholder="状态"
          style={{ width: 120 }}
          options={[
            { label: '在库', value: 1 },
            { label: '领用中', value: 2 },
            { label: '维修中', value: 3 },
            { label: '已报废', value: 4 },
          ]}
          onChange={(status) =>
            setParams((prev) => ({ ...prev, page: 1, status: status as number | undefined }))
          }
        />
        <Input.Search
          placeholder="编号或名称"
          allowClear
          style={{ width: 220 }}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onSearch={onSearch}
        />
      </Space>

      <ProTable<Asset>
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
          showSizeChanger: true,
          onChange: (page, pageSize) => setParams((prev) => ({ ...prev, page, pageSize })),
        }}
        locale={{ emptyText: '暂无资产数据' }}
      />
    </Card>
  );
}
