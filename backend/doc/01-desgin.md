# 基于多人协同的高校固定资产管理系统（FAMS）设计与微服务架构方案报告

## 一、 系统总体设计

### 1.1 总体架构拓扑

系统采用前后端分离的分布式微服务架构，后端基于 `go-zero` 架构搭建，并在最前端引入高性能组件进行流量调度。

* **反向代理层（Nginx）**：作为整个系统后端集群的唯一入口，Nginx 负责处理 SSL/TLS 卸载、静态资源反向代理、全局负载均衡，以及基础的安全防护（如 IP 限流、防恶意爬虫），将动态接口请求精确转发至下游的微服务网关。
* **网关接入层（API Gateway）**：利用 `go-zero` 内置的高性能路由与 JWT 鉴权中间件，统一承接来自 Nginx 转发的 HTTP 请求，执行统一的权限拦截、身份解析与流量控制，并将请求分发至下游 gRPC 微服务集群。
* **微服务业务层（RPC Services）**：各微服务独立部署，服务间通过高性能 gRPC 协议进行内部通信。使用 `etcd` 作为服务注册与发现中心，动态管理服务节点拓扑。
* **多租户/多层级隔离设计**：在 RPC 拦截器中解析网关透传的身份凭证，动态对底层数据库连接及 SQL/NoSQL 语句执行维度注入，实现全系统非功能性需求中的安全性与权限隔离。

### 1.2 协同设计与非功能性需求规范

根据总体设计要求，系统核心设计必须满足以下规范：

1. **多角色协同分工**：校级管理员、学院管理员、普通师生权限分离、各司其职。
2. **多级流程协同审批**：资产领用、归还、报修、报废等业务需多人逐级审核流转。
3. **多人联合协同盘点**：多名盘点人员同时在线录入数据、由系统引擎自动核对资产差异，共同完成盘点。
4. **全院数据协同共享与隔离**：同学院人员可共享查看本部门资产台账，非同部门数据相互隔离。
5. **审计留痕**：所有操作必须实名留痕、可追溯，精准记录操作人员与操作时间。
6. **非功能性指标**：系统需具备高协同性、严格的安全性控制、简洁清晰的可用性以及规范易维护的代码结构。

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
* **数据可见性**：严格受限于所属组织节点，仅能维护和查询本学院及其下属二级实验室的资产数据。系统通过数据隔离中间件，在 SQL 执行前自动组装 `WHERE department_id = :user_dept_id`。


* **普通师生用户（级别 3）**：
* **操作权限**：可在线提交资产领用、归还、故障报修工单及报废申请。
* **数据可见性**：默认仅可见个人名下绑定的资产单据与个人领用资产。对于“全院数据协同共享”需求，系统额外提供特定只读 API，允许师生跨组查看所属学院内标记为“共享/公用”状态的实验器材与设备。



---

## 三、 基于 go-zero 的微服务划分与功能职责

结合业务边界内聚度、数据隔离级别及高并发协同场景，将后端划分为 5 个核心微服务：

### 3.1 用户与权限微服务（User Service）

* **功能职责**：负责系统用户登录、注册、个人信息管理；维护独立的树状组织架构（学院/部门/实验室）；管理三级角色的级别设定与动态鉴权。
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
* `workflow-api`：接收师生提交的申请、管理员的审核指令（同意/驳回/转办）。
* `workflow-rpc`：推进流转状态机，负责校验多级审批人权限（学院初审 $\rightarrow$ 校级复审），并在节点终审通过时，负责向审计链路写入不可篡改的流水痕迹。



### 3.4 多人协同盘点微服务（Inventory Service）

* **功能职责**：处理高并发的多人在线联合盘点业务。支持校级/院级管理员发布特定范围与时间的盘点任务；承接多名工作人员对同区域 Luckysheet 前端电子表格的并发录入；利用高效比对引擎对账面原始数据与实际录入数据进行自动化精确校验，快速计算盘盈、盘亏差异并生成结构化分析明细。
* **架构设计**：
* `inventory-api`：对接纯前端协同表格的数据批量提交，处理并发写入冲突。
* `inventory-rpc`：管理盘点任务生命周期，控制自动化比对任务队列调度。



### 3.5 数据统计与报表微服务（Report Service）

* **功能职责**：提供各学院资产数量统计、资产类型分布统计、盘点差异统计等高维度聚合报表；支持全院协同共享视角的报表在线预览与数据异步导出。
* **架构设计**：
* `report-api`：独立承接大范围只读聚合查询，防止复杂的报表统计 SQL 拖垮在线核心业务数据库。



---

## 四、 底层数据库混合选型方案（Polyglot Persistence）

系统针对微服务各自的业务特性，实施混合存储设计，确保强一致性、高并发性以及非结构化扩展能力的平衡。

### 4.1 混合存储矩阵规划

* **PostgreSQL (16.0)**：应用于**用户与权限微服务**、**工作流审批微服务**、**协同盘点微服务**。
* *选型理由*：高校组织架构属于典型的多级树状结构，PostgreSQL 完美的递归查询能力（`WITH RECURSIVE` 语法）能够通过单条 SQL 高效获取整条组织链或下属所有二级单位，极大精简业务代码。工作流与盘点明细属于核心审计资产，依赖 PostgreSQL 严苛的事务一致性与 MVCC 并发控制。


* **MySQL (8.0)**：应用于**固定资产台账微服务**。
* *选型理由*：资产底账具有绝对结构化、类财务审计的特征。选用 MySQL 配合 InnoDB 存储引擎，利用其成熟的 B+ Tree 索引为资产编号（`asset_no`）建立强唯一性约束，通过标准的行级锁机制保障台账基础数据的原子性与稳定性。


* **MongoDB (7.0)**：应用于**协同盘点微服务（临时草稿）**。
* *选型理由*：多名盘点人员在 Luckysheet 前端多人协同表格中高频录入时，会产生大量临时的、未定型的行列单元格数据及多维备注。此类数据格式多变且写入密集，MongoDB 的 Document 模型支持 Schema-less 动态扩展，能够承载此类超高并发的协同暂存数据。


* **Redis (7.2)**：用于存储全系统的分布式用户 Session Token、JWT 撤销黑名单、协同流转状态缓存以及多人盘点时针对单台资产的分布式锁，防止并发覆盖。

### 4.2 核心数据表/集合设计（SQL & NoSQL 模式）

#### 4.2.1 组织架构表（`sys_department` - 存储于 PostgreSQL）

用于高效表达高校多层级院系与实验室的关联拓扑。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 组织节点唯一 ID |
| parent_id | bigint | Not Null | 父级组织 ID（顶层学校节点为 0） |
| dept_name | varchar(100) | Not Null | 部门/学院/实验室名称 |
| dept_code | varchar(30) | Unique, Not Null | 组织机构代码 |
| sort_order | int | Not Null, Default 0 | 排序权重 |

#### 4.2.2 用户表（`sys_user` - 存储于 PostgreSQL）

解耦后的纯净用户表，仅保留核心认证属性。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 用户自增唯一 ID |
| username | varchar(50) | Unique, Not Null | 登录账号/工号 |
| password_hash | varchar(255) | Not Null | 加盐哈希后的密码密文 |
| real_name | varchar(50) | Not Null | 真实姓名（实名审计核心） |
| role_level | tinyint | Not Null | 级别：1-校级管理员, 2-学院管理员, 3-普通师生 |
| department_id | bigint | Not Null | 所属组织架构节点外键 ID |
| status | tinyint | Not Null | 状态：1-启用, 0-禁用 |

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
| status | tinyint | Not Null | 状态：1-在库, 2-领用中, 3-维修中, 4-已报废 |

#### 4.2.4 审批申请主表（`workflow_request` - 存储于 PostgreSQL）

记录多角色多级流程流转状态。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 审批工单唯一自增 ID |
| asset_id | bigint | Not Null | 关联资产 ID |
| requester_id | bigint | Not Null | 申请人用户 ID |
| type | tinyint | Not Null | 业务类型：1-领用, 2-归还, 3-报修, 4-报废 |
| current_stage | tinyint | Not Null | 当前审批阶段：1-院级初审, 2-校级复审, 3-归档结束 |
| status | tinyint | Not Null | 流程状态：1-审批中, 2-审批通过, 3-已被驳回 |
| reason | varchar(255) | Null | 申请缘由说明或故障描述 |

#### 4.2.5 审批留痕流水表（`workflow_log` - 存储于 PostgreSQL）

完全满足协同工作实名留痕审计要求的关键数据表。

| 字段名 | 类型 | 约束 | 说明 |
| --- | --- | --- | --- |
| id | bigint | PK | 审计流水唯一自增 ID |
| request_id | bigint | Not Null | 关联的审批申请单 ID |
| operator_id | bigint | Not Null | 实际执行审批操作的人员 ID |
| action | varchar(50) | Not Null | 动作快照（如：提交申请、院级初审同意、校级复审通过、驳回） |
| comment | varchar(255) | Null | 具体审批签署意见 |
| operate_time | datetime | Not Null | 操作动作发生的精确时间 |

#### 4.2.6 多人协同盘点草稿集合（`inventory_draft` - 存储于 MongoDB）

应对纯前端 Luckysheet 多人实时高并发协同编辑的高性能无模式集合。

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
| asset_id | bigint | Not Null | 关联的资产 ID |
| operator_id | bigint | Null | 最终执行实名核对录入的盘点人员 ID |
| is_scanned | tinyint | Not Null, Default 0 | 状态快照：0-未盘点, 1-已盘点 |
| actual_location | varchar(100) | Null | 现场实际清查发现的物理地点 |
| diff_status | tinyint | Not Null, Default 0 | 账实比对结果：0-未比对, 1-账实相符, 2-盘盈, 3-盘亏 |

---

## 五、 中间件配置与核心业务协同详细流程

系统通过 Nginx 进行前置请求拦截与分发，并合理调配高性能中间件 Redis 与分布式消息队列 Kafka，实现各个微服务间解耦与高吞吐协同。

### 5.1 用户登录与凭证鉴权核心流程

1. **涉及组件**：Nginx、`user-api`、`user-rpc`，PostgreSQL（用户权限库），Redis（凭证缓存）。
2. **详细流程拓扑**：
* 客户端发起登录请求，请求首先到达 **Nginx 反向代理层**，Nginx 校验请求合法性后，将其转发给后端的 `user-api`（`/v1/user/login`）。
* `user-api` 将数据打包后通过 gRPC 协议传输至后台 `user-rpc` 微服务的 `FindUser` 方法。
* `user-rpc` 连接 PostgreSQL 数据库，快速检索 `sys_user` 关联 `sys_department`。比对密码加盐哈希值，确认账户未被禁用。
* 验证无误后，`user-rpc` 提取出该用户的 `id`（UID）、`role_level` 及 `department_id`（DeptID），利用 `go-zero` 框架内置的 JWT 生成组件，将这三个核心维度强制注入 JWT Claims 载荷，签发 Token。
* 同时，`user-rpc` 同步将此凭据写入 **Redis** 中（Key 为 `fams:auth:token:${uid}`，Value 为生成的 Token 字符串，过期时间设置为 24 小时），用以支持单点登录（SSO）控制与在线状态管理。
* `user-api` 最终将 Token 返回客户端。后续客户端发起的任何请求都会经过 Nginx 并由网关拦截器无状态解析出该用户的级别与部门 ID。



### 5.2 资产领用/归还多级审批与状态同步流程（Kafka 驱动解耦）

1. **涉及组件**：Nginx、`workflow-api`、`workflow-rpc` nudge `asset-rpc`，PostgreSQL（工作流库），MySQL（资产台账库），Kafka（分布式事件总线）。
2. **详细流程拓扑**：
* **工单创建**：普通师生在前端发起资产领用，请求经 Nginx 转发至 `workflow-api`，解析 JWT 拦截器注入的身份，发起 `workflow-rpc`，向 PostgreSQL 写入工作流主表 `workflow_request`，初始阶段标记为 `1 (院级初审中)`。
* **两级流转控制**：
* 学院管理员查询待办单，`workflow-rpc` 从 PostgreSQL 过滤出 `department_id` 与该管理员完全一致的记录（确保隔离，禁止越权操作数据），管理员初审通过，状态推移至 `2 (校级复审中)`。
* 校级管理员全局可见该待办单，审查完毕后在前端触发终审通过。


* **实名审计与本地事务**：`workflow-rpc` 在 PostgreSQL 中开启本地事务，将单据状态固化变更为 `3 (审批通过)`，并在同一事务内向 `workflow_log` 写入一条不可篡改的实名操作流水。
* **跨库解耦事件发布**：PostgreSQL 本地事务成功提交后，为了不与资产台账的物理 MySQL 库产生分布式强事务耦合，`workflow-rpc` 立即调用 Kafka 生产者组件，向名为 `fams-asset-lifecycle-events` 的 **Kafka Topic** 发送一条资产状态变更为“领用中”的异步领域事件：
```json
{
  "event_type": "ASSET_USE_APPROVED",
  "asset_id": 40912,
  "target_status": 2,
  "assigned_user_id": 20261102,
  "operator_id": 10001,
  "timestamp": 1783353600
}

```


* **资产底账消费同步**：`asset-rpc` 微服务下属的 Kafka 消费者集群监听到该消息。收到事件报文后，解析出资产主键，立即操作底层物理 **MySQL** 数据库，将 `asset_ledger` 表中对应资产的 `status` 变更为 `2 (领用中)`，并将 `user_id` 字段更新为事件中指定的领用人 ID。



### 5.3 多人联合协同盘点流程（Redis 分布式锁与 MongoDB 高并发暂存）

1. **涉及组件**：Nginx、`inventory-api`、`inventory-rpc`、`asset-rpc`，Redis（冲突控制锁），MongoDB（非结构化高频草稿），PostgreSQL（盘点归档库），Kafka（异步比对队列）。
2. **详细流程拓扑**：
* **多人并发协同无冲突写入**：当多名学院盘点人员同时在线盘点录入，通过纯前端 Luckysheet 组件批量提交数据时，请求通过 Nginx 均衡负载打入 `inventory-api`。
* `inventory-api` 拦截到请求后，针对更新报文里的每一台资产编号（`asset_no`），首先向全局 **Redis** 集群尝试申请分布式排他锁：
```
SET fams:lock:inventory:${asset_no} ${operator_id} NX PX 5000

```


若某个资产的 Redis 锁申请失败，则立即向当前操作人员的前端抛出冲突提示，从而保证并发修改绝对安全。
* **非结构化草稿高速缓存**：成功抢占 Redis 锁的请求，其现场采集的非结构化动态数据全速写入 **MongoDB** 的 `inventory_draft` 集合中进行无模式高并发草稿暂存。
* **组长终结归档与规则比对**：盘点终结时，`inventory-rpc` 从 MongoDB 中聚合读出该任务所有的最终暂存草稿，以标准的 PostgreSQL 强事务批量固化写入 `inventory_record` 表，将 `is_scanned` 字段批量置为 `1 (已盘点)`。
* 明细固化入库后，`inventory-rpc` 产生一个比对信号，发送至 **Kafka** 队列 `fams-inventory-comparison-tasks` 中。
* 后台专用的自动化比对 Worker 进程消费此消息，通过 gRPC 协议调用 `asset-rpc` 提取出 **MySQL** 中该资产的原始账面数据（存放地点、归属部门等），在内存中通过确定的业务规则与 PostgreSQL 中现场采集的实际物理地点进行精准字段级对齐核对。
* 判定属于“账实相符”、“盘盈”还是“盘亏”后，Worker 将最终的差异状态码（`diff_status`）异步更新回 PostgreSQL 的 `inventory_record` 表中，高效完成多人协同盘点与自动化账实差异清查。