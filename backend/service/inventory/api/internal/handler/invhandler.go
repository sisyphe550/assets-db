package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/inventory/model"
)

type InvHandler struct {
	DB *sql.DB
}

func NewInvHandler(db *sql.DB) *InvHandler { return &InvHandler{DB: db} }

// POST /inventory/tasks
func (h *InvHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	var req struct {
		TaskName    string  `json:"taskName"`
		ScopeDeptID int64   `json:"scopeDeptId"`
		StartTime   string  `json:"startTime"`
		EndTime     string  `json:"endTime"`
		AssigneeIDs []int64 `json:"assigneeIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	st, _ := time.Parse(time.RFC3339, req.StartTime)
	et, _ := time.Parse(time.RFC3339, req.EndTime)
	if !et.After(st) {
		writeErr(w, errx.ErrInvalidTimeRange)
		return
	}

	t := &model.InventoryTask{
		TaskName: req.TaskName, ScopeDeptID: req.ScopeDeptID,
		CreatorID: uid, StartTime: st, EndTime: et,
	}
	if err := model.NewInvModel(h.DB).CreateTask(r.Context(), t, req.AssigneeIDs); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"taskId": t.ID, "taskName": t.TaskName})
}

// POST /inventory/tasks/:id/archive
func (h *InvHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/tasks/")
	// 去掉 /archive
	path := r.URL.Path
	if i := strings.Index(path, "/archive"); i >= 0 {
		id = parseID(path[:i], "/tasks/")
	}

	var req struct{ Force bool `json:"force"` }
	json.NewDecoder(r.Body).Decode(&req)

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if task.Status != 1 {
		writeOK(w, map[string]any{"status": "already_archived"})
		return
	}

	// 简化归档：创建空白盘点记录
	if err := im.ArchiveTask(r.Context(), id, nil); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"taskId": id, "archivedRecordCount": 0, "comparisonJobQueued": true})
}

// GET /inventory/tasks/:id/records
func (h *InvHandler) Records(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/tasks/")
	if i := strings.Index(r.URL.Path, "/records"); i >= 0 {
		id = parseID(r.URL.Path[:i], "/tasks/")
	}
	records, err := model.NewInvModel(h.DB).GetRecords(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"records": records, "total": len(records)})
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

func parseID(path, prefix string) int64 {
	idx := strings.Index(path, prefix)
	if idx < 0 { return 0 }
	s := path[idx+len(prefix):]
	if i := strings.Index(s, "/"); i >= 0 { s = s[:i] }
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
