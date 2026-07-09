package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/sisyphus550/assets-db/backend/pkg/errx"
	"github.com/sisyphus550/assets-db/backend/service/asset/logic"
)

func main() {
	dsn := getEnv("MYSQL_DSN", "fams:fams_dev_pass@tcp(localhost:3306)/fams_asset?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/asset.rpc/GetAsset", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ AssetID int64 `json:"assetId"` }
		json.NewDecoder(r.Body).Decode(&req)
		a, err := logic.GetAsset(r.Context(), db, req.AssetID)
		if err != nil { rpcErr(w, err); return }
		rpcOK(w, map[string]any{"asset": a})
	})

	mux.HandleFunc("/asset.rpc/CheckAssetForWorkflow", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AssetID      int64 `json:"assetId"`
			WorkflowType int32 `json:"workflowType"`
			RequesterID  int64 `json:"requesterId"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		ok, deptID, reason := logic.CheckAssetForWorkflow(r.Context(), db, req.AssetID, req.WorkflowType, req.RequesterID)
		rpcOK(w, map[string]any{"ok": ok, "departmentId": deptID, "rejectReason": reason})
	})

	mux.HandleFunc("/asset.rpc/ChangeAssetStatus", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RequestID      int64  `json:"requestId"`
			AssetID        int64  `json:"assetId"`
			TargetStatus   int8   `json:"targetStatus"`
			AssignedUserID *int64 `json:"assignedUserId"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if err := logic.ChangeAssetStatus(r.Context(), db, req.AssetID, req.TargetStatus, req.AssignedUserID); err != nil {
			rpcErr(w, err)
			return
		}
		rpcOK(w, map[string]any{"ok": true})
	})

	mux.HandleFunc("/asset.rpc/ListAssetsByDeptIds", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ DeptIDs []int64 `json:"deptIds"` }
		json.NewDecoder(r.Body).Decode(&req)
		list, err := logic.ListByDeptIDs(r.Context(), db, req.DeptIDs)
		if err != nil { rpcErr(w, err); return }
		rpcOK(w, map[string]any{"assets": list})
	})

	mux.HandleFunc("/asset.rpc/CheckAssetScope", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AssetNo       string  `json:"assetNo"`
			ScopeDeptID   int64   `json:"scopeDeptId"`
			ScopeDeptIDs  []int64 `json:"scopeDeptIds"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		scopeIDs := req.ScopeDeptIDs
		if len(scopeIDs) == 0 && req.ScopeDeptID != 0 {
			scopeIDs = []int64{req.ScopeDeptID}
		}
		inScope, err := logic.CheckAssetScope(r.Context(), db, req.AssetNo, scopeIDs)
		if err != nil {
			rpcOK(w, map[string]any{"inScope": false})
			return
		}
		rpcOK(w, map[string]any{"inScope": inScope})
	})

	addr := ":" + getEnv("RPC_PORT", "8082")
	log.Printf("asset-rpc listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

func rpcOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func rpcErr(w http.ResponseWriter, err error) {
	code, _, msg := errx.ToHTTPError(err)
	rpcOK(w, map[string]any{"error": map[string]any{"code": code, "message": msg}})
}
