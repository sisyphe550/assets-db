import { useEffect, useMemo } from 'react';
import { Form, Input, InputNumber, Select, Switch, DatePicker, Button, Space } from 'antd';
import dayjs from 'dayjs';
import type { Asset, CreateAssetReq } from '@/types/api';
import { ASSET_CATEGORIES } from '@/utils/constants';
import { useGetDeptTreeQuery } from '@/store/api/userApi';

export interface AssetFormValues {
  assetNo: string;
  name: string;
  category: string;
  price: number;
  purchaseTime: dayjs.Dayjs;
  location: string;
  departmentId: number;
  isShared: boolean;
}

interface AssetFormProps {
  mode: 'create' | 'edit';
  roleLevel: 1 | 2;
  defaultDepartmentId?: number;
  initialAsset?: Asset;
  loading?: boolean;
  onSubmit: (values: CreateAssetReq) => Promise<void>;
  onCancel: () => void;
}

function flattenDeptTree(
  nodes: { id: number; deptName: string; children: typeof nodes | null }[],
): { label: string; value: number }[] {
  const result: { label: string; value: number }[] = [];
  const walk = (list: typeof nodes, prefix = '') => {
    for (const node of list) {
      const label = prefix ? `${prefix} / ${node.deptName}` : node.deptName;
      result.push({ label, value: node.id });
      if (node.children?.length) walk(node.children, label);
    }
  };
  walk(nodes);
  return result;
}

export default function AssetForm({
  mode,
  roleLevel,
  defaultDepartmentId,
  initialAsset,
  loading,
  onSubmit,
  onCancel,
}: AssetFormProps) {
  const [form] = Form.useForm<AssetFormValues>();
  const { data: deptTree } = useGetDeptTreeQuery(undefined, { skip: roleLevel !== 1 });

  const deptOptions = useMemo(
    () => (deptTree ? flattenDeptTree(deptTree) : []),
    [deptTree],
  );

  useEffect(() => {
    if (initialAsset) {
      form.setFieldsValue({
        assetNo: initialAsset.assetNo,
        name: initialAsset.name,
        category: initialAsset.category,
        price: initialAsset.price,
        purchaseTime: dayjs(initialAsset.purchaseTime),
        location: initialAsset.location,
        departmentId: initialAsset.departmentId,
        isShared: initialAsset.isShared === 1,
      });
    } else if (defaultDepartmentId) {
      form.setFieldValue('departmentId', defaultDepartmentId);
    }
  }, [initialAsset, defaultDepartmentId, form]);

  const handleFinish = async (values: AssetFormValues) => {
    const payload: CreateAssetReq = {
      assetNo: values.assetNo,
      name: values.name,
      category: values.category,
      price: values.price,
      purchaseTime: values.purchaseTime.format('YYYY-MM-DD'),
      location: values.location,
      departmentId: values.departmentId,
      isShared: values.isShared ? 1 : 0,
    };
    await onSubmit(payload);
  };

  return (
    <Form form={form} layout="vertical" onFinish={handleFinish}>
      <Form.Item
        name="assetNo"
        label="资产编号"
        rules={[
          { required: true, message: '请输入资产编号' },
          { pattern: /^[A-Z]+-\d{4}-\d{4}$/, message: '格式如 EQUIP-2026-0001' },
        ]}
      >
        <Input disabled={mode === 'edit'} placeholder="EQUIP-2026-0001" />
      </Form.Item>
      <Form.Item name="name" label="名称" rules={[{ required: true, message: '请输入名称' }, { max: 50 }]}>
        <Input />
      </Form.Item>
      <Form.Item name="category" label="类别" rules={[{ required: true, message: '请选择类别' }]}>
        <Select options={ASSET_CATEGORIES.map((c) => ({ label: c, value: c }))} />
      </Form.Item>
      <Form.Item name="price" label="价格" rules={[{ required: true, message: '请输入价格' }]}>
        <InputNumber min={0} precision={2} prefix="¥" style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        name="purchaseTime"
        label="购置日期"
        rules={[{ required: true, message: '请选择购置日期' }]}
      >
        <DatePicker style={{ width: '100%' }} />
      </Form.Item>
      <Form.Item
        name="location"
        label="存放地点"
        rules={[{ required: true, message: '请输入存放地点' }, { max: 100 }]}
      >
        <Input />
      </Form.Item>
      {roleLevel === 1 ? (
        <Form.Item
          name="departmentId"
          label="所属部门"
          rules={[{ required: true, message: '请选择部门' }]}
        >
          <Select options={deptOptions} showSearch optionFilterProp="label" />
        </Form.Item>
      ) : (
        <Form.Item name="departmentId" hidden>
          <InputNumber />
        </Form.Item>
      )}
      <Form.Item name="isShared" label="学院共享" valuePropName="checked">
        <Switch />
      </Form.Item>
      <Form.Item>
        <Space>
          <Button type="primary" htmlType="submit" loading={loading}>
            {mode === 'create' ? '创建' : '保存'}
          </Button>
          <Button onClick={onCancel}>取消</Button>
        </Space>
      </Form.Item>
    </Form>
  );
}
