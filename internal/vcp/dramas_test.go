package vcp

import (
	"strings"
	"testing"
)

func TestDramasLoad(t *testing.T) {
	dramas := GetDramas()
	if len(dramas) == 0 {
		t.Fatalf("expected at least some dramas, got 0")
	}
	t.Logf("Successfully loaded %d dramas", len(dramas))

	cats := GetCategories()
	if len(cats) == 0 {
		t.Fatalf("expected categories to be populated")
	}
	t.Logf("Available categories: %v", cats)

	// Test filtering
	filtered := FilterDramas("黄果原创🔥", "")
	t.Logf("Found %d dramas in 黄果原创🔥", len(filtered))

	// Test short videos
	sv := GetShortVideos()
	if len(sv) == 0 {
		t.Fatalf("expected short videos to be loaded")
	}
	t.Logf("Loaded %d short videos", len(sv))

	// Test Station Master Endpoint
	st := GetStationStatus()
	if st == nil {
		t.Fatalf("expected station status to be initialized")
	}
	t.Logf("Station Endpoint: %s, Total: %d, Local: %d, Cloud: %d",
		st.Endpoint, st.TotalDramas, st.LocalDramas, st.CloudDramas)

		// Test Downloaded Map
		downMap := GetLocalDownloadedDramasMap()
		t.Logf("Local Downloaded Dramas Map: %d dramas with downloaded files", len(downMap))
	}

func TestVideoPlayURLsAndTaxonomy(t *testing.T) {
	// 1. Sync Taxonomy
	SyncTaxonomyFromStation("")
	tax := GetShortVideoTaxonomy()
	if len(tax.Categories) == 0 {
		t.Fatalf("expected short video categories to be populated")
	}
	t.Logf("Taxonomy categories: %d, preset groups: %d", len(tax.Categories), len(tax.PresetGroups))

	// 2. Short Video Play URL & Embed URL
	svs := GetShortVideos()
	if len(svs) > 0 {
		sv := svs[0]
		playURL := GetShortVideoPlayURL(sv)
		embedURL := GetShortVideoEmbedURL(sv)
		t.Logf("Short Video ID=%s: playURL=%s, embedURL=%s", sv.ID, playURL, embedURL)
		if playURL == "" || embedURL == "" {
			t.Fatalf("failed to generate short video URLs")
		}
	}

		// 3. Drama Episode Play URL & Embed URL
		GetDramas()
		drama7698 := GetDrama(7698294694886788121)
		if drama7698 != nil {
			playURL := GetDramaEpisodePlayURL(drama7698, 0)
			embedURL := GetDramaEpisodeEmbedURL(drama7698, 0)
			t.Logf("Drama 7698294694886788121 ep 0: playURL=%s, embedURL=%s", playURL, embedURL)
			if playURL == "" || embedURL == "" {
				t.Fatalf("failed to generate drama episode URLs")
			}
			if !strings.Contains(embedURL, "next_src=") {
				t.Fatalf("embedURL missing next_src: %s", embedURL)
			}
		} else {
			t.Fatalf("expected drama 7698294694886788121 to exist")
		}

	// 4. Test Advanced Filter
	danceVideos := FilterShortVideosAdvanced("潮流舞蹈", "全部", "", "全部", "全部", "")
	t.Logf("Found %d videos in 潮流舞蹈", len(danceVideos))
	heatVideos := FilterShortVideosAdvanced("全部", "全部", "", "全部", "100万+ 爆款", "heat")
	t.Logf("Found %d 100万+ 爆款 videos sorted by heat", len(heatVideos))

	// 5. Test Drama Cover Bytes
	coverBytes, err := GetDramaCoverBytes(7698294694886788121)
	if err != nil || len(coverBytes) == 0 {
		t.Fatalf("failed to get cover bytes for 7698294694886788121: %v", err)
	}
	t.Logf("Successfully fetched cover bytes for 7698294694886788121: %d bytes", len(coverBytes))
}
