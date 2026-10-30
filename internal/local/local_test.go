package local

import (
	"os"
	"path/filepath"
	"testing"

	"netease-music-mygo/internal/source"
)

func TestLocalTrackIDAndRegistry(t *testing.T) {
	path := `C:\Music\Artist - Song.mp3`
	id := source.LocalTrackID(path)
	if id >= 0 {
		t.Fatalf("expected negative ID for local track, got %d", id)
	}
	if !source.IsLocal(id) {
		t.Fatalf("expected source.IsLocal(%d) to be true", id)
	}

	track := &source.LocalTrack{
		ID:       id,
		Path:     path,
		Title:    "Song",
		Artist:   "Artist",
		Album:    "Album",
		Duration: 180000,
		Format:   "mp3",
	}
	source.RegisterLocal(track)

	got, ok := source.GetLocal(id)
	if !ok || got.Title != "Song" {
		t.Fatalf("failed to retrieve registered local track: got %+v, ok=%v", got, ok)
	}

	converted := track.ToTrack()
	if converted.ID != id || converted.Name != "Song" || converted.Artists[0].Name != "Artist" {
		t.Fatalf("ToTrack conversion mismatch: %+v", converted)
	}
	if !converted.Playable() {
		t.Fatalf("expected local track to be Playable()")
	}
}

func TestScanDirectoryEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "flashmusic-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	tracks, err := ScanDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks in empty dir, got %d", len(tracks))
	}

	// Create a dummy .mp3 file with "Artist - Song.mp3" naming
	samplePath := filepath.Join(tmpDir, "周杰伦 - 晴天.mp3")
	if err := os.WriteFile(samplePath, []byte("fake mp3 audio data"), 0o644); err != nil {
		t.Fatal(err)
	}

	tracks, err = ScanDirectory(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}
	if tracks[0].Artist != "周杰伦" || tracks[0].Title != "晴天" {
		t.Fatalf("expected artist 周杰伦 and title 晴天, got %q - %q", tracks[0].Artist, tracks[0].Title)
	}
}
