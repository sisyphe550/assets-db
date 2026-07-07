// ComparisonWorker: 消费盘点比对任务，执行账面 vs 实地比对
// 算法对应 07-inventory-ops.md §6
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"

	"github.com/sisyphus550/assets-db/backend/pkg/strnorm"
)

type comparisonTask struct {
	TaskID    int64 `json:"task_id"`
	Timestamp int64 `json:"timestamp"`
}

type assetInfo struct {
	ID       int64  `json:"id"`
	Location string `json:"location"`
}

func main() {
	pgDSN := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9094")
	kafkaTopic := getEnv("KAFKA_TOPIC", "fams-inventory-comparison-tasks")
	groupID := getEnv("KAFKA_GROUP", "comparison-worker")
	assetRPC := getEnv("ASSET_RPC_URL", "http://localhost:8082")

	db, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    kafkaTopic,
		GroupID:  groupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("comparison-worker: listening %s/%s (asset-rpc=%s)", kafkaBroker, kafkaTopic, assetRPC)

	for {
		select {
		case <-ctx.Done():
			log.Println("comparison-worker: shutting down")
			return
		default:
		}

		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil { return }
			log.Printf("fetch error: %v", err)
			continue
		}

		var task comparisonTask
		if err := json.Unmarshal(msg.Value, &task); err != nil {
			log.Printf("invalid message: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("processing task_id=%d", task.TaskID)
		if err := runComparison(context.Background(), db, assetRPC, task.TaskID); err != nil {
			log.Printf("comparison error: task=%d err=%v", task.TaskID, err)
		}

		reader.CommitMessages(ctx, msg)
	}
}

// runComparison 执行核心比对逻辑
func runComparison(ctx context.Context, db *sql.DB, assetRPC string, taskID int64) error {
	// 1. 查询该任务下所有待比对的记录
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
		ActualLocation *string
	}
	var records []record
	for rows.Next() {
		var r record
		rows.Scan(&r.ID, &r.TaskID, &r.AssetID, &r.IsScanned, &r.ActualLocation)
		records = append(records, r)
	}
	rows.Close()

	if len(records) == 0 {
		log.Printf("task %d: no records to compare", taskID)
		db.ExecContext(ctx, `UPDATE inventory_task SET status=3 WHERE id=$1`, taskID)
		return nil
	}

	var match, surplus, loss int

	// 2. 逐条比对
	for _, r := range records {
		// 规则 A: asset_id IS NULL → 盘盈
		if r.AssetID == nil {
			db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=2 WHERE id=$1`, r.ID)
			surplus++
			continue
		}

		// 规则 B: is_scanned=0 → 盘亏
		if r.IsScanned == 0 {
			db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=3 WHERE id=$1`, r.ID)
			loss++
			continue
		}

		// 规则 C: is_scanned=1 → 比对位置
		bookAsset, err := fetchAsset(assetRPC, *r.AssetID)
		if err != nil {
			log.Printf("fetch asset %d error: %v", *r.AssetID, err)
			continue // 跳过此条，下次重试
		}

		actualLoc := ""
		if r.ActualLocation != nil {
			actualLoc = *r.ActualLocation
		}

		if strnorm.Equal(actualLoc, bookAsset.Location) {
			db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=1 WHERE id=$1`, r.ID)
			match++
		} else {
			db.ExecContext(ctx, `UPDATE inventory_record SET diff_status=3 WHERE id=$1`, r.ID)
			loss++
		}
	}

	// 3. 更新任务状态为已完成
	db.ExecContext(ctx, `UPDATE inventory_task SET status=3 WHERE id=$1`, taskID)

	// 4. 写入汇总到报表库（fams_report）
	upsertSummary(ctx, db, taskID, match, surplus, loss)

	log.Printf("task %d: match=%d surplus=%d loss=%d (total=%d)", taskID, match, surplus, loss, len(records))
	return nil
}

// fetchAsset 从 asset-rpc 获取账面资产信息
func fetchAsset(assetRPC string, assetID int64) (*assetInfo, error) {
	body, _ := json.Marshal(map[string]int64{"assetId": assetID})
	resp, err := http.Post(assetRPC+"/asset.rpc/GetAsset", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Asset struct {
			ID       int64  `json:"id"`
			Location string `json:"location"`
		} `json:"asset"`
	}
	json.Unmarshal(respBody, &result)

	// asset-rpc wraps response differently, try alternative
	if result.Asset.ID == 0 {
		var alt struct {
			ID       int64  `json:"id"`
			Location string `json:"location"`
		}
		json.Unmarshal(respBody, &alt)
		return &assetInfo{ID: alt.ID, Location: alt.Location}, nil
	}

	return &assetInfo{ID: result.Asset.ID, Location: result.Asset.Location}, nil
}

// upsertSummary 写入或更新盘点差异汇总
func upsertSummary(ctx context.Context, db *sql.DB, taskID int64, match, surplus, loss int) {
	// fams_report 库表通过 dblink 或同库写入（简化：直接写 fams_core 中的同名表）
	// 实际部署时 fams_report 是独立库，此处通过 PG 连接写入
	db.ExecContext(ctx,
		`INSERT INTO rpt_inventory_diff_summary (task_id, match_count, surplus_count, loss_count, updated_at)
		 VALUES ($1,$2,$3,$4,NOW())
		 ON CONFLICT (task_id) DO UPDATE SET match_count=$2, surplus_count=$3, loss_count=$4, updated_at=NOW()`,
		taskID, match, surplus, loss)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

// 避免未使用 import
var _ = strconv.Itoa
