package render

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
	gogputypes "github.com/gogpu/gogpu/gpu/types"
	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	uirender "github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/config"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const BackendName = "gogpu-wgpu"

type Game interface {
	Update() error
	Draw(*Image)
	Resize(width, height int)
	InputState() *input.State
}

type quitReceiver interface {
	SetQuitFunc(func())
}

type uiAppReceiver interface {
	SetUIApp(client.UIApp)
}

type uiAppBridge struct {
	*uiapp.App
}

func (b uiAppBridge) SetUIRoot(root widget.Widget) {
	if b.App != nil {
		b.App.SetRoot(root)
		if b.App.Window() != nil && b.App.Window().Context() != nil {
			b.App.Window().Context().ResetCursor()
		}
	}
}

func (b uiAppBridge) Cursor() widget.CursorType {
	if b.App == nil || b.App.Window() == nil || b.App.Window().Context() == nil {
		return widget.CursorDefault
	}
	return b.App.Window().Context().Cursor()
}

func (b uiAppBridge) HoveredWidget() widget.Widget {
	if b.App == nil || b.App.Window() == nil {
		return nil
	}
	return b.App.Window().HoveredWidget()
}

type overlayDrawer interface {
	DrawOverlay(*Image)
}

type uiOverlayDrawer interface {
	DrawUIOverlay(*Image)
}

type runtimeSettingsProvider interface {
	RuntimeFullscreen() bool
	RuntimeVSync() bool
	RuntimeFPS() bool
}

type cachedOverlayImage struct {
	image  *Image
	width  int
	height int
}

type runner struct {
	app             *gogpu.App
	ui              *uiapp.App
	uiCanvas        *ggcanvas.Canvas
	uiImage         *Image
	uiOverlayCanvas *ggcanvas.Canvas
	uiTextCache     map[string]cachedOverlayImage
	uiBubbleCache   map[string]cachedOverlayImage
	game            Game
	screen          *Image
	gpu             *gpuRenderer
	width           int
	height          int
	duration        time.Duration
	warmup          time.Duration
	renderCfg       config.RenderConfig
	started         time.Time
	measureStarted  time.Time
	lastLog         time.Time
	lastFrame       int64
	frames          int64
	measuredFrames  int64
	fpsStarted      time.Time
	fpsFrames       int64
	fpsDisplay      float64
	frameMSDisplay  float64
	fpsText         string
	uiOverlayScale  float64
	quit            func()
	cpuProfile      *os.File
	fullscreen      bool
	vsync           bool
	fps             bool
	vsyncWarned     bool
	uiDrawnOnce     bool
	uiScale         float64
}

func Run(game Game, cfg config.WindowConfig, renderCfg config.RenderConfig) error {
	configureGogpuVSync(renderCfg)
	appConfig := gogpu.DefaultConfig()
	api, err := graphicsAPI(renderCfg.GraphicsAPI)
	if err != nil {
		return err
	}
	appConfig = appConfig.
		WithGraphicsAPI(api).
		WithTitle(cfg.Title).
		WithSize(cfg.Width, cfg.Height).
		WithResizable(true).
		WithContinuousRender(true).
		WithVSync(renderCfg.VSync && renderCfg.BenchSeconds == 0)
	if cfg.Fullscreen {
		appConfig = appConfig.WithFullscreen()
	}
	gg := gogpu.NewApp(appConfig)
	setCursorApp(gg)
	defer setCursorApp(nil)
	events := newFanoutEventSource(gg.EventSource())
	uiTheme := rotheme.Default.AsTheme()
	uiTheme.Colors.Background = widget.RGBA8(0, 0, 0, 0)
	ui := uiapp.New(
		uiapp.WithWindowProvider(gg),
		uiapp.WithPlatformProvider(gg),
		uiapp.WithEventSource(events),
		uiapp.WithTheme(uiTheme),
		uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged),
	)

	r := &runner{
		app:        gg,
		ui:         ui,
		game:       game,
		width:      cfg.Width,
		height:     cfg.Height,
		duration:   time.Duration(renderCfg.BenchSeconds) * time.Second,
		warmup:     time.Duration(renderCfg.BenchWarmupSeconds) * time.Second,
		renderCfg:  renderCfg,
		quit:       gg.Quit,
		fullscreen: cfg.Fullscreen,
		vsync:      renderCfg.VSync,
		fps:        renderCfg.FPS,
	}
	if receiver, ok := game.(quitReceiver); ok {
		receiver.SetQuitFunc(gg.Quit)
	}
	if receiver, ok := game.(uiAppReceiver); ok {
		receiver.SetUIApp(uiAppBridge{App: ui})
	}
	game.Resize(cfg.Width, cfg.Height)
	wireInput(events, game.InputState())

	gg.OnResize(func(width, height int) {
		if width <= 0 || height <= 0 {
			return
		}
		r.width, r.height = width, height
		r.screen = nil
		r.game.Resize(width, height)
	})
	gg.OnUpdate(func(float64) {
		if err := r.update(); err != nil {
			log.Printf("update error: %v", err)
			gg.Quit()
		}
	})
	gg.OnDraw(func(ctx *gogpu.Context) {
		if err := r.draw(ctx); err != nil {
			log.Printf("draw error: %v", err)
			gg.Quit()
		}
	})
	gg.OnClose(func() {
		if r.cpuProfile != nil {
			pprof.StopCPUProfile()
			_ = r.cpuProfile.Close()
			r.cpuProfile = nil
		}
		if r.gpu != nil {
			r.gpu.release()
			r.gpu = nil
		}
		if r.uiCanvas != nil {
			_ = r.uiCanvas.Close()
			r.uiCanvas = nil
		}
		if r.uiOverlayCanvas != nil {
			_ = r.uiOverlayCanvas.Close()
			r.uiOverlayCanvas = nil
		}
	})
	return gg.Run()
}

func configureGogpuVSync(renderCfg config.RenderConfig) {
	if renderCfg.VSync && renderCfg.BenchSeconds == 0 {
		return
	}
	if os.Getenv("GOGPU_WAYLAND_FRAME_CALLBACK") == "" {
		_ = os.Setenv("GOGPU_WAYLAND_FRAME_CALLBACK", "0")
	}
}

func graphicsAPI(name string) (gogputypes.GraphicsAPI, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "vulkan":
		return gogpu.GraphicsAPIVulkan, nil
	case "auto":
		return gogpu.GraphicsAPIAuto, nil
	case "dx12", "directx12", "d3d12":
		return gogpu.GraphicsAPIDX12, nil
	case "metal":
		return gogpu.GraphicsAPIMetal, nil
	case "gles", "opengl", "opengles":
		return gogpu.GraphicsAPIGLES, nil
	case "software", "soft":
		return gogpu.GraphicsAPISoftware, nil
	default:
		return gogpu.GraphicsAPIAuto, fmt.Errorf("unknown graphics api %q", name)
	}
}

type fanoutEventSource struct {
	keyPress             []func(gpucontext.Key, gpucontext.Modifiers)
	keyRelease           []func(gpucontext.Key, gpucontext.Modifiers)
	textInput            []func(string)
	mouseMove            []func(float64, float64)
	mousePress           []func(gpucontext.MouseButton, float64, float64)
	mouseRelease         []func(gpucontext.MouseButton, float64, float64)
	scroll               []func(float64, float64)
	resize               []func(int, int)
	focus                []func(bool)
	imeCompositionStart  []func()
	imeCompositionEnd    []func(string)
	imeCompositionUpdate []func(gpucontext.IMEState)
}

func newFanoutEventSource(source gpucontext.EventSource) *fanoutEventSource {
	f := &fanoutEventSource{}
	source.OnKeyPress(func(key gpucontext.Key, mods gpucontext.Modifiers) {
		for _, fn := range f.keyPress {
			fn(key, mods)
		}
	})
	source.OnKeyRelease(func(key gpucontext.Key, mods gpucontext.Modifiers) {
		for _, fn := range f.keyRelease {
			fn(key, mods)
		}
	})
	source.OnTextInput(func(text string) {
		for _, fn := range f.textInput {
			fn(text)
		}
	})
	source.OnMouseMove(func(x, y float64) {
		for _, fn := range f.mouseMove {
			fn(x, y)
		}
	})
	source.OnMousePress(func(button gpucontext.MouseButton, x, y float64) {
		for _, fn := range f.mousePress {
			fn(button, x, y)
		}
	})
	source.OnMouseRelease(func(button gpucontext.MouseButton, x, y float64) {
		for _, fn := range f.mouseRelease {
			fn(button, x, y)
		}
	})
	source.OnScroll(func(x, y float64) {
		for _, fn := range f.scroll {
			fn(x, y)
		}
	})
	source.OnResize(func(width, height int) {
		for _, fn := range f.resize {
			fn(width, height)
		}
	})
	source.OnFocus(func(focused bool) {
		for _, fn := range f.focus {
			fn(focused)
		}
	})
	source.OnIMECompositionStart(func() {
		for _, fn := range f.imeCompositionStart {
			fn()
		}
	})
	source.OnIMECompositionUpdate(func(state gpucontext.IMEState) {
		for _, fn := range f.imeCompositionUpdate {
			fn(state)
		}
	})
	source.OnIMECompositionEnd(func(committed string) {
		for _, fn := range f.imeCompositionEnd {
			fn(committed)
		}
	})
	return f
}

func (f *fanoutEventSource) OnKeyPress(fn func(gpucontext.Key, gpucontext.Modifiers)) {
	f.keyPress = append(f.keyPress, fn)
}

func (f *fanoutEventSource) OnKeyRelease(fn func(gpucontext.Key, gpucontext.Modifiers)) {
	f.keyRelease = append(f.keyRelease, fn)
}

func (f *fanoutEventSource) OnTextInput(fn func(string)) {
	f.textInput = append(f.textInput, fn)
}

func (f *fanoutEventSource) OnMouseMove(fn func(float64, float64)) {
	f.mouseMove = append(f.mouseMove, fn)
}

func (f *fanoutEventSource) OnMousePress(fn func(gpucontext.MouseButton, float64, float64)) {
	f.mousePress = append(f.mousePress, fn)
}

func (f *fanoutEventSource) OnMouseRelease(fn func(gpucontext.MouseButton, float64, float64)) {
	f.mouseRelease = append(f.mouseRelease, fn)
}

func (f *fanoutEventSource) OnScroll(fn func(float64, float64)) {
	f.scroll = append(f.scroll, fn)
}

func (f *fanoutEventSource) OnResize(fn func(int, int)) {
	f.resize = append(f.resize, fn)
}

func (f *fanoutEventSource) OnFocus(fn func(bool)) {
	f.focus = append(f.focus, fn)
}

func (f *fanoutEventSource) OnIMECompositionStart(fn func()) {
	f.imeCompositionStart = append(f.imeCompositionStart, fn)
}

func (f *fanoutEventSource) OnIMECompositionUpdate(fn func(gpucontext.IMEState)) {
	f.imeCompositionUpdate = append(f.imeCompositionUpdate, fn)
}

func (f *fanoutEventSource) OnIMECompositionEnd(fn func(string)) {
	f.imeCompositionEnd = append(f.imeCompositionEnd, fn)
}

func wireInput(events gpucontext.EventSource, state *input.State) {
	if state == nil {
		return
	}
	events.OnKeyPress(func(key gpucontext.Key, _ gpucontext.Modifiers) {
		if mapped, ok := mapKey(key); ok {
			state.SetKey(mapped, true)
		}
	})
	events.OnKeyRelease(func(key gpucontext.Key, _ gpucontext.Modifiers) {
		if mapped, ok := mapKey(key); ok {
			state.SetKey(mapped, false)
		}
	})
	events.OnMouseMove(func(x, y float64) {
		state.SetMousePosition(int(x+0.5), int(y+0.5))
	})
	events.OnMousePress(func(button gpucontext.MouseButton, x, y float64) {
		state.SetMousePosition(int(x+0.5), int(y+0.5))
		if mapped, ok := mapMouseButton(button); ok {
			state.SetMouseButton(mapped, true)
		}
	})
	events.OnMouseRelease(func(button gpucontext.MouseButton, x, y float64) {
		state.SetMousePosition(int(x+0.5), int(y+0.5))
		if mapped, ok := mapMouseButton(button); ok {
			state.SetMouseButton(mapped, false)
		}
	})
	events.OnScroll(func(x, y float64) {
		state.AddWheel(x, y)
	})
	events.OnTextInput(func(text string) {
		state.AddTextInput(text)
	})
}

func mapKey(key gpucontext.Key) (input.Key, bool) {
	switch key {
	case gpucontext.KeyEnter:
		return input.KeyEnter, true
	case gpucontext.KeyEscape:
		return input.KeyEscape, true
	case gpucontext.KeyTab:
		return input.KeyTab, true
	case gpucontext.KeyUp:
		return input.KeyArrowUp, true
	case gpucontext.KeyDown:
		return input.KeyArrowDown, true
	case gpucontext.KeyLeft:
		return input.KeyArrowLeft, true
	case gpucontext.KeyRight:
		return input.KeyArrowRight, true
	case gpucontext.KeyBackspace:
		return input.KeyBackspace, true
	case gpucontext.KeyLeftShift, gpucontext.KeyRightShift:
		return input.KeyShift, true
	case gpucontext.KeyLeftControl, gpucontext.KeyRightControl:
		return input.KeyCtrl, true
	case gpucontext.KeyF1:
		return input.KeyF1, true
	case gpucontext.KeyF2:
		return input.KeyF2, true
	case gpucontext.KeyF3:
		return input.KeyF3, true
	case gpucontext.KeyF4:
		return input.KeyF4, true
	case gpucontext.KeyF5:
		return input.KeyF5, true
	case gpucontext.KeyF6:
		return input.KeyF6, true
	case gpucontext.KeyF7:
		return input.KeyF7, true
	case gpucontext.KeyF8:
		return input.KeyF8, true
	case gpucontext.KeyF9:
		return input.KeyF9, true
	default:
		return 0, false
	}
}

func mapMouseButton(button gpucontext.MouseButton) (input.MouseButton, bool) {
	switch button {
	case gpucontext.MouseButtonLeft:
		return input.MouseButtonLeft, true
	case gpucontext.MouseButtonRight:
		return input.MouseButtonRight, true
	default:
		return 0, false
	}
}

func (r *runner) update() error {
	r.applyRuntimeSettings()
	if r.duration > 0 && r.started.IsZero() {
		r.started = time.Now()
		r.lastLog = r.started
		if path := r.renderCfg.CPUProfile; path != "" {
			file, err := os.Create(path)
			if err != nil {
				log.Printf("cpu profile start failed: %v", err)
			} else if err := pprof.StartCPUProfile(file); err != nil {
				log.Printf("cpu profile start failed: %v", err)
				_ = file.Close()
			} else {
				r.cpuProfile = file
				log.Printf("cpu profile writing %s", path)
			}
		}
		log.Printf("benchmark start duration=%s warmup=%s vsync=%v", r.duration, r.warmup, r.renderCfg.VSync)
	}
	if err := r.game.Update(); err != nil {
		return err
	}
	if r.ui != nil {
		r.ui.Frame()
	}
	reapplyCursorMode()
	if r.duration <= 0 {
		return nil
	}
	now := time.Now()
	if r.measureStarted.IsZero() && now.Sub(r.started) >= r.warmup {
		r.measureStarted = now
		r.measuredFrames = 0
		log.Printf("benchmark measure start elapsed=%.3fs", now.Sub(r.started).Seconds())
	}
	if now.Sub(r.lastLog) >= time.Second {
		elapsed := now.Sub(r.started).Seconds()
		interval := now.Sub(r.lastLog).Seconds()
		frames := r.frames - r.lastFrame
		log.Printf("benchmark fps interval=%.1f average=%.1f frames=%d elapsed=%.1fs", float64(frames)/interval, float64(r.frames)/elapsed, r.frames, elapsed)
		r.lastLog = now
		r.lastFrame = r.frames
	}
	if now.Sub(r.started) >= r.duration {
		elapsed := now.Sub(r.started).Seconds()
		measuredElapsed := elapsed
		measuredFPS := float64(r.frames) / elapsed
		if !r.measureStarted.IsZero() {
			measuredElapsed = now.Sub(r.measureStarted).Seconds()
			if measuredElapsed > 0 {
				measuredFPS = float64(r.measuredFrames) / measuredElapsed
			}
		}
		log.Printf("benchmark result fps=%.1f measured_fps=%.1f frames=%d measured_frames=%d elapsed=%.3fs measured_elapsed=%.3fs", float64(r.frames)/elapsed, measuredFPS, r.frames, r.measuredFrames, elapsed, measuredElapsed)
		if r.cpuProfile != nil {
			pprof.StopCPUProfile()
			_ = r.cpuProfile.Close()
			r.cpuProfile = nil
		}
		r.quit()
	}
	return nil
}

func (r *runner) applyRuntimeSettings() {
	provider, ok := r.game.(runtimeSettingsProvider)
	if !ok || provider == nil {
		return
	}
	if fullscreen := provider.RuntimeFullscreen(); fullscreen != r.fullscreen {
		r.app.SetFullscreen(fullscreen)
		r.fullscreen = fullscreen
	}
	if fps := provider.RuntimeFPS(); fps != r.fps {
		r.fps = fps
		r.renderCfg.FPS = fps
		r.fpsStarted = time.Time{}
		r.fpsFrames = 0
		r.fpsDisplay = 0
		r.frameMSDisplay = 0
		r.fpsText = ""
	}
	if vsync := provider.RuntimeVSync(); vsync != r.vsync {
		r.vsync = vsync
		r.renderCfg.VSync = vsync
		if !r.vsyncWarned {
			log.Printf("runtime vsync changed to %v; current gogpu backend applies vsync at startup", vsync)
			r.vsyncWarned = true
		}
	}
}

func (r *runner) draw(ctx *gogpu.Context) error {
	width, height := ctx.Size()
	if width <= 0 || height <= 0 {
		width, height = r.width, r.height
	}
	if width <= 0 || height <= 0 {
		return nil
	}
	if r.screen == nil || r.screen.Bounds().Dx() != width || r.screen.Bounds().Dy() != height {
		r.screen = NewScreenImage(width, height)
		r.width, r.height = width, height
		r.game.Resize(width, height)
	}
	framebufferW, framebufferH := ctx.FramebufferSize()
	scaleX, scaleY := framebufferScale(width, height, framebufferW, framebufferH)
	deviceScale := ctx.ScaleFactor()
	if deviceScale <= 0 {
		deviceScale = float64(scaleX)
	}
	if r.gpu == nil {
		gpu, err := newGPURenderer(ctx, r.app, r.renderCfg)
		if err != nil {
			return err
		}
		r.gpu = gpu
		log.Printf("render backend=%s surface_format=%s", ctx.Backend(), r.gpu.format)
	}
	r.screen.BeginFrame()
	r.screen.SetScreenScale(scaleX, scaleY)
	r.game.Draw(r.screen)
	if err := r.drawUIOverlay(r.screen, deviceScale); err != nil {
		return err
	}
	if err := r.drawUI(r.screen, width, height, deviceScale); err != nil {
		return err
	}
	if drawer, ok := r.game.(uiOverlayDrawer); ok {
		drawer.DrawUIOverlay(r.screen)
		if err := r.drawUIOverlay(r.screen, deviceScale); err != nil {
			return err
		}
	}
	if drawer, ok := r.game.(overlayDrawer); ok {
		drawer.DrawOverlay(r.screen)
	}
	if err := r.drawFPSMeter(r.screen, deviceScale); err != nil {
		return err
	}
	if err := r.gpu.Draw(ctx, r.screen); err != nil {
		return err
	}
	r.frames++
	if !r.measureStarted.IsZero() {
		r.measuredFrames++
	}
	r.updateFPSCounter(time.Now())
	return nil
}

func framebufferScale(width, height, framebufferW, framebufferH int) (float32, float32) {
	scaleX, scaleY := float32(1), float32(1)
	if width > 0 && framebufferW > 0 {
		scaleX = float32(framebufferW) / float32(width)
	}
	if height > 0 && framebufferH > 0 {
		scaleY = float32(framebufferH) / float32(height)
	}
	return scaleX, scaleY
}

func (r *runner) drawUI(screen *Image, width, height int, deviceScale float64) error {
	if r.renderCfg.NoUI {
		return nil
	}
	if r.ui == nil || screen == nil || width <= 0 || height <= 0 {
		return nil
	}
	provider := r.app.GPUContextProvider()
	if provider == nil {
		return nil
	}
	if deviceScale <= 0 {
		deviceScale = 1
	}
	if r.uiCanvas == nil {
		canvas, err := ggcanvas.NewWithScale(provider, width, height, deviceScale)
		if err != nil {
			return fmt.Errorf("create ui canvas: %w", err)
		}
		r.uiCanvas = canvas
		r.uiScale = deviceScale
	}
	canvasW, canvasH := r.uiCanvas.Size()
	if canvasW != width || canvasH != height {
		if err := r.uiCanvas.Resize(width, height); err != nil {
			return fmt.Errorf("resize ui canvas: %w", err)
		}
		r.uiDrawnOnce = false
		r.uiImage = nil
	}
	if r.uiScale != deviceScale {
		r.uiCanvas.SetDeviceScale(deviceScale)
		r.uiScale = deviceScale
		r.uiDrawnOnce = false
		r.uiImage = nil
	}

	win := r.ui.Window()
	needsWork := !r.uiDrawnOnce || win.NeedsRedraw() || win.HasDirtyBoundaries() || win.NeedsAnimationFrame()
	if needsWork {
		win.ClearAnimationFrame()
		drawn := false
		if err := r.uiCanvas.Draw(func(cc *gg.Context) {
			baseCanvas := uirender.NewCanvas(cc, width, height)
			canvas := widget.Canvas(scaledImageCanvas{Canvas: baseCanvas, scale: float32(deviceScale)})
			if textMode, ok := baseCanvas.(widget.TextModeController); ok {
				textMode.SetTextMode(widget.TextModeVector)
				defer textMode.SetTextMode(widget.TextModeAuto)
			}
			drawn = win.DrawTo(canvas)
		}); err != nil {
			return fmt.Errorf("draw ui canvas: %w", err)
		}
		if drawn {
			r.uiDrawnOnce = true
			if _, err := r.uiCanvas.Flush(); err != nil {
				return fmt.Errorf("flush ui canvas: %w", err)
			}
			r.updateUIImage()
		}
	}
	if !r.uiDrawnOnce || r.uiImage == nil {
		return nil
	}
	var opts DrawImageOptions
	if b := r.uiImage.Bounds(); b.Dx() > 0 && b.Dy() > 0 {
		opts.GeoM.Scale(float64(width)/float64(b.Dx()), float64(height)/float64(b.Dy()))
	}
	opts.Filter = FilterNearest
	screen.DrawImage(r.uiImage, &opts)
	return nil
}

func (r *runner) updateUIImage() {
	r.uiImage = updateCanvasImage(r.uiCanvas, r.uiImage)
}

func updateCanvasImage(canvas *ggcanvas.Canvas, dstImage *Image) *Image {
	if canvas == nil || canvas.Context() == nil {
		return dstImage
	}
	src := canvas.Context().ResizeTarget().ImageView()
	if src == nil {
		return dstImage
	}
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if dstImage == nil || dstImage.pix == nil || dstImage.Bounds().Dx() != width || dstImage.Bounds().Dy() != height {
		dstImage = NewImage(width, height)
	}
	dst := dstImage.pix
	if src.Stride == width*4 && dst.Stride == width*4 {
		copy(dst.Pix, src.Pix)
	} else {
		for y := 0; y < height; y++ {
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+width*4], src.Pix[y*src.Stride:y*src.Stride+width*4])
		}
	}
	dstImage.version++
	return dstImage
}

func (r *runner) drawUIOverlay(screen *Image, deviceScale float64) error {
	if screen == nil || (len(screen.uiRects) == 0 && len(screen.uiTextBoxes) == 0 && len(screen.uiTextLabels) == 0) {
		return nil
	}
	defer screen.clearUIOverlayCommands()
	provider := r.app.GPUContextProvider()
	if provider == nil {
		return nil
	}
	if deviceScale <= 0 {
		deviceScale = 1
	}
	if r.uiOverlayScale != deviceScale {
		r.uiOverlayScale = deviceScale
		r.uiTextCache = nil
		r.uiBubbleCache = nil
	}
	for _, rect := range screen.uiRects {
		DrawRect(screen, rect.X, rect.Y, rect.W, rect.H, rect.Color)
	}
	for _, box := range screen.uiTextBoxes {
		cached, err := r.cachedTextBoxImage(provider, box, deviceScale)
		if err != nil {
			return err
		}
		x, y := uiTextBoxPosition(screen, box, cached)
		drawCachedOverlayImage(screen, cached, x, y)
	}
	for _, label := range screen.uiTextLabels {
		cached, err := r.cachedTextLabelImage(provider, label, deviceScale)
		if err != nil {
			return err
		}
		x := label.X
		if label.Centered {
			x -= float64(cached.width) / 2
		}
		drawCachedOverlayImage(screen, cached, x, label.Y)
	}
	return nil
}

func uiTextBoxPosition(screen *Image, box UITextBoxCommand, cached cachedOverlayImage) (float64, float64) {
	switch box.Anchor {
	case UITextBoxAnchorBottomCenter:
		return box.X - float64(cached.width)/2, box.Y - float64(cached.height)
	case UITextBoxAnchorTooltipCenter:
		screenW, screenH := screen.Bounds().Dx(), screen.Bounds().Dy()
		x := box.X - float64(cached.width)/2
		y := box.Y
		if y+float64(cached.height)+8 > float64(screenH) && box.AltY > 0 {
			y = box.AltY - float64(cached.height)
		}
		x = clampOverlayFloat64(x, 8, maxOverlayFloat64(8, float64(screenW-cached.width-8)))
		y = clampOverlayFloat64(y, 8, maxOverlayFloat64(8, float64(screenH-cached.height-8)))
		return x, y
	default:
		return box.X, box.Y
	}
}

func clampOverlayFloat64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxOverlayFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func drawCachedOverlayImage(screen *Image, cached cachedOverlayImage, x, y float64) {
	if cached.image == nil || cached.width <= 0 || cached.height <= 0 {
		return
	}
	x, y = snapScreenPoint(screen, x, y)
	var opts DrawImageOptions
	if b := cached.image.Bounds(); b.Dx() > 0 && b.Dy() > 0 {
		opts.GeoM.Scale(float64(cached.width)/float64(b.Dx()), float64(cached.height)/float64(b.Dy()))
	}
	opts.GeoM.Translate(x, y)
	opts.Filter = FilterNearest
	screen.DrawImage(cached.image, &opts)
}

func (r *runner) cachedTextLabelImage(provider gpucontext.DeviceProvider, label UITextLabelCommand, deviceScale float64) (cachedOverlayImage, error) {
	size := label.Size
	if size <= 0 {
		size = rotheme.Default.Typography.TextSize
	}
	key := fmt.Sprintf("text|%.3f|%s|%t|%.1f|%08x|%08x", deviceScale, label.Text, label.Bold, size, rgbaKey(label.Foreground), rgbaKey(label.Outline))
	if cached, ok := r.uiTextCache[key]; ok {
		return cached, nil
	}
	measure, err := r.ensureOverlayCanvas(provider, 1, 1, deviceScale)
	if err != nil {
		return cachedOverlayImage{}, err
	}
	var textW float32
	if err := measure.Draw(func(cc *gg.Context) {
		canvas := uirender.NewCanvas(cc, 1, 1)
		textW = rotheme.MeasureText(canvas, label.Text, size, label.Bold)
	}); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("measure ui text overlay: %w", err)
	}
	width := int(textW + 4.999)
	if width < 4 {
		width = 4
	}
	height := int(size + 8)
	if height < 16 {
		height = 16
	}
	canvas, err := r.ensureOverlayCanvas(provider, width, height, deviceScale)
	if err != nil {
		return cachedOverlayImage{}, err
	}
	fg := widget.RGBA8(label.Foreground.R, label.Foreground.G, label.Foreground.B, label.Foreground.A)
	outline := widget.RGBA8(label.Outline.R, label.Outline.G, label.Outline.B, label.Outline.A)
	if err := canvas.Draw(func(cc *gg.Context) {
		uiCanvas := uirender.NewCanvas(cc, width, height)
		uiCanvas.Clear(widget.RGBA8(0, 0, 0, 0))
		if textMode, ok := uiCanvas.(widget.TextModeController); ok {
			textMode.SetTextMode(widget.TextModeVector)
			defer textMode.SetTextMode(widget.TextModeAuto)
		}
		bounds := geometry.NewRect(2, 2, float32(width), float32(height))
		for _, offset := range [][2]float32{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} {
			rotheme.DrawText(uiCanvas, label.Text, bounds.TranslateXY(offset[0], offset[1]), size, outline, label.Bold, widget.TextAlignLeft)
		}
		rotheme.DrawText(uiCanvas, label.Text, bounds, size, fg, label.Bold, widget.TextAlignLeft)
	}); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("draw ui text overlay: %w", err)
	}
	if _, err := canvas.Flush(); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("flush ui text overlay: %w", err)
	}
	cached := cachedOverlayImage{image: updateCanvasImage(canvas, nil), width: width, height: height}
	if r.uiTextCache == nil {
		r.uiTextCache = make(map[string]cachedOverlayImage)
	}
	r.uiTextCache[key] = cached
	trimOverlayImageCache(r.uiTextCache)
	return cached, nil
}

func (r *runner) cachedTextBoxImage(provider gpucontext.DeviceProvider, box UITextBoxCommand, deviceScale float64) (cachedOverlayImage, error) {
	text := strings.TrimSpace(box.Text)
	if text == "" {
		return cachedOverlayImage{}, nil
	}
	maxWidth := float32(box.MaxWidth)
	if maxWidth <= 0 {
		maxWidth = 0
	}
	maxLines := box.MaxLines
	if maxLines <= 0 {
		maxLines = 1
	}
	key := fmt.Sprintf("box|%.3f|%d|%.1f|%s", deviceScale, maxLines, maxWidth, text)
	style := consoleOverlayTextBoxStyle()
	style.maxLines = maxLines
	if maxWidth > 0 {
		style.minWidth = 28
		style.maxWidth = maxWidth
		style.wrap = true
	}
	return r.cachedOverlayTextBoxImage(provider, key, text, deviceScale, style)
}

type overlayTextBoxStyle struct {
	size       float32
	lineH      float32
	padX       float32
	padY       float32
	minWidth   float32
	maxWidth   float32
	maxLines   int
	wrap       bool
	background widget.Color
	foreground widget.Color
}

func consoleOverlayTextBoxStyle() overlayTextBoxStyle {
	return overlayTextBoxStyle{
		size:       rotheme.Default.Typography.TextSize,
		lineH:      14,
		padX:       8,
		padY:       6,
		minWidth:   1,
		maxLines:   1,
		background: widget.RGBA8(14, 18, 24, 188),
		foreground: widget.RGBA8(235, 242, 250, 255),
	}
}

func (r *runner) cachedOverlayTextBoxImage(provider gpucontext.DeviceProvider, key, text string, deviceScale float64, style overlayTextBoxStyle) (cachedOverlayImage, error) {
	if cached, ok := r.uiBubbleCache[key]; ok {
		return cached, nil
	}
	measure, err := r.ensureOverlayCanvas(provider, 1, 1, deviceScale)
	if err != nil {
		return cachedOverlayImage{}, err
	}
	lines := []string{text}
	if err := measure.Draw(func(cc *gg.Context) {
		canvas := uirender.NewCanvas(cc, 1, 1)
		if style.wrap {
			lines = wrapOverlayText(canvas, text, style.size, false, style.maxWidth-style.padX*2)
		}
	}); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("measure text box overlay: %w", err)
	}
	if len(lines) == 0 {
		lines = []string{text}
	}
	if style.maxLines > 0 && len(lines) > style.maxLines {
		lines = append(lines[:style.maxLines-1], ellipsizeOverlayText(measure, strings.Join(lines[style.maxLines-1:], " "), style.size, false, style.maxWidth-style.padX*2))
	}
	textWidth := float32(0)
	if err := measure.Draw(func(cc *gg.Context) {
		canvas := uirender.NewCanvas(cc, 1, 1)
		for _, line := range lines {
			if w := rotheme.MeasureText(canvas, line, style.size, false); w > textWidth {
				textWidth = w
			}
		}
	}); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("measure text box width: %w", err)
	}
	width := int(maxFloat32(style.minWidth, textWidth+style.padX*2) + 0.999)
	if style.maxWidth > 0 && width > int(style.maxWidth) {
		width = int(style.maxWidth)
	}
	if width < 1 {
		width = 1
	}
	height := int(float32(len(lines))*style.lineH + style.padY*2 + 0.999)
	canvas, err := r.ensureOverlayCanvas(provider, width, height, deviceScale)
	if err != nil {
		return cachedOverlayImage{}, err
	}
	if err := canvas.Draw(func(cc *gg.Context) {
		uiCanvas := uirender.NewCanvas(cc, width, height)
		uiCanvas.Clear(widget.RGBA8(0, 0, 0, 0))
		if textMode, ok := uiCanvas.(widget.TextModeController); ok {
			textMode.SetTextMode(widget.TextModeVector)
			defer textMode.SetTextMode(widget.TextModeAuto)
		}
		uiCanvas.DrawRect(geometry.NewRect(0, 0, float32(width), float32(height)), style.background)
		for i, line := range lines {
			y := style.padY + float32(i)*style.lineH
			rotheme.DrawText(uiCanvas, line, geometry.NewRect(style.padX, y, float32(width)-style.padX*2, style.lineH), style.size, style.foreground, false, widget.TextAlignLeft)
		}
	}); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("draw text box overlay: %w", err)
	}
	if _, err := canvas.Flush(); err != nil {
		return cachedOverlayImage{}, fmt.Errorf("flush text box overlay: %w", err)
	}
	cached := cachedOverlayImage{image: updateCanvasImage(canvas, nil), width: width, height: height}
	if r.uiBubbleCache == nil {
		r.uiBubbleCache = make(map[string]cachedOverlayImage)
	}
	r.uiBubbleCache[key] = cached
	trimOverlayImageCache(r.uiBubbleCache)
	return cached, nil
}

func trimOverlayImageCache(cache map[string]cachedOverlayImage) {
	if len(cache) <= 512 {
		return
	}
	for key := range cache {
		delete(cache, key)
		if len(cache) <= 384 {
			return
		}
	}
}

func (r *runner) ensureOverlayCanvas(provider gpucontext.DeviceProvider, width, height int, deviceScale float64) (*ggcanvas.Canvas, error) {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if r.uiOverlayCanvas == nil {
		canvas, err := ggcanvas.NewWithScale(provider, width, height, deviceScale)
		if err != nil {
			return nil, fmt.Errorf("create ui overlay canvas: %w", err)
		}
		r.uiOverlayCanvas = canvas
		r.uiOverlayScale = deviceScale
		return canvas, nil
	}
	canvasW, canvasH := r.uiOverlayCanvas.Size()
	if canvasW != width || canvasH != height {
		if err := r.uiOverlayCanvas.Resize(width, height); err != nil {
			return nil, fmt.Errorf("resize ui overlay canvas: %w", err)
		}
	}
	if r.uiOverlayScale != deviceScale {
		r.uiOverlayCanvas.SetDeviceScale(deviceScale)
		r.uiOverlayScale = deviceScale
		r.uiTextCache = nil
		r.uiBubbleCache = nil
	}
	return r.uiOverlayCanvas, nil
}

func wrapOverlayText(canvas widget.Canvas, text string, size float32, bold bool, maxWidth float32) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, 2)
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		if line == "" || rotheme.MeasureText(canvas, candidate, size, bold) <= maxWidth {
			line = candidate
			continue
		}
		lines = append(lines, line)
		line = word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func ellipsizeOverlayText(canvas *ggcanvas.Canvas, text string, size float32, bold bool, maxWidth float32) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	result := text
	_ = canvas.Draw(func(cc *gg.Context) {
		uiCanvas := uirender.NewCanvas(cc, 1, 1)
		if rotheme.MeasureText(uiCanvas, result, size, bold) <= maxWidth {
			return
		}
		const suffix = "..."
		runes := []rune(result)
		for len(runes) > 0 {
			candidate := strings.TrimSpace(string(runes)) + suffix
			if rotheme.MeasureText(uiCanvas, candidate, size, bold) <= maxWidth {
				result = candidate
				return
			}
			runes = runes[:len(runes)-1]
		}
		result = suffix
	})
	return result
}

func rgbaKey(c color.RGBA) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}

func (r *runner) updateFPSCounter(now time.Time) {
	if !r.renderCfg.FPS {
		return
	}
	if r.fpsStarted.IsZero() {
		r.fpsStarted = now
		r.fpsText = "FPS --"
		return
	}
	r.fpsFrames++
	elapsed := now.Sub(r.fpsStarted)
	if elapsed < time.Second {
		return
	}
	seconds := elapsed.Seconds()
	r.fpsDisplay = float64(r.fpsFrames) / seconds
	r.frameMSDisplay = seconds * 1000 / float64(r.fpsFrames)
	r.fpsFrames = 0
	r.fpsStarted = now
	r.fpsText = fmt.Sprintf("FPS %.1f  %.2f ms", r.fpsDisplay, r.frameMSDisplay)
}

func (r *runner) drawFPSMeter(screen *Image, deviceScale float64) error {
	if !r.renderCfg.FPS || r.fpsText == "" || screen == nil {
		return nil
	}
	provider := r.app.GPUContextProvider()
	if provider == nil {
		return nil
	}
	if deviceScale <= 0 {
		deviceScale = 1
	}
	box := UITextBoxCommand{
		Text:   r.fpsText,
		X:      6,
		Y:      6,
		Anchor: UITextBoxAnchorTopLeft,
	}
	cached, err := r.cachedTextBoxImage(provider, box, deviceScale)
	if err != nil {
		return fmt.Errorf("draw fps overlay: %w", err)
	}
	x, y := uiTextBoxPosition(screen, box, cached)
	drawCachedOverlayImage(screen, cached, x, y)
	return nil
}
