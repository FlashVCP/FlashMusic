package app

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// closeConfirmDialog manages the modal state for the close application confirmation.
type closeConfirmDialog struct {
	open     bool
	remember bool
}

// closeAppConfirmModal renders the AlgerMusicPlayer style close app confirmation modal.
// Matches AlgerMusicPlayer/src/renderer/components/common/CloseAppConfirmModal.vue exactly:
// - Title: "关闭应用" (closeApp)
// - Subtitle: "请选择关闭方式" (closeTitle)
// - Checkbox: "记住我的选择" (rememberChoice)
// - Actions: "取消" (cancel), "退出应用" (exitApp), "最小化到托盘" (minimizeToTray)
func (a *app) closeAppConfirmModal(c *ui.Context) {
	p := a.pal
	d := &a.closeConfirm

	ui.DialogBase(c.Key("close-app-confirm"), &d.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.40))

		// Modal card: Width 360, Radius 18, Clean Apple/Alger card styling
		panel.Width(360).Padding(24, 26, 24, 26).Radius(18).Gap(14).
			Background(p.card).Border(1, p.hairline).Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.45)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})

		// Header row: Title & Close 'X' button
		ui.Row(c.Key("close-modal-header")).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "关闭应用").FontSize(18).FontWeight(700).TextColor(p.text)
			ui.Spacer(c)

			// Top-right close button (X)
			closeBtn := ui.ButtonBase(c.Key("close-modal-x")).Size(26, 26).Radius(13).
				Background(p.hover).Center().FocusRing(false).Label("关闭弹窗")
			closeBtn.Children(func() {
				ui.Icon(c, iconX).Size(13, 13).TextColor(p.muted)
			})
			if closeBtn.Clicked() {
				d.open = false
			}
		})

		// Description subtitle
		ui.Text(c, "请选择关闭方式").FontSize(13).TextColor(p.muted)

		// Remember choice checkbox row
		ui.Row(c.Key("close-modal-remember")).AlignItems(ui.Center).Gap(8).Margin(6, 0, 8, 0).Children(func() {
			chkBtn := ui.ButtonBase(c.Key("remember-btn")).AlignItems(ui.Center).Gap(8).FocusRing(false).Label("记住我的选择")
			chkBtn.Children(func() {
				// Circle icon or check
				box := ui.Box(c).Size(18, 18).Radius(9).Center()
				if d.remember {
					box.Background(p.accent).Children(func() {
						ui.Icon(c, iconCheck).Size(11, 11).TextColor(p.onAccent)
					})
				} else {
					box.Background(ui.Transparent).Border(1.5, p.faint)
				}

				txtColor := p.muted
				if d.remember {
					txtColor = p.text
				}
				ui.Text(c, "记住我的选择").FontSize(12.5).TextColor(txtColor)
			})
			if chkBtn.Clicked() {
				d.remember = !d.remember
			}
		})

		// Action buttons row: [取消]  [退出应用]  [最小化到托盘]
		ui.Row(c.Key("close-modal-actions")).Gap(10).Justify(ui.End).AlignItems(ui.Center).Margin(4, 0, 0, 0).Children(func() {
			// 1. 取消
			cancelBtn := ui.ButtonBase(c.Key("close-cancel-btn")).Height(32).PaddingX(14).Radius(999).
				AlignItems(ui.Center).FocusRing(false).Label("取消")
			cancelBtn.Background(p.hover).TextColor(p.muted)
			if cancelBtn.Hovered() {
				cancelBtn.Background(p.press).TextColor(p.text)
			}
			cancelBtn.Children(func() {
				ui.Text(c, "取消").FontSize(12.5).FontWeight(500)
			})
			if cancelBtn.Clicked() {
				d.open = false
			}

			// 2. 退出应用
			exitBtn := ui.ButtonBase(c.Key("close-exit-btn")).Height(32).PaddingX(14).Radius(999).
				AlignItems(ui.Center).FocusRing(false).Label("退出应用")
			exitBtn.Background(p.hover).TextColor(p.text)
			if exitBtn.Hovered() {
				exitBtn.Background(p.press).TextColor(ui.Hex("#ef4444"))
			}
			exitBtn.Children(func() {
				ui.Text(c, "退出应用").FontSize(12.5).FontWeight(500)
			})
			if exitBtn.Clicked() {
				if d.remember {
					a.settings.CloseAction = "close"
					a.settings.save()
				}
				d.open = false
				a.quitting = true
				mygo.App.Quit()
			}

			// 3. 最小化到托盘 (Primary Highlight Button)
			miniBtn := ui.ButtonBase(c.Key("close-mini-btn")).Height(32).PaddingX(16).Radius(999).
				Background(p.accent).TextColor(p.onAccent).
				Shadow(0, 4, 12, 0, p.accent.Alpha(0.35)).
				AlignItems(ui.Center).FocusRing(false).Label("最小化到托盘")
			miniBtn.Children(func() {
				ui.Text(c, "最小化到托盘").FontSize(12.5).FontWeight(600)
			})
			if miniBtn.Clicked() {
				if d.remember {
					a.settings.CloseAction = "minimize"
					a.settings.save()
				}
				d.open = false
				if runtime.GOOS == "darwin" || a.tray != nil {
					a.win.Hide()
				} else {
					a.win.Minimize()
				}
			}
		})
	})
}
