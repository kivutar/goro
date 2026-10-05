package ui

import (
	"image/color"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/uitest"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
)

func TestNPCDialogMenuDoubleClickConfirmsClickedEntry(t *testing.T) {
	ctx, _, app := newWindowInstanceTest()
	client, server := newIdentifyTestConnection(t)
	ctx.Network = client
	var dialog NPCDialog
	openMenu := func() {
		dialog.Apply(network.NPCDialog{Kind: network.NPCDialogMenu, NPCID: 100,
			Options: []string{"Prontera", "Geffen"}})
		dialog.Update(ctx)
		app.Frame()
		app.Window().DrawTo(&uitest.MockCanvas{})
	}
	clickRow := func(row int) {
		x := float32(dialog.menuWindow.x + npcMenuPad + 10)
		y := float32(dialog.menuWindow.y + ROWindowTitleHeight + npcMenuPad + row*npcMenuRowH + npcMenuRowH/2)
		app.HandleEvent(uitest.Click(x, y))
		app.HandleEvent(uitest.Release(x, y))
	}

	openMenu()
	clickRow(0)
	assertNoIdentifyTestPackets(t, client, server)

	// Replacing the menu must discard the previous menu's first click.
	openMenu()
	clickRow(0)
	assertNoIdentifyTestPackets(t, client, server)
	clickRow(1)
	if dialog.menuRow != 1 {
		t.Fatalf("clicked row = %d, want 1", dialog.menuRow)
	}
	assertNoIdentifyTestPackets(t, client, server)

	// The mouse's second click confirms its row, even if keyboard navigation
	// changed selection in between the two clicks.
	app.HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyUp, 0, event.ModNone))
	if dialog.menuRow != 0 {
		t.Fatalf("keyboard selection = %d, want 0", dialog.menuRow)
	}
	clickRow(1)
	readIdentifyTestPacket(t, server, network.BuildNPCMenuChoicePacket(100, 2))
	if dialog.action != npcDialogActionNone || dialog.menuWindow.IsOpen() {
		t.Fatal("confirmed NPC menu stayed active")
	}
	assertNoIdentifyTestPackets(t, client, server)

	// A following menu requires its own double click.
	openMenu()
	clickRow(1)
	assertNoIdentifyTestPackets(t, client, server)
	clickRow(1)
	readIdentifyTestPacket(t, server, network.BuildNPCMenuChoicePacket(100, 2))
	assertNoIdentifyTestPackets(t, client, server)
}

func TestNPCDialogControllerSelectionScrollsAndClamps(t *testing.T) {
	dialog := NPCDialog{}
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogMenu, NPCID: 100,
		Options: []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven"}})
	for i := 0; i < 10; i++ {
		dialog.Control(Context{}, "down")
	}
	if dialog.menuRow != 6 || dialog.menuScrollY.Get() != 2*npcMenuRowH {
		t.Fatalf("last choice not selected and visible: row=%d scroll=%v", dialog.menuRow, dialog.menuScrollY.Get())
	}
	for i := 0; i < 10; i++ {
		dialog.Control(Context{}, "up")
	}
	if dialog.menuRow != 0 || dialog.menuScrollY.Get() != 0 {
		t.Fatal("first choice not selected and visible")
	}
}

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

func TestNPCDialogTextSegmentsAdvanceByMeasuredRuns(t *testing.T) {
	base := color.RGBA{R: 246, G: 242, B: 232, A: 255}
	red := color.RGBA{R: 255, A: 255}
	runs := []npcDialogTextRun{
		{text: "hello ", color: base},
		{text: "red", color: red},
		{text: " again", color: base},
	}

	segments := npcDialogTextSegments(runs, func(text string) float32 {
		return float32(len([]rune(text)) * 10)
	})

	if len(segments) != 3 {
		t.Fatalf("segment count = %d, want 3: %#v", len(segments), segments)
	}
	if segments[0].x != 0 || segments[0].width != 60 {
		t.Fatalf("first segment = %#v, want x=0 width=60", segments[0])
	}
	if segments[1].x != 60 || segments[1].width != 30 || segments[1].color != red {
		t.Fatalf("colored segment = %#v, want x=60 width=30 red", segments[1])
	}
	if segments[2].x != 90 || segments[2].width != 60 {
		t.Fatalf("final segment = %#v, want x=90 width=60", segments[2])
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

func TestNPCDialogMenuStartsWithFirstSelection(t *testing.T) {
	dialog := NPCDialog{}
	dialog.menuRow = 2
	dialog.ensureMenuScrollSignal().Set(48)

	dialog.Apply(network.NPCDialog{
		Kind:    network.NPCDialogMenu,
		NPCID:   100,
		Options: []string{"Prontera", "Geffen"},
	})

	if dialog.menuRow != 0 {
		t.Fatalf("menu row = %d, want first selection", dialog.menuRow)
	}
	if scroll := dialog.ensureMenuScrollSignal().Get(); scroll != 0 {
		t.Fatalf("menu scroll = %.1f, want 0", scroll)
	}
}

func TestNPCDialogEmptyMenuStartsWithoutSelection(t *testing.T) {
	dialog := NPCDialog{}

	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogMenu, NPCID: 100})

	if dialog.menuRow != -1 {
		t.Fatalf("menu row = %d, want no selection for empty menu", dialog.menuRow)
	}
}

func TestNPCDialogMenuChoiceRequiresSelection(t *testing.T) {
	dialog := NPCDialog{
		open:    true,
		npcID:   100,
		action:  npcDialogActionMenu,
		options: []string{"Prontera", "Geffen"},
		menuRow: -1,
	}

	dialog.chooseSelected(Context{})

	if dialog.status != "" {
		t.Fatalf("status = %q, want no submit attempt without selection", dialog.status)
	}
	dialog.menuRow = 1
	dialog.chooseSelected(Context{})
	if dialog.status != "not connected" {
		t.Fatalf("status = %q, want submit attempt after selection", dialog.status)
	}
}

func TestNPCDialogIgnoresInitialEmptySay(t *testing.T) {
	dialog := NPCDialog{}
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: ""})

	if dialog.open {
		t.Fatalf("empty initial say opened dialog: %+v", dialog)
	}
}

func TestNPCDialogCloseHandlerRunsOnceForOpenDialog(t *testing.T) {
	dialog := NPCDialog{}
	closed := 0
	dialog.SetCloseHandler(func() { closed++ })
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	dialog.Reset()
	if closed != 1 {
		t.Fatalf("close handler calls = %d, want 1", closed)
	}
	dialog.Reset()
	if closed != 1 {
		t.Fatalf("closed dialog invoked handler again: %d", closed)
	}
}

func TestNPCDialogEscapeDoesNotCloseDialogWaitingForNext(t *testing.T) {
	inputState := input.NewState()
	ctx := Context{
		Input:   inputState,
		ScreenW: 1280,
		ScreenH: 720,
	}
	dialog := NPCDialog{}
	closed := 0
	dialog.SetCloseHandler(func() { closed++ })
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Hello"})
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogNext, NPCID: 100})
	dialog.Update(ctx) // Publish the dialog before processing keyboard input.

	inputState.SetKey(input.KeyEscape, true)
	if !dialog.Update(ctx) {
		t.Fatal("open NPC dialog did not consume Escape")
	}
	if !dialog.IsOpen() {
		t.Fatal("Escape closed a dialog waiting for Next")
	}
	if dialog.action != npcDialogActionNext {
		t.Fatalf("dialog action = %d, want Next", dialog.action)
	}
	if closed != 0 {
		t.Fatalf("close handler calls = %d, want 0", closed)
	}
}

func TestNPCDialogEscapeClosesDialogWaitingForClose(t *testing.T) {
	inputState := input.NewState()
	ctx := Context{
		Input:   inputState,
		ScreenW: 1280,
		ScreenH: 720,
	}
	dialog := NPCDialog{}
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogSay, NPCID: 100, Message: "Goodbye"})
	dialog.Apply(network.NPCDialog{Kind: network.NPCDialogClose, NPCID: 100})
	dialog.Update(ctx)

	inputState.SetKey(input.KeyEscape, true)
	if !dialog.Update(ctx) {
		t.Fatal("dialog waiting for Close did not consume Escape")
	}
	if dialog.IsOpen() {
		t.Fatal("Escape left a dialog waiting for Close open")
	}
}

func npcDialogPlainText(runs []npcDialogTextRun) string {
	text := ""
	for _, run := range runs {
		text += run.text
	}
	return text
}
