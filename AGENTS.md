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

## 4. 提交前必须编译通过 (Compile-Before-Commit — 强制)

**任何提交前，必须先本地跑通编译，严禁提交编译不过的代码。**

- 提交前强制执行并确认全部通过：
  ```bash
  go build ./...        # 必须 exit 0，不允许任何 package 编译失败
  go vet ./... 2>&1     # 建议同时跑，排查可疑写法
  ```
- **严禁**在 `go build ./...` 报错的状态下 `git commit`。哪怕只有一个子包（如 `internal/vcp`）编译失败也不行。
- **真实事故（2026-11-02，必须牢记）**：提交 `7a1fd04` 时 `internal/vcp/dramas.go` 编译不过（调用了未定义的 `extractCategoriesLocked`，并混入了一段错位的孤立代码导致语法错误）。由于该包编译失败，`go tool mygo build` **静默复用了上一次成功编译的旧缓存二进制**，使得之后每次"重新打包"分发的都是同一个旧的脏二进制。这直接导致：
  - 所有后续代码改动（修 PE 结构、改 syso、改元数据）全部没有生效，产物始终是老文件；
  - 该老二进制早被杀毒软件记入特征缓存，用户安装后反复被报 `Trojan:Win32/Bearfoos.A!ml` / `HEUR/QVM202.*`。
- **教训**：`mygo build` 显示 "Built ... successfully" **不代表**用的是你最新的源码——如果任一依赖包编译失败，它可能回退到缓存。因此发布前必须：
  1. 先 `go build ./...` 确认全绿；
  2. 构建后核对产物二进制的 SHA256 与源码改动一致（新代码必然产生新哈希）。

## 5. 杀毒误报的正确排查顺序 (Antivirus False-Positive Triage — 强制)

当 Windows Defender / 360 等报 FlashMusic 为病毒（`!ml`、`HEUR`、`QVM` 等机器学习/启发式名称）时，**先按以下顺序排查，禁止上来就瞎改 PE 结构、加壳参数或删业务代码**：

1. **先确认编译是否真通过**：`go build ./...`。编译失败 → 产物是脏缓存，先修编译（见第 4 节），这是最高频根因。
2. **核对产物哈希**：重新构建后，产物 `FlashMusic.exe` 的 SHA256 必须变化。若没变，说明在分发旧缓存文件。
3. **隔离对照实验**：分别把 `internal/speech`、`internal/netease`、`internal/vcp`、完整 app 编译成独立二进制，逐个扫描，定位真正触发模块——不要凭猜测归因。
4. **区分"文件静态内容报毒"与"安装/运行行为被拦"**：先扫桌面上的独立文件，再做完整安装后扫描；报毒记录里的文件路径指向哪个文件，就查哪个。
5. 只有在确认是**无签名可执行文件本身**被 ML 引擎误判后，才考虑：代码签名证书（Windows Authenticode，**不是** Apple 证书）或向微软/360 提交误报申诉。

**禁止**：在未定位根因前，盲目修改 `mygo` 模块源码（go module cache）、删除业务模块、或声称"已彻底解决"而无实机验证证据。

## 6. 远程验证的会话隔离注意 (Remote Session Caveat)

通过 SSH 远程在 Windows 上启动 GUI 程序时，SSH 进程通常在 **Session 0**（服务会话，无可见桌面），启动的窗口画在用户看不见的桌面上，且 `MainWindowHandle` 可能为 0。**这不代表程序打不开**。要验证 GUI 是否正常显示，必须让用户在其登录的交互桌面会话中双击运行，或用计划任务 `/IT` 投递到交互会话——不要仅凭 SSH 侧的 `hwnd=0` 判定程序有问题。

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
