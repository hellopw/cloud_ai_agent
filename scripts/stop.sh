#!/usr/bin/env bash
set -euo pipefail

PID_DIR="/tmp/cloud-ai-agent"

# ---------------------------------------------------------------------------
# Colors
# ---------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[stop]${NC} $*"; }
warn() { echo -e "${YELLOW}[stop]${NC} $*"; }
err()  { echo -e "${RED}[stop]${NC} $*"; }

# ---------------------------------------------------------------------------
# Stop a service by PID file
# ---------------------------------------------------------------------------
stop_service() {
    local name="$1"
    local pid_file="$PID_DIR/$name.pid"

    if [ ! -f "$pid_file" ]; then
        warn "No PID file for $name — skipping"
        return 0
    fi

    local pid
    pid="$(cat "$pid_file")"

    if [ -z "$pid" ]; then
        warn "Empty PID file for $name — cleaning up"
        rm -f "$pid_file"
        return 0
    fi

    if ! kill -0 "$pid" 2>/dev/null; then
        log "$name (pid $pid) is not running — cleaning up PID file"
        rm -f "$pid_file"
        return 0
    fi

    log "Stopping $name (pid $pid) ..."
    kill "$pid" 2>/dev/null || true

    # Wait up to 5s for graceful shutdown
    for i in $(seq 1 5); do
        if ! kill -0 "$pid" 2>/dev/null; then
            log "$name stopped"
            rm -f "$pid_file"
            return 0
        fi
        sleep 1
    done

    # Force kill if still alive
    warn "$name did not stop gracefully — force killing"
    kill -9 "$pid" 2>/dev/null || true
    rm -f "$pid_file"
}

# ---------------------------------------------------------------------------
# Cleanup any orphaned processes (belt-and-suspenders)
# ---------------------------------------------------------------------------
cleanup_orphans() {
    # Kill any go run / cloud-ai-agent processes for this project
    pkill -f "go run.*cmd/server" 2>/dev/null && warn "Cleaned up orphaned Go backend" || true
    pkill -f "vite.*--port.*3000" 2>/dev/null && warn "Cleaned up orphaned Vite dev server" || true
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
echo -e "${RED}╔══════════════════════════════════════════╗${NC}"
echo -e "${RED}║       Cloud AI Agent — Shutdown          ║${NC}"
echo -e "${RED}╚══════════════════════════════════════════╝${NC}"
echo ""

stop_service "frontend"
stop_service "backend"

# Clean up any stragglers
cleanup_orphans

# Clean PID dir if empty
rmdir "$PID_DIR/logs" 2>/dev/null || true
rmdir "$PID_DIR" 2>/dev/null || true

echo ""
log "All services stopped."
echo ""
