package model

import (
	"context"
	"database/sql"

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// SysDepartment 组织架构表模型
type SysDepartment struct {
	ID        int64  `db:"id"`
	ParentID  int64  `db:"parent_id"`
	DeptName  string `db:"dept_name"`
	DeptCode  string `db:"dept_code"`
	Path      string `db:"path"`
	SortOrder int    `db:"sort_order"`
}

type DeptModel struct {
	db *sql.DB
}

func NewDeptModel(db *sql.DB) *DeptModel {
	return &DeptModel{db: db}
}

// FindAll 返回全量部门列表
func (m *DeptModel) FindAll(ctx context.Context) ([]SysDepartment, error) {
	rows, err := m.db.QueryContext(ctx,
		`SELECT id, parent_id, dept_name, dept_code, path, sort_order
		 FROM sys_department ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SysDepartment
	for rows.Next() {
		var d SysDepartment
		if err := rows.Scan(&d.ID, &d.ParentID, &d.DeptName, &d.DeptCode, &d.Path, &d.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// FindByID 按 ID 查部门
func (m *DeptModel) FindByID(ctx context.Context, id int64) (*SysDepartment, error) {
	var d SysDepartment
	err := m.db.QueryRowContext(ctx,
		`SELECT id, parent_id, dept_name, dept_code, path, sort_order
		 FROM sys_department WHERE id = $1`, id).
		Scan(&d.ID, &d.ParentID, &d.DeptName, &d.DeptCode, &d.Path, &d.SortOrder)
	if err == sql.ErrNoRows {
		return nil, errx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Insert 新增部门
func (m *DeptModel) Insert(ctx context.Context, d *SysDepartment) error {
	_, err := m.db.ExecContext(ctx,
		`INSERT INTO sys_department (id, parent_id, dept_name, dept_code, path, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		d.ID, d.ParentID, d.DeptName, d.DeptCode, d.Path, d.SortOrder)
	if err != nil {
		if isDuplicate(err) {
			return errx.ErrDuplicateKey
		}
		return err
	}
	return nil
}

// HasChildren 检查是否有子节点
func (m *DeptModel) HasChildren(ctx context.Context, id int64) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_department WHERE parent_id = $1`, id).Scan(&count)
	return count > 0, err
}

// HasUsers 检查部门下是否有用户
func (m *DeptModel) HasUsers(ctx context.Context, id int64) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sys_user WHERE department_id = $1`, id).Scan(&count)
	return count > 0, err
}

// NextID 获取下一个 ID
func (m *DeptModel) NextID(ctx context.Context) (int64, error) {
	var id int64
	err := m.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(id), 10) + 1 FROM sys_department`).Scan(&id)
	return id, err
}

// ToDeptSlice 转为 dept.Department 列表（供子树计算）
func ToDeptSlice(list []SysDepartment) []dept.Department {
	result := make([]dept.Department, len(list))
	for i, d := range list {
		result[i] = dept.Department{ID: d.ID, ParentID: d.ParentID, Path: d.Path}
	}
	return result
}

// ToDeptFullSlice 转为 dept.DepartmentFull 列表（供树构建）
func ToDeptFullSlice(list []SysDepartment) []dept.DepartmentFull {
	result := make([]dept.DepartmentFull, len(list))
	for i, d := range list {
		result[i] = dept.DepartmentFull{
			ID: d.ID, ParentID: d.ParentID, DeptName: d.DeptName,
			DeptCode: d.DeptCode, Path: d.Path, SortOrder: d.SortOrder,
		}
	}
	return result
}
