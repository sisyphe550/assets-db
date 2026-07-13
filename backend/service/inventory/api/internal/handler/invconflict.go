package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/inventorydraft"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/inventory/model"
)

// GET /inventory/tasks/:id/conflicts
func (h *InvHandler) ListConflicts(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/conflicts")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if err := h.ensureInventoryAdminAccess(r, taskID); err != nil {
		writeErr(w, err)
		return
	}

	im := model.NewInvModel(h.DB)
	conflicts, err := im.ListConflicts(r.Context(), taskID, true)
	if err != nil {
		writeErr(w, err)
		return
	}

	operatorIDs := make([]int64, 0)
	for _, c := range conflicts {
		for _, cand := range c.Candidates {
			operatorIDs = append(operatorIDs, cand.OperatorID)
		}
	}
	names := h.loadUserNames(r.Context(), operatorIDs)
	assetMap := h.fetchScopeAssetMapForTask(r.Context(), taskID)

	list := make([]map[string]any, 0, len(conflicts))
	for _, c := range conflicts {
		cands := make([]map[string]any, 0, len(c.Candidates))
		for _, cand := range c.Candidates {
			cands = append(cands, map[string]any{
				"operatorId":     cand.OperatorID,
				"operatorName":   names[cand.OperatorID],
				"actualLocation": cand.ActualLocation,
				"notes":          cand.Notes,
				"foundName":      cand.FoundName,
				"updatedAt":      cand.UpdatedAt,
			})
		}
		item := map[string]any{
			"assetNo":    c.AssetNo,
			"candidates": cands,
		}
		if c.AssetID != nil {
			item["assetId"] = *c.AssetID
			if a, ok := assetMap[*c.AssetID]; ok {
				item["name"] = rpcAssetName(a)
				item["bookLocation"] = rpcAssetLocation(a)
			}
		}
		list = append(list, item)
	}

	pending, _ := im.CountPendingConflicts(r.Context(), taskID)
	writeOK(w, map[string]any{"list": list, "total": len(list), "pendingCount": pending})
}

// POST /inventory/tasks/:id/conflicts/:assetNo/resolve
func (h *InvHandler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/conflicts")
	assetNo := parseConflictAssetNo(r.URL.Path)
	if taskID == 0 || assetNo == "" {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if err := h.ensureInventoryAdminAccess(r, taskID); err != nil {
		writeErr(w, err)
		return
	}

	var req struct {
		Source         string `json:"source"`
		OperatorID     *int64 `json:"operatorId"`
		ActualLocation string `json:"actualLocation"`
		Notes          string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if task.Status != 2 {
		writeErr(w, errx.ErrInvalidState)
		return
	}

	conflicts, err := im.ListConflicts(r.Context(), taskID, true)
	if err != nil {
		writeErr(w, err)
		return
	}
	var target *model.AssigneeConflict
	for i := range conflicts {
		if conflicts[i].AssetNo == assetNo {
			target = &conflicts[i]
			break
		}
	}
	if target == nil {
		writeErr(w, errx.ErrNotFound)
		return
	}

	uid, _ := middleware.GetUID(r.Context())
	source := model.ResolveCustom
	var opID *int64
	actualLoc := strings.TrimSpace(req.ActualLocation)
	notes := strings.TrimSpace(req.Notes)

	switch req.Source {
	case "assignee":
		if req.OperatorID == nil {
			writeErr(w, errx.ErrInvalidParam)
			return
		}
		var picked *model.ConflictCandidate
		for i := range target.Candidates {
			if target.Candidates[i].OperatorID == *req.OperatorID {
				picked = &target.Candidates[i]
				break
			}
		}
		if picked == nil {
			writeErr(w, errx.ErrInvalidParam)
			return
		}
		source = model.ResolveFromAssignee
		id := *req.OperatorID
		opID = &id
		actualLoc = picked.ActualLocation
		notes = picked.Notes
	case "custom":
		if actualLoc == "" {
			writeErr(w, errx.ErrInvalidParam)
			return
		}
	default:
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	bookAssets := h.fetchBookAssetsByNo(r.Context(), task.ScopeDeptID)
	record := model.InventoryRecord{
		TaskID:         taskID,
		IsScanned:      1,
		ActualLocation: actualLoc,
	}
	if bookAsset, ok := bookAssets[assetNo]; ok {
		aid := rpcAssetIDInt64(bookAsset)
		record.AssetID = &aid
		record.OperatorID = opID
	} else {
		foundName := ""
		for _, cand := range target.Candidates {
			if cand.FoundName != "" {
				foundName = cand.FoundName
				break
			}
		}
		record.FoundAssetDesc = formatSurplusDesc(foundName, "", actualLoc)
		record.OperatorID = opID
	}

	allResolved, err := im.ResolveConflict(r.Context(), model.ResolveConflictInput{
		TaskID:         taskID,
		AssetNo:        assetNo,
		Source:         source,
		OperatorID:     opID,
		ActualLocation: actualLoc,
		Notes:          notes,
		ResolvedBy:     uid,
		Record:         record,
	})
	if err != nil {
		writeErr(w, err)
		return
	}

	comparisonQueued := false
	if allResolved {
		go h.sendComparisonTask(taskID)
		comparisonQueued = true
	}

	pending, _ := im.CountPendingConflicts(r.Context(), taskID)
	writeOK(w, map[string]any{
		"assetNo":              assetNo,
		"pendingConflictCount": pending,
		"comparisonJobQueued":  comparisonQueued,
		"allResolved":          allResolved,
	})
}

func (h *InvHandler) ensureInventoryAdminAccess(r *http.Request, taskID int64) error {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	deptID, _ := middleware.GetDeptID(r.Context())
	if roleLevel == 3 {
		return errx.ErrForbidden
	}
	if roleLevel == 2 {
		im := model.NewInvModel(h.DB)
		task, err := im.FindTask(r.Context(), taskID)
		if err != nil {
			return err
		}
		for _, id := range h.deptSubtreeIDs(r.Context(), deptID) {
			if id == task.ScopeDeptID {
				return nil
			}
		}
		return errx.ErrDeptAccessDenied
	}
	return nil
}

func (h *InvHandler) fetchScopeAssetMapForTask(ctx context.Context, taskID int64) map[int64]map[string]any {
	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(ctx, taskID)
	if err != nil {
		return map[int64]map[string]any{}
	}
	return h.fetchScopeAssetMap(ctx, task.ScopeDeptID)
}

func (h *InvHandler) fetchBookAssetsByNo(ctx context.Context, scopeDeptID int64) map[string]map[string]any {
	result := make(map[string]map[string]any)
	for id, a := range h.fetchScopeAssetMap(ctx, scopeDeptID) {
		_ = id
		if no := rpcAssetNo(a); no != "" {
			result[no] = a
		}
	}
	return result
}

func (h *InvHandler) loadUserNames(ctx context.Context, ids []int64) map[int64]string {
	names := make(map[int64]string)
	if len(ids) == 0 {
		return names
	}
	seen := make(map[int64]struct{})
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	for _, id := range unique {
		var name string
		if err := h.DB.QueryRowContext(ctx,
			`SELECT COALESCE(real_name, username) FROM sys_user WHERE id=$1`, id).Scan(&name); err == nil {
			names[id] = name
		}
	}
	return names
}

func (h *InvHandler) candidatesFromEntries(entries []inventorydraft.Entry, assigneeIDs []int64) []model.ConflictCandidate {
	assignee := inventorydraft.AssigneeEntries(entries, assigneeIDs)
	out := make([]model.ConflictCandidate, 0, len(assignee))
	for _, e := range assignee {
		out = append(out, model.ConflictCandidate{
			OperatorID:     e.OperatorID,
			ActualLocation: e.Location,
			Notes:          e.Notes,
			FoundName:      e.FoundName,
			UpdatedAt:      formatDraftUpdatedAt(e.UpdatedAt),
		})
	}
	return out
}

func parseConflictAssetNo(path string) string {
	idx := strings.Index(path, "/conflicts/")
	if idx < 0 {
		return ""
	}
	rest := path[idx+len("/conflicts/"):]
	rest = strings.TrimSuffix(rest, "/resolve")
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	return strings.TrimSpace(rest)
}

func (h *InvHandler) pendingConflictCount(ctx context.Context, taskID int64, status int16) int {
	if status < 2 {
		return 0
	}
	count, err := model.NewInvModel(h.DB).CountPendingConflicts(ctx, taskID)
	if err != nil {
		return 0
	}
	return count
}
