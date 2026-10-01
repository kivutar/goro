package game

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
)

func TestBashBeginEffectSpecUsesCylinderComponents(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBashBegin)
	if !ok {
		t.Fatal("bash begin effect spec missing")
	}
	if spec.duration != time.Second {
		t.Fatalf("duration = %s, want 1s", spec.duration)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\ef_bash.wav" {
		t.Fatalf("sfx = %#v", spec.sfx)
	}
	if len(spec.components) != 3 {
		t.Fatalf("components = %d, want 3", len(spec.components))
	}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "alpha_down" || first.circleSides != 20 || first.totalCircleSides != 20 {
		t.Fatalf("first component = %+v", first)
	}
	second := spec.components[1]
	if second.kind != effectComponentCylinder || second.textureName != "alpha_center" || second.duplicate != 10 || second.circleSides != 1 || second.totalCircleSides != 30 {
		t.Fatalf("second component = %+v", second)
	}
	third := spec.components[2]
	if third.kind != effectComponentCylinder || third.textureName != "alpha_center" || third.duplicate != 8 || third.topSize != 4.0 {
		t.Fatalf("third component = %+v", third)
	}
}

func TestWorldEffectSpecCatalogCoverage(t *testing.T) {
	coverage := effectCoverageSnapshot()
	if coverage.Implemented != 663 {
		t.Fatalf("implemented effects = %d, want 663", coverage.Implemented)
	}
	if coverage.ReferenceActive != 607 || coverage.ReferenceAll != 1147 {
		t.Fatalf("reference client totals = active %d all %d", coverage.ReferenceActive, coverage.ReferenceAll)
	}
	if coverage.ActivePercent < 109.2 || coverage.ActivePercent > 109.3 {
		t.Fatalf("active coverage = %.3f, want about 109.2", coverage.ActivePercent)
	}
}

func TestRobrowserActiveEffectsZeroToFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectHit1:           "EF_HIT1",
		effectBashHit:        "EF_HIT2",
		effectHit3:           "EF_HIT3",
		effectHit4:           "EF_HIT4",
		effectHit5:           "EF_HIT5",
		effectHit6:           "EF_HIT6",
		effectEntry:          "EF_ENTRY",
		effectExit:           "EF_EXIT",
		effectWarp:           "EF_WARP",
		effectEnhance:        "EF_ENHANCE",
		effectMammonite:      "EF_COIN",
		effectEndure:         "EF_ENDURE",
		effectBeginSpell:     "EF_BEGINSPELL",
		effectGlassWall:      "EF_GLASSWALL",
		effectHealSP:         "EF_HEALSP",
		effectSoulStrike:     "EF_SOULSTRIKE",
		effectBashBegin:      "EF_BASH",
		effectMagnumBreak:    "EF_MAGNUMBREAK",
		effectSteal:          "EF_STEAL",
		effectPoisonAttack:   "EF_PATTACK",
		effectDetoxication:   "EF_DETOXICATION",
		effectSight:          "EF_SIGHT",
		effectStoneCurse:     "EF_STONECURSE",
		effectFireBall:       "EF_FIREBALL",
		effectFireWall:       "EF_FIREWALL",
		effectIceArrow:       "EF_ICEARROW",
		effectFrostDiver:     "EF_FROSTDIVER",
		effectFrostDiverHit:  "EF_FROSTDIVER2",
		effectLightningBolt:  "EF_LIGHTBOLT",
		effectThunderStorm:   "EF_THUNDERSTORM",
		effectFireArrow:      "EF_FIREARROW",
		effectNapalmBeat:     "EF_NAPALMBEAT",
		effectRuwach:         "EF_RUWACH",
		effectTeleportOld:    "EF_TELEPORTATION",
		effectReadyPortalOld: "EF_READYPORTAL",
		effectIncAgility:     "EF_INCAGILITY",
		effectDecAgility:     "EF_DECAGILITY",
		effectAqua:           "EF_AQUA",
		effectSignum:         "EF_SIGNUM",
		effectAngelus:        "EF_ANGELUS",
		effectBlessing:       "EF_BLESSING",
		effectIncAgiDex:      "EF_INCAGIDEX",
		effectSmoke:          "EF_SMOKE",
		effectFirefly:        "EF_FIREFLY",
		effectTorch:          "EF_TORCH",
		effectFireHit:        "EF_FIREHIT",
		effectFireSplashHit:  "EF_FIRESPLASHHIT",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiftyToOneHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectFireSplashHit:  "EF_FIRESPLASHHIT",
		effectColdHit:        "EF_COLDHIT",
		effectWindHit:        "EF_WINDHIT",
		effectPoisonHit:      "EF_POISONHIT",
		effectBeginSpell2:    "EF_BEGINSPELL2",
		effectBeginSpell3:    "EF_BEGINSPELL3",
		effectBeginSpell4:    "EF_BEGINSPELL4",
		effectBeginSpell5:    "EF_BEGINSPELL5",
		effectBeginSpell6:    "EF_BEGINSPELL6",
		effectBeginSpell7:    "EF_BEGINSPELL7",
		effectLockOnTarget:   "EF_LOCKON",
		effectWarpZone:       "EF_WARPZONE",
		effectSightTrasher:   "EF_SIGHTRASHER",
		effectArrowShotRO:    "EF_ARROWSHOT",
		effectInvenom:        "EF_INVENOM",
		effectCure:           "EF_CURE",
		effectProvoke:        "EF_PROVOKE",
		effectMvp:            "EF_MVP",
		effectSkidTrap:       "EF_SKIDTRAP",
		effectBrandishSpear:  "EF_BRANDISHSPEAR",
		effectIceWall:        "EF_ICEWALL",
		effectGloria:         "EF_GLORIA",
		effectMagnificat:     "EF_MAGNIFICAT",
		effectResurrection:   "EF_RESURRECTION",
		effectRecovery:       "EF_RECOVERY",
		effectEarthSpike:     "EF_EARTHSPIKE",
		effectSpearBoomerang: "EF_SPEARBMR",
		effectPierce:         "EF_PIERCE",
		effectTurnUndead:     "EF_TURNUNDEAD",
		effectSanctuary:      "EF_SANCTUARY",
		effectImpositio:      "EF_IMPOSITIO",
		effectLexAeterna:     "EF_LEXAETERNA",
		effectAspersio:       "EF_ASPERSIO",
		effectLexDivina:      "EF_LEXDIVINA",
		effectSuffragium:     "EF_SUFFRAGIUM",
		effectStormGust:      "EF_STORMGUST",
		effectLordVermilion:  "EF_LORD",
		effectBenedictio:     "EF_BENEDICTIO",
		effectMeteorStorm:    "EF_METEORSTORM",
		effectJupitelThunder: "EF_YUFITEL",
		effectJupitelHit:     "EF_YUFITELHIT",
		effectQuagmire:       "EF_QUAGMIRE",
		effectFirePillar:     "EF_FIREPILLAR",
		effectFirePillarBomb: "EF_FIREPILLARBOMB",
		effectHasteUp:        "EF_HASTEUP",
		effectFlasher:        "EF_FLASHER",
		effectRemoveTrap:     "EF_REMOVETRAP",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsFiftyToOneHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
	}{
		{"EF_ARROWSHOT", effectArrowShotRO, "arrowshot", "", true},
		{"EF_INVENOM", effectInvenom, "invenom", "effect\\thief_invenom.wav", true},
		{"EF_SKIDTRAP", effectSkidTrap, "skidtrap", "effect\\hunter_skidtrap.wav", false},
		{"EF_BRANDISHSPEAR", effectBrandishSpear, "brandish", "effect\\knight_brandish_spear.wav", false},
		{"EF_RECOVERY", effectRecovery, "recovery", "effect\\priest_recovery.wav", true},
		{"EF_SANCTUARY", effectSanctuary, "sanctuary", "effect\\priest_sanctuary.wav", true},
		{"EF_IMPOSITIO", effectImpositio, "impositio", "effect\\priest_impositio.wav", true},
		{"EF_ASPERSIO", effectAspersio, "aspersio", "effect\\priest_aspersio.wav", true},
		{"EF_LEXDIVINA", effectLexDivina, "lexdivina", "effect\\priest_lexdivina.wav", true},
		{"EF_LORD", effectLordVermilion, "lord", "effect\\wizard_fire_ivy.wav", true},
		{"EF_BENEDICTIO", effectBenedictio, "benedictio", "effect\\priest_benedictio.wav", true},
		{"EF_QUAGMIRE", effectQuagmire, "quagmire", "effect\\wizard_quagmire.wav", false},
		{"EF_FIREPILLAR", effectFirePillar, "firepillar", "effect\\wizard_fire_pillar_a.wav", false},
		{"EF_FIREPILLARBOMB", effectFirePillarBomb, "firepillarbomb", "effect\\wizard_fire_pillar_b.wav", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_SPEARBMR", effectSpearBoomerang, "effect\\ef_fireball.wav"},
		{"EF_PIERCE", effectPierce, "effect\\ef_bash.wav"},
		{"EF_TURNUNDEAD", effectTurnUndead, "effect\\ef_bash.wav"},
		{"EF_HASTEUP", effectHasteUp, "effect\\black_adrenalinerush_b.wav"},
		{"EF_FLASHER", effectFlasher, "effect\\hunter_flasher.wav"},
		{"EF_REMOVETRAP", effectRemoveTrap, "effect\\hunter_removetrap.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserWarpZoneAndPoisonHitSpecs(t *testing.T) {
	poison, ok := worldEffectSpecForID(effectPoisonHit)
	if !ok || len(poison.components) != 1 {
		t.Fatalf("EF_POISONHIT spec = %+v ok=%t, want one SPR component", poison, ok)
	}
	if len(poison.sfx) != 1 || poison.sfx[0] != "effect\\ef_poisonattack.wav" {
		t.Fatalf("EF_POISONHIT sfx = %v", poison.sfx)
	}
	if component := poison.components[0]; component.kind != effectComponentSPR || component.spriteFile != "poisonhit" || component.attachedEntity {
		t.Fatalf("EF_POISONHIT component = %+v", component)
	}

	warp, ok := worldEffectSpecForID(effectWarpZone)
	if !ok || len(warp.components) != 3 || warp.duration != 2800*time.Millisecond {
		t.Fatalf("EF_WARPZONE spec = %+v ok=%t", warp, ok)
	}
	first, second, particle := warp.components[0], warp.components[1], warp.components[2]
	if first.kind != effectComponentCylinder || second.kind != effectComponentCylinder || first.textureName != "ring_blue" || second.textureName != "ring_blue" {
		t.Fatalf("EF_WARPZONE cylinders = %+v %+v", first, second)
	}
	if first.bottomSize != 2 || first.topSize != 3.3 || second.bottomSize != 1.9 || second.topSize != 3.2 || first.height != 1.1 || second.height != 1.1 {
		t.Fatalf("EF_WARPZONE cylinder sizes = %+v %+v", first, second)
	}
	if !first.attachedEntity || !second.attachedEntity || !first.blendAdditive || !second.blendAdditive || !first.fade || !second.fade || !first.rotate || !second.rotate || first.animation != 3 || second.animation != 3 {
		t.Fatalf("EF_WARPZONE cylinder flags = %+v %+v", first, second)
	}
	if particle.kind != effectComponent3D || particle.textureFile != "effect/pok1.tga" || particle.duration != time.Second || particle.duplicate != 3 || !particle.attachedEntity {
		t.Fatalf("EF_WARPZONE particle = %+v", particle)
	}
	if particle.posXStartRand != 3 || particle.posYStartRand != 3 || particle.posZEndRand != 2 || particle.posZEndMiddle != 2 || particle.sizeStart != effectTableSize(100) || particle.sizeRand != effectTableSize(17) {
		t.Fatalf("EF_WARPZONE particle motion/size = %+v", particle)
	}
}

func TestRobrowserSightTrasherEffectSpec(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectSightTrasher)
	if !ok || len(spec.components) != 16 || spec.duration != 800*time.Millisecond {
		t.Fatalf("EF_SIGHTRASHER spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wizard_sightrasher.wav" {
		t.Fatalf("EF_SIGHTRASHER sfx = %v", spec.sfx)
	}
	shadow, sight := spec.components[0], spec.components[1]
	if shadow.kind != effectComponent3D || !shadow.shadowTexture || shadow.spriteFile != "data\\sprite\\shadow" || shadow.duplicate != 4 || shadow.duplicateDelay != 100*time.Millisecond {
		t.Fatalf("EF_SIGHTRASHER shadow = %+v", shadow)
	}
	if shadow.posXEnd != 0 || shadow.posYEnd != -8 || shadow.posZ != 2 || shadow.posZEnd != 2 || shadow.sizeStart != effectTableSize(60) || shadow.sizeEnd != effectTableSize(160) || shadow.sizeDelta != -60 || shadow.blendMode != 8 {
		t.Fatalf("EF_SIGHTRASHER shadow motion/size = %+v", shadow)
	}
	if sight.kind != effectComponent3D || sight.shadowTexture || sight.spriteFile != "sight" || sight.alphaMax != 123.0/255.0 || sight.alphaMaxDelta != 3.0/255.0 || sight.sizeStart != effectTableSize(20) || sight.sizeEnd != effectTableSize(260) {
		t.Fatalf("EF_SIGHTRASHER sight = %+v", sight)
	}
	last := spec.components[len(spec.components)-1]
	if last.posXEnd != -5.66 || last.posYEnd != -5.66 {
		t.Fatalf("EF_SIGHTRASHER northwest component = %+v", last)
	}
}

func TestRobrowserQuadHornEffectSpecs(t *testing.T) {
	ice, ok := worldEffectSpecForID(effectIceWall)
	if !ok || len(ice.components) != 3 || ice.duration != 5*time.Minute {
		t.Fatalf("EF_ICEWALL spec = %+v ok=%t", ice, ok)
	}
	if len(ice.sfx) != 1 || ice.sfx[0] != "effect\\wizard_icewall.wav" {
		t.Fatalf("EF_ICEWALL sfx = %v", ice.sfx)
	}
	firstIce := ice.components[0]
	if firstIce.kind != effectComponentQuadHorn || firstIce.textureFile != "effect/ice.tga" || firstIce.duration != 5*time.Minute || firstIce.blendMode != 8 || firstIce.blendAdditive || firstIce.animation != 1 || firstIce.quadHornAnimSpeed != 50*time.Millisecond {
		t.Fatalf("EF_ICEWALL first component = %+v", firstIce)
	}
	if firstIce.quadHornHeightMin != 2.8 || firstIce.quadHornHeightMax != 3.3 || firstIce.quadHornOffsetXMin != 0.25 || firstIce.quadHornOffsetXMax != 0.75 || firstIce.quadHornOffsetZ != -0.1 || firstIce.quadHornBottomMin != 0.3 || firstIce.quadHornBottomMax != 0.5 || firstIce.quadHornRotateYMin != 1 || firstIce.quadHornRotateYMax != 360 {
		t.Fatalf("EF_ICEWALL robr ranges = %+v", firstIce)
	}
	if ice.components[2].quadHornHeightMin != 2.5 || ice.components[2].quadHornHeightMax != 2.9 {
		t.Fatalf("EF_ICEWALL third height range = %+v", ice.components[2])
	}

	earth, ok := worldEffectSpecForID(effectEarthSpike)
	if !ok || len(earth.components) != 5 || earth.duration != 5*time.Second || earth.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_EARTHSPIKE spec = %+v ok=%t", earth, ok)
	}
	if len(earth.sfx) != 1 || earth.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_EARTHSPIKE sfx = %v", earth.sfx)
	}
	main := earth.components[0]
	if main.kind != effectComponentQuadHorn || main.textureFile != "effect/stone.bmp" || main.duration != 5*time.Second || main.blendMode != 1 || main.animation != 3 || main.quadHornAnimSpeed != 120*time.Millisecond || !main.quadHornAnimOut {
		t.Fatalf("EF_EARTHSPIKE main = %+v", main)
	}
	if main.quadHornHeightMin != 0.95 || main.quadHornHeightMax != 1.5 || main.quadHornRotateZMin != -8 || main.quadHornRotateZMax != 8 {
		t.Fatalf("EF_EARTHSPIKE main ranges = %+v", main)
	}
	last := earth.components[4]
	if last.quadHornOffsetXMin != 0.5 || last.quadHornOffsetXMax != 0.7 || last.quadHornOffsetYMin != 0 || last.quadHornOffsetYMax != -0.2 || last.quadHornAnimSpeed != 100*time.Millisecond {
		t.Fatalf("EF_EARTHSPIKE last ranges = %+v", last)
	}
}

func TestRobrowserMeteorAndJupitelSpecs(t *testing.T) {
	meteor, ok := worldEffectSpecForID(effectMeteorStorm)
	if !ok || len(meteor.components) != 1 {
		t.Fatalf("EF_METEORSTORM spec = %+v ok=%t", meteor, ok)
	}
	if meteor.cameraShakeDelay != 600*time.Millisecond || meteor.cameraShake != 650*time.Millisecond {
		t.Fatalf("EF_METEORSTORM camera shake = delay %s duration %s", meteor.cameraShakeDelay, meteor.cameraShake)
	}
	if len(meteor.sfx) != 1 || meteor.sfx[0] != "effect\\wizard_meteor.wav" {
		t.Fatalf("EF_METEORSTORM sfx = %v", meteor.sfx)
	}
	component := meteor.components[0]
	if component.kind != effectComponentSTR || component.strFile != "meteor%d" || component.strRandMin != 1 || component.strRandMax != 4 || !component.attachedEntity {
		t.Fatalf("EF_METEORSTORM component = %+v", component)
	}

	jupitel, ok := worldEffectSpecForID(effectJupitelThunder)
	if !ok || len(jupitel.components) != 2 || jupitel.duration != 200*time.Millisecond {
		t.Fatalf("EF_YUFITEL spec = %+v ok=%t", jupitel, ok)
	}
	if len(jupitel.sfx) != 1 || jupitel.sfx[0] != "effect\\hunter_shockwavetrap.wav" {
		t.Fatalf("EF_YUFITEL sfx = %v", jupitel.sfx)
	}
	center, ball := jupitel.components[0], jupitel.components[1]
	if center.kind != effectComponent3D || center.textureFile != "effect/thunder_center.bmp" || center.duration != 200*time.Millisecond || center.sizeStart != effectTableSize(35) || !center.toSrc || !center.blendAdditive || !center.overlay || center.alphaMax != 0.66 {
		t.Fatalf("EF_YUFITEL center = %+v", center)
	}
	if ball.kind != effectComponent3D || len(ball.textureFiles) != 6 || ball.textureFiles[0] != "effect/thunder_ball_a.bmp" || ball.textureFiles[5] != "effect/thunder_ball_f.bmp" || ball.frameDelay != 10*time.Millisecond || ball.sizeStart != effectTableSize(45) || !ball.toSrc || !ball.blendAdditive || !ball.overlay {
		t.Fatalf("EF_YUFITEL ball = %+v", ball)
	}

	hit, ok := worldEffectSpecForID(effectJupitelHit)
	if !ok || len(hit.components) != 2 || hit.duration != 300*time.Millisecond {
		t.Fatalf("EF_YUFITELHIT spec = %+v ok=%t", hit, ok)
	}
	pang, blast := hit.components[0], hit.components[1]
	if pang.kind != effectComponent3D || pang.textureFile != "effect/thunder_pang.bmp" || pang.duration != 100*time.Millisecond || pang.sizeStart != 0 || pang.sizeEnd != effectTableSize(25) || !pang.rotateToTarget || !pang.fadeOut || !pang.overlay || !pang.attachedEntity || !pang.blendAdditive {
		t.Fatalf("EF_YUFITELHIT pang = %+v", pang)
	}
	if blast.kind != effectComponent3D || len(blast.textureFiles) != 5 || blast.textureFiles[0] != "effect/thunder_plazma_blast_a.bmp" || blast.textureFiles[4] != "effect/thunder_ball_f.bmp" || blast.frameDelay != 10*time.Millisecond || blast.duration != 300*time.Millisecond || blast.sizeStart != effectTableSize(75) || !blast.overlay || !blast.attachedEntity || !blast.blendAdditive {
		t.Fatalf("EF_YUFITELHIT blast = %+v", blast)
	}
}

func TestRobrowserActiveEffectsOneHundredToOneFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectRemoveTrap:     "EF_REMOVETRAP",
		effectRepairWeapon:   "EF_REPAIRWEAPON",
		effectCrashEarth:     "EF_CRASHEARTH",
		effectWeaponPerfect:  "EF_PERFECTION",
		effectMaximizePower:  "EF_MAXPOWER",
		effectBlastMine:      "EF_BLASTMINE",
		effectBlastMineBomb:  "EF_BLASTMINEBOMB",
		effectClaymore:       "EF_CLAYMORE",
		effectFreezingTrap:   "EF_FREEZING",
		effectBubble:         "EF_BUBBLE",
		effectGasPush:        "EF_GASPUSH",
		effectSpringTrap:     "EF_SPRINGTRAP",
		effectKyrie:          "EF_KYRIE",
		effectMagnus:         "EF_MAGNUS",
		effectBlitzBeat:      "EF_BLITZBEAT",
		effectWaterBall:      "EF_WATERBALL",
		effectWaterBall2:     "EF_WATERBALL2",
		effectDetecting:      "EF_DETECTING",
		effectCloaking:       "EF_CLOAKING",
		effectSonicBlow:      "EF_SONICBLOW",
		effectSonicBlowHit:   "EF_SONICBLOWHIT",
		effectGrimtooth:      "EF_GRIMTOOTH",
		effectVenomDust:      "EF_VENOMDUST",
		effectPoisonReact:    "EF_POISONREACT",
		effectPoisonReact2:   "EF_POISONREACT2",
		effectOverthrust:     "EF_OVERTHRUST",
		effectVenomSplasher:  "EF_SPLASHER",
		effectTwoHandQuicken: "EF_TWOHANDQUICKEN",
		effectAutoCounter:    "EF_AUTOCOUNTER",
		effectGrimtoothAtk:   "EF_GRIMTOOTHATK",
		effectFreeze:         "EF_FREEZE",
		effectFreezed:        "EF_FREEZED",
		effectIceCrash:       "EF_ICECRASH",
		effectSlowPoison:     "EF_SLOWPOISON",
		effectFirePillarOn:   "EF_FIREPILLARON",
		effectSandman:        "EF_SANDMAN",
		effectRevive:         "EF_REVIVE",
		effectPneuma:         "EF_PNEUMA",
		effectHeavenDrive:    "EF_HEAVENDRIVE",
		effectSonicBlow2:     "EF_SONICBLOW2",
		effectBrandishSpear2: "EF_BRANDISH2",
		effectShockwave:      "EF_SHOCKWAVE",
		effectShockwaveHit:   "EF_SHOCKWAVEHIT",
		effectEarthHit:       "EF_EARTHHIT",
		effectPierceSelf:     "EF_PIERCESELF",
		effectBowlingSelf:    "EF_BOWLINGSELF",
		effectSpearStabSelf:  "EF_SPEARSTABSELF",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsOneHundredToOneFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_BLASTMINEBOMB", effectBlastMineBomb, "blastmine", "effect\\hunter_blastmine.wav", false, false},
		{"EF_CLAYMORE", effectClaymore, "claymore", "effect\\hunter_claymoretrap.wav", false, false},
		{"EF_FREEZING", effectFreezingTrap, "freezing", "effect\\hunter_freezingtrap.wav", false, false},
		{"EF_GASPUSH", effectGasPush, "gaspush", "", false, false},
		{"EF_SPRINGTRAP", effectSpringTrap, "spring", "effect\\hunter_springtrap.wav", false, false},
		{"EF_MAGNUS", effectMagnus, "magnus", "effect\\priest_magnus.wav", false, false},
		{"EF_VENOMDUST", effectVenomDust, "venomdust", "effect\\assasin_poisonreact.wav", false, false},
		{"EF_POISONREACT", effectPoisonReact, "poisonreact_1st", "effect\\assasin_poisonreact.wav", true, false},
		{"EF_POISONREACT2", effectPoisonReact2, "poisonreact", "effect\\assasin_poisonreact.wav", true, false},
		{"EF_SPLASHER", effectVenomSplasher, "venomsplasher", "effect\\assasin_venomsplasher.wav", true, false},
		{"EF_TWOHANDQUICKEN", effectTwoHandQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, true},
		{"EF_AUTOCOUNTER", effectAutoCounter, "autocounter", "effect\\knight_autocounter.wav", true, false},
		{"EF_FREEZE", effectFreeze, "freeze", "", true, false},
		{"EF_FREEZED", effectFreezed, "freezed", "", true, false},
		{"EF_ICECRASH", effectIceCrash, "icecrash", "", true, false},
		{"EF_SLOWPOISON", effectSlowPoison, "slowp", "effect\\priest_slowpoison.wav", false, false},
		{"EF_SANDMAN", effectSandman, "sandman", "effect\\hunter_sandman.wav", false, false},
		{"EF_SONICBLOW2", effectSonicBlow2, "sonicblow", "", true, false},
		{"EF_BRANDISH2", effectBrandishSpear2, "brandish2", "effect\\knight_brandish_spear.wav", true, false},
		{"EF_SHOCKWAVEHIT", effectShockwaveHit, "shockwavehit", "", true, false},
		{"EF_EARTHHIT", effectEarthHit, "earthhit", "", true, false},
		{"EF_PIERCESELF", effectPierceSelf, "pierce", "", true, false},
		{"EF_BOWLINGSELF", effectBowlingSelf, "bowling", "_enemy_hit_normal1.wav", true, true},
		{"EF_SPEARSTABSELF", effectSpearStabSelf, "spearstab", "_enemy_hit_normal1.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_BLASTMINE", effectBlastMine, "effect\\hun_anklesnare.wav"},
		{"EF_BLITZBEAT", effectBlitzBeat, "effect\\hunter_blitzbeat.wav"},
		{"EF_DETECTING", effectDetecting, "effect\\hunter_detecting.wav"},
		{"EF_CLOAKING", effectCloaking, "effect\\assasin_cloaking.wav"},
		{"EF_GRIMTOOTH", effectGrimtooth, "effect\\ef_frostdiver.wav"},
		{"EF_OVERTHRUST", effectOverthrust, "effect\\black_overthrust.wav"},
		{"EF_REVIVE", effectRevive, "effect\\priest_resurrection.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserActiveEffectsOneFiftyToTwoHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSpearStabSelf:  "EF_SPEARSTABSELF",
		effectSpearBmrSelf:   "EF_SPEARBMRSELF",
		effectHolyLight:      "EF_HOLYHIT",
		effectConcentration:  "EF_CONCENTRATION",
		effectRefineOK:       "EF_REFINEOK",
		effectRefineFail:     "EF_REFINEFAIL",
		effectJobLevelUp:     "EF_JOBLVUP",
		effectRain:           "EF_RAIN",
		effectSnow:           "EF_SNOW",
		effectSakura:         "EF_SAKURA",
		effectBanjjakii:      "EF_BANJJAKII",
		effectMakeBlur:       "EF_MAKEBLUR",
		effectEnergyCoat:     "EF_ENERGYCOAT",
		effectCartRevolution: "EF_CARTREVOLUTION",
		effectVenomDust2:     "EF_VENOMDUST2",
		effectMentalBreak:    "EF_MENTALBREAK",
		effectMagicalAtkHit:  "EF_MAGICALATTHIT",
		effectSuiExplosion:   "EF_SUI_EXPLOSION",
		effectSuicide:        "EF_SUICIDE",
		effectComboAttack1:   "EF_COMBOATTACK1",
		effectComboAttack2:   "EF_COMBOATTACK2",
		effectComboAttack3:   "EF_COMBOATTACK3",
		effectComboAttack4:   "EF_COMBOATTACK4",
		effectComboAttack5:   "EF_COMBOATTACK5",
		effectGuidedAttack:   "EF_GUIDEDATTACK",
		effectPoisonAttack2:  "EF_POISONATTACK",
		effectSilenceAttack:  "EF_SILENCEATTACK",
		effectStunAttack:     "EF_STUNATTACK",
		effectPetrifyAttack:  "EF_PETRIFYATTACK",
		effectSleepAttack:    "EF_SLEEPATTACK",
		effectPong:           "EF_PONG",
		effectLevel99:        "EF_LEVEL99",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsOneFiftyToTwoHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_SPEARBMRSELF", effectSpearBmrSelf, "spearboomerang", "effect\\knight_spear_boomerang.wav", true, true},
		{"EF_HOLYHIT", effectHolyLight, "holyhit", "", true, false},
		{"EF_CONCENTRATION", effectConcentration, "concentration", "effect\\ac_concentration.wav", true, false},
		{"EF_REFINEOK", effectRefineOK, "bs_refinesuccess", "effect\\bs_refinesuccess.wav", true, false},
		{"EF_REFINEFAIL", effectRefineFail, "bs_refinefailed", "effect\\bs_refinefailed.wav", true, false},
		{"EF_ENERGYCOAT", effectEnergyCoat, "energycoat", "", true, false},
		{"EF_CARTREVOLUTION", effectCartRevolution, "cartrevolution", "effect\\ef_magnumbreak.wav", true, false},
		{"EF_MENTALBREAK", effectMentalBreak, "mentalbreak", "", true, false},
		{"EF_MAGICALATTHIT", effectMagicalAtkHit, "magical", "", true, false},
		{"EF_SUICIDE", effectSuicide, "suicide", "", true, false},
		{"EF_COMBOATTACK1", effectComboAttack1, "yunta_1", "", true, false},
		{"EF_COMBOATTACK2", effectComboAttack2, "yunta_2", "", true, false},
		{"EF_COMBOATTACK3", effectComboAttack3, "yunta_3", "", true, false},
		{"EF_COMBOATTACK4", effectComboAttack4, "yunta_4", "", true, false},
		{"EF_COMBOATTACK5", effectComboAttack5, "yunta_5", "", true, false},
		{"EF_GUIDEDATTACK", effectGuidedAttack, "homing", "", true, false},
		{"EF_POISONATTACK", effectPoisonAttack2, "poison", "", true, false},
		{"EF_SILENCEATTACK", effectSilenceAttack, "silence", "", true, false},
		{"EF_STUNATTACK", effectStunAttack, "stun", "", true, false},
		{"EF_PETRIFYATTACK", effectPetrifyAttack, "stonecurse", "", true, false},
		{"EF_SLEEPATTACK", effectSleepAttack, "sleep", "", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsOneFiftyToTwoHundredMatchTableRows(t *testing.T) {
	banjjakii, ok := worldEffectSpecForID(effectBanjjakii)
	if !ok || banjjakii.duration != time.Second || len(banjjakii.components) != 1 {
		t.Fatalf("EF_BANJJAKII spec = %+v ok=%t", banjjakii, ok)
	}
	if component := banjjakii.components[0]; component.kind != effectComponentSPR || component.spriteFile != "크리스마스" || component.duration != time.Second || !component.attachedEntity {
		t.Fatalf("EF_BANJJAKII component = %+v", component)
	}

	makeBlur, ok := worldEffectSpecForID(effectMakeBlur)
	if !ok || makeBlur.duration != 2*time.Second || len(makeBlur.components) != 1 {
		t.Fatalf("EF_MAKEBLUR spec = %+v ok=%t", makeBlur, ok)
	}
	if component := makeBlur.components[0]; component.kind != effectComponentFUNC || component.funcName != "MakeBlur" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_MAKEBLUR component = %+v", component)
	}

	venomDust, ok := worldEffectSpecForID(effectVenomDust2)
	if !ok || venomDust.duration != 100*time.Millisecond || len(venomDust.components) != 1 {
		t.Fatalf("EF_VENOMDUST2 spec = %+v ok=%t", venomDust, ok)
	}
	component := venomDust.components[0]
	if component.kind != effectComponent3D || component.spriteFile != "particle3" || component.duration != 100*time.Millisecond || !component.repeat || !component.spriteRepeat || !component.attachedEntity {
		t.Fatalf("EF_VENOMDUST2 component = %+v", component)
	}
	if component.alphaMax != 1 || component.sizeStart != effectTableSize(80) || component.sizeEnd != effectTableSize(80) || component.posZ != 0 || component.posZEnd != 0.5 {
		t.Fatalf("EF_VENOMDUST2 scalar fields = %+v", component)
	}

	sui, ok := worldEffectSpecForID(effectSuiExplosion)
	if !ok || len(sui.components) != 2 || sui.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_SUI_EXPLOSION spec = %+v ok=%t", sui, ok)
	}
	if len(sui.sfx) != 1 || sui.sfx[0] != "effect\\ef_hit2.wav" {
		t.Fatalf("EF_SUI_EXPLOSION sfx = %v", sui.sfx)
	}
	if str, quake := sui.components[0], sui.components[1]; str.kind != effectComponentSTR || str.strFile != "sui_explosion" || !str.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_SUI_EXPLOSION components = %+v %+v", str, quake)
	}

	pong, ok := worldEffectSpecForID(effectPong)
	if !ok || len(pong.components) != 1 {
		t.Fatalf("EF_PONG spec = %+v ok=%t", pong, ok)
	}
	if component := pong.components[0]; component.kind != effectComponentSTR || component.strFile != "pong%d" || component.strRandMin != 1 || component.strRandMax != 3 || component.attachedEntity {
		t.Fatalf("EF_PONG component = %+v", component)
	}

	level99, ok := worldEffectSpecForID(effectLevel99)
	if !ok || level99.duration != 5*time.Minute || len(level99.components) != 1 {
		t.Fatalf("EF_LEVEL99 spec = %+v ok=%t", level99, ok)
	}
	if component := level99.components[0]; component.kind != effectComponentFUNC || component.funcName != "Level99Aura" || component.funcAdapter != effectFuncLevel99Aura || component.textureFile != "effect/ring_blue.tga" || !component.attachedEntity {
		t.Fatalf("EF_LEVEL99 component = %+v", component)
	}
}

func TestRobrowserActiveEffectsTwoHundredToTwoFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectLevel99:       "EF_LEVEL99",
		effectLevel99Ground: "EF_LEVEL99_2",
		effectLevel99Bubble: "EF_LEVEL99_3",
		effectGumgang:       "EF_GUMGANG",
		effectPotionRed:     "EF_POTION1",
		effectPotionOrange:  "EF_POTION2",
		effectPotionYellow:  "EF_POTION3",
		effectPotionWhite:   "EF_POTION4",
		effectPotionBlue:    "EF_POTION5",
		effectPotionGreen:   "EF_POTION6",
		effectFood:          "EF_POTION7",
		effectFoodBlue:      "EF_POTION8",
		effectDarkBreath:    "EF_DARKBREATH",
		effectDefender:      "EF_DEFFENDER",
		effectKeeping:       "EF_KEEPING",
		effectSummonSlave:   "EF_SUMMONSLAVE",
		effectBloodDrain:    "EF_BLOODDRAIN",
		effectEnergyDrain:   "EF_ENERGYDRAIN",
		effectItemFast:      "EF_POTION_CON",
		effectItemFast2:     "EF_POTION_",
		effectItemFast3:     "EF_POTION_BERSERK",
		effectCrusaderDef:   "EF_DEFENDER",
		effectGrandCross:    "EF_GRANDCROSS",
		effectIntimidate:    "EF_INTIMIDATE",
		effectChookgi:       "EF_CHOOKGI",
		effectCloud:         "EF_CLOUD",
		effectCloud2:        "EF_CLOUD2",
		effectLineLink:      "EF_LINELINK",
		effectCloud3:        "EF_CLOUD3",
		effectSpellBreaker:  "EF_SPELLBREAKER",
		effectDispell:       "EF_DISPELL",
		effectBottomVolcano: "EF_BOTTOM_VO",
		effectBottomDeluge:  "EF_BOTTOM_DE",
		effectBottomViolent: "EF_BOTTOM_VI",
		effectBottomLand:    "EF_BOTTOM_LA",
		effectMagicRod:      "EF_MAGICROD",
		effectHolyCross:     "EF_HOLYCROSS",
		effectShieldCharge:  "EF_SHIELDCHARGE",
		effectProvidence:    "EF_PROVIDENCE",
		effectShieldBoomer:  "EF_SHIELDBOOMERANG",
		effectSpearQuicken:  "EF_SPEARQUICKEN",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsTwoHundredToTwoFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_DEFFENDER", effectDefender, "deffender", "", true, false},
		{"EF_KEEPING", effectKeeping, "keeping", "", true, false},
		{"EF_POTION_CON", effectItemFast, "집중", "effect\\ac_concentration.wav", true, false},
		{"EF_POTION_", effectItemFast2, "각성", "effect\\ac_concentration.wav", true, false},
		{"EF_POTION_BERSERK", effectItemFast3, "버서크", "effect\\ac_concentration.wav", true, false},
		{"EF_SPELLBREAKER", effectSpellBreaker, "spell", "effect\\sage_spell breake.wav", true, false},
		{"EF_DISPELL", effectDispell, "디스펠", "", true, false},
		{"EF_MAGICROD", effectMagicRod, "매직로드", "effect\\sage_magic rod.wav", true, false},
		{"EF_HOLYCROSS", effectHolyCross, "holy_cross", "effect\\cru_holy cross.wav", true, false},
		{"EF_SHIELDCHARGE", effectShieldCharge, "shield_charge", "", true, false},
		{"EF_PROVIDENCE", effectProvidence, "providence", "", true, false},
		{"EF_SPEARQUICKEN", effectSpearQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_INTIMIDATE", effectIntimidate, "effect\\rog_intimidate.wav"},
		{"EF_SHIELDBOOMERANG", effectShieldBoomer, "effect\\cru_shield boomerang.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestReferencePotionEffectsTwoHundredRowsAreAttached(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_POTION1", effectPotionRed, "빨간포션", "_heal_effect.wav"},
		{"EF_POTION2", effectPotionOrange, "주홍포션", "_heal_effect.wav"},
		{"EF_POTION3", effectPotionYellow, "노란포션", "_heal_effect.wav"},
		{"EF_POTION4", effectPotionWhite, "하얀포션", "_heal_effect.wav"},
		{"EF_POTION5", effectPotionBlue, "파란포션", "effect\\흡기.wav"},
		{"EF_POTION6", effectPotionGreen, "초록포션", ""},
		{"EF_POTION7", effectFood, "fruit", "_heal_effect.wav"},
		{"EF_POTION8", effectFoodBlue, "fruit_", "effect\\흡기.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsTwoHundredToTwoFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  effectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_2", effectLevel99Ground, "GroundAura", effectFuncGroundAura, "effect/pikapika2.bmp"},
		{"EF_LEVEL99_3", effectLevel99Bubble, "Level99Bubble", effectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_CHOOKGI", effectChookgi, "SpiritSphere", effectFuncSpiritSphere, "effect/thunder_center.bmp"},
		{"EF_BOTTOM_LA", effectBottomLand, "LandProtectorGround", effectFuncLandProtectorGround, "effect/aaa copy.bmp"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one FUNC component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != tc.adapter || component.textureFile != tc.texture {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	gumgang, ok := worldEffectSpecForID(effectGumgang)
	if !ok || len(gumgang.components) != 5 || gumgang.duration != 3600*time.Millisecond {
		t.Fatalf("EF_GUMGANG spec = %+v ok=%t", gumgang, ok)
	}
	for i, component := range gumgang.components {
		wantFile := fmt.Sprintf("effect/super%d.bmp", i+1)
		wantDelay := time.Duration(i) * 400 * time.Millisecond
		if component.kind != effectComponent3D || component.textureFile != wantFile || component.duration != 2*time.Second || component.delay != wantDelay || !component.fadeOut || !component.attachedEntity {
			t.Fatalf("EF_GUMGANG component %d = %+v", i, component)
		}
	}

	dark, ok := worldEffectSpecForID(effectDarkBreath)
	if !ok || len(dark.components) != 1 {
		t.Fatalf("EF_DARKBREATH spec = %+v ok=%t", dark, ok)
	}
	if component := dark.components[0]; component.kind != effectComponentSPR || component.spriteFile != "darkbreath" || !component.spriteHead || !component.attachedEntity {
		t.Fatalf("EF_DARKBREATH component = %+v", component)
	}
	slave, ok := worldEffectSpecForID(effectSummonSlave)
	if !ok || len(slave.components) != 1 {
		t.Fatalf("EF_SUMMONSLAVE spec = %+v ok=%t", slave, ok)
	}
	if component := slave.components[0]; component.kind != effectComponentSPR || component.spriteFile != "smoke" || !component.attachedEntity {
		t.Fatalf("EF_SUMMONSLAVE component = %+v", component)
	}

	for _, tc := range []struct {
		name  string
		id    int
		r     uint8
		g     uint8
		b     uint8
		funcs int
	}{
		{"EF_BLOODDRAIN", effectBloodDrain, 255, 102, 102, 0},
		{"EF_ENERGYDRAIN", effectEnergyDrain, 102, 102, 255, 2},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1+tc.funcs {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.spriteFile != "data/sprite/이팩트/particle1" || component.duration != 600*time.Millisecond || component.duplicate != 5 || !component.toSrc || !component.rotateToTarget || component.arc != 3 || component.retreat != 3 {
			t.Fatalf("%s particle = %+v", tc.name, component)
		}
		if component.color.R != tc.r || component.color.G != tc.g || component.color.B != tc.b || component.sizeStart != effectTableSize(150) || component.sizeEnd != effectTableSize(180) || component.posZ != 5 {
			t.Fatalf("%s particle scalar fields = %+v", tc.name, component)
		}
	}

	defender, ok := worldEffectSpecForID(effectCrusaderDef)
	if !ok || len(defender.components) != 1 || defender.duration != 3*time.Second {
		t.Fatalf("EF_DEFENDER spec = %+v ok=%t", defender, ok)
	}
	if component := defender.components[0]; component.kind != effectComponentCylinder || component.textureName != "ring_black" || component.duration != 3*time.Second || component.alphaMax != 0.6 || component.blendMode != 8 || component.bottomSize != 1.5 || component.topSize != 1.5 || component.height != 10 || !component.rotate || !component.fade || !component.attachedEntity {
		t.Fatalf("EF_DEFENDER component = %+v", component)
	}

	grand, ok := worldEffectSpecForID(effectGrandCross)
	if !ok || len(grand.components) != 25 || grand.duration != 2*time.Second {
		t.Fatalf("EF_GRANDCROSS spec = %+v ok=%t", grand, ok)
	}
	if len(grand.sfx) != 1 || grand.sfx[0] != "effect\\cru_grand cross.wav" {
		t.Fatalf("EF_GRANDCROSS sfx = %v", grand.sfx)
	}
	first := grand.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "ring_red" || first.totalCircleSides != 4 || first.circleSides != 4 || first.bottomSize != 0.7 || first.topSize != 0.7 || first.duplicate != 3 || first.duplicateDelay != 500*time.Millisecond || first.alphaMax != 0.1 || first.angleY != 45 {
		t.Fatalf("EF_GRANDCROSS center = %+v", first)
	}
	arc := grand.components[len(grand.components)-1]
	if arc.totalCircleSides != 20 || arc.circleSides != 5 || arc.bottomSize != 3 || arc.topSize != 3 || arc.posX != -3.5 || arc.posY != 3.5 || arc.angleY != -90 {
		t.Fatalf("EF_GRANDCROSS final arc = %+v", arc)
	}

	line, ok := worldEffectSpecForID(effectLineLink)
	if !ok || len(line.components) != 1 || line.duration != 100*time.Millisecond {
		t.Fatalf("EF_LINELINK spec = %+v ok=%t", line, ok)
	}
	if component := line.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/alpha_center.tga" || component.alphaMax != 0.5 || !component.fromSrc || !component.rotateToTarget || !component.rotateWithCamera || component.sizeStartX != effectTableSize(5) || component.sizeStartY != effectTableSize(50) || component.posZ != 1 {
		t.Fatalf("EF_LINELINK component = %+v", component)
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BOTTOM_VO", effectBottomVolcano, "ring_red"},
		{"EF_BOTTOM_DE", effectBottomDeluge, "ring_blue"},
		{"EF_BOTTOM_VI", effectBottomViolent, "ring_yellow"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != "PropertyGround" || component.funcAdapter != effectFuncPropertyGround || component.textureName != tc.texture || component.topSize != 3 || component.bottomSize != 1 || component.height != 2 || !component.repeat {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserActiveEffectsTwoFiftyToThreeHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSpearQuicken:   "EF_SPEARQUICKEN",
		effectDevotion:       "EF_DEVOTION",
		effectReflectShield:  "EF_REFLECTSHIELD",
		effectAbsorbSpirits:  "EF_ABSORBSPIRITS",
		effectSteelBody:      "EF_STEELBODY",
		effectFlameLauncher:  "EF_FLAMELAUNCHER",
		effectFrostWeapon:    "EF_FROSTWEAPON",
		effectLightningLoad:  "EF_LIGHTNINGLOADER",
		effectSeismicWeapon:  "EF_SEISMICWEAPON",
		effectGumgang2:       "EF_GUMGANG2",
		effectTeiHit1:        "EF_TEIHIT1",
		effectGumgang3:       "EF_GUMGANG3",
		effectTanji:          "EF_TANJI",
		effectTeiHit1X:       "EF_TEIHIT1X",
		effectChimto:         "EF_CHIMTO",
		effectStealCoin:      "EF_STEALCOIN",
		effectStripWeapon:    "EF_STRIPWEAPON",
		effectStripShield:    "EF_STRIPSHIELD",
		effectStripArmor:     "EF_STRIPARMOR",
		effectStripHelm:      "EF_STRIPHELM",
		effectChainCombo:     "EF_CHAINCOMBO",
		effectRogueCoin:      "EF_RG_COIN",
		effectBackStab:       "EF_BACKSTAP",
		effectTeiHit3:        "EF_TEIHIT3",
		effectBottomLullaby:  "EF_BOTTOM_LULLABY",
		effectBottomRichKim:  "EF_BOTTOM_RICHMANKIM",
		effectBottomChaos:    "EF_BOTTOM_ETERNALCHAOS",
		effectBottomDrum:     "EF_BOTTOM_DRUMBATTLEFIELD",
		effectBottomNibelung: "EF_BOTTOM_RINGNIBELUNGEN",
		effectBottomRoki:     "EF_BOTTOM_ROKISWEIL",
		effectBottomAbyss:    "EF_BOTTOM_INTOABYSS",
		effectBottomSieg:     "EF_BOTTOM_SIEGFRIED",
		effectBottomWhistle:  "EF_BOTTOM_WHISTLE",
		effectBottomSinX:     "EF_BOTTOM_ASSASSINCROSS",
		effectBottomBragi:    "EF_BOTTOM_POEMBRAGI",
		effectBottomApple:    "EF_BOTTOM_APPLEIDUN",
		effectBottomHumming:  "EF_BOTTOM_HUMMING",
		effectBottomForget:   "EF_BOTTOM_DONTFORGETME",
		effectBottomFortune:  "EF_BOTTOM_FORTUNEKISS",
		effectBottomService:  "EF_BOTTOM_SERVICEFORYOU",
		effectTalkFrostJoke:  "EF_TALK_FROSTJOKE",
		effectTalkScream:     "EF_TALK_SCREAM",
		effectPokJuk:         "EF_POKJUK",
		effectThrowItem:      "EF_THROWITEM",
		effectChemicalProt:   "EF_CHEMICALPROTECTION",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}

	for id, name := range map[int]string{
		effectBottomDissonance: "EF_BOTTOM_DISSONANCE",
		effectBottomUglyDance:  "EF_BOTTOM_UGLYDANCE",
	} {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeHundredToThreeFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectChemicalProt:  "EF_CHEMICALPROTECTION",
		effectDemonstration: "EF_DEMONSTRATION",
		effectChemical2:     "EF_CHEMICAL2",
		effectTeleportation: "EF_TELEPORTATION2",
		effectPharmacyOK:    "EF_PHARMACY_OK",
		effectPharmacyFail:  "EF_PHARMACY_FAIL",
		effectThrowItem3:    "EF_THROWITEM3",
		effectFirstAid:      "EF_FIRSTAID",
		effectLoud:          "EF_LOUD",
		effectHeal:          "EF_HEAL",
		effectHeal2:         "EF_HEAL2",
		effectExit2:         "EF_EXIT2",
		effectSafetyWall:    "EF_GLASSWALL2",
		effectReadyPortal:   "EF_READYPORTAL2",
		effectPortal:        "EF_PORTAL2",
		effectBottomMagnus:  "EF_BOTTOM_MAG",
		effectBottomSanc:    "EF_BOTTOM_SANC",
		effectHealOffensive: "EF_HEAL3",
		effectWarpZone2:     "EF_WARPZONE2",
		effectHeal4:         "EF_HEAL4",
		effectBeginAsura:    "EF_BEGINASURA",
		effectTripleAttack:  "EF_TRIPLEATTACK",
		effectHPTime:        "EF_HPTIME",
		effectSPTime:        "EF_SPTIME",
		effectMaple:         "EF_MAPLE",
		effectBlind:         "EF_BLIND",
		effectPoisonStatus:  "EF_POISON",
		effectGuard:         "EF_GUARD",
		effectJobLvUp50:     "EF_JOBLVUP50",
		effectMagnum2:       "EF_MAGNUM2",
		effectEntry2:        "EF_ENTRY2",
		effectColorPaper:    "EF_COLORPAPER",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeFiftyToFourHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectSoulBreaker:       "EF_SOULBREAKER",
		effectLevel99Aura1:      "EF_LEVEL99_4",
		effectFoodChocolate:     "EF_VALLENTINE",
		effectPressure:          "EF_PRESSURE",
		effectBash3D:            "EF_BASH3D",
		effectAuraBlade:         "EF_AURABLADE",
		effectRedBody:           "EF_REDBODY",
		effectLKConcentration:   "EF_LKCONCENTRATION",
		effectBottomGospel:      "EF_BOTTOM_GOSPEL",
		effectBaseLevelUp:       "EF_ANGEL",
		effectDeath:             "EF_DEVIL",
		effectDragonSmoke:       "EF_DRAGONSMOKE",
		effectBottomBasilica:    "EF_BOTTOM_BASILICA",
		effectHitLine2:          "EF_HITLINE2",
		effectBash3D2:           "EF_BASH3D2",
		effectEnergyDrain2:      "EF_ENERGYDRAIN2",
		effectTransBlueBody:     "EF_TRANSBLUEBODY",
		effectMagicCrasher:      "EF_MAGICCRASHER",
		effectLightBlade:        "EF_LIGHTBLADE",
		effectEnergyDrain3:      "EF_ENERGYDRAIN3",
		effectLineLink2:         "EF_LINELINK2",
		effectTrueSight:         "EF_TRUESIGHT",
		effectFalconAssault:     "EF_FALCONASSAULT",
		effectTripleAttack2:     "EF_TRIPLEATTACK2",
		effectPortal4:           "EF_PORTAL4",
		effectMeltdown:          "EF_MELTDOWN",
		effectCartBoost:         "EF_CARTBOOST",
		effectRejectSword:       "EF_REJECTSWORD",
		effectTripleAttack3:     "EF_TRIPLEATTACK3",
		effectMoonlit:           "EF_SPHEREWIND2",
		effectLevel99AuraMid:    "EF_LEVEL99_5",
		effectLevel99AuraBottom: "EF_LEVEL99_6",
		effectBash3D3:           "EF_BASH3D3",
		effectBash3D4:           "EF_BASH3D4",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourHundredToFourFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPortal5:       "EF_PORTAL5",
		effectMagicCrasher2: "EF_MAGICCRASHER2",
		effectBottomSpider:  "EF_BOTTOM_SPIDER",
		effectSoulBurn:      "EF_SOULBURN",
		effectSoulChange:    "EF_SOULCHANGE",
		effectSoulBreaker2:  "EF_SOULBREAKER2",
		effectBabyBody:      "EF_BABYBODY",
		effectBabyBody2:     "EF_BABYBODY2",
		effectGiantBody:     "EF_GIANTBODY",
		effectGiantBody2:    "EF_GIANTBODY2",
		effectQuakeBody:     "EF_QUAKEBODY",
		effectAssumptio2:    "EF_ASSUMPTIO2",
		effectStopEffect:    "EF_STOPEFFECT",
		effectJumpBody:      "EF_JUMPBODY",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourFiftyToFiveHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectDarkGrandCross: "EF_GRANDCROSS2",
		effectDarkSoulStrike: "EF_SOULSTRIKE2",
		effectDarkJupitelHit: "EF_YUFITEL2",
		effectNPCStop:        "EF_NPC_STOP",
		effectDarkCasting:    "EF_DARKCASTING",
		effectNPCPowerUp:     "EF_AGIUP",
		effectJumpKick:       "EF_JUMPKICK",
		effectBeginAsura1:    "EF_BEGINASURA1",
		effectBeginAsura2:    "EF_BEGINASURA2",
		effectBeginAsura3:    "EF_BEGINASURA3",
		effectBeginAsura4:    "EF_BEGINASURA4",
		effectBeginAsura5:    "EF_BEGINASURA5",
		effectBeginAsura6:    "EF_BEGINASURA6",
		effectBeginAsura7:    "EF_BEGINASURA7",
		effectMochi:          "EF_MOCHI",
		effectRamadan:        "EF_LAMADAN",
		effectEDP:            "EF_EDP",
		effectPreserve:       "EF_GUARD2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveHundredToFiveFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectCastSpin:      "EF_CASTSPIN",
		effectChookgi2:      "EF_CHOOKGI2",
		effectMapae:         "EF_MAPAE",
		effectItemPokJuk:    "EF_ITEMPOKJUK",
		effectValentine05:   "EF_05VAL",
		effectBeginAsura11:  "EF_BEGINASURA11",
		effectChemical2Dash: "EF_CHEMICAL2DASH",
		effectGroundSample:  "EF_GROUNDSAMPLE",
		effectCloud4:        "EF_CLOUD4",
		effectCloud5:        "EF_CLOUD5",
		effectBottomHermode: "EF_BOTTOM_HERMODE",
		effectItemFastDown:  "EF_ITEMFAST",
		effectTarotCard1:    "EF_TAROTCARD1",
		effectTarotCard2:    "EF_TAROTCARD2",
		effectTarotCard3:    "EF_TAROTCARD3",
		effectTarotCard4:    "EF_TAROTCARD4",
		effectTarotCard5:    "EF_TAROTCARD5",
		effectTarotCard6:    "EF_TAROTCARD6",
		effectTarotCard7:    "EF_TAROTCARD7",
		effectTarotCard8:    "EF_TAROTCARD8",
		effectTarotCard9:    "EF_TAROTCARD9",
		effectTarotCard10:   "EF_TAROTCARD10",
		effectTarotCard11:   "EF_TAROTCARD11",
		effectTarotCard12:   "EF_TAROTCARD12",
		effectTarotCard13:   "EF_TAROTCARD13",
		effectTarotCard14:   "EF_TAROTCARD14",
		effectAcidDemon:     "EF_ACIDDEMON",
		effectHated:         "EF_HATED",
		effectStin:          "EF_STIN",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveFiftyToSixHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectStin2:       "EF_STIN2",
		effectStin3:       "EF_STIN3",
		effectScreenQuake: "EF_SCREEN_QUAKE",
		effectHfliMoon1:   "EF_HFLIMOON1",
		effectHfliMoon2:   "EF_HFLIMOON2",
		effectHfliMoon3:   "EF_HFLIMOON3",
		effectHoUp:        "EF_HO_UP",
		effectHamiDefence: "EF_HAMIDEFENCE",
		effectHamiCastle:  "EF_HAMICASTLE",
		effectHamiBlood:   "EF_HAMIBLOOD",
		effectItemThunder: "EF_ITEM_THUNDER",
		effectItemCloud:   "EF_ITEM_CLOUD",
		effectItemCurse:   "EF_ITEM_CURSE",
		effectItemZZZ:     "EF_ITEM_ZZZ",
		effectItemRain:    "EF_ITEM_RAIN",
		effectM01:         "EF_M01",
		effectM02:         "EF_M02",
		effectM03:         "EF_M03",
		effectM04:         "EF_M04",
		effectM05:         "EF_M05",
		effectM06:         "EF_M06",
		effectM07:         "EF_M07",
		effectKaizel:      "EF_KAIZEL",
		effectCloud6:      "EF_CLOUD6",
		effectStatFoodSTR: "EF_FOOD01",
		effectStatFoodINT: "EF_FOOD02",
		effectStatFoodVIT: "EF_FOOD03",
		effectStatFoodAGI: "EF_FOOD04",
		effectStatFoodDEX: "EF_FOOD05",
		effectStatFoodLUK: "EF_FOOD06",
		effectThrowItem6:  "EF_THROWITEM6",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixHundredToSixFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectThrowItem6:    "EF_THROWITEM6",
		effectFireHit2:      "EF_FIREHIT2",
		effectNPCStop2:      "EF_NPC_STOP2",
		effectFVoice:        "EF_FVOICE",
		effectWink:          "EF_WINK",
		effectCookingOK:     "EF_COOKING_OK",
		effectCookingFail:   "EF_COOKING_FAIL",
		effectHapgyeok:      "EF_HAPGYEOK",
		effectThrowItem7:    "EF_THROWITEM7",
		effectThrowItem8:    "EF_THROWITEM8",
		effectThrowItem9:    "EF_THROWITEM9",
		effectThrowItem10:   "EF_THROWITEM10",
		effectKouenka:       "EF_KOUENKA",
		effectHyousensou:    "EF_HYOUSENSOU",
		effectStin4:         "EF_STIN4",
		effectThunderStorm2: "EF_THUNDERSTORM2",
		effectRGCoin3:       "EF_RG_COIN3",
		effectBash3D5:       "EF_BASH3D5",
		effectChookgi3:      "EF_CHOOKGI3",
		effectKirikage:      "EF_KIRIKAGE",
		effectTatami:        "EF_TATAMI",
		effectKasumikiri:    "EF_KASUMIKIRI",
		effectIssen:         "EF_ISSEN",
		effectKaen:          "EF_KAEN",
		effectBaku:          "EF_BAKU",
		effectHyousyouraku:  "EF_HYOUSYOURAKU",
		effectDesperado:     "EF_DESPERADO",
		effectLightningS:    "EF_LIGHTNING_S",
		effectBlindS:        "EF_BLIND_S",
		effectPoisonS:       "EF_POISON_S",
		effectFreezingS:     "EF_FREEZING_S",
		effectFlareS:        "EF_FLARE_S",
		effectRapidShower:   "EF_RAPIDSHOWER",
		effectMagicalBullet: "EF_MAGICALBULLET",
		effectSpreadAttack:  "EF_SPREADATTACK",
		effectTrackCasting:  "EF_TRACKCASTING",
		effectTracking:      "EF_TRACKING",
		effectTripleAction:  "EF_TRIPLEACTION",
		effectBullseye:      "EF_BULLSEYE",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixFiftyToSevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectNPCEarthquake:  "EF_NPC_EARTHQUAKE",
		effectDragonFear:     "EF_DRAGONFEAR",
		effectWideBleeding:   "EF_BLEEDING",
		effectWideConfuse:    "EF_WIDECONFUSE",
		effectBottomRunner:   "EF_BOTTOM_RUNNER",
		effectBottomTransfer: "EF_BOTTOM_TRANSFER",
		effectBottomEvilLand: "EF_BOTTOM_EVILLAND",
		effectGuard3:         "EF_GUARD3",
		effectCriticalWound:  "EF_CRITICALWOUND",
		effectFirecracker2:   "EF_POK_LOVE",
		effectFirecracker3:   "EF_POK_WHITE",
		effectFirecracker4:   "EF_POK_VALEN",
		effectFirecracker5:   "EF_POK_BIRTH",
		effectFirecracker6:   "EF_POK_CHRISTMAS",
		effectCloud7:         "EF_CLOUD7",
		effectCloud8:         "EF_CLOUD8",
		effectFlowerLeaf:     "EF_FLOWERLEAF",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenHundredToSevenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectItem315:          "EF_ITEM315",
		effectItem316:          "EF_ITEM316",
		effectItem317:          "EF_ITEM317",
		effectStormMin:         "EF_STORM_MIN",
		effectFirecracker7:     "EF_POK_JAP",
		effectBottomBlue:       "EF_BOTTOM_BLUE",
		effectBottomBlue2:      "EF_BOTTOM_BLUE2",
		effectChristmasCarol:   "EF_WEWISH",
		effectFirePillarOn2:    "EF_FIREPILLARON2",
		effectForestLight5:     "EF_FORESTLIGHT5",
		effectAdoramus:         "EF_ADO_STR",
		effectIgnitionBreak:    "EF_IGN_STR",
		effectFrostMisty:       "EF_FROSTMYSTY",
		effectCrimsonRock:      "EF_CRIMSON_STR",
		effectHellInferno:      "EF_HELL_STR",
		effectMarshOfAbyss:     "EF_SPR_MASH",
		effectDragonHowling:    "EF_DHOWL_STR",
		effectEarthWall:        "EF_EARTHWALL",
		effectChainLightning:   "EF_CHAINL_STR",
		effectAimedBolt:        "EF_AIMED_STR",
		effectArrowStorm:       "EF_ARROWSTORM_STR",
		effectLaulamus:         "EF_LAULAMUS_STR",
		effectLauagnus:         "EF_LAUAGNUS_STR",
		effectMillenniumShield: "EF_MILSHIELD_STR",
		effectConcentration2:   "EF_CONCENTRATION2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenFiftyToEightHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectGlassWall3:     "EF_GLASSWALL3",
		effectBerserkPotion2: "EF_POTION_BERSERK2",
		effectRolling1:       "EF_ROLLING1",
		effectRolling2:       "EF_ROLLING2",
		effectRolling3:       "EF_ROLLING3",
		effectRolling4:       "EF_ROLLING4",
		effectRolling5:       "EF_ROLLING5",
		effectRolling6:       "EF_ROLLING6",
		effectRolling7:       "EF_ROLLING7",
		effectRolling8:       "EF_ROLLING8",
		effectRolling9:       "EF_ROLLING9",
		effectRolling10:      "EF_ROLLING10",
		effectCastSpin2:      "EF_CASTSPIN2",
		effectCrashAxe:       "EF_CRASHAXE",
		effectStasis:         "EF_STASIS",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightHundredToEightFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBottomBasilica2:  "EF_BOTTOM_BASILICA2",
		effectRecognized:       "EF_RECOGNIZED",
		effectTetra:            "EF_TETRA",
		effectTetraCasting:     "EF_TETRACASTING",
		effectStretch:          "EF_STRETCH",
		effectEnervation:       "EF_ENERVATION",
		effectEnervation2:      "EF_ENERVATION2",
		effectEnervation3:      "EF_ENERVATION3",
		effectEnervation4:      "EF_ENERVATION4",
		effectEnervation5:      "EF_ENERVATION5",
		effectEnervation6:      "EF_ENERVATION6",
		effectBottomManhole:    "EF_BOTTOM_MANHOLE",
		effectManhole:          "EF_MANHOLE",
		effectForestLight6:     "EF_FORESTLIGHT6",
		effectBottomAni:        "EF_BOTTOM_ANI",
		effectBottomMaelstrom:  "EF_BOTTOM_MAELSTROM",
		effectBottomBloodyLust: "EF_BOTTOM_BLOODYLUST",
		effectHealN:            "EF_HEAL_N",
		effectChookgiN:         "EF_CHOOKGI_N",
		effectDance1:           "EF_DANCE1",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightFiftyToNineHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBotReverb:    "EF_BOT_REVERB",
		effectRainParticle: "EF_RAIN_PARTICLE",
		effectChemicalV2:   "EF_CHEMICAL_V2",
		effectBotReverb2:   "EF_BOT_REVERB2",
		effectCirclePower2: "EF_CIRCLEPOWER2",
		effectSecra2:       "EF_SECRA2",
		effectSprPlant2:    "EF_SPR_PLANT2",
		effectSprPlant3:    "EF_SPR_PLANT3",
		effectSprPlant4:    "EF_SPR_PLANT4",
		effectSprPlant5:    "EF_SPR_PLANT5",
		effectSprPlant6:    "EF_SPR_PLANT6",
		effectSprPlant7:    "EF_SPR_PLANT7",
		effectSprPlant8:    "EF_SPR_PLANT8",
		effectHeartAsura:   "EF_HEARTASURA",
		effectGlassWall4:   "EF_GLASSWALL4",
		effectBash3D6:      "EF_BASH3D6",
		effectElectric4:    "EF_ELECTRIC4",
		effectTeiHit1T:     "EF_TEIHIT1T",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineHundredToNineFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPressure2:    "EF_PRESSURE2",
		effectPrimeCharge2: "EF_PRIMECHARGE2",
		effectPrimeCharge3: "EF_PRIMECHARGE3",
		effectPrimeCharge4: "EF_PRIMECHARGE4",
		effectFireWall2:    "EF_FIREWALL2",
		effectSprPlant10:   "EF_SPR_PLANT10",
		effectShockwave2:   "EF_SHOCKWAVE2",
		effectColdThrow2:   "EF_COLDTHROW2",
		effectDemonicFire4: "EF_DEMONICFIRE4",
		effectPressure3:    "EF_PRESSURE3",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineFiftyToOneThousandHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectPoisonMist:     "EF_POISON_MIST",
		effectEraserCutter:   "EF_ERASER_CUTTER",
		effectLavaSlide:      "EF_LAVA_SLIDE",
		effectSonicClaw:      "EF_SONIC_CLAW",
		effectTinderBreaker:  "EF_TINDER_BREAKER",
		effectMidnightFrenzy: "EF_MIDNIGHT_FRENZY",
		effectVolcanicAsh:    "EF_VOLCANIC_ASH",
		effectRWC2011:        "EF_2011RWC",
		effectRWC2011Two:     "EF_2011RWC2",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsOneThousandToTenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectRunMakeOK:        "EF_RUN_MAKE_OK",
		effectRunMakeFailure:   "EF_RUN_MAKE_FAILURE",
		effectMIResultMakeOK:   "EF_MIRESULT_MAKE_OK",
		effectMIResultMakeFail: "EF_MIRESULT_MAKE_FAIL",
		effectAllRayProtect:    "EF_ALL_RAY_OF_PROTECTION",
		effectVenomFog:         "EF_VENOMFOG",
		effectDustStorm:        "EF_DUSTSTORM",
		effectDanceBladeAtk:    "EF_DANCE_BLADE_ATK",
		effectInvincibleOff2:   "EF_INVINCIBLEOFF2",
		effectDeathSummon:      "EF_DEATHSUMMON",
		effectGCDarkCrow:       "EF_GC_DARKCROW",
		effectAllFullThrottle:  "EF_ALL_FULL_THROTTLE",
		effectSRFlashCombo:     "EF_SR_FLASHCOMBO",
		effectRKLuxAnima:       "EF_RK_LUXANIMA",
		effectSOElemShield:     "EF_SO_ELEMENTAL_SHIELD",
		effectABOffertorium:    "EF_AB_OFFERTORIUM",
		effectWLTelekinesis:    "EF_WL_TELEKINESIS_INTENSE",
		effectGNIllusionDoping: "EF_GN_ILLUSIONDOPING",
		effectNCMagmaEruption:  "EF_NC_MAGMA_ERUPTION",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsTenFiftyToElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectNPCChill:        "EF_NPC_CHILL",
		effectOffertoriumRing: "EF_AB_OFFERTORIUM_RING",
		effectHammerOfGod:     "EF_HAMMER_OF_GOD",
		effectAchComplete:     "EF_ACH_COMPLETE",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsPostElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		effectBodyColor:        "EffectBodyColor",
		effectBakuretsuHadou:   "EF_BAKURETSU_HADOU",
		dropEffectPink:         "DROPEFFECT_PINK",
		dropEffectYellow:       "DROPEFFECT_YELLOW",
		dropEffectPurple:       "DROPEFFECT_PURPLE",
		effectDigitalSpace:     "EF_DIGITAL_SPACE",
		dropEffectBlue:         "DROPEFFECT_BLUE",
		dropEffectGreen:        "DROPEFFECT_GREEN",
		dropEffectRed:          "DROPEFFECT_RED",
		effectNewSuccess:       "EF_NEW_SUCCESS",
		effectNewFailure:       "EF_NEW_FAILURE",
		effectNewIntro:         "EF_NEW_INTRO",
		effectEnchantYellow:    "EF_UI_ENCHANT_INTRO_YELLOW",
		effectEnchantSuccess:   "EF_UI_ENCHANT_SUCCESS",
		effectEnchantFail:      "EF_UI_ENCHANT_FAIL",
		effectEnchantBlue:      "EF_UI_ENCHANT_INTRO_BLUE",
		effectEnchantUpSuccess: "EF_UI_ENCHANT_UP_SUCCESS",
		effectEnchantUpFail:    "EF_UI_ENCHANT_UP_FAIL",
		effectEnchantGreen:     "EF_UI_ENCHANT_INTRO_GREEN",
		effectEnchantResetOK:   "EF_UI_ENCHANT_RESET_SUCCESS",
		effectEnchantResetFail: "EF_UI_ENCHANT_RESET_FAIL",
	}
	for id, name := range active {
		if _, ok := worldEffectSpecForID(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserSimpleEffectsTwoFiftyToThreeHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_DEVOTION", effectDevotion, "devotion", "", true, false},
		{"EF_FLAMELAUNCHER", effectFlameLauncher, "enc_fire", "_enemy_hit_wind1.wav", true, false},
		{"EF_FROSTWEAPON", effectFrostWeapon, "enc_ice", "_enemy_hit_wind1.wav", true, false},
		{"EF_LIGHTNINGLOADER", effectLightningLoad, "enc_wind", "effect\\_enemy_hit_wind1.wav", true, false},
		{"EF_SEISMICWEAPON", effectSeismicWeapon, "enc_earth", "_enemy_hit_wind1.wav", true, false},
		{"EF_STEALCOIN", effectStealCoin, "steal_coin", "", true, false},
		{"EF_STRIPWEAPON", effectStripWeapon, "strip_weapon", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPSHIELD", effectStripShield, "strip_shield", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPARMOR", effectStripArmor, "strip_armor", "effect\\t_벗김.wav", true, false},
		{"EF_STRIPHELM", effectStripHelm, "strip_helm", "effect\\t_벗김.wav", true, false},
		{"EF_CHAINCOMBO", effectChainCombo, "연환", "effect\\mon_연환.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STEELBODY", effectSteelBody, "effect\\mon_금강불괴.wav"},
		{"EF_CHIMTO", effectChimto, "effect\\mon_침투경.wav"},
		{"EF_BACKSTAP", effectBackStab, "effect\\rog_back stap.wav"},
		{"EF_BOTTOM_LULLABY", effectBottomLullaby, "effect\\자장가.wav"},
		{"EF_BOTTOM_RICHMANKIM", effectBottomRichKim, "effect\\김서방돈.wav"},
		{"EF_BOTTOM_ETERNALCHAOS", effectBottomChaos, "effect\\영원의 혼돈.wav"},
		{"EF_BOTTOM_DRUMBATTLEFIELD", effectBottomDrum, "effect\\전장의.wav"},
		{"EF_BOTTOM_RINGNIBELUNGEN", effectBottomNibelung, "effect\\니벨룽겐의 반지.wav"},
		{"EF_BOTTOM_ROKISWEIL", effectBottomRoki, "effect\\로키.wav"},
		{"EF_BOTTOM_INTOABYSS", effectBottomAbyss, "effect\\심연속으로.wav"},
		{"EF_BOTTOM_SIEGFRIED", effectBottomSieg, "effect\\불사신.wav"},
		{"EF_BOTTOM_WHISTLE", effectBottomWhistle, "effect\\달빛세레나데.wav"},
		{"EF_BOTTOM_ASSASSINCROSS", effectBottomSinX, "effect\\석양의 어쌔신.wav"},
		{"EF_BOTTOM_POEMBRAGI", effectBottomBragi, "effect\\브라기의 시.wav"},
		{"EF_BOTTOM_APPLEIDUN", effectBottomApple, "effect\\이둔의 사과.wav"},
		{"EF_BOTTOM_HUMMING", effectBottomHumming, "effect\\흥얼거림.wav"},
		{"EF_BOTTOM_DONTFORGETME", effectBottomForget, "effect\\나를잊지말아요.wav"},
		{"EF_BOTTOM_FORTUNEKISS", effectBottomFortune, "effect\\행운의.wav"},
		{"EF_BOTTOM_SERVICEFORYOU", effectBottomService, "effect\\당신을 위한 서비스.wav"},
		{"EF_CHEMICALPROTECTION", effectChemicalProt, "apocalips_attack.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestBottomSongGroundEffectsMatchReferenceRows(t *testing.T) {
	for _, tc := range []struct {
		name        string
		id          int
		texture     string
		tint        color.RGBA
		textureSize float64
	}{
		{"277_ground", effectBottomDissonanceGround, "effect/lens_w.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 0.5},
		{"278_ground", effectBottomLullabyGround, "effect/zz.bmp", color.RGBA{R: 237, G: 158, B: 255, A: 13}, 0.5},
		{"279_ground", effectBottomRichKimGround, "effect/pocket.bmp", color.RGBA{R: 252, G: 199, B: 199, A: 13}, 0.5},
		{"280_ground", effectBottomChaosGround, "effect/lens_g.bmp", color.RGBA{R: 128, G: 255, B: 194, A: 13}, 0.5},
		{"281_ground", effectBottomDrumGround, "effect/melody_b.bmp", color.RGBA{R: 237, G: 101, B: 252, A: 13}, 0.5},
		{"282_ground", effectBottomNibelungGround, "effect/twirl.bmp", color.RGBA{R: 28, G: 236, B: 255, A: 13}, 0.5},
		{"283_ground", effectBottomRokiGround, "effect/safeline.bmp", color.RGBA{R: 220, G: 101, B: 252, A: 13}, 0.5},
		{"284_ground", effectBottomAbyssGround, "effect/bluegemstone.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 1},
		{"285_ground", effectBottomSiegGround, "effect/lens_b.bmp", color.RGBA{R: 72, G: 59, B: 255, A: 13}, 0.5},
		{"286_ground", effectBottomWhistleGround, "effect/melody_b.bmp", color.RGBA{R: 255, G: 192, B: 203, A: 13}, 0.5},
		{"287_ground", effectBottomSinXGround, "effect/lens_r.bmp", color.RGBA{R: 255, G: 204, B: 217, A: 102}, 0.5},
		{"288_ground", effectBottomBragiGround, "effect/spell_01.bmp", color.RGBA{}, 0.5},
		{"289_ground", effectBottomAppleGround, "effect/idun_apple.bmp", color.RGBA{R: 255, G: 255, B: 0, A: 13}, 1},
		{"290_ground", effectBottomUglyDanceGround, "effect/lens_w.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 0.5},
		{"291_ground", effectBottomHummingGround, "effect/melody_a.bmp", color.RGBA{R: 230, G: 209, B: 209, A: 13}, 0.5},
		{"292_ground", effectBottomForgetGround, "effect/lens_g.bmp", color.RGBA{R: 28, G: 255, B: 115, A: 13}, 0.5},
		{"293_ground", effectBottomFortuneGround, "effect/kiss.bmp", color.RGBA{R: 252, G: 111, B: 101, A: 13}, 2.5},
		{"294_ground", effectBottomServiceGround, "effect/safeline.bmp", color.RGBA{R: 255, G: 128, B: 183, A: 13}, 0.5},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		wantComponents := 2
		if tc.tint.A == 0 {
			wantComponents = 1
		}
		if !ok || spec.duration != 1500*time.Millisecond || len(spec.sfx) != 0 || len(spec.components) != wantComponents {
			t.Fatalf("%s spec = %+v ok=%t, want visible song ground", tc.name, spec, ok)
		}
		textureIndex := 0
		if tc.tint.A > 0 {
			tile := spec.components[0]
			if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != tc.tint || tile.sizeStart != 1 || !tile.renderBefore || tile.attachedEntity {
				t.Fatalf("%s flat tile = %+v", tc.name, tile)
			}
			textureIndex = 1
		}
		wantSize := tc.textureSize * 2
		texture := spec.components[textureIndex]
		if texture.kind != effectComponentFUNC || texture.funcName != "GroundTexture" || texture.funcAdapter != effectFuncGroundTexture || texture.textureFile != tc.texture || texture.duration != 1500*time.Millisecond || texture.sizeStart != wantSize || texture.sizeEnd != wantSize || texture.alphaMax != 0.7 || texture.angleStart != math.Pi || texture.posZ != 0.2 || texture.posZEnd != 0.6 || !texture.blendAdditive || !texture.renderBefore || texture.attachedEntity {
			t.Fatalf("%s texture = %+v", tc.name, texture)
		}
	}
}

func TestGroundTextureEffectHoverMovesBetweenZRange(t *testing.T) {
	start := time.Unix(1, 0)
	effect := worldEffect{effectID: effectBottomHummingGround, actorID: 172, starts: start}
	component := worldEffectComponent{posZ: 0.2, posZEnd: 0.6}

	first := groundTextureZOffset(component, effect, 1, start)
	later := groundTextureZOffset(component, effect, 1, start.Add(3*time.Second))
	for _, got := range []float64{first, later} {
		if got < 0.2 || got > 0.6 {
			t.Fatalf("hover z = %.3f, want inside 0.2..0.6", got)
		}
	}
	if math.Abs(first-later) < 0.001 {
		t.Fatalf("hover z did not move: %.3f then %.3f", first, later)
	}
}

func TestRobrowserSpecialEffectsTwoFiftyToThreeHundredMatchTableRows(t *testing.T) {
	reflectShield, ok := worldEffectSpecForID(effectReflectShield)
	if !ok || len(reflectShield.components) != 1 || reflectShield.duration != 3*time.Second {
		t.Fatalf("EF_REFLECTSHIELD spec = %+v ok=%t", reflectShield, ok)
	}
	if component := reflectShield.components[0]; component.kind != effectComponentCylinder || component.textureName != "ring_yellow" || component.duration != 3*time.Second || component.alphaMax != 0.6 || component.animation != 1 || component.blendMode != 8 || component.bottomSize != 1.5 || component.topSize != 1.5 || component.height != 10 || !component.rotate || !component.fade || !component.attachedEntity {
		t.Fatalf("EF_REFLECTSHIELD component = %+v", component)
	}

	absorb, ok := worldEffectSpecForID(effectAbsorbSpirits)
	if !ok || len(absorb.components) != 6 || absorb.duration != 1890*time.Millisecond {
		t.Fatalf("EF_ABSORBSPIRITS spec = %+v ok=%t", absorb, ok)
	}
	if len(absorb.sfx) != 1 || absorb.sfx[0] != "effect\\흡기.wav" {
		t.Fatalf("EF_ABSORBSPIRITS sfx = %v", absorb.sfx)
	}
	first, second, third := absorb.components[0], absorb.components[1], absorb.components[2]
	for i, component := range []worldEffectComponent{first, second, third} {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_ABSORBSPIRITS cylinder %d = %+v", i, component)
		}
		if component.color.R != 77 || component.color.G != 77 || component.color.B != 255 {
			t.Fatalf("EF_ABSORBSPIRITS cylinder %d color = %+v", i, component.color)
		}
	}
	if first.bottomSize != 1.1 || first.topSize != 1.1 || first.height != 15 || second.bottomSize != 1 || second.topSize != 1 || second.height != 13 || third.bottomSize != 1.1 || third.topSize != 3 || third.height != 2 {
		t.Fatalf("EF_ABSORBSPIRITS cylinder sizes = %+v %+v %+v", first, second, third)
	}
	sparkA, sparkB, sparkC := absorb.components[3], absorb.components[4], absorb.components[5]
	if sparkA.kind != effectComponent3D || sparkA.textureFile != "effect/pok3.tga" || sparkA.duration != 1500*time.Millisecond || sparkA.duplicate != 4 || sparkA.duplicateDelay != 10*time.Millisecond || sparkA.posXRand != 1.2 || sparkA.posYRand != 1.2 || sparkA.posZEndRand != 1 || sparkA.posZEndMiddle != 8 || !sparkA.sparkling || sparkA.sparkNumber != 2 {
		t.Fatalf("EF_ABSORBSPIRITS first particle = %+v", sparkA)
	}
	if sparkB.duration != 1300*time.Millisecond || sparkB.delay != 400*time.Millisecond || sparkB.duplicate != 20 || sparkB.posXRand != 1.5 || sparkB.posYRand != 1.5 || sparkB.posZEndRand != 3 || sparkB.posZEndMiddle != 6 || !sparkB.sparkling || sparkB.sparkNumber != 2 {
		t.Fatalf("EF_ABSORBSPIRITS second particle = %+v", sparkB)
	}
	if sparkC.duration != 1100*time.Millisecond || sparkC.delay != 200*time.Millisecond || sparkC.duplicate != 10 || sparkC.duplicateDelay != 50*time.Millisecond || sparkC.posXRand != 1 || sparkC.posYRand != 1 || sparkC.posZEnd != 6 || sparkC.posZStartRand != 1 || sparkC.sparkling {
		t.Fatalf("EF_ABSORBSPIRITS third particle = %+v", sparkC)
	}

	for _, tc := range []struct {
		name     string
		id       int
		duration time.Duration
		alpha    float64
		bottom   float64
		top      float64
		wav      string
	}{
		{"EF_GUMGANG2", effectGumgang2, 1500 * time.Millisecond, 0.5, 2, 5, "effect\\mon_폭기.wav"},
		{"EF_GUMGANG3", effectGumgang3, 1000 * time.Millisecond, 0.3, 3, 6, ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != tc.duration+300*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentCylinder || component.textureName != "ring_yellow" || component.duration != tc.duration || component.alphaMax != tc.alpha || component.animation != 4 || component.blendMode != 8 || component.blendAdditive || component.duplicate != 4 || component.duplicateDelay != 100*time.Millisecond || component.bottomSize != tc.bottom || component.topSize != tc.top || component.height != 2 || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		texture   string
		wav       string
		duplicate int
		delay     time.Duration
		blueRaid  bool
	}{
		{"EF_TEIHIT1", effectTeiHit1, "effect/alpha_center.tga", "effect\\mon_폭기.wav", 12, 250 * time.Millisecond, false},
		{"EF_TEIHIT1X", effectTeiHit1X, "effect/lens1.tga", "effect\\mon_아수라 패황권.wav", 24, 100 * time.Millisecond, false},
		{"EF_TEIHIT3", effectTeiHit3, "effect/lens1.tga", "", 20, 100 * time.Millisecond, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 550*time.Millisecond+tc.delay {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.duration != 550*time.Millisecond || component.delay != tc.delay || component.duplicate != tc.duplicate || component.alphaMax != 0.8 || !component.fadeIn || !component.fadeOut || component.posXEndRand != 40 || component.posYEndRand != 40 || component.sizeStartX != effectTableSize(10) || component.sizeStartY != effectTableSize(150) || component.sizeEndX != effectTableSize(10) || component.sizeEndY != effectTableSize(150) || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity || !component.overlay || !component.rotateToTarget || !component.rotateWithCamera {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.blueRaid {
			if component.color.R != 26 || component.color.G != 26 || component.color.B != 255 {
				t.Fatalf("%s color = %+v", tc.name, component.color)
			}
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	tanji, ok := worldEffectSpecForID(effectTanji)
	if !ok || len(tanji.components) != 1 || tanji.duration != 150*time.Millisecond {
		t.Fatalf("EF_TANJI spec = %+v ok=%t", tanji, ok)
	}
	if len(tanji.sfx) != 1 || tanji.sfx[0] != "effect\\mon_탄지신통.wav" {
		t.Fatalf("EF_TANJI sfx = %v", tanji.sfx)
	}
	if component := tanji.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/blue_ivy.bmp" || component.duration != 150*time.Millisecond || component.alphaMax != 1 || component.blendMode != 2 || !component.blendAdditive || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || component.angleStart != 90 || component.angleEnd != 90 || component.posZ != 1 || component.sizeStart != effectTableSize(50) || !component.attachedEntity {
		t.Fatalf("EF_TANJI component = %+v", component)
	}

	coin, ok := worldEffectSpecForID(effectRogueCoin)
	if !ok || len(coin.components) != 1 || coin.duration != 2950*time.Millisecond {
		t.Fatalf("EF_RG_COIN spec = %+v ok=%t", coin, ok)
	}
	if len(coin.sfx) != 1 || coin.sfx[0] != "effect\\rog_steal coin.wav" {
		t.Fatalf("EF_RG_COIN sfx = %v", coin.sfx)
	}
	if component := coin.components[0]; component.kind != effectComponent2D || component.textureFile != "effect/coin_a.bmp" || component.duration != 1500*time.Millisecond || component.duplicate != 30 || component.duplicateDelay != 50*time.Millisecond || component.alphaMax != 0.8 || !component.fadeOut || component.posXEndRand != 10 || component.posYEndRand != 10 || component.posZ != 2 || component.sizeStart != effectTableSize(20) || component.blendMode != 2 || !component.blendAdditive || !component.overlay || !component.rotateToTarget || !component.attachedEntity {
		t.Fatalf("EF_RG_COIN component = %+v", component)
	}

	for _, tc := range []struct {
		name     string
		id       int
		funcName string
	}{
		{"EF_TALK_FROSTJOKE", effectTalkFrostJoke, "FrostJokeTalk"},
		{"EF_TALK_SCREAM", effectTalkScream, "ScreamTalk"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	throwItem, ok := worldEffectSpecForID(effectThrowItem)
	if !ok || len(throwItem.components) != 1 || throwItem.duration != 300*time.Millisecond {
		t.Fatalf("EF_THROWITEM spec = %+v ok=%t", throwItem, ok)
	}
	if component := throwItem.components[0]; component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/염산병.bmp" || component.duration != 300*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 360 || component.posZ != 1 || component.sizeStart != effectTableSize(30) || !component.attachedEntity {
		t.Fatalf("EF_THROWITEM component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	demo, ok := worldEffectSpecForID(effectDemonstration)
	if !ok || len(demo.components) != 1 {
		t.Fatalf("EF_DEMONSTRATION spec = %+v ok=%t", demo, ok)
	}
	if component := demo.components[0]; component.kind != effectComponentSPR || component.spriteFile != "데몬스트레이션" || component.attachedEntity {
		t.Fatalf("EF_DEMONSTRATION component = %+v", component)
	}

	job, ok := worldEffectSpecForID(effectJobLvUp50)
	if !ok || len(job.components) != 1 {
		t.Fatalf("EF_JOBLVUP50 spec = %+v ok=%t", job, ok)
	}
	if component := job.components[0]; component.kind != effectComponentSTR || component.strFile != "joblvup" || !component.attachedEntity {
		t.Fatalf("EF_JOBLVUP50 component = %+v", component)
	}

	colorPaper, ok := worldEffectSpecForID(effectColorPaper)
	if !ok || len(colorPaper.components) != 0 || colorPaper.duration != 500*time.Millisecond {
		t.Fatalf("EF_COLORPAPER spec = %+v ok=%t", colorPaper, ok)
	}
	if len(colorPaper.sfx) != 1 || colorPaper.sfx[0] != "effect\\wedding.wav" {
		t.Fatalf("EF_COLORPAPER sfx = %#v", colorPaper.sfx)
	}

	for _, tc := range []struct {
		name   string
		id     int
		sfx    []string
		delays []time.Duration
	}{
		{"EF_TRIPLEATTACK", effectTripleAttack, []string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"}, []time.Duration{0, 200 * time.Millisecond, 400 * time.Millisecond}},
		{"EF_MAGNUM2", effectMagnum2, []string{"permeter_attack.wav", "effect\\ef_magnumbreak.wav"}, []time.Duration{0, 300 * time.Millisecond}},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if !reflect.DeepEqual(spec.sfx, tc.sfx) || !reflect.DeepEqual(spec.sfxDelays, tc.delays) {
			t.Fatalf("%s sound = %v delays %v", tc.name, spec.sfx, spec.sfxDelays)
		}
	}

	chemical, ok := worldEffectSpecForID(effectChemical2)
	if !ok || len(chemical.components) != 1 || chemical.duration != 500*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 spec = %+v ok=%t", chemical, ok)
	}
	if chemical.cameraShake != 200*time.Millisecond || chemical.cameraShakeDelay != 132*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 camera shake = delay %s duration %s", chemical.cameraShakeDelay, chemical.cameraShake)
	}
	if component := chemical.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
		t.Fatalf("EF_CHEMICAL2 component = %+v", component)
	}

	blind, ok := worldEffectSpecForID(effectBlind)
	if !ok || len(blind.components) != 1 || blind.duration != 500*time.Millisecond {
		t.Fatalf("EF_BLIND spec = %+v ok=%t", blind, ok)
	}
	if len(blind.sfx) != 1 || blind.sfx[0] != "_blind.wav" {
		t.Fatalf("EF_BLIND sfx = %#v", blind.sfx)
	}
	if component := blind.components[0]; component.kind != effectComponentFUNC || component.funcName != "Blind" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_BLIND component = %+v", component)
	}

	poison, ok := worldEffectSpecForID(effectPoisonStatus)
	if !ok || len(poison.components) != 1 || poison.duration != 500*time.Millisecond {
		t.Fatalf("EF_POISON spec = %+v ok=%t", poison, ok)
	}
	if component := poison.components[0]; component.kind != effectComponentFUNC || component.funcName != "Poison" || component.funcAdapter != effectFuncUnknown || component.attachedEntity {
		t.Fatalf("EF_POISON component = %+v", component)
	}
}

func TestRobrowserPortalAndGroundEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	exit, ok := worldEffectSpecForID(effectExit2)
	if !ok || len(exit.components) != 3 || exit.duration != 1500*time.Millisecond {
		t.Fatalf("EF_EXIT2 spec = %+v ok=%t", exit, ok)
	}
	if len(exit.sfx) != 1 || exit.sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_EXIT2 sfx = %#v", exit.sfx)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.3, 0.3, 35},
		{0.4, 0.6, 23},
		{0.5, 0.7, 5},
	} {
		component := exit.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_EXIT2 component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height {
			t.Fatalf("EF_EXIT2 component %d size = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 128, G: 128, B: 255, A: 255}) {
			t.Fatalf("EF_EXIT2 component %d color = %+v", i, component.color)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		color   color.RGBA
		alpha   float64
		height  float64
	}{
		{"EF_BOTTOM_MAG", effectBottomMagnus, "ring_red", color.RGBA{}, 0.2, 5},
		{"EF_BOTTOM_SANC", effectBottomSanc, "magic_green", color.RGBA{R: 128, G: 230, B: 128, A: 255}, 0.3, 2},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 || spec.duration != 50*time.Second {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		first, second := spec.components[0], spec.components[1]
		for i, component := range []worldEffectComponent{first, second} {
			if component.kind != effectComponentCylinder || component.textureName != tc.texture || component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != 0.7 || component.topSize != 0.7 || component.height != tc.height || component.angleY != 45 || component.blendMode != 2 || !component.blendAdditive || component.rotate || !component.attachedEntity {
				t.Fatalf("%s component %d = %+v", tc.name, i, component)
			}
			if component.color != tc.color {
				t.Fatalf("%s component %d color = %+v", tc.name, i, component.color)
			}
		}
		if first.duration != 50*time.Second || first.alphaMax != tc.alpha || first.fade || first.repeat || first.animation != 0 {
			t.Fatalf("%s first cylinder = %+v", tc.name, first)
		}
		if second.duration != 2*time.Second || second.alphaMax != 0.1 || !second.fade || !second.repeat || second.animation != 1 {
			t.Fatalf("%s second cylinder = %+v", tc.name, second)
		}
	}

	warp, ok := worldEffectSpecForID(effectWarpZone2)
	if !ok || len(warp.components) != 3 || warp.duration != 7*time.Second {
		t.Fatalf("EF_WARPZONE2 spec = %+v ok=%t", warp, ok)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
	}{{1.15, 1.9}, {1.05, 1.8}} {
		component := warp.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 4*time.Second || component.duplicate != 4 || component.duplicateDelay != time.Second || component.alphaMax != 0.48 || component.animation != 3 || component.bottomSize != want.bottom || component.topSize != want.top || component.height != 1.35 || !component.repeat || component.rotate || !component.fade || !component.attachedEntity {
			t.Fatalf("EF_WARPZONE2 cylinder %d = %+v", i, component)
		}
	}
	particle := warp.components[2]
	if particle.kind != effectComponent3D || particle.textureFile != "effect/pok1.tga" || particle.duration != time.Second || particle.duplicate != 5 || particle.duplicateDelay != 300*time.Millisecond || particle.sizeStart != effectTableSize(32) || particle.posXStartRand != 1.45 || particle.posYStartRand != 1.45 || particle.posZEndRand != 1.5 || particle.posZEndMiddle != 1.4 || !particle.repeat || !particle.blendAdditive || !particle.attachedEntity {
		t.Fatalf("EF_WARPZONE2 particle = %+v", particle)
	}

	entry, ok := worldEffectSpecForID(effectEntry2)
	if !ok || len(entry.components) != 4 || entry.duration != 1500*time.Millisecond {
		t.Fatalf("EF_ENTRY2 spec = %+v ok=%t", entry, ok)
	}
	if len(entry.sfx) != 1 || entry.sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY2 sfx = %#v", entry.sfx)
	}
	if entry.components[0].bottomSize != 0.3 || entry.components[3].topSize != 1.3 {
		t.Fatalf("EF_ENTRY2 cylinder sizes = %+v", entry.components)
	}
	for i, component := range entry.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_blue" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.5 || component.animation != 5 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity {
			t.Fatalf("EF_ENTRY2 component %d = %+v", i, component)
		}
	}
}

func TestRobrowserHealAndRecoveryEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name            string
		id              int
		duration        time.Duration
		firstHeight     float64
		secondHeight    float64
		thirdHeight     float64
		firstDuplicate  int
		secondDuplicate int
		thirdDuplicate  int
		secondSize      float64
		secondSizeRand  float64
		thirdSize       float64
		sparkNumber     int
	}{
		{"EF_HEAL2", effectHeal2, 1890 * time.Millisecond, 15, 13, 2, 4, 20, 10, 9, 2, 9, 2},
		{"EF_HEAL4", effectHeal4, 2 * time.Second, 18, 15, 3, 7, 25, 15, 10, 5, 11, 3},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 6 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "_heal_effect.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		heights := []float64{tc.firstHeight, tc.secondHeight, tc.thirdHeight}
		for i, height := range heights {
			component := spec.components[i]
			if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.duration != 1500*time.Millisecond || component.alphaMax != 0.3 || component.animation != 1 || component.blendMode != 2 || !component.blendAdditive || !component.fade || !component.rotate || !component.attachedEntity || component.height != height {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
			if component.color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) {
				t.Fatalf("%s cylinder %d color = %+v", tc.name, i, component.color)
			}
		}
		if spec.components[0].bottomSize != 1.1 || spec.components[0].topSize != 1.1 || spec.components[1].bottomSize != 1 || spec.components[2].topSize != 3 {
			t.Fatalf("%s cylinder sizes = %+v", tc.name, spec.components[:3])
		}
		first, second, third := spec.components[3], spec.components[4], spec.components[5]
		if first.kind != effectComponent3D || first.textureFile != "effect/pok3.tga" || first.duration != 1500*time.Millisecond || first.duplicate != tc.firstDuplicate || first.duplicateDelay != 10*time.Millisecond || first.posXRand != 1.2 || first.posYRand != 1.2 || first.posZEndRand != 1 || first.posZEndMiddle != 8 || first.sizeStart != effectTableSize(9) || !first.sparkling || first.sparkNumber != tc.sparkNumber {
			t.Fatalf("%s first particle = %+v", tc.name, first)
		}
		if second.duration != 1300*time.Millisecond || second.delay != 400*time.Millisecond || second.duplicate != tc.secondDuplicate || second.posXRand != 1.5 || second.posYRand != 1.5 || second.posZEndRand != 3 || second.posZEndMiddle != 6 || second.sizeStart != effectTableSize(tc.secondSize) || second.sizeRand != effectTableSize(tc.secondSizeRand) || !second.sparkling || second.sparkNumber != tc.sparkNumber {
			t.Fatalf("%s second particle = %+v", tc.name, second)
		}
		if third.duration != 1100*time.Millisecond || third.delay != 200*time.Millisecond || third.duplicate != tc.thirdDuplicate || third.duplicateDelay != 50*time.Millisecond || third.posXRand != 1 || third.posYRand != 1 || third.posZEnd != 6 || third.posZStartRand != 1 || third.sizeStart != effectTableSize(tc.thirdSize) || third.sparkling {
			t.Fatalf("%s third particle = %+v", tc.name, third)
		}
	}

	for _, tc := range []struct {
		name  string
		id    int
		color color.RGBA
		wav   string
	}{
		{"EF_HPTIME", effectHPTime, color.RGBA{R: 230, G: 255, B: 230, A: 255}, "_heal_effect.wav"},
		{"EF_SPTIME", effectSPTime, color.RGBA{R: 230, G: 230, B: 255, A: 255}, "effect\\흡기.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 1110*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != "effect/pok1.tga" || component.color != tc.color || component.duration != 500*time.Millisecond || component.delay != 500*time.Millisecond || component.duplicate != 12 || component.duplicateDelay != 10*time.Millisecond || component.alphaMax != 0.8 || component.sizeStart != effectTableSize(30) || component.sizeRand != effectTableSize(20) || component.posXRand != 0.6 || component.posYRand != 0.6 || component.posZStartRand != 1.5 || component.posZStartMiddle != 2 || component.posZEndRand != 1 || component.posZEndMiddle != 5 || !component.sparkling || component.sparkNumber != 3 || !component.blendAdditive || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserAsuraAndGuardEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	asura, ok := worldEffectSpecForID(effectBeginAsura)
	if !ok || len(asura.components) != 14 || asura.duration != 2100*time.Millisecond {
		t.Fatalf("EF_BEGINASURA spec = %+v ok=%t", asura, ok)
	}
	firstRing, secondRing := asura.components[0], asura.components[1]
	if firstRing.kind != effectComponentCylinder || firstRing.textureName != "ring_white" || firstRing.duration != 800*time.Millisecond || firstRing.animation != 2 || firstRing.bottomSize != 1 || firstRing.topSize != 4.5 || firstRing.height != -4 || !firstRing.fade || !firstRing.attachedEntity || firstRing.blendMode != 2 {
		t.Fatalf("EF_BEGINASURA first ring = %+v", firstRing)
	}
	if secondRing.topSize != 2.5 || secondRing.height != -4 {
		t.Fatalf("EF_BEGINASURA second ring = %+v", secondRing)
	}
	firstGlyph := asura.components[2]
	if firstGlyph.kind != effectComponent3D || firstGlyph.textureFile != "effect/asura1.tga" || firstGlyph.duration != 1200*time.Millisecond || firstGlyph.delay != 0 || firstGlyph.duplicate != 3 || firstGlyph.duplicateDelay != 150*time.Millisecond || firstGlyph.alphaMax != 1 || firstGlyph.alphaMaxDelta != -0.25 || !firstGlyph.fadeIn || firstGlyph.fadeOut || firstGlyph.sizeStart != effectTableSize(250) || firstGlyph.sizeEnd != effectTableSize(120) || !firstGlyph.sizeSmooth || firstGlyph.posX != -6 || firstGlyph.posZ != 4 || !firstGlyph.overlay || !firstGlyph.attachedEntity {
		t.Fatalf("EF_BEGINASURA first glyph = %+v", firstGlyph)
	}
	if firstGlyph.color != (color.RGBA{R: 26, G: 26, B: 26, A: 255}) {
		t.Fatalf("EF_BEGINASURA glyph color = %+v", firstGlyph.color)
	}
	lastGlyph := asura.components[13]
	if lastGlyph.textureFile != "effect/asura6.tga" || lastGlyph.duration != 400*time.Millisecond || lastGlyph.delay != 1700*time.Millisecond || lastGlyph.duplicate != 0 || !lastGlyph.fadeOut || lastGlyph.sizeStart != effectTableSize(120) || lastGlyph.sizeEnd != effectTableSize(200) || lastGlyph.posX != 6 || lastGlyph.posZ != 4 {
		t.Fatalf("EF_BEGINASURA last glyph = %+v", lastGlyph)
	}

	guard, ok := worldEffectSpecForID(effectGuard)
	if !ok || len(guard.components) != 3 || guard.duration != 600*time.Millisecond {
		t.Fatalf("EF_GUARD spec = %+v ok=%t", guard, ok)
	}
	if len(guard.sfx) != 1 || guard.sfx[0] != "effect\\kyrie_guard.wav" {
		t.Fatalf("EF_GUARD sfx = %#v", guard.sfx)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
		posZ   float64
	}{
		{1.5, 1, 0.7, 2.14},
		{1.5, 1.5, 1.14, 1},
		{1, 1.5, 0.7, 0.3},
	} {
		component := guard.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "guardk" || component.duration != 600*time.Millisecond || component.alphaMax != 0.6 || component.blendMode != 2 || !component.blendAdditive || !component.fade || component.totalCircleSides != 8 || component.circleSides != 5 || component.angleY != 112.5 || !component.attachedEntity {
			t.Fatalf("EF_GUARD component %d = %+v", i, component)
		}
		if component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height || component.posZ != want.posZ {
			t.Fatalf("EF_GUARD component %d shape = %+v", i, component)
		}
		if component.color != (color.RGBA{R: 232, G: 255, B: 230, A: 255}) {
			t.Fatalf("EF_GUARD component %d color = %+v", i, component.color)
		}
	}
}

func TestRobrowserSimpleEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		head     bool
	}{
		{"EF_LKCONCENTRATION", effectLKConcentration, "twohand", "effect\\knight_twohandquicken.wav", true, true},
		{"EF_DEVIL", effectDeath, "devil", "", true, false},
		{"EF_MELTDOWN", effectMeltdown, "melt", "", true, false},
		{"EF_CARTBOOST", effectCartBoost, "cart", "effect\\ef_incagility.wav", true, false},
		{"EF_REJECTSWORD", effectRejectSword, "sword", "effect\\kyrie_guard.wav", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
		head bool
	}{
		{"EF_VALLENTINE", effectFoodChocolate, "vallentine", "effect\\vallentine.wav", false},
		{"EF_DRAGONSMOKE", effectDragonSmoke, "poisonhit", "", false},
		{"EF_LIGHTBLADE", effectLightBlade, "한복천사", "", true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity || component.spriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_AURABLADE", effectAuraBlade, "effect\\오라 블레이드.wav"},
		{"EF_REDBODY", effectRedBody, "effect\\버서크.wav"},
		{"EF_BOTTOM_GOSPEL", effectBottomGospel, "effect\\가스펠.wav"},
		{"EF_HITLINE2", effectHitLine2, "effect\\맹호경파산.wav"},
		{"EF_LINELINK2", effectLineLink2, "effect\\소울 체인지.wav"},
		{"EF_TRUESIGHT", effectTrueSight, "effect\\hunter_detecting.wav"},
		{"EF_TRIPLEATTACK2", effectTripleAttack2, "effect\\샤프슈팅.wav"},
		{"EF_PORTAL4", effectPortal4, "effect\\윈드워크.wav"},
		{"EF_TRIPLEATTACK3", effectTripleAttack3, "effect\\애로우 발칸.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	base, ok := worldEffectSpecForID(effectBaseLevelUp)
	if !ok || len(base.components) != 1 || base.components[0].kind != effectComponentSTR || base.components[0].strFile != "angel" || !base.components[0].attachedEntity {
		t.Fatalf("EF_ANGEL spec = %+v ok=%t", base, ok)
	}
}

func TestRobrowserLevel99AliasesThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  effectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_4", effectLevel99Aura1, "Level99Bubble", effectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_LEVEL99_5", effectLevel99AuraMid, "Level99Aura", effectFuncLevel99Aura, "effect/ring_blue.tga"},
		{"EF_LEVEL99_6", effectLevel99AuraBottom, "GroundAura", effectFuncGroundAura, "effect/pikapika2.bmp"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 5*time.Minute {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != tc.adapter || component.textureFile != tc.texture || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserCombatEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	soul, ok := worldEffectSpecForID(effectSoulBreaker)
	if !ok || len(soul.components) != 1 || soul.duration != 500*time.Millisecond {
		t.Fatalf("EF_SOULBREAKER spec = %+v ok=%t", soul, ok)
	}
	if len(soul.sfx) != 1 || soul.sfx[0] != "effect\\기공포.wav" {
		t.Fatalf("EF_SOULBREAKER sfx = %#v", soul.sfx)
	}
	if component := soul.components[0]; component.kind != effectComponent3D || component.textureFile != "effect/purpleslash.tga" || component.alphaMax != 0.4 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateWithCamera || !component.rotateToTarget || component.angleStart != 90 || component.posZ != 2 || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(200) || !component.attachedEntity {
		t.Fatalf("EF_SOULBREAKER component = %+v", component)
	}

	pressure, ok := worldEffectSpecForID(effectPressure)
	if !ok || len(pressure.components) != 3 || pressure.duration != 1001*time.Millisecond {
		t.Fatalf("EF_PRESSURE spec = %+v ok=%t", pressure, ok)
	}
	if len(pressure.sfx) != 1 || pressure.sfx[0] != "effect\\프레셔.wav" || pressure.cameraShakeDelay != 500*time.Millisecond || pressure.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_PRESSURE timing/sfx = sfx %#v delay %s shake %s", pressure.sfx, pressure.cameraShakeDelay, pressure.cameraShake)
	}
	first, second, quake := pressure.components[0], pressure.components[1], pressure.components[2]
	if first.kind != effectComponent3D || first.textureFile != "effect/cross_old.bmp" || first.duration != 500*time.Millisecond || first.alphaMax != 0.6 || first.blendMode != 2 || !first.blendAdditive || !first.rotate || first.angleEnd != -611 || first.posZ != 20 || first.posZEnd != 5 || first.sizeStart != effectTableSize(100) || !first.attachedEntity {
		t.Fatalf("EF_PRESSURE first cross = %+v", first)
	}
	if second.kind != effectComponent3D || second.delay != 501*time.Millisecond || !second.fadeOut || second.angleStart != -611 || second.posZ != 5 || second.sizeStart != effectTableSize(100) || !second.attachedEntity {
		t.Fatalf("EF_PRESSURE second cross = %+v", second)
	}
	if quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_PRESSURE quake = %+v", quake)
	}

	for _, tc := range []struct {
		name      string
		id        int
		funcName  string
		wav       string
		duration  time.Duration
		delay     time.Duration
		duplicate int
	}{
		{"EF_BASH3D", effectBash3D, "Bash3D", "effect\\bash3d.wav", 500 * time.Millisecond, 200 * time.Millisecond, 5},
		{"EF_BASH3D2", effectBash3D2, "Bash3D2", "effect\\mon_폭기.wav", 400 * time.Millisecond, 50 * time.Millisecond, 8},
		{"EF_BASH3D3", effectBash3D3, "Bash3D3", "effect\\헤드 크러쉬.wav", 675 * time.Millisecond, 500 * time.Millisecond, 6},
		{"EF_BASH3D4", effectBash3D4, "Bash3D4", "effect\\비트 조인트.wav", 675 * time.Millisecond, 500 * time.Millisecond, 6},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 3 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		if component := spec.components[0]; component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
			t.Fatalf("%s body func = %+v", tc.name, component)
		}
		for i, component := range spec.components[1:] {
			wantTop := 4.5
			if i == 1 {
				wantTop = 7.2
			}
			if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.duration != 175*time.Millisecond || component.delay != tc.delay || component.duplicate != tc.duplicate || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.height != 0 || component.bottomSize != 0.01 || component.topSize != wantTop || component.animation != 2 || component.totalCircleSides != 30 || component.circleSides != 1 || !component.attachedEntity {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
		}
	}
}

func TestRobrowserBasilicaDrainAndMagicEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	basilica, ok := worldEffectSpecForID(effectBottomBasilica)
	if !ok || len(basilica.components) != 4 || basilica.duration != 20*time.Second {
		t.Fatalf("EF_BOTTOM_BASILICA spec = %+v ok=%t", basilica, ok)
	}
	for i, want := range []struct {
		size   float64
		height float64
		alpha  float64
		angleY float64
	}{
		{2.45, 2.0, 32.0 / 255.0, 0},
		{2.52, 2.1, 32.0 / 255.0, 10},
		{2.6, 2.0, 15.0 / 255.0, 26.6},
		{2.6, 2.0, 15.0 / 255.0, 79.8},
	} {
		component := basilica.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second || component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != want.size || component.topSize != want.size || component.height != want.height || math.Abs(component.alphaMax-want.alpha) > 0.0001 || component.blendMode != 2 || !component.blendAdditive || !component.rotateWithCamera || component.angleY != want.angleY || !component.attachedEntity {
			t.Fatalf("EF_BOTTOM_BASILICA component %d = %+v", i, component)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		color     color.RGBA
		sizeStart float64
		sizeEnd   float64
	}{
		{"EF_ENERGYDRAIN2", effectEnergyDrain2, color.RGBA{R: 204, G: 204, B: 255, A: 255}, 160, 190},
		{"EF_ENERGYDRAIN3", effectEnergyDrain3, color.RGBA{R: 178, G: 255, B: 178, A: 255}, 140, 170},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != 600*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.spriteFile != "data/sprite/이팩트/particle1" || !component.spriteRepeat || component.duration != 600*time.Millisecond || component.duplicate != 5 || !component.fromSrc || !component.toSrc || !component.rotateToTarget || component.color != tc.color || component.sizeStart != effectTableSize(tc.sizeStart) || component.sizeEnd != effectTableSize(tc.sizeEnd) || component.posZ != 5 || component.arc != 3 || component.retreat != 3 {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	trans, ok := worldEffectSpecForID(effectTransBlueBody)
	if !ok || len(trans.components) != 1 || trans.duration != 200*db.EffectFrameDuration {
		t.Fatalf("EF_TRANSBLUEBODY spec = %+v ok=%t", trans, ok)
	}
	if component := trans.components[0]; component.kind != effectComponentFUNC || component.funcName != "TransBlueBody" || component.funcAdapter != effectFuncBodyColor || !component.attachedEntity {
		t.Fatalf("EF_TRANSBLUEBODY component = %+v", component)
	}

	magic, ok := worldEffectSpecForID(effectMagicCrasher)
	if !ok || len(magic.components) != 2 || magic.duration != 100*db.EffectFrameDuration {
		t.Fatalf("EF_MAGICCRASHER spec = %+v ok=%t", magic, ok)
	}
	if len(magic.sfx) != 1 || magic.sfx[0] != "effect\\매직 크래쉬.wav" || magic.cameraShakeDelay != 30*db.EffectFrameDuration || magic.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_MAGICCRASHER timing/sfx = sfx %#v delay %s shake %s", magic.sfx, magic.cameraShakeDelay, magic.cameraShake)
	}
	if body, quake := magic.components[0], magic.components[1]; body.kind != effectComponentFUNC || body.funcName != "MagicCrasherBodyColor" || !body.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || quake.delay != 30*db.EffectFrameDuration || !quake.attachedEntity {
		t.Fatalf("EF_MAGICCRASHER components = %+v", magic.components)
	}

	falcon, ok := worldEffectSpecForID(effectFalconAssault)
	if !ok || len(falcon.components) != 1 || falcon.duration != 500*time.Millisecond {
		t.Fatalf("EF_FALCONASSAULT spec = %+v ok=%t", falcon, ok)
	}
	if len(falcon.sfx) != 1 || falcon.sfx[0] != "effect\\hunter_blitzbeat.wav" || falcon.cameraShakeDelay != 300*time.Millisecond || falcon.cameraShake != 200*time.Millisecond {
		t.Fatalf("EF_FALCONASSAULT timing/sfx = sfx %#v delay %s shake %s", falcon.sfx, falcon.cameraShakeDelay, falcon.cameraShake)
	}
	if component := falcon.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.delay != 300*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_FALCONASSAULT component = %+v", component)
	}

	moonlit, ok := worldEffectSpecForID(effectMoonlit)
	if !ok || len(moonlit.components) != 1 || moonlit.duration != 20*time.Second {
		t.Fatalf("EF_SPHEREWIND2 spec = %+v ok=%t", moonlit, ok)
	}
	if len(moonlit.sfx) != 1 || moonlit.sfx[0] != "effect\\달빛세레나데.wav" {
		t.Fatalf("EF_SPHEREWIND2 sfx = %#v", moonlit.sfx)
	}
	if component := moonlit.components[0]; component.kind != effectComponentFUNC || component.funcName != "FlatColorTile" || component.funcAdapter != effectFuncFlatColorTile || component.color != (color.RGBA{R: 255, G: 138, B: 187, A: 153}) || component.sizeStart != 1 || component.attachedEntity {
		t.Fatalf("EF_SPHEREWIND2 component = %+v", component)
	}
}

func TestRobrowserEffectsFourHundredToFourFiftyMatchTableRows(t *testing.T) {
	portal, ok := worldEffectSpecForID(effectPortal5)
	if !ok || len(portal.components) != 1 || portal.duration != 800*time.Millisecond {
		t.Fatalf("EF_PORTAL5 spec = %+v ok=%t", portal, ok)
	}
	if component := portal.components[0]; component.kind != effectComponentFUNC || component.funcName != "EffectBodyColor" || component.funcAdapter != effectFuncBodyColor || component.duration != 800*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_PORTAL5 component = %+v", component)
	}

	mindBreaker, ok := worldEffectSpecForID(effectMagicCrasher2)
	if !ok || len(mindBreaker.components) != 1 || mindBreaker.duration != time.Second {
		t.Fatalf("EF_MAGICCRASHER2 spec = %+v ok=%t", mindBreaker, ok)
	}
	if len(mindBreaker.sfx) != 1 || mindBreaker.sfx[0] != "effect\\swordman_provoke.wav" {
		t.Fatalf("EF_MAGICCRASHER2 sfx = %#v", mindBreaker.sfx)
	}
	if component := mindBreaker.components[0]; component.kind != effectComponentFUNC || component.funcName != "EffectBodyColor" || component.funcAdapter != effectFuncBodyColor || component.duration != time.Second || !component.attachedEntity {
		t.Fatalf("EF_MAGICCRASHER2 component = %+v", component)
	}

	spider, ok := worldEffectSpecForID(effectBottomSpider)
	if !ok || len(spider.components) != 1 || spider.duration != 5*time.Second {
		t.Fatalf("EF_BOTTOM_SPIDER spec = %+v ok=%t", spider, ok)
	}
	if component := spider.components[0]; component.kind != effectComponentFUNC || component.funcName != "SpiderWeb" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != "effect/spiderweb.tga" || component.duration != 5*time.Second || math.Abs(component.alphaMax-0.7) > 0.0001 || component.sizeStart != 1.5 || component.sizeEnd != 1.5 || component.posZ != 0.05 || !component.renderBefore || component.attachedEntity {
		t.Fatalf("EF_BOTTOM_SPIDER component = %+v", component)
	}

	fogWall, ok := worldEffectSpecForID(effectFogWallGround)
	if !ok || len(fogWall.components) != 2 || fogWall.duration != 1500*time.Millisecond {
		t.Fatalf("PF_FOGWALL ground spec = %+v ok=%t", fogWall, ok)
	}
	if component := fogWall.components[0]; component.kind != effectComponentFUNC || component.funcName != "FlatColorTile" || component.funcAdapter != effectFuncFlatColorTile || component.color != (color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 153}) || component.sizeStart != 1 || !component.renderBefore || component.attachedEntity {
		t.Fatalf("PF_FOGWALL flat tile component = %+v", component)
	}
	if component := fogWall.components[1]; component.kind != effectComponentFUNC || component.funcName != "GroundTexture" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != "effect/lens_w.bmp" || component.duration != 1500*time.Millisecond || component.sizeStart != 0.5 || component.sizeEnd != 0.5 || math.Abs(component.alphaMax-0.7) > 0.0001 || component.posZ != 0.4 || !component.blendAdditive || !component.renderBefore || component.attachedEntity {
		t.Fatalf("PF_FOGWALL texture component = %+v", component)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_SOULBURN", effectSoulBurn, "소울번"},
		{"EF_SOULCHANGE", effectSoulChange, "사랑효과"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

	meteor, ok := worldEffectSpecForID(effectSoulBreaker2)
	if !ok || len(meteor.components) != 8 || meteor.duration != 500*time.Millisecond {
		t.Fatalf("EF_SOULBREAKER2 spec = %+v ok=%t", meteor, ok)
	}
	if len(meteor.sfx) != 1 || meteor.sfx[0] != "effect\\메테오 어썰트.wav" {
		t.Fatalf("EF_SOULBREAKER2 sfx = %#v", meteor.sfx)
	}
	for i, tc := range []struct {
		posX    float64
		posY    float64
		posXEnd float64
		posYEnd float64
		angle   float64
	}{
		{-1, 0, -5, 0, 0},
		{-0.7, -0.7, -3.53, -3.53, -45},
		{0, -1, 0, -5, -90},
		{0.7, -0.7, 3.53, -3.53, -135},
		{1, 0, 5, 0, -180},
		{0.7, 0.7, 3.53, 3.53, -225},
		{0, 1, 0, 5, -270},
		{-0.7, 0.7, -3.53, 3.53, -315},
	} {
		component := meteor.components[i]
		if component.kind != effectComponent3D || component.textureFile != "effect/purpleslash.tga" || component.duration != 500*time.Millisecond || math.Abs(component.alphaMax-0.6) > 0.0001 || !component.fadeOut || !component.rotateWithCamera || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(200) || component.posX != tc.posX || component.posY != tc.posY || component.posXEnd != tc.posXEnd || component.posYEnd != tc.posYEnd || component.angleStart != tc.angle {
			t.Fatalf("EF_SOULBREAKER2 slash %d = %+v", i, component)
		}
	}

	for _, tc := range []struct {
		name       string
		id         int
		funcName   string
		duration   time.Duration
		targetSize float64
	}{
		{"EF_BABYBODY", effectBabyBody, "EffectSmallTransition", 300 * time.Millisecond, 2.5},
		{"EF_BABYBODY2", effectBabyBody2, "EffectSmall", 5 * time.Minute, 2.5},
		{"EF_GIANTBODY", effectGiantBody, "EffectBigTransition", 300 * time.Millisecond, 7.5},
		{"EF_GIANTBODY2", effectGiantBody2, "EffectBig", 5 * time.Minute, 7.5},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || spec.duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != tc.funcName || component.funcAdapter != effectFuncUnknown || component.duration != tc.duration || component.sizeEnd != tc.targetSize || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_QUAKEBODY", effectQuakeBody, "effect\\복호격.wav"},
		{"EF_STOPEFFECT", effectStopEffect, "effect\\t_효과음1.wav"},
		{"EF_JUMPBODY", effectJumpBody, "effect\\t_회피2.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound only", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
	}

	assumptio, ok := worldEffectSpecForID(effectAssumptio2)
	if !ok || len(assumptio.components) != 1 {
		t.Fatalf("EF_ASSUMPTIO2 spec = %+v ok=%t", assumptio, ok)
	}
	if len(assumptio.sfx) != 1 || assumptio.sfx[0] != "effect\\아숨프티오.wav" {
		t.Fatalf("EF_ASSUMPTIO2 sfx = %#v", assumptio.sfx)
	}
	if component := assumptio.components[0]; component.kind != effectComponentSTR || component.strFile != "asum" || !component.attachedEntity {
		t.Fatalf("EF_ASSUMPTIO2 component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	darkCross, ok := worldEffectSpecForID(effectDarkGrandCross)
	if !ok || len(darkCross.components) != 0 || len(darkCross.sfx) != 0 {
		t.Fatalf("EF_GRANDCROSS2 spec = %+v ok=%t, want empty robr row", darkCross, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_NPC_STOP", effectNPCStop, "스톱"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_MOCHI", effectMochi, "찹쌀떡"},
		{"EF_LAMADAN", effectRamadan, "ramadan"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_AGIUP", effectNPCPowerUp, "effect\\mon_폭기.wav"},
		{"EF_JUMPKICK", effectJumpKick, "effect\\t_날라차기.wav"},
		{"EF_EDP", effectEDP, "effect\\assasin_cloaking.wav"},
		{"EF_GUARD2", effectPreserve, "effect\\black_maximize_power_sword_bic.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserDarkCombatEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	soul, ok := worldEffectSpecForID(effectDarkSoulStrike)
	if !ok || len(soul.components) != 2 || soul.duration != 450*time.Millisecond {
		t.Fatalf("EF_SOULSTRIKE2 spec = %+v ok=%t", soul, ok)
	}
	if len(soul.sfx) != 1 || soul.sfx[0] != "effect\\ef_soulstrike.wav" {
		t.Fatalf("EF_SOULSTRIKE2 sfx = %#v", soul.sfx)
	}
	spark, particle := soul.components[0], soul.components[1]
	if spark.kind != effectComponent3D || spark.textureFile != "effect/pok3.tga" || spark.duration != 200*time.Millisecond || spark.delay != 250*time.Millisecond || spark.duplicateDelay != 150*time.Millisecond || !spark.fadeIn || !spark.fadeOut || !spark.toSrc || spark.posZEnd != 1 || !spark.posZSmooth || spark.posZStartRand != 5 || spark.posZStartMiddle != 6 || spark.sizeStart != effectTableSize(50) || !spark.attachedEntity {
		t.Fatalf("EF_SOULSTRIKE2 spark = %+v", spark)
	}
	if particle.kind != effectComponent3D || particle.spriteFile != "data/sprite/이팩트/particle5" || !particle.spriteRepeat || particle.duration != 250*time.Millisecond || particle.duplicate != 5 || particle.duplicateDelay != 20*time.Millisecond || !particle.toSrc || !particle.rotateToTarget || particle.sizeStart != effectTableSize(100) || particle.sizeEnd != effectTableSize(500) || particle.posZ != 3 || particle.arc != 4 || particle.retreat != 4 {
		t.Fatalf("EF_SOULSTRIKE2 particle = %+v", particle)
	}

	jupitel, ok := worldEffectSpecForID(effectDarkJupitelHit)
	if !ok || len(jupitel.components) != 2 || jupitel.duration != 300*time.Millisecond {
		t.Fatalf("EF_YUFITEL2 spec = %+v ok=%t", jupitel, ok)
	}
	pang, blast := jupitel.components[0], jupitel.components[1]
	if pang.kind != effectComponent3D || pang.textureFile != "effect/pokjuk_d.bmp" || pang.duration != 100*time.Millisecond || pang.sizeStart != 0 || pang.sizeEnd != effectTableSize(25) || pang.blendMode != 2 || !pang.blendAdditive || !pang.rotateToTarget || !pang.fadeOut || !pang.overlay || !pang.attachedEntity {
		t.Fatalf("EF_YUFITEL2 pang = %+v", pang)
	}
	if blast.kind != effectComponent3D || len(blast.textureFiles) != 5 || blast.textureFiles[0] != "effect/twirl_soft.bmp" || blast.textureFiles[1] != "effect/thunder_ball_b.bmp" || blast.textureFiles[3] != "effect/thunder_ball_c.bmp" || blast.frameDelay != 10*time.Millisecond || blast.duration != 300*time.Millisecond || blast.sizeStart != effectTableSize(75) || blast.blendMode != 2 || !blast.blendAdditive || !blast.overlay || !blast.attachedEntity {
		t.Fatalf("EF_YUFITEL2 blast = %+v", blast)
	}

	casting, ok := worldEffectSpecForID(effectDarkCasting)
	if !ok || len(casting.components) != 1 || casting.duration != 900*time.Millisecond {
		t.Fatalf("EF_DARKCASTING spec = %+v ok=%t", casting, ok)
	}
	if len(casting.sfx) != 1 || casting.sfx[0] != "effect\\ef_beginspell.wav" {
		t.Fatalf("EF_DARKCASTING sfx = %#v", casting.sfx)
	}
	ring := casting.components[0]
	if ring.kind != effectComponentCylinder || ring.textureName != "ring_black" || ring.alphaMax != 0.8 || ring.animation != 2 || ring.blendMode != 2 || !ring.blendAdditive || ring.bottomSize != 1 || ring.topSize != 5 || ring.height != 4 || !ring.fade || !ring.rotate || !ring.attachedEntity {
		t.Fatalf("EF_DARKCASTING ring = %+v", ring)
	}
}

func TestRobrowserMildWindEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BEGINASURA1", effectBeginAsura1, "effect/hanmoon1.tga"},
		{"EF_BEGINASURA2", effectBeginAsura2, "effect/hanmoon2.tga"},
		{"EF_BEGINASURA3", effectBeginAsura3, "effect/hanmoon3.tga"},
		{"EF_BEGINASURA4", effectBeginAsura4, "effect/hanmoon4.tga"},
		{"EF_BEGINASURA5", effectBeginAsura5, "effect/hanmoon7.tga"},
		{"EF_BEGINASURA6", effectBeginAsura6, "effect/hanmoon5.tga"},
		{"EF_BEGINASURA7", effectBeginAsura7, "effect/hanmoon6.tga"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 5 || spec.duration != time.Second {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\t_바람방출.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		first, second, last := spec.components[0], spec.components[1], spec.components[4]
		if first.kind != effectComponent3D || first.textureFile != tc.texture || first.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || first.alphaMax != 1 || first.sizeStart != effectTableSize(300) || first.sizeEnd != effectTableSize(100) || !first.sizeSmooth || first.posZ != 4 || first.blendMode != 2 || !first.blendAdditive || !first.fadeIn || !first.fadeOut || !first.attachedEntity {
			t.Fatalf("%s first glyph = %+v", tc.name, first)
		}
		if second.color != (color.RGBA{R: 178, G: 178, B: 255, A: 255}) || second.alphaMax != 0.2 || second.sizeStart != effectTableSize(220) || second.sizeEnd != effectTableSize(20) {
			t.Fatalf("%s second glyph = %+v", tc.name, second)
		}
		if last.color != (color.RGBA{R: 25, G: 25, B: 255, A: 255}) || last.alphaMax != 0.2 || last.sizeStart != effectTableSize(450) || last.sizeEnd != effectTableSize(100) {
			t.Fatalf("%s last glyph = %+v", tc.name, last)
		}
	}
}

func TestRobrowserSimpleEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_MAPAE", effectMapae, "mapae", "effect\\mapae.wav"},
		{"EF_ITEMPOKJUK", effectItemPokJuk, "itempokjuk", "effect\\itempokjuk.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_05VAL", effectValentine05, "05vallentine", ""},
		{"EF_ITEMFAST", effectItemFastDown, "fast", "effect\\fast.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	hermode, ok := worldEffectSpecForID(effectBottomHermode)
	if !ok || len(hermode.components) != 0 || len(hermode.sfx) != 0 {
		t.Fatalf("EF_BOTTOM_HERMODE spec = %+v ok=%t, want empty robr row", hermode, ok)
	}
	hermodeMusic, ok := worldEffectSpecForID(effectHermodeMusic)
	if !ok || len(hermodeMusic.components) != 0 || len(hermodeMusic.sfx) != 1 || hermodeMusic.sfx[0] != "effect\\헤르모드의 지팡이" {
		t.Fatalf("517_music spec = %+v ok=%t, want robr Hermode sound", hermodeMusic, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_HATED", effectHated, "effect\\t_보조마법.wav"},
		{"EF_STIN", effectStin, "effect\\t_에너지방출.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserFuncEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	spin, ok := worldEffectSpecForID(effectCastSpin)
	if !ok || spin.duration != 500*time.Millisecond || len(spin.components) != 1 {
		t.Fatalf("EF_CASTSPIN spec = %+v ok=%t", spin, ok)
	}
	spinComponent := spin.components[0]
	if spinComponent.kind != effectComponentFUNC || spinComponent.funcName != "CastSpin" || spinComponent.funcAdapter != effectFuncUnknown || !spinComponent.attachedEntity {
		t.Fatalf("EF_CASTSPIN component = %+v, want attached unsupported CastSpin FUNC", spinComponent)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgi2)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI2 spec = %+v ok=%t", chookgi, ok)
	}
	sphere := chookgi.components[0]
	if sphere.kind != effectComponentFUNC || sphere.funcName != "SpiritSphere" || sphere.funcAdapter != effectFuncSpiritSphere || sphere.textureFile != "effect/thunder_center.bmp" || sphere.duplicate != 5 || !sphere.attachedEntity {
		t.Fatalf("EF_CHOOKGI2 component = %+v", sphere)
	}

	chemical, ok := worldEffectSpecForID(effectChemical2Dash)
	if !ok || chemical.duration != 500*time.Millisecond || chemical.cameraShake != 200*time.Millisecond || chemical.cameraShakeDelay != 132*time.Millisecond || len(chemical.components) != 1 {
		t.Fatalf("EF_CHEMICAL2DASH spec = %+v ok=%t", chemical, ok)
	}
	if component := chemical.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || !component.attachedEntity {
		t.Fatalf("EF_CHEMICAL2DASH component = %+v", component)
	}

	acid, ok := worldEffectSpecForID(effectAcidDemon)
	if !ok || acid.duration != 500*time.Millisecond || acid.cameraShake != 200*time.Millisecond || acid.cameraShakeDelay != 200*time.Millisecond || len(acid.components) != 1 {
		t.Fatalf("EF_ACIDDEMON spec = %+v ok=%t", acid, ok)
	}
	if component := acid.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.delay != 200*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_ACIDDEMON component = %+v", component)
	}
}

func TestRobrowserBeginAsuraElevenMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBeginAsura11)
	if !ok || spec.duration != 2100*time.Millisecond || len(spec.components) != 14 {
		t.Fatalf("EF_BEGINASURA11 spec = %+v ok=%t", spec, ok)
	}
	firstRing, secondRing := spec.components[0], spec.components[1]
	for i, ring := range []worldEffectComponent{firstRing, secondRing} {
		if ring.kind != effectComponentCylinder || ring.textureName != "ring_white" || ring.duration != 800*time.Millisecond || ring.animation != 2 || ring.bottomSize != 1 || ring.height != -4 || !ring.fade || ring.rotate || !ring.attachedEntity || ring.blendMode != 2 || !ring.blendAdditive {
			t.Fatalf("EF_BEGINASURA11 ring %d = %+v", i, ring)
		}
	}
	if firstRing.topSize != 4.5 || secondRing.topSize != 2.5 {
		t.Fatalf("EF_BEGINASURA11 ring top sizes = %.1f %.1f", firstRing.topSize, secondRing.topSize)
	}

	firstGlyph, firstOut, lastOut := spec.components[2], spec.components[3], spec.components[13]
	if firstGlyph.kind != effectComponent3D || firstGlyph.textureFile != "effect/asura11.tga" || firstGlyph.duration != 1200*time.Millisecond || firstGlyph.delay != 0 || firstGlyph.posX != -8 || firstGlyph.posZ != 4 || firstGlyph.alphaMax != 1 || firstGlyph.duplicate != 3 || firstGlyph.duplicateDelay != 150*time.Millisecond || firstGlyph.alphaMaxDelta != -0.25 || !firstGlyph.fadeIn || firstGlyph.fadeOut || firstGlyph.sizeStart != effectTableSize(300) || firstGlyph.sizeEnd != effectTableSize(150) || !firstGlyph.sizeSmooth || !firstGlyph.attachedEntity || !firstGlyph.overlay {
		t.Fatalf("EF_BEGINASURA11 first glyph = %+v", firstGlyph)
	}
	if firstOut.textureFile != "effect/asura11.tga" || firstOut.duration != 400*time.Millisecond || firstOut.delay != 1200*time.Millisecond || firstOut.posX != -8 || firstOut.sizeStart != effectTableSize(150) || firstOut.sizeEnd != effectTableSize(250) || firstOut.fadeIn || !firstOut.fadeOut || firstOut.duplicate != 0 {
		t.Fatalf("EF_BEGINASURA11 first fade-out glyph = %+v", firstOut)
	}
	if lastOut.textureFile != "effect/asura16.tga" || lastOut.delay != 1700*time.Millisecond || lastOut.posX != 8 || lastOut.sizeStart != effectTableSize(150) || lastOut.sizeEnd != effectTableSize(250) {
		t.Fatalf("EF_BEGINASURA11 last glyph = %+v", lastOut)
	}
}

func TestRobrowserTarotCardsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	ids := []int{
		effectTarotCard1,
		effectTarotCard2,
		effectTarotCard3,
		effectTarotCard4,
		effectTarotCard5,
		effectTarotCard6,
		effectTarotCard7,
		effectTarotCard8,
		effectTarotCard9,
		effectTarotCard10,
		effectTarotCard11,
		effectTarotCard12,
		effectTarotCard13,
		effectTarotCard14,
	}
	for i, id := range ids {
		name := fmt.Sprintf("EF_TAROTCARD%d", i+1)
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != 3*time.Second || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\priest_slowpoison.wav" {
			t.Fatalf("%s sfx = %#v", name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != fmt.Sprintf("effect/tarot%02d.tga", i+1) || component.duration != 3*time.Second || component.alphaMax != 1 || !component.attachedEntity || !component.fadeIn || !component.fadeOut || component.posZ != 4 || component.sizeStart != effectTableSize(100) || component.sizeEnd != effectTableSize(70) || !component.sizeSmooth {
			t.Fatalf("%s component = %+v", name, component)
		}
	}
}

func TestRobrowserSimpleEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	stin2, ok := worldEffectSpecForID(effectStin2)
	if !ok || len(stin2.components) != 0 || len(stin2.sfx) != 5 || len(stin2.sfxDelays) != 5 {
		t.Fatalf("EF_STIN2 spec = %+v ok=%t", stin2, ok)
	}
	for i := range stin2.sfx {
		if stin2.sfx[i] != "effect\\t_날라차기.wav" || stin2.sfxDelays[i] != time.Duration(i)*200*time.Millisecond {
			t.Fatalf("EF_STIN2 sound %d = %q delay %s", i, stin2.sfx[i], stin2.sfxDelays[i])
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STIN3", effectStin3, "effect\\t_에너지방출.wav"},
		{"EF_KAIZEL", effectKaizel, "effect\\priest_resurrection.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_HFLIMOON1", effectHfliMoon1, "moonlight_1", "effect\\h_moonlight_1.wav"},
		{"EF_HFLIMOON2", effectHfliMoon2, "moonlight_2", "effect\\h_moonlight_2.wav"},
		{"EF_HFLIMOON3", effectHfliMoon3, "moonlight_3", "effect\\h_moonlight_3.wav"},
		{"EF_HO_UP", effectHoUp, "h_levelup", ""},
		{"EF_HAMIDEFENCE", effectHamiDefence, "defense", ""},
		{"EF_FOOD01", effectStatFoodSTR, "food_str", "_heal_effect.wav"},
		{"EF_FOOD02", effectStatFoodINT, "food_int", "_heal_effect.wav"},
		{"EF_FOOD03", effectStatFoodVIT, "food_vit", "_heal_effect.wav"},
		{"EF_FOOD04", effectStatFoodAGI, "food_agi", "_heal_effect.wav"},
		{"EF_FOOD05", effectStatFoodDEX, "food_dex", "_heal_effect.wav"},
		{"EF_FOOD06", effectStatFoodLUK, "food_luk", "_heal_effect.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		file      string
		wav       string
		direction bool
	}{
		{"EF_HAMICASTLE", effectHamiCastle, "캐슬링", "", false},
		{"EF_HAMIBLOOD", effectHamiBlood, "블러드러스트", "", false},
		{"EF_ITEM_THUNDER", effectItemThunder, "item_thunder", "", false},
		{"EF_ITEM_CLOUD", effectItemCloud, "item_cloud", "", false},
		{"EF_ITEM_CURSE", effectItemCurse, "item_curse", "", false},
		{"EF_ITEM_ZZZ", effectItemZZZ, "item_zzz", "_snore.wav", false},
		{"EF_ITEM_RAIN", effectItemRain, "item_rain", "", false},
		{"EF_M01", effectM01, "m_ef01", "", false},
		{"EF_M02", effectM02, "m_ef02", "", true},
		{"EF_M03", effectM03, "m_ef03", "", false},
		{"EF_M04", effectM04, "m_ef04", "", false},
		{"EF_M05", effectM05, "m_ef05", "dragon_breath.wav", false},
		{"EF_M06", effectM06, "m_ef06", "", false},
		{"EF_M07", effectM07, "m_ef07", "effect\\t_보조마법.wav", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || !component.attachedEntity || component.spriteDirection != tc.direction {
			t.Fatalf("%s component = %+v, want attached SPR %q direction=%t", tc.name, component, tc.file, tc.direction)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserFunctionAndProjectileEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	quake, ok := worldEffectSpecForID(effectScreenQuake)
	if !ok || quake.duration != 200*time.Millisecond || quake.cameraShake != 200*time.Millisecond || len(quake.components) != 1 {
		t.Fatalf("EF_SCREEN_QUAKE spec = %+v ok=%t", quake, ok)
	}
	if component := quake.components[0]; component.kind != effectComponentFUNC || component.funcName != "CameraQuake" || component.duration != 200*time.Millisecond || !component.attachedEntity {
		t.Fatalf("EF_SCREEN_QUAKE component = %+v", component)
	}

	throw, ok := worldEffectSpecForID(effectThrowItem6)
	if !ok || throw.duration != 200*time.Millisecond || len(throw.components) != 1 {
		t.Fatalf("EF_THROWITEM6 spec = %+v ok=%t", throw, ok)
	}
	component := throw.components[0]
	if component.kind != effectComponent3D || component.textureFile != "유저인터페이스/item/베넘나이프.bmp" || component.duration != 200*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 540 || component.posZ != 1 || component.sizeStart != effectTableSize(30) || component.sizeEnd != effectTableSize(30) || !component.attachedEntity {
		t.Fatalf("EF_THROWITEM6 component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
		rand     bool
	}{
		{"EF_FIREHIT2", effectFireHit2, "firehit%d", "", true, true},
		{"EF_COOKING_OK", effectCookingOK, "cook_suc", "_heal_effect.wav", true, false},
		{"EF_COOKING_FAIL", effectCookingFail, "cook_fail", "caramel_die.wav", true, false},
		{"EF_KOUENKA", effectKouenka, "firehit", "effect\\ef_firearrow%d.wav", true, true},
		{"EF_HYOUSENSOU", effectHyousensou, "freeze", "effect\\ef_icearrow%d.wav", true, true},
		{"EF_THUNDERSTORM2", effectThunderStorm2, "setsudan", "effect\\ef_thunderstorm.wav", true, false},
		{"EF_BAKU", effectBaku, "fire dragon", "effect\\폭염룡.wav", false, false},
		{"EF_HYOUSYOURAKU", effectHyousyouraku, "icy", "effect\\빙정락.wav", false, false},
		{"EF_TRACKCASTING", effectTrackCasting, "트랙킹", "", true, false},
		{"EF_BULLSEYE", effectBullseye, "불스아이", "", true, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.rand {
			if component.strRandMin != 1 || component.strRandMax != 3 {
				t.Fatalf("%s STR rand = %d..%d", tc.name, component.strRandMin, component.strRandMax)
			}
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		if tc.rand && (spec.sfxRandMin != 1 || spec.sfxRandMax != 3) {
			t.Fatalf("%s sfx rand = %d..%d", tc.name, spec.sfxRandMin, spec.sfxRandMax)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		file      string
		wav       string
		attached  bool
		stop      bool
		repeat    bool
		direction bool
	}{
		{"EF_NPC_STOP2", effectNPCStop2, "cconfine", "effect\\ef_hit6.wav", true, true, false, false},
		{"EF_FVOICE", effectFVoice, "fvoice", "amon_ra_die01.wav", false, false, false, false},
		{"EF_WINK", effectWink, "wink", "", false, false, false, false},
		{"EF_KIRIKAGE", effectKirikage, "그림자베기", "effect\\그림자베기.wav", true, false, false, false},
		{"EF_TATAMI", effectTatami, "다다미 뒤집기", "effect\\다다미뒤집기.wav", true, false, false, false},
		{"EF_KASUMIKIRI", effectKasumikiri, "안개베기", "effect\\안개베기.wav", true, false, false, false},
		{"EF_ISSEN", effectIssen, "일섬", "effect\\일섬.wav", true, false, false, false},
		{"EF_KAEN", effectKaen, "화염진", "effect\\화염진.wav", true, false, true, false},
		{"EF_DESPERADO", effectDesperado, "데스페라도", "effect\\데스페라도.wav", true, false, false, false},
		{"EF_LIGHTNING_S", effectLightningS, "라이트닝스피어", "", false, false, false, false},
		{"EF_BLIND_S", effectBlindS, "블라인드스피어", "", false, false, false, false},
		{"EF_POISON_S", effectPoisonS, "포이즌스피어", "", false, false, false, false},
		{"EF_FREEZING_S", effectFreezingS, "프리징스피어", "", false, false, false, false},
		{"EF_FLARE_S", effectFlareS, "플레어스피어", "", false, false, false, false},
		{"EF_RAPIDSHOWER", effectRapidShower, "래피드샤워", "effect\\래피드샤워.wav", true, false, false, false},
		{"EF_MAGICALBULLET", effectMagicalBullet, "매지컬불릿", "effect\\매지컬블릿.wav", true, false, false, false},
		{"EF_SPREADATTACK", effectSpreadAttack, "스프레드", "", true, false, false, true},
		{"EF_TRACKING", effectTracking, "트래킹", "", true, false, false, false},
		{"EF_TRIPLEACTION", effectTripleAction, "트리플액션", "effect\\트리플액션.wav", true, false, false, false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || component.attachedEntity != tc.attached || component.spriteStopAtEnd != tc.stop || component.repeat != tc.repeat || component.spriteDirection != tc.direction {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	hapgyeok, ok := worldEffectSpecForID(effectHapgyeok)
	if !ok || len(hapgyeok.components) != 2 {
		t.Fatalf("EF_HAPGYEOK spec = %+v ok=%t", hapgyeok, ok)
	}
	if len(hapgyeok.sfx) != 1 || hapgyeok.sfx[0] != "effect\\itempokjuk.wav" {
		t.Fatalf("EF_HAPGYEOK sfx = %#v", hapgyeok.sfx)
	}
	if spr, str := hapgyeok.components[0], hapgyeok.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != "합격_" || !spr.attachedEntity || str.kind != effectComponentSTR || str.strFile != "itempokjuk" || !str.attachedEntity {
		t.Fatalf("EF_HAPGYEOK components = %+v", hapgyeok.components)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_STIN4", effectStin4, "effect\\풍인.wav"},
		{"EF_RG_COIN3", effectRGCoin3, "effect\\디스암.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}
}

func TestRobrowserProjectileEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		texture string
		size    float64
	}{
		{"EF_THROWITEM7", effectThrowItem7, "유저인터페이스/item/수리검.bmp", 30},
		{"EF_THROWITEM8", effectThrowItem8, "유저인터페이스/item/쿠나이_독.bmp", 30},
		{"EF_THROWITEM9", effectThrowItem9, "유저인터페이스/item/풍마_뇌우.bmp", 30},
		{"EF_THROWITEM10", effectThrowItem10, "effect/coin_a.bmp", 20},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 200*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\닌자_던지기.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.duration != 200*time.Millisecond || component.alphaMax != 1 || !component.fadeIn || !component.fadeOut || !component.toSrc || !component.rotateToTarget || !component.rotateWithCamera || !component.rotate || component.angleStart != 180 || component.angleEnd != 540 || component.posZ != 1 || component.sizeStart != effectTableSize(tc.size) || component.sizeEnd != effectTableSize(tc.size) || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserFuncEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	dust, ok := worldEffectSpecForID(effectBash3D5)
	if !ok || dust.duration != 175*time.Millisecond || len(dust.components) != 3 {
		t.Fatalf("EF_BASH3D5 spec = %+v ok=%t", dust, ok)
	}
	if len(dust.sfx) != 1 || dust.sfx[0] != "effect\\bash3d5.wav" {
		t.Fatalf("EF_BASH3D5 sfx = %#v", dust.sfx)
	}
	body, first, second := dust.components[0], dust.components[1], dust.components[2]
	if body.kind != effectComponentFUNC || body.funcName != "Bash3D5" || !body.attachedEntity {
		t.Fatalf("EF_BASH3D5 body = %+v", body)
	}
	for i, component := range []worldEffectComponent{first, second} {
		if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.duration != 175*time.Millisecond || component.duplicate != 6 || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.height != 0 || component.bottomSize != 0.01 || component.animation != 2 || !component.attachedEntity || component.totalCircleSides != 30 || component.circleSides != 1 {
			t.Fatalf("EF_BASH3D5 cylinder %d = %+v", i, component)
		}
	}
	if first.topSize != 4.5 || second.topSize != 7.2 {
		t.Fatalf("EF_BASH3D5 top sizes = %.1f %.1f", first.topSize, second.topSize)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgi3)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI3 spec = %+v ok=%t", chookgi, ok)
	}
	if sphere := chookgi.components[0]; sphere.kind != effectComponentFUNC || sphere.funcName != "SpiritSphere" || sphere.funcAdapter != effectFuncSpiritSphere || sphere.textureFile != "effect/thunder_center.bmp" || sphere.duplicate != 5 || !sphere.attachedEntity {
		t.Fatalf("EF_CHOOKGI3 component = %+v", sphere)
	}
}

func TestRobrowserEffectsSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	earthquake, ok := worldEffectSpecForID(effectNPCEarthquake)
	if !ok || earthquake.cameraShake != 650*time.Millisecond || len(earthquake.components) != 2 {
		t.Fatalf("EF_NPC_EARTHQUAKE spec = %+v ok=%t", earthquake, ok)
	}
	if len(earthquake.sfx) != 1 || earthquake.sfx[0] != "effect\\earth_quake.wav" {
		t.Fatalf("EF_NPC_EARTHQUAKE sfx = %#v", earthquake.sfx)
	}
	if spr, quake := earthquake.components[0], earthquake.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != "어스퀘이크" || !spr.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || quake.duplicate != 3 || quake.duplicateDelay != 35*time.Millisecond || !quake.attachedEntity {
		t.Fatalf("EF_NPC_EARTHQUAKE components = %+v", earthquake.components)
	}

	dragon, ok := worldEffectSpecForID(effectDragonFear)
	if !ok || dragon.cameraShake != 650*time.Millisecond || len(dragon.components) != 2 {
		t.Fatalf("EF_DRAGONFEAR spec = %+v ok=%t", dragon, ok)
	}
	if len(dragon.sfx) != 1 || dragon.sfx[0] != "effect\\dragonfear.wav" {
		t.Fatalf("EF_DRAGONFEAR sfx = %#v", dragon.sfx)
	}
	if str, quake := dragon.components[0], dragon.components[1]; str.kind != effectComponentSTR || str.strFile != "dragon_h" || !str.attachedEntity || quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_DRAGONFEAR components = %+v", dragon.components)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_BLEEDING", effectWideBleeding, "wideb", "effect\\wideb.wav"},
		{"EF_WIDECONFUSE", effectWideConfuse, "dfear", "effect\\dragonfear.wav"},
		{"EF_CRITICALWOUND", effectCriticalWound, "cwound", ""},
		{"EF_FLOWERLEAF", effectFlowerLeaf, "flower_leaf", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BOTTOM_RUNNER", effectBottomRunner, "effect/hanmoon1.tga"},
		{"EF_BOTTOM_TRANSFER", effectBottomTransfer, "effect/hanmoon2.tga"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 1500*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one ground texture component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentFUNC || component.funcName != "GroundTexture" || component.funcAdapter != effectFuncGroundTexture || component.textureFile != tc.texture || component.sizeStart != 1 || component.sizeEnd != 1 || component.posZ != 0.05 || !component.blendAdditive || component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	evil, ok := worldEffectSpecForID(effectBottomEvilLand)
	if !ok || evil.duration != 1500*time.Millisecond || len(evil.components) != 2 {
		t.Fatalf("EF_BOTTOM_EVILLAND spec = %+v ok=%t", evil, ok)
	}
	tile, curse := evil.components[0], evil.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 160, G: 160, B: 160, A: 51}) || tile.sizeStart != 1 || tile.attachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND tile = %+v", tile)
	}
	if curse.kind != effectComponentFUNC || curse.funcName != "GroundTexture" || curse.funcAdapter != effectFuncGroundTexture || curse.textureFile != "effect/curse.bmp" || curse.sizeStart != 1 || curse.sizeEnd != 1 || curse.alphaMax != 0.7 || curse.posZ != 0.4 || !curse.blendAdditive || curse.attachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND curse = %+v", curse)
	}

	guard, ok := worldEffectSpecForID(effectGuard3)
	if !ok || len(guard.components) != 0 || guard.duration != 500*time.Millisecond {
		t.Fatalf("EF_GUARD3 spec = %+v ok=%t, want sound-only", guard, ok)
	}
	if len(guard.sfx) != 1 || guard.sfx[0] != "effect\\kyrie_guard.wav" {
		t.Fatalf("EF_GUARD3 sfx = %#v", guard.sfx)
	}
}

func TestHighWizardStringKeyEffectsMatchRobrowser(t *testing.T) {
	magicPower, ok := worldEffectSpecForID(effectMagicPower)
	if !ok || magicPower.duration != 500*time.Millisecond || len(magicPower.components) != 0 {
		t.Fatalf("ef_magicpower spec = %+v ok=%t, want sound-only", magicPower, ok)
	}
	if len(magicPower.sfx) != 1 || magicPower.sfx[0] != "effect\\마법력 증폭.wav" {
		t.Fatalf("ef_magicpower sfx = %#v", magicPower.sfx)
	}

	gravitation, ok := worldEffectSpecForID(effectGravitation)
	if !ok || gravitation.duration != 1500*time.Millisecond || gravitation.cameraShake != 200*time.Millisecond || len(gravitation.components) != 2 {
		t.Fatalf("522_ground spec = %+v ok=%t", gravitation, ok)
	}
	tile, lens := gravitation.components[0], gravitation.components[1]
	if tile.kind != effectComponentFUNC || tile.funcName != "FlatColorTile" || tile.funcAdapter != effectFuncFlatColorTile || tile.color != (color.RGBA{R: 255, G: 255, B: 255, A: 51}) || tile.sizeStart != 1 || !tile.attachedEntity {
		t.Fatalf("522_ground tile = %+v", tile)
	}
	if lens.kind != effectComponentFUNC || lens.funcName != "GroundTexture" || lens.funcAdapter != effectFuncGroundTexture || lens.textureFile != "effect/lens_w.bmp" || lens.sizeStart != 0.5 || lens.sizeEnd != 0.5 || lens.alphaMax != 0.7 || lens.posZ != 0.4 || !lens.blendAdditive || !lens.attachedEntity {
		t.Fatalf("522_ground lens = %+v", lens)
	}
}

func TestRobrowserFirecrackerBannersSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POK_LOVE", effectFirecracker2, "폭죽_러브"},
		{"EF_POK_WHITE", effectFirecracker3, "폭죽_화이트데이"},
		{"EF_POK_VALEN", effectFirecracker4, "폭죽_발렌타인"},
		{"EF_POK_BIRTH", effectFirecracker5, "폭죽_생일"},
		{"EF_POK_CHRISTMAS", effectFirecracker6, "폭죽_크리스마스"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want SPR banner plus STR itempokjuk", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\itempokjuk.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.sfx)
		}
		if spr, str := spec.components[0], spec.components[1]; spr.kind != effectComponentSPR || spr.spriteFile != tc.file || !spr.attachedEntity || str.kind != effectComponentSTR || str.strFile != "itempokjuk" || !str.attachedEntity {
			t.Fatalf("%s components = %+v", tc.name, spec.components)
		}
	}
}

func TestRobrowserSimpleEffectsSevenHundredToSevenFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		file     string
		wav      string
		attached bool
	}{
		{"EF_ITEM315", effectItem315, "mobile_ef02", "", true},
		{"EF_ITEM316", effectItem316, "mobile_ef01", "", true},
		{"EF_ITEM317", effectItem317, "mobile_ef03", "", true},
		{"EF_STORM_MIN", effectStormMin, "storm_min", "effect\\wizard_stormgust.wav", true},
		{"EF_POK_JAP", effectFirecracker7, "pokjuk_jap", "", false},
		{"EF_ADO_STR", effectAdoramus, "ado", "effect\\ab_adoramus.wav", true},
		{"EF_IGN_STR", effectIgnitionBreak, "이그니션브레이크", "effect\\wl_jackfrost.wav", true},
		{"EF_CRIMSON_STR", effectCrimsonRock, "crimson_r", "effect\\crimson_r.wav", true},
		{"EF_HELL_STR", effectHellInferno, "hell_in", "", true},
		{"EF_DHOWL_STR", effectDragonHowling, "dragon_h", "dragon_h.wav", true},
		{"EF_CHAINL_STR", effectChainLightning, "chainlight", "effect\\chainlight.wav", true},
		{"EF_AIMED_STR", effectAimedBolt, "aimed", "", true},
		{"EF_ARROWSTORM_STR", effectArrowStorm, "arrowstorm", "", true},
		{"EF_LAULAMUS_STR", effectLaulamus, "laulamus", "", true},
		{"EF_LAUAGNUS_STR", effectLauagnus, "lauagnus", "", true},
		{"EF_MILSHIELD_STR", effectMillenniumShield, "mil_shield", "", true},
		{"EF_CONCENTRATION2", effectConcentration2, "concentration", "", true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.attachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	wewish, ok := worldEffectSpecForID(effectChristmasCarol)
	if !ok || len(wewish.sfx) != 1 || wewish.sfx[0] != "effect\\wewish.wav" {
		t.Fatalf("EF_WEWISH spec = %+v ok=%t", wewish, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_FORESTLIGHT5", effectForestLight5, "effect\\ab_renovation.wav"},
		{"EF_FROSTMYSTY", effectFrostMisty, "effect\\t_에나지방출.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 0 || spec.duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	marsh, ok := worldEffectSpecForID(effectMarshOfAbyss)
	if !ok || len(marsh.components) != 1 {
		t.Fatalf("EF_SPR_MASH spec = %+v ok=%t, want one SPR component", marsh, ok)
	}
	if component := marsh.components[0]; component.kind != effectComponentSPR || component.spriteFile != "mashofa" || component.attachedEntity {
		t.Fatalf("EF_SPR_MASH component = %+v", component)
	}
}

func TestRobrowserCylinderEffectsSevenHundredToSevenFiftyMatchTableRows(t *testing.T) {
	for _, id := range []int{effectBottomBlue, effectBottomBlue2} {
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != 20*time.Second || len(spec.components) != 4 {
			t.Fatalf("bottom blue effect %d spec = %+v ok=%t", id, spec, ok)
		}
		for i, component := range spec.components {
			if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second || component.totalCircleSides != 4 || component.circleSides != 4 || !component.rotateWithCamera || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity {
				t.Fatalf("bottom blue effect %d component %d = %+v", id, i, component)
			}
		}
		if first := spec.components[0]; first.bottomSize != 1.5 || first.topSize != 1.5 || first.height != 2 || first.alphaMax != 40.0/255.0 || first.angleY != 0 || first.color != (color.RGBA{R: 51, G: 153, B: 255, A: 255}) {
			t.Fatalf("bottom blue first component = %+v", first)
		}
		if second := spec.components[1]; second.bottomSize != 1.58 || second.height != 2.1 || second.alphaMax != 32.0/255.0 || second.angleY != 10 {
			t.Fatalf("bottom blue second component = %+v", second)
		}
		if third := spec.components[2]; third.bottomSize != 1.65 || third.alphaMax != 15.0/255.0 || third.angleY != 26.6 || third.color != (color.RGBA{R: 25, G: 102, B: 255, A: 255}) {
			t.Fatalf("bottom blue third component = %+v", third)
		}
		if fourth := spec.components[3]; fourth.bottomSize != 1.65 || fourth.alphaMax != 15.0/255.0 || fourth.angleY != 79.8 {
			t.Fatalf("bottom blue fourth component = %+v", fourth)
		}
	}

	judex, ok := worldEffectSpecForID(effectFirePillarOn2)
	if !ok || judex.duration != time.Second || len(judex.components) != 3 {
		t.Fatalf("EF_FIREPILLARON2 spec = %+v ok=%t", judex, ok)
	}
	if len(judex.sfx) != 1 || judex.sfx[0] != "effect\\ab_judex.wav" {
		t.Fatalf("EF_FIREPILLARON2 sfx = %#v", judex.sfx)
	}
	want := []struct {
		bottom float64
		top    float64
		height float64
	}{
		{0.4, 0.5, 3.5},
		{0.45, 0.75, 2.5},
		{0.5, 1, 1.5},
	}
	for i, component := range judex.components {
		if component.kind != effectComponentCylinder || component.textureName != "ring_white" || component.duration != time.Second || component.attachedEntity || !component.rotate || component.bottomSize != want[i].bottom || component.topSize != want[i].top || component.height != want[i].height {
			t.Fatalf("EF_FIREPILLARON2 component %d = %+v", i, component)
		}
	}
}

func TestRobrowserEarthWallEffectSevenHundredToSevenFiftyMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectEarthWall)
	if !ok || spec.duration != time.Second || spec.cameraShake != 200*time.Millisecond || len(spec.components) != 2 {
		t.Fatalf("EF_EARTHWALL spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_EARTHWALL sfx = %#v", spec.sfx)
	}
	horn, quake := spec.components[0], spec.components[1]
	if horn.kind != effectComponentQuadHorn || horn.textureFile != "effect/stone.bmp" || horn.duration != time.Second || horn.quadHornHeightMin != 0.75 || horn.quadHornHeightMax != 1.2 || horn.quadHornOffsetXMin != 0.2 || horn.quadHornOffsetXMax != 0.2 || horn.quadHornOffsetYMin != 0.2 || horn.quadHornOffsetYMax != 0.2 || horn.quadHornOffsetZ != -0.1 || horn.quadHornBottomMin != 0.4 || horn.quadHornBottomMax != 0.9 || horn.blendMode != 1 || horn.quadHornRotateYMin != 1 || horn.quadHornRotateYMax != 360 || horn.quadHornRotateZMin != -8 || horn.quadHornRotateZMax != 8 || horn.animation != 3 || horn.quadHornAnimSpeed != 250*time.Millisecond || !horn.quadHornAnimOut {
		t.Fatalf("EF_EARTHWALL horn = %+v", horn)
	}
	if quake.kind != effectComponentFUNC || quake.funcName != "CameraQuake" || !quake.attachedEntity {
		t.Fatalf("EF_EARTHWALL quake = %+v", quake)
	}
}

func TestRobrowserEffectsSevenFiftyToEightHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POTION_BERSERK2", effectBerserkPotion2, "버서크"},
		{"EF_CRASHAXE", effectCrashAxe, "powerswing"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
	}

	spin, ok := worldEffectSpecForID(effectCastSpin2)
	if !ok || spin.duration != 500*time.Millisecond || len(spin.components) != 1 {
		t.Fatalf("EF_CASTSPIN2 spec = %+v ok=%t", spin, ok)
	}
	if component := spin.components[0]; component.kind != effectComponentFUNC || component.funcName != "CastSpin2" || !component.attachedEntity {
		t.Fatalf("EF_CASTSPIN2 component = %+v", component)
	}

	stasis, ok := worldEffectSpecForID(effectStasis)
	if !ok || stasis.duration != 500*time.Millisecond || len(stasis.components) != 0 {
		t.Fatalf("EF_STASIS spec = %+v ok=%t", stasis, ok)
	}
	if len(stasis.sfx) != 1 || stasis.sfx[0] != "effect\\wl_stasis.wav" {
		t.Fatalf("EF_STASIS sfx = %#v", stasis.sfx)
	}
}

func TestRobrowserGlassWall3EffectMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectGlassWall3)
	if !ok || len(spec.components) != 4 {
		t.Fatalf("EF_GLASSWALL3 spec = %+v ok=%t", spec, ok)
	}
	tint := color.RGBA{R: 153, G: 255, B: 153, A: 255}
	first := spec.components[0]
	if first.kind != effectComponentCylinder || first.textureName != "magic_green" || first.color != tint {
		t.Fatalf("EF_GLASSWALL3 first identity = %+v", first)
	}
	if first.duration != 500*time.Millisecond || first.alphaMax != 0.4 || first.animation != 4 || first.bottomSize != 2.4 || first.topSize != 3.9 || first.height != 0.1 || first.posZ != 0.1 {
		t.Fatalf("EF_GLASSWALL3 first timing/geometry = %+v", first)
	}
	if first.duplicate != 150 || first.duplicateDelay != 200*time.Millisecond || !first.fadeOut || !first.rotate || first.blendMode != 2 || !first.blendAdditive || !first.attachedEntity {
		t.Fatalf("EF_GLASSWALL3 first flags = %+v", first)
	}
	if got := worldEffectComponentDuration(spec, first); got != 30300*time.Millisecond {
		t.Fatalf("EF_GLASSWALL3 first resolved duration = %s, want 30300ms", got)
	}

	for i, want := range []struct {
		bottom float64
		top    float64
		height float64
		alpha  float64
		posZ   float64
		sides  int
		circle int
	}{
		{0.6, 0.6, 7, 0.4, 0, 32, 32},
		{0.8, 0.8, 6, 0.4, 0, 32, 32},
		{1, 1, 1, 0.5, 2, 20, 10},
	} {
		component := spec.components[i+1]
		texture := "magic_green"
		if i == 2 {
			texture = "alpha1"
		}
		if component.kind != effectComponentCylinder || component.textureName != texture || component.color != tint {
			t.Fatalf("EF_GLASSWALL3 component %d identity = %+v", i+1, component)
		}
		if component.duration != 30*time.Second || component.alphaMax != want.alpha || component.bottomSize != want.bottom || component.topSize != want.top || component.height != want.height || component.posZ != want.posZ {
			t.Fatalf("EF_GLASSWALL3 component %d geometry = %+v", i+1, component)
		}
		if !component.fade || !component.rotate || component.blendMode != 2 || !component.blendAdditive || !component.attachedEntity || component.totalCircleSides != want.sides || component.circleSides != want.circle {
			t.Fatalf("EF_GLASSWALL3 component %d flags = %+v", i+1, component)
		}
	}
}

func TestRobrowserRollingCutterCounterEffectsMatchTableRows(t *testing.T) {
	ids := []int{
		effectRolling1,
		effectRolling2,
		effectRolling3,
		effectRolling4,
		effectRolling5,
		effectRolling6,
		effectRolling7,
		effectRolling8,
		effectRolling9,
		effectRolling10,
	}
	for i, id := range ids {
		spec, ok := worldEffectSpecForID(id)
		if !ok || spec.duration != time.Second || len(spec.components) != 5 {
			t.Fatalf("EF_ROLLING%d spec = %+v ok=%t", i+1, spec, ok)
		}
		texture := fmt.Sprintf("effect/회전카운터%d.tga", i+1)
		for j, want := range []struct {
			alpha     float64
			color     color.RGBA
			sizeStart float64
			blendMode int
			additive  bool
		}{
			{1, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 200, 1, false},
			{0.2, color.RGBA{R: 178, G: 178, B: 255, A: 255}, 220, 2, true},
			{0.2, color.RGBA{R: 127, G: 127, B: 255, A: 255}, 240, 2, true},
			{0.2, color.RGBA{R: 76, G: 76, B: 255, A: 255}, 260, 2, true},
			{0.2, color.RGBA{R: 25, G: 25, B: 255, A: 255}, 280, 2, true},
		} {
			component := spec.components[j]
			if component.kind != effectComponent3D || component.textureFile != texture || component.duration != time.Second {
				t.Fatalf("EF_ROLLING%d component %d identity = %+v", i+1, j, component)
			}
			if component.alphaMax != want.alpha || component.color != want.color || component.sizeStart != effectTableSize(want.sizeStart) || component.sizeEnd != effectTableSize(20) {
				t.Fatalf("EF_ROLLING%d component %d visual = %+v", i+1, j, component)
			}
			if !component.fadeIn || !component.fadeOut || component.posZ != 4 || !component.sizeSmooth || component.blendMode != want.blendMode || component.blendAdditive != want.additive || !component.attachedEntity {
				t.Fatalf("EF_ROLLING%d component %d flags = %+v", i+1, j, component)
			}
		}
	}
}

func TestRobrowserBottomBasilica2EffectMatchesTableRow(t *testing.T) {
	spec, ok := worldEffectSpecForID(effectBottomBasilica2)
	if !ok || spec.duration != 20*time.Second || len(spec.components) != 4 {
		t.Fatalf("EF_BOTTOM_BASILICA2 spec = %+v ok=%t", spec, ok)
	}
	if len(spec.sfx) != 1 || spec.sfx[0] != "effect\\wl_whiteimprison.wav" {
		t.Fatalf("EF_BOTTOM_BASILICA2 sfx = %#v", spec.sfx)
	}
	for i, want := range []struct {
		size   float64
		height float64
		alpha  float64
		angleY float64
	}{
		{2.2, 3.0, 65.0 / 255.0, 0},
		{2.25, 3.1, 65.0 / 255.0, 10},
		{2.3, 3.0, 15.0 / 255.0, 0},
		{2.3, 3.0, 15.0 / 255.0, 53.2},
	} {
		component := spec.components[i]
		if component.kind != effectComponentCylinder || component.textureName != "alpha_down" || component.duration != 20*time.Second {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d identity = %+v", i, component)
		}
		if component.totalCircleSides != 4 || component.circleSides != 4 || component.bottomSize != want.size || component.topSize != want.size || component.height != want.height || math.Abs(component.alphaMax-want.alpha) > 0.0001 || component.angleY != want.angleY {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d geometry = %+v", i, component)
		}
		if component.blendMode != 2 || !component.blendAdditive || !component.rotateWithCamera || !component.attachedEntity {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d flags = %+v", i, component)
		}
	}
}

func TestRobrowserEffectsEightHundredToEightFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_ENERVATION", effectEnervation, "enervation", ""},
		{"EF_ENERVATION2", effectEnervation2, "groomy", ""},
		{"EF_ENERVATION3", effectEnervation3, "ignorance", ""},
		{"EF_ENERVATION4", effectEnervation4, "laziness", "effect\\laziness.wav"},
		{"EF_ENERVATION5", effectEnervation5, "unlucky", ""},
		{"EF_ENERVATION6", effectEnervation6, "weakness", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
			continue
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RECOGNIZED", effectRecognized, "effect\\wl_recognizedspell.wav"},
		{"EF_TETRA", effectTetra, "effect\\wl_tetravortex.wav"},
		{"EF_STRETCH", effectStretch, "effect\\bodypaint.wav"},
		{"EF_BOTTOM_MANHOLE", effectBottomManhole, "effect\\dimension.wav"},
		{"EF_MANHOLE", effectManhole, "effect\\manhole.wav"},
		{"EF_FORESTLIGHT6", effectForestLight6, "effect\\dimension.wav"},
		{"EF_BOTTOM_ANI", effectBottomAni, "effect\\chaospanic.wav"},
		{"EF_BOTTOM_MAELSTROM", effectBottomMaelstrom, "effect\\maelstrom.wav"},
		{"EF_BOTTOM_BLOODYLUST", effectBottomBloodyLust, "effect\\bloodylust.wav"},
		{"EF_HEAL_N", effectHealN, "effect\\기공포.wav"},
		{"EF_DANCE1", effectDance1, "effect\\수줍은하루의우울.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	tetraCasting, ok := worldEffectSpecForID(effectTetraCasting)
	if !ok || tetraCasting.duration != 500*time.Millisecond || len(tetraCasting.components) != 1 {
		t.Fatalf("EF_TETRACASTING spec = %+v ok=%t", tetraCasting, ok)
	}
	if component := tetraCasting.components[0]; component.kind != effectComponentFUNC || component.funcName != "TetraCasting" || component.funcAdapter != effectFuncUnknown || !component.attachedEntity {
		t.Fatalf("EF_TETRACASTING component = %+v", component)
	}

	chookgi, ok := worldEffectSpecForID(effectChookgiN)
	if !ok || chookgi.duration != 5*time.Minute || len(chookgi.components) != 1 {
		t.Fatalf("EF_CHOOKGI_N spec = %+v ok=%t", chookgi, ok)
	}
	if component := chookgi.components[0]; component.kind != effectComponentFUNC || component.funcName != "SpiritSphere" || component.funcAdapter != effectFuncSpiritSphere || !component.attachedEntity {
		t.Fatalf("EF_CHOOKGI_N component = %+v", component)
	}
}

func TestRobrowserEffectsEightFiftyToNineHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RAIN_PARTICLE", effectRainParticle, "effect\\rainstorm.wav"},
		{"EF_CHEMICAL_V2", effectChemicalV2, "effect\\안식의자장가.wav"},
		{"EF_CIRCLEPOWER2", effectCirclePower2, "effect\\순환하는자연의소리.wav"},
		{"EF_SPR_PLANT2", effectSprPlant2, "effect\\워그와함께춤을.wav"},
		{"EF_SPR_PLANT3", effectSprPlant3, "effect\\마나의노래.wav"},
		{"EF_SPR_PLANT4", effectSprPlant4, "effect\\새터데이나이트피버.wav"},
		{"EF_SPR_PLANT5", effectSprPlant5, "effect\\레라드의이슬.wav"},
		{"EF_SPR_PLANT6", effectSprPlant6, "effect\\멜로디오브싱크.wav"},
		{"EF_SPR_PLANT7", effectSprPlant7, "effect\\비욘드오브워크라이.wav"},
		{"EF_SPR_PLANT8", effectSprPlant8, "effect\\언리미티드허밍보이스.wav"},
		{"EF_HEARTASURA", effectHeartAsura, "effect\\세이렌의목소리.wav"},
		{"EF_ELECTRIC4", effectElectric4, "effect\\sr_earthshaker.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		wav     string
		tint    color.RGBA
	}{
		{"EF_BOT_REVERB", effectBotReverb, "effect/melody_b.bmp", "effect\\reverberation.wav", color.RGBA{R: 255, G: 153, B: 153, A: 255}},
		{"EF_BOT_REVERB2", effectBotReverb2, "effect/melody_a.bmp", "effect\\나락의노래.wav", color.RGBA{R: 153, G: 153, B: 255, A: 255}},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 100*time.Millisecond || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponent3D || component.textureFile != tc.texture || component.color != tc.tint || component.duration != 100*time.Millisecond || component.alphaMax != 0.6 || !component.attachedEntity || !component.repeat || component.posZ != 0.5 || component.sizeStart != effectTableSize(50) || component.sizeEnd != effectTableSize(50) {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	secra, ok := worldEffectSpecForID(effectSecra2)
	if !ok || secra.duration != 1500*time.Millisecond || len(secra.components) != 10 {
		t.Fatalf("EF_SECRA2 spec = %+v ok=%t", secra, ok)
	}
	if len(secra.sfx) != 1 || secra.sfx[0] != "effect\\ab_ancilla.wav" {
		t.Fatalf("EF_SECRA2 sfx = %#v", secra.sfx)
	}
	firstSecra, lastSecra := secra.components[0], secra.components[len(secra.components)-1]
	if firstSecra.kind != effectComponent3D || firstSecra.textureFile != "effect/priest_spell.bmp" || firstSecra.color != (color.RGBA{R: 255, G: 140, B: 140, A: 255}) || firstSecra.duration != 1500*time.Millisecond || firstSecra.alphaMax != 0.3 || firstSecra.blendMode != 2 || !firstSecra.blendAdditive || !firstSecra.fadeIn || !firstSecra.fadeOut || !firstSecra.attachedEntity || firstSecra.posZ != 7 || firstSecra.sizeStart != effectTableSize(850) || firstSecra.sizeEnd != effectTableSize(100) || !firstSecra.sizeSmooth {
		t.Fatalf("EF_SECRA2 first component = %+v", firstSecra)
	}
	if lastSecra.color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || lastSecra.sizeStart != effectTableSize(400) || lastSecra.sizeEnd != effectTableSize(100) {
		t.Fatalf("EF_SECRA2 last component = %+v", lastSecra)
	}

	glass, ok := worldEffectSpecForID(effectGlassWall4)
	if !ok || glass.duration != 30*time.Second || len(glass.components) != 3 {
		t.Fatalf("EF_GLASSWALL4 spec = %+v ok=%t", glass, ok)
	}
	if len(glass.sfx) != 1 || glass.sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_GLASSWALL4 sfx = %#v", glass.sfx)
	}
	tree, pulseA, pulseB := glass.components[0], glass.components[1], glass.components[2]
	if tree.kind != effectComponent3D || tree.textureFile != "effect/ef_epitree.tga" || tree.color != (color.RGBA{G: 255, A: 255}) || tree.duration != 30*time.Second || tree.alphaMax != 0.6 || tree.blendMode != 2 || !tree.blendAdditive || !tree.attachedEntity || tree.posZ != 7 || tree.sizeStart != effectTableSize(400) || tree.sizeEnd != effectTableSize(400) {
		t.Fatalf("EF_GLASSWALL4 tree = %+v", tree)
	}
	if pulseA.duration != 990*time.Millisecond || pulseA.duplicate != 15 || pulseA.duplicateDelay != 2*time.Second || pulseA.delay != 0 || pulseA.sizeStart != effectTableSize(380) || pulseA.sizeEnd != effectTableSize(420) {
		t.Fatalf("EF_GLASSWALL4 first pulse = %+v", pulseA)
	}
	if pulseB.delay != time.Second || pulseB.sizeStart != effectTableSize(420) || pulseB.sizeEnd != effectTableSize(380) {
		t.Fatalf("EF_GLASSWALL4 second pulse = %+v", pulseB)
	}

	bash, ok := worldEffectSpecForID(effectBash3D6)
	if !ok || bash.duration != 500*time.Millisecond || len(bash.components) != 3 {
		t.Fatalf("EF_BASH3D6 spec = %+v ok=%t", bash, ok)
	}
	if len(bash.sfx) != 1 || bash.sfx[0] != "effect\\bash3d.wav" {
		t.Fatalf("EF_BASH3D6 sfx = %#v", bash.sfx)
	}
	if body := bash.components[0]; body.kind != effectComponentFUNC || body.funcName != "Bash3D6" || body.funcAdapter != effectFuncUnknown || !body.attachedEntity {
		t.Fatalf("EF_BASH3D6 body = %+v", body)
	}
	for i, component := range bash.components[1:] {
		wantTop := 4.5
		if i == 1 {
			wantTop = 7.2
		}
		if component.kind != effectComponentCylinder || component.textureName != "alpha_center" || component.color != (color.RGBA{R: 76, G: 127, B: 255, A: 255}) || component.duration != 175*time.Millisecond || component.delay != 200*time.Millisecond || component.duplicate != 5 || component.alphaMax != 0.6 || !component.fade || component.angleX != -90 || component.angleZRandom != 360 || !component.fixedPerspective || component.posZ != 1.5 || component.bottomSize != 0.01 || component.topSize != wantTop || component.animation != 2 || !component.attachedEntity {
			t.Fatalf("EF_BASH3D6 cylinder %d = %+v", i, component)
		}
	}

	teiHit, ok := worldEffectSpecForID(effectTeiHit1T)
	if !ok || teiHit.duration != 350*time.Millisecond || len(teiHit.components) != 1 {
		t.Fatalf("EF_TEIHIT1T spec = %+v ok=%t", teiHit, ok)
	}
	if len(teiHit.sfx) != 1 || teiHit.sfx[0] != "effect\\mon_아수라 패황권.wav" {
		t.Fatalf("EF_TEIHIT1T sfx = %#v", teiHit.sfx)
	}
	component := teiHit.components[0]
	if component.kind != effectComponent3D || component.textureFile != "effect/lens1.tga" || component.color != (color.RGBA{R: 25, G: 25, B: 255, A: 255}) || component.duration != 250*time.Millisecond || component.delay != 100*time.Millisecond || component.duplicate != 24 || component.duplicateDelay != 0 || component.alphaMax != 0.8 || component.blendMode != 2 || !component.blendAdditive || !component.fadeIn || !component.fadeOut || !component.attachedEntity || component.posXEndRand != 40 || component.posYEndRand != 40 || component.sizeStartX != effectTableSize(10) || component.sizeStartY != effectTableSize(150) || component.sizeEndX != effectTableSize(10) || component.sizeEndY != effectTableSize(150) || !component.overlay || !component.rotateToTarget || !component.rotateWithCamera {
		t.Fatalf("EF_TEIHIT1T component = %+v", component)
	}
}

func TestRobrowserEffectsNineHundredToNineFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_PRIMECHARGE2", effectPrimeCharge2, "effect\\lg_prestige.wav"},
		{"EF_PRIMECHARGE3", effectPrimeCharge3, "effect\\lg_banding.wav"},
		{"EF_PRIMECHARGE4", effectPrimeCharge4, "effect\\lg_inspiration.wav"},
		{"EF_SPR_PLANT10", effectSprPlant10, "effect\\s사이킥웨이브.wav"},
		{"EF_COLDTHROW2", effectColdThrow2, "effect\\wl_jackfrost.wav"},
		{"EF_DEMONICFIRE4", effectDemonicFire4, "effect\\s워머.wav"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 500*time.Millisecond || len(spec.components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_FIREWALL2", effectFireWall2, "firewall_per"},
		{"EF_SHOCKWAVE2", effectShockwave2, "hunter_shockwave_blue"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || len(spec.sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.components[0]; component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		sfx     []string
		delays  []time.Duration
	}{
		{"EF_PRESSURE2", effectPressure2, "effect/shield.bmp", []string{"effect\\프레셔.wav", "effect\\lg_shieldpress.wav"}, []time.Duration{0, 500 * time.Millisecond}},
		{"EF_PRESSURE3", effectPressure3, "effect/cross1.bmp", []string{"effect\\프레셔.wav"}, nil},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 1001*time.Millisecond || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if spec.cameraShake != 0 || spec.cameraShakeDelay != 0 {
			t.Fatalf("%s camera shake = %s delay %s, want none", tc.name, spec.cameraShake, spec.cameraShakeDelay)
		}
		if !reflect.DeepEqual(spec.sfx, tc.sfx) || !reflect.DeepEqual(spec.sfxDelays, tc.delays) {
			t.Fatalf("%s sfx = %#v delays %#v", tc.name, spec.sfx, spec.sfxDelays)
		}

		first, second := spec.components[0], spec.components[1]
		if first.kind != effectComponent3D || first.textureFile != tc.texture || first.duration != 500*time.Millisecond || first.alphaMax != 0.6 || first.blendMode != 2 || !first.blendAdditive || !first.rotate || first.angleStart != 0 || first.angleEnd != -611 || first.posZ != 20 || first.posZEnd != 5 || first.sizeStart != effectTableSize(100) || first.sizeEnd != effectTableSize(100) || !first.attachedEntity {
			t.Fatalf("%s first component = %+v", tc.name, first)
		}
		if second.kind != effectComponent3D || second.textureFile != tc.texture || second.duration != 500*time.Millisecond || second.delay != 501*time.Millisecond || second.alphaMax != 0.6 || second.blendMode != 2 || !second.blendAdditive || !second.fadeOut || second.angleStart != -611 || second.angleEnd != -611 || second.posZ != 5 || second.sizeStart != effectTableSize(100) || second.sizeEnd != effectTableSize(100) || !second.attachedEntity {
			t.Fatalf("%s second component = %+v", tc.name, second)
		}
	}
}

func TestRobrowserEffectsNineFiftyToOneThousandMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POISON_MIST", effectPoisonMist, "poison_mist"},
		{"EF_ERASER_CUTTER", effectEraserCutter, "eraser_cutter"},
		{"EF_LAVA_SLIDE", effectLavaSlide, "lava_slide"},
		{"EF_SONIC_CLAW", effectSonicClaw, "sonic_claw"},
		{"EF_TINDER_BREAKER", effectTinderBreaker, "tinder"},
		{"EF_MIDNIGHT_FRENZY", effectMidnightFrenzy, "mid_frenzy"},
		{"EF_VOLCANIC_ASH", effectVolcanicAsh, "vash00"},
		{"EF_2011RWC", effectRWC2011, "rwc2011"},
		{"EF_2011RWC2", effectRWC2011Two, "rwc2011_2"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 || len(spec.sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.components[0]; component.kind != effectComponentSTR || component.strFile != tc.file || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsOneThousandToTenFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		id      int
		file    string
		wav     string
		randMin int
		randMax int
	}{
		{"EF_RUN_MAKE_OK", effectRunMakeOK, "rune_success", "", 0, 0},
		{"EF_RUN_MAKE_FAILURE", effectRunMakeFailure, "rune_fail", "", 0, 0},
		{"EF_MIRESULT_MAKE_OK", effectMIResultMakeOK, "changematerial_su", "", 0, 0},
		{"EF_MIRESULT_MAKE_FAIL", effectMIResultMakeFail, "changematerial_fa", "", 0, 0},
		{"EF_ALL_RAY_OF_PROTECTION", effectAllRayProtect, "guardian", "", 0, 0},
		{"EF_VENOMFOG", effectVenomFog, "bubble%d_1", "", 1, 4},
		{"EF_DUSTSTORM", effectDustStorm, "dust", "", 0, 0},
		{"EF_DANCE_BLADE_ATK", effectDanceBladeAtk, "dancingblade", "", 0, 0},
		{"EF_INVINCIBLEOFF2", effectInvincibleOff2, "invincibleoff2", "", 0, 0},
		{"EF_DEATHSUMMON", effectDeathSummon, "devil", "", 0, 0},
		{"EF_GC_DARKCROW", effectGCDarkCrow, "gc_darkcrow", "", 0, 0},
		{"EF_ALL_FULL_THROTTLE", effectAllFullThrottle, "all_full_throttle", "effect\\all_full_throttle.wav", 0, 0},
		{"EF_SR_FLASHCOMBO", effectSRFlashCombo, "sr_flashcombo", "effect\\sr_flashcombo.wav", 0, 0},
		{"EF_RK_LUXANIMA", effectRKLuxAnima, "rk_luxanima", "", 0, 0},
		{"EF_SO_ELEMENTAL_SHIELD", effectSOElemShield, "so_elemental_shield", "effect\\so_elemental_shield.wav", 0, 0},
		{"EF_AB_OFFERTORIUM", effectABOffertorium, "ab_offertorium", "effect\\ab_offertorium.wav", 0, 0},
		{"EF_WL_TELEKINESIS_INTENSE", effectWLTelekinesis, "wl_telekinesis_intense", "effect\\wl_telekinesis_intense.wav", 0, 0},
		{"EF_GN_ILLUSIONDOPING", effectGNIllusionDoping, "gn_illusiondoping", "effect\\gn_illusiondoping.wav", 0, 0},
		{"EF_NC_MAGMA_ERUPTION", effectNCMagmaEruption, "nc_magma_eruption", "effect\\nc_magma_eruption.wav", 0, 0},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
		} else if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.strRandMin != tc.randMin || component.strRandMax != tc.randMax || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsTenFiftyToElevenHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name        string
		id          int
		file        string
		texturePath string
		wav         string
	}{
		{"EF_NPC_CHILL", effectNPCChill, "chill", "", ""},
		{"EF_AB_OFFERTORIUM_RING", effectOffertoriumRing, "ab_offertorium_ring", "", ""},
		{"EF_HAMMER_OF_GOD", effectHammerOfGod, "stormgust", "", "effect\\RL_HAMMER_OF_GOD.wav"},
		{"EF_ACH_COMPLETE", effectAchComplete, "ach_complete/ppring3", "ach_complete/", ""},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
			}
		} else if len(spec.sfx) != 1 || spec.sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, tc.wav)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.texturePath != tc.texturePath || !component.attachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsPostElevenHundredMatchTableRows(t *testing.T) {
	body, ok := worldEffectSpecForID(effectBodyColor)
	if !ok || body.duration != 300*time.Millisecond || len(body.components) != 1 {
		t.Fatalf("EffectBodyColor spec = %+v ok=%t, want 300ms FUNC component", body, ok)
	}
	bodyComponent := body.components[0]
	if bodyComponent.kind != effectComponentFUNC || bodyComponent.funcName != "EffectBodyColor" || bodyComponent.funcAdapter != effectFuncBodyColor || !bodyComponent.attachedEntity {
		t.Fatalf("EffectBodyColor component = %+v", bodyComponent)
	}

	for _, tc := range []struct {
		name         string
		id           int
		file         string
		texturePath  string
		head         bool
		yOffset      float64
		renderBefore bool
	}{
		{"EF_BAKURETSU_HADOU", effectBakuretsuHadou, "bakuretsu_hadou/bakuretsu_hadou", "bakuretsu_hadou/", true, -50, false},
		{"EF_DIGITAL_SPACE", effectDigitalSpace, "digital_space/digital_space", "digital_space/", false, 0, true},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || spec.duration != 5*time.Minute || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want 5m SPR component", tc.name, spec, ok)
		}
		component := spec.components[0]
		if component.kind != effectComponentSPR || component.spriteFile != tc.file || component.texturePath != tc.texturePath || !component.attachedEntity {
			t.Fatalf("%s component = %+v, want SPR %q texturePath %q attached", tc.name, component, tc.file, tc.texturePath)
		}
		if !component.repeat || !component.spriteRepeat || component.spriteHead != tc.head || component.spriteYOffset != tc.yOffset || component.renderBefore != tc.renderBefore {
			t.Fatalf("%s component flags = %+v", tc.name, component)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
	}

	for _, tc := range []struct {
		name         string
		id           int
		colorName    string
		renderBefore bool
	}{
		{"DROPEFFECT_PINK", dropEffectPink, "pink", true},
		{"DROPEFFECT_YELLOW", dropEffectYellow, "yellow", false},
		{"DROPEFFECT_PURPLE", dropEffectPurple, "purple", false},
		{"DROPEFFECT_BLUE", dropEffectBlue, "blue", false},
		{"DROPEFFECT_GREEN", dropEffectGreen, "green", false},
		{"DROPEFFECT_RED", dropEffectRed, "red", false},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two STR components", tc.name, spec, ok)
		}
		wantSFX := "effect\\drop_" + tc.colorName + ".wav"
		if len(spec.sfx) != 1 || spec.sfx[0] != wantSFX {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.sfx, wantSFX)
		}
		wantFiles := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName,
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/dropitem_" + tc.colorName + "_bottom",
		}
		wantTexturePaths := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/",
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/",
		}
		for i, component := range spec.components {
			if component.kind != effectComponentSTR || component.strFile != wantFiles[i] || component.texturePath != wantTexturePaths[i] {
				t.Fatalf("%s component %d = %+v, want STR %q texturePath %q", tc.name, i, component, wantFiles[i], wantTexturePaths[i])
			}
			if component.attachedEntity || component.renderBefore != tc.renderBefore {
				t.Fatalf("%s component %d flags = %+v", tc.name, i, component)
			}
		}
	}

	for _, tc := range []struct {
		name        string
		id          int
		file        string
		texturePath string
	}{
		{"EF_NEW_SUCCESS", effectNewSuccess, "grade_enchant/new_success/new_success", "grade_enchant/new_success/"},
		{"EF_NEW_FAILURE", effectNewFailure, "grade_enchant/new_failed/new_failed", "grade_enchant/new_failed/"},
		{"EF_NEW_INTRO", effectNewIntro, "grade_enchant/new_intro/new_intro", "grade_enchant/new_intro/"},
		{"EF_UI_ENCHANT_INTRO_YELLOW", effectEnchantYellow, "ui_enchant/ui_intro_yellow/ui_intro_yellow", "ui_enchant/ui_intro_yellow/"},
		{"EF_UI_ENCHANT_SUCCESS", effectEnchantSuccess, "ui_enchant/ui_enchant_success/ui_enchant_success", "ui_enchant/ui_enchant_success/"},
		{"EF_UI_ENCHANT_FAIL", effectEnchantFail, "ui_enchant/ui_fail/ui_enchant_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_BLUE", effectEnchantBlue, "ui_enchant/ui_intro_blue/ui_intro_blue", "ui_enchant/ui_intro_blue/"},
		{"EF_UI_ENCHANT_UP_SUCCESS", effectEnchantUpSuccess, "ui_enchant/ui_levelup_success/ui_levelup_success", "ui_enchant/ui_levelup_success/"},
		{"EF_UI_ENCHANT_UP_FAIL", effectEnchantUpFail, "ui_enchant/ui_fail/ui_levelup_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_GREEN", effectEnchantGreen, "ui_enchant/ui_intro_green/ui_intro_green", "ui_enchant/ui_intro_green/"},
		{"EF_UI_ENCHANT_RESET_SUCCESS", effectEnchantResetOK, "ui_enchant/ui_reset_success/ui_reset_success", "ui_enchant/ui_reset_success/"},
		{"EF_UI_ENCHANT_RESET_FAIL", effectEnchantResetFail, "ui_enchant/ui_fail/ui_reset_fail", "ui_enchant/ui_fail/"},
	} {
		spec, ok := worldEffectSpecForID(tc.id)
		if !ok || len(spec.components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if len(spec.sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.sfx)
		}
		component := spec.components[0]
		if component.kind != effectComponentSTR || component.strFile != tc.file || component.texturePath != tc.texturePath || component.attachedEntity || component.renderBefore {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}
