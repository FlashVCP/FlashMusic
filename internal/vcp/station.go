package vcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ContentOrigin defines where a drama or video asset originated.
type ContentOrigin string

const (
	// OriginStation indicates the resource is physically stored on the Station Master's server/disk.
	// Plays directly via Station Master HTTP 206 partial content with 0 buffering.
	OriginStation ContentOrigin = "station"

	// OriginCloud indicates the resource was aggregated from FlashVCP Cloud via "一键采集".
	// Accelerated and distributed through DePIN L1 Edge Nodes.
	OriginCloud ContentOrigin = "cloud"

	// OriginCrawler indicates the resource was crawled by the Station Master from third-party sources.
	OriginCrawler ContentOrigin = "crawler"
)

// OriginLabel returns the user-facing display badge text.
func (o ContentOrigin) OriginLabel() string {
	switch o {
	case OriginStation:
		return "站长直出"
	case OriginCloud:
		return "云端采集"
	case OriginCrawler:
		return "全网聚合"
	default:
		return "站长直供"
	}
}

// OriginDescription returns a concise explanation of the delivery path.
func (o ContentOrigin) OriginDescription() string {
	switch o {
	case OriginStation:
		return "站长本地存储直接提供 · 0缓冲极速秒播"
	case OriginCloud:
		return "FlashVCP Cloud 官方云端聚合 · DePIN 边缘节点加速"
	case OriginCrawler:
		return "站长自定义程序采集 · 第三方多源聚合通道"
	default:
		return "站长自营内容"
	}
}

// StationStatus captures the health and metadata of the connected Station Master.
type StationStatus struct {
	Endpoint     string        `json:"endpoint"`
	Online       bool          `json:"online"`
	Name         string        `json:"name"`
	LatencyMs    int           `json:"latency_ms"`
	TotalDramas  int           `json:"total_dramas"`
	LocalDramas  int           `json:"local_dramas"`
	CloudDramas  int           `json:"cloud_dramas"`
	StorageDir   string        `json:"storage_dir"`
	LastSyncedAt time.Time     `json:"last_synced_at"`
}

var (
	stationMu       sync.RWMutex
	currentEndpoint = "https://open.flashvcp.org"
	stationStatus   = &StationStatus{
		Endpoint:    currentEndpoint,
		Online:      true,
		Name:        "FlashMate 生态示范站长平台 (Hong Kong)",
		LatencyMs:   25,
		TotalDramas: 53,
		LocalDramas: 1,
		CloudDramas: 52,
		StorageDir:  `FlashVCP Cloud Storage`,
	}
	httpClient = &http.Client{
		Timeout: 5000 * time.Millisecond,
	}
)

// GetStationEndpoint returns the currently configured Station Master URL.
func GetStationEndpoint() string {
	stationMu.RLock()
	defer stationMu.RUnlock()
	return currentEndpoint
}

// SetStationEndpoint updates the Station Master URL and triggers a re-fetch.
func SetStationEndpoint(ep string) {
	ep = strings.TrimSpace(ep)
	if ep == "" {
		ep = "https://open.flashvcp.org"
	}
	ep = strings.TrimRight(ep, "/")

	stationMu.Lock()
	currentEndpoint = ep
	stationStatus.Endpoint = ep
	stationMu.Unlock()

	// Invalidate local in-memory drama cache to reload from the new station
	dramasMu.Lock()
	cachedDramas = nil
	dramasMu.Unlock()

	go PingStation(ep)
}

// GetStationStatus returns the cached station health status.
func GetStationStatus() *StationStatus {
	stationMu.RLock()
	defer stationMu.RUnlock()
	return stationStatus
}

// PingStation checks connectivity and queries the station master server.
func PingStation(ep string) (*StationStatus, error) {
	start := time.Now()
	resp, err := httpClient.Get(ep + "/health")
	latency := int(time.Since(start).Milliseconds())

	stationMu.Lock()
	defer stationMu.Unlock()

	if err != nil {
		stationStatus.Online = false
		stationStatus.LatencyMs = latency
		return stationStatus, err
	}
	defer resp.Body.Close()

	stationStatus.Online = resp.StatusCode == http.StatusOK
	stationStatus.LatencyMs = latency
	stationStatus.LastSyncedAt = time.Now()
	return stationStatus, nil
}

// StationDramasResponse models the JSON returned by FlashVCP Open /api/admin/dramas.
type StationDramasResponse struct {
	OK         bool         `json:"ok"`
	Total      int          `json:"total"`
	Items      []*DramaItem `json:"items"`
	StorageDir string       `json:"storageDir"`
	VideosDir  string       `json:"videosDir"`
}

// FetchDramasFromStation queries the Station Master API.
func FetchDramasFromStation(ep string) ([]*DramaItem, error) {
	url := fmt.Sprintf("%s/api/admin/dramas", strings.TrimRight(ep, "/"))
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("station api returned status %d", resp.StatusCode)
	}

	var res StationDramasResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}

	stationMu.Lock()
	stationStatus.Online = true
	stationStatus.TotalDramas = res.Total
	stationStatus.StorageDir = res.StorageDir
	stationMu.Unlock()

	return res.Items, nil
}

func extractEpisodeNumber(name string) int {
	idx := strings.Index(name, "第")
	if idx != -1 {
		rest := name[idx+len("第"):]
		if endIdx := strings.Index(rest, "集"); endIdx != -1 {
			var n int
			if _, err := fmt.Sscanf(rest[:endIdx], "%d", &n); err == nil {
				return n
			}
		}
	}
	return 0
}

var knownHongguoDramas = map[string]int64{
	"一身腱子肉，乱世做谋臣":             7696066629439867966,
	"亲家母带全家来我家过年":             7695360003212856344,
	"分家没我份，破院枯井全是金条":           7698311709634595864,
	"十八岁太奶奶驾到，重整家族荣耀第五部":      7701935931787316286,
	"反向拿捏：和闺蜜穿七零带着婆婆一起离":      7692043430955863102,
	"和离后我带女儿横扫魔域":             7698294694886788121,
	"备用联系人（完）":                7697612795184958489,
	"夕阳红顶流，大爷大妈助我乐坛封神第二季":     7701272618531310617,
	"持械入宋：第一季":                7671966197985856536,
	"持械入宋：第十季":                7697209269287914521,
	"村花错嫁，竟是弟弟替哥哥相亲":           7697983901360196633,
	"满院亲戚全是上古大妖第六季":           7701567939715025944,
	"灵葫仙缘：葫中藏尽世间粮":           7697574151787121726,
	"真千金她真没有背景通天":             7698313218652916798,
	"纨绔下乡-顽郎亦不负流年第三季":         7699123435422632985,
	"聚宝仙盆之杂灵根才是真BOSS第十五季":     7701226234214108185,
	"边军破刀：一路砍到裂土封王第二季":       7700138431145315390,
	"退役归来，这个保镖不好惹":           7697622096326708286,
}

// GetLocalDownloadedDramasMap scans the videos folder to find which dramas have local MP4 files.
func GetLocalDownloadedDramasMap() map[int64][]string {
	result := make(map[int64][]string)
	candidateDirs := []string{
		`J:\FlashMedia\dramas\videos\hongguo`,
		`J:\FlashMedia\dramas\videos`,
		`I:\FlashMedia\dramas\videos\hongguo`,
		`I:\FlashMedia\dramas\videos`,
		`D:\FlashMedia\dramas\videos\hongguo`,
		`D:\FlashMedia\dramas\videos`,
	}

	for _, dir := range candidateDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			var id int64
			if _, err := fmt.Sscanf(name, "[%d]", &id); err == nil && id > 0 {
				// matched by [id]
			} else {
				cleanName := strings.Trim(name, "[]")
				if idFromMap, ok := knownHongguoDramas[cleanName]; ok {
					id = idFromMap
				} else {
					for k, v := range knownHongguoDramas {
						if strings.Contains(cleanName, k) || strings.Contains(k, cleanName) {
							id = v
							break
						}
					}
				}
			}

			if id > 0 && len(result[id]) == 0 {
				subDir := filepath.Join(dir, name)
				files, err := os.ReadDir(subDir)
				if err == nil {
					var mp4s []string
					for _, f := range files {
						if strings.HasSuffix(strings.ToLower(f.Name()), ".mp4") && !strings.Contains(f.Name(), ".part") {
							mp4s = append(mp4s, f.Name())
						}
					}
					sort.Slice(mp4s, func(i, j int) bool {
						ni := extractEpisodeNumber(mp4s[i])
						nj := extractEpisodeNumber(mp4s[j])
						if ni != nj {
							return ni < nj
						}
						return mp4s[i] < mp4s[j]
					})
					if len(mp4s) > 0 {
						result[id] = mp4s
					}
				}
			}
		}
	}
	return result
}

// GetShortVideoPlayURL resolves the stream URL for a short video from Station Master.
func GetShortVideoPlayURL(v *ShortVideoItem) string {
	if v == nil {
		return ""
	}
	ep := GetStationEndpoint()
	if ep == "" {
		ep = "https://open.flashvcp.org"
	}
	ep = strings.TrimRight(ep, "/")

	if v.HlsURL != "" {
		if strings.HasPrefix(v.HlsURL, "http://") || strings.HasPrefix(v.HlsURL, "https://") {
			return v.HlsURL
		}
		return ep + "/" + strings.TrimLeft(v.HlsURL, "/")
	}

	if v.ID != "" {
		return fmt.Sprintf("%s/media/%s/index.m3u8", ep, v.ID)
	}
	return ""
}

// GetShortVideoEmbedURL returns the clean embedded player URL for WebView2 playback.
func GetShortVideoEmbedURL(v *ShortVideoItem) string {
	return GetShortVideoEmbedURLWithNext(v, nil)
}

// GetShortVideoEmbedURLWithNext returns the embedded player URL with next video prefetch.
func GetShortVideoEmbedURLWithNext(v *ShortVideoItem, next *ShortVideoItem) string {
	if v == nil {
		return ""
	}
	playURL := GetShortVideoPlayURL(v)
	if playURL == "" {
		return ""
	}
	ep := GetStationEndpoint()
	if ep == "" {
		ep = "https://open.flashvcp.org"
	}
	ep = strings.TrimRight(ep, "/")

	nextSrc := ""
	if next != nil {
		nextSrc = GetShortVideoPlayURL(next)
	}

	embedURL := fmt.Sprintf("%s/player/embed?src=%s&title=%s&type=hls&autoplay=1",
		ep, url.QueryEscape(playURL), url.QueryEscape(v.Title))
	if nextSrc != "" {
		embedURL += fmt.Sprintf("&next_src=%s", url.QueryEscape(nextSrc))
	}
	return embedURL
}

// GetDramaEpisodePlayURL returns the HTTP 206 streaming URL for a specific episode of a drama.
func GetDramaEpisodePlayURL(drama *DramaItem, epIndex int) string {
	if drama == nil {
		return ""
	}
	epNum := epIndex + 1

	// 1. 如果有专属云端/母带短剧，通过站长统一点播通道或 CDN 播放 (默认启用阶梯渐进切片流以获得 0.15s 秒开体验)
	if drama.ID > 0 {
		cleanTitle := strings.Trim(drama.Title, "[]")
		fileName := fmt.Sprintf("%s - 第%02d集.mp4", cleanTitle, epNum)
		return fmt.Sprintf("https://origin.flashvcp.org/api/admin/dramas/ladder/%d/%s/index.m3u8?codec=h264", drama.ID, url.PathEscape(fileName))
	}

	// 2. If episode has video_url
	if epIndex >= 0 && epIndex < len(drama.Episodes) && drama.Episodes[epIndex].VideoURL != "" {
		u := drama.Episodes[epIndex].VideoURL
		if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
			return u
		}
		return u
	}

	return ""
}

// GetDramaEpisodeEmbedURL returns the embedded player URL for drama theater playback.
func GetDramaEpisodeEmbedURL(drama *DramaItem, epIndex int) string {
	if drama == nil {
		return ""
	}
	playURL := GetDramaEpisodePlayURL(drama, epIndex)
	if playURL == "" {
		return ""
	}

	// 纯净播放内核由全球 CDN (origin.flashvcp.org) 统一承载，彻底告别本地端口依赖
	playerHost := "https://origin.flashvcp.org"

	epNum := epIndex + 1
	title := fmt.Sprintf("%s - 第%d集", drama.Title, epNum)
	mediaType := "mp4"
	if strings.Contains(playURL, ".m3u8") {
		mediaType = "hls"
	}

	totalEp := len(drama.Episodes)
	if totalEp == 0 {
		totalEp = drama.EpisodeCount
	}

	nextSrc := ""
	if epIndex+1 < totalEp {
		nextPlayURL := GetDramaEpisodePlayURL(drama, epIndex+1)
		if nextPlayURL != "" {
			nextSrc = nextPlayURL
		}
	}

		embedURL := fmt.Sprintf("%s/player/embed?src=%s&title=%s&type=%s&autoplay=1",
			playerHost, url.QueryEscape(playURL), url.QueryEscape(title), mediaType)
		if nextSrc != "" {
			embedURL += fmt.Sprintf("&next_src=%s", url.QueryEscape(nextSrc))
		}
		return embedURL
	}

// PrefetchDramaEpisodes notifies the media origin server to pre-cache upcoming episodes.
func PrefetchDramaEpisodes(dramaID int64, epIndex, lookahead int) {
	if dramaID == 0 {
		return
	}
	// 媒体预热直接向源站 CDN 通道发起 (origin.flashvcp.org / 香港示范站)
	originHost := "https://origin.flashvcp.org"
	reqURL := fmt.Sprintf("%s/api/v1/media/prefetch?drama_id=%d&ep=%d&lookahead=%d", originHost, dramaID, epIndex, lookahead)
	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(reqURL)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()
}
