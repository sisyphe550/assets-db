package model

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
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

// UserListFilter 用户列表查询条件
type UserListFilter struct {
	DeptIDs   []int64 // department_id IN (...)，空表示不限制
	Keyword   string  // 模糊匹配 username / real_name
	RoleLevel *int16
	Page      int
	PageSize  int
}

// List 分页查询用户（不含 password_hash）
func (m *UserModel) List(ctx context.Context, f UserListFilter) ([]SysUser, int, error) {
	query := `SELECT id, username, password_hash, real_name, role_level, department_id, status
	          FROM sys_user WHERE 1=1`
	countQ := `SELECT COUNT(*) FROM sys_user WHERE 1=1`
	var args []any
	argIdx := 1

	if len(f.DeptIDs) > 0 {
		ph := make([]string, len(f.DeptIDs))
		for i, id := range f.DeptIDs {
			ph[i] = "$" + itoa(argIdx)
			args = append(args, id)
			argIdx++
		}
		cond := " AND department_id IN (" + strings.Join(ph, ",") + ")"
		query += cond
		countQ += cond
	}
	if f.Keyword != "" {
		kw := "%" + f.Keyword + "%"
		query += ` AND (username ILIKE $` + itoa(argIdx) + ` OR real_name ILIKE $` + itoa(argIdx) + `)`
		countQ += ` AND (username ILIKE $` + itoa(argIdx) + ` OR real_name ILIKE $` + itoa(argIdx) + `)`
		args = append(args, kw)
		argIdx++
	}
	if f.RoleLevel != nil {
		query += ` AND role_level = $` + itoa(argIdx)
		countQ += ` AND role_level = $` + itoa(argIdx)
		args = append(args, *f.RoleLevel)
		argIdx++
	}

	var total int
	countArgs := make([]any, len(args))
	copy(countArgs, args)
	if err := m.db.QueryRowContext(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.PageSize
	query += ` ORDER BY id ASC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, f.PageSize, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []SysUser
	for rows.Next() {
		var u SysUser
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RealName, &u.RoleLevel, &u.DepartmentID, &u.Status); err != nil {
			return nil, 0, err
		}
		u.PasswordHash = "" // 不向外暴露
		list = append(list, u)
	}
	return list, total, rows.Err()
}

func itoa(i int) string {
	return strconv.Itoa(i)
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
