//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

const (
	userAPI      = "http://localhost:8888"
	assetAPI     = "http://localhost:8889"
	workflowAPI  = "http://localhost:8890"
	inventoryAPI = "http://localhost:8891"
)

// ---- helpers ----

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func login(t *testing.T, username string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": "Test@123456",
	})
	resp, err := http.Post(userAPI+"/api/v1/user/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login %s: %v", username, err)
	}
	defer resp.Body.Close()
	var r apiResp
	json.NewDecoder(resp.Body).Decode(&r)
	if r.Code != 0 {
		t.Logf("login %s body: %s", username, string(r.Data))
		t.Fatalf("login %s failed: code=%d msg=%s", username, r.Code, r.Message)
	}

	var data struct {
		AccessToken string `json:"accessToken"`
	}
	json.Unmarshal(r.Data, &data)
	return data.AccessToken
}

func doRequest(t *testing.T, method, url, token string, body any) *apiResp {
	t.Helper()
	var reqBody []byte
	if body != nil {
		reqBody, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, url, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	var r apiResp
	json.NewDecoder(resp.Body).Decode(&r)
	return &r
}

func checkServices(t *testing.T) {
	t.Helper()
	services := map[string]string{
		"user-api":      userAPI + "/api/v1/user/login",
		"asset-api":     assetAPI + "/api/v1/asset/assets",
		"workflow-api":  workflowAPI + "/api/v1/workflow/requests",
		"inventory-api": inventoryAPI + "/api/v1/inventory/tasks",
	}
	for name, url := range services {
		_, err := http.Get(url)
		if err != nil {
			t.Skipf("%s not available (%s), skipping E2E tests", name, url)
		}
	}
}

// ---- E2E-01: Login → Create Asset → List → Detail → Delete ----

func TestE2E01_LoginCreateAsset(t *testing.T) {
	checkServices(t)

	// 1. Login as admin_school
	token := login(t, "admin_school")

	// 2. Create asset
	assetNo := fmt.Sprintf("E2E-%s", time.Now().Format("150405"))
	createResp := doRequest(t, "POST", assetAPI+"/api/v1/asset/assets", token, map[string]any{
		"assetNo":      assetNo,
		"name":         "E2E测试设备",
		"category":     "设备",
		"price":        9999.99,
		"purchaseTime": "2026-07-07T00:00:00+08:00",
		"location":     "E2E实验室",
		"departmentId": 15,
		"isShared":     0,
	})
	if createResp.Code != 0 {
		t.Fatalf("create asset failed: code=%d msg=%s", createResp.Code, createResp.Message)
	}
	t.Logf("E2E-01: created asset %s", assetNo)

	// 3. List and find the created asset
	listResp := doRequest(t, "GET", assetAPI+"/api/v1/asset/assets?keyword="+assetNo, token, nil)
	if listResp.Code != 0 {
		t.Fatalf("list assets failed: code=%d", listResp.Code)
	}

	var listData struct {
		List []struct {
			ID      int64  `json:"id"`
			AssetNo string `json:"assetNo"`
		} `json:"list"`
	}
	json.Unmarshal(listResp.Data, &listData)
	if len(listData.List) == 0 {
		t.Fatal("created asset not found in list")
	}
	assetID := listData.List[0].ID
	t.Logf("E2E-01: found asset in list, id=%d", assetID)

	// 4. Detail
	detailResp := doRequest(t, "GET", fmt.Sprintf("%s/api/v1/asset/assets/%d", assetAPI, assetID), token, nil)
	if detailResp.Code != 0 {
		t.Fatalf("detail failed: code=%d", detailResp.Code)
	}
	t.Logf("E2E-01: asset detail OK")

	// 5. Soft delete
	deleteResp := doRequest(t, "DELETE", fmt.Sprintf("%s/api/v1/asset/assets/%d", assetAPI, assetID), token, nil)
	if deleteResp.Code != 0 {
		t.Fatalf("delete failed: code=%d", deleteResp.Code)
	}
	t.Logf("E2E-01: soft deleted OK")
}

// ---- E2E-02: Full workflow: request → college approve → school approve ----

func TestE2E02_WorkflowApproval(t *testing.T) {
	checkServices(t)

	studentToken := login(t, "student_001")
	adminInfoToken := login(t, "admin_info")
	adminSchoolToken := login(t, "admin_school")

	// 1. student_001 creates use request for asset 501
	createResp := doRequest(t, "POST", workflowAPI+"/api/v1/workflow/requests", studentToken, map[string]any{
		"assetId": 501,
		"type":    1,
		"reason":  "E2E测试领用",
	})
	if createResp.Code != 0 {
		t.Fatalf("create workflow failed: code=%d msg=%s", createResp.Code, createResp.Message)
	}
	var createData struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(createResp.Data, &createData)
	reqID := createData.ID
	t.Logf("E2E-02: workflow request created, id=%d", reqID)

	// 2. admin_info (college) approves stage 1
	approve1Resp := doRequest(t, "POST", fmt.Sprintf("%s/api/v1/workflow/requests/%d/approve", workflowAPI, reqID), adminInfoToken, map[string]string{
		"comment": "院级同意",
	})
	if approve1Resp.Code != 0 {
		t.Fatalf("stage1 approve failed: code=%d msg=%s", approve1Resp.Code, approve1Resp.Message)
	}
	t.Logf("E2E-02: stage 1 approved")

	// 3. admin_school (school) approves stage 2
	approve2Resp := doRequest(t, "POST", fmt.Sprintf("%s/api/v1/workflow/requests/%d/approve", workflowAPI, reqID), adminSchoolToken, map[string]string{
		"comment": "校级同意",
	})
	if approve2Resp.Code != 0 {
		t.Fatalf("stage2 approve failed: code=%d msg=%s", approve2Resp.Code, approve2Resp.Message)
	}
	t.Logf("E2E-02: stage 2 approved (archived)")

	// 4. Verify final status and audit log
	detailResp := doRequest(t, "GET", fmt.Sprintf("%s/api/v1/workflow/requests/%d", workflowAPI, reqID), studentToken, nil)
	if detailResp.Code != 0 {
		t.Fatalf("detail failed: code=%d", detailResp.Code)
	}

	var detailData struct {
		Request struct {
			Status       int16 `json:"status"`
			CurrentStage int16 `json:"currentStage"`
		} `json:"request"`
		Logs []struct {
			Action string `json:"action"`
		} `json:"logs"`
	}
	json.Unmarshal(detailResp.Data, &detailData)

	if detailData.Request.Status != 2 {
		t.Errorf("expected status=2 (approved), got %d", detailData.Request.Status)
	}
	if detailData.Request.CurrentStage != 3 {
		t.Errorf("expected currentStage=3 (archived), got %d", detailData.Request.CurrentStage)
	}
	if len(detailData.Logs) != 3 {
		t.Errorf("expected 3 audit logs, got %d", len(detailData.Logs))
	}
	t.Logf("E2E-02: final status verified, %d audit logs", len(detailData.Logs))
}

// ---- E2E-03: Inventory flow: create → archive → records ----

func TestE2E03_InventoryFlow(t *testing.T) {
	checkServices(t)

	adminToken := login(t, "admin_school")

	// 1. Create task
	taskName := fmt.Sprintf("E2E盘点-%s", time.Now().Format("150405"))
	createResp := doRequest(t, "POST", inventoryAPI+"/api/v1/inventory/tasks", adminToken, map[string]any{
		"taskName":    taskName,
		"scopeDeptId": 15,
		"startTime":   "2026-01-01T00:00:00Z",
		"endTime":     "2026-12-31T23:59:59Z",
		"assigneeIds": []int64{10003, 10004},
	})
	if createResp.Code != 0 {
		t.Fatalf("create task failed: code=%d msg=%s", createResp.Code, createResp.Message)
	}
	var createData struct {
		TaskID int64 `json:"taskId"`
	}
	json.Unmarshal(createResp.Data, &createData)
	taskID := createData.TaskID
	t.Logf("E2E-03: task created, id=%d", taskID)

	// 2. Archive
	archiveResp := doRequest(t, "POST", fmt.Sprintf("%s/api/v1/inventory/tasks/%d/archive", inventoryAPI, taskID), adminToken, map[string]bool{
		"force": true,
	})
	if archiveResp.Code != 0 {
		t.Fatalf("archive failed: code=%d msg=%s", archiveResp.Code, archiveResp.Message)
	}
	t.Logf("E2E-03: task archived")

	// 3. Check records
	recordsResp := doRequest(t, "GET", fmt.Sprintf("%s/api/v1/inventory/tasks/%d/records", inventoryAPI, taskID), adminToken, nil)
	if recordsResp.Code != 0 {
		t.Fatalf("records failed: code=%d", recordsResp.Code)
	}
	t.Logf("E2E-03: records retrieved OK")
}

// ---- E2E-04: Role-based access control ----

func TestE2E04_RBAC(t *testing.T) {
	checkServices(t)

	studentToken := login(t, "student_001")
	adminInfoToken := login(t, "admin_info")

	// 1. student_001 (role=3) can't approve workflows
	// First create a request, then try to approve it
	createResp := doRequest(t, "POST", workflowAPI+"/api/v1/workflow/requests", studentToken, map[string]any{
		"assetId": 504,
		"type":    1,
		"reason":  "E2E RBAC test",
	})
	if createResp.Code != 0 && createResp.Code != 40902 {
		t.Fatalf("create request failed: code=%d", createResp.Code)
	}

	// Try to get a request and have student approve it (should fail)
	var reqID int64
	listResp := doRequest(t, "GET", workflowAPI+"/api/v1/workflow/requests?scope=my", studentToken, nil)
	var listData struct {
		List []struct{ ID int64 `json:"id"` } `json:"list"`
	}
	json.Unmarshal(listResp.Data, &listData)
	if len(listData.List) > 0 {
		reqID = listData.List[0].ID
		approveResp := doRequest(t, "POST", fmt.Sprintf("%s/api/v1/workflow/requests/%d/approve", workflowAPI, reqID), studentToken, map[string]string{
			"comment": "尝试越权审批",
		})
		// role=3 should get 40301
		if approveResp.Code != 40301 {
			t.Errorf("student should get 40301, got %d", approveResp.Code)
		} else {
			t.Logf("E2E-04: RBAC correctly blocked student from approving")
		}
	}

	// 2. admin_info (role=2) can't create role=2 users
	createUserResp := doRequest(t, "POST", userAPI+"/api/v1/user/users", adminInfoToken, map[string]any{
		"username":     "e2e_admin_test",
		"password":     "Test@123456",
		"realName":     "E2E越权测试",
		"roleLevel":    2,
		"departmentId": 103,
	})
	if createUserResp.Code != 40301 {
		t.Errorf("admin_info should get 40301 for creating role=2 user, got %d", createUserResp.Code)
	} else {
		t.Logf("E2E-04: RBAC correctly blocked admin_info from creating role=2 user")
	}

	_ = adminInfoToken
}

// ---- E2E-05: Duplicate prevention & reject+reapply ----

func TestE2E05_DuplicateAndReject(t *testing.T) {
	checkServices(t)

	studentToken := login(t, "student_001")
	adminInfoToken := login(t, "admin_info")

	// 1. Create request on asset 502
	createResp := doRequest(t, "POST", workflowAPI+"/api/v1/workflow/requests", studentToken, map[string]any{
		"assetId": 502,
		"type":    1,
		"reason":  "E2E duplicate test",
	})
	if createResp.Code != 0 && createResp.Code != 40902 {
		t.Fatalf("create failed: code=%d", createResp.Code)
	}

	// 2. Try duplicate (should get 40902)
	if createResp.Code == 0 {
		dupResp := doRequest(t, "POST", workflowAPI+"/api/v1/workflow/requests", studentToken, map[string]any{
			"assetId": 502,
			"type":    1,
			"reason":  "重复申请",
		})
		if dupResp.Code != 40902 {
			t.Errorf("duplicate should get 40902, got %d", dupResp.Code)
		} else {
			t.Logf("E2E-05: duplicate correctly blocked with 40902")
		}

		// 3. Reject the open request
		var reqID int64
		listResp := doRequest(t, "GET", workflowAPI+"/api/v1/workflow/requests?scope=my", studentToken, nil)
		var listData struct {
			List []struct{ ID int64 `json:"id"` } `json:"list"`
		}
		json.Unmarshal(listResp.Data, &listData)
		if len(listData.List) > 0 {
			reqID = listData.List[0].ID
			rejectResp := doRequest(t, "POST", fmt.Sprintf("%s/api/v1/workflow/requests/%d/reject", workflowAPI, reqID), adminInfoToken, map[string]string{
				"comment": "驳回测试",
			})
			if rejectResp.Code != 0 {
				t.Errorf("reject failed: code=%d", rejectResp.Code)
			} else {
				t.Logf("E2E-05: rejected OK")
			}

			// 4. Re-apply after reject (should succeed)
			reapplyResp := doRequest(t, "POST", workflowAPI+"/api/v1/workflow/requests", studentToken, map[string]any{
				"assetId": 502,
				"type":    1,
				"reason":  "驳回后重新申请",
			})
			if reapplyResp.Code != 0 {
				t.Errorf("re-apply after reject should succeed, got %d", reapplyResp.Code)
			} else {
				t.Logf("E2E-05: re-apply after reject OK")
			}
		}
	}

	_ = adminInfoToken
}

// ---- E2E-06: Frontend API gaps (user list, inventory list, asset scope, workflow assetId) ----

func TestE2E06_FrontendAPIGaps(t *testing.T) {
	checkServices(t)

	adminToken := login(t, "admin_school")
	studentToken := login(t, "student_001")
	studentMeToken := login(t, "student_me")

	// 1. GET /user/users
	listUsers := doRequest(t, "GET", userAPI+"/api/v1/user/users?page=1&pageSize=10&keyword=student", adminToken, nil)
	if listUsers.Code != 0 {
		t.Fatalf("list users failed: code=%d msg=%s", listUsers.Code, listUsers.Message)
	}
	var usersData struct {
		List  []struct{ Username string `json:"username"` } `json:"list"`
		Total int `json:"total"`
	}
	json.Unmarshal(listUsers.Data, &usersData)
	if usersData.Total < 3 {
		t.Errorf("expected at least 3 students in user list, got total=%d", usersData.Total)
	}
	t.Logf("E2E-06: user list OK, total=%d", usersData.Total)

	// 2. GET /user/users/:id
	getUser := doRequest(t, "GET", userAPI+"/api/v1/user/users/10003", adminToken, nil)
	if getUser.Code != 0 {
		t.Fatalf("get user failed: code=%d", getUser.Code)
	}
	t.Logf("E2E-06: get user by id OK")

	// 3. GET /inventory/tasks
	listTasks := doRequest(t, "GET", inventoryAPI+"/api/v1/inventory/tasks?page=1", adminToken, nil)
	if listTasks.Code != 0 {
		t.Fatalf("list inventory tasks failed: code=%d msg=%s", listTasks.Code, listTasks.Message)
	}
	t.Logf("E2E-06: inventory task list OK")

	// 4. Create task and GET /inventory/tasks/:id
	taskName := fmt.Sprintf("E2E-Gap-%s", time.Now().Format("150405"))
	createTask := doRequest(t, "POST", inventoryAPI+"/api/v1/inventory/tasks", adminToken, map[string]any{
		"taskName":    taskName,
		"scopeDeptId": 15,
		"startTime":   "2026-01-01T00:00:00Z",
		"endTime":     "2026-12-31T23:59:59Z",
		"assigneeIds": []int64{10003},
	})
	if createTask.Code != 0 {
		t.Fatalf("create task failed: code=%d", createTask.Code)
	}
	var taskData struct {
		TaskID             int64 `json:"taskId"`
		ExpectedAssetCount int   `json:"expectedAssetCount"`
		AssigneeIds        []int64 `json:"assigneeIds"`
	}
	json.Unmarshal(createTask.Data, &taskData)
	if taskData.TaskID == 0 {
		t.Fatal("taskId missing in create response")
	}
	getTask := doRequest(t, "GET", fmt.Sprintf("%s/api/v1/inventory/tasks/%d", inventoryAPI, taskData.TaskID), adminToken, nil)
	if getTask.Code != 0 {
		t.Fatalf("get task failed: code=%d", getTask.Code)
	}

	// 5. Student assigned task list
	assignedTasks := doRequest(t, "GET", inventoryAPI+"/api/v1/inventory/tasks?scope=assigned", studentToken, nil)
	if assignedTasks.Code != 0 {
		t.Fatalf("assigned tasks failed: code=%d", assignedTasks.Code)
	}
	t.Logf("E2E-06: assigned task list OK")

	// 6. GET /asset/assets/shared — student_001 sees shared asset in college
	sharedResp := doRequest(t, "GET", assetAPI+"/api/v1/asset/assets/shared", studentToken, nil)
	if sharedResp.Code != 0 {
		t.Fatalf("shared assets failed: code=%d", sharedResp.Code)
	}
	var sharedData struct {
		List []struct{ AssetNo string `json:"assetNo"` } `json:"list"`
	}
	json.Unmarshal(sharedResp.Data, &sharedData)
	foundShared := false
	for _, a := range sharedData.List {
		if a.AssetNo == "EQUIP-2026-0002" {
			foundShared = true
		}
	}
	if !foundShared {
		t.Errorf("student_001 should see shared asset EQUIP-2026-0002")
	}

	// 7. student_me should NOT see info college shared assets
	sharedMe := doRequest(t, "GET", assetAPI+"/api/v1/asset/assets/shared", studentMeToken, nil)
	var sharedMeData struct {
		List []struct{ AssetNo string `json:"assetNo"` } `json:"list"`
	}
	json.Unmarshal(sharedMe.Data, &sharedMeData)
	for _, a := range sharedMeData.List {
		if a.AssetNo == "EQUIP-2026-0002" {
			t.Errorf("student_me should not see EQUIP-2026-0002 from info college")
		}
	}
	t.Logf("E2E-06: shared asset college isolation OK")

	// 8. GET /asset/assets?scope=my — only user's assets
	myAssets := doRequest(t, "GET", assetAPI+"/api/v1/asset/assets?scope=my", studentToken, nil)
	if myAssets.Code != 0 {
		t.Fatalf("my assets failed: code=%d", myAssets.Code)
	}
	var myData struct {
		List []struct {
			AssetNo string `json:"assetNo"`
			UserId  int64  `json:"userId"`
		} `json:"list"`
	}
	json.Unmarshal(myAssets.Data, &myData)
	for _, a := range myData.List {
		if a.UserId != 10003 {
			t.Errorf("scope=my leaked asset %s with userId=%d", a.AssetNo, a.UserId)
		}
	}
	t.Logf("E2E-06: scope=my isolation OK")

	// 9. GET /workflow/requests?assetId=501
	wfByAsset := doRequest(t, "GET", workflowAPI+"/api/v1/workflow/requests?scope=all&assetId=501", adminToken, nil)
	if wfByAsset.Code != 0 {
		t.Fatalf("workflow by assetId failed: code=%d", wfByAsset.Code)
	}
	t.Logf("E2E-06: workflow assetId filter OK")
}

// sanitize for log output
func _s(s string) string {
	return strings.ReplaceAll(s, "\n", "")
}
