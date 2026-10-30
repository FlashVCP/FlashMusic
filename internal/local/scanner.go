package local

import (
	"os"
	"path/filepath"
	"strings"

	"netease-music-mygo/internal/source"
)

var supportedExts = map[string]bool{
	".mp3":  true,
	".flac": true,
}

// ScanDirectory recursively searches dir for supported music files,
// parses their tags, and registers them in the source registry.
func ScanDirectory(dir string) ([]*source.LocalTrack, error) {
	var tracks []*source.LocalTrack

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExts[ext] {
			return nil
		}

		track, parseErr := ParseFile(path)
		if parseErr == nil && track != nil {
			source.RegisterLocal(track)
			tracks = append(tracks, track)
		}
		return nil
	})

	return tracks, err
}
