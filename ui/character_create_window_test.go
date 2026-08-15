package ui

import (
	"math"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/ui/rotheme"
)

func TestCharacterSelectPage(t *testing.T) {
	if got := CharacterSelectPage(5); got != 1 {
		t.Fatalf("page = %d, want 1", got)
	}
}

func TestCharacterSelectFooterButtonStateForSelectedSlot(t *testing.T) {
	opts := CharacterSelectWindowOptions{
		SelectedSlot: 1,
		Characters: []session.Character{
			{ID: 10, Slot: 1},
		},
	}
	if characterSelectDeleteDisabled(opts) {
		t.Fatal("delete disabled for occupied slot")
	}
	if !characterSelectMakeDisabled(opts) {
		t.Fatal("make enabled for occupied slot")
	}

	opts.SelectedSlot = 2
	if !characterSelectDeleteDisabled(opts) {
		t.Fatal("delete enabled for empty slot")
	}
	if characterSelectMakeDisabled(opts) {
		t.Fatal("make disabled for empty slot")
	}
}

func TestCharacterSelectStatsUseTwoEqualColumnsAndSixRows(t *testing.T) {
	if characterSelectInfoTableGap != int(rotheme.TableGap) {
		t.Fatalf("character info table gap = %d, want theme gap %.1f", characterSelectInfoTableGap, rotheme.TableGap)
	}
	rows := characterSelectStatsRows(session.Character{
		Name: "Alice", Exp: 123456,
		Str: 1, Agi: 2, Vit: 3, Int: 4, Dex: 5, Luk: 6,
	})
	if len(rows) != characterSelectInfoRowCount {
		t.Fatalf("character info rows = %d, want %d", len(rows), characterSelectInfoRowCount)
	}
	wantLeftLabels := [...]string{"Name", "Job", "Level", "Exp", "HP", "SP"}
	wantRightLabels := [...]string{"STR", "AGI", "VIT", "INT", "DEX", "LUK"}
	for i, row := range rows {
		if len(row) != 4 {
			t.Fatalf("character info row %d cells = %d, want 4", i, len(row))
		}
		if row[0].Text != wantLeftLabels[i] || row[2].Text != wantRightLabels[i] {
			t.Fatalf("character info row %d labels = %q, %q", i, row[0].Text, row[2].Text)
		}
		leftWidth := row[0].Width + rotheme.TableGap + row[1].Width
		rightWidth := row[2].Width + rotheme.TableGap + row[3].Width
		if leftWidth != rightWidth || leftWidth != characterSelectInfoColumnW {
			t.Fatalf("character info row %d column widths = %.1f, %.1f, want %.1f each", i, leftWidth, rightWidth, characterSelectInfoColumnW)
		}
	}
	if rows[3][1].Text != "123456" {
		t.Fatalf("experience value = %q, want 123456", rows[3][1].Text)
	}
	if characterSelectInfoPanelH != 105 || characterSelectWindowH != 373 {
		t.Fatalf("character select heights = panel %d, window %d; want 105 and 373", characterSelectInfoPanelH, characterSelectWindowH)
	}
}

func TestCharacterSelectInfoPanelFitsSixRowsAndPreservesMargins(t *testing.T) {
	tree := (&CharacterSelectWindow{opts: CharacterSelectWindowOptions{
		SelectedSlot: 0,
		MaxSlots:     9,
		Characters:   []session.Character{{ID: 10, Slot: 0}},
	}}).widgetTree()
	tree.Layout(
		widget.NewContext(),
		geometry.Tight(geometry.Sz(characterSelectWindowW, characterSelectWindowH)),
	)

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("character-select window content tree is incomplete")
	}
	content := windowChildren[1].Children()[0]
	contentChildren := content.Children()
	if len(contentChildren) != 3 {
		t.Fatalf("character-select content children = %d, want 3", len(contentChildren))
	}
	panel := contentChildren[2]
	panelBounds := panel.(interface{ Bounds() geometry.Rect }).Bounds()
	if math.Abs(float64(panelBounds.Width()-characterSelectInfoPanelW)) > 0.001 || math.Abs(float64(panelBounds.Height()-characterSelectInfoPanelH)) > 0.001 {
		t.Fatalf("character info panel size = %.1fx%.1f, want %dx%d", panelBounds.Width(), panelBounds.Height(), characterSelectInfoPanelW, characterSelectInfoPanelH)
	}
	if len(panel.Children()) != 1 {
		t.Fatal("character info table is missing")
	}
	tableBounds := panel.Children()[0].(interface{ Bounds() geometry.Rect }).Bounds()
	if math.Abs(float64(tableBounds.Min.X-characterSelectInfoPanelPad)) > 0.001 || math.Abs(float64(tableBounds.Min.Y-characterSelectInfoPanelPad)) > 0.001 ||
		math.Abs(float64(tableBounds.Width()-(characterSelectInfoPanelW-2*characterSelectInfoPanelPad))) > 0.001 || math.Abs(float64(tableBounds.Height()-characterSelectInfoTableH)) > 0.001 {
		t.Fatalf("character info table bounds = %v, want padded %dpx inside panel", tableBounds, characterSelectInfoPanelPad)
	}
	contentBounds := content.(interface{ Bounds() geometry.Rect }).Bounds()
	if bottomMargin := contentBounds.Height() - panelBounds.Max.Y; math.Abs(float64(bottomMargin-10.8)) > 0.001 {
		t.Fatalf("character info bottom margin = %.1f, want preserved margin 10.8", bottomMargin)
	}
}

func TestCharacterSelectArrowHitboxesKeepFullWidth(t *testing.T) {
	tree := (&CharacterSelectWindow{opts: CharacterSelectWindowOptions{MaxSlots: 9}}).widgetTree()
	tree.Layout(
		widget.NewContext(),
		geometry.Tight(geometry.Sz(characterSelectWindowW, characterSelectWindowH)),
	)

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("character-select window content tree is incomplete")
	}
	content := windowChildren[1].Children()[0]
	if len(content.Children()) == 0 {
		t.Fatal("character-select window has no content row")
	}
	rowChildren := content.Children()[0].Children()
	if len(rowChildren) != 3 {
		t.Fatalf("character-select row children = %d, want 3", len(rowChildren))
	}

	for i, arrow := range []widget.Widget{rowChildren[0], rowChildren[2]} {
		bounder, ok := arrow.(interface{ Bounds() geometry.Rect })
		if !ok {
			t.Fatalf("arrow %d does not expose bounds", i)
		}
		if got := bounder.Bounds().Width(); got != rotheme.IconButtonSize {
			t.Fatalf("arrow %d hitbox width = %.1f, want %.1f", i, got, rotheme.IconButtonSize)
		}
	}
}

func TestCharacterCreateHairControlsFrameHead(t *testing.T) {
	bounds := geometry.NewRect(13, 21, characterCreatePanelW, characterCreatePanelH)
	left := characterCreatePreviewButtonRect(bounds, 0)
	color := characterCreatePreviewButtonRect(bounds, 1)
	right := characterCreatePreviewButtonRect(bounds, 2)

	if left.Min.Y != right.Min.Y {
		t.Fatalf("hair style arrow heights differ: left %.1f, right %.1f", left.Min.Y, right.Min.Y)
	}
	if color.Min.Y < bounds.Min.Y || color.Max.Y >= left.Min.Y {
		t.Fatalf("hair color arrow bounds = %v, want above hair style arrows at %.1f", color, left.Min.Y)
	}
	if left.Min.Y >= bounds.Min.Y+bounds.Height()/3 {
		t.Fatalf("hair style arrows start at %.1f, want within preview's upper third", left.Min.Y)
	}
}

func TestCharacterCreateNameUsesLabelStyle(t *testing.T) {
	tree := (&CharacterCreateWindow{}).widgetTree()
	var name *primitives.TextWidget
	var walk func(widget.Widget)
	walk = func(current widget.Widget) {
		if text, ok := current.(*primitives.TextWidget); ok && text.Content() == "Name" {
			name = text
		}
		for _, child := range current.Children() {
			walk(child)
		}
	}
	walk(tree)
	if name == nil {
		t.Fatal("character creation Name label is missing")
	}
	style := name.Style()
	if style.Color != rotheme.Default.Colors.LabelText || !style.Bold || style.FontFamily != "" {
		t.Fatalf("Name style = color %+v, bold %t, family %q; want semantic label style", style.Color, style.Bold, style.FontFamily)
	}
}

func TestCharacterCreateStatButtonsSitOutsideHexagon(t *testing.T) {
	tree := (&CharacterCreateWindow{}).widgetTree()
	tree.Layout(
		widget.NewContext(),
		geometry.Tight(geometry.Sz(characterCreateWindowW, characterCreateWindowH)),
	)

	var findGraph func(widget.Widget) *characterCreateStatGraph
	findGraph = func(current widget.Widget) *characterCreateStatGraph {
		if graph, ok := current.(*characterCreateStatGraph); ok {
			return graph
		}
		for _, child := range current.Children() {
			if graph := findGraph(child); graph != nil {
				return graph
			}
		}
		return nil
	}
	graph := findGraph(tree)
	if graph == nil {
		t.Fatal("character creation stat graph is missing from widget tree")
	}
	bounds := graph.Bounds()
	if bounds.Width() != characterCreateGraphW || bounds.Height() != characterCreateGraphH {
		t.Fatalf("stat graph size = %.1fx%.1f, want %dx%d", bounds.Width(), bounds.Height(), characterCreateGraphW, characterCreateGraphH)
	}
	cx := bounds.Min.X + bounds.Width()/2
	cy := bounds.Min.Y + bounds.Height()/2
	hexagon := characterCreateWidgetGraphPoints(cx, cy, characterCreateGraphOuterRadius)

	for stat := 0; stat < CharacterCreateStatCount; stat++ {
		rect := characterCreateStatButtonRect(bounds, stat)
		if rect.Min.X < bounds.Min.X || rect.Min.Y < bounds.Min.Y || rect.Max.X > bounds.Max.X || rect.Max.Y > bounds.Max.Y {
			t.Fatalf("stat %d button %v exceeds graph bounds %v", stat, rect, bounds)
		}

		buttonX := rect.Min.X + rect.Width()/2
		buttonY := rect.Min.Y + rect.Height()/2
		buttonDistance := math.Hypot(float64(buttonX-cx), float64(buttonY-cy))
		hexagonDistance := math.Hypot(float64(hexagon[stat].X-cx), float64(hexagon[stat].Y-cy))
		if buttonDistance <= hexagonDistance {
			t.Fatalf("stat %d button distance = %.1f, want greater than hexagon radius %.1f", stat, buttonDistance, hexagonDistance)
		}
	}
}

func TestCharacterCreateStatButtonsUseLabelStyle(t *testing.T) {
	graph := newCharacterCreateStatGraph([CharacterCreateStatCount]uint8{}, nil)
	graph.Layout(widget.NewContext(), geometry.Tight(geometry.Sz(characterCreateGraphW, characterCreateGraphH)))
	canvas := &uitest.MockCanvas{}
	graph.Draw(widget.NewContext(), canvas)

	if len(canvas.Texts) != CharacterCreateStatCount {
		t.Fatalf("stat button label draws = %d, want %d", len(canvas.Texts), CharacterCreateStatCount)
	}
	labels := CharacterCreateStatLabels()
	for stat, draw := range canvas.Texts {
		if draw.Text != labels[stat] {
			t.Fatalf("stat button %d label = %q, want %q", stat, draw.Text, labels[stat])
		}
		if draw.Color != rotheme.Default.Colors.LabelText || !draw.Bold || draw.Align != widget.TextAlignCenter {
			t.Fatalf("stat button %s style = color %+v, bold %t, align %v", draw.Text, draw.Color, draw.Bold, draw.Align)
		}
	}
	if len(canvas.StyledTexts) != 0 {
		t.Fatal("stat button labels unexpectedly used the custom DejaVu family")
	}
}

func TestCharacterCreatePreviewCenterAlignsWithStatGraph(t *testing.T) {
	window := &CharacterCreateWindow{}
	tree := window.widgetTree()
	tree.Layout(
		widget.NewContext(),
		geometry.Tight(geometry.Sz(characterCreateWindowW, characterCreateWindowH)),
	)

	var preview *characterCreatePreview
	var graph *characterCreateStatGraph
	var walk func(widget.Widget)
	walk = func(current widget.Widget) {
		if currentPreview, ok := current.(*characterCreatePreview); ok {
			preview = currentPreview
		}
		if currentGraph, ok := current.(*characterCreateStatGraph); ok {
			graph = currentGraph
		}
		for _, child := range current.Children() {
			walk(child)
		}
	}
	walk(tree)
	if preview == nil || graph == nil {
		t.Fatalf("character creation widgets missing: preview=%t graph=%t", preview != nil, graph != nil)
	}

	previewBounds := preview.Bounds()
	graphBounds := graph.Bounds()
	previewCenterY := previewBounds.Min.Y + previewBounds.Height()/2
	graphCenterY := graphBounds.Min.Y + graphBounds.Height()/2
	if previewCenterY != graphCenterY {
		t.Fatalf("preview center Y = %.1f, want graph center Y %.1f", previewCenterY, graphCenterY)
	}

	nameBlockY := characterCreatePreviewTopPad + characterCreatePanelH - characterCreateNameLift
	wantNameBlockY := characterCreateNameColumnTopPad + characterCreatePanelH
	if nameBlockY != wantNameBlockY {
		t.Fatalf("name block Y = %d, want restored position %d", nameBlockY, wantNameBlockY)
	}
}

func TestCharacterCreateStatFillIsCenteredOnGraph(t *testing.T) {
	graphBounds := geometry.NewRect(13, 21, characterCreateGraphW, characterCreateGraphH)
	cx := graphBounds.Min.X + graphBounds.Width()/2
	cy := graphBounds.Min.Y + graphBounds.Height()/2
	fillBounds := characterCreateStatFillBounds(cx, cy, characterCreateGraphOuterRadius)

	fillCenterX := fillBounds.Min.X + fillBounds.Width()/2
	fillCenterY := fillBounds.Min.Y + fillBounds.Height()/2
	if fillCenterX != cx || fillCenterY != cy {
		t.Fatalf("stat fill center = %.1f,%.1f, want graph center %.1f,%.1f", fillCenterX, fillCenterY, cx, cy)
	}
	if fillBounds.Width() != fillBounds.Height() {
		t.Fatalf("stat fill bounds = %v, want square SVG viewport", fillBounds)
	}
}

func TestCharacterCreateStatTableUsesLabelAndValueCellStyles(t *testing.T) {
	if characterCreateStatLabelW >= characterCreateStatValueW {
		t.Fatalf("stat label width = %d, want less than value width %d", characterCreateStatLabelW, characterCreateStatValueW)
	}
	rows := characterCreateStatRows([CharacterCreateStatCount]uint8{9, 8, 7, 6, 5, 4})
	if len(rows) != CharacterCreateStatCount {
		t.Fatalf("stat table rows = %d, want %d", len(rows), CharacterCreateStatCount)
	}
	for stat, row := range rows {
		if len(row) != 2 {
			t.Fatalf("stat %d cells = %d, want 2", stat, len(row))
		}
		if !row[0].Head {
			t.Fatalf("stat %d label cell does not use header background", stat)
		}
		if row[1].Head {
			t.Fatalf("stat %d value cell uses header background", stat)
		}
		if row[1].Align != widget.TextAlignRight {
			t.Fatalf("stat %d value alignment = %v, want right", stat, row[1].Align)
		}
	}
}

func TestCharacterCreateStatTablePanelFitsTable(t *testing.T) {
	panel := characterCreateStatTablePanel([CharacterCreateStatCount]uint8{})
	size := panel.Layout(widget.NewContext(), geometry.Loose(geometry.Sz(1000, 1000)))
	wantWidth := characterCreateStatLabelW + characterCreateStatValueW + rotheme.TableGap + 2*characterCreateStatPanelPad
	wantHeight := CharacterCreateStatCount*characterCreateStatRowH + (CharacterCreateStatCount-1)*rotheme.TableGap + 2*characterCreateStatPanelPad
	if wantWidth != 132 {
		t.Fatalf("stat table panel configured width = %.1f, want preserved width 132", wantWidth)
	}
	if size.Width != wantWidth || size.Height != wantHeight {
		t.Fatalf("stat table panel size = %.1fx%.1f, want %.1fx%.1f", size.Width, size.Height, wantWidth, wantHeight)
	}
}

func TestCharacterCreateStatTablePanelAlignsTopRight(t *testing.T) {
	tree := (&CharacterCreateWindow{}).widgetTree()
	tree.Layout(
		widget.NewContext(),
		geometry.Tight(geometry.Sz(characterCreateWindowW, characterCreateWindowH)),
	)

	windowChildren := tree.Children()
	if len(windowChildren) < 2 || len(windowChildren[1].Children()) != 1 {
		t.Fatal("character creation window content tree is incomplete")
	}
	content := windowChildren[1].Children()[0]
	if len(content.Children()) != 1 {
		t.Fatal("character creation content row is missing")
	}
	row := content.Children()[0]
	rowChildren := row.Children()
	if len(rowChildren) != 3 {
		t.Fatalf("character creation row children = %d, want 3", len(rowChildren))
	}
	panelBounds := rowChildren[2].(interface{ Bounds() geometry.Rect }).Bounds()
	rowBounds := row.(interface{ Bounds() geometry.Rect }).Bounds()
	if panelBounds.Min.Y != 0 {
		t.Fatalf("stat table panel Y = %.1f, want top-aligned at 0", panelBounds.Min.Y)
	}
	if panelBounds.Max.X != rowBounds.Width() {
		t.Fatalf("stat table panel right = %.1f, want row right %.1f", panelBounds.Max.X, rowBounds.Width())
	}
}

func TestCharacterCreateGraphDrawOrderIsValidHexagon(t *testing.T) {
	points := CharacterCreateGraphPoints(0, 0, 64)
	order := CharacterCreateGraphDrawOrder()
	seen := map[int]bool{}
	for _, stat := range order {
		if stat < 0 || stat >= CharacterCreateStatCount {
			t.Fatalf("stat index outside range in graph order: %d", stat)
		}
		if seen[stat] {
			t.Fatalf("duplicate stat index in graph order: %d", stat)
		}
		seen[stat] = true
	}

	if points[CharacterCreateStatDex][0] >= 0 || points[CharacterCreateStatDex][1] <= 0 {
		t.Fatalf("DEX graph point = %#v, want lower-left", points[CharacterCreateStatDex])
	}
	if points[CharacterCreateStatLuk][0] <= 0 || points[CharacterCreateStatLuk][1] <= 0 {
		t.Fatalf("LUK graph point = %#v, want lower-right", points[CharacterCreateStatLuk])
	}

	for i := 0; i < CharacterCreateStatCount; i++ {
		a1 := points[order[i]]
		a2 := points[order[(i+1)%CharacterCreateStatCount]]
		for j := i + 1; j < CharacterCreateStatCount; j++ {
			if graphEdgesAdjacent(i, j) {
				continue
			}
			b1 := points[order[j]]
			b2 := points[order[(j+1)%CharacterCreateStatCount]]
			if graphSegmentsCross(a1, a2, b1, b2) {
				t.Fatalf("graph edges %d and %d cross", i, j)
			}
		}
	}
}

func graphEdgesAdjacent(a, b int) bool {
	if a == b {
		return true
	}
	if a+1 == b || b+1 == a {
		return true
	}
	return (a == 0 && b == CharacterCreateStatCount-1) || (b == 0 && a == CharacterCreateStatCount-1)
}

func graphSegmentsCross(a1, a2, b1, b2 [2]float64) bool {
	o1 := graphOrientation(a1, a2, b1)
	o2 := graphOrientation(a1, a2, b2)
	o3 := graphOrientation(b1, b2, a1)
	o4 := graphOrientation(b1, b2, a2)
	return o1*o2 < 0 && o3*o4 < 0
}

func graphOrientation(a, b, c [2]float64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
}
