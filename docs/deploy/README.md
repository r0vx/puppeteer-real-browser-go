# 生产部署说明

## 环境
- Linux amd64 + Google Chrome 稳定版（不是发行版的 chromium 包：品牌与编解码器不同）。
- 无 GPU 时 WebGL 走 SwiftShader 软件渲染；库会把显卡标识与能力参数换成账号档案里的真实显卡，但渲染出来的图像特征仍是软件渲染（已知残余风险）。
- 容器用 tini 或 `docker run --init` 作 1 号进程，回收 Chrome 辅助进程。

## 字体
Windows 账号会被字体探测检查，需要安装 Windows 常用字体：微软雅黑（msyh）、宋体（simsun）、黑体（simhei）、Arial、Times New Roman、Segoe UI、Calibri、Consolas。字体需自备合法授权，放在 `fonts/windows/` 后按 `production.Dockerfile` 构建。

Mac 账号：苹方、SF 等 Apple 字体的授权只限 Apple 硬件，无法在 Linux 上合规安装，Mac 身份在字体探测上弱于 Windows 身份。

## 验证
`scripts/test-linux-chrome.sh` 在同构环境里跑全部测试，其中 `TestIdentityAcrossSurfaces` 校验请求头、主页面、跨站 iframe、专用 / 共享 Worker、弹窗的身份一致。

## 已知残余风险
软件渲染的 canvas / WebGL 图像特征、`speechSynthesis.getVoices()` 语音列表、Mac 身份字体、不走代理时服务器的 TCP / IP 指纹为 Linux。
