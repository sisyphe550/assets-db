# FAMS 基础设施配置文件说明（v1）

> 对应 P0 交付物。本文档记录 `deploy/`、`scripts/`、`Makefile` 中每个文件的功能职责与关键设计决策，供后续 Phase 实现与运维参考。

---

## 1. 文件清单与功能

### 1.1 `deploy/docker/docker-compose.yml`

**功能**：一键启动全部 14 个基础设施容器，包括数据库、中间件、可观测性组件及对应的 Exporter。

**Service 清单**：

| Service | 镜像 | 功能 |
|---|---|---|
| `postgres` | `postgres:16-alpine` | User/Workflow/Inventory 业务表 |
| `mysql` | `mysql:8.0` | Asset Ledger 台账 + 事件去重 |
| `mongo` | `mongo:7.0` | 盘点草稿高并发暂存 |
| `redis` | `redis:7.2-alpine` | JWT Session / Blacklist / 盘点锁 |
| `kafka` | `apache/kafka:3.9.0` | 领域事件总线（KRaft 单节点） |
| `etcd` | `quay.io/coreos/etcd:v3.5.5` | go-zero gRPC 服务注册发现 |
| `jaeger` | `jaegertracing/all-in-one:1.57.0` | OTLP 分布式链路追踪 |
| `prometheus` | `prom/prometheus:v2.53.0` | 指标采集与时序存储 |
| `grafana` | `grafana/grafana:11.0.0` | 可视化监控大盘 |
| `postgres-exporter` | `prometheuscommunity/postgres-exporter:v0.15.0` | PG 指标暴露 |
| `mysqld-exporter` | `prom/mysqld-exporter:v0.15.1` | MySQL 指标暴露 |
| `redis-exporter` | `oliver006/redis_exporter:v1.59.0` | Redis 指标暴露 |
| `kafka-exporter` | `danielqsj/kafka-exporter:v1.7.0` | Kafka 指标暴露 |
| `mongodb-exporter` | `percona/mongodb_exporter:0.40.0` | MongoDB 指标暴露 |

### 1.2 `deploy/docker/docker-compose-env.yml`

**功能**：本地开发环境变量（已在 `.gitignore` 中排除，不提交仓库）。

### 1.3 `deploy/docker/docker-compose-env.example.yml`

**功能**：环境变量模板文件，包含所有键名与开发默认值，供开发者复制后修改。提交到仓库。

**关键变量**：

| 变量 | 用途 |
|---|---|
| `POSTGRES_USER/PASSWORD/DB` | PG 连接信息 |
| `MYSQL_ROOT_PASSWORD/DATABASE/USER/PASSWORD` | MySQL 连接信息 |
| `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` | JWT 签名密钥（生产必须更换） |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | Token 有效期（2h / 24h） |
| `USER_API_PORT` ~ `REPORT_API_PORT` | 各微服务 HTTP 端口 |
| `KAFKA_TOPIC_LIFECYCLE` / `KAFKA_TOPIC_INVENTORY` | Kafka Topic 名称 |

### 1.4 `deploy/sql/postgres/001_init.sql`

**功能**：PostgreSQL `fams_core` 库 DDL，创建 8 张业务表及全部索引、约束。幂等（`IF NOT EXISTS`）。

**包含表**：`sys_department`、`sys_user`、`workflow_request`、`workflow_log`、`workflow_outbox`、`inventory_task`、`inventory_task_assignee`、`inventory_record`

### 1.5 `deploy/sql/postgres/002_seed.sql`

**功能**：固定 ID 种子数据（5 条组织 + 5 个用户）。密码 bcrypt hash 为固定值（cost=10），供 E2E 测试断言。`ON CONFLICT DO NOTHING` 保证幂等。

### 1.6 `deploy/sql/postgres/003_report_init.sql`

**功能**：PostgreSQL `fams_report` 库 DDL，创建 4 张报表聚合表（`rpt_asset_daily_snapshot`、`rpt_workflow_summary`、`rpt_inventory_diff_summary`、`rpt_export_job`）。

### 1.7 `deploy/sql/mysql/001_init.sql`

**功能**：MySQL `fams_asset` 库 DDL，创建 `asset_ledger`（InnoDB，utf8mb4）和 `asset_event_dedup`（幂等去重表）。

### 1.8 `deploy/sql/mysql/002_seed.sql`

**功能**：5 条种子资产（固定 ID 501–505），`ON DUPLICATE KEY UPDATE` 保证幂等。

### 1.9 `deploy/sql/mongo/001_init.js`

**功能**：MongoDB 初始化脚本，创建 `fams_inventory` 库和 `inventory_draft` 集合，并建立复合唯一索引 `{task_id, asset_no}` 和查询索引 `{task_id, updated_at}`、`{operator_id}`。

### 1.10 `deploy/prometheus/prometheus.yml`

**功能**：Prometheus 抓取配置。定义了 9 个 go-zero 微服务 metrics 目标（端口 9101–9109）、5 个基础设施 Exporter 目标，以及 Prometheus 自监控。抓取周期 15s。

### 1.11 `deploy/grafana/provisioning/datasources/datasources.yml`

**功能**：Grafana 数据源自动配置，对接 Prometheus（默认数据源）和 Jaeger，实现 Metrics to Trace 联动。

### 1.12 `deploy/grafana/provisioning/dashboards/dashboards.yml`

**功能**：Grafana Dashboard 自动加载配置，从 `/etc/grafana/provisioning/dashboards` 目录加载 JSON 面板文件。

### 1.13 `deploy/grafana/dashboards/fams-overview.json`

**功能**：全局监控大盘，包含 7 个 Panel：服务存活、API QPS、P99 时延、Kafka 消费堆积、审批工单积压、盘点锁冲突率、台账同步延迟。

### 1.14 `deploy/nginx/nginx.conf`

**功能**：反向代理配置，将 `/api/v1/<service>/` 路径分发到各微服务 API 进程（`host.docker.internal:8888–8892`）。

**安全措施**：
- 登录接口限流：10r/m per IP
- 通用 API 限流：100r/s per IP
- 客户端 body 限制：20m（适配盘点批量提交）
- 透传 `X-Real-IP`、`X-Forwarded-For` 头

### 1.15 `scripts/infra-up.sh`

**功能**：一键启动基础设施。执行流程：
1. 检查 `docker-compose-env.yml` 存在，缺失则从 `.example` 复制
2. 检测 10 个端口是否被占用，冲突则 exit 1
3. `docker compose up -d`
4. 轮询 9 个容器的 health status，最多等待 120s，超时 exit 1
5. 验证 Kafka Topic 已就绪

### 1.16 `scripts/infra-down.sh`

**功能**：停止并移除所有基础设施容器（保留 volumes）。

### 1.17 `scripts/infra-reset.sh`

**功能**：彻底重置基础设施——`docker compose down -v` 删除 volumes + 清理 `data/` 目录 + 重新启动。

### 1.18 `scripts/healthcheck.sh`

**功能**：逐一检查 9 个核心服务的可用性（PG/MySQL/Mongo/Redis/Kafka/etcd/Jaeger/Prometheus/Grafana），打印 PASS/FAIL 统计，任一失败则 exit 1。

### 1.19 `Makefile`

**功能**：统一命令入口。

| 命令 | 作用 |
|---|---|
| `make infra-up` | 启动基础设施 |
| `make infra-down` | 停止基础设施 |
| `make infra-reset` | 重置基础设施 |
| `make healthcheck` | 健康检查 |
| `make test-unit` | 单元测试（`-short`） |
| `make test-integration` | 集成测试（`-tags=integration`） |
| `make test-e2e` | E2E 测试（`-tags=e2e`） |
| `make test-all` | 全部测试 |
| `make lint` | golangci-lint |
| `make clean` | 清理所有容器与数据 |

### 1.20 `.gitignore`

**功能**：排除 `docker-compose-env.yml`（含密钥）、`data/` 目录（数据库持久化文件）、Go 构建产物、IDE 配置。

---

## 2. 关键设计决策

### 2.1 Kafka：KRaft 模式 + 双 Listener

**决策**：使用 `apache/kafka:3.9.0` 的 KRaft 模式（无需 ZooKeeper），配置双 Listener：
- `PLAINTEXT://0.0.0.0:9094` — 供宿主机 localhost 访问
- `PLAINTEXT_CONTAINER://kafka:9092` — 供同网络容器通过 service name 访问

**原因**：go-zero 服务在宿主机 `go run` 时通过 `localhost:9094` 连接 Kafka；同一 compose 内的 Exporter 通过 `kafka:9092` 连接。KRaft 消除 ZooKeeper 依赖，降低开发环境复杂度（参照用户提供的 polaris-io 项目经验）。

### 2.2 etcd：quay.io 镜像 + 显式 command

**决策**：使用 `quay.io/coreos/etcd:v3.5.5`（非 bitnami），通过 `command` 显式指定全部启动参数。

**原因**：quay.io 镜像无使用限制，`command` 显式声明使配置完全可见、可审计，避免环境变量隐式行为的调试困难。`--quota-backend-bytes=8GB` 给出充足的开发环境存储空间。

### 2.3 Jaeger：badger 内存存储 + OTLP

**决策**：开发环境使用 `badger` 存储引擎（持久化到 volume），不依赖 Elasticsearch；同时暴露 OTLP gRPC (4317)、OTLP HTTP (4318)、旧版 HTTP Collector (14268) 三个端口。

**原因**：设计文档 §6.5 明确"开发环境采用 all-in-one 单体部署（内存存储）"。选择 badger 而非纯内存（`BADGER_EPHEMERAL=false`）可保留重启后的 trace 数据。OTLP gRPC (4317) 是 go-zero Telemetry 的默认上报端口（设计 §6.2.1），其他端口为兼容性保留。

### 2.4 Jaeger 镜像版本：锁定 1.57.0

**决策**：使用 `jaegertracing/all-in-one:1.57.0` 而非示例项目中的 `1.63.0`。

**原因**：设计文档 P0.2 服务清单明确写定 1.57，且 Bitnami Jaeger Agent 端口（6831/6832/5775/5778）在 OTLP 方案下不再需要，仅暴露实际使用的端口（4317/4318/16686/14268/14250）。保持与设计文档一致，避免镜像版本不同导致的兼容性问题。

### 2.5 网络：bridge 模式 + fams-net

**决策**：使用 `fams-net` bridge 网络（`driver: bridge`，非 `external: true`），compose 自动管理网络生命周期。

**原因**：设计文档 §7.5 约定"所有服务加入 `fams-net` bridge 网络"。采用 compose 内置 bridge（而非外部网络）使 `infra-reset.sh` 可以完整拆除/重建网络。若需要跨 compose 通信，后续可切换为 external 模式。

### 2.6 数据库连接主机名：compose service name

**决策**：容器间互访使用 compose service name 作为主机名（如 `postgres:5432`、`mysql:3306`），不依赖 IP。

**原因**：设计文档 §7.5 明确"服务间以 compose service name 互访"。bridge 网络内置 DNS 解析，service name 是稳定标识，compose 重启后 IP 变化无影响。

### 2.7 健康检查：全部容器配置 healthcheck

**决策**：所有有 CLI 的容器都配置 `healthcheck`，`infra-up.sh` 轮询等待全部 healthy 后才返回。

**原因**：设计文档 P0.6 要求"容器 unhealthy 时 compose 配置 healthcheck，脚本等待最多 120s，超时 exit 1"。PostgreSQL 使用 `pg_isready`、MySQL 使用 `mysqladmin ping`、MongoDB 使用 `mongosh eval ping`、etcd 使用 `etcdctl endpoint health`；Kafka 使用 `kafka-topics.sh --list` 验证 Broker 完全就绪（而非仅端口监听）。

### 2.8 API 端口：与业务进程分离

**决策**：docker-compose 不包含业务 Go 进程，仅暴露 `USER_API_PORT=8888` ~ `REPORT_API_PORT=8892` 作为环境变量占位，Nginx 通过 `host.docker.internal` 代理到宿主机端口。

**原因**：设计文档 §7.5 明确"docker-compose 不包含业务 Go 进程（业务进程本地 go run 或后续独立 compose 叠加）"。这种分离使开发迭代无需重新构建镜像，`go run` + 热重载即可。

### 2.9 Redis：无密码 + appendonly

**决策**：开发环境不设 Redis 密码（`requirepass`），开启 `appendonly yes` 持久化。

**原因**：go-zero 框架的 Redis 配置方式在无密码时更简洁。开发环境 `fams-net` 内网隔离，无外部暴露风险。生产环境应通过 `docker-compose-env.yml` 添加密码。

### 2.10 MongoDB：无认证

**决策**：开发环境不启 MongoDB 认证（不设 `MONGO_INITDB_ROOT_USERNAME`）。

**原因**：同上，内网隔离 + 降低开发配置复杂度。生产环境必须启用认证。

### 2.11 镜像版本：全部写死 tag

**决策**：所有镜像使用具体版本号（如 `postgres:16-alpine`），禁止 `latest`。

**原因**：设计文档 P0.2 明确要求"镜像版本写死 tag，禁止 latest"。固定版本保证 CI 与本地环境一致性，避免上游镜像更新导致不可复现的问题。

### 2.12 数据持久化：named volume → bind mount

**决策**：使用 bind mount（`../../data/<svc>/data:/...`）而非 named volume，数据存储在仓库同级 `data/` 目录下。

**原因**：`infra-reset.sh` 可以简单 `rm -rf data/` 清理，开发者可以直接查看/备份数据文件。`.gitignore` 排除 `data/` 目录避免误提交。

### 2.13 时区：全部设置 `TZ=Asia/Shanghai`

**决策**：所有容器均设置 `TZ=Asia/Shanghai`（参照用户提供的示例项目）。

**原因**：确保数据库 `timestamptz` / `datetime` 的默认时区一致，避免时间比较与日志时间戳的时区错误。

---

## 3. 端口分配汇总

| 端口 | 服务 | 用途 |
|---|---|---|
| 5432 | PostgreSQL | 连接端口 |
| 3306 | MySQL | 连接端口 |
| 27017 | MongoDB | 连接端口 |
| 6379 | Redis | 连接端口 |
| 9092 | Kafka | 容器间 PLAINTEXT_CONTAINER |
| 9094 | Kafka | 宿主机 PLAINTEXT |
| 2379 | etcd | 客户端通信 |
| 2380 | etcd | peer 通信 |
| 4317 | Jaeger | OTLP gRPC（go-zero 上报） |
| 4318 | Jaeger | OTLP HTTP |
| 16686 | Jaeger | Web UI |
| 9090 | Prometheus | Web UI + API |
| 3000 | Grafana | Web UI |
| 9187 | PG Exporter | Prometheus scrape |
| 9104 | MySQL Exporter | Prometheus scrape |
| 9121 | Redis Exporter | Prometheus scrape |
| 9308 | Kafka Exporter | Prometheus scrape |
| 9216 | MongoDB Exporter | Prometheus scrape |

业务进程端口（不在 compose 内，供 Nginx 转发）：

| 端口 | 服务 |
|---|---|
| 8888 | user-api |
| 8889 | asset-api |
| 8890 | workflow-api |
| 8891 | inventory-api |
| 8892 | report-api |

Metrics 端口（不在 compose 内，供 Prometheus 抓取）：

| 端口 | 服务 |
|---|---|
| 9101 | user-api |
| 9102 | user-rpc |
| 9103 | asset-api |
| 9104 | asset-rpc |
| 9105 | workflow-api |
| 9106 | workflow-rpc |
| 9107 | inventory-api |
| 9108 | inventory-rpc |
| 9109 | report-api |

---

## 4. 与设计文档的对应关系

| 本文档内容 | 对应设计文档 |
|---|---|
| docker-compose service 清单 | `01-desgin.md` §7.5 + `02-plan.md` P0.2 |
| 环境变量定义 | `02-plan.md` P0.3 |
| PostgreSQL DDL 表结构 | `01-desgin.md` §4.2.1–4.2.5, §4.2.7–4.2.10 |
| MySQL DDL 表结构 | `01-desgin.md` §4.2.3, §7.6 |
| MongoDB 集合与索引 | `01-desgin.md` §4.2.6 + `02-plan.md` P0.4 |
| Report 库表 | `01-desgin.md` §4.2.11–4.2.12 |
| Seed 数据 | `05-seed-fixtures.md` §1–§3 |
| Prometheus scrape 目标 | `01-desgin.md` §6.3.1, §7.7 |
| Grafana 面板 PromQL | `01-desgin.md` §6.4.1 + `02-plan.md` P0.2 |
| Nginx 路由 | `02-plan.md` P8 |
| Metrics 端口 | `01-desgin.md` §7.7 |
| gRPC 端口 | `01-desgin.md` §7.8 |
| 测试命令 | `02-plan.md` P0.7 |
| Git 工作流 | `01-desgin.md` §7.2 |

---

*文档版本：v1.0 | 2026-07-07*
