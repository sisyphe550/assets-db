# FAMS 前端实现蓝图（AI 开发用）

> 本文档提供可直接编码的完整规格：依赖、配置、路由、Store、API、权限矩阵。  
> 配合 `01-design.md`（架构）、`03-pages.md`（交互）、`05-components.md`（组件）、`06-visual-design.md`（视觉）使用。

---

## 1. 依赖清单

### 1.1 package.json

```json
{
  "name": "fams-frontend",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext ts,tsx"
  },
  "dependencies": {
    "@ant-design/charts": "2.2.1",
    "@ant-design/icons": "5.5.1",
    "@ant-design/pro-components": "2.8.2",
    "@reduxjs/toolkit": "2.5.0",
    "@univerjs/core": "0.5.5",
    "@univerjs/design": "0.5.5",
    "@univerjs/engine-render": "0.5.5",
    "@univerjs/sheets": "0.5.5",
    "@univerjs/ui": "0.5.5",
    "antd": "5.22.5",
    "dayjs": "1.11.13",
    "react": "18.3.1",
    "react-dom": "18.3.1",
    "react-redux": "9.2.0",
    "react-router-dom": "6.28.0"
  },
  "devDependencies": {
    "@types/react": "18.3.12",
    "@types/react-dom": "18.3.1",
    "@vitejs/plugin-react": "4.3.4",
    "typescript": "5.6.3",
    "vite": "6.0.3"
  }
}
```

> **版本锁定**：不使用 `^`，Univer 和 Ant Design 升级需全量回归测试。

---

## 2. 配置文件

### 2.1 vite.config.ts

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { '@': path.resolve(__dirname, 'src') },
  },
  server: {
    port: 5173,
    proxy: {
      '/api/v1/user': 'http://localhost:8888',
      '/api/v1/asset': 'http://localhost:8889',
      '/api/v1/workflow': 'http://localhost:8890',
      '/api/v1/inventory': 'http://localhost:8891',
      '/api/v1/report': 'http://localhost:8892',
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          antd: ['antd', '@ant-design/pro-components', '@ant-design/icons'],
          charts: ['@ant-design/charts'],
          univer: ['@univerjs/core', '@univerjs/sheets', '@univerjs/ui'],
        },
      },
    },
  },
});
```

### 2.2 tsconfig.json

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  },
  "include": ["src"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

### 2.3 index.html

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/png" href="/logo.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>高校固定资产管理系统</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

---

## 3. 入口与全局配置

### 3.1 main.tsx

```typescript
import React from 'react';
import ReactDOM from 'react-dom/client';
import { Provider } from 'react-redux';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { store } from './store';
import App from './App';

dayjs.locale('zh-cn');

ReactDOM.createRoot(document.getElementById('root')!).render(
  <Provider store={store}>
    <ConfigProvider locale={zhCN}>
      <App />
    </ConfigProvider>
  </Provider>
);
```

### 3.2 App.tsx 路由树

```typescript
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom';
import { lazy, Suspense } from 'react';
import { Spin } from 'antd';
import RequireAuth from '@/components/auth/RequireAuth';
import AdminLayout from '@/layouts/AdminLayout';
import CollegeLayout from '@/layouts/CollegeLayout';
import UserLayout from '@/layouts/UserLayout';

const LoginPage = lazy(() => import('@/pages/login/LoginPage'));
const NotFoundPage = lazy(() => import('@/pages/NotFoundPage'));

// Admin pages
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

// College pages（复用 admin 组件或独立薄包装）
const CollegeDashboard = lazy(() => import('@/pages/college/DashboardPage'));
const CollegeAssetList = lazy(() => import('@/pages/college/AssetListPage'));
const CollegeAssetDetail = lazy(() => import('@/pages/college/AssetDetailPage'));
const CollegeWorkflowTodo = lazy(() => import('@/pages/college/WorkflowTodoPage'));
const CollegeInventoryList = lazy(() => import('@/pages/college/InventoryTaskListPage'));
const CollegeInventoryCreate = lazy(() => import('@/pages/college/InventoryTaskCreatePage'));
const CollegeInventoryDetail = lazy(() => import('@/pages/college/InventoryTaskDetailPage'));

// User pages
const UserAssets = lazy(() => import('@/pages/user/MyAssetsPage'));
const UserWorkflowMy = lazy(() => import('@/pages/user/WorkflowMyPage'));
const UserWorkflowCreate = lazy(() => import('@/pages/user/WorkflowCreatePage'));
const UserInventorySubmit = lazy(() => import('@/pages/user/InventorySubmitPage'));

const withSuspense = (el: React.ReactNode) => (
  <Suspense fallback={<Spin size="large" style={{ display: 'block', margin: '40vh auto' }} />}>
    {el}
  </Suspense>
);

const router = createBrowserRouter([
  { path: '/login', element: withSuspense(<LoginPage />) },
  {
    path: '/admin',
    element: <RequireAuth minRole={1}><AdminLayout /></RequireAuth>,
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
    element: <RequireAuth minRole={2}><CollegeLayout /></RequireAuth>,
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
    element: <RequireAuth minRole={3}><UserLayout /></RequireAuth>,
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
```

---

## 4. 侧边栏菜单配置

```typescript
// config/menu.ts
import type { MenuProps } from 'antd';

type MenuItem = Required<MenuProps>['items'][number];

export const adminMenu: MenuItem[] = [
  { key: '/admin/dashboard', label: '仪表盘', icon: 'DashboardOutlined' },
  { key: '/admin/assets', label: '资产管理', icon: 'DatabaseOutlined' },
  { key: '/admin/workflow/todo', label: '工单审批', icon: 'AuditOutlined' },
  { key: '/admin/departments', label: '组织架构', icon: 'ApartmentOutlined' },
  { key: '/admin/users', label: '用户管理', icon: 'TeamOutlined' },
  { key: '/admin/inventory/tasks', label: '盘点管理', icon: 'TableOutlined' },
  { key: '/admin/reports', label: '统计报表', icon: 'BarChartOutlined' },
];

export const collegeMenu: MenuItem[] = [
  { key: '/college/dashboard', label: '仪表盘', icon: 'DashboardOutlined' },
  { key: '/college/assets', label: '本院资产', icon: 'DatabaseOutlined' },
  { key: '/college/workflow/todo', label: '工单审批', icon: 'AuditOutlined' },
  { key: '/college/inventory/tasks', label: '盘点管理', icon: 'TableOutlined' },
];

export const userMenu: MenuItem[] = [
  { key: '/user/assets', label: '我的资产', icon: 'DatabaseOutlined' },
  { key: '/user/workflow/my', label: '我的工单', icon: 'AuditOutlined' },
  { key: '/user/workflow/create', label: '新建申请', icon: 'PlusCircleOutlined' },
  // 盘点录入：动态注入（仅显示指派给当前用户的进行中任务）
];
```

**动态盘点菜单**：`UserLayout` 挂载时调用 `GET /inventory/tasks?status=1`，过滤 `assigneeIds` 包含当前用户的任务，追加菜单项 `/user/inventory/:taskId`。

---

## 5. 权限矩阵

| 功能 | role=1 | role=2 | role=3 |
|---|---|---|---|
| 查看全校资产 | ✅ | ❌（本院） | ❌（个人+共享） |
| 新增/编辑/删除资产 | ✅ | ✅（本院） | ❌ |
| 院级初审 | ❌ | ✅ | ❌ |
| 校级复审 | ✅ | ❌ | ❌ |
| 创建工单 | ❌ | ❌ | ✅ |
| 管理组织架构 | ✅ | ❌ | ❌ |
| 创建用户 | ✅ | ✅（仅 role=3） | ❌ |
| 创建盘点任务 | ✅ | ✅（本院） | ❌ |
| 盘点录入 | ✅ | ✅ | ✅（仅指派任务） |
| 归档盘点 | ✅ | ✅ | ❌ |
| 统计报表 | ✅ | ✅（本院） | ❌ |
| 导出 CSV | ✅ | ✅ | ❌ |

**路由守卫**：

```typescript
// components/auth/RequireAuth.tsx
function getRoleHome(roleLevel: number): string {
  if (roleLevel === 1) return '/admin/dashboard';
  if (roleLevel === 2) return '/college/dashboard';
  return '/user/assets';
}

// minRole: 1=校级可进, 2=院级可进, 3=所有登录用户可进
// roleLevel 数字越小权限越高
```

---

## 6. Store 完整实现

### 6.1 store/index.ts

```typescript
import { configureStore } from '@reduxjs/toolkit';
import authReducer from './slices/authSlice';
import uiReducer from './slices/uiSlice';
import { authApi } from './api/authApi';
import { assetApi } from './api/assetApi';
import { workflowApi } from './api/workflowApi';
import { inventoryApi } from './api/inventoryApi';
import { reportApi } from './api/reportApi';
import { userApi } from './api/userApi';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    ui: uiReducer,
    [authApi.reducerPath]: authApi.reducer,
    [assetApi.reducerPath]: assetApi.reducer,
    [workflowApi.reducerPath]: workflowApi.reducer,
    [inventoryApi.reducerPath]: inventoryApi.reducer,
    [reportApi.reducerPath]: reportApi.reducer,
    [userApi.reducerPath]: userApi.reducer,
  },
  middleware: (getDefault) =>
    getDefault().concat(
      authApi.middleware,
      assetApi.middleware,
      workflowApi.middleware,
      inventoryApi.middleware,
      reportApi.middleware,
      userApi.middleware,
    ),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
```

### 6.2 store/api/baseQuery.ts

```typescript
import { fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { BaseQueryFn, FetchArgs, FetchBaseQueryError } from '@reduxjs/toolkit/query';
import { message } from 'antd';
import type { RootState } from '../index';
import { setCredentials, logout } from '../slices/authSlice';

const rawBaseQuery = fetchBaseQuery({
  baseUrl: '/api/v1',
  prepareHeaders: (headers, { getState }) => {
    const token = (getState() as RootState).auth.accessToken;
    if (token) headers.set('Authorization', `Bearer ${token}`);
    return headers;
  },
});

export const baseQueryWithReauth: BaseQueryFn<
  string | FetchArgs,
  unknown,
  FetchBaseQueryError
> = async (args, api, extraOptions) => {
  let result = await rawBaseQuery(args, api, extraOptions);

  if (result.error && result.error.status === 401) {
    const state = api.getState() as RootState;
    const refreshToken = state.auth.refreshToken;
    if (refreshToken) {
      const refreshResult = await rawBaseQuery(
        { url: '/user/refresh', method: 'POST', body: { refreshToken } },
        api,
        extraOptions,
      );
      if (refreshResult.data) {
        const data = refreshResult.data as { code: number; data: TokenPair };
        if (data.code === 0) {
          api.dispatch(setCredentials(data.data));
          result = await rawBaseQuery(args, api, extraOptions);
          return result;
        }
      }
    }
    api.dispatch(logout());
    message.error('登录已过期，请重新登录');
    window.location.href = '/login';
  }
  return result;
};

interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
}
```

### 6.3 RTK Query API Slices 端点清单

#### authApi.ts

| endpoint | 方法 | 路径 | invalidatesTags |
|---|---|---|---|
| login | mutation POST | `/user/login` | — |
| refresh | mutation POST | `/user/refresh` | — |
| logout | mutation POST | `/user/logout` | 全部 |
| getMe | query GET | `/user/me` | — |

#### assetApi.ts

| endpoint | 方法 | 路径 | tags |
|---|---|---|---|
| getAssets | query GET | `/asset/assets` | AssetList |
| getAsset | query GET | `/asset/assets/:id` | Asset:{id} |
| createAsset | mutation POST | `/asset/assets` | invalidates AssetList |
| updateAsset | mutation PUT | `/asset/assets/:id` | invalidates AssetList, Asset:{id} |
| deleteAsset | mutation DELETE | `/asset/assets/:id` | invalidates AssetList |
| getSharedAssets | query GET | `/asset/assets/shared` | SharedAssetList |

#### workflowApi.ts

| endpoint | 方法 | 路径 | tags |
|---|---|---|---|
| getRequests | query GET | `/workflow/requests` | WorkflowList |
| getRequest | query GET | `/workflow/requests/:id` | Workflow:{id} |
| createRequest | mutation POST | `/workflow/requests` | invalidates WorkflowList |
| approveRequest | mutation POST | `/workflow/requests/:id/approve` | invalidates WorkflowList, Workflow:{id}, AssetList |
| rejectRequest | mutation POST | `/workflow/requests/:id/reject` | invalidates WorkflowList, Workflow:{id} |

#### inventoryApi.ts

| endpoint | 方法 | 路径 | tags |
|---|---|---|---|
| getTasks | query GET | `/inventory/tasks` | InventoryList |
| getTask | query GET | `/inventory/tasks/:id` | Inventory:{id} |
| createTask | mutation POST | `/inventory/tasks` | invalidates InventoryList |
| getExpectedAssets | query GET | `/inventory/tasks/:id/expected-assets` | — |
| submitRecords | mutation POST | `/inventory/tasks/:id/submit` | — |
| archiveTask | mutation POST | `/inventory/tasks/:id/archive` | invalidates InventoryList, Inventory:{id} |
| getRecords | query GET | `/inventory/tasks/:id/records` | InventoryRecords:{id} |

#### reportApi.ts

| endpoint | 方法 | 路径 | tags |
|---|---|---|---|
| getAssetsByDept | query GET | `/report/assets/by-dept` | — |
| getAssetsByCategory | query GET | `/report/assets/by-category` | — |
| getInventoryDiff | query GET | `/report/inventory/diff/:taskId` | — |
| createExport | mutation POST | `/report/export` | — |
| getExportStatus | query GET | `/report/export/:jobId` | — |

#### userApi.ts

| endpoint | 方法 | 路径 | tags |
|---|---|---|---|
| getDeptTree | query GET | `/user/departments/tree` | DeptTree |
| createDept | mutation POST | `/user/departments` | invalidates DeptTree |
| listUsers | query GET | `/user/users` | UserList |
| getUser | query GET | `/user/users/:id` | User:{id} |
| getCollegeSubtree | query GET | `/user/departments/college-subtree` | — |
| updateUserStatus | mutation PUT | `/user/users/:id/status` | — |
| forceLogout | mutation POST | `/user/users/:id/force-logout` | — |

---

## 7. 类型定义（完整）

```typescript
// types/common.ts
export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PaginatedData<T> {
  list: T[];
  page: number;
  pageSize: number;
  total: number;
}

export interface ListParams {
  page?: number;
  pageSize?: number;
}

// types/api.ts
export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  tokenType: string;
}

export interface UserInfo {
  id: number;
  username: string;
  realName: string;
  roleLevel: 1 | 2 | 3;
  departmentId: number;
  departmentName: string;
  status: number;
}

export interface Asset {
  id: number;
  assetNo: string;
  name: string;
  category: string;
  price: number;
  purchaseTime: string;
  location: string;
  departmentId: number;
  userId: number | null;
  isShared: number;
  status: 1 | 2 | 3 | 4;
}

export interface CreateAssetReq {
  assetNo: string;
  name: string;
  category: string;
  price: number;
  purchaseTime: string;
  location: string;
  departmentId: number;
  isShared: number;
}

export interface WorkflowRequest {
  id: number;
  assetId: number;
  requesterId: number;
  departmentId: number;
  type: 1 | 2 | 3 | 4;
  currentStage: 1 | 2 | 3;
  status: 1 | 2 | 3;
  reason: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkflowLog {
  id: number;
  requestId: number;
  operatorId: number;
  action: string;
  comment: string;
  operateTime: string;
}

export interface WorkflowDetail {
  request: WorkflowRequest;
  logs: WorkflowLog[];
}

export interface InventoryTask {
  id: number;
  taskName: string;
  scopeDeptId: number;
  creatorId: number;
  startTime: string;
  endTime: string;
  status: 1 | 2 | 3;
  expectedAssetCount?: number;
  submittedCount?: number;
}

export interface ExpectedAsset {
  assetId: number;
  assetNo: string;
  name: string;
  bookLocation: string;
}

export interface SubmitItem {
  assetNo: string;
  modifiedCells: Record<string, string>;
  expectedUpdatedAt: string | null;
}

export interface SubmitResult {
  success: string[];
  conflicts: { assetNo: string; code: number; message: string }[];
  failures: { assetNo: string; code: number; message: string }[];
}

export interface DeptTreeNode {
  id: number;
  parentId: number;
  deptName: string;
  deptCode: string;
  path: string;
  children: DeptTreeNode[] | null;
}

export interface ExportJob {
  jobId: number;
  status: 0 | 1 | 2 | 3;
  downloadUrl: string | null;
  errorMessage: string | null;
}

export interface DeptStatItem {
  departmentId: number;
  departmentName: string;
  totalCount: number;
  inStockCount: number;
  inUseCount: number;
  totalValue: number;
}

export interface CategoryStatItem {
  category: string;
  count: number;
  totalValue: number;
}

export interface InventoryDiffSummary {
  matchCount: number;
  surplusCount: number;
  deficitCount: number;
  records: PaginatedData<InventoryRecord>;
}

export interface InventoryRecord {
  assetNo: string;
  name: string;
  bookLocation: string;
  actualLocation: string;
  diffStatus: 0 | 1 | 2 | 3;
}
```

---

## 8. 工具函数

### 8.1 utils/storage.ts

```typescript
const ACCESS_KEY = 'fams_access_token';
const REFRESH_KEY = 'fams_refresh_token';

export const storage = {
  getAccessToken: () => localStorage.getItem(ACCESS_KEY),
  setAccessToken: (t: string) => localStorage.setItem(ACCESS_KEY, t),
  getRefreshToken: () => localStorage.getItem(REFRESH_KEY),
  setRefreshToken: (t: string) => localStorage.setItem(REFRESH_KEY, t),
  clear: () => { localStorage.removeItem(ACCESS_KEY); localStorage.removeItem(REFRESH_KEY); },
};
```

### 8.2 utils/format.ts

```typescript
import dayjs from 'dayjs';

export const formatDate = (iso: string) => dayjs(iso).format('YYYY-MM-DD');
export const formatDateTime = (iso: string) => dayjs(iso).format('YYYY-MM-DD HH:mm');
export const formatPrice = (n: number) => `¥${n.toLocaleString('zh-CN', { minimumFractionDigits: 2 })}`;
```

### 8.3 utils/constants.ts

见 `01-design.md` §七，另加：

```typescript
export const ASSET_CATEGORIES = ['设备', '家具', '实验器材', '电子设备', '交通工具', '其他'] as const;

export const ROLE_HOME: Record<number, string> = {
  1: '/admin/dashboard',
  2: '/college/dashboard',
  3: '/user/assets',
};
```

---

## 9. 后端 API 状态（2026-07-08 已补全）

以下接口已在 `feat/frontend-api-gaps` 分支实现，详见 `backend/doc/12-frontend-api-gaps.md`：

| API | 状态 |
|---|---|
| `GET /user/users` | ✅ 已实现 |
| `GET /user/users/:id` | ✅ 已实现 |
| `GET /user/departments/college-subtree` | ✅ 已实现 |
| `GET /inventory/tasks` | ✅ 已实现 |
| `GET /inventory/tasks/:id` | ✅ 已实现 |
| `GET /asset/assets?scope=my` | ✅ 已实现 |
| `GET /asset/assets/shared`（is_shared 过滤） | ✅ 已修复 |
| `GET /workflow/requests?assetId=` | ✅ 已实现 |

---

## 10. 开发顺序（推荐）

按以下顺序实现，每步可独立验证：

| 步骤 | 内容 | 验证方式 |
|---|---|---|
| P1 | 项目脚手架 + 登录 + Token 管理 | `admin_school` 登录成功跳转 |
| P2 | 三个 Layout + 侧边栏 + 路由守卫 | 切换角色看到不同菜单 |
| P3 | 资产 CRUD（列表/详情/创建/编辑） | 新增资产后在列表可见 |
| P4 | 工单（创建/列表/审批 Drawer） | 完整领用→院审→校审流程 |
| P5 | 盘点（任务管理 + 可编辑 Table 录入） | 提交草稿 + 冲突标红 + 重进可见 |
| P6 | 报表 + 导出 | 图表渲染 + CSV 下载 |
| P7 | 组织树 + 用户管理 | 创建部门/用户 |

---

## 11. 响应数据处理约定

所有 RTK Query endpoint 需统一解包：

```typescript
// 在 endpoint 的 transformResponse 中
transformResponse: (response: ApiResponse<T>) => {
  if (response.code !== 0) {
    throw new Error(response.message);
  }
  return response.data;
},
```

Mutation 的错误处理：

```typescript
// 在组件中
const [createAsset] = useCreateAssetMutation();
try {
  await createAsset(body).unwrap();
  message.success('创建成功');
} catch (err: any) {
  // err 可能是 { code, message } 或网络错误
  message.error(err?.message || '操作失败');
}
```

---

## 12. 测试检查清单

| 场景 | 账号 | 预期 |
|---|---|---|
| 校级登录 | admin_school | 跳转 /admin/dashboard，看到 7 个菜单 |
| 院级登录 | admin_info | 跳转 /college/dashboard，看到 4 个菜单 |
| 师生登录 | student_001 | 跳转 /user/assets，看到 3+ 个菜单 |
| 越权访问 | student_001 访问 /admin/assets | 重定向到 /user/assets |
| 领用流程 | student_001 申请 → admin_info 初审 → admin_school 终审 | 资产状态变为领用中 |
| 盘点冲突 | student_001 + student_002 同时提交同一资产 | 后到者看到冲突标红 |
| Token 过期 | 等待 2h 或手动清除 accessToken | 自动 refresh 或跳转登录 |
| 登出 | 任意账号登出 | Token 失效，跳转登录 |

测试数据见 `backend/doc/05-seed-fixtures.md`。

---

*文档版本：v1.0 | 2026-07-08*
