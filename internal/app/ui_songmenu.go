package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/source"
)

// copyText puts text on the clipboard; tests replace it.
var copyText = func(text string) error { return mygo.Clipboard.Write(transfer.TextData(text)) }

// songMenu is three dots that open the menu of tr.
func (a *app) songMenu(c *ui.Context, tr *netease.Track, size float32, color ui.Color) ui.Element {
	b := iconButton(c, iconMore, size, color, "更多")
	b.Menu(a.songItems(tr, 0))
	return b
}

// songItems builds the menu of what can be done with tr, behind its dots
// or a right click on its row: copying its title, its title with its
// artists, or the link to its page, opening its artists' or album's, and,
// signed in, adding it to a playlist or taking it out of pid, the playlist
// it is listed in, if set.
func (a *app) songItems(tr *netease.Track, pid int64) func(m *ui.Menu) {
	return func(m *ui.Menu) {
		if source.IsLocal(tr.ID) {
			a.localSongItems(m, tr)
			return
		}
		if tr.ProgramID != 0 {
			a.programItems(m, tr)
			return
		}

		// 1. Playback controls for online tracks
		if m.Item("立即播放").Chosen() {
			a.pl.PlaySingle(*tr)
		}
		if m.Item("下一首播放").Chosen() {
			a.pl.PlayNext(*tr)
			a.toast = "已添加为下一首播放"
		}
		if m.Item("添加到待播清单末尾").Chosen() {
			a.pl.AppendQueue(*tr)
			a.toast = "已添加到播放队列末尾"
		}
		m.Separator()

		// 2. Favorite / Like
		liked := a.liked[tr.ID]
		likeLabel := "收藏到我喜欢的音乐"
		if liked {
			likeLabel = "从我喜欢的音乐中移除"
		}
		if m.Item(likeLabel).Chosen() {
			a.toggleLike(tr)
		}
		m.Separator()

		if m.Item("复制歌名").Chosen() {
			copyText(tr.Name)
		}
		if m.Item("复制歌名和歌手").Chosen() {
			copyText(tr.Name + " - " + tr.ArtistNames())
		}
		if m.Item("复制链接").Chosen() {
			copyText(tr.URL())
		}

		// Its artists' and album's pages, those NetEase has.
		linked := tr.Album.ID != 0
		for _, ar := range tr.Artists {
			linked = linked || ar.ID != 0
		}
		if linked {
			m.Separator()
		}
		for _, ar := range tr.Artists {
			if ar.ID != 0 && m.Item("查看歌手："+ar.Name).Chosen() {
				a.openArtist(ar.ID)
			}
		}
		if tr.Album.ID != 0 && m.Item("查看专辑："+tr.Album.Name).Chosen() {
			a.openPage(route{page: pageAlbum, id: tr.Album.ID})
		}
		a.playlistItems(m, tr, pid)
	}
}

// programItems builds the menu of an episode: copying its title or the
// link to its page, or opening its podcast's.
func (a *app) programItems(m *ui.Menu, tr *netease.Track) {
	if m.Item("复制节目名").Chosen() {
		copyText(tr.Name)
	}
	if m.Item("复制链接").Chosen() {
		copyText(tr.URL())
	}
	m.Separator()
	if m.Item("查看播客").Chosen() {
		a.openPage(route{page: pagePodcast, id: tr.PodcastID})
	}
}

func (a *app) localSongItems(m *ui.Menu, tr *netease.Track) {
	lt, _ := source.GetLocal(tr.ID)

	// 1. Playback controls
	if m.Item("立即播放").Chosen() {
		a.pl.PlaySingle(*tr)
	}
	if m.Item("下一首播放").Chosen() {
		a.pl.PlayNext(*tr)
		a.toast = "已添加为下一首播放"
	}
	if m.Item("添加到待播清单末尾").Chosen() {
		a.pl.AppendQueue(*tr)
		a.toast = "已添加到播放队列末尾"
	}
	m.Separator()

	// 2. Favorite / Like
	liked := a.liked[tr.ID]
	likeLabel := "收藏到我喜欢的音乐"
	if liked {
		likeLabel = "从我喜欢的音乐中移除"
	}
	if m.Item(likeLabel).Chosen() {
		a.toggleLike(tr)
	}
	m.Separator()

	// 3. Navigation (Artist / Album)
	artistName := tr.ArtistNames()
	if artistName != "" && artistName != "未知歌手" {
		if m.Item("查看歌手：" + artistName).Chosen() {
			a.openLocalArtist(artistName)
		}
	}
	if tr.Album.Name != "" && tr.Album.Name != "未知专辑" {
		if m.Item("查看专辑：" + tr.Album.Name).Chosen() {
			a.openLocalAlbum(tr.Album.Name)
		}
	}
	m.Separator()

	// 4. File operations & properties
	if lt != nil {
		if m.Item("在资源管理器中显示").Chosen() {
			mygo.Shell.ShowItemInFolder(lt.Path)
		}
		if m.Item("歌曲属性与详细信息…").Chosen() {
			a.showSongProperties(lt)
		}
		m.Separator()
		if m.Item("复制文件路径").Chosen() {
			copyText(lt.Path)
			a.toast = "已复制物理文件路径"
		}
	}
	if m.Item("复制歌名").Chosen() {
		copyText(tr.Name)
	}
	if m.Item("复制歌名和歌手").Chosen() {
		copyText(tr.Name + " - " + tr.ArtistNames())
	}
	m.Separator()

	// 5. Safe removal
	if m.Item("从本地曲库移除 (不删除本地文件)").Chosen() {
		a.removeLocalTrack(tr.ID)
	}
}
