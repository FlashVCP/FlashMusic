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

				// Step 1: Open Search Landing Page (query empty)
				step(0, func() {
					a.open(route{page: pageSearch, query: ""})
				})
				time.Sleep(2500 * time.Millisecond)
				shot("flashmusic_search_landing", a.win)

				// Step 2: Switch to "流行" category playlist view
				step(0, func() {
					a.searchCat = "流行"
					a.loadCategoryPlaylists("流行")
				})
				time.Sleep(3000 * time.Millisecond)
				shot("flashmusic_search_cat_pop", a.win)

				// Step 3: Search for "刀郎" (Replicating User Figure 3)
				step(0, func() {
					a.query = "刀郎"
					a.search("刀郎")
				})
				time.Sleep(3500 * time.Millisecond)
				shot("flashmusic_search_daolang", a.win)

			println("[FM_SHOT] All shots finished.")
			time.Sleep(500 * time.Millisecond)
			mygo.RunOnMain(mygo.App.Quit)
		}()
	}
}
