-- ==============================
-- FAMS PostgreSQL Seed 数据
-- 所有 ID 固定，供 E2E / 集成测试断言
-- ==============================

-- ==============================
-- 1. 组织架构（sys_department）
-- ==============================
INSERT INTO sys_department (id, parent_id, dept_name, dept_code, path, sort_order) VALUES
    (1,   0,  '本校',           'ROOT',    '/1/',        0),
    (15,  1,  '信息工程学院',   'INFO',    '/1/15/',     10),
    (103, 15, '软件工程实验室', 'SE_LAB',  '/1/15/103/', 1),
    (104, 15, '网络工程实验室', 'NET_LAB', '/1/15/104/', 2),
    (20,  1,  '机械工程学院',   'ME',      '/1/20/',     20)
ON CONFLICT (id) DO NOTHING;

-- ==============================
-- 2. 用户（sys_user）
-- 密码明文：Test@123456
-- bcrypt hash (cost=10)：
-- $2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre
-- ==============================
INSERT INTO sys_user (id, username, password_hash, real_name, role_level, department_id, status) VALUES
    (10001, 'admin_school', '$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre', '张校管', 1, 1,   1),
    (10002, 'admin_info',   '$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre', '王院管', 2, 15,  1),
    (10003, 'student_001',  '$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre', '李同学', 3, 103, 1),
    (10004, 'student_002',  '$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre', '赵同学', 3, 104, 1),
    (10005, 'student_me',   '$2a$10$BuwmRTR/mblICvn4jrLA0.9LVPRTEMdlQZBa7rMnttf6mieWX9pre', '周同学', 3, 20,  1)
ON CONFLICT (id) DO NOTHING;
