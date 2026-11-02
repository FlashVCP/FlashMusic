package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// searchResults and searchSuggest search NetEase; tests replace them.
var (
	searchResults = netease.SearchPage
	searchSuggest = netease.SearchSuggest
)

// searchLimit is how many results load at a time.
const searchLimit = 60

// searchKinds are the kinds of search, as their tabs show them.
var searchKinds = []struct {
	kind        netease.SearchType
	label, unit string
}{
	{netease.SearchSongs, "单曲", "首单曲"},
	{netease.SearchArtists, "歌手", "位歌手"},
	{netease.SearchAlbums, "专辑", "张专辑"},
	{netease.SearchPlaylists, "歌单", "个歌单"},
	{netease.SearchPodcasts, "播客", "个播客"},
}

// searchData is what a search found, as far as its pages have loaded.
type searchData struct {
	res         netease.SearchResult
	page        paged
	heroArtist  *netease.ArtistProfile
	heroLoading bool
}

// search searches for q, of the kind last searched for.
func (a *app) search(q string) {
	q = strings.TrimSpace(q)
	if q == "" {
		a.open(route{page: pageSearch, query: "", kind: netease.SearchSongs})
		return
	}
	kind := a.searched.kind
	if kind == 0 {
		kind = netease.SearchSongs
	}
	a.searchFor(route{page: pageSearch, query: q, kind: kind})
}

// searchFor opens the page of a search, remembering its query. Searched
// for again, it loads anew.
func (a *app) searchFor(r route) {
	if r.query != "" {
		a.rememberSearch(r.query)
	}
	a.dismissSuggestions()
	if r == a.route {
		a.nowPlaying = false
		a.loadSearch(r, true)
		return
	}
	a.open(r)
}

// loadSearch loads the first page of a search, unless it has, or again.
func (a *app) loadSearch(r route, again bool) {
	if r.query == "" {
		a.loadHotSearches()
		return
	}
	if a.searches[r] != nil && !again {
		return
	}
	d := &searchData{}
	a.searches[r] = d
	a.moreResults(r, d)
	if r.kind == netease.SearchSongs {
		a.loadHeroArtist(r.query, d)
	}
}

// loadHotSearches loads trending search rankings from NetEase.
func (a *app) loadHotSearches() {
	if len(a.hotSearches) > 0 || a.hotLoading {
		return
	}
	a.hotLoading = true
	go func() {
		hots, err := netease.HotSearches()
		a.post(func() {
			a.hotLoading = false
			if err == nil && len(hots) > 0 {
				a.hotSearches = hots
			}
		})
	}()
}

// loadHeroArtist checks if the search query matches an artist and fetches profile info.
func (a *app) loadHeroArtist(query string, d *searchData) {
	if d.heroLoading || d.heroArtist != nil {
		return
	}
	d.heroLoading = true
	go func() {
		res, err := searchResults(query, netease.SearchArtists, 1, 0)
		if err == nil && len(res.Artists) > 0 {
			arID := res.Artists[0].ID
			detail, err := netease.ArtistDetail(arID)
			a.post(func() {
				d.heroLoading = false
				if err == nil && detail != nil {
					d.heroArtist = detail
				}
			})
			return
		}
		a.post(func() {
			d.heroLoading = false
		})
	}()
}

// moreResults loads the next page of a search.
func (a *app) moreResults(r route, d *searchData) {
	if d.page.loading || r.query == "" {
		return
	}
	d.page.loading, d.page.err = true, ""
	offset := d.res.Len()
	go func() {
		res, err := searchResults(r.query, r.kind, searchLimit, offset)
		a.post(func() {
			d.page.loading = false
			if err != nil {
				d.page.err = "搜索失败，请稍后再试"
				return
			}
			d.res.Append(res)
			d.page.loaded, d.page.more = true, res.Len() > 0 && d.res.Len() < d.res.Total
		})
	}()
}

// searchHistoryLimit is how many searches the history keeps.
const searchHistoryLimit = 20

// rememberSearch puts q first in the history of searches, once.
func (a *app) rememberSearch(q string) {
	history := []string{q}
	for _, s := range a.settings.SearchHistory {
		if s != q && len(history) < searchHistoryLimit {
			history = append(history, s)
		}
	}
	a.settings.SearchHistory = history
	a.settings.save()
}

func (a *app) removeSearchHistory(item string) {
	var next []string
	for _, s := range a.settings.SearchHistory {
		if s != item {
			next = append(next, s)
		}
	}
	a.settings.SearchHistory = next
	a.settings.save()
}

func (a *app) clearSearchHistory() {
	a.settings.SearchHistory = nil
	a.settings.save()
}

func (a *app) searchPage(c *ui.Context) {
	r := a.route

	// Case 1: Empty Query - Alger-style Search Landing Page (热搜列表 + 搜索历史胶囊)
	if strings.TrimSpace(r.query) == "" {
		a.searchLandingPage(c)
		return
	}

	// Case 2: Query Present - Search Results
	d := a.searches[r]
	if d == nil {
		a.loadSearch(r, false)
		d = a.searches[r]
		if d == nil {
			d = &searchData{}
		}
	}

	header := func(c *ui.Context) { a.searchHeader(c, d) }
	more := func() { a.moreResults(r, d) }

	if r.kind == netease.SearchSongs {
		if !d.page.loaded && d.page.loading {
			a.trackSkeleton(c, a.pageShimmer(c), true, header)
			return
		}
		a.trackRows(c, d.res.Songs, true, header, func(c *ui.Context) {
			a.pageFoot(c, &d.page, more)
			a.searchEmpty(c, d)
		})
		return
	}

		a.scrollPage(c, func() {
			header(c)
			res := &d.res
			switch r.kind {
			case netease.SearchArtists:
				ui.Grid(c).Columns(a.columns(c, 132)).GapX(20).GapY(28).Children(func() {
					for i := range res.Artists {
						a.artistCard(c.Key(res.Artists[i].ID), &res.Artists[i])
					}
				})
			case netease.SearchAlbums:
				a.albumGrid(c, res.Albums)
				case netease.SearchPlaylists:
					ui.Grid(c).Columns(a.columns(c, 210)).GapX(20).GapY(28).Children(func() {
						for i := range res.Playlists {
							a.playlistCard(c.Key(res.Playlists[i].ID), &res.Playlists[i])
						}
					})
				case netease.SearchUsers:
					for i := range res.Users {
						a.userRow(c.Key(res.Users[i].UserID), &res.Users[i])
					}
				case netease.SearchPodcasts:
					for i := range res.Podcasts {
						a.podcastRow(c.Key(res.Podcasts[i].ID), &res.Podcasts[i])
					}
				}
			if !d.page.loaded && d.page.loading {
				a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 210), 210, false)
				return
			}
		a.pageFoot(c, &d.page, more)
		a.searchEmpty(c, d)
	})
}

// defaultPlaylistCategories matches the full category tags from AlgerMusicPlayer & NetEase catalogue.
var defaultPlaylistCategories = []string{
	"每日推荐", "全部", "华语", "欧美", "日韩", "流行", "摇滚", "民谣", "电子", "说唱",
	"轻音乐", "爵士", "乡村", "R&B/Soul", "古典", "民族", "古风", "ACG", "影视原声",
	"治愈", "怀旧", "清新", "浪漫", "放松", "孤独", "感动", "兴奋", "快乐", "安静", "思念",
	"清晨", "夜晚", "学习", "工作", "午休", "下午茶", "地铁", "驾车", "运动", "旅行", "散步", "酒吧",
}

func (a *app) loadPlaylistCategories() {
	if len(a.searchCatList) > 0 {
		return
	}
	a.searchCatList = defaultPlaylistCategories
	go func() {
		cats, err := netease.PlaylistCatalogue()
		if err == nil && len(cats) > 0 {
			full := make([]string, 0, len(cats)+2)
			full = append(full, "每日推荐", "全部")
			for _, cat := range cats {
				if cat != "全部" && cat != "每日推荐" {
					full = append(full, cat)
				}
			}
			a.post(func() {
				a.searchCatList = full
			})
		}
	}()
}

func (a *app) loadCategoryPlaylists(cat string) {
	if cat == "每日推荐" || cat == "" {
		a.loadHotSearches()
		return
	}
	if len(a.catPlaylists[cat]) > 0 {
		return
	}
	a.catLoading = true
	a.catErr = ""
	go func() {
		pls, _, err := netease.PlaylistsByCat(cat, "hot", 42, 0)
		a.post(func() {
			a.catLoading = false
			if err != nil {
				a.catErr = "加载歌单失败，请检查网络"
			} else {
				a.catPlaylists[cat] = pls
			}
		})
	}()
}

// searchCategorySelector renders the AlgerMusicPlayer CategorySelector horizontal scrolling pill tabs.
func (a *app) searchCategorySelector(c *ui.Context) {
	p := a.pal
	a.loadPlaylistCategories()

	cats := a.searchCatList
	if len(cats) == 0 {
		cats = defaultPlaylistCategories
	}

	ui.Box(c.Key("cat-selector-container")).Margin(0, 0, 18, 0).Children(func() {
		ui.ScrollHorizontal(c.Key("cat-scroll")).TrackScroll(&a.catScroll).Height(40).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				for _, cat := range cats {
					name := cat
					active := a.searchCat == name || (a.searchCat == "" && name == "每日推荐")
					pill := ui.ButtonBase(c.Key("cat-pill-" + name)).Height(32).PaddingX(16).Radius(999).
						AlignItems(ui.Center).FocusRing(false).Label(name)
					if active {
						pill.Background(p.accent).TextColor(p.onAccent)
						pill.Shadow(0, 4, 12, 0, p.accent.Alpha(0.35))
					} else {
						pill.Background(p.hover).TextColor(p.muted)
						if pill.Hovered() {
							pill.Background(p.press).TextColor(p.text)
						}
					}
					pill.Children(func() {
						weight := 500
						if active {
							weight = 700
						}
						ui.Text(c, name).FontSize(13).FontWeight(weight)
					})
					if pill.Clicked() && !active {
						a.searchCat = name
						a.loadCategoryPlaylists(name)
					}
				}
			})
		})
	})
}

// searchLandingPage renders the Alger-style discovery screen (Figure 2 & views/search/index.vue):
// 1. Top Category Navigation Bar (AlgerMusicPlayer Sticky CategorySelector)
// 2. Dynamic switching between "每日推荐" (Hot searches & trending history) and category playlist showcases.
func (a *app) searchLandingPage(c *ui.Context) {
	p := a.pal
	a.loadHotSearches()

	a.scrollPage(c, func() {
		// 1. Alger Category Selector Horizontal Bar
		a.searchCategorySelector(c)

		// 2. Case A: '每日推荐' - Show Hot Search Leaderboard & Search History
		if a.searchCat == "" || a.searchCat == "每日推荐" {
			// Large Landing Search Header (views/search/index.vue: h1 text-3xl md:text-4xl font-bold tracking-tight)
			ui.Column(c).Gap(6).Margin(4, 0, 24, 0).Children(func() {
				ui.Text(c, "热搜列表").FontSize(30).FontWeight(800).LetterSpacing(-0.8).TextColor(p.text)
				ui.Text(c, "探索当下最热门的搜索趋势 · 潮流热榜随时畅听").FontSize(13.5).TextColor(p.muted)
			})

			// Section 1: Search History (views/search/index.vue: searchHistory.length > 0)
			if len(a.settings.SearchHistory) > 0 {
				ui.Row(c).AlignItems(ui.Center).Margin(0, 0, 12, 0).Children(func() {
					ui.Text(c, "搜索历史").FontSize(18).FontWeight(700).LetterSpacing(-0.3).TextColor(p.text)
					ui.Spacer(c)
					clearBtn := ui.ButtonBase(c.Key("clear-hist-btn")).Padding(4, 8).Radius(6).Children(func() {
						ui.Text(c, "清空").FontSize(12).TextColor(p.muted)
					})
					if clearBtn.Hovered() {
						clearBtn.Background(p.hover)
					}
					if clearBtn.Clicked() {
						a.clearSearchHistory()
					}
				})

				// Capsule Pill Row (views/search/index.vue: px-4 py-1.5 rounded-full bg-neutral-100 dark:bg-neutral-900 text-sm)
				ui.Row(c).Gap(10).Margin(0, 0, 28, 0).AlignItems(ui.Center).Children(func() {
					for _, h := range a.settings.SearchHistory {
						val := h
						pill := ui.Row(c.Key("hist-pill-"+val)).Height(32).Padding(0, 8, 0, 14).Gap(6).Radius(999).
							Background(p.card).Border(1, p.hairline).AlignItems(ui.Center).FocusRing(false)
						if pill.Hovered() {
							pill.Background(p.hover).Border(1, p.accent.Alpha(0.4))
						}
						pill.Children(func() {
							txt := ui.Text(c, val).FontSize(13).FontWeight(500).TextColor(p.text).SingleLine()
							if pill.Clicked() {
								a.query = val
								a.search(val)
							}
							_ = txt
							// Close 'x' icon to delete this single history tag
							del := iconButton(c.Key("del-"+val), iconX, 11, p.faint, "删除").Size(18, 18)
							if del.Clicked() {
								a.removeSearchHistory(val)
							}
						})
					}
				})
			}

			// Section 2: Hot Search Leaderboard (热搜列表 - 2-column grid, rounded-2xl cards)
			hots := a.hotSearches
			if len(hots) == 0 {
				hots = []netease.HotSearchItem{
					{SearchWord: "晴天", Score: 1852000, Content: "周杰伦 经典校园怀旧之作", IconType: 1},
					{SearchWord: "稻香", Score: 1542000, Content: "回到最初的美好 与童年握手言和", IconType: 1},
					{SearchWord: "告白气球", Score: 1320000, Content: "塞纳河畔 左岸的咖啡", IconType: 1},
					{SearchWord: "富士山下", Score: 1105000, Content: "陈奕迅 粤语巅峰叙事曲", IconType: 0},
					{SearchWord: "罗刹海市", Score: 980000, Content: "刀郎 现象级民谣讽喻大作", IconType: 1},
					{SearchWord: "青花瓷", Score: 870000, Content: "天青色等烟雨 而我在等你", IconType: 0},
					{SearchWord: "七里香", Score: 810000, Content: "秋刀鱼的滋味 猫跟你都想了解", IconType: 0},
					{SearchWord: "泡沫", Score: 760000, Content: "邓紫棋 全网刷屏高音名场面", IconType: 0},
					{SearchWord: "江南", Score: 710000, Content: "林俊杰 千禧年中国风里程碑", IconType: 0},
					{SearchWord: "奢香夫人", Score: 680000, Content: "凤凰传奇 掀起全网合唱狂潮", IconType: 5},
					{SearchWord: "如愿", Score: 640000, Content: "王菲 献给父辈的一曲颂歌", IconType: 0},
					{SearchWord: "乌梅子酱", Score: 590000, Content: "李荣浩 甜度拉满的心动旋律", IconType: 5},
				}
			}

			// 2-column grid for hot search ranks (views/search/index.vue: p-4 rounded-2xl bg-neutral-50 dark:bg-neutral-900/50)
			ui.Grid(c.Key("hot-search-grid")).Columns(2).GapX(20).GapY(12).Children(func() {
				for i, item := range hots {
					rank := i + 1
					word := item.SearchWord
					card := ui.ButtonBase(c.Key(fmt.Sprintf("hot-%d-%s", rank, word))).Height(62).Padding(10, 16, 10, 16).
						Radius(16).Background(p.card).Border(1, p.hairline).AlignItems(ui.Center).Gap(14).FocusRing(false)
					hovered := card.Hovered()
					if hovered {
						card.Background(p.hover).Border(1, p.accent.Alpha(0.4)).Shadow(0, 4, 16, -2, p.cardShade)
					}
					card.Children(func() {
						// Rank Number: w-8 text-lg font-bold italic (index < 3: text-primary, else text-neutral-300 dark:text-neutral-700)
						numColor := p.faint
						if rank <= 3 {
							numColor = p.accent
						}
						ui.Textf(c, "%02d", rank).FontSize(17).FontWeight(800).TextColor(numColor).Width(32)

						// Word & Description (text-sm font-semibold truncate, mt-0.5 text-xs text-neutral-400)
						ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(3).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								wordColor := p.text
								if hovered {
									wordColor = p.accent
								}
								ui.Text(c, word).FontSize(14).FontWeight(600).TextColor(wordColor).SingleLine()
								if item.IconType == 1 {
									ui.Text(c, "HOT").FontSize(9.5).FontWeight(800).TextColor(ui.RGB(239, 68, 68)).
										Padding(1, 5).Radius(4).Background(ui.RGB(239, 68, 68).Alpha(0.12))
								} else if item.IconType == 5 {
									ui.Text(c, "NEW").FontSize(9.5).FontWeight(800).TextColor(p.accent).
										Padding(1, 5).Radius(4).Background(p.accent.Alpha(0.12))
								}
							})
							if item.Content != "" {
								ui.Text(c, item.Content).FontSize(12).TextColor(p.muted).SingleLine()
							}
						})

						// Score / Heat
						if item.Score > 0 {
							ui.Textf(c, "%d", item.Score).FontSize(11.5).FontFeatures("tnum").TextColor(p.faint)
						}
					})
					if card.Clicked() {
						a.query = word
						a.search(word)
					}
				}
			})
			return
		}

		// 3. Case B: Category Selected - Render Tencent 6-Column 1:1 Square Card Grid
		if len(a.catPlaylists[a.searchCat]) == 0 && !a.catLoading && a.catErr == "" {
			a.loadCategoryPlaylists(a.searchCat)
		}

		ui.Column(c).Gap(4).Margin(4, 0, 22, 0).Children(func() {
			ui.Row(c).AlignItems(ui.Baseline).Gap(10).Children(func() {
				ui.Text(c, a.searchCat).FontSize(28).FontWeight(800).LetterSpacing(-0.6).TextColor(p.text)
				ui.Text(c, "官方热门分类歌单").FontSize(14).FontWeight(600).TextColor(p.muted)
			})
			ui.Text(c, "精选官方高保真分类歌单 · 点击即可播放").FontSize(13).TextColor(p.muted)
		})

		pls := a.catPlaylists[a.searchCat]
		if len(pls) == 0 && a.catLoading {
			a.playlistCardsSkeleton(c, a.pageShimmer(c), a.columns(c, 210)*3, 210, false)
			return
		}

		if len(pls) == 0 && a.catErr != "" {
			ui.Box(c).Height(180).Center().Children(func() {
				ui.Column(c).AlignItems(ui.Center).Gap(12).Children(func() {
					ui.Text(c, a.catErr).FontSize(14).TextColor(p.muted)
					btn := pillButton(c.Key("cat-retry"), iconReload, "重试加载", p.hover, p.text)
					if btn.Clicked() {
						a.loadCategoryPlaylists(a.searchCat)
					}
				})
			})
			return
		}

		cols := a.columns(c, 210)
		ui.Grid(c.Key("cat-playlists-grid-" + a.searchCat)).Columns(cols).GapX(20).GapY(28).Children(func() {
			for i := range pls {
				a.playlistCard(c.Key(pls[i].ID), &pls[i])
			}
		})
	})
}

func (a *app) searchEmpty(c *ui.Context, d *searchData) {
	if d.page.loaded && d.res.Len() == 0 {
		a.empty(c, "没有找到相关结果")
	}
}

// searchHeader renders the results header, category tabs, and Alger-style action bar (Figure 3 & views/search/SearchResult.vue).
func (a *app) searchHeader(c *ui.Context, d *searchData) {
	p := a.pal
	r := a.route
	sub := "搜索"
	for _, k := range searchKinds {
		if k.kind == r.kind {
			if d.page.loaded {
				sub = fmt.Sprintf("找到 %d %s", d.res.Total, k.unit)
			}
		}
	}

	// 1. Alger Hero Card for Artists (Figure 3 top layout)
	if r.kind == netease.SearchSongs && d != nil && d.heroArtist != nil {
		a.searchArtistHero(c, d.heroArtist)
	}

	// 2. Query Title & Result Subtitle (views/search/SearchResult.vue: text-3xl md:text-4xl font-bold tracking-tight)
	ui.Column(c).Gap(4).Margin(4, 0, 18, 0).Children(func() {
		ui.Text(c, "“"+r.query+"”").FontSize(32).FontWeight(800).LetterSpacing(-0.8).TextColor(p.text).MaxLines(1)
		ui.Text(c, sub).FontSize(13.5).TextColor(p.muted)
	})

	// 3. Category Capsule Tabs (views/search/SearchResult.vue: px-4 py-2 rounded-full text-sm font-medium)
	// Active: bg-primary text-white shadow-lg shadow-primary/25
	// Inactive: bg-neutral-100 dark:bg-neutral-900 text-neutral-600 dark:text-neutral-400
	ui.Row(c).Gap(8).Margin(0, 0, 18, 0).AlignItems(ui.Center).Children(func() {
		for _, k := range searchKinds {
			kind := k.kind
			label := k.label
			active := r.kind == kind
			btn := ui.ButtonBase(c.Key("search-tab-"+label)).Height(34).Padding(0, 16, 0, 16).
				Radius(999).AlignItems(ui.Center).FocusRing(false)
			if active {
				btn.Background(p.accent).Shadow(0, 4, 14, 0, p.accent.Alpha(0.35))
			} else {
				btn.Background(p.card).Border(1, p.hairline)
				if btn.Hovered() {
					btn.Background(p.hover)
				}
			}
			btn.Children(func() {
				fg := p.muted
				if active {
					fg = p.onAccent
				}
				ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(fg).SingleLine()
			})
			if btn.Clicked() && !active {
				r.kind = kind
				a.show(r)
			}
		}
	})

	// 4. Action Bar (views/search/SearchResult.vue: Play All Button px-6 py-2.5 rounded-full bg-primary text-white)
	if r.kind == netease.SearchSongs && len(d.res.Songs) > 0 {
		songs := d.res.Songs
		ui.Row(c).Gap(12).AlignItems(ui.Center).Margin(0, 0, 16, 0).Children(func() {
			// Glowing Primary Play All Button with Alger green shadow
			playAll := ui.ButtonBase(c.Key("search-play-all")).Height(36).Padding(0, 20, 0, 20).Gap(8).
				Radius(999).Background(p.accent).TextColor(p.onAccent).AlignItems(ui.Center).
				Shadow(0, 4, 16, 0, p.accent.Alpha(0.35)).FocusRing(false)
			if playAll.Hovered() {
				playAll.Background(p.accent.Mix(ui.RGB(255, 255, 255), 0.15))
			}
			playAll.Children(func() {
				ui.Icon(c, iconPlay).Size(15, 15)
				ui.Text(c, "播放全部").FontSize(13.5).FontWeight(700).SingleLine()
			})
			if playAll.Clicked() {
				if a.pl.Shuffle() {
					a.pl.ToggleShuffle()
				}
				a.pl.PlayQueue(songs, firstPlayable(songs))
			}

			// Shuffle Play Button
			shuffleBtn := ui.ButtonBase(c.Key("search-shuffle-all")).Height(36).Padding(0, 16, 0, 16).Gap(6).
				Radius(999).Background(p.card).Border(1, p.hairline).TextColor(p.text).AlignItems(ui.Center).FocusRing(false)
			if shuffleBtn.Hovered() {
				shuffleBtn.Background(p.hover)
			}
			shuffleBtn.Children(func() {
				ui.Icon(c, iconShuffle).Size(14, 14).TextColor(p.muted)
				ui.Text(c, "随机播放").FontSize(13).FontWeight(600).SingleLine()
			})
			if shuffleBtn.Clicked() {
				if !a.pl.Shuffle() {
					a.pl.ToggleShuffle()
				}
				a.pl.PlayQueue(songs, firstPlayable(songs))
			}

			ui.Spacer(c)

			// Total Track Count Tag
			ui.Textf(c, "共 %d 首单曲", len(songs)).FontSize(12.5).TextColor(p.muted)
		})
	}
}

// searchArtistHero replicates the artist hero banner in Alger Figure 3 & views/artist/detail.vue:
// Round avatar with ring, bold artist name, '热门 50 单曲' capsule badge, album count, bio, and direct link.
func (a *app) searchArtistHero(c *ui.Context, ar *netease.ArtistProfile) {
	p := a.pal
	card := ui.Row(c.Key("search-artist-hero")).Height(136).Padding(16, 20, 16, 20).Gap(22).Radius(16).
		Background(p.card).Border(1, p.hairline).Shadow(0, 8, 24, -4, p.cardShade).
		AlignItems(ui.Center).Margin(0, 0, 24, 0)

	card.Children(func() {
		// Round Avatar with Alger ring (views/artist/detail.vue: w-36 h-36 rounded-full ring-4 shadow-2xl)
		photo := ar.Avatar
		if photo == "" {
			photo = ar.PicURL
		}
		ui.Box(c).Size(104, 104).Radius(52).Border(3, p.hairline).Shadow(0, 8, 24, -4, p.cardShade).Children(func() {
			a.coverFill(c, photo, 320, 52)
		})

		// Bio Details Column
		ui.Column(c).Grow(1).Shrink(1).MinWidth(0).Gap(6).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, ar.Name).FontSize(22).FontWeight(800).TextColor(p.text)
				// Capsule Tag: 热门 50 单曲 (bg-primary/10 text-primary rounded-full)
				ui.Box(c).Padding(3, 10, 3, 10).Radius(999).Background(p.accent.Alpha(0.12)).Children(func() {
					ui.Text(c, ar.Name+" 热门 50 单曲").FontSize(11).FontWeight(700).TextColor(p.accent)
				})
				if ar.AlbumSize > 0 {
					ui.Textf(c, "%d 张专辑", ar.AlbumSize).FontSize(12).TextColor(p.muted)
				}
			})

			// Biography text snippet
			if d := strings.TrimSpace(ar.BriefDesc); d != "" {
				desc := strings.ReplaceAll(d, "\n", " ")
				ui.Text(c, desc).FontSize(12.5).TextColor(p.muted).LineHeight(1.45).MaxLines(2)
			}

			// View Artist Page Link
			ui.Row(c).Gap(8).AlignItems(ui.Center).Margin(4, 0, 0, 0).Children(func() {
				linkBtn := ui.ButtonBase(c.Key("hero-ar-link")).Padding(2, 10).Radius(6).Gap(4).AlignItems(ui.Center).FocusRing(false)
				if linkBtn.Hovered() {
					linkBtn.Background(p.hover)
				}
				linkBtn.Children(func() {
					ui.Text(c, "查看歌手详情与全部专辑 →").FontSize(12.5).FontWeight(600).TextColor(p.accent)
				})
				if linkBtn.Clicked() {
					a.open(route{page: pageArtist, id: ar.ID})
				}
			})
		})
	})
}

// artistCard is an artist's round photo over their name, opening their page.
func (a *app) artistCard(c *ui.Context, ar *netease.ArtistProfile) {
	p := a.pal
	card := ui.ButtonBase(c).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(ar.Name)
	hovered := card.Hovered()
	card.Children(func() {
		photo := ar.Avatar
		if photo == "" {
			photo = ar.PicURL
		}
		art := ui.Box(c).AspectRatio(1).Radius(999)
		if hovered {
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.Children(func() { a.coverFill(c, photo, 400, 999) })
		ui.Text(c, ar.Name).FontSize(13).FontWeight(500).SingleLine().AlignSelf(ui.Center)
	})
	if card.Clicked() {
		a.open(route{page: pageArtist, id: ar.ID})
	}
}
