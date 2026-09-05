package render

import (
	"math"
	"os"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/config"
)

type emptyUITestRoot struct {
	*primitives.BoxWidget
}

func (r *emptyUITestRoot) IsUIRootEmpty() bool { return true }

func TestConfigureGogpuVSyncDisablesWaylandFrameGateWhenVSyncOff(t *testing.T) {
	t.Setenv("GOGPU_WAYLAND_FRAME_CALLBACK", "")
	if err := os.Unsetenv("GOGPU_WAYLAND_FRAME_CALLBACK"); err != nil {
		t.Fatal(err)
	}

	configureGogpuVSync(config.RenderConfig{VSync: false})

	if got := os.Getenv("GOGPU_WAYLAND_FRAME_CALLBACK"); got != "0" {
		t.Fatalf("GOGPU_WAYLAND_FRAME_CALLBACK = %q, want 0", got)
	}
}

func TestConfigureGogpuVSyncLeavesWaylandFrameGateWhenVSyncOn(t *testing.T) {
	t.Setenv("GOGPU_WAYLAND_FRAME_CALLBACK", "")
	if err := os.Unsetenv("GOGPU_WAYLAND_FRAME_CALLBACK"); err != nil {
		t.Fatal(err)
	}

	configureGogpuVSync(config.RenderConfig{VSync: true})

	if got, ok := os.LookupEnv("GOGPU_WAYLAND_FRAME_CALLBACK"); ok {
		t.Fatalf("GOGPU_WAYLAND_FRAME_CALLBACK = %q, want unset", got)
	}
}

func TestConfigureGogpuVSyncLeavesWaylandFrameGateForVSyncBenchmarks(t *testing.T) {
	t.Setenv("GOGPU_WAYLAND_FRAME_CALLBACK", "")
	if err := os.Unsetenv("GOGPU_WAYLAND_FRAME_CALLBACK"); err != nil {
		t.Fatal(err)
	}

	configureGogpuVSync(config.RenderConfig{VSync: true, BenchSeconds: 10})

	if got, ok := os.LookupEnv("GOGPU_WAYLAND_FRAME_CALLBACK"); ok {
		t.Fatalf("GOGPU_WAYLAND_FRAME_CALLBACK = %q, want unset", got)
	}
}

func TestConfigureGogpuVSyncPreservesExplicitWaylandFrameGate(t *testing.T) {
	t.Setenv("GOGPU_WAYLAND_FRAME_CALLBACK", "1")

	configureGogpuVSync(config.RenderConfig{VSync: false})

	if got := os.Getenv("GOGPU_WAYLAND_FRAME_CALLBACK"); got != "1" {
		t.Fatalf("GOGPU_WAYLAND_FRAME_CALLBACK = %q, want explicit value", got)
	}
}

func TestRequestUIRedrawMarksCleanTreeDirty(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	root := primitives.Box().Width(24).Height(12)
	app.SetRoot(root)
	app.Frame()
	if !app.Window().DrawTo(&uitest.MockCanvas{}) {
		t.Fatal("initial UI draw did not render")
	}
	if app.Window().NeedsRedraw() {
		t.Fatal("window stayed dirty after initial draw")
	}
	if widget.NeedsRedrawInTree(root) {
		t.Fatal("root stayed dirty after initial draw")
	}

	(&runner{ui: app}).requestUIRedraw()

	if !app.Window().NeedsRedraw() {
		t.Fatal("UI redraw request did not dirty the window")
	}
	if !widget.NeedsRedrawInTree(root) {
		t.Fatal("UI redraw request did not dirty the widget tree")
	}
	if !app.Window().DrawTo(&uitest.MockCanvas{}) {
		t.Fatal("requested UI redraw did not render")
	}
	if app.Window().NeedsRedraw() {
		t.Fatal("window stayed dirty after requested redraw")
	}
	if widget.NeedsRedrawInTree(root) {
		t.Fatal("root stayed dirty after requested redraw")
	}
}

func TestSetUIImageReleasesPreviousGPUTexture(t *testing.T) {
	oldImage := NewImage(1, 1)
	nextImage := NewImage(1, 1)
	gpu := &gpuRenderer{
		textures: map[*Image]*gpuImageTexture{
			oldImage: {},
		},
	}
	r := &runner{gpu: gpu, uiImage: oldImage}

	r.setUIImage(nextImage)

	if r.uiImage != nextImage {
		t.Fatal("runner did not publish the replacement UI image")
	}
	if _, ok := gpu.textures[oldImage]; ok {
		t.Fatal("old UI image texture was not released from the GPU cache")
	}
}

func TestSetUIImageKeepsCurrentGPUTextureWhenImageUnchanged(t *testing.T) {
	image := NewImage(1, 1)
	gpu := &gpuRenderer{
		textures: map[*Image]*gpuImageTexture{
			image: {},
		},
	}
	r := &runner{gpu: gpu, uiImage: image}

	r.setUIImage(image)

	if _, ok := gpu.textures[image]; !ok {
		t.Fatal("current UI image texture was released even though the image did not change")
	}
}

func TestUIAppBridgeDiscardsPublishedImageForEmptyRoot(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	oldImage := NewImage(1, 1)
	dragImage := NewImage(1, 1)
	gpu := &gpuRenderer{
		textures: map[*Image]*gpuImageTexture{
			oldImage:  {},
			dragImage: {},
		},
	}
	r := &runner{
		ui:             app,
		gpu:            gpu,
		uiImage:        oldImage,
		uiDrawnOnce:    true,
		uiGeneration:   7,
		uiPendingLists: []uiDrawList{{ops: []uiDrawOp{func(widget.Canvas) {}}}},
		uiDrag:         uiDragLayer{image: dragImage, active: true},
	}
	bridge := uiAppBridge{App: app, runner: r}

	bridge.SetUIRoot(&emptyUITestRoot{BoxWidget: primitives.Box()})

	if r.uiImage != nil || r.uiDrawnOnce {
		t.Fatal("empty root retained the previously published UI image")
	}
	if r.uiGeneration != 8 {
		t.Fatalf("UI generation = %d, want 8", r.uiGeneration)
	}
	if len(r.uiPendingLists) != 0 {
		t.Fatal("empty root retained a pending draw list")
	}
	if r.uiDrag.active || r.uiDrag.image != nil {
		t.Fatal("empty root retained the UI drag layer")
	}
	if _, ok := gpu.textures[oldImage]; ok {
		t.Fatal("empty root retained the old UI texture")
	}
	if _, ok := gpu.textures[dragImage]; ok {
		t.Fatal("empty root retained the drag texture")
	}
}

func TestUIAppBridgeKeepsPublishedImageForNonEmptyRoot(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	image := NewImage(1, 1)
	r := &runner{ui: app, uiImage: image, uiDrawnOnce: true, uiGeneration: 7}
	bridge := uiAppBridge{App: app, runner: r}

	bridge.SetUIRoot(primitives.Box())

	if r.uiImage != image || !r.uiDrawnOnce {
		t.Fatal("non-empty root discarded the published UI image")
	}
	if r.uiGeneration != 7 {
		t.Fatalf("UI generation = %d, want 7", r.uiGeneration)
	}
}

func TestSetUIDragLayerReleasesPreviousGPUTexture(t *testing.T) {
	oldImage := NewImage(1, 1)
	nextImage := NewImage(1, 1)
	gpu := &gpuRenderer{
		textures: map[*Image]*gpuImageTexture{
			oldImage: {},
		},
	}
	r := &runner{
		gpu: gpu,
		uiDrag: uiDragLayer{
			image:  oldImage,
			active: true,
		},
	}

	r.setUIDragLayer(uiDragLayer{image: nextImage, active: true})

	if r.uiDrag.image != nextImage {
		t.Fatal("runner did not publish the replacement drag image")
	}
	if _, ok := gpu.textures[oldImage]; ok {
		t.Fatal("old drag image texture was not released from the GPU cache")
	}
}

func TestUIDragReleaseWaitsForRestoredAsyncImage(t *testing.T) {
	dragImage := NewImage(4, 4)
	restoredImage := NewImage(8, 8)
	rasterizer := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 2),
	}
	r := &runner{
		uiAsync:      rasterizer,
		uiAsyncBusy:  true,
		uiGeneration: 7,
		uiPendingLists: []uiDrawList{
			{generation: 7, ops: []uiDrawOp{func(widget.Canvas) {}}},
		},
		uiDrag: uiDragLayer{
			token:  "window",
			image:  dragImage,
			active: true,
		},
	}

	r.endUIDragLayer("window")

	if !r.uiDrag.active || !r.uiDrag.releasePending || r.uiDrag.image != dragImage {
		t.Fatal("drag image was not retained during the release handoff")
	}
	if r.uiGeneration != 8 {
		t.Fatalf("UI generation = %d, want 8", r.uiGeneration)
	}
	if len(r.uiPendingLists) != 1 || r.uiPendingLists[0].generation != 7 {
		t.Fatal("release discarded queued UI work from the previous generation")
	}
	r.completeUIDragLayerRelease()
	if !r.uiDrag.active || !r.uiDrag.releasePending {
		t.Fatal("drag handoff ended without a published replacement image")
	}

	rasterizer.done <- uiRasterResult{generation: 7, width: 800, height: 600, scale: 1, image: NewImage(8, 8)}
	r.collectAsyncUIResults(800, 600, 1)
	if !r.uiDrag.active || !r.uiDrag.releasePending {
		t.Fatal("stale hidden-window result ended the drag handoff")
	}

	r.uiAsyncBusy = true
	rasterizer.done <- uiRasterResult{generation: 8, width: 800, height: 600, scale: 1, image: restoredImage}
	r.collectAsyncUIResults(800, 600, 1)
	if r.uiDrag.active || r.uiDrag.image != nil {
		t.Fatal("restored UI image did not end the drag handoff")
	}
	if r.uiImage != restoredImage {
		t.Fatal("restored UI image was not published")
	}
}

func TestDrawUIPublishedImageKeepsDragLayerWithoutBaseImage(t *testing.T) {
	screen := NewFrame(100, 80)
	r := &runner{
		uiDrag: uiDragLayer{
			rect:           geometry.NewRect(10, 12, 30, 24),
			image:          NewImage(30, 24),
			active:         true,
			releasePending: true,
		},
	}

	if err := r.drawUIPublishedImage(screen, 100, 80); err != nil {
		t.Fatal(err)
	}
	if len(screen.commands) != 1 {
		t.Fatalf("draw commands = %d, want retained drag image", len(screen.commands))
	}
}

func TestShouldRecordAsyncUIBackpressuresWhenPendingListExists(t *testing.T) {
	r := &runner{
		uiAsyncBusy:    true,
		uiPendingLists: []uiDrawList{{}},
	}

	if r.shouldRecordAsyncUI(true) {
		t.Fatal("async UI recording was not backpressured while a raster and pending list were already queued")
	}
	if !r.lastUIWork {
		t.Fatal("backpressured UI work was not reported")
	}
}

func TestShouldRecordAsyncUIAllowsFirstPendingListWhileBusy(t *testing.T) {
	r := &runner{uiAsyncBusy: true}

	if !r.shouldRecordAsyncUI(true) {
		t.Fatal("async UI did not allow recording the first pending draw list")
	}
}

func TestCollectStaleAsyncUIResultSubmitsReplacementList(t *testing.T) {
	rasterizer := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 1),
	}
	replacement := uiDrawList{generation: 2, ops: []uiDrawOp{func(widget.Canvas) {}}}
	r := &runner{
		uiAsync:        rasterizer,
		uiAsyncBusy:    true,
		uiGeneration:   2,
		uiPendingLists: []uiDrawList{replacement},
	}
	rasterizer.done <- uiRasterResult{generation: 1, width: 800, height: 600, scale: 1}

	r.collectAsyncUIResults(800, 600, 1)

	if !r.uiAsyncBusy {
		t.Fatal("replacement UI list was not submitted")
	}
	if len(r.uiPendingLists) != 0 {
		t.Fatal("submitted replacement UI list remained pending")
	}
	select {
	case job := <-rasterizer.jobs:
		if job.list.generation != 2 {
			t.Fatalf("replacement generation = %d, want 2", job.list.generation)
		}
		if len(job.list.ops) != len(replacement.ops) {
			t.Fatalf("replacement operations = %d, want %d", len(job.list.ops), len(replacement.ops))
		}
	default:
		t.Fatal("replacement UI job was not queued")
	}
}

func TestCollectAsyncUIResultDoesNotPublishBeforePendingList(t *testing.T) {
	rasterizer := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 1),
	}
	published := NewImage(8, 8)
	intermediate := NewImage(8, 8)
	replacement := uiDrawList{generation: 2, ops: []uiDrawOp{func(widget.Canvas) {}}}
	r := &runner{
		uiImage:        published,
		uiDrawnOnce:    true,
		uiAsync:        rasterizer,
		uiAsyncBusy:    true,
		uiGeneration:   2,
		uiPendingLists: []uiDrawList{replacement},
	}
	rasterizer.done <- uiRasterResult{generation: 2, width: 800, height: 600, scale: 1, image: intermediate}

	r.collectAsyncUIResults(800, 600, 1)

	if r.uiImage != published {
		t.Fatal("intermediate UI image was published while a newer draw list was pending")
	}
	if !r.uiDrawnOnce {
		t.Fatal("withholding an intermediate image discarded the published UI state")
	}
	if !r.uiAsyncBusy {
		t.Fatal("pending UI list was not submitted after withholding the intermediate image")
	}
	select {
	case job := <-rasterizer.jobs:
		if job.list.generation != replacement.generation {
			t.Fatalf("replacement generation = %d, want %d", job.list.generation, replacement.generation)
		}
	default:
		t.Fatal("replacement UI job was not queued")
	}
}

func TestDrawUIAsyncPublishesFirstResultWithStaleDirtyBoundary(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	root := primitives.Box().Width(24).Height(12)
	app.SetRoot(root)
	app.Frame()
	if !app.Window().DrawTo(&uitest.MockCanvas{}) {
		t.Fatal("initial UI draw did not render")
	}
	app.Window().AddDirtyBoundary(1)

	rasterizer := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 1),
	}
	first := NewImage(8, 8)
	r := &runner{
		ui:              app,
		uiAsync:         rasterizer,
		uiAsyncBusy:     true,
		uiGeneration:    2,
		uiLogicalWidth:  800,
		uiLogicalHeight: 600,
		uiScale:         1,
	}
	rasterizer.done <- uiRasterResult{generation: 2, width: 800, height: 600, scale: 1, image: first}

	if err := r.drawUIAsync(NewFrame(800, 600), 800, 600, 1); err != nil {
		t.Fatal(err)
	}

	if r.uiImage != first || !r.uiDrawnOnce {
		t.Fatal("first completed UI image was not published")
	}
}

func TestDrawUIAsyncRecordsChangesBeforePublishingCompletedResult(t *testing.T) {
	app := uiapp.New(uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged))
	root := primitives.Box().Width(24).Height(12)
	app.SetRoot(root)
	app.Frame()
	if !app.Window().DrawTo(&uitest.MockCanvas{}) {
		t.Fatal("initial UI draw did not render")
	}

	rasterizer := &asyncUIRasterizer{
		jobs: make(chan uiRasterJob, 1),
		done: make(chan uiRasterResult, 1),
	}
	published := NewImage(8, 8)
	obsolete := NewImage(8, 8)
	r := &runner{
		ui:              app,
		uiImage:         published,
		uiDrawnOnce:     true,
		uiAsync:         rasterizer,
		uiAsyncBusy:     true,
		uiGeneration:    2,
		uiLogicalWidth:  800,
		uiLogicalHeight: 600,
		uiScale:         1,
	}
	r.requestUIRedraw()
	rasterizer.done <- uiRasterResult{generation: 2, width: 800, height: 600, scale: 1, image: obsolete}

	if err := r.drawUIAsync(NewFrame(800, 600), 800, 600, 1); err != nil {
		t.Fatal(err)
	}

	if r.uiImage != published {
		t.Fatal("completed UI image was published before newer changes were recorded")
	}
	if !r.uiAsyncBusy {
		t.Fatal("newer UI draw list was not submitted")
	}
	select {
	case job := <-rasterizer.jobs:
		if len(job.list.ops) == 0 {
			t.Fatal("newer UI job did not contain draw operations")
		}
	default:
		t.Fatal("newer UI job was not queued")
	}
}

func TestDrawActorLabelOverlayUsesSharedSnappedOrigin(t *testing.T) {
	screen := NewFrame(320, 240)
	screen.SetScreenScale(1.25, 1.5)
	cached := cachedOverlayImage{
		image:  NewImage(120, 68),
		width:  60,
		height: 34,
	}
	label := UIActorLabelCommand{
		Emblem:  NewImage(24, 24),
		CenterX: 100.82,
		Y:       50.42,
	}

	drawActorLabelOverlay(screen, cached, label)

	if len(screen.commands) != 2 {
		t.Fatalf("draw commands = %d, want text and emblem", len(screen.commands))
	}
	textOrigin := screen.commands[0].Vertices[0]
	emblemOrigin := screen.commands[1].Vertices[0]
	blockLeft, blockTop := snapScreenPoint(screen, label.CenterX-float64(cached.width+actorLabelEmblemSize+actorLabelEmblemGap)/2, label.Y)
	assertFloatClose(t, "text x", float64(textOrigin.DstX), blockLeft+actorLabelEmblemSize+actorLabelEmblemGap)
	assertFloatClose(t, "text y", float64(textOrigin.DstY), blockTop)
	assertFloatClose(t, "emblem x", float64(emblemOrigin.DstX), blockLeft+2)
	assertFloatClose(t, "emblem y", float64(emblemOrigin.DstY), blockTop+5)
}

func TestUIDragLayerDrawRectPreservesPhysicalCropSize(t *testing.T) {
	r := &runner{
		uiImage: NewImage(1000, 750),
		width:   800,
		height:  600,
	}
	frame := geometry.NewRect(11, 21, 100, 80)
	capture := r.captureUIImageRect(frame)
	if capture.image == nil {
		t.Fatal("capture returned nil image")
	}
	if got := capture.image.Bounds().Dx(); got != 126 {
		t.Fatalf("capture width = %d, want 126 physical px", got)
	}
	if got := capture.image.Bounds().Dy(); got != 101 {
		t.Fatalf("capture height = %d, want 101 physical px", got)
	}

	if !r.beginUIDragLayer("window", frame) {
		t.Fatal("drag layer did not start")
	}
	r.moveUIDragLayer("window", geometry.NewRect(40, 55, 100, 80))
	drawRect := r.uiDrag.drawRect()
	assertFloatClose(t, "draw x", float64(drawRect.Min.X), 39.4)
	assertFloatClose(t, "draw y", float64(drawRect.Min.Y), 54.8)
	assertFloatClose(t, "draw width", float64(drawRect.Width()), 100.8)
	assertFloatClose(t, "draw height", float64(drawRect.Height()), 80.8)
}

func assertFloatClose(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("%s = %.4f, want %.4f", label, got, want)
	}
}
