package source

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"netease-music-mygo/internal/netease"
)

// CanonicalizeDirs removes duplicate or redundant subdirectories.
// For example, if both "D:\Music" and "D:\Music\Pop" are present,
// the subfolder "D:\Music\Pop" is removed to prevent redundant traversal.
func CanonicalizeDirs(dirs []string) []string {
	if len(dirs) <= 1 {
		return dirs
	}

	clean := make([]string, 0, len(dirs))
	seen := make(map[string]bool)
	for _, d := range dirs {
		p := filepath.Clean(d)
		if !seen[p] {
			seen[p] = true
			clean = append(clean, p)
		}
	}

	sort.Slice(clean, func(i, j int) bool {
		return len(clean[i]) < len(clean[j])
	})

	var result []string
	for i, d := range clean {
		isSub := false
		for j := 0; j < i; j++ {
			rel, err := filepath.Rel(clean[j], d)
			if err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
				isSub = true
				break
			}
		}
		if !isSub {
			result = append(result, d)
		}
	}
	return result
}

// PruneMissingLocal removes any registered local tracks that no longer exist on disk.
func PruneMissingLocal() int {
	registryMu.Lock()
	defer registryMu.Unlock()

	var removed int
	for id, track := range registry {
		if _, err := os.Stat(track.Path); err != nil {
			delete(registry, id)
			removed++
		}
	}
	return removed
}

// ClearLocalRegistry completely clears the local track registry (for fresh rescans).
func ClearLocalRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[int64]*LocalTrack)
}

func normalizeString(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func qualityScore(t *LocalTrack) int {
	score := 0
	switch strings.ToLower(t.Format) {
	case "flac":
		score += 1000
	case "wav":
		score += 900
	case "ape":
		score += 850
	case "mp3":
		score += 500
	default:
		score += 100
	}
	// Prefer larger files (higher bitrate)
	mb := int(t.Size >> 20)
	score += min(mb, 200)
	// Prefer tracks with embedded covers
	if len(t.Cover) > 0 {
		score += 50
	}
	// Prefer tracks with lyrics
	if t.Lyrics != "" {
		score += 50
	}
	return score
}

// DeduplicateLocal filters a list of LocalTracks, merging duplicates and
// keeping only the highest-quality version of each song.
// It returns the deduplicated list and the number of merged duplicates.
func DeduplicateLocal(tracks []*LocalTrack) ([]*LocalTrack, int) {
	if len(tracks) <= 1 {
		return tracks, 0
	}

	// 1. Group by normalized "Artist|Title"
	type songGroup struct {
		artistNorm string
		titleNorm  string
		versions   []*LocalTrack
	}

	var groups []*songGroup

	for _, t := range tracks {
		an := normalizeString(t.Artist)
		tn := normalizeString(t.Title)

		matched := false
		if tn != "" {
			for _, g := range groups {
				// Same title, and matching artist (or one is empty)
				if g.titleNorm == tn && (g.artistNorm == an || an == "" || g.artistNorm == "") {
					// Check duration similarity (within 3.5 seconds) if both have duration
					durDiff := math.Abs(float64(t.Duration - g.versions[0].Duration))
					if t.Duration == 0 || g.versions[0].Duration == 0 || durDiff <= 3500 {
						g.versions = append(g.versions, t)
						matched = true
						break
					}
				}
			}
		}

		if !matched {
			groups = append(groups, &songGroup{
				artistNorm: an,
				titleNorm:  tn,
				versions:   []*LocalTrack{t},
			})
		}
	}

	deduped := make([]*LocalTrack, 0, len(groups))
	totalDupes := 0

	for _, g := range groups {
		if len(g.versions) > 1 {
			totalDupes += len(g.versions) - 1
			// Sort versions by quality score descending
			sort.Slice(g.versions, func(i, j int) bool {
				return qualityScore(g.versions[i]) > qualityScore(g.versions[j])
			})
		}
		deduped = append(deduped, g.versions[0])
	}

	// Stable sort by path
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Path < deduped[j].Path
	})

	return deduped, totalDupes
}

// DeduplicateTracks deduplicates netease.Track lists generated from local tracks.
func DeduplicateTracks(tracks []netease.Track) ([]netease.Track, int) {
	if len(tracks) <= 1 {
		return tracks, 0
	}

	seen := make(map[string]netease.Track)
	var order []string

	for _, tr := range tracks {
		artist := ""
		if len(tr.Artists) > 0 {
			artist = tr.Artists[0].Name
		}
		key := normalizeString(artist) + "|" + normalizeString(tr.Name)
		if existing, found := seen[key]; found {
			// If duplicate, keep the one with longer duration or existing
			if tr.Duration > existing.Duration {
				seen[key] = tr
			}
		} else {
			seen[key] = tr
			order = append(order, key)
		}
	}

	res := make([]netease.Track, 0, len(order))
	for _, k := range order {
		res = append(res, seen[k])
	}
	return res, len(tracks) - len(res)
}
