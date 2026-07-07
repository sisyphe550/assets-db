# FAMS 前端

高校固定资产管理系统（Fixed Assets Management System）前端。

## 技术栈

- React 18 + TypeScript
- Vite
- Ant Design 5
- Redux Toolkit + RTK Query
- Univer（盘点协同表格）

## 设计文档

| 文档 | 内容 |
|---|---|
| `doc/01-design.md` | 技术选型、状态管理设计、路由设计、类型定义、枚举映射 |
| `doc/02-directory.md` | 完整目录结构、各目录职责分工、Univer 封装说明 |
| `doc/03-pages.md` | 页面级详细设计、交互流程、错误处理策略、跨页面数据流 |

## 快速开始

```bash
cd frontend/
npm install
npm run dev       # http://localhost:5173
```

## 后端依赖

后端 API 文档见 `../backend/doc/`，特别是：
- `../backend/doc/03-api-contract.md` — API 契约
- `../backend/doc/11-frontend-handoff.md` — 前后端交接文档
