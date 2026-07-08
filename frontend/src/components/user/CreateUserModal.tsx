import { Form, Input, Modal, Select, message } from 'antd';
import type { UserInfo } from '@/types/api';
import { useCreateUserMutation } from '@/store/api/userApi';
import DeptTreeSelect from '@/components/department/DeptTreeSelect';
import { ROLE_MAP } from '@/utils/constants';

interface CreateUserModalProps {
  open: boolean;
  onClose: () => void;
  onSuccess?: (user: UserInfo) => void;
  roleLevel: 1 | 2;
  restrictDeptId?: number;
}

const ROLE_OPTIONS_ADMIN = [
  { value: 1, label: ROLE_MAP[1] },
  { value: 2, label: ROLE_MAP[2] },
  { value: 3, label: ROLE_MAP[3] },
];

const ROLE_OPTIONS_COLLEGE = [{ value: 3, label: ROLE_MAP[3] }];

export default function CreateUserModal({
  open,
  onClose,
  onSuccess,
  roleLevel,
  restrictDeptId,
}: CreateUserModalProps) {
  const [form] = Form.useForm();
  const [createUser, { isLoading }] = useCreateUserMutation();

  const handleClose = () => {
    form.resetFields();
    onClose();
  };

  const onFinish = async (values: {
    username: string;
    password: string;
    realName: string;
    roleLevel: 1 | 2 | 3;
    departmentId: number;
  }) => {
    try {
      const user = await createUser(values).unwrap();
      message.success('用户创建成功');
      handleClose();
      onSuccess?.(user);
    } catch (err: unknown) {
      const e = err as { data?: { code?: number; message?: string }; message?: string };
      const code = e.data?.code;
      const msg = e.data?.message ?? e.message ?? '创建失败';
      if (code === 40903) {
        form.setFields([{ name: 'username', errors: [msg] }]);
      } else {
        message.error(msg);
      }
    }
  };

  return (
    <Modal
      title="创建用户"
      open={open}
      onCancel={handleClose}
      onOk={() => form.submit()}
      confirmLoading={isLoading}
      destroyOnClose
      width={480}
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        initialValues={{ roleLevel: roleLevel === 2 ? 3 : undefined }}
      >
        <Form.Item
          name="username"
          label="用户名"
          rules={[
            { required: true, message: '请输入用户名' },
            { pattern: /^[a-z0-9_]{3,20}$/, message: '3-20 位小写字母、数字或下划线' },
          ]}
        >
          <Input placeholder="student_002" />
        </Form.Item>
        <Form.Item
          name="password"
          label="初始密码"
          rules={[
            { required: true, message: '请输入密码' },
            { min: 6, message: '至少 6 位' },
          ]}
        >
          <Input.Password placeholder="Test@123456" />
        </Form.Item>
        <Form.Item
          name="realName"
          label="姓名"
          rules={[{ required: true, message: '请输入姓名' }, { max: 20 }]}
        >
          <Input placeholder="李同学" />
        </Form.Item>
        <Form.Item
          name="roleLevel"
          label="角色"
          rules={[{ required: true, message: '请选择角色' }]}
        >
          <Select
            options={roleLevel === 2 ? ROLE_OPTIONS_COLLEGE : ROLE_OPTIONS_ADMIN}
            disabled={roleLevel === 2}
          />
        </Form.Item>
        <Form.Item
          name="departmentId"
          label="所属部门"
          rules={[{ required: true, message: '请选择部门' }]}
        >
          <DeptTreeSelect restrictSubtree={roleLevel === 2 ? restrictDeptId : undefined} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
