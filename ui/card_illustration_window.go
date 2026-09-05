package ui

import (
	"fmt"
	"image"
	"strings"

	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
)

const (
	cardIllustrationWidth        = 300
	cardIllustrationHeight       = 400
	cardIllustrationWindowHeight = ROWindowTitleHeight + cardIllustrationHeight
)

type CardIllustrationWindow struct {
	Window
	title string
	image image.Image
}

func (w *CardIllustrationWindow) Open(ctx Context, itemID uint16, title string) error {
	if ctx.Resources == nil {
		return fmt.Errorf("card illustration resources unavailable")
	}
	resource, ok := ctx.Resources.ItemCardIllustrationName(int(itemID))
	if !ok {
		return fmt.Errorf("card %d has no illustration resource", itemID)
	}
	img, _, err := res.LoadImage(ctx.Resources, res.CardIllustrationTextureCandidates(resource))
	if err != nil {
		return fmt.Errorf("load card %d illustration: %w", itemID, err)
	}

	w.title = strings.TrimSpace(title)
	if w.title == "" {
		w.title = fmt.Sprintf("Card #%d", itemID)
	}
	w.image = img
	w.EnsureWindow(cardIllustrationWidth, cardIllustrationWindowHeight)
	w.SetSize(cardIllustrationWidth, cardIllustrationWindowHeight)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
	return nil
}

func (w *CardIllustrationWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *CardIllustrationWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.RebindContent(ctx, w.widgetTree(ctx))
}

func (w *CardIllustrationWindow) widgetTree(ctx Context) widget.Widget {
	return Win(
		Title(w.title),
		CloseButton(true),
		OnClose(func() {
			w.Close()
			w.Publish(ctx)
		}),
		Size(cardIllustrationWidth, cardIllustrationWindowHeight),
		Content(newStaticImageWidget(w.image, cardIllustrationWidth, cardIllustrationHeight)),
	)
}
