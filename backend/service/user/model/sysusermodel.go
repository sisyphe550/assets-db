package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// SysUser 用户表模型
type SysUser struct {
	ID           int64     `db:"id"`
	Username     string    `db:"username"`
	PasswordHash string    `db:"password_hash"`
	RealName     string    `db:"real_name"`
	RoleLevel    int16     `db:"role_level"`
	DepartmentID int64     `db:"department_id"`
	Status       int16     `db:"status"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type UserModel struct {
	db *sql.DB
}

func NewUserModel(db *sql.DB) *UserModel {
	return &UserModel{db: db}
}

func (m *UserModel) FindByUsername(ctx context.Context, username string) (*SysUser, error) {
	var u SysUser
	err := m.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, real_name, role_level, department_id, status
		 FROM sys_user WHERE username = $1`, username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RealName, &u.RoleLevel, &u.DepartmentID, &u.Status)
	if err == sql.ErrNoRows {
		return nil, errx.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *UserModel) FindByID(ctx context.Context, id int64) (*SysUser, error) {
	var u SysUser
	err := m.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, real_name, role_level, department_id, status
		 FROM sys_user WHERE id = $1`, id).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RealName, &u.RoleLevel, &u.DepartmentID, &u.Status)
	if err == sql.ErrNoRows {
		return nil, errx.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (m *UserModel) Insert(ctx context.Context, u *SysUser) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO sys_user (id, username, password_hash, real_name, role_level, department_id, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 1)`,
		u.ID, u.Username, u.PasswordHash, u.RealName, u.RoleLevel, u.DepartmentID)
	if err != nil {
		if isDuplicate(err) {
			return errx.ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (m *UserModel) UpdateStatus(ctx context.Context, id int64, status int16) error {
	result, err := m.db.ExecContext(ctx,
		`UPDATE sys_user SET status=$1, updated_at=NOW() WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errx.ErrUserNotFound
	}
	return nil
}

func (m *UserModel) NextID(ctx context.Context) (int64, error) {
	var id int64
	err := m.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(id), 10000) + 1 FROM sys_user`).Scan(&id)
	return id, err
}

func isDuplicate(err error) bool {
	return err != nil && (contains(err.Error(), "duplicate key") ||
		contains(err.Error(), "unique constraint") ||
		contains(err.Error(), "23505"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && searchSub(s, sub)
}

func searchSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
