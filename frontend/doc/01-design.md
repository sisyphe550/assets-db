# FAMS 前端设计文档

> 对应后端文档 `backend/doc/01-desgin.md`

---

## 一、技术选型与理由

| 层级 | 选择 | 理由 |
|---|---|---|
| 框架 | React 18 + TypeScript | 生态成熟，Ant Design 深度绑定 React |
| 构建 | Vite | 开发服务器秒启，HMR 快，ESBuild 打包 |
| UI 库 | Ant Design 5 | 表格/表单/树/Tabs 开箱即用，ProTable 天然适配 CRUD 后台 |
| 状态管理 | Redux Toolkit + RTK Query | 服务端缓存自动化，API loading/error 状态不需要手写 |
| 路由 | React Router v6 | layout route + nested routes 用角色划分 |
| 表格 | Univer | 盘点模块类 Excel 协同录入（Luckysheet 后继项目） |
| HTTP | `fetch` (RTK Query 内置) | 不需要 axios，RTK Query 的 `createApi` 自带 |
| 样式 | Ant Design 内置 + CSS Modules | 避免引入额外 CSS 方案 |

### 决策记录（已定稿）

以下决策经讨论确认，不再变更：

| # | 决策项 | 选择 | 确认日期 |
|---|---|---|---|
| 1 | 框架 | React 18 + TypeScript | 2026-07-07 |
| 2 | UI 库 | Ant Design 5 | 2026-07-07 |
| 3 | 状态管理 | Redux Toolkit + RTK Query | 2026-07-07 |
| 4 | 路由策略 | 三个角色分开路由 | 2026-07-07 |
| 5 | 后端对接 | 直连（Vite proxy），MSW 按需补充 | 2026-07-07 |
| 6 | 盘点表格 | Univer（直接上，不做简化版） | 2026-07-07 |
| 7 | 代码位置 | `./frontend/` | 2026-07-07 |

**为什么不用 Vue？**
后端团队（你）用 Go，前端用 React + TypeScript 类型系统更严谨，Ant Design 的 React 版本比 Element Plus 组件更丰富。且 Univer 的 React 封装更成熟。

**为什么用 RTK Query 而不是手写 fetch？**
系统有大量"列表→详情→操作→刷新列表"的循环。手写意味着每个页面都要管理 loading/error/data 三种状态 + useEffect + 缓存失效。RTK Query 的 tag-based invalidation 可以声明式地："创建资产后，自动刷新资产列表"。

**为什么不用 Next.js？**
系统是纯客户端渲染（CSR）的后台管理，没有 SEO 需求。Vite 比 Next.js 更轻更快。如果后续需要 SSR，可以迁移。

---

## 二、状态管理设计

### 2.1 Store 划分

```
store/
├── api/                    # RTK Query API slices（自动管理服务端数据缓存）
│   ├── authApi.ts          #   login / refresh / logout / me
│   ├── assetApi.ts         #   CRUD / list / shared
│   ├── workflowApi.ts      #   create / list / approve / reject / detail
│   ├── inventoryApi.ts     #   tasks / submit / archive / records
│   ├── reportApi.ts        #   by-dept / diff / export
│   └── userApi.ts          #   departments / users CRUD
│
└── slices/                 # Redux Toolkit slices（客户端 UI 状态）
    ├── authSlice.ts        #   accessToken / refreshToken / currentUser
    ├── notificationSlice.ts #  全局消息提示
    └── uiSlice.ts          #   sidebar 折叠 / 主题
```

### 2.2 为什么这样划分？

**api/ 和 slices/ 严格分离**：
- `api/` 中的代码只做"从后端拿数据、缓存数据、失效数据"——RTK Query 负责
- `slices/` 中的代码只做"UI 状态"——当前用户、全局通知、侧边栏展开

这种分离让你不会把"从 API 获取资产列表"和"用户点击了一个按钮后 UI 变化"混在一起。

### 2.3 RTK Query tag-based invalidation 示例

```typescript
// store/api/assetApi.ts
export const assetApi = createApi({
  reducerPath: 'assetApi',
  tagTypes: ['Asset', 'AssetList'],
  endpoints: (builder) => ({
    getAssets: builder.query<AssetListResponse, AssetListParams>({
      query: (params) => `/asset/assets?${new URLSearchParams(params)}`,
      providesTags: ['AssetList'],           // 列表数据
    }),
    getAsset: builder.query<Asset, number>({
      query: (id) => `/asset/assets/${id}`,
      providesTags: (_result, _error, id) => [{ type: 'Asset', id }], // 单条缓存
    }),
    createAsset: builder.mutation<void, CreateAssetReq>({
      query: (body) => ({ url: '/asset/assets', method: 'POST', body }),
      invalidatesTags: ['AssetList'],        // 创建后刷新列表
    }),
    updateAsset: builder.mutation<void, { id: number; body: UpdateAssetReq }>({
      query: ({ id, body }) => ({ url: `/asset/assets/${id}`, method: 'PUT', body }),
      invalidatesTags: (_result, _error, { id }) => ['AssetList', { type: 'Asset', id }],
    }),
  }),
});
```

关键机制：`createAsset` 成功后自动触发 `getAssets` 重新请求（因为 `invalidatesTags: ['AssetList']`）。

### 2.4 Auth Slice 结构

```typescript
interface AuthState {
  accessToken: string | null;       // 存在 localStorage，同时放 store 供 RTK Query 读取
  refreshToken: string | null;      // 存在 localStorage
  user: {                           // 来自 GET /me
    id: number;
    username: string;
    realName: string;
    roleLevel: 1 | 2 | 3;          // ← 前端所有权限判断基于此字段
    departmentId: number;
    departmentName: string;
  } | null;
  isAuthenticated: boolean;
}
```

**Token 生命周期**：
1. 登录成功 → accessToken + refreshToken 存 localStorage + store
2. 每次请求前，RTK Query 的 `baseQuery` 从 store 读取 token 注入 `Authorization` header
3. 收到 40101 → 尝试用 refreshToken 刷新 → 成功则重试请求，失败则清空 token 跳转登录页

---

## 三、路由设计

### 3.1 路由结构

```
/login                        # 登录页

/admin                         # role=1 Layout（侧边栏含全校管理菜单）
  /admin/dashboard             #  仪表盘
  /admin/assets                #  资产管理
  /admin/assets/:id            #  资产详情
  /admin/assets/create         #  新增资产
  /admin/workflow/todo         #  待审批
  /admin/workflow/all          #  全部工单
  /admin/workflow/:id          #  工单详情（含审批操作）
  /admin/departments           #  组织架构
  /admin/users                 #  用户管理
  /admin/inventory/tasks       #  盘点任务
  /admin/inventory/tasks/create#  创建任务
  /admin/inventory/tasks/:id   #  盘点详情（Univer 表格 + 差异报告）
  /admin/reports               #  统计报表

/college                       # role=2 Layout（侧边栏含本院管理菜单）
  /college/dashboard           #  本院仪表盘
  /college/assets              #  本院资产
  /college/assets/:id          #  资产详情
  /college/workflow/todo       #  待审批（院级初审）
  /college/inventory/tasks     #  盘点任务
  /college/inventory/tasks/create
  /college/inventory/tasks/:id

/user                          # role=3 Layout（侧边栏含个人信息菜单）
  /user/assets                 #  我的资产（已领用 + 共享）
  /user/workflow/my            #  我的工单
  /user/workflow/create        #  新建申请
  /user/inventory/:taskId      #  盘点录入
```

### 3.2 为什么三个角色分开路由？

| 方案 | 优点 | 缺点 |
|---|---|---|
| 分开路由 | sidebar 菜单完全不同，权限边界清晰，代码好维护 | 有一些重复页面（如资产详情三个角色都访问同一组件） |
| 统一路由 + permission | 代码量少 | sidebar 变化时组件逻辑复杂，权限判断散落各处 |

选分开路由，**但共享底层组件**。例如：
- `/admin/assets/:id`、`/college/assets/:id` 都渲染同一个 `<AssetDetail />` 组件
- 区别只在于路由 Layout 的侧边栏和顶层权限守卫

### 3.3 路由守卫

```typescript
// 每个 Layout Route 入口处
function RequireRole({ role, children }: { role: number; children: React.ReactNode }) {
  const user = useAppSelector(selectCurrentUser);
  if (!user) return <Navigate to="/login" />;
  if (user.roleLevel < role) return <Navigate to={`/${getRolePath(user.roleLevel)}`} />;
  return children;
}
```

---

## 四、目录结构设计

```
frontend/
├── doc/                           # 设计文档
│   ├── 01-design.md               #  本文档
│   ├── 02-directory.md            #  目录结构详细说明
│   └── 03-pages.md                #  页面级详细设计
│
├── public/                        # 静态资源
├── index.html
├── vite.config.ts
├── tsconfig.json
├── package.json
│
├── src/
│   ├── main.tsx                   # 入口
│   ├── App.tsx                    # 路由根组件
│   ├── vite-env.d.ts
│   │
│   ├── store/                     # Redux 状态管理
│   │   ├── index.ts              #   store 配置
│   │   ├── api/                  #   RTK Query API
│   │   │   ├── baseQuery.ts     #     统一 baseQuery（注入 token、处理 401）
│   │   │   ├── authApi.ts
│   │   │   ├── assetApi.ts
│   │   │   ├── workflowApi.ts
│   │   │   ├── inventoryApi.ts
│   │   │   ├── reportApi.ts
│   │   │   └── userApi.ts
│   │   └── slices/               #   Redux Toolkit Slices
│   │       ├── authSlice.ts
│   │       └── uiSlice.ts
│   │
│   ├── layouts/                   # 布局组件
│   │   ├── AdminLayout.tsx       #   校级管理员布局（侧边栏 + 顶栏 + 内容区）
│   │   ├── CollegeLayout.tsx     #   学院管理员布局
│   │   └── UserLayout.tsx        #   普通师生布局
│   │
│   ├── pages/                     # 页面组件（按角色分）
│   │   ├── login/                #   登录页（所有角色共用）
│   │   ├── admin/                #   校级管理员页面
│   │   ├── college/              #   学院管理员页面
│   │   └── user/                 #   普通师生页面
│   │
│   ├── components/                # 共享组件（跨角色）
│   │   ├── AssetTable.tsx        #   资产表格（admin/college 复用）
│   │   ├── AssetForm.tsx         #   资产表单（新增/编辑复用）
│   │   ├── AssetDetail.tsx       #   资产详情（所有角色复用）
│   │   ├── WorkflowDetail.tsx    #   工单详情（含审批操作按钮）
│   │   ├── WorkflowCreateForm.tsx#   创建工单表单
│   │   ├── DeptTree.tsx          #   组织树选择器
│   │   ├── UserSelect.tsx        #   用户搜索选择器
│   │   └── UniverSpreadsheet.tsx #   Univer 表格封装
│   │
│   ├── hooks/                     # 自定义 hooks
│   │   ├── useAuth.ts            #   登录/登出/刷新 token
│   │   └── usePermission.ts      #   权限判断便捷 hook
│   │
│   ├── utils/                     # 工具函数
│   │   ├── constants.ts          #   枚举映射（status/type 等 → 中文）
│   │   ├── request.ts            #   fetch 封装（如有特殊需求）
│   │   └── storage.ts            #   localStorage 封装
│   │
│   └── types/                     # TypeScript 类型定义
│       ├── api.ts                 #   API 请求/响应类型（对齐 03-api-contract.md）
│       └── common.ts              #   通用类型（分页、统一响应包裹等）
```

---

## 五、TypeScript 类型（对齐后端契约）

```typescript
// types/api.ts — 与 backend/doc/03-api-contract.md 严格对齐

// 统一响应包裹
interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// 分页
interface PaginatedData<T> {
  list: T[];
  page: number;
  pageSize: number;
  total: number;
}

// 用户
interface UserInfo {
  id: number;
  username: string;
  realName: string;
  roleLevel: 1 | 2 | 3;
  departmentId: number;
  departmentName: string;
  status: number;
}

// 资产
interface Asset {
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

// 工单
interface WorkflowRequest {
  id: number;
  assetId: number;
  requesterId: number;
  departmentId: number;
  type: 1 | 2 | 3 | 4;    // 1-领用 2-归还 3-报修 4-报废
  currentStage: 1 | 2 | 3; // 1-院审 2-校审 3-归档
  status: 1 | 2 | 3;       // 1-审批中 2-通过 3-驳回
  reason: string;
  createdAt: string;
  updatedAt: string;
}

interface WorkflowLog {
  id: number;
  requestId: number;
  operatorId: number;
  action: string;
  comment: string;
  operateTime: string;
}

// 盘点
interface InventoryTask {
  id: number;
  taskName: string;
  scopeDeptId: number;
  creatorId: number;
  startTime: string;
  endTime: string;
  status: 1 | 2 | 3;  // 1-进行中 2-已归档比对中 3-已完成
}

interface SubmitItem {
  assetNo: string;
  modifiedCells: Record<string, any>;
  expectedUpdatedAt: string | null;
}

// 部门树节点
interface DeptTreeNode {
  id: number;
  parentId: number;
  deptName: string;
  deptCode: string;
  path: string;
  children: DeptTreeNode[];
}
```

---

## 六、枚举映射（前端展示用）

```typescript
// utils/constants.ts

export const ASSET_STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: '在库', color: 'green' },
  2: { label: '领用中', color: 'blue' },
  3: { label: '维修中', color: 'orange' },
  4: { label: '已报废', color: 'red' },
};

export const WORKFLOW_TYPE_MAP: Record<number, string> = {
  1: '领用',
  2: '归还',
  3: '报修',
  4: '报废',
};

export const WORKFLOW_STATUS_MAP: Record<number, { label: string; color: string }> = {
  1: { label: '审批中', color: 'processing' },
  2: { label: '已通过', color: 'success' },
  3: { label: '已驳回', color: 'error' },
};

export const WORKFLOW_STAGE_MAP: Record<number, string> = {
  1: '待院级初审',
  2: '待校级复审',
  3: '已归档',
};

export const ROLE_MAP: Record<number, string> = {
  1: '校级管理员',
  2: '学院管理员',
  3: '普通师生',
};

export const INVENTORY_DIFF_MAP: Record<number, { label: string; color: string }> = {
  0: { label: '未比对', color: 'default' },
  1: { label: '相符', color: 'success' },
  2: { label: '盘盈', color: 'warning' },
  3: { label: '盘亏', color: 'error' },
};
```

---

## 七、与其他系统的集成

| 系统 | 集成方式 |
|---|---|
| 后端 API | 直连 `localhost:8888-8892`（通过 RTK Query `baseQuery` 配置 baseUrl） |
| Univer 表格 | `@univerjs/core` + `@univerjs/presets`，盘点页面按需加载 |
| localhost 跨域 | 后端各服务需添加 CORS header，或开发环境用 Vite proxy |

### Vite 开发代理配置（避免跨域）

```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api/v1/user':    'http://localhost:8888',
      '/api/v1/asset':   'http://localhost:8889',
      '/api/v1/workflow':'http://localhost:8890',
      '/api/v1/inventory':'http://localhost:8891',
      '/api/v1/report':  'http://localhost:8892',
    },
  },
});
```

---

*文档版本：v1.0 | 2026-07-07*
