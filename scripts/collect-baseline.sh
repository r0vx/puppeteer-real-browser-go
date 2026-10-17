#!/usr/bin/env bash
# 真机基准采集：启动本地采集服务，用 cloudflared 临时暴露为 https 地址；Ctrl-C 同时停止两者，地址随之失效
set -euo pipefail
cd "$(dirname "$0")/.."
PORT="${PORT:-8765}"

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "缺少 cloudflared，请先安装：brew install cloudflared" >&2
  exit 1
fi

BIN="$(mktemp -d)/baseline_server"
go build -o "$BIN" ./cmd/baseline_server
LOG="$(mktemp)"
"$BIN" -port "$PORT" &
SERVER=$!
cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:$PORT" >"$LOG" 2>&1 &
TUNNEL=$!
trap 'kill "$SERVER" "$TUNNEL" 2>/dev/null; rm -f "$LOG" "$BIN"' EXIT INT TERM

URL=""
for _ in $(seq 1 60); do
  URL="$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG" | head -1 || true)"
  [ -n "$URL" ] && break
  sleep 1
done
if [ -z "$URL" ]; then
  echo "cloudflared 60 秒内没有给出地址，输出如下：" >&2
  cat "$LOG" >&2
  exit 1
fi

echo
echo "================================================================"
echo "  在手机上打开： $URL"
echo "  结果保存到：   test/baseline/"
echo "  采集完成后按 Ctrl-C 结束"
echo "================================================================"
wait "$SERVER"
