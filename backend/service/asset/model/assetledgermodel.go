package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// AssetLedger 资产台账
type AssetLedger struct {
	ID           int64      `db:"id"`
	AssetNo      string     `db:"asset_no"`
	Name         string     `db:"name"`
	Category     string     `db:"category"`
	Price        float64    `db:"price"`
	PurchaseTime time.Time  `db:"purchase_time"`
	Location     string     `db:"location"`
	DepartmentID int64      `db:"department_id"`
	UserID       *int64     `db:"user_id"`
	IsShared     int8       `db:"is_shared"`
	Status       int8       `db:"status"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type AssetModel struct{ db *sql.DB }

func NewAssetModel(db *sql.DB) *AssetModel { return &AssetModel{db: db} }

func (m *AssetModel) Insert(ctx context.Context, a *AssetLedger) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO asset_ledger (asset_no, name, category, price, purchase_time, location, department_id, user_id, is_shared, status)
		 VALUES (?,?,?,?,?,?,?,?,?,1)`,
		a.AssetNo, a.Name, a.Category, a.Price, a.PurchaseTime, a.Location,
		a.DepartmentID, a.UserID, a.IsShared)
	if err != nil {
		if isMySQLDup(err) {
			return errx.ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (m *AssetModel) FindByID(ctx context.Context, id int64) (*AssetLedger, error) {
	var a AssetLedger
	err := m.db.QueryRowContext(ctx,
		`SELECT id, asset_no, name, category, price, purchase_time, location, department_id, user_id, is_shared, status, deleted_at
		 FROM asset_ledger WHERE id = ? AND deleted_at IS NULL`, id).
		Scan(&a.ID, &a.AssetNo, &a.Name, &a.Category, &a.Price, &a.PurchaseTime,
			&a.Location, &a.DepartmentID, &a.UserID, &a.IsShared, &a.Status, &a.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, errx.ErrAssetNotFound
	}
	return &a, err
}

// AssetListFilter 资产列表查询条件
type AssetListFilter struct {
	DeptIDs    []int64
	Category   string
	Keyword    string
	Status     *int8
	UserID     *int64
	SharedOnly bool
	Page       int
	PageSize   int
}

func (m *AssetModel) List(ctx context.Context, f AssetListFilter) ([]AssetLedger, int, error) {
	query := `SELECT id, asset_no, name, category, price, purchase_time, location, department_id, user_id, is_shared, status FROM asset_ledger WHERE deleted_at IS NULL`
	countQuery := `SELECT COUNT(*) FROM asset_ledger WHERE deleted_at IS NULL`
	var args []any

	if len(f.DeptIDs) > 0 {
		placeholders := "?"
		for i := 1; i < len(f.DeptIDs); i++ {
			placeholders += ",?"
		}
		cond := " AND department_id IN (" + placeholders + ")"
		query += cond
		countQuery += cond
		for _, id := range f.DeptIDs {
			args = append(args, id)
		}
	}
	if f.SharedOnly {
		query += " AND is_shared = 1"
		countQuery += " AND is_shared = 1"
	}
	if f.Category != "" {
		query += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, f.Category)
	}
	if f.Status != nil {
		query += " AND status = ?"
		countQuery += " AND status = ?"
		args = append(args, *f.Status)
	}
	if f.UserID != nil {
		query += " AND user_id = ?"
		countQuery += " AND user_id = ?"
		args = append(args, *f.UserID)
	}
	if f.Keyword != "" {
		query += " AND (name LIKE ? OR asset_no LIKE ?)"
		countQuery += " AND (name LIKE ? OR asset_no LIKE ?)"
		kw := "%" + f.Keyword + "%"
		args = append(args, kw, kw)
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := m.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	query += " LIMIT ? OFFSET ?"
	args = append(args, f.PageSize, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []AssetLedger
	for rows.Next() {
		var a AssetLedger
		if err := rows.Scan(&a.ID, &a.AssetNo, &a.Name, &a.Category, &a.Price, &a.PurchaseTime,
			&a.Location, &a.DepartmentID, &a.UserID, &a.IsShared, &a.Status); err != nil {
			return nil, 0, err
		}
		list = append(list, a)
	}
	return list, total, rows.Err()
}

func (m *AssetModel) Update(ctx context.Context, id int64, name, category, location string, deptID int64, isShared int8) error {
	result, err := m.db.ExecContext(ctx,
		`UPDATE asset_ledger SET name=?, category=?, location=?, department_id=?, is_shared=?, updated_at=NOW()
		 WHERE id=? AND deleted_at IS NULL`, name, category, location, deptID, isShared, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrAssetNotFound
	}
	return nil
}

func (m *AssetModel) SoftDelete(ctx context.Context, id int64) error {
	result, err := m.db.ExecContext(ctx,
		`UPDATE asset_ledger SET deleted_at=NOW(), updated_at=NOW() WHERE id=? AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrAssetNotFound
	}
	return nil
}

func (m *AssetModel) ChangeStatus(ctx context.Context, id int64, targetStatus int8, userID *int64) error {
	var uid any
	if userID != nil {
		uid = *userID
	}
	result, err := m.db.ExecContext(ctx,
		`UPDATE asset_ledger SET status=?, user_id=?, updated_at=NOW() WHERE id=?`, targetStatus, uid, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrAssetNotFound
	}
	return nil
}

func (m *AssetModel) HasOpenWorkflow(ctx context.Context, id int64) (bool, error) {
	// 跨库查询：由调用方通过 asset-rpc → workflow-rpc 实现
	// 此处留空，实际由 API handler 调用 workflow-rpc
	return false, nil
}

func isMySQLDup(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsStr(msg, "Duplicate entry") || containsStr(msg, "1062")
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
