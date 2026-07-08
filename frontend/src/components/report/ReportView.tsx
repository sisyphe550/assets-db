import { useState } from 'react';
import { Button, Card, Space, Typography } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import ReportCharts, { type ReportTab } from '@/components/report/ReportCharts';
import ExportModal from '@/components/report/ExportModal';

interface ReportViewProps {
  roleLevel: 1 | 2;
}

export default function ReportView({ roleLevel }: ReportViewProps) {
  const user = useAppSelector(selectCurrentUser);
  const [tab, setTab] = useState<ReportTab>('dept');
  const [exportOpen, setExportOpen] = useState(false);

  return (
    <Card>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          统计报表
        </Typography.Title>
        <Button icon={<DownloadOutlined />} onClick={() => setExportOpen(true)}>
          导出 CSV
        </Button>
      </Space>

      <ReportCharts
        activeTab={tab}
        onTabChange={setTab}
        departmentId={roleLevel === 2 ? user?.departmentId : undefined}
        restrictToSubtree={roleLevel === 2}
      />

      <ExportModal
        open={exportOpen}
        onClose={() => setExportOpen(false)}
        exportType="asset_list"
      />
    </Card>
  );
}
