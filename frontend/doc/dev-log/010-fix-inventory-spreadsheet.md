# 010 盘点表格与数据加载修复

- **日期**：2026-07-09
- **关联提交**：`fix(inventory): 恢复可编辑表格并修复重进页面数据丢失`

## 问题

1. **重进盘点详情页**：应盘资产与草稿内容空白，刷新后才显示
2. **Univer 表格**：单元格无法直接编辑；点击「添加盘盈行」后整表消失

## 根因

| 问题 | 根因 |
|---|---|
| 重进空白 | RTK Query 缓存命中后 `loading=false`，但 `rows` 仍在 `useEffect` 中合并；Univer/表格在空 `rows` 时即挂载 |
| 不可编辑 | Univer 0.5.5 关闭 toolbar/formulaBar 后 inline 输入体验差 |
| 增行消失 | `UniverSpreadsheet` 的 init `useEffect` 依赖 `rows.length`，增行触发整实例销毁重建 |

## 修复

1. `InventoryTaskDetailView`：增加 `rowsReady`，仅在 `buildRowsFromExpected` 完成后渲染表格；切换 `taskId` 时重置 hydration
2. 懒加载改回 **`InventorySpreadsheet`**（Ant Design 可编辑 Table）
3. 表格组件 `key={taskIdNum}` 保证每次进入任务为新实例
4. `UniverSpreadsheet.tsx` 保留于仓库，供后续单独迭代，**不接入主流程**

## 验证

1. `admin_info` → 盘点任务 → 填写实际位置 → 保存草稿
2. 返回列表再进入 → 数据应立即显示
3. 「添加盘盈行」→ 表格不消失，新行可编辑
4. `npm test`（44 项）通过

---

*记录人：AI 开发助手 | v1.0*
