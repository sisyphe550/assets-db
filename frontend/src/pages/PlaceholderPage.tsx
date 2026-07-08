import { Card, Typography } from 'antd';

interface PlaceholderPageProps {
  title: string;
  phase?: string;
}

export default function PlaceholderPage({ title, phase = '后续阶段' }: PlaceholderPageProps) {
  return (
    <Card>
      <Typography.Title level={4}>{title}</Typography.Title>
      <Typography.Paragraph type="secondary">
        该页面将在 {phase} 实现，当前为 P1-P2 基础框架占位。
      </Typography.Paragraph>
    </Card>
  );
}
