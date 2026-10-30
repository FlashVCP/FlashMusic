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

func (a *app) songApplePopover(c *ui.Context, anchor ui.Element, tr *netease.Track, pid int64) {
	pop := popoverColors(a.pal.dark)
	ui.PopoverBase(c.Key("apple-song-menu"), anchor, &a.songMenuOpen, func(panel ui.Element) {
		panel.AttachTo(anchor, ui.AnchorBottomEnd, ui.AnchorTopEnd).Margin(0, 0, 4, 0).
			Width(224).Padding(6).Gap(2).Radius(12).
			Background(pop.fill).Border(0.5, pop.edge).
			Shadow(0, 16, 40, -4, ui.RGBA(0, 0, 0, 0.28)).
			Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.12)).
			Role(ui.RoleMenu).FocusGroup(ui.Vertical).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.95, Y: -4}, Exit: &ui.Motion{Scale: 0.97}})

		menuItem := func(icon *ui.SVG, label string, action func()) {
			b := ui.ButtonBase(c.Key(label)).Role(ui.RoleMenuItem).Label(label).Height(30).PaddingX(8).Gap(9).
				Radius(7).AlignItems(ui.Center).Justify(ui.Start).FocusRing(false)
			if b.Hovered() || b.FocusVisible() {
				b.Background(a.pal.press)
			}
			b.Children(func() {
				if icon != nil {
					ui.Icon(c, icon).Size(14, 14).TextColor(a.pal.muted)
				}
				ui.Text(c, label).FontSize(13).FontWeight(500).TextColor(a.pal.text).Grow(1).SingleLine()
			})
			if b.Clicked() {
				a.songMenuOpen = false
				action()
			}
		}

		divider := func() {
			ui.Box(c).Height(0.5).Margin(3, 4).Background(pop.edge)
		}

		// 1. Playback controls
		menuItem(iconPlay, "立即播放", func() {
			a.pl.PlaySingle(*tr)
		})
		menuItem(iconNext, "下一首播放", func() {
			a.pl.PlayNext(*tr)
			a.toast = "已添加为下一首播放"
		})
		menuItem(iconQueue, "添加到待播末尾", func() {
			a.pl.AppendQueue(*tr)
			a.toast = "已添加到播放队列末尾"
		})
		divider()

		// 2. Favorite
		liked := a.liked[tr.ID]
		likeIcon, likeText := iconHeart, "收藏到我喜欢的音乐"
		if liked {
			likeIcon, likeText = iconHeartFill, "从我喜欢的音乐中移除"
		}
		menuItem(likeIcon, likeText, func() {
			a.toggleLike(tr)
		})
		divider()

		// 3. Navigation
		if source.IsLocal(tr.ID) {
			lt, _ := source.GetLocal(tr.ID)
			ar := tr.ArtistNames()
			if ar != "" && ar != "未知歌手" {
				menuItem(iconUser, "查看歌手："+ar, func() {
					a.openLocalArtist(ar)
				})
			}
			if tr.Album.Name != "" && tr.Album.Name != "未知专辑" {
				menuItem(iconNote, "查看专辑："+tr.Album.Name, func() {
					a.openLocalAlbum(tr.Album.Name)
				})
			}
			divider()

			if lt != nil {
				menuItem(iconFolder, "在资源管理器中显示", func() {
					mygo.Shell.ShowItemInFolder(lt.Path)
				})
				menuItem(iconNote, "歌曲属性与详细信息…", func() {
					a.showSongProperties(lt)
				})
				divider()
				menuItem(iconReload, "复制文件路径", func() {
					copyText(lt.Path)
					a.toast = "已复制物理文件路径"
				})
			}
			menuItem(nil, "复制歌名", func() { copyText(tr.Name) })
			menuItem(nil, "复制歌名和歌手", func() { copyText(tr.Name + " - " + tr.ArtistNames()) })
			divider()

			menuItem(iconTrash, "从曲库移除 (不删文件)", func() {
				a.removeLocalTrack(tr.ID)
			})
		} else {
			// Online track items
			for _, ar := range tr.Artists {
				if ar.ID != 0 {
					arID, arName := ar.ID, ar.Name
					menuItem(iconUser, "查看歌手："+arName, func() { a.openArtist(arID) })
				}
			}
			if tr.Album.ID != 0 {
				alID, alName := tr.Album.ID, tr.Album.Name
				menuItem(iconNote, "查看专辑："+alName, func() { a.openPage(route{page: pageAlbum, id: alID}) })
			}
			divider()
			menuItem(nil, "复制歌名", func() { copyText(tr.Name) })
			menuItem(nil, "复制链接", func() { copyText(tr.URL()) })
		}
	})
}
