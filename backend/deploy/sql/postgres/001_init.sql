-- ==============================
-- FAMS PostgreSQL 初始化 DDL
-- 库：fams_core
-- 包含：组织架构、用户、工作流、盘点 全部业务表
-- 幂等：使用 IF NOT EXISTS
-- ==============================

-- ==============================
-- 1. 组织架构表
-- ==============================
CREATE TABLE IF NOT EXISTS sys_department (
    id          BIGINT          NOT NULL PRIMARY KEY,
    parent_id   BIGINT          NOT NULL DEFAULT 0,
    dept_name   VARCHAR(100)    NOT NULL,
    dept_code   VARCHAR(30)     NOT NULL,
    path        VARCHAR(255)    NOT NULL,
    sort_order  INT             NOT NULL DEFAULT 0,
    CONSTRAINT uk_dept_code UNIQUE (dept_code)
);

CREATE INDEX IF NOT EXISTS idx_dept_parent ON sys_department (parent_id);
CREATE INDEX IF NOT EXISTS idx_dept_path   ON sys_department (path);

-- ==============================
-- 2. 用户表
-- ==============================
CREATE TABLE IF NOT EXISTS sys_user (
    id              BIGINT          NOT NULL PRIMARY KEY,
    username        VARCHAR(50)     NOT NULL,
    password_hash   VARCHAR(255)    NOT NULL,
    real_name       VARCHAR(50)     NOT NULL,
    role_level      SMALLINT        NOT NULL CHECK (role_level IN (1, 2, 3)),
    department_id   BIGINT          NOT NULL,
    status          SMALLINT        NOT NULL DEFAULT 1 CHECK (status IN (0, 1)),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    CONSTRAINT uk_username UNIQUE (username)
);

CREATE INDEX IF NOT EXISTS idx_user_dept ON sys_user (department_id);
CREATE INDEX IF NOT EXISTS idx_user_role ON sys_user (role_level);

-- ==============================
-- 3. 审批申请主表
-- ==============================
CREATE TABLE IF NOT EXISTS workflow_request (
    id              BIGINT          NOT NULL PRIMARY KEY,
    asset_id        BIGINT          NOT NULL,
    requester_id    BIGINT          NOT NULL,
    department_id   BIGINT          NOT NULL,
    type            SMALLINT        NOT NULL CHECK (type IN (1, 2, 3, 4)),
    current_stage   SMALLINT        NOT NULL DEFAULT 1 CHECK (current_stage IN (1, 2, 3)),
    status          SMALLINT        NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2, 3)),
    reason          VARCHAR(255),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

-- 防重复：同一资产在 status=1 时只能有一张工单
CREATE UNIQUE INDEX IF NOT EXISTS uk_asset_open_request
    ON workflow_request (asset_id) WHERE status = 1;

CREATE INDEX IF NOT EXISTS idx_wf_dept   ON workflow_request (department_id);
CREATE INDEX IF NOT EXISTS idx_wf_status ON workflow_request (status, current_stage);

-- ==============================
-- 4. 审批留痕流水表
-- ==============================
CREATE TABLE IF NOT EXISTS workflow_log (
    id           BIGINT          NOT NULL PRIMARY KEY,
    request_id   BIGINT          NOT NULL,
    operator_id  BIGINT          NOT NULL,
    action       VARCHAR(50)     NOT NULL,
    comment      VARCHAR(255),
    operate_time TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_log_request ON workflow_log (request_id, operate_time);

-- ==============================
-- 5. 事务发件箱表（Transactional Outbox）
-- ==============================
CREATE TABLE IF NOT EXISTS workflow_outbox (
    id              BIGINT          NOT NULL PRIMARY KEY,
    event_type      VARCHAR(50)     NOT NULL,
    partition_key   VARCHAR(50)     NOT NULL,
    payload         JSONB           NOT NULL,
    status          SMALLINT        NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2)),
    retry_count     INT             NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON workflow_outbox (status, id);

-- ==============================
-- 6. 盘点任务主表
-- ==============================
CREATE TABLE IF NOT EXISTS inventory_task (
    id              BIGINT          NOT NULL PRIMARY KEY,
    task_name       VARCHAR(100)    NOT NULL,
    scope_dept_id   BIGINT          NOT NULL,
    creator_id      BIGINT          NOT NULL,
    start_time      TIMESTAMPTZ     NOT NULL,
    end_time        TIMESTAMPTZ     NOT NULL,
    status          SMALLINT        NOT NULL DEFAULT 1 CHECK (status IN (1, 2, 3)),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_task_status ON inventory_task (status);
CREATE INDEX IF NOT EXISTS idx_task_scope  ON inventory_task (scope_dept_id);

-- ==============================
-- 7. 盘点任务指派表
-- ==============================
CREATE TABLE IF NOT EXISTS inventory_task_assignee (
    id          BIGINT          NOT NULL PRIMARY KEY,
    task_id     BIGINT          NOT NULL,
    user_id     BIGINT          NOT NULL,
    assigned_by BIGINT          NOT NULL,
    assigned_at TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_task_user
    ON inventory_task_assignee (task_id, user_id);

-- ==============================
-- 8. 盘点任务条目表
-- ==============================
CREATE TABLE IF NOT EXISTS inventory_task_item (
    task_id     BIGINT          NOT NULL,
    asset_id    BIGINT          NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_task_item_task
    ON inventory_task_item (task_id);

-- ==============================
-- 9. 盘点明细比对结果表
-- ==============================
CREATE TABLE IF NOT EXISTS inventory_record (
    id               BIGINT          NOT NULL PRIMARY KEY,
    task_id          BIGINT          NOT NULL,
    asset_id         BIGINT,
    found_asset_desc VARCHAR(255),
    operator_id      BIGINT,
    is_scanned       SMALLINT        NOT NULL DEFAULT 0 CHECK (is_scanned IN (0, 1)),
    actual_location  VARCHAR(100),
    diff_status      SMALLINT        NOT NULL DEFAULT 0 CHECK (diff_status IN (0, 1, 2, 3))
);

-- 防重复：同一任务对同一账面资产只能归档一次
CREATE UNIQUE INDEX IF NOT EXISTS uk_task_asset
    ON inventory_record (task_id, asset_id) WHERE asset_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_record_task   ON inventory_record (task_id);
CREATE INDEX IF NOT EXISTS idx_record_diff   ON inventory_record (task_id, diff_status);

-- ==============================
-- Sequence 初始化（所有表 ID 手动管理，不依赖自增）
-- ==============================
-- 开发/测试阶段使用固定 ID（见 002_seed.sql）；
-- 生产环境由应用层 snowflake 或序列生成。
