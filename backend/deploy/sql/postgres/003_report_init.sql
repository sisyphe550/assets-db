-- ==============================
-- FAMS PostgreSQL Report 独立库 DDL
-- 库：fams_report
-- 包含：聚合宽表、导出任务表
-- ==============================

-- ==============================
-- 1. 按日按部门资产快照
-- ==============================
CREATE TABLE IF NOT EXISTS rpt_asset_daily_snapshot (
    id              BIGINT          NOT NULL PRIMARY KEY,
    snapshot_date   DATE            NOT NULL,
    department_id   BIGINT          NOT NULL,
    total_count     INT             NOT NULL DEFAULT 0,
    in_stock_count  INT             NOT NULL DEFAULT 0,
    in_use_count    INT             NOT NULL DEFAULT 0,
    repair_count    INT             NOT NULL DEFAULT 0,
    scrap_count     INT             NOT NULL DEFAULT 0,
    total_value     DECIMAL(14,2)   NOT NULL DEFAULT 0.00
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_snapshot_date_dept
    ON rpt_asset_daily_snapshot (snapshot_date, department_id);

-- ==============================
-- 2. 按日按类型工单汇总
-- ==============================
CREATE TABLE IF NOT EXISTS rpt_workflow_summary (
    id              BIGINT          NOT NULL PRIMARY KEY,
    summary_date    DATE            NOT NULL,
    request_type    SMALLINT        NOT NULL CHECK (request_type IN (1, 2, 3, 4)),
    approved_count  INT             NOT NULL DEFAULT 0,
    rejected_count  INT             NOT NULL DEFAULT 0,
    pending_count   INT             NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_summary_date_type
    ON rpt_workflow_summary (summary_date, request_type);

-- ==============================
-- 3. 盘点差异汇总
-- ==============================
CREATE TABLE IF NOT EXISTS rpt_inventory_diff_summary (
    id              BIGINT          NOT NULL PRIMARY KEY,
    task_id         BIGINT          NOT NULL,
    match_count     INT             NOT NULL DEFAULT 0,
    surplus_count   INT             NOT NULL DEFAULT 0,
    loss_count      INT             NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_diff_task ON rpt_inventory_diff_summary (task_id);

-- ==============================
-- 4. 报表导出任务表
-- ==============================
CREATE TABLE IF NOT EXISTS rpt_export_job (
    id             BIGINT          NOT NULL PRIMARY KEY,
    creator_id     BIGINT          NOT NULL,
    export_type    VARCHAR(50)     NOT NULL,
    params         JSONB           NOT NULL DEFAULT '{}',
    status         SMALLINT        NOT NULL DEFAULT 0 CHECK (status IN (0, 1, 2, 3)),
    file_path      VARCHAR(255),
    error_message  VARCHAR(255),
    created_at     TIMESTAMPTZ     NOT NULL DEFAULT now(),
    finished_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_export_creator ON rpt_export_job (creator_id, created_at DESC);
