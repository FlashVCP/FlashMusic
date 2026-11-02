package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/vcp"
)

// shortVideosPage renders the modern, clean, premium Short Video feed.
func (a *app) shortVideosPage(c *ui.Context) {
	top, bottom := a.insets(c)
	p := a.pal

	if a.svCategory == "" {
		a.svCategory = "全部"
	}
	if a.svSort == "" {
		a.svSort = "heat"
	}

	tax := vcp.GetShortVideoTaxonomy()
	videos := vcp.FilterShortVideosAdvanced(a.svCategory, a.svTag, a.svQuery, a.svDuration, a.svHeat, a.svSort)

	ui.Scroll(c.Key("short-videos-page")).TrackScroll(&a.pageScroll).Grow(1).Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		// 1. Sleek Modern Page Header
		ui.Row(c.Key("sv-header")).AlignItems(ui.Center).Gap(14).Margin(0, 0, 16, 0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, iconSparkle).Size(22, 22).TextColor(p.accent)
				ui.Text(c, "潮流短视频").FontSize(20).FontWeight(700).TextColor(p.text)
			})

			ui.Box(c).Padding(3, 8, 3, 8).Radius(6).Background(p.accent.Alpha(0.12)).
				Border(1, p.accent.Alpha(0.25)).Children(func() {
				ui.Text(c, "✨ 4K超清母带 · 沉浸视界").FontSize(11).FontWeight(600).TextColor(p.accent)
			})

			ui.Spacer(c)

			// Real-time Keyword Search Capsule
			searchField := ui.Row(c.Key("sv-search-box")).Height(32).PaddingX(12).Gap(8).Radius(16).
				AlignItems(ui.Center).Width(240).Background(p.card).Border(1, p.hairline)
			searchField.Children(func() {
				ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.faint)
				ui.TextInput(c.Key("sv-query"), &a.svQuery).
					Placeholder("搜索短视频、创作者...").FontSize(12.5).Label("搜索短视频").
					Grow(1).MinWidth(0)

				if a.svQuery != "" {
					if iconButton(c.Key("clear-svq"), iconX, 12, p.muted, "清除").Size(16, 16).Clicked() {
						a.svQuery = ""
					}
				}
			})
		})

		// 2. Streamlined Category Capsule Chips (单行横向优雅胶囊栏)
		ui.Row(c.Key("sv-categories")).AlignItems(ui.Center).Gap(8).Wrap().Margin(0, 0, 12, 0).Children(func() {
			for _, cat := range tax.Categories {
				catName := cat
				active := a.svCategory == catName
				btn := pillButton(c.Key("sv-cat-"+catName), nil, catName, p.hover, p.text)
				if active {
					btn.Background(p.accent).TextColor(p.onAccent).Border(1, p.accent)
				} else {
					btn.Background(p.card).TextColor(p.muted).Border(1, p.hairline)
				}
				if btn.Clicked() {
					a.svCategory = catName
				}
			}
		})

		// 3. Subtle Single-Line Toolbar (数量统计 + 排序分段控件)
		ui.Row(c.Key("sv-toolbar")).AlignItems(ui.Center).Margin(0, 0, 16, 0).Children(func() {
			ui.Text(c, fmt.Sprintf("精选高能视频（共 %d 条）", len(videos))).
				FontSize(12).FontWeight(600).TextColor(p.muted)

			ui.Spacer(c)

			// Sort Segmented Tabs
			sortOptions := []struct {
				key   string
				label string
			}{
				{"heat", "🔥 最热播放"},
				{"likes", "❤️ 最多点赞"},
				{"newest", "⚡ 最新发布"},
			}
			ui.Row(c.Key("sv-sort-tabs")).Gap(6).Children(func() {
				for _, opt := range sortOptions {
					sortKey := opt.key
					sortLbl := opt.label
					active := a.svSort == sortKey
					sBtn := ui.ButtonBase(c.Key("sort-" + sortLbl)).Height(26).PaddingX(10).Radius(13).
						AlignItems(ui.Center).FocusRing(false)
					sBg := p.card
					sFg := p.muted
					if active {
						sBg = p.accent.Alpha(0.15)
						sFg = p.accent
					}
					sBtn.Background(sBg).Border(1, p.hairline)
					sBtn.Children(func() {
						ui.Text(c, sortLbl).FontSize(11.5).FontWeight(map[bool]int{true: 600, false: 500}[active]).TextColor(sFg)
					})
					if sBtn.Clicked() {
						a.svSort = sortKey
					}
				}
			})
		})

		// 4. Video Cards Waterfall Feed Grid
		if len(videos) == 0 {
			ui.Box(c).Height(240).Center().Children(func() {
				ui.Column(c).AlignItems(ui.Center).Gap(10).Children(func() {
					ui.Icon(c, iconSearch).Size(32, 32).TextColor(p.faint)
					ui.Text(c, "未找到相关短视频，点击下方重置所有筛选").FontSize(13).TextColor(p.muted)
					if pillButton(c.Key("reset-sv-filter"), nil, "重置全部分类与搜索", p.accent, p.onAccent).Clicked() {
						a.svCategory = "全部"
						a.svQuery = ""
						a.svSort = "heat"
					}
				})
			})
			return
		}

		cols := a.columns(c, 210)
		ui.Grid(c.Key("sv-feed-grid")).Columns(cols).GapX(18).GapY(26).Margin(4, 0, 0, 0).Children(func() {
			for i, v := range videos {
				item := v
				var next *vcp.ShortVideoItem
				if i+1 < len(videos) {
					next = videos[i+1]
				}
				a.shortVideoCard(c, item, next)
			}
		})
	})

	// Modal Popup Player
	a.shortVideoPlayModal(c)
}

// shortVideoCard renders a sleek 9:16 mobile-proportioned vertical video card with full-bleed poster.
func (a *app) shortVideoCard(c *ui.Context, item *vcp.ShortVideoItem, next *vcp.ShortVideoItem) {
	p := a.pal

	card := ui.ButtonBase(c.Key("sv-card-" + item.ID)).
		Column().Gap(0).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label(item.Title)

	hovered := card.Hovered()

	card.Children(func() {
		// 9:16 mobile aspect ratio poster frame
		art := ui.Box(c).AspectRatio(9.0 / 16.0).Radius(14).Clip()
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -4}
			art.Shadow(0, 14, 28, -4, p.cardShade)
			art.Border(1.5, p.accent.Alpha(0.6))
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
			art.Border(1, p.hairline)
		}
		art.AnimatePose(lift, ui.Snappy)

		art.Children(func() {
			// 1. Full-Bleed Vertical Cover Poster
			if item.CoverURL != "" {
				a.coverPosterFill(c, item.CoverURL, 14)
			} else {
				ui.Box(c).Fill().Background(ui.Hex("#18181c")).Center().Children(func() {
					ui.Icon(c, iconPlay).Size(40, 40).TextColor(p.faint)
				})
			}

			// 2. Top Bar: Category Pill & Duration Badge
			ui.Row(c).Absolute().Top(8).Left(8).Right(8).AlignItems(ui.Center).Children(func() {
				if item.Category != "" {
					ui.Box(c).Padding(2.5, 7).Radius(6).
						Background(ui.RGBA(15, 23, 42, 0.75)).Border(0.5, ui.RGBA(255, 255, 255, 0.2)).Children(func() {
						ui.Text(c, item.Category).FontSize(10.5).FontWeight(600).TextColor(white)
					})
				}

				ui.Spacer(c)

				if item.Duration != "" {
					ui.Box(c).Padding(2.5, 7).Radius(6).
						Background(ui.RGBA(15, 23, 42, 0.75)).Border(0.5, ui.RGBA(255, 255, 255, 0.2)).Children(func() {
						ui.Text(c, item.Duration).FontSize(10.5).FontWeight(600).TextColor(white)
					})
				}
			})

			// 3. Center Glowing Play Badge on Hover
			playBtn := ui.Box(c).Absolute().Fill().Center()
			playPose := ui.Pose{Fade: 0, Scale: 0.8}
			if hovered {
				playPose = ui.Pose{Fade: 1, Scale: 1.08}
			}
			playBtn.AnimatePose(playPose, ui.Snappy)
			playBtn.Children(func() {
				ui.Box(c).Size(48, 48).Radius(24).Background(p.accent).Center().
					Shadow(0, 4, 16, 0, p.accent.Alpha(0.65)).Children(func() {
					ui.Icon(c, iconPlay).Size(20, 20).TextColor(p.onAccent).Margin(0, 0, 0, 2)
				})
			})

			// 4. Bottom Smooth 3-Stop Dark Glass Gradient Overlay
			ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(110).Padding(12, 12, 12, 12).
				LinearGradient(ui.LinearGradient{
					Angle: 180,
					Stops: []ui.GradientStop{
						{Color: ui.RGBA(0, 0, 0, 0), At: 0},
						{Color: ui.RGBA(0, 0, 0, 0.7), At: 0.4},
						{Color: ui.RGBA(0, 0, 0, 0.95), At: 1},
					},
				}).Children(func() {
				ui.Column(c).Fill().Justify(ui.End).Gap(6).Children(func() {
					// Title
					ui.Text(c, item.Title).FontSize(13.5).FontWeight(700).TextColor(white).MaxLines(2).LineHeight(1.3)

					// Author & View Count Row
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						authorName := "@" + item.Author
						if strings.HasPrefix(item.Author, "@") {
							authorName = item.Author
						}
						ui.Text(c, authorName).FontSize(11.5).FontWeight(500).
							TextColor(ui.RGBA(255, 255, 255, 0.85)).SingleLine().Grow(1)

						ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
							ui.Icon(c, iconHeart).Size(12, 12).TextColor(p.accent)
							ui.Text(c, formatViews(item.PlayCount)).FontSize(11).FontWeight(600).TextColor(white)
						})
					})
				})
			})
		})
	})

	if card.Clicked() {
		a.activeSVID = item.ID
		a.svPlaying = true
		if a.pl != nil && a.pl.Playing() {
			a.pl.TogglePlay()
		}
		playURL := vcp.GetShortVideoEmbedURLWithNext(item, next)
		a.playVideo(playURL)
	}
}

// shortVideoPlayModal renders the immersive popup video player for short videos.
func (a *app) shortVideoPlayModal(c *ui.Context) {
	if !a.svPlaying || a.activeSVID == "" {
		return
	}
	p := a.pal

	var currentItem *vcp.ShortVideoItem
	for _, v := range vcp.GetShortVideos() {
		if v.ID == a.activeSVID {
			currentItem = v
			break
		}
	}
	if currentItem == nil {
		a.svPlaying = false
		return
	}

	ui.DialogBase(c.Key("sv-play-dialog"), &a.svPlaying, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.75))

		// 9:16 mobile aspect ratio frame modal
		panel.Width(440).Height(640).Radius(18).Clip().
			Background(ui.Hex("#0b0d13")).Border(1, p.hairline).
			Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.65)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})

		panel.Children(func() {
			// Top Header Bar
			ui.Row(c.Key("sv-modal-header")).Absolute().Top(0).Left(0).Right(0).Height(44).
				Padding(0, 16, 0, 16).AlignItems(ui.Center).
				Background(ui.RGBA(0, 0, 0, 0.4)).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					ui.Box(c).Size(8, 8).Radius(4).Background(p.accent)
					ui.Text(c, currentItem.Category).FontSize(12).FontWeight(600).TextColor(white)
				})

				ui.Spacer(c)

				// Close Button
				closeBtn := ui.ButtonBase(c.Key("close-sv-modal")).Size(28, 28).Radius(14).
					Background(ui.RGBA(255, 255, 255, 0.15)).Center().FocusRing(false)
				closeBtn.Children(func() {
					ui.Icon(c, iconX).Size(14, 14).TextColor(white)
				})
				if closeBtn.Clicked() {
					a.stopVideo()
				}
			})

			// Center Stage
			ui.Box(c.Key("sv-modal-stage")).Fill().Children(func() {
				if a.videoWeb != nil {
					ui.WebView(c, a.videoWeb).Fill()
				} else {
					a.coverPoster(c, currentItem.CoverURL, 440, 640, 0)
					ui.Box(c).Fill().Background(ui.RGBA(0, 0, 0, 0.55)).Center().Children(func() {
						ui.Column(c).Center().Gap(14).Children(func() {
							playCircle := ui.ButtonBase(c.Key("sv-center-play")).Size(68, 68).Radius(34).
								Background(p.accent).Center().FocusRing(false)
							playCircle.Children(func() {
								ui.Icon(c, iconPlay).Size(28, 28).TextColor(p.onAccent)
							})
							if playCircle.Clicked() {
								embedURL := vcp.GetShortVideoEmbedURL(currentItem)
								a.playVideo(embedURL)
							}
							ui.Text(c, currentItem.Title).FontSize(15).FontWeight(700).TextColor(white).MaxLines(2).TextAlign(ui.Center)
							ui.Text(c, "站长 4T 母带流直出 · 点击即可播放").FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.8))
						})
					})
				}
			})

			// Bottom Info Overlay
			ui.Box(c.Key("sv-modal-footer")).Absolute().Bottom(0).Left(0).Right(0).
				Padding(12, 16, 12, 16).Background(ui.RGBA(0, 0, 0, 0.65)).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, currentItem.Title).FontSize(13.5).FontWeight(600).TextColor(white).MaxLines(2)
					ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
						ui.Text(c, "@"+currentItem.Author).FontSize(12).TextColor(ui.RGBA(255, 255, 255, 0.8))
						ui.Spacer(c)
						ui.Icon(c, iconHeart).Size(13, 13).TextColor(p.accent)
						ui.Text(c, formatViews(currentItem.PlayCount)).FontSize(12).TextColor(white)
					})
				})
			})
		})
	})
}
