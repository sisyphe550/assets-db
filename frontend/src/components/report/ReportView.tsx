import { useState } from 'react';
import { Button, Card, Select, Space, Typography, message } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import ReportCharts, { type ReportTab } from '@/components/report/ReportCharts';
import ExportModal from '@/components/report/ExportModal';
import type { CreateExportReq } from '@/types/api';

interface ReportViewProps {
  roleLevel: 1 | 2;
}

const EXPORT_OPTIONS: { label: string; value: CreateExportReq['exportType'] }[] = [
  { label: '资产清单', value: 'asset_list' },
  { label: '工单审批日志', value: 'workflow_log' },
];

export default function ReportView({ roleLevel }: ReportViewProps) {
  const user = useAppSelector(selectCurrentUser);
  const [tab, setTab] = useState<ReportTab>('dept');
  const [exportOpen, setExportOpen] = useState(false);
  const [exportType, setExportType] = useState<CreateExportReq['exportType']>('asset_list');
  const [diffTaskId, setDiffTaskId] = useState<number | undefined>();

  const activeExportType = tab === 'diff' && diffTaskId ? 'inventory_diff' : exportType;
  const exportParams =
    tab === 'diff' && diffTaskId ? { taskId: diffTaskId } : undefined;
  const canExport = tab !== 'diff' || !!diffTaskId;

  const handleExportClick = () => {
    if (!canExport) {
      message.warning('请先在「盘点差异」Tab 中选择盘点任务');
      return;
    }
    setExportOpen(true);
  };

  return (
    <Card>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <Typography.Title level={4} style={{ margin: 0 }}>
          统计报表
        </Typography.Title>
        <Space>
          {tab !== 'diff' ? (
            <Select
              value={exportType}
              style={{ width: 180 }}
              options={EXPORT_OPTIONS}
              onChange={setExportType}
            />
          ) : null}
          <Button
            icon={<DownloadOutlined />}
            disabled={!canExport}
            onClick={handleExportClick}
          >
            导出 CSV
          </Button>
        </Space>
      </Space>

      <ReportCharts
        activeTab={tab}
        onTabChange={setTab}
        departmentId={roleLevel === 2 ? user?.departmentId : undefined}
        restrictToSubtree={roleLevel === 2}
        onDiffTaskChange={setDiffTaskId}
      />

      <ExportModal
        open={exportOpen}
        onClose={() => setExportOpen(false)}
        exportType={activeExportType}
        params={exportParams}
      />
    </Card>
  );
}
