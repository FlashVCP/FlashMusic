# FlashMusic

基于 Go 和 [MyGo](https://github.com/egoist/mygo) 开发的原生跨平台多源音乐客户端。原生 GPU 自绘界面，无 WebView、无 Electron，秒开且内存占用极低。

支持 **本地音乐库**、**网易云音乐** 及未来 **云端私有音源** 的统一调度与高品质播放。

## 核心特性

- **多音源支持**：
  - **本地音乐**：支持本地文件夹扫描、ID3 / FLAC 元数据解析、同名 `.lrc` 歌词与内嵌封面渲染
  - **网易云音乐**：完整保留原版特性（扫码登录、每日推荐、歌单、排行榜、私人 FM、无损/Hi-Res 边下边播等）
  - **云端音源（规划中）**：支持挂载 WebDAV / S3 / 私有流媒体服务
- **原生高审美与物理动效**：
  - 动态背景：根据封面色彩自适应生成柔和漂移的毛玻璃渐变背景
  - 律动歌词：支持逐行/逐字同步滚动，阻尼弹簧物理动效，大字“倾诉”与“云阶”排版
  - 3D 选歌封面墙：队列曲目自适应排布，支持触控板滑动甩动并跟随音乐节拍律动
  - 刘海播放器（macOS 特供）：嵌入屏幕顶部硬件刘海的 Mini 控制器
- **高品质播放管道**：
  - 纯 Go 音频解码（`go-mp3`、`flac`），支持最高 Hi-Res 24bit/96kHz 及无损格式
  - 8ms 余弦平滑淡入淡出（Cosine Easing），切歌与暂停零爆音
  - 本地歌曲智能缓存，音量均衡（Normalization），记忆上次播放进度与队列
- **离线端侧 AI**：
  - 播客节目支持本地运行 `Silero VAD` + `Qwen3-ASR`（基于 sherpa-onnx），音频不上云实时生成滚动字幕

## 构建与运行

需要 Go 1.27+。

```bash
# 1. 本地开发与热重载调试
go tool mygo dev

# 2. Windows 平台官方标准打包（自动嵌入 ICO、Manifest 与 NSIS 安装包）
go tool mygo build -platform windows/amd64
# 也可以直接运行一键脚本：
./scripts/build-windows.bat
# 或 npm 快捷脚本：
npm run build:windows

# 3. 运行测试
go test ./internal/...
```

> **注意**：请务必使用 `go tool mygo build` 或上述标准脚本打包，**严禁在根目录直接执行裸 `go build`**。裸命令无法将高分屏 Manifest、官方 ICO 图标及安装包资源段注入二进制，会导致可执行文件缺失图标和安装包。产物固定输出至 `build/windows-amd64/`。

## 上游与致谢

- 本项目基于 [egoist/mygo-cloudmusic](https://github.com/egoist/mygo-cloudmusic) 进行多源扩展与重构演进。
- [MyGo](https://github.com/egoist/mygo)：纯 Go 原生 UI 与应用框架
- [NeteaseCloudMusicApi](https://github.com/Binaryify/NeteaseCloudMusicApi)：eapi 加密与协议参考
- [folia-major](https://github.com/chthollyphile/folia-major)：歌词动画设计参考
- [sherpa-onnx](https://github.com/k2-fsa/sherpa-onnx)、[Qwen3-ASR](https://github.com/QwenLM/Qwen3-ASR) 与 [Silero VAD](https://github.com/snakers4/silero-vad)：端侧语音识别

## 许可证

[AGPL-3.0](LICENSE)
