import type { MenuProps } from 'antd';

export type AppMenuItem = Required<MenuProps>['items'][number] & {
  iconName?: string;
};

export const adminMenu: AppMenuItem[] = [
  { key: '/admin/dashboard', label: '仪表盘', iconName: 'DashboardOutlined' },
  { key: '/admin/assets', label: '资产管理', iconName: 'DatabaseOutlined' },
  { key: '/admin/workflow/todo', label: '工单审批', iconName: 'AuditOutlined' },
  { key: '/admin/departments', label: '组织架构', iconName: 'ApartmentOutlined' },
  { key: '/admin/users', label: '用户管理', iconName: 'TeamOutlined' },
  { key: '/admin/inventory/tasks', label: '盘点管理', iconName: 'TableOutlined' },
  { key: '/admin/workflow/all', label: '全部工单', iconName: 'UnorderedListOutlined' },
  { key: '/admin/reports', label: '统计报表', iconName: 'BarChartOutlined' },
];

export const collegeMenu: AppMenuItem[] = [
  { key: '/college/dashboard', label: '仪表盘', iconName: 'DashboardOutlined' },
  { key: '/college/assets', label: '本院资产', iconName: 'DatabaseOutlined' },
  { key: '/college/workflow/todo', label: '工单审批', iconName: 'AuditOutlined' },
  { key: '/college/inventory/tasks', label: '盘点管理', iconName: 'TableOutlined' },
  { key: '/college/users', label: '用户管理', iconName: 'TeamOutlined' },
  { key: '/college/reports', label: '统计报表', iconName: 'BarChartOutlined' },
];

export const userMenu: AppMenuItem[] = [
  { key: '/user/assets', label: '我的资产', iconName: 'DatabaseOutlined' },
  { key: '/user/workflow/my', label: '我的工单', iconName: 'AuditOutlined' },
  { key: '/user/workflow/create', label: '新建申请', iconName: 'PlusCircleOutlined' },
];
