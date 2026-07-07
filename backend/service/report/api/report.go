package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
	_ "github.com/go-sql-driver/mysql"

	"github.com/sisyphus550/assets-db/backend/pkg/middleware"
	"github.com/sisyphus550/assets-db/backend/service/report/api/internal/handler"
)

func main() {
	pgDSN := getEnv("POSTGRES_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_core?sslmode=disable")
	reportDSN := getEnv("POSTGRES_REPORT_DSN", "postgres://fams:fams_dev_pass@localhost:5432/fams_report?sslmode=disable")
	mysqlDSN := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true")

	pg, _ := sql.Open("postgres", pgDSN)
	reportDB, _ := sql.Open("postgres", reportDSN)
	mysql, _ := sql.Open("mysql", mysqlDSN)
	defer pg.Close()
	defer reportDB.Close()
	defer mysql.Close()

	h := handler.NewReportHandler(pg, reportDB, mysql)

	mux := http.NewServeMux()
	authMux := http.NewServeMux()

	authMux.HandleFunc("/api/v1/report/assets/by-dept", h.AssetsByDept)
	authMux.HandleFunc("/api/v1/report/inventory/diff/", h.InventoryDiff)
	authMux.HandleFunc("/api/v1/report/export", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/download") {
			h.ExportDownload(w, r)
		} else if r.Method == http.MethodPost {
			h.Export(w, r)
		} else {
			h.ExportStatus(w, r)
		}
	})

	authHandler := middleware.JWTAuth(getEnv("JWT_ACCESS_SECRET", "dev_access_secret_change_me_in_prod"), nil)(authMux)
	mux.Handle("/", authHandler)

	addr := ":" + getEnv("PORT", "8892")
	log.Printf("report-api listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
