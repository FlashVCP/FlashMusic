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
				time.Sleep(2000 * time.Millisecond)

				// Step 2: Trigger Close Modal
				step(0, func() {
					a.closeConfirm.open = true
					a.closeConfirm.remember = true
				})
				time.Sleep(1000 * time.Millisecond)
				shot("flashmusic_close_confirm_modal", a.win)

			println("[FM_SHOT] All shots finished.")
			time.Sleep(500 * time.Millisecond)
			mygo.RunOnMain(mygo.App.Quit)
		}()
	}
}
