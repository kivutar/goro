package ui

import (
	"fmt"
	"image"
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/gg/scene"
	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	uirender "github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func BenchmarkStorageWindowWheelScroll(b *testing.B) {
	benchmarkListWindowWheelScroll(b, "storage", newStorageWindowScrollBenchFixture)
}

func BenchmarkCartWindowWheelScroll(b *testing.B) {
	benchmarkListWindowWheelScroll(b, "cart", newCartWindowScrollBenchFixture)
}

func benchmarkListWindowWheelScroll(
	b *testing.B,
	name string,
	newFixture func(int, listBenchCanvasKind) *listWindowScrollBenchFixture,
) {
	for _, canvasKind := range []listBenchCanvasKind{listBenchCanvasNoop, listBenchCanvasRaster} {
		b.Run(fmt.Sprintf("%s/%s", name, canvasKind), func(b *testing.B) {
			fixture := newFixture(600, canvasKind)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				fixture.step()
			}
		})
	}
}

func TestStorageWindowWheelScrollAllocationBudget(t *testing.T) {
	fixture := newStorageWindowScrollBenchFixture(600, listBenchCanvasNoop)
	fixture.step()

	allocs := testing.AllocsPerRun(100, func() {
		fixture.step()
	})
	if allocs > 75 {
		t.Fatalf("storage wheel scroll allocations = %.0f, want <= 75", allocs)
	}
}

func TestCartWindowWheelScrollAllocationBudget(t *testing.T) {
	fixture := newCartWindowScrollBenchFixture(600, listBenchCanvasNoop)
	fixture.step()

	allocs := testing.AllocsPerRun(100, func() {
		fixture.step()
	})
	if allocs > 75 {
		t.Fatalf("cart wheel scroll allocations = %.0f, want <= 75", allocs)
	}
}

func TestListWindowWheelScrollFixturesScroll(t *testing.T) {
	for _, tc := range []struct {
		name       string
		newFixture func(int, listBenchCanvasKind) *listWindowScrollBenchFixture
	}{
		{name: "storage", newFixture: newStorageWindowScrollBenchFixture},
		{name: "cart", newFixture: newCartWindowScrollBenchFixture},
	} {
		fixture := tc.newFixture(600, listBenchCanvasNoop)
		before := fixture.currentScroll()
		fixture.step()
		if after := fixture.currentScroll(); after <= before {
			t.Fatalf("%s fixture scroll = %.1f after %.1f, want to increase", tc.name, after, before)
		}
	}
}

type listBenchCanvasKind string

const (
	listBenchCanvasNoop   listBenchCanvasKind = "noop"
	listBenchCanvasRaster listBenchCanvasKind = "raster"
)

type listWindowScrollBenchFixture struct {
	app           *uiapp.App
	input         *input.State
	canvas        widget.Canvas
	canvasFactory func() widget.Canvas
	wheel         *event.WheelEvent
	maxRow        int
	currentFrame  int
	resetScroll   func()
	currentScroll func() float32
	update        func()
}

func newStorageWindowScrollBenchFixture(itemCount int, canvasKind listBenchCanvasKind) *listWindowScrollBenchFixture {
	app := newListBenchApp()
	manager := NewManager()
	uiApp := basicMenuTestApp{app: app}
	manager.SetUIApp(uiApp)
	inputState := input.NewState()
	storage := &StorageWindow{}
	sessionState := benchStorageSession(itemCount)
	ctx := Context{
		Input:     inputState,
		Session:   sessionState,
		UIApp:     uiApp,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}

	storage.OpenWindow(ctx)
	inputState.SetMousePosition(storage.x+12, storage.y+storageWindowTitleH+16)
	canvasFactory := newListBenchCanvasFactory(canvasKind)
	app.Frame()
	app.Window().DrawTo(canvasFactory())

	wheelPosition := geometry.Pt(float32(storage.x+12), float32(storage.y+storageWindowTitleH+16))
	wheel := event.NewWheelEvent(geometry.Pt(0, 1), wheelPosition, wheelPosition, event.ModNone)
	maxRow := len(sessionState.Storage.Items) - storageRows

	return &listWindowScrollBenchFixture{
		app:           app,
		input:         inputState,
		canvasFactory: canvasFactory,
		wheel:         wheel,
		maxRow:        maxRow,
		resetScroll: func() {
			storage.ensureScrollSignal().Set(0)
		},
		currentScroll: func() float32 {
			return storage.ensureScrollSignal().Get()
		},
		update: func() {
			storage.Update(ctx, nil, nil, nil)
		},
	}
}

func newCartWindowScrollBenchFixture(itemCount int, canvasKind listBenchCanvasKind) *listWindowScrollBenchFixture {
	app := newListBenchApp()
	manager := NewManager()
	uiApp := basicMenuTestApp{app: app}
	manager.SetUIApp(uiApp)
	inputState := input.NewState()
	cart := &CartWindow{}
	sessionState := benchCartSession(itemCount)
	ctx := Context{
		Input:     inputState,
		Session:   sessionState,
		UIApp:     uiApp,
		UIManager: manager,
		ScreenW:   800,
		ScreenH:   600,
	}

	cart.OpenWindow(ctx)
	inputState.SetMousePosition(cart.x+12, cart.y+ROWindowTitleHeight+cartTableHeaderH+16)
	canvasFactory := newListBenchCanvasFactory(canvasKind)
	app.Frame()
	app.Window().DrawTo(canvasFactory())

	wheelPosition := geometry.Pt(float32(cart.x+12), float32(cart.y+ROWindowTitleHeight+cartTableHeaderH+16))
	wheel := event.NewWheelEvent(geometry.Pt(0, 1), wheelPosition, wheelPosition, event.ModNone)
	maxRow := len(sessionState.Cart.Items) - cartRows

	return &listWindowScrollBenchFixture{
		app:           app,
		input:         inputState,
		canvasFactory: canvasFactory,
		wheel:         wheel,
		maxRow:        maxRow,
		resetScroll: func() {
			cart.ensureScrollSignal().Set(0)
		},
		currentScroll: func() float32 {
			return cart.ensureScrollSignal().Get()
		},
		update: func() {
			cart.Update(ctx, nil, nil, nil)
		},
	}
}

func newListBenchApp() *uiapp.App {
	return uiapp.New(
		uiapp.WithWindowProvider(gpucontext.NullWindowProvider{W: 800, H: 600}),
		uiapp.WithRenderMode(uiapp.RenderModeFrameworkManaged),
	)
}

func newListBenchCanvasFactory(kind listBenchCanvasKind) func() widget.Canvas {
	switch kind {
	case listBenchCanvasRaster:
		const width = 800
		const height = 600
		dc := gg.NewContext(width, height)
		return func() widget.Canvas {
			return uirender.NewCanvas(dc, width, height)
		}
	default:
		canvas := storageBenchCanvas{}
		return func() widget.Canvas {
			return canvas
		}
	}
}

func (f *listWindowScrollBenchFixture) step() {
	if f.maxRow > 0 && f.currentFrame%f.maxRow == 0 {
		f.resetScroll()
	}
	f.app.HandleEvent(f.wheel)
	f.update()
	f.app.Frame()
	f.canvas = f.canvasFactory()
	f.app.Window().DrawTo(f.canvas)
	f.input.EndFrame()
	f.currentFrame++
}

func benchStorageSession(count int) *session.Session {
	items := make([]session.InventoryItem, count)
	for i := range items {
		items[i] = session.InventoryItem{
			Index:      uint16(i + 1),
			ItemID:     uint16(500 + i%120),
			Amount:     (i % 99) + 1,
			Identified: true,
		}
	}
	return &session.Session{
		Storage: session.Storage{
			Open:      true,
			Amount:    count,
			MaxAmount: count,
			Items:     items,
		},
	}
}

func benchCartSession(count int) *session.Session {
	items := make([]session.InventoryItem, count)
	for i := range items {
		items[i] = session.InventoryItem{
			Index:      uint16(i + 1),
			ItemID:     uint16(500 + i%120),
			Amount:     (i % 99) + 1,
			Identified: true,
		}
	}
	return &session.Session{
		Cart: session.Cart{
			Open:      true,
			Amount:    count,
			MaxAmount: count,
			Weight:    count * 10,
			MaxWeight: count * 20,
			Items:     items,
		},
	}
}

type storageBenchCanvas struct{}

func (storageBenchCanvas) Clear(widget.Color) {}

func (storageBenchCanvas) DrawRect(geometry.Rect, widget.Color) {}

func (storageBenchCanvas) FillRectDirect(geometry.Rect, widget.Color) {}

func (storageBenchCanvas) StrokeRect(geometry.Rect, widget.Color, float32) {}

func (storageBenchCanvas) DrawRoundRect(geometry.Rect, widget.Color, float32) {}

func (storageBenchCanvas) StrokeRoundRect(geometry.Rect, widget.Color, float32, float32) {}

func (storageBenchCanvas) DrawCircle(geometry.Point, float32, widget.Color) {}

func (storageBenchCanvas) StrokeCircle(geometry.Point, float32, widget.Color, float32) {}

func (storageBenchCanvas) StrokeArc(geometry.Point, float32, float64, float64, widget.Color, float32) {
}

func (storageBenchCanvas) DrawLine(geometry.Point, geometry.Point, widget.Color, float32) {}

func (storageBenchCanvas) DrawText(string, geometry.Rect, float32, widget.Color, bool, widget.TextAlign) {
}

func (storageBenchCanvas) MeasureText(string, float32, bool) float32 { return 0 }

func (storageBenchCanvas) DrawImage(image.Image, geometry.Point) {}

func (storageBenchCanvas) PushClip(geometry.Rect) {}

func (storageBenchCanvas) PushClipRoundRect(geometry.Rect, float32) {}

func (storageBenchCanvas) PopClip() {}

func (storageBenchCanvas) PushTransform(geometry.Point) {}

func (storageBenchCanvas) PopTransform() {}

func (storageBenchCanvas) TransformOffset() geometry.Point { return geometry.Point{} }

func (storageBenchCanvas) ScreenOriginBase() geometry.Point { return geometry.Point{} }

func (storageBenchCanvas) ClipBounds() geometry.Rect { return geometry.Rect{} }

func (storageBenchCanvas) ReplayScene(*scene.Scene) {}

var _ widget.Canvas = storageBenchCanvas{}
