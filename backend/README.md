# FAMS Backend

高校固定资产管理系统（Fixed Assets Management System）微服务后端。

## 技术栈

- **语言**: Go 1.22+
- **框架**: go-zero 1.6+
- **数据库**: PostgreSQL 16, MySQL 8.0, MongoDB 7.0, Redis 7.2
- **消息队列**: Kafka 3.x (KRaft)
- **服务注册**: etcd 3.5
- **可观测性**: Jaeger + Prometheus + Grafana

## 微服务

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| user-api | 8888 | HTTP | 用户认证、组织架构 |
| user-rpc | 8081 | HTTP | 用户/部门查询 RPC |
| asset-api | 8889 | HTTP | 资产台账 CRUD |
| asset-rpc | 8082 | HTTP | 资产状态变更 RPC |
| workflow-api | 8890 | HTTP | 审批工单流转 |
| inventory-api | 8891 | HTTP | 盘点任务管理 |
| report-api | 8892 | HTTP | 数据统计与导出 |
| export-worker | — | 进程 | Redis BRPOP 异步 CSV 导出（需与 report-api 同启） |

## 本地开发

```bash
# 1. 启动基础设施
make infra-up

# 2. 启动各微服务（每个终端一个）
go run service/user/api/user.go
go run service/user/rpc/user.go
go run service/asset/api/asset.go
go run service/asset/rpc/asset.go
go run service/workflow/api/workflow.go
go run service/inventory/api/inventory.go
go run service/report/api/report.go
go run service/report/export-worker/main.go   # 报表 CSV 异步导出

# 3. 运行测试
make test-unit        # 单元测试
make test-integration # 集成测试（需基础设施）
make test-e2e         # E2E 测试（需全部服务）
```

## 文档

- `doc/01-desgin.md` — 架构设计
- `doc/02-plan.md` — 分阶段开发任务书
- `doc/03-api-contract.md` — API 契约
- `doc/04-workflow-rules.md` — 工作流规则
- `doc/05-seed-fixtures.md` — 固定种子数据
- `doc/06-error-codes.md` — 错误码矩阵
- `doc/07-inventory-ops.md` — 盘点操作流程
- `doc/10-final-status.md` — 项目完成状态
- `doc/13-release-notes-2026-07-09.md` — 2026-07-09 修复汇总
