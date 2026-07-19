package rotheme

import (
	"math"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"
)

const TableHeaderPadY = float32(4)

type TableViewColumn struct {
	Key      string
	Title    string
	Width    float32
	Flex     float32
	MinWidth float32
	Align    widget.TextAlign
}

type TableViewCellContext struct {
	Row         int
	ColumnIndex int
	Column      TableViewColumn
	Width       float32
	Height      float32
}

type TableViewOption func(*tableViewConfig)

type tableViewConfig struct {
	columns         []TableViewColumn
	rowCount        int
	rowHeight       float32
	headerHeight    float32
	showHeader      bool
	emptyText       string
	scrollY         state.Signal[float32]
	selectedRow     state.Signal[int]
	buildCell       func(TableViewCellContext) widget.Widget
	buildSimpleCell func(TableViewCellContext) TableViewSimpleCell
	onRowClick      func(int)
	onRowEvent      func(int, event.Event) bool
	onRowEventCtx   func(widget.Context, int, event.Event) bool

	invalidateHover      bool
	dispatchHoverToCells bool
}

type TableViewWidget struct {
	widget.WidgetBase

	cfg tableViewConfig

	scroll *scrollview.Widget
	body   *tableViewBody

	width       float32
	height      float32
	bodyHeight  float32
	colWidths   []float32
	contentW    float32
	hoveredRow  int
	pressedRow  int
	bodyVisible []widget.Widget
	mountCtx    widget.Context
}

func TableView(opts ...TableViewOption) *TableViewWidget {
	cfg := tableViewConfig{
		rowHeight:            32,
		headerHeight:         Default.Typography.TextSize + TableHeaderPadY*2,
		showHeader:           true,
		emptyText:            "No rows",
		invalidateHover:      true,
		dispatchHoverToCells: true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	table := &TableViewWidget{
		cfg:        cfg,
		hoveredRow: -1,
		pressedRow: -1,
	}
	table.SetVisible(true)
	table.SetEnabled(true)

	table.body = &tableViewBody{table: table, rows: make(map[int]*tableViewRowWidget)}
	table.body.SetVisible(true)
	table.body.SetEnabled(true)

	optsScroll := []scrollview.Option{
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
		scrollview.ScrollStep(cfg.rowHeight),
	}
	if cfg.scrollY != nil {
		optsScroll = append(optsScroll, scrollview.ScrollYSignal(cfg.scrollY))
	}
	table.scroll = scrollview.New(table.body, optsScroll...)
	table.scroll.SetParent(table)

	return table
}

func TableViewColumns(columns []TableViewColumn) TableViewOption {
	return func(c *tableViewConfig) { c.columns = append([]TableViewColumn(nil), columns...) }
}

func TableViewRowCount(count int) TableViewOption {
	return func(c *tableViewConfig) { c.rowCount = count }
}

func TableViewRowHeight(height float32) TableViewOption {
	return func(c *tableViewConfig) { c.rowHeight = height }
}

func TableViewHeaderHeight(height float32) TableViewOption {
	return func(c *tableViewConfig) { c.headerHeight = height }
}

func TableViewShowHeader(show bool) TableViewOption {
	return func(c *tableViewConfig) { c.showHeader = show }
}

func TableViewEmptyText(text string) TableViewOption {
	return func(c *tableViewConfig) { c.emptyText = text }
}

func TableViewScrollYSignal(scrollY state.Signal[float32]) TableViewOption {
	return func(c *tableViewConfig) { c.scrollY = scrollY }
}

func TableViewSelectedRow(selectedRow state.Signal[int]) TableViewOption {
	return func(c *tableViewConfig) { c.selectedRow = selectedRow }
}

func TableViewBuildCell(build func(TableViewCellContext) widget.Widget) TableViewOption {
	return func(c *tableViewConfig) { c.buildCell = build }
}

func TableViewBuildSimpleCell(build func(TableViewCellContext) TableViewSimpleCell) TableViewOption {
	return func(c *tableViewConfig) { c.buildSimpleCell = build }
}

func TableViewOnRowClick(onClick func(int)) TableViewOption {
	return func(c *tableViewConfig) { c.onRowClick = onClick }
}

func TableViewOnRowEvent(onEvent func(int, event.Event) bool) TableViewOption {
	return func(c *tableViewConfig) { c.onRowEvent = onEvent }
}

func TableViewOnRowEventWithContext(onEvent func(widget.Context, int, event.Event) bool) TableViewOption {
	return func(c *tableViewConfig) { c.onRowEventCtx = onEvent }
}

func TableViewInvalidateHover(invalidate bool) TableViewOption {
	return func(c *tableViewConfig) { c.invalidateHover = invalidate }
}

func TableViewDispatchHoverToCells(dispatch bool) TableViewOption {
	return func(c *tableViewConfig) { c.dispatchHoverToCells = dispatch }
}

func (w *TableViewWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Biggest()
	if size.Width >= geometry.Infinity {
		size.Width = constraints.Constrain(geometry.Sz(240, 0)).Width
	}
	if size.Height >= geometry.Infinity {
		size.Height = w.contentHeight()
	}
	if size.Width <= 0 {
		size.Width = 240
	}
	if size.Height <= 0 {
		size.Height = w.contentHeight()
	}

	w.width = size.Width
	w.height = size.Height
	w.bodyHeight = size.Height - w.headerHeight()
	if w.bodyHeight < 0 {
		w.bodyHeight = 0
	}

	w.scroll.Layout(ctx, geometry.Tight(geometry.Sz(size.Width, w.bodyHeight)))
	w.setContentWidth(w.safeBodyWidth())
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *TableViewWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() {
		return
	}
	bounds := w.Bounds()
	if bounds.IsEmpty() {
		return
	}

	headerH := w.headerHeight()
	if headerH > 0 {
		w.drawHeader(canvas, geometry.NewRect(bounds.Min.X, bounds.Min.Y, bounds.Width(), headerH), w.safeBodyWidth())
	}

	scrollBounds := geometry.NewRect(bounds.Min.X, bounds.Min.Y+headerH, bounds.Width(), w.bodyHeight)
	w.scroll.SetBounds(scrollBounds)
	widget.StampScreenOrigin(w.scroll, canvas)
	w.scroll.Draw(ctx, canvas)
}

func (w *TableViewWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	w.updateScrollBounds()
	w.updateHoverFromEvent(ctx, e)
	return w.scroll.Event(ctx, e)
}

func (w *TableViewWidget) IsFocusable() bool {
	return w.IsVisible() && w.IsEnabled()
}

func (w *TableViewWidget) Children() []widget.Widget {
	if w.scroll == nil {
		return nil
	}
	return []widget.Widget{w.scroll}
}

func (w *TableViewWidget) Mount(ctx widget.Context) {
	w.mountCtx = ctx
	if w.body != nil {
		w.body.mountRows(ctx)
	}
}

func (w *TableViewWidget) Unmount() {
	if w.body != nil {
		w.body.clearRows()
	}
	w.mountCtx = nil
}

func (w *TableViewWidget) updateScrollBounds() {
	bounds := w.Bounds()
	if bounds.IsEmpty() || w.scroll == nil {
		return
	}
	headerH := w.headerHeight()
	w.scroll.SetBounds(geometry.NewRect(bounds.Min.X, bounds.Min.Y+headerH, bounds.Width(), w.bodyHeight))
}

func (w *TableViewWidget) updateHoverFromEvent(ctx widget.Context, e event.Event) {
	mouse, ok := e.(*event.MouseEvent)
	if !ok || w.scroll == nil {
		return
	}
	if mouse.MouseType == event.MouseLeave {
		w.setHoveredRow(ctx, -1)
		return
	}
	if mouse.MouseType != event.MouseMove && mouse.MouseType != event.MouseDrag {
		return
	}
	scrollBounds := w.scroll.Bounds()
	contentBounds := geometry.NewRect(scrollBounds.Min.X, scrollBounds.Min.Y, w.safeBodyWidth(), scrollBounds.Height())
	if scrollBounds.IsEmpty() || !contentBounds.Contains(mouse.Position) {
		w.setHoveredRow(ctx, -1)
	}
}

func (w *TableViewWidget) headerHeight() float32 {
	if w.cfg.showHeader {
		return w.cfg.headerHeight
	}
	return 0
}

func (w *TableViewWidget) contentHeight() float32 {
	return w.headerHeight() + float32(w.cfg.rowCount)*w.cfg.rowHeight
}

func (w *TableViewWidget) safeBodyWidth() float32 {
	width := w.width - w.scroll.ScrollbarInset()
	if width < 0 {
		return 0
	}
	return width
}

func (w *TableViewWidget) setContentWidth(width float32) {
	if w.contentW == width {
		return
	}
	w.contentW = width
	w.colWidths = w.resolveColumns(width)
	w.body.clearRows()
}

func (w *TableViewWidget) resolveColumns(width float32) []float32 {
	widths := make([]float32, len(w.cfg.columns))
	var fixed float32
	var flex float32
	for i, col := range w.cfg.columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixed += col.Width
			continue
		}
		value := col.Flex
		if value <= 0 {
			value = 1
		}
		flex += value
	}
	remaining := width - fixed
	if remaining < 0 {
		remaining = 0
	}
	for i, col := range w.cfg.columns {
		if col.Width > 0 {
			continue
		}
		value := col.Flex
		if value <= 0 {
			value = 1
		}
		colWidth := float32(0)
		if flex > 0 {
			colWidth = remaining * value / flex
		}
		if col.MinWidth > 0 {
			colWidth = float32(math.Max(float64(colWidth), float64(col.MinWidth)))
		}
		widths[i] = colWidth
	}
	return widths
}

func (w *TableViewWidget) drawHeader(canvas widget.Canvas, bounds geometry.Rect, contentWidth float32) {
	if bounds.IsEmpty() {
		return
	}
	colors := tableViewColors()
	canvas.DrawRect(bounds, colors.Header)

	x := bounds.Min.X
	widths := w.resolveColumns(contentWidth)
	for i, col := range w.cfg.columns {
		if i >= len(widths) {
			break
		}
		cell := geometry.NewRect(x, bounds.Min.Y, widths[i], bounds.Height())
		if col.Title != "" {
			textWidth := cell.Width() - 2*TableCellPadX
			if textWidth < 0 {
				textWidth = 0
			}
			textHeight := cell.Height() - 2*TableHeaderPadY
			if textHeight < 0 {
				textHeight = 0
			}
			text := geometry.NewRect(cell.Min.X+TableCellPadX, cell.Min.Y+TableHeaderPadY, textWidth, textHeight)
			DrawText(canvas, col.Title, text, Default.Typography.TextSize, Default.Colors.MutedText, false, col.Align)
		}
		x += widths[i]
	}
	canvas.DrawLine(
		geometry.Pt(bounds.Min.X, bounds.Max.Y-1),
		geometry.Pt(bounds.Max.X, bounds.Max.Y-1),
		Default.Colors.FooterLine,
		1,
	)
}

func (w *TableViewWidget) selected(row int) bool {
	return w.cfg.selectedRow != nil && w.cfg.selectedRow.Get() == row
}

func (w *TableViewWidget) callRowEvent(ctx widget.Context, row int, e event.Event) bool {
	if w.cfg.onRowEventCtx != nil {
		return w.cfg.onRowEventCtx(ctx, row, e)
	}
	if w.cfg.onRowEvent != nil {
		return w.cfg.onRowEvent(row, e)
	}
	return false
}

func tableViewColors() struct {
	Body     widget.Color
	Header   widget.Color
	Row      widget.Color
	AltRow   widget.Color
	Hover    widget.Color
	Selected widget.Color
} {
	return struct {
		Body     widget.Color
		Header   widget.Color
		Row      widget.Color
		AltRow   widget.Color
		Hover    widget.Color
		Selected widget.Color
	}{
		Body:     Default.Colors.WindowBody,
		Header:   Default.Colors.PanelBody,
		Row:      widget.RGBA8(246, 249, 253, 255),
		AltRow:   Default.Colors.PanelBody,
		Hover:    Default.Colors.ButtonHover,
		Selected: Default.Colors.ButtonDown,
	}
}

type tableViewBody struct {
	widget.WidgetBase
	table *TableViewWidget
	rows  map[int]*tableViewRowWidget
}

func (b *tableViewBody) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	if b.table == nil {
		return geometry.Size{}
	}
	width := constraints.MaxWidth
	if width >= geometry.Infinity {
		width = constraints.MinWidth
	}
	if width < constraints.MinWidth {
		width = constraints.MinWidth
	}
	return geometry.Sz(width, float32(b.table.cfg.rowCount)*b.table.cfg.rowHeight)
}

func (b *tableViewBody) Draw(ctx widget.Context, canvas widget.Canvas) {
	if b.table == nil {
		return
	}
	table := b.table
	table.setContentWidth(table.safeBodyWidth())

	bounds := table.scroll.Bounds()
	canvas.DrawRect(geometry.NewRect(0, 0, bounds.Width(), bounds.Height()), tableViewColors().Body)

	if table.cfg.rowCount == 0 {
		DrawText(canvas, table.cfg.emptyText, geometry.NewRect(0, 0, table.contentW, bounds.Height()), Default.Typography.TextSize, Default.Colors.MutedText, false, widget.TextAlignCenter)
		b.clearRows()
		return
	}

	if table.cfg.buildSimpleCell != nil {
		if len(b.rows) > 0 {
			b.clearRows()
		}
		table.bodyVisible = nil
		b.drawSimpleRows(canvas)
		return
	}

	start, end := b.visibleRange()
	b.trimRows(start, end)
	table.bodyVisible = table.bodyVisible[:0]
	for row := start; row < end; row++ {
		child := b.layoutRow(ctx, row)
		if child == nil {
			continue
		}
		table.bodyVisible = append(table.bodyVisible, child)
		widget.StampScreenOrigin(child, canvas)
		widget.DrawChild(child, ctx, canvas)
	}
}

func (b *tableViewBody) Event(ctx widget.Context, e event.Event) bool {
	if b.table == nil {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	table := b.table
	table.setContentWidth(table.safeBodyWidth())
	hoverEvent := isTableViewHoverEvent(mouse)

	outsideContentX := mouse.Position.X < 0 || mouse.Position.X >= table.contentW
	row := -1
	if !outsideContentX {
		row = b.rowAt(mouse.Position.Y)
	}
	switch mouse.MouseType {
	case event.MouseMove, event.MouseEnter, event.MouseDrag:
		table.setHoveredRow(ctx, row)
	case event.MouseLeave:
		table.setHoveredRow(ctx, -1)
	}

	if table.pressedRow >= 0 && (mouse.MouseType == event.MouseRelease || mouse.MouseType == event.MouseDrag) {
		if table.cfg.buildSimpleCell == nil && b.dispatchRowEvent(ctx, table.pressedRow, mouse) {
			if mouse.MouseType == event.MouseRelease {
				table.pressedRow = -1
			}
			return true
		}
		if table.callRowEvent(ctx, table.pressedRow, mouse) {
			if mouse.MouseType == event.MouseRelease {
				table.pressedRow = -1
			}
			return true
		}
		if mouse.MouseType == event.MouseRelease {
			table.pressedRow = -1
		}
	}

	if outsideContentX {
		return false
	}
	if row < 0 {
		return false
	}
	if hoverEvent && !table.cfg.dispatchHoverToCells {
		return table.callRowEvent(ctx, row, mouse)
	}
	if table.cfg.buildSimpleCell == nil {
		consumed := b.dispatchRowEvent(ctx, row, mouse)
		if mouse.MouseType == event.MousePress && consumed {
			table.pressedRow = row
		}
		if consumed {
			return true
		}
	}

	if table.callRowEvent(ctx, row, mouse) {
		if mouse.MouseType == event.MousePress {
			table.pressedRow = row
		}
		return true
	}
	if mouse.MouseType == event.MousePress {
		if table.cfg.selectedRow != nil {
			table.cfg.selectedRow.Set(row)
		}
		if table.cfg.onRowClick != nil {
			table.cfg.onRowClick(row)
		}
		ctx.RequestFocus(table)
		return true
	}
	return false
}

func (b *tableViewBody) Children() []widget.Widget {
	if b.table == nil || b.table.cfg.buildSimpleCell != nil || len(b.table.bodyVisible) == 0 {
		return nil
	}
	children := make([]widget.Widget, len(b.table.bodyVisible))
	copy(children, b.table.bodyVisible)
	return children
}

func (b *tableViewBody) visibleRange() (int, int) {
	table := b.table
	_, scrollY := table.scroll.ScrollOffset()
	rowHeight := table.cfg.rowHeight
	if rowHeight <= 0 {
		return 0, 0
	}
	startY := scrollY
	if startY < 0 {
		startY = 0
	}
	endY := scrollY + table.bodyHeight
	if endY < 0 {
		endY = 0
	}
	start := int(startY / rowHeight)
	end := int(endY/rowHeight) + 1
	if start < 0 {
		start = 0
	}
	if end > table.cfg.rowCount {
		end = table.cfg.rowCount
	}
	if end < start {
		end = start
	}
	return start, end
}

func (b *tableViewBody) drawSimpleRows(canvas widget.Canvas) {
	table := b.table
	start, end := b.visibleRange()
	for row := start; row < end; row++ {
		y := float32(row) * table.cfg.rowHeight
		rowBounds := geometry.NewRect(0, y, table.contentW, table.cfg.rowHeight)
		canvas.DrawRect(rowBounds, table.rowBackground(row))

		x := float32(0)
		for i, col := range table.cfg.columns {
			if i >= len(table.colWidths) {
				break
			}
			width := table.colWidths[i]
			cell := table.buildSimpleCell(row, i, col, width)
			cell.draw(canvas, geometry.NewRect(x, rowBounds.Min.Y, width, rowBounds.Height()))
			x += width
		}
	}
}

func (b *tableViewBody) rowAt(y float32) int {
	table := b.table
	if y < 0 || table.cfg.rowHeight <= 0 {
		return -1
	}
	row := int(y / table.cfg.rowHeight)
	if row < 0 || row >= table.cfg.rowCount {
		return -1
	}
	return row
}

func (b *tableViewBody) dispatchRowEvent(ctx widget.Context, row int, e *event.MouseEvent) bool {
	child := b.layoutRow(ctx, row)
	if child == nil {
		return false
	}
	return child.Event(ctx, e)
}

func (b *tableViewBody) layoutRow(ctx widget.Context, row int) *tableViewRowWidget {
	child := b.ensureRow(row)
	if child == nil {
		return nil
	}
	table := b.table
	y := float32(row) * table.cfg.rowHeight
	size := child.ensureLayout(ctx, table.contentW, table.cfg.rowHeight)
	child.SetBounds(geometry.NewRect(0, y, size.Width, size.Height))
	return child
}

func (b *tableViewBody) ensureRow(row int) *tableViewRowWidget {
	if row < 0 || row >= b.table.cfg.rowCount {
		return nil
	}
	if child := b.rows[row]; child != nil {
		return child
	}
	if b.rows == nil {
		b.rows = make(map[int]*tableViewRowWidget)
	}
	child := &tableViewRowWidget{
		table: b.table,
		row:   row,
	}
	child.child = b.table.buildRow(row)
	child.SetVisible(true)
	child.SetEnabled(true)
	setParent(child.child, child)
	setParent(child, b)
	b.rows[row] = child
	if b.table.mountCtx != nil {
		widget.MountTree(child, b.table.mountCtx)
	}
	return child
}

func (b *tableViewBody) clearRows() {
	for _, child := range b.rows {
		unmountMounted(child)
	}
	b.rows = make(map[int]*tableViewRowWidget)
	b.table.bodyVisible = nil
}

func (b *tableViewBody) trimRows(start, end int) {
	for row := range b.rows {
		if row < start || row >= end {
			unmountMounted(b.rows[row])
			delete(b.rows, row)
		}
	}
}

func (b *tableViewBody) mountRows(ctx widget.Context) {
	for _, child := range b.rows {
		widget.MountTree(child, ctx)
	}
}

type tableViewRowWidget struct {
	widget.WidgetBase
	table        *TableViewWidget
	row          int
	child        widget.Widget
	layoutWidth  float32
	layoutHeight float32
	layoutValid  bool
}

func (r *tableViewRowWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(r.table.contentW, r.table.cfg.rowHeight))
	r.ensureLayout(ctx, size.Width, size.Height)
	r.SetBounds(geometry.FromPointSize(r.Position(), size))
	return size
}

func (r *tableViewRowWidget) ensureLayout(ctx widget.Context, width, height float32) geometry.Size {
	size := geometry.Sz(width, height)
	if r.layoutValid && r.layoutWidth == width && r.layoutHeight == height {
		return size
	}
	if r.child != nil {
		r.child.Layout(ctx, geometry.Tight(size))
		setBounds(r.child, geometry.NewRect(0, 0, size.Width, size.Height))
	}
	r.layoutWidth = width
	r.layoutHeight = height
	r.layoutValid = true
	return size
}

func (r *tableViewRowWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	bounds := r.Bounds()
	canvas.DrawRect(bounds, r.background())
	if r.child == nil {
		return
	}
	canvas.PushTransform(bounds.Min)
	widget.StampScreenOrigin(r.child, canvas)
	widget.DrawChild(r.child, ctx, canvas)
	canvas.PopTransform()
}

func (r *tableViewRowWidget) Event(ctx widget.Context, e event.Event) bool {
	if r.child == nil {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return r.child.Event(ctx, e)
	}
	if !r.Bounds().Contains(mouse.Position) && mouse.MouseType != event.MouseRelease && mouse.MouseType != event.MouseDrag {
		return false
	}
	local := *mouse
	local.Position = mouse.Position.Sub(r.Bounds().Min)
	return r.child.Event(ctx, &local)
}

func (r *tableViewRowWidget) Children() []widget.Widget {
	if r.child == nil {
		return nil
	}
	return []widget.Widget{r.child}
}

func (r *tableViewRowWidget) background() widget.Color {
	return r.table.rowBackground(r.row)
}

func (w *TableViewWidget) rowBackground(row int) widget.Color {
	colors := tableViewColors()
	if w.selected(row) {
		return colors.Selected
	}
	if row == w.hoveredRow {
		return colors.Hover
	}
	if row%2 == 1 {
		return colors.AltRow
	}
	return colors.Row
}

func (w *TableViewWidget) buildRow(row int) widget.Widget {
	children := make([]widget.Widget, 0, len(w.cfg.columns))
	widths := make([]float32, 0, len(w.cfg.columns))

	for i, col := range w.cfg.columns {
		width := float32(0)
		if i < len(w.colWidths) {
			width = w.colWidths[i]
		}
		child := w.buildCell(row, i, col, width)
		children = append(children, child)
		widths = append(widths, width)
	}

	cells := make([]widget.Widget, 0, len(w.cfg.columns))
	for i, child := range children {
		width := widths[i]
		cells = append(cells,
			newTableViewCellWidget(child, width, w.cfg.rowHeight),
		)
	}
	return primitives.HBox(cells...).
		Height(w.cfg.rowHeight).
		CrossAlign(primitives.CrossAxisStretch)
}

func (w *TableViewWidget) buildCell(row, columnIndex int, col TableViewColumn, width float32) widget.Widget {
	var child widget.Widget
	if w.cfg.buildCell != nil {
		child = w.cfg.buildCell(TableViewCellContext{
			Row:         row,
			ColumnIndex: columnIndex,
			Column:      col,
			Width:       width,
			Height:      w.cfg.rowHeight,
		})
	}
	if child == nil {
		child = primitives.Box()
	}
	return child
}

func (w *TableViewWidget) buildSimpleCell(row, columnIndex int, col TableViewColumn, width float32) tableViewSimpleCell {
	if w.cfg.buildSimpleCell == nil {
		return tableViewSimpleCell{}
	}
	return w.cfg.buildSimpleCell(TableViewCellContext{
		Row:         row,
		ColumnIndex: columnIndex,
		Column:      col,
		Width:       width,
		Height:      w.cfg.rowHeight,
	}).simpleCell()
}

type tableViewCellWidget struct {
	widget.WidgetBase
	child  widget.Widget
	width  float32
	height float32
}

func newTableViewCellWidget(child widget.Widget, width, height float32) *tableViewCellWidget {
	w := &tableViewCellWidget{
		child:  child,
		width:  width,
		height: height,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	setParent(child, w)
	return w
}

func (w *tableViewCellWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(w.width, w.height))
	if w.child != nil {
		childSize := w.child.Layout(ctx, geometry.Constraints{
			MinWidth:  0,
			MaxWidth:  size.Width,
			MinHeight: 0,
			MaxHeight: size.Height,
		})
		y := (size.Height - childSize.Height) / 2
		if y < 0 {
			y = 0
		}
		setBounds(w.child, geometry.NewRect(0, y, childSize.Width, childSize.Height))
	}
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *tableViewCellWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if w.child == nil {
		return
	}
	bounds := w.Bounds()
	canvas.PushTransform(bounds.Min)
	widget.StampScreenOrigin(w.child, canvas)
	widget.DrawChild(w.child, ctx, canvas)
	canvas.PopTransform()
}

func (w *tableViewCellWidget) Event(ctx widget.Context, e event.Event) bool {
	if w.child == nil {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return w.child.Event(ctx, e)
	}
	if !w.Bounds().Contains(mouse.Position) && mouse.MouseType != event.MouseRelease && mouse.MouseType != event.MouseDrag {
		return false
	}
	local := *mouse
	local.Position = mouse.Position.Sub(w.Bounds().Min)
	return w.child.Event(ctx, &local)
}

func (w *tableViewCellWidget) Children() []widget.Widget {
	if w.child == nil {
		return nil
	}
	return []widget.Widget{w.child}
}

func isTableViewHoverEvent(mouse *event.MouseEvent) bool {
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove, event.MouseDrag:
		return true
	default:
		return false
	}
}

func (w *TableViewWidget) setHoveredRow(ctx widget.Context, row int) {
	if w.hoveredRow == row {
		return
	}
	previous := w.hoveredRow
	w.hoveredRow = row
	if !w.cfg.invalidateHover {
		return
	}
	w.invalidateHoverRow(ctx, previous)
	w.invalidateHoverRow(ctx, row)
}

func (w *TableViewWidget) invalidateHoverRow(ctx widget.Context, row int) {
	if row < 0 || w.body == nil {
		return
	}
	if w.cfg.buildSimpleCell != nil {
		if w.scroll == nil {
			return
		}
		_, scrollY := w.scroll.ScrollOffset()
		viewport := w.scroll.ScreenBounds()
		if viewport.IsEmpty() {
			return
		}
		y := viewport.Min.Y + float32(row)*w.cfg.rowHeight - scrollY
		bounds := geometry.NewRect(viewport.Min.X, y, w.contentW, w.cfg.rowHeight).Intersection(viewport)
		if bounds.IsEmpty() {
			return
		}
		w.body.MarkRedrawLocal()
		ctx.InvalidateRect(bounds)
		return
	}
	child := w.body.rows[row]
	if child == nil {
		return
	}
	child.MarkRedrawLocal()

	bounds := child.ScreenBounds()
	if w.scroll != nil {
		if viewport := w.scroll.ScreenBounds(); !viewport.IsEmpty() {
			bounds = bounds.Intersection(viewport)
		}
	}
	if bounds.IsEmpty() {
		return
	}
	ctx.InvalidateRect(bounds)
}

func setBounds(w widget.Widget, bounds geometry.Rect) {
	if setter, ok := w.(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(bounds)
	}
}

func setParent(w widget.Widget, parent widget.Widget) {
	if setter, ok := w.(interface{ SetParent(widget.Widget) }); ok {
		setter.SetParent(parent)
	}
}

func unmountMounted(w widget.Widget) {
	if w == nil {
		return
	}
	if mounted, ok := w.(interface{ IsMounted() bool }); ok && !mounted.IsMounted() {
		return
	}
	widget.UnmountTree(w)
}

var (
	_ widget.Widget    = (*TableViewWidget)(nil)
	_ widget.Focusable = (*TableViewWidget)(nil)
	_ widget.Lifecycle = (*TableViewWidget)(nil)
)
