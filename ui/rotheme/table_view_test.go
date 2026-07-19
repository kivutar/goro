package rotheme

import (
	"context"
	"image"
	"testing"

	"github.com/gogpu/gg/scene"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

func TestTableViewDefaultHeaderHeightUsesPadding(t *testing.T) {
	table := TableView()
	want := Default.Typography.TextSize + TableHeaderPadY*2
	if table.cfg.headerHeight != want {
		t.Fatalf("default header height = %.1f, want %.1f", table.cfg.headerHeight, want)
	}
}

func TestTableViewHeaderKeepsScrollBodyBelowHeader(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 100}}),
		TableViewRowCount(10),
		TableViewRowHeight(10),
		TableViewHeaderHeight(16),
	)
	table.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(120, 80)))
	table.SetBounds(geometry.NewRect(0, 0, 120, 80))
	table.updateScrollBounds()

	bounds := table.scroll.Bounds()
	if bounds.Min.Y != 16 || bounds.Height() != 64 {
		t.Fatalf("scroll bounds = %v, want y=16 height=64", bounds)
	}
}

func TestTableViewHeaderDrawsAcrossScrollbarGutter(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Title: "Name", Width: 100}}),
	)
	canvas := &tableViewHeaderCanvas{}
	table.drawHeader(canvas, geometry.NewRect(0, 0, 120, 20), 100)

	if len(canvas.rects) == 0 || canvas.rects[0].Width() != 120 {
		t.Fatalf("header rects = %v, want first rect width 120", canvas.rects)
	}
	if len(canvas.lines) == 0 || canvas.lines[0].to.X != 120 {
		t.Fatalf("header lines = %v, want line ending at x=120", canvas.lines)
	}
	if len(canvas.texts) == 0 || canvas.texts[0].bounds.Min.Y != TableHeaderPadY {
		t.Fatalf("header texts = %v, want top padding %.1f", canvas.texts, TableHeaderPadY)
	}
}

type tableViewHeaderCanvas struct {
	rects          []geometry.Rect
	clip           geometry.Rect
	transform      geometry.Point
	transformStack []geometry.Point
	lines          []struct {
		from geometry.Point
		to   geometry.Point
	}
	texts []struct {
		text   string
		bounds geometry.Rect
	}
}

func (c *tableViewHeaderCanvas) Clear(widget.Color) {}

func (c *tableViewHeaderCanvas) DrawRect(r geometry.Rect, _ widget.Color) {
	c.rects = append(c.rects, r)
}

func (c *tableViewHeaderCanvas) FillRectDirect(geometry.Rect, widget.Color) {}

func (c *tableViewHeaderCanvas) StrokeRect(geometry.Rect, widget.Color, float32) {}

func (c *tableViewHeaderCanvas) DrawRoundRect(geometry.Rect, widget.Color, float32) {}

func (c *tableViewHeaderCanvas) StrokeRoundRect(geometry.Rect, widget.Color, float32, float32) {}

func (c *tableViewHeaderCanvas) DrawCircle(geometry.Point, float32, widget.Color) {}

func (c *tableViewHeaderCanvas) StrokeCircle(geometry.Point, float32, widget.Color, float32) {}

func (c *tableViewHeaderCanvas) StrokeArc(geometry.Point, float32, float64, float64, widget.Color, float32) {
}

func (c *tableViewHeaderCanvas) DrawLine(from, to geometry.Point, _ widget.Color, _ float32) {
	c.lines = append(c.lines, struct {
		from geometry.Point
		to   geometry.Point
	}{from: from, to: to})
}

func (c *tableViewHeaderCanvas) DrawText(text string, bounds geometry.Rect, _ float32, _ widget.Color, _ bool, _ widget.TextAlign) {
	c.texts = append(c.texts, struct {
		text   string
		bounds geometry.Rect
	}{text: text, bounds: bounds})
}

func (c *tableViewHeaderCanvas) MeasureText(string, float32, bool) float32 { return 0 }

func (c *tableViewHeaderCanvas) DrawImage(image.Image, geometry.Point) {}

func (c *tableViewHeaderCanvas) PushClip(geometry.Rect) {}

func (c *tableViewHeaderCanvas) PushClipRoundRect(geometry.Rect, float32) {}

func (c *tableViewHeaderCanvas) PopClip() {}

func (c *tableViewHeaderCanvas) PushTransform(offset geometry.Point) {
	c.transformStack = append(c.transformStack, offset)
	c.transform = c.transform.Add(offset)
}

func (c *tableViewHeaderCanvas) PopTransform() {
	if len(c.transformStack) == 0 {
		return
	}
	offset := c.transformStack[len(c.transformStack)-1]
	c.transformStack = c.transformStack[:len(c.transformStack)-1]
	c.transform = c.transform.Sub(offset)
}

func (c *tableViewHeaderCanvas) TransformOffset() geometry.Point { return c.transform }

func (c *tableViewHeaderCanvas) ScreenOriginBase() geometry.Point { return geometry.Point{} }

func (c *tableViewHeaderCanvas) ClipBounds() geometry.Rect { return c.clip }

func (c *tableViewHeaderCanvas) ReplayScene(*scene.Scene) {}

var _ widget.Canvas = (*tableViewHeaderCanvas)(nil)

func TestTableViewReusesScrollSignalAcrossRebuild(t *testing.T) {
	scrollY := state.NewSignal[float32](32)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 100}}),
		TableViewRowCount(20),
		TableViewRowHeight(10),
		TableViewScrollYSignal(scrollY),
	)
	table.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(120, 80)))

	rebuilt := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 100}}),
		TableViewRowCount(20),
		TableViewRowHeight(10),
		TableViewScrollYSignal(scrollY),
	)
	rebuilt.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(120, 80)))

	if _, got := rebuilt.scroll.ScrollOffset(); got != 32 {
		t.Fatalf("scrollY after rebuild = %.1f, want 32.0", got)
	}
}

func TestTableViewDispatchesMouseEventsToCellWidgets(t *testing.T) {
	probe := &tableViewProbe{}
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "action", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return probe
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(80, 60)))
	table.SetBounds(geometry.NewRect(0, 0, 80, 60))

	consumed := table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		0,
		geometry.Pt(8, 18),
		geometry.Pt(8, 18),
		0,
	))

	if !consumed {
		t.Fatal("mouse press was not consumed by cell widget")
	}
	if !probe.pressed {
		t.Fatal("cell widget did not receive mouse press")
	}
}

func TestTableViewMouseMoveDoesNotRelayoutCachedRow(t *testing.T) {
	probe := &tableViewLayoutProbe{}
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return probe
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 80)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 80))
	table.body.layoutRow(ctx, 0)

	if probe.layouts != 1 {
		t.Fatalf("initial layouts = %d, want 1", probe.layouts)
	}

	table.Event(ctx, event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		geometry.Pt(8, 18),
		geometry.Pt(8, 18),
		0,
	))

	if probe.layouts != 1 {
		t.Fatalf("layouts after mouse move = %d, want 1", probe.layouts)
	}
}

func TestTableViewHoverInvalidatesOnlyChangedRows(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(10),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
	)
	ctx := widget.NewContext()
	var invalidated []geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) {
		invalidated = append(invalidated, r)
	})
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 70)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 70))
	widget.StampScreenOrigin(table, &tableViewHeaderCanvas{})
	table.Draw(ctx, &tableViewHeaderCanvas{})
	widget.ClearRedrawInTree(table)

	row0 := table.body.rows[0]
	row1 := table.body.rows[1]
	if row0 == nil || row1 == nil {
		t.Fatal("expected visible rows to be cached after draw")
	}

	table.setHoveredRow(ctx, 0)
	if table.NeedsRedraw() {
		t.Fatal("table should stay clean when hover changes")
	}
	if !row0.NeedsRedraw() {
		t.Fatal("new hovered row should be locally dirty")
	}
	if len(invalidated) != 1 {
		t.Fatalf("invalidated rect count = %d, want 1", len(invalidated))
	}
	if row0.ScreenBounds().Height() != invalidated[0].Height() {
		t.Fatalf("invalidated row height = %.1f, want %.1f", invalidated[0].Height(), row0.ScreenBounds().Height())
	}

	row0.ClearRedraw()
	invalidated = invalidated[:0]
	table.setHoveredRow(ctx, 1)
	if table.NeedsRedraw() {
		t.Fatal("table should stay clean when hover moves")
	}
	if !row0.NeedsRedraw() {
		t.Fatal("previous hovered row should be locally dirty")
	}
	if !row1.NeedsRedraw() {
		t.Fatal("new hovered row should be locally dirty")
	}
	if len(invalidated) != 2 {
		t.Fatalf("invalidated rect count = %d, want 2", len(invalidated))
	}
	for _, r := range invalidated {
		if r.Width() >= table.Bounds().Width() || r.Height() > table.cfg.rowHeight {
			t.Fatalf("invalidated rect %v should be row-sized, table bounds %v", r, table.Bounds())
		}
	}
}

func TestTableViewMouseMoveBetweenRowsInvalidatesHoverRows(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(3),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
	)
	ctx := widget.NewContext()
	var invalidated []geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) {
		invalidated = append(invalidated, r)
	})
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 70)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 70))
	widget.StampScreenOrigin(table, &tableViewHeaderCanvas{})
	table.Draw(ctx, &tableViewHeaderCanvas{})
	widget.ClearRedrawInTree(table)

	table.Event(ctx, event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		geometry.Pt(8, 18),
		geometry.Pt(8, 18),
		0,
	))
	if table.hoveredRow != 0 {
		t.Fatalf("hovered row = %d, want 0", table.hoveredRow)
	}
	if len(invalidated) != 1 {
		t.Fatalf("invalidated rect count after first hover = %d, want 1", len(invalidated))
	}

	invalidated = invalidated[:0]
	table.Event(ctx, event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		geometry.Pt(8, 38),
		geometry.Pt(8, 38),
		0,
	))
	if table.hoveredRow != 1 {
		t.Fatalf("hovered row = %d, want 1", table.hoveredRow)
	}
	if len(invalidated) != 2 {
		t.Fatalf("invalidated rect count after row change = %d, want 2", len(invalidated))
	}
}

func TestTableViewCanDisableHoverInvalidation(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
		TableViewInvalidateHover(false),
	)
	ctx := widget.NewContext()
	invalidations := 0
	ctx.SetOnInvalidateRect(func(geometry.Rect) {
		invalidations++
	})
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 70)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 70))
	table.body.layoutRow(ctx, 0)
	widget.ClearRedrawInTree(table)

	table.setHoveredRow(ctx, 0)

	if table.hoveredRow != 0 {
		t.Fatalf("hovered row = %d, want 0", table.hoveredRow)
	}
	if invalidations != 0 {
		t.Fatalf("hover invalidations = %d, want 0", invalidations)
	}
	if table.body.rows[0].NeedsRedraw() {
		t.Fatal("row should stay clean when hover invalidation is disabled")
	}
}

func TestTableViewCanSkipCellHoverDispatch(t *testing.T) {
	probe := &tableViewHoverProbe{}
	var rowEvents []event.MouseEventType
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewHeaderHeight(10),
		TableViewInvalidateHover(false),
		TableViewDispatchHoverToCells(false),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return probe
		}),
		TableViewOnRowEvent(func(row int, e event.Event) bool {
			mouse, ok := e.(*event.MouseEvent)
			if ok && row == 0 {
				rowEvents = append(rowEvents, mouse.MouseType)
			}
			return false
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 70)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 70))

	table.Event(ctx, event.NewMouseEvent(
		event.MouseMove,
		event.ButtonNone,
		0,
		geometry.Pt(8, 18),
		geometry.Pt(8, 18),
		0,
	))
	table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		0,
		geometry.Pt(8, 18),
		geometry.Pt(8, 18),
		0,
	))

	if probe.moves != 0 {
		t.Fatalf("cell mouse moves = %d, want 0", probe.moves)
	}
	if !probe.pressed {
		t.Fatal("cell mouse press should still be dispatched")
	}
	if len(rowEvents) != 1 || rowEvents[0] != event.MouseMove {
		t.Fatalf("row events = %v, want [MouseMove]", rowEvents)
	}
}

func TestTableViewCentersCellWidgetVertically(t *testing.T) {
	probe := &tableViewFixedProbe{size: geometry.Sz(20, 10)}
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(32),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return probe
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 70)))
	table.body.layoutRow(ctx, 0)

	if got, want := probe.Bounds().Min.Y, float32(11); got != want {
		t.Fatalf("cell child y = %.1f, want %.1f", got, want)
	}
}

func TestTableViewBuildSimpleCellDrawsWithoutCellWidgets(t *testing.T) {
	calls := 0
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(32),
		TableViewShowHeader(false),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			calls++
			return TableViewSimpleCell{Text: "Row"}
		}),
	)
	ctx := widget.NewContext()
	canvas := &tableViewHeaderCanvas{}
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 40)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 40))
	table.Draw(ctx, canvas)

	if calls == 0 {
		t.Fatal("simple cell builder was not called")
	}
	if got := len(table.body.rows); got != 0 {
		t.Fatalf("cached widget row count = %d, want 0", got)
	}
	if got := len(table.body.simpleRows); got != 1 {
		t.Fatalf("cached simple row count = %d, want 1", got)
	}
	if children := table.body.Children(); len(children) != 1 {
		t.Fatalf("body children = %d, want 1 simple row", len(children))
	}
	if len(canvas.texts) != 1 {
		t.Fatalf("drawn texts = %d, want 1", len(canvas.texts))
	}
	if got := canvas.texts[0].bounds.Min.Y; got <= 4 {
		t.Fatalf("text y = %.1f, want vertically centered away from top", got)
	}
}

func TestTableViewBuildSimpleCellDrawsIconButtonWithoutRowWidgets(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "action", Width: 24}}),
		TableViewRowCount(1),
		TableViewRowHeight(32),
		TableViewShowHeader(false),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewIconButtonCell(IconButtonPlus, false)
		}),
	)
	ctx := widget.NewContext()
	canvas := &tableViewHeaderCanvas{}
	table.Layout(ctx, geometry.Tight(geometry.Sz(40, 40)))
	table.SetBounds(geometry.NewRect(0, 0, 40, 40))
	table.Draw(ctx, canvas)

	if got := len(table.body.rows); got != 0 {
		t.Fatalf("cached widget row count = %d, want 0", got)
	}
	if got := len(table.body.simpleRows); got != 1 {
		t.Fatalf("cached simple row count = %d, want 1", got)
	}
	if len(canvas.lines) < 2 {
		t.Fatalf("drawn icon lines = %d, want at least 2", len(canvas.lines))
	}
}

func TestTableViewBuildSimpleCellHandlesRowClick(t *testing.T) {
	clicked := -1
	selected := state.NewSignal[int](-1)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(32),
		TableViewShowHeader(false),
		TableViewSelectedRow(selected),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewSimpleCell{Text: "Row"}
		}),
		TableViewOnRowClick(func(row int) {
			clicked = row
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 40)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 40))

	consumed := table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		event.ButtonStateLeft,
		geometry.Pt(8, 8),
		geometry.Pt(8, 8),
		0,
	))

	if !consumed {
		t.Fatal("simple row click was not consumed")
	}
	if clicked != 0 {
		t.Fatalf("clicked row = %d, want 0", clicked)
	}
	if selected.Get() != 0 {
		t.Fatalf("selected row = %d, want 0", selected.Get())
	}
}

func TestTableViewBuildSimpleCellIgnoresRightClick(t *testing.T) {
	clicked := -1
	selected := state.NewSignal[int](-1)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(32),
		TableViewShowHeader(false),
		TableViewSelectedRow(selected),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewSimpleCell{Text: "Row"}
		}),
		TableViewOnRowClick(func(row int) {
			clicked = row
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 40)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 40))

	consumed := table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonRight,
		event.ButtonStateRight,
		geometry.Pt(8, 8),
		geometry.Pt(8, 8),
		0,
	))

	if consumed {
		t.Fatal("right click should not be consumed by fallback row selection")
	}
	if clicked != -1 {
		t.Fatalf("clicked row = %d, want -1", clicked)
	}
	if selected.Get() != -1 {
		t.Fatalf("selected row = %d, want -1", selected.Get())
	}
}

func TestTableViewSelectionInvalidatesChangedRows(t *testing.T) {
	selected := state.NewSignal[int](-1)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(3),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
		TableViewSelectedRow(selected),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewSimpleCell{Text: "Row"}
		}),
	)
	ctx := widget.NewContext()
	var invalidated []geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) {
		invalidated = append(invalidated, r)
	})
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 60)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 60))
	widget.StampScreenOrigin(table, &tableViewHeaderCanvas{})
	table.Draw(ctx, &tableViewHeaderCanvas{})
	widget.ClearRedrawInTree(table)

	row0 := table.body.simpleRows[0]
	row1 := table.body.simpleRows[1]
	if row0 == nil || row1 == nil {
		t.Fatal("expected visible rows to be cached after draw")
	}

	table.setSelectedRow(ctx, 0)
	if table.NeedsRedraw() {
		t.Fatal("table should stay clean when selection changes internally")
	}
	if !row0.NeedsRedraw() {
		t.Fatal("new selected row should be locally dirty")
	}
	if len(invalidated) != 1 {
		t.Fatalf("invalidated rect count = %d, want 1", len(invalidated))
	}
	if invalidated[0].Height() != table.cfg.rowHeight {
		t.Fatalf("invalidated rect height = %.1f, want %.1f", invalidated[0].Height(), table.cfg.rowHeight)
	}

	row0.ClearRedraw()
	invalidated = invalidated[:0]
	table.setSelectedRow(ctx, 1)
	if table.NeedsRedraw() {
		t.Fatal("table should stay clean when selection moves internally")
	}
	if !row0.NeedsRedraw() {
		t.Fatal("previous selected row should be locally dirty")
	}
	if !row1.NeedsRedraw() {
		t.Fatal("new selected row should be locally dirty")
	}
	if len(invalidated) != 2 {
		t.Fatalf("invalidated rect count = %d, want 2", len(invalidated))
	}
}

func TestTableViewInternalSelectionDoesNotDirtyScheduler(t *testing.T) {
	selected := state.NewSignal[int](-1)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
		TableViewSelectedRow(selected),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewSimpleCell{Text: "Row"}
		}),
	)
	sched := state.NewScheduler(func([]widget.Widget) {})
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	var invalidated []geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) {
		invalidated = append(invalidated, r)
	})
	widget.MountTree(table, ctx)
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 40)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 40))
	widget.StampScreenOrigin(table, &tableViewHeaderCanvas{})
	table.Draw(ctx, &tableViewHeaderCanvas{})
	widget.ClearRedrawInTree(table)
	sched.Flush()

	consumed := table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		event.ButtonStateLeft,
		geometry.Pt(8, 8),
		geometry.Pt(8, 8),
		0,
	))

	if !consumed {
		t.Fatal("left click should be consumed")
	}
	if selected.Get() != 0 {
		t.Fatalf("selected row = %d, want 0", selected.Get())
	}
	if sched.PendingCount() != 0 {
		t.Fatalf("pending dirty widgets = %d, want 0", sched.PendingCount())
	}
	if len(invalidated) != 1 {
		t.Fatalf("invalidated rect count = %d, want 1", len(invalidated))
	}
	widget.UnmountTree(table)
}

func TestTableViewExternalSelectionSignalMarksSchedulerDirty(t *testing.T) {
	selected := state.NewSignal[int](-1)
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
		TableViewSelectedRow(selected),
	)
	var dirty []widget.Widget
	sched := state.NewScheduler(func(widgets []widget.Widget) {
		dirty = widgets
	})
	ctx := widget.NewContext()
	ctx.SetScheduler(sched)
	widget.MountTree(table, ctx)
	sched.Flush()
	dirty = nil

	selected.Set(0)

	if sched.PendingCount() != 1 {
		t.Fatalf("pending dirty widgets = %d, want 1", sched.PendingCount())
	}
	sched.Flush()
	if len(dirty) != 1 || dirty[0] != table {
		t.Fatalf("dirty widgets = %v, want table only", dirty)
	}
	widget.UnmountTree(table)
}

func TestTableViewBuildSimpleCellHoverInvalidatesRowRect(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(2),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
		TableViewBuildSimpleCell(func(TableViewCellContext) TableViewSimpleCell {
			return TableViewSimpleCell{Text: "Row"}
		}),
	)
	ctx := widget.NewContext()
	var invalidated []geometry.Rect
	ctx.SetOnInvalidateRect(func(r geometry.Rect) {
		invalidated = append(invalidated, r)
	})
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 60)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 60))
	widget.StampScreenOrigin(table, &tableViewHeaderCanvas{})
	table.Draw(ctx, &tableViewHeaderCanvas{})
	widget.ClearRedrawInTree(table)

	table.setHoveredRow(ctx, 0)

	if table.body.NeedsRedraw() {
		t.Fatal("simple table body should stay clean after hover change")
	}
	row := table.body.simpleRows[0]
	if row == nil {
		t.Fatal("simple row was not cached")
	}
	if !row.NeedsRedraw() {
		t.Fatal("simple hovered row should be dirty after hover change")
	}
	if len(invalidated) != 1 {
		t.Fatalf("invalidated rect count = %d, want 1", len(invalidated))
	}
	if invalidated[0].Height() != table.cfg.rowHeight {
		t.Fatalf("invalidated rect height = %.1f, want %.1f", invalidated[0].Height(), table.cfg.rowHeight)
	}
}

func TestTableViewBodySkipsBackgroundForRowDirtyClip(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 80}}),
		TableViewRowCount(2),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
	)
	table.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(80, 80)))
	table.setContentWidth(80)
	bounds := geometry.NewRect(0, 0, 80, 80)
	canvas := &tableViewHeaderCanvas{clip: geometry.NewRect(12, 34, 80, 20)}
	canvas.PushTransform(geometry.Pt(12, 34))

	if table.body.shouldDrawBackground(canvas, bounds) {
		t.Fatal("body background should be skipped when dirty clip is covered by row backgrounds")
	}

	canvas.clip = geometry.NewRect(12, 78, 80, 12)
	if !table.body.shouldDrawBackground(canvas, bounds) {
		t.Fatal("body background should draw when dirty clip reaches below the last row")
	}
}

type tableViewLayoutProbe struct {
	widget.WidgetBase
	layouts int
}

func (p *tableViewLayoutProbe) Layout(widget.Context, geometry.Constraints) geometry.Size {
	p.layouts++
	return geometry.Sz(20, 20)
}

func (p *tableViewLayoutProbe) Draw(widget.Context, widget.Canvas) {}

func (p *tableViewLayoutProbe) Event(widget.Context, event.Event) bool {
	return false
}

func (p *tableViewLayoutProbe) Children() []widget.Widget {
	return nil
}

var _ widget.Widget = (*tableViewLayoutProbe)(nil)

type tableViewProbe struct {
	widget.WidgetBase
	pressed bool
}

func (p *tableViewProbe) Layout(widget.Context, geometry.Constraints) geometry.Size {
	return geometry.Sz(20, 20)
}

func (p *tableViewProbe) Draw(widget.Context, widget.Canvas) {}

func (p *tableViewProbe) Event(_ widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if ok && mouse.MouseType == event.MousePress {
		p.pressed = true
		return true
	}
	return false
}

func (p *tableViewProbe) Children() []widget.Widget {
	return nil
}

var _ widget.Widget = (*tableViewProbe)(nil)

type tableViewFixedProbe struct {
	widget.WidgetBase
	size geometry.Size
}

func (p *tableViewFixedProbe) Layout(widget.Context, geometry.Constraints) geometry.Size {
	p.SetBounds(geometry.FromPointSize(p.Position(), p.size))
	return p.size
}

func (p *tableViewFixedProbe) Draw(widget.Context, widget.Canvas) {}

func (p *tableViewFixedProbe) Event(widget.Context, event.Event) bool {
	return false
}

func (p *tableViewFixedProbe) Children() []widget.Widget {
	return nil
}

var _ widget.Widget = (*tableViewFixedProbe)(nil)

type tableViewHoverProbe struct {
	widget.WidgetBase
	moves   int
	pressed bool
}

func (p *tableViewHoverProbe) Layout(widget.Context, geometry.Constraints) geometry.Size {
	return geometry.Sz(20, 20)
}

func (p *tableViewHoverProbe) Draw(widget.Context, widget.Canvas) {}

func (p *tableViewHoverProbe) Event(_ widget.Context, e event.Event) bool {
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseMove:
		p.moves++
	case event.MousePress:
		p.pressed = true
		return true
	}
	return false
}

func (p *tableViewHoverProbe) Children() []widget.Widget {
	return nil
}

var _ widget.Widget = (*tableViewHoverProbe)(nil)

func TestTableViewAllowsEmptyCellBuilder(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "blank", Width: 60}}),
		TableViewRowCount(1),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return primitives.Box()
		}),
	)
	table.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(80, 60)))
}

func TestTableViewMountsAndUnmountsLazyCellWidgets(t *testing.T) {
	probe := &tableViewLifecycleProbe{}
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "field", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewBuildCell(func(TableViewCellContext) widget.Widget {
			return probe
		}),
	)
	ctx := widget.NewContext()

	widget.MountTree(table, ctx)
	table.Layout(ctx, geometry.Tight(geometry.Sz(80, 60)))
	table.body.layoutRow(ctx, 0)

	if probe.mounts != 1 || !probe.IsMounted() {
		t.Fatalf("lazy cell mounts = %d mounted=%v, want mounts=1 mounted=true", probe.mounts, probe.IsMounted())
	}

	table.body.clearRows()

	if probe.unmounts != 1 || probe.IsMounted() {
		t.Fatalf("lazy cell unmounts = %d mounted=%v, want unmounts=1 mounted=false", probe.unmounts, probe.IsMounted())
	}
}

func TestTableViewMountSubscribesScrollSignalOnce(t *testing.T) {
	scrollY := &tableViewCountingSignal{}
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "field", Width: 60}}),
		TableViewRowCount(1),
		TableViewScrollYSignal(scrollY),
	)
	ctx := widget.NewContext()
	ctx.SetScheduler(tableViewSchedulerProbe{})

	widget.MountTree(table, ctx)

	if got, want := scrollY.subscriptions, 1; got != want {
		t.Fatalf("scroll signal subscriptions = %d, want %d", got, want)
	}

	widget.UnmountTree(table)

	if got, want := scrollY.unsubscriptions, 1; got != want {
		t.Fatalf("scroll signal unsubscriptions = %d, want %d", got, want)
	}
}

type tableViewLifecycleProbe struct {
	widget.WidgetBase
	mounts   int
	unmounts int
}

func (p *tableViewLifecycleProbe) Layout(widget.Context, geometry.Constraints) geometry.Size {
	return geometry.Sz(20, 20)
}

func (p *tableViewLifecycleProbe) Draw(widget.Context, widget.Canvas) {}

func (p *tableViewLifecycleProbe) Event(widget.Context, event.Event) bool {
	return false
}

func (p *tableViewLifecycleProbe) Children() []widget.Widget {
	return nil
}

func (p *tableViewLifecycleProbe) Mount(widget.Context) {
	p.mounts++
}

func (p *tableViewLifecycleProbe) Unmount() {
	p.unmounts++
}

var (
	_ widget.Widget    = (*tableViewLifecycleProbe)(nil)
	_ widget.Lifecycle = (*tableViewLifecycleProbe)(nil)
)

type tableViewSchedulerProbe struct{}

func (tableViewSchedulerProbe) MarkDirty(widget.Widget) {}

type tableViewCountingSignal struct {
	value           float32
	subscriptions   int
	unsubscriptions int
	callbacks       []func(float32)
}

func (s *tableViewCountingSignal) Get() float32 {
	return s.value
}

func (s *tableViewCountingSignal) Set(value float32) {
	s.value = value
	var callbacks []func(float32)
	callbacks = append(callbacks, s.callbacks...)
	for _, callback := range callbacks {
		if callback != nil {
			callback(value)
		}
	}
}

func (s *tableViewCountingSignal) Update(fn func(float32) float32) {
	s.Set(fn(s.value))
}

func (s *tableViewCountingSignal) AsReadonly() state.ReadonlySignal[float32] {
	return s
}

func (s *tableViewCountingSignal) Subscribe(_ context.Context, fn func(float32)) state.Unsubscribe {
	return s.SubscribeForever(fn)
}

func (s *tableViewCountingSignal) SubscribeForever(fn func(float32)) state.Unsubscribe {
	index := len(s.callbacks)
	s.callbacks = append(s.callbacks, fn)
	s.subscriptions++
	active := true
	return func() {
		if !active {
			return
		}
		active = false
		s.callbacks[index] = nil
		s.unsubscriptions++
	}
}

func TestTableViewRowEventCapturesDragAfterPress(t *testing.T) {
	var got []event.MouseEventType
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewOnRowEvent(func(row int, e event.Event) bool {
			mouse, ok := e.(*event.MouseEvent)
			if !ok || row != 0 {
				return false
			}
			got = append(got, mouse.MouseType)
			return true
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 80)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 80))

	table.Event(ctx, event.NewMouseEvent(
		event.MousePress,
		event.ButtonLeft,
		event.ButtonStateLeft,
		geometry.Pt(8, 32),
		geometry.Pt(8, 32),
		0,
	))
	table.Event(ctx, event.NewMouseEvent(
		event.MouseDrag,
		event.ButtonLeft,
		event.ButtonStateLeft,
		geometry.Pt(8, 70),
		geometry.Pt(8, 70),
		0,
	))
	table.Event(ctx, event.NewMouseEvent(
		event.MouseRelease,
		event.ButtonLeft,
		0,
		geometry.Pt(8, 70),
		geometry.Pt(8, 70),
		0,
	))

	want := []event.MouseEventType{event.MousePress, event.MouseDrag, event.MouseRelease}
	if len(got) != len(want) {
		t.Fatalf("row events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row events = %v, want %v", got, want)
		}
	}
}

func TestTableViewRowEventWithContextCanSetCursor(t *testing.T) {
	table := TableView(
		TableViewColumns([]TableViewColumn{{Key: "name", Width: 60}}),
		TableViewRowCount(1),
		TableViewRowHeight(20),
		TableViewShowHeader(false),
		TableViewDispatchHoverToCells(false),
		TableViewOnRowEventWithContext(func(ctx widget.Context, row int, e event.Event) bool {
			if row == 0 {
				ctx.SetCursor(widget.CursorPointer)
			}
			return false
		}),
	)
	ctx := widget.NewContext()
	table.Layout(ctx, geometry.Tight(geometry.Sz(100, 80)))
	table.SetBounds(geometry.NewRect(0, 0, 100, 80))

	table.Event(ctx, event.NewMouseEvent(
		event.MouseMove,
		0,
		0,
		geometry.Pt(8, 8),
		geometry.Pt(8, 8),
		0,
	))

	if ctx.Cursor() != widget.CursorPointer {
		t.Fatalf("cursor = %v, want %v", ctx.Cursor(), widget.CursorPointer)
	}
}
