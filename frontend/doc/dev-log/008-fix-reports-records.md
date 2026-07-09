# 008 — 报表、盘点记录与归档轮询修复

## 分支

`fix/frontend-p1-reports-inventory` → 合并 `main`

## 后端

### inventory-api
- **Records**：按契约返回 `{ list, page, pageSize, total }`，关联 asset-rpc 补全 `assetNo`/`name`/`bookLocation`
- 支持 `diffStatus` 查询参数
- **asset-rpc** 调用统一为 `deptIds` 数组

### report-api
- **by-dept**：快照空时回退 MySQL 实时统计
- **by-category**：新增 `GET /report/assets/by-category`（院级自动子树过滤）
- **export status**：返回 `errorMessage`

## 前端

- **reportApi**：`getAssetsByCategory`；导出状态含 `errorMessage`
- **ReportCharts**：类别 Tab 走 by-category API；差异明细仅展示盘盈/盘亏
- **DashboardOverview**：饼图走 by-category API
- **InventoryTaskDetailView**：`status=2` 自动轮询至完成（含刷新后进入比对中任务）
- **WorkflowTable**：支持 `?highlight=` 自动打开工单详情

## 测试

- `npm test` + `npm run build`
- 需重启 `inventory-api` 与 `report-api`
