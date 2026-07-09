package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	"github.com/sisyphus550/assets-db/backend/pkg/dept"
	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/pkg/inventorycmp"
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
	h := &InvHandler{DB: db, DraftCol: draftCol, RDB: rdb, AssetRPC: assetRPC}
	if draftCol != nil {
		ensureDraftIndexes(context.Background(), draftCol)
	}
	return h
}

func ensureDraftIndexes(ctx context.Context, col *mongo.Collection) {
	idx := col.Indexes()
	_, _ = idx.DropOne(ctx, "uk_task_asset")
	_, err := idx.CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "task_id", Value: 1},
			{Key: "asset_no", Value: 1},
			{Key: "operator_id", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetName("uk_task_asset_operator"),
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		log.Printf("WARNING: ensure draft index: %v", err)
	}
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
	expectedCount := h.countExpectedAssets(r, t.ScopeDeptID)
	writeOK(w, map[string]any{
		"taskId":             t.ID,
		"taskName":           t.TaskName,
		"scopeDeptId":        t.ScopeDeptID,
		"status":             1,
		"assigneeIds":        req.AssigneeIDs,
		"expectedAssetCount": expectedCount,
	})
}

// GET /inventory/tasks
func (h *InvHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	uid, _ := middleware.GetUID(r.Context())
	deptID, _ := middleware.GetDeptID(r.Context())

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var statusFilter *int16
	if s := q.Get("status"); s != "" {
		v, _ := strconv.Atoi(s)
		st := int16(v)
		statusFilter = &st
	}

	filter := model.TaskListFilter{
		Status:   statusFilter,
		Page:     page,
		PageSize: pageSize,
	}

	scope := q.Get("scope")
	if roleLevel == 3 || scope == "assigned" {
		filter.AssigneeUID = &uid
	} else if roleLevel == 2 {
		filter.ScopeDeptIDs = h.deptSubtreeIDs(r.Context(), deptID)
	}
	// role=1: no dept filter (all tasks)

	im := model.NewInvModel(h.DB)
	list, total, err := im.ListTasks(r.Context(), filter)
	if err != nil {
		writeErr(w, err)
		return
	}

	items := make([]map[string]any, len(list))
	for i, t := range list {
		assignees, _ := im.GetAssigneeIDs(r.Context(), t.ID)
		expected := h.countExpectedAssets(r, t.ScopeDeptID)
		submitted := h.countSubmittedDrafts(r.Context(), t.ID, uid, roleLevel == 3)
		items[i] = taskToMap(t, assignees, expected, submitted)
	}
	writeOK(w, map[string]any{"list": items, "page": page, "pageSize": pageSize, "total": total})
}

// GET /inventory/tasks/:id
func (h *InvHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	uid, _ := middleware.GetUID(r.Context())
	deptID, _ := middleware.GetDeptID(r.Context())

	taskID := parseIDForAction(r.URL.Path, "/tasks/", "")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}

	if roleLevel == 3 {
		ok, _ := im.IsAssignee(r.Context(), taskID, uid)
		if !ok {
			writeErr(w, errx.ErrNotAssigned)
			return
		}
	} else if roleLevel == 2 {
		allowed := false
		for _, id := range h.deptSubtreeIDs(r.Context(), deptID) {
			if id == task.ScopeDeptID {
				allowed = true
				break
			}
		}
		if !allowed {
			writeErr(w, errx.ErrDeptAccessDenied)
			return
		}
	}

	assignees, _ := im.GetAssigneeIDs(r.Context(), taskID)
	expected := h.countExpectedAssets(r, task.ScopeDeptID)
	submitted := h.countSubmittedDrafts(r.Context(), task.ID, uid, roleLevel == 3)
	writeOK(w, taskToMap(*task, assignees, expected, submitted))
}

func taskToMap(t model.InventoryTask, assignees []int64, expected, submitted int) map[string]any {
	return map[string]any{
		"id":                 t.ID,
		"taskName":           t.TaskName,
		"scopeDeptId":        t.ScopeDeptID,
		"creatorId":          t.CreatorID,
		"startTime":          t.StartTime.Format(time.RFC3339),
		"endTime":            t.EndTime.Format(time.RFC3339),
		"status":             t.Status,
		"assigneeIds":        assignees,
		"expectedAssetCount": expected,
		"submittedCount":     submitted,
	}
}

func (h *InvHandler) deptSubtreeIDs(ctx context.Context, rootDeptID int64) []int64 {
	rows, err := h.DB.QueryContext(ctx, `SELECT id, parent_id, path FROM sys_department`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var all []dept.Department
	for rows.Next() {
		var d dept.Department
		if err := rows.Scan(&d.ID, &d.ParentID, &d.Path); err != nil {
			return nil
		}
		all = append(all, d)
	}
	ids, err := dept.SubtreeIDs(all, rootDeptID)
	if err != nil {
		return nil
	}
	return ids
}

func (h *InvHandler) countExpectedAssets(r *http.Request, scopeDeptID int64) int {
	if h.AssetRPC == "" {
		return 0
	}
	body, _ := json.Marshal(map[string]any{"deptIds": []int64{scopeDeptID}})
	resp, err := http.Post(h.AssetRPC+"/asset.rpc/ListAssetsByDeptIds", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var result struct {
		Assets []map[string]any `json:"assets"`
	}
	json.Unmarshal(respBody, &result)
	return len(result.Assets)
}

func (h *InvHandler) countSubmittedDrafts(ctx context.Context, taskID int64, operatorID int64, perOperator bool) int {
	if h.DraftCol == nil {
		return 0
	}
	filter := bson.M{"task_id": taskID}
	if perOperator {
		filter["operator_id"] = operatorID
	}
	count, err := h.DraftCol.CountDocuments(ctx, filter)
	if err != nil {
		return 0
	}
	return int(count)
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

  // 逐条处理（初始化为空切片，避免 JSON 序列化为 null）
  success := make([]string, 0)
  conflicts := make([]map[string]any, 0)
  failures := make([]map[string]any, 0)

	for _, item := range req.Items {
		// 1. Redis 分布式锁
		lockKey := fmt.Sprintf("fams:lock:inventory:%d:%s:%d", taskID, item.AssetNo, uid)
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
		filter := bson.M{"task_id": taskID, "asset_no": item.AssetNo, "operator_id": uid}

		casMode := false
		// CAS 版本检查
		if item.ExpectedUpdatedAt != nil && *item.ExpectedUpdatedAt != "" {
			expectedTime, parseErr := parseDraftUpdatedAt(*item.ExpectedUpdatedAt)
			if parseErr == nil {
				filter["updated_at"] = expectedTime
				casMode = true
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

		var result *mongo.UpdateResult
		var mongoErr error
		if casMode {
			// CAS 更新：禁止 upsert，避免唯一索引冲突被误报为「草稿写入失败」
			result, mongoErr = h.DraftCol.UpdateOne(ctx, filter, update)
		} else {
			opts := options.Update().SetUpsert(true)
			result, mongoErr = h.DraftCol.UpdateOne(ctx, filter, update, opts)
		}

		if h.RDB != nil { redislock.Unlock(r.Context(), h.RDB, lockKey, lockOwner) }

		if mongoErr != nil {
			failures = append(failures, map[string]any{
				"assetNo": item.AssetNo, "code": 50301,
				"message": "草稿写入失败",
			})
			continue
		}

		if casMode {
			if result.MatchedCount == 0 {
				conflicts = append(conflicts, map[string]any{
					"assetNo": item.AssetNo, "code": 40901,
					"message": "数据版本冲突，请刷新后重试",
				})
				continue
			}
		} else if result.MatchedCount == 0 && result.UpsertedCount == 0 {
			failures = append(failures, map[string]any{
				"assetNo": item.AssetNo, "code": 50301,
				"message": "草稿写入失败",
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

// checkAssetInScope 调用 asset-rpc 校验资产在 scope 子树内
func (h *InvHandler) checkAssetInScope(ctx context.Context, assetNo string, scopeDeptID int64) (bool, error) {
	scopeIDs := h.deptSubtreeIDs(ctx, scopeDeptID)
	if len(scopeIDs) == 0 {
		scopeIDs = []int64{scopeDeptID}
	}
	body, _ := json.Marshal(map[string]any{
		"assetNo":        assetNo,
		"scopeDeptIds":   scopeIDs,
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
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/archive")
	var req struct{ Force bool `json:"force"` }
	json.NewDecoder(r.Body).Decode(&req)

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if task.Status != 1 {
		writeOK(w, map[string]any{"status": "already_archived"})
		return
	}

	// Step 1: 从 MongoDB 读取所有草稿
	draftAssetNos := make(map[string]map[string]any)
	if h.DraftCol != nil {
		cursor, err := h.DraftCol.Find(r.Context(), bson.M{"task_id": taskID})
		if err == nil {
			defer cursor.Close(r.Context())
			for cursor.Next(r.Context()) {
				var draft struct {
					AssetNo       string         `bson:"asset_no"`
					OperatorID    int64          `bson:"operator_id"`
					ModifiedCells map[string]any `bson:"modified_cells"`
				}
				cursor.Decode(&draft)
				draftAssetNos[draft.AssetNo] = map[string]any{
					"operator_id": draft.OperatorID,
					"location":    getCellStr(draft.ModifiedCells, "actual_location"),
					"found_name":  getCellStr(draft.ModifiedCells, "found_name"),
					"book_location": getCellStr(draft.ModifiedCells, "book_location"),
				}
			}
		}
	}

	// Step 2: 从 asset-rpc 获取 scope 内所有账面资产
	bookAssets := make(map[string]map[string]any)
	if h.AssetRPC != "" {
		body, _ := json.Marshal(map[string]any{"deptIds": []int64{task.ScopeDeptID}})
		resp, err := http.Post(h.AssetRPC+"/asset.rpc/ListAssetsByDeptIds", "application/json", bytes.NewReader(body))
		if err == nil {
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			var result struct {
				Assets []map[string]any `json:"assets"`
			}
			json.Unmarshal(respBody, &result)
			for _, a := range result.Assets {
				if assetNo := rpcAssetNo(a); assetNo != "" {
					bookAssets[assetNo] = a
				}
			}
		}
	}

	// Step 3: 生成 inventory_record
	var records []model.InventoryRecord
	recordCount := 0

	// 已扫描记录（来自 draft）
	for assetNo, draft := range draftAssetNos {
		opID, _ := draft["operator_id"].(int64)
		loc, _ := draft["location"].(string)
		foundName, _ := draft["found_name"].(string)
		bookLoc, _ := draft["book_location"].(string)

		if bookAsset, ok := bookAssets[assetNo]; ok {
			// 账面存在：正常记录
			aid := rpcAssetIDInt64(bookAsset)
			records = append(records, model.InventoryRecord{
				TaskID: taskID, AssetID: &aid,
				OperatorID: &opID, IsScanned: 1,
				ActualLocation: loc,
			})
		} else {
			// 账面无此编号：盘盈候选
			desc := formatSurplusDesc(foundName, bookLoc, loc)
			records = append(records, model.InventoryRecord{
				TaskID: taskID, AssetID: nil,
				FoundAssetDesc: desc,
				OperatorID: &opID, IsScanned: 1,
				ActualLocation: loc,
			})
		}
		recordCount++
	}

	// 盘亏候选（账面有但 draft 无）
	for assetNo, bookAsset := range bookAssets {
		if _, ok := draftAssetNos[assetNo]; !ok {
			aid := rpcAssetIDInt64(bookAsset)
			records = append(records, model.InventoryRecord{
				TaskID: taskID, AssetID: &aid,
				IsScanned: 0,
			})
			recordCount++
		}
	}

	// Step 4: 事务写入 + 更新任务状态
	if err := im.ArchiveTask(r.Context(), taskID, records); err != nil {
		writeErr(w, err)
		return
	}

	// Step 5: 发送 Kafka 消息触发比对 Worker
	go h.sendComparisonTask(taskID)

	writeOK(w, map[string]any{
		"taskId": taskID, "archivedRecordCount": recordCount,
		"scannedCount": len(draftAssetNos), "comparisonJobQueued": true,
	})
}

// sendComparisonTask 归档后异步执行账实比对
func (h *InvHandler) sendComparisonTask(taskID int64) {
	go func() {
		if err := inventorycmp.Run(context.Background(), h.DB, h.AssetRPC, taskID); err != nil {
			log.Printf("comparison failed: task_id=%d err=%v", taskID, err)
		}
	}()
}

// POST /inventory/tasks/:id/compare — 手动触发比对（用于 status=2 卡住的任务）
func (h *InvHandler) Compare(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/compare")
	if taskID == 0 {
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
		writeOK(w, map[string]any{"taskId": taskID, "status": task.Status, "alreadyDone": true})
		return
	}
	if err := inventorycmp.Run(r.Context(), h.DB, h.AssetRPC, taskID); err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}
	updated, _ := im.FindTask(r.Context(), taskID)
	status := int16(3)
	if updated != nil {
		status = updated.Status
	}
	writeOK(w, map[string]any{"taskId": taskID, "status": status, "compared": true})
}

func getCellStr(cells map[string]any, key string) string {
	if v, ok := cells[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}


// GET /inventory/tasks/:id/records
func (h *InvHandler) Records(w http.ResponseWriter, r *http.Request) {
	id := parseIDForAction(r.URL.Path, "/tasks/", "/records")
	if id == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var diffFilter *int16
	if s := q.Get("diffStatus"); s != "" {
		v, _ := strconv.Atoi(s)
		st := int16(v)
		diffFilter = &st
	}

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}

	records, err := im.ListRecords(r.Context(), id, diffFilter)
	if err != nil {
		writeErr(w, err)
		return
	}

	assetMap := h.fetchScopeAssetMap(r.Context(), task.ScopeDeptID)
	all := make([]map[string]any, 0, len(records))
	for _, rec := range records {
		all = append(all, recordToMap(rec, assetMap))
	}

	total := len(all)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	writeOK(w, map[string]any{
		"list":     all[start:end],
		"page":     page,
		"pageSize": pageSize,
		"total":    total,
	})
}

func (h *InvHandler) fetchScopeAssetMap(ctx context.Context, scopeDeptID int64) map[int64]map[string]any {
	result := make(map[int64]map[string]any)
	if h.AssetRPC == "" {
		return result
	}
	body, _ := json.Marshal(map[string]any{"deptIds": []int64{scopeDeptID}})
	resp, err := http.Post(h.AssetRPC+"/asset.rpc/ListAssetsByDeptIds", "application/json", bytes.NewReader(body))
	if err != nil {
		return result
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var rpcResult struct {
		Assets []map[string]any `json:"assets"`
	}
	if json.Unmarshal(respBody, &rpcResult) != nil {
		return result
	}
	for _, a := range rpcResult.Assets {
		id := rpcAssetIDInt64(a)
		if id > 0 {
			result[id] = a
		}
	}
	return result
}

func recordToMap(rec model.InventoryRecord, assets map[int64]map[string]any) map[string]any {
	assetNo := "-"
	name := rec.FoundAssetDesc
	bookLocation := "-"
	if rec.AssetID != nil {
		if a, ok := assets[*rec.AssetID]; ok {
			if no := rpcAssetNo(a); no != "" {
				assetNo = no
			}
			if n := rpcAssetName(a); n != "" {
				name = n
			}
			if loc := rpcAssetLocation(a); loc != "" {
				bookLocation = loc
			}
		}
	}
	if name == "" {
		name = "-"
	}
	return map[string]any{
		"assetNo":        assetNo,
		"name":           name,
		"bookLocation":   bookLocation,
		"actualLocation": rec.ActualLocation,
		"diffStatus":     rec.DiffStatus,
	}
}

// GET /inventory/tasks/:id/drafts — 读取已保存草稿（师生仅自己的）
func (h *InvHandler) Drafts(w http.ResponseWriter, r *http.Request) {
	taskID := parseIDForAction(r.URL.Path, "/tasks/", "/drafts")
	if taskID == 0 {
		writeErr(w, errx.ErrInvalidParam)
		return
	}

	roleLevel, _ := middleware.GetRoleLevel(r.Context())
	uid, _ := middleware.GetUID(r.Context())
	deptID, _ := middleware.GetDeptID(r.Context())

	im := model.NewInvModel(h.DB)
	task, err := im.FindTask(r.Context(), taskID)
	if err != nil {
		writeErr(w, err)
		return
	}

	if roleLevel == 3 {
		ok, _ := im.IsAssignee(r.Context(), taskID, uid)
		if !ok {
			writeErr(w, errx.ErrNotAssigned)
			return
		}
	} else if roleLevel == 2 {
		allowed := false
		for _, id := range h.deptSubtreeIDs(r.Context(), deptID) {
			if id == task.ScopeDeptID {
				allowed = true
				break
			}
		}
		if !allowed {
			writeErr(w, errx.ErrDeptAccessDenied)
			return
		}
	}

	if h.DraftCol == nil {
		writeOK(w, map[string]any{"list": []any{}, "total": 0})
		return
	}

	filter := bson.M{"task_id": taskID}
	if roleLevel == 3 {
		filter["operator_id"] = uid
	}

	cursor, err := h.DraftCol.Find(r.Context(), filter)
	if err != nil {
		writeErr(w, errx.ErrInternal)
		return
	}
	defer cursor.Close(r.Context())

	type draftDoc struct {
		AssetNo       string         `bson:"asset_no"`
		ModifiedCells map[string]any `bson:"modified_cells"`
		UpdatedAt     time.Time      `bson:"updated_at"`
	}

	list := make([]map[string]any, 0)
	for cursor.Next(r.Context()) {
		var doc draftDoc
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		list = append(list, map[string]any{
			"assetNo":       doc.AssetNo,
			"modifiedCells": doc.ModifiedCells,
			"updatedAt":     formatDraftUpdatedAt(doc.UpdatedAt),
		})
	}

	writeOK(w, map[string]any{"list": list, "total": len(list)})
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
		body, _ := json.Marshal(map[string]any{"deptIds": []int64{task.ScopeDeptID}})
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
						"assetId":      rpcAssetID(a),
						"assetNo":      rpcAssetNo(a),
						"name":         rpcAssetName(a),
						"bookLocation": rpcAssetLocation(a),
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

// rpcAsset* 兼容 asset-rpc 返回的 PascalCase 与 snake_case 字段名
func rpcAssetNo(a map[string]any) string {
	if v, ok := a["AssetNo"].(string); ok {
		return v
	}
	if v, ok := a["asset_no"].(string); ok {
		return v
	}
	return ""
}

func rpcAssetID(a map[string]any) any {
	if v, ok := a["ID"]; ok {
		return v
	}
	return a["id"]
}

func rpcAssetIDInt64(a map[string]any) int64 {
	if v, ok := a["ID"].(float64); ok {
		return int64(v)
	}
	if v, ok := a["id"].(float64); ok {
		return int64(v)
	}
	return 0
}

func rpcAssetName(a map[string]any) string {
	if v, ok := a["Name"].(string); ok {
		return v
	}
	if v, ok := a["name"].(string); ok {
		return v
	}
	return ""
}

func rpcAssetLocation(a map[string]any) string {
	if v, ok := a["Location"].(string); ok {
		return v
	}
	if v, ok := a["location"].(string); ok {
		return v
	}
	return ""
}

func formatDraftUpdatedAt(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

func parseDraftUpdatedAt(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02T15:04:05.000Z", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func formatSurplusDesc(name, bookLoc, actualLoc string) string {
	parts := make([]string, 0, 3)
	if name != "" {
		parts = append(parts, name)
	}
	if bookLoc != "" && bookLoc != "-" {
		parts = append(parts, "账面:"+bookLoc)
	}
	if actualLoc != "" {
		parts = append(parts, "@"+actualLoc)
	}
	if len(parts) == 0 {
		return "盘盈资产"
	}
	return strings.Join(parts, " ")
}

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
	s := path
	if action != "" {
		if i := strings.Index(s, action); i >= 0 {
			s = s[:i]
		}
	}
	idx := strings.Index(s, prefix)
	if idx < 0 {
		return 0
	}
	s = strings.TrimSuffix(s[idx+len(prefix):], "/")
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}

// 避免未使用 import
var _ = fmt.Sprintf
var _ = primitive.NilObjectID
