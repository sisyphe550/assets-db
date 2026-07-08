# P4 开发日志

- **分支**：`feat/frontend-p4-workflow`
- **日期**：2026-07-08
- **合并前提**：`npm test && npm run build` 全部通过

## 实现范围

### API 层
- `workflowApi`：列表/详情/创建/同意/驳回
- 终审通过后通过 `assetApi.util.invalidateTags` 刷新资产缓存

### 组件
- `StatusTag` 扩展：工单类型/状态/阶段
- `WorkflowTable`：ProTable 列表 + Drawer 详情入口
- `WorkflowDetail`：工单信息、关联资产、审批时间线、同意/驳回操作
- `utils/workflow.ts`：`canActOnWorkflow`、资产筛选逻辑

### 页面
| 路由 | 说明 |
|---|---|
| `/admin/workflow/todo` | 校级待复审 |
| `/admin/workflow/all` | 校级全部工单 |
| `/college/workflow/todo` | 院级待初审 |
| `/user/workflow/my` | 我的申请 |
| `/user/workflow/create` | 新建申请（支持 `?type=&assetId=` 预填） |

## 测试

```bash
cd frontend && npm test && npm run build
```

| 测试文件 | 覆盖点 |
|---|---|
| `workflow.test.ts` | 审批权限判断、资产类型筛选 |
| `StatusTag.test.tsx` | 工单类型标签 |

**结果**：2026-07-08 — 23 tests 全绿；build 成功。

## 手动验证步骤

1. 启动 workflow(8890) + asset(8889) + user(8888)
2. `student_001` → 新建领用申请 → `/user/workflow/my` 可见
3. `admin_info` → `/college/workflow/todo` 院级初审同意
4. `admin_school` → `/admin/workflow/todo` 校级复审同意
5. 资产状态应随审批更新（异步 Kafka，稍等刷新）

## 已知限制

- 列表仅显示申请人/操作人 ID，无姓名（后端未返回）
- 师生端无资产详情路由，工单中资产仅展示信息卡片

---

*记录人：AI 开发助手 | v1.0*
