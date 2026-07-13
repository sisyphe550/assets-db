package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/inventory/api/internal/handler"
)

func main() {
	pgDSN := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	mongoDB := getEnv("MONGO_DB", "fams_inventory")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	assetRPC := getEnv("ASSET_RPC_URL", "http://localhost:8082")

	db, err := sql.Open("postgres", pgDSN)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	mongoClient, err := mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Printf("WARNING: MongoDB not available: %v", err)
	} else {
		log.Printf("MongoDB connected: %s", mongoURI)
	}
	if mongoClient != nil {
		defer mongoClient.Disconnect(context.Background())
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("WARNING: Redis not available: %v", err)
		rdb = nil
	} else {
		log.Printf("Redis connected: %s", redisAddr)
	}

	draftCol := mongoClient.Database(mongoDB).Collection("inventory_draft")

	h := handler.NewInvHandler(db, draftCol, rdb, assetRPC)

	mux := http.NewServeMux()
	authMux := http.NewServeMux()

	authMux.HandleFunc("/api/v1/inventory/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateTask(w, r)
		case http.MethodGet:
			h.ListTasks(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	authMux.HandleFunc("/api/v1/inventory/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/submit") {
			h.Submit(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/archive") {
			h.Archive(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/compare") {
			if r.Method == http.MethodPost {
				h.Compare(w, r)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else if strings.Contains(r.URL.Path, "/conflicts") {
			if strings.HasSuffix(r.URL.Path, "/resolve") {
				if r.Method == http.MethodPost {
					h.ResolveConflict(w, r)
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			} else if r.Method == http.MethodGet {
				h.ListConflicts(w, r)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		} else if strings.HasSuffix(r.URL.Path, "/records") {
			h.Records(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/drafts") {
			h.Drafts(w, r)
		} else if strings.Contains(r.URL.Path, "/expected-assets") {
			h.ExpectedAssets(w, r)
		} else if r.Method == http.MethodGet {
			h.GetTask(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	authHandler := middleware.JWTAuth(getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod"), nil)(authMux)
	mux.Handle("/", authHandler)

	addr := ":" + getEnv("PORT", "8891")
	log.Printf("inventory-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
