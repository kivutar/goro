package game

import (
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
)

func soulLinkerSkillPacket(skill uint16, result byte) network.Packet {
	var w network.Writer
	w.Uint16(0x011A)
	w.Uint16(skill)
	w.Uint16(1)
	w.Uint32(300)
	w.Uint32(2000000)
	w.Uint8(result)
	return network.Packet{ID: 0x011A, Data: w.Bytes()}
}

func npcEffectUpdatePacket(id, opt3 uint32) network.Packet {
	var w network.Writer
	w.Uint16(network.PacketZCNPCShowEFSTUpdate)
	w.Uint32(id)
	w.Uint32(0)
	w.Uint32(75)
	w.Uint32(opt3)
	return network.Packet{ID: network.PacketZCNPCShowEFSTUpdate, Data: w.Bytes()}
}

func TestAllSoulLinkerSkillsHaveCastAndActivationEffects(t *testing.T) {
	active, links := 0, 0
	for skill := uint16(1); skill <= db.SkillSLHigh; skill++ {
		if !db.IsSoulLinkerActiveSkill(skill) {
			continue
		}
		active++
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, DelayTime: 1500})
		if len(m.scheduledSounds) != 1 || !m.hasActiveWorldEffect(effectBlueCasting, 2000000, time.Now()) {
			t.Fatalf("skill %d lacks blue casting or plays duplicate sounds: %+v", skill, m.scheduledSounds)
		}
		for _, id := range append(skillEffectIDs(skill), skillSuccessEffectIDs(skill)...) {
			if _, ok := worldEffectSpecForID(id); !ok {
				t.Fatalf("skill %d has missing effect %d", skill, id)
			}
		}
		if !db.IsSoulLinkSkill(skill) {
			continue
		}
		links++
		m = &WorldMode{}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Result: 1})
		if len(m.worldEffects) != 3 || len(m.scheduledSounds) != 2 {
			t.Fatalf("link %d: %+v %+v", skill, m.worldEffects, m.scheduledSounds)
		}
		if m.scheduledSounds[1].at.Sub(m.scheduledSounds[0].at) != 145*db.EffectFrameDuration {
			t.Fatal("Soul Link sound timing")
		}
		anim := m.actorAnims[150000]
		if anim.actionFamily != spriteActionPCSkill || anim.fixedMotion != 3 || !anim.hasFixedMotion {
			t.Fatalf("link %d caster pose: %+v", skill, anim)
		}
	}
	if active != 26 || links != 16 || db.IsSoulLinkerActiveSkill(db.SkillSLKaina) {
		t.Fatalf("active=%d links=%d; Kaina is passive", active, links)
	}
}

func TestSoulLinkAuraFollowsStatusAndKeepsPlayerColors(t *testing.T) {
	for _, statusFirst := range []bool{false, true} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		status := network.StatusEffectChange{StatusID: db.StatusSoullink, ActorID: 2000000, Active: true}
		if statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSLMonk, SourceID: 2000000, TargetID: 2000000, Result: 1})
		if !statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		count := 0
		var start time.Time
		for _, e := range m.worldEffects {
			if e.effectID == effectAsuraBody {
				count++
				start = e.starts
				if !e.persistent || !e.expires.IsZero() {
					t.Fatal("aura must last until status removal")
				}
			}
		}
		if count != 1 {
			t.Fatalf("statusFirst=%v: %d auras", statusFirst, count)
		}
		if actorStateTint(ctx.World.Player) != (color.RGBA{255, 255, 255, 255}) {
			t.Fatal("Soul Link must preserve base sprite color")
		}
		frame := render.NewFrame(800, 600)
		billboard := &spriteBillboard{image: render.NewImage(64, 64), anchorX: 32, anchorY: 60}
		entry := sceneActorDrawEntry{actor: ctx.World.Player, worldX: 10.5, worldY: 20.5}
		m.drawActorSpriteWithEffects3D(ctx, frame, newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0), entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, start.Add(time.Minute))
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() != 3 {
			t.Fatalf("want body and two silhouettes, got %d", commands.Len())
		}
		if got := commands.Index(1).FieldByName("ColorA").Float(); math.Abs(got-200.0/255) > 1e-6 {
			t.Fatalf("wrong aura alpha: %v", got)
		}
		status.Active = false
		m.applyStatusEffectChange(ctx, status)
		if m.hasActiveWorldEffect(effectAsuraBody, 150000, time.Now()) {
			t.Fatal("aura survived status removal")
		}
	}
}

func TestEswooShrinksUntilServerClearsIt(t *testing.T) {
	for _, snapshotFirst := range []bool{false, true} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		if snapshotFirst {
			m.handleNetworkPacket(ctx, npcEffectUpdatePacket(300, db.Opt3Overthrust), time.Now())
		}
		// rAthena does not send the SWOO status icon to monsters.
		m.handleNetworkPacket(ctx, soulLinkerSkillPacket(db.SkillSLSwoo, 1), time.Now())
		count := 0
		for _, e := range m.worldEffects {
			if e.effectID == effectBabyBody {
				count++
				for _, tc := range []struct {
					age  time.Duration
					size float64
				}{{0, 1}, {45 * db.EffectFrameDuration, .5}, {time.Minute, .5}} {
					if got := m.actorBodySizeMultiplier(300, e.starts.Add(tc.age)); math.Abs(got-tc.size) > 1e-8 {
						t.Fatalf("Eswoo size at %v = %v", tc.age, got)
					}
				}
			}
		}
		if count != 1 {
			t.Fatalf("snapshotFirst=%v: %d shrink effects", snapshotFirst, count)
		}
		if tint := actorStateTint(ctx.World.Actors[300]); tint != (color.RGBA{255, 255, 255, 255}) {
			t.Fatalf("monster got Overthrust tint: %v", tint)
		}
		m.handleNetworkPacket(ctx, npcEffectUpdatePacket(300, 0), time.Now())
		for _, e := range m.worldEffects {
			if e.effectID == effectBabyBodyBack {
				if got := m.actorBodySizeMultiplier(300, e.starts); got != .5 {
					t.Fatalf("return starts at %v", got)
				}
				if got := m.actorBodySizeMultiplier(300, e.starts.Add(15*db.EffectFrameDuration)); got != 1 {
					t.Fatalf("return ends at %v", got)
				}
			}
		}
		if !m.hasActiveWorldEffect(effectBabyBodyBack, 300, time.Now()) {
			t.Fatal("missing return animation")
		}
		m.removeActorEffectStateEffects(300)
		if m.hasActiveWorldEffect(effectBabyBodyBack, 300, time.Now()) {
			t.Fatal("size effect survived actor leaving")
		}
	}
}

func TestMonsterEffectUpdatesPreserveActorDuringLoginAndPlay(t *testing.T) {
	for _, login := range []bool{false, true} {
		ctx := soulLinkerEffectContext()
		actor := ctx.World.Actors[300]
		actor.BodyState, actor.HealthState = db.BodyStateStun, db.HealthStatePoison
		actor.HasState = true
		actor.Moving, actor.MoveStarted = true, time.Now().Add(-time.Second)
		ctx.World.Actors[300] = actor
		m := &WorldMode{}
		packet := npcEffectUpdatePacket(300, db.Opt3Overthrust|db.Opt3Energycoat|db.Opt3Steelbody)
		if login {
			if !(&LoginMode{}).applyLoginActorBootstrapPacket(ctx, packet) {
				t.Fatal("login dropped the monster effect snapshot")
			}
			m.syncCurrentActorEffectStateEffects(ctx)
		} else {
			m.handleNetworkPacket(ctx, packet, time.Now())
		}
		actor.Opt3State = db.Opt3Overthrust | db.Opt3Energycoat | db.Opt3Steelbody
		actor.Level, actor.HasLevel = 75, true
		if !reflect.DeepEqual(ctx.World.Actors[300], actor) {
			t.Fatalf("partial snapshot changed other actor data: %+v", ctx.World.Actors[300])
		}
		for _, id := range []int{effectBabyBody, effectSteelBody} {
			if !m.hasActiveWorldEffect(id, 300, time.Now().Add(time.Hour)) {
				t.Fatalf("missing persistent effect %d", id)
			}
		}
		m.handleNetworkPacket(ctx, npcEffectUpdatePacket(300, 0), time.Now())
		if ctx.World.Actors[300].Opt3State != 0 || m.hasActiveWorldEffect(effectSteelBody, 300, time.Now()) || m.hasActiveWorldEffect(effectBabyBody, 300, time.Now()) {
			t.Fatal("monster effects survived the authoritative clear")
		}
		m.handleNetworkPacket(ctx, npcEffectUpdatePacket(999, db.Opt3Overthrust), time.Now())
		if _, ok := ctx.World.Actors[999]; ok {
			t.Fatal("effect update created an unknown actor")
		}
	}
}

func TestSoulLinkerMonsterSkillSuccessOwnsActivation(t *testing.T) {
	for _, skill := range []uint16{db.SkillSLSwoo, db.SkillSLSke, db.SkillSLSka} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		m.handleNetworkPacket(ctx, soulLinkerSkillPacket(skill, 0), time.Now())
		if ctx.World.Actors[300].Opt3State != 0 || len(m.worldEffects) != 0 {
			t.Fatalf("failed skill %d changed monster effects", skill)
		}
		m.handleNetworkPacket(ctx, soulLinkerSkillPacket(skill, 1), time.Now())
		if ctx.World.Actors[300].Opt3State == 0 {
			t.Fatalf("skill %d still requires a status-icon packet", skill)
		}
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Job: 1002, X: 18, Y: 20, HasState: true, HasOpt3State: true})
		if m.hasActiveWorldEffect(effectBabyBody, 300, time.Now()) || m.hasActiveWorldEffect(effectSteelBody, 300, time.Now()) || m.hasActiveWorldEffect(effectAsuraBodyMonster, 300, time.Now()) {
			t.Fatalf("skill %d ignored a newer actor snapshot", skill)
		}
	}
}

func TestKaupeRendersEveryDodge(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	var w network.Writer
	w.Uint16(network.PacketZCNotifyEffect2)
	w.Uint32(300)
	w.Uint32(effectStormKick4)
	packet := network.Packet{ID: network.PacketZCNotifyEffect2, Data: w.Bytes()}
	m.handleNetworkPacket(ctx, packet, time.Now())
	first := m.worldEffects[0]
	m.handleNetworkPacket(ctx, packet, time.Now())
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 2 || m.worldEffects[0] != first || m.worldEffects[0].soulLights == m.worldEffects[1].soulLights {
		t.Fatal("successive dodges must have independent effects and sounds")
	}
}

func TestSoulLinkAndKaupeParticleTimingIndependentOfFrameRate(t *testing.T) {
	ctx := soulLinkerEffectContext()
	for _, id := range []int{effectSoulLink, effectStormKick4} {
		m := &WorldMode{}
		soulLinkerTestTextures(m)
		start := time.Unix(100, 0)
		m.addWorldEffectAt(ctx, id, 300, start)
		e := m.worldEffects[0]
		one, many := &soulLights{}, &soulLights{}
		right := modelPoint3{x: 1}
		one.advance(ctx, e, right, start.Add(100*db.EffectFrameDuration))
		for i := 0; i <= 100; i++ {
			many.advance(ctx, e, right, start.Add(time.Duration(i)*db.EffectFrameDuration))
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatalf("effect %d depends on render rate", id)
		}
		projection := newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, start.Add(30*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		e.soulLights.advance(ctx, e, right, start.Add(300*db.EffectFrameDuration))
		if len(e.soulLights.particles) != 0 {
			t.Fatalf("effect %d particles never die", id)
		}
		if id == effectStormKick4 && (len(m.scheduledSounds) != 1 || m.scheduledSounds[0].at.Sub(start) != 15*db.EffectFrameDuration) {
			t.Fatal("Kaupe dodge sound timing")
		}
	}
	for _, tc := range []struct {
		age, layer int
		alpha      float64
	}{{0, 10, 0}, {19, 10, 240.0 / 255}, {100, 10, 240.0 / 255}, {148, 10, 0}, {129, 8, 0}} {
		_, a := soulLinkGlyph(tc.age, tc.layer)
		if math.Abs(a-tc.alpha) > 1e-8 {
			t.Fatalf("glyph age=%d layer=%d alpha=%v", tc.age, tc.layer, a)
		}
	}
}

func TestBlueCastingFlaresDuringCastAndEndsWithIt(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	soulLinkerTestTextures(m)
	start := time.Unix(100, 0)
	for _, duration := range []int{6, 60, 180} {
		m.worldEffects = nil
		m.addWorldEffectBetweenAtDuration(ctx, effectBlueCasting, 300, 0, start, time.Duration(duration)*db.EffectFrameDuration)
		e := m.worldEffects[0]
		duration = max(duration, 56)
		if e.expires.Sub(e.starts) != time.Duration(duration)*db.EffectFrameDuration {
			t.Fatal("short casts need the native minimum flare duration")
		}
		spec, _ := worldEffectSpecForID(effectBlueCasting)
		frame := render.NewFrame(800, 600)
		m.drawBlueCastingEffect(frame, ctx, spec.components[0], e, 18.5, 20.5, 0, start.Add(15*db.EffectFrameDuration))
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 1 {
			t.Fatal("missing cast rings")
		}
		one, many := &blueCasting{}, &blueCasting{}
		one.advance(duration, duration)
		for i := 0; i <= duration; i++ {
			many.advance(i, duration)
		}
		if !reflect.DeepEqual(one, many) {
			t.Fatal("cast rings depend on render rate")
		}
		frame = render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, newSceneProjectionForTarget(800, 600, 18.5, 20.5, 0), start.Add(time.Duration(duration+1)*db.EffectFrameDuration), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldCommands").Len() != 0 || len(m.worldEffects) != 0 {
			t.Fatal("rings survived the cast")
		}
	}
}

func TestEskaDoesNotPlayFurySound(t *testing.T) {
	ctx := soulLinkerEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSLSka, SourceID: 2000000, TargetID: 300, Result: 1})
	if len(m.scheduledSounds) != 2 {
		t.Fatalf("Eska sounds: %+v", m.scheduledSounds)
	}
	for i, want := range []string{"effect\\t_방어형.wav", "effect\\mon_금강불괴.wav"} {
		if !reflect.DeepEqual(m.scheduledSounds[i].paths, []string{want}) {
			t.Fatalf("Eska sounds: %+v", m.scheduledSounds)
		}
	}
}
