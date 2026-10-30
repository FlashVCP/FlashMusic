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

// qualityScore computes an objective quality score for a track:
// 1. Hi-Res Lossless (FLAC >= 24-bit or >= 88.2kHz): ~3500+
// 2. Standard Lossless (FLAC 16-bit/44.1kHz, WAV, APE): ~2200 - 2500
// 3. Lossy high-bitrate (MP3 320kbps): ~1320
// 4. Lossy medium-bitrate (MP3 192kbps - 256kbps): ~1200
// 5. Lossy low-bitrate (MP3 128kbps): ~1128
func qualityScore(t *LocalTrack) int {
	score := 0
	format := strings.ToLower(t.Format)

	switch format {
	case "flac":
		if t.BitDepth >= 24 || t.SampleRate >= 88200 {
			// Hi-Res tier
			score = 3000
		} else {
			// Standard CD lossless tier
			score = 2000
		}
		// Reward bit depth (e.g. 24-bit vs 16-bit)
		if t.BitDepth > 0 {
			score += t.BitDepth * 15
		} else {
			score += 16 * 15
		}
		// Reward sample rate (e.g. 96kHz vs 44.1kHz)
		if t.SampleRate > 0 {
			score += t.SampleRate / 1000
		}
	case "wav", "ape":
		score = 2000
		if t.BitDepth > 0 {
			score += t.BitDepth * 10
		}
		if t.SampleRate > 0 {
			score += t.SampleRate / 1000
		}
	case "mp3":
		score = 1000
		// Reward parsed bitrate directly (320, 256, 192, 128, etc.)
		if t.Bitrate > 0 {
			score += t.Bitrate
		} else if t.Duration > 0 && t.Size > 0 {
			// Estimate bitrate if not directly parsed
			estBr := int((t.Size * 8) / t.Duration)
			score += min(estBr, 320)
		} else {
			score += 128
		}
	default:
		score = 500
	}

	// Tie-breakers:
	// 1. Embedded album artwork
	if len(t.Cover) > 0 {
		score += 30
	}
	// 2. Embedded or matched lyrics
	if t.Lyrics != "" {
		score += 20
	}
	// 3. Complete metadata (album name is not empty or "未知专辑")
	if t.Album != "" && t.Album != "未知专辑" {
		score += 10
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
