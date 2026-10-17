# 移动输入（移动身份子项目 M2）设计

状态：已定稿（2026-10-17）。用户指示"完成所有 M 的任务，中途不要停下来问"，标注"实现者决定"的选择题均由实现者按推荐项自行决定。

## 1. 背景

- M1 之后，手机身份（安卓）的视口、触屏能力、UA 都是手机，但页面操作仍是桌面方式：
  - `Click`、`RealClick`、`ClickSelector`、`RealClickSelector` 发的是鼠标事件（`pointerType: mouse`）；
  - `RealScroll` 发的是滚轮事件；
  - `RealSendKeys`、`SendKeys`、`Type` 发的是美式键盘的物理按键（`key: "a"`、`keyCode: 65`）。
- 真机上这三类操作全然不同，风控很容易看出。
- 真机基准：`test/baseline/android-chrome-pgkm10.json` 的 `interaction`，第二轮采集，带 `rt` 真实时间。

### 1.1 真机实测（一加 Ace，Chrome 138，Android 15）

**点按**（同一个点）：
- 事件顺序：`pointerover` → `pointerenter` → `pointerdown`（`pressure` 1）→ `touchstart`（`radiusX/Y` 0.104，`force` 1）。
- 约 40ms 后：`pointerrawupdate` → `gotpointercapture` → `pointermove`。位置没变，但接触面变大（`width` 0.458），所以产生了 `pointermove`。
- 约 80ms 时：再一次 `pointermove`（`width` 0.5）。
- 约 90ms 时：`pointerup`（`pressure` 0）→ `lostpointercapture` → `pointerout` → `pointerleave` → `touchend`（`radiusX/Y` 0.25，`force` 0）。
- 随后是兼容鼠标事件：`mouseover` → `mouseenter` → `mousemove` → `mousedown` → `mouseup` → `click`（`click` 的 `pointerType` 为 touch）。
- 其他属性：坐标是亚像素小数（如 197.458），不在设备像素网格上；`pointerId` 为 2；`isPrimary` 为 true。

**滑动**：
- 顺序：`pointerdown` → `touchstart` → `touchmove`…
- 开始滚动后出现 `pointercancel`（浏览器接管平移）。之后 `touchmove` 约每 16ms 一个，全部 `cancelable: false`。最后 `touchend`。
- 页面滚动位置逐帧变化。

**打字**（手机输入法，组字方式）：
- 第一个字：`keydown`（`key: "Unidentified"`、`code: ""`、`keyCode` 229）→ `compositionstart` → `compositionupdate`（`data` 为目前整个词）→ `beforeinput` / `input`（`insertCompositionText`，`isComposing: true`）→ 约 5ms 后 `keyup`（229）。
- 后面每个字：同样的顺序，但没有 `compositionstart`；`compositionupdate` 的 `data` 依次为 `a`、`ab`、`ab1`。
- 回车：`keydown`（`key: "Enter"`、`code: ""`、`keyCode` 13）→ `keypress` → `beforeinput`（`insertLineBreak`）→ `keyup`。
- 失去焦点时 `compositionend`。
- 字与字间隔约 150ms 以上。

## 2. 目标

手机身份下，库的点按、滑动、打字方法产生的事件序列与属性和真机一致：
- 事件类型顺序；
- `pointerType`、`pressure`、`width`、触点半径、`force`；
- `keyCode` 229、组字事件；
- `cancelable` 由浏览器原生决定；
- 时间间隔在真机的范围内。

桌面身份的行为完全不变。

## 3. 设计

### 3.1 点按（`Click`、`RealClick`、`ClickSelector`、`RealClickSelector`）

- 手机身份下改为一次手指点按，经 CDP `Input.dispatchTouchEvent` 发出。浏览器开着触屏模拟（M1），会原生合成 pointer 事件、兼容鼠标事件与 `click`。
- 一次点按的步骤：
  1. `touchStart`：半径 0.104，`force` 1；
  2. 等 30–60ms，`touchMove`：同一位置，半径 0.229；
  3. 等 30–50ms，`touchMove`：半径 0.25；
  4. 等 8–25ms，`touchEnd`。
  - 按住总时长约 70–130ms，与真机 90ms 同量级。
- 坐标：手指不会正好落在给定点，加 ±1.5 CSS 像素以内的随机偏移，并保留亚像素小数（以 1/24 像素为单位，与真机精度同级）。
- `Click` 与 `RealClick` 在手机上行为相同：手指没有"移动轨迹"，也没有悬停。
- `RealHover` 在手机身份下什么都不做，返回 nil：触屏没有悬停（实现者决定）。

### 3.2 滑动（`RealScroll`）

- 手机身份下改为手指拖动，方向与滚动方向相反（向下滚动就是手指向上拖）。
- 起点在视口中部附近、随机偏移。
- 一次拖动的距离不超过视口高度（或宽度）的 60%。要滚的距离更长时，分成多次拖动，每次之间停 120–250ms。
- 拖动过程：
  - `touchStart`；
  - 每 16ms 一个 `touchMove`，按缓出曲线移动（先快后慢）；末段放慢到接近静止，避免浏览器触发惯性滚动，使实际滚动距离接近请求值；
  - `touchEnd`。
- 每次拖动 120–260ms。
- 返回前等滚动停下（最多 1 秒）。

### 3.3 打字（`RealSendKeys`、`Type`、`SendKeys`）

- 手机身份下按真机输入法的组字方式：
  - 字母、数字、符号：`keydown`（Unidentified / 229）→ `Input.imeSetComposition`（当前整个词）→ 5–10ms 后 `keyup`（229）。字间隔 120–260ms。
  - 空格：先提交当前词（`Input.insertText` 提交组字，产生 `compositionend`），再按同样方式输入空格并直接提交。
  - 换行（`\n`）：先提交当前词，再发 `keydown` / `keypress` / `keyup`（`key: "Enter"`、`code: ""`、`keyCode` 13），与真机一致。
  - 文本结束时提交当前词，保证输入框里的值就是请求的文本。
- `SendKeys(selector, text)`：先点按元素（3.1），再 `focus()`，然后按上面的方式输入。
- 只模拟安卓。iOS 的打字方式在 M3 处理。

### 3.4 实现位置

- 只改默认通道的 `CustomCDPPage`：移动身份不支持 chromedp 通道（M1）。
- 新建 `mobile_input.go`，提供点按、滑动、组字三个函数；它们只依赖一个发送 CDP 命令的函数，便于单独测试。
- `CustomCDPPage` 的这几个方法开头判断 `p.identity.Screen.Mobile`，是手机身份时转到上面的函数。

## 4. 测试

- 集成测试（默认通道，安卓身份 `PGKM10`）：
  - 打开采集页（M0），用库的方法完成三步（点按钮、输入 `ab1`、在滑动区里滑动），调用 `finishBaseline()`，取上传的交互记录；
  - 与真机基准的交互记录比较。
- **点按**：步骤 `tap` 的事件类型顺序与真机一致：
  - 去掉连续重复的 `pointerrawupdate` / `pointermove`、`pointerenter` / `pointerleave` 次数之后比较；
  - `pointerdown` 的 `pointerType` 为 touch、`pressure` 1、`width` < 1；
  - `pointerup` 的 `pressure` 0；
  - `click` 的 `pointerType` 为 touch；
  - 按住时长 50–150ms。
- **打字**：
  - 所有 `keydown` 的 `keyCode` 都是 229、`key` 为 Unidentified、`code` 为空；
  - 有 `compositionstart`，`compositionupdate` 的 `data` 依次为 `a`、`ab`、`ab1`；
  - `beforeinput` 为 `insertCompositionText`；
  - 最后输入框的值为 `ab1`。
- **滑动**：
  - 有 `touchstart`、多个 `touchmove`（全部 `cancelable: false`）和 `touchend`；
  - 有 `pointercancel`；
  - 滑动区的 `scrollTop` 变化了，并且与请求的距离相差不超过 30%；
  - 相邻 `touchmove` 的间隔中位数在 12–24ms 之间。
- **回车**：`RealSendKeys("x\n")` 产生 `key: "Enter"`、`keyCode` 13、`code` 为空的 `keydown`，以及 `insertLineBreak`。
- **桌面身份不变**：现有的 `TestInputLooksHuman` 照常通过。
- macOS 与 Linux 容器全量通过。

## 5. 残余风险

- 点按位置和压力变化来自一台真机的样本，不同手指、不同手机会有差异。
- iOS 的点按与打字（M3）不同于安卓，这里不处理。
- `pointerId` 由浏览器分配。真机为 2，模拟时以实测为准，不伪装。
