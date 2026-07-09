// Package inventorycmp 盘点账实比对（07-inventory-ops.md §6）
package inventorycmp

import (
	"fmt"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/sisyphus550/assets-db/backend/pkg/strnorm"
)

type assetInfo struct {
	ID       int64
	Location string
}

// Run 执行任务下全部 diff_status=0 记录的比对，完成后将任务置为 status=3
func Run(ctx context.Context, db *sql.DB, assetRPC string, taskID int64) error {
	rows, err := db.QueryContext(ctx,
		`SELECT id, task_id, asset_id, is_scanned, actual_location, diff_status
		 FROM inventory_record WHERE task_id=$1 AND diff_status=0`, taskID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type record struct {
		ID             int64
		TaskID         int64
		AssetID        *int64
		IsScanned      int16
		ActualLocation sql.NullString
		DiffStatus     int16
	}
	var records []record
	for rows.Next() {
		var r record
		if err := rows.Scan(&r.ID, &r.TaskID, &r.AssetID, &r.IsScanned, &r.ActualLocation, &r.DiffStatus); err != nil {
			return err
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(records) == 0 {
		_, err = db.ExecContext(ctx, `UPDATE inventory_task SET status=3 WHERE id=$1 AND status=2`, taskID)
		return err
	}

	var match, surplus, loss int
	for _, r := range records {
		if r.AssetID == nil {
			_, _ = db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=2 WHERE id=$1`, r.ID)
			surplus++
			continue
		}
		if r.IsScanned == 0 {
			_, _ = db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=3 WHERE id=$1`, r.ID)
			loss++
			continue
		}

		bookAsset, err := fetchAsset(assetRPC, *r.AssetID)
		if err != nil {
			log.Printf("inventorycmp: fetch asset %d: %v", *r.AssetID, err)
			continue
		}

		actualLoc := ""
		if r.ActualLocation.Valid {
			actualLoc = r.ActualLocation.String
		}
		if strnorm.Equal(actualLoc, bookAsset.Location) {
			_, _ = db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=1 WHERE id=$1`, r.ID)
			match++
		} else {
			_, _ = db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=3 WHERE id=$1`, r.ID)
			loss++
		}
	}

	_, err = db.ExecContext(ctx, `UPDATE inventory_task SET status=3 WHERE id=$1`, taskID)
	if err != nil {
		return err
	}

	_, _ = db.ExecContext(ctx,
		`INSERT INTO rpt_inventory_diff_summary (task_id, match_count, surplus_count, loss_count, updated_at)
		 VALUES ($1,$2,$3,$4,NOW())
		 ON CONFLICT (task_id) DO UPDATE SET match_count=$2, surplus_count=$3, loss_count=$4, updated_at=NOW()`,
		taskID, match, surplus, loss)

	log.Printf("inventorycmp: task %d done match=%d surplus=%d loss=%d", taskID, match, surplus, loss)
	return nil
}

func fetchAsset(assetRPC string, assetID int64) (*assetInfo, error) {
	body, _ := json.Marshal(map[string]int64{"assetId": assetID})
	resp, err := http.Post(assetRPC+"/asset.rpc/GetAsset", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var wrapped struct {
		Asset map[string]any `json:"asset"`
	}
	if json.Unmarshal(respBody, &wrapped) == nil && wrapped.Asset != nil {
		return mapToAssetInfo(wrapped.Asset), nil
	}

	var raw map[string]any
	if json.Unmarshal(respBody, &raw) == nil {
		return mapToAssetInfo(raw), nil
	}
	return nil, fmt.Errorf("invalid asset response for id %d", assetID)
}

func mapToAssetInfo(m map[string]any) *assetInfo {
	id := toInt64(m["ID"])
	if id == 0 {
		id = toInt64(m["id"])
	}
	loc, _ := m["Location"].(string)
	if loc == "" {
		loc, _ = m["location"].(string)
	}
	return &assetInfo{ID: id, Location: loc}
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
