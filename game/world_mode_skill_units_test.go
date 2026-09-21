package game

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func TestGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectThunderStorm || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 21, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("deduped world effects = %d, want 1", len(mode.worldEffects))
	}
}

func TestGroundSkillNotifyPrefersDancerGroundEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: db.SkillDCHumming, SourceID: 2000000, Level: 4, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectBottomHummingGround || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v, want Dancer ground row", effect)
	}
}

func TestPneumaGroundSkillNotifyAddsCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyGroundSkillNotify(ctx, network.GroundSkillNotify{SkillID: 25, SourceID: 2000000, Level: 1, X: 123, Y: 456})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 0 || effect.effectID != effectPneuma || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestGospelGroundSkillNotifyPlaysOneCastSoundOnCaster(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	notify := network.GroundSkillNotify{SkillID: db.SkillPaGospel, SourceID: 2000000, Level: 10}

	mode.applyGroundSkillNotify(ctx, notify)
	mode.applyGroundSkillNotify(ctx, notify)

	if len(mode.worldEffects) != 1 {
		t.Fatalf("Gospel cast effects = %+v, want one deduplicated caster effect", mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.effectID != effectBottomGospel || effect.actorID != 2000000 {
		t.Fatalf("Gospel cast effect = %+v, want EF_BOTTOM_GOSPEL on caster", effect)
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\가스펠.wav" {
		t.Fatalf("Gospel cast sounds = %+v, want one Gospel sound", mode.scheduledSounds)
	}
}

func TestGospelSkillUnitUsesPersistentGroundVisualWithoutSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9179, CreatorID: 2000000, UnitID: 179, X: 12, Y: 34, Visible: true})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("Gospel unit effects = %+v, want one", mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.effectID != effectGospelGround || effect.actorID != 9179 || effect.x != 12 || effect.y != 34 || !effect.persistent {
		t.Fatalf("Gospel unit effect = %+v, want persistent 370_ground at unit cell", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("Gospel unit sounds = %+v, want cast sound owned by poseffect only", mode.scheduledSounds)
	}
}

func TestSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9001, CreatorID: 2000000, UnitID: 126, X: 123, Y: 456, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9001 || effect.effectID != effectSafetyWall || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9001})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestFireWallSkillUnitEntryUsesPersistentEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9007, CreatorID: 2000000, UnitID: 127, X: 12, Y: 34, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9007 || effect.effectID != effectFireWall || effect.x != 12 || effect.y != 34 {
		t.Fatalf("effect = %+v", effect)
	}
	if !effect.persistent {
		t.Fatalf("fire wall skill unit effect is not persistent")
	}
	if effect.expires.Sub(effect.starts) < skillUnitEffectFallbackDuration {
		t.Fatalf("fire wall lifetime = %s, want skill unit fallback", effect.expires.Sub(effect.starts))
	}
	if effect.duration != 0 {
		t.Fatalf("fire wall animation override = %s, want native component timing", effect.duration)
	}
}

func TestSkillUnitEntryRefreshesExistingPersistentCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 12, Y: 34, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	first := mode.worldEffects[0]

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9008, CreatorID: 2000000, UnitID: 172, X: 13, Y: 35, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects after refresh = %d, want 1: %+v", len(mode.worldEffects), mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9008 || effect.effectID != effectBottomHummingGround || effect.x != 13 || effect.y != 35 {
		t.Fatalf("effect after refresh = %+v, want moved humming ground effect", effect)
	}
	if !effect.starts.Equal(first.starts) {
		t.Fatalf("effect start changed from %s to %s, want loop phase preserved", first.starts, effect.starts)
	}
	if effect.expires.Before(first.expires) {
		t.Fatalf("effect expiry moved backward from %s to %s", first.expires, effect.expires)
	}
}

func TestWizardSkillUnitEntrySchedulesCellSound(t *testing.T) {
	for _, tc := range []struct {
		name   string
		unitID uint16
		path   string
	}{
		{name: "ice wall", unitID: 141, path: "effect\\wizard_icewall.wav"},
		{name: "quagmire", unitID: 142, path: "effect\\wizard_quagmire.wav"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9101, CreatorID: 2000000, UnitID: tc.unitID, X: 12, Y: 34, Visible: true})

			if len(mode.scheduledSounds) != 1 {
				t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
			}
			sound := mode.scheduledSounds[0]
			if len(sound.paths) != 1 || sound.paths[0] != tc.path {
				t.Fatalf("sound paths = %+v, want %s", sound.paths, tc.path)
			}
			if !sound.positioned || sound.actorID != 9101 || !sound.hasPosition {
				t.Fatalf("sound position metadata = %+v", sound)
			}
			x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
			if !ok || x != 12 || y != 34 {
				t.Fatalf("sound position = %.0f,%.0f ok=%t, want 12,34 true", x, y, ok)
			}
		})
	}
}

func TestStormGustSkillNotifySchedulesSound(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 34}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillWZStormgust, TargetID: 1100, SourceID: 2000000, Result: 1})

	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v, want one", mode.scheduledSounds)
	}
	sound := mode.scheduledSounds[0]
	if len(sound.paths) != 1 || sound.paths[0] != "effect\\wizard_stormgust.wav" {
		t.Fatalf("sound paths = %+v, want storm gust sfx", sound.paths)
	}
	x, y, ok := scheduledSoundPosition(ctx, sound, time.Now())
	if !ok || x != 12 || y != 34 {
		t.Fatalf("sound position = %.0f,%.0f ok=%t, want target position 12,34 true", x, y, ok)
	}
}

func TestQuagmireSkillUnitEntriesBuildFiveByFiveZone(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	const centerX, centerY = 50, 60
	entryID := uint32(9100)
	for dy := -2; dy <= 2; dy++ {
		for dx := -2; dx <= 2; dx++ {
			mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: 142, X: uint16(centerX + dx), Y: uint16(centerY + dy), Visible: true})
			entryID++
		}
	}

	if len(mode.worldEffects) != 25 {
		t.Fatalf("quagmire world effects = %d, want 25", len(mode.worldEffects))
	}
	cells := make(map[worldstate.WalkStep]bool, len(mode.worldEffects))
	for _, effect := range mode.worldEffects {
		if effect.effectID != effectQuagmire {
			t.Fatalf("effect id = %d, want quagmire: %+v", effect.effectID, effect)
		}
		if !effect.persistent {
			t.Fatalf("quagmire unit effect is not persistent: %+v", effect)
		}
		cells[worldstate.WalkStep{X: effect.x, Y: effect.y}] = true
	}
	for y := centerY - 2; y <= centerY+2; y++ {
		for x := centerX - 2; x <= centerX+2; x++ {
			if !cells[worldstate.WalkStep{X: x, Y: y}] {
				t.Fatalf("missing quagmire cell %d,%d in %+v", x, y, cells)
			}
		}
	}
}

func TestTrapSkillUnitEntryAddsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9201, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %+v, want none for trap placement model", mode.worldEffects)
	}
	unit, ok := mode.skillUnitModels[9201]
	if !ok {
		t.Fatal("trap model was not tracked")
	}
	if unit.unitID != 145 || unit.x != 12 || unit.y != 34 || unit.modelPath != db.SkillUnitModels[145].ModelPath {
		t.Fatalf("trap model = %+v", unit)
	}
	if len(unit.modelFallbacks) != 1 || unit.modelFallbacks[0] != "effect\\trap03.rsm" {
		t.Fatalf("trap fallbacks = %+v", unit.modelFallbacks)
	}
	if unit.scale != 0.15 || !unit.hasFixedFrame || unit.fixedFrame != 3 {
		t.Fatalf("trap render controls = scale %.3f fixed=%t frame=%d", unit.scale, unit.hasFixedFrame, unit.fixedFrame)
	}
}

func TestRobrowserRSMTrapSkillUnitModelsUseTrapRenderControls(t *testing.T) {
	expected := map[uint16]string{
		143: "외부소품\\트랩03_3.rsm",
		144: "외부소품\\트랩02.rsm",
		145: "외부소품\\트랩01.rsm",
		147: "외부소품\\트랩03.rsm",
		148: "외부소품\\트랩03_6.rsm",
		149: "외부소품\\트랩03_4.rsm",
		150: "외부소품\\트랩03_5.rsm",
		151: "외부소품\\트랩03_2.rsm",
		152: "외부소품\\트랩04.rsm",
		153: "외부소품\\트랩05.rsm",
		210: "event\\3차트랩_변화01.rsm",
		211: "event\\3차트랩_변수01.rsm",
		212: "event\\3차트랩_변지01.rsm",
		213: "event\\3차트랩_변풍01.rsm",
		214: "event\\3차트랩_화01.rsm",
		215: "event\\3차트랩_수01.rsm",
		216: "event\\3차트랩_풍01.rsm",
		217: "event\\3차트랩_지01.rsm",
		229: "event\\3차트랩_가시01.rsm",
	}
	for unitID, wantPath := range expected {
		spec, ok := db.SkillUnitModels[unitID]
		if !ok {
			t.Fatalf("trap unit %d is not mapped", unitID)
		}
		if spec.ModelPath != wantPath {
			t.Fatalf("trap unit %d model = %q, want %q", unitID, spec.ModelPath, wantPath)
		}
		if spec.Scale != 0.15 || !spec.HasFixedFrame || spec.FixedFrame != 3 {
			t.Fatalf("trap unit %d render controls = scale %.3f fixed=%t frame=%d", unitID, spec.Scale, spec.HasFixedFrame, spec.FixedFrame)
		}
	}
}

func TestHiddenTrapSkillUnitUpdateRevealsRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9202, CreatorID: 3000000, UnitID: 151, X: 12, Y: 34, Visible: false})
	if len(mode.skillUnitModels) != 0 {
		t.Fatalf("visible trap models = %+v, want none", mode.skillUnitModels)
	}
	if _, ok := mode.hiddenSkillUnits[9202]; !ok {
		t.Fatal("hidden trap model was not tracked")
	}

	mode.applySkillUnitUpdate(network.SkillUnitUpdate{ID: 9202})

	if _, ok := mode.hiddenSkillUnits[9202]; ok {
		t.Fatalf("hidden trap still tracked after reveal: %+v", mode.hiddenSkillUnits[9202])
	}
	if unit, ok := mode.skillUnitModels[9202]; !ok || unit.unitID != 151 || unit.x != 12 || unit.y != 34 {
		t.Fatalf("visible trap after reveal = %+v ok=%t", unit, ok)
	}
}

func TestUsedTrapLookChangeRemovesModelAndSpawnsTriggerBurst(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9203, CreatorID: 2000000, UnitID: 151, X: 12, Y: 34, Visible: true})
	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9203, Type: 0, Value: uint32(db.SkillUnitUsedTraps)}) {
		t.Fatal("used trap look change was not handled")
	}

	if len(mode.skillUnitModels) != 0 {
		t.Fatalf("trap models after trigger = %+v, want none", mode.skillUnitModels)
	}
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %+v, want freezing trap burst", mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.effectID != effectFreezingTrap || effect.actorID != 0 || effect.x != 12 || effect.y != 34 || effect.persistent {
		t.Fatalf("trigger burst = %+v", effect)
	}
}

func TestSkillUnitDisappearRemovesRuntimeRSMModel(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9204, CreatorID: 2000000, UnitID: 145, X: 12, Y: 34, Visible: true})
	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9204})

	if len(mode.skillUnitModels) != 0 || len(mode.hiddenSkillUnits) != 0 || len(mode.worldEffects) != 0 {
		t.Fatalf("skill unit state after disappear models=%+v hidden=%+v effects=%+v", mode.skillUnitModels, mode.hiddenSkillUnits, mode.worldEffects)
	}
}

func TestSkillUnitRSMPlacementUsesCellCenterAndTerrainHeight(t *testing.T) {
	world := &worldstate.World{
		GAT: &res.GAT{
			Width:  2,
			Height: 2,
			Cells: []res.GATCell{
				{Heights: [4]float32{1, 1, 1, 1}},
				{Heights: [4]float32{2, 2, 2, 2}},
				{Heights: [4]float32{3, 3, 3, 3}},
				{Heights: [4]float32{7, 7, 7, 7}},
			},
		},
	}
	unit := skillUnitModel{unitID: 145, x: 1, y: 1, modelPath: db.SkillUnitModels[145].ModelPath}

	placement := skillUnitRSMPlacement(world, 9205, unit)

	if placement.baseX != 1.5 || placement.baseY != 1.5 {
		t.Fatalf("base = %.1f,%.1f, want cell center 1.5,1.5", placement.baseX, placement.baseY)
	}
	if placement.model.Position.Y != 7 {
		t.Fatalf("model y = %.1f, want terrain height 7", placement.model.Position.Y)
	}
	if placement.model.Scale != (res.RSWVector3{X: 1, Y: 1, Z: 1}) {
		t.Fatalf("scale = %+v, want 1,1,1", placement.model.Scale)
	}
}

func TestTrapSkillUnitRSMPlacementUsesEffectScale(t *testing.T) {
	world := worldstate.New()
	unit := skillUnitModel{unitID: 145, x: 1, y: 1, modelPath: db.SkillUnitModels[145].ModelPath, scale: db.SkillUnitModels[145].Scale}

	placement := skillUnitRSMPlacement(world, 9206, unit)

	want := res.RSWVector3{X: 0.15, Y: 0.15, Z: 0.15}
	if placement.model.Scale != want {
		t.Fatalf("scale = %+v, want skill-unit trap scale %+v", placement.model.Scale, want)
	}
}

func TestTrapSkillUnitRSMFrameUsesClassicClientFixedFrame(t *testing.T) {
	rsm := &res.RSM{
		AnimLength: 12,
		Nodes: []res.RSMNode{{
			Name: "root",
			PositionKeyframes: []res.RSMPositionKeyframe{
				{Frame: 0},
				{Frame: 12, Pos: res.RSMVector3{X: 12}},
			},
		}},
	}
	unit := skillUnitModel{hasFixedFrame: true, fixedFrame: 3}

	frame, animated := skillUnitRSMFrame(rsm, res.RSWModel{}, unit, time.UnixMilli(1000))
	if animated || frame != 3 {
		t.Fatalf("trap RSM frame = %d animated=%t, want fixed frame 3", frame, animated)
	}
}

func TestMappedSkillUnitEntriesUsePersistentEffects(t *testing.T) {
	for unitID, spec := range skillUnitEffectSpecs {
		if len(spec.effectIDs) == 0 {
			continue
		}
		world := worldstate.New()
		world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
		mode := &WorldMode{}
		ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
		entryID := uint32(unitID) + 100000

		mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: entryID, CreatorID: 2000000, UnitID: unitID, X: 12, Y: 34, Visible: true})
		if len(mode.worldEffects) != len(spec.effectIDs) {
			t.Fatalf("unit %d world effects = %d, want %d", unitID, len(mode.worldEffects), len(spec.effectIDs))
		}
		for _, effect := range mode.worldEffects {
			if !effect.persistent {
				t.Fatalf("unit %d effect %d is not persistent", unitID, effect.effectID)
			}
		}
	}
}

func TestRepeatedSTRKeyIndexLoops(t *testing.T) {
	starts := time.Unix(100, 0)
	now := starts.Add(250 * time.Millisecond)
	if got := strEffectKeyIndex(starts, now, 60, 12, true); got != 3 {
		t.Fatalf("repeated key index = %.2f, want 3", got)
	}
	if got := strEffectKeyIndex(starts, now, 60, 12, false); got != 15 {
		t.Fatalf("one-shot key index = %.2f, want 15", got)
	}
}

func TestWarpPortalSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9003, CreatorID: 2000000, UnitID: 128, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9003 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9003})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestPreWarpPortalSkillUnitEntryUsesReadyPortalEffectUntilDisappear(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9005, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9005 || effect.effectID != effectReadyPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect = %+v", effect)
	}
	if effect.expires.Sub(effect.starts) < skillUnitEffectFallbackDuration {
		t.Fatalf("portal lifetime = %s, want skill unit fallback", effect.expires.Sub(effect.starts))
	}
	if effect.duration != 0 {
		t.Fatalf("portal animation override = %s, want native component timing", effect.duration)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9005})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestWarpPortalSkillUnitLookChangeKeepsPortalAtSameCell(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9006, CreatorID: 2000000, UnitID: 129, X: 30, Y: 40, Visible: true})
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectReadyPortal {
		t.Fatalf("world effects before look change = %+v, want ready portal", mode.worldEffects)
	}

	if !mode.applySkillUnitLookChange(ctx, network.ActorLookChange{ID: 9006, Type: 0, Value: 128}) {
		t.Fatal("skill unit look change was not handled")
	}
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1: %+v", len(mode.worldEffects), mode.worldEffects)
	}
	effect := mode.worldEffects[0]
	if effect.actorID != 9006 || effect.effectID != effectPortal || effect.x != 30 || effect.y != 40 {
		t.Fatalf("effect after look change = %+v, want portal on same unit cell", effect)
	}
}

func TestSkillUnitEntryDispatchesAllMappedUnitEffectArrays(t *testing.T) {
	const unitID uint16 = 65000
	skillUnitEffectSpecs[unitID] = skillUnitEffectSpec{effectIDs: []int{effectPneuma, effectSafetyWall}}
	defer delete(skillUnitEffectSpecs, unitID)

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9004, CreatorID: 2000000, UnitID: unitID, X: 123, Y: 456, Visible: true})

	want := []int{effectPneuma, effectSafetyWall}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, wantEffectID := range want {
		effect := mode.worldEffects[i]
		if effect.actorID != 9004 || effect.effectID != wantEffectID || effect.x != 123 || effect.y != 456 {
			t.Fatalf("effect %d = %+v, want effect %d on unit", i, effect, wantEffectID)
		}
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9004})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestPneumaSkillUnitEntryAddsAndRemovesCellEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillUnitEntry(ctx, network.SkillUnitEntry{ID: 9002, CreatorID: 2000000, UnitID: 133, X: 123, Y: 456, Visible: true})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 9002 || effect.effectID != effectPneuma || effect.x != 123 || effect.y != 456 {
		t.Fatalf("effect = %+v", effect)
	}

	mode.applySkillUnitDisappear(network.SkillUnitDisappear{ID: 9002})
	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects after disappear = %d, want 0", len(mode.worldEffects))
	}
}

func TestSkillNoDamageNotifyAddsProvokeEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 6, Amount: 2, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectProvoke || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyAddsSongTalkBeginEffects(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "dc_scream.txt"), []byte("SCREAM\r\n\tDancer line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "ba_frostjoke.txt"), []byte("FROST JOKE\r\n\tBard line\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager, err := res.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		skillID  uint16
		effectID int
		text     string
	}{
		{name: "scream", skillID: db.SkillDCScream, effectID: effectTalkScream, text: "Dancer line"},
		{name: "frost joke", skillID: db.SkillBaFrostjoke, effectID: effectTalkFrostJoke, text: "Bard line"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 150000, X: 10, Y: 20}
			sessionState := &session.Session{
				AccountID: 2000000,
				CharID:    150000,
				Selected:  session.Character{ID: 150000, Name: "Kivutar", Job: db.JobDancer},
			}
			mode := &WorldMode{}
			ctx := client.Context{Session: sessionState, World: world, Resources: manager}

			mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: tc.skillID, TargetID: 2000000, SourceID: 2000000, Result: 1})
			if len(mode.worldEffects) != 1 {
				t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
			}
			if effect := mode.worldEffects[0]; effect.actorID != 2000000 || effect.effectID != tc.effectID {
				t.Fatalf("effect = %+v, want actor 2000000 effect %d", effect, tc.effectID)
			}
			if bubble, ok := mode.speechBubbles[2000000]; !ok || bubble.text != tc.text {
				t.Fatalf("account bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
			if bubble, ok := mode.speechBubbles[150000]; !ok || bubble.text != tc.text {
				t.Fatalf("char bubble = %+v ok=%t, want %q", bubble, ok, tc.text)
			}
		})
	}
}

func TestSkillNoDamageNotifyAddsStealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 50, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectSteal || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
}

func TestSkillNoDamageNotifyEndureUsesReadyFightAction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 1}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1}}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 8, TargetID: 2000000, SourceID: 2000000, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client READYFIGHT action", anim)
	}
	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectEndure {
		t.Fatalf("world effects = %+v, want Endure effect", mode.worldEffects)
	}
}

func TestSkillNoDamageNotifyAddsHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.Actors[1100] = worldstate.Actor{ID: 1100, X: 12, Y: 22}
	sessionState := &session.Session{AccountID: 2000000, CharID: 150000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: 28, Amount: 234, TargetID: 1100, SourceID: 2000000, Result: 1})
	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.actorID != 1100 || effect.effectID != effectHeal || effect.x != 12 || effect.y != 22 {
		t.Fatalf("effect = %+v", effect)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source cast animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
}

func TestActorActionNotifyHealUsesCastAndOffensiveHealEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      8,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	found := false
	for _, effect := range mode.worldEffects {
		if effect.effectID == effectHealOffensive && effect.actorID == 300 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("world effects = %+v, want offensive heal effect on target", mode.worldEffects)
	}
}

func TestActorActionNotifyBashUsesRobrowserWeaponAttackOverride(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1015,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 1, Hair: 1, Weapon: 3}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     5,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCAttack2 {
		t.Fatalf("source animation = %+v, want reference client weapon attack override", anim)
	}
}

func TestActorActionNotifyHealDoesNotOverwriteLocalCastWithHurt(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: 4, Hair: 1}},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     28,
		SkillLevel:  3,
		SourceID:    2000000,
		TargetID:    2000000,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      84,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("source animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("source animation = %+v, want reference client DEFAULT skill action", anim)
	}
	if len(mode.worldEffects) != 2 || mode.worldEffects[0].effectID != effectHeal || mode.worldEffects[1].effectID != effectHealOffensive {
		t.Fatalf("world effects = %+v, want reference client heal effect followed by hit effect", mode.worldEffects)
	}
}

func TestSkillFailAckAddsConsoleErrorWithoutEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{AccountID: 2000000}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 6, Result: 0, Cause: 0})

	if len(mode.worldEffects) != 0 {
		t.Fatalf("world effects = %d, want 0", len(mode.worldEffects))
	}
	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Action failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestStealFailAckUsesSkillSpecificMessage(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}}

	mode.applySkillFailAck(ctx, network.SkillFailAck{SkillID: 50, Result: 0, Cause: 0})

	messages := mode.ui.console.Messages()
	if len(messages) != 1 || messages[0].Text != "Steal failed." {
		t.Fatalf("console messages = %+v", messages)
	}
}

func TestPickedInventoryItemAddsToExistingStack(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{Index: 7, ItemID: 512, Type: 0, Amount: 3, Identified: true}},
		},
	}

	addPickedSessionInventoryItem(sessionState, session.InventoryItem{Index: 7, ItemID: 512, Type: 3, Amount: 2})

	if got := sessionState.Inventory.Items[0].Amount; got != 5 {
		t.Fatalf("picked stack amount = %d, want 5", got)
	}
	if got := sessionState.Inventory.Items[0].Type; got != 0 {
		t.Fatalf("picked stack type = %d, want preserved healing type", got)
	}
}

func TestSessionItemFromNetworkMarksEquipmentByType(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      7,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
	})
	if !item.Equip {
		t.Fatalf("item = %+v, want equipment item", item)
	}
}

func TestSessionItemFromNetworkDefaultsAmmoLocation(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      8,
		ItemID:     1750,
		Type:       10,
		Identified: true,
		Amount:     100,
	})
	if !item.Equip || item.Location != db.EquipAmmo {
		t.Fatalf("ammo item = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestSessionItemFromNetworkDoesNotMarkCardsAsEquipment(t *testing.T) {
	item := sessionItemFromNetwork(network.InventoryItem{
		Index:      9,
		ItemID:     4001,
		Type:       db.ItemTypeCard,
		Location:   db.EquipAccessory1,
		Identified: true,
		Amount:     1,
		Equip:      true,
		Equipped:   true,
	})
	if item.Equip || item.Equipped || item.Location != db.EquipAccessory1 {
		t.Fatalf("card item = %+v, want non-equipment card with original location", item)
	}
}

func TestInventoryItemListReplacesDifferentItemAtReusedIndex(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{{
				Index:    11,
				ItemID:   1201,
				Type:     5,
				Location: 0x0002,
				Amount:   1,
				Equip:    true,
			}},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryItemList(ctx, []network.InventoryItem{{
		Index:  11,
		ItemID: 938,
		Type:   3,
		Amount: 2,
	}})

	item := sessionState.Inventory.Items[0]
	if item.Equip || item.Location != 0 || item.Type != 3 || item.ItemID != 938 {
		t.Fatalf("item = %+v, want clean replacement", item)
	}
}

func TestPickedEquipmentKeepsEquipMetadata(t *testing.T) {
	sessionState := &session.Session{}
	addPickedSessionInventoryItem(sessionState, session.InventoryItem{
		Index:      11,
		ItemID:     1201,
		Type:       5,
		Location:   0x0002,
		Identified: true,
		Amount:     1,
		Equip:      inventoryItemTypeIsEquipment(5),
	})
	if len(sessionState.Inventory.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(sessionState.Inventory.Items))
	}
	item := sessionState.Inventory.Items[0]
	if !item.Equip || item.Location != 0x0002 {
		t.Fatalf("picked item = %+v, want equipment metadata", item)
	}
}

func TestApplyInventoryEquipAckUpdatesEquippedState(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 1, ItemID: 1201, Type: 4, Location: 0x0002, Equip: true},
				{Index: 2, ItemID: 1202, Type: 4, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true})
	if !sessionState.Inventory.Items[0].Equipped {
		t.Fatal("equipped item was not marked equipped")
	}
	if sessionState.Inventory.Items[1].Equipped {
		t.Fatal("previous item in same location stayed equipped")
	}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 1, Location: 0x0002, Success: true, Unequip: true})
	if sessionState.Inventory.Items[0].Equipped {
		t.Fatal("unequipped item stayed equipped")
	}
}

func TestApplyInventoryEquipAckDefaultsAmmoLocation(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Equip: true},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyInventoryEquipAck(ctx, network.InventoryEquipAck{Index: 3, Success: true})

	item := sessionState.Inventory.Items[0]
	if !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
		t.Fatalf("ammo item after equip ack = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestApplyEquippedArrowMarksAmmoSlot(t *testing.T) {
	sessionState := &session.Session{
		Inventory: session.Inventory{
			Items: []session.InventoryItem{
				{Index: 3, ItemID: 1750, Type: 10, Amount: 100, Location: db.EquipAmmo, WearLocation: db.EquipAmmo, Equip: true, Equipped: true},
				{Index: 9, ItemID: 1751, Type: 10, Amount: 50},
			},
		},
	}
	ctx := client.Context{Session: sessionState}

	applyEquippedArrow(ctx, network.EquippedArrow{Index: 9})

	if sessionState.Inventory.Items[0].Equipped || sessionState.Inventory.Items[0].WearLocation != 0 {
		t.Fatal("previous arrow stayed equipped")
	}
	item := sessionState.Inventory.Items[1]
	if !item.Equip || !item.Equipped || item.Location != db.EquipAmmo || item.WearLocation != db.EquipAmmo {
		t.Fatalf("arrow item after ZC_EQUIP_ARROW = %+v, want equipped ammo location 0x%04X", item, db.EquipAmmo)
	}
}

func TestInventoryEquipmentRebuildsLocalWeaponAppearanceFromEquippedItem(t *testing.T) {
	sessionState := &session.Session{
		CharID:   150000,
		Selected: session.Character{ID: 150000, Job: 2, Weapon: 10},
		Characters: []session.Character{
			{ID: 150000, Job: 2, Weapon: 10},
		},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, Job: 2, Weapon: 10}
	ctx := client.Context{Session: sessionState, World: world}

	applyInventoryItemList(ctx, []network.InventoryItem{
		{Index: 2, ItemID: 1607, Type: 5, Location: 0x0002, WearLocation: 0x0002, Equip: true, Equipped: true, Identified: true},
	})

	if sessionState.Selected.Weapon != 1607 || sessionState.Selected.Shield != 0 {
		t.Fatalf("selected weapon = %d shield = %d, want 1607/0", sessionState.Selected.Weapon, sessionState.Selected.Shield)
	}
	if sessionState.Characters[0].Weapon != 1607 {
		t.Fatalf("character list weapon = %d, want 1607", sessionState.Characters[0].Weapon)
	}
	if world.Player.Weapon != 1607 || world.Player.Shield != 0 {
		t.Fatalf("world player weapon = %d shield = %d, want 1607/0", world.Player.Weapon, world.Player.Shield)
	}
}

func TestApplyStoragePacketsUpdateSessionStorage(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyStorageAmount(ctx, network.StorageAmount{Amount: 1, MaxAmount: 300})
	applyStorageItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Storage.Open {
		t.Fatal("storage was not marked open")
	}
	if sessionState.Storage.Amount != 1 || sessionState.Storage.MaxAmount != 300 {
		t.Fatalf("storage counts = %d/%d", sessionState.Storage.Amount, sessionState.Storage.MaxAmount)
	}
	if len(sessionState.Storage.Items) != 1 || sessionState.Storage.Items[0].ItemID != 512 || sessionState.Storage.Items[0].Amount != 4 {
		t.Fatalf("storage items = %+v", sessionState.Storage.Items)
	}

	applyStorageItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Storage.Items[0].Amount; got != 7 {
		t.Fatalf("storage amount after replace = %d, want 7", got)
	}
	applyStorageItemRemoved(ctx, network.StorageItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Storage.Items[0].Amount; got != 5 {
		t.Fatalf("storage amount after remove = %d, want 5", got)
	}
	applyStorageClosed(ctx)
	if sessionState.Storage.Open || len(sessionState.Storage.Items) != 0 {
		t.Fatalf("storage after close = %+v", sessionState.Storage)
	}
}

func TestApplyCartPacketsUpdateSessionCart(t *testing.T) {
	sessionState := &session.Session{}
	ctx := client.Context{Session: sessionState}

	applyCartAmount(ctx, network.CartAmount{Amount: 1, MaxAmount: 100, Weight: 450, MaxWeight: 80000})
	applyCartItemList(ctx, []network.InventoryItem{{Index: 3, ItemID: 512, Type: 0, Amount: 4, Identified: true}})
	if !sessionState.Cart.Open {
		t.Fatal("cart was not marked open")
	}
	if sessionState.Cart.Amount != 1 || sessionState.Cart.MaxAmount != 100 || sessionState.Cart.Weight != 450 || sessionState.Cart.MaxWeight != 80000 {
		t.Fatalf("cart counts = %+v", sessionState.Cart)
	}
	if len(sessionState.Cart.Items) != 1 || sessionState.Cart.Items[0].ItemID != 512 || sessionState.Cart.Items[0].Amount != 4 {
		t.Fatalf("cart items = %+v", sessionState.Cart.Items)
	}

	applyCartItemAdded(ctx, network.InventoryItem{Index: 3, ItemID: 512, Type: 0, Amount: 7, Identified: true})
	if got := sessionState.Cart.Items[0].Amount; got != 7 {
		t.Fatalf("cart amount after replace = %d, want 7", got)
	}
	applyCartItemRemoved(ctx, network.CartItemRemoved{Index: 3, Amount: 2})
	if got := sessionState.Cart.Items[0].Amount; got != 5 {
		t.Fatalf("cart amount after remove = %d, want 5", got)
	}
	applyCartClosed(ctx)
	if sessionState.Cart.Open {
		t.Fatalf("cart after close = %+v", sessionState.Cart)
	}
}

func TestAttackFocusTracksTargetAndAnimationStart(t *testing.T) {
	mode := &WorldMode{}
	first := time.Unix(10, 0)
	second := time.Unix(20, 0)

	mode.focusAttackTarget(100, first)
	if mode.attackFocusID != 100 || !mode.attackFocusStart.Equal(first) {
		t.Fatalf("first focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.focusAttackTarget(100, second)
	if !mode.attackFocusStart.Equal(first) {
		t.Fatalf("same target reset animation start to %v", mode.attackFocusStart)
	}

	mode.focusAttackTarget(200, second)
	if mode.attackFocusID != 200 || !mode.attackFocusStart.Equal(second) {
		t.Fatalf("second focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}

	mode.clearAttackFocus()
	if mode.attackFocusID != 0 || !mode.attackFocusStart.IsZero() {
		t.Fatalf("clear focus = id %d start %v", mode.attackFocusID, mode.attackFocusStart)
	}
}
