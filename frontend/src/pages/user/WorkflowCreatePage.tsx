import { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Modal,
  Radio,
  Space,
  Table,
  message,
} from 'antd';
import { useNavigate, useSearchParams } from 'react-router-dom';
import type { Asset } from '@/types/api';
import { useGetAssetsQuery } from '@/store/api/assetApi';
import { useCreateRequestMutation } from '@/store/api/workflowApi';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import StatusTag from '@/components/common/StatusTag';
import {
  filterAssetsForWorkflowType,
  getAssetQueryForWorkflowType,
} from '@/utils/workflow';

const TYPE_OPTIONS = [
  { label: '领用', value: 1 },
  { label: '归还', value: 2 },
  { label: '报修', value: 3 },
  { label: '报废', value: 4 },
];

export default function WorkflowCreatePage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const user = useAppSelector(selectCurrentUser);
  const [form] = Form.useForm();
  const [pickerOpen, setPickerOpen] = useState(false);
  const [selectedAsset, setSelectedAsset] = useState<Asset | null>(null);

  const initialType = Number(searchParams.get('type')) || 1;
  const initialAssetId = Number(searchParams.get('assetId')) || undefined;

  const watchType = Form.useWatch('type', form) ?? initialType;
  const assetQuery = user ? getAssetQueryForWorkflowType(watchType, user.id) : {};

  const { data: assetsData, isLoading: assetsLoading } = useGetAssetsQuery(
    { page: 1, pageSize: 100, ...assetQuery },
    { skip: !user },
  );

  const filteredAssets = useMemo(() => {
    if (!assetsData?.list || !user) return [];
    return filterAssetsForWorkflowType(watchType, user.id, assetsData.list);
  }, [assetsData?.list, user, watchType]);

  const [createRequest, { isLoading }] = useCreateRequestMutation();

  useEffect(() => {
    form.setFieldsValue({ type: initialType });
  }, [form, initialType]);

  useEffect(() => {
    if (initialAssetId && filteredAssets.length) {
      const found = filteredAssets.find((a) => a.id === initialAssetId);
      if (found) {
        setSelectedAsset(found);
        form.setFieldValue('assetId', found.id);
      }
    }
  }, [initialAssetId, filteredAssets, form]);

  const onFinish = async (values: { type: number; reason: string }) => {
    if (!selectedAsset) {
      message.warning('请选择资产');
      return;
    }
    try {
      await createRequest({
        assetId: selectedAsset.id,
        type: values.type as 1 | 2 | 3 | 4,
        reason: values.reason,
      }).unwrap();
      message.success('申请已提交');
      navigate('/user/workflow/my');
    } catch (err: unknown) {
      const e = err as { code?: number; message?: string };
      if (e.code === 40902) {
        message.error('该资产已有审批中的工单，请选择其他资产');
      } else {
        message.error(e.message ?? '提交失败');
      }
    }
  };

  return (
    <Card title="新建申请">
      <Form form={form} layout="vertical" onFinish={onFinish} style={{ maxWidth: 560 }}>
        <Form.Item name="type" label="申请类型" rules={[{ required: true }]}>
          <Radio.Group
            optionType="button"
            options={TYPE_OPTIONS}
            onChange={() => {
              setSelectedAsset(null);
              form.setFieldValue('assetId', undefined);
            }}
          />
        </Form.Item>

        <Form.Item label="选择资产" required>
          <Space direction="vertical" style={{ width: '100%' }}>
            <Button onClick={() => setPickerOpen(true)}>选择资产</Button>
            {selectedAsset && (
              <Card size="small">
                <Descriptions column={1} size="small">
                  <Descriptions.Item label="编号">{selectedAsset.assetNo}</Descriptions.Item>
                  <Descriptions.Item label="名称">{selectedAsset.name}</Descriptions.Item>
                  <Descriptions.Item label="地点">{selectedAsset.location}</Descriptions.Item>
                  <Descriptions.Item label="状态">
                    <StatusTag type="asset" value={selectedAsset.status} />
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </Space>
        </Form.Item>

        <Form.Item
          name="reason"
          label="申请原因"
          rules={[
            { required: true, message: '请填写申请原因' },
            { max: 200, message: '最多 200 字' },
          ]}
        >
          <Input.TextArea rows={4} showCount maxLength={200} />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={isLoading}>
              提交申请
            </Button>
            <Button onClick={() => navigate('/user/workflow/my')}>取消</Button>
          </Space>
        </Form.Item>
      </Form>

      <Modal
        title="选择资产"
        open={pickerOpen}
        onCancel={() => setPickerOpen(false)}
        footer={null}
        width={720}
      >
        <Table
          rowKey="id"
          loading={assetsLoading}
          dataSource={filteredAssets}
          pagination={false}
          locale={{ emptyText: '当前类型下没有可选资产' }}
          columns={[
            { title: '编号', dataIndex: 'assetNo' },
            { title: '名称', dataIndex: 'name' },
            { title: '地点', dataIndex: 'location' },
            {
              title: '状态',
              dataIndex: 'status',
              render: (s: number) => <StatusTag type="asset" value={s} />,
            },
            {
              title: '操作',
              render: (_: unknown, record: Asset) => (
                <Button
                  type="link"
                  onClick={() => {
                    setSelectedAsset(record);
                    form.setFieldValue('assetId', record.id);
                    setPickerOpen(false);
                  }}
                >
                  选择
                </Button>
              ),
            },
          ]}
        />
      </Modal>
    </Card>
  );
}
