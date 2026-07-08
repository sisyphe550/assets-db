# FAMS 前端 API 缺口补全记录

> 版本：v1.0 | 日期：2026-07-08 | 分支：`feat/frontend-api-gaps`

---

## 1. 背景

前端设计文档开发过程中发现以下后端缺口（契约已定义但实现缺失或不完整）。本次一次性补全。

| 缺口 | 严重度 | 状态 |
|---|---|---|
| `GET /user/users` 用户列表 | P0 | ✅ 已实现 |
| `GET /user/users/:id` 用户详情 | P1 | ✅ 已实现 |
| `GET /inventory/tasks` 任务列表 | P0 | ✅ 已实现 |
| `GET /inventory/tasks/:id` 任务详情 | P1 | ✅ 已实现 |
| `GET /asset/assets/shared` 未过滤 `is_shared` | P0 | ✅ 已修复 |
| `GET /asset/assets` 无 `scope=my` | P1 | ✅ 已实现 |
| `GET /workflow/requests?assetId=` | P2 | ✅ 已实现 |
| 创建盘点任务响应缺 `expectedAssetCount` | P2 | ✅ 已补全 |

---

## 2. 新增/变更 API 摘要

### User Service

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/user/users` | 分页用户列表，支持 keyword/departmentId/roleLevel |
| GET | `/api/v1/user/users/:id` | 单用户详情 |

### Asset Service

| 变更 | 说明 |
|---|---|
| `GET /asset/assets` | 新增 Query：`scope=my`、`userId=me\|{id}` |
| `GET /asset/assets/shared` | 修复：`is_shared=1` + 学院子树过滤（经 user-api 组织树计算） |

环境变量：`USER_API_URL`（asset-api 调用组织树，默认 `http://localhost:8888`）

### Inventory Service

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/inventory/tasks` | 任务列表，含 assigneeIds/expectedAssetCount/submittedCount |
| GET | `/api/v1/inventory/tasks/:id` | 任务详情 |

Query `scope=assigned`：role=3 仅返回指派给自己的任务。

### Workflow Service

| 变更 | 说明 |
|---|---|
| `GET /workflow/requests` | 新增 Query：`assetId`；`scope=all` 支持校级查全部 |

---

## 3. 代码变更清单

| 文件 | 变更 |
|---|---|
| `service/user/model/sysusermodel.go` | 新增 `List()` |
| `service/user/api/internal/handler/userhandler.go` | 新增 `ListUsers`、`GetUser` |
| `service/user/api/user.go` | 路由分发 GET /users |
| `service/inventory/model/models.go` | 重写 `ListTasks`，新增 `GetAssigneeIDs` |
| `service/inventory/api/internal/handler/invhandler.go` | 新增 `ListTasks`、`GetTask`；增强 `CreateTask` 响应 |
| `service/inventory/api/inventory.go` | 注册 GET 路由 |
| `service/asset/model/assetledgermodel.go` | `AssetListFilter` 支持 userId/sharedOnly |
| `service/asset/api/internal/handler/assethandler.go` | 修复 `SharedList`；`List` 支持 scope=my |
| `service/workflow/model/models.go` | `List` 支持 assetId、scope=all |
| `pkg/dept/path.go` | 新增 `CollegeSubtreeIDs` |
| `tests/e2e/e2e_test.go` | 新增 `TestE2E06_FrontendAPIGaps` |

---

## 4. 测试

```bash
# 单元测试
cd backend && go test ./pkg/... -short -v

# E2E（需基础设施 + 5 个 API 进程运行）
cd backend && go test ./tests/e2e/... -tags=e2e -v -run TestE2E06
```

---

## 5. 前端文档同步

以下前端文档已移除 workaround 说明（由本次后端补全替代）：

- `frontend/doc/07-implementation.md` §9
- `frontend/doc/03-pages.md` §19 用户管理 v1 降级方案

---

*文档版本：v1.0 | 2026-07-08*
