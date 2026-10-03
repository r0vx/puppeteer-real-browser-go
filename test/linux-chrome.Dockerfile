# 模拟生产的测试环境：Linux amd64 + Google Chrome 稳定版 + 无 GPU；含 Xvfb 供有界面测试，含中文字体（否则中文显示成方块）
FROM golang:1.25-bookworm
RUN apt-get update \
 && apt-get install -y --no-install-recommends wget ca-certificates xvfb fonts-liberation fonts-noto-cjk procps \
 && wget -q -O /tmp/chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
 && apt-get install -y --no-install-recommends /tmp/chrome.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/chrome.deb
