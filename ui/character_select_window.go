package ui

import (
	"fmt"
	"image"
	"image/color"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

type CharacterPreviewFunc func(screen *render.Image, character session.Character, centerX, feetY int)

type CharacterSelectWindowOptions struct {
	X, Y, W, H      int
	TitleH          int
	FooterH         int
	FooterPadX      int
	FooterGap       int
	ButtonH         int
	PreviewFeetLift int
	SelectedSlot    int
	MaxSlots        int
	Characters      []session.Character
	MouseX, MouseY  int
	HasMouse        bool
	DrawPreview     CharacterPreviewFunc
	PreviewImages   map[int]image.Image
}

type CharacterSelectWindowCallbacks struct {
	OnSelectSlot   func(int)
	OnActivateSlot func(int)
	OnPreviousPage func()
	OnNextPage     func()
	OnMake         func()
	OnOK           func()
	OnCancel       func()
	OnDelete       func()
}

type CharacterSelectWindow struct {
	opts      CharacterSelectWindowOptions
	callbacks CharacterSelectWindowCallbacks
	window    WindowState
}

func NewCharacterSelectWindow(opts CharacterSelectWindowOptions, callbacks CharacterSelectWindowCallbacks) *CharacterSelectWindow {
	w := &CharacterSelectWindow{
		opts:      opts,
		callbacks: callbacks,
		window:    NewWindowState(opts.W, opts.H),
	}
	w.window.OpenAt(opts.X, opts.Y, w.widgetTree())
	return w
}

func (w *CharacterSelectWindow) SetOptions(opts CharacterSelectWindowOptions) {
	if w == nil {
		return
	}
	sameTree := characterSelectWindowTreeEqual(w.opts, opts)
	w.opts = opts
	w.window.SetAutoPosition(opts.X, opts.Y)
	w.window.SetSize(opts.W, opts.H)
	if sameTree {
		return
	}
	w.window.SetRoot(w.widgetTree())
}

func (w *CharacterSelectWindow) Widget() widget.Widget {
	if w == nil {
		return nil
	}
	return w.window.Widget()
}

func (w *CharacterSelectWindow) Update(ctx client.Context) bool {
	if w == nil {
		return false
	}
	return w.window.Update(ctx)
}

func characterSelectWindowTreeEqual(a, b CharacterSelectWindowOptions) bool {
	if a.W != b.W ||
		a.H != b.H ||
		a.FooterH != b.FooterH ||
		a.FooterPadX != b.FooterPadX ||
		a.FooterGap != b.FooterGap ||
		a.SelectedSlot != b.SelectedSlot ||
		a.MaxSlots != b.MaxSlots ||
		len(a.Characters) != len(b.Characters) {
		return false
	}
	for i := range a.Characters {
		if a.Characters[i] != b.Characters[i] {
			return false
		}
	}
	if len(a.PreviewImages) != len(b.PreviewImages) {
		return false
	}
	for slot, image := range a.PreviewImages {
		if b.PreviewImages[slot] != image {
			return false
		}
	}
	return true
}

func (w *CharacterSelectWindow) widgetTree() widget.Widget {
	page := CharacterSelectPage(w.opts.SelectedSlot)
	pageCount := maxInt(1, (w.opts.MaxSlots+2)/3)
	pageStart := page * 3
	selected, hasSelection := characterBySlot(w.opts.Characters, w.opts.SelectedSlot)
	buttonW := func(label string) float32 {
		return float32(ButtonLabelWidth(label))
	}
	return Window(
		Title("Select Character"),
		CloseButton(false),
		Size(float32(w.opts.W), float32(w.opts.H)),
		FooterHeight(float32(w.opts.FooterH)),
		FooterPadding(float32(w.opts.FooterPadX)),
		Content(
			primitives.Box(
				primitives.HBox(
					rotheme.Button("<", func() {
						if w.callbacks.OnPreviousPage != nil {
							w.callbacks.OnPreviousPage()
						}
					}).
						Width(18).
						Height(18),
					primitives.HBox(
						w.slotWidget(pageStart),
						w.slotWidget(pageStart+1),
						w.slotWidget(pageStart+2),
					).
						Gap(25),
					rotheme.Button(">", func() {
						if w.callbacks.OnNextPage != nil {
							w.callbacks.OnNextPage()
						}
					}).
						Width(18).
						Height(18),
				).
					CrossAlign(primitives.CrossAxisCenter).
					Gap(18),

				rotheme.Text(fmt.Sprintf("%d / %d", page+1, pageCount)).
					Color(rotheme.Default.Colors.MutedText),

				w.infoPanel(selected, hasSelection),
			).
				PaddingTop(12).
				PaddingLeft(24).
				PaddingRight(24).
				Gap(9).
				CrossAlign(primitives.CrossAxisCenter),
		),
		Footer(
			primitives.HBox(
				rotheme.Button("Delete", func() {
					if w.callbacks.OnDelete != nil {
						w.callbacks.OnDelete()
					}
				}).
					Width(buttonW("Delete")),
				primitives.Expanded(primitives.Box()),
				rotheme.Button("Make", func() {
					if w.callbacks.OnMake != nil {
						w.callbacks.OnMake()
					}
				}).
					Width(buttonW("Make")),
				rotheme.Button("OK", func() {
					if w.callbacks.OnOK != nil {
						w.callbacks.OnOK()
					}
				}).
					Width(buttonW("OK")),
				rotheme.Button("Cancel", func() {
					if w.callbacks.OnCancel != nil {
						w.callbacks.OnCancel()
					}
				}).
					Width(buttonW("Cancel")),
			).
				CrossAlign(primitives.CrossAxisCenter).
				Gap(float32(w.opts.FooterGap)),
		),
	)
}

func (w *CharacterSelectWindow) slotWidget(slot int) widget.Widget {
	_, hasCharacter := characterBySlot(w.opts.Characters, slot)
	var preview image.Image
	if w.opts.PreviewImages != nil {
		preview = w.opts.PreviewImages[slot]
	}
	return primitives.Box(
		primitives.Expanded(
			button.New(
				button.TextOpt(""),
				button.PainterOpt(characterSelectSlotPainter{
					selected:     slot == w.opts.SelectedSlot,
					hasCharacter: hasCharacter,
					preview:      preview,
				}),
				button.OnClick(func() {
					if slot == w.opts.SelectedSlot {
						if w.callbacks.OnActivateSlot != nil {
							w.callbacks.OnActivateSlot(slot)
						}
						return
					}
					if w.callbacks.OnSelectSlot != nil {
						w.callbacks.OnSelectSlot(slot)
					}
				}),
			),
		),
	).
		Width(139).
		Height(144).
		CrossAlign(primitives.CrossAxisStretch)
}

type characterSelectSlotPainter struct {
	selected     bool
	hasCharacter bool
	preview      image.Image
}

func (p characterSelectSlotPainter) PaintButton(canvas widget.Canvas, state button.PaintState) {
	bounds := state.Bounds
	if bounds.IsEmpty() {
		return
	}
	bg := rotheme.Default.Colors.PanelBody
	border := rotheme.Default.Colors.WindowBorder
	if p.selected {
		bg = widget.RGBA8(222, 237, 252, 255)
		border = rotheme.Default.Colors.ButtonBorder
	}
	if state.Hovered || state.Pressed {
		border = rotheme.Default.Colors.ButtonBorder
	}
	canvas.DrawRect(bounds, bg)
	canvas.StrokeRect(bounds, border, 1)
	if p.preview != nil {
		imgBounds := p.preview.Bounds()
		x := bounds.Min.X + (bounds.Width()-float32(imgBounds.Dx()))/2
		y := bounds.Min.Y + (bounds.Height()-float32(imgBounds.Dy()))/2
		canvas.DrawImage(p.preview, geometry.Pt(x, y))
		return
	}
	if !p.hasCharacter {
		rotheme.DrawText(canvas, "Create", bounds, rotheme.Default.Typography.TextSize, rotheme.Default.Colors.MutedText, false, widget.TextAlignCenter)
	}
}

func (w *CharacterSelectWindow) infoPanel(character session.Character, hasCharacter bool) widget.Widget {
	if !hasCharacter {
		return primitives.Box(
			rotheme.Text("Empty Slot"),
			rotheme.Text("Use Make to create a character later."),
		).
			Width(318).
			Height(88).
			Padding(12).
			Gap(8).
			Background(rotheme.Default.Colors.PanelBody).
			BorderStyle(1, rotheme.Default.Colors.WindowBorder)
	}
	return primitives.Box(
		primitives.HBox(
			primitives.Box(
				rotheme.Text(trimRunes(character.Name, 24)),
				rotheme.Text(fmt.Sprintf("Job: %s", trimRunes(CharacterJobName(character), 18))),
				rotheme.Text(fmt.Sprintf("Lv: %d / Job %d", character.Level, character.JobLevel)),
				rotheme.Text(fmt.Sprintf("HP: %d / %d", character.HP, character.MaxHP)),
				rotheme.Text(fmt.Sprintf("SP: %d / %d", character.SP, character.MaxSP)),
			).
				Width(160).
				Gap(2),
			primitives.Box(
				rotheme.Text(fmt.Sprintf("STR %d", character.Str)),
				rotheme.Text(fmt.Sprintf("AGI %d", character.Agi)),
				rotheme.Text(fmt.Sprintf("VIT %d", character.Vit)),
			).
				Width(58).
				Gap(2),
			primitives.Box(
				rotheme.Text(fmt.Sprintf("INT %d", character.Int)),
				rotheme.Text(fmt.Sprintf("DEX %d", character.Dex)),
				rotheme.Text(fmt.Sprintf("LUK %d", character.Luk)),
			).
				Width(58).
				Gap(2),
		).
			Gap(8),
	).
		Width(318).
		Height(88).
		Padding(10).
		Background(rotheme.Default.Colors.PanelBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder)
}

func DrawCharacterSelectWindow(screen *render.Image, opts CharacterSelectWindowOptions) {
	if screen == nil {
		return
	}
	DrawTitledWindowFrame(screen, opts.X, opts.Y, opts.W, opts.H, opts.TitleH)
	DrawWindowTitle(screen, opts.X, opts.Y, opts.TitleH, 12, "Select Character", TitleTextColor)

	page := CharacterSelectPage(opts.SelectedSlot)
	pageStart := page * 3
	for localSlot := 0; localSlot < 3; localSlot++ {
		slot := pageStart + localSlot
		slotX, slotY, slotW, slotH := CharacterSelectSlotRect(opts, localSlot)
		selected := slot == opts.SelectedSlot
		bg := PanelBodyColor
		border := WindowBorderColor
		if selected {
			bg = SelectionColor
			border = SelectionBorder
		}
		DrawSurface(screen, slotX, slotY, slotW, slotH, bg, border)
		if character, ok := characterBySlot(opts.Characters, slot); ok {
			if opts.DrawPreview != nil {
				opts.DrawPreview(screen, character, slotX+slotW/2, slotY+slotH-15-opts.PreviewFeetLift)
			}
		} else {
			render.DebugPrintAtColor(screen, "Create", slotX+45, slotY+58, MutedTextColor)
		}
	}

	leftX, leftY, leftW, leftH := CharacterSelectLeftArrowRect(opts)
	rightX, rightY, rightW, rightH := CharacterSelectRightArrowRect(opts)
	drawCharSelectArrow(screen, leftX, leftY, leftW, leftH, "<")
	drawCharSelectArrow(screen, rightX, rightY, rightW, rightH, ">")
	drawSelectedCharacterInfo(screen, opts)
	drawCharacterSelectFooter(screen, opts)
}

func CharacterSelectSlotRect(opts CharacterSelectWindowOptions, localSlot int) (int, int, int, int) {
	lefts := [3]int{60, 224, 386}
	if localSlot < 0 || localSlot >= len(lefts) {
		localSlot = 0
	}
	return opts.X + lefts[localSlot] - 5, opts.Y + 40, 139, 144
}

func CharacterSelectLeftArrowRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return opts.X + 24, opts.Y + 105, 18, 18
}

func CharacterSelectRightArrowRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return opts.X + 534, opts.Y + 105, 18, 18
}

func CharacterSelectFooterRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return opts.X, opts.Y + opts.H - opts.FooterH, opts.W, opts.FooterH
}

func CharacterSelectInfoPanelRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return opts.X + 16, opts.Y + 202, 318, 88
}

func CharacterSelectPagerTextRect(opts CharacterSelectWindowOptions, label string) (int, int, int, int) {
	_, slotY, _, slotH := CharacterSelectSlotRect(opts, 0)
	_, panelY, _, _ := CharacterSelectInfoPanelRect(opts)
	textW, textH := render.DebugTextSize(label)
	gap := panelY - (slotY + slotH)
	if gap < textH {
		gap = textH
	}
	return opts.X + (opts.W-textW)/2, slotY + slotH + (gap-textH)/2, textW, textH
}

func CharacterSelectDeleteButtonRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	_, footerY, _, footerH := CharacterSelectFooterRect(opts)
	return opts.X + opts.FooterPadX, footerY + (footerH-opts.ButtonH)/2, ButtonLabelWidth("Delete"), opts.ButtonH
}

func CharacterSelectMakeButtonRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return characterSelectRightButtonRect(opts, 0)
}

func CharacterSelectOKButtonRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return characterSelectRightButtonRect(opts, 1)
}

func CharacterSelectCancelButtonRect(opts CharacterSelectWindowOptions) (int, int, int, int) {
	return characterSelectRightButtonRect(opts, 2)
}

func characterSelectRightButtonRect(opts CharacterSelectWindowOptions, index int) (int, int, int, int) {
	labels := [...]string{"Make", "OK", "Cancel"}
	if index < 0 || index >= len(labels) {
		index = 0
	}
	_, footerY, _, footerH := CharacterSelectFooterRect(opts)
	totalW := 0
	for _, label := range labels {
		totalW += ButtonLabelWidth(label)
	}
	totalW += opts.FooterGap * (len(labels) - 1)
	bx := opts.X + opts.W - opts.FooterPadX - totalW
	for i := 0; i < index; i++ {
		bx += ButtonLabelWidth(labels[i]) + opts.FooterGap
	}
	return bx, footerY + (footerH-opts.ButtonH)/2, ButtonLabelWidth(labels[index]), opts.ButtonH
}

func CharacterSelectPage(slot int) int {
	if slot < 0 {
		return 0
	}
	return slot / 3
}

func drawSelectedCharacterInfo(screen *render.Image, opts CharacterSelectWindowOptions) {
	character, ok := characterBySlot(opts.Characters, opts.SelectedSlot)
	panelX, panelY, panelW, panelH := CharacterSelectInfoPanelRect(opts)
	DrawPanelSurface(screen, panelX, panelY, panelW, panelH, PanelBodyColor)
	if !ok {
		render.DebugPrintAtColor(screen, "Empty Slot", panelX+18, panelY+14, TextColor)
		render.DebugPrintAtColor(screen, "Use Make to create a character later.", panelX+18, panelY+34, TextColor)
		return
	}
	render.DebugPrintAtColor(screen, trimRunes(character.Name, 24), panelX+14, panelY+8, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("Job: %s", trimRunes(CharacterJobName(character), 18)), panelX+14, panelY+24, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("Lv: %d / Job %d", character.Level, character.JobLevel), panelX+14, panelY+40, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("HP: %d / %d", character.HP, character.MaxHP), panelX+14, panelY+56, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("SP: %d / %d", character.SP, character.MaxSP), panelX+14, panelY+72, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("STR %d", character.Str), panelX+180, panelY+8, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("AGI %d", character.Agi), panelX+180, panelY+24, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("VIT %d", character.Vit), panelX+180, panelY+40, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("INT %d", character.Int), panelX+246, panelY+8, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("DEX %d", character.Dex), panelX+246, panelY+24, TextColor)
	render.DebugPrintAtColor(screen, fmt.Sprintf("LUK %d", character.Luk), panelX+246, panelY+40, TextColor)
}

func drawCharacterSelectFooter(screen *render.Image, opts CharacterSelectWindowOptions) {
	page := CharacterSelectPage(opts.SelectedSlot)
	pageCount := maxInt(1, (opts.MaxSlots+2)/3)
	DrawWindowFooter(screen, opts.X, opts.Y, opts.W, opts.H, opts.FooterH)
	pageLabel := fmt.Sprintf("%d / %d", page+1, pageCount)
	pageX, pageY, _, _ := CharacterSelectPagerTextRect(opts, pageLabel)
	render.DebugPrintAtColor(screen, pageLabel, pageX, pageY, MutedTextColor)

	drawCharSelectButtonRect(screen, opts, rectArray(CharacterSelectDeleteButtonRect(opts)), "Delete", TextColor)
	drawCharSelectButtonRect(screen, opts, rectArray(CharacterSelectMakeButtonRect(opts)), "Make", TextColor)
	drawCharSelectButtonRect(screen, opts, rectArray(CharacterSelectOKButtonRect(opts)), "OK", TextColor)
	drawCharSelectButtonRect(screen, opts, rectArray(CharacterSelectCancelButtonRect(opts)), "Cancel", TextColor)
}

func drawCharSelectButtonRect(screen *render.Image, opts CharacterSelectWindowOptions, rect [4]int, label string, textColor color.RGBA) {
	x, y, w, h := rect[0], rect[1], rect[2], rect[3]
	bg := ButtonColor
	if opts.HasMouse && pointInRect(opts.MouseX, opts.MouseY, x, y, w, h) {
		bg = ButtonHoverColor
	}
	DrawButtonLabel(screen, x, y, w, h, label, bg, textColor)
}

func drawCharSelectArrow(screen *render.Image, x, y, w, h int, label string) {
	DrawButtonLabel(screen, x, y, w, h, label, ButtonColor, TextColor)
}

func characterBySlot(characters []session.Character, slot int) (session.Character, bool) {
	for _, character := range characters {
		if int(character.Slot) == slot {
			return character, true
		}
	}
	return session.Character{}, false
}
