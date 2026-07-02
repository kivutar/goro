package ui

import (
	"github.com/gogpu/ui/core/button"
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

	children := []widget.Widget{
		primitives.HBox(
			rotheme.Title(cfg.title),
			primitives.Expanded(primitives.Box()),
			windowCloseButton(cfg.closeButton),
		).
			CrossAlign(primitives.CrossAxisCenter).
			PaddingXY(14, 0).
			Height(cfg.titleHeight).
			Background(rotheme.Default.Colors.WindowTitle),
	}
	if cfg.content != nil {
		children = append(children, primitives.Expanded(cfg.content))
	}
	if cfg.footer != nil {
		children = append(children,
			primitives.Box(cfg.footer).
				Padding(cfg.footerPadding).
				Background(rotheme.Default.Colors.WindowFooter),
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
	return button.New(button.TextOpt("x")).PaddingXY(4, 0)
}
