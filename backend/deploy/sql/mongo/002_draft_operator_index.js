// 迁移：草稿唯一键加入 operator_id，支持多人同任务独立草稿
db = db.getSiblingDB("fams_inventory");

try {
    db.inventory_draft.dropIndex("uk_task_asset");
    print("dropped uk_task_asset");
} catch (e) {
    print("skip drop uk_task_asset:", e);
}

db.inventory_draft.createIndex(
    { task_id: 1, asset_no: 1, operator_id: 1 },
    { unique: true, name: "uk_task_asset_operator" }
);
print("created uk_task_asset_operator");
