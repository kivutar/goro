package game

import (
	"image/color"
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

func TestOriginalHitCylinderDimensionsAndMotion(t *testing.T) {
	starts := time.Unix(100, 0)
	for _, tt := range []struct {
		id, index                      int
		inner, outer, height, distance float64
	}{
		{effectHit3, 0, .3, .3, .14, .7 / 5 * (1 - 1.0/30)},
		{effectHit3, 1, .3, .8, .09, .7 / 5 * (1 - 1.0/30)},
		{effectHit4, 0, .1, .8, .08, .7 / 5 * (1 - 1.0/30)},
		{effectSonicBlowHit, 0, .8, 1.4, .7, .4 + .4/5*(1-1.0/24)},
	} {
		spec, _ := worldEffectSpecForID(tt.id)
		component := spec.components[tt.index]
		effect := worldEffect{effectID: tt.id, starts: starts, actorDirection: 0}
		mode := WorldMode{textures: map[string]*render.Image{"__effect_" + component.textureName: render.NewImage(64, 64)}}
		frame := render.NewFrame(800, 600)
		mode.drawFuncEffect(frame, client.Context{Resources: &res.Manager{}}, sceneProjection{}, effect, component, tt.index, 10, 20, 3, 0, starts)
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() != 1 {
			t.Fatalf("effect %d component %d: no cylinder", tt.id, tt.index)
		}
		vertices := commands.Index(0).FieldByName("Vertices")
		var centers [2]modelPoint3
		for i := 0; i < 20; i++ {
			v := vertices.Index(i)
			centers[i%2] = add3(centers[i%2], mul3(modelPoint3{x: v.FieldByName("X").Float(), y: v.FieldByName("Y").Float(), z: v.FieldByName("Z").Float()}, .1))
		}
		if !modelPointNear(sub3(centers[1], centers[0]), modelPoint3{z: tt.height}, 1e-5) || math.Abs(centers[0].z-20-tt.distance) > 1e-5 {
			t.Fatalf("effect %d component %d: cylinder centres = %+v", tt.id, tt.index, centers)
		}
		for side, wantRadius := range []float64{tt.inner, tt.outer} {
			v := vertices.Index(side)
			p := modelPoint3{x: v.FieldByName("X").Float(), y: v.FieldByName("Y").Float(), z: v.FieldByName("Z").Float()}
			d := sub3(p, centers[side])
			if math.Abs(math.Sqrt(dot3(d, d))-wantRadius) > 1e-5 {
				t.Fatalf("effect %d: wrong radius", tt.id)
			}
		}
		if tt.id == effectSonicBlowHit {
			if centers[0].x < 10-1e-5 || centers[0].x > 10.4+1e-5 || centers[0].y < 3.2-1e-5 || centers[0].y > 5.2+1e-5 {
				t.Fatalf("sonic impact offset = %+v", centers[0])
			}
		} else if math.Abs(centers[0].y-5) > 1e-5 {
			t.Fatalf("hit altitude = %g", centers[0].y)
		}
		_, alpha := hitRingAnimation(component, 9*db.EffectFrameDuration)
		if tt.id != effectSonicBlowHit && math.Abs(alpha-254.0/255) > 1e-9 {
			t.Fatal("legacy hit fades before tick ten")
		}
	}
}

func TestHitParticleTrailUsesEarlierPositionsAndCurrentFade(t *testing.T) {
	p := hitParticle{axis: modelPoint3{z: 1}, radius: 1, speed: .2, size: 1.2, frames: 10}
	// Three updates: distances .19, .18, .17; history trails by one/two ticks.
	for trail, distance := range []float64{1.54, 1.37, 1.19} {
		position, size, alpha := p.segment(2, trail, 2)
		if !modelPointNear(position, modelPoint3{y: 2, z: distance}, 1e-9) {
			t.Fatalf("trail %d position = %+v", trail, position)
		}
		if math.Abs(size-.84*(1-float64(trail)/6)) > 1e-9 || math.Abs(alpha-(254.0/255)*.7*(1-float64(trail)/3)) > 1e-9 {
			t.Fatalf("trail %d size/alpha = %g/%g", trail, size, alpha)
		}
	}
	if pos, _, _ := p.segment(0, 2, 2); pos != (modelPoint3{}) {
		t.Fatal("initial trail history must start at actor anchor")
	}
	if _, size, alpha := p.segment(9, 0, 2); size != 0 || alpha != 0 {
		t.Fatal("particle did not shrink/fade completely")
	}
	for _, id := range []int{effectHit3, effectHit4} {
		spec, _ := worldEffectSpecForID(id)
		component := spec.components[len(spec.components)-1]
		want := 8
		if id == effectHit4 {
			want = 5
		}
		if component.duplicate != want {
			t.Fatalf("effect %d: want %d trails", id, want)
		}
		for i := 0; i < 32; i++ {
			p := newHitParticle(worldEffect{effectID: id, starts: time.Unix(100, 0)}, component, i)
			if p.frames < 6 || p.frames > 29 || p.size < .6 || p.size >= 1.6 || math.Abs(dot3(p.axis, p.axis)-1) > 1e-9 {
				t.Fatalf("invalid randomized particle: %+v", p)
			}
		}
	}
}

func TestOriginalBodyColourWindowsAndRestoration(t *testing.T) {
	starts := time.Unix(100, 0)
	base := color.RGBA{R: 255, G: 255, B: 255, A: 173}
	for _, id := range []int{effectMagicCrasher, effectTransBlueBody} {
		spec, _ := worldEffectSpecForID(id)
		mode := WorldMode{worldEffects: []worldEffect{{effectID: id, actorID: 42, starts: starts, expires: starts.Add(spec.duration)}}}
		for _, frame := range []int{-1, 0, 29, 30, 60, 61, 199, 200} {
			now := starts.Add(time.Duration(frame) * db.EffectFrameDuration)
			got := mode.actorBodyColorTint(42, base, now)
			if got.A != base.A {
				t.Fatal("body effect changed transparency")
			}
			wantBlend := render.BlendSourceOver
			if id == effectMagicCrasher && frame >= 30 && frame <= 60 {
				wantBlend = render.BlendLighter
			}
			if got := mode.actorRenderBlend(42, now); got != wantBlend {
				t.Fatalf("effect %d frame %d: blend %v, want %v", id, frame, got, wantBlend)
			}
			if id == effectMagicCrasher {
				active := frame >= 30 && frame <= 60
				if (got != base) != active {
					t.Fatalf("Magic Crasher at frame %d: tint %+v", frame, got)
				}
				if again := mode.actorBodyColorTint(42, base, now); again != got {
					t.Fatal("flash changes within a tick")
				}
			} else {
				want := base
				if frame >= 0 && frame < 200 {
					want.R, want.G = uint8(205-frame), uint8(205-frame)
				}
				if got != want {
					t.Fatalf("blue transition at frame %d: %+v, want %+v", frame, got, want)
				}
			}
			if mode.actorBodyColorTint(99, base, now) != base {
				t.Fatal("effect tinted another actor")
			}
		}
		// A second cast replaces the first colour instead of multiplying two tints.
		second := mode.worldEffects[0]
		second.starts = starts.Add(10 * db.EffectFrameDuration)
		second.expires = second.starts.Add(spec.duration)
		mode.worldEffects = append(mode.worldEffects, second)
		onlySecond := WorldMode{worldEffects: []worldEffect{second}}
		now := starts.Add(40 * db.EffectFrameDuration)
		if mode.actorBodyColorTint(42, base, now) != onlySecond.actorBodyColorTint(42, base, now) {
			t.Fatal("recasting compounds body colour")
		}
	}
}

func TestMagicCrasherBlendReachesActorDraws(t *testing.T) {
	sprite := &spriteView{
		spr:    &res.SPR{Frames: []res.SPRFrame{{Type: res.SPRFrameRGBA, Width: 8, Height: 8, Data: solidRGBAFrame(8, 8)}}},
		act:    &res.ACT{Actions: []res.ACTAction{{DelayMS: 100, Animations: []res.ACTAnimation{{Layers: []res.ACTLayer{{Index: 0, SPRType: res.SPRFrameRGBA, ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1}}}}}}}},
		images: make(map[spriteFrameKey]*render.Image), billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	mode := &WorldMode{
		playerView: &humanoidSpriteView{body: sprite, billboards: make(map[humanoidBillboardKey]*spriteBillboard)},
		nonPCViews: map[int]*spriteView{1002: sprite},
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 42, X: 10, Y: 20}
	ctx := client.Context{World: world, Session: &session.Session{AccountID: 43, CharID: 44}}
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	for _, target := range []uint32{42, 43, 44, 99} {
		for _, expired := range []bool{false, true} {
			now := time.Now()
			effect := worldEffect{effectID: effectMagicCrasher, actorID: target, starts: now.Add(-700 * time.Millisecond), expires: now.Add(time.Second)}
			if expired {
				effect.expires = now.Add(-time.Millisecond)
			}
			mode.worldEffects = []worldEffect{effect}
			for _, local := range []bool{false, true} {
				for _, stealth := range []stealthView{stealthVisible, stealthSilhouette} {
					entry := sceneActorDrawEntry{actor: worldstate.Actor{ID: 99, Job: 1002}, worldX: 10.5, worldY: 20.5, scale: 1, stealth: stealth}
					frame := render.NewFrame(800, 600)
					var drawn bool
					if local {
						entry.actor = world.Player
						drawn = mode.drawPlayerSprite3D(ctx, frame, projection, entry, 0, 0, 1, 1)
					} else {
						drawn = mode.drawNonPCSprite3D(frame, ctx, projection, entry, 0, 1)
					}
					commands := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards")
					if !drawn || commands.Len() != 1 {
						t.Fatalf("local=%t: missing actor billboard", local)
					}
					want := render.BlendSourceOver
					if !expired && stealth == stealthVisible && (local == (target != 99)) {
						want = render.BlendLighter
					}
					options := commands.Index(0).FieldByName("Options")
					if got := render.Blend(options.FieldByName("Blend").Int()); got != want {
						t.Fatalf("target=%d local=%t expired=%t stealth=%d: blend %v, want %v", target, local, expired, stealth, got, want)
					}
					if !options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() {
						t.Fatal("flash changed actor depth options")
					}
				}
			}
		}
	}
}

func TestBlessingCircleIsFlatAndUsesOriginalFade(t *testing.T) {
	spec, _ := worldEffectSpecForID(effectBlessing)
	component := spec.components[3]
	starts := time.Unix(100, 0)
	mode := WorldMode{textures: map[string]*render.Image{"__effect_alpha_down": render.NewImage(64, 64)}}
	for _, tt := range []struct {
		frame int
		alpha float64
	}{{-1, 0}, {0, 100.0 / 255 / 30}, {29, 100.0 / 255}, {119, 100.0 / 255}, {120, 100.0 / 255 * 29 / 30}, {149, 0}, {150, 0}} {
		if got := blessingCircleAlpha(component, time.Duration(tt.frame)*db.EffectFrameDuration); math.Abs(got-tt.alpha) > 1e-9 {
			t.Fatalf("frame %d alpha = %g want %g", tt.frame, got, tt.alpha)
		}
	}
	frame := render.NewFrame(800, 600)
	mode.drawBlessingCircleEffect(frame, client.Context{Resources: &res.Manager{}}, component, worldEffect{starts: starts}, 10, 20, 3, starts.Add(40*db.EffectFrameDuration))
	commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatal("missing blessing circle")
	}
	cmd := commands.Index(0)
	vertices := cmd.FieldByName("Vertices")
	for i := 0; i < vertices.Len(); i++ {
		v := vertices.Index(i)
		x, y, z := v.FieldByName("X").Float(), v.FieldByName("Y").Float(), v.FieldByName("Z").Float()
		radius := math.Hypot(x-10, z-20)
		wantRadius := 2.0
		if i%3 == 0 {
			wantRadius = 0
		}
		if y != 3 || math.Abs(radius-wantRadius) > 1e-5 {
			t.Fatalf("vertex %d is not on the ground circle: %g,%g,%g", i, x, y, z)
		}
	}
	options := cmd.FieldByName("Options")
	if options.FieldByName("DepthTest").Bool() || options.FieldByName("DepthWrite").Bool() || render.Blend(options.FieldByName("Blend").Int()) != render.BlendSourceOver {
		t.Fatal("blessing must use RF_ALPHA|RF_NODEPTHCHECK")
	}
}

func TestOriginalEffectsRenderWithOldROAssets(t *testing.T) {
	manager := realDataManager(t)
	world := worldstate.New()
	ctx := client.Context{World: world, Resources: manager}
	starts := time.Unix(100, 0)
	projection := newSceneProjectionForTarget(800, 600, 10.5, 20.5, 0)
	for _, id := range []int{effectSonicBlowHit, effectHit3, effectHit4, effectBlessing} {
		world.Actors[42] = worldstate.Actor{ID: 42, X: 10, Y: 20}
		mode := WorldMode{}
		if !mode.addWorldEffectAt(ctx, id, 42, starts) {
			t.Fatalf("effect %d failed to spawn", id)
		}
		frame := render.NewFrame(800, 600)
		mode.drawWorldEffects(frame, ctx, projection, starts.Add(2*db.EffectFrameDuration))
		if len(mode.textureMiss) > 0 || len(mode.effectViewMiss) > 0 {
			t.Fatalf("effect %d missing resources: %v %v", id, mode.textureMiss, mode.effectViewMiss)
		}
		commands := reflect.ValueOf(frame).Elem().FieldByName("worldCommands")
		if commands.Len() == 0 {
			t.Fatalf("effect %d drew nothing", id)
		}
		if id == effectHit3 || id == effectHit4 {
			count := reflect.ValueOf(frame).Elem().FieldByName("worldBillboards").Len()
			want := 24
			if id == effectHit4 {
				want = 15
			}
			if count != want {
				t.Fatalf("effect %d: got %d particle sprites, want %d", id, count, want)
			}
		}
		if id != effectBlessing {
			world.Actors[42] = worldstate.Actor{ID: 42, X: 30, Y: 40, Dir: 4}
			moved := render.NewFrame(800, 600)
			mode.drawWorldEffects(moved, ctx, projection, starts.Add(2*db.EffectFrameDuration))
			if !reflect.DeepEqual(frame, moved) {
				t.Fatalf("effect %d moved with the target instead of staying at impact", id)
			}
		}
	}
}
