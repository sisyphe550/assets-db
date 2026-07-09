// Export Worker: 消费 Redis 导出队列，生成 CSV 文件
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const queueKey = "fams:export:queue"

type exportJob struct {
	JobID      int64          `json:"jobId"`
	ExportType string         `json:"exportType"`
	CreatorID  int64          `json:"creatorId"`
	Params     map[string]any `json:"params"`
}

type exportDeps struct {
	reportPG *sql.DB
	corePG   *sql.DB
	mysql    *sql.DB
	exportDir string
}

func main() {
	reportDSN := getEnv("POSTGRES_REPORT_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_report?sslmode=disable")
	coreDSN := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	mysqlDSN := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	exportDir := resolveExportDir()

	reportPG, err := sql.Open("postgres", reportDSN)
	if err != nil {
		log.Fatalf("connect report postgres: %v", err)
	}
	defer reportPG.Close()

	corePG, err := sql.Open("postgres", coreDSN)
	if err != nil {
		log.Fatalf("connect core postgres: %v", err)
	}
	defer corePG.Close()

	mysql, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer mysql.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		log.Fatalf("create export dir: %v", err)
	}

	deps := exportDeps{reportPG: reportPG, corePG: corePG, mysql: mysql, exportDir: exportDir}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("export-worker: waiting for jobs on %s (dir=%s)", queueKey, exportDir)

	for {
		select {
		case <-ctx.Done():
			log.Println("export-worker: shutting down")
			return
		default:
		}

		result, err := rdb.BRPop(ctx, 5*time.Second, queueKey).Result()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		if len(result) < 2 {
			continue
		}

		var job exportJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("invalid job: %v", err)
			continue
		}

		log.Printf("processing job %d (type=%s)", job.JobID, job.ExportType)
		reportPG.ExecContext(ctx, `UPDATE rpt_export_job SET status=1 WHERE id=$1`, job.JobID)

		filePath, genErr := generateCSV(ctx, deps, job)
		if genErr != nil {
			log.Printf("job %d failed: %v", job.JobID, genErr)
			reportPG.ExecContext(ctx,
				`UPDATE rpt_export_job SET status=3, error_message=$1, finished_at=NOW() WHERE id=$2`,
				genErr.Error(), job.JobID)
			continue
		}

		reportPG.ExecContext(ctx,
			`UPDATE rpt_export_job SET status=2, file_path=$1, finished_at=NOW() WHERE id=$2`,
			filePath, job.JobID)
		log.Printf("job %d completed: %s", job.JobID, filePath)
	}
}

func generateCSV(ctx context.Context, deps exportDeps, job exportJob) (string, error) {
	fileName := fmt.Sprintf("export_%d_%s.csv", job.JobID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(deps.exportDir, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	f.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(f)
	defer cw.Flush()

	switch job.ExportType {
	case "asset_list":
		return filePath, writeAssetList(ctx, deps.mysql, cw)
	case "workflow_log":
		return filePath, writeWorkflowLog(ctx, deps.corePG, cw)
	case "inventory_diff":
		taskID := paramInt64(job.Params, "taskId")
		if taskID == 0 {
			return "", fmt.Errorf("inventory_diff 缺少 taskId 参数")
		}
		return filePath, writeInventoryDiff(ctx, deps.corePG, cw, taskID)
	default:
		return "", fmt.Errorf("unsupported export type: %s", job.ExportType)
	}
}

func writeAssetList(ctx context.Context, mysql *sql.DB, cw *csv.Writer) error {
	cw.Write([]string{"ID", "资产编号", "名称", "类别", "价格", "购置时间", "地点", "状态"})
	rows, err := mysql.QueryContext(ctx,
		`SELECT id, asset_no, name, category, price, purchase_time, location,
		        CASE status WHEN 1 THEN '在库' WHEN 2 THEN '领用中' WHEN 3 THEN '维修中' WHEN 4 THEN '已报废' END
		 FROM asset_ledger WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var assetNo, name, category, location, statusLabel string
		var price float64
		var purchaseTime time.Time
		if err := rows.Scan(&id, &assetNo, &name, &category, &price, &purchaseTime, &location, &statusLabel); err != nil {
			continue
		}
		cw.Write([]string{
			fmt.Sprintf("%d", id), assetNo, name, category,
			fmt.Sprintf("%.2f", price), purchaseTime.Format("2006-01-02"), location, statusLabel,
		})
	}
	return rows.Err()
}

func writeWorkflowLog(ctx context.Context, pg *sql.DB, cw *csv.Writer) error {
	cw.Write([]string{"工单ID", "资产ID", "申请人", "类型", "阶段", "状态", "原因", "操作", "操作人", "备注", "操作时间"})
	rows, err := pg.QueryContext(ctx, `
		SELECT w.id, w.asset_id, COALESCE(u.real_name, ''), w.type, w.current_stage, w.status, w.reason,
		       l.action, COALESCE(op.real_name, ''), COALESCE(l.comment, ''), l.operate_time
		FROM workflow_request w
		LEFT JOIN sys_user u ON u.id = w.requester_id
		LEFT JOIN workflow_log l ON l.request_id = w.id
		LEFT JOIN sys_user op ON op.id = l.operator_id
		ORDER BY w.id, l.operate_time`)
	if err != nil {
		return fmt.Errorf("query workflow: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var wfID, assetID int64
		var requester, reason, action, operator, comment string
		var wfType, stage, status int16
		var operateTime time.Time
		if err := rows.Scan(&wfID, &assetID, &requester, &wfType, &stage, &status, &reason, &action, &operator, &comment, &operateTime); err != nil {
			continue
		}
		cw.Write([]string{
			fmt.Sprintf("%d", wfID), fmt.Sprintf("%d", assetID), requester,
			workflowTypeLabel(wfType), stageLabel(stage), statusLabel(status),
			reason, action, operator, comment, operateTime.Format(time.RFC3339),
		})
	}
	return rows.Err()
}

func writeInventoryDiff(ctx context.Context, pg *sql.DB, cw *csv.Writer, taskID int64) error {
	cw.Write([]string{"任务ID", "资产ID", "盘盈描述", "实际位置", "差异状态", "是否扫描"})
	rows, err := pg.QueryContext(ctx, `
		SELECT task_id, COALESCE(asset_id::text, ''), COALESCE(found_asset_desc, ''), COALESCE(actual_location, ''),
		       diff_status, is_scanned
		FROM inventory_record WHERE task_id=$1 ORDER BY id`, taskID)
	if err != nil {
		return fmt.Errorf("query inventory records: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tid int64
		var assetID, foundDesc, actualLoc string
		var diffStatus, isScanned int16
		if err := rows.Scan(&tid, &assetID, &foundDesc, &actualLoc, &diffStatus, &isScanned); err != nil {
			continue
		}
		cw.Write([]string{
			fmt.Sprintf("%d", tid), assetID, foundDesc, actualLoc, diffStatusLabel(diffStatus),
			map[int16]string{0: "否", 1: "是"}[isScanned],
		})
	}
	return rows.Err()
}

func paramInt64(params map[string]any, key string) int64 {
	if params == nil {
		return 0
	}
	switch v := params[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	default:
		return 0
	}
}

func workflowTypeLabel(t int16) string {
	return map[int16]string{1: "领用", 2: "归还", 3: "维修", 4: "报废"}[t]
}

func stageLabel(s int16) string {
	return map[int16]string{1: "院级初审", 2: "校级终审", 3: "已归档"}[s]
}

func statusLabel(s int16) string {
	return map[int16]string{1: "进行中", 2: "已通过", 3: "已驳回"}[s]
}

func diffStatusLabel(s int16) string {
	return map[int16]string{0: "待比对", 1: "相符", 2: "盘盈", 3: "盘亏"}[s]
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func resolveExportDir() string {
	if v := os.Getenv("EXPORT_DIR"); v != "" {
		return v
	}
	for _, candidate := range []string{"deploy/export", "../../../deploy/export", "../../../../deploy/export"} {
		if abs, err := filepath.Abs(candidate); err == nil {
			if err := os.MkdirAll(abs, 0o755); err == nil {
				return abs
			}
		}
	}
	abs, _ := filepath.Abs("deploy/export")
	_ = os.MkdirAll(abs, 0o755)
	return abs
}
