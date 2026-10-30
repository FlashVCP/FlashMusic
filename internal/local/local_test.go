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

func TestLocalGrouping(t *testing.T) {
	t1 := &source.LocalTrack{
		ID:     -101,
		Path:   `C:\Music\Pop\Jay - Track1.mp3`,
		Title:  "Track1",
		Artist: "Jay",
		Album:  "Fantasy",
	}
	t2 := &source.LocalTrack{
		ID:     -102,
		Path:   `C:\Music\Pop\Jay - Track2.mp3`,
		Title:  "Track2",
		Artist: "Jay",
		Album:  "Fantasy",
	}
	t3 := &source.LocalTrack{
		ID:     -103,
		Path:   `C:\Music\Rock\LinkinPark - InTheEnd.flac`,
		Title:  "InTheEnd",
		Artist: "LinkinPark",
		Album:  "Hybrid Theory",
	}

	source.RegisterLocal(t1)
	source.RegisterLocal(t2)
	source.RegisterLocal(t3)

	albums := source.GroupLocalByAlbum()
	if len(albums) < 2 {
		t.Fatalf("expected at least 2 albums, got %d", len(albums))
	}

	artists := source.GroupLocalByArtist()
	if len(artists) < 2 {
		t.Fatalf("expected at least 2 artists, got %d", len(artists))
	}

		folders := source.GroupLocalByFolder()
		if len(folders) < 2 {
			t.Fatalf("expected at least 2 folders, got %d", len(folders))
		}
	}

func TestCanonicalizeDirs(t *testing.T) {
	input := []string{
		`C:\Music`,
		`C:\Music\Pop`,
		`C:\Music\Rock\Alternative`,
		`D:\Audio`,
		`C:\Music`,
	}
	output := source.CanonicalizeDirs(input)
	if len(output) != 2 {
		t.Fatalf("expected 2 canonical root dirs, got %d: %+v", len(output), output)
	}
	if output[0] != `C:\Music` && output[1] != `C:\Music` {
		t.Fatalf("expected C:\\Music to be preserved in %+v", output)
	}
}

func TestDeduplicateLocal(t *testing.T) {
	// 4 versions of the same song across different directories:
	// 1. Hi-Res FLAC (24bit / 96kHz)
	hiresFlac := &source.LocalTrack{
		ID:         -201,
		Path:       `D:\Music\HiRes\Jay - 晴天.flac`,
		Title:      "晴天",
		Artist:     "周杰伦",
		Album:      "叶惠美",
		Duration:   269000,
		Format:     "flac",
		BitDepth:   24,
		SampleRate: 96000,
		Size:       50 << 20,
	}
	// 2. CD Lossless FLAC (16bit / 44.1kHz)
	cdFlac := &source.LocalTrack{
		ID:         -202,
		Path:       `D:\Music\CD\Jay - 晴天.flac`,
		Title:      "晴天",
		Artist:     "周杰伦",
		Album:      "叶惠美",
		Duration:   269000,
		Format:     "flac",
		BitDepth:   16,
		SampleRate: 44100,
		Size:       25 << 20,
	}
	// 3. MP3 320kbps
	mp3_320 := &source.LocalTrack{
		ID:         -203,
		Path:       `D:\Music\320k\周杰伦 - 晴天.mp3`,
		Title:      "晴天",
		Artist:     "周杰伦",
		Album:      "叶惠美",
		Duration:   270000,
		Format:     "mp3",
		Bitrate:    320,
		Size:       10 << 20,
	}
	// 4. MP3 128kbps
	mp3_128 := &source.LocalTrack{
		ID:         -204,
		Path:       `D:\Music\128k\周杰伦 - 晴天.mp3`,
		Title:      "晴天",
		Artist:     "周杰伦",
		Album:      "叶惠美",
		Duration:   270500,
		Format:     "mp3",
		Bitrate:    128,
		Size:       4 << 20,
	}
	otherTrack := &source.LocalTrack{
		ID:       -205,
		Path:     `D:\Music\Pop\周杰伦 - 七里香.mp3`,
		Title:    "七里香",
		Artist:   "周杰伦",
		Duration: 299000,
		Format:   "mp3",
		Bitrate:  320,
	}

	tracks := []*source.LocalTrack{mp3_128, cdFlac, mp3_320, hiresFlac, otherTrack}
	deduped, dupes := source.DeduplicateLocal(tracks)

	if dupes != 3 {
		t.Fatalf("expected 3 duplicates merged, got %d", dupes)
	}
	if len(deduped) != 2 {
		t.Fatalf("expected 2 deduped tracks, got %d", len(deduped))
	}

	// Verify that the Hi-Res FLAC version was strictly chosen over CD FLAC and MP3s
	var chosenQingTian *source.LocalTrack
	for _, tr := range deduped {
		if tr.Title == "晴天" {
			chosenQingTian = tr
			break
		}
	}
	if chosenQingTian == nil || chosenQingTian.Path != hiresFlac.Path {
		t.Fatalf("expected Hi-Res FLAC track to be selected for 晴天, got %+v", chosenQingTian)
	}

	// Test tie between CD FLAC and MP3s (without Hi-Res)
	tracks2 := []*source.LocalTrack{mp3_128, mp3_320, cdFlac}
	deduped2, _ := source.DeduplicateLocal(tracks2)
	if deduped2[0].Path != cdFlac.Path {
		t.Fatalf("expected CD FLAC to win over MP3s, got %s", deduped2[0].Path)
	}

		// Test tie between 320k and 128k MP3
		tracks3 := []*source.LocalTrack{mp3_128, mp3_320}
		deduped3, _ := source.DeduplicateLocal(tracks3)
		if deduped3[0].Path != mp3_320.Path {
			t.Fatalf("expected 320k MP3 to win over 128k MP3, got %s", deduped3[0].Path)
		}
	}
