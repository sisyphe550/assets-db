// Package outbox Transactional Outbox 投递器
// 轮询 workflow_outbox 表，将事件可靠投递至 Kafka
// 对应 01-desgin.md §5.2 和 02-plan.md P1.6
package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// Event 发件箱事件
type Event struct {
	ID           int64           `db:"id"`
	EventType    string          `db:"event_type"`
	PartitionKey string          `db:"partition_key"`
	Payload      json.RawMessage `db:"payload"`
	RetryCount   int             `db:"retry_count"`
}

// KafkaProducer Kafka 生产者接口（避免循环依赖）
type KafkaProducer interface {
	Send(ctx context.Context, topic, key string, value []byte) error
}

// Dispatcher 发件箱投递器
type Dispatcher struct {
	db       *sql.DB
	producer KafkaProducer
	topic    string
	maxRetry int // 默认 10
}

// New 创建 Dispatcher
func New(db *sql.DB, producer KafkaProducer, topic string) *Dispatcher {
	return &Dispatcher{
		db:       db,
		producer: producer,
		topic:    topic,
		maxRetry: 10,
	}
}

// PollAndSend 单次轮询与投递
// 使用 FOR UPDATE SKIP LOCKED 避免多实例重复投递
func (d *Dispatcher) PollAndSend(ctx context.Context) (int, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx,
		`SELECT id, event_type, partition_key, payload, retry_count
		 FROM workflow_outbox
		 WHERE status = 0
		 ORDER BY id
		 LIMIT 100
		 FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.EventType, &e.PartitionKey, &e.Payload, &e.RetryCount); err != nil {
			return 0, err
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sent := 0
	for _, e := range events {
		err := d.producer.Send(ctx, d.topic, e.PartitionKey, e.Payload)
		if err != nil {
			// 投递失败：增加重试计数
			_ = d.markRetry(ctx, tx, e)
			log.Printf("[outbox] send failed event=%d type=%s err=%v", e.ID, e.EventType, err)
			continue
		}
		// 投递成功：标记已投递
		_, err = tx.ExecContext(ctx,
			`UPDATE workflow_outbox SET status=1, sent_at=NOW() WHERE id=$1`, e.ID)
		if err != nil {
			return sent, err
		}
		sent++
	}

	if err := tx.Commit(); err != nil {
		return sent, err
	}
	return sent, nil
}

func (d *Dispatcher) markRetry(ctx context.Context, tx *sql.Tx, e Event) error {
	newCount := e.RetryCount + 1
	newStatus := 0
	if newCount >= d.maxRetry {
		newStatus = 2 // 死信
	}
	_, err := tx.ExecContext(ctx,
		`UPDATE workflow_outbox SET retry_count=$1, status=$2 WHERE id=$3`,
		newCount, newStatus, e.ID)
	return err
}

// Run 启动循环投递（阻塞，应由 goroutine 调用）
func (d *Dispatcher) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, err := d.PollAndSend(ctx)
			if err != nil {
				log.Printf("[outbox] poll error: %v", err)
			}
		}
	}
}

// RetryBackoff 计算指数退避时间
// retry 0 → 1s, 1 → 2s, 2 → 4s, ..., max 300s
func RetryBackoff(retryCount int) time.Duration {
	sec := math.Pow(2, float64(retryCount))
	if sec > 300 {
		sec = 300
	}
	return time.Duration(sec) * time.Second
}

// ErrDeadLetter 死信错误
var ErrDeadLetter = errx.New(50001, 500, "消息已达最大重试次数，已转入死信队列")
