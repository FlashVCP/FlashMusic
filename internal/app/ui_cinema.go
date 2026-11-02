package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/vcp"
)

// cinemaPage renders the Cinema & Movies view driven by station master defaults and APIs.
func (a *app) cinemaPage(c *ui.Context) {
	top, bottom := a.insets(c)
	p := a.pal

	if a.cinemaChannel == "" {
		a.cinemaChannel = "精选影视"
	}
	if a.cinemaGenre == "" {
		a.cinemaGenre = "全部"
	}

	tax := vcp.GetCinemaTaxonomy()
	movies := vcp.FilterCinemas(a.cinemaChannel, a.cinemaGenre, a.cinemaQuery)

	ui.Scroll(c.Key("cinema-scroll-page")).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		// Page Header
		ui.Row(c.Key("cinema-header")).AlignItems(ui.Center).Gap(14).Margin(0, 0, 16, 0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, iconTv).Size(22, 22).TextColor(p.accent)
				ui.Text(c, "院线影视").FontSize(20).FontWeight(700).TextColor(p.text)
			})

			ui.Box(c).Padding(2, 7, 2, 7).Radius(4).Background(p.press).Children(func() {
				ui.Text(c, "4K 蓝光杜比私享影院 · 站长直供").FontSize(11).FontWeight(600).TextColor(p.accent)
			})

			ui.Spacer(c)

			// Movie Search Box
			searchField := ui.Row(c.Key("cinema-search-box")).Height(32).PaddingX(12).Gap(8).Radius(16).
				AlignItems(ui.Center).Width(240).Background(p.card).Border(1, p.hairline)
			searchField.Children(func() {
				ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.faint)
				ui.TextInputBase(c.Key("cinema-query"), &a.cinemaQuery).
					Placeholder("搜索电影、主演、导演...").FontSize(12.5).Label("搜索电影").
					Grow(1).MinWidth(0)

				if a.cinemaQuery != "" {
					if iconButton(c.Key("clear-cq"), iconX, 12, p.muted, "清除").Size(16, 16).Clicked() {
						a.cinemaQuery = ""
					}
				}
			})
		})

		// Channel Tabs Row (对齐站长管理端默认栏目: 找剧 / 社区 / 漫剧 / 经典电影 / 精选影视 / 4K蓝光 / 纪录片)
		ui.Row(c.Key("cinema-channels")).AlignItems(ui.Center).Gap(20).Padding(0, 4, 6, 4).Children(func() {
			for _, ch := range tax.Channels {
				channelName := ch
				active := a.cinemaChannel == channelName
				btn := ui.ButtonBase(c.Key("ch-" + channelName)).Height(32).PaddingX(6).
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
						ui.Text(c, channelName).FontSize(size).FontWeight(weight).TextColor(textColor)
						if active {
							ui.Box(c).Width(24).Height(3).Radius(1.5).Background(p.accent)
						} else {
							ui.Box(c).Height(3)
						}
					})
				})
				if btn.Clicked() {
					a.cinemaChannel = channelName
				}
			}
		})

		// Genre Tags Row (对齐站长管理端默认标签: 全部 / 喜剧 / 动作 / 爱情 / 都市 / 悬疑 / 古装 / 奇幻 / 战争 / 科幻)
		ui.Row(c.Key("cinema-genres")).AlignItems(ui.Center).Gap(8).Wrap().Margin(4, 0, 10, 0).Children(func() {
			for _, g := range tax.Genres {
				genreName := g
				active := a.cinemaGenre == genreName
				btn := ui.ButtonBase(c.Key("genre-" + genreName)).Height(26).PaddingX(12).Radius(999).
					AlignItems(ui.Center).FocusRing(false).Label(genreName)

				if active {
					btn.Background(p.accent).TextColor(p.onAccent)
				} else {
					btn.Background(p.card).TextColor(p.muted).Border(1, p.hairline)
					if btn.Hovered() {
						btn.Background(p.hover).TextColor(p.text)
					}
				}
				btn.Children(func() {
					ui.Text(c, genreName).FontSize(11.5).FontWeight(map[bool]int{true: 700, false: 500}[active])
				})
				if btn.Clicked() {
					a.cinemaGenre = genreName
				}
			}

			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("精选呈现 %d 部院线佳作", len(movies))).FontSize(12).TextColor(p.faint)
		})

		// Featured Movie Billboard (when on first channel and no query)
		if (a.cinemaGenre == "全部" || a.cinemaGenre == "") && a.cinemaQuery == "" && len(movies) > 0 {
			a.cinemaHeroBanner(c, movies[0])
		}

		// Movie Grid Cards
		if len(movies) == 0 {
			ui.Box(c).Height(180).Center().Children(func() {
				ui.Text(c, "未找到符合条件的影视资源，换个分类或标签试试吧").FontSize(14).TextColor(p.muted)
			})
			return
		}

		cols := a.columns(c, 190)
		ui.Grid(c.Key("cinema-grid")).Columns(cols).GapX(18).GapY(28).Children(func() {
			for _, m := range movies {
				movie := m
				a.cinemaCard(c, movie)
			}
		})
	})
}

func (a *app) cinemaHeroBanner(c *ui.Context, hero *vcp.CinemaItem) {
	p := a.pal
	card := ui.Box(c.Key("cinema-hero")).Height(210).Radius(14).
		Background(p.card).Border(1, p.hairline).Padding(16, 20, 16, 20).Margin(0, 0, 10, 0)

	card.Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Center).Gap(20).Children(func() {
			// Poster Cover
			ui.Box(c).Size(130, 182).Radius(10).Clip().Children(func() {
				a.coverPosterFill(c, hero.CoverURL, 10)
			})

			// Info
			ui.Column(c).Grow(1).Gap(8).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
					ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(p.accent).Children(func() {
						ui.Text(c, hero.Resolution).FontSize(10.5).FontWeight(700).TextColor(p.onAccent)
					})
					ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(p.hover).Children(func() {
						ui.Text(c, hero.Genre+" · "+hero.Year).FontSize(11).TextColor(p.muted)
					})
					ui.Text(c, hero.Score+" 分 · 杜比全景声").FontSize(12).FontWeight(700).TextColor(ui.Hex("#f59e0b"))
				})

				ui.Text(c, hero.Title).FontSize(23).FontWeight(700).TextColor(p.text)
				ui.Text(c, "主演: "+hero.Actors+" · 片长: "+hero.Duration).FontSize(12.5).TextColor(p.muted)

				desc := hero.Description
				if len([]rune(desc)) > 110 {
					desc = string([]rune(desc)[:110]) + "..."
				}
				ui.Text(c, desc).FontSize(12.5).TextColor(p.faint).LineHeight(1.4)

				ui.Row(c).AlignItems(ui.Center).Gap(12).Margin(4, 0, 0, 0).Children(func() {
					pillButton(c.Key("hero-cinema-play"), iconPlay, "立即播放正片（PC 宽屏大屏）", p.accent, p.onAccent)
					pillButton(c.Key("hero-cinema-trailer"), iconFilm, "预告片与花絮", p.hover, p.text).Border(1, p.hairline)
				})
			})
		})
	})
}

func (a *app) cinemaCard(c *ui.Context, m *vcp.CinemaItem) {
	p := a.pal

	card := ui.ButtonBase(c.Key("cinema-card-" + m.ID)).
		Column().Gap(8).AlignItems(ui.Stretch).Justify(ui.Start).FocusRing(false).Label(m.Title)

	hovered := card.Hovered()

	card.Children(func() {
		// Poster Container: Standard 2:3 Cinematic Ratio (0.667)
		art := ui.Box(c).AspectRatio(0.667).Radius(10).Clip()
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
			a.coverPosterFill(c, m.CoverURL, 10)

			// Top Left Resolution Badge
			ui.Box(c).Absolute().Top(7).Left(7).Padding(2, 6).Radius(5).
				Background(ui.RGBA(0, 0, 0, 0.72)).Children(func() {
				ui.Text(c, m.Resolution).FontSize(10).FontWeight(700).TextColor(white)
			})

			// Top Right Gold Score Badge
			ui.Box(c).Absolute().Top(7).Right(7).Padding(2, 6).Radius(5).
				Background(ui.Hex("#f59e0b")).Children(func() {
				ui.Text(c, m.Score).FontSize(10.5).FontWeight(800).TextColor(white)
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

			// Bottom Duration & Year Bar
			ui.Box(c).Absolute().Left(0).Right(0).Bottom(0).Height(36).
				LinearGradient(ui.LinearGradient{
					Angle: 180,
					Stops: []ui.GradientStop{
						{Color: ui.RGBA(0, 0, 0, 0), At: 0},
						{Color: ui.RGBA(0, 0, 0, 0.6), At: 0.45},
						{Color: ui.RGBA(0, 0, 0, 0.88), At: 1},
					},
				}).Padding(0, 8, 6, 8).AlignItems(ui.End).Children(func() {
				ui.Row(c).Fill().AlignItems(ui.Center).Children(func() {
					ui.Text(c, m.Duration).FontSize(10.5).FontWeight(600).TextColor(white)
					ui.Spacer(c)
					ui.Text(c, m.Year).FontSize(10.5).TextColor(white.Alpha(0.8))
				})
			})
		})

		// Title
		titleCol := p.text
		if hovered {
			titleCol = p.accent
		}
		ui.Text(c, m.Title).FontSize(13.5).FontWeight(600).TextColor(titleCol).MaxLines(1)

		// Actors / Genre
		ui.Text(c, m.Genre+" · "+m.Actors).FontSize(11.5).TextColor(p.muted).SingleLine()
	})
}
