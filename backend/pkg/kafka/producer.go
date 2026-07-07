// Package kafka Kafka 生产者/消费者封装
// 包含 traceparent header 透传
package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
)

// Producer Kafka 生产者封装
type Producer struct {
	writer *kafka.Writer
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	Brokers []string
	Topic   string
}

// NewProducer 创建生产者
func NewProducer(cfg ProducerConfig) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     &kafka.Hash{}, // 按 key 哈希分区
			BatchTimeout: 10 * time.Millisecond,
			Async:        false,
		},
	}
}

// Send 发送消息到 Kafka
// key: 分区键（如 asset_id 字符串）
// value: JSON 消息体
func (p *Producer) Send(ctx context.Context, key string, value []byte) error {
	// 注入 traceparent header
	headers := injectTraceContext(ctx)

	msg := kafka.Message{
		Key:     []byte(key),
		Value:   value,
		Headers: headers,
		Time:    time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return errx.ErrServiceUnavailable
	}
	return nil
}

// SendJSON 发送 JSON 对象消息
func (p *Producer) SendJSON(ctx context.Context, key string, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal kafka message: %w", err)
	}
	return p.Send(ctx, key, body)
}

// Close 关闭生产者
func (p *Producer) Close() error {
	return p.writer.Close()
}

// injectTraceContext 将 W3C traceparent 写入 Kafka Headers
func injectTraceContext(ctx context.Context) []kafka.Header {
	propagator := otel.GetTextMapPropagator()
	carrier := &headerCarrier{headers: make([]kafka.Header, 0)}
	propagator.Inject(ctx, carrier)
	return carrier.headers
}

type headerCarrier struct {
	headers []kafka.Header
}

func (c *headerCarrier) Get(key string) string {
	for _, h := range c.headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *headerCarrier) Set(key, value string) {
	c.headers = append(c.headers, kafka.Header{
		Key:   key,
		Value: []byte(value),
	})
}

func (c *headerCarrier) Keys() []string {
	keys := make([]string, len(c.headers))
	for i, h := range c.headers {
		keys[i] = h.Key
	}
	return keys
}

// 确保 headerCarrier 实现 propagation.TextMapCarrier
var _ propagation.TextMapCarrier = (*headerCarrier)(nil)

