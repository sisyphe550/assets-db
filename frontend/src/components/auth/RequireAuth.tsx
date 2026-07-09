import { Navigate } from 'react-router-dom';
import { Button, Result, Spin } from 'antd';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser, selectIsAuthenticated } from '@/store/slices/authSlice';
import { useGetMeQuery } from '@/store/api/authApi';
import { ROLE_HOME } from '@/utils/constants';

interface RequireAuthProps {
  minRole: 1 | 2 | 3;
  children: React.ReactNode;
}

export default function RequireAuth({ minRole, children }: RequireAuthProps) {
  const isAuthenticated = useAppSelector(selectIsAuthenticated);
  const user = useAppSelector(selectCurrentUser);
  const { isLoading, isError, refetch } = useGetMeQuery(undefined, {
    skip: !isAuthenticated,
  });

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (isLoading && !user) {
    return <Spin size="large" tip="加载中..." style={{ display: 'block', margin: '40vh auto' }} />;
  }

  if (isError && !user) {
    return <Navigate to="/login" replace />;
  }

  if (isError && user) {
    return (
      <Result
        status="error"
        title="无法验证登录状态"
        subTitle="请重试或重新登录"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  if (user && user.status === 0) {
    return (
      <Result
        status="403"
        title="账户已禁用"
        subTitle="请联系管理员"
      />
    );
  }

  if (user && user.roleLevel !== minRole) {
    return <Navigate to={ROLE_HOME[user.roleLevel]} replace />;
  }

  return <>{children}</>;
}
