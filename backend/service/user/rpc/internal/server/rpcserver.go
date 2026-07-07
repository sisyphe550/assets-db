package server

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/user/rpc/internal/logic"
)

// RPCServer 用户 RPC 服务（简化实现：HTTP JSON 替代 gRPC）
type RPCServer struct {
	DB           *sql.DB
	AccessSecret string
}

// Start 启动 RPC 服务器
func (s *RPCServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/user.rpc/FindUser", s.handleFindUser)
	mux.HandleFunc("/user.rpc/GetDeptSubtree", s.handleGetDeptSubtree)
	mux.HandleFunc("/user.rpc/ValidateToken", s.handleValidateToken)
	return http.ListenAndServe(addr, mux)
}

func (s *RPCServer) handleFindUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   int64  `json:"userId"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPCError(w, errx.ErrInvalidParam)
		return
	}
	u, err := logic.FindUser(r.Context(), s.DB, req.UserID, req.Username)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeRPC(w, map[string]any{
		"id":           u.ID,
		"username":     u.Username,
		"passwordHash": u.PasswordHash,
		"realName":     u.RealName,
		"roleLevel":    u.RoleLevel,
		"departmentId": u.DepartmentID,
		"status":       u.Status,
	})
}

func (s *RPCServer) handleGetDeptSubtree(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeptID int64 `json:"deptId"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	ids, err := logic.GetDeptSubtree(r.Context(), s.DB, req.DeptID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeRPC(w, map[string]any{"deptIds": ids})
}

func (s *RPCServer) handleValidateToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	claims, err := middleware.ParseRefreshToken(req.Token, s.AccessSecret)
	if err != nil {
		writeRPC(w, map[string]any{"valid": false})
		return
	}
	writeRPC(w, map[string]any{
		"valid":     true,
		"userId":    claims.UID,
		"roleLevel": claims.RoleLevel,
		"deptId":    claims.DeptID,
	})
}

func writeRPC(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeRPCError(w http.ResponseWriter, err error) {
	code, _, msg := errx.ToHTTPError(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // RPC 层不返回 HTTP 错误码
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": msg},
	})
}

// ErrPlaceholder avoids unused import
var _ = http.StatusOK
