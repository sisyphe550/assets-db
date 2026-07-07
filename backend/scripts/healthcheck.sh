#!/usr/bin/env bash
# ==============================
# FAMS 健康检查脚本
# ==============================
set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PASS=0
FAIL=0

check() {
    local name=$1
    local cmd=$2
    if eval "$cmd" > /dev/null 2>&1; then
        echo -e "${GREEN}[OK]${NC} $name"
        PASS=$((PASS + 1))
    else
        echo -e "${RED}[FAIL]${NC} $name"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== FAMS Health Check ==="
echo ""

check "PostgreSQL"        "docker exec fams-postgres pg_isready -U fams"
check "MySQL"             "docker exec fams-mysql mysqladmin ping -ufams -pfams_dev_pass"
check "MongoDB"           "docker exec fams-mongo mongosh --eval 'db.adminCommand(\"ping\")' --quiet"
check "Redis"             "docker exec fams-redis redis-cli ping"
check "Kafka"             "docker exec fams-kafka kafka-topics.sh --bootstrap-server localhost:9094 --list"
check "etcd"              "docker exec fams-etcd etcdctl endpoint health --endpoints=http://127.0.0.1:2379"
check "Jaeger"            "curl -sf http://localhost:16686/ > /dev/null"
check "Prometheus"        "curl -sf http://localhost:9090/-/healthy > /dev/null"
check "Grafana"           "curl -sf http://localhost:3000/api/health > /dev/null"

echo ""
echo "=== Result: ${PASS} passed, ${FAIL} failed ==="

if [ $FAIL -gt 0 ]; then
    exit 1
fi
