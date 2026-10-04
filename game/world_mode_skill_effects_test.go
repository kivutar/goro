package game

import (
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/input"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	"github.com/kivutar/goro/session"
	gameui "github.com/kivutar/goro/ui"
	worldstate "github.com/kivutar/goro/world"
)

func TestEffectBodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectBodyColor,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(300 * time.Millisecond),
		}},
	}
	if got := mode.actorBodyColorTint(42, base, starts); got != base {
		t.Fatalf("initial tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(50*time.Millisecond)); got != (color.RGBA{R: 177, G: 75, B: 100, A: 255}) {
		t.Fatalf("half flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("full flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(300*time.Millisecond)); got != (color.RGBA{R: 255, G: 0, B: 0, A: 255}) {
		t.Fatalf("final flash tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(301*time.Millisecond)); got != base {
		t.Fatalf("expired tint = %+v, want base %+v", got, base)
	}
	if got := mode.actorBodyColorTint(99, base, starts.Add(100*time.Millisecond)); got != base {
		t.Fatalf("other actor tint = %+v, want base %+v", got, base)
	}
}

func TestPortal5BodyColorTintFollowsRobrowserFlashWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectPortal5,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(800 * time.Millisecond),
		}},
	}
	if got := mode.actorBodyColorTint(42, base, starts); got != (color.RGBA{R: 100, G: 150, B: 0, A: 25}) {
		t.Fatalf("initial tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(500*time.Millisecond)); got != (color.RGBA{R: 100, G: 150, B: 100, A: 140}) {
		t.Fatalf("mid tint = %+v", got)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(800*time.Millisecond)); got != base {
		t.Fatalf("final tint = %+v, want base %+v", got, base)
	}
}

func TestMagicCrasher2BodyColorTintUsesRobrowserRandomColorWindow(t *testing.T) {
	base := color.RGBA{R: 100, G: 150, B: 200, A: 255}
	starts := time.Unix(10, 0)
	mode := WorldMode{
		worldEffects: []worldEffect{{
			effectID: effectMagicCrasher2,
			actorID:  42,
			starts:   starts,
			expires:  starts.Add(time.Second),
		}},
	}
	first := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	second := mode.actorBodyColorTint(42, base, starts.Add(100*time.Millisecond))
	if first != second {
		t.Fatalf("random tint should be deterministic per frame: first %+v second %+v", first, second)
	}
	if first == base || first.A != base.A {
		t.Fatalf("active random tint = %+v, want color-channel change with unchanged alpha", first)
	}
	if got := mode.actorBodyColorTint(42, base, starts.Add(time.Second)); got != base {
		t.Fatalf("expired random tint = %+v, want base %+v", got, base)
	}
}

func TestActorBodySizeMultiplierFollowsRobrowserEswooSizes(t *testing.T) {
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name     string
		effectID int
		duration time.Duration
		at       time.Duration
		want     float64
	}{
		{"EF_BABYBODY start", effectBabyBody, 300 * time.Millisecond, 0, 1},
		{"EF_BABYBODY middle", effectBabyBody, 300 * time.Millisecond, 150 * time.Millisecond, 0.75},
		{"EF_BABYBODY end", effectBabyBody, 300 * time.Millisecond, 300 * time.Millisecond, 0.5},
		{"EF_BABYBODY2", effectBabyBody2, 5 * time.Minute, time.Second, 0.5},
		{"EF_GIANTBODY middle", effectGiantBody, 300 * time.Millisecond, 150 * time.Millisecond, 1.25},
		{"EF_GIANTBODY end", effectGiantBody, 300 * time.Millisecond, 300 * time.Millisecond, 1.5},
		{"EF_GIANTBODY2", effectGiantBody2, 5 * time.Minute, time.Second, 1.5},
	} {
		mode := WorldMode{
			worldEffects: []worldEffect{{
				effectID: tc.effectID,
				actorID:  42,
				starts:   starts,
				expires:  starts.Add(tc.duration),
				duration: tc.duration,
			}},
		}
		if got := mode.actorBodySizeMultiplier(42, starts.Add(tc.at)); got != tc.want {
			t.Fatalf("%s multiplier = %f, want %f", tc.name, got, tc.want)
		}
	}
}

func TestRobrowserRepairWeaponAndShockwaveSpecs(t *testing.T) {
	repair, ok := worldEffectSpecForID(effectRepairWeapon)
	if !ok || len(repair.components) != 1 || repair.duration != 1820*time.Millisecond {
		t.Fatalf("EF_REPAIRWEAPON spec = %+v ok=%t", repair, ok)
	}
	if len(repair.sfx) != 2 || repair.sfx[0] != "effect\\black_weapon_repair_a.wav" || repair.sfx[1] != "effect\\black_weapon_repair_a.wav" {
		t.Fatalf("EF_REPAIRWEAPON sfx = %v", repair.sfx)
	}
	if len(repair.sfxDelays) != 2 || repair.sfxDelays[0] != 480*time.Millisecond || repair.sfxDelays[1] != 1320*time.Millisecond {
		t.Fatalf("EF_REPAIRWEAPON sfx delays = %v", repair.sfxDelays)
	}
	if component := repair.components[0]; component.kind != effectComponentSTR || component.strFile != "repairweapon" || !component.attachedEntity {
		t.Fatalf("EF_REPAIRWEAPON component = %+v", component)
	}

	shockwave, ok := worldEffectSpecForID(effectShockwave)
	if !ok || len(shockwave.components) != 1 {
		t.Fatalf("EF_SHOCKWAVE spec = %+v ok=%t", shockwave, ok)
	}
	if len(shockwave.sfx) != 1 || shockwave.sfx[0] != "effect\\hunter_shockwavetrap.wav" {
		t.Fatalf("EF_SHOCKWAVE sfx = %v", shockwave.sfx)
	}
	if component := shockwave.components[0]; component.kind != effectComponentSPR || component.spriteFile != "shockwave" || !component.attachedEntity {
		t.Fatalf("EF_SHOCKWAVE component = %+v", component)
	}
}

func TestRobrowserWaterBallAndSonicBlowSpecs(t *testing.T) {
	water, ok := worldEffectSpecForID(effectWaterBall)
	if !ok || len(water.components) != 1 || water.duration != 500*time.Millisecond {
		t.Fatalf("EF_WATERBALL spec = %+v ok=%t", water, ok)
	}
	component := water.components[0]
	if component.kind != effectComponent3D || len(component.textureFiles) != 3 || component.textureFiles[0] != "effect/water_out_a.bmp" || component.textureFiles[2] != "effect/water_out_c.bmp" {
		t.Fatalf("EF_WATERBALL texture files = %+v", component)
	}
	if component.frameDelay != 10*time.Millisecond || component.duration != 500*time.Millisecond || !component.fadeOut || component.posXRand != 1.5 || component.posZRand != 1.5 || component.posYEnd != 3 || !component.posYSmooth || component.sizeStart != effectTableSize(30.5) || !component.rotateWithCamera || !component.blendAdditive || !component.attachedEntity {
		t.Fatalf("EF_WATERBALL component = %+v", component)
	}

	water2, ok := worldEffectSpecForID(effectWaterBall2)
	if !ok || len(water2.components) != 1 || water2.duration != 1450*time.Millisecond {
		t.Fatalf("EF_WATERBALL2 spec = %+v ok=%t", water2, ok)
	}
	projectile := water2.components[0]
	if projectile.kind != effectComponent3D || projectile.spriteFile != "data\\sprite\\이팩트\\waterball" || projectile.duration != 500*time.Millisecond || projectile.duplicate != 20 || projectile.duplicateDelay != 50*time.Millisecond {
		t.Fatalf("EF_WATERBALL2 projectile resource/timing = %+v", projectile)
	}
	if !projectile.fromSrc || !projectile.rotateToTarget || !projectile.fadeOut || projectile.sizeStart != effectTableSize(50) || projectile.posZ != 5 || projectile.posZEnd != 0.0001 || projectile.arc != 7.5 || projectile.retreat != 5 {
		t.Fatalf("EF_WATERBALL2 projectile motion = %+v", projectile)
	}

	sonic, ok := worldEffectSpecForID(effectSonicBlow)
	if !ok || len(sonic.components) != 1 || sonic.duration != 400*time.Millisecond {
		t.Fatalf("EF_SONICBLOW spec = %+v ok=%t", sonic, ok)
	}
	ring := sonic.components[0]
	if ring.kind != effectComponent3D || ring.textureFile != "effect/ring2.bmp" || ring.duration != 400*time.Millisecond || ring.alphaMax != 1 || !ring.fadeOut || ring.sizeStart != effectTableSize(100) || ring.sizeEnd != effectTableSize(300) || !ring.blendAdditive || !ring.attachedEntity {
		t.Fatalf("EF_SONICBLOW ring = %+v", ring)
	}
	spin, ok := worldEffectSpecForID(effectSonicBlowHit)
	if !ok || len(spin.components) != 1 || spin.components[0].kind != effectComponentFUNC || spin.components[0].funcAdapter != effectFuncHitRing || !spin.components[0].attachedEntity {
		t.Fatalf("EF_SONICBLOWHIT spec = %+v ok=%t", spin, ok)
	}
}

func TestRobrowserCrashEarthFirePillarAndQuadHornOneHundredSpecs(t *testing.T) {
	crash, ok := worldEffectSpecForID(effectCrashEarth)
	if !ok || len(crash.components) != 1 {
		t.Fatalf("EF_CRASHEARTH spec = %+v ok=%t", crash, ok)
	}
	if crash.cameraShakeDelay != 350*time.Millisecond || crash.cameraShake != 650*time.Millisecond {
		t.Fatalf("EF_CRASHEARTH camera shake = delay %s duration %s", crash.cameraShakeDelay, crash.cameraShake)
	}
	if component := crash.components[0]; component.kind != effectComponentSTR || component.strFile != "crashearth" || component.attachedEntity {
		t.Fatalf("EF_CRASHEARTH component = %+v", component)
	}

	fire, ok := worldEffectSpecForID(effectFirePillarOn)
	if !ok || len(fire.components) != 3 || fire.duration != 6*time.Second {
		t.Fatalf("EF_FIREPILLARON spec = %+v ok=%t", fire, ok)
	}
	for i, component := range fire.components {
		if component.kind != effectComponentCylinder || component.textureName != "magic_red" || component.duration != 5*time.Second || component.delay != time.Second || !component.rotate || component.attachedEntity {
			t.Fatalf("EF_FIREPILLARON component %d = %+v", i, component)
		}
	}
	if fire.components[0].bottomSize != 1 || fire.components[0].topSize != 2 || fire.components[0].height != 3 || fire.components[2].bottomSize != 0.5 || fire.components[2].topSize != 1 || fire.components[2].height != 7 {
		t.Fatalf("EF_FIREPILLARON cylinder sizes = %+v", fire.components)
	}

	grim, ok := worldEffectSpecForID(effectGrimtoothAtk)
	if !ok || len(grim.components) != 3 || grim.duration != 15*time.Second {
		t.Fatalf("EF_GRIMTOOTHATK spec = %+v ok=%t", grim, ok)
	}
	first := grim.components[0]
	if first.kind != effectComponentQuadHorn || first.textureFile != "effect/stone.bmp" || first.duration != 15*time.Second || first.quadHornHeightMin != 2.5 || first.quadHornBottomMin != 0.15 || first.quadHornRotateXMin != -15 || first.quadHornOffsetYMin != 0.4 || first.quadHornOffsetZ != -0.2 || first.animation != 3 || first.quadHornAnimSpeed != 120*time.Millisecond || !first.quadHornAnimOut {
		t.Fatalf("EF_GRIMTOOTHATK first = %+v", first)
	}
	if grim.components[1].quadHornRotateYMin != 45 || grim.components[1].quadHornRotateZMin != -15 || grim.components[2].quadHornRotateZMin != 15 {
		t.Fatalf("EF_GRIMTOOTHATK rotations = %+v %+v", grim.components[1], grim.components[2])
	}

	heaven, ok := worldEffectSpecForID(effectHeavenDrive)
	if !ok || len(heaven.components) != 25 || heaven.duration != time.Second || heaven.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_HEAVENDRIVE spec = %+v ok=%t", heaven, ok)
	}
	if len(heaven.sfx) != 1 || heaven.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_HEAVENDRIVE sfx = %v", heaven.sfx)
	}
	center := heaven.components[12]
	if center.kind != effectComponentQuadHorn || center.textureFile != "effect/stone.bmp" || center.duration != time.Second || center.posX != 0 || center.posY != 0 || center.quadHornHeightMin != 0.75 || center.quadHornHeightMax != 1.2 || center.quadHornBottomMin != 0.4 || center.quadHornBottomMax != 0.7 || center.quadHornAnimSpeed != 250*time.Millisecond || !center.quadHornAnimOut {
		t.Fatalf("EF_HEAVENDRIVE center = %+v", center)
	}
	if heaven.components[0].posX != -2 || heaven.components[0].posY != -2 || heaven.components[24].posX != 2 || heaven.components[24].posY != 2 {
		t.Fatalf("EF_HEAVENDRIVE grid edges = %+v %+v", heaven.components[0], heaven.components[24])
	}
}

func TestRobrowserQuadHornRuntimeDefaults(t *testing.T) {
	if got := quadHornDefaultOffset(0); got != 0.5 {
		t.Fatalf("quadHornDefaultOffset(0) = %v, want robr default 0.5", got)
	}
	if got := quadHornDefaultOffset(-0.2); got != -0.2 {
		t.Fatalf("quadHornDefaultOffset(-0.2) = %v, want -0.2", got)
	}
	effect := worldEffect{effectID: effectEarthSpike, actorID: 1, starts: time.Unix(10, 20)}
	if got := quadHornRange(effect, 1, 0, -0.2); got != -0.2 {
		t.Fatalf("quadHornRange max<min = %v, want -0.2", got)
	}
}

func TestQuadHornUVsMatchSourceFaceStrips(t *testing.T) {
	if len(quadHornUVs) != 12 {
		t.Fatalf("quadHornUVs len = %d, want 12", len(quadHornUVs))
	}
	for face := 0; face < 4; face++ {
		u0 := float32(face) * 0.2
		uvs := quadHornUVs[face*3 : face*3+3]
		want := []texturePoint{
			{u: u0, v: 0},
			{u: u0, v: 1},
			{u: u0 + 0.2, v: 1},
		}
		if !reflect.DeepEqual(uvs, want) {
			t.Fatalf("quad horn face %d uvs = %+v, want %+v", face, uvs, want)
		}
	}
}

func TestQuadHornDefaultRotationKeepsApexAboveBase(t *testing.T) {
	effect := worldEffect{effectID: effectIceWall, actorID: 1, starts: time.Unix(10, 20)}
	component := worldEffectComponent{}
	salt := 0
	rotateX := quadHornRange(effect, salt+5, component.quadHornRotateXMin, component.quadHornRotateXMax)
	rotateY := quadHornRange(effect, salt+6, component.quadHornRotateYMin, component.quadHornRotateYMax)
	rotateZ := quadHornRange(effect, salt+7, component.quadHornRotateZMin, component.quadHornRotateZMax)
	height := 3.0
	bottomSize := 0.4
	origin := modelPoint3{y: height * 0.9}
	apex := add3(origin, rotateEffectCylinderVector(modelPoint3{y: height}, rotateX, rotateY, rotateZ))
	base := add3(origin, rotateEffectCylinderVector(modelPoint3{x: -bottomSize, y: -height, z: bottomSize}, rotateX, rotateY, rotateZ))

	if apex.y <= base.y {
		t.Fatalf("quad horn apex y = %.3f, base y = %.3f, want apex above base", apex.y, base.y)
	}
}

func TestRobrowserOldPortalEffectsZeroToFifty(t *testing.T) {
	entry, ok := worldEffectSpecForID(effectEntry)
	if !ok {
		t.Fatal("EF_ENTRY spec missing")
	}
	if entry.duration != 500*time.Millisecond || len(entry.sfx) != 1 || entry.sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY timing/sfx = duration %s sfx %#v", entry.duration, entry.sfx)
	}
	if len(entry.components) != 3 {
		t.Fatalf("EF_ENTRY components = %d, want 3", len(entry.components))
	}
	for i, component := range entry.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 500*time.Millisecond || component.animation != 1 || !component.fade || !component.rotate {
			t.Fatalf("EF_ENTRY component %d = %+v", i, component)
		}
	}
	if entry.components[0].attachedEntity || !entry.components[1].attachedEntity || !entry.components[2].attachedEntity {
		t.Fatalf("EF_ENTRY attachment flags = %t %t %t", entry.components[0].attachedEntity, entry.components[1].attachedEntity, entry.components[2].attachedEntity)
	}
	if entry.components[0].height != 7.5 || entry.components[1].height != 8 || entry.components[2].topSize != 1.5 {
		t.Fatalf("EF_ENTRY dimensions = %+v", entry.components)
	}

	warp, ok := worldEffectSpecForID(effectWarp)
	if !ok || len(warp.components) != 1 {
		t.Fatalf("EF_WARP spec = %+v ok=%t", warp, ok)
	}
	wave := warp.components[0]
	if wave.kind != effectComponentCylinder || wave.textureName != "ring_yellow" || wave.animation != 4 || wave.duplicate != 4 || wave.duplicateDelay != 300*time.Millisecond {
		t.Fatalf("EF_WARP wave = %+v", wave)
	}
	if wave.bottomSize != 10 || wave.topSize != 13 || wave.posZ != 0.1 || !wave.attachedEntity {
		t.Fatalf("EF_WARP dimensions = %+v", wave)
	}
	if got := worldEffectComponentDuration(warp, wave); got != 1900*time.Millisecond {
		t.Fatalf("EF_WARP resolved duration = %s, want 1900ms", got)
	}

	teleport, ok := worldEffectSpecForID(effectTeleportOld)
	if !ok || len(teleport.components) != 1 {
		t.Fatalf("EF_TELEPORTATION spec = %+v ok=%t", teleport, ok)
	}
	beam := teleport.components[0]
	if teleport.duration != time.Second || len(teleport.sfx) != 1 || teleport.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_TELEPORTATION timing/sfx = %s %#v", teleport.duration, teleport.sfx)
	}
	if beam.kind != effectComponentCylinder || beam.textureName != "ring_blue" || beam.animation != 5 || beam.height != 35 || beam.bottomSize != 0.8 || beam.topSize != 0.7 || !beam.rotate {
		t.Fatalf("EF_TELEPORTATION beam = %+v", beam)
	}

	ready, ok := worldEffectSpecForID(effectReadyPortalOld)
	if !ok || len(ready.components) != 1 {
		t.Fatalf("EF_READYPORTAL spec = %+v ok=%t", ready, ok)
	}
	portal := ready.components[0]
	if ready.duration != 25*time.Second || len(ready.sfx) != 1 || ready.sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_READYPORTAL timing/sfx = %s %#v", ready.duration, ready.sfx)
	}
	if portal.kind != effectComponentCylinder || portal.textureName != "alpha_down" || portal.color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || portal.height != 15 || portal.alphaMax != 0.6 {
		t.Fatalf("EF_READYPORTAL cylinder = %+v", portal)
	}
}

func TestRobrowserOldRestoreEffectsZeroToFifty(t *testing.T) {
	exit, ok := worldEffectSpecForID(effectExit)
	if !ok {
		t.Fatal("EF_EXIT spec missing")
	}
	if exit.duration != 2*time.Second || len(exit.sfx) != 1 || exit.sfx[0] != "_heal_effect.wav" || len(exit.components) != 3 {
		t.Fatalf("EF_EXIT spec = %+v", exit)
	}
	if cylinder := exit.components[0]; cylinder.kind != effectComponentCylinder || cylinder.textureName != "alpha_down" || cylinder.duration != 2*time.Second || cylinder.animation != 1 || cylinder.alphaMax != 0.2 || !cylinder.blendAdditive {
		t.Fatalf("EF_EXIT cylinder = %+v", cylinder)
	}
	if particle := exit.components[1]; particle.kind != effectComponent3D || particle.textureFile != "effect/pok3.tga" || particle.delay != 400*time.Millisecond || particle.duplicate != 6 || particle.duplicateDelay != 80*time.Millisecond || !particle.sparkling {
		t.Fatalf("EF_EXIT first particle = %+v", particle)
	}
	if particle := exit.components[2]; particle.duration != 900*time.Millisecond || particle.delay != 200*time.Millisecond || particle.duplicate != 3 || particle.duplicateDelay != 200*time.Millisecond || particle.posZEnd != 6 {
		t.Fatalf("EF_EXIT second particle = %+v", particle)
	}

	enhance, ok := worldEffectSpecForID(effectEnhance)
	if !ok || len(enhance.components) != 3 {
		t.Fatalf("EF_ENHANCE spec = %+v ok=%t", enhance, ok)
	}
	if enhance.components[0].textureName != "alpha_down" || enhance.components[0].blendAdditive != true || enhance.components[0].duration != 2*time.Second {
		t.Fatalf("EF_ENHANCE cylinder = %+v", enhance.components[0])
	}
	for _, tc := range []struct {
		index     int
		delay     time.Duration
		duplicate int
	}{
		{index: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 2, delay: 400 * time.Millisecond, duplicate: 3},
	} {
		component := enhance.components[tc.index]
		if component.kind != effectComponent3D || component.textureFile != "effect/ac_center2.tga" || component.delay != tc.delay || component.duplicate != tc.duplicate || component.duplicateDelay != 200*time.Millisecond {
			t.Fatalf("EF_ENHANCE particle %d = %+v", tc.index, component)
		}
		if component.sizeStartX != 2.5*effectPixelRatio || component.sizeRandY != 15*effectPixelRatio || component.sizeRandYMiddle != 45*effectPixelRatio {
			t.Fatalf("EF_ENHANCE particle %d size = %+v", tc.index, component)
		}
	}

	healSP, ok := worldEffectSpecForID(effectHealSP)
	if !ok || len(healSP.components) != 3 {
		t.Fatalf("EF_HEALSP spec = %+v ok=%t", healSP, ok)
	}
	if len(healSP.sfx) != 1 || healSP.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("EF_HEALSP sfx = %#v", healSP.sfx)
	}
	blue := color.RGBA{R: 25, G: 128, B: 255, A: 255}
	if cylinder := healSP.components[0]; cylinder.textureName != "ring_blue" || cylinder.color != blue || !cylinder.rotate || !cylinder.blendAdditive {
		t.Fatalf("EF_HEALSP cylinder = %+v", cylinder)
	}
	if healSP.components[1].color != blue || healSP.components[2].color != blue {
		t.Fatalf("EF_HEALSP particle tints = %+v %+v", healSP.components[1].color, healSP.components[2].color)
	}
}

func TestRobrowserOldBoltSoundAndStatusEffectsZeroToFifty(t *testing.T) {
	glass, ok := worldEffectSpecForID(effectGlassWall)
	if !ok || len(glass.components) != 1 {
		t.Fatalf("EF_GLASSWALL spec = %+v ok=%t", glass, ok)
	}
	if component := glass.components[0]; component.kind != effectComponentSTR || component.strFile != "effect/safetywall" || component.attachedEntity {
		t.Fatalf("EF_GLASSWALL component = %+v", component)
	}
	if len(glass.sfx) != 1 || glass.sfx[0] != "effect\\ef_glasswall.wav" {
		t.Fatalf("EF_GLASSWALL sfx = %#v", glass.sfx)
	}

	ice, ok := worldEffectSpecForID(effectIceArrow)
	if !ok {
		t.Fatal("EF_ICEARROW spec missing")
	}
	if len(ice.components) != 0 || len(ice.sfx) != 1 || ice.sfx[0] != "effect\\ef_icearrow%d.wav" || ice.sfxRandMin != 1 || ice.sfxRandMax != 3 {
		t.Fatalf("EF_ICEARROW spec = %+v", ice)
	}

	fire, ok := worldEffectSpecForID(effectFireArrow)
	if !ok {
		t.Fatal("EF_FIREARROW spec missing")
	}
	if len(fire.components) != 0 || len(fire.sfx) != 1 || fire.sfx[0] != "effect\\ef_firearrow1.wav" {
		t.Fatalf("EF_FIREARROW spec = %+v", fire)
	}

	incAgiDex, ok := worldEffectSpecForID(effectIncAgiDex)
	if !ok || len(incAgiDex.components) != 1 {
		t.Fatalf("EF_INCAGIDEX spec = %+v ok=%t", incAgiDex, ok)
	}
	if len(incAgiDex.sfx) != 1 || incAgiDex.sfx[0] != "effect\\ef_incagidex.wav" {
		t.Fatalf("EF_INCAGIDEX sfx = %#v", incAgiDex.sfx)
	}
	overlay := incAgiDex.components[0]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/dex_agi_up.bmp" || overlay.duration != time.Second || !overlay.fadeIn || !overlay.fadeOut || !overlay.attachedEntity || !overlay.overlay {
		t.Fatalf("EF_INCAGIDEX overlay = %+v", overlay)
	}
	if overlay.posZ != 0.4 || overlay.posZEnd != 3 || overlay.sizeStart != 100*effectPixelRatio || overlay.sizeStartY != 45*effectPixelRatio || !overlay.sizeSmooth {
		t.Fatalf("EF_INCAGIDEX overlay geometry = %+v", overlay)
	}
}

func TestPneumaEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectPneuma)
	if !ok {
		t.Fatal("Pneuma effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("Pneuma components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "pneuma%d" || component.strRandMin != 1 || component.strRandMax != 3 || component.attachedEntity {
		t.Fatalf("Pneuma component = %+v", component)
	}
}

func TestTorchEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectTorch)
	if !ok {
		t.Fatal("torch effect spec missing")
	}
	if spec.duration != 24*time.Hour {
		t.Fatalf("torch duration = %s", spec.duration)
	}
	if len(spec.components) != 1 {
		t.Fatalf("torch components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "torch_01" || !component.spriteRepeat {
		t.Fatalf("torch component = %+v", component)
	}
	if component.duration != 600*time.Millisecond || component.spriteDelay != 100*time.Millisecond {
		t.Fatalf("torch timing = duration %s delay %s", component.duration, component.spriteDelay)
	}
	if component.posX != 0.1 || component.posZ != 0.8 || component.sizeStart != effectTableSize(100) || component.angleStart != 270 || !component.rotateToTarget {
		t.Fatalf("torch placement = %+v", component)
	}
	if got := worldEffectSpriteAngle(component); got != 360 {
		t.Fatalf("torch effective angle = %.1f, want 360.0", got)
	}
}

func TestFireflyEffectSpecUsesFaintSpriteParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFirefly)
	if !ok {
		t.Fatal("firefly effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("firefly components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "" || component.spriteFile == "" || !component.spriteRepeat {
		t.Fatalf("firefly component = %+v", component)
	}
	if component.alphaMax > 0.25 || component.sizeEnd > effectTableSize(120) {
		t.Fatalf("firefly should stay faint and moderately sized: %+v", component)
	}
}

func TestBubbleEffectSpecMatchesRoBrowserShape(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBubble)
	if !ok {
		t.Fatal("bubble effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("bubble components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "bubble%d" || component.strRandMin != 1 || component.strRandMax != 4 {
		t.Fatalf("bubble component = %+v", component)
	}
}

func TestWorldEffectSpecLookupReturnsCopy(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	spec.sfx[0] = "mutated.wav"
	spec.components[0].textureName = "mutated"

	again, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing after mutation")
	}
	if again.sfx[0] != "effect\\ef_bash.wav" || again.components[0].textureName != "alpha_down" {
		t.Fatalf("catalog mutated: %+v", again)
	}
}

func TestResolveEffectSTRFileUsesDeterministicRandRange(t *testing.T) {
	component := worldEffectComponent{
		strFile:    "firehit%d",
		strRandMin: 1,
		strRandMax: 3,
	}
	effect := worldEffect{effectID: effectFireHit, actorID: 100, starts: time.Unix(10, 20)}
	got := resolveEffectSTRFile(component, effect, false)
	if got != "firehit1" && got != "firehit2" && got != "firehit3" {
		t.Fatalf("resolved STR file = %q, want firehit1..3", got)
	}
	if again := resolveEffectSTRFile(component, effect, false); again != got {
		t.Fatalf("resolved STR file changed from %q to %q", got, again)
	}
}

func TestResolveEffectSTRFileUsesMinFileForLessEffects(t *testing.T) {
	component := worldEffectComponent{
		strFile:    "angelus",
		strMinFile: "jong_mini",
	}
	if got := resolveEffectSTRFile(component, worldEffect{}, true); got != "jong_mini" {
		t.Fatalf("resolved STR file = %q, want jong_mini", got)
	}
}

func expectEffectIDs(t *testing.T, label string, got []int, want ...int) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s effects = %v, want %v", label, got, want)
	}
}

func TestSwordmanSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "SM_BASH begin", skillBeginEffectIDs(5), effectBashBegin)
	expectEffectIDs(t, "SM_BASH hit", skillHitEffectIDs(5), effectBashHit)
	expectEffectIDs(t, "SM_PROVOKE success", skillSuccessEffectIDs(6), effectProvoke)
	expectEffectIDs(t, "SM_MAGNUM target", skillEffectIDs(7), effectQuakeMagnum)
	expectEffectIDs(t, "SM_MAGNUM caster", skillEffectOnCasterIDs(7), effectMagnumBreak)
	expectEffectIDs(t, "SM_ENDURE", skillEffectIDs(8), effectEndure)
}

func TestNoviceSkillEffectMappings(t *testing.T) {
	if action := skillAction(db.SkillNVBasic); !action.defined || action.action != skillActorActionNone {
		t.Fatalf("basic skill action = %+v, want no source action", action)
	}
	expectEffectIDs(t, "NV_FIRSTAID", skillEffectIDs(142), effectFirstAid)
	expectEffectIDs(t, "NV_TRICKDEAD", skillEffectIDs(143))
}

func TestMageSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "MG_SIGHT success", skillSuccessEffectIDs(10))
	expectEffectIDs(t, "MG_SIGHT immediate", skillEffectIDs(10))
	expectEffectIDs(t, "MG_NAPALMBEAT hit", skillHitEffectIDs(11), effectBashHit)
	expectEffectIDs(t, "MG_SAFETYWALL ground", skillGroundEffectIDs(12))
	expectEffectIDs(t, "MG_SOULSTRIKE before-hit", skillBeforeHitEffectIDs(13), effectSoulStrike)
	expectEffectIDs(t, "MG_SOULSTRIKE hit", skillHitEffectIDs(13), effectBashHit)
	expectEffectIDs(t, "MG_COLDBOLT before-hit", skillBeforeHitEffectIDs(14), effectColdBolt)
	expectEffectIDs(t, "MG_COLDBOLT hit", skillHitEffectIDs(14), effectColdHit)
	expectEffectIDs(t, "MG_FROSTDIVER", skillEffectIDs(15), effectFrostDiver)
	expectEffectIDs(t, "MG_FROSTDIVER before-hit", skillBeforeHitEffectIDs(15))
	expectEffectIDs(t, "MG_FROSTDIVER hit", skillHitEffectIDs(15), effectFrostDiverHit)
	expectEffectIDs(t, "MG_STONECURSE", skillEffectIDs(16), effectStoneCurse)
	expectEffectIDs(t, "MG_FIREBOLT before-hit", skillBeforeHitEffectIDs(19), effectFireBolt)
	expectEffectIDs(t, "MG_FIREBALL before-hit", skillBeforeHitEffectIDs(17), effectFireBall)
	expectEffectIDs(t, "MG_FIREWALL ground", skillGroundEffectIDs(18), effectFireWall)
	for _, skillID := range []uint16{17, 18, 19} {
		expectEffectIDs(t, "fire skill hit", skillHitEffectIDs(skillID), effectFireHit)
	}
	for _, skillID := range []uint16{20, 21} {
		expectEffectIDs(t, "wind skill hit", skillHitEffectIDs(skillID), effectWindHit)
	}
	expectEffectIDs(t, "MG_LIGHTNINGBOLT before-hit", skillBeforeHitEffectIDs(20))
	expectEffectIDs(t, "MG_LIGHTNINGBOLT", skillEffectIDs(20), effectLightningBolt)
	expectEffectIDs(t, "MG_THUNDERSTORM", skillEffectIDs(21), effectThunderStorm)
	expectEffectIDs(t, "MG_THUNDERSTORM ground", skillGroundEffectIDs(21))
	expectEffectIDs(t, "MG_ENERGYCOAT", skillEffectIDs(157), effectEnergyCoat)
	expectEffectIDs(t, "MG_THUNDERSTORM before-hit", skillBeforeHitEffectIDs(21))
	for _, skillID := range []uint16{20, 21} {
		expectEffectIDs(t, "wind skill begin", skillBeginEffectIDs(skillID))
	}
}

func TestFireBoltEffectSpecUsesFallingFrameList(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBolt)
	if !ok {
		t.Fatal("fire bolt effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || len(component.textureFiles) != 6 || component.duration != 500*time.Millisecond {
		t.Fatalf("component = %+v", component)
	}
	if component.posZ != 20 || component.posZEnd != 0.0001 || component.posXStartMiddle != 5 || component.posYStartMiddle != 2 || component.angleStart != 112.5 || !component.blendAdditive {
		t.Fatalf("fire bolt trajectory = %+v", component)
	}
	if component.sizeStartX != 100*effectPixelRatio || component.sizeStartY != 50*effectPixelRatio {
		t.Fatalf("fire bolt size = %.3f x %.3f", component.sizeStartX, component.sizeStartY)
	}
}

func TestFireBallEffectSpecMatchesRobrowserProjectileAndHit(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBall)
	if !ok {
		t.Fatal("fire ball effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_fireball.wav" {
		t.Fatalf("fire ball sfx = %#v", spec.sfx)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "fireball" || !component.spriteRepeat {
		t.Fatalf("projectile sprite = %+v", component)
	}
	if component.duration != 250*time.Millisecond || component.delay != 160*time.Millisecond || component.delayOffsetDelta != -40*time.Millisecond {
		t.Fatalf("projectile timing = duration %s delay %s delta %s", component.duration, component.delay, component.delayOffsetDelta)
	}
	if component.duplicate != 5 || component.duplicateDelay != 0 {
		t.Fatalf("projectile duplicates = %d delay %s", component.duplicate, component.duplicateDelay)
	}
	if worldEffectComponentStartOffset(component, 0) != 160*time.Millisecond || worldEffectComponentStartOffset(component, 4) != 0 {
		t.Fatalf("projectile duplicate offsets = first %s last %s", worldEffectComponentStartOffset(component, 0), worldEffectComponentStartOffset(component, 4))
	}
	if worldEffectComponentDuration(spec, component) != 410*time.Millisecond {
		t.Fatalf("projectile resolved duration = %s, want 410ms", worldEffectComponentDuration(spec, component))
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.alphaMax != 0.2 || component.alphaMaxDelta != 0.2 {
		t.Fatalf("projectile orientation/alpha = %+v", component)
	}
	if component.posZ != 2 || component.sizeStart != 200*effectPixelRatio || component.sizeEnd != 200*effectPixelRatio {
		t.Fatalf("projectile position/size = %+v", component)
	}

	hitSpec, ok := worldEffectSpecForID(effectFireHit)
	if !ok || len(hitSpec.components) != 1 {
		t.Fatalf("fire hit effect missing or wrong component count: ok=%t components=%d", ok, len(hitSpec.components))
	}
	hit := hitSpec.components[0]
	if hit.kind != effectComponentSTR || hit.strFile != "firehit%d" || hit.strRandMin != 1 || hit.strRandMax != 3 || !hit.attachedEntity {
		t.Fatalf("fire hit STR = %+v", hit)
	}
	if len(hitSpec.sfx) != 1 || hitSpec.sfx[0] != "effect\\ef_firehit.wav" {
		t.Fatalf("fire hit sfx = %#v", hitSpec.sfx)
	}
}

func TestRepeatedWorldEffectDuplicateProgressUsesDuplicateDelay(t *testing.T) {
	component := worldEffectComponent{
		duration:       4 * time.Second,
		delay:          200 * time.Millisecond,
		repeat:         true,
		duplicate:      4,
		duplicateDelay: time.Second,
	}
	starts := time.Unix(1000, 0)
	now := starts.Add(2700 * time.Millisecond)
	for _, tc := range []struct {
		duplicate int
		want      float64
	}{
		{0, 0.625},
		{1, 0.375},
		{2, 0.125},
	} {
		got, active := worldEffectComponentDuplicateProgressForDraw(starts, component, tc.duplicate, component.duration, now)
		if !active {
			t.Fatalf("duplicate %d inactive, want progress %.3f", tc.duplicate, tc.want)
		}
		if math.Abs(got-tc.want) > 0.000001 {
			t.Fatalf("duplicate %d progress = %.6f, want %.6f", tc.duplicate, got, tc.want)
		}
	}
	if got, active := worldEffectComponentDuplicateProgressForDraw(starts, component, 3, component.duration, now); active {
		t.Fatalf("duplicate 3 active with progress %.6f, want delayed until its own start", got)
	}
}

func TestCylinderAnimationThreeScalesRadiusAndPulsesHeightLikeRobrowser(t *testing.T) {
	component := worldEffectComponent{
		animation:  3,
		bottomSize: 2,
		topSize:    4,
		height:     6,
	}
	for _, tc := range []struct {
		progress   float64
		wantBottom float64
		wantTop    float64
		wantHeight float64
	}{
		{0, 2, 4, 0},
		{0.25, 1.5, 3, 3},
		{0.5, 1, 2, 6},
		{0.75, 0.5, 1, 3},
	} {
		gotBottom, gotTop, gotHeight := effectCylinderAnimatedDimensions(component, tc.progress)
		if math.Abs(gotBottom-tc.wantBottom) > 0.000001 || math.Abs(gotTop-tc.wantTop) > 0.000001 || math.Abs(gotHeight-tc.wantHeight) > 0.000001 {
			t.Fatalf("progress %.2f dimensions = %.6f %.6f %.6f, want %.6f %.6f %.6f", tc.progress, gotBottom, gotTop, gotHeight, tc.wantBottom, tc.wantTop, tc.wantHeight)
		}
	}
}

func TestBashHitEffectSpecMatchesRobrowserLensCircle(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashHit)
	if !ok {
		t.Fatal("bash hit effect missing")
	}
	if spec.duration != 350*time.Millisecond {
		t.Fatalf("duration = %s, want 350ms", spec.duration)
	}
	if len(spec.components) != 8 {
		t.Fatalf("components = %d, want 8 reference client lens slashes", len(spec.components))
	}
	for i, component := range spec.components {
		if component.kind != effectComponent2D {
			t.Fatalf("component %d kind = %d, want 2D", i, component.kind)
		}
		wantTexture := "effect/lens1.tga"
		if i%2 == 1 {
			wantTexture = "effect/lens2.tga"
		}
		if component.textureFile != wantTexture || component.duration != 250*time.Millisecond || !component.fadeOut || !component.overlay {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.durationRandMin != 200*time.Millisecond || component.durationRandMax != 350*time.Millisecond {
			t.Fatalf("component %d duration rand = %s..%s", i, component.durationRandMin, component.durationRandMax)
		}
		if component.sizeStartXRandMin != 25*effectPixelRatio || component.sizeStartXRandMax != 40*effectPixelRatio {
			t.Fatalf("component %d start x range = %.3f..%.3f", i, component.sizeStartXRandMin, component.sizeStartXRandMax)
		}
		if component.sizeStartY != 10*effectPixelRatio || component.sizeEndX != 1*effectPixelRatio {
			t.Fatalf("component %d fixed axis sizes = %.3f %.3f", i, component.sizeStartY, component.sizeEndX)
		}
		if component.sizeEndYRandMin != 250*effectPixelRatio || component.sizeEndYRandMax != 300*effectPixelRatio {
			t.Fatalf("component %d end y range = %.3f..%.3f", i, component.sizeEndYRandMin, component.sizeEndYRandMax)
		}
		if !component.circlePattern || component.circleInnerSize != 2.2 || component.circleOuterRandMin != 5 || component.circleOuterRandMax != 6 {
			t.Fatalf("component %d circle pattern = %+v", i, component)
		}
		if component.angleRandMax <= component.angleRandMin {
			t.Fatalf("component %d angle range = %.1f..%.1f", i, component.angleRandMin, component.angleRandMax)
		}
	}
	mode := &WorldMode{}
	effect := worldEffect{effectID: effectBashHit, actorID: 300, starts: time.Unix(10, 20)}
	startX, startY, _ := mode.effect3DOffset(client.Context{}, spec.components[0], effect, 0, 0, 0, 0, 0, 0)
	endX, endY, _ := mode.effect3DOffset(client.Context{}, spec.components[0], effect, 0, 0, 1, 0, 0, 0)
	if math.Hypot(endX, endY) <= math.Hypot(startX, startY) {
		t.Fatalf("circle pattern does not move outward: start=(%.2f,%.2f) end=(%.2f,%.2f)", startX, startY, endX, endY)
	}
}

func TestRegularHitEffectKeepsRobrowserParticleBurst(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHit1)
	if !ok || len(spec.components) != 2 {
		t.Fatalf("regular hit spec = %+v ok=%t, want sparks and a ring", spec, ok)
	}
	if spec.duration != 300*time.Millisecond {
		t.Fatalf("duration = %s, want 300ms", spec.duration)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/pok3.tga" || component.duration != 300*time.Millisecond {
		t.Fatalf("component resource/timing = %+v", component)
	}
	if component.duplicate != 4 || component.alphaMax != 0.8 || !component.fadeIn || !component.fadeOut || !component.sparkling {
		t.Fatalf("component duplicate/fade = %+v", component)
	}
	if component.posZ != 1 || component.posXEndRand != 2 || component.posYEndRand != 2 || component.posZEndRand != 2 {
		t.Fatalf("component position = %+v", component)
	}
	if component.sizeStart != effectTableSize(10) || component.sizeEnd != effectTableSize(10) || component.sizeRand != effectTableSize(20) || !component.sizeSmooth {
		t.Fatalf("component size = %+v", component)
	}
}

func TestSkillHitSlashSpecsMatchRobrowser(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		kind     effectComponentKind
		width    float64
		height   float64
		sfx      string
		overlay  bool
	}{
		{"hit5", effectHit5, effectComponent3D, effectTableSize(15), effectTableSize(200), "effect\\ef_hit5.wav", false},
		{"hit6", effectHit6, effectComponent2D, effectTableSize(10), effectTableSize(150), "effect\\ef_hit6.wav", true},
	} {
		spec, ok := worldEffectSpecForID(tc.effectID)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two slash components", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.sfx {
			t.Fatalf("%s sfx = %v", tc.name, spec.sfx)
		}
		for i, component := range spec.components {
			if component.kind != tc.kind || component.textureFile != "effect/lens2.tga" || component.duration != 400*time.Millisecond || component.alphaMax != 1 || !component.fadeOut || !component.rotate || component.overlay != tc.overlay {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.posZ != 1 || component.sizeStartX != tc.width || component.sizeEndX != tc.width || component.sizeStartY != effectTableSize(10) || component.sizeEndY != tc.height {
				t.Fatalf("%s component %d size/position = %+v", tc.name, i, component)
			}
		}
		if spec.components[0].angleStart != 90 || spec.components[0].angleEnd != 0 || spec.components[1].angleStart != 180 || spec.components[1].angleEnd != 90 {
			t.Fatalf("%s slash angles = %+v %+v", tc.name, spec.components[0], spec.components[1])
		}
	}
}

func TestColdBoltEffectSpecMatchesRobrowserProjectileAndRing(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectColdBolt)
	if !ok {
		t.Fatal("cold bolt effect missing")
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	projectile := spec.components[0]
	if projectile.kind != effectComponent3D || projectile.textureFile != "effect/icearrow.tga" || projectile.duration != 500*time.Millisecond {
		t.Fatalf("projectile = %+v", projectile)
	}
	if projectile.posZ != 20 || projectile.posZEnd != 0.0001 || projectile.posXStartMiddle != 5 || projectile.posYStartMiddle != 2 || projectile.sizeStart != 50*effectPixelRatio {
		t.Fatalf("cold bolt projectile trajectory = %+v", projectile)
	}
	ring := spec.components[1]
	if ring.kind != effectComponentCylinder || ring.textureName != "ring_blue" || ring.delay != 500*time.Millisecond || ring.duration != 1000*time.Millisecond {
		t.Fatalf("ring = %+v", ring)
	}
	if ring.bottomSize != 3 || ring.topSize != 5 || ring.animation != 4 {
		t.Fatalf("cold bolt ring dimensions = %+v", ring)
	}
}

func TestSightEffectSpecOrbitsAroundActor(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectSight)
	if !ok {
		t.Fatal("sight effect missing")
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want robr shadow + sight sprite", len(spec.components))
	}
	shadow := spec.components[0]
	if !shadow.shadowTexture || shadow.spriteFile != "data\\sprite\\shadow" || shadow.duplicate != 10 {
		t.Fatalf("sight shadow component = %+v", shadow)
	}
	if shadow.sizeStart != 30*effectPixelRatio || shadow.sizeDelta != 10*effectPixelRatio {
		t.Fatalf("sight shadow size = %.3f delta %.3f", shadow.sizeStart, shadow.sizeDelta)
	}
	component := spec.components[1]
	if component.spriteFile != "sight" || component.duplicate != 10 || component.orbitRadiusX != 3 || component.orbitRadiusY != 3 || component.orbitRotations != 10 {
		t.Fatalf("sight orbit component = %+v", component)
	}
	if component.sizeStart != 60*effectPixelRatio || component.sizeDelta != 20*effectPixelRatio || component.alphaMaxDelta != 3.0/255.0 {
		t.Fatalf("sight orbit size/alpha delta = %.3f delta %.3f alpha_delta %.3f", component.sizeStart, component.sizeDelta, component.alphaMaxDelta)
	}
	ctx := client.Context{}
	effect := worldEffect{effectID: effectSight, actorID: 2000000}
	mode := &WorldMode{}
	x0, y0, _ := mode.effect3DOffset(ctx, component, effect, 0, 0, 0, 0, 0, 0)
	x1, y1, _ := mode.effect3DOffset(ctx, component, effect, 0, 1, 0, 0, 0, 0)
	x2, y2, _ := mode.effect3DOffset(ctx, component, effect, 0, 0, 0.025, 0, 0, 0)
	if math.Hypot(x0-x1, y0-y1) < 0.1 {
		t.Fatalf("sight duplicates overlap: duplicate0=(%.3f,%.3f) duplicate1=(%.3f,%.3f)", x0, y0, x1, y1)
	}
	if math.Hypot(x0-x2, y0-y2) < 0.1 {
		t.Fatalf("sight orbit does not move over time: start=(%.3f,%.3f) later=(%.3f,%.3f)", x0, y0, x2, y2)
	}
}

func TestFireBallSpriteRotationUsesRobrowserWorldTrajectory(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectFireBall)
	if !ok || len(spec.components) == 0 {
		t.Fatal("fire ball effect missing")
	}
	component := spec.components[0]
	tests := []struct {
		name      string
		sourceX   int
		sourceY   int
		targetX   int
		targetY   int
		cameraYaw float64
		want      float64
	}{
		{name: "same row", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, want: -math.Pi / 2},
		{name: "same column", sourceX: 12, sourceY: 18, targetX: 12, targetY: 20, want: 0},
		{name: "diagonal", sourceX: 10, sourceY: 18, targetX: 12, targetY: 20, want: -math.Pi / 4},
		{name: "camera yaw", sourceX: 10, sourceY: 20, targetX: 12, targetY: 20, cameraYaw: 45, want: -3 * math.Pi / 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: tt.sourceX, Y: tt.sourceY}
			world.UpsertActor(worldstate.Actor{ID: 300, X: tt.targetX, Y: tt.targetY})
			ctx := client.Context{
				Session: &session.Session{AccountID: 2000000, CharID: 150000},
				World:   world,
			}
			effect := worldEffect{effectID: effectFireBall, actorID: 300, targetID: 2000000}
			startX, startY, _, endX, endY, _, ok := effectTrajectoryEndpoints(ctx, component, effect)
			if !ok {
				t.Fatal("trajectory endpoints missing")
			}
			if math.Hypot(endX-startX, endY-startY) <= 0.001 {
				t.Fatalf("trajectory did not span caster and target: %.2f,%.2f -> %.2f,%.2f", startX, startY, endX, endY)
			}
			projection := newSceneProjectionForTargetYaw(800, 600, float64(tt.targetX), float64(tt.targetY), 0, tt.cameraYaw)
			angle, ok := effectSpriteRobrowserRotation(ctx, projection, component, effect, 0)
			if !ok {
				t.Fatal("rotation missing")
			}
			wantFromFormula := -(90 - math.Atan2(endY-startY, endX-startX)*180/math.Pi + tt.cameraYaw) * math.Pi / 180
			if math.Abs(angle-wantFromFormula) > 0.001 {
				t.Fatalf("angle = %.3f, want robr formula %.3f", angle, wantFromFormula)
			}
			if math.Abs(angle-tt.want) > 0.001 {
				t.Fatalf("angle = %.3f, want %.3f", angle, tt.want)
			}
		})
	}
}

func TestGroundSampleEffectSpecUsesMagicTargetPlane(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectGroundSample)
	if !ok {
		t.Fatal("ground sample effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncGroundSample || component.funcName != "MagicTarget" || component.textureFile != "effect/magic_target.tga" || component.sizeStart != 1 {
		t.Fatalf("component = %+v", component)
	}
	if component.duration != 0 {
		t.Fatalf("ground sample component duration = %s, want inherited cast duration", component.duration)
	}
}

func TestGroundSampleRotationAngleUsesClientSpeedOverride(t *testing.T) {
	start := time.Unix(10, 0)
	now := start.Add(time.Second)
	speed := 0.5 / 7 * 60
	effect := worldEffect{starts: start, groundSampleRotationRadiansPerSecond: speed}
	if got := groundSampleRotationAngle(effect, now); math.Abs(got-speed) > 0.001 {
		t.Fatalf("override angle = %.4f, want %.4f", got, speed)
	}
	fallback := worldEffect{starts: start}
	wantFallback := 40 * math.Pi / 180
	if got := groundSampleRotationAngle(fallback, now); math.Abs(got-wantFallback) > 0.001 {
		t.Fatalf("fallback angle = %.4f, want %.4f", got, wantFallback)
	}
}

func TestGroundSampleDrawOptionsClampRotatedMagicTargetTexture(t *testing.T) {
	options := groundSampleDrawOptions()
	if options.Filter != render.FilterLinear {
		t.Fatalf("filter = %v, want linear", options.Filter)
	}
	if options.Address != render.AddressClampToZero {
		t.Fatalf("address = %v, want clamp-to-zero to avoid repeated rotated MagicTarget corners", options.Address)
	}
	if !options.DepthTest {
		t.Fatal("ground sample depth test is disabled")
	}
}

func TestCastRingEffectSpecUsesMagicRingCylinder(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectCastRing)
	if !ok {
		t.Fatal("cast ring effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncCastRing || component.funcName != "CastRing" || component.textureName != "ring_yellow" {
		t.Fatalf("component = %+v", component)
	}
	if component.bottomSize != 0.8 || component.topSize != 2.45 || component.height != 2.8 {
		t.Fatalf("magic ring dimensions = bottom %.2f top %.2f height %.2f", component.bottomSize, component.topSize, component.height)
	}
	if component.duration != 0 {
		t.Fatalf("cast ring component duration = %s, want inherited cast duration", component.duration)
	}
}

func TestLockOnTargetEffectSpecMatchesRobrowserCastTargetCircle(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectLockOnTarget)
	if !ok {
		t.Fatal("lock-on target effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentFUNC || component.funcAdapter != effectFuncLockOnTarget || component.funcName != "LockOnTarget" || component.textureFile != "effect/lockon128.tga" || !component.attachedEntity {
		t.Fatalf("component = %+v", component)
	}
	start := time.Unix(10, 0)
	if got := lockOnTargetSize(start, start); got != 15 {
		t.Fatalf("initial lock-on size = %.1f, want 15", got)
	}
	if got := lockOnTargetSize(start, start.Add(250*time.Millisecond)); got != 3 {
		t.Fatalf("settled lock-on size = %.1f, want 3", got)
	}
	if got := lockOnTargetTint(start, start); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("initial lock-on tint = %+v", got)
	}
	if got := lockOnTargetTint(start, start.Add(380*time.Millisecond)); got != (color.RGBA{R: 255, G: 12, B: 12, A: 255}) {
		t.Fatalf("low lock-on tint = %+v", got)
	}
}

func TestBeginSpellEffectsInheritCastDuration(t *testing.T) {
	for _, effectID := range []int{effectBeginSpell, effectBeginSpell2, effectBeginSpell3, effectBeginSpell4, effectBeginSpell5, effectBeginSpell6, effectBeginSpell7} {
		spec, ok := worldEffectSpecForID(effectID)
		if !ok {
			t.Fatalf("begin spell effect %d missing", effectID)
		}
		for i, component := range spec.components {
			if component.duration != 0 {
				t.Fatalf("effect %d component %d duration = %s, want inherited cast duration", effectID, i, component.duration)
			}
		}
	}
}

func TestApplyActorActionNotifyRepeatsFireBoltHits(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     19,
		SkillLevel:  4,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		Damage:      1008,
		HitCount:    4,
		Action:      network.ActorActionSkill,
	})

	if len(mode.worldEffects) != 8 {
		t.Fatalf("world effects = %d, want 8", len(mode.worldEffects))
	}
	for i := 0; i < 4; i++ {
		effect := mode.worldEffects[i]
		if effect.effectID != effectFireBolt || effect.actorID != 300 || effect.targetID != 2000000 {
			t.Fatalf("before-hit effect %d = %+v", i, effect)
		}
		if i > 0 {
			if delay := effect.starts.Sub(mode.worldEffects[i-1].starts); delay != multiHitDelay {
				t.Fatalf("before-hit delay %d = %s, want %s", i, delay, multiHitDelay)
			}
		}
	}
	for i := 4; i < 8; i++ {
		effect := mode.worldEffects[i]
		if effect.effectID != effectFireHit || effect.actorID != 300 {
			t.Fatalf("hit effect %d = %+v", i, effect)
		}
		if i > 4 {
			if delay := effect.starts.Sub(mode.worldEffects[i-1].starts); delay != multiHitDelay {
				t.Fatalf("hit delay %d = %s, want %s", i, delay, multiHitDelay)
			}
		}
	}
	if len(mode.damageFloaters) != 8 {
		t.Fatalf("damage floaters = %d, want 8", len(mode.damageFloaters))
	}
	wantFloaters := []struct {
		text string
		kind damageFloaterKind
	}{
		{text: "252", kind: damageFloaterNormal},
		{text: "252", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "504", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "756", kind: damageFloaterCombo},
		{text: "252", kind: damageFloaterNormal},
		{text: "1008", kind: damageFloaterCombo},
	}
	for i, want := range wantFloaters {
		floater := mode.damageFloaters[i]
		if floater.text != want.text || floater.kind != want.kind {
			t.Fatalf("floater %d = %+v, want text=%q kind=%d", i, floater, want.text, want.kind)
		}
		if i > 1 {
			if delay := floater.starts.Sub(mode.damageFloaters[i-2].starts); delay != multiHitDelay {
				t.Fatalf("floater %d delay = %s, want %s", i, delay, multiHitDelay)
			}
		}
		if floater.kind == damageFloaterCombo {
			if floater.duration != damageFloaterDuration(damageFloaterCombo) {
				t.Fatalf("combo floater %d duration = %s, want %s", i, floater.duration, damageFloaterDuration(damageFloaterCombo))
			}
			wantVisible := damageFloaterComboTransientDuration()
			if i == len(wantFloaters)-1 {
				wantVisible = damageFloaterDuration(damageFloaterCombo)
			}
			if visible := floater.expires.Sub(floater.starts); visible != wantVisible {
				t.Fatalf("combo floater %d visible = %s, want %s", i, visible, wantVisible)
			}
		}
	}
}

func TestLightningBoltStartsWithSkillAction(t *testing.T) {
	for _, tt := range []struct {
		name   string
		hits   uint16
		damage int32
	}{
		{"single hit", 1, 100},
		{"four hits", 4, 400},
		{"ten hits", 10, 1000},
		{"zero damage", 4, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			world := worldstate.New()
			world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
			world.UpsertActor(worldstate.Actor{ID: 300, X: 11, Y: 20, Job: 1002, ObjectType: actorObjectTypeMob, HasObjectType: true})
			mode := &WorldMode{}
			ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
			mode.applyActorActionNotify(ctx, network.ActorActionNotify{
				SkillID: db.SkillMGLightningbolt, SkillLevel: tt.hits,
				SourceID: 2000000, TargetID: 300, SourceSpeed: 580, TargetSpeed: 480,
				Damage: tt.damage, HitCount: tt.hits, Action: network.ActorActionSkill,
			})

			hitCount := int(tt.hits)
			if tt.damage == 0 {
				hitCount = 0
			}
			if len(mode.worldEffects) != 1+hitCount {
				t.Fatalf("effects = %d, want one lightning animation and %d impacts", len(mode.worldEffects), hitCount)
			}
			started := mode.actorAnims[2000000].started
			bolt := mode.worldEffects[0]
			if started.IsZero() || bolt.effectID != effectLightningBolt || bolt.actorID != 300 || !bolt.starts.Equal(started) {
				t.Fatalf("lightning = %+v, want one animation on target starting with caster at %s", bolt, started)
			}
			for i, hit := range mode.worldEffects[1:] {
				want := started.Add(580*time.Millisecond + time.Duration(i)*200*time.Millisecond)
				if hit.effectID != effectWindHit || hit.actorID != 300 || !hit.starts.Equal(want) {
					t.Fatalf("impact %d = %+v, want wind hit on target at %s", i, hit, want)
				}
			}
			var damageTimes []time.Time
			for _, floater := range mode.damageFloaters {
				if floater.kind == damageFloaterNormal {
					damageTimes = append(damageTimes, floater.starts)
				}
			}
			if len(damageTimes) != hitCount {
				t.Fatalf("damage numbers = %d, want %d", len(damageTimes), hitCount)
			}
			for i, hitAt := range damageTimes {
				if !hitAt.Equal(mode.worldEffects[i+1].starts) {
					t.Fatalf("damage %d does not coincide with its impact", i)
				}
			}
		})
	}
}

func TestLightningBoltHasNoExtraImpactFlash(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectLightningBolt)
	if !ok || len(spec.components) != 1 || spec.components[0].strFile != "lightning" {
		t.Fatal("lightning must play only its STR; impact flashes are scheduled per damage hit")
	}
}

func TestZeroDamageSkillKeepsExecutionEffectsWithoutHitReaction(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGSoulstrike,
		SkillLevel:  8,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    4,
		Action:      8,
	})

	if len(mode.worldEffects) != 4 {
		t.Fatalf("world effects = %+v, want four Soul Strike execution effects", mode.worldEffects)
	}
	for i, effect := range mode.worldEffects {
		if effect.effectID != effectSoulStrike || effect.actorID != 300 || effect.targetID != 2000000 {
			t.Fatalf("execution effect %d = %+v, want Soul Strike toward target", i, effect)
		}
	}
	if _, ok := mode.actorAnims[300]; ok {
		t.Fatal("zero-damage target received a hurt animation")
	}
	if len(mode.scheduledSounds) != 4 {
		t.Fatalf("scheduled sounds = %+v, want four Soul Strike effect sounds", mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != "effect\\ef_soulstrike.wav" {
			t.Fatalf("scheduled sound %d = %+v, want only the Soul Strike effect sound", i, sound)
		}
	}
	if len(mode.damageFloaters) != 1 || mode.damageFloaters[0].text != "miss" || mode.damageFloaters[0].kind != damageFloaterMiss {
		t.Fatalf("damage floaters = %+v, want one miss", mode.damageFloaters)
	}
}

func TestZeroDamageSkillKeepsImmediateEffectWithoutHitEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             11,
		Y:             20,
		Job:           1063,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{
		Session: &session.Session{AccountID: 2000000, CharID: 150000},
		World:   world,
	}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SkillID:     db.SkillMGFrostdiver,
		SkillLevel:  1,
		SourceID:    2000000,
		TargetID:    300,
		SourceSpeed: 580,
		TargetSpeed: 480,
		HitCount:    1,
		Action:      network.ActorActionSkill,
	})

	if len(mode.worldEffects) != 1 || mode.worldEffects[0].effectID != effectFrostDiver {
		t.Fatalf("world effects = %+v, want Frost Diver execution effect without hit effect", mode.worldEffects)
	}
	wantStart := mode.actorAnims[2000000].started.Add(580 * time.Millisecond)
	if !mode.worldEffects[0].starts.Equal(wantStart) {
		t.Fatalf("default effect start = %s, want hit time %s", mode.worldEffects[0].starts, wantStart)
	}
}

func TestSkillCastAuraEffectMappings(t *testing.T) {
	tests := []struct {
		property uint32
		want     int
	}{
		{property: 0, want: effectBeginSpell},
		{property: 1, want: effectBeginSpell2},
		{property: 2, want: effectBeginSpell5},
		{property: 3, want: effectBeginSpell3},
		{property: 4, want: effectBeginSpell4},
		{property: 5, want: effectBeginSpell7},
		{property: 6, want: effectBeginSpell6},
		{property: 8, want: effectBeginSpell6},
		{property: 9, want: effectBeginSpell},
	}
	for _, tt := range tests {
		if got := skillCastAuraEffectID(tt.property); got != tt.want {
			t.Fatalf("cast aura property %d = %d, want %d", tt.property, got, tt.want)
		}
	}
}

func TestSkillVisualMetadataMappings(t *testing.T) {
	if skillAction(5).action != skillActorActionAttack || skillAction(7).action != skillActorActionAttack {
		t.Fatalf("swordman weapon-action skills = bash:%d magnum:%d", skillAction(5).action, skillAction(7).action)
	}
	if skillAction(8).action != skillActorActionReadyFight {
		t.Fatalf("endure action = %d, want ready fight", skillAction(8).action)
	}
	if skillAction(28).action != skillActorActionSkill {
		t.Fatalf("heal action = %d, want default skill action", skillAction(28).action)
	}
	defaultAction := skillAction(28)
	if !defaultAction.play || defaultAction.repeat || defaultAction.next == nil || defaultAction.next.action != skillActorActionIdle || !defaultAction.next.repeat {
		t.Fatalf("default skill action shape = %+v next=%+v, want robr-style skill action followed by repeating idle", defaultAction, defaultAction.next)
	}
	if size := skillCastGroundSampleSize(19); size != 1 {
		t.Fatalf("firebolt marker size = %.1f, want default 1", size)
	}
	for skillID, wantSize := range map[uint16]float64{
		db.SkillACShower:        5,
		db.SkillALPneuma:        5,
		db.SkillASGrimtooth:     5,
		db.SkillASVenomdust:     5,
		db.SkillBSHammerfall:    5,
		db.SkillCRSlimpitcher:   9,
		db.SkillHTBlitzbeat:     5,
		db.SkillHTDetecting:     5,
		db.SkillHWGanbantein:    5,
		db.SkillHWGravitation:   7,
		db.SkillMGFireball:      7,
		db.SkillMGFirewall:      2,
		db.SkillMGNapalmbeat:    5,
		db.SkillMGThunderstorm:  7,
		db.SkillPRBenedictio:    5,
		db.SkillPRMagnus:        9,
		db.SkillPRSanctuary:     7,
		db.SkillSNFalconassault: 5,
		db.SkillWZFirepillar:    5,
		db.SkillWZFrostnova:     5,
		db.SkillWZHeavendrive:   7,
		db.SkillWZMeteor:        11,
		db.SkillWZQuagmire:      7,
		db.SkillWZStormgust:     11,
		db.SkillWZVermilion:     13,
		db.SkillWZWaterball:     5,
	} {
		if size := skillCastGroundSampleSize(skillID); size != wantSize {
			t.Fatalf("skill %d marker size = %.1f, want classic client rendered scope %.1f", skillID, size, wantSize)
		}
		gotSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		wantSpeed := 0.5 / wantSize * 60
		if math.Abs(gotSpeed-wantSpeed) > 0.001 {
			t.Fatalf("skill %d marker speed = %.4f rad/s, want %.4f", skillID, gotSpeed, wantSpeed)
		}
	}
	if speed := skillCastGroundSampleRotationRadiansPerSecond(19); speed != 0 {
		t.Fatalf("default marker speed override = %.4f, want fallback path", speed)
	}
}

func TestWindHitEffectSpecMatchesRobrowserRandomSTRAndSFX(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectWindHit)
	if !ok {
		t.Fatal("wind hit effect missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "windhit%d" || component.strRandMin != 1 || component.strRandMax != 3 || !component.attachedEntity {
		t.Fatalf("wind hit component = %+v", component)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_hit_fist%d.wav" || spec.sfxRandMin != 1 || spec.sfxRandMax != 3 {
		t.Fatalf("wind hit sfx = %+v rand=%d..%d", spec.sfx, spec.sfxRandMin, spec.sfxRandMax)
	}

	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}
	if !mode.addWorldEffectAt(ctx, effectWindHit, 2000000, time.Unix(10, 20)) {
		t.Fatal("wind hit effect was not added")
	}
	if len(mode.scheduledSounds) != 1 || len(mode.scheduledSounds[0].paths) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	path := mode.scheduledSounds[0].paths[0]
	if strings.Contains(path, "%d") || !strings.HasPrefix(path, "_hit_fist") || !strings.HasSuffix(path, ".wav") {
		t.Fatalf("scheduled wind hit sound path = %q", path)
	}
}

func TestSkillCastNotifyAddsDurationAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 20, Property: 4, DelayTime: 2500})

	if len(mode.worldEffects) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.worldEffects))
	}
	circle := mode.worldEffects[0]
	if circle.effectID != effectCastRing || circle.actorID != 2000000 || circle.targetID != 0 || circle.duration != 2500*time.Millisecond {
		t.Fatalf("circle = %+v", circle)
	}
	lockon := mode.worldEffects[1]
	if lockon.effectID != effectLockOnTarget || lockon.actorID != 1100 || lockon.targetID != 0 || lockon.duration != 2500*time.Millisecond {
		t.Fatalf("lockon = %+v", lockon)
	}
	aura := mode.worldEffects[2]
	if aura.effectID != effectBeginSpell4 || aura.actorID != 2000000 || aura.targetID != 1100 || aura.duration != 2500*time.Millisecond {
		t.Fatalf("aura = %+v", aura)
	}
	bar, ok := mode.actorCastBars[150000]
	if !ok {
		t.Fatal("local cast bar missing")
	}
	if bar.duration != 2500*time.Millisecond || bar.color != (color.RGBA{R: 0, G: 255, B: 0, A: 255}) {
		t.Fatalf("cast bar = %+v", bar)
	}
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("cast animation missing")
	}
	if anim.actionFamily != spriteActionPCReadyFight || anim.duration != 2500*time.Millisecond || anim.hasFixedMotion {
		t.Fatalf("cast animation = %+v", anim)
	}
	if world.Dir != directionFromDelta(10, 20, 12, 20, 4) {
		t.Fatalf("cast dir = %d", world.Dir)
	}
}

func TestSkillCastNotifyHonorsHideCastAura(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: db.SkillACChargearrow, Property: 4, DelayTime: 1200})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and target lock-on", mode.worldEffects)
	}
	if mode.worldEffects[0].effectID != effectCastRing {
		t.Fatalf("effect = %+v, want cast ring", mode.worldEffects[0])
	}
	if mode.worldEffects[1].effectID != effectLockOnTarget || mode.worldEffects[1].actorID != 1100 {
		t.Fatalf("effect = %+v, want target lock-on", mode.worldEffects[1])
	}
	if _, ok := mode.actorCastBars[150000]; !ok {
		t.Fatal("cast bar should remain visible when only hideCastAura is set")
	}
}

func TestSelfTargetSkillCastDoesNotAddLockOnTarget(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.addSkillCastEffects(ctx, db.SkillMGFireball, 3, 2000000, 2000000, 0, 0, 900*time.Millisecond, time.Now(), "self")

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %+v, want cast ring and aura", mode.worldEffects)
	}
	if mode.worldEffects[0].effectID != effectCastRing || mode.worldEffects[1].effectID != effectBeginSpell3 {
		t.Fatalf("effects = %+v", mode.worldEffects)
	}
}

func TestSkillCastEffectsDedupeServerAndLocalFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start, "local")
	mode.addSkillCastEffects(ctx, 19, 3, 2000000, 1100, 0, 0, 2800*time.Millisecond, start.Add(20*time.Millisecond), "server")

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectCastRing || mode.worldEffects[1].effectID != effectBeginSpell3 {
		t.Fatalf("effects = %+v", mode.worldEffects)
	}
}

func TestSkillResultAnimationReplacesCastStance(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20, Job: 4, Dir: 4}
	world.UpsertActor(worldstate.Actor{ID: 1100, X: 12, Y: 20})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000, CharID: 150000}, World: world}

	mode.applySkillCastNotify(ctx, network.SkillCastNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, DelayTime: 1200})
	if anim, ok := mode.actorAnims[150000]; !ok || anim.actionFamily != spriteActionPCReadyFight {
		t.Fatalf("cast stance animation = %+v ok=%t", anim, ok)
	}

	mode.applySkillNoDamageNotify(ctx, network.SkillNoDamageNotify{SourceID: 2000000, TargetID: 1100, SkillID: 28, Amount: 234, Result: 1})
	anim, ok := mode.actorAnims[150000]
	if !ok {
		t.Fatal("skill result animation missing")
	}
	if anim.actionFamily != spriteActionPCSkill || anim.hasFixedMotion {
		t.Fatalf("skill result animation = %+v, want delivery skill action", anim)
	}
}

func TestGroundSkillCastEffectsAddGroundSampleMarker(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	start := time.Now()
	mode.addSkillCastEffects(ctx, 21, 4, 2000000, 0, 123, 456, 1800*time.Millisecond, start, "local-ground")

	if len(mode.worldEffects) != 3 {
		t.Fatalf("world effects = %d, want 3", len(mode.worldEffects))
	}
	marker := mode.worldEffects[0]
	wantSpeed := 0.5 / 7 * 60
	if marker.effectID != effectGroundSample || marker.actorID != 0 || marker.x != 123 || marker.y != 456 || marker.duration != 1800*time.Millisecond || marker.size != 7 || math.Abs(marker.groundSampleRotationRadiansPerSecond-wantSpeed) > 0.001 {
		t.Fatalf("ground marker = %+v", marker)
	}
	if mode.worldEffects[1].effectID != effectCastRing || mode.worldEffects[2].effectID != effectBeginSpell4 {
		t.Fatalf("cast effects = %+v", mode.worldEffects)
	}
}

func TestAcolyteSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_DP passive", skillEffectIDs(22))
	expectEffectIDs(t, "AL_DEMONBANE passive", skillEffectIDs(23))
	expectEffectIDs(t, "AL_RUWACH hit", skillHitEffectIDs(24), effectBashHit)
	expectEffectIDs(t, "AL_PNEUMA ground", skillGroundEffectIDs(25), effectPneuma)
	expectEffectIDs(t, "AL_TELEPORT", skillEffectIDs(26))
	expectEffectIDs(t, "AL_WARP", skillEffectIDs(27))
	expectEffectIDs(t, "AL_HEAL", skillEffectIDs(28), effectHeal)
	expectEffectIDs(t, "AL_HEAL hit", skillHitEffectIDs(28), effectHealOffensive)
	expectEffectIDs(t, "AL_INCAGI", skillEffectIDs(29), effectIncAgility)
	expectEffectIDs(t, "AL_DECAGI", skillEffectIDs(30), effectDecAgility)
	expectEffectIDs(t, "AL_HOLYWATER", skillEffectIDs(31), effectAqua)
	expectEffectIDs(t, "AL_CRUCIS", skillEffectIDs(32), effectSignum)
	expectEffectIDs(t, "AL_ANGELUS", skillEffectIDs(33), effectAngelus)
	expectEffectIDs(t, "AL_BLESSING", skillEffectIDs(34), effectBlessing)
	expectEffectIDs(t, "AL_CURE", skillEffectIDs(35), effectCure)
}

func TestMercenarySupportSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "MER_MAGNIFICAT", skillEffectIDs(db.SkillMerMagnificat), effectMagnificat)
	expectEffectIDs(t, "MER_QUICKEN", skillEffectIDs(db.SkillMerQuicken), effectTwoHandQuicken)
	expectEffectIDs(t, "MER_PROVOKE success", skillSuccessEffectIDs(db.SkillMerProvoke), effectProvoke)
	expectEffectIDs(t, "MER_DECAGI", skillEffectIDs(db.SkillMerDecagi), effectDecAgility)
	expectEffectIDs(t, "MER_LEXDIVINA", skillEffectIDs(db.SkillMerLexdivina), effectLexDivina)
	expectEffectIDs(t, "MER_ESTIMATION", skillEffectIDs(db.SkillMerEstimation))
	expectEffectIDs(t, "MER_KYRIE", skillEffectIDs(db.SkillMerKyrie), effectKyrie)
	expectEffectIDs(t, "MER_BLESSING", skillEffectIDs(db.SkillMerBlessing), effectBlessing)
	expectEffectIDs(t, "MER_INCAGI", skillEffectIDs(db.SkillMerIncagi), effectIncAgility)

	for _, tc := range []struct {
		name    string
		skillID uint16
	}{
		{"MER_SIGHT", db.SkillMerSight},
		{"MER_CRASH", db.SkillMerCrash},
		{"MER_REGAIN", db.SkillMerRegain},
		{"MER_TENDER", db.SkillMerTender},
		{"MER_BENEDICTION", db.SkillMerBenediction},
		{"MER_RECUPERATE", db.SkillMerRecuperate},
		{"MER_MENTALCURE", db.SkillMerMentalcure},
		{"MER_COMPRESS", db.SkillMerCompress},
		{"MER_AUTOBERSERK", db.SkillMerAutoberserk},
		{"MER_SCAPEGOAT", db.SkillMerScapegoat},
	} {
		expectEffectIDs(t, tc.name, skillEffectIDs(tc.skillID))
		expectEffectIDs(t, tc.name+" success", skillSuccessEffectIDs(tc.skillID))
	}
}

func TestImportedSkillEffectFallback(t *testing.T) {
	expectEffectIDs(t, "PR_IMPOSITIO imported", skillEffectIDs(db.SkillPRImpositio), effectImpositio)
	expectEffectIDs(t, "ALL_RESURRECTION imported", skillEffectIDs(db.SkillALLResurrection), effectResurrection, 140)
	expectEffectIDs(t, "PR_SUFFRAGIUM imported", skillEffectIDs(db.SkillPRSuffragium), effectSuffragium)
	expectEffectIDs(t, "PR_ASPERSIO imported", skillEffectIDs(db.SkillPRAspersio), effectAspersio)
	expectEffectIDs(t, "PR_BENEDICTIO imported", skillEffectIDs(db.SkillPRBenedictio), effectBenedictio)
	expectEffectIDs(t, "PR_SANCTUARY imported", skillEffectIDs(db.SkillPRSanctuary), effectSanctuary)
	expectEffectIDs(t, "PR_KYRIE imported", skillEffectIDs(db.SkillPRKyrie), effectKyrie)
	expectEffectIDs(t, "PR_MAGNIFICAT imported", skillEffectIDs(db.SkillPRMagnificat), effectMagnificat)
	expectEffectIDs(t, "PR_GLORIA imported", skillEffectIDs(db.SkillPRGloria), effectGloria)
	expectEffectIDs(t, "PR_LEXDIVINA imported", skillEffectIDs(db.SkillPRLexdivina), effectLexDivina)
	expectEffectIDs(t, "PR_LEXAETERNA imported", skillEffectIDs(db.SkillPRLexaeterna), effectLexAeterna)
	expectEffectIDs(t, "PR_TURNUNDEAD imported hit", skillHitEffectIDs(db.SkillPRTurnundead), effectHolyLight)
	expectEffectIDs(t, "PR_MAGNUS imported", skillEffectIDs(db.SkillPRMagnus), effectMagnus)
	expectEffectIDs(t, "PR_MAGNUS imported ground", skillGroundEffectIDs(db.SkillPRMagnus), effectBottomMagnus)
	expectEffectIDs(t, "PR_SANCTUARY imported ground", skillGroundEffectIDs(db.SkillPRSanctuary), effectBottomSanc)
	expectEffectIDs(t, "PR_SLOWPOISON imported", skillEffectIDs(db.SkillPRSlowpoison), effectSlowPoison)
	expectEffectIDs(t, "PR_STRECOVERY imported", skillEffectIDs(db.SkillPRStrecovery), effectRecovery)
	expectEffectIDs(t, "PR_REDEMPTIO imported empty", skillEffectIDs(db.SkillPRRedemptio))
	expectEffectIDs(t, "WZ_FIREPILLAR imported", skillEffectIDs(db.SkillWZFirepillar), effectFirePillar)
	expectEffectIDs(t, "WZ_FIREPILLAR imported ground", skillGroundEffectIDs(db.SkillWZFirepillar), effectFirePillarOn)
	expectEffectIDs(t, "WZ_FIREPILLAR imported hit", skillHitEffectIDs(db.SkillWZFirepillar), effectFirePillarBomb)
	expectEffectIDs(t, "WZ_SIGHTRASHER imported", skillEffectIDs(db.SkillWZSightrasher), effectSightTrasher)
	expectEffectIDs(t, "WZ_SIGHTRASHER imported hit", skillHitEffectIDs(db.SkillWZSightrasher), effectFireHit)
	expectEffectIDs(t, "WZ_FIREIVY unused", skillEffectIDs(db.SkillWZFireivy))
	expectEffectIDs(t, "WZ_METEOR imported", skillEffectIDs(db.SkillWZMeteor), effectMeteorStorm)
	expectEffectIDs(t, "WZ_METEOR imported hit", skillHitEffectIDs(db.SkillWZMeteor), effectFireHit)
	expectEffectIDs(t, "WZ_JUPITEL imported", skillEffectIDs(db.SkillWZJupitel), effectJupitelThunder)
	expectEffectIDs(t, "WZ_JUPITEL imported before hit", skillBeforeHitEffectIDs(db.SkillWZJupitel), effectJupitelHit)
	expectEffectIDs(t, "WZ_WATERBALL imported self before hit", skillBeforeHitEffectSelfIDs(db.SkillWZWaterball), effectWaterBall)
	expectEffectIDs(t, "WZ_WATERBALL imported caster hit", skillHitEffectOnCasterIDs(db.SkillWZWaterball), effectWaterBall2)
	expectEffectIDs(t, "WZ_VERMILION imported", skillEffectIDs(db.SkillWZVermilion), effectLordVermilion)
	expectEffectIDs(t, "WZ_VERMILION imported hit", skillHitEffectIDs(db.SkillWZVermilion), effectWindHit)
	expectEffectIDs(t, "WZ_ICEWALL imported ground", skillGroundEffectIDs(db.SkillWZIcewall), effectIceWall)
	expectEffectIDs(t, "WZ_FROSTNOVA imported caster", skillEffectOnCasterIDs(db.SkillWZFrostnova), effectFrostDiverHit)
	expectEffectIDs(t, "WZ_FROSTNOVA imported hit", skillHitEffectIDs(db.SkillWZFrostnova), effectColdHit)
	expectEffectIDs(t, "WZ_EARTHSPIKE imported", skillEffectIDs(db.SkillWZEarthspike), effectEarthSpike)
	expectEffectIDs(t, "WZ_EARTHSPIKE imported hit", skillHitEffectIDs(db.SkillWZEarthspike), effectEarthHit)
	expectEffectIDs(t, "WZ_HEAVENDRIVE imported", skillEffectIDs(db.SkillWZHeavendrive), effectHeavenDrive)
	expectEffectIDs(t, "WZ_HEAVENDRIVE imported hit", skillHitEffectIDs(db.SkillWZHeavendrive), effectEarthHit)
	expectEffectIDs(t, "WZ_QUAGMIRE imported ground", skillGroundEffectIDs(db.SkillWZQuagmire), effectQuagmire)
	expectEffectIDs(t, "WZ_STORMGUST imported", skillEffectIDs(db.SkillWZStormgust), effectStormGust)
	expectEffectIDs(t, "WZ_STORMGUST imported hit", skillHitEffectIDs(db.SkillWZStormgust), effectColdHit)
	expectEffectIDs(t, "WZ_ESTIMATION imported empty", skillEffectIDs(db.SkillWZEstimation))
	expectEffectIDs(t, "WZ_SIGHTBLASTER imported", skillEffectIDs(db.SkillWZSightblaster), 601)
	expectEffectIDs(t, "MC_IDENTIFY imported empty", skillEffectIDs(db.SkillMCIdentify))
	expectEffectIDs(t, "MC_VENDING imported empty", skillEffectIDs(db.SkillMCVending))
	expectEffectIDs(t, "MC_CHANGECART imported empty", skillEffectIDs(db.SkillMCChangecart))
	expectEffectIDs(t, "MC_CARTDECORATE imported empty", skillEffectIDs(db.SkillMCCartdecorate))
	expectEffectIDs(t, "BS_REPAIRWEAPON imported", skillEffectIDs(db.SkillBSRepairweapon), effectRepairWeapon)
	expectEffectIDs(t, "BS_HAMMERFALL imported", skillEffectIDs(db.SkillBSHammerfall), effectCrashEarth)
	expectEffectIDs(t, "BS_ADRENALINE imported", skillEffectIDs(db.SkillBSAdrenaline), effectHasteUp)
	expectEffectIDs(t, "BS_ADRENALINE imported begin", skillBeginEffectIDs(db.SkillBSAdrenaline), effectAdrenalineCast)
	expectEffectIDs(t, "BS_WEAPONPERFECT imported", skillEffectIDs(db.SkillBSWeaponperfect), effectWeaponPerfect)
	expectEffectIDs(t, "BS_OVERTHRUST imported", skillEffectIDs(db.SkillBSOverthrust), effectOverthrust)
	expectEffectIDs(t, "BS_MAXIMIZE imported", skillEffectIDs(db.SkillBSMaximize), effectMaximizePower)
	expectEffectIDs(t, "BS_MAXIMIZE imported begin", skillBeginEffectIDs(db.SkillBSMaximize), effectMaximizeSounds)
	expectEffectIDs(t, "BS_ADRENALINE2 imported", skillEffectIDs(db.SkillBSAdrenaline2), effectHasteUp)
	expectEffectIDs(t, "BS_ADRENALINE2 imported begin", skillBeginEffectIDs(db.SkillBSAdrenaline2), effectAdrenalineCast)
	expectEffectIDs(t, "BS_GREED imported", skillEffectIDs(db.SkillBSGreed), effectGreedSound)
	expectEffectIDs(t, "KN_PIERCE imported caster", skillEffectOnCasterIDs(db.SkillKNPierce), effectPierceSelf)
	expectEffectIDs(t, "KN_PIERCE imported hit", skillHitEffectIDs(db.SkillKNPierce), effectEarthHit)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported", skillEffectIDs(db.SkillKNBrandishspear), effectBrandishSpear)
	expectEffectIDs(t, "KN_BRANDISHSPEAR imported caster", skillEffectOnCasterIDs(db.SkillKNBrandishspear), effectBrandishSpear2)
	expectEffectIDs(t, "KN_SPEARSTAB imported caster", skillEffectOnCasterIDs(db.SkillKNSpearstab), effectSpearStabSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported caster", skillEffectOnCasterIDs(db.SkillKNSpearboomerang), effectSpearBmrSelf)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillKNSpearboomerang), effectSpearProjectile)
	expectEffectIDs(t, "KN_SPEARBOOMERANG imported hit", skillHitEffectIDs(db.SkillKNSpearboomerang), effectSpearBoomerang)
	expectEffectIDs(t, "KN_TWOHANDQUICKEN imported", skillEffectIDs(db.SkillKNTwohandquicken), effectTwoHandQuicken)
	expectEffectIDs(t, "KN_ONEHAND imported", skillEffectIDs(db.SkillKNOnehand), effectTwoHandQuicken)
	expectEffectIDs(t, "KN_CHARGEATK imported begin", skillBeginEffectIDs(db.SkillKNChargeatk), effectWhitePulse)
	expectEffectIDs(t, "KN_CHARGEATK imported hit", skillHitEffectIDs(db.SkillKNChargeatk), effectEnemyHitNormal1)
	expectEffectIDs(t, "KN_BOWLINGBASH imported caster", skillEffectOnCasterIDs(db.SkillKNBowlingbash), effectBowlingSelf)
	expectEffectIDs(t, "HT_SKIDTRAP imported", skillEffectIDs(db.SkillHTSkidtrap), effectSkidTrap)
	expectEffectIDs(t, "HT_LANDMINE imported empty", skillEffectIDs(db.SkillHTLandmine))
	expectEffectIDs(t, "HT_ANKLESNARE imported ground", skillGroundEffectIDs(db.SkillHTAnklesnare), effectAnkleSnareGround)
	expectEffectIDs(t, "HT_SHOCKWAVE imported", skillEffectIDs(db.SkillHTShockwave), effectShockwave)
	expectEffectIDs(t, "HT_SHOCKWAVE imported hit", skillHitEffectIDs(db.SkillHTShockwave), effectShockwaveHit)
	expectEffectIDs(t, "HT_SANDMAN imported hit", skillHitEffectIDs(db.SkillHTSandman), effectSandman)
	expectEffectIDs(t, "HT_FLASHER imported hit", skillHitEffectIDs(db.SkillHTFlasher), effectFlasher)
	expectEffectIDs(t, "HT_FREEZINGTRAP imported hit", skillHitEffectIDs(db.SkillHTFreezingtrap), effectFreezingTrap)
	expectEffectIDs(t, "HT_BLASTMINE imported hit", skillHitEffectIDs(db.SkillHTBlastmine), effectBlastMineBomb)
	expectEffectIDs(t, "HT_CLAYMORE imported hit", skillHitEffectIDs(db.SkillHTClaymoretrap), effectClaymore)
	expectEffectIDs(t, "HT_REMOVETRAP imported", skillEffectIDs(db.SkillHTRemovetrap), effectRemoveTrap)
	expectEffectIDs(t, "HT_BLITZBEAT imported", skillEffectIDs(db.SkillHTBlitzbeat), effectBlitzBeat)
	expectEffectIDs(t, "HT_DETECTING imported", skillEffectIDs(db.SkillHTDetecting), effectDetecting)
	expectEffectIDs(t, "HT_SPRINGTRAP imported", skillEffectIDs(db.SkillHTSpringtrap), effectSpringTrap)
	expectEffectIDs(t, "HT_TALKIEBOX imported empty", skillEffectIDs(db.SkillHTTalkiebox))
	expectEffectIDs(t, "HT_POWER imported empty", skillEffectIDs(db.SkillHTPower))
	expectEffectIDs(t, "AS_CLOAKING imported", skillEffectIDs(db.SkillASCloaking), effectCloaking)
	expectEffectIDs(t, "AS_SONICBLOW imported", skillEffectIDs(db.SkillASSonicblow), effectSonicBlow2)
	expectEffectIDs(t, "AS_SONICBLOW imported caster", skillEffectOnCasterIDs(db.SkillASSonicblow), effectSonicBlow)
	expectEffectIDs(t, "AS_SONICBLOW imported hit", skillHitEffectIDs(db.SkillASSonicblow), effectSonicBlowHit)
	expectEffectIDs(t, "AS_GRIMTOOTH imported", skillEffectIDs(db.SkillASGrimtooth), effectGrimtooth)
	expectEffectIDs(t, "AS_GRIMTOOTH imported hit", skillHitEffectIDs(db.SkillASGrimtooth), effectGrimtoothAtk)
	expectEffectIDs(t, "AS_POISONREACT imported", skillEffectIDs(db.SkillASPoisonreact), effectPoisonReact)
	expectEffectIDs(t, "AS_POISONREACT imported hit", skillHitEffectIDs(db.SkillASPoisonreact), effectPoisonReact2)
	expectEffectIDs(t, "AS_VENOMDUST imported", skillEffectIDs(db.SkillASVenomdust), effectVenomDust)
	expectEffectIDs(t, "AS_VENOMDUST imported ground", skillGroundEffectIDs(db.SkillASVenomdust), effectVenomDust2)
	expectEffectIDs(t, "AS_SPLASHER imported", skillEffectIDs(db.SkillASSplasher), effectVenomSplasher)
	expectEffectIDs(t, "NPC_DARKCROSS imported", skillEffectIDs(db.SkillNPCDarkcross), effectDarkGrandCross)
	expectEffectIDs(t, "NPC_DARKSTRIKE imported", skillEffectIDs(db.SkillNPCDarkstrike), effectDarkSoulStrike)
	expectEffectIDs(t, "NPC_STOP imported", skillEffectIDs(db.SkillNPCStop), effectNPCStop)
	expectEffectIDs(t, "NPC_POWERUP imported", skillEffectIDs(db.SkillNPCPowerup), effectNPCPowerUp)
	expectEffectIDs(t, "NPC_DARKBREATH imported", skillEffectIDs(db.SkillNPCDarkbreath), effectDarkBreath)
	expectEffectIDs(t, "NPC_DEFENDER imported", skillEffectIDs(db.SkillNPCDefender), effectDefender)
	expectEffectIDs(t, "NPC_KEEPING imported", skillEffectIDs(db.SkillNPCKeeping), effectKeeping)
	expectEffectIDs(t, "NPC_BLOODDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCBlooddrain), effectBloodDrain)
	expectEffectIDs(t, "NPC_ENERGYDRAIN imported caster", skillEffectOnCasterIDs(db.SkillNPCEnergydrain), effectEnergyDrain)
	expectEffectIDs(t, "NPC_EARTHQUAKE imported caster", skillEffectOnCasterIDs(db.SkillNPCEarthquake), effectNPCEarthquake)
	expectEffectIDs(t, "NPC_DRAGONFEAR imported", skillEffectIDs(db.SkillNPCDragonfear), effectDragonFear)
	expectEffectIDs(t, "NPC_WIDEBLEEDING imported caster", skillEffectOnCasterIDs(db.SkillNPCWidebleeding), effectWideBleeding)
	expectEffectIDs(t, "NPC_EVILLAND imported ground", skillGroundEffectIDs(db.SkillNPCEvilland), effectBottomEvilLand)
	expectEffectIDs(t, "NPC_CRITICALWOUND imported hit", skillHitEffectIDs(db.SkillNPCCriticalwound), effectCriticalWound)
	expectEffectIDs(t, "RG_STEALCOIN imported success", skillSuccessEffectIDs(db.SkillRGStealcoin), effectStealCoin, effectRogueCoin)
	expectEffectIDs(t, "RG_BACKSTAP imported hit", skillHitEffectIDs(db.SkillRGBackstap), effectBackStab)
	expectEffectIDs(t, "RG_RAID imported caster", skillEffectOnCasterIDs(db.SkillRGRaid), effectTeiHit3)
	expectEffectIDs(t, "RG_STRIPWEAPON imported success", skillSuccessEffectIDs(db.SkillRGStripweapon), effectStripWeapon)
	expectEffectIDs(t, "RG_STRIPSHIELD imported success", skillSuccessEffectIDs(db.SkillRGStripshield), effectStripShield)
	expectEffectIDs(t, "RG_STRIPARMOR imported success", skillSuccessEffectIDs(db.SkillRGStriparmor), effectStripArmor)
	expectEffectIDs(t, "RG_STRIPHELM imported success", skillSuccessEffectIDs(db.SkillRGStriphelm), effectStripHelm)
	expectEffectIDs(t, "RG_INTIMIDATE imported", skillEffectIDs(db.SkillRGIntimidate), effectIntimidate)
	expectEffectIDs(t, "RG_GRAFFITI imported empty", skillEffectIDs(db.SkillRGGraffiti))
	expectEffectIDs(t, "RG_FLAGGRAFFITI imported empty", skillEffectIDs(db.SkillRGFlaggraffiti))
	expectEffectIDs(t, "RG_CLEANER imported empty", skillEffectIDs(db.SkillRGCleaner))
	expectEffectIDs(t, "RG_CLOSECONFINE imported", skillEffectIDs(db.SkillRGCloseconfine), 602)
	expectEffectIDs(t, "RG_CLOSECONFINE imported ground", skillGroundEffectIDs(db.SkillRGCloseconfine), effectNPCStop2)
	expectEffectIDs(t, "AM_PHARMACY imported empty", skillEffectIDs(db.SkillAMPharmacy))
	expectEffectIDs(t, "AM_DEMONSTRATION imported ground", skillGroundEffectIDs(db.SkillAMDemonstration), effectDemonstration)
	expectEffectIDs(t, "AM_ACIDTERROR imported before hit", skillBeforeHitEffectIDs(db.SkillAMAcidterror), effectThrowItem)
	expectEffectIDs(t, "AM_POTIONPITCHER imported", skillEffectIDs(db.SkillAMPotionpitcher), 299)
	expectEffectIDs(t, "AM_CANNIBALIZE imported empty", skillEffectIDs(db.SkillAMCannibalize))
	expectEffectIDs(t, "AM_SPHEREMINE imported empty", skillEffectIDs(db.SkillAMSpheremine))
	expectEffectIDs(t, "AM_CP_WEAPON imported", skillEffectIDs(db.SkillAMCpWeapon), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_SHIELD imported", skillEffectIDs(db.SkillAMCpShield), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_ARMOR imported", skillEffectIDs(db.SkillAMCpArmor), effectChemicalProt)
	expectEffectIDs(t, "AM_CP_HELM imported", skillEffectIDs(db.SkillAMCpHelm), effectChemicalProt)
	expectEffectIDs(t, "AM_CALLHOMUN imported empty", skillEffectIDs(db.SkillAMCallhomun))
	expectEffectIDs(t, "AM_REST imported empty", skillEffectIDs(db.SkillAMRest))
	expectEffectIDs(t, "AM_RESURRECTHOMUN imported empty", skillEffectIDs(db.SkillAMResurrecthomun))
	expectEffectIDs(t, "CR_AUTOGUARD imported", skillEffectIDs(db.SkillCRAutoguard), effectGuard)
	expectEffectIDs(t, "CR_SHIELDCHARGE imported", skillEffectIDs(db.SkillCRShieldcharge), effectShieldCharge)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported", skillEffectIDs(db.SkillCRShieldboomerang), effectShieldBoomer)
	expectEffectIDs(t, "CR_SHIELDBOOMERANG imported before hit", skillBeforeHitEffectIDs(db.SkillCRShieldboomerang), effectShieldProjectile)
	expectEffectIDs(t, "CR_REFLECTSHIELD imported", skillEffectIDs(db.SkillCRReflectshield), effectReflectShield)
	expectEffectIDs(t, "CR_HOLYCROSS imported", skillEffectIDs(db.SkillCRHolycross), effectHolyCross)
	expectEffectIDs(t, "CR_GRANDCROSS imported", skillEffectIDs(db.SkillCRGrandcross), effectGrandCross)
	expectEffectIDs(t, "CR_DEVOTION imported", skillEffectIDs(db.SkillCRDevotion), effectDevotion)
	expectEffectIDs(t, "CR_PROVIDENCE imported", skillEffectIDs(db.SkillCRProvidence), effectProvidence)
	expectEffectIDs(t, "CR_DEFENDER imported", skillEffectIDs(db.SkillCRDefender), effectCrusaderDef)
	expectEffectIDs(t, "CR_SPEARQUICKEN imported", skillEffectIDs(db.SkillCRSpearquicken), effectSpearQuicken)
	expectEffectIDs(t, "MO_CALLSPIRITS imported empty", skillEffectIDs(db.SkillMOCallspirits))
	expectEffectIDs(t, "MO_ABSORBSPIRITS imported self success", skillSuccessEffectSelfIDs(db.SkillMOAbsorbspirits), effectAbsorbSpirits)
	expectEffectIDs(t, "MO_BODYRELOCATION imported empty", skillEffectIDs(db.SkillMOBodyrelocation))
	expectEffectIDs(t, "MO_INVESTIGATE imported", skillEffectIDs(db.SkillMOInvestigate), effectChimto)
	expectEffectIDs(t, "MO_FINGEROFFENSIVE imported", skillEffectIDs(db.SkillMOFingeroffensive), effectTanji)
	expectEffectIDs(t, "MO_STEELBODY imported", skillEffectIDs(db.SkillMOSteelbody), effectSteelBody, effectQuake)
	expectEffectIDs(t, "MO_BLADESTOP imported empty", skillEffectIDs(db.SkillMOBladestop))
	expectEffectIDs(t, "MO_EXPLOSIONSPIRITS imported caster", skillEffectOnCasterIDs(db.SkillMOExplosionspirits), effectGumgang2, effectQuake)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported", skillEffectIDs(db.SkillMOExtremityfist), effectBeginAsura, effectQuake)
	expectEffectIDs(t, "MO_EXTREMITYFIST imported hit", skillHitEffectIDs(db.SkillMOExtremityfist), effectTeiHit1X)
	expectEffectIDs(t, "MO_TRIPLEATTACK imported", skillEffectIDs(db.SkillMOTripleattack), effectTripleAttack)
	expectEffectIDs(t, "MO_CHAINCOMBO imported", skillEffectIDs(db.SkillMOChaincombo), effectTeiHit1, effectChainCombo)
	expectEffectIDs(t, "MO_CHAINCOMBO imported caster", skillEffectOnCasterIDs(db.SkillMOChaincombo), effectGumgang3)
	expectEffectIDs(t, "MO_COMBOFINISH imported", skillEffectIDs(db.SkillMOCombofinish), 330, effectQuake)
	expectEffectIDs(t, "SA_CASTCANCEL imported empty", skillEffectIDs(db.SkillSACastcancel))
	expectEffectIDs(t, "SA_MAGICROD imported success", skillSuccessEffectIDs(db.SkillSAMagicrod), effectMagicRod)
	expectEffectIDs(t, "SA_SPELLBREAKER imported success", skillSuccessEffectIDs(db.SkillSASpellbreaker), effectSpellBreaker)
	expectEffectIDs(t, "SA_AUTOSPELL imported empty", skillEffectIDs(db.SkillSAAutospell))
	expectEffectIDs(t, "SA_FLAMELAUNCHER imported success", skillSuccessEffectIDs(db.SkillSAFlamelauncher), effectFlameLauncher)
	expectEffectIDs(t, "SA_FROSTWEAPON imported success", skillSuccessEffectIDs(db.SkillSAFrostweapon), effectFrostWeapon)
	expectEffectIDs(t, "SA_LIGHTNINGLOADER imported success", skillSuccessEffectIDs(db.SkillSALightningloader), effectLightningLoad)
	expectEffectIDs(t, "SA_SEISMICWEAPON imported success", skillSuccessEffectIDs(db.SkillSASeismicweapon), effectSeismicWeapon)
	expectEffectIDs(t, "SA_DISPELL imported success", skillSuccessEffectIDs(db.SkillSADispell), effectDispell)
	expectEffectIDs(t, "SA_VOLCANO imported caster", skillEffectOnCasterIDs(db.SkillSAVolcano), 225)
	expectEffectIDs(t, "SA_VOLCANO imported ground", skillGroundEffectIDs(db.SkillSAVolcano), effectBottomVolcano)
	expectEffectIDs(t, "SA_DELUGE imported caster", skillEffectOnCasterIDs(db.SkillSADeluge), 236)
	expectEffectIDs(t, "SA_DELUGE imported ground", skillGroundEffectIDs(db.SkillSADeluge), effectBottomDeluge)
	expectEffectIDs(t, "SA_VIOLENTGALE imported caster", skillEffectOnCasterIDs(db.SkillSAViolentgale), 237)
	expectEffectIDs(t, "SA_VIOLENTGALE imported ground", skillGroundEffectIDs(db.SkillSAViolentgale), effectBottomViolent)
	expectEffectIDs(t, "SA_LANDPROTECTOR imported caster", skillEffectOnCasterIDs(db.SkillSALandprotector), 238)
	expectEffectIDs(t, "SA_LANDPROTECTOR imported ground", skillGroundEffectIDs(db.SkillSALandprotector), effectBottomLand)
	expectEffectIDs(t, "SA_ABRACADABRA imported empty", skillEffectIDs(db.SkillSAAbracadabra))
	for _, skillID := range []uint16{
		db.SkillSAMonocell,
		db.SkillSAClasschange,
		db.SkillSASummonmonster,
		db.SkillSAReverseorcish,
		db.SkillSADeath,
		db.SkillSAFortune,
		db.SkillSATamingmonster,
		db.SkillSAQuestion,
		db.SkillSAGravity,
		db.SkillSALevelup,
		db.SkillSAInstantdeath,
		db.SkillSAFullrecovery,
		db.SkillSAComa,
	} {
		expectEffectIDs(t, "SA_ABRACADABRA result imported empty", skillEffectIDs(skillID))
	}
	expectEffectIDs(t, "BD_ADAPTATION imported empty", skillEffectIDs(db.SkillBDAdaptation))
	expectEffectIDs(t, "BD_ENCORE imported empty", skillEffectIDs(db.SkillBDEncore))
	expectEffectIDs(t, "BD_LULLABY imported", skillEffectIDs(db.SkillBDLullaby), effectBottomLullaby)
	expectEffectIDs(t, "BD_LULLABY imported ground", skillGroundEffectIDs(db.SkillBDLullaby), effectBottomLullabyGround)
	expectEffectIDs(t, "BD_RICHMANKIM imported", skillEffectIDs(db.SkillBDRichmankim), effectBottomRichKim)
	expectEffectIDs(t, "BD_RICHMANKIM imported ground", skillGroundEffectIDs(db.SkillBDRichmankim), effectBottomRichKimGround)
	expectEffectIDs(t, "BD_ETERNALCHAOS imported", skillEffectIDs(db.SkillBDEternalchaos), effectBottomChaos)
	expectEffectIDs(t, "BD_ETERNALCHAOS imported ground", skillGroundEffectIDs(db.SkillBDEternalchaos), effectBottomChaosGround)
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported", skillEffectIDs(db.SkillBDDrumbattlefield), effectBottomDrum)
	expectEffectIDs(t, "BD_DRUMBATTLEFIELD imported ground", skillGroundEffectIDs(db.SkillBDDrumbattlefield), effectBottomDrumGround)
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported", skillEffectIDs(db.SkillBDRingnibelungen), effectBottomNibelung)
	expectEffectIDs(t, "BD_RINGNIBELUNGEN imported ground", skillGroundEffectIDs(db.SkillBDRingnibelungen), effectBottomNibelungGround)
	expectEffectIDs(t, "BD_ROKISWEIL imported", skillEffectIDs(db.SkillBDRokisweil), effectBottomRoki)
	expectEffectIDs(t, "BD_ROKISWEIL imported ground", skillGroundEffectIDs(db.SkillBDRokisweil), effectBottomRokiGround)
	expectEffectIDs(t, "BD_INTOABYSS imported", skillEffectIDs(db.SkillBDIntoabyss), effectBottomAbyss)
	expectEffectIDs(t, "BD_INTOABYSS imported ground", skillGroundEffectIDs(db.SkillBDIntoabyss), effectBottomAbyssGround)
	expectEffectIDs(t, "BD_SIEGFRIED imported", skillEffectIDs(db.SkillBDSiegfried), effectBottomSieg)
	expectEffectIDs(t, "BD_SIEGFRIED imported ground", skillGroundEffectIDs(db.SkillBDSiegfried), effectBottomSiegGround)
	expectEffectIDs(t, "BA_MUSICALLESSON imported empty", skillEffectIDs(db.SkillBaMusicallesson))
	expectEffectIDs(t, "BA_MUSICALSTRIKE imported before hit", skillBeforeHitEffectIDs(db.SkillBaMusicalstrike), effectArrowShot)
	if !skillHidesCastAura(db.SkillBaMusicalstrike) {
		t.Fatal("BA_MUSICALSTRIKE should hide cast aura like robr")
	}
	expectEffectIDs(t, "BA_DISSONANCE imported ground", skillGroundEffectIDs(db.SkillBaDissonance), effectBottomDissonanceGround)
	expectEffectIDs(t, "BA_FROSTJOKE imported begin", skillBeginEffectIDs(db.SkillBaFrostjoke), effectTalkFrostJoke)
	expectEffectIDs(t, "BA_WHISTLE imported", skillEffectIDs(db.SkillBaWhistle), effectBottomWhistle)
	expectEffectIDs(t, "BA_WHISTLE imported ground", skillGroundEffectIDs(db.SkillBaWhistle), effectBottomWhistleGround)
	expectEffectIDs(t, "BA_ASSASSINCROSS imported", skillEffectIDs(db.SkillBaAssassincross), effectBottomSinX)
	expectEffectIDs(t, "BA_ASSASSINCROSS imported ground", skillGroundEffectIDs(db.SkillBaAssassincross), effectBottomSinXGround)
	expectEffectIDs(t, "BA_POEMBRAGI imported", skillEffectIDs(db.SkillBaPoembragi), effectBottomBragi)
	expectEffectIDs(t, "BA_POEMBRAGI imported ground", skillGroundEffectIDs(db.SkillBaPoembragi), effectBottomBragiGround)
	expectEffectIDs(t, "BA_APPLEIDUN imported", skillEffectIDs(db.SkillBaAppleidun), effectBottomApple)
	expectEffectIDs(t, "BA_APPLEIDUN imported ground", skillGroundEffectIDs(db.SkillBaAppleidun), effectBottomAppleGround)
	expectEffectIDs(t, "BA_PANGVOICE imported success", skillSuccessEffectIDs(db.SkillBaPangvoice), effectFVoice)
	expectEffectIDs(t, "DC_DANCINGLESSON imported empty", skillEffectIDs(db.SkillDCDancinglesson))
	expectEffectIDs(t, "DC_THROWARROW imported before hit", skillBeforeHitEffectIDs(db.SkillDCThrowarrow), effectArrowShot)
	if !skillHidesCastAura(db.SkillDCThrowarrow) {
		t.Fatal("DC_THROWARROW should hide cast aura like robr")
	}
	expectEffectIDs(t, "DC_UGLYDANCE imported ground", skillGroundEffectIDs(db.SkillDCUglydance), effectBottomUglyDanceGround)
	expectEffectIDs(t, "DC_SCREAM imported begin", skillBeginEffectIDs(db.SkillDCScream), effectTalkScream)
	expectEffectIDs(t, "DC_HUMMING imported", skillEffectIDs(db.SkillDCHumming), effectBottomHumming)
	expectEffectIDs(t, "DC_HUMMING imported ground", skillGroundEffectIDs(db.SkillDCHumming), effectBottomHummingGround)
	expectEffectIDs(t, "DC_DONTFORGETME imported", skillEffectIDs(db.SkillDCDontforgetme), effectBottomForget)
	expectEffectIDs(t, "DC_DONTFORGETME imported ground", skillGroundEffectIDs(db.SkillDCDontforgetme), effectBottomForgetGround)
	expectEffectIDs(t, "DC_FORTUNEKISS imported", skillEffectIDs(db.SkillDCFortunekiss), effectBottomFortune)
	expectEffectIDs(t, "DC_FORTUNEKISS imported ground", skillGroundEffectIDs(db.SkillDCFortunekiss), effectBottomFortuneGround)
	expectEffectIDs(t, "DC_SERVICEFORYOU imported", skillEffectIDs(db.SkillDCServiceforyou), effectBottomService)
	expectEffectIDs(t, "DC_SERVICEFORYOU imported ground", skillGroundEffectIDs(db.SkillDCServiceforyou), effectBottomServiceGround)
	expectEffectIDs(t, "DC_WINKCHARM imported success", skillSuccessEffectIDs(db.SkillDCWinkcharm), effectWink)
	expectEffectIDs(t, "SL_KAIZEL imported", skillEffectIDs(db.SkillSLKaizel), effectKaizel)
	expectEffectIDs(t, "SL_STUN imported", skillEffectIDs(db.SkillSLStun), effectStin3)
	expectEffectIDs(t, "SL_SMA imported", skillEffectIDs(db.SkillSLSma), effectStin2)
	expectEffectIDs(t, "SL_SWOO imported", skillEffectIDs(db.SkillSLSwoo), effectM07)
	expectEffectIDs(t, "SL_SKA imported", skillEffectIDs(db.SkillSLSka), effectSteelBody, effectGumgang2, effectQuake)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported", skillEffectIDs(db.SkillAMBerserkpitcher), effectItemFast3)
	expectEffectIDs(t, "AM_BERSERKPITCHER imported before hit", skillBeforeHitEffectIDs(db.SkillAMBerserkpitcher), 541)
	expectEffectIDs(t, "AM_TWILIGHT1 imported", skillEffectIDs(db.SkillAMTwilight1), 497)
	expectEffectIDs(t, "AM_TWILIGHT2 imported", skillEffectIDs(db.SkillAMTwilight2), 498)
	expectEffectIDs(t, "AM_TWILIGHT3 imported", skillEffectIDs(db.SkillAMTwilight3), 499)
	expectEffectIDs(t, "CR_ALCHEMY imported empty", skillEffectIDs(db.SkillCRAlchemy))
	expectEffectIDs(t, "CR_SYNTHESISPOTION imported empty", skillEffectIDs(db.SkillCRSynthesispotion))
	expectEffectIDs(t, "CR_SLIMPITCHER imported empty", skillEffectIDs(db.SkillCRSlimpitcher))
	expectEffectIDs(t, "CR_FULLPROTECTION imported", skillEffectIDs(db.SkillCRFullprotection), effectChemicalProt, 500)
	expectEffectIDs(t, "CR_CULTIVATION imported empty", skillEffectIDs(db.SkillCRCultivation))
	expectEffectIDs(t, "SA_CREATECON imported empty", skillEffectIDs(db.SkillSACreatecon))
	expectEffectIDs(t, "SA_ELEMENTWATER imported", skillEffectIDs(db.SkillSAElementwater), effectFrostWeapon)
	expectEffectIDs(t, "SA_ELEMENTGROUND imported", skillEffectIDs(db.SkillSAElementground), effectSeismicWeapon)
	expectEffectIDs(t, "SA_ELEMENTFIRE imported", skillEffectIDs(db.SkillSAElementfire), effectFlameLauncher)
	expectEffectIDs(t, "SA_ELEMENTWIND imported", skillEffectIDs(db.SkillSAElementwind), effectLightningLoad)
	expectEffectIDs(t, "MO_KITRANSLATION imported empty", skillEffectIDs(db.SkillMOKitranslation))
	expectEffectIDs(t, "MO_BALKYOUNG imported", skillEffectIDs(db.SkillMOBalkyoung), 514)
	expectEffectIDs(t, "LK_PARRYING imported", skillEffectIDs(db.SkillLKParrying), effectGuard)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported", skillEffectIDs(db.SkillLKSpiralpierce), effectMagnum2)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported begin", skillBeginEffectIDs(db.SkillLKSpiralpierce), effectSpiralBeforeCast)
	expectEffectIDs(t, "LK_SPIRALPIERCE imported hit", skillHitEffectIDs(db.SkillLKSpiralpierce), effectSpearHitSound)
	expectEffectIDs(t, "LK_AURABLADE imported", skillEffectIDs(db.SkillLKAurablade), effectAuraBlade)
	expectEffectIDs(t, "LK_AURABLADE imported begin", skillBeginEffectIDs(db.SkillLKAurablade), effectWhitePulse)
	expectEffectIDs(t, "LK_CONCENTRATION imported", skillEffectIDs(db.SkillLKConcentration), effectLKConcentration)
	expectEffectIDs(t, "LK_TENSIONRELAX imported empty", skillEffectIDs(db.SkillLKTensionrelax))
	expectEffectIDs(t, "LK_BERSERK imported", skillEffectIDs(db.SkillLKBerserk), effectRedBody, effectQuake)
	expectEffectIDs(t, "LK_FURY imported", skillEffectIDs(db.SkillLKFury), effectRedBody, effectQuake)
	expectEffectIDs(t, "HP_ASSUMPTIO imported", skillEffectIDs(db.SkillHPAssumptio), effectAssumptio2)
	expectEffectIDs(t, "HP_BASILICA imported ground", skillGroundEffectIDs(db.SkillHPBasilica), effectBottomBasilica)
	expectEffectIDs(t, "HP_MEDITATIO has no robr effect row", skillEffectIDs(db.SkillHPMeditatio))
	expectEffectIDs(t, "HP_MANARECHARGE has no robr effect row", skillEffectIDs(db.SkillHPManarecharge))
	expectEffectIDs(t, "HW_MAGICCRASHER imported", skillEffectIDs(db.SkillHWMagiccrasher), effectMagicCrasher)
	expectEffectIDs(t, "HW_MAGICPOWER imported", skillEffectIDs(db.SkillHWMagicpower), effectMagicPower)
	expectEffectIDs(t, "HW_MAGICPOWER imported begin", skillBeginEffectIDs(db.SkillHWMagicpower), effectBashBegin)
	if !skillHidesCastAura(db.SkillHWMagicpower) {
		t.Fatal("HW_MAGICPOWER should hide cast aura like robr")
	}
	expectEffectIDs(t, "HW_SOULDRAIN imported caster", skillEffectOnCasterIDs(db.SkillHWSouldrain), effectEnergyDrain)
	expectEffectIDs(t, "HW_NAPALMVULCAN imported", skillEffectIDs(db.SkillHWNapalmvulcan), 401)
	expectEffectIDs(t, "HW_GANBANTEIN imported", skillEffectIDs(db.SkillHWGanbantein), 223)
	expectEffectIDs(t, "HW_GANBANTEIN imported ground", skillGroundEffectIDs(db.SkillHWGanbantein), 224)
	expectEffectIDs(t, "HW_GRAVITATION imported ground", skillGroundEffectIDs(db.SkillHWGravitation), effectGravitation)
	expectEffectIDs(t, "PA_PRESSURE imported before hit", skillBeforeHitEffectIDs(db.SkillPaPressure), effectPressure)
	expectEffectIDs(t, "PA_SACRIFICE imported", skillEffectIDs(db.SkillPaSacrifice), effectBash3D)
	expectEffectIDs(t, "PA_GOSPEL imported", skillEffectIDs(db.SkillPaGospel), effectBottomGospel)
	expectEffectIDs(t, "PA_GOSPEL imported ground", skillGroundEffectIDs(db.SkillPaGospel), effectGospelGround)
	expectEffectIDs(t, "PA_SHIELDCHAIN imported before hit", skillBeforeHitEffectIDs(db.SkillPaShieldchain), effectShieldProjectile)
	expectEffectIDs(t, "CH_PALMSTRIKE imported hit", skillHitEffectIDs(db.SkillChPalmstrike), effectHitLine2, effectQuake)
	expectEffectIDs(t, "CH_TIGERFIST imported", skillEffectIDs(db.SkillChTigerfist), effectBash3D2, effectQuake)
	expectEffectIDs(t, "CH_CHAINCRUSH imported", skillEffectIDs(db.SkillChChaincrush), effectChemical2Dash)
	expectEffectIDs(t, "CH_SOULCOLLECT imported begin", skillBeginEffectIDs(db.SkillChSoulcollect), effectPortal5, effectBeginSpell)
	expectEffectIDs(t, "PF_HPCONVERSION imported", skillEffectIDs(db.SkillPFHpconversion), effectEnergyDrain3)
	expectEffectIDs(t, "PF_HPCONVERSION imported caster", skillEffectOnCasterIDs(db.SkillPFHpconversion), effectEnergyDrain2)
	expectEffectIDs(t, "PF_HPCONVERSION imported self success", skillSuccessEffectSelfIDs(db.SkillPFHpconversion), effectTransBlueBody)
	expectEffectIDs(t, "PF_SOULCHANGE imported", skillEffectIDs(db.SkillPFSoulchange), effectLineLink2)
	expectEffectIDs(t, "PF_SOULCHANGE imported success", skillSuccessEffectIDs(db.SkillPFSoulchange), 385)
	expectEffectIDs(t, "PF_SOULBURN imported", skillEffectIDs(db.SkillPFSoulburn), effectSoulBurn)
	expectEffectIDs(t, "PF_MINDBREAKER imported success", skillSuccessEffectIDs(db.SkillPFMindbreaker), effectMagicCrasher2)
	expectEffectIDs(t, "PF_MEMORIZE imported", skillEffectIDs(db.SkillPFMemorize), 505)
	expectEffectIDs(t, "PF_FOGWALL imported ground", skillGroundEffectIDs(db.SkillPFFogwall), effectFogWallGround)
	expectEffectIDs(t, "PF_SPIDERWEB imported ground", skillGroundEffectIDs(db.SkillPFSpiderweb), effectBottomSpider)
	expectEffectIDs(t, "PF_DOUBLECASTING imported", skillEffectIDs(db.SkillPFDoublecasting), 521)
	expectEffectIDs(t, "ASC_BREAKER imported before hit", skillBeforeHitEffectIDs(db.SkillASCBreaker), effectSoulBreaker)
	expectEffectIDs(t, "ASC_METEORASSAULT imported caster", skillEffectOnCasterIDs(db.SkillASCMeteorassault), effectSoulBreaker2)
	if !skillHidesCastAura(db.SkillASCMeteorassault) {
		t.Fatal("ASC_METEORASSAULT should hide cast aura like robr")
	}
	expectEffectIDs(t, "ASC_CDP imported empty", skillEffectIDs(db.SkillASCCdp))
	expectEffectIDs(t, "SN_SIGHT imported", skillEffectIDs(db.SkillSNSight), effectTrueSight)
	expectEffectIDs(t, "SN_FALCONASSAULT imported", skillEffectIDs(db.SkillSNFalconassault), effectFalconAssault)
	expectEffectIDs(t, "HT_PHANTASMIC imported before hit", skillBeforeHitEffectIDs(db.SkillHTPhantasmic), effectArrowShot)
	expectEffectIDs(t, "HT_PHANTASMIC imported hit", skillHitEffectIDs(db.SkillHTPhantasmic), effectBashHit)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported begin", skillBeginEffectIDs(db.SkillSNSharpshooting), effectSharpShootingCast)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported before hit", skillBeforeHitEffectIDs(db.SkillSNSharpshooting), effectArrowShot)
	expectEffectIDs(t, "SN_SHARPSHOOTING imported hit", skillHitEffectIDs(db.SkillSNSharpshooting), effectTripleAttack2)
	expectEffectIDs(t, "SN_WINDWALK imported", skillEffectIDs(db.SkillSNWindwalk), effectPortal4)
	expectEffectIDs(t, "WS_MELTDOWN imported", skillEffectIDs(db.SkillWSMeltdown), effectMeltdown)
	expectEffectIDs(t, "WS_CREATECOIN imported empty", skillEffectIDs(db.SkillWSCreatecoin))
	expectEffectIDs(t, "WS_CREATENUGGET imported empty", skillEffectIDs(db.SkillWSCreatenugget))
	expectEffectIDs(t, "WS_CARTBOOST imported", skillEffectIDs(db.SkillWSCartboost), effectCartBoost)
	expectEffectIDs(t, "WS_SYSTEMCREATE imported empty", skillEffectIDs(db.SkillWSSystemcreate))
	expectEffectIDs(t, "WS_WEAPONREFINE imported empty", skillEffectIDs(db.SkillWSWeaponrefine))
	expectEffectIDs(t, "WS_CARTTERMINATION imported", skillEffectIDs(db.SkillWSCarttermination), 518)
	expectEffectIDs(t, "WS_OVERTHRUSTMAX imported", skillEffectIDs(db.SkillWSOverthrustmax), effectOverthrust)
	expectEffectIDs(t, "ASC_EDP imported", skillEffectIDs(db.SkillASCEdp), effectEDP)
	expectEffectIDs(t, "ST_CHASEWALK imported begin", skillBeginEffectIDs(db.SkillSTChasewalk), effectCastSpin)
	expectEffectIDs(t, "ST_REJECTSWORD imported", skillEffectIDs(db.SkillSTRejectsword), effectRejectSword)
	expectEffectIDs(t, "ST_PRESERVE imported", skillEffectIDs(db.SkillSTPreserve), effectPreserve)
	expectEffectIDs(t, "ST_PRESERVE imported begin", skillBeginEffectIDs(db.SkillSTPreserve), effectSharpShootingCast)
	expectEffectIDs(t, "ST_FULLSTRIP imported success", skillSuccessEffectIDs(db.SkillSTFullstrip), 495)
	expectEffectIDs(t, "CG_ARROWVULCAN imported", skillEffectIDs(db.SkillCGArrowvulcan), effectTripleAttack3)
	expectEffectIDs(t, "CG_ARROWVULCAN imported before hit", skillBeforeHitEffectIDs(db.SkillCGArrowvulcan), effectArrowShot)
	expectEffectIDs(t, "CG_MOONLIT imported", skillEffectIDs(db.SkillCGMoonlit), effectMoonlit)
	expectEffectIDs(t, "CG_MOONLIT imported ground", skillGroundEffectIDs(db.SkillCGMoonlit), effectMoonlit)
	expectEffectIDs(t, "CG_MARIONETTE imported", skillEffectIDs(db.SkillCGMarionette), 395)
	expectEffectIDs(t, "CG_MARIONETTE imported hit", skillHitEffectIDs(db.SkillCGMarionette), 396)
	expectEffectIDs(t, "CG_LONGINGFREEDOM imported", skillEffectIDs(db.SkillCGLongingfreedom), 500)
	expectEffectIDs(t, "CG_HERMODE imported music", skillEffectIDs(db.SkillCGHermode), effectHermodeMusic)
	expectEffectIDs(t, "CG_HERMODE imported ground", skillGroundEffectIDs(db.SkillCGHermode), effectBottomHermode)
	expectEffectIDs(t, "CG_TAROTCARD imported success", skillSuccessEffectIDs(db.SkillCGTarotcard), 500)
	expectEffectIDs(t, "CR_ACIDDEMONSTRATION imported", skillEffectIDs(db.SkillCRAciddemonstration), effectAcidDemon)
	expectEffectIDs(t, "SL_KAAHI imported", skillEffectIDs(db.SkillSLKaahi), effectHated)
	expectEffectIDs(t, "SL_STIN imported", skillEffectIDs(db.SkillSLStin), effectStin)
	expectEffectIDs(t, "WE_MALE imported empty", skillEffectIDs(db.SkillWEMale))
	expectEffectIDs(t, "WE_FEMALE imported empty", skillEffectIDs(db.SkillWEFemale))
	expectEffectIDs(t, "WE_CALLPARTNER imported empty", skillEffectIDs(db.SkillWECallpartner))
	expectEffectIDs(t, "WE_BABY imported", skillEffectIDs(db.SkillWEBaby), 408)
	expectEffectIDs(t, "WE_CALLPARENT imported empty", skillEffectIDs(db.SkillWECallparent))
	expectEffectIDs(t, "WE_CALLBABY imported empty", skillEffectIDs(db.SkillWECallbaby))
	expectEffectIDs(t, "LK_HEADCRUSH imported begin", skillBeginEffectIDs(db.SkillLKHeadcrush), effectBash3D3)
	expectEffectIDs(t, "LK_HEADCRUSH imported hit", skillHitEffectIDs(db.SkillLKHeadcrush), effectEnemyHitNormal1)
	expectEffectIDs(t, "LK_JOINTBEAT imported begin", skillBeginEffectIDs(db.SkillLKJointbeat), effectBash3D4)
	expectEffectIDs(t, "LK_JOINTBEAT imported hit", skillHitEffectIDs(db.SkillLKJointbeat), effectEnemyHitNormal1)
	expectEffectIDs(t, "TK_JUMPKICK imported hit", skillHitEffectIDs(db.SkillTKJumpkick), effectJumpKick)
	expectEffectIDs(t, "GS_INCREASING imported", skillEffectIDs(db.SkillGSIncreasing), effectNPCPowerUp)
	expectEffectIDs(t, "GS_TRIPLEACTION imported", skillEffectIDs(db.SkillGSTripleaction), effectTripleAction)
	expectEffectIDs(t, "GS_BULLSEYE imported", skillEffectIDs(db.SkillGSBullseye), effectBullseye)
	expectEffectIDs(t, "GS_MAGICALBULLET imported", skillEffectIDs(db.SkillGSMagicalbullet), effectMagicalBullet)
	expectEffectIDs(t, "GS_TRACKING imported", skillEffectIDs(db.SkillGSTracking), effectTrackCasting)
	expectEffectIDs(t, "GS_TRACKING imported hit", skillHitEffectIDs(db.SkillGSTracking), effectTracking)
	expectEffectIDs(t, "GS_DISARM imported", skillEffectIDs(db.SkillGSDisarm), effectRGCoin3)
	expectEffectIDs(t, "GS_RAPIDSHOWER imported", skillEffectIDs(db.SkillGSRapidshower), effectRapidShower)
	expectEffectIDs(t, "GS_DESPERADO imported", skillEffectIDs(db.SkillGSDesperado), effectDesperado)
	expectEffectIDs(t, "GS_DUST imported", skillEffectIDs(db.SkillGSDust), effectBash3D5)
	expectEffectIDs(t, "GS_FULLBUSTER imported", skillEffectIDs(db.SkillGSFullbuster), effectM02)
	expectEffectIDs(t, "GS_SPREADATTACK imported", skillEffectIDs(db.SkillGSSpreadattack), effectSpreadAttack)
	expectEffectIDs(t, "NJ_SYURIKEN imported before hit", skillBeforeHitEffectIDs(db.SkillNJSyuriken), effectThrowItem7)
	expectEffectIDs(t, "NJ_KUNAI imported before hit", skillBeforeHitEffectIDs(db.SkillNJKunai), effectThrowItem8)
	expectEffectIDs(t, "NJ_HUUMA imported before hit", skillBeforeHitEffectIDs(db.SkillNJHuuma), effectThrowItem9)
	expectEffectIDs(t, "NJ_ZENYNAGE imported before hit", skillBeforeHitEffectIDs(db.SkillNJZenynage), effectThrowItem10)
	expectEffectIDs(t, "NJ_TATAMIGAESHI imported ground", skillGroundEffectIDs(db.SkillNJTatamigaeshi), effectTatami)
	expectEffectIDs(t, "NJ_KASUMIKIRI imported", skillEffectIDs(db.SkillNJKasumikiri), effectKasumikiri)
	expectEffectIDs(t, "NJ_KIRIKAGE imported", skillEffectIDs(db.SkillNJKirikage), effectKirikage)
	expectEffectIDs(t, "NJ_KOUENKA imported", skillEffectIDs(db.SkillNJKouenka), effectKouenka)
	expectEffectIDs(t, "NJ_KAENSIN imported ground", skillGroundEffectIDs(db.SkillNJKaensin), effectKaen)
	expectEffectIDs(t, "NJ_BAKUENRYU imported", skillEffectIDs(db.SkillNJBakuenryu), effectBaku)
	expectEffectIDs(t, "NJ_HYOUSENSOU imported", skillEffectIDs(db.SkillNJHyousensou), effectHyousensou)
	expectEffectIDs(t, "NJ_HYOUSYOURAKU imported", skillEffectIDs(db.SkillNJHyousyouraku), effectHyousyouraku)
	expectEffectIDs(t, "NJ_HUUJIN imported", skillEffectIDs(db.SkillNJHuujin), effectStin4)
	expectEffectIDs(t, "NJ_RAIGEKISAI imported", skillEffectIDs(db.SkillNJRaigekisai), effectThunderStorm2)
	expectEffectIDs(t, "NJ_ISSEN imported", skillEffectIDs(db.SkillNJIssen), effectIssen)
	expectEffectIDs(t, "AS_VENOMKNIFE imported before hit", skillBeforeHitEffectIDs(db.SkillASVenomknife), effectThrowItem6)
	expectEffectIDs(t, "ALL_WEWISH imported", skillEffectIDs(db.SkillALLWewish), effectChristmasCarol)
	expectEffectIDs(t, "NPC_VENOMFOG imported", skillEffectIDs(db.SkillNPCVenomfog), effectVenomFog)
	expectEffectIDs(t, "RK_IGNITIONBREAK imported caster", skillEffectOnCasterIDs(db.SkillRKIgnitionbreak), effectIgnitionBreak)
	expectEffectIDs(t, "RK_DRAGONBREATH imported hit", skillHitEffectIDs(db.SkillRKDragonbreath), effectM05)
	expectEffectIDs(t, "RK_DRAGONHOWLING imported", skillEffectIDs(db.SkillRKDragonhowling), effectDragonHowling)
	expectEffectIDs(t, "RK_MILLENNIUMSHIELD imported", skillEffectIDs(db.SkillRKMillenniumshield), effectMillenniumShield)
	expectEffectIDs(t, "RK_ENCHANTBLADE imported", skillEffectIDs(db.SkillRKEnchantblade), effectBerserkPotion2)
	expectEffectIDs(t, "RK_SONICWAVE imported", skillEffectIDs(db.SkillRKSonicwave), effectHealN)
	expectEffectIDs(t, "WL_WHITEIMPRISON imported", skillEffectIDs(db.SkillWLWhiteimprison), effectBottomBasilica2)
	expectEffectIDs(t, "WL_FROSTMISTY imported", skillEffectIDs(db.SkillWLFrostmisty), effectFrostMisty)
	expectEffectIDs(t, "WL_MARSHOFABYSS imported", skillEffectIDs(db.SkillWLMarshofabyss), effectMarshOfAbyss)
	expectEffectIDs(t, "WL_RECOGNIZEDSPELL imported", skillEffectIDs(db.SkillWLRecognizedspell), effectRecognized)
	expectEffectIDs(t, "WL_STASIS imported", skillEffectIDs(db.SkillWLStasis), effectStasis)
	expectEffectIDs(t, "WL_CRIMSONROCK imported", skillEffectIDs(db.SkillWLCrimsonrock), effectCrimsonRock)
	expectEffectIDs(t, "WL_HELLINFERNO imported ground", skillGroundEffectIDs(db.SkillWLHellinferno), effectHellInferno)
	expectEffectIDs(t, "WL_CHAINLIGHTNING_ATK imported", skillEffectIDs(db.SkillWLChainlightningAtk), effectChainLightning)
	expectEffectIDs(t, "WL_EARTHSTRAIN imported ground", skillGroundEffectIDs(db.SkillWLEarthstrain), effectEarthWall)
	expectEffectIDs(t, "WL_TETRAVORTEX imported", skillEffectIDs(db.SkillWLTetravortex), effectTetra)
	expectEffectIDs(t, "WL_TETRAVORTEX imported begin", skillBeginEffectIDs(db.SkillWLTetravortex), effectTetraCasting)
	expectEffectIDs(t, "GC_ROLLINGCUTTER imported", skillEffectIDs(db.SkillGCRollingcutter), effectCastSpin2)
	expectEffectIDs(t, "AB_JUDEX imported", skillEffectIDs(db.SkillABJudex), effectFirePillarOn2)
	expectEffectIDs(t, "AB_JUDEX imported hit", skillHitEffectIDs(db.SkillABJudex), effectHolyLight)
	expectEffectIDs(t, "AB_ADORAMUS imported", skillEffectIDs(db.SkillABAdoramus), effectAdoramus)
	expectEffectIDs(t, "AB_EPICLESIS imported", skillEffectIDs(db.SkillABEpiclesis), effectGlassWall4)
	expectEffectIDs(t, "AB_EPICLESIS imported ground", skillGroundEffectIDs(db.SkillABEpiclesis), effectGlassWall3)
	expectEffectIDs(t, "RA_ARROWSTORM imported", skillEffectIDs(db.SkillRAArrowstorm), effectArrowStorm)
	expectEffectIDs(t, "RA_AIMEDBOLT imported", skillEffectIDs(db.SkillRAAimedbolt), effectAimedBolt)
	expectEffectIDs(t, "RA_AIMEDBOLT imported before hit", skillBeforeHitEffectIDs(db.SkillRAAimedbolt), effectArrowShot)
	expectEffectIDs(t, "RA_DETONATOR imported", skillEffectIDs(db.SkillRADetonator), effectConcentration2)
	expectEffectIDs(t, "NC_POWERSWING imported", skillEffectIDs(db.SkillNCPowerswing), effectCrashAxe)
	expectEffectIDs(t, "SR_EARTHSHAKER imported", skillEffectIDs(db.SkillSREarthshaker), effectElectric4)
	expectEffectIDs(t, "GC_DARKCROW imported", skillEffectIDs(db.SkillGCDarkcrow), effectGCDarkCrow)
	expectEffectIDs(t, "GN_ILLUSIONDOPING imported", skillEffectIDs(db.SkillGNIllusiondoping), effectGNIllusionDoping)
	expectEffectIDs(t, "RK_LUXANIMA imported", skillEffectIDs(db.SkillRKLuxanima), effectRKLuxAnima)
	expectEffectIDs(t, "NC_MAGMA_ERUPTION imported", skillEffectIDs(db.SkillNCMagmaEruption), effectNCMagmaEruption)
	expectEffectIDs(t, "SO_ELEMENTAL_SHIELD imported", skillEffectIDs(db.SkillSOElementalShield), effectSOElemShield)
	expectEffectIDs(t, "SR_FLASHCOMBO imported", skillEffectIDs(db.SkillSRFlashcombo), effectSRFlashCombo)
	expectEffectIDs(t, "AB_OFFERTORIUM imported", skillEffectIDs(db.SkillABOffertorium), effectABOffertorium)
	expectEffectIDs(t, "WL_TELEKINESIS_INTENSE imported", skillEffectIDs(db.SkillWLTelekinesisIntense), effectWLTelekinesis)
	expectEffectIDs(t, "ALL_FULL_THROTTLE imported", skillEffectIDs(db.SkillALLFullThrottle), effectAllFullThrottle)
	expectEffectIDs(t, "SC_BODYPAINT imported", skillEffectIDs(db.SkillSCBodypaint), effectStretch)
	expectEffectIDs(t, "SC_ENERVATION imported", skillEffectIDs(db.SkillSCEnervation), effectEnervation)
	expectEffectIDs(t, "SC_GROOMY imported", skillEffectIDs(db.SkillSCGroomy), effectEnervation2)
	expectEffectIDs(t, "SC_IGNORANCE imported", skillEffectIDs(db.SkillSCIgnorance), effectEnervation3)
	expectEffectIDs(t, "SC_LAZINESS imported", skillEffectIDs(db.SkillSCLaziness), effectEnervation4)
	expectEffectIDs(t, "SC_UNLUCKY imported", skillEffectIDs(db.SkillSCUnlucky), effectEnervation5)
	expectEffectIDs(t, "SC_WEAKNESS imported", skillEffectIDs(db.SkillSCWeakness), effectEnervation6)
	expectEffectIDs(t, "SC_MANHOLE imported ground", skillGroundEffectIDs(db.SkillSCManhole), effectBottomManhole)
	expectEffectIDs(t, "SC_MANHOLE imported success", skillSuccessEffectIDs(db.SkillSCManhole), effectManhole)
	expectEffectIDs(t, "SC_DIMENSIONDOOR imported ground", skillGroundEffectIDs(db.SkillSCDimensiondoor), effectForestLight6)
	expectEffectIDs(t, "SC_CHAOSPANIC imported ground", skillGroundEffectIDs(db.SkillSCChaospanic), effectBottomAni)
	expectEffectIDs(t, "SC_MAELSTROM imported ground", skillGroundEffectIDs(db.SkillSCMaelstrom), effectBottomMaelstrom)
	expectEffectIDs(t, "SC_BLOODYLUST imported ground", skillGroundEffectIDs(db.SkillSCBloodylust), effectBottomBloodyLust)
	expectEffectIDs(t, "LG_SHIELDPRESS imported before hit", skillBeforeHitEffectIDs(db.SkillLGShieldpress), effectPressure2)
	expectEffectIDs(t, "LG_PRESTIGE imported", skillEffectIDs(db.SkillLGPrestige), effectPrimeCharge2)
	expectEffectIDs(t, "LG_BANDING imported", skillEffectIDs(db.SkillLGBanding), effectPrimeCharge3)
	expectEffectIDs(t, "LG_INSPIRATION imported", skillEffectIDs(db.SkillLGInspiration), effectPrimeCharge4)
	expectEffectIDs(t, "SO_FIREWALK imported ground", skillGroundEffectIDs(db.SkillSOFirewalk), effectFireWall2)
	expectEffectIDs(t, "SO_ELECTRICWALK imported ground", skillGroundEffectIDs(db.SkillSOElectricwalk), effectShockwave2)
	expectEffectIDs(t, "SO_DIAMONDDUST imported", skillEffectIDs(db.SkillSODiamonddust), effectColdThrow2)
	expectEffectIDs(t, "SO_PSYCHIC_WAVE imported", skillEffectIDs(db.SkillSOPsychicWave), effectSprPlant10)
	expectEffectIDs(t, "SO_WARMER imported", skillEffectIDs(db.SkillSOWarmer), effectDemonicFire4)
	expectEffectIDs(t, "SO_VARETYR_SPEAR imported before hit", skillBeforeHitEffectIDs(db.SkillSOVaretyrSpear), effectPressure3)
	expectEffectIDs(t, "WM_REVERBERATION imported ground", skillGroundEffectIDs(db.SkillWmReverberation), effectBotReverb)
	expectEffectIDs(t, "WM_REVERBERATION_MELEE imported", skillEffectIDs(db.SkillWmReverberationMelee), effectBotReverb2)
	expectEffectIDs(t, "WM_SEVERE_RAINSTORM imported", skillEffectIDs(db.SkillWmSevereRainstorm), effectRainParticle)
	expectEffectIDs(t, "WM_POEMOFNETHERWORLD imported ground", skillGroundEffectIDs(db.SkillWmPoemofnetherworld), effectBotReverb2)
	expectEffectIDs(t, "WM_VOICEOFSIREN imported ground", skillGroundEffectIDs(db.SkillWmVoiceofsiren), effectHeartAsura)
	expectEffectIDs(t, "WM_LULLABY_DEEPSLEEP imported", skillEffectIDs(db.SkillWmLullabyDeepsleep), effectChemicalV2)
	expectEffectIDs(t, "WM_SIRCLEOFNATURE imported", skillEffectIDs(db.SkillWmSircleofnature), effectCirclePower2)
	expectEffectIDs(t, "WM_RANDOMIZESPELL imported", skillEffectIDs(db.SkillWmRandomizespell), effectSecra2)
	expectEffectIDs(t, "WM_GLOOMYDAY imported", skillEffectIDs(db.SkillWmGloomyday), effectDance1)
	expectEffectIDs(t, "WM_SONG_OF_MANA imported ground", skillGroundEffectIDs(db.SkillWmSongOfMana), effectSprPlant3)
	expectEffectIDs(t, "WM_DANCE_WITH_WUG imported ground", skillGroundEffectIDs(db.SkillWmDanceWithWug), effectSprPlant2)
	expectEffectIDs(t, "WM_SATURDAY_NIGHT_FEVER imported ground", skillGroundEffectIDs(db.SkillWmSaturdayNightFever), effectSprPlant4)
	expectEffectIDs(t, "WM_LERADS_DEW imported ground", skillGroundEffectIDs(db.SkillWmLeradsDew), effectSprPlant5)
	expectEffectIDs(t, "WM_MELODYOFSINK imported ground", skillGroundEffectIDs(db.SkillWmMelodyofsink), effectSprPlant6)
	expectEffectIDs(t, "WM_BEYOND_OF_WARCRY imported ground", skillGroundEffectIDs(db.SkillWmBeyondOfWarcry), effectSprPlant7)
	expectEffectIDs(t, "WM_UNLIMITED_HUMMING_VOICE imported ground", skillGroundEffectIDs(db.SkillWmUnlimitedHummingVoice), effectSprPlant8)
	expectEffectIDs(t, "HAMI_CASTLE imported", skillEffectIDs(db.SkillHamiCastle), effectHamiCastle)
	expectEffectIDs(t, "HAMI_DEFENCE imported", skillEffectIDs(db.SkillHamiDefence), effectHamiDefence)
	expectEffectIDs(t, "HAMI_BLOODLUST imported", skillEffectIDs(db.SkillHamiBloodlust), effectHamiBlood)
	expectEffectIDs(t, "MH_POISON_MIST imported", skillEffectIDs(db.SkillMhPoisonMist), effectPoisonMist)
	expectEffectIDs(t, "MH_ERASER_CUTTER imported", skillEffectIDs(db.SkillMhEraserCutter), effectEraserCutter)
	expectEffectIDs(t, "MH_SONIC_CRAW imported", skillEffectIDs(db.SkillMhSonicCraw), effectSonicClaw)
	expectEffectIDs(t, "MH_MIDNIGHT_FRENZY imported", skillEffectIDs(db.SkillMhMidnightFrenzy), effectMidnightFrenzy)
	expectEffectIDs(t, "MH_TINDER_BREAKER imported", skillEffectIDs(db.SkillMhTinderBreaker), effectTinderBreaker)
	expectEffectIDs(t, "MH_LAVA_SLIDE imported", skillEffectIDs(db.SkillMhLavaSlide), effectLavaSlide)
	expectEffectIDs(t, "MH_VOLCANIC_ASH imported", skillEffectIDs(db.SkillMhVolcanicAsh), effectVolcanicAsh)
	expectEffectIDs(t, "AB_CHEAL imported", skillEffectIDs(db.SkillABCheal), effectHeal2)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported", skillEffectIDs(db.SkillABHighnessheal), effectHeal4)
	expectEffectIDs(t, "AB_HIGHNESSHEAL imported hit", skillHitEffectIDs(db.SkillABHighnessheal), effectHealOffensive)
}

func TestRobrowserMiniSTREffectSpecs(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		min  string
	}{
		{"Mammonite", effectMammonite, "maemor", "memor_min"},
		{"Angelus", effectAngelus, "angelus", "jong_mini"},
		{"Cure", effectCure, "cure", "cure_min"},
		{"Gloria", effectGloria, "gloria", "gloria_min"},
		{"Magnificat", effectMagnificat, "magnificat", "magnificat_min"},
		{"Resurrection", effectResurrection, "resurrection", "resurrection_min"},
		{"Lex Aeterna", effectLexAeterna, "lexaeterna", "lexaeterna_min"},
		{"Suffragium", effectSuffragium, "suffragium", "suffragium_min"},
		{"Storm Gust", effectStormGust, "stormgust", "storm_min"},
		{"Weapon Perfection", effectWeaponPerfect, "weaponperfection", "weaponperfection_min"},
		{"Maximize Power", effectMaximizePower, "maximizepower", "maximize_min"},
		{"Kyrie Eleison", effectKyrie, "kyrie", "kyrie_min"},
		{"Christmas Carol", effectChristmasCarol, "angelus", "jong_mini"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.strMinFile != tc.min || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q min %q attached", tc.name, component, tc.file, tc.min)
		}
	}
}

func TestMVPEffectSpecMatchesRoBrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectMvp)
	if !ok {
		t.Fatal("MVP effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("MVP components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "mvp" || !component.attachedEntity {
		t.Fatalf("MVP component = %+v, want attached STR mvp", component)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\st_mvp.wav" {
		t.Fatalf("MVP sfx = %#v", spec.sfx)
	}
}

func TestImportedSkillActionFallback(t *testing.T) {
	archer := worldstate.Actor{Job: 3}
	if action := skillAction(db.SkillACDouble).actionFamilyForActor(archer); action != spriteActionPCAttack3 {
		t.Fatalf("AC_DOUBLE action = %d, want ATTACK3", action)
	}
	shower := skillAction(db.SkillACShower)
	if action := shower.actionFamilyForActor(archer); action != attackActionFamilyForActor(archer) {
		t.Fatalf("AC_SHOWER action = %d, want normal attack family", action)
	}
	if shower.speed != 50*time.Millisecond || shower.next == nil || shower.next.action != skillActorActionReadyFight {
		t.Fatalf("AC_SHOWER timing = %+v, want robr speed 50ms then READYFIGHT", shower)
	}
	merchant := worldstate.Actor{Job: 5}
	if action := skillAction(db.SkillMCCartrevolution).actionFamilyForActor(merchant); action != spriteActionPCAttack2 {
		t.Fatalf("MC_CARTREVOLUTION action = %d, want ATTACK2", action)
	}
	counter := skillAction(db.SkillKNAutocounter)
	if !counter.defined || counter.action != skillActorActionAttack || !counter.hasFrame || counter.frame != 0 || counter.play || counter.next != nil {
		t.Fatalf("KN_AUTOCOUNTER action = %+v, want robr attack frame 0 with play=false next=false", counter)
	}
	relax := skillAction(db.SkillLKTensionrelax)
	if !relax.defined || relax.action != skillActorActionNone {
		t.Fatalf("LK_TENSIONRELAX action = %+v, want robr false action", relax)
	}
	trap := skillAction(db.SkillHTLandmine)
	if !trap.defined || trap.action != skillActorActionPickup || !trap.play || trap.repeat || trap.next == nil || trap.next.action != skillActorActionIdle {
		t.Fatalf("HT_LANDMINE action = %+v, want robr PICKUP then IDLE", trap)
	}
	sight := skillAction(db.SkillSNSight)
	if !sight.defined || sight.action != skillActorActionSkill || !sight.play || sight.repeat || sight.next == nil || sight.next.action != skillActorActionIdle {
		t.Fatalf("SN_SIGHT action = %+v, want local skill action for robr ACTION then IDLE", sight)
	}
	for _, skillID := range []uint16{db.SkillDCWinkcharm, db.SkillBDRokisweil} {
		action := skillAction(skillID)
		if !action.defined || action.action != skillActorActionSkill || !action.play || !action.repeat || action.next != nil || !action.hasFrame || action.frame != 1 || action.length != 3 || action.speed != 250*time.Millisecond {
			t.Fatalf("dance/play action for skill %d = %+v, want robr repeating SKILL frame 1", skillID, action)
		}
	}
	sonic := skillAction(db.SkillASSonicblow)
	hits := 0
	sawReadyFight := false
	for spec := &sonic; spec != nil; spec = spec.next {
		if spec.action == skillActorActionReadyFight {
			if !spec.repeat || !spec.play || spec.next != nil {
				t.Fatalf("AS_SONICBLOW ready fight tail = %+v, want robr READYFIGHT loop", spec)
			}
			sawReadyFight = true
			break
		}
		if spec.action != skillActorActionAttack || spec.repeat || !spec.play {
			t.Fatalf("AS_SONICBLOW chain node %d = %+v, want robr ATTACK hit", hits+1, spec)
		}
		hits++
		if hits == 1 {
			if spec.speed != 0 {
				t.Fatalf("AS_SONICBLOW first hit speed = %s, want default", spec.speed)
			}
			continue
		}
		if spec.speed != 30*time.Millisecond {
			t.Fatalf("AS_SONICBLOW hit %d speed = %s, want 30ms", hits, spec.speed)
		}
	}
	if hits != 8 {
		t.Fatalf("AS_SONICBLOW hits = %d, want robr 8-hit chain", hits)
	}
	if !sawReadyFight {
		t.Fatal("AS_SONICBLOW chain missing robr READYFIGHT tail")
	}
}

func TestHunterStringKeyEffectsMatchRobrowser(t *testing.T) {
	cast, ok := worldEffectSpecForID(effectSharpShootingCast)
	if !ok || cast.duration != 10*time.Second || len(cast.components) != 1 {
		t.Fatalf("496_beforecast spec = %+v ok=%t, want one 10s CastRing component", cast, ok)
	}
	component := cast.components[0]
	if component.kind != effectComponentFUNC || component.funcName != "CastRing" || component.funcAdapter != effectFuncCastRing || component.textureName != "ring_jadu" {
		t.Fatalf("496_beforecast component identity = %+v", component)
	}
	if component.bottomSize != 0.8 || component.topSize != 2.45 || component.height != 2.8 || component.posZ != 0.08 {
		t.Fatalf("496_beforecast component shape = %+v", component)
	}
	if component.totalCircleSides != 20 || component.circleSides != 20 || component.alphaMax != 0.9 || !component.fade || !component.rotate || !component.attachedEntity {
		t.Fatalf("496_beforecast component flags = %+v", component)
	}
}

func TestBlacksmithStringKeyEffectsMatchRobrowser(t *testing.T) {
	adrenaline, ok := worldEffectSpecForID(effectAdrenalineCast)
	if !ok || adrenaline.duration != 500*time.Millisecond || len(adrenaline.components) != 0 || len(adrenaline.sfx) != 1 || adrenaline.sfx[0] != "effect\\black_adrenalinerush_a.wav" {
		t.Fatalf("98_beforecast spec = %+v ok=%t, want robr adrenaline pre-cast sound", adrenaline, ok)
	}

	maximize, ok := worldEffectSpecForID(effectMaximizeSounds)
	if !ok || maximize.duration != 950*time.Millisecond || len(maximize.components) != 0 {
		t.Fatalf("maximize_power_sounds spec = %+v ok=%t, want robr delayed sound row", maximize, ok)
	}
	wantSFX := []string{
		"effect\\black_maximize_power_circle.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword.wav",
		"effect\\black_maximize_power_sword_bic.wav",
	}
	wantDelays := []time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond}
	if !reflect.DeepEqual(maximize.sfx, wantSFX) || !reflect.DeepEqual(maximize.sfxDelays, wantDelays) {
		t.Fatalf("maximize_power_sounds sfx = %#v delays %#v", maximize.sfx, maximize.sfxDelays)
	}

	greed, ok := worldEffectSpecForID(effectGreedSound)
	if !ok || greed.duration != 500*time.Millisecond || len(greed.components) != 0 || len(greed.sfx) != 1 || greed.sfx[0] != "effect\\ef_entry.wav" {
		t.Fatalf("ef_greed_sound spec = %+v ok=%t, want robr greed sound", greed, ok)
	}
}

func TestKnightStringKeyEffectsMatchRobrowser(t *testing.T) {
	white, ok := worldEffectSpecForID(effectWhitePulse)
	if !ok || white.duration != 500*time.Millisecond || len(white.components) != 0 || len(white.sfx) != 0 {
		t.Fatalf("white_pulse spec = %+v ok=%t, want robr no-draw 500ms row", white, ok)
	}

	projectile, ok := worldEffectSpecForID(effectSpearProjectile)
	if !ok || projectile.duration != 140*time.Millisecond || len(projectile.components) != 1 {
		t.Fatalf("ef_spear_projectile spec = %+v ok=%t, want one 140ms 3D component", projectile, ok)
	}
	component := projectile.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "창" || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.attachedEntity {
		t.Fatalf("ef_spear_projectile component flags = %+v", component)
	}
	if component.duration != 140*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.posZ != 1 || component.angleStart != 180 || component.angleEnd != 180 {
		t.Fatalf("ef_spear_projectile component timing/shape = %+v", component)
	}
	if component.sizeStart != 100*effectPixelRatio || component.sizeEnd != 100*effectPixelRatio {
		t.Fatalf("ef_spear_projectile size = %.3f/%.3f", component.sizeStart, component.sizeEnd)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"spear_hit_sound", effectSpearHitSound, "_hit_spear.wav"},
		{"enemy_hit_normal1", effectEnemyHitNormal1, "_enemy_hit_normal1.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 || len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s spec = %+v ok=%t, want sound %q", tc.name, spec, ok, tc.wav)
		}
	}

	beforeCast, ok := worldEffectSpecForID(effectSpiralBeforeCast)
	if !ok || beforeCast.duration != 500*time.Millisecond || len(beforeCast.components) != 1 {
		t.Fatalf("339_beforecast spec = %+v ok=%t, want body color FUNC", beforeCast, ok)
	}
	beforeComponent := beforeCast.components[0]
	if beforeComponent.kind != effectComponentFUNC || beforeComponent.funcName != "EffectBodyColor" || beforeComponent.funcAdapter != effectFuncBodyColor || !beforeComponent.attachedEntity {
		t.Fatalf("339_beforecast component = %+v", beforeComponent)
	}

	quake, ok := worldEffectSpecForID(effectQuake)
	if !ok || quake.duration != 650*time.Millisecond || quake.cameraShake != 650*time.Millisecond || len(quake.components) != 1 {
		t.Fatalf("quake spec = %+v ok=%t, want 650ms CameraQuake", quake, ok)
	}
	quakeComponent := quake.components[0]
	if quakeComponent.kind != effectComponentFUNC || quakeComponent.funcName != "CameraQuake" || quakeComponent.duration != 650*time.Millisecond || !quakeComponent.attachedEntity {
		t.Fatalf("quake component = %+v", quakeComponent)
	}
}

func TestCrusaderStringKeyEffectsMatchRobrowser(t *testing.T) {
	projectile, ok := worldEffectSpecForID(effectShieldProjectile)
	if !ok || projectile.duration != 140*time.Millisecond || len(projectile.components) != 1 {
		t.Fatalf("ef_shield_projectile spec = %+v ok=%t, want one 140ms 3D component", projectile, ok)
	}
	component := projectile.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/shield_boomerang.bmp" || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || !component.attachedEntity {
		t.Fatalf("ef_shield_projectile component flags = %+v", component)
	}
	if component.duration != 140*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.posZ != 1 || component.angleStart != 180 || component.angleEnd != 540 {
		t.Fatalf("ef_shield_projectile component timing/shape = %+v", component)
	}
	if component.sizeStart != 50*effectPixelRatio || component.sizeEnd != 50*effectPixelRatio {
		t.Fatalf("ef_shield_projectile size = %.3f/%.3f", component.sizeStart, component.sizeEnd)
	}

	gospel, ok := worldEffectSpecForID(effectGospelGround)
	if !ok || gospel.duration != 1500*time.Millisecond || len(gospel.components) != 2 {
		t.Fatalf("370_ground spec = %+v ok=%t, want two song ground components", gospel, ok)
	}
	tile, cross := gospel.components[0], gospel.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 255, G: 255, B: 255, A: 13}) || tile.sizeStart != 1 || !tile.renderBefore || tile.attachedEntity {
		t.Fatalf("370_ground tile = %+v", tile)
	}
	if cross.kind != effectComponentFUNC || cross.funcName != "GroundTexture" || cross.funcAdapter != effectFuncGroundTexture || cross.textureFile != "effect/cross_old.bmp" || cross.duration != 1500*time.Millisecond || cross.sizeStart != 0.5 || cross.sizeEnd != 0.5 || cross.alphaMax != 0.7 || cross.posZ != 0.4 || !cross.blendAdditive || !cross.renderBefore || cross.attachedEntity {
		t.Fatalf("370_ground cross = %+v", cross)
	}
}

func TestArcherThiefMerchantSkillEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AC_CONCENTRATION", skillEffectIDs(45), effectConcentration)
	expectEffectIDs(t, "AC_DOUBLE begin", skillBeginEffectIDs(46), effectBashBegin)
	expectEffectIDs(t, "AC_DOUBLE before-hit", skillBeforeHitEffectIDs(46), effectArrowShot)
	expectEffectIDs(t, "AC_DOUBLE hit", skillHitEffectIDs(46), effectBashHit)
	expectEffectIDs(t, "AC_SHOWER", skillEffectIDs(47), effectArrowShower)
	expectEffectIDs(t, "AC_SHOWER hit", skillHitEffectIDs(47), effectBashHit)
	expectEffectIDs(t, "AC_CHARGEARROW before-hit", skillBeforeHitEffectIDs(148), effectArrowShot)
	if !skillHidesCastAura(db.SkillACChargearrow) {
		t.Fatal("AC_CHARGEARROW should hide cast aura like roBrowser")
	}
	expectEffectIDs(t, "TF_DOUBLE passive", skillEffectIDs(48))
	expectEffectIDs(t, "TF_MISS passive", skillEffectIDs(49))
	expectEffectIDs(t, "TF_STEAL success", skillSuccessEffectIDs(50), effectSteal)
	expectEffectIDs(t, "TF_HIDING", skillEffectIDs(51))
	expectEffectIDs(t, "TF_POISON hit", skillHitEffectIDs(52), effectPoisonAttack)
	expectEffectIDs(t, "TF_DETOXIFY", skillEffectIDs(53), effectDetoxication)
	expectEffectIDs(t, "TF_SPRINKLESAND", skillEffectIDs(149), effectSprinkleSand)
	expectEffectIDs(t, "TF_BACKSLIDING", skillEffectIDs(150))
	expectEffectIDs(t, "TF_PICKSTONE", skillEffectIDs(151))
	if !skillHidesCastAura(db.SkillTFPickstone) {
		t.Fatal("TF_PICKSTONE should hide cast aura like roBrowser")
	}
	expectEffectIDs(t, "TF_THROWSTONE before-hit", skillBeforeHitEffectIDs(152), effectThrowItem3)
	expectEffectIDs(t, "MC_MAMMONITE", skillEffectIDs(42), effectMammonite)
	expectEffectIDs(t, "MC_CARTREVOLUTION begin", skillBeginEffectIDs(153), effectCartRevolution)
	expectEffectIDs(t, "MC_CARTREVOLUTION hit", skillHitEffectIDs(153), effectCartRevolution)
	expectEffectIDs(t, "MC_LOUD", skillEffectIDs(155), effectLoud)
	expectEffectIDs(t, "AL_HOLYLIGHT", skillEffectIDs(156), effectHolyLight)
}

func TestThiefThrowStoneEffectFollowsRoBrowserTable(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectThrowItem3)
	if !ok || len(spec.components) != 1 {
		t.Fatalf("throw stone spec = %#v ok=%t, want one component", spec, ok)
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/돌.bmp" {
		t.Fatalf("throw stone component = %#v, want stone texture 3D component", component)
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.posZ != 1 {
		t.Fatalf("throw stone trajectory flags = %#v", component)
	}
}

func TestArcherProjectileEffectsFollowRoBrowserTable(t *testing.T) {
	shot, ok := worldEffectSpecForID(effectArrowShot)
	if !ok || len(shot.components) != 1 {
		t.Fatalf("arrow shot spec = %#v ok=%t, want one component", shot, ok)
	}
	component := shot.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "data/sprite/npc/skel_archer_arrow" {
		t.Fatalf("arrow shot component = %#v, want skel_archer_arrow 3D sprite", component)
	}
	if !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.duration != 140*time.Millisecond {
		t.Fatalf("arrow shot robr flags = %#v", component)
	}

	shower, ok := worldEffectSpecForID(effectArrowShower)
	if !ok || len(shower.components) != 1 {
		t.Fatalf("arrow shower spec = %#v ok=%t, want one component", shower, ok)
	}
	component = shower.components[0]
	if component.duplicate != 10 || component.posXEndRand != 1.5 || component.posYEndRand != 1.5 {
		t.Fatalf("arrow shower scatter = %#v, want robr duplicate/scatter values", component)
	}
}

func TestBowNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{X: 10, Y: 20}
	world.Actors[300] = worldstate.Actor{ID: 300, X: 15, Y: 20, Job: 1002, ObjectType: actorObjectTypeMob, HasObjectType: true}
	ctx := client.Context{
		Session: &session.Session{
			AccountID: 200,
			Selected:  session.Character{ID: 200, Job: 3, Weapon: 11},
		},
		World: world,
	}
	mode := &WorldMode{}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    200,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.worldEffects))
	}
	projectile := mode.worldEffects[0]
	if projectile.effectID != effectArrowShot || projectile.actorID != 300 || projectile.targetID != 200 {
		t.Fatalf("normal bow projectile = %+v", projectile)
	}
	if projectile.duration != referenceBowArrowFlightDuration || projectile.expires.Sub(projectile.starts) != referenceBowArrowFlightDuration {
		t.Fatalf("normal bow projectile flight = duration %s lifetime %s, want %s", projectile.duration, projectile.expires.Sub(projectile.starts), referenceBowArrowFlightDuration)
	}
	hit := mode.worldEffects[1]
	if hit.effectID != effectHit1 || hit.actorID != 300 || !hit.starts.Equal(projectile.expires) {
		t.Fatalf("normal bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestArcherMercenaryNormalAttackAddsArrowProjectileEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	world.UpsertActor(worldstate.Actor{
		ID:            400,
		X:             10,
		Y:             20,
		Job:           6017,
		ObjectType:    actorObjectTypeMercenary,
		HasObjectType: true,
	})
	world.UpsertActor(worldstate.Actor{
		ID:            300,
		X:             15,
		Y:             20,
		Job:           1002,
		ObjectType:    actorObjectTypeMob,
		HasObjectType: true,
	})
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyActorActionNotify(ctx, network.ActorActionNotify{
		SourceID:    400,
		TargetID:    300,
		Damage:      12,
		HitCount:    1,
		Action:      0,
		SourceSpeed: 500,
		TargetSpeed: 500,
	})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want arrow projectile and regular hit", len(mode.worldEffects))
	}
	projectile := mode.worldEffects[0]
	if projectile.effectID != effectArrowShot || projectile.actorID != 300 || projectile.targetID != 400 {
		t.Fatalf("mercenary arrow projectile = %+v", projectile)
	}
	hit := mode.worldEffects[1]
	if hit.effectID != effectHit1 || hit.actorID != 300 || !hit.starts.After(projectile.starts) {
		t.Fatalf("mercenary bow hit = %+v projectile=%+v", hit, projectile)
	}
}

func TestWarpEffectMappings(t *testing.T) {
	expectEffectIDs(t, "AL_TELEPORT begin", skillBeginEffectIDs(26))
	expectEffectIDs(t, "Butterfly Wing item", itemUseEffectIDs(602), effectTeleportation)
	expectEffectIDs(t, "Fly Wing item", itemUseEffectIDs(601))
}

func TestSpeedPotionItemEffectMappingsMatchRobrowser(t *testing.T) {
	expectEffectIDs(t, "Concentration Potion item", itemUseEffectIDs(645), effectItemFast)
	expectEffectIDs(t, "Awakening Potion item", itemUseEffectIDs(656), effectItemFast2)
	expectEffectIDs(t, "Berserk Potion item", itemUseEffectIDs(657), effectItemFast3)
}

func TestTeleportModalRules(t *testing.T) {
	lv1 := session.Skill{ID: 26, Level: 1, Type: skillTargetEnemy, Range: 9}
	lv2 := session.Skill{ID: 26, Level: 2, Type: skillTargetEnemy, Range: 9}
	if !gameui.TeleportWarpListBypassesModal(lv1, network.WarpPointList{SkillID: 26, MapNames: []string{"Random"}}) {
		t.Fatal("Teleport level 1 should bypass the modal")
	}
	if gameui.TeleportWarpListBypassesModal(lv2, network.WarpPointList{SkillID: 26, MapNames: []string{"Random", "prontera"}}) {
		t.Fatal("Teleport level 2 with a save point should show the modal")
	}
}

func TestWarpPortalListOpensDestinationModal(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{Skills: session.Skills{List: []session.Skill{
		{ID: 27, Level: 4, Type: skillTargetPlace, Range: 9},
	}}}}
	mode.applyWarpPointList(ctx, network.WarpPointList{SkillID: 27, MapNames: []string{"prontera", "geffen", "payon"}})
	if !mode.ui.teleportModal.IsOpen() {
		t.Fatal("warp portal list should open the destination modal")
	}
	if mode.ui.teleportModal.Title() != "Warp Portal" {
		t.Fatalf("modal title = %q", mode.ui.teleportModal.Title())
	}
}

func TestSkillUnitEffectMappings(t *testing.T) {
	expectEffectIDs(t, "UNT_SAFETYWALL", skillUnitEffectIDs(126), effectSafetyWall)
	expectEffectIDs(t, "UNT_FIREWALL", skillUnitEffectIDs(127), effectFireWall)
	expectEffectIDs(t, "UNT_WARPPORTAL", skillUnitEffectIDs(128), effectPortal)
	expectEffectIDs(t, "UNT_PRE_WARPPORTAL", skillUnitEffectIDs(129), effectReadyPortal)
	expectEffectIDs(t, "UNT_SANCTUARY", skillUnitEffectIDs(131), effectBottomSanc)
	expectEffectIDs(t, "UNT_MAGNUS", skillUnitEffectIDs(132), effectBottomMagnus)
	expectEffectIDs(t, "UNT_PNEUMA", skillUnitEffectIDs(133), effectPneuma)
	expectEffectIDs(t, "UNT_FIREPILLAR_WAITING", skillUnitEffectIDs(135), effectFirePillarOn)
	expectEffectIDs(t, "UNT_ICEWALL", skillUnitEffectIDs(141), effectIceWall)
	expectEffectIDs(t, "UNT_QUAGMIRE", skillUnitEffectIDs(142), effectQuagmire)
	expectEffectIDs(t, "UNT_VENOMDUST", skillUnitEffectIDs(146), effectVenomDust2)
	expectEffectIDs(t, "UNT_VOLCANO", skillUnitEffectIDs(154), effectBottomVolcano)
	expectEffectIDs(t, "UNT_DELUGE", skillUnitEffectIDs(155), effectBottomDeluge)
	expectEffectIDs(t, "UNT_VIOLENTGALE", skillUnitEffectIDs(156), effectBottomViolent)
	expectEffectIDs(t, "UNT_LANDPROTECTOR", skillUnitEffectIDs(157), effectBottomLand)
	expectEffectIDs(t, "UNT_LULLABY", skillUnitEffectIDs(158), effectBottomLullabyGround)
	expectEffectIDs(t, "UNT_RICHMANKIM", skillUnitEffectIDs(159), effectBottomRichKimGround)
	expectEffectIDs(t, "UNT_ETERNALCHAOS", skillUnitEffectIDs(160), effectBottomChaosGround)
	expectEffectIDs(t, "UNT_DRUMBATTLEFIELD", skillUnitEffectIDs(161), effectBottomDrumGround)
	expectEffectIDs(t, "UNT_RINGNIBELUNGEN", skillUnitEffectIDs(162), effectBottomNibelungGround)
	expectEffectIDs(t, "UNT_ROKISWEIL", skillUnitEffectIDs(163), effectBottomRokiGround)
	expectEffectIDs(t, "UNT_INTOABYSS", skillUnitEffectIDs(164), effectBottomAbyssGround)
	expectEffectIDs(t, "UNT_SIEGFRIED", skillUnitEffectIDs(165), effectBottomSiegGround)
	expectEffectIDs(t, "UNT_DISSONANCE", skillUnitEffectIDs(166), effectBottomDissonanceGround)
	expectEffectIDs(t, "UNT_WHISTLE", skillUnitEffectIDs(167), effectBottomWhistleGround)
	expectEffectIDs(t, "UNT_ASSASSINCROSS", skillUnitEffectIDs(168), effectBottomSinXGround)
	expectEffectIDs(t, "UNT_POEMBRAGI", skillUnitEffectIDs(169), effectBottomBragiGround)
	expectEffectIDs(t, "UNT_APPLEIDUN", skillUnitEffectIDs(170), effectBottomAppleGround)
	expectEffectIDs(t, "UNT_UGLYDANCE", skillUnitEffectIDs(171), effectBottomUglyDanceGround)
	expectEffectIDs(t, "UNT_HUMMING", skillUnitEffectIDs(172), effectBottomHummingGround)
	expectEffectIDs(t, "UNT_DONTFORGETME", skillUnitEffectIDs(173), effectBottomForgetGround)
	expectEffectIDs(t, "UNT_FORTUNEKISS", skillUnitEffectIDs(174), effectBottomFortuneGround)
	expectEffectIDs(t, "UNT_SERVICEFORYOU", skillUnitEffectIDs(175), effectBottomServiceGround)
	expectEffectIDs(t, "UNT_DEMONSTRATION", skillUnitEffectIDs(177), effectDemonstration)
	expectEffectIDs(t, "UNT_GOSPEL", skillUnitEffectIDs(179), effectGospelGround)
	expectEffectIDs(t, "UNT_BASILICA", skillUnitEffectIDs(180), effectBottomBasilica)
	expectEffectIDs(t, "UNT_MOONLIT", skillUnitEffectIDs(181), effectMoonlit)
	expectEffectIDs(t, "UNT_FOGWALL", skillUnitEffectIDs(182), effectFogWallGround)
	expectEffectIDs(t, "UNT_SPIDERWEB", skillUnitEffectIDs(183), effectBottomSpider)
	expectEffectIDs(t, "UNT_GRAVITATION", skillUnitEffectIDs(184), effectGravitation)
	expectEffectIDs(t, "UNT_HERMODE", skillUnitEffectIDs(185), effectBottomHermode)
	expectEffectIDs(t, "UNT_TATAMIGAESHI", skillUnitEffectIDs(188), effectTatami)
	expectEffectIDs(t, "UNT_KAEN", skillUnitEffectIDs(189), effectKaen)
	expectEffectIDs(t, "UNT_EVILLAND", skillUnitEffectIDs(199), effectBottomEvilLand)
	expectEffectIDs(t, "UNT_EPICLESIS", skillUnitEffectIDs(202), effectGlassWall3)
	expectEffectIDs(t, "UNT_EARTHSTRAIN", skillUnitEffectIDs(203), effectEarthWall)
	expectEffectIDs(t, "UNT_MANHOLE", skillUnitEffectIDs(204), effectBottomManhole)
	expectEffectIDs(t, "UNT_DIMENSIONDOOR", skillUnitEffectIDs(205), effectForestLight6)
	expectEffectIDs(t, "UNT_CHAOSPANIC", skillUnitEffectIDs(206), effectBottomAni)
	expectEffectIDs(t, "UNT_MAELSTROM", skillUnitEffectIDs(207), effectBottomMaelstrom)
	expectEffectIDs(t, "UNT_BLOODYLUST", skillUnitEffectIDs(208), effectBottomBloodyLust)
	expectEffectIDs(t, "UNT_REVERBERATION", skillUnitEffectIDs(218), effectBotReverb)
	expectEffectIDs(t, "UNT_FIREWALK", skillUnitEffectIDs(220), effectFireWall2)
	expectEffectIDs(t, "UNT_ELECTRICWALK", skillUnitEffectIDs(221), effectShockwave2)
	expectEffectIDs(t, "UNT_NETHERWORLD", skillUnitEffectIDs(222), effectBotReverb2)
}

func TestMagnumBreakEffectSpecUsesWorldCylinders(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectMagnumBreak)
	if !ok {
		t.Fatal("magnum break effect spec missing")
	}
	if spec.cameraShake != 0 {
		t.Fatalf("camera shake = %s, want none; quake_magnum carries shake", spec.cameraShake)
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	for i, component := range spec.components {
		if component.kind != effectComponentCylinder {
			t.Fatalf("component %d kind = %d, want cylinder", i, component.kind)
		}
		if component.fixedPerspective {
			t.Fatalf("component %d is fixed perspective, want world-space cylinder", i)
		}
		if component.animation != 4 || component.height <= 0 {
			t.Fatalf("component %d = %+v, want animation 4 with height", i, component)
		}
	}
}

func TestQuakeMagnumEffectStartsCameraShake(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectQuakeMagnum)
	if !ok {
		t.Fatal("quake magnum effect spec missing")
	}
	if spec.cameraShake != 50*time.Millisecond || len(spec.components) != 0 {
		t.Fatalf("quake magnum spec = %+v, want no-draw 50ms camera shake", spec)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.addWorldEffectAt(ctx, effectQuakeMagnum, 2000000, starts) {
		t.Fatal("add quake magnum effect failed")
	}
	if !mode.cameraShakeStart.Equal(starts) || !mode.cameraShakeEnd.Equal(starts.Add(50*time.Millisecond)) {
		t.Fatalf("camera shake = %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(10 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(60 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("expired camera shake offset = %.3f, %.3f, want zero", x, y)
	}
}

func TestSpeedPotionEffectSpecsMatchRobrowserSTRRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		effectID int
		file     string
	}{
		{"Concentration Potion", effectItemFast, "집중"},
		{"Awakening Potion", effectItemFast2, "각성"},
		{"Berserk Potion", effectItemFast3, "버서크"},
	} {
		spec, ok := worldEffectSpecForID(tc.effectID)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ac_concentration.wav" {
			t.Fatalf("%s sfx = %v, want ac_concentration", tc.name, spec.sfx)
		}
	}
}

func TestBerserkPotionEffectStartsDelayedCameraShake(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectItemFast3)
	if !ok {
		t.Fatal("berserk potion effect spec missing")
	}
	if spec.cameraShake != 200*time.Millisecond || spec.cameraShakeDelay != 200*time.Millisecond {
		t.Fatalf("berserk potion shake = delay %s duration %s, want 200ms/200ms", spec.cameraShakeDelay, spec.cameraShake)
	}
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	starts := time.Unix(100, 0)
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	if !mode.addWorldEffectAt(ctx, effectItemFast3, 2000000, starts) {
		t.Fatal("add berserk potion effect failed")
	}
	shakeStart := starts.Add(200 * time.Millisecond)
	shakeEnd := starts.Add(400 * time.Millisecond)
	if !mode.cameraShakeStart.Equal(shakeStart) || !mode.cameraShakeEnd.Equal(shakeEnd) {
		t.Fatalf("camera shake = %s..%s, want %s..%s", mode.cameraShakeStart, mode.cameraShakeEnd, shakeStart, shakeEnd)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(100 * time.Millisecond)); x != 0 || y != 0 {
		t.Fatalf("early camera shake offset = %.3f, %.3f, want zero", x, y)
	}
	if x, y := mode.cameraShakeOffset(starts.Add(250 * time.Millisecond)); x == 0 && y == 0 {
		t.Fatalf("active camera shake offset = %.3f, %.3f, want non-zero", x, y)
	}
}

func TestEndureEffectSpecMatchesRobrowser3DTexture(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEndure)
	if !ok {
		t.Fatal("endure effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/endure.tga" || component.duration != time.Second {
		t.Fatalf("component = %+v", component)
	}
	if !component.fadeIn || !component.fadeOut || !component.sizeSmooth {
		t.Fatalf("component fade/size flags = %+v", component)
	}
	if component.posZ != 2 || component.sizeStart != 200*effectPixelRatio || component.sizeEnd != 70*effectPixelRatio {
		t.Fatalf("component position/size = %+v", component)
	}
}

func TestTeleportationEffectSpecUsesRobrowserCylinderStack(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectTeleportation)
	if !ok {
		t.Fatal("teleportation effect spec missing")
	}
	if spec.duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.duration)
	}
	if !spec.detachLocalActor {
		t.Fatal("teleportation should detach from the local actor")
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	expected := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.6, 0.8, 25},
		{0.8, 1.0, 13},
		{1.0, 1.3, 5},
	}
	for i, want := range expected {
		component := spec.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.blendMode != 2 || !component.blendAdditive {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height {
			t.Fatalf("component %d size = %.1f %.1f %.1f, want %.1f %.1f %.1f", i, component.bottomSize, component.topSize, component.height, want.bottom, want.top, want.height)
		}
		if component.fixedPerspective {
			t.Fatalf("component %d uses fixed perspective, want world-space cylinder", i)
		}
		if !component.attachedEntity {
			t.Fatalf("component %d is not attached to the entity", i)
		}
	}
}

func TestWarpPortalEffectSpecUsesPortal2Cylinders(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectPortal)
	if !ok {
		t.Fatal("portal effect spec missing")
	}
	if len(spec.sfx) != 2 || spec.sfx[0] != "effect\\ef_readyportal.wav" || spec.sfx[1] != "effect\\ef_portal.wav" {
		t.Fatalf("portal sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "ring_blue" || first.duration != 500*time.Millisecond || first.animation != 4 {
		t.Fatalf("first portal component = %+v", first)
	}
	if !first.repeat || first.repeatDelay != -300*time.Millisecond {
		t.Fatalf("first portal repeat = %t delay=%s, want reference client repeat -300ms", first.repeat, first.repeatDelay)
	}
	if spec.components[3].textureName != "alpha1" || spec.components[3].posZ != 2 || spec.components[3].height != 1 {
		t.Fatalf("portal cap component = %+v", spec.components[3])
	}
}

func TestHealEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHeal)
	if !ok {
		t.Fatal("heal effect spec missing")
	}
	if spec.duration != 1840*time.Millisecond {
		t.Fatalf("duration = %s, want 1840ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	for i, component := range spec.components[:2] {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.duration != 1500*time.Millisecond || component.height != 8 || component.alphaMax != 0.2 {
			t.Fatalf("component %d timing/shape = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) || !component.blendAdditive {
			t.Fatalf("component %d tint/blend = %+v", i, component)
		}
	}
	firstParticle := spec.components[2]
	if firstParticle.kind != effectComponent3D || firstParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("first heal particle = %+v", firstParticle)
	}
	if firstParticle.duration != 1300*time.Millisecond || firstParticle.delay != 400*time.Millisecond || firstParticle.duplicateDelay != 10*time.Millisecond || firstParticle.duplicate != 15 {
		t.Fatalf("first heal particle timing = %+v", firstParticle)
	}
	if firstParticle.alphaMax != 0.6 || !firstParticle.fadeIn || !firstParticle.fadeOut || firstParticle.sparkling || firstParticle.sparkNumber != 0 {
		t.Fatalf("first heal particle fade = %+v", firstParticle)
	}
	if firstParticle.posXRand != 1.5 || firstParticle.posYRand != 1.5 || firstParticle.posZEndRand != 2 || firstParticle.posZEndMiddle != 6 {
		t.Fatalf("first heal particle position = %+v", firstParticle)
	}
	if firstParticle.sizeStart != 9*effectPixelRatio || firstParticle.sizeEnd != 9*effectPixelRatio || firstParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("first heal particle size = %+v", firstParticle)
	}
	secondParticle := spec.components[3]
	if secondParticle.kind != effectComponent3D || secondParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("second heal particle = %+v", secondParticle)
	}
	if secondParticle.duration != 1100*time.Millisecond || secondParticle.delay != 200*time.Millisecond || secondParticle.duplicateDelay != 50*time.Millisecond || secondParticle.duplicate != 7 {
		t.Fatalf("second heal particle timing = %+v", secondParticle)
	}
	if secondParticle.alphaMax != 0.6 || !secondParticle.fadeIn || !secondParticle.fadeOut || secondParticle.sparkling || secondParticle.sparkNumber != 0 {
		t.Fatalf("second heal particle fade = %+v", secondParticle)
	}
	if secondParticle.posXRand != 1 || secondParticle.posYRand != 1 || secondParticle.posZEnd != 5 || secondParticle.posZStartRand != 1 {
		t.Fatalf("second heal particle position = %+v", secondParticle)
	}
	if secondParticle.sizeStart != 9*effectPixelRatio || secondParticle.sizeEnd != 9*effectPixelRatio || secondParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("second heal particle size = %+v", secondParticle)
	}
}

func TestHealOffensiveEffectSpecUsesRobrowserCylindersAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectHealOffensive)
	if !ok {
		t.Fatal("offensive heal effect spec missing")
	}
	if spec.duration != 1490*time.Millisecond {
		t.Fatalf("duration = %s, want 1490ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	for i, component := range spec.components[:2] {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.animation != 1 {
			t.Fatalf("component %d = %+v", i, component)
		}
		if component.duration != time.Second || !component.blendAdditive || component.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
			t.Fatalf("component %d timing/tint = %+v", i, component)
		}
	}
	if spec.components[0].height != 10 || spec.components[1].height != 9 {
		t.Fatalf("cylinder heights = %.1f %.1f", spec.components[0].height, spec.components[1].height)
	}
	firstParticle := spec.components[2]
	if firstParticle.kind != effectComponent3D || firstParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("first offensive heal particle = %+v", firstParticle)
	}
	if firstParticle.duration != time.Second || firstParticle.delay != 400*time.Millisecond || firstParticle.duplicateDelay != 10*time.Millisecond || firstParticle.duplicate != 10 {
		t.Fatalf("first offensive heal particle timing = %+v", firstParticle)
	}
	if firstParticle.alphaMax != 0.8 || !firstParticle.fadeIn || !firstParticle.fadeOut || !firstParticle.blendAdditive || !firstParticle.sparkling || firstParticle.sparkNumber != 2 {
		t.Fatalf("first offensive heal particle fade/blend = %+v", firstParticle)
	}
	if firstParticle.posXRand != 1.5 || firstParticle.posYRand != 1.5 || firstParticle.posZEndRand != 3 || firstParticle.posZEndMiddle != 6 {
		t.Fatalf("first offensive heal particle position = %+v", firstParticle)
	}
	secondParticle := spec.components[3]
	if secondParticle.kind != effectComponent3D || secondParticle.textureFile != "effect/pok3.tga" {
		t.Fatalf("second offensive heal particle = %+v", secondParticle)
	}
	if secondParticle.duration != 900*time.Millisecond || secondParticle.delay != 200*time.Millisecond || secondParticle.duplicateDelay != 50*time.Millisecond || secondParticle.duplicate != 5 {
		t.Fatalf("second offensive heal particle timing = %+v", secondParticle)
	}
	if secondParticle.alphaMax != 0.8 || !secondParticle.fadeIn || !secondParticle.fadeOut || !secondParticle.blendAdditive || !secondParticle.sparkling || secondParticle.sparkNumber != 2 {
		t.Fatalf("second offensive heal particle fade/blend = %+v", secondParticle)
	}
	if secondParticle.posXRand != 1 || secondParticle.posYRand != 1 || secondParticle.posZEnd != 6 || secondParticle.posZStartRand != 1 {
		t.Fatalf("second offensive heal particle position = %+v", secondParticle)
	}
	if secondParticle.sizeStart != 9*effectPixelRatio || secondParticle.sizeEnd != 9*effectPixelRatio || secondParticle.sizeRand != 2*effectPixelRatio {
		t.Fatalf("second offensive heal particle size = %+v", secondParticle)
	}
}

func TestIncreaseAgilityEffectSpecUsesRobrowserParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectIncAgility)
	if !ok {
		t.Fatal("increase agility effect spec missing")
	}
	if spec.duration != 1500*time.Millisecond {
		t.Fatalf("duration = %s, want 1500ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_incagility.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	particleCases := []struct {
		index     int
		alphaMax  float64
		delay     time.Duration
		duplicate int
	}{
		{index: 0, alphaMax: 1, delay: 500 * time.Millisecond, duplicate: 7},
		{index: 1, alphaMax: 0.75, delay: 400 * time.Millisecond, duplicate: 3},
		{index: 2, alphaMax: 1, delay: 0, duplicate: 10},
	}
	for _, tc := range particleCases {
		component := spec.components[tc.index]
		if component.kind != effectComponent3D || component.textureFile != "effect/ac_center2.tga" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.duration != 1000*time.Millisecond || component.delay != tc.delay || component.duplicateDelay != 200*time.Millisecond || component.duplicate != tc.duplicate {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.alphaMax != tc.alphaMax || component.fadeIn || !component.fadeOut {
			t.Fatalf("particle %d fade = %+v", tc.index, component)
		}
		if component.posXRand != 1.5 || component.posYRand != 1 || component.posZStartRand != 1 || component.posZStartMiddle != 1 || component.posZEndRand != 1 || component.posZEndMiddle != 6 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.sizeStartX != 2.5*effectPixelRatio || component.sizeEndX != 2.5*effectPixelRatio {
			t.Fatalf("particle %d x size = %+v", tc.index, component)
		}
		if component.sizeStartY != 0 || component.sizeEndY != 0 || component.sizeRandY != 15*effectPixelRatio || component.sizeRandYMiddle != 45*effectPixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
		if component.blendAdditive {
			t.Fatalf("particle %d should use normal alpha blending", tc.index)
		}
	}
	overlay := spec.components[3]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/agi_up.bmp" {
		t.Fatalf("overlay resource = %+v", overlay)
	}
	if overlay.duration != 1000*time.Millisecond || overlay.alphaMax != 1 || !overlay.fadeIn || !overlay.fadeOut {
		t.Fatalf("overlay timing/fade = %+v", overlay)
	}
	if overlay.posZ != 0.4 || overlay.posZEnd != 3 {
		t.Fatalf("overlay position = %+v", overlay)
	}
	if overlay.sizeStart != 100*effectPixelRatio || overlay.sizeEnd != 100*effectPixelRatio || overlay.sizeStartY != 45*effectPixelRatio || overlay.sizeEndY != 45*effectPixelRatio || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if !overlay.overlay {
		t.Fatal("overlay should use reference client overlay rendering")
	}
	if overlay.blendAdditive {
		t.Fatal("overlay should use normal alpha blending")
	}
}

func TestDecreaseAgilityEffectSpecUsesRobrowserParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectDecAgility)
	if !ok {
		t.Fatal("decrease agility effect spec missing")
	}
	if spec.duration != 1000*time.Millisecond {
		t.Fatalf("duration = %s, want 1000ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_decagility.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 2 {
		t.Fatalf("components = %d, want 2", len(spec.components))
	}
	particle := spec.components[0]
	if particle.kind != effectComponent3D || particle.textureFile != "effect/ac_center2.tga" {
		t.Fatalf("particle resource = %+v", particle)
	}
	if particle.duration != 1000*time.Millisecond || particle.duplicateDelay != 200*time.Millisecond || particle.duplicate != 20 {
		t.Fatalf("particle timing = %+v", particle)
	}
	if particle.alphaMax != 1 || particle.fadeIn || !particle.fadeOut {
		t.Fatalf("particle fade = %+v", particle)
	}
	if particle.posXRand != 1.5 || particle.posYRand != 1 || particle.posZStartRand != 1 || particle.posZStartMiddle != 6 || particle.posZEndRand != 1 || particle.posZEndMiddle != 1 {
		t.Fatalf("particle position = %+v", particle)
	}
	if particle.sizeStartX != effectTableSize(2.5) || particle.sizeEndX != effectTableSize(2.5) {
		t.Fatalf("particle x size = %+v", particle)
	}
	if particle.sizeStartY != 0 || particle.sizeEndY != 0 || particle.sizeRandY != effectTableSize(15) || particle.sizeRandYMiddle != effectTableSize(45) {
		t.Fatalf("particle size = %+v", particle)
	}
	if particle.blendAdditive {
		t.Fatal("particle should use normal alpha blending")
	}
	overlay := spec.components[1]
	if overlay.kind != effectComponent3D || overlay.textureFile != "effect/slow.bmp" {
		t.Fatalf("overlay resource = %+v", overlay)
	}
	if overlay.duration != 1000*time.Millisecond || overlay.alphaMax != 1 || !overlay.fadeIn || !overlay.fadeOut {
		t.Fatalf("overlay timing/fade = %+v", overlay)
	}
	if overlay.posZ != 2.8 || overlay.posZEnd != 0.4 {
		t.Fatalf("overlay position = %+v", overlay)
	}
	if overlay.sizeStart != effectTableSize(100) || overlay.sizeEnd != effectTableSize(100) || overlay.sizeStartY != effectTableSize(45) || overlay.sizeEndY != effectTableSize(45) || !overlay.sizeSmooth {
		t.Fatalf("overlay size = %+v", overlay)
	}
	if overlay.overlay {
		t.Fatal("overlay should use regular reference client 3D rendering")
	}
	if overlay.blendAdditive {
		t.Fatal("overlay should use normal alpha blending")
	}
}

func TestAngelusEffectSpecUsesRobrowserSTR(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectAngelus)
	if !ok {
		t.Fatal("angelus effect spec missing")
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_angelus.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSTR || component.strFile != "angelus" || component.strMinFile != "jong_mini" {
		t.Fatalf("STR resource = %+v", component)
	}
	if !component.attachedEntity || !component.spriteHead {
		t.Fatalf("STR attachment flags = %+v", component)
	}
}

func TestBlessingEffectSpecUsesRobrowserSpritesAndParticles(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBlessing)
	if !ok {
		t.Fatal("blessing effect spec missing")
	}
	if spec.duration != 2500*time.Millisecond {
		t.Fatalf("duration = %s, want 2500ms", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_blessing.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 4 {
		t.Fatalf("components = %d, want 4", len(spec.components))
	}
	sprite := spec.components[0]
	if sprite.kind != effectComponentSPR || sprite.spriteFile != "축복" {
		t.Fatalf("sprite component = %+v", sprite)
	}
	if sprite.duration != 1500*time.Millisecond || sprite.spriteDelay != 30*time.Millisecond || !sprite.spriteRepeat || !sprite.spriteHead || sprite.spriteYOffset != -120 || !sprite.worldSizedSprite {
		t.Fatalf("sprite timing/placement = %+v", sprite)
	}

	particleCases := []struct {
		index       int
		delay       time.Duration
		posXRand    float64
		posYRand    float64
		sparkling   bool
		sparkNumber int
	}{
		{index: 1, delay: 300 * time.Millisecond, posXRand: 1.2, posYRand: 1, sparkling: true, sparkNumber: 2},
		{index: 2, delay: 400 * time.Millisecond, posXRand: 1.4, posYRand: 1.1},
	}
	for _, tc := range particleCases {
		component := spec.components[tc.index]
		if component.kind != effectComponent3D || component.spriteFile != "particle6" {
			t.Fatalf("particle %d resource = %+v", tc.index, component)
		}
		if component.duration != 1200*time.Millisecond || component.delay != tc.delay || component.duplicateDelay != 0 || component.duplicate != 6 {
			t.Fatalf("particle %d timing = %+v", tc.index, component)
		}
		if component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || component.sparkling != tc.sparkling || component.sparkNumber != tc.sparkNumber {
			t.Fatalf("particle %d fade/sparkle = %+v", tc.index, component)
		}
		if component.posXRand != tc.posXRand || component.posYRand != tc.posYRand || component.posZStartRand != 2 || component.posZStartMiddle != 5.5 || component.posZEndRand != 0.5 || component.posZEndMiddle != 1 {
			t.Fatalf("particle %d position = %+v", tc.index, component)
		}
		if component.sizeStart != 50*effectPixelRatio || component.sizeEnd != 50*effectPixelRatio {
			t.Fatalf("particle %d size = %+v", tc.index, component)
		}
	}

	aura := spec.components[3]
	if aura.funcAdapter != effectFuncBlessingCircle || aura.textureName != "alpha_down" || !aura.overlay {
		t.Fatalf("blessing ground circle = %+v", aura)
	}
}

func TestEmotionEffectSpecUsesEntityAttachmentOffset(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEmotion)
	if !ok {
		t.Fatal("emotion effect spec missing")
	}
	if len(spec.components) != 1 {
		t.Fatalf("components = %d, want 1", len(spec.components))
	}
	component := spec.components[0]
	if component.kind != effectComponentSPR || component.spriteFile != "emotion" {
		t.Fatalf("emotion component = %+v", component)
	}
	if !component.attachedEntity || component.spriteYOffset != -100 || component.spriteHead {
		t.Fatalf("emotion placement = %+v", component)
	}
}

func TestSPREffectShowsEveryFrameBeforeExpiring(t *testing.T) {
	image := render.NewImage(4, 4)
	image.Fill(color.White)
	animation := res.ACTAnimation{Layers: []res.ACTLayer{{
		ScaleX: 1, ScaleY: 1, Color: [4]float32{1, 1, 1, 1},
	}}}
	act := &res.ACT{Actions: make([]res.ACTAction, 20)}
	act.Actions[0] = res.ACTAction{Animations: []res.ACTAnimation{animation}, DelayMS: 50}
	// Pupa's sweating emotion uses action 19: 24 frames at 50 ms each.
	act.Actions[19] = res.ACTAction{Animations: make([]res.ACTAnimation, 24), DelayMS: 50}
	for i := range act.Actions[19].Animations {
		act.Actions[19].Animations[i] = animation
	}
	view := &spriteView{
		act:        act,
		spr:        &res.SPR{Frames: []res.SPRFrame{{Width: 4, Height: 4}}},
		images:     map[spriteFrameKey]*render.Image{{}: image},
		billboards: make(map[singleSpriteBillboardKey]*spriteBillboard),
	}
	mode := WorldMode{effectViews: map[string]*spriteView{"emotion": view}}
	ctx := client.Context{Resources: &res.Manager{}}
	projection := newSceneProjectionForTarget(800, 600, 0, 0, 0)
	starts := time.Unix(10, 0)
	for _, tc := range []struct {
		name                  string
		action, ms            int
		stop, repeat, visible bool
	}{
		{"first", 19, 0, false, false, true},
		{"last begins", 19, 1150, false, false, true},
		{"last ends", 19, 1199, false, false, true},
		{"expired", 19, 1200, false, false, false},
		{"single frame", 0, 0, false, false, true},
		{"single expired", 0, 50, false, false, false},
		{"hold last", 19, 1200, true, false, true},
		{"repeat", 19, 1200, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := render.NewFrame(800, 600)
			effect := worldEffect{starts: starts, hasSpriteFrame: true, spriteFrameOverride: tc.action}
			component := worldEffectComponent{spriteFile: "emotion", spriteStopAtEnd: tc.stop, spriteRepeat: tc.repeat}
			mode.drawSPREffect(screen, ctx, projection, effect, component, 0, 0, 0, starts.Add(time.Duration(tc.ms)*time.Millisecond))
			visible := !reflect.DeepEqual(screen, render.NewFrame(800, 600))
			if visible != tc.visible {
				t.Fatalf("visible=%t, want %t", visible, tc.visible)
			}
		})
	}
}

func TestWorldEffectDuplicateDeltasMatchRobrowserSemantics(t *testing.T) {
	component := worldEffectComponent{
		alphaMax:      0.2,
		alphaMaxDelta: 0.2,
		sizeStart:     100 * effectPixelRatio,
		sizeEnd:       100 * effectPixelRatio,
		sizeDelta:     -10,
	}
	if got := effectBillboardAlphaForDuplicate(0.5, component, 2); math.Abs(got-0.6) > 0.001 {
		t.Fatalf("duplicate alpha = %.3f, want 0.6", got)
	}
	sizeX, sizeY := effect3DSize(component, worldEffect{}, 0, 0.5, 2)
	want := 80 * effectPixelRatio
	if math.Abs(sizeX-want) > 0.001 || math.Abs(sizeY-want) > 0.001 {
		t.Fatalf("duplicate size = %.3f x %.3f, want %.3f", sizeX, sizeY, want)
	}
}

func TestEffect3DSpriteScaleUsesRobrowserSpriteUnits(t *testing.T) {
	size := effectTableSize(200)
	got := effect3DSpriteScale(size)
	want := size / 100
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("sprite pixel scale = %.5f, want %.5f", got, want)
	}
	fireballWorldWidth := 64 * got
	wantWidth := 128 * effectPixelRatio
	if math.Abs(fireballWorldWidth-wantWidth) > 0.001 {
		t.Fatalf("64px fireball width = %.3f, want robr 128/35 %.3f", fireballWorldWidth, wantWidth)
	}
}

func TestEffect3DSpriteDrawOptionsHonorAdditiveBlend(t *testing.T) {
	defaultOptions := effect3DSpriteDrawOptions(worldEffectComponent{})
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default sprite effect blend = %v, want source-over", got)
	}
	if got := defaultOptions.DepthBias; got != 0 {
		t.Fatalf("default sprite effect depth bias = %.3f, want 0", got)
	}
	if got := effect3DSpriteDrawOptions(worldEffectComponent{blendAdditive: true}).Blend; got != render.BlendLighter {
		t.Fatalf("additive sprite effect blend = %v, want lighter", got)
	}
	if got := effect3DSpriteDrawOptions(worldEffectComponent{worldSizedSprite: true}).DepthBias; got != strEffectDepthBias {
		t.Fatalf("world-sized sprite effect depth bias = %.3f, want %.3f", got, strEffectDepthBias)
	}
	if effect3DSpriteDrawOptions(worldEffectComponent{overlay: true}).DepthTest {
		t.Fatal("overlay sprite effect depth test enabled, want disabled")
	}
}

func TestTexturedEffectBillboardDrawOptionsHonorOverlay(t *testing.T) {
	defaultOptions := texturedEffectBillboardDrawOptions(false, false)
	if !defaultOptions.DepthTest {
		t.Fatal("default textured effect depth test disabled, want enabled")
	}
	if got := defaultOptions.Blend; got != render.BlendSourceOver {
		t.Fatalf("default textured effect blend = %v, want source-over", got)
	}
	additiveOptions := texturedEffectBillboardDrawOptions(true, false)
	if got := additiveOptions.Blend; got != render.BlendLighter {
		t.Fatalf("additive textured effect blend = %v, want lighter", got)
	}
	overlayOptions := texturedEffectBillboardDrawOptions(false, true)
	if overlayOptions.DepthTest {
		t.Fatal("overlay textured effect depth test enabled, want disabled")
	}
}

func TestWorldSizedSpriteBillboardUsesCenterDepthLikeRobrowser(t *testing.T) {
	options := render.DrawTrianglesOptions{DepthBias: 0.01}
	cmd := worldSpriteBillboardCommand(
		render.WhiteImage(),
		options,
		modelPoint3{x: 1, y: 2, z: 3},
		modelPoint3{x: 4, y: 5, z: 6},
		modelPoint3{x: 7, y: 8, z: 9},
		10,
		20,
		3,
		4,
		color.RGBA{R: 51, G: 102, B: 153, A: 204},
	)

	if cmd.UpAxis != [3]float32{7, 8, 9} {
		t.Fatalf("visible up axis = %+v, want rendered billboard up axis", cmd.UpAxis)
	}
	if cmd.DepthUpAxis != [3]float32{} {
		t.Fatalf("depth up axis = %+v, want center-depth axis", cmd.DepthUpAxis)
	}
	if cmd.DepthBias != options.DepthBias || cmd.Options.DepthBias != options.DepthBias {
		t.Fatalf("depth bias = command %.3f options %.3f, want %.3f", cmd.DepthBias, cmd.Options.DepthBias, options.DepthBias)
	}
}

func TestWorldEffectOrbitReplacesBasePositionLikeRobrowser(t *testing.T) {
	component := worldEffectComponent{
		posX:           -2,
		posY:           4,
		orbitRadiusX:   3,
		orbitRadiusY:   3,
		orbitRotations: 8,
		orbitPhase:     0.7,
		orbitClockwise: true,
	}
	x, y, _ := (&WorldMode{}).effect3DOffset(client.Context{}, component, worldEffect{}, 0, 0, 0, 0, 0, 0)
	angle := -0.7 * math.Pi / 2
	wantX := -math.Cos(angle) * 3
	wantY := math.Sin(angle) * 3
	if math.Abs(x-wantX) > 0.001 || math.Abs(y-wantY) > 0.001 {
		t.Fatalf("orbit offset = %.3f, %.3f; want %.3f, %.3f", x, y, wantX, wantY)
	}
	if math.Abs(x-(-2+wantX)) < 0.001 || math.Abs(y-(4+wantY)) < 0.001 {
		t.Fatalf("orbit offset incorrectly included base position: %.3f, %.3f", x, y)
	}
}

func TestWorldEffectBillboardSparklingAlphaMatchesRobrowser(t *testing.T) {
	component := worldEffectComponent{
		alphaMax:    1,
		sparkling:   true,
		sparkNumber: 2,
	}
	if got := effectBillboardAlphaForDuplicate(0, component, 0); math.Abs(got-1) > 0.001 {
		t.Fatalf("spark alpha at start = %.3f, want 1", got)
	}
	if got := effectBillboardAlphaForDuplicate(0.25, component, 0); math.Abs(got-0.008) > 0.001 {
		t.Fatalf("spark alpha at quarter = %.3f, want about 0.008", got)
	}
	if got := effectBillboardAlphaForDuplicate(0.75, component, 0); math.Abs(got-0.067) > 0.001 {
		t.Fatalf("spark alpha at three quarters = %.3f, want about 0.067", got)
	}
}

func TestWorldEffectBillboardAngleCanRotateWithCamera(t *testing.T) {
	projection := newSceneProjectionForTargetYaw(800, 600, 0, 0, 0, 45)
	component := worldEffectComponent{angleStart: 90, angleEnd: 180, rotateWithCamera: true}
	got := worldEffectBillboardAngleForEffect(component, projection, worldEffect{}, 0, 0.5)
	want := degreesToRadians(180)
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("angle = %.3f, want %.3f", got, want)
	}
}

func TestEffectCylinderAngleXRotatesHeightAxisLikeRobrowser(t *testing.T) {
	got := rotateEffectCylinderVector(modelPoint3{y: 1}, -90, 0, 0)
	want := modelPoint3{z: -1}
	if !modelPointNear(got, want, 0.001) {
		t.Fatalf("rotated height axis = %+v, want %+v", got, want)
	}
}

func TestWorldCylinderBandAllowsRobrowserNegativeHeight(t *testing.T) {
	screen := render.NewFrame(320, 240)
	drawWorldCylinderBandWithBasis(screen, render.WhiteImage(), render.WhiteImage(), 0, 0, 0, 1, 2, -4, color.RGBA{A: 255}, 8, modelPoint3{x: 1}, modelPoint3{z: 1}, modelPoint3{y: 1})
	commands := reflect.ValueOf(screen).Elem().FieldByName("worldCommands")
	if commands.Len() != 1 {
		t.Fatalf("world commands = %d, want one negative-height cylinder", commands.Len())
	}
	vertices := commands.Index(0).FieldByName("Vertices")
	if vertices.Len() < 2 {
		t.Fatalf("vertices = %d, want cylinder vertices", vertices.Len())
	}
	bottomY := vertices.Index(0).FieldByName("Y").Float()
	topY := vertices.Index(1).FieldByName("Y").Float()
	if topY >= bottomY {
		t.Fatalf("top Y = %.1f bottom Y = %.1f, want top below bottom", topY, bottomY)
	}
}

func TestSTRAnimationAttachedEntityUsesActorAnchor(t *testing.T) {
	anim := res.STRAnimation{Pos: [2]float32{320, 320}}

	_, groundY := strAnimationOffset(anim, false)
	_, attachedY := strAnimationOffset(anim, true)

	if groundY != -0.5 {
		t.Fatalf("ground STR y offset = %.3f, want -0.5", groundY)
	}
	if attachedY != 0 {
		t.Fatalf("attached STR y offset = %.3f, want 0", attachedY)
	}
}

func TestSTRAnimationLocalOffsetFlipsYBeforeRotationLikeRobrowser(t *testing.T) {
	anim := res.STRAnimation{
		Pos:   [2]float32{330, 340},
		Angle: 90,
	}
	gotX, gotY := strAnimationLocalOffset(anim, 10, 20, true)
	wantX, wantY := -10.0/35.0, -30.0/35.0
	if math.Abs(gotX-wantX) > 0.0001 || math.Abs(gotY-wantY) > 0.0001 {
		t.Fatalf("STR local offset = %.4f %.4f, want %.4f %.4f", gotX, gotY, wantX, wantY)
	}
}

func TestSTRAnimationBlendMatchesRobrowserD3DBlend(t *testing.T) {
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 7}); got != render.BlendSrcAlphaDstAlpha {
		t.Fatalf("SRC_ALPHA/DST_ALPHA blend = %v, want BlendSrcAlphaDstAlpha", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 2}); got != render.BlendLighter {
		t.Fatalf("SRC_ALPHA/ONE blend = %v, want BlendLighter", got)
	}
	if got := strAnimationBlend(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6}); got != render.BlendSourceOver {
		t.Fatalf("regular STR blend = %v, want BlendSourceOver", got)
	}
}

func TestSTRAnimationDrawOptionsDisableFogToMatchRobrowser(t *testing.T) {
	options := strAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if !options.DisableFog {
		t.Fatal("STR draw options enabled map fog, want disabled")
	}
}

func TestSTRAnimationDrawOptionsUseDepthBiasToMatchRobrowser(t *testing.T) {
	options := strAnimationDrawOptions(res.STRAnimation{SrcAlpha: 5, DestAlpha: 6})
	if options.DepthBias <= 0 {
		t.Fatalf("STR depth bias = %.3f, want positive robr-style camera bias", options.DepthBias)
	}
}

func TestSTRAnimationVertexUsesCenterDepthToMatchRobrowser(t *testing.T) {
	point := modelPoint3{x: 1, y: 2, z: 3}
	depthPoint := modelPoint3{x: 4, y: 5, z: 6}
	vertex := strAnimationVertex3D(point, texturePoint{u: 0.25, v: 0.5}, color.RGBA{R: 51, G: 102, B: 153, A: 204}, 80, 40, depthPoint)

	if vertex.X != 1 || vertex.Y != 2 || vertex.Z != 3 {
		t.Fatalf("vertex position = %.1f %.1f %.1f, want rendered point", vertex.X, vertex.Y, vertex.Z)
	}
	if vertex.DepthX != 4 || vertex.DepthY != 5 || vertex.DepthZ != 6 {
		t.Fatalf("vertex depth = %.1f %.1f %.1f, want STR center depth", vertex.DepthX, vertex.DepthY, vertex.DepthZ)
	}
	if vertex.SrcX != 20 || vertex.SrcY != 20 {
		t.Fatalf("vertex uv = %.1f %.1f, want texture pixel coords", vertex.SrcX, vertex.SrcY)
	}
}

func TestLevelUpEffectSpecsUseSTRResources(t *testing.T) {
	base, ok := worldEffectSpecForID(effectBaseLevelUp)
	if !ok {
		t.Fatal("base level-up effect spec missing")
	}
	if len(base.components) != 1 || base.components[0].kind != effectComponentSTR || base.components[0].strFile != "angel" || !base.components[0].attachedEntity {
		t.Fatalf("base level-up spec = %+v", base)
	}
	if len(base.sfx) != 1 || base.sfx[0] != "levelup.wav" {
		t.Fatalf("base level-up sfx = %#v", base.sfx)
	}
	job, ok := worldEffectSpecForID(effectJobLevelUp)
	if !ok {
		t.Fatal("job level-up effect spec missing")
	}
	if len(job.components) != 1 || job.components[0].kind != effectComponentSTR || job.components[0].strFile != "joblvup" {
		t.Fatalf("job level-up spec = %+v", job)
	}
	if len(job.sfx) != 0 {
		t.Fatalf("job level-up sfx = %#v", job.sfx)
	}
}

func TestSpecialEffectNotifyAddsLevelUpEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectBaseLevelUp || mode.worldEffects[1].effectID != effectJobLevelUp {
		t.Fatalf("world effects = %+v", mode.worldEffects)
	}
	if len(mode.scheduledSounds) != 1 {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
	if mode.scheduledSounds[0].paths[0] != "levelup.wav" {
		t.Fatalf("base level-up sound = %+v", mode.scheduledSounds[0])
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("local level-up effects did not show both availability notifications")
	}
}

func TestRemoteLevelUpEffectDoesNotShowAvailabilityNotification(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectBaseLevelUp})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 3000000, EffectID: network.SpecialEffectJobLevelUp})

	if mode.ui.levelUpNotifications.BaseVisible() || mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("remote level-up effect showed a local availability notification")
	}
}

func TestLevelUpNotificationActionsOpenStatusAndSkillWindows(t *testing.T) {
	mode := &WorldMode{}
	ctx := client.Context{
		Session:   &session.Session{},
		UIManager: gameui.NewManager(),
		ScreenW:   800,
		ScreenH:   600,
	}

	if !mode.handleLevelUpNotificationAction(ctx, gameui.LevelUpNotificationBase|gameui.LevelUpNotificationJob) {
		t.Fatal("level-up notification action was not handled")
	}
	if !mode.ui.statsWindow.IsOpen() {
		t.Fatal("base notification did not open the status window")
	}
	if !mode.ui.skillWindow.IsOpen() {
		t.Fatal("job notification did not open the skill window")
	}
}

func TestSpecialEffectNotifyMapsServerResultEffects(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	for _, effectID := range []uint32{
		network.SpecialEffectRefineFailure,
		network.SpecialEffectRefineSuccess,
		network.SpecialEffectPharmacySuccess,
		network.SpecialEffectPharmacyFailure,
		network.SpecialEffectGameOver,
	} {
		mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: effectID})
	}

	want := []int{effectRefineOK, effectPharmacyFail}
	wantSFX := []string{
		"effect\\bs_refinefailed.wav",
		"effect\\bs_refinesuccess.wav",
		"effect\\p_success.wav",
		"effect\\p_failed.wav",
	}
	if len(mode.worldEffects) != len(want) {
		t.Fatalf("world effects = %d, want %d: %+v", len(mode.worldEffects), len(want), mode.worldEffects)
	}
	for i, effect := range mode.worldEffects {
		if effect.effectID != want[i] {
			t.Fatalf("world effect %d = %+v, want effect %d", i, effect, want[i])
		}
	}
	if len(mode.scheduledSounds) != len(wantSFX) {
		t.Fatalf("scheduled sounds = %d, want %d: %+v", len(mode.scheduledSounds), len(wantSFX), mode.scheduledSounds)
	}
	for i, sound := range mode.scheduledSounds {
		if len(sound.paths) != 1 || sound.paths[0] != wantSFX[i] {
			t.Fatalf("scheduled sound %d = %+v, want %q", i, sound, wantSFX[i])
		}
	}
}

func TestMakingItemWindowEnterConsumesBeforeConsole(t *testing.T) {
	inputState := input.NewState()
	netClient := network.NewClient(20080910, false)
	mode := &WorldMode{}
	ctx := client.Context{
		Input:   inputState,
		Network: netClient,
		ScreenW: 800,
		ScreenH: 600,
	}
	mode.ui.makingItem.OpenList(ctx, network.MakingItemList{
		Items: []network.MakingItemOption{{ItemID: 501}},
	})

	inputState.SetKey(input.KeyEnter, true)
	if _, err := mode.Update(ctx); err != nil {
		t.Fatal(err)
	}

	if mode.ui.console.Active() {
		t.Fatal("console became active before making item window consumed Enter")
	}
	if !mode.ui.makingItem.IsOpen() {
		t.Fatal("making item window closed even though the disconnected test client could not send")
	}
}

func TestMVPNotifyAddsMVPBannerEffect(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	mode := &WorldMode{}
	ctx := client.Context{Session: &session.Session{AccountID: 2000000}, World: world}

	mode.applyMVPNotify(ctx, network.MVPNotify{AID: 2000000})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectMvp || effect.actorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 1 || mode.scheduledSounds[0].paths[0] != "effect\\st_mvp.wav" {
		t.Fatalf("scheduled sounds = %+v", mode.scheduledSounds)
	}
}

func TestParameterChangeLevelUpFallbackIsDeduped(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{BaseLevel: 10, JobLevel: 4},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusBaseLevel, Value: 11})
	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 5})

	if len(mode.worldEffects) != 2 {
		t.Fatalf("world effects = %d, want 2", len(mode.worldEffects))
	}
	if mode.worldEffects[0].effectID != effectBaseLevelUp || mode.worldEffects[1].effectID != effectJobLevelUp {
		t.Fatalf("world effects = %+v", mode.worldEffects)
	}
	if !mode.ui.levelUpNotifications.BaseVisible() || !mode.ui.levelUpNotifications.JobVisible() {
		t.Fatal("parameter level-up fallback did not show both availability notifications")
	}
}

func TestSpecialEffectNotifyDedupesParameterLevelUpFallback(t *testing.T) {
	world := worldstate.New()
	world.Player = worldstate.Actor{ID: 2000000, X: 10, Y: 20}
	sessionState := &session.Session{
		AccountID: 2000000,
		Progress:  session.Progress{JobLevel: 21},
	}
	mode := &WorldMode{}
	ctx := client.Context{Session: sessionState, World: world}

	mode.applyParameterChange(ctx, network.ParameterChange{VarID: network.StatusJobLevel, Value: 22})
	mode.applySpecialEffectNotify(ctx, network.SpecialEffectNotify{AID: 2000000, EffectID: network.SpecialEffectJobLevelUp})

	if len(mode.worldEffects) != 1 {
		t.Fatalf("world effects = %d, want 1", len(mode.worldEffects))
	}
	if effect := mode.worldEffects[0]; effect.effectID != effectJobLevelUp || effect.actorID != 2000000 {
		t.Fatalf("world effect = %+v", effect)
	}
	if len(mode.scheduledSounds) != 0 {
		t.Fatalf("scheduled sounds = %+v, want none for job level-up", mode.scheduledSounds)
	}
}
