package ui

import (
	"fmt"
	"log"

	"github.com/gogpu/ui/core/checkbox"
	"github.com/gogpu/ui/core/slider"
	"github.com/gogpu/ui/offscreen"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	settingsWindowWidth  = 300
	settingsWindowHeight = 272
	settingsWindowTitleH = 28
)

type SettingsWindow struct {
	open       bool
	x          int
	y          int
	positioned bool
	dragging   bool
	dragDX     int
	dragDY     int
	status     string
	uiApp      client.UIApp
	root       widget.Widget
}

func (w *SettingsWindow) OpenWindow(ctx client.Context) {
	w.open = true
	w.EnsurePosition(ctx)
	w.rebuild(ctx)
}

func (w *SettingsWindow) Update(ctx client.Context) bool {
	if !w.open || ctx.Input == nil {
		return false
	}
	w.SetUIApp(ctx.UIApp)
	w.EnsurePosition(ctx)
	width, height := ctx.ScreenSize()
	if w.dragging {
		if ctx.Input.MousePressed(render.MouseButtonLeft) {
			w.x = clampSettingsWindowInt(ctx.Input.MouseX-w.dragDX, 8, maxInt(8, width-settingsWindowWidth-8))
			w.y = clampSettingsWindowInt(ctx.Input.MouseY-w.dragDY, 8, maxInt(8, height-settingsWindowHeight-8))
			w.setAppRoot()
			return true
		}
		w.dragging = false
		return true
	}
	if ctx.Input.JustPressed(render.KeyEscape) {
		w.Close()
		return true
	}
	inside := pointInRect(ctx.Input.MouseX, ctx.Input.MouseY, w.x, w.y, settingsWindowWidth, settingsWindowHeight)
	if !ctx.Input.MouseJustPressed(render.MouseButtonLeft) {
		return inside
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	if !inside {
		return false
	}
	if pointInRect(mx, my, w.x, w.y, settingsWindowWidth, settingsWindowTitleH) {
		w.dragging = true
		w.dragDX = mx - w.x
		w.dragDY = my - w.y
		return true
	}
	return true
}

func (w *SettingsWindow) Draw(screen *render.Image, ctx client.Context) {
	if !w.open || screen == nil {
		return
	}
	w.EnsurePosition(ctx)
	image := w.renderTree(ctx)
	if image == nil {
		return
	}
	var opts render.DrawImageOptions
	opts.GeoM.Translate(float64(w.x), float64(w.y))
	opts.Filter = render.FilterNearest
	screen.DrawImage(image, &opts)
}

func (w *SettingsWindow) IsOpen() bool {
	return w.open
}

func (w *SettingsWindow) Close() {
	w.open = false
	w.root = nil
	if w.uiApp != nil {
		w.uiApp.SetRoot(primitives.Box())
	}
}

func (w *SettingsWindow) SetUIApp(uiApp client.UIApp) {
	if w == nil || w.uiApp == uiApp {
		return
	}
	w.uiApp = uiApp
	w.setAppRoot()
}

func (w *SettingsWindow) rebuild(ctx client.Context) {
	w.root = w.widgetTree(ctx)
	w.setAppRoot()
}

func (w *SettingsWindow) setAppRoot() {
	if w.uiApp == nil || w.root == nil {
		return
	}
	w.uiApp.SetRoot(
		primitives.Box(w.root).
			PaddingLeft(float32(w.x)).
			PaddingTop(float32(w.y)).
			Width(float32(w.x + settingsWindowWidth)).
			Height(float32(w.y + settingsWindowHeight)),
	)
}

func (w *SettingsWindow) renderTree(ctx client.Context) *render.Image {
	if w.root == nil {
		w.rebuild(ctx)
	}
	r := offscreen.NewRenderer(settingsWindowWidth, settingsWindowHeight, offscreen.WithTheme(rotheme.Default.AsTheme()))
	r.Render(w.root)
	img := r.Image()
	if img == nil {
		return nil
	}
	return render.NewImageFromImage(img)
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
						w.saveSettings(ctx, fmt.Sprintf("fullscreen %s", settingsBoolText(enabled)))
					}),
				),

				checkbox.New(
					checkbox.Checked(settingsRuntimeVSync(ctx)),
					checkbox.LabelOpt("VSync (Restart)"),
					checkbox.OnToggle(func(enabled bool) {
						if ctx.Runtime != nil {
							ctx.Runtime.SetVSync(enabled)
						}
						w.saveSettings(ctx, "vsync saved for restart")
					}),
				),

				checkbox.New(
					checkbox.Checked(settingsRuntimeFPS(ctx)),
					checkbox.LabelOpt("FPS meter"),
					checkbox.OnToggle(func(enabled bool) {
						if ctx.Runtime != nil {
							ctx.Runtime.SetFPS(enabled)
						}
						w.saveSettings(ctx, fmt.Sprintf("fps meter %s", settingsBoolText(enabled)))
					}),
				),

				rotheme.SectionLabel("Sound"),

				primitives.HBox(
					rotheme.Text("BGM Vol"),
					slider.New(
						slider.Min(0),
						slider.Max(1),
						slider.Value(float32(settingsVolumeBGM(ctx))),
						slider.OnChange(func(v float32) {
							if ctx.Audio != nil {
								ctx.Audio.SetBGMVolume(float64(v))
							}
							w.saveSettings(ctx, fmt.Sprintf("bgm volume %d%%", int(settingsVolumeBGM(ctx)*100+0.5)))
						}),
					),
				).Gap(8),

				primitives.HBox(
					rotheme.Text("SFX Vol"),
					slider.New(
						slider.Min(0),
						slider.Max(1),
						slider.Value(float32(settingsVolumeSFX(ctx))),
						slider.OnChange(func(v float32) {
							if ctx.Audio != nil {
								ctx.Audio.SetSFXVolume(float64(v))
							}
							w.saveSettings(ctx, fmt.Sprintf("sfx volume %d%%", int(settingsVolumeSFX(ctx)*100+0.5)))
						}),
					),
				).Gap(8),
			).
				Padding(14).
				Gap(8).
				Background(rotheme.Default.Colors.WindowBody),
		),
	)
}

func (w *SettingsWindow) saveSettings(ctx client.Context, successStatus string) {
	settings := config.UserSettings{
		Fullscreen: settingsRuntimeFullscreen(ctx),
		VSync:      settingsRuntimeVSync(ctx),
		FPS:        settingsRuntimeFPS(ctx),
		BGMVolume:  settingsVolumeBGM(ctx),
		SFXVolume:  settingsVolumeSFX(ctx),
	}
	path, err := config.SaveUserSettings(settings)
	if err != nil {
		w.status = "settings save failed"
		log.Printf("settings save failed: %v", err)
		return
	}
	log.Printf("settings saved path=%s", path)
	w.status = successStatus
}

func (w *SettingsWindow) EnsurePosition(ctx client.Context) {
	if w.positioned {
		return
	}
	width, height := ctx.ScreenSize()
	w.x = maxInt(8, (width-settingsWindowWidth)/2)
	w.y = maxInt(8, (height-settingsWindowHeight)/2)
	w.positioned = true
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

func settingsBoolText(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func clampSettingsWindowInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}
