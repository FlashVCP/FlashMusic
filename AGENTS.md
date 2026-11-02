# AGENTS.md

## 构建与打包规范 (Build & Packaging Standards)

所有在此仓库作业的 AI 助手与开发者必须严格遵守以下打包与编译准则，不得随意在根目录输出二进制文件或遗漏应用品牌资产：

### 1. 严禁事项 (Forbidden)
- **严禁** 在项目根目录下执行裸 `go build`、`go build -o FlashMusic.exe .` 或产生任何散落在根目录的 `.exe` 二进制文件。
- 裸 `go build` 会绕过 MyGo 原生资源嵌入管道，导致以下严重问题：
  - 缺失 Windows 资源段（`.syso`），可执行文件在资源管理器和任务栏中失去高保真 ICO 图标；
  - 缺失 Windows 高分屏 DPI 感知清单（Manifest）；
  - 无法生成官方 NSIS 桌面安装包；
  - 污染 Git 工作区并破坏干净的工作树。

### 2. 官方标准构建命令 (Standard Build Commands)
所有构建必须通过项目官方的 `mygo` 构建工具进行：
- **Windows 标准打包（生成 exe 及安装包）**：
  ```bash
  go tool mygo build -platform windows/amd64
  # 或者运行标准脚本：
  ./scripts/build-windows.bat
  # 或者通过 npm 运行：
  npm run build:windows
  ```
- **构建输出目录规范**：
  所有产物固定且唯一输出至 `build/windows-amd64/` 目录：
  - `build/windows-amd64/FlashMusic.exe`：内嵌官方 ICO 图标与 DPI 清单的正式可执行文件；
  - `build/windows-amd64/flashmusic-setup-0.1.0-amd64.exe`：带官方安装徽标的 Windows NSIS 安装包；
  - `build/windows-amd64/FlashMusic.ico` & `icon.ico`：全尺寸 Windows 独立图标文件；
  - `build/windows-amd64/icon.png` & `logo-color.png`：官方图像资源备份。
- **本地开发与热重载调试**：
  ```bash
  go tool mygo dev
  ```

### 3. 图标、托盘与品牌资产规范 (Brand, Icon & Tray Standards)
每次修改或构建时，必须确认以下官方品牌资产完备且保持引用生效：
- **应用主图标 (App Icon)**：
  - 统一高保真源文件：`resources/icon.png` (1024x1024 RGBA)；
  - 多尺寸 Windows 图标：`resources/FlashMusic.ico` 与 `resources/icon.ico` (包含 16, 24, 32, 48, 64, 128, 256 全尺寸)；
  - 必须在 `mygo.json` 中保持配置 `"icon": "resources/icon.png"` 及 `"windows": { "displayName": "FlashMusic 闪音" }`。
- **系统通知托盘 (System Tray)**：
  - Windows 通知区域托盘图标必须使用彩色的 `internal/app/tray/tray.png` (64x64 RGBA)；
  - macOS 菜单栏托盘使用深浅自适应的 `internal/app/tray/tray-template.png` (36x36 模板)；
  - `tray.go` 中托盘 ID 固定为 `flashmusic`，悬浮 ToolTip 固定为 `FlashMusic 闪音`。
- **界面与侧边栏品牌 LOGO**：
  - 侧边栏顶部品牌标头使用 `flashmusic://brand/icon`（内嵌 `internal/app/brand/icon.png`）；
  - 全屏与弹窗品牌展示使用 `flashmusic://brand/logo`（内嵌 `internal/app/brand/logo.png`）。

---

## Tests

Only write a test for behaviour that can actually break. If the code under test could change without the test noticing, or the test would fail on a harmless rewording, don't write it.

**Write tests for:**

- What input does: clicks, keys, drags, gestures and scrolling, and the state or effect they lead to (seeking when the progress bar is let go, Space after opening Now Playing, back and forward, flinging the cover, liking a song, leaving a radio).
- Logic that doesn't need a view: parsing lyrics, formatting times and counts, the player's queue, seeking, fading and cache, decoding streams and images.
- A regression you just fixed. Say in the test's comment what went wrong.
- Frame budgets, in `perf_test.go`, so idle views don't start asking for frames they don't need.

**Don't write tests that:**

- Check that a list of labels shows on screen. That repeats the view code and breaks whenever the wording changes.
- Check default settings or the order of rows in a form.
- Test MyGo itself, for example that `AnimateTo` or a spring moves from 0 to 1.
- Depend on the wall clock. `ui.NewTester` starts its clock at the current time, so anything with a phase taken from the clock passes or fails depending on when the test runs. Also, `FramePending` ignores frames waiting on `After` or `Painter.After`.

**How:**

- Build views with `ui.NewTester` and set up the app with `newTestApp` (in `internal/app/helpers_test.go`). Find elements by their accessibility label, then `Click`, `RightClick`, `Press`/`Move`/`Release` or `Key`, and check the app's state afterwards.
- Swap out the network and the clipboard through package variables such as `likeSong` and `copyText`, and restore the real one with `defer`.
- Live tests that call NetEase skip unless `LIVE` is set.
- Look at visuals instead of asserting on them. The `NM_FRAMES` tests save PNGs into that directory and skip when it isn't set. Check pixels only for coverage, such as whether a layer still covers the window.
- Start each test with a one-sentence comment saying the behaviour it checks, in the style of the code.
- Run `go test ./internal/...` before committing, and run a new UI test with `-count=5`. Fix or delete a flaky test; don't retry it.
