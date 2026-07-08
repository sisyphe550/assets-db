package dept

import (
	"reflect"
	"testing"
)

func TestBuildPath(t *testing.T) {
	tests := []struct {
		parentPath string
		id         int64
		want       string
	}{
		{"", 1, "/1/"},
		{"/", 1, "/1/"},
		{"/1/", 15, "/1/15/"},
		{"/1/15/", 103, "/1/15/103/"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := BuildPath(tt.parentPath, tt.id)
			if got != tt.want {
				t.Errorf("BuildPath(%q, %d) = %q, want %q", tt.parentPath, tt.id, got, tt.want)
			}
		})
	}
}

func TestSubtreeIDs(t *testing.T) {
	// 模拟 seed 数据中的组织架构
	all := []Department{
		{ID: 1, ParentID: 0, Path: "/1/"},
		{ID: 15, ParentID: 1, Path: "/1/15/"},
		{ID: 103, ParentID: 15, Path: "/1/15/103/"},
		{ID: 104, ParentID: 15, Path: "/1/15/104/"},
		{ID: 20, ParentID: 1, Path: "/1/20/"},
	}

	tests := []struct {
		name     string
		deptID   int64
		wantIDs  []int64
		wantErr  bool
	}{
		{
			name:    "学院管理员看到本院+下属实验室",
			deptID:  15,
			wantIDs: []int64{15, 103, 104},
		},
		{
			name:    "实验室管理员仅看到自身",
			deptID:  103,
			wantIDs: []int64{103},
		},
		{
			name:    "另一学院管理员仅看到本院",
			deptID:  20,
			wantIDs: []int64{20},
		},
		{
			name:    "根节点不存在",
			deptID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := SubtreeIDs(all, tt.deptID)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(ids, tt.wantIDs) {
				t.Errorf("SubtreeIDs() = %v, want %v", ids, tt.wantIDs)
			}
		})
	}
}

func TestToTree(t *testing.T) {
	all := []DepartmentFull{
		{ID: 1, ParentID: 0, DeptName: "本校", DeptCode: "ROOT", Path: "/1/"},
		{ID: 15, ParentID: 1, DeptName: "信息工程学院", DeptCode: "INFO", Path: "/1/15/"},
		{ID: 103, ParentID: 15, DeptName: "软件工程实验室", DeptCode: "SE_LAB", Path: "/1/15/103/"},
	}

	nodes := ToTree(all, 1)
	if len(nodes) != 1 {
		t.Fatalf("expected 1 root, got %d", len(nodes))
	}
	root := nodes[0]
	if root.ID != 15 {
		t.Errorf("root id = %d, want 15", root.ID)
	}
	if len(root.Children) != 1 {
		t.Errorf("root children = %d, want 1", len(root.Children))
	}
	if root.Children[0].ID != 103 {
		t.Errorf("child id = %d, want 103", root.Children[0].ID)
	}
}

func TestPathDepth(t *testing.T) {
	tests := []struct {
		path  string
		depth int
	}{
		{"/1/", 1},
		{"/1/15/", 2},
		{"/1/15/103/", 3},
	}
	for _, tt := range tests {
		if got := PathDepth(tt.path); got != tt.depth {
			t.Errorf("PathDepth(%q) = %d, want %d", tt.path, got, tt.depth)
		}
	}
}

func TestAncestorID(t *testing.T) {
	id, err := AncestorID("/1/15/103/", 1)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Errorf("depth=1: %d, want 1", id)
	}

	id, err = AncestorID("/1/15/103/", 2)
	if err != nil {
		t.Fatal(err)
	}
	if id != 15 {
		t.Errorf("depth=2: %d, want 15", id)
	}

	_, err = AncestorID("/1/", 5)
	if err == nil {
		t.Error("expected error for too-shallow path")
	}
}

func TestChildIDs(t *testing.T) {
	all := []Department{
		{ID: 1, ParentID: 0, Path: "/1/"},
		{ID: 15, ParentID: 1, Path: "/1/15/"},
		{ID: 20, ParentID: 1, Path: "/1/20/"},
	}

	ids := ChildIDs(all, 1)
	if len(ids) != 2 {
		t.Errorf("ChildIDs(1) = %v, want 2 items", ids)
	}

	ids = ChildIDs(all, 15)
	if len(ids) != 0 {
		t.Errorf("ChildIDs(15) = %v, want empty", ids)
	}
}

func TestCollegeSubtreeIDs(t *testing.T) {
	all := []Department{
		{ID: 1, ParentID: 0, Path: "/1/"},
		{ID: 15, ParentID: 1, Path: "/1/15/"},
		{ID: 103, ParentID: 15, Path: "/1/15/103/"},
		{ID: 104, ParentID: 15, Path: "/1/15/104/"},
		{ID: 20, ParentID: 1, Path: "/1/20/"},
	}

	ids, err := CollegeSubtreeIDs(all, 103)
	if err != nil {
		t.Fatal(err)
	}
	want := []int64{15, 103, 104}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("CollegeSubtreeIDs(103) = %v, want %v", ids, want)
	}

	ids, err = CollegeSubtreeIDs(all, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []int64{20}) {
		t.Errorf("CollegeSubtreeIDs(20) = %v, want [20]", ids)
	}
}
