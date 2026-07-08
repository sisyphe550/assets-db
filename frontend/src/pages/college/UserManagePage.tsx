import UserTable from '@/components/user/UserTable';
import { useAppSelector } from '@/store/hooks';
import { selectCurrentUser } from '@/store/slices/authSlice';

export default function CollegeUserManagePage() {
  const user = useAppSelector(selectCurrentUser);
  return <UserTable roleLevel={2} restrictDeptId={user?.departmentId} />;
}
