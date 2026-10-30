package source

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"netease-music-mygo/internal/netease"
)

type Kind string

const (
	KindNetEase Kind = "netease"
	KindLocal   Kind = "local"
	KindCloud   Kind = "cloud"
)

// LocalTrack represents a local audio file's extracted metadata and state.
type LocalTrack struct {
	ID       int64
	Path     string
	Title    string
	Artist   string
	Album    string
	Duration int64 // in milliseconds
	Format   string
	Size     int64
	CoverURL string
	Cover    []byte
	Lyrics   string
}

// ToTrack converts a LocalTrack to a netease.Track compatible structure
// so all existing lists, player queues, Now Playing, and cover walls can consume it directly.
func (lt *LocalTrack) ToTrack() netease.Track {
	cover := lt.CoverURL
	if cover == "" {
		cover = fmt.Sprintf("flashmusic://cover/%d", lt.ID)
	}
	artistName := lt.Artist
	if artistName == "" {
		artistName = "未知歌手"
	}
	albumName := lt.Album
	if albumName == "" {
		albumName = "本地音乐"
	}
	title := lt.Title
	if title == "" {
		title = filepath.Base(lt.Path)
	}
	return netease.Track{
		ID:       lt.ID,
		Name:     title,
		Artists:  []netease.Artist{{Name: artistName}},
		Album:    netease.Album{Name: albumName, PicURL: cover},
		Duration: lt.Duration,
	}
}

// LocalTrackID generates a deterministic negative int64 ID from a file path.
// NetEase IDs are always positive (>0); negative IDs uniquely represent local tracks.
func LocalTrackID(path string) int64 {
	h := fnv.New64a()
	h.Write([]byte(filepath.Clean(path)))
	v := int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
	if v <= 0 {
		v = 1
	}
	return -v
}

// IsLocal reports whether an ID corresponds to a local track.
func IsLocal(id int64) bool {
	return id < 0
}

var (
	registryMu sync.RWMutex
	registry   = make(map[int64]*LocalTrack)
)

// RegisterLocal stores a local track in the memory registry.
func RegisterLocal(track *LocalTrack) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[track.ID] = track
}

// GetLocal retrieves a local track by its negative ID.
func GetLocal(id int64) (*LocalTrack, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	t, ok := registry[id]
	return t, ok
}

// RemoveLocal removes a single track from the registry by its negative ID.
func RemoveLocal(id int64) bool {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[id]; ok {
		delete(registry, id)
		return true
	}
	return false
}

// RemoveLocalByFolder removes all tracks belonging to dir (or its subdirectories) from registry.
func RemoveLocalByFolder(dir string) int {
	registryMu.Lock()
	defer registryMu.Unlock()
	cleanDir := filepath.Clean(dir)
	var removed int
	for id, track := range registry {
		trackDir := filepath.Clean(filepath.Dir(track.Path))
		rel, err := filepath.Rel(cleanDir, trackDir)
		if err == nil && !strings.HasPrefix(rel, "..") {
			delete(registry, id)
			removed++
		}
	}
	return removed
}

// GetCoverData retrieves the embedded cover image bytes for a local track, if any.
func GetCoverData(id int64) []byte {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if t, ok := registry[id]; ok && len(t.Cover) > 0 {
		return t.Cover
	}
	return nil
}

// GetAllLocal returns all registered local tracks in stable path order.
func GetAllLocal() []*LocalTrack {
	registryMu.RLock()
	defer registryMu.RUnlock()
	list := make([]*LocalTrack, 0, len(registry))
	for _, t := range registry {
		list = append(list, t)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Path < list[j].Path
	})
	return list
}

// LocalAlbum groups local tracks by their album name.
type LocalAlbum struct {
	Name     string
	Artist   string
	CoverURL string
	Tracks   []netease.Track
}

// LocalArtist groups local tracks by their artist name.
type LocalArtist struct {
	Name     string
	CoverURL string
	Tracks   []netease.Track
}

// LocalFolder groups local tracks by their parent folder path.
type LocalFolder struct {
	Path     string
	Name     string
	CoverURL string
	Tracks   []netease.Track
}

// GroupLocalByAlbum groups all registered local tracks into LocalAlbums with stable deterministic sorting.
func GroupLocalByAlbum() []*LocalAlbum {
	registryMu.RLock()
	defer registryMu.RUnlock()

	albumMap := make(map[string]*LocalAlbum)

	for _, lt := range registry {
		al := strings.TrimSpace(lt.Album)
		if al == "" {
			al = "未知专辑"
		}
		item, exists := albumMap[al]
		if !exists {
			item = &LocalAlbum{
				Name:   al,
				Artist: lt.Artist,
			}
			albumMap[al] = item
		}
		if item.CoverURL == "" || len(lt.Cover) > 0 {
			item.CoverURL = fmt.Sprintf("flashmusic://cover/%d", lt.ID)
		}
		item.Tracks = append(item.Tracks, lt.ToTrack())
	}

	res := make([]*LocalAlbum, 0, len(albumMap))
	for _, item := range albumMap {
		item.Tracks, _ = DeduplicateTracks(item.Tracks)
		sort.Slice(item.Tracks, func(i, j int) bool {
			return item.Tracks[i].Name < item.Tracks[j].Name
		})
		res = append(res, item)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

// GroupLocalByArtist groups all registered local tracks into LocalArtists with stable deterministic sorting.
func GroupLocalByArtist() []*LocalArtist {
	registryMu.RLock()
	defer registryMu.RUnlock()

	artistMap := make(map[string]*LocalArtist)

	for _, lt := range registry {
		ar := strings.TrimSpace(lt.Artist)
		if ar == "" {
			ar = "未知歌手"
		}
		item, exists := artistMap[ar]
		if !exists {
			item = &LocalArtist{
				Name: ar,
			}
			artistMap[ar] = item
		}
		if item.CoverURL == "" || len(lt.Cover) > 0 {
			item.CoverURL = fmt.Sprintf("flashmusic://cover/%d", lt.ID)
		}
		item.Tracks = append(item.Tracks, lt.ToTrack())
	}

	res := make([]*LocalArtist, 0, len(artistMap))
	for _, item := range artistMap {
		item.Tracks, _ = DeduplicateTracks(item.Tracks)
		sort.Slice(item.Tracks, func(i, j int) bool {
			return item.Tracks[i].Name < item.Tracks[j].Name
		})
		res = append(res, item)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res
}

// GroupLocalByFolder groups all registered local tracks into LocalFolders with stable deterministic sorting.
func GroupLocalByFolder() []*LocalFolder {
	registryMu.RLock()
	defer registryMu.RUnlock()

	folderMap := make(map[string]*LocalFolder)

	for _, lt := range registry {
		dir := filepath.Dir(lt.Path)
		item, exists := folderMap[dir]
		if !exists {
			item = &LocalFolder{
				Path: dir,
				Name: filepath.Base(dir),
			}
			folderMap[dir] = item
		}
		if item.CoverURL == "" || len(lt.Cover) > 0 {
			item.CoverURL = fmt.Sprintf("flashmusic://cover/%d", lt.ID)
		}
		item.Tracks = append(item.Tracks, lt.ToTrack())
	}

	res := make([]*LocalFolder, 0, len(folderMap))
	for _, item := range folderMap {
		item.Tracks, _ = DeduplicateTracks(item.Tracks)
		sort.Slice(item.Tracks, func(i, j int) bool {
			return item.Tracks[i].Name < item.Tracks[j].Name
		})
		res = append(res, item)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Path < res[j].Path
	})
	return res
}
