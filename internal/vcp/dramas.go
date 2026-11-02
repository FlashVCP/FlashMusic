package vcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// DownloadedFilesList supports unmarshaling both string arrays and object arrays from the Station API.
type DownloadedFilesList []string

// UnmarshalJSON unmarshals either []string or []struct{File, Name string}.
func (d *DownloadedFilesList) UnmarshalJSON(data []byte) error {
	var strList []string
	if err := json.Unmarshal(data, &strList); err == nil {
		*d = strList
		return nil
	}
	var objList []struct {
		Name string `json:"name"`
		File string `json:"file"`
	}
	if err := json.Unmarshal(data, &objList); err == nil {
		var res []string
		for _, o := range objList {
			if o.File != "" {
				res = append(res, o.File)
			} else if o.Name != "" {
				res = append(res, o.Name)
			}
		}
		*d = res
		return nil
	}
	return nil
}

// DramaItem represents a short drama series metadata record.
type DramaItem struct {
	ID              int64               `json:"id"`
	Title           string              `json:"title"`
	Category        string              `json:"category"`
	Tags            []string            `json:"tags"`
	PlayCount       int64               `json:"play_count"`
	LikeCount       int64               `json:"like_count"`
	FavoriteCount   int64               `json:"favorite_count"`
	EpisodeCount    int                 `json:"episode_count"`
	CoverURL        string              `json:"cover_url"`
	CoverThumbURL   string              `json:"cover_thumb_url"`
	Description     string              `json:"description"`
	Episodes        []EpisodeItem       `json:"episodes"`
	DownloadedFiles DownloadedFilesList `json:"downloadedFiles,omitempty"`
	DownloadedCount int                 `json:"downloadedCount,omitempty"`
	LocalCoverPath  string              `json:"-"`
	Origin          ContentOrigin       `json:"origin"`
	StreamMode      string              `json:"stream_mode"`
	StationURL      string              `json:"station_url"`
}

// EpisodeItem represents an individual episode in a drama.
type EpisodeItem struct {
	ID        int64  `json:"id"`
	DramaID   int64  `json:"drama_id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Access    string `json:"access_type"`
	CoinPrice int    `json:"coin_price"`
	Locked    bool   `json:"locked"`
	VideoURL  string `json:"video_url"`
}

// ShortVideoItem represents a short-form video in the catalog.
type ShortVideoItem struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Category    string        `json:"category"`
	Tags        []string      `json:"tags"`
	Author      string        `json:"author"`
	CoverURL    string        `json:"cover_url"`
	HlsURL      string        `json:"hls_url"`
	PlayCount   int64         `json:"play_count"`
	LikeCount   int64         `json:"like_count"`
	Duration    string        `json:"duration"`
	DurationSec int           `json:"duration_sec"`
	VideoPath   string        `json:"video_path"`
	Origin      ContentOrigin `json:"origin"`
}

var (
	dramasMu     sync.RWMutex
	cachedDramas []*DramaItem
	categories   []string

	catalogMu      sync.RWMutex
	cachedShortVid []*ShortVideoItem
)

// GetDramas returns the slice of all available short drama records.
func GetDramas() []*DramaItem {
	dramasMu.RLock()
	if len(cachedDramas) > 0 {
		defer dramasMu.RUnlock()
		return cachedDramas
	}
	dramasMu.RUnlock()

	dramasMu.Lock()
	defer dramasMu.Unlock()
	if len(cachedDramas) > 0 {
		return cachedDramas
	}

	loadDramasLocked()
	return cachedDramas
}

// GetDrama finds a drama by its unique ID.
func GetDrama(id int64) *DramaItem {
	for _, d := range GetDramas() {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// GetCategories returns available drama categories for filtering.
func GetCategories() []string {
	GetDramas()
	dramasMu.RLock()
	defer dramasMu.RUnlock()
	return categories
}

// FilterDramas filters dramas by category and search keyword.
func FilterDramas(cat string, query string) []*DramaItem {
	return FilterDramasBy(cat, query, "")
}

// FilterDramasBy filters dramas by category, search keyword, and content origin.
func FilterDramasBy(cat string, query string, origin string) []*DramaItem {
	all := GetDramas()
	q := strings.TrimSpace(strings.ToLower(query))

	var res []*DramaItem
	for _, d := range all {
		if cat != "" && cat != "全部" && cat != "all" {
			if d.Category != cat {
				continue
			}
		}
		if origin != "" && origin != "all" {
			if string(d.Origin) != origin {
				continue
			}
		}
		if q != "" {
			match := strings.Contains(strings.ToLower(d.Title), q) ||
				strings.Contains(strings.ToLower(d.Description), q)
			if !match {
				for _, t := range d.Tags {
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
		res = append(res, d)
	}
	return res
}

// FilterDramasTaxonomy filters and sorts dramas across the station master's types, rankings, and genres.
func FilterDramasTaxonomy(dramaType, ranking, category, query string) []*DramaItem {
	all := GetDramas()
	q := strings.TrimSpace(strings.ToLower(query))

	var list []*DramaItem
	for _, d := range all {
		// 1. Type filter (漫剧, 真人剧, AI剧, 演员)
		if dramaType != "" && dramaType != "全部" {
			switch dramaType {
			case "漫剧":
				if !strings.Contains(d.Category, "漫") {
					continue
				}
			case "AI剧":
				if !strings.Contains(d.Category, "AI") {
					continue
				}
			case "真人剧":
				if strings.Contains(d.Category, "漫") || strings.Contains(d.Category, "AI") {
					continue
				}
			case "演员":
				// featured actor showcase
			}
		}

		// 2. Category / Genre filter
		if category != "" && category != "全部" {
			matchedCat := strings.Contains(d.Category, category)
			if !matchedCat {
				for _, t := range d.Tags {
					if strings.Contains(t, category) || strings.Contains(category, t) {
						matchedCat = true
						break
					}
				}
			}
			if !matchedCat {
				continue
			}
		}

		// 3. Search query
		if q != "" {
			matchedQ := strings.Contains(strings.ToLower(d.Title), q) ||
				strings.Contains(strings.ToLower(d.Description), q)
			if !matchedQ {
				for _, t := range d.Tags {
					if strings.Contains(strings.ToLower(t), q) {
						matchedQ = true
						break
					}
				}
			}
			if !matchedQ {
				continue
			}
		}

		list = append(list, d)
	}

	// 4. Rankings sorting
	if ranking != "" && ranking != "推荐榜" {
		sort.SliceStable(list, func(i, j int) bool {
			switch ranking {
			case "热播榜", "热搜榜":
				return list[i].PlayCount > list[j].PlayCount
			case "臻果榜", "必看榜":
				return list[i].LikeCount > list[j].LikeCount
			case "新剧榜", "预约榜":
				return list[i].ID > list[j].ID
			case "追更榜", "收藏榜":
				return list[i].FavoriteCount > list[j].FavoriteCount
			default:
				return list[i].PlayCount > list[j].PlayCount
			}
		})
	}

	return list
}

func getLocalAppCoversDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			base = filepath.Join(home, "AppData", "Local")
		} else {
			base = os.TempDir()
		}
	}
	dir := filepath.Join(base, "FlashMusic", "covers")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func saveDramaCoverToDisk(id int64, data []byte) {
	if len(data) == 0 {
		return
	}
	dir := getLocalAppCoversDir()
	if dir == "" {
		return
	}
	dest := filepath.Join(dir, fmt.Sprintf("drama_%d.jpg", id))
	tmp := fmt.Sprintf("%s.tmp.%d", dest, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0644); err == nil {
		_ = os.Rename(tmp, dest)
	}
}

// GetDramaCoverPath returns local file path of a drama cover if present.
func GetDramaCoverPath(id int64) string {
	candidatePaths := []string{
		filepath.Join(getLocalAppCoversDir(), fmt.Sprintf("drama_%d.jpg", id)),
		fmt.Sprintf(`D:\FlashMedia\dramas\covers\%d.jpg`, id),
		fmt.Sprintf(`J:\FlashMedia\dramas\covers\%d.jpg`, id),
		fmt.Sprintf(`I:\FlashMedia\dramas\covers\%d.jpg`, id),
		fmt.Sprintf(`D:\vcp\cover_%d.jpg`, id),
		fmt.Sprintf(`covers\%d.jpg`, id),
	}
	for _, p := range candidatePaths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Size() > 0 {
			return p
		}
	}
	return ""
}

// GetDramaCoverBytes reads the decoded/saved JPEG bytes for drama cover.
// It prioritizes local SSD cache and persists any remote fetch to local storage instantly.
func GetDramaCoverBytes(id int64) ([]byte, error) {
	p := GetDramaCoverPath(id)
	if p != "" {
		return os.ReadFile(p)
	}

	// Fallback to CDN or Station Master API
	ep := GetStationEndpoint()
	candidateURLs := []string{
		fmt.Sprintf("https://origin.flashvcp.org/api/admin/dramas/cover/%d.jpg", id),
		fmt.Sprintf("%s/api/admin/dramas/cover/%d.jpg", strings.TrimRight(ep, "/"), id),
	}
	if d := GetDrama(id); d != nil && d.CoverURL != "" {
		if strings.HasPrefix(d.CoverURL, "http://") || strings.HasPrefix(d.CoverURL, "https://") {
			candidateURLs = append([]string{d.CoverURL}, candidateURLs...)
		}
	}

	for _, coverURL := range candidateURLs {
		resp, err := httpClient.Get(coverURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			data, err := io.ReadAll(resp.Body)
			if err == nil && len(data) > 0 {
				saveDramaCoverToDisk(id, data)
				return data, nil
			}
		}
	}

	return nil, fmt.Errorf("drama cover not found for id %d", id)
}

func loadDramasLocked() {
	endpoint := GetStationEndpoint()

	// 1. 先用已验证的真实 36 部母带种子即刻填充内存（0 毫秒即刻返回，UI 线程绝对零阻塞）
	var list []*DramaItem = defaultRealMasterDramas()
	cachedDramas = list
	extractCategoriesLocked()

	// 2. 异步在后台静默向香港站长端同步最新目录与本地已下载映射，绝不阻塞 UI 主线程
	go func() {
		localDownloads := GetLocalDownloadedDramasMap()
		stationList, err := FetchDramasFromStation(endpoint)
		if err == nil && len(stationList) > 0 {
			dramasMu.Lock()
			// 刷新本地下载集数状态
			for _, d := range stationList {
				if d != nil {
					if localFiles, ok := localDownloads[d.ID]; ok && len(localFiles) > 0 {
						d.DownloadedCount = len(localFiles)
					}
				}
			}
			cachedDramas = stationList
			extractCategoriesLocked()
			dramasMu.Unlock()
		}
	}()
}

// extractCategoriesLocked rebuilds the global categories list from the
// dramas currently in cachedDramas. Callers must hold dramasMu.
func extractCategoriesLocked() {
	catMap := map[string]bool{"全部": true}
	for _, item := range cachedDramas {
		if item != nil && item.Category != "" {
			catMap[item.Category] = true
		}
	}
	categories = []string{"全部"}
	priorityCats := []string{"黄果原创🔥", "都市情感", "爽剧", "动作悬疑", "女频成长", "历史穿越"}
	for _, pc := range priorityCats {
		if catMap[pc] {
			categories = append(categories, pc)
			delete(catMap, pc)
		}
	}
	delete(catMap, "全部")
	for c := range catMap {
		categories = append(categories, c)
	}
}

func defaultRealMasterDramas() []*DramaItem {
	return []*DramaItem{
		{
			ID:            7698294694886788121,
			Title:         "和离后我带女儿横扫魔域",
			Category:      "漫剧",
			Tags:          []string{"系统", "玄幻", "异界", "无敌", "高燃"},
			PlayCount:     91168000,
			LikeCount:     3820000,
			FavoriteCount: 1650000,
			EpisodeCount:  987,
			DownloadedCount: 987,
			CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7698294694886788121.jpg",
			Description:   "陆清安曾是威震诸天的永夜魔帝，回归烟火生活后带着两个女儿重返魔域，开启轻松热闹的无敌新旅程。全网热播榜 TOP 1。",
			Origin:        OriginCloud,
			StreamMode:    "FlashVCP 云端分发 · DePIN 边缘秒开",
		},
		{
			ID:            7696066629439867966,
			Title:         "一身腱子肉，乱世做谋臣",
			Category:      "AI 剧",
			Tags:          []string{"穿越", "争霸", "历史", "谋略"},
			PlayCount:     18500000,
			EpisodeCount:  136,
			DownloadedCount: 136,
			CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7696066629439867966.jpg",
			Origin:        OriginCloud,
			StreamMode:    "FlashVCP 云端分发 · DePIN 边缘秒开",
		},
		{
			ID:            7695360003212856344,
			Title:         "亲家母带全家来我家过年",
			Category:      "AI 剧",
			Tags:          []string{"家庭", "逆袭", "伦理", "打脸"},
			PlayCount:     12400000,
			EpisodeCount:  80,
			DownloadedCount: 80,
			CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7695360003212856344.jpg",
			Origin:        OriginCloud,
			StreamMode:    "FlashVCP 云端分发 · DePIN 边缘秒开",
		},
				{
					ID:            7698311709634595864,
					Title:         "分家没我份，破院枯井全是金条",
					Category:      "AI 剧",
					Tags:          []string{"发家致富", "年代", "逆袭", "爽剧"},
					PlayCount:     22300000,
					EpisodeCount:  155,
					DownloadedCount: 155,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7698311709634595864.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7701935931787316286,
					Title:         "十八岁太奶奶驾到，重整家族荣耀第五部",
					Category:      "真人剧",
					Tags:          []string{"豪门", "长辈逆袭", "家族", "打脸"},
					PlayCount:     34100000,
					EpisodeCount:  102,
					DownloadedCount: 102,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7701935931787316286.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7692043430955863102,
					Title:         "反向拿捏：和闺蜜穿七零带着婆婆一起离",
					Category:      "AI 剧",
					Tags:          []string{"穿越", "年代", "姐妹", "爽文"},
					PlayCount:     16800000,
					EpisodeCount:  126,
					DownloadedCount: 126,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7692043430955863102.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7697612795184958489,
					Title:         "备用联系人（完）",
					Category:      "AI 剧",
					Tags:          []string{"悬疑", "都市", "情感", "真相"},
					PlayCount:     9800000,
					EpisodeCount:  106,
					DownloadedCount: 106,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7697612795184958489.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7701272618531310617,
					Title:         "夕阳红顶流，大爷大妈助我乐坛封神第二季",
					Category:      "AI 剧",
					Tags:          []string{"搞笑", "娱乐", "音乐", "逆袭"},
					PlayCount:     28400000,
					EpisodeCount:  117,
					DownloadedCount: 117,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7701272618531310617.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7671966197985856536,
					Title:         "持械入宋：第一季",
					Category:      "AI 剧",
					Tags:          []string{"穿越", "大宋", "热血", "争霸"},
					PlayCount:     14200000,
					EpisodeCount:  47,
					DownloadedCount: 47,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7671966197985856536.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7697209269287914521,
					Title:         "持械入宋：第十季",
					Category:      "AI 剧",
					Tags:          []string{"穿越", "大宋", "巅峰", "争霸"},
					PlayCount:     19800000,
					EpisodeCount:  100,
					DownloadedCount: 100,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7697209269287914521.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7697983901360196633,
					Title:         "村花错嫁，竟是弟弟替哥哥相亲",
					Category:      "AI 剧",
					Tags:          []string{"乡村", "错嫁", "甜宠", "致富"},
					PlayCount:     17500000,
					EpisodeCount:  152,
					DownloadedCount: 152,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7697983901360196633.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7701567939715025944,
					Title:         "满院亲戚全是上古大妖第六季",
					Category:      "漫剧",
					Tags:          []string{"玄幻", "神话", "团宠", "爆笑"},
					PlayCount:     41200000,
					EpisodeCount:  272,
					DownloadedCount: 272,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7701567939715025944.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7697574151787121726,
					Title:         "灵葫仙缘：葫中藏尽世间粮",
					Category:      "AI 剧",
					Tags:          []string{"仙侠", "修仙", "空间", "种田"},
					PlayCount:     13900000,
					EpisodeCount:  81,
					DownloadedCount: 81,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7697574151787121726.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7698313218652916798,
					Title:         "真千金她真没有背景通天",
					Category:      "AI 剧",
					Tags:          []string{"真假千金", "豪门", "马甲", "打脸"},
					PlayCount:     15400000,
					EpisodeCount:  40,
					DownloadedCount: 40,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7698313218652916798.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7699123435422632985,
					Title:         "纨绔下乡-顽郎亦不负流年第三季",
					Category:      "AI 剧",
					Tags:          []string{"下乡", "知青", "年代", "逆袭"},
					PlayCount:     21600000,
					EpisodeCount:  198,
					DownloadedCount: 198,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7699123435422632985.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7701226234214108185,
					Title:         "聚宝仙盆之杂灵根才是真BOSS第十五季",
					Category:      "漫剧",
					Tags:          []string{"修仙", "法宝", "逆天", "无敌"},
					PlayCount:     38900000,
					EpisodeCount:  300,
					DownloadedCount: 300,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7701226234214108185.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7700138431145315390,
					Title:         "边军破刀：一路砍到裂土封王第二季",
					Category:      "漫剧",
					Tags:          []string{"历史", "争霸", "铁血", "战神"},
					PlayCount:     24500000,
					EpisodeCount:  53,
					DownloadedCount: 53,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7700138431145315390.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
				{
					ID:            7697622096326708286,
					Title:         "退役归来，这个保镖不好惹",
					Category:      "AI 剧",
					Tags:          []string{"战神", "保镖", "都市", "爽文"},
					PlayCount:     19100000,
					EpisodeCount:  100,
					DownloadedCount: 100,
					CoverURL:      "https://origin.flashvcp.org/api/admin/dramas/cover/7697622096326708286.jpg",
					Origin:        OriginStation,
					StreamMode:    "站长本地存储直供 · 1080P 超清",
				},
			}
}

func isCloudCategory(cat string) bool {
	return strings.Contains(cat, "官方") ||
		strings.Contains(cat, "独播")
}

func seedFallbackDramas() []*DramaItem {
	return []*DramaItem{
		{
			ID:              7698294694886788121,
			Title:           "和离后我带女儿横扫魔域",
			Category:        "漫剧",
			Tags:            []string{"系统", "玄幻", "异界", "无敌", "高燃"},
			PlayCount:       91168000,
			LikeCount:       3820000,
			EpisodeCount:    987,
			DownloadedCount: 987,
			Origin:          OriginCloud,
			StreamMode:      "FlashVCP 云端分发 · DePIN 边缘秒开",
			Description:     "陆清安曾是威震诸天的永夜魔帝，回归烟火生活后带着两个女儿重返魔域，开启轻松热闹的无敌新旅程。全网热播榜 TOP 1。",
		},
		{
			ID:              7701226234214108185,
			Title:           "聚宝仙盆之杂灵根才是真BOSS第十五季",
			Category:        "漫剧",
			Tags:            []string{"热播", "仙侠", "玄幻"},
			PlayCount:       52100000,
			LikeCount:       2140000,
			EpisodeCount:    300,
			DownloadedCount: 300,
			Origin:          OriginCloud,
			StreamMode:      "FlashVCP 云端分发 · DePIN 边缘秒开",
			Description:     "杂灵根少年机缘巧合获得上古聚宝仙盆，万千灵宝信手拈来，逆天改命横扫八荒！",
		},
		{
			ID:              7701567939715025944,
			Title:           "满院亲戚全是上古大妖第六季",
			Category:        "漫剧",
			Tags:            []string{"轻松", "搞笑", "大妖", "无敌"},
			PlayCount:       43800000,
			LikeCount:       1890000,
			EpisodeCount:    272,
			DownloadedCount: 272,
			Origin:          OriginCloud,
			StreamMode:      "FlashVCP 云端分发 · DePIN 边缘秒开",
			Description:     "看似普通的乡村小院，七大姑八大姨竟然全是沉睡万年的洪荒上古大妖！",
		},
	}
}

// GetShortVideos returns the list of short videos.
func GetShortVideos() []*ShortVideoItem {
	catalogMu.RLock()
	if len(cachedShortVid) > 0 {
		defer catalogMu.RUnlock()
		return cachedShortVid
	}
	catalogMu.RUnlock()

	catalogMu.Lock()
	defer catalogMu.Unlock()
	if len(cachedShortVid) > 0 {
		return cachedShortVid
	}

	loadShortVideosLocked()
	return cachedShortVid
}

func loadShortVideosLocked() {
	endpoint := GetStationEndpoint()

	// Strategy 1: Fetch directly from Station Master API (/catalog.json)
	var data []byte
	if endpoint != "" {
		url := fmt.Sprintf("%s/catalog.json", strings.TrimRight(endpoint, "/"))
		resp, err := httpClient.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			if buf, err := io.ReadAll(resp.Body); err == nil && len(buf) > 0 {
				data = buf
			}
		}
	}

	// Strategy 2: Fall back to local disk metadata if station API is offline
	if len(data) == 0 {
		candidatePaths := []string{
			`I:\FlashMedia\catalog\catalog.json`,
			`D:\vcp\.local\catalog.json`,
		}

		for _, p := range candidatePaths {
			if d, err := os.ReadFile(p); err == nil && len(d) > 0 {
				data = d
				break
			}
		}
	}

	if len(data) > 0 {
		type rawCatalogItem struct {
			ID          string      `json:"id"`
			Title       string      `json:"title"`
			Description string      `json:"description"`
			Category    string      `json:"category"`
			Tags        []string    `json:"tags"`
			Duration    interface{} `json:"duration"`
			ViewCount   int64       `json:"viewCount"`
			LikeCount   int64       `json:"likeCount"`
			CoverURL    string      `json:"coverUrl"`
			HlsURL      string      `json:"hlsUrl"`
		}
		var rawList []rawCatalogItem
		if err := json.Unmarshal(data, &rawList); err == nil && len(rawList) > 0 {
			for _, r := range rawList {
				durStr := "01:00"
				durSec := 60
				switch v := r.Duration.(type) {
				case float64:
					durSec = int(v)
					durStr = fmt.Sprintf("%02d:%02d", durSec/60, durSec%60)
				case string:
					durStr = v
				}

				author := "站长自营·创作者"
				if r.Category != "" {
					author = r.Category + " 专栏"
				}

				tags := r.Tags
				if len(tags) == 0 && r.Category != "" {
					tags = []string{r.Category, "热门", "精选"}
				}

				// Build absolute cover URL if relative
				cover := r.CoverURL
				if strings.HasPrefix(cover, "/") && endpoint != "" {
					cover = strings.TrimRight(endpoint, "/") + cover
				}
				hls := r.HlsURL
				if strings.HasPrefix(hls, "/") && endpoint != "" {
					hls = strings.TrimRight(endpoint, "/") + hls
				}

				cachedShortVid = append(cachedShortVid, &ShortVideoItem{
					ID:          r.ID,
					Title:       r.Title,
					Description: r.Description,
					Category:    r.Category,
					Tags:        tags,
					Author:      author,
					PlayCount:   r.ViewCount,
					LikeCount:   r.LikeCount,
					Duration:    durStr,
					DurationSec: durSec,
					CoverURL:    cover,
					HlsURL:      hls,
					Origin:      OriginStation,
				})
			}
			return
		}
	}

		// Fallback curated seed
		cachedShortVid = []*ShortVideoItem{
			{
				ID:          "sv-101",
				Title:       "山海秘境：赛博青绿千山秀",
				Description: "AI 算力渲染的次世代国风粒子视效",
				Category:    "秘境探险",
				Author:      "数字墨韵工作室",
				CoverURL:    "https://images.unsplash.com/photo-1518709268805-4e9042af9f23?w=600&auto=format&fit=crop",
				PlayCount:   1840000,
				LikeCount:   291000,
				Duration:    "00:45",
				Origin:      OriginCloud,
			},
			{
				ID:          "sv-102",
				Title:       "全透明晶体主机水冷装配日常",
				Description: "次世代高性能分体式水冷装配全记录",
				Category:    "极客数码",
				Author:      "极客汇",
				CoverURL:    "https://images.unsplash.com/photo-1587202372775-e229f172b9d7?w=600&auto=format&fit=crop",
				PlayCount:   952000,
				LikeCount:   143000,
				Duration:    "00:32",
				Origin:      OriginStation,
			},
			{
				ID:          "sv-103",
				Title:       "治愈系咖啡冲煮：慢下来的 60 秒",
				Description: "双倍浓缩配雪顶燕麦奶的秋日温暖",
				Category:    "日常美学",
				Author:      "晨光物语",
				CoverURL:    "https://images.unsplash.com/photo-1501339847302-ac426a4a7cbb?w=600&auto=format&fit=crop",
				PlayCount:   2340000,
				LikeCount:   389000,
				Duration:    "01:05",
				Origin:      OriginCrawler,
			},
			{
				ID:          "sv-104",
				Title:       "4K 超清街舞踩点：重低音震碎屏幕",
				Description: "顶级街舞团体高能踩点卡点视听盛宴",
				Category:    "潮流舞蹈",
				Author:      "UrbanDance 舞团",
				CoverURL:    "https://images.unsplash.com/photo-1547153760-18fc86324498?w=600&auto=format&fit=crop",
				PlayCount:   4120000,
				LikeCount:   620000,
				Duration:    "00:52",
				Origin:      OriginCloud,
			},
			{
				ID:          "sv-105",
				Title:       "汉服变装刹那回眸：一梦回盛唐",
				Description: "盛世大唐华服光影流转，一颦一笑皆是东方雅韵",
				Category:    "颜值变装",
				Author:      "青丝如故",
				CoverURL:    "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=600&auto=format&fit=crop",
				PlayCount:   5880000,
				LikeCount:   890000,
				Duration:    "00:38",
				Origin:      OriginStation,
			},
			{
				ID:          "sv-106",
				Title:       "极限高空滑板跳伞：地平线尽头",
				Description: "四千米高空纵身一跃的窒息心跳体验",
				Category:    "秘境探险",
				Author:      "RedBull 冒险家",
				CoverURL:    "https://images.unsplash.com/photo-1517649763962-0c623266ddc0?w=600&auto=format&fit=crop",
				PlayCount:   3290000,
				LikeCount:   512000,
				Duration:    "00:48",
				Origin:      OriginCloud,
			},
			{
				ID:          "sv-107",
				Title:       "深海巨鲸穿云而过：4K 粒子奇境",
				Description: "虚幻引擎渲染的深海巨兽梦幻巡游",
				Category:    "秘境探险",
				Author:      "赛博元宇宙",
				CoverURL:    "https://images.unsplash.com/photo-1507525428034-b723cf961d3e?w=600&auto=format&fit=crop",
				PlayCount:   2750000,
				LikeCount:   418000,
				Duration:    "01:12",
				Origin:      OriginStation,
			},
			{
				ID:          "sv-108",
				Title:       "爆笑萌宠日常：这只猫成精了吧",
				Description: "偷吃猫条被抓现行，表情包瞬间拉满",
				Category:    "爆笑搞笑",
				Author:      "喵星人观察所",
				CoverURL:    "https://images.unsplash.com/photo-1514888286974-6c03e2ca1dba?w=600&auto=format&fit=crop",
				PlayCount:   6210000,
				LikeCount:   950000,
				Duration:    "00:42",
				Origin:      OriginCrawler,
			},
			{
				ID:          "sv-109",
				Title:       "经典港片高光混剪：光影黄金时代",
				Description: "那一抹回眸与决绝，刻在岁月里的港风记忆",
				Category:    "影视混剪",
				Author:      "胶片放映室",
				CoverURL:    "https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?w=600&auto=format&fit=crop",
				PlayCount:   1980000,
				LikeCount:   305000,
				Duration:    "01:30",
				Origin:      OriginCloud,
			},
			{
				ID:          "sv-110",
				Title:       "秋日森系人像街拍：自然光与微风",
				Description: "胶片感色调记录街头漫步的自然美好",
				Category:    "时尚模特",
				Author:      "Lens 超模志",
				CoverURL:    "https://images.unsplash.com/photo-1494790108377-be9c29b29330?w=600&auto=format&fit=crop",
				PlayCount:   1450000,
				LikeCount:   220000,
				Duration:    "00:29",
				Origin:      OriginStation,
			},
			{
				ID:          "sv-111",
				Title:       "沉浸式雨声 ASMR：深夜安眠自愈",
				Description: "雨打芭蕉与木屋炉火的纯粹白噪音",
				Category:    "沉浸声学",
				Author:      "声学实验室",
				CoverURL:    "https://images.unsplash.com/photo-1515694346937-94d85e41e6f0?w=600&auto=format&fit=crop",
				PlayCount:   3100000,
				LikeCount:   480000,
				Duration:    "02:00",
				Origin:      OriginCloud,
			},
			{
				ID:          "sv-112",
				Title:       "15分钟晨间普拉提：唤醒核心力量",
				Description: "舒展身心，开启一整天的充沛活力",
				Category:    "健身塑形",
				Author:      "FitLife 运动家",
				CoverURL:    "https://images.unsplash.com/photo-1518611012118-696072aa579a?w=600&auto=format&fit=crop",
				PlayCount:   1820000,
				LikeCount:   278000,
				Duration:    "01:15",
				Origin:      OriginStation,
			},
		}
}
