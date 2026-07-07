# FAMS 后端开发完成状态报告

> 更新日期：2026-07-07

---

## 1. 总体状态：90% 完成

后端核心功能已开发完成并通过测试。剩余 10% 主要是跨服务调用链路完善和边缘场景覆盖。

---

## 2. 已完成并测试通过

### 2.1 微服务（5 个 API + 2 个 RPC）

| 服务 | 端口 | 接口数 | 测试方式 | 状态 |
|---|---|---|---|---|
| user-api | 8888 | 9 | curl + E2E | PASS |
| user-rpc | 8081 | 3 (gRPC) | grpcurl | PASS |
| asset-api | 8889 | 6 | curl + E2E | PASS |
| asset-rpc | 8082 | 4 (HTTP) | 编译 | `*` |
| workflow-api | 8890 | 5 | curl + E2E | PASS |
| inventory-api | 8891 | 4 | curl + E2E | PASS |
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

## 3. 已知限制（10% 未完成）

### 3.1 跨服务调用未打通

| 问题 | 影响 |
|---|---|
| workflow-api 创建工单时不调用 asset-rpc 校验资产状态 | 可能对已领用资产创建重复工单 |
| workflow-api 终审通过后不调用 asset-rpc 变更资产状态 | 审批通过但台账 status 不变 |
| 这些调用在代码中被注释为 `// 生产环境应通过 gRPC 调用` | |

**修复方案**：在 workflow handler 中集成 asset-rpc gRPC client。

### 3.2 Kafka Consumer 未启动

| 问题 | 影响 |
|---|---|
| asset-rpc 侧 Kafka consumer 未独立启动 | Outbox 投递到 Kafka 后无人消费 |
| 台账异步同步未生效 | 审批通过后资产 status 不自动更新 |
| 报表快照表无数据 | /assets/by-dept 返回空 |

**修复方案**：创建 `service/asset/consumer/main.go` 订阅 `fams-asset-lifecycle-events`。

### 3.3 盘点草稿批量提交未实现

| 问题 | 影响 |
|---|---|
| POST /tasks/:id/submit 接口未实现 handler | 盘点员无法提交草稿 |
| Redis 分布式锁在批量提交中未应用 | |
| MongoDB CAS 乐观锁未应用 | |

**修复方案**：在 inventory handler 中实现 Submit handler，逐条加锁+CAS 写入 MongoDB。

### 3.4 MongoDB 连接未集成到 inventory-api

| 问题 | 影响 |
|---|---|
| inventory-api main.go 不连接 MongoDB | 草稿提交无法落库 |

**修复方案**：inventory-api main.go 添加 MongoDB 连接并传给 handler。

### 3.5 报表异步导出 Worker 未实现

| 问题 | 影响 |
|---|---|
| Export 只写 job 不生成文件 | download 返回空 CSV |
| Redis 导出队列未消费 | |

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

---

*文档版本：v1.0 | 2026-07-07*
