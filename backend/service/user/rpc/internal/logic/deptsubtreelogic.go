package logic

import (
	"context"
	"database/sql"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/service/user/model"
)

// GetDeptSubtree 获取部门子树 ID 列表
func GetDeptSubtree(ctx context.Context, db *sql.DB, deptID int64) ([]int64, error) {
	dm := model.NewDeptModel(db)
	all, err := dm.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	ids, err := dept.SubtreeIDs(model.ToDeptSlice(all), deptID)
	if err != nil {
		return []int64{deptID}, nil // fallback: 至少返回自身
	}
	return ids, nil
}

// GetDeptSubtreeOrNil 校级管理员返回 nil（全量可见）
func GetDeptSubtreeOrNil(ctx context.Context, db *sql.DB, roleLevel int16, deptID int64) ([]int64, error) {
	if roleLevel == 1 {
		return nil, nil // nil = 全量
	}
	return GetDeptSubtree(ctx, db, deptID)
}

// ErrDeptSubtree 子树错误
var ErrDeptSubtree = errx.ErrNotFound
