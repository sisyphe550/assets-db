import { useMemo, useState } from 'react';
import {
  Button,
  Card,
  Descriptions,
  Modal,
  Result,
  Space,
  Spin,
  Typography,
  message,
} from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import StatusTag from '@/components/common/StatusTag';
import AssetForm from '@/components/asset/AssetForm';
import {
  useDeleteAssetMutation,
  useGetAssetQuery,
  useUpdateAssetMutation,
} from '@/store/api/assetApi';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import { useGetDeptTreeQuery } from '@/store/api/userApi';
import { getApiErrorCode } from '@/utils/api';
import { formatDate, formatPrice } from '@/utils/format';
import { collectSubtreeIds, findDeptNode } from '@/utils/dept';
import type { CreateAssetReq } from '@/types/api';

interface AssetDetailViewProps {
  basePath: '/admin' | '/college';
}

export default function AssetDetailView({ basePath }: AssetDetailViewProps) {
  const { id } = useParams();
  const assetId = Number(id);
  const navigate = useNavigate();
  const user = useAppSelector(selectCurrentUser);
  const [editOpen, setEditOpen] = useState(false);

  const { data, isLoading, isError, error, refetch } = useGetAssetQuery(assetId, {
    skip: !assetId,
  });
  const [updateAsset, { isLoading: updating }] = useUpdateAssetMutation();
  const [deleteAsset, { isLoading: deleting }] = useDeleteAssetMutation();
  const { data: deptTree } = useGetDeptTreeQuery(undefined, {
    skip: !user || user.roleLevel !== 2,
  });

  const allowedDeptIds = useMemo(() => {
    if (!user || user.roleLevel !== 2 || !deptTree) return null;
    const root = findDeptNode(deptTree, user.departmentId);
    const ids = root ? collectSubtreeIds(root) : [user.departmentId];
    return new Set(ids);
  }, [user, deptTree]);

  const canEdit =
    user &&
    (user.roleLevel === 1 ||
      (user.roleLevel === 2 && data && allowedDeptIds?.has(data.departmentId)));

  const canDelete = user?.roleLevel === 1;

  const handleUpdate = async (values: CreateAssetReq) => {
    try {
      await updateAsset({
        id: assetId,
        body: {
          name: values.name,
          category: values.category,
          location: values.location,
          departmentId: values.departmentId,
          isShared: values.isShared,
        },
      }).unwrap();
      message.success('保存成功');
      setEditOpen(false);
    } catch (err: unknown) {
      const e = err as { message?: string };
      message.error(e.message ?? '保存失败');
    }
  };

  const handleDelete = () => {
    Modal.confirm({
      title: '确认删除该资产？',
      content: '删除后不可恢复（逻辑删除）',
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteAsset(assetId).unwrap();
          message.success('已删除');
          navigate(`${basePath}/assets`);
        } catch (err: unknown) {
          const e = err as { message?: string };
          message.error(e.message ?? '删除失败');
        }
      },
    });
  };

  if (isLoading) {
    return <Spin size="large" style={{ display: 'block', margin: '40vh auto' }} />;
  }

  if (isError) {
    const code = getApiErrorCode(error);
    if (code === 40401) {
      return (
        <Result
          status="404"
          title="资产不存在"
          extra={
            <Button type="primary" onClick={() => navigate(`${basePath}/assets`)}>
              返回列表
            </Button>
          }
        />
      );
    }
    if (code === 40302) {
      return <Result status="403" title="无权查看该资产" />;
    }
    return (
      <Result
        status="error"
        title="加载失败"
        extra={
          <Button type="primary" onClick={() => refetch()}>
            重试
          </Button>
        }
      />
    );
  }

  if (!data) return null;

  return (
    <>
      <Card
        title={
          <Space>
            <Button
              type="text"
              icon={<ArrowLeftOutlined />}
              onClick={() => navigate(`${basePath}/assets`)}
            />
            资产详情
          </Space>
        }
        extra={
          canEdit ? (
            <Space>
              <Button onClick={() => setEditOpen(true)}>编辑</Button>
              {canDelete && (
                <Button danger loading={deleting} onClick={handleDelete}>
                  删除
                </Button>
              )}
            </Space>
          ) : null
        }
      >
        <Descriptions bordered column={2}>
          <Descriptions.Item label="资产编号">
            <Typography.Text copyable>{data.assetNo}</Typography.Text>
          </Descriptions.Item>
          <Descriptions.Item label="名称">{data.name}</Descriptions.Item>
          <Descriptions.Item label="状态">
            <StatusTag type="asset" value={data.status} />
          </Descriptions.Item>
          <Descriptions.Item label="类别">{data.category}</Descriptions.Item>
          <Descriptions.Item label="价格">{formatPrice(data.price)}</Descriptions.Item>
          <Descriptions.Item label="购置日期">{formatDate(data.purchaseTime)}</Descriptions.Item>
          <Descriptions.Item label="存放地点">{data.location}</Descriptions.Item>
          <Descriptions.Item label="所属部门ID">{data.departmentId}</Descriptions.Item>
          <Descriptions.Item label="领用人ID">{data.userId ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="学院共享">{data.isShared ? '是' : '否'}</Descriptions.Item>
        </Descriptions>
      </Card>

      <Modal
        title="编辑资产"
        open={editOpen}
        footer={null}
        onCancel={() => setEditOpen(false)}
        destroyOnClose
        width={560}
      >
        {user && user.roleLevel <= 2 && (
          <AssetForm
            mode="edit"
            roleLevel={user.roleLevel as 1 | 2}
            defaultDepartmentId={user.departmentId}
            initialAsset={data}
            loading={updating}
            onSubmit={handleUpdate}
            onCancel={() => setEditOpen(false)}
          />
        )}
      </Modal>
    </>
  );
}
