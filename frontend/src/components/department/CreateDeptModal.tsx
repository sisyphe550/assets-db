import { Form, Input, InputNumber, Modal, message } from 'antd';
import { useCreateDeptMutation } from '@/store/api/userApi';

interface CreateDeptModalProps {
  open: boolean;
  parentId: number | null;
  parentName?: string;
  onClose: () => void;
  onSuccess?: () => void;
}

export default function CreateDeptModal({
  open,
  parentId,
  parentName,
  onClose,
  onSuccess,
}: CreateDeptModalProps) {
  const [form] = Form.useForm();
  const [createDept, { isLoading }] = useCreateDeptMutation();

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: { deptName: string; deptCode: string; sortOrder?: number }) => {
    if (parentId === null) return;
    try {
      await createDept({
        parentId,
        deptName: values.deptName,
        deptCode: values.deptCode,
        sortOrder: values.sortOrder ?? 0,
      }).unwrap();
      message.success('部门创建成功');
      handleClose();
      onSuccess?.();
    } catch (err: unknown) {
      const e = err as { data?: { code?: number; message?: string }; message?: string };
      const code = e.data?.code;
      const msg = e.data?.message ?? e.message ?? '创建失败';
      if (code === 40903) {
        form.setFields([{ name: 'deptCode', errors: [msg] }]);
      } else {
        message.error(msg);
      }
    }
  };

  return (
    <Modal
      title={`新增子部门${parentName ? ` — ${parentName}` : ''}`}
      open={open}
      onCancel={handleClose}
      onOk={() => form.submit()}
      confirmLoading={isLoading}
      destroyOnClose
    >
      <Form form={form} layout="vertical" onFinish={onFinish} initialValues={{ sortOrder: 0 }}>
        <Form.Item
          name="deptName"
          label="部门名称"
          rules={[{ required: true, message: '请输入部门名称' }, { max: 50 }]}
        >
          <Input placeholder="机械工程学院" />
        </Form.Item>
        <Form.Item
          name="deptCode"
          label="部门代码"
          rules={[
            { required: true, message: '请输入部门代码' },
            { pattern: /^[A-Z]{2,10}$/, message: '2-10 位大写字母' },
          ]}
        >
          <Input placeholder="ME" />
        </Form.Item>
        <Form.Item name="sortOrder" label="排序" rules={[{ type: 'number', min: 0 }]}>
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
