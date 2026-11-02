package db

import (
	"image/color"
	"time"
)

const EffectFrameDuration = time.Second / 60

const (
	ReferenceActiveEffectTableEntries = 607
	ReferenceNumericEffectConstants   = 1147
)

// Native effect IDs.
const (
	effectHit1              = 0
	effectBashHit           = 1
	effectHit3              = 2
	effectHit4              = 3
	effectHit5              = 4
	effectHit6              = 5
	effectEntry             = 6
	effectExit              = 7
	effectWarp              = 8
	effectEnhance           = 9
	effectMammonite         = 10
	effectEndure            = 11
	effectBeginSpell        = 12
	effectGlassWall         = 13
	effectHealSP            = 14
	effectSoulStrike        = 15
	effectBashBegin         = 16
	effectMagnumBreak       = 17
	effectSteal             = 18
	effectPoisonAttack      = 20
	effectDetoxication      = 21
	effectSight             = 22
	effectStoneCurse        = 23
	effectFireBall          = 24
	effectFireWall          = 25
	effectIceArrow          = 26
	effectFrostDiver        = 27
	effectFrostDiverHit     = 28
	effectLightningBolt     = 29
	effectThunderStorm      = 30
	effectFireArrow         = 31
	effectNapalmBeat        = 32
	effectRuwach            = 33
	effectTeleportOld       = 34
	effectReadyPortalOld    = 35
	effectIncAgility        = 37
	effectDecAgility        = 38
	effectAqua              = 39
	effectSignum            = 40
	effectAngelus           = 41
	effectBlessing          = 42
	effectIncAgiDex         = 43
	effectSmoke             = 44
	effectFirefly           = 45
	effectTorch             = 47
	effectFireHit           = 49
	effectFireSplashHit     = 50
	effectColdHit           = 51
	effectWindHit           = 52
	effectPoisonHit         = 53
	effectBeginSpell2       = 54
	effectBeginSpell3       = 55
	effectBeginSpell4       = 56
	effectBeginSpell5       = 57
	effectBeginSpell6       = 58
	effectBeginSpell7       = 59
	effectLockOnTarget      = 60
	effectWarpZone          = 61
	effectSightTrasher      = 62
	effectArrowShotRO       = 64
	effectInvenom           = 65
	effectCure              = 66
	effectProvoke           = 67
	effectMvp               = 68
	effectSkidTrap          = 69
	effectBrandishSpear     = 70
	effectBowlingBash       = 73
	effectIceWall           = 74
	effectGloria            = 75
	effectMagnificat        = 76
	effectResurrection      = 77
	effectRecovery          = 78
	effectEarthSpike        = 79
	effectSpearBoomerang    = 80
	effectPierce            = 81
	effectTurnUndead        = 82
	effectSanctuary         = 83
	effectImpositio         = 84
	effectLexAeterna        = 85
	effectAspersio          = 86
	effectLexDivina         = 87
	effectSuffragium        = 88
	effectStormGust         = 89
	effectLordVermilion     = 90
	effectBenedictio        = 91
	effectMeteorStorm       = 92
	effectJupitelThunder    = 93
	effectJupitelHit        = 94
	effectQuagmire          = 95
	effectFirePillar        = 96
	effectFirePillarBomb    = 97
	effectHasteUp           = 98
	effectFlasher           = 99
	effectRemoveTrap        = 100
	effectRepairWeapon      = 101
	effectCrashEarth        = 102
	effectWeaponPerfect     = 103
	effectMaximizePower     = 104
	effectBlastMine         = 105
	effectBlastMineBomb     = 106
	effectClaymore          = 107
	effectFreezingTrap      = 108
	effectBubble            = 109
	effectGasPush           = 110
	effectSpringTrap        = 111
	effectKyrie             = 112
	effectMagnus            = 113
	effectBlitzBeat         = 115
	effectWaterBall         = 116
	effectWaterBall2        = 117
	effectDetecting         = 119
	effectCloaking          = 120
	effectSonicBlow         = 121
	effectSonicBlowHit      = 122
	effectGrimtooth         = 123
	effectVenomDust         = 124
	effectEnchantPoison     = 125
	effectPoisonReact       = 126
	effectPoisonReact2      = 127
	effectOverthrust        = 128
	effectVenomSplasher     = 129
	effectTwoHandQuicken    = 130
	effectAutoCounter       = 131
	effectGrimtoothAtk      = 132
	effectFreeze            = 133
	effectFreezed           = 134
	effectIceCrash          = 135
	effectSlowPoison        = 136
	effectFirePillarOn      = 138
	effectSandman           = 139
	effectRevive            = 140
	effectPneuma            = 141
	effectHeavenDrive       = 142
	effectSonicBlow2        = 143
	effectBrandishSpear2    = 144
	effectShockwave         = 145
	effectShockwaveHit      = 146
	effectEarthHit          = 147
	effectPierceSelf        = 148
	effectBowlingSelf       = 149
	effectSpearStabSelf     = 150
	effectSpearBmrSelf      = 151
	effectHolyLight         = 152
	effectConcentration     = 153
	effectRefineOK          = 154
	effectRefineFail        = 155
	effectJobLevelUp        = 158
	effectRain              = 161
	effectSnow              = 162
	effectSakura            = 163
	effectBanjjakii         = 165
	effectMakeBlur          = 166
	effectEnergyCoat        = 169
	effectCartRevolution    = 170
	effectVenomDust2        = 171
	effectMentalBreak       = 181
	effectMagicalAtkHit     = 182
	effectSuiExplosion      = 183
	effectSuicide           = 185
	effectComboAttack1      = 186
	effectComboAttack2      = 187
	effectComboAttack3      = 188
	effectComboAttack4      = 189
	effectComboAttack5      = 190
	effectGuidedAttack      = 191
	effectPoisonAttack2     = 192
	effectSilenceAttack     = 193
	effectStunAttack        = 194
	effectPetrifyAttack     = 195
	effectSleepAttack       = 197
	effectPong              = 199
	effectLevel99           = 200
	effectLevel99Ground     = 201
	effectLevel99Bubble     = 202
	effectGumgang           = 203
	effectPotionRed         = 204
	effectPotionOrange      = 205
	effectPotionYellow      = 206
	effectPotionWhite       = 207
	effectPotionBlue        = 208
	effectPotionGreen       = 209
	effectFood              = 210
	effectFoodBlue          = 211
	effectDarkBreath        = 212
	effectDefender          = 213
	effectKeeping           = 214
	effectSummonSlave       = 215
	effectBloodDrain        = 216
	effectEnergyDrain       = 217
	effectItemFast          = 218
	effectItemFast2         = 219
	effectItemFast3         = 220
	effectCrusaderDef       = 222
	effectGanbantein        = 223
	effectVolcano           = 225
	effectGrandCross        = 226
	effectIntimidate        = 227
	effectChookgi           = 228
	effectCloud             = 229
	effectCloud2            = 230
	effectMapPillar         = 231
	effectLineLink          = 232
	effectCloud3            = 233
	effectSpellBreaker      = 234
	effectDispell           = 235
	effectDeluge            = 236
	effectViolentGale       = 237
	effectLandProtector     = 238
	effectBottomVolcano     = 239
	effectBottomDeluge      = 240
	effectBottomViolent     = 241
	effectBottomLand        = 242
	effectMagicRod          = 244
	effectHolyCross         = 245
	effectShieldCharge      = 246
	effectMapPillar2        = 247
	effectProvidence        = 248
	effectShieldBoomer      = 249
	effectSpearQuicken      = 250
	effectDevotion          = 251
	effectReflectShield     = 252
	effectAbsorbSpirits     = 253
	effectSteelBody         = 254
	effectFlameLauncher     = 255
	effectFrostWeapon       = 256
	effectLightningLoad     = 257
	effectSeismicWeapon     = 258
	effectMapPillar3        = 259
	effectMapPillar4        = 260
	effectGumgang2          = 261
	effectTeiHit1           = 262
	effectGumgang3          = 263
	effectTeiHit2           = 264
	effectTanji             = 265
	effectTeiHit1X          = 266
	effectChimto            = 267
	effectStealCoin         = 268
	effectStripWeapon       = 269
	effectStripShield       = 270
	effectStripArmor        = 271
	effectStripHelm         = 272
	effectChainCombo        = 273
	effectRogueCoin         = 274
	effectBackStab          = 275
	effectTeiHit3           = 276
	effectBottomDissonance  = 277
	effectBottomLullaby     = 278
	effectBottomRichKim     = 279
	effectBottomChaos       = 280
	effectBottomDrum        = 281
	effectBottomNibelung    = 282
	effectBottomRoki        = 283
	effectBottomAbyss       = 284
	effectBottomSieg        = 285
	effectBottomWhistle     = 286
	effectBottomSinX        = 287
	effectBottomBragi       = 288
	effectBottomApple       = 289
	effectBottomUglyDance   = 290
	effectBottomHumming     = 291
	effectBottomForget      = 292
	effectBottomFortune     = 293
	effectBottomService     = 294
	effectTalkFrostJoke     = 295
	effectTalkScream        = 296
	effectPokJuk            = 297
	effectThrowItem         = 298
	effectThrowItem2        = 299
	effectChemicalProt      = 300
	effectDemonstration     = 302
	effectChemical2         = 303
	effectTeleportation     = 304
	effectPharmacyOK        = 305
	effectPharmacyFail      = 306
	effectThrowItem3        = 308
	effectFirstAid          = 309
	effectSprinkleSand      = 310
	effectLoud              = 311
	effectHeal              = 312
	effectHeal2             = 313
	effectExit2             = 314
	effectSafetyWall        = 315
	effectReadyPortal       = 316
	effectPortal            = 317
	effectBottomMagnus      = 318
	effectBottomSanc        = 319
	effectHealOffensive     = 320
	effectWarpZone2         = 321
	effectHeal4             = 325
	effectBeginAsura        = 328
	effectTripleAttack      = 329
	effectHitLine           = 330
	effectHPTime            = 331
	effectSPTime            = 332
	effectMaple             = 333
	effectBlind             = 334
	effectPoisonStatus      = 335
	effectGuard             = 336
	effectJobLvUp50         = 337
	effectAngel2            = 338
	effectMagnum2           = 339
	effectCoupleCasting     = 342
	effectEntry2            = 344
	effectColorPaper        = 347
	effectSoulBreaker       = 361
	effectLevel99Aura1      = 362
	effectFoodChocolate     = 363
	effectPressure          = 365
	effectBash3D            = 366
	effectAuraBlade         = 367
	effectRedBody           = 368
	effectLKConcentration   = 369
	effectBottomGospel      = 370
	effectBaseLevelUp       = 371
	effectDeath             = 372
	effectDragonSmoke       = 373
	effectBottomBasilica    = 374
	effectAssumptio         = 375
	effectHitLine2          = 376
	effectBash3D2           = 377
	effectEnergyDrain2      = 378
	effectTransBlueBody     = 379
	effectMagicCrasher      = 380
	effectLightBlade        = 382
	effectEnergyDrain3      = 383
	effectLineLink2         = 384
	effectLinkLight         = 385
	effectTrueSight         = 386
	effectFalconAssault     = 387
	effectTripleAttack2     = 388
	effectPortal4           = 389
	effectMeltdown          = 390
	effectCartBoost         = 391
	effectRejectSword       = 392
	effectTripleAttack3     = 393
	effectMoonlit           = 394
	effectMarionetteLink    = 395
	effectPinkBody          = 396
	effectLevel99AuraMid    = 397
	effectLevel99AuraBottom = 398
	effectBash3D3           = 399
	effectBash3D4           = 400
	effectNapalmVulcan      = 401
	effectPortal5           = 402
	effectMagicCrasher2     = 403
	effectBottomSpider      = 404
	effectFogWallGround     = 405
	effectSoulBurn          = 406
	effectSoulChange        = 407
	effectSoulBreaker2      = 409
	effectRainbow           = 410
	effectPeong             = 411
	effectPressedBody       = 413
	effectSpinedBody        = 414
	effectKickedBody        = 415
	effectDoubleGumgang     = 418
	effectReflectBody       = 419
	effectBabyBody          = 420
	effectBabyBody2         = 421
	effectGiantBody         = 422
	effectGiantBody2        = 423
	effectAsuraBody         = 424
	effectFourWayBody       = 425
	effectQuakeBody         = 426
	effectAsuraBodyMonster  = 427
	effectHitLine4          = 429
	effectHitLine5          = 430
	effectHitLine6          = 431
	effectStormKick         = 435
	effectChemical3         = 439
	effectAssumptio2        = 440
	effectBlueCasting       = 441
	effectRun               = 442
	effectStopRun           = 443
	effectStopEffect        = 444
	effectJumpBody          = 445
	effectLandBody          = 446
	effectFoot3             = 447
	effectFoot4             = 448
	effectTaeReady          = 449
	effectDarkGrandCross    = 450
	effectDarkSoulStrike    = 451
	effectDarkJupitelHit    = 452
	effectNPCStop           = 453
	effectDarkCasting       = 454
	effectAgiUp             = 456
	effectJumpKick          = 457
	effectQuakeBody2        = 458
	effectStormKick3        = 461
	effectStormKick4        = 462
	effectSpinedBody2       = 466
	effectBeginAsura1       = 467
	effectBeginAsura2       = 468
	effectBeginAsura3       = 469
	effectBeginAsura4       = 470
	effectBeginAsura5       = 471
	effectBeginAsura6       = 472
	effectBeginAsura7       = 473
	effectDevil1            = 475
	effectDevil2            = 476
	effectDevil3            = 477
	effectDevil4            = 478
	effectDevil5            = 479
	effectDevil6            = 480
	effectDevil7            = 481
	effectDevil8            = 482
	effectDevil9            = 483
	effectDevil10           = 484
	effectBlackDevil        = 487
	effectFlowerCast        = 488
	effectResistPotion      = 491
	effectMochi             = effectResistPotion
	effectRamadan           = 492
	effectEDP               = 493
	effectFullStrip         = 495
	effectPreserve          = 496
	effectSlim              = 497
	effectSlim2             = 498
	effectSlim3             = 499
	effectChemicalBody      = 500
	effectCastSpin          = 501
	effectPierceBody        = 502
	effectSoulLink          = 503
	effectChookgi2          = 504
	effectMemorize          = 505
	effectItemAccel         = 507
	effectMapae             = effectItemAccel
	effectFirecracker       = 508
	effectItemPokJuk        = effectFirecracker
	effectValentine05       = 509
	effectBeginAsura11      = 510
	effectChemical2Dash     = 512
	effectGroundSample      = 513
	effectCloud4            = 515
	effectCloud5            = 516
	effectBottomHermode     = 517
	effectCartTermination   = 518
	effectItemSlow          = 519
	effectItemFastDown      = effectItemSlow
	effectShieldBoomerang3  = 520
	effectDoubleCastBody    = 521
	effectGravitation       = 522
	effectTarotCard1        = 523
	effectTarotCard2        = 524
	effectTarotCard3        = 525
	effectTarotCard4        = 526
	effectTarotCard5        = 527
	effectTarotCard6        = 528
	effectTarotCard7        = 529
	effectTarotCard8        = 530
	effectTarotCard9        = 531
	effectTarotCard10       = 532
	effectTarotCard11       = 533
	effectTarotCard12       = 534
	effectTarotCard13       = 535
	effectTarotCard14       = 536
	effectAcidDemon         = 537
	effectThrowItem4        = 539
	effectBabyBodyBack      = 540
	effectThrowItem5        = 541
	effectBlueBody          = 542
	effectHated             = 543
	effectRedLightBody      = 544
	effectSmaReady          = 546
	effectStin              = 547
	effectRedHit            = 548
	effectBlueHit           = 549
	effectQuakeBody3        = 550
	effectStin2             = 553
	effectHitTexture        = 554
	effectStin3             = 555
	effectScreenQuake       = 563
	effectHfliMoon1         = 565
	effectHfliMoon2         = 566
	effectHfliMoon3         = 567
	effectHoUp              = 568
	effectHamiDefence       = 569
	effectHamiCastle        = 570
	effectHamiBlood         = 571
	effectHated2            = 572
	effectTwilight1         = 573
	effectTwilight2         = 574
	effectTwilight3         = 575
	effectBoxThunder        = 576
	effectItemThunder       = effectBoxThunder
	effectBoxResentment     = 577
	effectItemCloud         = effectBoxResentment
	effectItemCurse         = 578
	effectBoxDrowsiness     = 579
	effectItemZZZ           = effectBoxDrowsiness
	effectBoxSunlight       = 580
	effectItemRain          = effectBoxSunlight
	effectM01               = 583
	effectM02               = 584
	effectM03               = 585
	effectM04               = 586
	effectM05               = 587
	effectM06               = 588
	effectM07               = 589
	effectKaizel            = 590
	effectCloud6            = 592
	effectStatFoodSTR       = 593
	effectStatFoodINT       = 594
	effectStatFoodVIT       = 595
	effectStatFoodAGI       = 596
	effectStatFoodDEX       = 597
	effectStatFoodLUK       = 598
	effectShrink            = 599
	effectThrowItem6        = 600
	effectSightBlaster      = 601
	effectQuakeBody4        = 602
	effectFireHit2          = 603
	effectNPCStop2          = 604
	effectNPCStop2Del       = 605
	effectFVoice            = 606
	effectWink              = 607
	effectCookingOK         = 608
	effectCookingFail       = 609
	effectFirecracker1      = 612
	effectHapgyeok          = 612
	effectThrowItem7        = 613
	effectThrowItem8        = 614
	effectThrowItem9        = 615
	effectThrowItem10       = 616
	effectKouenka           = 618
	effectHyousensou        = 619
	effectBottomSuiton      = 620
	effectStin4             = 621
	effectThunderStorm2     = 622
	effectChemical4         = 623
	effectStin5             = 624
	effectMadnessBlue       = 625
	effectMadnessRed        = 626
	effectRGCoin3           = 627
	effectBash3D5           = 628
	effectChookgi3          = 629
	effectKirikage          = 630
	effectTatami            = 631
	effectKasumikiri        = 632
	effectIssen             = 633
	effectKaen              = 634
	effectBaku              = 635
	effectHyousyouraku      = 636
	effectDesperado         = 637
	effectLightningS        = 638
	effectBlindS            = 639
	effectPoisonS           = 640
	effectFreezingS         = 641
	effectFlareS            = 642
	effectRapidShower       = 643
	effectMagicalBullet     = 644
	effectSpreadAttack      = 645
	effectTrackCasting      = 646
	effectTracking          = 647
	effectTripleAction      = 648
	effectBullseye          = 649
	effectBubbleDrop        = 665
	effectNPCEarthquake     = 666
	effectDragonFear        = 668
	effectWideBleeding      = 669
	effectWideConfuse       = 670
	effectBottomRunner      = 671
	effectBottomTransfer    = 672
	effectBottomEvilLand    = 674
	effectGuard3            = 675
	effectCriticalWound     = 677
	effectFirecracker2      = 682
	effectFirecracker3      = 683
	effectFirecracker4      = 684
	effectFirecracker5      = 685
	effectFirecracker6      = 686
	effectTorchRed          = 690
	effectTorchGreen        = 691
	effectMapGhost          = 692
	effectGlow1             = 693
	effectGlow2             = 694
	effectGlow4             = 695
	effectTorchPurple       = 696
	effectCloud7            = 697
	effectCloud8            = 698
	effectFlowerLeaf        = 699
	effectItem315           = 704
	effectItem316           = 705
	effectItem317           = 706
	effectStormMin          = 708
	effectFirecracker7      = 709
	effectBottomBlue        = 715
	effectBottomBlue2       = 716
	effectChristmasCarol    = 717
	effectFirePillarOn2     = 718
	effectForestLight5      = 719
	effectAdoramus          = 721
	effectIgnitionBreak     = 722
	effectFrostMisty        = 726
	effectCrimsonRock       = 727
	effectHellInferno       = 728
	effectMarshOfAbyss      = 729
	effectDragonHowling     = 731
	effectEarthWall         = 732
	effectChainLightning    = 734
	effectAimedBolt         = 745
	effectArrowStorm        = 746
	effectLaulamus          = 747
	effectLauagnus          = 748
	effectMillenniumShield  = 749
	effectConcentration2    = 750
	effectGlassWall3        = 754
	effectBerserkPotion2    = 756
	effectRolling1          = 758
	effectRolling2          = 759
	effectRolling3          = 760
	effectRolling4          = 761
	effectRolling5          = 762
	effectRolling6          = 763
	effectRolling7          = 764
	effectRolling8          = 765
	effectRolling9          = 766
	effectRolling10         = 767
	effectCastSpin2         = 775
	effectCrashAxe          = 795
	effectStasis            = 799
	effectBottomBasilica2   = 802
	effectRecognized        = 803
	effectTetra             = 804
	effectTetraCasting      = 805
	effectStretch           = 811
	effectEnervation        = 813
	effectEnervation2       = 814
	effectEnervation3       = 815
	effectEnervation4       = 816
	effectEnervation5       = 817
	effectEnervation6       = 818
	effectBottomManhole     = 822
	effectManhole           = 823
	effectForestLight6      = 825
	effectBottomAni         = 827
	effectBottomMaelstrom   = 828
	effectBottomBloodyLust  = 829
	effectHealN             = 832
	effectChookgiN          = 833
	effectDance1            = 847
	effectBotReverb         = 856
	effectRainParticle      = 857
	effectChemicalV2        = 858
	effectBotReverb2        = 860
	effectCirclePower2      = 861
	effectSecra2            = 862
	effectSprPlant2         = 866
	effectSprPlant3         = 868
	effectSprPlant4         = 870
	effectSprPlant5         = 872
	effectSprPlant6         = 874
	effectSprPlant7         = 876
	effectSprPlant8         = 878
	effectHeartAsura        = 879
	effectGlassWall4        = 883
	effectBash3D6           = 885
	effectElectric4         = 888
	effectTeiHit1T          = 889
	effectPressure2         = 906
	effectPrimeCharge2      = 908
	effectPrimeCharge3      = 909
	effectPrimeCharge4      = 910
	effectFireWall2         = 920
	effectSprPlant10        = 922
	effectShockwave2        = 926
	effectColdThrow2        = 928
	effectDemonicFire4      = 929
	effectPressure3         = 930
	effectPoisonMist        = 959
	effectEraserCutter      = 960
	effectLavaSlide         = 964
	effectSonicClaw         = 965
	effectTinderBreaker     = 966
	effectMidnightFrenzy    = 967
	effectVolcanicAsh       = 975
	effectRWC2011           = 987
	effectRWC2011Two        = 988
	effectRunMakeOK         = 1015
	effectRunMakeFailure    = 1016
	effectMIResultMakeOK    = 1017
	effectMIResultMakeFail  = 1018
	effectAllRayProtect     = 1019
	effectVenomFog          = 1020
	effectDustStorm         = 1021
	effectDanceBladeAtk     = 1029
	effectInvincibleOff2    = 1031
	effectDeathSummon       = 1033
	effectGCDarkCrow        = 1040
	effectAllFullThrottle   = 1042
	effectSRFlashCombo      = 1043
	effectRKLuxAnima        = 1044
	effectSOElemShield      = 1046
	effectABOffertorium     = 1047
	effectWLTelekinesis     = 1048
	effectGNIllusionDoping  = 1049
	effectNCMagmaEruption   = 1050
	effectNPCChill          = 1055
	effectOffertoriumRing   = 1057
	effectHammerOfGod       = 1062
	effectAchComplete       = 1094
	effectBodyColor         = 1111
	effectBakuretsuHadou    = 1130
	dropEffectPink          = 1186
	dropEffectYellow        = 1189
	dropEffectPurple        = 1190
	effectDigitalSpace      = 1240
	dropEffectBlue          = 1869
	dropEffectGreen         = 1870
	dropEffectRed           = 1871
	effectNewSuccess        = 1872
	effectNewFailure        = 1873
	effectNewIntro          = 1874
	effectEnchantYellow     = 1875
	effectEnchantSuccess    = 1876
	effectEnchantFail       = 1877
	effectEnchantBlue       = 1878
	effectEnchantUpSuccess  = 1879
	effectEnchantUpFail     = 1880
	effectEnchantGreen      = 1881
	effectEnchantResetOK    = 1882
	effectEnchantResetFail  = 1883
)

// Goro-local effect IDs.
const (
	effectColdBolt               = 10014
	effectFireBolt               = 10019
	effectCastRing               = 10021
	effectQuakeMagnum            = 10022
	effectArrowShot              = 10060
	effectArrowShower            = 10061
	effectMagicPower             = SkillEffectMagicPower
	effectWhitePulse             = SkillEffectWhitePulse
	effectSpearProjectile        = SkillEffectSpearProjectile
	effectSpearHitSound          = SkillEffectSpearHitSound
	effectEnemyHitNormal1        = SkillEffectEnemyHitNormal1
	effectQuake                  = SkillEffectQuake
	effectAnkleSnareGround       = SkillEffectAnkleSnareGround
	effectSharpShootingCast      = SkillEffectSharpShootingCast
	effectAdrenalineCast         = SkillEffectAdrenalineCast
	effectMaximizeSounds         = SkillEffectMaximizeSounds
	effectGreedSound             = SkillEffectGreedSound
	effectGospelGround           = SkillEffectGospelGround
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
	effectSteelBodyRing          = SkillEffectSteelBodyRing
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
	HitRing            *EffectHitRing
	HitParticles       *EffectHitParticles
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

// EffectSpecs is adapted from robr's DB/Effects/EffectTable.js and verified
// original-client behavior. Leave effects unsupported until their reference
// behavior is understood rather than guessing local visuals.
var EffectSpecs = map[int]EffectSpec{
	effectHit1:              hit1EffectSpec(),
	effectBashHit:           bashHitEffectSpec(),
	effectHit3:              hit3EffectSpec(),
	effectHit4:              hit4EffectSpec(),
	effectHit5:              hit5EffectSpec(),
	effectHit6:              hit6EffectSpec(),
	effectEntry:             entryEffectSpec(),
	effectExit:              exitEffectSpec(),
	effectWarp:              warpEffectSpec(),
	effectEnhance:           enhanceEffectSpec(),
	effectMammonite:         mammoniteEffectSpec(),
	effectEndure:            endureEffectSpec(),
	effectBeginSpell:        beginSpellEffectSpec(),
	effectGlassWall:         glassWallEffectSpec(),
	effectHealSP:            healSPEffectSpec(),
	effectSoulStrike:        soulStrikeEffectSpec(),
	effectBashBegin:         bashBeginEffectSpec(),
	effectMagnumBreak:       magnumBreakEffectSpec(),
	effectSteal:             stealEffectSpec(),
	effectPoisonAttack:      poisonAttackEffectSpec(),
	effectDetoxication:      detoxicationEffectSpec(),
	effectSight:             mageSightEffectSpec(),
	effectStoneCurse:        stoneCurseEffectSpec(),
	effectFireBall:          fireBallEffectSpec(),
	effectFireWall:          fireWallEffectSpec(),
	effectIceArrow:          coldBoltEffectSpec(),
	effectFrostDiver:        frostDiverEffectSpec(),
	effectFrostDiverHit:     mageFrostHitEffectSpec(),
	effectLightningBolt:     lightningBoltEffectSpec(),
	effectThunderStorm:      thunderStormEffectSpec(),
	effectFireArrow:         fireBoltEffectSpec(),
	effectNapalmBeat:        napalmBeatEffectSpec(),
	effectRuwach:            ruwachEffectSpec(),
	effectTeleportOld:       teleportOldEffectSpec(),
	effectReadyPortalOld:    readyPortalOldEffectSpec(),
	effectIncAgility:        incAgilityEffectSpec(),
	effectDecAgility:        decAgilityEffectSpec(),
	effectAqua:              aquaEffectSpec(),
	effectSignum:            signumEffectSpec(),
	effectAngelus:           angelusEffectSpec(),
	effectBlessing:          blessingEffectSpec(),
	effectIncAgiDex:         concentrationBeginEffectSpec(),
	effectSmoke:             smokeEffectSpec(),
	effectFirefly:           fireflyEffectSpec(),
	effectTorch:             torchEffectSpec(),
	effectFireHit:           fireHitEffectSpec(),
	effectFireSplashHit:     fireSplashHitEffectSpec(),
	effectColdHit:           coldHitEffectSpec(),
	effectWindHit:           windHitEffectSpec(),
	effectPoisonHit:         poisonHitEffectSpec(),
	effectBeginSpell2:       beginSpell2EffectSpec(),
	effectBeginSpell3:       beginSpell3EffectSpec(),
	effectBeginSpell4:       beginSpell4EffectSpec(),
	effectBeginSpell5:       beginSpell5EffectSpec(),
	effectBeginSpell6:       beginSpell6EffectSpec(),
	effectBeginSpell7:       beginSpell7EffectSpec(),
	effectLockOnTarget:      lockOnTargetEffectSpec(),
	effectWarpZone:          warpZoneEffectSpec(),
	effectSightTrasher:      sightTrasherEffectSpec(),
	effectArrowShotRO:       arrowShotROEffectSpec(),
	effectInvenom:           invenomEffectSpec(),
	effectCure:              cureEffectSpec(),
	effectProvoke:           provokeEffectSpec(),
	effectMvp:               mvpEffectSpec(),
	effectSkidTrap:          skidTrapEffectSpec(),
	effectBrandishSpear:     brandishSpearEffectSpec(),
	effectBowlingBash:       bowlingBashEffectSpec(),
	effectIceWall:           iceWallEffectSpec(),
	effectGloria:            gloriaEffectSpec(),
	effectMagnificat:        magnificatEffectSpec(),
	effectResurrection:      resurrectionEffectSpec(),
	effectRecovery:          recoveryEffectSpec(),
	effectEarthSpike:        earthSpikeEffectSpec(),
	effectSpearBoomerang:    spearBoomerangEffectSpec(),
	effectPierce:            pierceEffectSpec(),
	effectTurnUndead:        turnUndeadEffectSpec(),
	effectSanctuary:         sanctuaryEffectSpec(),
	effectImpositio:         impositioEffectSpec(),
	effectLexAeterna:        lexAeternaEffectSpec(),
	effectAspersio:          aspersioEffectSpec(),
	effectLexDivina:         lexDivinaEffectSpec(),
	effectSuffragium:        suffragiumEffectSpec(),
	effectStormGust:         stormGustEffectSpec(),
	effectLordVermilion:     lordVermilionEffectSpec(),
	effectBenedictio:        benedictioEffectSpec(),
	effectMeteorStorm:       meteorStormEffectSpec(),
	effectJupitelThunder:    jupitelThunderEffectSpec(),
	effectJupitelHit:        jupitelHitEffectSpec(),
	effectQuagmire:          quagmireEffectSpec(),
	effectFirePillar:        firePillarEffectSpec(),
	effectFirePillarBomb:    firePillarBombEffectSpec(),
	effectHasteUp:           hasteUpEffectSpec(),
	effectFlasher:           flasherEffectSpec(),
	effectRemoveTrap:        removeTrapEffectSpec(),
	effectRepairWeapon:      repairWeaponEffectSpec(),
	effectCrashEarth:        crashEarthEffectSpec(),
	effectWeaponPerfect:     weaponPerfectEffectSpec(),
	effectMaximizePower:     maximizePowerEffectSpec(),
	effectBlastMine:         blastMineSoundEffectSpec(),
	effectBlastMineBomb:     blastMineEffectSpec(),
	effectClaymore:          claymoreEffectSpec(),
	effectFreezingTrap:      freezingTrapEffectSpec(),
	effectBubble:            bubbleEffectSpec(),
	effectGasPush:           gasPushEffectSpec(),
	effectSpringTrap:        springTrapEffectSpec(),
	effectKyrie:             kyrieEffectSpec(),
	effectMagnus:            magnusEffectSpec(),
	effectBlitzBeat:         blitzBeatEffectSpec(),
	effectWaterBall:         waterBallEffectSpec(),
	effectWaterBall2:        waterBall2EffectSpec(),
	effectDetecting:         detectingEffectSpec(),
	effectCloaking:          cloakingEffectSpec(),
	effectSonicBlow:         sonicBlowEffectSpec(),
	effectSonicBlowHit:      sonicBlowHitEffectSpec(),
	effectGrimtooth:         grimtoothGroundEffectSpec(),
	effectVenomDust:         venomDustEffectSpec(),
	effectEnchantPoison:     enchantPoisonEffectSpec(),
	effectPoisonReact:       poisonReactEffectSpec(),
	effectPoisonReact2:      poisonReact2EffectSpec(),
	effectOverthrust:        overthrustEffectSpec(),
	effectVenomSplasher:     venomSplasherEffectSpec(),
	effectTwoHandQuicken:    twoHandQuickenEffectSpec(),
	effectAutoCounter:       autoCounterEffectSpec(),
	effectGrimtoothAtk:      grimtoothAtkEffectSpec(),
	effectFreeze:            freezeEffectSpec(),
	effectFreezed:           freezedEffectSpec(),
	effectIceCrash:          iceCrashEffectSpec(),
	effectSlowPoison:        slowPoisonEffectSpec(),
	effectFirePillarOn:      firePillarOnEffectSpec(),
	effectSandman:           sandmanEffectSpec(),
	effectRevive:            reviveEffectSpec(),
	effectPneuma:            pneumaEffectSpec(),
	effectHeavenDrive:       heavenDriveEffectSpec(),
	effectSonicBlow2:        sonicBlow2EffectSpec(),
	effectBrandishSpear2:    brandishSpear2EffectSpec(),
	effectShockwave:         shockwaveEffectSpec(),
	effectShockwaveHit:      shockwaveHitEffectSpec(),
	effectEarthHit:          earthHitEffectSpec(),
	effectPierceSelf:        pierceSelfEffectSpec(),
	effectBowlingSelf:       bowlingSelfEffectSpec(),
	effectSpearStabSelf:     spearStabSelfEffectSpec(),
	effectSpearBmrSelf:      spearBmrSelfEffectSpec(),
	effectHolyLight:         holyLightEffectSpec(),
	effectConcentration:     concentrationEffectSpec(),
	effectRefineOK:          refineOKEffectSpec(),
	effectRefineFail:        refineFailEffectSpec(),
	effectJobLevelUp:        jobLevelUpEffectSpec(),
	effectRain:              weatherRainEffectSpec(),
	effectSnow:              weatherSnowEffectSpec(),
	effectSakura:            sakuraEffectSpec(),
	effectBanjjakii:         banjjakiiEffectSpec(),
	effectMakeBlur:          makeBlurEffectSpec(),
	effectEnergyCoat:        energyCoatEffectSpec(),
	effectCartRevolution:    cartRevolutionEffectSpec(),
	effectVenomDust2:        venomDust2EffectSpec(),
	effectMentalBreak:       mentalBreakEffectSpec(),
	effectMagicalAtkHit:     magicalAtkHitEffectSpec(),
	effectSuiExplosion:      suiExplosionEffectSpec(),
	effectSuicide:           suicideEffectSpec(),
	effectComboAttack1:      comboAttack1EffectSpec(),
	effectComboAttack2:      comboAttack2EffectSpec(),
	effectComboAttack3:      comboAttack3EffectSpec(),
	effectComboAttack4:      comboAttack4EffectSpec(),
	effectComboAttack5:      comboAttack5EffectSpec(),
	effectGuidedAttack:      guidedAttackEffectSpec(),
	effectPoisonAttack2:     poisonAttack2EffectSpec(),
	effectSilenceAttack:     silenceAttackEffectSpec(),
	effectStunAttack:        stunAttackEffectSpec(),
	effectPetrifyAttack:     petrifyAttackEffectSpec(),
	effectSleepAttack:       sleepAttackEffectSpec(),
	effectPong:              pongEffectSpec(),
	effectLevel99:           level99EffectSpec(),
	effectLevel99Ground:     level99GroundEffectSpec(),
	effectLevel99Bubble:     level99BubbleEffectSpec(),
	effectGumgang:           gumgangEffectSpec(),
	effectPotionRed:         potionRedEffectSpec(),
	effectPotionOrange:      potionOrangeEffectSpec(),
	effectPotionYellow:      potionYellowEffectSpec(),
	effectPotionWhite:       potionWhiteEffectSpec(),
	effectPotionBlue:        bluePotionEffectSpec(),
	effectPotionGreen:       potionGreenEffectSpec(),
	effectFood:              foodEffectSpec(),
	effectFoodBlue:          foodBlueEffectSpec(),
	effectDarkBreath:        darkBreathEffectSpec(),
	effectDefender:          defenderEffectSpec(),
	effectKeeping:           keepingEffectSpec(),
	effectSummonSlave:       summonSlaveEffectSpec(),
	effectBloodDrain:        bloodDrainEffectSpec(),
	effectEnergyDrain:       energyDrainEffectSpec(),
	effectItemFast:          itemFastEffectSpec(),
	effectItemFast2:         itemFast2EffectSpec(),
	effectItemFast3:         berserkPotionEffectSpec(),
	effectCrusaderDef:       crusaderDefEffectSpec(),
	effectGanbantein:        ganbanteinEffectSpec(),
	effectVolcano:           volcanoEffectSpec(),
	effectGrandCross:        grandCrossEffectSpec(),
	effectIntimidate:        intimidateEffectSpec(),
	effectChookgi:           spiritSphereEffectSpec(),
	effectCloud:             cloudEffectSpec(),
	effectCloud2:            cloud2EffectSpec(),
	effectMapPillar:         mapPillarBlueEffectSpec(),
	effectLineLink:          devotionLinkEffectSpec(),
	effectCloud3:            cloud3EffectSpec(),
	effectSpellBreaker:      spellBreakerEffectSpec(),
	effectDispell:           dispellEffectSpec(),
	effectDeluge:            delugeEffectSpec(),
	effectViolentGale:       violentGaleEffectSpec(),
	effectLandProtector:     landProtectorEffectSpec(),
	effectBottomVolcano:     bottomVolcanoEffectSpec(),
	effectBottomDeluge:      bottomDelugeEffectSpec(),
	effectBottomViolent:     bottomViolentEffectSpec(),
	effectBottomLand:        landProtectorGroundEffectSpec(),
	effectMagicRod:          magicRodEffectSpec(),
	effectHolyCross:         holyCrossEffectSpec(),
	effectShieldCharge:      shieldChargeEffectSpec(),
	effectMapPillar2:        mapPillar2EffectSpec(),
	effectProvidence:        providenceEffectSpec(),
	effectShieldBoomer:      shieldBoomerangEffectSpec(),
	effectSpearQuicken:      spearQuickenEffectSpec(),
	effectDevotion:          devotionEffectSpec(),
	effectReflectShield:     reflectShieldEffectSpec(),
	effectAbsorbSpirits:     absorbSpiritsEffectSpec(),
	effectSteelBody:         steelBodyEffectSpec(),
	effectFlameLauncher:     flameLauncherEffectSpec(),
	effectFrostWeapon:       frostWeaponEffectSpec(),
	effectLightningLoad:     lightningLoadEffectSpec(),
	effectSeismicWeapon:     seismicWeaponEffectSpec(),
	effectMapPillar3:        mapPillar3EffectSpec(),
	effectMapPillar4:        mapPillar4EffectSpec(),
	effectGumgang2:          gumgang2EffectSpec(),
	effectTeiHit1:           teiHit1EffectSpec(),
	effectGumgang3:          gumgang3EffectSpec(),
	effectTeiHit2:           teiHit2EffectSpec(),
	effectTanji:             tanjiEffectSpec(),
	effectTeiHit1X:          teiHit1XEffectSpec(),
	effectChimto:            chimtoEffectSpec(),
	effectStealCoin:         stealCoinEffectSpec(),
	effectStripWeapon:       stripWeaponEffectSpec(),
	effectStripShield:       stripShieldEffectSpec(),
	effectStripArmor:        stripArmorEffectSpec(),
	effectStripHelm:         stripHelmEffectSpec(),
	effectChainCombo:        chainComboEffectSpec(),
	effectRogueCoin:         rogueCoinEffectSpec(),
	effectBackStab:          backStabEffectSpec(),
	effectTeiHit3:           teiHit3EffectSpec(),
	effectBottomDissonance:  bottomDissonanceEffectSpec(),
	effectBottomLullaby:     bottomLullabyEffectSpec(),
	effectBottomRichKim:     bottomRichKimEffectSpec(),
	effectBottomChaos:       bottomChaosEffectSpec(),
	effectBottomDrum:        bottomDrumEffectSpec(),
	effectBottomNibelung:    bottomNibelungEffectSpec(),
	effectBottomRoki:        songOutEffectSpec(),
	effectBottomAbyss:       bottomAbyssEffectSpec(),
	effectBottomSieg:        bottomSiegEffectSpec(),
	effectBottomWhistle:     bottomWhistleEffectSpec(),
	effectBottomSinX:        bottomSinXEffectSpec(),
	effectBottomBragi:       bottomBragiEffectSpec(),
	effectBottomApple:       bottomAppleEffectSpec(),
	effectBottomUglyDance:   bottomUglyDanceEffectSpec(),
	effectBottomHumming:     bottomHummingEffectSpec(),
	effectBottomForget:      bottomForgetEffectSpec(),
	effectBottomFortune:     bottomFortuneEffectSpec(),
	effectBottomService:     bottomServiceEffectSpec(),
	effectTalkFrostJoke:     talkFrostJokeEffectSpec(),
	effectTalkScream:        talkScreamEffectSpec(),
	effectPokJuk:            pokJukEffectSpec(),
	effectThrowItem:         acidTerrorBottleEffectSpec(),
	effectThrowItem2:        throwItem2EffectSpec(),
	effectChemicalProt:      chemicalProtectionEffectSpec(),
	effectDemonstration:     demonstrationEffectSpec(),
	effectChemical2:         chemical2EffectSpec(),
	effectTeleportation:     teleportationEffectSpec(),
	effectPharmacyOK:        pharmacyOKEffectSpec(),
	effectPharmacyFail:      pharmacyFailEffectSpec(),
	effectThrowItem3:        throwItem3EffectSpec(),
	effectFirstAid:          firstAidEffectSpec(),
	effectSprinkleSand:      sprinkleSandEffectSpec(),
	effectLoud:              loudEffectSpec(),
	effectHeal:              healEffectSpec(),
	effectHeal2:             heal2EffectSpec(),
	effectExit2:             exit2EffectSpec(),
	effectSafetyWall:        safetyWallEffectSpec(),
	effectReadyPortal:       readyPortalEffectSpec(),
	effectPortal:            portalEffectSpec(),
	effectBottomMagnus:      bottomMagnusEffectSpec(),
	effectBottomSanc:        bottomSancEffectSpec(),
	effectHealOffensive:     healOffensiveEffectSpec(),
	effectWarpZone2:         warpZone2EffectSpec(),
	effectHeal4:             heal4EffectSpec(),
	effectBeginAsura:        beginAsuraCastEffectSpec(),
	effectTripleAttack:      tripleAttackEffectSpec(),
	effectHitLine:           hitLineEffectSpec(),
	effectHPTime:            hpTimeEffectSpec(),
	effectSPTime:            spTimeEffectSpec(),
	effectMaple:             mapleEffectSpec(),
	effectBlind:             blindEffectSpec(),
	effectPoisonStatus:      poisonStatusEffectSpec(),
	effectGuard:             guardEffectSpec(),
	effectJobLvUp50:         jobLvUp50EffectSpec(),
	effectAngel2:            superAngelEffectSpec(),
	effectMagnum2:           magnum2EffectSpec(),
	effectCoupleCasting:     coupleCastingEffectSpec(),
	effectEntry2:            entry2EffectSpec(),
	effectColorPaper:        colorPaperEffectSpec(),
	effectSoulBreaker:       soulBreakerBaseEffectSpec(),
	effectLevel99Aura1:      level99BubbleEffectSpec(),
	effectFoodChocolate:     foodChocolateEffectSpec(),
	effectPressure:          pressureEffectSpec(),
	effectBash3D:            sacrificeEffectSpec(),
	effectAuraBlade:         auraBladeEffectSpec(),
	effectRedBody:           redBodyEffectSpec(),
	effectLKConcentration:   lkConcentrationEffectSpec(),
	effectBottomGospel:      bottomGospelEffectSpec(),
	effectBaseLevelUp:       baseLevelUpEffectSpec(),
	effectDeath:             deathEffectSpec(),
	effectDragonSmoke:       dragonSmokeEffectSpec(),
	effectBottomBasilica:    bottomBasilicaEffectSpec(),
	effectAssumptio:         assumptioEffectSpec(),
	effectHitLine2:          palmStrikeEffectSpec(),
	effectBash3D2:           tigerFistEffectSpec(),
	effectEnergyDrain2:      soulDrainEffectSpec(),
	effectTransBlueBody:     transBlueBodyEffectSpec(),
	effectMagicCrasher:      magicCrasherEffectSpec(),
	effectLightBlade:        lightBladeEffectSpec(),
	effectEnergyDrain3:      hpConversionEffectSpec(),
	effectLineLink2:         soulChangeLinkEffectSpec(),
	effectLinkLight:         linkLightEffectSpec(),
	effectTrueSight:         trueSightEffectSpec(),
	effectFalconAssault:     falconAssaultEffectSpec(),
	effectTripleAttack2:     sharpShootingEffectSpec(),
	effectPortal4:           windWalkEffectSpec(),
	effectMeltdown:          meltdownEffectSpec(),
	effectCartBoost:         cartBoostEffectSpec(),
	effectRejectSword:       rejectSwordEffectSpec(),
	effectTripleAttack3:     arrowVulcanEffectSpec(),
	effectMoonlit:           moonlitEffectSpec(),
	effectMarionetteLink:    marionetteEffectSpec(),
	effectPinkBody:          pinkBodyEffectSpec(),
	effectLevel99AuraMid:    level99EffectSpec(),
	effectLevel99AuraBottom: level99GroundEffectSpec(),
	effectBash3D3:           bash3D3EffectSpec(),
	effectBash3D4:           bash3D4EffectSpec(),
	effectNapalmVulcan:      napalmVulcanEffectSpec(),
	effectPortal5:           portal5EffectSpec(),
	effectMagicCrasher2:     magicCrasher2EffectSpec(),
	effectBottomSpider:      spiderWebEffectSpec(),
	effectFogWallGround:     fogWallGroundEffectSpec(),
	effectSoulBurn:          soulBurnEffectSpec(),
	effectSoulChange:        soulChangeEffectSpec(),
	effectSoulBreaker2:      soulBreaker2EffectSpec(),
	effectRainbow:           rainbowEffectSpec(),
	effectPeong:             taekwonJumpEffectSpec(),
	effectPressedBody:       pressedBodyEffectSpec(),
	effectSpinedBody:        spinedBodyEffectSpec(),
	effectKickedBody:        kickedBodyEffectSpec(),
	effectDoubleGumgang:     starWarmthEffectSpec(),
	effectReflectBody:       reflectBodyEffectSpec(),
	effectBabyBody:          babyBodyEffectSpec(),
	effectBabyBody2:         babyBody2EffectSpec(),
	effectGiantBody:         giantBodyEffectSpec(),
	effectGiantBody2:        giantBody2EffectSpec(),
	effectAsuraBody:         asuraBodyEffectSpec(),
	effectFourWayBody:       fourWayBodyEffectSpec(),
	effectQuakeBody:         quakeBodyEffectSpec(),
	effectAsuraBodyMonster:  asuraBodyMonsterEffectSpec(),
	effectHitLine4:          hitLineBurstEffectSpec(),
	effectHitLine5:          hitLineBurstEffectSpec(),
	effectHitLine6:          hitLineBurstEffectSpec(),
	effectStormKick:         stormKickEffectSpec(),
	effectChemical3:         taekwonDashEffectSpec(),
	effectAssumptio2:        assumptio2EffectSpec(),
	effectBlueCasting:       blueCastingEffectSpec(),
	effectRun:               runEffectSpec(),
	effectStopRun:           stopRunEffectSpec(),
	effectStopEffect:        taekwonStopEffectSpec(),
	effectJumpBody:          jumpBodyEffectSpec(),
	effectLandBody:          landBodyEffectSpec(),
	effectFoot3:             foot3EffectSpec(),
	effectFoot4:             foot4EffectSpec(),
	effectTaeReady:          taeReadyEffectSpec(),
	effectDarkGrandCross:    darkGrandCrossEffectSpec(),
	effectDarkSoulStrike:    darkSoulStrikeEffectSpec(),
	effectDarkJupitelHit:    darkJupitelHitEffectSpec(),
	effectNPCStop:           npcStopEffectSpec(),
	effectDarkCasting:       darkCastingEffectSpec(),
	effectAgiUp:             gunslingerAgilityEffectSpec(),
	effectJumpKick:          jumpKickEffectSpec(),
	effectQuakeBody2:        quakeBody2EffectSpec(),
	effectStormKick3:        stormKick3EffectSpec(),
	effectStormKick4:        kaupeDodgeEffectSpec(),
	effectSpinedBody2:       spinedBody2EffectSpec(),
	effectBeginAsura1:       beginAsura1EffectSpec(),
	effectBeginAsura2:       beginAsura2EffectSpec(),
	effectBeginAsura3:       beginAsura3EffectSpec(),
	effectBeginAsura4:       beginAsura4EffectSpec(),
	effectBeginAsura5:       beginAsura5EffectSpec(),
	effectBeginAsura6:       beginAsura6EffectSpec(),
	effectBeginAsura7:       beginAsura7EffectSpec(),
	effectDevil1:            starBlindEffectSpec(),
	effectDevil2:            starBlindEffectSpec(),
	effectDevil3:            starBlindEffectSpec(),
	effectDevil4:            starBlindEffectSpec(),
	effectDevil5:            starBlindEffectSpec(),
	effectDevil6:            starBlindEffectSpec(),
	effectDevil7:            starBlindEffectSpec(),
	effectDevil8:            starBlindEffectSpec(),
	effectDevil9:            starBlindEffectSpec(),
	effectDevil10:           starBlindEffectSpec(),
	effectBlackDevil:        starBlackDevilEffectSpec(),
	effectFlowerCast:        starComfortEffectSpec(),
	effectMochi:             mochiEffectSpec(),
	effectRamadan:           ramadanEffectSpec(),
	effectEDP:               enchantDeadlyPoisonEffectSpec(),
	effectFullStrip:         fullStripEffectSpec(),
	effectPreserve:          preserveEffectSpec(),
	effectSlim:              slimEffectSpec(),
	effectSlim2:             slim2EffectSpec(),
	effectSlim3:             slim3EffectSpec(),
	effectChemicalBody:      chemicalBodyEffectSpec(),
	effectCastSpin:          castSpinEffectSpec(),
	effectPierceBody:        pierceBodyEffectSpec(),
	effectSoulLink:          soulLinkEffectSpec(),
	effectChookgi2:          spiritSphereEffectSpec(),
	effectMemorize:          memorizeEffectSpec(),
	effectMapae:             mapaeEffectSpec(),
	effectItemPokJuk:        itemPokJukEffectSpec(),
	effectValentine05:       valentine05EffectSpec(),
	effectBeginAsura11:      beginAsura11EffectSpec(),
	effectChemical2Dash:     chainCrushEffectSpec(),
	effectGroundSample:      groundSampleEffectSpec(),
	effectCloud4:            einbrochWeatherCloudEffectSpec(),
	effectCloud5:            cloud5EffectSpec(),
	effectBottomHermode:     hermodeEffectSpec(),
	effectCartTermination:   cartTerminationEffectSpec(),
	effectItemFastDown:      itemFastDownEffectSpec(),
	effectShieldBoomerang3:  shieldChainEffectSpec(),
	effectDoubleCastBody:    doubleCastBodyEffectSpec(),
	effectGravitation:       gravitationEffectSpec(),
	effectTarotCard1:        tarotCard1EffectSpec(),
	effectTarotCard2:        tarotCard2EffectSpec(),
	effectTarotCard3:        tarotCard3EffectSpec(),
	effectTarotCard4:        tarotCard4EffectSpec(),
	effectTarotCard5:        tarotCard5EffectSpec(),
	effectTarotCard6:        tarotCard6EffectSpec(),
	effectTarotCard7:        tarotCard7EffectSpec(),
	effectTarotCard8:        tarotCard8EffectSpec(),
	effectTarotCard9:        tarotCard9EffectSpec(),
	effectTarotCard10:       tarotCard10EffectSpec(),
	effectTarotCard11:       tarotCard11EffectSpec(),
	effectTarotCard12:       tarotCard12EffectSpec(),
	effectTarotCard13:       tarotCard13EffectSpec(),
	effectTarotCard14:       tarotCard14EffectSpec(),
	effectAcidDemon:         acidDemonEffectSpec(),
	effectThrowItem4:        acidBottlesEffectSpec(),
	effectBabyBodyBack:      babyBodyBackEffectSpec(),
	effectThrowItem5:        throwItem5EffectSpec(),
	effectBlueBody:          blueBodyEffectSpec(),
	effectHated:             hatedEffectSpec(),
	effectRedLightBody:      redLightBodyEffectSpec(),
	effectSmaReady:          esmaReadyEffectSpec(),
	effectStin:              estinEffectSpec(),
	effectRedHit:            redHitEffectSpec(),
	effectBlueHit:           blueHitEffectSpec(),
	effectQuakeBody3:        quakeBody3EffectSpec(),
	effectStin2:             esmaEffectSpec(),
	effectHitTexture:        esmaHitTextureEffectSpec(),
	effectStin3:             estunEffectSpec(),
	effectScreenQuake:       screenQuakeEffectSpec(),
	effectHfliMoon1:         hfliMoon1EffectSpec(),
	effectHfliMoon2:         hfliMoon2EffectSpec(),
	effectHfliMoon3:         hfliMoon3EffectSpec(),
	effectHoUp:              hoUpEffectSpec(),
	effectHamiDefence:       hamiDefenceEffectSpec(),
	effectHamiCastle:        hamiCastleEffectSpec(),
	effectHamiBlood:         hamiBloodEffectSpec(),
	effectHated2:            starWarmthParticlesEffectSpec(),
	effectTwilight1:         twilight1EffectSpec(),
	effectTwilight2:         twilight2EffectSpec(),
	effectTwilight3:         twilight3EffectSpec(),
	effectItemThunder:       itemThunderEffectSpec(),
	effectItemCloud:         itemCloudEffectSpec(),
	effectItemCurse:         itemCurseEffectSpec(),
	effectItemZZZ:           itemZZZEffectSpec(),
	effectItemRain:          itemRainEffectSpec(),
	effectM01:               m01EffectSpec(),
	effectM02:               m02EffectSpec(),
	effectM03:               m03EffectSpec(),
	effectM04:               m04EffectSpec(),
	effectM05:               m05EffectSpec(),
	effectM06:               m06EffectSpec(),
	effectM07:               m07EffectSpec(),
	effectKaizel:            kaizelEffectSpec(),
	effectCloud6:            cloud6EffectSpec(),
	effectStatFoodSTR:       statFoodSTREffectSpec(),
	effectStatFoodINT:       statFoodINTEffectSpec(),
	effectStatFoodVIT:       statFoodVITEffectSpec(),
	effectStatFoodAGI:       statFoodAGIEffectSpec(),
	effectStatFoodDEX:       statFoodDEXEffectSpec(),
	effectStatFoodLUK:       statFoodLUKEffectSpec(),
	effectShrink:            shrinkEffectSpec(),
	effectThrowItem6:        throwItem6EffectSpec(),
	effectSightBlaster:      sightBlasterEffectSpec(),
	effectQuakeBody4:        closeConfineHitEffectSpec(),
	effectFireHit2:          fireHit2EffectSpec(),
	effectNPCStop2:          closeConfineRingEffectSpec(),
	effectNPCStop2Del:       npcStop2DelEffectSpec(),
	effectFVoice:            pangVoiceEffectSpec(),
	effectWink:              winkEffectSpec(),
	effectCookingOK:         cookingOKEffectSpec(),
	effectCookingFail:       cookingFailEffectSpec(),
	effectHapgyeok:          hapgyeokEffectSpec(),
	effectThrowItem7:        throwItem7EffectSpec(),
	effectThrowItem8:        throwItem8EffectSpec(),
	effectThrowItem9:        throwItem9EffectSpec(),
	effectThrowItem10:       throwItem10EffectSpec(),
	effectKouenka:           ninjaFirePetalEffectSpec(),
	effectHyousensou:        ninjaIceSpearEffectSpec(),
	effectBottomSuiton:      bottomSuitonEffectSpec(),
	effectStin4:             stin4EffectSpec(),
	effectThunderStorm2:     thunderStorm2EffectSpec(),
	effectChemical4:         piercingShotEffectSpec(),
	effectStin5:             stin5EffectSpec(),
	effectMadnessBlue:       madnessBlueEffectSpec(),
	effectMadnessRed:        madnessRedEffectSpec(),
	effectRGCoin3:           disarmEffectSpec(),
	effectBash3D5:           bash3D5EffectSpec(),
	effectChookgi3:          spiritSphereEffectSpec(),
	effectKirikage:          kirikageEffectSpec(),
	effectTatami:            tatamiEffectSpec(),
	effectKasumikiri:        kasumikiriEffectSpec(),
	effectIssen:             issenEffectSpec(),
	effectKaen:              kaenEffectSpec(),
	effectBaku:              bakuEffectSpec(),
	effectHyousyouraku:      hyousyourakuEffectSpec(),
	effectDesperado:         desperadoEffectSpec(),
	effectLightningS:        lightningSEffectSpec(),
	effectBlindS:            blindSEffectSpec(),
	effectPoisonS:           poisonSEffectSpec(),
	effectFreezingS:         freezingSEffectSpec(),
	effectFlareS:            flareSEffectSpec(),
	effectRapidShower:       rapidShowerEffectSpec(),
	effectMagicalBullet:     magicalBulletEffectSpec(),
	effectSpreadAttack:      spreadAttackEffectSpec(),
	effectTrackCasting:      trackCastingEffectSpec(),
	effectTracking:          trackingEffectSpec(),
	effectTripleAction:      tripleActionEffectSpec(),
	effectBullseye:          bullseyeEffectSpec(),
	effectBubbleDrop:        bubbleDropEffectSpec(),
	effectNPCEarthquake:     npcEarthquakeEffectSpec(),
	effectDragonFear:        dragonFearEffectSpec(),
	effectWideBleeding:      wideBleedingEffectSpec(),
	effectWideConfuse:       wideConfuseEffectSpec(),
	effectBottomRunner:      bottomRunnerEffectSpec(),
	effectBottomTransfer:    bottomTransferEffectSpec(),
	effectBottomEvilLand:    evilLandEffectSpec(),
	effectGuard3:            guard3EffectSpec(),
	effectCriticalWound:     criticalWoundEffectSpec(),
	effectFirecracker2:      firecracker2EffectSpec(),
	effectFirecracker3:      firecracker3EffectSpec(),
	effectFirecracker4:      firecracker4EffectSpec(),
	effectFirecracker5:      firecracker5EffectSpec(),
	effectFirecracker6:      firecracker6EffectSpec(),
	effectTorchRed:          torchRedEffectSpec(),
	effectTorchGreen:        torchGreenEffectSpec(),
	effectMapGhost:          mapGhostEffectSpec(),
	effectGlow1:             glow1EffectSpec(),
	effectGlow2:             glow2EffectSpec(),
	effectGlow4:             glow4EffectSpec(),
	effectTorchPurple:       torchPurpleEffectSpec(),
	effectCloud7:            cloud7EffectSpec(),
	effectCloud8:            cloud8EffectSpec(),
	effectFlowerLeaf:        flowerLeafEffectSpec(),
	effectItem315:           item315EffectSpec(),
	effectItem316:           item316EffectSpec(),
	effectItem317:           item317EffectSpec(),
	effectStormMin:          stormMinEffectSpec(),
	effectFirecracker7:      firecracker7EffectSpec(),
	effectBottomBlue:        bottomBlueEffectSpec(),
	effectBottomBlue2:       bottomBlueEffectSpec(),
	effectChristmasCarol:    christmasCarolEffectSpec(),
	effectFirePillarOn2:     judexEffectSpec(),
	effectForestLight5:      forestLight5EffectSpec(),
	effectAdoramus:          adoramusEffectSpec(),
	effectIgnitionBreak:     ignitionBreakEffectSpec(),
	effectFrostMisty:        frostMistyEffectSpec(),
	effectCrimsonRock:       crimsonRockEffectSpec(),
	effectHellInferno:       hellInfernoEffectSpec(),
	effectMarshOfAbyss:      marshOfAbyssEffectSpec(),
	effectDragonHowling:     dragonHowlingEffectSpec(),
	effectEarthWall:         earthWallEffectSpec(),
	effectChainLightning:    chainLightningEffectSpec(),
	effectAimedBolt:         aimedBoltEffectSpec(),
	effectArrowStorm:        arrowStormEffectSpec(),
	effectLaulamus:          laulamusEffectSpec(),
	effectLauagnus:          lauagnusEffectSpec(),
	effectMillenniumShield:  millenniumShieldEffectSpec(),
	effectConcentration2:    concentration2EffectSpec(),
	effectGlassWall3:        glassWall3EffectSpec(),
	effectBerserkPotion2:    berserkPotion2EffectSpec(),
	effectRolling1:          rolling1EffectSpec(),
	effectRolling2:          rolling2EffectSpec(),
	effectRolling3:          rolling3EffectSpec(),
	effectRolling4:          rolling4EffectSpec(),
	effectRolling5:          rolling5EffectSpec(),
	effectRolling6:          rolling6EffectSpec(),
	effectRolling7:          rolling7EffectSpec(),
	effectRolling8:          rolling8EffectSpec(),
	effectRolling9:          rolling9EffectSpec(),
	effectRolling10:         rolling10EffectSpec(),
	effectCastSpin2:         castSpin2EffectSpec(),
	effectCrashAxe:          crashAxeEffectSpec(),
	effectStasis:            stasisEffectSpec(),
	effectBottomBasilica2:   bottomBasilica2EffectSpec(),
	effectRecognized:        recognizedEffectSpec(),
	effectTetra:             tetraEffectSpec(),
	effectTetraCasting:      tetraCastingEffectSpec(),
	effectStretch:           stretchEffectSpec(),
	effectEnervation:        enervationEffectSpec(),
	effectEnervation2:       enervation2EffectSpec(),
	effectEnervation3:       enervation3EffectSpec(),
	effectEnervation4:       enervation4EffectSpec(),
	effectEnervation5:       enervation5EffectSpec(),
	effectEnervation6:       enervation6EffectSpec(),
	effectBottomManhole:     bottomManholeEffectSpec(),
	effectManhole:           manholeEffectSpec(),
	effectForestLight6:      forestLight6EffectSpec(),
	effectBottomAni:         bottomAniEffectSpec(),
	effectBottomMaelstrom:   bottomMaelstromEffectSpec(),
	effectBottomBloodyLust:  bottomBloodyLustEffectSpec(),
	effectHealN:             healNEffectSpec(),
	effectChookgiN:          spiritSphereEffectSpec(),
	effectDance1:            dance1EffectSpec(),
	effectBotReverb:         botReverbEffectSpec(),
	effectRainParticle:      rainParticleEffectSpec(),
	effectChemicalV2:        chemicalV2EffectSpec(),
	effectBotReverb2:        botReverb2EffectSpec(),
	effectCirclePower2:      circlePower2EffectSpec(),
	effectSecra2:            secra2EffectSpec(),
	effectSprPlant2:         sprPlant2EffectSpec(),
	effectSprPlant3:         sprPlant3EffectSpec(),
	effectSprPlant4:         sprPlant4EffectSpec(),
	effectSprPlant5:         sprPlant5EffectSpec(),
	effectSprPlant6:         sprPlant6EffectSpec(),
	effectSprPlant7:         sprPlant7EffectSpec(),
	effectSprPlant8:         sprPlant8EffectSpec(),
	effectHeartAsura:        heartAsuraEffectSpec(),
	effectGlassWall4:        glassWall4EffectSpec(),
	effectBash3D6:           bash3D6EffectSpec(),
	effectElectric4:         electric4EffectSpec(),
	effectTeiHit1T:          teiHit1TEffectSpec(),
	effectPressure2:         pressure2EffectSpec(),
	effectPrimeCharge2:      primeCharge2EffectSpec(),
	effectPrimeCharge3:      primeCharge3EffectSpec(),
	effectPrimeCharge4:      primeCharge4EffectSpec(),
	effectFireWall2:         fireWall2EffectSpec(),
	effectSprPlant10:        sprPlant10EffectSpec(),
	effectShockwave2:        shockwave2EffectSpec(),
	effectColdThrow2:        coldThrow2EffectSpec(),
	effectDemonicFire4:      demonicFire4EffectSpec(),
	effectPressure3:         pressure3EffectSpec(),
	effectPoisonMist:        poisonMistEffectSpec(),
	effectEraserCutter:      eraserCutterEffectSpec(),
	effectLavaSlide:         lavaSlideEffectSpec(),
	effectSonicClaw:         sonicClawEffectSpec(),
	effectTinderBreaker:     tinderBreakerEffectSpec(),
	effectMidnightFrenzy:    midnightFrenzyEffectSpec(),
	effectVolcanicAsh:       volcanicAshEffectSpec(),
	effectRWC2011:           rwc2011EffectSpec(),
	effectRWC2011Two:        rwc2011TwoEffectSpec(),
	effectRunMakeOK:         runMakeOKEffectSpec(),
	effectRunMakeFailure:    runMakeFailureEffectSpec(),
	effectMIResultMakeOK:    miResultMakeOKEffectSpec(),
	effectMIResultMakeFail:  miResultMakeFailEffectSpec(),
	effectAllRayProtect:     allRayProtectEffectSpec(),
	effectVenomFog:          venomFogEffectSpec(),
	effectDustStorm:         dustStormEffectSpec(),
	effectDanceBladeAtk:     danceBladeAtkEffectSpec(),
	effectInvincibleOff2:    invincibleOff2EffectSpec(),
	effectDeathSummon:       deathSummonEffectSpec(),
	effectGCDarkCrow:        gcDarkCrowEffectSpec(),
	effectAllFullThrottle:   allFullThrottleEffectSpec(),
	effectSRFlashCombo:      srFlashComboEffectSpec(),
	effectRKLuxAnima:        rkLuxAnimaEffectSpec(),
	effectSOElemShield:      soElemShieldEffectSpec(),
	effectABOffertorium:     abOffertoriumEffectSpec(),
	effectWLTelekinesis:     wlTelekinesisEffectSpec(),
	effectGNIllusionDoping:  gnIllusionDopingEffectSpec(),
	effectNCMagmaEruption:   ncMagmaEruptionEffectSpec(),
	effectNPCChill:          npcChillEffectSpec(),
	effectOffertoriumRing:   offertoriumRingEffectSpec(),
	effectHammerOfGod:       hammerOfGodEffectSpec(),
	effectAchComplete:       achCompleteEffectSpec(),
	effectBodyColor:         bodyColorEffectSpec(),
	effectBakuretsuHadou:    bakuretsuHadouEffectSpec(),
	dropEffectPink:          dropPinkEffectSpec(),
	dropEffectYellow:        dropYellowEffectSpec(),
	dropEffectPurple:        dropPurpleEffectSpec(),
	effectDigitalSpace:      digitalSpaceEffectSpec(),
	dropEffectBlue:          dropBlueEffectSpec(),
	dropEffectGreen:         dropGreenEffectSpec(),
	dropEffectRed:           dropRedEffectSpec(),
	effectNewSuccess:        newSuccessEffectSpec(),
	effectNewFailure:        newFailureEffectSpec(),
	effectNewIntro:          newIntroEffectSpec(),
	effectEnchantYellow:     enchantYellowEffectSpec(),
	effectEnchantSuccess:    enchantSuccessEffectSpec(),
	effectEnchantFail:       enchantFailEffectSpec(),
	effectEnchantBlue:       enchantBlueEffectSpec(),
	effectEnchantUpSuccess:  enchantUpSuccessEffectSpec(),
	effectEnchantUpFail:     enchantUpFailEffectSpec(),
	effectEnchantGreen:      enchantGreenEffectSpec(),
	effectEnchantResetOK:    enchantResetOKEffectSpec(),
	effectEnchantResetFail:  enchantResetFailEffectSpec(),

	effectColdBolt:               coldBoltEffectSpec(),
	effectFireBolt:               fireBoltEffectSpec(),
	effectCastRing:               castRingEffectSpec(),
	effectQuakeMagnum:            quakeMagnumEffectSpec(),
	effectArrowShot:              arrowShotEffectSpec(),
	effectArrowShower:            arrowShowerEffectSpec(),
	effectMagicPower:             magicPowerEffectSpec(),
	effectWhitePulse:             whitePulseEffectSpec(),
	effectSpearHitSound:          spearHitSoundEffectSpec(),
	effectEnemyHitNormal1:        enemyHitNormal1EffectSpec(),
	effectQuake:                  quakeEffectSpec(),
	effectSharpShootingCast:      sharpShootingCastEffectSpec(),
	effectAdrenalineCast:         adrenalineCastEffectSpec(),
	effectMaximizeSounds:         maximizeSoundsEffectSpec(),
	effectGreedSound:             greedSoundEffectSpec(),
	effectGospelGround:           gospelGroundEffectSpec(),
	effectHermodeMusic:           hermodeMusicEffectSpec(),
	effectBottomDissonanceGround: bottomDissonanceEffectSpec(),
	effectBottomLullabyGround:    bottomLullabyEffectSpec(),
	effectBottomRichKimGround:    bottomRichKimEffectSpec(),
	effectBottomChaosGround:      bottomChaosEffectSpec(),
	effectBottomDrumGround:       bottomDrumEffectSpec(),
	effectBottomNibelungGround:   bottomNibelungEffectSpec(),
	effectBottomRokiGround:       songOutEffectSpec(),
	effectBottomAbyssGround:      bottomAbyssEffectSpec(),
	effectBottomSiegGround:       bottomSiegEffectSpec(),
	effectBottomWhistleGround:    bottomWhistleEffectSpec(),
	effectBottomSinXGround:       bottomSinXEffectSpec(),
	effectBottomBragiGround:      bottomBragiEffectSpec(),
	effectBottomAppleGround:      bottomAppleEffectSpec(),
	effectBottomUglyDanceGround:  bottomUglyDanceEffectSpec(),
	effectBottomHummingGround:    bottomHummingEffectSpec(),
	effectBottomForgetGround:     bottomForgetEffectSpec(),
	effectBottomFortuneGround:    bottomFortuneEffectSpec(),
	effectBottomServiceGround:    bottomServiceEffectSpec(),
	effectGunslingerCoinSound:    gunslingerCoinSoundEffectSpec(),
	effectSteelBodyRing:          steelBodyRingEffectSpec(),
	SkillEffectChaseWalkSmoke:    chaseWalkSmokeEffectSpec(),
}
