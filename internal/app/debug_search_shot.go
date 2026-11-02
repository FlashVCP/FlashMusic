package app

import (
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
)

// When FM_SHOT_DIR is set, FlashMusic captures the landing search view and the query search view.
func init() {
	dir := os.Getenv("FM_SHOT_DIR")
	if dir == "" {
		return
	}
	shotHook = func(a *app) {
		go func() {
			shot := func(name string, w *mygo.Window) {
				png, err := w.CapturePage()
				if err == nil {
					_ = os.WriteFile(filepath.Join(dir, name+".png"), png, 0o644)
					println("[FM_SHOT] Captured:", name)
				} else {
					println("[FM_SHOT] Capture err:", err.Error())
				}
			}
			step := func(d time.Duration, fn func()) {
				time.Sleep(d)
				mygo.RunOnMain(func() { fn(); a.redraw() })
			}

			// Wait 3s for boot and data loading
			time.Sleep(3 * time.Second)

				// Step 1: Open 微短剧场 (pageDramas)
				step(0, func() {
					a.open(route{page: pageDramas})
				})
				time.Sleep(2500 * time.Millisecond)
				shot("flashmusic_vision_dramas", a.win)

				// Step 2: Open 潮流短视频 (pageShortVideos)
				step(0, func() {
					a.open(route{page: pageShortVideos})
				})
				time.Sleep(2500 * time.Millisecond)
				shot("flashmusic_vision_shortvideos", a.win)

				// Step 3: Open 院线影视 (pageCinema)
				step(0, func() {
					a.open(route{page: pageCinema})
				})
				time.Sleep(2500 * time.Millisecond)
				shot("flashmusic_vision_cinema", a.win)

				// Step 4: Open 互动直播 (pageLive)
				step(0, func() {
					a.open(route{page: pageLive})
				})
				time.Sleep(2000 * time.Millisecond)
				shot("flashmusic_vision_live", a.win)

				// Step 5: Open 语音空间 (pageVoiceRoom)
				step(0, func() {
					a.open(route{page: pageVoiceRoom})
				})
				time.Sleep(2000 * time.Millisecond)
				shot("flashmusic_vision_voiceroom", a.win)

			println("[FM_SHOT] All shots finished.")
			time.Sleep(500 * time.Millisecond)
			mygo.RunOnMain(mygo.App.Quit)
		}()
	}
}
