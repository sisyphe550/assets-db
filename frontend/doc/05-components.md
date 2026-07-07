# FAMS 前端组件规格

> 每个组件的 props、状态（loading/empty/error）、表单校验规则

---

## 1. 通用规范

### 1.1 每次 API 调用的三种状态

每个调用 RTK Query endpoint 的组件必须处理以下三种状态：

```typescript
const { data, isLoading, isError, error } = useGetAssetsQuery(params);

if (isLoading) return <Spin tip="加载中..." size="large" />;       // 居中的 loading 动画
if (isError)   return <Result status="error" title="加载失败"      // 错误时用 Ant Design Result
                              subTitle={error.message}
                              extra={<Button onClick={refetch}>重试</Button>} />;
if (!data?.list?.length) return <Empty description="暂无数据" />;  // 空列表用 Ant Design Empty
```

| 状态 | 组件 | 位置 |
|---|---|---|
| Loading | `<Spin tip="加载中..." size="large">` | 页面正中 |
| Error | `<Result status="error" ... extra={<Button>重试</Button>}>` | 页面正中 |
| Empty | `<Empty description="暂无数据" />` | 页面正中 |
| 详情 404 | `<Result status="404" title="资源不存在" extra={<Button>返回列表</Button>}>` | 页面正中 |

### 1.2 Toast / Notification 规范

| 操作 | 方式 | 位置 |
|---|---|---|
| 创建成功 | `message.success('创建成功')` | 页面顶部居中 |
| 编辑成功 | `message.success('保存成功')` | 页面顶部居中 |
| 删除成功 | `message.success('已删除')` | 页面顶部居中 |
| 操作失败 | `message.error(res.message)` | 页面顶部居中 |
| 网络异常 | `notification.error({ message: '网络异常', description: '...' })` | 右上角 |

### 1.3 按钮状态

| 状态 | 表现 |
|---|---|
| 默认 | 正常可点击 |
| 提交中 | `loading={true}` + disabled |
| 不可操作 | `disabled` + Tooltip 说明原因 |

---

## 2. 各页面组件规格

### 2.1 LoginPage

| 属性 | 规格 |
|---|---|
| 文件 | `pages/login/LoginPage.tsx` |
| 宽度 | 400px 居中卡片 |
| 背景 | `#f0f2f5` 全屏 |
| Logo | 文字"高校固定资产管理系统" |

**表单字段**：

| 字段 | 组件 | 校验规则 |
|---|---|---|
| username | `<Input placeholder="用户名/工号">` | `required: '请输入用户名'` |
| password | `<Input.Password placeholder="密码">` | `required: '请输入密码', min: 6` |

**状态**：
- Loading：登录按钮显示 `loading` + 禁用表单
- Error (40101)：`message.error('用户名或密码错误')`
- Error (40301)：`message.error('账户已被禁用，请联系管理员')`
- 网络错误：`message.error('网络异常，请稍后重试')`

---

### 2.2 AssetTable（资产列表表格）

| 属性 | 规格 |
|---|---|
| 组件 | `<ProTable>` |
| 文件 | `components/asset/AssetTable.tsx` |

**Props**：

```typescript
interface AssetTableProps {
  roleLevel: 1 | 2 | 3;       // 当前用户角色
  departmentId?: number;       // role=2 时限制部门
  onCreate?: () => void;       // 新增按钮回调（role≤2 时显示）
}
```

**列定义**：

| 列 | 宽度 | 渲染 |
|---|---|---|
| assetNo | 160 | `<Typography.Text copyable>` |
| name | 150 | 普通文本 |
| category | 100 | `<Tag>` |
| price | 100 | `¥{price.toLocaleString()}` |
| location | 150 | 普通文本 |
| departmentId | — | role=1 显示，其他隐藏 |
| status | 80 | `<StatusTag status={status} />` |
| userId | 100 | 领用人姓名（如有） |
| 操作 | 100 | "详情" `<Button type="link">` |

**搜索栏**（ProTable 内置）：

| 搜索项 | 组件 | 说明 |
|---|---|---|
| category | `<Select>` | 设备/家具/实验器材等 |
| status | `<Select>` | 在库/领用中/维修中/已报废 |
| keyword | `<Input.Search>` | 模糊匹配 assetNo + name |

**状态**：
- Loading：ProTable 自带 skeleton
- Empty：ProTable `locale.emptyText` → "暂无资产数据"
- Error：表格区域替换为 `<Result>` + 重试按钮

---

### 2.3 AssetForm（资产创建/编辑）

| 属性 | 规格 |
|---|---|
| 组件 | 新页面，非弹窗 |
| 文件 | `pages/admin/AssetCreatePage.tsx`、`pages/admin/AssetDetailPage.tsx` |

**表单字段**：

| 字段 | 组件 | 校验规则 |
|---|---|---|
| assetNo | `<Input>` | `required`, `pattern: /^[A-Z]+-\d{4}-\d{4}$/` |
| name | `<Input>` | `required`, `max: 50` |
| category | `<Select>` | `required` |
| price | `<InputNumber>` | `required`, `min: 0`, `precision: 2`, 前缀 ¥ |
| purchaseTime | `<DatePicker>` | `required` |
| location | `<Input>` | `required`, `max: 100` |
| departmentId | `<TreeSelect>` | `required`（role=1 时显示，role=2 默认本院） |
| isShared | `<Switch>` | 默认关闭 |

**状态**：
- 编辑模式：表单回填已有数据（通过 `useGetAssetQuery(id)` 获取）
- 提交中：保存按钮 `loading`
- 40903（assetNo 重复）：`form.setFields([{ name: 'assetNo', errors: ['该资产编号已存在'] }])`

---

### 2.4 WorkflowCreateForm（创建工单）

| 属性 | 规格 |
|---|---|
| 组件 | 新页面 |
| 文件 | `pages/user/WorkflowCreatePage.tsx` |

**表单字段**：

| 字段 | 组件 | 校验规则 |
|---|---|---|
| type | `<Radio.Group>` 按钮样式 | `required: '请选择申请类型'` |
| assetId | 资产选择器（弹窗表格） | `required: '请选择资产'` |
| reason | `<Input.TextArea rows={4}>` | `required: '请填写申请原因'`, `max: 200` |

**资产选择器逻辑**：

```typescript
// 根据 type 动态过滤可选资产
function getAssetFilter(type: number, userId: number) {
  switch (type) {
    case 1: return { status: 1 };                       // 在库
    case 2: return { status: 2, userId: userId };       // 领用中 + 本人
    case 3: return { status__in: [1, 2] };              // 在库或领用中
    case 4: return { status__in: [1, 3] };              // 在库或维修中
  }
}
```

**交互**：
1. 选择 type → assetId 选择器过滤可选资产
2. 选择资产 → 显示资产基本信息（名称/地点/状态）确认卡片
3. 填写 reason → 提交
4. 40902 冲突 → `message.error('该资产已有审批中的工单，请选择其他资产')`
5. 42201 拒绝 → `message.error(data.message)` 显示后端返回的拒绝原因

---

### 2.5 WorkflowDetail（工单详情 + 审批）

| 属性 | 规格 |
|---|---|
| 组件 | `<Drawer>` 右侧滑出 |
| 文件 | `components/workflow/WorkflowDetail.tsx` |
| 宽度 | 640px |

**Props**：

```typescript
interface WorkflowDetailProps {
  requestId: number;
  open: boolean;
  onClose: () => void;
}
```

**内容结构**：
1. 工单基本信息（`<Descriptions>` 两列布局）
2. 关联资产信息（卡片，含链接跳转资产详情）
3. 审批时间线（`<Timeline>`）
4. 操作区（仅对应当前审批阶段显示）

**审批按钮显示逻辑**：

```typescript
function canAct(user: UserInfo, request: WorkflowRequest): Action | null {
  if (request.status !== 1) return null;           // 已归档/驳回
  if (user.roleLevel === 2 && request.currentStage === 1) return 'approve_or_reject';
  if (user.roleLevel === 1 && request.currentStage === 2) return 'approve_or_reject';
  return null;
}
```

**操作 UI**：
```
┌────────────────────────────────┐
│ 审批意见: [________________]   │
│                         [驳回] │
│              [同意(主要按钮)]  │
└────────────────────────────────┘
```
- 同意：绿色主按钮，点击弹出二次确认弹窗
- 驳回：红色次按钮，点击弹出确认弹窗（必填审批意见）
- 操作后关闭 Drawer，列表自动刷新（RTK Query `invalidatesTags`）

---

### 2.6 WorkflowTable（工单列表）

| 属性 | 规格 |
|---|---|
| 组件 | `<ProTable>` |
| 文件 | `components/workflow/WorkflowTable.tsx` |

**Props**：

```typescript
interface WorkflowTableProps {
  scope: 'my' | 'todo' | 'all';  // 我的申请 / 待审批 / 全部
  roleLevel: 1 | 2 | 3;
}
```

**列定义**：

| 列 | 宽度 | 渲染 |
|---|---|---|
| id | 80 | `#${id}` |
| assetId | 100 | 链接到资产详情 |
| type | 80 | `<Tag>` 领用/归还/报修/报废 |
| reason | 200 | 截断 + Tooltip |
| requesterId | 100 | 申请人姓名 |
| status | 80 | `<StatusTag>` |
| currentStage | 100 | 当前阶段 |
| createdAt | 160 | 格式化时间 |
| 操作 | 80 | "详情" → 打开 Drawer |

**状态**：
- scope=todo 且无待审批项 → Empty "暂无待审批工单"
- scope=my 且无记录 → Empty "暂无申请记录" + 引导按钮"新建申请"

---

### 2.7 InventoryTaskForm（创建盘点任务）

| 属性 | 规格 |
|---|---|
| 组件 | 新页面 |
| 文件 | `pages/admin/InventoryTaskCreatePage.tsx` |

**表单字段**：

| 字段 | 组件 | 校验规则 |
|---|---|---|
| taskName | `<Input>` | `required`, `max: 50` |
| scopeDeptId | `<TreeSelect>` | `required: '请选择盘点范围'` |
| startTime | `<DatePicker showTime>` | `required` |
| endTime | `<DatePicker showTime>` | `required`, 自定义校验 `endTime > startTime` |
| assigneeIds | `<Select mode="multiple">` | `required: '请至少选择一名盘点员'` |

**assigneeIds 数据源**：调用 `GET /user/departments/tree` 获取 scope 内的用户列表

**状态**：
- 42203（时间窗无效）：`form.setFields([{ name: 'endTime', errors: ['结束时间必须晚于开始时间'] }])`
- 40302（scope 越权）：`message.error('盘点范围超出您的管辖范围')`

---

### 2.8 UniverSpreadsheet（盘点表格封装）

| 属性 | 规格 |
|---|---|
| 组件 | 懒加载 `React.lazy(() => import('./UniverSpreadsheet'))` |
| 文件 | `components/inventory/UniverSpreadsheet.tsx` |
| 加载中 | `<Skeleton active paragraph={{ rows: 10 }} />` |

**Props**：

```typescript
interface UniverSpreadsheetProps {
  taskId: number;                        // 盘点任务 ID
  expectedAssets: ExpectedAsset[];        // GET /tasks/:id/expected-assets 结果
  readOnly: boolean;                      // task.status !== 1 时锁定
  onSubmitResult?: (result: SubmitResult) => void;  // 提交结果回调（标红冲突行）
}
```

**初始化**：
1. 创建 Univer 工作簿
2. 设置首列"资产编号"、第二列"资产名称"、第三列"账面位置"为只读
3. 设置"实际位置"、"备注"列可编辑
4. 可选："盘盈资产"行（手动添加新行，assetNo 为未知编号）

**提交**：
1. 遍历所有修改过的行 → 组装 `SubmitItem[]`
2. 调用 `POST /tasks/:id/submit`
3. 成功行 → 绿色背景
4. 冲突行 → `rowClassName` 标红 + `Alert` 显示冲突原因
5. 失败行 → `Alert type="warning"` 显示失败原因

**内存管理**：组件卸载时 `univer.dispose()` 释放内存

---

### 2.9 DashboardPage（仪表盘）

| 属性 | 规格 |
|---|---|
| 组件 | 页面 |
| 文件 | `pages/admin/DashboardPage.tsx` |

**布局**：

```
Row gutter={[16, 16]}
  Col span={6} × 4  → 统计卡片
Row gutter={[16, 16]}
  Col span={12}      → 按类别饼图
  Col span={12}      → 按部门柱状图
Row gutter={[16, 16]}
  Col span={24}      → 最近工单（ProTable，前 10 条，无分页）
```

**统计卡片**：

| 卡片 | API | 数据字段 |
|---|---|---|
| 资产总数 | `GET /report/assets/by-dept` | `sum(totalCount)` |
| 在库资产 | 同上 | `sum(inStockCount)` |
| 领用中 | 同上 | `sum(inUseCount)` |
| 待审批 | `GET /workflow/requests?scope=todo&pageSize=1` | `data.total` |

每张卡片的 `<Statistic>` 配置：`valueStyle={{ fontSize: 32 }}`，带 `prefix={<Icon />}`

---

### 2.10 DepartmentPage（组织架构管理）

| 属性 | 规格 |
|---|---|
| 组件 | 页面 |
| 文件 | `pages/admin/DepartmentPage.tsx` |
| 只读（role≠1） | 隐藏新增/编辑按钮 |

**结构**：左侧 `<Tree>` + 右侧选中节点详情

```
┌──────────────────┬──────────────────────────────┐
│ 组织树（Tree）    │ 选中节点详情（Descriptions）  │
│                  │                              │
│  ├ 本校          │ 名称: 信息工程学院            │
│  │ ├ 信息工程学院 │ 代码: INFO                   │
│  │ │ ├ 软件实验室 │ 路径: /1/15/                 │
│  │ │ └ 网络实验室 │ [编辑] [新增子部门]          │
│  │ └ 机械工程学院 │                              │
│                  │                              │
└──────────────────┴──────────────────────────────┘
```

**状态**：
- Loading：Tree 区域 `<Skeleton>` + 详情区域空
- Empty：不应该出现（至少有"本校"根节点）
- 新增子部门：在选中节点下弹出 `<Modal>` 表单

---

### 2.11 UserManagePage（用户管理）

| 属性 | 规格 |
|---|---|
| 组件 | 页面 |
| 文件 | `pages/admin/UserManagePage.tsx` |

**表格**（ProTable）：

| 列 | 渲染 |
|---|---|
| username | 普通文本 |
| realName | 普通文本 |
| roleLevel | `<Tag>` 校级/院级/师生 |
| departmentId | 部门名称 |
| status | `<Switch checked={status===1}>` 启用/禁用 |
| 操作 | "强制下线" `<Button danger type="link">` |

**新增用户弹窗**（`<Modal>` 表单）：

| 字段 | 组件 | 校验规则 |
|---|---|---|
| username | `<Input>` | `required`, `pattern: /^[a-z0-9_]{3,20}$/` |
| password | `<Input.Password>` | `required`, `min: 6` |
| realName | `<Input>` | `required`, `max: 20` |
| roleLevel | `<Select>` | `required`（role=2 时只能选 3） |
| departmentId | `<TreeSelect>` | `required`（role=2 时限制本院子树） |

---

## 3. 表单校验规则汇总

| 页面 | 字段 | 规则 |
|---|---|---|
| Login | username | required |
| Login | password | required, min: 6 |
| AssetForm | assetNo | required, pattern: `/^[A-Z]+-\d{4}-\d{4}$/` |
| AssetForm | name | required, max: 50 |
| AssetForm | price | required, min: 0, type: number |
| AssetForm | location | required, max: 100 |
| WorkflowCreate | type | required (1/2/3/4) |
| WorkflowCreate | assetId | required |
| WorkflowCreate | reason | required, max: 200 |
| InventoryTask | taskName | required, max: 50 |
| InventoryTask | startTime/endTime | required, endTime > startTime |
| InventoryTask | assigneeIds | required, min: 1 |
| CreateUser | username | required, pattern: `/^[a-z0-9_]{3,20}$/` |
| CreateUser | password | required, min: 6 |
| CreateUser | realName | required, max: 20 |
| Approve/Reject | comment | 驳回时 required |

所有校验错误信息使用中文。Ant Design `<Form.Item rules={[...]}>` 声明式配置。

---

## 4. Univer 集成说明

### 4.1 安装

```bash
npm install @univerjs/core @univerjs/design @univerjs/ui @univerjs/engine-render @univerjs/sheets
```

### 4.2 版本锁定

Univer 当前处于快速迭代期。**锁定安装时的最新稳定版本**，不要用 `^` 或 `latest`。

### 4.3 关键配置

```typescript
// 初始化 Univer 工作簿
const univer = new Univer({
  theme: defaultTheme,
  locale: LocaleType.ZH_CN,
});

univer.registerPlugin(UniverSheetsPlugin);

const workbook = univer.createWorkbook({
  id: `inventory-${taskId}`,
  name: '盘点表',
  sheetConfig: {
    rowCount: expectedAssets.length + 1,  // +1 for header
    columnCount: 5,                         // 编号/名称/账面位置/实际位置/备注
  },
});
```

### 4.4 空状态与错误

| 状态 | 处理 |
|---|---|
| Univer 加载中 | `<Skeleton active paragraph={{ rows: 10 }} />` |
| Univer 加载失败 | `<Result status="error" title="表格组件加载失败" extra={<Button>重试</Button>}>` |
| expectedAssets 为空 | 不初始化 Univer，直接显示 Empty |

---

*文档版本：v1.0 | 2026-07-07*
