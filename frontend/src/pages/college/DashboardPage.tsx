import { Card, Typography } from 'antd';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

export default function CollegeDashboardPage() {
  const user = useAppSelector(selectCurrentUser);
  return (
    <Card>
      <Typography.Title level={4}>本院仪表盘</Typography.Title>
      <Typography.Paragraph>
        欢迎，{user?.realName ?? '院级管理员'}（{user?.departmentName}）。
      </Typography.Paragraph>
    </Card>
  );
}
