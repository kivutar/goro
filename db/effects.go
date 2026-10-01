package db

import (
	"image/color"
	"math"
	"strconv"
	"time"
)

const (
	ReferenceActiveEffectTableEntries = 607
	ReferenceNumericEffectConstants   = 1147
)

const (
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
	effectMagicPower     = SkillEffectMagicPower
	effectGravitation    = SkillEffectGravitationGround
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
	effectWhitePulse             = SkillEffectWhitePulse
	effectSpearProjectile        = SkillEffectSpearProjectile
	effectSpiralBeforeCast       = SkillEffectSpiralBeforeCast
	effectSpearHitSound          = SkillEffectSpearHitSound
	effectEnemyHitNormal1        = SkillEffectEnemyHitNormal1
	effectQuake                  = SkillEffectQuake
	effectAnkleSnareGround       = SkillEffectAnkleSnareGround
	effectSharpShootingCast      = SkillEffectSharpShootingCast
	effectAdrenalineCast         = SkillEffectAdrenalineCast
	effectMaximizeSounds         = SkillEffectMaximizeSounds
	effectGreedSound             = SkillEffectGreedSound
	effectGospelGround           = SkillEffectGospelGround
	effectShieldProjectile       = SkillEffectShieldProjectile
	effectFogWallGround          = SkillEffectFogWallGround
	effectHermodeMusic           = SkillEffectHermodeMusic
	effectBottomDissonanceGround = SkillEffectBottomDissonanceGround
	effectBottomLullabyGround    = SkillEffectBottomLullabyGround
	effectBottomRichKimGround    = SkillEffectBottomRichKimGround
	effectBottomChaosGround      = SkillEffectBottomChaosGround
	effectBottomDrumGround       = SkillEffectBottomDrumGround
	effectBottomNibelungGround   = SkillEffectBottomNibelungGround
	effectBottomRokiGround       = SkillEffectBottomRokiGround
	effectBottomAbyssGround      = SkillEffectBottomAbyssGround
	effectBottomSiegGround       = SkillEffectBottomSiegGround
	effectBottomWhistleGround    = SkillEffectBottomWhistleGround
	effectBottomSinXGround       = SkillEffectBottomSinXGround
	effectBottomBragiGround      = SkillEffectBottomBragiGround
	effectBottomAppleGround      = SkillEffectBottomAppleGround
	effectBottomUglyDanceGround  = SkillEffectBottomUglyDanceGround
	effectBottomHummingGround    = SkillEffectBottomHummingGround
	effectBottomForgetGround     = SkillEffectBottomForgetGround
	effectBottomFortuneGround    = SkillEffectBottomFortuneGround
	effectBottomServiceGround    = SkillEffectBottomServiceGround
	effectGunslingerCoinSound    = SkillEffectGunslingerCoinSound
)

const EffectPixelRatio = 1.0 / 35.0

type EffectComponentKind int

const (
	EffectComponentSTR EffectComponentKind = iota + 1
	EffectComponentCylinder
	EffectComponent2D
	EffectComponent3D
	EffectComponentSPR
	EffectComponentFUNC
	EffectComponentQuadHorn
)

type EffectSpec struct {
	Duration         time.Duration
	CameraShake      time.Duration
	CameraShakeDelay time.Duration
	DetachLocalActor bool
	SFX              []string
	SFXDelays        []time.Duration
	SFXRandMin       int
	SFXRandMax       int
	Components       []EffectComponent
}

type EffectComponent struct {
	Kind               EffectComponentKind
	FuncName           string
	Color              color.RGBA
	Duration           time.Duration
	DurationRandMin    time.Duration
	DurationRandMax    time.Duration
	Delay              time.Duration
	DuplicateDelay     time.Duration
	DelayOffsetDelta   time.Duration
	Repeat             bool
	RepeatDelay        time.Duration
	STRFile            string
	STRMinFile         string
	STRRandMin         int
	STRRandMax         int
	AttachedEntity     bool
	RenderBefore       bool // robr renderBeforeEntities
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
	SpriteFrame        int
	SpriteDelay        time.Duration
	SpriteXOffset      float64
	SpriteYOffset      float64
	FromSrc            bool
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
	PosXStartMiddle    float64
	PosYStartMiddle    float64
	PosZStartMiddle    float64
	PosXEndRand        float64
	PosYEndRand        float64
	PosZEndRand        float64
	PosXEndMiddle      float64
	PosYEndMiddle      float64
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
	SizeStartXRandMin  float64
	SizeStartXRandMax  float64
	SizeStartYRandMin  float64
	SizeStartYRandMax  float64
	SizeEndXRandMin    float64
	SizeEndXRandMax    float64
	SizeEndYRandMin    float64
	SizeEndYRandMax    float64
	SizeRandX          float64
	SizeRandY          float64
	SizeRandXMiddle    float64
	SizeRandYMiddle    float64
	SizeDelta          float64
	SizeSmooth         bool
	AngleStart         float64
	AngleEnd           float64
	AngleX             float64
	AngleY             float64
	AngleZ             float64
	AngleRandMin       float64
	AngleRandMax       float64
	CirclePattern      bool
	CircleInnerSize    float64
	CircleOuterRandMin float64
	CircleOuterRandMax float64
	OrbitRadiusX       float64
	OrbitRadiusY       float64
	OrbitRadiusZ       float64
	OrbitRotations     float64
	OrbitPhase         float64
	OrbitPhaseDelta    float64
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

func effectTableSize(value float64) float64 {
	return value * EffectPixelRatio
}

func effectTableColor(red, green, blue float64) color.RGBA {
	return color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255}
}

func strEffectSpec(file, wav string) EffectSpec {
	return strEffectSpecRandom(file, wav, 0, 0)
}

func strEffectSpecRandom(file, wav string, randMin, randMax int) EffectSpec {
	return strEffectSpecRandomAttached(file, wav, randMin, randMax, false, false)
}

func strEffectSpecAttached(file, wav string, head bool) EffectSpec {
	return strEffectSpecRandomAttached(file, wav, 0, 0, true, head)
}

func strEffectSpecAttachedMin(file, minFile, wav string, head bool) EffectSpec {
	spec := strEffectSpecAttached(file, wav, head)
	spec.Components[0].STRMinFile = minFile
	return spec
}

func strEffectSpecAttachedTexturePath(file, texturePath, wav string, head bool) EffectSpec {
	return strEffectSpecTexturePath(file, texturePath, wav, true, head)
}

func strEffectSpecTexturePath(file, texturePath, wav string, attached, head bool) EffectSpec {
	spec := strEffectSpecRandomAttached(file, wav, 0, 0, attached, head)
	spec.Components[0].TexturePath = texturePath
	return spec
}

func strEffectSpecRandomAttached(file, wav string, randMin, randMax int, attached, head bool) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        file,
			STRRandMin:     randMin,
			STRRandMax:     randMax,
			AttachedEntity: attached,
			SpriteHead:     head,
		}},
	}
	if wav != "" {
		spec.SFX = []string{wav}
		spec.SFXRandMin = randMin
		spec.SFXRandMax = randMax
	}
	return spec
}

func sprEffectSpec(file, wav string, attached, head bool) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     file,
			AttachedEntity: attached,
			SpriteHead:     head,
		}},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func sprDirectionEffectSpec(file, wav string) EffectSpec {
	spec := sprEffectSpec(file, wav, true, false)
	spec.Components[0].SpriteDirection = true
	return spec
}

func sprStopAtEndEffectSpec(file, wav string, attached bool) EffectSpec {
	spec := sprEffectSpec(file, wav, attached, false)
	spec.Components[0].SpriteStopAtEnd = true
	return spec
}

func sprRepeatEffectSpec(file, wav string, attached bool) EffectSpec {
	spec := sprEffectSpec(file, wav, attached, false)
	spec.Components[0].Repeat = true
	return spec
}

func sprTexturePathEffectSpec(file, texturePath string, attached, repeat, head, renderBefore bool, yOffset float64) EffectSpec {
	spec := EffectSpec{
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     file,
			TexturePath:    texturePath,
			AttachedEntity: attached,
			Repeat:         repeat,
			SpriteRepeat:   repeat,
			SpriteHead:     head,
			SpriteYOffset:  yOffset,
			RenderBefore:   renderBefore,
		}},
	}
	if repeat {
		spec.Duration = 5 * time.Minute
	}
	return spec
}

func dropEffectSpec(colorName string, renderBefore bool) EffectSpec {
	topFile := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "/dropitem_" + colorName
	topTexturePath := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "/"
	bottomFile := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "_bottom/dropitem_" + colorName + "_bottom"
	bottomTexturePath := "new_dropitem/dropitem_" + colorName + "/dropitem_" + colorName + "_bottom/"
	return EffectSpec{
		SFX: []string{"effect\\drop_" + colorName + ".wav"},
		Components: []EffectComponent{
			{
				Kind:         EffectComponentSTR,
				STRFile:      topFile,
				TexturePath:  topTexturePath,
				RenderBefore: renderBefore,
			},
			{
				Kind:         EffectComponentSTR,
				STRFile:      bottomFile,
				TexturePath:  bottomTexturePath,
				RenderBefore: renderBefore,
			},
		},
	}
}

func funcEffectSpec(name string, duration time.Duration, attached bool) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       name,
			AttachedEntity: attached,
		}},
	}
}

func banjjakiiEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "크리스마스",
			Duration:       time.Second,
			AttachedEntity: true,
		}},
	}
}

func venomDust2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 100 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "particle3",
			SpriteRepeat:   true,
			Repeat:         true,
			Duration:       100 * time.Millisecond,
			AlphaMax:       1,
			AttachedEntity: true,
			SizeStart:      effectTableSize(80),
			SizeEnd:        effectTableSize(80),
			PosZ:           0,
			PosZEnd:        0.5,
		}},
	}
}

func suiExplosionEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\ef_hit2.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "sui_explosion",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func level99EffectSpec() EffectSpec {
	spec := funcEffectSpec("Level99Aura", 5*time.Minute, true)
	spec.Components[0].TextureFile = "effect/ring_blue.tga"
	return spec
}

func level99GroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundAura",
			TextureFile:    "effect/pikapika2.bmp",
			AttachedEntity: true,
			SizeStart:      effectTableSize(115),
			SizeEnd:        effectTableSize(130),
			PosZ:           0.05,
		}},
	}
}

func level99BubbleEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Level99Bubble",
			TextureFile:    "effect/whitelight.tga",
			AttachedEntity: true,
			Color:          color.RGBA{R: 80, G: 80, B: 255, A: 250},
			SizeStart:      effectTableSize(22.2),
			PosZ:           0.05,
		}},
	}
}

func gumgangEffectSpec() EffectSpec {
	components := make([]EffectComponent, 0, 5)
	for i := 1; i <= 5; i++ {
		components = append(components, EffectComponent{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/super" + strconv.Itoa(i) + ".bmp",
			Duration:       2 * time.Second,
			Delay:          time.Duration(i-1) * 400 * time.Millisecond,
			AlphaMax:       1,
			FadeOut:        true,
			AttachedEntity: true,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(100),
			BlendAdditive:  true,
		})
	}
	return EffectSpec{
		Duration:   3600 * time.Millisecond,
		Components: components,
	}
}

func drainEffectSpec(tint color.RGBA, withBodyColor bool) EffectSpec {
	spec := EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/이팩트/particle1",
			SpriteRepeat:     true,
			Duration:         600 * time.Millisecond,
			Duplicate:        5,
			DuplicateDelay:   0,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			AlphaMax:         1,
			Color:            tint,
			SizeStart:        effectTableSize(150),
			SizeEnd:          effectTableSize(180),
			PosZ:             5,
			Arc:              3,
			Retreat:          3,
		}},
	}
	if withBodyColor {
		spec.Components = append(spec.Components,
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainOwnerBodyColor", AttachedEntity: true, Delay: 500 * time.Millisecond, Duration: 400 * time.Millisecond},
			EffectComponent{Kind: EffectComponentFUNC, FuncName: "EnergyDrainTargetBodyColor", AttachedEntity: true, Duration: 400 * time.Millisecond},
		)
	}
	return spec
}

func defenderCylinderEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 3 * time.Second,
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      texture,
			Duration:         3 * time.Second,
			AlphaMax:         0.6,
			Animation:        1,
			BlendMode:        8,
			BottomSize:       1.5,
			TopSize:          1.5,
			Height:           10,
			Fade:             true,
			Rotate:           true,
			AttachedEntity:   true,
			TotalCircleSides: 32,
			CircleSides:      32,
		}},
	}
}

func robrCylinderBlendComponent(textureName string, tint color.RGBA, duration time.Duration, alphaMax float64, animation int, bottomSize, topSize, height float64, attached, rotate, fade bool, blendMode int) EffectComponent {
	component := robrCylinderComponent(textureName, tint, duration, alphaMax, animation, bottomSize, topSize, height, attached, rotate, fade, blendMode == 2)
	component.BlendMode = blendMode
	component.BlendAdditive = blendMode == 2
	return component
}

func absorbSpiritsEffectSpec() EffectSpec {
	tint := color.RGBA{R: 77, G: 77, B: 255, A: 255}
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"effect\\흡기.wav"},
		Components: []EffectComponent{
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 1.1, 15, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1, 1, 13, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", tint, 1500*time.Millisecond, 0.3, 1, 1.1, 3, 2, true, true, true, 2),
			absorbSpiritParticle(tint, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, true),
			absorbSpiritParticle(tint, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			absorbSpiritParticle(tint, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, false),
		},
	}
}

func absorbSpiritParticle(tint color.RGBA, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:             EffectComponent3D,
		TextureFile:      "effect/pok3.tga",
		Color:            tint,
		Duration:         duration,
		Delay:            delay,
		Duplicate:        duplicate,
		DuplicateDelay:   duplicateDelay,
		AlphaMax:         0.8,
		FadeIn:           true,
		FadeOut:          true,
		PosXRand:         posXRand,
		PosYRand:         posYRand,
		PosZStartRand:    posZStartRand,
		PosZStartMiddle:  posZStartMiddle,
		PosZEnd:          posZEnd,
		PosZEndRand:      posZEndRand,
		PosZEndMiddle:    posZEndMiddle,
		SizeStart:        effectTableSize(9),
		SizeEnd:          effectTableSize(9),
		SizeRand:         effectTableSize(2),
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Sparkling:        sparkling,
		RotateWithCamera: false,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func gumgangRingEffectSpec(duration time.Duration, alphaMax, bottomSize, topSize float64, wav string) EffectSpec {
	component := robrCylinderBlendComponent("ring_yellow", color.RGBA{}, duration, alphaMax, 4, bottomSize, topSize, 2, true, true, true, 8)
	component.Duplicate = 4
	component.DuplicateDelay = 100 * time.Millisecond
	spec := EffectSpec{
		Duration:   duration + 300*time.Millisecond,
		Components: []EffectComponent{component},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func teiHitEffectSpec(texture, wav string, duplicate int, delay time.Duration, tint color.RGBA) EffectSpec {
	component := EffectComponent{
		Kind:             EffectComponent3D,
		TextureFile:      texture,
		Color:            tint,
		Duration:         550 * time.Millisecond,
		Delay:            delay,
		Duplicate:        duplicate,
		AlphaMax:         0.8,
		FadeIn:           true,
		FadeOut:          true,
		PosXEndRand:      40,
		PosYEndRand:      40,
		SizeStartX:       effectTableSize(10),
		SizeStartY:       effectTableSize(150),
		SizeEndX:         effectTableSize(10),
		SizeEndY:         effectTableSize(150),
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Overlay:          true,
		RotateToTarget:   true,
		RotateWithCamera: true,
	}
	spec := EffectSpec{
		Duration:   550*time.Millisecond + delay,
		Components: []EffectComponent{component},
	}
	if wav != "" {
		spec.SFX = []string{wav}
	}
	return spec
}

func tanjiEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 150 * time.Millisecond,
		SFX:      []string{"effect\\mon_탄지신통.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/blue_ivy.bmp",
			Duration:         150 * time.Millisecond,
			AlphaMax:         1,
			BlendMode:        2,
			BlendAdditive:    true,
			ToSrc:            true,
			RotateWithCamera: true,
			RotateToTarget:   true,
			AngleStart:       90,
			AngleEnd:         90,
			PosZ:             1,
			SizeStart:        effectTableSize(50),
			SizeEnd:          effectTableSize(50),
			AttachedEntity:   true,
		}},
	}
}

func rogueCoinEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2950 * time.Millisecond,
		SFX:      []string{"effect\\rog_steal coin.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent2D,
			TextureFile:    "effect/coin_a.bmp",
			Duration:       1500 * time.Millisecond,
			Duplicate:      30,
			DuplicateDelay: 50 * time.Millisecond,
			AlphaMax:       0.8,
			FadeOut:        true,
			PosXEndRand:    10,
			PosYEndRand:    10,
			PosZ:           2,
			SizeStart:      effectTableSize(20),
			SizeEnd:        effectTableSize(20),
			BlendMode:      2,
			BlendAdditive:  true,
			Overlay:        true,
			RotateToTarget: true,
			AttachedEntity: true,
		}},
	}
}

func throwItemEffectSpec(texture string, size float64) EffectSpec {
	return throwItemEffectSpecFull(texture, size, 300*time.Millisecond, 0.5)
}

func throwItemEffectSpecFull(texture string, size float64, duration time.Duration, rotations float64) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      texture,
			Duration:         duration,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Rotate:           true,
			AngleStart:       180,
			AngleEnd:         180 + 360*rotations,
			PosZ:             1,
			SizeStart:        effectTableSize(size),
			SizeEnd:          effectTableSize(size),
			AttachedEntity:   true,
		}},
	}
}

func throwItemSoundEffectSpec(texture string, size float64, wav string) EffectSpec {
	spec := throwItemEffectSpecFull(texture, size, 200*time.Millisecond, 1)
	spec.SFX = []string{wav}
	return spec
}

func hapgyeokEffectSpec() EffectSpec {
	return firecrackerBannerEffectSpec("합격_")
}

func firecrackerBannerEffectSpec(sprite string) EffectSpec {
	return EffectSpec{
		SFX: []string{"effect\\itempokjuk.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSPR,
				SpriteFile:     sprite,
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentSTR,
				STRFile:        "itempokjuk",
				AttachedEntity: true,
			},
		},
	}
}

func npcEarthquakeEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\earth_quake.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSPR,
				SpriteFile:     "어스퀘이크",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
				Duplicate:      3,
				DuplicateDelay: 35 * time.Millisecond,
			},
		},
	}
}

func dragonFearEffectSpec() EffectSpec {
	return EffectSpec{
		CameraShake: 650 * time.Millisecond,
		SFX:         []string{"effect\\dragonfear.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentSTR,
				STRFile:        "dragon_h",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func groundTextureEffectSpec(texture string) EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundTexture",
			TextureFile:    texture,
			Duration:       1500 * time.Millisecond,
			SizeStart:      1,
			SizeEnd:        1,
			PosZ:           0.05,
			BlendAdditive:  true,
			AttachedEntity: false,
		}},
	}
}

func evilLandEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:      EffectComponentFUNC,
				FuncName:  "FlatColorTile",
				Color:     color.RGBA{R: 160, G: 160, B: 160, A: 51},
				SizeStart: 1,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/curse.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      1,
				SizeEnd:        1,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				AttachedEntity: false,
			},
		},
	}
}

func gravitationEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    1500 * time.Millisecond,
		CameraShake: 200 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "FlatColorTile",
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 51},
				SizeStart:      1,
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/lens_w.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      0.5,
				SizeEnd:        0.5,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				AttachedEntity: true,
			},
		},
	}
}

func bottomBlueEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		Components: []EffectComponent{
			bottomBlueCylinder(1.5, 2.0, 40.0/255.0, 0, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.58, 2.1, 32.0/255.0, 10, color.RGBA{R: 51, G: 153, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 26.6, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
			bottomBlueCylinder(1.65, 2.0, 15.0/255.0, 79.8, color.RGBA{R: 25, G: 102, B: 255, A: 255}),
		},
	}
}

func bottomBlueCylinder(size, height, alpha, angleY float64, tint color.RGBA) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_down",
		Duration:         20 * time.Second,
		AlphaMax:         alpha,
		BottomSize:       size,
		TopSize:          size,
		Height:           height,
		AngleY:           angleY,
		RotateWithCamera: true,
		TotalCircleSides: 4,
		CircleSides:      4,
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		Color:            tint,
	}
}

func judexEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1 * time.Second,
		SFX:      []string{"effect\\ab_judex.wav"},
		Components: []EffectComponent{
			judexCylinder(0.4, 0.5, 3.5),
			judexCylinder(0.45, 0.75, 2.5),
			judexCylinder(0.5, 1, 1.5),
		},
	}
}

func judexCylinder(bottom, top, height float64) EffectComponent {
	return EffectComponent{
		Kind:        EffectComponentCylinder,
		TextureName: "ring_white",
		Duration:    1 * time.Second,
		BottomSize:  bottom,
		TopSize:     top,
		Height:      height,
		Rotate:      true,
	}
}

func glassWall3EffectSpec() EffectSpec {
	tint := color.RGBA{R: 153, G: 255, B: 153, A: 255}
	portalWave := EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "magic_green",
		Color:            tint,
		Duration:         500 * time.Millisecond,
		AlphaMax:         0.4,
		Animation:        4,
		BottomSize:       2.4,
		TopSize:          3.9,
		Height:           0.1,
		PosZ:             0.1,
		FadeOut:          true,
		Rotate:           true,
		Duplicate:        150,
		DuplicateDelay:   200 * time.Millisecond,
		BlendMode:        2,
		BlendAdditive:    true,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
	}
	cap := robrCylinderBlendComponent("alpha1", tint, 30*time.Second, 0.5, 0, 1, 1, 1, true, true, true, 2)
	cap.PosZ = 2
	cap.TotalCircleSides = 20
	cap.CircleSides = 10
	return EffectSpec{
		Components: []EffectComponent{
			portalWave,
			robrCylinderBlendComponent("magic_green", tint, 30*time.Second, 0.4, 0, 0.6, 0.6, 7, true, true, true, 2),
			robrCylinderBlendComponent("magic_green", tint, 30*time.Second, 0.4, 0, 0.8, 0.8, 6, true, true, true, 2),
			cap,
		},
	}
}

func rollingCutterCounterEffectSpec(index int) EffectSpec {
	texture := "effect/회전카운터" + strconv.Itoa(index) + ".tga"
	return EffectSpec{
		Duration: time.Second,
		Components: []EffectComponent{
			rollingCutterCounterComponent(texture, 1.0, 1.0, 1.0, 1.0, 200, 1),
			rollingCutterCounterComponent(texture, 0.2, 0.7, 0.7, 1.0, 220, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.5, 0.5, 1.0, 240, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.3, 0.3, 1.0, 260, 2),
			rollingCutterCounterComponent(texture, 0.2, 0.1, 0.1, 1.0, 280, 2),
		},
	}
}

func rollingCutterCounterComponent(texture string, alphaMax, red, green, blue, sizeStart float64, blendMode int) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    texture,
		Color:          color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255},
		Duration:       time.Second,
		AlphaMax:       alphaMax,
		FadeIn:         true,
		FadeOut:        true,
		PosZ:           4,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(20),
		SizeSmooth:     true,
		BlendMode:      blendMode,
		BlendAdditive:  blendMode == 2,
		AttachedEntity: true,
	}
}

func earthWallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    1 * time.Second,
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\wizard_earthspike.wav"},
		Components: []EffectComponent{
			quadHornEffectComponent(
				"effect/stone.bmp",
				time.Second,
				[2]float64{0.75, 1.2},
				[2]float64{0.2, 0.2},
				[2]float64{0.2, 0.2},
				-0.1,
				[2]float64{0.4, 0.9},
				[2]float64{1, 360},
				[2]float64{-8, 8},
				3,
				250*time.Millisecond,
				true,
				1,
			),
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func screenQuakeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    200 * time.Millisecond,
		CameraShake: 200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Duration:       200 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func chemical2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         500 * time.Millisecond,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 132 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			AttachedEntity: true,
		}},
	}
}

func acidDemonEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         500 * time.Millisecond,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Delay:          200 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func heal2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1890 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 15),
			heal2CylinderComponent(1, 1, 13),
			heal2CylinderComponent(1.1, 3, 2),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 4, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 2, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 20, 1.5, 1.5, 0, 0, 0, 3, 6, 9, 2, 2, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 10, 1, 1, 1, 0, 6, 0, 0, 9, 2, 0, false),
		},
	}
}

func heal4EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2 * time.Second,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			heal2CylinderComponent(1.1, 1.1, 18),
			heal2CylinderComponent(1, 1, 15),
			heal2CylinderComponent(1.1, 3, 3),
			healSparkParticle(0.8, 1500*time.Millisecond, 0, 10*time.Millisecond, 7, 1.2, 1.2, 0, 0, 0, 1, 8, 9, 2, 3, true),
			healSparkParticle(0.8, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 25, 1.5, 1.5, 0, 0, 0, 3, 6, 10, 5, 3, true),
			healSparkParticle(0.8, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 15, 1, 1, 1, 0, 6, 0, 0, 11, 2, 0, false),
		},
	}
}

func heal2CylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return robrCylinderBlendComponent("ring_white", color.RGBA{R: 178, G: 255, B: 178, A: 255}, 1500*time.Millisecond, 0.3, 1, bottomSize, topSize, height, true, true, true, 2)
}

func healSparkParticle(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle, size, sizeRand float64, sparkNumber int, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		TextureFile:     "effect/pok3.tga",
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		Duration:        duration,
		Delay:           delay,
		Duplicate:       duplicate,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       effectTableSize(size),
		SizeEnd:         effectTableSize(size),
		SizeRand:        effectTableSize(sizeRand),
		BlendMode:       2,
		BlendAdditive:   true,
		AttachedEntity:  true,
		Sparkling:       sparkling,
		SparkNumber:     sparkNumber,
	}
	return component
}

func exit2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		SFX:      []string{"effect\\ef_teleportation.wav"},
		Components: []EffectComponent{
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.3, 0.3, 35, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.4, 0.6, 23, true, true, true, 2),
			robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 1500*time.Millisecond, 0.3, 1, 0.5, 0.7, 5, true, true, true, 2),
		},
	}
}

func bottomSquareEffectSpec(texture string, tint color.RGBA, alpha, bottomSize, topSize, height float64, fade bool) EffectSpec {
	return EffectSpec{
		Duration: 50 * time.Second,
		Components: []EffectComponent{
			bottomSquareCylinder(texture, tint, alpha, bottomSize, topSize, height, 50*time.Second, false, fade),
			bottomSquareCylinder(texture, tint, 0.1, bottomSize, topSize, height, 2*time.Second, true, true),
		},
	}
}

func bottomSquareCylinder(texture string, tint color.RGBA, alpha, bottomSize, topSize, height float64, duration time.Duration, repeat, fade bool) EffectComponent {
	component := robrCylinderBlendComponent(texture, tint, duration, alpha, 0, bottomSize, topSize, height, true, false, fade, 2)
	component.TotalCircleSides = 4
	component.CircleSides = 4
	component.AngleY = 45
	component.Repeat = repeat
	if repeat {
		component.Animation = 1
	}
	return component
}

func warpZone2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 7 * time.Second,
		Components: []EffectComponent{
			warpZone2Cylinder(1.15, 1.9),
			warpZone2Cylinder(1.05, 1.8),
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pok1.tga",
				Color:          color.RGBA{R: 230, G: 255, B: 230, A: 255},
				Duration:       time.Second,
				Duplicate:      5,
				DuplicateDelay: 300 * time.Millisecond,
				AlphaMax:       1,
				FadeIn:         true,
				FadeOut:        true,
				PosXStartRand:  1.45,
				PosYStartRand:  1.45,
				PosZEndRand:    1.5,
				PosZEndMiddle:  1.4,
				SizeStart:      effectTableSize(32),
				SizeEnd:        effectTableSize(32),
				BlendMode:      2,
				BlendAdditive:  true,
				AttachedEntity: true,
				Repeat:         true,
			},
		},
	}
}

func warpZone2Cylinder(bottomSize, topSize float64) EffectComponent {
	component := robrCylinderBlendComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 4*time.Second, 0.48, 3, bottomSize, topSize, 1.35, true, false, true, 2)
	component.Duplicate = 4
	component.DuplicateDelay = time.Second
	component.Repeat = true
	return component
}

func beginAsuraEffectSpec() EffectSpec {
	components := []EffectComponent{
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 4.5, -4, true, false, true, 2),
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 2.5, -4, true, false, true, 2),
	}
	positions := []float64{-6, -3.6, -1.2, 1.2, 3.6, 6}
	for i, posX := range positions {
		components = append(components,
			asuraGlyphComponent(i+1, posX, time.Duration(i)*100*time.Millisecond, 1200*time.Millisecond, 250, 120, true, false),
			asuraGlyphComponent(i+1, posX, time.Duration(1200+i*100)*time.Millisecond, 400*time.Millisecond, 120, 200, false, true),
		)
	}
	return EffectSpec{
		Duration:   2100 * time.Millisecond,
		Components: components,
	}
}

func beginAsura11EffectSpec() EffectSpec {
	components := []EffectComponent{
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 4.5, -4, true, false, true, 2),
		robrCylinderBlendComponent("ring_white", color.RGBA{}, 800*time.Millisecond, 0, 2, 1, 2.5, -4, true, false, true, 2),
	}
	positions := []float64{-8, -4.8, -1.6, 1.6, 4.8, 8}
	for i, posX := range positions {
		components = append(components,
			asuraGlyphComponent(i+11, posX, time.Duration(i)*100*time.Millisecond, 1200*time.Millisecond, 300, 150, true, false),
			asuraGlyphComponent(i+11, posX, time.Duration(1200+i*100)*time.Millisecond, 400*time.Millisecond, 150, 250, false, true),
		)
	}
	return EffectSpec{
		Duration:   2100 * time.Millisecond,
		Components: components,
	}
}

func asuraGlyphComponent(index int, posX float64, delay, duration time.Duration, sizeStart, sizeEnd float64, duplicate, fadeOut bool) EffectComponent {
	component := EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    "effect/asura" + strconv.Itoa(index) + ".tga",
		Color:          color.RGBA{R: 26, G: 26, B: 26, A: 255},
		Duration:       duration,
		Delay:          delay,
		AlphaMax:       1,
		AttachedEntity: true,
		FadeIn:         !fadeOut,
		FadeOut:        fadeOut,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(sizeEnd),
		SizeSmooth:     true,
		PosX:           posX,
		PosZ:           4,
		Overlay:        true,
	}
	if duplicate {
		component.Duplicate = 3
		component.DuplicateDelay = 150 * time.Millisecond
		component.AlphaMaxDelta = -0.25
	}
	return component
}

func tripleAttackEffectSpec() EffectSpec {
	return delayedSoundEffectSpec(
		[]string{"effect\\ef_hit2.wav", "effect\\ef_hit4.wav", "effect\\ef_hit2.wav"},
		[]time.Duration{0, 200 * time.Millisecond, 400 * time.Millisecond},
	)
}

func naturalRecoveryEffectSpec(tint color.RGBA, wav string) EffectSpec {
	return EffectSpec{
		Duration: 1110 * time.Millisecond,
		SFX:      []string{wav},
		Components: []EffectComponent{{
			Kind:            EffectComponent3D,
			TextureFile:     "effect/pok1.tga",
			Color:           tint,
			Duration:        500 * time.Millisecond,
			Delay:           500 * time.Millisecond,
			Duplicate:       12,
			DuplicateDelay:  10 * time.Millisecond,
			AlphaMax:        0.8,
			FadeIn:          true,
			FadeOut:         true,
			SizeStart:       effectTableSize(30),
			SizeEnd:         effectTableSize(30),
			SizeRand:        effectTableSize(20),
			BlendMode:       2,
			BlendAdditive:   true,
			PosXRand:        0.6,
			PosYRand:        0.6,
			PosZStartRand:   1.5,
			PosZStartMiddle: 2,
			PosZEndRand:     1,
			PosZEndMiddle:   5,
			Sparkling:       true,
			SparkNumber:     3,
			AttachedEntity:  true,
		}},
	}
}

func guardEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 600 * time.Millisecond,
		SFX:      []string{"effect\\kyrie_guard.wav"},
		Components: []EffectComponent{
			guardCylinderComponent(1.5, 1, 0.7, 2.14),
			guardCylinderComponent(1.5, 1.5, 1.14, 1),
			guardCylinderComponent(1, 1.5, 0.7, 0.3),
		},
	}
}

func guardCylinderComponent(bottomSize, topSize, height, posZ float64) EffectComponent {
	component := robrCylinderBlendComponent("guardk", color.RGBA{R: 232, G: 255, B: 230, A: 255}, 600*time.Millisecond, 0.6, 0, bottomSize, topSize, height, true, false, true, 2)
	component.TotalCircleSides = 8
	component.CircleSides = 5
	component.AngleY = 112.5
	component.PosZ = posZ
	return component
}

func magnum2EffectSpec() EffectSpec {
	return delayedSoundEffectSpec(
		[]string{"permeter_attack.wav", "effect\\ef_magnumbreak.wav"},
		[]time.Duration{0, 300 * time.Millisecond},
	)
}

func entry2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		SFX:      []string{"effect\\ef_portal.wav"},
		Components: []EffectComponent{
			teleportCylinderComponent(0.3, 0.3, 35),
			teleportCylinderComponent(0.6, 0.8, 25),
			teleportCylinderComponent(0.8, 1.0, 13),
			teleportCylinderComponent(1.0, 1.3, 5),
		},
	}
}

func soulBreakerEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\기공포.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/purpleslash.tga",
			Duration:         500 * time.Millisecond,
			AlphaMax:         0.4,
			FadeIn:           true,
			FadeOut:          true,
			ToSrc:            true,
			RotateWithCamera: true,
			RotateToTarget:   true,
			AngleStart:       90,
			PosZ:             2,
			SizeStart:        effectTableSize(100),
			SizeEnd:          effectTableSize(200),
			AttachedEntity:   true,
		}},
	}
}

func portal5EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 800 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "EffectBodyColor",
			Duration:       800 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func magicCrasher2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\swordman_provoke.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "EffectBodyColor",
			Duration:       1000 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func spiderWebEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Second,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "SpiderWeb",
			TextureFile:    "effect/spiderweb.tga",
			Duration:       5 * time.Second,
			AlphaMax:       0.7,
			SizeStart:      1.5,
			SizeEnd:        1.5,
			PosZ:           0.05,
			RenderBefore:   true,
			AttachedEntity: false,
		}},
	}
}

func soulBreaker2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\메테오 어썰트.wav"},
		Components: []EffectComponent{
			soulBreaker2SlashComponent(-1, 0, -5, 0, 0),
			soulBreaker2SlashComponent(-0.7, -0.7, -3.53, -3.53, -45),
			soulBreaker2SlashComponent(0, -1, 0, -5, -90),
			soulBreaker2SlashComponent(0.7, -0.7, 3.53, -3.53, -135),
			soulBreaker2SlashComponent(1, 0, 5, 0, -180),
			soulBreaker2SlashComponent(0.7, 0.7, 3.53, 3.53, -225),
			soulBreaker2SlashComponent(0, 1, 0, 5, -270),
			soulBreaker2SlashComponent(-0.7, 0.7, -3.53, 3.53, -315),
		},
	}
}

func soulBreaker2SlashComponent(posX, posY, posXEnd, posYEnd, angle float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponent3D,
		TextureFile:      "effect/purpleslash.tga",
		Duration:         500 * time.Millisecond,
		AlphaMax:         0.6,
		FadeOut:          true,
		RotateWithCamera: true,
		AngleStart:       angle,
		PosX:             posX,
		PosY:             posY,
		PosXEnd:          posXEnd,
		PosYEnd:          posYEnd,
		SizeStart:        effectTableSize(100),
		SizeEnd:          effectTableSize(200),
	}
}

func bodySizeEffectSpec(name string, duration time.Duration, targetSize float64) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       name,
			Duration:       duration,
			SizeEnd:        targetSize,
			AttachedEntity: true,
		}},
	}
}

func pressureEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         1001 * time.Millisecond,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 500 * time.Millisecond,
		SFX:              []string{"effect\\프레셔.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/cross_old.bmp",
				Duration:       500 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				Rotate:         true,
				AngleStart:     0,
				AngleEnd:       -611,
				PosZ:           20,
				PosZEnd:        5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/cross_old.bmp",
				Duration:       500 * time.Millisecond,
				Delay:          501 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				FadeOut:        true,
				AngleStart:     -611,
				PosZ:           5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				AttachedEntity: true,
			},
		},
	}
}

func pressureDropEffectSpec(textureFile, delayedSFX string) EffectSpec {
	spec := EffectSpec{
		Duration: 1001 * time.Millisecond,
		SFX:      []string{"effect\\프레셔.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    textureFile,
				Duration:       500 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				Rotate:         true,
				AngleStart:     0,
				AngleEnd:       -611,
				PosZ:           20,
				PosZEnd:        5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponent3D,
				TextureFile:    textureFile,
				Duration:       500 * time.Millisecond,
				Delay:          501 * time.Millisecond,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				FadeOut:        true,
				AngleStart:     -611,
				AngleEnd:       -611,
				PosZ:           5,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				AttachedEntity: true,
			},
		},
	}
	if delayedSFX != "" {
		spec.SFX = append(spec.SFX, delayedSFX)
		spec.SFXDelays = []time.Duration{0, 500 * time.Millisecond}
	}
	return spec
}

func bash3DEffectSpec(funcName, wav string, duration, delay time.Duration, duplicate int) EffectSpec {
	return EffectSpec{
		Duration: duration,
		SFX:      []string{wav},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       funcName,
				AttachedEntity: true,
			},
			bash3DCylinderComponent(4.5, delay, duplicate),
			bash3DCylinderComponent(7.2, delay, duplicate),
		},
	}
}

func bash3DCylinderComponent(topSize float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "alpha_center",
		Duration:         175 * time.Millisecond,
		Delay:            delay,
		Duplicate:        duplicate,
		AlphaMax:         0.6,
		Fade:             true,
		AngleX:           -90,
		AngleZRandom:     360,
		FixedPerspective: true,
		PosZ:             1.5,
		Height:           0,
		BottomSize:       0.01,
		TopSize:          topSize,
		Animation:        2,
		AttachedEntity:   true,
		TotalCircleSides: 30,
		CircleSides:      1,
	}
}

func basilicaEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		Components: []EffectComponent{
			basilicaCylinderComponent(2.45, 2.0, 32.0/255.0, 0),
			basilicaCylinderComponent(2.52, 2.1, 32.0/255.0, 10),
			basilicaCylinderComponent(2.6, 2.0, 15.0/255.0, 26.6),
			basilicaCylinderComponent(2.6, 2.0, 15.0/255.0, 79.8),
		},
	}
}

func bottomBasilica2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		SFX:      []string{"effect\\wl_whiteimprison.wav"},
		Components: []EffectComponent{
			basilicaCylinderComponent(2.2, 3.0, 65.0/255.0, 0),
			basilicaCylinderComponent(2.25, 3.1, 65.0/255.0, 10),
			basilicaCylinderComponent(2.3, 3.0, 15.0/255.0, 0),
			basilicaCylinderComponent(2.3, 3.0, 15.0/255.0, 53.2),
		},
	}
}

func basilicaCylinderComponent(size, height, alpha, angleY float64) EffectComponent {
	component := robrCylinderBlendComponent("alpha_down", color.RGBA{}, 20*time.Second, alpha, 0, size, size, height, true, false, false, 2)
	component.TotalCircleSides = 4
	component.CircleSides = 4
	component.RotateWithCamera = true
	component.AngleY = angleY
	return component
}

func melodyBillboardEffectSpec(texture, wav string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 100 * time.Millisecond,
		SFX:      []string{wav},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    texture,
			Color:          tint,
			Duration:       100 * time.Millisecond,
			AlphaMax:       0.6,
			AttachedEntity: true,
			SizeStart:      effectTableSize(50),
			SizeEnd:        effectTableSize(50),
			PosZ:           0.5,
			Repeat:         true,
		}},
	}
}

func secra2EffectSpec() EffectSpec {
	components := make([]EffectComponent, 0, 10)
	for i := 0; i < 10; i++ {
		sizeStart := 850 - float64(i)*50
		tint := effectTableColor(1, 0.55+float64(i)*0.05, 0.55+float64(i)*0.05)
		components = append(components, EffectComponent{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/priest_spell.bmp",
			Color:          tint,
			Duration:       1500 * time.Millisecond,
			AlphaMax:       0.3,
			BlendMode:      2,
			BlendAdditive:  true,
			FadeIn:         true,
			FadeOut:        true,
			AttachedEntity: true,
			PosZ:           7,
			SizeStart:      effectTableSize(sizeStart),
			SizeEnd:        effectTableSize(100),
			SizeSmooth:     true,
		})
	}
	return EffectSpec{
		Duration:   1500 * time.Millisecond,
		SFX:        []string{"effect\\ab_ancilla.wav"},
		Components: components,
	}
}

func glassWall4EffectSpec() EffectSpec {
	tint := effectTableColor(0.0001, 1, 0.0001)
	return EffectSpec{
		Duration: 30 * time.Second,
		SFX:      []string{"effect\\ef_readyportal.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/ef_epitree.tga",
				Color:          tint,
				Duration:       30 * time.Second,
				AlphaMax:       0.6,
				BlendMode:      2,
				BlendAdditive:  true,
				AttachedEntity: true,
				PosZ:           7,
				SizeStart:      effectTableSize(400),
				SizeEnd:        effectTableSize(400),
			},
			epiclesisTreePulseComponent(380, 420, 0),
			epiclesisTreePulseComponent(420, 380, time.Second),
		},
	}
}

func epiclesisTreePulseComponent(sizeStart, sizeEnd float64, delay time.Duration) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    "effect/ef_epitree.tga",
		Color:          effectTableColor(0.0001, 1, 0.0001),
		Duration:       990 * time.Millisecond,
		Delay:          delay,
		Duplicate:      15,
		DuplicateDelay: 2 * time.Second,
		AlphaMax:       0.6,
		BlendMode:      2,
		BlendAdditive:  true,
		AttachedEntity: true,
		PosZ:           7,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(sizeEnd),
	}
}

func bash3D6EffectSpec() EffectSpec {
	spec := bash3DEffectSpec("Bash3D6", "effect\\bash3d.wav", 500*time.Millisecond, 200*time.Millisecond, 5)
	tint := effectTableColor(0.3, 0.5, 1)
	for i := 1; i < len(spec.Components); i++ {
		spec.Components[i].Color = tint
	}
	return spec
}

func teiHit1TEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 350 * time.Millisecond,
		SFX:      []string{"effect\\mon_아수라 패황권.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/lens1.tga",
			Color:            effectTableColor(0.1, 0.1, 1),
			Duration:         250 * time.Millisecond,
			Delay:            100 * time.Millisecond,
			Duplicate:        24,
			DuplicateDelay:   0,
			AlphaMax:         0.8,
			BlendMode:        2,
			BlendAdditive:    true,
			FadeIn:           true,
			FadeOut:          true,
			AttachedEntity:   true,
			PosXEndRand:      40,
			PosYEndRand:      40,
			SizeStartX:       effectTableSize(10),
			SizeStartY:       effectTableSize(150),
			SizeEndX:         effectTableSize(10),
			SizeEndY:         effectTableSize(150),
			Overlay:          true,
			RotateToTarget:   true,
			RotateWithCamera: true,
		}},
	}
}

func energyDrainProjectileEffectSpec(tint color.RGBA, sizeStart, sizeEnd float64) EffectSpec {
	return EffectSpec{
		Duration: 600 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "data/sprite/이팩트/particle1",
			SpriteRepeat:   true,
			Duration:       600 * time.Millisecond,
			Duplicate:      5,
			FromSrc:        true,
			ToSrc:          true,
			RotateToTarget: true,
			Color:          tint,
			SizeStart:      effectTableSize(sizeStart),
			SizeEnd:        effectTableSize(sizeEnd),
			PosZ:           5,
			Arc:            3,
			Retreat:        3,
		}},
	}
}

func transBlueBodyEffectSpec() EffectSpec {
	return funcEffectSpec("TransBlueBody", 900*time.Millisecond, true)
}

func magicCrasherEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         time.Second,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 300 * time.Millisecond,
		SFX:              []string{"effect\\매직 크래쉬.wav"},
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "MagicCrasherBodyColor",
				AttachedEntity: true,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "CameraQuake",
				Delay:          300 * time.Millisecond,
				AttachedEntity: true,
			},
		},
	}
}

func falconAssaultEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:         500 * time.Millisecond,
		CameraShake:      200 * time.Millisecond,
		CameraShakeDelay: 300 * time.Millisecond,
		SFX:              []string{"effect\\hunter_blitzbeat.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Delay:          300 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func moonlitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 20 * time.Second,
		SFX:      []string{"effect\\달빛세레나데.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "FlatColorTile",
			Color:          color.RGBA{R: 255, G: 138, B: 187, A: 153},
			SizeStart:      1,
			AttachedEntity: false,
		}},
	}
}

func darkSoulStrikeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 450 * time.Millisecond,
		SFX:      []string{"effect\\ef_soulstrike.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				TextureFile:     "effect/pok3.tga",
				Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
				Duration:        200 * time.Millisecond,
				Delay:           250 * time.Millisecond,
				DuplicateDelay:  150 * time.Millisecond,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				ToSrc:           true,
				PosZEnd:         1,
				PosZSmooth:      true,
				PosZStartRand:   5,
				PosZStartMiddle: 6,
				SizeStart:       effectTableSize(50),
				SizeEnd:         effectTableSize(50),
				AttachedEntity:  true,
			},
			{
				Kind:           EffectComponent3D,
				SpriteFile:     "data/sprite/이팩트/particle5",
				SpriteRepeat:   true,
				Duration:       250 * time.Millisecond,
				Duplicate:      5,
				DuplicateDelay: 20 * time.Millisecond,
				ToSrc:          true,
				RotateToTarget: true,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(500),
				PosZ:           3,
				Arc:            4,
				Retreat:        4,
			},
		},
	}
}

func darkJupitelHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pokjuk_d.bmp",
				Duration:       100 * time.Millisecond,
				SizeStart:      0,
				SizeEnd:        effectTableSize(25),
				BlendMode:      2,
				BlendAdditive:  true,
				RotateToTarget: true,
				FadeOut:        true,
				Overlay:        true,
				AttachedEntity: true,
			},
			{
				Kind: EffectComponent3D,
				TextureFiles: []string{
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_b.bmp",
					"effect/twirl_soft.bmp",
					"effect/thunder_ball_c.bmp",
					"effect/twirl_soft.bmp",
				},
				FrameDelay:     10 * time.Millisecond,
				Duration:       300 * time.Millisecond,
				SizeStart:      effectTableSize(75),
				SizeEnd:        effectTableSize(75),
				BlendMode:      2,
				BlendAdditive:  true,
				Overlay:        true,
				AttachedEntity: true,
			},
		},
	}
}

func darkCastingEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "ring_black",
			Color:            color.RGBA{R: 255, G: 255, B: 255, A: 255},
			AlphaMax:         0.8,
			Animation:        2,
			AttachedEntity:   true,
			BlendMode:        2,
			BlendAdditive:    true,
			BottomSize:       1,
			TopSize:          5,
			Height:           4,
			Fade:             true,
			Rotate:           true,
			TotalCircleSides: 32,
			CircleSides:      32,
		}},
	}
}

func mildWindEffectSpec(texture string) EffectSpec {
	components := []EffectComponent{
		mildWindComponent(texture, 1, 1, 1, 1, 300, 100),
		mildWindComponent(texture, 0.2, 0.7, 0.7, 1, 220, 20),
		mildWindComponent(texture, 0.2, 0.5, 0.5, 1, 350, 100),
		mildWindComponent(texture, 0.2, 0.3, 0.3, 1, 400, 100),
		mildWindComponent(texture, 0.2, 0.1, 0.1, 1, 450, 100),
	}
	return EffectSpec{
		Duration:   time.Second,
		SFX:        []string{"effect\\t_바람방출.wav"},
		Components: components,
	}
}

func mildWindComponent(texture string, alphaMax, red, green, blue, sizeStart, sizeEnd float64) EffectComponent {
	return EffectComponent{
		Kind:           EffectComponent3D,
		TextureFile:    texture,
		Color:          color.RGBA{R: uint8(red * 255), G: uint8(green * 255), B: uint8(blue * 255), A: 255},
		Duration:       time.Second,
		AlphaMax:       alphaMax,
		BlendMode:      2,
		BlendAdditive:  true,
		FadeIn:         true,
		FadeOut:        true,
		PosZ:           4,
		SizeStart:      effectTableSize(sizeStart),
		SizeEnd:        effectTableSize(sizeEnd),
		SizeSmooth:     true,
		AttachedEntity: true,
	}
}

func grandCrossEffectSpec() EffectSpec {
	components := make([]EffectComponent, 0, 25)
	addSquare := func(x, y float64) {
		components = append(components, grandCrossCylinder(4, 4, 0.7, 45, x, y))
	}
	addArc := func(angleY, x, y float64) {
		components = append(components, grandCrossCylinder(20, 5, 3, angleY, x, y))
	}
	addSquare(0, 0)
	for _, x := range []float64{-1, -2, -3, -4, 1, 2, 3, 4} {
		addSquare(x, 0)
	}
	for _, y := range []float64{1, 2, 3, 4, -1, -2, -3, -4} {
		addSquare(0, y)
	}
	addSquare(1, 1)
	addSquare(1, -1)
	addSquare(-1, -1)
	addSquare(-1, 1)
	addArc(-180, 3.5, 3.5)
	addArc(90, 3.5, -3.5)
	addArc(0, -3.5, -3.5)
	addArc(-90, -3.5, 3.5)
	return EffectSpec{
		Duration:   2 * time.Second,
		SFX:        []string{"effect\\cru_grand cross.wav"},
		Components: components,
	}
}

func grandCrossCylinder(totalSides, sides int, size, angleY, posX, posY float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "ring_red",
		Duration:         time.Second,
		Duplicate:        3,
		DuplicateDelay:   500 * time.Millisecond,
		AlphaMax:         0.1,
		Animation:        1,
		BlendMode:        2,
		BlendAdditive:    true,
		BottomSize:       size,
		TopSize:          size,
		Height:           5,
		Fade:             true,
		AngleY:           angleY,
		PosX:             posX,
		PosY:             posY,
		TotalCircleSides: totalSides,
		CircleSides:      sides,
	}
}

func propertyGroundEffectSpec(name, texture string) EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponentFUNC,
			FuncName:         name,
			TextureName:      texture,
			Duration:         1500 * time.Millisecond,
			Repeat:           true,
			BottomSize:       1,
			TopSize:          3,
			Height:           2,
			TotalCircleSides: 20,
			CircleSides:      20,
		}},
	}
}

func landProtectorGroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponentFUNC,
			FuncName:    "LandProtectorGround",
			TextureFile: "effect/aaa copy.bmp",
			Duration:    1500 * time.Millisecond,
			Repeat:      true,
			SizeStart:   0.8,
			SizeEnd:     0.85,
			PosZ:        0.05,
		}},
	}
}

func soundOnlyEffectSpec(paths ...string) EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      paths,
	}
}

func whitePulseEffectSpec() EffectSpec {
	return EffectSpec{Duration: 500 * time.Millisecond}
}

func spearProjectileEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 140 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "창",
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Duration:         140 * time.Millisecond,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			PosZ:             1,
			SizeStart:        effectTableSize(100),
			SizeEnd:          effectTableSize(100),
			AngleStart:       180,
			AngleEnd:         180,
			AttachedEntity:   true,
		}},
	}
}

func shieldProjectileEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 140 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/shield_boomerang.bmp",
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Rotate:           true,
			Duration:         140 * time.Millisecond,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			PosZ:             1,
			SizeStart:        effectTableSize(50),
			SizeEnd:          effectTableSize(50),
			AngleStart:       180,
			AngleEnd:         540,
			AttachedEntity:   true,
		}},
	}
}

func gospelGroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "FlatColorTile",
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 13},
				SizeStart:      1,
				RenderBefore:   true,
				AttachedEntity: false,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/cross_old.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      0.5,
				SizeEnd:        0.5,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				RenderBefore:   true,
				AttachedEntity: false,
			},
		},
	}
}

func fogWallGroundEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1500 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "FlatColorTile",
				Color:          color.RGBA{R: 0xaa, G: 0xaa, B: 0xaa, A: 153},
				SizeStart:      1,
				RenderBefore:   true,
				AttachedEntity: false,
			},
			{
				Kind:           EffectComponentFUNC,
				FuncName:       "GroundTexture",
				TextureFile:    "effect/lens_w.bmp",
				Duration:       1500 * time.Millisecond,
				SizeStart:      0.5,
				SizeEnd:        0.5,
				AlphaMax:       0.7,
				PosZ:           0.4,
				BlendAdditive:  true,
				RenderBefore:   true,
				AttachedEntity: false,
			},
		},
	}
}

func songGroundEffectSpec(texture string, tint color.RGBA, textureSize float64) EffectSpec {
	if textureSize <= 0 {
		textureSize = 0.5
	}
	// roBrowser's HoveringTexture applies this value to a [-1,1] quad.
	textureWorldSize := textureSize * 2
	spec := EffectSpec{
		Duration: 1500 * time.Millisecond,
	}
	if tint.A > 0 {
		spec.Components = append(spec.Components, EffectComponent{
			Kind:           EffectComponentFUNC,
			FuncName:       "FlatColorTile",
			Color:          tint,
			SizeStart:      1,
			RenderBefore:   true,
			AttachedEntity: false,
		})
	}
	if texture != "" {
		spec.Components = append(spec.Components, EffectComponent{
			Kind:           EffectComponentFUNC,
			FuncName:       "GroundTexture",
			TextureFile:    texture,
			Duration:       1500 * time.Millisecond,
			SizeStart:      textureWorldSize,
			SizeEnd:        textureWorldSize,
			AlphaMax:       0.7,
			AngleStart:     math.Pi,
			PosZ:           0.2,
			PosZEnd:        0.6,
			BlendAdditive:  true,
			RenderBefore:   true,
			AttachedEntity: false,
		})
	}
	return spec
}

func spiralBeforeCastEffectSpec() EffectSpec {
	return funcEffectSpec("EffectBodyColor", 500*time.Millisecond, true)
}

func sharpShootingCastEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 10 * time.Second,
		Components: []EffectComponent{{
			Kind:             EffectComponentFUNC,
			FuncName:         "CastRing",
			TextureName:      "ring_jadu",
			AlphaMax:         0.9,
			Fade:             true,
			Rotate:           true,
			BottomSize:       0.8,
			TopSize:          2.45,
			Height:           2.8,
			PosZ:             0.08,
			TotalCircleSides: 20,
			CircleSides:      20,
			AttachedEntity:   true,
		}},
	}
}

func quakeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    650 * time.Millisecond,
		CameraShake: 650 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "CameraQuake",
			Duration:       650 * time.Millisecond,
			AttachedEntity: true,
		}},
	}
}

func spiritSphereEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "SpiritSphere",
			TextureFile:    "effect/thunder_center.bmp",
			AttachedEntity: true,
			Duplicate:      5,
		}},
	}
}

func tarotCardEffectSpec(index int) EffectSpec {
	return EffectSpec{
		Duration: 3 * time.Second,
		SFX:      []string{"effect\\priest_slowpoison.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/tarot" + twoDigitString(index) + ".tga",
			Duration:       3 * time.Second,
			AlphaMax:       1,
			AttachedEntity: true,
			FadeIn:         true,
			FadeOut:        true,
			PosZ:           4,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(70),
			SizeSmooth:     true,
		}},
	}
}

func twoDigitString(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}

func delayedSoundEffectSpec(paths []string, delays []time.Duration) EffectSpec {
	return EffectSpec{
		Duration:  500 * time.Millisecond,
		SFX:       paths,
		SFXDelays: delays,
	}
}

func delayedSoundEffectSpecDuration(duration time.Duration, paths []string, delays []time.Duration) EffectSpec {
	spec := delayedSoundEffectSpec(paths, delays)
	spec.Duration = duration
	return spec
}

func repeatedSoundEffectSpec(path string, count int, interval time.Duration) EffectSpec {
	paths := make([]string, 0, count)
	delays := make([]time.Duration, 0, count)
	for i := 0; i < count; i++ {
		paths = append(paths, path)
		delays = append(delays, time.Duration(i)*interval)
	}
	return delayedSoundEffectSpec(paths, delays)
}

func potionEffectSpec(file string, c color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 850 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          c,
			STRFile:        file,
			AttachedEntity: true,
		}},
	}
}

func healingPotionEffectSpec(file string, c color.RGBA) EffectSpec {
	spec := potionEffectSpec(file, c)
	spec.SFX = []string{"_heal_effect.wav"}
	return spec
}

func berserkPotionEffectSpec() EffectSpec {
	spec := strEffectSpecAttached("버서크", "effect\\ac_concentration.wav", false)
	spec.CameraShake = 200 * time.Millisecond
	spec.CameraShakeDelay = 200 * time.Millisecond
	return spec
}

func robrCylinderComponent(textureName string, tint color.RGBA, duration time.Duration, alphaMax float64, animation int, bottomSize, topSize, height float64, attached, rotate, fade, additive bool) EffectComponent {
	blendMode := 0
	if additive {
		blendMode = 2
	}
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            tint,
		TextureName:      textureName,
		Duration:         duration,
		AlphaMax:         alphaMax,
		Fade:             fade,
		Rotate:           rotate,
		Animation:        animation,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		AttachedEntity:   attached,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        blendMode,
		BlendAdditive:    additive,
	}
}

func tintedEffectComponent(component EffectComponent, tint color.RGBA) EffectComponent {
	component.Color = tint
	return component
}

func quadHornEffectComponent(textureFile string, duration time.Duration, height, offsetX, offsetY [2]float64, offsetZ float64, bottomSize, rotateY, rotateZ [2]float64, animation int, animationSpeed time.Duration, animationOut bool, blendMode int) EffectComponent {
	return quadHornEffectComponentFull(textureFile, duration, height, offsetX, offsetY, offsetZ, bottomSize, [2]float64{}, rotateY, rotateZ, animation, animationSpeed, animationOut, blendMode)
}

func quadHornEffectComponentFull(textureFile string, duration time.Duration, height, offsetX, offsetY [2]float64, offsetZ float64, bottomSize, rotateX, rotateY, rotateZ [2]float64, animation int, animationSpeed time.Duration, animationOut bool, blendMode int) EffectComponent {
	return EffectComponent{
		Kind:               EffectComponentQuadHorn,
		TextureFile:        textureFile,
		Duration:           duration,
		AttachedEntity:     false,
		Color:              color.RGBA{R: 255, G: 255, B: 255, A: 255},
		BlendMode:          blendMode,
		BlendAdditive:      blendMode == 2,
		Animation:          animation,
		QuadHornHeightMin:  height[0],
		QuadHornHeightMax:  height[1],
		QuadHornOffsetXMin: offsetX[0],
		QuadHornOffsetXMax: offsetX[1],
		QuadHornOffsetYMin: offsetY[0],
		QuadHornOffsetYMax: offsetY[1],
		QuadHornOffsetZ:    offsetZ,
		QuadHornBottomMin:  bottomSize[0],
		QuadHornBottomMax:  bottomSize[1],
		QuadHornRotateXMin: rotateX[0],
		QuadHornRotateXMax: rotateX[1],
		QuadHornRotateYMin: rotateY[0],
		QuadHornRotateYMax: rotateY[1],
		QuadHornRotateZMin: rotateZ[0],
		QuadHornRotateZMax: rotateZ[1],
		QuadHornAnimSpeed:  animationSpeed,
		QuadHornAnimOut:    animationOut,
	}
}

func iceWallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 5 * time.Minute,
		SFX:      []string{"effect\\wizard_icewall.wav"},
		Components: []EffectComponent{
			quadHornEffectComponent("effect/ice.tga", 5*time.Minute, [2]float64{2.8, 3.3}, [2]float64{0.25, 0.75}, [2]float64{0.25, 0.75}, -0.1, [2]float64{0.3, 0.5}, [2]float64{1, 360}, [2]float64{}, 1, 50*time.Millisecond, false, 8),
			quadHornEffectComponent("effect/ice.tga", 5*time.Minute, [2]float64{2.3, 2.8}, [2]float64{0.25, 0.75}, [2]float64{0.25, 0.75}, -0.1, [2]float64{0.3, 0.5}, [2]float64{1, 360}, [2]float64{}, 1, 50*time.Millisecond, false, 8),
			quadHornEffectComponent("effect/ice.tga", 5*time.Minute, [2]float64{2.5, 2.9}, [2]float64{0.25, 0.75}, [2]float64{0.25, 0.75}, -0.1, [2]float64{0.3, 0.5}, [2]float64{1, 360}, [2]float64{}, 1, 50*time.Millisecond, false, 8),
		},
	}
}

func earthSpikeEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    5 * time.Second,
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\wizard_earthspike.wav"},
		Components: []EffectComponent{
			quadHornEffectComponent("effect/stone.bmp", 5*time.Second, [2]float64{0.95, 1.5}, [2]float64{0.4, 0.6}, [2]float64{0.4, 0.6}, -0.1, [2]float64{0.5, 0.6}, [2]float64{1, 360}, [2]float64{-8, 8}, 3, 120*time.Millisecond, true, 1),
			quadHornEffectComponent("effect/stone.bmp", 5*time.Second, [2]float64{0.2, 0.4}, [2]float64{0, 0}, [2]float64{0.5, 0.5}, -0.1, [2]float64{0.1, 0.2}, [2]float64{1, 360}, [2]float64{-15, 15}, 2, 100*time.Millisecond, true, 1),
			quadHornEffectComponent("effect/stone.bmp", 5*time.Second, [2]float64{0.2, 0.4}, [2]float64{0, 0.5}, [2]float64{0.5, 1.0}, -0.1, [2]float64{0.1, 0.2}, [2]float64{1, 360}, [2]float64{-15, 15}, 2, 100*time.Millisecond, true, 1),
			quadHornEffectComponent("effect/stone.bmp", 5*time.Second, [2]float64{0.2, 0.4}, [2]float64{1.0, 1.2}, [2]float64{0.5, 0.8}, -0.1, [2]float64{0.1, 0.2}, [2]float64{1, 360}, [2]float64{-15, 15}, 2, 100*time.Millisecond, true, 1),
			quadHornEffectComponent("effect/stone.bmp", 5*time.Second, [2]float64{0.2, 0.4}, [2]float64{0.5, 0.7}, [2]float64{0.0, -0.2}, -0.1, [2]float64{0.1, 0.2}, [2]float64{1, 360}, [2]float64{-15, 15}, 2, 100*time.Millisecond, true, 1),
		},
	}
}

func warpZoneEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 2800 * time.Millisecond,
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 2800*time.Millisecond, 0.3, 3, 2, 3.3, 1.1, true, true, true, true),
			robrCylinderComponent("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 2800*time.Millisecond, 0.3, 3, 1.9, 3.2, 1.1, true, true, true, true),
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/pok1.tga",
				Duration:       time.Second,
				Duplicate:      3,
				AlphaMax:       1,
				FadeIn:         true,
				FadeOut:        true,
				PosXStartRand:  3,
				PosYStartRand:  3,
				PosZ:           0,
				PosZEndRand:    2,
				PosZEndMiddle:  2,
				SizeStart:      effectTableSize(100),
				SizeEnd:        effectTableSize(100),
				SizeRand:       effectTableSize(17),
				AttachedEntity: true,
				Color:          color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
		},
	}
}

func sightTrasherEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:   800 * time.Millisecond,
		SFX:        []string{"effect\\wizard_sightrasher.wav"},
		Components: sightTrasherComponents(),
	}
}

func sightTrasherComponents() []EffectComponent {
	directions := [][2]float64{
		{0, -8},
		{5.66, -5.66},
		{8, 0},
		{5.66, 5.66},
		{0, 8},
		{-5.66, 5.66},
		{-8, 0},
		{-5.66, -5.66},
	}
	components := make([]EffectComponent, 0, len(directions)*2)
	for _, direction := range directions {
		components = append(components, sightTrasherComponent(true, direction[0], direction[1]), sightTrasherComponent(false, direction[0], direction[1]))
	}
	return components
}

func sightTrasherComponent(shadow bool, xEnd, yEnd float64) EffectComponent {
	component := EffectComponent{
		Kind:           EffectComponent3D,
		Duration:       800 * time.Millisecond,
		Duplicate:      4,
		DuplicateDelay: 100 * time.Millisecond,
		AlphaMax:       0.5,
		PosXEnd:        xEnd,
		PosYEnd:        yEnd,
		PosZ:           2,
		PosZEnd:        2,
		SizeStart:      effectTableSize(60),
		SizeEnd:        effectTableSize(160),
		SizeDelta:      -60,
		FadeIn:         true,
		FadeOut:        true,
		SpriteFile:     "data\\sprite\\shadow",
		ShadowTexture:  true,
		SpriteRepeat:   true,
		BlendMode:      8,
	}
	if !shadow {
		component.SpriteFile = "sight"
		component.ShadowTexture = false
		component.AlphaMax = 123.0 / 255.0
		component.AlphaMaxDelta = 3.0 / 255.0
		component.SizeStart = effectTableSize(20)
		component.SizeEnd = effectTableSize(260)
	}
	return component
}

func jupitelThunderEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 200 * time.Millisecond,
		SFX:      []string{"effect\\hunter_shockwavetrap.wav"},
		Components: []EffectComponent{
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/thunder_center.bmp",
				Duration:      200 * time.Millisecond,
				ToSrc:         true,
				BlendMode:     2,
				BlendAdditive: true,
				Overlay:       true,
				AlphaMax:      0.66,
				SizeStart:     effectTableSize(35),
				SizeEnd:       effectTableSize(35),
			},
			{
				Kind: EffectComponent3D,
				TextureFiles: []string{
					"effect/thunder_ball_a.bmp",
					"effect/thunder_ball_b.bmp",
					"effect/thunder_ball_c.bmp",
					"effect/thunder_ball_d.bmp",
					"effect/thunder_ball_e.bmp",
					"effect/thunder_ball_f.bmp",
				},
				FrameDelay:    10 * time.Millisecond,
				Duration:      200 * time.Millisecond,
				ToSrc:         true,
				BlendMode:     2,
				BlendAdditive: true,
				Overlay:       true,
				SizeStart:     effectTableSize(45),
				SizeEnd:       effectTableSize(45),
			},
		},
	}
}

func jupitelHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{
			{
				Kind:           EffectComponent3D,
				TextureFile:    "effect/thunder_pang.bmp",
				Duration:       100 * time.Millisecond,
				SizeStart:      0,
				SizeEnd:        effectTableSize(25),
				BlendMode:      2,
				BlendAdditive:  true,
				RotateToTarget: true,
				FadeOut:        true,
				Overlay:        true,
				AttachedEntity: true,
			},
			{
				Kind: EffectComponent3D,
				TextureFiles: []string{
					"effect/thunder_plazma_blast_a.bmp",
					"effect/thunder_plazma_blast_b.bmp",
					"effect/thunder_ball_d.bmp",
					"effect/thunder_ball_e.bmp",
					"effect/thunder_ball_f.bmp",
				},
				FrameDelay:     10 * time.Millisecond,
				Duration:       300 * time.Millisecond,
				SizeStart:      effectTableSize(75),
				SizeEnd:        effectTableSize(75),
				BlendMode:      2,
				BlendAdditive:  true,
				Overlay:        true,
				AttachedEntity: true,
			},
		},
	}
}

func repairWeaponEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1820 * time.Millisecond,
		SFX:      []string{"effect\\black_weapon_repair_a.wav", "effect\\black_weapon_repair_a.wav"},
		SFXDelays: []time.Duration{
			480 * time.Millisecond,
			1320 * time.Millisecond,
		},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "repairweapon",
			AttachedEntity: true,
		}},
	}
}

func crashEarthEffectSpec() EffectSpec {
	spec := strEffectSpec("crashearth", "effect\\black_hammerfall.wav")
	spec.CameraShake = 650 * time.Millisecond
	spec.CameraShakeDelay = 350 * time.Millisecond
	return spec
}

func waterBallEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\wizard_waterball_chulung.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponent3D,
			TextureFiles: []string{
				"effect/water_out_a.bmp",
				"effect/water_out_b.bmp",
				"effect/water_out_c.bmp",
			},
			FrameDelay:       10 * time.Millisecond,
			Duration:         500 * time.Millisecond,
			FadeOut:          true,
			PosXRand:         1.5,
			PosZRand:         1.5,
			PosY:             0,
			PosYEnd:          3,
			PosYSmooth:       true,
			SizeStart:        effectTableSize(30.5),
			SizeEnd:          effectTableSize(30.5),
			RotateWithCamera: true,
			BlendMode:        2,
			BlendAdditive:    true,
			AttachedEntity:   true,
		}},
	}
}

func waterBall2EffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 1450 * time.Millisecond,
		SFX:      []string{"effect\\wizard_waterball_chulung.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "data\\sprite\\이팩트\\waterball",
			Duration:       500 * time.Millisecond,
			Duplicate:      20,
			DuplicateDelay: 50 * time.Millisecond,
			FromSrc:        true,
			RotateToTarget: true,
			FadeOut:        true,
			SizeStart:      effectTableSize(50),
			SizeEnd:        effectTableSize(50),
			PosZ:           5,
			PosZEnd:        0.0001,
			Arc:            7.5,
			Retreat:        5,
		}},
	}
}

func sonicBlowEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 400 * time.Millisecond,
		SFX:      []string{"effect\\ef_stonecurse.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/ring2.bmp",
			Duration:       400 * time.Millisecond,
			AlphaMax:       1,
			FadeOut:        true,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(300),
			BlendMode:      2,
			BlendAdditive:  true,
			AttachedEntity: true,
		}},
	}
}

func sonicBlowHitEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "SonicBlowHitSpin",
			AttachedEntity: true,
		}},
	}
}

func grimtoothAttackEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 15 * time.Second,
		Components: []EffectComponent{
			quadHornEffectComponentFull("effect/stone.bmp", 15*time.Second, [2]float64{2.5, 2.5}, [2]float64{0, 0}, [2]float64{0.4, 0.4}, -0.2, [2]float64{0.15, 0.15}, [2]float64{-15, -15}, [2]float64{0, 0}, [2]float64{0, 0}, 3, 120*time.Millisecond, true, 1),
			quadHornEffectComponentFull("effect/stone.bmp", 15*time.Second, [2]float64{2.5, 2.5}, [2]float64{0, 0}, [2]float64{0.5, 0.5}, 0, [2]float64{0.15, 0.15}, [2]float64{5, 5}, [2]float64{45, 45}, [2]float64{-15, -15}, 3, 120*time.Millisecond, true, 1),
			quadHornEffectComponentFull("effect/stone.bmp", 15*time.Second, [2]float64{2.5, 2.5}, [2]float64{0, 0}, [2]float64{0.5, 0.5}, 0, [2]float64{0.15, 0.15}, [2]float64{5, 5}, [2]float64{45, 45}, [2]float64{15, 15}, 3, 120*time.Millisecond, true, 1),
		},
	}
}

func firePillarOnEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 6 * time.Second,
		Components: []EffectComponent{
			firePillarOnCylinder(1.0, 2.0, 3),
			firePillarOnCylinder(0.7, 1.5, 5),
			firePillarOnCylinder(0.5, 1.0, 7),
		},
	}
}

func firePillarOnCylinder(bottomSize, topSize, height float64) EffectComponent {
	component := robrCylinderComponent("magic_red", color.RGBA{}, 5*time.Second, 1, 0, bottomSize, topSize, height, false, true, false, false)
	component.Delay = time.Second
	return component
}

func heavenDriveEffectSpec() EffectSpec {
	return EffectSpec{
		Duration:    time.Second,
		CameraShake: 200 * time.Millisecond,
		SFX:         []string{"effect\\wizard_earthspike.wav"},
		Components:  heavenDriveComponents(),
	}
}

func heavenDriveComponents() []EffectComponent {
	components := make([]EffectComponent, 0, 25)
	for x := -2; x <= 2; x++ {
		for y := -2; y <= 2; y++ {
			component := quadHornEffectComponent("effect/stone.bmp", time.Second, [2]float64{0.75, 1.2}, [2]float64{0.2, 0.2}, [2]float64{0.2, 0.2}, -0.1, [2]float64{0.4, 0.7}, [2]float64{1, 360}, [2]float64{-8, 8}, 3, 250*time.Millisecond, true, 1)
			component.PosX = float64(x)
			component.PosY = float64(y)
			components = append(components, component)
		}
	}
	return components
}

func teleportCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      "ring_blue",
		Duration:         1500 * time.Millisecond,
		AlphaMax:         0.5,
		Fade:             true,
		Rotate:           true,
		Animation:        5,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}

func readyPortalCylinderComponent() EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      "ring_blue",
		Duration:         500 * time.Millisecond,
		Repeat:           true,
		RepeatDelay:      -300 * time.Millisecond,
		AlphaMax:         0.4,
		FadeOut:          true,
		Rotate:           true,
		Animation:        4,
		BottomSize:       2.4,
		TopSize:          3.9,
		Height:           0.1,
		PosZ:             0.1,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}

func portalCylinderComponent(bottomSize, topSize, height, posZ float64, textureName string, alphaMax float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
		TextureName:      textureName,
		Duration:         25000 * time.Millisecond,
		AlphaMax:         alphaMax,
		Fade:             true,
		Rotate:           true,
		Animation:        0,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		PosZ:             posZ,
		AttachedEntity:   true,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendMode:        2,
		BlendAdditive:    true,
	}
}

func healCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		Color:            color.RGBA{R: 178, G: 255, B: 178, A: 255},
		TextureName:      "ring_white",
		Duration:         1500 * time.Millisecond,
		AlphaMax:         0.2,
		Fade:             true,
		Rotate:           true,
		Animation:        1,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           height,
		TotalCircleSides: 32,
		CircleSides:      32,
		BlendAdditive:    true,
	}
}

func healOffensiveCylinderComponent(bottomSize, topSize, height float64) EffectComponent {
	component := healCylinderComponent(bottomSize, topSize, height)
	component.Duration = time.Second
	component.Color = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	component.BlendAdditive = true
	return component
}

func healParticleComponent(alpha float64, duration, delay, duplicateDelay time.Duration, duplicate int, posXRand, posYRand, posZStartRand, posZStartMiddle, posZEnd, posZEndRand, posZEndMiddle float64, sparkling bool) EffectComponent {
	component := EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/pok3.tga",
		Duration:        duration,
		Delay:           delay,
		DuplicateDelay:  duplicateDelay,
		AlphaMax:        alpha,
		Sparkling:       sparkling,
		FadeIn:          true,
		FadeOut:         true,
		PosXRand:        posXRand,
		PosYRand:        posYRand,
		PosZStartRand:   posZStartRand,
		PosZStartMiddle: posZStartMiddle,
		PosZEnd:         posZEnd,
		PosZEndRand:     posZEndRand,
		PosZEndMiddle:   posZEndMiddle,
		SizeStart:       9 * EffectPixelRatio,
		SizeEnd:         9 * EffectPixelRatio,
		SizeRand:        2 * EffectPixelRatio,
		Duplicate:       duplicate,
		BlendAdditive:   true,
	}
	if sparkling {
		component.SparkNumber = 2
	}
	return component
}

func coloredTorchEffectSpec(tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "torch_01",
			SpriteRepeat:   true,
			Duration:       600 * time.Millisecond,
			SpriteDelay:    100 * time.Millisecond,
			PosX:           0.1,
			PosZ:           0.8,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(100),
			AngleStart:     270,
			AngleEnd:       270,
			RotateToTarget: true,
			AlphaMax:       1,
			BlendAdditive:  true,
			Color:          tint,
		}},
	}
}

func mapGlowEffectSpec(tint color.RGBA, radius float64) EffectSpec {
	return EffectSpec{
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "alpha_center",
			Duration:         1600 * time.Millisecond,
			AlphaMax:         0.36,
			Fade:             true,
			Rotate:           true,
			BottomSize:       radius,
			TopSize:          radius,
			Height:           0.06,
			TotalCircleSides: 32,
			CircleSides:      32,
			BlendAdditive:    true,
			Color:            tint,
		}},
	}
}

func mapPillarEffectSpec(texture string, tint color.RGBA, radiusStart, alphaMax float64) EffectSpec {
	const (
		duration   = 16 * time.Second
		worldScale = 0.2
	)
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponentFUNC,
			FuncName:      "MapPillar",
			TextureName:   texture,
			Duration:      duration,
			AlphaMax:      alphaMax,
			BottomSize:    radiusStart * worldScale,
			Height:        120 * worldScale,
			BlendMode:     2,
			BlendAdditive: true,
			Color:         tint,
		}},
	}
}

func mapFallingParticleEffectSpec(textureFile string, tint color.RGBA, count int) EffectSpec {
	return EffectSpec{
		Duration: 1400 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    textureFile,
			Duration:       1400 * time.Millisecond,
			Duplicate:      count,
			DuplicateDelay: 18 * time.Millisecond,
			AlphaMax:       0.78,
			FadeIn:         true,
			FadeOut:        true,
			BlendAdditive:  true,
			PosXRand:       4,
			PosYRand:       4,
			PosZStartRand:  1.5,
			PosZ:           12,
			PosZEnd:        0.2,
			PosXEndRand:    1,
			PosYEndRand:    1,
			PosZSmooth:     true,
			SizeStart:      effectTableSize(8),
			SizeEnd:        effectTableSize(5),
			Color:          tint,
		}},
	}
}

func mapRainbowBand(tint color.RGBA, zOffset float64) EffectComponent {
	return EffectComponent{
		Kind:          EffectComponent3D,
		TextureFile:   "effect/alpha_center.tga",
		Duration:      2200 * time.Millisecond,
		AlphaMax:      0.3,
		Fade:          true,
		BlendAdditive: true,
		PosZ:          7 + zOffset,
		SizeStartX:    effectTableSize(900),
		SizeStartY:    effectTableSize(45),
		SizeEndX:      effectTableSize(900),
		SizeEndY:      effectTableSize(45),
		AngleStart:    -18,
		Color:         tint,
	}
}

func weatherRainEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/alpha_center.tga",
			Duration:      900 * time.Millisecond,
			Duplicate:     85,
			AlphaMax:      0.5,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      28,
			PosYRand:      28,
			PosZ:          18,
			PosZEnd:       0,
			PosXEnd:       -3.5,
			PosYEnd:       2.0,
			PosZSmooth:    false,
			SizeStartX:    effectTableSize(2),
			SizeStartY:    effectTableSize(52),
			SizeEndX:      effectTableSize(2),
			SizeEndY:      effectTableSize(52),
			AngleStart:    -25,
			Color:         color.RGBA{R: 185, G: 215, B: 255, A: 255},
		}},
	}
}

func weatherSnowEffectSpec() EffectSpec {
	return EffectSpec{
		Duration: 8 * time.Second,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFile:   "effect/pok3.tga",
			Duration:      8 * time.Second,
			Duplicate:     180,
			AlphaMax:      0.85,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: true,
			PosXRand:      60,
			PosYRand:      60,
			PosZStartRand: 2,
			PosZ:          20,
			PosZEnd:       -4,
			PosZSmooth:    false,
			SizeStart:     effectTableSize(7),
			SizeEnd:       effectTableSize(7),
			Color:         color.RGBA{R: 245, G: 250, B: 255, A: 255},
		}},
	}
}

func weatherLeafEffectSpec(spriteFile string, tint color.RGBA) EffectSpec {
	return EffectSpec{
		Duration: 5200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       spriteFile,
			SpriteRepeat:     true,
			Duration:         5200 * time.Millisecond,
			Duplicate:        36,
			AlphaMax:         0.92,
			FadeIn:           true,
			FadeOut:          true,
			PosXRand:         30,
			PosYRand:         30,
			PosZ:             20,
			PosZEnd:          0,
			PosXEnd:          -4,
			PosYEnd:          2,
			PosXEndRand:      3,
			PosYEndRand:      3,
			PosXSmooth:       true,
			PosYSmooth:       true,
			SizeStart:        effectTableSize(80),
			SizeEnd:          effectTableSize(80),
			AngleRandMin:     0,
			AngleRandMax:     360,
			RotateWithCamera: true,
			Color:            tint,
		}},
	}
}

func weatherCloudEffectSpec(textures []string, tint color.RGBA, alpha, radius, z float64, duration time.Duration) EffectSpec {
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  textures,
			Duration:      duration,
			Duplicate:     28,
			AlphaMax:      alpha,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			PosXRand:      radius,
			PosYRand:      radius,
			PosXEnd:       2.4,
			PosYEnd:       0.8,
			PosZ:          z,
			SizeStartX:    effectTableSize(1500),
			SizeStartY:    effectTableSize(900),
			SizeEndX:      effectTableSize(1500),
			SizeEndY:      effectTableSize(900),
			AngleRandMin:  -10,
			AngleRandMax:  10,
			Color:         tint,
		}},
	}
}

func einbrochWeatherCloudEffectSpec() EffectSpec {
	const duration = 34 * time.Second
	size := 35 * math.Sqrt2 * 0.1
	return EffectSpec{
		Duration: duration,
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			TextureFiles:  []string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"},
			Duration:      duration,
			Duplicate:     320,
			AlphaMax:      170.0 / 255.0,
			FadeIn:        true,
			FadeOut:       true,
			BlendAdditive: false,
			Overlay:       true,
			PosXRand:      15,
			PosYRand:      15,
			PosZRand:      0.5,
			PosXEnd:       1.5,
			PosYEnd:       1.5,
			PosZ:          -2,
			SizeStartX:    size,
			SizeStartY:    size,
			SizeEndX:      size,
			SizeEndY:      size,
			SizeRand:      10 * math.Sqrt2 * 0.1,
			Color:         color.RGBA{R: 252, G: 171, B: 143, A: 255},
		}},
	}
}

// EffectSpecs is adapted from robr's DB/Effects/EffectTable.js and verified
// original-client behavior. Leave effects unsupported until their reference
// behavior is understood rather than guessing local visuals.
var EffectSpecs = map[int]EffectSpec{
	effectRain:        weatherRainEffectSpec(),
	effectSnow:        weatherSnowEffectSpec(),
	effectSakura:      weatherLeafEffectSpec("data/sprite/이팩트/sakura01", color.RGBA{R: 255, G: 210, B: 225, A: 255}),
	effectMaple:       weatherLeafEffectSpec("data/sprite/이팩트/단풍", color.RGBA{R: 255, G: 170, B: 80, A: 255}),
	effectCloud:       weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.18, 35, -10, 20*time.Second),
	effectCloud2:      weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 20*time.Second),
	effectCloud3:      weatherCloudEffectSpec([]string{"effect/fog1.tga", "effect/fog2.tga", "effect/fog3.tga"}, color.RGBA{R: 120, G: 110, B: 100, A: 255}, 0.78, 45, -10, 34*time.Second),
	effectCloud4:      einbrochWeatherCloudEffectSpec(),
	effectCloud5:      weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 225, G: 212, B: 194, A: 255}, 0.70, 50, 4, 8*time.Second),
	effectCloud6:      weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.58, 35, 4, 28*time.Second),
	effectCloud7:      weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 51, G: 79, B: 161, A: 255}, 0.55, 35, 4, 20*time.Second),
	effectCloud8:      weatherCloudEffectSpec([]string{"effect/cloud4.tga", "effect/cloud1.tga", "effect/cloud2.tga"}, color.RGBA{R: 255, G: 140, B: 51, A: 255}, 0.62, 35, 4, 20*time.Second),
	effectBeginSpell:  castAuraEffectSpec("ring_yellow", color.RGBA{R: 255, G: 245, B: 120, A: 255}, 0.8, 4, 5, false),
	effectBeginSpell2: elementalCastAuraEffectSpec("ring_blue", color.RGBA{R: 128, G: 128, B: 255, A: 255}, 0.6),
	effectBeginSpell3: elementalCastAuraEffectSpec("ring_red", color.RGBA{R: 255, G: 100, B: 100, A: 255}, 0.7),
	effectBeginSpell4: elementalCastAuraEffectSpec("ring_white", color.RGBA{R: 150, G: 255, B: 150, A: 255}, 0.6),
	effectBeginSpell5: elementalCastAuraEffectSpec("ring_yellow", color.RGBA{R: 255, G: 245, B: 120, A: 255}, 0.8),
	effectBeginSpell6: castAuraEffectSpec("ring_white", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.8, 4, 5, true),
	effectBeginSpell7: elementalCastAuraEffectSpec("ring_purple", color.RGBA{R: 200, G: 160, B: 255, A: 255}, 0.7),
	effectLockOnTarget: {
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "LockOnTarget",
			TextureFile:    "effect/lockon128.tga",
			AttachedEntity: true,
		}},
	},
	effectSmoke: {
		Duration: 10 * time.Second,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "굴뚝연기",
			SpriteRepeat:     true,
			Duration:         10 * time.Second,
			Duplicate:        10,
			DuplicateDelay:   time.Second,
			AlphaMax:         0.8,
			FadeOut:          true,
			BlendAdditive:    true,
			PosZ:             0,
			PosZEnd:          20,
			PosXEndRand:      3,
			PosXSmooth:       true,
			SizeStart:        effectTableSize(70),
			SizeEnd:          effectTableSize(300),
			SizeSmooth:       true,
			AngleStart:       -90,
			AngleEnd:         0,
			Rotate:           true,
			RotateWithCamera: true,
			Color:            color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}},
	},
	effectFirefly: {
		Duration: 3 * time.Second,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "data/sprite/이팩트/particle1",
			SpriteRepeat:   true,
			Duration:       3 * time.Second,
			Duplicate:      3,
			DuplicateDelay: 420 * time.Millisecond,
			PosXRand:       0.18,
			PosYRand:       0.18,
			PosZ:           0.25,
			PosZEnd:        1.0,
			PosZStartRand:  0.2,
			PosZEndRand:    0.25,
			SizeStart:      effectTableSize(70),
			SizeEnd:        effectTableSize(110),
			AlphaMax:       0.22,
			FadeIn:         true,
			FadeOut:        true,
			Color:          color.RGBA{R: 230, G: 250, B: 255, A: 255},
		}},
	},
	effectTorch: {
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "torch_01",
			SpriteRepeat:   true,
			Duration:       600 * time.Millisecond,
			SpriteDelay:    100 * time.Millisecond,
			PosX:           0.1,
			PosZ:           0.8,
			SizeStart:      effectTableSize(100),
			SizeEnd:        effectTableSize(100),
			AngleStart:     270,
			AngleEnd:       270,
			RotateToTarget: true,
			AlphaMax:       1,
			Color:          color.RGBA{R: 255, G: 255, B: 255, A: 255},
		}},
	},
	effectTorchRed:    coloredTorchEffectSpec(color.RGBA{R: 255, G: 110, B: 75, A: 255}),
	effectTorchGreen:  coloredTorchEffectSpec(color.RGBA{R: 90, G: 255, B: 135, A: 255}),
	effectTorchPurple: coloredTorchEffectSpec(color.RGBA{R: 190, G: 120, B: 255, A: 255}),
	effectBubble:      strEffectSpecRandom("bubble%d", "", 1, 4),
	effectDragonSmoke: {
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "poisonhit",
			AttachedEntity: true,
		}},
	},
	effectBanjjakii:  banjjakiiEffectSpec(),
	effectMapPillar:  mapPillarEffectSpec("ring_blue", color.RGBA{R: 110, G: 175, B: 255, A: 255}, 2, 50.0/255.0),
	effectMapPillar2: mapPillarEffectSpec("ring_blue", color.RGBA{R: 110, G: 175, B: 255, A: 255}, 11, 70.0/255.0),
	effectMapPillar3: mapPillarEffectSpec("magic_green", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2, 50.0/255.0),
	effectMapPillar4: mapPillarEffectSpec("ring_red", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 2, 50.0/255.0),
	effectMapGhost: {
		Duration: 24 * time.Hour,
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			SpriteFile:     "particle2",
			SpriteRepeat:   true,
			Duration:       1400 * time.Millisecond,
			Duplicate:      7,
			DuplicateDelay: 190 * time.Millisecond,
			AlphaMax:       0.5,
			FadeIn:         true,
			FadeOut:        true,
			BlendAdditive:  true,
			PosZ:           0.2,
			PosZEnd:        2,
			PosXRand:       0.35,
			PosYRand:       0.35,
			SizeStart:      effectTableSize(70),
			SizeEnd:        effectTableSize(120),
			Color:          color.RGBA{R: 180, G: 230, B: 255, A: 255},
		}},
	},
	effectGlow1:      mapGlowEffectSpec(color.RGBA{R: 255, G: 245, B: 190, A: 255}, 1.2),
	effectGlow2:      mapGlowEffectSpec(color.RGBA{R: 150, G: 210, B: 255, A: 255}, 1.45),
	effectGlow4:      mapGlowEffectSpec(color.RGBA{R: 220, G: 175, B: 255, A: 255}, 1.8),
	effectBubbleDrop: mapFallingParticleEffectSpec("effect/pok3.tga", color.RGBA{R: 95, G: 180, B: 255, A: 255}, 30),
	effectRainbow: {
		Duration: 24 * time.Hour,
		Components: []EffectComponent{
			mapRainbowBand(color.RGBA{R: 255, G: 90, B: 90, A: 255}, 0),
			mapRainbowBand(color.RGBA{R: 255, G: 210, B: 90, A: 255}, 0.18),
			mapRainbowBand(color.RGBA{R: 90, G: 255, B: 120, A: 255}, 0.36),
			mapRainbowBand(color.RGBA{R: 90, G: 170, B: 255, A: 255}, 0.54),
			mapRainbowBand(color.RGBA{R: 185, G: 110, B: 255, A: 255}, 0.72),
		},
	},
	effectPneuma: strEffectSpecRandom("pneuma%d", "", 1, 3),
	effectCastRing: {
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponentFUNC,
			FuncName:         "CastRing",
			TextureName:      "ring_yellow",
			AlphaMax:         0.9,
			Fade:             true,
			Rotate:           true,
			Animation:        1,
			BottomSize:       0.8,
			TopSize:          2.45,
			Height:           2.8,
			PosZ:             0.08,
			TotalCircleSides: 20,
			CircleSides:      20,
			Color:            color.RGBA{R: 255, G: 245, B: 150, A: 255},
		}},
	},
	effectGroundSample: {
		Duration: 900 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponentFUNC,
			FuncName:    "MagicTarget",
			TextureFile: "effect/magic_target.tga",
			AlphaMax:    0.9,
			PosZ:        0.08,
			SizeStart:   1,
			SizeEnd:     1,
		}},
	},
	effectHit1: {
		Duration: 300 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			TextureFile: "effect/pok3.tga",
			Duration:    300 * time.Millisecond,
			Duplicate:   4,
			AlphaMax:    0.8,
			FadeIn:      true,
			FadeOut:     true,
			Sparkling:   true,
			PosZ:        1,
			PosXEndRand: 2,
			PosYEndRand: 2,
			PosZEndRand: 2,
			SizeStart:   effectTableSize(10),
			SizeEnd:     effectTableSize(10),
			SizeRand:    effectTableSize(20),
			SizeSmooth:  true,
		}, {
			// Sakexe/HighPriest CRagEffect::Hit1: native units / 5;
			// flip the X rotation because our vertical axis points up.
			Kind:           EffectComponentFUNC,
			FuncName:       "HitRing",
			TextureName:    "ring_blue",
			Duration:       10 * time.Second / 60,
			FrameDelay:     time.Second / 60,
			AlphaMax:       254.0 / 255,
			BottomSize:     1,
			TopSize:        2,
			Height:         0.7,
			PosZ:           2,
			AngleX:         90,
			CircleSides:    10,
			AttachedEntity: true,
		}},
	},
	effectBashHit: {
		Duration:   350 * time.Millisecond,
		SFX:        []string{"effect\\ef_hit2.wav"},
		Components: bashHitComponents(),
	},
	effectHit3: {
		Duration: 150 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit3.wav"},
		Components: []EffectComponent{
			hitCylinderComponent(0.37, 1),
			hitCylinderComponent(0.37, 0.37),
		},
	},
	effectHit4: {
		Duration: 150 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit4.wav"},
		Components: []EffectComponent{
			hitCylinderComponent(0.15, 1),
		},
	},
	effectHit5: {
		Duration: 400 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit5.wav"},
		Components: []EffectComponent{
			hitSlashComponent(EffectComponent3D, effectTableSize(15), effectTableSize(200), 90, 0, false),
			hitSlashComponent(EffectComponent3D, effectTableSize(15), effectTableSize(200), 180, 90, false),
		},
	},
	effectHit6: {
		Duration: 400 * time.Millisecond,
		SFX:      []string{"effect\\ef_hit6.wav"},
		Components: []EffectComponent{
			hitSlashComponent(EffectComponent2D, effectTableSize(10), effectTableSize(150), 90, 0, true),
			hitSlashComponent(EffectComponent2D, effectTableSize(10), effectTableSize(150), 180, 90, true),
		},
	},
	effectEntry: {
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\ef_portal.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{}, 500*time.Millisecond, 0.62, 1, 0.9, 0.9, 7.5, false, true, true, false),
			robrCylinderComponent("ring_blue", color.RGBA{}, 500*time.Millisecond, 0.62, 1, 0.85, 0.85, 8, true, true, true, false),
			robrCylinderComponent("ring_blue", color.RGBA{}, 500*time.Millisecond, 0.8, 1, 0.9, 1.5, 1, true, true, true, false),
		},
	},
	effectExit: {
		Duration: 2000 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("alpha_down", color.RGBA{}, 2000*time.Millisecond, 0.2, 1, 0.95, 0.95, 10, true, false, true, true),
			healParticleComponent(0.8, 1000*time.Millisecond, 400*time.Millisecond, 80*time.Millisecond, 6, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			healParticleComponent(0.8, 900*time.Millisecond, 200*time.Millisecond, 200*time.Millisecond, 3, 1, 1, 1, 0, 6, 0, 0, true),
		},
	},
	effectWarp: {
		Duration: 1000 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      "ring_yellow",
			Duration:         1000 * time.Millisecond,
			Duplicate:        4,
			DuplicateDelay:   300 * time.Millisecond,
			AlphaMax:         0.8,
			Fade:             true,
			Animation:        4,
			BottomSize:       10,
			TopSize:          13,
			PosZ:             0.1,
			AttachedEntity:   true,
			TotalCircleSides: 32,
			CircleSides:      32,
		}},
	},
	effectEnhance: {
		Duration: 2000 * time.Millisecond,
		Components: []EffectComponent{
			robrCylinderComponent("alpha_down", color.RGBA{}, 2000*time.Millisecond, 0.2, 1, 0.95, 0.95, 10, true, false, true, true),
			incAgilityParticleComponent(0.8, 500*time.Millisecond, 7),
			incAgilityParticleComponent(0.8, 400*time.Millisecond, 3),
		},
	},
	effectArrowShot: {
		Duration: 140 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/npc/skel_archer_arrow",
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Duration:         140 * time.Millisecond,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			PosZ:             1,
			SizeStart:        effectTableSize(100),
			SizeEnd:          effectTableSize(100),
			AngleStart:       180,
		}},
	},
	effectArrowShower: {
		Duration: 140 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "data/sprite/npc/skel_archer_arrow",
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Duration:         140 * time.Millisecond,
			Duplicate:        10,
			DuplicateDelay:   0,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			PosZ:             1,
			PosXEndRand:      1.5,
			PosYEndRand:      1.5,
			SizeStart:        effectTableSize(100),
			SizeEnd:          effectTableSize(100),
			AngleStart:       180,
		}},
	},
	effectSafetyWall: {
		Duration: 50 * time.Second,
		SFX:      []string{"effect\\ef_glasswall.wav"},
		Components: []EffectComponent{
			{
				Kind:    EffectComponentSTR,
				STRFile: "safetywall",
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_down",
				Duration:         50 * time.Second,
				AlphaMax:         0.4,
				Fade:             true,
				Rotate:           true,
				Animation:        0,
				BottomSize:       0.6,
				TopSize:          0.6,
				Height:           7,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            color.RGBA{R: 128, G: 25, B: 128, A: 255},
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_down",
				Duration:         50 * time.Second,
				Delay:            50 * time.Millisecond,
				AlphaMax:         0.4,
				Fade:             true,
				Rotate:           true,
				Animation:        0,
				BottomSize:       0.65,
				TopSize:          0.65,
				Height:           6,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            color.RGBA{R: 128, G: 25, B: 128, A: 255},
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_down",
				Duration:         50 * time.Second,
				Delay:            100 * time.Millisecond,
				AlphaMax:         0.4,
				Fade:             true,
				Rotate:           true,
				Animation:        0,
				BottomSize:       0.7,
				TopSize:          0.7,
				Height:           7,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            color.RGBA{R: 128, G: 25, B: 128, A: 255},
			},
		},
	},
	effectMammonite:      strEffectSpecAttachedMin("maemor", "memor_min", "effect\\ef_coin2.wav", false),
	effectCartRevolution: strEffectSpecAttached("cartrevolution", "effect\\ef_magnumbreak.wav", false),
	effectLoud:           strEffectSpecAttached("loud", "effect\\고성방가.wav", false),
	effectGlassWall:      strEffectSpec("effect/safetywall", "effect\\ef_glasswall.wav"),
	effectHealSP: {
		Duration: 2000 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{R: 25, G: 128, B: 255, A: 255}, 2000*time.Millisecond, 0.2, 1, 0.95, 0.95, 10, true, true, true, true),
			tintedEffectComponent(healParticleComponent(0.8, 1000*time.Millisecond, 400*time.Millisecond, 80*time.Millisecond, 6, 1.5, 1.5, 0, 0, 0, 3, 6, true), color.RGBA{R: 25, G: 128, B: 255, A: 255}),
			tintedEffectComponent(healParticleComponent(0.8, 900*time.Millisecond, 200*time.Millisecond, 200*time.Millisecond, 3, 1, 1, 1, 0, 6, 0, 0, true), color.RGBA{R: 25, G: 128, B: 255, A: 255}),
		},
	},
	effectSight: {
		Duration: 12200 * time.Millisecond,
		SFX:      []string{"effect\\ef_sight.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "data\\sprite\\shadow",
				ShadowTexture:   true,
				SpriteRepeat:    true,
				Duration:        12200 * time.Millisecond,
				AlphaMax:        0.5,
				PosX:            -2,
				PosZ:            4,
				SizeStart:       effectTableSize(30),
				SizeEnd:         effectTableSize(30),
				SizeDelta:       effectTableSize(10),
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  10,
				OrbitPhase:      0.9,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
				Duplicate:       10,
			},
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "sight",
				SpriteRepeat:    true,
				Duration:        12200 * time.Millisecond,
				AlphaMax:        123.0 / 255.0,
				AlphaMaxDelta:   3.0 / 255.0,
				PosX:            -2,
				PosZ:            4,
				SizeStart:       effectTableSize(60),
				SizeEnd:         effectTableSize(60),
				SizeDelta:       effectTableSize(20),
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  10,
				OrbitPhase:      0.9,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
				Duplicate:       10,
			},
		},
	},
	effectNapalmBeat: {
		Duration: 700 * time.Millisecond,
		SFX:      []string{"effect\\ef_napalmbeat.wav"},
	},
	effectPokJuk: {
		Duration: 4 * time.Second,
		Components: []EffectComponent{
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				AlphaMax:      0.9,
				FadeIn:        true,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          0,
				PosZEnd:       8,
				PosXEndRand:   1.5,
				PosYEndRand:   1.5,
				PosZSmooth:    true,
				SizeStart:     effectTableSize(8),
				SizeEnd:       effectTableSize(12),
				Color:         color.RGBA{R: 255, G: 255, B: 255, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				Delay:         850 * time.Millisecond,
				Duplicate:     45,
				AlphaMax:      0.9,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   4.5,
				PosYEndRand:   4.5,
				PosZEndRand:   4.5,
				SizeStart:     effectTableSize(14),
				SizeEnd:       effectTableSize(4),
				SizeRand:      effectTableSize(4),
				Color:         color.RGBA{R: 120, G: 180, B: 255, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      900 * time.Millisecond,
				Delay:         900 * time.Millisecond,
				Duplicate:     45,
				AlphaMax:      0.85,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   4.0,
				PosYEndRand:   4.0,
				PosZEndRand:   4.0,
				SizeStart:     effectTableSize(12),
				SizeEnd:       effectTableSize(3),
				SizeRand:      effectTableSize(3),
				Color:         color.RGBA{R: 255, G: 120, B: 160, A: 255},
			},
			{
				Kind:          EffectComponent3D,
				TextureFile:   "effect/pok3.tga",
				Duration:      750 * time.Millisecond,
				Delay:         950 * time.Millisecond,
				Duplicate:     30,
				AlphaMax:      0.75,
				FadeOut:       true,
				BlendAdditive: true,
				PosZ:          8,
				PosXEndRand:   3.2,
				PosYEndRand:   3.2,
				PosZEndRand:   3.2,
				SizeStart:     effectTableSize(10),
				SizeEnd:       effectTableSize(2),
				SizeRand:      effectTableSize(2),
				Color:         color.RGBA{R: 255, G: 255, B: 130, A: 255},
			},
		},
	},
	effectSoulStrike: {
		Duration: 450 * time.Millisecond,
		SFX:      []string{"effect\\ef_soulstrike.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
				TextureFile:     "effect/pok3.tga",
				Duration:        200 * time.Millisecond,
				Delay:           250 * time.Millisecond,
				DuplicateDelay:  150 * time.Millisecond,
				ToSrc:           true,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				PosZEnd:         1,
				PosZStartRand:   5,
				PosZStartMiddle: 6,
				PosZSmooth:      true,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
			},
			{
				Kind:           EffectComponent3D,
				SpriteFile:     "data/sprite/이팩트/particle1",
				SpriteRepeat:   true,
				Duration:       250 * time.Millisecond,
				Duplicate:      5,
				DuplicateDelay: 20 * time.Millisecond,
				ToSrc:          true,
				RotateToTarget: true,
				Arc:            4,
				Retreat:        3,
				PosZ:           3,
				SizeStart:      100 * EffectPixelRatio,
				SizeEnd:        effectTableSize(500),
			},
		},
	},
	effectEndure: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_endure.wav"},
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			TextureFile: "effect/endure.tga",
			Duration:    1000 * time.Millisecond,
			AlphaMax:    1,
			FadeIn:      true,
			FadeOut:     true,
			PosZ:        2,
			SizeStart:   200 * EffectPixelRatio,
			SizeEnd:     70 * EffectPixelRatio,
			SizeSmooth:  true,
		}},
	},
	effectBashBegin: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_bash.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_down",
				Duration:         1000 * time.Millisecond,
				AlphaMax:         0.6,
				Fade:             true,
				Rotate:           true,
				FixedPerspective: true,
				Animation:        2,
				BottomSize:       0.1,
				TopSize:          2.0,
				PosZ:             1.5,
				TotalCircleSides: 20,
				CircleSides:      20,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_center",
				Duration:         1000 * time.Millisecond,
				AlphaMax:         0.6,
				Fade:             true,
				Rotate:           true,
				FixedPerspective: true,
				Animation:        2,
				BottomSize:       0.01,
				TopSize:          2.5,
				PosZ:             1.5,
				TotalCircleSides: 30,
				CircleSides:      1,
				Duplicate:        10,
				AngleZRandom:     360,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "alpha_center",
				Duration:         1000 * time.Millisecond,
				AlphaMax:         0.6,
				Fade:             true,
				Rotate:           true,
				FixedPerspective: true,
				Animation:        2,
				BottomSize:       0.01,
				TopSize:          4.0,
				PosZ:             1.5,
				TotalCircleSides: 30,
				CircleSides:      1,
				Duplicate:        8,
				AngleZRandom:     360,
			},
		},
	},
	effectProvoke: {
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\swordman_provoke.wav"},
		Components: []EffectComponent{{
			Kind:    EffectComponentSTR,
			Color:   color.RGBA{R: 255, G: 70, B: 42, A: 255},
			STRFile: "provoke",
		}},
	},
	effectMagnumBreak: {
		Duration: 300 * time.Millisecond,
		SFX:      []string{"effect\\ef_magnumbreak.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "ring_yellow",
				Duration:         300 * time.Millisecond,
				AlphaMax:         0.7,
				Fade:             true,
				Rotate:           true,
				Animation:        4,
				BottomSize:       4,
				TopSize:          6,
				Height:           1,
				TotalCircleSides: 32,
				CircleSides:      32,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "대폭발",
				Duration:         300 * time.Millisecond,
				AlphaMax:         0.6,
				Fade:             true,
				Rotate:           true,
				Animation:        4,
				BottomSize:       4,
				TopSize:          1,
				Height:           4,
				TotalCircleSides: 32,
				CircleSides:      32,
			},
		},
	},
	effectQuakeMagnum: {
		Duration:    50 * time.Millisecond,
		CameraShake: 50 * time.Millisecond,
	},
	effectSteal: {
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\ef_steal.wav"},
		Components: []EffectComponent{{
			Kind:          EffectComponent3D,
			Color:         color.RGBA{R: 255, G: 255, B: 216, A: 255},
			TextureFile:   "effect/pok1.tga",
			Duration:      500 * time.Millisecond,
			AlphaMax:      1,
			FadeOut:       true,
			PosXEndRand:   3.5,
			PosYEndRand:   3.5,
			PosZEndRand:   1,
			PosZEndMiddle: 3,
			SizeStart:     200 * EffectPixelRatio,
			SizeEnd:       10 * EffectPixelRatio,
			Duplicate:     7,
		}},
	},
	effectThrowItem3: {
		Duration: 200 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "유저인터페이스/item/돌.bmp",
			Duration:         200 * time.Millisecond,
			AlphaMax:         1,
			FadeIn:           true,
			FadeOut:          true,
			ToSrc:            true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			Rotate:           true,
			AngleStart:       180,
			AngleEnd:         360,
			PosZ:             1,
			SizeStart:        effectTableSize(20),
			SizeEnd:          effectTableSize(20),
		}},
	},
	effectSummonSlave: {
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "smoke",
			AttachedEntity: true,
		}},
	},
	effectPoisonAttack: {
		Duration: 2800 * time.Millisecond,
		SFX:      []string{"effect\\ef_detoxication.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			Color:          color.RGBA{R: 255, G: 178, B: 255, A: 255},
			TextureFile:    "effect/pok1.tga",
			Duration:       1000 * time.Millisecond,
			AlphaMax:       1,
			FadeIn:         true,
			FadeOut:        true,
			PosXRand:       1,
			PosYRand:       1,
			PosZEnd:        5,
			SizeStart:      100 * EffectPixelRatio,
			SizeEnd:        100 * EffectPixelRatio,
			SizeRand:       20 * EffectPixelRatio,
			Duplicate:      10,
			DuplicateDelay: 200 * time.Millisecond,
		}},
	},
	effectDetoxication: {
		Duration: 2800 * time.Millisecond,
		SFX:      []string{"effect\\ef_detoxication.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			Color:          color.RGBA{R: 178, G: 255, B: 255, A: 255},
			TextureFile:    "effect/pok1.tga",
			Duration:       1000 * time.Millisecond,
			AlphaMax:       1,
			FadeIn:         true,
			FadeOut:        true,
			PosXRand:       1,
			PosYRand:       1,
			PosZEnd:        5,
			SizeStart:      100 * EffectPixelRatio,
			SizeEnd:        100 * EffectPixelRatio,
			SizeRand:       20 * EffectPixelRatio,
			Duplicate:      10,
			DuplicateDelay: 200 * time.Millisecond,
		}},
	},
	effectStoneCurse: strEffectSpecAttached("stonecurse", "", false),
	effectIceArrow: {
		Duration:   500 * time.Millisecond,
		SFX:        []string{"effect\\ef_icearrow%d.wav"},
		SFXRandMin: 1,
		SFXRandMax: 3,
	},
	effectColdBolt: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_icearrow1.wav", "effect\\ef_icearrow2.wav", "effect\\ef_icearrow3.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				TextureFile:     "effect/icearrow.tga",
				Duration:        500 * time.Millisecond,
				AlphaMax:        1,
				PosXStartMiddle: 5,
				PosXStartRand:   1,
				PosYStartMiddle: 2,
				PosYStartRand:   1,
				PosZ:            20,
				PosXEnd:         0.0001,
				PosYEnd:         0.0001,
				PosZEnd:         0.0001,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
				AngleStart:      112.5,
				AngleEnd:        112.5,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      "ring_blue",
				Duration:         1000 * time.Millisecond,
				Delay:            500 * time.Millisecond,
				AlphaMax:         0.7,
				Fade:             true,
				Rotate:           true,
				Animation:        4,
				BottomSize:       3,
				TopSize:          5,
				Height:           0.1,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            color.RGBA{R: 128, G: 170, B: 255, A: 255},
			},
		},
	},
	effectFireBolt: {
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\ef_firearrow1.wav", "effect\\ef_firearrow2.wav", "effect\\ef_firearrow3.wav"},
		Components: []EffectComponent{{
			Kind: EffectComponent3D,
			TextureFiles: []string{
				"effect/불화살1.tga",
				"effect/불화살2.tga",
				"effect/불화살3.tga",
				"effect/불화살4.tga",
				"effect/불화살5.tga",
				"effect/불화살6.tga",
			},
			Duration:        500 * time.Millisecond,
			AlphaMax:        1,
			BlendAdditive:   true,
			PosXStartMiddle: 5,
			PosXStartRand:   1,
			PosYStartMiddle: 2,
			PosYStartRand:   1,
			PosZ:            20,
			PosXEnd:         0.0001,
			PosYEnd:         0.0001,
			PosZEnd:         0.0001,
			SizeStartX:      100 * EffectPixelRatio,
			SizeEndX:        100 * EffectPixelRatio,
			SizeStartY:      50 * EffectPixelRatio,
			SizeEndY:        50 * EffectPixelRatio,
			AngleStart:      112.5,
			AngleEnd:        112.5,
		}},
	},
	effectFireBall: {
		Duration: 410 * time.Millisecond,
		SFX:      []string{"effect\\ef_fireball.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			SpriteFile:       "fireball",
			SpriteRepeat:     true,
			ToSrc:            true,
			RotateToTarget:   true,
			Duration:         250 * time.Millisecond,
			Delay:            160 * time.Millisecond,
			DelayOffsetDelta: -40 * time.Millisecond,
			AlphaMax:         0.2,
			AlphaMaxDelta:    0.2,
			RotateWithCamera: true,
			PosZ:             2,
			SizeStart:        200 * EffectPixelRatio,
			SizeEnd:          200 * EffectPixelRatio,
			Duplicate:        5,
		}},
	},
	effectFireWall: strEffectSpecRandom("firewall%d", "effect\\ef_firewall.wav", 1, 2),
	effectFrostDiver: {
		Duration: 450 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponent3D,
			TextureFile: "effect/ice.tga",
			Duration:    450 * time.Millisecond,
			ToSrc:       true,
			AlphaMax:    0.75,
			FadeOut:     true,
			PosZ:        0.7,
			PosZEnd:     0.9,
			SizeStart:   90 * EffectPixelRatio,
			SizeEnd:     130 * EffectPixelRatio,
			AngleStart:  0,
			AngleEnd:    -180,
		}},
	},
	effectFrostDiverHit: strEffectSpecAttached("freeze", "effect\\ef_frostdiver2.wav", false),
	effectLightningBolt: strEffectSpecAttached("lightning", "", false),
	effectThunderStorm:  strEffectSpec("thunderstorm", "effect\\magician_thunderstorm.wav"),
	effectFireArrow:     soundOnlyEffectSpec("effect\\ef_firearrow1.wav"),
	effectTeleportOld: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_teleportation.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("ring_blue", color.RGBA{R: 255, G: 255, B: 255, A: 255}, 1000*time.Millisecond, 0.5, 5, 0.8, 0.7, 35, true, true, true, false),
		},
	},
	effectReadyPortalOld: {
		Duration: 25000 * time.Millisecond,
		SFX:      []string{"effect\\ef_readyportal.wav"},
		Components: []EffectComponent{
			robrCylinderComponent("alpha_down", color.RGBA{R: 178, G: 178, B: 255, A: 255}, 25000*time.Millisecond, 0.6, 0, 0.6, 0.6, 15, true, true, true, false),
		},
	},
	effectRuwach: {
		Duration: 12200 * time.Millisecond,
		SFX:      []string{"effect\\ef_ruwach.wav"},
		Components: []EffectComponent{
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "data\\sprite\\shadow",
				ShadowTexture:   true,
				Duration:        12200 * time.Millisecond,
				AlphaMax:        0.5,
				PosX:            -2,
				SizeStart:       effectTableSize(0.4),
				SizeEnd:         effectTableSize(0.4),
				SizeDelta:       0.15,
				Duplicate:       8,
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  8,
				OrbitPhase:      0.7,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
			},
			{
				Kind:            EffectComponent3D,
				Color:           color.RGBA{G: 255, B: 255, A: 255},
				SpriteFile:      "particle2",
				Duration:        12200 * time.Millisecond,
				AlphaMax:        0.07,
				AlphaMaxDelta:   0.07,
				BlendAdditive:   true,
				PosX:            -2,
				PosZ:            2,
				SizeStart:       effectTableSize(80),
				SizeEnd:         effectTableSize(80),
				SizeDelta:       30,
				Duplicate:       10,
				OrbitRadiusX:    3,
				OrbitRadiusY:    3,
				OrbitRotations:  8,
				OrbitPhase:      0.7,
				OrbitPhaseDelta: -0.1,
				OrbitClockwise:  true,
			},
		},
	},
	effectIncAgility: {
		Duration: 1500 * time.Millisecond,
		SFX:      []string{"effect\\ef_incagility.wav"},
		Components: []EffectComponent{
			incAgilityParticleComponent(1, 500*time.Millisecond, 7),
			incAgilityParticleComponent(0.75, 400*time.Millisecond, 3),
			incAgilityParticleComponent(1, 0, 10),
			{
				Kind:        EffectComponent3D,
				Color:       color.RGBA{R: 255, G: 255, B: 255, A: 255},
				TextureFile: "effect/agi_up.bmp",
				Duration:    1000 * time.Millisecond,
				AlphaMax:    1,
				FadeIn:      true,
				FadeOut:     true,
				PosZ:        0.4,
				PosZEnd:     3,
				SizeStart:   100 * EffectPixelRatio,
				SizeEnd:     100 * EffectPixelRatio,
				SizeStartY:  45 * EffectPixelRatio,
				SizeEndY:    45 * EffectPixelRatio,
				SizeSmooth:  true,
				Overlay:     true,
			},
		},
	},
	effectDecAgility: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_decagility.wav"},
		Components: []EffectComponent{
			decAgilityParticleComponent(),
			{
				Kind:        EffectComponent3D,
				TextureFile: "effect/slow.bmp",
				Duration:    1000 * time.Millisecond,
				AlphaMax:    1,
				FadeIn:      true,
				FadeOut:     true,
				PosZ:        2.8,
				PosZEnd:     0.4,
				SizeStart:   effectTableSize(100),
				SizeEnd:     effectTableSize(100),
				SizeStartY:  effectTableSize(45),
				SizeEndY:    effectTableSize(45),
				SizeSmooth:  true,
			},
		},
	},
	effectIncAgiDex: {
		Duration: 1000 * time.Millisecond,
		SFX:      []string{"effect\\ef_incagidex.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponent3D,
			TextureFile:    "effect/dex_agi_up.bmp",
			Duration:       1000 * time.Millisecond,
			AlphaMax:       1,
			FadeIn:         true,
			FadeOut:        true,
			PosZ:           0.4,
			PosZEnd:        3,
			SizeStart:      100 * EffectPixelRatio,
			SizeEnd:        100 * EffectPixelRatio,
			SizeStartY:     45 * EffectPixelRatio,
			SizeEndY:       45 * EffectPixelRatio,
			SizeSmooth:     true,
			AttachedEntity: true,
			Overlay:        true,
		}},
	},
	effectAqua: {
		SFX: []string{"effect\\ef_aqua.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentSPR,
			SpriteFile:       "성수뜨기",
			SpriteHead:       true,
			WorldSizedSprite: true,
		}},
	},
	effectSignum:  strEffectSpecAttached("cross", "effect\\ef_signum.wav", false),
	effectAngelus: strEffectSpecAttachedMin("angelus", "jong_mini", "effect\\ef_angelus.wav", true),
	effectGloria:  strEffectSpecAttachedMin("gloria", "gloria_min", "effect\\priest_gloria.wav", false),
	effectBlessing: {
		Duration: 2500 * time.Millisecond,
		SFX:      []string{"effect\\ef_blessing.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentSPR,
				SpriteFile:       "축복",
				Duration:         1500 * time.Millisecond,
				SpriteDelay:      30 * time.Millisecond,
				SpriteRepeat:     true,
				SpriteHead:       true,
				SpriteYOffset:    -120,
				WorldSizedSprite: true,
			},
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "particle6",
				Duration:        1200 * time.Millisecond,
				Delay:           300 * time.Millisecond,
				Duplicate:       6,
				DuplicateDelay:  0,
				AlphaMax:        1,
				Sparkling:       true,
				SparkNumber:     2,
				FadeIn:          true,
				FadeOut:         true,
				PosXRand:        1.2,
				PosYRand:        1,
				PosZStartRand:   2,
				PosZStartMiddle: 5.5,
				PosZEndRand:     0.5,
				PosZEndMiddle:   1,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
			},
			{
				Kind:            EffectComponent3D,
				SpriteFile:      "particle6",
				Duration:        1200 * time.Millisecond,
				Delay:           400 * time.Millisecond,
				Duplicate:       6,
				DuplicateDelay:  0,
				AlphaMax:        1,
				FadeIn:          true,
				FadeOut:         true,
				PosXRand:        1.4,
				PosYRand:        1.1,
				PosZStartRand:   2,
				PosZStartMiddle: 5.5,
				PosZEndRand:     0.5,
				PosZEndMiddle:   1,
				SizeStart:       50 * EffectPixelRatio,
				SizeEnd:         50 * EffectPixelRatio,
			},
			{
				Kind:          EffectComponent3D,
				Color:         color.RGBA{R: 25, G: 191, B: 255, A: 255},
				TextureFile:   "effect/pok2.tga",
				Duration:      2500 * time.Millisecond,
				AlphaMax:      0.3,
				FadeIn:        true,
				FadeOut:       true,
				SizeStart:     140 * EffectPixelRatio,
				SizeEnd:       140 * EffectPixelRatio,
				BlendAdditive: true,
			},
		},
	},
	effectFireHit: strEffectSpecRandomAttached("firehit%d", "effect\\ef_firehit.wav", 1, 3, true, false),
	effectFireSplashHit: {
		Duration: 500 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:        EffectComponent2D,
			TextureFile: "effect/firering.tga",
			Duration:    500 * time.Millisecond,
			FadeOut:     true,
			PosZ:        1,
			SizeStart:   10 * EffectPixelRatio,
			SizeEnd:     300 * EffectPixelRatio,
			AngleStart:  0,
			AngleEnd:    -360,
		}},
	},
	effectColdHit: soundOnlyEffectSpec("_hit_fist3.wav", "_hit_fist4.wav"),
	effectWindHit: strEffectSpecRandomAttached("windhit%d", "_hit_fist%d.wav", 1, 3, true, false),
	effectPoisonHit: {
		SFX: []string{"effect\\ef_poisonattack.wav"},
		Components: []EffectComponent{{
			Kind:       EffectComponentSPR,
			SpriteFile: "poisonhit",
		}},
	},
	effectWarpZone:       warpZoneEffectSpec(),
	effectSightTrasher:   sightTrasherEffectSpec(),
	effectArrowShotRO:    strEffectSpecAttached("arrowshot", "", false),
	effectInvenom:        strEffectSpecAttached("invenom", "effect\\thief_invenom.wav", false),
	effectSkidTrap:       strEffectSpec("skidtrap", "effect\\hunter_skidtrap.wav"),
	effectBrandishSpear:  strEffectSpec("brandish", "effect\\knight_brandish_spear.wav"),
	effectCure:           strEffectSpecAttachedMin("cure", "cure_min", "effect\\acolyte_cure.wav", false),
	effectMvp:            strEffectSpecAttached("mvp", "effect\\st_mvp.wav", false),
	effectIceWall:        iceWallEffectSpec(),
	effectMagnificat:     strEffectSpecAttachedMin("magnificat", "magnificat_min", "effect\\priest_magnificat.wav", false),
	effectResurrection:   strEffectSpecAttachedMin("resurrection", "resurrection_min", "effect\\priest_resurrection.wav", false),
	effectRecovery:       strEffectSpecAttached("recovery", "effect\\priest_recovery.wav", false),
	effectEarthSpike:     earthSpikeEffectSpec(),
	effectSpearBoomerang: soundOnlyEffectSpec("effect\\ef_fireball.wav"),
	effectPierce:         soundOnlyEffectSpec("effect\\ef_bash.wav"),
	effectTurnUndead:     soundOnlyEffectSpec("effect\\ef_bash.wav"),
	effectSanctuary:      strEffectSpecAttached("sanctuary", "effect\\priest_sanctuary.wav", false),
	effectImpositio:      strEffectSpecAttached("impositio", "effect\\priest_impositio.wav", false),
	effectLexAeterna:     strEffectSpecAttachedMin("lexaeterna", "lexaeterna_min", "effect\\priest_lexaeterna.wav", false),
	effectAspersio:       strEffectSpecAttached("aspersio", "effect\\priest_aspersio.wav", false),
	effectLexDivina:      strEffectSpecAttached("lexdivina", "effect\\priest_lexdivina.wav", false),
	effectSuffragium:     strEffectSpecAttachedMin("suffragium", "suffragium_min", "effect\\priest_suffragium.wav", false),
	effectStormGust:      strEffectSpecAttachedMin("stormgust", "storm_min", "effect\\wizard_stormgust.wav", false),
	effectLordVermilion:  strEffectSpecAttached("lord", "effect\\wizard_fire_ivy.wav", false),
	effectBenedictio:     strEffectSpecAttached("benedictio", "effect\\priest_benedictio.wav", false),
	effectMeteorStorm: {
		CameraShake:      650 * time.Millisecond,
		CameraShakeDelay: 600 * time.Millisecond,
		SFX:              []string{"effect\\wizard_meteor.wav"},
		SFXRandMin:       1,
		SFXRandMax:       4,
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "meteor%d",
			STRRandMin:     1,
			STRRandMax:     4,
			AttachedEntity: true,
		}},
	},
	effectJupitelThunder: jupitelThunderEffectSpec(),
	effectJupitelHit:     jupitelHitEffectSpec(),
	effectQuagmire:       strEffectSpec("quagmire", "effect\\wizard_quagmire.wav"),
	effectFirePillar:     strEffectSpec("firepillar", "effect\\wizard_fire_pillar_a.wav"),
	effectFirePillarBomb: strEffectSpec("firepillarbomb", "effect\\wizard_fire_pillar_b.wav"),
	effectWhitePulse:     whitePulseEffectSpec(),
	effectAdrenalineCast: soundOnlyEffectSpec("effect\\black_adrenalinerush_a.wav"),
	effectHasteUp:        soundOnlyEffectSpec("effect\\black_adrenalinerush_b.wav"),
	effectFlasher:        soundOnlyEffectSpec("effect\\hunter_flasher.wav"),
	effectRemoveTrap:     soundOnlyEffectSpec("effect\\hunter_removetrap.wav"),
	effectRepairWeapon:   repairWeaponEffectSpec(),
	effectCrashEarth:     crashEarthEffectSpec(),
	effectBlastMine:      soundOnlyEffectSpec("effect\\hun_anklesnare.wav"),
	effectBlastMineBomb:  strEffectSpec("blastmine", "effect\\hunter_blastmine.wav"),
	effectClaymore:       strEffectSpec("claymore", "effect\\hunter_claymoretrap.wav"),
	effectFreezingTrap:   strEffectSpec("freezing", "effect\\hunter_freezingtrap.wav"),
	effectGasPush:        strEffectSpec("gaspush", ""),
	effectSpringTrap:     strEffectSpec("spring", "effect\\hunter_springtrap.wav"),
	effectWeaponPerfect:  strEffectSpecAttachedMin("weaponperfection", "weaponperfection_min", "effect\\black_weapon_perfection.wav", false),
	effectMaximizePower:  strEffectSpecAttachedMin("maximizepower", "maximize_min", "", false),
	effectMaximizeSounds: delayedSoundEffectSpecDuration(950*time.Millisecond,
		[]string{
			"effect\\black_maximize_power_circle.wav",
			"effect\\black_maximize_power_sword.wav",
			"effect\\black_maximize_power_sword.wav",
			"effect\\black_maximize_power_sword_bic.wav",
		},
		[]time.Duration{time.Millisecond, 550 * time.Millisecond, 700 * time.Millisecond, 950 * time.Millisecond},
	),
	effectGreedSound: soundOnlyEffectSpec("effect\\ef_entry.wav"),

	effectGunslingerCoinSound: soundOnlyEffectSpec("effect\\플립.wav"),

	effectKyrie:          strEffectSpecAttachedMin("kyrie", "kyrie_min", "effect\\priest_kyrie_eleison_a.wav", false),
	effectMagnus:         strEffectSpec("magnus", "effect\\priest_magnus.wav"),
	effectBlitzBeat:      soundOnlyEffectSpec("effect\\hunter_blitzbeat.wav"),
	effectWaterBall:      waterBallEffectSpec(),
	effectWaterBall2:     waterBall2EffectSpec(),
	effectDetecting:      soundOnlyEffectSpec("effect\\hunter_detecting.wav"),
	effectCloaking:       soundOnlyEffectSpec("effect\\assasin_cloaking.wav"),
	effectSonicBlow:      sonicBlowEffectSpec(),
	effectSonicBlowHit:   sonicBlowHitEffectSpec(),
	effectGrimtooth:      soundOnlyEffectSpec("effect\\ef_frostdiver.wav"),
	effectVenomDust:      strEffectSpec("venomdust", "effect\\assasin_poisonreact.wav"),
	effectPoisonReact:    strEffectSpecAttached("poisonreact_1st", "effect\\assasin_poisonreact.wav", false),
	effectPoisonReact2:   strEffectSpecAttached("poisonreact", "effect\\assasin_poisonreact.wav", false),
	effectOverthrust:     soundOnlyEffectSpec("effect\\black_overthrust.wav"),
	effectVenomSplasher:  strEffectSpecAttached("venomsplasher", "effect\\assasin_venomsplasher.wav", false),
	effectTwoHandQuicken: strEffectSpecAttached("twohand", "effect\\knight_twohandquicken.wav", true),
	effectAutoCounter:    strEffectSpecAttached("autocounter", "effect\\knight_autocounter.wav", false),
	effectGrimtoothAtk:   grimtoothAttackEffectSpec(),
	effectFreeze:         strEffectSpecAttached("freeze", "", false),
	effectFreezed:        strEffectSpecAttached("freezed", "", false),
	effectIceCrash:       strEffectSpecAttached("icecrash", "", false),
	effectSlowPoison:     strEffectSpec("slowp", "effect\\priest_slowpoison.wav"),
	effectFirePillarOn:   firePillarOnEffectSpec(),
	effectSandman:        strEffectSpec("sandman", "effect\\hunter_sandman.wav"),
	effectRevive:         soundOnlyEffectSpec("effect\\priest_resurrection.wav"),
	effectHeavenDrive:    heavenDriveEffectSpec(),
	effectSonicBlow2:     strEffectSpecAttached("sonicblow", "", false),
	effectBrandishSpear2: strEffectSpecAttached("brandish2", "effect\\knight_brandish_spear.wav", false),
	effectShockwave: {
		SFX: []string{"effect\\hunter_shockwavetrap.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "shockwave",
			AttachedEntity: true,
		}},
	},
	effectShockwaveHit:     strEffectSpecAttached("shockwavehit", "", false),
	effectEarthHit:         strEffectSpecAttached("earthhit", "", false),
	effectPierceSelf:       strEffectSpecAttached("pierce", "", false),
	effectBowlingSelf:      strEffectSpecAttached("bowling", "_enemy_hit_normal1.wav", true),
	effectSpearStabSelf:    strEffectSpecAttached("spearstab", "_enemy_hit_normal1.wav", false),
	effectSpearBmrSelf:     strEffectSpecAttached("spearboomerang", "effect\\knight_spear_boomerang.wav", true),
	effectSpearProjectile:  spearProjectileEffectSpec(),
	effectSpearHitSound:    soundOnlyEffectSpec("_hit_spear.wav"),
	effectShieldProjectile: shieldProjectileEffectSpec(),
	effectEnemyHitNormal1:  soundOnlyEffectSpec("_enemy_hit_normal1.wav"),
	effectHolyLight:        strEffectSpecAttached("holyhit", "", false),
	effectConcentration:    strEffectSpecAttached("concentration", "effect\\ac_concentration.wav", false),
	effectRefineOK:         strEffectSpecAttached("bs_refinesuccess", "effect\\bs_refinesuccess.wav", false),
	effectRefineFail:       strEffectSpecAttached("bs_refinefailed", "effect\\bs_refinefailed.wav", false),
	effectMakeBlur:         funcEffectSpec("MakeBlur", 2*time.Second, false),
	effectEnergyCoat:       strEffectSpecAttached("energycoat", "", false),
	effectVenomDust2:       venomDust2EffectSpec(),
	effectMentalBreak:      strEffectSpecAttached("mentalbreak", "", false),
	effectMagicalAtkHit:    strEffectSpecAttached("magical", "", false),
	effectSuiExplosion:     suiExplosionEffectSpec(),
	effectSuicide:          strEffectSpecAttached("suicide", "", false),
	effectComboAttack1:     strEffectSpecAttached("yunta_1", "", false),
	effectComboAttack2:     strEffectSpecAttached("yunta_2", "", false),
	effectComboAttack3:     strEffectSpecAttached("yunta_3", "", false),
	effectComboAttack4:     strEffectSpecAttached("yunta_4", "", false),
	effectComboAttack5:     strEffectSpecAttached("yunta_5", "", false),
	effectGuidedAttack:     strEffectSpecAttached("homing", "", false),
	effectPoisonAttack2:    strEffectSpecAttached("poison", "", false),
	effectSilenceAttack:    strEffectSpecAttached("silence", "", false),
	effectStunAttack:       strEffectSpecAttached("stun", "", false),
	effectPetrifyAttack:    strEffectSpecAttached("stonecurse", "", false),
	effectSleepAttack:      strEffectSpecAttached("sleep", "", false),
	effectPong:             strEffectSpecRandom("pong%d", "", 1, 3),
	effectLevel99:          level99EffectSpec(),
	effectLevel99Ground:    level99GroundEffectSpec(),
	effectLevel99Bubble:    level99BubbleEffectSpec(),
	effectGumgang:          gumgangEffectSpec(),
	effectFirstAid: {
		Duration: time.Second,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{{
			Kind:          EffectComponent2D,
			TextureFile:   "effect/pikapika2.bmp",
			Duration:      time.Second,
			AlphaMax:      0.2,
			BlendAdditive: true,
			FadeOut:       true,
			PosZ:          2,
			SizeStart:     100 * EffectPixelRatio,
			SizeEnd:       100 * EffectPixelRatio,
		}},
	},
	effectTeleportation: {
		Duration:         1500 * time.Millisecond,
		DetachLocalActor: true,
		SFX:              []string{"effect\\ef_teleportation.wav"},
		Components: []EffectComponent{
			teleportCylinderComponent(0.3, 0.3, 35),
			teleportCylinderComponent(0.6, 0.8, 25),
			teleportCylinderComponent(0.8, 1.0, 13),
			teleportCylinderComponent(1.0, 1.3, 5),
		},
	},
	effectReadyPortal: {
		Duration: 500 * time.Millisecond,
		SFX:      []string{"effect\\ef_readyportal.wav"},
		Components: []EffectComponent{
			readyPortalCylinderComponent(),
		},
	},
	effectPortal: {
		Duration: 25000 * time.Millisecond,
		SFX:      []string{"effect\\ef_readyportal.wav", "effect\\ef_portal.wav"},
		Components: []EffectComponent{
			readyPortalCylinderComponent(),
			portalCylinderComponent(0.6, 0.6, 15, 0, "ring_blue", 0.3),
			portalCylinderComponent(0.8, 0.8, 13, 0, "ring_blue", 0.3),
			{
				Kind:             EffectComponentCylinder,
				Color:            color.RGBA{R: 153, G: 153, B: 255, A: 255},
				TextureName:      "alpha1",
				Duration:         25000 * time.Millisecond,
				AlphaMax:         0.5,
				Fade:             true,
				Rotate:           true,
				Animation:        0,
				BottomSize:       1,
				TopSize:          1,
				Height:           1,
				PosZ:             2,
				TotalCircleSides: 20,
				CircleSides:      10,
				BlendAdditive:    true,
			},
		},
	},
	effectPharmacyOK:   strEffectSpecAttached("p_success", "effect\\p_success.wav", false),
	effectPharmacyFail: strEffectSpecAttached("p_failed", "effect\\p_failed.wav", false),
	effectHeal: {
		Duration: 1840 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			healCylinderComponent(0.95, 0.95, 8),
			healCylinderComponent(1.0, 1.0, 8),
			healParticleComponent(0.6, 1300*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 15, 1.5, 1.5, 0, 0, 0, 2, 6, false),
			healParticleComponent(0.6, 1100*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 7, 1, 1, 1, 0, 5, 0, 0, false),
		},
	},
	effectHealOffensive: {
		Duration: 1490 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{
			healOffensiveCylinderComponent(0.95, 0.95, 10),
			healOffensiveCylinderComponent(1.0, 1.0, 9),
			healParticleComponent(0.8, 1000*time.Millisecond, 400*time.Millisecond, 10*time.Millisecond, 10, 1.5, 1.5, 0, 0, 0, 3, 6, true),
			healParticleComponent(0.8, 900*time.Millisecond, 200*time.Millisecond, 50*time.Millisecond, 5, 1, 1, 1, 0, 6, 0, 0, true),
		},
	},
	effectBaseLevelUp: {
		Duration: 1300 * time.Millisecond,
		SFX:      []string{"levelup.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "angel",
			AttachedEntity: true,
		}},
	},
	effectJobLevelUp: {
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			STRFile:        "joblvup",
			AttachedEntity: true,
		}},
	},
	effectPotionRed:    healingPotionEffectSpec("빨간포션", color.RGBA{R: 255, G: 82, B: 70, A: 255}),
	effectPotionOrange: healingPotionEffectSpec("주홍포션", color.RGBA{R: 255, G: 145, B: 58, A: 255}),
	effectPotionYellow: healingPotionEffectSpec("노란포션", color.RGBA{R: 255, G: 226, B: 76, A: 255}),
	effectPotionWhite:  healingPotionEffectSpec("하얀포션", color.RGBA{R: 245, G: 245, B: 255, A: 255}),
	effectPotionBlue:   bluePotionEffectSpec(),
	effectPotionGreen:  potionEffectSpec("초록포션", color.RGBA{R: 78, G: 225, B: 98, A: 255}),
	effectDarkBreath:   {Components: []EffectComponent{{Kind: EffectComponentSPR, SpriteFile: "darkbreath", SpriteHead: true, AttachedEntity: true}}},
	effectDefender:     strEffectSpecAttached("deffender", "", false),
	effectKeeping:      strEffectSpecAttached("keeping", "", false),
	effectBloodDrain:   drainEffectSpec(color.RGBA{R: 255, G: 102, B: 102, A: 255}, false),
	effectEnergyDrain:  drainEffectSpec(color.RGBA{R: 102, G: 102, B: 255, A: 255}, true),
	effectItemFast:     strEffectSpecAttached("집중", "effect\\ac_concentration.wav", false),
	effectItemFast2:    strEffectSpecAttached("각성", "effect\\ac_concentration.wav", false),
	effectItemFast3:    berserkPotionEffectSpec(),
	effectCrusaderDef:  defenderCylinderEffectSpec("ring_black"),
	effectGrandCross:   grandCrossEffectSpec(),
	effectIntimidate:   soundOnlyEffectSpec("effect\\rog_intimidate.wav"),
	effectChookgi:      spiritSphereEffectSpec(),
	effectLineLink: {
		Duration: 100 * time.Millisecond,
		Components: []EffectComponent{{
			Kind:             EffectComponent3D,
			TextureFile:      "effect/alpha_center.tga",
			Duration:         100 * time.Millisecond,
			AlphaMax:         0.5,
			Color:            color.RGBA{R: 26, G: 26, B: 230, A: 255},
			AttachedEntity:   true,
			BlendMode:        2,
			BlendAdditive:    true,
			FadeIn:           true,
			FadeOut:          true,
			FromSrc:          true,
			RotateToTarget:   true,
			RotateWithCamera: true,
			SizeStartX:       effectTableSize(5),
			SizeStartY:       effectTableSize(50),
			SizeEndX:         effectTableSize(5),
			SizeEndY:         effectTableSize(50),
			PosZ:             1,
			AngleStart:       180,
			Overlay:          true,
		}},
	},
	effectSpellBreaker:  strEffectSpecAttached("spell", "effect\\sage_spell breake.wav", false),
	effectDispell:       strEffectSpecAttached("디스펠", "", false),
	effectBottomVolcano: propertyGroundEffectSpec("PropertyGround", "ring_red"),
	effectBottomDeluge:  propertyGroundEffectSpec("PropertyGround", "ring_blue"),
	effectBottomViolent: propertyGroundEffectSpec("PropertyGround", "ring_yellow"),
	effectBottomLand:    landProtectorGroundEffectSpec(),
	effectMagicRod:      strEffectSpecAttached("매직로드", "effect\\sage_magic rod.wav", false),
	effectHolyCross:     strEffectSpecAttached("holy_cross", "effect\\cru_holy cross.wav", false),
	effectShieldCharge:  strEffectSpecAttached("shield_charge", "", false),
	effectProvidence:    strEffectSpecAttached("providence", "", false),
	effectShieldBoomer:  soundOnlyEffectSpec("effect\\cru_shield boomerang.wav"),
	effectSpearQuicken:  strEffectSpecAttached("twohand", "effect\\knight_twohandquicken.wav", true),
	effectDevotion:      strEffectSpecAttached("devotion", "", false),
	effectReflectShield: defenderCylinderEffectSpec("ring_yellow"),
	effectAbsorbSpirits: absorbSpiritsEffectSpec(),
	effectSteelBody:     soundOnlyEffectSpec("effect\\mon_금강불괴.wav"),
	effectFlameLauncher: strEffectSpecAttached("enc_fire", "_enemy_hit_wind1.wav", false),
	effectFrostWeapon:   strEffectSpecAttached("enc_ice", "_enemy_hit_wind1.wav", false),
	effectLightningLoad: strEffectSpecAttached("enc_wind", "effect\\_enemy_hit_wind1.wav", false),
	effectSeismicWeapon: strEffectSpecAttached("enc_earth", "_enemy_hit_wind1.wav", false),
	effectGumgang2:      gumgangRingEffectSpec(1500*time.Millisecond, 0.5, 2, 5, "effect\\mon_폭기.wav"),
	effectTeiHit1:       teiHitEffectSpec("effect/alpha_center.tga", "effect\\mon_폭기.wav", 12, 250*time.Millisecond, color.RGBA{}),
	effectGumgang3:      gumgangRingEffectSpec(1000*time.Millisecond, 0.3, 3, 6, ""),
	effectTanji:         tanjiEffectSpec(),
	effectTeiHit1X:      teiHitEffectSpec("effect/lens1.tga", "effect\\mon_아수라 패황권.wav", 24, 100*time.Millisecond, color.RGBA{}),
	effectChimto:        soundOnlyEffectSpec("effect\\mon_침투경.wav"),
	effectStealCoin:     strEffectSpecAttached("steal_coin", "", false),
	effectStripWeapon:   strEffectSpecAttached("strip_weapon", "effect\\t_벗김.wav", false),
	effectStripShield:   strEffectSpecAttached("strip_shield", "effect\\t_벗김.wav", false),
	effectStripArmor:    strEffectSpecAttached("strip_armor", "effect\\t_벗김.wav", false),
	effectStripHelm:     strEffectSpecAttached("strip_helm", "effect\\t_벗김.wav", false),
	effectChainCombo:    strEffectSpecAttached("연환", "effect\\mon_연환.wav", false),
	effectRogueCoin:     rogueCoinEffectSpec(),
	effectBackStab:      soundOnlyEffectSpec("effect\\rog_back stap.wav"),
	effectTeiHit3:       teiHitEffectSpec("effect/lens1.tga", "", 20, 100*time.Millisecond, color.RGBA{R: 26, G: 26, B: 255, A: 255}),
	effectBottomLullaby: soundOnlyEffectSpec("effect\\자장가.wav"),
	effectBottomRichKim: soundOnlyEffectSpec("effect\\김서방돈.wav"),
	effectBottomChaos:   soundOnlyEffectSpec("effect\\영원의 혼돈.wav"),
	effectBottomDrum:    soundOnlyEffectSpec("effect\\전장의.wav"),
	effectBottomNibelung: soundOnlyEffectSpec(
		"effect\\니벨룽겐의 반지.wav",
	),
	effectBottomRoki:             soundOnlyEffectSpec("effect\\로키.wav"),
	effectBottomAbyss:            soundOnlyEffectSpec("effect\\심연속으로.wav"),
	effectBottomSieg:             soundOnlyEffectSpec("effect\\불사신.wav"),
	effectBottomWhistle:          soundOnlyEffectSpec("effect\\달빛세레나데.wav"),
	effectBottomSinX:             soundOnlyEffectSpec("effect\\석양의 어쌔신.wav"),
	effectBottomBragi:            soundOnlyEffectSpec("effect\\브라기의 시.wav"),
	effectBottomApple:            soundOnlyEffectSpec("effect\\이둔의 사과.wav"),
	effectBottomHumming:          soundOnlyEffectSpec("effect\\흥얼거림.wav"),
	effectBottomForget:           soundOnlyEffectSpec("effect\\나를잊지말아요.wav"),
	effectBottomFortune:          soundOnlyEffectSpec("effect\\행운의.wav"),
	effectBottomService:          soundOnlyEffectSpec("effect\\당신을 위한 서비스.wav"),
	effectBottomDissonanceGround: songGroundEffectSpec("effect/lens_w.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 0.5),
	effectBottomLullabyGround:    songGroundEffectSpec("effect/zz.bmp", color.RGBA{R: 237, G: 158, B: 255, A: 13}, 0.5),
	effectBottomRichKimGround:    songGroundEffectSpec("effect/pocket.bmp", color.RGBA{R: 252, G: 199, B: 199, A: 13}, 0.5),
	effectBottomChaosGround:      songGroundEffectSpec("effect/lens_g.bmp", color.RGBA{R: 128, G: 255, B: 194, A: 13}, 0.5),
	effectBottomDrumGround:       songGroundEffectSpec("effect/melody_b.bmp", color.RGBA{R: 237, G: 101, B: 252, A: 13}, 0.5),
	effectBottomNibelungGround:   songGroundEffectSpec("effect/twirl.bmp", color.RGBA{R: 28, G: 236, B: 255, A: 13}, 0.5),
	effectBottomRokiGround:       songGroundEffectSpec("effect/safeline.bmp", color.RGBA{R: 220, G: 101, B: 252, A: 13}, 0.5),
	effectBottomAbyssGround:      songGroundEffectSpec("effect/bluegemstone.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 1),
	effectBottomSiegGround:       songGroundEffectSpec("effect/lens_b.bmp", color.RGBA{R: 72, G: 59, B: 255, A: 13}, 0.5),
	effectBottomWhistleGround:    songGroundEffectSpec("effect/melody_b.bmp", color.RGBA{R: 255, G: 192, B: 203, A: 13}, 0.5),
	effectBottomSinXGround:       songGroundEffectSpec("effect/lens_r.bmp", color.RGBA{R: 255, G: 204, B: 217, A: 102}, 0.5),
	effectBottomBragiGround:      songGroundEffectSpec("effect/spell_01.bmp", color.RGBA{}, 0.5),
	effectBottomAppleGround:      songGroundEffectSpec("effect/idun_apple.bmp", color.RGBA{R: 255, G: 255, B: 0, A: 13}, 1),
	effectBottomUglyDanceGround:  songGroundEffectSpec("effect/lens_w.bmp", color.RGBA{R: 255, G: 255, B: 255, A: 13}, 0.5),
	effectBottomHummingGround:    songGroundEffectSpec("effect/melody_a.bmp", color.RGBA{R: 230, G: 209, B: 209, A: 13}, 0.5),
	effectBottomForgetGround:     songGroundEffectSpec("effect/lens_g.bmp", color.RGBA{R: 28, G: 255, B: 115, A: 13}, 0.5),
	effectBottomFortuneGround:    songGroundEffectSpec("effect/kiss.bmp", color.RGBA{R: 252, G: 111, B: 101, A: 13}, 2.5),
	effectBottomServiceGround:    songGroundEffectSpec("effect/safeline.bmp", color.RGBA{R: 255, G: 128, B: 183, A: 13}, 0.5),
	effectTalkFrostJoke:          funcEffectSpec("FrostJokeTalk", 500*time.Millisecond, true),
	effectTalkScream:             funcEffectSpec("ScreamTalk", 500*time.Millisecond, true),
	effectThrowItem:              throwItemEffectSpec("유저인터페이스/item/염산병.bmp", 30),
	effectChemicalProt:           soundOnlyEffectSpec("apocalips_attack.wav"),
	effectDemonstration: {
		Components: []EffectComponent{{
			Kind:           EffectComponentSPR,
			SpriteFile:     "데몬스트레이션",
			AttachedEntity: false,
		}},
	},
	effectChemical2:    chemical2EffectSpec(),
	effectHeal2:        heal2EffectSpec(),
	effectExit2:        exit2EffectSpec(),
	effectBottomMagnus: bottomSquareEffectSpec("ring_red", color.RGBA{}, 0.2, 0.7, 0.7, 5, false),
	effectBottomSanc:   bottomSquareEffectSpec("magic_green", color.RGBA{R: 128, G: 230, B: 128, A: 255}, 0.3, 0.7, 0.7, 2, false),
	effectWarpZone2:    warpZone2EffectSpec(),
	effectHeal4:        heal4EffectSpec(),
	effectBeginAsura:   beginAsuraEffectSpec(),
	effectTripleAttack: tripleAttackEffectSpec(),
	effectHPTime:       naturalRecoveryEffectSpec(color.RGBA{R: 230, G: 255, B: 230, A: 255}, "_heal_effect.wav"),
	effectSPTime:       naturalRecoveryEffectSpec(color.RGBA{R: 230, G: 230, B: 255, A: 255}, "effect\\흡기.wav"),
	effectBlind: {
		Duration: 500 * time.Millisecond,
		SFX:      []string{"_blind.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentFUNC,
			FuncName:       "Blind",
			AttachedEntity: false,
		}},
	},
	effectPoisonStatus: funcEffectSpec("Poison", 500*time.Millisecond, false),
	effectGuard:        guardEffectSpec(),
	effectJobLvUp50:    strEffectSpecAttached("joblvup", "", false),
	effectMagnum2:      magnum2EffectSpec(),
	effectEntry2:       entry2EffectSpec(),
	effectColorPaper:   soundOnlyEffectSpec("effect\\wedding.wav"),
	effectSoulBreaker:  soulBreakerEffectSpec(),
	effectLevel99Aura1: level99BubbleEffectSpec(),
	effectFoodChocolate: sprEffectSpec(
		"vallentine",
		"effect\\vallentine.wav",
		true,
		false,
	),
	effectPressure:          pressureEffectSpec(),
	effectBash3D:            bash3DEffectSpec("Bash3D", "effect\\bash3d.wav", 500*time.Millisecond, 200*time.Millisecond, 5),
	effectAuraBlade:         soundOnlyEffectSpec("effect\\오라 블레이드.wav"),
	effectRedBody:           soundOnlyEffectSpec("effect\\버서크.wav"),
	effectSpiralBeforeCast:  spiralBeforeCastEffectSpec(),
	effectQuake:             quakeEffectSpec(),
	effectLKConcentration:   strEffectSpecAttached("twohand", "effect\\knight_twohandquicken.wav", true),
	effectBottomGospel:      soundOnlyEffectSpec("effect\\가스펠.wav"),
	effectGospelGround:      gospelGroundEffectSpec(),
	effectFogWallGround:     fogWallGroundEffectSpec(),
	effectDeath:             strEffectSpecAttached("devil", "", false),
	effectBottomBasilica:    basilicaEffectSpec(),
	effectHitLine2:          soundOnlyEffectSpec("effect\\맹호경파산.wav"),
	effectBash3D2:           bash3DEffectSpec("Bash3D2", "effect\\mon_폭기.wav", 400*time.Millisecond, 50*time.Millisecond, 8),
	effectEnergyDrain2:      energyDrainProjectileEffectSpec(color.RGBA{R: 204, G: 204, B: 255, A: 255}, 160, 190),
	effectTransBlueBody:     transBlueBodyEffectSpec(),
	effectMagicCrasher:      magicCrasherEffectSpec(),
	effectMagicPower:        soundOnlyEffectSpec("effect\\마법력 증폭.wav"),
	effectLightBlade:        sprEffectSpec("한복천사", "", true, true),
	effectEnergyDrain3:      energyDrainProjectileEffectSpec(color.RGBA{R: 178, G: 255, B: 178, A: 255}, 140, 170),
	effectLineLink2:         soundOnlyEffectSpec("effect\\소울 체인지.wav"),
	effectTrueSight:         soundOnlyEffectSpec("effect\\hunter_detecting.wav"),
	effectFalconAssault:     falconAssaultEffectSpec(),
	effectSharpShootingCast: sharpShootingCastEffectSpec(),
	effectTripleAttack2:     soundOnlyEffectSpec("effect\\샤프슈팅.wav"),
	effectPortal4:           soundOnlyEffectSpec("effect\\윈드워크.wav"),
	effectMeltdown:          strEffectSpecAttached("melt", "", false),
	effectCartBoost:         strEffectSpecAttached("cart", "effect\\ef_incagility.wav", false),
	effectRejectSword:       strEffectSpecAttached("sword", "effect\\kyrie_guard.wav", false),
	effectTripleAttack3:     soundOnlyEffectSpec("effect\\애로우 발칸.wav"),
	effectMoonlit:           moonlitEffectSpec(),
	effectLevel99AuraMid:    level99EffectSpec(),
	effectLevel99AuraBottom: level99GroundEffectSpec(),
	effectBash3D3:           bash3DEffectSpec("Bash3D3", "effect\\헤드 크러쉬.wav", 675*time.Millisecond, 500*time.Millisecond, 6),
	effectBash3D4:           bash3DEffectSpec("Bash3D4", "effect\\비트 조인트.wav", 675*time.Millisecond, 500*time.Millisecond, 6),
	effectPortal5:           portal5EffectSpec(),
	effectMagicCrasher2:     magicCrasher2EffectSpec(),
	effectBottomSpider:      spiderWebEffectSpec(),
	effectSoulBurn:          strEffectSpecAttached("소울번", "", false),
	effectSoulChange:        strEffectSpecAttached("사랑효과", "", false),
	effectSoulBreaker2:      soulBreaker2EffectSpec(),
	effectBabyBody:          bodySizeEffectSpec("EffectSmallTransition", 300*time.Millisecond, 2.5),
	effectBabyBody2:         bodySizeEffectSpec("EffectSmall", 5*time.Minute, 2.5),
	effectGiantBody:         bodySizeEffectSpec("EffectBigTransition", 300*time.Millisecond, 7.5),
	effectGiantBody2:        bodySizeEffectSpec("EffectBig", 5*time.Minute, 7.5),
	effectQuakeBody:         soundOnlyEffectSpec("effect\\복호격.wav"),
	effectAssumptio2:        strEffectSpecAttached("asum", "effect\\아숨프티오.wav", false),
	effectStopEffect:        soundOnlyEffectSpec("effect\\t_효과음1.wav"),
	effectJumpBody:          soundOnlyEffectSpec("effect\\t_회피2.wav"),
	effectDarkGrandCross:    {},
	effectDarkSoulStrike:    darkSoulStrikeEffectSpec(),
	effectDarkJupitelHit:    darkJupitelHitEffectSpec(),
	effectNPCStop:           sprEffectSpec("스톱", "", true, false),
	effectDarkCasting:       darkCastingEffectSpec(),
	effectNPCPowerUp:        soundOnlyEffectSpec("effect\\mon_폭기.wav"),
	effectJumpKick:          soundOnlyEffectSpec("effect\\t_날라차기.wav"),
	effectBeginAsura1:       mildWindEffectSpec("effect/hanmoon1.tga"),
	effectBeginAsura2:       mildWindEffectSpec("effect/hanmoon2.tga"),
	effectBeginAsura3:       mildWindEffectSpec("effect/hanmoon3.tga"),
	effectBeginAsura4:       mildWindEffectSpec("effect/hanmoon4.tga"),
	effectBeginAsura5:       mildWindEffectSpec("effect/hanmoon7.tga"),
	effectBeginAsura6:       mildWindEffectSpec("effect/hanmoon5.tga"),
	effectBeginAsura7:       mildWindEffectSpec("effect/hanmoon6.tga"),
	effectMochi:             strEffectSpecAttached("찹쌀떡", "", false),
	effectRamadan:           strEffectSpecAttached("ramadan", "", false),
	effectEDP:               soundOnlyEffectSpec("effect\\assasin_cloaking.wav"),
	effectPreserve:          soundOnlyEffectSpec("effect\\black_maximize_power_sword_bic.wav"),
	effectCastSpin:          funcEffectSpec("CastSpin", 500*time.Millisecond, true),
	effectChookgi2:          spiritSphereEffectSpec(),
	effectMapae:             strEffectSpecAttached("mapae", "effect\\mapae.wav", false),
	effectItemPokJuk:        strEffectSpecAttached("itempokjuk", "effect\\itempokjuk.wav", false),
	effectValentine05:       sprEffectSpec("05vallentine", "", true, false),
	effectBeginAsura11:      beginAsura11EffectSpec(),
	effectChemical2Dash:     chemical2EffectSpec(),
	effectBottomHermode:     {},
	effectHermodeMusic:      soundOnlyEffectSpec("effect\\헤르모드의 지팡이"),
	effectItemFastDown:      sprEffectSpec("fast", "effect\\fast.wav", true, false),
	effectTarotCard1:        tarotCardEffectSpec(1),
	effectTarotCard2:        tarotCardEffectSpec(2),
	effectTarotCard3:        tarotCardEffectSpec(3),
	effectTarotCard4:        tarotCardEffectSpec(4),
	effectTarotCard5:        tarotCardEffectSpec(5),
	effectTarotCard6:        tarotCardEffectSpec(6),
	effectTarotCard7:        tarotCardEffectSpec(7),
	effectTarotCard8:        tarotCardEffectSpec(8),
	effectTarotCard9:        tarotCardEffectSpec(9),
	effectTarotCard10:       tarotCardEffectSpec(10),
	effectTarotCard11:       tarotCardEffectSpec(11),
	effectTarotCard12:       tarotCardEffectSpec(12),
	effectTarotCard13:       tarotCardEffectSpec(13),
	effectTarotCard14:       tarotCardEffectSpec(14),
	effectAcidDemon:         acidDemonEffectSpec(),
	effectHated:             soundOnlyEffectSpec("effect\\t_보조마법.wav"),
	effectStin:              soundOnlyEffectSpec("effect\\t_에너지방출.wav"),
	effectStin2:             repeatedSoundEffectSpec("effect\\t_날라차기.wav", 5, 200*time.Millisecond),
	effectStin3:             soundOnlyEffectSpec("effect\\t_에너지방출.wav"),
	effectScreenQuake:       screenQuakeEffectSpec(),
	effectHfliMoon1:         strEffectSpecAttached("moonlight_1", "effect\\h_moonlight_1.wav", false),
	effectHfliMoon2:         strEffectSpecAttached("moonlight_2", "effect\\h_moonlight_2.wav", false),
	effectHfliMoon3:         strEffectSpecAttached("moonlight_3", "effect\\h_moonlight_3.wav", false),
	effectHoUp:              strEffectSpecAttached("h_levelup", "", false),
	effectHamiDefence:       strEffectSpecAttached("defense", "", false),
	effectHamiCastle:        sprEffectSpec("캐슬링", "", true, false),
	effectHamiBlood:         sprEffectSpec("블러드러스트", "", true, false),
	effectItemThunder:       sprEffectSpec("item_thunder", "", true, false),
	effectItemCloud:         sprEffectSpec("item_cloud", "", true, false),
	effectItemCurse:         sprEffectSpec("item_curse", "", true, false),
	effectItemZZZ:           sprEffectSpec("item_zzz", "_snore.wav", true, false),
	effectItemRain:          sprEffectSpec("item_rain", "", true, false),
	effectM01:               sprEffectSpec("m_ef01", "", true, false),
	effectM02:               sprDirectionEffectSpec("m_ef02", ""),
	effectM03:               sprEffectSpec("m_ef03", "", true, false),
	effectM04:               sprEffectSpec("m_ef04", "", true, false),
	effectM05:               sprEffectSpec("m_ef05", "dragon_breath.wav", true, false),
	effectM06:               sprEffectSpec("m_ef06", "", true, false),
	effectM07:               sprEffectSpec("m_ef07", "effect\\t_보조마법.wav", true, false),
	effectKaizel:            soundOnlyEffectSpec("effect\\priest_resurrection.wav"),
	effectStatFoodSTR:       strEffectSpecAttached("food_str", "_heal_effect.wav", false),
	effectStatFoodINT:       strEffectSpecAttached("food_int", "_heal_effect.wav", false),
	effectStatFoodVIT:       strEffectSpecAttached("food_vit", "_heal_effect.wav", false),
	effectStatFoodAGI:       strEffectSpecAttached("food_agi", "_heal_effect.wav", false),
	effectStatFoodDEX:       strEffectSpecAttached("food_dex", "_heal_effect.wav", false),
	effectStatFoodLUK:       strEffectSpecAttached("food_luk", "_heal_effect.wav", false),
	effectThrowItem6:        throwItemEffectSpecFull("유저인터페이스/item/베넘나이프.bmp", 30, 200*time.Millisecond, 1),
	effectFireHit2:          strEffectSpecRandomAttached("firehit%d", "", 1, 3, true, false),
	effectNPCStop2:          sprStopAtEndEffectSpec("cconfine", "effect\\ef_hit6.wav", true),
	effectFVoice:            sprEffectSpec("fvoice", "amon_ra_die01.wav", false, false),
	effectWink:              sprEffectSpec("wink", "", false, false),
	effectCookingOK:         strEffectSpecAttached("cook_suc", "_heal_effect.wav", false),
	effectCookingFail:       strEffectSpecAttached("cook_fail", "caramel_die.wav", false),
	effectHapgyeok:          hapgyeokEffectSpec(),
	effectThrowItem7:        throwItemSoundEffectSpec("유저인터페이스/item/수리검.bmp", 30, "effect\\닌자_던지기.wav"),
	effectThrowItem8:        throwItemSoundEffectSpec("유저인터페이스/item/쿠나이_독.bmp", 30, "effect\\닌자_던지기.wav"),
	effectThrowItem9:        throwItemSoundEffectSpec("유저인터페이스/item/풍마_뇌우.bmp", 30, "effect\\닌자_던지기.wav"),
	effectThrowItem10:       throwItemSoundEffectSpec("effect/coin_a.bmp", 20, "effect\\닌자_던지기.wav"),
	effectKouenka:           strEffectSpecRandomAttached("firehit", "effect\\ef_firearrow%d.wav", 1, 3, true, false),
	effectHyousensou:        strEffectSpecRandomAttached("freeze", "effect\\ef_icearrow%d.wav", 1, 3, true, false),
	effectStin4:             soundOnlyEffectSpec("effect\\풍인.wav"),
	effectThunderStorm2:     strEffectSpecAttached("setsudan", "effect\\ef_thunderstorm.wav", false),
	effectRGCoin3:           soundOnlyEffectSpec("effect\\디스암.wav"),
	effectBash3D5:           bash3DEffectSpec("Bash3D5", "effect\\bash3d5.wav", 175*time.Millisecond, 0, 6),
	effectChookgi3:          spiritSphereEffectSpec(),
	effectKirikage:          sprEffectSpec("그림자베기", "effect\\그림자베기.wav", true, false),
	effectTatami:            sprEffectSpec("다다미 뒤집기", "effect\\다다미뒤집기.wav", true, false),
	effectKasumikiri:        sprEffectSpec("안개베기", "effect\\안개베기.wav", true, false),
	effectIssen:             sprEffectSpec("일섬", "effect\\일섬.wav", true, false),
	effectKaen:              sprRepeatEffectSpec("화염진", "effect\\화염진.wav", true),
	effectBaku:              strEffectSpec("fire dragon", "effect\\폭염룡.wav"),
	effectHyousyouraku:      strEffectSpec("icy", "effect\\빙정락.wav"),
	effectDesperado:         sprEffectSpec("데스페라도", "effect\\데스페라도.wav", true, false),
	effectLightningS:        sprEffectSpec("라이트닝스피어", "", false, false),
	effectBlindS:            sprEffectSpec("블라인드스피어", "", false, false),
	effectPoisonS:           sprEffectSpec("포이즌스피어", "", false, false),
	effectFreezingS:         sprEffectSpec("프리징스피어", "", false, false),
	effectFlareS:            sprEffectSpec("플레어스피어", "", false, false),
	effectRapidShower:       sprEffectSpec("래피드샤워", "effect\\래피드샤워.wav", true, false),
	effectMagicalBullet:     sprEffectSpec("매지컬불릿", "effect\\매지컬블릿.wav", true, false),
	effectSpreadAttack:      sprDirectionEffectSpec("스프레드", ""),
	effectTrackCasting:      strEffectSpecAttached("트랙킹", "", false),
	effectTracking:          sprEffectSpec("트래킹", "", true, false),
	effectTripleAction:      sprEffectSpec("트리플액션", "effect\\트리플액션.wav", true, false),
	effectBullseye:          strEffectSpecAttached("불스아이", "", false),
	effectNPCEarthquake:     npcEarthquakeEffectSpec(),
	effectDragonFear:        dragonFearEffectSpec(),
	effectWideBleeding:      strEffectSpecAttached("wideb", "effect\\wideb.wav", false),
	effectWideConfuse:       strEffectSpecAttached("dfear", "effect\\dragonfear.wav", false),
	effectBottomRunner:      groundTextureEffectSpec("effect/hanmoon1.tga"),
	effectBottomTransfer:    groundTextureEffectSpec("effect/hanmoon2.tga"),
	effectBottomEvilLand:    evilLandEffectSpec(),
	effectGravitation:       gravitationEffectSpec(),
	effectGuard3:            soundOnlyEffectSpec("effect\\kyrie_guard.wav"),
	effectCriticalWound:     strEffectSpecAttached("cwound", "", false),
	effectFirecracker2:      firecrackerBannerEffectSpec("폭죽_러브"),
	effectFirecracker3:      firecrackerBannerEffectSpec("폭죽_화이트데이"),
	effectFirecracker4:      firecrackerBannerEffectSpec("폭죽_발렌타인"),
	effectFirecracker5:      firecrackerBannerEffectSpec("폭죽_생일"),
	effectFirecracker6:      firecrackerBannerEffectSpec("폭죽_크리스마스"),
	effectFlowerLeaf:        strEffectSpecAttached("flower_leaf", "", false),
	effectItem315:           strEffectSpecAttached("mobile_ef02", "", false),
	effectItem316:           strEffectSpecAttached("mobile_ef01", "", false),
	effectItem317:           strEffectSpecAttached("mobile_ef03", "", false),
	effectStormMin:          strEffectSpecAttached("storm_min", "effect\\wizard_stormgust.wav", false),
	effectFirecracker7:      strEffectSpec("pokjuk_jap", ""),
	effectBottomBlue:        bottomBlueEffectSpec(),
	effectBottomBlue2:       bottomBlueEffectSpec(),
	effectFirePillarOn2:     judexEffectSpec(),
	effectForestLight5:      soundOnlyEffectSpec("effect\\ab_renovation.wav"),
	effectAdoramus:          strEffectSpecAttached("ado", "effect\\ab_adoramus.wav", false),
	effectIgnitionBreak:     strEffectSpecAttached("이그니션브레이크", "effect\\wl_jackfrost.wav", false),
	effectFrostMisty:        soundOnlyEffectSpec("effect\\t_에나지방출.wav"),
	effectCrimsonRock:       strEffectSpecAttached("crimson_r", "effect\\crimson_r.wav", false),
	effectHellInferno:       strEffectSpecAttached("hell_in", "", false),
	effectMarshOfAbyss:      sprEffectSpec("mashofa", "", false, false),
	effectDragonHowling:     strEffectSpecAttached("dragon_h", "dragon_h.wav", false),
	effectEarthWall:         earthWallEffectSpec(),
	effectChainLightning:    strEffectSpecAttached("chainlight", "effect\\chainlight.wav", false),
	effectAimedBolt:         strEffectSpecAttached("aimed", "", false),
	effectArrowStorm:        strEffectSpecAttached("arrowstorm", "", false),
	effectLaulamus:          strEffectSpecAttached("laulamus", "", false),
	effectLauagnus:          strEffectSpecAttached("lauagnus", "", false),
	effectMillenniumShield:  strEffectSpecAttached("mil_shield", "", false),
	effectConcentration2:    strEffectSpecAttached("concentration", "", false),
	effectGlassWall3:        glassWall3EffectSpec(),
	effectBerserkPotion2:    strEffectSpecAttached("버서크", "", false),
	effectRolling1:          rollingCutterCounterEffectSpec(1),
	effectRolling2:          rollingCutterCounterEffectSpec(2),
	effectRolling3:          rollingCutterCounterEffectSpec(3),
	effectRolling4:          rollingCutterCounterEffectSpec(4),
	effectRolling5:          rollingCutterCounterEffectSpec(5),
	effectRolling6:          rollingCutterCounterEffectSpec(6),
	effectRolling7:          rollingCutterCounterEffectSpec(7),
	effectRolling8:          rollingCutterCounterEffectSpec(8),
	effectRolling9:          rollingCutterCounterEffectSpec(9),
	effectRolling10:         rollingCutterCounterEffectSpec(10),
	effectCastSpin2:         funcEffectSpec("CastSpin2", 500*time.Millisecond, true),
	effectCrashAxe:          strEffectSpecAttached("powerswing", "", false),
	effectStasis:            soundOnlyEffectSpec("effect\\wl_stasis.wav"),
	effectBottomBasilica2:   bottomBasilica2EffectSpec(),
	effectRecognized:        soundOnlyEffectSpec("effect\\wl_recognizedspell.wav"),
	effectTetra:             soundOnlyEffectSpec("effect\\wl_tetravortex.wav"),
	effectTetraCasting:      funcEffectSpec("TetraCasting", 500*time.Millisecond, true),
	effectStretch:           soundOnlyEffectSpec("effect\\bodypaint.wav"),
	effectEnervation:        strEffectSpecAttached("enervation", "", false),
	effectEnervation2:       strEffectSpecAttached("groomy", "", false),
	effectEnervation3:       strEffectSpecAttached("ignorance", "", false),
	effectEnervation4:       strEffectSpecAttached("laziness", "effect\\laziness.wav", false),
	effectEnervation5:       strEffectSpecAttached("unlucky", "", false),
	effectEnervation6:       strEffectSpecAttached("weakness", "", false),
	effectBottomManhole:     soundOnlyEffectSpec("effect\\dimension.wav"),
	effectManhole:           soundOnlyEffectSpec("effect\\manhole.wav"),
	effectForestLight6:      soundOnlyEffectSpec("effect\\dimension.wav"),
	effectBottomAni:         soundOnlyEffectSpec("effect\\chaospanic.wav"),
	effectBottomMaelstrom:   soundOnlyEffectSpec("effect\\maelstrom.wav"),
	effectBottomBloodyLust:  soundOnlyEffectSpec("effect\\bloodylust.wav"),
	effectHealN:             soundOnlyEffectSpec("effect\\기공포.wav"),
	effectChookgiN:          spiritSphereEffectSpec(),
	effectDance1:            soundOnlyEffectSpec("effect\\수줍은하루의우울.wav"),
	effectBotReverb:         melodyBillboardEffectSpec("effect/melody_b.bmp", "effect\\reverberation.wav", effectTableColor(1, 0.6, 0.6)),
	effectRainParticle:      soundOnlyEffectSpec("effect\\rainstorm.wav"),
	effectChemicalV2:        soundOnlyEffectSpec("effect\\안식의자장가.wav"),
	effectBotReverb2:        melodyBillboardEffectSpec("effect/melody_a.bmp", "effect\\나락의노래.wav", effectTableColor(0.6, 0.6, 1)),
	effectCirclePower2:      soundOnlyEffectSpec("effect\\순환하는자연의소리.wav"),
	effectSecra2:            secra2EffectSpec(),
	effectSprPlant2:         soundOnlyEffectSpec("effect\\워그와함께춤을.wav"),
	effectSprPlant3:         soundOnlyEffectSpec("effect\\마나의노래.wav"),
	effectSprPlant4:         soundOnlyEffectSpec("effect\\새터데이나이트피버.wav"),
	effectSprPlant5:         soundOnlyEffectSpec("effect\\레라드의이슬.wav"),
	effectSprPlant6:         soundOnlyEffectSpec("effect\\멜로디오브싱크.wav"),
	effectSprPlant7:         soundOnlyEffectSpec("effect\\비욘드오브워크라이.wav"),
	effectSprPlant8:         soundOnlyEffectSpec("effect\\언리미티드허밍보이스.wav"),
	effectHeartAsura:        soundOnlyEffectSpec("effect\\세이렌의목소리.wav"),
	effectGlassWall4:        glassWall4EffectSpec(),
	effectBash3D6:           bash3D6EffectSpec(),
	effectElectric4:         soundOnlyEffectSpec("effect\\sr_earthshaker.wav"),
	effectTeiHit1T:          teiHit1TEffectSpec(),
	effectPressure2:         pressureDropEffectSpec("effect/shield.bmp", "effect\\lg_shieldpress.wav"),
	effectPrimeCharge2:      soundOnlyEffectSpec("effect\\lg_prestige.wav"),
	effectPrimeCharge3:      soundOnlyEffectSpec("effect\\lg_banding.wav"),
	effectPrimeCharge4:      soundOnlyEffectSpec("effect\\lg_inspiration.wav"),
	effectFireWall2:         strEffectSpecAttached("firewall_per", "", false),
	effectSprPlant10:        soundOnlyEffectSpec("effect\\s사이킥웨이브.wav"),
	effectShockwave2:        strEffectSpecAttached("hunter_shockwave_blue", "", false),
	effectColdThrow2:        soundOnlyEffectSpec("effect\\wl_jackfrost.wav"),
	effectDemonicFire4:      soundOnlyEffectSpec("effect\\s워머.wav"),
	effectPressure3:         pressureDropEffectSpec("effect/cross1.bmp", ""),
	effectPoisonMist:        strEffectSpecAttached("poison_mist", "", false),
	effectEraserCutter:      strEffectSpecAttached("eraser_cutter", "", false),
	effectLavaSlide:         strEffectSpecAttached("lava_slide", "", false),
	effectSonicClaw:         strEffectSpecAttached("sonic_claw", "", false),
	effectTinderBreaker:     strEffectSpecAttached("tinder", "", false),
	effectMidnightFrenzy:    strEffectSpecAttached("mid_frenzy", "", false),
	effectVolcanicAsh:       strEffectSpecAttached("vash00", "", false),
	effectRWC2011:           strEffectSpecAttached("rwc2011", "", false),
	effectRWC2011Two:        strEffectSpecAttached("rwc2011_2", "", false),
	effectRunMakeOK:         strEffectSpecAttached("rune_success", "", false),
	effectRunMakeFailure:    strEffectSpecAttached("rune_fail", "", false),
	effectMIResultMakeOK:    strEffectSpecAttached("changematerial_su", "", false),
	effectMIResultMakeFail:  strEffectSpecAttached("changematerial_fa", "", false),
	effectAllRayProtect:     strEffectSpecAttached("guardian", "", false),
	effectVenomFog:          strEffectSpecRandomAttached("bubble%d_1", "", 1, 4, true, false),
	effectDustStorm:         strEffectSpecAttached("dust", "", false),
	effectDanceBladeAtk:     strEffectSpecAttached("dancingblade", "", false),
	effectInvincibleOff2:    strEffectSpecAttached("invincibleoff2", "", false),
	effectDeathSummon:       strEffectSpecAttached("devil", "", false),
	effectGCDarkCrow:        strEffectSpecAttached("gc_darkcrow", "", false),
	effectAllFullThrottle:   strEffectSpecAttached("all_full_throttle", "effect\\all_full_throttle.wav", false),
	effectSRFlashCombo:      strEffectSpecAttached("sr_flashcombo", "effect\\sr_flashcombo.wav", false),
	effectRKLuxAnima:        strEffectSpecAttached("rk_luxanima", "", false),
	effectSOElemShield:      strEffectSpecAttached("so_elemental_shield", "effect\\so_elemental_shield.wav", false),
	effectABOffertorium:     strEffectSpecAttached("ab_offertorium", "effect\\ab_offertorium.wav", false),
	effectWLTelekinesis:     strEffectSpecAttached("wl_telekinesis_intense", "effect\\wl_telekinesis_intense.wav", false),
	effectGNIllusionDoping:  strEffectSpecAttached("gn_illusiondoping", "effect\\gn_illusiondoping.wav", false),
	effectNCMagmaEruption:   strEffectSpecAttached("nc_magma_eruption", "effect\\nc_magma_eruption.wav", false),
	effectNPCChill:          strEffectSpecAttached("chill", "", false),
	effectOffertoriumRing:   strEffectSpecAttached("ab_offertorium_ring", "", false),
	effectHammerOfGod:       strEffectSpecAttached("stormgust", "effect\\RL_HAMMER_OF_GOD.wav", false),
	effectAchComplete:       strEffectSpecAttachedTexturePath("ach_complete/ppring3", "ach_complete/", "", false),
	effectBodyColor:         funcEffectSpec("EffectBodyColor", 300*time.Millisecond, true),
	effectBakuretsuHadou:    sprTexturePathEffectSpec("bakuretsu_hadou/bakuretsu_hadou", "bakuretsu_hadou/", true, true, true, false, -50),
	dropEffectPink:          dropEffectSpec("pink", true),
	dropEffectYellow:        dropEffectSpec("yellow", false),
	dropEffectPurple:        dropEffectSpec("purple", false),
	effectDigitalSpace:      sprTexturePathEffectSpec("digital_space/digital_space", "digital_space/", true, true, false, true, 0),
	dropEffectBlue:          dropEffectSpec("blue", false),
	dropEffectGreen:         dropEffectSpec("green", false),
	dropEffectRed:           dropEffectSpec("red", false),
	effectNewSuccess:        strEffectSpecTexturePath("grade_enchant/new_success/new_success", "grade_enchant/new_success/", "", false, false),
	effectNewFailure:        strEffectSpecTexturePath("grade_enchant/new_failed/new_failed", "grade_enchant/new_failed/", "", false, false),
	effectNewIntro:          strEffectSpecTexturePath("grade_enchant/new_intro/new_intro", "grade_enchant/new_intro/", "", false, false),
	effectEnchantYellow:     strEffectSpecTexturePath("ui_enchant/ui_intro_yellow/ui_intro_yellow", "ui_enchant/ui_intro_yellow/", "", false, false),
	effectEnchantSuccess:    strEffectSpecTexturePath("ui_enchant/ui_enchant_success/ui_enchant_success", "ui_enchant/ui_enchant_success/", "", false, false),
	effectEnchantFail:       strEffectSpecTexturePath("ui_enchant/ui_fail/ui_enchant_fail", "ui_enchant/ui_fail/", "", false, false),
	effectEnchantBlue:       strEffectSpecTexturePath("ui_enchant/ui_intro_blue/ui_intro_blue", "ui_enchant/ui_intro_blue/", "", false, false),
	effectEnchantUpSuccess:  strEffectSpecTexturePath("ui_enchant/ui_levelup_success/ui_levelup_success", "ui_enchant/ui_levelup_success/", "", false, false),
	effectEnchantUpFail:     strEffectSpecTexturePath("ui_enchant/ui_fail/ui_levelup_fail", "ui_enchant/ui_fail/", "", false, false),
	effectEnchantGreen:      strEffectSpecTexturePath("ui_enchant/ui_intro_green/ui_intro_green", "ui_enchant/ui_intro_green/", "", false, false),
	effectEnchantResetOK:    strEffectSpecTexturePath("ui_enchant/ui_reset_success/ui_reset_success", "ui_enchant/ui_reset_success/", "", false, false),
	effectEnchantResetFail:  strEffectSpecTexturePath("ui_enchant/ui_fail/ui_reset_fail", "ui_enchant/ui_fail/", "", false, false),
	effectFood: {
		Duration: 850 * time.Millisecond,
		SFX:      []string{"_heal_effect.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          color.RGBA{R: 255, G: 182, B: 86, A: 255},
			STRFile:        "fruit",
			AttachedEntity: true,
		}},
	},
	effectFoodBlue: {
		Duration: 850 * time.Millisecond,
		SFX:      []string{"effect\\흡기.wav"},
		Components: []EffectComponent{{
			Kind:           EffectComponentSTR,
			Color:          color.RGBA{R: 132, G: 112, B: 255, A: 255},
			STRFile:        "fruit_",
			AttachedEntity: true,
		}},
	},
	effectBottomDissonance: {},
	effectBottomUglyDance:  {},
	effectChristmasCarol:   strEffectSpecAttachedMin("angelus", "jong_mini", "effect\\wewish.wav", false),
}

func bashHitComponents() []EffectComponent {
	angleRanges := [][2]float64{
		{0, 35},
		{50, 85},
		{100, 135},
		{150, 185},
		{200, 235},
		{255, 290},
		{300, 335},
		{340, 360},
	}
	components := make([]EffectComponent, 0, len(angleRanges))
	for index, angleRange := range angleRanges {
		textureFile := "effect/lens1.tga"
		if index%2 == 1 {
			textureFile = "effect/lens2.tga"
		}
		components = append(components, EffectComponent{
			Kind:               EffectComponent2D,
			TextureFile:        textureFile,
			Duration:           250 * time.Millisecond,
			DurationRandMin:    200 * time.Millisecond,
			DurationRandMax:    350 * time.Millisecond,
			AlphaMax:           12,
			Fade:               true,
			FadeOut:            true,
			SizeStartXRandMin:  25 * EffectPixelRatio,
			SizeStartXRandMax:  40 * EffectPixelRatio,
			SizeStartY:         10 * EffectPixelRatio,
			SizeEndX:           1 * EffectPixelRatio,
			SizeEndYRandMin:    250 * EffectPixelRatio,
			SizeEndYRandMax:    300 * EffectPixelRatio,
			AngleRandMin:       angleRange[0],
			AngleRandMax:       angleRange[1],
			CirclePattern:      true,
			CircleInnerSize:    2.2,
			CircleOuterRandMin: 5,
			CircleOuterRandMax: 6,
			Overlay:            true,
		})
	}
	return components
}

func hitCylinderComponent(bottomSize, topSize float64) EffectComponent {
	return EffectComponent{
		Kind:             EffectComponentCylinder,
		TextureName:      "lens2",
		Duration:         150 * time.Millisecond,
		AlphaMax:         0.8,
		Fade:             true,
		RotateWithCamera: true,
		Animation:        1,
		BottomSize:       bottomSize,
		TopSize:          topSize,
		Height:           4,
		PosZ:             1,
		AngleX:           -90,
		AttachedEntity:   true,
		TotalCircleSides: 24,
		CircleSides:      24,
	}
}

func hitSlashComponent(kind EffectComponentKind, sizeX, sizeEndY, angleStart, angleEnd float64, overlay bool) EffectComponent {
	return EffectComponent{
		Kind:        kind,
		TextureFile: "effect/lens2.tga",
		Duration:    400 * time.Millisecond,
		AlphaMax:    1,
		FadeOut:     true,
		Rotate:      true,
		PosZ:        1,
		SizeStartX:  sizeX,
		SizeEndX:    sizeX,
		SizeStartY:  effectTableSize(10),
		SizeEndY:    sizeEndY,
		AngleStart:  angleStart,
		AngleEnd:    angleEnd,
		Overlay:     overlay,
	}
}

func castAuraEffectSpec(texture string, tint color.RGBA, alphaMax, height, topSize float64, rotate bool) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{{
			Kind:             EffectComponentCylinder,
			TextureName:      texture,
			AlphaMax:         alphaMax,
			Fade:             true,
			Rotate:           rotate,
			Animation:        2,
			BottomSize:       1,
			TopSize:          topSize,
			Height:           height,
			TotalCircleSides: 32,
			CircleSides:      32,
			Color:            tint,
		}},
	}
}

func elementalCastAuraEffectSpec(texture string, tint color.RGBA, alphaMax float64) EffectSpec {
	return EffectSpec{
		Duration: 900 * time.Millisecond,
		SFX:      []string{"effect\\ef_beginspell.wav"},
		Components: []EffectComponent{
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         0.3,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1,
				Height:           30,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        1,
				BottomSize:       1,
				TopSize:          1.3,
				Height:           2,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
			{
				Kind:             EffectComponentCylinder,
				TextureName:      texture,
				AlphaMax:         alphaMax,
				Fade:             true,
				Rotate:           true,
				Animation:        2,
				BottomSize:       1,
				TopSize:          4,
				Height:           3,
				TotalCircleSides: 32,
				CircleSides:      32,
				Color:            tint,
			},
		},
	}
}

func bluePotionEffectSpec() EffectSpec {
	spec := potionEffectSpec("파란포션", color.RGBA{R: 92, G: 150, B: 255, A: 255})
	spec.SFX = []string{"effect\\흡기.wav"}
	return spec
}

func incAgilityParticleComponent(alpha float64, delay time.Duration, duplicate int) EffectComponent {
	return EffectComponent{
		Kind:            EffectComponent3D,
		Color:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		TextureFile:     "effect/ac_center2.tga",
		Duration:        1000 * time.Millisecond,
		Delay:           delay,
		DuplicateDelay:  200 * time.Millisecond,
		AlphaMax:        alpha,
		FadeOut:         true,
		PosXRand:        1.5,
		PosYRand:        1,
		PosZStartRand:   1,
		PosZStartMiddle: 1,
		PosZEndRand:     1,
		PosZEndMiddle:   6,
		SizeStartX:      2.5 * EffectPixelRatio,
		SizeEndX:        2.5 * EffectPixelRatio,
		SizeRandY:       15 * EffectPixelRatio,
		SizeRandYMiddle: 45 * EffectPixelRatio,
		Duplicate:       duplicate,
	}
}

func decAgilityParticleComponent() EffectComponent {
	return EffectComponent{
		Kind:            EffectComponent3D,
		TextureFile:     "effect/ac_center2.tga",
		Duration:        1000 * time.Millisecond,
		DuplicateDelay:  200 * time.Millisecond,
		AlphaMax:        1,
		FadeOut:         true,
		PosXRand:        1.5,
		PosYRand:        1,
		PosZStartRand:   1,
		PosZStartMiddle: 6,
		PosZEndRand:     1,
		PosZEndMiddle:   1,
		SizeStartX:      effectTableSize(2.5),
		SizeEndX:        effectTableSize(2.5),
		SizeRandY:       effectTableSize(15),
		SizeRandYMiddle: effectTableSize(45),
		Duplicate:       20,
	}
}
