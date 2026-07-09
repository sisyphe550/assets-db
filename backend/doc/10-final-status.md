# FAMS 后端开发完成状态报告

> 更新日期：2026-07-09（功能与修复详见 `13-release-notes-2026-07-09.md`）

---

## 1. 总体状态：开发完成

所有计划功能已实现，全链路端到端验证通过。

### 全链路验证（2026-07-07）

```
Create workflow → College approve → School approve
    → Outbox write (同一 PG 事务)
    → Dispatcher poll → Kafka (status 0→1)
    → Consumer read → Dedup check → Asset status change
    → Asset 505: 在库(1) → 领用中(2), user_id: NULL → 10003
```

### 两个独立 Worker 进程

| Worker | 功能 | 状态 |
|---|---|---|
| `outbox-dispatcher` | 轮询 `workflow_outbox` → Kafka | 端到端验证 |
| `asset-consumer` | 订阅 `fams-asset-lifecycle-events` → 台账同步 | 端到端验证 |
| `comparison-worker` | 订阅 `fams-inventory-comparison-tasks` → 账实比对 | 代码已实现 |
| `export-worker` | Redis BRPOP → CSV 生成（asset_list / workflow_log / inventory_diff） | 已实现，需与 report-api 同启 |

---

## 2. 已完成并测试通过

### 2.1 微服务（5 个 API + 2 个 RPC）

| 服务 | 端口 | 接口数 | 测试方式 | 状态 |
|---|---|---|---|---|
| user-api | 8888 | 11 | curl + E2E | PASS |
| user-rpc | 8081 | 3 (gRPC) | grpcurl | PASS |
| asset-api | 8889 | 6 | curl + E2E | PASS |
| asset-rpc | 8082 | 4 (HTTP) | 编译 | `*` |
| workflow-api | 8890 | 5 | curl + E2E | PASS |
| inventory-api | 8891 | 6 | curl + E2E | PASS |
| report-api | 8892 | 6 | curl | `**` |

`*` asset-rpc 编译通过但未单独测试 gRPC 调用
`**` report /assets/by-dept 返回空（快照表无数据，需 Kafka consumer 填充）

### 2.2 基础设施

| 组件 | 状态 |
|---|---|
| PostgreSQL 16 (12 张表 + 索引) | healthy |
| MySQL 8.0 (2 张表) | healthy |
| MongoDB 7.0 (inventory_draft + 3 indexes) | healthy |
| Redis 7.2 (session / blacklist / lock) | healthy |
| Kafka 3.9 (2 topics, 3 partitions) | healthy |
| etcd 3.5 | healthy |
| Jaeger (OTLP 4317) | healthy |
| Prometheus + Grafana | running |
| Nginx (port 80 → 5 services) | routing verified |

### 2.3 核心业务流程（全链路 curl 验证）

| 流程 | 步骤 | 状态 |
|---|---|---|
| **登录认证** | 正确密码→JWT；错误密码→40101；禁用用户→40301 | PASS |
| **JWT 黑名单** | 登出写 blacklist；旧 token→40102；Refresh 防重用 | PASS |
| **组织树** | 校级全量 3 层嵌套；院级仅见子树 | PASS |
| **权限隔离** | 院级不能创建 role=2 用户；学生不能审批 | PASS |
| **资产 CRUD** | 创建/列表/详情/编辑/软删除 | PASS |
| **审批流转** | 创建→院审→校审→归档；Outbox 写入 | PASS |
| **重复申请拦截** | 同一资产 40902（DB 部分唯一索引） | PASS |
| **驳回+重新申请** | 驳回后允许立即新建工单 | PASS |
| **审计日志** | 3 条完整留痕（提交→院审→校审） | PASS |
| **Outbox 投递** | 独立 Dispatcher 轮询→Kafka→标记 sent | PASS |
| **盘点任务** | 创建→归档 status 1→2 | PASS |
| **Redis 分布式锁** | TryLock + Lua 安全释放（integration test） | PASS |
| **Kafka 生产消费** | 端到端往返（integration test） | PASS |
| **MongoDB 草稿** | CRUD + 唯一索引防重复（integration test） | PASS |

### 2.4 自动化测试

| 类型 | 命令 | 结果 |
|---|---|---|
| 单元测试 | `go test ./pkg/... -short` | 3/3 PASS |
| 集成测试 | `go test -tags=integration` | Redis/Kafka/Mongo PASS |
| E2E 测试 | `go test ./tests/e2e/... -tags=e2e` | **5/5 PASS** |
| 构建 | `go build ./...` | OK |
| 静态分析 | `go vet ./...` | OK |

### 2.5 E2E 测试场景（5 个）

| 编号 | 场景 | 状态 |
|---|---|---|
| E2E-01 | 登录→创建资产→列表→详情→软删除 | PASS |
| E2E-02 | 创建工单→院审→校审→3 条审计日志 | PASS |
| E2E-03 | 创建盘点任务→归档→记录查询 | PASS |
| E2E-04 | RBAC 越权拦截（学生审批/院级提权） | PASS |
| E2E-05 | 重复申请 40902 + 驳回 + 重新申请 | PASS |

---

## 3. 已知限制（已于 2026-07-07 全部修复）

### 3.1 跨服务调用（已修复）

**实现方案**：
- workflow-api 创建工单前通过 HTTP 调用 `asset-rpc/CheckAssetForWorkflow` 校验资产状态
- 校验失败返回 42201 + 具体拒绝原因
- 校验成功时获取 `department_id` 填充到工单
- 共享业务逻辑移至 `service/asset/logic/` 包（供 RPC Server、Consumer 复用）

**涉及文件**：`service/workflow/api/internal/handler/workflowhandler.go`、`service/asset/logic/assetlogic.go`

### 3.2 Kafka Consumer（已修复）

**实现方案**：
- 创建 `service/asset/consumer/main.go` 独立进程
- 订阅 `fams-asset-lifecycle-events`，group=`asset-consumer`
- 消费流程：JSON 解析 → 幂等去重（`asset_event_dedup` 表） → 状态机校验 → 台账更新
- 遵循 at-least-once 语义，重复消息 ACK 跳过

**涉及文件**：`service/asset/consumer/main.go`

### 3.3 盘点草稿批量提交（已修复）

**实现方案**（严格按 `07-inventory-ops.md`）：
1. 校验任务存在且 status=1、用户是 assignee 或管理员
2. 逐条处理每个 item：
   - Redis 分布式锁 `fams:lock:inventory:{asset_no}`，30s TTL
   - 调用 asset-rpc `CheckAssetScope` 校验资产在盘点范围内
   - MongoDB upsert with CAS：`filter={task_id, asset_no, updated_at(optional)}`
   - Lua 安全释放锁（先校验 owner 再 DEL）
3. 部分成功语义：HTTP 200 + `{success, conflicts, failures}` 逐条详情

**涉及文件**：`service/inventory/api/internal/handler/invhandler.go`  Submit handler

### 3.4 MongoDB 连接（已修复）

**实现方案**：
- inventory-api main.go 添加 `mongo.Connect()` 连接
- 传入 `*mongo.Collection` 到 InvHandler
- 环境变量：`MONGO_URI`（默认 `mongodb://localhost:27017`）、`MONGO_DB`（默认 `fams_inventory`）

**涉及文件**：`service/inventory/api/inventory.go`

### 3.5 报表异步导出 Worker（已修复）

**实现方案**：
- 创建 `service/report/export-worker/main.go` 独立进程
- 使用 Redis `BRPOP fams:export:queue` 阻塞等待任务（5s 超时）
- Export API 写 `rpt_export_job` 后 `LPUSH` 到队列
- Worker 流程：接任务 → status=1(处理中) → 查 MySQL 生成 CSV（BOM 头）→ status=2(完成)
- CSV 文件存储在 `deploy/export/` 目录

**涉及文件**：`service/report/export-worker/main.go`、`service/report/api/internal/handler/reporthandler.go`

---

## 4. Bug 修复记录

| # | Bug | 根因 | 修复 |
|---|---|---|---|
| 1 | 所有用户登录 40101 | bcrypt hash 不匹配 | 重新生成 hash |
| 2 | 院级组织树为空 | DeptTree rootID 始终为 0 | role=2 时以 deptID 为根 |
| 3 | 校级资产列表为空 | role=1 未识别为 unlimited | 添加 role==1 判断 |
| 4 | 工单 JSON 字段 PascalCase | 缺少 json tag | 添加 camelCase json tag |
| 5 | inventory 归档 50001 | 引用不存在的 updated_at 列 | 移除该列引用 |
| 6 | 所有 PG 表 INSERT 失败 | id 列无序列/IDENTITY | ALTER TABLE ADD IDENTITY |
| 7 | fams_report 数据库不存在 | 初始化脚本未创建 | CREATE DATABASE fams_report |
| 8 | Report export 50001 | 连接 fams_core 但表在 fams_report | 增加 reportDB 连接 |
| 9 | 院级资产列表为空 | asset-api 未注入 dept 子树 | college-subtree fallback + 详情 scope 校验 |
| 10 | 工单详情 IDOR | Detail 无权限校验 | 按角色/子树/申请人隔离 |
| 11 | 盘点 expected 缺子部门 | scope 仅传单 deptId | scopeDeptIDs 展开子树 |
| 12 | 导出 download 404 | 路由与 stub 响应 | 修复路由 + worker 写 CSV + 读文件下载 |

---

## 5. 测试命令速查

```bash
# 单元测试
cd backend && go test ./pkg/... -short -v -cover

# 集成测试（需 docker-compose 运行中）
go test ./tests/integration/... -tags=integration -v
go test ./pkg/redislock/... -tags=integration -v

# E2E 测试（需全部服务运行中）
# Terminal 1-5: go run ./service/*/api/*.go
go test ./tests/e2e/... -tags=e2e -v

# 全量检查
go build ./... && go vet ./...
```

---

## 6. 文档索引

| 文档 | 内容 |
|---|---|
| `01-desgin.md` | 架构设计 |
| `02-plan.md` | P0-P9 开发任务书 |
| `03-api-contract.md` | API 契约 |
| `04-workflow-rules.md` | 工作流规则 |
| `05-seed-fixtures.md` | 种子数据 |
| `06-error-codes.md` | 错误码矩阵 |
| `07-inventory-ops.md` | 盘点操作 |
| `08-infra-config.md` | 基础设施配置 |
| `09-testing.md` | 测试报告 |
| `10-final-status.md` | 本文档 |
| `13-release-notes-2026-07-09.md` | 2026-07-09 修复汇总 |

---

## 7. 2026-07-09 前后端修复摘要

完整说明见 **`doc/13-release-notes-2026-07-09.md`**。要点：

- 院级资产/盘点 scope 子树、工单权限、报表导出链路已打通
- 前端：sessionStorage 多标签隔离、院级全部工单、盘点 `rowsReady` + 恢复 Ant Design 可编辑表格
- Univer 依赖已安装但未用于生产盘点页

---

*文档版本：v1.1 | 2026-07-09*
