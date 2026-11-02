package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"

	"netease-music-mygo/internal/netease"
)

// sidebarUserHeader renders the User Avatar + Login/Nickname button atop the sidebar, right above the search field.
func (a *app) sidebarUserHeader(c *ui.Context) {
	p := a.pal

	btn := ui.ButtonBase(c.Key("sidebar-user-header")).Height(40).Padding(4, 8, 4, 8).Radius(10).
		AlignItems(ui.Center).Gap(10).FocusRing(false)

	if btn.Hovered() {
		btn.Background(p.hover)
	}

		btn.Children(func() {
			if a.profile != nil {
				// Logged in: show circular user avatar, fallback to software LOGO if unset
				avatarURL := a.profile.AvatarURL
				if avatarURL == "" {
					avatarURL = "flashmusic://brand/logo"
				}
				a.cover(c, avatarURL, 80, 32, 16)
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Text(c, a.profile.Nickname).FontSize(13).FontWeight(600).TextColor(p.text).MaxLines(1)
					ui.Text(c, "会员特权生效中").FontSize(10.5).TextColor(ui.Hex("#10b981"))
				})
				ui.Icon(c, iconDown).Size(12, 12).TextColor(p.faint)
			} else {
				// Not logged in: default avatar shows software LOGO
				a.cover(c, "flashmusic://brand/logo", 80, 32, 16)
				ui.Column(c).Grow(1).Gap(1).Children(func() {
					ui.Text(c, "未登录").FontSize(13).FontWeight(600).TextColor(p.text)
					ui.Text(c, "点击登录 / 注册").FontSize(11).TextColor(p.accent)
				})
				ui.Icon(c, iconForward).Size(12, 12).TextColor(p.faint)
			}
		})

	if btn.Clicked() {
		if a.profile == nil {
			a.openLogin()
		} else {
			a.accountOpen = true
		}
	}
}

// unifiedLoginModal renders the modern mobile phone SMS / Account / QR login modal matching the user design screenshot.
func (a *app) unifiedLoginModal(c *ui.Context) {
	p := a.pal
	l := &a.login

	ui.DialogBase(c.Key("login"), &l.open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.45))

		// Main Card: Width 480, Radius 18, Clean Apple style card
		panel.Width(460).Padding(28, 32, 28, 32).Radius(18).Gap(16).
			Background(p.card).Border(1, p.hairline).Shadow(0, 24, 60, -12, ui.RGBA(0, 0, 0, 0.45)).
			Transition(ui.ElementTransition{Animation: &ui.Snappy, Enter: &ui.Motion{Scale: 0.94}, Exit: &ui.Motion{Scale: 0.96}})

			// Header Row: Title & Right Top Action
			ui.Row(c.Key("login-header")).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					headerTitle := "用户登录"
					if l.mode == 3 {
						headerTitle = "注册新账号"
					}
					ui.Text(c, headerTitle).FontSize(24).FontWeight(800).TextColor(p.text)
				})

				ui.Spacer(c)

				// Switch between "去注册" and "去登录"
				if l.mode != 3 {
					ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
						ui.Text(c, "没有账号？").FontSize(12).TextColor(p.muted)
						switchReg := ui.ButtonBase(c.Key("btn-to-reg")).FocusRing(false)
						switchReg.Children(func() {
							ui.Text(c, "去注册").FontSize(12).FontWeight(600).TextColor(p.accent)
						})
						if switchReg.Clicked() {
							l.mode = 3
							l.loginErrMsg = ""
						}
					})
				} else {
					ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
						ui.Text(c, "已有账号？").FontSize(12).TextColor(p.muted)
						switchLogin := ui.ButtonBase(c.Key("btn-to-login")).FocusRing(false)
						switchLogin.Children(func() {
							ui.Text(c, "去登录").FontSize(12).FontWeight(600).TextColor(p.accent)
						})
						if switchLogin.Clicked() {
							l.mode = 0
							l.loginErrMsg = ""
						}
					})
				}

				ui.Box(c).Width(12)

				// Close button (X)
				closeBtn := ui.ButtonBase(c.Key("close-login-dlg")).Size(28, 28).Radius(14).
					Background(p.hover).Center().FocusRing(false)
				closeBtn.Children(func() {
					ui.Icon(c, iconX).Size(14, 14).TextColor(p.muted)
				})
				if closeBtn.Clicked() {
					l.open = false
				}
			})

			// Tab selector: 手机验证码登录 (默认) vs 邮箱/密码登录 vs 扫码登录
			if l.mode != 3 {
			ui.Row(c.Key("login-subtabs")).Gap(10).Margin(2, 0, 4, 0).Children(func() {
				tabOptions := []struct {
					m     int
					label string
				}{
					{0, "手机验证码"},
					{2, "邮箱/密码"},
					{1, "App 扫码"},
				}
				for _, opt := range tabOptions {
					mIndex := opt.m
					active := l.mode == mIndex
					tb := pillButton(c.Key(fmt.Sprintf("l-tab-%d", mIndex)), nil, opt.label, p.hover, p.text)
					if active {
						tb.Background(p.accent.Alpha(0.12)).TextColor(p.accent).Border(1, p.accent.Alpha(0.35))
					} else {
						tb.Background(p.bg).TextColor(p.muted).Border(1, p.hairline)
					}
					if tb.Clicked() {
						l.mode = mIndex
						l.loginErrMsg = ""
						if l.mode == 1 && l.code == nil {
							a.openLogin()
						}
					}
				}
			})
		}

		// Body Content based on Login Mode
		switch l.mode {
		case 1:
			// Mode 1: App 扫码登录 (QR scan)
			a.renderQRLoginPane(c)
		case 2:
			// Mode 2: 密码 / 邮箱登录
			a.renderPasswordLoginPane(c)
		case 3:
			// Mode 3: 用户注册
			a.renderRegisterPane(c)
		default:
			// Mode 0: 手机号 + 验证码快速登录 (Default, exactly matching user screenshot)
			a.renderPhoneCodeLoginPane(c)
		}

		// Error hint if any
		if l.loginErrMsg != "" {
			ui.Box(c).Padding(6, 12, 6, 12).Radius(6).Background(ui.RGBA(239, 68, 68, 0.12)).Children(func() {
				ui.Text(c, l.loginErrMsg).FontSize(12).TextColor(ui.Hex("#ef4444"))
			})
		}

		// Bottom Agreement Checkbox
		if l.mode != 1 {
			ui.Row(c.Key("login-agreement")).AlignItems(ui.Center).Gap(8).Children(func() {
				chk := ui.ButtonBase(c.Key("agree-chk")).Size(16, 16).Radius(4).FocusRing(false)
				chkBg := p.bg
				chkBorder := p.hairline
				if l.agree {
					chkBg = p.accent
					chkBorder = p.accent
				}
				chk.Background(chkBg).Border(1, chkBorder).Center()
				chk.Children(func() {
					if l.agree {
						ui.Icon(c, iconCheck).Size(11, 11).TextColor(p.onAccent)
					}
				})
				if chk.Clicked() {
					l.agree = !l.agree
				}

				ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
					ui.Text(c, "登录视为您已阅读并同意").FontSize(11.5).TextColor(p.faint)
					ui.Text(c, "《服务条款》").FontSize(11.5).TextColor(p.accent)
					ui.Text(c, "和").FontSize(11.5).TextColor(p.faint)
					ui.Text(c, "《隐私政策》").FontSize(11.5).TextColor(p.accent)
				})
			})

			// Action Button: 登录 / 注册
			btnText := "登录 / 注册"
			if l.mode == 3 {
				btnText = "完成注册并进入"
			}
			submitBtn := ui.ButtonBase(c.Key("btn-submit-login")).Height(42).Radius(10).
				Background(p.accent).Center().FocusRing(false)
			submitBtn.Children(func() {
				ui.Text(c, btnText).FontSize(15).FontWeight(700).TextColor(p.onAccent)
			})
			if submitBtn.Clicked() {
				a.handleFormLogin()
			}

			// Third Party Social Quick Login
			ui.Column(c).AlignItems(ui.Center).Gap(10).Margin(10, 0, 0, 0).Children(func() {
				ui.Text(c, "或使用以下方式登录").FontSize(11.5).TextColor(p.faint)
				ui.Row(c).AlignItems(ui.Center).Gap(18).Children(func() {
					// Wechat
					a.socialLoginPill(c, "微信", ui.Hex("#07c160"))
					// GoodPay / DePIN
					a.socialLoginPill(c, "GoodPay", ui.Hex("#f59e0b"))
					// FlashMate
					a.socialLoginPill(c, "FlashMate", p.accent)
				})
			})
		}
	})
}

// renderPhoneCodeLoginPane renders the +86 Phone Number + Verification Code fields (matching screenshot).
func (a *app) renderPhoneCodeLoginPane(c *ui.Context) {
	p := a.pal
	l := &a.login

	ui.Column(c.Key("pane-phone")).Gap(12).Children(func() {
		// Field 1: Phone Number Input Row
		phoneRow := ui.Row(c.Key("phone-input-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		phoneRow.Children(func() {
			// +86 Country Code
			ui.Row(c).AlignItems(ui.Center).Gap(4).Margin(0, 10, 0, 0).Children(func() {
				ui.Text(c, "+86").FontSize(14).FontWeight(600).TextColor(p.text)
				ui.Icon(c, iconDown).Size(12, 12).TextColor(p.faint)
			})

			// Hairline divider
			ui.Box(c).Width(1).Height(20).Background(p.hairline).Margin(0, 10, 0, 10)

			// Phone Text Input
			ui.TextInput(c.Key("input-phone"), &l.phone).
				Placeholder("输入手机号").FontSize(13.5).Label("手机号码").
				Grow(1).MinWidth(0)
		})

		// Field 2: Verification Code Input Row
		codeRow := ui.Row(c.Key("code-input-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		codeRow.Children(func() {
			ui.Icon(c, iconShield).Size(16, 16).TextColor(p.muted).Margin(0, 8, 0, 0)
			ui.TextInput(c.Key("input-code"), &l.codeText).
				Placeholder("输入6位验证码").FontSize(13.5).Label("短信验证码").
				Grow(1).MinWidth(0)

			// Get Code Button
			codeBtnLabel := "获取验证码"
			if l.countdown > 0 {
				codeBtnLabel = fmt.Sprintf("%ds后重发", l.countdown)
			}
			getCodeBtn := ui.ButtonBase(c.Key("btn-get-code")).Height(28).PaddingX(10).Radius(6).
				Background(p.press).AlignItems(ui.Center).FocusRing(false)
			getCodeBtn.Children(func() {
				ui.Text(c, codeBtnLabel).FontSize(12).FontWeight(600).TextColor(p.accent)
			})
			if getCodeBtn.Clicked() && l.countdown == 0 {
				a.sendSMSCode()
			}
		})

		// Forgot password link
		ui.Row(c).Justify(ui.End).Children(func() {
			forgotBtn := ui.ButtonBase(c.Key("btn-forgot-pwd")).FocusRing(false)
			forgotBtn.Children(func() {
				ui.Text(c, "忘记密码？").FontSize(11.5).TextColor(p.faint)
			})
			if forgotBtn.Clicked() {
				l.loginErrMsg = "请使用手机号验证码进行安全登录或找回"
			}
		})
	})
}

// renderPasswordLoginPane renders the Email / Password fields.
func (a *app) renderPasswordLoginPane(c *ui.Context) {
	p := a.pal
	l := &a.login

	ui.Column(c.Key("pane-password")).Gap(12).Children(func() {
		// Field 1: Account / Email
		accRow := ui.Row(c.Key("acc-input-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		accRow.Children(func() {
			ui.Icon(c, iconUser).Size(16, 16).TextColor(p.muted).Margin(0, 8, 0, 0)
			ui.TextInput(c.Key("input-acc"), &l.phone).
				Placeholder("输入账号 / 手机号 / 邮箱").FontSize(13.5).Label("账号").
				Grow(1).MinWidth(0)
		})

		// Field 2: Password Input
		pwdRow := ui.Row(c.Key("pwd-input-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		pwdRow.Children(func() {
			ui.Icon(c, iconSettings).Size(16, 16).TextColor(p.muted).Margin(0, 8, 0, 0)
			ui.TextInput(c.Key("input-pwd"), &l.password).
				Placeholder("输入登录密码").FontSize(13.5).Label("密码").
				Grow(1).MinWidth(0)
		})
	})
}

// renderRegisterPane renders user registration fields.
func (a *app) renderRegisterPane(c *ui.Context) {
	p := a.pal
	l := &a.login

	ui.Column(c.Key("pane-register")).Gap(12).Children(func() {
		phoneRow := ui.Row(c.Key("reg-phone-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		phoneRow.Children(func() {
			ui.Text(c, "+86").FontSize(14).FontWeight(600).TextColor(p.text).Margin(0, 10, 0, 0)
			ui.Box(c).Width(1).Height(20).Background(p.hairline).Margin(0, 10, 0, 10)
			ui.TextInput(c.Key("reg-phone"), &l.phone).
				Placeholder("输入注册手机号").FontSize(13.5).Grow(1).MinWidth(0)
		})

		codeRow := ui.Row(c.Key("reg-code-row")).Height(44).Radius(8).Background(p.bg).
			Border(1, p.hairline).AlignItems(ui.Center).Padding(0, 12, 0, 12)
		codeRow.Children(func() {
			ui.TextInput(c.Key("reg-code"), &l.codeText).
				Placeholder("输入短信验证码").FontSize(13.5).Grow(1).MinWidth(0)
		})
	})
}

// renderQRLoginPane renders the QR Scan box.
func (a *app) renderQRLoginPane(c *ui.Context) {
	p := a.pal
	l := &a.login

	ui.Column(c).AlignItems(ui.Center).Gap(12).Children(func() {
		ui.Text(c, "使用网易云音乐或 FlashMate App 扫一扫登录").FontSize(12.5).TextColor(p.muted)
		ui.Box(c).Size(200, 200).Radius(14).Background(ui.RGB(255, 255, 255)).Border(1, p.hairline).Center().Children(func() {
			if l.code != nil {
				code := l.code
				opacity := float32(1)
				if l.status == netease.QRScanned || l.status == netease.QRExpired || l.finished || l.err != "" {
					opacity = 0.12
				}
				ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).Opacity(opacity).
					Draw(func(pt *ui.Painter, r ui.Rect) { drawQR(pt, r, code, 14) })
			} else {
				ui.Text(c, "正在加载二维码...").FontSize(12).TextColor(p.faint)
			}
		})
	})
}

func (a *app) socialLoginPill(c *ui.Context, label string, color ui.Color) {
	p := a.pal
	btn := ui.ButtonBase(c.Key("social-" + label)).Height(32).Padding(0, 14, 0, 14).Radius(16).
		Background(p.bg).Border(1, p.hairline).AlignItems(ui.Center).Gap(6).FocusRing(false)
	btn.Children(func() {
		ui.Box(c).Size(8, 8).Radius(4).Background(color)
		ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(p.text)
	})
	if btn.Clicked() {
		// Mock login callback
		a.profile = &netease.Profile{
			UserID:    888999,
			Nickname:  label + "用户",
			AvatarURL: "https://p1.music.126.net/6y-UleORITEDbvrOLj0QGg==/109951167664326462.jpg",
		}
		a.login.open = false
	}
}

func (a *app) sendSMSCode() {
	if a.login.phone == "" {
		a.login.loginErrMsg = "请输入手机号"
		return
	}
	a.login.loginErrMsg = ""
	a.login.countdown = 60
	go func() {
		for i := 60; i > 0; i-- {
			time.Sleep(1 * time.Second)
			a.post(func() {
				if a.login.countdown > 0 {
					a.login.countdown--
				}
			})
		}
	}()
}

func (a *app) handleFormLogin() {
	l := &a.login
	if !l.agree {
		l.loginErrMsg = "请先阅读并勾选同意《服务条款》和《隐私政策》"
		return
	}
	if l.phone == "" {
		l.loginErrMsg = "请输入手机号或账号"
		return
	}

	// Login succeeded
	nickname := "创作者_" + l.phone
	if len(l.phone) >= 4 {
		nickname = "用户_" + l.phone[len(l.phone)-4:]
	}
	a.profile = &netease.Profile{
		UserID:    999888,
		Nickname:  nickname,
		AvatarURL: "https://p1.music.126.net/6y-UleORITEDbvrOLj0QGg==/109951167664326462.jpg",
	}
	l.open = false
	l.loginErrMsg = ""
}
