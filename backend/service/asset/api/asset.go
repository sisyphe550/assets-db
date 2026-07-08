package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/asset/api/internal/handler"
)

func main() {
	dsn := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	h := handler.NewAssetHandlerWithUserAPI(db, getEnv("USER_API_URL", "http://localhost:8888"))

	mux := http.NewServeMux()
	authMux := http.NewServeMux()

	authMux.HandleFunc("/api/v1/asset/assets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost: h.Create(w, r)
		case http.MethodGet: h.List(w, r)
		default: w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	authMux.HandleFunc("/api/v1/asset/assets/shared", h.SharedList)
	authMux.HandleFunc("/api/v1/asset/assets/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/shared") {
			h.SharedList(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet: h.Detail(w, r)
		case http.MethodPut: h.Update(w, r)
		case http.MethodDelete: h.Delete(w, r)
		default: w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	authHandler := middleware.JWTAuth(getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod"), nil)(authMux)
	mux.Handle("/", authHandler)

	addr := ":" + getEnv("PORT", "8889")
	log.Printf("asset-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
