import { Layout } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import type { MenuProps } from 'antd';
import { TopHeader, SidebarMenu } from '@/components/common/AppShell';
import { adminMenu } from '@/config/menu';
import { matchMenuSelectedKey } from '@/config/menuSelected';
import { useEffect } from 'react';
import { useGetMeQuery } from '@/store/api/authApi';
import { useAppDispatch } from '@/store/hooks';
import { setUser } from '@/store/slices/authSlice';

const { Content } = Layout;

export default function AdminLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { data: me } = useGetMeQuery();

  useEffect(() => {
    if (me) dispatch(setUser(me));
  }, [me, dispatch]);

  const onMenuClick: MenuProps['onClick'] = ({ key }) => navigate(key);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <SidebarMenu
        items={adminMenu}
        selectedKey={matchMenuSelectedKey(location.pathname, adminMenu)}
        onMenuClick={onMenuClick}
      />
      <Layout>
        <TopHeader />
        <Content style={{ margin: 24, background: '#f5f5f5', minHeight: 280 }}>
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
