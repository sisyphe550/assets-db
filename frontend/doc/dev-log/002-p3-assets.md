# P3 开发日志

- **分支**：`feat/frontend-p3-assets`
- **日期**：2026-07-08
- **合并前提**：`npm test && npm run build` 全部通过

## 实现范围

### API 层
- `assetApi`：列表/详情/创建/更新/删除/共享资产
- `userApi`（最小）：`getDeptTree` 供创建表单选部门
- Store 注册双 API middleware

### 组件
- `StatusTag`：资产状态标签
- `AssetTable`：ProTable 列表 + 筛选（类别/状态/关键词）
- `AssetForm`：创建/编辑表单（校级 TreeSelect 部门，院级默认本院）
- `AssetDetailView`：详情 Descriptions + 编辑 Modal + 删除确认

### 页面
| 路由 | 页面 |
|---|---|
| `/admin/assets` | 校级资产列表 |
| `/admin/assets/create` | 新增资产 |
| `/admin/assets/:id` | 资产详情 |
| `/college/assets` | 院级资产列表 |
| `/college/assets/create` | 院级新增（复用创建页） |
| `/college/assets/:id` | 院级详情 |
| `/user/assets` | 我的资产（已领用 + 学院共享 Tabs） |

## 测试

```bash
cd frontend
npm test
npm run build
```

| 测试文件 | 覆盖点 |
|---|---|
| `StatusTag.test.tsx` | 状态标签渲染 |
| `format.test.ts` | 价格/日期格式化 |
| `MyAssetsPage.test.tsx` | 已领用列表与快捷操作按钮 |

**结果**：2026-07-08 — 16 tests 全绿；build 成功。

## 手动验证步骤

1. 启动后端 asset(8889) + user(8888) 服务
2. `admin_school` 登录 → `/admin/assets` 列表、新增、详情编辑删除
3. `admin_info` 登录 → `/college/assets` 本院资产 CRUD
4. `student_001` 登录 → `/user/assets` 查看已领用与共享资产 Tab

## 已知限制 / 后续事项

- 列表未展示部门/领用人姓名（后端仅返回 ID）
- 校级「导出」按钮留待报表阶段
- 工单快捷跳转（归还/报修）依赖 P4 创建页

---

*记录人：AI 开发助手 | v1.0*
