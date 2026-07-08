import { useEffect, useState } from 'react';
import { Modal, Progress, Typography, message } from 'antd';
import {
  useCreateExportMutation,
  useGetExportStatusQuery,
} from '@/store/api/reportApi';
import { downloadWithAuth } from '@/utils/download';
import type { CreateExportReq } from '@/types/api';

const STATUS_LABEL: Record<number, string> = {
  0: '排队中',
  1: '处理中',
  2: '已完成',
  3: '失败',
};

interface ExportModalProps {
  open: boolean;
  onClose: () => void;
  exportType: CreateExportReq['exportType'];
}

export default function ExportModal({ open, onClose, exportType }: ExportModalProps) {
  const [jobId, setJobId] = useState<number | null>(null);
  const [polling, setPolling] = useState(false);
  const [createExport, { isLoading: creating }] = useCreateExportMutation();
  const { data: job, isFetching } = useGetExportStatusQuery(jobId!, {
    skip: jobId === null,
    pollingInterval: polling ? 2000 : 0,
  });

  useEffect(() => {
    if (!open) {
      setJobId(null);
      setPolling(false);
      return;
    }
    const start = async () => {
      try {
        const result = await createExport({ exportType }).unwrap();
        setJobId(result.jobId);
        setPolling(true);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '导出任务创建失败';
        message.error(msg);
        onClose();
      }
    };
    void start();
  }, [open, exportType, createExport, onClose]);

  useEffect(() => {
    if (job?.status !== undefined && job.status >= 2) {
      setPolling(false);
    }
  }, [job?.status]);

  useEffect(() => {
    if (!job || job.status !== 2) return;
    const url = job.downloadUrl;
    if (!url) return;
    void downloadWithAuth(url).catch((err: Error) => message.error(err.message));
  }, [job]);

  const status = job?.status ?? (creating ? 0 : undefined);
  const percent =
    status === 2 ? 100 : status === 3 ? 100 : status === 1 ? 66 : status === 0 ? 33 : 10;

  return (
    <Modal
      title="导出 CSV"
      open={open}
      onCancel={onClose}
      footer={null}
      destroyOnClose
      maskClosable={false}
    >
      <Progress
        percent={percent}
        status={status === 3 ? 'exception' : status === 2 ? 'success' : 'active'}
      />
      <Typography.Paragraph style={{ marginTop: 16 }}>
        {status !== undefined ? STATUS_LABEL[status] ?? '准备中' : '准备中'}
        {isFetching ? '…' : ''}
      </Typography.Paragraph>
      {status === 3 && (
        <Typography.Text type="danger">{job?.errorMessage ?? '导出失败，请稍后重试'}</Typography.Text>
      )}
      {status === 2 && (
        <Typography.Text type="success">文件已开始下载，可关闭此窗口。</Typography.Text>
      )}
    </Modal>
  );
}
