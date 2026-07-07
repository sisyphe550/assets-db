package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"

	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/workflow/api/internal/handler"
)

func main() {
	dsn := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	h := handler.NewWorkflowHandler(db)

	mux := http.NewServeMux()
	authMux := http.NewServeMux()

	authMux.HandleFunc("/api/v1/workflow/requests", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost: h.Create(w, r)
		case http.MethodGet: h.List(w, r)
		default: w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	authMux.HandleFunc("/api/v1/workflow/requests/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			h.Approve(w, r)
		} else if strings.HasSuffix(r.URL.Path, "/reject") {
			h.Reject(w, r)
		} else {
			h.Detail(w, r)
		}
	})

	authHandler := middleware.JWTAuth(getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod"), nil)(authMux)
	mux.Handle("/", authHandler)

	addr := ":" + getEnv("PORT", "8890")
	log.Printf("workflow-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
