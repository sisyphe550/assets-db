# FAMS — 高校固定资产管理系统

**Fixed Assets Management System（FAMS）** 是一套面向高校的**前后端分离、微服务架构**固定资产全生命周期管理平台。覆盖资产台账、多级审批工单、多人协同盘点、统计报表与异步导出，支持校级 / 院级 / 师生三级角色与院系子树数据隔离。

> 仓库路径：`assets-db/`  
> 后端：Go + go-zero | 前端：React 18 + TypeScript + Ant Design

---

## 目录

- [核心能力](#核心能力)
- [架构总览](#架构总览)
- [设计优势](#设计优势)
- [性能与 QPS 容量](#性能与-qps-容量)
- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [环境要求](#环境要求)
- [快速启动](#快速启动)
- [测试账号](#测试账号)
- [服务端口一览](#服务端口一览)
- [测试与构建](#测试与构建)
- [可观测性](#可观测性)
- [文档索引](#文档索引)

---

## 核心能力

| 模块 | 功能 |
|---|---|
| **用户与组织** | JWT 登录 / 刷新 / 登出；树状组织架构；三级 RBAC；院级子树数据隔离 |
| **资产台账** | CRUD、软删除、分类筛选；领用 / 归还 / 报修 / 报废状态机 |
| **工单审批** | 师生申请 → 院级初审 → 校级终审；Outbox + Kafka 异步同步台账 |
| **协同盘点** | 任务发布、应盘清单、MongoDB 草稿、Redis 分布式锁 + CAS 冲突检测、账实比对 |
| **统计报表** | 按部门 / 类别 / 盘点差异；G2 图表；Redis 队列异步 CSV 导出 |
| **可观测性** | Jaeger 链路追踪、Prometheus 指标、Grafana 大盘（开发环境已编排） |

---

## 架构总览

```
┌─────────────────────────────────────────────────────────────────────────┐
│  Browser  →  Vite Dev (5173) / Nginx (80)  →  5 × HTTP API (8888–8892) │
└─────────────────────────────────────────────────────────────────────────┘
         │                              │
         │  React + RTK Query           │  go-zero JWT / 中间件
         ▼                              ▼
┌──────────────────┐          ┌──────────────────────────────────────────┐
│  frontend/       │          │  backend/service/                         │
│  3 Layout 角色   │          │  user / asset / workflow / inventory /   │
│  pages+components│          │  report  (+ user-rpc, asset-rpc)         │
└──────────────────┘          └──────────────┬───────────────────────────┘
                                               │
                    ┌──────────────────────────┼──────────────────────────┐
                    ▼                          ▼                          ▼
            PostgreSQL 16              MySQL 8.0                   MongoDB 7
         (组织/工单/盘点)            (资产台账)                  (盘点草稿)
                    │                          │                          │
                    └──────── Redis 7.2 ──────┴──── Kafka 3.x ──────────┘
                              (Session/锁/队列)      (领域事件 / 比对任务)
                                               │
                    ┌──────────────────────────┴──────────────────────────┐
                    ▼                          ▼                          ▼
           outbox-dispatcher          asset-consumer              export-worker
           comparison-worker          (异步 Worker 进程)
```

**数据流要点**：

- 审批终审通过 → `workflow_outbox`（同事务）→ Dispatcher → Kafka → Asset Consumer → MySQL 台账状态更新
- 盘点提交 → Redis 锁 + MongoDB CAS → 归档后 Kafka 触发账实比对
- 报表导出 → `report-api` 写任务 → Redis 队列 → `export-worker` 生成 CSV

---

## 设计优势

### 后端架构优势

| 维度 | 说明 |
|---|---|
| **微服务边界清晰** | 按业务域拆分为 5 个 API + 2 个 RPC，读写与报表查询隔离，避免复杂统计 SQL 拖垮在线业务 |
| **Polyglot Persistence** | PG（组织树 + 工单审计）、MySQL（资产台账强一致）、MongoDB（盘点草稿 Schema-less 高写入）、Redis（锁 / 会话 / 队列）各司其职 |
| **组织子树隔离** | `sys_department.path` 物化路径 + `dept.SubtreeIDs`，院级管理员自动展开本学院及下属实验室，SQL 层 `department_id IN (...)` 过滤 |
| **可靠异步** | Transactional Outbox 保证工单事件与 DB 写入原子性；Kafka 消费幂等表 `asset_event_dedup` 防重复处理 |
| **盘点并发安全** | Redis 分布式锁（30s TTL + Lua 安全释放）+ MongoDB `updated_at` CAS，支持多人分区盘点、提交时逐行冲突反馈 |
| **统一工程规范** | `pkg/errx` 27 个业务错误码、`pkg/middleware` JWT/黑名单、`pkg/redislock` 可复用锁封装 |
| **可弹性部署** | 代码保持微服务边界，小规模可单机 docker-compose 合并部署；规模增长后可按服务横向扩容 |
| **全链路可观测** | go-zero 内置 `/metrics`、Jaeger OTLP、Grafana 预置 FAMS 大盘（QPS / P99 / Kafka Lag / 锁冲突率） |

### 前端架构优势

| 维度 | 说明 |
|---|---|
| **三角色独立路由** | `/admin`、`/college`、`/user` 三套 Layout + `RequireAuth` 严格角色匹配，菜单与权限矩阵对齐 |
| **RTK Query 声明式数据流** | 列表 / 详情 / 变更后 `invalidatesTags` 自动刷新，无需手写 loading-error-cache 样板代码 |
| **Token 刷新与下载** | `baseQueryWithReauth` + `authFetch` 单例 mutex，API 与 CSV 导出共用刷新链路 |
| **组件分层清晰** | `pages/` 薄路由壳 + `components/` 可复用业务组件（`AssetTable`、`WorkflowDetail`、`InventoryTaskDetailView` 等） |
| **盘点体验优化** | `rowsReady` 门控避免缓存命中时空表；`InventorySpreadsheet` 可编辑 Table + 盘盈行 + 冲突行标红 |
| **图表稳定渲染** | `ChartBox` + `ResizeObserver` 解决 Tabs 切换后 G2 偏移；按路由懒加载 charts / Univer chunk |
| **类型安全** | `types/api.ts` 与后端契约对齐；Vitest 覆盖 utils / 登录 / 关键业务逻辑（46+ 用例） |

---

## 性能与 QPS 容量

> 以下容量基于**架构设计目标与 Nginx/中间件配置推导**，非正式压测报告。生产上线前建议用 `wrk` / `k6` 对核心接口实测校准。

### 业务场景基准

高校固定资产系统的真实负载特征：

- **日常**：管理员浏览台账 / 审批，约 **10–50** 并发用户
- **盘点高峰**：每学期集中盘点，约 **30–80** 人同时在线录入（按实验室分区，非全表实时协同）
- **峰值接口**：盘点批量 `POST /inventory/tasks/:id/submit`、资产列表分页、报表导出（异步，不占 API 长尾）

架构文档明确：完整微服务形态面向**可扩展性**设计，但真实并发规模有限，支持**按需降级为单机合并部署**。

### 网关与限流配置（`deploy/nginx/nginx.conf`）

| 规则 | 阈值 | 说明 |
|---|---|---|
| 登录接口 | **10 req/min/IP** | 防暴力破解 |
| 通用 API | **100 req/s/IP** | 单 IP 软上限 |
| 请求体 | **20 MB** | 适配盘点批量提交 |

### 分场景容量估算

| 部署形态 | 估算 API QPS（读） | 估算 API QPS（写） | 说明 |
|---|---|---|---|
| **开发机单机**（1C API 进程/服务） | 200–800 / 服务 | 50–200 / 服务 | 满足本地开发与 E2E |
| **小规模生产**（2C4G × 5 API + 单库实例） | **1,000–3,000** 合计 | **200–500** 合计 | 覆盖 5000 师生、日常管理 |
| **标准生产**（API 2+ 副本 + 读写分离 + Redis 集群） | **5,000–15,000** 合计 | **1,000–3,000** 合计 | Nginx 多实例 + 水平扩容 |
| **盘点高峰** | — | **~100–300** 提交/s | 受 Redis 锁 + MongoDB 写入限制；冲突行返回 40901 可重试 |

**瓶颈与扩展方向**：

| 组件 | 典型瓶颈 | 扩展方式 |
|---|---|---|
| `inventory-api` submit | Redis 锁 + Mongo upsert | 水平扩容 API；Mongo 分片 / 副本集 |
| `report-api` 聚合查询 | PG/MySQL 复杂统计 | 只读副本 + Kafka 物化宽表（`fams_report`） |
| `export-worker` | CSV 生成 CPU | 多 Worker 消费同一 Redis 队列 |
| Kafka Consumer | 消费 Lag | 增加 partition（已预留 ≥3）与 consumer 实例 |

**异步削峰**：报表导出、台账同步、账实比对均走 **Redis / Kafka 队列**，API 快速返回，有效 QPS 远高于同步写路径。

---

## 技术栈

### 后端

| 类别 | 技术 |
|---|---|
| 语言 / 框架 | Go 1.22+、go-zero 1.6+ |
| 关系型库 | PostgreSQL 16、MySQL 8.0 |
| 文档库 | MongoDB 7.0 |
| 缓存 / 锁 | Redis 7.2 |
| 消息队列 | Kafka 3.x（KRaft） |
| 服务发现 | etcd 3.5 |
| 可观测性 | Jaeger、Prometheus、Grafana |
| 反向代理 | Nginx |

### 前端

| 类别 | 技术 |
|---|---|
| 框架 | React 18、TypeScript 5.6 |
| 构建 | Vite 6 |
| UI | Ant Design 5、ProComponents |
| 状态 | Redux Toolkit、RTK Query |
| 路由 | React Router v6 |
| 图表 | @ant-design/charts (G2) |

---

## 项目结构

```
assets-db/
├── README.md                 # 本文件
├── AGENTS.md                 # AI / Cursor 开发指引
├── backend/
│   ├── service/              # 微服务（user/asset/workflow/inventory/report）
│   ├── pkg/                  # 公共库（errx/middleware/redislock/outbox/...）
│   ├── deploy/               # docker-compose、SQL、Nginx、Prometheus、Grafana
│   ├── scripts/              # infra-up / healthcheck 等
│   ├── tests/                # integration / e2e
│   └── doc/                  # 架构设计、API 契约、测试报告
└── frontend/
    ├── src/
    │   ├── pages/            # 按角色划分的页面入口
    │   ├── components/       # 可复用业务组件
    │   ├── store/api/        # RTK Query 端点
    │   ├── layouts/          # Admin / College / User 三套布局
    │   └── utils/            # 业务工具函数
    └── doc/                  # 前端设计、实现蓝图、dev-log
```

---

## 环境要求

| 依赖 | 版本建议 |
|---|---|
| Go | ≥ 1.22（`go.mod` 当前 1.25+） |
| Node.js | ≥ 18（推荐 20+） |
| Docker + Compose | 用于基础设施（PG/MySQL/Mongo/Redis/Kafka 等） |
| 内存 | 本地全栈建议 ≥ 8 GB（14 个基础设施容器） |

---

## 快速启动

### 1. 克隆仓库

```bash
git clone https://github.com/sisyphe550/assets-db.git
cd assets-db
```

### 2. 启动基础设施（Docker）

```bash
cd backend
make infra-up          # 启动 PG / MySQL / Mongo / Redis / Kafka / etcd / Jaeger / Prometheus / Grafana
make healthcheck       # 可选：检查各容器健康状态
```

首次运行会自动从 `deploy/docker/docker-compose-env.example.yml` 复制环境变量文件。

**常用 Makefile 命令**：

```bash
make infra-down        # 停止容器（保留数据卷）
make infra-reset     # 彻底重置（删除 volumes + 重新初始化）
```

### 3. 启动后端微服务

每个服务**单独开一个终端**（开发模式直接 `go run`）：

```bash
cd backend

# 终端 1 — 用户服务
go run service/user/api/user.go
go run service/user/rpc/user.go

# 终端 2 — 资产服务
go run service/asset/api/asset.go
go run service/asset/rpc/asset.go

# 终端 3 — 工单服务
go run service/workflow/api/workflow.go

# 终端 4 — 盘点服务
go run service/inventory/api/inventory.go

# 终端 5 — 报表服务
go run service/report/api/report.go
go run service/report/export-worker/main.go    # CSV 异步导出（与 report-api 配套）
```

**可选 Worker**（完整异步链路）：

```bash
go run service/workflow/outbox-dispatcher/main.go   # Outbox → Kafka
go run service/asset/consumer/main.go             # Kafka → 台账同步
go run service/inventory/comparison-worker/main.go # 盘点账实比对
```

### 4. 启动前端

```bash
cd frontend
npm install
npm run dev          # http://localhost:5173
```

前端通过 `vite.config.ts` 将 `/api/v1/*` 代理到各后端端口，**无需额外配置 CORS**。

### 5. 访问系统

打开浏览器访问 [http://localhost:5173](http://localhost:5173)，使用下方测试账号登录。

---

## 测试账号

密码统一为 **`Test@123456`**

| 用户名 | 角色 | 院系 | 典型用途 |
|---|---|---|---|
| `admin_school` | 校级管理员 | 全校 | 终审、全校资产、发布盘点 |
| `admin_info` | 院级管理员 | 信息工程学院 | 院审、本院资产、用户管理 |
| `student_001` | 师生 | 软件工程实验室 | 领用申请、盘点录入 |
| `student_002` | 师生 | 网络工程实验室 | 盘点冲突测试 |

完整种子数据见 [`backend/doc/05-seed-fixtures.md`](backend/doc/05-seed-fixtures.md)。

---

## 服务端口一览

### HTTP API（业务）

| 服务 | 端口 | 说明 |
|---|---|---|
| user-api | **8888** | 登录、组织树、用户管理 |
| asset-api | **8889** | 资产 CRUD |
| workflow-api | **8890** | 工单申请与审批 |
| inventory-api | **8891** | 盘点任务与草稿提交 |
| report-api | **8892** | 统计报表与导出任务 |

### RPC

| 服务 | 端口 |
|---|---|
| user-rpc | 8081 |
| asset-rpc | 8082 |

### 前端

| 服务 | 端口 |
|---|---|
| Vite Dev Server | **5173** |

### 基础设施（Docker 默认）

| 组件 | 端口 |
|---|---|
| PostgreSQL | 5432 |
| MySQL | 3306 |
| MongoDB | 27017 |
| Redis | 6379 |
| Kafka | 9092 |
| Jaeger UI | 16686 |
| Prometheus | 9090 |
| Grafana | 3000 |

### Nginx 统一入口（生产）

配置见 `backend/deploy/nginx/nginx.conf`，将 `/api/v1/<service>/` 转发至对应 API 端口。

---

## 测试与构建

### 后端

```bash
cd backend

# 单元测试
make test-unit
# 或
go test ./pkg/... -short -v -cover

# 集成测试（需 make infra-up）
make test-integration

# E2E（需全部 API 进程运行）
make test-e2e

# 编译检查
go build ./...
```

### 前端

```bash
cd frontend

npm test             # Vitest，46+ 用例
npm run build        # tsc + vite production build
npm run preview      # 预览生产构建
```

---

## 可观测性

基础设施启动后可直接访问：

| 工具 | 地址 | 用途 |
|---|---|---|
| **Jaeger** | http://localhost:16686 | 分布式链路追踪 |
| **Prometheus** | http://localhost:9090 | 指标查询 |
| **Grafana** | http://localhost:3000 | 监控大盘（默认密码见 `docker-compose-env.yml`） |

预置 Dashboard：`backend/deploy/grafana/dashboards/fams-overview.json`  
包含：服务存活、API QPS、P99 时延、Kafka 消费堆积、审批积压、盘点锁冲突率等 Panel。

---

## 文档索引

### 后端（`backend/doc/`）

| 文档 | 内容 |
|---|---|
| [01-desgin.md](backend/doc/01-desgin.md) | 架构设计、混合存储、盘点/工单流程 |
| [03-api-contract.md](backend/doc/03-api-contract.md) | **API 契约（权威）** |
| [05-seed-fixtures.md](backend/doc/05-seed-fixtures.md) | 测试账号与固定数据 |
| [06-error-codes.md](backend/doc/06-error-codes.md) | 27 个业务错误码 |
| [08-infra-config.md](backend/doc/08-infra-config.md) | Docker / Nginx / Prometheus 配置说明 |
| [10-final-status.md](backend/doc/10-final-status.md) | 后端完成度与全链路验证 |
| [13-release-notes-2026-07-09.md](backend/doc/13-release-notes-2026-07-09.md) | 近期修复汇总 |

### 前端（`frontend/doc/`）

| 文档 | 内容 |
|---|---|
| [01-design.md](frontend/doc/01-design.md) | 技术选型与状态管理设计 |
| [03-pages.md](frontend/doc/03-pages.md) | 21 个页面交互设计 |
| [05-components.md](frontend/doc/05-components.md) | 组件规格与 Props 约定 |
| [07-implementation.md](frontend/doc/07-implementation.md) | 实现蓝图与 RTK Query 端点 |
| [dev-log/](frontend/doc/dev-log/) | 分阶段开发日志 |

### 开发辅助

- [AGENTS.md](AGENTS.md) — Cursor / AI 开发指引与易踩坑说明
- [backend/README.md](backend/README.md) — 后端精简说明
- [frontend/README.md](frontend/README.md) — 前端精简说明

---

## 典型业务路径（验证清单）

启动完成后，可按以下路径冒烟测试：

1. `admin_school` 登录 → 仪表盘 → 资产列表
2. `student_001` 登录 → 新建领用申请 → `admin_info` 院审 → `admin_school` 终审
3. `admin_info` 创建盘点任务 → `student_001` 录入草稿 → 提交 → 院管归档
4. `admin_school` 报表页 → 导出 CSV（确认 `export-worker` 已启动）

---

## License

本项目用于高校固定资产管理场景的教学与工程实践。部署生产环境前请更换 JWT 密钥、数据库密码，并执行安全审计。
