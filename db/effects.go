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
	EffectHit1              = 0
	EffectBashHit           = 1
	EffectHit3              = 2
	EffectHit4              = 3
	EffectHit5              = 4
	EffectHit6              = 5
	EffectEntry             = 6
	EffectExit              = 7
	EffectWarp              = 8
	EffectEnhance           = 9
	EffectMammonite         = 10
	EffectEndure            = 11
	EffectBeginSpell        = 12
	EffectGlassWall         = 13
	EffectHealSP            = 14
	EffectSoulStrike        = 15
	EffectBashBegin         = 16
	EffectMagnumBreak       = 17
	EffectSteal             = 18
	EffectPoisonAttack      = 20
	EffectDetoxication      = 21
	EffectSight             = 22
	EffectStoneCurse        = 23
	EffectFireBall          = 24
	EffectFireWall          = 25
	EffectIceArrow          = 26
	EffectFrostDiver        = 27
	EffectFrostDiverHit     = 28
	EffectLightningBolt     = 29
	EffectThunderStorm      = 30
	EffectFireArrow         = 31
	EffectNapalmBeat        = 32
	EffectRuwach            = 33
	EffectTeleportOld       = 34
	EffectReadyPortalOld    = 35
	EffectIncAgility        = 37
	EffectDecAgility        = 38
	EffectAqua              = 39
	EffectSignum            = 40
	EffectAngelus           = 41
	EffectBlessing          = 42
	EffectIncAgiDex         = 43
	EffectSmoke             = 44
	EffectFirefly           = 45
	EffectTorch             = 47
	EffectFireHit           = 49
	EffectFireSplashHit     = 50
	EffectColdHit           = 51
	EffectWindHit           = 52
	EffectPoisonHit         = 53
	EffectBeginSpell2       = 54
	EffectBeginSpell3       = 55
	EffectBeginSpell4       = 56
	EffectBeginSpell5       = 57
	EffectBeginSpell6       = 58
	EffectBeginSpell7       = 59
	EffectLockOnTarget      = 60
	EffectWarpZone          = 61
	EffectSightTrasher      = 62
	EffectArrowShotRO       = 64
	EffectInvenom           = 65
	EffectCure              = 66
	EffectProvoke           = 67
	EffectMvp               = 68
	EffectSkidTrap          = 69
	EffectBrandishSpear     = 70
	EffectBowlingBash       = 73
	EffectIceWall           = 74
	EffectGloria            = 75
	EffectMagnificat        = 76
	EffectResurrection      = 77
	EffectRecovery          = 78
	EffectEarthSpike        = 79
	EffectSpearBoomerang    = 80
	EffectPierce            = 81
	EffectTurnUndead        = 82
	EffectSanctuary         = 83
	EffectImpositio         = 84
	EffectLexAeterna        = 85
	EffectAspersio          = 86
	EffectLexDivina         = 87
	EffectSuffragium        = 88
	EffectStormGust         = 89
	EffectLordVermilion     = 90
	EffectBenedictio        = 91
	EffectMeteorStorm       = 92
	EffectJupitelThunder    = 93
	EffectJupitelHit        = 94
	EffectQuagmire          = 95
	EffectFirePillar        = 96
	EffectFirePillarBomb    = 97
	EffectHasteUp           = 98
	EffectFlasher           = 99
	EffectRemoveTrap        = 100
	EffectRepairWeapon      = 101
	EffectCrashEarth        = 102
	EffectWeaponPerfect     = 103
	EffectMaximizePower     = 104
	EffectBlastMine         = 105
	EffectBlastMineBomb     = 106
	EffectClaymore          = 107
	EffectFreezingTrap      = 108
	EffectBubble            = 109
	EffectGasPush           = 110
	EffectSpringTrap        = 111
	EffectKyrie             = 112
	EffectMagnus            = 113
	EffectBlitzBeat         = 115
	EffectWaterBall         = 116
	EffectWaterBall2        = 117
	EffectDetecting         = 119
	EffectCloaking          = 120
	EffectSonicBlow         = 121
	EffectSonicBlowHit      = 122
	EffectGrimtooth         = 123
	EffectVenomDust         = 124
	EffectEnchantPoison     = 125
	EffectPoisonReact       = 126
	EffectPoisonReact2      = 127
	EffectOverthrust        = 128
	EffectVenomSplasher     = 129
	EffectTwoHandQuicken    = 130
	EffectAutoCounter       = 131
	EffectGrimtoothAtk      = 132
	EffectFreeze            = 133
	EffectFreezed           = 134
	EffectIceCrash          = 135
	EffectSlowPoison        = 136
	EffectFirePillarOn      = 138
	EffectSandman           = 139
	EffectRevive            = 140
	EffectPneuma            = 141
	EffectHeavenDrive       = 142
	EffectSonicBlow2        = 143
	EffectBrandishSpear2    = 144
	EffectShockwave         = 145
	EffectShockwaveHit      = 146
	EffectEarthHit          = 147
	EffectPierceSelf        = 148
	EffectBowlingSelf       = 149
	EffectSpearStabSelf     = 150
	EffectSpearBmrSelf      = 151
	EffectHolyLight         = 152
	EffectConcentration     = 153
	EffectRefineOK          = 154
	EffectRefineFail        = 155
	EffectJobLevelUp        = 158
	EffectRain              = 161
	EffectSnow              = 162
	EffectSakura            = 163
	EffectBanjjakii         = 165
	EffectMakeBlur          = 166
	EffectEnergyCoat        = 169
	EffectCartRevolution    = 170
	EffectVenomDust2        = 171
	EffectMentalBreak       = 181
	EffectMagicalAtkHit     = 182
	EffectSuiExplosion      = 183
	EffectSuicide           = 185
	EffectComboAttack1      = 186
	EffectComboAttack2      = 187
	EffectComboAttack3      = 188
	EffectComboAttack4      = 189
	EffectComboAttack5      = 190
	EffectGuidedAttack      = 191
	EffectPoisonAttack2     = 192
	EffectSilenceAttack     = 193
	EffectStunAttack        = 194
	EffectPetrifyAttack     = 195
	EffectSleepAttack       = 197
	EffectPong              = 199
	EffectLevel99           = 200
	EffectLevel99Ground     = 201
	EffectLevel99Bubble     = 202
	EffectGumgang           = 203
	EffectPotionRed         = 204
	EffectPotionOrange      = 205
	EffectPotionYellow      = 206
	EffectPotionWhite       = 207
	EffectPotionBlue        = 208
	EffectPotionGreen       = 209
	EffectFood              = 210
	EffectFoodBlue          = 211
	EffectDarkBreath        = 212
	EffectDefender          = 213
	EffectKeeping           = 214
	EffectSummonSlave       = 215
	EffectBloodDrain        = 216
	EffectEnergyDrain       = 217
	EffectItemFast          = 218
	EffectItemFast2         = 219
	EffectItemFast3         = 220
	EffectCrusaderDef       = 222
	EffectGanbantein        = 223
	EffectVolcano           = 225
	EffectGrandCross        = 226
	EffectIntimidate        = 227
	EffectChookgi           = 228
	EffectCloud             = 229
	EffectCloud2            = 230
	EffectMapPillar         = 231
	EffectLineLink          = 232
	EffectCloud3            = 233
	EffectSpellBreaker      = 234
	EffectDispell           = 235
	EffectDeluge            = 236
	EffectViolentGale       = 237
	EffectLandProtector     = 238
	EffectBottomVolcano     = 239
	EffectBottomDeluge      = 240
	EffectBottomViolent     = 241
	EffectBottomLand        = 242
	EffectMagicRod          = 244
	EffectHolyCross         = 245
	EffectShieldCharge      = 246
	EffectMapPillar2        = 247
	EffectProvidence        = 248
	EffectShieldBoomer      = 249
	EffectSpearQuicken      = 250
	EffectDevotion          = 251
	EffectReflectShield     = 252
	EffectAbsorbSpirits     = 253
	EffectSteelBody         = 254
	EffectFlameLauncher     = 255
	EffectFrostWeapon       = 256
	EffectLightningLoad     = 257
	EffectSeismicWeapon     = 258
	EffectMapPillar3        = 259
	EffectMapPillar4        = 260
	EffectGumgang2          = 261
	EffectTeiHit1           = 262
	EffectGumgang3          = 263
	EffectTeiHit2           = 264
	EffectTanji             = 265
	EffectTeiHit1X          = 266
	EffectChimto            = 267
	EffectStealCoin         = 268
	EffectStripWeapon       = 269
	EffectStripShield       = 270
	EffectStripArmor        = 271
	EffectStripHelm         = 272
	EffectChainCombo        = 273
	EffectRogueCoin         = 274
	EffectBackStab          = 275
	EffectTeiHit3           = 276
	EffectBottomDissonance  = 277
	EffectBottomLullaby     = 278
	EffectBottomRichKim     = 279
	EffectBottomChaos       = 280
	EffectBottomDrum        = 281
	EffectBottomNibelung    = 282
	EffectBottomRoki        = 283
	EffectBottomAbyss       = 284
	EffectBottomSieg        = 285
	EffectBottomWhistle     = 286
	EffectBottomSinX        = 287
	EffectBottomBragi       = 288
	EffectBottomApple       = 289
	EffectBottomUglyDance   = 290
	EffectBottomHumming     = 291
	EffectBottomForget      = 292
	EffectBottomFortune     = 293
	EffectBottomService     = 294
	EffectTalkFrostJoke     = 295
	EffectTalkScream        = 296
	EffectPokJuk            = 297
	EffectThrowItem         = 298
	EffectThrowItem2        = 299
	EffectChemicalProt      = 300
	EffectDemonstration     = 302
	EffectChemical2         = 303
	EffectTeleportation     = 304
	EffectPharmacyOK        = 305
	EffectPharmacyFail      = 306
	EffectThrowItem3        = 308
	EffectFirstAid          = 309
	EffectSprinkleSand      = 310
	EffectLoud              = 311
	EffectHeal              = 312
	EffectHeal2             = 313
	EffectExit2             = 314
	EffectSafetyWall        = 315
	EffectReadyPortal       = 316
	EffectPortal            = 317
	EffectBottomMagnus      = 318
	EffectBottomSanc        = 319
	EffectHealOffensive     = 320
	EffectWarpZone2         = 321
	EffectHeal4             = 325
	EffectBeginAsura        = 328
	EffectTripleAttack      = 329
	EffectHitLine           = 330
	EffectHPTime            = 331
	EffectSPTime            = 332
	EffectMaple             = 333
	EffectBlind             = 334
	EffectPoisonStatus      = 335
	EffectGuard             = 336
	EffectJobLvUp50         = 337
	EffectAngel2            = 338
	EffectMagnum2           = 339
	EffectCoupleCasting     = 342
	EffectEntry2            = 344
	EffectColorPaper        = 347
	EffectSoulBreaker       = 361
	EffectLevel99Aura1      = 362
	EffectFoodChocolate     = 363
	EffectPressure          = 365
	EffectBash3D            = 366
	EffectAuraBlade         = 367
	EffectRedBody           = 368
	EffectLKConcentration   = 369
	EffectBottomGospel      = 370
	EffectBaseLevelUp       = 371
	EffectDeath             = 372
	EffectDragonSmoke       = 373
	EffectBottomBasilica    = 374
	EffectAssumptio         = 375
	EffectHitLine2          = 376
	EffectBash3D2           = 377
	EffectEnergyDrain2      = 378
	EffectTransBlueBody     = 379
	EffectMagicCrasher      = 380
	EffectLightBlade        = 382
	EffectEnergyDrain3      = 383
	EffectLineLink2         = 384
	EffectLinkLight         = 385
	EffectTrueSight         = 386
	EffectFalconAssault     = 387
	EffectTripleAttack2     = 388
	EffectPortal4           = 389
	EffectMeltdown          = 390
	EffectCartBoost         = 391
	EffectRejectSword       = 392
	EffectTripleAttack3     = 393
	EffectMoonlit           = 394
	EffectMarionetteLink    = 395
	EffectPinkBody          = 396
	EffectLevel99AuraMid    = 397
	EffectLevel99AuraBottom = 398
	EffectBash3D3           = 399
	EffectBash3D4           = 400
	EffectNapalmVulcan      = 401
	EffectPortal5           = 402
	EffectMagicCrasher2     = 403
	EffectBottomSpider      = 404
	EffectFogWallGround     = 405
	EffectSoulBurn          = 406
	EffectSoulChange        = 407
	EffectSoulBreaker2      = 409
	EffectRainbow           = 410
	EffectPeong             = 411
	EffectPressedBody       = 413
	EffectSpinedBody        = 414
	EffectKickedBody        = 415
	EffectDoubleGumgang     = 418
	EffectReflectBody       = 419
	EffectBabyBody          = 420
	EffectBabyBody2         = 421
	EffectGiantBody         = 422
	EffectGiantBody2        = 423
	EffectAsuraBody         = 424
	EffectFourWayBody       = 425
	EffectQuakeBody         = 426
	EffectAsuraBodyMonster  = 427
	EffectHitLine4          = 429
	EffectHitLine5          = 430
	EffectHitLine6          = 431
	EffectStormKick         = 435
	EffectChemical3         = 439
	EffectAssumptio2        = 440
	EffectBlueCasting       = 441
	EffectRun               = 442
	EffectStopRun           = 443
	EffectStopEffect        = 444
	EffectJumpBody          = 445
	EffectLandBody          = 446
	EffectFoot3             = 447
	EffectFoot4             = 448
	EffectTaeReady          = 449
	EffectDarkGrandCross    = 450
	EffectDarkSoulStrike    = 451
	EffectDarkJupitelHit    = 452
	EffectNPCStop           = 453
	EffectDarkCasting       = 454
	EffectAgiUp             = 456
	EffectJumpKick          = 457
	EffectQuakeBody2        = 458
	EffectStormKick3        = 461
	EffectStormKick4        = 462
	EffectSpinedBody2       = 466
	EffectBeginAsura1       = 467
	EffectBeginAsura2       = 468
	EffectBeginAsura3       = 469
	EffectBeginAsura4       = 470
	EffectBeginAsura5       = 471
	EffectBeginAsura6       = 472
	EffectBeginAsura7       = 473
	EffectDevil1            = 475
	EffectDevil2            = 476
	EffectDevil3            = 477
	EffectDevil4            = 478
	EffectDevil5            = 479
	EffectDevil6            = 480
	EffectDevil7            = 481
	EffectDevil8            = 482
	EffectDevil9            = 483
	EffectDevil10           = 484
	EffectBlackDevil        = 487
	EffectFlowerCast        = 488
	EffectResistPotion      = 491
	EffectMochi             = EffectResistPotion
	EffectRamadan           = 492
	EffectEDP               = 493
	EffectFullStrip         = 495
	EffectPreserve          = 496
	EffectSlim              = 497
	EffectSlim2             = 498
	EffectSlim3             = 499
	EffectChemicalBody      = 500
	EffectCastSpin          = 501
	EffectPierceBody        = 502
	EffectSoulLink          = 503
	EffectChookgi2          = 504
	EffectMemorize          = 505
	EffectItemAccel         = 507
	EffectMapae             = EffectItemAccel
	EffectFirecracker       = 508
	EffectItemPokJuk        = EffectFirecracker
	EffectValentine05       = 509
	EffectBeginAsura11      = 510
	EffectChemical2Dash     = 512
	EffectGroundSample      = 513
	EffectCloud4            = 515
	EffectCloud5            = 516
	EffectBottomHermode     = 517
	EffectCartTermination   = 518
	EffectItemSlow          = 519
	EffectItemFastDown      = EffectItemSlow
	EffectShieldBoomerang3  = 520
	EffectDoubleCastBody    = 521
	EffectGravitation       = 522
	EffectTarotCard1        = 523
	EffectTarotCard2        = 524
	EffectTarotCard3        = 525
	EffectTarotCard4        = 526
	EffectTarotCard5        = 527
	EffectTarotCard6        = 528
	EffectTarotCard7        = 529
	EffectTarotCard8        = 530
	EffectTarotCard9        = 531
	EffectTarotCard10       = 532
	EffectTarotCard11       = 533
	EffectTarotCard12       = 534
	EffectTarotCard13       = 535
	EffectTarotCard14       = 536
	EffectAcidDemon         = 537
	EffectThrowItem4        = 539
	EffectBabyBodyBack      = 540
	EffectThrowItem5        = 541
	EffectBlueBody          = 542
	EffectHated             = 543
	EffectRedLightBody      = 544
	EffectSmaReady          = 546
	EffectStin              = 547
	EffectRedHit            = 548
	EffectBlueHit           = 549
	EffectQuakeBody3        = 550
	EffectStin2             = 553
	EffectHitTexture        = 554
	EffectStin3             = 555
	EffectScreenQuake       = 563
	EffectHfliMoon1         = 565
	EffectHfliMoon2         = 566
	EffectHfliMoon3         = 567
	EffectHoUp              = 568
	EffectHamiDefence       = 569
	EffectHamiCastle        = 570
	EffectHamiBlood         = 571
	EffectHated2            = 572
	EffectTwilight1         = 573
	EffectTwilight2         = 574
	EffectTwilight3         = 575
	EffectBoxThunder        = 576
	EffectItemThunder       = EffectBoxThunder
	EffectBoxResentment     = 577
	EffectItemCloud         = EffectBoxResentment
	EffectItemCurse         = 578
	EffectBoxDrowsiness     = 579
	EffectItemZZZ           = EffectBoxDrowsiness
	EffectBoxSunlight       = 580
	EffectItemRain          = EffectBoxSunlight
	EffectM01               = 583
	EffectM02               = 584
	EffectM03               = 585
	EffectM04               = 586
	EffectM05               = 587
	EffectM06               = 588
	EffectM07               = 589
	EffectKaizel            = 590
	EffectCloud6            = 592
	EffectStatFoodSTR       = 593
	EffectStatFoodINT       = 594
	EffectStatFoodVIT       = 595
	EffectStatFoodAGI       = 596
	EffectStatFoodDEX       = 597
	EffectStatFoodLUK       = 598
	EffectShrink            = 599
	EffectThrowItem6        = 600
	EffectSightBlaster      = 601
	EffectQuakeBody4        = 602
	EffectFireHit2          = 603
	EffectNPCStop2          = 604
	EffectNPCStop2Del       = 605
	EffectFVoice            = 606
	EffectWink              = 607
	EffectCookingOK         = 608
	EffectCookingFail       = 609
	EffectFirecracker1      = 612
	EffectHapgyeok          = 612
	EffectThrowItem7        = 613
	EffectThrowItem8        = 614
	EffectThrowItem9        = 615
	EffectThrowItem10       = 616
	EffectKouenka           = 618
	EffectHyousensou        = 619
	EffectBottomSuiton      = 620
	EffectStin4             = 621
	EffectThunderStorm2     = 622
	EffectChemical4         = 623
	EffectStin5             = 624
	EffectMadnessBlue       = 625
	EffectMadnessRed        = 626
	EffectRGCoin3           = 627
	EffectBash3D5           = 628
	EffectChookgi3          = 629
	EffectKirikage          = 630
	EffectTatami            = 631
	EffectKasumikiri        = 632
	EffectIssen             = 633
	EffectKaen              = 634
	EffectBaku              = 635
	EffectHyousyouraku      = 636
	EffectDesperado         = 637
	EffectLightningS        = 638
	EffectBlindS            = 639
	EffectPoisonS           = 640
	EffectFreezingS         = 641
	EffectFlareS            = 642
	EffectRapidShower       = 643
	EffectMagicalBullet     = 644
	EffectSpreadAttack      = 645
	EffectTrackCasting      = 646
	EffectTracking          = 647
	EffectTripleAction      = 648
	EffectBullseye          = 649
	EffectBubbleDrop        = 665
	EffectNPCEarthquake     = 666
	EffectDragonFear        = 668
	EffectWideBleeding      = 669
	EffectWideConfuse       = 670
	EffectBottomRunner      = 671
	EffectBottomTransfer    = 672
	EffectBottomEvilLand    = 674
	EffectGuard3            = 675
	EffectCriticalWound     = 677
	EffectFirecracker2      = 682
	EffectFirecracker3      = 683
	EffectFirecracker4      = 684
	EffectFirecracker5      = 685
	EffectFirecracker6      = 686
	EffectTorchRed          = 690
	EffectTorchGreen        = 691
	EffectMapGhost          = 692
	EffectGlow1             = 693
	EffectGlow2             = 694
	EffectGlow4             = 695
	EffectTorchPurple       = 696
	EffectCloud7            = 697
	EffectCloud8            = 698
	EffectFlowerLeaf        = 699
	EffectItem315           = 704
	EffectItem316           = 705
	EffectItem317           = 706
	EffectStormMin          = 708
	EffectFirecracker7      = 709
	EffectBottomBlue        = 715
	EffectBottomBlue2       = 716
	EffectChristmasCarol    = 717
	EffectFirePillarOn2     = 718
	EffectForestLight5      = 719
	EffectAdoramus          = 721
	EffectIgnitionBreak     = 722
	EffectFrostMisty        = 726
	EffectCrimsonRock       = 727
	EffectHellInferno       = 728
	EffectMarshOfAbyss      = 729
	EffectDragonHowling     = 731
	EffectEarthWall         = 732
	EffectChainLightning    = 734
	EffectAimedBolt         = 745
	EffectArrowStorm        = 746
	EffectLaulamus          = 747
	EffectLauagnus          = 748
	EffectMillenniumShield  = 749
	EffectConcentration2    = 750
	EffectGlassWall3        = 754
	EffectBerserkPotion2    = 756
	EffectRolling1          = 758
	EffectRolling2          = 759
	EffectRolling3          = 760
	EffectRolling4          = 761
	EffectRolling5          = 762
	EffectRolling6          = 763
	EffectRolling7          = 764
	EffectRolling8          = 765
	EffectRolling9          = 766
	EffectRolling10         = 767
	EffectCastSpin2         = 775
	EffectCrashAxe          = 795
	EffectStasis            = 799
	EffectBottomBasilica2   = 802
	EffectRecognized        = 803
	EffectTetra             = 804
	EffectTetraCasting      = 805
	EffectStretch           = 811
	EffectEnervation        = 813
	EffectEnervation2       = 814
	EffectEnervation3       = 815
	EffectEnervation4       = 816
	EffectEnervation5       = 817
	EffectEnervation6       = 818
	EffectBottomManhole     = 822
	EffectManhole           = 823
	EffectForestLight6      = 825
	EffectBottomAni         = 827
	EffectBottomMaelstrom   = 828
	EffectBottomBloodyLust  = 829
	EffectHealN             = 832
	EffectChookgiN          = 833
	EffectDance1            = 847
	EffectBotReverb         = 856
	EffectRainParticle      = 857
	EffectChemicalV2        = 858
	EffectBotReverb2        = 860
	EffectCirclePower2      = 861
	EffectSecra2            = 862
	EffectSprPlant2         = 866
	EffectSprPlant3         = 868
	EffectSprPlant4         = 870
	EffectSprPlant5         = 872
	EffectSprPlant6         = 874
	EffectSprPlant7         = 876
	EffectSprPlant8         = 878
	EffectHeartAsura        = 879
	EffectGlassWall4        = 883
	EffectBash3D6           = 885
	EffectElectric4         = 888
	EffectTeiHit1T          = 889
	EffectPressure2         = 906
	EffectPrimeCharge2      = 908
	EffectPrimeCharge3      = 909
	EffectPrimeCharge4      = 910
	EffectFireWall2         = 920
	EffectSprPlant10        = 922
	EffectShockwave2        = 926
	EffectColdThrow2        = 928
	EffectDemonicFire4      = 929
	EffectPressure3         = 930
	EffectPoisonMist        = 959
	EffectEraserCutter      = 960
	EffectLavaSlide         = 964
	EffectSonicClaw         = 965
	EffectTinderBreaker     = 966
	EffectMidnightFrenzy    = 967
	EffectVolcanicAsh       = 975
	EffectRWC2011           = 987
	EffectRWC2011Two        = 988
	EffectRunMakeOK         = 1015
	EffectRunMakeFailure    = 1016
	EffectMIResultMakeOK    = 1017
	EffectMIResultMakeFail  = 1018
	EffectAllRayProtect     = 1019
	EffectVenomFog          = 1020
	EffectDustStorm         = 1021
	EffectDanceBladeAtk     = 1029
	EffectInvincibleOff2    = 1031
	EffectDeathSummon       = 1033
	EffectGCDarkCrow        = 1040
	EffectAllFullThrottle   = 1042
	EffectSRFlashCombo      = 1043
	EffectRKLuxAnima        = 1044
	EffectSOElemShield      = 1046
	EffectABOffertorium     = 1047
	EffectWLTelekinesis     = 1048
	EffectGNIllusionDoping  = 1049
	EffectNCMagmaEruption   = 1050
	EffectNPCChill          = 1055
	EffectOffertoriumRing   = 1057
	EffectHammerOfGod       = 1062
	EffectAchComplete       = 1094
	EffectBodyColor         = 1111
	EffectBakuretsuHadou    = 1130
	EffectDropPink          = 1186
	EffectDropYellow        = 1189
	EffectDropPurple        = 1190
	EffectDigitalSpace      = 1240
	EffectDropBlue          = 1869
	EffectDropGreen         = 1870
	EffectDropRed           = 1871
	EffectNewSuccess        = 1872
	EffectNewFailure        = 1873
	EffectNewIntro          = 1874
	EffectEnchantYellow     = 1875
	EffectEnchantSuccess    = 1876
	EffectEnchantFail       = 1877
	EffectEnchantBlue       = 1878
	EffectEnchantUpSuccess  = 1879
	EffectEnchantUpFail     = 1880
	EffectEnchantGreen      = 1881
	EffectEnchantResetOK    = 1882
	EffectEnchantResetFail  = 1883
)

// Goro-local effect IDs.
const (
	EffectColdBolt               = 10014
	EffectFireBolt               = 10019
	EffectCastRing               = 10021
	EffectQuakeMagnum            = 10022
	EffectArrowShot              = 10060
	EffectArrowShower            = 10061
	EffectMagicPower             = SkillEffectMagicPower
	EffectWhitePulse             = SkillEffectWhitePulse
	EffectSpearProjectile        = SkillEffectSpearProjectile
	EffectSpearHitSound          = SkillEffectSpearHitSound
	EffectEnemyHitNormal1        = SkillEffectEnemyHitNormal1
	EffectQuake                  = SkillEffectQuake
	EffectAnkleSnareGround       = SkillEffectAnkleSnareGround
	EffectSharpShootingCast      = SkillEffectSharpShootingCast
	EffectAdrenalineCast         = SkillEffectAdrenalineCast
	EffectMaximizeSounds         = SkillEffectMaximizeSounds
	EffectGreedSound             = SkillEffectGreedSound
	EffectGospelGround           = SkillEffectGospelGround
	EffectHermodeMusic           = SkillEffectHermodeMusic
	EffectBottomDissonanceGround = SkillEffectBottomDissonanceGround
	EffectBottomLullabyGround    = SkillEffectBottomLullabyGround
	EffectBottomRichKimGround    = SkillEffectBottomRichKimGround
	EffectBottomChaosGround      = SkillEffectBottomChaosGround
	EffectBottomDrumGround       = SkillEffectBottomDrumGround
	EffectBottomNibelungGround   = SkillEffectBottomNibelungGround
	EffectBottomRokiGround       = SkillEffectBottomRokiGround
	EffectBottomAbyssGround      = SkillEffectBottomAbyssGround
	EffectBottomSiegGround       = SkillEffectBottomSiegGround
	EffectBottomWhistleGround    = SkillEffectBottomWhistleGround
	EffectBottomSinXGround       = SkillEffectBottomSinXGround
	EffectBottomBragiGround      = SkillEffectBottomBragiGround
	EffectBottomAppleGround      = SkillEffectBottomAppleGround
	EffectBottomUglyDanceGround  = SkillEffectBottomUglyDanceGround
	EffectBottomHummingGround    = SkillEffectBottomHummingGround
	EffectBottomForgetGround     = SkillEffectBottomForgetGround
	EffectBottomFortuneGround    = SkillEffectBottomFortuneGround
	EffectBottomServiceGround    = SkillEffectBottomServiceGround
	EffectGunslingerCoinSound    = SkillEffectGunslingerCoinSound
	EffectSteelBodyRing          = SkillEffectSteelBodyRing
	EffectEmotion                = 1000000
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
	EffectHit1:              hit1EffectSpec(),
	EffectBashHit:           bashHitEffectSpec(),
	EffectHit3:              hit3EffectSpec(),
	EffectHit4:              hit4EffectSpec(),
	EffectHit5:              hit5EffectSpec(),
	EffectHit6:              hit6EffectSpec(),
	EffectEntry:             entryEffectSpec(),
	EffectExit:              exitEffectSpec(),
	EffectWarp:              warpEffectSpec(),
	EffectEnhance:           enhanceEffectSpec(),
	EffectMammonite:         mammoniteEffectSpec(),
	EffectEndure:            endureEffectSpec(),
	EffectBeginSpell:        beginSpellEffectSpec(),
	EffectGlassWall:         glassWallEffectSpec(),
	EffectHealSP:            healSPEffectSpec(),
	EffectSoulStrike:        soulStrikeEffectSpec(),
	EffectBashBegin:         bashBeginEffectSpec(),
	EffectMagnumBreak:       magnumBreakEffectSpec(),
	EffectSteal:             stealEffectSpec(),
	EffectPoisonAttack:      poisonAttackEffectSpec(),
	EffectDetoxication:      detoxicationEffectSpec(),
	EffectSight:             mageSightEffectSpec(),
	EffectStoneCurse:        stoneCurseEffectSpec(),
	EffectFireBall:          fireBallEffectSpec(),
	EffectFireWall:          fireWallEffectSpec(),
	EffectIceArrow:          coldBoltEffectSpec(),
	EffectFrostDiver:        frostDiverEffectSpec(),
	EffectFrostDiverHit:     mageFrostHitEffectSpec(),
	EffectLightningBolt:     lightningBoltEffectSpec(),
	EffectThunderStorm:      thunderStormEffectSpec(),
	EffectFireArrow:         fireBoltEffectSpec(),
	EffectNapalmBeat:        napalmBeatEffectSpec(),
	EffectRuwach:            ruwachEffectSpec(),
	EffectTeleportOld:       teleportOldEffectSpec(),
	EffectReadyPortalOld:    readyPortalOldEffectSpec(),
	EffectIncAgility:        incAgilityEffectSpec(),
	EffectDecAgility:        decAgilityEffectSpec(),
	EffectAqua:              aquaEffectSpec(),
	EffectSignum:            signumEffectSpec(),
	EffectAngelus:           angelusEffectSpec(),
	EffectBlessing:          blessingEffectSpec(),
	EffectIncAgiDex:         concentrationBeginEffectSpec(),
	EffectSmoke:             smokeEffectSpec(),
	EffectFirefly:           fireflyEffectSpec(),
	EffectTorch:             torchEffectSpec(),
	EffectFireHit:           fireHitEffectSpec(),
	EffectFireSplashHit:     fireSplashHitEffectSpec(),
	EffectColdHit:           coldHitEffectSpec(),
	EffectWindHit:           windHitEffectSpec(),
	EffectPoisonHit:         poisonHitEffectSpec(),
	EffectBeginSpell2:       beginSpell2EffectSpec(),
	EffectBeginSpell3:       beginSpell3EffectSpec(),
	EffectBeginSpell4:       beginSpell4EffectSpec(),
	EffectBeginSpell5:       beginSpell5EffectSpec(),
	EffectBeginSpell6:       beginSpell6EffectSpec(),
	EffectBeginSpell7:       beginSpell7EffectSpec(),
	EffectLockOnTarget:      lockOnTargetEffectSpec(),
	EffectWarpZone:          warpZoneEffectSpec(),
	EffectSightTrasher:      sightTrasherEffectSpec(),
	EffectArrowShotRO:       arrowShotROEffectSpec(),
	EffectInvenom:           invenomEffectSpec(),
	EffectCure:              cureEffectSpec(),
	EffectProvoke:           provokeEffectSpec(),
	EffectMvp:               mvpEffectSpec(),
	EffectSkidTrap:          skidTrapEffectSpec(),
	EffectBrandishSpear:     brandishSpearEffectSpec(),
	EffectBowlingBash:       bowlingBashEffectSpec(),
	EffectIceWall:           iceWallEffectSpec(),
	EffectGloria:            gloriaEffectSpec(),
	EffectMagnificat:        magnificatEffectSpec(),
	EffectResurrection:      resurrectionEffectSpec(),
	EffectRecovery:          recoveryEffectSpec(),
	EffectEarthSpike:        earthSpikeEffectSpec(),
	EffectSpearBoomerang:    spearBoomerangEffectSpec(),
	EffectPierce:            pierceEffectSpec(),
	EffectTurnUndead:        turnUndeadEffectSpec(),
	EffectSanctuary:         sanctuaryEffectSpec(),
	EffectImpositio:         impositioEffectSpec(),
	EffectLexAeterna:        lexAeternaEffectSpec(),
	EffectAspersio:          aspersioEffectSpec(),
	EffectLexDivina:         lexDivinaEffectSpec(),
	EffectSuffragium:        suffragiumEffectSpec(),
	EffectStormGust:         stormGustEffectSpec(),
	EffectLordVermilion:     lordVermilionEffectSpec(),
	EffectBenedictio:        benedictioEffectSpec(),
	EffectMeteorStorm:       meteorStormEffectSpec(),
	EffectJupitelThunder:    jupitelThunderEffectSpec(),
	EffectJupitelHit:        jupitelHitEffectSpec(),
	EffectQuagmire:          quagmireEffectSpec(),
	EffectFirePillar:        firePillarEffectSpec(),
	EffectFirePillarBomb:    firePillarBombEffectSpec(),
	EffectHasteUp:           hasteUpEffectSpec(),
	EffectFlasher:           flasherEffectSpec(),
	EffectRemoveTrap:        removeTrapEffectSpec(),
	EffectRepairWeapon:      repairWeaponEffectSpec(),
	EffectCrashEarth:        crashEarthEffectSpec(),
	EffectWeaponPerfect:     weaponPerfectEffectSpec(),
	EffectMaximizePower:     maximizePowerEffectSpec(),
	EffectBlastMine:         blastMineSoundEffectSpec(),
	EffectBlastMineBomb:     blastMineEffectSpec(),
	EffectClaymore:          claymoreEffectSpec(),
	EffectFreezingTrap:      freezingTrapEffectSpec(),
	EffectBubble:            bubbleEffectSpec(),
	EffectGasPush:           gasPushEffectSpec(),
	EffectSpringTrap:        springTrapEffectSpec(),
	EffectKyrie:             kyrieEffectSpec(),
	EffectMagnus:            magnusEffectSpec(),
	EffectBlitzBeat:         blitzBeatEffectSpec(),
	EffectWaterBall:         waterBallEffectSpec(),
	EffectWaterBall2:        waterBall2EffectSpec(),
	EffectDetecting:         detectingEffectSpec(),
	EffectCloaking:          cloakingEffectSpec(),
	EffectSonicBlow:         sonicBlowEffectSpec(),
	EffectSonicBlowHit:      sonicBlowHitEffectSpec(),
	EffectGrimtooth:         grimtoothGroundEffectSpec(),
	EffectVenomDust:         venomDustEffectSpec(),
	EffectEnchantPoison:     enchantPoisonEffectSpec(),
	EffectPoisonReact:       poisonReactEffectSpec(),
	EffectPoisonReact2:      poisonReact2EffectSpec(),
	EffectOverthrust:        overthrustEffectSpec(),
	EffectVenomSplasher:     venomSplasherEffectSpec(),
	EffectTwoHandQuicken:    twoHandQuickenEffectSpec(),
	EffectAutoCounter:       autoCounterEffectSpec(),
	EffectGrimtoothAtk:      grimtoothAtkEffectSpec(),
	EffectFreeze:            freezeEffectSpec(),
	EffectFreezed:           freezedEffectSpec(),
	EffectIceCrash:          iceCrashEffectSpec(),
	EffectSlowPoison:        slowPoisonEffectSpec(),
	EffectFirePillarOn:      firePillarOnEffectSpec(),
	EffectSandman:           sandmanEffectSpec(),
	EffectRevive:            reviveEffectSpec(),
	EffectPneuma:            pneumaEffectSpec(),
	EffectHeavenDrive:       heavenDriveEffectSpec(),
	EffectSonicBlow2:        sonicBlow2EffectSpec(),
	EffectBrandishSpear2:    brandishSpear2EffectSpec(),
	EffectShockwave:         shockwaveEffectSpec(),
	EffectShockwaveHit:      shockwaveHitEffectSpec(),
	EffectEarthHit:          earthHitEffectSpec(),
	EffectPierceSelf:        pierceSelfEffectSpec(),
	EffectBowlingSelf:       bowlingSelfEffectSpec(),
	EffectSpearStabSelf:     spearStabSelfEffectSpec(),
	EffectSpearBmrSelf:      spearBmrSelfEffectSpec(),
	EffectHolyLight:         holyLightEffectSpec(),
	EffectConcentration:     concentrationEffectSpec(),
	EffectRefineOK:          refineOKEffectSpec(),
	EffectRefineFail:        refineFailEffectSpec(),
	EffectJobLevelUp:        jobLevelUpEffectSpec(),
	EffectRain:              weatherRainEffectSpec(),
	EffectSnow:              weatherSnowEffectSpec(),
	EffectSakura:            sakuraEffectSpec(),
	EffectBanjjakii:         banjjakiiEffectSpec(),
	EffectMakeBlur:          makeBlurEffectSpec(),
	EffectEnergyCoat:        energyCoatEffectSpec(),
	EffectCartRevolution:    cartRevolutionEffectSpec(),
	EffectVenomDust2:        venomDust2EffectSpec(),
	EffectMentalBreak:       mentalBreakEffectSpec(),
	EffectMagicalAtkHit:     magicalAtkHitEffectSpec(),
	EffectSuiExplosion:      suiExplosionEffectSpec(),
	EffectSuicide:           suicideEffectSpec(),
	EffectComboAttack1:      comboAttack1EffectSpec(),
	EffectComboAttack2:      comboAttack2EffectSpec(),
	EffectComboAttack3:      comboAttack3EffectSpec(),
	EffectComboAttack4:      comboAttack4EffectSpec(),
	EffectComboAttack5:      comboAttack5EffectSpec(),
	EffectGuidedAttack:      guidedAttackEffectSpec(),
	EffectPoisonAttack2:     poisonAttack2EffectSpec(),
	EffectSilenceAttack:     silenceAttackEffectSpec(),
	EffectStunAttack:        stunAttackEffectSpec(),
	EffectPetrifyAttack:     petrifyAttackEffectSpec(),
	EffectSleepAttack:       sleepAttackEffectSpec(),
	EffectPong:              pongEffectSpec(),
	EffectLevel99:           level99EffectSpec(),
	EffectLevel99Ground:     level99GroundEffectSpec(),
	EffectLevel99Bubble:     level99BubbleEffectSpec(),
	EffectGumgang:           gumgangEffectSpec(),
	EffectPotionRed:         potionRedEffectSpec(),
	EffectPotionOrange:      potionOrangeEffectSpec(),
	EffectPotionYellow:      potionYellowEffectSpec(),
	EffectPotionWhite:       potionWhiteEffectSpec(),
	EffectPotionBlue:        bluePotionEffectSpec(),
	EffectPotionGreen:       potionGreenEffectSpec(),
	EffectFood:              foodEffectSpec(),
	EffectFoodBlue:          foodBlueEffectSpec(),
	EffectDarkBreath:        darkBreathEffectSpec(),
	EffectDefender:          defenderEffectSpec(),
	EffectKeeping:           keepingEffectSpec(),
	EffectSummonSlave:       summonSlaveEffectSpec(),
	EffectBloodDrain:        bloodDrainEffectSpec(),
	EffectEnergyDrain:       energyDrainEffectSpec(),
	EffectItemFast:          itemFastEffectSpec(),
	EffectItemFast2:         itemFast2EffectSpec(),
	EffectItemFast3:         berserkPotionEffectSpec(),
	EffectCrusaderDef:       crusaderDefEffectSpec(),
	EffectGanbantein:        ganbanteinEffectSpec(),
	EffectVolcano:           volcanoEffectSpec(),
	EffectGrandCross:        grandCrossEffectSpec(),
	EffectIntimidate:        intimidateEffectSpec(),
	EffectChookgi:           spiritSphereEffectSpec(),
	EffectCloud:             cloudEffectSpec(),
	EffectCloud2:            cloud2EffectSpec(),
	EffectMapPillar:         mapPillarBlueEffectSpec(),
	EffectLineLink:          devotionLinkEffectSpec(),
	EffectCloud3:            cloud3EffectSpec(),
	EffectSpellBreaker:      spellBreakerEffectSpec(),
	EffectDispell:           dispellEffectSpec(),
	EffectDeluge:            delugeEffectSpec(),
	EffectViolentGale:       violentGaleEffectSpec(),
	EffectLandProtector:     landProtectorEffectSpec(),
	EffectBottomVolcano:     bottomVolcanoEffectSpec(),
	EffectBottomDeluge:      bottomDelugeEffectSpec(),
	EffectBottomViolent:     bottomViolentEffectSpec(),
	EffectBottomLand:        landProtectorGroundEffectSpec(),
	EffectMagicRod:          magicRodEffectSpec(),
	EffectHolyCross:         holyCrossEffectSpec(),
	EffectShieldCharge:      shieldChargeEffectSpec(),
	EffectMapPillar2:        mapPillar2EffectSpec(),
	EffectProvidence:        providenceEffectSpec(),
	EffectShieldBoomer:      shieldBoomerangEffectSpec(),
	EffectSpearQuicken:      spearQuickenEffectSpec(),
	EffectDevotion:          devotionEffectSpec(),
	EffectReflectShield:     reflectShieldEffectSpec(),
	EffectAbsorbSpirits:     absorbSpiritsEffectSpec(),
	EffectSteelBody:         steelBodyEffectSpec(),
	EffectFlameLauncher:     flameLauncherEffectSpec(),
	EffectFrostWeapon:       frostWeaponEffectSpec(),
	EffectLightningLoad:     lightningLoadEffectSpec(),
	EffectSeismicWeapon:     seismicWeaponEffectSpec(),
	EffectMapPillar3:        mapPillar3EffectSpec(),
	EffectMapPillar4:        mapPillar4EffectSpec(),
	EffectGumgang2:          gumgang2EffectSpec(),
	EffectTeiHit1:           teiHit1EffectSpec(),
	EffectGumgang3:          gumgang3EffectSpec(),
	EffectTeiHit2:           teiHit2EffectSpec(),
	EffectTanji:             tanjiEffectSpec(),
	EffectTeiHit1X:          teiHit1XEffectSpec(),
	EffectChimto:            chimtoEffectSpec(),
	EffectStealCoin:         stealCoinEffectSpec(),
	EffectStripWeapon:       stripWeaponEffectSpec(),
	EffectStripShield:       stripShieldEffectSpec(),
	EffectStripArmor:        stripArmorEffectSpec(),
	EffectStripHelm:         stripHelmEffectSpec(),
	EffectChainCombo:        chainComboEffectSpec(),
	EffectRogueCoin:         rogueCoinEffectSpec(),
	EffectBackStab:          backStabEffectSpec(),
	EffectTeiHit3:           teiHit3EffectSpec(),
	EffectBottomDissonance:  bottomDissonanceEffectSpec(),
	EffectBottomLullaby:     bottomLullabyEffectSpec(),
	EffectBottomRichKim:     bottomRichKimEffectSpec(),
	EffectBottomChaos:       bottomChaosEffectSpec(),
	EffectBottomDrum:        bottomDrumEffectSpec(),
	EffectBottomNibelung:    bottomNibelungEffectSpec(),
	EffectBottomRoki:        songOutEffectSpec(),
	EffectBottomAbyss:       bottomAbyssEffectSpec(),
	EffectBottomSieg:        bottomSiegEffectSpec(),
	EffectBottomWhistle:     bottomWhistleEffectSpec(),
	EffectBottomSinX:        bottomSinXEffectSpec(),
	EffectBottomBragi:       bottomBragiEffectSpec(),
	EffectBottomApple:       bottomAppleEffectSpec(),
	EffectBottomUglyDance:   bottomUglyDanceEffectSpec(),
	EffectBottomHumming:     bottomHummingEffectSpec(),
	EffectBottomForget:      bottomForgetEffectSpec(),
	EffectBottomFortune:     bottomFortuneEffectSpec(),
	EffectBottomService:     bottomServiceEffectSpec(),
	EffectTalkFrostJoke:     talkFrostJokeEffectSpec(),
	EffectTalkScream:        talkScreamEffectSpec(),
	EffectPokJuk:            pokJukEffectSpec(),
	EffectThrowItem:         acidTerrorBottleEffectSpec(),
	EffectThrowItem2:        throwItem2EffectSpec(),
	EffectChemicalProt:      chemicalProtectionEffectSpec(),
	EffectDemonstration:     demonstrationEffectSpec(),
	EffectChemical2:         chemical2EffectSpec(),
	EffectTeleportation:     teleportationEffectSpec(),
	EffectPharmacyOK:        pharmacyOKEffectSpec(),
	EffectPharmacyFail:      pharmacyFailEffectSpec(),
	EffectThrowItem3:        throwItem3EffectSpec(),
	EffectFirstAid:          firstAidEffectSpec(),
	EffectSprinkleSand:      sprinkleSandEffectSpec(),
	EffectLoud:              loudEffectSpec(),
	EffectHeal:              healEffectSpec(),
	EffectHeal2:             heal2EffectSpec(),
	EffectExit2:             exit2EffectSpec(),
	EffectSafetyWall:        safetyWallEffectSpec(),
	EffectReadyPortal:       readyPortalEffectSpec(),
	EffectPortal:            portalEffectSpec(),
	EffectBottomMagnus:      bottomMagnusEffectSpec(),
	EffectBottomSanc:        bottomSancEffectSpec(),
	EffectHealOffensive:     healOffensiveEffectSpec(),
	EffectWarpZone2:         warpZone2EffectSpec(),
	EffectHeal4:             heal4EffectSpec(),
	EffectBeginAsura:        beginAsuraCastEffectSpec(),
	EffectTripleAttack:      tripleAttackEffectSpec(),
	EffectHitLine:           hitLineEffectSpec(),
	EffectHPTime:            hpTimeEffectSpec(),
	EffectSPTime:            spTimeEffectSpec(),
	EffectMaple:             mapleEffectSpec(),
	EffectBlind:             blindEffectSpec(),
	EffectPoisonStatus:      poisonStatusEffectSpec(),
	EffectGuard:             guardEffectSpec(),
	EffectJobLvUp50:         jobLvUp50EffectSpec(),
	EffectAngel2:            superAngelEffectSpec(),
	EffectMagnum2:           magnum2EffectSpec(),
	EffectCoupleCasting:     coupleCastingEffectSpec(),
	EffectEntry2:            entry2EffectSpec(),
	EffectColorPaper:        colorPaperEffectSpec(),
	EffectSoulBreaker:       soulBreakerBaseEffectSpec(),
	EffectLevel99Aura1:      level99BlueBubbleEffectSpec(),
	EffectFoodChocolate:     foodChocolateEffectSpec(),
	EffectPressure:          pressureEffectSpec(),
	EffectBash3D:            sacrificeEffectSpec(),
	EffectAuraBlade:         auraBladeEffectSpec(),
	EffectRedBody:           redBodyEffectSpec(),
	EffectLKConcentration:   lkConcentrationEffectSpec(),
	EffectBottomGospel:      bottomGospelEffectSpec(),
	EffectBaseLevelUp:       baseLevelUpEffectSpec(),
	EffectDeath:             deathEffectSpec(),
	EffectDragonSmoke:       dragonSmokeEffectSpec(),
	EffectBottomBasilica:    bottomBasilicaEffectSpec(),
	EffectAssumptio:         assumptioEffectSpec(),
	EffectHitLine2:          palmStrikeEffectSpec(),
	EffectBash3D2:           tigerFistEffectSpec(),
	EffectEnergyDrain2:      soulDrainEffectSpec(),
	EffectTransBlueBody:     transBlueBodyEffectSpec(),
	EffectMagicCrasher:      magicCrasherEffectSpec(),
	EffectLightBlade:        lightBladeEffectSpec(),
	EffectEnergyDrain3:      hpConversionEffectSpec(),
	EffectLineLink2:         soulChangeLinkEffectSpec(),
	EffectLinkLight:         linkLightEffectSpec(),
	EffectTrueSight:         trueSightEffectSpec(),
	EffectFalconAssault:     falconAssaultEffectSpec(),
	EffectTripleAttack2:     sharpShootingEffectSpec(),
	EffectPortal4:           windWalkEffectSpec(),
	EffectMeltdown:          meltdownEffectSpec(),
	EffectCartBoost:         cartBoostEffectSpec(),
	EffectRejectSword:       rejectSwordEffectSpec(),
	EffectTripleAttack3:     arrowVulcanEffectSpec(),
	EffectMoonlit:           moonlitEffectSpec(),
	EffectMarionetteLink:    marionetteEffectSpec(),
	EffectPinkBody:          pinkBodyEffectSpec(),
	EffectLevel99AuraMid:    level99EffectSpec(),
	EffectLevel99AuraBottom: level99GroundEffectSpec(),
	EffectBash3D3:           bash3D3EffectSpec(),
	EffectBash3D4:           bash3D4EffectSpec(),
	EffectNapalmVulcan:      napalmVulcanEffectSpec(),
	EffectPortal5:           portal5EffectSpec(),
	EffectMagicCrasher2:     magicCrasher2EffectSpec(),
	EffectBottomSpider:      spiderWebEffectSpec(),
	EffectFogWallGround:     fogWallGroundEffectSpec(),
	EffectSoulBurn:          soulBurnEffectSpec(),
	EffectSoulChange:        soulChangeEffectSpec(),
	EffectSoulBreaker2:      soulBreaker2EffectSpec(),
	EffectRainbow:           rainbowEffectSpec(),
	EffectPeong:             taekwonJumpEffectSpec(),
	EffectPressedBody:       pressedBodyEffectSpec(),
	EffectSpinedBody:        spinedBodyEffectSpec(),
	EffectKickedBody:        kickedBodyEffectSpec(),
	EffectDoubleGumgang:     starWarmthEffectSpec(),
	EffectReflectBody:       reflectBodyEffectSpec(),
	EffectBabyBody:          babyBodyEffectSpec(),
	EffectBabyBody2:         babyBody2EffectSpec(),
	EffectGiantBody:         giantBodyEffectSpec(),
	EffectGiantBody2:        giantBody2EffectSpec(),
	EffectAsuraBody:         asuraBodyEffectSpec(),
	EffectFourWayBody:       fourWayBodyEffectSpec(),
	EffectQuakeBody:         quakeBodyEffectSpec(),
	EffectAsuraBodyMonster:  asuraBodyMonsterEffectSpec(),
	EffectHitLine4:          hitLineBurstEffectSpec(),
	EffectHitLine5:          hitLineBurstEffectSpec(),
	EffectHitLine6:          hitLineBurstEffectSpec(),
	EffectStormKick:         stormKickEffectSpec(),
	EffectChemical3:         taekwonDashEffectSpec(),
	EffectAssumptio2:        assumptio2EffectSpec(),
	EffectBlueCasting:       blueCastingEffectSpec(),
	EffectRun:               runEffectSpec(),
	EffectStopRun:           stopRunEffectSpec(),
	EffectStopEffect:        taekwonStopEffectSpec(),
	EffectJumpBody:          jumpBodyEffectSpec(),
	EffectLandBody:          landBodyEffectSpec(),
	EffectFoot3:             foot3EffectSpec(),
	EffectFoot4:             foot4EffectSpec(),
	EffectTaeReady:          taeReadyEffectSpec(),
	EffectDarkGrandCross:    darkGrandCrossEffectSpec(),
	EffectDarkSoulStrike:    darkSoulStrikeEffectSpec(),
	EffectDarkJupitelHit:    darkJupitelHitEffectSpec(),
	EffectNPCStop:           npcStopEffectSpec(),
	EffectDarkCasting:       darkCastingEffectSpec(),
	EffectAgiUp:             gunslingerAgilityEffectSpec(),
	EffectJumpKick:          jumpKickEffectSpec(),
	EffectQuakeBody2:        quakeBody2EffectSpec(),
	EffectStormKick3:        stormKick3EffectSpec(),
	EffectStormKick4:        kaupeDodgeEffectSpec(),
	EffectSpinedBody2:       spinedBody2EffectSpec(),
	EffectBeginAsura1:       beginAsura1EffectSpec(),
	EffectBeginAsura2:       beginAsura2EffectSpec(),
	EffectBeginAsura3:       beginAsura3EffectSpec(),
	EffectBeginAsura4:       beginAsura4EffectSpec(),
	EffectBeginAsura5:       beginAsura5EffectSpec(),
	EffectBeginAsura6:       beginAsura6EffectSpec(),
	EffectBeginAsura7:       beginAsura7EffectSpec(),
	EffectDevil1:            starBlindEffectSpec(),
	EffectDevil2:            starBlindEffectSpec(),
	EffectDevil3:            starBlindEffectSpec(),
	EffectDevil4:            starBlindEffectSpec(),
	EffectDevil5:            starBlindEffectSpec(),
	EffectDevil6:            starBlindEffectSpec(),
	EffectDevil7:            starBlindEffectSpec(),
	EffectDevil8:            starBlindEffectSpec(),
	EffectDevil9:            starBlindEffectSpec(),
	EffectDevil10:           starBlindEffectSpec(),
	EffectBlackDevil:        starBlackDevilEffectSpec(),
	EffectFlowerCast:        starComfortEffectSpec(),
	EffectMochi:             mochiEffectSpec(),
	EffectRamadan:           ramadanEffectSpec(),
	EffectEDP:               enchantDeadlyPoisonEffectSpec(),
	EffectFullStrip:         fullStripEffectSpec(),
	EffectPreserve:          preserveEffectSpec(),
	EffectSlim:              slimEffectSpec(),
	EffectSlim2:             slim2EffectSpec(),
	EffectSlim3:             slim3EffectSpec(),
	EffectChemicalBody:      chemicalBodyEffectSpec(),
	EffectCastSpin:          castSpinEffectSpec(),
	EffectPierceBody:        pierceBodyEffectSpec(),
	EffectSoulLink:          soulLinkEffectSpec(),
	EffectChookgi2:          spiritSphereEffectSpec(),
	EffectMemorize:          memorizeEffectSpec(),
	EffectMapae:             mapaeEffectSpec(),
	EffectItemPokJuk:        itemPokJukEffectSpec(),
	EffectValentine05:       valentine05EffectSpec(),
	EffectBeginAsura11:      beginAsura11EffectSpec(),
	EffectChemical2Dash:     chainCrushEffectSpec(),
	EffectGroundSample:      groundSampleEffectSpec(),
	EffectCloud4:            einbrochWeatherCloudEffectSpec(),
	EffectCloud5:            cloud5EffectSpec(),
	EffectBottomHermode:     hermodeEffectSpec(),
	EffectCartTermination:   cartTerminationEffectSpec(),
	EffectItemFastDown:      itemFastDownEffectSpec(),
	EffectShieldBoomerang3:  shieldChainEffectSpec(),
	EffectDoubleCastBody:    doubleCastBodyEffectSpec(),
	EffectGravitation:       gravitationEffectSpec(),
	EffectTarotCard1:        tarotCard1EffectSpec(),
	EffectTarotCard2:        tarotCard2EffectSpec(),
	EffectTarotCard3:        tarotCard3EffectSpec(),
	EffectTarotCard4:        tarotCard4EffectSpec(),
	EffectTarotCard5:        tarotCard5EffectSpec(),
	EffectTarotCard6:        tarotCard6EffectSpec(),
	EffectTarotCard7:        tarotCard7EffectSpec(),
	EffectTarotCard8:        tarotCard8EffectSpec(),
	EffectTarotCard9:        tarotCard9EffectSpec(),
	EffectTarotCard10:       tarotCard10EffectSpec(),
	EffectTarotCard11:       tarotCard11EffectSpec(),
	EffectTarotCard12:       tarotCard12EffectSpec(),
	EffectTarotCard13:       tarotCard13EffectSpec(),
	EffectTarotCard14:       tarotCard14EffectSpec(),
	EffectAcidDemon:         acidDemonEffectSpec(),
	EffectThrowItem4:        acidBottlesEffectSpec(),
	EffectBabyBodyBack:      babyBodyBackEffectSpec(),
	EffectThrowItem5:        throwItem5EffectSpec(),
	EffectBlueBody:          blueBodyEffectSpec(),
	EffectHated:             hatedEffectSpec(),
	EffectRedLightBody:      redLightBodyEffectSpec(),
	EffectSmaReady:          esmaReadyEffectSpec(),
	EffectStin:              estinEffectSpec(),
	EffectRedHit:            redHitEffectSpec(),
	EffectBlueHit:           blueHitEffectSpec(),
	EffectQuakeBody3:        quakeBody3EffectSpec(),
	EffectStin2:             esmaEffectSpec(),
	EffectHitTexture:        esmaHitTextureEffectSpec(),
	EffectStin3:             estunEffectSpec(),
	EffectScreenQuake:       screenQuakeEffectSpec(),
	EffectHfliMoon1:         hfliMoon1EffectSpec(),
	EffectHfliMoon2:         hfliMoon2EffectSpec(),
	EffectHfliMoon3:         hfliMoon3EffectSpec(),
	EffectHoUp:              hoUpEffectSpec(),
	EffectHamiDefence:       hamiDefenceEffectSpec(),
	EffectHamiCastle:        hamiCastleEffectSpec(),
	EffectHamiBlood:         hamiBloodEffectSpec(),
	EffectHated2:            starWarmthParticlesEffectSpec(),
	EffectTwilight1:         twilight1EffectSpec(),
	EffectTwilight2:         twilight2EffectSpec(),
	EffectTwilight3:         twilight3EffectSpec(),
	EffectItemThunder:       itemThunderEffectSpec(),
	EffectItemCloud:         itemCloudEffectSpec(),
	EffectItemCurse:         itemCurseEffectSpec(),
	EffectItemZZZ:           itemZZZEffectSpec(),
	EffectItemRain:          itemRainEffectSpec(),
	EffectM01:               m01EffectSpec(),
	EffectM02:               m02EffectSpec(),
	EffectM03:               m03EffectSpec(),
	EffectM04:               m04EffectSpec(),
	EffectM05:               m05EffectSpec(),
	EffectM06:               m06EffectSpec(),
	EffectM07:               m07EffectSpec(),
	EffectKaizel:            kaizelEffectSpec(),
	EffectCloud6:            cloud6EffectSpec(),
	EffectStatFoodSTR:       statFoodSTREffectSpec(),
	EffectStatFoodINT:       statFoodINTEffectSpec(),
	EffectStatFoodVIT:       statFoodVITEffectSpec(),
	EffectStatFoodAGI:       statFoodAGIEffectSpec(),
	EffectStatFoodDEX:       statFoodDEXEffectSpec(),
	EffectStatFoodLUK:       statFoodLUKEffectSpec(),
	EffectShrink:            shrinkEffectSpec(),
	EffectThrowItem6:        throwItem6EffectSpec(),
	EffectSightBlaster:      sightBlasterEffectSpec(),
	EffectQuakeBody4:        closeConfineHitEffectSpec(),
	EffectFireHit2:          fireHit2EffectSpec(),
	EffectNPCStop2:          closeConfineRingEffectSpec(),
	EffectNPCStop2Del:       npcStop2DelEffectSpec(),
	EffectFVoice:            pangVoiceEffectSpec(),
	EffectWink:              winkEffectSpec(),
	EffectCookingOK:         cookingOKEffectSpec(),
	EffectCookingFail:       cookingFailEffectSpec(),
	EffectHapgyeok:          hapgyeokEffectSpec(),
	EffectThrowItem7:        throwItem7EffectSpec(),
	EffectThrowItem8:        throwItem8EffectSpec(),
	EffectThrowItem9:        throwItem9EffectSpec(),
	EffectThrowItem10:       throwItem10EffectSpec(),
	EffectKouenka:           ninjaFirePetalEffectSpec(),
	EffectHyousensou:        ninjaIceSpearEffectSpec(),
	EffectBottomSuiton:      bottomSuitonEffectSpec(),
	EffectStin4:             stin4EffectSpec(),
	EffectThunderStorm2:     thunderStorm2EffectSpec(),
	EffectChemical4:         piercingShotEffectSpec(),
	EffectStin5:             stin5EffectSpec(),
	EffectMadnessBlue:       madnessBlueEffectSpec(),
	EffectMadnessRed:        madnessRedEffectSpec(),
	EffectRGCoin3:           disarmEffectSpec(),
	EffectBash3D5:           bash3D5EffectSpec(),
	EffectChookgi3:          spiritSphereEffectSpec(),
	EffectKirikage:          kirikageEffectSpec(),
	EffectTatami:            tatamiEffectSpec(),
	EffectKasumikiri:        kasumikiriEffectSpec(),
	EffectIssen:             issenEffectSpec(),
	EffectKaen:              kaenEffectSpec(),
	EffectBaku:              bakuEffectSpec(),
	EffectHyousyouraku:      hyousyourakuEffectSpec(),
	EffectDesperado:         desperadoEffectSpec(),
	EffectLightningS:        lightningSEffectSpec(),
	EffectBlindS:            blindSEffectSpec(),
	EffectPoisonS:           poisonSEffectSpec(),
	EffectFreezingS:         freezingSEffectSpec(),
	EffectFlareS:            flareSEffectSpec(),
	EffectRapidShower:       rapidShowerEffectSpec(),
	EffectMagicalBullet:     magicalBulletEffectSpec(),
	EffectSpreadAttack:      spreadAttackEffectSpec(),
	EffectTrackCasting:      trackCastingEffectSpec(),
	EffectTracking:          trackingEffectSpec(),
	EffectTripleAction:      tripleActionEffectSpec(),
	EffectBullseye:          bullseyeEffectSpec(),
	EffectBubbleDrop:        bubbleDropEffectSpec(),
	EffectNPCEarthquake:     npcEarthquakeEffectSpec(),
	EffectDragonFear:        dragonFearEffectSpec(),
	EffectWideBleeding:      wideBleedingEffectSpec(),
	EffectWideConfuse:       wideConfuseEffectSpec(),
	EffectBottomRunner:      bottomRunnerEffectSpec(),
	EffectBottomTransfer:    bottomTransferEffectSpec(),
	EffectBottomEvilLand:    evilLandEffectSpec(),
	EffectGuard3:            guard3EffectSpec(),
	EffectCriticalWound:     criticalWoundEffectSpec(),
	EffectFirecracker2:      firecracker2EffectSpec(),
	EffectFirecracker3:      firecracker3EffectSpec(),
	EffectFirecracker4:      firecracker4EffectSpec(),
	EffectFirecracker5:      firecracker5EffectSpec(),
	EffectFirecracker6:      firecracker6EffectSpec(),
	EffectTorchRed:          torchRedEffectSpec(),
	EffectTorchGreen:        torchGreenEffectSpec(),
	EffectMapGhost:          mapGhostEffectSpec(),
	EffectGlow1:             glow1EffectSpec(),
	EffectGlow2:             glow2EffectSpec(),
	EffectGlow4:             glow4EffectSpec(),
	EffectTorchPurple:       torchPurpleEffectSpec(),
	EffectCloud7:            cloud7EffectSpec(),
	EffectCloud8:            cloud8EffectSpec(),
	EffectFlowerLeaf:        flowerLeafEffectSpec(),
	EffectItem315:           item315EffectSpec(),
	EffectItem316:           item316EffectSpec(),
	EffectItem317:           item317EffectSpec(),
	EffectStormMin:          stormMinEffectSpec(),
	EffectFirecracker7:      firecracker7EffectSpec(),
	EffectBottomBlue:        bottomBlueEffectSpec(),
	EffectBottomBlue2:       bottomBlueEffectSpec(),
	EffectChristmasCarol:    christmasCarolEffectSpec(),
	EffectFirePillarOn2:     judexEffectSpec(),
	EffectForestLight5:      forestLight5EffectSpec(),
	EffectAdoramus:          adoramusEffectSpec(),
	EffectIgnitionBreak:     ignitionBreakEffectSpec(),
	EffectFrostMisty:        frostMistyEffectSpec(),
	EffectCrimsonRock:       crimsonRockEffectSpec(),
	EffectHellInferno:       hellInfernoEffectSpec(),
	EffectMarshOfAbyss:      marshOfAbyssEffectSpec(),
	EffectDragonHowling:     dragonHowlingEffectSpec(),
	EffectEarthWall:         earthWallEffectSpec(),
	EffectChainLightning:    chainLightningEffectSpec(),
	EffectAimedBolt:         aimedBoltEffectSpec(),
	EffectArrowStorm:        arrowStormEffectSpec(),
	EffectLaulamus:          laulamusEffectSpec(),
	EffectLauagnus:          lauagnusEffectSpec(),
	EffectMillenniumShield:  millenniumShieldEffectSpec(),
	EffectConcentration2:    concentration2EffectSpec(),
	EffectGlassWall3:        glassWall3EffectSpec(),
	EffectBerserkPotion2:    berserkPotion2EffectSpec(),
	EffectRolling1:          rolling1EffectSpec(),
	EffectRolling2:          rolling2EffectSpec(),
	EffectRolling3:          rolling3EffectSpec(),
	EffectRolling4:          rolling4EffectSpec(),
	EffectRolling5:          rolling5EffectSpec(),
	EffectRolling6:          rolling6EffectSpec(),
	EffectRolling7:          rolling7EffectSpec(),
	EffectRolling8:          rolling8EffectSpec(),
	EffectRolling9:          rolling9EffectSpec(),
	EffectRolling10:         rolling10EffectSpec(),
	EffectCastSpin2:         castSpin2EffectSpec(),
	EffectCrashAxe:          crashAxeEffectSpec(),
	EffectStasis:            stasisEffectSpec(),
	EffectBottomBasilica2:   bottomBasilica2EffectSpec(),
	EffectRecognized:        recognizedEffectSpec(),
	EffectTetra:             tetraEffectSpec(),
	EffectTetraCasting:      tetraCastingEffectSpec(),
	EffectStretch:           stretchEffectSpec(),
	EffectEnervation:        enervationEffectSpec(),
	EffectEnervation2:       enervation2EffectSpec(),
	EffectEnervation3:       enervation3EffectSpec(),
	EffectEnervation4:       enervation4EffectSpec(),
	EffectEnervation5:       enervation5EffectSpec(),
	EffectEnervation6:       enervation6EffectSpec(),
	EffectBottomManhole:     bottomManholeEffectSpec(),
	EffectManhole:           manholeEffectSpec(),
	EffectForestLight6:      forestLight6EffectSpec(),
	EffectBottomAni:         bottomAniEffectSpec(),
	EffectBottomMaelstrom:   bottomMaelstromEffectSpec(),
	EffectBottomBloodyLust:  bottomBloodyLustEffectSpec(),
	EffectHealN:             healNEffectSpec(),
	EffectChookgiN:          spiritSphereEffectSpec(),
	EffectDance1:            dance1EffectSpec(),
	EffectBotReverb:         botReverbEffectSpec(),
	EffectRainParticle:      rainParticleEffectSpec(),
	EffectChemicalV2:        chemicalV2EffectSpec(),
	EffectBotReverb2:        botReverb2EffectSpec(),
	EffectCirclePower2:      circlePower2EffectSpec(),
	EffectSecra2:            secra2EffectSpec(),
	EffectSprPlant2:         sprPlant2EffectSpec(),
	EffectSprPlant3:         sprPlant3EffectSpec(),
	EffectSprPlant4:         sprPlant4EffectSpec(),
	EffectSprPlant5:         sprPlant5EffectSpec(),
	EffectSprPlant6:         sprPlant6EffectSpec(),
	EffectSprPlant7:         sprPlant7EffectSpec(),
	EffectSprPlant8:         sprPlant8EffectSpec(),
	EffectHeartAsura:        heartAsuraEffectSpec(),
	EffectGlassWall4:        glassWall4EffectSpec(),
	EffectBash3D6:           bash3D6EffectSpec(),
	EffectElectric4:         electric4EffectSpec(),
	EffectTeiHit1T:          teiHit1TEffectSpec(),
	EffectPressure2:         pressure2EffectSpec(),
	EffectPrimeCharge2:      primeCharge2EffectSpec(),
	EffectPrimeCharge3:      primeCharge3EffectSpec(),
	EffectPrimeCharge4:      primeCharge4EffectSpec(),
	EffectFireWall2:         fireWall2EffectSpec(),
	EffectSprPlant10:        sprPlant10EffectSpec(),
	EffectShockwave2:        shockwave2EffectSpec(),
	EffectColdThrow2:        coldThrow2EffectSpec(),
	EffectDemonicFire4:      demonicFire4EffectSpec(),
	EffectPressure3:         pressure3EffectSpec(),
	EffectPoisonMist:        poisonMistEffectSpec(),
	EffectEraserCutter:      eraserCutterEffectSpec(),
	EffectLavaSlide:         lavaSlideEffectSpec(),
	EffectSonicClaw:         sonicClawEffectSpec(),
	EffectTinderBreaker:     tinderBreakerEffectSpec(),
	EffectMidnightFrenzy:    midnightFrenzyEffectSpec(),
	EffectVolcanicAsh:       volcanicAshEffectSpec(),
	EffectRWC2011:           rwc2011EffectSpec(),
	EffectRWC2011Two:        rwc2011TwoEffectSpec(),
	EffectRunMakeOK:         runMakeOKEffectSpec(),
	EffectRunMakeFailure:    runMakeFailureEffectSpec(),
	EffectMIResultMakeOK:    miResultMakeOKEffectSpec(),
	EffectMIResultMakeFail:  miResultMakeFailEffectSpec(),
	EffectAllRayProtect:     allRayProtectEffectSpec(),
	EffectVenomFog:          venomFogEffectSpec(),
	EffectDustStorm:         dustStormEffectSpec(),
	EffectDanceBladeAtk:     danceBladeAtkEffectSpec(),
	EffectInvincibleOff2:    invincibleOff2EffectSpec(),
	EffectDeathSummon:       deathSummonEffectSpec(),
	EffectGCDarkCrow:        gcDarkCrowEffectSpec(),
	EffectAllFullThrottle:   allFullThrottleEffectSpec(),
	EffectSRFlashCombo:      srFlashComboEffectSpec(),
	EffectRKLuxAnima:        rkLuxAnimaEffectSpec(),
	EffectSOElemShield:      soElemShieldEffectSpec(),
	EffectABOffertorium:     abOffertoriumEffectSpec(),
	EffectWLTelekinesis:     wlTelekinesisEffectSpec(),
	EffectGNIllusionDoping:  gnIllusionDopingEffectSpec(),
	EffectNCMagmaEruption:   ncMagmaEruptionEffectSpec(),
	EffectNPCChill:          npcChillEffectSpec(),
	EffectOffertoriumRing:   offertoriumRingEffectSpec(),
	EffectHammerOfGod:       hammerOfGodEffectSpec(),
	EffectAchComplete:       achCompleteEffectSpec(),
	EffectBodyColor:         bodyColorEffectSpec(),
	EffectBakuretsuHadou:    bakuretsuHadouEffectSpec(),
	EffectDropPink:          dropPinkEffectSpec(),
	EffectDropYellow:        dropYellowEffectSpec(),
	EffectDropPurple:        dropPurpleEffectSpec(),
	EffectDigitalSpace:      digitalSpaceEffectSpec(),
	EffectDropBlue:          dropBlueEffectSpec(),
	EffectDropGreen:         dropGreenEffectSpec(),
	EffectDropRed:           dropRedEffectSpec(),
	EffectNewSuccess:        newSuccessEffectSpec(),
	EffectNewFailure:        newFailureEffectSpec(),
	EffectNewIntro:          newIntroEffectSpec(),
	EffectEnchantYellow:     enchantYellowEffectSpec(),
	EffectEnchantSuccess:    enchantSuccessEffectSpec(),
	EffectEnchantFail:       enchantFailEffectSpec(),
	EffectEnchantBlue:       enchantBlueEffectSpec(),
	EffectEnchantUpSuccess:  enchantUpSuccessEffectSpec(),
	EffectEnchantUpFail:     enchantUpFailEffectSpec(),
	EffectEnchantGreen:      enchantGreenEffectSpec(),
	EffectEnchantResetOK:    enchantResetOKEffectSpec(),
	EffectEnchantResetFail:  enchantResetFailEffectSpec(),

	EffectColdBolt:               coldBoltEffectSpec(),
	EffectFireBolt:               fireBoltEffectSpec(),
	EffectCastRing:               castRingEffectSpec(),
	EffectQuakeMagnum:            quakeMagnumEffectSpec(),
	EffectArrowShot:              arrowShotEffectSpec(),
	EffectArrowShower:            arrowShowerEffectSpec(),
	EffectMagicPower:             magicPowerEffectSpec(),
	EffectWhitePulse:             whitePulseEffectSpec(),
	EffectSpearHitSound:          spearHitSoundEffectSpec(),
	EffectEnemyHitNormal1:        enemyHitNormal1EffectSpec(),
	EffectQuake:                  quakeEffectSpec(),
	EffectSharpShootingCast:      sharpShootingCastEffectSpec(),
	EffectAdrenalineCast:         adrenalineCastEffectSpec(),
	EffectMaximizeSounds:         maximizeSoundsEffectSpec(),
	EffectGreedSound:             greedSoundEffectSpec(),
	EffectGospelGround:           gospelGroundEffectSpec(),
	EffectHermodeMusic:           hermodeMusicEffectSpec(),
	EffectBottomDissonanceGround: bottomDissonanceEffectSpec(),
	EffectBottomLullabyGround:    bottomLullabyEffectSpec(),
	EffectBottomRichKimGround:    bottomRichKimEffectSpec(),
	EffectBottomChaosGround:      bottomChaosEffectSpec(),
	EffectBottomDrumGround:       bottomDrumEffectSpec(),
	EffectBottomNibelungGround:   bottomNibelungEffectSpec(),
	EffectBottomRokiGround:       songOutEffectSpec(),
	EffectBottomAbyssGround:      bottomAbyssEffectSpec(),
	EffectBottomSiegGround:       bottomSiegEffectSpec(),
	EffectBottomWhistleGround:    bottomWhistleEffectSpec(),
	EffectBottomSinXGround:       bottomSinXEffectSpec(),
	EffectBottomBragiGround:      bottomBragiEffectSpec(),
	EffectBottomAppleGround:      bottomAppleEffectSpec(),
	EffectBottomUglyDanceGround:  bottomUglyDanceEffectSpec(),
	EffectBottomHummingGround:    bottomHummingEffectSpec(),
	EffectBottomForgetGround:     bottomForgetEffectSpec(),
	EffectBottomFortuneGround:    bottomFortuneEffectSpec(),
	EffectBottomServiceGround:    bottomServiceEffectSpec(),
	EffectGunslingerCoinSound:    gunslingerCoinSoundEffectSpec(),
	EffectSteelBodyRing:          steelBodyRingEffectSpec(),
	SkillEffectChaseWalkSmoke:    chaseWalkSmokeEffectSpec(),
}
