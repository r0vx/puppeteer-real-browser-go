#!/usr/bin/env bash
# 在模拟生产的 Linux 环境（amd64 + Google Chrome 稳定版、无 GPU）里跑库的全部测试；额外参数透传给 go test
set -euo pipefail
cd "$(dirname "$0")/.."
docker build --platform linux/amd64 -t prbg-linux-chrome -f test/linux-chrome.Dockerfile test
docker run --rm --init --platform linux/amd64 --shm-size=1g \
  -v "$PWD":/src:ro -v prbg-gomod-amd64:/go/pkg/mod -v prbg-gocache-amd64:/root/.cache/go-build -w /src \
  prbg-linux-chrome go test -race -count=1 -timeout 1800s "$@" ./pkg/... ./internal/...
