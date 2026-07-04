package ui

import (
	"image/color"
	"testing"

	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

func TestNPCDialogTextRunsParseColorCodes(t *testing.T) {
	base := color.RGBA{R: 246, G: 242, B: 232, A: 255}
	runs := npcDialogTextRuns("hello ^FF3300red^000000 base", base)

	if len(runs) != 3 {
		t.Fatalf("run count = %d, want 3: %#v", len(runs), runs)
	}
	if runs[0].text != "hello " || runs[0].color != base {
		t.Fatalf("first run = %#v", runs[0])
	}
	if runs[1].text != "red" || runs[1].color != (color.RGBA{R: 255, G: 51, B: 0, A: 255}) {
		t.Fatalf("colored run = %#v", runs[1])
	}
	if runs[2].text != " base" || runs[2].color != base {
		t.Fatalf("reset run = %#v", runs[2])
	}
}

func TestNPCDialogWrapIgnoresColorCodeWidth(t *testing.T) {
	base := color.RGBA{R: 246, G: 242, B: 232, A: 255}
	lines := wrapNPCDialogLines([]string{"one ^00AAFFtwo^000000 three"}, 10)

	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2: %#v", len(lines), lines)
	}
	if got := npcDialogPlainText(lines[0]); got != "one two" {
		t.Fatalf("first line = %q", got)
	}
	if got := npcDialogPlainText(lines[1]); got != "three" {
		t.Fatalf("second line = %q", got)
	}
	if lines[0][1].text != "two" || lines[0][1].color == base {
		t.Fatalf("wrapped colored run not preserved: %#v", lines[0])
	}
}

func TestNPCDialogChoiceWindowOpensBelowDialogImmediately(t *testing.T) {
	dialog := NPCDialog{}
	ctx := Context{
		Input:   input.NewState(),
		ScreenW: 1280,
		ScreenH: 720,
	}
	dialog.Apply(network.NPCDialog{
		Kind:    network.NPCDialogMenu,
		NPCID:   100,
		Options: []string{"Prontera", "Geffen", "Payon", "Alberta"},
	})

	if !dialog.Update(ctx) {
		t.Fatal("dialog update did not open choice window")
	}

	expectedX, expectedY, _, _ := dialog.menuBounds(ctx.ScreenW, ctx.ScreenH, dialog.dialogWindow.x, dialog.dialogWindow.y, dialog.dialogWindow.width, dialog.dialogWindow.height)
	if dialog.menuWindow.x != expectedX || dialog.menuWindow.y != expectedY {
		t.Fatalf("choice position = %d,%d, want %d,%d", dialog.menuWindow.x, dialog.menuWindow.y, expectedX, expectedY)
	}
	if dialog.menuWindow.x == 0 && dialog.menuWindow.y == 0 {
		t.Fatal("choice window opened at origin")
	}
}

func npcDialogPlainText(runs []npcDialogTextRun) string {
	text := ""
	for _, run := range runs {
		text += run.text
	}
	return text
}
