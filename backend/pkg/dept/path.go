// Package dept 组织架构物化路径解析
// 对应 01-desgin.md §4.2.1 sys_department.path 的前缀匹配算法
package dept

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// Department 组织节点
type Department struct {
	ID       int64  `db:"id"`
	ParentID int64  `db:"parent_id"`
	Path     string `db:"path"`
}

// BuildPath 构建物化路径
// 规则：parentPath + id + "/"
// 根节点 parentPath 为空或 "/"
func BuildPath(parentPath string, id int64) string {
	parentPath = strings.TrimRight(parentPath, "/")
	if parentPath == "" {
		return fmt.Sprintf("/%d/", id)
	}
	return fmt.Sprintf("%s/%d/", parentPath, id)
}

// SubtreeIDs 计算管理员可见的全部 dept_id 列表（含自身及子孙）
// 输入：全量部门列表 + 管理员所属 deptID
// 返回：子树中所有 dept_id
//
// 规则：
//   - role=1（校级管理员）：调用方应传 rootDeptID=0，本函数不在此处理全量逻辑
//   - role=2（学院管理员）：返回该学院及所有下属实验室的 IDs
//   - 若管理员 deptID 不在列表中找到，返回 40401
func SubtreeIDs(all []Department, rootDeptID int64) ([]int64, error) {
	// 找到管理员的部门
	var rootPath string
	found := false
	for _, d := range all {
		if d.ID == rootDeptID {
			rootPath = d.Path
			found = true
			break
		}
	}
	if !found {
		return nil, errx.ErrNotFound
	}

	// 以前缀匹配收集整棵子树
	var ids []int64
	for _, d := range all {
		if strings.HasPrefix(d.Path, rootPath) {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}

// ChildIDs 返回直接子节点 ID 列表
func ChildIDs(all []Department, parentID int64) []int64 {
	var ids []int64
	for _, d := range all {
		if d.ParentID == parentID {
			ids = append(ids, d.ID)
		}
	}
	return ids
}

// BuildTree 将扁平部门列表转为树结构
type TreeNode struct {
	ID       int64       `json:"id"`
	ParentID int64       `json:"parentId"`
	DeptName string      `json:"deptName"`
	DeptCode string      `json:"deptCode"`
	Path     string      `json:"path"`
	Children []*TreeNode `json:"children"`
}

// DepartmentFull 完整部门信息
type DepartmentFull struct {
	ID        int64  `db:"id"`
	ParentID  int64  `db:"parent_id"`
	DeptName  string `db:"dept_name"`
	DeptCode  string `db:"dept_code"`
	Path      string `db:"path"`
	SortOrder int    `db:"sort_order"`
}

// ToTree 将扁平部门列表转为树结构
func ToTree(all []DepartmentFull, rootID int64) []*TreeNode {
	nodeMap := make(map[int64]*TreeNode)
	var roots []*TreeNode

	for _, d := range all {
		node := &TreeNode{
			ID:       d.ID,
			ParentID: d.ParentID,
			DeptName: d.DeptName,
			DeptCode: d.DeptCode,
			Path:     d.Path,
		}
		nodeMap[d.ID] = node
	}

	for _, d := range all {
		node := nodeMap[d.ID]
		if d.ParentID == rootID {
			roots = append(roots, node)
		} else if parent, ok := nodeMap[d.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	return roots
}

// PathDepth 计算路径深度
// "/1/" → 1, "/1/15/" → 2, "/1/15/103/" → 3
func PathDepth(path string) int {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts)
}

// AncestorID 获取指定深度的祖先 ID
// PathDepth 为 2 表示学院层级
func AncestorID(path string, depth int) (int64, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < depth {
		return 0, fmt.Errorf("path too shallow")
	}
	return strconv.ParseInt(parts[depth-1], 10, 64)
}

// CollegeSubtreeIDs 获取用户所属学院及全部下属部门 ID（用于共享资产可见范围）
func CollegeSubtreeIDs(all []Department, userDeptID int64) ([]int64, error) {
	var userPath string
	for _, d := range all {
		if d.ID == userDeptID {
			userPath = d.Path
			break
		}
	}
	if userPath == "" {
		return nil, errx.ErrNotFound
	}
	collegeID, err := AncestorID(userPath, 2)
	if err != nil {
		return nil, err
	}
	return SubtreeIDs(all, collegeID)
}
