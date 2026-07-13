package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

const (
	ConflictPending     int16 = 0
	ConflictResolved    int16 = 1
	ResolveFromAssignee int16 = 1
	ResolveCustom       int16 = 2
)

type ConflictCandidate struct {
	OperatorID     int64  `json:"operatorId"`
	ActualLocation string `json:"actualLocation"`
	Notes          string `json:"notes"`
	FoundName      string `json:"foundName"`
	UpdatedAt      string `json:"updatedAt"`
}

type AssigneeConflict struct {
	ID                     int64
	TaskID                 int64
	AssetNo                string
	AssetID                *int64
	Status                 int16
	Candidates             []ConflictCandidate
	ResolvedSource         *int16
	ResolvedOperatorID     *int64
	ResolvedActualLocation string
	ResolvedNotes          string
	ResolvedBy             *int64
	ResolvedAt             *time.Time
}

type ConflictInput struct {
	AssetNo    string
	AssetID    *int64
	Candidates []ConflictCandidate
}

type ResolveConflictInput struct {
	TaskID         int64
	AssetNo        string
	Source         int16
	OperatorID     *int64
	ActualLocation string
	Notes          string
	ResolvedBy     int64
	Record         InventoryRecord
}

func (m *InvModel) nextConflictID(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(id), 0) + 1 FROM inventory_assignee_conflict`).Scan(&id)
	return id, err
}

func (m *InvModel) InsertConflicts(ctx context.Context, tx *sql.Tx, taskID int64, conflicts []ConflictInput) error {
	for _, c := range conflicts {
		id, err := m.nextConflictID(ctx, tx)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(c.Candidates)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx,
			`INSERT INTO inventory_assignee_conflict (id, task_id, asset_no, asset_id, status, candidates)
			 VALUES ($1,$2,$3,$4,0,$5::jsonb)`,
			id, taskID, c.AssetNo, c.AssetID, string(raw))
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *InvModel) CountPendingConflicts(ctx context.Context, taskID int64) (int, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_assignee_conflict WHERE task_id=$1 AND status=0`, taskID).
		Scan(&count)
	return count, err
}

func (m *InvModel) ListConflicts(ctx context.Context, taskID int64, pendingOnly bool) ([]AssigneeConflict, error) {
	query := `SELECT id, task_id, asset_no, asset_id, status, candidates,
	                 resolved_source, resolved_operator_id, resolved_actual_location,
	                 resolved_notes, resolved_by, resolved_at
	          FROM inventory_assignee_conflict WHERE task_id=$1`
	if pendingOnly {
		query += ` AND status=0`
	}
	query += ` ORDER BY asset_no`
	rows, err := m.db.QueryContext(ctx, query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AssigneeConflict
	for rows.Next() {
		var c AssigneeConflict
		var raw []byte
		var resolvedSource sql.NullInt16
		var resolvedOp, resolvedBy sql.NullInt64
		var resolvedLocation, resolvedNotes sql.NullString
		var resolvedAt sql.NullTime
		if err := rows.Scan(
			&c.ID, &c.TaskID, &c.AssetNo, &c.AssetID, &c.Status, &raw,
			&resolvedSource, &resolvedOp, &resolvedLocation,
			&resolvedNotes, &resolvedBy, &resolvedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(raw, &c.Candidates)
		if resolvedSource.Valid {
			v := resolvedSource.Int16
			c.ResolvedSource = &v
		}
		if resolvedOp.Valid {
			v := resolvedOp.Int64
			c.ResolvedOperatorID = &v
		}
		if resolvedLocation.Valid {
			c.ResolvedActualLocation = resolvedLocation.String
		}
		if resolvedNotes.Valid {
			c.ResolvedNotes = resolvedNotes.String
		}
		if resolvedBy.Valid {
			v := resolvedBy.Int64
			c.ResolvedBy = &v
		}
		if resolvedAt.Valid {
			t := resolvedAt.Time
			c.ResolvedAt = &t
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (m *InvModel) ResolveConflict(ctx context.Context, in ResolveConflictInput) (bool, error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var status int16
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM inventory_assignee_conflict WHERE task_id=$1 AND asset_no=$2 FOR UPDATE`,
		in.TaskID, in.AssetNo).Scan(&status)
	if err == sql.ErrNoRows {
		return false, errx.ErrNotFound
	}
	if err != nil {
		return false, err
	}
	if status != ConflictPending {
		return false, errx.ErrInvalidState
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE inventory_assignee_conflict
		 SET status=1, resolved_source=$3, resolved_operator_id=$4,
		     resolved_actual_location=$5, resolved_notes=$6, resolved_by=$7, resolved_at=NOW()
		 WHERE task_id=$1 AND asset_no=$2`,
		in.TaskID, in.AssetNo, in.Source, in.OperatorID, in.ActualLocation, in.Notes, in.ResolvedBy)
	if err != nil {
		return false, err
	}

	if in.Record.AssetID != nil {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO inventory_record (task_id, asset_id, found_asset_desc, operator_id, is_scanned, actual_location, diff_status)
			 VALUES ($1,$2,$3,$4,$5,$6,0)
			 ON CONFLICT (task_id, asset_id) WHERE asset_id IS NOT NULL DO UPDATE SET
			   operator_id=EXCLUDED.operator_id,
			   is_scanned=EXCLUDED.is_scanned,
			   actual_location=EXCLUDED.actual_location,
			   found_asset_desc=EXCLUDED.found_asset_desc,
			   diff_status=0`,
			in.TaskID, in.Record.AssetID, in.Record.FoundAssetDesc, in.Record.OperatorID,
			in.Record.IsScanned, in.Record.ActualLocation)
	} else {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO inventory_record (task_id, asset_id, found_asset_desc, operator_id, is_scanned, actual_location, diff_status)
			 VALUES ($1,NULL,$2,$3,$4,$5,0)`,
			in.TaskID, in.Record.FoundAssetDesc, in.Record.OperatorID, in.Record.IsScanned, in.Record.ActualLocation)
	}
	if err != nil {
		return false, err
	}

	var pending int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM inventory_assignee_conflict WHERE task_id=$1 AND status=0`, in.TaskID).
		Scan(&pending); err != nil {
		return false, err
	}
	allResolved := pending == 0
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return allResolved, nil
}

func (m *InvModel) ArchiveTaskWithConflicts(ctx context.Context, taskID int64, records []InventoryRecord, conflicts []ConflictInput) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, r := range records {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO inventory_record (task_id, asset_id, found_asset_desc, operator_id, is_scanned, actual_location, diff_status)
			 VALUES ($1,$2,$3,$4,$5,$6,0)
			 ON CONFLICT (task_id, asset_id) WHERE asset_id IS NOT NULL DO NOTHING`,
			taskID, r.AssetID, r.FoundAssetDesc, r.OperatorID, r.IsScanned, r.ActualLocation)
		if err != nil {
			return err
		}
	}

	if err := m.InsertConflicts(ctx, tx, taskID, conflicts); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `UPDATE inventory_task SET status=2 WHERE id=$1`, taskID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ArchiveTask 保留兼容：无冲突时使用
func (m *InvModel) ArchiveTask(ctx context.Context, taskID int64, records []InventoryRecord) error {
	return m.ArchiveTaskWithConflicts(ctx, taskID, records, nil)
}
