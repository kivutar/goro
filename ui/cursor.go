package ui

import (
	"fmt"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/session"
)

func DrawPendingSkillCursorLevel(screen *render.Image, ctx client.Context, skill session.Skill) {
	if screen == nil || ctx.Input == nil {
		return
	}
	if skill.ID == 0 || skill.Level <= 0 {
		return
	}
	label := fmt.Sprintf("Lv%d", skill.Level)
	x := ctx.Input.MouseX + 18
	y := ctx.Input.MouseY + 16
	width := len([]rune(label))*7 + 8
	DrawSurface(screen, x, y, width, 15, PanelBodyColor, WindowBorderColor)
	render.DebugPrintAtColor(screen, label, x+4, y+1, TitleTextColor)
}
