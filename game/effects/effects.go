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
	Hit1              = 0
	BashHit           = 1
	Hit3              = 2
	Hit4              = 3
	Hit5              = 4
	Hit6              = 5
	Entry             = 6
	Exit              = 7
	Warp              = 8
	Enhance           = 9
	Mammonite         = 10
	Endure            = 11
	BeginSpell        = 12
	GlassWall         = 13
	HealSP            = 14
	SoulStrike        = 15
	BashBegin         = 16
	MagnumBreak       = 17
	Steal             = 18
	PoisonAttack      = 20
	Detoxication      = 21
	Sight             = 22
	StoneCurse        = 23
	FireBall          = 24
	FireWall          = 25
	IceArrow          = 26
	FrostDiver        = 27
	FrostDiverHit     = 28
	LightningBolt     = 29
	ThunderStorm      = 30
	FireArrow         = 31
	NapalmBeat        = 32
	Ruwach            = 33
	TeleportOld       = 34
	ReadyPortalOld    = 35
	IncAgility        = 37
	DecAgility        = 38
	Aqua              = 39
	Signum            = 40
	Angelus           = 41
	Blessing          = 42
	IncAgiDex         = 43
	Smoke             = 44
	Firefly           = 45
	Torch             = 47
	FireHit           = 49
	FireSplashHit     = 50
	ColdHit           = 51
	WindHit           = 52
	PoisonHit         = 53
	BeginSpell2       = 54
	BeginSpell3       = 55
	BeginSpell4       = 56
	BeginSpell5       = 57
	BeginSpell6       = 58
	BeginSpell7       = 59
	LockOnTarget      = 60
	WarpZone          = 61
	SightTrasher      = 62
	ArrowShotRO       = 64
	Invenom           = 65
	Cure              = 66
	Provoke           = 67
	Mvp               = 68
	SkidTrap          = 69
	BrandishSpear     = 70
	BowlingBash       = 73
	IceWall           = 74
	Gloria            = 75
	Magnificat        = 76
	Resurrection      = 77
	Recovery          = 78
	EarthSpike        = 79
	SpearBoomerang    = 80
	Pierce            = 81
	TurnUndead        = 82
	Sanctuary         = 83
	Impositio         = 84
	LexAeterna        = 85
	Aspersio          = 86
	LexDivina         = 87
	Suffragium        = 88
	StormGust         = 89
	LordVermilion     = 90
	Benedictio        = 91
	MeteorStorm       = 92
	JupitelThunder    = 93
	JupitelHit        = 94
	Quagmire          = 95
	FirePillar        = 96
	FirePillarBomb    = 97
	HasteUp           = 98
	Flasher           = 99
	RemoveTrap        = 100
	RepairWeapon      = 101
	CrashEarth        = 102
	WeaponPerfect     = 103
	MaximizePower     = 104
	BlastMine         = 105
	BlastMineBomb     = 106
	Claymore          = 107
	FreezingTrap      = 108
	Bubble            = 109
	GasPush           = 110
	SpringTrap        = 111
	Kyrie             = 112
	Magnus            = 113
	BlitzBeat         = 115
	WaterBall         = 116
	WaterBall2        = 117
	Detecting         = 119
	Cloaking          = 120
	SonicBlow         = 121
	SonicBlowHit      = 122
	Grimtooth         = 123
	VenomDust         = 124
	EnchantPoison     = 125
	PoisonReact       = 126
	PoisonReact2      = 127
	Overthrust        = 128
	VenomSplasher     = 129
	TwoHandQuicken    = 130
	AutoCounter       = 131
	GrimtoothAtk      = 132
	Freeze            = 133
	Freezed           = 134
	IceCrash          = 135
	SlowPoison        = 136
	FirePillarOn      = 138
	Sandman           = 139
	Revive            = 140
	Pneuma            = 141
	HeavenDrive       = 142
	SonicBlow2        = 143
	BrandishSpear2    = 144
	Shockwave         = 145
	ShockwaveHit      = 146
	EarthHit          = 147
	PierceSelf        = 148
	BowlingSelf       = 149
	SpearStabSelf     = 150
	SpearBmrSelf      = 151
	HolyLight         = 152
	Concentration     = 153
	RefineOK          = 154
	RefineFail        = 155
	JobLevelUp        = 158
	Rain              = 161
	Snow              = 162
	Sakura            = 163
	Banjjakii         = 165
	MakeBlur          = 166
	EnergyCoat        = 169
	CartRevolution    = 170
	VenomDust2        = 171
	MentalBreak       = 181
	MagicalAtkHit     = 182
	SuiExplosion      = 183
	Suicide           = 185
	ComboAttack1      = 186
	ComboAttack2      = 187
	ComboAttack3      = 188
	ComboAttack4      = 189
	ComboAttack5      = 190
	GuidedAttack      = 191
	PoisonAttack2     = 192
	SilenceAttack     = 193
	StunAttack        = 194
	PetrifyAttack     = 195
	SleepAttack       = 197
	Pong              = 199
	Level99           = 200
	Level99Ground     = 201
	Level99Bubble     = 202
	Gumgang           = 203
	PotionRed         = 204
	PotionOrange      = 205
	PotionYellow      = 206
	PotionWhite       = 207
	PotionBlue        = 208
	PotionGreen       = 209
	Food              = 210
	FoodBlue          = 211
	DarkBreath        = 212
	Defender          = 213
	Keeping           = 214
	SummonSlave       = 215
	BloodDrain        = 216
	EnergyDrain       = 217
	ItemFast          = 218
	ItemFast2         = 219
	ItemFast3         = 220
	CrusaderDef       = 222
	Ganbantein        = 223
	Volcano           = 225
	GrandCross        = 226
	Intimidate        = 227
	Chookgi           = 228
	Cloud             = 229
	Cloud2            = 230
	MapPillar         = 231
	LineLink          = 232
	Cloud3            = 233
	SpellBreaker      = 234
	Dispell           = 235
	Deluge            = 236
	ViolentGale       = 237
	LandProtector     = 238
	BottomVolcano     = 239
	BottomDeluge      = 240
	BottomViolent     = 241
	BottomLand        = 242
	MagicRod          = 244
	HolyCross         = 245
	ShieldCharge      = 246
	MapPillar2        = 247
	Providence        = 248
	ShieldBoomer      = 249
	SpearQuicken      = 250
	Devotion          = 251
	ReflectShield     = 252
	AbsorbSpirits     = 253
	SteelBody         = 254
	FlameLauncher     = 255
	FrostWeapon       = 256
	LightningLoad     = 257
	SeismicWeapon     = 258
	MapPillar3        = 259
	MapPillar4        = 260
	Gumgang2          = 261
	TeiHit1           = 262
	Gumgang3          = 263
	TeiHit2           = 264
	Tanji             = 265
	TeiHit1X          = 266
	Chimto            = 267
	StealCoin         = 268
	StripWeapon       = 269
	StripShield       = 270
	StripArmor        = 271
	StripHelm         = 272
	ChainCombo        = 273
	RogueCoin         = 274
	BackStab          = 275
	TeiHit3           = 276
	BottomLullaby     = 278
	BottomRichKim     = 279
	BottomChaos       = 280
	BottomDrum        = 281
	BottomNibelung    = 282
	BottomRoki        = 283
	BottomAbyss       = 284
	BottomSieg        = 285
	BottomWhistle     = 286
	BottomSinX        = 287
	BottomBragi       = 288
	BottomApple       = 289
	BottomHumming     = 291
	BottomForget      = 292
	BottomFortune     = 293
	BottomService     = 294
	TalkFrostJoke     = 295
	TalkScream        = 296
	PokJuk            = 297
	ThrowItem         = 298
	ThrowItem2        = 299
	ChemicalProt      = 300
	Demonstration     = 302
	Chemical2         = 303
	Teleportation     = 304
	PharmacyOK        = 305
	PharmacyFail      = 306
	ThrowItem3        = 308
	FirstAid          = 309
	SprinkleSand      = 310
	Loud              = 311
	Heal              = 312
	Heal2             = 313
	Exit2             = 314
	SafetyWall        = 315
	ReadyPortal       = 316
	Portal            = 317
	BottomMagnus      = 318
	BottomSanc        = 319
	HealOffensive     = 320
	WarpZone2         = 321
	Heal4             = 325
	BeginAsura        = 328
	TripleAttack      = 329
	HitLine           = 330
	HPTime            = 331
	SPTime            = 332
	Maple             = 333
	Blind             = 334
	PoisonStatus      = 335
	Guard             = 336
	JobLvUp50         = 337
	Magnum2           = 339
	CoupleCasting     = 342
	Entry2            = 344
	ColorPaper        = 347
	SoulBreaker       = 361
	Level99Aura1      = 362
	FoodChocolate     = 363
	Pressure          = 365
	Bash3D            = 366
	AuraBlade         = 367
	RedBody           = 368
	LKConcentration   = 369
	BottomGospel      = 370
	BaseLevelUp       = 371
	Death             = 372
	DragonSmoke       = 373
	BottomBasilica    = 374
	Assumptio         = 375
	HitLine2          = 376
	Bash3D2           = 377
	EnergyDrain2      = 378
	TransBlueBody     = 379
	MagicCrasher      = 380
	LightBlade        = 382
	EnergyDrain3      = 383
	LineLink2         = 384
	LinkLight         = 385
	TrueSight         = 386
	FalconAssault     = 387
	TripleAttack2     = 388
	Portal4           = 389
	Meltdown          = 390
	CartBoost         = 391
	RejectSword       = 392
	TripleAttack3     = 393
	Moonlit           = 394
	MarionetteLink    = 395
	PinkBody          = 396
	Level99AuraMid    = 397
	Level99AuraBottom = 398
	Bash3D3           = 399
	Bash3D4           = 400
	NapalmVulcan      = 401
	Portal5           = 402
	MagicCrasher2     = 403
	BottomSpider      = 404
	FogWallGround     = 405
	SoulBurn          = 406
	SoulChange        = 407
	SoulBreaker2      = 409
	Rainbow           = 410
	Peong             = 411
	PressedBody       = 413
	SpinedBody        = 414
	KickedBody        = 415
	DoubleGumgang     = 418
	ReflectBody       = 419
	BabyBody          = 420
	BabyBody2         = 421
	GiantBody         = 422
	GiantBody2        = 423
	AsuraBody         = 424
	FourWayBody       = 425
	QuakeBody         = 426
	AsuraBodyMonster  = 427
	HitLine4          = 429
	HitLine5          = 430
	HitLine6          = 431
	StormKick         = 435
	Chemical3         = 439
	Assumptio2        = 440
	BlueCasting       = 441
	Run               = 442
	StopRun           = 443
	StopEffect        = 444
	JumpBody          = 445
	LandBody          = 446
	Foot3             = 447
	Foot4             = 448
	TaeReady          = 449
	DarkGrandCross    = 450
	DarkSoulStrike    = 451
	DarkJupitelHit    = 452
	NPCStop           = 453
	DarkCasting       = 454
	AgiUp             = 456
	JumpKick          = 457
	QuakeBody2        = 458
	StormKick3        = 461
	StormKick4        = 462
	SpinedBody2       = 466
	BeginAsura1       = 467
	BeginAsura2       = 468
	BeginAsura3       = 469
	BeginAsura4       = 470
	BeginAsura5       = 471
	BeginAsura6       = 472
	BeginAsura7       = 473
	Devil1            = 475
	Devil2            = 476
	Devil3            = 477
	Devil4            = 478
	Devil5            = 479
	Devil6            = 480
	Devil7            = 481
	Devil8            = 482
	Devil9            = 483
	Devil10           = 484
	BlackDevil        = 487
	FlowerCast        = 488
	ResistPotion      = 491
	Mochi             = ResistPotion
	Ramadan           = 492
	EDP               = 493
	FullStrip         = 495
	Preserve          = 496
	Slim              = 497
	Slim2             = 498
	Slim3             = 499
	ChemicalBody      = 500
	CastSpin          = 501
	PierceBody        = 502
	SoulLink          = 503
	Chookgi2          = 504
	Memorize          = 505
	ItemAccel         = 507
	Mapae             = ItemAccel
	Firecracker       = 508
	ItemPokJuk        = Firecracker
	Valentine05       = 509
	BeginAsura11      = 510
	Chemical2Dash     = 512
	GroundSample      = 513
	Cloud4            = 515
	Cloud5            = 516
	BottomHermode     = 517
	CartTermination   = 518
	ItemSlow          = 519
	ItemFastDown      = ItemSlow
	ShieldBoomerang3  = 520
	DoubleCastBody    = 521
	Gravitation       = 522
	TarotCard1        = 523
	TarotCard2        = 524
	TarotCard3        = 525
	TarotCard4        = 526
	TarotCard5        = 527
	TarotCard6        = 528
	TarotCard7        = 529
	TarotCard8        = 530
	TarotCard9        = 531
	TarotCard10       = 532
	TarotCard11       = 533
	TarotCard12       = 534
	TarotCard13       = 535
	TarotCard14       = 536
	AcidDemon         = 537
	ThrowItem4        = 539
	BabyBodyBack      = 540
	ThrowItem5        = 541
	BlueBody          = 542
	Hated             = 543
	RedLightBody      = 544
	SmaReady          = 546
	Stin              = 547
	RedHit            = 548
	BlueHit           = 549
	QuakeBody3        = 550
	Stin2             = 553
	HitTexture        = 554
	Stin3             = 555
	ScreenQuake       = 563
	HfliMoon1         = 565
	HfliMoon2         = 566
	HfliMoon3         = 567
	HoUp              = 568
	HamiDefence       = 569
	HamiCastle        = 570
	HamiBlood         = 571
	Hated2            = 572
	Twilight1         = 573
	Twilight2         = 574
	Twilight3         = 575
	BoxThunder        = 576
	ItemThunder       = BoxThunder
	BoxResentment     = 577
	ItemCloud         = BoxResentment
	ItemCurse         = 578
	BoxDrowsiness     = 579
	ItemZZZ           = BoxDrowsiness
	BoxSunlight       = 580
	ItemRain          = BoxSunlight
	M01               = 583
	M02               = 584
	M03               = 585
	M04               = 586
	M05               = 587
	M06               = 588
	M07               = 589
	Kaizel            = 590
	Cloud6            = 592
	StatFoodSTR       = 593
	StatFoodINT       = 594
	StatFoodVIT       = 595
	StatFoodAGI       = 596
	StatFoodDEX       = 597
	StatFoodLUK       = 598
	Shrink            = 599
	ThrowItem6        = 600
	SightBlaster      = 601
	QuakeBody4        = 602
	FireHit2          = 603
	NPCStop2          = 604
	NPCStop2Del       = 605
	FVoice            = 606
	Wink              = 607
	CookingOK         = 608
	CookingFail       = 609
	Firecracker1      = 612
	Hapgyeok          = 612
	ThrowItem7        = 613
	ThrowItem8        = 614
	ThrowItem9        = 615
	ThrowItem10       = 616
	Kouenka           = 618
	Hyousensou        = 619
	BottomSuiton      = 620
	Stin4             = 621
	ThunderStorm2     = 622
	Chemical4         = 623
	Stin5             = 624
	MadnessBlue       = 625
	MadnessRed        = 626
	RGCoin3           = 627
	Bash3D5           = 628
	Chookgi3          = 629
	Kirikage          = 630
	Tatami            = 631
	Kasumikiri        = 632
	Issen             = 633
	Kaen              = 634
	Baku              = 635
	Hyousyouraku      = 636
	Desperado         = 637
	LightningS        = 638
	BlindS            = 639
	PoisonS           = 640
	FreezingS         = 641
	FlareS            = 642
	RapidShower       = 643
	MagicalBullet     = 644
	SpreadAttack      = 645
	TrackCasting      = 646
	Tracking          = 647
	TripleAction      = 648
	Bullseye          = 649
	BubbleDrop        = 665
	NPCEarthquake     = 666
	DragonFear        = 668
	WideBleeding      = 669
	WideConfuse       = 670
	BottomRunner      = 671
	BottomTransfer    = 672
	BottomEvilLand    = 674
	Guard3            = 675
	CriticalWound     = 677
	Firecracker2      = 682
	Firecracker3      = 683
	Firecracker4      = 684
	Firecracker5      = 685
	Firecracker6      = 686
	TorchRed          = 690
	TorchGreen        = 691
	MapGhost          = 692
	Glow1             = 693
	Glow2             = 694
	Glow4             = 695
	TorchPurple       = 696
	Cloud7            = 697
	Cloud8            = 698
	FlowerLeaf        = 699
	Item315           = 704
	Item316           = 705
	Item317           = 706
	StormMin          = 708
	Firecracker7      = 709
	BottomBlue        = 715
	BottomBlue2       = 716
	ChristmasCarol    = 717
	FirePillarOn2     = 718
	ForestLight5      = 719
	Adoramus          = 721
	IgnitionBreak     = 722
	FrostMisty        = 726
	CrimsonRock       = 727
	HellInferno       = 728
	MarshOfAbyss      = 729
	DragonHowling     = 731
	EarthWall         = 732
	ChainLightning    = 734
	AimedBolt         = 745
	ArrowStorm        = 746
	Laulamus          = 747
	Lauagnus          = 748
	MillenniumShield  = 749
	Concentration2    = 750
	GlassWall3        = 754
	BerserkPotion2    = 756
	Rolling1          = 758
	Rolling2          = 759
	Rolling3          = 760
	Rolling4          = 761
	Rolling5          = 762
	Rolling6          = 763
	Rolling7          = 764
	Rolling8          = 765
	Rolling9          = 766
	Rolling10         = 767
	CastSpin2         = 775
	CrashAxe          = 795
	Stasis            = 799
	BottomBasilica2   = 802
	Recognized        = 803
	Tetra             = 804
	TetraCasting      = 805
	Stretch           = 811
	Enervation        = 813
	Enervation2       = 814
	Enervation3       = 815
	Enervation4       = 816
	Enervation5       = 817
	Enervation6       = 818
	BottomManhole     = 822
	Manhole           = 823
	ForestLight6      = 825
	BottomAni         = 827
	BottomMaelstrom   = 828
	BottomBloodyLust  = 829
	HealN             = 832
	ChookgiN          = 833
	Dance1            = 847
	BotReverb         = 856
	RainParticle      = 857
	ChemicalV2        = 858
	BotReverb2        = 860
	CirclePower2      = 861
	Secra2            = 862
	SprPlant2         = 866
	SprPlant3         = 868
	SprPlant4         = 870
	SprPlant5         = 872
	SprPlant6         = 874
	SprPlant7         = 876
	SprPlant8         = 878
	HeartAsura        = 879
	GlassWall4        = 883
	Bash3D6           = 885
	Electric4         = 888
	TeiHit1T          = 889
	Pressure2         = 906
	PrimeCharge2      = 908
	PrimeCharge3      = 909
	PrimeCharge4      = 910
	FireWall2         = 920
	SprPlant10        = 922
	Shockwave2        = 926
	ColdThrow2        = 928
	DemonicFire4      = 929
	Pressure3         = 930
	PoisonMist        = 959
	EraserCutter      = 960
	LavaSlide         = 964
	SonicClaw         = 965
	TinderBreaker     = 966
	MidnightFrenzy    = 967
	VolcanicAsh       = 975
	RWC2011           = 987
	RWC2011Two        = 988
	RunMakeOK         = 1015
	RunMakeFailure    = 1016
	MIResultMakeOK    = 1017
	MIResultMakeFail  = 1018
	AllRayProtect     = 1019
	VenomFog          = 1020
	DustStorm         = 1021
	DanceBladeAtk     = 1029
	InvincibleOff2    = 1031
	DeathSummon       = 1033
	GCDarkCrow        = 1040
	AllFullThrottle   = 1042
	SRFlashCombo      = 1043
	RKLuxAnima        = 1044
	SOElemShield      = 1046
	ABOffertorium     = 1047
	WLTelekinesis     = 1048
	GNIllusionDoping  = 1049
	NCMagmaEruption   = 1050
	NPCChill          = 1055
	OffertoriumRing   = 1057
	HammerOfGod       = 1062
	AchComplete       = 1094
	BodyColor         = 1111
	BakuretsuHadou    = 1130
	DropPink          = 1186
	DropYellow        = 1189
	DropPurple        = 1190
	DigitalSpace      = 1240
	DropBlue          = 1869
	DropGreen         = 1870
	DropRed           = 1871
	NewSuccess        = 1872
	NewFailure        = 1873
	NewIntro          = 1874
	EnchantYellow     = 1875
	EnchantSuccess    = 1876
	EnchantFail       = 1877
	EnchantBlue       = 1878
	EnchantUpSuccess  = 1879
	EnchantUpFail     = 1880
	EnchantGreen      = 1881
	EnchantResetOK    = 1882
	EnchantResetFail  = 1883
)

// Goro-local effect IDs.
const (
	ColdBolt               = 10014
	FireBolt               = 10019
	CastRing               = 10021
	QuakeMagnum            = 10022
	ArrowShot              = 10060
	ArrowShower            = 10061
	MagicPower             = db.SkillEffectMagicPower
	WhitePulse             = db.SkillEffectWhitePulse
	SpearProjectile        = db.SkillEffectSpearProjectile
	SpearHitSound          = db.SkillEffectSpearHitSound
	EnemyHitNormal1        = db.SkillEffectEnemyHitNormal1
	Quake                  = db.SkillEffectQuake
	AnkleSnareGround       = db.SkillEffectAnkleSnareGround
	SharpShootingCast      = db.SkillEffectSharpShootingCast
	AdrenalineCast         = db.SkillEffectAdrenalineCast
	MaximizeSounds         = db.SkillEffectMaximizeSounds
	GreedSound             = db.SkillEffectGreedSound
	GospelGround           = db.SkillEffectGospelGround
	HermodeMusic           = db.SkillEffectHermodeMusic
	BottomDissonanceGround = db.SkillEffectBottomDissonanceGround
	BottomLullabyGround    = db.SkillEffectBottomLullabyGround
	BottomRichKimGround    = db.SkillEffectBottomRichKimGround
	BottomChaosGround      = db.SkillEffectBottomChaosGround
	BottomDrumGround       = db.SkillEffectBottomDrumGround
	BottomNibelungGround   = db.SkillEffectBottomNibelungGround
	BottomRokiGround       = db.SkillEffectBottomRokiGround
	BottomAbyssGround      = db.SkillEffectBottomAbyssGround
	BottomSiegGround       = db.SkillEffectBottomSiegGround
	BottomWhistleGround    = db.SkillEffectBottomWhistleGround
	BottomSinXGround       = db.SkillEffectBottomSinXGround
	BottomBragiGround      = db.SkillEffectBottomBragiGround
	BottomAppleGround      = db.SkillEffectBottomAppleGround
	BottomUglyDanceGround  = db.SkillEffectBottomUglyDanceGround
	BottomHummingGround    = db.SkillEffectBottomHummingGround
	BottomForgetGround     = db.SkillEffectBottomForgetGround
	BottomFortuneGround    = db.SkillEffectBottomFortuneGround
	BottomServiceGround    = db.SkillEffectBottomServiceGround
	GunslingerCoinSound    = db.SkillEffectGunslingerCoinSound
	Emotion                = 1000000
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
	ActorID                              uint32
	TargetID                             uint32
	castSourceID                         uint32
	ServerTick                           uint32
	ActorDirection                       int
	ActorOrigin                          scene.Point3
	ProjectileDirection                  scene.Point3
	HitCount                             int
	Shield                               *shieldFlight
	ShieldChain                          *shieldChainFlight
	PressureLanded                       bool
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
	HasActorTransform                    bool
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
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component)
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
	effect := Instance{
		EffectID: effectID,
		ActorID:  actorID,
		TargetID: targetID,
		X:        x,
		Y:        y,
		Starts:   starts,
		Expires:  starts.Add(duration),
		Duration: durationOverride,
	}
	if effectID == EnergyDrain2 {
		initSoulDrain(ctx, &effect)
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
	if effectID == CartRevolution {
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
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component)
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
	effect := Instance{
		EffectID:   effectID,
		ActorID:    actorID,
		X:          x,
		Y:          y,
		Starts:     starts,
		Expires:    expires,
		Persistent: persistent,
	}
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
	duration := spec.Duration
	for _, component := range spec.Components {
		componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component)
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
	effect := Instance{
		EffectID:                             effectID,
		ActorID:                              actorID,
		X:                                    x,
		Y:                                    y,
		Starts:                               starts,
		Expires:                              starts.Add(duration),
		Duration:                             durationOverride,
		Size:                                 sizeOverride,
		GroundSampleRotationRadiansPerSecond: rotationRadiansPerSecond,
	}
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
		if effect.EffectID == TripleAction {
			m.scheduleSound(starts.Add(delay), path)
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
		if effect.EffectID == Pressure && !effect.PressureLanded {
			at := now
			// Process 1 starts at time zero; process 32 ends the fall.
			landingAt := effect.Starts.Add(31 * db.EffectFrameDuration)
			if !now.Before(landingAt) {
				at = landingAt
				effect.PressureLanded = true
			}
			effect.ActorOrigin = EffectWorldAnchor(ctx, effect, at)
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
			// Concentration keeps its tint until status end, but its STR plays once.
			if effect.Persistent && effect.EffectID != LKConcentration {
				component.Repeat = true
			}
			componentDuration := m.WorldEffectResolvedComponentDuration(ctx, spec, component)
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

func (m *System) WorldEffectResolvedComponentDuration(ctx client.Context, spec Spec, component Component) time.Duration {
	duration := WorldEffectComponentDuration(spec, component)
	if component.Kind == ComponentSTR {
		if str := m.LoadWorldEffectSTR(ctx.Resources, ResolveEffectSTRFile(component, Instance{}, lessEffectsEnabled(ctx)), component.TexturePath); str != nil {
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
	value := uint32(effect.EffectID*1103515245) ^ effect.ActorID ^ uint32(effect.Starts.UnixNano()) ^ uint32(salt*2654435761)
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
