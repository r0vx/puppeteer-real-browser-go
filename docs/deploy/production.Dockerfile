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
# Windows 账号需要的字体：自备合法授权的字体文件，放在构建上下文的 fonts/windows/ 下（必须含微软雅黑或宋体，否则中文显示成方块）
COPY fonts/windows/ /usr/share/fonts/windows/
# 只让 Chrome 看到 Windows 字体：把 docs/deploy/fontconfig/ 复制到构建上下文
COPY fontconfig/55-windows-fonts-only.conf /etc/fonts/conf.d/55-windows-fonts-only.conf
RUN fc-cache -f \
 && (fc-list :lang=zh family | grep -q . || (echo "缺少中文字体：fonts/windows/ 里至少要有微软雅黑（msyh）或宋体（simsun）" >&2; exit 1))
COPY --from=build /out/app /usr/local/bin/app
# tini 作为 1 号进程回收 Chrome 的辅助进程，否则关闭浏览器时会多等 3 秒
ENTRYPOINT ["tini", "--", "/usr/local/bin/app"]
