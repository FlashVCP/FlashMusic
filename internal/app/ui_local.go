package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/local"
	"netease-music-mygo/internal/netease"
)

func (a *app) localPage(c *ui.Context) {
	header := func(c *ui.Context) {
		a.localHeader(c)
	}

	if len(a.localTracks) == 0 {
		top, bottom := a.insets(c)
		ui.Scroll(c.Key("local-empty")).TrackScroll(&a.pageScroll).Grow(1).
			Padding(top+pageGap, pagePad, bottom+48, pagePad).
			ScrollbarInsets(top, 0, bottom, 0).Children(func() {
			a.startMark(c)
			defer a.endMark(c)
			header(c)
			a.localEmpty(c)
		})
		return
	}

	a.trackRows(c, a.localTracks, true, header)
}

func (a *app) localHeader(c *ui.Context) {
	p := a.pal
	ui.Column(c).Padding(16, 0, 24, 0).Gap(10).Children(func() {
		ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(56, 56).Radius(14).Background(p.accent.Alpha(0.12)).Center().Children(func() {
				ui.Icon(c, iconFolder).Size(28, 28).TextColor(p.accent)
			})
			ui.Column(c).Grow(1).Gap(3).Children(func() {
				ui.Text(c, "本地音乐").FontSize(28).FontWeight(800).LetterSpacing(-0.5)
				sub := fmt.Sprintf("%d 首歌曲", len(a.localTracks))
				if a.localScanning {
					sub = "正在扫描本地音乐..."
				} else if len(a.localDirs) > 0 {
					sub += fmt.Sprintf(" · 已扫描 %d 个文件夹", len(a.localDirs))
				}
				ui.Text(c, sub).FontSize(13).TextColor(p.muted)
			})
		})

		ui.Row(c).Gap(10).Margin(6, 0, 0, 0).Children(func() {
			if len(a.localTracks) > 0 {
				if pillButton(c.Key("play-all"), iconPlay, "播放全部", p.accent, p.onAccent).Clicked() {
					if a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(a.localTracks, 0)
				}
				if pillButton(c.Key("shuffle-all"), iconShuffle, "随机播放", p.hover.Over(p.bg), p.text).Clicked() {
					if !a.pl.Shuffle() {
						a.pl.ToggleShuffle()
					}
					a.pl.PlayQueue(a.localTracks, 0)
				}
			}
			if pillButton(c.Key("add-folder"), iconPlus, "添加文件夹", p.hover.Over(p.bg), p.text).Clicked() {
				a.addLocalFolder()
			}
			if len(a.localDirs) > 0 {
				label := "重新扫描"
				if a.localScanning {
					label = "扫描中..."
				}
				if pillButton(c.Key("rescan"), iconReload, label, p.hover.Over(p.bg), p.text).Disabled(a.localScanning).Clicked() {
					a.scanLocalDirs()
				}
			}
		})

		if len(a.localDirs) > 0 {
			ui.Row(c).Gap(8).Margin(4, 0, 0, 0).Children(func() {
				ui.Text(c, "扫描路径:").FontSize(11.5).TextColor(p.faint)
				for _, dir := range a.localDirs {
					ui.Text(c, filepath.Base(dir)).FontSize(11.5).TextColor(p.muted).
						Padding(1, 6).Radius(4).Background(p.hover)
				}
			})
		}
	})
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
		var allTracks []netease.Track
		for _, dir := range dirs {
			tracks, err := local.ScanDirectory(dir)
			if err != nil {
				continue
			}
			for _, lt := range tracks {
				allTracks = append(allTracks, lt.ToTrack())
			}
		}
		a.post(func() {
			a.localScanning = false
			a.localTracks = allTracks
		})
	}()
}
