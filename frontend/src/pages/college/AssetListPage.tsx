import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';
import AssetTable from '@/components/asset/AssetTable';

export default function AssetListPage() {
  const user = useAppSelector(selectCurrentUser);
  if (!user) return null;
  return <AssetTable basePath="/college" roleLevel={2} />;
}
