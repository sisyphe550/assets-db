//go:build integration
// +build integration

package kafkatest

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

func TestKafkaProduceConsume(t *testing.T) {
	topic := "fams-test-" + time.Now().Format("150405")
	brokers := []string{"localhost:9094"}

	// Create topic
	conn, err := kafka.Dial("tcp", brokers[0])
	if err != nil {
		t.Skipf("Kafka not available: %v", err)
	}
	defer conn.Close()

	if err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	}); err != nil {
		t.Fatal("create topic:", err)
	}
	defer conn.DeleteTopics(topic)

	// Produce
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  brokers,
		Topic:    topic,
		Balancer: &kafka.Hash{},
	})
	defer writer.Close()

	testMsg := map[string]any{
		"event_type": "ASSET_USE_APPROVED",
		"asset_id":   501,
		"request_id": 1,
	}
	body, _ := json.Marshal(testMsg)
	err = writer.WriteMessages(context.Background(), kafka.Message{
		Key:   []byte("501"),
		Value: body,
	})
	if err != nil {
		t.Fatal("write:", err)
	}
	t.Log("produced message")

	// Consume
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: "test-group-" + time.Now().Format("150405"),
		MaxWait: 2 * time.Second,
	})
	defer reader.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msg, err := reader.ReadMessage(ctx)
	if err != nil {
		t.Fatal("read:", err)
	}

	t.Logf("consumed: key=%s value=%s", string(msg.Key), string(msg.Value))

	var result map[string]any
	json.Unmarshal(msg.Value, &result)
	if result["event_type"] != "ASSET_USE_APPROVED" {
		t.Errorf("unexpected event_type: %v", result["event_type"])
	}
}
