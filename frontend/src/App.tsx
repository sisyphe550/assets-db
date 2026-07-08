import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom';
import { lazy, Suspense, type ReactNode } from 'react';
import { Spin } from 'antd';
import RequireAuth from '@/components/auth/RequireAuth';
import AdminLayout from '@/layouts/AdminLayout';
import CollegeLayout from '@/layouts/CollegeLayout';
import UserLayout from '@/layouts/UserLayout';

const LoginPage = lazy(() => import('@/pages/login/LoginPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

const AdminDashboard = lazy(() => import('@/pages/admin/DashboardPage'));
const AdminAssetList = lazy(() => import('@/pages/admin/AssetListPage'));
const AdminAssetDetail = lazy(() => import('@/pages/admin/AssetDetailPage'));
const AdminAssetCreate = lazy(() => import('@/pages/admin/AssetCreatePage'));
const AdminWorkflowTodo = lazy(() => import('@/pages/admin/WorkflowTodoPage'));
const AdminWorkflowList = lazy(() => import('@/pages/admin/WorkflowListPage'));
const AdminDepartment = lazy(() => import('@/pages/admin/DepartmentPage'));
const AdminUsers = lazy(() => import('@/pages/admin/UserManagePage'));
const AdminInventoryList = lazy(() => import('@/pages/admin/InventoryTaskListPage'));
const AdminInventoryCreate = lazy(() => import('@/pages/admin/InventoryTaskCreatePage'));
const AdminInventoryDetail = lazy(() => import('@/pages/admin/InventoryTaskDetailPage'));
const AdminReports = lazy(() => import('@/pages/admin/ReportPage'));

const CollegeDashboard = lazy(() => import('@/pages/college/DashboardPage'));
const CollegeAssetList = lazy(() => import('@/pages/college/AssetListPage'));
const CollegeAssetDetail = lazy(() => import('@/pages/college/AssetDetailPage'));
const CollegeWorkflowTodo = lazy(() => import('@/pages/college/WorkflowTodoPage'));
const CollegeInventoryList = lazy(() => import('@/pages/college/InventoryTaskListPage'));
const CollegeInventoryCreate = lazy(() => import('@/pages/college/InventoryTaskCreatePage'));
const CollegeInventoryDetail = lazy(() => import('@/pages/college/InventoryTaskDetailPage'));

const UserAssets = lazy(() => import('@/pages/user/MyAssetsPage'));
const UserWorkflowMy = lazy(() => import('@/pages/user/WorkflowMyPage'));
const UserWorkflowCreate = lazy(() => import('@/pages/user/WorkflowCreatePage'));
const UserInventorySubmit = lazy(() => import('@/pages/user/InventorySubmitPage'));

const withSuspense = (el: ReactNode) => (
  <Suspense fallback={<Spin size="large" style={{ display: 'block', margin: '40vh auto' }} />}>
    {el}
  </Suspense>
);

const router = createBrowserRouter([
  { path: '/login', element: withSuspense(<LoginPage />) },
  {
    path: '/admin',
    element: (
      <RequireAuth minRole={1}>
        <AdminLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="dashboard" replace /> },
      { path: 'dashboard', element: withSuspense(<AdminDashboard />) },
      { path: 'assets', element: withSuspense(<AdminAssetList />) },
      { path: 'assets/create', element: withSuspense(<AdminAssetCreate />) },
      { path: 'assets/:id', element: withSuspense(<AdminAssetDetail />) },
      { path: 'workflow/todo', element: withSuspense(<AdminWorkflowTodo />) },
      { path: 'workflow/all', element: withSuspense(<AdminWorkflowList />) },
      { path: 'departments', element: withSuspense(<AdminDepartment />) },
      { path: 'users', element: withSuspense(<AdminUsers />) },
      { path: 'inventory/tasks', element: withSuspense(<AdminInventoryList />) },
      { path: 'inventory/tasks/create', element: withSuspense(<AdminInventoryCreate />) },
      { path: 'inventory/tasks/:id', element: withSuspense(<AdminInventoryDetail />) },
      { path: 'reports', element: withSuspense(<AdminReports />) },
    ],
  },
  {
    path: '/college',
    element: (
      <RequireAuth minRole={2}>
        <CollegeLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="dashboard" replace /> },
      { path: 'dashboard', element: withSuspense(<CollegeDashboard />) },
      { path: 'assets', element: withSuspense(<CollegeAssetList />) },
      { path: 'assets/:id', element: withSuspense(<CollegeAssetDetail />) },
      { path: 'workflow/todo', element: withSuspense(<CollegeWorkflowTodo />) },
      { path: 'inventory/tasks', element: withSuspense(<CollegeInventoryList />) },
      { path: 'inventory/tasks/create', element: withSuspense(<CollegeInventoryCreate />) },
      { path: 'inventory/tasks/:id', element: withSuspense(<CollegeInventoryDetail />) },
    ],
  },
  {
    path: '/user',
    element: (
      <RequireAuth minRole={3}>
        <UserLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <Navigate to="assets" replace /> },
      { path: 'assets', element: withSuspense(<UserAssets />) },
      { path: 'workflow/my', element: withSuspense(<UserWorkflowMy />) },
      { path: 'workflow/create', element: withSuspense(<UserWorkflowCreate />) },
      { path: 'inventory/:taskId', element: withSuspense(<UserInventorySubmit />) },
    ],
  },
  { path: '/', element: <Navigate to="/login" replace /> },
  { path: '*', element: withSuspense(<NotFoundPage />) },
]);

export default function App() {
  return <RouterProvider router={router} />;
}
