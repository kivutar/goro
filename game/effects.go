package game

import (
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kivutar/goro/client"
	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/glog"
	"github.com/kivutar/goro/network"
	"github.com/kivutar/goro/render"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

const (
	effectPixelRatio = db.EffectPixelRatio

	effectFireBolt       = 10019
	effectNapalmBeat     = 32
	effectGroundSample   = 513
	effectCastRing       = 10021
	effectProvoke        = 67
	effectMvp            = 68
	effectEndure         = 11
	effectBeginSpell     = 12
	effectSafetyWall     = 315
	effectColdBolt       = 10014
	effectBashBegin      = 16
	effectHit1           = 0
	effectBashHit        = 1
	effectHit3           = 2
	effectHit4           = 3
	effectHit5           = 4
	effectHit6           = 5
	effectEntry          = 6
	effectExit           = 7
	effectWarp           = 8
	effectEnhance        = 9
	effectArrowShot      = 10060
	effectArrowShower    = 10061
	effectMagicPower     = db.SkillEffectMagicPower
	effectGravitation    = db.SkillEffectGravitationGround
	effectMammonite      = 10
	effectCartRevolution = 170
	effectSight          = 22
	effectSoulStrike     = 15
	effectGlassWall      = 13
	effectHealSP         = 14
	effectMagnumBreak    = 17
	effectQuakeMagnum    = 10022
	effectSteal          = 18
	effectSummonSlave    = 215
	effectPoisonAttack   = 20
	effectDetoxication   = 21
	effectStoneCurse     = 23
	effectFireBall       = 24
	effectFireWall       = 25
	effectIceArrow       = 26
	effectFrostDiver     = 27
	effectFrostDiverHit  = 28
	effectLightningBolt  = 29
	effectThunderStorm   = 30
	effectFireArrow      = 31
	effectTeleportOld    = 34
	effectReadyPortalOld = 35
	effectIncAgility     = 37
	effectDecAgility     = 38
	effectIncAgiDex      = 43
	effectRuwach         = 33
	effectAqua           = 39
	effectSignum         = 40
	effectAngelus        = 41
	effectBlessing       = 42
	effectGloria         = 75
	effectMagnificat     = 76
	effectResurrection   = 77
	effectLexAeterna     = 85
	effectSuffragium     = 88
	effectStormGust      = 89
	effectWeaponPerfect  = 103
	effectMaximizePower  = 104
	effectKyrie          = 112
	effectChristmasCarol = 717
	effectFireHit        = 49
	effectFireSplashHit  = 50
	effectColdHit        = 51
	effectWindHit        = 52
	effectPoisonHit      = 53
	effectBeginSpell2    = 54
	effectBeginSpell3    = 55
	effectBeginSpell4    = 56
	effectBeginSpell5    = 57
	effectBeginSpell6    = 58
	effectBeginSpell7    = 59
	effectLockOnTarget   = 60
	effectWarpZone       = 61
	effectSightTrasher   = 62
	effectArrowShotRO    = 64
	effectInvenom        = 65
	effectSkidTrap       = 69
	effectBrandishSpear  = 70
	effectIceWall        = 74
	effectRecovery       = 78
	effectEarthSpike     = 79
	effectSpearBoomerang = 80
	effectPierce         = 81
	effectTurnUndead     = 82
	effectSanctuary      = 83
	effectImpositio      = 84
	effectAspersio       = 86
	effectLexDivina      = 87
	effectLordVermilion  = 90
	effectBenedictio     = 91
	effectMeteorStorm    = 92
	effectJupitelThunder = 93
	effectJupitelHit     = 94
	effectQuagmire       = 95
	effectFirePillar     = 96
	effectFirePillarBomb = 97
	effectHasteUp        = 98
	effectFlasher        = 99
	effectRemoveTrap     = 100
	effectRepairWeapon   = 101
	effectCrashEarth     = 102
	effectBlastMine      = 105
	effectBlastMineBomb  = 106
	effectClaymore       = 107
	effectFreezingTrap   = 108
	effectGasPush        = 110
	effectSpringTrap     = 111
	effectMagnus         = 113
	effectBlitzBeat      = 115
	effectWaterBall      = 116
	effectWaterBall2     = 117
	effectDetecting      = 119
	effectCloaking       = 120
	effectSonicBlow      = 121
	effectSonicBlowHit   = 122
	effectGrimtooth      = 123
	effectVenomDust      = 124
	effectPoisonReact    = 126
	effectPoisonReact2   = 127
	effectOverthrust     = 128
	effectVenomSplasher  = 129
	effectTwoHandQuicken = 130
	effectAutoCounter    = 131
	effectGrimtoothAtk   = 132
	effectFreeze         = 133
	effectFreezed        = 134
	effectIceCrash       = 135
	effectSlowPoison     = 136
	effectFirePillarOn   = 138
	effectSandman        = 139
	effectRevive         = 140
	effectHeavenDrive    = 142
	effectSonicBlow2     = 143
	effectBrandishSpear2 = 144
	effectShockwave      = 145
	effectShockwaveHit   = 146
	effectEarthHit       = 147
	effectPierceSelf     = 148
	effectBowlingSelf    = 149
	effectSpearStabSelf  = 150
	effectSpearBmrSelf   = 151
	effectRain           = 161
	effectSnow           = 162
	effectSakura         = 163
	effectBanjjakii      = 165
	effectMakeBlur       = 166
	effectSmoke          = 44
	effectFirefly        = 45
	effectTorch          = 47
	effectBubble         = 109
	effectCure           = 66
	effectPneuma         = 141
	effectHolyLight      = 152
	effectConcentration  = 153
	effectRefineOK       = 154
	effectRefineFail     = 155
	effectTeleportation  = 304
	effectPharmacyOK     = 305
	effectPharmacyFail   = 306
	effectFirstAid       = 309
	effectHeal           = 312
	effectReadyPortal    = 316
	effectPortal         = 317
	effectHealOffensive  = 320
	effectBaseLevelUp    = 371
	effectJobLevelUp     = 158
	effectVenomDust2     = 171
	effectMentalBreak    = 181
	effectMagicalAtkHit  = 182
	effectSuiExplosion   = 183
	effectSuicide        = 185
	effectComboAttack1   = 186
	effectComboAttack2   = 187
	effectComboAttack3   = 188
	effectComboAttack4   = 189
	effectComboAttack5   = 190
	effectGuidedAttack   = 191
	effectPoisonAttack2  = 192
	effectSilenceAttack  = 193
	effectStunAttack     = 194
	effectPetrifyAttack  = 195
	effectSleepAttack    = 197
	effectPong           = 199
	effectLevel99        = 200
	effectLevel99Ground  = 201
	effectLevel99Bubble  = 202
	effectGumgang        = 203
	effectPotionRed      = 204
	effectPotionOrange   = 205
	effectPotionYellow   = 206
	effectPotionWhite    = 207
	effectPotionBlue     = 208
	effectPotionGreen    = 209
	effectFood           = 210
	effectFoodBlue       = 211
	effectDarkBreath     = 212
	effectDefender       = 213
	effectKeeping        = 214
	effectBloodDrain     = 216
	effectEnergyDrain    = 217
	effectItemFast       = 218
	effectItemFast2      = 219
	effectItemFast3      = 220
	effectCrusaderDef    = 222
	effectGrandCross     = 226
	effectIntimidate     = 227
	effectChookgi        = 228
	effectLineLink       = 232
	effectSpellBreaker   = 234
	effectDispell        = 235
	effectBottomVolcano  = 239
	effectBottomDeluge   = 240
	effectBottomViolent  = 241
	effectBottomLand     = 242
	effectMagicRod       = 244
	effectHolyCross      = 245
	effectShieldCharge   = 246
	effectProvidence     = 248
	effectShieldBoomer   = 249
	effectSpearQuicken   = 250
	effectDevotion       = 251
	effectReflectShield  = 252
	effectAbsorbSpirits  = 253
	effectSteelBody      = 254
	effectFlameLauncher  = 255
	effectFrostWeapon    = 256
	effectLightningLoad  = 257
	effectSeismicWeapon  = 258
	effectGumgang2       = 261
	effectTeiHit1        = 262
	effectGumgang3       = 263
	effectTanji          = 265
	effectTeiHit1X       = 266
	effectChimto         = 267
	effectStealCoin      = 268
	effectStripWeapon    = 269
	effectStripShield    = 270
	effectStripArmor     = 271
	effectStripHelm      = 272
	effectChainCombo     = 273
	effectRogueCoin      = 274
	effectBackStab       = 275
	effectTeiHit3        = 276
	effectBottomLullaby  = 278
	effectBottomRichKim  = 279
	effectBottomChaos    = 280
	effectBottomDrum     = 281
	effectBottomNibelung = 282
	effectBottomRoki     = 283
	effectBottomAbyss    = 284
	effectBottomSieg     = 285
	effectBottomWhistle  = 286
	effectBottomSinX     = 287
	effectBottomBragi    = 288
	effectBottomApple    = 289
	effectBottomHumming  = 291
	effectBottomForget   = 292
	effectBottomFortune  = 293
	effectBottomService  = 294
	effectTalkFrostJoke  = 295
	effectTalkScream     = 296
	effectThrowItem      = 298
	effectChemicalProt   = 300
	effectDemonstration  = 302
	effectChemical2      = 303
	effectHeal2          = 313
	effectExit2          = 314
	effectBottomMagnus   = 318
	effectBottomSanc     = 319
	effectWarpZone2      = 321
	effectHeal4          = 325
	effectBeginAsura     = 328
	effectTripleAttack   = 329
	effectHPTime         = 331
	effectSPTime         = 332
	effectBlind          = 334
	effectPoisonStatus   = 335
	effectGuard          = 336
	effectJobLvUp50      = 337
	effectMagnum2        = 339
	effectEntry2         = 344
	effectColorPaper     = 347
	effectFoodChocolate  = 363
	effectResistPotion   = 491
	effectItemAccel      = 507
	effectFirecracker    = 508
	effectItemSlow       = 519
	effectBoxThunder     = 576
	effectBoxResentment  = 577
	effectBoxDrowsiness  = 579
	effectBoxSunlight    = 580
	effectStatFoodSTR    = 593
	effectStatFoodINT    = 594
	effectStatFoodVIT    = 595
	effectStatFoodAGI    = 596
	effectStatFoodDEX    = 597
	effectStatFoodLUK    = 598
	effectFirecracker1   = 612
	effectFirecracker2   = 682
	effectFirecracker3   = 683
	effectFirecracker4   = 684
	effectFirecracker5   = 685
	effectFirecracker6   = 686
	effectFirecracker7   = 709
	effectEnergyCoat     = 169
	effectThrowItem3     = 308
	effectSprinkleSand   = 310
	effectLoud           = 311
	effectPokJuk         = 297
	effectCloud          = 229
	effectCloud2         = 230
	effectMapPillar      = 231
	effectMapPillar2     = 247
	effectMapPillar3     = 259
	effectMapPillar4     = 260
	effectCloud3         = 233
	effectMaple          = 333
	effectDragonSmoke    = 373
	effectRainbow        = 410
	effectCloud4         = 515
	effectCloud5         = 516
	effectCloud6         = 592
	effectBubbleDrop     = 665
	effectTorchRed       = 690
	effectTorchGreen     = 691
	effectMapGhost       = 692
	effectGlow1          = 693
	effectGlow2          = 694
	effectGlow4          = 695
	effectTorchPurple    = 696
	effectCloud7         = 697
	effectCloud8         = 698
	effectEmotion        = 1000000
)

const (
	effectBottomDissonance = 277
	effectBottomUglyDance  = 290
)

const (
	effectSoulBreaker       = 361
	effectLevel99Aura1      = 362
	effectPressure          = 365
	effectBash3D            = 366
	effectAuraBlade         = 367
	effectRedBody           = 368
	effectLKConcentration   = 369
	effectBottomGospel      = 370
	effectDeath             = 372
	effectBottomBasilica    = 374
	effectHitLine2          = 376
	effectBash3D2           = 377
	effectEnergyDrain2      = 378
	effectTransBlueBody     = 379
	effectMagicCrasher      = 380
	effectLightBlade        = 382
	effectEnergyDrain3      = 383
	effectLineLink2         = 384
	effectTrueSight         = 386
	effectFalconAssault     = 387
	effectTripleAttack2     = 388
	effectPortal4           = 389
	effectMeltdown          = 390
	effectCartBoost         = 391
	effectRejectSword       = 392
	effectTripleAttack3     = 393
	effectMoonlit           = 394
	effectLevel99AuraMid    = 397
	effectLevel99AuraBottom = 398
	effectBash3D3           = 399
	effectBash3D4           = 400
	effectPortal5           = 402
	effectMagicCrasher2     = 403
	effectBottomSpider      = 404
	effectSoulBurn          = 406
	effectSoulChange        = 407
	effectSoulBreaker2      = 409
	effectBabyBody          = 420
	effectBabyBody2         = 421
	effectGiantBody         = 422
	effectGiantBody2        = 423
	effectQuakeBody         = 426
	effectAssumptio2        = 440
	effectStopEffect        = 444
	effectJumpBody          = 445
)

const (
	effectDarkGrandCross   = 450
	effectDarkSoulStrike   = 451
	effectDarkJupitelHit   = 452
	effectNPCStop          = 453
	effectDarkCasting      = 454
	effectNPCPowerUp       = 456
	effectJumpKick         = 457
	effectBeginAsura1      = 467
	effectBeginAsura2      = 468
	effectBeginAsura3      = 469
	effectBeginAsura4      = 470
	effectBeginAsura5      = 471
	effectBeginAsura6      = 472
	effectBeginAsura7      = 473
	effectMochi            = effectResistPotion
	effectRamadan          = 492
	effectEDP              = 493
	effectPreserve         = 496
	effectCastSpin         = 501
	effectChookgi2         = 504
	effectMapae            = effectItemAccel
	effectItemPokJuk       = effectFirecracker
	effectValentine05      = 509
	effectBeginAsura11     = 510
	effectChemical2Dash    = 512
	effectBottomHermode    = 517
	effectItemFastDown     = effectItemSlow
	effectTarotCard1       = 523
	effectTarotCard2       = 524
	effectTarotCard3       = 525
	effectTarotCard4       = 526
	effectTarotCard5       = 527
	effectTarotCard6       = 528
	effectTarotCard7       = 529
	effectTarotCard8       = 530
	effectTarotCard9       = 531
	effectTarotCard10      = 532
	effectTarotCard11      = 533
	effectTarotCard12      = 534
	effectTarotCard13      = 535
	effectTarotCard14      = 536
	effectAcidDemon        = 537
	effectHated            = 543
	effectStin             = 547
	effectStin2            = 553
	effectStin3            = 555
	effectScreenQuake      = 563
	effectHfliMoon1        = 565
	effectHfliMoon2        = 566
	effectHfliMoon3        = 567
	effectHoUp             = 568
	effectHamiDefence      = 569
	effectHamiCastle       = 570
	effectHamiBlood        = 571
	effectItemThunder      = effectBoxThunder
	effectItemCloud        = effectBoxResentment
	effectItemCurse        = 578
	effectItemZZZ          = effectBoxDrowsiness
	effectItemRain         = effectBoxSunlight
	effectM01              = 583
	effectM02              = 584
	effectM03              = 585
	effectM04              = 586
	effectM05              = 587
	effectM06              = 588
	effectM07              = 589
	effectKaizel           = 590
	effectThrowItem6       = 600
	effectFireHit2         = 603
	effectNPCStop2         = 604
	effectFVoice           = 606
	effectWink             = 607
	effectCookingOK        = 608
	effectCookingFail      = 609
	effectHapgyeok         = 612
	effectThrowItem7       = 613
	effectThrowItem8       = 614
	effectThrowItem9       = 615
	effectThrowItem10      = 616
	effectKouenka          = 618
	effectHyousensou       = 619
	effectStin4            = 621
	effectThunderStorm2    = 622
	effectRGCoin3          = 627
	effectBash3D5          = 628
	effectChookgi3         = 629
	effectKirikage         = 630
	effectTatami           = 631
	effectKasumikiri       = 632
	effectIssen            = 633
	effectKaen             = 634
	effectBaku             = 635
	effectHyousyouraku     = 636
	effectDesperado        = 637
	effectLightningS       = 638
	effectBlindS           = 639
	effectPoisonS          = 640
	effectFreezingS        = 641
	effectFlareS           = 642
	effectRapidShower      = 643
	effectMagicalBullet    = 644
	effectSpreadAttack     = 645
	effectTrackCasting     = 646
	effectTracking         = 647
	effectTripleAction     = 648
	effectBullseye         = 649
	effectNPCEarthquake    = 666
	effectDragonFear       = 668
	effectWideBleeding     = 669
	effectWideConfuse      = 670
	effectBottomRunner     = 671
	effectBottomTransfer   = 672
	effectBottomEvilLand   = 674
	effectGuard3           = 675
	effectCriticalWound    = 677
	effectFlowerLeaf       = 699
	effectItem315          = 704
	effectItem316          = 705
	effectItem317          = 706
	effectStormMin         = 708
	effectBottomBlue       = 715
	effectBottomBlue2      = 716
	effectFirePillarOn2    = 718
	effectForestLight5     = 719
	effectAdoramus         = 721
	effectIgnitionBreak    = 722
	effectFrostMisty       = 726
	effectCrimsonRock      = 727
	effectHellInferno      = 728
	effectMarshOfAbyss     = 729
	effectDragonHowling    = 731
	effectEarthWall        = 732
	effectChainLightning   = 734
	effectAimedBolt        = 745
	effectArrowStorm       = 746
	effectLaulamus         = 747
	effectLauagnus         = 748
	effectMillenniumShield = 749
	effectConcentration2   = 750
	effectGlassWall3       = 754
	effectBerserkPotion2   = 756
	effectRolling1         = 758
	effectRolling2         = 759
	effectRolling3         = 760
	effectRolling4         = 761
	effectRolling5         = 762
	effectRolling6         = 763
	effectRolling7         = 764
	effectRolling8         = 765
	effectRolling9         = 766
	effectRolling10        = 767
	effectCastSpin2        = 775
	effectCrashAxe         = 795
	effectStasis           = 799
	effectBottomBasilica2  = 802
	effectRecognized       = 803
	effectTetra            = 804
	effectTetraCasting     = 805
	effectStretch          = 811
	effectEnervation       = 813
	effectEnervation2      = 814
	effectEnervation3      = 815
	effectEnervation4      = 816
	effectEnervation5      = 817
	effectEnervation6      = 818
	effectBottomManhole    = 822
	effectManhole          = 823
	effectForestLight6     = 825
	effectBottomAni        = 827
	effectBottomMaelstrom  = 828
	effectBottomBloodyLust = 829
	effectHealN            = 832
	effectChookgiN         = 833
	effectDance1           = 847
	effectBotReverb        = 856
	effectRainParticle     = 857
	effectChemicalV2       = 858
	effectBotReverb2       = 860
	effectCirclePower2     = 861
	effectSecra2           = 862
	effectSprPlant2        = 866
	effectSprPlant3        = 868
	effectSprPlant4        = 870
	effectSprPlant5        = 872
	effectSprPlant6        = 874
	effectSprPlant7        = 876
	effectSprPlant8        = 878
	effectHeartAsura       = 879
	effectGlassWall4       = 883
	effectBash3D6          = 885
	effectElectric4        = 888
	effectTeiHit1T         = 889
	effectPressure2        = 906
	effectPrimeCharge2     = 908
	effectPrimeCharge3     = 909
	effectPrimeCharge4     = 910
	effectFireWall2        = 920
	effectSprPlant10       = 922
	effectShockwave2       = 926
	effectColdThrow2       = 928
	effectDemonicFire4     = 929
	effectPressure3        = 930
	effectPoisonMist       = 959
	effectEraserCutter     = 960
	effectLavaSlide        = 964
	effectSonicClaw        = 965
	effectTinderBreaker    = 966
	effectMidnightFrenzy   = 967
	effectVolcanicAsh      = 975
	effectRWC2011          = 987
	effectRWC2011Two       = 988
	effectRunMakeOK        = 1015
	effectRunMakeFailure   = 1016
	effectMIResultMakeOK   = 1017
	effectMIResultMakeFail = 1018
	effectAllRayProtect    = 1019
	effectVenomFog         = 1020
	effectDustStorm        = 1021
	effectDanceBladeAtk    = 1029
	effectInvincibleOff2   = 1031
	effectDeathSummon      = 1033
	effectGCDarkCrow       = 1040
	effectAllFullThrottle  = 1042
	effectSRFlashCombo     = 1043
	effectRKLuxAnima       = 1044
	effectSOElemShield     = 1046
	effectABOffertorium    = 1047
	effectWLTelekinesis    = 1048
	effectGNIllusionDoping = 1049
	effectNCMagmaEruption  = 1050
	effectNPCChill         = 1055
	effectOffertoriumRing  = 1057
	effectHammerOfGod      = 1062
	effectAchComplete      = 1094
	effectBodyColor        = 1111
	effectBakuretsuHadou   = 1130
	dropEffectPink         = 1186
	dropEffectYellow       = 1189
	dropEffectPurple       = 1190
	effectDigitalSpace     = 1240
	dropEffectBlue         = 1869
	dropEffectGreen        = 1870
	dropEffectRed          = 1871
	effectNewSuccess       = 1872
	effectNewFailure       = 1873
	effectNewIntro         = 1874
	effectEnchantYellow    = 1875
	effectEnchantSuccess   = 1876
	effectEnchantFail      = 1877
	effectEnchantBlue      = 1878
	effectEnchantUpSuccess = 1879
	effectEnchantUpFail    = 1880
	effectEnchantGreen     = 1881
	effectEnchantResetOK   = 1882
	effectEnchantResetFail = 1883
)

const (
	effectWhitePulse             = db.SkillEffectWhitePulse
	effectSpearProjectile        = db.SkillEffectSpearProjectile
	effectSpiralBeforeCast       = db.SkillEffectSpiralBeforeCast
	effectSpearHitSound          = db.SkillEffectSpearHitSound
	effectEnemyHitNormal1        = db.SkillEffectEnemyHitNormal1
	effectQuake                  = db.SkillEffectQuake
	effectAnkleSnareGround       = db.SkillEffectAnkleSnareGround
	effectSharpShootingCast      = db.SkillEffectSharpShootingCast
	effectAdrenalineCast         = db.SkillEffectAdrenalineCast
	effectMaximizeSounds         = db.SkillEffectMaximizeSounds
	effectGreedSound             = db.SkillEffectGreedSound
	effectGospelGround           = db.SkillEffectGospelGround
	effectShieldProjectile       = db.SkillEffectShieldProjectile
	effectFogWallGround          = db.SkillEffectFogWallGround
	effectHermodeMusic           = db.SkillEffectHermodeMusic
	effectBottomDissonanceGround = db.SkillEffectBottomDissonanceGround
	effectBottomLullabyGround    = db.SkillEffectBottomLullabyGround
	effectBottomRichKimGround    = db.SkillEffectBottomRichKimGround
	effectBottomChaosGround      = db.SkillEffectBottomChaosGround
	effectBottomDrumGround       = db.SkillEffectBottomDrumGround
	effectBottomNibelungGround   = db.SkillEffectBottomNibelungGround
	effectBottomRokiGround       = db.SkillEffectBottomRokiGround
	effectBottomAbyssGround      = db.SkillEffectBottomAbyssGround
	effectBottomSiegGround       = db.SkillEffectBottomSiegGround
	effectBottomWhistleGround    = db.SkillEffectBottomWhistleGround
	effectBottomSinXGround       = db.SkillEffectBottomSinXGround
	effectBottomBragiGround      = db.SkillEffectBottomBragiGround
	effectBottomAppleGround      = db.SkillEffectBottomAppleGround
	effectBottomUglyDanceGround  = db.SkillEffectBottomUglyDanceGround
	effectBottomHummingGround    = db.SkillEffectBottomHummingGround
	effectBottomForgetGround     = db.SkillEffectBottomForgetGround
	effectBottomFortuneGround    = db.SkillEffectBottomFortuneGround
	effectBottomServiceGround    = db.SkillEffectBottomServiceGround
	effectGunslingerCoinSound    = db.SkillEffectGunslingerCoinSound
)

const skillUnitEffectFallbackDuration = 5 * time.Minute

type effectComponentKind int

const (
	effectComponentSTR effectComponentKind = iota + 1
	effectComponentCylinder
	effectComponent2D
	effectComponent3D
	effectComponentSPR
	effectComponentFUNC
	effectComponentQuadHorn
)

type worldEffect struct {
	effectID                             int
	actorID                              uint32
	targetID                             uint32
	x                                    int
	y                                    int
	starts                               time.Time
	expires                              time.Time
	duration                             time.Duration
	size                                 float64
	groundSampleRotationRadiansPerSecond float64
	persistent                           bool
	spriteFrameOverride                  int
	hasSpriteFrame                       bool
}

type worldEffectSpec struct {
	duration         time.Duration
	cameraShake      time.Duration
	cameraShakeDelay time.Duration
	detachLocalActor bool
	sfx              []string
	sfxDelays        []time.Duration
	sfxRandMin       int
	sfxRandMax       int
	components       []worldEffectComponent
}

type worldEffectComponent struct {
	kind               effectComponentKind
	funcAdapter        effectFuncAdapter
	funcName           string
	color              color.RGBA
	duration           time.Duration
	durationRandMin    time.Duration
	durationRandMax    time.Duration
	delay              time.Duration
	duplicateDelay     time.Duration
	delayOffsetDelta   time.Duration
	repeat             bool
	repeatDelay        time.Duration
	strFile            string
	strMinFile         string
	strRandMin         int
	strRandMax         int
	attachedEntity     bool
	renderBefore       bool
	texturePath        string
	textureName        string
	textureFile        string
	textureFiles       []string
	frameDelay         time.Duration
	spriteFile         string
	shadowTexture      bool
	spriteHead         bool
	spriteDirection    bool
	spriteRepeat       bool
	spriteStopAtEnd    bool
	spriteFrame        int
	spriteDelay        time.Duration
	spriteXOffset      float64
	spriteYOffset      float64
	fromSrc            bool
	toSrc              bool
	arc                float64
	retreat            float64
	alphaMax           float64
	alphaMaxDelta      float64
	sparkling          bool
	sparkNumber        int
	fade               bool
	fadeIn             bool
	fadeOut            bool
	rotate             bool
	rotateWithCamera   bool
	fixedPerspective   bool
	rotateToTarget     bool
	worldSizedSprite   bool
	animation          int
	bottomSize         float64
	topSize            float64
	height             float64
	posX               float64
	posY               float64
	posZ               float64
	posXEnd            float64
	posYEnd            float64
	posZEnd            float64
	posXRand           float64
	posYRand           float64
	posZRand           float64
	posXStartRand      float64
	posYStartRand      float64
	posZStartRand      float64
	posXStartMiddle    float64
	posYStartMiddle    float64
	posZStartMiddle    float64
	posXEndRand        float64
	posYEndRand        float64
	posZEndRand        float64
	posXEndMiddle      float64
	posYEndMiddle      float64
	posZEndMiddle      float64
	posXSmooth         bool
	posYSmooth         bool
	posZSmooth         bool
	sizeStart          float64
	sizeEnd            float64
	sizeRand           float64
	sizeStartX         float64
	sizeStartY         float64
	sizeEndX           float64
	sizeEndY           float64
	sizeStartXRandMin  float64
	sizeStartXRandMax  float64
	sizeStartYRandMin  float64
	sizeStartYRandMax  float64
	sizeEndXRandMin    float64
	sizeEndXRandMax    float64
	sizeEndYRandMin    float64
	sizeEndYRandMax    float64
	sizeRandX          float64
	sizeRandY          float64
	sizeRandXMiddle    float64
	sizeRandYMiddle    float64
	sizeDelta          float64
	sizeSmooth         bool
	angleStart         float64
	angleEnd           float64
	angleX             float64
	angleY             float64
	angleZ             float64
	angleRandMin       float64
	angleRandMax       float64
	circlePattern      bool
	circleInnerSize    float64
	circleOuterRandMin float64
	circleOuterRandMax float64
	orbitRadiusX       float64
	orbitRadiusY       float64
	orbitRadiusZ       float64
	orbitRotations     float64
	orbitPhase         float64
	orbitPhaseDelta    float64
	orbitClockwise     bool
	totalCircleSides   int
	circleSides        int
	duplicate          int
	angleZRandom       float64
	blendMode          int
	blendAdditive      bool
	overlay            bool
	quadHornHeightMin  float64
	quadHornHeightMax  float64
	quadHornOffsetXMin float64
	quadHornOffsetXMax float64
	quadHornOffsetYMin float64
	quadHornOffsetYMax float64
	quadHornOffsetZ    float64
	quadHornBottomMin  float64
	quadHornBottomMax  float64
	quadHornRotateXMin float64
	quadHornRotateXMax float64
	quadHornRotateYMin float64
	quadHornRotateYMax float64
	quadHornRotateZMin float64
	quadHornRotateZMax float64
	quadHornAnimSpeed  time.Duration
	quadHornAnimOut    bool
}

func (m *WorldMode) addItemUseEffect(ctx client.Context, ack network.UseItemAck) {
	if ack.Result == 0 {
		return
	}
	spec := itemUseEffectSpec(ack.ItemID)
	effectIDs := spec.effectIDs
	casterEffectIDs := spec.effectIDsOnCaster
	if len(effectIDs) == 0 && len(casterEffectIDs) == 0 {
		return
	}
	actorID := itemUseEffectActorID(ctx, ack, spec.target)
	for _, effectID := range effectIDs {
		targetID := actorID
		if effectDetachesLocalActor(effectID) && isLocalActor(ctx, targetID) {
			targetID = 0
		}
		if m.addWorldEffect(ctx, effectID, targetID) {
			glog.Debugf("item effect item=%d actor=%d effect=%d", ack.ItemID, targetID, effectID)
		}
	}
	for _, effectID := range casterEffectIDs {
		casterID := actorID
		if effectDetachesLocalActor(effectID) && isLocalActor(ctx, casterID) {
			casterID = 0
		}
		if m.addWorldEffect(ctx, effectID, casterID) {
			glog.Debugf("item caster effect item=%d actor=%d effect=%d", ack.ItemID, casterID, effectID)
		}
	}
}

func itemUseEffectActorID(ctx client.Context, ack network.UseItemAck, target itemEffectTarget) uint32 {
	ackActorID := itemUseAckActorID(ctx, ack)
	if target == itemEffectTargetMercenary && (ackActorID == 0 || isLocalActor(ctx, ackActorID)) {
		if actorID := visibleMercenaryActorID(ctx); actorID != 0 {
			return actorID
		}
	}
	return ackActorID
}

func itemUseAckActorID(ctx client.Context, ack network.UseItemAck) uint32 {
	actorID := ack.AID
	if actorID == 0 && ctx.Session != nil {
		actorID = ctx.Session.AccountID
		if actorID == 0 {
			actorID = ctx.Session.CharID
		}
	}
	return actorID
}

func visibleMercenaryActorID(ctx client.Context) uint32 {
	if ctx.World == nil {
		return 0
	}
	if ctx.Session != nil && ctx.Session.Mercenary.ID != 0 {
		if actor, ok := ctx.World.Actors[ctx.Session.Mercenary.ID]; ok && actorIsMercenary(actor) {
			return actor.ID
		}
	}
	return findCompanionActorID(ctx, actorObjectTypeMercenary)
}

func (m *WorldMode) applySkillNoDamageNotify(ctx client.Context, notify network.SkillNoDamageNotify) {
	if notify.Result == 0 {
		return
	}
	now := time.Now()
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	m.startSkillNoDamageSourceAnimation(ctx, notify, now)
	m.applyFalconSkillNoDamageNotify(ctx, notify, now)
	m.addSkillBeginEffectsAt(ctx, notify.SkillID, notify.SourceID, notify.TargetID, now)
	if notify.SkillID == db.SkillTKSevenwind && notify.Amount >= 1 && notify.Amount <= 7 {
		effectID := effectBeginAsura1 + int(notify.Amount) - 1
		if m.addWorldEffectAt(ctx, effectID, notify.TargetID, now) {
			glog.Debugf("mild wind effect skill=%d target=%d effect=%d level=%d", notify.SkillID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillEffectIDs(notify.SkillID) {
		if m.addWorldEffectBetweenAt(ctx, effectID, notify.TargetID, notify.SourceID, now) {
			glog.Debugf("skill effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillEffectOnCasterIDs(notify.SkillID) {
		if m.addWorldEffectAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill caster effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillSuccessEffectIDs(notify.SkillID) {
		if m.addWorldEffectBetweenAt(ctx, effectID, notify.TargetID, notify.SourceID, now) {
			glog.Debugf("skill success effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
	for _, effectID := range skillSuccessEffectSelfIDs(notify.SkillID) {
		if m.addWorldEffectAt(ctx, effectID, notify.SourceID, now) {
			glog.Debugf("skill success self effect skill=%d src=%d target=%d effect=%d amount=%d", notify.SkillID, notify.SourceID, notify.TargetID, effectID, notify.Amount)
		}
	}
}

func (m *WorldMode) applySkillCastNotify(ctx client.Context, notify network.SkillCastNotify) {
	now := time.Now()
	m.applySkillNameBubble(ctx, notify.SourceID, notify.SkillID, now)
	m.applyFalconSkillCastNotify(ctx, notify, now)
	if notify.DelayTime == 0 {
		return
	}
	duration := time.Duration(notify.DelayTime) * time.Millisecond
	m.startSkillCastSourceAnimation(ctx, notify, duration, now)
	if !skillHidesCastBar(notify.SkillID) {
		m.startActorCastBar(ctx, notify.SourceID, duration, now)
	}
	m.addSkillCastEffects(ctx, notify.SkillID, notify.Property, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y), duration, now, "server")
}

func (m *WorldMode) startActorCastBar(ctx client.Context, sourceID uint32, duration time.Duration, started time.Time) {
	if sourceID == 0 || duration <= 0 {
		return
	}
	if started.IsZero() {
		started = time.Now()
	}
	bar := actorCastBar{
		started:  started,
		duration: duration,
		color:    color.RGBA{R: 0, G: 255, B: 0, A: 255},
	}
	m.setActorCastBar(sourceID, bar)
	if ctx.Session == nil || !isLocalActor(ctx, sourceID) {
		return
	}
	m.setActorCastBar(ctx.Session.AccountID, bar)
	m.setActorCastBar(ctx.Session.CharID, bar)
}

func (m *WorldMode) setActorCastBar(actorID uint32, bar actorCastBar) {
	if actorID == 0 {
		return
	}
	if m.actorCastBars == nil {
		m.actorCastBars = make(map[uint32]actorCastBar)
	}
	m.actorCastBars[actorID] = bar
}

func (m *WorldMode) clearActorCastBar(ctx client.Context, actorID uint32) {
	if actorID == 0 || m.actorCastBars == nil {
		return
	}
	delete(m.actorCastBars, actorID)
	if ctx.Session == nil || !isLocalActor(ctx, actorID) {
		return
	}
	delete(m.actorCastBars, ctx.Session.AccountID)
	delete(m.actorCastBars, ctx.Session.CharID)
}

func (m *WorldMode) startSkillNoDamageSourceAnimation(ctx client.Context, notify network.SkillNoDamageNotify, now time.Time) {
	source, ok, _ := actorForCombatID(ctx, notify.SourceID)
	if !ok {
		return
	}
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, 0, 0)
	m.startSkillActionAnimation(ctx, notify.SourceID, source, skillAction(notify.SkillID), now, defaultAttackAnimationDuration)
}

func (m *WorldMode) startSkillCastSourceAnimation(ctx client.Context, notify network.SkillCastNotify, duration time.Duration, now time.Time) {
	m.faceSkillSource(ctx, notify.SourceID, notify.TargetID, int(notify.X), int(notify.Y))
	m.startSkillSourceCastAnimation(ctx, notify.SourceID, notify.SkillID, duration, now)
}

func (m *WorldMode) startSkillSourceCastAnimation(ctx client.Context, sourceID uint32, skillID uint16, duration time.Duration, now time.Time) {
	source, ok, _ := actorForCombatID(ctx, sourceID)
	if !ok {
		return
	}
	m.startCombatAnimation(ctx, sourceID, skillCastActionFamilyForActor(source, skillID), now, duration)
}

func (m *WorldMode) startSkillActionAnimation(ctx client.Context, id uint32, actor worldstate.Actor, spec skillActionSpec, started time.Time, duration time.Duration) {
	anim := spec.actorAnimationForActorWithResources(ctx.Resources, actor, started, duration)
	m.setActorAction(ctx, id, anim)
	if ctx.Session == nil || !isLocalActor(ctx, id) {
		return
	}
	m.setActorAction(ctx, ctx.Session.AccountID, anim)
	m.setActorAction(ctx, ctx.Session.CharID, anim)
}

func (m *WorldMode) faceSkillSource(ctx client.Context, sourceID, targetID uint32, cellX, cellY int) {
	source, sourceOK, sourceLocal := actorForCombatID(ctx, sourceID)
	if !sourceOK {
		return
	}
	if target, targetOK, _ := actorForCombatID(ctx, targetID); targetOK {
		m.faceCombatSource(ctx, source, sourceLocal, target)
		return
	}
	if cellX == 0 && cellY == 0 {
		return
	}
	dir := directionFromDelta(source.X, source.Y, cellX, cellY, source.Dir)
	if sourceLocal {
		ctx.World.Player.Dir = dir
		ctx.World.Dir = dir
		return
	}
	source.Dir = dir
	upsertActor(ctx, source)
}

func (m *WorldMode) applyGroundSkillNotify(ctx client.Context, notify network.GroundSkillNotify) {
	// rAthena sends Battle Chant's cast notification at 0,0 and follows it with
	// one UNT_GOSPEL entry per field cell. Play EF_BOTTOM_GOSPEL once on the
	// caster here; the unit entries own the persistent ground visuals.
	if notify.SkillID == db.SkillPaGospel && notify.X == 0 && notify.Y == 0 {
		for _, effectID := range skillEffectIDs(notify.SkillID) {
			if m.addWorldEffectIfMissing(ctx, effectID, notify.SourceID) {
				glog.Debugf("ground skill caster effect skill=%d src=%d level=%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, effectID)
			}
		}
		return
	}
	effectIDs := skillGroundEffectIDs(notify.SkillID)
	if len(effectIDs) == 0 {
		effectIDs = skillEffectIDs(notify.SkillID)
	}
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	m.applyFalconGroundSkillNotify(ctx, notify, now)
	for _, effectID := range effectIDs {
		if m.addWorldEffectAtCellIfMissing(ctx, effectID, int(notify.X), int(notify.Y), now) {
			glog.Debugf("ground skill effect skill=%d src=%d level=%d cell=%d,%d effect=%d", notify.SkillID, notify.SourceID, notify.Level, notify.X, notify.Y, effectID)
		}
	}
}

func (m *WorldMode) applySkillUnitEntry(ctx client.Context, entry network.SkillUnitEntry) {
	if m.applySkillUnitModelEntry(entry) {
		return
	}
	if !entry.Visible {
		return
	}
	effectIDs := skillUnitEffectIDs(entry.UnitID)
	if len(effectIDs) == 0 {
		return
	}
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.addWorldEffectAtCellLifetime(ctx, effectID, entry.ID, int(entry.X), int(entry.Y), now, skillUnitEffectFallbackDuration, true) {
			glog.Debugf("skill unit effect unit=%d id=%d creator=%d cell=%d,%d effect=%d", entry.UnitID, entry.ID, entry.CreatorID, entry.X, entry.Y, effectID)
		}
	}
}

func (m *WorldMode) applySkillUnitLookChange(ctx client.Context, look network.ActorLookChange) bool {
	if look.Type != 0 || look.ID == 0 {
		return false
	}
	if m.applyUsedTrapLookChange(ctx, look) {
		return true
	}
	effectIDs := skillUnitEffectIDs(uint16(look.Value))
	if len(effectIDs) == 0 {
		return false
	}
	x, y, ok := m.skillUnitCell(look.ID)
	if !ok {
		return false
	}
	m.removeSkillUnitEffects(look.ID)
	now := time.Now()
	for _, effectID := range effectIDs {
		if m.addWorldEffectAtCellLifetime(ctx, effectID, look.ID, x, y, now, skillUnitEffectFallbackDuration, true) {
			glog.Debugf("skill unit effect changed id=%d unit=%d cell=%d,%d effect=%d", look.ID, look.Value, x, y, effectID)
		}
	}
	return true
}

func (m *WorldMode) applySkillUnitDisappear(disappear network.SkillUnitDisappear) {
	if disappear.ID == 0 {
		return
	}
	removedEffects := m.removeSkillUnitEffects(disappear.ID)
	removedModel := m.removeSkillUnitModelOnly(disappear.ID)
	if removedEffects || removedModel {
		glog.Debugf("skill unit effect removed id=%d", disappear.ID)
	}
}

func (m *WorldMode) skillUnitCell(id uint32) (int, int, bool) {
	if x, y, ok := m.skillUnitModelCell(id); ok {
		return x, y, true
	}
	for _, effect := range m.worldEffects {
		if effect.actorID == id {
			return effect.x, effect.y, true
		}
	}
	return 0, 0, false
}

func (m *WorldMode) removeSkillUnitEffects(id uint32) bool {
	return m.removeWorldEffectsForActor(id)
}

func (m *WorldMode) removeWorldEffectsForActor(id uint32) bool {
	active := m.worldEffects[:0]
	removed := false
	for _, effect := range m.worldEffects {
		if effect.actorID == id {
			removed = true
			continue
		}
		active = append(active, effect)
	}
	m.worldEffects = active
	return removed
}

func (m *WorldMode) applySpecialEffectNotify(ctx client.Context, notify network.SpecialEffectNotify) {
	effectID := specialEffectID(notify.EffectID)
	if effectID <= 0 {
		return
	}
	if isLocalActor(ctx, notify.AID) {
		switch notify.EffectID {
		case network.SpecialEffectBaseLevelUp,
			network.SpecialEffectSuperNoviceBaseLevelUp,
			network.SpecialEffectTaekwonBaseLevelUp:
			m.ui.levelUpNotifications.NotifyBase()
		case network.SpecialEffectJobLevelUp,
			network.SpecialEffectSuperNoviceJobLevelUp:
			m.ui.levelUpNotifications.NotifyJob()
		}
	}
	spec, _ := worldEffectSpecForID(effectID)
	if craftingResultSpecialEffect(notify.EffectID) {
		m.replaceCraftingResultEffect(ctx, notify, effectID, spec)
		return
	}
	if m.addWorldEffectIfMissing(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v", notify.AID, notify.EffectID, effectID, spec.sfx)
	}
}

func (m *WorldMode) applyMVPNotify(ctx client.Context, notify network.MVPNotify) {
	if m.addWorldEffect(ctx, effectMvp, notify.AID) {
		glog.Debugf("mvp effect actor=%d effect=%d", notify.AID, effectMvp)
	}
}

func (m *WorldMode) applySkillFailAck(ctx client.Context, ack network.SkillFailAck) {
	if ack.Result != 0 {
		return
	}
	m.clearSenseRequest(ack.SkillID)
	message := skillFailMessage(ack)
	glog.Debugf("skill fail ack skill=%d num=%d item=%d result=%d cause=%d msg=%q", ack.SkillID, ack.Number, ack.ItemID, ack.Result, ack.Cause, message)
	m.ui.console.AddErrorMessage("%s", message)
}

func specialEffectID(effectID uint32) int {
	if mapped, ok := networkSpecialEffectIDs[effectID]; ok {
		return mapped
	}
	if _, ok := worldEffectSpecForID(int(effectID)); ok {
		return int(effectID)
	}
	return 0
}

var networkSpecialEffectIDs = map[uint32]int{
	network.SpecialEffectBaseLevelUp:            effectBaseLevelUp,
	network.SpecialEffectJobLevelUp:             effectJobLevelUp,
	network.SpecialEffectRefineFailure:          effectRefineFail,
	network.SpecialEffectRefineSuccess:          effectRefineOK,
	network.SpecialEffectGameOver:               0,
	network.SpecialEffectPharmacySuccess:        effectPharmacyOK,
	network.SpecialEffectPharmacyFailure:        effectPharmacyFail,
	network.SpecialEffectSuperNoviceBaseLevelUp: effectBaseLevelUp,
	network.SpecialEffectSuperNoviceJobLevelUp:  effectJobLevelUp,
	network.SpecialEffectTaekwonBaseLevelUp:     effectBaseLevelUp,
}

func craftingResultSpecialEffect(effectID uint32) bool {
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

func (m *WorldMode) replaceCraftingResultEffect(ctx client.Context, notify network.SpecialEffectNotify, effectID int, spec worldEffectSpec) {
	for _, activeID := range craftingResultEffectGroup(effectID) {
		m.removeWorldEffect(activeID, notify.AID)
	}
	if m.addWorldEffect(ctx, effectID, notify.AID) {
		glog.Debugf("special effect actor=%d special=%d effect=%d sfx=%v restart=true", notify.AID, notify.EffectID, effectID, spec.sfx)
	}
}

func craftingResultEffectGroup(effectID int) []int {
	switch effectID {
	case effectRefineOK, effectRefineFail:
		return []int{effectRefineOK, effectRefineFail}
	case effectPharmacyOK, effectPharmacyFail:
		return []int{effectPharmacyOK, effectPharmacyFail}
	default:
		return []int{effectID}
	}
}

func skillFailMessage(ack network.SkillFailAck) string {
	if ack.Cause == 0 {
		if messages, ok := skillFailMessagesBySkill[ack.SkillID]; ok {
			if message, ok := messages[ack.Number]; ok {
				return message
			}
		}
	}
	if message, ok := skillFailMessagesByCause[ack.Cause]; ok {
		return message
	}
	return "Action failed."
}

var skillFailMessagesBySkill = map[uint16]map[uint32]string{
	1: {
		0: "Basic skill failed.",
		1: "Cannot use emotions.",
		2: "Cannot sit.",
		3: "Cannot chat.",
		4: "Cannot form a party.",
		5: "Cannot shout.",
		6: "Cannot PK.",
		7: "Cannot align.",
	},
	50: {
		0: "Steal failed.",
	},
}

var skillFailMessagesByCause = map[uint8]string{
	0: "Action failed.",
	1: "Not enough SP.",
	2: "Not enough HP.",
	4: "Action is still on cooldown.",
	5: "Not enough Zeny.",
	9: "Too much weight.",
}

func (m *WorldMode) addWorldEffect(ctx client.Context, effectID int, actorID uint32) bool {
	return m.addWorldEffectAt(ctx, effectID, actorID, time.Now())
}

func (m *WorldMode) addWorldEffectIfMissing(ctx client.Context, effectID int, actorID uint32) bool {
	if m.hasActiveWorldEffect(effectID, actorID, time.Now()) {
		return false
	}
	return m.addWorldEffect(ctx, effectID, actorID)
}

func (m *WorldMode) hasActiveWorldEffect(effectID int, actorID uint32, now time.Time) bool {
	for _, effect := range m.worldEffects {
		if effect.effectID == effectID && effect.actorID == actorID && now.Before(effect.expires) {
			return true
		}
	}
	return false
}

func (m *WorldMode) removeWorldEffect(effectID int, actorID uint32) bool {
	removed := false
	active := m.worldEffects[:0]
	for _, effect := range m.worldEffects {
		if effect.effectID == effectID && effect.actorID == actorID {
			removed = true
			continue
		}
		active = append(active, effect)
	}
	m.worldEffects = active
	return removed
}

func (m *WorldMode) addWorldEffectAt(ctx client.Context, effectID int, actorID uint32, starts time.Time) bool {
	return m.addWorldEffectBetweenAt(ctx, effectID, actorID, 0, starts)
}

func (m *WorldMode) addWorldEffectBetweenAt(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time) bool {
	return m.addWorldEffectBetweenAtDuration(ctx, effectID, actorID, targetID, starts, 0)
}

func (m *WorldMode) addWorldEffectBetweenAtDuration(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time, durationOverride time.Duration) bool {
	if ctx.World == nil {
		return false
	}
	spec, ok := worldEffectSpecForID(effectID)
	if !ok {
		return false
	}
	x, y, ok := effectAnchor(ctx, actorID)
	if !ok {
		return false
	}
	duration := spec.duration
	for _, component := range spec.components {
		componentDuration := m.worldEffectResolvedComponentDuration(ctx, spec, component)
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
	effect := worldEffect{
		effectID: effectID,
		actorID:  actorID,
		targetID: targetID,
		x:        x,
		y:        y,
		starts:   starts,
		expires:  starts.Add(duration),
		duration: durationOverride,
	}
	m.worldEffects = append(m.worldEffects, effect)
	m.scheduleWorldEffectSound(starts, spec, effect)
	m.startWorldEffectCameraShake(starts, spec)
	m.applyWorldEffectSideEffects(ctx, effect, starts)
	return true
}

func (m *WorldMode) addWorldEffectAtCellLifetime(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, lifetimeOverride time.Duration, persistent bool) bool {
	if ctx.World == nil {
		return false
	}
	spec, ok := worldEffectSpecForID(effectID)
	if !ok {
		return false
	}
	duration := spec.duration
	for _, component := range spec.components {
		componentDuration := m.worldEffectResolvedComponentDuration(ctx, spec, component)
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
		active := m.worldEffects[:0]
		updated := false
		moved := false
		for _, effect := range m.worldEffects {
			if effect.effectID == effectID && effect.actorID == actorID {
				if starts.Before(effect.expires) {
					if !updated {
						moved = effect.x != x || effect.y != y
						effect.x = x
						effect.y = y
						if effect.expires.Before(expires) {
							effect.expires = expires
						}
						effect.persistent = true
						active = append(active, effect)
						updated = true
					}
				}
				continue
			}
			active = append(active, effect)
		}
		m.worldEffects = active
		if updated {
			return moved
		}
	}
	effect := worldEffect{
		effectID:   effectID,
		actorID:    actorID,
		x:          x,
		y:          y,
		starts:     starts,
		expires:    expires,
		persistent: persistent,
	}
	m.worldEffects = append(m.worldEffects, effect)
	m.scheduleWorldEffectSound(starts, spec, effect)
	m.startWorldEffectCameraShake(starts, spec)
	return true
}

func (m *WorldMode) addWorldEffectAtCellDurationSize(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64) bool {
	return m.addWorldEffectAtCellDurationSizeRotation(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, 0)
}

func (m *WorldMode) addWorldEffectAtCellDurationSizeRotation(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64, rotationRadiansPerSecond float64) bool {
	if ctx.World == nil {
		return false
	}
	spec, ok := worldEffectSpecForID(effectID)
	if !ok {
		return false
	}
	duration := spec.duration
	for _, component := range spec.components {
		componentDuration := m.worldEffectResolvedComponentDuration(ctx, spec, component)
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
	effect := worldEffect{
		effectID:                             effectID,
		actorID:                              actorID,
		x:                                    x,
		y:                                    y,
		starts:                               starts,
		expires:                              starts.Add(duration),
		duration:                             durationOverride,
		size:                                 sizeOverride,
		groundSampleRotationRadiansPerSecond: rotationRadiansPerSecond,
	}
	m.worldEffects = append(m.worldEffects, effect)
	m.scheduleWorldEffectSound(starts, spec, effect)
	m.startWorldEffectCameraShake(starts, spec)
	return true
}

func (m *WorldMode) startWorldEffectCameraShake(starts time.Time, spec worldEffectSpec) {
	if spec.cameraShake <= 0 {
		return
	}
	m.startCameraShake(starts.Add(spec.cameraShakeDelay), spec.cameraShake)
}

func (m *WorldMode) scheduleWorldEffectSound(starts time.Time, spec worldEffectSpec, effect worldEffect) {
	if len(spec.sfx) == 0 {
		return
	}
	paths := resolveEffectSFX(spec, effect)
	for i, path := range paths {
		delay := time.Duration(0)
		if i < len(spec.sfxDelays) {
			delay = spec.sfxDelays[i]
		}
		m.scheduleSoundAtWorldEffect(starts.Add(delay), effect, path)
	}
}

func resolveEffectSFX(spec worldEffectSpec, effect worldEffect) []string {
	out := make([]string, 0, len(spec.sfx))
	for i, path := range spec.sfx {
		if strings.Contains(path, "%d") && spec.sfxRandMax >= spec.sfxRandMin && spec.sfxRandMin > 0 {
			span := spec.sfxRandMax - spec.sfxRandMin + 1
			index := spec.sfxRandMin + int(deterministicUnit(effect, 701+i)*float64(span))
			if index > spec.sfxRandMax {
				index = spec.sfxRandMax
			}
			path = strings.Replace(path, "%d", strconv.Itoa(index), 1)
		}
		out = append(out, path)
	}
	return out
}

func (m *WorldMode) addWorldEffectAtCellIfMissing(ctx client.Context, effectID int, x, y int, starts time.Time) bool {
	return m.addWorldEffectAtCellDurationSizeIfMissing(ctx, effectID, 0, x, y, starts, 0, 0)
}

func (m *WorldMode) addWorldEffectAtCellDurationSizeIfMissing(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64) bool {
	return m.addWorldEffectAtCellDurationSizeRotationIfMissing(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, 0)
}

func (m *WorldMode) addWorldEffectAtCellDurationSizeRotationIfMissing(ctx client.Context, effectID int, actorID uint32, x, y int, starts time.Time, durationOverride time.Duration, sizeOverride float64, rotationRadiansPerSecond float64) bool {
	now := time.Now()
	for _, effect := range m.worldEffects {
		if effect.effectID == effectID && effect.actorID == actorID && effect.x == x && effect.y == y && now.Before(effect.expires) {
			return false
		}
	}
	return m.addWorldEffectAtCellDurationSizeRotation(ctx, effectID, actorID, x, y, starts, durationOverride, sizeOverride, rotationRadiansPerSecond)
}

func (m *WorldMode) addWorldEffectBetweenAtDurationIfMissing(ctx client.Context, effectID int, actorID, targetID uint32, starts time.Time, durationOverride time.Duration) bool {
	now := time.Now()
	for _, effect := range m.worldEffects {
		if effect.effectID == effectID && effect.actorID == actorID && effect.targetID == targetID && now.Before(effect.expires) {
			return false
		}
	}
	return m.addWorldEffectBetweenAtDuration(ctx, effectID, actorID, targetID, starts, durationOverride)
}

func (m *WorldMode) addSkillCastEffects(ctx client.Context, skillID uint16, property uint32, sourceID, targetID uint32, cellX, cellY int, duration time.Duration, starts time.Time, source string) {
	if duration <= 0 || sourceID == 0 {
		return
	}
	if targetID == 0 && (cellX != 0 || cellY != 0) {
		markerSize := skillCastGroundSampleSize(skillID)
		markerRotationSpeed := skillCastGroundSampleRotationRadiansPerSecond(skillID)
		if m.addWorldEffectAtCellDurationSizeRotationIfMissing(ctx, effectGroundSample, 0, cellX, cellY, starts, duration, markerSize, markerRotationSpeed) {
			glog.Debugf("skill cast ground marker source=%s skill=%d src=%d cell=%d,%d delay_ms=%d", source, skillID, sourceID, cellX, cellY, duration.Milliseconds())
		}
	}
	if m.addWorldEffectBetweenAtDurationIfMissing(ctx, effectCastRing, sourceID, 0, starts, duration) {
		glog.Debugf("skill cast circle source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
	}
	if targetID != 0 && targetID != sourceID {
		if m.addWorldEffectBetweenAtDurationIfMissing(ctx, effectLockOnTarget, targetID, 0, starts, duration) {
			glog.Debugf("skill cast target lockon source=%s skill=%d src=%d target=%d delay_ms=%d", source, skillID, sourceID, targetID, duration.Milliseconds())
		}
	}
	if skillHidesCastAura(skillID) {
		return
	}
	effectID := skillCastAuraEffectID(property)
	if effectID <= 0 {
		return
	}
	if m.addWorldEffectBetweenAtDurationIfMissing(ctx, effectID, sourceID, targetID, starts, duration) {
		glog.Debugf("skill cast aura source=%s skill=%d src=%d target=%d property=%d effect=%d delay_ms=%d", source, skillID, sourceID, targetID, property, effectID, duration.Milliseconds())
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

func effectAnchor(ctx client.Context, actorID uint32) (int, int, bool) {
	if ctx.World == nil {
		return 0, 0, false
	}
	if actorID == 0 || isLocalActor(ctx, actorID) {
		return ctx.World.Player.X, ctx.World.Player.Y, true
	}
	if actor, ok := ctx.World.Actors[actorID]; ok {
		return actor.X, actor.Y, true
	}
	return 0, 0, false
}

type itemEffectSpec struct {
	effectIDs         []int
	effectIDsOnCaster []int
	target            itemEffectTarget
}

type itemEffectTarget uint8

const (
	itemEffectTargetAck itemEffectTarget = iota
	itemEffectTargetMercenary
)

// This mirrors reference client's DB/Items/ItemEffect.js shape. Keep item effects as
// data so future imports can preserve array-valued effectId/effectIdOnCaster.
var itemEffectSpecs = buildItemEffects()

func buildItemEffects() map[uint16]itemEffectSpec {
	effects := map[uint16]itemEffectSpec{}
	add := func(effectID int, itemIDs ...uint16) {
		for _, itemID := range itemIDs {
			itemEffect := effects[itemID]
			itemEffect.effectIDs = append(itemEffect.effectIDs, effectID)
			effects[itemID] = itemEffect
		}
	}
	target := func(target itemEffectTarget, itemIDs ...uint16) {
		for _, itemID := range itemIDs {
			itemEffect := effects[itemID]
			itemEffect.target = target
			effects[itemID] = itemEffect
		}
	}
	add(effectPotionRed,
		501, 507, 512, 513, 515, 516, 545, 549, 557, 562, 563, 564, 565, 566, 567, 568, 569, 570, 571, 572,
		574, 575, 576, 577, 578, 579, 580, 581, 583, 584, 585, 586, 587, 588, 589, 590, 591, 592, 593, 594,
		595, 596, 597, 598, 607, 608, 663, 669, 680, 685, 11505, 11507, 11508, 11509, 11510, 11511, 11512,
		11513, 11514, 11515, 11516, 11517, 11519, 11520, 11521, 11522, 11525, 11526, 11527, 11528, 11529,
		11530, 11531, 11532, 11533, 11534, 11536, 11537, 11538, 11539, 11540, 11541, 11542, 11543, 11544,
		11545, 11546, 11547, 11550, 11551, 11552, 11553, 11554, 11555, 11567, 11568, 11570, 11577, 11578,
		11580, 11581, 11582, 11588, 11589, 11590, 11592, 11596, 11597, 11598, 11599, 11600, 11602, 11605,
		11701, 11702, 11703, 11704, 11705, 11706, 11707, 11708, 11709, 11710, 11711, 11712, 11713, 12021,
		12022, 12046, 12047, 12048, 12051, 12052, 12053, 12056, 12057, 12058, 12061, 12063, 12066, 12067,
		12068, 12101, 12102, 12131, 12133, 12188, 12192, 12195, 12196, 12197, 12202, 12203, 12204, 12205,
		12206, 12207, 12226, 12227, 12228, 12229, 12230, 12231, 12233, 12234, 12245, 12257, 12271, 12274,
		12275, 12292, 12293, 12331, 12332, 12335, 12336, 12337, 12436, 12601, 12624, 12704, 12709, 12711,
		12717, 12718, 12719, 12720, 12721, 12722, 12723, 12724, 12734, 12735, 12736, 12737, 12738, 14522,
		14523, 14524, 14551, 14552, 14553, 14575, 14576, 14577, 14578, 14579, 14580, 14672, 14673, 22567,
		22568, 22624, 22657, 22658, 22659, 22686, 22770, 22771, 22772, 22773, 22774, 22775, 22776, 22985)
	add(effectPotionOrange, 502, 582, 599, 11506, 11569)
	add(effectPotionYellow, 503, 508, 546, 11500, 11523, 11566, 11574, 11594)
	add(effectPotionWhite, 504, 509, 547, 11501, 11503, 11524, 11548, 11557, 11558, 11565, 11573, 12428)
	add(effectPotionBlue, 505, 510, 514, 11502, 11504, 11518, 11549, 11572, 11593)
	add(effectPotionGreen, 506, 511, 11571, 11595)
	add(effectFood,
		517, 518, 519, 520, 521, 522, 523, 525, 526, 528, 529, 530, 531, 532, 534, 535, 536, 537, 538, 539,
		540, 541, 542, 543, 544, 548, 550, 551, 552, 553, 554, 555, 556, 12017, 12184, 12185, 12298, 12404,
		12422, 12423, 12424, 12425, 12426, 12427, 12459, 12460, 12461, 12462, 12463, 12464, 12465, 12515,
		12516, 12517, 12518, 12519, 12520, 12521, 12529, 12531, 12648, 12676, 12679, 12680, 12683, 12684,
		12774, 12831, 14534, 14535, 14537, 14541, 14542, 14543, 14544, 14600, 14614, 16254, 16481, 22546)
	add(effectFoodBlue, 533)
	add(effectFoodChocolate, 558, 559, 560, 561, 573, 11535, 11583, 11584, 11585, 11586, 11587, 12062, 12322, 22555, 22556, 22557)
	add(effectItemFast, 645, 12241, 17263, 22542)
	add(effectItemFast2, 656, 12242, 17264, 22544)
	add(effectItemFast3, 657, 12243, 17265, 22543)
	add(effectItemSlow, 12016, 22545)
	add(effectBoxThunder, 12028)
	add(effectBoxResentment, 12030)
	add(effectBoxDrowsiness, 12031)
	add(effectBoxSunlight, 12033)
	add(effectResistPotion, 12118, 12119, 12120, 12121, 12299)
	add(effectStatFoodSTR, 12041, 12042, 12043, 12044, 12045, 12071, 12072, 12073, 12074, 12075)
	add(effectStatFoodINT, 12049, 12050, 12076, 12077, 12078, 12079, 12080, 14554, 14555, 14556)
	add(effectStatFoodVIT, 12054, 12055, 12081, 12082, 12083, 12084, 12085, 14557, 14558, 14559)
	add(effectStatFoodAGI, 12059, 12060, 12086, 12087, 12088, 12089, 12090, 14560, 14561, 14562)
	add(effectStatFoodDEX, 12064, 12065, 12091, 12092, 12093, 12094, 12095, 14563, 14564, 14565)
	add(effectStatFoodLUK, 12069, 12070, 12096, 12097, 12098, 12099, 12100, 14566, 14567, 14568)
	add(effectFirecracker, 12018)
	add(effectFirecracker7, 12326)
	add(effectFirecracker1, 12788)
	add(effectFirecracker2, 14546)
	add(effectFirecracker3, 14547)
	add(effectFirecracker4, 14548)
	add(effectFirecracker5, 14549)
	add(effectFirecracker6, 14550)
	add(effectItemAccel, 662, 12262)

	// reference client does not route Butterfly Wing through ItemEffect; keep Goro's
	// local map-change anticipation here but use the same table shape.
	add(effectTeleportation, 602)

	// eAthena/rAthena send the normal item-use ACK with the player's AID even
	// though these scripts apply to sd->md. Keep the robr effect IDs above, but
	// bind their visual target to the current mercenary when one is visible.
	target(itemEffectTargetMercenary, 12184, 12185, 12241, 12242, 12243)
	return effects
}

func itemUseEffectIDs(itemID uint16) []int {
	return itemUseEffectSpec(itemID).effectIDs
}

func itemUseEffectSpec(itemID uint16) itemEffectSpec {
	return itemEffectSpecs[itemID]
}

type skillEffectSpec struct {
	effectIDs              []int
	effectIDsOnCaster      []int
	beforeHitEffectIDs     []int
	beforeHitEffectIDsSelf []int
	hitEffectIDs           []int
	hitEffectIDsOnCaster   []int
	successEffectIDs       []int
	successEffectIDsSelf   []int
	beginCastEffectIDs     []int
	groundEffectIDs        []int
	hideCastBar            bool
	hideCastAura           bool
	action                 skillActionSpec
	forceGroundTarget      bool
	forceSelfTarget        bool
	passive                bool
}

type skillActorAction int

const skillActorActionNone skillActorAction = -1

const (
	skillActorActionIdle skillActorAction = iota
	skillActorActionSkill
	skillActorActionAttack
	skillActorActionAttack1
	skillActorActionAttack2
	skillActorActionAttack3
	skillActorActionPickup
	skillActorActionReadyFight
)

type skillActionSpec struct {
	defined  bool
	action   skillActorAction
	frame    int
	hasFrame bool
	length   int
	speed    time.Duration
	play     bool
	repeat   bool
	delay    time.Duration
	next     *skillActionSpec
}

var (
	idleSkillActionSpec       = newSkillActionSpec(skillActorActionIdle, true, nil)
	defaultSkillActionSpec    = newSkillActionSpec(skillActorActionSkill, false, &idleSkillActionSpec)
	attackSkillActionSpec     = newSkillActionSpec(skillActorActionAttack, false, &idleSkillActionSpec)
	readyFightSkillActionSpec = newSkillActionSpec(skillActorActionReadyFight, false, &idleSkillActionSpec)
)

func sonicBlowSkillActionSpec() skillActionSpec {
	next := newSkillActionSpec(skillActorActionReadyFight, true, nil)
	for hit := 8; hit >= 1; hit-- {
		spec := newSkillActionSpec(skillActorActionAttack, false, &next)
		if hit > 1 {
			spec.speed = 30 * time.Millisecond
		}
		next = spec
	}
	return next
}

func danceSkillActionSpec() skillActionSpec {
	spec := newSkillActionSpec(skillActorActionSkill, true, nil)
	spec.hasFrame = true
	spec.frame = 1
	spec.length = 3
	spec.speed = 250 * time.Millisecond
	return spec
}

func newSkillActionSpec(action skillActorAction, repeat bool, next *skillActionSpec) skillActionSpec {
	return skillActionSpec{
		defined: true,
		action:  action,
		frame:   0,
		repeat:  repeat,
		play:    true,
		next:    cloneSkillActionSpec(next),
	}
}

func cloneSkillActionSpec(spec *skillActionSpec) *skillActionSpec {
	if spec == nil {
		return nil
	}
	clone := *spec
	clone.next = cloneSkillActionSpec(spec.next)
	return &clone
}

func (s skillActionSpec) actionFamilyForActor(actor worldstate.Actor) int {
	return s.actionFamilyForActorWithResources(nil, actor)
}

func (s skillActionSpec) actionFamilyForActorWithResources(manager *res.Manager, actor worldstate.Actor) int {
	if s.action == skillActorActionNone {
		return -1
	}
	mercenary := actorIsMercenary(actor)
	if !res.HasPlayerJobToken(actorVisualJob(actor)) && !mercenary {
		return spriteActionNonPCAttack
	}
	switch s.action {
	case skillActorActionIdle:
		return spriteActionIdle
	case skillActorActionAttack:
		return attackActionFamilyForActorWithResources(manager, actor)
	case skillActorActionAttack1:
		return spriteActionPCAttack1
	case skillActorActionAttack2:
		return spriteActionPCAttack2
	case skillActorActionAttack3:
		return spriteActionPCAttack3
	case skillActorActionPickup:
		return spriteActionPickup
	case skillActorActionReadyFight:
		if mercenary {
			return spriteActionPCSkill
		}
		return spriteActionPCReadyFight
	default:
		return spriteActionPCSkill
	}
}

func (s skillActionSpec) actorAnimationForActorWithResources(manager *res.Manager, actor worldstate.Actor, started time.Time, duration time.Duration) actorAnimation {
	if !s.defined {
		s = defaultSkillActionSpec
	}
	if s.delay > 0 && !started.IsZero() {
		started = started.Add(s.delay)
	}
	anim := actorAnimation{
		actionFamily: s.actionFamilyForActorWithResources(manager, actor),
		started:      started,
		startDelay:   s.delay,
		duration:     duration,
		loop:         s.repeat,
		play:         s.play,
		hasPlay:      true,
	}
	if s.hasFrame {
		anim.frameOffset = s.frame
		anim.hasFrameOffset = true
		if !s.play {
			anim.fixedMotion = s.frame
			anim.hasFixedMotion = true
		}
	}
	if s.length > 0 {
		anim.length = s.length
		anim.hasLength = true
	}
	if s.speed > 0 {
		anim.speed = s.speed
		anim.hasSpeed = true
	}
	if s.next != nil && !s.next.isNaturalIdleFallback() {
		next := s.next.actorAnimationForActorWithResources(manager, actor, time.Time{}, duration)
		anim.next = &next
	}
	return anim
}

func (s skillActionSpec) isNaturalIdleFallback() bool {
	return s.defined && s.action == skillActorActionIdle && s.repeat && s.play && s.next == nil
}

func skillEffectSpecFor(skillID uint16) skillEffectSpec {
	return importedSkillEffectSpec(skillID)
}

func importedSkillEffectSpec(skillID uint16) skillEffectSpec {
	out := skillEffectSpec{}
	if spec, ok := db.SkillEffects[skillID]; ok {
		out.effectIDs = copyIntSlice(spec.EffectIDs)
		out.effectIDsOnCaster = copyIntSlice(spec.EffectIDsOnCaster)
		out.beforeHitEffectIDs = copyIntSlice(spec.BeforeHitEffectIDs)
		out.beforeHitEffectIDsSelf = copyIntSlice(spec.BeforeHitEffectIDsSelf)
		out.hitEffectIDs = copyIntSlice(spec.HitEffectIDs)
		out.hitEffectIDsOnCaster = copyIntSlice(spec.HitEffectIDsOnCaster)
		out.successEffectIDs = copyIntSlice(spec.SuccessEffectIDs)
		out.successEffectIDsSelf = copyIntSlice(spec.SuccessEffectIDsSelf)
		out.beginCastEffectIDs = copyIntSlice(spec.BeginCastEffectIDs)
		out.groundEffectIDs = copyIntSlice(spec.GroundEffectIDs)
		out.hideCastBar = spec.HideCastBar
		out.hideCastAura = spec.HideCastAura
	}
	if action, ok := importedSkillActionSpec(skillID); ok {
		out.action = action
	}
	return out
}

func importedSkillActionSpec(skillID uint16) (skillActionSpec, bool) {
	if skillID == db.SkillNJTatamigaeshi {
		spec := newSkillActionSpec(skillActorActionPickup, false, nil)
		spec.hasFrame = true
		spec.frame = 1
		spec.play = false
		return spec, true
	}
	if skillID == db.SkillACShower {
		spec := newSkillActionSpec(skillActorActionAttack, false, &readyFightSkillActionSpec)
		spec.speed = 50 * time.Millisecond
		return spec, true
	}
	if skillID == db.SkillASSonicblow {
		return sonicBlowSkillActionSpec(), true
	}
	switch db.SkillActions[skillID] {
	case db.SkillActionNone:
		return skillActionSpec{defined: true, action: skillActorActionNone}, true
	case db.SkillActionIdle:
		return idleSkillActionSpec, true
	case db.SkillActionAttack:
		return attackSkillActionSpec, true
	case db.SkillActionAttack1:
		return newSkillActionSpec(skillActorActionAttack1, false, &idleSkillActionSpec), true
	case db.SkillActionAttack2:
		return newSkillActionSpec(skillActorActionAttack2, false, &idleSkillActionSpec), true
	case db.SkillActionAttack3:
		return newSkillActionSpec(skillActorActionAttack3, false, &idleSkillActionSpec), true
	case db.SkillActionSkill, db.SkillActionAction:
		return defaultSkillActionSpec, true
	case db.SkillActionAttackFixedFrame:
		spec := newSkillActionSpec(skillActorActionAttack, false, nil)
		spec.hasFrame = true
		spec.frame = 0
		spec.play = false
		return spec, true
	case db.SkillActionPickup:
		return newSkillActionSpec(skillActorActionPickup, false, &idleSkillActionSpec), true
	case db.SkillActionReadyfight:
		return readyFightSkillActionSpec, true
	case db.SkillActionDance:
		return danceSkillActionSpec(), true
	default:
		return skillActionSpec{}, false
	}
}

func copyIntSlice(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	return append([]int(nil), in...)
}

func skillSuccessEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).successEffectIDs
}

func skillSuccessEffectSelfIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).successEffectIDsSelf
}

func skillEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).effectIDs
}

func skillEffectOnCasterIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).effectIDsOnCaster
}

func skillBeginEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beginCastEffectIDs
}

func skillBeforeHitEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beforeHitEffectIDs
}

func skillBeforeHitEffectSelfIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).beforeHitEffectIDsSelf
}

func skillGroundEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).groundEffectIDs
}

func skillHidesCastBar(skillID uint16) bool {
	return skillEffectSpecFor(skillID).hideCastBar
}

func skillHidesCastAura(skillID uint16) bool {
	return skillEffectSpecFor(skillID).hideCastAura
}

func skillForcesGroundTarget(skillID uint16) bool {
	return skillEffectSpecFor(skillID).forceGroundTarget
}

func skillForcesSelfTarget(skillID uint16) bool {
	return skillEffectSpecFor(skillID).forceSelfTarget
}

func skillForcesPassive(skillID uint16) bool {
	return skillEffectSpecFor(skillID).passive
}

type skillUnitEffectSpec struct {
	effectIDs []int
}

// Mostly mirrors reference client's DB/Skills/SkillUnit.js: unit id -> effect id.
var skillUnitEffectSpecs = map[uint16]skillUnitEffectSpec{
	126: {effectIDs: []int{effectSafetyWall}},             // UNT_SAFETYWALL -> EF_GLASSWALL2
	127: {effectIDs: []int{effectFireWall}},               // UNT_FIREWALL -> EF_FIREWALL
	128: {effectIDs: []int{effectPortal}},                 // UNT_WARPPORTAL -> EF_PORTAL2
	129: {effectIDs: []int{effectReadyPortal}},            // UNT_PRE_WARPPORTAL -> EF_READYPORTAL2
	131: {effectIDs: []int{effectBottomSanc}},             // UNT_SANCTUARY -> EF_BOTTOM_SANC
	132: {effectIDs: []int{effectBottomMagnus}},           // UNT_MAGNUS -> EF_BOTTOM_MAG
	133: {effectIDs: []int{effectPneuma}},                 // UNT_PNEUMA -> EF_PNEUMA
	135: {effectIDs: []int{effectFirePillarOn}},           // UNT_FIREPILLAR_WAITING -> EF_FIREPILLARON
	141: {effectIDs: []int{effectIceWall}},                // UNT_ICEWALL -> EF_ICEWALL
	142: {effectIDs: []int{effectQuagmire}},               // UNT_QUAGMIRE -> EF_QUAGMIRE
	146: {effectIDs: []int{effectVenomDust2}},             // UNT_VENOMDUST -> EF_VENOMDUST2
	154: {effectIDs: []int{effectBottomVolcano}},          // UNT_VOLCANO -> EF_BOTTOM_VO
	155: {effectIDs: []int{effectBottomDeluge}},           // UNT_DELUGE -> EF_BOTTOM_DE
	156: {effectIDs: []int{effectBottomViolent}},          // UNT_VIOLENTGALE -> EF_BOTTOM_VI
	157: {effectIDs: []int{effectBottomLand}},             // UNT_LANDPROTECTOR -> EF_BOTTOM_LA
	158: {effectIDs: []int{effectBottomLullabyGround}},    // UNT_LULLABY -> 278_ground
	159: {effectIDs: []int{effectBottomRichKimGround}},    // UNT_RICHMANKIM -> 279_ground
	160: {effectIDs: []int{effectBottomChaosGround}},      // UNT_ETERNALCHAOS -> 280_ground
	161: {effectIDs: []int{effectBottomDrumGround}},       // UNT_DRUMBATTLEFIELD -> 281_ground
	162: {effectIDs: []int{effectBottomNibelungGround}},   // UNT_RINGNIBELUNGEN -> 282_ground
	163: {effectIDs: []int{effectBottomRokiGround}},       // UNT_ROKISWEIL -> 283_ground
	164: {effectIDs: []int{effectBottomAbyssGround}},      // UNT_INTOABYSS -> 284_ground
	165: {effectIDs: []int{effectBottomSiegGround}},       // UNT_SIEGFRIED -> 285_ground
	166: {effectIDs: []int{effectBottomDissonanceGround}}, // UNT_DISSONANCE -> 277_ground
	167: {effectIDs: []int{effectBottomWhistleGround}},    // UNT_WHISTLE -> 286_ground
	168: {effectIDs: []int{effectBottomSinXGround}},       // UNT_ASSASSINCROSS -> 287_ground
	169: {effectIDs: []int{effectBottomBragiGround}},      // UNT_POEMBRAGI -> 288_ground
	170: {effectIDs: []int{effectBottomAppleGround}},      // UNT_APPLEIDUN -> 289_ground
	171: {effectIDs: []int{effectBottomUglyDanceGround}},  // UNT_UGLYDANCE -> 290_ground
	172: {effectIDs: []int{effectBottomHummingGround}},    // UNT_HUMMING -> 291_ground
	173: {effectIDs: []int{effectBottomForgetGround}},     // UNT_DONTFORGETME -> 292_ground
	174: {effectIDs: []int{effectBottomFortuneGround}},    // UNT_FORTUNEKISS -> 293_ground
	175: {effectIDs: []int{effectBottomServiceGround}},    // UNT_SERVICEFORYOU -> 294_ground
	177: {effectIDs: []int{effectDemonstration}},          // UNT_DEMONSTRATION -> EF_DEMONSTRATION
	179: {effectIDs: []int{effectGospelGround}},           // UNT_GOSPEL -> 370_ground
	180: {effectIDs: []int{effectBottomBasilica}},         // UNT_BASILICA -> EF_BOTTOM_BASILICA
	181: {effectIDs: []int{effectMoonlit}},                // UNT_MOONLIT -> 394_ground
	182: {effectIDs: []int{effectFogWallGround}},          // UNT_FOGWALL -> 405_ground
	183: {effectIDs: []int{effectBottomSpider}},           // UNT_SPIDERWEB -> EF_BOTTOM_SPIDER
	184: {effectIDs: []int{effectGravitation}},            // UNT_GRAVITATION -> 522_ground
	185: {effectIDs: []int{effectBottomHermode}},          // UNT_HERMODE -> EF_BOTTOM_HERMODE
	188: {effectIDs: []int{effectTatami}},                 // UNT_TATAMIGAESHI -> EF_TATAMI
	189: {effectIDs: []int{effectKaen}},                   // UNT_KAEN -> EF_KAEN
	199: {effectIDs: []int{effectBottomEvilLand}},         // UNT_EVILLAND -> EF_BOTTOM_EVILLAND
	202: {effectIDs: []int{effectGlassWall3}},             // UNT_EPICLESIS -> EF_GLASSWALL3
	203: {effectIDs: []int{effectEarthWall}},              // UNT_EARTHSTRAIN -> EF_EARTHWALL
	204: {effectIDs: []int{effectBottomManhole}},          // UNT_MANHOLE -> EF_BOTTOM_MANHOLE
	205: {effectIDs: []int{effectForestLight6}},           // UNT_DIMENSIONDOOR -> EF_FORESTLIGHT6
	206: {effectIDs: []int{effectBottomAni}},              // UNT_CHAOSPANIC -> EF_BOTTOM_ANI
	207: {effectIDs: []int{effectBottomMaelstrom}},        // UNT_MAELSTROM -> EF_BOTTOM_MAELSTROM
	208: {effectIDs: []int{effectBottomBloodyLust}},       // UNT_BLOODYLUST -> EF_BOTTOM_BLOODYLUST
	218: {effectIDs: []int{effectBotReverb}},              // UNT_REVERBERATION -> EF_BOT_REVERB
	220: {effectIDs: []int{effectFireWall2}},              // UNT_FIREWALK -> EF_FIREWALL2
	221: {effectIDs: []int{effectShockwave2}},             // UNT_ELECTRICWALK -> EF_SHOCKWAVE2
	222: {effectIDs: []int{effectBotReverb2}},             // UNT_NETHERWORLD -> EF_BOT_REVERB2
}

func skillUnitEffectIDs(unitID uint16) []int {
	return skillUnitEffectSpecs[unitID].effectIDs
}

func skillCastAuraEffectID(property uint32) int {
	switch property {
	case 1:
		return effectBeginSpell2
	case 2:
		return effectBeginSpell5
	case 3:
		return effectBeginSpell3
	case 4:
		return effectBeginSpell4
	case 5:
		return effectBeginSpell7
	case 6, 8:
		return effectBeginSpell6
	default:
		return effectBeginSpell
	}
}

func skillHitEffectIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).hitEffectIDs
}

func skillHitEffectOnCasterIDs(skillID uint16) []int {
	return skillEffectSpecFor(skillID).hitEffectIDsOnCaster
}

func skillAction(skillID uint16) skillActionSpec {
	spec := skillEffectSpecFor(skillID).action
	if !spec.defined {
		return defaultSkillActionSpec
	}
	return spec
}

func worldEffectSpecForID(effectID int) (worldEffectSpec, bool) {
	if effectID == effectEmotion {
		return worldEffectSpec{
			components: []worldEffectComponent{{
				kind:           effectComponentSPR,
				spriteFile:     "emotion",
				attachedEntity: true,
				spriteYOffset:  -100,
			}},
		}, true
	}
	spec, ok := db.EffectSpecs[effectID]
	if !ok {
		return worldEffectSpec{}, false
	}
	return convertDBWorldEffectSpec(spec), true
}

func effectDetachesLocalActor(effectID int) bool {
	spec, ok := worldEffectSpecForID(effectID)
	return ok && spec.detachLocalActor
}

func convertDBWorldEffectSpec(spec db.EffectSpec) worldEffectSpec {
	out := worldEffectSpec{
		duration:         spec.Duration,
		cameraShake:      spec.CameraShake,
		cameraShakeDelay: spec.CameraShakeDelay,
		detachLocalActor: spec.DetachLocalActor,
		sfxRandMin:       spec.SFXRandMin,
		sfxRandMax:       spec.SFXRandMax,
	}
	if len(spec.SFX) > 0 {
		out.sfx = append([]string(nil), spec.SFX...)
	}
	if len(spec.SFXDelays) > 0 {
		out.sfxDelays = append([]time.Duration(nil), spec.SFXDelays...)
	}
	if len(spec.Components) > 0 {
		out.components = make([]worldEffectComponent, 0, len(spec.Components))
		for _, component := range spec.Components {
			out.components = append(out.components, convertDBWorldEffectComponent(component))
		}
	}
	return out
}

func convertDBWorldEffectComponent(component db.EffectComponent) worldEffectComponent {
	return worldEffectComponent{
		kind:               convertDBEffectComponentKind(component.Kind),
		funcAdapter:        effectFuncAdapterForName(component.FuncName),
		funcName:           component.FuncName,
		color:              component.Color,
		duration:           component.Duration,
		durationRandMin:    component.DurationRandMin,
		durationRandMax:    component.DurationRandMax,
		delay:              component.Delay,
		duplicateDelay:     component.DuplicateDelay,
		delayOffsetDelta:   component.DelayOffsetDelta,
		repeat:             component.Repeat,
		repeatDelay:        component.RepeatDelay,
		strFile:            component.STRFile,
		strMinFile:         component.STRMinFile,
		strRandMin:         component.STRRandMin,
		strRandMax:         component.STRRandMax,
		attachedEntity:     component.AttachedEntity,
		renderBefore:       component.RenderBefore,
		texturePath:        component.TexturePath,
		textureName:        component.TextureName,
		textureFile:        component.TextureFile,
		textureFiles:       append([]string(nil), component.TextureFiles...),
		frameDelay:         component.FrameDelay,
		spriteFile:         component.SpriteFile,
		shadowTexture:      component.ShadowTexture,
		spriteHead:         component.SpriteHead,
		spriteDirection:    component.SpriteDirection,
		spriteRepeat:       component.SpriteRepeat,
		spriteStopAtEnd:    component.SpriteStopAtEnd,
		spriteFrame:        component.SpriteFrame,
		spriteDelay:        component.SpriteDelay,
		spriteXOffset:      component.SpriteXOffset,
		spriteYOffset:      component.SpriteYOffset,
		fromSrc:            component.FromSrc,
		toSrc:              component.ToSrc,
		arc:                component.Arc,
		retreat:            component.Retreat,
		alphaMax:           component.AlphaMax,
		alphaMaxDelta:      component.AlphaMaxDelta,
		sparkling:          component.Sparkling,
		sparkNumber:        component.SparkNumber,
		fade:               component.Fade,
		fadeIn:             component.FadeIn,
		fadeOut:            component.FadeOut,
		rotate:             component.Rotate,
		rotateWithCamera:   component.RotateWithCamera,
		fixedPerspective:   component.FixedPerspective,
		rotateToTarget:     component.RotateToTarget,
		worldSizedSprite:   component.WorldSizedSprite,
		animation:          component.Animation,
		bottomSize:         component.BottomSize,
		topSize:            component.TopSize,
		height:             component.Height,
		posX:               component.PosX,
		posY:               component.PosY,
		posZ:               component.PosZ,
		posXEnd:            component.PosXEnd,
		posYEnd:            component.PosYEnd,
		posZEnd:            component.PosZEnd,
		posXRand:           component.PosXRand,
		posYRand:           component.PosYRand,
		posZRand:           component.PosZRand,
		posXStartRand:      component.PosXStartRand,
		posYStartRand:      component.PosYStartRand,
		posZStartRand:      component.PosZStartRand,
		posXStartMiddle:    component.PosXStartMiddle,
		posYStartMiddle:    component.PosYStartMiddle,
		posZStartMiddle:    component.PosZStartMiddle,
		posXEndRand:        component.PosXEndRand,
		posYEndRand:        component.PosYEndRand,
		posZEndRand:        component.PosZEndRand,
		posXEndMiddle:      component.PosXEndMiddle,
		posYEndMiddle:      component.PosYEndMiddle,
		posZEndMiddle:      component.PosZEndMiddle,
		posXSmooth:         component.PosXSmooth,
		posYSmooth:         component.PosYSmooth,
		posZSmooth:         component.PosZSmooth,
		sizeStart:          component.SizeStart,
		sizeEnd:            component.SizeEnd,
		sizeRand:           component.SizeRand,
		sizeStartX:         component.SizeStartX,
		sizeStartY:         component.SizeStartY,
		sizeEndX:           component.SizeEndX,
		sizeEndY:           component.SizeEndY,
		sizeStartXRandMin:  component.SizeStartXRandMin,
		sizeStartXRandMax:  component.SizeStartXRandMax,
		sizeStartYRandMin:  component.SizeStartYRandMin,
		sizeStartYRandMax:  component.SizeStartYRandMax,
		sizeEndXRandMin:    component.SizeEndXRandMin,
		sizeEndXRandMax:    component.SizeEndXRandMax,
		sizeEndYRandMin:    component.SizeEndYRandMin,
		sizeEndYRandMax:    component.SizeEndYRandMax,
		sizeRandX:          component.SizeRandX,
		sizeRandY:          component.SizeRandY,
		sizeRandXMiddle:    component.SizeRandXMiddle,
		sizeRandYMiddle:    component.SizeRandYMiddle,
		sizeDelta:          component.SizeDelta,
		sizeSmooth:         component.SizeSmooth,
		angleStart:         component.AngleStart,
		angleEnd:           component.AngleEnd,
		angleX:             component.AngleX,
		angleY:             component.AngleY,
		angleZ:             component.AngleZ,
		angleRandMin:       component.AngleRandMin,
		angleRandMax:       component.AngleRandMax,
		circlePattern:      component.CirclePattern,
		circleInnerSize:    component.CircleInnerSize,
		circleOuterRandMin: component.CircleOuterRandMin,
		circleOuterRandMax: component.CircleOuterRandMax,
		orbitRadiusX:       component.OrbitRadiusX,
		orbitRadiusY:       component.OrbitRadiusY,
		orbitRadiusZ:       component.OrbitRadiusZ,
		orbitRotations:     component.OrbitRotations,
		orbitPhase:         component.OrbitPhase,
		orbitPhaseDelta:    component.OrbitPhaseDelta,
		orbitClockwise:     component.OrbitClockwise,
		totalCircleSides:   component.TotalCircleSides,
		circleSides:        component.CircleSides,
		duplicate:          component.Duplicate,
		angleZRandom:       component.AngleZRandom,
		blendMode:          component.BlendMode,
		blendAdditive:      component.BlendAdditive,
		overlay:            component.Overlay,
		quadHornHeightMin:  component.QuadHornHeightMin,
		quadHornHeightMax:  component.QuadHornHeightMax,
		quadHornOffsetXMin: component.QuadHornOffsetXMin,
		quadHornOffsetXMax: component.QuadHornOffsetXMax,
		quadHornOffsetYMin: component.QuadHornOffsetYMin,
		quadHornOffsetYMax: component.QuadHornOffsetYMax,
		quadHornOffsetZ:    component.QuadHornOffsetZ,
		quadHornBottomMin:  component.QuadHornBottomMin,
		quadHornBottomMax:  component.QuadHornBottomMax,
		quadHornRotateXMin: component.QuadHornRotateXMin,
		quadHornRotateXMax: component.QuadHornRotateXMax,
		quadHornRotateYMin: component.QuadHornRotateYMin,
		quadHornRotateYMax: component.QuadHornRotateYMax,
		quadHornRotateZMin: component.QuadHornRotateZMin,
		quadHornRotateZMax: component.QuadHornRotateZMax,
		quadHornAnimSpeed:  component.QuadHornAnimSpeed,
		quadHornAnimOut:    component.QuadHornAnimOut,
	}
}

func convertDBEffectComponentKind(kind db.EffectComponentKind) effectComponentKind {
	switch kind {
	case db.EffectComponentSTR:
		return effectComponentSTR
	case db.EffectComponentCylinder:
		return effectComponentCylinder
	case db.EffectComponent2D:
		return effectComponent2D
	case db.EffectComponent3D:
		return effectComponent3D
	case db.EffectComponentSPR:
		return effectComponentSPR
	case db.EffectComponentFUNC:
		return effectComponentFUNC
	case db.EffectComponentQuadHorn:
		return effectComponentQuadHorn
	default:
		return 0
	}
}

func effectFuncAdapterForName(name string) effectFuncAdapter {
	switch name {
	case "MagicTarget":
		return effectFuncGroundSample
	case "CastRing":
		return effectFuncCastRing
	case "LockOnTarget":
		return effectFuncLockOnTarget
	case "Level99Aura":
		return effectFuncLevel99Aura
	case "GroundAura":
		return effectFuncGroundAura
	case "Level99Bubble":
		return effectFuncLevel99Bubble
	case "PropertyGround":
		return effectFuncPropertyGround
	case "LandProtectorGround":
		return effectFuncLandProtectorGround
	case "SpiritSphere":
		return effectFuncSpiritSphere
	case "FlatColorTile":
		return effectFuncFlatColorTile
	case "GroundTexture", "SpiderWeb":
		return effectFuncGroundTexture
	case "EffectBodyColor":
		return effectFuncBodyColor
	case "MapPillar":
		return effectFuncMapPillar
	default:
		return effectFuncUnknown
	}
}

func (m *WorldMode) drawWorldEffects(screen *render.Frame, ctx client.Context, projection sceneProjection, now time.Time) {
	if len(m.worldEffects) == 0 || screen == nil || ctx.World == nil {
		return
	}
	if m.whitePixel == nil {
		m.whitePixel = render.NewImage(1, 1)
		m.whitePixel.Fill(color.White)
	}
	active := m.worldEffects[:0]
	for _, effect := range m.worldEffects {
		if now.After(effect.expires) {
			continue
		}
		spec, ok := worldEffectSpecForID(effect.effectID)
		if !ok {
			continue
		}
		active = append(active, effect)
		if now.Before(effect.starts) {
			continue
		}
		x, y := float64(effect.x), float64(effect.y)
		if actor, ok := ctx.World.Actors[effect.actorID]; ok {
			x, y = actorRenderPosition(actor, now)
		} else if isLocalActor(ctx, effect.actorID) {
			x, y = actorRenderPosition(ctx.World.Player, now)
		}
		worldX := cellCenter(x)
		worldY := cellCenter(y)
		worldZ := terrainHeightAt(ctx.World, x, y) + 0.07
		for index, component := range spec.components {
			if effect.persistent {
				component.repeat = true
			}
			componentDuration := m.worldEffectResolvedComponentDuration(ctx, spec, component)
			if effect.duration > 0 && !component.repeat {
				componentDuration = effect.duration
			}
			progress := worldEffectComponentProgressForDraw(effect.starts, component, componentDuration, now)
			if progress >= 1 {
				continue
			}
			m.drawWorldEffectComponent(screen, ctx, projection, effect, component, index, worldX, worldY, worldZ, progress, componentDuration, now)
		}
	}
	m.worldEffects = active
}

func (m *WorldMode) worldEffectResolvedComponentDuration(ctx client.Context, spec worldEffectSpec, component worldEffectComponent) time.Duration {
	duration := worldEffectComponentDuration(spec, component)
	if component.kind == effectComponentSTR {
		if str := m.loadWorldEffectSTR(ctx.Resources, resolveEffectSTRFile(component, worldEffect{}, lessEffectsEnabled(ctx)), component.texturePath); str != nil {
			duration = strEffectDuration(str, duration)
		}
	}
	if component.kind == effectComponentSPR && component.duration <= 0 && !component.spriteRepeat {
		if view := m.effectSpriteView(ctx.Resources, component.spriteFile); view != nil && len(view.act.Actions) > 0 {
			actionIndex := component.spriteFrame
			if actionIndex < 0 || actionIndex >= len(view.act.Actions) {
				actionIndex = 0
			}
			duration = actionAnimationDuration(view.act.Actions[actionIndex], duration)
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

func worldEffectComponentDuration(spec worldEffectSpec, component worldEffectComponent) time.Duration {
	duration := spec.duration
	if component.duration > 0 {
		duration = component.duration
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

func worldEffectComponentStartOffset(component worldEffectComponent, duplicateIndex int) time.Duration {
	return component.delay + worldEffectComponentDuplicateStartOffset(component, duplicateIndex)
}

func worldEffectComponentDuplicateStartOffset(component worldEffectComponent, duplicateIndex int) time.Duration {
	if duplicateIndex < 0 {
		duplicateIndex = 0
	}
	return time.Duration(duplicateIndex) * (component.duplicateDelay + component.delayOffsetDelta)
}

func worldEffectComponentMaxStartOffset(component worldEffectComponent) time.Duration {
	duplicates := maxInt(component.duplicate, 1)
	maxOffset := time.Duration(0)
	for i := 0; i < duplicates; i++ {
		offset := worldEffectComponentStartOffset(component, i)
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	return maxOffset
}

func worldEffectComponentProgress(starts time.Time, duration time.Duration, now time.Time) float64 {
	if duration <= 0 {
		return 1
	}
	return clampFloat(float64(now.Sub(starts))/float64(duration), 0, 1)
}

func worldEffectComponentProgressForDraw(starts time.Time, component worldEffectComponent, duration time.Duration, now time.Time) float64 {
	if !component.repeat {
		return worldEffectComponentProgress(starts, duration, now)
	}
	if duration <= 0 {
		return 1
	}
	componentStart := starts.Add(component.delay)
	if now.Before(componentStart) {
		return 1
	}
	cycle := duration + component.repeatDelay
	if cycle <= 0 {
		cycle = duration
	}
	cycleElapsed := now.Sub(componentStart) % cycle
	if cycleElapsed >= duration {
		return 1
	}
	return clampFloat(float64(cycleElapsed)/float64(duration), 0, 1)
}

func worldEffectComponentDuplicateProgressForDraw(starts time.Time, component worldEffectComponent, duplicateIndex int, duration time.Duration, now time.Time) (float64, bool) {
	starts = starts.Add(worldEffectComponentDuplicateStartOffset(component, duplicateIndex))
	if component.repeat {
		progress := worldEffectComponentProgressForDraw(starts, component, duration, now)
		return progress, progress < 1
	}
	starts = starts.Add(component.delay)
	if now.Before(starts) {
		return 0, false
	}
	progress := worldEffectComponentProgress(starts, duration, now)
	return progress, progress < 1
}

func (m *WorldMode) drawWorldEffectComponent(screen *render.Frame, ctx client.Context, projection sceneProjection, effect worldEffect, component worldEffectComponent, componentIndex int, worldX, worldY, worldZ, progress float64, componentDuration time.Duration, now time.Time) {
	switch component.kind {
	case effectComponentSTR:
		m.drawSTREffect(screen, ctx, projection, component, effect, worldX, worldY, worldZ, now)
	case effectComponentCylinder:
		m.drawCylinderEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, componentDuration, now)
	case effectComponent2D:
		m.draw2DEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, now)
	case effectComponent3D:
		m.draw3DEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, now)
	case effectComponentSPR:
		m.drawSPREffect(screen, ctx, projection, effect, component, worldX, worldY, worldZ, now)
	case effectComponentFUNC:
		m.drawFuncEffect(screen, ctx, projection, effect, component, componentIndex, worldX, worldY, worldZ, progress, now)
	case effectComponentQuadHorn:
		m.drawQuadHornEffect(screen, ctx, effect, component, componentIndex, worldX, worldY, worldZ, progress, componentDuration)
	default:
	}
}

func effectComponentAlpha(progress float64, component worldEffectComponent) float64 {
	alphaMax := component.alphaMax
	if alphaMax <= 0 {
		alphaMax = 1
	}
	if component.fade {
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
	case component.fadeIn && progress < 0.25:
		return progress / 0.25 * alphaMax
	case component.fadeOut && progress > 0.75:
		return (1 - progress) / 0.25 * alphaMax
	default:
		return alphaMax
	}
}

func effectComponentTint(component worldEffectComponent, alpha float64) color.RGBA {
	tint := component.color
	if tint.A == 0 {
		tint = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	tint.A = uint8(clampFloat(alpha, 0, 1) * 255)
	return tint
}

func deterministicAngle(effect worldEffect, salt int) float64 {
	return deterministicUnit(effect, salt) * 2 * math.Pi
}

func deterministicSigned(effect worldEffect, salt int) float64 {
	return deterministicUnit(effect, salt)*2 - 1
}

func deterministicDurationRange(effect worldEffect, salt int, min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(deterministicUnit(effect, salt)*float64(max-min))
}

func deterministicFloatRange(effect worldEffect, salt int, min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + deterministicUnit(effect, salt)*(max-min)
}

func deterministicUnit(effect worldEffect, salt int) float64 {
	value := uint32(effect.effectID*1103515245) ^ effect.actorID ^ uint32(effect.starts.UnixNano()) ^ uint32(salt*2654435761)
	value ^= value >> 16
	value *= 2246822519
	value ^= value >> 13
	return float64(value&0xFFFFFF) / float64(0xFFFFFF)
}

func (m *WorldMode) effectFileTexture(manager *res.Manager, path string) *render.Image {
	return m.effectFileTextureWithBlackKey(manager, path, true)
}

func (m *WorldMode) effectFileTextureWithBlackKey(manager *res.Manager, path string, blackKey bool) *render.Image {
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

func (m *WorldMode) effectTexture(manager *res.Manager, name string) *render.Image {
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
