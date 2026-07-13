import { useMemo, useState } from 'react';
import { Button, Card, Empty, Modal, Select, Space, Table, Typography, message } from 'antd';
import { DeleteOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons';
import {
  useGetTaskItemsQuery,
  usePublishTaskMutation,
  useUpdateTaskItemsMutation,
} from '@/store/api/inventoryApi';
import type { ExpectedAsset } from '@/types/api';

interface InventoryTaskItemConfiguratorProps {
  taskId: number;
  onPublished?: () => void;
}

export default function InventoryTaskItemConfigurator({
  taskId,
  onPublished,
}: InventoryTaskItemConfiguratorProps) {
  const { data, isLoading, refetch } = useGetTaskItemsQuery(taskId);
  const [updateTaskItems, { isLoading: saving }] = useUpdateTaskItemsMutation();
  const [publishTask, { isLoading: publishing }] = usePublishTaskMutation();
  const [modalOpen, setModalOpen] = useState(false);
  const [selectedAssetIds, setSelectedAssetIds] = useState<number[]>([]);

  const selectedItems = data?.list ?? [];
  const availableItems = data?.available ?? [];
  const selectedIdSet = useMemo(
    () => new Set(selectedItems.map((item) => item.assetId)),
    [selectedItems],
  );

  const assetOptions = useMemo(
    () =>
      availableItems.map((item) => ({
        label: `${item.assetNo} - ${item.name}`,
        value: item.assetId,
        disabled: selectedIdSet.has(item.assetId),
      })),
    [availableItems, selectedIdSet],
  );

  const saveAssetIds = async (assetIds: number[]) => {
    await updateTaskItems({ taskId, assetIds }).unwrap();
    setSelectedAssetIds([]);
    setModalOpen(false);
    refetch();
  };

  const handleAdd = async () => {
    if (!selectedAssetIds.length) {
      message.warning('请选择盘点条目');
      return;
    }
    const next = Array.from(
      new Set([...selectedItems.map((item) => item.assetId), ...selectedAssetIds]),
    );
    try {
      await saveAssetIds(next);
      message.success('盘点条目已更新');
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '保存盘点条目失败');
    }
  };

  const handleDelete = async (assetId: number) => {
    const next = selectedItems
      .filter((item) => item.assetId !== assetId)
      .map((item) => item.assetId);
    try {
      await saveAssetIds(next);
      message.success('盘点条目已删除');
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '删除盘点条目失败');
    }
  };

  const handlePublish = () => {
    if (!selectedItems.length) {
      message.warning('请先添加盘点条目');
      return;
    }
    Modal.confirm({
      title: '发布盘点任务',
      content: '发布后盘点员将看到这些盘点条目，任务进入进行中状态。',
      okText: '发布',
      cancelText: '取消',
      onOk: async () => {
        try {
          await publishTask(taskId).unwrap();
          message.success('盘点任务已发布');
          onPublished?.();
        } catch (err: unknown) {
          const e = err as { message?: string };
          message.error(e.message ?? '发布失败');
        }
      },
    });
  };

  return (
    <Card
      title="盘点条目配置"
      extra={
        <Space>
          <Button icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
            添加盘点条目
          </Button>
          <Button
            type="primary"
            icon={<SendOutlined />}
            loading={publishing}
            onClick={handlePublish}
          >
            发布任务
          </Button>
        </Space>
      }
    >
      <Typography.Paragraph type="secondary">
        已选择 {selectedItems.length} 条盘点资产
      </Typography.Paragraph>
      <Table<ExpectedAsset>
        rowKey="assetId"
        loading={isLoading}
        dataSource={selectedItems}
        pagination={false}
        locale={{ emptyText: <Empty description="暂无盘点条目" /> }}
        columns={[
          { title: '资产编号', dataIndex: 'assetNo', width: 180 },
          { title: '资产名称', dataIndex: 'name', width: 220 },
          { title: '账面位置', dataIndex: 'bookLocation' },
          {
            title: '操作',
            width: 90,
            render: (_, record) => (
              <Button
                type="link"
                danger
                icon={<DeleteOutlined />}
                loading={saving}
                onClick={() => handleDelete(record.assetId)}
              >
                删除
              </Button>
            ),
          },
        ]}
      />
      <Modal
        title="添加盘点条目"
        open={modalOpen}
        onOk={handleAdd}
        confirmLoading={saving}
        onCancel={() => {
          setModalOpen(false);
          setSelectedAssetIds([]);
        }}
        okText="添加"
        cancelText="取消"
      >
        <Select
          mode="multiple"
          showSearch
          optionFilterProp="label"
          value={selectedAssetIds}
          options={assetOptions}
          style={{ width: '100%' }}
          placeholder="选择资产编号或资产名称"
          onChange={setSelectedAssetIds}
        />
      </Modal>
    </Card>
  );
}
