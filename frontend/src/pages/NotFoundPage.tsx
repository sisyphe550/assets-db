import { Button, Result } from 'antd';
import { useNavigate } from 'react-router-dom';
import { ROLE_HOME } from '@/utils/constants';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

export default function NotFoundPage() {
  const navigate = useNavigate();
  const user = useAppSelector(selectCurrentUser);

  return (
    <Result
      status="404"
      title="页面不存在"
      extra={[
        <Button
          key="home"
          type="primary"
          onClick={() => navigate(user ? ROLE_HOME[user.roleLevel] : '/login')}
        >
          返回首页
        </Button>,
        <Button key="back" onClick={() => navigate(-1)}>
          返回上一页
        </Button>,
      ]}
    />
  );
}
