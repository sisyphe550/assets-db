package logic

import (
	"context"
	"database/sql"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/service/user/model"
)

// FindUser 按 username 或 user_id 查询用户
func FindUser(ctx context.Context, db *sql.DB, userID int64, username string) (*model.SysUser, error) {
	um := model.NewUserModel(db)
	if userID > 0 {
		return um.FindByID(ctx, userID)
	}
	if username != "" {
		return um.FindByUsername(ctx, username)
	}
	return nil, errx.ErrInvalidParam
}
