#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────
# LumenRoute Development Launcher
# Starts Go backend (:8080) and React frontend (:5173) together.
# Press Ctrl+C to stop both.
# ──────────────────────────────────────

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
GO_PID=""

cleanup() {
    echo ""
    echo "🛑 Shutting down..."
    if [ -n "$GO_PID" ] && kill -0 "$GO_PID" 2>/dev/null; then
        kill "$GO_PID" 2>/dev/null || true
        wait "$GO_PID" 2>/dev/null || true
        echo "   Go backend stopped."
    fi
    exit 0
}
trap cleanup SIGINT SIGTERM

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  LumenRoute Dev Environment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# ── .env ───────────────────────────────
ENV_FILE="$ROOT_DIR/.env.local"
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck source=/dev/null
    source "$ENV_FILE"
    set +a
    echo "✅ Loaded config from .env.local"
else
    echo "⚠️  .env.local not found, using defaults"
fi

# ── Go Backend ─────────────────────────
BACKEND_PORT="${LUMENROUTE_SERVER_PORT:-8080}"
echo "🔧 Building Go backend..."
go build -o "$ROOT_DIR/lumenroute" "$ROOT_DIR/cmd/server" || {
    echo "❌ Go build failed"
    exit 1
}
echo "✅ Go backend built"

echo "🚀 Starting Go backend on :$BACKEND_PORT ..."
"$ROOT_DIR/lumenroute" &
GO_PID=$!

# Wait for backend to be ready
for i in $(seq 1 30); do
    if curl -s "http://localhost:$BACKEND_PORT/api/auth/login" -o /dev/null 2>/dev/null; then
        echo "✅ Go backend is ready (port $BACKEND_PORT)"
        break
    fi
    if [ "$i" -eq 30 ]; then
        echo "❌ Go backend failed to start within 30s"
        cleanup
    fi
    sleep 1
done

# ── React Frontend ─────────────────────
echo ""
echo "📦 Installing frontend dependencies (if needed)..."
cd "$ROOT_DIR/web"
if [ ! -d "node_modules" ]; then
    npm install
else
    echo "   node_modules exists, skipping npm install"
fi

echo ""
echo "🚀 Starting React frontend (http://localhost:5173) ..."
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Go Backend : http://localhost:$BACKEND_PORT"
echo "  Frontend   : http://localhost:5173"
echo "  Press Ctrl+C to stop all services"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

npx vite --host
