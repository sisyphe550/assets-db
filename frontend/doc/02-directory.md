# FAMS 前端目录结构

---

## 1. 完整目录树

```
frontend/
├── doc/
│   ├── 01-design.md              # 技术选型、UI 规范、状态管理、路由设计、类型定义
│   ├── 02-directory.md           # 本文档：目录结构详细说明
│   ├── 03-pages.md               # 页面级详细设计（21 个页面）
│   ├── 04-backend-api.md         # 后端 API 参考（鉴权/端点/错误码/业务规则）
│   ├── 05-components.md          # 组件规格（Props/loading-empty-error/表单校验）
│   ├── 06-visual-design.md       # 美术设计规范（Design Tokens/线框图/图标）
│   └── 07-implementation.md      # AI 开发蓝图（依赖/配置/路由/Store/API）
│
├── public/
│   └── favicon.ico
│
├── index.html                     # Vite 入口 HTML
│
├── package.json
├── vite.config.ts                 # Vite 配置（proxy、插件等）
├── tsconfig.json
├── tsconfig.node.json
│
└── src/
    ├── main.tsx                   # ReactDOM.createRoot + Provider 包裹
    ├── App.tsx                    # RouterProvider + 路由树
    │
    ├── store/                     # ──────── Redux 状态管理 ────────
    │   ├── index.ts              #  configureStore + TypedUseSelectorHook
    │   ├── hooks.ts              #  useAppSelector / useAppDispatch
    │   │
    │   ├── api/                  #  RTK Query API endpoints
    │   │   ├── baseQuery.ts     #    fetchBaseQuery（注入 token，401 重试）
    │   │   ├── authApi.ts       #    login / refresh / logout / me
    │   │   ├── assetApi.ts      #    assets CRUD / list / shared
    │   │   ├── workflowApi.ts   #    requests CRUD / approve / reject
    │   │   ├── inventoryApi.ts  #    tasks CRUD / submit / archive / records
    │   │   ├── reportApi.ts     #    by-dept / by-category / diff / export
    │   │   └── userApi.ts       #    departments tree / users CRUD
    │   │
    │   └── slices/              #  Redux Toolkit Slices（客户端 UI 状态）
    │       ├── authSlice.ts     #    当前用户、token
    │       └── uiSlice.ts       #    sidebar 折叠、全局通知
    │
    ├── layouts/                   # ──────── 布局组件 ────────
    │   ├── AdminLayout.tsx       #  校级管理员布局
    │   ├── CollegeLayout.tsx     #  学院管理员布局
    │   └── UserLayout.tsx        #  普通师生布局
    │
    ├── components/                # ──────── 共享组件 ────────
    │   ├── auth/                 #  鉴权相关
    │   │   └── RequireAuth.tsx  #    路由守卫
    │   │
    │   ├── asset/                #  资产相关
    │   │   ├── AssetTable.tsx   #    资产列表表格（ProTable）
    │   │   ├── AssetForm.tsx    #    资产新增/编辑表单
    │   │   ├── AssetDetail.tsx  #    资产详情展示
    │   │   └── AssetPickerModal.tsx # 资产选择弹窗（工单创建用）
    │   │
    │   ├── workflow/             #  工单相关
    │   │   ├── WorkflowTable.tsx #   工单列表表格
    │   │   ├── WorkflowDetail.tsx#   工单详情（含审批时间线）
    │   │   ├── WorkflowCreateForm.tsx # 创建工单表单
    │   │   └── WorkflowTimeline.tsx   # 审批日志时间线
    │   │
    │   ├── inventory/            #  盘点相关
    │   │   ├── InventoryTaskTable.tsx# 盘点任务列表
    │   │   ├── InventoryTaskForm.tsx #  创建任务表单
    │   │   └── UniverSpreadsheet.tsx #  Univer 表格封装（关键组件）
    │   │
    │   ├── department/           #  组织架构
    │   │   └── DeptTreeSelect.tsx#   部门树选择器
    │   │
    │   ├── user/                 #  用户管理
    │   │   └── CreateUserModal.tsx #  创建用户弹窗
    │   │
    │   ├── report/               #  报表
    │   │   ├── StatCard.tsx     #    统计卡片
    │   │   ├── DiffSummary.tsx  #    盘点差异汇总
    │   │   ├── ReportCharts.tsx #    报表图表组
    │   │   └── ExportModal.tsx  #    导出进度弹窗
    │   │
    │   └── common/               #  通用组件
    │       ├── TopHeader.tsx    #    顶栏
    │       ├── SidebarMenu.tsx  #    侧边栏菜单
    │       ├── PageHeader.tsx   #    页面标题 + 面包屑
    │       ├── StatusTag.tsx    #    状态标签（资产/工单/盘点）
    │       └── ErrorBoundary.tsx#    错误边界
    │
    ├── pages/                     # ──────── 页面组件 ────────
    │   ├── login/
    │   │   └── LoginPage.tsx
    │   │
    │   ├── admin/                #  role=1
    │   │   ├── DashboardPage.tsx
    │   │   ├── AssetListPage.tsx
    │   │   ├── AssetDetailPage.tsx
    │   │   ├── AssetCreatePage.tsx
    │   │   ├── WorkflowTodoPage.tsx
    │   │   ├── WorkflowListPage.tsx
    │   │   ├── WorkflowDetailPage.tsx
    │   │   ├── DepartmentPage.tsx
    │   │   ├── UserManagePage.tsx
    │   │   ├── InventoryTaskListPage.tsx
    │   │   ├── InventoryTaskCreatePage.tsx
    │   │   ├── InventoryTaskDetailPage.tsx
    │   │   └── ReportPage.tsx
    │   │
    │   ├── college/              #  role=2
    │   │   ├── DashboardPage.tsx
    │   │   ├── AssetListPage.tsx
    │   │   ├── AssetDetailPage.tsx
    │   │   ├── WorkflowTodoPage.tsx
    │   │   ├── InventoryTaskListPage.tsx
    │   │   ├── InventoryTaskCreatePage.tsx
    │   │   └── InventoryTaskDetailPage.tsx
    │   │
    │   └── user/                 #  role=3
    │       ├── MyAssetsPage.tsx
    │       ├── WorkflowMyPage.tsx
    │       ├── WorkflowCreatePage.tsx
    │       └── InventorySubmitPage.tsx
    │
    │   └── NotFoundPage.tsx      #   404 页面（pages 根级）
    │
    ├── config/                    # ──────── 配置 ────────
    │   └── menu.ts               #   三角色侧边栏菜单定义
    │
    ├── styles/                    # ──────── 样式 ────────
    │   └── tokens.ts             #   Design Tokens（色彩/间距）
    │
    ├── hooks/                     # ──────── 自定义 Hooks ────────
    │   ├── useAuth.ts            #   登录/登出/刷新 token 逻辑
    │   └── usePermission.ts      #   便捷权限判断
    │
    ├── utils/                     # ──────── 工具 ────────
    │   ├── constants.ts          #   枚举映射（status/type/role → 中文 + 颜色）
    │   ├── format.ts             #   日期/价格格式化
    │   └── storage.ts            #   localStorage 封装（token 存取）
    │
    └── types/                     # ──────── 类型 ────────
        ├── api.ts                 #   API 请求/响应类型
        └── common.ts              #   通用类型
```

---

## 2. 各目录职责分工

### 2.1 `pages/` vs `components/` 的分界

| 目录 | 职责 | 示例 |
|---|---|---|
| `pages/` | **页面入口**，负责组合组件、调用 API、处理 URL 参数 | `AssetListPage.tsx` 调 `useGetAssetsQuery`，传数据给 `AssetTable` |
| `components/` | **纯 UI 组件**，只通过 props 接收数据，不直接调 API | `AssetTable.tsx` 接收 `dataSource` prop 渲染 ProTable |

这样分的好处：`AssetTable` 可以被 `admin/AssetListPage` 和 `college/AssetListPage` 共同使用，而页面只需传不同的查询参数。

### 2.2 角色 Layout 的异同

```
 AdminLayout            CollegeLayout           UserLayout
 ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
 │ 顶栏: 用户名+退出  │    │ 顶栏: 用户名+退出  │    │ 顶栏: 用户名+退出  │
 ├────────┬────────┤    ├────────┬────────┤    ├────────┬────────┤
 │ 侧边栏 │ 内容区  │    │ 侧边栏 │ 内容区  │    │ 侧边栏 │ 内容区  │
 │        │        │    │        │        │    │        │        │
 │ 仪表盘 │        │    │ 仪表盘 │        │    │ 我的资产│        │
 │ 资产管理│        │    │ 资产管理│        │    │ 我的工单│        │
 │ 工单审批│        │    │ 工单审批│        │    │ 新建申请│        │
 │ 组织架构│        │    │ 盘点管理│        │    │ 盘点录入│        │
 │ 用户管理│        │    │        │        │    │        │        │
 │ 盘点管理│        │    │        │        │    │        │        │
 │ 统计报表│        │    │        │        │    │        │        │
 └────────┴────────┘    └────────┴────────┘    └────────┴────────┘
```

**实现方式**：三个 Layout 的顶栏完全一样（同一个 `<TopHeader />` 组件）。侧边栏根据角色渲染不同的 `<Menu>` 项。内容区是 `<Outlet />`。

### 2.3 盘点表格（`InventorySpreadsheet.tsx`，生产）

生产盘点详情页使用 Ant Design 可编辑 Table，封装于 `InventorySpreadsheet.tsx`：

```
Props:
  - rows: InventoryRow[]           # 合并 expected + 草稿
  - readOnly: boolean
  - onRowsChange / onSubmitResult

职责:
  1. 只读列：资产编号、名称、账面位置
  2. 可编辑列：实际位置、备注
  3. 添加盘盈行、批量提交 → POST /tasks/:id/submit
  4. 冲突行标红、成功行标绿
```

**挂载门控**：`InventoryTaskDetailView` 在 `rowsReady` 后才 lazy-load 表格（`key={taskId}`）。

**实验组件** `UniverSpreadsheet.tsx`：Univer 0.5.5 封装，未接入主流程。详见 `doc/dev-log/010-fix-inventory-spreadsheet.md`。

---

## 3. 构建流程

### 3.1 开发

```bash
cd frontend/
npm install
npm run dev     # Vite 开发服务器 (localhost:5173)
                # API 请求通过 vite.config.ts proxy 转发到后端
```

### 3.2 生产构建

```bash
npm run build   # 输出到 dist/
                # 部署时由 Nginx 托管静态文件
```

### 3.3 Nginx 生产部署

```nginx
# 前端静态文件
location / {
    root /app/frontend/dist;
    try_files $uri /index.html;   # SPA 路由 fallback
}

# 后端 API 反向代理
location /api/ {
    proxy_pass http://fams-gateway;  # 后端 Nginx 网关
}
```

---

*文档版本：v1.1 | 2026-07-08*
