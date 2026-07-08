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

const (
	queueKey     = "fams:export:queue"
	exportDir    = "deploy/export"
)

type exportJob struct {
	JobID      int64  `json:"jobId"`
	ExportType string `json:"exportType"`
	CreatorID  int64  `json:"creatorId"`
}

func main() {
	pgDSN := getEnv("POSTGRES_REPORT_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_report?sslmode=disable")
	mysqlDSN := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")

	pg, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pg.Close()

	mysql, err := sql.Open("mysql", mysqlDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer mysql.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	// 确保导出目录存在
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		log.Fatalf("create export dir: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("export-worker: waiting for jobs on %s", queueKey)

	for {
		select {
		case <-ctx.Done():
			log.Println("export-worker: shutting down")
			return
		default:
		}

		// BRPOP 阻塞等待（超时 5s 以便检查 ctx）
		result, err := rdb.BRPop(ctx, 5*time.Second, queueKey).Result()
		if err != nil {
			if ctx.Err() != nil { return }
			continue
		}
		// result[0]=key, result[1]=value
		if len(result) < 2 {
			continue
		}

		var job exportJob
		if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
			log.Printf("invalid job: %v", err)
			continue
		}

		log.Printf("processing job %d (type=%s)", job.JobID, job.ExportType)

		// 更新状态为处理中
		pg.ExecContext(ctx, `UPDATE rpt_export_job SET status=1 WHERE id=$1`, job.JobID)

		// 生成 CSV
		filePath, genErr := generateCSV(ctx, mysql, job)
		if genErr != nil {
			log.Printf("job %d failed: %v", job.JobID, genErr)
			pg.ExecContext(ctx,
				`UPDATE rpt_export_job SET status=3, error_message=$1, finished_at=NOW() WHERE id=$2`,
				genErr.Error(), job.JobID)
			continue
		}

		// 更新状态为完成
		pg.ExecContext(ctx,
			`UPDATE rpt_export_job SET status=2, file_path=$1, finished_at=NOW() WHERE id=$2`,
			filePath, job.JobID)
		log.Printf("job %d completed: %s", job.JobID, filePath)
	}
}

func generateCSV(ctx context.Context, mysql *sql.DB, job exportJob) (string, error) {
	fileName := fmt.Sprintf("export_%d_%s.csv", job.JobID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(exportDir, fileName)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// 写入 BOM（Excel 兼容中文）
	f.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(f)
	defer cw.Flush()

	switch job.ExportType {
	case "asset_list":
		cw.Write([]string{"ID", "资产编号", "名称", "类别", "价格", "购置时间", "地点", "状态"})
		rows, err := mysql.QueryContext(ctx,
			`SELECT id, asset_no, name, category, price, purchase_time, location,
			        CASE status WHEN 1 THEN '在库' WHEN 2 THEN '领用中' WHEN 3 THEN '维修中' WHEN 4 THEN '已报废' END
			 FROM asset_ledger WHERE deleted_at IS NULL ORDER BY id`)
		if err != nil {
			return "", fmt.Errorf("query assets: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id int64
			var assetNo, name, category, location, statusLabel string
			var price float64
			var purchaseTime time.Time
			rows.Scan(&id, &assetNo, &name, &category, &price, &purchaseTime, &location, &statusLabel)
			cw.Write([]string{
				fmt.Sprintf("%d", id), assetNo, name, category,
				fmt.Sprintf("%.2f", price), purchaseTime.Format("2006-01-02"), location, statusLabel,
			})
		}
	default:
		cw.Write([]string{"NOTE", "Unsupported export type: " + job.ExportType})
	}

	return filePath, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
