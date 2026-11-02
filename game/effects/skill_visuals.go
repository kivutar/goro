package effects

import (
	"slices"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
)

func SpecialEffectID(effectID uint32) int {
	if mapped, ok := networkSpecialEffectIDs[effectID]; ok {
		return mapped
	}
	if _, ok := Lookup(int(effectID)); ok {
		return int(effectID)
	}
	return 0
}

var networkSpecialEffectIDs = map[uint32]int{
	network.SpecialEffectBaseLevelUp:            BaseLevelUp,
	network.SpecialEffectJobLevelUp:             JobLevelUp,
	network.SpecialEffectRefineFailure:          RefineFail,
	network.SpecialEffectRefineSuccess:          RefineOK,
	network.SpecialEffectGameOver:               0,
	network.SpecialEffectPharmacySuccess:        PharmacyOK,
	network.SpecialEffectPharmacyFailure:        PharmacyFail,
	network.SpecialEffectSuperNoviceBaseLevelUp: Angel2,
	network.SpecialEffectSuperNoviceJobLevelUp:  JobLvUp50,
	network.SpecialEffectTaekwonBaseLevelUp:     BaseLevelUp,
}

func CraftingResultSpecialEffect(effectID uint32) bool {
	switch effectID {
	case network.SpecialEffectRefineFailure,
		network.SpecialEffectRefineSuccess,
		network.SpecialEffectPharmacySuccess,
		network.SpecialEffectPharmacyFailure:
		return true
	default:
		return false
	}
}

func craftingResultEffectGroup(effectID int) []int {
	switch effectID {
	case RefineOK, RefineFail:
		return []int{RefineOK, RefineFail}
	case PharmacyOK, PharmacyFail:
		return []int{PharmacyOK, PharmacyFail}
	default:
		return []int{effectID}
	}
}

func skillCastGroundSampleSize(skillID uint16) float64 {
	if size, ok := db.SkillGroundCastClientScopeSize(skillID); ok {
		return size
	}
	return db.SkillGroundCastSize(skillID, 0)
}

func skillCastGroundSampleRotationRadiansPerSecond(skillID uint16) float64 {
	speed, _ := db.SkillGroundCastClientRotationRadiansPerSecond(skillID)
	return speed
}

type skillEffectSpec struct {
	EffectTiming           db.SkillEffectTiming
	EffectIDs              []int
	EffectIDsOnCaster      []int
	BeforeHitEffectIDs     []int
	BeforeHitEffectIDsSelf []int
	HitEffectIDs           []int
	HitEffectIDsOnCaster   []int
	SuccessEffectIDs       []int
	SuccessEffectIDsSelf   []int
	BeginCastEffectIDs     []int
	GroundEffectIDs        []int
	HideCastBar            bool
	HideCastAura           bool
	Action                 SkillActionSpec
	ForceGroundTarget      bool
	ForceSelfTarget        bool
	Passive                bool
}

func SkillEffectSpecFor(skillID uint16) skillEffectSpec {
	return importedSkillEffectSpec(skillID)
}

func importedSkillEffectSpec(skillID uint16) skillEffectSpec {
	out := skillEffectSpec{}
	if spec, ok := db.SkillEffects[skillID]; ok {
		out.EffectTiming = spec.EffectTiming
		out.EffectIDs = copyIntSlice(spec.EffectIDs)
		out.EffectIDsOnCaster = copyIntSlice(spec.EffectIDsOnCaster)
		out.BeforeHitEffectIDs = copyIntSlice(spec.BeforeHitEffectIDs)
		out.BeforeHitEffectIDsSelf = copyIntSlice(spec.BeforeHitEffectIDsSelf)
		out.HitEffectIDs = copyIntSlice(spec.HitEffectIDs)
		out.HitEffectIDsOnCaster = copyIntSlice(spec.HitEffectIDsOnCaster)
		out.SuccessEffectIDs = copyIntSlice(spec.SuccessEffectIDs)
		out.SuccessEffectIDsSelf = copyIntSlice(spec.SuccessEffectIDsSelf)
		out.BeginCastEffectIDs = copyIntSlice(spec.BeginCastEffectIDs)
		out.GroundEffectIDs = copyIntSlice(spec.GroundEffectIDs)
		out.HideCastBar = spec.HideCastBar
		out.HideCastAura = spec.HideCastAura
	}
	if action, ok := importedSkillActionSpec(skillID); ok {
		out.Action = action
	}
	return out
}

func copyIntSlice(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	return append([]int(nil), in...)
}

func SkillSuccessEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).SuccessEffectIDs
}

func SkillSuccessEffectSelfIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).SuccessEffectIDsSelf
}

func SkillEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).EffectIDs
}

func SkillEffectOnCasterIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).EffectIDsOnCaster
}

func SkillBeginEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).BeginCastEffectIDs
}

func SkillBeforeHitEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).BeforeHitEffectIDs
}

func SkillBeforeHitEffectSelfIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).BeforeHitEffectIDsSelf
}

func SkillGroundEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).GroundEffectIDs
}

func SkillHidesCastBar(skillID uint16) bool {
	return SkillEffectSpecFor(skillID).HideCastBar
}

func skillHidesCastAura(skillID uint16) bool {
	return SkillEffectSpecFor(skillID).HideCastAura
}

func SkillForcesGroundTarget(skillID uint16) bool {
	return SkillEffectSpecFor(skillID).ForceGroundTarget
}

func SkillForcesSelfTarget(skillID uint16) bool {
	return SkillEffectSpecFor(skillID).ForceSelfTarget
}

func SkillForcesPassive(skillID uint16) bool {
	return SkillEffectSpecFor(skillID).Passive
}

type skillUnitEffectSpec struct {
	effectIDs []int
}

// Mostly mirrors reference client's DB/Skills/SkillUnit.js: unit id -> effect id.
var skillUnitEffectSpecs = map[uint16]skillUnitEffectSpec{
	126: {effectIDs: []int{SafetyWall}},             // UNT_SAFETYWALL -> EF_GLASSWALL2
	127: {effectIDs: []int{FireWall}},               // UNT_FIREWALL -> EF_FIREWALL
	128: {effectIDs: []int{Portal}},                 // UNT_WARPPORTAL -> EF_PORTAL2
	129: {effectIDs: []int{ReadyPortal}},            // UNT_PRE_WARPPORTAL -> EF_READYPORTAL2
	131: {effectIDs: []int{BottomSanc}},             // UNT_SANCTUARY -> EF_BOTTOM_SANC
	132: {effectIDs: []int{BottomMagnus}},           // UNT_MAGNUS -> EF_BOTTOM_MAG
	133: {effectIDs: []int{Pneuma}},                 // UNT_PNEUMA -> EF_PNEUMA
	135: {effectIDs: []int{FirePillarOn}},           // UNT_FIREPILLAR_WAITING -> EF_FIREPILLARON
	136: {effectIDs: []int{FirePillarBomb}},         // UNT_FIREPILLAR_ACTIVE -> EF_FIREPILLARBOMB
	141: {effectIDs: []int{IceWall}},                // UNT_ICEWALL -> EF_ICEWALL
	142: {effectIDs: []int{Quagmire}},               // UNT_QUAGMIRE -> EF_QUAGMIRE
	146: {effectIDs: []int{VenomDust2}},             // UNT_VENOMDUST -> EF_VENOMDUST2
	154: {effectIDs: []int{BottomVolcano}},          // UNT_VOLCANO -> EF_BOTTOM_VO
	155: {effectIDs: []int{BottomDeluge}},           // UNT_DELUGE -> EF_BOTTOM_DE
	156: {effectIDs: []int{BottomViolent}},          // UNT_VIOLENTGALE -> EF_BOTTOM_VI
	157: {effectIDs: []int{BottomLand}},             // UNT_LANDPROTECTOR -> EF_BOTTOM_LA
	158: {effectIDs: []int{BottomLullabyGround}},    // UNT_LULLABY -> 278_ground
	159: {effectIDs: []int{BottomRichKimGround}},    // UNT_RICHMANKIM -> 279_ground
	160: {effectIDs: []int{BottomChaosGround}},      // UNT_ETERNALCHAOS -> 280_ground
	161: {effectIDs: []int{BottomDrumGround}},       // UNT_DRUMBATTLEFIELD -> 281_ground
	162: {effectIDs: []int{BottomNibelungGround}},   // UNT_RINGNIBELUNGEN -> 282_ground
	163: {effectIDs: []int{BottomRokiGround}},       // UNT_ROKISWEIL -> 283_ground
	164: {effectIDs: []int{BottomAbyssGround}},      // UNT_INTOABYSS -> 284_ground
	165: {effectIDs: []int{BottomSiegGround}},       // UNT_SIEGFRIED -> 285_ground
	166: {effectIDs: []int{BottomDissonanceGround}}, // UNT_DISSONANCE -> 277_ground
	167: {effectIDs: []int{BottomWhistleGround}},    // UNT_WHISTLE -> 286_ground
	168: {effectIDs: []int{BottomSinXGround}},       // UNT_ASSASSINCROSS -> 287_ground
	169: {effectIDs: []int{BottomBragiGround}},      // UNT_POEMBRAGI -> 288_ground
	170: {effectIDs: []int{BottomAppleGround}},      // UNT_APPLEIDUN -> 289_ground
	171: {effectIDs: []int{BottomUglyDanceGround}},  // UNT_UGLYDANCE -> 290_ground
	172: {effectIDs: []int{BottomHummingGround}},    // UNT_HUMMING -> 291_ground
	173: {effectIDs: []int{BottomForgetGround}},     // UNT_DONTFORGETME -> 292_ground
	174: {effectIDs: []int{BottomFortuneGround}},    // UNT_FORTUNEKISS -> 293_ground
	175: {effectIDs: []int{BottomServiceGround}},    // UNT_SERVICEFORYOU -> 294_ground
	177: {effectIDs: []int{Demonstration}},          // UNT_DEMONSTRATION -> EF_DEMONSTRATION
	179: {effectIDs: []int{GospelGround}},           // UNT_GOSPEL -> 370_ground
	180: {effectIDs: []int{BottomBasilica}},         // UNT_BASILICA -> EF_BOTTOM_BASILICA
	181: {},                                         // UNT_MOONLIT: the caster owns the sphere.
	182: {effectIDs: []int{FogWallGround}},          // UNT_FOGWALL -> EF_BOTTOM_FOGWALL
	183: {effectIDs: []int{BottomSpider}},           // UNT_SPIDERWEB -> EF_BOTTOM_SPIDER
	184: {effectIDs: []int{Gravitation}},            // UNT_GRAVITATION -> EF_GRAVITATION
	185: {effectIDs: []int{BottomHermode}},          // UNT_HERMODE -> EF_BOTTOM_HERMODE
	187: {effectIDs: []int{BottomSuiton}},           // UNT_SUITON
	188: {effectIDs: []int{Tatami}},                 // UNT_TATAMIGAESHI -> EF_TATAMI
	189: {effectIDs: []int{Kaen}},                   // UNT_KAEN -> EF_KAEN
	190: {effectIDs: []int{LightningS}},             // UNT_GROUNDDRIFT_LIGHTNING
	191: {effectIDs: []int{BlindS}},                 // UNT_GROUNDDRIFT_BLIND
	192: {effectIDs: []int{PoisonS}},                // UNT_GROUNDDRIFT_POISON
	193: {effectIDs: []int{FreezingS}},              // UNT_GROUNDDRIFT_FREEZING
	194: {effectIDs: []int{FlareS}},                 // UNT_GROUNDDRIFT_FLARE
	199: {effectIDs: []int{BottomEvilLand}},         // UNT_EVILLAND -> EF_BOTTOM_EVILLAND
	202: {effectIDs: []int{GlassWall3}},             // UNT_EPICLESIS -> EF_GLASSWALL3
	203: {effectIDs: []int{EarthWall}},              // UNT_EARTHSTRAIN -> EF_EARTHWALL
	204: {effectIDs: []int{BottomManhole}},          // UNT_MANHOLE -> EF_BOTTOM_MANHOLE
	205: {effectIDs: []int{ForestLight6}},           // UNT_DIMENSIONDOOR -> EF_FORESTLIGHT6
	206: {effectIDs: []int{BottomAni}},              // UNT_CHAOSPANIC -> EF_BOTTOM_ANI
	207: {effectIDs: []int{BottomMaelstrom}},        // UNT_MAELSTROM -> EF_BOTTOM_MAELSTROM
	208: {effectIDs: []int{BottomBloodyLust}},       // UNT_BLOODYLUST -> EF_BOTTOM_BLOODYLUST
	218: {effectIDs: []int{BotReverb}},              // UNT_REVERBERATION -> EF_BOT_REVERB
	220: {effectIDs: []int{FireWall2}},              // UNT_FIREWALK -> EF_FIREWALL2
	221: {effectIDs: []int{Shockwave2}},             // UNT_ELECTRICWALK -> EF_SHOCKWAVE2
	222: {effectIDs: []int{BotReverb2}},             // UNT_NETHERWORLD -> EF_BOT_REVERB2
}

func SkillUnitEffectIDs(unitID uint16) []int {
	return skillUnitEffectSpecs[unitID].effectIDs
}

func skillCastAuraEffectID(property uint32) int {
	switch property {
	case 1:
		return BeginSpell2
	case 2:
		return BeginSpell4
	case 3:
		return BeginSpell3
	case 4:
		return BeginSpell5
	case 5:
		return BeginSpell7
	case 6:
		return BeginSpell6
	default:
		return BeginSpell
	}
}

func SkillHitEffectIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).HitEffectIDs
}

func SkillHitEffectOnCasterIDs(skillID uint16) []int {
	return SkillEffectSpecFor(skillID).HitEffectIDsOnCaster
}

func IsBardPerformance(skillID uint16) bool {
	return skillID >= db.SkillBDLullaby && skillID <= db.SkillBDSiegfried ||
		skillID == db.SkillBaDissonance || skillID >= db.SkillBaWhistle && skillID <= db.SkillBaAppleidun
}

func IsDancerPerformance(skillID uint16) bool {
	return skillID == db.SkillDCUglydance || skillID >= db.SkillDCHumming && skillID <= db.SkillDCServiceforyou
}

func NinjaCastingSkill(id uint16) bool {
	return id == db.SkillNJHuuma || id == db.SkillNJBunsinjyutsu || id >= db.SkillNJKouenka && id <= db.SkillNJNen
}

func (m *System) ReplaceCraftingResultEffect(ctx client.Context, notify network.SpecialEffectNotify, effectID int, spec Spec) {
	for _, activeID := range craftingResultEffectGroup(effectID) {
		m.Remove(activeID, notify.AID)
	}
	if m.Add(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v restart=true", notify.AID, notify.EffectID, effectID, spec.Sfx)
	}
}

func (m *System) RemoveSkillCastEffects(ctx client.Context, sourceID uint32) {
	if sourceID == 0 {
		return
	}
	local := scene.IsLocalActor(ctx, sourceID)
	m.Active = slices.DeleteFunc(m.Active, func(effect Instance) bool {
		return effect.castSourceID == sourceID || (local && scene.IsLocalActor(ctx, effect.castSourceID))
	})
}

func (m *System) AddSkillCastEffects(ctx client.Context, skillID uint16, property uint32, sourceID, targetID uint32, cellX, cellY int, duration time.Duration, starts time.Time, source string) {
	if duration <= 0 || sourceID == 0 {
		return
	}
	m.RemoveSkillCastEffects(ctx, sourceID)
	first := len(m.Active)
	defer func() {
		// Target and ground markers belong to the caster too, not their anchor.
		for i := first; i < len(m.Active); i++ {
			m.Active[i].castSourceID = sourceID
		}
	}()
	if targetID == 0 && (cellX != 0 || cellY != 0) {
		markerSize := skillCastGroundSampleSize(skillID)
		markerRotationSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		if m.addCellRotation(ctx, GroundSample, 0, cellX, cellY, starts, duration, markerSize, markerRotationSpeed) {
			glog.Debugf("skill cast ground marker source=%s skill=%d src=%d cell=%d,%d delay_ms=%d", source, skillID, sourceID, cellX, cellY, duration.Milliseconds())
		}
	}
	if m.AddBetweenDuration(ctx, CastRing, sourceID, 0, starts, duration) {
		glog.Debugf("skill cast circle source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
	}
	if targetID != 0 && targetID != sourceID {
		if m.AddBetweenDuration(ctx, LockOnTarget, targetID, 0, starts, duration) {
			glog.Debugf("skill cast target lockon source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
		}
	}
	if skillID == db.SkillACChargearrow || skillID == db.SkillSNFalconassault || skillID == db.SkillBaMusicalstrike || skillID == db.SkillDCThrowarrow ||
		(skillID >= db.SkillBSIron && skillID <= db.SkillBSMaximize && skillID != db.SkillBSRepairweapon) || skillID == db.SkillBSAdrenaline2 ||
		skillID == db.SkillWSMeltdown || skillID == db.SkillWSCartboost || skillID == db.SkillWSOverthrustmax ||
		skillID == db.SkillWSCarttermination || skillID == db.SkillWSWeaponrefine ||
		skillID == db.SkillAMDemonstration || skillID == db.SkillAMSpheremine || skillID == db.SkillAMPharmacy || skillID == db.SkillCRCultivation || skillID == db.SkillHWMagicpower || skillID == db.SkillPFFogwall {
		m.AddAt(ctx, BashBegin, sourceID, starts)
		return
	}
	if skillHidesCastAura(skillID) {
		return
	}
	if skillID == db.SkillMOExtremityfist {
		effectID := BeginAsura
		if actor, ok, _ := scene.ActorForCombatID(ctx, sourceID); ok && actor.Job == db.JobMonkH {
			effectID = BeginAsura11
		}
		// The glyph sequence has its own timing, independent of cast speed.
		m.AddAt(ctx, effectID, sourceID, starts)
		return
	}
	effectID := skillCastAuraEffectID(property)
	if skillID == db.SkillLKSpiralpierce {
		effectID = PierceBody
	}
	if skillID == db.SkillKNChargeatk || skillID == db.SkillPaPressure || skillID == db.SkillPaShieldchain ||
		skillID == db.SkillHPAssumptio || skillID == db.SkillHPBasilica || skillID == db.SkillPRRedemptio ||
		skillID == db.SkillSNSharpshooting || skillID == db.SkillSNWindwalk || skillID == db.SkillBaPangvoice ||
		skillID == db.SkillCGArrowvulcan || skillID == db.SkillCGTarotcard || skillID == db.SkillHWGanbantein {
		effectID = BeginSpell6
	}
	switch skillID {
	case db.SkillTKRun, db.SkillTKSevenwind, db.SkillTKMission, db.SkillSGFeel, db.SkillSGHate, db.SkillSGFusion, db.SkillNJHuuma, db.SkillNJBunsinjyutsu, db.SkillNJNen:
		effectID = BlueCasting
	case db.SkillACConcentration:
		effectID = IncAgiDex
	case db.SkillSTChasewalk:
		m.AddAt(ctx, CastSpin, sourceID, starts)
		return
	case db.SkillSTPreserve:
		effectID = BeginSpell7
	case db.SkillSTRejectsword, db.SkillSTFullstrip:
		effectID = BashBegin
	case db.SkillASCBreaker, db.SkillASCMeteorassault:
		effectID = BeginSpell7
	case db.SkillASCEdp, db.SkillASCCdp:
		effectID = BashBegin
	case db.SkillAMCallhomun, db.SkillAMRest, db.SkillAMResurrecthomun:
		effectID = CoupleCasting
	case db.SkillAMTwilight1, db.SkillAMTwilight2, db.SkillAMTwilight3:
		effectID = Twilight1 + int(skillID-db.SkillAMTwilight1)
	case db.SkillSAElementwater:
		effectID = BeginSpell2
	case db.SkillSAElementfire:
		effectID = BeginSpell3
	case db.SkillSAElementground:
		effectID = BeginSpell4
	case db.SkillSAElementwind:
		effectID = BeginSpell5
	}
	if _, ok := GunslingerCastAction(skillID, starts); ok {
		effectID = BlueCasting
	}
	if db.IsSoulLinkerActiveSkill(skillID) {
		effectID = BlueCasting
	}
	if effectID <= 0 {
		return
	}
	if m.AddBetweenDuration(ctx, effectID, sourceID, targetID, starts, duration) {
		glog.Debugf("skill cast aura source=%s skill=%d src=%d target=%d property=%d effect=%d delay_ms=%d", source, skillID, sourceID, targetID, property, effectID, duration.Milliseconds())
	}
}

func (m *System) ApplyAlchemistSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	effectID, level := ThrowItem2, int(notify.Amount)
	switch notify.SkillID {
	case db.SkillAMPotionpitcher:
	case db.SkillAMBerserkpitcher:
		effectID, level = ThrowItem5, 4
	default:
		return
	}
	if m.AddBetween(ctx, effectID, notify.TargetID, notify.SourceID, now) {
		m.Active[len(m.Active)-1].ItemLevel = level
	}
}

func (m *System) AddGunslingerBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillGSGlittering || action.SkillID > db.SkillGSGrounddrift {
		return false
	}
	for _, id := range SkillBeforeHitEffectIDs(action.SkillID) {
		m.AddBetween(ctx, id, action.TargetID, action.SourceID, starts)
	}
	return true
}

func (m *System) AddMageProjectileEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	count := scene.ActionVisualHitCount(action)
	frames := 52
	switch action.SkillID {
	case db.SkillMGFireball, db.SkillMGFrostdiver:
		count = 1
	case db.SkillMGSoulstrike:
		count = min(count, 5)
		frames = 6 + (count-1)*11 + 40
	case db.SkillMGColdbolt, db.SkillMGFirebolt:
		frames = 12 + (count-1)*10 + 70
		if action.SkillID == db.SkillMGFirebolt {
			frames = max(frames, 160)
		}
	default:
		return false
	}
	id := SkillBeforeHitEffectIDs(action.SkillID)[0]
	if m.AddBetweenDuration(ctx, id, action.TargetID, action.SourceID, starts, time.Duration(frames)*db.EffectFrameDuration) {
		effect := &m.Active[len(m.Active)-1]
		effect.HitCount = count
		if action.SkillID == db.SkillMGSoulstrike {
			for i := 1; i < count; i++ {
				m.scheduleSoundAtWorldEffect(starts.Add(time.Duration(6+11*i)*db.EffectFrameDuration), *effect, "effect\\ef_soulstrike.wav")
			}
		}
	}
	return true
}

func (m *System) AddNinjaBeforeHitEffect(ctx client.Context, action network.ActorActionNotify, starts time.Time) bool {
	if action.SkillID < db.SkillNJSyuriken || action.SkillID > db.SkillNJIssen {
		return false
	}
	for _, id := range SkillBeforeHitEffectIDs(action.SkillID) {
		if !m.AddBetween(ctx, id, action.TargetID, action.SourceID, starts) {
			continue
		}
		effect := &m.Active[len(m.Active)-1]
		if id >= ThrowItem7 && id <= ThrowItem10 {
			m.scheduleSound(starts, "effect\\닌자_던지기.wav")
		}
		if id == Stin4 {
			effect.HitCount = scene.ActionVisualHitCount(action)
			if effect.HitCount > 36 {
				effect.HitCount = 1
			}
		}
	}
	return true
}

func (m *System) ApplyProfessorSkill(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	switch notify.SkillID {
	case db.SkillPFSoulchange:
		m.AddBetween(ctx, LineLink2, notify.SourceID, notify.TargetID, now)
	case db.SkillPFDoublecasting:
		if _, ok, _ := scene.ActorForCombatID(ctx, notify.TargetID); ok {
			m.scheduleSound(now, "effect\\마법력 증폭.wav")
		}
	}
}

func (m *System) AddSkillUnitEffects(ctx client.Context, entry network.SkillUnitEntry) {
	if !entry.Visible {
		return
	}
	effectIDs := SkillUnitEffectIDs(entry.UnitID)
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.AddCellLifetime(ctx, effectID, entry.ID, int(entry.X), int(entry.Y), now, SkillUnitEffectFallbackDuration, true) {
			glog.Debugf("skill unit effect unit=%d id=%d creator=%d cell=%d,%d effect=%d", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, effectID)
		}
	}
}
