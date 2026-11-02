package netease

import (
	"os"
	"testing"
)

func TestPlayableFollowsPrivilege(t *testing.T) {
	tracks := []Track{{ID: 1, Fee: 1}, {ID: 2, Fee: 8}, {ID: 3, Fee: 8}, {ID: 4, Fee: 1}}
	for _, tr := range tracks {
		if !tr.Playable() {
			t.Fatalf("expected all tracks to be Playable with multi-source unblock")
		}
	}
}

func TestVIPSongResolve(t *testing.T) {
	if os.Getenv("LIVE") == "" {
		t.Skip("skipping live network test without LIVE=1")
	}
	tr := Track{
		ID:      186016,
		Name:    "晴天",
		Artists: []Artist{{Name: "周杰伦"}},
		Album:   Album{Name: "叶惠美"},
	}
	st, err := SongURLForTrack(tr, "exhigh")
	if err != nil {
		t.Fatalf("failed to resolve VIP song: %v", err)
	}
	t.Logf("Resolved VIP song: URL=%s, size=%d, type=%s", st.URL, st.Size, st.Type)
	if st.URL == "" || st.Size < 500000 {
		t.Fatalf("invalid stream data: %+v", st)
	}
}
