// Package middleware 部门数据隔离中间件
package middleware

import (
	"context"
	"net/http"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// DeptSubtreeProvider 获取用户可见的子树 ID 集合
// role=1 返回 nil（全量可见）
// role=2/3 返回部门子树 IDs
type DeptSubtreeProvider func(ctx context.Context, uid, deptID int64, roleLevel int16) ([]int64, error)

// RequireDeptScope 部门数据隔离中间件
// 将用户可见的部门子树 IDs 注入 context，供 handler 中的数据查询使用
func RequireDeptScope(provider DeptSubtreeProvider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := GetUID(r.Context())
			if !ok {
				code, httpStatus, msg := errx.ToHTTPError(errx.ErrUnauthenticated)
				writeJSON(w, httpStatus, code, msg)
				return
			}
			roleLevel, _ := GetRoleLevel(r.Context())
			deptID, _ := GetDeptID(r.Context())

			subtreeIDs, err := provider(r.Context(), uid, deptID, roleLevel)
			if err != nil {
				code, httpStatus, msg := errx.ToHTTPError(err)
				writeJSON(w, httpStatus, code, msg)
				return
			}

			ctx := r.Context()
			if subtreeIDs != nil {
				ctx = context.WithValue(ctx, ctxKey("dept_subtree"), subtreeIDs)
			}
			// role=1 时 subtreeIDs=nil，表示不限制
			ctx = context.WithValue(ctx, ctxKey("dept_subtree_unlimited"), subtreeIDs == nil)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetDeptSubtree 从 context 取部门子树 ID 集合
func GetDeptSubtree(ctx context.Context) (ids []int64, unlimited bool) {
	unlimited, _ = ctx.Value(ctxKey("dept_subtree_unlimited")).(bool)
	ids, _ = ctx.Value(ctxKey("dept_subtree")).([]int64)
	return
}

// DeptSubtreeProviderFromDB 基于数据库查询的 DeptSubtreeProvider
// allDepts: 全量部门列表（由调用方缓存或查询）
func DeptSubtreeProviderFromDB(allDepts []dept.Department) DeptSubtreeProvider {
	return func(ctx context.Context, uid, deptID int64, roleLevel int16) ([]int64, error) {
		if roleLevel == 1 {
			return nil, nil // 校级管理员全量可见
		}
		return dept.SubtreeIDs(allDepts, deptID)
	}
}
