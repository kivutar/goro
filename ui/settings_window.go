package ui

import (
	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/slider"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	settingsWindowW = 300
	settingsWindowH = 430
)

type SettingsWindow struct {
	Window
}

func (w *SettingsWindow) OpenWindow(ctx client.Context) {
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	w.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) Update(ctx client.Context) bool {
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *SettingsWindow) Rebind(ctx client.Context) {
	if !w.IsOpen() {
		return
	}
	w.refresh(ctx)
}

func (w *SettingsWindow) widgetTree(ctx client.Context) widget.Widget {
	return Win(
		Title("Settings"),
		CloseButton(true),
		OnClose(w.Close),
		Size(settingsWindowW, settingsWindowH),
		Content(w.contentTree(ctx)),
	)
}

func (w *SettingsWindow) contentTree(ctx client.Context) widget.Widget {
	return primitives.Box(
		rotheme.SectionLabel("Display"),

		rotheme.Checkbox(
			checkbox.Checked(settingsRuntimeFullscreen(ctx)),
			checkbox.LabelOpt("Fullscreen"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Runtime != nil {
					ctx.Runtime.SetFullscreen(enabled)
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsRuntimeVSync(ctx)),
			checkbox.LabelOpt("VSync (Restart)"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Runtime != nil {
					ctx.Runtime.SetVSync(enabled)
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsRuntimeFPS(ctx)),
			checkbox.LabelOpt("FPS meter"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Runtime != nil {
					ctx.Runtime.SetFPS(enabled)
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.SectionLabel("Sound"),

		primitives.HBox(
			rotheme.Text("BGM Vol"),
			primitives.Expanded(
				rotheme.Slider(
					slider.Min(0),
					slider.Max(1),
					slider.Value(float32(settingsVolumeBGM(ctx))),
					slider.OnChange(func(v float32) {
						if ctx.Audio != nil {
							ctx.Audio.SetBGMVolume(float64(v))
						}
						w.saveSettings(ctx)
						w.refresh(ctx)
					}),
				),
			),
		).Gap(8),

		primitives.HBox(
			rotheme.Text("SFX Vol"),
			primitives.Expanded(
				rotheme.Slider(
					slider.Min(0),
					slider.Max(1),
					slider.Value(float32(settingsVolumeSFX(ctx))),
					slider.OnChange(func(v float32) {
						if ctx.Audio != nil {
							ctx.Audio.SetSFXVolume(float64(v))
						}
						w.saveSettings(ctx)
						w.refresh(ctx)
					}),
				),
			),
		).Gap(8),

		rotheme.SectionLabel("Gameplay"),

		rotheme.Checkbox(
			checkbox.Checked(settingsNoShift(ctx)),
			checkbox.LabelOpt("No Shift"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Session != nil {
					ctx.Session.NoShift = enabled
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsNoCtrl(ctx)),
			checkbox.LabelOpt("No Ctrl"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Session != nil {
					ctx.Session.NoCtrl = enabled
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsLessEffects(ctx)),
			checkbox.LabelOpt("Less Effects"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Session != nil {
					ctx.Session.LessEffects = enabled
				}
				if ctx.Network != nil {
					_ = ctx.Network.SendLessEffect(enabled)
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsSnapTargets(ctx)),
			checkbox.LabelOpt("Snap to targets"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Session != nil {
					ctx.Session.SnapTargets = enabled
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),

		rotheme.Checkbox(
			checkbox.Checked(settingsSnapItems(ctx)),
			checkbox.LabelOpt("Snap to items"),
			checkbox.OnToggle(func(enabled bool) {
				if ctx.Session != nil {
					ctx.Session.SnapItems = enabled
				}
				w.saveSettings(ctx)
				w.refresh(ctx)
			}),
		),
	).
		Padding(14).
		Gap(8)
}

func (w *SettingsWindow) refresh(ctx client.Context) {
	w.EnsureWindow(settingsWindowW, settingsWindowH)
	w.ctx = ctx
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) saveSettings(ctx client.Context) {
	settings := config.UserSettings{
		Fullscreen:  settingsRuntimeFullscreen(ctx),
		VSync:       settingsRuntimeVSync(ctx),
		FPS:         settingsRuntimeFPS(ctx),
		BGMVolume:   settingsVolumeBGM(ctx),
		SFXVolume:   settingsVolumeSFX(ctx),
		NoShift:     settingsNoShift(ctx),
		NoCtrl:      settingsNoCtrl(ctx),
		LessEffects: settingsLessEffects(ctx),
		SnapTargets: settingsSnapTargets(ctx),
		SnapItems:   settingsSnapItems(ctx),
	}
	path, err := config.SaveUserSettings(settings)
	if err != nil {
		glog.Warnf("settings save failed: %v", err)
		return
	}
	glog.Debugf("settings saved path=%s", path)
}

func settingsVolumeBGM(ctx client.Context) float64 {
	if ctx.Audio != nil {
		return ctx.Audio.BGMVolume()
	}
	return ctx.Config.Audio.BGMVolume
}

func settingsVolumeSFX(ctx client.Context) float64 {
	if ctx.Audio != nil {
		return ctx.Audio.SFXVolume()
	}
	return ctx.Config.Audio.SFXVolume
}

func settingsRuntimeFullscreen(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.Fullscreen()
	}
	return ctx.Config.Window.Fullscreen
}

func settingsRuntimeVSync(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.VSync()
	}
	return ctx.Config.Render.VSync
}

func settingsRuntimeFPS(ctx client.Context) bool {
	if ctx.Runtime != nil {
		return ctx.Runtime.FPS()
	}
	return ctx.Config.Render.FPS
}

func settingsNoShift(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.NoShift
	}
	return ctx.Config.Gameplay.NoShift
}

func settingsNoCtrl(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.NoCtrl
	}
	return ctx.Config.Gameplay.NoCtrl
}

func settingsLessEffects(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.LessEffects
	}
	return ctx.Config.Gameplay.LessEffects
}

func settingsSnapTargets(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapTargets
	}
	return ctx.Config.Gameplay.SnapTargets
}

func settingsSnapItems(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.SnapItems
	}
	return ctx.Config.Gameplay.SnapItems
}
