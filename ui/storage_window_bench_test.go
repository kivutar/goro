package ui

import (
	"image"
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/gpucontext"
	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
)

func BenchmarkStorageWindowWheelScroll(b *testing.B) {
	fixture := newStorageWindowScrollBenchFixture(600)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fixture.step()
	}
}

func TestStorageWindowWheelScrollAllocationBudget(t *testing.T) {
	fixture := newStorageWindowScrollBenchFixture(600)
	fixture.step()

	allocs := testing.AllocsPerRun(100, fixture.step)
	if allocs > 100 {
		t.Fatalf("storage wheel scroll allocations = %.0f, want <= 100", allocs)
	}
}

type storageWindowScrollBenchFixture struct {
	app          *uiapp.App
	input        *input.State
	storage      *StorageWindow
	session      *session.Session
	ctx          Context
	canvas       storageBenchCanvas
	wheel        *event.WheelEvent
	maxRow       int
	currentFrame int
}

func newStorageWindowScrollBenchFixture(itemCount int) *storageWindowScrollBenchFixture {
	app := uiapp.New(uiapp.WithWindowProvider(gpucontext.NullWindowProvider{W: 800, H: 600}))
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
	canvas := storageBenchCanvas{}
	app.Frame()
	app.Window().DrawTo(canvas)

	wheelPosition := geometry.Pt(float32(storage.x+12), float32(storage.y+storageWindowTitleH+16))
	wheel := event.NewWheelEvent(wheelPosition, wheelPosition, geometry.Pt(0, 1), event.ModNone)
	maxRow := len(sessionState.Storage.Items) - storageRows

	return &storageWindowScrollBenchFixture{
		app:     app,
		input:   inputState,
		storage: storage,
		session: sessionState,
		ctx:     ctx,
		canvas:  canvas,
		wheel:   wheel,
		maxRow:  maxRow,
	}
}

func (f *storageWindowScrollBenchFixture) step() {
	if f.maxRow > 0 && f.currentFrame%f.maxRow == 0 {
		f.storage.ensureScrollSignal().Set(0)
	}
	f.app.HandleEvent(f.wheel)
	f.storage.Update(f.ctx, nil, nil, nil)
	f.app.Frame()
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
