import { Layout, Menu, Dropdown, Button, theme } from 'antd';
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LogoutOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import { toggleSidebar, selectSidebarCollapsed } from '@/store/slices/uiSlice';
import { useLogoutMutation } from '@/store/api/authApi';
import { logout } from '@/store/slices/authSlice';
import type { MenuProps } from 'antd';
import type { AppMenuItem } from '@/config/menu';
import { resolveMenuItems } from '@/config/menuIcons';

const { Header, Sider } = Layout;

export function TopHeader() {
  const dispatch = useAppDispatch();
  const collapsed = useAppSelector(selectSidebarCollapsed);
  const user = useAppSelector(selectCurrentUser);
  const navigate = useNavigate();
  const [logoutApi] = useLogoutMutation();
  const { token } = theme.useToken();

  const handleLogout = async () => {
    try {
      await logoutApi().unwrap();
    } catch {
      // 即使 API 失败也清空本地状态
    }
    dispatch(logout());
    navigate('/login');
  };

  const dropdownItems: MenuProps['items'] = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: handleLogout,
    },
  ];

  return (
    <Header
      style={{
        display: 'flex',
        alignItems: 'center',
        padding: '0 24px',
        background: '#001529',
        gap: 16,
      }}
    >
      <Button
        type="text"
        icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
        onClick={() => dispatch(toggleSidebar())}
        style={{ color: '#fff', fontSize: 16 }}
        aria-label="折叠侧边栏"
      />
      <img src="/logo.svg" alt="FAMS" width={32} height={32} />
      <span style={{ color: '#fff', fontWeight: 600, fontSize: 16, flex: 1 }}>
        高校固定资产管理系统
      </span>
      <Dropdown menu={{ items: dropdownItems }} placement="bottomRight">
        <Button type="text" style={{ color: token.colorTextLightSolid }} icon={<UserOutlined />}>
          {user?.realName ?? '用户'}
        </Button>
      </Dropdown>
    </Header>
  );
}

interface SidebarMenuProps {
  items: AppMenuItem[];
  extraItems?: AppMenuItem[];
  selectedKey: string;
  onMenuClick: MenuProps['onClick'];
}

export function SidebarMenu({ items, extraItems = [], selectedKey, onMenuClick }: SidebarMenuProps) {
  const collapsed = useAppSelector(selectSidebarCollapsed);
  const allItems = resolveMenuItems([...items, ...extraItems]);

  return (
    <Sider
      collapsible
      collapsed={collapsed}
      trigger={null}
      width={220}
      collapsedWidth={80}
      breakpoint="lg"
      style={{ background: '#001529' }}
    >
      <Menu
        theme="dark"
        mode="inline"
        selectedKeys={[selectedKey]}
        items={allItems}
        onClick={onMenuClick}
        style={{ marginTop: 8 }}
      />
    </Sider>
  );
}
