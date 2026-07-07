package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/user/model"
)

// UserHandler 用户 API 处理器
type UserHandler struct {
	DB             *sql.DB
	AccessSecret   string
	RefreshSecret  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
}

// NewUserHandler 创建处理器
func NewUserHandler(db *sql.DB, accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *UserHandler {
	return &UserHandler{
		DB: db, AccessSecret: accessSecret, RefreshSecret: refreshSecret,
		AccessTTL: accessTTL, RefreshTTL: refreshTTL,
	}
}

// ==========================================
// Login
// ==========================================

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	um := model.NewUserModel(h.DB)
	u, err := um.FindByUsername(r.Context(), req.Username)
	if err != nil {
		writeErr(w, errx.ErrUnauthenticated) // 不区分用户名错误/密码错误
		return
	}
	if u.Status == 0 {
		writeErr(w, errx.ErrForbidden) // 40301 账户已禁用
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
		writeErr(w, errx.ErrUnauthenticated)
		return
	}

	accessToken, refreshToken, jti, err := middleware.GenerateTokens(
		u.ID, u.RoleLevel, u.DepartmentID,
		h.AccessSecret, h.RefreshSecret, h.AccessTTL, h.RefreshTTL)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}

	writeOK(w, map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expiresIn":    int64(h.AccessTTL.Seconds()),
		"tokenType":    "Bearer",
	})
	_ = jti
}

// ==========================================
// Refresh
// ==========================================

func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	claims, err := middleware.ParseRefreshToken(req.RefreshToken, h.RefreshSecret)
	if err != nil {
		writeErr(w, errx.ErrUnauthenticated)
		return
	}

	// 检查用户是否仍启用
	um := model.NewUserModel(h.DB)
	u, err := um.FindByID(r.Context(), claims.UID)
	if err != nil || u.Status == 0 {
		writeErr(w, errx.ErrTokenRevoked)
		return
	}

	accessToken, refreshToken, jti, err := middleware.GenerateTokens(
		claims.UID, claims.RoleLevel, claims.DeptID,
		h.AccessSecret, h.RefreshSecret, h.AccessTTL, h.RefreshTTL)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}

	writeOK(w, map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"expiresIn":    int64(h.AccessTTL.Seconds()),
		"tokenType":    "Bearer",
	})
	_ = jti
}

// ==========================================
// Logout
// ==========================================

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 黑名单处理由中间件在后续请求中完成
	// 此处仅返回成功（简化实现；生产环境应写 Redis 黑名单）
	writeOK(w, nil)
}

// ==========================================
// Me
// ==========================================

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	um := model.NewUserModel(h.DB)
	u, err := um.FindByID(r.Context(), uid)
	if err != nil {
		writeErr(w, err)
		return
	}

	dm := model.NewDeptModel(h.DB)
	d, _ := dm.FindByID(r.Context(), u.DepartmentID)
	deptName := ""
	if d != nil {
		deptName = d.DeptName
	}

	writeOK(w, map[string]any{
		"id":             u.ID,
		"username":       u.Username,
		"realName":       u.RealName,
		"roleLevel":      u.RoleLevel,
		"departmentId":   u.DepartmentID,
		"departmentName": deptName,
		"status":         u.Status,
	})
}

// ==========================================
// DeptTree
// ==========================================

func (h *UserHandler) DeptTree(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	deptID, _ := middleware.GetDeptID(r.Context())

	dm := model.NewDeptModel(h.DB)
	all, err := dm.FindAll(r.Context())
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}

	// 按角色裁剪
	var filtered []model.SysDepartment
	if roleLevel == 1 {
		filtered = all
	} else {
		ids, _ := dept.SubtreeIDs(model.ToDeptSlice(all), deptID)
		idSet := make(map[int64]bool)
		for _, id := range ids {
			idSet[id] = true
		}
		for _, d := range all {
			if idSet[d.ID] {
				filtered = append(filtered, d)
			}
		}
	}

	nodes := dept.ToTree(model.ToDeptFullSlice(filtered), 0)
	if nodes == nil {
		nodes = []*dept.TreeNode{}
	}
	writeOK(w, map[string]any{"nodes": nodes})
}

// ==========================================
// CreateDept
// ==========================================

func (h *UserHandler) CreateDept(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	if roleLevel != 1 {
		writeErr(w, errx.ErrForbidden)
		return
	}

	var req struct {
		ParentID  int64  `json:"parentId"`
		DeptName  string `json:"deptName"`
		DeptCode  string `json:"deptCode"`
		SortOrder int    `json:"sortOrder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	dm := model.NewDeptModel(h.DB)

	// 校验父节点存在
	parent, err := dm.FindByID(r.Context(), req.ParentID)
	if err != nil {
		writeErr(w, errx.ErrNotFound)
		return
	}

	id, _ := dm.NextID(r.Context())
	d := &model.SysDepartment{
		ID:        id,
		ParentID:  req.ParentID,
		DeptName:  req.DeptName,
		DeptCode:  req.DeptCode,
		Path:      dept.BuildPath(parent.Path, id),
		SortOrder: req.SortOrder,
	}

	if err := dm.Insert(r.Context(), d); err != nil {
		writeErr(w, err)
		return
	}

	writeOK(w, map[string]any{
		"id":       d.ID,
		"parentId": d.ParentID,
		"deptName": d.DeptName,
		"deptCode": d.DeptCode,
		"path":     d.Path,
	})
}

// ==========================================
// CreateUser
// ==========================================

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	adminDeptID, _ := middleware.GetDeptID(r.Context())

	var req struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		RealName     string `json:"realName"`
		RoleLevel    int16  `json:"roleLevel"`
		DepartmentId int64  `json:"departmentId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	// role=2 只能创建 role=3 用户
	if roleLevel == 2 && req.RoleLevel <= 2 {
		writeErr(w, errx.ErrForbidden)
		return
	}

	// 验证目标部门在管理员子树内
	if roleLevel == 2 {
		dm := model.NewDeptModel(h.DB)
		all, _ := dm.FindAll(r.Context())
		ids, _ := dept.SubtreeIDs(model.ToDeptSlice(all), adminDeptID)
		found := false
		for _, id := range ids {
			if id == req.DepartmentId {
				found = true
				break
			}
		}
		if !found {
			writeErr(w, errx.ErrDeptAccessDenied)
			return
		}
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	um := model.NewUserModel(h.DB)
	id, _ := um.NextID(r.Context())

	u := &model.SysUser{
		ID:           id,
		Username:     req.Username,
		PasswordHash: string(hash),
		RealName:     req.RealName,
		RoleLevel:    req.RoleLevel,
		DepartmentID: req.DepartmentId,
	}
	if err := um.Insert(r.Context(), u); err != nil {
		writeErr(w, err)
		return
	}

	writeOK(w, map[string]any{
		"id":       u.ID,
		"username": u.Username,
	})
}

// ==========================================
// UpdateUserStatus
// ==========================================

func (h *UserHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	adminDeptID, _ := middleware.GetDeptID(r.Context())

	// 解析路径参数 :id
	targetID := parsePathID(r.URL.Path, "/users/", "/status")
	if targetID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	var req struct {
		Status int16 `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	um := model.NewUserModel(h.DB)
	target, err := um.FindByID(r.Context(), targetID)
	if err != nil {
		writeErr(w, err)
		return
	}

	// 权限检查
	if roleLevel == 2 {
		dm := model.NewDeptModel(h.DB)
		all, _ := dm.FindAll(r.Context())
		ids, _ := dept.SubtreeIDs(model.ToDeptSlice(all), adminDeptID)
		found := false
		for _, id := range ids {
			if id == target.DepartmentID {
				found = true
				break
			}
		}
		if !found {
			writeErr(w, errx.ErrDeptAccessDenied)
			return
		}
	}

	if err := um.UpdateStatus(r.Context(), targetID, req.Status); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, nil)
}

// ==========================================
// ForceLogout
// ==========================================

func (h *UserHandler) ForceLogout(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	adminDeptID, _ := middleware.GetDeptID(r.Context())

	targetID := parsePathID(r.URL.Path, "/users/", "/force-logout")
	if targetID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	// 权限检查同 UpdateUserStatus
	if roleLevel == 2 {
		um := model.NewUserModel(h.DB)
		target, err := um.FindByID(r.Context(), targetID)
		if err != nil {
			writeErr(w, err)
			return
		}
		dm := model.NewDeptModel(h.DB)
		all, _ := dm.FindAll(r.Context())
		ids, _ := dept.SubtreeIDs(model.ToDeptSlice(all), adminDeptID)
		found := false
		for _, id := range ids {
			if id == target.DepartmentID {
				found = true
				break
			}
		}
		if !found {
			writeErr(w, errx.ErrDeptAccessDenied)
			return
		}
	}

	writeOK(w, nil)
}

// ==========================================
// 辅助函数
// ==========================================

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{
		"code":    0,
		"message": "ok",
		"data":    data,
	})
}

func writeErr(w http.ResponseWriter, err error) {
	code, httpStatus, msg := errx.ToHTTPError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": msg,
		"data":    nil,
	})
}

// parsePathID 解析路径中的 ID，如 /api/v1/user/users/10003/status → 10003
func parsePathID(path, prefix, suffix string) int64 {
	path = strings.TrimPrefix(path, "/api/v1/user")
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	id, _ := strconv.ParseInt(path, 10, 64)
	return id
}

// 避免未使用 import
var _ = context.Background
