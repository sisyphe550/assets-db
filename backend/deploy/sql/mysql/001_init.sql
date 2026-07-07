-- ==============================
-- FAMS MySQL 初始化 DDL
-- 库：fams_asset
-- 包含：资产台账、事件去重表
-- 幂等：使用 IF NOT EXISTS
-- ==============================

-- ==============================
-- 1. 固定资产台账表
-- ==============================
CREATE TABLE IF NOT EXISTS asset_ledger (
    id              BIGINT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    asset_no        VARCHAR(50)     NOT NULL,
    name            VARCHAR(100)    NOT NULL,
    category        VARCHAR(50)     NOT NULL,
    price           DECIMAL(10,2)   NOT NULL DEFAULT 0.00,
    purchase_time   DATETIME        NOT NULL,
    location        VARCHAR(100)    NOT NULL,
    department_id   BIGINT          NOT NULL,
    user_id         BIGINT          NULL,
    is_shared       TINYINT         NOT NULL DEFAULT 0,
    status          TINYINT         NOT NULL DEFAULT 1,
    deleted_at      DATETIME        NULL,
    created_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT uk_asset_no UNIQUE (asset_no),
    INDEX idx_asset_dept (department_id),
    INDEX idx_asset_status (status),
    INDEX idx_asset_deleted (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ==============================
-- 2. 事件消费去重表
-- ==============================
CREATE TABLE IF NOT EXISTS asset_event_dedup (
    id           BIGINT          NOT NULL AUTO_INCREMENT PRIMARY KEY,
    request_id   BIGINT          NOT NULL,
    event_type   VARCHAR(50)     NOT NULL,
    processed_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_dedup_request UNIQUE (request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
