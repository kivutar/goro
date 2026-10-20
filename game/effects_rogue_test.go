package game

import (
	"image/color"
	"math"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	"github.com/kivutar/goro/world"
)

func rogueSkillContext() client.Context {
	w := world.New()
	w.Player = world.Actor{ID: 100, X: 10, Y: 20, Job: db.JobRogue}
	w.Actors[300] = world.Actor{ID: 300, X: 16, Y: 20, Job: 1002}
	return client.Context{World: w, Resources: &res.Manager{}, Session: &session.Session{AccountID: 2000000, CharID: 100}}
}

func TestRogueAttackOwnersAndTiming(t *testing.T) {
	ctx := rogueSkillContext()
	starts := time.Unix(100, 0)
	for _, skill := range []uint16{db.SkillRGBackstap, db.SkillRGIntimidate} {
		m := &WorldMode{}
		// The action effect also plays on a miss.
		action := network.ActorActionNotify{SkillID: skill, SourceID: 2000000, TargetID: 300, Damage: 0}
		m.addSkillEffect(ctx, action, starts, starts.Add(time.Second))
		m.addSkillHitEffect(ctx, action, starts.Add(time.Second))
		if len(m.worldEffects) != 1 {
			t.Fatalf("skill %d: effects = %+v", skill, m.worldEffects)
		}
		effect := m.worldEffects[0]
		if effect.actorID != action.SourceID || !effect.starts.Equal(starts) {
			t.Fatalf("skill %d must start on the caster at action time: %+v", skill, effect)
		}
		if skill == db.SkillRGBackstap {
			if effect.actorOrigin.x != 16.5 || effect.projectileDirection.x != 6 || effect.projectileDirection.z != 0 {
				t.Fatalf("Back Stab must start at the victim and continue away from the caster: %+v", effect)
			}
			if len(m.scheduledSounds) != 1 || !m.scheduledSounds[0].at.Equal(starts.Add(20*db.EffectFrameDuration)) {
				t.Fatalf("Back Stab sound = %+v", m.scheduledSounds)
			}
		} else {
			if effect.actorOrigin.x != 10.5 || len(m.scheduledSounds) != 2 ||
				!m.scheduledSounds[0].at.Equal(starts) || !m.scheduledSounds[1].at.Equal(starts.Add(35*db.EffectFrameDuration)) {
				t.Fatalf("Snatch origin or sounds: %+v, %+v", effect, m.scheduledSounds)
			}
		}
	}
}

func TestSightlessMindOneBurstForMultipleVictims(t *testing.T) {
	ctx := rogueSkillContext()
	m := &WorldMode{}
	m.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SkillID: db.SkillRGRaid, SourceID: 2000000, TargetID: 2000000, Result: 1})
	for _, target := range []uint32{300, 301, 302} {
		action := network.ActorActionNotify{SkillID: db.SkillRGRaid, SourceID: 2000000, TargetID: target, Damage: 50}
		m.addSkillEffect(ctx, action, time.Now(), time.Now())
		m.addSkillHitEffect(ctx, action, time.Now())
	}
	if len(m.worldEffects) != 1 || m.worldEffects[0].effectID != effectTeiHit3 || m.worldEffects[0].actorID != 2000000 {
		t.Fatalf("Raid must emit once per cast, not per victim: %+v", m.worldEffects)
	}
}

func TestRogueSuccessEffectsAndDivestSound(t *testing.T) {
	ctx := rogueSkillContext()
	for _, skill := range []uint16{db.SkillRGStealcoin, db.SkillRGStripweapon, db.SkillRGStripshield, db.SkillRGStriparmor, db.SkillRGStriphelm, db.SkillRGCloseconfine} {
		m := &WorldMode{}
		notify := network.SkillNoDamageNotify{SkillID: skill, SourceID: 2000000, TargetID: 300}
		m.applySkillNoDamageNotify(ctx, notify)
		if len(m.worldEffects) != 0 || len(m.scheduledSounds) != 0 {
			t.Fatalf("failed skill %d emitted effects", skill)
		}
		notify.Result = 1
		m.applySkillNoDamageNotify(ctx, notify)
		count := 1
		if skill == db.SkillRGStealcoin {
			count = 2
		}
		if len(m.worldEffects) != count || len(m.scheduledSounds) != 1 {
			t.Fatalf("skill %d: effects=%+v sounds=%+v", skill, m.worldEffects, m.scheduledSounds)
		}
		for _, effect := range m.worldEffects {
			if effect.actorID != 300 {
				t.Fatalf("skill %d effect on wrong actor: %+v", skill, effect)
			}
		}
		if skill >= db.SkillRGStripweapon && skill <= db.SkillRGStriphelm {
			sound := m.scheduledSounds[0]
			if !reflect.DeepEqual(sound.paths, []string{"effect\\복호격.wav"}) || !sound.at.Equal(m.worldEffects[0].starts.Add(6*db.EffectFrameDuration)) {
				t.Fatalf("divest sound = %+v", sound)
			}
		}
	}
}

func TestRogueParticlesNativePhases(t *testing.T) {
	starts := time.Unix(100, 0)
	for _, tc := range []struct {
		id, count, before, peak, end int
	}{
		{effectRogueCoin, 120, 17, 56, 108},
		{effectIntimidate, 80, 69, 108, 146},
		{effectTeiHit3, 80, 29, 49, 79},
	} {
		effect := worldEffect{effectID: tc.id, starts: starts, actorID: 100}
		visible := 0
		for i := 0; i < tc.count; i++ {
			for _, frame := range []int{tc.before, tc.end} {
				if _, tint := rogueBurstParticle(effect, i, frame); tint.A != 0 {
					t.Fatalf("effect %d particle %d visible outside its native phase at tick %d", tc.id, i, frame)
				}
			}
			corners, tint := rogueBurstParticle(effect, i, tc.peak)
			if tint.A > 0 {
				visible++
				for _, p := range corners {
					if !isFinite(p.x) || !isFinite(p.y) || !isFinite(p.z) {
						t.Fatal("non-finite particle geometry")
					}
				}
			}
		}
		if visible != tc.count {
			t.Fatalf("effect %d: %d visible particles, want %d", tc.id, visible, tc.count)
		}
	}
	effect := worldEffect{effectID: effectBackStab, starts: starts, projectileDirection: modelPoint3{x: 6}}
	for i := 0; i < 80; i++ {
		if _, _, _, alpha := backStabParticle(effect, i, 30); alpha != 0 {
			t.Fatal("Back Stab appeared before its delayed particle launch")
		}
		position, size, _, alpha := backStabParticle(effect, i, 49)
		if alpha <= 0 || position.x <= 0 || math.Abs(position.z) > position.x || size <= 0 {
			t.Fatalf("Back Stab must travel in the attack direction: %+v, %g, %d", position, size, alpha)
		}
		if _, _, _, alpha := backStabParticle(effect, i, 125); alpha != 0 {
			t.Fatal("Back Stab did not fade out")
		}
	}
}

func TestBackStab2008Opacity(t *testing.T) {
	effect := worldEffect{effectID: effectBackStab, starts: time.Unix(100, 0)}
	for i := 0; i < 80; i++ {
		for _, tc := range []struct{ age, alpha int }{{1, 0}, {2, 15}, {10, 135}, {11, 132}, {54, 3}, {55, 0}} {
			frame := tc.age + 39 - hitEffectRandom(effect, i*5, 10)
			_, _, _, alpha := backStabParticle(effect, i, frame)
			if alpha != tc.alpha {
				t.Fatalf("particle %d at age %d: alpha = %d, want 2008 alpha %d", i, tc.age, alpha, tc.alpha)
			}
		}
	}
}

func TestCloseConfineStatusOwnsBothRings(t *testing.T) {
	ctx := rogueSkillContext()
	m := &WorldMode{}
	for _, status := range []network.StatusEffectChange{
		{StatusID: db.StatusRgCconfineM, ActorID: 2000000, Active: true},
		{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true},
	} {
		m.applyStatusEffectChange(ctx, status)
		m.applyStatusEffectChange(ctx, status)
	}
	if len(m.worldEffects) != 2 || len(m.scheduledSounds) != 0 {
		t.Fatalf("expected one silent ring per participant: %+v, %+v", m.worldEffects, m.scheduledSounds)
	}
	for _, effect := range m.worldEffects {
		if effect.effectID != effectNPCStop2 || effect.expired(time.Now().Add(time.Hour)) {
			t.Fatalf("ring must stay until status removal: %+v", effect)
		}
	}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusRgCconfineM, ActorID: 100})
	if len(m.worldEffects) != 1 || m.worldEffects[0].actorID != 300 {
		t.Fatalf("local character/account aliases did not remove the same ring: %+v", m.worldEffects)
	}
	m.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 300, EffectID: uint32(effectNPCStop2Del)})
	if len(m.worldEffects) != 0 {
		t.Fatal("explicit ring deletion did not remove it")
	}
	status := network.StatusEffectChange{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true, HasDuration: true, Duration: time.Second}
	m.applyStatusEffectChange(ctx, status)
	m.worldEffects[0].expires = time.Now().Add(-time.Second)
	m.applyStatusEffectChange(ctx, status)
	active := 0
	for _, effect := range m.worldEffects {
		if !effect.expired(time.Now()) {
			active++
			if effect.expired(time.Now().Add(500*time.Millisecond)) || !effect.expired(time.Now().Add(2*time.Second)) {
				t.Fatal("ring lost the status deadline")
			}
		}
	}
	if active != 1 {
		t.Fatal("refresh revived an expired ring alongside its replacement")
	}
}

func TestCloseConfineBodyAndRingRendering(t *testing.T) {
	ctx := rogueSkillContext()
	starts := time.Unix(100, 0)
	m := &WorldMode{worldEffects: []worldEffect{{effectID: effectQuakeBody4, actorID: 300, starts: starts, expires: starts.Add(60 * db.EffectFrameDuration)}}}
	billboard := &spriteBillboard{image: render.NewImage(64, 64), anchorX: 32, anchorY: 60}
	projection := newSceneProjectionForTarget(800, 600, 16.5, 20.5, 0)
	entry := sceneActorDrawEntry{actor: ctx.World.Actors[300], worldX: 16.5, worldY: 20.5}
	for _, tick := range []int{0, 20, 21, 22, 59, 60, 61} {
		frame := render.NewFrame(800, 600)
		m.drawActorSpriteWithEffects3D(ctx, frame, projection, entry, billboard, 1, 1, 1, color.RGBA{255, 255, 255, 255}, render.BlendSourceOver, starts.Add(time.Duration(tick)*db.EffectFrameDuration))
		draw := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Index(0)
		red := draw.FieldByName("ColorR").Float() > draw.FieldByName("ColorG").Float()
		if red != (tick == 22) {
			t.Fatalf("wrong Close Confine tint at tick %d", tick)
		}
	}
	if billboard.anchorX != 32 || billboard.anchorY != 60 || ctx.World.Actors[300].X != 16 {
		t.Fatal("Close Confine modified cached geometry or actor position")
	}
	m = &WorldMode{effectViews: map[string]*spriteView{"cconfine": effectRenderTestSprite()}}
	m.applyStatusEffectChange(ctx, network.StatusEffectChange{StatusID: db.StatusRgCconfineS, ActorID: 300, Active: true})
	frame := render.NewFrame(800, 600)
	m.drawWorldEffects(frame, ctx, projection, time.Now().Add(time.Minute), false)
	draws := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
	if draws.Len() != 1 {
		t.Fatal("ring did not keep its final sprite frame")
	}
	draw := draws.Index(0)
	if math.Abs(draw.FieldByName("ColorA").Float()-100.0/255) > .001 || draw.FieldByName("Options").FieldByName("DepthTest").Bool() {
		t.Fatal("Close Confine ring must use native translucency and overlay rendering")
	}
}

func TestRogueParticleRendering(t *testing.T) {
	ctx := rogueSkillContext()
	projection := newSceneProjectionForTarget(800, 600, 14, 20, 0)
	starts := time.Unix(100, 0)
	for _, tc := range []struct{ id, tick, count int }{
		{effectRogueCoin, 56, 120}, {effectIntimidate, 108, 80}, {effectTeiHit3, 49, 80},
	} {
		spec, _ := worldEffectSpecForID(tc.id)
		m := &WorldMode{textures: map[string]*render.Image{
			"__effectfile_" + spec.components[0].textureFile + "|blackkey=true": render.NewImage(8, 8),
		}}
		m.addWorldEffectAt(ctx, tc.id, 300, starts)
		frame := render.NewFrame(800, 600)
		m.drawWorldEffects(frame, ctx, projection, starts.Add(time.Duration(tc.tick)*db.EffectFrameDuration), false)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 || commands.Index(0).FieldByName("Vertices").Len() != tc.count*4 {
			t.Fatalf("effect %d did not batch its native particles", tc.id)
		}
		options := commands.Index(0).FieldByName("Options")
		if !options.FieldByName("DepthTest").Bool() {
			t.Fatalf("effect %d must respect scene depth", tc.id)
		}
		wantBlend := render.BlendLighter
		if tc.id == effectTeiHit3 {
			wantBlend = render.BlendSourceOver
		}
		if options.FieldByName("Blend").Int() != int64(wantBlend) {
			t.Fatalf("effect %d uses the wrong native blending", tc.id)
		}
	}
}

func TestRogueRealAssets(t *testing.T) {
	root := os.Getenv("GORO_DATA_DIR")
	if root == "" {
		t.Skip("set GORO_DATA_DIR to check real effect assets")
	}
	manager, err := res.NewManager(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, archive := range manager.Archives {
		t.Cleanup(func() { _ = archive.Close() })
	}
	m := &WorldMode{}
	for _, id := range []int{effectStealCoin, effectRogueCoin, effectIntimidate, effectBackStab, effectTeiHit3, effectStripWeapon, effectStripShield, effectStripArmor, effectStripHelm, effectQuakeBody4, effectNPCStop2} {
		spec, _ := worldEffectSpecForID(id)
		for _, c := range spec.components {
			if c.textureFile != "" && m.effectFileTexture(manager, c.textureFile) == nil {
				t.Errorf("effect %d missing texture %s", id, c.textureFile)
			}
			if c.spriteFile != "" && m.effectSpriteView(manager, c.spriteFile) == nil {
				t.Errorf("effect %d missing sprite %s", id, c.spriteFile)
			}
			if c.strFile != "" && m.loadWorldEffectSTR(manager, c.strFile, c.texturePath) == nil {
				t.Errorf("effect %d missing STR %s", id, c.strFile)
			}
		}
		for _, sound := range spec.sfx {
			if _, err := manager.ReadFile("data/wav/" + sound); err != nil {
				t.Error(err)
			}
		}
	}
	if m.effectFileTexture(manager, "effect/alpabet.bmp") == nil {
		t.Fatal("missing Scribble bitmap alphabet")
	}
}
