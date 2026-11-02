package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/player"
)

const pagePad = 40

// columns returns how many cards of at least minWidth DIPs fit across the page.
// Standard desktop layouts follow the Tencent QQ Music 6-column specification (minWidth >= 160 capped at 6),
// ensuring generous 220-250px cover dimensions without over-stretching or shrinking.
func (a *app) columns(c *ui.Context, minWidth float32) int {
	w, _ := c.Size()
	inner := w - sidebarWidth - 2*pagePad
	cols := int((inner + 20) / (minWidth + 20))
	if minWidth >= 160 {
		return max(2, min(6, cols))
	}
	return max(2, min(8, cols))
}

func filterPlaylists(list []netease.Playlist, cat string) []netease.Playlist {
	if cat == "" || cat == "全部" || cat == "为你推荐" || cat == "官方精选" {
		return list
	}
	var out []netease.Playlist
	for _, pl := range list {
		if strings.Contains(pl.Name, cat) || strings.Contains(pl.Description, cat) {
			out = append(out, pl)
		}
	}
	if len(out) == 0 {
		return list
	}
	return out
}

func (a *app) heading(c *ui.Context, title, sub string) {
	ui.Column(c).Gap(4).Margin(0, 0, 18, 0).Children(func() {
		if sub != "" {
			ui.Text(c, sub).FontSize(12).FontWeight(600).TextColor(a.pal.muted).LetterSpacing(0.4)
		}
		ui.Text(c, title).FontSize(30).FontWeight(700).LetterSpacing(-0.6)
	})
}

func (a *app) section(c *ui.Context, title string) ui.Element {
	return ui.Text(c, title).FontSize(19).FontWeight(700).LetterSpacing(-0.3).Margin(34, 0, 14, 0)
}

// sectionHeader renders a Tencent QQ Music style section header with title, quick-filter category pills, and "更多 >" link.
func (a *app) sectionHeader(c *ui.Context, title string, pills []string, activePill string, onSelect func(string), onMore func()) {
	p := a.pal
	ui.Row(c.Key("sec-hdr-" + title)).Margin(34, 0, 16, 0).AlignItems(ui.Center).Children(func() {
		// Section Title
		ui.Text(c, title).FontSize(20).FontWeight(700).TextColor(p.text).LetterSpacing(-0.4)

		// Category Pills
		if len(pills) > 0 {
			ui.Row(c).Margin(0, 0, 0, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				for _, pill := range pills {
					cat := pill
					selected := cat == activePill || (activePill == "" && (cat == "为你推荐" || cat == "全部"))
					btn := ui.ButtonBase(c.Key("pill-" + title + "-" + cat)).Height(28).PaddingX(12).Radius(999).
						AlignItems(ui.Center).FocusRing(false).Label(cat)
					if selected {
						btn.Background(p.accent).TextColor(p.onAccent)
					} else {
						btn.Background(p.hover).TextColor(p.muted)
						if btn.Hovered() {
							btn.Background(p.press).TextColor(p.text)
						}
					}
					btn.Children(func() {
						ui.Text(c, cat).FontSize(12).FontWeight(500)
					})
					if btn.Clicked() && onSelect != nil {
						onSelect(cat)
					}
				}
			})
		}

		ui.Box(c).Grow(1)

		// Right "更多 >" Link
		if onMore != nil {
			moreBtn := ui.ButtonBase(c.Key("more-" + title)).Height(26).PaddingX(8).Radius(6).
				AlignItems(ui.Center).FocusRing(false).Label("更多")
			if moreBtn.Hovered() {
				moreBtn.Background(p.hover)
			}
			moreBtn.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
					ui.Text(c, "更多").FontSize(12).TextColor(p.muted)
					ui.Icon(c, iconForward).Size(12, 12).TextColor(p.muted)
				})
			})
			if moreBtn.Clicked() {
				onMore()
			}
		}
	})
}

// homeSubNav provides Tencent-style category tabs at the top of the music discovery page.
func (a *app) homeSubNav(c *ui.Context, now time.Time) {
	p := a.pal
	ui.Row(c.Key("home-sub-nav")).Height(40).Margin(0, 0, 16, 0).AlignItems(ui.Center).Children(func() {
		tabs := []struct {
			name  string
			route route
		}{
			{"精选", route{page: pageHome}},
			{"排行榜", route{page: pageCharts}},
			{"电台主播", route{page: pagePodcasts}},
			{"私人 FM", route{page: pageFM}},
			{"本地音乐", route{page: pageLocal}},
		}

		ui.Row(c).Gap(20).AlignItems(ui.Center).Children(func() {
			for _, t := range tabs {
				tab := t
				active := a.route.page == tab.route.page
				btn := ui.ButtonBase(c.Key("subnav-" + tab.name)).PaddingY(4).PaddingX(2).FocusRing(false).Label(tab.name)
				btn.Children(func() {
					ui.Column(c).AlignItems(ui.Center).Gap(4).Children(func() {
						col := p.muted
						weight := 500
						if active {
							col = p.text
							weight = 700
						} else if btn.Hovered() {
							col = p.text
						}
						ui.Text(c, tab.name).FontSize(16).FontWeight(weight).TextColor(col)
						if active {
							ui.Box(c).Height(3).Width(20).Radius(2).Background(p.accent)
						} else {
							ui.Box(c).Height(3).Width(20)
						}
					})
				})
				if btn.Clicked() && !active {
					a.open(tab.route)
				}
			}
		})

		ui.Box(c).Grow(1)

		// Right: Date & Account
		ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Text(c, fmt.Sprintf("%d月%d日 %s", now.Month(), now.Day(), weekday(now))).
				FontSize(12).FontWeight(500).TextColor(p.muted)

			if a.profile == nil {
				loginBtn := pillButton(c.Key("home-login-btn"), iconUser, "登录网易云", p.hover, p.text).
					Border(0.5, p.hairline)
				if loginBtn.Clicked() {
					a.openLogin()
				}
			} else {
				prof := a.profile
				chip := ui.ButtonBase(c.Key("home-user-chip")).Height(30).PaddingX(8).Gap(6).Radius(999).
					Background(p.hover).Border(0.5, p.hairline).AlignItems(ui.Center).FocusRing(false).Label(prof.Nickname)
				if chip.Hovered() || a.accountOpen {
					chip.Background(p.press)
				}
				chip.Children(func() {
					a.avatar(c, prof.AvatarURL, 20, p.press)
					ui.Text(c, prof.Nickname).FontSize(12).FontWeight(600).TextColor(p.text)
					ui.Icon(c, iconDown).Size(10, 10).TextColor(p.muted)
				})
				if chip.Clicked() {
					a.accountOpen = !a.accountOpen
				}
				a.accountMenu(c, chip)
			}
		})
	})
}

func (a *app) homeHeading(c *ui.Context, now time.Time) {
	ui.Row(c).AlignItems(ui.End).Margin(0, 0, 16, 0).Children(func() {
		// Left: Greeting
		ui.Column(c).Grow(1).Gap(4).Children(func() {
			hello := greeting(now)
			if a.profile != nil {
				hello += "，" + a.profile.Nickname
			}
			ui.Text(c, hello).FontSize(26).FontWeight(700).LetterSpacing(-0.5)
		})
	})
}

func (a *app) homePage(c *ui.Context) {
	loading := len(a.featured) == 0 && len(a.charts) == 0
	if loading && !a.homeLoading {
		a.centered(c, a.homeErr, a.loadHome)
		return
	}
	now := c.Now()
	top, bottom := a.insets(c)
	ui.Scroll(c.Key("home")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		a.homeSubNav(c, now)
		if loading {
			a.homeSkeleton(c, a.pageShimmer(c))
			return
		}
		a.dailyCard(c, now)

		recCols := a.columns(c, 210)

		// 1. 今日为你推荐
		if len(a.recPlaylists) > 0 {
			recPills := []string{"为你推荐", "官方精选", "流行", "轻音乐", "经典华语", "摇滚电音"}
			a.sectionHeader(c, "今日为你推荐", recPills, a.recCategory, func(cat string) {
				a.recCategory = cat
			}, func() {
				a.open(route{page: pagePlaylist, id: a.recPlaylists[0].ID})
			})

			displayRec := filterPlaylists(a.recPlaylists, a.recCategory)
			maxCount := min(len(displayRec), recCols*2)
			ui.Grid(c).Columns(recCols).GapX(20).GapY(28).Children(func() {
				for i := range displayRec[:maxCount] {
					a.playlistCard(c.Key(displayRec[i].ID), &displayRec[i])
				}
			})
		}

		// 2. 官方榜单
		a.sectionHeader(c, "官方权威榜单", nil, "", nil, func() {
			a.open(route{page: pageCharts})
		})
		official := a.charts[:min(4, len(a.charts))]
		chartCols := min(4, max(2, a.columns(c, 260)))
		ui.Grid(c).Columns(chartCols).Gap(20).Children(func() {
			for i := range official {
				a.chartCard(c.Key(official[i].ID), &official[i])
			}
		})

		// 3. 推荐歌单
		featPills := []string{"全部", "华语", "流行", "欧美", "二次元", "民谣", "电子"}
		a.sectionHeader(c, "推荐歌单", featPills, a.featCategory, func(cat string) {
			a.featCategory = cat
		}, func() {
			if len(a.featured) > 0 {
				a.open(route{page: pagePlaylist, id: a.featured[0].ID})
			}
		})

		displayFeat := filterPlaylists(a.featured, a.featCategory)
		maxFeat := min(len(displayFeat), recCols*3)
		ui.Grid(c).Columns(recCols).GapX(20).GapY(28).Children(func() {
			for i := range displayFeat[:maxFeat] {
				a.playlistCard(c.Key(displayFeat[i].ID), &displayFeat[i])
			}
		})
	})
}

func weekday(t time.Time) string {
	return [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[t.Weekday()]
}

func (a *app) chartsPage(c *ui.Context) {
	loading := len(a.charts) == 0
	if loading && !a.homeLoading {
		a.centered(c, a.homeErr, a.loadHome)
		return
	}
	top, bottom := a.insets(c)
	ui.Scroll(c.Key("charts")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad, bottom+48, pagePad).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		a.heading(c, "排行榜", "每日更新")
			if loading {
				s := a.pageShimmer(c)
				a.chartCardsSkeleton(c, s)
				a.section(c, "全部榜单")
				a.playlistCardsSkeleton(c, s, a.columns(c, 210)*2, 210, true)
				return
			}
			official := a.charts[:min(4, len(a.charts))]
			cols := min(4, max(2, a.columns(c, 260)))
			ui.Grid(c).Columns(cols).Gap(20).Children(func() {
				for i := range official {
					a.chartCard(c.Key(official[i].ID), &official[i])
				}
			})
			if len(a.charts) > 4 {
				a.section(c, "全部榜单")
				rest := a.charts[4:]
				ui.Grid(c).Columns(a.columns(c, 210)).GapX(20).GapY(28).Children(func() {
					for i := range rest {
						a.playlistCard(c.Key(rest[i].ID), &rest[i])
					}
				})
			}
	})
}

// playlistCard is a square cover with the playlist's name below: it lifts
// under the pointer, where a play button shows on it.
func (a *app) playlistCard(c *ui.Context, pl *netease.Playlist) {
	if a.coverCard(c, pl.Name, pl.Cover(), pl.PlayCount, pl.UpdateFreq, func() { a.loadPlaylist(pl.ID, true) }) {
		a.open(route{page: pagePlaylist, id: pl.ID})
	}
}

// coverCard is a playlist's card, or a podcast's: its cover, with how many
// times it was played on it, over its name and a line under that. The
// button on the cover calls play; it reports whether the rest was clicked.
func (a *app) coverCard(c *ui.Context, name, cover string, playCount float64, sub string, play func()) bool {
	p := a.pal
	// From the top: a grid stretches the cards of a row as tall as the
	// tallest, whose name or line under it takes two lines.
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label(name)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(10)
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -3}
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(lift, ui.Snappy)
		art.Children(func() {
			a.coverFill(c, cover, 400, 10)
			if playCount > 0 {
				ui.Row(c).Absolute().Top(8).Right(8).Padding(2, 7).Radius(999).Gap(3).AlignItems(ui.Center).
					Background(ui.RGBA(0, 0, 0, 0.35)).TextColor(ui.RGB(255, 255, 255)).Children(func() {
					ui.Icon(c, iconHeadset).Size(10, 10)
					ui.Text(c, plays(playCount)).FontSize(10.5).FontWeight(600)
				})
			}
			a.coverPlayButton(c, hovered, play)
		})
			titleCol := p.text
			if hovered {
				titleCol = p.accent
			}
			ui.Text(c, name).FontSize(13.5).FontWeight(500).TextColor(titleCol).MaxLines(2).LineHeight(1.35)
			if sub != "" {
				ui.Text(c, sub).FontSize(12).TextColor(p.muted).SingleLine().Margin(-6, 0, 0, 0)
			}
	})
	return card.Clicked()
}

// coverFill fills its parent with a cover, its corners rounded by radius.
// The cover rounds itself rather than its parent clipping it: a clip moved
// by a transform, as a card lifting under the pointer, is drawn on the CPU.
func (a *app) coverFill(c *ui.Context, url string, px int, radius float32) {
	bmp := bitmap(a.art.get(netease.Sized(url, px), false))
	box := ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Radius(radius)
	in := artIn(box, bmp)
	if in < 1 {
		box.Background(a.pal.hover)
	}
	box.Children(func() { artImage(c, bmp, radius, in) })
}

// coverPlayButton plays a playlist or a podcast from its card, without
// opening it.
func (a *app) coverPlayButton(c *ui.Context, shown bool, play func()) {
	b := ui.ButtonBase(c.Key("play")).Absolute().Bottom(10).Right(10).Size(38, 38).Radius(999).Center().
		Background(a.pal.accent).TextColor(ui.RGB(255, 255, 255)).Label("播放").FocusRing(false).
		Shadow(0, 4, 12, 0, ui.RGBA(0, 0, 0, 0.3))
	pose := ui.Pose{Fade: 1, Scale: 0.8, Y: 6}
	if shown || b.Hovered() {
		pose = ui.Pose{Scale: 1}
		if b.Hovered() {
			pose.Scale = 1.08
		}
	}
	b.AnimatePose(pose, ui.Snappy)
	b.Children(func() { ui.Icon(c, iconPlay).Size(15, 15).Margin(0, 0, 0, 2) })
	if b.Clicked() {
		play()
	}
}

// chartCard shows a chart with its top three songs beside its cover.
func (a *app) chartCard(c *ui.Context, pl *netease.Playlist) {
	p := a.pal
	card := ui.ButtonBase(c).Gap(14).Padding(12).Radius(14).AlignItems(ui.Center).FocusRing(false).Label(pl.Name)
	hovered := card.Hovered()
	bg := p.hover
	if hovered {
		bg = p.press
	}
	card.Background(card.AnimateColor("bg", bg, ui.Animation{Duration: 140 * time.Millisecond}))
	card.Children(func() {
		ui.Box(c).Size(84, 84).Radius(9).Shadow(0, 3, 10, -2, p.cardShade).Children(func() {
			a.coverFill(c, pl.Cover(), 200, 9)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(5).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Baseline).Children(func() {
				ui.Text(c, pl.Name).FontSize(14).FontWeight(700).SingleLine()
				ui.Text(c, pl.UpdateFreq).FontSize(11).TextColor(p.faint).SingleLine()
			})
			for i, line := range pl.Preview {
				ui.Row(c).Gap(7).Children(func() {
					ui.Textf(c, "%d", i+1).FontSize(12).FontWeight(700).Width(10).FontFeatures("tnum").TextColor(p.muted)
					ui.RichText(c,
						ui.Span{Text: line.First, Weight: 500},
						ui.Span{Text: "  " + line.Second, Color: p.muted},
					).FontSize(12).SingleLine().Shrink(1).MinWidth(0)
				})
			}
		})
	})
	if card.Clicked() {
		a.open(route{page: pagePlaylist, id: pl.ID})
	}
}

func (a *app) playlistPage(c *ui.Context) {
	id := a.route.id
	pl := a.playlists[id]
	if pl == nil {
		if !a.loadingPl[id] {
			a.centered(c, a.plErr[id], func() { a.loadPlaylist(id, false) })
			return
		}
		s := a.pageShimmer(c)
		a.trackSkeleton(c, s, true, func(c *ui.Context) { a.playlistHeaderSkeleton(c, s) })
		return
	}
	a.trackRows(c, pl.Tracks, true, func(c *ui.Context) { a.playlistHeader(c, pl) })
}

func (a *app) playlistHeader(c *ui.Context, pl *netease.Playlist) {
	p := a.pal
	ui.Row(c).Gap(28).AlignItems(ui.End).Padding(0, 0, 26, 0).Children(func() {
		ui.Box(c).Size(212, 212).Radius(14).Shadow(0, 18, 40, -10, p.cardShade).Children(func() {
			a.coverFill(c, pl.Cover(), 600, 14)
		})
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(8).Children(func() {
			kind := "歌单"
			if pl.UpdateFreq != "" {
				kind = "榜单 · " + pl.UpdateFreq
			}
			ui.Text(c, kind).FontSize(12).FontWeight(600).TextColor(p.accent)
			ui.Text(c, pl.Name).FontSize(30).FontWeight(800).LetterSpacing(-0.6).MaxLines(2).LineHeight(1.2)
			var meta []string
			if pl.Creator.Nickname != "" {
				meta = append(meta, pl.Creator.Nickname)
			}
			meta = append(meta, fmt.Sprintf("%d 首", len(pl.Tracks)))
			if pl.PlayCount > 0 {
				meta = append(meta, plays(pl.PlayCount)+" 次播放")
			}
			ui.Text(c, strings.Join(meta, " · ")).FontSize(13).TextColor(p.muted)
			if d := strings.TrimSpace(pl.Description); d != "" {
				ui.Text(c, strings.ReplaceAll(d, "\n", " ")).FontSize(12.5).TextColor(p.muted).MaxLines(2).LineHeight(1.45).Margin(2, 0, 0, 0)
			}
			ui.Row(c).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				if pillButton(c.Key("play"), iconPlay, "播放", p.accent, p.onAccent).Clicked() {
					if a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
				}
				if pillButton(c.Key("shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
					if !a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(pl.Tracks, firstPlayable(pl.Tracks))
				}
				if mine := a.editable(pl.ID); mine != nil {
					more := iconButton(c.Key("more"), iconMore, 18, p.muted, "更多")
					more.Menu(func(m *ui.Menu) {
						if m.Item("重命名…").Chosen() {
							a.startRename(mine)
						}
						m.Separator()
						if m.Item("删除歌单…").Chosen() {
							a.confirmDelete(mine)
						}
					})
				}
				switch {
				case pl.SpecialType != 5:
				case a.pl.Radio() == heartRadio:
					if pillButton(c.Key("heart"), iconX, "退出心动模式", p.hover.Over(p.bg), p.text).Clicked() {
						a.exitRadio()
					}
				default:
					label := "心动模式"
					if a.radioStarting == heartRadio {
						label = "正在开启…"
					}
					if pillButton(c.Key("heart"), iconHeartFill, label, p.hover.Over(p.bg), p.text).Clicked() {
						a.startHeartMode()
					}
				}
			})
		})
	})
}

// trackRows lists tracks under a header, which scrolls with them; a click
// plays the list from the track clicked.
func (a *app) trackRows(c *ui.Context, tracks []netease.Track, covers bool, header func(c *ui.Context), footer ...func(c *ui.Context)) {
	p := a.pal
	cur := a.pl.Current()
	// A song's menu can take it out of the playlist it is listed in.
	var pid int64
	if a.route.page == pagePlaylist {
		pid = a.route.id
	}
	w, _ := c.Size()
	// An album's page has no column of the album its songs are on.
	wide := w-sidebarWidth > 820 && a.route.page != pageAlbum
	top, bottom := a.insets(c)
	// Rows out of view are not built, their marks with them: a list
	// scrolled away from its start or end has content under the bars.
	a.underTop, a.underBottom = true, true
	rows := len(tracks) + 1 + len(footer)
	ui.List(c.Key(a.route), &a.trackList, rows, func(i int) {
		if i == 0 {
			ui.Column(c).Children(func() {
				a.startMark(c)
				header(c)
			})
			return
		}
		if i == rows-1 {
			defer a.endMark(c)
		}
		if i > len(tracks) {
			footer[i-len(tracks)-1](c)
			return
		}
			i--
			tr := &tracks[i]
			playing := cur != nil && cur.ID == tr.ID
			row := ui.ButtonBase(c.Key(tr.ID)).Height(56).Padding(0, 12, 0, 12).Gap(14).Radius(12).AlignItems(ui.Center).FocusRing(false).Label(tr.Name)
			if row.RightClicked() {
				a.songMenuTrack = tr
				a.songMenuOpen = true
			}
			hovered := row.Hovered()
			bg := ui.Transparent
			switch {
			case playing:
				bg = p.accent.Alpha(0.09)
			case hovered:
				bg = p.hover
			}
			row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
			if !tr.Playable() {
				row.Opacity(0.42)
			}
			row.Children(func() {
				ui.Box(c).Width(26).Center().Children(func() {
					switch {
					case playing:
						a.nowPlayingBars(c)
					case hovered && tr.Playable():
						ui.Icon(c, iconPlay).Size(14, 14).TextColor(p.accent)
					default:
						ui.Textf(c, "%d", i+1).FontSize(13).FontFeatures("tnum").TextColor(p.faint)
					}
				})
				if covers {
					a.cover(c, tr.Album.PicURL, 120, 44, 10)
				}
				ui.Column(c).Grow(3).Basis(0).MinWidth(0).Gap(3).Children(func() {
					name := p.text
					if playing {
						name = p.accent
					}
					spans := []ui.Span{{Text: tr.Name, Weight: 600, Color: name}}
					if len(tr.Alias) > 0 {
						spans = append(spans, ui.Span{Text: "（" + tr.Alias[0] + "）", Color: p.faint})
					}
					ui.RichText(c, spans...).FontSize(14.5).SingleLine()
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						// Gold SQ 320K audio quality badge (Alger / Hi-Res style)
						ui.Box(c).Padding(1, 4, 1, 4).Radius(3).Background(ui.RGB(212, 168, 83).Alpha(0.15)).Border(0.8, ui.RGB(212, 168, 83).Alpha(0.6)).Children(func() {
							ui.Text(c, "SQ 320K").FontSize(8.5).FontWeight(700).TextColor(ui.RGB(226, 184, 102))
						})
						if !tr.Playable() {
							ui.Text(c, "VIP").FontSize(9).FontWeight(800).TextColor(p.accent).Padding(0, 4).Radius(3).Border(1, p.accent.Alpha(0.6))
						}
						a.artistLinks(c, tr, p.muted, p.text).FontSize(12.5).SingleLine().Shrink(1).MinWidth(0)
						if tr.Reason != "" {
							ui.Text(c, tr.Reason).FontSize(10.5).FontWeight(600).TextColor(p.accent).Padding(1, 6).Radius(999).Background(p.accent.Alpha(0.1)).SingleLine()
						}
					})
				})
				if wide {
					album := ui.RichText(c).Grow(2).Basis(0).MinWidth(0).FontSize(12.5).TextColor(p.muted).SingleLine()
					album.Children(func() {
						if tr.Album.ID == 0 {
							ui.Text(c, tr.Album.Name)
						} else {
							a.link(c.Key("album"), tr.Album.Name, p.muted, p.text, route{page: pageAlbum, id: tr.Album.ID})
						}
					})
				}
				// A liked song's heart shows; the others' as the pointer is on them.
				ui.Box(c).Size(32, 32).Center().Children(func() {
					if a.liked[tr.ID] || hovered {
						a.likeButton(c.Key("like"), tr, 15, p.muted)
					}
				})
				// Three dots "···" more button
				ui.Box(c).Size(32, 32).Center().Children(func() {
					if hovered || (a.songMenuOpen && a.songMenuTrack != nil && a.songMenuTrack.ID == tr.ID) {
						moreBtn := iconButton(c.Key("row-more"), iconMore, 16, p.muted, "更多")
						if moreBtn.Clicked() {
							a.songMenuTrack = tr
							a.songMenuOpen = !a.songMenuOpen
						}
					}
				})
				ui.Text(c, clock(tr.Length())).FontSize(12.5).FontFeatures("tnum").TextColor(p.faint).Width(44).TextAlign(ui.End)
			})
			if a.songMenuOpen && a.songMenuTrack != nil && a.songMenuTrack.ID == tr.ID {
				a.songApplePopover(c, row, tr, pid)
			}
			if row.Clicked() && tr.Playable() {
				a.pl.PlayQueue(tracks, i)
			}
	}).Grow(1).Padding(top+pageGap, pagePad-12, bottom+32, pagePad-12).ScrollbarInsets(top, 0, bottom, 0)
}

// nowPlayingBars dance in the row of the track playing.
func (a *app) nowPlayingBars(c *ui.Context) {
	color := a.pal.accent
	ui.Box(c).Size(16, 14).Draw(func(p *ui.Painter, r ui.Rect) {
		lv, live := a.pl.Levels()
		a.rowBars.step(p.Now(), lv, live)
		drawBars(p, r, &a.rowBars, 4, color, 2.5)
		if live || a.rowBars.group(0, player.Bands) > 0.01 {
			p.AnimationFrame()
		}
	})
}
