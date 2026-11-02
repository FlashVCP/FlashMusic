package app

import (
	"fmt"
	"time"

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

	ui.Column(c.Key("cinema-page")).Fill().Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		// Page Header
		ui.Row(c.Key("cinema-header")).AlignItems(ui.Center).Gap(14).Margin(0, 0, 14, 0).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, iconTv).Size(22, 22).TextColor(p.accent)
				ui.Text(c, "院线影视").FontSize(20).FontWeight(700).TextColor(p.text)
			})

			ui.Box(c).Padding(2, 7, 2, 7).Radius(4).Background(p.press).Children(func() {
				ui.Text(c, "4K 蓝光杜比私享影院 · 站长直供").FontSize(11).FontWeight(600).TextColor(p.accent)
			})

			ui.Spacer(c)

			// Movie Search Box
			searchField := ui.Row(c.Key("cinema-search-box")).Height(30).PaddingX(10).Gap(6).Radius(15).
				AlignItems(ui.Center).Width(220).Background(p.card).Border(1, p.hairline)
			searchField.Children(func() {
				ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.faint)
				ui.TextInput(c.Key("cinema-query"), &a.cinemaQuery).
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
		ui.Row(c.Key("cinema-channels")).AlignItems(ui.Center).Gap(18).Padding(0, 4, 6, 4).Children(func() {
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
					size = 16
				}
				btn.Children(func() {
					ui.Column(c).AlignItems(ui.Center).Gap(3).Children(func() {
						ui.Text(c, channelName).FontSize(size).FontWeight(weight).TextColor(textColor)
						if active {
							ui.Box(c).Width(24).Height(2.5).Radius(1.5).Background(p.accent)
						} else {
							ui.Box(c).Height(2.5)
						}
					})
				})
				if btn.Clicked() {
					a.cinemaChannel = channelName
				}
			}
		})

		// Genre Tags Row (对齐站长管理端默认标签: 全部 / 喜剧 / 动作 / 爱情 / 都市 / 悬疑 / 古装 / 奇幻 / 战争 / 科幻)
		ui.Row(c.Key("cinema-genres")).AlignItems(ui.Center).Gap(8).Wrap().Children(func() {
			for _, g := range tax.Genres {
				genreName := g
				active := a.cinemaGenre == genreName
				btn := ui.ButtonBase(c.Key("genre-" + genreName)).Height(28).PaddingX(12).Radius(14).
					AlignItems(ui.Center).FocusRing(false)

				bg := p.card
				border := p.hairline
				fg := p.muted
				if active {
					bg = p.accent.Alpha(0.12)
					border = p.accent.Alpha(0.4)
					fg = p.accent
				} else if btn.Hovered() {
					bg = p.hover
				}
				btn.Background(bg).Border(1, border)
				btn.Children(func() {
					ui.Text(c, genreName).FontSize(12).FontWeight(map[bool]int{true: 700, false: 500}[active]).TextColor(fg)
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

		ui.Row(c.Key("cinema-grid")).Wrap().Gap(16).Children(func() {
			for _, m := range movies {
				movie := m
				a.cinemaCard(c, movie)
			}
		})
	})
}

func (a *app) cinemaHeroBanner(c *ui.Context, hero *vcp.CinemaItem) {
	p := a.pal
	card := ui.Box(c.Key("cinema-hero")).Height(200).Radius(12).
		Background(p.card).Border(1, p.hairline).Padding(18, 24, 18, 24).Margin(10, 0, 10, 0)

	card.Children(func() {
		ui.Row(c).Fill().AlignItems(ui.Center).Gap(24).Children(func() {
			// Poster Cover
			a.cover(c, hero.CoverURL, 300, 160, 8)

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

				ui.Text(c, hero.Title).FontSize(22).FontWeight(700).TextColor(p.text)
				ui.Text(c, "主演: "+hero.Actors+" · 片长: "+hero.Duration).FontSize(12.5).TextColor(p.muted)

				desc := hero.Description
				if len([]rune(desc)) > 80 {
					desc = string([]rune(desc)[:80]) + "..."
				}
				ui.Text(c, desc).FontSize(12.5).TextColor(p.faint)

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
	cardW := float32(180)
	cardH := float32(275)

	btn := ui.ButtonBase(c.Key("cinema-card-"+m.ID)).
		Width(cardW).Height(cardH).Radius(10).Padding(6).
		AlignItems(ui.Start).Justify(ui.Start).FocusRing(false)

	bg := ui.Transparent
	if btn.Hovered() {
		bg = p.hover
	}
	btn.Background(btn.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))

	btn.Children(func() {
		ui.Column(c).Fill().Gap(6).Children(func() {
			// Poster with Resolution & Score Badges
			ui.Box(c).Width(cardW - 12).Height(200).Radius(8).Clip().Children(func() {
				a.cover(c, m.CoverURL, 200, cardW-12, 8)

				// Top Left Resolution Badge
				ui.Box(c).Padding(2, 6, 2, 6).Radius(4).
					Background(ui.RGBA(0, 0, 0, 0.7)).Margin(6, 0, 0, 6).Children(func() {
					ui.Text(c, m.Resolution).FontSize(10).FontWeight(700).TextColor(white)
				})

				// Top Right Score Badge
				ui.Box(c).Absolute().Top(6).Right(6).Padding(2, 6, 2, 6).Radius(4).
					Background(ui.Hex("#f59e0b")).Children(func() {
					ui.Text(c, m.Score).FontSize(10.5).FontWeight(800).TextColor(white)
				})

				// Bottom Duration Bar
				ui.Box(c).Height(26).Background(ui.RGBA(0, 0, 0, 0.6)).
					AlignItems(ui.Center).PaddingX(8).Children(func() {
					ui.Text(c, m.Duration).FontSize(11).TextColor(white)
					ui.Spacer(c)
					ui.Text(c, m.Year).FontSize(11).TextColor(ui.RGBA(255, 255, 255, 0.75))
				})
			})

			// Title
			ui.Text(c, m.Title).FontSize(13.5).FontWeight(600).TextColor(p.text).MaxLines(1)

			// Actors / Genre
			ui.Text(c, m.Genre+" · "+m.Actors).FontSize(11.5).TextColor(p.faint).MaxLines(1)
		})
	})
}
