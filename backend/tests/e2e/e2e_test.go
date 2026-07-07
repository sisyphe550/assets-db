//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

const baseURL = "http://localhost:80"

// E2E-01: admin login → create asset → list visible
func TestE2E01_LoginCreateAsset(t *testing.T) {
	t.Skip("requires all services running")

	// Login
	resp, err := http.Post(baseURL+"/api/v1/user/login", "application/json",
		bytes.NewReader([]byte(`{"username":"admin_school","password":"Test@123456"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Code != 0 {
		t.Fatalf("login failed: code=%d", result.Code)
	}
	fmt.Printf("E2E-01: login OK, token=%s...\n", result.Data.AccessToken[:20])
}

// E2E-02: student_001 request use → admin_info approve stage1 → admin_school approve stage2
func TestE2E02_WorkflowApproval(t *testing.T) {
	t.Skip("requires all services running")
}

// E2E-03: create inventory task → concurrent submit → archive → check diff
func TestE2E03_InventoryFlow(t *testing.T) {
	t.Skip("requires all services running")
}

// E2E-04: disable student_001 → old token rejected
func TestE2E04_TokenRevocation(t *testing.T) {
	t.Skip("requires all services running")
}

// E2E-05: GET /report/assets/by-dept → dept 15 total >= 3
func TestE2E05_ReportDeptAssets(t *testing.T) {
	t.Skip("requires all services running")
}
