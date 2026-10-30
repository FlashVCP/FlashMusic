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
	// Two tracks with same artist & title, one FLAC (high quality) and one MP3 (low quality)
	flacTrack := &source.LocalTrack{
		ID:       -201,
		Path:     `D:\Music\FLAC\Jay - 晴天.flac`,
		Title:    "晴天",
		Artist:   "周杰伦",
		Album:    "叶惠美",
		Duration: 269000,
		Format:   "flac",
		Size:     35 << 20,
	}
	mp3Track := &source.LocalTrack{
		ID:       -202,
		Path:     `D:\Music\Car\周杰伦 - 晴天.mp3`,
		Title:    "晴天",
		Artist:   "周杰伦",
		Album:    "叶惠美",
		Duration: 270000,
		Format:   "mp3",
		Size:     8 << 20,
	}
	otherTrack := &source.LocalTrack{
		ID:       -203,
		Path:     `D:\Music\Pop\周杰伦 - 七里香.mp3`,
		Title:    "七里香",
		Artist:   "周杰伦",
		Duration: 299000,
		Format:   "mp3",
	}

	tracks := []*source.LocalTrack{flacTrack, mp3Track, otherTrack}
	deduped, dupes := source.DeduplicateLocal(tracks)

	if dupes != 1 {
		t.Fatalf("expected 1 duplicate merged, got %d", dupes)
	}
	if len(deduped) != 2 {
		t.Fatalf("expected 2 deduped tracks, got %d", len(deduped))
	}

	// Verify that the FLAC version was chosen over MP3
	var chosenQingTian *source.LocalTrack
	for _, tr := range deduped {
		if tr.Title == "晴天" {
			chosenQingTian = tr
			break
		}
	}
	if chosenQingTian == nil || chosenQingTian.Format != "flac" {
		t.Fatalf("expected FLAC track to be selected for 晴天, got %+v", chosenQingTian)
	}
}
