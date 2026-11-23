package actors_test

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/effects"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestAppendActorDrawEntryUsesPathRenderDirection(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	actor := worldstate.Actor{
		X:            2,
		Y:            1,
		Dir:          4,
		Moving:       true,
		FromX:        0,
		FromY:        0,
		ToX:          2,
		ToY:          1,
		MoveStarted:  now.Add(-225 * time.Millisecond),
		MoveDuration: 450 * time.Millisecond,
		MovePath: []worldstate.WalkStep{
			{X: 0, Y: 0},
			{X: 0, Y: 1},
			{X: 1, Y: 1},
			{X: 2, Y: 1},
		},
	}
	projection := testutil.Projection(800, 600, 0.5, 1.5, 0)

	entries := actors.AppendActorDrawEntry(nil, world, projection, actor, false, now, 800, 600)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if got, want := entries[0].Actor.Dir, actors.DirectionFromDelta(0, 1, 1, 1, 4); got != want {
		t.Fatalf("entry direction = %d, want %d", got, want)
	}
}

func TestActorBillboardSortDepthUsesTopInCameraProjection(t *testing.T) {
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	footDepth := projection.Depth(10.5, 20.5, 0)
	topDepth := projection.Depth(10.5, 20.5, scene.ActorBillboardWorldHeightUnit)
	got := actors.ActorBillboardSortDepth(projection, 10.5, 20.5, 0)
	want := math.Min(footDepth, topDepth)
	if got != want {
		t.Fatalf("billboard depth = %.4f, want closer of foot %.4f and top %.4f", got, footDepth, topDepth)
	}
	if got >= footDepth {
		t.Fatalf("billboard depth = %.4f, want closer than foot depth %.4f", got, footDepth)
	}
}

func TestActorShadowFactorAveragesGNDLightmapAlpha(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 128
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 3, 3)
	want := float64(128) / 255
	if math.Abs(got.R-want) > 0.0001 || math.Abs(got.G-want) > 0.0001 || math.Abs(got.B-want) > 0.0001 {
		t.Fatalf("shadow factor = %+v, want gray %.4f", got, want)
	}
}

func TestActorShadowFactorAddsLightmapColorAndNormalizes(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 102
			lightmap.Color[y][x] = color.RGBA{R: 128, G: 64, A: 255}
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	// CalcLmIntensityFloatRGBAll: intensity + color/128, divided by the
	// brightest channel when it exceeds one.
	got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 3, 3)
	intensity := float64(102) / 255
	r, g, b := 1+intensity, 0.5+intensity, intensity
	want := [3]float64{r / r, g / r, b / r}
	if math.Abs(got.R-want[0]) > 0.0001 || math.Abs(got.G-want[1]) > 0.0001 || math.Abs(got.B-want[2]) > 0.0001 {
		t.Fatalf("shadow factor = %+v, want %v", got, want)
	}
}

func TestActorShadowFactorSamplesQuarterTexelsOfNeighbours(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 255
		}
	}
	// Even/even GAT cells read texel [1][1] of their GND cell; odd columns
	// read [1][5] and odd rows [5][1].
	lightmap.Alpha[1][1] = 0
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}

	got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 2, 2)
	// Only the centre sample (2, 2) reads the dark texel; its four neighbours
	// have an odd coordinate.
	want := float64(4*255) / 5 / 255
	if math.Abs(got.R-want) > 0.0001 {
		t.Fatalf("shadow factor = %+v, want gray %.4f", got, want)
	}
}

func TestActorShadowFactorIgnoresGroundShadowBelowElevatedGAT(t *testing.T) {
	var lightmap res.GNDLightmap
	for y := range lightmap.Alpha {
		for x := range lightmap.Alpha[y] {
			lightmap.Alpha[y][x] = 32
		}
	}
	gnd := &res.GND{
		Width:     4,
		Height:    4,
		Lightmaps: []res.GNDLightmap{lightmap},
		Surfaces:  []res.GNDSurface{{LightmapID: 0}},
		Cells:     make([]res.GNDCell, 16),
	}
	for i := range gnd.Cells {
		gnd.Cells[i].Top = 0
	}
	gat := &res.GAT{
		Width:  8,
		Height: 8,
		Cells:  make([]res.GATCell, 64),
	}
	for i := range gat.Cells {
		gat.Cells[i].Heights = [4]float32{2, 2, 2, 2}
	}

	got := actors.ActorShadowFactor(&worldstate.World{GAT: gat, GND: gnd}, 3, 3)
	if got != scene.FullLight {
		t.Fatalf("elevated GAT shadow factor = %+v, want full light", got)
	}
}

func TestActorShadowFactorDefaultsToLitWithoutGroundLightmap(t *testing.T) {
	if got := actors.ActorShadowFactor(nil, 3, 3); got != scene.FullLight {
		t.Fatalf("nil world shadow = %+v, want full light", got)
	}
	gnd := &res.GND{
		Width:    1,
		Height:   1,
		Surfaces: []res.GNDSurface{{LightmapID: 0}},
		Cells:    []res.GNDCell{{Top: -1}},
	}
	if got := actors.ActorShadowFactor(&worldstate.World{GND: gnd}, 0, 0); got != scene.FullLight {
		t.Fatalf("missing top surface shadow = %+v, want full light", got)
	}
}

func TestWarpActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: scene.ActorJobWarpPortal}
	x, y := actors.ActorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("warp anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestDisabledEffectsKeepVisibleWarpMarkers(t *testing.T) {
	ctx := testutil.ArcherEffectContext()
	ctx.World.Actors = map[uint32]worldstate.Actor{
		300: {ID: 300, Job: scene.ActorJobWarpPortal, X: 10, Y: 20},
		301: {ID: 301, Job: scene.ActorJobHiddenWarpNPC, X: 10, Y: 20},
	}
	m := &actors.System{ShadowView: testutil.EffectRenderTestSprite()}
	m.Bind(actors.Bindings{Effects: &effects.System{}})
	projection := testutil.Projection(800, 600, 10.5, 20.5, 0)
	for _, hidden := range []bool{false, true, false} {
		ctx.Session.NoEffects = hidden
		frame := render.NewFrame(800, 600)
		entries := m.CollectSceneActorEntries(frame, ctx, projection)
		if len(entries) != 2 {
			t.Fatalf("draw entries = %d, want player and visible portal only", len(entries))
		}
		for _, entry := range entries {
			if entry.Actor.ID != 300 {
				continue
			}
			if entry.CastShadow != hidden || hidden && entry.ShadowScale != 3 {
				t.Fatalf("hidden=%t: portal shadow=%t scale=%g", hidden, entry.CastShadow, entry.ShadowScale)
			}
			m.DrawActorShadowEntry(frame, ctx, projection, entry)
		}
		if drew := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() > 0; drew != hidden {
			t.Fatalf("hidden=%t: portal marker drawn=%t", hidden, drew)
		}
	}
}

func TestNormalActorUsesCenteredWorldAnchor(t *testing.T) {
	actor := worldstate.Actor{Job: 1002}
	x, y := actors.ActorWorldAnchor(actor, 30, 40)
	if x != 30.5 || y != 40.5 {
		t.Fatalf("normal actor anchor = %.1f, %.1f; want 30.5, 40.5", x, y)
	}
}

func TestActorOverlayLifeBarIsBelowNameLabel(t *testing.T) {
	nameY := actors.ActorNameLabelY(100, 1.2)
	barY := actors.ActorLifeBarY(100, 1.2)
	if barY <= nameY+10 {
		t.Fatalf("bar y = %.1f, name y = %.1f; want bar below name", barY, nameY)
	}
}

func TestActorOverlayCastBarIsBelowSpeechBubble(t *testing.T) {
	bubbleBottomY := actors.ActorSpeechBubbleBottomY(100, 1.2)
	barY := actors.ActorCastBarY(100, 1.2)
	if barY <= bubbleBottomY {
		t.Fatalf("cast bar y = %.1f, bubble bottom y = %.1f; want cast bar below bubble", barY, bubbleBottomY)
	}
}
