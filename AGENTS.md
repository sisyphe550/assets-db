# AGENTS.md

## 项目概况

FAMS（高校固定资产管理系统）为 **前后端均已实现** 的全栈项目，仓库根目录为 `assets-db/`。

| 目录 | 说明 |
|---|---|
| `backend/` | Go 微服务（user / asset / workflow / inventory / report + RPC + Worker） |
| `frontend/` | React 18 + TypeScript + Vite + Ant Design + RTK Query |
| `backend/doc/` | 后端架构、API 契约、测试报告 |
| `frontend/doc/` | 前端设计、实现蓝图、开发日志（`dev-log/`） |

## 本地开发

### 基础设施

```bash
cd backend && make infra-up    # Docker：PG / MySQL / Mongo / Redis / Kafka 等
```

### 后端服务（各开一终端）

```bash
go run service/user/api/user.go
go run service/asset/api/asset.go
go run service/asset/rpc/asset.go
go run service/workflow/api/workflow.go
go run service/inventory/api/inventory.go
go run service/report/api/report.go
go run service/report/export-worker/main.go   # 报表 CSV 导出（与 report-api 配套）
```

其他可选 Worker：`service/asset/consumer/main.go` 等。

### 前端

```bash
cd frontend && npm install && npm run dev   # http://localhost:5173，API 走 Vite proxy
npm test && npm run build
```

### 测试账号

密码均为 `Test@123456`：`admin_school`（校级）、`admin_info`（院级）、`student_001`（师生）。详见 `backend/doc/05-seed-fixtures.md`。

## 架构要点（易踩坑）

1. **院级数据隔离**：资产列表通过 `user-api /departments/college-subtree` 展开院系子树；盘点 `expected-assets` 同样展开 scope 子部门。
2. **盘点表格**：生产使用 `InventorySpreadsheet`（Ant Design 可编辑 Table）；`UniverSpreadsheet.tsx` 为实验代码，**未接入主流程**。
3. **盘点草稿**：MongoDB + CAS；多操作员按 `operator_id` 隔离；前端 `rowsReady` 门控避免重进页面空表。
4. **报表导出**：`POST /report/export` → Redis 队列 → `export-worker` 生成 CSV → `GET .../download`。
5. **JWT**：前端 Token 存 `sessionStorage`（多标签隔离）。

## 文档索引

| 文档 | 用途 |
|---|---|
| `backend/doc/10-final-status.md` | 后端完成度与 Bug 记录 |
| `backend/doc/13-release-notes-2026-07-09.md` | 2026-07-09 前后端修复汇总 |
| `backend/doc/03-api-contract.md` | API 契约（权威） |
| `frontend/doc/07-implementation.md` | 前端实现蓝图 |
| `frontend/doc/dev-log/` | 分阶段开发日志 |

## Cursor Cloud 注意事项

- Docker 需手动启动：`sudo nohup dockerd > /tmp/dockerd.log 2>&1 &`
- `goctl` 在 `$(go env GOPATH)/bin`，非交互 shell 需 `export PATH="$PATH:$(go env GOPATH)/bin"`
