#!/usr/bin/env bash
# ==============================
# FAMS 基础设施重置脚本
# 清空所有数据卷并重启
# ==============================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_DIR="$SCRIPT_DIR/../deploy/docker"

cd "$COMPOSE_DIR"
docker compose --env-file docker-compose-env.yml down -v
echo "infrastructure stopped and volumes removed"

# 清理数据目录
DATA_DIR="$SCRIPT_DIR/../data"
if [ -d "$DATA_DIR" ]; then
    rm -rf "$DATA_DIR"
    echo "data directory cleaned"
fi

# 重启
"$SCRIPT_DIR/infra-up.sh"
