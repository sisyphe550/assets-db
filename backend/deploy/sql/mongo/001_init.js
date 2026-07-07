// ==============================
// FAMS MongoDB 初始化脚本
// 库：fams_inventory
// 集合：inventory_draft
// ==============================

db = db.getSiblingDB("fams_inventory");

// ---- 创建盘点草稿集合 ----
db.createCollection("inventory_draft");

// ---- 复合唯一索引：同一任务同一资产编号仅一条草稿 ----
db.inventory_draft.createIndex(
    { task_id: 1, asset_no: 1 },
    { unique: true, name: "uk_task_asset" }
);

// ---- 查询索引：按任务 + 更新时间 ----
db.inventory_draft.createIndex(
    { task_id: 1, updated_at: 1 },
    { name: "idx_task_updated" }
);

// ---- 查询索引：按操作员 ----
db.inventory_draft.createIndex(
    { operator_id: 1 },
    { name: "idx_operator" }
);

print("MongoDB fams_inventory initialized: inventory_draft collection with indexes");
