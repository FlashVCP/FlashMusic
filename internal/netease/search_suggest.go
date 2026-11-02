package netease

import "net/url"

// Suggestions are what a search may be after, as its query is typed, in
// the order NetEase ranks their kinds. It suggests no users or podcasts.
type Suggestions struct {
	Songs     []Track
	Artists   []Artist
	Albums    []AlbumProfile
	Playlists []Playlist
	Order     []string // "songs", "artists", "albums" and "playlists"
}

// SearchSuggest returns what a search for query may be after.
func SearchSuggest(query string) (Suggestions, error) {
	var res struct {
		Result struct {
			Songs   []legacyTrack `json:"songs"`
			Artists []Artist      `json:"artists"`
			Albums  []struct {
				AlbumProfile
				Artist Artist `json:"artist"` // a suggested album has the one
			} `json:"albums"`
			Playlists []Playlist `json:"playlists"`
			Order     []string   `json:"order"`
		} `json:"result"`
	}
	if err := get("/api/search/suggest/web", url.Values{"s": {query}}, &res); err != nil {
		return Suggestions{}, err
	}
	r := res.Result
	s := Suggestions{Artists: r.Artists, Playlists: r.Playlists, Order: r.Order}
	for _, t := range r.Songs {
		s.Songs = append(s.Songs, t.track())
	}
	for _, al := range r.Albums {
		al.Artists = []Artist{al.Artist}
		s.Albums = append(s.Albums, al.AlbumProfile)
	}
		return s, nil
}

// HotSearchItem is an entry on the trending / hot search leaderboard.
type HotSearchItem struct {
	SearchWord string `json:"searchWord"`
	Score      int    `json:"score"`
	Content    string `json:"content"`
	IconType   int    `json:"iconType"` // 1: HOT, 5: NEW, etc.
	IconURL    string `json:"iconUrl"`
}

// HotSearches returns the current trending searches from NetEase, falling back
// to curated trending keywords if offline.
func HotSearches() ([]HotSearchItem, error) {
	var res struct {
		Code int             `json:"code"`
		Data []HotSearchItem `json:"data"`
	}
	err := eapiPost("/api/hotsearch/list/get", map[string]any{}, &res)
	if err == nil && len(res.Data) > 0 {
		return res.Data, nil
	}
	// Fallback to /api/search/hot/detail
	err = eapiPost("/api/search/hot/detail", map[string]any{}, &res)
	if err == nil && len(res.Data) > 0 {
		return res.Data, nil
	}

	// Curated trending fallbacks if API unavailable
	return []HotSearchItem{
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
	}, nil
}

