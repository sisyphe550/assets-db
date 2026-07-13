# FAMS HTTP API 契约（v1）

> 与 `01-desgin.md` §7.3 响应格式一致。所有需鉴权接口 Header：`Authorization: Bearer <accessToken>`  
> Base URL（经 Nginx）：`http://localhost/api/v1`

---

## 0. 通用约定

### 0.1 统一响应包裹

成功：`{ "code": 0, "message": "ok", "data": <T> }`  
失败：`{ "code": <int>, "message": "<string>", "data": null }`

### 0.2 分页（列表接口通用 Query）

| 参数 | 类型 | 默认 | 说明 |
| --- | --- | --- | --- |
| page | int | 1 | 页码，≥1 |
| pageSize | int | 20 | 每页条数，1–100 |

**分页响应 data 结构**：

```json
{
  "list": [],
  "page": 1,
  "pageSize": 20,
  "total": 100
}
```

### 0.3 部分成功语义（盘点 batch submit）

HTTP 200 + `code: 0`，即使 `data.conflicts` 非空，表示请求已被处理（逐条独立成败）。

---

## 1. User Service（`/user`）

### 1.1 POST `/user/login`

**鉴权**：否

**Request**：

```json
{
  "username": "admin_school",
  "password": "Test@123456"
}
```

**Response data**：

```json
{
  "accessToken": "eyJ...",
  "refreshToken": "eyJ...",
  "expiresIn": 7200,
  "tokenType": "Bearer"
}
```

**错误**：40101 用户名或密码错误；40301 账户已禁用

---

### 1.2 POST `/user/refresh`

**鉴权**：Refresh Token（Body 传递，不用 Access）

**Request**：

```json
{
  "refreshToken": "eyJ..."
}
```

**Response data**：同 login

**错误**：40101 过期；40102 已撤销/重复使用

---

### 1.3 POST `/user/logout`

**鉴权**：Access Token

**Request**：空 body 或 `{}`

**Response data**：`null`

**错误**：40101 / 40102

---

### 1.4 GET `/user/me`

**Response data**：

```json
{
  "id": 10001,
  "username": "admin_school",
  "realName": "张校管",
  "roleLevel": 1,
  "departmentId": 1,
  "departmentName": "本校",
  "status": 1
}
```

---

### 1.5 GET `/user/departments/tree`

**Query**：无（按当前用户权限裁剪子树）

**Response data**：

```json
{
  "nodes": [
    {
      "id": 1,
      "parentId": 0,
      "deptName": "本校",
      "deptCode": "ROOT",
      "path": "/1/",
      "children": []
    }
  ]
}
```

---

### 1.5.1 GET `/user/departments/college-subtree`

**鉴权**：是

**说明**：返回当前用户所属学院及全部下属部门 ID（用于共享资产过滤等场景）

**Response data**：

```json
{
  "deptIds": [15, 103, 104]
}
```

---

### 1.6 POST `/user/departments`

**权限**：role=1

**Request**：

```json
{
  "parentId": 1,
  "deptName": "机械工程学院",
  "deptCode": "ME",
  "sortOrder": 10
}
```

**Response data**：新建节点完整对象（含 `id`, `path`）

**错误**：40401 父节点不存在；40903 dept_code 重复

---

---

### 1.10 GET `/user/users`

**权限**：role≤2

**Query**：`page`, `pageSize`, `keyword?`（匹配 username/realName）, `departmentId?`, `roleLevel?`

**行为**：
- role=1：全校用户
- role=2：仅本学院子树用户

**Response data**：标准分页，`list` 项含 `id`, `username`, `realName`, `roleLevel`, `departmentId`, `departmentName`, `status`（不含 password）

---

### 1.11 GET `/user/users/:id`

**权限**：role≤2（role=2 仅限子树内用户）

**Response data**：同 list 单项结构

**错误**：40402 用户不存在；40302 越权

---

### 1.7 POST `/user/users`（管理员创建用户，非自助注册）

**权限**：role=1 任意创建；role=2 仅能在本学院子树创建 role=3 用户

**Request**：

```json
{
  "username": "student_001",
  "password": "Test@123456",
  "realName": "李同学",
  "roleLevel": 3,
  "departmentId": 103
}
```

**Response data**：用户对象（不含 password）

**错误**：40301 学院管理员创建 role≤2；40302 部门不在子树；40903 username 重复

---

### 1.8 PUT `/user/users/:id/status`

**权限**：role=1；role=2 仅本学院子树用户

**Request**：

```json
{
  "status": 0
}
```

**副作用**：status=0 时将该用户 session jti 写黑名单

**错误**：40401；40302

---

### 1.9 POST `/user/users/:id/force-logout`

**权限**：role=1 或 role=2（子树内）

**Response data**：`null`

---

## 2. Asset Service（`/asset`）

### 2.1 POST `/asset/assets`

**权限**：role≤2 + departmentId 在子树内

**Request**：

```json
{
  "assetNo": "EQUIP-2026-0001",
  "name": "激光切割机",
  "category": "设备",
  "price": 150000.00,
  "purchaseTime": "2025-09-01T00:00:00+08:00",
  "location": "一号实验楼101",
  "departmentId": 15,
  "isShared": 0
}
```

**Response data**：完整资产对象（含 `id`, `status`=1）

**错误**：40903 asset_no 重复；40302 部门越权

---

### 2.2 GET `/asset/assets`

**Query**：`page`, `pageSize`, `category?`, `status?`, `keyword?`（匹配 name/asset_no）, `scope?`（`my`=仅当前用户领用资产）, `userId?`（`me` 或具体 ID）

**Response data**：分页结构，list 项：

```json
{
  "id": 501,
  "assetNo": "EQUIP-2026-0001",
  "name": "激光切割机",
  "category": "设备",
  "price": 150000.00,
  "purchaseTime": "2025-09-01T00:00:00+08:00",
  "location": "一号实验楼101",
  "departmentId": 15,
  "userId": null,
  "isShared": 0,
  "status": 1
}
```

默认过滤 `deleted_at IS NULL`。

---

### 2.3 GET `/asset/assets/:id`

**错误**：40401；40302

---

### 2.4 PUT `/asset/assets/:id`

**Request**：可部分更新 `name`, `category`, `location`, `departmentId`, `isShared`（不可直接改 `status`，走工作流）

**错误**：42201 审批中资产禁止改 departmentId

---

### 2.5 DELETE `/asset/assets/:id`

**逻辑删除**：写 `deleted_at`，返回 `null`

**错误**：42201 存在 status=1 审批中工单

---

### 2.6 GET `/asset/assets/shared`

**权限**：role=3

**Query**：`page`, `pageSize`

**过滤规则**：

```sql
WHERE is_shared = 1
  AND deleted_at IS NULL
  AND department_id IN (:user_college_subtree_ids)
```

`user_college_subtree_ids`：取用户 `department_id` 向上找到学院节点（path 深度=2）再展开子树。

---

## 3. Workflow Service（`/workflow`）

> 状态机详见 `04-workflow-rules.md`

### 3.1 POST `/workflow/requests`

**权限**：role=3

**Request**：

```json
{
  "assetId": 501,
  "type": 1,
  "reason": "课程设计需要使用"
}
```

`type`：1-领用, 2-归还, 3-报修, 4-报废

**Response data**：

```json
{
  "id": 88001,
  "assetId": 501,
  "type": 1,
  "currentStage": 1,
  "status": 1,
  "createdAt": "2026-07-06T10:00:00+08:00"
}
```

**错误**：40401 资产不存在；42201 前置状态不满足；40902 已有审批中工单

---

### 3.2 GET `/workflow/requests`

**Query**：`page`, `pageSize`, `scope=my|todo|done|all`, `type?`, `status?`, `assetId?`

| scope | 行为 |
| --- | --- |
| my | requester_id = 当前 uid |
| todo | role=2 院级 stage=1 且 dept 在子树；role=1 校级 stage=2 |
| done | 当前用户参与审批过的 log |
| all | 全部工单（role≤2） |

`assetId`：可选，筛选指定资产的工单历史。

---

### 3.3 GET `/workflow/requests/:id`

**Response data**：工单 + `logs[]`（按 operate_time 升序）

---

### 3.4 POST `/workflow/requests/:id/approve`

**权限**：role≤2，阶段权限见 workflow-rules

**Request**：

```json
{
  "comment": "同意领用"
}
```

**错误**：42202 工单已归档；40301 阶段权限不足；42201 资产状态已变

---

### 3.5 POST `/workflow/requests/:id/reject`

**Request**：同 approve

**结果**：status=3，current_stage 不变，写 log

---

## 4. Inventory Service（`/inventory`）

> 操作流程详见 `07-inventory-ops.md`

### 4.1 POST `/inventory/tasks`

**权限**：role≤2

**Request**：

```json
{
  "taskName": "2026信息学院实验室盘点",
  "scopeDeptId": 15,
  "startTime": "2026-07-01T00:00:00+08:00",
  "endTime": "2026-07-31T23:59:59+08:00",
  "assigneeIds": [10003, 10004]
}
```

**Response data**：任务草稿对象。新建后 `status=0`（待发布），`expectedAssetCount=0`；管理员需先配置盘点条目，再发布任务。

**错误**：40302 scope 超出管理员子树；42203 时间窗非法（end ≤ start）

---

### 4.2 GET `/inventory/tasks`

**Query**：`page`, `pageSize`, `status?`, `scope?`（`assigned`=仅当前用户被指派任务，role=3 默认行为）

**Response data**（分页 list 项）：

```json
{
  "id": 1,
  "taskName": "2026信息学院实验室盘点",
  "scopeDeptId": 15,
  "creatorId": 10002,
  "startTime": "2026-07-01T00:00:00+08:00",
  "endTime": "2026-07-31T23:59:59+08:00",
  "status": 0,
  "assigneeIds": [10003, 10004],
  "expectedAssetCount": 50,
  "submittedCount": 15
}
```

**权限**：role≤2 见管辖范围内任务；role=3 仅见指派给自己的任务

---

### 4.2.1 GET `/inventory/tasks/:id`

**Response data**：同 list 单项结构

**错误**：40405 任务不存在；40302/40303 越权或未指派

---

### 4.3 GET `/inventory/tasks/:id/items`

**权限**：role≤2 管理员；院级管理员仅可访问本院子树范围任务

**说明**：返回任务已选择盘点条目，以及该任务 scope 内可选资产清单。用于待发布任务的条目配置。

**Response data**：

```json
{
  "list": [
    {
      "assetId": 501,
      "assetNo": "EQUIP-2026-0001",
      "name": "激光切割机",
      "bookLocation": "一号实验楼101"
    }
  ],
  "available": [
    {
      "assetId": 501,
      "assetNo": "EQUIP-2026-0001",
      "name": "激光切割机",
      "bookLocation": "一号实验楼101"
    }
  ],
  "total": 1
}
```

---

### 4.3.1 PUT `/inventory/tasks/:id/items`

**权限**：role≤2 管理员；仅 `status=0` 可修改

**Request**：

```json
{
  "assetIds": [501, 502]
}
```

**Response data**：更新后的 `list` 与 `total`

**错误**：42201 任务状态非法；40001 资产不在任务 scope 内或请求参数无效

---

### 4.3.2 POST `/inventory/tasks/:id/publish`

**权限**：role≤2 管理员；仅 `status=0` 可发布

**说明**：任务必须至少配置 1 条盘点资产，且至少指派 1 名盘点员。发布后 `status=1`，普通用户开始盘点。

**Response data**：

```json
{
  "taskId": 1,
  "status": 1,
  "expectedAssetCount": 2
}
```

---

### 4.3.3 GET `/inventory/tasks/:id/expected-assets`

**说明**：返回该任务应盘资产清单（供前端表格预填）。新任务按 `inventory_task_item` 返回已配置条目；历史任务若没有条目配置，继续兼容为 scope 内账面资产清单。

**Response data**：

```json
{
  "list": [
    {
      "assetId": 501,
      "assetNo": "EQUIP-2026-0001",
      "name": "激光切割机",
      "bookLocation": "一号实验楼101"
    }
  ],
  "total": 50
}
```

---

### 4.4 POST `/inventory/tasks/:id/submit`

**权限**：须在 `inventory_task_assignee` 中；或 role≤2 管理员

**Request**：

```json
{
  "items": [
    {
      "assetNo": "EQUIP-2026-0001",
      "modifiedCells": {
        "actual_location": "一号实验楼101",
        "temp_notes": "正常",
        "found_name": ""
      },
      "expectedUpdatedAt": null
    },
    {
      "assetNo": "UNKNOWN-9999",
      "modifiedCells": {
        "actual_location": "仓库",
        "found_name": "未登记投影仪"
      },
      "expectedUpdatedAt": null
    }
  ]
}
```

**Response data**：

```json
{
  "success": ["EQUIP-2026-0001"],
  "conflicts": [],
  "failures": []
}
```

**conflicts 项**：`{ "assetNo", "code": 40901, "message" }`  
**failures 项**：`{ "assetNo", "code": 50301, "message" }`

**错误（整单）**：40401 任务不存在；42201 任务非进行中；40303 未指派

---

### 4.5 POST `/inventory/tasks/:id/archive`

**Request**：

```json
{
  "force": false
}
```

`force=true`：role≤2 可无视时间窗强制归档

**Response data**：`{ "taskId", "archivedRecordCount", "comparisonJobQueued": true }`

---

### 4.6 GET `/inventory/tasks/:id/records`

**Query**：`page`, `pageSize`, `diffStatus?`

---

## 5. Report Service（`/report`）

### 5.1 GET `/report/assets/by-dept`

**Query**：`date?`（默认今天，读快照表）

**Response data**：

```json
{
  "items": [
    {
      "departmentId": 15,
      "departmentName": "信息工程学院",
      "totalCount": 120,
      "inStockCount": 80,
      "inUseCount": 30,
      "totalValue": 1500000.00
    }
  ]
}
```

---

### 5.2 GET `/report/assets/by-category`

**Query**：`departmentId?`（校级可选；院级自动限制子树）

---

### 5.3 GET `/report/inventory/diff/:taskId`

**Response data**：来自 `rpt_inventory_diff_summary` + 明细可选分页

---

### 5.4 POST `/report/export`

**Request**：

```json
{
  "exportType": "asset_list",
  "params": {
    "departmentId": 15,
    "category": "设备"
  }
}
```

`exportType`：`asset_list` | `inventory_diff` | `workflow_log`

**Response data**：

```json
{
  "jobId": 90001
}
```

---

### 5.5 GET `/report/export/:jobId`

**Response data**：

```json
{
  "jobId": 90001,
  "status": 2,
  "downloadUrl": "/api/v1/report/export/90001/download",
  "errorMessage": null
}
```

`status`：0-排队, 1-处理中, 2-已完成, 3-失败

---

### 5.6 GET `/report/export/:jobId/download`

**Response**：`Content-Type: text/csv`，文件流

**错误**：42204 job 未完成；40401 job 不存在

---

## 6. gRPC 契约摘要（内部）

完整 proto 文件路径：`service/<name>/rpc/*.proto`

### asset.proto 关键 RPC

```protobuf
message CheckAssetForWorkflowReq {
  int64 asset_id = 1;
  int32 workflow_type = 2; // 1-4
  int64 requester_id = 3;
}
message CheckAssetForWorkflowResp {
  bool ok = 1;
  int64 department_id = 2;
  string reject_reason = 3; // 42201 文案
}

message ChangeAssetStatusReq {
  int64 request_id = 1;
  int64 asset_id = 2;
  int32 target_status = 3;
  int64 assigned_user_id = 4; // 领用时填 requester；归还时填 0
}
```

`CheckAssetAvailable` 已重命名为 `CheckAssetForWorkflow`，按 type 分支校验（见 `04-workflow-rules.md`）。

### user.proto 关键 RPC

```protobuf
rpc GetDeptSubtree(GetDeptSubtreeReq) returns (GetDeptSubtreeResp);
// Resp: repeated int64 dept_ids; role=1 时返回空表示不限制
```

---

*文档版本：v1.0 | 2026-07-07*
