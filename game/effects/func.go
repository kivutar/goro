package effects

import (
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/render"
)

type EffectFuncAdapter int

const (
	EffectFuncUnknown EffectFuncAdapter = iota
	EffectFuncGroundSample
	EffectFuncCastRing
	EffectFuncLockOnTarget
	EffectFuncLevel99Aura
	EffectFuncLevel99Ground
	EffectFuncLevel99Bubble
	effectFuncPropertyGround
	EffectFuncGroundSigil
	EffectFuncSpiritSphere
	EffectFuncGumgang
	EffectFuncFlatColorTile
	EffectFuncGroundTexture
	EffectFuncBodyColor
	EffectFuncMapPillar
	EffectFuncHitRing
	effectFuncHitParticles
	effectFuncBlessingCircle
	effectFuncEstin
	effectFuncRisingBuffParticles
	effectFuncHitLineBurst
	effectFuncEsma
	effectFuncKaizel
	effectFuncEsmaHitTexture
	effectFuncSoulLink
	effectFuncKaupeDodge
	effectFuncBlueCasting
	EffectFuncSonicBlow
	EffectFuncThrowItem
	effectFuncSlimPitcher
	effectFuncAcidDemon
	effectFuncGrimtooth
	effectFuncRisingParticles
	effectFuncSteal
	effectFuncMammoniteCoins
	EffectFuncArrowShower
	effectFuncHunter
	effectFuncSniper
	effectFuncClownGypsy
	effectFuncBlacksmith
	effectFuncCartTermination
	effectFuncChemicalProtection
	effectFuncTwilight
	effectFuncBerserkPotion
	effectFuncSongMusic
	effectFuncSongVertical
	effectFuncSongLight
	effectFuncSongOut
	effectFuncDirectedSprite
	effectFuncCartRevolution
	effectFuncSprinkleSand
	effectFuncRogueBurst
	effectFuncImpactParticles
	effectFuncBash
	effectFuncBashHit
	effectFuncEndure
	effectFuncMagnumBreak
	effectFuncBowlingBash
	effectFuncSpearBoomerang
	effectFuncShieldBoomerang
	effectFuncShieldChain
	effectFuncPressure
	effectFuncBashRays
	effectFuncGrandCross
	effectFuncCrusaderDefense
	effectFuncProvidence
	effectFuncDevotionLink
	effectFuncGuard
	effectFuncEquipmentBurst
	EffectFuncGospelCross
	effectFuncSpiralPierce
	EffectFuncAgility
	effectFuncMonkRays
	effectFuncTripleAttack
	effectFuncChainCrush
	effectFuncGlyphs
	effectFuncTaekwonStorm
	effectFuncStarRings
	effectFuncStarBlind
	effectFuncTaekwonStop
	effectFuncChemicalDash
	effectFuncTaekwonJump
	effectFuncTaekwonFoot
	effectFuncHitLine
	effectFuncTurnUndead
	effectFuncRevive
	effectFuncGroundWalls
	effectFuncMageProjectile
	effectFuncFrostDiverHit
	EffectFuncMageBolt
	effectFuncSight
	effectFuncGroundSpikes
	effectFuncWaterBall
	effectFuncJupitel
	effectFuncSightTrasher
	effectFuncFirePillar
	effectFuncGroundCast
	effectFuncAbsorbSpirits
	effectFuncSoulDrain
	effectFuncGravitation
	effectFuncHPConversion
	effectFuncSoulChangeLink
	effectFuncSoulBreaker
	effectFuncNinjaWind
	effectFuncNinjaPetals
	effectFuncNativeSprite
)

func (m *System) DrawFuncEffect(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, componentIndex int, worldX, worldY, worldZ, progress float64, now time.Time) {
	switch component.FuncAdapter {
	case effectFuncHPConversion:
		m.drawHPConversion(screen, ctx, projection, component, effect, now)
	case effectFuncSoulBreaker:
		m.drawSoulBreaker(screen, ctx, component, effect, now)
	case effectFuncSoulChangeLink:
		m.drawSoulChangeLink(screen, ctx, component, effect, now)
	case effectFuncSoulDrain:
		m.drawSoulDrain(screen, ctx, projection, component, effect, now)
	case effectFuncGravitation:
		m.drawGravitation(screen, ctx, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncGroundSpikes:
		m.drawGroundSpikes(screen, ctx, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncWaterBall:
		m.drawWizardWaterBall(screen, ctx, projection, component, effect, now)
	case effectFuncJupitel:
		m.drawJupitel(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncSightTrasher:
		m.drawSightTrasher(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncFirePillar:
		m.drawFirePillar(screen, ctx, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncMageProjectile:
		m.drawMageProjectile(screen, ctx, projection, component, effect, now)
	case effectFuncFrostDiverHit:
		m.DrawFrostDiverSpikes(screen, ctx, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case EffectFuncMageBolt:
		m.drawMageBolt(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncSight:
		m.drawSightEffect(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncTurnUndead:
		m.drawTurnUndeadEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncRevive:
		m.drawReviveEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundWalls:
		m.drawGroundWallsEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncMonkRays:
		m.drawMonkRays(screen, ctx, component, effect, now)
	case effectFuncSongMusic:
		m.drawSongMusic(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncSongVertical, effectFuncSongLight, effectFuncSongOut:
		m.drawSongGround(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case effectFuncDirectedSprite:
		m.drawDirectedSpriteEffect(screen, ctx, projection, component, effect, now)
	case effectFuncClownGypsy:
		m.drawClownGypsyEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBlacksmith:
		m.drawBlacksmithEffect(screen, ctx, projection, component, effect, now)
	case effectFuncChemicalProtection:
		m.drawChemicalProtection(screen, ctx, projection, component, effect, now)
	case effectFuncBerserkPotion:
		m.DrawBerserkPotion(screen, ctx, component, effect, now)
	case effectFuncTwilight:
		m.drawTwilight(screen, ctx, projection, component, effect, now)
	case effectFuncCartTermination:
		m.drawCartTermination(screen, ctx, projection, component, effect, now)
	case effectFuncSniper:
		m.drawSniperEffect(screen, ctx, component, effect, now)
	case effectFuncGlyphs:
		m.DrawEffectGlyphs(screen, ctx, projection, component, effect, now)
	case effectFuncStarRings:
		m.drawStarRings(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncStarBlind:
		m.drawStarBlind(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncTaekwonStorm:
		m.drawTaekwonStorm(screen, ctx, component, effect, now)
	case effectFuncTaekwonStop:
		m.drawTaekwonStop(screen, ctx, component, effect, now)
	case effectFuncChemicalDash:
		m.drawChemicalDash(screen, ctx, component, effect, now)
	case effectFuncTaekwonFoot:
		m.drawTaekwonFoot(screen, ctx, component, effect, now)
	case effectFuncTaekwonJump:
		m.drawTaekwonJump(screen, ctx, projection, component, effect, now)
	case effectFuncChainCrush:
		m.drawChainCrush(screen, ctx, component, effect, now)
	case effectFuncTripleAttack:
		m.drawTripleAttack(screen, ctx, component, effect, now)
	case effectFuncHitLine:
		m.drawHitLine(screen, ctx, component, effect, now)
	case EffectFuncAgility:
		m.DrawAgilityEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSpiralPierce:
		m.drawSpiralPierceEffect(screen, ctx, component, effect, now)
	case effectFuncEquipmentBurst:
		m.drawEquipmentBurst(screen, ctx, projection, component, effect, now)
	case effectFuncGuard:
		m.drawGuardEffect(screen, ctx, projection, component, effect, now)
	case EffectFuncGospelCross:
		m.drawGospelCrossEffect(screen, ctx, projection, component, effect, now)
	case effectFuncShieldBoomerang:
		m.drawShieldBoomerangEffect(screen, ctx, projection, component, effect, now)
	case effectFuncShieldChain:
		m.drawShieldChainEffect(screen, ctx, projection, component, effect, now)
	case effectFuncPressure:
		m.drawPressureEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBashRays:
		m.drawBashRays(screen, ctx, projection, component, effect, now)
	case effectFuncGrandCross:
		m.drawGrandCrossEffect(screen, ctx, projection, component, effect, now)
	case effectFuncCrusaderDefense:
		m.drawCrusaderDefenseEffect(screen, ctx, projection, component, effect, now)
	case effectFuncProvidence:
		m.drawProvidenceEffect(screen, ctx, projection, component, effect, now)
	case effectFuncDevotionLink:
		m.drawDevotionLinkEffect(screen, ctx, projection, component, effect, now)

	case effectFuncBowlingBash:
		m.drawBowlingBashEffect(screen, ctx, component, effect, now)
	case effectFuncSpearBoomerang:
		m.drawSpearBoomerangEffect(screen, ctx, projection, component, effect, now)
	case effectFuncBashHit:
		m.drawBashHitEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncBash:
		m.drawBashEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEndure:
		m.drawEndureEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncMagnumBreak:
		m.drawMagnumBreakEffect(screen, ctx, component, effect, now)
	case effectFuncHunter:
		m.drawHunterEffect(screen, ctx, projection, component, effect, scene.Point3{X: worldX, Y: worldZ, Z: worldY}, now)
	case EffectFuncArrowShower:
		m.DrawArrowShower(screen, ctx, projection, component, effect, now)
	case effectFuncMammoniteCoins:
		m.DrawMammoniteCoins(screen, ctx, projection, component, effect, now)
	case effectFuncCartRevolution:
		m.DrawCartRevolutionEffect(screen, ctx, component, effect, now)
	case effectFuncRogueBurst:
		m.drawRogueBurst(screen, ctx, component, effect, now)
	case effectFuncImpactParticles:
		m.drawImpactParticles(screen, ctx, projection, component, effect, now)
	case effectFuncSlimPitcher:
		m.drawSlimPitcher(screen, ctx, projection, component, effect, now)
	case effectFuncAcidDemon:
		m.drawAcidDemon(screen, ctx, component, effect, now)
	case EffectFuncThrowItem:
		m.drawThrowItemEffect(screen, ctx, projection, component, effect, now)
	case EffectFuncSonicBlow:
		m.drawSonicBlowEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGrimtooth:
		m.drawGrimtoothEffect(screen, ctx, component, effect, now)
	case effectFuncRisingParticles:
		m.drawRisingParticlesEffect(screen, ctx, projection, component, effect, now)
	case effectFuncSteal:
		m.drawStealEffect(screen, ctx, projection, component, effect, now)
	case effectFuncSprinkleSand:
		m.drawSprinkleSandEffect(screen, ctx, projection, component, effect, now)
	case EffectFuncGroundSample:
		m.drawGroundPlaneEffect(screen, ctx, component, effect, worldX, worldY, progress, now)
	case EffectFuncCastRing:
		m.drawCastRingEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress)
	case EffectFuncLockOnTarget:
		m.drawLockOnTargetEffect(screen, ctx, component, effect, worldX, worldY, worldZ, progress, now)
	case EffectFuncLevel99Aura:
		m.drawLevel99AuraEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncLevel99Ground:
		m.drawLevel99GroundEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncLevel99Bubble:
		m.drawLevel99BubbleEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncPropertyGround:
		m.drawPropertyGroundEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncAbsorbSpirits:
		m.drawAbsorbSpiritsEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncGroundCast:
		m.drawGroundCastEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncGroundSigil:
		m.drawGroundSigilEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncSpiritSphere:
		m.drawSpiritSphereEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncGumgang:
		m.drawGumgangEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case EffectFuncFlatColorTile:
		m.drawFlatColorTileEffect(screen, component, worldX, worldY, worldZ)
	case EffectFuncGroundTexture:
		m.drawGroundTextureEffect(screen, ctx, component, effect, componentIndex, worldX, worldY, worldZ, progress, now)
	case EffectFuncMapPillar:
		m.drawMapPillarEffect(screen, ctx, component, worldX, worldY, worldZ, progress)
	case EffectFuncHitRing:
		m.drawHitRingEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncHitParticles:
		m.drawHitParticlesEffect(screen, ctx, projection, component, effect, componentIndex, worldX, worldY, worldZ, now)
	case effectFuncBlessingCircle:
		m.drawBlessingCircleEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncSoulLink:
		m.drawSoulLinkEffect(screen, ctx, projection, component, effect, now)
	case effectFuncKaupeDodge:
		m.drawSoulLights(screen, ctx, projection, component, effect, now)
	case effectFuncBlueCasting:
		m.DrawBlueCastingEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEstin:
		m.drawEstinEffect(screen, ctx, component, effect, now)
	case effectFuncRisingBuffParticles:
		m.DrawRisingBuffParticles(screen, ctx, projection, component, effect, now)
	case effectFuncNinjaWind:
		m.drawNinjaWind(screen, ctx, component, effect, now)
	case effectFuncNinjaPetals:
		m.drawNinjaPetals(screen, ctx, projection, component, effect, now)
	case effectFuncNativeSprite:
		m.drawNativeSpriteEffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEsma:
		m.drawEsmaEffect(screen, ctx, component, effect, now)
	case effectFuncKaizel:
		m.DrawKaizelEffect(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	case effectFuncEsmaHitTexture:
		m.drawEsmaHitTexture(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectFuncHitLineBurst:
		m.DrawHitLineBurst(screen, ctx, component, effect, worldX, worldY, worldZ, now)
	default:
	}
}

func effectFuncAdapterForName(name string) EffectFuncAdapter {
	switch name {
	case "SoulDrain":
		return effectFuncSoulDrain
	case "HPConversion":
		return effectFuncHPConversion
	case "SoulChangeLink":
		return effectFuncSoulChangeLink
	case "SoulBreaker":
		return effectFuncSoulBreaker
	case "Gravitation":
		return effectFuncGravitation
	case "ChemicalProtection":
		return effectFuncChemicalProtection
	case "BerserkPotion":
		return effectFuncBerserkPotion
	case "Twilight":
		return effectFuncTwilight
	case "CartTermination":
		return effectFuncCartTermination
	case "HasteUp", "Overthrust":
		return effectFuncBlacksmith
	case "AbsorbSpirits":
		return effectFuncAbsorbSpirits
	case "GroundCast":
		return effectFuncGroundCast
	case "WizardSpikes":
		return effectFuncGroundSpikes
	case "WaterBall":
		return effectFuncWaterBall
	case "Jupitel":
		return effectFuncJupitel
	case "SightTrasher":
		return effectFuncSightTrasher
	case "FirePillar":
		return effectFuncFirePillar
	case "MageProjectile":
		return effectFuncMageProjectile
	case "FrostDiverHit":
		return effectFuncFrostDiverHit
	case "MageBolt":
		return EffectFuncMageBolt
	case "Sight":
		return effectFuncSight
	case "TurnUndead":
		return effectFuncTurnUndead
	case "Revive":
		return effectFuncRevive
	case "PriestGround":
		return effectFuncGroundWalls
	case "Agility":
		return EffectFuncAgility
	case "FullStrip":
		return effectFuncEquipmentBurst
	case "Guard":
		return effectFuncGuard
	case "GospelCross":
		return EffectFuncGospelCross
	case "DevotionLink":
		return effectFuncDevotionLink
	case "Providence":
		return effectFuncProvidence
	case "CrusaderDefense":
		return effectFuncCrusaderDefense
	case "GrandCross":
		return effectFuncGrandCross
	case "Sacrifice":
		return effectFuncBashRays
	case "Pressure":
		return effectFuncPressure
	case "ShieldChain":
		return effectFuncShieldChain
	case "ShieldBoomerang":
		return effectFuncShieldBoomerang
	case "BowlingBash":
		return effectFuncBowlingBash
	case "SpearBoomerang":
		return effectFuncSpearBoomerang
	case "BashHit":
		return effectFuncBashHit
	case "Bash":
		return effectFuncBash
	case "Endure":
		return effectFuncEndure
	case "MagnumBreak":
		return effectFuncMagnumBreak
	case "MagicTarget":
		return EffectFuncGroundSample
	case "CastRing":
		return EffectFuncCastRing
	case "LockOnTarget":
		return EffectFuncLockOnTarget
	case "Level99Aura":
		return EffectFuncLevel99Aura
	case "GroundAura":
		return EffectFuncLevel99Ground
	case "Level99Bubble":
		return EffectFuncLevel99Bubble
	case "NinjaWind":
		return effectFuncNinjaWind
	case "NinjaPetals":
		return effectFuncNinjaPetals
	case "NativeSprite":
		return effectFuncNativeSprite
	case "PropertyGround":
		return effectFuncPropertyGround
	case "LandProtectorGround":
		return EffectFuncGroundSigil
	case "SpiritSphere":
		return EffectFuncSpiritSphere
	case "Gumgang":
		return EffectFuncGumgang
	case "FlatColorTile":
		return EffectFuncFlatColorTile
	case "GroundTexture", "SpiderWeb":
		return EffectFuncGroundTexture
	case "EffectBodyColor", "MagicCrasherBodyColor", "TransBlueBody", "LightBlade":
		return EffectFuncBodyColor
	case "MapPillar":
		return EffectFuncMapPillar
	case "HitRing":
		return EffectFuncHitRing
	case "HitParticles":
		return effectFuncHitParticles
	case "BlessingCircle":
		return effectFuncBlessingCircle
	case "SoulLink":
		return effectFuncSoulLink
	case "KaupeDodge":
		return effectFuncKaupeDodge
	case "RogueBurst":
		return effectFuncRogueBurst
	case "ImpactParticles":
		return effectFuncImpactParticles
	case "MonkRays":
		return effectFuncMonkRays
	case "SongMusic":
		return effectFuncSongMusic
	case "SongVertical":
		return effectFuncSongVertical
	case "SongLight":
		return effectFuncSongLight
	case "SongOut":
		return effectFuncSongOut
	case "DirectedSprite":
		return effectFuncDirectedSprite
	case "ArrowVulcan", "Moonlit", "Marionette", "Hermode", "TarotCard":
		return effectFuncClownGypsy
	case "TrueSight", "WindWalk", "SharpShooting":
		return effectFuncSniper
	case "AsuraGlyphs":
		return effectFuncGlyphs
	case "StarRings":
		return effectFuncStarRings
	case "StarBlind":
		return effectFuncStarBlind
	case "TaekwonStorm":
		return effectFuncTaekwonStorm
	case "TaekwonStop":
		return effectFuncTaekwonStop
	case "TaekwonDash":
		return effectFuncChemicalDash
	case "TaekwonFoot":
		return effectFuncTaekwonFoot
	case "TaekwonJump":
		return effectFuncTaekwonJump
	case "ChainCrush":
		return effectFuncChainCrush
	case "TripleAttack":
		return effectFuncTripleAttack
	case "HitLine":
		return effectFuncHitLine
	case "Tanji":
		return effectFuncShieldBoomerang
	case "SonicBlow":
		return EffectFuncSonicBlow
	case "SlimPitcher":
		return effectFuncSlimPitcher
	case "AcidDemon":
		return effectFuncAcidDemon
	case "ThrowItem":
		return EffectFuncThrowItem
	case "Grimtooth":
		return effectFuncGrimtooth
	case "RisingParticles":
		return effectFuncRisingParticles
	case "Steal":
		return effectFuncSteal
	case "Detecting", "RemoveTrap", "Flasher", "BlitzBeat":
		return effectFuncHunter
	case "ArrowShower":
		return EffectFuncArrowShower
	case "MammoniteCoins":
		return effectFuncMammoniteCoins
	case "CartRevolution":
		return effectFuncCartRevolution
	case "SprinkleSand":
		return effectFuncSprinkleSand
	case "BlueCasting":
		return effectFuncBlueCasting
	case "SpiralPierce":
		return effectFuncSpiralPierce
	case "Estin":
		return effectFuncEstin
	case "Estun", "SoulLinkerParticles":
		return effectFuncRisingBuffParticles
	case "Esma":
		return effectFuncEsma
	case "Kaizel":
		return effectFuncKaizel
	case "EsmaHitTexture":
		return effectFuncEsmaHitTexture
	case "SoulLinkerHitLines":
		return effectFuncHitLineBurst
	default:
		return EffectFuncUnknown
	}
}
