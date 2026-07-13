-- 盘点员草稿冲突表（归档后待管理员裁决）
CREATE TABLE IF NOT EXISTS inventory_assignee_conflict (
    id                       BIGINT       NOT NULL PRIMARY KEY,
    task_id                  BIGINT       NOT NULL,
    asset_no                 VARCHAR(50)  NOT NULL,
    asset_id                 BIGINT,
    status                   SMALLINT     NOT NULL DEFAULT 0 CHECK (status IN (0, 1)),
    candidates               JSONB        NOT NULL,
    resolved_source          SMALLINT     CHECK (resolved_source IN (1, 2)),
    resolved_operator_id     BIGINT,
    resolved_actual_location VARCHAR(100),
    resolved_notes           VARCHAR(255),
    resolved_by              BIGINT,
    resolved_at              TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_inv_conflict_task_asset
    ON inventory_assignee_conflict (task_id, asset_no);

CREATE INDEX IF NOT EXISTS idx_inv_conflict_task_status
    ON inventory_assignee_conflict (task_id, status);
