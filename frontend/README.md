# FAMS 前端

高校固定资产管理系统（Fixed Assets Management System）前端。

## 技术栈（已定稿）

| 层级 | 选择 |
|---|---|
| 框架 | React 18 + TypeScript |
| 构建 | Vite |
| UI 库 | Ant Design 5 |
| 状态管理 | Redux Toolkit + RTK Query |
| 路由 | React Router v6（三个角色分开 Layout） |
| 表格 | Univer（盘点模块） |
| 后端对接 | 直连（Vite proxy），MSW 按需补充 |

## 设计文档

| 文档 | 内容 |
|---|---|
| `doc/01-design.md` | 技术选型、7 项决策记录、状态管理设计、Store 划分、路由树、TypeScript 类型 |
| `doc/02-directory.md` | 完整目录树、pages vs components 分界、Layout 差异、Univer 封装设计、构建流程 |
| `doc/03-pages.md` | 21 个页面详细交互设计、审批按钮逻辑、盘点三阶段、跨页面数据流、错误处理策略 |
| `doc/04-backend-api.md` | 后端 API 参考：全部 5 服务端点、鉴权流程、错误码、业务枚举、关键业务规则 |
| `doc/05-components.md` | 组件规格：Props/loading-empty-error 三态/表单校验规则汇总（含 15 个共享组件） |
| `doc/06-visual-design.md` | **美术设计规范**：品牌标识、Design Tokens、图标系统、全页面线框图、图表/空状态/交互状态 |
| `doc/07-implementation.md` | **AI 开发蓝图**：package.json、vite 配置、完整路由树、RTK Query 端点、权限矩阵、开发顺序 |

> 按 `07-implementation.md` §10 推荐顺序开发，配合 `06-visual-design.md` 还原视觉。

## 快速开始

```bash
cd frontend/
npm install
npm run dev       # http://localhost:5173
                  # API 请求通过 vite.config.ts proxy 转发到后端
```

## 后端文档

后端完整文档见 `../backend/doc/`，前端开发主要参考：
- `../backend/doc/03-api-contract.md` — API 契约（权威来源）
- `../backend/doc/06-error-codes.md` — 完整错误码（27 个）
- `../backend/doc/05-seed-fixtures.md` — 测试账号与固定数据
- `../backend/doc/10-final-status.md` — 后端完成状态（95%+）
