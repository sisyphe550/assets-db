package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// WorkflowRequest 审批工单
type WorkflowRequest struct {
	ID           int64     `db:"id"`
	AssetID      int64     `db:"asset_id"`
	RequesterID  int64     `db:"requester_id"`
	DepartmentID int64     `db:"department_id"`
	Type         int16     `db:"type"`
	CurrentStage int16     `db:"current_stage"`
	Status       int16     `db:"status"`
	Reason       string    `db:"reason"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// WorkflowLog 审批留痕
type WorkflowLog struct {
	ID          int64     `db:"id"`
	RequestID   int64     `db:"request_id"`
	OperatorID  int64     `db:"operator_id"`
	Action      string    `db:"action"`
	Comment     string    `db:"comment"`
	OperateTime time.Time `db:"operate_time"`
}

// OutboxEvent 发件箱事件
type OutboxEvent struct {
	EventType    string `json:"event_type"`
	RequestID    int64  `json:"request_id"`
	AssetID      int64  `json:"asset_id"`
	TargetStatus int16  `json:"target_status"`
	AssignedUserID int64 `json:"assigned_user_id"`
	OperatorID   int64  `json:"operator_id"`
	Timestamp    int64  `json:"timestamp"`
}

type WorkflowModel struct{ db *sql.DB }

func NewWorkflowModel(db *sql.DB) *WorkflowModel { return &WorkflowModel{db: db} }

// Insert 创建工单
func (m *WorkflowModel) Insert(ctx context.Context, w *WorkflowRequest) (int64, error) {
	err := m.db.QueryRowContext(ctx,
		`INSERT INTO workflow_request (asset_id, requester_id, department_id, type, current_stage, status, reason)
		 VALUES ($1,$2,$3,$4,1,1,$5) RETURNING id`,
		w.AssetID, w.RequesterID, w.DepartmentID, w.Type, w.Reason).Scan(&w.ID)
	if err != nil {
		if isPgDup(err) {
			return 0, errx.ErrDuplicateOpen
		}
		return 0, err
	}
	// 插入日志
	m.InsertLog(ctx, w.ID, w.RequesterID, "提交申请", "")
	return w.ID, nil
}

func (m *WorkflowModel) FindByID(ctx context.Context, id int64) (*WorkflowRequest, error) {
	var w WorkflowRequest
	err := m.db.QueryRowContext(ctx,
		`SELECT id, asset_id, requester_id, department_id, type, current_stage, status, reason, created_at, updated_at
		 FROM workflow_request WHERE id=$1`, id).
		Scan(&w.ID, &w.AssetID, &w.RequesterID, &w.DepartmentID, &w.Type, &w.CurrentStage, &w.Status, &w.Reason, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, errx.ErrWorkflowNotFound
	}
	return &w, err
}

// List 查询工单列表
func (m *WorkflowModel) List(ctx context.Context, scope string, uid int64, deptIDs []int64, page, pageSize int) ([]WorkflowRequest, int, error) {
	var query, countQ string
	var args []any

	switch scope {
	case "my":
		query = `SELECT id, asset_id, requester_id, department_id, type, current_stage, status, reason, created_at, updated_at
				 FROM workflow_request WHERE requester_id=$1`
		countQ = `SELECT COUNT(*) FROM workflow_request WHERE requester_id=$1`
		args = append(args, uid)
	case "todo":
		// role=2: stage=1 + dept in subtree; role=1: stage=2
		query = `SELECT id, asset_id, requester_id, department_id, type, current_stage, status, reason, created_at, updated_at
				 FROM workflow_request WHERE status=1`
		countQ = `SELECT COUNT(*) FROM workflow_request WHERE status=1`
	case "done":
		query = `SELECT DISTINCT w.id, w.asset_id, w.requester_id, w.department_id, w.type, w.current_stage, w.status, w.reason, w.created_at, w.updated_at
				 FROM workflow_request w JOIN workflow_log l ON w.id=l.request_id WHERE l.operator_id=$1`
		countQ = `SELECT COUNT(DISTINCT w.id) FROM workflow_request w JOIN workflow_log l ON w.id=l.request_id WHERE l.operator_id=$1`
		args = append(args, uid)
	default:
		return nil, 0, errx.ErrInvalidParam
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := m.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	args = append(args, pageSize, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []WorkflowRequest
	for rows.Next() {
		var w WorkflowRequest
		if err := rows.Scan(&w.ID, &w.AssetID, &w.RequesterID, &w.DepartmentID, &w.Type, &w.CurrentStage, &w.Status, &w.Reason, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, 0, err
		}
		list = append(list, w)
	}
	return list, total, rows.Err()
}

// FindLogs 查询审批日志
func (m *WorkflowModel) FindLogs(ctx context.Context, requestID int64) ([]WorkflowLog, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, request_id, operator_id, action, comment, operate_time
		 FROM workflow_log WHERE request_id=$1 ORDER BY operate_time ASC`, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []WorkflowLog
	for rows.Next() {
		var l WorkflowLog
		if err := rows.Scan(&l.ID, &l.RequestID, &l.OperatorID, &l.Action, &l.Comment, &l.OperateTime); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// InsertLog 写入审批日志
func (m *WorkflowModel) InsertLog(ctx context.Context, requestID, operatorID int64, action, comment string) error {
	var id int64
	return m.db.QueryRowContext(ctx,
		`INSERT INTO workflow_log (request_id, operator_id, action, comment) VALUES ($1,$2,$3,$4) RETURNING id`,
		requestID, operatorID, action, comment).Scan(&id)
}

// ApproveStage1 院级初审通过
func (m *WorkflowModel) ApproveStage1(ctx context.Context, requestID, operatorID int64, comment string) error {
	result, err := m.db.ExecContext(ctx,
		`UPDATE workflow_request SET current_stage=2, updated_at=NOW() WHERE id=$1 AND status=1 AND current_stage=1`, requestID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrInvalidState
	}
	return m.InsertLog(ctx, requestID, operatorID, "院级初审同意", comment)
}

// ApproveStage2AndArchive 校级终审通过（事务：更新工单 + 写日志 + outbox）
func (m *WorkflowModel) ApproveStage2AndArchive(ctx context.Context, requestID, operatorID int64, comment string, eventType string, targetStatus int16, assignedUserID int64) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. 更新工单
	result, err := tx.ExecContext(ctx,
		`UPDATE workflow_request SET status=2, current_stage=3, updated_at=NOW() WHERE id=$1 AND status=1 AND current_stage=2`, requestID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrInvalidState
	}

	// 2. 写日志
	var logID int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO workflow_log (request_id, operator_id, action, comment) VALUES ($1,$2,$3,$4) RETURNING id`,
		requestID, operatorID, "校级复审通过", comment).Scan(&logID); err != nil {
		return err
	}

	// 3. 写 Outbox
	payload, _ := json.Marshal(OutboxEvent{
		EventType: eventType, RequestID: requestID, AssetID: 0, // asset_id 由调用方填入
		TargetStatus: targetStatus, AssignedUserID: assignedUserID,
		OperatorID: operatorID, Timestamp: time.Now().Unix(),
	})
	// 简化：asset_id 从 workflow_request 读取
	var assetID int64
	tx.QueryRowContext(ctx, `SELECT asset_id FROM workflow_request WHERE id=$1`, requestID).Scan(&assetID)
	payload, _ = json.Marshal(OutboxEvent{
		EventType: eventType, RequestID: requestID, AssetID: assetID,
		TargetStatus: targetStatus, AssignedUserID: assignedUserID,
		OperatorID: operatorID, Timestamp: time.Now().Unix(),
	})

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO workflow_outbox (event_type, partition_key, payload) VALUES ($1,$2,$3)`,
		eventType, itoa(assetID), string(payload)); err != nil {
		return err
	}

	return tx.Commit()
}

// Reject 驳回
func (m *WorkflowModel) Reject(ctx context.Context, requestID, operatorID int64, stage int16, comment string) error {
	action := "院级初审驳回"
	if stage == 2 {
		action = "校级复审驳回"
	}
	result, err := m.db.ExecContext(ctx,
		`UPDATE workflow_request SET status=3, updated_at=NOW() WHERE id=$1 AND status=1`, requestID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrInvalidState
	}
	return m.InsertLog(ctx, requestID, operatorID, action, comment)
}

func isPgDup(err error) bool {
	if err == nil { return false }
	msg := err.Error()
	return contains(msg, "duplicate key") || contains(msg, "23505")
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub { return true }
	}
	return false
}

func itoa(i int64) string { return fmt.Sprintf("%d", i) }
