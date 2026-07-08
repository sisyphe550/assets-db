# P1-P2 开发日志

- **分支**：`feat/frontend-p1-p2-foundation`
- **日期**：2026-07-08
- **合并前提**：`npm test && npm run build` 全部通过

## 实现范围

### P1 脚手架与认证
- Vite + React 18 + TypeScript + Ant Design 5 项目脚手架
- Redux Toolkit Store：`authSlice`、`uiSlice`
- RTK Query：`authApi`（login / logout / getMe）
- `baseQueryWithReauth`：401 自动 refresh，失败跳转登录
- `unwrapApiResponse` 统一解包 `{ code, message, data }`
- Token 本地存储（`localStorage`）
- 登录页：表单校验、错误提示、按 `roleLevel` 跳转首页

### P2 Layout 与路由
- 三角色 Layout：`AdminLayout` / `CollegeLayout` / `UserLayout`
- 公共壳：`AppShell`（顶栏 + 可折叠侧边栏 + 退出登录）
- 侧边栏菜单配置（`config/menu.ts` + 图标解析）
- `RequireAuth` 路由守卫（未登录、禁用账户、角色越权）
- 完整路由树（`App.tsx`），未实现页面使用 `PlaceholderPage` 占位
- 仪表盘页（admin / college）与 404 页

## 测试

```bash
cd frontend
npm install
npm test
npm run build
```

| 测试文件 | 覆盖点 |
|---|---|
| `storage.test.ts` | Token 读写与清除 |
| `api.test.ts` | `unwrapApiResponse` 成功/失败 |
| `authSlice.test.ts` | 登录态 reducer |
| `LoginPage.test.tsx` | 登录表单渲染、MSW 模拟成功/失败 |

**结果**：2026-07-08 全部通过（11 tests, 4 files）；`npm run build` 成功。

## 手动验证步骤

1. 启动后端 5 个 API 服务（端口 8888–8892）
2. `cd frontend && npm run dev`，访问 http://localhost:5173
3. 使用 `admin_school` / `Test@123456` 登录 → 跳转 `/admin/dashboard`
4. 使用 `admin_info` / `Test@123456` 登录 → 跳转 `/college/dashboard`
5. 使用 `student_001` / `Test@123456` 登录 → 跳转 `/user/assets`
6. 侧边栏导航可进入各占位页；退出登录回到 `/login`
7. 未登录直接访问 `/admin/dashboard` → 重定向 `/login`

## 已知限制 / 后续事项

- P3–P7 业务页面均为占位，待各阶段分支实现
- 用户端盘点动态菜单（`UserLayout` 按任务注入）留待 P5
- Charts / Univer 依赖留待 P5–P6 引入
- E2E 浏览器自动化暂未接入，阶段验收对照后端 E2E + 手动验证

---

*记录人：AI 开发助手 | v1.0*
