# 009 — 图表布局与盘点比对卡住修复

## 问题

1. 报表饼图/柱状图位置偏移，饼图扇区显示 `undefined`
2. 盘点任务归档后一直停在「比对中」（status=2）

## 根因

- G2 图表在 flex 容器内未约束宽度，且 Pie `label` 未指定 `text` 字段
- 归档后 `sendComparisonTask` 仅打日志，未执行比对；comparison-worker 依赖 Kafka 且 Scan 字段错位

## 修复

### 后端
- 新增 `pkg/inventorycmp` 统一比对逻辑（兼容 asset-rpc PascalCase）
- 归档后 goroutine 自动执行比对
- 新增 `POST /inventory/tasks/:id/compare` 手动/补偿触发

### 前端
- `ChartBox` 约束图表容器；Pie/Column 加 `autoFit`，关闭错误 label
- `status=2` 时自动调用 `compareTask` + 轮询至完成
