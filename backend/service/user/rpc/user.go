package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/sisyphus550/assets-db/backend/service/user/rpc/internal/server"
)

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	accessSecret := getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	s := &server.RPCServer{
		DB:           db,
		AccessSecret: accessSecret,
	}

	addr := ":" + getEnv("RPC_PORT", "8081")
	log.Printf("user-rpc listening on %s", addr)
	if err := s.Start(addr); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
