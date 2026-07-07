package handler

import (
	"database/sql"
	"encoding/json"
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
	DB *sql.DB
}

func NewAssetHandler(db *sql.DB) *AssetHandler { return &AssetHandler{DB: db} }

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
	if page < 1 { page = 1 }
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 { pageSize = 20 }

	// 部门隔离
	subIDs, unlimited := middleware.GetDeptSubtree(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	// role=1 校管全量可见，或 dept scope middleware 设置了 unlimited
	if roleLevel == 1 {
		unlimited = true
	}
	var deptIDs []int64
	if !unlimited {
		deptIDs = subIDs
		if len(deptIDs) == 0 {
			writeOK(w, map[string]any{"list": []any{}, "page": page, "pageSize": pageSize, "total": 0})
			return
		}
	}

	var statusFilter *int8
	if s := q.Get("status"); s != "" {
		v, _ := strconv.Atoi(s)
		st := int8(v)
		statusFilter = &st
	}

	list, total, err := model.NewAssetModel(h.DB).List(r.Context(), deptIDs,
		q.Get("category"), q.Get("keyword"), statusFilter, page, pageSize)
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
	if err := model.NewAssetModel(h.DB).Update(r.Context(), id, req.Name, req.Category, req.Location, req.DepartmentId, req.IsShared); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// DELETE /asset/assets/:id
func (h *AssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := parseID(r.URL.Path, "/assets/")
	if err := model.NewAssetModel(h.DB).SoftDelete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// GET /asset/assets/shared
func (h *AssetHandler) SharedList(w http.ResponseWriter, r *http.Request) {
	// role=3 用户看到本学院 is_shared=1 的资产
	// 简化实现：由 dept middleware 限制子树
	h.List(w, r)
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
