# FAMS 错误码矩阵（v1）

> 实现位置：`pkg/errx/codes.go`  
> HTTP 响应格式见 `01-desgin.md` §7.3

---

## 1. 错误码分段

| 段位 | 用途 |
| --- | --- |
| 0 | 成功 |
| 400xx | 请求参数错误 |
| 401xx | 认证失败 |
| 403xx | 权限/越权 |
| 404xx | 资源不存在 |
| 409xx | 冲突/重复 |
| 422xx | 业务状态不允许 |
| 500xx | 服务器内部错误 |
| 503xx | 依赖不可用 |

---

## 2. 全局错误码表

| code | HTTP | message（默认） | 说明 |
| --- | --- | --- | --- |
| 0 | 200 | ok | 成功 |
| 40001 | 400 | 请求参数无效 | JSON 解析失败、缺少必填字段 |
| 40002 | 400 | 分页参数无效 | page<1 或 pageSize 不在 1–100 |
| 40003 | 400 | 时间格式无效 | ISO8601 解析失败 |
| 40101 | 401 | 未登录或凭证无效 | JWT 缺失/过期/签名错误/用户名密码错误 |
| 40102 | 401 | 凭证已撤销 | 黑名单/Refresh 重用 |
| 40301 | 403 | 无操作权限 | 角色级别不足 |
| 40302 | 403 | 无权访问该数据 | 组织子树隔离 |
| 40303 | 403 | 未指派参与该盘点任务 | inventory submit |
| 40401 | 404 | 资源不存在 | 通用 |
| 40402 | 404 | 用户不存在 | |
| 40403 | 404 | 资产不存在 | |
| 40404 | 404 | 工单不存在 | |
| 40405 | 404 | 盘点任务不存在 | |
| 40406 | 404 | 导出任务不存在 | |
| 40901 | 409 | 操作冲突 | Redis 锁/CAS |
| 40902 | 409 | 该资产已有进行中的申请 | workflow 唯一索引 |
| 40903 | 409 | 唯一键冲突 | username/asset_no/dept_code |
| 42201 | 422 | 业务状态不允许 | 资产状态/工单状态/任务状态 |
| 42202 | 422 | 工单已归档 | 重复 approve/reject |
| 42203 | 422 | 时间窗设置无效 | end_time ≤ start_time |
| 42204 | 422 | 导出任务未完成 | 下载时 status≠2 |
| 42205 | 422 | 存在进行中的审批，无法修改资产 | PUT asset |
| 50001 | 500 | 服务器内部错误 | 未预期错误 |
| 50301 | 503 | 服务依赖不可用 | DB/Redis/Kafka/Mongo |

---

## 3. 按接口错误矩阵

### 3.1 POST `/user/login`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| username/password 错误 | 40101 |
| 用户 status=0 | 40301（账户已禁用） |
| body 无效 | 40001 |

### 3.2 POST `/user/refresh`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| Refresh 过期/无效 | 40101 |
| Refresh 已用/黑名单 | 40102 |

### 3.3 POST `/user/users`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| role=2 创建 role≤2 | 40301 |
| departmentId 不在子树 | 40302 |
| username 重复 | 40903 |
| 参数缺失 | 40001 |

### 3.4 POST `/asset/assets`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| asset_no 重复 | 40903 |
| departmentId 越权 | 40302 |
| 参数无效 | 40001 |

### 3.5 DELETE `/asset/assets/:id`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| 存在 open workflow | 42205 |
| 不存在 | 40403 |
| 越权 | 40302 |

### 3.6 POST `/workflow/requests`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| 资产不存在 | 40403 |
| CheckAssetForWorkflow 失败 | 42201 |
| 重复 open 工单 | 40902 |
| type 不在 1–4 | 40001 |

### 3.7 POST `/workflow/requests/:id/approve`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| 工单不存在 | 40404 |
| 已归档/驳回 | 42202 |
| 阶段/角色不匹配 | 40301 |
| dept 越权 | 40302 |
| 终审资产 Check 失败 | 42201 |

### 3.8 POST `/workflow/requests/:id/reject`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| 40404 / 42202 / 40301 / 40302 | 同 approve |

### 3.9 POST `/inventory/tasks/:id/submit`

| 条件 | code |
| --- | --- |
| 全部成功 | 0（success 填满） |
| 部分冲突 | 0（conflicts 非空，仍 200） |
| 任务不存在 | 40405 |
| 任务非 status=1 | 42201 |
| 未指派 | 40303 |
| items 空 | 40001 |
| Redis/Mongo 不可用 | failures 内 50301 或整单 50301 |

### 3.10 POST `/inventory/tasks/:id/archive`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| 任务不存在 | 40405 |
| 已归档（幂等） | 0 + 返回当前状态 |
| 未到期且 force=false | 42201 |
| force=true 但 role=3 | 40301 |

### 3.11 POST `/report/export`

| 条件 | code |
| --- | --- |
| 成功 | 0 |
| exportType 非法 | 40001 |
| 参数越权 | 40302 |

### 3.12 GET `/report/export/:jobId/download`

| 条件 | code |
| --- | --- |
| 成功 | CSV 200 |
| job 不存在 | 40406 |
| status≠2 | 42204 |

---

## 4. gRPC → HTTP 映射规则

| gRPC codes | 映射 code |
| --- | --- |
| InvalidArgument | 40001 |
| Unauthenticated | 40101 |
| PermissionDenied | 40301 |
| NotFound | 40401 |
| AlreadyExists | 40903 |
| FailedPrecondition | 42201 |
| Unavailable | 50301 |
| Internal | 50001 |

RPC 响应 `message` 字段携带 `errx` 数字码字符串，API 层解析后返回统一 JSON。

---

## 5. 日志与对外暴露

| 级别 | 行为 |
| --- | --- |
| 4xx | log Info，不打印 stack |
| 5xx | log Error + stack，响应仅 50001 |
| 50301 | log Error + 依赖名 |

禁止在 JSON 中返回：`sql`, `stack`, `dsn`

---

*文档版本：v1.0 | 2026-07-07*
