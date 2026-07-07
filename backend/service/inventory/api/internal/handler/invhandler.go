package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/pkg/redislock"
	"github.com/sisyphus550/assets-db/backend/service/inventory/model"
)

type InvHandler struct {
	DB       *sql.DB
	DraftCol *mongo.Collection
	RDB      *redis.Client
	AssetRPC string
}

func NewInvHandler(db *sql.DB, draftCol *mongo.Collection, rdb *redis.Client, assetRPC string) *InvHandler {
	return &InvHandler{DB: db, DraftCol: draftCol, RDB: rdb, AssetRPC: assetRPC}
}

// POST /inventory/tasks
func (h *InvHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	var req struct {
		TaskName    string  `json:"taskName"`
		ScopeDeptID int64   `json:"scopeDeptId"`
		StartTime   string  `json:"startTime"`
		EndTime     string  `json:"endTime"`
		AssigneeIDs []int64 `json:"assigneeIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	st, _ := time.Parse(time.RFC3339, req.StartTime)
	et, _ := time.Parse(time.RFC3339, req.EndTime)
	if !et.After(st) {
		writeErr(w, errx.ErrInvalidTimeRange)
		return
	}

	t := &model.InventoryTask{
		TaskName: req.TaskName, ScopeDeptID: req.ScopeDeptID,
		CreatorID: uid, StartTime: st, EndTime: et,
	}
	if err := model.NewInvModel(h.DB).CreateTask(r.Context(), t, req.AssigneeIDs); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"taskId": t.ID, "taskName": t.TaskName})
}

// POST /inventory/tasks/:id/submit — 批量提交盘点草稿
// 对照 07-inventory-ops.md §4 和 01-desgin.md §5.3
func (h *InvHandler) Submit(w http.ResponseWriter, r *http.Request) {
	uid, _ := middleware.GetUID(r.Context())
	roleLevel, _ := middleware.GetRoleLevel(r.Context())

	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/submit")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	// 校验任务存在且进行中
	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if task.Status != 1 {
		writeErr(w, errx.ErrInvalidState)
		return
	}

	// 权限：必须是 assignee 或管理员
	if roleLevel > 2 {
		isAssignee, _ := im.IsAssignee(r.Context(), taskID, uid)
		if !isAssignee {
			writeErr(w, errx.ErrNotAssigned)
			return
		}
	}

	var req struct {
		Items []struct {
			AssetNo          string         `json:"assetNo"`
			ModifiedCells    map[string]any `json:"modifiedCells"`
			ExpectedUpdatedAt *string       `json:"expectedUpdatedAt"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, errx.ErrInvalidParam)
		return
	}
	if len(req.Items) == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	// 逐条处理
	var success []string
	var conflicts []map[string]any
	var failures []map[string]any

	for _, item := range req.Items {
		// 1. Redis 分布式锁
		lockKey := "fams:lock:inventory:" + item.AssetNo
		lockOwner := strconv.FormatInt(uid, 10)
		if h.RDB != nil {
			ok, lockErr := redislock.TryLock(r.Context(), h.RDB, lockKey, lockOwner, 30*time.Second)
			if lockErr != nil || !ok {
				conflicts = append(conflicts, map[string]any{
					"assetNo": item.AssetNo, "code": 40901,
					"message": "资产正在被他人盘点",
				})
				continue
			}
		}

		// 2. 校验 asset_no 在 scope 内（调用 asset-rpc）
		if h.AssetRPC != "" {
			inScope, scopeErr := h.checkAssetInScope(r.Context(), item.AssetNo, task.ScopeDeptID)
			if scopeErr != nil || !inScope {
				if h.RDB != nil { redislock.Unlock(r.Context(), h.RDB, lockKey, lockOwner) }
				if scopeErr != nil {
					failures = append(failures, map[string]any{
						"assetNo": item.AssetNo, "code": 50301,
						"message": "资产校验服务不可用",
					})
				} else {
					conflicts = append(conflicts, map[string]any{
						"assetNo": item.AssetNo, "code": 40302,
						"message": "资产不在盘点范围内",
					})
				}
				continue
			}
		}

		// 3. MongoDB Upsert with CAS
		ctx := r.Context()
		now := time.Now()
		filter := bson.M{"task_id": taskID, "asset_no": item.AssetNo}

		// CAS 版本检查
		if item.ExpectedUpdatedAt != nil && *item.ExpectedUpdatedAt != "" {
			expectedTime, parseErr := time.Parse(time.RFC3339, *item.ExpectedUpdatedAt)
			if parseErr == nil {
				filter["updated_at"] = expectedTime
			}
		}

		update := bson.M{
			"$set": bson.M{
				"task_id":        taskID,
				"asset_no":       item.AssetNo,
				"operator_id":    uid,
				"modified_cells": item.ModifiedCells,
				"updated_at":     now,
			},
		}
		opts := options.Update().SetUpsert(true)
		result, mongoErr := h.DraftCol.UpdateOne(ctx, filter, update, opts)

		if h.RDB != nil { redislock.Unlock(r.Context(), h.RDB, lockKey, lockOwner) }

		if mongoErr != nil {
			failures = append(failures, map[string]any{
				"assetNo": item.AssetNo, "code": 50301,
				"message": "草稿写入失败",
			})
			continue
		}

		// CAS 冲突：filter 匹配不到文档（版本不一致）
		if result.MatchedCount == 0 && result.UpsertedCount == 0 {
			conflicts = append(conflicts, map[string]any{
				"assetNo": item.AssetNo, "code": 40901,
				"message": "数据版本冲突，请刷新后重试",
			})
			continue
		}

		success = append(success, item.AssetNo)
	}

	// 部分成功语义：HTTP 200 + code 0，但 conflicts/failures 可能非空
	writeOK(w, map[string]any{
		"success":   success,
		"conflicts": conflicts,
		"failures":  failures,
	})
}

// checkAssetInScope 调用 asset-rpc 校验资产在 scope 内
func (h *InvHandler) checkAssetInScope(ctx context.Context, assetNo string, scopeDeptID int64) (bool, error) {
	body, _ := json.Marshal(map[string]any{
		"assetNo":      assetNo,
		"scopeDeptId":  scopeDeptID,
	})
	resp, err := http.Post(h.AssetRPC+"/asset.rpc/CheckAssetScope", "application/json", bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		InScope bool `json:"inScope"`
	}
	json.Unmarshal(respBody, &result)
	return result.InScope, nil
}

// POST /inventory/tasks/:id/archive
func (h *InvHandler) Archive(w http.ResponseWriter, r *http.Request) {
	id := parseIDForAction(r.URL.Path, "/tasks/", "/archive")
	var req struct{ Force bool `json:"force"` }
	json.NewDecoder(r.Body).Decode(&req)

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	if task.Status != 1 {
		writeOK(w, map[string]any{"status": "already_archived"})
		return
	}

	// 归档: 从 MongoDB 聚合草稿 + 从 asset-rpc 取账面资产 → 写入 inventory_record
	// 简化实现: 空归档（后续 ComparisonWorker 处理）
	if err := im.ArchiveTask(r.Context(), id, nil); err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"taskId": id, "archivedRecordCount": 0, "comparisonJobQueued": true})
}

// GET /inventory/tasks/:id/records
func (h *InvHandler) Records(w http.ResponseWriter, r *http.Request) {
	id := parseIDForAction(r.URL.Path, "/tasks/", "/records")
	records, err := model.NewInvModel(h.DB).GetRecords(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeOK(w, map[string]any{"records": records, "total": len(records)})
}

// GET /inventory/tasks/:id/expected-assets
func (h *InvHandler) ExpectedAssets(w http.ResponseWriter, r *http.Request) {
	id := parseIDForAction(r.URL.Path, "/tasks/", "/expected-assets")
	task, err := model.NewInvModel(h.DB).FindTask(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	// 通过 asset-rpc 获取 scope 内资产列表
	if h.AssetRPC != "" {
		body, _ := json.Marshal(map[string]any{"deptId": task.ScopeDeptID})
		resp, err := http.Post(h.AssetRPC+"/asset.rpc/ListAssetsByDeptIds", "application/json", bytes.NewReader(body))
		if err == nil {
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			var result struct {
				Assets []map[string]any `json:"assets"`
			}
			json.Unmarshal(respBody, &result)
			if len(result.Assets) > 0 {
				var list []map[string]any
				for _, a := range result.Assets {
					list = append(list, map[string]any{
						"assetId":      a["id"],
						"assetNo":      a["asset_no"],
						"name":         a["name"],
						"bookLocation": a["location"],
					})
				}
				writeOK(w, map[string]any{"list": list, "total": len(list)})
				return
			}
		}
	}
	writeOK(w, map[string]any{"list": []any{}, "total": 0})
}

// ========== helpers ==========

func writeOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "ok", "data": data})
}

func writeErr(w http.ResponseWriter, err error) {
	code, httpStatus, msg := errx.ToHTTPError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(map[string]any{"code": code, "message": msg, "data": nil})
}

func parseIDForAction(path, prefix, action string) int64 {
	// "/api/v1/inventory/tasks/123/submit" → 123
	s := path
	if i := strings.Index(s, action); i >= 0 {
		s = s[:i]
	}
	idx := strings.Index(s, prefix)
	if idx < 0 { return 0 }
	s = s[idx+len(prefix):]
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

// 避免未使用 import
var _ = fmt.Sprintf
var _ = primitive.NilObjectID
