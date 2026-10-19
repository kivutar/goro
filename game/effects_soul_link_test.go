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
	for _, statusFirst := range []bool{false, true} {
		ctx := soulLinkerEffectContext()
		m := &WorldMode{}
		status := network.StatusEffectChange{StatusID: db.StatusSwoo, ActorID: 300, Active: true}
		if statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
		m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillSLSwoo, SourceID: 2000000, TargetID: 300, Result: 1})
		if !statusFirst {
			m.applyStatusEffectChange(ctx, status)
		}
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
			t.Fatalf("statusFirst=%v: %d shrink effects", statusFirst, count)
		}
		if tint := actorStateTint(ctx.World.Actors[300]); tint != (color.RGBA{255, 255, 255, 255}) {
			t.Fatalf("monster got Overthrust tint: %v", tint)
		}
		status.Active = false
		m.applyStatusEffectChange(ctx, status)
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
