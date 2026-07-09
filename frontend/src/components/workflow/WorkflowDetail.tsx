import { useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Drawer,
  Input,
  Modal,
  Result,
  Space,
  Spin,
  Timeline,
  Typography,
  message,
} from 'antd';
import { useNavigate } from 'react-router-dom';
import {
  useApproveRequestMutation,
  useGetRequestQuery,
  useRejectRequestMutation,
} from '@/store/api/workflowApi';
import { useGetAssetQuery } from '@/store/api/assetApi';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import StatusTag from '@/components/common/StatusTag';
import { formatDateTime } from '@/utils/format';
import { canActOnWorkflow } from '@/utils/workflow';

interface WorkflowDetailProps {
  requestId: number;
  open: boolean;
  onClose: () => void;
  assetBasePath?: '/admin' | '/college' | '/user';
}

export default function WorkflowDetail({
  requestId,
  open,
  onClose,
  assetBasePath = '/admin',
}: WorkflowDetailProps) {
  const navigate = useNavigate();
  const user = useAppSelector(selectCurrentUser);
  const [comment, setComment] = useState('');

  const { data, isLoading, isError, refetch } = useGetRequestQuery(requestId, { skip: !open });
  const { data: asset } = useGetAssetQuery(data?.request.assetId ?? 0, {
    skip: !data?.request.assetId,
  });
  const [approve, { isLoading: approving }] = useApproveRequestMutation();
  const [reject, { isLoading: rejecting }] = useRejectRequestMutation();

  const request = data?.request;
  const canAct = user && request ? canActOnWorkflow(user, request) : null;

  const handleApprove = () => {
    Modal.confirm({
      title: '确认同意该申请？',
      content: comment ? `审批意见：${comment}` : undefined,
      okText: '同意',
      onOk: async () => {
        try {
          await approve({ id: requestId, body: { comment } }).unwrap();
          message.success('审批通过');
          setComment('');
          onClose();
        } catch (err: unknown) {
          const e = err as { message?: string };
          message.error(e.message ?? '操作失败');
        }
      },
    });
  };

  const handleReject = () => {
    if (!comment.trim()) {
      message.warning('驳回时请填写审批意见');
      return;
    }
    Modal.confirm({
      title: '确认驳回该申请？',
      okType: 'danger',
      okText: '驳回',
      onOk: async () => {
        try {
          await reject({ id: requestId, body: { comment } }).unwrap();
          message.success('已驳回');
          setComment('');
          onClose();
        } catch (err: unknown) {
          const e = err as { message?: string };
          message.error(e.message ?? '操作失败');
        }
      },
    });
  };

  return (
    <Drawer title={`工单详情 #${requestId}`} width={640} open={open} onClose={onClose}>
      {isLoading && <Spin style={{ display: 'block', margin: '40px auto' }} />}
      {isError && (
        <Result
          status="error"
          title="加载失败"
          extra={
            <Button type="primary" onClick={() => refetch()}>
              重试
            </Button>
          }
        />
      )}
      {request && (
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Descriptions bordered column={2} size="small" title="工单信息">
            <Descriptions.Item label="工单号">#{request.id}</Descriptions.Item>
            <Descriptions.Item label="类型">
              <StatusTag type="workflowType" value={request.type} />
            </Descriptions.Item>
            <Descriptions.Item label="状态">
              <StatusTag type="workflow" value={request.status} />
            </Descriptions.Item>
            <Descriptions.Item label="当前阶段">
              <StatusTag type="workflowStage" value={request.currentStage} />
            </Descriptions.Item>
            <Descriptions.Item label="申请人">
              {request.requesterName ?? `#${request.requesterId}`}
            </Descriptions.Item>
            <Descriptions.Item label="提交时间">
              {formatDateTime(request.createdAt)}
            </Descriptions.Item>
            <Descriptions.Item label="申请原因" span={2}>
              {request.reason}
            </Descriptions.Item>
          </Descriptions>

          <Card size="small" title="关联资产">
            {asset ? (
              <Descriptions column={1} size="small">
                <Descriptions.Item label="资产编号">
                  <Typography.Text copyable>{asset.assetNo}</Typography.Text>
                </Descriptions.Item>
                <Descriptions.Item label="名称">{asset.name}</Descriptions.Item>
                <Descriptions.Item label="地点">{asset.location}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  <StatusTag type="asset" value={asset.status} />
                </Descriptions.Item>
                {assetBasePath !== '/user' && (
                  <Descriptions.Item>
                    <Button
                      type="link"
                      size="small"
                      onClick={() => navigate(`${assetBasePath}/assets/${asset.id}`)}
                    >
                      查看资产详情
                    </Button>
                  </Descriptions.Item>
                )}
              </Descriptions>
            ) : (
              <Typography.Text type="secondary">资产 ID: {request.assetId}</Typography.Text>
            )}
          </Card>

          <Card size="small" title="审批历史">
            <Timeline
              items={(data?.logs ?? []).map((log) => ({
                children: (
                  <div>
                    <div>
                      <strong>{log.action}</strong> — 操作人 {log.operatorId}
                    </div>
                    <div style={{ color: '#888', fontSize: 12 }}>
                      {formatDateTime(log.operateTime)}
                    </div>
                    {log.comment && <div>意见：{log.comment}</div>}
                  </div>
                ),
              }))}
            />
          </Card>

          {canAct && (
            <Card size="small" title="审批操作">
              <Input.TextArea
                rows={3}
                placeholder="审批意见（驳回时必填）"
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                style={{ marginBottom: 12 }}
              />
              <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
                <Button danger loading={rejecting} onClick={handleReject}>
                  驳回
                </Button>
                <Button type="primary" loading={approving} onClick={handleApprove}>
                  同意
                </Button>
              </Space>
            </Card>
          )}
        </Space>
      )}
    </Drawer>
  );
}
