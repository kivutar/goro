package ui

import (
	"fmt"
	"strings"

	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/ui/rotheme"
)

const (
	bookWindowWidth        = 555
	bookWindowHeight       = 455
	bookWindowPaddingX     = 28
	bookWindowPaddingY     = 12
	bookWindowLineHeight   = 14
	bookWindowLinesPerPage = 25
	bookWindowLineRunes    = 70
	bookWindowNavigationH  = 28
)

type BookWindow struct {
	Window
	itemID     uint16
	title      string
	content    res.BookContent
	lines      []itemInfoTextLine
	page       int
	totalPages int
}

func (w *BookWindow) Open(ctx Context, itemID uint16, title string) error {
	if ctx.Resources == nil {
		return fmt.Errorf("book resources unavailable")
	}
	content, err := ctx.Resources.LoadBook(itemID)
	if err != nil {
		return fmt.Errorf("read book %d: %w", itemID, err)
	}
	w.itemID = itemID
	w.title = strings.TrimSpace(title)
	if w.title == "" {
		w.title = fmt.Sprintf("Book #%d", itemID)
	}
	w.content = content
	w.lines = wrapBookTextLines(content.Lines)
	if !bookLinesHaveContent(w.lines) {
		w.lines = []itemInfoTextLine{parseItemInfoTextLine("This book is empty.")}
	}
	w.page = 0
	w.totalPages = maxInt(1, (len(w.lines)+bookWindowLinesPerPage-1)/bookWindowLinesPerPage)
	w.EnsureWindow(bookWindowWidth, bookWindowHeight)
	w.SetSize(bookWindowWidth, bookWindowHeight)
	w.Window.Open(ctx, w.widgetTree(ctx))
	w.Publish(ctx)
	return nil
}

func bookLinesHaveContent(lines []itemInfoTextLine) bool {
	for _, line := range lines {
		if strings.TrimSpace(itemInfoLinePlainText(line)) != "" {
			return true
		}
	}
	return false
}

func (w *BookWindow) Update(ctx Context) bool {
	if !w.IsOpen() {
		return false
	}
	consumed := w.Window.Update(ctx)
	w.Publish(ctx)
	return consumed
}

func (w *BookWindow) Rebind(ctx Context) {
	if !w.IsOpen() {
		return
	}
	w.ctx = ctx
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *BookWindow) widgetTree(ctx Context) widget.Widget {
	background := widget.RGBA8(w.content.Background.R, w.content.Background.G, w.content.Background.B, w.content.Background.A)
	lineWidgets := make([]widget.Widget, 0, bookWindowLinesPerPage)
	for _, line := range w.pageLines() {
		lineWidgets = append(lineWidgets, itemInfoTextLineWidget(line))
	}
	for len(lineWidgets) < bookWindowLinesPerPage {
		lineWidgets = append(lineWidgets, itemInfoTextLineWidget(itemInfoTextLine{}))
	}
	return Win(
		Title(w.windowTitle()),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(bookWindowWidth, bookWindowHeight),
		Background(background),
		Content(
			primitives.Box(
				primitives.Box(lineWidgets...).
					Gap(0).
					Height(bookWindowLinesPerPage*bookWindowLineHeight),
				primitives.Expanded(primitives.Box()),
				primitives.HBox(
					rotheme.ButtonDisabled("Previous", w.page == 0, func() { w.setPage(ctx, w.page-1) }),
					primitives.Expanded(primitives.Box()),
					rotheme.ButtonDisabled("Next", w.page+1 >= w.totalPages, func() { w.setPage(ctx, w.page+1) }),
				).
					CrossAlign(primitives.CrossAxisCenter).
					Height(bookWindowNavigationH),
			).
				PaddingXY(bookWindowPaddingX, bookWindowPaddingY).
				Background(background),
		),
	)
}

func (w *BookWindow) setPage(ctx Context, page int) {
	page = clampWindowInt(page, 0, maxInt(0, w.totalPages-1))
	if page == w.page {
		return
	}
	w.page = page
	w.SetContent(w.widgetTree(ctx))
	w.Publish(ctx)
}

func (w *BookWindow) pageLines() []itemInfoTextLine {
	start := w.page * bookWindowLinesPerPage
	if start < 0 || start >= len(w.lines) {
		return nil
	}
	end := minInt(start+bookWindowLinesPerPage, len(w.lines))
	return w.lines[start:end]
}

func (w *BookWindow) windowTitle() string {
	return fmt.Sprintf("%s  (%d/%d)", w.title, w.page+1, maxInt(1, w.totalPages))
}

func wrapBookTextLines(lines []string) []itemInfoTextLine {
	parsed := make([]itemInfoTextLine, 0, len(lines))
	for _, line := range lines {
		parsed = append(parsed, parseItemInfoTextLine(line))
	}
	return wrapItemInfoTextLines(parsed, bookWindowLineRunes)
}
