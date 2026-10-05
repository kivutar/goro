package ui

import (
	"fmt"
	"github.com/kivutar/goro/input"
	"image/color"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gogpu/ui/core/listview"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	npcDialogWidth       = 360
	npcDialogHeight      = 260
	npcDialogPad         = 12
	npcDialogButtonW     = 78
	npcDialogLineH       = 14
	npcDialogMaxMessages = 32
	npcMenuWidth         = 260
	npcMenuMinRows       = 4
	npcMenuMaxRows       = 5
	npcMenuRowH          = 20
	npcMenuPad           = 8
	npcMenuMinHeight     = ROWindowTitleHeight + npcMenuPad*2 + npcMenuMinRows*npcMenuRowH + ROWindowFooterHeight
	npcInputNumberWidth  = 174
	npcInputTextWidth    = 274
	npcInputHeight       = 52
	npcInputMaxLength    = 80
)

var (
	npcDialogTextColor  = TextColor
	npcDialogMutedColor = MutedTextColor
)

type npcDialogAction int

const (
	npcDialogActionNone npcDialogAction = iota
	npcDialogActionNext
	npcDialogActionClose
	npcDialogActionMenu
	npcDialogActionNumberInput
	npcDialogActionStringInput
)

type NPCDialog struct {
	open        bool
	hasDialog   bool
	npcID       uint32
	lines       []string
	options     []string
	action      npcDialogAction
	clearOnText bool
	status      string
	input       string
	inputField  *textfield.Widget
	menuRow     int
	menuScrollY state.Signal[float32]

	dialogWindow Window
	menuWindow   Window
	inputWindow  Window
	dirty        bool
	onClose      func()
}

type npcDialogTextRun struct {
	text  string
	color color.RGBA
}

type npcDialogTextSegment struct {
	text  string
	color color.RGBA
	x     float32
	width float32
}

func (d *NPCDialog) Apply(packet network.NPCDialog) {
	switch packet.Kind {
	case network.NPCDialogSay:
		if strings.TrimSpace(packet.Message) == "" && !d.open {
			return
		}
		if d.clearOnText {
			d.lines = d.lines[:0]
			d.clearOnText = false
		}
		d.open = true
		d.hasDialog = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionNone
		d.options = nil
		d.clearInput()
		if packet.Message != "" {
			d.lines = append(d.lines, packet.Message)
			if len(d.lines) > npcDialogMaxMessages {
				d.lines = append([]string(nil), d.lines[len(d.lines)-npcDialogMaxMessages:]...)
			}
		}
		d.dirty = true
	case network.NPCDialogNext:
		if !d.open && len(d.lines) == 0 {
			return
		}
		d.open = true
		d.hasDialog = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionNext
		d.options = nil
		d.clearInput()
		d.dirty = true
	case network.NPCDialogClose:
		if !d.hasDialog {
			d.Reset()
			return
		}
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionClose
		d.options = nil
		d.clearInput()
		d.dirty = true
	case network.NPCDialogMenu:
		d.open = true
		d.npcID = packet.NPCID
		d.action = npcDialogActionMenu
		d.options = append([]string(nil), packet.Options...)
		d.menuRow = initialNPCMenuRow(d.options)
		d.ensureMenuScrollSignal().Set(0)
		d.clearInput()
		d.dirty = true
	case network.NPCDialogClear:
		if !d.open || d.npcID == 0 || packet.NPCID == 0 || d.npcID == packet.NPCID {
			d.Reset()
		}
	case network.NPCDialogNumberInput:
		d.openInput(packet.NPCID, npcDialogActionNumberInput)
	case network.NPCDialogStringInput:
		d.openInput(packet.NPCID, npcDialogActionStringInput)
	}
}

func (d *NPCDialog) Reset() {
	wasOpen := d.open
	d.closeWindows()
	d.open = false
	d.hasDialog = false
	d.npcID = 0
	d.lines = nil
	d.options = nil
	d.action = npcDialogActionNone
	d.clearOnText = false
	d.status = ""
	d.menuRow = -1
	if d.menuScrollY != nil {
		d.menuScrollY.Set(0)
	}
	d.clearInput()
	d.dirty = true
	if wasOpen && d.onClose != nil {
		d.onClose()
	}
}

func (d *NPCDialog) SetCloseHandler(handler func()) {
	d.onClose = handler
}

func (d *NPCDialog) ResetPublished(ctx Context) {
	d.Reset()
	d.publish(ctx)
}

func (d *NPCDialog) IsOpen() bool {
	return d != nil && d.open
}

func (d *NPCDialog) Update(ctx Context) bool {
	if !d.open {
		if d.dialogWindow.published != nil || d.menuWindow.published != nil || d.inputWindow.published != nil {
			d.publish(ctx)
			return true
		}
		return false
	}
	if ctx.Input == nil {
		return false
	}
	if d.openWindows(ctx) {
		return true
	}
	if ctx.Input.JustPressed(input.KeyEscape) {
		return d.Control(ctx, "cancel")
	}
	if ctx.Input.JustPressed(input.KeyEnter) {
		return d.Control(ctx, "confirm")
	}

	consumed := false
	if d.action == npcDialogActionMenu && d.menuWindow.Update(ctx) {
		consumed = true
	}
	if d.isInputAction() && d.inputWindow.Update(ctx) {
		consumed = true
	}
	if d.dialogWindow.Update(ctx) {
		consumed = true
	}
	if consumed {
		d.publish(ctx)
		return true
	}
	return false
}

// Control shares dialog actions between keyboard and scripted controllers.
func (d *NPCDialog) Control(ctx Context, action string) bool {
	if !d.IsOpen() {
		return false
	}
	switch action {
	case "confirm":
		switch d.action {
		case npcDialogActionNext:
			d.next(ctx)
		case npcDialogActionClose:
			d.close(ctx)
		case npcDialogActionMenu:
			d.chooseSelected(ctx)
		case npcDialogActionNumberInput, npcDialogActionStringInput:
			d.submitInput(ctx)
		}
	case "cancel":
		// Like Escape in the original client, only cancel menus and Close.
		switch d.action {
		case npcDialogActionMenu:
			d.choose(ctx, 255)
		case npcDialogActionClose:
			d.close(ctx)
		}
	case "up", "down":
		if d.action == npcDialogActionMenu && len(d.options) != 0 {
			delta := 1
			if action == "up" {
				delta = -1
			}
			d.menuRow = max(0, min(len(d.options)-1, d.menuRow+delta))
			scroll := d.ensureMenuScrollSignal()
			top := float32(d.menuRow * npcMenuRowH)
			if top < scroll.Get() {
				scroll.Set(top)
			} else if bottom := top + npcMenuRowH; bottom > scroll.Get()+npcMenuMaxRows*npcMenuRowH {
				scroll.Set(bottom - npcMenuMaxRows*npcMenuRowH)
			}
			d.dirty = true
			d.refresh(ctx)
		}
	default:
		return false
	}
	d.publish(ctx)
	return true
}

func (d *NPCDialog) next(ctx Context) {
	if ctx.Network == nil {
		d.status = "not connected"
		d.dirty = true
		d.refresh(ctx)
		return
	}
	if err := ctx.Network.SendNPCNext(d.npcID); err != nil {
		d.status = err.Error()
		d.dirty = true
		d.refresh(ctx)
		return
	}
	d.action = npcDialogActionNone
	d.clearOnText = true
	d.status = ""
	d.dirty = true
	d.refresh(ctx)
}

func (d *NPCDialog) openInput(npcID uint32, action npcDialogAction) {
	d.open = true
	d.npcID = npcID
	d.action = action
	d.options = nil
	d.status = ""
	d.input = ""
	d.inputField = nil
	d.dirty = true
}

func (d *NPCDialog) submitInput(ctx Context) {
	text := strings.TrimSpace(d.input)
	if text == "" {
		return
	}
	if ctx.Network == nil {
		d.status = "not connected"
		d.dirty = true
		d.refresh(ctx)
		return
	}
	var err error
	switch d.action {
	case npcDialogActionNumberInput:
		value, parseErr := strconv.ParseInt(text, 10, 32)
		if parseErr != nil {
			value = 0
		}
		err = ctx.Network.SendNPCNumberInput(d.npcID, int32(value))
	case npcDialogActionStringInput:
		err = ctx.Network.SendNPCStringInput(d.npcID, text)
	default:
		return
	}
	if err != nil {
		d.status = err.Error()
		d.dirty = true
		d.refresh(ctx)
		return
	}
	d.action = npcDialogActionNone
	d.clearOnText = true
	d.status = ""
	d.clearInput()
	d.dirty = true
	d.refresh(ctx)
}

func (d *NPCDialog) clearInput() {
	d.input = ""
	d.inputField = nil
}

func (d *NPCDialog) close(ctx Context) {
	if ctx.Network != nil && d.npcID != 0 {
		if err := ctx.Network.SendNPCClose(d.npcID); err != nil {
			d.status = err.Error()
			d.dirty = true
			d.refresh(ctx)
			return
		}
	}
	d.Reset()
	d.publish(ctx)
}

func (d *NPCDialog) choose(ctx Context, choice int) {
	if ctx.Network == nil {
		d.status = "not connected"
		d.dirty = true
		d.refresh(ctx)
		return
	}
	cancel := choice < 1 || choice > 254
	if choice < 1 {
		choice = 255
	}
	if choice > 255 {
		choice = 255
	}
	if err := ctx.Network.SendNPCMenuChoice(d.npcID, uint8(choice)); err != nil {
		d.status = err.Error()
		d.dirty = true
		d.refresh(ctx)
		return
	}
	if cancel || choice == 255 {
		d.Reset()
		d.publish(ctx)
		return
	}
	d.action = npcDialogActionNone
	d.options = nil
	d.status = ""
	d.dirty = true
	d.refresh(ctx)
}

func (d *NPCDialog) chooseSelected(ctx Context) {
	if d.menuRow < 0 || d.menuRow >= len(d.options) {
		return
	}
	d.choose(ctx, d.menuRow+1)
}

func (d *NPCDialog) ensureWindows(ctx Context) {
	width, height := ctx.ScreenSize()
	x, y, w, h := npcDialogBounds(width, height)
	if d.dialogWindow.width == 0 {
		d.dialogWindow = NewWindow(w, h)
		d.dialogWindow.SetAutoPosition(x, y)
	} else {
		if d.dialogWindow.width != w || d.dialogWindow.height != h {
			d.dirty = true
		}
		d.dialogWindow.SetSize(w, h)
		if d.dialogWindow.SetAutoPosition(x, y) {
			d.dirty = true
		}
	}
	menuX, menuY, menuW, menuH := d.menuBounds(width, height, d.dialogWindow.x, d.dialogWindow.y, w, h)
	if d.menuWindow.width == 0 {
		d.menuWindow = NewWindow(menuW, menuH)
		d.menuWindow.SetAutoPosition(menuX, menuY)
	} else {
		if d.menuWindow.width != menuW || d.menuWindow.height != menuH {
			d.dirty = true
		}
		d.menuWindow.SetSize(menuW, menuH)
		if d.menuWindow.SetAutoPosition(menuX, menuY) {
			d.dirty = true
		}
	}
	inputX, inputY, inputW, inputH := d.inputBounds(width, height, d.dialogWindow.x, d.dialogWindow.y, w, h)
	if d.inputWindow.width == 0 {
		d.inputWindow = NewWindow(inputW, inputH)
		d.inputWindow.titleHeight = 0
		d.inputWindow.SetAutoPosition(inputX, inputY)
	} else {
		d.inputWindow.titleHeight = 0
		if d.inputWindow.width != inputW || d.inputWindow.height != inputH {
			d.dirty = true
		}
		d.inputWindow.SetSize(inputW, inputH)
		if d.inputWindow.SetAutoPosition(inputX, inputY) {
			d.dirty = true
		}
	}
}

func (d *NPCDialog) openWindows(ctx Context) bool {
	d.ensureWindows(ctx)
	changed := d.dirty
	// Menus and input prompts can be sent without a preceding text dialog.
	if d.hasDialog || d.status != "" {
		if !d.dialogWindow.IsOpen() {
			d.dialogWindow.OpenAt(d.dialogWindow.x, d.dialogWindow.y, d.dialogTree(ctx, d.dialogWindow.width, d.dialogWindow.height))
			changed = true
		} else if d.dirty {
			d.dialogWindow.SetContent(d.dialogTree(ctx, d.dialogWindow.width, d.dialogWindow.height))
		}
	} else if d.dialogWindow.IsOpen() {
		d.dialogWindow.Close()
		changed = true
	}
	if d.action == npcDialogActionMenu {
		if !d.menuWindow.IsOpen() {
			d.menuWindow.OpenAt(d.menuWindow.x, d.menuWindow.y, d.menuTree(ctx, d.menuWindow.width, d.menuWindow.height))
			changed = true
		} else if d.dirty {
			d.menuWindow.SetContent(d.menuTree(ctx, d.menuWindow.width, d.menuWindow.height))
		}
	} else if d.menuWindow.IsOpen() {
		d.menuWindow.Close()
		changed = true
	}
	if d.isInputAction() {
		if !d.inputWindow.IsOpen() {
			d.inputWindow.OpenAt(d.inputWindow.x, d.inputWindow.y, d.inputTree(ctx, d.inputWindow.width, d.inputWindow.height))
			changed = true
		} else if d.dirty {
			d.inputWindow.SetContent(d.inputTree(ctx, d.inputWindow.width, d.inputWindow.height))
		}
	} else if d.inputWindow.IsOpen() {
		d.inputWindow.Close()
		changed = true
	}
	d.dirty = false
	if changed {
		d.focusInput()
		d.publish(ctx)
	}
	return changed
}

func (d *NPCDialog) closeWindows() {
	if d.dialogWindow.IsOpen() {
		d.dialogWindow.Close()
	}
	if d.menuWindow.IsOpen() {
		d.menuWindow.Close()
	}
	if d.inputWindow.IsOpen() {
		d.inputWindow.Close()
	}
}

func (d *NPCDialog) refresh(ctx Context) {
	if !d.open {
		return
	}
	d.openWindows(ctx)
}

func (d *NPCDialog) publish(ctx Context) {
	if ctx.UIManager == nil {
		return
	}
	if !d.open {
		d.dialogWindow.Unpublish(ctx)
		d.menuWindow.Unpublish(ctx)
		d.inputWindow.Unpublish(ctx)
		return
	}
	d.dialogWindow.Publish(ctx)
	if d.action == npcDialogActionMenu && d.menuWindow.IsOpen() {
		d.menuWindow.Publish(ctx)
	} else {
		d.menuWindow.Unpublish(ctx)
	}
	if d.isInputAction() && d.inputWindow.IsOpen() {
		d.inputWindow.Publish(ctx)
	} else {
		d.inputWindow.Unpublish(ctx)
	}
}

func (d *NPCDialog) dialogTree(ctx Context, width, height int) widget.Widget {
	contentHeight := height - ROWindowTitleHeight
	var footer []widget.Widget
	if d.action == npcDialogActionNext || d.action == npcDialogActionClose {
		contentHeight -= ROWindowFooterHeight
		label := "Next"
		action := d.next
		if d.action == npcDialogActionClose {
			label = "Close"
			action = d.close
		}
		footer = []widget.Widget{
			primitives.Expanded(primitives.Box()),
			rotheme.Button(label, func() {
				action(ctx)
			}).Width(npcDialogButtonW),
		}
	}
	lines := d.dialogLineWidgets(width, contentHeight)
	if len(lines) == 0 {
		lines = append(lines, rotheme.Text(""))
	}
	if d.status != "" {
		lines = append(lines, rotheme.Text(trimRunes(d.status, 64)).Color(npcDialogWidgetColor(ErrorTextColor)))
	} else if d.action == npcDialogActionNone {
		lines = append(lines,
			primitives.Box(
				primitives.Expanded(primitives.Box()),
				rotheme.Text("Waiting...").Color(npcDialogWidgetColor(npcDialogMutedColor)),
			),
		)
	}
	options := []WindowOption{
		Title(d.title(ctx)),
		CloseButton(false),
		Size(float32(width), float32(height)),
		Content(
			primitives.Box(lines...).
				Padding(npcDialogPad).
				Gap(2),
		),
	}
	if footer != nil {
		options = append(options,
			Footer(footer...),
		)
	}
	return Win(options...)
}

func (d *NPCDialog) isInputAction() bool {
	return d.action == npcDialogActionNumberInput || d.action == npcDialogActionStringInput
}

func (d *NPCDialog) inputTree(ctx Context, width, height int) widget.Widget {
	label := ""
	if d.action == npcDialogActionNumberInput {
		label = "Input number"
	}
	children := make([]widget.Widget, 0, 2)
	if label != "" {
		children = append(children, rotheme.Text(label).Color(npcDialogWidgetColor(TextColor)))
	}
	children = append(children,
		primitives.HBox(
			primitives.Expanded(
				primitives.Box(d.inputWidget(ctx)).
					Height(22).
					CrossAlign(primitives.CrossAxisStretch),
			),
			rotheme.Button("OK", func() {
				d.submitInput(ctx)
			}).Width(42),
		).
			Gap(8).
			CrossAlign(primitives.CrossAxisCenter),
	)
	return primitives.Box(
		primitives.Box(children...).
			PaddingXY(10, 5).
			Gap(3).
			CrossAlign(primitives.CrossAxisStretch),
	).
		Width(float32(width)).
		Height(float32(height)).
		Background(widget.RGBA8(255, 255, 255, 255)).
		BorderStyle(1, widget.RGBA8(193, 198, 194, 255)).
		Rounded(5)
}

func (d *NPCDialog) inputWidget(ctx Context) *textfield.Widget {
	if d.inputField != nil {
		return d.inputField
	}
	inputType := textfield.TypeText
	if d.action == npcDialogActionNumberInput {
		inputType = textfield.TypeNumber
	}
	d.inputField = rotheme.TextField(
		d.input,
		inputType,
		func(value string) {
			d.input = value
		},
		func(string) {
			d.submitInput(ctx)
		},
		textfield.MaxLength(npcInputMaxLength),
	)
	d.focusInput()
	return d.inputField
}

func (d *NPCDialog) focusInput() {
	if d.inputField != nil && d.isInputAction() {
		d.inputField.SetFocused(true)
	}
}

func (d *NPCDialog) dialogLineWidgets(width, contentHeight int) []widget.Widget {
	lineMax := maxInt(12, (width-2*npcDialogPad)/7)
	maxLines := maxInt(1, (contentHeight-2*npcDialogPad)/npcDialogLineH)
	wrapped := wrapNPCDialogLines(d.lines, lineMax)
	if len(wrapped) > maxLines {
		wrapped = wrapped[len(wrapped)-maxLines:]
	}
	lines := make([]widget.Widget, 0, len(wrapped))
	for _, line := range wrapped {
		lines = append(lines, npcDialogTextLine(line))
	}
	return lines
}

func (d *NPCDialog) menuTree(ctx Context, width, height int) widget.Widget {
	return Win(
		Title("Choose"),
		CloseButton(false),
		Size(float32(width), float32(height)),
		Content(
			primitives.Box(d.menuList(ctx)).
				Padding(npcMenuPad),
		),
		Footer(
			primitives.Expanded(primitives.Box()),
			rotheme.ButtonDisabledFn("OK", func() bool {
				return d.menuRow < 0 || d.menuRow >= len(d.options)
			}, func() {
				d.chooseSelected(ctx)
			}),
			rotheme.Button("Cancel", func() {
				d.choose(ctx, 255)
			}),
		),
	)
}

func (d *NPCDialog) menuList(ctx Context) widget.Widget {
	lastClickRow := -1
	var lastClickAt time.Time
	lv := listview.New(
		listview.ItemCount(len(d.options)),
		listview.FixedItemHeight(npcMenuRowH),
		listview.ScrollYSignal(d.ensureMenuScrollSignal()),
		listview.SelectionModeOpt(listview.SelectionSingle),
		listview.SelectedIndex(d.menuRow),
		listview.OnSelectionChange(func(index int) {
			d.menuRow = index
		}),
		listview.OnItemClick(func(index int) {
			// Item clicks arrive before the list updates its selection.
			d.menuRow = index
			now := time.Now()
			if lastClickRow == index && now.Sub(lastClickAt) <= 360*time.Millisecond {
				lastClickRow = -1
				lastClickAt = time.Time{}
				d.chooseSelected(ctx)
				return
			}
			lastClickRow, lastClickAt = index, now
		}),
		listview.PainterOpt(rotheme.SelectListPainter{EmptyText: "No options."}),
		listview.BuildItem(func(item listview.ItemContext) widget.Widget {
			if item.Index < 0 || item.Index >= len(d.options) {
				return rotheme.SelectListRow("", true, npcMenuRowH)
			}
			label := fmt.Sprintf("%d. %s", item.Index+1, npcDialogRunsPlainText(npcDialogTextRuns(d.options[item.Index], npcDialogTextColor)))
			return rotheme.SelectListRow(trimRunes(label, 34), true, npcMenuRowH)
		}),
	)
	lv.SetFocused(true)
	return lv
}

func (d *NPCDialog) ensureMenuScrollSignal() state.Signal[float32] {
	if d.menuScrollY == nil {
		d.menuScrollY = state.NewSignal[float32](0)
	}
	return d.menuScrollY
}

func initialNPCMenuRow(options []string) int {
	if len(options) == 0 {
		return -1
	}
	return 0
}

func npcDialogTextLine(runs []npcDialogTextRun) widget.Widget {
	line := &npcDialogTextLineWidget{
		runs: append([]npcDialogTextRun(nil), runs...),
	}
	line.SetVisible(true)
	line.SetEnabled(true)
	return line
}

func npcDialogRunsPlainText(runs []npcDialogTextRun) string {
	var b strings.Builder
	for _, run := range runs {
		b.WriteString(run.text)
	}
	return b.String()
}

func npcDialogWidgetColor(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}

type npcDialogTextLineWidget struct {
	widget.WidgetBase
	runs []npcDialogTextRun
}

func (w *npcDialogTextLineWidget) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	width := npcDialogEstimatedTextWidth(npcDialogRunsPlainText(w.runs))
	size := constraints.Constrain(geometry.Sz(width, npcDialogLineH))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *npcDialogTextLineWidget) Draw(_ widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	if bounds.IsEmpty() {
		return
	}
	segments := npcDialogTextSegments(w.runs, func(text string) float32 {
		return rotheme.MeasureText(canvas, text, rotheme.Default.Typography.TextSize, false)
	})
	if len(segments) == 0 {
		return
	}
	canvas.PushClip(bounds)
	defer canvas.PopClip()
	for _, segment := range segments {
		if segment.width <= 0 {
			continue
		}
		textBounds := geometry.NewRect(bounds.Min.X+segment.x, bounds.Min.Y, segment.width+1, bounds.Height())
		rotheme.DrawText(canvas, segment.text, textBounds, rotheme.Default.Typography.TextSize, npcDialogWidgetColor(segment.color), false, widget.TextAlignLeft)
	}
}

func (w *npcDialogTextLineWidget) Event(_ widget.Context, _ event.Event) bool {
	return false
}

func npcDialogTextSegments(runs []npcDialogTextRun, measure func(string) float32) []npcDialogTextSegment {
	segments := make([]npcDialogTextSegment, 0, len(runs))
	x := float32(0)
	for _, run := range runs {
		if run.text == "" {
			continue
		}
		width := measure(run.text)
		if width < 0 {
			width = 0
		}
		segments = append(segments, npcDialogTextSegment{
			text:  run.text,
			color: run.color,
			x:     x,
			width: width,
		})
		x += width
	}
	return segments
}

func npcDialogEstimatedTextWidth(text string) float32 {
	return float32(utf8.RuneCountInString(text)) * rotheme.Default.Typography.TextSize * 0.6
}

func npcDialogBounds(width, height int) (int, int, int, int) {
	w := minInt(npcDialogWidth, maxInt(260, width-2*windowScreenMargin))
	h := minInt(npcDialogHeight, maxInt(130, height-2*windowScreenMargin))
	x := (width - w) / 2
	y := (height - h) / 2
	if x < windowScreenMargin {
		x = windowScreenMargin
	}
	if y < windowScreenMargin {
		y = windowScreenMargin
	}
	return x, y, w, h
}

func (d *NPCDialog) menuBounds(width, height, dialogX, dialogY, dialogW, dialogH int) (int, int, int, int) {
	w := minInt(npcMenuWidth, maxInt(220, width-2*windowScreenMargin))
	rows := maxInt(npcMenuMinRows, minInt(len(d.options), npcMenuMaxRows))
	h := maxInt(npcMenuMinHeight, ROWindowTitleHeight+npcMenuPad*2+rows*npcMenuRowH+ROWindowFooterHeight)
	x := dialogX + (dialogW-w)/2
	y := dialogY + dialogH + 8
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, width-w-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, height-h-windowScreenMargin))
	return x, y, w, h
}

func (d *NPCDialog) inputBounds(width, height, dialogX, dialogY, dialogW, dialogH int) (int, int, int, int) {
	w := npcInputTextWidth
	if d.action == npcDialogActionNumberInput {
		w = npcInputNumberWidth
	}
	w = minInt(w, maxInt(140, width-2*windowScreenMargin))
	h := npcInputHeight
	x := dialogX + (dialogW-w)/2
	y := dialogY + dialogH + 8
	x = clampWindowInt(x, windowScreenMargin, maxInt(windowScreenMargin, width-w-windowScreenMargin))
	y = clampWindowInt(y, windowScreenMargin, maxInt(windowScreenMargin, height-h-windowScreenMargin))
	return x, y, w, h
}

func (d *NPCDialog) title(ctx Context) string {
	name := ""
	if ctx.World != nil && d.npcID != 0 {
		if actor, ok := ctx.World.Actors[d.npcID]; ok {
			name = strings.TrimSpace(actor.Name)
		}
	}
	if name == "" {
		name = "NPC"
	}
	return name
}

func wrapNPCDialogLines(lines []string, maxRunes int) [][]npcDialogTextRun {
	if maxRunes < 8 {
		maxRunes = 8
	}
	var out [][]npcDialogTextRun
	for _, line := range lines {
		for _, split := range strings.Split(line, "\n") {
			out = append(out, wrapNPCDialogTextRuns(npcDialogTextRuns(split, npcDialogTextColor), maxRunes)...)
		}
	}
	return out
}

type npcDialogColoredRune struct {
	r     rune
	color color.RGBA
}

func npcDialogTextRuns(text string, base color.RGBA) []npcDialogTextRun {
	current := base
	var runs []npcDialogTextRun
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		runs = append(runs, npcDialogTextRun{text: b.String(), color: current})
		b.Reset()
	}
	for i := 0; i < len(text); {
		if c, ok := parseNPCDialogColorCode(text, i, base); ok {
			flush()
			current = c
			i += 7
			continue
		}
		r, size := utf8.DecodeRuneInString(text[i:])
		b.WriteRune(r)
		i += size
	}
	flush()
	return runs
}

func parseNPCDialogColorCode(text string, at int, base color.RGBA) (color.RGBA, bool) {
	if at+7 > len(text) || text[at] != '^' {
		return color.RGBA{}, false
	}
	var value [6]byte
	for i := 0; i < 6; i++ {
		c := text[at+1+i]
		if !isNPCDialogHex(c) {
			return color.RGBA{}, false
		}
		value[i] = c
	}
	if strings.EqualFold(string(value[:]), "000000") {
		return base, true
	}
	return color.RGBA{
		R: npcDialogHexByte(value[0], value[1]),
		G: npcDialogHexByte(value[2], value[3]),
		B: npcDialogHexByte(value[4], value[5]),
		A: 255,
	}, true
}

func isNPCDialogHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')
}

func npcDialogHexByte(hi, lo byte) uint8 {
	return npcDialogHexNibble(hi)<<4 | npcDialogHexNibble(lo)
}

func npcDialogHexNibble(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	default:
		return 0
	}
}

func wrapNPCDialogTextRuns(runs []npcDialogTextRun, maxRunes int) [][]npcDialogTextRun {
	chars := npcDialogRunsToRunes(runs)
	if len(chars) == 0 {
		return nil
	}
	var out [][]npcDialogTextRun
	for len(chars) > maxRunes {
		breakAt := maxRunes
		for i := maxRunes - 1; i > 0; i-- {
			if chars[i].r == ' ' || chars[i].r == '\t' {
				breakAt = i
				break
			}
		}
		out = append(out, npcDialogRunesToRuns(chars[:breakAt]))
		chars = chars[breakAt:]
		for len(chars) > 0 && (chars[0].r == ' ' || chars[0].r == '\t') {
			chars = chars[1:]
		}
	}
	if len(chars) > 0 {
		out = append(out, npcDialogRunesToRuns(chars))
	}
	return out
}

func npcDialogRunsToRunes(runs []npcDialogTextRun) []npcDialogColoredRune {
	var chars []npcDialogColoredRune
	for _, run := range runs {
		for _, r := range run.text {
			chars = append(chars, npcDialogColoredRune{r: r, color: run.color})
		}
	}
	return chars
}

func npcDialogRunesToRuns(chars []npcDialogColoredRune) []npcDialogTextRun {
	if len(chars) == 0 {
		return nil
	}
	runs := []npcDialogTextRun{{color: chars[0].color}}
	var b strings.Builder
	current := chars[0].color
	for _, char := range chars {
		if char.color != current {
			runs[len(runs)-1].text = b.String()
			b.Reset()
			current = char.color
			runs = append(runs, npcDialogTextRun{color: current})
		}
		b.WriteRune(char.r)
	}
	runs[len(runs)-1].text = b.String()
	return runs
}
