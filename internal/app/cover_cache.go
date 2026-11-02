package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

var (
	coverDiskDirOnce sync.Once
	coverDiskDir     string

	revalidatingMu sync.Mutex
	revalidating   = make(map[string]time.Time)
)

// getCoverCacheDir returns the persistent on-disk directory for image and album covers.
func getCoverCacheDir() string {
	coverDiskDirOnce.Do(func() {
		if d, err := mygo.App.Path(mygo.PathCache); err == nil && d != "" {
			coverDiskDir = filepath.Join(d, "covers")
		} else {
			base := os.Getenv("LOCALAPPDATA")
			if base == "" {
				home, _ := os.UserHomeDir()
				if home != "" {
					base = filepath.Join(home, "AppData", "Local")
				} else {
					base = os.TempDir()
				}
			}
			coverDiskDir = filepath.Join(base, "FlashMusic", "covers")
		}
		_ = os.MkdirAll(coverDiskDir, 0755)
	})
	return coverDiskDir
}

// urlCachePath returns the deterministic local file path for a remote image URL.
func urlCachePath(rawURL string) string {
	dir := getCoverCacheDir()
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(dir, hex.EncodeToString(sum[:]))
}

// readCoverDiskCache checks the local SSD persistent cache for a given URL.
// Returns the file bytes and true on hit; nil and false on cache miss.
func readCoverDiskCache(rawURL string) ([]byte, bool) {
	if rawURL == "" {
		return nil, false
	}
	path := urlCachePath(rawURL)
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}

// writeCoverDiskCache writes downloaded image bytes atomically to the local disk cache.
func writeCoverDiskCache(rawURL string, data []byte) {
	if rawURL == "" || len(data) == 0 {
		return
	}
	path := urlCachePath(rawURL)
	tmp := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, 0644); err == nil {
		_ = os.Rename(tmp, path)
	}
}

// revalidateCoverBackground checks in the background if the remote server has an updated
// version of the image (HTTP 304 vs 200). It only triggers once every 24 hours per URL.
func revalidateCoverBackground(rawURL string) {
	if rawURL == "" {
		return
	}

	revalidatingMu.Lock()
	last, running := revalidating[rawURL]
	if running && time.Since(last) < 12*time.Hour {
		revalidatingMu.Unlock()
		return
	}
	revalidating[rawURL] = time.Now()
	revalidatingMu.Unlock()

	path := urlCachePath(rawURL)
	fi, err := os.Stat(path)
	if err != nil {
		return
	}

	// Only revalidate if file is at least 24 hours old
	if time.Since(fi.ModTime()) < 24*time.Hour {
		return
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "FlashMusic/1.0")
	req.Header.Set("If-Modified-Since", fi.ModTime().UTC().Format(http.TimeFormat))

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		// Server confirmed unchanged: touch mod time to extend validation window
		now := time.Now()
		_ = os.Chtimes(path, now, now)
		return
	}

	if resp.StatusCode == http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err == nil && len(data) > 0 {
			writeCoverDiskCache(rawURL, data)
		}
	}
}
