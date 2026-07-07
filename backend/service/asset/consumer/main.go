// Kafka Consumer: 消费 fams-asset-lifecycle-events，同步台账状态
// 对应 01-desgin.md §5.2 最后一步
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"

	assetlogic "github.com/sisyphus550/assets-db/backend/service/asset/logic"
)

type lifecycleEvent struct {
	EventType      string `json:"event_type"`
	RequestID      int64  `json:"request_id"`
	AssetID        int64  `json:"asset_id"`
	TargetStatus   int8   `json:"target_status"`
	AssignedUserID int64  `json:"assigned_user_id"`
}

func main() {
	dsn := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true")
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9094")
	kafkaTopic := getEnv("KAFKA_TOPIC", "fams-asset-lifecycle-events")
	groupID := getEnv("KAFKA_GROUP", "asset-consumer")

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
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

	log.Printf("asset-consumer: listening %s/%s (group=%s)", kafkaBroker, kafkaTopic, groupID)

	for {
		select {
		case <-ctx.Done():
			log.Println("asset-consumer: shutting down")
			return
		default:
		}

		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil { return }
			log.Printf("fetch error: %v", err)
			continue
		}

		var event lifecycleEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("invalid message: %v", err)
			reader.CommitMessages(ctx, msg)
			continue
		}

		// 幂等去重
		dup, err := assetlogic.DedupEvent(context.Background(), db, event.RequestID, event.EventType)
		if err != nil {
			log.Printf("dedup error: %v", err)
			continue // 不 ACK，等待重试
		}
		if dup {
			reader.CommitMessages(ctx, msg)
			continue
		}

		// 变更资产状态
		var userID *int64
		if event.AssignedUserID != 0 {
			userID = &event.AssignedUserID
		}
		if err := assetlogic.ChangeAssetStatus(context.Background(), db, event.AssetID, event.TargetStatus, userID); err != nil {
			log.Printf("change status error: request=%d asset=%d err=%v", event.RequestID, event.AssetID, err)
		} else {
			log.Printf("asset %d status→%d (event=%s request=%d)", event.AssetID, event.TargetStatus, event.EventType, event.RequestID)
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("commit error: %v", err)
		}
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

