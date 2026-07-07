package strnorm

import "testing"

func TestLocation(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"一号实验楼101", "一号实验楼101"},
		{"  一号实验楼101  ", "一号实验楼101"},
		{"一号实验楼　101", "一号实验楼 101"}, // 全角空格
		{"一号  实验楼", "一号 实验楼"},       // 多空格合并
	}
	for _, tt := range tests {
		got := Location(tt.in)
		if got != tt.want {
			t.Errorf("Location(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEqual(t *testing.T) {
	if !Equal("一号实验楼101", "  一号实验楼101 ") {
		t.Error("expected equal")
	}
	if Equal("A", "B") {
		t.Error("expected not equal")
	}
}
