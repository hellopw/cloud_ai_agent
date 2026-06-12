#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PID_DIR="/tmp/cloud-ai-agent"
LOG_DIR="$PID_DIR/logs"

mkdir -p "$PID_DIR" "$LOG_DIR"

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[start]${NC} $*"; }
warn() { echo -e "${YELLOW}[start]${NC} $*"; }
err()  { echo -e "${RED}[start]${NC} $*"; }

# ---------------------------------------------------------------------------
# Check prerequisites
# ---------------------------------------------------------------------------
check_cmd() {
    if ! command -v "$1" &>/dev/null; then
        err "Missing required command: $1"
        exit 1
    fi
}

check_cmd go
check_cmd node

# ---------------------------------------------------------------------------
# Start Backend (Go)
# ---------------------------------------------------------------------------
start_backend() {
    local pid_file="$PID_DIR/backend.pid"
    local log_file="$LOG_DIR/backend.log"

    if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        warn "Backend is already running (pid $(cat "$pid_file"))"
        return 0
    fi

    log "Starting Go backend on :8080 ..."
    cd "$PROJECT_ROOT/backend"

    nohup go run ./cmd/server \
        >>"$log_file" 2>&1 &

    local pid=$!
    echo "$pid" > "$pid_file"
    log "Backend started (pid $pid), logs → $log_file"
}

# ---------------------------------------------------------------------------
# Start Frontend (Vite)
# ---------------------------------------------------------------------------
start_frontend() {
    local pid_file="$PID_DIR/frontend.pid"
    local log_file="$LOG_DIR/frontend.log"

    if [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" 2>/dev/null; then
        warn "Frontend is already running (pid $(cat "$pid_file"))"
        return 0
    fi

    # Ensure dependencies are installed
    if [ ! -d "$PROJECT_ROOT/frontend/node_modules" ]; then
        log "Installing frontend dependencies ..."
        cd "$PROJECT_ROOT/frontend"
        npm install >>"$log_file" 2>&1
    fi

    log "Starting Vite dev server on :3000 ..."
    cd "$PROJECT_ROOT/frontend"

    nohup npm run dev \
        >>"$log_file" 2>&1 &

    local pid=$!
    echo "$pid" > "$pid_file"
    log "Frontend started (pid $pid), logs → $log_file"
}

# ---------------------------------------------------------------------------
# Wait for services to be ready
# ---------------------------------------------------------------------------
wait_ready() {
    local url="$1"
    local name="$2"
    local max=30

    log "Waiting for $name ($url) ..."
    for i in $(seq 1 $max); do
        if curl -s -o /dev/null "$url" 2>/dev/null; then
            log "$name is ready"
            return 0
        fi
        sleep 1
    done
    warn "$name did not respond within ${max}s — check the log"
}

# ---------------------------------------------------------------------------
# Open browser
# ---------------------------------------------------------------------------
open_browser() {
    local url="http://localhost:3000"
    log "Opening $url in browser ..."
    if command -v open &>/dev/null; then
        open "$url"
    elif command -v xdg-open &>/dev/null; then
        xdg-open "$url"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║       Cloud AI Agent — Dev Startup       ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
echo ""

start_backend
start_frontend

# Wait for both to be ready
wait_ready "http://localhost:8080/api/health" "Backend"
wait_ready "http://localhost:3000"              "Frontend"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  All services running!${NC}"
echo -e "  Frontend : ${YELLOW}http://localhost:3000${NC}"
echo -e "  Backend  : ${YELLOW}http://localhost:8080${NC}"
echo -e "  Health   : ${YELLOW}http://localhost:8080/api/health${NC}"
echo -e "  Stop     : ${YELLOW}./scripts/stop.sh${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

open_browser
