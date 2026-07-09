// Package logic 资产服务共享业务逻辑（供 RPC、API、Consumer 复用）
package logic

import (
	"context"
	"database/sql"
	"strings"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/service/asset/model"
)

// CheckAssetForWorkflow 按工单类型校验资产状态（04-workflow-rules.md §2）
func CheckAssetForWorkflow(ctx context.Context, db *sql.DB, assetID int64, workflowType int32, requesterID int64) (ok bool, deptID int64, rejectReason string) {
	m := model.NewAssetModel(db)
	a, err := m.FindByID(ctx, assetID)
	if err != nil {
		return false, 0, "资产不存在"
	}

	switch workflowType {
	case 1:
		if a.Status != 1 { return false, a.DepartmentID, "资产当前不可领用" }
		if a.UserID != nil { return false, a.DepartmentID, "资产已被领用" }
	case 2:
		if a.Status != 2 { return false, a.DepartmentID, "资产当前不在领用状态" }
		if a.UserID == nil || *a.UserID != requesterID { return false, a.DepartmentID, "您不是该资产当前领用人，无法归还" }
	case 3:
		if a.Status != 1 && a.Status != 2 { return false, a.DepartmentID, "资产当前不可报修" }
	case 4:
		if a.Status != 1 && a.Status != 3 { return false, a.DepartmentID, "资产当前不可报废" }
	default:
		return false, 0, "未知工单类型"
	}
	return true, a.DepartmentID, ""
}

// ChangeAssetStatus 变更资产状态（状态机约束）
func ChangeAssetStatus(ctx context.Context, db *sql.DB, assetID int64, targetStatus int8, userID *int64) error {
	valid := map[int8][]int8{
		1: {2, 3, 4}, 2: {1, 3}, 3: {1, 2, 4}, 4: {},
	}
	m := model.NewAssetModel(db)
	a, err := m.FindByID(ctx, assetID)
	if err != nil { return err }
	allowed, ok := valid[a.Status]
	if !ok { return errx.ErrInvalidState }
	for _, s := range allowed {
		if s == targetStatus { return m.ChangeStatus(ctx, assetID, targetStatus, userID) }
	}
	return errx.ErrInvalidState
}

// GetAsset 获取资产详情
func GetAsset(ctx context.Context, db *sql.DB, id int64) (*model.AssetLedger, error) {
	return model.NewAssetModel(db).FindByID(ctx, id)
}

// ListByDeptIDs 按部门子树查询资产
func ListByDeptIDs(ctx context.Context, db *sql.DB, deptIDs []int64) ([]model.AssetLedger, error) {
	list, _, err := model.NewAssetModel(db).List(ctx, model.AssetListFilter{
		DeptIDs: deptIDs, Page: 1, PageSize: 10000,
	})
	return list, err
}

// DedupEvent 消费去重，返回 true 表示重复
func DedupEvent(ctx context.Context, db *sql.DB, requestID int64, eventType string) (bool, error) {
	_, err := db.ExecContext(ctx,
		`INSERT INTO asset_event_dedup (request_id, event_type, processed_at) VALUES (?, ?, NOW())`, requestID, eventType)
	if err != nil {
		if isMySQLDup(err) { return true, nil }
		return false, err
	}
	return false, nil
}

// CheckAssetScope 校验资产是否在盘点 scope 部门子树内
func CheckAssetScope(ctx context.Context, db *sql.DB, assetNo string, scopeDeptIDs []int64) (bool, error) {
	if strings.HasPrefix(assetNo, "UNKNOWN-") {
		return true, nil
	}
	var deptID int64
	err := db.QueryRowContext(ctx,
		`SELECT department_id FROM asset_ledger WHERE asset_no=? AND deleted_at IS NULL`, assetNo).Scan(&deptID)
	if err == sql.ErrNoRows {
		return true, nil // 未知 asset_no → 可能是盘盈，允许提交
	}
	if err != nil {
		return false, err
	}
	for _, id := range scopeDeptIDs {
		if id == deptID {
			return true, nil
		}
	}
	return false, nil
}

func isMySQLDup(err error) bool {
	if err == nil { return false }
	msg := err.Error()
	return contains(msg, "Duplicate entry") || contains(msg, "1062")
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}
