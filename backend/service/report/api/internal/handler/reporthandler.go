package handler

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
)

type ReportHandler struct {
	PG       *sql.DB
	ReportDB *sql.DB
	MySQL    *sql.DB
	RDB      *redis.Client
}

func NewReportHandler(pg, reportDB, mysql *sql.DB, rdb *redis.Client) *ReportHandler {
	return &ReportHandler{PG: pg, ReportDB: reportDB, MySQL: mysql, RDB: rdb}
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
		ExportType string `json:"exportType"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var jobID int64
	err := h.ReportDB.QueryRowContext(r.Context(),
		`INSERT INTO rpt_export_job (creator_id, export_type, params) VALUES ($1,$2,'{}') RETURNING id`,
		uid, req.ExportType).Scan(&jobID)
	if err != nil {
		writeErr(w, err)
		return
	}

	// 投递到 Redis 队列
	if h.RDB != nil {
		jobJSON, _ := json.Marshal(map[string]any{
			"jobId": jobID, "exportType": req.ExportType, "creatorId": uid,
		})
		h.RDB.LPush(r.Context(), "fams:export:queue", string(jobJSON))
	}

	writeOK(w, map[string]any{"jobId": jobID})
}

// GET /report/export/:jobId
func (h *ReportHandler) ExportStatus(w http.ResponseWriter, r *http.Request) {
	jobID := parseLastPathSeg(r.URL.Path)
	var status int16
	var errMsg *string
	h.ReportDB.QueryRowContext(r.Context(),
		`SELECT status, error_message FROM rpt_export_job WHERE id=$1`, jobID).Scan(&status, &errMsg)
	writeOK(w, map[string]any{"jobId": jobID, "status": status})
}

// GET /report/export/:jobId/download
func (h *ReportHandler) ExportDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=export.csv")
	cw := csv.NewWriter(w)
	cw.Write([]string{"id", "name", "category", "status"})
	cw.Flush()
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

func parseLastPathSeg(path string) int64 {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			id, _ := strconv.ParseInt(path[i+1:], 10, 64)
			return id
		}
	}
	return 0
}
