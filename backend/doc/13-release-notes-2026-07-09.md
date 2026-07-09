# FAMS 发布说明（2026-07-09）

> 汇总 2026-07-08 ~ 2026-07-09 前后端修复与功能补全，供文档与交接对齐代码现状。

---

## 1. 后端

### 1.1 院级资产可见性（P0）

**现象**：院级管理员「本院资产」列表为空，无法看到下级实验室资产。

**根因**：`asset-api` 未挂载 `RequireDeptScope`，`List` 中 `GetDeptSubtree` 为空导致直接返回空列表。

**修复**：`visibleDeptIDs()` 在缺少中间件时调用 `user-api /departments/college-subtree`；`Detail` / `Update` / `Delete` 增加 `ensureAssetInScope`。

**涉及文件**：`service/asset/api/internal/handler/assethandler.go`

### 1.2 工单访问控制（P0）

**现象**：任意登录用户可读任意工单详情；院级管理员可审批其他学院工单。

**修复**：
- `Detail`：校级全量；院级仅 `department_id ∈ 子树`；师生仅本人申请
- `Approve` / `Reject` 院级阶段校验工单所属院系

**涉及文件**：`service/workflow/api/internal/handler/workflowhandler.go`

### 1.3 盘点 scope 子树（P1）

**现象**：院级 scope=学院时，`expected-assets` 与应盘计数不含下级实验室资产。

**修复**：`scopeDeptIDs()` 统一展开 scope 部门子树，用于 `expected-assets`、归档比对、`countExpectedAssets`。

**涉及文件**：`service/inventory/api/internal/handler/invhandler.go`

### 1.4 工单列表申请人姓名（P1）

**修复**：列表/详情返回 `requesterName`（关联 `sys_user.real_name`）。

### 1.5 报表异步导出（P2）

**修复**：
- `report-api` 路由：`/export/:jobId`、`/export/:jobId/download`
- `ExportStatus` 完成时返回 `downloadUrl`；下载读取 worker 生成的 CSV
- `export-worker` 支持 `asset_list` / `workflow_log` / `inventory_diff`（后者需 `params.taskId`）
- 导出任务 ID 使用 `MAX(id)+1` 写入（`rpt_export_job` 无自增）

**运行**：需同时启动 `report-api` 与 `export-worker`，Redis 可用。

---

## 2. 前端

### 2.1 多标签登录隔离

Token 从 `localStorage` 迁至 `sessionStorage`，清除 legacy 项。

### 2.2 院级全部工单

新增 `/college/workflow/all`；`scope=all|todo` 按院系子树过滤。

### 2.3 盘点草稿与编辑体验

| 问题 | 修复 |
|---|---|
| 草稿 GET 持久化、CAS 二次保存、盘盈行可编辑 | 后端 + 前端 `buildRowsFromExpected` |
| 多人草稿串数据 | Mongo 唯一键 `{task_id, asset_no, operator_id}` |
| refetch 冲掉未保存编辑 | 合并草稿时间戳，避免全量覆盖 |
| **重进盘点页数据空白** | `rowsReady` 门控，草稿合并后再挂载表格 |
| **Univer 不可编辑 / 增行后表格消失** | **改回 `InventorySpreadsheet`**（Ant Design Table） |

### 2.4 Univer 状态

- 依赖已安装（`@univerjs/*@0.5.5`），`UniverSpreadsheet.tsx` 保留为实验实现
- **盘点详情页当前使用 `InventorySpreadsheet`**，与 submit API 完全兼容
- Univer 已知问题：依赖 `rows.length` 整实例重建；无公式栏时难以 inline 编辑

### 2.5 报表

- 导出类型：资产清单 / 工单日志 / 盘点差异（差异 Tab 选中任务）
- 差异明细 `pageSize` 上限提至 500

---

## 3. 测试

```bash
# 前端
cd frontend && npm test && npm run build

# 后端单元
cd backend && go test ./pkg/... -short

# 院级资产（示例）
# admin_info 登录 → GET /asset/assets 应含 dept 15/103/104 资产
```

---

## 4. 仍待完善（非阻塞）

| 项 | 说明 |
|---|---|
| Univer 生产接入 | 需解决增行重建、列级只读、inline 编辑 |
| asset-api RequireDeptScope | 可改为正式中间件，减少 handler 内 fallback |
| 报表快照表 | `rpt_asset_daily_snapshot` 需 Kafka 消费者填充后方院级统计更准确 |
| Redis JWT 黑名单 | 中间件 Redis 传 nil 时登出黑名单未生效 |

---

*文档版本：v1.0 | 2026-07-09*
