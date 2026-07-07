#!/usr/bin/env bash
# ==============================
# FAMS 基础设施停止脚本
# ==============================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COMPOSE_DIR="$SCRIPT_DIR/../deploy/docker"

cd "$COMPOSE_DIR"
docker compose --env-file docker-compose-env.yml down
echo "infrastructure stopped"
