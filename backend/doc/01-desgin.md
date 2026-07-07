# 基于多人协同的高校固定资产管理系统（FAMS）设计与微服务架构方案报告

## 一、 系统总体设计

### 1.1 总体架构拓扑

系统采用前后端分离的分布式微服务架构，后端基于 `go-zero` 架构搭建，并在最前端引入高性能组件进行流量调度。

* **反向代理层（Nginx）**：作为整个系统后端集群的唯一入口，Nginx 负责处理 SSL/TLS 卸载、静态资源反向代理、全局负载均衡，以及基础的安全防护（如 IP 限流、防恶意爬虫），将动态接口请求精确转发至下游的微服务网关。
* **网关接入层（API Gateway）**：利用 `go-zero` 内置的高性能路由与 JWT 鉴权中间件，统一承接来自 Nginx 转发的 HTTP 请求，执行统一的权限拦截、身份解析与流量控制，并将请求分发至下游 gRPC 微服务集群。
* **微服务业务层（RPC Services）**：各微服务独立部署，服务间通过高性能 gRPC 协议进行内部通信。使用 `etcd` 作为服务注册与发现中心，动态管理服务节点拓扑。
* **多租户/多层级隔离设计**：在 RPC 拦截器中解析网关透传的身份凭证，动态对底层数据库连接及 SQL/NoSQL 语句执行维度注入，实现全系统非功能性需求中的安全性与权限隔离。
* **可观测性基础设施层（Observability）**：旁路部署 Jaeger（分布式链路追踪）、Prometheus（指标采集与时序存储）与 Grafana（可视化与统一告警），对网关、微服务、消息队列及底层存储进行全链路、无侵入的运行时观测（详见第六章）。

**部署弹性说明**：本架构的完整形态（5 个微服务 + 多种异构存储 + Kafka/etcd + 可观测性组件）面向架构完整性与横向扩展能力设计。考虑到高校资产管理的真实并发规模有限（盘点高峰约数十人同时在线），系统各服务在代码层面保持微服务边界，但支持**按需降级合并部署**：小规模场景下可将全部 api/rpc 进程以单机 docker-compose 编排、各数据库单实例运行、Jaeger 采用 all-in-one 形态，待规模增长后再平滑拆分扩容，避免过度设计带来的运维负担。

### 1.2 协同设计与非功能性需求规范

根据总体设计要求，系统核心设计必须满足以下规范：

1. **多角色协同分工**：校级管理员、学院管理员、普通师生权限分离、各司其职。
2. **多级流程协同审批**：资产领用、归还、报修、报废等业务需多人逐级审核流转。
3. **多人联合协同盘点**：多名盘点人员同时在线录入数据、由系统引擎自动核对资产差异，共同完成盘点。
4. **全院数据协同共享与隔离**：同学院人员可共享查看本部门资产台账，非同部门数据相互隔离。
5. **审计留痕**：所有操作必须实名留痕、可追溯，精准记录操作人员与操作时间。
6. **非功能性指标**：系统需具备高协同性、严格的安全性控制、简洁清晰的可用性以及规范易维护的代码结构。
7. **全链路可观测性**：系统需具备完善的运行时观测能力，任意一次跨微服务请求可通过 Jaeger 完整还原调用链路，核心服务与中间件的性能指标由 Prometheus 持续采集，并经 Grafana 实现可视化监控大盘与阈值告警。

---

## 二、 组织架构、用户模型与权限可见性隔离

为满足组织架构独立性与细粒度的权限可见性要求，系统对用户与组织架构进行了物理表及逻辑层面的彻底分离。

### 2.1 权限与可见性控制矩阵

系统基于 RBAC 模型的变种（结合级别与组织路径），划分出以下三级角色的操作与数据边界：

* **校级管理员（级别 1）**：
* **操作权限**：具备全盘资产终审、报废最终核销、全校盘点任务发布与系统全局参数配置权限。
* **数据可见性**：拥有全局穿透视野，可无视组织机构代码限制，查询全校所有学院及实验室的资产底账、流转工单及审计日志。


* **学院资产管理员（级别 2）**：
* **操作权限**：负责本学院范围内的资产新增录入、基础编辑、多级流转的院级初审、以及指派具体的协同盘点人员。
* **数据可见性**：严格受限于所属组织节点，仅能维护和查询本学院及其下属二级实验室的资产数据。系统通过数据隔离中间件，先基于 `sys_department.path` 物化路径以前缀匹配解析出该管理员所辖的**整棵组织子树节点集合**（含本学院及全部下属实验室），再在 SQL 执行前自动组装 `WHERE department_id IN (:dept_subtree_ids)`，确保等值过滤不会漏掉下属实验室数据。


* **普通师生用户（级别 3）**：
* **操作权限**：可在线提交资产领用、归还、故障报修工单及报废申请。
* **数据可见性**：默认仅可见个人名下绑定的资产单据与个人领用资产。对于“全院数据协同共享”需求，系统额外提供特定只读 API，允许师生跨组查看所属学院内标记为“共享/公用”状态的实验器材与设备。



---

## 三、 基于 go-zero 的微服务划分与功能职责

结合业务边界内聚度、数据隔离级别及高并发协同场景，将后端划分为 5 个核心微服务：

### 3.1 用户与权限微服务（User Service）

* **功能职责**：负责系统用户登录、个人信息管理；维护独立的树状组织架构（学院/部门/实验室）；管理三级角色的级别设定与动态鉴权。
* **用户创建策略**：**不开放自助注册**。新用户仅由校级管理员（role=1）或学院管理员（role=2，仅限本学院子树）通过管理 API 创建；详见 `03-api-contract.md` §2.6。
* **架构设计**：
* `user-api`：暴露对外的登录及鉴权接口，校验成功后发放内含用户唯一标识（UID）、角色级别（RoleLevel）及所属组织ID（DeptID）的强加密 JWT Token。
* `user-rpc`：向其他兄弟微服务提供底层的分布式用户/组织架构元数据查询接口。



### 3.2 固定资产台账微服务（Asset Service）

* **功能职责**：管理资产全生命周期底账，支持资产新增录入、编辑、逻辑删除及细粒度分类管理（设备、家具、实验器材等）；记录资产编号、购置时间、价值、存放地点、归属部门及当前实际使用人；提供高性能的多维台账查询、多条件筛选与数据导出功能。
* **架构设计**：
* `asset-api`：供管理员执行资产增删改查及报表导出。
* `asset-rpc`：提供原子化的资产状态变更为核心方法（如：变更为在库、领用中、维修中、已报废），仅供系统内部的工作流审批流及盘点流异步触发调用。



### 3.3 工作流审批微服务（Workflow Service）

* **功能职责**：承载核心业务的流转协同审批，涵盖资产领用、归还、故障报修工单流转、资产报废两级审核等全流程。
* **架构设计**：
* `workflow-api`：接收师生提交的申请、管理员的审核指令（同意/驳回）。**v1 不实现转办**（见 §7.11）。
* `workflow-rpc`：推进流转状态机，负责校验多级审批人权限（学院初审 $\rightarrow$ 校级复审），并在节点终审通过时，负责向审计链路写入不可篡改的流水痕迹。四种工单类型的前置校验与事件映射详见 `04-workflow-rules.md`。



### 3.4 多人协同盘点微服务（Inventory Service）

* **功能职责**：处理高并发的多人在线联合盘点业务。支持校级/院级管理员发布特定范围与时间的盘点任务；承接多名工作人员对同区域前端电子表格的并发录入；利用高效比对引擎对账面原始数据与实际录入数据进行自动化精确校验，快速计算盘盈、盘亏差异并生成结构化分析明细。
* **前端表格组件选型说明**：Luckysheet 开源项目已停止维护，官方团队已转向后继项目 **Univer**，本系统优先选用 Univer（或对 Luckysheet 稳定版本做锁定封装）作为协同表格组件。同时明确协同模式的取舍：系统采用**“批量提交 + 提交时冲突检测”**模式而非 WebSocket 实时协同编辑——各盘点人员独立填报、批量上报，冲突在提交时由后端分布式锁检出并逐行提示；该模式实现成本低、链路简单，代价是无法实时看到他人正在编辑的单元格，对于按区域分工的盘点场景完全够用。
* **架构设计**：
* `inventory-api`：对接前端协同表格的数据批量提交，处理并发写入冲突。
* `inventory-rpc`：管理盘点任务生命周期，控制自动化比对任务队列调度。



### 3.5 数据统计与报表微服务（Report Service）

* **功能职责**：提供各学院资产数量统计、资产类型分布统计、盘点差异统计等高维度聚合报表；支持全院协同共享视角的报表在线预览与数据异步导出。
* **架构设计**：
* `report-api`：独立承接大范围只读聚合查询，防止复杂的报表统计 SQL 拖垮在线核心业务数据库。
* **报表数据链路**：为遵循微服务“数据私有”原则，`report-api` **不直连**兄弟服务的业务主库，其数据来源分为两级——① 各业务数据库（MySQL/PostgreSQL）的**只读副本（Read Replica）**，承接实时性要求高的明细查询；② Report Service 自身订阅 Kafka 领域事件（`fams-asset-lifecycle-events` 等），将资产状态、审批结论、盘点差异增量物化为独立报表库中的聚合宽表，承接高维度统计分析，实现与在线核心业务的彻底读写隔离。



---

## 四、 底层数据库混合选型方案（Polyglot Persistence）

系统针对微服务各自的业务特性，实施混合存储设计，确保强一致性、高并发性以及非结构化扩展能力的平衡。

### 4.1 混合存储矩阵规划

* **PostgreSQL (16.0)**：应用于**用户与权限微服务**、**工作流审批微服务**、**协同盘点微服务**。
* *选型理由*：高校组织架构属于典型的多级树状结构，PostgreSQL 完美的递归查询能力（`WITH RECURSIVE` 语法）能够通过单条 SQL 高效获取整条组织链或下属所有二级单位，极大精简业务代码。工作流与盘点明细属于核心审计资产，依赖 PostgreSQL 严苛的事务一致性与 MVCC 并发控制。


* **MySQL (8.0)**：应用于**固定资产台账微服务**。
* *选型理由*：资产底账具有绝对结构化、类财务审计的特征。选用 MySQL 配合 InnoDB 存储引擎，利用其成熟的 B+ Tree 索引为资产编号（`asset_no`）建立强唯一性约束，通过标准的行级锁机制保障台账基础数据的原子性与稳定性。


* **MongoDB (7.0)**：应用于**协同盘点微服务（临时草稿）**。
* *选型理由*：多名盘点人员在前端多人协同表格（Univer）中高频录入时，会产生大量临时的、未定型的行列单元格数据及多维备注。此类数据格式多变且写入密集，MongoDB 的 Document 模型支持 Schema-less 动态扩展，能够承载此类超高并发的协同暂存数据。


* **Redis (7.2)**：用于存储全系统的分布式用户 Session Token、JWT 撤销黑名单、协同流转状态缓存以及多人盘点时针对单台资产的分布式锁，防止并发覆盖。

**开发环境数据库命名约定**（单实例、逻辑隔离，生产可拆库）：

| 实例 | 库名 | 归属服务 |
| --- | --- | --- |
| PostgreSQL | `fams_core` | User / Workflow / Inventory 全部 PG 表 |
| PostgreSQL | `fams_report` | Report 物化宽表、导出任务 |
| MySQL | `fams_asset` | Asset 台账与事件去重 |
| MongoDB | `fams_inventory` | 盘点草稿 |

### 4.2 核心数据表/集合设计（SQL & NoSQL 模式）

#### 4.2.1 组织架构表（`sys_department` - 存储于 PostgreSQL）

用于高效表达高校多层级院系与实验室的关联拓扑。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 组织节点唯一 ID |
| parent_id | bigint | Not Null | 父级组织 ID（顶层学校节点为 0） |
| dept_name | varchar(100) | Not Null | 部门/学院/实验室名称 |
| dept_code | varchar(30) | Unique, Not Null | 组织机构代码 |
| path | varchar(255) | Not Null | 物化路径（如 `/1/15/103/`），以前缀匹配高效获取整棵组织子树，避免高频递归查询 |
| sort_order | int | Not Null, Default 0 | 排序权重 |

#### 4.2.2 用户表（`sys_user` - 存储于 PostgreSQL）

解耦后的纯净用户表，仅保留核心认证属性。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 用户自增唯一 ID |
| username | varchar(50) | Unique, Not Null | 登录账号/工号 |
| password_hash | varchar(255) | Not Null | 加盐哈希后的密码密文 |
| real_name | varchar(50) | Not Null | 真实姓名（实名审计核心） |
| role_level | smallint | Not Null | 级别：1-校级管理员, 2-学院管理员, 3-普通师生 |
| department_id | bigint | Not Null | 所属组织架构节点外键 ID |
| status | smallint | Not Null | 状态：1-启用, 0-禁用 |

#### 4.2.3 固定资产台账表（`asset_ledger` - 存储于 MySQL）

承载严谨的高校资产核心结构化底账。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 资产底账唯一自增 ID |
| asset_no | varchar(50) | Unique, Not Null | 资产编号（财务追溯唯一标识） |
| name | varchar(100) | Not Null | 资产名称 |
| category | varchar(50) | Not Null | 细分类别（设备、家具、实验器材等） |
| price | decimal(10,2) | Not Null | 购置资产价值 |
| purchase_time | datetime | Not Null | 购置时间 |
| location | varchar(100) | Not Null | 物理存放地点 |
| department_id | bigint | Not Null | 归属学院/部门 ID（数据隔离维度） |
| user_id | bigint | Null | 当前领用人/实际使用人用户 ID |
| is_shared | tinyint | Not Null, Default 0 | 是否学院内共享可见：0-否, 1-是（供师生只读 API 过滤） |
| status | tinyint | Not Null | 业务状态：1-在库, 2-领用中, 3-维修中, 4-已报废 |
| deleted_at | datetime | Null | 逻辑删除时间；非 NULL 表示已删除，列表默认过滤 |

> **逻辑删除**：DELETE 接口仅写 `deleted_at=now()`，**不修改** `status` 枚举值；查询默认 `WHERE deleted_at IS NULL`。

#### 4.2.4 审批申请主表（`workflow_request` - 存储于 PostgreSQL）

记录多角色多级流程流转状态。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 审批工单唯一自增 ID |
| asset_id | bigint | Not Null | 关联资产 ID |
| requester_id | bigint | Not Null | 申请人用户 ID |
| department_id | bigint | Not Null | 资产归属部门 ID（建单时自资产台账冗余固化，支撑院级待办隔离过滤，避免跨服务 JOIN） |
| type | smallint | Not Null | 业务类型：1-领用, 2-归还, 3-报修, 4-报废 |
| current_stage | smallint | Not Null | 当前审批阶段：1-院级初审, 2-校级复审, 3-归档结束 |
| status | smallint | Not Null | 流程状态：1-审批中, 2-审批通过, 3-已被驳回 |
| reason | varchar(255) | Null | 申请缘由说明或故障描述 |
| created_at | timestamptz | Not Null | 工单创建时间 |
| updated_at | timestamptz | Not Null | 最后状态变更时间 |

> **防重复申请约束**：对该表建立部分唯一索引 `CREATE UNIQUE INDEX uk_asset_open_request ON workflow_request (asset_id) WHERE status = 1;`，从数据库层面保证同一资产在任意时刻至多存在一张“审批中”的工单。
>
> **驳回后再申请**：驳回后 `status=3`，不在部分唯一索引范围内，**允许**用户立即对同一资产重新提交新工单。

#### 4.2.5 审批留痕流水表（`workflow_log` - 存储于 PostgreSQL）

完全满足协同工作实名留痕审计要求的关键数据表。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 审计流水唯一自增 ID |
| request_id | bigint | Not Null | 关联的审批申请单 ID |
| operator_id | bigint | Not Null | 实际执行审批操作的人员 ID |
| action | varchar(50) | Not Null | 动作快照（如：提交申请、院级初审同意、校级复审通过、驳回） |
| comment | varchar(255) | Null | 具体审批签署意见 |
| operate_time | timestamptz | Not Null | 操作动作发生的精确时间 |

#### 4.2.6 多人协同盘点草稿集合（`inventory_draft` - 存储于 MongoDB）

应对前端协同表格多人高并发录入的高性能无模式集合。

```json
{
  "_id": "ObjectId",
  "task_id": 20260701,
  "asset_no": "EQUIP-2026-0091",
  "operator_id": 10023,
  "modified_cells": {
    "actual_location": "一号实验楼302",
    "temp_notes": "多功能机械臂底座略有磨损",
    "photo_urls": ["https://oss/fams/abc.png"]
  },
  "updated_at": "ISODate"
}

```

#### 4.2.7 盘点明细比对结果表（`inventory_record` - 存储于 PostgreSQL）

盘点终结阶段固化生成的结构化比对结果。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 盘点明细唯一自增 ID |
| task_id | bigint | Not Null | 关联的盘点任务 ID |
| asset_id | bigint | Null | 关联的账面资产 ID。**盘盈记录（账上无此资产）该字段为 NULL**，仅账面资产明细必填 |
| found_asset_desc | varchar(255) | Null | 盘盈资产的现场描述快照（名称/型号/地点等），仅盘盈记录填写 |
| operator_id | bigint | Null | 最终执行实名核对录入的盘点人员 ID |
| is_scanned | smallint | Not Null, Default 0 | 状态快照：0-未盘点, 1-已盘点 |
| actual_location | varchar(100) | Null | 现场实际清查发现的物理地点 |
| diff_status | smallint | Not Null, Default 0 | 账实比对结果：0-未比对, 1-账实相符, 2-盘盈, 3-盘亏 |

> **唯一性约束**：对账面资产明细建立部分唯一索引 `CREATE UNIQUE INDEX uk_task_asset ON inventory_record (task_id, asset_id) WHERE asset_id IS NOT NULL;`，防止同一任务对同一资产重复归档。

#### 4.2.8 盘点任务主表（`inventory_task` - 存储于 PostgreSQL）

定义每一次协同盘点的范围、周期与生命周期状态，被 `inventory_draft` 与 `inventory_record` 的 `task_id` 引用。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 盘点任务唯一自增 ID |
| task_name | varchar(100) | Not Null | 任务名称（如：2026 年度信息学院实验室盘点） |
| scope_dept_id | bigint | Not Null | 盘点范围对应的组织节点 ID（含其整棵子树） |
| creator_id | bigint | Not Null | 任务发布人（校级/院级管理员）用户 ID |
| start_time | timestamptz | Not Null | 盘点窗口开始时间 |
| end_time | timestamptz | Not Null | 盘点窗口截止时间 |
| status | smallint | Not Null | 任务状态：1-进行中, 2-已归档比对中, 3-已完成 |
| created_at | timestamptz | Not Null | 任务创建时间 |

#### 4.2.9 事务发件箱表（`workflow_outbox` - 存储于 PostgreSQL）

采用 **Transactional Outbox 模式**解决“PostgreSQL 事务提交成功、Kafka 事件发送失败”导致的静默数据不一致问题：领域事件与业务数据在**同一本地事务**内落库，由后台投递进程异步可靠外发（详见 5.2 节）。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 发件箱记录唯一自增 ID |
| event_type | varchar(50) | Not Null | 领域事件类型（如 ASSET_USE_APPROVED） |
| partition_key | varchar(50) | Not Null | Kafka 分区键（取 `asset_id`，保证同一资产事件严格有序） |
| payload | jsonb | Not Null | 完整事件报文 |
| status | smallint | Not Null, Default 0 | 投递状态：0-待投递, 1-已投递, 2-投递失败（死信，需人工介入） |
| retry_count | int | Not Null, Default 0 | 已重试次数（上限 10，超限置 status=2） |
| created_at | timestamptz | Not Null | 事件产生时间 |
| sent_at | timestamptz | Null | 实际投递成功时间 |

#### 4.2.10 盘点任务指派表（`inventory_task_assignee` - 存储于 PostgreSQL）

记录院级管理员指派的具体协同盘点人员，仅被指派者可在该任务下提交草稿。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 自增 ID |
| task_id | bigint | Not Null | 关联盘点任务 ID |
| user_id | bigint | Not Null | 被指派盘点人员用户 ID |
| assigned_by | bigint | Not Null | 指派人（管理员）用户 ID |
| assigned_at | timestamptz | Not Null | 指派时间 |

> **唯一性约束**：`CREATE UNIQUE INDEX uk_task_user ON inventory_task_assignee (task_id, user_id);`

#### 4.2.11 报表聚合宽表（存储于 PostgreSQL 独立库 `fams_report`）

Report Service 专用库，由 Kafka 事件消费者增量写入，禁止业务服务直连写入。

**`rpt_asset_daily_snapshot`**（按日按部门快照）：

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | PK |
| snapshot_date | date | 快照日期 |
| department_id | bigint | 部门 ID |
| total_count | int | 资产总数 |
| in_stock_count | int | 在库数 |
| in_use_count | int | 领用中数 |
| repair_count | int | 维修中数 |
| scrap_count | int | 已报废数 |
| total_value | decimal(14,2) | 资产总价值 |

**`rpt_workflow_summary`**（按日按类型汇总）：

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | PK |
| summary_date | date | 汇总日期 |
| request_type | smallint | 1-领用, 2-归还, 3-报修, 4-报废 |
| approved_count | int | 当日审批通过数 |
| rejected_count | int | 当日驳回数 |
| pending_count | int | 当前积压数（Gauge 同步） |

**`rpt_inventory_diff_summary`**（按任务汇总）：

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | PK |
| task_id | bigint | 盘点任务 ID |
| match_count | int | 账实相符数 |
| surplus_count | int | 盘盈数 |
| loss_count | int | 盘亏数 |
| updated_at | timestamptz | 最后更新时间 |

#### 4.2.12 报表导出任务表（`rpt_export_job` - 存储于 PostgreSQL `fams_report`）

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 导出任务 ID（即 API 返回的 jobId） |
| creator_id | bigint | Not Null | 发起人用户 ID |
| export_type | varchar(50) | Not Null | 导出类型：asset_list / inventory_diff / workflow_log |
| params | jsonb | Not Null | 导出参数（筛选条件快照） |
| status | smallint | Not Null | 0-排队, 1-处理中, 2-已完成, 3-失败 |
| file_path | varchar(255) | Null | 生成文件相对路径（CSV） |
| error_message | varchar(255) | Null | 失败原因 |
| created_at | timestamptz | Not Null | 创建时间 |
| finished_at | timestamptz | Null | 完成时间 |

---

## 五、 中间件配置与核心业务协同详细流程

系统通过 Nginx 进行前置请求拦截与分发，并合理调配高性能中间件 Redis 与分布式消息队列 Kafka，实现各个微服务间解耦与高吞吐协同。

### 5.1 用户登录与凭证鉴权核心流程

1. **涉及组件**：Nginx、`user-api`、`user-rpc`，PostgreSQL（用户权限库），Redis（凭证缓存）。
2. **详细流程拓扑**：
* 客户端发起登录请求，请求首先到达 **Nginx 反向代理层**，Nginx 校验请求合法性后，将其转发给后端的 `user-api`（`/v1/user/login`）。
* `user-api` 将数据打包后通过 gRPC 协议传输至后台 `user-rpc` 微服务的 `FindUser` 方法。
* `user-rpc` 连接 PostgreSQL 数据库，快速检索 `sys_user` 关联 `sys_department`。比对密码加盐哈希值，确认账户未被禁用。
* 验证无误后，`user-rpc` 提取出该用户的 `id`（UID）、`role_level` 及 `department_id`（DeptID），利用 `go-zero` 框架内置的 JWT 生成组件，将这三个核心维度强制注入 JWT Claims 载荷，签发**短时效 Access Token（有效期 2 小时）**，并同时签发有效期 24 小时的 Refresh Token 用于无感续期，压缩权限信息在 Claims 中的“漂移窗口”。
* 同时，`user-rpc` 同步将此凭据写入 **Redis** 中（Key 为 `fams:auth:token:${uid}`，Value 为生成的 Token 字符串，过期时间与 Access Token 对齐），用以支持单点登录（SSO）控制与在线状态管理。
* `user-api` 最终将 Token 返回客户端。后续客户端发起的任何请求都会经过 Nginx，由网关拦截器完成两级校验：**① 无状态校验** JWT 签名与有效期，解析出该用户的级别与部门 ID；**② 有状态复核**——以 O(1) 复杂度查询 Redis 中的 JWT 撤销黑名单（`fams:auth:blacklist:${jti}`），命中则立即拒绝。
* **权限即时失效机制**：当发生用户禁用、调岗、角色降级或管理员强制下线时，`user-rpc` 在变更落库的同时将该用户存量 Token 的 `jti` 写入 Redis 黑名单（TTL 与 Token 剩余有效期对齐），确保敏感权限变更即刻生效，而非等待 Token 自然过期。



### 5.2 资产领用/归还多级审批与状态同步流程（Kafka 驱动解耦）

1. **涉及组件**：Nginx、`workflow-api`、`workflow-rpc`、`asset-rpc`，PostgreSQL（工作流库，含 Outbox 发件箱），MySQL（资产台账库），Kafka（分布式事件总线）。
2. **详细流程拓扑**：
* **工单创建（含资产状态前置校验）**：普通师生在前端发起资产领用，请求经 Nginx 转发至 `workflow-api`，解析 JWT 拦截器注入的身份，发起 `workflow-rpc`。建单前 `workflow-rpc` 先通过 gRPC 调用 `asset-rpc` 校验目标资产当前状态必须为 `1 (在库)`，并取回该资产的归属部门 ID；校验通过后向 PostgreSQL 写入工作流主表 `workflow_request`（同时将归属部门冗余固化至 `department_id` 字段），初始阶段标记为 `1 (院级初审中)`。若两名用户并发对同一资产发起领用，后到者会被 4.2.4 节定义的部分唯一索引 `uk_asset_open_request` 直接拦截，从数据库层面杜绝重复领用。
* **两级流转控制**：
* 学院管理员查询待办单，`workflow-rpc` 基于数据隔离中间件解析出该管理员的组织子树集合，从 PostgreSQL 过滤出 `department_id IN (:dept_subtree_ids)` 的记录（确保隔离，禁止越权操作数据），管理员初审通过，审批阶段推移至 `2 (校级复审中)`。
* 校级管理员全局可见该待办单，审查完毕后在前端触发终审通过。


* **实名审计与本地事务（Outbox 同事务落库）**：`workflow-rpc` 在终审时先经 `asset-rpc` 复核资产状态仍为可领用，随后在 PostgreSQL 中开启本地事务，原子化完成三件事：① 将工单 `status` 固化变更为 `2 (审批通过)`、`current_stage` 推进至 `3 (归档结束)`；② 向 `workflow_log` 写入一条不可篡改的实名操作流水；③ 向 `workflow_outbox` 发件箱表写入待投递的领域事件。三者同事务提交、同事务回滚，从根本上消除“业务落库成功而事件丢失”的双写不一致风险。
* **跨库解耦事件可靠投递**：后台独立的 **Outbox Dispatcher 投递进程**轮询 `workflow_outbox` 中 `status = 0 (待投递)` 的记录，以 `asset_id` 作为**分区键**（保证同一资产的事件在 Kafka 分区内严格有序）将事件发送至名为 `fams-asset-lifecycle-events` 的 **Kafka Topic**，发送获得 Broker ACK 后回写 `status = 1 (已投递)`；发送失败则累加 `retry_count` 并按指数退避重试。事件报文如下：
```json
{
  "event_type": "ASSET_USE_APPROVED",
  "request_id": 88213,
  "asset_id": 40912,
  "target_status": 2,
  "assigned_user_id": 20261102,
  "operator_id": 10001,
  "timestamp": 1783353600
}

```


* **资产底账幂等消费同步**：`asset-rpc` 微服务下属的 Kafka 消费者集群监听到该消息。由于 Outbox 重投与 Kafka 的 at-least-once 语义均可能造成消息重复，消费者以 `request_id` 作为幂等键（本地去重表判重），重复消息直接 ACK 跳过；首次消费则解析出资产主键，操作底层物理 **MySQL** 数据库，将 `asset_ledger` 表中对应资产的 `status` 变更为 `2 (领用中)`，并将 `user_id` 字段更新为事件中指定的领用人 ID。
* **兜底对账**：部署定时对账任务，周期性核对“已归档通过的工单”与 MySQL 台账实际状态的一致性，对超时未同步的记录自动触发事件重放并告警（对应 6.3.2 节 `fams_asset_sync_lag_seconds` 指标），形成最终一致性的最后一道防线。



### 5.3 多人联合协同盘点流程（Redis 分布式锁与 MongoDB 高并发暂存）

1. **涉及组件**：Nginx、`inventory-api`、`inventory-rpc`、`asset-rpc`，Redis（冲突控制锁），MongoDB（非结构化高频草稿），PostgreSQL（盘点归档库），Kafka（异步比对队列）。
2. **详细流程拓扑**：
* **多人并发协同无冲突写入**：当多名学院盘点人员同时在线盘点录入，通过前端协同表格组件批量提交数据时，请求通过 Nginx 均衡负载打入 `inventory-api`。
* `inventory-api` 拦截到请求后，针对更新报文里的每一台资产编号（`asset_no`），首先向全局 **Redis** 集群尝试申请分布式排他锁（锁 TTL 设置为 30 秒，须显著大于单次“落库 MongoDB”的最大预期耗时，避免持锁期间锁提前过期）：
```
SET fams:lock:inventory:${asset_no} ${operator_id} NX PX 30000

```


* **逐条冲突语义与部分成功**：批量提交按资产逐条处理——某台资产抢锁失败不阻塞整批，接口返回**逐条明细结果**（成功列表 + 冲突列表），前端据此仅将冲突行标红提示重试，避免整批提交“部分成功部分失败”造成表格状态与后端不一致。
* **锁的安全释放**：草稿落库完成后立即通过 **Lua 脚本原子释放锁**——先校验锁 Value 仍等于自己的 `operator_id` 再执行 DEL，防止因执行超时导致锁已过期、误删他人新持有的锁。
* **二级乐观校验兜底**：考虑到极端情况下锁 TTL 过期后仍可能发生并发写入，MongoDB 草稿更新时附带 `updated_at` 版本比对（CAS 语义）：仅当文档当前版本与提交方读取时的版本一致才允许覆盖，否则同样返回冲突。分布式锁负责将冲突概率降至最低，乐观校验保证正确性下限。
* **非结构化草稿高速缓存**：成功抢占 Redis 锁的请求，其现场采集的非结构化动态数据全速写入 **MongoDB** 的 `inventory_draft` 集合中进行无模式高并发草稿暂存。
* **组长终结归档与规则比对**：盘点终结时，`inventory-rpc` 从 MongoDB 中聚合读出该任务所有的最终暂存草稿，以标准的 PostgreSQL 强事务批量固化写入 `inventory_record` 表，将 `is_scanned` 字段批量置为 `1 (已盘点)`。
* 明细固化入库后，`inventory-rpc` 产生一个比对信号，发送至 **Kafka** 队列 `fams-inventory-comparison-tasks` 中。
* 后台专用的自动化比对 Worker 进程消费此消息，通过 gRPC 协议调用 `asset-rpc` 提取出 **MySQL** 中该资产的原始账面数据（存放地点、归属部门等），在内存中通过确定的业务规则与 PostgreSQL 中现场采集的实际物理地点进行精准字段级对齐核对。
* 三类差异的判定规则：账面存在且现场核对一致者判定**账实相符**；账面存在但现场未清查到（`is_scanned = 0`）者判定**盘亏**；现场录入的资产编号在账面完全不存在者判定**盘盈**——Worker 为其生成 `asset_id` 为 NULL 的独立明细记录，并将现场采集的名称/型号/地点固化至 `found_asset_desc` 字段（对应 4.2.7 节表结构设计）。
* 判定完成后，Worker 将最终的差异状态码（`diff_status`）异步更新回 PostgreSQL 的 `inventory_record` 表中，高效完成多人协同盘点与自动化账实差异清查。

---

## 六、 全链路可观测性体系设计（Jaeger + Prometheus + Grafana）

微服务拆分后，一次业务请求往往横跨 Nginx、API 网关、多个 gRPC 微服务、Kafka 异步链路与多种异构存储，传统的单机日志排查方式已无法定位性能瓶颈与故障根因。为此，系统在业务集群旁路引入三大可观测性组件，构建“链路追踪（Tracing）+ 指标监控（Metrics）+ 可视化告警（Visualization & Alerting）”三位一体的观测体系。该体系对业务代码近乎零侵入，完全依托 `go-zero` 框架内置的 OpenTelemetry 与 Prometheus 支持实现。

### 6.1 可观测性总体架构拓扑

* **Jaeger（分布式链路追踪）**：各 `api`/`rpc` 服务通过 OpenTelemetry SDK 将 Span 上报至 Jaeger Collector，追踪数据落盘后由 Jaeger Query/UI 提供检索，用于还原任意一次请求的完整跨服务调用链。
* **Prometheus（指标采集与时序存储）**：以 Pull 模式周期性抓取各微服务及基础设施 Exporter 暴露的 `/metrics` 端点，持久化为时序数据，并内置 PromQL 聚合查询与告警规则评估能力。
* **Grafana（统一可视化与告警面板）**：同时接入 Prometheus 与 Jaeger 双数据源，为运维与开发人员提供分服务、分中间件的监控大盘，并承担阈值告警的统一出口。

三者与业务集群的关系为**旁路观测**：追踪上报为异步批量发送，指标抓取为外部拉取，均不阻塞核心业务请求路径，即使可观测性组件整体宕机也不影响资产业务的正常运转。

### 6.2 基于 Jaeger 的分布式链路追踪设计

#### 6.2.1 接入方式（go-zero 原生 Telemetry 配置）

`go-zero` 框架内置 OpenTelemetry 集成，各微服务在 YAML 配置文件中声明 `Telemetry` 节点即可完成 Trace 的生成、传播与上报。由于 OpenTelemetry Go SDK 已弃用旧版 Jaeger Thrift Exporter，本系统统一采用 **OTLP gRPC 协议**直连 Jaeger 的原生 OTLP 接收端口（4317）：

```yaml
Name: workflow-rpc
Telemetry:
  Name: workflow-rpc
  Endpoint: jaeger-collector:4317
  Sampler: 1.0        # 开发/测试环境全量采样；生产环境可降为 0.1 概率采样
  Batcher: otlpgrpc   # 弃用旧版 jaeger batcher，走 OTLP 标准协议
```

框架的 HTTP 中间件与 gRPC 拦截器会自动在服务边界注入/提取 W3C `traceparent` 上下文，实现 TraceID 在 `api → rpc → rpc` 同步调用链上的无感透传。需要说明的是：**Kafka 生产/消费侧的 Trace 上下文透传并非框架自动行为**，需在事件发布与消费的封装层手写少量埋点（生产者将 `traceparent` 写入消息 Header、消费者从 Header 恢复上下文），是全链路中唯一需要主动编码的部分。

#### 6.2.2 全链路追踪范围与关键 Span 设计

以 5.2 节“资产领用多级审批”为例，一次终审通过操作将形成如下完整 Trace 树：

* `workflow-api`（HTTP 入口 Span，记录路由、状态码与 JWT 解析出的操作人 UID）
* → `workflow-rpc.Approve`（gRPC Span，记录审批阶段推移）
* → PostgreSQL 事务 Span（记录 `workflow_request` 更新与 `workflow_log` 留痕写入耗时）
* → Kafka Producer Span（发送 `fams-asset-lifecycle-events` 事件，并将 TraceID 注入 Kafka 消息 Header）
* → `asset-rpc` 消费者 Span（通过消息 Header 还原上游 Trace 上下文，形成跨异步边界的完整链路）
* → MySQL 更新 Span（`asset_ledger` 状态变更耗时）

针对 Kafka 异步链路，生产者在消息 Header 中显式携带 `traceparent` 字段，消费者侧以 `Span Link` 方式关联上游上下文，从而保证“审批通过 → 台账异步同步”这类跨库解耦流程在 Jaeger UI 中仍可作为一条逻辑链路完整回放。

#### 6.2.3 追踪数据的典型排障场景

* **慢请求定位**：当报表导出接口响应变慢时，可在 Jaeger 中按 `report-api` + 耗时倒序检索，直观判断瓶颈位于聚合 SQL 还是 gRPC 网络。
* **异步链路丢单排查**：若资产审批通过后台账状态未同步，可通过工单对应的 TraceID 直接观察 Kafka 消费者 Span 是否存在、是否报错，替代人工翻查多台机器日志。
* **审计辅助**：Span Tag 中统一注入 `uid`、`dept_id` 业务维度标签，与第 2.1 节的实名审计要求形成技术侧互补。

### 6.3 基于 Prometheus 的指标监控设计

#### 6.3.1 指标暴露与采集拓扑

`go-zero` 内置 Prometheus 指标支持，各微服务在配置中开启独立的指标端口即可自动暴露请求级指标：

```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9101
  Path: /metrics
```

Prometheus Server 以 15s 为默认周期，对以下目标执行 Pull 抓取：

| 采集目标 | 暴露方式 | 核心观测指标 |
| --- | --- | --- |
| 5 个微服务的 api/rpc 进程 | go-zero 内置 `/metrics` | HTTP/gRPC 请求 QPS、时延分位（P50/P90/P99）、错误码分布 |
| Nginx | nginx-prometheus-exporter | 活跃连接数、每秒请求数、4xx/5xx 比例 |
| MySQL | mysqld_exporter | 慢查询数、InnoDB 行锁等待、连接池使用率 |
| PostgreSQL | postgres_exporter | 事务提交/回滚速率、长事务、表膨胀 |
| MongoDB | mongodb_exporter | 文档写入速率、WiredTiger 缓存命中率 |
| Redis | redis_exporter | 内存占用、Key 逐出数、分布式锁命令（SET NX）失败率 |
| Kafka | kafka_exporter | 各 Topic 消息堆积量（Consumer Lag）、生产/消费速率 |
| 宿主机/容器 | node_exporter / cAdvisor | CPU、内存、磁盘 IO、网络吞吐 |

#### 6.3.2 面向核心业务场景的定制指标

在框架默认指标之外，结合本系统的协同业务特征补充以下业务级自定义指标（通过 go-zero 提供的 `metric.NewGaugeVec` / `NewCounterVec` / `NewHistogramVec` 注册）：

* `fams_workflow_pending{stage}`：按审批阶段统计的当前积压工单数（Gauge 类型，瞬时水位语义，故命名不带 `_total` 后缀），用于观测“院级初审/校级复审”是否出现审批堆积。
* `fams_inventory_lock_conflict_total`：多人协同盘点时 Redis 分布式锁抢占失败次数，反映并发录入冲突热度。
* `fams_asset_sync_lag_seconds`：从审批终审通过（Kafka 事件产生）到 MySQL 台账状态落库的端到端同步延迟直方图，度量最终一致性时效。
* `fams_inventory_diff_total{type}`：按“盘盈/盘亏/相符”维度统计的比对结果计数，供报表与告警复用。

### 6.4 基于 Grafana 的可视化监控与统一告警

#### 6.4.1 数据源与大盘规划

Grafana 同时接入 **Prometheus**（指标）与 **Jaeger**（链路）两类数据源，并按职责划分以下监控大盘（Dashboard）：

1. **全局总览盘**：全系统 QPS、错误率、P99 时延、各微服务健康状态（基于 `up` 指标）一屏总览。
2. **微服务明细盘**：每个服务独立面板，含接口级时延分位、gRPC 调用成功率、Goroutine/GC 等 Go 运行时指标。
3. **中间件专项盘**：MySQL/PostgreSQL/MongoDB/Redis/Kafka 各自的容量、性能与堆积水位。
4. **业务协同盘**：审批积压趋势、盘点锁冲突热力、台账异步同步延迟等 6.3.2 节定制业务指标的可视化。

同时启用 Grafana 的 **Trace to Metrics / Metrics to Trace 联动**：在指标面板中发现某时段 P99 突刺后，可直接下钻跳转至 Jaeger 中该时段的慢 Trace 样本，实现“看见异常 → 定位根因”的闭环。

#### 6.4.2 告警规则设计

告警统一由 Grafana Alerting 管理，通知渠道对接邮件与 Webhook（可扩展至企业微信/钉钉）。核心告警规则如下：

| 告警项 | 触发条件（示例阈值） | 级别 |
| --- | --- | --- |
| 服务实例离线 | `up == 0` 持续 1 分钟 | P0 |
| 接口错误率突增 | 任一服务 5xx 比例 > 1% 持续 5 分钟 | P1 |
| 接口时延劣化 | 任一核心接口 P99 > 1s 持续 5 分钟 | P1 |
| Kafka 消费堆积 | `fams-asset-lifecycle-events` Consumer Lag > 1000 | P1 |
| 台账同步延迟 | `fams_asset_sync_lag_seconds` P99 > 30s | P2 |
| 审批工单积压 | 任一阶段积压工单数环比增长超 200% | P2 |
| 数据库慢查询 | MySQL 慢查询速率持续升高 / PG 出现超 60s 长事务 | P2 |

### 6.5 部署形态与资源隔离

* 可观测性三件套（Jaeger、Prometheus、Grafana）与各类 Exporter 统一以容器方式随 `docker-compose` / K8s 编排部署，与业务微服务共用同一内网但独立命名空间，避免资源争抢。
* Jaeger 开发环境采用 all-in-one 单体部署（内存存储）；生产环境拆分为 Collector + Query 两级，追踪数据落盘至 Elasticsearch 并设置 7 天 TTL。
* Prometheus 时序数据默认保留 15 天，Grafana 大盘与告警规则以 JSON Provisioning 文件纳入代码仓库版本化管理，保证监控配置与系统代码同步演进、可审计可回溯。

---

## 七、 工程实现规范（供开发任务书引用）

本节定义代码仓库结构、Git 工作流、统一错误码、API 约定、测试分层与 Docker 编排约定，作为 `02-plan.md` 各阶段任务的共同约束。

### 7.1 代码仓库目录结构

```
assets-db/backend/
├── deploy/                          # 部署与基础设施编排
│   ├── docker/
│   │   ├── docker-compose.yml       # 基础设施服务定义（DB/中间件/可观测性）
│   │   └── docker-compose-env.yml   # 环境变量（端口、密码、Topic 名等）
│   ├── nginx/                       # 反向代理配置
│   ├── prometheus/                  # 抓取配置与告警规则
│   ├── grafana/                     # Dashboard 与数据源 Provisioning
│   └── sql/                         # 各库初始化 DDL 与 seed 数据
│       ├── postgres/
│       └── mysql/
├── doc/                             # 设计文档与任务书
├── scripts/                         # 本地开发脚本（migrate、seed、healthcheck）
├── pkg/                             # 跨服务公共库
│   ├── errx/                        # 统一错误码与 HTTP/gRPC 错误映射
│   ├── middleware/                  # JWT 校验、黑名单、组织子树隔离
│   ├── dept/                        # path 物化路径解析与子树 ID 集合
│   ├── redislock/                   # 分布式锁（含 Lua 安全释放）
│   ├── outbox/                      # Transactional Outbox 投递器
│   └── kafka/                       # Kafka 生产者/消费者封装（含 traceparent）
├── service/
│   ├── user/    (api + rpc)
│   ├── asset/   (api + rpc)
│   ├── workflow/(api + rpc + outbox-dispatcher)
│   ├── inventory/(api + rpc + comparison-worker)
│   └── report/  (api + event-consumer)
└── go.mod                         # Go Workspace 或单模块根
```

### 7.2 Git 工作流约定

| 项 | 约定 |
| --- | --- |
| 主分支 | `main`（始终可部署、通过 CI） |
| 功能分支命名 | `feat/<phase>-<short-desc>`，如 `feat/p2-user-login` |
| 修复分支命名 | `fix/<issue>-<short-desc>` |
| 提交信息 | `<type>(<scope>): <subject>`，type 取 `feat`/`fix`/`test`/`chore`/`docs` |
| 合并方式 | 功能分支 → `main` 使用 **Squash Merge**，保留一条清晰 commit |
| PR 要求 | 必须通过本阶段定义的测试命令 + `golangci-lint run`（如已配置） |
| 推送 | 每完成一个可验收子任务至少 1 次 commit；阶段验收前 `git push -u origin <branch>` |

### 7.3 统一 API 与错误响应约定

**HTTP 路径前缀**：`/api/v1/<service>/...`（经 Nginx 转发至各 `*-api`）。

**统一成功响应**：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

**统一错误响应**：

```json
{
  "code": 40001,
  "message": "资产当前不可领用",
  "data": null
}
```

**核心错误码段（节选，完整矩阵见 `06-error-codes.md`）**：

| code | HTTP | 含义 | 典型触发场景 |
| --- | --- | --- | --- |
| 0 | 200 | 成功 | — |
| 40101 | 401 | 未登录或 Token 无效 | JWT 缺失/签名错误/过期 |
| 40102 | 401 | Token 已撤销 | Redis 黑名单命中 |
| 40301 | 403 | 无操作权限 | 角色级别不足 |
| 40302 | 403 | 数据越权 | 组织子树隔离拦截 |
| 40401 | 404 | 资源不存在 | 用户/资产/工单/任务不存在 |
| 40901 | 409 | 并发冲突 | 盘点锁抢占失败 / MongoDB CAS 冲突 |
| 40902 | 409 | 重复申请 | `uk_asset_open_request` 唯一索引冲突 |
| 42201 | 422 | 业务状态不允许 | 资产非在库却发起领用 |
| 50001 | 500 | 内部错误 | 未预期 panic/依赖不可用 |
| 50301 | 503 | 依赖暂不可用 | DB/Redis/Kafka 连接失败 |

**gRPC 错误**：各 RPC 使用 `status.Error(codes.X, errx.CodeString)`，API 层映射为上述 HTTP 响应，禁止向客户端泄露 SQL/堆栈。

### 7.4 测试分层约定

| 层级 | 目录约定 | 运行命令 | 必覆盖场景 |
| --- | --- | --- | --- |
| 单元测试 | `*_test.go` 与源码同包 | `go test ./... -short` | 纯逻辑：状态机、path 解析、幂等判重 |
| 集成测试 | `tests/integration/<service>/` | `go test ./tests/integration/... -tags=integration` | 依赖 docker-compose 起 DB/Redis/Kafka |
| 契约测试 | `tests/contract/` | 同上 | api `.api` 文件与 handler 响应结构一致 |
| E2E 冒烟 | `tests/e2e/` | `go test ./tests/e2e/... -tags=e2e` | 登录→建单→审批→台账同步 全链路 |

集成/E2E 测试前执行：`docker compose -f deploy/docker/docker-compose.yml --env-file deploy/docker/docker-compose-env.yml up -d`

### 7.5 Docker Compose 编排约定

* **`deploy/docker/docker-compose.yml`**：定义 PostgreSQL、MySQL、MongoDB、Redis、Kafka（KRaft 单节点）、etcd、Jaeger all-in-one、Prometheus、Grafana 及各类 Exporter；**不包含**业务 Go 进程（业务进程本地 `go run` 或后续独立 compose 叠加）。
* **`deploy/docker/docker-compose-env.yml`**：键值环境变量（或通过 `env_file` 引用），包含各库连接串、端口映射、Kafka Topic 名、JWT 密钥占位符等；**禁止**提交真实生产密钥，使用 `.example` 副本供复制。
* **网络**：所有服务加入 `fams-net` bridge 网络，服务间以 compose service name 互访（如 `postgres:5432`）。
* **数据卷**：数据库与 Prometheus 使用 named volume 持久化；开发环境提供 `make infra-reset` 脚本一键清空。

### 7.6 幂等消费去重表（`asset_event_dedup` - 存储于 MySQL）

配合 5.2 节 Kafka at-least-once 语义，`asset-rpc` 侧维护消费去重表：

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 自增 ID |
| request_id | bigint | Unique, Not Null | 工单 ID，幂等键 |
| event_type | varchar(50) | Not Null | 已处理事件类型 |
| processed_at | datetime | Not Null | 处理完成时间 |

消费逻辑：先 `INSERT`，唯一键冲突则 ACK 跳过；插入成功再更新 `asset_ledger`。

### 7.7 各服务 Prometheus Metrics 端口分配

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

各 API 业务端口（HTTP）约定：user-api `8888`、asset-api `8889`、workflow-api `8890`、inventory-api `8891`、report-api `8892`。

### 7.8 各服务 gRPC 端口分配

| 服务 | gRPC Port | etcd Key 前缀 |
| --- | --- | --- |
| user-rpc | 8081 | `user.rpc` |
| asset-rpc | 8082 | `asset.rpc` |
| workflow-rpc | 8083 | `workflow.rpc` |
| inventory-rpc | 8084 | `inventory.rpc` |

Report Service 无对外 RPC，仅 HTTP API。

### 7.9 Go Workspace 与模块约定

根目录使用 **单模块** `module github.com/fams/backend`（或仓库实际 path），`go.mod` 位于 `assets-db/backend/`。各微服务代码置于 `service/<name>/`，公共库置于 `pkg/`，**不使用** go.work 多模块（降低 AI 与 CI 复杂度）。

go-zero 代码生成约定：

```bash
# API 生成
goctl api go -api service/user/api/user.api -dir service/user/api -style goZero

# RPC 生成
goctl rpc protoc service/user/rpc/user.proto --go_out=service/user/rpc --go-grpc_out=service/user/rpc --zrpc_out=service/user/rpc -style goZero
```

### 7.10 SSO 与 Token 续期约定

| 场景 | 行为 |
| --- | --- |
| 同一 uid 新登录 | 覆盖 Redis `fams:auth:token:${uid}`，旧 Access Token 的 `jti` **不**自动黑名单（自然过期）；若需强制踢人，管理员调用 disable/force-logout |
| Refresh Token | 一次性轮换：刷新成功后签发新 Refresh，旧 Refresh 的 `jti` 写入黑名单 |
| Refresh 重复使用 | 40102（疑似盗用） |
| force-logout API | 将该 uid 当前 Redis session 中的 jti 黑名单 + DEL session key |

### 7.11 转办（Transfer）v1 范围决策

**v1 不实现** `POST /workflow/requests/:id/transfer`。审批仅支持同意/驳回；转办留作 v1.1，避免状态机膨胀。任务书 P4 中该接口标记为 **Out of Scope**。

### 7.12 文档索引

| 文档 | 用途 |
| --- | --- |
| `01-desgin.md` | 架构、数据模型、核心流程、工程规范 |
| `02-plan.md` | 分 Phase 开发任务书（Git/测试/DoD） |
| `03-api-contract.md` | 全部 HTTP API Request/Response 契约 |
| `04-workflow-rules.md` | 四种工单状态机与前置校验 |
| `05-seed-fixtures.md` | 固定 Seed ID/账号（E2E 断言） |
| `06-error-codes.md` | 按接口列出的完整错误码矩阵 |
| `07-inventory-ops.md` | 盘点 scope/归档/比对逐步算法 |
| `08-infra-config.md` | 基础设施文件清单、功能说明、关键设计决策与端口汇总 |
| `09-testing.md` | 测试报告：覆盖率、curl 验证结果、bug 记录 |
| `10-final-status.md` | 开发完成状态：已完成 90%、已知限制、bug 记录、测试速查 |