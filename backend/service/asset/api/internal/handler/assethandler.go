package handler

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/asset/model"
)

type AssetHandler struct {
	DB         *sql.DB
	UserAPIURL string
}

func NewAssetHandler(db *sql.DB) *AssetHandler {
	return &AssetHandler{DB: db, UserAPIURL: "http://localhost:8888"}
}

func NewAssetHandlerWithUserAPI(db *sql.DB, userAPIURL string) *AssetHandler {
	return &AssetHandler{DB: db, UserAPIURL: userAPIURL}
}

// POST /asset/assets
func (h *AssetHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetNo      string `json:"assetNo"`
		Name         string `json:"name"`
		Category     string `json:"category"`
		Price        float64 `json:"price"`
		PurchaseTime string `json:"purchaseTime"`
		Location     string `json:"location"`
		DepartmentId int64  `json:"departmentId"`
		IsShared     int8   `json:"isShared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errx.ErrInvalidParam)
		return
	}
	pt, _ := time.Parse(time.RFC3339, req.PurchaseTime)
	a := &model.AssetLedger{
		AssetNo: req.AssetNo, Name: req.Name, Category: req.Category,
		Price: req.Price, PurchaseTime: pt, Location: req.Location,
		DepartmentID: req.DepartmentId, IsShared: req.IsShared,
	}
	if err := model.NewAssetModel(h.DB).Insert(r.Context(), a); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{
		"id": a.ID, "assetNo": a.AssetNo, "name": a.Name, "status": 1,
	})
}

// GET /asset/assets
func (h *AssetHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	uid, _ := middleware.GetUID(r.Context())

	deptIDs, unlimited, err := h.visibleDeptIDs(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !unlimited && len(deptIDs) == 0 {
		writeOK(w, map[string]any{"list": []any{}, "page": page, "pageSize": pageSize, "total": 0})
		return
	}

	var statusFilter *int8
	if s := q.Get("status"); s != "" {
		v, _ := strconv.Atoi(s)
		st := int8(v)
		statusFilter = &st
	}

	var userIDFilter *int64
	if q.Get("scope") == "my" {
		userIDFilter = &uid
	} else if u := q.Get("userId"); u != "" {
		if u == "me" {
			userIDFilter = &uid
		} else {
			id, _ := strconv.ParseInt(u, 10, 64)
			if id > 0 {
				userIDFilter = &id
			}
		}
	}

	list, total, err := model.NewAssetModel(h.DB).List(r.Context(), model.AssetListFilter{
		DeptIDs:    deptIDs,
		Category:   q.Get("category"),
		Keyword:    q.Get("keyword"),
		Status:     statusFilter,
		UserID:     userIDFilter,
		SharedOnly: false,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeErr(w, err)
		return
	}

	items := make([]map[string]any, len(list))
	for i, a := range list {
		items[i] = assetToMap(a)
	}
	writeOK(w, map[string]any{"list": items, "page": page, "pageSize": pageSize, "total": total})
}

// GET /asset/assets/:id
func (h *AssetHandler) Detail(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/assets/")
	if id == 0 {
		writeJSON(w, http.StatusBadRequest, errx.ErrInvalidParam)
		return
	}
	a, err := model.NewAssetModel(h.DB).FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := h.ensureAssetInScope(r, a.DepartmentID); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, assetToMap(*a))
}

// PUT /asset/assets/:id
func (h *AssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/assets/")
	var req struct {
		Name         string `json:"name"`
		Category     string `json:"category"`
		Location     string `json:"location"`
		DepartmentId int64  `json:"departmentId"`
		IsShared     int8   `json:"isShared"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errx.ErrInvalidParam)
		return
	}
	existing, err := model.NewAssetModel(h.DB).FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := h.ensureAssetInScope(r, existing.DepartmentID); err != nil {
		writeErr(w, err)
		return
	}
	if req.DepartmentId > 0 {
		if err := h.ensureAssetInScope(r, req.DepartmentId); err != nil {
			writeErr(w, err)
			return
		}
	}
	if err := model.NewAssetModel(h.DB).Update(r.Context(), id, req.Name, req.Category, req.Location, req.DepartmentId, req.IsShared); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// DELETE /asset/assets/:id
func (h *AssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/assets/")
	existing, err := model.NewAssetModel(h.DB).FindByID(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if err := h.ensureAssetInScope(r, existing.DepartmentID); err != nil {
		writeErr(w, err)
		return
	}
	if err := model.NewAssetModel(h.DB).SoftDelete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// GET /asset/assets/shared
func (h *AssetHandler) SharedList(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	if roleLevel != 3 {
		writeErr(w, errx.ErrForbidden)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	collegeDeptIDs, err := h.collegeSubtreeIDs(r)
	if err != nil || len(collegeDeptIDs) == 0 {
		writeOK(w, map[string]any{"list": []any{}, "page": page, "pageSize": pageSize, "total": 0})
		return
	}

	list, total, err := model.NewAssetModel(h.DB).List(r.Context(), model.AssetListFilter{
		DeptIDs:    collegeDeptIDs,
		SharedOnly: true,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeErr(w, err)
		return
	}

	items := make([]map[string]any, len(list))
	for i, a := range list {
		items[i] = assetToMap(a)
	}
	writeOK(w, map[string]any{"list": items, "page": page, "pageSize": pageSize, "total": total})
}

// visibleDeptIDs 返回当前用户可见的部门 ID 集合。
// asset-api 未挂载 RequireDeptScope 时，通过 user-api 的 college-subtree 接口补全院级子树。
func (h *AssetHandler) visibleDeptIDs(r *http.Request) (deptIDs []int64, unlimited bool, err error) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	if roleLevel == 1 {
		return nil, true, nil
	}
	subIDs, unlimited := middleware.GetDeptSubtree(r.Context())
	if unlimited {
		return nil, true, nil
	}
	if len(subIDs) > 0 {
		return subIDs, false, nil
	}
	ids, err := h.collegeSubtreeIDs(r)
	if err != nil {
		return nil, false, err
	}
	return ids, false, nil
}

func (h *AssetHandler) ensureAssetInScope(r *http.Request, departmentID int64) error {
	deptIDs, unlimited, err := h.visibleDeptIDs(r)
	if err != nil {
		return err
	}
	if unlimited {
		return nil
	}
	for _, id := range deptIDs {
		if id == departmentID {
			return nil
		}
	}
	return errx.ErrForbidden
}

func (h *AssetHandler) collegeSubtreeIDs(r *http.Request) ([]int64, error) {
	req, err := http.NewRequest(http.MethodGet, h.UserAPIURL+"/api/v1/user/departments/college-subtree", nil)
	if err != nil {
		return nil, err
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Code int `json:"code"`
		Data struct {
			DeptIds []int64 `json:"deptIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, errx.ErrInternal
	}
	return result.Data.DeptIds, nil
}

// ========== helpers ==========

func assetToMap(a model.AssetLedger) map[string]any {
	m := map[string]any{
		"id": a.ID, "assetNo": a.AssetNo, "name": a.Name,
		"category": a.Category, "price": a.Price,
		"purchaseTime": a.PurchaseTime.Format(time.RFC3339),
		"location": a.Location, "departmentId": a.DepartmentID,
		"isShared": a.IsShared, "status": a.Status,
	}
	if a.UserID != nil {
		m["userId"] = *a.UserID
	}
	return m
}

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": data})
}

func writeErr(w http.ResponseWriter, err error) {
	code, httpStatus, msg := errx.ToHTTPError(err)
	writeJSON(w, httpStatus, errx.New(code, httpStatus, msg))
}

func writeJSON(w http.ResponseWriter, httpStatus int, err *errx.BizError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]any{"code": err.Code, "message": err.Message, "data": nil})
}

func parseID(path, prefix string) int64 {
	idx := strings.Index(path, prefix)
	if idx < 0 { return 0 }
	s := path[idx+len(prefix):]
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
