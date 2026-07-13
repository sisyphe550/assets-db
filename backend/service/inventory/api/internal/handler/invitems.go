package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/service/inventory/model"
)

func (h *InvHandler) ListTaskItems(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/items")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if err := h.ensureInventoryAdminAccess(r, taskID); err != nil {
		writeErr(w, err)
		return
	}

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	selectedIDs, _ := im.GetTaskItemAssetIDs(r.Context(), taskID)
	available := h.fetchScopeAssets(r.Context(), task.ScopeDeptID)
	selected := selectExpectedTaskAssets(available, selectedIDs, false)

	writeOK(w, map[string]any{
		"list":      assetsToExpectedList(selected),
		"available": assetsToExpectedList(available),
		"total":     len(selected),
	})
}

func (h *InvHandler) UpdateTaskItems(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/items")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if err := h.ensureInventoryAdminAccess(r, taskID); err != nil {
		writeErr(w, err)
		return
	}

	var req struct {
		AssetIDs []int64 `json:"assetIds"`
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
	if task.Status != model.InventoryTaskDraft {
		writeErr(w, errx.ErrInvalidState)
		return
	}

	available := h.fetchScopeAssets(r.Context(), task.ScopeDeptID)
	availableByID := make(map[int64]struct{}, len(available))
	for _, a := range available {
		id := rpcAssetIDInt64(a)
		if id > 0 {
			availableByID[id] = struct{}{}
		}
	}
	for _, id := range req.AssetIDs {
		if _, ok := availableByID[id]; !ok {
			writeErr(w, errx.ErrInvalidParam)
			return
		}
	}

	if err := im.ReplaceTaskItems(r.Context(), taskID, req.AssetIDs); err != nil {
		writeErr(w, err)
		return
	}
	selected := selectExpectedTaskAssets(available, req.AssetIDs, false)
	writeOK(w, map[string]any{
		"list":  assetsToExpectedList(selected),
		"total": len(selected),
	})
}

func (h *InvHandler) PublishTask(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/publish")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if err := h.ensureInventoryAdminAccess(r, taskID); err != nil {
		writeErr(w, err)
		return
	}

	im := model.NewInvModel(h.DB)
	if err := im.PublishTask(r.Context(), taskID); err != nil {
		writeErr(w, err)
		return
	}
	task, _ := im.FindTask(r.Context(), taskID)
	expected := 0
	if task != nil {
		expected = len(h.expectedAssetsForTask(r.Context(), im, *task))
	}
	writeOK(w, map[string]any{
		"taskId":             taskID,
		"status":             model.InventoryTaskRunning,
		"expectedAssetCount": expected,
	})
}

func (h *InvHandler) expectedAssetsForTask(ctx context.Context, im *model.InvModel, task model.InventoryTask) []map[string]any {
	scopeAssets := h.fetchScopeAssets(ctx, task.ScopeDeptID)
	selectedIDs, err := im.GetTaskItemAssetIDs(ctx, task.ID)
	if err != nil {
		return []map[string]any{}
	}
	legacyFallback := task.Status != model.InventoryTaskDraft
	return selectExpectedTaskAssets(scopeAssets, selectedIDs, legacyFallback)
}

func (h *InvHandler) fetchScopeAssets(ctx context.Context, scopeDeptID int64) []map[string]any {
	if h.AssetRPC == "" {
		return nil
	}
	body, _ := json.Marshal(map[string]any{"deptIds": h.scopeDeptIDs(ctx, scopeDeptID)})
	resp, err := http.Post(h.AssetRPC+"/asset.rpc/ListAssetsByDeptIds", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Assets []map[string]any `json:"assets"`
	}
	if json.Unmarshal(respBody, &result) != nil {
		return nil
	}
	return result.Assets
}

func selectExpectedTaskAssets(scopeAssets []map[string]any, selectedIDs []int64, legacyFallback bool) []map[string]any {
	if len(selectedIDs) == 0 {
		if legacyFallback {
			return scopeAssets
		}
		return []map[string]any{}
	}

	selected := make(map[int64]struct{}, len(selectedIDs))
	for _, id := range selectedIDs {
		selected[id] = struct{}{}
	}

	out := make([]map[string]any, 0, len(selectedIDs))
	for _, asset := range scopeAssets {
		if _, ok := selected[rpcAssetIDInt64(asset)]; ok {
			out = append(out, asset)
		}
	}
	return out
}

func assetsToExpectedList(assets []map[string]any) []map[string]any {
	list := make([]map[string]any, 0, len(assets))
	for _, a := range assets {
		list = append(list, map[string]any{
			"assetId":      rpcAssetID(a),
			"assetNo":      rpcAssetNo(a),
			"name":         rpcAssetName(a),
			"bookLocation": rpcAssetLocation(a),
		})
	}
	return list
}
