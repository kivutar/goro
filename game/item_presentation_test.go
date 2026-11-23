package game

import (
	"math"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/actors"
	"github.com/kivutar/goro/game/internal/testutil"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func TestOverlappingGroundItemsKeepDrawOrder(t *testing.T) {
	world := worldstate.New()
	// Different sprites at the same sub-tile position share exactly one depth.
	// IDs ending in the same digits must still have a deterministic order.
	for i, id := range []uint32{201, 101, 301} {
		world.Items[id] = worldstate.FloorItem{ID: id, ItemID: uint16(501 + i), X: 10, Y: 20, SubX: 6, SubY: 6}
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	items := (&WorldMode{}).collectSceneItemEntries(render.NewFrame(800, 600), client.Context{World: world}, projection, time.Now())
	if len(items) != 3 {
		t.Fatalf("item entries = %d, want 3", len(items))
	}
	// Exercise all collection orders explicitly, without relying on random map
	// iteration to expose a flicker.
	for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		shuffled := []sceneItemDrawEntry{items[order[0]], items[order[1]], items[order[2]]}
		var sprites, shadows []uint32
		for _, entry := range sortedSceneDrawEntries(nil, shuffled) {
			if entry.itemShadowIndex >= 0 {
				if len(sprites) != 0 {
					t.Fatal("item shadow drawn over a settled item")
				}
				shadows = append(shadows, shuffled[entry.itemShadowIndex].item.ID)
			} else {
				sprites = append(sprites, shuffled[entry.itemIndex].item.ID)
			}
		}
		want := []uint32{101, 201, 301}
		if !slices.Equal(sprites, want) || !slices.Equal(shadows, want) {
			t.Fatalf("input order %v: sprites %v, shadows %v, want %v for both", order, sprites, shadows, want)
		}
	}
}

func TestSceneDrawOrderKeepsDepthBeforeEntityID(t *testing.T) {
	actors := []scene.ActorDrawEntry{{
		Actor: worldstate.Actor{ID: 20}, Depth: 100, ShadowDepth: 102, CastShadow: true,
	}}
	items := []sceneItemDrawEntry{
		{item: worldstate.FloorItem{ID: 30}, depth: 100, shadowDepth: 102},
		{item: worldstate.FloorItem{ID: 10}, depth: 100, shadowDepth: 102},
		{item: worldstate.FloorItem{ID: 1}, depth: 90, shadowDepth: 92},
		{item: worldstate.FloorItem{ID: 99}, depth: 110, shadowDepth: 112},
	}
	entries := sortedSceneDrawEntries(actors, items)
	var ids []uint32
	for i, entry := range entries {
		ids = append(ids, entry.entityID)
		if i > 0 && entry.depth > entries[i-1].depth {
			t.Fatalf("entry %d at depth %.2f drawn after nearer entry at %.2f", i, entry.depth, entries[i-1].depth)
		}
	}
	want := []uint32{99, 99, 10, 20, 30, 10, 20, 30, 1, 1}
	if !slices.Equal(ids, want) {
		t.Fatalf("draw order = %v, want %v", ids, want)
	}
}

func TestFallingGroundItemKeepsShadowOnTerrain(t *testing.T) {
	now := time.Now()
	world := worldstate.New()
	world.Items[1] = worldstate.FloorItem{
		ID:        1,
		ItemID:    501,
		X:         10,
		Y:         20,
		Falling:   true,
		DroppedAt: now,
	}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	mode := &WorldMode{}
	entries := mode.collectSceneItemEntries(render.NewFrame(800, 600), client.Context{World: world}, projection, now)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.worldZ <= entry.groundZ {
		t.Fatalf("falling item z = %.2f, ground z = %.2f", entry.worldZ, entry.groundZ)
	}
	wantDepth := projection.Depth(entry.worldX, entry.worldY, entry.groundZ+scene.ActorShadowTerrainLift)
	if math.Abs(entry.shadowDepth-wantDepth) > 1e-9 {
		t.Fatalf("shadow depth = %.9f, want terrain depth %.9f", entry.shadowDepth, wantDepth)
	}
}

func TestGroundItemSortsAfterItsShadow(t *testing.T) {
	world := worldstate.New()
	world.Items[1] = worldstate.FloorItem{ID: 1, ItemID: 501, X: 10, Y: 20}
	projection := testutil.Projection(800, 600, scene.CellCenter(10), scene.CellCenter(20), 0)
	entries := (&WorldMode{}).collectSceneItemEntries(
		render.NewFrame(800, 600),
		client.Context{World: world},
		projection,
		time.Now(),
	)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].depth >= entries[0].shadowDepth {
		t.Fatalf("item depth = %.4f, shadow depth = %.4f; item must sort after its shadow", entries[0].depth, entries[0].shadowDepth)
	}
}

func TestDrawGroundItemShadowUsesSharedOriginalShadowSprite(t *testing.T) {
	if groundItemShadowScale != 0.25 {
		t.Fatalf("floor-item shadow scale = %.2f, want roBrowser's 0.25", groundItemShadowScale)
	}
	shadowView := &scene.SpriteView{
		Spr: &res.SPR{
			RGBAIndex: 0,
			Frames: []res.SPRFrame{{
				Type:   res.SPRFrameRGBA,
				Width:  16,
				Height: 8,
				Data:   testutil.SolidRGBAFrame(16, 8),
			}},
		},
		Act: &res.ACT{Actions: []res.ACTAction{{
			Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{
				Index:   0,
				SPRType: res.SPRFrameRGBA,
				ScaleX:  1,
				ScaleY:  1,
				Color:   [4]float32{1, 1, 1, 1},
			}}}},
		}}},
		Images:     make(map[scene.SpriteFrameKey]*render.Image),
		Billboards: make(map[scene.SpriteBillboardKey]*scene.Billboard),
	}
	mode := &WorldMode{actorSystem: actors.System{ShadowView: shadowView}}
	entry := sceneItemDrawEntry{
		worldX:      scene.CellCenter(10),
		worldY:      scene.CellCenter(20),
		groundZ:     0,
		shadowScale: groundItemShadowScale,
		shadow:      scene.FullLight,
	}
	projection := testutil.Projection(800, 600, entry.worldX, entry.worldY, 0)
	frame := render.NewFrame(800, 600)
	if !mode.drawGroundItemShadowEntry3D(frame, projection, entry) {
		t.Fatal("floor-item shadow was not drawn")
	}
	entry.billboard = &scene.Billboard{Image: render.NewImage(16, 16), AnchorX: 8, AnchorY: 16}
	entry.scale = 1
	mode.drawGroundItemEntry3D(frame, projection, entry)
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if commands.Len() != 2 {
		t.Fatal("expected the shadow followed by the item sprite")
	}
	if commands.Index(0).FieldByName("Options").FieldByName("DepthWrite").Bool() ||
		!commands.Index(1).FieldByName("Options").FieldByName("DepthWrite").Bool() {
		t.Fatal("the item must occlude translucent geometry; its shadow must not")
	}
}

func TestGroundItemShadowIsOffsetDownWithoutMutatingSharedBillboard(t *testing.T) {
	base := &scene.Billboard{AnchorX: 20, AnchorY: 30}
	got := groundItemShadowBillboard(base)
	if got == base {
		t.Fatal("item shadow offset should copy the shared billboard")
	}
	if got.AnchorX != 20 || got.AnchorY != 30-groundItemShadowOffsetY {
		t.Fatalf("item shadow anchor = %.1f, %.1f", got.AnchorX, got.AnchorY)
	}
	if base.AnchorX != 20 || base.AnchorY != 30 {
		t.Fatalf("shared shadow billboard was mutated: %.1f, %.1f", base.AnchorX, base.AnchorY)
	}
}
