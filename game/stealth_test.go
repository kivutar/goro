package game

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func stealthTestContext() client.Context {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 100, X: 10, Y: 20, HasState: true}
	w.GAT = &res.GAT{Width: 32, Height: 32, Cells: make([]res.GATCell, 32*32)}
	return client.Context{World: w, Session: &session.Session{
		AccountID: 100, CharID: 200, Vitals: session.Vitals{HP: 100},
		Selected: session.Character{ID: 200, HP: 100},
	}, ScreenW: 800, ScreenH: 600}
}

func TestStealthVisibilityAndPicking2008(t *testing.T) {
	for _, mapProperty := range []worldstate.MapProperty{worldstate.MapPropertyNothing, worldstate.MapPropertyFreePvPZone, worldstate.MapPropertyAgitZone} {
		for _, viewer := range []string{"self", "party", "other", "detection", "self detection", "GM"} {
			for _, state := range []uint32{0, db.EffectStateHide, db.EffectStateCloak, db.EffectStateChasewalk, db.EffectStateCloak | db.EffectStateChasewalk, db.EffectStateInvisible, actorStealthMask} {
				t.Run(fmt.Sprintf("map=%d/%s/option=%x", mapProperty, viewer, state), func(t *testing.T) {
					ctx := stealthTestContext()
					ctx.World.MapProperty = mapProperty
					actor := worldstate.Actor{ID: 300, X: 10, Y: 20, Job: db.JobAssassin, Name: "Hidden player", GuildID: 2, EmblemVersion: 1, Appearance: true, HasState: true, EffectState: state}
					local := viewer == "self" || viewer == "self detection"
					detection := viewer == "detection" || viewer == "self detection"
					if local {
						ctx.World.Player.EffectState = state
						actor.ID = ctx.Session.CharID
					} else {
						ctx.World.UpsertActor(actor)
					}
					if viewer == "party" {
						ctx.Session.Party.Members = []session.PartyMember{{AccountID: actor.ID}}
					}
					if detection {
						ctx.Session.Statuses.Active = map[uint16]session.StatusEffect{db.StatusClairvoyance: {}}
					}
					if viewer == "GM" {
						ctx.Session.AdminList = []uint32{ctx.Session.AccountID}
					}
					want := stealthHidden
					switch {
					case state == 0:
						want = stealthVisible
					case state&db.EffectStateInvisible != 0:
					case detection:
						want = stealthSilhouette
					case state == db.EffectStateHide && (local || viewer == "GM"):
						want = stealthShadow
					}
					projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
					mode := &WorldMode{}
					entries := mode.collectSceneActorEntries(render.NewFrame(800, 600), ctx, projection)
					found := false
					for _, entry := range entries {
						if entry.actor.ID == actor.ID {
							found = true
							if entry.stealth != want {
								t.Fatalf("render view = %d, want %d", entry.stealth, want)
							}
						}
					}
					if found != (local || want != stealthHidden) {
						t.Fatalf("actor in draw list = %t, want view %d", found, want)
					}
					if !local && state != 0 {
						if _, ok := hoveredCursorActor(ctx, projection, 400, 300, time.Now(), nil); ok {
							t.Fatal("hidden actor was hoverable")
						}
						if actorCanBeAttackClicked(ctx, actor) || actorCanOpenPlayerContext(ctx, actor) || actorCanBeSkillTargeted(ctx, session.Skill{ID: db.SkillALHeal, Type: skillTargetFriend}, actor) {
							t.Fatal("hidden actor was targetable")
						}
						if siegeActorShowsGuildEmblem(sceneActorDrawEntry{actor: actor}) {
							t.Fatal("hidden actor exposed its guild emblem")
						}
					}
				})
			}
		}
	}
}

func TestStealthDrawsShadowOrBlackBody(t *testing.T) {
	sprite := &spriteView{
		spr:    &res.SPR{Frames: []res.SPRFrame{{Type: res.SPRFrameRGBA, Width: 8, Height: 8, Data: solidRGBAFrame(8, 8)}}},
		act:    &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 0, SPRType: res.SPRFrameRGBA, ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1}}}}}}}},
		images: make(map[spriteFrameKey]*render.Image), billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	mode := &WorldMode{shadowView: sprite, playerView: &humanoidSpriteView{body: sprite, billboards: make(map[humanoidBillboardKey]*spriteBillboard)}}
	ctx := stealthTestContext()
	projection := newSceneProjectionForTarget(800, 600, cellCenter(10), cellCenter(20), 0)
	for _, tc := range []struct {
		view         stealthView
		body, shadow bool
	}{{stealthVisible, true, true}, {stealthShadow, false, true}, {stealthSilhouette, true, false}, {stealthHidden, false, false}} {
		t.Run(fmt.Sprint(tc.view), func(t *testing.T) {
			entry := mode.collectSceneActorEntries(render.NewFrame(800, 600), ctx, projection)[0]
			entry.stealth = tc.view
			frame := render.NewFrame(800, 600)
			mode.drawSceneActorEntry(frame, ctx, projection, entry)
			commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if (commands.Len() > 0) != tc.body {
				t.Fatalf("body draws = %d, want body=%t", commands.Len(), tc.body)
			}
			if tc.view == stealthSilhouette {
				for _, channel := range []string{"ColorR", "ColorG", "ColorB"} {
					if commands.Index(0).FieldByName(channel).Float() != 0 {
						t.Fatal("detected body was not black")
					}
				}
				if commands.Index(0).FieldByName("ColorA").Float() != 1 {
					t.Fatal("detected body was translucent")
				}
			}
			frame.BeginFrame()
			mode.drawActorShadowEntry(frame, ctx, projection, entry)
			commands = reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
			if (commands.Len() > 0) != tc.shadow {
				t.Fatalf("shadow draws = %d, want shadow=%t", commands.Len(), tc.shadow)
			}
		})
	}
}

func TestStealthActionPackets(t *testing.T) {
	ctx := stealthTestContext()
	netClient, server := newBotTestConnection(t, 20080910)
	ctx.Network = netClient
	mode := &WorldMode{}
	actor := worldstate.Actor{ID: 300, X: 11, Y: 20, Job: 1002, HasObjectType: true, ObjectType: actorObjectTypeMob, HasState: true}
	ctx.World.UpsertActor(actor)
	item := session.InventoryItem{Index: 7, ItemID: 501, Type: db.ItemTypeHealing}
	floorItem := worldstate.FloorItem{ID: 400, X: 10, Y: 20}
	ctx.World.Player.EffectState = db.EffectStateHide
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillRGTunneldrive, Level: 0}}
	assertNoBotTestPacket(t, server, func() error {
		if mode.requestWalk(ctx, 11, 20, "test") || mode.requestPickup(ctx, floorItem, "test") {
			t.Fatal("Hiding allowed movement or pickup")
		}
		mode.requestAttack(ctx, actor, "test")
		if client.SetSitting(ctx, true) == nil || gameui.UseInventoryItem(ctx, item) == nil {
			t.Fatal("Hiding allowed sitting or item use")
		}
		if mode.skills().Use(ctx, session.Skill{ID: db.SkillALHeal, Level: 1, Type: skillTargetFriend, Range: 9}, "test") == nil || mode.pendingSkill.skill.ID != 0 {
			t.Fatal("Hiding started a forbidden skill")
		}
		return nil
	})
	ctx.Session.Skills.List = []session.Skill{{ID: db.SkillRGTunneldrive, Level: 1}}
	if !mode.requestWalk(ctx, 11, 20, "test") {
		t.Fatal("Tunnel Drive could not walk")
	}
	walk, _ := network.BuildWalkToXYPacketForClientDate(11, 20, 20080910)
	readBotTestPackets(t, server, walk)
	for _, id := range []uint16{db.SkillTFHiding, db.SkillASGrimtooth, db.SkillRGBackstap, db.SkillRGRaid, db.SkillNJShadowjump, db.SkillNJKirikage} {
		skill := session.Skill{ID: id, Level: 1}
		if err := mode.skills().SendToID(ctx, skill, ctx.Session.AccountID, "test"); err != nil {
			t.Fatalf("allowed hidden skill %d: %v", id, err)
		}
		readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(id, 1, ctx.Session.AccountID, 20080910))
	}
	ctx.World.Player.EffectState = db.EffectStateCloak | db.EffectStateChasewalk
	assertNoBotTestPacket(t, server, func() error {
		mode.requestAttack(ctx, actor, "test")
		if mode.skills().SendToGround(ctx, session.Skill{ID: db.SkillWZStormgust, Level: 1}, 11, 20, "test") == nil {
			t.Fatal("Chase Walk allowed another skill")
		}
		return nil
	})
	if err := mode.skills().Use(ctx, session.Skill{ID: db.SkillSTChasewalk, Type: skillTargetSelf, Level: 1}, "test"); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, server, network.BuildUseSkillToIDPacketForClientDate(db.SkillSTChasewalk, 1, ctx.Session.AccountID, 20080910))
	ctx.World.Player.EffectState = db.EffectStateCloak
	mode.requestAttack(ctx, actor, "test")
	readLegacyBotTestActionPacket(t, server, actor.ID, network.ActionAttack)
	if err := gameui.UseInventoryItem(ctx, item); err != nil {
		t.Fatal(err)
	}
	readBotTestPackets(t, server, network.BuildUseInventoryItemPacketForClientDate(item.Index, ctx.Session.AccountID, 20080910))
	actor.EffectState = db.EffectStateHide
	ctx.World.UpsertActor(actor)
	assertNoBotTestPacket(t, server, func() error {
		mode.requestAttack(ctx, actor, "test")
		if mode.skills().SendToID(ctx, session.Skill{ID: db.SkillALHeal, Level: 1}, actor.ID, "test") == nil {
			t.Fatal("direct skill send targeted a hidden actor")
		}
		return nil
	})
}

func TestStealthDoesNotRestrictCompanionSkills(t *testing.T) {
	ctx := stealthTestContext()
	ctx.World.Player.EffectState = db.EffectStateHide | db.EffectStateChasewalk
	if !localStealthAllowsSkill(ctx, db.SkillHomunBegin+1) || !localStealthAllowsSkill(ctx, db.SkillMercenaryBegin+1) {
		t.Fatal("owner stealth restricted a companion's skills")
	}
}

func TestStealthCollectionHandlesPlayerOutsideViewport(t *testing.T) {
	ctx := stealthTestContext()
	ctx.World.Player.X = 100000
	ctx.World.Player.EffectState = db.EffectStateHide
	projection := newSceneProjectionForTarget(800, 600, 0, 0, 0)
	mode := &WorldMode{}
	if entries := mode.collectSceneActorEntries(render.NewFrame(800, 600), ctx, projection); len(entries) != 0 {
		t.Fatal("offscreen player remained in the draw list")
	}
}

func TestStealthCancelsQueuedActionsAndHidesAttachments(t *testing.T) {
	ctx := stealthTestContext()
	actor := worldstate.Actor{ID: 300, Job: db.JobAssassin, X: 11, Y: 20, HasLevel: true, Level: 99, Appearance: true, HasState: true}
	ctx.World.UpsertActor(actor)
	mode := &WorldMode{}
	mode.syncLevel99AuraEffects(ctx, time.Now())
	if len(mode.worldEffects) == 0 {
		t.Fatal("missing initial level aura")
	}
	mode.pendingAttack = attackIntent{targetID: actor.ID}
	mode.lockedAttackID, mode.attackFocusID = actor.ID, actor.ID
	mode.pendingSkill = pendingSkillTarget{skill: session.Skill{ID: db.SkillALHeal}, targetID: actor.ID}
	mode.applyActorStateChange(ctx, network.ActorStateChange{ID: actor.ID, EffectState: db.EffectStateCloak})
	mode.syncLevel99AuraEffects(ctx, time.Now())
	if mode.pendingAttack.targetID != 0 || mode.lockedAttackID != 0 || mode.attackFocusID != 0 || mode.pendingSkill.skill.ID != 0 || len(mode.worldEffects) != 0 {
		t.Fatal("hidden target retained an action or aura")
	}
	mode.applyActorStateChange(ctx, network.ActorStateChange{ID: actor.ID})
	mode.syncLevel99AuraEffects(ctx, time.Now())
	if len(mode.worldEffects) == 0 {
		t.Fatal("revealed actor did not regain its aura")
	}
	mode.pendingAttack = attackIntent{targetID: actor.ID}
	mode.pendingPickup = pickupIntent{itemID: 400}
	mode.pendingSkill = pendingSkillTarget{skill: session.Skill{ID: db.SkillALHeal}, targetID: actor.ID}
	mode.applyActorStateChange(ctx, network.ActorStateChange{ID: ctx.Session.AccountID, EffectState: db.EffectStateHide})
	if mode.pendingAttack.targetID != 0 || mode.pendingPickup.itemID != 0 || mode.pendingSkill.skill.ID != 0 {
		t.Fatal("Hiding retained a queued local action")
	}
	entry := sceneActorDrawEntry{actor: worldstate.Actor{ID: 100, EffectState: db.EffectStateHide | db.EffectStateCart1 | db.EffectStateFalcon, HasCart: true, HasCartState: true}}
	if mode.drawActorCart3D(nil, ctx, sceneProjection{}, entry, 0, 1, 1) {
		t.Fatal("hidden cart was drawn")
	}
	mode.drawSceneActorFalcons(nil, ctx, sceneProjection{}, []sceneActorDrawEntry{entry})
	if len(mode.falcons) != 0 {
		t.Fatal("hidden actor created a falcon")
	}
}
