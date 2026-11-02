package app

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
)

// playVideo loads and starts playing the given media URL inside the embedded WebView2.
// It also pauses any playing music to prevent audio collision.
func (a *app) playVideo(url string) {
	if url == "" {
		return
	}
	a.videoWebURL = url

	// Pause background music
	if a.pl != nil && a.pl.Playing() {
		a.pl.TogglePlay()
	}

	if a.win == nil {
		return
	}

	if a.videoWeb == nil || a.videoWeb.IsDestroyed() {
		v, err := a.win.NewWebView(mygo.WebViewOptions{
			URL:             url,
			BackgroundColor: "#0b0b0e",
			Page: mygo.PageOptions{
				TrustedOrigins: []string{"*"},
			},
		})
		if err != nil {
			log.Printf("FlashMusic: create WebView failed: %v", err)
			return
		}
		a.videoWeb = v

		// Hook fullscreen messages from embedded player page via Title updates
		v.Page().OnPageTitleUpdated(func(e *mygo.TitleEvent) {
			if strings.HasPrefix(e.Title, "mygo:toggle-fullscreen:") {
				e.PreventDefault()
				a.win.ToggleFullScreen()
				a.redraw()
			} else if strings.HasPrefix(e.Title, "mygo:exit-fullscreen:") {
				e.PreventDefault()
				if a.win.IsFullScreen() {
					a.win.SetFullScreen(false)
					a.redraw()
				}
			}
		})
	} else {
		if err := a.videoWeb.Page().LoadURL(url); err != nil {
			log.Printf("FlashMusic: WebView LoadURL failed: %v", err)
		}
	}

	a.redraw()
}

// stopVideo stops the active video playback and resets the viewport.
func (a *app) stopVideo() {
	a.svPlaying = false
	a.dramaPlaying = false
	a.videoWebURL = ""
	if a.videoWeb != nil && !a.videoWeb.IsDestroyed() {
		_ = a.videoWeb.Page().LoadURL("about:blank")
	}
	a.redraw()
}
