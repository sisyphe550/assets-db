-- 盘点任务条目：管理员先配置资产条目，再发布给盘点员执行。

ALTER TABLE inventory_task
    DROP CONSTRAINT IF EXISTS inventory_task_status_check;

ALTER TABLE inventory_task
    ADD CONSTRAINT inventory_task_status_check CHECK (status IN (0, 1, 2, 3));

ALTER TABLE inventory_task
    ALTER COLUMN status SET DEFAULT 0;

CREATE TABLE IF NOT EXISTS inventory_task_item (
    task_id     BIGINT      NOT NULL,
    asset_id    BIGINT      NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, asset_id)
);

CREATE INDEX IF NOT EXISTS idx_task_item_task
    ON inventory_task_item (task_id);
