package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"

	"github.com/sisyphus550/assets-db/backend/pkg/outbox"
)

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	kafkaBroker := getEnv("KAFKA_BROKER", "localhost:9094")
	kafkaTopic := getEnv("KAFKA_TOPIC", "fams-asset-lifecycle-events")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	producer := &kafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(kafkaBroker),
			Topic:        kafkaTopic,
			Balancer:     &kafka.Hash{},
			BatchTimeout: 10 * time.Millisecond,
		},
	}
	defer producer.writer.Close()

	disp := outbox.New(db, producer, kafkaTopic)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("outbox-dispatcher: polling %s → Kafka(%s/%s) every 5s", dsn, kafkaBroker, kafkaTopic)

	// 启动轮询
	go disp.Run(ctx, 5*time.Second)

	<-ctx.Done()
	log.Println("outbox-dispatcher: shutting down")
}

type kafkaProducer struct {
	writer *kafka.Writer
}

func (p *kafkaProducer) Send(ctx context.Context, topic, key string, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
