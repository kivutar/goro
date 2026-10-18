package game

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func effectRenderTestSprite() *spriteView {
	return &spriteView{
		spr: &res.SPR{},
		act: &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{}}}}},
		billboards: map[singleSpriteBillboardKey]*spriteBillboard{
			{}: {image: render.NewImage(8, 8), anchorX: 4, anchorY: 8},
		},
	}
}

func TestWorldDrawPlacesEffectsBeforeAndAfterEntities(t *testing.T) {
	const effectID = -1
	db.EffectSpecs[effectID] = db.EffectSpec{Components: []db.EffectComponent{
		{Kind: db.EffectComponentSPR, SpriteFile: "after", Duration: time.Hour},
		{Kind: db.EffectComponentSPR, SpriteFile: "before", Duration: time.Hour, RenderBefore: true},
	}}
	t.Cleanup(func() { delete(db.EffectSpecs, effectID) })
	for _, terrain := range []string{"gat", "gnd", "models"} {
		t.Run(terrain, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 42, X: 1, Y: 1}
			world.Actors[100] = worldstate.Actor{ID: 100, Job: 1002, X: 1, Y: 1}
			world.GAT = &res.GAT{Width: 4, Height: 4, Cells: make([]res.GATCell, 16)}
			if terrain != "gat" {
				world.GND = &res.GND{Width: 2, Height: 2, Cells: make([]res.GNDCell, 4)}
			}
			if terrain == "models" {
				world.RSW = &res.RSW{}
				world.RSM = map[string]*res.RSM{"empty": {}}
			}
			ctx := client.Context{World: world, Resources: &res.Manager{}, Session: &session.Session{CharID: 42}}
			ctx.Config.Render.NoUI = true
			before, actor, after := effectRenderTestSprite(), effectRenderTestSprite(), effectRenderTestSprite()
			starts := time.Now().Add(-time.Second)
			mode := &WorldMode{
				effectViews: map[string]*spriteView{"before": before, "after": after},
				nonPCViews:  map[int]*spriteView{1002: actor},
				worldEffects: []worldEffect{
					{effectID: effectID, x: 1, y: 1, starts: starts, expires: starts.Add(time.Hour)},
					{effectID: effectID, x: 1, y: 1, starts: starts, expires: starts},
					{effectID: effectID, x: 1, y: 1, starts: starts.Add(time.Hour)},
				},
			}
			frame := render.NewFrame(800, 600)
			mode.Draw(ctx, frame)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if commands.Len() != 3 {
				t.Fatalf("draws = %d, want one effect behind, one actor, one effect in front", commands.Len())
			}
			for i, view := range []*spriteView{before, actor, after} {
				want := reflect.ValueOf(view.billboards[singleSpriteBillboardKey{}].image).Pointer()
				if commands.Index(i).FieldByName("Texture").Pointer() != want {
					t.Fatalf("draw %d has wrong texture/order", i)
				}
			}
			if len(mode.worldEffects) != 2 {
				t.Fatal("expired effects should be removed and scheduled effects retained")
			}
		})
	}
}

func TestEffectShadowFollowsTerrainAtMovingPosition(t *testing.T) {
	world := worldstate.New()
	world.GAT = &res.GAT{Width: 3, Height: 1, Cells: []res.GATCell{
		{}, {Heights: [4]float32{6, 6, 6, 6}}, {},
	}}
	ctx := client.Context{World: world, Resources: &res.Manager{}}
	mode := &WorldMode{effectViews: map[string]*spriteView{"shadow": effectRenderTestSprite()}}
	projection := newSceneProjectionForTarget(800, 600, 1.5, .5, 0)
	starts := time.Unix(100, 0)
	for _, shadow := range []bool{false, true} {
		component := worldEffectComponent{
			spriteFile: "shadow", shadowTexture: shadow, duration: time.Second,
			posXEnd: 2, posZ: 4, sizeStart: 1, sizeEnd: 1,
		}
		frame := render.NewFrame(800, 600)
		mode.draw3DEffect(frame, ctx, projection, worldEffect{starts: starts}, component, 0, .5, .5, 10, starts.Add(500*time.Millisecond))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 1 {
			t.Fatalf("shadow=%t: draws = %d, want 1", shadow, commands.Len())
		}
		center := commands.Index(0).FieldByName("Center")
		wantZ := 14.0
		if shadow {
			wantZ = 6 + actorShadowTerrainLift
		}
		if math.Abs(center.Index(0).Float()-1.5) > 1e-5 || math.Abs(center.Index(1).Float()-wantZ) > 1e-5 {
			t.Fatalf("shadow=%t: center=%v, want x=1.5 height=%g", shadow, center, wantZ)
		}
	}
}

func TestWorldEffectDrawReusesConvertedSpecs(t *testing.T) {
	ctx := client.Context{World: worldstate.New(), Resources: &res.Manager{}}
	now := time.Unix(100, 0)
	emotion := effectRenderTestSprite()
	emotion.billboards[singleSpriteBillboardKey{anchorY: -100}] = emotion.billboards[singleSpriteBillboardKey{}]
	mode := &WorldMode{
		effectViews:  map[string]*spriteView{"emotion": emotion},
		worldEffects: []worldEffect{{effectID: effectEmotion, starts: now}},
	}
	frame := render.NewFrame(800, 600)
	projection := newSceneProjectionForTarget(800, 600, .5, .5, 0)
	allocations := testing.AllocsPerRun(10, func() {
		frame.BeginFrame()
		mode.drawWorldEffects(frame, ctx, projection, now, true)
		mode.drawWorldEffects(frame, ctx, projection, now, false)
	})
	if allocations != 0 {
		t.Fatalf("warmed effect draws allocate %g times per frame, want 0", allocations)
	}
	if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != 1 {
		t.Fatalf("draws = %d, want the effect still rendered once", got)
	}
}
