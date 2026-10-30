package source

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
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

// GetCoverData retrieves the embedded cover image bytes for a local track, if any.
func GetCoverData(id int64) []byte {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if t, ok := registry[id]; ok && len(t.Cover) > 0 {
		return t.Cover
	}
	return nil
}

// GetAllLocal returns all registered local tracks.
func GetAllLocal() []*LocalTrack {
	registryMu.RLock()
	defer registryMu.RUnlock()
	list := make([]*LocalTrack, 0, len(registry))
	for _, t := range registry {
		list = append(list, t)
	}
	return list
}
