# FAMS 固定 Seed 夹具（E2E / 集成测试）

> 密码统一明文：`Test@123456`  
> bcrypt hash（cost=10，**固定使用此值写入 seed SQL**）：

```
$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre
```

---

## 1. 组织架构（PostgreSQL `fams_core.sys_department`）

| id | parent_id | dept_name | dept_code | path |
| --- | --- | --- | --- | --- |
| 1 | 0 | 本校 | ROOT | `/1/` |
| 15 | 1 | 信息工程学院 | INFO | `/1/15/` |
| 103 | 15 | 软件工程实验室 | SE_LAB | `/1/15/103/` |
| 104 | 15 | 网络工程实验室 | NET_LAB | `/1/15/104/` |
| 20 | 1 | 机械工程学院 | ME | `/1/20/` |

---

## 2. 用户（PostgreSQL `fams_core.sys_user`）

| id | username | real_name | role_level | department_id | 用途 |
| --- | --- | --- | --- | --- | --- |
| 10001 | admin_school | 张校管 | 1 | 1 | E2E 校级终审 |
| 10002 | admin_info | 王院管 | 2 | 15 | E2E 院级初审、创建盘点 |
| 10003 | student_001 | 李同学 | 3 | 103 | E2E 领用申请、盘点员 |
| 10004 | student_002 | 赵同学 | 3 | 104 | E2E 盘点冲突 |
| 10005 | student_me | 周同学 | 3 | 20 | 跨学院越权测试 |

---

## 3. 资产（MySQL `fams_asset.asset_ledger`）

| id | asset_no | name | category | department_id | user_id | is_shared | status | 用途 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 501 | EQUIP-2026-0001 | 激光切割机 | 设备 | 15 | NULL | 0 | 1 | E2E 领用 |
| 502 | EQUIP-2026-0002 | 3D打印机 | 设备 | 15 | NULL | 1 | 1 | 共享资产 API |
| 503 | EQUIP-2026-0003 | 实验台 | 家具 | 103 | 10003 | 0 | 2 | E2E 归还 |
| 504 | EQUIP-2026-0004 | 示波器 | 设备 | 104 | NULL | 0 | 1 | 盘点 scope |
| 505 | EQUIP-2026-0005 | 车床 | 设备 | 20 | NULL | 0 | 1 | 越权测试 |

所有资产 `price=10000.00`, `purchase_time='2025-01-01'`, `location` 与实验室对应, `deleted_at=NULL`。

---

## 4. 预置工单（可选，集成测试自建）

默认 **不** seed 开放工单；集成测试运行时创建。

---

## 5. 预置盘点任务（可选）

E2E 运行时创建；固定 ID 约定（测试 insert 后使用）：

| 字段 | 值 |
| --- | --- |
| task_name | E2E-Inventory-2026 |
| scope_dept_id | 15 |
| creator_id | 10002 |
| assignee_ids | [10003, 10004] |

---

## 6. E2E 断言常量（Go 测试代码直接引用）

```go
const (
  SeedPassword     = "Test@123456"
  AdminSchoolUID   = int64(10001)
  AdminInfoUID     = int64(10002)
  Student001UID    = int64(10003)
  AssetForUseID    = int64(501)
  AssetForReturnID = int64(503)
  DeptInfoID       = int64(15)
)
```

---

## 7. Seed SQL 文件约定

| 文件 | 内容 |
| --- | --- |
| `deploy/sql/postgres/002_seed.sql` | §1 组织 + §2 用户 |
| `deploy/sql/mysql/002_seed.sql` | §3 资产 |
| `deploy/sql/postgres/003_report_init.sql` | 创建 `fams_report` 库表（4.2.11–4.2.12） |

**ID 显式 INSERT**（不用 auto_increment 漂移）：

```sql
INSERT INTO sys_department (id, parent_id, dept_name, dept_code, path, sort_order) VALUES
  (1, 0, '本校', 'ROOT', '/1/', 0),
  ...
ON CONFLICT (id) DO NOTHING;
```

MySQL：

```sql
INSERT INTO asset_ledger (id, asset_no, name, ...) VALUES
  (501, 'EQUIP-2026-0001', ...);
```

---

## 8. 报表 E2E 期望聚合（seed 静态）

对 `GET /report/assets/by-dept?date=today`（快照由 consumer 首次写入后）：

| department_id | total_count（≥） |
| --- | --- |
| 15 | 3（501,502,503 归属 15 或子部门；503 在 103 属 15 子树） |
| 20 | 1 |

精确值依赖 snapshot job；E2E 断言 `total_count >= N` 而非相等。

---

*文档版本：v1.0 | 2026-07-07*
