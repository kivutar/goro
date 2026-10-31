package effects

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/internal/testutil"
)

func TestWorldEffectSpecCatalogCoverage(t *testing.T) {
	coverage := testutil.EffectCoverageSnapshot()
	if coverage.Implemented != 750 {
		t.Fatalf("implemented effects = %d, want 750", coverage.Implemented)
	}
	if coverage.ReferenceActive != 607 || coverage.ReferenceAll != 1147 {
		t.Fatalf("reference client totals = active %d all %d", coverage.ReferenceActive, coverage.ReferenceAll)
	}
	if coverage.ActivePercent < 123.55 || coverage.ActivePercent > 123.56 {
		t.Fatalf("active coverage = %.3f, want about 123.56", coverage.ActivePercent)
	}
}

func TestRobrowserActiveEffectsZeroToFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		Hit1:           "EF_HIT1",
		BashHit:        "EF_HIT2",
		Hit3:           "EF_HIT3",
		Hit4:           "EF_HIT4",
		Hit5:           "EF_HIT5",
		Hit6:           "EF_HIT6",
		Entry:          "EF_ENTRY",
		Exit:           "EF_EXIT",
		Warp:           "EF_WARP",
		Enhance:        "EF_ENHANCE",
		Mammonite:      "EF_COIN",
		Endure:         "EF_ENDURE",
		BeginSpell:     "EF_BEGINSPELL",
		GlassWall:      "EF_GLASSWALL",
		HealSP:         "EF_HEALSP",
		SoulStrike:     "EF_SOULSTRIKE",
		BashBegin:      "EF_BASH",
		MagnumBreak:    "EF_MAGNUMBREAK",
		Steal:          "EF_STEAL",
		PoisonAttack:   "EF_PATTACK",
		Detoxication:   "EF_DETOXICATION",
		Sight:          "EF_SIGHT",
		StoneCurse:     "EF_STONECURSE",
		FireBall:       "EF_FIREBALL",
		FireWall:       "EF_FIREWALL",
		IceArrow:       "EF_ICEARROW",
		FrostDiver:     "EF_FROSTDIVER",
		FrostDiverHit:  "EF_FROSTDIVER2",
		LightningBolt:  "EF_LIGHTBOLT",
		ThunderStorm:   "EF_THUNDERSTORM",
		FireArrow:      "EF_FIREARROW",
		NapalmBeat:     "EF_NAPALMBEAT",
		Ruwach:         "EF_RUWACH",
		TeleportOld:    "EF_TELEPORTATION",
		ReadyPortalOld: "EF_READYPORTAL",
		IncAgility:     "EF_INCAGILITY",
		DecAgility:     "EF_DECAGILITY",
		Aqua:           "EF_AQUA",
		Signum:         "EF_SIGNUM",
		Angelus:        "EF_ANGELUS",
		Blessing:       "EF_BLESSING",
		IncAgiDex:      "EF_INCAGIDEX",
		Smoke:          "EF_SMOKE",
		Firefly:        "EF_FIREFLY",
		Torch:          "EF_TORCH",
		FireHit:        "EF_FIREHIT",
		FireSplashHit:  "EF_FIRESPLASHHIT",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiftyToOneHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		FireSplashHit:  "EF_FIRESPLASHHIT",
		ColdHit:        "EF_COLDHIT",
		WindHit:        "EF_WINDHIT",
		PoisonHit:      "EF_POISONHIT",
		BeginSpell2:    "EF_BEGINSPELL2",
		BeginSpell3:    "EF_BEGINSPELL3",
		BeginSpell4:    "EF_BEGINSPELL4",
		BeginSpell5:    "EF_BEGINSPELL5",
		BeginSpell6:    "EF_BEGINSPELL6",
		BeginSpell7:    "EF_BEGINSPELL7",
		LockOnTarget:   "EF_LOCKON",
		WarpZone:       "EF_WARPZONE",
		SightTrasher:   "EF_SIGHTRASHER",
		ArrowShotRO:    "EF_ARROWSHOT",
		Invenom:        "EF_INVENOM",
		Cure:           "EF_CURE",
		Provoke:        "EF_PROVOKE",
		Mvp:            "EF_MVP",
		SkidTrap:       "EF_SKIDTRAP",
		BrandishSpear:  "EF_BRANDISHSPEAR",
		IceWall:        "EF_ICEWALL",
		Gloria:         "EF_GLORIA",
		Magnificat:     "EF_MAGNIFICAT",
		Resurrection:   "EF_RESURRECTION",
		Recovery:       "EF_RECOVERY",
		EarthSpike:     "EF_EARTHSPIKE",
		SpearBoomerang: "EF_SPEARBMR",
		Pierce:         "EF_PIERCE",
		TurnUndead:     "EF_TURNUNDEAD",
		Sanctuary:      "EF_SANCTUARY",
		Impositio:      "EF_IMPOSITIO",
		LexAeterna:     "EF_LEXAETERNA",
		Aspersio:       "EF_ASPERSIO",
		LexDivina:      "EF_LEXDIVINA",
		Suffragium:     "EF_SUFFRAGIUM",
		StormGust:      "EF_STORMGUST",
		LordVermilion:  "EF_LORD",
		Benedictio:     "EF_BENEDICTIO",
		MeteorStorm:    "EF_METEORSTORM",
		JupitelThunder: "EF_YUFITEL",
		JupitelHit:     "EF_YUFITELHIT",
		Quagmire:       "EF_QUAGMIRE",
		FirePillar:     "EF_FIREPILLAR",
		FirePillarBomb: "EF_FIREPILLARBOMB",
		HasteUp:        "EF_HASTEUP",
		Flasher:        "EF_FLASHER",
		RemoveTrap:     "EF_REMOVETRAP",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
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
		{"EF_ARROWSHOT", ArrowShotRO, "arrowshot", "", true},
		{"EF_INVENOM", Invenom, "invenom", "effect\\thief_invenom.wav", true},
		{"EF_SKIDTRAP", SkidTrap, "skidtrap", "effect\\hunter_skidtrap.wav", false},
		{"EF_BRANDISHSPEAR", BrandishSpear, "brandish", "", true},
		{"EF_RECOVERY", Recovery, "recovery", "effect\\priest_recovery.wav", true},
		{"EF_SANCTUARY", Sanctuary, "sanctuary", "effect\\priest_sanctuary.wav", true},
		{"EF_IMPOSITIO", Impositio, "impositio", "effect\\priest_impositio.wav", true},
		{"EF_ASPERSIO", Aspersio, "aspersio", "effect\\priest_aspersio.wav", true},
		{"EF_LEXDIVINA", LexDivina, "lexdivina", "effect\\priest_lexdivina.wav", true},
		{"EF_LORD", LordVermilion, "lord", "effect\\wizard_fire_ivy.wav", true},
		{"EF_BENEDICTIO", Benedictio, "benedictio", "effect\\priest_benedictio.wav", true},
		{"EF_QUAGMIRE", Quagmire, "quagmire", "effect\\wizard_quagmire.wav", false},
		{"EF_FIREPILLAR", FirePillar, "firepillar", "effect\\wizard_fire_pillar_a.wav", false},
		{"EF_FIREPILLARBOMB", FirePillarBomb, "firepillarbomb", "effect\\wizard_fire_pillar_b.wav", false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

}

func TestRobrowserWarpZoneAndPoisonHitSpecs(t *testing.T) {
	poison, ok := Lookup(PoisonHit)
	if !ok || len(poison.Components) != 1 {
		t.Fatalf("EF_POISONHIT spec = %+v ok=%t, want one SPR component", poison, ok)
	}
	if len(poison.Sfx) != 1 || poison.Sfx[0] != "effect\\ef_poisonattack.wav" {
		t.Fatalf("EF_POISONHIT sfx = %v", poison.Sfx)
	}
	if component := poison.Components[0]; component.Kind != ComponentSPR || component.SpriteFile != "poisonhit" || component.AttachedEntity {
		t.Fatalf("EF_POISONHIT component = %+v", component)
	}

	warp, ok := Lookup(WarpZone)
	if !ok || len(warp.Components) != 3 || warp.Duration != 2800*time.Millisecond {
		t.Fatalf("EF_WARPZONE spec = %+v ok=%t", warp, ok)
	}
	first, second, particle := warp.Components[0], warp.Components[1], warp.Components[2]
	if first.Kind != ComponentCylinder || second.Kind != ComponentCylinder || first.TextureName != "ring_blue" || second.TextureName != "ring_blue" {
		t.Fatalf("EF_WARPZONE cylinders = %+v %+v", first, second)
	}
	if first.BottomSize != 2 || first.TopSize != 3.3 || second.BottomSize != 1.9 || second.TopSize != 3.2 || first.Height != 1.1 || second.Height != 1.1 {
		t.Fatalf("EF_WARPZONE cylinder sizes = %+v %+v", first, second)
	}
	if !first.AttachedEntity || !second.AttachedEntity || !first.BlendAdditive || !second.BlendAdditive || !first.Fade || !second.Fade || !first.Rotate || !second.Rotate || first.Animation != 3 || second.Animation != 3 {
		t.Fatalf("EF_WARPZONE cylinder flags = %+v %+v", first, second)
	}
	if particle.Kind != Component3D || particle.TextureFile != "effect/pok1.tga" || particle.Duration != time.Second || particle.Duplicate != 3 || !particle.AttachedEntity {
		t.Fatalf("EF_WARPZONE particle = %+v", particle)
	}
	if particle.PosXStartRand != 3 || particle.PosYStartRand != 3 || particle.PosZEndRand != 2 || particle.PosZEndMiddle != 2 || particle.SizeStart != testutil.EffectTableSize(100) || particle.SizeRand != testutil.EffectTableSize(17) {
		t.Fatalf("EF_WARPZONE particle motion/size = %+v", particle)
	}
}

func TestMeteorSpec(t *testing.T) {
	meteor, ok := Lookup(MeteorStorm)
	if !ok || len(meteor.Components) != 1 {
		t.Fatalf("EF_METEORSTORM spec = %+v ok=%t", meteor, ok)
	}
	if meteor.CameraShakeDelay != 40*db.EffectFrameDuration || meteor.CameraShake != 650*time.Millisecond {
		t.Fatalf("EF_METEORSTORM camera shake = delay %s duration %s", meteor.CameraShakeDelay, meteor.CameraShake)
	}
	if len(meteor.Sfx) != 1 || meteor.Sfx[0] != "effect\\wizard_meteor.wav" {
		t.Fatalf("EF_METEORSTORM sfx = %v", meteor.Sfx)
	}
	component := meteor.Components[0]
	if component.Kind != ComponentSTR || component.StrFile != "meteor%d" || component.StrRandMin != 1 || component.StrRandMax != 4 || !component.AttachedEntity {
		t.Fatalf("EF_METEORSTORM component = %+v", component)
	}

}

func TestRobrowserActiveEffectsOneHundredToOneFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		RemoveTrap:     "EF_REMOVETRAP",
		RepairWeapon:   "EF_REPAIRWEAPON",
		CrashEarth:     "EF_CRASHEARTH",
		WeaponPerfect:  "EF_PERFECTION",
		MaximizePower:  "EF_MAXPOWER",
		BlastMine:      "EF_BLASTMINE",
		BlastMineBomb:  "EF_BLASTMINEBOMB",
		Claymore:       "EF_CLAYMORE",
		FreezingTrap:   "EF_FREEZING",
		Bubble:         "EF_BUBBLE",
		GasPush:        "EF_GASPUSH",
		SpringTrap:     "EF_SPRINGTRAP",
		Kyrie:          "EF_KYRIE",
		Magnus:         "EF_MAGNUS",
		BlitzBeat:      "EF_BLITZBEAT",
		WaterBall:      "EF_WATERBALL",
		WaterBall2:     "EF_WATERBALL2",
		Detecting:      "EF_DETECTING",
		Cloaking:       "EF_CLOAKING",
		SonicBlow:      "EF_SONICBLOW",
		SonicBlowHit:   "EF_SONICBLOWHIT",
		Grimtooth:      "EF_GRIMTOOTH",
		VenomDust:      "EF_VENOMDUST",
		PoisonReact:    "EF_POISONREACT",
		PoisonReact2:   "EF_POISONREACT2",
		Overthrust:     "EF_OVERTHRUST",
		VenomSplasher:  "EF_SPLASHER",
		TwoHandQuicken: "EF_TWOHANDQUICKEN",
		AutoCounter:    "EF_AUTOCOUNTER",
		GrimtoothAtk:   "EF_GRIMTOOTHATK",
		Freeze:         "EF_FREEZE",
		Freezed:        "EF_FREEZED",
		IceCrash:       "EF_ICECRASH",
		SlowPoison:     "EF_SLOWPOISON",
		FirePillarOn:   "EF_FIREPILLARON",
		Sandman:        "EF_SANDMAN",
		Revive:         "EF_REVIVE",
		Pneuma:         "EF_PNEUMA",
		HeavenDrive:    "EF_HEAVENDRIVE",
		SonicBlow2:     "EF_SONICBLOW2",
		BrandishSpear2: "EF_BRANDISH2",
		Shockwave:      "EF_SHOCKWAVE",
		ShockwaveHit:   "EF_SHOCKWAVEHIT",
		EarthHit:       "EF_EARTHHIT",
		PierceSelf:     "EF_PIERCESELF",
		BowlingSelf:    "EF_BOWLINGSELF",
		SpearStabSelf:  "EF_SPEARSTABSELF",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
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
		{"EF_BLASTMINEBOMB", BlastMineBomb, "blastmine", "effect\\hunter_blastmine.wav", false, false},
		{"EF_CLAYMORE", Claymore, "claymore", "effect\\hunter_claymoretrap.wav", false, false},
		{"EF_FREEZING", FreezingTrap, "freezing", "effect\\hunter_freezingtrap.wav", false, false},
		{"EF_GASPUSH", GasPush, "gaspush", "", false, false},
		{"EF_SPRINGTRAP", SpringTrap, "spring", "effect\\hunter_springtrap.wav", false, false},
		{"EF_MAGNUS", Magnus, "magnus", "effect\\priest_magnus.wav", false, false},
		{"EF_VENOMDUST", VenomDust, "venomdust", "effect\\assasin_poisonreact.wav", false, false},
		{"EF_POISONREACT", PoisonReact, "poisonreact_1st", "", true, false},
		{"EF_POISONREACT2", PoisonReact2, "poisonreact", "effect\\assasin_poisonreact.wav", true, false},
		{"EF_SPLASHER", VenomSplasher, "venomsplasher", "effect\\assasin_venomsplasher.wav", true, false},
		{"EF_TWOHANDQUICKEN", TwoHandQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, false},
		{"EF_AUTOCOUNTER", AutoCounter, "autocounter", "effect\\knight_autocounter.wav", true, false},
		{"EF_FREEZE", Freeze, "freeze", "", true, false},
		{"EF_FREEZED", Freezed, "freezed", "", true, false},
		{"EF_ICECRASH", IceCrash, "icecrash", "", true, false},
		{"EF_SLOWPOISON", SlowPoison, "slowp", "effect\\priest_slowpoison.wav", false, false},
		{"EF_SANDMAN", Sandman, "sandman", "effect\\hunter_sandman.wav", false, false},
		{"EF_SONICBLOW2", SonicBlow2, "sonicblow", "", true, false},
		{"EF_BRANDISH2", BrandishSpear2, "brandish2", "", true, false},
		{"EF_SHOCKWAVEHIT", ShockwaveHit, "shockwavehit", "", true, false},
		{"EF_EARTHHIT", EarthHit, "earthhit", "", true, false},
		{"EF_PIERCESELF", PierceSelf, "pierce", "", true, false},
		{"EF_BOWLINGSELF", BowlingSelf, "bowling", "", true, false},
		{"EF_SPEARSTABSELF", SpearStabSelf, "spearstab", "", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_BLASTMINE", BlastMine, "effect\\hun_anklesnare.wav"},
		{"EF_CLOAKING", Cloaking, "effect\\assasin_cloaking.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 0 || spec.Duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}
}

func TestRobrowserActiveEffectsOneFiftyToTwoHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		SpearStabSelf:  "EF_SPEARSTABSELF",
		SpearBmrSelf:   "EF_SPEARBMRSELF",
		HolyLight:      "EF_HOLYHIT",
		Concentration:  "EF_CONCENTRATION",
		RefineOK:       "EF_REFINEOK",
		RefineFail:     "EF_REFINEFAIL",
		JobLevelUp:     "EF_JOBLVUP",
		Rain:           "EF_RAIN",
		Snow:           "EF_SNOW",
		Sakura:         "EF_SAKURA",
		Banjjakii:      "EF_BANJJAKII",
		MakeBlur:       "EF_MAKEBLUR",
		EnergyCoat:     "EF_ENERGYCOAT",
		CartRevolution: "EF_CARTREVOLUTION",
		VenomDust2:     "EF_VENOMDUST2",
		MentalBreak:    "EF_MENTALBREAK",
		MagicalAtkHit:  "EF_MAGICALATTHIT",
		SuiExplosion:   "EF_SUI_EXPLOSION",
		Suicide:        "EF_SUICIDE",
		ComboAttack1:   "EF_COMBOATTACK1",
		ComboAttack2:   "EF_COMBOATTACK2",
		ComboAttack3:   "EF_COMBOATTACK3",
		ComboAttack4:   "EF_COMBOATTACK4",
		ComboAttack5:   "EF_COMBOATTACK5",
		GuidedAttack:   "EF_GUIDEDATTACK",
		PoisonAttack2:  "EF_POISONATTACK",
		SilenceAttack:  "EF_SILENCEATTACK",
		StunAttack:     "EF_STUNATTACK",
		PetrifyAttack:  "EF_PETRIFYATTACK",
		SleepAttack:    "EF_SLEEPATTACK",
		Pong:           "EF_PONG",
		Level99:        "EF_LEVEL99",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
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
		{"EF_SPEARBMRSELF", SpearBmrSelf, "spearboomerang", "", true, false},
		{"EF_HOLYHIT", HolyLight, "holyhit", "", true, false},
		{"EF_CONCENTRATION", Concentration, "concentration", "effect\\ac_concentration.wav", true, false},
		{"EF_REFINEOK", RefineOK, "bs_refinesuccess", "effect\\bs_refinesuccess.wav", true, false},
		{"EF_REFINEFAIL", RefineFail, "bs_refinefailed", "effect\\bs_refinefailed.wav", true, false},
		{"EF_ENERGYCOAT", EnergyCoat, "energycoat", "", true, false},
		{"EF_MENTALBREAK", MentalBreak, "mentalbreak", "", true, false},
		{"EF_MAGICALATTHIT", MagicalAtkHit, "magical", "", true, false},
		{"EF_SUICIDE", Suicide, "suicide", "", true, false},
		{"EF_COMBOATTACK1", ComboAttack1, "yunta_1", "", true, false},
		{"EF_COMBOATTACK2", ComboAttack2, "yunta_2", "", true, false},
		{"EF_COMBOATTACK3", ComboAttack3, "yunta_3", "", true, false},
		{"EF_COMBOATTACK4", ComboAttack4, "yunta_4", "", true, false},
		{"EF_COMBOATTACK5", ComboAttack5, "yunta_5", "", true, false},
		{"EF_GUIDEDATTACK", GuidedAttack, "homing", "", true, false},
		{"EF_POISONATTACK", PoisonAttack2, "poison", "", true, false},
		{"EF_SILENCEATTACK", SilenceAttack, "silence", "", true, false},
		{"EF_STUNATTACK", StunAttack, "stun", "", true, false},
		{"EF_PETRIFYATTACK", PetrifyAttack, "stonecurse", "", true, false},
		{"EF_SLEEPATTACK", SleepAttack, "sleep", "", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsOneFiftyToTwoHundredMatchTableRows(t *testing.T) {
	banjjakii, ok := Lookup(Banjjakii)
	if !ok || banjjakii.Duration != time.Second || len(banjjakii.Components) != 1 {
		t.Fatalf("EF_BANJJAKII spec = %+v ok=%t", banjjakii, ok)
	}
	if component := banjjakii.Components[0]; component.Kind != ComponentSPR || component.SpriteFile != "크리스마스" || component.Duration != time.Second || !component.AttachedEntity {
		t.Fatalf("EF_BANJJAKII component = %+v", component)
	}

	makeBlur, ok := Lookup(MakeBlur)
	if !ok || makeBlur.Duration != 2*time.Second || len(makeBlur.Components) != 1 {
		t.Fatalf("EF_MAKEBLUR spec = %+v ok=%t", makeBlur, ok)
	}
	if component := makeBlur.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "MakeBlur" || component.FuncAdapter != EffectFuncUnknown || component.AttachedEntity {
		t.Fatalf("EF_MAKEBLUR component = %+v", component)
	}

	sui, ok := Lookup(SuiExplosion)
	if !ok || len(sui.Components) != 2 || sui.CameraShake != 200*time.Millisecond {
		t.Fatalf("EF_SUI_EXPLOSION spec = %+v ok=%t", sui, ok)
	}
	if len(sui.Sfx) != 1 || sui.Sfx[0] != "effect\\ef_hit2.wav" {
		t.Fatalf("EF_SUI_EXPLOSION sfx = %v", sui.Sfx)
	}
	if str, quake := sui.Components[0], sui.Components[1]; str.Kind != ComponentSTR || str.StrFile != "sui_explosion" || !str.AttachedEntity || quake.Kind != ComponentFUNC || quake.FuncName != "CameraQuake" || !quake.AttachedEntity {
		t.Fatalf("EF_SUI_EXPLOSION components = %+v %+v", str, quake)
	}

	pong, ok := Lookup(Pong)
	if !ok || len(pong.Components) != 1 {
		t.Fatalf("EF_PONG spec = %+v ok=%t", pong, ok)
	}
	if component := pong.Components[0]; component.Kind != ComponentSTR || component.StrFile != "pong%d" || component.StrRandMin != 1 || component.StrRandMax != 3 || component.AttachedEntity {
		t.Fatalf("EF_PONG component = %+v", component)
	}

	level99, ok := Lookup(Level99)
	if !ok || level99.Duration != 5*time.Minute || len(level99.Components) != 1 {
		t.Fatalf("EF_LEVEL99 spec = %+v ok=%t", level99, ok)
	}
	if component := level99.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "Level99Aura" || component.FuncAdapter != EffectFuncLevel99Aura || component.TextureFile != "effect/ring_blue.tga" || !component.AttachedEntity {
		t.Fatalf("EF_LEVEL99 component = %+v", component)
	}
}

func TestRobrowserActiveEffectsTwoHundredToTwoFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		Level99:       "EF_LEVEL99",
		Level99Ground: "EF_LEVEL99_2",
		Level99Bubble: "EF_LEVEL99_3",
		Gumgang:       "EF_GUMGANG",
		PotionRed:     "EF_POTION1",
		PotionOrange:  "EF_POTION2",
		PotionYellow:  "EF_POTION3",
		PotionWhite:   "EF_POTION4",
		PotionBlue:    "EF_POTION5",
		PotionGreen:   "EF_POTION6",
		Food:          "EF_POTION7",
		FoodBlue:      "EF_POTION8",
		DarkBreath:    "EF_DARKBREATH",
		Defender:      "EF_DEFFENDER",
		Keeping:       "EF_KEEPING",
		SummonSlave:   "EF_SUMMONSLAVE",
		BloodDrain:    "EF_BLOODDRAIN",
		EnergyDrain:   "EF_ENERGYDRAIN",
		ItemFast:      "EF_POTION_CON",
		ItemFast2:     "EF_POTION_",
		ItemFast3:     "EF_POTION_BERSERK",
		CrusaderDef:   "EF_DEFENDER",
		GrandCross:    "EF_GRANDCROSS",
		Intimidate:    "EF_INTIMIDATE",
		Chookgi:       "EF_CHOOKGI",
		Cloud:         "EF_CLOUD",
		Cloud2:        "EF_CLOUD2",
		LineLink:      "EF_LINELINK",
		Cloud3:        "EF_CLOUD3",
		SpellBreaker:  "EF_SPELLBREAKER",
		Dispell:       "EF_DISPELL",
		BottomVolcano: "EF_BOTTOM_VO",
		BottomDeluge:  "EF_BOTTOM_DE",
		BottomViolent: "EF_BOTTOM_VI",
		BottomLand:    "EF_BOTTOM_LA",
		MagicRod:      "EF_MAGICROD",
		HolyCross:     "EF_HOLYCROSS",
		ShieldCharge:  "EF_SHIELDCHARGE",
		Providence:    "EF_PROVIDENCE",
		ShieldBoomer:  "EF_SHIELDBOOMERANG",
		SpearQuicken:  "EF_SPEARQUICKEN",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
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
		{"EF_DEFFENDER", Defender, "deffender", "", true, false},
		{"EF_KEEPING", Keeping, "keeping", "", true, false},
		{"EF_POTION_CON", ItemFast, "집중", "effect\\ac_concentration.wav", true, false},
		{"EF_POTION_", ItemFast2, "각성", "effect\\ac_concentration.wav", true, false},
		{"EF_SPELLBREAKER", SpellBreaker, "spell", "effect\\sage_spell breake.wav", true, false},
		{"EF_DISPELL", Dispell, "디스펠", "", true, false},
		{"EF_MAGICROD", MagicRod, "매직로드", "effect\\sage_magic rod.wav", true, false},
		{"EF_HOLYCROSS", HolyCross, "holy_cross", "effect\\cru_holy cross.wav", true, false},
		{"EF_SHIELDCHARGE", ShieldCharge, "shield_charge", "", true, false},
		{"EF_SPEARQUICKEN", SpearQuicken, "twohand", "effect\\knight_twohandquicken.wav", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
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
		{"EF_POTION1", PotionRed, "빨간포션", "_heal_effect.wav"},
		{"EF_POTION2", PotionOrange, "주홍포션", "_heal_effect.wav"},
		{"EF_POTION3", PotionYellow, "노란포션", "_heal_effect.wav"},
		{"EF_POTION4", PotionWhite, "하얀포션", "_heal_effect.wav"},
		{"EF_POTION5", PotionBlue, "파란포션", "effect\\흡기.wav"},
		{"EF_POTION6", PotionGreen, "초록포션", ""},
		{"EF_POTION7", Food, "fruit", "_heal_effect.wav"},
		{"EF_POTION8", FoodBlue, "fruit_", "effect\\흡기.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}
}

func TestRobrowserSpecialEffectsTwoHundredToTwoFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  EffectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_2", Level99Ground, "GroundAura", EffectFuncLevel99Ground, "effect/pikapika2.bmp"},
		{"EF_LEVEL99_3", Level99Bubble, "Level99Bubble", EffectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_CHOOKGI", Chookgi, "SpiritSphere", EffectFuncSpiritSphere, "effect/thunder_center.bmp"},
		{"EF_BOTTOM_LA", BottomLand, "LandProtectorGround", EffectFuncGroundSigil, "effect/aaa copy.bmp"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one FUNC component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentFUNC || component.FuncName != tc.funcName || component.FuncAdapter != tc.adapter || component.TextureFile != tc.texture {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	gumgang, ok := Lookup(Gumgang)
	if !ok || len(gumgang.Components) != 1 {
		t.Fatalf("EF_GUMGANG spec = %+v ok=%t", gumgang, ok)
	}
	if component := gumgang.Components[0]; component.FuncAdapter != EffectFuncGumgang || !component.AttachedEntity || len(component.TextureFiles) != 5 {
		t.Fatalf("EF_GUMGANG must use the original persistent lightning: %+v", component)
	}

	dark, ok := Lookup(DarkBreath)
	if !ok || len(dark.Components) != 1 {
		t.Fatalf("EF_DARKBREATH spec = %+v ok=%t", dark, ok)
	}
	if component := dark.Components[0]; component.Kind != ComponentSPR || component.SpriteFile != "darkbreath" || !component.SpriteHead || !component.AttachedEntity {
		t.Fatalf("EF_DARKBREATH component = %+v", component)
	}
	slave, ok := Lookup(SummonSlave)
	if !ok || len(slave.Components) != 1 {
		t.Fatalf("EF_SUMMONSLAVE spec = %+v ok=%t", slave, ok)
	}
	if component := slave.Components[0]; component.Kind != ComponentSPR || component.SpriteFile != "smoke" || !component.AttachedEntity {
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
		{"EF_BLOODDRAIN", BloodDrain, 255, 102, 102, 0},
		{"EF_ENERGYDRAIN", EnergyDrain, 102, 102, 255, 2},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1+tc.funcs {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != Component3D || component.SpriteFile != "data/sprite/이팩트/particle1" || component.Duration != 600*time.Millisecond || component.Duplicate != 5 || !component.ToSrc || !component.RotateToTarget || component.Arc != 3 || component.Retreat != 3 {
			t.Fatalf("%s particle = %+v", tc.name, component)
		}
		if component.Color.R != tc.r || component.Color.G != tc.g || component.Color.B != tc.b || component.SizeStart != testutil.EffectTableSize(150) || component.SizeEnd != testutil.EffectTableSize(180) || component.PosZ != 5 {
			t.Fatalf("%s particle scalar fields = %+v", tc.name, component)
		}
	}

}

func TestRobrowserActiveEffectsTwoFiftyToThreeHundredHaveSpecs(t *testing.T) {
	const (
		effectBottomDissonance = 277
		effectBottomUglyDance  = 290
	)
	active := map[int]string{
		SpearQuicken:   "EF_SPEARQUICKEN",
		Devotion:       "EF_DEVOTION",
		ReflectShield:  "EF_REFLECTSHIELD",
		AbsorbSpirits:  "EF_ABSORBSPIRITS",
		SteelBody:      "EF_STEELBODY",
		FlameLauncher:  "EF_FLAMELAUNCHER",
		FrostWeapon:    "EF_FROSTWEAPON",
		LightningLoad:  "EF_LIGHTNINGLOADER",
		SeismicWeapon:  "EF_SEISMICWEAPON",
		Gumgang2:       "EF_GUMGANG2",
		TeiHit1:        "EF_TEIHIT1",
		Gumgang3:       "EF_GUMGANG3",
		Tanji:          "EF_TANJI",
		TeiHit1X:       "EF_TEIHIT1X",
		Chimto:         "EF_CHIMTO",
		StealCoin:      "EF_STEALCOIN",
		StripWeapon:    "EF_STRIPWEAPON",
		StripShield:    "EF_STRIPSHIELD",
		StripArmor:     "EF_STRIPARMOR",
		StripHelm:      "EF_STRIPHELM",
		ChainCombo:     "EF_CHAINCOMBO",
		RogueCoin:      "EF_RG_COIN",
		BackStab:       "EF_BACKSTAP",
		TeiHit3:        "EF_TEIHIT3",
		BottomLullaby:  "EF_BOTTOM_LULLABY",
		BottomRichKim:  "EF_BOTTOM_RICHMANKIM",
		BottomChaos:    "EF_BOTTOM_ETERNALCHAOS",
		BottomDrum:     "EF_BOTTOM_DRUMBATTLEFIELD",
		BottomNibelung: "EF_BOTTOM_RINGNIBELUNGEN",
		BottomRoki:     "EF_BOTTOM_ROKISWEIL",
		BottomAbyss:    "EF_BOTTOM_INTOABYSS",
		BottomSieg:     "EF_BOTTOM_SIEGFRIED",
		BottomWhistle:  "EF_BOTTOM_WHISTLE",
		BottomSinX:     "EF_BOTTOM_ASSASSINCROSS",
		BottomBragi:    "EF_BOTTOM_POEMBRAGI",
		BottomApple:    "EF_BOTTOM_APPLEIDUN",
		BottomHumming:  "EF_BOTTOM_HUMMING",
		BottomForget:   "EF_BOTTOM_DONTFORGETME",
		BottomFortune:  "EF_BOTTOM_FORTUNEKISS",
		BottomService:  "EF_BOTTOM_SERVICEFORYOU",
		TalkFrostJoke:  "EF_TALK_FROSTJOKE",
		TalkScream:     "EF_TALK_SCREAM",
		PokJuk:         "EF_POKJUK",
		ThrowItem:      "EF_THROWITEM",
		ChemicalProt:   "EF_CHEMICALPROTECTION",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}

	for id, name := range map[int]string{
		effectBottomDissonance: "EF_BOTTOM_DISSONANCE",
		effectBottomUglyDance:  "EF_BOTTOM_UGLYDANCE",
	} {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeHundredToThreeFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		ChemicalProt:  "EF_CHEMICALPROTECTION",
		Demonstration: "EF_DEMONSTRATION",
		Chemical2:     "EF_CHEMICAL2",
		Teleportation: "EF_TELEPORTATION2",
		PharmacyOK:    "EF_PHARMACY_OK",
		PharmacyFail:  "EF_PHARMACY_FAIL",
		ThrowItem3:    "EF_THROWITEM3",
		FirstAid:      "EF_FIRSTAID",
		Loud:          "EF_LOUD",
		Heal:          "EF_HEAL",
		Heal2:         "EF_HEAL2",
		Exit2:         "EF_EXIT2",
		SafetyWall:    "EF_GLASSWALL2",
		ReadyPortal:   "EF_READYPORTAL2",
		Portal:        "EF_PORTAL2",
		BottomMagnus:  "EF_BOTTOM_MAG",
		BottomSanc:    "EF_BOTTOM_SANC",
		HealOffensive: "EF_HEAL3",
		WarpZone2:     "EF_WARPZONE2",
		Heal4:         "EF_HEAL4",
		BeginAsura:    "EF_BEGINASURA",
		TripleAttack:  "EF_TRIPLEATTACK",
		HPTime:        "EF_HPTIME",
		SPTime:        "EF_SPTIME",
		Maple:         "EF_MAPLE",
		Blind:         "EF_BLIND",
		PoisonStatus:  "EF_POISON",
		Guard:         "EF_GUARD",
		JobLvUp50:     "EF_JOBLVUP50",
		Magnum2:       "EF_MAGNUM2",
		Entry2:        "EF_ENTRY2",
		ColorPaper:    "EF_COLORPAPER",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsThreeFiftyToFourHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		SoulBreaker:       "EF_SOULBREAKER",
		Level99Aura1:      "EF_LEVEL99_4",
		FoodChocolate:     "EF_VALLENTINE",
		Pressure:          "EF_PRESSURE",
		Bash3D:            "EF_BASH3D",
		AuraBlade:         "EF_AURABLADE",
		RedBody:           "EF_REDBODY",
		LKConcentration:   "EF_LKCONCENTRATION",
		BottomGospel:      "EF_BOTTOM_GOSPEL",
		BaseLevelUp:       "EF_ANGEL",
		Death:             "EF_DEVIL",
		DragonSmoke:       "EF_DRAGONSMOKE",
		BottomBasilica:    "EF_BOTTOM_BASILICA",
		HitLine2:          "EF_HITLINE2",
		Bash3D2:           "EF_BASH3D2",
		EnergyDrain2:      "EF_ENERGYDRAIN2",
		TransBlueBody:     "EF_TRANSBLUEBODY",
		MagicCrasher:      "EF_MAGICCRASHER",
		LightBlade:        "EF_LIGHTBLADE",
		EnergyDrain3:      "EF_ENERGYDRAIN3",
		LineLink2:         "EF_LINELINK2",
		TrueSight:         "EF_TRUESIGHT",
		FalconAssault:     "EF_FALCONASSAULT",
		TripleAttack2:     "EF_TRIPLEATTACK2",
		Portal4:           "EF_PORTAL4",
		Meltdown:          "EF_MELTDOWN",
		CartBoost:         "EF_CARTBOOST",
		RejectSword:       "EF_REJECTSWORD",
		TripleAttack3:     "EF_TRIPLEATTACK3",
		Moonlit:           "EF_SPHEREWIND2",
		Level99AuraMid:    "EF_LEVEL99_5",
		Level99AuraBottom: "EF_LEVEL99_6",
		Bash3D3:           "EF_BASH3D3",
		Bash3D4:           "EF_BASH3D4",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourHundredToFourFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		Portal5:       "EF_PORTAL5",
		MagicCrasher2: "EF_MAGICCRASHER2",
		BottomSpider:  "EF_BOTTOM_SPIDER",
		SoulBurn:      "EF_SOULBURN",
		SoulChange:    "EF_SOULCHANGE",
		SoulBreaker2:  "EF_SOULBREAKER2",
		BabyBody:      "EF_BABYBODY",
		BabyBody2:     "EF_BABYBODY2",
		GiantBody:     "EF_GIANTBODY",
		GiantBody2:    "EF_GIANTBODY2",
		QuakeBody:     "EF_QUAKEBODY",
		Assumptio2:    "EF_ASSUMPTIO2",
		StopEffect:    "EF_STOPEFFECT",
		JumpBody:      "EF_JUMPBODY",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFourFiftyToFiveHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		DarkGrandCross: "EF_GRANDCROSS2",
		DarkSoulStrike: "EF_SOULSTRIKE2",
		DarkJupitelHit: "EF_YUFITEL2",
		NPCStop:        "EF_NPC_STOP",
		DarkCasting:    "EF_DARKCASTING",
		AgiUp:          "EF_AGIUP",
		JumpKick:       "EF_JUMPKICK",
		BeginAsura1:    "EF_BEGINASURA1",
		BeginAsura2:    "EF_BEGINASURA2",
		BeginAsura3:    "EF_BEGINASURA3",
		BeginAsura4:    "EF_BEGINASURA4",
		BeginAsura5:    "EF_BEGINASURA5",
		BeginAsura6:    "EF_BEGINASURA6",
		BeginAsura7:    "EF_BEGINASURA7",
		Mochi:          "EF_MOCHI",
		Ramadan:        "EF_LAMADAN",
		EDP:            "EF_EDP",
		Preserve:       "EF_GUARD2",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveHundredToFiveFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		CastSpin:      "EF_CASTSPIN",
		Chookgi2:      "EF_CHOOKGI2",
		Mapae:         "EF_MAPAE",
		ItemPokJuk:    "EF_ITEMPOKJUK",
		Valentine05:   "EF_05VAL",
		BeginAsura11:  "EF_BEGINASURA11",
		Chemical2Dash: "EF_CHEMICAL2DASH",
		GroundSample:  "EF_GROUNDSAMPLE",
		Cloud4:        "EF_CLOUD4",
		Cloud5:        "EF_CLOUD5",
		BottomHermode: "EF_BOTTOM_HERMODE",
		ItemFastDown:  "EF_ITEMFAST",
		TarotCard1:    "EF_TAROTCARD1",
		TarotCard2:    "EF_TAROTCARD2",
		TarotCard3:    "EF_TAROTCARD3",
		TarotCard4:    "EF_TAROTCARD4",
		TarotCard5:    "EF_TAROTCARD5",
		TarotCard6:    "EF_TAROTCARD6",
		TarotCard7:    "EF_TAROTCARD7",
		TarotCard8:    "EF_TAROTCARD8",
		TarotCard9:    "EF_TAROTCARD9",
		TarotCard10:   "EF_TAROTCARD10",
		TarotCard11:   "EF_TAROTCARD11",
		TarotCard12:   "EF_TAROTCARD12",
		TarotCard13:   "EF_TAROTCARD13",
		TarotCard14:   "EF_TAROTCARD14",
		AcidDemon:     "EF_ACIDDEMON",
		Hated:         "EF_HATED",
		Stin:          "EF_STIN",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsFiveFiftyToSixHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		Stin2:       "EF_STIN2",
		Stin3:       "EF_STIN3",
		ScreenQuake: "EF_SCREEN_QUAKE",
		HfliMoon1:   "EF_HFLIMOON1",
		HfliMoon2:   "EF_HFLIMOON2",
		HfliMoon3:   "EF_HFLIMOON3",
		HoUp:        "EF_HO_UP",
		HamiDefence: "EF_HAMIDEFENCE",
		HamiCastle:  "EF_HAMICASTLE",
		HamiBlood:   "EF_HAMIBLOOD",
		ItemThunder: "EF_ITEM_THUNDER",
		ItemCloud:   "EF_ITEM_CLOUD",
		ItemCurse:   "EF_ITEM_CURSE",
		ItemZZZ:     "EF_ITEM_ZZZ",
		ItemRain:    "EF_ITEM_RAIN",
		M01:         "EF_M01",
		M02:         "EF_M02",
		M03:         "EF_M03",
		M04:         "EF_M04",
		M05:         "EF_M05",
		M06:         "EF_M06",
		M07:         "EF_M07",
		Kaizel:      "EF_KAIZEL",
		Cloud6:      "EF_CLOUD6",
		StatFoodSTR: "EF_FOOD01",
		StatFoodINT: "EF_FOOD02",
		StatFoodVIT: "EF_FOOD03",
		StatFoodAGI: "EF_FOOD04",
		StatFoodDEX: "EF_FOOD05",
		StatFoodLUK: "EF_FOOD06",
		ThrowItem6:  "EF_THROWITEM6",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixHundredToSixFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		ThrowItem6:    "EF_THROWITEM6",
		FireHit2:      "EF_FIREHIT2",
		NPCStop2:      "EF_NPC_STOP2",
		FVoice:        "EF_FVOICE",
		Wink:          "EF_WINK",
		CookingOK:     "EF_COOKING_OK",
		CookingFail:   "EF_COOKING_FAIL",
		Hapgyeok:      "EF_HAPGYEOK",
		ThrowItem7:    "EF_THROWITEM7",
		ThrowItem8:    "EF_THROWITEM8",
		ThrowItem9:    "EF_THROWITEM9",
		ThrowItem10:   "EF_THROWITEM10",
		Kouenka:       "EF_KOUENKA",
		Hyousensou:    "EF_HYOUSENSOU",
		Stin4:         "EF_STIN4",
		ThunderStorm2: "EF_THUNDERSTORM2",
		RGCoin3:       "EF_RG_COIN3",
		Bash3D5:       "EF_BASH3D5",
		Chookgi3:      "EF_CHOOKGI3",
		Kirikage:      "EF_KIRIKAGE",
		Tatami:        "EF_TATAMI",
		Kasumikiri:    "EF_KASUMIKIRI",
		Issen:         "EF_ISSEN",
		Kaen:          "EF_KAEN",
		Baku:          "EF_BAKU",
		Hyousyouraku:  "EF_HYOUSYOURAKU",
		Desperado:     "EF_DESPERADO",
		LightningS:    "EF_LIGHTNING_S",
		BlindS:        "EF_BLIND_S",
		PoisonS:       "EF_POISON_S",
		FreezingS:     "EF_FREEZING_S",
		FlareS:        "EF_FLARE_S",
		RapidShower:   "EF_RAPIDSHOWER",
		MagicalBullet: "EF_MAGICALBULLET",
		SpreadAttack:  "EF_SPREADATTACK",
		TrackCasting:  "EF_TRACKCASTING",
		Tracking:      "EF_TRACKING",
		TripleAction:  "EF_TRIPLEACTION",
		Bullseye:      "EF_BULLSEYE",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSixFiftyToSevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		NPCEarthquake:  "EF_NPC_EARTHQUAKE",
		DragonFear:     "EF_DRAGONFEAR",
		WideBleeding:   "EF_BLEEDING",
		WideConfuse:    "EF_WIDECONFUSE",
		BottomRunner:   "EF_BOTTOM_RUNNER",
		BottomTransfer: "EF_BOTTOM_TRANSFER",
		BottomEvilLand: "EF_BOTTOM_EVILLAND",
		Guard3:         "EF_GUARD3",
		CriticalWound:  "EF_CRITICALWOUND",
		Firecracker2:   "EF_POK_LOVE",
		Firecracker3:   "EF_POK_WHITE",
		Firecracker4:   "EF_POK_VALEN",
		Firecracker5:   "EF_POK_BIRTH",
		Firecracker6:   "EF_POK_CHRISTMAS",
		Cloud7:         "EF_CLOUD7",
		Cloud8:         "EF_CLOUD8",
		FlowerLeaf:     "EF_FLOWERLEAF",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenHundredToSevenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		Item315:          "EF_ITEM315",
		Item316:          "EF_ITEM316",
		Item317:          "EF_ITEM317",
		StormMin:         "EF_STORM_MIN",
		Firecracker7:     "EF_POK_JAP",
		BottomBlue:       "EF_BOTTOM_BLUE",
		BottomBlue2:      "EF_BOTTOM_BLUE2",
		ChristmasCarol:   "EF_WEWISH",
		FirePillarOn2:    "EF_FIREPILLARON2",
		ForestLight5:     "EF_FORESTLIGHT5",
		Adoramus:         "EF_ADO_STR",
		IgnitionBreak:    "EF_IGN_STR",
		FrostMisty:       "EF_FROSTMYSTY",
		CrimsonRock:      "EF_CRIMSON_STR",
		HellInferno:      "EF_HELL_STR",
		MarshOfAbyss:     "EF_SPR_MASH",
		DragonHowling:    "EF_DHOWL_STR",
		EarthWall:        "EF_EARTHWALL",
		ChainLightning:   "EF_CHAINL_STR",
		AimedBolt:        "EF_AIMED_STR",
		ArrowStorm:       "EF_ARROWSTORM_STR",
		Laulamus:         "EF_LAULAMUS_STR",
		Lauagnus:         "EF_LAUAGNUS_STR",
		MillenniumShield: "EF_MILSHIELD_STR",
		Concentration2:   "EF_CONCENTRATION2",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsSevenFiftyToEightHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		GlassWall3:     "EF_GLASSWALL3",
		BerserkPotion2: "EF_POTION_BERSERK2",
		Rolling1:       "EF_ROLLING1",
		Rolling2:       "EF_ROLLING2",
		Rolling3:       "EF_ROLLING3",
		Rolling4:       "EF_ROLLING4",
		Rolling5:       "EF_ROLLING5",
		Rolling6:       "EF_ROLLING6",
		Rolling7:       "EF_ROLLING7",
		Rolling8:       "EF_ROLLING8",
		Rolling9:       "EF_ROLLING9",
		Rolling10:      "EF_ROLLING10",
		CastSpin2:      "EF_CASTSPIN2",
		CrashAxe:       "EF_CRASHAXE",
		Stasis:         "EF_STASIS",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightHundredToEightFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		BottomBasilica2:  "EF_BOTTOM_BASILICA2",
		Recognized:       "EF_RECOGNIZED",
		Tetra:            "EF_TETRA",
		TetraCasting:     "EF_TETRACASTING",
		Stretch:          "EF_STRETCH",
		Enervation:       "EF_ENERVATION",
		Enervation2:      "EF_ENERVATION2",
		Enervation3:      "EF_ENERVATION3",
		Enervation4:      "EF_ENERVATION4",
		Enervation5:      "EF_ENERVATION5",
		Enervation6:      "EF_ENERVATION6",
		BottomManhole:    "EF_BOTTOM_MANHOLE",
		Manhole:          "EF_MANHOLE",
		ForestLight6:     "EF_FORESTLIGHT6",
		BottomAni:        "EF_BOTTOM_ANI",
		BottomMaelstrom:  "EF_BOTTOM_MAELSTROM",
		BottomBloodyLust: "EF_BOTTOM_BLOODYLUST",
		HealN:            "EF_HEAL_N",
		ChookgiN:         "EF_CHOOKGI_N",
		Dance1:           "EF_DANCE1",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsEightFiftyToNineHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		BotReverb:    "EF_BOT_REVERB",
		RainParticle: "EF_RAIN_PARTICLE",
		ChemicalV2:   "EF_CHEMICAL_V2",
		BotReverb2:   "EF_BOT_REVERB2",
		CirclePower2: "EF_CIRCLEPOWER2",
		Secra2:       "EF_SECRA2",
		SprPlant2:    "EF_SPR_PLANT2",
		SprPlant3:    "EF_SPR_PLANT3",
		SprPlant4:    "EF_SPR_PLANT4",
		SprPlant5:    "EF_SPR_PLANT5",
		SprPlant6:    "EF_SPR_PLANT6",
		SprPlant7:    "EF_SPR_PLANT7",
		SprPlant8:    "EF_SPR_PLANT8",
		HeartAsura:   "EF_HEARTASURA",
		GlassWall4:   "EF_GLASSWALL4",
		Bash3D6:      "EF_BASH3D6",
		Electric4:    "EF_ELECTRIC4",
		TeiHit1T:     "EF_TEIHIT1T",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineHundredToNineFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		Pressure2:    "EF_PRESSURE2",
		PrimeCharge2: "EF_PRIMECHARGE2",
		PrimeCharge3: "EF_PRIMECHARGE3",
		PrimeCharge4: "EF_PRIMECHARGE4",
		FireWall2:    "EF_FIREWALL2",
		SprPlant10:   "EF_SPR_PLANT10",
		Shockwave2:   "EF_SHOCKWAVE2",
		ColdThrow2:   "EF_COLDTHROW2",
		DemonicFire4: "EF_DEMONICFIRE4",
		Pressure3:    "EF_PRESSURE3",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsNineFiftyToOneThousandHaveSpecs(t *testing.T) {
	active := map[int]string{
		PoisonMist:     "EF_POISON_MIST",
		EraserCutter:   "EF_ERASER_CUTTER",
		LavaSlide:      "EF_LAVA_SLIDE",
		SonicClaw:      "EF_SONIC_CLAW",
		TinderBreaker:  "EF_TINDER_BREAKER",
		MidnightFrenzy: "EF_MIDNIGHT_FRENZY",
		VolcanicAsh:    "EF_VOLCANIC_ASH",
		RWC2011:        "EF_2011RWC",
		RWC2011Two:     "EF_2011RWC2",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsOneThousandToTenFiftyHaveSpecs(t *testing.T) {
	active := map[int]string{
		RunMakeOK:        "EF_RUN_MAKE_OK",
		RunMakeFailure:   "EF_RUN_MAKE_FAILURE",
		MIResultMakeOK:   "EF_MIRESULT_MAKE_OK",
		MIResultMakeFail: "EF_MIRESULT_MAKE_FAIL",
		AllRayProtect:    "EF_ALL_RAY_OF_PROTECTION",
		VenomFog:         "EF_VENOMFOG",
		DustStorm:        "EF_DUSTSTORM",
		DanceBladeAtk:    "EF_DANCE_BLADE_ATK",
		InvincibleOff2:   "EF_INVINCIBLEOFF2",
		DeathSummon:      "EF_DEATHSUMMON",
		GCDarkCrow:       "EF_GC_DARKCROW",
		AllFullThrottle:  "EF_ALL_FULL_THROTTLE",
		SRFlashCombo:     "EF_SR_FLASHCOMBO",
		RKLuxAnima:       "EF_RK_LUXANIMA",
		SOElemShield:     "EF_SO_ELEMENTAL_SHIELD",
		ABOffertorium:    "EF_AB_OFFERTORIUM",
		WLTelekinesis:    "EF_WL_TELEKINESIS_INTENSE",
		GNIllusionDoping: "EF_GN_ILLUSIONDOPING",
		NCMagmaEruption:  "EF_NC_MAGMA_ERUPTION",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsTenFiftyToElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		NPCChill:        "EF_NPC_CHILL",
		OffertoriumRing: "EF_AB_OFFERTORIUM_RING",
		HammerOfGod:     "EF_HAMMER_OF_GOD",
		AchComplete:     "EF_ACH_COMPLETE",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
			t.Fatalf("%s (%d) spec missing", name, id)
		}
	}
}

func TestRobrowserActiveEffectsPostElevenHundredHaveSpecs(t *testing.T) {
	active := map[int]string{
		BodyColor:        "EffectBodyColor",
		BakuretsuHadou:   "EF_BAKURETSU_HADOU",
		DropPink:         "DROPEFFECT_PINK",
		DropYellow:       "DROPEFFECT_YELLOW",
		DropPurple:       "DROPEFFECT_PURPLE",
		DigitalSpace:     "EF_DIGITAL_SPACE",
		DropBlue:         "DROPEFFECT_BLUE",
		DropGreen:        "DROPEFFECT_GREEN",
		DropRed:          "DROPEFFECT_RED",
		NewSuccess:       "EF_NEW_SUCCESS",
		NewFailure:       "EF_NEW_FAILURE",
		NewIntro:         "EF_NEW_INTRO",
		EnchantYellow:    "EF_UI_ENCHANT_INTRO_YELLOW",
		EnchantSuccess:   "EF_UI_ENCHANT_SUCCESS",
		EnchantFail:      "EF_UI_ENCHANT_FAIL",
		EnchantBlue:      "EF_UI_ENCHANT_INTRO_BLUE",
		EnchantUpSuccess: "EF_UI_ENCHANT_UP_SUCCESS",
		EnchantUpFail:    "EF_UI_ENCHANT_UP_FAIL",
		EnchantGreen:     "EF_UI_ENCHANT_INTRO_GREEN",
		EnchantResetOK:   "EF_UI_ENCHANT_RESET_SUCCESS",
		EnchantResetFail: "EF_UI_ENCHANT_RESET_FAIL",
	}
	for id, name := range active {
		if _, ok := Lookup(id); !ok {
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
		{"EF_DEVOTION", Devotion, "devotion", "", true, false},
		{"EF_FLAMELAUNCHER", FlameLauncher, "enc_fire", "_enemy_hit_wind1.wav", true, false},
		{"EF_FROSTWEAPON", FrostWeapon, "enc_ice", "_enemy_hit_wind1.wav", true, false},
		{"EF_LIGHTNINGLOADER", LightningLoad, "enc_wind", "_enemy_hit_wind1.wav", true, false},
		{"EF_SEISMICWEAPON", SeismicWeapon, "enc_earth", "_enemy_hit_wind1.wav", true, false},
		{"EF_STEALCOIN", StealCoin, "steal_coin", "", true, false},
		{"EF_STRIPWEAPON", StripWeapon, "strip_weapon", "effect\\복호격.wav", true, false},
		{"EF_STRIPSHIELD", StripShield, "strip_shield", "effect\\복호격.wav", true, false},
		{"EF_STRIPARMOR", StripArmor, "strip_armor", "effect\\복호격.wav", true, false},
		{"EF_STRIPHELM", StripHelm, "strip_helm", "effect\\복호격.wav", true, false},
		{"EF_CHAINCOMBO", ChainCombo, "연환", "effect\\mon_연환.wav", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v, want STR %q attached=%t head=%t", tc.name, component, tc.file, tc.attached, tc.head)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

}

func TestGroundTextureEffectHoverMovesBetweenZRange(t *testing.T) {
	start := time.Unix(1, 0)
	effect := Instance{EffectID: BottomHummingGround, ActorID: 172, Starts: start}
	component := Component{PosZ: 0.2, PosZEnd: 0.6}

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

	for _, tc := range []struct {
		name     string
		id       int
		funcName string
	}{
		{"EF_TALK_FROSTJOKE", TalkFrostJoke, "FrostJokeTalk"},
		{"EF_TALK_SCREAM", TalkScream, "ScreamTalk"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || spec.Duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentFUNC || component.FuncName != tc.funcName || component.FuncAdapter != EffectFuncUnknown || !component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

}

func TestRobrowserSimpleEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	demo, ok := Lookup(Demonstration)
	if !ok || len(demo.Components) != 1 {
		t.Fatalf("EF_DEMONSTRATION spec = %+v ok=%t", demo, ok)
	}
	if component := demo.Components[0]; component.Kind != Component3D || component.SpriteFile != "데몬스트레이션" || component.AttachedEntity {
		t.Fatalf("EF_DEMONSTRATION component = %+v", component)
	}

	job, ok := Lookup(JobLvUp50)
	if !ok || len(job.Components) != 1 {
		t.Fatalf("EF_JOBLVUP50 spec = %+v ok=%t", job, ok)
	}
	if component := job.Components[0]; component.Kind != ComponentSTR || component.StrFile != "joblvup" || !component.AttachedEntity {
		t.Fatalf("EF_JOBLVUP50 component = %+v", component)
	}

	colorPaper, ok := Lookup(ColorPaper)
	if !ok || len(colorPaper.Components) != 0 || colorPaper.Duration != 500*time.Millisecond {
		t.Fatalf("EF_COLORPAPER spec = %+v ok=%t", colorPaper, ok)
	}
	if len(colorPaper.Sfx) != 1 || colorPaper.Sfx[0] != "effect\\wedding.wav" {
		t.Fatalf("EF_COLORPAPER sfx = %#v", colorPaper.Sfx)
	}

	chemical, ok := Lookup(Chemical2)
	if !ok || len(chemical.Components) != 1 || chemical.Duration != 500*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 spec = %+v ok=%t", chemical, ok)
	}
	if chemical.CameraShake != 200*time.Millisecond || chemical.CameraShakeDelay != 132*time.Millisecond {
		t.Fatalf("EF_CHEMICAL2 camera shake = delay %s duration %s", chemical.CameraShakeDelay, chemical.CameraShake)
	}
	if component := chemical.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "CameraQuake" || component.FuncAdapter != EffectFuncUnknown || !component.AttachedEntity {
		t.Fatalf("EF_CHEMICAL2 component = %+v", component)
	}

	blind, ok := Lookup(Blind)
	if !ok || len(blind.Components) != 1 || blind.Duration != 500*time.Millisecond {
		t.Fatalf("EF_BLIND spec = %+v ok=%t", blind, ok)
	}
	if len(blind.Sfx) != 1 || blind.Sfx[0] != "_blind.wav" {
		t.Fatalf("EF_BLIND sfx = %#v", blind.Sfx)
	}
	if component := blind.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "Blind" || component.FuncAdapter != EffectFuncUnknown || component.AttachedEntity {
		t.Fatalf("EF_BLIND component = %+v", component)
	}

	poison, ok := Lookup(PoisonStatus)
	if !ok || len(poison.Components) != 1 || poison.Duration != 500*time.Millisecond {
		t.Fatalf("EF_POISON spec = %+v ok=%t", poison, ok)
	}
	if component := poison.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "Poison" || component.FuncAdapter != EffectFuncUnknown || component.AttachedEntity {
		t.Fatalf("EF_POISON component = %+v", component)
	}
}

func TestRobrowserPortalAndGroundEffectsThreeHundredToThreeFiftyMatchTableRows(t *testing.T) {
	exit, ok := Lookup(Exit2)
	if !ok || len(exit.Components) != 3 || exit.Duration != 1500*time.Millisecond {
		t.Fatalf("EF_EXIT2 spec = %+v ok=%t", exit, ok)
	}
	if len(exit.Sfx) != 1 || exit.Sfx[0] != "effect\\ef_teleportation.wav" {
		t.Fatalf("EF_EXIT2 sfx = %#v", exit.Sfx)
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
		component := exit.Components[i]
		if component.Kind != ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 1500*time.Millisecond || component.AlphaMax != 0.3 || component.Animation != 1 || component.BlendMode != 2 || !component.BlendAdditive || !component.Fade || !component.Rotate || !component.AttachedEntity {
			t.Fatalf("EF_EXIT2 component %d = %+v", i, component)
		}
		if component.BottomSize != want.bottom || component.TopSize != want.top || component.Height != want.height {
			t.Fatalf("EF_EXIT2 component %d size = %+v", i, component)
		}
		if component.Color != (color.RGBA{R: 128, G: 128, B: 255, A: 255}) {
			t.Fatalf("EF_EXIT2 component %d color = %+v", i, component.Color)
		}
	}

	warp, ok := Lookup(WarpZone2)
	if !ok || len(warp.Components) != 3 || warp.Duration != 7*time.Second {
		t.Fatalf("EF_WARPZONE2 spec = %+v ok=%t", warp, ok)
	}
	for i, want := range []struct {
		bottom float64
		top    float64
	}{{1.15, 1.9}, {1.05, 1.8}} {
		component := warp.Components[i]
		if component.Kind != ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 4*time.Second || component.Duplicate != 4 || component.DuplicateDelay != time.Second || component.AlphaMax != 0.48 || component.Animation != 3 || component.BottomSize != want.bottom || component.TopSize != want.top || component.Height != 1.35 || !component.Repeat || component.Rotate || !component.Fade || !component.AttachedEntity {
			t.Fatalf("EF_WARPZONE2 cylinder %d = %+v", i, component)
		}
	}
	particle := warp.Components[2]
	if particle.Kind != Component3D || particle.TextureFile != "effect/pok1.tga" || particle.Duration != time.Second || particle.Duplicate != 5 || particle.DuplicateDelay != 300*time.Millisecond || particle.SizeStart != testutil.EffectTableSize(32) || particle.PosXStartRand != 1.45 || particle.PosYStartRand != 1.45 || particle.PosZEndRand != 1.5 || particle.PosZEndMiddle != 1.4 || !particle.Repeat || !particle.BlendAdditive || !particle.AttachedEntity {
		t.Fatalf("EF_WARPZONE2 particle = %+v", particle)
	}

	entry, ok := Lookup(Entry2)
	if !ok || len(entry.Components) != 4 || entry.Duration != 1500*time.Millisecond {
		t.Fatalf("EF_ENTRY2 spec = %+v ok=%t", entry, ok)
	}
	if len(entry.Sfx) != 1 || entry.Sfx[0] != "effect\\ef_portal.wav" {
		t.Fatalf("EF_ENTRY2 sfx = %#v", entry.Sfx)
	}
	if entry.Components[0].BottomSize != 0.3 || entry.Components[3].TopSize != 1.3 {
		t.Fatalf("EF_ENTRY2 cylinder sizes = %+v", entry.Components)
	}
	for i, component := range entry.Components {
		if component.Kind != ComponentCylinder || component.TextureName != "ring_blue" || component.Duration != 1500*time.Millisecond || component.AlphaMax != 0.5 || component.Animation != 5 || component.BlendMode != 2 || !component.BlendAdditive || !component.Fade || !component.Rotate || !component.AttachedEntity {
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
		{"EF_HEAL2", Heal2, 1890 * time.Millisecond, 15, 13, 2, 4, 20, 10, 9, 2, 9, 2},
		{"EF_HEAL4", Heal4, 2 * time.Second, 18, 15, 3, 7, 25, 15, 10, 5, 11, 3},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 6 || spec.Duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != "_heal_effect.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.Sfx)
		}
		heights := []float64{tc.firstHeight, tc.secondHeight, tc.thirdHeight}
		for i, height := range heights {
			component := spec.Components[i]
			if component.Kind != ComponentCylinder || component.TextureName != "ring_white" || component.Duration != 1500*time.Millisecond || component.AlphaMax != 0.3 || component.Animation != 1 || component.BlendMode != 2 || !component.BlendAdditive || !component.Fade || !component.Rotate || !component.AttachedEntity || component.Height != height {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
			if component.Color != (color.RGBA{R: 178, G: 255, B: 178, A: 255}) {
				t.Fatalf("%s cylinder %d color = %+v", tc.name, i, component.Color)
			}
		}
		if spec.Components[0].BottomSize != 1.1 || spec.Components[0].TopSize != 1.1 || spec.Components[1].BottomSize != 1 || spec.Components[2].TopSize != 3 {
			t.Fatalf("%s cylinder sizes = %+v", tc.name, spec.Components[:3])
		}
		first, second, third := spec.Components[3], spec.Components[4], spec.Components[5]
		if first.Kind != Component3D || first.TextureFile != "effect/pok3.tga" || first.Duration != 1500*time.Millisecond || first.Duplicate != tc.firstDuplicate || first.DuplicateDelay != 10*time.Millisecond || first.PosXRand != 1.2 || first.PosYRand != 1.2 || first.PosZEndRand != 1 || first.PosZEndMiddle != 8 || first.SizeStart != testutil.EffectTableSize(9) || !first.Sparkling || first.SparkNumber != tc.sparkNumber {
			t.Fatalf("%s first particle = %+v", tc.name, first)
		}
		if second.Duration != 1300*time.Millisecond || second.Delay != 400*time.Millisecond || second.Duplicate != tc.secondDuplicate || second.PosXRand != 1.5 || second.PosYRand != 1.5 || second.PosZEndRand != 3 || second.PosZEndMiddle != 6 || second.SizeStart != testutil.EffectTableSize(tc.secondSize) || second.SizeRand != testutil.EffectTableSize(tc.secondSizeRand) || !second.Sparkling || second.SparkNumber != tc.sparkNumber {
			t.Fatalf("%s second particle = %+v", tc.name, second)
		}
		if third.Duration != 1100*time.Millisecond || third.Delay != 200*time.Millisecond || third.Duplicate != tc.thirdDuplicate || third.DuplicateDelay != 50*time.Millisecond || third.PosXRand != 1 || third.PosYRand != 1 || third.PosZEnd != 6 || third.PosZStartRand != 1 || third.SizeStart != testutil.EffectTableSize(tc.thirdSize) || third.Sparkling {
			t.Fatalf("%s third particle = %+v", tc.name, third)
		}
	}

	for _, tc := range []struct {
		name  string
		id    int
		color color.RGBA
		wav   string
	}{
		{"EF_HPTIME", HPTime, color.RGBA{R: 230, G: 255, B: 230, A: 255}, "_heal_effect.wav"},
		{"EF_SPTIME", SPTime, color.RGBA{R: 230, G: 230, B: 255, A: 255}, "effect\\흡기.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || spec.Duration != 1110*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.Sfx)
		}
		component := spec.Components[0]
		if component.Kind != Component3D || component.TextureFile != "effect/pok1.tga" || component.Color != tc.color || component.Duration != 500*time.Millisecond || component.Delay != 500*time.Millisecond || component.Duplicate != 12 || component.DuplicateDelay != 10*time.Millisecond || component.AlphaMax != 0.8 || component.SizeStart != testutil.EffectTableSize(30) || component.SizeRand != testutil.EffectTableSize(20) || component.PosXRand != 0.6 || component.PosYRand != 0.6 || component.PosZStartRand != 1.5 || component.PosZStartMiddle != 2 || component.PosZEndRand != 1 || component.PosZEndMiddle != 5 || !component.Sparkling || component.SparkNumber != 3 || !component.BlendAdditive || !component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
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
		{"EF_LKCONCENTRATION", LKConcentration, "twohand", "", true, true},
		{"EF_DEVIL", Death, "devil", "", true, false},
		{"EF_CARTBOOST", CartBoost, "cart", "effect\\ef_incagility.wav", true, false},
		{"EF_REJECTSWORD", RejectSword, "sword", "effect\\kyrie_guard.wav", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
		head bool
	}{
		{"EF_VALLENTINE", FoodChocolate, "vallentine", "effect\\vallentine.wav", false},
		{"EF_DRAGONSMOKE", DragonSmoke, "poisonhit", "", false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSPR || component.SpriteFile != tc.file || !component.AttachedEntity || component.SpriteHead != tc.head {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_BOTTOM_GOSPEL", BottomGospel, "effect\\가스펠.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 0 || spec.Duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	base, ok := Lookup(BaseLevelUp)
	if !ok || len(base.Components) != 1 || base.Components[0].Kind != ComponentSTR || base.Components[0].StrFile != "angel" || !base.Components[0].AttachedEntity {
		t.Fatalf("EF_ANGEL spec = %+v ok=%t", base, ok)
	}
}

func TestRobrowserLevel99AliasesThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name     string
		id       int
		funcName string
		adapter  EffectFuncAdapter
		texture  string
	}{
		{"EF_LEVEL99_4", Level99Aura1, "Level99Bubble", EffectFuncLevel99Bubble, "effect/whitelight.tga"},
		{"EF_LEVEL99_5", Level99AuraMid, "Level99Aura", EffectFuncLevel99Aura, "effect/ring_blue.tga"},
		{"EF_LEVEL99_6", Level99AuraBottom, "GroundAura", EffectFuncLevel99Ground, "effect/pikapika2.bmp"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || spec.Duration != 5*time.Minute {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentFUNC || component.FuncName != tc.funcName || component.FuncAdapter != tc.adapter || component.TextureFile != tc.texture || !component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserCombatEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name      string
		id        int
		funcName  string
		wav       string
		duration  time.Duration
		delay     time.Duration
		duplicate int
	}{} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 3 || spec.Duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v", tc.name, spec.Sfx)
		}
		if component := spec.Components[0]; component.Kind != ComponentFUNC || component.FuncName != tc.funcName || component.FuncAdapter != EffectFuncUnknown || !component.AttachedEntity {
			t.Fatalf("%s body func = %+v", tc.name, component)
		}
		for i, component := range spec.Components[1:] {
			wantTop := 4.5
			if i == 1 {
				wantTop = 7.2
			}
			if component.Kind != ComponentCylinder || component.TextureName != "alpha_center" || component.Duration != 175*time.Millisecond || component.Delay != tc.delay || component.Duplicate != tc.duplicate || component.AlphaMax != 0.6 || !component.Fade || component.AngleX != -90 || component.AngleZRandom != 360 || !component.FixedPerspective || component.PosZ != 1.5 || component.Height != 0 || component.BottomSize != 0.01 || component.TopSize != wantTop || component.Animation != 2 || component.TotalCircleSides != 30 || component.CircleSides != 1 || !component.AttachedEntity {
				t.Fatalf("%s cylinder %d = %+v", tc.name, i, component)
			}
		}
	}
}

func TestRobrowserDrainAndMagicEffectsThreeFiftyToFourHundredMatchTableRows(t *testing.T) {
	trans, ok := Lookup(TransBlueBody)
	if !ok || len(trans.Components) != 1 || trans.Duration != 200*db.EffectFrameDuration {
		t.Fatalf("EF_TRANSBLUEBODY spec = %+v ok=%t", trans, ok)
	}
	if component := trans.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "TransBlueBody" || component.FuncAdapter != EffectFuncBodyColor || !component.AttachedEntity {
		t.Fatalf("EF_TRANSBLUEBODY component = %+v", component)
	}

	magic, ok := Lookup(MagicCrasher)
	if !ok || len(magic.Components) != 1 || magic.Duration != 100*db.EffectFrameDuration {
		t.Fatalf("EF_MAGICCRASHER spec = %+v ok=%t", magic, ok)
	}
	if len(magic.Sfx) != 1 || magic.Sfx[0] != "effect\\매직 크래쉬.wav" || magic.CameraShakeDelay != 30*db.EffectFrameDuration || magic.CameraShake != 650*time.Millisecond {
		t.Fatalf("EF_MAGICCRASHER timing/sfx = sfx %#v delay %s shake %s", magic.Sfx, magic.CameraShakeDelay, magic.CameraShake)
	}
	if body := magic.Components[0]; body.Kind != ComponentFUNC || body.FuncName != "MagicCrasherBodyColor" || !body.AttachedEntity {
		t.Fatalf("EF_MAGICCRASHER components = %+v", magic.Components)
	}

	falcon, ok := Lookup(FalconAssault)
	if !ok || len(falcon.Components) != 1 || falcon.Duration != 200*db.EffectFrameDuration {
		t.Fatalf("EF_FALCONASSAULT spec = %+v ok=%t", falcon, ok)
	}
	if len(falcon.Sfx) != 0 || falcon.CameraShakeDelay != 30*db.EffectFrameDuration || falcon.CameraShake != 200*time.Millisecond {
		t.Fatalf("EF_FALCONASSAULT timing/sfx = sfx %#v delay %s shake %s", falcon.Sfx, falcon.CameraShakeDelay, falcon.CameraShake)
	}
	if component := falcon.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "CameraQuake" || component.Delay != 30*db.EffectFrameDuration || !component.AttachedEntity {
		t.Fatalf("EF_FALCONASSAULT component = %+v", component)
	}

}

func TestRobrowserEffectsFourHundredToFourFiftyMatchTableRows(t *testing.T) {

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_SOULBURN", SoulBurn, "소울번"},
		{"EF_SOULCHANGE", SoulChange, "사람효과"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name       string
		id         int
		funcName   string
		duration   time.Duration
		targetSize float64
	}{
		{"EF_BABYBODY", BabyBody, "EffectSmallTransition", 300 * time.Millisecond, 2.5},
		{"EF_BABYBODY2", BabyBody2, "EffectSmall", 5 * time.Minute, 2.5},
		{"EF_GIANTBODY", GiantBody, "EffectBigTransition", 300 * time.Millisecond, 7.5},
		{"EF_GIANTBODY2", GiantBody2, "EffectBig", 5 * time.Minute, 7.5},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || spec.Duration != tc.duration {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentFUNC || component.FuncName != tc.funcName || component.FuncAdapter != EffectFuncUnknown || component.Duration != tc.duration || component.SizeEnd != tc.targetSize || !component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	assumptio, ok := Lookup(Assumptio2)
	if !ok || len(assumptio.Components) != 1 {
		t.Fatalf("EF_ASSUMPTIO2 spec = %+v ok=%t", assumptio, ok)
	}
	if len(assumptio.Sfx) != 1 || assumptio.Sfx[0] != "effect\\아숨프티오.wav" {
		t.Fatalf("EF_ASSUMPTIO2 sfx = %#v", assumptio.Sfx)
	}
	if component := assumptio.Components[0]; component.Kind != ComponentSTR || component.StrFile != "asum" || !component.AttachedEntity {
		t.Fatalf("EF_ASSUMPTIO2 component = %+v", component)
	}
}

func TestRobrowserSimpleEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	darkCross, ok := Lookup(DarkGrandCross)
	if !ok || len(darkCross.Components) != 0 || len(darkCross.Sfx) != 0 {
		t.Fatalf("EF_GRANDCROSS2 spec = %+v ok=%t, want empty robr row", darkCross, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_NPC_STOP", NPCStop, "스톱"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSPR || component.SpriteFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_MOCHI", Mochi, "찹쌀떡"},
		{"EF_LAMADAN", Ramadan, "ramadan"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
	}

}

func TestRobrowserDarkCombatEffectsFourFiftyToFiveHundredMatchTableRows(t *testing.T) {
	soul, ok := Lookup(DarkSoulStrike)
	if !ok || len(soul.Components) != 2 || soul.Duration != 450*time.Millisecond {
		t.Fatalf("EF_SOULSTRIKE2 spec = %+v ok=%t", soul, ok)
	}
	if len(soul.Sfx) != 1 || soul.Sfx[0] != "effect\\ef_soulstrike.wav" {
		t.Fatalf("EF_SOULSTRIKE2 sfx = %#v", soul.Sfx)
	}
	spark, particle := soul.Components[0], soul.Components[1]
	if spark.Kind != Component3D || spark.TextureFile != "effect/pok3.tga" || spark.Duration != 200*time.Millisecond || spark.Delay != 250*time.Millisecond || spark.DuplicateDelay != 150*time.Millisecond || !spark.FadeIn || !spark.FadeOut || !spark.ToSrc || spark.PosZEnd != 1 || !spark.PosZSmooth || spark.PosZStartRand != 5 || spark.PosZStartMiddle != 6 || spark.SizeStart != testutil.EffectTableSize(50) || !spark.AttachedEntity {
		t.Fatalf("EF_SOULSTRIKE2 spark = %+v", spark)
	}
	if particle.Kind != Component3D || particle.SpriteFile != "data/sprite/이팩트/particle5" || !particle.SpriteRepeat || particle.Duration != 250*time.Millisecond || particle.Duplicate != 5 || particle.DuplicateDelay != 20*time.Millisecond || !particle.ToSrc || !particle.RotateToTarget || particle.SizeStart != testutil.EffectTableSize(100) || particle.SizeEnd != testutil.EffectTableSize(500) || particle.PosZ != 3 || particle.Arc != 4 || particle.Retreat != 4 {
		t.Fatalf("EF_SOULSTRIKE2 particle = %+v", particle)
	}

	jupitel, ok := Lookup(DarkJupitelHit)
	if !ok || len(jupitel.Components) != 2 || jupitel.Duration != 300*time.Millisecond {
		t.Fatalf("EF_YUFITEL2 spec = %+v ok=%t", jupitel, ok)
	}
	pang, blast := jupitel.Components[0], jupitel.Components[1]
	if pang.Kind != Component3D || pang.TextureFile != "effect/pokjuk_d.bmp" || pang.Duration != 100*time.Millisecond || pang.SizeStart != 0 || pang.SizeEnd != testutil.EffectTableSize(25) || pang.BlendMode != 2 || !pang.BlendAdditive || !pang.RotateToTarget || !pang.FadeOut || !pang.Overlay || !pang.AttachedEntity {
		t.Fatalf("EF_YUFITEL2 pang = %+v", pang)
	}
	if blast.Kind != Component3D || len(blast.TextureFiles) != 5 || blast.TextureFiles[0] != "effect/twirl_soft.bmp" || blast.TextureFiles[1] != "effect/thunder_ball_b.bmp" || blast.TextureFiles[3] != "effect/thunder_ball_c.bmp" || blast.FrameDelay != 10*time.Millisecond || blast.Duration != 300*time.Millisecond || blast.SizeStart != testutil.EffectTableSize(75) || blast.BlendMode != 2 || !blast.BlendAdditive || !blast.Overlay || !blast.AttachedEntity {
		t.Fatalf("EF_YUFITEL2 blast = %+v", blast)
	}

	casting, ok := Lookup(DarkCasting)
	if !ok || len(casting.Components) != 1 || casting.Duration != 900*time.Millisecond {
		t.Fatalf("EF_DARKCASTING spec = %+v ok=%t", casting, ok)
	}
	if len(casting.Sfx) != 1 || casting.Sfx[0] != "effect\\ef_beginspell.wav" {
		t.Fatalf("EF_DARKCASTING sfx = %#v", casting.Sfx)
	}
	ring := casting.Components[0]
	if ring.Kind != ComponentCylinder || ring.TextureName != "ring_black" || ring.AlphaMax != 0.8 || ring.Animation != 2 || ring.BlendMode != 2 || !ring.BlendAdditive || ring.BottomSize != 1 || ring.TopSize != 5 || ring.Height != 4 || !ring.Fade || !ring.Rotate || !ring.AttachedEntity {
		t.Fatalf("EF_DARKCASTING ring = %+v", ring)
	}
}

func TestRobrowserSimpleEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_MAPAE", Mapae, "mapae", "effect\\mapae.wav"},
		{"EF_ITEMPOKJUK", ItemPokJuk, "itempokjuk", "effect\\itempokjuk.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_05VAL", Valentine05, "05vallentine", ""},
		{"EF_ITEMFAST", ItemFastDown, "fast", "effect\\fast.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSPR || component.SpriteFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached SPR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

}

func TestRobrowserFuncEffectsFiveHundredToFiveFiftyMatchTableRows(t *testing.T) {
	chookgi, ok := Lookup(Chookgi2)
	if !ok || chookgi.Duration != 5*time.Minute || len(chookgi.Components) != 1 {
		t.Fatalf("EF_CHOOKGI2 spec = %+v ok=%t", chookgi, ok)
	}
	sphere := chookgi.Components[0]
	if sphere.Kind != ComponentFUNC || sphere.FuncName != "SpiritSphere" || sphere.FuncAdapter != EffectFuncSpiritSphere || sphere.TextureFile != "effect/thunder_center.bmp" || sphere.Duplicate != 5 || !sphere.AttachedEntity {
		t.Fatalf("EF_CHOOKGI2 component = %+v", sphere)
	}

}

func TestRobrowserSimpleEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_HFLIMOON1", HfliMoon1, "moonlight_1", "effect\\h_moonlight_1.wav"},
		{"EF_HFLIMOON2", HfliMoon2, "moonlight_2", "effect\\h_moonlight_2.wav"},
		{"EF_HFLIMOON3", HfliMoon3, "moonlight_3", "effect\\h_moonlight_3.wav"},
		{"EF_HO_UP", HoUp, "h_levelup", ""},
		{"EF_HAMIDEFENCE", HamiDefence, "defense", ""},
		{"EF_FOOD01", StatFoodSTR, "food_str", "_heal_effect.wav"},
		{"EF_FOOD02", StatFoodINT, "food_int", "_heal_effect.wav"},
		{"EF_FOOD03", StatFoodVIT, "food_vit", "_heal_effect.wav"},
		{"EF_FOOD04", StatFoodAGI, "food_agi", "_heal_effect.wav"},
		{"EF_FOOD05", StatFoodDEX, "food_dex", "_heal_effect.wav"},
		{"EF_FOOD06", StatFoodLUK, "food_luk", "_heal_effect.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name      string
		id        int
		file      string
		wav       string
		direction bool
	}{
		{"EF_HAMICASTLE", HamiCastle, "캐슬링", "", false},
		{"EF_HAMIBLOOD", HamiBlood, "블러드러스트", "", false},
		{"EF_ITEM_THUNDER", ItemThunder, "item_thunder", "", false},
		{"EF_ITEM_CLOUD", ItemCloud, "item_cloud", "", false},
		{"EF_ITEM_CURSE", ItemCurse, "item_curse", "", false},
		{"EF_ITEM_ZZZ", ItemZZZ, "item_zzz", "_snore.wav", false},
		{"EF_ITEM_RAIN", ItemRain, "item_rain", "", false},
		{"EF_M01", M01, "m_ef01", "", false},
		{"EF_M03", M03, "m_ef03", "", false},
		{"EF_M04", M04, "m_ef04", "", false},
		{"EF_M05", M05, "m_ef05", "dragon_breath.wav", false},
		{"EF_M06", M06, "m_ef06", "", false},
		{"EF_M07", M07, "m_ef07", "", false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one SPR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSPR || component.SpriteFile != tc.file || !component.AttachedEntity || component.SpriteDirection != tc.direction {
			t.Fatalf("%s component = %+v, want attached SPR %q direction=%t", tc.name, component, tc.file, tc.direction)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}
}

func TestRobrowserFunctionAndProjectileEffectsFiveFiftyToSixHundredMatchTableRows(t *testing.T) {
	quake, ok := Lookup(ScreenQuake)
	if !ok || quake.Duration != 200*time.Millisecond || quake.CameraShake != 200*time.Millisecond || len(quake.Components) != 1 {
		t.Fatalf("EF_SCREEN_QUAKE spec = %+v ok=%t", quake, ok)
	}
	if component := quake.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "CameraQuake" || component.Duration != 200*time.Millisecond || !component.AttachedEntity {
		t.Fatalf("EF_SCREEN_QUAKE component = %+v", component)
	}

	throw, ok := Lookup(ThrowItem6)
	if !ok || throw.Duration != 200*db.EffectFrameDuration || len(throw.Components) != 1 {
		t.Fatalf("EF_THROWITEM6 spec = %+v ok=%t", throw, ok)
	}
	component := throw.Components[0]
	if component.Kind != ComponentFUNC || component.FuncAdapter != EffectFuncThrowItem || component.TextureFile != "유저인터페이스/item/베넘나이프.bmp" || component.AttachedEntity {
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
		{"EF_FIREHIT2", FireHit2, "firehit%d", "", true, true},
		{"EF_COOKING_OK", CookingOK, "cook_suc", "_heal_effect.wav", true, false},
		{"EF_COOKING_FAIL", CookingFail, "cook_fail", "caramel_die.wav", true, false},
		{"EF_BAKU", Baku, "fire dragon", "effect\\폭염룡.wav", false, false},
		{"EF_HYOUSYOURAKU", Hyousyouraku, "icy", "effect\\빙정락.wav", false, false},
		{"EF_TRACKCASTING", TrackCasting, "트랙킹", "", true, false},
		{"EF_BULLSEYE", Bullseye, "불스아이", "", true, false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.rand {
			if component.StrRandMin != 1 || component.StrRandMax != 3 {
				t.Fatalf("%s STR rand = %d..%d", tc.name, component.StrRandMin, component.StrRandMax)
			}
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
		if tc.rand && (spec.SfxRandMin != 1 || spec.SfxRandMax != 3) {
			t.Fatalf("%s sfx rand = %d..%d", tc.name, spec.SfxRandMin, spec.SfxRandMax)
		}
	}

	hapgyeok, ok := Lookup(Hapgyeok)
	if !ok || len(hapgyeok.Components) != 2 {
		t.Fatalf("EF_HAPGYEOK spec = %+v ok=%t", hapgyeok, ok)
	}
	if len(hapgyeok.Sfx) != 1 || hapgyeok.Sfx[0] != "effect\\itempokjuk.wav" {
		t.Fatalf("EF_HAPGYEOK sfx = %#v", hapgyeok.Sfx)
	}
	if spr, str := hapgyeok.Components[0], hapgyeok.Components[1]; spr.Kind != ComponentSPR || spr.SpriteFile != "합격_" || !spr.AttachedEntity || str.Kind != ComponentSTR || str.StrFile != "itempokjuk" || !str.AttachedEntity {
		t.Fatalf("EF_HAPGYEOK components = %+v", hapgyeok.Components)
	}

}

func TestRobrowserFuncEffectsSixHundredToSixFiftyMatchTableRows(t *testing.T) {
	chookgi, ok := Lookup(Chookgi3)
	if !ok || chookgi.Duration != 5*time.Minute || len(chookgi.Components) != 1 {
		t.Fatalf("EF_CHOOKGI3 spec = %+v ok=%t", chookgi, ok)
	}
	if sphere := chookgi.Components[0]; sphere.Kind != ComponentFUNC || sphere.FuncName != "SpiritSphere" || sphere.FuncAdapter != EffectFuncSpiritSphere || sphere.TextureFile != "effect/thunder_center.bmp" || sphere.Duplicate != 5 || !sphere.AttachedEntity {
		t.Fatalf("EF_CHOOKGI3 component = %+v", sphere)
	}
}

func TestRobrowserEffectsSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	earthquake, ok := Lookup(NPCEarthquake)
	if !ok || earthquake.CameraShake != 650*time.Millisecond || len(earthquake.Components) != 2 {
		t.Fatalf("EF_NPC_EARTHQUAKE spec = %+v ok=%t", earthquake, ok)
	}
	if len(earthquake.Sfx) != 1 || earthquake.Sfx[0] != "effect\\earth_quake.wav" {
		t.Fatalf("EF_NPC_EARTHQUAKE sfx = %#v", earthquake.Sfx)
	}
	if spr, quake := earthquake.Components[0], earthquake.Components[1]; spr.Kind != ComponentSPR || spr.SpriteFile != "어스퀘이크" || !spr.AttachedEntity || quake.Kind != ComponentFUNC || quake.FuncName != "CameraQuake" || quake.Duplicate != 3 || quake.DuplicateDelay != 35*time.Millisecond || !quake.AttachedEntity {
		t.Fatalf("EF_NPC_EARTHQUAKE components = %+v", earthquake.Components)
	}

	dragon, ok := Lookup(DragonFear)
	if !ok || dragon.CameraShake != 650*time.Millisecond || len(dragon.Components) != 2 {
		t.Fatalf("EF_DRAGONFEAR spec = %+v ok=%t", dragon, ok)
	}
	if len(dragon.Sfx) != 1 || dragon.Sfx[0] != "effect\\dragonfear.wav" {
		t.Fatalf("EF_DRAGONFEAR sfx = %#v", dragon.Sfx)
	}
	if str, quake := dragon.Components[0], dragon.Components[1]; str.Kind != ComponentSTR || str.StrFile != "dragon_h" || !str.AttachedEntity || quake.Kind != ComponentFUNC || quake.FuncName != "CameraQuake" || !quake.AttachedEntity {
		t.Fatalf("EF_DRAGONFEAR components = %+v", dragon.Components)
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
		wav  string
	}{
		{"EF_BLEEDING", WideBleeding, "wideb", "effect\\wideb.wav"},
		{"EF_WIDECONFUSE", WideConfuse, "dfear", "effect\\dragonfear.wav"},
		{"EF_CRITICALWOUND", CriticalWound, "cwound", ""},
		{"EF_FLOWERLEAF", FlowerLeaf, "flower_leaf", ""},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want attached STR %q", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
	}{
		{"EF_BOTTOM_RUNNER", BottomRunner, "effect/hanmoon1.tga"},
		{"EF_BOTTOM_TRANSFER", BottomTransfer, "effect/hanmoon2.tga"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 1500*time.Millisecond || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one ground texture component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentFUNC || component.FuncName != "GroundTexture" || component.FuncAdapter != EffectFuncGroundTexture || component.TextureFile != tc.texture || component.SizeStart != 1 || component.SizeEnd != 1 || component.PosZ != 0.05 || !component.BlendAdditive || component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	evil, ok := Lookup(BottomEvilLand)
	if !ok || evil.Duration != 1500*time.Millisecond || len(evil.Components) != 2 {
		t.Fatalf("EF_BOTTOM_EVILLAND spec = %+v ok=%t", evil, ok)
	}
	tile, curse := evil.Components[0], evil.Components[1]
	if tile.Kind != ComponentFUNC || tile.FuncName != "FlatColorTile" || tile.FuncAdapter != EffectFuncFlatColorTile || tile.Color != (color.RGBA{R: 160, G: 160, B: 160, A: 51}) || tile.SizeStart != 1 || tile.AttachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND tile = %+v", tile)
	}
	if curse.Kind != ComponentFUNC || curse.FuncName != "GroundTexture" || curse.FuncAdapter != EffectFuncGroundTexture || curse.TextureFile != "effect/curse.bmp" || curse.SizeStart != 1 || curse.SizeEnd != 1 || curse.AlphaMax != 0.7 || curse.PosZ != 0.4 || !curse.BlendAdditive || curse.AttachedEntity {
		t.Fatalf("EF_BOTTOM_EVILLAND curse = %+v", curse)
	}

	guard, ok := Lookup(Guard3)
	if !ok || len(guard.Components) != 0 || guard.Duration != 500*time.Millisecond {
		t.Fatalf("EF_GUARD3 spec = %+v ok=%t, want sound-only", guard, ok)
	}
	if len(guard.Sfx) != 1 || guard.Sfx[0] != "effect\\kyrie_guard.wav" {
		t.Fatalf("EF_GUARD3 sfx = %#v", guard.Sfx)
	}
}

func TestMagicPowerLegacySoundAlias(t *testing.T) {
	magicPower, ok := Lookup(MagicPower)
	if !ok || magicPower.Duration != 500*time.Millisecond || len(magicPower.Components) != 0 {
		t.Fatalf("ef_magicpower spec = %+v ok=%t, want sound-only", magicPower, ok)
	}
	if len(magicPower.Sfx) != 1 || magicPower.Sfx[0] != "effect\\마법력 증폭.wav" {
		t.Fatalf("ef_magicpower sfx = %#v", magicPower.Sfx)
	}
}

func TestRobrowserFirecrackerBannersSixFiftyToSevenHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POK_LOVE", Firecracker2, "폭죽_러브"},
		{"EF_POK_WHITE", Firecracker3, "폭죽_화이트데이"},
		{"EF_POK_VALEN", Firecracker4, "폭죽_발렌타인"},
		{"EF_POK_BIRTH", Firecracker5, "폭죽_생일"},
		{"EF_POK_CHRISTMAS", Firecracker6, "폭죽_크리스마스"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want SPR banner plus STR itempokjuk", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\itempokjuk.wav" {
			t.Fatalf("%s sfx = %#v", tc.name, spec.Sfx)
		}
		if spr, str := spec.Components[0], spec.Components[1]; spr.Kind != ComponentSPR || spr.SpriteFile != tc.file || !spr.AttachedEntity || str.Kind != ComponentSTR || str.StrFile != "itempokjuk" || !str.AttachedEntity {
			t.Fatalf("%s components = %+v", tc.name, spec.Components)
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
		{"EF_ITEM315", Item315, "mobile_ef02", "", true},
		{"EF_ITEM316", Item316, "mobile_ef01", "", true},
		{"EF_ITEM317", Item317, "mobile_ef03", "", true},
		{"EF_STORM_MIN", StormMin, "storm_min", "effect\\wizard_stormgust.wav", true},
		{"EF_POK_JAP", Firecracker7, "pokjuk_jap", "", false},
		{"EF_ADO_STR", Adoramus, "ado", "effect\\ab_adoramus.wav", true},
		{"EF_IGN_STR", IgnitionBreak, "이그니션브레이크", "effect\\wl_jackfrost.wav", true},
		{"EF_CRIMSON_STR", CrimsonRock, "crimson_r", "effect\\crimson_r.wav", true},
		{"EF_HELL_STR", HellInferno, "hell_in", "", true},
		{"EF_DHOWL_STR", DragonHowling, "dragon_h", "dragon_h.wav", true},
		{"EF_CHAINL_STR", ChainLightning, "chainlight", "effect\\chainlight.wav", true},
		{"EF_AIMED_STR", AimedBolt, "aimed", "", true},
		{"EF_ARROWSTORM_STR", ArrowStorm, "arrowstorm", "", true},
		{"EF_LAULAMUS_STR", Laulamus, "laulamus", "", true},
		{"EF_LAUAGNUS_STR", Lauagnus, "lauagnus", "", true},
		{"EF_MILSHIELD_STR", MillenniumShield, "mil_shield", "", true},
		{"EF_CONCENTRATION2", Concentration2, "concentration", "", true},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.AttachedEntity != tc.attached {
			t.Fatalf("%s component = %+v, want STR %q attached=%t", tc.name, component, tc.file, tc.attached)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	wewish, ok := Lookup(ChristmasCarol)
	if !ok || len(wewish.Sfx) != 1 || wewish.Sfx[0] != "effect\\wewish.wav" {
		t.Fatalf("EF_WEWISH spec = %+v ok=%t", wewish, ok)
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_FORESTLIGHT5", ForestLight5, "effect\\ab_renovation.wav"},
		{"EF_FROSTMYSTY", FrostMisty, "effect\\t_에나지방출.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 0 || spec.Duration != 500*time.Millisecond {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	marsh, ok := Lookup(MarshOfAbyss)
	if !ok || len(marsh.Components) != 1 {
		t.Fatalf("EF_SPR_MASH spec = %+v ok=%t, want one SPR component", marsh, ok)
	}
	if component := marsh.Components[0]; component.Kind != ComponentSPR || component.SpriteFile != "mashofa" || component.AttachedEntity {
		t.Fatalf("EF_SPR_MASH component = %+v", component)
	}
}

func TestRobrowserCylinderEffectsSevenHundredToSevenFiftyMatchTableRows(t *testing.T) {
	for _, id := range []int{BottomBlue, BottomBlue2} {
		spec, ok := Lookup(id)
		if !ok || spec.Duration != 20*time.Second || len(spec.Components) != 4 {
			t.Fatalf("bottom blue effect %d spec = %+v ok=%t", id, spec, ok)
		}
		for i, component := range spec.Components {
			if component.Kind != ComponentCylinder || component.TextureName != "alpha_down" || component.Duration != 20*time.Second || component.TotalCircleSides != 4 || component.CircleSides != 4 || !component.RotateWithCamera || component.BlendMode != 2 || !component.BlendAdditive || !component.AttachedEntity {
				t.Fatalf("bottom blue effect %d component %d = %+v", id, i, component)
			}
		}
		if first := spec.Components[0]; first.BottomSize != 1.5 || first.TopSize != 1.5 || first.Height != 2 || first.AlphaMax != 40.0/255.0 || first.AngleY != 0 || first.Color != (color.RGBA{R: 51, G: 153, B: 255, A: 255}) {
			t.Fatalf("bottom blue first component = %+v", first)
		}
		if second := spec.Components[1]; second.BottomSize != 1.58 || second.Height != 2.1 || second.AlphaMax != 32.0/255.0 || second.AngleY != 10 {
			t.Fatalf("bottom blue second component = %+v", second)
		}
		if third := spec.Components[2]; third.BottomSize != 1.65 || third.AlphaMax != 15.0/255.0 || third.AngleY != 26.6 || third.Color != (color.RGBA{R: 25, G: 102, B: 255, A: 255}) {
			t.Fatalf("bottom blue third component = %+v", third)
		}
		if fourth := spec.Components[3]; fourth.BottomSize != 1.65 || fourth.AlphaMax != 15.0/255.0 || fourth.AngleY != 79.8 {
			t.Fatalf("bottom blue fourth component = %+v", fourth)
		}
	}

	judex, ok := Lookup(FirePillarOn2)
	if !ok || judex.Duration != time.Second || len(judex.Components) != 3 {
		t.Fatalf("EF_FIREPILLARON2 spec = %+v ok=%t", judex, ok)
	}
	if len(judex.Sfx) != 1 || judex.Sfx[0] != "effect\\ab_judex.wav" {
		t.Fatalf("EF_FIREPILLARON2 sfx = %#v", judex.Sfx)
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
	for i, component := range judex.Components {
		if component.Kind != ComponentCylinder || component.TextureName != "ring_white" || component.Duration != time.Second || component.AttachedEntity || !component.Rotate || component.BottomSize != want[i].bottom || component.TopSize != want[i].top || component.Height != want[i].height {
			t.Fatalf("EF_FIREPILLARON2 component %d = %+v", i, component)
		}
	}
}

func TestRobrowserEarthWallEffectSevenHundredToSevenFiftyMatchesTableRow(t *testing.T) {
	spec, ok := Lookup(EarthWall)
	if !ok || spec.Duration != time.Second || spec.CameraShake != 200*time.Millisecond || len(spec.Components) != 2 {
		t.Fatalf("EF_EARTHWALL spec = %+v ok=%t", spec, ok)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\wizard_earthspike.wav" {
		t.Fatalf("EF_EARTHWALL sfx = %#v", spec.Sfx)
	}
	horn, quake := spec.Components[0], spec.Components[1]
	if horn.Kind != ComponentQuadHorn || horn.TextureFile != "effect/stone.bmp" || horn.Duration != time.Second || horn.QuadHornHeightMin != 0.75 || horn.QuadHornHeightMax != 1.2 || horn.QuadHornOffsetXMin != 0.2 || horn.QuadHornOffsetXMax != 0.2 || horn.QuadHornOffsetYMin != 0.2 || horn.QuadHornOffsetYMax != 0.2 || horn.QuadHornOffsetZ != -0.1 || horn.QuadHornBottomMin != 0.4 || horn.QuadHornBottomMax != 0.9 || horn.BlendMode != 1 || horn.QuadHornRotateYMin != 1 || horn.QuadHornRotateYMax != 360 || horn.QuadHornRotateZMin != -8 || horn.QuadHornRotateZMax != 8 || horn.Animation != 3 || horn.QuadHornAnimSpeed != 250*time.Millisecond || !horn.QuadHornAnimOut {
		t.Fatalf("EF_EARTHWALL horn = %+v", horn)
	}
	if quake.Kind != ComponentFUNC || quake.FuncName != "CameraQuake" || !quake.AttachedEntity {
		t.Fatalf("EF_EARTHWALL quake = %+v", quake)
	}
}

func TestRobrowserEffectsSevenFiftyToEightHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_POTION_BERSERK2", BerserkPotion2, "버서크"},
		{"EF_CRASHAXE", CrashAxe, "powerswing"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if len(spec.Sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
		}
	}

	spin, ok := Lookup(CastSpin2)
	if !ok || spin.Duration != 500*time.Millisecond || len(spin.Components) != 1 {
		t.Fatalf("EF_CASTSPIN2 spec = %+v ok=%t", spin, ok)
	}
	if component := spin.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "CastSpin2" || !component.AttachedEntity {
		t.Fatalf("EF_CASTSPIN2 component = %+v", component)
	}

	stasis, ok := Lookup(Stasis)
	if !ok || stasis.Duration != 500*time.Millisecond || len(stasis.Components) != 0 {
		t.Fatalf("EF_STASIS spec = %+v ok=%t", stasis, ok)
	}
	if len(stasis.Sfx) != 1 || stasis.Sfx[0] != "effect\\wl_stasis.wav" {
		t.Fatalf("EF_STASIS sfx = %#v", stasis.Sfx)
	}
}

func TestRobrowserGlassWall3EffectMatchesTableRow(t *testing.T) {
	spec, ok := Lookup(GlassWall3)
	if !ok || len(spec.Components) != 4 {
		t.Fatalf("EF_GLASSWALL3 spec = %+v ok=%t", spec, ok)
	}
	tint := color.RGBA{R: 153, G: 255, B: 153, A: 255}
	first := spec.Components[0]
	if first.Kind != ComponentCylinder || first.TextureName != "magic_green" || first.Color != tint {
		t.Fatalf("EF_GLASSWALL3 first identity = %+v", first)
	}
	if first.Duration != 500*time.Millisecond || first.AlphaMax != 0.4 || first.Animation != 4 || first.BottomSize != 2.4 || first.TopSize != 3.9 || first.Height != 0.1 || first.PosZ != 0.1 {
		t.Fatalf("EF_GLASSWALL3 first timing/geometry = %+v", first)
	}
	if first.Duplicate != 150 || first.DuplicateDelay != 200*time.Millisecond || !first.FadeOut || !first.Rotate || first.BlendMode != 2 || !first.BlendAdditive || !first.AttachedEntity {
		t.Fatalf("EF_GLASSWALL3 first flags = %+v", first)
	}
	if got := WorldEffectComponentDuration(spec, first); got != 30300*time.Millisecond {
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
		component := spec.Components[i+1]
		texture := "magic_green"
		if i == 2 {
			texture = "alpha1"
		}
		if component.Kind != ComponentCylinder || component.TextureName != texture || component.Color != tint {
			t.Fatalf("EF_GLASSWALL3 component %d identity = %+v", i+1, component)
		}
		if component.Duration != 30*time.Second || component.AlphaMax != want.alpha || component.BottomSize != want.bottom || component.TopSize != want.top || component.Height != want.height || component.PosZ != want.posZ {
			t.Fatalf("EF_GLASSWALL3 component %d geometry = %+v", i+1, component)
		}
		if !component.Fade || !component.Rotate || component.BlendMode != 2 || !component.BlendAdditive || !component.AttachedEntity || component.TotalCircleSides != want.sides || component.CircleSides != want.circle {
			t.Fatalf("EF_GLASSWALL3 component %d flags = %+v", i+1, component)
		}
	}
}

func TestRobrowserRollingCutterCounterEffectsMatchTableRows(t *testing.T) {
	ids := []int{
		Rolling1,
		Rolling2,
		Rolling3,
		Rolling4,
		Rolling5,
		Rolling6,
		Rolling7,
		Rolling8,
		Rolling9,
		Rolling10,
	}
	for i, id := range ids {
		spec, ok := Lookup(id)
		if !ok || spec.Duration != time.Second || len(spec.Components) != 5 {
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
			component := spec.Components[j]
			if component.Kind != Component3D || component.TextureFile != texture || component.Duration != time.Second {
				t.Fatalf("EF_ROLLING%d component %d identity = %+v", i+1, j, component)
			}
			if component.AlphaMax != want.alpha || component.Color != want.color || component.SizeStart != testutil.EffectTableSize(want.sizeStart) || component.SizeEnd != testutil.EffectTableSize(20) {
				t.Fatalf("EF_ROLLING%d component %d visual = %+v", i+1, j, component)
			}
			if !component.FadeIn || !component.FadeOut || component.PosZ != 4 || !component.SizeSmooth || component.BlendMode != want.blendMode || component.BlendAdditive != want.additive || !component.AttachedEntity {
				t.Fatalf("EF_ROLLING%d component %d flags = %+v", i+1, j, component)
			}
		}
	}
}

func TestRobrowserBottomBasilica2EffectMatchesTableRow(t *testing.T) {
	spec, ok := Lookup(BottomBasilica2)
	if !ok || spec.Duration != 20*time.Second || len(spec.Components) != 4 {
		t.Fatalf("EF_BOTTOM_BASILICA2 spec = %+v ok=%t", spec, ok)
	}
	if len(spec.Sfx) != 1 || spec.Sfx[0] != "effect\\wl_whiteimprison.wav" {
		t.Fatalf("EF_BOTTOM_BASILICA2 sfx = %#v", spec.Sfx)
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
		component := spec.Components[i]
		if component.Kind != ComponentCylinder || component.TextureName != "alpha_down" || component.Duration != 20*time.Second {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d identity = %+v", i, component)
		}
		if component.TotalCircleSides != 4 || component.CircleSides != 4 || component.BottomSize != want.size || component.TopSize != want.size || component.Height != want.height || math.Abs(component.AlphaMax-want.alpha) > 0.0001 || component.AngleY != want.angleY {
			t.Fatalf("EF_BOTTOM_BASILICA2 component %d geometry = %+v", i, component)
		}
		if component.BlendMode != 2 || !component.BlendAdditive || !component.RotateWithCamera || !component.AttachedEntity {
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
		{"EF_ENERVATION", Enervation, "enervation", ""},
		{"EF_ENERVATION2", Enervation2, "groomy", ""},
		{"EF_ENERVATION3", Enervation3, "ignorance", ""},
		{"EF_ENERVATION4", Enervation4, "laziness", "effect\\laziness.wav"},
		{"EF_ENERVATION5", Enervation5, "unlucky", ""},
		{"EF_ENERVATION6", Enervation6, "weakness", ""},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want STR %q attached", tc.name, component, tc.file)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
			continue
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RECOGNIZED", Recognized, "effect\\wl_recognizedspell.wav"},
		{"EF_TETRA", Tetra, "effect\\wl_tetravortex.wav"},
		{"EF_STRETCH", Stretch, "effect\\bodypaint.wav"},
		{"EF_BOTTOM_MANHOLE", BottomManhole, "effect\\dimension.wav"},
		{"EF_MANHOLE", Manhole, "effect\\manhole.wav"},
		{"EF_FORESTLIGHT6", ForestLight6, "effect\\dimension.wav"},
		{"EF_BOTTOM_ANI", BottomAni, "effect\\chaospanic.wav"},
		{"EF_BOTTOM_MAELSTROM", BottomMaelstrom, "effect\\maelstrom.wav"},
		{"EF_BOTTOM_BLOODYLUST", BottomBloodyLust, "effect\\bloodylust.wav"},
		{"EF_HEAL_N", HealN, "effect\\기공포.wav"},
		{"EF_DANCE1", Dance1, "effect\\수줍은하루의우울.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 500*time.Millisecond || len(spec.Components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	tetraCasting, ok := Lookup(TetraCasting)
	if !ok || tetraCasting.Duration != 500*time.Millisecond || len(tetraCasting.Components) != 1 {
		t.Fatalf("EF_TETRACASTING spec = %+v ok=%t", tetraCasting, ok)
	}
	if component := tetraCasting.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "TetraCasting" || component.FuncAdapter != EffectFuncUnknown || !component.AttachedEntity {
		t.Fatalf("EF_TETRACASTING component = %+v", component)
	}

	chookgi, ok := Lookup(ChookgiN)
	if !ok || chookgi.Duration != 5*time.Minute || len(chookgi.Components) != 1 {
		t.Fatalf("EF_CHOOKGI_N spec = %+v ok=%t", chookgi, ok)
	}
	if component := chookgi.Components[0]; component.Kind != ComponentFUNC || component.FuncName != "SpiritSphere" || component.FuncAdapter != EffectFuncSpiritSphere || !component.AttachedEntity {
		t.Fatalf("EF_CHOOKGI_N component = %+v", component)
	}
}

func TestRobrowserEffectsEightFiftyToNineHundredMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_RAIN_PARTICLE", RainParticle, "effect\\rainstorm.wav"},
		{"EF_CHEMICAL_V2", ChemicalV2, "effect\\안식의자장가.wav"},
		{"EF_CIRCLEPOWER2", CirclePower2, "effect\\순환하는자연의소리.wav"},
		{"EF_SPR_PLANT2", SprPlant2, "effect\\워그와함께춤을.wav"},
		{"EF_SPR_PLANT3", SprPlant3, "effect\\마나의노래.wav"},
		{"EF_SPR_PLANT4", SprPlant4, "effect\\새터데이나이트피버.wav"},
		{"EF_SPR_PLANT5", SprPlant5, "effect\\레라드의이슬.wav"},
		{"EF_SPR_PLANT6", SprPlant6, "effect\\멜로디오브싱크.wav"},
		{"EF_SPR_PLANT7", SprPlant7, "effect\\비욘드오브워크라이.wav"},
		{"EF_SPR_PLANT8", SprPlant8, "effect\\언리미티드허밍보이스.wav"},
		{"EF_HEARTASURA", HeartAsura, "effect\\세이렌의목소리.wav"},
		{"EF_ELECTRIC4", Electric4, "effect\\sr_earthshaker.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 500*time.Millisecond || len(spec.Components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name    string
		id      int
		texture string
		wav     string
		tint    color.RGBA
	}{
		{"EF_BOT_REVERB", BotReverb, "effect/melody_b.bmp", "effect\\reverberation.wav", color.RGBA{R: 255, G: 153, B: 153, A: 255}},
		{"EF_BOT_REVERB2", BotReverb2, "effect/melody_a.bmp", "effect\\나락의노래.wav", color.RGBA{R: 153, G: 153, B: 255, A: 255}},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 100*time.Millisecond || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
		component := spec.Components[0]
		if component.Kind != Component3D || component.TextureFile != tc.texture || component.Color != tc.tint || component.Duration != 100*time.Millisecond || component.AlphaMax != 0.6 || !component.AttachedEntity || !component.Repeat || component.PosZ != 0.5 || component.SizeStart != testutil.EffectTableSize(50) || component.SizeEnd != testutil.EffectTableSize(50) {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}

	secra, ok := Lookup(Secra2)
	if !ok || secra.Duration != 1500*time.Millisecond || len(secra.Components) != 10 {
		t.Fatalf("EF_SECRA2 spec = %+v ok=%t", secra, ok)
	}
	if len(secra.Sfx) != 1 || secra.Sfx[0] != "effect\\ab_ancilla.wav" {
		t.Fatalf("EF_SECRA2 sfx = %#v", secra.Sfx)
	}
	firstSecra, lastSecra := secra.Components[0], secra.Components[len(secra.Components)-1]
	if firstSecra.Kind != Component3D || firstSecra.TextureFile != "effect/priest_spell.bmp" || firstSecra.Color != (color.RGBA{R: 255, G: 140, B: 140, A: 255}) || firstSecra.Duration != 1500*time.Millisecond || firstSecra.AlphaMax != 0.3 || firstSecra.BlendMode != 2 || !firstSecra.BlendAdditive || !firstSecra.FadeIn || !firstSecra.FadeOut || !firstSecra.AttachedEntity || firstSecra.PosZ != 7 || firstSecra.SizeStart != testutil.EffectTableSize(850) || firstSecra.SizeEnd != testutil.EffectTableSize(100) || !firstSecra.SizeSmooth {
		t.Fatalf("EF_SECRA2 first component = %+v", firstSecra)
	}
	if lastSecra.Color != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) || lastSecra.SizeStart != testutil.EffectTableSize(400) || lastSecra.SizeEnd != testutil.EffectTableSize(100) {
		t.Fatalf("EF_SECRA2 last component = %+v", lastSecra)
	}

	glass, ok := Lookup(GlassWall4)
	if !ok || glass.Duration != 30*time.Second || len(glass.Components) != 3 {
		t.Fatalf("EF_GLASSWALL4 spec = %+v ok=%t", glass, ok)
	}
	if len(glass.Sfx) != 1 || glass.Sfx[0] != "effect\\ef_readyportal.wav" {
		t.Fatalf("EF_GLASSWALL4 sfx = %#v", glass.Sfx)
	}
	tree, pulseA, pulseB := glass.Components[0], glass.Components[1], glass.Components[2]
	if tree.Kind != Component3D || tree.TextureFile != "effect/ef_epitree.tga" || tree.Color != (color.RGBA{G: 255, A: 255}) || tree.Duration != 30*time.Second || tree.AlphaMax != 0.6 || tree.BlendMode != 2 || !tree.BlendAdditive || !tree.AttachedEntity || tree.PosZ != 7 || tree.SizeStart != testutil.EffectTableSize(400) || tree.SizeEnd != testutil.EffectTableSize(400) {
		t.Fatalf("EF_GLASSWALL4 tree = %+v", tree)
	}
	if pulseA.Duration != 990*time.Millisecond || pulseA.Duplicate != 15 || pulseA.DuplicateDelay != 2*time.Second || pulseA.Delay != 0 || pulseA.SizeStart != testutil.EffectTableSize(380) || pulseA.SizeEnd != testutil.EffectTableSize(420) {
		t.Fatalf("EF_GLASSWALL4 first pulse = %+v", pulseA)
	}
	if pulseB.Delay != time.Second || pulseB.SizeStart != testutil.EffectTableSize(420) || pulseB.SizeEnd != testutil.EffectTableSize(380) {
		t.Fatalf("EF_GLASSWALL4 second pulse = %+v", pulseB)
	}

	bash, ok := Lookup(Bash3D6)
	if !ok || bash.Duration != 500*time.Millisecond || len(bash.Components) != 3 {
		t.Fatalf("EF_BASH3D6 spec = %+v ok=%t", bash, ok)
	}
	if len(bash.Sfx) != 1 || bash.Sfx[0] != "effect\\bash3d.wav" {
		t.Fatalf("EF_BASH3D6 sfx = %#v", bash.Sfx)
	}
	if body := bash.Components[0]; body.Kind != ComponentFUNC || body.FuncName != "Bash3D6" || body.FuncAdapter != EffectFuncUnknown || !body.AttachedEntity {
		t.Fatalf("EF_BASH3D6 body = %+v", body)
	}
	for i, component := range bash.Components[1:] {
		wantTop := 4.5
		if i == 1 {
			wantTop = 7.2
		}
		if component.Kind != ComponentCylinder || component.TextureName != "alpha_center" || component.Color != (color.RGBA{R: 76, G: 127, B: 255, A: 255}) || component.Duration != 175*time.Millisecond || component.Delay != 200*time.Millisecond || component.Duplicate != 5 || component.AlphaMax != 0.6 || !component.Fade || component.AngleX != -90 || component.AngleZRandom != 360 || !component.FixedPerspective || component.PosZ != 1.5 || component.BottomSize != 0.01 || component.TopSize != wantTop || component.Animation != 2 || !component.AttachedEntity {
			t.Fatalf("EF_BASH3D6 cylinder %d = %+v", i, component)
		}
	}

	teiHit, ok := Lookup(TeiHit1T)
	if !ok || teiHit.Duration != 350*time.Millisecond || len(teiHit.Components) != 1 {
		t.Fatalf("EF_TEIHIT1T spec = %+v ok=%t", teiHit, ok)
	}
	if len(teiHit.Sfx) != 1 || teiHit.Sfx[0] != "effect\\mon_아수라 패황권.wav" {
		t.Fatalf("EF_TEIHIT1T sfx = %#v", teiHit.Sfx)
	}
	component := teiHit.Components[0]
	if component.Kind != Component3D || component.TextureFile != "effect/lens1.tga" || component.Color != (color.RGBA{R: 25, G: 25, B: 255, A: 255}) || component.Duration != 250*time.Millisecond || component.Delay != 100*time.Millisecond || component.Duplicate != 24 || component.DuplicateDelay != 0 || component.AlphaMax != 0.8 || component.BlendMode != 2 || !component.BlendAdditive || !component.FadeIn || !component.FadeOut || !component.AttachedEntity || component.PosXEndRand != 40 || component.PosYEndRand != 40 || component.SizeStartX != testutil.EffectTableSize(10) || component.SizeStartY != testutil.EffectTableSize(150) || component.SizeEndX != testutil.EffectTableSize(10) || component.SizeEndY != testutil.EffectTableSize(150) || !component.Overlay || !component.RotateToTarget || !component.RotateWithCamera {
		t.Fatalf("EF_TEIHIT1T component = %+v", component)
	}
}

func TestRobrowserEffectsNineHundredToNineFiftyMatchTableRows(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   int
		wav  string
	}{
		{"EF_PRIMECHARGE2", PrimeCharge2, "effect\\lg_prestige.wav"},
		{"EF_PRIMECHARGE3", PrimeCharge3, "effect\\lg_banding.wav"},
		{"EF_PRIMECHARGE4", PrimeCharge4, "effect\\lg_inspiration.wav"},
		{"EF_SPR_PLANT10", SprPlant10, "effect\\s사이킥웨이브.wav"},
		{"EF_COLDTHROW2", ColdThrow2, "effect\\wl_jackfrost.wav"},
		{"EF_DEMONICFIRE4", DemonicFire4, "effect\\s워머.wav"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 500*time.Millisecond || len(spec.Components) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want sound-only 500ms", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
	}

	for _, tc := range []struct {
		name string
		id   int
		file string
	}{
		{"EF_FIREWALL2", FireWall2, "firewall_per"},
		{"EF_SHOCKWAVE2", Shockwave2, "hunter_shockwave_blue"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || len(spec.Sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.Components[0]; component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
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
		{"EF_PRESSURE2", Pressure2, "effect/shield.bmp", []string{"effect\\프레셔.wav", "effect\\lg_shieldpress.wav"}, []time.Duration{0, 500 * time.Millisecond}},
		{"EF_PRESSURE3", Pressure3, "effect/cross1.bmp", []string{"effect\\프레셔.wav"}, nil},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 1001*time.Millisecond || len(spec.Components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t", tc.name, spec, ok)
		}
		if spec.CameraShake != 0 || spec.CameraShakeDelay != 0 {
			t.Fatalf("%s camera shake = %s delay %s, want none", tc.name, spec.CameraShake, spec.CameraShakeDelay)
		}
		if !reflect.DeepEqual(spec.Sfx, tc.sfx) || !reflect.DeepEqual(spec.SfxDelays, tc.delays) {
			t.Fatalf("%s sfx = %#v delays %#v", tc.name, spec.Sfx, spec.SfxDelays)
		}

		first, second := spec.Components[0], spec.Components[1]
		if first.Kind != Component3D || first.TextureFile != tc.texture || first.Duration != 500*time.Millisecond || first.AlphaMax != 0.6 || first.BlendMode != 2 || !first.BlendAdditive || !first.Rotate || first.AngleStart != 0 || first.AngleEnd != -611 || first.PosZ != 20 || first.PosZEnd != 5 || first.SizeStart != testutil.EffectTableSize(100) || first.SizeEnd != testutil.EffectTableSize(100) || !first.AttachedEntity {
			t.Fatalf("%s first component = %+v", tc.name, first)
		}
		if second.Kind != Component3D || second.TextureFile != tc.texture || second.Duration != 500*time.Millisecond || second.Delay != 501*time.Millisecond || second.AlphaMax != 0.6 || second.BlendMode != 2 || !second.BlendAdditive || !second.FadeOut || second.AngleStart != -611 || second.AngleEnd != -611 || second.PosZ != 5 || second.SizeStart != testutil.EffectTableSize(100) || second.SizeEnd != testutil.EffectTableSize(100) || !second.AttachedEntity {
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
		{"EF_POISON_MIST", PoisonMist, "poison_mist"},
		{"EF_ERASER_CUTTER", EraserCutter, "eraser_cutter"},
		{"EF_LAVA_SLIDE", LavaSlide, "lava_slide"},
		{"EF_SONIC_CLAW", SonicClaw, "sonic_claw"},
		{"EF_TINDER_BREAKER", TinderBreaker, "tinder"},
		{"EF_MIDNIGHT_FRENZY", MidnightFrenzy, "mid_frenzy"},
		{"EF_VOLCANIC_ASH", VolcanicAsh, "vash00"},
		{"EF_2011RWC", RWC2011, "rwc2011"},
		{"EF_2011RWC2", RWC2011Two, "rwc2011_2"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 || len(spec.Sfx) != 0 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component and no sound", tc.name, spec, ok)
		}
		if component := spec.Components[0]; component.Kind != ComponentSTR || component.StrFile != tc.file || !component.AttachedEntity {
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
		{"EF_RUN_MAKE_OK", RunMakeOK, "rune_success", "", 0, 0},
		{"EF_RUN_MAKE_FAILURE", RunMakeFailure, "rune_fail", "", 0, 0},
		{"EF_MIRESULT_MAKE_OK", MIResultMakeOK, "changematerial_su", "", 0, 0},
		{"EF_MIRESULT_MAKE_FAIL", MIResultMakeFail, "changematerial_fa", "", 0, 0},
		{"EF_ALL_RAY_OF_PROTECTION", AllRayProtect, "guardian", "", 0, 0},
		{"EF_VENOMFOG", VenomFog, "bubble%d_1", "", 1, 4},
		{"EF_DUSTSTORM", DustStorm, "dust", "", 0, 0},
		{"EF_DANCE_BLADE_ATK", DanceBladeAtk, "dancingblade", "", 0, 0},
		{"EF_INVINCIBLEOFF2", InvincibleOff2, "invincibleoff2", "", 0, 0},
		{"EF_DEATHSUMMON", DeathSummon, "devil", "", 0, 0},
		{"EF_GC_DARKCROW", GCDarkCrow, "gc_darkcrow", "", 0, 0},
		{"EF_ALL_FULL_THROTTLE", AllFullThrottle, "all_full_throttle", "effect\\all_full_throttle.wav", 0, 0},
		{"EF_SR_FLASHCOMBO", SRFlashCombo, "sr_flashcombo", "effect\\sr_flashcombo.wav", 0, 0},
		{"EF_RK_LUXANIMA", RKLuxAnima, "rk_luxanima", "", 0, 0},
		{"EF_SO_ELEMENTAL_SHIELD", SOElemShield, "so_elemental_shield", "effect\\so_elemental_shield.wav", 0, 0},
		{"EF_AB_OFFERTORIUM", ABOffertorium, "ab_offertorium", "effect\\ab_offertorium.wav", 0, 0},
		{"EF_WL_TELEKINESIS_INTENSE", WLTelekinesis, "wl_telekinesis_intense", "effect\\wl_telekinesis_intense.wav", 0, 0},
		{"EF_GN_ILLUSIONDOPING", GNIllusionDoping, "gn_illusiondoping", "effect\\gn_illusiondoping.wav", 0, 0},
		{"EF_NC_MAGMA_ERUPTION", NCMagmaEruption, "nc_magma_eruption", "effect\\nc_magma_eruption.wav", 0, 0},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
		} else if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.StrRandMin != tc.randMin || component.StrRandMax != tc.randMax || !component.AttachedEntity {
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
		{"EF_NPC_CHILL", NPCChill, "chill", "", ""},
		{"EF_AB_OFFERTORIUM_RING", OffertoriumRing, "ab_offertorium_ring", "", ""},
		{"EF_HAMMER_OF_GOD", HammerOfGod, "stormgust", "", "effect\\RL_HAMMER_OF_GOD.wav"},
		{"EF_ACH_COMPLETE", AchComplete, "ach_complete/ppring3", "ach_complete/", ""},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if tc.wav == "" {
			if len(spec.Sfx) != 0 {
				t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
			}
		} else if len(spec.Sfx) != 1 || spec.Sfx[0] != tc.wav {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, tc.wav)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.TexturePath != tc.texturePath || !component.AttachedEntity {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}

func TestRobrowserEffectsPostElevenHundredMatchTableRows(t *testing.T) {
	body, ok := Lookup(BodyColor)
	if !ok || body.Duration != 300*time.Millisecond || len(body.Components) != 1 {
		t.Fatalf("EffectBodyColor spec = %+v ok=%t, want 300ms FUNC component", body, ok)
	}
	bodyComponent := body.Components[0]
	if bodyComponent.Kind != ComponentFUNC || bodyComponent.FuncName != "EffectBodyColor" || bodyComponent.FuncAdapter != EffectFuncBodyColor || !bodyComponent.AttachedEntity {
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
		{"EF_BAKURETSU_HADOU", BakuretsuHadou, "bakuretsu_hadou/bakuretsu_hadou", "bakuretsu_hadou/", true, -50, false},
		{"EF_DIGITAL_SPACE", DigitalSpace, "digital_space/digital_space", "digital_space/", false, 0, true},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || spec.Duration != 5*time.Minute || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want 5m SPR component", tc.name, spec, ok)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSPR || component.SpriteFile != tc.file || component.TexturePath != tc.texturePath || !component.AttachedEntity {
			t.Fatalf("%s component = %+v, want SPR %q texturePath %q attached", tc.name, component, tc.file, tc.texturePath)
		}
		if !component.Repeat || !component.SpriteRepeat || component.SpriteHead != tc.head || component.SpriteYOffset != tc.yOffset || component.RenderBefore != tc.renderBefore {
			t.Fatalf("%s component flags = %+v", tc.name, component)
		}
		if len(spec.Sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
		}
	}

	for _, tc := range []struct {
		name         string
		id           int
		colorName    string
		renderBefore bool
	}{
		{"DROPEFFECT_PINK", DropPink, "pink", true},
		{"DROPEFFECT_YELLOW", DropYellow, "yellow", false},
		{"DROPEFFECT_PURPLE", DropPurple, "purple", false},
		{"DROPEFFECT_BLUE", DropBlue, "blue", false},
		{"DROPEFFECT_GREEN", DropGreen, "green", false},
		{"DROPEFFECT_RED", DropRed, "red", false},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 2 {
			t.Fatalf("%s spec = %+v ok=%t, want two STR components", tc.name, spec, ok)
		}
		wantSFX := "effect\\drop_" + tc.colorName + ".wav"
		if len(spec.Sfx) != 1 || spec.Sfx[0] != wantSFX {
			t.Fatalf("%s sfx = %#v, want %q", tc.name, spec.Sfx, wantSFX)
		}
		wantFiles := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName,
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/dropitem_" + tc.colorName + "_bottom",
		}
		wantTexturePaths := []string{
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "/",
			"new_dropitem/dropitem_" + tc.colorName + "/dropitem_" + tc.colorName + "_bottom/",
		}
		for i, component := range spec.Components {
			if component.Kind != ComponentSTR || component.StrFile != wantFiles[i] || component.TexturePath != wantTexturePaths[i] {
				t.Fatalf("%s component %d = %+v, want STR %q texturePath %q", tc.name, i, component, wantFiles[i], wantTexturePaths[i])
			}
			if component.AttachedEntity || component.RenderBefore != tc.renderBefore {
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
		{"EF_NEW_SUCCESS", NewSuccess, "grade_enchant/new_success/new_success", "grade_enchant/new_success/"},
		{"EF_NEW_FAILURE", NewFailure, "grade_enchant/new_failed/new_failed", "grade_enchant/new_failed/"},
		{"EF_NEW_INTRO", NewIntro, "grade_enchant/new_intro/new_intro", "grade_enchant/new_intro/"},
		{"EF_UI_ENCHANT_INTRO_YELLOW", EnchantYellow, "ui_enchant/ui_intro_yellow/ui_intro_yellow", "ui_enchant/ui_intro_yellow/"},
		{"EF_UI_ENCHANT_SUCCESS", EnchantSuccess, "ui_enchant/ui_enchant_success/ui_enchant_success", "ui_enchant/ui_enchant_success/"},
		{"EF_UI_ENCHANT_FAIL", EnchantFail, "ui_enchant/ui_fail/ui_enchant_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_BLUE", EnchantBlue, "ui_enchant/ui_intro_blue/ui_intro_blue", "ui_enchant/ui_intro_blue/"},
		{"EF_UI_ENCHANT_UP_SUCCESS", EnchantUpSuccess, "ui_enchant/ui_levelup_success/ui_levelup_success", "ui_enchant/ui_levelup_success/"},
		{"EF_UI_ENCHANT_UP_FAIL", EnchantUpFail, "ui_enchant/ui_fail/ui_levelup_fail", "ui_enchant/ui_fail/"},
		{"EF_UI_ENCHANT_INTRO_GREEN", EnchantGreen, "ui_enchant/ui_intro_green/ui_intro_green", "ui_enchant/ui_intro_green/"},
		{"EF_UI_ENCHANT_RESET_SUCCESS", EnchantResetOK, "ui_enchant/ui_reset_success/ui_reset_success", "ui_enchant/ui_reset_success/"},
		{"EF_UI_ENCHANT_RESET_FAIL", EnchantResetFail, "ui_enchant/ui_fail/ui_reset_fail", "ui_enchant/ui_fail/"},
	} {
		spec, ok := Lookup(tc.id)
		if !ok || len(spec.Components) != 1 {
			t.Fatalf("%s spec = %+v ok=%t, want one STR component", tc.name, spec, ok)
		}
		if len(spec.Sfx) != 0 {
			t.Fatalf("%s sfx = %#v, want none", tc.name, spec.Sfx)
		}
		component := spec.Components[0]
		if component.Kind != ComponentSTR || component.StrFile != tc.file || component.TexturePath != tc.texturePath || component.AttachedEntity || component.RenderBefore {
			t.Fatalf("%s component = %+v", tc.name, component)
		}
	}
}
