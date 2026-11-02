package app

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/vcp"
)

const sidebarWidth = 232

func (a *app) view(c *ui.Context) {
	t := *c.Theme()
	a.pal = paletteFor(&t)
	p := a.pal
	t.Accent, t.AccentText = p.accent, p.onAccent
	t.Selection = p.accent.Alpha(0.3)
	c.SetTheme(&t)

	a.shortcuts(c)

	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
	}
	root := ui.Box(c).Fill()
		if !c.Focused(a.searchBox) {
			// Space plays and pauses wherever the focus is, as in Music. On
			// the container around everything, it comes before a button the
			// pointer focused, which would take Space as a click; on the
			// window, it would come after.
			root.OnShortcut(0, ui.KeySpace, a.handleSpace)
			// And where nothing has the focus, as once Now Playing covers the
			// button that had it.
			c.OnShortcut(0, ui.KeySpace, a.handleSpace)
		}
		npOpen, npSettled := a.npState(c)
		root.Children(func() {
			// 全屏沉浸播放：当窗口全屏且处于视频播放中时，视频画面铺满整个屏幕
			if a.win != nil && a.win.IsFullScreen() && (a.dramaPlaying || a.svPlaying) && a.videoWeb != nil {
				ui.WebView(c, a.videoWeb).Fill()
				return
			}

			page := ui.Row(c).Fill().AlignItems(ui.Stretch)
			page.Children(func() {
			if npOpen && npSettled {
				// Covered by Now Playing: not built, which would cost each
				// of its frames as much again as the stage itself. Where
				// it is scrolled is the app's (pageScroll, navScroll,
				// trackList), so it shows again as it was.
				return
			}
			side := ui.Column(c).Width(sidebarWidth)
			if !c.Vibrancy() {
				side.Background(p.sidebar)
			}
			side.Children(func() { a.sidebar(c) })
			// Keyed, so that a popover of the sidebar opening or closing
			// keeps what the page and its bars hold, their edges' fades.
				ui.Box(c.Key("main")).Grow(1).Background(p.bg).BorderWidth(0, 0, 0, 1).BorderColor(p.hairline).Children(func() {
					a.content(c)
					// The bars float over the page, which scrolls under them:
					// positioned, they paint over it.
					a.pageTop(c)
					if a.isMusicPage() {
						a.musicSearchBar(c)
					}
					if !a.hidePlayerBar() {
						a.playerBar(c)
					}
				})
		})
		a.nowPlayingLayer(c, npOpen, npSettled)
	})
		a.unifiedLoginModal(c)
		a.playlistNameDialog(c)
		a.deletePlaylistDialog(c)
		a.songPropertiesDialog(c)
	if a.toast != "" {
		c.Toast(a.toast)
		a.toast = ""
	}
	a.syncWindows()
}

func (a *app) shortcuts(c *ui.Context) {
	// The side buttons of a mouse, the keys of keyboards that have them
	// and swipes between pages, and the keys of Music.
	c.OnShortcut(0, ui.KeyBack, a.back)
	c.OnShortcut(0, ui.KeyForward, a.goForward)
	c.OnShortcut(ui.Cmd, ui.KeyBracketLeft, a.back)
	c.OnShortcut(ui.Cmd, ui.KeyBracketRight, a.goForward)
		c.OnShortcut(ui.Cmd, ui.KeyF, func() {
			a.nowPlaying = false
			a.searchBox.Focus(c)
		})
		if a.win != nil && a.win.IsFullScreen() {
			c.OnShortcut(0, ui.KeyEscape, func() {
				a.win.SetFullScreen(false)
				a.redraw()
			})
		}
		if a.nowPlaying {
		c.OnShortcut(0, ui.KeyEscape, func() { a.nowPlaying = false })
		// ⌘ and a number choose Now Playing's tab of that place.
		if a.pl.Current() != nil {
			for i, t := range a.npTabs() {
				c.OnShortcut(ui.Cmd, ui.Key1+ui.Key(i), func() { a.choosePane(t.pane) })
			}
		}
	}
}

// go opens a page, remembering the one before for the back button, and
// closes Now Playing over it, though it is the page open.
func (a *app) open(r route) {
	a.nowPlaying = false
	if r == a.route {
		return
	}
	if a.svPlaying || a.dramaPlaying {
		a.stopVideo()
	}
	a.history, a.forward = append(a.history, a.route), nil
	a.show(r)
}

// show shows the page of r.
func (a *app) show(r route) {
	a.route = r
	if r.page == pageSearch {
		a.query, a.searched = r.query, r
	}
	a.load(r)
}

// load loads what the page of r shows, unless it has.
func (a *app) load(r route) {
	switch r.page {
	case pageHome:
		a.freshenHome()
	case pagePlaylist:
		a.loadPlaylist(r.id, false)
	case pageSearch:
		a.loadSearch(r, false)
	case pageArtist:
		a.loadArtist(r.id)
	case pageAlbum:
		a.loadAlbum(r.id)
	case pageUser:
		a.loadUser(r.id)
	case pagePodcast:
		a.loadPodcast(r.id)
	case pagePodcasts:
		a.loadPodcastTab()
	case pageLocal:
		if len(a.localTracks) == 0 && !a.localScanning {
			a.scanLocalDirs()
		}
	case pageDramas:
		if a.activeDramaID == 0 {
			if list := vcp.GetDramas(); len(list) > 0 {
				a.activeDramaID = list[0].ID
			}
		}
	case pageNodeBoard:
		a.nodeTab = 0
	case pageNodeStorage:
		a.nodeTab = 2
	case pageNodeEarnings:
		a.nodeTab = 3
	}
}

// back goes to the page before, as the back button of a mouse does, or
// closes Now Playing, which covers the pages.
func (a *app) back() {
	if a.nowPlaying {
		a.nowPlaying = false
		return
	}
	if len(a.history) == 0 {
		return
	}
	a.forward = append(a.forward, a.route)
	r := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.show(r)
}

// goForward goes to the page back left, as the forward button of a mouse
// does.
func (a *app) goForward() {
	if len(a.forward) == 0 {
		return
	}
	a.history = append(a.history, a.route)
	r := a.forward[len(a.forward)-1]
	a.forward = a.forward[:len(a.forward)-1]
	a.nowPlaying = false
	a.show(r)
}

func (a *app) sidebarBrandHeader(c *ui.Context) {
	p := a.pal
	ui.Row(c.Key("sidebar-brand-header")).Height(38).Padding(2, 6, 2, 6).AlignItems(ui.Center).Gap(10).Margin(0, 0, 4, 0).Children(func() {
		a.cover(c, "flashmusic://brand/icon", 64, 28, 8).Shadow(0, 2, 6, 0, p.cardShade)
		ui.Column(c).Grow(1).Gap(1).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
				ui.Text(c, "FlashMusic").FontSize(14.5).FontWeight(700).TextColor(p.text)
			})
			ui.Text(c, "闪音 · 原生高保真").FontSize(10).FontWeight(500).TextColor(p.muted)
		})
	})
}

func (a *app) sidebar(c *ui.Context) {
	// Under the window buttons where they sit over the sidebar, as the
	// traffic lights do; elsewhere, as close to the top as to the sides.
	top := float32(14)
	if c.TitleBar().Left > 0 {
		top = topBar(c) + 4
	}
	// Where nothing in it takes the press, the sidebar moves the window, as
	// the title bar it reaches up into would.
	ui.Column(c).Fill().Padding(top, 14, 14, 14).Gap(4).DragWindow().Children(func() {
		// 品牌 LOGO 与产品标头
		a.sidebarBrandHeader(c)
		// 用户头像与登录入口
		a.sidebarUserHeader(c)

			ui.Scroll(c.Key("nav")).TrackScroll(&a.navScroll).Grow(1).Margin(0, -6).Padding(0, 6).Gap(3).Children(func() {
				// 专区 1: 视界中心 (Video & Live)
				a.navSectionHeader(c, "视界中心", "VOD·LIVE")
				a.navItem(c, iconClapperboard, "微短剧场", route{page: pageDramas})
				a.navItem(c, iconSparkle, "潮流短视频", route{page: pageShortVideos})
				a.navItem(c, iconBroadcast, "互动直播", route{page: pageLive})
				a.navItem(c, iconMic, "语音空间", route{page: pageVoiceRoom})
				a.navItem(c, iconTv, "院线影视", route{page: pageCinema})

					// 专区 2: 音乐世界 (Music World)
					a.navSectionHeader(c, "音乐世界", "MUSIC")
					a.localNavGroup(c)
					a.navItem(c, iconHome, "为你推荐", route{page: pageHome})
					a.navItem(c, iconSearch, "搜索与发现", route{page: pageSearch})
					a.navItem(c, iconCharts, "排行榜", route{page: pageCharts})
					a.navItem(c, iconPodcast, "电台主播", route{page: pagePodcasts})
					a.navItem(c, iconRadio, "私人 FM", route{page: pageFM})
					if a.searched.query != "" {
						a.navItem(c, iconSearch, "搜索："+a.searched.query, a.searched)
					}
				a.myPlaylists(c, "我的歌单", a.mine, true, &a.mineCollapsed)
				a.myPlaylists(c, "收藏的歌单", a.saved, false, &a.savedCollapsed)

				// 专区 3: 边缘节点 (Cloud Node)
				a.navSectionHeader(c, "边缘节点", "ONLINE")
				a.navItem(c, iconServer, "节点看板", route{page: pageNodeBoard})
				a.navItem(c, iconFolder, "缓存空间", route{page: pageNodeStorage})
				a.navItem(c, iconCoins, "收益中心", route{page: pageNodeEarnings})
			})

			// What plays, as a card at the foot of the sidebar, over who listens.
			if tr := a.pl.Current(); tr != nil && !a.hidePlayerBar() {
				a.sidebarNowPlaying(c, tr)
			}
		a.accountRow(c)
		})
}

func (a *app) navSectionHeader(c *ui.Context, label string, tag string) {
	p := a.pal
	ui.Row(c.Key("sec-" + label)).AlignItems(ui.Center).Margin(14, 8, 4, 8).Gap(6).Children(func() {
		ui.Text(c, label).FontSize(11).FontWeight(700).TextColor(p.faint)
		if tag != "" {
			ui.Box(c).Padding(1, 5, 1, 5).Radius(4).Background(p.press).Children(func() {
				ui.Text(c, tag).FontSize(9.5).FontWeight(600).TextColor(p.accent)
			})
		}
	})
}

func (a *app) navItem(c *ui.Context, icon *ui.SVG, label string, r route) {
	p := a.pal
	selected := a.route.page == r.page && !a.nowPlaying
	b := ui.ButtonBase(c.Key(label)).Height(34).PaddingX(10).Gap(10).Radius(8).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
	bg := ui.Transparent
	switch {
	case selected:
		bg = p.press
	case b.Hovered():
		bg = p.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	color := p.text
	if selected {
		color = p.accent
	}
	b.Children(func() {
		ui.Icon(c, icon).Size(17, 17).TextColor(color)
		ui.Text(c, label).FontSize(13.5).FontWeight(500).TextColor(p.text)
	})
		if b.Clicked() {
			a.open(r)
		}
	}

func (a *app) localNavGroup(c *ui.Context) {
	p := a.pal
	selectedParent := a.route.page == pageLocal && !a.nowPlaying

	b := ui.ButtonBase(c.Key("nav-local-music")).Height(34).PaddingX(10).Gap(10).Radius(8).
		AlignItems(ui.Center).Justify(ui.Start).FocusRing(false).Role(ui.RoleDisclosure).Label("本地音乐")

	bg := ui.Transparent
	if selectedParent {
		bg = p.press
	} else if b.Hovered() {
		bg = p.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	color := p.text
	if selectedParent {
		color = p.accent
	}

	b.Children(func() {
		ui.Icon(c, iconFolder).Size(17, 17).TextColor(color)
		ui.Text(c, "本地音乐").FontSize(13.5).FontWeight(600).TextColor(p.text).Grow(1)
		arrow := iconDown
		if a.localCollapsed {
			arrow = iconForward
		}
		ui.Icon(c, arrow).Size(12, 12).TextColor(p.faint)
	})

	if b.Clicked() {
		if a.route.page != pageLocal {
			a.open(route{page: pageLocal})
		} else {
			a.localCollapsed = !a.localCollapsed
		}
	}

	if !a.localCollapsed {
		type subItem struct {
			tab   int
			label string
			icon  *ui.SVG
		}
		subItems := []subItem{
			{0, "全部单曲", iconNote},
			{4, "最新添加", iconDaily},
			{2, "按歌手", iconUser},
			{1, "按专辑", iconNote},
			{3, "按文件夹", iconFolder},
		}

		for _, item := range subItems {
			subKey := fmt.Sprintf("nav-local-tab-%d", item.tab)
			subSelected := selectedParent && a.localTab == item.tab && a.localFilterType == ""
			sub := ui.ButtonBase(c.Key(subKey)).Height(28).Padding(0, 10, 0, 26).Gap(8).
				Radius(7).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false).Label(item.label)

			subBg := ui.Transparent
			if subSelected {
				subBg = p.press
			} else if sub.Hovered() {
				subBg = p.hover
			}
			sub.Background(sub.AnimateColor("bg", subBg, ui.Animation{Duration: 120 * time.Millisecond}))
			subTextColor := p.muted
			subIconColor := p.faint
			if subSelected {
				subTextColor = p.text
				subIconColor = p.accent
			}

			sub.Children(func() {
				ui.Icon(c, item.icon).Size(13, 13).TextColor(subIconColor)
				ui.Text(c, item.label).FontSize(12.5).FontWeight(500).TextColor(subTextColor)
			})

			if sub.Clicked() {
				a.localTab = item.tab
				a.localFilterType = ""
				a.localFilterName = ""
				a.localFilterSongs = nil
				if a.route.page != pageLocal {
					a.open(route{page: pageLocal})
				}
			}
		}
	}
}

func (a *app) sidebarNowPlaying(c *ui.Context, tr *netease.Track) {
	p := a.pal
	card := ui.ButtonBase(c.Key("np-card")).Gap(10).Padding(8).Radius(12).AlignItems(ui.Center).FocusRing(false).Label("正在播放")
	bg := p.hover
	if card.Hovered() {
		bg = p.press
	}
	card.Background(card.AnimateColor("bg", bg, ui.Animation{Duration: 120 * time.Millisecond}))
	card.Children(func() {
		a.cover(c, tr.Album.PicURL, 96, 40, 6).Shadow(0, 2, 6, 0, p.cardShade)
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, tr.Name).FontSize(12.5).FontWeight(600).SingleLine()
			a.artistLinks(c, tr, p.muted, p.text).FontSize(11.5).SingleLine()
		})
	})
	if card.Clicked() {
		a.nowPlaying = true
	}
}

// playerBarHeight is the height of the player along the foot of the pages.
const playerBarHeight = 84

// pageGap is the room between the bar over a page and its content.
const (
	pageGap        = 12
	musicBarHeight = float32(48)
)

func (a *app) isMusicPage() bool {
	switch a.route.page {
	case pageHome, pageDaily, pageCharts, pagePlaylist, pageSearch, pageFM, pageArtist, pageAlbum, pageUser, pagePodcast, pagePodcasts, pageLocal:
		return true
	default:
		return false
	}
}

// insets returns the room the bars over a page take at its top and its
// bottom: pages pad their content by it, and scroll under the bars.
func (a *app) insets(c *ui.Context) (top, bottom float32) {
	top = topBar(c)
	if runtime.GOOS == "darwin" {
		// The edge reaches a little below the traffic lights' strip.
		top += topEdgePad
	}
	if a.isMusicPage() {
		top += musicBarHeight
	}
	if a.hidePlayerBar() {
		return top, 0
	}
	return top, playerBarHeight
}

// hidePlayerBar reports whether the bottom music player bar should be suppressed,
// e.g., when the user is inside the Short Drama theater or playing video content,
// preventing visual clutter and confusion between music and video controls.
func (a *app) hidePlayerBar() bool {
	if a.route.page == pageDramas && a.activeDramaTab == 1 {
		return true
	}
	if a.route.page == pageShortVideos {
		return true
	}
	if a.dramaPlaying || a.svPlaying {
		return true
	}
	return false
}

// handleSpace toggles play/pause for the active media context (drama video in theater mode, or music).
func (a *app) handleSpace() {
	if a.route.page == pageDramas && a.activeDramaTab == 1 {
		a.toggleDramaPlay()
		return
	}
	if a.route.page == pageShortVideos && a.svPlaying {
		return
	}
	a.pl.TogglePlay()
}

// topEdgePad is the room the edge over a page leaves below the strip, on
// macOS.
const topEdgePad = 6

// buttonsHeight is the height of the window buttons off macOS, which fill
// the edge over a page.
const buttonsHeight = topBarHeight + topEdgePad

// topBar returns the height of the strip along the top of the window, as
// tall as the window buttons and the room above and below them, which
// what sits in it centers on.
func topBar(c *ui.Context) float32 {
	if runtime.GOOS != "darwin" {
		// The whole edge, which the buttons fill: what sits in the strip
		// centers on their glyphs.
		return buttonsHeight
	}
	return max(c.TitleBar().Height, topBarHeight)
}

// topBarHeight is the least height of the strip along the top.
const topBarHeight = 40

// startMark marks where a page's content starts, first in what scrolls:
// once it has moved up, content is under the bar at the top.
func (a *app) startMark(c *ui.Context) {
	b := ui.Box(c.Key("start-mark")).Height(0).Bounds()
	if b.W > 0 {
		top, _ := a.insets(c)
		a.underTop = b.Y < top+pageGap-0.5
	}
}

// endMark marks where a page's content ends, last in what scrolls: while it
// is below the top of the player, content is under the player.
func (a *app) endMark(c *ui.Context) {
	b := ui.Box(c.Key("end-mark")).Height(0).Bounds()
	if b.W > 0 {
		_, h := c.Size()
		_, bottom := a.insets(c)
		a.underBottom = b.Y > h-bottom+0.5
	}
}

// edgeShown returns how much of a scroll edge shows over el: fully while
// content is under its bar, and none at rest, as AppKit shows its edges.
func edgeShown(el ui.Element, under bool) float32 {
	target := float32(0)
	if under {
		target = 1
	}
	return el.AnimateTo("shown", target, ui.Animation{Duration: 180 * time.Millisecond})
}

// pageTop is the title bar strip along the top of the window, frosting what scrolls under
// it and providing the window dragging surface.
func (a *app) pageTop(c *ui.Context) {
	top := topBar(c)
	p := a.pal
	edge := ui.Box(c.Key("top-edge")).Absolute().Top(0).Left(0).Right(0).Height(top).DragWindow().
		Material(glass.ScrollEdge{Hard: true, Background: p.bg})
	edge.Opacity(edgeShown(edge, a.underTop))

	if len(a.history) > 0 || len(a.forward) > 0 {
		ui.Row(c.Key("top-nav-btns")).Absolute().Top(0).Left(20).Height(top).AlignItems(ui.Center).Gap(4).Children(func() {
			back := iconButton(c.Key("back"), iconBack, 18, a.pal.muted, "返回").Disabled(len(a.history) == 0)
			if len(a.history) == 0 {
				back.Opacity(0.35)
			}
			if back.Clicked() {
				a.back()
			}
			if len(a.forward) > 0 && iconButton(c.Key("forward"), iconForward, 18, a.pal.muted, "前进").Clicked() {
				a.goForward()
			}
		})
	}
}

// musicSearchBar is the Alger-style full-width adaptive search bar positioned below the title bar,
// only displayed within the music world section.
func (a *app) musicSearchBar(c *ui.Context) {
	top := topBar(c)
	p := a.pal
	focused := c.Focused(a.topSearchBox)
	searchBg := p.card
	borderColor := p.hairline
	if focused {
		searchBg = p.bg
		borderColor = p.accent
	}

	edge := ui.Box(c.Key("music-search-edge")).Absolute().Top(top).Left(0).Right(0).Height(musicBarHeight).
		Material(glass.ScrollEdge{Hard: true, Background: p.bg})
	edge.Opacity(edgeShown(edge, a.underTop))

	ui.Row(c.Key("music-search-bar-row")).Absolute().Top(top + 6).Left(20).Right(20).Height(36).
		AlignItems(ui.Center).Gap(10).Children(func() {
			// Search Input Wrap with Grow(1) to adapt smoothly to full window width (Alger style)
			searchWrap := ui.Row(c.Key("music-search-wrap")).Grow(1).Height(36).Padding(0, 14, 0, 14).
				Gap(10).Radius(999).Background(searchBg).Border(1.5, borderColor).AlignItems(ui.Center)
			if focused {
				searchWrap.Shadow(0, 0, 0, 3, p.accent.Alpha(0.2))
			}
			searchWrap.Children(func() {
				iconCol := p.muted
				if focused {
					iconCol = p.accent
				}
				searchIconBtn := iconButton(c.Key("music-search-icon"), iconSearch, 15, iconCol, "搜索").Size(22, 22)
				if searchIconBtn.Clicked() {
					val := strings.TrimSpace(a.query)
					if val == "" && len(a.hotSearches) > 0 && a.hotSearches[0].SearchWord != "" {
						val = a.hotSearches[0].SearchWord
						a.query = val
					}
					a.search(val)
				}

				placeholder := "搜索音乐、歌手、歌单"
				if len(a.hotSearches) > 0 && a.hotSearches[0].SearchWord != "" {
					placeholder = a.hotSearches[0].SearchWord
				}

					in := ui.TextInputBase(c.Key("top-search-input"), &a.query).
						Placeholder(placeholder).
						FontSize(13.5).Label("搜索").Grow(1).MinWidth(0)
				in.Bind(&a.topSearchBox)
				if in.Submitted() {
					val := strings.TrimSpace(a.query)
					if val == "" && placeholder != "搜索音乐、歌手、歌单" {
						val = placeholder
						a.query = val
					}
					a.search(val)
				}

				if a.query != "" {
					if iconButton(c.Key("music-search-clear"), iconX, 11, p.muted, "清空").Size(18, 18).Clicked() {
						a.query = ""
						a.search("")
					}
				}
			})
		})
}

func (a *app) content(c *ui.Context) {
	// A new page starts at its top. Not as it is opened: a control in a
	// list, as a page's tabs in its header, opens it as the list builds.
	if a.scrolled != a.route {
		a.trackList, a.pageScroll, a.scrolled = ui.ListState{}, ui.ScrollState{}, a.route
	}
	// Pages mark where their content is, as they scroll.
	a.underTop, a.underBottom = false, false
	switch a.route.page {
	case pageHome:
		a.homePage(c)
	case pageDaily:
		a.dailyPage(c)
	case pageCharts:
		a.chartsPage(c)
	case pagePlaylist:
		a.playlistPage(c)
	case pageSearch:
		a.searchPage(c)
	case pageFM:
		a.fmPage(c)
	case pageArtist:
		a.artistPage(c)
	case pageAlbum:
		a.albumPage(c)
	case pageUser:
		a.userPage(c)
		case pagePodcast:
			a.podcastPage(c)
	case pagePodcasts:
		a.podcastsPage(c)
	case pageLocal:
		a.localPage(c)
	// 视界中心 (Video & Live)
	case pageDramas:
		a.dramasPage(c)
	case pageShortVideos:
		a.shortVideosPage(c)
	case pageLive:
		a.livePage(c)
	case pageVoiceRoom:
		a.voiceRoomPage(c)
	case pageCinema:
		a.cinemaPage(c)
	// 边缘节点 (Cloud Node)
	case pageNodeBoard:
		a.nodeBoardPage(c)
	case pageNodeStorage:
		a.nodeStoragePage(c)
	case pageNodeEarnings:
		a.nodeEarningsPage(c)
	}
}

// centered shows a message, with a retry button if retry is set: what a
// page shows in place of what failed to load.
func (a *app) centered(c *ui.Context, err string, retry func()) {
	top, bottom := a.insets(c)
	ui.Column(c).Grow(1).Padding(top, 0, bottom, 0).Center().Gap(12).Children(func() {
		ui.Text(c, err).TextColor(a.pal.muted).FontSize(13)
		if retry != nil && ui.Button(c, "重试").Clicked() {
			retry()
		}
	})
}

// Loading what the pages show.

func (a *app) loadHome() {
	a.homeLoading, a.homeErr = true, ""
	signedIn := a.profile != nil
	at := time.Now()
	go func() {
		featured, err1 := netease.FeaturedPlaylists(30)
		charts, err2 := netease.Charts()
		daily, _ := netease.DailySongs()
		var rec []netease.Playlist
		if signedIn {
			rec, _ = netease.RecommendedPlaylists()
		}
		a.post(func() {
			a.homeLoading = false
			if err1 != nil && err2 != nil {
				// Loading the new day's again, what shows stays, to try
				// once more.
					if len(a.featured) == 0 && len(a.charts) == 0 {
						a.homeErr = "无法连接在线音乐服务，请检查网络"
					}
				return
			}
			a.featured, a.charts, a.daily, a.recPlaylists = featured, charts, daily, rec
			a.homeAt = at
		})
	}()
}

// freshenHome loads the home page again once NetEase has picked the day's
// songs anew since it last did: the app may stay open overnight.
func (a *app) freshenHome() {
	if !a.homeLoading && !a.homeAt.IsZero() && dailyRenewed(a.homeAt, time.Now()) {
		a.loadHome()
	}
}

// beijing is NetEase's time, without daylight saving.
var beijing = time.FixedZone("CST", 8*60*60)

// dailyRenewed reports whether NetEase has picked new daily songs between
// loaded and now. It does at 6:00, Beijing time.
func dailyRenewed(loaded, now time.Time) bool {
	b := now.In(beijing)
	six := time.Date(b.Year(), b.Month(), b.Day(), 6, 0, 0, 0, beijing)
	if b.Before(six) {
		six = six.AddDate(0, 0, -1)
	}
	return loaded.Before(six)
}

func (a *app) loadPlaylist(id int64, thenPlay bool) {
	if pl := a.playlists[id]; pl != nil {
		if thenPlay {
			a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
		}
		return
	}
	if a.loadingPl[id] {
		return
	}
	a.loadingPl[id] = true
	delete(a.plErr, id)
	go func() {
		pl, err := netease.PlaylistDetail(id)
		a.post(func() {
			delete(a.loadingPl, id)
			if err != nil {
				a.plErr[id] = "歌单加载失败"
				return
			}
			a.playlists[id] = pl
			if thenPlay {
				a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
			}
		})
	}()
}

func firstPlayable(tracks []netease.Track) int {
	for i := range tracks {
		if tracks[i].Playable() {
			return i
		}
	}
	return 0
}
