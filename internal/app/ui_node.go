package app

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// nodeBoardPage renders the DePIN Edge Node Center (Overview tab).
func (a *app) nodeBoardPage(c *ui.Context) {
	a.nodeCenterView(c)
}

// nodeStoragePage renders the Storage & Sharing tab of Node Center.
func (a *app) nodeStoragePage(c *ui.Context) {
	a.nodeCenterView(c)
}

// nodeEarningsPage renders the Assets & Earnings tab of Node Center.
func (a *app) nodeEarningsPage(c *ui.Context) {
	a.nodeCenterView(c)
}

// nodeCenterView renders the unified 5-tab Node Center modeled after FlashMate Desktop & FlashVCP Cloud.
func (a *app) nodeCenterView(c *ui.Context) {
	top, bottom := a.insets(c)
	p := a.pal

	ui.Column(c.Key("node-center-container")).Fill().Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).Gap(14).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		// Top Header Row
		ui.Row(c.Key("nc-header-bar")).AlignItems(ui.Center).Gap(12).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(8).Children(func() {
				ui.Icon(c, iconServer).Size(22, 22).TextColor(p.accent)
				ui.Text(c, "边缘节点中心").FontSize(20).FontWeight(700).TextColor(p.text)
			})

			// FlashMate Device & Credit Tier Badge
			ui.Row(c.Key("nc-badges")).AlignItems(ui.Center).Gap(6).Children(func() {
				// AAA Credit Badge
				ui.Box(c).Padding(2, 8, 2, 8).Radius(5).
					Background(ui.RGBA(16, 185, 129, 0.15)).Border(0.5, ui.RGBA(16, 185, 129, 0.4)).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(4).Children(func() {
						ui.Icon(c, iconShield).Size(12, 12).TextColor(ui.Hex("#10b981"))
						ui.Text(c, fmt.Sprintf("%s 级高信用节点", a.nodeCreditTier)).FontSize(11).FontWeight(700).TextColor(ui.Hex("#10b981"))
					})
				})

				// Status Badge
				statusColor := ui.Hex("#10b981")
				statusText := "运行就绪 · 高速分发"
				if a.nodeStatus == "paused" {
					statusColor = ui.Hex("#f59e0b")
					statusText = "已暂停共享"
				}
				ui.Box(c).Padding(2, 7, 2, 7).Radius(5).
					Background(p.press).Border(1, p.hairline).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(5).Children(func() {
						ui.Box(c).Size(6, 6).Radius(3).Background(statusColor)
						ui.Text(c, statusText).FontSize(11).TextColor(p.muted)
					})
				})
			})

			ui.Spacer(c)

			// Refresh State Button
			refreshBtn := pillButton(c.Key("nc-refresh"), iconReload, "刷新探针", p.hover, p.text).
				Border(1, p.hairline)
			if refreshBtn.Clicked() {
				if a.nodeStatus == "running" {
					a.nodeUploadSpeed = 4.85 + (float64(time.Now().Unix()%10) * 0.11)
					a.nodeDownloadSpeed = 18.2 + (float64(time.Now().Unix()%8) * 0.15)
				}
			}
		})

		// 5 Top Navigation Tabs: 概览 (Overview), 环境健康 (Environment), 共享策略 (Sharing), 资产收益 (Assets), 任务账单 (Tasks)
		tabs := []struct {
			name string
			icon *ui.SVG
		}{
			{"运行概览", iconServer},
			{"环境健康 (12项)", iconCheck},
			{"共享策略与存储", iconFolder},
			{"资产与收益 (XFB)", iconCoins},
			{"云端任务账单", iconNetwork},
		}

		ui.Row(c.Key("nc-tab-bar")).Gap(8).Children(func() {
			for i, t := range tabs {
				tabIdx := i
				active := a.nodeTab == tabIdx
				btn := pillButton(c.Key(fmt.Sprintf("nc-tab-%d", i)), t.icon, t.name, p.hover, p.text)
				if active {
					btn.Background(p.accent).TextColor(p.onAccent).Border(0, ui.Transparent)
				} else {
					btn.Background(p.card).TextColor(p.muted).Border(1, p.hairline)
				}
				if btn.Clicked() {
					a.nodeTab = tabIdx
				}
			}
		})

		// Active Tab Content
		switch a.nodeTab {
		case 1:
			a.nodeEnvironmentTab(c)
		case 2:
			a.nodeSharingTab(c)
		case 3:
			a.nodeAssetsTab(c)
		case 4:
			a.nodeTaskHistoryTab(c)
		default:
			a.nodeOverviewTab(c)
		}
	})
}

// nodeOverviewTab renders Tab 0: Overview (matching FlashMate Overview.tsx).
func (a *app) nodeOverviewTab(c *ui.Context) {
	p := a.pal

	ui.Column(c.Key("tab-overview")).Gap(14).Children(func() {
		// Device Service Card with Action Controls
		ui.Box(c.Key("device-service-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(18, 20, 18, 20).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(18).Children(func() {
				// State Ring Icon
				ringBg := ui.RGBA(16, 185, 129, 0.12)
				ringBorder := ui.Hex("#10b981")
				ringIcon := iconBolt
				if a.nodeStatus == "paused" {
					ringBg = ui.RGBA(245, 158, 11, 0.12)
					ringBorder = ui.Hex("#f59e0b")
					ringIcon = iconPause
				}
				ui.Box(c).Size(54, 54).Radius(27).Background(ringBg).
					Border(1.5, ringBorder).Center().Children(func() {
					ui.Icon(c, ringIcon).Size(26, 26).TextColor(ringBorder)
				})

				// Description
				ui.Column(c).Grow(1).Gap(4).Children(func() {
					titleText := "边缘节点服务运行中"
					hintText := "本节点正通过 FlashMate P2P 协议向同城对等节点高速分发微短剧切片，上行通道通畅。"
					if a.nodeStatus == "paused" {
						titleText = "边缘节点服务已挂起暂停"
						hintText = "共享分发已暂停，本地播放不受影响，随时点击开始共享恢复带宽贡献与收益积累。"
					}
					ui.Text(c, titleText).FontSize(16).FontWeight(700).TextColor(p.text)
					ui.Text(c, hintText).FontSize(12.5).TextColor(p.muted)
					ui.Text(c, fmt.Sprintf("设备 ID: %s · 会话在线持续: %s · 准入协议: EdgeMesh v1.0.4",
						a.nodeID, formatDurationSince(a.nodeSessionStart))).FontSize(11.5).TextColor(p.faint)
				})

				// Start / Pause Control
				if a.nodeStatus == "running" {
					pauseBtn := pillButton(c.Key("btn-pause-node"), iconPause, "暂停共享", p.hover, p.text).
						Border(1, p.hairline)
					if pauseBtn.Clicked() {
						a.nodeStatus = "paused"
						a.nodeUploadSpeed = 0.0
					}
				} else {
					startBtn := pillButton(c.Key("btn-start-node"), iconPlay, "开始共享", p.accent, p.onAccent)
					if startBtn.Clicked() {
						a.nodeStatus = "running"
						a.nodeUploadSpeed = 4.85
					}
				}
			})
		})

		// 4 Performance Metrics Cards Grid
		ui.Row(c.Key("metrics-grid")).Gap(12).Children(func() {
			a.nodeMetricCard(c, "实时上行带宽", fmt.Sprintf("%.2f MB/s", a.nodeUploadSpeed),
				fmt.Sprintf("配额限额 %d Mbps", a.nodeUploadCapMbps), iconBolt, p.accent)

			a.nodeMetricCard(c, "实时下行拉取", fmt.Sprintf("%.2f MB/s", a.nodeDownloadSpeed),
				"热点分片智能预热", iconNetwork, ui.Hex("#3b82f6"))

			a.nodeMetricCard(c, "今日贡献流量", fmt.Sprintf("%.1f GB", a.nodeTodayTrafficGB),
				"较昨日环比 +18%", iconServer, ui.Hex("#10b981"))

			a.nodeMetricCard(c, "累计收益折算", fmt.Sprintf("¥%.1f 元", a.nodeOffsetRMB),
				fmt.Sprintf("折合 %.1f XFB 资产", a.nodeEarnedXFB), iconCoins, ui.Hex("#f59e0b"))
		})

		// Recent Cloud Task Execution Preview
		ui.Box(c.Key("cloud-tasks-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(16, 20, 16, 20).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconNetwork).Size(16, 16).TextColor(p.accent).Margin(0, 6, 0, 0)
					ui.Text(c, "近期云端调度与切片分发任务 (Cloud Task Feed)").FontSize(14).FontWeight(700).TextColor(p.text)
					ui.Spacer(c)
					moreBtn := pillButton(c.Key("view-all-tasks"), iconMore, "查看全部记录", p.hover, p.muted).
						Border(1, p.hairline)
					if moreBtn.Clicked() {
						a.nodeTab = 4
					}
				})

				// Mini Task Rows
				tasks := []struct {
					id    string
					typ   string
					size  string
					state string
					time  string
				}{
					{"task-9021-fca", "MEDIA_SEGMENT_CACHE", "14.2 MB", "SUCCEEDED", "1分钟前"},
					{"task-9020-e81", "MEDIA_SEGMENT_CACHE", "28.5 MB", "SUCCEEDED", "3分钟前"},
					{"task-9019-d74", "PROOF_OF_DELIVERY", "4.1 MB", "SUCCEEDED", "6分钟前"},
				}

					for _, t := range tasks {
						ui.Row(c.Key("task-mini-"+t.id)).AlignItems(ui.Center).Padding(6, 4, 6, 4).Children(func() {
							ui.Box(c).Size(6, 6).Radius(3).Background(ui.Hex("#10b981")).Margin(0, 8, 0, 0)
							ui.Text(c, t.id).FontSize(12).FontWeight(600).TextColor(p.text)
							ui.Box(c).Padding(1, 5, 1, 5).Radius(3).Background(p.press).Margin(0, 8, 0, 8).Children(func() {
								ui.Text(c, t.typ).FontSize(10).TextColor(p.accent)
							})
							ui.Text(c, t.size).FontSize(12).TextColor(p.muted)
							ui.Spacer(c)
							ui.Text(c, t.time).FontSize(11.5).TextColor(p.faint)
						})
					}
				})
			})
		})
	}

func (a *app) nodeMetricCard(c *ui.Context, label, val, sub string, icon *ui.SVG, iconColor ui.Color) {
	p := a.pal
	ui.Box(c.Key("metric-"+label)).Grow(1).Radius(10).Background(p.card).
		Border(1, p.hairline).Padding(14, 16, 14, 16).Children(func() {
		ui.Column(c).Gap(6).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, label).FontSize(12).FontWeight(600).TextColor(p.muted)
				ui.Spacer(c)
				ui.Icon(c, icon).Size(16, 16).TextColor(iconColor)
			})
			ui.Text(c, val).FontSize(20).FontWeight(700).TextColor(p.text)
			ui.Text(c, sub).FontSize(11).TextColor(p.faint)
		})
	})
}

// nodeEnvironmentTab renders Tab 1: 12-Probe Environment Health Check (GuidedEnvironment.tsx).
func (a *app) nodeEnvironmentTab(c *ui.Context) {
	p := a.pal

	probes := []struct {
		key   string
		label string
		desc  string
		val   string
	}{
		{"network", "网络网关连通性", "IPv4/IPv6 外网访问畅通，DNS 解析正常", "已连接 (100% 可靠)"},
		{"direct", "直连打洞与 NAT 类型", "Full Cone NAT 优质对等打洞能力", "Full Cone 极佳"},
		{"upload", "实测上行带宽", "满足 FlashMate 边缘节点最低 10 Mbps 门槛", "实测 48.6 Mbps (优秀)"},
		{"latency", "往返时延 (RTT)", "至 FlashVCP Cloud 核心节点网络延迟", "14 ms (低延迟)"},
		{"stability", "连接稳定性", "会话内掉线监控与心跳保活率", "0 次中断 · 99.98%"},
		{"configuration", "调度策略合规", "共享计划与带宽上限配置合规", "24/7 全天候持续调度"},
		{"cpu", "CPU 核心与可用算力", "多核并发处理与切片加解密负载", "8 核心可用 (负载 12%)"},
		{"disk", "磁盘读写吞吐", "高速 NVMe/SATA 连续读写效能", "顺序读 3200 MB/s"},
		{"directory", "共享存储目录权限", "分片存储盘读写与持久化权限", "I:\\FlashMedia (可读写)"},
		{"power", "电源供电接入", "检测是否接入 AC 稳定交流电源", "交流电源已接入"},
		{"version", "协议与 Agent 版本", "Edge Node Protocol 握手协议版本", "EdgeMesh v1.0.4"},
		{"sleep", "系统防休眠策略", "分发期间后台保活防止掉线", "唤醒策略已生效"},
	}

	ui.Column(c.Key("tab-environment")).Gap(14).Children(func() {
		// Summary Banner
		ui.Box(c.Key("env-banner")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(16, 20, 16, 20).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconCheck).Size(18, 18).TextColor(ui.Hex("#10b981"))
						ui.Text(c, "环境体检全部通过 · 具备最佳边缘分发能力").
							FontSize(15).FontWeight(700).TextColor(p.text)
					})
					ui.Text(c, "已通过 12 项硬件、网络与系统环境指标探针检测，网络贡献度系数 1.25x 加成。").
						FontSize(12.5).TextColor(p.muted)
				})
				ui.Spacer(c)
				recheckBtn := pillButton(c.Key("btn-recheck"), iconReload, "重新体检", p.accent, p.onAccent)
				if recheckBtn.Clicked() {
					a.nodeTodayTrafficGB += 0.1
				}
			})
		})

		// 12 Probe Cards (2 Columns Grid)
		ui.Row(c.Key("probes-grid")).Wrap().Gap(12).Children(func() {
			for _, probe := range probes {
				item := probe
				ui.Box(c.Key("probe-"+item.key)).Width(380).Radius(10).
					Background(p.card).Border(1, p.hairline).Padding(12, 16, 12, 16).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						ui.Box(c).Size(22, 22).Radius(11).Background(ui.RGBA(16, 185, 129, 0.15)).
							Center().Margin(0, 10, 0, 0).Children(func() {
							ui.Icon(c, iconCheck).Size(13, 13).TextColor(ui.Hex("#10b981"))
						})
						ui.Column(c).Grow(1).Gap(2).Children(func() {
							ui.Row(c).AlignItems(ui.Center).Children(func() {
								ui.Text(c, item.label).FontSize(13).FontWeight(600).TextColor(p.text)
								ui.Spacer(c)
								ui.Text(c, item.val).FontSize(11.5).FontWeight(600).TextColor(ui.Hex("#10b981"))
							})
							ui.Text(c, item.desc).FontSize(11.5).TextColor(p.muted)
						})
					})
				})
			}
		})
	})
}

// nodeSharingTab renders Tab 2: Bandwidth & Storage Configuration (Sharing.tsx).
func (a *app) nodeSharingTab(c *ui.Context) {
	p := a.pal

	ui.Column(c.Key("tab-sharing")).Gap(16).Children(func() {
		// Section 1: Bandwidth Sharing Presets
		ui.Box(c.Key("bw-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(18, 20, 18, 20).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "上行带宽共享挡位").FontSize(15).FontWeight(700).TextColor(p.text)
					ui.Spacer(c)
					ui.Text(c, fmt.Sprintf("当前上限: %d Mbps", a.nodeUploadCapMbps)).
						FontSize(12).FontWeight(600).TextColor(p.accent)
				})
				ui.Text(c, "根据您的日常网络使用偏好分配闲置上行带宽，系统保证优先保障本地音乐与视频播放流畅。").
					FontSize(12.5).TextColor(p.muted)

				// 4 Preset Buttons (10M / 20M / 35M / 50M)
				presets := []struct {
					mbps  int
					label string
					desc  string
				}{
					{10, "轻量挡 (10 Mbps)", "适合边下载边工作，低占用"},
					{20, "均衡挡 (20 Mbps)", "日常推荐，兼顾影音与共享"},
					{35, "全速挡 (35 Mbps)", "性能全开，最大化 XFB 收益"},
					{50, "极客挡 (50 Mbps)", "大带宽专享，高速对等网"},
				}

					ui.Row(c.Key("bw-presets")).Gap(10).Children(func() {
						for _, pr := range presets {
							preset := pr
							active := a.nodeUploadCapMbps == preset.mbps
							btn := ui.ButtonBase(c.Key(fmt.Sprintf("preset-%d", preset.mbps))).
								Grow(1).Radius(8).Padding(12, 14, 12, 14).FocusRing(false)

							bg := p.bg
							borderColor := p.hairline
							if active {
								bg = p.accent.Alpha(0.08)
								borderColor = p.accent
							} else if btn.Hovered() {
								bg = p.hover
							}
							btn.Background(bg).Border(1, borderColor)

							btn.Children(func() {
								ui.Column(c).Gap(3).Children(func() {
									ui.Text(c, preset.label).FontSize(13).FontWeight(700).
										TextColor(map[bool]ui.Color{true: p.accent, false: p.text}[active])
									ui.Text(c, preset.desc).FontSize(11).TextColor(p.faint)
								})
							})

							if btn.Clicked() {
								a.nodeUploadCapMbps = preset.mbps
							}
						}
					})
			})
		})

		// Section 2: Storage Quota & Directory
		ui.Box(c.Key("storage-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(18, 20, 18, 20).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "存储配额与本地缓存管理").FontSize(15).FontWeight(700).TextColor(p.text)
					ui.Spacer(c)
					ui.Text(c, "主存储驱动器: I:\\FlashMedia").FontSize(12).TextColor(p.faint)
				})

				// Progress Gauge
				usedGB := float64(a.nodeCachedBytes) / (1024 * 1024 * 1024)
				totalGB := float64(a.nodeTotalQuotaBytes) / (1024 * 1024 * 1024)
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, fmt.Sprintf("已分配缓存容量: %.1f GB / 上限 %.1f GB (%.0f%%)",
						usedGB, totalGB, (usedGB/totalGB)*100)).FontSize(13).FontWeight(600).TextColor(p.text)
				})

				ui.Box(c).Height(10).Radius(5).Background(p.hover).Children(func() {
					ui.Box(c).Width(240).Height(10).Radius(5).Background(p.accent)
				})

				// Action buttons
				ui.Row(c).Gap(10).Children(func() {
					clearBtn := pillButton(c.Key("btn-clear-slices"), iconTrash, "一键清理过期分片", p.hover, p.text).
						Border(1, p.hairline)
					if clearBtn.Clicked() {
						a.nodeCachedBytes = 12 * 1024 * 1024 * 1024
					}
					pillButton(c.Key("btn-choose-dir"), iconFolder, "更换存储目录", p.hover, p.text).
						Border(1, p.hairline)
				})
			})
		})

		// Section 3: Schedule & Power Policy
		ui.Box(c.Key("policy-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(18, 20, 18, 20).Children(func() {
			ui.Column(c).Gap(10).Children(func() {
				ui.Text(c, "运行调度与电源保活策略").FontSize(15).FontWeight(700).TextColor(p.text)

				ui.Row(c).AlignItems(ui.Center).Gap(12).Children(func() {
					pillButton(c.Key("sched-247"), nil, "✓ 24/7 全天候持续加速", p.press, p.accent).
						Border(1, p.accent.Alpha(0.3))
					pillButton(c.Key("sched-night"), nil, "夜间智能调度 (22:00~07:00)", p.hover, p.muted).
						Border(1, p.hairline)
				})

				ui.Row(c).AlignItems(ui.Center).Gap(16).Margin(6, 0, 0, 0).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconCheck).Size(14, 14).TextColor(ui.Hex("#10b981"))
						ui.Text(c, "共享期间阻止系统进入休眠模式 (保持唤醒)").FontSize(12).TextColor(p.muted)
					})
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconCheck).Size(14, 14).TextColor(ui.Hex("#10b981"))
						ui.Text(c, "使用笔记本电池供电时自动暂停分发").FontSize(12).TextColor(p.muted)
					})
				})
			})
		})
	})
}

// nodeAssetsTab renders Tab 3: Assets, Coupons & XFB Rewards (Assets.tsx).
func (a *app) nodeAssetsTab(c *ui.Context) {
	p := a.pal

	ui.Column(c.Key("tab-assets")).Gap(16).Children(func() {
		// Summary Banner
		ui.Box(c.Key("assets-summary-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(20, 24, 20, 24).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(6).Children(func() {
					ui.Text(c, "节点累计收益总值 (本周期)").FontSize(13).TextColor(p.faint)
					ui.Row(c).AlignItems(ui.Baseline).Gap(10).Children(func() {
						ui.Text(c, fmt.Sprintf("%.2f", a.nodeEarnedXFB)).FontSize(36).FontWeight(800).TextColor(p.text)
						ui.Text(c, "XFB").FontSize(18).FontWeight(700).TextColor(p.accent)
						ui.Text(c, fmt.Sprintf("≈ ¥%.2f 元", a.nodeOffsetRMB)).FontSize(15).TextColor(p.muted)
					})
					ui.Text(c, "站长专属权益: 累计收益已自动冲抵本月自营微短剧 CDN 流量账单 100%").
						FontSize(12).TextColor(ui.Hex("#10b981"))
				})
				ui.Spacer(c)
				ui.Column(c).Gap(8).Children(func() {
					pillButton(c.Key("btn-bind-goodpay"), iconWallet, "划转至 GoodPay 钱包", p.accent, p.onAccent)
					pillButton(c.Key("btn-swap-xfb"), iconCoins, "生态商城闪兑消费", p.hover, p.text).
						Border(1, p.hairline)
				})
			})
		})

		// 4 Breakdown Metrics (Standby, Business, Pending, Settled)
		ui.Row(c.Key("breakdown-grid")).Gap(12).Children(func() {
			a.nodeMetricCard(c, "待机在线基础奖励", "¥"+a.nodeStandbyMinor+" 元", "每日心跳保活结算", iconServer, p.accent)
			a.nodeMetricCard(c, "业务分发按量奖励", "¥"+a.nodeBusinessMinor+" 元", "切片上行按量核算", iconBolt, ui.Hex("#3b82f6"))
			a.nodeMetricCard(c, "待结算收益凭证", "¥14.60 元", "次日 00:00 自动入账", iconCoins, ui.Hex("#f59e0b"))
			a.nodeMetricCard(c, "已抵扣 CDN 费用", "¥137.00 元", "累计抵扣金额", iconCheck, ui.Hex("#10b981"))
		})

		// GoodPay Coupons & Vouchers Section
		ui.Box(c.Key("coupons-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(16, 20, 16, 20).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconWallet).Size(16, 16).TextColor(p.accent).Margin(0, 6, 0, 0)
					ui.Text(c, "GoodPay 专属抵扣优惠券").FontSize(14).FontWeight(700).TextColor(p.text)
					ui.Spacer(c)
					ui.Text(c, "已关联 GoodPay 钱包账户").FontSize(12).TextColor(ui.Hex("#10b981"))
				})

				// 2 Coupon Vouchers
				coupons := []struct {
					title   string
					amount  string
					xfbBack string
					expiry  string
					tag     string
				}{
					{"自营微短剧全网带宽抵扣券", "¥50.00", "25.0 XFB", "2026-12-31", "可用"},
					{"FlashMusic 发烧音质月度权益包", "¥20.00", "10.0 XFB", "2026-11-30", "可用"},
				}

					ui.Row(c.Key("coupons-row")).Gap(12).Children(func() {
						for _, cp := range coupons {
							coupon := cp
							ui.Box(c.Key("cp-"+coupon.title)).Grow(1).Radius(8).
								Background(p.bg).Border(1, p.hairline).Padding(12, 16, 12, 16).Children(func() {
								ui.Row(c).AlignItems(ui.Center).Children(func() {
								ui.Column(c).Gap(2).Children(func() {
									ui.Text(c, coupon.title).FontSize(13).FontWeight(600).TextColor(p.text)
									ui.Text(c, fmt.Sprintf("支持质押储备: %s · 有效期至: %s", coupon.xfbBack, coupon.expiry)).
										FontSize(11.5).TextColor(p.faint)
								})
								ui.Spacer(c)
								ui.Text(c, coupon.amount).FontSize(18).FontWeight(700).TextColor(p.accent)
							})
						})
					}
				})
			})
		})
	})
}

// nodeTaskHistoryTab renders Tab 4: Cloud Task History (CloudTaskHistory.tsx).
func (a *app) nodeTaskHistoryTab(c *ui.Context) {
	p := a.pal

	tasks := []struct {
		id      string
		attempt string
		action  string
		bytes   string
		status  string
		time    string
	}{
		{"task-9021-fca", "att-01", "MEDIA_SEGMENT_CACHE (短剧切片分发)", "14.2 MB", "SUCCEEDED", "14:28:10"},
		{"task-9020-e81", "att-01", "MEDIA_SEGMENT_CACHE (短剧切片分发)", "28.5 MB", "SUCCEEDED", "14:26:05"},
		{"task-9019-d74", "att-01", "PROOF_OF_DELIVERY (对等交付签名凭证)", "4.1 MB", "SUCCEEDED", "14:22:40"},
		{"task-9018-c62", "att-01", "MEDIA_SEGMENT_CACHE (短剧切片分发)", "32.0 MB", "SUCCEEDED", "14:18:12"},
		{"task-9017-b51", "att-01", "USAGE_WINDOW_REPORT (10分钟带宽审计)", "512 KB", "COMMITTED", "14:10:00"},
		{"task-9016-a40", "att-01", "MEDIA_SEGMENT_CACHE (短剧切片分发)", "19.8 MB", "SUCCEEDED", "14:04:33"},
	}

	ui.Column(c.Key("tab-tasks")).Gap(14).Children(func() {
		ui.Box(c.Key("tasks-header-card")).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(16, 20, 16, 20).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, "云端任务执行账单 (Cloud Node Task Receipts)").FontSize(15).FontWeight(700).TextColor(p.text)
					ui.Text(c, "由 FlashVCP Cloud Hub 统一下发的切片加速与防篡改加密证明账单流水。").
						FontSize(12).TextColor(p.muted)
				})
				ui.Spacer(c)
				ui.Text(c, "共 6 条近期已确认账单").FontSize(12).TextColor(p.faint)
			})
		})

		// Tasks Table
		ui.Box(c.Key("tasks-table")).Radius(10).Background(p.card).
			Border(1, p.hairline).Padding(12, 16, 12, 16).Children(func() {
			ui.Column(c).Gap(8).Children(func() {
				// Table Header
				ui.Row(c).Padding(4, 0, 8, 0).Children(func() {
					ui.Text(c, "任务标识 / ID").FontSize(12).FontWeight(600).TextColor(p.faint).Width(150)
					ui.Text(c, "任务类型与分发动作").FontSize(12).FontWeight(600).TextColor(p.faint).Grow(1)
					ui.Text(c, "有效字节数").FontSize(12).FontWeight(600).TextColor(p.faint).Width(110)
					ui.Text(c, "完成时间").FontSize(12).FontWeight(600).TextColor(p.faint).Width(100)
					ui.Text(c, "状态").FontSize(12).FontWeight(600).TextColor(p.faint).Width(90)
				})

				// Rows
				for _, row := range tasks {
					t := row
					ui.Row(c.Key("row-"+t.id)).AlignItems(ui.Center).Padding(6, 0, 6, 0).Children(func() {
						ui.Text(c, t.id).FontSize(12.5).FontWeight(600).TextColor(p.text).Width(150)
						ui.Text(c, t.action).FontSize(12.5).TextColor(p.muted).Grow(1)
						ui.Text(c, t.bytes).FontSize(12.5).FontWeight(600).TextColor(p.text).Width(110)
						ui.Text(c, t.time).FontSize(12).TextColor(p.faint).Width(100)
						ui.Box(c).Width(90).Children(func() {
							ui.Box(c).Padding(2, 6, 2, 6).Radius(4).
								Background(ui.RGBA(16, 185, 129, 0.15)).Children(func() {
								ui.Text(c, t.status).FontSize(10.5).FontWeight(700).TextColor(ui.Hex("#10b981"))
							})
						})
					})
				}
			})
		})
	})
}

func formatDurationSince(t time.Time) string {
	dur := time.Since(t)
	hours := int(dur.Hours())
	mins := int(dur.Minutes()) % 60
	if hours > 0 {
		return fmt.Sprintf("%d小时%d分", hours, mins)
	}
	return fmt.Sprintf("%d分钟", mins)
}

// -------------------------------------------------------------
// 视界中心其他辅助页面 (互动直播, 语音空间)
// -------------------------------------------------------------

// livePage renders 互动直播.
func (a *app) livePage(c *ui.Context) {
	top, bottom := a.insets(c)
	p := a.pal

	ui.Column(c.Key("live-page")).Fill().Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).Gap(16).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Icon(c, iconBroadcast).Size(22, 22).TextColor(p.accent)
			ui.Text(c, "互动直播大厅").FontSize(20).FontWeight(700).TextColor(p.text)
			ui.Box(c).Padding(2, 6, 2, 6).Radius(4).Background(ui.RGBA(239, 68, 68, 0.15)).Children(func() {
				ui.Text(c, "LIVE NOW").FontSize(10.5).FontWeight(700).TextColor(ui.Hex("#ef4444"))
			})
		})

		ui.Box(c.Key("live-banner")).Height(160).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(20).Center().Children(func() {
			ui.Column(c).Center().Gap(8).Children(func() {
				ui.Icon(c, iconBroadcast).Size(32, 32).TextColor(p.accent)
				ui.Text(c, "FlashMusic 音乐人与短剧主创连麦直播间").FontSize(16).FontWeight(700).TextColor(p.text)
				ui.Text(c, "高保真低延迟音频传输，支持与全网主播实时互动、点歌送礼与弹幕共鸣").
					FontSize(12.5).TextColor(p.muted)
			})
		})
	})
}

// voiceRoomPage renders 语音空间.
func (a *app) voiceRoomPage(c *ui.Context) {
	top, bottom := a.insets(c)
	p := a.pal

	ui.Column(c.Key("voice-room-page")).Fill().Padding(top+pageGap, pagePad-8, bottom+32, pagePad-8).Gap(16).Children(func() {
		a.startMark(c)
		defer a.endMark(c)

		ui.Row(c).AlignItems(ui.Center).Gap(10).Children(func() {
			ui.Icon(c, iconMic).Size(22, 22).TextColor(p.accent)
			ui.Text(c, "语音空间").FontSize(20).FontWeight(700).TextColor(p.text)
		})

		ui.Box(c.Key("voice-card")).Height(160).Radius(12).Background(p.card).
			Border(1, p.hairline).Padding(20).Center().Children(func() {
			ui.Column(c).Center().Gap(8).Children(func() {
				ui.Icon(c, iconMic).Size(32, 32).TextColor(p.accent)
				ui.Text(c, "多人高清语音连麦与深夜留声机").FontSize(16).FontWeight(700).TextColor(p.text)
				ui.Text(c, "基于端侧 AI 降噪技术与空间音频音效，畅享无拘束的声音陪伴").
						FontSize(12.5).TextColor(p.muted)
				})
			})
		})
	}

