package game

import (
	"encoding/binary"
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	worldstate "github.com/kivutar/goro/world"
)

func monkEffectContext() client.Context {
	w := worldstate.New()
	w.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: db.JobMonk}
	w.Actors[300] = worldstate.Actor{ID: 300, X: 12, Y: 20, Job: db.JobMonk, Appearance: true}
	return client.Context{World: w, Session: &session.Session{
		AccountID: 2000000, CharID: 150000, Selected: session.Character{ID: 150000, Job: db.JobMonk},
	}}
}

func spiritSpherePacket(id uint16, actorID uint32, count uint16) network.Packet {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data, id)
	binary.LittleEndian.PutUint32(data[2:], actorID)
	binary.LittleEndian.PutUint16(data[6:], count)
	return network.Packet{ID: id, Data: data}
}

func TestCallSpiritsFinishesStandingWithoutAnotherSkillAnimation(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOCallspirits, DelayTime: 1000})
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOCallspirits, Result: 1})
	if anim := m.actorAnims[150000]; anim.actionFamily != spriteActionIdle {
		t.Fatalf("Call Spirits result animation = %+v, want standing", anim)
	}
	if ctx.World.Player.SpiritSpheres != 0 {
		t.Fatal("skill result must not invent a sphere count")
	}
}

func TestFuryHasOnlyActivationSound(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOExplosionspirits, DelayTime: 1000})
	if len(m.scheduledSounds) != 0 {
		t.Fatalf("Fury cast scheduled unwanted sound: %+v", m.scheduledSounds)
	}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOExplosionspirits, Result: 1})
	if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{"effect\\mon_폭기.wav"}) {
		t.Fatalf("Fury activation sounds = %+v", m.scheduledSounds)
	}
}

func TestSpiritSpherePacketsUpdateCountsAndLifecycle(t *testing.T) {
	for _, actorID := range []uint32{2000000, 300} {
		ctx := monkEffectContext()
		m := &WorldMode{}
		for _, count := range []uint16{1, 3, 3, 5, 2, 0, 1} {
			m.handleNetworkPacket(ctx, spiritSpherePacket(network.PacketZCSpirits, actorID, count), time.Now())
			actor, _, _ := actorForCombatID(ctx, actorID)
			if actor.SpiritSpheres != count || !actor.HasSpiritSpheres {
				t.Fatalf("actor %d count=%d, want %d", actorID, actor.SpiritSpheres, count)
			}
			if m.hasActiveWorldEffect(effectChookgi, actorID, time.Now().Add(time.Hour)) != (count > 0) {
				t.Fatalf("sphere effect does not follow count %d: %+v", count, m.worldEffects)
			}
			if count > 0 && len(m.worldEffects) != 1 {
				t.Fatal("sphere updates duplicate the effect")
			}
		}
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: actorID, EffectState: db.EffectStateCloak})
		if m.hasActiveWorldEffect(effectChookgi, actorID, time.Now()) {
			t.Fatal("spheres reveal a cloaked actor")
		}
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: actorID})
		if !m.hasActiveWorldEffect(effectChookgi, actorID, time.Now()) {
			t.Fatal("spheres did not return on reveal")
		}
		m.applyActorVanish(ctx, network.ActorVanish{ID: actorID})
		if m.hasActiveWorldEffect(effectChookgi, actorID, time.Now()) {
			t.Fatal("sphere effect survived actor removal")
		}
	}
}

func TestSpiritEffectsSurviveLoginAndActorRefresh(t *testing.T) {
	ctx := monkEffectContext()
	login := &LoginMode{}
	for _, id := range []uint32{2000000, 300} {
		if !login.applyLoginActorBootstrapPacket(ctx, spiritSpherePacket(network.PacketZCSpirits2, id, 3)) {
			t.Fatal("login dropped sphere packet")
		}
		login.applyLoginStatusEffectPacket(ctx, testStatusEffectChangePacket(db.StatusExplosionspirits, id, true))
	}
	m := &WorldMode{}
	m.syncCurrentActorEffectStateEffects(ctx)
	if len(m.worldEffects) != 4 {
		t.Fatalf("login effects = %+v, want spheres and Fury for both actors", m.worldEffects)
	}
	original := append([]worldEffect(nil), m.worldEffects...)
	m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, X: 13, Y: 20, Job: db.JobMonk, Appearance: true, HasState: true})
	// Account and character IDs refer to the same actor, without duplicating its aura.
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 150000, Active: true})
	if !reflect.DeepEqual(m.worldEffects, original) || ctx.World.Actors[300].SpiritSpheres != 3 {
		t.Fatal("refresh restarted or duplicated state effects")
	}
	for _, id := range []uint32{2000000, 300} {
		m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: id, Active: false})
		if m.hasActiveWorldEffect(effectGumgang, id, time.Now()) {
			t.Fatal("Fury aura survived status removal")
		}
	}
	// A new WorldMode is created on a map transition, retaining local state.
	next := &WorldMode{}
	next.syncCurrentActorEffectStateEffects(ctx)
	if !next.hasActiveWorldEffect(effectChookgi, 2000000, time.Now()) {
		t.Fatal("map transition lost local spheres")
	}
}

func TestMonkBuffsFollowActorEntrySnapshots(t *testing.T) {
	for _, job := range []int16{db.JobMonk, db.JobSuperNovice} {
		ctx := monkEffectContext()
		delete(ctx.World.Actors, 300)
		m := &WorldMode{}
		data := make([]byte, 60)
		binary.LittleEndian.PutUint32(data[2:], 300)
		binary.LittleEndian.PutUint16(data[16:], uint16(job))
		applyEntry := func(state uint32) {
			t.Helper()
			binary.LittleEndian.PutUint32(data[44:], state)
			entry, ok, err := network.ParseActorEntry(network.Packet{ID: 0x02EE, Data: data})
			if err != nil || !ok {
				t.Fatalf("parse entry: ok=%t err=%v", ok, err)
			}
			m.upsertNetworkActor(ctx, entry)
		}
		assertAuras := func(active bool) {
			t.Helper()
			for _, id := range []int{effectGumgang, effectSteelBody} {
				if m.hasActiveWorldEffect(id, 300, time.Now().Add(time.Hour)) != active {
					t.Fatalf("job %d effect %d active != %t", job, id, active)
				}
			}
		}
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		assertAuras(true)
		original := append([]worldEffect(nil), m.worldEffects...)
		applyEntry(db.Opt3Explosionspirits | db.Opt3Steelbody)
		m.upsertNetworkActor(ctx, network.ActorEntry{ID: 300, Moving: true, ToX: 15, ToY: 20})
		assertAuras(true)
		if !reflect.DeepEqual(m.worldEffects, original) {
			t.Fatal("actor refresh restarted the auras")
		}
		applyEntry(0)
		assertAuras(false)
	}
}

func TestRuwachUsesSightSoundOncePerActivation(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	for _, state := range []uint32{db.EffectStateRuwach, db.EffectStateRuwach, db.EffectStateRuwach | db.EffectStateFalcon} {
		m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: state})
	}
	if len(m.scheduledSounds) != 1 || !reflect.DeepEqual(m.scheduledSounds[0].paths, []string{"effect\\ef_sight.wav"}) {
		t.Fatalf("Ruwach sounds = %+v", m.scheduledSounds)
	}
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300})
	m.applyActorStateChange(ctx, network.ActorStateChange{ID: 300, EffectState: db.EffectStateRuwach})
	if len(m.scheduledSounds) != 2 || len(m.worldEffects) != 1 {
		t.Fatal("Ruwach did not restart on a new activation")
	}
}

func TestGumgangFlashesMatchOriginalUpdateLoop(t *testing.T) {
	for i, pulse := range []int{70, 130, 190, 250} {
		counter, cycle := 10, 0
		for frame := 1; frame <= 200; frame++ {
			counter++
			if counter < 10 {
				pulse += 25
			} else {
				pulse -= 12
				if pulse < 12 {
					counter, pulse = 0, 0
					cycle++
				}
			}
			alpha, gotCycle := gumgangFlash(frame, i)
			if math.Abs(alpha-float64(pulse)/255) > 1e-9 || gotCycle != cycle {
				t.Fatalf("flash %d frame %d: alpha=%g cycle=%d, want %d/255 cycle=%d", i, frame, alpha, gotCycle, pulse, cycle)
			}
		}
	}
}

func TestMonkStateEffectsDrawExactCountsAndKeepLooping(t *testing.T) {
	ctx := monkEffectContext()
	ctx.Resources = &res.Manager{}
	m := &WorldMode{textures: make(map[string]*render.Image)}
	for _, id := range []int{effectChookgi, effectGumgang} {
		spec, _ := worldEffectSpecForID(id)
		for _, component := range spec.components {
			for _, path := range append(component.textureFiles, component.textureFile) {
				m.textures["__effectfile_"+path+"|blackkey=true"] = render.NewImage(8, 8)
			}
		}
	}
	projection := newSceneProjectionForTarget(800, 600, 12.5, 20.5, 0)
	for _, count := range []uint16{1, 3, 5, 0} {
		m.handleNetworkPacket(ctx, spiritSpherePacket(network.PacketZCSpirits, 300, count), time.Now())
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, time.Now().Add(time.Hour), false)
		if got := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len(); got != int(count)*2 {
			t.Fatalf("count %d drew %d billboards, want %d", count, got, count*2)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 300, Active: true})
	for _, elapsed := range []time.Duration{0, time.Second, time.Hour} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, m.worldEffects[0].starts.Add(elapsed), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
		if commands.Len() < 3 || commands.Len() > 4 {
			t.Fatalf("Fury at %s drew %d flashes, want 3 or 4", elapsed, commands.Len())
		}
		if commands.Index(0).FieldByName("ColorB").Float() != 1 {
			t.Fatal("Fury must use the original blue tint")
		}
	}
}

func TestMonkEffectsRenderWithOldROAssets(t *testing.T) {
	ctx := monkEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	m.handleNetworkPacket(ctx, spiritSpherePacket(network.PacketZCSpirits, 300, 3), time.Now())
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusExplosionspirits, ActorID: 300, Active: true})
	projection := newSceneProjectionForTarget(800, 600, 12.5, 20.5, 0)
	for _, elapsed := range []time.Duration{time.Second, 20 * time.Second, time.Hour} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, m.worldEffects[0].starts.Add(elapsed), false)
		count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
		if count < 9 || len(m.textureMiss) != 0 {
			t.Fatalf("Monk effects at %s: %d billboards, missing=%v", elapsed, count, m.textureMiss)
		}
	}
}

func TestSteelBodyAuraPacketOrderAndLifetime(t *testing.T) {
	for _, statusFirst := range []bool{true, false} {
		ctx := monkEffectContext()
		m := &WorldMode{}
		status := func(active bool) {
			m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSteelbody, ActorID: 300, Active: active})
		}
		if statusFirst {
			status(true)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 300, TargetID: 300, SkillID: db.SkillMOSteelbody, Result: 1})
		if !statusFirst {
			status(true)
		}
		status(true)
		if len(m.scheduledSounds) != 1 {
			t.Fatalf("statusFirst=%t: duplicate activation sounds: %+v", statusFirst, m.scheduledSounds)
		}
		auras := 0
		for _, effect := range m.worldEffects {
			if effect.effectID == effectSteelBody {
				auras++
				if effect.expired(time.Now().Add(time.Hour)) || !effect.persistent {
					t.Fatal("Steel Body aura does not follow status lifetime")
				}
			}
		}
		if auras != 1 || !m.hasActiveWorldEffect(db.SkillEffectSteelBodyRing, 300, time.Now()) {
			t.Fatalf("Steel Body effects = %+v", m.worldEffects)
		}
		status(false)
		if m.hasActiveWorldEffect(effectSteelBody, 300, time.Now()) {
			t.Fatal("Steel Body aura survived status removal")
		}
	}
}

func TestSteelBodyDoesNotReviveExpiredActivation(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	m.addWorldEffectAt(ctx, effectSteelBody, 2000000, time.Now().Add(-time.Minute))
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusSteelbody, ActorID: 150000, Active: true})
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 150000, TargetID: 150000, SkillID: db.SkillMOSteelbody, Result: 1})
	active := 0
	for _, effect := range m.worldEffects {
		if effect.effectID == effectSteelBody && !effect.expired(time.Now()) {
			active++
			if effect.actorID != 2000000 || !effect.persistent {
				t.Fatal("local aliases duplicated the aura")
			}
		}
	}
	if active != 1 {
		t.Fatalf("active Steel Body auras = %d, want 1", active)
	}
}

func TestMonkSkillResultsUseDedicatedPoses(t *testing.T) {
	for _, tc := range []struct {
		skill         uint16
		family, frame int
		play          bool
	}{
		{db.SkillMOInvestigate, spriteActionPCSkill, 3, true},
		{db.SkillMOFingeroffensive, spriteActionPCSkill, 3, true},
		{db.SkillMOChaincombo, spriteActionPCSkill, 5, false},
		{db.SkillMOCombofinish, spriteActionPCSkill, 1, true},
		{db.SkillMOTripleattack, spriteActionPCAttack3, 0, true},
	} {
		ctx := monkEffectContext()
		m := &WorldMode{}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: tc.skill, SkillLevel: 5, SourceSpeed: 800, Damage: 100, HitCount: 1, Action: network.ActorActionSkill})
		anim := m.actorAnims[150000]
		if anim.actionFamily != tc.family || anim.frameOffset != tc.frame || anim.play != tc.play {
			t.Fatalf("skill %d pose = %+v", tc.skill, anim)
		}
		if _, active := m.actorAnimation(150000, anim.started.Add(2*time.Second)); active {
			t.Fatalf("skill %d pose never released", tc.skill)
		}
	}
}

func TestMonkTwoFrameSkillsKeepTheirOriginalTiming(t *testing.T) {
	for _, skillID := range []uint16{db.SkillMOInvestigate, db.SkillMOFingeroffensive} {
		for _, sourceSpeed := range []int32{80, 2000} {
			ctx := monkEffectContext()
			m := &WorldMode{}
			m.applyActorActionNotify(ctx, network.ActorActionNotify{
				SourceID: 2000000, TargetID: 300, SkillID: skillID, SourceSpeed: sourceSpeed,
				Action: network.ActorActionSkill, HitCount: 1, Damage: 100,
			})
			started := m.actorAnims[150000].started
			action := res.ACTAction{DelayMS: 100, Animations: make([]res.ACTAnimation, 8)}
			for _, tc := range []struct {
				elapsed time.Duration
				frame   int
			}{
				{0, 3}, {150 * time.Millisecond, 3}, {170 * time.Millisecond, 4}, {1099 * time.Millisecond, 4},
			} {
				anim, active := m.actorAnimation(150000, started.Add(tc.elapsed))
				if !active {
					t.Fatalf("skill %d ended at %s with source speed %d", skillID, tc.elapsed, sourceSpeed)
				}
				state := spriteState{}
				applyActorAnimationToSpriteState(&state, anim, true)
				if frame := bodyMotionForState(action, state, started, started.Add(tc.elapsed)); frame != tc.frame {
					t.Fatalf("skill %d at %s: frame=%d, want %d", skillID, tc.elapsed, frame, tc.frame)
				}
			}
			if _, active := m.actorAnimation(150000, started.Add(1100*time.Millisecond)); active {
				t.Fatalf("skill %d did not release its pose after 1100ms", skillID)
			}
		}
	}
}

func TestBladeStopWaitingPoseExpires(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 2000000, SkillID: db.SkillMOBladestop, Result: 1})
	anim := m.actorAnims[150000]
	if anim.actionFamily != spriteActionPCSkill || anim.fixedMotion != 3 || anim.play {
		t.Fatalf("Blade Stop waiting pose = %+v", anim)
	}
	if _, active := m.actorAnimation(150000, anim.started.Add(2*time.Second)); !active {
		t.Fatal("Blade Stop waiting pose ended too soon")
	}
	if _, active := m.actorAnimation(150000, anim.started.Add(3*time.Second)); active {
		t.Fatal("Blade Stop waiting pose did not expire")
	}
}

func TestThrowSpiritSphereSizeUsesSkillLevel(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	starts := time.Now()
	for _, level := range []uint16{1, 5} {
		m.addSkillEffect(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOFingeroffensive, SkillLevel: level}, starts, starts)
	}
	spec, _ := worldEffectSpecForID(effectTanji)
	component := spec.components[0]
	small, _ := effect3DSize(component, m.worldEffects[0], 0, 0.5, 0)
	large, _ := effect3DSize(component, m.worldEffects[1], 0, 0.5, 0)
	if math.Abs(large/small-4.0/2.4) > 1e-6 {
		t.Fatalf("sphere sizes = %g, %g; want original level scaling", small, large)
	}
}

func TestAsuraGlyphsBelongToCasterDuringCastOnly(t *testing.T) {
	for _, job := range []int16{db.JobMonk, db.JobMonkH} {
		ctx := monkEffectContext()
		ctx.World.Player.Job = job
		ctx.Session.Selected.Job = job
		m := &WorldMode{}
		m.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOExtremityfist, DelayTime: 3000})
		want := effectBeginAsura
		if job == db.JobMonkH {
			want = effectBeginAsura11
		}
		if !m.hasActiveWorldEffect(want, 2000000, time.Now()) || len(m.scheduledSounds) != 0 {
			t.Fatalf("Asura cast = %+v sounds=%+v", m.worldEffects, m.scheduledSounds)
		}
		m.applyActorActionNotify(ctx, network.ActorActionNotify{SourceID: 2000000, TargetID: 300, SkillID: db.SkillMOExtremityfist, Action: network.ActorActionSkill, Damage: 100, HitCount: 1})
		glyphs := 0
		for _, effect := range m.worldEffects {
			if effect.effectID == effectBeginAsura || effect.effectID == effectBeginAsura11 {
				glyphs++
				if effect.actorID != 2000000 {
					t.Fatal("Asura glyphs appeared on victim")
				}
			}
		}
		if glyphs != 1 {
			t.Fatalf("Asura replayed glyphs on impact: %d sequences", glyphs)
		}
	}
}

func TestBladeStopFreezesAndReleasesBothActors(t *testing.T) {
	ctx := monkEffectContext()
	m := &WorldMode{}
	now := time.Now()
	m.setActorAction(ctx, 300, actorAnimation{actionFamily: spriteActionPCAttack1, started: now, fixedMotion: 2, hasFixedMotion: true})
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 2000000, TargetID: 300, Active: true})
	if anim := m.actorAnims[150000]; anim.actionFamily != spriteActionPCSkill || anim.fixedMotion != 4 || anim.play {
		t.Fatalf("caster pose = %+v", anim)
	}
	if anim := m.actorAnims[300]; anim.actionFamily != spriteActionPCAttack1 || anim.play || !anim.hasFixedMotion || anim.fixedMotion != 2 {
		t.Fatalf("victim was put in another animation: %+v", anim)
	}
	for _, id := range []uint32{2000000, 300} {
		actor, _, _ := actorForCombatID(ctx, id)
		if tint := actorStateTint(actor); tint != (color.RGBA{R: 64, G: 64, B: 64, A: 255}) {
			t.Fatalf("Blade Stop tint = %v", tint)
		}
		if anim, ok := m.actorAnimation(id, time.Now().Add(time.Minute)); !ok || !anim.hasFixedMotion {
			t.Fatalf("actor %d resumed before Blade Stop release", id)
		}
	}
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 2000000, TargetID: 300})
	for _, id := range []uint32{2000000, 150000, 300} {
		actor, _, _ := actorForCombatID(ctx, id)
		if actor.Opt3State&db.Opt3Bladestop != 0 || !m.actorAnims[id].play || m.actorAnims[id].hasFixedMotion {
			t.Fatalf("actor %d still stopped after release", id)
		}
	}
}

func TestBladeStopCapturesVictimsPlayingFrame(t *testing.T) {
	ctx := monkEffectContext()
	actions := make([]res.ACTAction, 13*8)
	for i := range actions {
		actions[i] = res.ACTAction{DelayMS: 100, Animations: make([]res.ACTAnimation, 4)}
	}
	started := time.Now().Add(-150 * time.Millisecond)
	m := &WorldMode{playerView: &humanoidSpriteView{body: &spriteView{act: &res.ACT{Actions: actions}}, started: started}}
	m.setCombatActorAction(ctx, 2000000, actorAnimation{actionFamily: spriteActionPCAttack1, started: started, play: true, hasPlay: true, duration: time.Second})
	m.applyActorBladeStop(ctx, network.ActorBladeStop{SourceID: 300, TargetID: 2000000, Active: true})
	anim := m.actorAnims[150000]
	want := minInt(3, int(anim.started.Sub(started)/(100*time.Millisecond)))
	if want == 0 || anim.fixedMotion != want || anim.play {
		t.Fatalf("victim pose = %+v, want frozen frame %d", anim, want)
	}
	if later, ok := m.actorAnimation(150000, time.Now().Add(time.Minute)); !ok || later.fixedMotion != want {
		t.Fatal("Blade Stop victim resumed animation before release")
	}
}

func TestInvestigateSpriteRendersWithOldROAssets(t *testing.T) {
	ctx := monkEffectContext()
	ctx.Resources = realDataManager(t)
	m := &WorldMode{}
	starts := time.Now()
	m.addWorldEffectBetweenAt(ctx, effectChimto, 300, 2000000, starts)
	projection := newSceneProjectionForTarget(800, 600, 12.5, 20.5, 0)
	for _, elapsed := range []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond} {
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(elapsed), false)
		if reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len() == 0 {
			t.Fatalf("Investigate has no sprite at %s", elapsed)
		}
	}
}
