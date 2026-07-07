# FAMS 后端 → 前端交接文档

> 版本：v1.0 | 日期：2026-07-07 | 状态：后端开发完成，全链路验证通过

---

## 1. 快速开始

### 1.1 启动后端

```bash
cd backend/

# 1. 配置环境变量
cp deploy/docker/docker-compose-env.example.yml deploy/docker/docker-compose-env.yml

# 2. 启动基础设施（PostgreSQL / MySQL / MongoDB / Redis / Kafka / etcd）
make infra-up
# 或: docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/docker-compose-env.yml up -d

# 3. 启动后端微服务（每个终端一个）
go run ./service/user/api/user.go          # :8888
go run ./service/asset/api/asset.go        # :8889
go run ./service/workflow/api/workflow.go  # :8890
go run ./service/inventory/api/inventory.go # :8891
go run ./service/report/api/report.go      # :8892

# 4. (可选) 启动 Nginx 统一网关
docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/docker-compose-env.yml up -d nginx
# 之后可通过 http://localhost 访问所有 API
```

### 1.2 测试账号

| 用户名 | 密码 | 角色 | 部门 |
|---|---|---|---|
| `admin_school` | `Test@123456` | 1-校级管理员 | 本校 |
| `admin_info` | `Test@123456` | 2-学院管理员 | 信息工程学院 |
| `student_001` | `Test@123456` | 3-普通师生 | 软件工程实验室 |
| `student_002` | `Test@123456` | 3-普通师生 | 网络工程实验室 |
| `student_me` | `Test@123456` | 3-普通师生 | 机械工程学院 |

---

## 2. 服务架构

```
                     ┌─────────────┐
                     │   Nginx:80  │  (可选，也可直连各服务端口)
                     └──────┬──────┘
          ┌─────────────────┼─────────────────┐
          │                 │                 │
     ┌────▼────┐      ┌────▼────┐       ┌────▼────┐
     │user-api │      │asset-api│       │workflow │  ...
     │  :8888  │      │  :8889  │       │  :8890  │
     └─────────┘      └─────────┘       └─────────┘
```

| 服务 | 端口 | 说明 |
|---|---|---|
| user-api | 8888 | 登录、Token、用户管理、组织树 |
| asset-api | 8889 | 资产台账 CRUD、列表查询 |
| workflow-api | 8890 | 领用/归还/报修/报废审批 |
| inventory-api | 8891 | 盘点任务、草稿提交、归档 |
| report-api | 8892 | 统计报表、数据导出 |

**前端调用方式**：
- 开发环境直连各服务端口（如 `http://localhost:8888/api/v1/user/login`）
- 生产环境统一通过 Nginx 端口 80（如 `http://localhost/api/v1/user/login`）

---

## 3. 鉴权

### 3.1 登录

```
POST /api/v1/user/login
Content-Type: application/json

Request:
{
  "username": "admin_school",
  "password": "Test@123456"
}

Response (200):
{
  "code": 0,
  "message": "ok",
  "data": {
    "accessToken": "eyJ...",
    "refreshToken": "eyJ...",
    "expiresIn": 7200,
    "tokenType": "Bearer"
  }
}
```

### 3.2 在请求中携带 Token

```
GET /api/v1/user/me
Authorization: Bearer <accessToken>
```

**重要规则**：
- Access Token 有效期 **2 小时**，过期后用 Refresh Token 续期
- Refresh Token 有效期 **24 小时**，**一次性使用**（刷新后旧 Refresh Token 作废）
- 登出后 Token 写入黑名单，**立即失效**（不可继续使用）

### 3.3 刷新 Token

```
POST /api/v1/user/refresh
Content-Type: application/json

Request:
{ "refreshToken": "eyJ..." }

Response: 同 login（新的 accessToken + refreshToken）
```

### 3.4 登出

```
POST /api/v1/user/logout
Authorization: Bearer <accessToken>

Response: { "code": 0, "message": "ok", "data": null }
```

---

## 4. 统一响应格式

### 成功

```json
{ "code": 0, "message": "ok", "data": <具体数据> }
```

### 失败

```json
{ "code": <错误码>, "message": "<中文错误描述>", "data": null }
```

### 分页列表

```json
{
  "code": 0, "message": "ok",
  "data": {
    "list": [...],
    "page": 1,
    "pageSize": 20,
    "total": 100
  }
}
```

**分页参数**：`?page=1&pageSize=20`（page 从 1 开始，pageSize 范围 1-100）

---

## 5. 核心 API 速查

详细契约见 `03-api-contract.md`。

### 5.1 User Service (`:8888`)

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/user/login` | 否 | 登录 |
| POST | `/api/v1/user/refresh` | Refresh | 续期 |
| POST | `/api/v1/user/logout` | 是 | 登出 |
| GET | `/api/v1/user/me` | 是 | 当前用户信息 |
| GET | `/api/v1/user/departments/tree` | 是 | 组织树（按角色裁剪） |

### 5.2 Asset Service (`:8889`)

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/asset/assets` | role≤2 | 新增资产 |
| GET | `/api/v1/asset/assets` | 是 | 分页列表（部门隔离） |
| GET | `/api/v1/asset/assets/:id` | 是 | 资产详情 |
| PUT | `/api/v1/asset/assets/:id` | role≤2 | 编辑资产 |
| DELETE | `/api/v1/asset/assets/:id` | role≤2 | 逻辑删除 |
| GET | `/api/v1/asset/assets/shared` | role=3 | 学院共享资产 |

**列表过滤参数**：`?category=设备&status=1&keyword=激光&page=1&pageSize=20`

### 5.3 Workflow Service (`:8890`)

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/workflow/requests` | role=3 | 创建申请 |
| GET | `/api/v1/workflow/requests` | 是 | 列表（scope=my/todo） |
| GET | `/api/v1/workflow/requests/:id` | 是 | 详情（含审批日志） |
| POST | `/api/v1/workflow/requests/:id/approve` | role≤2 | 同意 |
| POST | `/api/v1/workflow/requests/:id/reject` | role≤2 | 驳回 |

**工单 type 枚举**：1-领用, 2-归还, 3-报修, 4-报废

**审批链**：院级初审（stage=1）→ 校级复审（stage=2）→ 归档（stage=3）

### 5.4 Inventory Service (`:8891`)

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| POST | `/api/v1/inventory/tasks` | role≤2 | 创建盘点任务 |
| GET | `/api/v1/inventory/tasks` | role≤2 | 任务列表 |
| GET | `/api/v1/inventory/tasks/:id/expected-assets` | 是 | 应盘资产清单 |
| POST | `/api/v1/inventory/tasks/:id/submit` | 指派员 | 批量提交草稿 |
| POST | `/api/v1/inventory/tasks/:id/archive` | role≤2 | 归档 |
| GET | `/api/v1/inventory/tasks/:id/records` | 是 | 比对结果 |

**批量提交请求格式**：
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

**批量提交响应（部分成功语义）**：
```json
{
  "code": 0,
  "data": {
    "success": ["EQUIP-2026-0001"],
    "conflicts": [{ "assetNo": "EQUIP-2026-0002", "code": 40901, "message": "资产正在被他人盘点" }],
    "failures": []
  }
}
```

### 5.5 Report Service (`:8892`)

| 方法 | 路径 | 鉴权 | 说明 |
|---|---|---|---|
| GET | `/api/v1/report/assets/by-dept` | 是 | 按部门资产统计 |
| GET | `/api/v1/report/inventory/diff/:taskId` | 是 | 盘点差异统计 |
| POST | `/api/v1/report/export` | 是 | 异步导出（返回 jobId） |
| GET | `/api/v1/report/export/:jobId` | 是 | 查询导出进度 |
| GET | `/api/v1/report/export/:jobId/download` | 是 | 下载 CSV |

---

## 6. 常用错误码

前端应针对以下错误码做特殊处理：

| code | HTTP | 含义 | 前端处理建议 |
|---|---|---|---|
| 0 | 200 | 成功 | — |
| 40101 | 401 | Token 无效/过期 | 跳转登录页 |
| 40102 | 401 | Token 已被撤销 | 提示"已被登出"，跳转登录页 |
| 40301 | 403 | 无操作权限 | 提示"权限不足" |
| 40302 | 403 | 无权访问该数据 | 提示"无权访问" |
| 40901 | 409 | 操作冲突（盘点锁） | 提示"正在被他人操作，请重试" |
| 40902 | 409 | 已有进行中的申请 | 提示"该资产已有审批中的工单" |
| 42201 | 422 | 业务状态不允许 | 提示具体 message |
| 50001 | 500 | 服务器错误 | 提示"系统异常，请稍后重试" |

完整 27 个错误码见 `06-error-codes.md`。

---

## 7. 前端关键业务规则

### 7.1 三级角色

| 角色 | roleLevel | 权限范围 |
|---|---|---|
| 校级管理员 | 1 | 全局管理、终审、全校数据可见 |
| 学院管理员 | 2 | 本院管理、初审、本院数据可见 |
| 普通师生 | 3 | 提交申请、查看个人资产、共享资产 |

### 7.2 部门数据隔离

- 角色 2/3 只能看到**本部门及其下属部门**的数据
- 角色 1 全局可见
- 组织树通过 `GET /departments/tree` 获取，按当前用户权限裁剪

### 7.3 资产状态枚举

| status | 含义 |
|---|---|
| 1 | 在库 |
| 2 | 领用中 |
| 3 | 维修中 |
| 4 | 已报废 |

### 7.4 工单状态枚举

| status | 含义 |
|---|---|
| 1 | 审批中 |
| 2 | 审批通过（已归档） |
| 3 | 已被驳回 |

| currentStage | 含义 |
|---|---|
| 1 | 待院级初审 |
| 2 | 待校级复审 |
| 3 | 归档结束 |

### 7.5 审批规则

- **领用**：资产必须在库（status=1）且未被他人领用
- **归还**：必须是当前领用人本人归还
- **报修**：在库或领用中均可
- **报废**：在库或维修中均可（领用中需先归还）
- 同一资产同时只能有 **1 张审批中工单**（驳回后可立即重新申请）

### 7.6 盘点规则

- 盘点采用"批量提交 + 冲突检测"模式（非实时协同）
- 多人在同一盘点任务中提交时，同一资产**加分布式锁**（30s TTL），先到先得
- 提交时可选传 `expectedUpdatedAt` 做**乐观锁版本检查**（首次传 null）
- **部分成功语义**：批量提交返回 `success` + `conflicts` + `failures` 三个列表，前端应逐条展示结果
- 前端表格组件推荐使用 **Univer**（Luckysheet 的后继项目）

---

## 8. 接口额外说明

### 8.1 资产列表过滤

```
GET /api/v1/asset/assets?category=设备&status=1&keyword=激光&page=1&pageSize=20
```

- `category`：精确匹配
- `status`：1/2/3/4
- `keyword`：模糊匹配 `asset_no` 和 `name`
- 默认过滤 `deleted_at IS NULL`

### 8.2 工单列表 scope

```
GET /api/v1/workflow/requests?scope=my     # 我提交的
GET /api/v1/workflow/requests?scope=todo   # 待我审批的
```

### 8.3 组织树结构

```json
{
  "nodes": [
    {
      "id": 1, "parentId": 0, "deptName": "本校", "deptCode": "ROOT",
      "path": "/1/",
      "children": [
        {
          "id": 15, "parentId": 1, "deptName": "信息工程学院",
          "children": [ ... ]
        }
      ]
    }
  ]
}
```

### 8.4 用户信息 (/me)

```json
{
  "id": 10001, "username": "admin_school", "realName": "张校管",
  "roleLevel": 1, "departmentId": 1, "departmentName": "本校", "status": 1
}
```

---

## 9. 前端开发建议

### 9.1 开发流程

1. **先对接 user-api**：实现登录页面 + Token 管理（存储、过期检测、自动刷新）
2. **再对接 asset-api**：实现资产管理页面（列表、筛选、CRUD）
3. **再对接 workflow-api**：实现审批流页面（申请表单、待办列表、审批操作）
4. **最后对接 inventory + report**：盘点表格（Univer）、统计图表

### 9.2 Token 管理

```javascript
// 建议封装
class AuthManager {
  getToken() { return localStorage.getItem('accessToken') }
  setToken(t) { localStorage.setItem('accessToken', t) }
  getRefreshToken() { return localStorage.getItem('refreshToken') }

  async refreshIfNeeded() {
    // 如果 accessToken 即将过期，用 refreshToken 获取新 token
  }

  async fetch(url, options) {
    await this.refreshIfNeeded()
    const res = await fetch(url, {
      ...options,
      headers: { ...options.headers, 'Authorization': `Bearer ${this.getToken()}` }
    })
    if (res.status === 401) { /* 跳转登录 */ }
    return res
  }
}
```

### 9.3 错误处理

```javascript
const res = await api.post('/workflow/requests', body)
if (res.code === 40902) {
  alert('该资产已有审批中的工单')
} else if (res.code === 42201) {
  alert(res.message) // "资产当前不可领用"
}
```

### 9.4 盘点表格（Univer）

- 用 `GET /tasks/:id/expected-assets` 获取应盘资产列表作为表格初始行
- 用户编辑后调用 `POST /tasks/:id/submit` 批量提交
- 提交响应中的 `conflicts` 列表应在表格中**标红高亮**提示重试
- `expectedUpdatedAt` 首次提交传 `null`，重试时传上次返回的 `updatedAt`

---

## 10. 环境信息

| 项目 | 值 |
|---|---|
| 后端仓库 | `github.com/sisyphe550/assets-db` |
| 代码路径 | `backend/` |
| Go 版本 | ≥1.22 |
| 基础设施 | Docker Compose（14 个容器） |
| 数据库 | PostgreSQL 16 + MySQL 8.0 + MongoDB 7.0 + Redis 7.2 |
| 前端端口 | 未定（后端所有端口见 §2） |

---

*文档版本：v1.0 | 2026-07-07*
