import AssetTable from '@/components/asset/AssetTable';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

export default function AssetListPage() {
  const user = useAppSelector(selectCurrentUser);
  if (!user || user.roleLevel > 2) return null;
  return (
    <AssetTable
      basePath="/admin"
      roleLevel={user.roleLevel as 1 | 2}
    />
  );
}
