package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/local"
	"netease-music-mygo/internal/netease"
	"netease-music-mygo/internal/source"
)

func (a *app) localPage(c *ui.Context) {
	if len(a.localTracks) == 0 {
		top, bottom := a.insets(c)
		ui.Scroll(c.Key("local-empty")).TrackScroll(&a.pageScroll).Grow(1).
			Padding(top+pageGap, pagePad, bottom+48, pagePad).
			ScrollbarInsets(top, 0, bottom, 0).Children(func() {
			a.startMark(c)
			defer a.endMark(c)
			a.localHeader(c)
			a.localEmpty(c)
		})
		return
	}

		// Filtered sub-view (Album / Artist / Folder details)
		if a.localFilterType != "" && len(a.localFilterSongs) > 0 {
			songs := a.localFilterSongs
			if a.localQuery != "" {
				songs = source.FilterTracks(songs, a.localQuery)
			}
			a.trackRows(c, songs, true, func(c *ui.Context) {
				a.localFilteredHeader(c)
			})
			return
		}

		header := func(c *ui.Context) {
			a.localHeader(c)
		}

		switch a.localTab {
		case 1:
			a.localAlbumsView(c, header)
		case 2:
			a.localArtistsView(c, header)
		case 3:
			a.localFoldersView(c, header)
		case 4:
			songs := a.localRecentTracks
			if len(songs) == 0 && len(a.localTracks) > 0 {
				songs = a.localTracks
			}
			if a.localQuery != "" {
				songs = source.FilterTracks(songs, a.localQuery)
			}
			a.trackRows(c, songs, true, header)
		default:
			songs := a.localTracks
			if a.localQuery != "" {
				songs = source.FilterTracks(songs, a.localQuery)
			}
			a.trackRows(c, songs, true, header)
		}
	}

	func (a *app) localHeader(c *ui.Context) {
		p := a.pal
		ui.Column(c).Padding(16, 0, 24, 0).Gap(20).Children(func() {
			// Row 1: Title and Stats on Left, Segmented Tabs on Right
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(52, 52).Radius(14).Background(p.accent.Alpha(0.12)).Center().Children(func() {
						ui.Icon(c, iconFolder).Size(26, 26).TextColor(p.accent)
					})
					ui.Column(c).Gap(3).Children(func() {
						titleText := "本地音乐"
						switch a.localTab {
						case 4:
							titleText = "本地音乐 · 最新添加"
						case 1:
							titleText = "本地音乐 · 按专辑"
						case 2:
							titleText = "本地音乐 · 按歌手"
						case 3:
							titleText = "本地音乐 · 按文件夹"
						}
						ui.Text(c, titleText).FontSize(26).FontWeight(800).LetterSpacing(-0.5)

						sub := fmt.Sprintf("%d 首歌曲", len(a.localTracks))
						if a.localScanning {
							sub = "正在扫描本地音乐..."
						} else if a.localQuery != "" {
							sub = fmt.Sprintf("实时搜索 %q 结果", a.localQuery)
						} else {
							if len(a.localDirs) > 0 {
								sub += fmt.Sprintf(" · 已扫描 %d 个文件夹", len(a.localDirs))
							}
							if a.localDupesCount > 0 {
								sub += fmt.Sprintf(" · 智能去重 %d 首副本", a.localDupesCount)
							}
						}
						ui.Text(c, sub).FontSize(12.5).TextColor(p.muted)
					})
				})

				ui.Spacer(c)

				// Categorization Tabs (Float Right)
				if len(a.localTracks) > 0 {
					segmented(c.Key("local-tabs"), &a.localTab, "全部单曲", "按专辑", "按歌手", "按文件夹", "最新添加")
				}
			})

			// Row 2: Clean Action Toolbar (Primary Capsule + Frosted Action Buttons + Search Box)
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				if len(a.localTracks) > 0 {
					playSongs := a.localTracks
					if a.localTab == 4 && len(a.localRecentTracks) > 0 {
						playSongs = a.localRecentTracks
					}
					if a.localQuery != "" {
						playSongs = source.FilterTracks(playSongs, a.localQuery)
					}

					playAll := pillButton(c.Key("play-all"), iconPlay, "播放全部", p.accent, p.onAccent).
						Shadow(0, 4, 14, 0, p.accent.Alpha(0.32))
					if playAll.Clicked() && len(playSongs) > 0 {
						if a.pl.Shuffle() {
							a.pl.ToggleShuffle()
						}
						a.pl.PlayQueue(playSongs, 0)
					}

					shuffleAll := pillButton(c.Key("shuffle-all"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).
						Border(0.5, p.hairline)
					if shuffleAll.Clicked() && len(playSongs) > 0 {
						if !a.pl.Shuffle() {
							a.pl.ToggleShuffle()
						}
						a.pl.PlayQueue(playSongs, 0)
					}

					// Divider
					ui.Box(c).Width(1).Height(20).Background(p.hairline).MarginX(4)
				}

				addFolder := pillButton(c.Key("add-folder"), iconPlus, "添加文件夹", p.hover.Over(p.bg), p.text).
					Border(0.5, p.hairline)
				if addFolder.Clicked() {
					a.addLocalFolder()
				}

				if len(a.localDirs) > 0 || len(a.localTracks) > 0 {
					label := "重新扫描"
					if a.localScanning {
						label = "扫描中..."
					}
					rescan := pillButton(c.Key("rescan"), iconReload, label, p.hover.Over(p.bg), p.text).
						Border(0.5, p.hairline).Disabled(a.localScanning)
					if rescan.Clicked() {
						a.scanLocalDirs()
					}

					clearLib := pillButton(c.Key("clear-lib"), iconTrash, "清空曲库", ui.RGBA(255, 60, 60, 0.08), ui.Hex("#ff4d4f")).
						Border(0.5, ui.RGBA(255, 60, 60, 0.2))
					if clearLib.Clicked() {
						a.clearLocalLibrary()
					}
				}

				ui.Spacer(c)

				// Real-time Local Search Box
				if len(a.localTracks) > 0 {
					searchField := ui.Row(c).Height(32).PaddingX(10).Gap(7).Radius(9).AlignItems(ui.Center).Width(240).
						Background(p.hover).Border(1, p.hairline)
					searchField.Children(func() {
						ui.Icon(c, iconSearch).Size(14, 14).TextColor(p.muted)
						ui.TextInput(c.Key("local-search-input"), &a.localQuery).
							Placeholder("搜索本地歌曲 / 歌手...").FontSize(12.5).Label("搜索本地音乐").
							Grow(1).MinWidth(0)
					if a.localQuery != "" {
						if iconButton(c.Key("clear-local-search"), iconX, 12, p.muted, "清除").Size(18, 18).Clicked() {
							a.localQuery = ""
						}
					}
				})
			}
		})
	})
}

func (a *app) localFilteredHeader(c *ui.Context) {
	p := a.pal
	ui.Column(c).Padding(16, 0, 24, 0).Gap(14).Children(func() {
		if pillButton(c.Key("back-to-all"), iconBack, "返回全部本地音乐", p.hover.Over(p.bg), p.text).Clicked() {
			a.localFilterType = ""
			a.localFilterName = ""
			a.localFilterSongs = nil
			return
		}

		iconTarget := iconNote
		typeLabel := "专辑"
		switch a.localFilterType {
		case "artist":
			iconTarget = iconUser
			typeLabel = "歌手"
		case "folder":
			iconTarget = iconFolder
			typeLabel = "文件夹"
		}

		ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
			if len(a.localFilterSongs) > 0 && a.localFilterSongs[0].Album.PicURL != "" {
				a.cover(c, a.localFilterSongs[0].Album.PicURL, 140, 72, 10)
			} else {
				ui.Box(c).Size(72, 72).Radius(10).Background(p.hover).Center().Children(func() {
					ui.Icon(c, iconTarget).Size(32, 32).TextColor(p.muted)
				})
			}
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Text(c, typeLabel+" · "+a.localFilterName).FontSize(24).FontWeight(700).LetterSpacing(-0.4)
				ui.Textf(c, "共 %d 首歌曲", len(a.localFilterSongs)).FontSize(13).TextColor(p.muted)
			})
		})

		ui.Row(c).Gap(10).Margin(4, 0, 0, 0).Children(func() {
			if pillButton(c.Key("filter-play"), iconPlay, "播放这组歌曲", p.accent, p.onAccent).Clicked() {
				if a.pl.Shuffle() {
					a.pl.ToggleShuffle()
				}
				a.pl.PlayQueue(a.localFilterSongs, 0)
			}
			if pillButton(c.Key("filter-shuffle"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
				if !a.pl.Shuffle() {
					a.pl.ToggleShuffle()
				}
				a.pl.PlayQueue(a.localFilterSongs, 0)
			}
		})
	})
}

func (a *app) localAlbumsView(c *ui.Context, header func(c *ui.Context)) {
	top, bottom := a.insets(c)
	albums := a.localAlbums
	if len(albums) == 0 && len(a.localTracks) > 0 {
		albums = source.GroupLocalByAlbum()
		a.localAlbums = albums
	}
	if a.localQuery != "" {
		q := strings.ToLower(strings.TrimSpace(a.localQuery))
		var filtered []*source.LocalAlbum
		for _, al := range albums {
			if strings.Contains(strings.ToLower(al.Name), q) || strings.Contains(strings.ToLower(al.Artist), q) {
				filtered = append(filtered, al)
			}
		}
		albums = filtered
	}

	ui.Scroll(c.Key("local-albums")).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad, bottom+48, pagePad).
		ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		header(c)

		ui.Grid(c).Columns(a.columns(c, 156)).GapX(20).GapY(28).Children(func() {
			for _, al := range albums {
				a.localAlbumCard(c, al)
			}
		})
	})
}

func (a *app) localAlbumCard(c *ui.Context, al *source.LocalAlbum) {
	p := a.pal
	card := ui.ButtonBase(c.Key(al.Name)).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(al.Name)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(10)
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -3}
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(lift, ui.Snappy)
		art.Children(func() { a.coverFill(c, al.CoverURL, 400, 10) })

		ui.Text(c, al.Name).FontSize(13).FontWeight(500).MaxLines(2).LineHeight(1.35)
		artist := al.Artist
		if artist == "" {
			artist = "未知歌手"
		}
		ui.Textf(c, "%s · %d 首", artist, len(al.Tracks)).FontSize(12).TextColor(p.muted).SingleLine().Margin(-6, 0, 0, 0)
	})
	if card.Clicked() {
		a.localFilterType = "album"
		a.localFilterName = al.Name
		a.localFilterSongs = al.Tracks
		a.pageScroll = ui.ScrollState{}
	}
}

func (a *app) localArtistsView(c *ui.Context, header func(c *ui.Context)) {
	top, bottom := a.insets(c)
	artists := a.localArtists
	if len(artists) == 0 && len(a.localTracks) > 0 {
		artists = source.GroupLocalByArtist()
		a.localArtists = artists
	}
	if a.localQuery != "" {
		q := strings.ToLower(strings.TrimSpace(a.localQuery))
		var filtered []*source.LocalArtist
		for _, ar := range artists {
			if strings.Contains(strings.ToLower(ar.Name), q) {
				filtered = append(filtered, ar)
			}
		}
		artists = filtered
	}

	ui.Scroll(c.Key("local-artists")).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad, bottom+48, pagePad).
		ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		header(c)

		ui.Grid(c).Columns(a.columns(c, 140)).GapX(20).GapY(28).Children(func() {
			for _, ar := range artists {
				a.localArtistCard(c, ar)
			}
		})
	})
}

func (a *app) localArtistCard(c *ui.Context, ar *source.LocalArtist) {
	p := a.pal
	card := ui.ButtonBase(c.Key(ar.Name)).Column().Gap(10).AlignItems(ui.Stretch).FocusRing(false).Label(ar.Name)
	hovered := card.Hovered()
	card.Children(func() {
		art := ui.Box(c).AspectRatio(1).Radius(999)
		lift := ui.Pose{}
		if hovered {
			lift = ui.Pose{Y: -3}
			art.Shadow(0, 12, 28, -6, p.cardShade)
		} else {
			art.Shadow(0, 2, 8, -2, p.cardShade)
		}
		art.AnimatePose(lift, ui.Snappy)
		art.Children(func() { a.coverFill(c, ar.CoverURL, 400, 999) })

		ui.Text(c, ar.Name).FontSize(13.5).FontWeight(600).SingleLine().AlignSelf(ui.Center)
		ui.Textf(c, "%d 首歌曲", len(ar.Tracks)).FontSize(12).TextColor(p.muted).SingleLine().AlignSelf(ui.Center).Margin(-6, 0, 0, 0)
	})
	if card.Clicked() {
		a.localFilterType = "artist"
		a.localFilterName = ar.Name
		a.localFilterSongs = ar.Tracks
		a.pageScroll = ui.ScrollState{}
	}
}

func (a *app) localFoldersView(c *ui.Context, header func(c *ui.Context)) {
	top, bottom := a.insets(c)
	folders := a.localFolders
	if len(folders) == 0 && len(a.localTracks) > 0 {
		folders = source.GroupLocalByFolder()
		a.localFolders = folders
	}
	if a.localQuery != "" {
		q := strings.ToLower(strings.TrimSpace(a.localQuery))
		var filtered []*source.LocalFolder
		for _, f := range folders {
			if strings.Contains(strings.ToLower(f.Name), q) || strings.Contains(strings.ToLower(f.Path), q) {
				filtered = append(filtered, f)
			}
		}
		folders = filtered
	}

	ui.Scroll(c.Key("local-folders")).TrackScroll(&a.pageScroll).Grow(1).
		Padding(top+pageGap, pagePad, bottom+48, pagePad).
		ScrollbarInsets(top, 0, bottom, 0).Children(func() {
		a.startMark(c)
		defer a.endMark(c)
		header(c)

		ui.Grid(c).Columns(a.columns(c, 180)).GapX(20).GapY(24).Children(func() {
			for _, f := range folders {
				a.localFolderCard(c, f)
			}
		})
	})
}

func (a *app) localFolderCard(c *ui.Context, f *source.LocalFolder) {
	p := a.pal
	card := ui.ButtonBase(c.Key(f.Path)).Padding(14).Radius(12).FocusRing(false).Label(f.Name).
		ContextMenu(func(m *ui.Menu) {
			if m.Item("在资源管理器中打开").Chosen() {
				mygo.Shell.OpenPath(f.Path)
			}
			m.Separator()
			if m.Item("从曲库移除此文件夹 (保留本地文件)").Chosen() {
				a.removeLocalFolder(f.Path)
			}
		})
	hovered := card.Hovered()
	bg := p.hover
	if hovered {
		bg = p.press
	}
	card.Background(card.AnimateColor("bg", bg, ui.Animation{Duration: 100 * time.Millisecond}))
	card.Children(func() {
		ui.Column(c).Grow(1).Gap(8).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconFolder).Size(22, 22).TextColor(p.accent)
				ui.Text(c, f.Name).FontSize(14).FontWeight(600).SingleLine().Grow(1).Shrink(1)
			})
			ui.Text(c, f.Path).FontSize(11.5).TextColor(p.faint).MaxLines(1)
			ui.Textf(c, "包含 %d 首歌曲", len(f.Tracks)).FontSize(12).TextColor(p.muted)
		})
	})
	if card.Clicked() {
		a.localFilterType = "folder"
		a.localFilterName = f.Name
		a.localFilterSongs = f.Tracks
		a.pageScroll = ui.ScrollState{}
	}
}

func (a *app) localEmpty(c *ui.Context) {
	p := a.pal
	ui.Column(c).Fill().Padding(60, 20, 60, 20).Gap(16).AlignItems(ui.Center).Justify(ui.Center).Children(func() {
		ui.Box(c).Size(80, 80).Radius(20).Background(p.hover).Center().Children(func() {
			ui.Icon(c, iconNote).Size(36, 36).TextColor(p.muted)
		})
		ui.Column(c).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "还没有添加本地音乐").FontSize(18).FontWeight(600)
			ui.Text(c, "点击下方按钮选择包含 MP3 或 FLAC 格式音频的本地文件夹").FontSize(13).TextColor(p.muted)
		})
		if pillButton(c.Key("empty-add"), iconPlus, "选择本地文件夹", p.accent, p.onAccent).Clicked() {
			a.addLocalFolder()
		}
	})
}

func (a *app) addLocalFolder() {
	go func() {
		dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Title:     "选择本地音乐文件夹",
			Directory: true,
			Multiple:  false,
		})
		if err != nil || len(dirs) == 0 {
			return
		}
		newDir := filepath.Clean(dirs[0])
		a.post(func() {
			for _, d := range a.localDirs {
				if d == newDir {
					return
				}
			}
			a.localDirs = append(a.localDirs, newDir)
			a.settings.LocalDirs = a.localDirs
			a.settings.save()
			a.scanLocalDirs()
		})
	}()
}

func (a *app) scanLocalDirs() {
	if a.localScanning {
		return
	}
	a.localScanning = true

	// 1. Canonicalize directory list (prune nested redundant folders)
	a.localDirs = source.CanonicalizeDirs(a.localDirs)
	a.settings.LocalDirs = a.localDirs
	a.settings.save()

	dirs := append([]string(nil), a.localDirs...)
	if len(dirs) == 0 {
		if home, err := os.UserHomeDir(); err == nil {
			musicDir := filepath.Join(home, "Music")
			if _, err := os.Stat(musicDir); err == nil {
				dirs = []string{musicDir}
				a.localDirs = dirs
				a.settings.LocalDirs = dirs
				a.settings.save()
			}
		}
	}

	go func() {
		// Prune any deleted files from registry first
		source.PruneMissingLocal()

		for _, dir := range dirs {
			local.ScanDirectory(dir)
		}

		rawTracks := source.GetAllLocal()
		dedupedLocal, dupesCount := source.DeduplicateLocal(rawTracks)

			var allTracks []netease.Track
			for _, lt := range dedupedLocal {
				allTracks = append(allTracks, lt.ToTrack())
			}
			recentTracks := source.SortLocalByRecent(dedupedLocal)
			albums := source.GroupLocalByAlbum()
			artists := source.GroupLocalByArtist()
			folders := source.GroupLocalByFolder()
			a.post(func() {
				a.localScanning = false
				a.localTracks = allTracks
				a.localRecentTracks = recentTracks
				a.localDupesCount = dupesCount
				a.localAlbums = albums
				a.localArtists = artists
				a.localFolders = folders
			if a.localFilterType != "" {
				// Refresh active filter
				switch a.localFilterType {
				case "album":
					for _, al := range albums {
						if al.Name == a.localFilterName {
							a.localFilterSongs = al.Tracks
							break
						}
					}
				case "artist":
					for _, ar := range artists {
						if ar.Name == a.localFilterName {
							a.localFilterSongs = ar.Tracks
							break
						}
					}
				case "folder":
					for _, f := range folders {
						if f.Name == a.localFilterName {
							a.localFilterSongs = f.Tracks
							break
						}
					}
					}
				}
			})
		}()
	}

func (a *app) removeLocalTrack(id int64) {
	source.RemoveLocal(id)
	for i, t := range a.localTracks {
		if t.ID == id {
			a.localTracks = append(a.localTracks[:i], a.localTracks[i+1:]...)
			break
		}
	}
	if a.localFilterType != "" {
		for i, t := range a.localFilterSongs {
			if t.ID == id {
				a.localFilterSongs = append(a.localFilterSongs[:i], a.localFilterSongs[i+1:]...)
				break
			}
		}
	}
	a.localAlbums = source.GroupLocalByAlbum()
	a.localArtists = source.GroupLocalByArtist()
	a.localFolders = source.GroupLocalByFolder()
	a.toast = "已从本地曲库移除（本地文件未受影响）"
	a.redraw()
}

func (a *app) removeLocalFolder(dir string) {
	cleanDir := filepath.Clean(dir)
	var newDirs []string
	for _, d := range a.localDirs {
		if filepath.Clean(d) != cleanDir {
			newDirs = append(newDirs, d)
		}
	}
	a.localDirs = newDirs
	a.settings.LocalDirs = newDirs
	a.settings.save()

	source.RemoveLocalByFolder(cleanDir)

	rawTracks := source.GetAllLocal()
	dedupedLocal, dupes := source.DeduplicateLocal(rawTracks)
	var allTracks []netease.Track
	for _, lt := range dedupedLocal {
		allTracks = append(allTracks, lt.ToTrack())
	}
	a.localTracks = allTracks
	a.localDupesCount = dupes
	a.localAlbums = source.GroupLocalByAlbum()
	a.localArtists = source.GroupLocalByArtist()
	a.localFolders = source.GroupLocalByFolder()

	if a.localFilterType == "folder" && a.localFilterName == filepath.Base(cleanDir) {
		a.localFilterType = ""
		a.localFilterName = ""
		a.localFilterSongs = nil
	}
	a.toast = "已从曲库移除该文件夹（本地文件保持完好）"
	a.redraw()
}

func (a *app) clearLocalLibrary() {
	source.ClearLocalRegistry()
	a.localTracks = nil
	a.localAlbums = nil
	a.localArtists = nil
	a.localFolders = nil
	a.localDirs = nil
	a.localDupesCount = 0
	a.localFilterType = ""
	a.localFilterName = ""
	a.localFilterSongs = nil
	a.settings.LocalDirs = nil
	a.settings.save()
	a.toast = "已清空本地曲库（电脑中的音乐文件保持完好）"
	a.redraw()
}

func (a *app) openLocalArtist(name string) {
	a.localTab = 2 // 歌手视图
	a.localFilterType = "artist"
	a.localFilterName = name
	for _, ar := range a.localArtists {
		if ar.Name == name {
			a.localFilterSongs = ar.Tracks
			break
		}
	}
	a.open(route{page: pageLocal})
}

func (a *app) openLocalAlbum(name string) {
	a.localTab = 1 // 专辑视图
	a.localFilterType = "album"
	a.localFilterName = name
	for _, al := range a.localAlbums {
		if al.Name == name {
			a.localFilterSongs = al.Tracks
			break
		}
	}
	a.open(route{page: pageLocal})
}

func (a *app) showSongProperties(lt *source.LocalTrack) {
	a.propTrack = lt
	a.propOpen = true
	a.redraw()
}

func (a *app) songPropertiesDialog(c *ui.Context) {
	if !a.propOpen || a.propTrack == nil {
		return
	}
	p := a.pal
	t := a.propTrack
	ui.DialogBase(c.Key("song-properties"), &a.propOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.45))
		panel.Width(440).Padding(24).Radius(18).Gap(16).
			Background(p.bg).Border(1, p.hairline).Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.4)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})

		// Header
		ui.Row(c).Gap(14).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(40, 40).Radius(10).Background(p.accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, iconNote).Size(20, 20).TextColor(p.accent)
			})
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, "歌曲属性与详细信息").FontSize(17).FontWeight(700)
				ui.Text(c, "音频规格与物理存储").FontSize(12).TextColor(p.muted)
			})
		})

		// Details rows
		item := func(label, val string) {
			ui.Row(c).Padding(6, 0).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(13).FontWeight(600).TextColor(p.muted).Width(80)
				ui.Text(c, val).FontSize(13).FontWeight(500).TextColor(p.text).Grow(1).Shrink(1).SingleLine()
			})
		}

		ui.Column(c).Padding(8, 12).Radius(10).Background(p.hover).Dividers(1, p.hairline).Children(func() {
			item("歌曲名称", t.Title)
			item("音乐家", t.Artist)
			item("唱片专辑", t.Album)
			formatStr := strings.ToUpper(t.Format)
			if t.Format == "flac" {
				if t.BitDepth >= 24 || t.SampleRate >= 88200 {
					formatStr += " (Hi-Res 高解析)"
				} else {
					formatStr += " (无损 CD 级)"
				}
			}
			item("音频格式", formatStr)
			if t.SampleRate > 0 {
				item("采样率", fmt.Sprintf("%d Hz (%.1f kHz)", t.SampleRate, float64(t.SampleRate)/1000.0))
			}
			if t.BitDepth > 0 {
				item("采样位深", fmt.Sprintf("%d bit", t.BitDepth))
			}
			if t.Bitrate > 0 {
				item("比特率", fmt.Sprintf("%d kbps", t.Bitrate))
			}
			if t.Duration > 0 {
				item("音频时长", clock(time.Duration(t.Duration)*time.Millisecond))
			}
			if t.Size > 0 {
				item("文件大小", megabytes(t.Size))
			}
		})

		// File Path
		ui.Column(c).Gap(4).Children(func() {
			ui.Text(c, "文件物理路径").FontSize(11.5).TextColor(p.faint)
			ui.Text(c, t.Path).FontSize(12).TextColor(p.muted).MaxLines(2).LineHeight(1.35)
		})

		// Actions
		ui.Row(c).Gap(10).Justify(ui.End).Children(func() {
			if pillButton(c.Key("prop-open-folder"), iconFolder, "在文件夹中显示", p.hover.Over(p.bg), p.text).Clicked() {
				mygo.Shell.ShowItemInFolder(t.Path)
			}
			if pillButton(c.Key("prop-copy-path"), nil, "复制路径", p.hover.Over(p.bg), p.text).Clicked() {
				copyText(t.Path)
				a.toast = "已复制物理文件路径"
			}
			if pillButton(c.Key("prop-close"), nil, "关闭", p.accent, p.onAccent).Clicked() {
				a.propOpen = false
			}
		})
	})
}
