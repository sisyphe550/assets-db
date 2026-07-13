package inventorydraft

import (
	"strings"

	"github.com/sisyphus550/assets-db/backend/pkg/strnorm"
)

func entryKey(e Entry) string {
	return strings.Join([]string{
		strnorm.Location(e.Location),
		strnorm.Normalize(e.Notes),
		strnorm.Normalize(e.FoundName),
		strnorm.Normalize(e.BookLocation),
	}, "\x00")
}

// LatestPerOperator 每个操作员保留最新一条草稿
func LatestPerOperator(entries []Entry) []Entry {
	latest := make(map[int64]Entry, len(entries))
	for _, e := range entries {
		if cur, ok := latest[e.OperatorID]; !ok || e.UpdatedAt.After(cur.UpdatedAt) {
			latest[e.OperatorID] = e
		}
	}
	out := make([]Entry, 0, len(latest))
	for _, e := range latest {
		out = append(out, e)
	}
	return out
}

// AssigneeEntries 仅保留指派盘点员的最新草稿
func AssigneeEntries(entries []Entry, assigneeIDs []int64) []Entry {
	if len(assigneeIDs) == 0 {
		return nil
	}
	set := make(map[int64]struct{}, len(assigneeIDs))
	for _, id := range assigneeIDs {
		set[id] = struct{}{}
	}
	var out []Entry
	for _, e := range LatestPerOperator(entries) {
		if _, ok := set[e.OperatorID]; ok {
			out = append(out, e)
		}
	}
	return out
}

// HasAssigneeConflict 两名及以上指派盘点员对同一资产填写内容不一致
func HasAssigneeConflict(entries []Entry, assigneeIDs []int64) bool {
	assignee := AssigneeEntries(entries, assigneeIDs)
	if len(assignee) < 2 {
		return false
	}
	keys := make(map[string]struct{}, len(assignee))
	for _, e := range assignee {
		keys[entryKey(e)] = struct{}{}
	}
	return len(keys) > 1
}

// PickConsensus 无冲突时取指派盘点员最新草稿；无指派员草稿时回退 PickForArchive
func PickConsensus(entries []Entry, assigneeIDs []int64) Entry {
	assignee := AssigneeEntries(entries, assigneeIDs)
	if len(assignee) == 0 {
		return PickForArchive(entries, assigneeIDs)
	}
	best := assignee[0]
	for _, e := range assignee[1:] {
		if e.UpdatedAt.After(best.UpdatedAt) {
			best = e
		}
	}
	return best
}
