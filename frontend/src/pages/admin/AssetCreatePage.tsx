import { Card, message } from 'antd';
import { useNavigate } from 'react-router-dom';
import AssetForm from '@/components/asset/AssetForm';
import { useCreateAssetMutation } from '@/store/api/assetApi';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import type { CreateAssetReq } from '@/types/api';

export default function AssetCreatePage() {
  const navigate = useNavigate();
  const user = useAppSelector(selectCurrentUser);
  const [createAsset, { isLoading }] = useCreateAssetMutation();

  if (!user || user.roleLevel > 2) {
    return null;
  }

  const basePath = user.roleLevel === 1 ? '/admin' : '/college';

  const onSubmit = async (values: CreateAssetReq) => {
    try {
      await createAsset(values).unwrap();
      message.success('创建成功');
      navigate(`${basePath}/assets`);
    } catch (err: unknown) {
      const e = err as { code?: number; message?: string };
      if (e.code === 40903) {
        message.error('该资产编号已存在');
      } else {
        message.error(e.message ?? '创建失败');
      }
    }
  };

  return (
    <Card title="新增资产">
      <AssetForm
        mode="create"
        roleLevel={user.roleLevel as 1 | 2}
        defaultDepartmentId={user.departmentId}
        loading={isLoading}
        onSubmit={onSubmit}
        onCancel={() => navigate(`${basePath}/assets`)}
      />
    </Card>
  );
}
