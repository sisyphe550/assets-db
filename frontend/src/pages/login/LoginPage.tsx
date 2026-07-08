import { Button, Card, Form, Input, Typography, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useLoginMutation, useLazyGetMeQuery } from '@/store/api/authApi';
import { useAppDispatch } from '@/store/hooks';
import { setCredentials, setUser } from '@/store/slices/authSlice';
import { ROLE_HOME } from '@/utils/constants';

export default function LoginPage() {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const [login, { isLoading }] = useLoginMutation();
  const [fetchMe] = useLazyGetMeQuery();

  const onFinish = async (values: { username: string; password: string }) => {
    try {
      const tokens = await login(values).unwrap();
      dispatch(setCredentials(tokens));
      const me = await fetchMe().unwrap();
      dispatch(setUser(me));
      message.success('登录成功');
      navigate(ROLE_HOME[me.roleLevel] ?? '/login');
    } catch (err: unknown) {
      const e = err as { code?: number; message?: string };
      if (e.code === 40301) {
        message.error('账户已被禁用，请联系管理员');
      } else {
        message.error(e.message ?? '用户名或密码错误');
      }
    }
  };

  return (
    <div
      style={{
        minHeight: '100vh',
        background: '#f0f2f5',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
    >
      <Card style={{ width: 400, borderRadius: 8 }}>
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <img src="/logo.svg" alt="FAMS" width={64} height={64} />
          <Typography.Title level={3} style={{ marginTop: 16, color: '#1677FF' }}>
            高校固定资产管理系统
          </Typography.Title>
          <Typography.Text type="secondary">Fixed Assets Management System</Typography.Text>
        </div>
        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="用户名/工号" />
          </Form.Item>
          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少 6 位' },
            ]}
          >
            <Input.Password placeholder="密码" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={isLoading}>
              登 录
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
