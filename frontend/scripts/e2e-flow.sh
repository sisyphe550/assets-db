#!/usr/bin/env bash
# FAMS 前端完整业务流程 API 联调（模拟 UI 操作序列）
set -euo pipefail

BASE_USER="http://127.0.0.1:8888/api/v1"
BASE_ASSET="http://127.0.0.1:8889/api/v1"
BASE_WF="http://127.0.0.1:8890/api/v1"
BASE_INV="http://127.0.0.1:8891/api/v1"
PASS="Test@123456"

login() {
  curl -s -X POST "$BASE_USER/user/login" -H 'Content-Type: application/json' \
    -d "{\"username\":\"$1\",\"password\":\"$PASS\"}"
}

tok() { login "$1" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['accessToken'])"; }

echo "=== 业务流程联调 ==="

# 1. 领用流程
echo "[领用流程]"
STU=$(tok student_001)
INFO=$(tok admin_info)
SCHOOL=$(tok admin_school)

WF=$(curl -s -X POST "$BASE_WF/workflow/requests" -H "Authorization: Bearer $STU" \
  -H 'Content-Type: application/json' \
  -d '{"assetId":501,"type":1,"reason":"E2E测试领用申请"}')
WF_ID=$(echo "$WF" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('id',0))")
WF_CODE=$(echo "$WF" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',''))")
echo "  创建工单: code=$WF_CODE id=$WF_ID"

if [[ "$WF_ID" -gt 0 ]]; then
  APR1=$(curl -s -X POST "$BASE_WF/workflow/requests/$WF_ID/approve" \
    -H "Authorization: Bearer $INFO" -H 'Content-Type: application/json' -d '{"comment":"院级同意"}')
  echo "  院级初审: code=$(echo "$APR1" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',''))")"
  APR2=$(curl -s -X POST "$BASE_WF/workflow/requests/$WF_ID/approve" \
    -H "Authorization: Bearer $SCHOOL" -H 'Content-Type: application/json' -d '{"comment":"校级同意"}')
  echo "  校级复审: code=$(echo "$APR2" | python3 -c "import sys,json; print(json.load(sys.stdin).get('code',''))")"
  ASSET=$(curl -s -H "Authorization: Bearer $SCHOOL" "$BASE_ASSET/asset/assets/501")
  STATUS=$(echo "$ASSET" | python3 -c "import sys,json; print(json.load(sys.stdin).get('data',{}).get('status',''))")
  echo "  资产501状态: $STATUS (期望 2=领用中，Kafka 异步可能仍为1)"
fi

# 2. 盘点草稿提交
echo "[盘点流程]"
TASK=$(curl -s -X POST "$BASE_INV/inventory/tasks" -H "Authorization: Bearer $INFO" \
  -H 'Content-Type: application/json' \
  -d '{"taskName":"E2E-Frontend-Test","scopeDeptId":15,"startTime":"2026-07-08T00:00:00Z","endTime":"2026-07-15T23:59:59Z","assigneeIds":[10003,10004]}')
TASK_ID=$(echo "$TASK" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',{}).get('taskId') or d.get('data',{}).get('id') or 0)")
echo "  创建盘点任务: taskId=$TASK_ID"

if [[ "$TASK_ID" -gt 0 ]]; then
  EXP=$(curl -s -H "Authorization: Bearer $STU" "$BASE_INV/inventory/tasks/$TASK_ID/expected-assets")
  ASSET_NO=$(echo "$EXP" | python3 -c "import sys,json; l=json.load(sys.stdin).get('data',{}).get('list',[]); print(l[0]['assetNo'] if l else '')" 2>/dev/null || true)
  if [[ -n "$ASSET_NO" ]]; then
    SUB=$(curl -s -X POST "$BASE_INV/inventory/tasks/$TASK_ID/submit" -H "Authorization: Bearer $STU" \
      -H 'Content-Type: application/json' \
      -d "{\"items\":[{\"assetNo\":\"$ASSET_NO\",\"modifiedCells\":{\"actual_location\":\"102\"},\"expectedUpdatedAt\":null}]}")
    SUCC=$(echo "$SUB" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('data',{}).get('success',[])))" 2>/dev/null || echo 0)
    echo "  提交草稿 $ASSET_NO: success=$SUCC 条"
  else
    echo "  跳过提交: 无预期资产"
  fi
fi

# 3. P7 组织树 + 用户
echo "[P7 组织与用户]"
TREE=$(curl -s -H "Authorization: Bearer $SCHOOL" "$BASE_USER/user/departments/tree")
ROOT_NAME=$(echo "$TREE" | python3 -c "import sys,json; n=json.load(sys.stdin)['data']['nodes'][0]['deptName']; print(n)" 2>/dev/null || echo '?')
echo "  组织树根节点: $ROOT_NAME"

# 4. 菜单项数量（静态检查）
echo "[菜单配置]"
ADMIN_MENU=$(grep -c "key: '/admin" /Users/sisyphus/Code/Go/assets-db/frontend/src/config/menu.ts || true)
COLLEGE_MENU=$(grep -c "key: '/college" /Users/sisyphus/Code/Go/assets-db/frontend/src/config/menu.ts || true)
echo "  校级菜单项: $ADMIN_MENU (期望 7)"
echo "  院级菜单项: $COLLEGE_MENU (期望 5，含统计报表)"

# 5. 院级无用户管理菜单
if grep -q "/college/users" /Users/sisyphus/Code/Go/assets-db/frontend/src/config/menu.ts; then
  echo "  院级用户管理菜单: 已配置"
else
  echo "  院级用户管理菜单: 未配置（已知限制，API 已支持）"
fi

echo "=== 完成 ==="
