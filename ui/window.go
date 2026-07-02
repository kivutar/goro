package ui

import (
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/ui/rotheme"
)

type WindowOption func(*windowConfig)

type windowConfig struct {
	title         string
	closeButton   bool
	content       widget.Widget
	footer        widget.Widget
	width         float32
	height        float32
	titleHeight   float32
	footerPadding float32
	footerHeight  float32
}

func Window(options ...WindowOption) widget.Widget {
	cfg := windowConfig{
		width:         300,
		height:        240,
		titleHeight:   28,
		footerPadding: 8,
	}
	for _, option := range options {
		option(&cfg)
	}

	titleContent := primitives.HBox(
		rotheme.Title(cfg.title),
		primitives.Expanded(primitives.Box()),
		windowCloseButton(cfg.closeButton),
	).
		CrossAlign(primitives.CrossAxisCenter).
		PaddingXY(14, 0).
		Height(cfg.titleHeight)

	children := []widget.Widget{
		roTitleBar(
			titleContent,
			cfg.titleHeight,
		),
	}
	if cfg.content != nil {
		children = append(children, primitives.Expanded(cfg.content))
	}
	if cfg.footer != nil {
		footerBody := primitives.Box(cfg.footer).
			Padding(cfg.footerPadding).
			Background(rotheme.Default.Colors.WindowFooter)
		if cfg.footerHeight > 0 {
			footerBody = primitives.Box(
				primitives.Expanded(primitives.Box()),
				cfg.footer,
				primitives.Expanded(primitives.Box()),
			).
				PaddingXY(cfg.footerPadding, 0).
				Height(cfg.footerHeight - 1).
				Background(rotheme.Default.Colors.WindowFooter)
		}
		footer := primitives.Box(
			primitives.HBox(
				primitives.Expanded(
					primitives.Box().
						Height(1).
						Background(rotheme.Default.Colors.FooterLine),
				),
			).
				Height(1),
			footerBody,
		)
		children = append(children,
			footer,
		)
	}

	return primitives.Box(children...).
		Width(cfg.width).
		Height(cfg.height).
		Background(rotheme.Default.Colors.WindowBody).
		BorderStyle(1, rotheme.Default.Colors.WindowBorder).
		Rounded(WindowRadius)
}

func Title(title string) WindowOption {
	return func(cfg *windowConfig) {
		cfg.title = title
	}
}

func CloseButton(enabled bool) WindowOption {
	return func(cfg *windowConfig) {
		cfg.closeButton = enabled
	}
}

func Content(content widget.Widget) WindowOption {
	return func(cfg *windowConfig) {
		cfg.content = content
	}
}

func Footer(footer widget.Widget) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footer = footer
	}
}

func TitleHeight(height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.titleHeight = height
	}
}

func FooterPadding(padding float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footerPadding = padding
	}
}

func FooterHeight(height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.footerHeight = height
	}
}

func Size(width, height float32) WindowOption {
	return func(cfg *windowConfig) {
		cfg.width = width
		cfg.height = height
	}
}

func windowCloseButton(enabled bool) widget.Widget {
	if !enabled {
		return primitives.Box().Width(17).Height(17)
	}
	return primitives.Box(
		rotheme.Button("x", nil).
			PaddingXY(0, 0).
			MinWidth(17),
	).
		Width(17).
		Height(17)
}
