package inventorydraft

import (
	"testing"
	"time"
)

func TestPickForArchivePrefersAssigneeOverAdmin(t *testing.T) {
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	entries := []Entry{
		{
			AssetNo: "EQUIP-2026-0001", OperatorID: 10001,
			Location: "管理员位置", UpdatedAt: base.Add(2 * time.Hour),
		},
		{
			AssetNo: "EQUIP-2026-0001", OperatorID: 10003,
			Location: "学生位置", UpdatedAt: base,
		},
	}
	picked := PickForArchive(entries, []int64{10003})
	if picked.Location != "学生位置" {
		t.Fatalf("expected assignee draft, got %q", picked.Location)
	}
}

func TestPickForArchiveUsesLatestAmongAssignees(t *testing.T) {
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	entries := []Entry{
		{AssetNo: "A", OperatorID: 10003, Location: "旧", UpdatedAt: base},
		{AssetNo: "A", OperatorID: 10003, Location: "新", UpdatedAt: base.Add(time.Hour)},
	}
	picked := PickForArchive(entries, []int64{10003})
	if picked.Location != "新" {
		t.Fatalf("expected latest assignee draft, got %q", picked.Location)
	}
}

func TestPickForArchiveFallsBackToLatestWhenNoAssigneeDraft(t *testing.T) {
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	entries := []Entry{
		{AssetNo: "A", OperatorID: 10001, Location: "旧", UpdatedAt: base},
		{AssetNo: "A", OperatorID: 10002, Location: "新", UpdatedAt: base.Add(time.Hour)},
	}
	picked := PickForArchive(entries, []int64{10003})
	if picked.Location != "新" {
		t.Fatalf("expected latest draft fallback, got %q", picked.Location)
	}
}
