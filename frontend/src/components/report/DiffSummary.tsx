import { Col, Row } from 'antd';
import { CheckCircleOutlined, MinusCircleOutlined, PlusCircleOutlined } from '@ant-design/icons';
import StatCard from '@/components/report/StatCard';

interface DiffSummaryProps {
  matchCount: number;
  surplusCount: number;
  deficitCount: number;
  loading?: boolean;
}

export default function DiffSummary({
  matchCount,
  surplusCount,
  deficitCount,
  loading,
}: DiffSummaryProps) {
  return (
    <Row gutter={16}>
      <Col xs={24} sm={8}>
        <StatCard
          title="相符"
          value={matchCount}
          prefix={<CheckCircleOutlined style={{ color: '#52c41a' }} />}
          loading={loading}
        />
      </Col>
      <Col xs={24} sm={8}>
        <StatCard
          title="盘盈"
          value={surplusCount}
          prefix={<PlusCircleOutlined style={{ color: '#faad14' }} />}
          loading={loading}
        />
      </Col>
      <Col xs={24} sm={8}>
        <StatCard
          title="盘亏"
          value={deficitCount}
          prefix={<MinusCircleOutlined style={{ color: '#ff4d4f' }} />}
          loading={loading}
        />
      </Col>
    </Row>
  );
}
