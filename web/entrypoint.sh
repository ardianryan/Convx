#!/bin/sh
set -e

echo "🎵 Convx Music — Starting services..."

# Start Go backend on internal port
GO_PORT="${GO_BACKEND_PORT:-7555}"
PORT="$GO_PORT" /app/convx-go &
GO_PID=$!
echo "✅ Go backend started (PID: $GO_PID) on port $GO_PORT"

# Wait for Go backend to be ready
for i in $(seq 1 30); do
  if wget -q -O /dev/null "http://localhost:${GO_PORT}/api/health" 2>/dev/null; then
    echo "✅ Go backend is healthy"
    break
  fi
  sleep 0.5
done

# Start Node.js gateway on the main port
cd /app/server
export PORT="${CONVX_PORT:-7554}"
export GO_BACKEND_PORT="$GO_PORT"
echo "🚀 Starting Node.js gateway on port $PORT..."
exec node index.js
