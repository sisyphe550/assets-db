package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/user/api/internal/handler"
)

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	accessSecret := getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod")
	refreshSecret := getEnv("JWT_REFRESH_SECRET", "dev_refresh_secret_change_me_in_prod")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("WARNING: Redis not available at %s, blacklist disabled: %v", redisAddr, err)
		rdb = nil
	} else {
		log.Printf("Redis connected: %s", redisAddr)
	}

	h := handler.NewUserHandler(db, rdb, accessSecret, refreshSecret, 2*time.Hour, 24*time.Hour)

	mux := http.NewServeMux()

	// 无需鉴权
	mux.HandleFunc("/api/v1/user/login", h.Login)
	mux.HandleFunc("/api/v1/user/refresh", h.Refresh)

	// 需要鉴权（含黑名单检查）
	authMux := http.NewServeMux()
	authMux.HandleFunc("/api/v1/user/logout", h.Logout)
	authMux.HandleFunc("/api/v1/user/me", h.Me)
	authMux.HandleFunc("/api/v1/user/departments/tree", h.DeptTree)
	authMux.HandleFunc("/api/v1/user/departments/college-subtree", h.CollegeSubtree)
	authMux.HandleFunc("/api/v1/user/departments", h.CreateDept)
	authMux.HandleFunc("/api/v1/user/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateUser(w, r)
		case http.MethodGet:
			h.ListUsers(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	authMux.HandleFunc("/api/v1/user/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && !hasSuffix(r.URL.Path, "/status") && !hasSuffix(r.URL.Path, "/force-logout") {
			h.GetUser(w, r)
		} else if r.Method == http.MethodPut && hasSuffix(r.URL.Path, "/status") {
			h.UpdateUserStatus(w, r)
		} else if r.Method == http.MethodPost && hasSuffix(r.URL.Path, "/force-logout") {
			h.ForceLogout(w, r)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// JWT 中间件带 Redis 黑名单
	authHandler := middleware.JWTAuth(accessSecret, rdb)(authMux)
	mux.Handle("/api/v1/user/", authHandler)

	addr := ":" + getEnv("PORT", "8888")
	log.Printf("user-api listening on %s (redis=%v)", addr, rdb != nil)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func hasSuffix(path, suffix string) bool {
	return len(path) >= len(suffix) && path[len(path)-len(suffix):] == suffix
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" { return v }
	return def
}

// 避免未使用 import
var _ = strings.TrimSpace
