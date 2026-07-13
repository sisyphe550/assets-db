package inventorydraft

import "time"

// Entry 归档时参与合并的一条 Mongo 草稿
type Entry struct {
	AssetNo      string
	OperatorID   int64
	Location     string
	FoundName    string
	BookLocation string
	UpdatedAt    time.Time
}

// PickForArchive 同一资产多条草稿时，优先采用指派盘点员的最新记录
func PickForArchive(entries []Entry, assigneeIDs []int64) Entry {
	if len(entries) == 0 {
		return Entry{}
	}
	assigneeSet := make(map[int64]struct{}, len(assigneeIDs))
	for _, id := range assigneeIDs {
		assigneeSet[id] = struct{}{}
	}

	candidates := entries
	filtered := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if _, ok := assigneeSet[e.OperatorID]; ok {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) > 0 {
		candidates = filtered
	}

	best := candidates[0]
	for _, e := range candidates[1:] {
		if e.UpdatedAt.After(best.UpdatedAt) {
			best = e
		}
	}
	return best
}
