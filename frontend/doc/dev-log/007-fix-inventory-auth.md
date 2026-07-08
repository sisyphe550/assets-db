# 007 — 盘点权限与 P1 前端修复

## 分支

`fix/frontend-p0-inventory-auth` → 合并 `main`

## P0

- **UserLayout**：`useGetTasksQuery` 增加 `scope: 'assigned'`，师生侧栏仅展示指派任务
- **InventoryTaskDetailView**：40302/40303 展示 403；非执行人提交前校验 `assigneeIds`

## P1 盘点

- **inventory.ts**：`rowHasEdits()` — 未改位置/备注的行不提交；支持 `expectedUpdatedAt` CAS
- 执行页 `actualLocation` 默认空（不预填账面位置）
- 归档 `Modal.confirm` + 轮询至 `status=3`；`status=2` 显示「比对中」

## P1 权限与菜单

- 院级 **用户管理**：`/college/users`、`UserManagePage`
- 校级菜单 **全部工单**；院级菜单 **用户管理**
- **AssetDetailView**：院级仅子树内可编辑；删除按钮仅 `role=1`

## P1 报表

- **ReportCharts** 差异 Tab：增加盘点记录明细表（`getRecords`）

## 工具

- **api.ts**：`getApiErrorCode()` 供 403 分支判断

## 测试

- `inventory.test.ts`：`rowHasEdits` 4 项
- `api.test.ts`：`getApiErrorCode`
