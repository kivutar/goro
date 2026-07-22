// Imported from robr DB/Skills tables.
package db

const (
	SkillNVBasic                            uint16 = 1
	SkillSMSword                            uint16 = 2
	SkillSMTwohand                          uint16 = 3
	SkillSMRecovery                         uint16 = 4
	SkillSMBash                             uint16 = 5
	SkillSMProvoke                          uint16 = 6
	SkillSMMagnum                           uint16 = 7
	SkillSMEndure                           uint16 = 8
	SkillMGSrecovery                        uint16 = 9
	SkillMGSight                            uint16 = 10
	SkillMGNapalmbeat                       uint16 = 11
	SkillMGSafetywall                       uint16 = 12
	SkillMGSoulstrike                       uint16 = 13
	SkillMGColdbolt                         uint16 = 14
	SkillMGFrostdiver                       uint16 = 15
	SkillMGStonecurse                       uint16 = 16
	SkillMGFireball                         uint16 = 17
	SkillMGFirewall                         uint16 = 18
	SkillMGFirebolt                         uint16 = 19
	SkillMGLightningbolt                    uint16 = 20
	SkillMGThunderstorm                     uint16 = 21
	SkillALDp                               uint16 = 22
	SkillALDemonbane                        uint16 = 23
	SkillALRuwach                           uint16 = 24
	SkillALPneuma                           uint16 = 25
	SkillALTeleport                         uint16 = 26
	SkillALWarp                             uint16 = 27
	SkillALHeal                             uint16 = 28
	SkillALIncagi                           uint16 = 29
	SkillALDecagi                           uint16 = 30
	SkillALHolywater                        uint16 = 31
	SkillALCrucis                           uint16 = 32
	SkillALAngelus                          uint16 = 33
	SkillALBlessing                         uint16 = 34
	SkillALCure                             uint16 = 35
	SkillMCInccarry                         uint16 = 36
	SkillMCDiscount                         uint16 = 37
	SkillMCOvercharge                       uint16 = 38
	SkillMCPushcart                         uint16 = 39
	SkillMCIdentify                         uint16 = 40
	SkillMCVending                          uint16 = 41
	SkillMCMammonite                        uint16 = 42
	SkillACOwl                              uint16 = 43
	SkillACVulture                          uint16 = 44
	SkillACConcentration                    uint16 = 45
	SkillACDouble                           uint16 = 46
	SkillACShower                           uint16 = 47
	SkillTFDouble                           uint16 = 48
	SkillTFMiss                             uint16 = 49
	SkillTFSteal                            uint16 = 50
	SkillTFHiding                           uint16 = 51
	SkillTFPoison                           uint16 = 52
	SkillTFDetoxify                         uint16 = 53
	SkillALLResurrection                    uint16 = 54
	SkillKNSpearmastery                     uint16 = 55
	SkillKNPierce                           uint16 = 56
	SkillKNBrandishspear                    uint16 = 57
	SkillKNSpearstab                        uint16 = 58
	SkillKNSpearboomerang                   uint16 = 59
	SkillKNTwohandquicken                   uint16 = 60
	SkillKNAutocounter                      uint16 = 61
	SkillKNBowlingbash                      uint16 = 62
	SkillKNRiding                           uint16 = 63
	SkillKNCavaliermastery                  uint16 = 64
	SkillPRMacemastery                      uint16 = 65
	SkillPRImpositio                        uint16 = 66
	SkillPRSuffragium                       uint16 = 67
	SkillPRAspersio                         uint16 = 68
	SkillPRBenedictio                       uint16 = 69
	SkillPRSanctuary                        uint16 = 70
	SkillPRSlowpoison                       uint16 = 71
	SkillPRStrecovery                       uint16 = 72
	SkillPRKyrie                            uint16 = 73
	SkillPRMagnificat                       uint16 = 74
	SkillPRGloria                           uint16 = 75
	SkillPRLexdivina                        uint16 = 76
	SkillPRTurnundead                       uint16 = 77
	SkillPRLexaeterna                       uint16 = 78
	SkillPRMagnus                           uint16 = 79
	SkillWZFirepillar                       uint16 = 80
	SkillWZSightrasher                      uint16 = 81
	SkillWZFireivy                          uint16 = 82
	SkillWZMeteor                           uint16 = 83
	SkillWZJupitel                          uint16 = 84
	SkillWZVermilion                        uint16 = 85
	SkillWZWaterball                        uint16 = 86
	SkillWZIcewall                          uint16 = 87
	SkillWZFrostnova                        uint16 = 88
	SkillWZStormgust                        uint16 = 89
	SkillWZEarthspike                       uint16 = 90
	SkillWZHeavendrive                      uint16 = 91
	SkillWZQuagmire                         uint16 = 92
	SkillWZEstimation                       uint16 = 93
	SkillBSIron                             uint16 = 94
	SkillBSSteel                            uint16 = 95
	SkillBSEnchantedstone                   uint16 = 96
	SkillBSOrideocon                        uint16 = 97
	SkillBSDagger                           uint16 = 98
	SkillBSSword                            uint16 = 99
	SkillBSTwohandsword                     uint16 = 100
	SkillBSAxe                              uint16 = 101
	SkillBSMace                             uint16 = 102
	SkillBSKnuckle                          uint16 = 103
	SkillBSSpear                            uint16 = 104
	SkillBSHiltbinding                      uint16 = 105
	SkillBSFindingore                       uint16 = 106
	SkillBSWeaponresearch                   uint16 = 107
	SkillBSRepairweapon                     uint16 = 108
	SkillBSSkintemper                       uint16 = 109
	SkillBSHammerfall                       uint16 = 110
	SkillBSAdrenaline                       uint16 = 111
	SkillBSWeaponperfect                    uint16 = 112
	SkillBSOverthrust                       uint16 = 113
	SkillBSMaximize                         uint16 = 114
	SkillHTSkidtrap                         uint16 = 115
	SkillHTLandmine                         uint16 = 116
	SkillHTAnklesnare                       uint16 = 117
	SkillHTShockwave                        uint16 = 118
	SkillHTSandman                          uint16 = 119
	SkillHTFlasher                          uint16 = 120
	SkillHTFreezingtrap                     uint16 = 121
	SkillHTBlastmine                        uint16 = 122
	SkillHTClaymoretrap                     uint16 = 123
	SkillHTRemovetrap                       uint16 = 124
	SkillHTTalkiebox                        uint16 = 125
	SkillHTBeastbane                        uint16 = 126
	SkillHTFalcon                           uint16 = 127
	SkillHTSteelcrow                        uint16 = 128
	SkillHTBlitzbeat                        uint16 = 129
	SkillHTDetecting                        uint16 = 130
	SkillHTSpringtrap                       uint16 = 131
	SkillASRight                            uint16 = 132
	SkillASLeft                             uint16 = 133
	SkillASKatar                            uint16 = 134
	SkillASCloaking                         uint16 = 135
	SkillASSonicblow                        uint16 = 136
	SkillASGrimtooth                        uint16 = 137
	SkillASEnchantpoison                    uint16 = 138
	SkillASPoisonreact                      uint16 = 139
	SkillASVenomdust                        uint16 = 140
	SkillASSplasher                         uint16 = 141
	SkillNVFirstaid                         uint16 = 142
	SkillNVTrickdead                        uint16 = 143
	SkillSMMovingrecovery                   uint16 = 144
	SkillSMFatalblow                        uint16 = 145
	SkillSMAutoberserk                      uint16 = 146
	SkillACMakingarrow                      uint16 = 147
	SkillACChargearrow                      uint16 = 148
	SkillTFSprinklesand                     uint16 = 149
	SkillTFBacksliding                      uint16 = 150
	SkillTFPickstone                        uint16 = 151
	SkillTFThrowstone                       uint16 = 152
	SkillMCCartrevolution                   uint16 = 153
	SkillMCChangecart                       uint16 = 154
	SkillMCLoud                             uint16 = 155
	SkillALHolylight                        uint16 = 156
	SkillMGEnergycoat                       uint16 = 157
	SkillNPCPiercingatt                     uint16 = 158
	SkillNPCMentalbreaker                   uint16 = 159
	SkillNPCRangeattack                     uint16 = 160
	SkillNPCAttrichange                     uint16 = 161
	SkillNPCChangewater                     uint16 = 162
	SkillNPCChangeground                    uint16 = 163
	SkillNPCChangefire                      uint16 = 164
	SkillNPCChangewind                      uint16 = 165
	SkillNPCChangepoison                    uint16 = 166
	SkillNPCChangeholy                      uint16 = 167
	SkillNPCChangedarkness                  uint16 = 168
	SkillNPCChangetelekinesis               uint16 = 169
	SkillNPCCriticalslash                   uint16 = 170
	SkillNPCComboattack                     uint16 = 171
	SkillNPCGuidedattack                    uint16 = 172
	SkillNPCSelfdestruction                 uint16 = 173
	SkillNPCSplashattack                    uint16 = 174
	SkillNPCSuicide                         uint16 = 175
	SkillNPCPoison                          uint16 = 176
	SkillNPCBlindattack                     uint16 = 177
	SkillNPCSilenceattack                   uint16 = 178
	SkillNPCStunattack                      uint16 = 179
	SkillNPCPetrifyattack                   uint16 = 180
	SkillNPCCurseattack                     uint16 = 181
	SkillNPCSleepattack                     uint16 = 182
	SkillNPCRandomattack                    uint16 = 183
	SkillNPCWaterattack                     uint16 = 184
	SkillNPCGroundattack                    uint16 = 185
	SkillNPCFireattack                      uint16 = 186
	SkillNPCWindattack                      uint16 = 187
	SkillNPCPoisonattack                    uint16 = 188
	SkillNPCHolyattack                      uint16 = 189
	SkillNPCDarknessattack                  uint16 = 190
	SkillNPCTelekinesisattack               uint16 = 191
	SkillNPCMagicalattack                   uint16 = 192
	SkillNPCMetamorphosis                   uint16 = 193
	SkillNPCProvocation                     uint16 = 194
	SkillNPCSmoking                         uint16 = 195
	SkillNPCSummonslave                     uint16 = 196
	SkillNPCEmotion                         uint16 = 197
	SkillNPCTransformation                  uint16 = 198
	SkillNPCBlooddrain                      uint16 = 199
	SkillNPCEnergydrain                     uint16 = 200
	SkillNPCKeeping                         uint16 = 201
	SkillNPCDarkbreath                      uint16 = 202
	SkillNPCDarkblessing                    uint16 = 203
	SkillNPCBarrier                         uint16 = 204
	SkillNPCDefender                        uint16 = 205
	SkillNPCLick                            uint16 = 206
	SkillNPCHallucination                   uint16 = 207
	SkillNPCRebirth                         uint16 = 208
	SkillNPCSummonmonster                   uint16 = 209
	SkillRGSnatcher                         uint16 = 210
	SkillRGStealcoin                        uint16 = 211
	SkillRGBackstap                         uint16 = 212
	SkillRGTunneldrive                      uint16 = 213
	SkillRGRaid                             uint16 = 214
	SkillRGStripweapon                      uint16 = 215
	SkillRGStripshield                      uint16 = 216
	SkillRGStriparmor                       uint16 = 217
	SkillRGStriphelm                        uint16 = 218
	SkillRGIntimidate                       uint16 = 219
	SkillRGGraffiti                         uint16 = 220
	SkillRGFlaggraffiti                     uint16 = 221
	SkillRGCleaner                          uint16 = 222
	SkillRGGangster                         uint16 = 223
	SkillRGCompulsion                       uint16 = 224
	SkillRGPlagiarism                       uint16 = 225
	SkillAMAxemastery                       uint16 = 226
	SkillAMLearningpotion                   uint16 = 227
	SkillAMPharmacy                         uint16 = 228
	SkillAMDemonstration                    uint16 = 229
	SkillAMAcidterror                       uint16 = 230
	SkillAMPotionpitcher                    uint16 = 231
	SkillAMCannibalize                      uint16 = 232
	SkillAMSpheremine                       uint16 = 233
	SkillAMCpWeapon                         uint16 = 234
	SkillAMCpShield                         uint16 = 235
	SkillAMCpArmor                          uint16 = 236
	SkillAMCpHelm                           uint16 = 237
	SkillAMBioethics                        uint16 = 238
	SkillAMBiotechnology                    uint16 = 239
	SkillAMCreatecreature                   uint16 = 240
	SkillAMCultivation                      uint16 = 241
	SkillAMFlamecontrol                     uint16 = 242
	SkillAMCallhomun                        uint16 = 243
	SkillAMRest                             uint16 = 244
	SkillAMDrillmaster                      uint16 = 245
	SkillAMHealhomun                        uint16 = 246
	SkillAMResurrecthomun                   uint16 = 247
	SkillCRTrust                            uint16 = 248
	SkillCRAutoguard                        uint16 = 249
	SkillCRShieldcharge                     uint16 = 250
	SkillCRShieldboomerang                  uint16 = 251
	SkillCRReflectshield                    uint16 = 252
	SkillCRHolycross                        uint16 = 253
	SkillCRGrandcross                       uint16 = 254
	SkillCRDevotion                         uint16 = 255
	SkillCRProvidence                       uint16 = 256
	SkillCRDefender                         uint16 = 257
	SkillCRSpearquicken                     uint16 = 258
	SkillMOIronhand                         uint16 = 259
	SkillMOSpiritsrecovery                  uint16 = 260
	SkillMOCallspirits                      uint16 = 261
	SkillMOAbsorbspirits                    uint16 = 262
	SkillMOTripleattack                     uint16 = 263
	SkillMOBodyrelocation                   uint16 = 264
	SkillMODodge                            uint16 = 265
	SkillMOInvestigate                      uint16 = 266
	SkillMOFingeroffensive                  uint16 = 267
	SkillMOSteelbody                        uint16 = 268
	SkillMOBladestop                        uint16 = 269
	SkillMOExplosionspirits                 uint16 = 270
	SkillMOExtremityfist                    uint16 = 271
	SkillMOChaincombo                       uint16 = 272
	SkillMOCombofinish                      uint16 = 273
	SkillSAAdvancedbook                     uint16 = 274
	SkillSACastcancel                       uint16 = 275
	SkillSAMagicrod                         uint16 = 276
	SkillSASpellbreaker                     uint16 = 277
	SkillSAFreecast                         uint16 = 278
	SkillSAAutospell                        uint16 = 279
	SkillSAFlamelauncher                    uint16 = 280
	SkillSAFrostweapon                      uint16 = 281
	SkillSALightningloader                  uint16 = 282
	SkillSASeismicweapon                    uint16 = 283
	SkillSADragonology                      uint16 = 284
	SkillSAVolcano                          uint16 = 285
	SkillSADeluge                           uint16 = 286
	SkillSAViolentgale                      uint16 = 287
	SkillSALandprotector                    uint16 = 288
	SkillSADispell                          uint16 = 289
	SkillSAAbracadabra                      uint16 = 290
	SkillSAMonocell                         uint16 = 291
	SkillSAClasschange                      uint16 = 292
	SkillSASummonmonster                    uint16 = 293
	SkillSAReverseorcish                    uint16 = 294
	SkillSADeath                            uint16 = 295
	SkillSAFortune                          uint16 = 296
	SkillSATamingmonster                    uint16 = 297
	SkillSAQuestion                         uint16 = 298
	SkillSAGravity                          uint16 = 299
	SkillSALevelup                          uint16 = 300
	SkillSAInstantdeath                     uint16 = 301
	SkillSAFullrecovery                     uint16 = 302
	SkillSAComa                             uint16 = 303
	SkillBDAdaptation                       uint16 = 304
	SkillBDEncore                           uint16 = 305
	SkillBDLullaby                          uint16 = 306
	SkillBDRichmankim                       uint16 = 307
	SkillBDEternalchaos                     uint16 = 308
	SkillBDDrumbattlefield                  uint16 = 309
	SkillBDRingnibelungen                   uint16 = 310
	SkillBDRokisweil                        uint16 = 311
	SkillBDIntoabyss                        uint16 = 312
	SkillBDSiegfried                        uint16 = 313
	SkillBDRagnarok                         uint16 = 314
	SkillBaMusicallesson                    uint16 = 315
	SkillBaMusicalstrike                    uint16 = 316
	SkillBaDissonance                       uint16 = 317
	SkillBaFrostjoke                        uint16 = 318
	SkillBaWhistle                          uint16 = 319
	SkillBaAssassincross                    uint16 = 320
	SkillBaPoembragi                        uint16 = 321
	SkillBaAppleidun                        uint16 = 322
	SkillDCDancinglesson                    uint16 = 323
	SkillDCThrowarrow                       uint16 = 324
	SkillDCUglydance                        uint16 = 325
	SkillDCScream                           uint16 = 326
	SkillDCHumming                          uint16 = 327
	SkillDCDontforgetme                     uint16 = 328
	SkillDCFortunekiss                      uint16 = 329
	SkillDCServiceforyou                    uint16 = 330
	SkillNPCRandommove                      uint16 = 331
	SkillNPCSpeedup                         uint16 = 332
	SkillNPCRevenge                         uint16 = 333
	SkillWEMale                             uint16 = 334
	SkillWEFemale                           uint16 = 335
	SkillWECallpartner                      uint16 = 336
	SkillITMTomahawk                        uint16 = 337
	SkillNPCDarkcross                       uint16 = 338
	SkillNPCGranddarkness                   uint16 = 339
	SkillNPCDarkstrike                      uint16 = 340
	SkillNPCDarkthunder                     uint16 = 341
	SkillNPCStop                            uint16 = 342
	SkillNPCWeaponbraker                    uint16 = 343
	SkillNPCArmorbrake                      uint16 = 344
	SkillNPCHelmbrake                       uint16 = 345
	SkillNPCShieldbrake                     uint16 = 346
	SkillNPCUndeadattack                    uint16 = 347
	SkillNPCChangeundead                    uint16 = 348
	SkillNPCPowerup                         uint16 = 349
	SkillNPCAgiup                           uint16 = 350
	SkillNPCSiegemode                       uint16 = 351
	SkillNPCCallslave                       uint16 = 352
	SkillNPCInvisible                       uint16 = 353
	SkillNPCRun                             uint16 = 354
	SkillLKAurablade                        uint16 = 355
	SkillLKParrying                         uint16 = 356
	SkillLKConcentration                    uint16 = 357
	SkillLKTensionrelax                     uint16 = 358
	SkillLKBerserk                          uint16 = 359
	SkillLKFury                             uint16 = 360
	SkillHPAssumptio                        uint16 = 361
	SkillHPBasilica                         uint16 = 362
	SkillHPMeditatio                        uint16 = 363
	SkillHWSouldrain                        uint16 = 364
	SkillHWMagiccrasher                     uint16 = 365
	SkillHWMagicpower                       uint16 = 366
	SkillPaPressure                         uint16 = 367
	SkillPaSacrifice                        uint16 = 368
	SkillPaGospel                           uint16 = 369
	SkillChPalmstrike                       uint16 = 370
	SkillChTigerfist                        uint16 = 371
	SkillChChaincrush                       uint16 = 372
	SkillPFHpconversion                     uint16 = 373
	SkillPFSoulchange                       uint16 = 374
	SkillPFSoulburn                         uint16 = 375
	SkillASCKatar                           uint16 = 376
	SkillASCHallucination                   uint16 = 377
	SkillASCEdp                             uint16 = 378
	SkillASCBreaker                         uint16 = 379
	SkillSNSight                            uint16 = 380
	SkillSNFalconassault                    uint16 = 381
	SkillSNSharpshooting                    uint16 = 382
	SkillSNWindwalk                         uint16 = 383
	SkillWSMeltdown                         uint16 = 384
	SkillWSCreatecoin                       uint16 = 385
	SkillWSCreatenugget                     uint16 = 386
	SkillWSCartboost                        uint16 = 387
	SkillWSSystemcreate                     uint16 = 388
	SkillSTChasewalk                        uint16 = 389
	SkillSTRejectsword                      uint16 = 390
	SkillSTStealbackpack                    uint16 = 391
	SkillCRAlchemy                          uint16 = 392
	SkillCRSynthesispotion                  uint16 = 393
	SkillCGArrowvulcan                      uint16 = 394
	SkillCGMoonlit                          uint16 = 395
	SkillCGMarionette                       uint16 = 396
	SkillLKSpiralpierce                     uint16 = 397
	SkillLKHeadcrush                        uint16 = 398
	SkillLKJointbeat                        uint16 = 399
	SkillHWNapalmvulcan                     uint16 = 400
	SkillChSoulcollect                      uint16 = 401
	SkillPFMindbreaker                      uint16 = 402
	SkillPFMemorize                         uint16 = 403
	SkillPFFogwall                          uint16 = 404
	SkillPFSpiderweb                        uint16 = 405
	SkillASCMeteorassault                   uint16 = 406
	SkillASCCdp                             uint16 = 407
	SkillWEBaby                             uint16 = 408
	SkillWECallparent                       uint16 = 409
	SkillWECallbaby                         uint16 = 410
	SkillTKRun                              uint16 = 411
	SkillTKReadystorm                       uint16 = 412
	SkillTKStormkick                        uint16 = 413
	SkillTKReadydown                        uint16 = 414
	SkillTKDownkick                         uint16 = 415
	SkillTKReadyturn                        uint16 = 416
	SkillTKTurnkick                         uint16 = 417
	SkillTKReadycounter                     uint16 = 418
	SkillTKCounter                          uint16 = 419
	SkillTKDodge                            uint16 = 420
	SkillTKJumpkick                         uint16 = 421
	SkillTKHptime                           uint16 = 422
	SkillTKSptime                           uint16 = 423
	SkillTKPower                            uint16 = 424
	SkillTKSevenwind                        uint16 = 425
	SkillTKHighjump                         uint16 = 426
	SkillSGFeel                             uint16 = 427
	SkillSGSunWarm                          uint16 = 428
	SkillSGMoonWarm                         uint16 = 429
	SkillSGStarWarm                         uint16 = 430
	SkillSGSunComfort                       uint16 = 431
	SkillSGMoonComfort                      uint16 = 432
	SkillSGStarComfort                      uint16 = 433
	SkillSGHate                             uint16 = 434
	SkillSGSunAnger                         uint16 = 435
	SkillSGMoonAnger                        uint16 = 436
	SkillSGStarAnger                        uint16 = 437
	SkillSGSunBless                         uint16 = 438
	SkillSGMoonBless                        uint16 = 439
	SkillSGStarBless                        uint16 = 440
	SkillSGDevil                            uint16 = 441
	SkillSGFriend                           uint16 = 442
	SkillSGKnowledge                        uint16 = 443
	SkillSGFusion                           uint16 = 444
	SkillSLAlchemist                        uint16 = 445
	SkillAMBerserkpitcher                   uint16 = 446
	SkillSLMonk                             uint16 = 447
	SkillSLStar                             uint16 = 448
	SkillSLSage                             uint16 = 449
	SkillSLCrusader                         uint16 = 450
	SkillSLSupernovice                      uint16 = 451
	SkillSLKnight                           uint16 = 452
	SkillSLWizard                           uint16 = 453
	SkillSLPriest                           uint16 = 454
	SkillSLBarddancer                       uint16 = 455
	SkillSLRogue                            uint16 = 456
	SkillSLAssasin                          uint16 = 457
	SkillSLBlacksmith                       uint16 = 458
	SkillBSAdrenaline2                      uint16 = 459
	SkillSLHunter                           uint16 = 460
	SkillSLSoullinker                       uint16 = 461
	SkillSLKaizel                           uint16 = 462
	SkillSLKaahi                            uint16 = 463
	SkillSLKaupe                            uint16 = 464
	SkillSLKaite                            uint16 = 465
	SkillSLKaina                            uint16 = 466
	SkillSLStin                             uint16 = 467
	SkillSLStun                             uint16 = 468
	SkillSLSma                              uint16 = 469
	SkillSLSwoo                             uint16 = 470
	SkillSLSke                              uint16 = 471
	SkillSLSka                              uint16 = 472
	SkillSMSelfprovoke                      uint16 = 473
	SkillNPCEmotionOn                       uint16 = 474
	SkillSTPreserve                         uint16 = 475
	SkillSTFullstrip                        uint16 = 476
	SkillWSWeaponrefine                     uint16 = 477
	SkillCRSlimpitcher                      uint16 = 478
	SkillCRFullprotection                   uint16 = 479
	SkillPaShieldchain                      uint16 = 480
	SkillHPManarecharge                     uint16 = 481
	SkillPFDoublecasting                    uint16 = 482
	SkillHWGanbantein                       uint16 = 483
	SkillHWGravitation                      uint16 = 484
	SkillWSCarttermination                  uint16 = 485
	SkillWSOverthrustmax                    uint16 = 486
	SkillCGLongingfreedom                   uint16 = 487
	SkillCGHermode                          uint16 = 488
	SkillCGTarotcard                        uint16 = 489
	SkillCRAciddemonstration                uint16 = 490
	SkillCRCultivation                      uint16 = 491
	SkillItemEnchantarms                    uint16 = 492
	SkillTKMission                          uint16 = 493
	SkillSLHigh                             uint16 = 494
	SkillKNOnehand                          uint16 = 495
	SkillAMTwilight1                        uint16 = 496
	SkillAMTwilight2                        uint16 = 497
	SkillAMTwilight3                        uint16 = 498
	SkillHTPower                            uint16 = 499
	SkillGSGlittering                       uint16 = 500
	SkillGSFling                            uint16 = 501
	SkillGSTripleaction                     uint16 = 502
	SkillGSBullseye                         uint16 = 503
	SkillGSMadnesscancel                    uint16 = 504
	SkillGSAdjustment                       uint16 = 505
	SkillGSIncreasing                       uint16 = 506
	SkillGSMagicalbullet                    uint16 = 507
	SkillGSCracker                          uint16 = 508
	SkillGSSingleaction                     uint16 = 509
	SkillGSSnakeeye                         uint16 = 510
	SkillGSChainaction                      uint16 = 511
	SkillGSTracking                         uint16 = 512
	SkillGSDisarm                           uint16 = 513
	SkillGSPiercingshot                     uint16 = 514
	SkillGSRapidshower                      uint16 = 515
	SkillGSDesperado                        uint16 = 516
	SkillGSGatlingfever                     uint16 = 517
	SkillGSDust                             uint16 = 518
	SkillGSFullbuster                       uint16 = 519
	SkillGSSpreadattack                     uint16 = 520
	SkillGSGrounddrift                      uint16 = 521
	SkillNJTobidougu                        uint16 = 522
	SkillNJSyuriken                         uint16 = 523
	SkillNJKunai                            uint16 = 524
	SkillNJHuuma                            uint16 = 525
	SkillNJZenynage                         uint16 = 526
	SkillNJTatamigaeshi                     uint16 = 527
	SkillNJKasumikiri                       uint16 = 528
	SkillNJShadowjump                       uint16 = 529
	SkillNJKirikage                         uint16 = 530
	SkillNJUtsusemi                         uint16 = 531
	SkillNJBunsinjyutsu                     uint16 = 532
	SkillNJNinpou                           uint16 = 533
	SkillNJKouenka                          uint16 = 534
	SkillNJKaensin                          uint16 = 535
	SkillNJBakuenryu                        uint16 = 536
	SkillNJHyousensou                       uint16 = 537
	SkillNJSuiton                           uint16 = 538
	SkillNJHyousyouraku                     uint16 = 539
	SkillNJHuujin                           uint16 = 540
	SkillNJRaigekisai                       uint16 = 541
	SkillNJKamaitachi                       uint16 = 542
	SkillNJNen                              uint16 = 543
	SkillNJIssen                            uint16 = 544
	SkillMbFighting                         uint16 = 545
	SkillMbNeutral                          uint16 = 546
	SkillMbTaimingPuti                      uint16 = 547
	SkillMbWhitepotion                      uint16 = 548
	SkillMbMental                           uint16 = 549
	SkillMbCardpitcher                      uint16 = 550
	SkillMbPetpitcher                       uint16 = 551
	SkillMbBodystudy                        uint16 = 552
	SkillMbBodyalter                        uint16 = 553
	SkillMbPetmemory                        uint16 = 554
	SkillMbMTeleport                        uint16 = 555
	SkillMbBGain                            uint16 = 556
	SkillMbMGain                            uint16 = 557
	SkillMbMission                          uint16 = 558
	SkillMbMunakknowledge                   uint16 = 559
	SkillMbMunakball                        uint16 = 560
	SkillMbScroll                           uint16 = 561
	SkillMbBGathering                       uint16 = 562
	SkillMbMGathering                       uint16 = 563
	SkillMbBExclude                         uint16 = 564
	SkillMbBDrift                           uint16 = 565
	SkillMbBWallrush                        uint16 = 566
	SkillMbMWallrush                        uint16 = 567
	SkillMbBWallshift                       uint16 = 568
	SkillMbMWallcrash                       uint16 = 569
	SkillMbMReincarnation                   uint16 = 570
	SkillMbBEquip                           uint16 = 571
	SkillSLDeathknight                      uint16 = 572
	SkillSLCollector                        uint16 = 573
	SkillSLNinja                            uint16 = 574
	SkillSLGunner                           uint16 = 575
	SkillAMTwilight4                        uint16 = 576
	SkillDaReset                            uint16 = 577
	SkillDeBerserkaizer                     uint16 = 578
	SkillDaDarkpower                        uint16 = 579
	SkillDePassive                          uint16 = 580
	SkillDePattack                          uint16 = 581
	SkillDePspeed                           uint16 = 582
	SkillDePdefense                         uint16 = 583
	SkillDePcritical                        uint16 = 584
	SkillDePhp                              uint16 = 585
	SkillDePsp                              uint16 = 586
	SkillDeReset                            uint16 = 587
	SkillDeRanking                          uint16 = 588
	SkillDePtriple                          uint16 = 589
	SkillDeEnergy                           uint16 = 590
	SkillDeNightmare                        uint16 = 591
	SkillDeSlash                            uint16 = 592
	SkillDeCoil                             uint16 = 593
	SkillDeWave                             uint16 = 594
	SkillDeRebirth                          uint16 = 595
	SkillDeAura                             uint16 = 596
	SkillDeFreezer                          uint16 = 597
	SkillDeChangeattack                     uint16 = 598
	SkillDePunish                           uint16 = 599
	SkillDePoison                           uint16 = 600
	SkillDeInstant                          uint16 = 601
	SkillDeWarning                          uint16 = 602
	SkillDeRankedknife                      uint16 = 603
	SkillDeRankedgradius                    uint16 = 604
	SkillDeGauge                            uint16 = 605
	SkillDeGtime                            uint16 = 606
	SkillDeGpain                            uint16 = 607
	SkillDeGskill                           uint16 = 608
	SkillDeGkill                            uint16 = 609
	SkillDeAccel                            uint16 = 610
	SkillDeBlockdouble                      uint16 = 611
	SkillDeBlockmelee                       uint16 = 612
	SkillDeBlockfar                         uint16 = 613
	SkillDeFrontattack                      uint16 = 614
	SkillDeDangerattack                     uint16 = 615
	SkillDeTwinattack                       uint16 = 616
	SkillDeWindattack                       uint16 = 617
	SkillDeWaterattack                      uint16 = 618
	SkillDaEnergy                           uint16 = 619
	SkillDaCloud                            uint16 = 620
	SkillDaFirstslot                        uint16 = 621
	SkillDaHeaddef                          uint16 = 622
	SkillDaSpace                            uint16 = 623
	SkillDaTransform                        uint16 = 624
	SkillDaExplosion                        uint16 = 625
	SkillDaReward                           uint16 = 626
	SkillDaCrush                            uint16 = 627
	SkillDaItemrebuild                      uint16 = 628
	SkillDaIllusion                         uint16 = 629
	SkillDaNuetralize                       uint16 = 630
	SkillDaRunner                           uint16 = 631
	SkillDaTransfer                         uint16 = 632
	SkillDaWall                             uint16 = 633
	SkillDaZeny                             uint16 = 634
	SkillDaRevenge                          uint16 = 635
	SkillDaEarplug                          uint16 = 636
	SkillDaContract                         uint16 = 637
	SkillDaBlack                            uint16 = 638
	SkillDaDream                            uint16 = 639
	SkillDaMagiccart                        uint16 = 640
	SkillDaCopy                             uint16 = 641
	SkillDaCrystal                          uint16 = 642
	SkillDaExp                              uint16 = 643
	SkillDaCartswing                        uint16 = 644
	SkillDaRebuild                          uint16 = 645
	SkillDaJobchange                        uint16 = 646
	SkillDaEdarkness                        uint16 = 647
	SkillDaEguardian                        uint16 = 648
	SkillDaTimeout                          uint16 = 649
	SkillALLTimein                          uint16 = 650
	SkillDaZenyrank                         uint16 = 651
	SkillDaAccessorymix                     uint16 = 652
	SkillNPCEarthquake                      uint16 = 653
	SkillNPCFirebreath                      uint16 = 654
	SkillNPCIcebreath                       uint16 = 655
	SkillNPCThunderbreath                   uint16 = 656
	SkillNPCAcidbreath                      uint16 = 657
	SkillNPCDarknessbreath                  uint16 = 658
	SkillNPCDragonfear                      uint16 = 659
	SkillNPCBleeding                        uint16 = 660
	SkillNPCPulsestrike                     uint16 = 661
	SkillNPCHelljudgement                   uint16 = 662
	SkillNPCWidesilence                     uint16 = 663
	SkillNPCWidefreeze                      uint16 = 664
	SkillNPCWidebleeding                    uint16 = 665
	SkillNPCWidestone                       uint16 = 666
	SkillNPCWideconfuse                     uint16 = 667
	SkillNPCWidesleep                       uint16 = 668
	SkillNPCWidesight                       uint16 = 669
	SkillNPCEvilland                        uint16 = 670
	SkillNPCMagicmirror                     uint16 = 671
	SkillNPCSlowcast                        uint16 = 672
	SkillNPCCriticalwound                   uint16 = 673
	SkillNPCExpulsion                       uint16 = 674
	SkillNPCStoneskin                       uint16 = 675
	SkillNPCAntimagic                       uint16 = 676
	SkillNPCWidecurse                       uint16 = 677
	SkillNPCWidestun                        uint16 = 678
	SkillNPCVampireGift                     uint16 = 679
	SkillNPCWidesouldrain                   uint16 = 680
	SkillALLInccarry                        uint16 = 681
	SkillNPCTalk                            uint16 = 682
	SkillNPCHellpower                       uint16 = 683
	SkillNPCWidehelldignity                 uint16 = 684
	SkillNPCInvincible                      uint16 = 685
	SkillNPCInvincibleoff                   uint16 = 686
	SkillNPCAllheal                         uint16 = 687
	SkillGmSandman                          uint16 = 688
	SkillCashBlessing                       uint16 = 689
	SkillCashIncagi                         uint16 = 690
	SkillCashAssumptio                      uint16 = 691
	SkillALLCatcry                          uint16 = 692
	SkillALLPartyflee                       uint16 = 693
	SkillALLAngelProtect                    uint16 = 694
	SkillALLDreamSummernight                uint16 = 695
	SkillNPCChangeundead2                   uint16 = 696
	SkillALLReverseorcish                   uint16 = 697
	SkillALLWewish                          uint16 = 698
	SkillALLSonkran                         uint16 = 699
	SkillNPCWidehealthfear                  uint16 = 700
	SkillNPCWidebodyburnning                uint16 = 701
	SkillNPCWidefrostmisty                  uint16 = 702
	SkillNPCWidecold                        uint16 = 703
	SkillNPCWideDeepSleep                   uint16 = 704
	SkillNPCWidesiren                       uint16 = 705
	SkillNPCVenomfog                        uint16 = 706
	SkillNPCMillenniumshield                uint16 = 707
	SkillNPCComet                           uint16 = 708
	SkillNPCIcemine                         uint16 = 709
	SkillNPCIceexplo                        uint16 = 710
	SkillNPCFlamecross                      uint16 = 711
	SkillNPCPulsestrike2                    uint16 = 712
	SkillNPCDancingblade                    uint16 = 713
	SkillNPCDancingbladeAtk                 uint16 = 714
	SkillNPCDarkpiercing                    uint16 = 715
	SkillNPCMaxpain                         uint16 = 716
	SkillNPCMaxpainAtk                      uint16 = 717
	SkillNPCDeathsummon                     uint16 = 718
	SkillNPCHellburning                     uint16 = 719
	SkillNPCJackfrost                       uint16 = 720
	SkillNPCWideweb                         uint16 = 721
	SkillNPCWidesuck                        uint16 = 722
	SkillNPCStormgust2                      uint16 = 723
	SkillNPCFirestorm                       uint16 = 724
	SkillNPCReverberation                   uint16 = 725
	SkillNPCReverberationAtk                uint16 = 726
	SkillNPCLexAeterna                      uint16 = 727
	SkillNPCArrowstorm                      uint16 = 728
	SkillNPCCheal                           uint16 = 729
	SkillNPCSRCursedcircle                  uint16 = 730
	SkillNPCDragonbreath                    uint16 = 731
	SkillNPCFatalmenace                     uint16 = 732
	SkillNPCMagmaEruption                   uint16 = 733
	SkillNPCMagmaEruptionDotdamage          uint16 = 734
	SkillNPCMandragora                      uint16 = 735
	SkillNPCPsychicWave                     uint16 = 736
	SkillNPCRayofgenesis                    uint16 = 737
	SkillNPCVenomimpress                    uint16 = 738
	SkillNPCCloudKill                       uint16 = 739
	SkillNPCIgnitionbreak                   uint16 = 740
	SkillNPCPhantomthrust                   uint16 = 741
	SkillNPCPoisonBuster                    uint16 = 742
	SkillNPCHallucinationwalk               uint16 = 743
	SkillNPCElectricwalk                    uint16 = 744
	SkillNPCFirewalk                        uint16 = 745
	SkillNPCWidedispel                      uint16 = 746
	SkillNPCLeash                           uint16 = 747
	SkillNPCWideleash                       uint16 = 748
	SkillNPCWidecriticalwound               uint16 = 749
	SkillNPCEarthquakeK                     uint16 = 750
	SkillNPCALLStatDown                     uint16 = 751
	SkillNPCGradualGravity                  uint16 = 752
	SkillNPCDamageHeal                      uint16 = 753
	SkillNPCImmuneProperty                  uint16 = 754
	SkillNPCMoveCoordinate                  uint16 = 755
	SkillNPCWidebleeding2                   uint16 = 756
	SkillNPCWidesilence2                    uint16 = 757
	SkillNPCWidestun2                       uint16 = 758
	SkillNPCWidestone2                      uint16 = 759
	SkillNPCWidesleep2                      uint16 = 760
	SkillNPCWidecurse2                      uint16 = 761
	SkillNPCWideconfuse2                    uint16 = 762
	SkillNPCWidefreeze2                     uint16 = 763
	SkillNPCBleeding2                       uint16 = 764
	SkillNPCIcebreath2                      uint16 = 765
	SkillNPCAcidbreath2                     uint16 = 766
	SkillNPCEvilland2                       uint16 = 767
	SkillNPCHelljudgement2                  uint16 = 768
	SkillNPCRainofmeteor                    uint16 = 769
	SkillNPCGrounddrive                     uint16 = 770
	SkillNPCRelieveOn                       uint16 = 771
	SkillNPCRelieveOff                      uint16 = 772
	SkillNPCLockonLaser                     uint16 = 773
	SkillNPCLockonLaserAtk                  uint16 = 774
	SkillNPCSeedtrap                        uint16 = 775
	SkillNPCDeadlycurse                     uint16 = 776
	SkillNPCRandombreak                     uint16 = 777
	SkillNPCStripShadow                     uint16 = 778
	SkillNPCDeadlycurse2                    uint16 = 779
	SkillNPCCaneOfEvilEye                   uint16 = 780
	SkillNPCCurseOfRedCube                  uint16 = 781
	SkillNPCCurseOfBlueCube                 uint16 = 782
	SkillNPCKillingAura                     uint16 = 783
	SkillNPCLast                            uint16 = 785
	SkillKNChargeatk                        uint16 = 1001
	SkillCRShrink                           uint16 = 1002
	SkillASSonicaccel                       uint16 = 1003
	SkillASVenomknife                       uint16 = 1004
	SkillRGCloseconfine                     uint16 = 1005
	SkillWZSightblaster                     uint16 = 1006
	SkillSACreatecon                        uint16 = 1007
	SkillSAElementwater                     uint16 = 1008
	SkillHTPhantasmic                       uint16 = 1009
	SkillBaPangvoice                        uint16 = 1010
	SkillDCWinkcharm                        uint16 = 1011
	SkillBSUnfairlytrick                    uint16 = 1012
	SkillBSGreed                            uint16 = 1013
	SkillPRRedemptio                        uint16 = 1014
	SkillMOKitranslation                    uint16 = 1015
	SkillMOBalkyoung                        uint16 = 1016
	SkillSAElementground                    uint16 = 1017
	SkillSAElementfire                      uint16 = 1018
	SkillSAElementwind                      uint16 = 1019
	SkillThirdjobBegin                      uint16 = 2000
	SkillRKEnchantblade                     uint16 = 2001
	SkillRKSonicwave                        uint16 = 2002
	SkillRKDeathbound                       uint16 = 2003
	SkillRKHundredspear                     uint16 = 2004
	SkillRKWindcutter                       uint16 = 2005
	SkillRKIgnitionbreak                    uint16 = 2006
	SkillRKDragontraining                   uint16 = 2007
	SkillRKDragonbreath                     uint16 = 2008
	SkillRKDragonhowling                    uint16 = 2009
	SkillRKRunemastery                      uint16 = 2010
	SkillRKMillenniumshield                 uint16 = 2011
	SkillRKCrushstrike                      uint16 = 2012
	SkillRKRefresh                          uint16 = 2013
	SkillRKGiantgrowth                      uint16 = 2014
	SkillRKStonehardskin                    uint16 = 2015
	SkillRKVitalityactivation               uint16 = 2016
	SkillRKStormblast                       uint16 = 2017
	SkillRKFightingspirit                   uint16 = 2018
	SkillRKAbundance                        uint16 = 2019
	SkillRKPhantomthrust                    uint16 = 2020
	SkillGCVenomimpress                     uint16 = 2021
	SkillGCCrossimpact                      uint16 = 2022
	SkillGCDarkillusion                     uint16 = 2023
	SkillGCResearchnewpoison                uint16 = 2024
	SkillGCCreatenewpoison                  uint16 = 2025
	SkillGCAntidote                         uint16 = 2026
	SkillGCPoisoningweapon                  uint16 = 2027
	SkillGCWeaponblocking                   uint16 = 2028
	SkillGCCounterslash                     uint16 = 2029
	SkillGCWeaponcrush                      uint16 = 2030
	SkillGCVenompressure                    uint16 = 2031
	SkillGCPoisonsmoke                      uint16 = 2032
	SkillGCCloakingexceed                   uint16 = 2033
	SkillGCPhantommenace                    uint16 = 2034
	SkillGCHallucinationwalk                uint16 = 2035
	SkillGCRollingcutter                    uint16 = 2036
	SkillGCCrossripperslasher               uint16 = 2037
	SkillABJudex                            uint16 = 2038
	SkillABAncilla                          uint16 = 2039
	SkillABAdoramus                         uint16 = 2040
	SkillABClementia                        uint16 = 2041
	SkillABCanto                            uint16 = 2042
	SkillABCheal                            uint16 = 2043
	SkillABEpiclesis                        uint16 = 2044
	SkillABPraefatio                        uint16 = 2045
	SkillABOratio                           uint16 = 2046
	SkillABLaudaagnus                       uint16 = 2047
	SkillABLaudaramus                       uint16 = 2048
	SkillABEucharistica                     uint16 = 2049
	SkillABRenovatio                        uint16 = 2050
	SkillABHighnessheal                     uint16 = 2051
	SkillABClearance                        uint16 = 2052
	SkillABExpiatio                         uint16 = 2053
	SkillABDuplelight                       uint16 = 2054
	SkillABDuplelightMelee                  uint16 = 2055
	SkillABDuplelightMagic                  uint16 = 2056
	SkillABSilentium                        uint16 = 2057
	SkillWLStartmark                        uint16 = 2200
	SkillWLWhiteimprison                    uint16 = 2201
	SkillWLSoulexpansion                    uint16 = 2202
	SkillWLFrostmisty                       uint16 = 2203
	SkillWLJackfrost                        uint16 = 2204
	SkillWLMarshofabyss                     uint16 = 2205
	SkillWLRecognizedspell                  uint16 = 2206
	SkillWLSiennaexecrate                   uint16 = 2207
	SkillWLRadius                           uint16 = 2208
	SkillWLStasis                           uint16 = 2209
	SkillWLDrainlife                        uint16 = 2210
	SkillWLCrimsonrock                      uint16 = 2211
	SkillWLHellinferno                      uint16 = 2212
	SkillWLComet                            uint16 = 2213
	SkillWLChainlightning                   uint16 = 2214
	SkillWLChainlightningAtk                uint16 = 2215
	SkillWLEarthstrain                      uint16 = 2216
	SkillWLTetravortex                      uint16 = 2217
	SkillWLTetravortexFire                  uint16 = 2218
	SkillWLTetravortexWater                 uint16 = 2219
	SkillWLTetravortexWind                  uint16 = 2220
	SkillWLTetravortexGround                uint16 = 2221
	SkillWLSummonfb                         uint16 = 2222
	SkillWLSummonbl                         uint16 = 2223
	SkillWLSummonwb                         uint16 = 2224
	SkillWLSummonAtkFire                    uint16 = 2225
	SkillWLSummonAtkWind                    uint16 = 2226
	SkillWLSummonAtkWater                   uint16 = 2227
	SkillWLSummonAtkGround                  uint16 = 2228
	SkillWLSummonstone                      uint16 = 2229
	SkillWLRelease                          uint16 = 2230
	SkillWLReadingSb                        uint16 = 2231
	SkillWLFreezeSP                         uint16 = 2232
	SkillWLEndmark                          uint16 = 2232
	SkillRAArrowstorm                       uint16 = 2233
	SkillRAFearbreeze                       uint16 = 2234
	SkillRARangermain                       uint16 = 2235
	SkillRAAimedbolt                        uint16 = 2236
	SkillRADetonator                        uint16 = 2237
	SkillRAElectricshocker                  uint16 = 2238
	SkillRAClusterbomb                      uint16 = 2239
	SkillRAWugmastery                       uint16 = 2240
	SkillRAWugrider                         uint16 = 2241
	SkillRAWugdash                          uint16 = 2242
	SkillRAWugstrike                        uint16 = 2243
	SkillRAWugbite                          uint16 = 2244
	SkillRAToothofwug                       uint16 = 2245
	SkillRASensitivekeen                    uint16 = 2246
	SkillRACamouflage                       uint16 = 2247
	SkillRAResearchtrap                     uint16 = 2248
	SkillRAMagentatrap                      uint16 = 2249
	SkillRACobalttrap                       uint16 = 2250
	SkillRAMaizetrap                        uint16 = 2251
	SkillRAVerduretrap                      uint16 = 2252
	SkillRAFiringtrap                       uint16 = 2253
	SkillRAIceboundtrap                     uint16 = 2254
	SkillNCMadolicence                      uint16 = 2255
	SkillNCBoostknuckle                     uint16 = 2256
	SkillNCPilebunker                       uint16 = 2257
	SkillNCVulcanarm                        uint16 = 2258
	SkillNCFlamelauncher                    uint16 = 2259
	SkillNCColdslower                       uint16 = 2260
	SkillNCArmscannon                       uint16 = 2261
	SkillNCAcceleration                     uint16 = 2262
	SkillNCHovering                         uint16 = 2263
	SkillNCFSideslide                       uint16 = 2264
	SkillNCBSideslide                       uint16 = 2265
	SkillNCMainframe                        uint16 = 2266
	SkillNCSelfdestruction                  uint16 = 2267
	SkillNCShapeshift                       uint16 = 2268
	SkillNCEmergencycool                    uint16 = 2269
	SkillNCInfraredscan                     uint16 = 2270
	SkillNCAnalyze                          uint16 = 2271
	SkillNCMagneticfield                    uint16 = 2272
	SkillNCNeutralbarrier                   uint16 = 2273
	SkillNCStealthfield                     uint16 = 2274
	SkillNCRepair                           uint16 = 2275
	SkillNCTrainingaxe                      uint16 = 2276
	SkillNCResearchfe                       uint16 = 2277
	SkillNCAxeboomerang                     uint16 = 2278
	SkillNCPowerswing                       uint16 = 2279
	SkillNCAxetornado                       uint16 = 2280
	SkillNCSilversniper                     uint16 = 2281
	SkillNCMagicdecoy                       uint16 = 2282
	SkillNCDisjoint                         uint16 = 2283
	SkillSCStartmark                        uint16 = 2284
	SkillSCFatalmenace                      uint16 = 2284
	SkillSCReproduce                        uint16 = 2285
	SkillSCAutoshadowspell                  uint16 = 2286
	SkillSCShadowform                       uint16 = 2287
	SkillSCTriangleshot                     uint16 = 2288
	SkillSCBodypaint                        uint16 = 2289
	SkillSCInvisibility                     uint16 = 2290
	SkillSCDeadlyinfect                     uint16 = 2291
	SkillSCEnervation                       uint16 = 2292
	SkillSCGroomy                           uint16 = 2293
	SkillSCIgnorance                        uint16 = 2294
	SkillSCLaziness                         uint16 = 2295
	SkillSCUnlucky                          uint16 = 2296
	SkillSCWeakness                         uint16 = 2297
	SkillSCStripaccessary                   uint16 = 2298
	SkillSCManhole                          uint16 = 2299
	SkillSCDimensiondoor                    uint16 = 2300
	SkillSCChaospanic                       uint16 = 2301
	SkillSCMaelstrom                        uint16 = 2302
	SkillSCBloodylust                       uint16 = 2303
	SkillSCFeintbomb                        uint16 = 2304
	SkillSCEndmark                          uint16 = 2306
	SkillLGCannonspear                      uint16 = 2307
	SkillLGBanishingpoint                   uint16 = 2308
	SkillLGTrample                          uint16 = 2309
	SkillLGShieldpress                      uint16 = 2310
	SkillLGReflectdamage                    uint16 = 2311
	SkillLGPinpointattack                   uint16 = 2312
	SkillLGForceofvanguard                  uint16 = 2313
	SkillLGRageburst                        uint16 = 2314
	SkillLGShieldspell                      uint16 = 2315
	SkillLGExeedbreak                       uint16 = 2316
	SkillLGOverbrand                        uint16 = 2317
	SkillLGPrestige                         uint16 = 2318
	SkillLGBanding                          uint16 = 2319
	SkillLGMoonslasher                      uint16 = 2320
	SkillLGRayofgenesis                     uint16 = 2321
	SkillLGPiety                            uint16 = 2322
	SkillLGEarthdrive                       uint16 = 2323
	SkillLGHesperuslit                      uint16 = 2324
	SkillLGInspiration                      uint16 = 2325
	SkillSRDragoncombo                      uint16 = 2326
	SkillSRSkynetblow                       uint16 = 2327
	SkillSREarthshaker                      uint16 = 2328
	SkillSRFallenempire                     uint16 = 2329
	SkillSRTigercannon                      uint16 = 2330
	SkillSRHellgate                         uint16 = 2331
	SkillSRRampageblaster                   uint16 = 2332
	SkillSRCrescentelbow                    uint16 = 2333
	SkillSRCursedcircle                     uint16 = 2334
	SkillSRLightningwalk                    uint16 = 2335
	SkillSRKnucklearrow                     uint16 = 2336
	SkillSRWindmill                         uint16 = 2337
	SkillSRRaisingdragon                    uint16 = 2338
	SkillSRGentletouch                      uint16 = 2339
	SkillSRAssimilatepower                  uint16 = 2340
	SkillSRPowervelocity                    uint16 = 2341
	SkillSRCrescentelbowAutospell           uint16 = 2342
	SkillSRGateofhell                       uint16 = 2343
	SkillSRGentletouchQuiet                 uint16 = 2344
	SkillSRGentletouchCure                  uint16 = 2345
	SkillSRGentletouchEnergygain            uint16 = 2346
	SkillSRGentletouchChange                uint16 = 2347
	SkillSRGentletouchRevitalize            uint16 = 2348
	SkillWAStartmark                        uint16 = 2349
	SkillWASwingDance                       uint16 = 2350
	SkillWASymphonyOfLover                  uint16 = 2351
	SkillWAMoonlitSerenade                  uint16 = 2352
	SkillWAEndmark                          uint16 = 2379
	SkillMIStartmark                        uint16 = 2380
	SkillMIRushWindmill                     uint16 = 2381
	SkillMIEchosong                         uint16 = 2382
	SkillMIHarmonize                        uint16 = 2383
	SkillMIEndmark                          uint16 = 2410
	SkillWmStartmark                        uint16 = 2411
	SkillWmLesson                           uint16 = 2412
	SkillWmMetalicsound                     uint16 = 2413
	SkillWmReverberation                    uint16 = 2414
	SkillWmReverberationMelee               uint16 = 2415
	SkillWmReverberationMagic               uint16 = 2416
	SkillWmDominionImpulse                  uint16 = 2417
	SkillWmSevereRainstorm                  uint16 = 2418
	SkillWmPoemofnetherworld                uint16 = 2419
	SkillWmVoiceofsiren                     uint16 = 2420
	SkillWmDeadhillhere                     uint16 = 2421
	SkillWmLullabyDeepsleep                 uint16 = 2422
	SkillWmSircleofnature                   uint16 = 2423
	SkillWmRandomizespell                   uint16 = 2424
	SkillWmGloomyday                        uint16 = 2425
	SkillWmGreatEcho                        uint16 = 2426
	SkillWmSongOfMana                       uint16 = 2427
	SkillWmDanceWithWug                     uint16 = 2428
	SkillWmSoundOfDestruction               uint16 = 2429
	SkillWmSaturdayNightFever               uint16 = 2430
	SkillWmLeradsDew                        uint16 = 2431
	SkillWmMelodyofsink                     uint16 = 2432
	SkillWmBeyondOfWarcry                   uint16 = 2433
	SkillWmUnlimitedHummingVoice            uint16 = 2434
	SkillWmEndmark                          uint16 = 2441
	SkillSOStartmark                        uint16 = 2442
	SkillSOFirewalk                         uint16 = 2443
	SkillSOElectricwalk                     uint16 = 2444
	SkillSOSpellfist                        uint16 = 2445
	SkillSOEarthgrave                       uint16 = 2446
	SkillSODiamonddust                      uint16 = 2447
	SkillSOPoisonBuster                     uint16 = 2448
	SkillSOPsychicWave                      uint16 = 2449
	SkillSOCloudKill                        uint16 = 2450
	SkillSOStriking                         uint16 = 2451
	SkillSOWarmer                           uint16 = 2452
	SkillSOVacuumExtreme                    uint16 = 2453
	SkillSOVaretyrSpear                     uint16 = 2454
	SkillSOArrullo                          uint16 = 2455
	SkillSOElControl                        uint16 = 2456
	SkillSOSummonAgni                       uint16 = 2457
	SkillSOSummonAqua                       uint16 = 2458
	SkillSOSummonVentus                     uint16 = 2459
	SkillSOSummonTera                       uint16 = 2460
	SkillSOElAction                         uint16 = 2461
	SkillSOElAnalysis                       uint16 = 2462
	SkillSOElSympathy                       uint16 = 2463
	SkillSOElCure                           uint16 = 2464
	SkillSOFireInsignia                     uint16 = 2465
	SkillSOWaterInsignia                    uint16 = 2466
	SkillSOWindInsignia                     uint16 = 2467
	SkillSOEarthInsignia                    uint16 = 2468
	SkillSOEndmark                          uint16 = 2472
	SkillGNStartMark                        uint16 = 2473
	SkillGNTrainingSword                    uint16 = 2474
	SkillGNRemodelingCart                   uint16 = 2475
	SkillGNCartTornado                      uint16 = 2476
	SkillGNCartcannon                       uint16 = 2477
	SkillGNCartboost                        uint16 = 2478
	SkillGNThornsTrap                       uint16 = 2479
	SkillGNBloodSucker                      uint16 = 2480
	SkillGNSporeExplosion                   uint16 = 2481
	SkillGNWallofthorn                      uint16 = 2482
	SkillGNCrazyweed                        uint16 = 2483
	SkillGNCrazyweedAtk                     uint16 = 2484
	SkillGNDemonicFire                      uint16 = 2485
	SkillGNFireExpansion                    uint16 = 2486
	SkillGNFireExpansionSmokePowder         uint16 = 2487
	SkillGNFireExpansionTearGas             uint16 = 2488
	SkillGNFireExpansionAcid                uint16 = 2489
	SkillGNHellsPlant                       uint16 = 2490
	SkillGNHellsPlantAtk                    uint16 = 2491
	SkillGNMandragora                       uint16 = 2492
	SkillGNSlingitem                        uint16 = 2493
	SkillGNChangematerial                   uint16 = 2494
	SkillGNMixCooking                       uint16 = 2495
	SkillGNMakebomb                         uint16 = 2496
	SkillGNSPharmacy                        uint16 = 2497
	SkillGNSlingitemRangemeleeatk           uint16 = 2498
	SkillGNEndmark                          uint16 = 2513
	SkillEtcThirdjobSkillStart              uint16 = 2514
	SkillABSecrament                        uint16 = 2515
	SkillWmSevereRainstormMelee             uint16 = 2516
	SkillSRHowlingoflion                    uint16 = 2517
	SkillSRRideinlightning                  uint16 = 2518
	SkillLGOverbrandBrandish                uint16 = 2519
	SkillLGOverbrandPlusatk                 uint16 = 2520
	SkillEtcThirdjobSkillEnd                uint16 = 2531
	SkillThirdjobEnd                        uint16 = 2532
	SkillALLOdinsRecall                     uint16 = 2533
	SkillReturnToEldicastes                 uint16 = 2534
	SkillALLBuyingStore                     uint16 = 2535
	SkillALLGuardianRecall                  uint16 = 2536
	SkillALLOdinsPower                      uint16 = 2537
	SkillXxBeerBottleCap                    uint16 = 2538
	SkillNPCAssassincross                   uint16 = 2539
	SkillNPCDissonance                      uint16 = 2540
	SkillNPCUglydance                       uint16 = 2541
	SkillALLTetany                          uint16 = 2542
	SkillALLRayOfProtection                 uint16 = 2543
	SkillMCCartdecorate                     uint16 = 2544
	SkillGmItemAtkmax                       uint16 = 2545
	SkillGmItemAtkmin                       uint16 = 2546
	SkillGmItemMatkmax                      uint16 = 2547
	SkillGmItemMatkmin                      uint16 = 2548
	SkillGmApHeal                           uint16 = 2549
	SkillUpperExtendedJobStart              uint16 = 2550
	SkillRLGlitteringGreed                  uint16 = 2551
	SkillRLRichsCoin                        uint16 = 2552
	SkillRLMassSpiral                       uint16 = 2553
	SkillRLBanishingBuster                  uint16 = 2554
	SkillRLBTrap                            uint16 = 2555
	SkillRLFlicker                          uint16 = 2556
	SkillRLSStorm                           uint16 = 2557
	SkillRLEChain                           uint16 = 2558
	SkillRLQdShot                           uint16 = 2559
	SkillRLCMarker                          uint16 = 2560
	SkillRLFiredance                        uint16 = 2561
	SkillRLHMine                            uint16 = 2562
	SkillRLPAlter                           uint16 = 2563
	SkillRLFallenAngel                      uint16 = 2564
	SkillRLRTrip                            uint16 = 2565
	SkillRLDTail                            uint16 = 2566
	SkillRLFireRain                         uint16 = 2567
	SkillRLHeatBarrel                       uint16 = 2568
	SkillRLAMBlast                          uint16 = 2569
	SkillRLSlugshot                         uint16 = 2570
	SkillRLHammerOfGod                      uint16 = 2571
	SkillRLRTripPlusatk                     uint16 = 2572
	SkillRLBFlickerAtk                      uint16 = 2573
	SkillRLGlitteringGreedAtk               uint16 = 2574
	SkillSJLightofmoon                      uint16 = 2574
	SkillSJLunarstance                      uint16 = 2575
	SkillSJFullmoonkick                     uint16 = 2576
	SkillSJLightofstar                      uint16 = 2577
	SkillSJStarstance                       uint16 = 2578
	SkillSJNewmoonkick                      uint16 = 2579
	SkillSJFlashkick                        uint16 = 2580
	SkillSJStaremperor                      uint16 = 2581
	SkillSJNovaexplosing                    uint16 = 2582
	SkillSJUniversestance                   uint16 = 2583
	SkillSJFallingstar                      uint16 = 2584
	SkillSJGravitycontrol                   uint16 = 2585
	SkillSJBookofdimension                  uint16 = 2586
	SkillSJBookofcreatingstar               uint16 = 2587
	SkillSJDocument                         uint16 = 2588
	SkillSJPurify                           uint16 = 2589
	SkillSJLightofsun                       uint16 = 2590
	SkillSJSunstance                        uint16 = 2591
	SkillSJSolarburst                       uint16 = 2592
	SkillSJProminencekick                   uint16 = 2593
	SkillSJFallingstarAtk                   uint16 = 2594
	SkillSJFallingstarAtk2                  uint16 = 2595
	SkillSPSoulgolem                        uint16 = 2596
	SkillSPSoulshadow                       uint16 = 2597
	SkillSPSoulfalcon                       uint16 = 2598
	SkillSPSoulfairy                        uint16 = 2599
	SkillSPCurseexplosion                   uint16 = 2600
	SkillSPSoulcurse                        uint16 = 2601
	SkillSPSpa                              uint16 = 2602
	SkillSPSha                              uint16 = 2603
	SkillSPSwhoo                            uint16 = 2604
	SkillSPSoulunity                        uint16 = 2605
	SkillSPSouldivision                     uint16 = 2606
	SkillSPSoulreaper                       uint16 = 2607
	SkillSPSoulrevolve                      uint16 = 2608
	SkillSPSoulcollect                      uint16 = 2609
	SkillSPSoulexplosion                    uint16 = 2610
	SkillSPSoulenergy                       uint16 = 2611
	SkillSPKaute                            uint16 = 2612
	SkillKOYamikumo                         uint16 = 3001
	SkillKORight                            uint16 = 3002
	SkillKOLeft                             uint16 = 3003
	SkillKOJyumonjikiri                     uint16 = 3004
	SkillKOSetsudan                         uint16 = 3005
	SkillKOBakuretsu                        uint16 = 3006
	SkillKOHappokunai                       uint16 = 3007
	SkillKOMuchanage                        uint16 = 3008
	SkillKOHuumaranka                       uint16 = 3009
	SkillKOMakibishi                        uint16 = 3010
	SkillKOMeikyousisui                     uint16 = 3011
	SkillKOZanzou                           uint16 = 3012
	SkillKOKyougaku                         uint16 = 3013
	SkillKOJyusatsu                         uint16 = 3014
	SkillKOKahuEnten                        uint16 = 3015
	SkillKOHyouhuHubuki                     uint16 = 3016
	SkillKOKazehuSeiran                     uint16 = 3017
	SkillKODohuKoukai                       uint16 = 3018
	SkillKOKaihou                           uint16 = 3019
	SkillKOZenkai                           uint16 = 3020
	SkillKOGenwaku                          uint16 = 3021
	SkillKOIzayoi                           uint16 = 3022
	SkillKgKagehumi                         uint16 = 3023
	SkillKgKyomu                            uint16 = 3024
	SkillKgKagemusya                        uint16 = 3025
	SkillObZangetsu                         uint16 = 3026
	SkillObOborogensou                      uint16 = 3027
	SkillObOborogensouTransitionAtk         uint16 = 3028
	SkillObAkaitsuki                        uint16 = 3029
	SkillUpperExtendedJobEnd                uint16 = 3030
	SkillEclSnowflip                        uint16 = 3031
	SkillEclPeonymamy                       uint16 = 3032
	SkillEclSadagui                         uint16 = 3033
	SkillEclSequoiadust                     uint16 = 3034
	SkillEclageRecall                       uint16 = 3035
	SkillBaPoembragi2                       uint16 = 3036
	SkillDCFortunekiss2                     uint16 = 3037
	SkillItemOptionSplashAttack             uint16 = 3038
	SkillGmForceTransfer                    uint16 = 3039
	SkillGmWideResurrection                 uint16 = 3040
	SkillALLNiflheimRecall                  uint16 = 3041
	SkillALLPronteraRecall                  uint16 = 3042
	SkillALLGlastheimRecall                 uint16 = 3043
	SkillALLThanatosRecall                  uint16 = 3044
	SkillLevelExpansionStart                uint16 = 5000
	SkillGCDarkcrow                         uint16 = 5001
	SkillRAUnlimit                          uint16 = 5002
	SkillGNIllusiondoping                   uint16 = 5003
	SkillRKDragonbreathWater                uint16 = 5004
	SkillRKLuxanima                         uint16 = 5005
	SkillNCMagmaEruption                    uint16 = 5006
	SkillWmFriggSong                        uint16 = 5007
	SkillSOElementalShield                  uint16 = 5008
	SkillSRFlashcombo                       uint16 = 5009
	SkillSCEscape                           uint16 = 5010
	SkillABOffertorium                      uint16 = 5011
	SkillWLTelekinesisIntense               uint16 = 5012
	SkillLGKingsGrace                       uint16 = 5013
	SkillALLFullThrottle                    uint16 = 5014
	SkillNCMagmaEruptionDotdamage           uint16 = 5015
	SkillLevelExpansionEnd                  uint16 = 5016
	SkillDoramTribeStart                    uint16 = 5017
	SkillSUBasicSkill                       uint16 = 5018
	SkillSUBite                             uint16 = 5019
	SkillSUHide                             uint16 = 5020
	SkillSUScratch                          uint16 = 5021
	SkillSUStoop                            uint16 = 5022
	SkillSULope                             uint16 = 5023
	SkillSUSpritemable                      uint16 = 5024
	SkillSUPowerofland                      uint16 = 5025
	SkillSUSvStemspear                      uint16 = 5026
	SkillSUCnPowdering                      uint16 = 5027
	SkillSUCnMeteor                         uint16 = 5028
	SkillSUSvRoottwist                      uint16 = 5029
	SkillSUSvRoottwistAtk                   uint16 = 5030
	SkillSUPoweroflife                      uint16 = 5031
	SkillSUScaroftarou                      uint16 = 5032
	SkillSUPickypeck                        uint16 = 5033
	SkillSUPickypeckDoubleAtk               uint16 = 5034
	SkillSUArclousedash                     uint16 = 5035
	SkillSULunaticcarrotbeat                uint16 = 5036
	SkillSUPowerofsea                       uint16 = 5037
	SkillSUTunabelly                        uint16 = 5038
	SkillSUTunaparty                        uint16 = 5039
	SkillSUBunchofshrimp                    uint16 = 5040
	SkillSUFreshshrimp                      uint16 = 5041
	SkillSUCnMeteor2                        uint16 = 5042
	SkillSULunaticcarrotbeat2               uint16 = 5043
	SkillSUSoulattack                       uint16 = 5044
	SkillSUPowerofflock                     uint16 = 5045
	SkillSUSvgSpirit                        uint16 = 5046
	SkillSUHiss                             uint16 = 5047
	SkillSUNyanggrass                       uint16 = 5048
	SkillSUGrooming                         uint16 = 5049
	SkillSUPurring                          uint16 = 5050
	SkillSUShrimparty                       uint16 = 5051
	SkillSUSpiritoflife                     uint16 = 5052
	SkillSUMeowmeow                         uint16 = 5053
	SkillSUSpiritofland                     uint16 = 5054
	SkillSUChattering                       uint16 = 5055
	SkillSUSpiritofsea                      uint16 = 5056
	SkillDoramTribeEnd                      uint16 = 5057
	SkillLast                               uint16 = 5058
	SkillWECallallfamily                    uint16 = 5063
	SkillWEOneforever                       uint16 = 5064
	SkillWECheerup                          uint16 = 5065
	SkillCGSpecialsinger                    uint16 = 5068
	SkillABVituperatum                      uint16 = 5072
	SkillABConvenio                         uint16 = 5073
	SkillNVBreakthrough                     uint16 = 5075
	SkillNVHelpangel                        uint16 = 5076
	SkillNVTranscendence                    uint16 = 5077
	SkillWLReadingSbReading                 uint16 = 5078
	SkillDkServantweapon                    uint16 = 5201
	SkillDkServantweaponAtk                 uint16 = 5202
	SkillDkServantWSign                     uint16 = 5203
	SkillDkServantWPhantom                  uint16 = 5204
	SkillDkServantWDemol                    uint16 = 5205
	SkillDkChargingpierce                   uint16 = 5206
	SkillDkTwohanddef                       uint16 = 5207
	SkillDkHackandslasher                   uint16 = 5208
	SkillDkHackandslasherAtk                uint16 = 5209
	SkillDkDragonicAura                     uint16 = 5210
	SkillDkMadnessCrusher                   uint16 = 5211
	SkillDkVigor                            uint16 = 5212
	SkillDkStormslash                       uint16 = 5213
	SkillAgDeadlyProjection                 uint16 = 5214
	SkillAgDestructiveHurricane             uint16 = 5215
	SkillAgRainOfCrystal                    uint16 = 5216
	SkillAgMysteryIllusion                  uint16 = 5217
	SkillAgViolentQuake                     uint16 = 5218
	SkillAgViolentQuakeAtk                  uint16 = 5219
	SkillAgSoulVcStrike                     uint16 = 5220
	SkillAgStrantumTremor                   uint16 = 5221
	SkillAgALLBloom                         uint16 = 5222
	SkillAgALLBloomAtk                      uint16 = 5223
	SkillAgALLBloomAtk2                     uint16 = 5224
	SkillAgCrystalImpact                    uint16 = 5225
	SkillAgCrystalImpactAtk                 uint16 = 5226
	SkillAgTornadoStorm                     uint16 = 5227
	SkillAgTwohandstaff                     uint16 = 5228
	SkillAgFloralFlareRoad                  uint16 = 5229
	SkillAgAstralStrike                     uint16 = 5230
	SkillAgAstralStrikeAtk                  uint16 = 5231
	SkillAgClimax                           uint16 = 5232
	SkillAgRockDown                         uint16 = 5233
	SkillAgStormCannon                      uint16 = 5234
	SkillAgCrimsonArrow                     uint16 = 5235
	SkillAgCrimsonArrowAtk                  uint16 = 5236
	SkillAgFrozenSlash                      uint16 = 5237
	SkillIqPowerfulFaith                    uint16 = 5238
	SkillIqFirmFaith                        uint16 = 5239
	SkillIqWillOfFaith                      uint16 = 5240
	SkillIqOleumSanctum                     uint16 = 5241
	SkillIqSincereFaith                     uint16 = 5242
	SkillIqMassiveFBlaster                  uint16 = 5243
	SkillIqExposionBlaster                  uint16 = 5244
	SkillIqFirstBrand                       uint16 = 5245
	SkillIqFirstFaithPower                  uint16 = 5246
	SkillIqJudge                            uint16 = 5247
	SkillIqSecondFlame                      uint16 = 5248
	SkillIqSecondFaith                      uint16 = 5249
	SkillIqSecondJudgement                  uint16 = 5250
	SkillIqThirdPunish                      uint16 = 5251
	SkillIqThirdFlameBomb                   uint16 = 5252
	SkillIqThirdConsecration                uint16 = 5253
	SkillIqThirdExorFlame                   uint16 = 5254
	SkillIgGuardStance                      uint16 = 5255
	SkillIgGuardianShield                   uint16 = 5256
	SkillIgReboundShield                    uint16 = 5257
	SkillIgShieldMastery                    uint16 = 5258
	SkillIgSpearSwordM                      uint16 = 5259
	SkillIgAttackStance                     uint16 = 5260
	SkillIgUltimateSacrifice                uint16 = 5261
	SkillIgHolyShield                       uint16 = 5262
	SkillIgGrandJudgement                   uint16 = 5263
	SkillIgJudgementCross                   uint16 = 5264
	SkillIgShieldShooting                   uint16 = 5265
	SkillIgOverslash                        uint16 = 5266
	SkillIgCrossRain                        uint16 = 5267
	SkillShcShadowExceed                    uint16 = 5285
	SkillShcDancingKnife                    uint16 = 5286
	SkillShcSavageImpact                    uint16 = 5287
	SkillShcShadowSense                     uint16 = 5288
	SkillShcEternalSlash                    uint16 = 5289
	SkillShcPotentVenom                     uint16 = 5290
	SkillShcShadowStab                      uint16 = 5291
	SkillShcImpactCrater                    uint16 = 5292
	SkillShcEnchantingShadow                uint16 = 5293
	SkillShcFatalShadowCrow                 uint16 = 5294
	SkillCdReparatio                        uint16 = 5268
	SkillCdMedialeVotum                     uint16 = 5269
	SkillCdMaceBookM                        uint16 = 5270
	SkillCdArgutusVita                      uint16 = 5271
	SkillCdArgutusTelum                     uint16 = 5272
	SkillCdArbitrium                        uint16 = 5273
	SkillCdArbitriumAtk                     uint16 = 5274
	SkillCdPresensAcies                     uint16 = 5275
	SkillCdFidusAnimus                      uint16 = 5276
	SkillCdEffligo                          uint16 = 5277
	SkillCdCompetentia                      uint16 = 5278
	SkillCdPneumaticusProcella              uint16 = 5279
	SkillCdDilectioHeal                     uint16 = 5280
	SkillCdReligio                          uint16 = 5281
	SkillCdBenedictum                       uint16 = 5282
	SkillCdPetitio                          uint16 = 5283
	SkillCdFramen                           uint16 = 5284
	SkillBoBionicPharmacy                   uint16 = 5336
	SkillBoBionicsM                         uint16 = 5337
	SkillBoTheWholeProtection               uint16 = 5338
	SkillBoAdvanceProtection                uint16 = 5339
	SkillBoAcidifiedZoneWater               uint16 = 5340
	SkillBoAcidifiedZoneGround              uint16 = 5341
	SkillBoAcidifiedZoneWind                uint16 = 5342
	SkillBoAcidifiedZoneFire                uint16 = 5343
	SkillBoWoodenwarrior                    uint16 = 5344
	SkillBoWoodenFairy                      uint16 = 5345
	SkillBoCreeper                          uint16 = 5346
	SkillBoResearchreport                   uint16 = 5347
	SkillBoHelltree                         uint16 = 5348
	SkillWhAdvancedTrap                     uint16 = 5323
	SkillWhWindSign                         uint16 = 5324
	SkillWhNaturefriendly                   uint16 = 5325
	SkillWhHawkrush                         uint16 = 5326
	SkillWhHawkM                            uint16 = 5327
	SkillWhCalamitygale                     uint16 = 5328
	SkillWhHawkboomerang                    uint16 = 5329
	SkillWhGalestorm                        uint16 = 5330
	SkillWhDeepblindtrap                    uint16 = 5331
	SkillWhSolidtrap                        uint16 = 5332
	SkillWhSwifttrap                        uint16 = 5333
	SkillWhCresciveBolt                     uint16 = 5334
	SkillWhFlametrap                        uint16 = 5335
	SkillTrStageManner                      uint16 = 5349
	SkillTrRetrospection                    uint16 = 5350
	SkillTrMysticSymphony                   uint16 = 5351
	SkillTrKvasirSonata                     uint16 = 5352
	SkillTrRoseblossom                      uint16 = 5353
	SkillTrRoseblossomAtk                   uint16 = 5354
	SkillTrRhythmshooting                   uint16 = 5355
	SkillTrMetalicFury                      uint16 = 5356
	SkillTrSoundblend                       uint16 = 5357
	SkillTrGefNocturn                       uint16 = 5358
	SkillTrRokiCapriccio                    uint16 = 5359
	SkillTrAinRhapsody                      uint16 = 5360
	SkillTrMusicalInterlude                 uint16 = 5361
	SkillTrJawaiiSerenade                   uint16 = 5362
	SkillTrNipelheimRequiem                 uint16 = 5363
	SkillTrPronMarch                        uint16 = 5364
	SkillAbcDaggerAndBowM                   uint16 = 5311
	SkillAbcMagicSwordM                     uint16 = 5312
	SkillAbcStripShadow                     uint16 = 5313
	SkillAbcAbyssDagger                     uint16 = 5314
	SkillAbcUnluckyRush                     uint16 = 5315
	SkillAbcChainReactionShot               uint16 = 5316
	SkillAbcFromTheAbyss                    uint16 = 5317
	SkillAbcAbyssSlayer                     uint16 = 5318
	SkillAbcAbyssStrike                     uint16 = 5319
	SkillAbcDeftStab                        uint16 = 5320
	SkillAbcAbyssSquare                     uint16 = 5321
	SkillAbcFrenzyShot                      uint16 = 5322
	SkillAbcChainReactionShotAtk            uint16 = 5382
	SkillAbcFromTheAbyssAtk                 uint16 = 5383
	SkillNPCBoThrowrock                     uint16 = 5384
	SkillNPCBoWoodenAttack                  uint16 = 5385
	SkillNPCBoHellHowling                   uint16 = 5386
	SkillNPCBoHellDusty                     uint16 = 5387
	SkillNPCBoFairyDusty                    uint16 = 5388
	SkillMtAxeStomp                         uint16 = 5295
	SkillMtRushQuake                        uint16 = 5296
	SkillMtMMachine                         uint16 = 5297
	SkillMtAMachine                         uint16 = 5298
	SkillMtDMachine                         uint16 = 5299
	SkillMtTwoaxedef                        uint16 = 5300
	SkillMtAbrM                             uint16 = 5301
	SkillMtSummonAbrBattleWarior            uint16 = 5302
	SkillMtSummonAbrDualCannon              uint16 = 5303
	SkillMtSummonAbrMotherNet               uint16 = 5304
	SkillMtSummonAbrInfinity                uint16 = 5305
	SkillAbrBattleBuster                    uint16 = 8601
	SkillAbrDualCannonFire                  uint16 = 8602
	SkillAbrNetRepair                       uint16 = 8603
	SkillAbrNetSupport                      uint16 = 8604
	SkillAbrInfinityBuster                  uint16 = 8605
	SkillEmMagicBookM                       uint16 = 5365
	SkillEmSpellEnchanting                  uint16 = 5366
	SkillEmActivityBurn                     uint16 = 5367
	SkillEmIncreasingActivity               uint16 = 5368
	SkillEmDiamondStorm                     uint16 = 5369
	SkillEmLightningLand                    uint16 = 5370
	SkillEmVenomSwamp                       uint16 = 5371
	SkillEmConflagration                    uint16 = 5372
	SkillEmTerraDrive                       uint16 = 5373
	SkillEmElementalSpiritM                 uint16 = 5374
	SkillEmSummonElementalArdor             uint16 = 5375
	SkillEmSummonElementalDiluvio           uint16 = 5376
	SkillEmSummonElementalProcella          uint16 = 5377
	SkillEmSummonElementalTerremotus        uint16 = 5378
	SkillEmSummonElementalSerpens           uint16 = 5379
	SkillEmElementalBuster                  uint16 = 5380
	SkillEmElementalVeil                    uint16 = 5381
	SkillEmElementalBusterFire              uint16 = 5389
	SkillEmElementalBusterWater             uint16 = 5390
	SkillEmElementalBusterWind              uint16 = 5391
	SkillEmElementalBusterGround            uint16 = 5392
	SkillEmElementalBusterPoison            uint16 = 5393
	SkillNwPFI                              uint16 = 5401
	SkillNwGrenadeMastery                   uint16 = 5402
	SkillNwIntensiveAim                     uint16 = 5403
	SkillNwGrenadeFragment                  uint16 = 5404
	SkillNwTheVigilanteAtNight              uint16 = 5405
	SkillNwOnlyOneBullet                    uint16 = 5406
	SkillNwSpiralShooting                   uint16 = 5407
	SkillNwMagazineForOne                   uint16 = 5408
	SkillNwWildFire                         uint16 = 5409
	SkillNwBasicGrenade                     uint16 = 5410
	SkillNwHastyFireInTheHole               uint16 = 5411
	SkillNwGrenadesDropping                 uint16 = 5412
	SkillNwAutoFiringLauncher               uint16 = 5413
	SkillNwHiddenCard                       uint16 = 5414
	SkillNwMissionBombard                   uint16 = 5415
	SkillSoaTalismanMastery                 uint16 = 5416
	SkillSoaSoulMastery                     uint16 = 5417
	SkillSoaTalismanOfProtection            uint16 = 5418
	SkillSoaTalismanOfWarrior               uint16 = 5419
	SkillSoaTalismanOfMagician              uint16 = 5420
	SkillSoaSoulGathering                   uint16 = 5421
	SkillSoaTotemOfTutelary                 uint16 = 5422
	SkillSoaTalismanOfFiveElements          uint16 = 5423
	SkillSoaTalismanOfSoulStealing          uint16 = 5424
	SkillSoaExorcismOfMaliciousSoul         uint16 = 5425
	SkillSoaTalismanOfBlueDragon            uint16 = 5426
	SkillSoaTalismanOfWhiteTiger            uint16 = 5427
	SkillSoaTalismanOfRedPhoenix            uint16 = 5428
	SkillSoaTalismanOfBlackTortoise         uint16 = 5429
	SkillSoaTalismanOfFourBearingGod        uint16 = 5430
	SkillSoaCircleOfDirectionsAndElementals uint16 = 5431
	SkillSoaSoulOfHeavenAndEarth            uint16 = 5432
	SkillShMysticalCreatureMastery          uint16 = 5433
	SkillShCommuneWithChulHo                uint16 = 5434
	SkillShChulHoSonicClaw                  uint16 = 5435
	SkillShHowlingOfChulHo                  uint16 = 5436
	SkillShHogogongStrike                   uint16 = 5437
	SkillShCommuneWithKiSul                 uint16 = 5438
	SkillShKiSulWaterSpraying               uint16 = 5439
	SkillShMarineFestivalOfKiSul            uint16 = 5440
	SkillShSandyFestivalOfKiSul             uint16 = 5441
	SkillShKiSulRampage                     uint16 = 5442
	SkillShCommuneWithHyunRok               uint16 = 5443
	SkillShColorsOfHyunRok                  uint16 = 5444
	SkillShHyunRoksBreeze                   uint16 = 5445
	SkillShHyunRokCannon                    uint16 = 5446
	SkillShTemporaryCommunion               uint16 = 5447
	SkillShBlessingOfMysticalCreatures      uint16 = 5448
	SkillHnSelfstudyTatics                  uint16 = 5449
	SkillHnSelfstudySocery                  uint16 = 5450
	SkillHnDoublebowlingbash                uint16 = 5451
	SkillHnMegaSonicBlow                    uint16 = 5452
	SkillHnShieldChainRush                  uint16 = 5453
	SkillHnSpiralPierceMax                  uint16 = 5454
	SkillHnMeteorStormBuster                uint16 = 5455
	SkillHnJupitelThunderStorm              uint16 = 5456
	SkillHnJackFrostNova                    uint16 = 5457
	SkillHnHellsDrive                       uint16 = 5458
	SkillHnGroundGravitation                uint16 = 5459
	SkillHnNapalmVulcanStrike               uint16 = 5460
	SkillHnBreakinglimit                    uint16 = 5461
	SkillHnRulebreak                        uint16 = 5462
	SkillSkeSkyMastery                      uint16 = 5463
	SkillSkeWarBookMastery                  uint16 = 5464
	SkillSkeRisingSun                       uint16 = 5465
	SkillSkeNoonBlast                       uint16 = 5466
	SkillSkeSunsetBlast                     uint16 = 5467
	SkillSkeRisingMoon                      uint16 = 5468
	SkillSkeMidnightKick                    uint16 = 5469
	SkillSkeDawnBreak                       uint16 = 5470
	SkillSkeTwinklingGalaxy                 uint16 = 5471
	SkillSkeStarBurst                       uint16 = 5472
	SkillSkeStarCannon                      uint16 = 5473
	SkillSkeALLInTheSky                     uint16 = 5474
	SkillSkeEnchantingSky                   uint16 = 5475
	SkillSsTokedasu                         uint16 = 5476
	SkillSsShimiru                          uint16 = 5477
	SkillSsAkumukesu                        uint16 = 5478
	SkillSsShinkirou                        uint16 = 5479
	SkillSsKagegari                         uint16 = 5480
	SkillSsKagenomai                        uint16 = 5481
	SkillSsKagegissen                       uint16 = 5482
	SkillSsFuumashouaku                     uint16 = 5483
	SkillSsFuumakouchiku                    uint16 = 5484
	SkillSsKunaiwaikyoku                    uint16 = 5485
	SkillSsKunaikaiten                      uint16 = 5486
	SkillSsKunaikussetsu                    uint16 = 5487
	SkillSsSekienhou                        uint16 = 5488
	SkillSsReiketsuhou                      uint16 = 5489
	SkillSsRaidenpou                        uint16 = 5490
	SkillSsKinryuuhou                       uint16 = 5491
	SkillSsAntenpou                         uint16 = 5492
	SkillSsKageakumu                        uint16 = 5493
	SkillSsHitouakumu                       uint16 = 5494
	SkillSsAnkokuryuuakumu                  uint16 = 5495
	SkillNwTheVigilanteAtNightGunGatling    uint16 = 5496
	SkillNwTheVigilanteAtNightGunShotgun    uint16 = 5497
	SkillDkDragonicBreath                   uint16 = 6001
	SkillMtSparkBlaster                     uint16 = 6002
	SkillMtTripleLaser                      uint16 = 6003
	SkillMtMightySmash                      uint16 = 6004
	SkillBoExplosivePowder                  uint16 = 6005
	SkillBoMayhemicThorns                   uint16 = 6006
	SkillEmElFlametechnic                   uint16 = 8443
	SkillEmElFlamearmor                     uint16 = 8444
	SkillEmElFlamerock                      uint16 = 8445
	SkillEmElColdForce                      uint16 = 8446
	SkillEmElCrystalArmor                   uint16 = 8447
	SkillEmElAgeOfIce                       uint16 = 8448
	SkillEmElGraceBreeze                    uint16 = 8449
	SkillEmElEyesOfStorm                    uint16 = 8450
	SkillEmElStormWind                      uint16 = 8451
	SkillEmElEarthCare                      uint16 = 8452
	SkillEmElStrongProtection               uint16 = 8453
	SkillEmElAvalanche                      uint16 = 8454
	SkillEmElDeepPoisoning                  uint16 = 8455
	SkillEmElPoisonShield                   uint16 = 8456
	SkillEmElDeadlyPoison                   uint16 = 8457
	SkillHomunBegin                         uint16 = 8000
	SkillHlifHeal                           uint16 = 8001
	SkillHlifAvoid                          uint16 = 8002
	SkillHlifBrain                          uint16 = 8003
	SkillHlifChange                         uint16 = 8004
	SkillHamiCastle                         uint16 = 8005
	SkillHamiDefence                        uint16 = 8006
	SkillHamiSkin                           uint16 = 8007
	SkillHamiBloodlust                      uint16 = 8008
	SkillHfliMoon                           uint16 = 8009
	SkillHfliFleet                          uint16 = 8010
	SkillHfliSpeed                          uint16 = 8011
	SkillHfliSbr44                          uint16 = 8012
	SkillHvanCaprice                        uint16 = 8013
	SkillHvanChaotic                        uint16 = 8014
	SkillHvanInstruct                       uint16 = 8015
	SkillHvanExplosion                      uint16 = 8016
	SkillMutationBasejob                    uint16 = 8017
	SkillMhSummonLegion                     uint16 = 8018
	SkillMhNeedleOfParalyze                 uint16 = 8019
	SkillMhPoisonMist                       uint16 = 8020
	SkillMhPainKiller                       uint16 = 8021
	SkillMhLightOfRegene                    uint16 = 8022
	SkillMhOveredBoost                      uint16 = 8023
	SkillMhEraserCutter                     uint16 = 8024
	SkillMhXenoSlasher                      uint16 = 8025
	SkillMhSilentBreeze                     uint16 = 8026
	SkillMhStyleChange                      uint16 = 8027
	SkillMhSonicCraw                        uint16 = 8028
	SkillMhSilverveinRush                   uint16 = 8029
	SkillMhMidnightFrenzy                   uint16 = 8030
	SkillMhStahlHorn                        uint16 = 8031
	SkillMhGoldeneFerse                     uint16 = 8032
	SkillMhSteinwand                        uint16 = 8033
	SkillMhHeiligeStange                    uint16 = 8034
	SkillMhAngriffsModus                    uint16 = 8035
	SkillMhTinderBreaker                    uint16 = 8036
	SkillMhCbc                              uint16 = 8037
	SkillMhEqc                              uint16 = 8038
	SkillMhMagmaFlow                        uint16 = 8039
	SkillMhGraniticArmor                    uint16 = 8040
	SkillMhLavaSlide                        uint16 = 8041
	SkillMhPyroclastic                      uint16 = 8042
	SkillMhVolcanicAsh                      uint16 = 8043
	SkillMhBlastForge                       uint16 = 8044
	SkillMhTempering                        uint16 = 8045
	SkillMhClassyFlutter                    uint16 = 8046
	SkillMhTwisterCutter                    uint16 = 8047
	SkillMhAbsoluteZephyr                   uint16 = 8048
	SkillMhBrushupClaw                      uint16 = 8049
	SkillMhBlazingAndFurious                uint16 = 8050
	SkillMhTheOneFighterRises               uint16 = 8051
	SkillMhPolishingNeedle                  uint16 = 8052
	SkillMhToxinOfMandara                   uint16 = 8053
	SkillMhNeedleStinger                    uint16 = 8054
	SkillMhLichtGehorn                      uint16 = 8055
	SkillMhGlanzenSpies                     uint16 = 8056
	SkillMhHeiligePferd                     uint16 = 8057
	SkillMhGoldeneTone                      uint16 = 8058
	SkillMhBlazingLava                      uint16 = 8059
	SkillMhLast                             uint16 = 8060
	SkillHomunLast                          uint16 = 8061
	SkillMercenaryBegin                     uint16 = 8200
	SkillMsBash                             uint16 = 8201
	SkillMsMagnum                           uint16 = 8202
	SkillMsBowlingbash                      uint16 = 8203
	SkillMsParrying                         uint16 = 8204
	SkillMsReflectshield                    uint16 = 8205
	SkillMsBerserk                          uint16 = 8206
	SkillMaDouble                           uint16 = 8207
	SkillMaShower                           uint16 = 8208
	SkillMaSkidtrap                         uint16 = 8209
	SkillMaLandmine                         uint16 = 8210
	SkillMaSandman                          uint16 = 8211
	SkillMaFreezingtrap                     uint16 = 8212
	SkillMaRemovetrap                       uint16 = 8213
	SkillMaChargearrow                      uint16 = 8214
	SkillMaSharpshooting                    uint16 = 8215
	SkillMlPierce                           uint16 = 8216
	SkillMlBrandish                         uint16 = 8217
	SkillMlSpiralpierce                     uint16 = 8218
	SkillMlDefender                         uint16 = 8219
	SkillMlAutoguard                        uint16 = 8220
	SkillMlDevotion                         uint16 = 8221
	SkillMerMagnificat                      uint16 = 8222
	SkillMerQuicken                         uint16 = 8223
	SkillMerSight                           uint16 = 8224
	SkillMerCrash                           uint16 = 8225
	SkillMerRegain                          uint16 = 8226
	SkillMerTender                          uint16 = 8227
	SkillMerBenediction                     uint16 = 8228
	SkillMerRecuperate                      uint16 = 8229
	SkillMerMentalcure                      uint16 = 8230
	SkillMerCompress                        uint16 = 8231
	SkillMerProvoke                         uint16 = 8232
	SkillMerAutoberserk                     uint16 = 8233
	SkillMerDecagi                          uint16 = 8234
	SkillMerScapegoat                       uint16 = 8235
	SkillMerLexdivina                       uint16 = 8236
	SkillMerEstimation                      uint16 = 8237
	SkillMerKyrie                           uint16 = 8238
	SkillMerBlessing                        uint16 = 8239
	SkillMerIncagi                          uint16 = 8240
	SkillMerInvincibleoff2                  uint16 = 8241
	SkillMercenaryLast                      uint16 = 8242
	SkillElementalBegin                     uint16 = 8400
	SkillElCircleOfFire                     uint16 = 8401
	SkillElFireCloak                        uint16 = 8402
	SkillElFireMantle                       uint16 = 8403
	SkillElWaterScreen                      uint16 = 8404
	SkillElWaterDrop                        uint16 = 8405
	SkillElWaterBarrier                     uint16 = 8406
	SkillElWindStep                         uint16 = 8407
	SkillElWindCurtain                      uint16 = 8408
	SkillElZephyr                           uint16 = 8409
	SkillElSolidSkin                        uint16 = 8410
	SkillElStoneShield                      uint16 = 8411
	SkillElPowerOfGaia                      uint16 = 8412
	SkillElPyrotechnic                      uint16 = 8413
	SkillElHeater                           uint16 = 8414
	SkillElTropic                           uint16 = 8415
	SkillElAquaplay                         uint16 = 8416
	SkillElCooler                           uint16 = 8417
	SkillElChillyAir                        uint16 = 8418
	SkillElGust                             uint16 = 8419
	SkillElBlast                            uint16 = 8420
	SkillElWildStorm                        uint16 = 8421
	SkillElPetrology                        uint16 = 8422
	SkillElCursedSoil                       uint16 = 8423
	SkillElUpheaval                         uint16 = 8424
	SkillElFireArrow                        uint16 = 8425
	SkillElFireBomb                         uint16 = 8426
	SkillElFireBombAtk                      uint16 = 8427
	SkillElFireWave                         uint16 = 8428
	SkillElFireWaveAtk                      uint16 = 8429
	SkillElIceNeedle                        uint16 = 8430
	SkillElWaterScrew                       uint16 = 8431
	SkillElWaterScrewAtk                    uint16 = 8432
	SkillElTidalWeapon                      uint16 = 8433
	SkillElWindSlash                        uint16 = 8434
	SkillElHurricane                        uint16 = 8435
	SkillElHurricaneAtk                     uint16 = 8436
	SkillElTypoonMis                        uint16 = 8437
	SkillElTypoonMisAtk                     uint16 = 8438
	SkillElStoneHammer                      uint16 = 8439
	SkillElRockCrusher                      uint16 = 8440
	SkillElRockCrusherAtk                   uint16 = 8441
	SkillElStoneRain                        uint16 = 8442
	SkillElementalLast                      uint16 = 8443
	SkillFollowerNPCReset                   uint16 = 9999
	SkillGdApproval                         uint16 = 10000
	SkillGdKafracontract                    uint16 = 10001
	SkillGdGuardresearch                    uint16 = 10002
	SkillGdGuardup                          uint16 = 10003
	SkillGdExtension                        uint16 = 10004
	SkillGdGloryguild                       uint16 = 10005
	SkillGdLeadership                       uint16 = 10006
	SkillGdGlorywounds                      uint16 = 10007
	SkillGdSoulcold                         uint16 = 10008
	SkillGdHawkeyes                         uint16 = 10009
	SkillGdBattleorder                      uint16 = 10010
	SkillGdRegeneration                     uint16 = 10011
	SkillGdRestore                          uint16 = 10012
	SkillGdEmergencycall                    uint16 = 10013
	SkillGdDevelopment                      uint16 = 10014
	SkillGdItememergencycall                uint16 = 10015
	SkillGdGuildStorage                     uint16 = 10016
	SkillGdChargeshoutFlag                  uint16 = 10017
	SkillGdChargeshoutBeating               uint16 = 10018
	SkillGdEmergencyMove                    uint16 = 10019
	SkillGdLast                             uint16 = 10020
	SkillSysFirstjoblv                      uint16 = 10100
	SkillSysSecondjoblv                     uint16 = 10101
	SkillScript000                          uint16 = 11000
	SkillItemSavageSteak                    uint16 = 11000
	SkillItemCocktailWargBlood              uint16 = 11001
	SkillItemMinorBbq                       uint16 = 11002
	SkillItemSiromaIceTea                   uint16 = 11003
	SkillItemDroceraHerbSteamed             uint16 = 11004
	SkillItemPuttiTailsNoodles              uint16 = 11005
	SkillItemBananaBomb                     uint16 = 11006
	SkillScript999                          uint16 = 11999
	SkillEfstDressUp                        uint16 = 12000
)

var SkillResourceName = map[uint16]string{
	SkillNVBasic:                            "NV_BASIC",
	SkillSMSword:                            "SM_SWORD",
	SkillSMTwohand:                          "SM_TWOHAND",
	SkillSMRecovery:                         "SM_RECOVERY",
	SkillSMBash:                             "SM_BASH",
	SkillSMProvoke:                          "SM_PROVOKE",
	SkillSMMagnum:                           "SM_MAGNUM",
	SkillSMEndure:                           "SM_ENDURE",
	SkillMGSrecovery:                        "MG_SRECOVERY",
	SkillMGSight:                            "MG_SIGHT",
	SkillMGNapalmbeat:                       "MG_NAPALMBEAT",
	SkillMGSafetywall:                       "MG_SAFETYWALL",
	SkillMGSoulstrike:                       "MG_SOULSTRIKE",
	SkillMGColdbolt:                         "MG_COLDBOLT",
	SkillMGFrostdiver:                       "MG_FROSTDIVER",
	SkillMGStonecurse:                       "MG_STONECURSE",
	SkillMGFireball:                         "MG_FIREBALL",
	SkillMGFirewall:                         "MG_FIREWALL",
	SkillMGFirebolt:                         "MG_FIREBOLT",
	SkillMGLightningbolt:                    "MG_LIGHTNINGBOLT",
	SkillMGThunderstorm:                     "MG_THUNDERSTORM",
	SkillALDp:                               "AL_DP",
	SkillALDemonbane:                        "AL_DEMONBANE",
	SkillALRuwach:                           "AL_RUWACH",
	SkillALPneuma:                           "AL_PNEUMA",
	SkillALTeleport:                         "AL_TELEPORT",
	SkillALWarp:                             "AL_WARP",
	SkillALHeal:                             "AL_HEAL",
	SkillALIncagi:                           "AL_INCAGI",
	SkillALDecagi:                           "AL_DECAGI",
	SkillALHolywater:                        "AL_HOLYWATER",
	SkillALCrucis:                           "AL_CRUCIS",
	SkillALAngelus:                          "AL_ANGELUS",
	SkillALBlessing:                         "AL_BLESSING",
	SkillALCure:                             "AL_CURE",
	SkillMCInccarry:                         "MC_INCCARRY",
	SkillMCDiscount:                         "MC_DISCOUNT",
	SkillMCOvercharge:                       "MC_OVERCHARGE",
	SkillMCPushcart:                         "MC_PUSHCART",
	SkillMCIdentify:                         "MC_IDENTIFY",
	SkillMCVending:                          "MC_VENDING",
	SkillMCMammonite:                        "MC_MAMMONITE",
	SkillACOwl:                              "AC_OWL",
	SkillACVulture:                          "AC_VULTURE",
	SkillACConcentration:                    "AC_CONCENTRATION",
	SkillACDouble:                           "AC_DOUBLE",
	SkillACShower:                           "AC_SHOWER",
	SkillTFDouble:                           "TF_DOUBLE",
	SkillTFMiss:                             "TF_MISS",
	SkillTFSteal:                            "TF_STEAL",
	SkillTFHiding:                           "TF_HIDING",
	SkillTFPoison:                           "TF_POISON",
	SkillTFDetoxify:                         "TF_DETOXIFY",
	SkillALLResurrection:                    "ALL_RESURRECTION",
	SkillKNSpearmastery:                     "KN_SPEARMASTERY",
	SkillKNPierce:                           "KN_PIERCE",
	SkillKNBrandishspear:                    "KN_BRANDISHSPEAR",
	SkillKNSpearstab:                        "KN_SPEARSTAB",
	SkillKNSpearboomerang:                   "KN_SPEARBOOMERANG",
	SkillKNTwohandquicken:                   "KN_TWOHANDQUICKEN",
	SkillKNAutocounter:                      "KN_AUTOCOUNTER",
	SkillKNBowlingbash:                      "KN_BOWLINGBASH",
	SkillKNRiding:                           "KN_RIDING",
	SkillKNCavaliermastery:                  "KN_CAVALIERMASTERY",
	SkillPRMacemastery:                      "PR_MACEMASTERY",
	SkillPRImpositio:                        "PR_IMPOSITIO",
	SkillPRSuffragium:                       "PR_SUFFRAGIUM",
	SkillPRAspersio:                         "PR_ASPERSIO",
	SkillPRBenedictio:                       "PR_BENEDICTIO",
	SkillPRSanctuary:                        "PR_SANCTUARY",
	SkillPRSlowpoison:                       "PR_SLOWPOISON",
	SkillPRStrecovery:                       "PR_STRECOVERY",
	SkillPRKyrie:                            "PR_KYRIE",
	SkillPRMagnificat:                       "PR_MAGNIFICAT",
	SkillPRGloria:                           "PR_GLORIA",
	SkillPRLexdivina:                        "PR_LEXDIVINA",
	SkillPRTurnundead:                       "PR_TURNUNDEAD",
	SkillPRLexaeterna:                       "PR_LEXAETERNA",
	SkillPRMagnus:                           "PR_MAGNUS",
	SkillWZFirepillar:                       "WZ_FIREPILLAR",
	SkillWZSightrasher:                      "WZ_SIGHTRASHER",
	SkillWZFireivy:                          "WZ_FIREIVY",
	SkillWZMeteor:                           "WZ_METEOR",
	SkillWZJupitel:                          "WZ_JUPITEL",
	SkillWZVermilion:                        "WZ_VERMILION",
	SkillWZWaterball:                        "WZ_WATERBALL",
	SkillWZIcewall:                          "WZ_ICEWALL",
	SkillWZFrostnova:                        "WZ_FROSTNOVA",
	SkillWZStormgust:                        "WZ_STORMGUST",
	SkillWZEarthspike:                       "WZ_EARTHSPIKE",
	SkillWZHeavendrive:                      "WZ_HEAVENDRIVE",
	SkillWZQuagmire:                         "WZ_QUAGMIRE",
	SkillWZEstimation:                       "WZ_ESTIMATION",
	SkillBSIron:                             "BS_IRON",
	SkillBSSteel:                            "BS_STEEL",
	SkillBSEnchantedstone:                   "BS_ENCHANTEDSTONE",
	SkillBSOrideocon:                        "BS_ORIDEOCON",
	SkillBSDagger:                           "BS_DAGGER",
	SkillBSSword:                            "BS_SWORD",
	SkillBSTwohandsword:                     "BS_TWOHANDSWORD",
	SkillBSAxe:                              "BS_AXE",
	SkillBSMace:                             "BS_MACE",
	SkillBSKnuckle:                          "BS_KNUCKLE",
	SkillBSSpear:                            "BS_SPEAR",
	SkillBSHiltbinding:                      "BS_HILTBINDING",
	SkillBSFindingore:                       "BS_FINDINGORE",
	SkillBSWeaponresearch:                   "BS_WEAPONRESEARCH",
	SkillBSRepairweapon:                     "BS_REPAIRWEAPON",
	SkillBSSkintemper:                       "BS_SKINTEMPER",
	SkillBSHammerfall:                       "BS_HAMMERFALL",
	SkillBSAdrenaline:                       "BS_ADRENALINE",
	SkillBSWeaponperfect:                    "BS_WEAPONPERFECT",
	SkillBSOverthrust:                       "BS_OVERTHRUST",
	SkillBSMaximize:                         "BS_MAXIMIZE",
	SkillHTSkidtrap:                         "HT_SKIDTRAP",
	SkillHTLandmine:                         "HT_LANDMINE",
	SkillHTAnklesnare:                       "HT_ANKLESNARE",
	SkillHTShockwave:                        "HT_SHOCKWAVE",
	SkillHTSandman:                          "HT_SANDMAN",
	SkillHTFlasher:                          "HT_FLASHER",
	SkillHTFreezingtrap:                     "HT_FREEZINGTRAP",
	SkillHTBlastmine:                        "HT_BLASTMINE",
	SkillHTClaymoretrap:                     "HT_CLAYMORETRAP",
	SkillHTRemovetrap:                       "HT_REMOVETRAP",
	SkillHTTalkiebox:                        "HT_TALKIEBOX",
	SkillHTBeastbane:                        "HT_BEASTBANE",
	SkillHTFalcon:                           "HT_FALCON",
	SkillHTSteelcrow:                        "HT_STEELCROW",
	SkillHTBlitzbeat:                        "HT_BLITZBEAT",
	SkillHTDetecting:                        "HT_DETECTING",
	SkillHTSpringtrap:                       "HT_SPRINGTRAP",
	SkillASRight:                            "AS_RIGHT",
	SkillASLeft:                             "AS_LEFT",
	SkillASKatar:                            "AS_KATAR",
	SkillASCloaking:                         "AS_CLOAKING",
	SkillASSonicblow:                        "AS_SONICBLOW",
	SkillASGrimtooth:                        "AS_GRIMTOOTH",
	SkillASEnchantpoison:                    "AS_ENCHANTPOISON",
	SkillASPoisonreact:                      "AS_POISONREACT",
	SkillASVenomdust:                        "AS_VENOMDUST",
	SkillASSplasher:                         "AS_SPLASHER",
	SkillNVFirstaid:                         "NV_FIRSTAID",
	SkillNVTrickdead:                        "NV_TRICKDEAD",
	SkillSMMovingrecovery:                   "SM_MOVINGRECOVERY",
	SkillSMFatalblow:                        "SM_FATALBLOW",
	SkillSMAutoberserk:                      "SM_AUTOBERSERK",
	SkillACMakingarrow:                      "AC_MAKINGARROW",
	SkillACChargearrow:                      "AC_CHARGEARROW",
	SkillTFSprinklesand:                     "TF_SPRINKLESAND",
	SkillTFBacksliding:                      "TF_BACKSLIDING",
	SkillTFPickstone:                        "TF_PICKSTONE",
	SkillTFThrowstone:                       "TF_THROWSTONE",
	SkillMCCartrevolution:                   "MC_CARTREVOLUTION",
	SkillMCChangecart:                       "MC_CHANGECART",
	SkillMCLoud:                             "MC_LOUD",
	SkillALHolylight:                        "AL_HOLYLIGHT",
	SkillMGEnergycoat:                       "MG_ENERGYCOAT",
	SkillNPCPiercingatt:                     "NPC_PIERCINGATT",
	SkillNPCMentalbreaker:                   "NPC_MENTALBREAKER",
	SkillNPCRangeattack:                     "NPC_RANGEATTACK",
	SkillNPCAttrichange:                     "NPC_ATTRICHANGE",
	SkillNPCChangewater:                     "NPC_CHANGEWATER",
	SkillNPCChangeground:                    "NPC_CHANGEGROUND",
	SkillNPCChangefire:                      "NPC_CHANGEFIRE",
	SkillNPCChangewind:                      "NPC_CHANGEWIND",
	SkillNPCChangepoison:                    "NPC_CHANGEPOISON",
	SkillNPCChangeholy:                      "NPC_CHANGEHOLY",
	SkillNPCChangedarkness:                  "NPC_CHANGEDARKNESS",
	SkillNPCChangetelekinesis:               "NPC_CHANGETELEKINESIS",
	SkillNPCCriticalslash:                   "NPC_CRITICALSLASH",
	SkillNPCComboattack:                     "NPC_COMBOATTACK",
	SkillNPCGuidedattack:                    "NPC_GUIDEDATTACK",
	SkillNPCSelfdestruction:                 "NPC_SELFDESTRUCTION",
	SkillNPCSplashattack:                    "NPC_SPLASHATTACK",
	SkillNPCSuicide:                         "NPC_SUICIDE",
	SkillNPCPoison:                          "NPC_POISON",
	SkillNPCBlindattack:                     "NPC_BLINDATTACK",
	SkillNPCSilenceattack:                   "NPC_SILENCEATTACK",
	SkillNPCStunattack:                      "NPC_STUNATTACK",
	SkillNPCPetrifyattack:                   "NPC_PETRIFYATTACK",
	SkillNPCCurseattack:                     "NPC_CURSEATTACK",
	SkillNPCSleepattack:                     "NPC_SLEEPATTACK",
	SkillNPCRandomattack:                    "NPC_RANDOMATTACK",
	SkillNPCWaterattack:                     "NPC_WATERATTACK",
	SkillNPCGroundattack:                    "NPC_GROUNDATTACK",
	SkillNPCFireattack:                      "NPC_FIREATTACK",
	SkillNPCWindattack:                      "NPC_WINDATTACK",
	SkillNPCPoisonattack:                    "NPC_POISONATTACK",
	SkillNPCHolyattack:                      "NPC_HOLYATTACK",
	SkillNPCDarknessattack:                  "NPC_DARKNESSATTACK",
	SkillNPCTelekinesisattack:               "NPC_TELEKINESISATTACK",
	SkillNPCMagicalattack:                   "NPC_MAGICALATTACK",
	SkillNPCMetamorphosis:                   "NPC_METAMORPHOSIS",
	SkillNPCProvocation:                     "NPC_PROVOCATION",
	SkillNPCSmoking:                         "NPC_SMOKING",
	SkillNPCSummonslave:                     "NPC_SUMMONSLAVE",
	SkillNPCEmotion:                         "NPC_EMOTION",
	SkillNPCTransformation:                  "NPC_TRANSFORMATION",
	SkillNPCBlooddrain:                      "NPC_BLOODDRAIN",
	SkillNPCEnergydrain:                     "NPC_ENERGYDRAIN",
	SkillNPCKeeping:                         "NPC_KEEPING",
	SkillNPCDarkbreath:                      "NPC_DARKBREATH",
	SkillNPCDarkblessing:                    "NPC_DARKBLESSING",
	SkillNPCBarrier:                         "NPC_BARRIER",
	SkillNPCDefender:                        "NPC_DEFENDER",
	SkillNPCLick:                            "NPC_LICK",
	SkillNPCHallucination:                   "NPC_HALLUCINATION",
	SkillNPCRebirth:                         "NPC_REBIRTH",
	SkillNPCSummonmonster:                   "NPC_SUMMONMONSTER",
	SkillRGSnatcher:                         "RG_SNATCHER",
	SkillRGStealcoin:                        "RG_STEALCOIN",
	SkillRGBackstap:                         "RG_BACKSTAP",
	SkillRGTunneldrive:                      "RG_TUNNELDRIVE",
	SkillRGRaid:                             "RG_RAID",
	SkillRGStripweapon:                      "RG_STRIPWEAPON",
	SkillRGStripshield:                      "RG_STRIPSHIELD",
	SkillRGStriparmor:                       "RG_STRIPARMOR",
	SkillRGStriphelm:                        "RG_STRIPHELM",
	SkillRGIntimidate:                       "RG_INTIMIDATE",
	SkillRGGraffiti:                         "RG_GRAFFITI",
	SkillRGFlaggraffiti:                     "RG_FLAGGRAFFITI",
	SkillRGCleaner:                          "RG_CLEANER",
	SkillRGGangster:                         "RG_GANGSTER",
	SkillRGCompulsion:                       "RG_COMPULSION",
	SkillRGPlagiarism:                       "RG_PLAGIARISM",
	SkillAMAxemastery:                       "AM_AXEMASTERY",
	SkillAMLearningpotion:                   "AM_LEARNINGPOTION",
	SkillAMPharmacy:                         "AM_PHARMACY",
	SkillAMDemonstration:                    "AM_DEMONSTRATION",
	SkillAMAcidterror:                       "AM_ACIDTERROR",
	SkillAMPotionpitcher:                    "AM_POTIONPITCHER",
	SkillAMCannibalize:                      "AM_CANNIBALIZE",
	SkillAMSpheremine:                       "AM_SPHEREMINE",
	SkillAMCpWeapon:                         "AM_CP_WEAPON",
	SkillAMCpShield:                         "AM_CP_SHIELD",
	SkillAMCpArmor:                          "AM_CP_ARMOR",
	SkillAMCpHelm:                           "AM_CP_HELM",
	SkillAMBioethics:                        "AM_BIOETHICS",
	SkillAMBiotechnology:                    "AM_BIOTECHNOLOGY",
	SkillAMCreatecreature:                   "AM_CREATECREATURE",
	SkillAMCultivation:                      "AM_CULTIVATION",
	SkillAMFlamecontrol:                     "AM_FLAMECONTROL",
	SkillAMCallhomun:                        "AM_CALLHOMUN",
	SkillAMRest:                             "AM_REST",
	SkillAMDrillmaster:                      "AM_DRILLMASTER",
	SkillAMHealhomun:                        "AM_HEALHOMUN",
	SkillAMResurrecthomun:                   "AM_RESURRECTHOMUN",
	SkillCRTrust:                            "CR_TRUST",
	SkillCRAutoguard:                        "CR_AUTOGUARD",
	SkillCRShieldcharge:                     "CR_SHIELDCHARGE",
	SkillCRShieldboomerang:                  "CR_SHIELDBOOMERANG",
	SkillCRReflectshield:                    "CR_REFLECTSHIELD",
	SkillCRHolycross:                        "CR_HOLYCROSS",
	SkillCRGrandcross:                       "CR_GRANDCROSS",
	SkillCRDevotion:                         "CR_DEVOTION",
	SkillCRProvidence:                       "CR_PROVIDENCE",
	SkillCRDefender:                         "CR_DEFENDER",
	SkillCRSpearquicken:                     "CR_SPEARQUICKEN",
	SkillMOIronhand:                         "MO_IRONHAND",
	SkillMOSpiritsrecovery:                  "MO_SPIRITSRECOVERY",
	SkillMOCallspirits:                      "MO_CALLSPIRITS",
	SkillMOAbsorbspirits:                    "MO_ABSORBSPIRITS",
	SkillMOTripleattack:                     "MO_TRIPLEATTACK",
	SkillMOBodyrelocation:                   "MO_BODYRELOCATION",
	SkillMODodge:                            "MO_DODGE",
	SkillMOInvestigate:                      "MO_INVESTIGATE",
	SkillMOFingeroffensive:                  "MO_FINGEROFFENSIVE",
	SkillMOSteelbody:                        "MO_STEELBODY",
	SkillMOBladestop:                        "MO_BLADESTOP",
	SkillMOExplosionspirits:                 "MO_EXPLOSIONSPIRITS",
	SkillMOExtremityfist:                    "MO_EXTREMITYFIST",
	SkillMOChaincombo:                       "MO_CHAINCOMBO",
	SkillMOCombofinish:                      "MO_COMBOFINISH",
	SkillSAAdvancedbook:                     "SA_ADVANCEDBOOK",
	SkillSACastcancel:                       "SA_CASTCANCEL",
	SkillSAMagicrod:                         "SA_MAGICROD",
	SkillSASpellbreaker:                     "SA_SPELLBREAKER",
	SkillSAFreecast:                         "SA_FREECAST",
	SkillSAAutospell:                        "SA_AUTOSPELL",
	SkillSAFlamelauncher:                    "SA_FLAMELAUNCHER",
	SkillSAFrostweapon:                      "SA_FROSTWEAPON",
	SkillSALightningloader:                  "SA_LIGHTNINGLOADER",
	SkillSASeismicweapon:                    "SA_SEISMICWEAPON",
	SkillSADragonology:                      "SA_DRAGONOLOGY",
	SkillSAVolcano:                          "SA_VOLCANO",
	SkillSADeluge:                           "SA_DELUGE",
	SkillSAViolentgale:                      "SA_VIOLENTGALE",
	SkillSALandprotector:                    "SA_LANDPROTECTOR",
	SkillSADispell:                          "SA_DISPELL",
	SkillSAAbracadabra:                      "SA_ABRACADABRA",
	SkillSAMonocell:                         "SA_MONOCELL",
	SkillSAClasschange:                      "SA_CLASSCHANGE",
	SkillSASummonmonster:                    "SA_SUMMONMONSTER",
	SkillSAReverseorcish:                    "SA_REVERSEORCISH",
	SkillSADeath:                            "SA_DEATH",
	SkillSAFortune:                          "SA_FORTUNE",
	SkillSATamingmonster:                    "SA_TAMINGMONSTER",
	SkillSAQuestion:                         "SA_QUESTION",
	SkillSAGravity:                          "SA_GRAVITY",
	SkillSALevelup:                          "SA_LEVELUP",
	SkillSAInstantdeath:                     "SA_INSTANTDEATH",
	SkillSAFullrecovery:                     "SA_FULLRECOVERY",
	SkillSAComa:                             "SA_COMA",
	SkillBDAdaptation:                       "BD_ADAPTATION",
	SkillBDEncore:                           "BD_ENCORE",
	SkillBDLullaby:                          "BD_LULLABY",
	SkillBDRichmankim:                       "BD_RICHMANKIM",
	SkillBDEternalchaos:                     "BD_ETERNALCHAOS",
	SkillBDDrumbattlefield:                  "BD_DRUMBATTLEFIELD",
	SkillBDRingnibelungen:                   "BD_RINGNIBELUNGEN",
	SkillBDRokisweil:                        "BD_ROKISWEIL",
	SkillBDIntoabyss:                        "BD_INTOABYSS",
	SkillBDSiegfried:                        "BD_SIEGFRIED",
	SkillBDRagnarok:                         "BD_RAGNAROK",
	SkillBaMusicallesson:                    "BA_MUSICALLESSON",
	SkillBaMusicalstrike:                    "BA_MUSICALSTRIKE",
	SkillBaDissonance:                       "BA_DISSONANCE",
	SkillBaFrostjoke:                        "BA_FROSTJOKE",
	SkillBaWhistle:                          "BA_WHISTLE",
	SkillBaAssassincross:                    "BA_ASSASSINCROSS",
	SkillBaPoembragi:                        "BA_POEMBRAGI",
	SkillBaAppleidun:                        "BA_APPLEIDUN",
	SkillDCDancinglesson:                    "DC_DANCINGLESSON",
	SkillDCThrowarrow:                       "DC_THROWARROW",
	SkillDCUglydance:                        "DC_UGLYDANCE",
	SkillDCScream:                           "DC_SCREAM",
	SkillDCHumming:                          "DC_HUMMING",
	SkillDCDontforgetme:                     "DC_DONTFORGETME",
	SkillDCFortunekiss:                      "DC_FORTUNEKISS",
	SkillDCServiceforyou:                    "DC_SERVICEFORYOU",
	SkillNPCRandommove:                      "NPC_RANDOMMOVE",
	SkillNPCSpeedup:                         "NPC_SPEEDUP",
	SkillNPCRevenge:                         "NPC_REVENGE",
	SkillWEMale:                             "WE_MALE",
	SkillWEFemale:                           "WE_FEMALE",
	SkillWECallpartner:                      "WE_CALLPARTNER",
	SkillITMTomahawk:                        "ITM_TOMAHAWK",
	SkillNPCDarkcross:                       "NPC_DARKCROSS",
	SkillNPCGranddarkness:                   "NPC_GRANDDARKNESS",
	SkillNPCDarkstrike:                      "NPC_DARKSTRIKE",
	SkillNPCDarkthunder:                     "NPC_DARKTHUNDER",
	SkillNPCStop:                            "NPC_STOP",
	SkillNPCWeaponbraker:                    "NPC_WEAPONBRAKER",
	SkillNPCArmorbrake:                      "NPC_ARMORBRAKE",
	SkillNPCHelmbrake:                       "NPC_HELMBRAKE",
	SkillNPCShieldbrake:                     "NPC_SHIELDBRAKE",
	SkillNPCUndeadattack:                    "NPC_UNDEADATTACK",
	SkillNPCChangeundead:                    "NPC_CHANGEUNDEAD",
	SkillNPCPowerup:                         "NPC_POWERUP",
	SkillNPCAgiup:                           "NPC_AGIUP",
	SkillNPCSiegemode:                       "NPC_SIEGEMODE",
	SkillNPCCallslave:                       "NPC_CALLSLAVE",
	SkillNPCInvisible:                       "NPC_INVISIBLE",
	SkillNPCRun:                             "NPC_RUN",
	SkillLKAurablade:                        "LK_AURABLADE",
	SkillLKParrying:                         "LK_PARRYING",
	SkillLKConcentration:                    "LK_CONCENTRATION",
	SkillLKTensionrelax:                     "LK_TENSIONRELAX",
	SkillLKBerserk:                          "LK_BERSERK",
	SkillLKFury:                             "LK_FURY",
	SkillHPAssumptio:                        "HP_ASSUMPTIO",
	SkillHPBasilica:                         "HP_BASILICA",
	SkillHPMeditatio:                        "HP_MEDITATIO",
	SkillHWSouldrain:                        "HW_SOULDRAIN",
	SkillHWMagiccrasher:                     "HW_MAGICCRASHER",
	SkillHWMagicpower:                       "HW_MAGICPOWER",
	SkillPaPressure:                         "PA_PRESSURE",
	SkillPaSacrifice:                        "PA_SACRIFICE",
	SkillPaGospel:                           "PA_GOSPEL",
	SkillChPalmstrike:                       "CH_PALMSTRIKE",
	SkillChTigerfist:                        "CH_TIGERFIST",
	SkillChChaincrush:                       "CH_CHAINCRUSH",
	SkillPFHpconversion:                     "PF_HPCONVERSION",
	SkillPFSoulchange:                       "PF_SOULCHANGE",
	SkillPFSoulburn:                         "PF_SOULBURN",
	SkillASCKatar:                           "ASC_KATAR",
	SkillASCHallucination:                   "ASC_HALLUCINATION",
	SkillASCEdp:                             "ASC_EDP",
	SkillASCBreaker:                         "ASC_BREAKER",
	SkillSNSight:                            "SN_SIGHT",
	SkillSNFalconassault:                    "SN_FALCONASSAULT",
	SkillSNSharpshooting:                    "SN_SHARPSHOOTING",
	SkillSNWindwalk:                         "SN_WINDWALK",
	SkillWSMeltdown:                         "WS_MELTDOWN",
	SkillWSCreatecoin:                       "WS_CREATECOIN",
	SkillWSCreatenugget:                     "WS_CREATENUGGET",
	SkillWSCartboost:                        "WS_CARTBOOST",
	SkillWSSystemcreate:                     "WS_SYSTEMCREATE",
	SkillSTChasewalk:                        "ST_CHASEWALK",
	SkillSTRejectsword:                      "ST_REJECTSWORD",
	SkillSTStealbackpack:                    "ST_STEALBACKPACK",
	SkillCRAlchemy:                          "CR_ALCHEMY",
	SkillCRSynthesispotion:                  "CR_SYNTHESISPOTION",
	SkillCGArrowvulcan:                      "CG_ARROWVULCAN",
	SkillCGMoonlit:                          "CG_MOONLIT",
	SkillCGMarionette:                       "CG_MARIONETTE",
	SkillLKSpiralpierce:                     "LK_SPIRALPIERCE",
	SkillLKHeadcrush:                        "LK_HEADCRUSH",
	SkillLKJointbeat:                        "LK_JOINTBEAT",
	SkillHWNapalmvulcan:                     "HW_NAPALMVULCAN",
	SkillChSoulcollect:                      "CH_SOULCOLLECT",
	SkillPFMindbreaker:                      "PF_MINDBREAKER",
	SkillPFMemorize:                         "PF_MEMORIZE",
	SkillPFFogwall:                          "PF_FOGWALL",
	SkillPFSpiderweb:                        "PF_SPIDERWEB",
	SkillASCMeteorassault:                   "ASC_METEORASSAULT",
	SkillASCCdp:                             "ASC_CDP",
	SkillWEBaby:                             "WE_BABY",
	SkillWECallparent:                       "WE_CALLPARENT",
	SkillWECallbaby:                         "WE_CALLBABY",
	SkillTKRun:                              "TK_RUN",
	SkillTKReadystorm:                       "TK_READYSTORM",
	SkillTKStormkick:                        "TK_STORMKICK",
	SkillTKReadydown:                        "TK_READYDOWN",
	SkillTKDownkick:                         "TK_DOWNKICK",
	SkillTKReadyturn:                        "TK_READYTURN",
	SkillTKTurnkick:                         "TK_TURNKICK",
	SkillTKReadycounter:                     "TK_READYCOUNTER",
	SkillTKCounter:                          "TK_COUNTER",
	SkillTKDodge:                            "TK_DODGE",
	SkillTKJumpkick:                         "TK_JUMPKICK",
	SkillTKHptime:                           "TK_HPTIME",
	SkillTKSptime:                           "TK_SPTIME",
	SkillTKPower:                            "TK_POWER",
	SkillTKSevenwind:                        "TK_SEVENWIND",
	SkillTKHighjump:                         "TK_HIGHJUMP",
	SkillSGFeel:                             "SG_FEEL",
	SkillSGSunWarm:                          "SG_SUN_WARM",
	SkillSGMoonWarm:                         "SG_MOON_WARM",
	SkillSGStarWarm:                         "SG_STAR_WARM",
	SkillSGSunComfort:                       "SG_SUN_COMFORT",
	SkillSGMoonComfort:                      "SG_MOON_COMFORT",
	SkillSGStarComfort:                      "SG_STAR_COMFORT",
	SkillSGHate:                             "SG_HATE",
	SkillSGSunAnger:                         "SG_SUN_ANGER",
	SkillSGMoonAnger:                        "SG_MOON_ANGER",
	SkillSGStarAnger:                        "SG_STAR_ANGER",
	SkillSGSunBless:                         "SG_SUN_BLESS",
	SkillSGMoonBless:                        "SG_MOON_BLESS",
	SkillSGStarBless:                        "SG_STAR_BLESS",
	SkillSGDevil:                            "SG_DEVIL",
	SkillSGFriend:                           "SG_FRIEND",
	SkillSGKnowledge:                        "SG_KNOWLEDGE",
	SkillSGFusion:                           "SG_FUSION",
	SkillSLAlchemist:                        "SL_ALCHEMIST",
	SkillAMBerserkpitcher:                   "AM_BERSERKPITCHER",
	SkillSLMonk:                             "SL_MONK",
	SkillSLStar:                             "SL_STAR",
	SkillSLSage:                             "SL_SAGE",
	SkillSLCrusader:                         "SL_CRUSADER",
	SkillSLSupernovice:                      "SL_SUPERNOVICE",
	SkillSLKnight:                           "SL_KNIGHT",
	SkillSLWizard:                           "SL_WIZARD",
	SkillSLPriest:                           "SL_PRIEST",
	SkillSLBarddancer:                       "SL_BARDDANCER",
	SkillSLRogue:                            "SL_ROGUE",
	SkillSLAssasin:                          "SL_ASSASIN",
	SkillSLBlacksmith:                       "SL_BLACKSMITH",
	SkillBSAdrenaline2:                      "BS_ADRENALINE2",
	SkillSLHunter:                           "SL_HUNTER",
	SkillSLSoullinker:                       "SL_SOULLINKER",
	SkillSLKaizel:                           "SL_KAIZEL",
	SkillSLKaahi:                            "SL_KAAHI",
	SkillSLKaupe:                            "SL_KAUPE",
	SkillSLKaite:                            "SL_KAITE",
	SkillSLKaina:                            "SL_KAINA",
	SkillSLStin:                             "SL_STIN",
	SkillSLStun:                             "SL_STUN",
	SkillSLSma:                              "SL_SMA",
	SkillSLSwoo:                             "SL_SWOO",
	SkillSLSke:                              "SL_SKE",
	SkillSLSka:                              "SL_SKA",
	SkillSMSelfprovoke:                      "SM_SELFPROVOKE",
	SkillNPCEmotionOn:                       "NPC_EMOTION_ON",
	SkillSTPreserve:                         "ST_PRESERVE",
	SkillSTFullstrip:                        "ST_FULLSTRIP",
	SkillWSWeaponrefine:                     "WS_WEAPONREFINE",
	SkillCRSlimpitcher:                      "CR_SLIMPITCHER",
	SkillCRFullprotection:                   "CR_FULLPROTECTION",
	SkillPaShieldchain:                      "PA_SHIELDCHAIN",
	SkillHPManarecharge:                     "HP_MANARECHARGE",
	SkillPFDoublecasting:                    "PF_DOUBLECASTING",
	SkillHWGanbantein:                       "HW_GANBANTEIN",
	SkillHWGravitation:                      "HW_GRAVITATION",
	SkillWSCarttermination:                  "WS_CARTTERMINATION",
	SkillWSOverthrustmax:                    "WS_OVERTHRUSTMAX",
	SkillCGLongingfreedom:                   "CG_LONGINGFREEDOM",
	SkillCGHermode:                          "CG_HERMODE",
	SkillCGTarotcard:                        "CG_TAROTCARD",
	SkillCRAciddemonstration:                "CR_ACIDDEMONSTRATION",
	SkillCRCultivation:                      "CR_CULTIVATION",
	SkillItemEnchantarms:                    "ITEM_ENCHANTARMS",
	SkillTKMission:                          "TK_MISSION",
	SkillSLHigh:                             "SL_HIGH",
	SkillKNOnehand:                          "KN_ONEHAND",
	SkillAMTwilight1:                        "AM_TWILIGHT1",
	SkillAMTwilight2:                        "AM_TWILIGHT2",
	SkillAMTwilight3:                        "AM_TWILIGHT3",
	SkillHTPower:                            "HT_POWER",
	SkillGSGlittering:                       "GS_GLITTERING",
	SkillGSFling:                            "GS_FLING",
	SkillGSTripleaction:                     "GS_TRIPLEACTION",
	SkillGSBullseye:                         "GS_BULLSEYE",
	SkillGSMadnesscancel:                    "GS_MADNESSCANCEL",
	SkillGSAdjustment:                       "GS_ADJUSTMENT",
	SkillGSIncreasing:                       "GS_INCREASING",
	SkillGSMagicalbullet:                    "GS_MAGICALBULLET",
	SkillGSCracker:                          "GS_CRACKER",
	SkillGSSingleaction:                     "GS_SINGLEACTION",
	SkillGSSnakeeye:                         "GS_SNAKEEYE",
	SkillGSChainaction:                      "GS_CHAINACTION",
	SkillGSTracking:                         "GS_TRACKING",
	SkillGSDisarm:                           "GS_DISARM",
	SkillGSPiercingshot:                     "GS_PIERCINGSHOT",
	SkillGSRapidshower:                      "GS_RAPIDSHOWER",
	SkillGSDesperado:                        "GS_DESPERADO",
	SkillGSGatlingfever:                     "GS_GATLINGFEVER",
	SkillGSDust:                             "GS_DUST",
	SkillGSFullbuster:                       "GS_FULLBUSTER",
	SkillGSSpreadattack:                     "GS_SPREADATTACK",
	SkillGSGrounddrift:                      "GS_GROUNDDRIFT",
	SkillNJTobidougu:                        "NJ_TOBIDOUGU",
	SkillNJSyuriken:                         "NJ_SYURIKEN",
	SkillNJKunai:                            "NJ_KUNAI",
	SkillNJHuuma:                            "NJ_HUUMA",
	SkillNJZenynage:                         "NJ_ZENYNAGE",
	SkillNJTatamigaeshi:                     "NJ_TATAMIGAESHI",
	SkillNJKasumikiri:                       "NJ_KASUMIKIRI",
	SkillNJShadowjump:                       "NJ_SHADOWJUMP",
	SkillNJKirikage:                         "NJ_KIRIKAGE",
	SkillNJUtsusemi:                         "NJ_UTSUSEMI",
	SkillNJBunsinjyutsu:                     "NJ_BUNSINJYUTSU",
	SkillNJNinpou:                           "NJ_NINPOU",
	SkillNJKouenka:                          "NJ_KOUENKA",
	SkillNJKaensin:                          "NJ_KAENSIN",
	SkillNJBakuenryu:                        "NJ_BAKUENRYU",
	SkillNJHyousensou:                       "NJ_HYOUSENSOU",
	SkillNJSuiton:                           "NJ_SUITON",
	SkillNJHyousyouraku:                     "NJ_HYOUSYOURAKU",
	SkillNJHuujin:                           "NJ_HUUJIN",
	SkillNJRaigekisai:                       "NJ_RAIGEKISAI",
	SkillNJKamaitachi:                       "NJ_KAMAITACHI",
	SkillNJNen:                              "NJ_NEN",
	SkillNJIssen:                            "NJ_ISSEN",
	SkillMbFighting:                         "MB_FIGHTING",
	SkillMbNeutral:                          "MB_NEUTRAL",
	SkillMbTaimingPuti:                      "MB_TAIMING_PUTI",
	SkillMbWhitepotion:                      "MB_WHITEPOTION",
	SkillMbMental:                           "MB_MENTAL",
	SkillMbCardpitcher:                      "MB_CARDPITCHER",
	SkillMbPetpitcher:                       "MB_PETPITCHER",
	SkillMbBodystudy:                        "MB_BODYSTUDY",
	SkillMbBodyalter:                        "MB_BODYALTER",
	SkillMbPetmemory:                        "MB_PETMEMORY",
	SkillMbMTeleport:                        "MB_M_TELEPORT",
	SkillMbBGain:                            "MB_B_GAIN",
	SkillMbMGain:                            "MB_M_GAIN",
	SkillMbMission:                          "MB_MISSION",
	SkillMbMunakknowledge:                   "MB_MUNAKKNOWLEDGE",
	SkillMbMunakball:                        "MB_MUNAKBALL",
	SkillMbScroll:                           "MB_SCROLL",
	SkillMbBGathering:                       "MB_B_GATHERING",
	SkillMbMGathering:                       "MB_M_GATHERING",
	SkillMbBExclude:                         "MB_B_EXCLUDE",
	SkillMbBDrift:                           "MB_B_DRIFT",
	SkillMbBWallrush:                        "MB_B_WALLRUSH",
	SkillMbMWallrush:                        "MB_M_WALLRUSH",
	SkillMbBWallshift:                       "MB_B_WALLSHIFT",
	SkillMbMWallcrash:                       "MB_M_WALLCRASH",
	SkillMbMReincarnation:                   "MB_M_REINCARNATION",
	SkillMbBEquip:                           "MB_B_EQUIP",
	SkillSLDeathknight:                      "SL_DEATHKNIGHT",
	SkillSLCollector:                        "SL_COLLECTOR",
	SkillSLNinja:                            "SL_NINJA",
	SkillSLGunner:                           "SL_GUNNER",
	SkillAMTwilight4:                        "AM_TWILIGHT4",
	SkillDaReset:                            "DA_RESET",
	SkillDeBerserkaizer:                     "DE_BERSERKAIZER",
	SkillDaDarkpower:                        "DA_DARKPOWER",
	SkillDePassive:                          "DE_PASSIVE",
	SkillDePattack:                          "DE_PATTACK",
	SkillDePspeed:                           "DE_PSPEED",
	SkillDePdefense:                         "DE_PDEFENSE",
	SkillDePcritical:                        "DE_PCRITICAL",
	SkillDePhp:                              "DE_PHP",
	SkillDePsp:                              "DE_PSP",
	SkillDeReset:                            "DE_RESET",
	SkillDeRanking:                          "DE_RANKING",
	SkillDePtriple:                          "DE_PTRIPLE",
	SkillDeEnergy:                           "DE_ENERGY",
	SkillDeNightmare:                        "DE_NIGHTMARE",
	SkillDeSlash:                            "DE_SLASH",
	SkillDeCoil:                             "DE_COIL",
	SkillDeWave:                             "DE_WAVE",
	SkillDeRebirth:                          "DE_REBIRTH",
	SkillDeAura:                             "DE_AURA",
	SkillDeFreezer:                          "DE_FREEZER",
	SkillDeChangeattack:                     "DE_CHANGEATTACK",
	SkillDePunish:                           "DE_PUNISH",
	SkillDePoison:                           "DE_POISON",
	SkillDeInstant:                          "DE_INSTANT",
	SkillDeWarning:                          "DE_WARNING",
	SkillDeRankedknife:                      "DE_RANKEDKNIFE",
	SkillDeRankedgradius:                    "DE_RANKEDGRADIUS",
	SkillDeGauge:                            "DE_GAUGE",
	SkillDeGtime:                            "DE_GTIME",
	SkillDeGpain:                            "DE_GPAIN",
	SkillDeGskill:                           "DE_GSKILL",
	SkillDeGkill:                            "DE_GKILL",
	SkillDeAccel:                            "DE_ACCEL",
	SkillDeBlockdouble:                      "DE_BLOCKDOUBLE",
	SkillDeBlockmelee:                       "DE_BLOCKMELEE",
	SkillDeBlockfar:                         "DE_BLOCKFAR",
	SkillDeFrontattack:                      "DE_FRONTATTACK",
	SkillDeDangerattack:                     "DE_DANGERATTACK",
	SkillDeTwinattack:                       "DE_TWINATTACK",
	SkillDeWindattack:                       "DE_WINDATTACK",
	SkillDeWaterattack:                      "DE_WATERATTACK",
	SkillDaEnergy:                           "DA_ENERGY",
	SkillDaCloud:                            "DA_CLOUD",
	SkillDaFirstslot:                        "DA_FIRSTSLOT",
	SkillDaHeaddef:                          "DA_HEADDEF",
	SkillDaSpace:                            "DA_SPACE",
	SkillDaTransform:                        "DA_TRANSFORM",
	SkillDaExplosion:                        "DA_EXPLOSION",
	SkillDaReward:                           "DA_REWARD",
	SkillDaCrush:                            "DA_CRUSH",
	SkillDaItemrebuild:                      "DA_ITEMREBUILD",
	SkillDaIllusion:                         "DA_ILLUSION",
	SkillDaNuetralize:                       "DA_NUETRALIZE",
	SkillDaRunner:                           "DA_RUNNER",
	SkillDaTransfer:                         "DA_TRANSFER",
	SkillDaWall:                             "DA_WALL",
	SkillDaZeny:                             "DA_ZENY",
	SkillDaRevenge:                          "DA_REVENGE",
	SkillDaEarplug:                          "DA_EARPLUG",
	SkillDaContract:                         "DA_CONTRACT",
	SkillDaBlack:                            "DA_BLACK",
	SkillDaDream:                            "DA_DREAM",
	SkillDaMagiccart:                        "DA_MAGICCART",
	SkillDaCopy:                             "DA_COPY",
	SkillDaCrystal:                          "DA_CRYSTAL",
	SkillDaExp:                              "DA_EXP",
	SkillDaCartswing:                        "DA_CARTSWING",
	SkillDaRebuild:                          "DA_REBUILD",
	SkillDaJobchange:                        "DA_JOBCHANGE",
	SkillDaEdarkness:                        "DA_EDARKNESS",
	SkillDaEguardian:                        "DA_EGUARDIAN",
	SkillDaTimeout:                          "DA_TIMEOUT",
	SkillALLTimein:                          "ALL_TIMEIN",
	SkillDaZenyrank:                         "DA_ZENYRANK",
	SkillDaAccessorymix:                     "DA_ACCESSORYMIX",
	SkillNPCEarthquake:                      "NPC_EARTHQUAKE",
	SkillNPCFirebreath:                      "NPC_FIREBREATH",
	SkillNPCIcebreath:                       "NPC_ICEBREATH",
	SkillNPCThunderbreath:                   "NPC_THUNDERBREATH",
	SkillNPCAcidbreath:                      "NPC_ACIDBREATH",
	SkillNPCDarknessbreath:                  "NPC_DARKNESSBREATH",
	SkillNPCDragonfear:                      "NPC_DRAGONFEAR",
	SkillNPCBleeding:                        "NPC_BLEEDING",
	SkillNPCPulsestrike:                     "NPC_PULSESTRIKE",
	SkillNPCHelljudgement:                   "NPC_HELLJUDGEMENT",
	SkillNPCWidesilence:                     "NPC_WIDESILENCE",
	SkillNPCWidefreeze:                      "NPC_WIDEFREEZE",
	SkillNPCWidebleeding:                    "NPC_WIDEBLEEDING",
	SkillNPCWidestone:                       "NPC_WIDESTONE",
	SkillNPCWideconfuse:                     "NPC_WIDECONFUSE",
	SkillNPCWidesleep:                       "NPC_WIDESLEEP",
	SkillNPCWidesight:                       "NPC_WIDESIGHT",
	SkillNPCEvilland:                        "NPC_EVILLAND",
	SkillNPCMagicmirror:                     "NPC_MAGICMIRROR",
	SkillNPCSlowcast:                        "NPC_SLOWCAST",
	SkillNPCCriticalwound:                   "NPC_CRITICALWOUND",
	SkillNPCExpulsion:                       "NPC_EXPULSION",
	SkillNPCStoneskin:                       "NPC_STONESKIN",
	SkillNPCAntimagic:                       "NPC_ANTIMAGIC",
	SkillNPCWidecurse:                       "NPC_WIDECURSE",
	SkillNPCWidestun:                        "NPC_WIDESTUN",
	SkillNPCVampireGift:                     "NPC_VAMPIRE_GIFT",
	SkillNPCWidesouldrain:                   "NPC_WIDESOULDRAIN",
	SkillALLInccarry:                        "ALL_INCCARRY",
	SkillNPCTalk:                            "NPC_TALK",
	SkillNPCHellpower:                       "NPC_HELLPOWER",
	SkillNPCWidehelldignity:                 "NPC_WIDEHELLDIGNITY",
	SkillNPCInvincible:                      "NPC_INVINCIBLE",
	SkillNPCInvincibleoff:                   "NPC_INVINCIBLEOFF",
	SkillNPCAllheal:                         "NPC_ALLHEAL",
	SkillGmSandman:                          "GM_SANDMAN",
	SkillCashBlessing:                       "CASH_BLESSING",
	SkillCashIncagi:                         "CASH_INCAGI",
	SkillCashAssumptio:                      "CASH_ASSUMPTIO",
	SkillALLCatcry:                          "ALL_CATCRY",
	SkillALLPartyflee:                       "ALL_PARTYFLEE",
	SkillALLAngelProtect:                    "ALL_ANGEL_PROTECT",
	SkillALLDreamSummernight:                "ALL_DREAM_SUMMERNIGHT",
	SkillNPCChangeundead2:                   "NPC_CHANGEUNDEAD2",
	SkillALLReverseorcish:                   "ALL_REVERSEORCISH",
	SkillALLWewish:                          "ALL_WEWISH",
	SkillALLSonkran:                         "ALL_SONKRAN",
	SkillNPCWidehealthfear:                  "NPC_WIDEHEALTHFEAR",
	SkillNPCWidebodyburnning:                "NPC_WIDEBODYBURNNING",
	SkillNPCWidefrostmisty:                  "NPC_WIDEFROSTMISTY",
	SkillNPCWidecold:                        "NPC_WIDECOLD",
	SkillNPCWideDeepSleep:                   "NPC_WIDE_DEEP_SLEEP",
	SkillNPCWidesiren:                       "NPC_WIDESIREN",
	SkillNPCVenomfog:                        "NPC_VENOMFOG",
	SkillNPCMillenniumshield:                "NPC_MILLENNIUMSHIELD",
	SkillNPCComet:                           "NPC_COMET",
	SkillNPCIcemine:                         "NPC_ICEMINE",
	SkillNPCIceexplo:                        "NPC_ICEEXPLO",
	SkillNPCFlamecross:                      "NPC_FLAMECROSS",
	SkillNPCPulsestrike2:                    "NPC_PULSESTRIKE2",
	SkillNPCDancingblade:                    "NPC_DANCINGBLADE",
	SkillNPCDancingbladeAtk:                 "NPC_DANCINGBLADE_ATK",
	SkillNPCDarkpiercing:                    "NPC_DARKPIERCING",
	SkillNPCMaxpain:                         "NPC_MAXPAIN",
	SkillNPCMaxpainAtk:                      "NPC_MAXPAIN_ATK",
	SkillNPCDeathsummon:                     "NPC_DEATHSUMMON",
	SkillNPCHellburning:                     "NPC_HELLBURNING",
	SkillNPCJackfrost:                       "NPC_JACKFROST",
	SkillNPCWideweb:                         "NPC_WIDEWEB",
	SkillNPCWidesuck:                        "NPC_WIDESUCK",
	SkillNPCStormgust2:                      "NPC_STORMGUST2",
	SkillNPCFirestorm:                       "NPC_FIRESTORM",
	SkillNPCReverberation:                   "NPC_REVERBERATION",
	SkillNPCReverberationAtk:                "NPC_REVERBERATION_ATK",
	SkillNPCLexAeterna:                      "NPC_LEX_AETERNA",
	SkillNPCArrowstorm:                      "NPC_ARROWSTORM",
	SkillNPCCheal:                           "NPC_CHEAL",
	SkillNPCSRCursedcircle:                  "NPC_SR_CURSEDCIRCLE",
	SkillNPCDragonbreath:                    "NPC_DRAGONBREATH",
	SkillNPCFatalmenace:                     "NPC_FATALMENACE",
	SkillNPCMagmaEruption:                   "NPC_MAGMA_ERUPTION",
	SkillNPCMagmaEruptionDotdamage:          "NPC_MAGMA_ERUPTION_DOTDAMAGE",
	SkillNPCMandragora:                      "NPC_MANDRAGORA",
	SkillNPCPsychicWave:                     "NPC_PSYCHIC_WAVE",
	SkillNPCRayofgenesis:                    "NPC_RAYOFGENESIS",
	SkillNPCVenomimpress:                    "NPC_VENOMIMPRESS",
	SkillNPCCloudKill:                       "NPC_CLOUD_KILL",
	SkillNPCIgnitionbreak:                   "NPC_IGNITIONBREAK",
	SkillNPCPhantomthrust:                   "NPC_PHANTOMTHRUST",
	SkillNPCPoisonBuster:                    "NPC_POISON_BUSTER",
	SkillNPCHallucinationwalk:               "NPC_HALLUCINATIONWALK",
	SkillNPCElectricwalk:                    "NPC_ELECTRICWALK",
	SkillNPCFirewalk:                        "NPC_FIREWALK",
	SkillNPCWidedispel:                      "NPC_WIDEDISPEL",
	SkillNPCLeash:                           "NPC_LEASH",
	SkillNPCWideleash:                       "NPC_WIDELEASH",
	SkillNPCWidecriticalwound:               "NPC_WIDECRITICALWOUND",
	SkillNPCEarthquakeK:                     "NPC_EARTHQUAKE_K",
	SkillNPCALLStatDown:                     "NPC_ALL_STAT_DOWN",
	SkillNPCGradualGravity:                  "NPC_GRADUAL_GRAVITY",
	SkillNPCDamageHeal:                      "NPC_DAMAGE_HEAL",
	SkillNPCImmuneProperty:                  "NPC_IMMUNE_PROPERTY",
	SkillNPCMoveCoordinate:                  "NPC_MOVE_COORDINATE",
	SkillNPCWidebleeding2:                   "NPC_WIDEBLEEDING2",
	SkillNPCWidesilence2:                    "NPC_WIDESILENCE2",
	SkillNPCWidestun2:                       "NPC_WIDESTUN2",
	SkillNPCWidestone2:                      "NPC_WIDESTONE2",
	SkillNPCWidesleep2:                      "NPC_WIDESLEEP2",
	SkillNPCWidecurse2:                      "NPC_WIDECURSE2",
	SkillNPCWideconfuse2:                    "NPC_WIDECONFUSE2",
	SkillNPCWidefreeze2:                     "NPC_WIDEFREEZE2",
	SkillNPCBleeding2:                       "NPC_BLEEDING2",
	SkillNPCIcebreath2:                      "NPC_ICEBREATH2",
	SkillNPCAcidbreath2:                     "NPC_ACIDBREATH2",
	SkillNPCEvilland2:                       "NPC_EVILLAND2",
	SkillNPCHelljudgement2:                  "NPC_HELLJUDGEMENT2",
	SkillNPCRainofmeteor:                    "NPC_RAINOFMETEOR",
	SkillNPCGrounddrive:                     "NPC_GROUNDDRIVE",
	SkillNPCRelieveOn:                       "NPC_RELIEVE_ON",
	SkillNPCRelieveOff:                      "NPC_RELIEVE_OFF",
	SkillNPCLockonLaser:                     "NPC_LOCKON_LASER",
	SkillNPCLockonLaserAtk:                  "NPC_LOCKON_LASER_ATK",
	SkillNPCSeedtrap:                        "NPC_SEEDTRAP",
	SkillNPCDeadlycurse:                     "NPC_DEADLYCURSE",
	SkillNPCRandombreak:                     "NPC_RANDOMBREAK",
	SkillNPCStripShadow:                     "NPC_STRIP_SHADOW",
	SkillNPCDeadlycurse2:                    "NPC_DEADLYCURSE2",
	SkillNPCCaneOfEvilEye:                   "NPC_CANE_OF_EVIL_EYE",
	SkillNPCCurseOfRedCube:                  "NPC_CURSE_OF_RED_CUBE",
	SkillNPCCurseOfBlueCube:                 "NPC_CURSE_OF_BLUE_CUBE",
	SkillNPCKillingAura:                     "NPC_KILLING_AURA",
	SkillNPCLast:                            "NPC_LAST",
	SkillKNChargeatk:                        "KN_CHARGEATK",
	SkillCRShrink:                           "CR_SHRINK",
	SkillASSonicaccel:                       "AS_SONICACCEL",
	SkillASVenomknife:                       "AS_VENOMKNIFE",
	SkillRGCloseconfine:                     "RG_CLOSECONFINE",
	SkillWZSightblaster:                     "WZ_SIGHTBLASTER",
	SkillSACreatecon:                        "SA_CREATECON",
	SkillSAElementwater:                     "SA_ELEMENTWATER",
	SkillHTPhantasmic:                       "HT_PHANTASMIC",
	SkillBaPangvoice:                        "BA_PANGVOICE",
	SkillDCWinkcharm:                        "DC_WINKCHARM",
	SkillBSUnfairlytrick:                    "BS_UNFAIRLYTRICK",
	SkillBSGreed:                            "BS_GREED",
	SkillPRRedemptio:                        "PR_REDEMPTIO",
	SkillMOKitranslation:                    "MO_KITRANSLATION",
	SkillMOBalkyoung:                        "MO_BALKYOUNG",
	SkillSAElementground:                    "SA_ELEMENTGROUND",
	SkillSAElementfire:                      "SA_ELEMENTFIRE",
	SkillSAElementwind:                      "SA_ELEMENTWIND",
	SkillThirdjobBegin:                      "THIRDJOB_BEGIN",
	SkillRKEnchantblade:                     "RK_ENCHANTBLADE",
	SkillRKSonicwave:                        "RK_SONICWAVE",
	SkillRKDeathbound:                       "RK_DEATHBOUND",
	SkillRKHundredspear:                     "RK_HUNDREDSPEAR",
	SkillRKWindcutter:                       "RK_WINDCUTTER",
	SkillRKIgnitionbreak:                    "RK_IGNITIONBREAK",
	SkillRKDragontraining:                   "RK_DRAGONTRAINING",
	SkillRKDragonbreath:                     "RK_DRAGONBREATH",
	SkillRKDragonhowling:                    "RK_DRAGONHOWLING",
	SkillRKRunemastery:                      "RK_RUNEMASTERY",
	SkillRKMillenniumshield:                 "RK_MILLENNIUMSHIELD",
	SkillRKCrushstrike:                      "RK_CRUSHSTRIKE",
	SkillRKRefresh:                          "RK_REFRESH",
	SkillRKGiantgrowth:                      "RK_GIANTGROWTH",
	SkillRKStonehardskin:                    "RK_STONEHARDSKIN",
	SkillRKVitalityactivation:               "RK_VITALITYACTIVATION",
	SkillRKStormblast:                       "RK_STORMBLAST",
	SkillRKFightingspirit:                   "RK_FIGHTINGSPIRIT",
	SkillRKAbundance:                        "RK_ABUNDANCE",
	SkillRKPhantomthrust:                    "RK_PHANTOMTHRUST",
	SkillGCVenomimpress:                     "GC_VENOMIMPRESS",
	SkillGCCrossimpact:                      "GC_CROSSIMPACT",
	SkillGCDarkillusion:                     "GC_DARKILLUSION",
	SkillGCResearchnewpoison:                "GC_RESEARCHNEWPOISON",
	SkillGCCreatenewpoison:                  "GC_CREATENEWPOISON",
	SkillGCAntidote:                         "GC_ANTIDOTE",
	SkillGCPoisoningweapon:                  "GC_POISONINGWEAPON",
	SkillGCWeaponblocking:                   "GC_WEAPONBLOCKING",
	SkillGCCounterslash:                     "GC_COUNTERSLASH",
	SkillGCWeaponcrush:                      "GC_WEAPONCRUSH",
	SkillGCVenompressure:                    "GC_VENOMPRESSURE",
	SkillGCPoisonsmoke:                      "GC_POISONSMOKE",
	SkillGCCloakingexceed:                   "GC_CLOAKINGEXCEED",
	SkillGCPhantommenace:                    "GC_PHANTOMMENACE",
	SkillGCHallucinationwalk:                "GC_HALLUCINATIONWALK",
	SkillGCRollingcutter:                    "GC_ROLLINGCUTTER",
	SkillGCCrossripperslasher:               "GC_CROSSRIPPERSLASHER",
	SkillABJudex:                            "AB_JUDEX",
	SkillABAncilla:                          "AB_ANCILLA",
	SkillABAdoramus:                         "AB_ADORAMUS",
	SkillABClementia:                        "AB_CLEMENTIA",
	SkillABCanto:                            "AB_CANTO",
	SkillABCheal:                            "AB_CHEAL",
	SkillABEpiclesis:                        "AB_EPICLESIS",
	SkillABPraefatio:                        "AB_PRAEFATIO",
	SkillABOratio:                           "AB_ORATIO",
	SkillABLaudaagnus:                       "AB_LAUDAAGNUS",
	SkillABLaudaramus:                       "AB_LAUDARAMUS",
	SkillABEucharistica:                     "AB_EUCHARISTICA",
	SkillABRenovatio:                        "AB_RENOVATIO",
	SkillABHighnessheal:                     "AB_HIGHNESSHEAL",
	SkillABClearance:                        "AB_CLEARANCE",
	SkillABExpiatio:                         "AB_EXPIATIO",
	SkillABDuplelight:                       "AB_DUPLELIGHT",
	SkillABDuplelightMelee:                  "AB_DUPLELIGHT_MELEE",
	SkillABDuplelightMagic:                  "AB_DUPLELIGHT_MAGIC",
	SkillABSilentium:                        "AB_SILENTIUM",
	SkillWLStartmark:                        "WL_STARTMARK",
	SkillWLWhiteimprison:                    "WL_WHITEIMPRISON",
	SkillWLSoulexpansion:                    "WL_SOULEXPANSION",
	SkillWLFrostmisty:                       "WL_FROSTMISTY",
	SkillWLJackfrost:                        "WL_JACKFROST",
	SkillWLMarshofabyss:                     "WL_MARSHOFABYSS",
	SkillWLRecognizedspell:                  "WL_RECOGNIZEDSPELL",
	SkillWLSiennaexecrate:                   "WL_SIENNAEXECRATE",
	SkillWLRadius:                           "WL_RADIUS",
	SkillWLStasis:                           "WL_STASIS",
	SkillWLDrainlife:                        "WL_DRAINLIFE",
	SkillWLCrimsonrock:                      "WL_CRIMSONROCK",
	SkillWLHellinferno:                      "WL_HELLINFERNO",
	SkillWLComet:                            "WL_COMET",
	SkillWLChainlightning:                   "WL_CHAINLIGHTNING",
	SkillWLChainlightningAtk:                "WL_CHAINLIGHTNING_ATK",
	SkillWLEarthstrain:                      "WL_EARTHSTRAIN",
	SkillWLTetravortex:                      "WL_TETRAVORTEX",
	SkillWLTetravortexFire:                  "WL_TETRAVORTEX_FIRE",
	SkillWLTetravortexWater:                 "WL_TETRAVORTEX_WATER",
	SkillWLTetravortexWind:                  "WL_TETRAVORTEX_WIND",
	SkillWLTetravortexGround:                "WL_TETRAVORTEX_GROUND",
	SkillWLSummonfb:                         "WL_SUMMONFB",
	SkillWLSummonbl:                         "WL_SUMMONBL",
	SkillWLSummonwb:                         "WL_SUMMONWB",
	SkillWLSummonAtkFire:                    "WL_SUMMON_ATK_FIRE",
	SkillWLSummonAtkWind:                    "WL_SUMMON_ATK_WIND",
	SkillWLSummonAtkWater:                   "WL_SUMMON_ATK_WATER",
	SkillWLSummonAtkGround:                  "WL_SUMMON_ATK_GROUND",
	SkillWLSummonstone:                      "WL_SUMMONSTONE",
	SkillWLRelease:                          "WL_RELEASE",
	SkillWLReadingSb:                        "WL_READING_SB",
	SkillWLFreezeSP:                         "WL_FREEZE_SP",
	SkillRAArrowstorm:                       "RA_ARROWSTORM",
	SkillRAFearbreeze:                       "RA_FEARBREEZE",
	SkillRARangermain:                       "RA_RANGERMAIN",
	SkillRAAimedbolt:                        "RA_AIMEDBOLT",
	SkillRADetonator:                        "RA_DETONATOR",
	SkillRAElectricshocker:                  "RA_ELECTRICSHOCKER",
	SkillRAClusterbomb:                      "RA_CLUSTERBOMB",
	SkillRAWugmastery:                       "RA_WUGMASTERY",
	SkillRAWugrider:                         "RA_WUGRIDER",
	SkillRAWugdash:                          "RA_WUGDASH",
	SkillRAWugstrike:                        "RA_WUGSTRIKE",
	SkillRAWugbite:                          "RA_WUGBITE",
	SkillRAToothofwug:                       "RA_TOOTHOFWUG",
	SkillRASensitivekeen:                    "RA_SENSITIVEKEEN",
	SkillRACamouflage:                       "RA_CAMOUFLAGE",
	SkillRAResearchtrap:                     "RA_RESEARCHTRAP",
	SkillRAMagentatrap:                      "RA_MAGENTATRAP",
	SkillRACobalttrap:                       "RA_COBALTTRAP",
	SkillRAMaizetrap:                        "RA_MAIZETRAP",
	SkillRAVerduretrap:                      "RA_VERDURETRAP",
	SkillRAFiringtrap:                       "RA_FIRINGTRAP",
	SkillRAIceboundtrap:                     "RA_ICEBOUNDTRAP",
	SkillNCMadolicence:                      "NC_MADOLICENCE",
	SkillNCBoostknuckle:                     "NC_BOOSTKNUCKLE",
	SkillNCPilebunker:                       "NC_PILEBUNKER",
	SkillNCVulcanarm:                        "NC_VULCANARM",
	SkillNCFlamelauncher:                    "NC_FLAMELAUNCHER",
	SkillNCColdslower:                       "NC_COLDSLOWER",
	SkillNCArmscannon:                       "NC_ARMSCANNON",
	SkillNCAcceleration:                     "NC_ACCELERATION",
	SkillNCHovering:                         "NC_HOVERING",
	SkillNCFSideslide:                       "NC_F_SIDESLIDE",
	SkillNCBSideslide:                       "NC_B_SIDESLIDE",
	SkillNCMainframe:                        "NC_MAINFRAME",
	SkillNCSelfdestruction:                  "NC_SELFDESTRUCTION",
	SkillNCShapeshift:                       "NC_SHAPESHIFT",
	SkillNCEmergencycool:                    "NC_EMERGENCYCOOL",
	SkillNCInfraredscan:                     "NC_INFRAREDSCAN",
	SkillNCAnalyze:                          "NC_ANALYZE",
	SkillNCMagneticfield:                    "NC_MAGNETICFIELD",
	SkillNCNeutralbarrier:                   "NC_NEUTRALBARRIER",
	SkillNCStealthfield:                     "NC_STEALTHFIELD",
	SkillNCRepair:                           "NC_REPAIR",
	SkillNCTrainingaxe:                      "NC_TRAININGAXE",
	SkillNCResearchfe:                       "NC_RESEARCHFE",
	SkillNCAxeboomerang:                     "NC_AXEBOOMERANG",
	SkillNCPowerswing:                       "NC_POWERSWING",
	SkillNCAxetornado:                       "NC_AXETORNADO",
	SkillNCSilversniper:                     "NC_SILVERSNIPER",
	SkillNCMagicdecoy:                       "NC_MAGICDECOY",
	SkillNCDisjoint:                         "NC_DISJOINT",
	SkillSCStartmark:                        "SC_STARTMARK",
	SkillSCReproduce:                        "SC_REPRODUCE",
	SkillSCAutoshadowspell:                  "SC_AUTOSHADOWSPELL",
	SkillSCShadowform:                       "SC_SHADOWFORM",
	SkillSCTriangleshot:                     "SC_TRIANGLESHOT",
	SkillSCBodypaint:                        "SC_BODYPAINT",
	SkillSCInvisibility:                     "SC_INVISIBILITY",
	SkillSCDeadlyinfect:                     "SC_DEADLYINFECT",
	SkillSCEnervation:                       "SC_ENERVATION",
	SkillSCGroomy:                           "SC_GROOMY",
	SkillSCIgnorance:                        "SC_IGNORANCE",
	SkillSCLaziness:                         "SC_LAZINESS",
	SkillSCUnlucky:                          "SC_UNLUCKY",
	SkillSCWeakness:                         "SC_WEAKNESS",
	SkillSCStripaccessary:                   "SC_STRIPACCESSARY",
	SkillSCManhole:                          "SC_MANHOLE",
	SkillSCDimensiondoor:                    "SC_DIMENSIONDOOR",
	SkillSCChaospanic:                       "SC_CHAOSPANIC",
	SkillSCMaelstrom:                        "SC_MAELSTROM",
	SkillSCBloodylust:                       "SC_BLOODYLUST",
	SkillSCFeintbomb:                        "SC_FEINTBOMB",
	SkillSCEndmark:                          "SC_ENDMARK",
	SkillLGCannonspear:                      "LG_CANNONSPEAR",
	SkillLGBanishingpoint:                   "LG_BANISHINGPOINT",
	SkillLGTrample:                          "LG_TRAMPLE",
	SkillLGShieldpress:                      "LG_SHIELDPRESS",
	SkillLGReflectdamage:                    "LG_REFLECTDAMAGE",
	SkillLGPinpointattack:                   "LG_PINPOINTATTACK",
	SkillLGForceofvanguard:                  "LG_FORCEOFVANGUARD",
	SkillLGRageburst:                        "LG_RAGEBURST",
	SkillLGShieldspell:                      "LG_SHIELDSPELL",
	SkillLGExeedbreak:                       "LG_EXEEDBREAK",
	SkillLGOverbrand:                        "LG_OVERBRAND",
	SkillLGPrestige:                         "LG_PRESTIGE",
	SkillLGBanding:                          "LG_BANDING",
	SkillLGMoonslasher:                      "LG_MOONSLASHER",
	SkillLGRayofgenesis:                     "LG_RAYOFGENESIS",
	SkillLGPiety:                            "LG_PIETY",
	SkillLGEarthdrive:                       "LG_EARTHDRIVE",
	SkillLGHesperuslit:                      "LG_HESPERUSLIT",
	SkillLGInspiration:                      "LG_INSPIRATION",
	SkillSRDragoncombo:                      "SR_DRAGONCOMBO",
	SkillSRSkynetblow:                       "SR_SKYNETBLOW",
	SkillSREarthshaker:                      "SR_EARTHSHAKER",
	SkillSRFallenempire:                     "SR_FALLENEMPIRE",
	SkillSRTigercannon:                      "SR_TIGERCANNON",
	SkillSRHellgate:                         "SR_HELLGATE",
	SkillSRRampageblaster:                   "SR_RAMPAGEBLASTER",
	SkillSRCrescentelbow:                    "SR_CRESCENTELBOW",
	SkillSRCursedcircle:                     "SR_CURSEDCIRCLE",
	SkillSRLightningwalk:                    "SR_LIGHTNINGWALK",
	SkillSRKnucklearrow:                     "SR_KNUCKLEARROW",
	SkillSRWindmill:                         "SR_WINDMILL",
	SkillSRRaisingdragon:                    "SR_RAISINGDRAGON",
	SkillSRGentletouch:                      "SR_GENTLETOUCH",
	SkillSRAssimilatepower:                  "SR_ASSIMILATEPOWER",
	SkillSRPowervelocity:                    "SR_POWERVELOCITY",
	SkillSRCrescentelbowAutospell:           "SR_CRESCENTELBOW_AUTOSPELL",
	SkillSRGateofhell:                       "SR_GATEOFHELL",
	SkillSRGentletouchQuiet:                 "SR_GENTLETOUCH_QUIET",
	SkillSRGentletouchCure:                  "SR_GENTLETOUCH_CURE",
	SkillSRGentletouchEnergygain:            "SR_GENTLETOUCH_ENERGYGAIN",
	SkillSRGentletouchChange:                "SR_GENTLETOUCH_CHANGE",
	SkillSRGentletouchRevitalize:            "SR_GENTLETOUCH_REVITALIZE",
	SkillWAStartmark:                        "WA_STARTMARK",
	SkillWASwingDance:                       "WA_SWING_DANCE",
	SkillWASymphonyOfLover:                  "WA_SYMPHONY_OF_LOVER",
	SkillWAMoonlitSerenade:                  "WA_MOONLIT_SERENADE",
	SkillWAEndmark:                          "WA_ENDMARK",
	SkillMIStartmark:                        "MI_STARTMARK",
	SkillMIRushWindmill:                     "MI_RUSH_WINDMILL",
	SkillMIEchosong:                         "MI_ECHOSONG",
	SkillMIHarmonize:                        "MI_HARMONIZE",
	SkillMIEndmark:                          "MI_ENDMARK",
	SkillWmStartmark:                        "WM_STARTMARK",
	SkillWmLesson:                           "WM_LESSON",
	SkillWmMetalicsound:                     "WM_METALICSOUND",
	SkillWmReverberation:                    "WM_REVERBERATION",
	SkillWmReverberationMelee:               "WM_REVERBERATION_MELEE",
	SkillWmReverberationMagic:               "WM_REVERBERATION_MAGIC",
	SkillWmDominionImpulse:                  "WM_DOMINION_IMPULSE",
	SkillWmSevereRainstorm:                  "WM_SEVERE_RAINSTORM",
	SkillWmPoemofnetherworld:                "WM_POEMOFNETHERWORLD",
	SkillWmVoiceofsiren:                     "WM_VOICEOFSIREN",
	SkillWmDeadhillhere:                     "WM_DEADHILLHERE",
	SkillWmLullabyDeepsleep:                 "WM_LULLABY_DEEPSLEEP",
	SkillWmSircleofnature:                   "WM_SIRCLEOFNATURE",
	SkillWmRandomizespell:                   "WM_RANDOMIZESPELL",
	SkillWmGloomyday:                        "WM_GLOOMYDAY",
	SkillWmGreatEcho:                        "WM_GREAT_ECHO",
	SkillWmSongOfMana:                       "WM_SONG_OF_MANA",
	SkillWmDanceWithWug:                     "WM_DANCE_WITH_WUG",
	SkillWmSoundOfDestruction:               "WM_SOUND_OF_DESTRUCTION",
	SkillWmSaturdayNightFever:               "WM_SATURDAY_NIGHT_FEVER",
	SkillWmLeradsDew:                        "WM_LERADS_DEW",
	SkillWmMelodyofsink:                     "WM_MELODYOFSINK",
	SkillWmBeyondOfWarcry:                   "WM_BEYOND_OF_WARCRY",
	SkillWmUnlimitedHummingVoice:            "WM_UNLIMITED_HUMMING_VOICE",
	SkillWmEndmark:                          "WM_ENDMARK",
	SkillSOStartmark:                        "SO_STARTMARK",
	SkillSOFirewalk:                         "SO_FIREWALK",
	SkillSOElectricwalk:                     "SO_ELECTRICWALK",
	SkillSOSpellfist:                        "SO_SPELLFIST",
	SkillSOEarthgrave:                       "SO_EARTHGRAVE",
	SkillSODiamonddust:                      "SO_DIAMONDDUST",
	SkillSOPoisonBuster:                     "SO_POISON_BUSTER",
	SkillSOPsychicWave:                      "SO_PSYCHIC_WAVE",
	SkillSOCloudKill:                        "SO_CLOUD_KILL",
	SkillSOStriking:                         "SO_STRIKING",
	SkillSOWarmer:                           "SO_WARMER",
	SkillSOVacuumExtreme:                    "SO_VACUUM_EXTREME",
	SkillSOVaretyrSpear:                     "SO_VARETYR_SPEAR",
	SkillSOArrullo:                          "SO_ARRULLO",
	SkillSOElControl:                        "SO_EL_CONTROL",
	SkillSOSummonAgni:                       "SO_SUMMON_AGNI",
	SkillSOSummonAqua:                       "SO_SUMMON_AQUA",
	SkillSOSummonVentus:                     "SO_SUMMON_VENTUS",
	SkillSOSummonTera:                       "SO_SUMMON_TERA",
	SkillSOElAction:                         "SO_EL_ACTION",
	SkillSOElAnalysis:                       "SO_EL_ANALYSIS",
	SkillSOElSympathy:                       "SO_EL_SYMPATHY",
	SkillSOElCure:                           "SO_EL_CURE",
	SkillSOFireInsignia:                     "SO_FIRE_INSIGNIA",
	SkillSOWaterInsignia:                    "SO_WATER_INSIGNIA",
	SkillSOWindInsignia:                     "SO_WIND_INSIGNIA",
	SkillSOEarthInsignia:                    "SO_EARTH_INSIGNIA",
	SkillSOEndmark:                          "SO_ENDMARK",
	SkillGNStartMark:                        "GN_START_MARK",
	SkillGNTrainingSword:                    "GN_TRAINING_SWORD",
	SkillGNRemodelingCart:                   "GN_REMODELING_CART",
	SkillGNCartTornado:                      "GN_CART_TORNADO",
	SkillGNCartcannon:                       "GN_CARTCANNON",
	SkillGNCartboost:                        "GN_CARTBOOST",
	SkillGNThornsTrap:                       "GN_THORNS_TRAP",
	SkillGNBloodSucker:                      "GN_BLOOD_SUCKER",
	SkillGNSporeExplosion:                   "GN_SPORE_EXPLOSION",
	SkillGNWallofthorn:                      "GN_WALLOFTHORN",
	SkillGNCrazyweed:                        "GN_CRAZYWEED",
	SkillGNCrazyweedAtk:                     "GN_CRAZYWEED_ATK",
	SkillGNDemonicFire:                      "GN_DEMONIC_FIRE",
	SkillGNFireExpansion:                    "GN_FIRE_EXPANSION",
	SkillGNFireExpansionSmokePowder:         "GN_FIRE_EXPANSION_SMOKE_POWDER",
	SkillGNFireExpansionTearGas:             "GN_FIRE_EXPANSION_TEAR_GAS",
	SkillGNFireExpansionAcid:                "GN_FIRE_EXPANSION_ACID",
	SkillGNHellsPlant:                       "GN_HELLS_PLANT",
	SkillGNHellsPlantAtk:                    "GN_HELLS_PLANT_ATK",
	SkillGNMandragora:                       "GN_MANDRAGORA",
	SkillGNSlingitem:                        "GN_SLINGITEM",
	SkillGNChangematerial:                   "GN_CHANGEMATERIAL",
	SkillGNMixCooking:                       "GN_MIX_COOKING",
	SkillGNMakebomb:                         "GN_MAKEBOMB",
	SkillGNSPharmacy:                        "GN_S_PHARMACY",
	SkillGNSlingitemRangemeleeatk:           "GN_SLINGITEM_RANGEMELEEATK",
	SkillGNEndmark:                          "GN_ENDMARK",
	SkillEtcThirdjobSkillStart:              "ETC_THIRDJOB_SKILL_START",
	SkillABSecrament:                        "AB_SECRAMENT",
	SkillWmSevereRainstormMelee:             "WM_SEVERE_RAINSTORM_MELEE",
	SkillSRHowlingoflion:                    "SR_HOWLINGOFLION",
	SkillSRRideinlightning:                  "SR_RIDEINLIGHTNING",
	SkillLGOverbrandBrandish:                "LG_OVERBRAND_BRANDISH",
	SkillLGOverbrandPlusatk:                 "LG_OVERBRAND_PLUSATK",
	SkillEtcThirdjobSkillEnd:                "ETC_THIRDJOB_SKILL_END",
	SkillThirdjobEnd:                        "THIRDJOB_END",
	SkillALLOdinsRecall:                     "ALL_ODINS_RECALL",
	SkillReturnToEldicastes:                 "RETURN_TO_ELDICASTES",
	SkillALLBuyingStore:                     "ALL_BUYING_STORE",
	SkillALLGuardianRecall:                  "ALL_GUARDIAN_RECALL",
	SkillALLOdinsPower:                      "ALL_ODINS_POWER",
	SkillXxBeerBottleCap:                    "XX_BEER_BOTTLE_CAP",
	SkillNPCAssassincross:                   "NPC_ASSASSINCROSS",
	SkillNPCDissonance:                      "NPC_DISSONANCE",
	SkillNPCUglydance:                       "NPC_UGLYDANCE",
	SkillALLTetany:                          "ALL_TETANY",
	SkillALLRayOfProtection:                 "ALL_RAY_OF_PROTECTION",
	SkillMCCartdecorate:                     "MC_CARTDECORATE",
	SkillGmItemAtkmax:                       "GM_ITEM_ATKMAX",
	SkillGmItemAtkmin:                       "GM_ITEM_ATKMIN",
	SkillGmItemMatkmax:                      "GM_ITEM_MATKMAX",
	SkillGmItemMatkmin:                      "GM_ITEM_MATKMIN",
	SkillGmApHeal:                           "GM_AP_HEAL",
	SkillUpperExtendedJobStart:              "UPPER_EXTENDED_JOB_START",
	SkillRLGlitteringGreed:                  "RL_GLITTERING_GREED",
	SkillRLRichsCoin:                        "RL_RICHS_COIN",
	SkillRLMassSpiral:                       "RL_MASS_SPIRAL",
	SkillRLBanishingBuster:                  "RL_BANISHING_BUSTER",
	SkillRLBTrap:                            "RL_B_TRAP",
	SkillRLFlicker:                          "RL_FLICKER",
	SkillRLSStorm:                           "RL_S_STORM",
	SkillRLEChain:                           "RL_E_CHAIN",
	SkillRLQdShot:                           "RL_QD_SHOT",
	SkillRLCMarker:                          "RL_C_MARKER",
	SkillRLFiredance:                        "RL_FIREDANCE",
	SkillRLHMine:                            "RL_H_MINE",
	SkillRLPAlter:                           "RL_P_ALTER",
	SkillRLFallenAngel:                      "RL_FALLEN_ANGEL",
	SkillRLRTrip:                            "RL_R_TRIP",
	SkillRLDTail:                            "RL_D_TAIL",
	SkillRLFireRain:                         "RL_FIRE_RAIN",
	SkillRLHeatBarrel:                       "RL_HEAT_BARREL",
	SkillRLAMBlast:                          "RL_AM_BLAST",
	SkillRLSlugshot:                         "RL_SLUGSHOT",
	SkillRLHammerOfGod:                      "RL_HAMMER_OF_GOD",
	SkillRLRTripPlusatk:                     "RL_R_TRIP_PLUSATK",
	SkillRLBFlickerAtk:                      "RL_B_FLICKER_ATK",
	SkillRLGlitteringGreedAtk:               "RL_GLITTERING_GREED_ATK",
	SkillSJLunarstance:                      "SJ_LUNARSTANCE",
	SkillSJFullmoonkick:                     "SJ_FULLMOONKICK",
	SkillSJLightofstar:                      "SJ_LIGHTOFSTAR",
	SkillSJStarstance:                       "SJ_STARSTANCE",
	SkillSJNewmoonkick:                      "SJ_NEWMOONKICK",
	SkillSJFlashkick:                        "SJ_FLASHKICK",
	SkillSJStaremperor:                      "SJ_STAREMPEROR",
	SkillSJNovaexplosing:                    "SJ_NOVAEXPLOSING",
	SkillSJUniversestance:                   "SJ_UNIVERSESTANCE",
	SkillSJFallingstar:                      "SJ_FALLINGSTAR",
	SkillSJGravitycontrol:                   "SJ_GRAVITYCONTROL",
	SkillSJBookofdimension:                  "SJ_BOOKOFDIMENSION",
	SkillSJBookofcreatingstar:               "SJ_BOOKOFCREATINGSTAR",
	SkillSJDocument:                         "SJ_DOCUMENT",
	SkillSJPurify:                           "SJ_PURIFY",
	SkillSJLightofsun:                       "SJ_LIGHTOFSUN",
	SkillSJSunstance:                        "SJ_SUNSTANCE",
	SkillSJSolarburst:                       "SJ_SOLARBURST",
	SkillSJProminencekick:                   "SJ_PROMINENCEKICK",
	SkillSJFallingstarAtk:                   "SJ_FALLINGSTAR_ATK",
	SkillSJFallingstarAtk2:                  "SJ_FALLINGSTAR_ATK2",
	SkillSPSoulgolem:                        "SP_SOULGOLEM",
	SkillSPSoulshadow:                       "SP_SOULSHADOW",
	SkillSPSoulfalcon:                       "SP_SOULFALCON",
	SkillSPSoulfairy:                        "SP_SOULFAIRY",
	SkillSPCurseexplosion:                   "SP_CURSEEXPLOSION",
	SkillSPSoulcurse:                        "SP_SOULCURSE",
	SkillSPSpa:                              "SP_SPA",
	SkillSPSha:                              "SP_SHA",
	SkillSPSwhoo:                            "SP_SWHOO",
	SkillSPSoulunity:                        "SP_SOULUNITY",
	SkillSPSouldivision:                     "SP_SOULDIVISION",
	SkillSPSoulreaper:                       "SP_SOULREAPER",
	SkillSPSoulrevolve:                      "SP_SOULREVOLVE",
	SkillSPSoulcollect:                      "SP_SOULCOLLECT",
	SkillSPSoulexplosion:                    "SP_SOULEXPLOSION",
	SkillSPSoulenergy:                       "SP_SOULENERGY",
	SkillSPKaute:                            "SP_KAUTE",
	SkillKOYamikumo:                         "KO_YAMIKUMO",
	SkillKORight:                            "KO_RIGHT",
	SkillKOLeft:                             "KO_LEFT",
	SkillKOJyumonjikiri:                     "KO_JYUMONJIKIRI",
	SkillKOSetsudan:                         "KO_SETSUDAN",
	SkillKOBakuretsu:                        "KO_BAKURETSU",
	SkillKOHappokunai:                       "KO_HAPPOKUNAI",
	SkillKOMuchanage:                        "KO_MUCHANAGE",
	SkillKOHuumaranka:                       "KO_HUUMARANKA",
	SkillKOMakibishi:                        "KO_MAKIBISHI",
	SkillKOMeikyousisui:                     "KO_MEIKYOUSISUI",
	SkillKOZanzou:                           "KO_ZANZOU",
	SkillKOKyougaku:                         "KO_KYOUGAKU",
	SkillKOJyusatsu:                         "KO_JYUSATSU",
	SkillKOKahuEnten:                        "KO_KAHU_ENTEN",
	SkillKOHyouhuHubuki:                     "KO_HYOUHU_HUBUKI",
	SkillKOKazehuSeiran:                     "KO_KAZEHU_SEIRAN",
	SkillKODohuKoukai:                       "KO_DOHU_KOUKAI",
	SkillKOKaihou:                           "KO_KAIHOU",
	SkillKOZenkai:                           "KO_ZENKAI",
	SkillKOGenwaku:                          "KO_GENWAKU",
	SkillKOIzayoi:                           "KO_IZAYOI",
	SkillKgKagehumi:                         "KG_KAGEHUMI",
	SkillKgKyomu:                            "KG_KYOMU",
	SkillKgKagemusya:                        "KG_KAGEMUSYA",
	SkillObZangetsu:                         "OB_ZANGETSU",
	SkillObOborogensou:                      "OB_OBOROGENSOU",
	SkillObOborogensouTransitionAtk:         "OB_OBOROGENSOU_TRANSITION_ATK",
	SkillObAkaitsuki:                        "OB_AKAITSUKI",
	SkillUpperExtendedJobEnd:                "UPPER_EXTENDED_JOB_END",
	SkillEclSnowflip:                        "ECL_SNOWFLIP",
	SkillEclPeonymamy:                       "ECL_PEONYMAMY",
	SkillEclSadagui:                         "ECL_SADAGUI",
	SkillEclSequoiadust:                     "ECL_SEQUOIADUST",
	SkillEclageRecall:                       "ECLAGE_RECALL",
	SkillBaPoembragi2:                       "BA_POEMBRAGI2",
	SkillDCFortunekiss2:                     "DC_FORTUNEKISS2",
	SkillItemOptionSplashAttack:             "ITEM_OPTION_SPLASH_ATTACK",
	SkillGmForceTransfer:                    "GM_FORCE_TRANSFER",
	SkillGmWideResurrection:                 "GM_WIDE_RESURRECTION",
	SkillALLNiflheimRecall:                  "ALL_NIFLHEIM_RECALL",
	SkillALLPronteraRecall:                  "ALL_PRONTERA_RECALL",
	SkillALLGlastheimRecall:                 "ALL_GLASTHEIM_RECALL",
	SkillALLThanatosRecall:                  "ALL_THANATOS_RECALL",
	SkillLevelExpansionStart:                "LEVEL_EXPANSION_START",
	SkillGCDarkcrow:                         "GC_DARKCROW",
	SkillRAUnlimit:                          "RA_UNLIMIT",
	SkillGNIllusiondoping:                   "GN_ILLUSIONDOPING",
	SkillRKDragonbreathWater:                "RK_DRAGONBREATH_WATER",
	SkillRKLuxanima:                         "RK_LUXANIMA",
	SkillNCMagmaEruption:                    "NC_MAGMA_ERUPTION",
	SkillWmFriggSong:                        "WM_FRIGG_SONG",
	SkillSOElementalShield:                  "SO_ELEMENTAL_SHIELD",
	SkillSRFlashcombo:                       "SR_FLASHCOMBO",
	SkillSCEscape:                           "SC_ESCAPE",
	SkillABOffertorium:                      "AB_OFFERTORIUM",
	SkillWLTelekinesisIntense:               "WL_TELEKINESIS_INTENSE",
	SkillLGKingsGrace:                       "LG_KINGS_GRACE",
	SkillALLFullThrottle:                    "ALL_FULL_THROTTLE",
	SkillNCMagmaEruptionDotdamage:           "NC_MAGMA_ERUPTION_DOTDAMAGE",
	SkillLevelExpansionEnd:                  "LEVEL_EXPANSION_END",
	SkillDoramTribeStart:                    "DORAM_TRIBE_START",
	SkillSUBasicSkill:                       "SU_BASIC_SKILL",
	SkillSUBite:                             "SU_BITE",
	SkillSUHide:                             "SU_HIDE",
	SkillSUScratch:                          "SU_SCRATCH",
	SkillSUStoop:                            "SU_STOOP",
	SkillSULope:                             "SU_LOPE",
	SkillSUSpritemable:                      "SU_SPRITEMABLE",
	SkillSUPowerofland:                      "SU_POWEROFLAND",
	SkillSUSvStemspear:                      "SU_SV_STEMSPEAR",
	SkillSUCnPowdering:                      "SU_CN_POWDERING",
	SkillSUCnMeteor:                         "SU_CN_METEOR",
	SkillSUSvRoottwist:                      "SU_SV_ROOTTWIST",
	SkillSUSvRoottwistAtk:                   "SU_SV_ROOTTWIST_ATK",
	SkillSUPoweroflife:                      "SU_POWEROFLIFE",
	SkillSUScaroftarou:                      "SU_SCAROFTAROU",
	SkillSUPickypeck:                        "SU_PICKYPECK",
	SkillSUPickypeckDoubleAtk:               "SU_PICKYPECK_DOUBLE_ATK",
	SkillSUArclousedash:                     "SU_ARCLOUSEDASH",
	SkillSULunaticcarrotbeat:                "SU_LUNATICCARROTBEAT",
	SkillSUPowerofsea:                       "SU_POWEROFSEA",
	SkillSUTunabelly:                        "SU_TUNABELLY",
	SkillSUTunaparty:                        "SU_TUNAPARTY",
	SkillSUBunchofshrimp:                    "SU_BUNCHOFSHRIMP",
	SkillSUFreshshrimp:                      "SU_FRESHSHRIMP",
	SkillSUCnMeteor2:                        "SU_CN_METEOR2",
	SkillSULunaticcarrotbeat2:               "SU_LUNATICCARROTBEAT2",
	SkillSUSoulattack:                       "SU_SOULATTACK",
	SkillSUPowerofflock:                     "SU_POWEROFFLOCK",
	SkillSUSvgSpirit:                        "SU_SVG_SPIRIT",
	SkillSUHiss:                             "SU_HISS",
	SkillSUNyanggrass:                       "SU_NYANGGRASS",
	SkillSUGrooming:                         "SU_GROOMING",
	SkillSUPurring:                          "SU_PURRING",
	SkillSUShrimparty:                       "SU_SHRIMPARTY",
	SkillSUSpiritoflife:                     "SU_SPIRITOFLIFE",
	SkillSUMeowmeow:                         "SU_MEOWMEOW",
	SkillSUSpiritofland:                     "SU_SPIRITOFLAND",
	SkillSUChattering:                       "SU_CHATTERING",
	SkillSUSpiritofsea:                      "SU_SPIRITOFSEA",
	SkillDoramTribeEnd:                      "DORAM_TRIBE_END",
	SkillLast:                               "LAST",
	SkillWECallallfamily:                    "WE_CALLALLFAMILY",
	SkillWEOneforever:                       "WE_ONEFOREVER",
	SkillWECheerup:                          "WE_CHEERUP",
	SkillCGSpecialsinger:                    "CG_SPECIALSINGER",
	SkillABVituperatum:                      "AB_VITUPERATUM",
	SkillABConvenio:                         "AB_CONVENIO",
	SkillNVBreakthrough:                     "NV_BREAKTHROUGH",
	SkillNVHelpangel:                        "NV_HELPANGEL",
	SkillNVTranscendence:                    "NV_TRANSCENDENCE",
	SkillWLReadingSbReading:                 "WL_READING_SB_READING",
	SkillDkServantweapon:                    "DK_SERVANTWEAPON",
	SkillDkServantweaponAtk:                 "DK_SERVANTWEAPON_ATK",
	SkillDkServantWSign:                     "DK_SERVANT_W_SIGN",
	SkillDkServantWPhantom:                  "DK_SERVANT_W_PHANTOM",
	SkillDkServantWDemol:                    "DK_SERVANT_W_DEMOL",
	SkillDkChargingpierce:                   "DK_CHARGINGPIERCE",
	SkillDkTwohanddef:                       "DK_TWOHANDDEF",
	SkillDkHackandslasher:                   "DK_HACKANDSLASHER",
	SkillDkHackandslasherAtk:                "DK_HACKANDSLASHER_ATK",
	SkillDkDragonicAura:                     "DK_DRAGONIC_AURA",
	SkillDkMadnessCrusher:                   "DK_MADNESS_CRUSHER",
	SkillDkVigor:                            "DK_VIGOR",
	SkillDkStormslash:                       "DK_STORMSLASH",
	SkillAgDeadlyProjection:                 "AG_DEADLY_PROJECTION",
	SkillAgDestructiveHurricane:             "AG_DESTRUCTIVE_HURRICANE",
	SkillAgRainOfCrystal:                    "AG_RAIN_OF_CRYSTAL",
	SkillAgMysteryIllusion:                  "AG_MYSTERY_ILLUSION",
	SkillAgViolentQuake:                     "AG_VIOLENT_QUAKE",
	SkillAgViolentQuakeAtk:                  "AG_VIOLENT_QUAKE_ATK",
	SkillAgSoulVcStrike:                     "AG_SOUL_VC_STRIKE",
	SkillAgStrantumTremor:                   "AG_STRANTUM_TREMOR",
	SkillAgALLBloom:                         "AG_ALL_BLOOM",
	SkillAgALLBloomAtk:                      "AG_ALL_BLOOM_ATK",
	SkillAgALLBloomAtk2:                     "AG_ALL_BLOOM_ATK2",
	SkillAgCrystalImpact:                    "AG_CRYSTAL_IMPACT",
	SkillAgCrystalImpactAtk:                 "AG_CRYSTAL_IMPACT_ATK",
	SkillAgTornadoStorm:                     "AG_TORNADO_STORM",
	SkillAgTwohandstaff:                     "AG_TWOHANDSTAFF",
	SkillAgFloralFlareRoad:                  "AG_FLORAL_FLARE_ROAD",
	SkillAgAstralStrike:                     "AG_ASTRAL_STRIKE",
	SkillAgAstralStrikeAtk:                  "AG_ASTRAL_STRIKE_ATK",
	SkillAgClimax:                           "AG_CLIMAX",
	SkillAgRockDown:                         "AG_ROCK_DOWN",
	SkillAgStormCannon:                      "AG_STORM_CANNON",
	SkillAgCrimsonArrow:                     "AG_CRIMSON_ARROW",
	SkillAgCrimsonArrowAtk:                  "AG_CRIMSON_ARROW_ATK",
	SkillAgFrozenSlash:                      "AG_FROZEN_SLASH",
	SkillIqPowerfulFaith:                    "IQ_POWERFUL_FAITH",
	SkillIqFirmFaith:                        "IQ_FIRM_FAITH",
	SkillIqWillOfFaith:                      "IQ_WILL_OF_FAITH",
	SkillIqOleumSanctum:                     "IQ_OLEUM_SANCTUM",
	SkillIqSincereFaith:                     "IQ_SINCERE_FAITH",
	SkillIqMassiveFBlaster:                  "IQ_MASSIVE_F_BLASTER",
	SkillIqExposionBlaster:                  "IQ_EXPOSION_BLASTER",
	SkillIqFirstBrand:                       "IQ_FIRST_BRAND",
	SkillIqFirstFaithPower:                  "IQ_FIRST_FAITH_POWER",
	SkillIqJudge:                            "IQ_JUDGE",
	SkillIqSecondFlame:                      "IQ_SECOND_FLAME",
	SkillIqSecondFaith:                      "IQ_SECOND_FAITH",
	SkillIqSecondJudgement:                  "IQ_SECOND_JUDGEMENT",
	SkillIqThirdPunish:                      "IQ_THIRD_PUNISH",
	SkillIqThirdFlameBomb:                   "IQ_THIRD_FLAME_BOMB",
	SkillIqThirdConsecration:                "IQ_THIRD_CONSECRATION",
	SkillIqThirdExorFlame:                   "IQ_THIRD_EXOR_FLAME",
	SkillIgGuardStance:                      "IG_GUARD_STANCE",
	SkillIgGuardianShield:                   "IG_GUARDIAN_SHIELD",
	SkillIgReboundShield:                    "IG_REBOUND_SHIELD",
	SkillIgShieldMastery:                    "IG_SHIELD_MASTERY",
	SkillIgSpearSwordM:                      "IG_SPEAR_SWORD_M",
	SkillIgAttackStance:                     "IG_ATTACK_STANCE",
	SkillIgUltimateSacrifice:                "IG_ULTIMATE_SACRIFICE",
	SkillIgHolyShield:                       "IG_HOLY_SHIELD",
	SkillIgGrandJudgement:                   "IG_GRAND_JUDGEMENT",
	SkillIgJudgementCross:                   "IG_JUDGEMENT_CROSS",
	SkillIgShieldShooting:                   "IG_SHIELD_SHOOTING",
	SkillIgOverslash:                        "IG_OVERSLASH",
	SkillIgCrossRain:                        "IG_CROSS_RAIN",
	SkillShcShadowExceed:                    "SHC_SHADOW_EXCEED",
	SkillShcDancingKnife:                    "SHC_DANCING_KNIFE",
	SkillShcSavageImpact:                    "SHC_SAVAGE_IMPACT",
	SkillShcShadowSense:                     "SHC_SHADOW_SENSE",
	SkillShcEternalSlash:                    "SHC_ETERNAL_SLASH",
	SkillShcPotentVenom:                     "SHC_POTENT_VENOM",
	SkillShcShadowStab:                      "SHC_SHADOW_STAB",
	SkillShcImpactCrater:                    "SHC_IMPACT_CRATER",
	SkillShcEnchantingShadow:                "SHC_ENCHANTING_SHADOW",
	SkillShcFatalShadowCrow:                 "SHC_FATAL_SHADOW_CROW",
	SkillCdReparatio:                        "CD_REPARATIO",
	SkillCdMedialeVotum:                     "CD_MEDIALE_VOTUM",
	SkillCdMaceBookM:                        "CD_MACE_BOOK_M",
	SkillCdArgutusVita:                      "CD_ARGUTUS_VITA",
	SkillCdArgutusTelum:                     "CD_ARGUTUS_TELUM",
	SkillCdArbitrium:                        "CD_ARBITRIUM",
	SkillCdArbitriumAtk:                     "CD_ARBITRIUM_ATK",
	SkillCdPresensAcies:                     "CD_PRESENS_ACIES",
	SkillCdFidusAnimus:                      "CD_FIDUS_ANIMUS",
	SkillCdEffligo:                          "CD_EFFLIGO",
	SkillCdCompetentia:                      "CD_COMPETENTIA",
	SkillCdPneumaticusProcella:              "CD_PNEUMATICUS_PROCELLA",
	SkillCdDilectioHeal:                     "CD_DILECTIO_HEAL",
	SkillCdReligio:                          "CD_RELIGIO",
	SkillCdBenedictum:                       "CD_BENEDICTUM",
	SkillCdPetitio:                          "CD_PETITIO",
	SkillCdFramen:                           "CD_FRAMEN",
	SkillBoBionicPharmacy:                   "BO_BIONIC_PHARMACY",
	SkillBoBionicsM:                         "BO_BIONICS_M",
	SkillBoTheWholeProtection:               "BO_THE_WHOLE_PROTECTION",
	SkillBoAdvanceProtection:                "BO_ADVANCE_PROTECTION",
	SkillBoAcidifiedZoneWater:               "BO_ACIDIFIED_ZONE_WATER",
	SkillBoAcidifiedZoneGround:              "BO_ACIDIFIED_ZONE_GROUND",
	SkillBoAcidifiedZoneWind:                "BO_ACIDIFIED_ZONE_WIND",
	SkillBoAcidifiedZoneFire:                "BO_ACIDIFIED_ZONE_FIRE",
	SkillBoWoodenwarrior:                    "BO_WOODENWARRIOR",
	SkillBoWoodenFairy:                      "BO_WOODEN_FAIRY",
	SkillBoCreeper:                          "BO_CREEPER",
	SkillBoResearchreport:                   "BO_RESEARCHREPORT",
	SkillBoHelltree:                         "BO_HELLTREE",
	SkillWhAdvancedTrap:                     "WH_ADVANCED_TRAP",
	SkillWhWindSign:                         "WH_WIND_SIGN",
	SkillWhNaturefriendly:                   "WH_NATUREFRIENDLY",
	SkillWhHawkrush:                         "WH_HAWKRUSH",
	SkillWhHawkM:                            "WH_HAWK_M",
	SkillWhCalamitygale:                     "WH_CALAMITYGALE",
	SkillWhHawkboomerang:                    "WH_HAWKBOOMERANG",
	SkillWhGalestorm:                        "WH_GALESTORM",
	SkillWhDeepblindtrap:                    "WH_DEEPBLINDTRAP",
	SkillWhSolidtrap:                        "WH_SOLIDTRAP",
	SkillWhSwifttrap:                        "WH_SWIFTTRAP",
	SkillWhCresciveBolt:                     "WH_CRESCIVE_BOLT",
	SkillWhFlametrap:                        "WH_FLAMETRAP",
	SkillTrStageManner:                      "TR_STAGE_MANNER",
	SkillTrRetrospection:                    "TR_RETROSPECTION",
	SkillTrMysticSymphony:                   "TR_MYSTIC_SYMPHONY",
	SkillTrKvasirSonata:                     "TR_KVASIR_SONATA",
	SkillTrRoseblossom:                      "TR_ROSEBLOSSOM",
	SkillTrRoseblossomAtk:                   "TR_ROSEBLOSSOM_ATK",
	SkillTrRhythmshooting:                   "TR_RHYTHMSHOOTING",
	SkillTrMetalicFury:                      "TR_METALIC_FURY",
	SkillTrSoundblend:                       "TR_SOUNDBLEND",
	SkillTrGefNocturn:                       "TR_GEF_NOCTURN",
	SkillTrRokiCapriccio:                    "TR_ROKI_CAPRICCIO",
	SkillTrAinRhapsody:                      "TR_AIN_RHAPSODY",
	SkillTrMusicalInterlude:                 "TR_MUSICAL_INTERLUDE",
	SkillTrJawaiiSerenade:                   "TR_JAWAII_SERENADE",
	SkillTrNipelheimRequiem:                 "TR_NIPELHEIM_REQUIEM",
	SkillTrPronMarch:                        "TR_PRON_MARCH",
	SkillAbcDaggerAndBowM:                   "ABC_DAGGER_AND_BOW_M",
	SkillAbcMagicSwordM:                     "ABC_MAGIC_SWORD_M",
	SkillAbcStripShadow:                     "ABC_STRIP_SHADOW",
	SkillAbcAbyssDagger:                     "ABC_ABYSS_DAGGER",
	SkillAbcUnluckyRush:                     "ABC_UNLUCKY_RUSH",
	SkillAbcChainReactionShot:               "ABC_CHAIN_REACTION_SHOT",
	SkillAbcFromTheAbyss:                    "ABC_FROM_THE_ABYSS",
	SkillAbcAbyssSlayer:                     "ABC_ABYSS_SLAYER",
	SkillAbcAbyssStrike:                     "ABC_ABYSS_STRIKE",
	SkillAbcDeftStab:                        "ABC_DEFT_STAB",
	SkillAbcAbyssSquare:                     "ABC_ABYSS_SQUARE",
	SkillAbcFrenzyShot:                      "ABC_FRENZY_SHOT",
	SkillAbcChainReactionShotAtk:            "ABC_CHAIN_REACTION_SHOT_ATK",
	SkillAbcFromTheAbyssAtk:                 "ABC_FROM_THE_ABYSS_ATK",
	SkillNPCBoThrowrock:                     "NPC_BO_THROWROCK",
	SkillNPCBoWoodenAttack:                  "NPC_BO_WOODEN_ATTACK",
	SkillNPCBoHellHowling:                   "NPC_BO_HELL_HOWLING",
	SkillNPCBoHellDusty:                     "NPC_BO_HELL_DUSTY",
	SkillNPCBoFairyDusty:                    "NPC_BO_FAIRY_DUSTY",
	SkillMtAxeStomp:                         "MT_AXE_STOMP",
	SkillMtRushQuake:                        "MT_RUSH_QUAKE",
	SkillMtMMachine:                         "MT_M_MACHINE",
	SkillMtAMachine:                         "MT_A_MACHINE",
	SkillMtDMachine:                         "MT_D_MACHINE",
	SkillMtTwoaxedef:                        "MT_TWOAXEDEF",
	SkillMtAbrM:                             "MT_ABR_M",
	SkillMtSummonAbrBattleWarior:            "MT_SUMMON_ABR_BATTLE_WARIOR",
	SkillMtSummonAbrDualCannon:              "MT_SUMMON_ABR_DUAL_CANNON",
	SkillMtSummonAbrMotherNet:               "MT_SUMMON_ABR_MOTHER_NET",
	SkillMtSummonAbrInfinity:                "MT_SUMMON_ABR_INFINITY",
	SkillAbrBattleBuster:                    "ABR_BATTLE_BUSTER",
	SkillAbrDualCannonFire:                  "ABR_DUAL_CANNON_FIRE",
	SkillAbrNetRepair:                       "ABR_NET_REPAIR",
	SkillAbrNetSupport:                      "ABR_NET_SUPPORT",
	SkillAbrInfinityBuster:                  "ABR_INFINITY_BUSTER",
	SkillEmMagicBookM:                       "EM_MAGIC_BOOK_M",
	SkillEmSpellEnchanting:                  "EM_SPELL_ENCHANTING",
	SkillEmActivityBurn:                     "EM_ACTIVITY_BURN",
	SkillEmIncreasingActivity:               "EM_INCREASING_ACTIVITY",
	SkillEmDiamondStorm:                     "EM_DIAMOND_STORM",
	SkillEmLightningLand:                    "EM_LIGHTNING_LAND",
	SkillEmVenomSwamp:                       "EM_VENOM_SWAMP",
	SkillEmConflagration:                    "EM_CONFLAGRATION",
	SkillEmTerraDrive:                       "EM_TERRA_DRIVE",
	SkillEmElementalSpiritM:                 "EM_ELEMENTAL_SPIRIT_M",
	SkillEmSummonElementalArdor:             "EM_SUMMON_ELEMENTAL_ARDOR",
	SkillEmSummonElementalDiluvio:           "EM_SUMMON_ELEMENTAL_DILUVIO",
	SkillEmSummonElementalProcella:          "EM_SUMMON_ELEMENTAL_PROCELLA",
	SkillEmSummonElementalTerremotus:        "EM_SUMMON_ELEMENTAL_TERREMOTUS",
	SkillEmSummonElementalSerpens:           "EM_SUMMON_ELEMENTAL_SERPENS",
	SkillEmElementalBuster:                  "EM_ELEMENTAL_BUSTER",
	SkillEmElementalVeil:                    "EM_ELEMENTAL_VEIL",
	SkillEmElementalBusterFire:              "EM_ELEMENTAL_BUSTER_FIRE",
	SkillEmElementalBusterWater:             "EM_ELEMENTAL_BUSTER_WATER",
	SkillEmElementalBusterWind:              "EM_ELEMENTAL_BUSTER_WIND",
	SkillEmElementalBusterGround:            "EM_ELEMENTAL_BUSTER_GROUND",
	SkillEmElementalBusterPoison:            "EM_ELEMENTAL_BUSTER_POISON",
	SkillNwPFI:                              "NW_P_F_I",
	SkillNwGrenadeMastery:                   "NW_GRENADE_MASTERY",
	SkillNwIntensiveAim:                     "NW_INTENSIVE_AIM",
	SkillNwGrenadeFragment:                  "NW_GRENADE_FRAGMENT",
	SkillNwTheVigilanteAtNight:              "NW_THE_VIGILANTE_AT_NIGHT",
	SkillNwOnlyOneBullet:                    "NW_ONLY_ONE_BULLET",
	SkillNwSpiralShooting:                   "NW_SPIRAL_SHOOTING",
	SkillNwMagazineForOne:                   "NW_MAGAZINE_FOR_ONE",
	SkillNwWildFire:                         "NW_WILD_FIRE",
	SkillNwBasicGrenade:                     "NW_BASIC_GRENADE",
	SkillNwHastyFireInTheHole:               "NW_HASTY_FIRE_IN_THE_HOLE",
	SkillNwGrenadesDropping:                 "NW_GRENADES_DROPPING",
	SkillNwAutoFiringLauncher:               "NW_AUTO_FIRING_LAUNCHER",
	SkillNwHiddenCard:                       "NW_HIDDEN_CARD",
	SkillNwMissionBombard:                   "NW_MISSION_BOMBARD",
	SkillSoaTalismanMastery:                 "SOA_TALISMAN_MASTERY",
	SkillSoaSoulMastery:                     "SOA_SOUL_MASTERY",
	SkillSoaTalismanOfProtection:            "SOA_TALISMAN_OF_PROTECTION",
	SkillSoaTalismanOfWarrior:               "SOA_TALISMAN_OF_WARRIOR",
	SkillSoaTalismanOfMagician:              "SOA_TALISMAN_OF_MAGICIAN",
	SkillSoaSoulGathering:                   "SOA_SOUL_GATHERING",
	SkillSoaTotemOfTutelary:                 "SOA_TOTEM_OF_TUTELARY",
	SkillSoaTalismanOfFiveElements:          "SOA_TALISMAN_OF_FIVE_ELEMENTS",
	SkillSoaTalismanOfSoulStealing:          "SOA_TALISMAN_OF_SOUL_STEALING",
	SkillSoaExorcismOfMaliciousSoul:         "SOA_EXORCISM_OF_MALICIOUS_SOUL",
	SkillSoaTalismanOfBlueDragon:            "SOA_TALISMAN_OF_BLUE_DRAGON",
	SkillSoaTalismanOfWhiteTiger:            "SOA_TALISMAN_OF_WHITE_TIGER",
	SkillSoaTalismanOfRedPhoenix:            "SOA_TALISMAN_OF_RED_PHOENIX",
	SkillSoaTalismanOfBlackTortoise:         "SOA_TALISMAN_OF_BLACK_TORTOISE",
	SkillSoaTalismanOfFourBearingGod:        "SOA_TALISMAN_OF_FOUR_BEARING_GOD",
	SkillSoaCircleOfDirectionsAndElementals: "SOA_CIRCLE_OF_DIRECTIONS_AND_ELEMENTALS",
	SkillSoaSoulOfHeavenAndEarth:            "SOA_SOUL_OF_HEAVEN_AND_EARTH",
	SkillShMysticalCreatureMastery:          "SH_MYSTICAL_CREATURE_MASTERY",
	SkillShCommuneWithChulHo:                "SH_COMMUNE_WITH_CHUL_HO",
	SkillShChulHoSonicClaw:                  "SH_CHUL_HO_SONIC_CLAW",
	SkillShHowlingOfChulHo:                  "SH_HOWLING_OF_CHUL_HO",
	SkillShHogogongStrike:                   "SH_HOGOGONG_STRIKE",
	SkillShCommuneWithKiSul:                 "SH_COMMUNE_WITH_KI_SUL",
	SkillShKiSulWaterSpraying:               "SH_KI_SUL_WATER_SPRAYING",
	SkillShMarineFestivalOfKiSul:            "SH_MARINE_FESTIVAL_OF_KI_SUL",
	SkillShSandyFestivalOfKiSul:             "SH_SANDY_FESTIVAL_OF_KI_SUL",
	SkillShKiSulRampage:                     "SH_KI_SUL_RAMPAGE",
	SkillShCommuneWithHyunRok:               "SH_COMMUNE_WITH_HYUN_ROK",
	SkillShColorsOfHyunRok:                  "SH_COLORS_OF_HYUN_ROK",
	SkillShHyunRoksBreeze:                   "SH_HYUN_ROKS_BREEZE",
	SkillShHyunRokCannon:                    "SH_HYUN_ROK_CANNON",
	SkillShTemporaryCommunion:               "SH_TEMPORARY_COMMUNION",
	SkillShBlessingOfMysticalCreatures:      "SH_BLESSING_OF_MYSTICAL_CREATURES",
	SkillHnSelfstudyTatics:                  "HN_SELFSTUDY_TATICS",
	SkillHnSelfstudySocery:                  "HN_SELFSTUDY_SOCERY",
	SkillHnDoublebowlingbash:                "HN_DOUBLEBOWLINGBASH",
	SkillHnMegaSonicBlow:                    "HN_MEGA_SONIC_BLOW",
	SkillHnShieldChainRush:                  "HN_SHIELD_CHAIN_RUSH",
	SkillHnSpiralPierceMax:                  "HN_SPIRAL_PIERCE_MAX",
	SkillHnMeteorStormBuster:                "HN_METEOR_STORM_BUSTER",
	SkillHnJupitelThunderStorm:              "HN_JUPITEL_THUNDER_STORM",
	SkillHnJackFrostNova:                    "HN_JACK_FROST_NOVA",
	SkillHnHellsDrive:                       "HN_HELLS_DRIVE",
	SkillHnGroundGravitation:                "HN_GROUND_GRAVITATION",
	SkillHnNapalmVulcanStrike:               "HN_NAPALM_VULCAN_STRIKE",
	SkillHnBreakinglimit:                    "HN_BREAKINGLIMIT",
	SkillHnRulebreak:                        "HN_RULEBREAK",
	SkillSkeSkyMastery:                      "SKE_SKY_MASTERY",
	SkillSkeWarBookMastery:                  "SKE_WAR_BOOK_MASTERY",
	SkillSkeRisingSun:                       "SKE_RISING_SUN",
	SkillSkeNoonBlast:                       "SKE_NOON_BLAST",
	SkillSkeSunsetBlast:                     "SKE_SUNSET_BLAST",
	SkillSkeRisingMoon:                      "SKE_RISING_MOON",
	SkillSkeMidnightKick:                    "SKE_MIDNIGHT_KICK",
	SkillSkeDawnBreak:                       "SKE_DAWN_BREAK",
	SkillSkeTwinklingGalaxy:                 "SKE_TWINKLING_GALAXY",
	SkillSkeStarBurst:                       "SKE_STAR_BURST",
	SkillSkeStarCannon:                      "SKE_STAR_CANNON",
	SkillSkeALLInTheSky:                     "SKE_ALL_IN_THE_SKY",
	SkillSkeEnchantingSky:                   "SKE_ENCHANTING_SKY",
	SkillSsTokedasu:                         "SS_TOKEDASU",
	SkillSsShimiru:                          "SS_SHIMIRU",
	SkillSsAkumukesu:                        "SS_AKUMUKESU",
	SkillSsShinkirou:                        "SS_SHINKIROU",
	SkillSsKagegari:                         "SS_KAGEGARI",
	SkillSsKagenomai:                        "SS_KAGENOMAI",
	SkillSsKagegissen:                       "SS_KAGEGISSEN",
	SkillSsFuumashouaku:                     "SS_FUUMASHOUAKU",
	SkillSsFuumakouchiku:                    "SS_FUUMAKOUCHIKU",
	SkillSsKunaiwaikyoku:                    "SS_KUNAIWAIKYOKU",
	SkillSsKunaikaiten:                      "SS_KUNAIKAITEN",
	SkillSsKunaikussetsu:                    "SS_KUNAIKUSSETSU",
	SkillSsSekienhou:                        "SS_SEKIENHOU",
	SkillSsReiketsuhou:                      "SS_REIKETSUHOU",
	SkillSsRaidenpou:                        "SS_RAIDENPOU",
	SkillSsKinryuuhou:                       "SS_KINRYUUHOU",
	SkillSsAntenpou:                         "SS_ANTENPOU",
	SkillSsKageakumu:                        "SS_KAGEAKUMU",
	SkillSsHitouakumu:                       "SS_HITOUAKUMU",
	SkillSsAnkokuryuuakumu:                  "SS_ANKOKURYUUAKUMU",
	SkillNwTheVigilanteAtNightGunGatling:    "NW_THE_VIGILANTE_AT_NIGHT_GUN_GATLING",
	SkillNwTheVigilanteAtNightGunShotgun:    "NW_THE_VIGILANTE_AT_NIGHT_GUN_SHOTGUN",
	SkillDkDragonicBreath:                   "DK_DRAGONIC_BREATH",
	SkillMtSparkBlaster:                     "MT_SPARK_BLASTER",
	SkillMtTripleLaser:                      "MT_TRIPLE_LASER",
	SkillMtMightySmash:                      "MT_MIGHTY_SMASH",
	SkillBoExplosivePowder:                  "BO_EXPLOSIVE_POWDER",
	SkillBoMayhemicThorns:                   "BO_MAYHEMIC_THORNS",
	SkillEmElFlametechnic:                   "EM_EL_FLAMETECHNIC",
	SkillEmElFlamearmor:                     "EM_EL_FLAMEARMOR",
	SkillEmElFlamerock:                      "EM_EL_FLAMEROCK",
	SkillEmElColdForce:                      "EM_EL_COLD_FORCE",
	SkillEmElCrystalArmor:                   "EM_EL_CRYSTAL_ARMOR",
	SkillEmElAgeOfIce:                       "EM_EL_AGE_OF_ICE",
	SkillEmElGraceBreeze:                    "EM_EL_GRACE_BREEZE",
	SkillEmElEyesOfStorm:                    "EM_EL_EYES_OF_STORM",
	SkillEmElStormWind:                      "EM_EL_STORM_WIND",
	SkillEmElEarthCare:                      "EM_EL_EARTH_CARE",
	SkillEmElStrongProtection:               "EM_EL_STRONG_PROTECTION",
	SkillEmElAvalanche:                      "EM_EL_AVALANCHE",
	SkillEmElDeepPoisoning:                  "EM_EL_DEEP_POISONING",
	SkillEmElPoisonShield:                   "EM_EL_POISON_SHIELD",
	SkillEmElDeadlyPoison:                   "EM_EL_DEADLY_POISON",
	SkillHomunBegin:                         "HOMUN_BEGIN",
	SkillHlifHeal:                           "HLIF_HEAL",
	SkillHlifAvoid:                          "HLIF_AVOID",
	SkillHlifBrain:                          "HLIF_BRAIN",
	SkillHlifChange:                         "HLIF_CHANGE",
	SkillHamiCastle:                         "HAMI_CASTLE",
	SkillHamiDefence:                        "HAMI_DEFENCE",
	SkillHamiSkin:                           "HAMI_SKIN",
	SkillHamiBloodlust:                      "HAMI_BLOODLUST",
	SkillHfliMoon:                           "HFLI_MOON",
	SkillHfliFleet:                          "HFLI_FLEET",
	SkillHfliSpeed:                          "HFLI_SPEED",
	SkillHfliSbr44:                          "HFLI_SBR44",
	SkillHvanCaprice:                        "HVAN_CAPRICE",
	SkillHvanChaotic:                        "HVAN_CHAOTIC",
	SkillHvanInstruct:                       "HVAN_INSTRUCT",
	SkillHvanExplosion:                      "HVAN_EXPLOSION",
	SkillMutationBasejob:                    "MUTATION_BASEJOB",
	SkillMhSummonLegion:                     "MH_SUMMON_LEGION",
	SkillMhNeedleOfParalyze:                 "MH_NEEDLE_OF_PARALYZE",
	SkillMhPoisonMist:                       "MH_POISON_MIST",
	SkillMhPainKiller:                       "MH_PAIN_KILLER",
	SkillMhLightOfRegene:                    "MH_LIGHT_OF_REGENE",
	SkillMhOveredBoost:                      "MH_OVERED_BOOST",
	SkillMhEraserCutter:                     "MH_ERASER_CUTTER",
	SkillMhXenoSlasher:                      "MH_XENO_SLASHER",
	SkillMhSilentBreeze:                     "MH_SILENT_BREEZE",
	SkillMhStyleChange:                      "MH_STYLE_CHANGE",
	SkillMhSonicCraw:                        "MH_SONIC_CRAW",
	SkillMhSilverveinRush:                   "MH_SILVERVEIN_RUSH",
	SkillMhMidnightFrenzy:                   "MH_MIDNIGHT_FRENZY",
	SkillMhStahlHorn:                        "MH_STAHL_HORN",
	SkillMhGoldeneFerse:                     "MH_GOLDENE_FERSE",
	SkillMhSteinwand:                        "MH_STEINWAND",
	SkillMhHeiligeStange:                    "MH_HEILIGE_STANGE",
	SkillMhAngriffsModus:                    "MH_ANGRIFFS_MODUS",
	SkillMhTinderBreaker:                    "MH_TINDER_BREAKER",
	SkillMhCbc:                              "MH_CBC",
	SkillMhEqc:                              "MH_EQC",
	SkillMhMagmaFlow:                        "MH_MAGMA_FLOW",
	SkillMhGraniticArmor:                    "MH_GRANITIC_ARMOR",
	SkillMhLavaSlide:                        "MH_LAVA_SLIDE",
	SkillMhPyroclastic:                      "MH_PYROCLASTIC",
	SkillMhVolcanicAsh:                      "MH_VOLCANIC_ASH",
	SkillMhBlastForge:                       "MH_BLAST_FORGE",
	SkillMhTempering:                        "MH_TEMPERING",
	SkillMhClassyFlutter:                    "MH_CLASSY_FLUTTER",
	SkillMhTwisterCutter:                    "MH_TWISTER_CUTTER",
	SkillMhAbsoluteZephyr:                   "MH_ABSOLUTE_ZEPHYR",
	SkillMhBrushupClaw:                      "MH_BRUSHUP_CLAW",
	SkillMhBlazingAndFurious:                "MH_BLAZING_AND_FURIOUS",
	SkillMhTheOneFighterRises:               "MH_THE_ONE_FIGHTER_RISES",
	SkillMhPolishingNeedle:                  "MH_POLISHING_NEEDLE",
	SkillMhToxinOfMandara:                   "MH_TOXIN_OF_MANDARA",
	SkillMhNeedleStinger:                    "MH_NEEDLE_STINGER",
	SkillMhLichtGehorn:                      "MH_LICHT_GEHORN",
	SkillMhGlanzenSpies:                     "MH_GLANZEN_SPIES",
	SkillMhHeiligePferd:                     "MH_HEILIGE_PFERD",
	SkillMhGoldeneTone:                      "MH_GOLDENE_TONE",
	SkillMhBlazingLava:                      "MH_BLAZING_LAVA",
	SkillMhLast:                             "MH_LAST",
	SkillHomunLast:                          "HOMUN_LAST",
	SkillMercenaryBegin:                     "MERCENARY_BEGIN",
	SkillMsBash:                             "MS_BASH",
	SkillMsMagnum:                           "MS_MAGNUM",
	SkillMsBowlingbash:                      "MS_BOWLINGBASH",
	SkillMsParrying:                         "MS_PARRYING",
	SkillMsReflectshield:                    "MS_REFLECTSHIELD",
	SkillMsBerserk:                          "MS_BERSERK",
	SkillMaDouble:                           "MA_DOUBLE",
	SkillMaShower:                           "MA_SHOWER",
	SkillMaSkidtrap:                         "MA_SKIDTRAP",
	SkillMaLandmine:                         "MA_LANDMINE",
	SkillMaSandman:                          "MA_SANDMAN",
	SkillMaFreezingtrap:                     "MA_FREEZINGTRAP",
	SkillMaRemovetrap:                       "MA_REMOVETRAP",
	SkillMaChargearrow:                      "MA_CHARGEARROW",
	SkillMaSharpshooting:                    "MA_SHARPSHOOTING",
	SkillMlPierce:                           "ML_PIERCE",
	SkillMlBrandish:                         "ML_BRANDISH",
	SkillMlSpiralpierce:                     "ML_SPIRALPIERCE",
	SkillMlDefender:                         "ML_DEFENDER",
	SkillMlAutoguard:                        "ML_AUTOGUARD",
	SkillMlDevotion:                         "ML_DEVOTION",
	SkillMerMagnificat:                      "MER_MAGNIFICAT",
	SkillMerQuicken:                         "MER_QUICKEN",
	SkillMerSight:                           "MER_SIGHT",
	SkillMerCrash:                           "MER_CRASH",
	SkillMerRegain:                          "MER_REGAIN",
	SkillMerTender:                          "MER_TENDER",
	SkillMerBenediction:                     "MER_BENEDICTION",
	SkillMerRecuperate:                      "MER_RECUPERATE",
	SkillMerMentalcure:                      "MER_MENTALCURE",
	SkillMerCompress:                        "MER_COMPRESS",
	SkillMerProvoke:                         "MER_PROVOKE",
	SkillMerAutoberserk:                     "MER_AUTOBERSERK",
	SkillMerDecagi:                          "MER_DECAGI",
	SkillMerScapegoat:                       "MER_SCAPEGOAT",
	SkillMerLexdivina:                       "MER_LEXDIVINA",
	SkillMerEstimation:                      "MER_ESTIMATION",
	SkillMerKyrie:                           "MER_KYRIE",
	SkillMerBlessing:                        "MER_BLESSING",
	SkillMerIncagi:                          "MER_INCAGI",
	SkillMerInvincibleoff2:                  "MER_INVINCIBLEOFF2",
	SkillMercenaryLast:                      "MERCENARY_LAST",
	SkillElementalBegin:                     "ELEMENTAL_BEGIN",
	SkillElCircleOfFire:                     "EL_CIRCLE_OF_FIRE",
	SkillElFireCloak:                        "EL_FIRE_CLOAK",
	SkillElFireMantle:                       "EL_FIRE_MANTLE",
	SkillElWaterScreen:                      "EL_WATER_SCREEN",
	SkillElWaterDrop:                        "EL_WATER_DROP",
	SkillElWaterBarrier:                     "EL_WATER_BARRIER",
	SkillElWindStep:                         "EL_WIND_STEP",
	SkillElWindCurtain:                      "EL_WIND_CURTAIN",
	SkillElZephyr:                           "EL_ZEPHYR",
	SkillElSolidSkin:                        "EL_SOLID_SKIN",
	SkillElStoneShield:                      "EL_STONE_SHIELD",
	SkillElPowerOfGaia:                      "EL_POWER_OF_GAIA",
	SkillElPyrotechnic:                      "EL_PYROTECHNIC",
	SkillElHeater:                           "EL_HEATER",
	SkillElTropic:                           "EL_TROPIC",
	SkillElAquaplay:                         "EL_AQUAPLAY",
	SkillElCooler:                           "EL_COOLER",
	SkillElChillyAir:                        "EL_CHILLY_AIR",
	SkillElGust:                             "EL_GUST",
	SkillElBlast:                            "EL_BLAST",
	SkillElWildStorm:                        "EL_WILD_STORM",
	SkillElPetrology:                        "EL_PETROLOGY",
	SkillElCursedSoil:                       "EL_CURSED_SOIL",
	SkillElUpheaval:                         "EL_UPHEAVAL",
	SkillElFireArrow:                        "EL_FIRE_ARROW",
	SkillElFireBomb:                         "EL_FIRE_BOMB",
	SkillElFireBombAtk:                      "EL_FIRE_BOMB_ATK",
	SkillElFireWave:                         "EL_FIRE_WAVE",
	SkillElFireWaveAtk:                      "EL_FIRE_WAVE_ATK",
	SkillElIceNeedle:                        "EL_ICE_NEEDLE",
	SkillElWaterScrew:                       "EL_WATER_SCREW",
	SkillElWaterScrewAtk:                    "EL_WATER_SCREW_ATK",
	SkillElTidalWeapon:                      "EL_TIDAL_WEAPON",
	SkillElWindSlash:                        "EL_WIND_SLASH",
	SkillElHurricane:                        "EL_HURRICANE",
	SkillElHurricaneAtk:                     "EL_HURRICANE_ATK",
	SkillElTypoonMis:                        "EL_TYPOON_MIS",
	SkillElTypoonMisAtk:                     "EL_TYPOON_MIS_ATK",
	SkillElStoneHammer:                      "EL_STONE_HAMMER",
	SkillElRockCrusher:                      "EL_ROCK_CRUSHER",
	SkillElRockCrusherAtk:                   "EL_ROCK_CRUSHER_ATK",
	SkillElStoneRain:                        "EL_STONE_RAIN",
	SkillFollowerNPCReset:                   "FOLLOWER_NPC_RESET",
	SkillGdApproval:                         "GD_APPROVAL",
	SkillGdKafracontract:                    "GD_KAFRACONTRACT",
	SkillGdGuardresearch:                    "GD_GUARDRESEARCH",
	SkillGdGuardup:                          "GD_GUARDUP",
	SkillGdExtension:                        "GD_EXTENSION",
	SkillGdGloryguild:                       "GD_GLORYGUILD",
	SkillGdLeadership:                       "GD_LEADERSHIP",
	SkillGdGlorywounds:                      "GD_GLORYWOUNDS",
	SkillGdSoulcold:                         "GD_SOULCOLD",
	SkillGdHawkeyes:                         "GD_HAWKEYES",
	SkillGdBattleorder:                      "GD_BATTLEORDER",
	SkillGdRegeneration:                     "GD_REGENERATION",
	SkillGdRestore:                          "GD_RESTORE",
	SkillGdEmergencycall:                    "GD_EMERGENCYCALL",
	SkillGdDevelopment:                      "GD_DEVELOPMENT",
	SkillGdItememergencycall:                "GD_ITEMEMERGENCYCALL",
	SkillGdGuildStorage:                     "GD_GUILD_STORAGE",
	SkillGdChargeshoutFlag:                  "GD_CHARGESHOUT_FLAG",
	SkillGdChargeshoutBeating:               "GD_CHARGESHOUT_BEATING",
	SkillGdEmergencyMove:                    "GD_EMERGENCY_MOVE",
	SkillGdLast:                             "GD_LAST",
	SkillSysFirstjoblv:                      "SYS_FIRSTJOBLV",
	SkillSysSecondjoblv:                     "SYS_SECONDJOBLV",
	SkillScript000:                          "SCRIPT_000",
	SkillItemCocktailWargBlood:              "ITEM_COCKTAIL_WARG_BLOOD",
	SkillItemMinorBbq:                       "ITEM_MINOR_BBQ",
	SkillItemSiromaIceTea:                   "ITEM_SIROMA_ICE_TEA",
	SkillItemDroceraHerbSteamed:             "ITEM_DROCERA_HERB_STEAMED",
	SkillItemPuttiTailsNoodles:              "ITEM_PUTTI_TAILS_NOODLES",
	SkillItemBananaBomb:                     "ITEM_BANANA_BOMB",
	SkillScript999:                          "SCRIPT_999",
	SkillEfstDressUp:                        "EFST_DRESS_UP",
}

type SkillEffectSpec struct {
	EffectIDs              []int
	EffectIDsOnCaster      []int
	GroundEffectIDs        []int
	HitEffectIDs           []int
	HitEffectIDsOnCaster   []int
	BeforeHitEffectIDs     []int
	BeforeHitEffectIDsSelf []int
	BeginCastEffectIDs     []int
	SuccessEffectIDs       []int
	SuccessEffectIDsSelf   []int
	HideCastBar            bool
	HideCastAura           bool
}

// SkillGroundCastSizes mirrors roBrowser Renderer/Effects/MagicTarget.js
// CastSize entries. Values are the MagicTarget plane sizes by skill level.
// roBrowser falls back to the first entry when the caster level is unknown.
var SkillGroundCastSizes = map[uint16][]float64{
	SkillABEpiclesis:          {5},
	SkillACShower:             {3, 3, 3, 3, 3, 5, 5, 5, 5, 5},
	SkillALPneuma:             {3},
	SkillAMDemonstration:      {3},
	SkillASVenomdust:          {2},
	SkillCRSlimpitcher:        {7},
	SkillGCPoisonsmoke:        {5},
	SkillGNCrazyweed:          {9},
	SkillGNDemonicFire:        {5},
	SkillGNFireExpansion:      {5},
	SkillGNHellsPlant:         {3},
	SkillHTClaymoretrap:       {5},
	SkillHTDetecting:          {3},
	SkillHTFlasher:            {3},
	SkillHWGanbantein:         {3},
	SkillHWGravitation:        {5},
	SkillKOBakuretsu:          {3},
	SkillKOHuumaranka:         {7},
	SkillKOMuchanage:          {3, 3, 3, 3, 3, 3, 3, 3, 5},
	SkillKOZenkai:             {5},
	SkillLGOverbrand:          {3},
	SkillLGRayofgenesis:       {11},
	SkillMGFirewall:           {1},
	SkillMGThunderstorm:       {5},
	SkillMhLavaSlide:          {3, 3, 3, 5, 5, 5, 7, 7, 7, 9},
	SkillMhPoisonMist:         {7},
	SkillMhVolcanicAsh:        {3},
	SkillMhXenoSlasher:        {3, 3, 3, 5, 5, 5, 7, 7, 7, 9},
	SkillNCArmscannon:         {7, 5, 3},
	SkillNCColdslower:         {5, 7, 9},
	SkillNCMagmaEruption:      {7},
	SkillNJRaigekisai:         {3, 3, 5, 5, 7},
	SkillNJSuiton:             {3, 3, 3, 5, 5, 5, 7, 7, 7, 9},
	SkillNPCCloudKill:         {7},
	SkillNPCComet:             {19},
	SkillNPCEvilland:          {11},
	SkillNPCMagmaEruption:     {7},
	SkillNPCPsychicWave:       {7, 7, 9, 9, 11},
	SkillNPCRayofgenesis:      {11},
	SkillPFFogwall:            {3},
	SkillPRBenedictio:         {3},
	SkillPRMagnus:             {7},
	SkillPRSanctuary:          {5},
	SkillRAFiringtrap:         {5},
	SkillRGGraffiti:           {5},
	SkillRKDragonbreath:       {3, 3, 3, 5, 5, 5, 7, 7, 9, 9},
	SkillRKDragonbreathWater:  {3, 3, 3, 5, 5, 5, 7, 7, 9, 9},
	SkillRLHammerOfGod:        {5},
	SkillSADeluge:             {7},
	SkillSALandprotector:      {7, 7, 9, 9, 11},
	SkillSAViolentgale:        {7},
	SkillSAVolcano:            {7},
	SkillSCBloodylust:         {7},
	SkillSCChaospanic:         {5},
	SkillSCDimensiondoor:      {1},
	SkillSCMaelstrom:          {5},
	SkillSCManhole:            {3},
	SkillSOArrullo:            {3, 3, 5, 5, 7},
	SkillSOCloudKill:          {7},
	SkillSODiamonddust:        {7, 7, 7, 9, 9},
	SkillSOEarthgrave:         {7, 7, 7, 9, 9},
	SkillSOEarthInsignia:      {3},
	SkillSOFireInsignia:       {3},
	SkillSOPsychicWave:        {7, 7, 9, 9, 11},
	SkillSOVacuumExtreme:      {3, 3, 5, 5, 7},
	SkillSOWarmer:             {7},
	SkillSOWaterInsignia:      {3},
	SkillSOWindInsignia:       {3},
	SkillSRRideinlightning:    {3, 3, 5, 5, 7},
	SkillWLComet:              {19},
	SkillWLCrimsonrock:        {7},
	SkillWmPoemofnetherworld:  {3},
	SkillWmSevereRainstorm:    {11},
	SkillWmSoundOfDestruction: {9, 9, 11, 13, 15},
	SkillWZHeavendrive:        {5},
	SkillWZMeteor:             {7},
	SkillWZQuagmire:           {5},
	SkillWZStormgust:          {11},
	SkillWZVermilion:          {11},
}

func SkillGroundCastSize(skillID uint16, level int) float64 {
	sizes := SkillGroundCastSizes[skillID]
	if len(sizes) == 0 {
		return 1
	}
	if level > 0 && level <= len(sizes) && sizes[level-1] > 0 {
		return sizes[level-1]
	}
	if sizes[0] > 0 {
		return sizes[0]
	}
	return 1
}

const (
	// Synthetic numeric aliases for robr EffectTable.js string keys referenced
	// from SkillEffect.js. These are not Ragnarok packet effect IDs.
	SkillEffectColdBolt          = 10014
	SkillEffectFireBolt          = 10019
	SkillEffectQuakeMagnum       = 10022
	SkillEffectArrowShot         = 10060
	SkillEffectArrowShower       = 10061
	SkillEffectMagicPower        = 10366
	SkillEffectGravitationGround = 10484
	SkillEffectWhitePulse        = 11000
	SkillEffectSpearProjectile   = 11001
	SkillEffectSpiralBeforeCast  = 11002
	SkillEffectSpearHitSound     = 11003
	SkillEffectEnemyHitNormal1   = 11004
	SkillEffectQuake             = 11005
	SkillEffectAnkleSnareGround  = 11006
	SkillEffectSharpShootingCast = 11007
	SkillEffectAdrenalineCast    = 11008
	SkillEffectMaximizeSounds    = 11009
	SkillEffectGreedSound        = 11010
	SkillEffectGospelGround      = 11011
	SkillEffectShieldProjectile  = 11012
	SkillEffectFogWallGround     = 11013
	SkillEffectHermodeMusic      = 11014
)

var SkillEffects = map[uint16]SkillEffectSpec{
	SkillSMBash:                     {HitEffectIDs: []int{1}, BeginCastEffectIDs: []int{16}},
	SkillSMProvoke:                  {SuccessEffectIDs: []int{67}},
	SkillSMMagnum:                   {EffectIDs: []int{SkillEffectQuakeMagnum}, EffectIDsOnCaster: []int{17}},
	SkillSMEndure:                   {EffectIDs: []int{11}},
	SkillMGNapalmbeat:               {HitEffectIDs: []int{1}},
	SkillMGSoulstrike:               {HitEffectIDs: []int{1}, BeforeHitEffectIDs: []int{15}},
	SkillMGColdbolt:                 {HitEffectIDs: []int{51}, BeforeHitEffectIDs: []int{SkillEffectColdBolt}},
	SkillMGFrostdiver:               {EffectIDs: []int{27}, HitEffectIDs: []int{28}},
	SkillMGStonecurse:               {EffectIDs: []int{23}},
	SkillMGFireball:                 {HitEffectIDs: []int{49}, BeforeHitEffectIDs: []int{24}},
	SkillMGFirewall:                 {GroundEffectIDs: []int{25}, HitEffectIDs: []int{49}},
	SkillMGFirebolt:                 {HitEffectIDs: []int{49}, BeforeHitEffectIDs: []int{SkillEffectFireBolt}},
	SkillMGLightningbolt:            {EffectIDs: []int{29}, HitEffectIDs: []int{52}},
	SkillMGThunderstorm:             {EffectIDs: []int{30}, HitEffectIDs: []int{52}},
	SkillALRuwach:                   {HitEffectIDs: []int{1}},
	SkillALPneuma:                   {GroundEffectIDs: []int{141}},
	SkillALHeal:                     {EffectIDs: []int{312}, HitEffectIDs: []int{320}},
	SkillALIncagi:                   {EffectIDs: []int{37}},
	SkillALDecagi:                   {EffectIDs: []int{38}},
	SkillALHolywater:                {EffectIDs: []int{39}},
	SkillALCrucis:                   {EffectIDs: []int{40}},
	SkillALAngelus:                  {EffectIDs: []int{41}},
	SkillALBlessing:                 {EffectIDs: []int{42}},
	SkillALCure:                     {EffectIDs: []int{66}},
	SkillMCIdentify:                 {},
	SkillMCVending:                  {},
	SkillMCMammonite:                {EffectIDs: []int{10}},
	SkillACConcentration:            {EffectIDs: []int{153}},
	SkillACDouble:                   {HitEffectIDs: []int{1}, BeginCastEffectIDs: []int{16}, BeforeHitEffectIDs: []int{SkillEffectArrowShot}},
	SkillACShower:                   {HitEffectIDs: []int{1}, EffectIDs: []int{SkillEffectArrowShower}},
	SkillACMakingarrow:              {},
	SkillTFSteal:                    {SuccessEffectIDs: []int{18}},
	SkillTFHiding:                   {},
	SkillTFPoison:                   {HitEffectIDs: []int{20}},
	SkillTFDetoxify:                 {EffectIDs: []int{21}},
	SkillALLResurrection:            {EffectIDs: []int{77, 140}},
	SkillKNPierce:                   {EffectIDsOnCaster: []int{148}, HitEffectIDs: []int{147}},
	SkillKNBrandishspear:            {EffectIDs: []int{70}, EffectIDsOnCaster: []int{144}, HideCastBar: true, HideCastAura: true},
	SkillKNSpearstab:                {EffectIDsOnCaster: []int{effectSpearStabSelf}},
	SkillKNSpearboomerang:           {EffectIDsOnCaster: []int{effectSpearBmrSelf}, BeforeHitEffectIDs: []int{SkillEffectSpearProjectile}, HitEffectIDs: []int{effectSpearBoomerang}},
	SkillKNTwohandquicken:           {EffectIDs: []int{effectTwoHandQuicken}},
	SkillKNAutocounter:              {HideCastAura: true},
	SkillKNBowlingbash:              {EffectIDsOnCaster: []int{149}, HitEffectIDs: []int{1}, HideCastBar: true, HideCastAura: true},
	SkillKNOnehand:                  {EffectIDs: []int{effectTwoHandQuicken}},
	SkillKNChargeatk:                {BeginCastEffectIDs: []int{SkillEffectWhitePulse}, HitEffectIDs: []int{SkillEffectEnemyHitNormal1}},
	SkillPRImpositio:                {EffectIDs: []int{84}},
	SkillPRSuffragium:               {EffectIDs: []int{88}},
	SkillPRAspersio:                 {EffectIDs: []int{86}},
	SkillPRBenedictio:               {EffectIDs: []int{91}},
	SkillPRSanctuary:                {EffectIDs: []int{effectSanctuary}, GroundEffectIDs: []int{effectBottomSanc}},
	SkillPRSlowpoison:               {EffectIDs: []int{136}},
	SkillPRStrecovery:               {EffectIDs: []int{78}},
	SkillPRKyrie:                    {EffectIDs: []int{effectKyrie}},
	SkillPRMagnificat:               {EffectIDs: []int{effectMagnificat}},
	SkillPRGloria:                   {EffectIDs: []int{effectGloria}},
	SkillPRLexdivina:                {EffectIDs: []int{effectLexDivina}},
	SkillPRTurnundead:               {HitEffectIDs: []int{effectHolyLight}},
	SkillPRLexaeterna:               {EffectIDs: []int{85}},
	SkillPRMagnus:                   {EffectIDs: []int{effectMagnus}, GroundEffectIDs: []int{effectBottomMagnus}, HitEffectIDs: []int{effectHolyLight}},
	SkillPRRedemptio:                {},
	SkillWZFirepillar:               {EffectIDs: []int{96}, GroundEffectIDs: []int{138}, HitEffectIDs: []int{97}},
	SkillWZSightrasher:              {EffectIDs: []int{62}, HitEffectIDs: []int{49}},
	SkillWZMeteor:                   {EffectIDs: []int{92}, HitEffectIDs: []int{49}},
	SkillWZJupitel:                  {EffectIDs: []int{93}, BeforeHitEffectIDs: []int{94}},
	SkillWZVermilion:                {EffectIDs: []int{90}, HitEffectIDs: []int{52}},
	SkillWZWaterball:                {HitEffectIDsOnCaster: []int{117}, BeforeHitEffectIDsSelf: []int{116}},
	SkillWZIcewall:                  {GroundEffectIDs: []int{74}},
	SkillWZFrostnova:                {EffectIDsOnCaster: []int{28}, HitEffectIDs: []int{51}},
	SkillWZStormgust:                {EffectIDs: []int{89}, HitEffectIDs: []int{51}},
	SkillWZEarthspike:               {EffectIDs: []int{79}, HitEffectIDs: []int{147}},
	SkillWZHeavendrive:              {EffectIDs: []int{142}, HitEffectIDs: []int{147}},
	SkillWZQuagmire:                 {GroundEffectIDs: []int{95}},
	SkillBSRepairweapon:             {EffectIDs: []int{101}},
	SkillBSHammerfall:               {EffectIDs: []int{102}},
	SkillBSAdrenaline:               {EffectIDs: []int{98}, BeginCastEffectIDs: []int{SkillEffectAdrenalineCast}},
	SkillBSWeaponperfect:            {EffectIDs: []int{103}},
	SkillBSOverthrust:               {EffectIDs: []int{128}},
	SkillBSMaximize:                 {EffectIDs: []int{104}, BeginCastEffectIDs: []int{SkillEffectMaximizeSounds}},
	SkillHTSkidtrap:                 {EffectIDs: []int{69}},
	SkillHTLandmine:                 {},
	SkillHTAnklesnare:               {GroundEffectIDs: []int{SkillEffectAnkleSnareGround}},
	SkillHTShockwave:                {EffectIDs: []int{145}, HitEffectIDs: []int{146}},
	SkillHTSandman:                  {HitEffectIDs: []int{139}},
	SkillHTFlasher:                  {HitEffectIDs: []int{99}},
	SkillHTFreezingtrap:             {HitEffectIDs: []int{108}},
	SkillHTBlastmine:                {HitEffectIDs: []int{106}},
	SkillHTClaymoretrap:             {HitEffectIDs: []int{107}},
	SkillHTRemovetrap:               {EffectIDs: []int{100}},
	SkillHTBlitzbeat:                {EffectIDs: []int{115}},
	SkillHTDetecting:                {EffectIDs: []int{119}},
	SkillHTSpringtrap:               {EffectIDs: []int{111}},
	SkillHTTalkiebox:                {},
	SkillHTPower:                    {},
	SkillASCloaking:                 {EffectIDs: []int{120}},
	SkillASSonicblow:                {EffectIDs: []int{143}, EffectIDsOnCaster: []int{121}, HitEffectIDs: []int{122}},
	SkillASGrimtooth:                {EffectIDs: []int{123}, HitEffectIDs: []int{132}},
	SkillASEnchantpoison:            {EffectIDs: []int{20}},
	SkillASPoisonreact:              {EffectIDs: []int{126}, HitEffectIDs: []int{127}},
	SkillASVenomdust:                {EffectIDs: []int{124}, GroundEffectIDs: []int{effectVenomDust2}},
	SkillASSplasher:                 {EffectIDs: []int{129}},
	SkillNVFirstaid:                 {EffectIDs: []int{effectFirstAid}},
	SkillACChargearrow:              {BeforeHitEffectIDs: []int{SkillEffectArrowShot}, HideCastAura: true},
	SkillTFSprinklesand:             {EffectIDs: []int{310}},
	SkillTFBacksliding:              {},
	SkillTFPickstone:                {HideCastAura: true},
	SkillTFThrowstone:               {BeforeHitEffectIDs: []int{effectThrowItem3}},
	SkillMCCartrevolution:           {HitEffectIDs: []int{effectCartRevolution}, BeginCastEffectIDs: []int{effectCartRevolution}},
	SkillMCChangecart:               {},
	SkillMCLoud:                     {EffectIDs: []int{effectLoud}},
	SkillMCCartdecorate:             {},
	SkillALHolylight:                {EffectIDs: []int{152}},
	SkillMGEnergycoat:               {EffectIDs: []int{169}},
	SkillNPCPiercingatt:             {EffectIDsOnCaster: []int{148}},
	SkillNPCMentalbreaker:           {EffectIDs: []int{181}},
	SkillNPCChangewater:             {EffectIDs: []int{174}},
	SkillNPCChangeground:            {EffectIDs: []int{177}},
	SkillNPCChangefire:              {EffectIDs: []int{173}},
	SkillNPCChangewind:              {EffectIDs: []int{175}},
	SkillNPCChangepoison:            {EffectIDs: []int{179}},
	SkillNPCChangeholy:              {EffectIDs: []int{178}},
	SkillNPCChangedarkness:          {EffectIDs: []int{172}},
	SkillNPCCriticalslash:           {EffectIDsOnCaster: []int{16}},
	SkillNPCGuidedattack:            {EffectIDs: []int{191}},
	SkillNPCSelfdestruction:         {EffectIDs: []int{183}},
	SkillNPCSuicide:                 {EffectIDs: []int{185}},
	SkillNPCPoison:                  {EffectIDs: []int{192}},
	SkillNPCSilenceattack:           {EffectIDs: []int{193}},
	SkillNPCStunattack:              {EffectIDs: []int{194}},
	SkillNPCPetrifyattack:           {EffectIDs: []int{195}},
	SkillNPCCurseattack:             {EffectIDs: []int{196}},
	SkillNPCSleepattack:             {EffectIDs: []int{197}},
	SkillNPCWaterattack:             {HitEffectIDs: []int{51}},
	SkillNPCGroundattack:            {HitEffectIDs: []int{147}},
	SkillNPCFireattack:              {HitEffectIDs: []int{49}},
	SkillNPCWindattack:              {HitEffectIDs: []int{52}},
	SkillNPCPoisonattack:            {HitEffectIDs: []int{53}},
	SkillNPCHolyattack:              {HitEffectIDs: []int{effectHolyLight}},
	SkillNPCDarknessattack:          {EffectIDs: []int{184}, HitEffectIDs: []int{180}},
	SkillNPCTelekinesisattack:       {EffectIDs: []int{198}},
	SkillNPCMagicalattack:           {HitEffectIDs: []int{effectMagicalAtkHit}},
	SkillNPCProvocation:             {SuccessEffectIDs: []int{67}},
	SkillNPCBlooddrain:              {EffectIDsOnCaster: []int{effectBloodDrain}},
	SkillNPCEnergydrain:             {EffectIDsOnCaster: []int{effectEnergyDrain}},
	SkillNPCKeeping:                 {EffectIDs: []int{effectKeeping}},
	SkillNPCDarkbreath:              {EffectIDs: []int{effectDarkBreath}},
	SkillNPCDefender:                {EffectIDs: []int{effectDefender}},
	SkillRGStealcoin:                {SuccessEffectIDs: []int{effectStealCoin, effectRogueCoin}},
	SkillRGBackstap:                 {HitEffectIDs: []int{effectBackStab}},
	SkillRGRaid:                     {EffectIDsOnCaster: []int{effectTeiHit3}},
	SkillRGStripweapon:              {SuccessEffectIDs: []int{effectStripWeapon}},
	SkillRGStripshield:              {SuccessEffectIDs: []int{effectStripShield}},
	SkillRGStriparmor:               {SuccessEffectIDs: []int{effectStripArmor}},
	SkillRGStriphelm:                {SuccessEffectIDs: []int{effectStripHelm}},
	SkillRGIntimidate:               {EffectIDs: []int{effectIntimidate}},
	SkillRGGraffiti:                 {},
	SkillRGFlaggraffiti:             {},
	SkillRGCleaner:                  {},
	SkillAMPharmacy:                 {},
	SkillAMDemonstration:            {GroundEffectIDs: []int{effectDemonstration}},
	SkillAMAcidterror:               {BeforeHitEffectIDs: []int{effectThrowItem}},
	SkillAMPotionpitcher:            {EffectIDs: []int{299}},
	SkillAMCannibalize:              {},
	SkillAMSpheremine:               {},
	SkillAMCpWeapon:                 {EffectIDs: []int{effectChemicalProt}},
	SkillAMCpShield:                 {EffectIDs: []int{effectChemicalProt}},
	SkillAMCpArmor:                  {EffectIDs: []int{effectChemicalProt}},
	SkillAMCpHelm:                   {EffectIDs: []int{effectChemicalProt}},
	SkillAMCallhomun:                {},
	SkillAMRest:                     {},
	SkillAMResurrecthomun:           {},
	SkillCRAutoguard:                {EffectIDs: []int{effectGuard}},
	SkillCRShieldcharge:             {EffectIDs: []int{effectShieldCharge}},
	SkillCRShieldboomerang:          {EffectIDs: []int{effectShieldBoomer}, BeforeHitEffectIDs: []int{SkillEffectShieldProjectile}},
	SkillCRReflectshield:            {EffectIDs: []int{effectReflectShield}},
	SkillCRHolycross:                {EffectIDs: []int{effectHolyCross}},
	SkillCRGrandcross:               {EffectIDs: []int{effectGrandCross}},
	SkillCRDevotion:                 {EffectIDs: []int{effectDevotion}},
	SkillCRProvidence:               {EffectIDs: []int{effectProvidence}},
	SkillCRDefender:                 {EffectIDs: []int{effectCrusaderDef}},
	SkillCRSpearquicken:             {EffectIDs: []int{effectSpearQuicken}},
	SkillMOCallspirits:              {},
	SkillMOAbsorbspirits:            {SuccessEffectIDsSelf: []int{effectAbsorbSpirits}},
	SkillMOTripleattack:             {EffectIDs: []int{effectTripleAttack}},
	SkillMOBodyrelocation:           {},
	SkillMOInvestigate:              {EffectIDs: []int{effectChimto}},
	SkillMOFingeroffensive:          {EffectIDs: []int{effectTanji}, HitEffectIDs: []int{1}},
	SkillMOSteelbody:                {EffectIDs: []int{effectSteelBody, effectQuake}},
	SkillMOBladestop:                {},
	SkillMOExplosionspirits:         {EffectIDsOnCaster: []int{effectGumgang2, effectQuake}, BeginCastEffectIDs: []int{12}},
	SkillMOExtremityfist:            {EffectIDs: []int{effectBeginAsura, effectQuake}, HitEffectIDs: []int{effectTeiHit1X}, BeginCastEffectIDs: []int{12}},
	SkillMOChaincombo:               {EffectIDs: []int{effectTeiHit1, effectChainCombo}, EffectIDsOnCaster: []int{effectGumgang3}},
	SkillMOCombofinish:              {EffectIDs: []int{330, effectQuake}},
	SkillSACastcancel:               {},
	SkillSAMagicrod:                 {SuccessEffectIDs: []int{effectMagicRod}},
	SkillSASpellbreaker:             {SuccessEffectIDs: []int{effectSpellBreaker}},
	SkillSAAutospell:                {},
	SkillSAFlamelauncher:            {SuccessEffectIDs: []int{effectFlameLauncher}},
	SkillSAFrostweapon:              {SuccessEffectIDs: []int{effectFrostWeapon}},
	SkillSALightningloader:          {SuccessEffectIDs: []int{effectLightningLoad}},
	SkillSASeismicweapon:            {SuccessEffectIDs: []int{effectSeismicWeapon}},
	SkillSAVolcano:                  {EffectIDsOnCaster: []int{225}, GroundEffectIDs: []int{effectBottomVolcano}},
	SkillSADeluge:                   {EffectIDsOnCaster: []int{236}, GroundEffectIDs: []int{effectBottomDeluge}},
	SkillSAViolentgale:              {EffectIDsOnCaster: []int{237}, GroundEffectIDs: []int{effectBottomViolent}},
	SkillSALandprotector:            {EffectIDsOnCaster: []int{238}, GroundEffectIDs: []int{effectBottomLand}},
	SkillSADispell:                  {SuccessEffectIDs: []int{effectDispell}},
	SkillSAAbracadabra:              {},
	SkillSAMonocell:                 {},
	SkillSAClasschange:              {},
	SkillSASummonmonster:            {},
	SkillSAReverseorcish:            {},
	SkillSADeath:                    {},
	SkillSAFortune:                  {},
	SkillSATamingmonster:            {},
	SkillSAQuestion:                 {},
	SkillSAGravity:                  {},
	SkillSALevelup:                  {},
	SkillSAInstantdeath:             {},
	SkillSAFullrecovery:             {},
	SkillSAComa:                     {},
	SkillBDAdaptation:               {},
	SkillBDEncore:                   {},
	SkillBDLullaby:                  {EffectIDs: []int{effectBottomLullaby}, GroundEffectIDs: []int{effectBottomLullaby}},
	SkillBDRichmankim:               {EffectIDs: []int{effectBottomRichKim}, GroundEffectIDs: []int{effectBottomRichKim}},
	SkillBDEternalchaos:             {EffectIDs: []int{effectBottomChaos}, GroundEffectIDs: []int{effectBottomChaos}},
	SkillBDDrumbattlefield:          {EffectIDs: []int{effectBottomDrum}, GroundEffectIDs: []int{effectBottomDrum}},
	SkillBDRingnibelungen:           {EffectIDs: []int{effectBottomNibelung}, GroundEffectIDs: []int{effectBottomNibelung}},
	SkillBDRokisweil:                {EffectIDs: []int{effectBottomRoki}, GroundEffectIDs: []int{effectBottomRoki}},
	SkillBDIntoabyss:                {EffectIDs: []int{effectBottomAbyss}, GroundEffectIDs: []int{effectBottomAbyss}},
	SkillBDSiegfried:                {EffectIDs: []int{effectBottomSieg}, GroundEffectIDs: []int{effectBottomSieg}},
	SkillBaMusicallesson:            {},
	SkillBaMusicalstrike:            {BeforeHitEffectIDs: []int{SkillEffectArrowShot}, HideCastAura: true},
	SkillBaDissonance:               {GroundEffectIDs: []int{effectBottomDissonance}},
	SkillBaFrostjoke:                {BeginCastEffectIDs: []int{effectTalkFrostJoke}},
	SkillBaWhistle:                  {EffectIDs: []int{effectBottomWhistle}, GroundEffectIDs: []int{effectBottomWhistle}},
	SkillBaAssassincross:            {EffectIDs: []int{effectBottomSinX}, GroundEffectIDs: []int{effectBottomSinX}},
	SkillBaPoembragi:                {EffectIDs: []int{effectBottomBragi}, GroundEffectIDs: []int{effectBottomBragi}},
	SkillBaAppleidun:                {EffectIDs: []int{effectBottomApple}, GroundEffectIDs: []int{effectBottomApple}},
	SkillDCDancinglesson:            {},
	SkillDCThrowarrow:               {BeforeHitEffectIDs: []int{SkillEffectArrowShot}, HideCastAura: true},
	SkillDCUglydance:                {GroundEffectIDs: []int{effectBottomUglyDance}},
	SkillDCScream:                   {BeginCastEffectIDs: []int{effectTalkScream}},
	SkillDCHumming:                  {EffectIDs: []int{effectBottomHumming}, GroundEffectIDs: []int{effectBottomHumming}},
	SkillDCDontforgetme:             {EffectIDs: []int{effectBottomForget}, GroundEffectIDs: []int{effectBottomForget}},
	SkillDCFortunekiss:              {EffectIDs: []int{effectBottomFortune}, GroundEffectIDs: []int{effectBottomFortune}},
	SkillDCServiceforyou:            {EffectIDs: []int{effectBottomService}, GroundEffectIDs: []int{effectBottomService}},
	SkillITMTomahawk:                {EffectIDs: []int{494}},
	SkillNPCDarkcross:               {EffectIDs: []int{effectDarkGrandCross}},
	SkillNPCDarkstrike:              {EffectIDs: []int{effectDarkSoulStrike}},
	SkillNPCDarkthunder:             {EffectIDs: []int{93}, HitEffectIDs: []int{94}},
	SkillNPCStop:                    {EffectIDs: []int{effectNPCStop}},
	SkillNPCPowerup:                 {EffectIDs: []int{effectNPCPowerUp}},
	SkillLKAurablade:                {EffectIDs: []int{effectAuraBlade}, BeginCastEffectIDs: []int{SkillEffectWhitePulse}},
	SkillLKParrying:                 {EffectIDs: []int{effectGuard}},
	SkillLKConcentration:            {EffectIDs: []int{effectLKConcentration}},
	SkillLKBerserk:                  {EffectIDs: []int{effectRedBody, SkillEffectQuake}},
	SkillLKFury:                     {EffectIDs: []int{effectRedBody, SkillEffectQuake}},
	SkillHPAssumptio:                {EffectIDs: []int{440}},
	SkillHPBasilica:                 {GroundEffectIDs: []int{effectBottomBasilica}},
	SkillHWMagiccrasher:             {EffectIDs: []int{effectMagicCrasher}},
	SkillHWMagicpower:               {EffectIDs: []int{SkillEffectMagicPower}, BeginCastEffectIDs: []int{16}, HideCastAura: true},
	SkillHWSouldrain:                {EffectIDsOnCaster: []int{effectEnergyDrain}},
	SkillPaPressure:                 {BeforeHitEffectIDs: []int{effectPressure}},
	SkillPaSacrifice:                {EffectIDs: []int{effectBash3D}},
	SkillPaGospel:                   {EffectIDs: []int{effectBottomGospel}, GroundEffectIDs: []int{SkillEffectGospelGround}},
	SkillPaShieldchain:              {BeforeHitEffectIDs: []int{SkillEffectShieldProjectile}},
	SkillChPalmstrike:               {HitEffectIDs: []int{effectHitLine2, effectQuake}},
	SkillChTigerfist:                {EffectIDs: []int{effectBash3D2, effectQuake}, EffectIDsOnCaster: []int{effectGumgang3}},
	SkillChChaincrush:               {EffectIDs: []int{effectChemical2Dash}},
	SkillPFHpconversion:             {EffectIDs: []int{effectEnergyDrain3}, EffectIDsOnCaster: []int{effectEnergyDrain2}, SuccessEffectIDsSelf: []int{effectTransBlueBody}},
	SkillPFSoulchange:               {EffectIDs: []int{effectLineLink2}, SuccessEffectIDs: []int{385}},
	SkillPFSoulburn:                 {EffectIDs: []int{406}},
	SkillASCEdp:                     {EffectIDs: []int{effectEDP}},
	SkillASCBreaker:                 {BeforeHitEffectIDs: []int{effectSoulBreaker}},
	SkillSNSight:                    {EffectIDs: []int{effectTrueSight}},
	SkillSNFalconassault:            {EffectIDs: []int{effectFalconAssault}, HideCastAura: true},
	SkillSNSharpshooting:            {HitEffectIDs: []int{effectTripleAttack2}, BeforeHitEffectIDs: []int{SkillEffectArrowShot}, BeginCastEffectIDs: []int{SkillEffectSharpShootingCast}},
	SkillSNWindwalk:                 {EffectIDs: []int{effectPortal4}},
	SkillWSMeltdown:                 {EffectIDs: []int{effectMeltdown}},
	SkillWSCreatecoin:               {},
	SkillWSCreatenugget:             {},
	SkillWSCartboost:                {EffectIDs: []int{effectCartBoost}},
	SkillWSSystemcreate:             {},
	SkillSTChasewalk:                {BeginCastEffectIDs: []int{effectCastSpin}},
	SkillSTRejectsword:              {EffectIDs: []int{effectRejectSword}},
	SkillCGArrowvulcan:              {EffectIDs: []int{effectTripleAttack3}, BeforeHitEffectIDs: []int{SkillEffectArrowShot}},
	SkillCGMoonlit:                  {EffectIDs: []int{effectMoonlit}, GroundEffectIDs: []int{effectMoonlit}},
	SkillCGMarionette:               {EffectIDs: []int{395}, HitEffectIDs: []int{396}},
	SkillLKSpiralpierce:             {EffectIDs: []int{effectMagnum2}, BeginCastEffectIDs: []int{SkillEffectSpiralBeforeCast}, HitEffectIDs: []int{SkillEffectSpearHitSound}, HideCastAura: true},
	SkillLKHeadcrush:                {BeginCastEffectIDs: []int{effectBash3D3}, HitEffectIDs: []int{SkillEffectEnemyHitNormal1}},
	SkillLKJointbeat:                {BeginCastEffectIDs: []int{effectBash3D4}, HitEffectIDs: []int{SkillEffectEnemyHitNormal1}},
	SkillHWNapalmvulcan:             {EffectIDs: []int{401}},
	SkillChSoulcollect:              {BeginCastEffectIDs: []int{402, 12}},
	SkillPFMindbreaker:              {SuccessEffectIDs: []int{403}},
	SkillPFMemorize:                 {EffectIDs: []int{505}},
	SkillPFFogwall:                  {GroundEffectIDs: []int{SkillEffectFogWallGround}},
	SkillPFSpiderweb:                {GroundEffectIDs: []int{404}},
	SkillASCMeteorassault:           {EffectIDsOnCaster: []int{409}, HideCastAura: true},
	SkillASCCdp:                     {},
	SkillWEBaby:                     {EffectIDs: []int{408}},
	SkillTKRun:                      {EffectIDs: []int{443}, GroundEffectIDs: []int{442}},
	SkillTKStormkick:                {EffectIDs: []int{435}},
	SkillTKDownkick:                 {EffectIDs: []int{413}},
	SkillTKTurnkick:                 {EffectIDs: []int{414}},
	SkillTKCounter:                  {EffectIDs: []int{415}},
	SkillTKJumpkick:                 {EffectIDs: []int{439}, HitEffectIDs: []int{effectJumpKick}},
	SkillTKHighjump:                 {EffectIDsOnCaster: []int{445}, GroundEffectIDs: []int{411}, HideCastAura: true},
	SkillSGFeel:                     {EffectIDs: []int{432}},
	SkillSGSunWarm:                  {EffectIDs: []int{488}},
	SkillSGMoonWarm:                 {EffectIDs: []int{488}},
	SkillSGStarWarm:                 {EffectIDs: []int{488}},
	SkillSGSunComfort:               {EffectIDs: []int{441}},
	SkillSGMoonComfort:              {EffectIDs: []int{441}},
	SkillSGStarComfort:              {EffectIDs: []int{441}},
	SkillSGHate:                     {GroundEffectIDs: []int{487}},
	SkillSGFusion:                   {EffectIDs: []int{433}},
	SkillAMBerserkpitcher:           {EffectIDs: []int{effectItemFast3}, BeforeHitEffectIDs: []int{541}},
	SkillSLBlacksmith:               {EffectIDs: []int{424, 503}},
	SkillSLSoullinker:               {EffectIDs: []int{424, 503}},
	SkillSLKaizel:                   {EffectIDs: []int{effectKaizel}},
	SkillSLKaahi:                    {EffectIDs: []int{effectHated}},
	SkillSLKaupe:                    {EffectIDs: []int{546}},
	SkillSLKaite:                    {EffectIDs: []int{419}},
	SkillSLStin:                     {EffectIDs: []int{effectStin}},
	SkillSLStun:                     {EffectIDs: []int{effectStin3}},
	SkillSLSma:                      {EffectIDs: []int{effectStin2}, SuccessEffectIDs: []int{425}},
	SkillSLSwoo:                     {EffectIDs: []int{effectM07}, SuccessEffectIDs: []int{420}},
	SkillSLSke:                      {EffectIDs: []int{427}},
	SkillSLSka:                      {EffectIDs: []int{effectSteelBody, effectGumgang2}},
	SkillSTPreserve:                 {EffectIDs: []int{effectPreserve}, BeginCastEffectIDs: []int{SkillEffectSharpShootingCast}},
	SkillSTFullstrip:                {SuccessEffectIDs: []int{495}},
	SkillCRAlchemy:                  {},
	SkillCRSynthesispotion:          {},
	SkillCRSlimpitcher:              {},
	SkillCRFullprotection:           {EffectIDs: []int{effectChemicalProt, 500}},
	SkillCRCultivation:              {},
	SkillPFDoublecasting:            {EffectIDs: []int{521}},
	SkillHWGanbantein:               {EffectIDs: []int{223}, GroundEffectIDs: []int{224}},
	SkillHWGravitation:              {GroundEffectIDs: []int{SkillEffectGravitationGround}},
	SkillWSCarttermination:          {EffectIDs: []int{518}},
	SkillWSWeaponrefine:             {},
	SkillWSOverthrustmax:            {EffectIDs: []int{128}},
	SkillCGLongingfreedom:           {EffectIDs: []int{500}},
	SkillCGHermode:                  {EffectIDs: []int{SkillEffectHermodeMusic}, GroundEffectIDs: []int{effectBottomHermode}},
	SkillCGTarotcard:                {SuccessEffectIDs: []int{500}},
	SkillCRAciddemonstration:        {EffectIDs: []int{effectAcidDemon}},
	SkillSLHigh:                     {EffectIDs: []int{424, 503}},
	SkillAMTwilight1:                {EffectIDs: []int{497}},
	SkillAMTwilight2:                {EffectIDs: []int{498}},
	SkillAMTwilight3:                {EffectIDs: []int{499}},
	SkillGSTripleaction:             {EffectIDs: []int{effectTripleAction}},
	SkillGSBullseye:                 {EffectIDs: []int{effectBullseye}},
	SkillGSMadnesscancel:            {EffectIDs: []int{625}},
	SkillGSAdjustment:               {EffectIDs: []int{626}},
	SkillGSIncreasing:               {EffectIDs: []int{effectNPCPowerUp}},
	SkillGSMagicalbullet:            {EffectIDs: []int{effectMagicalBullet}},
	SkillGSTracking:                 {EffectIDs: []int{effectTrackCasting}, HitEffectIDs: []int{effectTracking}},
	SkillGSDisarm:                   {EffectIDs: []int{effectRGCoin3}},
	SkillGSRapidshower:              {EffectIDs: []int{effectRapidShower}},
	SkillGSDesperado:                {EffectIDs: []int{effectDesperado}},
	SkillGSGatlingfever:             {EffectIDs: []int{626}},
	SkillGSDust:                     {EffectIDs: []int{effectBash3D5}},
	SkillGSFullbuster:               {EffectIDs: []int{effectM02}},
	SkillGSSpreadattack:             {EffectIDs: []int{effectSpreadAttack}},
	SkillNJSyuriken:                 {BeforeHitEffectIDs: []int{effectThrowItem7}},
	SkillNJKunai:                    {BeforeHitEffectIDs: []int{effectThrowItem8}},
	SkillNJHuuma:                    {BeforeHitEffectIDs: []int{effectThrowItem9}},
	SkillNJZenynage:                 {BeforeHitEffectIDs: []int{effectThrowItem10}},
	SkillNJTatamigaeshi:             {GroundEffectIDs: []int{effectTatami}},
	SkillNJKasumikiri:               {EffectIDs: []int{effectKasumikiri}},
	SkillNJKirikage:                 {EffectIDs: []int{effectKirikage}},
	SkillNJBunsinjyutsu:             {EffectIDs: []int{617}},
	SkillNJKouenka:                  {EffectIDs: []int{effectKouenka}},
	SkillNJKaensin:                  {GroundEffectIDs: []int{effectKaen}},
	SkillNJBakuenryu:                {EffectIDs: []int{effectBaku}},
	SkillNJHyousensou:               {EffectIDs: []int{effectHyousensou}},
	SkillNJSuiton:                   {GroundEffectIDs: []int{620}},
	SkillNJHyousyouraku:             {EffectIDs: []int{effectHyousyouraku}},
	SkillNJHuujin:                   {EffectIDs: []int{effectStin4}},
	SkillNJRaigekisai:               {EffectIDs: []int{effectThunderStorm2}},
	SkillNJIssen:                    {EffectIDs: []int{effectIssen}},
	SkillNPCEarthquake:              {EffectIDsOnCaster: []int{effectNPCEarthquake}},
	SkillNPCFirebreath:              {HitEffectIDs: []int{49}},
	SkillNPCThunderbreath:           {HitEffectIDs: []int{52}},
	SkillNPCDragonfear:              {EffectIDs: []int{effectDragonFear}},
	SkillNPCPulsestrike:             {EffectIDsOnCaster: []int{409}},
	SkillNPCWidefreeze:              {EffectIDsOnCaster: []int{89}},
	SkillNPCWidebleeding:            {EffectIDsOnCaster: []int{effectWideBleeding}},
	SkillNPCEvilland:                {GroundEffectIDs: []int{effectBottomEvilLand}},
	SkillNPCCriticalwound:           {HitEffectIDs: []int{effectCriticalWound}},
	SkillALLWewish:                  {EffectIDs: []int{effectChristmasCarol}},
	SkillNPCVenomfog:                {EffectIDs: []int{effectVenomFog}},
	SkillCRShrink:                   {EffectIDs: []int{599}},
	SkillASVenomknife:               {BeforeHitEffectIDs: []int{effectThrowItem6}},
	SkillRGCloseconfine:             {EffectIDs: []int{602}, GroundEffectIDs: []int{effectNPCStop2}},
	SkillWZSightblaster:             {EffectIDs: []int{601}},
	SkillSACreatecon:                {},
	SkillSAElementwater:             {EffectIDs: []int{effectFrostWeapon}},
	SkillHTPhantasmic:               {HitEffectIDs: []int{1}, BeforeHitEffectIDs: []int{SkillEffectArrowShot}},
	SkillBaPangvoice:                {SuccessEffectIDs: []int{effectFVoice}},
	SkillDCWinkcharm:                {SuccessEffectIDs: []int{effectWink}},
	SkillMOKitranslation:            {},
	SkillMOBalkyoung:                {EffectIDs: []int{514}},
	SkillSAElementground:            {EffectIDs: []int{effectSeismicWeapon}},
	SkillSAElementfire:              {EffectIDs: []int{effectFlameLauncher}},
	SkillSAElementwind:              {EffectIDs: []int{effectLightningLoad}},
	SkillBSAdrenaline2:              {EffectIDs: []int{98}, BeginCastEffectIDs: []int{SkillEffectAdrenalineCast}},
	SkillBSGreed:                    {EffectIDs: []int{SkillEffectGreedSound}},
	SkillRKEnchantblade:             {EffectIDs: []int{effectBerserkPotion2}},
	SkillRKSonicwave:                {EffectIDs: []int{effectHealN}},
	SkillRKHundredspear:             {EffectIDs: []int{723}},
	SkillRKIgnitionbreak:            {EffectIDsOnCaster: []int{effectIgnitionBreak}},
	SkillRKDragonbreath:             {HitEffectIDs: []int{effectM05}},
	SkillRKDragonhowling:            {EffectIDs: []int{effectDragonHowling}},
	SkillRKMillenniumshield:         {EffectIDs: []int{effectMillenniumShield}},
	SkillWLWhiteimprison:            {EffectIDs: []int{effectBottomBasilica2}},
	SkillWLFrostmisty:               {EffectIDs: []int{effectFrostMisty}},
	SkillWLJackfrost:                {GroundEffectIDs: []int{801}},
	SkillWLMarshofabyss:             {EffectIDs: []int{effectMarshOfAbyss}},
	SkillWLRecognizedspell:          {EffectIDs: []int{effectRecognized}},
	SkillWLStasis:                   {EffectIDs: []int{effectStasis}},
	SkillWLCrimsonrock:              {EffectIDs: []int{effectCrimsonRock}},
	SkillWLHellinferno:              {EffectIDs: []int{800}, GroundEffectIDs: []int{effectHellInferno}},
	SkillWLChainlightningAtk:        {EffectIDs: []int{effectChainLightning}},
	SkillWLEarthstrain:              {GroundEffectIDs: []int{effectEarthWall}},
	SkillWLTetravortex:              {EffectIDs: []int{effectTetra}, BeginCastEffectIDs: []int{effectTetraCasting}},
	SkillWLRelease:                  {EffectIDs: []int{751}},
	SkillGCVenomimpress:             {EffectIDs: []int{788}},
	SkillGCPoisonsmoke:              {EffectIDs: []int{924}},
	SkillGCRollingcutter:            {EffectIDs: []int{effectCastSpin2}},
	SkillGCCrossripperslasher:       {EffectIDs: []int{769}},
	SkillABJudex:                    {EffectIDs: []int{effectFirePillarOn2}, HitEffectIDs: []int{152}},
	SkillABAdoramus:                 {EffectIDs: []int{effectAdoramus}},
	SkillABCheal:                    {EffectIDs: []int{effectHeal2}},
	SkillABEpiclesis:                {EffectIDs: []int{effectGlassWall4}, GroundEffectIDs: []int{effectGlassWall3}},
	SkillABOratio:                   {EffectIDs: []int{755}},
	SkillABHighnessheal:             {EffectIDs: []int{effectHeal4}, HitEffectIDs: []int{effectHealOffensive}},
	SkillABClearance:                {EffectIDs: []int{753}},
	SkillRAArrowstorm:               {EffectIDs: []int{effectArrowStorm}},
	SkillRAAimedbolt:                {EffectIDs: []int{effectAimedBolt}, BeforeHitEffectIDs: []int{SkillEffectArrowShot}},
	SkillRADetonator:                {EffectIDs: []int{effectConcentration2}},
	SkillRACamouflage:               {EffectIDs: []int{744}},
	SkillRAMagentatrap:              {GroundEffectIDs: []int{739}},
	SkillRACobalttrap:               {GroundEffectIDs: []int{740}},
	SkillRAMaizetrap:                {GroundEffectIDs: []int{741}},
	SkillRAVerduretrap:              {GroundEffectIDs: []int{742}},
	SkillNCFlamelauncher:            {EffectIDs: []int{787}},
	SkillNCInfraredscan:             {EffectIDs: []int{794}},
	SkillNCMagneticfield:            {EffectIDs: []int{781}},
	SkillNCRepair:                   {EffectIDs: []int{785}},
	SkillNCAxeboomerang:             {EffectIDs: []int{774}},
	SkillNCPowerswing:               {EffectIDs: []int{effectCrashAxe}},
	SkillSCBodypaint:                {EffectIDs: []int{effectStretch}},
	SkillSCEnervation:               {EffectIDs: []int{effectEnervation}},
	SkillSCGroomy:                   {EffectIDs: []int{effectEnervation2}},
	SkillSCIgnorance:                {EffectIDs: []int{effectEnervation3}},
	SkillSCLaziness:                 {EffectIDs: []int{effectEnervation4}},
	SkillSCUnlucky:                  {EffectIDs: []int{effectEnervation5}},
	SkillSCWeakness:                 {EffectIDs: []int{effectEnervation6}},
	SkillSCStripaccessary:           {EffectIDs: []int{820}},
	SkillSCManhole:                  {GroundEffectIDs: []int{effectBottomManhole}, SuccessEffectIDs: []int{effectManhole}},
	SkillSCDimensiondoor:            {GroundEffectIDs: []int{effectForestLight6}},
	SkillSCChaospanic:               {GroundEffectIDs: []int{effectBottomAni}},
	SkillSCMaelstrom:                {GroundEffectIDs: []int{effectBottomMaelstrom}},
	SkillSCBloodylust:               {GroundEffectIDs: []int{effectBottomBloodyLust}},
	SkillLGShieldpress:              {BeforeHitEffectIDs: []int{effectPressure2}},
	SkillLGPrestige:                 {EffectIDs: []int{effectPrimeCharge2}},
	SkillLGBanding:                  {EffectIDs: []int{effectPrimeCharge3}},
	SkillLGInspiration:              {EffectIDs: []int{effectPrimeCharge4}},
	SkillSREarthshaker:              {EffectIDs: []int{effectElectric4}},
	SkillWmReverberation:            {GroundEffectIDs: []int{effectBotReverb}},
	SkillWmReverberationMelee:       {EffectIDs: []int{effectBotReverb2}},
	SkillWmDominionImpulse:          {EffectIDs: []int{863}},
	SkillWmSevereRainstorm:          {EffectIDs: []int{effectRainParticle}},
	SkillWmPoemofnetherworld:        {GroundEffectIDs: []int{effectBotReverb2}},
	SkillWmVoiceofsiren:             {GroundEffectIDs: []int{effectHeartAsura}},
	SkillWmLullabyDeepsleep:         {EffectIDs: []int{effectChemicalV2}},
	SkillWmSircleofnature:           {EffectIDs: []int{effectCirclePower2}},
	SkillWmRandomizespell:           {EffectIDs: []int{effectSecra2}},
	SkillWmGloomyday:                {EffectIDs: []int{effectDance1}},
	SkillWmSongOfMana:               {EffectIDsOnCaster: []int{865}, GroundEffectIDs: []int{effectSprPlant3}},
	SkillWmDanceWithWug:             {EffectIDsOnCaster: []int{867}, GroundEffectIDs: []int{effectSprPlant2}},
	SkillWmSaturdayNightFever:       {EffectIDsOnCaster: []int{871}, GroundEffectIDs: []int{effectSprPlant4}},
	SkillWmLeradsDew:                {EffectIDsOnCaster: []int{871}, GroundEffectIDs: []int{effectSprPlant5}},
	SkillWmMelodyofsink:             {EffectIDsOnCaster: []int{873}, GroundEffectIDs: []int{effectSprPlant6}},
	SkillWmBeyondOfWarcry:           {EffectIDsOnCaster: []int{875}, GroundEffectIDs: []int{effectSprPlant7}},
	SkillWmUnlimitedHummingVoice:    {EffectIDsOnCaster: []int{877}, GroundEffectIDs: []int{effectSprPlant8}},
	SkillSOFirewalk:                 {GroundEffectIDs: []int{effectFireWall2}},
	SkillSOElectricwalk:             {GroundEffectIDs: []int{effectShockwave2}},
	SkillSOEarthgrave:               {EffectIDs: []int{927}},
	SkillSODiamonddust:              {EffectIDs: []int{effectColdThrow2}},
	SkillSOPoisonBuster:             {EffectIDs: []int{923}},
	SkillSOPsychicWave:              {EffectIDs: []int{effectSprPlant10}},
	SkillSOWarmer:                   {EffectIDs: []int{effectDemonicFire4}},
	SkillSOVacuumExtreme:            {EffectIDs: []int{921}},
	SkillSOVaretyrSpear:             {BeforeHitEffectIDs: []int{effectPressure3}},
	SkillGNWallofthorn:              {GroundEffectIDs: []int{912}},
	SkillGNCrazyweed:                {EffectIDs: []int{915}},
	SkillGNDemonicFire:              {EffectIDs: []int{916}},
	SkillGNFireExpansion:            {EffectIDs: []int{917}},
	SkillGNFireExpansionSmokePowder: {EffectIDs: []int{918}},
	SkillGNHellsPlant:               {EffectIDs: []int{919}},
	SkillGCDarkcrow:                 {EffectIDs: []int{effectGCDarkCrow}},
	SkillGNIllusiondoping:           {EffectIDs: []int{effectGNIllusionDoping}},
	SkillRKLuxanima:                 {EffectIDs: []int{effectRKLuxAnima}},
	SkillNCMagmaEruption:            {EffectIDs: []int{effectNCMagmaEruption}},
	SkillSOElementalShield:          {EffectIDs: []int{effectSOElemShield}},
	SkillSRFlashcombo:               {EffectIDs: []int{effectSRFlashCombo}},
	SkillABOffertorium:              {EffectIDs: []int{effectABOffertorium}},
	SkillWLTelekinesisIntense:       {EffectIDs: []int{effectWLTelekinesis}},
	SkillALLFullThrottle:            {EffectIDs: []int{effectAllFullThrottle}},
	SkillHlifChange:                 {EffectIDs: []int{505}},
	SkillHamiCastle:                 {EffectIDs: []int{effectHamiCastle}},
	SkillHamiDefence:                {EffectIDs: []int{effectHamiDefence}},
	SkillHamiBloodlust:              {EffectIDs: []int{effectHamiBlood}},
	SkillHfliFleet:                  {EffectIDs: []int{564}},
	SkillHfliSpeed:                  {EffectIDs: []int{564}},
	SkillHvanExplosion:              {EffectIDs: []int{183}},
	SkillMerMagnificat:              {EffectIDs: []int{effectMagnificat}},
	SkillMerQuicken:                 {EffectIDs: []int{effectTwoHandQuicken}},
	SkillMerSight:                   {},
	SkillMerCrash:                   {},
	SkillMerRegain:                  {},
	SkillMerTender:                  {},
	SkillMerBenediction:             {},
	SkillMerRecuperate:              {},
	SkillMerMentalcure:              {},
	SkillMerCompress:                {},
	SkillMerProvoke:                 {SuccessEffectIDs: []int{effectProvoke}},
	SkillMerAutoberserk:             {},
	SkillMerDecagi:                  {EffectIDs: []int{effectDecAgility}},
	SkillMerScapegoat:               {},
	SkillMerLexdivina:               {EffectIDs: []int{effectLexDivina}},
	SkillMerEstimation:              {},
	SkillMerKyrie:                   {EffectIDs: []int{effectKyrie}},
	SkillMerBlessing:                {EffectIDs: []int{effectBlessing}},
	SkillMerIncagi:                  {EffectIDs: []int{effectIncAgility}},
	SkillMhPoisonMist:               {EffectIDs: []int{effectPoisonMist}},
	SkillMhEraserCutter:             {EffectIDs: []int{effectEraserCutter}},
	SkillMhSilentBreeze:             {EffectIDs: []int{961}},
	SkillMhSonicCraw:                {EffectIDs: []int{effectSonicClaw}},
	SkillMhMidnightFrenzy:           {EffectIDs: []int{effectMidnightFrenzy}},
	SkillMhTinderBreaker:            {EffectIDs: []int{effectTinderBreaker}},
	SkillMhMagmaFlow:                {EffectIDs: []int{962}},
	SkillMhLavaSlide:                {EffectIDs: []int{effectLavaSlide}},
	SkillMhVolcanicAsh:              {EffectIDs: []int{effectVolcanicAsh}},
}

type SkillActionKind uint8

const (
	SkillActionDefault SkillActionKind = iota
	SkillActionIdle
	SkillActionSkill
	SkillActionAttack
	SkillActionAttack1
	SkillActionAttack2
	SkillActionAttack3
	SkillActionPickup
	SkillActionAction
	SkillActionReadyfight
	SkillActionNone
	SkillActionAttackFixedFrame
	SkillActionDance
)

var SkillActions = map[uint16]SkillActionKind{
	SkillNVBasic:              SkillActionNone,
	SkillNVTrickdead:          SkillActionNone,
	SkillTFBacksliding:        SkillActionNone,
	SkillSTChasewalk:          SkillActionIdle,
	SkillChSoulcollect:        SkillActionIdle,
	SkillSMBash:               SkillActionAttack,
	SkillSMMagnum:             SkillActionAttack,
	SkillKNPierce:             SkillActionAttack,
	SkillKNBrandishspear:      SkillActionAttack,
	SkillKNSpearstab:          SkillActionAttack,
	SkillKNBowlingbash:        SkillActionAttack,
	SkillBSHammerfall:         SkillActionAttack,
	SkillACChargearrow:        SkillActionAttack,
	SkillRGBackstap:           SkillActionAttack,
	SkillRGRaid:               SkillActionAttack,
	SkillRGIntimidate:         SkillActionAttack,
	SkillCRShieldcharge:       SkillActionAttack,
	SkillCRHolycross:          SkillActionAttack,
	SkillMOChaincombo:         SkillActionAttack,
	SkillMOCombofinish:        SkillActionAttack,
	SkillBaMusicalstrike:      SkillActionAttack,
	SkillDCThrowarrow:         SkillActionAttack,
	SkillNPCDarkcross:         SkillActionAttack,
	SkillChPalmstrike:         SkillActionAttack,
	SkillChTigerfist:          SkillActionAttack,
	SkillChChaincrush:         SkillActionAttack,
	SkillLKSpiralpierce:       SkillActionAttack,
	SkillLKHeadcrush:          SkillActionAttack,
	SkillLKJointbeat:          SkillActionAttack,
	SkillHWMagicpower:         SkillActionAttack,
	SkillPaSacrifice:          SkillActionAttack,
	SkillASCMeteorassault:     SkillActionAttack,
	SkillTKStormkick:          SkillActionAttack,
	SkillTKDownkick:           SkillActionAttack,
	SkillTKTurnkick:           SkillActionAttack,
	SkillTKCounter:            SkillActionAttack,
	SkillTKJumpkick:           SkillActionAttack,
	SkillCRAciddemonstration:  SkillActionAttack,
	SkillGSTripleaction:       SkillActionAttack,
	SkillGSBullseye:           SkillActionAttack,
	SkillGSTracking:           SkillActionAttack,
	SkillGSDisarm:             SkillActionAttack,
	SkillGSPiercingshot:       SkillActionAttack,
	SkillGSRapidshower:        SkillActionAttack,
	SkillGSDesperado:          SkillActionAttack,
	SkillGSDust:               SkillActionAttack,
	SkillGSFullbuster:         SkillActionAttack,
	SkillGSSpreadattack:       SkillActionAttack,
	SkillGSGrounddrift:        SkillActionAttack,
	SkillNJHuuma:              SkillActionAttack,
	SkillNJKasumikiri:         SkillActionAttack,
	SkillNJKirikage:           SkillActionAttack,
	SkillNJIssen:              SkillActionAttack,
	SkillRKSonicwave:          SkillActionAttack,
	SkillRKHundredspear:       SkillActionAttack,
	SkillRKWindcutter:         SkillActionAttack,
	SkillRKIgnitionbreak:      SkillActionAttack,
	SkillRKDragonbreath:       SkillActionAttack,
	SkillGCDarkillusion:       SkillActionAttack,
	SkillGCCounterslash:       SkillActionAttack,
	SkillGCWeaponcrush:        SkillActionAttack,
	SkillGCVenompressure:      SkillActionAttack,
	SkillGCPhantommenace:      SkillActionAttack,
	SkillGCRollingcutter:      SkillActionAttack,
	SkillGCCrossripperslasher: SkillActionAttack,
	SkillNCPilebunker:         SkillActionAttack,
	SkillNCVulcanarm:          SkillActionAttack,
	SkillNCFlamelauncher:      SkillActionAttack,
	SkillNCColdslower:         SkillActionAttack,
	SkillNCArmscannon:         SkillActionAttack,
	SkillNCPowerswing:         SkillActionAttack,
	SkillNCAxetornado:         SkillActionAttack,
	SkillSCFatalmenace:        SkillActionAttack,
	SkillLGCannonspear:        SkillActionAttack,
	SkillLGMoonslasher:        SkillActionAttack,
	SkillLGBanishingpoint:     SkillActionAttack,
	SkillLGTrample:            SkillActionAttack,
	SkillLGShieldpress:        SkillActionAttack,
	SkillLGPinpointattack:     SkillActionAttack,
	SkillLGRageburst:          SkillActionAttack,
	SkillLGOverbrand:          SkillActionAttack,
	SkillLGRayofgenesis:       SkillActionAttack,
	SkillLGEarthdrive:         SkillActionAttack,
	SkillSRDragoncombo:        SkillActionAttack,
	SkillSRSkynetblow:         SkillActionAttack,
	SkillSRFallenempire:       SkillActionAttack,
	SkillSRTigercannon:        SkillActionAttack,
	SkillSRCrescentelbow:      SkillActionAttack,
	SkillSRGateofhell:         SkillActionAttack,
	SkillKNSpearboomerang:     SkillActionAttack1,
	SkillCRShieldboomerang:    SkillActionAttack1,
	SkillAMDemonstration:      SkillActionAttack1,
	SkillAMAcidterror:         SkillActionAttack1,
	SkillAMPotionpitcher:      SkillActionAttack1,
	SkillAMCannibalize:        SkillActionAttack1,
	SkillTFSprinklesand:       SkillActionAttack1,
	SkillTFThrowstone:         SkillActionAttack1,
	SkillNJSyuriken:           SkillActionAttack1,
	SkillNJKunai:              SkillActionAttack1,
	SkillNJZenynage:           SkillActionAttack1,
	SkillITMTomahawk:          SkillActionAttack1,
	SkillASVenomknife:         SkillActionAttack1,
	SkillPaShieldchain:        SkillActionAttack1,
	SkillNCAxeboomerang:       SkillActionAttack1,
	SkillGNSlingitem:          SkillActionAttack1,
	SkillTFPoison:             SkillActionAttack2,
	SkillMCMammonite:          SkillActionAttack2,
	SkillMCCartrevolution:     SkillActionAttack2,
	SkillGNCartTornado:        SkillActionAttack2,
	SkillACDouble:             SkillActionAttack3,
	SkillASCBreaker:           SkillActionAttack3,
	SkillHTPhantasmic:         SkillActionAttack3,
	SkillSNSharpshooting:      SkillActionAttack3,
	SkillRAArrowstorm:         SkillActionAttack3,
	SkillRAAimedbolt:          SkillActionAttack3,
	SkillSCTriangleshot:       SkillActionAttack3,
	SkillHTLandmine:           SkillActionPickup,
	SkillHTAnklesnare:         SkillActionPickup,
	SkillHTShockwave:          SkillActionPickup,
	SkillHTSandman:            SkillActionPickup,
	SkillHTFlasher:            SkillActionPickup,
	SkillHTFreezingtrap:       SkillActionPickup,
	SkillHTBlastmine:          SkillActionPickup,
	SkillHTClaymoretrap:       SkillActionPickup,
	SkillHTRemovetrap:         SkillActionPickup,
	SkillHTTalkiebox:          SkillActionPickup,
	SkillTFPickstone:          SkillActionPickup,
	SkillBSGreed:              SkillActionPickup,
	SkillRAElectricshocker:    SkillActionPickup,
	SkillRAClusterbomb:        SkillActionPickup,
	SkillRAMagentatrap:        SkillActionPickup,
	SkillRACobalttrap:         SkillActionPickup,
	SkillRAMaizetrap:          SkillActionPickup,
	SkillRAVerduretrap:        SkillActionPickup,
	SkillRAFiringtrap:         SkillActionPickup,
	SkillRAIceboundtrap:       SkillActionPickup,
	SkillNJTatamigaeshi:       SkillActionPickup,
	SkillSREarthshaker:        SkillActionPickup,
	SkillSNSight:              SkillActionAction,
	SkillDCWinkcharm:          SkillActionDance,
	SkillDCFortunekiss:        SkillActionDance,
	SkillDCUglydance:          SkillActionDance,
	SkillDCHumming:            SkillActionDance,
	SkillDCDontforgetme:       SkillActionDance,
	SkillDCServiceforyou:      SkillActionDance,
	SkillBaAppleidun:          SkillActionDance,
	SkillBaDissonance:         SkillActionDance,
	SkillBaWhistle:            SkillActionDance,
	SkillBaAssassincross:      SkillActionDance,
	SkillBaPoembragi:          SkillActionDance,
	SkillBDLullaby:            SkillActionDance,
	SkillBDRichmankim:         SkillActionDance,
	SkillBDEternalchaos:       SkillActionDance,
	SkillBDDrumbattlefield:    SkillActionDance,
	SkillBDSiegfried:          SkillActionDance,
	SkillCGHermode:            SkillActionDance,
	SkillBDRingnibelungen:     SkillActionDance,
	SkillBDRokisweil:          SkillActionDance,
	SkillBDIntoabyss:          SkillActionDance,
	SkillCGMoonlit:            SkillActionDance,
	SkillCGMarionette:         SkillActionDance,
	SkillSMEndure:             SkillActionReadyfight,
	SkillACShower:             SkillActionAttack,
	SkillKNAutocounter:        SkillActionAttackFixedFrame,
	SkillLKTensionrelax:       SkillActionNone,
	SkillMOBladestop:          SkillActionSkill,
	SkillMOInvestigate:        SkillActionSkill,
	SkillMOFingeroffensive:    SkillActionSkill,
	SkillMOExtremityfist:      SkillActionAttack,
	SkillCGArrowvulcan:        SkillActionAttack,
	SkillASSonicblow:          SkillActionAttack,
	SkillGCCrossimpact:        SkillActionAttack,
}
