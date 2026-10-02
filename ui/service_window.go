package ui

import (
	"time"

	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	serviceWindowWidth   = 280
	serviceWindowRows    = 6
	serviceWindowRowH    = 22
	serviceWindowPadding = 10
	serviceWindowHeight  = ROWindowTitleHeight + serviceWindowPadding*2 + serviceWindowRows*serviceWindowRowH + ROWindowFooterHeight
	serviceDoubleClick   = 360 * time.Millisecond
)

type ServiceWindowCallbacks struct {
	OnSelect func(int)
	OnCancel func()
}

type ServiceWindowOptions struct {
	Title    string
	Selected int
}

type ServiceWindow struct {
	Window
	title        string
	services     []string
	selected     state.Signal[int]
	scrollY      state.Signal[float32]
	callbacks    ServiceWindowCallbacks
	lastClickAt  time.Time
	lastClickRow int
	list         *listview.Widget
}

func NewServiceWindow(ctx client.Context, services []string, options ServiceWindowOptions, callbacks ServiceWindowCallbacks) *ServiceWindow {
	selected := clampServiceIndex(options.Selected, len(services))
	title := options.Title
	if title == "" {
		title = "Service"
	}
	w := &ServiceWindow{
		title:        title,
		services:     append([]string(nil), services...),
		selected:     state.NewSignal(selected),
		scrollY:      state.NewSignal[float32](0),
		callbacks:    callbacks,
		lastClickRow: -1,
	}
	w.Window = NewWindow(serviceWindowWidth, serviceWindowHeight)
	w.CloseOnEsc = false
	x, y := serviceWindowPosition(ctx)
	w.OpenAt(x, y, w.widgetTree())
	if wc := windowWidgetContext(ctx); wc != nil {
		wc.RequestFocus(w.list)
	}
	return w
}

func (w *ServiceWindow) SetContext(ctx client.Context) {
	if w == nil {
		return
	}
	x, y := serviceWindowPosition(ctx)
	w.SetAutoPosition(x, y)
}

func (w *ServiceWindow) Update(ctx client.Context) bool {
	if w == nil {
		return false
	}
	if ctx.Input != nil && ctx.Input.JustPressed(input.KeyEnter) {
		w.confirm()
		return true
	}
	return w.Window.Update(ctx)
}

func (w *ServiceWindow) SelectedIndex() int {
	if w == nil || w.selected == nil {
		return -1
	}
	return clampServiceIndex(w.selected.Get(), len(w.services))
}

func (w *ServiceWindow) Control(action string) bool {
	if w == nil || !w.IsOpen() {
		return false
	}
	switch action {
	case "up", "down":
		if len(w.services) == 0 {
			return true
		}
		delta := 1
		if action == "up" {
			delta = -1
		}
		index := max(0, min(len(w.services)-1, w.SelectedIndex()+delta))
		w.selected.Set(index)
		w.list.ScrollToIndex(index)
	case "confirm":
		w.confirm()
	case "cancel":
		w.cancel()
	}
	return true
}

func (w *ServiceWindow) widgetTree() widget.Widget {
	return Win(
		Title(w.title),
		CloseButton(false),
		Size(serviceWindowWidth, serviceWindowHeight),
		Content(
			primitives.Box(w.serviceList()).
				Padding(serviceWindowPadding).
				CrossAlign(primitives.CrossAxisStretch),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabledFn("OK", func() bool {
				return w.SelectedIndex() < 0
			}, w.confirm),
			rotheme.Button("Cancel", w.cancel),
		),
	)
}

func (w *ServiceWindow) serviceList() widget.Widget {
	lv := listview.New(
		listview.ItemCount(len(w.services)),
		listview.FixedItemHeight(serviceWindowRowH),
		listview.ScrollYSignal(w.scrollY),
		listview.SelectionModeOpt(listview.SelectionSingle),
		listview.SelectedIndexSignal(w.selected),
		listview.OnItemClick(w.recordClick),
		listview.PainterOpt(rotheme.SelectListPainter{EmptyText: "No services."}),
		listview.BuildItem(func(item listview.ItemContext) widget.Widget {
			if item.Index < 0 || item.Index >= len(w.services) {
				return rotheme.SelectListRow("", false, serviceWindowRowH)
			}
			return rotheme.SelectListRow(trimRunes(w.services[item.Index], 36), true, serviceWindowRowH)
		}),
	)
	lv.SetFocused(true)
	w.list = lv
	return lv
}

func (w *ServiceWindow) recordClick(index int) {
	now := time.Now()
	if w.lastClickRow == index && now.Sub(w.lastClickAt) <= serviceDoubleClick {
		w.lastClickAt = time.Time{}
		w.lastClickRow = -1
		w.confirm()
		return
	}
	w.lastClickAt = now
	w.lastClickRow = index
}

func (w *ServiceWindow) confirm() {
	index := w.SelectedIndex()
	if index >= 0 && w.callbacks.OnSelect != nil {
		w.callbacks.OnSelect(index)
	}
}

func (w *ServiceWindow) cancel() {
	if w.callbacks.OnCancel != nil {
		w.callbacks.OnCancel()
	}
}

func serviceWindowPosition(ctx client.Context) (int, int) {
	width, height := ctx.ScreenSize()
	x := (width - serviceWindowWidth) / 2
	y := (height*2)/3 - serviceWindowHeight/2
	if y < 48 {
		y = (height - serviceWindowHeight) / 2
	}
	if x < 8 {
		x = 8
	}
	if y < 8 {
		y = 8
	}
	return x, y
}

func clampServiceIndex(index, count int) int {
	if count <= 0 {
		return -1
	}
	if index < 0 || index >= count {
		return 0
	}
	return index
}
