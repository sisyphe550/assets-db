#!/usr/bin/env bash
# ==============================
# FAMS 基础设施启动脚本
# ==============================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_DIR="$SCRIPT_DIR/../deploy/docker"
ENV_FILE="$COMPOSE_DIR/docker-compose-env.yml"
ENV_EXAMPLE="$COMPOSE_DIR/docker-compose-env.example.yml"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*"; }

# ---- 检查 env 文件 ----
if [ ! -f "$ENV_FILE" ]; then
    if [ -f "$ENV_EXAMPLE" ]; then
        warn "env file not found, copying from example"
        cp "$ENV_EXAMPLE" "$ENV_FILE"
        warn "please review $ENV_FILE before proceeding"
    else
        err "env file not found: $ENV_FILE"
        exit 1
    fi
fi

# ---- 检查端口占用 ----
check_port() {
    local port=$1
    if lsof -i ":$port" -sTCP:LISTEN > /dev/null 2>&1; then
        err "port $port is already in use"
        return 1
    fi
}

log "checking port availability..."
PORTS=(5432 3306 27017 6379 9092 2379 4317 16686 9090 3000)
HAS_CONFLICT=0
for p in "${PORTS[@]}"; do
    if ! check_port "$p"; then
        HAS_CONFLICT=1
    fi
done
if [ $HAS_CONFLICT -ne 0 ]; then
    err "port conflicts detected, please free them first"
    exit 1
fi
log "all ports available"

# ---- 启动 compose ----
log "starting infrastructure..."
cd "$COMPOSE_DIR"
docker compose --env-file docker-compose-env.yml up -d

# ---- 等待健康检查 ----
log "waiting for services to be healthy (max 120s)..."
TIMEOUT=120
ELAPSED=0
INTERVAL=5

SERVICES=(
    "fams-postgres"
    "fams-mysql"
    "fams-mongo"
    "fams-redis"
    "fams-kafka"
    "fams-etcd"
    "fams-jaeger"
    "fams-prometheus"
    "fams-grafana"
)

while [ $ELAPSED -lt $TIMEOUT ]; do
    ALL_HEALTHY=true
    for svc in "${SERVICES[@]}"; do
        STATUS=$(docker inspect -f '{{.State.Health.Status}}' "$svc" 2>/dev/null || echo "missing")
        if [ "$STATUS" != "healthy" ]; then
            ALL_HEALTHY=false
            break
        fi
    done

    if $ALL_HEALTHY; then
        log "all services healthy (${ELAPSED}s)"
        break
    fi

    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
done

if ! $ALL_HEALTHY; then
    err "timeout waiting for services to become healthy"
    docker compose ps
    exit 1
fi

# ---- 验证 Kafka Topic ----
log "verifying Kafka topics..."
docker exec fams-kafka kafka-topics.sh --bootstrap-server localhost:9094 --list

log "infrastructure ready. Next steps:"
echo "  Grafana:  http://localhost:3000 (admin/fams_dev_pass)"
echo "  Jaeger:   http://localhost:16686"
echo "  Prometheus: http://localhost:9090"
echo ""
echo "  Run healthcheck: ./scripts/healthcheck.sh"
