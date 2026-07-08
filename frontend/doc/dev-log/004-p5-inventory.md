# P5 开发日志

- **分支**：`feat/frontend-p5-inventory`
- **日期**：2026-07-08
- **合并前提**：`npm test && npm run build` 全部通过

## 实现范围

### API 层
- `inventoryApi`：任务列表/详情/创建/归档、预期资产、草稿提交、盘点记录
- `userApi.listUsers`：创建任务时选择盘点员
- `createTask` 响应 `taskId` → 前端映射为 `id`

### 类型与工具
- `InventoryTask`、`ExpectedAsset`、`SubmitItem`、`SubmitResult`、`InventoryRecord` 等类型
- `INVENTORY_STATUS_MAP`、`INVENTORY_DIFF_MAP` 常量
- `utils/inventory.ts`：`buildSubmitItems`、`applySubmitResult`（冲突/成功行状态）

### 组件
- `StatusTag` 扩展：`inventory`、`inventoryDiff` 类型
- `InventoryTaskTable`：任务列表 + 归档操作
- `InventoryTaskForm`：创建任务（部门树 + 盘点员多选）
- `InventoryTaskDetailView`：详情、预期资产加载、草稿提交、差异报告
- `InventorySpreadsheet`：**Ant Design 可编辑 Table**（非 Univer SDK），冲突行标红、成功行标绿

### 页面
| 路由 | 说明 |
|---|---|
| `/admin/inventory/tasks` | 校级任务列表 |
| `/admin/inventory/tasks/create` | 创建盘点任务 |
| `/admin/inventory/tasks/:id` | 任务详情 + 归档 |
| `/college/inventory/tasks` | 院级任务列表 |
| `/college/inventory/tasks/create` | 创建盘点任务 |
| `/college/inventory/tasks/:id` | 任务详情 + 归档 |
| `/user/inventory/:taskId` | 师生提交盘点草稿 |

### Layout
- `UserLayout`：动态注入进行中盘点任务菜单（`GET /inventory/tasks?status=1`）

## 测试

```bash
cd frontend && npm test && npm run build
```

| 测试文件 | 覆盖点 |
|---|---|
| `inventory.test.ts` | 提交项构建、冲突/成功行标记 |
| `StatusTag.test.tsx` | 盘点状态、差异类型标签 |

**结果**：2026-07-08 — 26 tests 全绿；build 成功。

## 手动验证步骤

1. 启动 inventory(8891) + asset(8889) + user(8888)
2. `admin_school` → `/admin/inventory/tasks/create` 创建任务，指派盘点员
3. `student_001` → 侧边栏出现进行中任务 → 编辑实际位置 → 提交草稿
4. 再次提交同一资产应看到冲突行标红
5. 管理员 → 任务详情 → 归档 → 查看差异报告

## 已知限制

- **Univer SDK 未集成**：文档要求 Univer 0.5.5，当前用 Ant Design 可编辑 Table 实现同等 submit API 交互，可后续替换
- 盘点员列表依赖 `listUsers`，大量用户时分页未做虚拟滚动
- 差异报告为只读表格，无导出

---

*记录人：AI 开发助手 | v1.0*
