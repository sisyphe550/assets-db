import {
  DashboardOutlined,
  DatabaseOutlined,
  AuditOutlined,
  ApartmentOutlined,
  TeamOutlined,
  TableOutlined,
  BarChartOutlined,
  PlusCircleOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons';
import type { AppMenuItem } from '@/config/menu';

const iconMap: Record<string, React.ReactNode> = {
  DashboardOutlined: <DashboardOutlined />,
  DatabaseOutlined: <DatabaseOutlined />,
  AuditOutlined: <AuditOutlined />,
  ApartmentOutlined: <ApartmentOutlined />,
  TeamOutlined: <TeamOutlined />,
  TableOutlined: <TableOutlined />,
  BarChartOutlined: <BarChartOutlined />,
  PlusCircleOutlined: <PlusCircleOutlined />,
  UnorderedListOutlined: <UnorderedListOutlined />,
};

export function resolveMenuItems(items: AppMenuItem[]) {
  return items.map((item) => {
    if (!item || typeof item !== 'object') return item;
    const { iconName, ...rest } = item as AppMenuItem & { iconName?: string };
    return {
      ...rest,
      icon: iconName ? iconMap[iconName] : undefined,
    };
  });
}
