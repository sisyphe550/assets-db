# FAMS 后端开发任务书（AI Agent 执行版）

> **指导文档**：`doc/01-desgin.md`（含第七章工程实现规范）  
> **执行者**：AI Agent / 开发 Agent  
> **仓库根路径**：`assets-db/backend/`  
> **原则**：每个 Phase 独立分支 → 实现 → 测试 → 推送 → PR → Squash 合并 `main`；不得跳过测试；不得跨 Phase 偷跑依赖。

---

## 0. 全局执行约定

### 0.1 阶段与分支对照表

| Phase | 分支名 | 依赖 | 合并至 |
| --- | --- | --- | --- |
| P0 | `feat/p0-infra-docker` | — | `main` |
| P1 | `feat/p1-shared-pkg` | P0 | `main` |
| P2 | `feat/p2-user-service` | P1 | `main` |
| P3 | `feat/p3-asset-service` | P1 | `main` |
| P4 | `feat/p4-workflow-service` | P2, P3 | `main` |
| P5 | `feat/p5-inventory-service` | P3, P4(部分) | `main` |
| P6 | `feat/p6-report-service` | P3, P4, P5 | `main` |
| P7 | `feat/p7-observability` | P0, 各服务骨架 | `main` |
| P8 | `feat/p8-nginx-gateway` | P2–P6 | `main` |
| P9 | `feat/p9-e2e-integration` | 全部 | `main` |

### 0.2 每个 Phase 标准 Git 流程（必须逐步执行）

```bash
# 1. 同步主分支
git checkout main && git pull origin main

# 2. 创建功能分支
git checkout -b feat/pX-xxx

# 3. 开发过程中按子任务 commit（示例）
git add <files>
git commit -m "feat(user): implement login handler with jwt blacklist check"

# 4. 阶段验收测试全部通过后推送
git push -u origin feat/pX-xxx

# 5. 创建 PR（使用 gh 或平台 UI），标题格式：[P2] User Service - login & auth
# 6. CI/本地测试通过后 Squash Merge 至 main
# 7. 合并后删除远程分支，本地同步
git checkout main && git pull origin main
git branch -d feat/pX-xxx
```

### 0.3 每个 Phase 交付物检查清单（Definition of Done）

- [ ] 代码符合 `01-desgin.md` 第七章目录结构
- [ ] 本 Phase「测试」章节命令全部通过
- [ ] 新增/变更 API 与错误码已登记在 `pkg/errx/codes.go`
- [ ] 配置项写入 `deploy/docker/docker-compose-env.yml` 示例键名
- [ ] 无硬编码密钥；敏感项仅出现在 `.example` 或 env 文件
- [ ] PR 描述含：变更摘要、测试命令输出摘要、已知限制

### 0.4 技术栈版本锁定

| 组件 | 版本 |
| --- | --- |
| Go | ≥ 1.22 |
| go-zero | ≥ 1.6 |
| PostgreSQL | 16 |
| MySQL | 8.0 |
| MongoDB | 7.0 |
| Redis | 7.2 |
| Kafka | 3.x（KRaft 单节点） |
| etcd | 3.5 |

---

## P0：基础设施与 Docker Compose

**分支**：`feat/p0-infra-docker`  
**目标**：一键启动全部底层依赖，提供 DDL 初始化脚本与健康检查。

### P0.1 任务范围

| 子任务 ID | 内容 | 产出路径 |
| --- | --- | --- |
| P0-1 | 创建 `deploy/docker/docker-compose.yml` | 见下方服务清单 |
| P0-2 | 创建 `deploy/docker/docker-compose-env.yml` | 全部 env 键 |
| P0-3 | PostgreSQL 初始化 DDL | `deploy/sql/postgres/001_init.sql` |
| P0-4 | MySQL 初始化 DDL | `deploy/sql/mysql/001_init.sql` |
| P0-5 | MongoDB init（索引） | `deploy/sql/mongo/001_init.js` |
| P0-6 | 本地脚本 | `scripts/infra-up.sh`, `scripts/infra-down.sh`, `scripts/infra-reset.sh` |
| P0-7 | Makefile 入口 | `Makefile`（`make infra-up` 等） |

### P0.2 docker-compose.yml 服务清单

必须包含以下 service（镜像版本写死 tag，禁止 `latest`）：

| Service | 镜像 | 挂载/初始化 |
| --- | --- | --- |
| `postgres` | `postgres:16-alpine` | 挂载 `deploy/sql/postgres/` 至 `/docker-entrypoint-initdb.d/` |
| `mysql` | `mysql:8.0` | 挂载 `deploy/sql/mysql/` |
| `mongo` | `mongo:7.0` | 启动后 exec init 脚本创建索引 |
| `redis` | `redis:7.2-alpine` | — |
| `kafka` | `bitnami/kafka:3.7` 或等效 | KRaft 单节点，预创建 Topic |
| `etcd` | `bitnami/etcd:3.5` | 供 go-zero rpc 注册 |
| `jaeger` | `jaegertracing/all-in-one:1.57` | OTLP 4317 暴露 |
| `prometheus` | `prom/prometheus:v2.53.0` | 挂载 `deploy/prometheus/prometheus.yml` |
| `grafana` | `grafana/grafana:11.0.0` | 挂载 provisioning 目录 |
| `postgres-exporter` | 官方 exporter | 依赖 postgres |
| `mysqld-exporter` | 官方 exporter | 依赖 mysql |
| `redis-exporter` | 官方 exporter | 依赖 redis |

**Kafka 预创建 Topic**（通过 init 容器或 kafka 配置）：

- `fams-asset-lifecycle-events`（分区数 ≥ 3，按 `asset_id` 哈希）
- `fams-inventory-comparison-tasks`（分区数 ≥ 3）

### P0.3 docker-compose-env.yml 必须定义的键

```yaml
# 示例结构（实际文件为 KEY=VALUE 或 env_file 格式）
POSTGRES_USER=fams
POSTGRES_PASSWORD=fams_dev_pass
POSTGRES_DB=fams_user
POSTGRES_PORT=5432

MYSQL_ROOT_PASSWORD=fams_root_pass
MYSQL_DATABASE=fams_asset
MYSQL_USER=fams
MYSQL_PASSWORD=fams_dev_pass
MYSQL_PORT=3306

MONGO_INITDB_DATABASE=fams_inventory
MONGO_PORT=27017

REDIS_PORT=6379

KAFKA_BROKERS=kafka:9092
ETCD_HOST=etcd:2379

JAEGER_OTLP_ENDPOINT=jaeger:4317
PROMETHEUS_PORT=9090
GRAFANA_PORT=3000
GRAFANA_ADMIN_PASSWORD=fams_dev_pass

JWT_ACCESS_SECRET=dev_access_secret_change_me
JWT_REFRESH_SECRET=dev_refresh_secret_change_me
JWT_ACCESS_TTL=2h
JWT_REFRESH_TTL=24h
```

同时提供 `deploy/docker/docker-compose-env.example.yml` 提交仓库；真实 env 文件加入 `.gitignore`。

### P0.4 DDL 要求

**PostgreSQL `001_init.sql`** 必须创建 schema 并包含 `01-desgin.md` 4.2.1–4.2.5、4.2.7–4.2.9 全部表及索引：

- `sys_department`, `sys_user`
- `workflow_request` + `uk_asset_open_request` 部分唯一索引
- `workflow_log`, `workflow_outbox`
- `inventory_task`, `inventory_record` + `uk_task_asset` 部分唯一索引

**MySQL `001_init.sql`** 必须包含：

- `asset_ledger`（4.2.3）
- `asset_event_dedup`（7.6 幂等去重表）

**MongoDB `001_init.js`** 必须创建：

- 库 `fams_inventory`
- 集合 `inventory_draft`
- 复合索引 `{ task_id: 1, asset_no: 1 }` 唯一
- 索引 `{ task_id: 1, updated_at: 1 }`

### P0.5 Seed 数据（`deploy/sql/postgres/002_seed.sql`）

| 数据 | 要求 |
| --- | --- |
| 组织树 | 根节点「本校」+ 至少 1 个学院 + 2 个实验室，`path` 正确 |
| 用户 | 校级管理员 1、学院管理员 1、普通师生 2；密码统一 `Test@123456` 的 bcrypt hash |
| 资产 | 在 MySQL seed 中至少 5 条，`department_id` 与 PG 组织对应 |

### P0.6 边界条件与报错处理

| 场景 | 期望行为 |
| --- | --- |
| 端口已被占用 | `infra-up.sh` 检测并打印冲突端口，exit 1 |
| 容器 unhealthy | compose 配置 `healthcheck`，脚本等待最多 120s，超时 exit 1 |
| DDL 重复执行 | 使用 `IF NOT EXISTS` / idempotent migration |
| env 文件缺失 | 脚本提示复制 `.example` 文件 |

### P0.7 测试

```bash
# 启动
make infra-up

# 健康检查脚本（需实现 scripts/healthcheck.sh）
./scripts/healthcheck.sh
# 期望：postgres/mysql/mongo/redis/kafka/etcd/jaeger/prometheus/grafana 全部 OK

# 验证 DDL
docker exec -it fams-postgres psql -U fams -d fams_user -c "\dt"
docker exec -it fams-mysql mysql -ufams -pfams_dev_pass fams_asset -e "SHOW TABLES;"

# 停止
make infra-down
```

### P0.8 验收标准

- [ ] `make infra-up && ./scripts/healthcheck.sh` 一次通过
- [ ] 两个 Kafka Topic 已存在
- [ ] seed 用户可查询到且 `path` 字段符合 `/1/xx/` 格式
- [ ] PR 合并后 `main` 分支包含完整 deploy 目录

### P0.9 Git 操作

```bash
git checkout -b feat/p0-infra-docker
# ... 开发与测试 ...
git push -u origin feat/p0-infra-docker
# PR 标题：[P0] Infrastructure docker-compose and DDL init
# Squash merge → main
```

---

## P1：公共库（pkg）

**分支**：`feat/p1-shared-pkg`  
**依赖**：P0  
**目标**：实现全服务复用的错误码、JWT 中间件、组织子树、Redis 锁、Outbox 与 Kafka 封装。

### P1.1 任务拆分

| 子任务 ID | 包路径 | 功能 |
| --- | --- | --- |
| P1-1 | `pkg/errx/` | 错误码常量、HTTP/gRPC 映射、`BizError` 类型 |
| P1-2 | `pkg/middleware/jwt.go` | Access Token 解析、Claims 注入 context |
| P1-3 | `pkg/middleware/blacklist.go` | Redis `fams:auth:blacklist:${jti}` 校验 |
| P1-4 | `pkg/middleware/deptscope.go` | 从 context 取 uid → 调 user-rpc 或本地缓存取子树 IDs |
| P1-5 | `pkg/dept/path.go` | `BuildPath`, `ParseSubtreeIDs(pathPrefix)` |
| P1-6 | `pkg/redislock/lock.go` | `TryLock`, `UnlockLua`（校验 owner 再 DEL） |
| P1-7 | `pkg/outbox/dispatcher.go` | 轮询 PG outbox → Kafka，指数退避，最大重试 10 次 |
| P1-8 | `pkg/kafka/producer.go` | 发送消息 + `traceparent` header |
| P1-9 | `pkg/kafka/consumer.go` | 消费 + 从 header 恢复 trace context |

### P1.2 `pkg/errx` 实现要求

```go
// 必须实现
type BizError struct {
    Code    int
    Message string
    HTTP    int
}

func (e *BizError) Error() string
func ToHttpError(err error) (code int, httpStatus int, message string)
func ToGrpcError(err error) error
```

**必须注册的错误码**（与 01-desgin 7.3 一致）：`40101`, `40102`, `40301`, `40302`, `40401`, `40901`, `40902`, `42201`, `50001`, `50301`。

**边界**：

- 未知 error → 映射 `50001`，日志记录原始 error，响应不含堆栈
- `BizError` 可直接返回给 API 层

### P1.3 JWT 中间件

**Claims 结构**：

```go
type FamsClaims struct {
    jwt.RegisteredClaims
    UID       int64  `json:"uid"`
    RoleLevel int16  `json:"role_level"`
    DeptID    int64  `json:"dept_id"`
}
```

**校验顺序**：

1. Authorization Bearer 存在性
2. 签名与过期（Access Token）
3. Redis 黑名单检查 `jti`
4. 写入 context：`UidKey`, `RoleLevelKey`, `DeptIDKey`

**报错**：

| 条件 | code |
| --- | --- |
| 无 Header | 40101 |
| 过期/签名错 | 40101 |
| 黑名单命中 | 40102 |

### P1.4 组织子树 `pkg/dept/path.go`

```go
// ParseSubtreeIDs 输入 dept 表 rows（id, path），和管理员 deptID
// 返回该管理员可见的全部 dept_id 列表（含自身及子孙）
func SubtreeIDs(all []Department, rootDeptID int64) ([]int64, error)
```

**测试用例（单元测试必须覆盖）**：

- 校级管理员（role=1）→ 返回 nil 表示不限制（由调用方解释为全量）
- 学院管理员 → 含学院 ID + 下属实验室 IDs
- 实验室管理员 → 仅自身 ID
- 根节点不存在 → 返回 40401

### P1.5 Redis 分布式锁

```go
func TryLock(ctx context.Context, rds *redis.Redis, key, owner string, ttl time.Duration) (bool, error)
func Unlock(ctx context.Context, rds *redis.Redis, key, owner string) error // Lua 脚本
```

**Lua 脚本逻辑**：

```
if redis.call("get", KEYS[1]) == ARGV[1] then return redis.call("del", KEYS[1]) else return 0 end
```

**报错**：

- Redis 连接失败 → 50301
- 抢锁失败（NX 未设置）→ 40901（由调用方决定是否逐条返回）

### P1.6 Outbox Dispatcher

**轮询 SQL**：

```sql
SELECT * FROM workflow_outbox WHERE status = 0 ORDER BY id LIMIT 100 FOR UPDATE SKIP LOCKED;
```

**投递成功**：更新 `status=1`, `sent_at=now()`  
**投递失败**：`retry_count++`，退避 `min(2^retry_count, 300)` 秒  
**超过 10 次**：标记 `status=2`（死信），记录日志，触发 metric（后续 P7）

**边界**：

- Kafka 不可用：不更新 status，等待下轮
- 重复投递：消费者幂等，允许 at-least-once

### P1.7 测试

```bash
go test ./pkg/... -short -v -cover

# 集成测试（需 Redis）
go test ./pkg/redislock/... -tags=integration -v
go test ./pkg/dept/... -v
```

**覆盖率要求**：`pkg/dept`, `pkg/errx`, `pkg/redislock` ≥ 80%。

### P1.8 验收与 Git

```bash
git checkout main && git pull
git checkout -b feat/p1-shared-pkg
# 完成后
git push -u origin feat/p1-shared-pkg
# PR：[P1] Shared pkg - errx jwt dept redislock outbox kafka
```

---

## P2：用户与权限微服务（User Service）

**分支**：`feat/p2-user-service`  
**依赖**：P0, P1  
**目标**：完成 `user-api` + `user-rpc`、登录/刷新/登出、组织 CRUD、JWT 签发与黑名单。

### P2.1 目录结构

```
service/user/
├── api/
│   ├── user.api                 # go-zero API 定义
│   ├── internal/handler/
│   ├── internal/logic/
│   └── user.go
├── rpc/
│   ├── user.proto
│   ├── internal/logic/
│   └── user.go
└── model/                       # sqlx 生成或手写
    ├── sysusermodel.go
    └── sysdepartmentmodel.go
```

### P2.2 API 接口清单

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/v1/user/login` | 否 | 登录，返回 access + refresh token |
| POST | `/api/v1/user/refresh` | Refresh Token | 续期 |
| POST | `/api/v1/user/logout` | Access Token | 写黑名单 + 删 Redis session |
| GET | `/api/v1/user/me` | 是 | 当前用户信息 |
| GET | `/api/v1/user/departments/tree` | 是 | 组织树（按权限裁剪） |
| POST | `/api/v1/user/departments` | role=1 | 新增组织节点 |
| PUT | `/api/v1/user/users/:id/status` | role=1,2 | 禁用/启用，写黑名单 |

### P2.3 RPC 接口清单（user.proto）

```protobuf
service User {
  rpc FindUser(FindUserReq) returns (FindUserResp);
  rpc GetDeptSubtree(GetDeptSubtreeReq) returns (GetDeptSubtreeResp);
  rpc ValidateToken(ValidateTokenReq) returns (ValidateTokenResp);
}
```

`GetDeptSubtree`：供其他服务做数据隔离，返回 dept_id 列表。

### P2.4 登录流程实现（对齐 5.1）

1. 查 `sys_user` + `sys_department`，`status=0` → 40301「账户已禁用」
2. bcrypt 比对失败 → 40101「用户名或密码错误」（不区分细节，防枚举）
3. 签发 Access（2h）+ Refresh（24h），Claims 含 uid/role_level/dept_id/jti
4. Redis `SET fams:auth:token:${uid}` = accessToken，TTL=2h
5. 返回 `{ accessToken, refreshToken, expiresIn }`

**Refresh 边界**：

- Refresh Token 过期 → 40101，要求重新登录
- 用户已被禁用 → 40102

**Logout**：

- `SET fams:auth:blacklist:${jti}` TTL=剩余 access 有效期
- `DEL fams:auth:token:${uid}`

### P2.5 组织树 path 维护

新增/移动部门时必须事务内更新：

- 新节点 `path = parent.path + id + "/"`  
- 移动节点：批量更新子孙 `path` 前缀

**报错**：

- `dept_code` 重复 → 40901
- 父节点不存在 → 40401
- 删除有子节点/用户的部门 → 42201

### P2.6 测试

```bash
# 单元
go test ./service/user/... -short -v

# 集成（docker-compose 运行中）
go test ./tests/integration/user/... -tags=integration -v
```

**集成测试用例**：

| 用例 | 断言 |
| --- | --- |
| 正确密码登录 | code=0，返回 token |
| 错误密码 | code=40101 |
| 禁用用户登录 | code=40301 |
| 登录后 logout 再访问 /me | code=40102 |
| 学院管理员查组织树 | 仅看到本院及子节点 |
| 校级管理员查组织树 | 看到全校 |

### P2.7 验收与 Git

```bash
git checkout -b feat/p2-user-service
go run service/user/rpc/user.go -f service/user/rpc/etc/user.yaml
go run service/user/api/user.go -f service/user/api/etc/user-api.yaml
git push -u origin feat/p2-user-service
# PR：[P2] User service - login auth dept tree
```

---

## P3：固定资产台账微服务（Asset Service）

**分支**：`feat/p3-asset-service`  
**依赖**：P0, P1（P2 可选，部门 ID 先用 seed 数据）  
**目标**：资产 CRUD、状态变更 RPC、Kafka 消费者幂等同步。

### P3.1 API 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/v1/asset/assets` | role≤2 + 子树 | 新增资产 |
| GET | `/api/v1/asset/assets` | 子树隔离 | 分页列表 |
| GET | `/api/v1/asset/assets/:id` | 子树隔离 | 详情 |
| PUT | `/api/v1/asset/assets/:id` | role≤2 + 子树 | 编辑 |
| DELETE | `/api/v1/asset/assets/:id` | role≤2 | 逻辑删除（status=0 或 deleted_at） |
| GET | `/api/v1/asset/assets/shared` | role=3 | 学院内共享资产只读 |

**列表过滤**：`department_id IN (:subtree)`；校级不过滤。

### P3.2 RPC 接口（asset.proto）

```protobuf
rpc GetAsset(GetAssetReq) returns (GetAssetResp);
rpc CheckAssetAvailable(CheckAssetAvailableReq) returns (CheckAssetAvailableResp); // status 必须为在库
rpc ChangeAssetStatus(ChangeAssetStatusReq) returns (ChangeAssetStatusResp);     // 内部/Kafka 调用
```

`CheckAssetAvailable` 返回 `department_id` 供 workflow 冗余。

**状态机约束**：

| 当前 status | 允许变更为 |
| --- | --- |
| 1 在库 | 2 领用中, 3 维修中, 4 已报废 |
| 2 领用中 | 1 在库, 3 维修中 |
| 3 维修中 | 1 在库, 2 领用中 |
| 4 已报废 | 不可变更 |

非法转换 → RPC 返回 42201。

### P3.3 Kafka 消费者（asset-rpc 内嵌或独立 `asset-consumer`）

**Topic**：`fams-asset-lifecycle-events`

**处理流程**：

1. 从 header 恢复 trace（P1 kafka 包）
2. `INSERT INTO asset_event_dedup(request_id, event_type, processed_at)` 
3. 若 Duplicate entry → ACK 跳过
4. 解析 payload，调用 `ChangeAssetStatus`
5. 记录 `fams_asset_sync_lag_seconds`（event timestamp → now）

**报错**：

- JSON 非法 → 记录 dead letter 日志，ACK（避免阻塞）
- 资产不存在 → 40401，ACK + 告警
- MySQL 不可用 → 不 ACK，等待重试

### P3.4 测试

```bash
go test ./service/asset/... -short -v
go test ./tests/integration/asset/... -tags=integration -v
```

**必测**：

- 重复 `asset_no` 创建 → 40901
- 跨学院查询资产 → 40302
- 消费者重复 message → 仅更新一次
- 非法状态转换 → 42201

### P3.5 Git

```bash
git checkout -b feat/p3-asset-service
git push -u origin feat/p3-asset-service
# PR：[P3] Asset service - CRUD status kafka consumer
```

---

## P4：工作流审批微服务（Workflow Service）

**分支**：`feat/p4-workflow-service`  
**依赖**：P2, P3  
**目标**：两级审批、Outbox、Dispatcher 进程、对账任务。

### P4.1 组件

| 进程 | 说明 |
| --- | --- |
| `workflow-api` | HTTP 入口 |
| `workflow-rpc` | 状态机与事务 |
| `workflow-outbox-dispatcher` | 独立进程，调用 `pkg/outbox` |

### P4.2 API 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/v1/workflow/requests` | role=3 | 创建申请（领用/归还/报修/报废） |
| GET | `/api/v1/workflow/requests` | 按角色过滤 | 我的申请 / 待办 |
| GET | `/api/v1/workflow/requests/:id` | 相关人可见 | 详情含 log |
| POST | `/api/v1/workflow/requests/:id/approve` | role≤2 | 同意 |
| POST | `/api/v1/workflow/requests/:id/reject` | role≤2 | 驳回 |
| POST | `/api/v1/workflow/requests/:id/transfer` | role≤2 | 转办（可选） |

### P4.3 创建工单逻辑（5.2）

```
1. gRPC asset.CheckAssetAvailable(asset_id)
   - 失败 → 42201 / 40401
2. INSERT workflow_request (status=1, current_stage=1, department_id=资产部门)
   - 唯一索引冲突 → 40902「该资产已有进行中的申请」
3. INSERT workflow_log (action='提交申请')
```

### P4.4 审批状态机

| 操作 | 条件 | 结果 |
| --- | --- | --- |
| 院级同意 | role=2, stage=1, dept 在子树内 | stage=2 |
| 校级同意 | role=1, stage=2 | status=2, stage=3 + log + outbox |
| 驳回 | role≤2, 对应阶段权限 | status=3 + log |
| 越权审批 | dept 不在子树 | 40302 |

**终审事务（同一 PG 事务）**：

1. 再次 `CheckAssetAvailable`
2. UPDATE workflow_request
3. INSERT workflow_log
4. INSERT workflow_outbox（event_type 按 type 映射）

### P4.5 Outbox 事件类型

| type | event_type | target_status |
| --- | --- | --- |
| 领用 | ASSET_USE_APPROVED | 2 |
| 归还 | ASSET_RETURN_APPROVED | 1 |
| 报修 | ASSET_REPAIR_APPROVED | 3 |
| 报废 | ASSET_SCRAP_APPROVED | 4 |

### P4.6 对账任务（`workflow-reconciler` 定时）

每 5 分钟：

- 查 `status=2 AND outbox 已投递 AND asset 未同步超过 30s` 的工单
- 重新插入 outbox 或重发 Kafka
- 递增 metric / 日志告警

### P4.7 测试

```bash
go test ./service/workflow/... -short -v
go test ./tests/integration/workflow/... -tags=integration -v
```

**集成测试全链路**：

1. 师生登录 → 创建领用申请
2. 学院管理员初审通过
3. 校级管理员终审通过
4. 等待 consumer → 断言 MySQL asset status=2, user_id=申请人
5. 并发双申请同一 asset → 一个 40902

### P4.8 Git

```bash
git checkout -b feat/p4-workflow-service
git push -u origin feat/p4-workflow-service
# PR：[P4] Workflow service - approval outbox dispatcher reconciler
```

---

## P5：多人协同盘点微服务（Inventory Service）

**分支**：`feat/p5-inventory-service`  
**依赖**：P3, P0  
**目标**：盘点任务、批量草稿提交、Redis 锁、MongoDB CAS、比对 Worker。

### P5.1 API 接口

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| POST | `/api/v1/inventory/tasks` | role≤2 | 创建盘点任务 |
| GET | `/api/v1/inventory/tasks` | role≤2 | 任务列表 |
| POST | `/api/v1/inventory/tasks/:id/submit` | role≤3 | 批量提交草稿 |
| POST | `/api/v1/inventory/tasks/:id/archive` | role≤2 | 终结归档 |
| GET | `/api/v1/inventory/tasks/:id/records` | 子树 | 比对结果 |

### P5.2 批量提交请求/响应

**请求**：

```json
{
  "items": [
    {
      "asset_no": "EQUIP-2026-0091",
      "modified_cells": { "actual_location": "...", "temp_notes": "..." },
      "expected_updated_at": "2026-07-06T10:00:00Z"
    }
  ]
}
```

**响应（部分成功）**：

```json
{
  "code": 0,
  "data": {
    "success": ["EQUIP-2026-0091"],
    "conflicts": [
      { "asset_no": "EQUIP-2026-0092", "code": 40901, "message": "资产正在被他人盘点" }
    ]
  }
}
```

**逐条处理伪代码**：

```
for item in items:
  ok = TryLock(asset_no, operator_id, 30s)
  if !ok → append conflicts; continue
  defer Unlock(asset_no, operator_id)
  err = mongo.UpsertDraftCAS(task_id, asset_no, item, expected_updated_at)
  if err == ErrVersionConflict → append conflicts; continue
  if err != nil → append failures(50301); continue
  append success
```

### P5.3 归档与比对

**archive**：

1. 校验任务 status=1，当前时间在窗口内或管理员强制
2. Mongo 聚合该 task 全部 draft
3. PG 事务批量 INSERT inventory_record（is_scanned=1）
4. 未出现在 draft 中的账面资产 → 后续 Worker 判盘亏
5. UPDATE task status=2
6. 发送 Kafka `fams-inventory-comparison-tasks`

**comparison-worker**：

- 消费消息，按 task_id 拉取 records
- 调 asset-rpc 查账面
- 判定 diff_status：相符/盘盈/盘亏（见 5.3）
- UPDATE inventory_record.diff_status
- metric `fams_inventory_diff_total{type}` ++

### P5.4 边界条件

| 场景 | 行为 |
| --- | --- |
| 任务未开始/已结束提交 | 42201 |
| 非任务 scope 内 asset_no | 40302 或 40401 |
| 空 batch | 400 参数错误 |
| archive 重复调用 | 幂等，已归档返回当前状态 |

### P5.5 测试

```bash
go test ./service/inventory/... -short -v
go test ./tests/integration/inventory/... -tags=integration -v
```

**必测**：

- 两用户抢同一 asset_no → 一个 success 一个 40901
- CAS 版本冲突 → 40901
- 归档后 Worker 判定盘盈/盘亏/相符正确

### P5.6 Git

```bash
git checkout -b feat/p5-inventory-service
git push -u origin feat/p5-inventory-service
# PR：[P5] Inventory service - draft lock archive comparison worker
```

---

## P6：数据统计与报表微服务（Report Service）

**分支**：`feat/p6-report-service`  
**依赖**：P3, P4, P5  
**目标**：只读副本查询 + Kafka 物化宽表 + 聚合 API。

### P6.1 数据链路（3.5）

| 来源 | 用途 |
| --- | --- |
| MySQL 只读副本 | 资产明细只读 |
| PostgreSQL 只读副本 | 工单/盘点只读 |
| Kafka 消费 | 写入 report 库聚合表 |

**report 库表（PG 或 MySQL 独立 schema）**：

- `rpt_asset_daily_snapshot`
- `rpt_workflow_summary`
- `rpt_inventory_diff_summary`

### P6.2 API 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/report/assets/by-dept` | 各学院资产数量 |
| GET | `/api/v1/report/assets/by-category` | 类型分布 |
| GET | `/api/v1/report/inventory/diff/:taskId` | 盘点差异统计 |
| POST | `/api/v1/report/export` | 异步导出，返回 jobId |

**约束**：

- 所有接口只读；禁止 INSERT/UPDATE/DELETE 到业务主库
- 导出任务走异步队列，避免 >10s 阻塞

### P6.3 docker-compose 补充

在 P0 compose 中为 MySQL/PostgreSQL 增加 **read-only 副本** 或同实例只读账号 `fams_read`，连接串写入 `docker-compose-env.yml`：

```
MYSQL_READ_DSN=fams_read:xxx@tcp(mysql:3306)/fams_asset
POSTGRES_READ_DSN=postgres://fams_read:xxx@postgres:5432/fams_user
```

### P6.4 测试

```bash
go test ./service/report/... -short -v
go test ./tests/integration/report/... -tags=integration -v
```

### P6.5 Git

```bash
git checkout -b feat/p6-report-service
git push -u origin feat/p6-report-service
# PR：[P6] Report service - read replica aggregation export
```

---

## P7：全链路可观测性

**分支**：`feat/p7-observability`  
**依赖**：P0，各服务已可运行  
**目标**：Telemetry、Prometheus 指标、Grafana 大盘、业务 metric。

### P7.1 各服务 YAML 统一追加

```yaml
Telemetry:
  Name: <service-name>
  Endpoint: ${JAEGER_OTLP_ENDPOINT}
  Sampler: 1.0
  Batcher: otlpgrpc

Prometheus:
  Host: 0.0.0.0
  Port: <unique-port>
  Path: /metrics
```

**端口分配表**（写入 `docker-compose-env.yml` 和 `01-desgin` 运维附录）：

| 服务 | Metrics Port |
| --- | --- |
| user-api | 9101 |
| user-rpc | 9102 |
| asset-api | 9103 |
| asset-rpc | 9104 |
| workflow-api | 9105 |
| workflow-rpc | 9106 |
| inventory-api | 9107 |
| inventory-rpc | 9108 |
| report-api | 9109 |

### P7.2 业务指标埋点

| 指标 | 埋点位置 |
| --- | --- |
| `fams_workflow_pending{stage}` | workflow-rpc 定时 gauge 刷新 |
| `fams_inventory_lock_conflict_total` | inventory-api 抢锁失败 |
| `fams_asset_sync_lag_seconds` | asset kafka consumer |
| `fams_inventory_diff_total{type}` | comparison-worker |

### P7.3 Grafana Dashboard

路径：`deploy/grafana/dashboards/`

- `fams-overview.json`
- `fams-business.json`

数据源 provisioning：`deploy/grafana/provisioning/datasources/datasources.yml`

### P7.4 测试

```bash
# 启动 infra + 至少 user-api
make infra-up
# 手动或脚本：
curl http://localhost:9101/metrics | grep go_goroutines
# 访问 Grafana http://localhost:3000 确认 dashboard 有数据
# Jaeger UI http://localhost:16686 能看到 login trace
```

### P7.5 Git

```bash
git checkout -b feat/p7-observability
git push -u origin feat/p7-observability
# PR：[P7] Observability - jaeger prometheus grafana dashboards
```

---

## P8：Nginx 反向代理与网关聚合

**分支**：`feat/p8-nginx-gateway`  
**依赖**：P2–P6 API 可独立运行  
**目标**：统一入口、路由分发、限流。

### P8.1 Nginx 配置（`deploy/nginx/nginx.conf`）

| 路径前缀 | 上游 |
| --- | --- |
| `/api/v1/user/` | user-api:8888 |
| `/api/v1/asset/` | asset-api:8889 |
| `/api/v1/workflow/` | workflow-api:8890 |
| `/api/v1/inventory/` | inventory-api:8891 |
| `/api/v1/report/` | report-api:8892 |

**安全**：

- `limit_req_zone` 登录接口 10r/m/IP
- 客户端最大 body 20m（盘点批量提交）

### P8.2 docker-compose 追加 nginx service

挂载 `deploy/nginx/nginx.conf`，暴露 `80:80`。

### P8.3 测试

```bash
# 通过 nginx 登录
curl -X POST http://localhost/api/v1/user/login -d '{"username":"admin","password":"Test@123456"}'
# 经 nginx 创建资产申请全链路
```

### P8.4 Git

```bash
git checkout -b feat/p8-nginx-gateway
git push -u origin feat/p8-nginx-gateway
# PR：[P8] Nginx gateway routing rate limit
```

---

## P9：端到端集成与 CI

**分支**：`feat/p9-e2e-integration`  
**依赖**：全部 Phase  
**目标**：E2E 测试套件 + GitHub Actions（或等效 CI）。

### P9.1 E2E 测试场景（`tests/e2e/`）

| 场景 ID | 描述 |
| --- | --- |
| E2E-01 | 登录 → 创建资产 → 查询列表 |
| E2E-02 | 领用申请 → 院审 → 校审 → 台账 status=2 |
| E2E-03 | 盘点任务 → 双人提交冲突 → 归档 → 差异统计 |
| E2E-04 | 禁用用户 → 已有 token 访问 → 40102 |
| E2E-05 | 报表 API 返回与 seed 一致聚合数字 |

### P9.2 CI 流水线（`.github/workflows/backend-ci.yml`）

```yaml
# 伪代码结构
jobs:
  test:
    steps:
      - checkout
      - setup-go
      - docker compose up -d
      - go test ./pkg/... -short
      - go test ./service/... -short
      - go test ./tests/integration/... -tags=integration
      - go test ./tests/e2e/... -tags=e2e
      - docker compose down
```

### P9.3 验收

- [ ] CI 在干净环境 green
- [ ] E2E-01 ~ E2E-05 全部通过
- [ ] README（`assets-db/backend/README.md`）含本地启动步骤

### P9.4 Git

```bash
git checkout -b feat/p9-e2e-integration
git push -u origin feat/p9-e2e-integration
# PR：[P9] E2E tests and CI pipeline
```

---

## 附录 A：Phase 依赖关系图

```mermaid
flowchart TD
  P0[P0 Infra Docker] --> P1[P1 Shared pkg]
  P0 --> P7[P7 Observability]
  P1 --> P2[P2 User]
  P1 --> P3[P3 Asset]
  P2 --> P4[P4 Workflow]
  P3 --> P4
  P3 --> P5[P5 Inventory]
  P4 --> P5
  P3 --> P6[P6 Report]
  P4 --> P6
  P5 --> P6
  P2 --> P8[P8 Nginx]
  P3 --> P8
  P4 --> P8
  P5 --> P8
  P6 --> P8
  P8 --> P9[P9 E2E CI]
  P7 --> P9
```

## 附录 B：AI Agent 执行注意事项

1. **不要一次完成多个 Phase**：严格按依赖顺序，合并后再开下一分支。
2. **不要省略 `-tags=integration`**：集成测试是 Phase 验收门槛。
3. **遇到设计缺口**：先更新 `01-desgin.md` 相应章节，再实现代码，并在 PR 中注明「设计同步更新」。
4. **错误码冲突**：新增前 grep `pkg/errx/codes.go`，避免重复 code。
5. **MongoDB/Redis/Kafka 不可用时**：业务 API 返回 50301，并打 structured log（含 trace_id）。
6. **密码与密钥**：仅使用 `docker-compose-env.example.yml` 中的占位值，禁止提交真实凭据。

## 附录 C：建议执行时间线（参考）

| Phase | 预估工时 |
| --- | --- |
| P0 | 1–2 天 |
| P1 | 1–2 天 |
| P2 | 2–3 天 |
| P3 | 2–3 天 |
| P4 | 3–4 天 |
| P5 | 3–4 天 |
| P6 | 2 天 |
| P7 | 1–2 天 |
| P8 | 1 天 |
| P9 | 2 天 |

**合计**：约 18–25 个工作日（单人全职；AI Agent 并行需谨慎处理 Phase 依赖）。

---

*文档版本：v1.0 | 与 `01-desgin.md` 同步 | 最后更新：2026-07-06*
