#!/usr/bin/env bash
# 真机基准采集：启动本地采集服务，用 cloudflared 临时暴露为 https 地址；Ctrl-C 同时停止两者，地址随之失效
set -euo pipefail
cd "$(dirname "$0")/.."
PORT="${PORT:-8765}"

if ! command -v cloudflared >/dev/null 2>&1; then
  echo "缺少 cloudflared，请先安装：brew install cloudflared" >&2
  exit 1
fi

TMP="$(mktemp -d)"
BIN="$TMP/baseline_server"
LOG="$TMP/cloudflared.log"
SERVER=""
TUNNEL=""
trap 'kill $SERVER $TUNNEL 2>/dev/null; rm -rf "$TMP"' EXIT INT TERM
go build -o "$BIN" ./cmd/baseline_server
"$BIN" -port "$PORT" &
SERVER=$!

# 确认是本服务在监听再开隧道：端口被别的程序占用时，绝不能把那个程序暴露到公网
ready=""
for _ in $(seq 1 50); do
  if ! kill -0 "$SERVER" 2>/dev/null; then
    echo "采集服务没有启动（端口 $PORT 可能被占用，可用 PORT=其他端口 重试）" >&2
    exit 1
  fi
  if curl -sf "http://127.0.0.1:$PORT/" 2>/dev/null | grep -q collectBaseline; then
    ready=1
    break
  fi
  sleep 0.2
done
if [ -z "$ready" ]; then
  echo "采集服务 10 秒内没有就绪，端口 $PORT 上不是本服务" >&2
  exit 1
fi

cloudflared tunnel --no-autoupdate --url "http://127.0.0.1:$PORT" >"$LOG" 2>&1 &
TUNNEL=$!

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
