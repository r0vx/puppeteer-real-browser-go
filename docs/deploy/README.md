# 生产部署说明

## 环境
- Linux amd64 + Google Chrome 稳定版（不是发行版的 chromium 包：品牌与编解码器不同）。
- 无 GPU 时 WebGL 走 SwiftShader 软件渲染；库会把显卡标识与能力参数换成账号档案里的真实显卡，但渲染出来的图像特征仍是软件渲染（已知残余风险）。
- 容器用 tini 或 `docker run --init` 作 1 号进程，回收 Chrome 辅助进程。

## 字体
网站会做字体探测：用一串候选字体测量文字宽度，判断哪些字体存在。2026-10-03 在模拟生产的容器里实测，不做处理时只能检测到 Liberation 字体：一看就是 Linux，没有任何 Windows 字体，而且**中文全部显示成方块**。生产镜像必须处理：

1. **安装 Windows 字体**（自备合法授权文件，放在 `fonts/windows/`）：微软雅黑（msyh）、宋体（simsun）、黑体（simhei）、Arial、Times New Roman、Segoe UI、Calibri、Consolas、Tahoma、Verdana。**至少要有微软雅黑或宋体**，否则中文显示成方块（`production.Dockerfile` 构建时会检查，没有就报错）。
2. **屏蔽 Linux 自带的西文字体**：把 `fontconfig/99-windows-fonts-only.conf` 放进 `/etc/fonts/conf.d/`。它会隐藏 Liberation、DejaVu，并让 sans-serif / serif / monospace 优先用 Windows 字体。中文字体不屏蔽：实在拿不到微软雅黑时，可以装 `fonts-noto-cjk` 兜底，中文能正常显示，但字体探测能看出这是 Linux。
3. **检查**：
   ```bash
   docker run --rm <镜像> fc-list : family | sort -u   # 只应出现 Windows 字体，不能有 Liberation / DejaVu
   docker run --rm <镜像> fc-match sans-serif          # 应为 Arial
   docker run --rm <镜像> fc-list :lang=zh family      # 不能为空
   ```

Mac 账号：苹方、SF 等 Apple 字体的授权只限 Apple 硬件，无法在 Linux 上合规安装，Mac 身份在字体探测上弱于 Windows 身份。在 Mac 开发机上跑 Windows 账号时，字体探测会看到 Mac 字体，属于开发环境的差异，不代表生产环境。

## 代理与地区
账号的时区、语言要和代理出口 IP 的地区一致（如上海时区的账号用国内出口）。不一致时 BrowserScan 这类检测会扣分（实测 -10%）。配置了代理时，库会在 profile 里禁止 WebRTC 绕过代理直连 UDP，真实公网 IP 不会经 STUN 泄露。

## 验证
`scripts/test-linux-chrome.sh` 在同构环境里跑全部测试，其中 `TestIdentityAcrossSurfaces` 校验请求头、主页面、跨站 iframe、专用 / 共享 Worker、弹窗的身份一致。

## 已知残余风险
软件渲染的 canvas / WebGL 图像特征、`speechSynthesis.getVoices()` 语音列表、Mac 身份字体、不走代理时服务器的 TCP / IP 指纹为 Linux、出口 IP 是机房 IP 时 Cloudflare 托管质询可能无论如何都不放行（原版 Chrome 也一样）。
