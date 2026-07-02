# gogpu/ui Settings Window Sketch

This is the intended shape for future gogpu/ui-based RO windows. The window body
should stay as a readable widget tree, while RO-specific title bars, footers,
frames, and painters live in reusable packages such as `rowindow` and `rotheme`.
Input should use canonical gogpu/ui callbacks such as `checkbox.OnToggle`,
`slider.OnChange`, and `button.OnClick`. Keep callbacks inline when they are a
single direct state update; move them to named functions when they also save
settings, update status text, or touch multiple systems.

```go
func settingsWindowExample(ctx settingsContext) widget.Widget {
	return rowindow.New(
		rowindow.Title("Settings"),
		rowindow.CloseButton(true),

		rowindow.Content(
			primitives.Box(
				primitives.Text("Display").
					FontSize(14).Bold().
					Color(rotheme.TitleText),

				checkbox.New(
					checkbox.Checked(ctx.Fullscreen()),
					checkbox.LabelOpt("Fullscreen"),
					checkbox.OnToggle(func(enabled bool) {
						saveFullscreenSetting(ctx, enabled)
					}),
				),

				checkbox.New(
					checkbox.Checked(ctx.VSync()),
					checkbox.LabelOpt("VSync (Restart)"),
					checkbox.OnToggle(func(enabled bool) {
						saveVSyncSetting(ctx, enabled)
					}),
				),

				checkbox.New(
					checkbox.Checked(ctx.FPS()),
					checkbox.LabelOpt("FPS meter"),
					checkbox.OnToggle(func(enabled bool) {
						saveFPSSetting(ctx, enabled)
					}),
				),

				primitives.Text("Sound").
					FontSize(14).Bold().
					Color(rotheme.TitleText),

				primitives.HBox(
					primitives.Text("BGM Vol").FontSize(14),
					slider.New(
						slider.Min(0),
						slider.Max(1),
						slider.Value(float32(ctx.BGMVolume())),
						slider.OnChange(func(v float32) {
							saveBGMVolume(ctx, float64(v))
						}),
					),
				).Gap(8),

				primitives.HBox(
					primitives.Text("SFX Vol").FontSize(14),
					slider.New(
						slider.Min(0),
						slider.Max(1),
						slider.Value(float32(ctx.SFXVolume())),
						slider.OnChange(func(v float32) {
							saveSFXVolume(ctx, float64(v))
						}),
					),
				).Gap(8),
			).
				Padding(14).
				Gap(8).
				Background(rotheme.WindowBody),
		),

		rowindow.Footer(
			button.New(
				button.TextOpt("OK"),
				button.VariantOpt(button.Filled),
				button.OnClick(ctx.CloseSettings),
			),
		),
	)
}

func saveFullscreenSetting(ctx settingsContext, enabled bool) {
	ctx.SetFullscreen(enabled)
	ctx.SaveSettings()
	ctx.SetStatus("fullscreen saved")
}

func saveVSyncSetting(ctx settingsContext, enabled bool) {
	ctx.SetVSync(enabled)
	ctx.SaveSettings()
	ctx.SetStatus("vsync saved for restart")
}

func saveFPSSetting(ctx settingsContext, enabled bool) {
	ctx.SetFPS(enabled)
	ctx.SaveSettings()
	ctx.SetStatus("fps meter saved")
}

func saveBGMVolume(ctx settingsContext, volume float64) {
	ctx.SetBGMVolume(volume)
	ctx.SaveSettings()
}

func saveSFXVolume(ctx settingsContext, volume float64) {
	ctx.SetSFXVolume(volume)
	ctx.SaveSettings()
}
```
