package ui

import (
	"log"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/slider"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	settingsWindowWidth  = 300
	settingsWindowHeight = 272
)

type SettingsWindow struct {
	window WindowState
	ctx    client.Context
}

func (w *SettingsWindow) OpenWindow(ctx client.Context) {
	w.ensureWindow()
	w.ctx = ctx
	w.window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) Update(ctx client.Context) bool {
	w.ensureWindow()
	w.ctx = ctx
	if !w.window.IsOpen() {
		return false
	}
	consumed := w.window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *SettingsWindow) IsOpen() bool {
	w.ensureWindow()
	return w.window.IsOpen()
}

func (w *SettingsWindow) Close() {
	w.ensureWindow()
	w.window.Close()
	w.Publish(w.ctx)
}

func (w *SettingsWindow) Publish(ctx client.Context) {
	w.ensureWindow()
	if ctx.UIManager == nil {
		return
	}
	if !w.window.IsOpen() {
		ctx.UIManager.Clear()
		return
	}
	ctx.UIManager.SetRoot(w.window.Widget())
}

func (w *SettingsWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(settingsWindowWidth, settingsWindowHeight)
	}
}

func (w *SettingsWindow) widgetTree(ctx client.Context) widget.Widget {
	return Window(
		Title("Settings"),
		CloseButton(true),
		OnClose(w.Close),
		Size(settingsWindowWidth, settingsWindowHeight),

		Content(
			primitives.Box(
				rotheme.SectionLabel("Display"),

				checkbox.New(
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

				checkbox.New(
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

				checkbox.New(
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
						slider.New(
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
						slider.New(
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
			).
				Padding(14).
				Gap(8).
				Background(rotheme.Default.Colors.WindowBody),
		),
	)
}

func (w *SettingsWindow) refresh(ctx client.Context) {
	w.ensureWindow()
	w.ctx = ctx
	w.window.SetRoot(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *SettingsWindow) saveSettings(ctx client.Context) {
	settings := config.UserSettings{
		Fullscreen: settingsRuntimeFullscreen(ctx),
		VSync:      settingsRuntimeVSync(ctx),
		FPS:        settingsRuntimeFPS(ctx),
		BGMVolume:  settingsVolumeBGM(ctx),
		SFXVolume:  settingsVolumeSFX(ctx),
	}
	path, err := config.SaveUserSettings(settings)
	if err != nil {
		log.Printf("settings save failed: %v", err)
		return
	}
	log.Printf("settings saved path=%s", path)
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
