import { Card, Typography } from 'antd';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import DashboardOverview from '@/components/report/DashboardOverview';

export default function AdminDashboardPage() {
  const user = useAppSelector(selectCurrentUser);
  return (
    <Card>
      <Typography.Title level={4}>仪表盘</Typography.Title>
      <Typography.Paragraph>
        欢迎，{user?.realName ?? '管理员'}。以下为全校资产与工单概览。
      </Typography.Paragraph>
      <DashboardOverview roleLevel={1} basePath="/admin" />
    </Card>
  );
}
