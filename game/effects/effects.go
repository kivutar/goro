package effects

import (
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
)

// Native effect IDs.
const (
	Hit1              = db.EffectHit1
	BashHit           = db.EffectBashHit
	Hit3              = db.EffectHit3
	Hit4              = db.EffectHit4
	Hit5              = db.EffectHit5
	Hit6              = db.EffectHit6
	Entry             = db.EffectEntry
	Exit              = db.EffectExit
	Warp              = db.EffectWarp
	Enhance           = db.EffectEnhance
	Mammonite         = db.EffectMammonite
	Endure            = db.EffectEndure
	BeginSpell        = db.EffectBeginSpell
	GlassWall         = db.EffectGlassWall
	HealSP            = db.EffectHealSP
	SoulStrike        = db.EffectSoulStrike
	BashBegin         = db.EffectBashBegin
	MagnumBreak       = db.EffectMagnumBreak
	Steal             = db.EffectSteal
	PoisonAttack      = db.EffectPoisonAttack
	Detoxication      = db.EffectDetoxication
	Sight             = db.EffectSight
	StoneCurse        = db.EffectStoneCurse
	FireBall          = db.EffectFireBall
	FireWall          = db.EffectFireWall
	IceArrow          = db.EffectIceArrow
	FrostDiver        = db.EffectFrostDiver
	FrostDiverHit     = db.EffectFrostDiverHit
	LightningBolt     = db.EffectLightningBolt
	ThunderStorm      = db.EffectThunderStorm
	FireArrow         = db.EffectFireArrow
	NapalmBeat        = db.EffectNapalmBeat
	Ruwach            = db.EffectRuwach
	TeleportOld       = db.EffectTeleportOld
	ReadyPortalOld    = db.EffectReadyPortalOld
	IncAgility        = db.EffectIncAgility
	DecAgility        = db.EffectDecAgility
	Aqua              = db.EffectAqua
	Signum            = db.EffectSignum
	Angelus           = db.EffectAngelus
	Blessing          = db.EffectBlessing
	IncAgiDex         = db.EffectIncAgiDex
	Smoke             = db.EffectSmoke
	Firefly           = db.EffectFirefly
	Torch             = db.EffectTorch
	FireHit           = db.EffectFireHit
	FireSplashHit     = db.EffectFireSplashHit
	ColdHit           = db.EffectColdHit
	WindHit           = db.EffectWindHit
	PoisonHit         = db.EffectPoisonHit
	BeginSpell2       = db.EffectBeginSpell2
	BeginSpell3       = db.EffectBeginSpell3
	BeginSpell4       = db.EffectBeginSpell4
	BeginSpell5       = db.EffectBeginSpell5
	BeginSpell6       = db.EffectBeginSpell6
	BeginSpell7       = db.EffectBeginSpell7
	LockOnTarget      = db.EffectLockOnTarget
	WarpZone          = db.EffectWarpZone
	SightTrasher      = db.EffectSightTrasher
	ArrowShotRO       = db.EffectArrowShotRO
	Invenom           = db.EffectInvenom
	Cure              = db.EffectCure
	Provoke           = db.EffectProvoke
	Mvp               = db.EffectMvp
	SkidTrap          = db.EffectSkidTrap
	BrandishSpear     = db.EffectBrandishSpear
	BowlingBash       = db.EffectBowlingBash
	IceWall           = db.EffectIceWall
	Gloria            = db.EffectGloria
	Magnificat        = db.EffectMagnificat
	Resurrection      = db.EffectResurrection
	Recovery          = db.EffectRecovery
	EarthSpike        = db.EffectEarthSpike
	SpearBoomerang    = db.EffectSpearBoomerang
	Pierce            = db.EffectPierce
	TurnUndead        = db.EffectTurnUndead
	Sanctuary         = db.EffectSanctuary
	Impositio         = db.EffectImpositio
	LexAeterna        = db.EffectLexAeterna
	Aspersio          = db.EffectAspersio
	LexDivina         = db.EffectLexDivina
	Suffragium        = db.EffectSuffragium
	StormGust         = db.EffectStormGust
	LordVermilion     = db.EffectLordVermilion
	Benedictio        = db.EffectBenedictio
	MeteorStorm       = db.EffectMeteorStorm
	JupitelThunder    = db.EffectJupitelThunder
	JupitelHit        = db.EffectJupitelHit
	Quagmire          = db.EffectQuagmire
	FirePillar        = db.EffectFirePillar
	FirePillarBomb    = db.EffectFirePillarBomb
	HasteUp           = db.EffectHasteUp
	Flasher           = db.EffectFlasher
	RemoveTrap        = db.EffectRemoveTrap
	RepairWeapon      = db.EffectRepairWeapon
	CrashEarth        = db.EffectCrashEarth
	WeaponPerfect     = db.EffectWeaponPerfect
	MaximizePower     = db.EffectMaximizePower
	BlastMine         = db.EffectBlastMine
	BlastMineBomb     = db.EffectBlastMineBomb
	Claymore          = db.EffectClaymore
	FreezingTrap      = db.EffectFreezingTrap
	Bubble            = db.EffectBubble
	GasPush           = db.EffectGasPush
	SpringTrap        = db.EffectSpringTrap
	Kyrie             = db.EffectKyrie
	Magnus            = db.EffectMagnus
	BlitzBeat         = db.EffectBlitzBeat
	WaterBall         = db.EffectWaterBall
	WaterBall2        = db.EffectWaterBall2
	Detecting         = db.EffectDetecting
	Cloaking          = db.EffectCloaking
	SonicBlow         = db.EffectSonicBlow
	SonicBlowHit      = db.EffectSonicBlowHit
	Grimtooth         = db.EffectGrimtooth
	VenomDust         = db.EffectVenomDust
	EnchantPoison     = db.EffectEnchantPoison
	PoisonReact       = db.EffectPoisonReact
	PoisonReact2      = db.EffectPoisonReact2
	Overthrust        = db.EffectOverthrust
	VenomSplasher     = db.EffectVenomSplasher
	TwoHandQuicken    = db.EffectTwoHandQuicken
	AutoCounter       = db.EffectAutoCounter
	GrimtoothAtk      = db.EffectGrimtoothAtk
	Freeze            = db.EffectFreeze
	Freezed           = db.EffectFreezed
	IceCrash          = db.EffectIceCrash
	SlowPoison        = db.EffectSlowPoison
	FirePillarOn      = db.EffectFirePillarOn
	Sandman           = db.EffectSandman
	Revive            = db.EffectRevive
	Pneuma            = db.EffectPneuma
	HeavenDrive       = db.EffectHeavenDrive
	SonicBlow2        = db.EffectSonicBlow2
	BrandishSpear2    = db.EffectBrandishSpear2
	Shockwave         = db.EffectShockwave
	ShockwaveHit      = db.EffectShockwaveHit
	EarthHit          = db.EffectEarthHit
	PierceSelf        = db.EffectPierceSelf
	BowlingSelf       = db.EffectBowlingSelf
	SpearStabSelf     = db.EffectSpearStabSelf
	SpearBmrSelf      = db.EffectSpearBmrSelf
	HolyLight         = db.EffectHolyLight
	Concentration     = db.EffectConcentration
	RefineOK          = db.EffectRefineOK
	RefineFail        = db.EffectRefineFail
	JobLevelUp        = db.EffectJobLevelUp
	Rain              = db.EffectRain
	Snow              = db.EffectSnow
	Sakura            = db.EffectSakura
	Banjjakii         = db.EffectBanjjakii
	MakeBlur          = db.EffectMakeBlur
	EnergyCoat        = db.EffectEnergyCoat
	CartRevolution    = db.EffectCartRevolution
	VenomDust2        = db.EffectVenomDust2
	MentalBreak       = db.EffectMentalBreak
	MagicalAtkHit     = db.EffectMagicalAtkHit
	SuiExplosion      = db.EffectSuiExplosion
	Suicide           = db.EffectSuicide
	ComboAttack1      = db.EffectComboAttack1
	ComboAttack2      = db.EffectComboAttack2
	ComboAttack3      = db.EffectComboAttack3
	ComboAttack4      = db.EffectComboAttack4
	ComboAttack5      = db.EffectComboAttack5
	GuidedAttack      = db.EffectGuidedAttack
	PoisonAttack2     = db.EffectPoisonAttack2
	SilenceAttack     = db.EffectSilenceAttack
	StunAttack        = db.EffectStunAttack
	PetrifyAttack     = db.EffectPetrifyAttack
	SleepAttack       = db.EffectSleepAttack
	Pong              = db.EffectPong
	Level99           = db.EffectLevel99
	Level99Ground     = db.EffectLevel99Ground
	Level99Bubble     = db.EffectLevel99Bubble
	Gumgang           = db.EffectGumgang
	PotionRed         = db.EffectPotionRed
	PotionOrange      = db.EffectPotionOrange
	PotionYellow      = db.EffectPotionYellow
	PotionWhite       = db.EffectPotionWhite
	PotionBlue        = db.EffectPotionBlue
	PotionGreen       = db.EffectPotionGreen
	Food              = db.EffectFood
	FoodBlue          = db.EffectFoodBlue
	DarkBreath        = db.EffectDarkBreath
	Defender          = db.EffectDefender
	Keeping           = db.EffectKeeping
	SummonSlave       = db.EffectSummonSlave
	BloodDrain        = db.EffectBloodDrain
	EnergyDrain       = db.EffectEnergyDrain
	ItemFast          = db.EffectItemFast
	ItemFast2         = db.EffectItemFast2
	ItemFast3         = db.EffectItemFast3
	CrusaderDef       = db.EffectCrusaderDef
	Ganbantein        = db.EffectGanbantein
	Volcano           = db.EffectVolcano
	GrandCross        = db.EffectGrandCross
	Intimidate        = db.EffectIntimidate
	Chookgi           = db.EffectChookgi
	Cloud             = db.EffectCloud
	Cloud2            = db.EffectCloud2
	MapPillar         = db.EffectMapPillar
	LineLink          = db.EffectLineLink
	Cloud3            = db.EffectCloud3
	SpellBreaker      = db.EffectSpellBreaker
	Dispell           = db.EffectDispell
	Deluge            = db.EffectDeluge
	ViolentGale       = db.EffectViolentGale
	LandProtector     = db.EffectLandProtector
	BottomVolcano     = db.EffectBottomVolcano
	BottomDeluge      = db.EffectBottomDeluge
	BottomViolent     = db.EffectBottomViolent
	BottomLand        = db.EffectBottomLand
	MagicRod          = db.EffectMagicRod
	HolyCross         = db.EffectHolyCross
	ShieldCharge      = db.EffectShieldCharge
	MapPillar2        = db.EffectMapPillar2
	Providence        = db.EffectProvidence
	ShieldBoomer      = db.EffectShieldBoomer
	SpearQuicken      = db.EffectSpearQuicken
	Devotion          = db.EffectDevotion
	ReflectShield     = db.EffectReflectShield
	AbsorbSpirits     = db.EffectAbsorbSpirits
	SteelBody         = db.EffectSteelBody
	FlameLauncher     = db.EffectFlameLauncher
	FrostWeapon       = db.EffectFrostWeapon
	LightningLoad     = db.EffectLightningLoad
	SeismicWeapon     = db.EffectSeismicWeapon
	MapPillar3        = db.EffectMapPillar3
	MapPillar4        = db.EffectMapPillar4
	Gumgang2          = db.EffectGumgang2
	TeiHit1           = db.EffectTeiHit1
	Gumgang3          = db.EffectGumgang3
	TeiHit2           = db.EffectTeiHit2
	Tanji             = db.EffectTanji
	TeiHit1X          = db.EffectTeiHit1X
	Chimto            = db.EffectChimto
	StealCoin         = db.EffectStealCoin
	StripWeapon       = db.EffectStripWeapon
	StripShield       = db.EffectStripShield
	StripArmor        = db.EffectStripArmor
	StripHelm         = db.EffectStripHelm
	ChainCombo        = db.EffectChainCombo
	RogueCoin         = db.EffectRogueCoin
	BackStab          = db.EffectBackStab
	TeiHit3           = db.EffectTeiHit3
	BottomLullaby     = db.EffectBottomLullaby
	BottomRichKim     = db.EffectBottomRichKim
	BottomChaos       = db.EffectBottomChaos
	BottomDrum        = db.EffectBottomDrum
	BottomNibelung    = db.EffectBottomNibelung
	BottomRoki        = db.EffectBottomRoki
	BottomAbyss       = db.EffectBottomAbyss
	BottomSieg        = db.EffectBottomSieg
	BottomWhistle     = db.EffectBottomWhistle
	BottomSinX        = db.EffectBottomSinX
	BottomBragi       = db.EffectBottomBragi
	BottomApple       = db.EffectBottomApple
	BottomHumming     = db.EffectBottomHumming
	BottomForget      = db.EffectBottomForget
	BottomFortune     = db.EffectBottomFortune
	BottomService     = db.EffectBottomService
	TalkFrostJoke     = db.EffectTalkFrostJoke
	TalkScream        = db.EffectTalkScream
	PokJuk            = db.EffectPokJuk
	ThrowItem         = db.EffectThrowItem
	ThrowItem2        = db.EffectThrowItem2
	ChemicalProt      = db.EffectChemicalProt
	Demonstration     = db.EffectDemonstration
	Chemical2         = db.EffectChemical2
	Teleportation     = db.EffectTeleportation
	PharmacyOK        = db.EffectPharmacyOK
	PharmacyFail      = db.EffectPharmacyFail
	ThrowItem3        = db.EffectThrowItem3
	FirstAid          = db.EffectFirstAid
	SprinkleSand      = db.EffectSprinkleSand
	Loud              = db.EffectLoud
	Heal              = db.EffectHeal
	Heal2             = db.EffectHeal2
	Exit2             = db.EffectExit2
	SafetyWall        = db.EffectSafetyWall
	ReadyPortal       = db.EffectReadyPortal
	Portal            = db.EffectPortal
	BottomMagnus      = db.EffectBottomMagnus
	BottomSanc        = db.EffectBottomSanc
	HealOffensive     = db.EffectHealOffensive
	WarpZone2         = db.EffectWarpZone2
	Heal4             = db.EffectHeal4
	BeginAsura        = db.EffectBeginAsura
	TripleAttack      = db.EffectTripleAttack
	HitLine           = db.EffectHitLine
	HPTime            = db.EffectHPTime
	SPTime            = db.EffectSPTime
	Maple             = db.EffectMaple
	Blind             = db.EffectBlind
	PoisonStatus      = db.EffectPoisonStatus
	Guard             = db.EffectGuard
	JobLvUp50         = db.EffectJobLvUp50
	Angel2            = db.EffectAngel2
	Magnum2           = db.EffectMagnum2
	CoupleCasting     = db.EffectCoupleCasting
	Entry2            = db.EffectEntry2
	ColorPaper        = db.EffectColorPaper
	SoulBreaker       = db.EffectSoulBreaker
	Level99Aura1      = db.EffectLevel99Aura1
	FoodChocolate     = db.EffectFoodChocolate
	Pressure          = db.EffectPressure
	Bash3D            = db.EffectBash3D
	AuraBlade         = db.EffectAuraBlade
	RedBody           = db.EffectRedBody
	LKConcentration   = db.EffectLKConcentration
	BottomGospel      = db.EffectBottomGospel
	BaseLevelUp       = db.EffectBaseLevelUp
	Death             = db.EffectDeath
	DragonSmoke       = db.EffectDragonSmoke
	BottomBasilica    = db.EffectBottomBasilica
	Assumptio         = db.EffectAssumptio
	HitLine2          = db.EffectHitLine2
	Bash3D2           = db.EffectBash3D2
	EnergyDrain2      = db.EffectEnergyDrain2
	TransBlueBody     = db.EffectTransBlueBody
	MagicCrasher      = db.EffectMagicCrasher
	LightBlade        = db.EffectLightBlade
	EnergyDrain3      = db.EffectEnergyDrain3
	LineLink2         = db.EffectLineLink2
	LinkLight         = db.EffectLinkLight
	TrueSight         = db.EffectTrueSight
	FalconAssault     = db.EffectFalconAssault
	TripleAttack2     = db.EffectTripleAttack2
	Portal4           = db.EffectPortal4
	Meltdown          = db.EffectMeltdown
	CartBoost         = db.EffectCartBoost
	RejectSword       = db.EffectRejectSword
	TripleAttack3     = db.EffectTripleAttack3
	Moonlit           = db.EffectMoonlit
	MarionetteLink    = db.EffectMarionetteLink
	PinkBody          = db.EffectPinkBody
	Level99AuraMid    = db.EffectLevel99AuraMid
	Level99AuraBottom = db.EffectLevel99AuraBottom
	Bash3D3           = db.EffectBash3D3
	Bash3D4           = db.EffectBash3D4
	NapalmVulcan      = db.EffectNapalmVulcan
	Portal5           = db.EffectPortal5
	MagicCrasher2     = db.EffectMagicCrasher2
	BottomSpider      = db.EffectBottomSpider
	FogWallGround     = db.EffectFogWallGround
	SoulBurn          = db.EffectSoulBurn
	SoulChange        = db.EffectSoulChange
	SoulBreaker2      = db.EffectSoulBreaker2
	Rainbow           = db.EffectRainbow
	Peong             = db.EffectPeong
	PressedBody       = db.EffectPressedBody
	SpinedBody        = db.EffectSpinedBody
	KickedBody        = db.EffectKickedBody
	DoubleGumgang     = db.EffectDoubleGumgang
	ReflectBody       = db.EffectReflectBody
	BabyBody          = db.EffectBabyBody
	BabyBody2         = db.EffectBabyBody2
	GiantBody         = db.EffectGiantBody
	GiantBody2        = db.EffectGiantBody2
	AsuraBody         = db.EffectAsuraBody
	FourWayBody       = db.EffectFourWayBody
	QuakeBody         = db.EffectQuakeBody
	AsuraBodyMonster  = db.EffectAsuraBodyMonster
	HitLine4          = db.EffectHitLine4
	HitLine5          = db.EffectHitLine5
	HitLine6          = db.EffectHitLine6
	StormKick         = db.EffectStormKick
	Chemical3         = db.EffectChemical3
	Assumptio2        = db.EffectAssumptio2
	BlueCasting       = db.EffectBlueCasting
	Run               = db.EffectRun
	StopRun           = db.EffectStopRun
	StopEffect        = db.EffectStopEffect
	JumpBody          = db.EffectJumpBody
	LandBody          = db.EffectLandBody
	Foot3             = db.EffectFoot3
	Foot4             = db.EffectFoot4
	TaeReady          = db.EffectTaeReady
	DarkGrandCross    = db.EffectDarkGrandCross
	DarkSoulStrike    = db.EffectDarkSoulStrike
	DarkJupitelHit    = db.EffectDarkJupitelHit
	NPCStop           = db.EffectNPCStop
	DarkCasting       = db.EffectDarkCasting
	AgiUp             = db.EffectAgiUp
	JumpKick          = db.EffectJumpKick
	QuakeBody2        = db.EffectQuakeBody2
	StormKick3        = db.EffectStormKick3
	StormKick4        = db.EffectStormKick4
	SpinedBody2       = db.EffectSpinedBody2
	BeginAsura1       = db.EffectBeginAsura1
	BeginAsura2       = db.EffectBeginAsura2
	BeginAsura3       = db.EffectBeginAsura3
	BeginAsura4       = db.EffectBeginAsura4
	BeginAsura5       = db.EffectBeginAsura5
	BeginAsura6       = db.EffectBeginAsura6
	BeginAsura7       = db.EffectBeginAsura7
	Devil1            = db.EffectDevil1
	Devil2            = db.EffectDevil2
	Devil3            = db.EffectDevil3
	Devil4            = db.EffectDevil4
	Devil5            = db.EffectDevil5
	Devil6            = db.EffectDevil6
	Devil7            = db.EffectDevil7
	Devil8            = db.EffectDevil8
	Devil9            = db.EffectDevil9
	Devil10           = db.EffectDevil10
	BlackDevil        = db.EffectBlackDevil
	FlowerCast        = db.EffectFlowerCast
	ResistPotion      = db.EffectResistPotion
	Mochi             = db.EffectMochi
	Ramadan           = db.EffectRamadan
	EDP               = db.EffectEDP
	FullStrip         = db.EffectFullStrip
	Preserve          = db.EffectPreserve
	Slim              = db.EffectSlim
	Slim2             = db.EffectSlim2
	Slim3             = db.EffectSlim3
	ChemicalBody      = db.EffectChemicalBody
	CastSpin          = db.EffectCastSpin
	PierceBody        = db.EffectPierceBody
	SoulLink          = db.EffectSoulLink
	Chookgi2          = db.EffectChookgi2
	Memorize          = db.EffectMemorize
	ItemAccel         = db.EffectItemAccel
	Mapae             = db.EffectMapae
	Firecracker       = db.EffectFirecracker
	ItemPokJuk        = db.EffectItemPokJuk
	Valentine05       = db.EffectValentine05
	BeginAsura11      = db.EffectBeginAsura11
	Chemical2Dash     = db.EffectChemical2Dash
	GroundSample      = db.EffectGroundSample
	Cloud4            = db.EffectCloud4
	Cloud5            = db.EffectCloud5
	BottomHermode     = db.EffectBottomHermode
	CartTermination   = db.EffectCartTermination
	ItemSlow          = db.EffectItemSlow
	ItemFastDown      = db.EffectItemFastDown
	ShieldBoomerang3  = db.EffectShieldBoomerang3
	DoubleCastBody    = db.EffectDoubleCastBody
	Gravitation       = db.EffectGravitation
	TarotCard1        = db.EffectTarotCard1
	TarotCard2        = db.EffectTarotCard2
	TarotCard3        = db.EffectTarotCard3
	TarotCard4        = db.EffectTarotCard4
	TarotCard5        = db.EffectTarotCard5
	TarotCard6        = db.EffectTarotCard6
	TarotCard7        = db.EffectTarotCard7
	TarotCard8        = db.EffectTarotCard8
	TarotCard9        = db.EffectTarotCard9
	TarotCard10       = db.EffectTarotCard10
	TarotCard11       = db.EffectTarotCard11
	TarotCard12       = db.EffectTarotCard12
	TarotCard13       = db.EffectTarotCard13
	TarotCard14       = db.EffectTarotCard14
	AcidDemon         = db.EffectAcidDemon
	ThrowItem4        = db.EffectThrowItem4
	BabyBodyBack      = db.EffectBabyBodyBack
	ThrowItem5        = db.EffectThrowItem5
	BlueBody          = db.EffectBlueBody
	Hated             = db.EffectHated
	RedLightBody      = db.EffectRedLightBody
	SmaReady          = db.EffectSmaReady
	Stin              = db.EffectStin
	RedHit            = db.EffectRedHit
	BlueHit           = db.EffectBlueHit
	QuakeBody3        = db.EffectQuakeBody3
	Stin2             = db.EffectStin2
	HitTexture        = db.EffectHitTexture
	Stin3             = db.EffectStin3
	ScreenQuake       = db.EffectScreenQuake
	HfliMoon1         = db.EffectHfliMoon1
	HfliMoon2         = db.EffectHfliMoon2
	HfliMoon3         = db.EffectHfliMoon3
	HoUp              = db.EffectHoUp
	HamiDefence       = db.EffectHamiDefence
	HamiCastle        = db.EffectHamiCastle
	HamiBlood         = db.EffectHamiBlood
	Hated2            = db.EffectHated2
	Twilight1         = db.EffectTwilight1
	Twilight2         = db.EffectTwilight2
	Twilight3         = db.EffectTwilight3
	BoxThunder        = db.EffectBoxThunder
	ItemThunder       = db.EffectItemThunder
	BoxResentment     = db.EffectBoxResentment
	ItemCloud         = db.EffectItemCloud
	ItemCurse         = db.EffectItemCurse
	BoxDrowsiness     = db.EffectBoxDrowsiness
	ItemZZZ           = db.EffectItemZZZ
	BoxSunlight       = db.EffectBoxSunlight
	ItemRain          = db.EffectItemRain
	M01               = db.EffectM01
	M02               = db.EffectM02
	M03               = db.EffectM03
	M04               = db.EffectM04
	M05               = db.EffectM05
	M06               = db.EffectM06
	M07               = db.EffectM07
	Kaizel            = db.EffectKaizel
	Cloud6            = db.EffectCloud6
	StatFoodSTR       = db.EffectStatFoodSTR
	StatFoodINT       = db.EffectStatFoodINT
	StatFoodVIT       = db.EffectStatFoodVIT
	StatFoodAGI       = db.EffectStatFoodAGI
	StatFoodDEX       = db.EffectStatFoodDEX
	StatFoodLUK       = db.EffectStatFoodLUK
	Shrink            = db.EffectShrink
	ThrowItem6        = db.EffectThrowItem6
	SightBlaster      = db.EffectSightBlaster
	QuakeBody4        = db.EffectQuakeBody4
	FireHit2          = db.EffectFireHit2
	NPCStop2          = db.EffectNPCStop2
	NPCStop2Del       = db.EffectNPCStop2Del
	FVoice            = db.EffectFVoice
	Wink              = db.EffectWink
	CookingOK         = db.EffectCookingOK
	CookingFail       = db.EffectCookingFail
	Firecracker1      = db.EffectFirecracker1
	Hapgyeok          = db.EffectHapgyeok
	ThrowItem7        = db.EffectThrowItem7
	ThrowItem8        = db.EffectThrowItem8
	ThrowItem9        = db.EffectThrowItem9
	ThrowItem10       = db.EffectThrowItem10
	Kouenka           = db.EffectKouenka
	Hyousensou        = db.EffectHyousensou
	BottomSuiton      = db.EffectBottomSuiton
	Stin4             = db.EffectStin4
	ThunderStorm2     = db.EffectThunderStorm2
	Chemical4         = db.EffectChemical4
	Stin5             = db.EffectStin5
	MadnessBlue       = db.EffectMadnessBlue
	MadnessRed        = db.EffectMadnessRed
	RGCoin3           = db.EffectRGCoin3
	Bash3D5           = db.EffectBash3D5
	Chookgi3          = db.EffectChookgi3
	Kirikage          = db.EffectKirikage
	Tatami            = db.EffectTatami
	Kasumikiri        = db.EffectKasumikiri
	Issen             = db.EffectIssen
	Kaen              = db.EffectKaen
	Baku              = db.EffectBaku
	Hyousyouraku      = db.EffectHyousyouraku
	Desperado         = db.EffectDesperado
	LightningS        = db.EffectLightningS
	BlindS            = db.EffectBlindS
	PoisonS           = db.EffectPoisonS
	FreezingS         = db.EffectFreezingS
	FlareS            = db.EffectFlareS
	RapidShower       = db.EffectRapidShower
	MagicalBullet     = db.EffectMagicalBullet
	SpreadAttack      = db.EffectSpreadAttack
	TrackCasting      = db.EffectTrackCasting
	Tracking          = db.EffectTracking
	TripleAction      = db.EffectTripleAction
	Bullseye          = db.EffectBullseye
	BubbleDrop        = db.EffectBubbleDrop
	NPCEarthquake     = db.EffectNPCEarthquake
	DragonFear        = db.EffectDragonFear
	WideBleeding      = db.EffectWideBleeding
	WideConfuse       = db.EffectWideConfuse
	BottomRunner      = db.EffectBottomRunner
	BottomTransfer    = db.EffectBottomTransfer
	BottomEvilLand    = db.EffectBottomEvilLand
	Guard3            = db.EffectGuard3
	CriticalWound     = db.EffectCriticalWound
	Firecracker2      = db.EffectFirecracker2
	Firecracker3      = db.EffectFirecracker3
	Firecracker4      = db.EffectFirecracker4
	Firecracker5      = db.EffectFirecracker5
	Firecracker6      = db.EffectFirecracker6
	TorchRed          = db.EffectTorchRed
	TorchGreen        = db.EffectTorchGreen
	MapGhost          = db.EffectMapGhost
	Glow1             = db.EffectGlow1
	Glow2             = db.EffectGlow2
	Glow4             = db.EffectGlow4
	TorchPurple       = db.EffectTorchPurple
	Cloud7            = db.EffectCloud7
	Cloud8            = db.EffectCloud8
	FlowerLeaf        = db.EffectFlowerLeaf
	Item315           = db.EffectItem315
	Item316           = db.EffectItem316
	Item317           = db.EffectItem317
	StormMin          = db.EffectStormMin
	Firecracker7      = db.EffectFirecracker7
	BottomBlue        = db.EffectBottomBlue
	BottomBlue2       = db.EffectBottomBlue2
	ChristmasCarol    = db.EffectChristmasCarol
	FirePillarOn2     = db.EffectFirePillarOn2
	ForestLight5      = db.EffectForestLight5
	Adoramus          = db.EffectAdoramus
	IgnitionBreak     = db.EffectIgnitionBreak
	FrostMisty        = db.EffectFrostMisty
	CrimsonRock       = db.EffectCrimsonRock
	HellInferno       = db.EffectHellInferno
	MarshOfAbyss      = db.EffectMarshOfAbyss
	DragonHowling     = db.EffectDragonHowling
	EarthWall         = db.EffectEarthWall
	ChainLightning    = db.EffectChainLightning
	AimedBolt         = db.EffectAimedBolt
	ArrowStorm        = db.EffectArrowStorm
	Laulamus          = db.EffectLaulamus
	Lauagnus          = db.EffectLauagnus
	MillenniumShield  = db.EffectMillenniumShield
	Concentration2    = db.EffectConcentration2
	GlassWall3        = db.EffectGlassWall3
	BerserkPotion2    = db.EffectBerserkPotion2
	Rolling1          = db.EffectRolling1
	Rolling2          = db.EffectRolling2
	Rolling3          = db.EffectRolling3
	Rolling4          = db.EffectRolling4
	Rolling5          = db.EffectRolling5
	Rolling6          = db.EffectRolling6
	Rolling7          = db.EffectRolling7
	Rolling8          = db.EffectRolling8
	Rolling9          = db.EffectRolling9
	Rolling10         = db.EffectRolling10
	CastSpin2         = db.EffectCastSpin2
	CrashAxe          = db.EffectCrashAxe
	Stasis            = db.EffectStasis
	BottomBasilica2   = db.EffectBottomBasilica2
	Recognized        = db.EffectRecognized
	Tetra             = db.EffectTetra
	TetraCasting      = db.EffectTetraCasting
	Stretch           = db.EffectStretch
	Enervation        = db.EffectEnervation
	Enervation2       = db.EffectEnervation2
	Enervation3       = db.EffectEnervation3
	Enervation4       = db.EffectEnervation4
	Enervation5       = db.EffectEnervation5
	Enervation6       = db.EffectEnervation6
	BottomManhole     = db.EffectBottomManhole
	Manhole           = db.EffectManhole
	ForestLight6      = db.EffectForestLight6
	BottomAni         = db.EffectBottomAni
	BottomMaelstrom   = db.EffectBottomMaelstrom
	BottomBloodyLust  = db.EffectBottomBloodyLust
	HealN             = db.EffectHealN
	ChookgiN          = db.EffectChookgiN
	Dance1            = db.EffectDance1
	BotReverb         = db.EffectBotReverb
	RainParticle      = db.EffectRainParticle
	ChemicalV2        = db.EffectChemicalV2
	BotReverb2        = db.EffectBotReverb2
	CirclePower2      = db.EffectCirclePower2
	Secra2            = db.EffectSecra2
	SprPlant2         = db.EffectSprPlant2
	SprPlant3         = db.EffectSprPlant3
	SprPlant4         = db.EffectSprPlant4
	SprPlant5         = db.EffectSprPlant5
	SprPlant6         = db.EffectSprPlant6
	SprPlant7         = db.EffectSprPlant7
	SprPlant8         = db.EffectSprPlant8
	HeartAsura        = db.EffectHeartAsura
	GlassWall4        = db.EffectGlassWall4
	Bash3D6           = db.EffectBash3D6
	Electric4         = db.EffectElectric4
	TeiHit1T          = db.EffectTeiHit1T
	Pressure2         = db.EffectPressure2
	PrimeCharge2      = db.EffectPrimeCharge2
	PrimeCharge3      = db.EffectPrimeCharge3
	PrimeCharge4      = db.EffectPrimeCharge4
	FireWall2         = db.EffectFireWall2
	SprPlant10        = db.EffectSprPlant10
	Shockwave2        = db.EffectShockwave2
	ColdThrow2        = db.EffectColdThrow2
	DemonicFire4      = db.EffectDemonicFire4
	Pressure3         = db.EffectPressure3
	PoisonMist        = db.EffectPoisonMist
	EraserCutter      = db.EffectEraserCutter
	LavaSlide         = db.EffectLavaSlide
	SonicClaw         = db.EffectSonicClaw
	TinderBreaker     = db.EffectTinderBreaker
	MidnightFrenzy    = db.EffectMidnightFrenzy
	VolcanicAsh       = db.EffectVolcanicAsh
	RWC2011           = db.EffectRWC2011
	RWC2011Two        = db.EffectRWC2011Two
	RunMakeOK         = db.EffectRunMakeOK
	RunMakeFailure    = db.EffectRunMakeFailure
	MIResultMakeOK    = db.EffectMIResultMakeOK
	MIResultMakeFail  = db.EffectMIResultMakeFail
	AllRayProtect     = db.EffectAllRayProtect
	VenomFog          = db.EffectVenomFog
	DustStorm         = db.EffectDustStorm
	DanceBladeAtk     = db.EffectDanceBladeAtk
	InvincibleOff2    = db.EffectInvincibleOff2
	DeathSummon       = db.EffectDeathSummon
	GCDarkCrow        = db.EffectGCDarkCrow
	AllFullThrottle   = db.EffectAllFullThrottle
	SRFlashCombo      = db.EffectSRFlashCombo
	RKLuxAnima        = db.EffectRKLuxAnima
	SOElemShield      = db.EffectSOElemShield
	ABOffertorium     = db.EffectABOffertorium
	WLTelekinesis     = db.EffectWLTelekinesis
	GNIllusionDoping  = db.EffectGNIllusionDoping
	NCMagmaEruption   = db.EffectNCMagmaEruption
	NPCChill          = db.EffectNPCChill
	OffertoriumRing   = db.EffectOffertoriumRing
	HammerOfGod       = db.EffectHammerOfGod
	AchComplete       = db.EffectAchComplete
	BodyColor         = db.EffectBodyColor
	BakuretsuHadou    = db.EffectBakuretsuHadou
	DropPink          = db.EffectDropPink
	DropYellow        = db.EffectDropYellow
	DropPurple        = db.EffectDropPurple
	DigitalSpace      = db.EffectDigitalSpace
	DropBlue          = db.EffectDropBlue
	DropGreen         = db.EffectDropGreen
	DropRed           = db.EffectDropRed
	NewSuccess        = db.EffectNewSuccess
	NewFailure        = db.EffectNewFailure
	NewIntro          = db.EffectNewIntro
	EnchantYellow     = db.EffectEnchantYellow
	EnchantSuccess    = db.EffectEnchantSuccess
	EnchantFail       = db.EffectEnchantFail
	EnchantBlue       = db.EffectEnchantBlue
	EnchantUpSuccess  = db.EffectEnchantUpSuccess
	EnchantUpFail     = db.EffectEnchantUpFail
	EnchantGreen      = db.EffectEnchantGreen
	EnchantResetOK    = db.EffectEnchantResetOK
	EnchantResetFail  = db.EffectEnchantResetFail
)

// Goro-local effect IDs.
const (
	ColdBolt               = db.EffectColdBolt
	FireBolt               = db.EffectFireBolt
	CastRing               = db.EffectCastRing
	QuakeMagnum            = db.EffectQuakeMagnum
	ArrowShot              = db.EffectArrowShot
	ArrowShower            = db.EffectArrowShower
	MagicPower             = db.EffectMagicPower
	WhitePulse             = db.EffectWhitePulse
	SpearProjectile        = db.EffectSpearProjectile
	SpearHitSound          = db.EffectSpearHitSound
	EnemyHitNormal1        = db.EffectEnemyHitNormal1
	Quake                  = db.EffectQuake
	AnkleSnareGround       = db.EffectAnkleSnareGround
	SharpShootingCast      = db.EffectSharpShootingCast
	AdrenalineCast         = db.EffectAdrenalineCast
	MaximizeSounds         = db.EffectMaximizeSounds
	GreedSound             = db.EffectGreedSound
	GospelGround           = db.EffectGospelGround
	HermodeMusic           = db.EffectHermodeMusic
	BottomDissonanceGround = db.EffectBottomDissonanceGround
	BottomLullabyGround    = db.EffectBottomLullabyGround
	BottomRichKimGround    = db.EffectBottomRichKimGround
	BottomChaosGround      = db.EffectBottomChaosGround
	BottomDrumGround       = db.EffectBottomDrumGround
	BottomNibelungGround   = db.EffectBottomNibelungGround
	BottomRokiGround       = db.EffectBottomRokiGround
	BottomAbyssGround      = db.EffectBottomAbyssGround
	BottomSiegGround       = db.EffectBottomSiegGround
	BottomWhistleGround    = db.EffectBottomWhistleGround
	BottomSinXGround       = db.EffectBottomSinXGround
	BottomBragiGround      = db.EffectBottomBragiGround
	BottomAppleGround      = db.EffectBottomAppleGround
	BottomUglyDanceGround  = db.EffectBottomUglyDanceGround
	BottomHummingGround    = db.EffectBottomHummingGround
	BottomForgetGround     = db.EffectBottomForgetGround
	BottomFortuneGround    = db.EffectBottomFortuneGround
	BottomServiceGround    = db.EffectBottomServiceGround
	GunslingerCoinSound    = db.EffectGunslingerCoinSound
	Emotion                = db.EffectEmotion
)

const SkillUnitEffectFallbackDuration = 5 * time.Minute

type ComponentKind int

const (
	ComponentSTR ComponentKind = iota + 1
	ComponentCylinder
	Component2D
	Component3D
	ComponentSPR
	ComponentFUNC
	ComponentQuadHorn
)

type Instance struct {
	EffectID                             int
	Params                               [4]float32
	ActorID                              uint32
	TargetID                             uint32
	castSourceID                         uint32
	randomSeed                           uint32
	ServerTick                           uint32
	ActorDirection                       int
	ActorOrigin                          scene.Point3
	ProjectileDirection                  scene.Point3
	HitCount                             int
	Shield                               *shieldFlight
	ShieldChain                          *shieldChainFlight
	pressure                             *pressureFlight
	acidBottles                          *acidBottleFlight
	firefly                              *fireflyPath
	mapParticles                         *mapParticleState
	level99Bubbles                       *level99BubbleState
	sakuraPetals                         *sakuraPetalState
	snow                                 *snowState
	NinjaWind                            *NinjaWind
	SpiritSpheres                        *spiritSpheres
	NinjaPetals                          *NinjaPetals
	Esma                                 *esmaProjectile
	BuffParticles                        *risingBuffParticles
	SoulLights                           *SoulLights
	BlueCasting                          *CastingState
	asuraGlyphs                          *asuraGlyphs
	taekwonJump                          *taekwonJumpParticles
	taekwonRun                           *taekwonRunTrail
	Twilight                             *twilightParticles
	ItemLevel                            int
	CoinTrails                           *[5][]scene.Point3
	ArrowTargets                         *[9]scene.Point3
	arrow                                *arrowFlight
	arrowVulcan                          *arrowVulcanRays
	HasActorTransform                    bool
	superAngel                           *superAngel
	X                                    int
	Y                                    int
	Starts                               time.Time
	Expires                              time.Time
	Duration                             time.Duration
	Size                                 float64
	GroundSampleRotationRadiansPerSecond float64
	Persistent                           bool
	SpriteFrameOverride                  int
	HasSpriteFrame                       bool
}

// State-owned effects have no deadline; their server update removes them.
func (effect Instance) Expired(now time.Time) bool {
	return !effect.Expires.IsZero() && now.After(effect.Expires)
}

type Spec struct {
	Duration         time.Duration
	CameraShake      time.Duration
	CameraShakeDelay time.Duration
	DetachLocalActor bool
	Sfx              []string
	SfxDelays        []time.Duration
	SfxRandMin       int
	SfxRandMax       int
	Components       []Component
}

type Component struct {
	Kind               ComponentKind
	hitRing            *db.EffectHitRing
	hitParticles       *db.EffectHitParticles
	FuncAdapter        EffectFuncAdapter
	FuncName           string
	Color              color.RGBA
	Duration           time.Duration
	durationRandMin    time.Duration
	durationRandMax    time.Duration
	Delay              time.Duration
	DuplicateDelay     time.Duration
	delayOffsetDelta   time.Duration
	Repeat             bool
	RepeatDelay        time.Duration
	StrFile            string
	StrMinFile         string
	StrRandMin         int
	StrRandMax         int
	AttachedEntity     bool
	RenderBefore       bool
	TexturePath        string
	TextureName        string
	TextureFile        string
	TextureFiles       []string
	FrameDelay         time.Duration
	SpriteFile         string
	ShadowTexture      bool
	SpriteHead         bool
	SpriteDirection    bool
	SpriteRepeat       bool
	SpriteStopAtEnd    bool
	spriteFrame        int
	SpriteDelay        time.Duration
	spriteXOffset      float64
	SpriteYOffset      float64
	fromSrc            bool
	ToSrc              bool
	Arc                float64
	Retreat            float64
	AlphaMax           float64
	AlphaMaxDelta      float64
	Sparkling          bool
	SparkNumber        int
	Fade               bool
	FadeIn             bool
	FadeOut            bool
	Rotate             bool
	RotateWithCamera   bool
	FixedPerspective   bool
	RotateToTarget     bool
	WorldSizedSprite   bool
	Animation          int
	BottomSize         float64
	TopSize            float64
	Height             float64
	PosX               float64
	PosY               float64
	PosZ               float64
	PosXEnd            float64
	PosYEnd            float64
	PosZEnd            float64
	PosXRand           float64
	PosYRand           float64
	PosZRand           float64
	PosXStartRand      float64
	PosYStartRand      float64
	PosZStartRand      float64
	posXStartMiddle    float64
	posYStartMiddle    float64
	PosZStartMiddle    float64
	PosXEndRand        float64
	PosYEndRand        float64
	PosZEndRand        float64
	posXEndMiddle      float64
	posYEndMiddle      float64
	PosZEndMiddle      float64
	PosXSmooth         bool
	PosYSmooth         bool
	PosZSmooth         bool
	SizeStart          float64
	SizeEnd            float64
	SizeRand           float64
	SizeStartX         float64
	SizeStartY         float64
	SizeEndX           float64
	SizeEndY           float64
	sizeStartXRandMin  float64
	sizeStartXRandMax  float64
	sizeStartYRandMin  float64
	sizeStartYRandMax  float64
	sizeEndXRandMin    float64
	sizeEndXRandMax    float64
	sizeEndYRandMin    float64
	sizeEndYRandMax    float64
	sizeRandX          float64
	SizeRandY          float64
	sizeRandXMiddle    float64
	SizeRandYMiddle    float64
	SizeDelta          float64
	SizeSmooth         bool
	AngleStart         float64
	AngleEnd           float64
	AngleX             float64
	AngleY             float64
	angleZ             float64
	angleRandMin       float64
	angleRandMax       float64
	circlePattern      bool
	circleInnerSize    float64
	circleOuterRandMin float64
	circleOuterRandMax float64
	OrbitRadiusX       float64
	OrbitRadiusY       float64
	orbitRadiusZ       float64
	OrbitRotations     float64
	OrbitPhase         float64
	orbitPhaseDelta    float64
	OrbitClockwise     bool
	TotalCircleSides   int
	CircleSides        int
	Duplicate          int
	AngleZRandom       float64
	BlendMode          int
	BlendAdditive      bool
	Overlay            bool
	QuadHornHeightMin  float64
	QuadHornHeightMax  float64
	QuadHornOffsetXMin float64
	QuadHornOffsetXMax float64
	QuadHornOffsetYMin float64
	QuadHornOffsetYMax float64
	QuadHornOffsetZ    float64
	QuadHornBottomMin  float64
	QuadHornBottomMax  float64
	QuadHornRotateXMin float64
	QuadHornRotateXMax float64
	QuadHornRotateYMin float64
	QuadHornRotateYMax float64
	QuadHornRotateZMin float64
	QuadHornRotateZMax float64
	QuadHornAnimSpeed  time.Duration
	QuadHornAnimOut    bool
}

func (m *System) RemoveUnit(id uint32) bool {
	return m.RemoveActor(id)
}

func (m *System) RemoveActor(id uint32) bool {
	delete(m.StarFlights, id)
	active := m.Active[:0]
	removed := false
	for _, effect := range m.Active {
		if effect.ActorID == id || ((effect.EffectID == LineLink || effect.EffectID == LineLink2 || effect.EffectID == MarionetteLink) && effect.TargetID == id) {
			removed = true
			continue
		}
		active = append(active, effect)
	}
	m.Active = active
	return removed
}

func (m *System) Add(ctx client.Context, effectID int, actorID uint32) bool {
	return m.AddAt(ctx, effectID, actorID, time.Now())
}

func (m *System) AddIfMissing(ctx client.Context, effectID int, actorID uint32) bool {
	if m.HasActive(effectID, actorID, time.Now()) {
		return false
	}
	return m.Add(ctx, effectID, actorID)
}

func (m *System) HasActive(effectID int, actorID uint32, now time.Time) bool {
	for _, effect := range m.Active {
		if effect.EffectID == effectID && effect.ActorID == actorID && !effect.Expired(now) {
			return true
		}
	}
	return false
}

func (m *System) Remove(effectID int, actorID uint32) bool {
	removed := false
	active := m.Active[:0]
	for _, effect := range m.Active {
		if effect.EffectID == effectID && effect.ActorID == actorID {
			removed = true
			continue
		}
		active = append(active, effect)
	}
	m.Active = active
	return removed
}

func (m *System) AddAt(ctx client.Context, effectID int, actorID uint32, starts time.Time) bool {
	return m.AddBetween(ctx, effectID, actorID, 0, starts)
}

func (m *System) AddBetween(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time) bool {
	return m.AddBetweenDuration(ctx, effectID, actorID, targetID, starts, 0)
}

func (m *System) AddBetweenDuration(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time, durationOverride time.Duration) bool {
	return m.addBetweenDuration(ctx, effectID, actorID, targetID, starts, durationOverride, false)
}

// AddPersistentBetween keeps an effect until its owning server state removes it.
func (m *System) AddPersistentBetween(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time) bool {
	return m.addBetweenDuration(ctx, effectID, actorID, targetID, starts, 0, true)
}

func (m *System) addBetweenDuration(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time, durationOverride time.Duration, persistent bool) bool {
	if ctx.World == nil {
		return false
	}
	spec, ok := Lookup(effectID)
	if !ok {
		return false
	}
	x, y, ok := EffectAnchor(ctx, actorID)
	if !ok {
		return false
	}
	if ctx.Config.Headless {
		// Scream and Frost Joke still send the caster's chat line.
		m.applyWorldEffectSideEffects(ctx, Instance{EffectID: effectID, ActorID: actorID}, starts)
		return false
	}
	if effectID == SightTrasher {
		m.Remove(Sight, actorID)
		if scene.IsLocalActor(ctx, actorID) {
			m.Remove(Sight, scene.LocalSkillTarget(ctx))
		}
	}
	if !persistent && !effectEnabled(ctx, effectID) {
		return false
	}
	// SAINTCASTING keeps a visible startup flare even for very short casts.
	if (effectID == BlueCasting || effectID == CoupleCasting) && durationOverride > 0 {
		durationOverride = max(durationOverride, 56*db.EffectFrameDuration)
	}
	effect := Instance{
		EffectID: effectID,
		ActorID:  actorID,
		TargetID: targetID,
		X:        x,
		Y:        y,
		Starts:   starts,
		Duration: durationOverride,
	}
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
		if componentDuration > duration {
			duration = componentDuration
		}
	}
	if duration <= 0 {
		duration = 500 * time.Millisecond
	}
	if durationOverride > 0 {
		duration = durationOverride
	}
	effect.Expires = starts.Add(duration)
	if effectID == BashBegin {
		// Envenom launches two flares on the same actor at the same instant.
		effect.randomSeed = m.nextRandomSeed
		m.nextRandomSeed++
	}
	if effectID == EnergyDrain2 {
		initSoulDrain(ctx, &effect)
	}
	if effectID == FirstAid || effectID == Angel2 {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
		if effectID == Angel2 {
			effect.superAngel = &superAngel{origins: [7]scene.Point3{effect.ActorOrigin}, count: 1}
		}
	}
	if effectID == EnergyDrain3 || effectID == FullStrip || effectID == Preserve || effectID == db.SkillEffectChaseWalkSmoke {
		effect.ActorDirection = EffectActorDirection(ctx, actorID, starts)
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == SoulBreaker || effectID == SoulBreaker2 {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
		if targetID != 0 {
			effect.ProjectileDirection = scene.Sub3(EffectWorldAnchor(ctx, Instance{ActorID: targetID}, starts), effect.ActorOrigin)
		}
	}
	if effectID == BackStab || effectID == TeiHit2 {
		initImpactParticles(ctx, &effect)
	}
	if effectID == Chemical4 || effectID == M02 || effectID == SpreadAttack {
		m.initGunslingerShot(ctx, &effect)
	}
	if effectID == RGCoin3 {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == Chookgi || effectID == Chookgi2 || effectID == Chookgi3 || effectID == ChookgiN {
		effect.HitCount = 5
		if actor, ok, _ := scene.ActorForCombatID(ctx, actorID); ok && actor.HasSpiritSpheres {
			effect.HitCount = min(int(actor.SpiritSpheres), 10)
		}
		effect.SpiritSpheres = NewSpiritSpheres(effect)
	}
	if effectID == FVoice || effectID == Wink {
		m.initDirectedSpriteEffect(ctx, &effect)
	}
	if effectID == TripleAttack || effectID == TripleAttack2 || effectID == TripleAttack3 || effectID == Chemical2Dash {
		initTripleAttack(ctx, &effect)
	}
	if effectID == TripleAttack3 {
		effect.arrowVulcan = &arrowVulcanRays{origins: [9]scene.Point3{effect.ActorOrigin}, count: 1}
	}
	if effectID >= TarotCard1 && effectID <= TarotCard14 || effectID == TeiHit1 || effectID == TeiHit1X || effectID == HitLine || effectID == HitLine2 || effectID == Portal4 || effectID == Portal5 || effectID == BeginAsura || effectID == BeginAsura11 {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == Hyousensou || effectID == RogueCoin || effectID == Intimidate || effectID == TeiHit3 || effectID == MagnumBreak || effectID == GrandCross || effectID == Pressure || effectID == ShieldBoomerang3 {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == Steal || effectID == Detoxication || effectID == SprinkleSand || effectID == ChemicalProt || effectID == AcidDemon {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == Mammonite {
		m.initMammonite(ctx, &effect)
	}
	if effectID == ArrowShot {
		initArrow(ctx, &effect)
	}
	if effectID == CartRevolution || effectID == HasteUp || effectID == RepairWeapon || effectID == MaximizePower {
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == Grimtooth || effectID == ThrowItem6 || effectID == ThrowItem3 || isAlchemistBottle(effectID) || effectID >= ThrowItem7 && effectID <= ThrowItem10 {
		initDirectedEffect(ctx, &effect)
	}
	if effectID == HitLine5 {
		effect.ProjectileDirection = scene.Sub3(EffectWorldAnchor(ctx, effect, starts), EffectWorldAnchor(ctx, Instance{ActorID: targetID}, starts))
	}
	if effectID == Chemical3 || effectID == StormKick || effectID == StormKick3 || effectID == Foot3 || effectID == Foot4 {
		if effectID == Chemical3 {
			destination := EffectWorldAnchor(ctx, effect, starts)
			effect.ActorID, effect.TargetID = targetID, actorID
			effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
			effect.ProjectileDirection = scene.Sub3(destination, EffectWorldAnchor(ctx, effect, starts))
		}
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID == ShieldBoomer || effectID == Tanji {
		initShieldBoomerang(ctx, &effect)
	}
	if effectID == ShieldBoomerang3 {
		effect.ShieldChain = &shieldChainFlight{Origins: [5]scene.Point3{effect.ActorOrigin}, Count: 1}
	}
	if effectID == Pressure {
		effect.pressure = &pressureFlight{}
	}
	if effectID == SpearBoomerang {
		initSpearBoomerang(ctx, &effect)
	}
	if effectID == FrostDiver || effectID == FireBall || effectID == SoulStrike ||
		effectID == JupitelThunder || effectID == WaterBall2 {
		initMageProjectile(ctx, &effect)
	}
	if effectID == Stin4 || effectID == Stin5 {
		destination := EffectWorldAnchor(ctx, effect, starts)
		effect.ActorOrigin = destination
		if _, ok, _ := scene.ActorForCombatID(ctx, targetID); targetID != 0 && ok {
			effect.ActorOrigin = EffectWorldAnchor(ctx, Instance{ActorID: targetID}, starts)
		}
		effect.ProjectileDirection = scene.Sub3(destination, effect.ActorOrigin)
		effect.ActorID, effect.TargetID = targetID, actorID
		effect.X, effect.Y, _ = EffectAnchor(ctx, effect.ActorID)
		effect.HasActorTransform = true
		effect.NinjaWind = &NinjaWind{}
	}
	if effectID == Kouenka {
		effect.NinjaPetals = &NinjaPetals{}
	}
	if effectID == Kirikage || effectID == Kasumikiri {
		effect.HasSpriteFrame = true
		effect.SpriteFrameOverride = 2 * HitEffectRandom(effect, 0, 2)
		if effectID == Kirikage {
			effect.SpriteFrameOverride *= 2
		}
	}
	if effectID == Stin || effectID == Stin3 || effectID == Stin2 {
		initSoulLinkerProjectile(ctx, &effect)
	}
	if effectID == Hated || effectID == Hated2 || effectID == Kaizel || effectID == SmaReady {
		effect.BuffParticles = &risingBuffParticles{}
	}
	if effectID == SoulLink || effectID == StormKick4 {
		effect.SoulLights = &SoulLights{}
		effect.ActorOrigin = EffectWorldAnchor(ctx, effect, starts)
		effect.HasActorTransform = true
	}
	if effectID >= Twilight1 && effectID <= Twilight3 {
		effect.Twilight = NewTwilightParticles(ctx, effect)
	}
	if effectID == BlueCasting || effectID == AuraBlade || effectID == CoupleCasting || effectID == BeginAsura || effectID == BeginAsura11 {
		effect.BlueCasting = &CastingState{auraBlade: effectID == AuraBlade}
	}
	if effectID == BeginAsura || effectID == BeginAsura11 {
		effect.asuraGlyphs = &asuraGlyphs{}
	}
	if effectID == Run {
		effect.taekwonRun = &taekwonRunTrail{position: EffectWorldAnchor(ctx, effect, starts)}
	}
	if effectID == Peong {
		effect.taekwonJump = &taekwonJumpParticles{tick: 35}
	}
	if persistent {
		effect.Persistent, effect.Expires = true, time.Time{}
	}
	m.PrepareEnvironmentEffect(&effect, scene.Projection{CameraYaw: m.cameraYaw(ctx)})
	m.Active = append(m.Active, effect)
	if effectEnabled(ctx, effectID) {
		m.ScheduleWorldEffectSound(starts, spec, effect)
		m.startWorldEffectCameraShake(starts, spec)
	}
	m.applyWorldEffectSideEffects(ctx, effect, starts)
	return true
}

func (m *System) AddCellLifetime(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, lifetimeOverride time.Duration, persistent bool) bool {
	if ctx.Config.Headless || ctx.World == nil || !persistent && !effectEnabled(ctx, effectID) {
		return false
	}
	spec, ok := Lookup(effectID)
	if !ok {
		return false
	}
	effect := Instance{
		EffectID:   effectID,
		ActorID:    actorID,
		X:          x,
		Y:          y,
		Starts:     starts,
		Persistent: persistent,
	}
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
		if componentDuration > duration {
			duration = componentDuration
		}
	}
	if duration <= 0 {
		duration = 500 * time.Millisecond
	}
	if lifetimeOverride > duration {
		duration = lifetimeOverride
	}
	expires := starts.Add(duration)
	if persistent && actorID != 0 {
		active := m.Active[:0]
		updated := false
		moved := false
		for _, effect := range m.Active {
			if effect.EffectID == effectID && effect.ActorID == actorID {
				if !effect.Expired(starts) {
					if !updated {
						moved = effect.X != x || effect.Y != y
						effect.X = x
						effect.Y = y
						if effect.Expires.Before(expires) {
							effect.Expires = expires
						}
						effect.Persistent = true
						active = append(active, effect)
						updated = true
					}
				}
				continue
			}
			active = append(active, effect)
		}
		m.Active = active
		if updated {
			return moved
		}
	}
	effect.Expires = expires
	m.PrepareEnvironmentEffect(&effect, scene.Projection{CameraYaw: m.cameraYaw(ctx)})
	m.Active = append(m.Active, effect)
	if effectEnabled(ctx, effectID) {
		m.ScheduleWorldEffectSound(starts, spec, effect)
		m.startWorldEffectCameraShake(starts, spec)
	}
	return true
}

func (m *System) AddCell(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64) bool {
	return m.addCellRotation(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, 0)
}

func (m *System) addCellRotation(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64, rotationRadiansPerSecond float64) bool {
	if ctx.Config.Headless || ctx.World == nil || !effectEnabled(ctx, effectID) {
		return false
	}
	spec, ok := Lookup(effectID)
	if !ok {
		return false
	}
	effect := Instance{
		EffectID:                             effectID,
		ActorID:                              actorID,
		X:                                    x,
		Y:                                    y,
		Starts:                               starts,
		Duration:                             durationOverride,
		Size:                                 sizeOverride,
		GroundSampleRotationRadiansPerSecond: rotationRadiansPerSecond,
	}
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
		if componentDuration > duration {
			duration = componentDuration
		}
	}
	if duration <= 0 {
		duration = 500 * time.Millisecond
	}
	if durationOverride > 0 {
		duration = durationOverride
	}
	effect.Expires = starts.Add(duration)
	m.PrepareEnvironmentEffect(&effect, scene.Projection{CameraYaw: m.cameraYaw(ctx)})
	m.Active = append(m.Active, effect)
	m.ScheduleWorldEffectSound(starts, spec, effect)
	m.startWorldEffectCameraShake(starts, spec)
	return true
}

func (m *System) startWorldEffectCameraShake(starts time.Time, spec Spec) {
	if spec.CameraShake <= 0 {
		return
	}
	m.startCameraShake(starts.Add(spec.CameraShakeDelay), spec.CameraShake)
}

func (m *System) ScheduleWorldEffectSound(starts time.Time, spec Spec, effect Instance) {
	if len(spec.Sfx) == 0 {
		return
	}
	// Quagmire has one effect per cell; the original only sounds one in twelve.
	if effect.EffectID == Quagmire && HitEffectRandom(effect, 0, 12) != 0 {
		return
	}
	if effect.EffectID == Kaen && HitEffectRandom(effect, 0, 8) != 0 {
		return
	}
	paths := resolveEffectSFX(spec, effect)
	for i, path := range paths {
		delay := time.Duration(0)
		if i < len(spec.SfxDelays) {
			delay = spec.SfxDelays[i]
		}
		if effect.EffectID == TripleAction || effect.EffectID == HolyCross || effect.EffectID == BottomGospel || effect.EffectID == GreedSound {
			m.scheduleSound(starts.Add(delay), path)
		} else if effect.EffectID == CartRevolution || effect.EffectID == HasteUp || effect.EffectID == RepairWeapon || effect.EffectID == MaximizePower {
			x, y := float64(effect.X), float64(effect.Y)
			if effect.HasActorTransform {
				x, y = effect.ActorOrigin.X-.5, effect.ActorOrigin.Z-.5
			}
			m.scheduleSoundVolumeAtPosition(starts.Add(delay), 1, x, y, path)
		} else {
			m.scheduleSoundAtWorldEffect(starts.Add(delay), effect, path)
		}
	}
}

func resolveEffectSFX(spec Spec, effect Instance) []string {
	out := make([]string, 0, len(spec.Sfx))
	for i, path := range spec.Sfx {
		if strings.Contains(path, "%d") && spec.SfxRandMax >= spec.SfxRandMin && spec.SfxRandMin > 0 {
			span := spec.SfxRandMax - spec.SfxRandMin + 1
			index := spec.SfxRandMin + int(deterministicUnit(effect, 701+i)*float64(span))
			if index > spec.SfxRandMax {
				index = spec.SfxRandMax
			}
			path = strings.Replace(path, "%d", strconv.Itoa(index), 1)
		}
		out = append(out, path)
	}
	return out
}

func (m *System) AddCellIfMissing(ctx client.Context, effectID int, x, y int, starts time.Time) bool {
	return m.addCellSizeIfMissing(ctx, effectID, 0, x, y, starts, 0, 0)
}

func (m *System) addCellSizeIfMissing(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64) bool {
	return m.AddCellRotationIfMissing(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, 0)
}

func (m *System) AddCellRotationIfMissing(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64, rotationRadiansPerSecond float64) bool {
	now := time.Now()
	for _, effect := range m.Active {
		if effect.EffectID == effectID && effect.ActorID == actorID && effect.X == x && effect.Y == y && !effect.Expired(now) {
			return false
		}
	}
	return m.addCellRotation(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, rotationRadiansPerSecond)
}

func (m *System) AddBetweenIfMissing(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time, durationOverride time.Duration) bool {
	now := time.Now()
	for _, effect := range m.Active {
		if effect.EffectID == effectID && effect.ActorID == actorID && effect.TargetID == targetID && !effect.Expired(now) {
			return false
		}
	}
	return m.AddBetweenDuration(ctx, effectID, actorID, targetID, starts, durationOverride)
}

func EffectActorDirection(ctx client.Context, actorID uint32, at time.Time) int {
	if ctx.World == nil {
		return 0
	}
	if actorID == 0 || scene.IsLocalActor(ctx, actorID) {
		return scene.ActorRenderDirection(ctx.World.Player, at)
	}
	return scene.ActorRenderDirection(ctx.World.Actors[actorID], at)
}

func EffectAnchor(ctx client.Context, actorID uint32) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	if actorID == 0 || scene.IsLocalActor(ctx, actorID) {
		return ctx.World.Player.X, ctx.World.Player.Y, true
	}
	if actor, ok := ctx.World.Actors[actorID]; ok {
		return actor.X, actor.Y, true
	}
	return 0, 0, false
}

func Lookup(effectID int) (Spec, bool) {
	if effectID == Emotion {
		return Spec{
			Components: []Component{{
				Kind:           ComponentSPR,
				SpriteFile:     "emotion",
				AttachedEntity: true,
				SpriteYOffset:  -100,
			}},
		}, true
	}
	spec, ok := db.EffectSpecs[effectID]
	if !ok {
		return Spec{}, false
	}
	return convertDBWorldEffectSpec(spec), true
}

func EffectDetachesLocalActor(effectID int) bool {
	spec, ok := Lookup(effectID)
	return ok && spec.DetachLocalActor
}

func convertDBWorldEffectSpec(spec db.EffectSpec) Spec {
	out := Spec{
		Duration:         spec.Duration,
		CameraShake:      spec.CameraShake,
		CameraShakeDelay: spec.CameraShakeDelay,
		DetachLocalActor: spec.DetachLocalActor,
		SfxRandMin:       spec.SFXRandMin,
		SfxRandMax:       spec.SFXRandMax,
	}
	if len(spec.SFX) > 0 {
		out.Sfx = append([]string(nil), spec.SFX...)
	}
	if len(spec.SFXDelays) > 0 {
		out.SfxDelays = append([]time.Duration(nil), spec.SFXDelays...)
	}
	if len(spec.Components) > 0 {
		out.Components = make([]Component, 0, len(spec.Components))
		for _, component := range spec.Components {
			out.Components = append(out.Components, convertDBWorldEffectComponent(component))
		}
	}
	return out
}

func convertDBWorldEffectComponent(component db.EffectComponent) Component {
	return Component{
		Kind:               convertDBEffectComponentKind(component.Kind),
		hitRing:            component.HitRing,
		hitParticles:       component.HitParticles,
		FuncAdapter:        effectFuncAdapterForName(component.FuncName),
		FuncName:           component.FuncName,
		Color:              component.Color,
		Duration:           component.Duration,
		durationRandMin:    component.DurationRandMin,
		durationRandMax:    component.DurationRandMax,
		Delay:              component.Delay,
		DuplicateDelay:     component.DuplicateDelay,
		delayOffsetDelta:   component.DelayOffsetDelta,
		Repeat:             component.Repeat,
		RepeatDelay:        component.RepeatDelay,
		StrFile:            component.STRFile,
		StrMinFile:         component.STRMinFile,
		StrRandMin:         component.STRRandMin,
		StrRandMax:         component.STRRandMax,
		AttachedEntity:     component.AttachedEntity,
		RenderBefore:       component.RenderBefore,
		TexturePath:        component.TexturePath,
		TextureName:        component.TextureName,
		TextureFile:        component.TextureFile,
		TextureFiles:       append([]string(nil), component.TextureFiles...),
		FrameDelay:         component.FrameDelay,
		SpriteFile:         component.SpriteFile,
		ShadowTexture:      component.ShadowTexture,
		SpriteHead:         component.SpriteHead,
		SpriteDirection:    component.SpriteDirection,
		SpriteRepeat:       component.SpriteRepeat,
		SpriteStopAtEnd:    component.SpriteStopAtEnd,
		spriteFrame:        component.SpriteFrame,
		SpriteDelay:        component.SpriteDelay,
		spriteXOffset:      component.SpriteXOffset,
		SpriteYOffset:      component.SpriteYOffset,
		fromSrc:            component.FromSrc,
		ToSrc:              component.ToSrc,
		Arc:                component.Arc,
		Retreat:            component.Retreat,
		AlphaMax:           component.AlphaMax,
		AlphaMaxDelta:      component.AlphaMaxDelta,
		Sparkling:          component.Sparkling,
		SparkNumber:        component.SparkNumber,
		Fade:               component.Fade,
		FadeIn:             component.FadeIn,
		FadeOut:            component.FadeOut,
		Rotate:             component.Rotate,
		RotateWithCamera:   component.RotateWithCamera,
		FixedPerspective:   component.FixedPerspective,
		RotateToTarget:     component.RotateToTarget,
		WorldSizedSprite:   component.WorldSizedSprite,
		Animation:          component.Animation,
		BottomSize:         component.BottomSize,
		TopSize:            component.TopSize,
		Height:             component.Height,
		PosX:               component.PosX,
		PosY:               component.PosY,
		PosZ:               component.PosZ,
		PosXEnd:            component.PosXEnd,
		PosYEnd:            component.PosYEnd,
		PosZEnd:            component.PosZEnd,
		PosXRand:           component.PosXRand,
		PosYRand:           component.PosYRand,
		PosZRand:           component.PosZRand,
		PosXStartRand:      component.PosXStartRand,
		PosYStartRand:      component.PosYStartRand,
		PosZStartRand:      component.PosZStartRand,
		posXStartMiddle:    component.PosXStartMiddle,
		posYStartMiddle:    component.PosYStartMiddle,
		PosZStartMiddle:    component.PosZStartMiddle,
		PosXEndRand:        component.PosXEndRand,
		PosYEndRand:        component.PosYEndRand,
		PosZEndRand:        component.PosZEndRand,
		posXEndMiddle:      component.PosXEndMiddle,
		posYEndMiddle:      component.PosYEndMiddle,
		PosZEndMiddle:      component.PosZEndMiddle,
		PosXSmooth:         component.PosXSmooth,
		PosYSmooth:         component.PosYSmooth,
		PosZSmooth:         component.PosZSmooth,
		SizeStart:          component.SizeStart,
		SizeEnd:            component.SizeEnd,
		SizeRand:           component.SizeRand,
		SizeStartX:         component.SizeStartX,
		SizeStartY:         component.SizeStartY,
		SizeEndX:           component.SizeEndX,
		SizeEndY:           component.SizeEndY,
		sizeStartXRandMin:  component.SizeStartXRandMin,
		sizeStartXRandMax:  component.SizeStartXRandMax,
		sizeStartYRandMin:  component.SizeStartYRandMin,
		sizeStartYRandMax:  component.SizeStartYRandMax,
		sizeEndXRandMin:    component.SizeEndXRandMin,
		sizeEndXRandMax:    component.SizeEndXRandMax,
		sizeEndYRandMin:    component.SizeEndYRandMin,
		sizeEndYRandMax:    component.SizeEndYRandMax,
		sizeRandX:          component.SizeRandX,
		SizeRandY:          component.SizeRandY,
		sizeRandXMiddle:    component.SizeRandXMiddle,
		SizeRandYMiddle:    component.SizeRandYMiddle,
		SizeDelta:          component.SizeDelta,
		SizeSmooth:         component.SizeSmooth,
		AngleStart:         component.AngleStart,
		AngleEnd:           component.AngleEnd,
		AngleX:             component.AngleX,
		AngleY:             component.AngleY,
		angleZ:             component.AngleZ,
		angleRandMin:       component.AngleRandMin,
		angleRandMax:       component.AngleRandMax,
		circlePattern:      component.CirclePattern,
		circleInnerSize:    component.CircleInnerSize,
		circleOuterRandMin: component.CircleOuterRandMin,
		circleOuterRandMax: component.CircleOuterRandMax,
		OrbitRadiusX:       component.OrbitRadiusX,
		OrbitRadiusY:       component.OrbitRadiusY,
		orbitRadiusZ:       component.OrbitRadiusZ,
		OrbitRotations:     component.OrbitRotations,
		OrbitPhase:         component.OrbitPhase,
		orbitPhaseDelta:    component.OrbitPhaseDelta,
		OrbitClockwise:     component.OrbitClockwise,
		TotalCircleSides:   component.TotalCircleSides,
		CircleSides:        component.CircleSides,
		Duplicate:          component.Duplicate,
		AngleZRandom:       component.AngleZRandom,
		BlendMode:          component.BlendMode,
		BlendAdditive:      component.BlendAdditive,
		Overlay:            component.Overlay,
		QuadHornHeightMin:  component.QuadHornHeightMin,
		QuadHornHeightMax:  component.QuadHornHeightMax,
		QuadHornOffsetXMin: component.QuadHornOffsetXMin,
		QuadHornOffsetXMax: component.QuadHornOffsetXMax,
		QuadHornOffsetYMin: component.QuadHornOffsetYMin,
		QuadHornOffsetYMax: component.QuadHornOffsetYMax,
		QuadHornOffsetZ:    component.QuadHornOffsetZ,
		QuadHornBottomMin:  component.QuadHornBottomMin,
		QuadHornBottomMax:  component.QuadHornBottomMax,
		QuadHornRotateXMin: component.QuadHornRotateXMin,
		QuadHornRotateXMax: component.QuadHornRotateXMax,
		QuadHornRotateYMin: component.QuadHornRotateYMin,
		QuadHornRotateYMax: component.QuadHornRotateYMax,
		QuadHornRotateZMin: component.QuadHornRotateZMin,
		QuadHornRotateZMax: component.QuadHornRotateZMax,
		QuadHornAnimSpeed:  component.QuadHornAnimSpeed,
		QuadHornAnimOut:    component.QuadHornAnimOut,
	}
}

func convertDBEffectComponentKind(kind db.EffectComponentKind) ComponentKind {
	switch kind {
	case db.EffectComponentSTR:
		return ComponentSTR
	case db.EffectComponentCylinder:
		return ComponentCylinder
	case db.EffectComponent2D:
		return Component2D
	case db.EffectComponent3D:
		return Component3D
	case db.EffectComponentSPR:
		return ComponentSPR
	case db.EffectComponentFUNC:
		return ComponentFUNC
	case db.EffectComponentQuadHorn:
		return ComponentQuadHorn
	default:
		return 0
	}
}

func (m *System) Draw(screen *render.Frame, ctx client.Context, projection scene.Projection, now time.Time, beforeEntities bool) {
	if len(m.Active) == 0 || screen == nil || ctx.World == nil {
		return
	}
	if m.whitePixel == nil {
		m.whitePixel = render.NewImage(1, 1)
		m.whitePixel.Fill(color.White)
	}
	active := m.Active[:0]
	for _, effect := range m.Active {
		if effect.Expired(now) {
			continue
		}
		spec, ok := m.specs[effect.EffectID]
		if !ok {
			spec, ok = Lookup(effect.EffectID)
			if !ok {
				continue
			}
			if m.specs == nil {
				m.specs = make(map[int]Spec)
			}
			m.specs[effect.EffectID] = spec
		}
		if now.Before(effect.Starts) {
			active = append(active, effect)
			continue
		}
		if !effect.HasActorTransform {
			effect.ActorDirection = EffectActorDirection(ctx, effect.ActorID, effect.Starts)
			effect.ActorOrigin = EffectWorldAnchor(ctx, effect, effect.Starts)
			effect.HasActorTransform = true
		}
		if effect.pressure != nil {
			effect.pressure.advance(ctx, effect, now)
		}
		if effect.arrow != nil {
			effect.arrow.release(ctx, effect)
		}
		if effect.acidBottles != nil {
			effect.acidBottles.advance(ctx, effect, spec, now)
		}
		if effect.Shield != nil {
			effect.Shield.advance(ctx, effect, now)
		}
		if effect.ShieldChain != nil {
			effect.ShieldChain.advance(ctx, effect, now)
		}
		if effect.arrowVulcan != nil {
			effect.arrowVulcan.advance(ctx, effect, now)
		}
		active = append(active, effect)
		// Keep lifetimes and actor state while hidden so toggling back on cannot
		// resurrect expired effects or lose a still-active server status.
		if !effectEnabled(ctx, effect.EffectID) {
			continue
		}
		anchor := EffectWorldAnchor(ctx, effect, now)
		if effect.EffectID == db.SkillEffectChaseWalkSmoke {
			anchor = effect.ActorOrigin
		}
		for index, component := range spec.Components {
			if component.RenderBefore != beforeEntities {
				continue
			}
			// These persistent effects play their startup STR only once.
			if effect.Persistent && effect.EffectID != LKConcentration && effect.EffectID != SafetyWall {
				component.Repeat = true
			}
			componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component, effect)
			if effect.Duration > 0 && !component.Repeat {
				componentDuration = effect.Duration
			}
			progress := worldEffectComponentProgressForDraw(effect.Starts, component, componentDuration, now)
			if progress >= 1 {
				continue
			}
			m.DrawWorldEffectComponent(screen, ctx, projection, effect, component, index, anchor.X, anchor.Z, anchor.Y, progress, componentDuration, now)
		}
	}
	m.Active = active
}

func EffectWorldAnchor(ctx client.Context, effect Instance, at time.Time) scene.Point3 {
	x, y := float64(effect.X), float64(effect.Y)
	if actor, ok := ctx.World.Actors[effect.ActorID]; ok {
		x, y = scene.ActorRenderPosition(actor, at)
	} else if scene.IsLocalActor(ctx, effect.ActorID) {
		x, y = scene.ActorRenderPosition(ctx.World.Player, at)
	}
	return scene.Point3{X: scene.CellCenter(x), Y: scene.TerrainHeightAt(ctx.World, x, y) + 0.07, Z: scene.CellCenter(y)}
}

func (m *System) WorldEffectResolvedComponentDuration(ctx client.Context, spec Spec, component Component, effect Instance) time.Duration {
	duration := WorldEffectComponentDuration(spec, component)
	if component.Kind == ComponentSTR {
		if str := m.LoadWorldEffectSTR(ctx.Resources, ResolveEffectSTRFile(component, effect, lessEffectsEnabled(ctx)), component.TexturePath); str != nil {
			duration = strEffectDuration(str, duration)
		}
	}
	if component.Kind == ComponentSPR && component.Duration <= 0 && !component.SpriteRepeat {
		if view := m.EffectSpriteView(ctx.Resources, component.SpriteFile); view != nil && len(view.Act.Actions) > 0 {
			actionIndex := component.spriteFrame
			if actionIndex < 0 || actionIndex >= len(view.Act.Actions) {
				actionIndex = 0
			}
			action := view.Act.Actions[actionIndex]
			if component.SpriteDelay > 0 && len(action.Animations) > 0 {
				duration = component.SpriteDelay * time.Duration(len(action.Animations))
			} else {
				duration = scene.ActionAnimationDuration(action, duration)
			}
		}
	}
	return duration
}

func lessEffectsEnabled(ctx client.Context) bool {
	if ctx.Session != nil {
		return ctx.Session.LessEffects
	}
	return ctx.Config.Gameplay.LessEffects
}

func WorldEffectComponentDuration(spec Spec, component Component) time.Duration {
	duration := spec.Duration
	if component.Duration > 0 {
		duration = component.Duration
	}
	if component.durationRandMax > 0 {
		duration = component.durationRandMax
		if duration < component.durationRandMin {
			duration = component.durationRandMin
		}
	}
	duration += worldEffectComponentMaxStartOffset(component)
	return duration
}

func worldEffectComponentStartOffset(component Component, duplicateIndex int) time.Duration {
	return component.Delay + worldEffectComponentDuplicateStartOffset(component, duplicateIndex)
}

func worldEffectComponentDuplicateStartOffset(component Component, duplicateIndex int) time.Duration {
	if duplicateIndex < 0 {
		duplicateIndex = 0
	}
	return time.Duration(duplicateIndex) * (component.DuplicateDelay + component.delayOffsetDelta)
}

func worldEffectComponentMaxStartOffset(component Component) time.Duration {
	duplicates := scene.MaxInt(component.Duplicate, 1)
	maxOffset := time.Duration(0)
	for i := 0; i < duplicates; i++ {
		offset := worldEffectComponentStartOffset(component, i)
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	return maxOffset
}

func WorldEffectComponentProgress(starts time.Time, duration time.Duration, now time.Time) float64 {
	if duration <= 0 {
		return 1
	}
	return scene.ClampFloat(float64(now.Sub(starts))/float64(duration), 0, 1)
}

func worldEffectComponentProgressForDraw(starts time.Time, component Component, duration time.Duration, now time.Time) float64 {
	if !component.Repeat {
		return WorldEffectComponentProgress(starts, duration, now)
	}
	if duration <= 0 {
		return 1
	}
	componentStart := starts.Add(component.Delay)
	if now.Before(componentStart) {
		return 1
	}
	cycle := duration + component.RepeatDelay
	if cycle <= 0 {
		cycle = duration
	}
	cycleElapsed := now.Sub(componentStart) % cycle
	if cycleElapsed >= duration {
		return 1
	}
	return scene.ClampFloat(float64(cycleElapsed)/float64(duration), 0, 1)
}

func WorldEffectComponentDuplicateProgressForDraw(starts time.Time, component Component, duplicateIndex int, duration time.Duration, now time.Time) (float64, bool) {
	starts = starts.Add(worldEffectComponentDuplicateStartOffset(component, duplicateIndex))
	if component.Repeat {
		progress := worldEffectComponentProgressForDraw(starts, component, duration, now)
		return progress, progress < 1
	}
	starts = starts.Add(component.Delay)
	if now.Before(starts) {
		return 0, false
	}
	progress := WorldEffectComponentProgress(starts, duration, now)
	return progress, progress < 1
}

func (m *System) DrawWorldEffectComponent(screen *render.Frame, ctx client.Context, projection scene.Projection, effect Instance, component Component, componentIndex int, worldX, worldY, worldZ, progress float64, componentDuration time.Duration, now time.Time) {
	if !effectEnabled(ctx, effect.EffectID) {
		return
	}
	switch component.Kind {
	case ComponentSTR:
		m.drawSTREffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case ComponentCylinder:
		m.drawCylinderEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, componentDuration, now)
	case Component2D:
		m.draw2DEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, now)
	case Component3D:
		m.draw3DEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, now)
	case ComponentSPR:
		m.DrawSPREffect(screen, ctx, projection, effect, component, worldX, worldY, worldZ, now)
	case ComponentFUNC:
		m.DrawFuncEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, now)
	case ComponentQuadHorn:
		m.drawQuadHornEffect(screen, ctx, effect, component, componentIndex, worldX, worldY, worldZ, progress, componentDuration)
	default:
	}
}

func effectComponentAlpha(progress float64, component Component) float64 {
	alphaMax := component.AlphaMax
	if alphaMax <= 0 {
		alphaMax = 1
	}
	if component.Fade {
		switch {
		case progress < 0.25:
			return progress / 0.25 * alphaMax
		case progress > 0.75:
			return (1 - progress) / 0.25 * alphaMax
		default:
			return alphaMax
		}
	}
	switch {
	case component.FadeIn && progress < 0.25:
		return progress / 0.25 * alphaMax
	case component.FadeOut && progress > 0.75:
		return (1 - progress) / 0.25 * alphaMax
	default:
		return alphaMax
	}
}

func effectComponentTint(component Component, alpha float64) color.RGBA {
	tint := component.Color
	if tint.A == 0 {
		tint = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	tint.A = uint8(scene.ClampFloat(alpha, 0, 1) * 255)
	return tint
}

func deterministicAngle(effect Instance, salt int) float64 {
	return deterministicUnit(effect, salt) * 2 * math.Pi
}

func deterministicSigned(effect Instance, salt int) float64 {
	return deterministicUnit(effect, salt)*2 - 1
}

func deterministicDurationRange(effect Instance, salt int, min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(deterministicUnit(effect, salt)*float64(max-min))
}

func deterministicFloatRange(effect Instance, salt int, min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + deterministicUnit(effect, salt)*(max-min)
}

func deterministicUnit(effect Instance, salt int) float64 {
	value := uint32(effect.EffectID*1103515245) ^ effect.ActorID ^ uint32(effect.Starts.UnixNano()) ^ uint32(salt*2654435761) ^ effect.randomSeed
	value ^= value >> 16
	value *= 2246822519
	value ^= value >> 13
	return float64(value&0xFFFFFF) / float64(0xFFFFFF)
}

func (m *System) EffectFileTexture(manager *res.Manager, path string) *render.Image {
	return m.EffectFileTextureWithBlackKey(manager, path, true)
}

func (m *System) EffectFileTextureWithBlackKey(manager *res.Manager, path string, blackKey bool) *render.Image {
	path = strings.TrimSpace(path)
	if manager == nil || path == "" {
		return nil
	}
	key := "__effectfile_" + path + "|blackkey=" + strconv.FormatBool(blackKey)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	normalized := strings.ReplaceAll(path, "/", "\\")
	candidates := []string{
		"data\\texture\\" + normalized,
		strings.ReplaceAll("data\\texture\\"+normalized, "\\", "/"),
		normalized,
		strings.ReplaceAll(normalized, "\\", "/"),
	}
	img, _, err := res.LoadImageExact(manager, candidates)
	if err != nil {
		m.textureMiss[key] = struct{}{}
		glog.Warnf("effect texture missing path=%s: %v", path, err)
		return nil
	}
	out := res.ApplyEffectTransparencyWithBlackKey(img, blackKey)
	texture := render.NewImageFromImage(out)
	if !blackKey {
		texture = render.NewImageFromStraightAlpha(out)
	}
	m.textures[key] = texture
	return texture
}

func (m *System) effectTexture(manager *res.Manager, name string) *render.Image {
	if manager == nil || strings.TrimSpace(name) == "" {
		return nil
	}
	key := "__effect_" + strings.TrimSpace(name)
	if m.textures == nil {
		m.textures = make(map[string]*render.Image)
	}
	if m.textureMiss == nil {
		m.textureMiss = make(map[string]struct{})
	}
	if texture, ok := m.textures[key]; ok {
		return texture
	}
	if _, ok := m.textureMiss[key]; ok {
		return nil
	}
	img, _, err := res.LoadImage(manager, res.EffectTextureCandidates(name))
	if err != nil {
		m.textureMiss[key] = struct{}{}
		return nil
	}
	texture := render.NewImageFromImage(res.ApplyEffectTransparency(img))
	m.textures[key] = texture
	return texture
}
