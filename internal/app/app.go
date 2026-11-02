package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/player"
	"netease-music-mygo/internal/source"
)

type pageKind int

const (
	pageHome pageKind = iota
	pageDaily
	pageCharts
	pagePlaylist
	pageSearch
	pageFM
	pageArtist
	pageAlbum
	pageUser
	pagePodcast
	pagePodcasts
	pageLocal
	// 视界中心 (Video & Live)
	pageDramas      // 微短剧场
	pageShortVideos // 潮流短视频
	pageLive        // 互动直播
	pageVoiceRoom   // 语音空间
	pageCinema      // 院线影视
	// 边缘节点 (Cloud Node)
	pageNodeBoard    // 节点看板
	pageNodeStorage  // 缓存空间
	pageNodeEarnings // 收益中心
)

type route struct {
	page pageKind
	// id is the playlist, artist, album, user or podcast the page shows.
	id int64
	// query and kind are what a search page searched for.
	query string
	kind  netease.SearchType
}

// app is the state both windows show. Everything in it is read and changed
// on the main thread: by the views, and by goroutines through post.
type app struct {
	win   *mygo.Window
	notch *notch
	tray  *mygo.Tray // the icon in the menu bar or notification area, while it shows
	art   *artCache
	pl    *player.Player
	pal   palette

	route   route
	history []route
	forward []route // the pages gone back from, the last first to go to

	profile      *netease.Profile
	mine, saved  []netease.Playlist // the listener's playlists: made, and saved
	recPlaylists []netease.Playlist // the day's playlists picked for the listener
	recCategory  string             // category pill for 今日为你推荐
	featCategory string             // category pill for 推荐歌单
	login        loginState
	accountOpen  bool           // the dropdown of the account row shows
		naming       nameDialog     // the name of a playlist asked for
		deleting     deleteDialog   // whether to delete a playlist, asked
		closeConfirm closeConfirmDialog // confirm modal when closing the main window
		liked        map[int64]bool // the songs the listener likes
	toast        string         // a message to show once
	picked       pickedUp       // the song a login picked up, waiting to play

	// The episodes the listener likes, and doesn't, as far as NetEase has
	// answered, and those asked about: see ui_like.go.
	likedPrograms, askedPrograms map[int64]bool

	daily       []netease.Track
	dailySpin   spinState // the record on the daily card
	npSpin      spinState // the record on Now Playing cover
	dailyBars   barState  // the bars on its play button
	featured    []netease.Playlist
	charts      []netease.Playlist
	homeErr     string
	homeLoading bool
	homeAt      time.Time // when what the home page shows was asked for

	playlists  map[int64]*netease.Playlist
	loadingPl  map[int64]bool
	plErr      map[int64]string
	artists    map[int64]*artistData
	artistTab  int // what an artist's page lists: artistHot, artistLatest, artistAlbums or artistEvents
	albums     map[int64]*albumData
		users      map[int64]*userData
		podcasts   map[int64]*podcastData
		podcastTab *podcastTab // the podcasts tab's lists

		localTracks        []netease.Track
		localRecentTracks  []netease.Track
		localAlbums        []*source.LocalAlbum
		localArtists       []*source.LocalArtist
		localFolders       []*source.LocalFolder
		localDirs          []string
		localScanning      bool
		localErr           string
		localDupesCount    int
		localTab           int // 0: 单曲, 1: 专辑, 2: 歌手, 3: 文件夹, 4: 最新添加
		localQuery         string
		localSearchBox     ui.Handle
		localCollapsed     bool
		localFilterType    string
		localFilterName    string
		localFilterSongs   []netease.Track
		propTrack          *source.LocalTrack
		propOpen           bool
		songMenuTrack      *netease.Track
		songMenuOpen       bool

		// 视界中心 (Video & Live) 状态 - 对齐站长端默认栏目标签与分类
		activeDramaTab      int       // 0: 精选大厅, 1: 剧场播放, 2: 追剧历史
		dramaType           string    // 漫剧 / 真人剧 / AI剧 / 演员
		dramaRanking        string    // 推荐榜 / 热播榜 / 新剧榜 / 追更榜 / 必看榜...
		dramaCategory       string    // 题材分类：都市 / 爽剧逆袭 / 战神 / 古装 / 甜宠 / 悬疑...
			dramaQuery          string    // 剧目搜索关键词
			dramaSearchBox      ui.Handle // 剧目搜索输入框句柄
				activeDramaID       int64     // 当前选中的播放剧目 ID
				activeEpisodeIndex  int       // 当前播放的分集索引 (0-based)
					dramaEpRangeStart   int       // 选集分页分段起始集 (0: 1-50, 50: 51-100...)
					dramaWidescreenMode bool      // 默认 true: PC 宽屏影院模式 (16:9 舞台 + 右侧抽屉); false: 9:16 竖屏模式
					dramaTheaterExpanded bool     // 影院沉浸放大模式 (全宽大舞台，收起右侧选集抽屉)
					dramaAutoNext       bool      // 自动连播开关 (默认 true)
				dramaDrawerScroll   ui.ScrollState
				dramaHistory        []int64   // 追剧历史剧目 ID 列表
			dramaPlaying        bool      // 是否正在播放短剧视频

			// 院线影视 (Cinema) 状态
			cinemaChannel       string    // 频道：精选影视 / 经典电影 / 漫剧 / 4K蓝光 / 纪录片
			cinemaGenre         string    // 题材：全部 / 动作 / 喜剧 / 爱情 / 都市 / 悬疑 / 古装 / 奇幻 / 战争 / 科幻
			cinemaQuery         string    // 影视搜索关键词
			activeCinemaID      string    // 当前选中的院线电影 ID

			// 潮流短视频 (Short Video) 状态 - 对齐母带中心与选品池
			svCategory   string // 归档分类：全部 / 潮流舞蹈 / 魅惑性感 / 颜值变装 / 秘境探险...
			svTag        string // 标签：全部 / 热门 / 高能 / 治愈 / 神同步 / 变装...
			svDuration   string // 时长筛选：全部 / 竖屏短视频 (<60s) / 精选片段 (1~5分钟) / 深度中视频 (5~30分钟)...
			svHeat       string // 热度门槛：全部 / 100万+ 爆款 / 10万+ 热门 / 1万+ 视频
			svSort       string // 排序：综合 / heat (最热播放) / likes (最高点赞)
			svQuery      string // 短视频搜索关键词
			activeSVID   string // 当前选中的短视频 ID
			svPlaying    bool   // 是否正在全屏/弹窗播放短视频
			svPresetOpen bool   // 是否展开预设组面板

			// 全局视频播放 WebView 组件
			videoWeb     *mygo.WebView // 内嵌 WebView2 播放器视口
			videoWebURL  string        // 当前 WebView 加载的 URL

		// 边缘节点 (Cloud Node) 状态 - 对齐 FlashMate 桌面端与 FlashVCP Cloud 模型
		nodeActive          bool      // 边缘节点运行状态: running / paused
		nodeStatus          string    // "running", "waiting", "draining", "paused", "offline"
		nodeCreditTier      string    // AAA / AA / B / probation
		nodeID              string    // 节点唯一身份 ID (如 fm-node-cn-8f4b)
		nodeSessionStart    time.Time // 当前会话启动时间
		nodeUploadSpeed     float64   // 实时上传速率 (MB/s)
		nodeDownloadSpeed   float64   // 实时下载速率 (MB/s)
		nodeUploadCapMbps   int       // 用户设置的最大上传带宽限制 (10~1000 Mbps)
		nodeTodayTrafficGB  float64   // 今日上行贡献流量 (GB)
		nodeCachedBytes     int64     // 已缓存切片容量
		nodeTotalQuotaBytes int64     // 存储配额总量
		nodeEarnedXFB       float64   // 累计赚取 XFB 资产
		nodeOffsetRMB       float64   // 累计带宽抵扣金 (元)
		nodeStandbyMinor    string    // 待机奖励 (元)
		nodeBusinessMinor   string    // 业务分发奖励 (元)
		nodeScheduleMode    string    // "continuous", "manual", "scheduled"
		nodeWakeOnShare     bool      // 共享时阻止系统休眠
		nodePauseOnBattery  bool      // 电池供电时自动暂停
		nodeTab             int       // 0: 概览, 1: 环境检测, 2: 共享配置, 3: 收益与优惠券, 4: 任务记录

		trackList  ui.ListState
	// Where the page and the sidebar are scrolled, kept while Now Playing
	// covers them, unbuilt.
	pageScroll ui.ScrollState
	navScroll  ui.ScrollState
	scrolled   route // the page trackList and pageScroll are of

	mineCollapsed, savedCollapsed bool

	query        string
	searchBox    ui.Handle
	topSearchBox ui.Handle
	hotSearches  []netease.HotSearchItem
	hotLoading   bool
	searches     map[route]*searchData // the results of each search, by its page
	searched     route                 // the search last shown, which the sidebar goes back to
	suggestions  suggestState          // what the search field's query may be after

	// 搜索与发现中的歌单分类导航状态 (AlgerMusicPlayer 架构)
	searchCat     string                        // 当前选中的歌单分类，默认 "每日推荐"
	searchCatList []string                      // 歌单分类标签列表 (来自 /api/playlist/catalogue)
	catPlaylists  map[string][]netease.Playlist // 按分类缓存的歌单列表
	catLoading    bool                          // 是否正在加载分类歌单
	catErr        string                        // 分类歌单加载错误
	catScroll     ui.ScrollState                // 歌单分类横向滑动状态

	nowPlaying  bool
	npOpen      bool      // Now Playing as it last showed: open or not
	npChangedAt time.Time // when it last opened or closed
	npPane      int       // paneLyrics, paneQueue, paneStage, paneComments, paneProgram or paneCaptions
	npPaneTop   float32   // how far the cover beside the pane starts below its top
	lyricScroll ui.ScrollState
	lyricGlide  lyricGlide
	lyricAt     int
	lyricTrack  int64
	queueList   ui.ListState
	queueAt     int // the row of the queue pane last scrolled to, -1 while it is hidden
	comments    commentState
	tabCount    rollState // the count on the comments' tab, as it rolls

	fmMode        int    // the FM mode chosen, in fmModes
	radioStarting string // the radio asked for its first songs, or ""

	scrub      float64
	scrubbing  bool
	volume     float64
	volumeOpen bool // the volume popover of the player bar shows
	mutedFrom  float64

	rowBars barState
	// underTop and underBottom say whether the page has content under its
	// bars, which show their scroll edges then.
	underTop, underBottom bool
	barScrub              scrubState
	npScrub               scrubState
	backdropCache         backdropCache // the backdrop of Now Playing, as last painted
	npArt                 *artwork      // the cover whose colors Now Playing shows
	npCoverState          coverState    // the cover of Now Playing, as it is tilted and flicked
	wallState             wallState     // the wall of the queue's covers, as it is dragged
	quiet                 quietState    // the quiet parts of the song playing, for its hot comments
	captions              captionState  // the episode playing's captions, and the model that writes them
	sig                   playerSig
	quitting              bool

	settings    settings
	settingsWin *mygo.Window
	cacheUsed   int64 // the bytes the song cache takes, -1 until measured
	updates     bool  // the update plugin is used (useUpdater)

	restored     bool   // playback picked up where it was left
	savedSession string // the session as last kept
	sessionDue   bool   // a save of the session is due
}

// post runs fn on the main thread and redraws both windows. Without a
// window, as in tests, it drops fn.
func (a *app) post(fn func()) {
	if a.win == nil {
		return
	}
	a.win.Update(func() {
		fn()
		a.redraw()
	})
}

func (a *app) redraw() {
	if a.win != nil {
		a.win.Invalidate()
	}
	if a.notch != nil && a.notch.win != nil {
		a.notch.win.Invalidate()
	}
}

// shotHook, when set, drives the app to check its design.
var shotHook func(a *app)

func (a *app) showMain() {
	if a.win != nil {
		if a.win.IsMinimized() {
			a.win.Restore()
		}
		a.win.Show()
		a.win.Focus()
		// From the menu bar alone the app may be inactive.
		mygo.App.Focus()
	}
}

// Run runs the app until it quits.
func Run() error {
	// Enable HEVC / H.265 hardware decoding and allow seamless autoplay in embedded WebView2
	if os.Getenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS") == "" {
		_ = os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--enable-features=PlatformHEVCDecoderSupport --autoplay-policy=no-user-gesture-required")
	}

	a := &app{
		playlists:  map[int64]*netease.Playlist{},
		loadingPl:  map[int64]bool{},
		plErr:      map[int64]string{},
		artists:    map[int64]*artistData{},
		albums:     map[int64]*albumData{},
		users:      map[int64]*userData{},
		podcasts:   map[int64]*podcastData{},
		podcastTab: &podcastTab{},
		route:      route{page: pageHome},
		activeDramaTab:      0,
			nowPlaying:          false,
			npPane:              paneLyrics,
			searches:            map[route]*searchData{},
			searchCat:           "每日推荐",
			catPlaylists:        map[string][]netease.Playlist{},
			lyricAt:             -1,
			queueAt:             -1,
			cacheUsed:           -1,
			dramaWidescreenMode: true,
			dramaAutoNext:       true,
			nodeActive:          true,
			nodeStatus:          "running",
			nodeCreditTier:      "AAA",
			nodeID:              "fm-node-cn-east-8f4b",
			nodeSessionStart:    time.Now().Add(-3*time.Hour - 24*time.Minute),
			nodeUploadSpeed:     4.85,
			nodeDownloadSpeed:   18.20,
			nodeUploadCapMbps:   35,
			nodeTodayTrafficGB:  12.4,
			nodeCachedBytes:     28 * 1024 * 1024 * 1024,
			nodeTotalQuotaBytes: 100 * 1024 * 1024 * 1024,
			nodeEarnedXFB:       68.5,
			nodeOffsetRMB:       137.0,
			nodeStandbyMinor:    "12.80",
			nodeBusinessMinor:   "124.20",
			nodeScheduleMode:    "continuous",
			nodeWakeOnShare:     true,
			nodePauseOnBattery:  true,
			nodeTab:             0,
		}
		moveOldData()
		a.useUpdater()
		mygo.App.WhenReady(func() {
			a.settings = loadSettings()
			a.settings.LyricsStage = false
			a.settings.applyAppearance()
			a.npPane = a.lyricsPaneChosen()
			// Before the window: on Windows it lays out its content as it is
			// made, and the view asks the player what plays.
			a.art = newArtCache(a.post)
			a.pl = player.New(a.post)
			a.pl.Quality = func() string { return a.settings.Quality }
			a.pl.Normalize = func() bool { return a.settings.Normalize }
			a.pl.Cache = newSongCache(a.settings.CacheMB)
			a.pl.Changed = func() {
				a.keepSession()
				a.syncCaptions()
			}
			a.volume = a.settings.Volume
			a.pl.SetVolume(a.volume)

				// Default desktop bounds referencing AlgerMusicPlayer (1782x1059 / user reference screenshot 1848x1086)
				initW, initH := 1782, 1059
				if appData := os.Getenv("APPDATA"); appData != "" {
					algerCfgPath := filepath.Join(appData, "AlgerMusicPlayer", "config.json")
					if data, err := os.ReadFile(algerCfgPath); err == nil {
						var cfg struct {
							WindowState struct {
								Width  int `json:"width"`
								Height int `json:"height"`
							} `json:"windowState"`
						}
						if err := json.Unmarshal(data, &cfg); err == nil && cfg.WindowState.Width >= 1280 && cfg.WindowState.Height >= 720 {
							initW = cfg.WindowState.Width
							initH = cfg.WindowState.Height
						}
					}
				}
				if d := mygo.Screen.PrimaryDisplay(); d.WorkArea.Width > 0 && d.WorkArea.Height > 0 {
					maxW := int(float64(d.WorkArea.Width) * 0.96)
					maxH := int(float64(d.WorkArea.Height) * 0.96)
					if initW > maxW {
						initW = maxW
					}
					if initH > maxH {
						initH = maxH
					}
					if initW < 1280 {
						initW = min(1280, d.WorkArea.Width)
					}
				}

				a.win = mygo.NewWindow(mygo.WindowOptions{
					Title:         mygo.App.Name(),
					Width:         initW,
					Height:        initH,
					MinWidth:      1280,
					MinHeight:     720,
					StateKey:      "main",
					TitleBarStyle: mygo.TitleBarHidden,
					// As tall as the edge over a page (Linux, Windows).
					TitleBarHeight: buttonsHeight,
					// Inset from the corner, within the sidebar, as macOS 26's
					// apps with a sidebar place them.
					TrafficLightPosition: &mygo.Point{X: 20, Y: 20},
					Vibrancy:             mygo.VibrancySidebar,
					BackgroundColor:      "light-dark(#ffffff, #141416)",
					Content:              ui.View(a.view),
				})

				// Ensure stale legacy window state (<= 1366 or <= 800) is upgraded to match AlgerMusicPlayer's spacious layout
				if curW, curH := a.win.Size(); curW <= 1366 || curH <= 800 {
					a.win.SetSize(initW, initH)
					a.win.Center()
				}
		a.setMenu()
		a.boot()
		// The notch player is macOS's, over its menu bar: elsewhere it would
		// hang at the top of the screen as a black box.
		if runtime.GOOS == "darwin" {
			a.notch = newNotch(a)
		}
		// From the menu bar alone, the Dock icon shows while the window
		// does.
		a.syncDock()
		a.win.OnShow(a.syncDock)
		a.win.OnHide(a.syncDock)
		// Shown or switched to the morning after, it shows the new day's.
		a.win.OnFocus(a.freshenHome)
		a.applyTray()
		if shotHook != nil {
			shotHook(a)
		}
			// Closing the window: follow user preference (ask / minimize / close)
			a.win.OnClose(func(e *mygo.CloseEvent) {
				if a.quitting {
					return
				}
				e.PreventDefault()
				action := a.settings.CloseAction
				if action == "" {
					action = "ask"
				}
				switch action {
				case "minimize":
					if runtime.GOOS == "darwin" || a.tray != nil {
						a.win.Hide()
					} else {
						a.win.Minimize()
					}
				case "close":
					a.quitting = true
					mygo.App.Quit()
				default: // "ask"
					a.closeConfirm.open = true
					a.closeConfirm.remember = false
					a.redraw()
				}
			})
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) {
		a.quitting = true
		a.settings.Volume = a.volume
		a.settings.save()
		a.saveSession()
		if write := a.transcriptWrite(); write != nil {
			write()
		}
	})
	// Windows leaves an icon in the notification area after its app until
	// the pointer passes over it.
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		if a.tray != nil {
			a.tray.Destroy()
		}
	})
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnActivate(func(bool) { a.showMain() })
	return mygo.App.Run()
}
