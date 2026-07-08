# FAMS 前端开发流程规范

> 企业团队级开发流程：分支管理 → 实现 → 测试 → 文档 → 合并

---

## 1. 阶段划分

与 `07-implementation.md` §10 对齐，每阶段独立分支、独立测试、独立合并：

| 阶段 | 分支命名 | 内容 | 合并前提 |
|---|---|---|---|
| P1-P2 | `feat/frontend-p1-p2-foundation` | 脚手架、登录、Layout、路由守卫 | `npm test` + `npm run build` 通过 |
| P3 | `feat/frontend-p3-assets` | 资产 CRUD | 测试通过 + 手动验证资产列表 |
| P4 | `feat/frontend-p4-workflow` | 工单审批 | 测试通过 + 领用流程可走通 |
| P5 | `feat/frontend-p5-inventory` | 盘点 + Univer | 测试通过 + 草稿提交 |
| P6 | `feat/frontend-p6-reports` | 报表导出 | 测试通过 |
| P7 | `feat/frontend-p7-admin` | 组织树 + 用户管理 | 测试通过 |

---

## 2. Git 分支规范

```
main
 └── feat/frontend-p{N}-{简述}    # 功能分支，从 main 拉出
```

**规则**：
1. 每个阶段从最新 `main` 创建分支
2. 提交信息格式：`feat(frontend): P{N} 简述` 或 `test(frontend): ...` / `docs(frontend): ...`
3. **禁止**直接 push 到 `main`（除 hotfix）
4. 合并前必须：测试全绿 + 开发日志已写
5. 合并方式：Fast-forward 或 Squash merge（本仓库使用 merge commit 保留阶段历史）

---

## 3. 测试策略

| 层级 | 工具 | 命令 | 何时运行 |
|---|---|---|---|
| 单元测试 | Vitest | `npm test` | 每次提交前、CI、合并前 |
| 集成测试 | Vitest + RTL + MSW | `npm test` | 组件与 API mock 联调 |
| 构建验证 | tsc + Vite | `npm run build` | 合并前 |
| E2E（可选） | 后端 E2E + 手动 | 见 dev-log | 阶段验收时对照后端 |

**合并门禁**：`npm test && npm run build` 必须 exit 0。

---

## 4. 文档要求

每阶段完成后在 `frontend/doc/dev-log/` 追加日志：

```
NNN-p{N}-{简述}.md
```

日志模板：

```markdown
# P{N} 开发日志

- 分支：feat/frontend-p{N}-...
- 日期：YYYY-MM-DD
- 实现范围：（列表）
- 测试命令与结果
- 手动验证步骤
- 已知限制 / 后续事项
```

---

## 5. 本地开发

```bash
# 1. 启动后端（另开终端）
cd backend && make infra-up  # 如未启动
# 启动 5 个 API 进程（见 backend/doc/11-frontend-handoff.md）

# 2. 前端
cd frontend
npm install
npm run dev        # http://localhost:5173

# 3. 测试
npm test
npm run build
```

---

## 6. 目录与 dev-log 索引

| 文档 | 说明 |
|---|---|
| `doc/08-dev-process.md` | 本文档 |
| `doc/dev-log/001-p1-p2-foundation.md` | P1-P2 实现记录 |
| `doc/07-implementation.md` | 技术蓝图 |

---

*文档版本：v1.0 | 2026-07-08*
