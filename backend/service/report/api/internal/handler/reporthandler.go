package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
)

type ReportHandler struct {
	PG       *sql.DB
	ReportDB *sql.DB
	MySQL    *sql.DB
	RDB      *redis.Client
	ExportDir string
}

func NewReportHandler(pg, reportDB, mysql *sql.DB, rdb *redis.Client, exportDir string) *ReportHandler {
	return &ReportHandler{PG: pg, ReportDB: reportDB, MySQL: mysql, RDB: rdb, ExportDir: exportDir}
}

// GET /report/assets/by-dept
func (h *ReportHandler) AssetsByDept(w http.ResponseWriter, r *http.Request) {
	rows, err := h.PG.QueryContext(r.Context(),
		`SELECT department_id, SUM(total_count), SUM(in_stock_count), SUM(in_use_count), SUM(total_value)
		 FROM rpt_asset_daily_snapshot WHERE snapshot_date = CURRENT_DATE
		 GROUP BY department_id ORDER BY department_id`)
	if err != nil {
		// 快照表可能无数据，从 MySQL 实时查询
		h.assetsByDeptLive(w, r)
		return
	}
	defer rows.Close()

	type item struct {
		DeptID     int64   `json:"departmentId"`
		TotalCount int     `json:"totalCount"`
		InStock    int     `json:"inStockCount"`
		InUse      int     `json:"inUseCount"`
		TotalValue float64 `json:"totalValue"`
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.DeptID, &it.TotalCount, &it.InStock, &it.InUse, &it.TotalValue); err != nil {
			continue
		}
		items = append(items, it)
	}
	if len(items) == 0 {
		h.assetsByDeptLive(w, r)
		return
	}
	writeOK(w, map[string]any{"items": items})
}

func (h *ReportHandler) assetsByDeptLive(w http.ResponseWriter, r *http.Request) {
	rows, err := h.MySQL.QueryContext(r.Context(),
		`SELECT department_id, COUNT(*), SUM(CASE WHEN status=1 THEN 1 ELSE 0 END),
		        SUM(CASE WHEN status=2 THEN 1 ELSE 0 END), SUM(price)
		 FROM asset_ledger WHERE deleted_at IS NULL GROUP BY department_id`)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}
	defer rows.Close()

	type item struct {
		DeptID     int64   `json:"departmentId"`
		TotalCount int     `json:"totalCount"`
		InStock    int     `json:"inStockCount"`
		InUse      int     `json:"inUseCount"`
		TotalValue float64 `json:"totalValue"`
	}
	var items []item
	for rows.Next() {
		var it item
		rows.Scan(&it.DeptID, &it.TotalCount, &it.InStock, &it.InUse, &it.TotalValue)
		items = append(items, it)
	}
	writeOK(w, map[string]any{"items": items})
}

// GET /report/assets/by-category
func (h *ReportHandler) AssetsByCategory(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	userDeptID, _ := middleware.GetDeptID(r.Context())

	var deptIDs []int64
	if deptFilter := r.URL.Query().Get("departmentId"); deptFilter != "" {
		id, _ := strconv.ParseInt(deptFilter, 10, 64)
		if id > 0 {
			deptIDs = h.deptSubtreeIDs(r.Context(), id)
		}
	} else if roleLevel == 2 {
		deptIDs = h.deptSubtreeIDs(r.Context(), userDeptID)
	}

	query := `SELECT category, COUNT(*), COALESCE(SUM(price), 0)
		 FROM asset_ledger WHERE deleted_at IS NULL`
	args := []any{}
	if len(deptIDs) > 0 {
		ph := make([]string, len(deptIDs))
		for i, id := range deptIDs {
			ph[i] = "?"
			args = append(args, id)
		}
		query += ` AND department_id IN (` + strings.Join(ph, ",") + `)`
	}
	query += ` GROUP BY category ORDER BY COUNT(*) DESC`

	rows, err := h.MySQL.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}
	defer rows.Close()

	type item struct {
		Category   string  `json:"category"`
		Count      int     `json:"count"`
		TotalValue float64 `json:"totalValue"`
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.Category, &it.Count, &it.TotalValue); err != nil {
			continue
		}
		items = append(items, it)
	}
	if items == nil {
		items = []item{}
	}
	writeOK(w, map[string]any{"items": items})
}

func (h *ReportHandler) deptSubtreeIDs(ctx context.Context, rootDeptID int64) []int64 {
	rows, err := h.PG.QueryContext(ctx, `SELECT id, parent_id, path FROM sys_department`)
	if err != nil {
		return []int64{rootDeptID}
	}
	defer rows.Close()
	var all []dept.Department
	for rows.Next() {
		var d dept.Department
		if err := rows.Scan(&d.ID, &d.ParentID, &d.Path); err != nil {
			return []int64{rootDeptID}
		}
		all = append(all, d)
	}
	ids, err := dept.SubtreeIDs(all, rootDeptID)
	if err != nil || len(ids) == 0 {
		return []int64{rootDeptID}
	}
	return ids
}

// GET /report/inventory/diff/:taskId
func (h *ReportHandler) InventoryDiff(w http.ResponseWriter, r *http.Request) {
	taskID := parseLastPathSeg(r.URL.Path)
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	rows, err := h.PG.QueryContext(r.Context(),
		`SELECT diff_status, COUNT(*) FROM inventory_record WHERE task_id=$1 GROUP BY diff_status`, taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	defer rows.Close()

	result := map[string]int{"match": 0, "surplus": 0, "loss": 0}
	for rows.Next() {
		var status, count int
		rows.Scan(&status, &count)
		switch status {
		case 1: result["match"] = count
		case 2: result["surplus"] = count
		case 3: result["loss"] = count
		}
	}
	writeOK(w, result)
}

// POST /report/export
func (h *ReportHandler) Export(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	var req struct {
		ExportType string         `json:"exportType"`
		Params     map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if req.ExportType == "" {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	paramsJSON, _ := json.Marshal(req.Params)
	if req.Params == nil {
		paramsJSON = []byte("{}")
	}

	jobID, err := h.nextExportJobID(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	_, err = h.ReportDB.ExecContext(r.Context(),
		`INSERT INTO rpt_export_job (id, creator_id, export_type, params, status) VALUES ($1,$2,$3,$4,0)`,
		jobID, uid, req.ExportType, paramsJSON)
	if err != nil {
		writeErr(w, err)
		return
	}

	if h.RDB != nil {
		jobJSON, _ := json.Marshal(map[string]any{
			"jobId": jobID, "exportType": req.ExportType, "creatorId": uid, "params": req.Params,
		})
		h.RDB.LPush(r.Context(), "fams:export:queue", string(jobJSON))
	}

	writeOK(w, map[string]any{"jobId": jobID})
}

func (h *ReportHandler) nextExportJobID(ctx context.Context) (int64, error) {
	var id int64
	err := h.ReportDB.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) + 1 FROM rpt_export_job`).Scan(&id)
	return id, err
}

// GET /report/export/:jobId
func (h *ReportHandler) ExportStatus(w http.ResponseWriter, r *http.Request) {
	jobID := parseExportJobID(r.URL.Path)
	if jobID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	uid, _ := middleware.GetUID(r.Context())

	var status int16
	var errMsg *string
	var creatorID int64
	err := h.ReportDB.QueryRowContext(r.Context(),
		`SELECT status, error_message, creator_id FROM rpt_export_job WHERE id=$1`, jobID).
		Scan(&status, &errMsg, &creatorID)
	if err == sql.ErrNoRows {
		writeErr(w, errx.ErrNotFound)
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	if creatorID != uid {
		roleLevel, _ := middleware.GetRoleLevel(r.Context())
		if roleLevel != 1 {
			writeErr(w, errx.ErrForbidden)
			return
		}
	}

	resp := map[string]any{"jobId": jobID, "status": status, "errorMessage": errMsg}
	if status == 2 {
		resp["downloadUrl"] = fmt.Sprintf("/api/v1/report/export/%d/download", jobID)
	}
	writeOK(w, resp)
}

// GET /report/export/:jobId/download
func (h *ReportHandler) ExportDownload(w http.ResponseWriter, r *http.Request) {
	jobID := parseExportJobID(strings.TrimSuffix(r.URL.Path, "/download"))
	if jobID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	uid, _ := middleware.GetUID(r.Context())

	var status int16
	var filePath sql.NullString
	var creatorID int64
	err := h.ReportDB.QueryRowContext(r.Context(),
		`SELECT status, file_path, creator_id FROM rpt_export_job WHERE id=$1`, jobID).
		Scan(&status, &filePath, &creatorID)
	if err == sql.ErrNoRows {
		writeErr(w, errx.ErrNotFound)
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	if creatorID != uid {
		roleLevel, _ := middleware.GetRoleLevel(r.Context())
		if roleLevel != 1 {
			writeErr(w, errx.ErrForbidden)
			return
		}
	}
	if status != 2 || !filePath.Valid || filePath.String == "" {
		writeErr(w, errx.ErrInvalidState)
		return
	}

	data, err := os.ReadFile(filePath.String)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}
	baseName := filepath.Base(filePath.String)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+baseName)
	w.Write(data)
}

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": data})
}

func writeErr(w http.ResponseWriter, err error) {
	code, httpStatus, msg := errx.ToHTTPError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg, "data": nil})
}

func parseExportJobID(path string) int64 {
	path = strings.TrimSuffix(path, "/download")
	path = strings.TrimSuffix(path, "/")
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return 0
	}
	id, _ := strconv.ParseInt(path[idx+1:], 10, 64)
	return id
}

func parseLastPathSeg(path string) int64 {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			id, _ := strconv.ParseInt(path[i+1:], 10, 64)
			return id
		}
	}
	return 0
}
