package ui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	itemInfoWindowWidth       = 468
	itemInfoWindowHeight      = 304
	itemInfoWindowPad         = 10
	itemInfoIllustrationWidth = 132
	itemInfoLineH             = 14
	itemInfoDetailsH          = 72
	itemInfoDescriptionW      = itemInfoWindowWidth - itemInfoWindowPad*2 - itemInfoIllustrationWidth - 12
	itemInfoDescriptionH      = itemInfoWindowHeight - ROWindowTitleHeight - itemInfoWindowPad*2 - itemInfoDetailsH - 10
)

type ItemInfoWindow struct {
	window       WindowState
	item         session.InventoryItem
	title        string
	details      []string
	lines        []string
	illustration image.Image
}

func (w *ItemInfoWindow) openItem(ctx Context, item session.InventoryItem, mouseX, mouseY int) {
	if item.ItemID == 0 {
		return
	}
	w.ensureWindow()
	w.item = item
	w.title = "Item Information"
	w.details = itemInfoDetailLines(item)
	w.lines = itemInfoDescriptionLines(ctx, item)
	w.illustration = nil

	screenW, screenH := ctx.ScreenSize()
	x := clampWindowInt(mouseX+14, 8, maxInt(8, screenW-itemInfoWindowWidth-8))
	y := clampWindowInt(mouseY-22, 8, maxInt(8, screenH-itemInfoWindowHeight-8))
	w.window.OpenAt(x, y, w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *ItemInfoWindow) Update(ctx Context, assets AssetProvider) bool {
	w.ensureWindow()
	if !w.window.IsOpen() {
		return false
	}
	if w.illustration == nil && assets != nil {
		w.illustration = assets.ItemInfoIllustrationImage(ctx.Resources, w.item, itemInfoIllustrationWidth-14, itemInfoWindowHeight-ROWindowTitleHeight-itemInfoWindowPad*2-14)
		w.window.SetContent(w.widgetTree(ctx))
	}
	consumed := w.window.Update(ctx)
	if !w.window.IsOpen() {
		w.Publish(ctx)
		return consumed
	}
	w.Publish(ctx)
	return consumed
}

func (w *ItemInfoWindow) Publish(ctx Context) {
	w.ensureWindow()
	if !w.window.IsOpen() {
		w.window.Unpublish(ctx)
		return
	}
	w.window.Publish(ctx)
}

func (w *ItemInfoWindow) ensureWindow() {
	if w.window.width == 0 {
		w.window = NewWindowState(itemInfoWindowWidth, itemInfoWindowHeight)
	}
}

func (w *ItemInfoWindow) widgetTree(ctx Context) widget.Widget {
	return Window(
		Title(w.title),
		CloseButton(true),
		OnClose(func() {
			w.window.Close()
			w.Publish(ctx)
		}),
		Size(itemInfoWindowWidth, itemInfoWindowHeight),
		Content(
			primitives.HBox(
				w.illustrationPanel(),
				w.infoPanel(ctx),
			).
				Padding(itemInfoWindowPad).
				Gap(12),
		),
	)
}

func (w *ItemInfoWindow) illustrationPanel() widget.Widget {
	return primitives.Box(
		primitives.Box(
			newStaticImageWidget(w.illustration, itemInfoIllustrationWidth-14, itemInfoWindowHeight-ROWindowTitleHeight-itemInfoWindowPad*2-14),
		).
			Padding(7),
	).
		Width(itemInfoIllustrationWidth).
		Background(rotheme.Default.Colors.PanelBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder)
}

func (w *ItemInfoWindow) infoPanel(ctx Context) widget.Widget {
	return primitives.Box(
		w.detailPanel(ctx),
		w.descriptionPanel(),
	).
		Width(itemInfoDescriptionW).
		Gap(10)
}

func (w *ItemInfoWindow) detailPanel(ctx Context) widget.Widget {
	name := inventoryItemDisplayName(ctx.Resources, w.item)
	if w.item.Refine > 0 {
		name = fmt.Sprintf("+%d %s", w.item.Refine, name)
	}
	details := make([]widget.Widget, 0, len(w.details)+1)
	details = append(details, rotheme.Text(name))
	for _, line := range w.details {
		details = append(details,
			rotheme.Text(line).
				Color(itemInfoWidgetColor(inventoryMutedColor)).
				LineHeight(itemInfoLineH/rotheme.Default.Typography.TextSize),
		)
	}
	return primitives.Box(details...).
		Height(itemInfoDetailsH).
		Gap(1)
}

func (w *ItemInfoWindow) descriptionPanel() widget.Widget {
	lines := w.wrappedLines(itemInfoDescriptionRunes())
	if len(lines) == 0 {
		lines = []string{"No description available."}
	}
	textLines := make([]widget.Widget, 0, len(lines))
	for _, line := range lines {
		textLines = append(textLines,
			rotheme.Text(line).
				Color(itemInfoWidgetColor(itemInfoDescriptionColor(line))).
				LineHeight(itemInfoLineH/rotheme.Default.Typography.TextSize),
		)
	}
	return primitives.Box(
		scrollview.New(
			primitives.Box(textLines...).
				Padding(7).
				Gap(0),
			scrollview.DirectionOpt(scrollview.Vertical),
			scrollview.ScrollbarOpt(scrollview.ScrollbarAuto),
			scrollview.ScrollStep(itemInfoLineH),
		),
	).
		Height(itemInfoDescriptionH).
		Background(rotheme.Default.Colors.PanelBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder)
}

func (w *ItemInfoWindow) wrappedLines(maxRunes int) []string {
	return wrapItemInfoLines(w.lines, maxRunes)
}

func itemInfoDescriptionRunes() int {
	return maxInt(10, (itemInfoDescriptionW-18)/7)
}

func itemInfoDetailLines(item session.InventoryItem) []string {
	lines := []string{fmt.Sprintf("Item ID: %d", item.ItemID)}
	if item.Amount > 1 {
		lines = append(lines, fmt.Sprintf("Amount: %d", item.Amount))
	}
	if item.Type != 0 {
		lines = append(lines, fmt.Sprintf("Type: %s", itemInfoTypeLabel(item.Type)))
	}
	if item.Equipped {
		lines = append(lines, "Equipped")
	}
	if !item.Identified {
		lines = append(lines, "Unidentified")
	}
	return lines
}

func itemInfoDescriptionLines(ctx Context, item session.InventoryItem) []string {
	if ctx.Resources == nil {
		return nil
	}
	lines, ok := ctx.Resources.ItemDescription(int(item.ItemID), item.Identified)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(stripItemInfoColorCodes(strings.ReplaceAll(line, "_", " ")))
		if line == "" {
			out = append(out, "")
			continue
		}
		out = append(out, line)
	}
	return out
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

func itemInfoTypeLabel(itemType uint8) string {
	switch itemType {
	case 0:
		return "Healing"
	case 2:
		return "Usable"
	case 3:
		return "Etc"
	case 4:
		return "Weapon"
	case 5:
		return "Armor"
	case 6:
		return "Card"
	case 7:
		return "Pet Egg"
	case 8:
		return "Pet Equipment"
	case 10:
		return "Ammo"
	case 11:
		return "Delayed Consumable"
	case 12:
		return "Shadow Gear"
	case 18:
		return "Cash Usable"
	default:
		return fmt.Sprintf("%d", itemType)
	}
}

func itemInfoDescriptionColor(line string) color.RGBA {
	if strings.HasPrefix(strings.TrimSpace(line), "Class :") || strings.HasPrefix(strings.TrimSpace(line), "Weight :") {
		return inventoryMutedColor
	}
	return inventoryTextColor
}

func itemInfoWidgetColor(c color.RGBA) widget.Color {
	return widget.RGBA8(c.R, c.G, c.B, c.A)
}
