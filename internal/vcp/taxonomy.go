package vcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// DramaTaxonomy defines the multi-tier navigation taxonomy for short dramas.
type DramaTaxonomy struct {
	Types    []string `json:"types"`    // e.g. ["全部", "漫剧", "真人剧", "AI剧", "演员"]
	Rankings []string `json:"rankings"` // e.g. ["推荐榜", "热播榜", "新剧榜", "热搜榜", "追更榜", "必看榜", "收藏榜"]
	Genres   []string `json:"genres"`   // e.g. ["全部", "都市", "爽剧逆袭", "战神", "古装", "甜宠", "悬疑", "穿越", "奇幻"]
}

// CinemaTaxonomy defines the channels and genre tags for movies and cinema.
type CinemaTaxonomy struct {
	Channels []string `json:"channels"` // e.g. ["精选影视", "经典电影", "漫剧", "4K蓝光", "纪录片"]
	Genres   []string `json:"genres"`   // e.g. ["全部", "动作", "喜剧", "爱情", "都市", "悬疑", "古装", "奇幻", "战争", "科幻"]
}

// ShortVideoPreset represents a quick search / crawl preset keyword from the platform pool.
type ShortVideoPreset struct {
	Label    string `json:"label"`
	Keyword  string `json:"keyword"`
	Category string `json:"category"`
}

// ShortVideoPresetGroup represents a category group of keyword presets (aligned with 8901 player).
type ShortVideoPresetGroup struct {
	Name    string             `json:"name"`
	Icon    string             `json:"icon"`
	Presets []ShortVideoPreset `json:"presets"`
}

// ShortVideoTaxonomy defines the categories, tags, preset groups, and algorithm chips for short videos.
type ShortVideoTaxonomy struct {
	Categories   []string                `json:"categories"`    // 归档分类
	PresetGroups []ShortVideoPresetGroup `json:"preset_groups"` // 预设关键词分类组
	Modifiers    []string                `json:"modifiers"`     // 算法流量修饰词
	Durations    []string                `json:"durations"`     // 时长筛选
	HeatLevels   []string                `json:"heat_levels"`   // 热度门槛
	Tags         []string                `json:"tags"`          // 标签
}

// CinemaItem represents a feature film or movie asset.
type CinemaItem struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Channel     string   `json:"channel"`
	Genre       string   `json:"genre"`
	Score       string   `json:"score"`
	Year        string   `json:"year"`
	Duration    string   `json:"duration"`
	Resolution  string   `json:"resolution"` // "4K HDR", "1080P", "蓝光"
	CoverURL    string   `json:"cover_url"`
	Description string   `json:"description"`
	Actors      string   `json:"actors"`
	PlayCount   int64    `json:"play_count"`
	VideoURL    string   `json:"video_url"`
}

var (
	taxonomyMu    sync.RWMutex
	dramaTaxonomy = &DramaTaxonomy{
		Types:    []string{"全部", "漫剧", "真人剧", "AI剧", "演员"},
		Rankings: []string{"推荐榜", "热播榜", "新剧榜", "热搜榜", "追更榜", "必看榜", "收藏榜"},
		Genres:   []string{"全部", "都市", "爽剧逆袭", "战神", "古装", "甜宠", "悬疑", "穿越", "奇幻"},
	}

	cinemaTaxonomy = &CinemaTaxonomy{
		Channels: []string{"精选影视", "经典电影", "漫剧", "4K蓝光", "纪录片"},
		Genres:   []string{"全部", "动作", "喜剧", "爱情", "都市", "悬疑", "古装", "奇幻", "战争", "科幻"},
	}

		shortVideoTaxonomy = &ShortVideoTaxonomy{
			Categories: []string{"全部", "潮流舞蹈", "颜值变装", "秘境探险", "爆笑搞笑", "影视混剪", "极客数码", "时尚模特", "日常美学", "沉浸声学", "健身塑形"},
		PresetGroups: []ShortVideoPresetGroup{
			{
				Name: "热辣颜值穿搭",
				Icon: "🌶️",
				Presets: []ShortVideoPreset{
					{Label: "穿搭试衣 (Try On)", Keyword: "try on haul", Category: "魅惑性感"},
					{Label: "泳装试衣 (Bikini)", Keyword: "bikini try on", Category: "魅惑性感"},
					{Label: "韩团 4K 直拍", Keyword: "kpop fancam 4k", Category: "潮流舞蹈"},
					{Label: "练习室排练", Keyword: "kpop dance practice", Category: "潮流舞蹈"},
					{Label: "瑜伽形体塑形", Keyword: "yoga workout stretch", Category: "健身塑形"},
					{Label: "街拍超模秀", Keyword: "street fashion model", Category: "时尚模特"},
					{Label: "欧美热辣穿搭", Keyword: "curvy fashion model", Category: "魅惑性感"},
					{Label: "4K 漫展变装", Keyword: "cosplay anime 4k", Category: "颜值变装"},
				},
			},
			{
				Name: "极致解压猎奇",
				Icon: "⚡",
				Presets: []ShortVideoPreset{
					{Label: "液压机粉碎一切", Keyword: "hydraulic press", Category: "极限刺激"},
					{Label: "强迫症极致解压", Keyword: "oddly satisfying", Category: "秘境探险"},
					{Label: "高压水枪深度清洗", Keyword: "power washing ASMR", Category: "秘境探险"},
					{Label: "沉浸式旧物翻新", Keyword: "restoration ASMR", Category: "深度解说"},
					{Label: "监控高能抓拍反转", Keyword: "instant karma caught on camera", Category: "爆笑搞笑"},
					{Label: "楼顶跑酷第一视角", Keyword: "parkour pov rooftop", Category: "极限刺激"},
					{Label: "深海未解探险 4K", Keyword: "deep sea exploration 4k", Category: "纪实纪录片"},
					{Label: "全球高能沙雕合集", Keyword: "fail army compilation", Category: "爆笑搞笑"},
				},
			},
			{
				Name: "纪实解说爽片",
				Icon: "📜",
				Presets: []ShortVideoPreset{
					{Label: "真实大案刑侦纪实", Keyword: "true crime documentary", Category: "深度解说"},
					{Label: "真实审讯心理博弈", Keyword: "police interrogation breakdown", Category: "深度解说"},
					{Label: "世界顶级未解之谜", Keyword: "unsolved mysteries world", Category: "深度解说"},
					{Label: "百年废墟城堡探险", Keyword: "abandoned exploration urbex", Category: "秘境探险"},
					{Label: "远古失落文明探索", Keyword: "ancient civilization history", Category: "历史故事"},
					{Label: "重型军事装备大片", Keyword: "military weapons documentary", Category: "纪实纪录片"},
					{Label: "4K 顶级打斗爽片", Keyword: "movie fight scenes 4k", Category: "极限刺激"},
				},
			},
		},
		Modifiers: []string{
			"4K 60fps",
			"compilation",
			"viral",
			"remastered",
			"ASMR",
			"no copyright",
		},
		Durations: []string{
			"全部",
			"竖屏短视频 (<60s)",
			"精选片段 (1~5分钟)",
			"深度中视频 (5~30分钟)",
			"完整大片 (<2小时)",
		},
		HeatLevels: []string{
			"全部",
			"100万+ 爆款",
			"10万+ 热门",
			"1万+ 视频",
		},
		Tags: []string{"全部", "热门", "高能", "治愈", "神同步", "变装", "Vlog", "AI创作", "经典回忆"},
	}

	cinemaMu      sync.RWMutex
	cachedCinemas []*CinemaItem
)

// GetDramaTaxonomy returns the short drama taxonomy defined by the station master.
func GetDramaTaxonomy() *DramaTaxonomy {
	taxonomyMu.RLock()
	defer taxonomyMu.RUnlock()
	return dramaTaxonomy
}

// GetCinemaTaxonomy returns the cinema channels and tags defined by the station master.
func GetCinemaTaxonomy() *CinemaTaxonomy {
	taxonomyMu.RLock()
	defer taxonomyMu.RUnlock()
	return cinemaTaxonomy
}

// GetShortVideoTaxonomy returns the short video categories and tags defined by the station master.
func GetShortVideoTaxonomy() *ShortVideoTaxonomy {
	taxonomyMu.RLock()
	defer taxonomyMu.RUnlock()
	return shortVideoTaxonomy
}

// SyncTaxonomyFromStation attempts to pull custom categories and tags from station master API.
func SyncTaxonomyFromStation(endpoint string) {
	if endpoint == "" {
		endpoint = GetStationEndpoint()
	}
	url := fmt.Sprintf("%s/api/v1/audience/channels", strings.TrimRight(endpoint, "/"))
	resp, err := httpClient.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

		type remoteTaxonomyPayload struct {
			Dramas      *DramaTaxonomy      `json:"dramas"`
			Cinemas     *CinemaTaxonomy     `json:"cinemas"`
			ShortVideos *ShortVideoTaxonomy `json:"short_videos"`
		}

		var payload remoteTaxonomyPayload
		if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
			taxonomyMu.Lock()
			if payload.Dramas != nil && len(payload.Dramas.Types) > 0 {
				dramaTaxonomy = payload.Dramas
			}
			if payload.Cinemas != nil && len(payload.Cinemas.Channels) > 0 {
				cinemaTaxonomy = payload.Cinemas
			}
			if payload.ShortVideos != nil && len(payload.ShortVideos.Categories) > 0 {
				shortVideoTaxonomy = payload.ShortVideos
			}
			taxonomyMu.Unlock()
		}
	}

// GetCinemas returns the list of feature movies.
func GetCinemas() []*CinemaItem {
	cinemaMu.RLock()
	if len(cachedCinemas) > 0 {
		defer cinemaMu.RUnlock()
		return cachedCinemas
	}
	cinemaMu.RUnlock()

	cinemaMu.Lock()
	defer cinemaMu.Unlock()
	if len(cachedCinemas) > 0 {
		return cachedCinemas
	}

	cachedCinemas = defaultCinemaCatalog()
	return cachedCinemas
}

// FilterCinemas filters cinema movies by channel, genre, and search query.
func FilterCinemas(channel, genre, query string) []*CinemaItem {
	all := GetCinemas()
	q := strings.TrimSpace(strings.ToLower(query))

	var res []*CinemaItem
	for _, m := range all {
		if channel != "" && channel != "精选影视" && m.Channel != channel {
			continue
		}
		if genre != "" && genre != "全部" && m.Genre != genre {
			continue
		}
		if q != "" {
			match := strings.Contains(strings.ToLower(m.Title), q) ||
				strings.Contains(strings.ToLower(m.Description), q) ||
				strings.Contains(strings.ToLower(m.Actors), q)
			if !match {
				continue
			}
		}
		res = append(res, m)
	}
	return res
}

func defaultCinemaCatalog() []*CinemaItem {
	return []*CinemaItem{
		{
			ID:          "m-101",
			Title:       "万里归途：深空救援",
			Channel:     "精选影视",
			Genre:       "动作",
			Score:       "9.4",
			Year:        "2026",
			Duration:    "128分钟",
			Resolution:  "4K HDR",
			Description: "近未来星际航行遇险，外交领事官临危受命，跨越数万光年带撤离人员重返地球的惊险传奇。",
			Actors:      "张译 / 王俊凯 / 殷桃",
			PlayCount:   1890000,
		},
		{
			ID:          "m-102",
			Title:       "长安幻夜：青丘龙隐",
			Channel:     "经典电影",
			Genre:       "古装",
			Score:       "9.1",
			Year:        "2025",
			Duration:    "116分钟",
			Resolution:  "4K 杜比",
			Description: "大唐上元夜灯火通明，狄仁杰携大理寺金吾卫破获青丘龙涎迷案，揭开深宫数十年秘闻。",
			Actors:      "刘德华 / 李冰冰 / 梁家辉",
			PlayCount:   1420000,
		},
		{
			ID:          "m-103",
			Title:       "极道追凶：雨夜无声",
			Channel:     "精选影视",
			Genre:       "悬疑",
			Score:       "9.3",
			Year:        "2026",
			Duration:    "110分钟",
			Resolution:  "1080P 超清",
			Description: "一起横跨二十年的陈年旧案突现转机，两位老刑警在风雨如晦的迷雾中寻觅蛛丝马迹。",
			Actors:      "张颂文 / 秦昊 / 辛芷蕾",
			PlayCount:   2150000,
		},
		{
			ID:          "m-104",
			Title:       "跨越山海去爱你",
			Channel:     "精选影视",
			Genre:       "爱情",
			Score:       "8.9",
			Year:        "2026",
			Duration:    "105分钟",
			Resolution:  "4K HDR",
			Description: "从雪域高原到江南水乡，两颗漂泊灵魂在时光列车上的相遇与守护，温暖治愈。",
			Actors:      "许光汉 / 谭松韵",
			PlayCount:   980000,
		},
		{
			ID:          "m-105",
			Title:       "铁血军魂：长征突围",
			Channel:     "经典电影",
			Genre:       "战争",
			Score:       "9.6",
			Year:        "2024",
			Duration:    "135分钟",
			Resolution:  "4K 修复",
			Description: "重温烽火连天的峥嵘岁月，红军先锋团在绝境中飞夺天险、浴血奋战的壮丽史诗。",
			Actors:      "唐国强 / 刘劲 / 王伍福",
			PlayCount:   3200000,
		},
		{
			ID:          "m-106",
			Title:       "疯狂大赢家",
			Channel:     "精选影视",
			Genre:       "喜剧",
			Score:       "8.8",
			Year:        "2025",
			Duration:    "102分钟",
			Resolution:  "1080P 超清",
			Description: "三个小人物误打误撞卷入一场啼笑皆非的商业收购战，笑点密集，反转不断。",
			Actors:      "沈腾 / 马丽 / 艾伦",
			PlayCount:   2780000,
		},
		{
			ID:          "m-107",
			Title:       "赛博重构：2088",
			Channel:     "4K蓝光",
			Genre:       "科幻",
			Score:       "9.5",
			Year:        "2026",
			Duration:    "140分钟",
			Resolution:  "4K 杜比",
			Description: "意识上传时代的深海服务器，人类最后的觉醒者对抗超级AI矩阵的宏大叙事。",
			Actors:      "吴京 / 屈楚萧 / 刘德华",
				PlayCount:   3890000,
			},
		}
	}

// FilterShortVideos filters short-form videos by category, tag, and search query.
func FilterShortVideos(category, tag, query string) []*ShortVideoItem {
	return FilterShortVideosAdvanced(category, tag, query, "", "", "")
}

// FilterShortVideosAdvanced filters short-form videos with category, tag, query, duration, heat level, and sort.
func FilterShortVideosAdvanced(category, tag, query, durationFilter, heatFilter, sortBy string) []*ShortVideoItem {
	all := GetShortVideos()
	q := strings.TrimSpace(strings.ToLower(query))

	var res []*ShortVideoItem
	for _, v := range all {
		// Category check
		if category != "" && category != "全部" && category != "推荐" {
			if !strings.EqualFold(v.Category, category) && !strings.Contains(v.Category, category) {
				continue
			}
		}

		// Tag check
		if tag != "" && tag != "全部" && tag != "热门" {
			matchedTag := false
			for _, t := range v.Tags {
				if strings.Contains(strings.ToLower(t), strings.ToLower(tag)) {
					matchedTag = true
					break
				}
			}
			if !matchedTag && !strings.Contains(strings.ToLower(v.Description), strings.ToLower(tag)) && !strings.Contains(strings.ToLower(v.Title), strings.ToLower(tag)) {
				continue
			}
		}

		// Duration check
		if durationFilter != "" && durationFilter != "全部" {
			sec := v.DurationSec
			switch {
			case strings.Contains(durationFilter, "<60s") || strings.Contains(durationFilter, "竖屏"):
				if sec >= 60 && sec > 0 {
					continue
				}
			case strings.Contains(durationFilter, "1~5分钟") || strings.Contains(durationFilter, "精选片段"):
				if sec < 60 || sec > 300 {
					continue
				}
			case strings.Contains(durationFilter, "5~30分钟") || strings.Contains(durationFilter, "中视频"):
				if sec < 300 || sec > 1800 {
					continue
				}
			case strings.Contains(durationFilter, "<2小时") || strings.Contains(durationFilter, "完整大片"):
				if sec < 1800 {
					continue
				}
			}
		}

		// Heat level check (PlayCount)
		if heatFilter != "" && heatFilter != "全部" {
			switch {
			case strings.Contains(heatFilter, "100万"):
				if v.PlayCount < 1000000 {
					continue
				}
			case strings.Contains(heatFilter, "10万"):
				if v.PlayCount < 100000 {
					continue
				}
			case strings.Contains(heatFilter, "1万"):
				if v.PlayCount < 10000 {
					continue
				}
			}
		}

		// Query check
		if q != "" {
			match := strings.Contains(strings.ToLower(v.Title), q) ||
				strings.Contains(strings.ToLower(v.Description), q) ||
				strings.Contains(strings.ToLower(v.Author), q) ||
				strings.Contains(strings.ToLower(v.Category), q)
			if !match {
				for _, t := range v.Tags {
					if strings.Contains(strings.ToLower(t), q) {
						match = true
						break
					}
				}
			}
			if !match {
				continue
			}
		}

		res = append(res, v)
	}

	// Sort logic
	if sortBy == "heat" || sortBy == "最热" || sortBy == "播放量" {
		sort.Slice(res, func(i, j int) bool {
			return res[i].PlayCount > res[j].PlayCount
		})
	} else if sortBy == "likes" || sortBy == "点赞数" {
		sort.Slice(res, func(i, j int) bool {
			return res[i].LikeCount > res[j].LikeCount
		})
	}

	return res
}
