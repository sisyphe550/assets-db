package inventorydraft

import (
	"testing"
	"time"
)

func TestHasAssigneeConflictWhenLocationsDiffer(t *testing.T) {
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	entries := []Entry{
		{AssetNo: "A", OperatorID: 10003, Location: "101", UpdatedAt: base},
		{AssetNo: "A", OperatorID: 10004, Location: "102", UpdatedAt: base},
	}
	if !HasAssigneeConflict(entries, []int64{10003, 10004}) {
		t.Fatal("expected conflict")
	}
}

func TestNoConflictWhenAssigneeContentsMatch(t *testing.T) {
	base := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	entries := []Entry{
		{AssetNo: "A", OperatorID: 10003, Location: "101", Notes: "ok", UpdatedAt: base},
		{AssetNo: "A", OperatorID: 10004, Location: "101", Notes: "ok", UpdatedAt: base.Add(time.Hour)},
	}
	if HasAssigneeConflict(entries, []int64{10003, 10004}) {
		t.Fatal("expected no conflict when normalized content matches")
	}
}

func TestNoConflictWithSingleAssignee(t *testing.T) {
	if HasAssigneeConflict([]Entry{{OperatorID: 10003, Location: "101"}}, []int64{10003}) {
		t.Fatal("single assignee should not conflict")
	}
}
