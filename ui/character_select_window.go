package ui

import (
	"fmt"
	"image"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

type CharacterSelectWindowOptions struct {
	SelectedSlot  int
	MaxSlots      int
	Characters    []session.Character
	PreviewImages map[int]image.Image
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

const (
	characterSelectWindowW   = 576
	characterSelectWindowH   = 356
	characterSelectFooterH   = 42
	characterSelectFooterPad = 12
	characterSelectFooterGap = 8
)

func NewCharacterSelectWindow(ctx client.Context, opts CharacterSelectWindowOptions, callbacks CharacterSelectWindowCallbacks) *CharacterSelectWindow {
	x, y, width, height := characterSelectWindowRect(ctx)
	w := &CharacterSelectWindow{
		opts:      opts,
		callbacks: callbacks,
		window:    NewWindowState(width, height),
	}
	w.window.OpenAt(x, y, w.widgetTree())
	return w
}

func (w *CharacterSelectWindow) SetOptions(ctx client.Context, opts CharacterSelectWindowOptions) {
	if w == nil {
		return
	}
	sameTree := characterSelectWindowTreeEqual(w.opts, opts)
	x, y, width, height := characterSelectWindowRect(ctx)
	w.opts = opts
	w.window.SetAutoPosition(x, y)
	w.window.SetSize(width, height)
	if sameTree {
		return
	}
	w.window.SetContent(w.widgetTree())
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

func (w *CharacterSelectWindow) Publish(ctx client.Context) {
	if w == nil || ctx.UIManager == nil {
		return
	}
	w.window.Publish(ctx)
}

func characterSelectWindowTreeEqual(a, b CharacterSelectWindowOptions) bool {
	if a.SelectedSlot != b.SelectedSlot ||
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
		Size(characterSelectWindowW, characterSelectWindowH),
		FooterHeight(characterSelectFooterH),
		FooterPadding(characterSelectFooterPad),
		Content(
			primitives.Box(
				primitives.HBox(
					rotheme.IconButton(rotheme.IconButtonLeft, func() {
						if w.callbacks.OnPreviousPage != nil {
							w.callbacks.OnPreviousPage()
						}
					}),
					primitives.HBox(
						w.slotWidget(pageStart),
						w.slotWidget(pageStart+1),
						w.slotWidget(pageStart+2),
					).
						Gap(25),
					rotheme.IconButton(rotheme.IconButtonRight, func() {
						if w.callbacks.OnNextPage != nil {
							w.callbacks.OnNextPage()
						}
					}),
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
				Gap(characterSelectFooterGap),
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

func CharacterSelectPage(slot int) int {
	if slot < 0 {
		return 0
	}
	return slot / 3
}

func characterSelectWindowRect(ctx client.Context) (int, int, int, int) {
	return centeredWindowRect(ctx, characterSelectWindowW, characterSelectWindowH)
}

func characterBySlot(characters []session.Character, slot int) (session.Character, bool) {
	for _, character := range characters {
		if int(character.Slot) == slot {
			return character, true
		}
	}
	return session.Character{}, false
}
