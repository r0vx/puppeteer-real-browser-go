# 生产镜像示例：Google Chrome 稳定版 + Windows 字体 + tini
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY . .
# 换成你的程序入口
RUN go build -o /out/app ./cmd/yourapp

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates wget tini fontconfig \
 && wget -q -O /tmp/chrome.deb https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb \
 && apt-get install -y --no-install-recommends /tmp/chrome.deb \
 && rm -rf /var/lib/apt/lists/* /tmp/chrome.deb
# Windows 账号需要的字体：自备合法授权的字体文件，放在构建上下文的 fonts/windows/ 下
COPY fonts/windows/ /usr/share/fonts/windows/
RUN fc-cache -f
COPY --from=build /out/app /usr/local/bin/app
# tini 作为 1 号进程回收 Chrome 的辅助进程，否则关闭浏览器时会多等 3 秒
ENTRYPOINT ["tini", "--", "/usr/local/bin/app"]
