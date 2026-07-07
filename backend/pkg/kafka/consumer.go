package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// Consumer Kafka 消费者封装
type Consumer struct {
	reader *kafka.Reader
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

// NewConsumer 创建消费者
func NewConsumer(cfg ConsumerConfig) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        cfg.Brokers,
			Topic:          cfg.Topic,
			GroupID:        cfg.GroupID,
			MinBytes:       10,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
		}),
	}
}

// MessageHandler 消息处理函数
type MessageHandler func(ctx context.Context, key, value []byte) error

// Consume 开始消费（阻塞，应由 goroutine 调用）
func (c *Consumer) Consume(ctx context.Context, handler MessageHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("[kafka] fetch error: %v", err)
			time.Sleep(time.Second)
			continue
		}

		// 从 Kafka Headers 恢复 trace context
		msgCtx := extractTraceContext(ctx, msg.Headers)

		if err := handler(msgCtx, msg.Key, msg.Value); err != nil {
			log.Printf("[kafka] handle error: key=%s err=%v", string(msg.Key), err)
			// 业务错误不阻塞，ACK 后继续
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[kafka] commit error: %v", err)
		}
	}
}

// Close 关闭消费者
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// extractTraceContext 从 Kafka Headers 恢复 W3C traceparent
func extractTraceContext(ctx context.Context, headers []kafka.Header) context.Context {
	propagator := otel.GetTextMapPropagator()
	carrier := &headerCarrier{headers: headers}
	return propagator.Extract(ctx, carrier)
}

// ErrConsumerClosed 消费者已关闭
var ErrConsumerClosed = errx.ErrServiceUnavailable
