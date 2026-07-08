import { Card, Typography } from 'antd';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

export default function AdminDashboardPage() {
  const user = useAppSelector(selectCurrentUser);
  return (
    <Card>
      <Typography.Title level={4}>仪表盘</Typography.Title>
      <Typography.Paragraph>
        欢迎，{user?.realName ?? '管理员'}。校级管理视图将在 P6 完善统计图表。
      </Typography.Paragraph>
    </Card>
  );
}
