package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/vcp"
)

// dramasPage is the main view for 视界中心 -> 微短剧场.
func (a *app) dramasPage(c *ui.Context) {
	top, bottom := a.insets(c)

	if a.activeDramaTab == 1 {
		// 剧场播放模式：固定沉浸式影院舞台与右侧抽屉，不进行全页垂直滚动
		ui.Column(c.Key("dramas-theater-page")).Fill().Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).Children(func() {
			a.startMark(c)
			defer a.endMark(c)
			a.dramasHeaderBar(c)
			a.dramaTheater(c)
		})
		return
	}

	// 精选大厅与追剧历史：采用丝滑全页面滚动容器，容纳大剧目海报流
	ui.Scroll(c.Key("dramas-scroll-page")).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		a.dramasHeaderBar(c)
		switch a.activeDramaTab {
		case 2:
			a.dramasHistory(c)
		default:
			a.dramasHall(c)
		}
	})
}

// dramasHeaderBar renders the unified top navigation for 微短剧场.
func (a *app) dramasHeaderBar(c *ui.Context) {
	p := a.pal
	ui.Row(c.Key("dramas-header")).AlignItems(ui.Center).Gap(14).Margin(0, 0, 18, 0).Children(func() {
		// Title & Icon
		ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
			ui.Icon(c, iconClapperboard).Size(22, 22).TextColor(p.accent)
			ui.Text(c, "微短剧场").FontSize(20).FontWeight(700).TextColor(p.text)
		})

		// Sub Tabs: 精选大厅 / 剧场播放 / 追剧历史
		tabs := []struct {
			name string
			idx  int
		}{
			{"精选大厅", 0},
			{"剧场播放", 1},
			{"追剧历史", 2},
		}
		ui.Row(c.Key("dramas-tabs")).Gap(6).Children(func() {
			for _, tab := range tabs {
				tabIdx := tab.idx
				tabName := tab.name
				active := a.activeDramaTab == tabIdx
				btn := ui.ButtonBase(c.Key("tab-" + tabName)).Height(30).PaddingX(14).Radius(999).
					AlignItems(ui.Center).FocusRing(false).Label(tabName)
				if active {
					btn.Background(p.accent).TextColor(p.onAccent)
				} else {
					btn.Background(p.card).TextColor(p.muted).Border(1, p.hairline)
					if btn.Hovered() {
						btn.Background(p.hover).TextColor(p.text)
					}
				}
				btn.Children(func() {
					weight := 500
					if active {
						weight = 700
					}
					ui.Text(c, tabName).FontSize(12.5).FontWeight(weight)
				})
				if btn.Clicked() {
					a.activeDramaTab = tabIdx
					if tabIdx != 1 && a.dramaPlaying {
						a.stopVideo()
					}
					if tabIdx == 1 && a.pl != nil && a.pl.Playing() {
						a.pl.TogglePlay()
					}
				}
			}
		})

		ui.Spacer(c)

		// Real-time Search Box with Pill border
		searchField := ui.Row(c.Key("drama-search-box")).Height(32).PaddingX(12).Gap(8).Radius(16).
			AlignItems(ui.Center).Width(240).Background(p.card).Border(1, p.hairline)
		searchField.Children(func() {
			ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.faint)
			ui.TextInputBase(c.Key("drama-query"), &a.dramaQuery).
				Placeholder("搜索剧名、标签、主角...").FontSize(12.5).Label("搜索剧目").
				Grow(1).MinWidth(0)

			if a.dramaQuery != "" {
				if iconButton(c.Key("clear-dq"), iconX, 12, p.muted, "清除").Size(16, 16).Clicked() {
					a.dramaQuery = ""
				}
			}
		})
	})
}

// dramasHall displays the curated short drama catalog grid with multi-tier taxonomy from the station master.
func (a *app) dramasHall(c *ui.Context) {
	p := a.pal
	tax := vcp.GetDramaTaxonomy()
	if a.dramaType == "" {
		a.dramaType = "全部"
	}
	if a.dramaRanking == "" {
		a.dramaRanking = "推荐榜"
	}

	filtered := vcp.FilterDramasTaxonomy(a.dramaType, a.dramaRanking, a.dramaCategory, a.dramaQuery)

	ui.Column(c.Key("hall-view")).Gap(14).Children(func() {
		// Tier 1: Main Type Tabs (全部 | 漫剧 | 真人剧 | AI剧 | 演员) - 对齐站长端默认栏目
		ui.Row(c.Key("type-bar")).AlignItems(ui.Center).Gap(20).Padding(0, 4, 4, 4).Children(func() {
			for _, dt := range tax.Types {
				typeName := dt
				active := a.dramaType == typeName
				btn := ui.ButtonBase(c.Key("type-" + typeName)).Height(32).PaddingX(6).
					AlignItems(ui.Center).FocusRing(false)

				textColor := p.muted
				weight := 500
				size := float32(14)
				if active {
					textColor = p.text
					weight = 700
					size = 15.5
				} else if btn.Hovered() {
					textColor = p.text
				}
				btn.Children(func() {
					ui.Column(c).AlignItems(ui.Center).Gap(4).Children(func() {
						ui.Text(c, typeName).FontSize(size).FontWeight(weight).TextColor(textColor)
						if active {
							ui.Box(c).Width(24).Height(3).Radius(1.5).Background(p.accent)
						} else {
							ui.Box(c).Height(3)
						}
					})
				})
				if btn.Clicked() {
					a.dramaType = typeName
				}
			}
		})

		// Tier 2: Rankings & Badges Bar (推荐榜 | 热播榜 | 臻果榜 | 预约榜 | 新剧榜 | 追更榜 | 必看榜 | 收藏榜)
		ui.Row(c.Key("rankings-bar")).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
			for _, rk := range tax.Rankings {
				rkName := rk
				active := a.dramaRanking == rkName
				btn := ui.ButtonBase(c.Key("rk-" + rkName)).Height(28).PaddingX(12).Radius(14).
					AlignItems(ui.Center).FocusRing(false).Label(rkName)

				bg := p.card
				border := p.hairline
				fg := p.muted
				if active {
					bg = p.accent.Alpha(0.15)
					border = p.accent.Alpha(0.4)
					fg = p.accent
				} else if btn.Hovered() {
					bg = p.hover
					fg = p.text
				}
				btn.Background(bg).Border(1, border)
				btn.Children(func() {
					ui.Text(c, rkName).FontSize(12).FontWeight(map[bool]int{true: 700, false: 500}[active]).TextColor(fg)
				})
				if btn.Clicked() {
					a.dramaRanking = rkName
				}
			}

			ui.Spacer(c)
			countLabel := fmt.Sprintf("共 %d 部精选短剧", len(filtered))
			if a.dramaQuery != "" || (a.dramaCategory != "" && a.dramaCategory != "全部") || (a.dramaType != "" && a.dramaType != "全部") {
				countLabel = fmt.Sprintf("过滤呈现 %d 部短剧", len(filtered))
			}
			ui.Text(c, countLabel).FontSize(12).TextColor(p.faint)
		})

		// Tier 3: Genre Pills (全部 | 都市 | 爽剧逆袭 | 战神 | 古装 | 甜宠 | 悬疑 | 穿越 | 奇幻)
		ui.Row(c.Key("genre-pills")).AlignItems(ui.Center).Gap(6).Wrap().Children(func() {
			for _, cat := range tax.Genres {
				catName := cat
				isActive := (a.dramaCategory == "" && catName == "全部") || a.dramaCategory == catName
				btn := ui.ButtonBase(c.Key("cat-" + catName)).Height(26).PaddingX(11).Radius(999).
					AlignItems(ui.Center).FocusRing(false).Label(catName)
				if isActive {
					btn.Background(p.accent).TextColor(p.onAccent)
				} else {
					btn.Background(p.card).TextColor(p.muted).Border(1, p.hairline)
					if btn.Hovered() {
						btn.Background(p.hover).TextColor(p.text)
					}
				}
				btn.Children(func() {
					ui.Text(c, catName).FontSize(11.5).FontWeight(map[bool]int{true: 700, false: 500}[isActive])
				})
				if btn.Clicked() {
					if catName == "全部" {
						a.dramaCategory = ""
					} else {
						a.dramaCategory = catName
					}
				}
			}
		})

		// Hero Featured Banner (when on "全部" and no search query)
		if (a.dramaCategory == "" || a.dramaCategory == "全部") && a.dramaQuery == "" && (a.dramaType == "" || a.dramaType == "全部") && len(filtered) > 0 {
			hero := filtered[0]
			a.dramaHeroBanner(c, hero)
		}

		// Drama Cards Grid (Responsive multi-column standard grid)
		if len(filtered) == 0 {
			ui.Box(c).Height(180).Center().Children(func() {
				ui.Text(c, "未找到符合条件的剧目，换个分类或榜单试试吧").FontSize(14).TextColor(p.muted)
			})
			return
		}

		cols := a.columns(c, 190)
		ui.Grid(c.Key("drama-grid")).Columns(cols).GapX(18).GapY(28).Children(func() {
			for _, item := range filtered {
				drama := item
				a.dramaCard(c, drama)
			}
		})
	})
}

// dramaHeroBanner renders a high-impact horizontal billboard for the top featured drama.
func (a *app) dramaHeroBanner(c *ui.Context, hero *vcp.DramaItem) {
	p := a.pal
	card := ui.Box(c.Key("hero-banner")).Height(210).Radius(14).
		Background(p.card).Border(1, p.hairline).Padding(16, 20, 16, 20).Margin(0, 0, 10, 0)

	card.Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Center).Gap(20).Children(func() {
			// Cover Poster (Vertical 3:4.2)
			coverURL := fmt.Sprintf("flashmusic://drama-cover/%d", hero.ID)
			a.coverPoster(c, coverURL, 130, 182, 10)

			// Info
			ui.Column(c).Grow(1).Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Padding(3, 8, 3, 8).Radius(4).Background(p.accent).Children(func() {
						ui.Text(c, "🔥 全网热播 TOP 1").FontSize(11).FontWeight(700).TextColor(p.onAccent)
					})
					if hero.Category != "" {
						ui.Box(c).Padding(3, 8, 3, 8).Radius(4).Background(p.hover).Children(func() {
							ui.Text(c, hero.Category).FontSize(11.5).TextColor(p.text)
						})
					}
					ui.Box(c).Padding(3, 8, 3, 8).Radius(4).Background(p.bg).Border(1, p.hairline).Children(func() {
						ui.Text(c, fmt.Sprintf("%d 集全 · 1080P 超清", hero.EpisodeCount)).FontSize(11.5).TextColor(p.muted)
					})
					ui.Box(c).Padding(3, 8, 3, 8).Radius(4).Background(p.press).Children(func() {
						ui.Text(c, hero.Origin.OriginLabel()).FontSize(11).FontWeight(600).TextColor(p.accent)
					})
				})

				ui.Text(c, hero.Title).FontSize(23).FontWeight(700).TextColor(p.text)
				desc := hero.Description
				if len([]rune(desc)) > 110 {
					desc = string([]rune(desc)[:110]) + "..."
				}
				ui.Text(c, desc).FontSize(12.5).TextColor(p.muted).LineHeight(1.4)

				ui.Row(c).AlignItems(ui.Center).Gap(12).Margin(4, 0, 0, 0).Children(func() {
					playBtn := pillButton(c.Key("hero-play"), iconPlay, "立即播放（PC 宽屏影院）", p.accent, p.onAccent)
					if playBtn.Clicked() {
						a.dramaWidescreenMode = true
						a.startDramaEpisode(hero, 0)
					}

					p916Btn := pillButton(c.Key("hero-916"), iconAspectRatio, "📱 9:16 竖屏模式", p.hover, p.text).
						Border(1, p.hairline)
					if p916Btn.Clicked() {
						a.dramaWidescreenMode = false
						a.startDramaEpisode(hero, 0)
					}
				})
			})
		})
	})
}

// dramaCard renders a single poster card with cover, episode badge, title, and play count.
func (a *app) dramaCard(c *ui.Context, drama *vcp.DramaItem) {
	p := a.pal
	card := ui.ButtonBase(c.Key(fmt.Sprintf("drama-%d", drama.ID))).
		Column().Gap(8).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label(drama.Title)

	hovered := card.Hovered()

	card.Children(func() {
		// Poster Container: 3:4.2 (0.714 AspectRatio)
		art := ui.Box(c).AspectRatio(0.714).Radius(10).Clip()
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -4}
			art.Shadow(0, 14, 28, -4, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(lift, ui.Snappy)

		art.Children(func() {
			coverURL := fmt.Sprintf("flashmusic://drama-cover/%d", drama.ID)
			a.coverPosterFill(c, coverURL, 10)

			// Top Badges
			ui.Row(c).Absolute().Top(7).Left(7).Right(7).AlignItems(ui.Center).Children(func() {
				// Left badge: HD
				ui.Box(c).Padding(2, 6).Radius(5).Background(ui.RGBA(0, 0, 0, 0.72)).Children(func() {
					ui.Text(c, "1080P 超清").FontSize(10).FontWeight(600).TextColor(white)
				})
				ui.Spacer(c)
				// Right badge: Total Episodes
				ui.Box(c).Padding(2, 6).Radius(5).Background(ui.RGBA(0, 0, 0, 0.72)).Children(func() {
					ui.Text(c, fmt.Sprintf("%d集全", drama.EpisodeCount)).FontSize(10).FontWeight(600).TextColor(white)
				})
			})

			// Center Glowing Play Button on Hover
			playBtn := ui.Box(c).Absolute().Fill().Center()
			playPose := ui.Pose{Fade: 0, Scale: 0.8}
			if hovered {
				playPose = ui.Pose{Fade: 1, Scale: 1.08}
			}
			playBtn.AnimatePose(playPose, ui.Snappy)
			playBtn.Children(func() {
				ui.Box(c).Size(44, 44).Radius(22).Background(p.accent).Center().
					Shadow(0, 4, 16, 0, p.accent.Alpha(0.6)).Children(func() {
					ui.Icon(c, iconPlay).Size(18, 18).TextColor(p.onAccent).Margin(0, 0, 0, 2)
				})
			})

			// Bottom 3-stop Gradient Overlay with Playcount & Origin
			ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(40).
				LinearGradient(ui.LinearGradient{
					Angle: 180,
					Stops: []ui.GradientStop{
						{Color: ui.RGBA(0, 0, 0, 0), At: 0},
						{Color: ui.RGBA(0, 0, 0, 0.55), At: 0.45},
						{Color: ui.RGBA(0, 0, 0, 0.88), At: 1},
					},
				}).Padding(0, 8, 6, 8).AlignItems(ui.End).Children(func() {
				ui.Row(c).Fill().AlignItems(ui.Center).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
						ui.Icon(c, iconPlay).Size(10, 10).TextColor(white.Alpha(0.9))
						ui.Text(c, formatViews(drama.PlayCount)).FontSize(10.5).FontWeight(600).TextColor(white)
					})
					ui.Spacer(c)
					ui.Text(c, drama.Origin.OriginLabel()).FontSize(10).FontWeight(500).TextColor(white.Alpha(0.75))
				})
			})
		})

		// Title (13.5px font-weight 600, hover highlight with p.accent)
		titleCol := p.text
		if hovered {
			titleCol = p.accent
		}
		ui.Text(c, drama.Title).FontSize(13.5).FontWeight(600).TextColor(titleCol).MaxLines(1)

		// Metadata Sub-row: Category Pill + Description/Tag
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			if drama.Category != "" {
				ui.Box(c).Padding(1, 5).Radius(4).Background(p.hover).Children(func() {
					ui.Text(c, drama.Category).FontSize(10.5).FontWeight(600).TextColor(p.accent)
				})
			}
			desc := drama.Description
			if len(drama.Tags) > 0 && drama.Tags[0] != "" {
				desc = drama.Tags[0]
			}
			if desc != "" {
				ui.Text(c, desc).FontSize(11.5).TextColor(p.muted).SingleLine().Grow(1).Shrink(1)
			}
		})
	})

	if card.Clicked() {
		a.startDramaEpisode(drama, 0)
	}
}

// dramaTheater renders the PC Widescreen (16:9) or optional 9:16 portrait video theater.
func (a *app) dramaTheater(c *ui.Context) {
	p := a.pal
	drama := vcp.GetDrama(a.activeDramaID)
	if drama == nil {
		all := vcp.GetDramas()
		if len(all) > 0 {
			drama = all[0]
			a.activeDramaID = drama.ID
		}
	}
	if drama == nil {
		ui.Box(c).Height(200).Center().Children(func() {
			ui.Text(c, "暂无可播剧目，请返回精选大厅选择").FontSize(14).TextColor(p.muted)
		})
		return
	}

	// 首次进入剧场播放时，如果尚未起播，自动加载首集极速开播
	if !a.dramaPlaying && a.videoWeb == nil {
		a.startDramaEpisode(drama, a.activeEpisodeIndex)
	}

	totalEp := len(drama.Episodes)
	if totalEp == 0 {
		totalEp = drama.EpisodeCount
	}
	if a.activeEpisodeIndex < 0 || a.activeEpisodeIndex >= totalEp {
		a.activeEpisodeIndex = 0
	}
	currentEpNum := a.activeEpisodeIndex + 1

	ui.Column(c.Key("theater-view")).Fill().Gap(14).Children(func() {
		// Theater Top Navigation Toolbar
		ui.Row(c.Key("theater-bar")).AlignItems(ui.Center).Gap(12).Children(func() {
				backBtn := pillButton(c.Key("th-back"), iconBack, "返回剧场大厅", p.hover, p.text).
					Border(1, p.hairline)
				if backBtn.Clicked() {
					a.activeDramaTab = 0
					if a.dramaPlaying {
						a.stopVideo()
					}
				}

			ui.Text(c, fmt.Sprintf("《%s》", drama.Title)).FontSize(17).FontWeight(700).TextColor(p.text)
			ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(p.press).Children(func() {
				ui.Text(c, fmt.Sprintf("第 %d 集 / 共 %d 集", currentEpNum, totalEp)).
					FontSize(11.5).FontWeight(600).TextColor(p.accent)
			})

			ui.Spacer(c)

				// 画幅切换按钮 (PC 宽屏影院 16:9 vs 9:16 沉浸竖屏)
				modeLabel := "🖥️ PC 宽屏影院 (默认)"
				modeBadge := "16:9"
				if !a.dramaWidescreenMode {
					modeLabel = "📱 9:16 沉浸竖屏"
					modeBadge = "手机微窗"
				}
				ratioBtn := ui.ButtonBase(c.Key("toggle-aspect")).Height(30).PaddingX(10).Radius(15).
					Background(p.card).Border(1, p.hairline).AlignItems(ui.Center).Gap(6).FocusRing(false)
				ratioBtn.Children(func() {
					ui.Icon(c, iconAspectRatio).Size(14, 14).TextColor(p.accent)
					ui.Text(c, modeLabel).FontSize(12).FontWeight(600).TextColor(p.text)
					ui.Box(c).Padding(1, 4, 1, 4).Radius(3).Background(p.press).Children(func() {
						ui.Text(c, modeBadge).FontSize(10).TextColor(p.accent)
					})
				})
				if ratioBtn.Clicked() {
					a.dramaWidescreenMode = !a.dramaWidescreenMode
				}

				// 影院沉浸放大切换按钮 (放大 16:9 舞台并收起右侧抽屉，或还原双栏布局)
				expandLabel := "⛶ 影院放大"
				if a.dramaTheaterExpanded {
					expandLabel = "🗗 还原双栏"
				}
				expandBtn := ui.ButtonBase(c.Key("toggle-theater-expand")).Height(30).PaddingX(10).Radius(15).
					Background(p.card).Border(1, p.hairline).AlignItems(ui.Center).Gap(6).FocusRing(false)
				expandBtn.Children(func() {
					ui.Text(c, expandLabel).FontSize(12).FontWeight(600).TextColor(p.text)
				})
				if expandBtn.Clicked() {
					a.dramaTheaterExpanded = !a.dramaTheaterExpanded
				}

				// 1080P Ultra-HD Tag
				ui.Box(c).Padding(3, 8, 3, 8).Radius(6).Background(p.card).Border(1, p.hairline).Children(func() {
					ui.Text(c, "1080P 超清").FontSize(11.5).FontWeight(600).TextColor(p.muted)
				})
			})

			// Main Theater Body: Left Video Canvas + Right Episode Selector Drawer
			if a.dramaWidescreenMode {
				a.dramaWidescreenTheater(c, drama, currentEpNum, totalEp)
			} else {
				a.dramaPortraitTheater(c, drama, currentEpNum, totalEp)
			}
		})
	}

// dramaWidescreenTheater implements the default desktop PC 16:9 theater layout.
func (a *app) dramaWidescreenTheater(c *ui.Context, drama *vcp.DramaItem, currentEpNum, totalEp int) {
	p := a.pal

	ui.Row(c.Key("widescreen-body")).Fill().AlignItems(ui.Start).Gap(18).Children(func() {
		// Left: 16:9 Stage Area (fills available width, or 100% full-width when expanded)
		ui.Column(c.Key("ws-left-stage")).Grow(1).Gap(12).Children(func() {
			// 16:9 Screen Box (严格锁定 16:9 物理黄金比例，告别拉伸变形)
			screen := ui.Box(c.Key("screen-16-9")).AspectRatio(16.0 / 9.0).Radius(12).Clip().
				Background(ui.Hex("#000")).Border(1, p.hairline)

				screen.Children(func() {
					if a.dramaPlaying && a.videoWeb != nil {
						ui.WebView(c, a.videoWeb).Fill()
					} else {
						// 1. Standby full-bleed cover backdrop (100% 铺满整个 16:9 舞台)
						coverURL := fmt.Sprintf("flashmusic://drama-cover/%d", drama.ID)
						a.coverBackdrop(c, coverURL)

						// 2. Translucent dark overlay with floating glowing play button (浮在封面正中央)
						overlay := ui.ButtonBase(c.Key("stage-overlay-btn")).Absolute().Fill().
							Background(ui.RGBA(0, 0, 0, 0.42)).Center().FocusRing(false)
						overlay.Children(func() {
							ui.Column(c).Center().Gap(14).Children(func() {
								// Center Glowing Play Circle (大号发光播放按钮)
								playCircle := ui.Box(c).Size(72, 72).Radius(36).
									Background(p.accent).Center().Shadow(0, 6, 24, 0, p.accent.Alpha(0.65))
								playCircle.Children(func() {
									ui.Icon(c, iconPlay).Size(30, 30).TextColor(p.onAccent)
								})

									ui.Text(c, fmt.Sprintf("《%s》第 %d 集 · 待起播", drama.Title, currentEpNum)).
										FontSize(17).FontWeight(700).TextColor(white)
									ui.Text(c, "1080P 超清 · 点击立即播放").
										FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.75))
								})
							})
							if overlay.Clicked() {
								a.startDramaEpisode(drama, a.activeEpisodeIndex)
							}
						}
					})

				// Controls Toolbar below screen
				ui.Box(c.Key("ws-control-bar")).Height(42).Radius(8).Background(p.card).
					Border(1, p.hairline).PaddingX(12).AlignItems(ui.Center).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
							// Prev Episode
							prevBtn := pillButton(c.Key("bar-prev"), iconPrev, "上一集", p.hover, p.text).
								Border(1, p.hairline)
							if prevBtn.Clicked() && a.activeEpisodeIndex > 0 {
								a.startDramaEpisode(drama, a.activeEpisodeIndex-1)
							}

								// Play / Pause
								playLabel := "▶ 播放"
								if a.dramaPlaying {
									playLabel = "⏸ 暂停"
								}
								playBtn := pillButton(c.Key("bar-play-toggle"), nil, playLabel, p.accent, p.onAccent)
								if playBtn.Clicked() {
									a.toggleDramaPlay()
								}

							// Next Episode
							nextBtn := pillButton(c.Key("bar-next"), iconNext, "下一集", p.hover, p.text).
								Border(1, p.hairline)
							if nextBtn.Clicked() && a.activeEpisodeIndex < totalEp-1 {
								a.startDramaEpisode(drama, a.activeEpisodeIndex+1)
							}

						ui.Text(c, fmt.Sprintf("正在播放: 第 %d 集 / 共 %d 集", currentEpNum, totalEp)).
							FontSize(12).FontWeight(600).TextColor(p.text)

						ui.Spacer(c)
					})
				})

			// Below Stage: Synopsis and Tags Card
			ui.Box(c.Key("drama-meta-card")).Radius(10).Background(p.card).
				Border(1, p.hairline).Padding(14, 18, 14, 18).Children(func() {
				ui.Column(c).Gap(8).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
						ui.Text(c, drama.Title).FontSize(16).FontWeight(700).TextColor(p.text)
						if drama.Category != "" {
							ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(p.hover).Children(func() {
								ui.Text(c, drama.Category).FontSize(11).TextColor(p.accent)
							})
						}
						for _, t := range drama.Tags {
							ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(p.bg).Border(1, p.hairline).Children(func() {
								ui.Text(c, t).FontSize(10.5).TextColor(p.muted)
							})
						}
						ui.Spacer(c)
						ui.Text(c, fmt.Sprintf("🔥 热度 %s · 点赞 %s", formatViews(drama.PlayCount), formatViews(drama.LikeCount))).
							FontSize(11.5).TextColor(p.faint)
					})
					ui.Text(c, drama.Description).FontSize(12.5).TextColor(p.muted)
				})
			})
		})

			// Right: Episode Selection Drawer (width 350, hidden when theater is expanded)
			if !a.dramaTheaterExpanded {
				a.dramaEpisodeDrawer(c, drama, currentEpNum, totalEp, 350)
			}
		})
	}

// dramaPortraitTheater implements the 9:16 mobile portrait preview mode.
func (a *app) dramaPortraitTheater(c *ui.Context, drama *vcp.DramaItem, currentEpNum, totalEp int) {
	p := a.pal

	ui.Row(c.Key("portrait-body")).Fill().AlignItems(ui.Start).Gap(24).Justify(ui.Center).Children(func() {
		// Centered 9:16 Phone Canvas
		phone := ui.Box(c.Key("phone-frame")).AspectRatio(9.0 / 16.0).Height(560).Radius(18).Clip().
			Background(ui.Hex("#000")).Border(1.5, p.accent.Alpha(0.4))

		phone.Children(func() {
			if a.dramaPlaying && a.videoWeb != nil {
				ui.WebView(c, a.videoWeb).Fill()
				} else {
					coverURL := fmt.Sprintf("flashmusic://drama-cover/%d", drama.ID)
					a.coverBackdrop(c, coverURL)

					// Translucent overlay with centered floating play button
					overlay := ui.ButtonBase(c.Key("p-stage-overlay-btn")).Absolute().Fill().
						Background(ui.RGBA(0, 0, 0, 0.42)).Center().FocusRing(false)
					overlay.Children(func() {
						ui.Column(c).Center().Gap(12).Children(func() {
							playCircle := ui.Box(c).Size(64, 64).Radius(32).
								Background(p.accent).Center().Shadow(0, 6, 20, 0, p.accent.Alpha(0.65))
							playCircle.Children(func() {
								ui.Icon(c, iconPlay).Size(26, 26).TextColor(p.onAccent)
							})
							ui.Text(c, fmt.Sprintf("第 %d 集", currentEpNum)).FontSize(15).FontWeight(600).TextColor(white)
							ui.Text(c, "9:16 沉浸竖屏 · 点击立即开播").FontSize(11.5).TextColor(ui.RGBA(255, 255, 255, 0.85))
						})
					})
					if overlay.Clicked() {
						a.startDramaEpisode(drama, a.activeEpisodeIndex)
					}
				}
		})

		// Right: Episode Selection Drawer (width 350)
		if !a.dramaTheaterExpanded {
			a.dramaEpisodeDrawer(c, drama, currentEpNum, totalEp, 350)
		}
	})
}

// dramaEpisodeDrawer renders the right-hand episode selector drawer with clean single-row sliding pagination.
func (a *app) dramaEpisodeDrawer(c *ui.Context, drama *vcp.DramaItem, currentEpNum, totalEp int, width float32) {
	p := a.pal

	drawer := ui.Box(c.Key("ep-drawer")).Width(width).Radius(12).
		Background(p.card).Border(1, p.hairline).Padding(14)

	drawer.Children(func() {
		ui.Column(c).Fill().Gap(10).Children(func() {
			// 1. Header: Title + Auto Next Toggle
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Icon(c, iconClapperboard).Size(16, 16).TextColor(p.accent)
					ui.Text(c, "正片选集").FontSize(14).FontWeight(700).TextColor(p.text)
					ui.Text(c, fmt.Sprintf("(共%d集)", totalEp)).FontSize(12).TextColor(p.muted)
				})
				ui.Spacer(c)

				// Auto Next Toggle
				autoLabel := "⚡ 自动连播 开"
				autoBg := p.accent.Alpha(0.12)
				autoBorder := p.accent.Alpha(0.3)
				autoText := p.accent
				if !a.dramaAutoNext {
					autoLabel = "⚪ 自动连播 关"
					autoBg = p.bg
					autoBorder = p.hairline
					autoText = p.faint
				}
				autoBtn := ui.ButtonBase(c.Key("toggle-auto-next")).Height(24).PaddingX(8).Radius(12).
					Background(autoBg).Border(1, autoBorder).Center().FocusRing(false)
				autoBtn.Children(func() {
					ui.Text(c, autoLabel).FontSize(10.5).FontWeight(600).TextColor(autoText)
				})
				if autoBtn.Clicked() {
					a.dramaAutoNext = !a.dramaAutoNext
				}
			})

			// 2. Single-Row Sliding Pagination Bar (严禁换行，固定单行 32px 高度)
			const pageSize = 50
			totalRanges := (totalEp + pageSize - 1) / pageSize
			if a.dramaEpRangeStart < 0 || a.dramaEpRangeStart >= totalEp {
				a.dramaEpRangeStart = (a.activeEpisodeIndex / pageSize) * pageSize
			}
			curRangeIdx := a.dramaEpRangeStart / pageSize

			// 计算可见的 4 组滑动窗口
			const windowSize = 4
			winStart := curRangeIdx - 1
			if winStart < 0 {
				winStart = 0
			}
			if winStart+windowSize > totalRanges {
				winStart = max(0, totalRanges-windowSize)
			}
			winEnd := min(totalRanges, winStart+windowSize)

			ui.Box(c.Key("range-bar")).Height(32).Radius(8).Background(p.bg).
				Border(1, p.hairline).PaddingX(6).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Fill().AlignItems(ui.Center).Gap(4).Children(func() {
					// Prev Range Arrow [◀]
					prevRangeBtn := ui.ButtonBase(c.Key("range-prev")).Size(24, 24).Radius(4).Center().FocusRing(false)
					if curRangeIdx > 0 {
						prevRangeBtn.Background(p.card)
					} else {
						prevRangeBtn.Background(ui.Transparent)
					}
					prevRangeBtn.Children(func() {
						arrowColor := p.text
						if curRangeIdx == 0 {
							arrowColor = p.hairline
						}
						ui.Icon(c, iconPrev).Size(12, 12).TextColor(arrowColor)
					})
					if prevRangeBtn.Clicked() && curRangeIdx > 0 {
						a.dramaEpRangeStart = (curRangeIdx - 1) * pageSize
					}

					// 4 Sliding Range Tabs
					ui.Row(c.Key("range-pills")).Grow(1).AlignItems(ui.Center).Justify(ui.Center).Gap(6).Children(func() {
						for r := winStart; r < winEnd; r++ {
							rangeIdx := r
							rStart := rangeIdx * pageSize
							rEnd := min(rStart+pageSize, totalEp)
							isSelected := a.dramaEpRangeStart == rStart

							label := fmt.Sprintf("%d-%d", rStart+1, rEnd)
							tabBtn := ui.ButtonBase(c.Key(fmt.Sprintf("rtab-%d", rangeIdx))).
								Height(24).PaddingX(8).Radius(6).Center().FocusRing(false)

							tBg := ui.Transparent
							tText := p.muted
							tWeight := 500
							if isSelected {
								tBg = p.accent
								tText = p.onAccent
								tWeight = 700
							} else if tabBtn.Hovered() {
								tBg = p.hover
								tText = p.text
							}
							tabBtn.Background(tBg)
							tabBtn.Children(func() {
								ui.Text(c, label).FontSize(11).FontWeight(tWeight).TextColor(tText)
							})

							if tabBtn.Clicked() {
								a.dramaEpRangeStart = rStart
							}
						}
					})

					// Next Range Arrow [▶]
					nextRangeBtn := ui.ButtonBase(c.Key("range-next")).Size(24, 24).Radius(4).Center().FocusRing(false)
					if curRangeIdx < totalRanges-1 {
						nextRangeBtn.Background(p.card)
					} else {
						nextRangeBtn.Background(ui.Transparent)
					}
					nextRangeBtn.Children(func() {
						arrowColor := p.text
						if curRangeIdx >= totalRanges-1 {
							arrowColor = p.hairline
						}
						ui.Icon(c, iconNext).Size(12, 12).TextColor(arrowColor)
					})
					if nextRangeBtn.Clicked() && curRangeIdx < totalRanges-1 {
						a.dramaEpRangeStart = (curRangeIdx + 1) * pageSize
					}
				})
			})

			// 3. Sub-bar: Range Info & Current Episode Indicator
			startEp := a.dramaEpRangeStart
			endEp := min(startEp+pageSize, totalEp)
			ui.Row(c).AlignItems(ui.Center).Padding(0, 2, 0, 2).Children(func() {
				ui.Text(c, fmt.Sprintf("分集范围: 第 %d - %d 集", startEp+1, endEp)).
					FontSize(11).TextColor(p.faint)
				ui.Spacer(c)
				ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
					ui.Box(c).Size(6, 6).Radius(3).Background(p.accent)
					ui.Text(c, fmt.Sprintf("正在播放: 第 %d 集", currentEpNum)).
						FontSize(11).FontWeight(600).TextColor(p.accent)
				})
			})

			// 4. 5-Column Clean Episode Matrix (5列精致方块)
			ui.Row(c.Key("ep-grid")).Wrap().Gap(6).Children(func() {
				for i := startEp; i < endEp; i++ {
					epIndex := i
					epNum := epIndex + 1
					isCurrent := epIndex == a.activeEpisodeIndex
					isFree := epIndex < 10

					label := fmt.Sprintf("%d", epNum)
					btn := ui.ButtonBase(c.Key(fmt.Sprintf("ep-btn-%d", epNum))).
						Size(58, 36).Radius(6).Center().FocusRing(false)

					bg := p.bg
					borderColor := p.hairline
					textColor := p.text
					weight := 500
					if isCurrent {
						bg = p.accent
						borderColor = p.accent
						textColor = p.onAccent
						weight = 700
					} else if btn.Hovered() {
						bg = p.hover
						textColor = p.text
					}
					btn.Background(btn.AnimateColor("bg", bg, ui.Animation{Duration: 80 * time.Millisecond})).
						Border(1, borderColor)

					btn.Children(func() {
						ui.Row(c).AlignItems(ui.Center).Justify(ui.Center).Gap(3).Children(func() {
							if isCurrent {
								ui.Icon(c, iconPlay).Size(10, 10).TextColor(p.onAccent)
							}
							ui.Text(c, label).FontSize(13).FontWeight(weight).TextColor(textColor)
							if isFree && !isCurrent {
								ui.Box(c).Size(4, 4).Radius(2).Background(ui.Hex("#10b981"))
							}
						})
					})

							if btn.Clicked() {
								a.startDramaEpisode(drama, epIndex)
							}
				}
			})
		})
	})
}

// dramasHistory renders the user's drama playback history.
func (a *app) dramasHistory(c *ui.Context) {
	p := a.pal

	if len(a.dramaHistory) == 0 {
		ui.Box(c).Height(240).Center().Children(func() {
			ui.Column(c).Center().Gap(10).Children(func() {
				ui.Icon(c, iconClapperboard).Size(40, 40).TextColor(p.faint)
				ui.Text(c, "暂无追剧历史，快去精选大厅观看心仪短剧吧").FontSize(14).TextColor(p.muted)
				goHall := pillButton(c.Key("go-hall"), iconHome, "前往精选大厅", p.accent, p.onAccent)
				if goHall.Clicked() {
					a.activeDramaTab = 0
				}
			})
		})
		return
	}

	ui.Column(c.Key("history-view")).Gap(14).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("追剧历史记录（共 %d 部）", len(a.dramaHistory))).
				FontSize(14).FontWeight(600).TextColor(p.text)
			ui.Spacer(c)
			clearHistory := pillButton(c.Key("clear-dh"), iconTrash, "清空历史", p.hover, p.muted).
				Border(1, p.hairline)
			if clearHistory.Clicked() {
				a.dramaHistory = nil
			}
		})

		cols := a.columns(c, 190)
		ui.Grid(c.Key("history-grid")).Columns(cols).GapX(18).GapY(28).Children(func() {
			for _, id := range a.dramaHistory {
				if drama := vcp.GetDrama(id); drama != nil {
					a.dramaCard(c, drama)
				}
			}
		})
	})
}

func (a *app) recordDramaHistory(id int64) {
	for i, h := range a.dramaHistory {
		if h == id {
			a.dramaHistory = append(a.dramaHistory[:i], a.dramaHistory[i+1:]...)
			break
		}
	}
	a.dramaHistory = append([]int64{id}, a.dramaHistory...)
}

func formatViews(count int64) string {
	if count >= 10000 {
		return fmt.Sprintf("%.1f万", float64(count)/10000.0)
	}
	return fmt.Sprintf("%d", count)
}

// startDramaEpisode initiates playback of the specified drama episode and triggers lookahead prefetch.
func (a *app) startDramaEpisode(drama *vcp.DramaItem, epIndex int) {
	if drama == nil {
		return
	}
	a.activeDramaID = drama.ID
	a.activeEpisodeIndex = epIndex
	a.dramaEpRangeStart = (epIndex / 50) * 50
	a.activeDramaTab = 1
	a.dramaPlaying = true
	a.recordDramaHistory(drama.ID)
	embedURL := vcp.GetDramaEpisodeEmbedURL(drama, epIndex)
	a.playVideo(embedURL)
	vcp.PrefetchDramaEpisodes(drama.ID, epIndex, 2)
}

// playNextDramaEpisode advances playback to the next episode of the active drama and updates UI.
func (a *app) playNextDramaEpisode() {
	drama := vcp.GetDrama(a.activeDramaID)
	if drama == nil {
		return
	}
	totalEp := len(drama.Episodes)
	if totalEp == 0 {
		totalEp = drama.EpisodeCount
	}
	if a.activeEpisodeIndex < totalEp-1 {
		a.startDramaEpisode(drama, a.activeEpisodeIndex+1)
		a.redraw()
	}
}

// toggleDramaPlay toggles video play/pause for the active short drama.
func (a *app) toggleDramaPlay() {
	if a.dramaPlaying {
		a.stopVideo()
		return
	}
	drama := vcp.GetDrama(a.activeDramaID)
	if drama == nil {
		all := vcp.GetDramas()
		if len(all) > 0 {
			drama = all[0]
			a.activeDramaID = drama.ID
		}
	}
	if drama != nil {
		a.startDramaEpisode(drama, a.activeEpisodeIndex)
	}
}
