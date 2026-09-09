package ui

import (
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
)

type skillInfoWindow struct {
	Window
	grid *skillGridWidget
}

func (w *skillInfoWindow) createSkillInfoWidget(ctx Context, title string, skillDesc []string) widget.Widget {
	options := []WindowOption{
		Title(title),
		CloseButton(true),
		OnClose(func() {
			w.Window.Close()
			w.Publish(ctx)
		}),
		Size(float32(w.width), float32(w.height)),
		Content(
			primitives.HBox(
				//w.illustrationPanel(),
				w.infoPanel(ctx, skillDesc),
			).
				Padding(10).
				Gap(12),
		),
	}

	return Win(options...)
}

func (w *skillInfoWindow) descriptionPanel(ctx Context, skillDesc []string) widget.Widget {
	var desc []itemInfoTextLine
	for _, l := range skillDesc {
		desc = append(desc, parseItemInfoTextLine(l))
	}

	lines := wrapItemInfoTextLines(desc, itemInfoDescriptionRunes())
	if len(lines) == 0 {
		lines = []itemInfoTextLine{parseItemInfoTextLine("No description available.")}
	}
	textLines := make([]widget.Widget, 0, len(lines))
	for _, line := range desc {
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
	).Width(float32(w.width - ROScrollbarGutter))
}

func (w *skillInfoWindow) infoPanel(ctx Context, skillDesc []string) widget.Widget {
	children := []widget.Widget{
		w.descriptionPanel(ctx, skillDesc),
	}
	return primitives.Box(children...).
		Width(itemInfoDescriptionW).
		Gap(8)
}

//func (w *skillInfoWindow) illustrationPanel() widget.Widget {
//	return primitives.Box(
//		//newStaticImageWidget(image.new, itemInfoIllustrationWidth, itemInfoIllustrationH),
//	).
//		Height(itemInfoIllustrationH).
//		Width(itemInfoIllustrationWidth).
//		Background(rotheme.Default.Colors.PanelBody).
//		BorderStyle(1, rotheme.Default.Colors.WindowBorder)
//}

func (w *skillInfoWindow) openSkillInfo(ctx Context, skill session.Skill, mx int, my int) {
	maxRunes := 38

	skillInfoWindowWidth := 340
	skillInfoWindowHeight := 300

	// Create new window, so we don't hijack Skill Tree Window.
	w.EnsureWindow(skillInfoWindowWidth, skillInfoWindowHeight)

	// SetSize appears to be redundant, we've just created the window and specified the size.
	//w.SetSize(skillInfoWindowWidth, skillInfoWindowHeight)

	name := trimRunes(skillDisplayName(ctx.Resources, skill), maxRunes)
	skillDesc := append([]string{name}, skillTooltipLines(ctx, skill)...)

	skillInfoWidget := w.createSkillInfoWidget(ctx, name, skillDesc)

	screenW, screenH := ctx.ScreenSize()
	x := clampWindowInt(mx+14, windowScreenMargin, maxInt(windowScreenMargin, screenW-skillInfoWindowWidth-windowScreenMargin))
	y := clampWindowInt(my-22, windowScreenMargin, maxInt(windowScreenMargin, screenH-skillInfoWindowHeight-windowScreenMargin))

	w.OpenAt(x, y, skillInfoWidget)
	w.Publish(ctx)
}
