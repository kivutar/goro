package render

import (
	"fmt"
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

type runtimeSettingsProvider interface {
	RuntimeFullscreen() bool
	RuntimeVSync() bool
	RuntimeFPS() bool
}

type fpsCounterReceiver interface {
	SetFPSCounter(string)
}

type runner struct {
	app            *gogpu.App
	ui             *uiapp.App
	uiCanvas       *ggcanvas.Canvas
	uiImage        *Image
	game           Game
	screen         *Image
	gpu            *gpuRenderer
	width          int
	height         int
	duration       time.Duration
	warmup         time.Duration
	renderCfg      config.RenderConfig
	started        time.Time
	measureStarted time.Time
	lastLog        time.Time
	lastFrame      int64
	frames         int64
	measuredFrames int64
	fpsStarted     time.Time
	fpsFrames      int64
	fpsDisplay     float64
	frameMSDisplay float64
	quit           func()
	cpuProfile     *os.File
	fullscreen     bool
	vsync          bool
	fps            bool
	vsyncWarned    bool
	uiDrawnOnce    bool
}

func Run(game Game, cfg config.WindowConfig, renderCfg config.RenderConfig) error {
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
	})
	return gg.Run()
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
	if r.gpu == nil {
		gpu, err := newGPURenderer(ctx, r.app, r.renderCfg)
		if err != nil {
			return err
		}
		r.gpu = gpu
		log.Printf("render backend=%s surface_format=%s", ctx.Backend(), r.gpu.format)
	}
	r.screen.BeginFrame()
	r.publishFPSCounter()
	r.game.Draw(r.screen)
	if err := r.drawUI(r.screen, width, height); err != nil {
		return err
	}
	if drawer, ok := r.game.(overlayDrawer); ok {
		drawer.DrawOverlay(r.screen)
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

func (r *runner) drawUI(screen *Image, width, height int) error {
	if r.ui == nil || screen == nil || width <= 0 || height <= 0 {
		return nil
	}
	provider := r.app.GPUContextProvider()
	if provider == nil {
		return nil
	}
	if r.uiCanvas == nil {
		canvas, err := ggcanvas.New(provider, width, height)
		if err != nil {
			return fmt.Errorf("create ui canvas: %w", err)
		}
		r.uiCanvas = canvas
	}
	canvasW, canvasH := r.uiCanvas.Size()
	if canvasW != width || canvasH != height {
		if err := r.uiCanvas.Resize(width, height); err != nil {
			return fmt.Errorf("resize ui canvas: %w", err)
		}
		r.uiDrawnOnce = false
		r.uiImage = nil
	}

	win := r.ui.Window()
	needsWork := !r.uiDrawnOnce || win.NeedsRedraw() || win.HasDirtyBoundaries() || win.NeedsAnimationFrame()
	if needsWork {
		win.ClearAnimationFrame()
		drawn := false
		if err := r.uiCanvas.Draw(func(cc *gg.Context) {
			canvas := uirender.NewCanvas(cc, width, height)
			if textMode, ok := canvas.(widget.TextModeController); ok {
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
	opts.Filter = FilterNearest
	screen.DrawImage(r.uiImage, &opts)
	return nil
}

func (r *runner) updateUIImage() {
	if r.uiCanvas == nil || r.uiCanvas.Context() == nil {
		return
	}
	src := r.uiCanvas.Context().ResizeTarget().ImageView()
	if src == nil {
		return
	}
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if r.uiImage == nil || r.uiImage.pix == nil || r.uiImage.Bounds().Dx() != width || r.uiImage.Bounds().Dy() != height {
		r.uiImage = NewImage(width, height)
	}
	dst := r.uiImage.pix
	if src.Stride == width*4 && dst.Stride == width*4 {
		copy(dst.Pix, src.Pix)
	} else {
		for y := 0; y < height; y++ {
			copy(dst.Pix[y*dst.Stride:y*dst.Stride+width*4], src.Pix[y*src.Stride:y*src.Stride+width*4])
		}
	}
	r.uiImage.version++
}

func (r *runner) updateFPSCounter(now time.Time) {
	if !r.renderCfg.FPS {
		return
	}
	if r.fpsStarted.IsZero() {
		r.fpsStarted = now
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
}

func (r *runner) publishFPSCounter() {
	receiver, ok := r.game.(fpsCounterReceiver)
	if !ok {
		return
	}
	if !r.renderCfg.FPS {
		receiver.SetFPSCounter("")
		return
	}
	text := "FPS --"
	if r.fpsDisplay > 0 {
		text = fmt.Sprintf("FPS %.1f  %.2f ms", r.fpsDisplay, r.frameMSDisplay)
	}
	receiver.SetFPSCounter(text)
}
