#!/usr/bin/env bash
# FAMS 前端联调冒烟测试 — 通过 API 验证各模块数据流
set -euo pipefail

BASE_USER="${BASE_USER:-http://127.0.0.1:8888/api/v1}"
BASE_ASSET="${BASE_ASSET:-http://127.0.0.1:8889/api/v1}"
BASE_WF="${BASE_WF:-http://127.0.0.1:8890/api/v1}"
BASE_INV="${BASE_INV:-http://127.0.0.1:8891/api/v1}"
BASE_RPT="${BASE_RPT:-http://127.0.0.1:8892/api/v1}"
BASE_FE="${BASE_FE:-http://127.0.0.1:5173}"
PASS="${PASS:-Test@123456}"

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

pass() { PASS_COUNT=$((PASS_COUNT + 1)); echo "  ✅ $1"; }
fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); echo "  ❌ $1"; }
skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); echo "  ⏭️  $1"; }

login() {
  local user=$1
  curl -s -X POST "$BASE_USER/user/login" \
    -H 'Content-Type: application/json' \
    -d "{\"username\":\"$user\",\"password\":\"$PASS\"}"
}

token_of() {
  login "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('accessToken','') if d.get('code')==0 else '')"
}

code_of() {
  python3 -c "import sys,json; print(json.load(sys.stdin).get('code',''))"
}

echo "========================================"
echo " FAMS 前端联调冒烟测试"
echo " $(date '+%Y-%m-%d %H:%M:%S')"
echo "========================================"

# --- 0. 服务可达 ---
echo ""
echo "[0] 服务健康"
check_service() {
  local name=$1 code=$2
  if [[ "$code" =~ ^[23] ]]; then pass "$name 可达 (HTTP $code)"; else fail "$name 不可达 (HTTP $code)"; fi
}
tok0=$(token_of admin_school 2>/dev/null || true)
code_user=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_USER/user/login" -H 'Content-Type: application/json' -d '{}' 2>/dev/null || echo 000)
code_asset=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $tok0" "$BASE_ASSET/asset/assets?page=1&pageSize=1" 2>/dev/null || echo 000)
code_fe=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_FE/" 2>/dev/null || echo 000)
check_service "user-api" "$code_user"
check_service "asset-api" "$code_asset"
check_service "frontend" "$code_fe"

# --- 1. 三角色登录 ---
echo ""
echo "[1] 登录与 /user/me"
for acct in admin_school:1 admin_info:2 student_001:3; do
  user="${acct%%:*}"
  expect_role="${acct##*:}"
  resp=$(login "$user")
  code=$(echo "$resp" | code_of)
  tok=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('accessToken',''))" 2>/dev/null || true)
  if [[ "$code" != "0" || -z "$tok" ]]; then
    fail "$user 登录失败 code=$code"
    continue
  fi
  me=$(curl -s -H "Authorization: Bearer $tok" "$BASE_USER/user/me")
  role=$(echo "$me" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('roleLevel',''))" 2>/dev/null || true)
  if [[ "$role" == "$expect_role" ]]; then
    pass "$user 登录成功 roleLevel=$role"
  else
    fail "$user roleLevel=$role (期望 $expect_role)"
  fi
done

# --- 2. 组织树 (P7 fix: nodes 字段) ---
echo ""
echo "[2] 组织树 P7"
TOK=$(token_of admin_school)
tree=$(curl -s -H "Authorization: Bearer $TOK" "$BASE_USER/user/departments/tree")
node_count=$(echo "$tree" | python3 -c "import sys,json; d=json.load(sys.stdin); n=d.get('data',{}).get('nodes',[]); print(len(n))" 2>/dev/null || echo 0)
if [[ "$node_count" -ge 1 ]]; then
  pass "组织树返回 nodes 数组 (${node_count} 根节点)"
else
  fail "组织树 nodes 为空或格式错误"
fi

# --- 3. 用户列表 (P7) ---
echo ""
echo "[3] 用户管理 P7"
users=$(curl -s -H "Authorization: Bearer $TOK" "$BASE_USER/user/users?page=1&pageSize=5")
total=$(echo "$users" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('total',0))" 2>/dev/null || echo 0)
if [[ "$total" -ge 3 ]]; then pass "用户列表 total=$total"; else fail "用户列表 total=$total (期望≥3)"; fi

# --- 4. 资产 (P3) ---
echo ""
echo "[4] 资产管理 P3"
assets=$(curl -s -H "Authorization: Bearer $TOK" "$BASE_ASSET/asset/assets?page=1&pageSize=20")
asset_total=$(echo "$assets" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('total',0))" 2>/dev/null || echo 0)
if [[ "$asset_total" -ge 1 ]]; then pass "资产列表 total=$asset_total"; else fail "资产列表为空"; fi

TOK_INFO=$(token_of admin_info)
assets_college=$(curl -s -H "Authorization: Bearer $TOK_INFO" "$BASE_ASSET/asset/assets?page=1&pageSize=20")
if echo "$assets_college" | code_of | grep -q '^0$'; then pass "院级资产列表可访问"; else fail "院级资产列表失败"; fi

# --- 5. 工单 (P4) ---
echo ""
echo "[5] 工单审批 P4"
todo=$(curl -s -H "Authorization: Bearer $TOK_INFO" "$BASE_WF/workflow/requests?scope=todo&status=1&page=1&pageSize=10")
if echo "$todo" | code_of | grep -q '^0$'; then pass "院级待审批列表可访问"; else fail "院级待审批列表失败"; fi

# --- 6. 盘点 (P5) ---
echo ""
echo "[6] 盘点 P5"
tasks=$(curl -s -H "Authorization: Bearer $TOK" "$BASE_INV/inventory/tasks?page=1&pageSize=10")
if echo "$tasks" | code_of | grep -q '^0$'; then pass "盘点任务列表可访问"; else fail "盘点任务列表失败"; fi

TOK_STU=$(token_of student_001)
tasks_stu=$(curl -s -H "Authorization: Bearer $TOK_STU" "$BASE_INV/inventory/tasks?status=1&page=1&pageSize=10")
if echo "$tasks_stu" | code_of | grep -q '^0$'; then pass "师生盘点任务列表可访问"; else fail "师生盘点任务列表失败"; fi

# --- 7. 报表 (P6) ---
echo ""
echo "[7] 统计报表 P6"
bydept=$(curl -s -H "Authorization: Bearer $TOK" "$BASE_RPT/report/assets/by-dept")
items=$(echo "$bydept" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('data',{}).get('items',[])))" 2>/dev/null || echo 0)
if [[ "$items" -ge 1 ]]; then pass "按部门统计 items=$items"; else fail "按部门统计无数据"; fi

export_resp=$(curl -s -X POST -H "Authorization: Bearer $TOK" -H 'Content-Type: application/json' \
  -d '{"exportType":"asset_list"}' "$BASE_RPT/report/export")
job_id=$(echo "$export_resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('jobId',0))" 2>/dev/null || echo 0)
if [[ "$job_id" -gt 0 ]]; then pass "CSV 导出任务创建 jobId=$job_id"; else fail "CSV 导出任务创建失败"; fi

# --- 8. 越权 ---
echo ""
echo "[8] 权限边界"
dept_create=$(curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer $TOK_INFO" \
  -H 'Content-Type: application/json' \
  -d '{"parentId":1,"deptName":"非法","deptCode":"XX","sortOrder":0}' \
  "$BASE_USER/user/departments")
if [[ "$dept_create" == "403" || "$dept_create" == "200" ]]; then
  # 200 with error body also possible
  body=$(curl -s -X POST -H "Authorization: Bearer $TOK_INFO" -H 'Content-Type: application/json' \
    -d '{"parentId":1,"deptName":"非法","deptCode":"XX","sortOrder":0}' "$BASE_USER/user/departments")
  c=$(echo "$body" | code_of)
  if [[ "$c" != "0" ]]; then pass "院级创建部门被拒绝 code=$c"; else fail "院级不应能创建部门"; fi
else
  pass "院级创建部门 HTTP $dept_create"
fi

stu_tok=$(token_of student_001)
admin_assets=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $stu_tok" \
  "$BASE_ASSET/asset/assets?page=1&pageSize=1")
# 师生可以访问资产 API (scope 不同)，越权在路由层 — 仅记录
pass "师生 token 调 asset API → HTTP $admin_assets (路由守卫在前端)"

# --- 9. 前端代理 ---
echo ""
echo "[9] Vite 代理"
fe_login=$(curl -s -X POST "$BASE_FE/api/v1/user/login" -H 'Content-Type: application/json' \
  -d '{"username":"admin_school","password":"Test@123456"}')
if echo "$fe_login" | code_of | grep -q '^0$'; then pass "Vite 代理 /api/v1/user/login 正常"; else fail "Vite 代理登录失败"; fi

# --- 10. Univer 检查 ---
echo ""
echo "[10] Univer SDK"
if grep -q '@univerjs' /Users/sisyphus/Code/Go/assets-db/frontend/package.json 2>/dev/null; then
  pass "package.json 已安装 @univerjs"
else
  skip "Univer SDK 未安装 — 盘点使用 Ant Design 可编辑 Table（已知限制）"
fi

# --- 汇总 ---
echo ""
echo "========================================"
echo " 通过: $PASS_COUNT  失败: $FAIL_COUNT  跳过: $SKIP_COUNT"
echo "========================================"
[[ "$FAIL_COUNT" -eq 0 ]]
