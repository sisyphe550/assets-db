import { Layout } from 'antd';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import type { MenuProps } from 'antd';
import { TopHeader, SidebarMenu } from '@/components/common/AppShell';
import { userMenu, type AppMenuItem } from '@/config/menu';
import { useEffect, useMemo } from 'react';
import { useGetMeQuery } from '@/store/api/authApi';
import { useGetTasksQuery } from '@/store/api/inventoryApi';
import { useAppDispatch } from '@/store/hooks';
import { setUser } from '@/store/slices/authSlice';

const { Content } = Layout;

export default function UserLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { data: me } = useGetMeQuery();
  const { data: assignedTasks } = useGetTasksQuery(
    { page: 1, pageSize: 100, status: 1, scope: 'assigned' },
    { skip: !me },
  );

  useEffect(() => {
    if (me) dispatch(setUser(me));
  }, [me, dispatch]);

  const inventoryMenuItems: AppMenuItem[] = useMemo(
    () =>
      (assignedTasks?.list ?? []).map((task) => ({
        key: `/user/inventory/${task.id}`,
        label: task.taskName.length > 12 ? `${task.taskName.slice(0, 12)}…` : task.taskName,
        iconName: 'TableOutlined',
      })),
    [assignedTasks?.list],
  );

  const onMenuClick: MenuProps['onClick'] = ({ key }) => navigate(key);

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <SidebarMenu
        items={userMenu}
        extraItems={inventoryMenuItems}
        selectedKey={location.pathname}
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
