package ui

import (
	"math"
	"strings"
	"testing"

	uiapp "github.com/gogpu/ui/app"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestCharacterEXPPanelWrapsBothRowsInRoundedGreyContainer(t *testing.T) {
	const width float32 = characterWindowWidth - 24
	panel := characterEXPPanel(session.Progress{
		BaseLevel:   42,
		JobLevel:    27,
		BaseExp:     25,
		NextBaseExp: 100,
		JobExp:      50,
		NextJobExp:  100,
	}, width)
	panel.Layout(widget.NewContext(), geometry.Constraints{MaxWidth: width, MaxHeight: 100})

	box, ok := panel.(*primitives.BoxWidget)
	if !ok {
		t.Fatalf("EXP panel = %T, want box", panel)
	}
	style := box.Style()
	if style.Radius != characterEXPPanelRadius {
		t.Fatalf("EXP panel radius = %.1f, want %.1f", style.Radius, characterEXPPanelRadius)
	}
	uitest.AssertColorEqual(t, style.Background, rotheme.Default.Colors.WindowFooter)
	if style.Border.Width != 0 {
		t.Fatalf("EXP panel border width = %.1f, want none", style.Border.Width)
	}

	rows := panel.Children()
	if len(rows) != 2 {
		t.Fatalf("EXP panel rows = %d, want Base and Job EXP", len(rows))
	}
	first := rows[0].(interface{ Bounds() geometry.Rect }).Bounds()
	second := rows[1].(interface{ Bounds() geometry.Rect }).Bounds()
	wantRowWidth := width - 2*characterEXPPanelPaddingX
	if first.Min.X != characterEXPPanelPaddingX || first.Min.Y != characterEXPPanelPaddingY || first.Width() != wantRowWidth {
		t.Fatalf("Base EXP row bounds = %v, want inset width %.1f", first, wantRowWidth)
	}
	if second.Min.X != characterEXPPanelPaddingX || second.Min.Y != first.Max.Y+characterEXPPanelGap || second.Width() != wantRowWidth {
		t.Fatalf("Job EXP row bounds = %v, want below Base EXP with %.1f gap", second, characterEXPPanelGap)
	}
	assertCharacterEXPRow := func(row widget.Widget, wantLabel string) {
		t.Helper()
		children := row.Children()
		if len(children) != 2 {
			t.Fatalf("%s row children = %d, want label and bar", wantLabel, len(children))
		}
		labelChildren := children[0].Children()
		if len(labelChildren) != 1 {
			t.Fatalf("%s label children = %d, want text", wantLabel, len(labelChildren))
		}
		text, ok := labelChildren[0].(interface{ Content() string })
		if !ok {
			t.Fatalf("EXP row label = %T, want text", labelChildren[0])
		}
		if text.Content() != wantLabel {
			t.Fatalf("EXP row label = %q, want %q", text.Content(), wantLabel)
		}
		labelBounds := children[0].(interface{ Bounds() geometry.Rect }).Bounds()
		barSlotBounds := children[1].(interface{ Bounds() geometry.Rect }).Bounds()
		if barSlotBounds.Min.X != labelBounds.Max.X+characterEXPLabelBarGap {
			t.Fatalf("%s bar x = %.1f, want %.1f after label", wantLabel, barSlotBounds.Min.X, labelBounds.Max.X+characterEXPLabelBarGap)
		}
		barChildren := children[1].Children()
		if len(barChildren) != 1 {
			t.Fatalf("%s bar slot children = %d, want bar", wantLabel, len(barChildren))
		}
		bar, ok := barChildren[0].(*characterBarWidget)
		if !ok {
			t.Fatalf("%s bar = %T, want character bar", wantLabel, barChildren[0])
		}
		uitest.AssertColorEqual(t, bar.background, widget.ColorWhite)
		barBounds := bar.Bounds()
		labelCenterY := labelBounds.Center().Y
		barCenterY := barSlotBounds.Min.Y + barBounds.Center().Y
		if math.Abs(float64(labelCenterY-barCenterY)) > 0.001 {
			t.Fatalf("%s vertical centers differ: label=%.3f bar=%.3f", wantLabel, labelCenterY, barCenterY)
		}
	}
	assertCharacterEXPRow(rows[0], "Base Lv. 42")
	assertCharacterEXPRow(rows[1], "Job Lv. 27")
	panelBounds := panel.(interface{ Bounds() geometry.Rect }).Bounds()
	if second.Max.Y+characterEXPPanelPaddingY != panelBounds.Max.Y {
		t.Fatalf("EXP panel bottom = %.1f, want %.1f after bottom inset", panelBounds.Max.Y, second.Max.Y+characterEXPPanelPaddingY)
	}
}

func TestCharacterWindowDoesNotDuplicateLevelOrEXPLabels(t *testing.T) {
	root := (&CharacterWindow{}).widgetTree(Context{Session: &session.Session{
		Progress: session.Progress{BaseLevel: 42, JobLevel: 27},
	}})
	var labels []string
	var walk func(widget.Widget)
	walk = func(current widget.Widget) {
		if text, ok := current.(interface{ Content() string }); ok {
			labels = append(labels, text.Content())
		}
		for _, child := range current.Children() {
			walk(child)
		}
	}
	walk(root)
	joined := strings.Join(labels, "\n")
	for _, label := range []string{"Base Lv. 42", "Job Lv. 27"} {
		if strings.Count(joined, label) != 1 {
			t.Fatalf("character window label %q appears %d times, want once", label, strings.Count(joined, label))
		}
	}
	if strings.Contains(joined, "Base EXP") || strings.Contains(joined, "Job EXP") {
		t.Fatalf("character window still contains EXP percentage labels: %q", joined)
	}
}

func TestCharacterWindowCompactToggleAndLiveValues(t *testing.T) {
	app := uiapp.New()
	bridge := basicMenuTestApp{app: app}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := Context{Input: input.NewState(), UIApp: bridge, UIManager: manager, ScreenW: 800, ScreenH: 600,
		Session: &session.Session{
			Selected: session.Character{ID: 1, Name: "Malki", Job: 4},
			Vitals:   session.Vitals{HP: 75, MaxHP: 100, SP: 30, MaxSP: 50},
			Progress: session.Progress{BaseLevel: 42, JobLevel: 27, BaseExp: 257, NextBaseExp: 1000},
		}}
	var character CharacterWindow
	var menu BasicMenu
	character.Update(ctx)
	menu.FollowCharacterWindow(ctx, &character)
	menu.Update(ctx, BasicMenuCallbacks{})
	menu.toggleCollapsed()
	menu.FollowCharacterWindow(ctx, &character)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	x := character.x + character.width - windowTitleButtonPadR - windowTitleButtonSize/2
	y := character.y + ROWindowTitleHeight/2
	ctx.Input.SetMousePosition(x, y)
	ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
	app.Window().HandleEvent(uitest.Click(float32(x), float32(y)))
	if !character.Update(ctx) || character.dragging {
		t.Fatal("header toggle click leaked to the map or started a drag")
	}
	ctx.Input.EndFrame()
	ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
	app.Window().HandleEvent(uitest.Release(float32(x), float32(y)))
	menu.FollowCharacterWindow(ctx, &character)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if !character.compact || character.height != characterWindowCompactHeight || character.width != characterWindowWidth {
		t.Fatalf("compact window dimensions = %dx%d, compact=%v", character.width, character.height, character.compact)
	}
	if !menu.collapsed || menu.y != character.y+character.height+basicMenuFollowGap {
		t.Fatal("compact mode changed the menu's state or detached it")
	}
	assertCharacterWindowCompactText(t, character.content, "Malki", "Lv. 42 / Acolyte / Lv. 27 / Exp. 25.7%", "HP 75 / 100", "SP 30 / 50")
	if app.Window().Context().FocusedWidget() != nil {
		t.Fatal("header toggle kept keyboard focus")
	}
	app.Window().HandleEvent(event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, event.ModNone))
	if !character.compact {
		t.Fatal("Enter toggled the character window again")
	}
	content, published := character.content, character.published
	character.Update(ctx)
	if character.content != content || character.published != published {
		t.Fatal("idle compact window rebuilt its widgets")
	}
	ctx.Session.Vitals.HP = 65
	ctx.Session.Progress.BaseExp = 500
	character.Update(ctx)
	assertCharacterWindowCompactText(t, character.content, "Malki", "Lv. 42 / Acolyte / Lv. 27 / Exp. 50.0%", "HP 65 / 100", "SP 30 / 50")
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	app.Window().HandleEvent(uitest.Click(float32(x), float32(y)))
	app.Window().HandleEvent(uitest.Release(float32(x), float32(y)))
	menu.FollowCharacterWindow(ctx, &character)
	if character.compact || character.height != characterWindowHeight || !menu.collapsed {
		t.Fatal("header toggle did not restore the full window independently of the menu")
	}
}

func assertCharacterWindowCompactText(t *testing.T, root widget.Widget, want ...string) {
	t.Helper()
	var texts []string
	var walk func(widget.Widget)
	walk = func(current widget.Widget) {
		if _, ok := current.(*characterBarWidget); ok {
			t.Error("compact view still contains a bar")
		}
		if text, ok := current.(interface{ Content() string }); ok {
			texts = append(texts, text.Content())
		}
		for _, child := range current.Children() {
			walk(child)
		}
	}
	walk(root)
	if got := strings.Join(texts, "\n"); got != strings.Join(want, "\n") {
		t.Fatalf("compact text = %q, want %q", got, strings.Join(want, "\n"))
	}
}

func TestCharacterWindowExpandingKeepsAttachedMenuOnScreen(t *testing.T) {
	for _, collapsed := range []bool{false, true} {
		ctx := Context{Input: input.NewState(), UIManager: NewManager(), Session: &session.Session{}, ScreenW: 800, ScreenH: 300}
		var character CharacterWindow
		var menu BasicMenu
		character.Update(ctx)
		menu.FollowCharacterWindow(ctx, &character)
		menu.Update(ctx, BasicMenuCallbacks{})
		if collapsed {
			menu.toggleCollapsed()
			menu.FollowCharacterWindow(ctx, &character)
		}
		character.ToggleCompact()
		bottomY := ctx.ScreenH - windowScreenMargin - character.height - character.dragBottom
		character.setPosition(ctx, character.x, bottomY)
		menu.FollowCharacterWindow(ctx, &character)
		character.ToggleCompact()
		menu.FollowCharacterWindow(ctx, &character)
		if character.y >= bottomY || menu.y+menu.height != ctx.ScreenH-windowScreenMargin {
			t.Fatalf("menu collapsed=%v: expanded group off screen: character y=%d, menu bottom=%d", collapsed, character.y, menu.y+menu.height)
		}
	}
}

func TestCharacterWindowRebindKeepsCompactStateAndOwnsToggle(t *testing.T) {
	app := uiapp.New()
	bridge := basicMenuTestApp{app: app}
	manager := NewManager()
	manager.SetUIApp(bridge)
	ctx := Context{Input: input.NewState(), UIApp: bridge, UIManager: manager, Session: &session.Session{}, ScreenW: 800, ScreenH: 600}
	var original CharacterWindow
	original.Update(ctx)
	original.ToggleCompact()
	carried := original
	carried.Rebind(ctx)
	app.Frame()
	app.Window().DrawTo(&uitest.MockCanvas{})
	if !carried.compact || carried.height != characterWindowCompactHeight {
		t.Fatal("map transition expanded the compact window")
	}
	x := float32(carried.x + carried.width - windowTitleButtonPadR - windowTitleButtonSize/2)
	y := float32(carried.y + ROWindowTitleHeight/2)
	app.Window().HandleEvent(uitest.Click(x, y))
	app.Window().HandleEvent(uitest.Release(x, y))
	if carried.compact || !original.compact {
		t.Fatal("carried toggle still affected the original window")
	}
}

func TestCharacterWindowToggleSurvivesUpdatesWhilePressed(t *testing.T) {
	for _, mode := range []string{"full", "compact"} {
		t.Run(mode, func(t *testing.T) {
			app := uiapp.New()
			bridge := basicMenuTestApp{app: app}
			manager := NewManager()
			manager.SetUIApp(bridge)
			ctx := Context{Input: input.NewState(), UIApp: bridge, UIManager: manager, ScreenW: 800, ScreenH: 600,
				Session: &session.Session{
					Selected: session.Character{ID: 1, Name: "Malki", Job: 4},
					Vitals:   session.Vitals{HP: 75, MaxHP: 100, SP: 30, MaxSP: 50},
					Progress: session.Progress{BaseLevel: 42, JobLevel: 27, BaseExp: 257, NextBaseExp: 1000},
				}}
			var character CharacterWindow
			character.Update(ctx)
			if mode == "compact" {
				character.ToggleCompact()
			}
			app.Frame()
			app.Window().DrawTo(&uitest.MockCanvas{})
			root, published := character.content, character.published
			header := root.Children()[0]
			wasCompact := character.compact
			wantHP := "65 / 100"
			if wasCompact {
				wantHP = "HP " + wantHP
			}
			x := character.x + character.width - windowTitleButtonPadR - windowTitleButtonSize/2
			y := character.y + ROWindowTitleHeight/2
			ctx.Input.SetMousePosition(x, y)
			ctx.Input.SetMouseButton(input.MouseButtonLeft, true)
			app.Window().HandleEvent(uitest.Click(float32(x), float32(y)))
			character.Update(ctx)
			ctx.Input.EndFrame()
			for _, update := range []func(){
				func() { ctx.Session.Vitals.HP = 65 },
				func() { ctx.Session.Vitals.SP = 25 },
				func() { ctx.Session.Progress.BaseExp = 500 },
				func() { ctx.Session.Progress.JobLevel = 28 },
				func() { ctx.Session.Inventory.Weight = 80 },
				func() { ctx.Session.Inventory.Zeny = 99 },
				func() { ctx.Session.Selected.Name = "Kivy" },
				func() { ctx.Session.Selected.Job = 5 },
			} {
				oldBody := character.body.child
				update()
				character.Update(ctx)
				if character.content != root || character.published != published || character.content.Children()[0] != header {
					t.Fatal("live update replaced the frame or header")
				}
				if oldBody.IsMounted() || !character.body.child.IsMounted() || character.body.child.Parent() != character.body {
					t.Fatal("body replacement did not preserve the widget lifecycle")
				}
				app.Frame()
				canvas := &uitest.MockCanvas{}
				app.Window().DrawTo(canvas)
				foundHP := false
				for _, text := range canvas.StyledTexts {
					if text.Text == wantHP {
						foundHP = true
					}
				}
				if !foundHP {
					t.Fatal("updated HP was not drawn while the toggle was pressed")
				}
			}
			wantTitle := "Kivy (Merchant)"
			if wasCompact {
				wantTitle = "Kivy"
			}
			if character.title.Get() != wantTitle {
				t.Fatalf("title = %q, want %q", character.title.Get(), wantTitle)
			}
			ctx.Input.SetMouseButton(input.MouseButtonLeft, false)
			app.Window().HandleEvent(uitest.Release(float32(x), float32(y)))
			if character.compact == wasCompact {
				t.Fatal("updates cancelled the pressed toggle")
			}
		})
	}
}
