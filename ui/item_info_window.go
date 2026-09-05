package ui

import (
	"image"
	"image/color"
	"strconv"
	"strings"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	itemInfoWindowWidth       = 340
	itemInfoWindowMaxHeight   = 304
	itemInfoWindowPad         = 10
	itemInfoIllustrationWidth = 75
	itemInfoIllustrationH     = 100
	itemInfoSlotIcon          = 24
	itemInfoLineH             = 14
	itemInfoDescriptionW      = itemInfoWindowWidth - itemInfoWindowPad*2 - itemInfoIllustrationWidth - 12
	itemInfoDescriptionMaxH   = itemInfoWindowMaxHeight - ROWindowTitleHeight - itemInfoWindowPad*2
)

type ItemInfoWindow struct {
	Window
	item            session.InventoryItem
	title           string
	lines           []itemInfoTextLine
	illustration    image.Image
	bookAvailable   bool
	readBookRequest ItemInfoReadBookRequest
	cardArtRequest  ItemInfoCardIllustrationRequest
	tooltip         tooltipState
	slotIcons       map[string]image.Image
	slotIconMiss    map[string]struct{}
}

type ItemInfoReadBookRequest struct {
	ItemID uint16
	Title  string
}

type ItemInfoCardIllustrationRequest struct {
	ItemID uint16
	Title  string
}

func (w *ItemInfoWindow) openItem(ctx Context, item session.InventoryItem, mouseX, mouseY int) {
	if item.ItemID == 0 {
		return
	}
	w.EnsureWindow(itemInfoWindowWidth, itemInfoWindowMaxHeight)
	w.item = item
	w.title = itemInfoTitle(ctx, item)
	w.lines = itemInfoDescriptionLines(ctx, item)
	w.illustration = nil
	w.bookAvailable = itemInfoShowsReadBook(ctx, item)
	w.readBookRequest = ItemInfoReadBookRequest{}
	w.cardArtRequest = ItemInfoCardIllustrationRequest{}
	w.tooltip.Hide()

	height := w.windowHeight(ctx)
	w.SetSize(itemInfoWindowWidth, height)
	screenW, screenH := ctx.ScreenSize()
	x := clampWindowInt(mouseX+14, windowScreenMargin, maxInt(windowScreenMargin, screenW-itemInfoWindowWidth-windowScreenMargin))
	y := clampWindowInt(mouseY-22, windowScreenMargin, maxInt(windowScreenMargin, screenH-height-windowScreenMargin))
	w.OpenAt(x, y, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *ItemInfoWindow) Update(ctx Context, assets AssetProvider) bool {
	w.EnsureWindow(itemInfoWindowWidth, itemInfoWindowMaxHeight)
	if !w.IsOpen() {
		w.tooltip.Hide()
		return false
	}
	w.SetSize(itemInfoWindowWidth, w.windowHeight(ctx))
	if w.illustration == nil && assets != nil {
		w.illustration = assets.ItemInfoIllustrationImage(ctx.Resources, w.item, itemInfoIllustrationWidth, itemInfoIllustrationH)
		w.SetContent(w.widgetTree(ctx))
	}
	consumed := w.Window.Update(ctx)
	if !w.IsOpen() {
		w.tooltip.Hide()
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *ItemInfoWindow) Rebind(ctx Context, assets AssetProvider) {
	w.EnsureWindow(itemInfoWindowWidth, itemInfoWindowMaxHeight)
	if !w.IsOpen() {
		return
	}
	w.bookAvailable = itemInfoShowsReadBook(ctx, w.item)
	w.SetSize(itemInfoWindowWidth, w.windowHeight(ctx))
	if assets != nil {
		w.illustration = assets.ItemInfoIllustrationImage(ctx.Resources, w.item, itemInfoIllustrationWidth, itemInfoIllustrationH)
	}
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *ItemInfoWindow) widgetTree(ctx Context) widget.Widget {
	options := []WindowOption{
		Title(w.title),
		CloseButton(true),
		OnClose(func() {
			w.tooltip.Hide()
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(float32(itemInfoWindowWidth), float32(w.windowHeight(ctx))),
		Content(
			primitives.HBox(
				w.illustrationPanel(),
				w.infoPanel(ctx),
			).
				Padding(itemInfoWindowPad).
				Gap(12),
		),
	}
	if w.footerHeight(ctx) > 0 {
		options = append(options,
			Footer(w.footerWidgets(ctx)...),
		)
	}
	return Win(options...)
}

func (w *ItemInfoWindow) illustrationPanel() widget.Widget {
	return primitives.Box(
		newStaticImageWidget(w.illustration, itemInfoIllustrationWidth, itemInfoIllustrationH),
	).
		Height(itemInfoIllustrationH).
		Width(itemInfoIllustrationWidth).
		Background(rotheme.Default.Colors.PanelBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder)
}

func (w *ItemInfoWindow) infoPanel(ctx Context) widget.Widget {
	children := []widget.Widget{
		w.descriptionPanel(ctx),
	}
	return primitives.Box(children...).
		Width(itemInfoDescriptionW).
		Gap(8)
}

func itemInfoTitle(ctx Context, item session.InventoryItem) string {
	name := inventoryItemDisplayName(ctx.Resources, item)
	if item.Refine > 0 {
		name = "+" + strconv.Itoa(int(item.Refine)) + " " + name
	}
	return name
}

func (w *ItemInfoWindow) descriptionPanel(ctx Context) widget.Widget {
	lines := w.wrappedLines(itemInfoDescriptionRunes())
	if len(lines) == 0 {
		lines = []itemInfoTextLine{parseItemInfoTextLine("No description available.")}
	}
	textLines := make([]widget.Widget, 0, len(lines))
	for _, line := range lines {
		textLines = append(textLines, itemInfoTextLineWidget(line))
	}
	return primitives.Box(
		scrollview.New(
			primitives.Box(
				primitives.Box(textLines...).
					Gap(0),
			).
				PaddingRight(ROScrollbarGutter),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(itemInfoLineH),
		),
	).
		Height(float32(w.descriptionHeight(ctx)))
}

func itemInfoTextLineWidget(line itemInfoTextLine) widget.Widget {
	if len(line.Runs) == 0 {
		return rotheme.Text(" ").
			Color(itemInfoWidgetColor(inventoryTextColor)).
			LineHeight(itemInfoLineH / rotheme.Default.Typography.TextSize)
	}
	runs := make([]widget.Widget, 0, len(line.Runs))
	for _, run := range line.Runs {
		if run.Text == "" {
			continue
		}
		runs = append(runs,
			rotheme.Text(run.Text).
				Color(itemInfoWidgetColor(run.Color)).
				LineHeight(itemInfoLineH/rotheme.Default.Typography.TextSize),
		)
	}
	if len(runs) == 0 {
		return rotheme.Text(" ").
			Color(itemInfoWidgetColor(inventoryTextColor)).
			LineHeight(itemInfoLineH / rotheme.Default.Typography.TextSize)
	}
	return primitives.HBox(runs...).
		CrossAlign(primitives.CrossAxisStart).
		Height(itemInfoLineH)
}

func (w *ItemInfoWindow) cardSlotsFooter(ctx Context) []widget.Widget {
	slots := make([]widget.Widget, 0, 4)
	slotCount, _ := ctx.Resources.ItemSlotCount(int(w.item.ItemID))
	for i := 0; i < 4; i++ {
		cardID := w.cardSlotCardID(i)
		slots = append(slots,
			newItemInfoCardSlotWidget(
				w.cardSlotIcon(ctx, i, slotCount),
				itemInfoSlotIcon,
				itemInfoSlotIcon,
				cardID,
				func(cardID uint16) { w.showCardTooltip(ctx, cardID) },
				func() { w.tooltip.Hide() },
				func(cardID uint16, x, y int) { w.openCard(ctx, cardID, x, y) },
			),
		)
	}
	return slots
}

func (w *ItemInfoWindow) footerWidgets(ctx Context) []widget.Widget {
	children := make([]widget.Widget, 0, 6)
	if itemInfoShowsCardIllustration(ctx, w.item) {
		children = append(children, rotheme.Button("View", w.requestCardIllustration))
	}
	if w.bookAvailable {
		children = append(children, rotheme.Button("Read", func() {
			w.readBookRequest = ItemInfoReadBookRequest{ItemID: w.item.ItemID, Title: w.title}
		}))
	}
	if itemInfoShowsCardSlots(ctx, w.item) {
		children = append(children, w.cardSlotsFooter(ctx)...)
	}
	return children
}

func (w *ItemInfoWindow) requestCardIllustration() {
	w.cardArtRequest = ItemInfoCardIllustrationRequest{ItemID: w.item.ItemID, Title: w.title}
}

func (w *ItemInfoWindow) PopReadBookRequest() ItemInfoReadBookRequest {
	request := w.readBookRequest
	w.readBookRequest = ItemInfoReadBookRequest{}
	return request
}

func (w *ItemInfoWindow) PopCardIllustrationRequest() ItemInfoCardIllustrationRequest {
	request := w.cardArtRequest
	w.cardArtRequest = ItemInfoCardIllustrationRequest{}
	return request
}

func (w *ItemInfoWindow) DrawTooltip(ctx Context, screen *render.Frame) {
	w.tooltip.Draw(ctx, screen)
}

func (w *ItemInfoWindow) showCardTooltip(ctx Context, cardID uint16) {
	if cardID == 0 || ctx.Input == nil {
		w.tooltip.Hide()
		return
	}
	card := session.InventoryItem{ItemID: cardID, Type: db.ItemTypeCard, Identified: true}
	w.tooltip.Show(ctx, inventoryItemDisplayName(ctx.Resources, card), ctx.Input.MouseX, ctx.Input.MouseY+18, ctx.Input.MouseY-6)
}

func (w *ItemInfoWindow) openCard(ctx Context, cardID uint16, mouseX, mouseY int) {
	if cardID == 0 {
		return
	}
	w.openItem(ctx, session.InventoryItem{ItemID: cardID, Type: db.ItemTypeCard, Identified: true}, mouseX, mouseY)
}

func (w *ItemInfoWindow) cardSlotCardID(index int) uint16 {
	if index < 0 || index >= len(w.item.Cards) {
		return 0
	}
	cardID := w.item.Cards[index]
	if cardID == 0x00ff || cardID == 0x00fe || cardID == 0xff00 {
		return 0
	}
	return cardID
}

func (w *ItemInfoWindow) cardSlotIcon(ctx Context, index, slotCount int) image.Image {
	if cardID := w.cardSlotCardID(index); cardID != 0 {
		return w.loadItemIcon(ctx.Resources, cardID)
	}
	if index < slotCount {
		return w.loadInterfaceIcon(ctx.Resources, "empty_card_slot")
	}
	return w.loadInterfaceIcon(ctx.Resources, "basic_interface\\coparison_disable_card_slot", "basic_interface\\comparison_disable_card_slot", "coparison_disable_card_slot", "comparison_disable_card_slot")
}

func (w *ItemInfoWindow) loadInterfaceIcon(manager *res.Manager, resources ...string) image.Image {
	var candidates []string
	for _, resource := range resources {
		candidates = append(candidates, res.InterfaceTextureCandidates(resource)...)
	}
	return w.loadSlotIcon(manager, "interface:"+strings.Join(resources, "|"), candidates)
}

func (w *ItemInfoWindow) loadItemIcon(manager *res.Manager, itemID uint16) image.Image {
	if manager == nil || itemID == 0 {
		return nil
	}
	resourceName, ok := manager.ItemResourceName(int(itemID), true)
	if !ok {
		return nil
	}
	return w.loadSlotIcon(manager, "item:"+resourceName, res.ItemIconTextureCandidates(resourceName))
}

func (w *ItemInfoWindow) loadSlotIcon(manager *res.Manager, key string, candidates []string) image.Image {
	if manager == nil || key == "" {
		return nil
	}
	if w.slotIcons == nil {
		w.slotIcons = make(map[string]image.Image)
	}
	if w.slotIconMiss == nil {
		w.slotIconMiss = make(map[string]struct{})
	}
	if img := w.slotIcons[key]; img != nil {
		return img
	}
	if _, ok := w.slotIconMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, candidates)
	if err != nil {
		w.slotIconMiss[key] = struct{}{}
		return nil
	}
	w.slotIcons[key] = img
	return img
}

func (w *ItemInfoWindow) windowHeight(ctx Context) int {
	return ROWindowTitleHeight + itemInfoWindowPad*2 + w.bodyHeight(ctx) + w.footerHeight(ctx)
}

func (w *ItemInfoWindow) bodyHeight(ctx Context) int {
	lineCount := len(w.wrappedLines(itemInfoDescriptionRunes()))
	if lineCount == 0 {
		lineCount = 1
	}
	descriptionH := lineCount * itemInfoLineH
	maxDescriptionH := itemInfoDescriptionMaxH - w.footerHeight(ctx)
	descriptionH = clampWindowInt(descriptionH, itemInfoLineH, maxDescriptionH)
	return maxInt(itemInfoIllustrationH, descriptionH)
}

func (w *ItemInfoWindow) footerHeight(ctx Context) int {
	if w.bookAvailable || itemInfoShowsCardIllustration(ctx, w.item) || itemInfoShowsCardSlots(ctx, w.item) {
		return ROWindowFooterHeight
	}
	return 0
}

func itemInfoShowsCardIllustration(ctx Context, item session.InventoryItem) bool {
	if ctx.Resources == nil || item.ItemID == 0 || item.Type != db.ItemTypeCard {
		return false
	}
	_, ok := ctx.Resources.ItemCardIllustrationName(int(item.ItemID))
	return ok
}

func itemInfoShowsReadBook(ctx Context, item session.InventoryItem) bool {
	return ctx.Resources != nil && ctx.Resources.HasBook(item.ItemID)
}

func (w *ItemInfoWindow) descriptionHeight(ctx Context) int {
	return w.bodyHeight(ctx)
}

func (w *ItemInfoWindow) wrappedLines(maxRunes int) []itemInfoTextLine {
	return wrapItemInfoTextLines(w.lines, maxRunes)
}

func itemInfoDescriptionRunes() int {
	return maxInt(10, (itemInfoDescriptionW-18)/7)
}

func itemInfoShowsCardSlots(ctx Context, item session.InventoryItem) bool {
	if ctx.Resources == nil || !item.Identified || !inventoryItemCanShowCards(item) {
		return false
	}
	if item.Type == db.ItemTypeArmor && item.Location == 0 {
		return false
	}
	if !inventoryItemCanShowSlots(item) {
		return false
	}
	if slotCount, ok := ctx.Resources.ItemSlotCount(int(item.ItemID)); ok && slotCount > 0 {
		return true
	}
	for _, cardID := range item.Cards {
		if cardID != 0 {
			return true
		}
	}
	return false
}

func itemInfoDescriptionLines(ctx Context, item session.InventoryItem) []itemInfoTextLine {
	if ctx.Resources == nil {
		return nil
	}
	lines, ok := ctx.Resources.ItemDescription(int(item.ItemID), item.Identified)
	if !ok {
		return nil
	}
	out := make([]itemInfoTextLine, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(strings.ReplaceAll(line, "_", " "))
		if line == "" {
			out = append(out, itemInfoTextLine{})
			continue
		}
		out = append(out, parseItemInfoTextLine(line))
	}
	return out
}

type itemInfoTextLine struct {
	Runs []itemInfoTextRun
}

type itemInfoTextRun struct {
	Text  string
	Color color.RGBA
}

func parseItemInfoTextLine(text string) itemInfoTextLine {
	runes := []rune(text)
	line := itemInfoTextLine{}
	currentColor := inventoryTextColor
	current := make([]rune, 0, len(runes))
	flush := func() {
		if len(current) == 0 {
			return
		}
		line.Runs = append(line.Runs, itemInfoTextRun{Text: string(current), Color: currentColor})
		current = current[:0]
	}
	for i := 0; i < len(runes); i++ {
		if runes[i] == '^' && i+6 < len(runes) && isHexRunes(runes[i+1:i+7]) {
			flush()
			currentColor = parseItemInfoColor(runes[i+1 : i+7])
			i += 6
			continue
		}
		current = append(current, runes[i])
	}
	flush()
	return line
}

func parseItemInfoColor(runes []rune) color.RGBA {
	value, err := strconv.ParseUint(string(runes), 16, 32)
	if err != nil {
		return inventoryTextColor
	}
	return color.RGBA{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
		A: 255,
	}
}

func isHexRunes(runes []rune) bool {
	if len(runes) != 6 {
		return false
	}
	for _, r := range runes {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func wrapItemInfoTextLines(lines []itemInfoTextLine, maxRunes int) []itemInfoTextLine {
	var out []itemInfoTextLine
	for _, line := range lines {
		if strings.TrimSpace(itemInfoLinePlainText(line)) == "" {
			out = append(out, itemInfoTextLine{})
			continue
		}
		out = append(out, wrapItemInfoTextLine(line, maxRunes)...)
	}
	return out
}

func wrapItemInfoTextLine(line itemInfoTextLine, maxRunes int) []itemInfoTextLine {
	words := itemInfoLineWords(line)
	if len(words) == 0 {
		return []itemInfoTextLine{{}}
	}
	if maxRunes <= 0 {
		return []itemInfoTextLine{line}
	}
	var out []itemInfoTextLine
	current := itemInfoTextLine{}
	currentLen := 0
	for _, word := range words {
		wordLen := runeLen(word.Text)
		if wordLen > maxRunes {
			if currentLen > 0 {
				out = append(out, current)
				current = itemInfoTextLine{}
				currentLen = 0
			}
			chunks := splitItemInfoTextRun(word, maxRunes)
			out = append(out, chunks[:len(chunks)-1]...)
			current = chunks[len(chunks)-1]
			currentLen = runeLen(itemInfoLinePlainText(current))
			continue
		}
		if currentLen == 0 {
			current.Runs = append(current.Runs, word)
			currentLen = wordLen
			continue
		}
		if currentLen+1+wordLen <= maxRunes {
			appendItemInfoWord(&current, itemInfoTextRun{Text: " ", Color: word.Color})
			appendItemInfoWord(&current, word)
			currentLen += 1 + wordLen
			continue
		}
		out = append(out, current)
		current = itemInfoTextLine{Runs: []itemInfoTextRun{word}}
		currentLen = wordLen
	}
	if currentLen > 0 {
		out = append(out, current)
	}
	return out
}

func splitItemInfoTextRun(run itemInfoTextRun, maxRunes int) []itemInfoTextLine {
	runes := []rune(run.Text)
	if maxRunes <= 0 || len(runes) <= maxRunes {
		return []itemInfoTextLine{{Runs: []itemInfoTextRun{run}}}
	}
	lines := make([]itemInfoTextLine, 0, (len(runes)+maxRunes-1)/maxRunes)
	for len(runes) > 0 {
		count := minInt(maxRunes, len(runes))
		lines = append(lines, itemInfoTextLine{Runs: []itemInfoTextRun{{Text: string(runes[:count]), Color: run.Color}}})
		runes = runes[count:]
	}
	return lines
}

func itemInfoLineWords(line itemInfoTextLine) []itemInfoTextRun {
	var words []itemInfoTextRun
	for _, run := range line.Runs {
		for _, word := range strings.Fields(run.Text) {
			words = append(words, itemInfoTextRun{Text: word, Color: run.Color})
		}
	}
	return words
}

func appendItemInfoWord(line *itemInfoTextLine, run itemInfoTextRun) {
	last := len(line.Runs) - 1
	if last >= 0 && line.Runs[last].Color == run.Color {
		line.Runs[last].Text += run.Text
		return
	}
	line.Runs = append(line.Runs, run)
}

func itemInfoLinePlainText(line itemInfoTextLine) string {
	var b strings.Builder
	for _, run := range line.Runs {
		b.WriteString(run.Text)
	}
	return b.String()
}

func stripItemInfoColorCodes(text string) string {
	runes := []rune(text)
	out := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		if runes[i] == '^' && i+6 < len(runes) && isHexRunes(runes[i+1:i+7]) {
			i += 6
			continue
		}
		out = append(out, runes[i])
	}
	return string(out)
}

func wrapItemInfoLines(lines []string, maxRunes int) []string {
	var out []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, wrapItemInfoLine(line, maxRunes)...)
	}
	return out
}

func wrapItemInfoLine(line string, maxRunes int) []string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	var out []string
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if runeLen(current)+1+runeLen(word) <= maxRunes {
			current += " " + word
			continue
		}
		out = append(out, current)
		current = word
	}
	if current != "" {
		out = append(out, current)
	}
	return out
}

func runeLen(text string) int {
	return len([]rune(text))
}

func itemInfoWidgetColor(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}

type itemInfoCardSlotWidget struct {
	widget.WidgetBase
	image   image.Image
	width   int
	height  int
	cardID  uint16
	onHover func(uint16)
	onLeave func()
	onOpen  func(uint16, int, int)
}

func newItemInfoCardSlotWidget(img image.Image, width, height int, cardID uint16, onHover func(uint16), onLeave func(), onOpen func(uint16, int, int)) *itemInfoCardSlotWidget {
	w := &itemInfoCardSlotWidget{
		image:   img,
		width:   width,
		height:  height,
		cardID:  cardID,
		onHover: onHover,
		onLeave: onLeave,
		onOpen:  onOpen,
	}
	w.SetVisible(true)
	w.SetEnabled(true)
	return w
}

func (w *itemInfoCardSlotWidget) Layout(ctx widget.Context, constraints geometry.Constraints) geometry.Size {
	size := constraints.Constrain(geometry.Sz(float32(w.width), float32(w.height)))
	w.SetBounds(geometry.FromPointSize(w.Position(), size))
	return size
}

func (w *itemInfoCardSlotWidget) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !w.IsVisible() || w.image == nil {
		return
	}
	canvas.DrawImage(w.image, w.Bounds().Min)
}

func (w *itemInfoCardSlotWidget) Event(ctx widget.Context, e event.Event) bool {
	if !w.IsVisible() || !w.IsEnabled() {
		return false
	}
	mouse, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch mouse.MouseType {
	case event.MouseEnter, event.MouseMove:
		if w.cardID == 0 {
			if w.onLeave != nil {
				w.onLeave()
			}
			return false
		}
		if w.onHover != nil {
			w.onHover(w.cardID)
		}
		ctx.SetCursor(widget.CursorPointer)
		return true
	case event.MouseLeave:
		if w.onLeave != nil {
			w.onLeave()
		}
		ctx.SetCursor(widget.CursorDefault)
	case event.MousePress:
		if w.cardID != 0 && mouse.Button == event.ButtonRight {
			if w.onOpen != nil {
				w.onOpen(w.cardID, int(mouse.GlobalPosition.X), int(mouse.GlobalPosition.Y))
			}
			return true
		}
	}
	return false
}
