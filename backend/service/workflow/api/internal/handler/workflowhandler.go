package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/lib/pq"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/workflow/model"
)

type WorkflowHandler struct {
	DB       *sql.DB
	AssetRPC string // asset-rpc 地址
}

func NewWorkflowHandler(db *sql.DB, assetRPC string) *WorkflowHandler {
	return &WorkflowHandler{DB: db, AssetRPC: assetRPC}
}

// POST /workflow/requests
func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	var req struct {
		AssetID int64  `json:"assetId"`
		Type    int16  `json:"type"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if req.Type < 1 || req.Type > 4 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	// 调用 asset-rpc 校验资产状态并获取 department_id
	departmentID := int64(0)
	if h.AssetRPC != "" {
		ok, deptID, rejectReason := h.callCheckAsset(req.AssetID, int32(req.Type), uid)
		if !ok {
			writeErrMsg(w, 42201, rejectReason)
			return
		}
		departmentID = deptID
	}

	wf := &model.WorkflowRequest{
		AssetID: req.AssetID, RequesterID: uid,
		DepartmentID: departmentID,
		Type: req.Type, Reason: req.Reason,
	}
	id, err := model.NewWorkflowModel(h.DB).Insert(r.Context(), wf)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeOK(w, map[string]any{
		"id": id, "assetId": req.AssetID, "type": req.Type,
		"currentStage": 1, "status": 1,
	})
}

// GET /workflow/requests
func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	adminDeptID, _ := middleware.GetDeptID(r.Context())

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	scope := q.Get("scope")
	if scope == "" {
		scope = "my"
	}

	if scope == "all" && roleLevel >= 3 {
		writeErr(w, errx.ErrForbidden)
		return
	}

	var deptIDs []int64
	var stageFilter *int16
	switch scope {
	case "todo":
		switch roleLevel {
		case 1:
			stage := int16(2)
			stageFilter = &stage
		case 2:
			deptIDs = h.deptSubtreeIDs(r.Context(), adminDeptID)
			stage := int16(1)
			stageFilter = &stage
		}
	case "all":
		if roleLevel == 2 {
			deptIDs = h.deptSubtreeIDs(r.Context(), adminDeptID)
		}
	}

	var assetIDFilter *int64
	if aid := q.Get("assetId"); aid != "" {
		id, _ := strconv.ParseInt(aid, 10, 64)
		if id > 0 {
			assetIDFilter = &id
		}
	}

	list, total, err := model.NewWorkflowModel(h.DB).List(
		r.Context(), scope, uid, deptIDs, stageFilter, assetIDFilter, page, pageSize,
	)
	if err != nil {
		writeErr(w, err)
		return
	}
	names := h.lookupUserNames(r.Context(), requesterIDs(list))
	items := make([]map[string]any, len(list))
	for i, w := range list {
		items[i] = workflowToMap(w, names[w.RequesterID])
	}
	writeOK(w, map[string]any{"list": items, "page": page, "pageSize": pageSize, "total": total})
}

func (h *WorkflowHandler) deptSubtreeIDs(ctx context.Context, rootDeptID int64) []int64 {
	rows, err := h.DB.QueryContext(ctx, `SELECT id, parent_id, path FROM sys_department`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var all []dept.Department
	for rows.Next() {
		var d dept.Department
		if err := rows.Scan(&d.ID, &d.ParentID, &d.Path); err != nil {
			return nil
		}
		all = append(all, d)
	}
	ids, err := dept.SubtreeIDs(all, rootDeptID)
	if err != nil {
		return nil
	}
	return ids
}

func (h *WorkflowHandler) ensureWorkflowReadable(r *http.Request, wf *model.WorkflowRequest) error {
	uid, _ := middleware.GetUID(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	switch roleLevel {
	case 1:
		return nil
	case 2:
		return h.ensureWorkflowInAdminSubtree(r.Context(), wf.DepartmentID)
	case 3:
		if wf.RequesterID == uid {
			return nil
		}
		return errx.ErrForbidden
	default:
		return errx.ErrForbidden
	}
}

func (h *WorkflowHandler) ensureWorkflowInAdminSubtree(ctx context.Context, departmentID int64) error {
	adminDeptID, _ := middleware.GetDeptID(ctx)
	for _, id := range h.deptSubtreeIDs(ctx, adminDeptID) {
		if id == departmentID {
			return nil
		}
	}
	return errx.ErrForbidden
}

func requesterIDs(list []model.WorkflowRequest) []int64 {
	seen := make(map[int64]struct{}, len(list))
	var ids []int64
	for _, w := range list {
		if _, ok := seen[w.RequesterID]; ok {
			continue
		}
		seen[w.RequesterID] = struct{}{}
		ids = append(ids, w.RequesterID)
	}
	return ids
}

func (h *WorkflowHandler) lookupUserNames(ctx context.Context, ids []int64) map[int64]string {
	names := make(map[int64]string, len(ids))
	if len(ids) == 0 {
		return names
	}
	rows, err := h.DB.QueryContext(ctx, `SELECT id, real_name FROM sys_user WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name string
		if rows.Scan(&id, &name) == nil {
			names[id] = name
		}
	}
	return names
}

func workflowToMap(w model.WorkflowRequest, requesterName string) map[string]any {
	m := map[string]any{
		"id": w.ID, "assetId": w.AssetID, "requesterId": w.RequesterID,
		"departmentId": w.DepartmentID, "type": w.Type,
		"currentStage": w.CurrentStage, "status": w.Status,
		"reason": w.Reason,
		"createdAt": w.CreatedAt.Format(time.RFC3339),
		"updatedAt": w.UpdatedAt.Format(time.RFC3339),
	}
	if requesterName != "" {
		m["requesterName"] = requesterName
	}
	return m
}

// GET /workflow/requests/:id
func (h *WorkflowHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := parsePathID(r.URL.Path, "/requests/")
	wf, err := model.NewWorkflowModel(h.DB).FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := h.ensureWorkflowReadable(r, wf); err != nil {
		writeErr(w, err)
		return
	}
	logs, _ := model.NewWorkflowModel(h.DB).FindLogs(r.Context(), id)
	names := h.lookupUserNames(r.Context(), []int64{wf.RequesterID})
	writeOK(w, map[string]any{
		"request": workflowToMap(*wf, names[wf.RequesterID]),
		"logs":    logs,
	})
}

// POST /workflow/requests/:id/approve
func (h *WorkflowHandler) Approve(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	id := parsePathID(r.URL.Path, "/requests/")
	// 去掉 /approve 后缀
	if strings.HasSuffix(r.URL.Path, "/approve") {
		id = parsePathID(strings.TrimSuffix(r.URL.Path, "/approve"), "/requests/")
	}

	var req struct{ Comment string `json:"comment"` }
	json.NewDecoder(r.Body).Decode(&req)

	wm := model.NewWorkflowModel(h.DB)
	wf, err := wm.FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}

	if wf.Status != 1 {
		writeErr(w, errx.ErrAlreadyArchived)
		return
	}

	// 事件映射
	eventMap := map[int16]struct {
		EventType    string
		TargetStatus int16
		AssignedUID  int64
	}{
		1: {"ASSET_USE_APPROVED", 2, wf.RequesterID},
		2: {"ASSET_RETURN_APPROVED", 1, 0},
		3: {"ASSET_REPAIR_APPROVED", 3, 0},
		4: {"ASSET_SCRAP_APPROVED", 4, 0},
	}

	switch wf.CurrentStage {
	case 1: // 院级初审
		if roleLevel != 2 {
			writeErr(w, errx.ErrForbidden)
			return
		}
		if err := h.ensureWorkflowInAdminSubtree(r.Context(), wf.DepartmentID); err != nil {
			writeErr(w, err)
			return
		}
		if err := wm.ApproveStage1(r.Context(), id, uid, req.Comment); err != nil {
			writeErr(w, err)
			return
		}
	case 2: // 校级终审
		if roleLevel != 1 {
			writeErr(w, errx.ErrForbidden)
			return
		}
		ev, ok := eventMap[wf.Type]
		if !ok {
			writeErr(w, errx.ErrInvalidState)
			return
		}
		if err := wm.ApproveStage2AndArchive(r.Context(), id, uid, req.Comment, ev.EventType, ev.TargetStatus, ev.AssignedUID); err != nil {
			writeErr(w, err)
			return
		}
	default:
		writeErr(w, errx.ErrAlreadyArchived)
		return
	}
	writeOK(w, nil)
}

// POST /workflow/requests/:id/reject
func (h *WorkflowHandler) Reject(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	id := parsePathID(r.URL.Path, "/requests/")
	if strings.HasSuffix(r.URL.Path, "/reject") {
		id = parsePathID(strings.TrimSuffix(r.URL.Path, "/reject"), "/requests/")
	}

	var req struct{ Comment string `json:"comment"` }
	json.NewDecoder(r.Body).Decode(&req)

	wm := model.NewWorkflowModel(h.DB)
	wf, err := wm.FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if wf.Status != 1 {
		writeErr(w, errx.ErrAlreadyArchived)
		return
	}

	// 权限检查
	if wf.CurrentStage == 1 && roleLevel != 2 {
		writeErr(w, errx.ErrForbidden)
		return
	}
	if wf.CurrentStage == 1 && roleLevel == 2 {
		if err := h.ensureWorkflowInAdminSubtree(r.Context(), wf.DepartmentID); err != nil {
			writeErr(w, err)
			return
		}
	}
	if wf.CurrentStage == 2 && roleLevel != 1 {
		writeErr(w, errx.ErrForbidden)
		return
	}

	if err := wm.Reject(r.Context(), id, uid, wf.CurrentStage, req.Comment); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// callCheckAsset 调用 asset-rpc 校验资产状态
func (h *WorkflowHandler) callCheckAsset(assetID int64, wfType int32, requesterID int64) (bool, int64, string) {
	body, _ := json.Marshal(map[string]any{
		"assetId":      assetID,
		"workflowType": wfType,
		"requesterId":  requesterID,
	})
	resp, err := http.Post(h.AssetRPC+"/asset.rpc/CheckAssetForWorkflow", "application/json", bytes.NewReader(body))
	if err != nil {
		return false, 0, "资产校验服务不可用"
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		OK           bool   `json:"ok"`
		DepartmentID int64  `json:"departmentId"`
		RejectReason string `json:"rejectReason"`
	}
	json.Unmarshal(respBody, &result)
	return result.OK, result.DepartmentID, result.RejectReason
}

func writeErrMsg(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnprocessableEntity)
	json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg, "data": nil})
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

func parsePathID(path, prefix string) int64 {
	idx := strings.Index(path, prefix)
	if idx < 0 { return 0 }
	s := path[idx+len(prefix):]
	if i := strings.Index(s, "/"); i >= 0 { s = s[:i] }
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
