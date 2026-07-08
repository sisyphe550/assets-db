package model

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

type InventoryTask struct {
	ID          int64     `db:"id"`
	TaskName    string    `db:"task_name"`
	ScopeDeptID int64     `db:"scope_dept_id"`
	CreatorID   int64     `db:"creator_id"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	Status      int16     `db:"status"`
}

type InventoryRecord struct {
	ID             int64  `db:"id"`
	TaskID         int64  `db:"task_id"`
	AssetID        *int64 `db:"asset_id"`
	FoundAssetDesc string `db:"found_asset_desc"`
	OperatorID     *int64 `db:"operator_id"`
	IsScanned      int16  `db:"is_scanned"`
	ActualLocation string `db:"actual_location"`
	DiffStatus     int16  `db:"diff_status"`
}

type InvModel struct{ db *sql.DB }

func NewInvModel(db *sql.DB) *InvModel { return &InvModel{db: db} }

func (m *InvModel) CreateTask(ctx context.Context, t *InventoryTask, assigneeIDs []int64) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	err = tx.QueryRowContext(ctx,
		`INSERT INTO inventory_task (task_name, scope_dept_id, creator_id, start_time, end_time, status)
		 VALUES ($1,$2,$3,$4,$5,1) RETURNING id`,
		t.TaskName, t.ScopeDeptID, t.CreatorID, t.StartTime, t.EndTime).Scan(&t.ID)
	if err != nil { return err }

	for _, uid := range assigneeIDs {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO inventory_task_assignee (task_id, user_id, assigned_by) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			t.ID, uid, t.CreatorID)
		if err != nil { return err }
	}
	return tx.Commit()
}

func (m *InvModel) FindTask(ctx context.Context, id int64) (*InventoryTask, error) {
	var t InventoryTask
	err := m.db.QueryRowContext(ctx,
		`SELECT id, task_name, scope_dept_id, creator_id, start_time, end_time, status
		 FROM inventory_task WHERE id=$1`, id).
		Scan(&t.ID, &t.TaskName, &t.ScopeDeptID, &t.CreatorID, &t.StartTime, &t.EndTime, &t.Status)
	if err == sql.ErrNoRows {
		return nil, errx.ErrTaskNotFound
	}
	return &t, err
}

// TaskListFilter 盘点任务列表筛选
type TaskListFilter struct {
	ScopeDeptIDs []int64 // scope_dept_id IN (...)，nil 表示不限制
	AssigneeUID  *int64  // 非空时仅返回指派给该用户的任务
	Status       *int16
	Page         int
	PageSize     int
}

func (m *InvModel) ListTasks(ctx context.Context, f TaskListFilter) ([]InventoryTask, int, error) {
	base := `FROM inventory_task t`
	where := ` WHERE 1=1`
	var args []any
	argIdx := 1

	if f.AssigneeUID != nil {
		base += ` JOIN inventory_task_assignee a ON t.id = a.task_id`
		where += ` AND a.user_id = $` + itoa(argIdx)
		args = append(args, *f.AssigneeUID)
		argIdx++
	}
	if len(f.ScopeDeptIDs) > 0 {
		ph := make([]string, len(f.ScopeDeptIDs))
		for i, id := range f.ScopeDeptIDs {
			ph[i] = "$" + itoa(argIdx)
			args = append(args, id)
			argIdx++
		}
		where += ` AND t.scope_dept_id IN (` + strings.Join(ph, ",") + `)`
	}
	if f.Status != nil {
		where += ` AND t.status = $` + itoa(argIdx)
		args = append(args, *f.Status)
		argIdx++
	}

	countQ := `SELECT COUNT(DISTINCT t.id) ` + base + where
	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := m.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT DISTINCT t.id, t.task_name, t.scope_dept_id, t.creator_id, t.start_time, t.end_time, t.status, t.created_at ` +
		base + where + ` ORDER BY t.created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []InventoryTask
	for rows.Next() {
		var t InventoryTask
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.TaskName, &t.ScopeDeptID, &t.CreatorID, &t.StartTime, &t.EndTime, &t.Status, &createdAt); err != nil {
			return nil, 0, err
		}
		list = append(list, t)
	}
	return list, total, rows.Err()
}

func (m *InvModel) GetAssigneeIDs(ctx context.Context, taskID int64) ([]int64, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT user_id FROM inventory_task_assignee WHERE task_id=$1 ORDER BY user_id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (m *InvModel) CountSubmitted(ctx context.Context, taskID int64) (int, error) {
	// 已归档后从 inventory_record 统计；进行中返回 0（由 handler 从 Mongo 补充）
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_record WHERE task_id=$1 AND is_scanned=1`, taskID).Scan(&count)
	return count, err
}

func (m *InvModel) ArchiveTask(ctx context.Context, taskID int64, records []InventoryRecord) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil { return err }
	defer tx.Rollback()

	for _, r := range records {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO inventory_record (task_id, asset_id, found_asset_desc, operator_id, is_scanned, actual_location, diff_status)
			 VALUES ($1,$2,$3,$4,$5,$6,0)
			 ON CONFLICT (task_id, asset_id) WHERE asset_id IS NOT NULL DO NOTHING`,
			taskID, r.AssetID, r.FoundAssetDesc, r.OperatorID, r.IsScanned, r.ActualLocation)
		if err != nil { return err }
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE inventory_task SET status=2 WHERE id=$1`, taskID)
	if err != nil { return err }

	return tx.Commit()
}

func (m *InvModel) GetRecords(ctx context.Context, taskID int64) ([]InventoryRecord, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, task_id, asset_id, found_asset_desc, operator_id, is_scanned, actual_location, diff_status
		 FROM inventory_record WHERE task_id=$1`, taskID)
	if err != nil { return nil, err }
	defer rows.Close()
	var list []InventoryRecord
	for rows.Next() {
		var r InventoryRecord
		if err := rows.Scan(&r.ID, &r.TaskID, &r.AssetID, &r.FoundAssetDesc, &r.OperatorID, &r.IsScanned, &r.ActualLocation, &r.DiffStatus); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (m *InvModel) IsAssignee(ctx context.Context, taskID, userID int64) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_task_assignee WHERE task_id=$1 AND user_id=$2`, taskID, userID).Scan(&count)
	return count > 0, err
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }
