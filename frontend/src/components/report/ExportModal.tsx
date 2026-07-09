import { useEffect, useRef, useState } from 'react';
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
  params?: CreateExportReq['params'];
}

export default function ExportModal({ open, onClose, exportType, params }: ExportModalProps) {
  const [jobId, setJobId] = useState<number | null>(null);
  const [polling, setPolling] = useState(false);
  const downloadedJobIdRef = useRef<number | null>(null);
  const [createExport, { isLoading: creating }] = useCreateExportMutation();
  const { data: job, isFetching } = useGetExportStatusQuery(jobId!, {
    skip: jobId === null,
    pollingInterval: polling ? 2000 : 0,
    refetchOnFocus: false,
    refetchOnReconnect: false,
  });

  useEffect(() => {
    if (!open) {
      setJobId(null);
      setPolling(false);
      downloadedJobIdRef.current = null;
      return;
    }
    const start = async () => {
      try {
        const result = await createExport({ exportType, params }).unwrap();
        setJobId(result.jobId);
        setPolling(true);
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : '导出任务创建失败';
        message.error(msg);
        onClose();
      }
    };
    void start();
  }, [open, exportType, params, createExport, onClose]);

  useEffect(() => {
    if (job?.status !== undefined && job.status >= 2) {
      setPolling(false);
    }
  }, [job?.status]);

  useEffect(() => {
    if (!job || job.status !== 2 || !job.downloadUrl) return;
    if (downloadedJobIdRef.current === job.jobId) return;
    downloadedJobIdRef.current = job.jobId;
    void downloadWithAuth(job.downloadUrl).catch((err: Error) => message.error(err.message));
  }, [job?.jobId, job?.status, job?.downloadUrl]);

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
