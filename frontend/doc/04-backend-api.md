# 后端 API 参考（前端开发用）

> 整合自 `backend/doc/11-frontend-handoff.md`，供前端开发时查阅。
> 完整契约见 `backend/doc/03-api-contract.md`。

---

## 1. 启动后端

```bash
cd backend/
cp deploy/docker/docker-compose-env.example.yml deploy/docker/docker-compose-env.yml
make infra-up                                 # Docker 基础设施
go run ./service/user/api/user.go             # :8888
go run ./service/asset/api/asset.go           # :8889
go run ./service/workflow/api/workflow.go     # :8890
go run ./service/inventory/api/inventory.go   # :8891
go run ./service/report/api/report.go         # :8892
```

前端通过 Vite proxy 访问，无需关心端口号：

```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api/v1/user':     'http://localhost:8888',
      '/api/v1/asset':    'http://localhost:8889',
      '/api/v1/workflow': 'http://localhost:8890',
      '/api/v1/inventory':'http://localhost:8891',
      '/api/v1/report':   'http://localhost:8892',
    },
  },
});
```

---

## 2. 测试账号

| 用户名 | 密码 | roleLevel | 部门 | 用途 |
|---|---|---|---|---|
| `admin_school` | `Test@123456` | 1 | 本校 | 校级终审、全局管理 |
| `admin_info` | `Test@123456` | 2 | 信息工程学院 | 院级初审、本院管理 |
| `student_001` | `Test@123456` | 3 | 软件工程实验室 | 领用申请、盘点员 |
| `student_002` | `Test@123456` | 3 | 网络工程实验室 | 盘点冲突测试 |
| `student_me` | `Test@123456` | 3 | 机械工程学院 | 跨学院越权测试 |

---

## 3. 鉴权

### 3.1 登录

```
POST /api/v1/user/login
Body: { "username": "admin_school", "password": "Test@123456" }

Response:
{
  "code": 0,
  "data": {
    "accessToken": "eyJ...",    // 有效期 2h
    "refreshToken": "eyJ...",   // 有效期 24h，一次性使用
    "expiresIn": 7200,
    "tokenType": "Bearer"
  }
}
```

### 3.2 请求携带 Token

```
Authorization: Bearer <accessToken>
```

### 3.3 刷新 Token

```
POST /api/v1/user/refresh
Body: { "refreshToken": "eyJ..." }
Response: 同 login（新的 accessToken + refreshToken）
```

刷新后**旧 Refresh Token 立即作废**。如果前端使用旧 Refresh Token 再次刷新，返回 40102。

### 3.4 登出

```
POST /api/v1/user/logout
Authorization: Bearer <accessToken>
Response: { "code": 0, "data": null }
```

登出后该 accessToken **立即写入黑名单**，后续请求返回 40102。

### 3.5 获取当前用户

```
GET /api/v1/user/me
Response:
{
  "code": 0,
  "data": {
    "id": 10001, "username": "admin_school", "realName": "张校管",
    "roleLevel": 1, "departmentId": 1, "departmentName": "本校", "status": 1
  }
}
```

---

## 4. 统一响应格式

### 成功

```json
{ "code": 0, "message": "ok", "data": <具体数据> }
```

### 失败

```json
{ "code": <错误码>, "message": "<中文描述>", "data": null }
```

### 分页列表

```json
{
  "code": 0,
  "data": { "list": [...], "page": 1, "pageSize": 20, "total": 100 }
}
```

| 参数 | 默认 | 范围 |
|---|---|---|
| page | 1 | ≥1 |
| pageSize | 20 | 1–100 |

---

## 5. 全部 API 端点

### 5.1 User Service

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/user/login` | 否 | 登录 |
| POST | `/api/v1/user/refresh` | Refresh | 续期 |
| POST | `/api/v1/user/logout` | 是 | 登出 |
| GET | `/api/v1/user/me` | 是 | 当前用户 |
| GET | `/api/v1/user/departments/tree` | 是 | 组织树（按角色裁剪） |
| POST | `/api/v1/user/departments` | role=1 | 新增部门 |
| POST | `/api/v1/user/users` | role≤2 | 创建用户 |
| PUT | `/api/v1/user/users/:id/status` | role≤2 | 禁用/启用 |
| POST | `/api/v1/user/users/:id/force-logout` | role≤2 | 强制下线 |

### 5.2 Asset Service

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/asset/assets` | role≤2 | 新增资产 |
| GET | `/api/v1/asset/assets` | 是 | 分页列表（部门隔离） |
| GET | `/api/v1/asset/assets/:id` | 是 | 详情 |
| PUT | `/api/v1/asset/assets/:id` | role≤2 | 编辑 |
| DELETE | `/api/v1/asset/assets/:id` | role≤2 | 逻辑删除 |
| GET | `/api/v1/asset/assets/shared` | role=3 | 学院内共享资产 |

**列表参数**：`?category=设备&status=1&keyword=激光&page=1&pageSize=20`

### 5.3 Workflow Service

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/workflow/requests` | role=3 | 创建申请 |
| GET | `/api/v1/workflow/requests` | 是 | 列表 |
| GET | `/api/v1/workflow/requests/:id` | 是 | 详情 + 审批日志 |
| POST | `/api/v1/workflow/requests/:id/approve` | role≤2 | 同意 |
| POST | `/api/v1/workflow/requests/:id/reject` | role≤2 | 驳回 |

**创建申请 Body**：
```json
{ "assetId": 501, "type": 1, "reason": "课程设计需要使用" }
```
`type`：1-领用, 2-归还, 3-报修, 4-报废

**列表参数**：`?scope=my|todo&page=1&pageSize=20`

**工单详情 Response 结构**：
```json
{
  "code": 0,
  "data": {
    "request": {
      "id": 88001, "assetId": 501, "requesterId": 10003,
      "departmentId": 15, "type": 1, "currentStage": 1, "status": 1,
      "reason": "...", "createdAt": "...", "updatedAt": "..."
    },
    "logs": [
      { "id": 1, "requestId": 88001, "operatorId": 10003,
        "action": "提交申请", "comment": "", "operateTime": "..." }
    ]
  }
}
```

### 5.4 Inventory Service

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/inventory/tasks` | role≤2 | 创建任务 |
| GET | `/api/v1/inventory/tasks` | role≤2 | 任务列表 |
| GET | `/api/v1/inventory/tasks/:id/expected-assets` | 指派员 | 应盘资产清单 |
| POST | `/api/v1/inventory/tasks/:id/submit` | 指派员 | 批量提交草稿 |
| POST | `/api/v1/inventory/tasks/:id/archive` | role≤2 | 归档 |
| GET | `/api/v1/inventory/tasks/:id/records` | 是 | 比对结果（含 diffStatus） |

**提交 Body**：
```json
{
  "items": [
    {
      "assetNo": "EQUIP-2026-0001",
      "modifiedCells": { "actual_location": "一号实验楼101", "temp_notes": "正常" },
      "expectedUpdatedAt": null
    }
  ]
}
```

**提交 Response（部分成功）**：
```json
{
  "code": 0,
  "data": {
    "success": ["EQUIP-2026-0001"],
    "conflicts": [{ "assetNo": "EQUIP-2026-0002", "code": 40901, "message": "资产正在被他人盘点" }],
    "failures": [{ "assetNo": "EQUIP-2026-0003", "code": 50301, "message": "草稿写入失败" }]
  }
}
```

### 5.5 Report Service

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/api/v1/report/assets/by-dept` | 是 | 按部门统计 |
| GET | `/api/v1/report/inventory/diff/:taskId` | 是 | 盘点差异统计 |
| POST | `/api/v1/report/export` | 是 | 异步导出 |
| GET | `/api/v1/report/export/:jobId` | 是 | 查询进度 |
| GET | `/api/v1/report/export/:jobId/download` | 是 | 下载 CSV |

**导出 Body**：`{ "exportType": "asset_list" }` — 返回 `{ "jobId": 90001 }`

**导出状态**：0-排队, 1-处理中, 2-已完成, 3-失败

---

## 6. 常用错误码

| code | HTTP | 含义 | 前端处理 |
|---|---|---|---|
| 0 | 200 | 成功 | — |
| 40001 | 400 | 参数无效 | 表单校验提示 |
| 40101 | 401 | Token 无效/过期 | 自动刷新 token，失败则跳转登录页 |
| 40102 | 401 | Token 被撤销 | 提示"已被登出"，跳转登录页 |
| 40301 | 403 | 无操作权限 | 提示"权限不足" |
| 40302 | 403 | 数据越权 | 提示"无权访问该数据" |
| 40401 | 404 | 资源不存在 | 显示 404 页面 |
| 40901 | 409 | 操作冲突 | 盘点冲突 → 对应行标红 |
| 40902 | 409 | 已有进行中工单 | 弹窗"该资产已有审批中工单" |
| 40903 | 409 | 唯一键冲突 | 提示具体的重复字段 |
| 42201 | 422 | 业务状态不允许 | 显示 `message` 内容（后端返回具体原因） |
| 42202 | 422 | 工单已归档 | 提示"该工单已归档，无法操作" |
| 50001 | 500 | 服务器错误 | 全局 notification |
| 50301 | 503 | 服务不可用 | 全局 notification，自动重试 |

**前端错误处理策略**：
- `baseQuery` 统一拦截 40101/40102 → 尝试刷新 → 失败跳登录
- 4xx 错误在触发操作的组件内展示（40902 在工单创建页弹窗，40901 在盘点表格行标红）
- 5xx 错误全局 notification，不阻塞用户操作

---

## 7. 业务枚举

```typescript
// 资产状态
const ASSET_STATUS = { 1: '在库', 2: '领用中', 3: '维修中', 4: '已报废' };

// 工单类型
const WORKFLOW_TYPE = { 1: '领用', 2: '归还', 3: '报修', 4: '报废' };

// 工单状态
const WORKFLOW_STATUS = { 1: '审批中', 2: '已通过', 3: '已驳回' };

// 审批阶段
const WORKFLOW_STAGE = { 1: '待院级初审', 2: '待校级复审', 3: '已归档' };

// 角色
const ROLE = { 1: '校级管理员', 2: '学院管理员', 3: '普通师生' };

// 盘点差异
const DIFF_STATUS = { 0: '未比对', 1: '相符', 2: '盘盈', 3: '盘亏' };
```

---

## 8. 关键业务规则

### 三级角色

| 角色 | roleLevel | 权限范围 |
|---|---|---|
| 校级管理员 | 1 | 全局管理、终审、全校数据可见 |
| 学院管理员 | 2 | 本院管理、初审、本院数据可见 |
| 普通师生 | 3 | 提交申请、查看个人/共享资产 |

### 部门数据隔离

- role=2/3 只能看到本部门及下属部门的数据
- `GET /departments/tree` 返回的组织树已按当前用户裁剪

### 资产状态机

```
在库(1) ──→ 领用中(2) ──→ 归还 ──→ 在库(1)
  │            │
  ├──→ 维修中(3)
  ├──→ 已报废(4)  ←── 维修中(3) 也可报废
```

状态变更**不能通过 PUT /assets/:id 直接修改**，必须走工作流。

### 审批规则

| 工单类型 | 前置条件 |
|---|---|
| 领用 | asset.status=1（在库）且未被他人领用 |
| 归还 | asset.status=2（领用中）且当前用户是领用人 |
| 报修 | asset.status IN (1,2) |
| 报废 | asset.status IN (1,3)（领用中需先归还） |

- 同一资产同时只能有 **1 张审批中工单**（40902）
- 驳回后立即允许重新申请
- 审批链：院级初审 → 校级复审 → 归档

### 盘点冲突处理

- 多人提交同一 asset_no 时，Redis 分布式锁（30s TTL），先到先得
- 后到者返回 40901 + "资产正在被他人盘点"
- `expectedUpdatedAt` 做乐观锁 CAS：传 null 首次写入，传上次更新时间的重试时做版本比对
- 批量提交采用**部分成功语义**：不要因为一条冲突就整批回滚

---

## 9. 组织树数据结构

```json
{
  "nodes": [{
    "id": 1, "parentId": 0, "deptName": "本校", "deptCode": "ROOT",
    "path": "/1/", "children": [
      {
        "id": 15, "parentId": 1, "deptName": "信息工程学院",
        "deptCode": "INFO", "path": "/1/15/", "children": [
          { "id": 103, "parentId": 15, "deptName": "软件工程实验室", "children": null }
        ]
      }
    ]
  }]
}
```

前端可用 Ant Design `TreeSelect` 组件渲染，`fieldNames` 映射即可。

---

## 10. 后端进程完整列表

| 进程 | 端口 | 类型 | 说明 |
|---|---|---|---|
| user-api | 8888 | HTTP API | 用户认证与权限 |
| user-rpc | 8081 | gRPC | 用户查询（前端不直连） |
| asset-api | 8889 | HTTP API | 资产 CRUD |
| asset-rpc | 8082 | HTTP | 资产校验（前端不直连） |
| asset-consumer | — | Kafka | 台账异步同步（后台） |
| workflow-api | 8890 | HTTP API | 工单审批 |
| outbox-dispatcher | — | 定时轮询 | Outbox→Kafka（后台） |
| inventory-api | 8891 | HTTP API | 盘点任务 |
| comparison-worker | — | Kafka | 盘点比对（后台） |
| report-api | 8892 | HTTP API | 报表导出 |
| export-worker | — | Redis | CSV 生成（后台） |

**前端只需要关心 5 个 API 端口（8888–8892）。**

---

*文档版本：v1.0 | 2026-07-07*
