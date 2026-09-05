package db

type SkillRequirement struct {
	SkillID uint16
	Level   int
}

var SkillRequirements = map[uint16][]SkillRequirement{
	SkillSMTwohand:        {{SkillID: SkillSMSword, Level: 1}},
	SkillSMMagnum:         {{SkillID: SkillSMBash, Level: 5}},
	SkillSMEndure:         {{SkillID: SkillSMProvoke, Level: 5}},
	SkillKNPierce:         {{SkillID: SkillKNSpearmastery, Level: 1}},
	SkillKNBrandishspear:  {{SkillID: SkillKNRiding, Level: 1}, {SkillID: SkillKNSpearstab, Level: 3}},
	SkillKNSpearstab:      {{SkillID: SkillKNPierce, Level: 5}},
	SkillKNSpearboomerang: {{SkillID: SkillKNPierce, Level: 3}},
	SkillKNTwohandquicken: {{SkillID: SkillSMTwohand, Level: 1}},
	SkillKNAutocounter:    {{SkillID: SkillSMTwohand, Level: 1}},
	SkillKNBowlingbash: {
		{SkillID: SkillSMBash, Level: 10},
		{SkillID: SkillSMMagnum, Level: 3},
		{SkillID: SkillSMTwohand, Level: 5},
		{SkillID: SkillKNTwohandquicken, Level: 10},
		{SkillID: SkillKNAutocounter, Level: 5},
	},
	SkillKNRiding:          {{SkillID: SkillSMEndure, Level: 1}},
	SkillKNCavaliermastery: {{SkillID: SkillKNRiding, Level: 1}},
	SkillKNOnehand:         {{SkillID: SkillKNTwohandquicken, Level: 10}},
	SkillLKSpiralpierce: {
		{SkillID: SkillKNSpearmastery, Level: 5},
		{SkillID: SkillKNPierce, Level: 5},
		{SkillID: SkillKNRiding, Level: 1},
		{SkillID: SkillKNSpearstab, Level: 5},
	},
	SkillLKHeadcrush:       {{SkillID: SkillKNSpearmastery, Level: 9}, {SkillID: SkillKNRiding, Level: 1}},
	SkillLKJointbeat:       {{SkillID: SkillKNCavaliermastery, Level: 3}, {SkillID: SkillLKHeadcrush, Level: 3}},
	SkillLKAurablade:       {{SkillID: SkillSMMagnum, Level: 5}, {SkillID: SkillSMTwohand, Level: 5}},
	SkillLKParrying:        {{SkillID: SkillSMProvoke, Level: 5}, {SkillID: SkillSMTwohand, Level: 10}, {SkillID: SkillKNTwohandquicken, Level: 3}},
	SkillLKConcentration:   {{SkillID: SkillSMRecovery, Level: 5}, {SkillID: SkillKNSpearmastery, Level: 5}, {SkillID: SkillKNRiding, Level: 1}},
	SkillLKTensionrelax:    {{SkillID: SkillSMProvoke, Level: 5}, {SkillID: SkillSMRecovery, Level: 10}, {SkillID: SkillSMEndure, Level: 3}},
	SkillCRShieldcharge:    {{SkillID: SkillCRAutoguard, Level: 5}},
	SkillCRShieldboomerang: {{SkillID: SkillCRShieldcharge, Level: 3}},
	SkillCRReflectshield:   {{SkillID: SkillCRShieldboomerang, Level: 3}},
	SkillCRHolycross:       {{SkillID: SkillCRTrust, Level: 7}},
	SkillCRGrandcross:      {{SkillID: SkillCRTrust, Level: 10}, {SkillID: SkillCRHolycross, Level: 6}},
	SkillCRDevotion:        {{SkillID: SkillCRGrandcross, Level: 4}, {SkillID: SkillCRReflectshield, Level: 5}},
	SkillCRProvidence:      {{SkillID: SkillALDp, Level: 5}, {SkillID: SkillALHeal, Level: 5}},
	SkillCRDefender:        {{SkillID: SkillCRShieldboomerang, Level: 1}},
	SkillCRSpearquicken:    {{SkillID: SkillKNSpearmastery, Level: 10}},
	SkillPaPressure: {
		{SkillID: SkillSMEndure, Level: 5},
		{SkillID: SkillCRTrust, Level: 5},
		{SkillID: SkillCRShieldcharge, Level: 2},
	},
	SkillPaShieldchain:     {{SkillID: SkillCRShieldboomerang, Level: 5}},
	SkillPaSacrifice:       {{SkillID: SkillSMEndure, Level: 1}, {SkillID: SkillCRTrust, Level: 5}, {SkillID: SkillCRDevotion, Level: 3}},
	SkillPaGospel:          {{SkillID: SkillCRTrust, Level: 8}, {SkillID: SkillALDp, Level: 3}, {SkillID: SkillALDemonbane, Level: 5}},
	SkillMGSafetywall:      {{SkillID: SkillMGNapalmbeat, Level: 7}, {SkillID: SkillMGSoulstrike, Level: 5}},
	SkillMGSoulstrike:      {{SkillID: SkillMGNapalmbeat, Level: 4}},
	SkillMGFrostdiver:      {{SkillID: SkillMGColdbolt, Level: 5}},
	SkillMGFireball:        {{SkillID: SkillMGFirebolt, Level: 4}},
	SkillMGFirewall:        {{SkillID: SkillMGSight, Level: 1}, {SkillID: SkillMGFireball, Level: 5}},
	SkillMGThunderstorm:    {{SkillID: SkillMGLightningbolt, Level: 4}},
	SkillWZFirepillar:      {{SkillID: SkillMGFirewall, Level: 1}},
	SkillWZSightrasher:     {{SkillID: SkillMGSight, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}},
	SkillWZMeteor:          {{SkillID: SkillMGThunderstorm, Level: 1}, {SkillID: SkillWZSightrasher, Level: 2}},
	SkillWZJupitel:         {{SkillID: SkillMGNapalmbeat, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}},
	SkillWZVermilion:       {{SkillID: SkillMGThunderstorm, Level: 1}, {SkillID: SkillWZJupitel, Level: 5}},
	SkillWZWaterball:       {{SkillID: SkillMGColdbolt, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}},
	SkillWZIcewall:         {{SkillID: SkillMGStonecurse, Level: 1}, {SkillID: SkillMGFrostdiver, Level: 1}},
	SkillWZFrostnova:       {{SkillID: SkillWZIcewall, Level: 1}},
	SkillWZStormgust:       {{SkillID: SkillMGFrostdiver, Level: 1}, {SkillID: SkillWZJupitel, Level: 3}},
	SkillWZEarthspike:      {{SkillID: SkillMGStonecurse, Level: 1}},
	SkillWZHeavendrive:     {{SkillID: SkillWZEarthspike, Level: 3}},
	SkillWZQuagmire:        {{SkillID: SkillWZHeavendrive, Level: 1}},
	SkillHWSouldrain:       {{SkillID: SkillMGSrecovery, Level: 5}, {SkillID: SkillMGSoulstrike, Level: 7}},
	SkillHWMagiccrasher:    {{SkillID: SkillMGSrecovery, Level: 1}},
	SkillHWNapalmvulcan:    {{SkillID: SkillMGNapalmbeat, Level: 5}},
	SkillHWGanbantein:      {{SkillID: SkillWZEstimation, Level: 1}, {SkillID: SkillWZIcewall, Level: 1}},
	SkillHWGravitation:     {{SkillID: SkillWZQuagmire, Level: 1}, {SkillID: SkillHWMagiccrasher, Level: 1}, {SkillID: SkillHWMagicpower, Level: 10}},
	SkillSACastcancel:      {{SkillID: SkillSAAdvancedbook, Level: 2}},
	SkillSAMagicrod:        {{SkillID: SkillSAAdvancedbook, Level: 4}},
	SkillSASpellbreaker:    {{SkillID: SkillSAMagicrod, Level: 1}},
	SkillSAFreecast:        {{SkillID: SkillSACastcancel, Level: 1}},
	SkillSAAutospell:       {{SkillID: SkillSAFreecast, Level: 4}},
	SkillSAFlamelauncher:   {{SkillID: SkillMGFirebolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}},
	SkillSAFrostweapon:     {{SkillID: SkillMGColdbolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}},
	SkillSALightningloader: {{SkillID: SkillMGLightningbolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}},
	SkillSASeismicweapon:   {{SkillID: SkillMGStonecurse, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}},
	SkillSADragonology:     {{SkillID: SkillSAAdvancedbook, Level: 9}},
	SkillSAVolcano:         {{SkillID: SkillSAFlamelauncher, Level: 2}},
	SkillSADeluge:          {{SkillID: SkillSAFrostweapon, Level: 2}},
	SkillSAViolentgale:     {{SkillID: SkillSALightningloader, Level: 2}},
	SkillSALandprotector:   {{SkillID: SkillSADeluge, Level: 3}, {SkillID: SkillSAViolentgale, Level: 3}, {SkillID: SkillSAVolcano, Level: 3}},
	SkillSADispell:         {{SkillID: SkillSASpellbreaker, Level: 3}},
	SkillSAAbracadabra:     {{SkillID: SkillSAAutospell, Level: 5}, {SkillID: SkillSADispell, Level: 1}, {SkillID: SkillSALandprotector, Level: 1}},
	SkillPFHpconversion:    {{SkillID: SkillMGSrecovery, Level: 1}, {SkillID: SkillSAMagicrod, Level: 1}},
	SkillPFSoulchange:      {{SkillID: SkillSAMagicrod, Level: 3}, {SkillID: SkillSASpellbreaker, Level: 2}},
	SkillPFSoulburn:        {{SkillID: SkillSACastcancel, Level: 5}, {SkillID: SkillSAMagicrod, Level: 3}, {SkillID: SkillSADispell, Level: 3}},
	SkillPFMindbreaker:     {{SkillID: SkillMGSrecovery, Level: 3}, {SkillID: SkillPFSoulburn, Level: 2}},
	SkillPFMemorize:        {{SkillID: SkillSAAdvancedbook, Level: 5}, {SkillID: SkillSAFreecast, Level: 5}, {SkillID: SkillSAAutospell, Level: 1}},
	SkillPFFogwall:         {{SkillID: SkillSAViolentgale, Level: 2}, {SkillID: SkillSADeluge, Level: 2}},
	SkillPFSpiderweb:       {{SkillID: SkillSADragonology, Level: 4}},
	SkillPFDoublecasting:   {{SkillID: SkillSAAutospell, Level: 1}},
	SkillALPneuma:          {{SkillID: SkillALWarp, Level: 4}},
	SkillALTeleport:        {{SkillID: SkillALRuwach, Level: 1}},
	SkillALWarp:            {{SkillID: SkillALTeleport, Level: 2}},
	SkillALIncagi:          {{SkillID: SkillALHeal, Level: 3}},
	SkillALDecagi:          {{SkillID: SkillALIncagi, Level: 1}},
	SkillALCrucis:          {{SkillID: SkillALDemonbane, Level: 3}},
	SkillALAngelus:         {{SkillID: SkillALDp, Level: 3}},
	SkillALBlessing:        {{SkillID: SkillALDp, Level: 5}},
	SkillALCure:            {{SkillID: SkillALHeal, Level: 2}},
	SkillALDemonbane:       {{SkillID: SkillALDp, Level: 3}},
	SkillALLResurrection:   {{SkillID: SkillMGSrecovery, Level: 4}, {SkillID: SkillPRStrecovery, Level: 1}},
	SkillPRSuffragium:      {{SkillID: SkillPRImpositio, Level: 2}},
	SkillPRAspersio:        {{SkillID: SkillALHolywater, Level: 1}, {SkillID: SkillPRImpositio, Level: 3}},
	SkillPRBenedictio:      {{SkillID: SkillPRAspersio, Level: 5}, {SkillID: SkillPRGloria, Level: 3}},
	SkillPRSanctuary:       {{SkillID: SkillALHeal, Level: 1}},
	SkillPRKyrie:           {{SkillID: SkillALAngelus, Level: 2}},
	SkillPRGloria:          {{SkillID: SkillPRKyrie, Level: 4}, {SkillID: SkillPRMagnificat, Level: 3}},
	SkillPRLexdivina:       {{SkillID: SkillALRuwach, Level: 1}},
	SkillPRTurnundead:      {{SkillID: SkillALLResurrection, Level: 1}, {SkillID: SkillPRLexdivina, Level: 3}},
	SkillPRLexaeterna:      {{SkillID: SkillPRLexdivina, Level: 5}},
	SkillPRMagnus:          {{SkillID: SkillMGSafetywall, Level: 1}, {SkillID: SkillPRLexaeterna, Level: 1}, {SkillID: SkillPRTurnundead, Level: 3}},
	SkillHPManarecharge:    {{SkillID: SkillPRMacemastery, Level: 10}, {SkillID: SkillALDemonbane, Level: 10}},
	SkillHPAssumptio:       {{SkillID: SkillALAngelus, Level: 1}, {SkillID: SkillMGSrecovery, Level: 3}, {SkillID: SkillPRImpositio, Level: 3}},
	SkillHPBasilica:        {{SkillID: SkillPRGloria, Level: 2}, {SkillID: SkillMGSrecovery, Level: 1}, {SkillID: SkillPRKyrie, Level: 3}},
	SkillHPMeditatio:       {{SkillID: SkillMGSrecovery, Level: 5}, {SkillID: SkillPRLexdivina, Level: 5}, {SkillID: SkillPRAspersio, Level: 3}},
	SkillMOIronhand:        {{SkillID: SkillALDemonbane, Level: 10}, {SkillID: SkillALDp, Level: 10}},
	SkillMOSpiritsrecovery: {{SkillID: SkillMOBladestop, Level: 2}},
	SkillMOCallspirits:     {{SkillID: SkillMOIronhand, Level: 2}},
	SkillMOAbsorbspirits:   {{SkillID: SkillMOCallspirits, Level: 5}},
	SkillMOTripleattack:    {{SkillID: SkillMODodge, Level: 5}},
	SkillMOBodyrelocation: {
		{SkillID: SkillMOSpiritsrecovery, Level: 2},
		{SkillID: SkillMOExtremityfist, Level: 3},
		{SkillID: SkillMOSteelbody, Level: 3},
	},
	SkillMODodge:            {{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOCallspirits, Level: 5}},
	SkillMOInvestigate:      {{SkillID: SkillMOCallspirits, Level: 5}},
	SkillMOFingeroffensive:  {{SkillID: SkillMOInvestigate, Level: 3}},
	SkillMOSteelbody:        {{SkillID: SkillMOCombofinish, Level: 3}},
	SkillMOBladestop:        {{SkillID: SkillMODodge, Level: 5}},
	SkillMOExplosionspirits: {{SkillID: SkillMOAbsorbspirits, Level: 1}},
	SkillMOExtremityfist:    {{SkillID: SkillMOExplosionspirits, Level: 3}, {SkillID: SkillMOFingeroffensive, Level: 3}},
	SkillMOChaincombo:       {{SkillID: SkillMOTripleattack, Level: 5}},
	SkillMOCombofinish:      {{SkillID: SkillMOChaincombo, Level: 3}},
	SkillChSoulcollect:      {{SkillID: SkillMOExplosionspirits, Level: 5}},
	SkillChPalmstrike:       {{SkillID: SkillMOIronhand, Level: 7}, {SkillID: SkillMOCallspirits, Level: 5}},
	SkillChTigerfist:        {{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOTripleattack, Level: 5}, {SkillID: SkillMOCombofinish, Level: 3}},
	SkillChChaincrush:       {{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOCallspirits, Level: 5}, {SkillID: SkillChTigerfist, Level: 2}},
	SkillGSFling:            {{SkillID: SkillGSGlittering, Level: 1}},
	SkillGSTripleaction:     {{SkillID: SkillGSGlittering, Level: 1}},
	SkillGSBullseye:         {{SkillID: SkillGSGlittering, Level: 5}},
	SkillGSMadnesscancel:    {{SkillID: SkillGSGlittering, Level: 4}},
	SkillGSAdjustment:       {{SkillID: SkillGSGlittering, Level: 4}},
	SkillGSIncreasing:       {{SkillID: SkillGSGlittering, Level: 2}},
	SkillGSMagicalbullet:    {{SkillID: SkillGSGlittering, Level: 1}},
	SkillGSCracker:          {{SkillID: SkillGSGlittering, Level: 1}},
	SkillGSChainaction:      {{SkillID: SkillGSSingleaction, Level: 1}},
	SkillGSTracking:         {{SkillID: SkillGSSingleaction, Level: 5}},
	SkillGSDisarm:           {{SkillID: SkillGSTracking, Level: 7}},
	SkillGSPiercingshot:     {{SkillID: SkillGSTracking, Level: 5}},
	SkillGSRapidshower:      {{SkillID: SkillGSChainaction, Level: 3}},
	SkillGSDesperado:        {{SkillID: SkillGSRapidshower, Level: 5}},
	SkillGSGatlingfever:     {{SkillID: SkillGSRapidshower, Level: 7}, {SkillID: SkillGSDesperado, Level: 5}},
	SkillGSDust:             {{SkillID: SkillGSSingleaction, Level: 5}},
	SkillGSFullbuster:       {{SkillID: SkillGSDust, Level: 3}},
	SkillGSSpreadattack:     {{SkillID: SkillGSSingleaction, Level: 5}},
	SkillGSGrounddrift:      {{SkillID: SkillGSSpreadattack, Level: 7}},
	SkillNJSyuriken:         {{SkillID: SkillNJTobidougu, Level: 1}},
	SkillNJKunai:            {{SkillID: SkillNJSyuriken, Level: 5}},
	SkillNJHuuma:            {{SkillID: SkillNJTobidougu, Level: 5}, {SkillID: SkillNJKunai, Level: 5}},
	SkillNJZenynage:         {{SkillID: SkillNJTobidougu, Level: 10}, {SkillID: SkillNJHuuma, Level: 5}},
	SkillNJKasumikiri:       {{SkillID: SkillNJShadowjump, Level: 1}},
	SkillNJShadowjump:       {{SkillID: SkillNJTatamigaeshi, Level: 1}},
	SkillNJKirikage:         {{SkillID: SkillNJKasumikiri, Level: 5}},
	SkillNJUtsusemi:         {{SkillID: SkillNJShadowjump, Level: 5}},
	SkillNJBunsinjyutsu: {
		{SkillID: SkillNJNen, Level: 1},
		{SkillID: SkillNJUtsusemi, Level: 4},
		{SkillID: SkillNJKirikage, Level: 3},
	},
	SkillNJKouenka:        {{SkillID: SkillNJNinpou, Level: 1}},
	SkillNJKaensin:        {{SkillID: SkillNJKouenka, Level: 5}},
	SkillNJBakuenryu:      {{SkillID: SkillNJNinpou, Level: 10}, {SkillID: SkillNJKaensin, Level: 7}},
	SkillNJHyousensou:     {{SkillID: SkillNJNinpou, Level: 1}},
	SkillNJSuiton:         {{SkillID: SkillNJHyousensou, Level: 5}},
	SkillNJHyousyouraku:   {{SkillID: SkillNJNinpou, Level: 10}, {SkillID: SkillNJSuiton, Level: 7}},
	SkillNJHuujin:         {{SkillID: SkillNJNinpou, Level: 1}},
	SkillNJRaigekisai:     {{SkillID: SkillNJHuujin, Level: 5}},
	SkillNJKamaitachi:     {{SkillID: SkillNJNinpou, Level: 10}, {SkillID: SkillNJRaigekisai, Level: 5}},
	SkillNJNen:            {{SkillID: SkillNJNinpou, Level: 5}},
	SkillNJIssen:          {{SkillID: SkillNJTobidougu, Level: 7}, {SkillID: SkillNJNen, Level: 1}, {SkillID: SkillNJKirikage, Level: 5}},
	SkillMCDiscount:       {{SkillID: SkillMCInccarry, Level: 3}},
	SkillMCOvercharge:     {{SkillID: SkillMCDiscount, Level: 3}},
	SkillMCPushcart:       {{SkillID: SkillMCInccarry, Level: 5}},
	SkillMCVending:        {{SkillID: SkillMCPushcart, Level: 3}},
	SkillBSSteel:          {{SkillID: SkillBSIron, Level: 1}},
	SkillBSEnchantedstone: {{SkillID: SkillBSIron, Level: 1}},
	SkillBSOrideocon:      {{SkillID: SkillBSEnchantedstone, Level: 1}},
	SkillBSSword:          {{SkillID: SkillBSDagger, Level: 1}},
	SkillBSTwohandsword:   {{SkillID: SkillBSSword, Level: 1}},
	SkillBSAxe:            {{SkillID: SkillBSSword, Level: 2}},
	SkillBSMace:           {{SkillID: SkillBSKnuckle, Level: 1}},
	SkillBSKnuckle:        {{SkillID: SkillBSDagger, Level: 1}},
	SkillBSSpear:          {{SkillID: SkillBSDagger, Level: 2}},
	SkillBSFindingore:     {{SkillID: SkillBSHiltbinding, Level: 1}, {SkillID: SkillBSSteel, Level: 1}},
	SkillBSWeaponresearch: {{SkillID: SkillBSHiltbinding, Level: 1}},
	SkillBSRepairweapon:   {{SkillID: SkillBSWeaponresearch, Level: 1}},
	SkillBSAdrenaline:     {{SkillID: SkillBSHammerfall, Level: 2}},
	SkillBSWeaponperfect:  {{SkillID: SkillBSWeaponresearch, Level: 2}, {SkillID: SkillBSAdrenaline, Level: 2}},
	SkillBSOverthrust:     {{SkillID: SkillBSAdrenaline, Level: 3}},
	SkillBSMaximize:       {{SkillID: SkillBSWeaponperfect, Level: 3}, {SkillID: SkillBSOverthrust, Level: 2}},
	SkillBSAdrenaline2:    {{SkillID: SkillBSAdrenaline, Level: 5}},
	SkillWSMeltdown: {
		{SkillID: SkillBSSkintemper, Level: 3},
		{SkillID: SkillBSHiltbinding, Level: 1},
		{SkillID: SkillBSWeaponresearch, Level: 5},
		{SkillID: SkillBSOverthrust, Level: 3},
	},
	SkillWSCartboost: {
		{SkillID: SkillMCPushcart, Level: 5},
		{SkillID: SkillBSHiltbinding, Level: 1},
		{SkillID: SkillMCCartrevolution, Level: 1},
		{SkillID: SkillMCChangecart, Level: 1},
	},
	SkillWSWeaponrefine:    {{SkillID: SkillBSWeaponresearch, Level: 10}},
	SkillWSCarttermination: {{SkillID: SkillMCMammonite, Level: 10}, {SkillID: SkillBSHammerfall, Level: 5}, {SkillID: SkillWSCartboost, Level: 1}},
	SkillWSOverthrustmax:   {{SkillID: SkillBSOverthrust, Level: 5}},
	SkillACVulture:         {{SkillID: SkillACOwl, Level: 3}},
	SkillACConcentration:   {{SkillID: SkillACVulture, Level: 1}},
	SkillACShower:          {{SkillID: SkillACDouble, Level: 5}},
	SkillHTPower:           {{SkillID: SkillACDouble, Level: 10}},
	SkillHTAnklesnare:      {{SkillID: SkillHTSkidtrap, Level: 1}},
	SkillHTShockwave:       {{SkillID: SkillHTAnklesnare, Level: 1}},
	SkillHTSandman:         {{SkillID: SkillHTFlasher, Level: 1}},
	SkillHTFlasher:         {{SkillID: SkillHTSkidtrap, Level: 1}},
	SkillHTFreezingtrap:    {{SkillID: SkillHTFlasher, Level: 1}},
	SkillHTBlastmine: {
		{SkillID: SkillHTLandmine, Level: 1},
		{SkillID: SkillHTSandman, Level: 1},
		{SkillID: SkillHTFreezingtrap, Level: 1},
	},
	SkillHTClaymoretrap: {{SkillID: SkillHTShockwave, Level: 1}, {SkillID: SkillHTBlastmine, Level: 1}},
	SkillHTRemovetrap:   {{SkillID: SkillHTLandmine, Level: 1}},
	SkillHTTalkiebox:    {{SkillID: SkillHTRemovetrap, Level: 1}, {SkillID: SkillHTShockwave, Level: 1}},
	SkillHTFalcon:       {{SkillID: SkillHTBeastbane, Level: 1}},
	SkillHTSteelcrow:    {{SkillID: SkillHTBlitzbeat, Level: 5}},
	SkillHTBlitzbeat:    {{SkillID: SkillHTFalcon, Level: 1}},
	SkillHTDetecting:    {{SkillID: SkillACConcentration, Level: 1}, {SkillID: SkillHTFalcon, Level: 1}},
	SkillHTSpringtrap:   {{SkillID: SkillHTFalcon, Level: 1}, {SkillID: SkillHTRemovetrap, Level: 1}},
	SkillSNSight: {
		{SkillID: SkillACOwl, Level: 10},
		{SkillID: SkillACVulture, Level: 10},
		{SkillID: SkillACConcentration, Level: 10},
		{SkillID: SkillHTFalcon, Level: 1},
	},
	SkillSNFalconassault: {
		{SkillID: SkillACVulture, Level: 5},
		{SkillID: SkillHTFalcon, Level: 1},
		{SkillID: SkillHTBlitzbeat, Level: 5},
		{SkillID: SkillHTSteelcrow, Level: 3},
	},
	SkillSNSharpshooting: {{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACConcentration, Level: 10}},
	SkillSNWindwalk:      {{SkillID: SkillACConcentration, Level: 9}},
	SkillTFHiding:        {{SkillID: SkillTFSteal, Level: 5}},
	SkillTFDetoxify:      {{SkillID: SkillTFPoison, Level: 3}},
	SkillRGSnatcher:      {{SkillID: SkillTFSteal, Level: 1}},
	SkillRGStealcoin:     {{SkillID: SkillRGSnatcher, Level: 4}},
	SkillRGBackstap:      {{SkillID: SkillRGStealcoin, Level: 4}},
	SkillRGTunneldrive:   {{SkillID: SkillTFHiding, Level: 1}},
	SkillRGRaid:          {{SkillID: SkillRGTunneldrive, Level: 2}, {SkillID: SkillRGBackstap, Level: 2}},
	SkillRGStripweapon:   {{SkillID: SkillRGStriparmor, Level: 5}},
	SkillRGStripshield:   {{SkillID: SkillRGStriphelm, Level: 5}},
	SkillRGStriparmor:    {{SkillID: SkillRGStripshield, Level: 5}},
	SkillRGStriphelm:     {{SkillID: SkillRGStealcoin, Level: 2}},
	SkillRGIntimidate:    {{SkillID: SkillRGBackstap, Level: 4}, {SkillID: SkillRGRaid, Level: 5}},
	SkillRGGraffiti:      {{SkillID: SkillRGFlaggraffiti, Level: 5}},
	SkillRGFlaggraffiti:  {{SkillID: SkillRGCleaner, Level: 1}},
	SkillRGCleaner:       {{SkillID: SkillRGGangster, Level: 1}},
	SkillRGGangster:      {{SkillID: SkillRGStripshield, Level: 3}},
	SkillRGCompulsion:    {{SkillID: SkillRGGangster, Level: 1}},
	SkillRGPlagiarism:    {{SkillID: SkillRGIntimidate, Level: 5}},
	SkillSTChasewalk:     {{SkillID: SkillTFHiding, Level: 5}, {SkillID: SkillRGTunneldrive, Level: 3}},
	SkillSTPreserve:      {{SkillID: SkillRGPlagiarism, Level: 10}},
	SkillSTFullstrip:     {{SkillID: SkillRGStripweapon, Level: 5}},
	SkillASLeft:          {{SkillID: SkillASRight, Level: 2}},
	SkillASCloaking:      {{SkillID: SkillTFHiding, Level: 2}},
	SkillASSonicblow:     {{SkillID: SkillASKatar, Level: 4}},
	SkillASGrimtooth:     {{SkillID: SkillASCloaking, Level: 2}, {SkillID: SkillASSonicblow, Level: 5}},
	SkillASEnchantpoison: {{SkillID: SkillTFPoison, Level: 1}},
	SkillASPoisonreact:   {{SkillID: SkillASEnchantpoison, Level: 3}},
	SkillASVenomdust:     {{SkillID: SkillASEnchantpoison, Level: 5}},
	SkillASSplasher:      {{SkillID: SkillASVenomdust, Level: 5}, {SkillID: SkillASPoisonreact, Level: 5}},
	SkillASCKatar:        {{SkillID: SkillTFDouble, Level: 5}, {SkillID: SkillASKatar, Level: 7}},
	SkillASCEdp:          {{SkillID: SkillASCCdp, Level: 1}},
	SkillASCBreaker: {
		{SkillID: SkillTFDouble, Level: 5},
		{SkillID: SkillTFPoison, Level: 5},
		{SkillID: SkillASCloaking, Level: 3},
		{SkillID: SkillASEnchantpoison, Level: 6},
	},
	SkillASCMeteorassault: {
		{SkillID: SkillASKatar, Level: 5},
		{SkillID: SkillASRight, Level: 3},
		{SkillID: SkillASSonicblow, Level: 5},
		{SkillID: SkillASCBreaker, Level: 1},
	},
	SkillASCCdp: {
		{SkillID: SkillTFPoison, Level: 10},
		{SkillID: SkillTFDetoxify, Level: 1},
		{SkillID: SkillASEnchantpoison, Level: 5},
	},

	SkillAMPharmacy:       {{SkillID: SkillAMLearningpotion, Level: 5}},
	SkillAMDemonstration:  {{SkillID: SkillAMPharmacy, Level: 4}},
	SkillAMAcidterror:     {{SkillID: SkillAMPharmacy, Level: 5}},
	SkillAMPotionpitcher:  {{SkillID: SkillAMPharmacy, Level: 3}},
	SkillAMCannibalize:    {{SkillID: SkillAMPharmacy, Level: 6}},
	SkillAMSpheremine:     {{SkillID: SkillAMPharmacy, Level: 2}},
	SkillAMCpWeapon:       {{SkillID: SkillAMCpArmor, Level: 3}},
	SkillAMCpShield:       {{SkillID: SkillAMCpHelm, Level: 3}},
	SkillAMCpArmor:        {{SkillID: SkillAMCpShield, Level: 3}},
	SkillAMCpHelm:         {{SkillID: SkillAMPharmacy, Level: 2}},
	SkillAMCallhomun:      {{SkillID: SkillAMRest, Level: 1}},
	SkillAMRest:           {{SkillID: SkillAMBioethics, Level: 1}},
	SkillAMResurrecthomun: {{SkillID: SkillAMCallhomun, Level: 1}},
	SkillAMTwilight1:      {{SkillID: SkillAMPharmacy, Level: 10}},
	SkillAMTwilight2:      {{SkillID: SkillAMPharmacy, Level: 10}},
	SkillAMTwilight3:      {{SkillID: SkillAMPharmacy, Level: 10}},
	SkillCRSlimpitcher:    {{SkillID: SkillAMPotionpitcher, Level: 5}},
	SkillCRFullprotection: {
		{SkillID: SkillAMCpWeapon, Level: 5},
		{SkillID: SkillAMCpArmor, Level: 5},
		{SkillID: SkillAMCpShield, Level: 5},
		{SkillID: SkillAMCpHelm, Level: 5},
	},
	SkillCRAciddemonstration: {{SkillID: SkillAMDemonstration, Level: 5}, {SkillID: SkillAMAcidterror, Level: 5}},

	SkillBDEncore:         {{SkillID: SkillBDAdaptation, Level: 1}},
	SkillBDRichmankim:     {{SkillID: SkillBDSiegfried, Level: 3}},
	SkillBDEternalchaos:   {{SkillID: SkillBDRokisweil, Level: 1}},
	SkillBDRingnibelungen: {{SkillID: SkillBDDrumbattlefield, Level: 3}},
	SkillBDIntoabyss:      {{SkillID: SkillBDLullaby, Level: 1}},
	SkillBaMusicalstrike:  {{SkillID: SkillBaMusicallesson, Level: 3}},
	SkillBaDissonance:     {{SkillID: SkillBDAdaptation, Level: 1}, {SkillID: SkillBaMusicallesson, Level: 1}},
	SkillBaFrostjoke:      {{SkillID: SkillBDEncore, Level: 1}},
	SkillBaWhistle:        {{SkillID: SkillBaDissonance, Level: 3}},
	SkillBaAssassincross:  {{SkillID: SkillBaDissonance, Level: 3}},
	SkillBaPoembragi:      {{SkillID: SkillBaDissonance, Level: 3}},
	SkillBaAppleidun:      {{SkillID: SkillBaDissonance, Level: 3}},
	SkillDCThrowarrow:     {{SkillID: SkillDCDancinglesson, Level: 3}},
	SkillDCUglydance:      {{SkillID: SkillBDAdaptation, Level: 1}, {SkillID: SkillDCDancinglesson, Level: 1}},
	SkillDCScream:         {{SkillID: SkillBDEncore, Level: 1}},
	SkillDCHumming:        {{SkillID: SkillDCUglydance, Level: 3}},
	SkillDCDontforgetme:   {{SkillID: SkillDCUglydance, Level: 3}},
	SkillDCFortunekiss:    {{SkillID: SkillDCUglydance, Level: 3}},
	SkillDCServiceforyou:  {{SkillID: SkillDCUglydance, Level: 3}},

	SkillTKReadystorm:   {{SkillID: SkillTKStormkick, Level: 1}},
	SkillTKReadydown:    {{SkillID: SkillTKDownkick, Level: 1}},
	SkillTKReadyturn:    {{SkillID: SkillTKTurnkick, Level: 1}},
	SkillTKReadycounter: {{SkillID: SkillTKCounter, Level: 1}},
	SkillTKDodge:        {{SkillID: SkillTKJumpkick, Level: 7}},
	SkillTKSevenwind: {
		{SkillID: SkillTKHptime, Level: 5},
		{SkillID: SkillTKSptime, Level: 5},
		{SkillID: SkillTKPower, Level: 5},
	},
	SkillTKMission: {{SkillID: SkillTKPower, Level: 5}},

	SkillSGSunWarm:     {{SkillID: SkillSGFeel, Level: 1}},
	SkillSGMoonWarm:    {{SkillID: SkillSGFeel, Level: 2}},
	SkillSGStarWarm:    {{SkillID: SkillSGFeel, Level: 3}},
	SkillSGSunComfort:  {{SkillID: SkillSGFeel, Level: 1}},
	SkillSGMoonComfort: {{SkillID: SkillSGFeel, Level: 2}},
	SkillSGStarComfort: {{SkillID: SkillSGFeel, Level: 3}},
	SkillSGSunAnger:    {{SkillID: SkillSGHate, Level: 1}},
	SkillSGMoonAnger:   {{SkillID: SkillSGHate, Level: 2}},
	SkillSGStarAnger:   {{SkillID: SkillSGHate, Level: 3}},
	SkillSGSunBless:    {{SkillID: SkillSGFeel, Level: 1}, {SkillID: SkillSGHate, Level: 1}},
	SkillSGMoonBless:   {{SkillID: SkillSGFeel, Level: 2}, {SkillID: SkillSGHate, Level: 2}},
	SkillSGStarBless:   {{SkillID: SkillSGFeel, Level: 3}, {SkillID: SkillSGHate, Level: 3}},
	SkillSGFusion:      {{SkillID: SkillSGKnowledge, Level: 9}},

	SkillSLSupernovice: {{SkillID: SkillSLStar, Level: 1}},
	SkillSLKnight:      {{SkillID: SkillSLCrusader, Level: 1}},
	SkillSLWizard:      {{SkillID: SkillSLSage, Level: 1}},
	SkillSLPriest:      {{SkillID: SkillSLMonk, Level: 1}},
	SkillSLRogue:       {{SkillID: SkillSLAssasin, Level: 1}},
	SkillSLBlacksmith:  {{SkillID: SkillSLAlchemist, Level: 1}},
	SkillSLHunter:      {{SkillID: SkillSLBarddancer, Level: 1}},
	SkillSLSoullinker:  {{SkillID: SkillSLStar, Level: 1}},
	SkillSLKaizel:      {{SkillID: SkillSLPriest, Level: 1}},
	SkillSLKaahi: {
		{SkillID: SkillSLCrusader, Level: 1},
		{SkillID: SkillSLMonk, Level: 1},
		{SkillID: SkillSLPriest, Level: 1},
	},
	SkillSLKaupe: {{SkillID: SkillSLAssasin, Level: 1}, {SkillID: SkillSLRogue, Level: 1}},
	SkillSLKaite: {{SkillID: SkillSLSage, Level: 1}, {SkillID: SkillSLWizard, Level: 1}},
	SkillSLKaina: {{SkillID: SkillTKSptime, Level: 1}},
	SkillSLStin:  {{SkillID: SkillSLWizard, Level: 1}},
	SkillSLStun:  {{SkillID: SkillSLWizard, Level: 1}},
	SkillSLSma:   {{SkillID: SkillSLStin, Level: 7}, {SkillID: SkillSLStun, Level: 7}},
	SkillSLSwoo:  {{SkillID: SkillSLPriest, Level: 1}},
	SkillSLSke:   {{SkillID: SkillSLKnight, Level: 1}},
	SkillSLSka:   {{SkillID: SkillSLMonk, Level: 1}},
	SkillSLHigh:  {{SkillID: SkillSLSupernovice, Level: 5}},
}

var SkillRequirementsByJob = map[int]map[uint16][]SkillRequirement{
	JobPriest:     priestSkillRequirementOverrides,
	JobPriestH:    priestSkillRequirementOverrides,
	JobPriestB:    priestSkillRequirementOverrides,
	JobCrusader:   crusaderSkillRequirementOverrides,
	JobCrusader2:  crusaderSkillRequirementOverrides,
	JobCrusaderH:  crusaderSkillRequirementOverrides,
	JobCrusader2H: crusaderSkillRequirementOverrides,
	JobCrusaderB:  crusaderSkillRequirementOverrides,
	JobCrusader2B: crusaderSkillRequirementOverrides,
	JobSage:       sageSkillRequirementOverrides,
	JobSageH:      sageSkillRequirementOverrides,
	JobSageB:      sageSkillRequirementOverrides,
	JobRogue:      rogueSkillRequirementOverrides,
	JobRogueH:     rogueSkillRequirementOverrides,
	JobRogueB:     rogueSkillRequirementOverrides,
	JobBard:       bardSkillRequirementOverrides,
	JobBardH:      clownSkillRequirementOverrides,
	JobBardB:      bardSkillRequirementOverrides,
	JobDancer:     dancerSkillRequirementOverrides,
	JobDancerH:    gypsySkillRequirementOverrides,
	JobDancerB:    dancerSkillRequirementOverrides,
}

var priestSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillMGSafetywall: {{SkillID: SkillPRSanctuary, Level: 3}, {SkillID: SkillPRAspersio, Level: 4}},
}

var crusaderSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillALCure: {{SkillID: SkillCRTrust, Level: 5}},
	SkillALDp:   {{SkillID: SkillALCure, Level: 1}},
	SkillALHeal: {{SkillID: SkillCRTrust, Level: 10}, {SkillID: SkillALDemonbane, Level: 5}},
}

var sageSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillWZEarthspike:  {{SkillID: SkillSASeismicweapon, Level: 1}},
	SkillWZHeavendrive: {{SkillID: SkillWZEarthspike, Level: 1}},
}

var rogueSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillACVulture:    {},
	SkillACDouble:     {{SkillID: SkillACVulture, Level: 10}},
	SkillHTRemovetrap: {{SkillID: SkillACDouble, Level: 5}},
}

var bardSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillBDLullaby:         {{SkillID: SkillBaWhistle, Level: 10}},
	SkillBDDrumbattlefield: {{SkillID: SkillBaAppleidun, Level: 10}},
	SkillBDRokisweil:       {{SkillID: SkillBaAssassincross, Level: 10}},
	SkillBDSiegfried:       {{SkillID: SkillBaPoembragi, Level: 10}},
}

var dancerSkillRequirementOverrides = map[uint16][]SkillRequirement{
	SkillBDLullaby:         {{SkillID: SkillDCHumming, Level: 10}},
	SkillBDDrumbattlefield: {{SkillID: SkillDCServiceforyou, Level: 10}},
	SkillBDRokisweil:       {{SkillID: SkillDCDontforgetme, Level: 10}},
	SkillBDSiegfried:       {{SkillID: SkillDCFortunekiss, Level: 10}},
}

var clownSkillRequirementOverrides = combinedSkillRequirements(bardSkillRequirementOverrides, map[uint16][]SkillRequirement{
	SkillCGArrowvulcan:    {{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACShower, Level: 5}, {SkillID: SkillBaMusicalstrike, Level: 1}},
	SkillCGMoonlit:        {{SkillID: SkillACConcentration, Level: 5}, {SkillID: SkillBaMusicallesson, Level: 7}},
	SkillCGMarionette:     {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaMusicallesson, Level: 5}},
	SkillCGHermode:        {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaMusicallesson, Level: 10}},
	SkillCGTarotcard:      {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaDissonance, Level: 3}},
	SkillCGLongingfreedom: {{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillBaDissonance, Level: 3}, {SkillID: SkillBaMusicallesson, Level: 10}},
	SkillCGSpecialsinger:  {{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillBaDissonance, Level: 3}, {SkillID: SkillBaMusicallesson, Level: 10}},
})

var gypsySkillRequirementOverrides = combinedSkillRequirements(dancerSkillRequirementOverrides, map[uint16][]SkillRequirement{
	SkillCGArrowvulcan:    {{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACShower, Level: 5}, {SkillID: SkillDCThrowarrow, Level: 1}},
	SkillCGMoonlit:        {{SkillID: SkillACConcentration, Level: 5}, {SkillID: SkillDCDancinglesson, Level: 7}},
	SkillCGMarionette:     {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCDancinglesson, Level: 5}},
	SkillCGHermode:        {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCDancinglesson, Level: 10}},
	SkillCGTarotcard:      {{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCUglydance, Level: 3}},
	SkillCGLongingfreedom: {{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillDCUglydance, Level: 3}, {SkillID: SkillDCDancinglesson, Level: 10}},
	SkillCGSpecialsinger:  {{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillDCUglydance, Level: 3}, {SkillID: SkillDCDancinglesson, Level: 10}},
})

func combinedSkillRequirements(requirementSets ...map[uint16][]SkillRequirement) map[uint16][]SkillRequirement {
	out := make(map[uint16][]SkillRequirement)
	for _, requirements := range requirementSets {
		for skillID, skillRequirements := range requirements {
			out[skillID] = skillRequirements
		}
	}
	return out
}

var SkillMaxLevels = map[uint16]int{
	SkillNVBasic:            9,
	SkillNVFirstaid:         1,
	SkillNVTrickdead:        1,
	SkillWEMale:             1,
	SkillWEFemale:           1,
	SkillWECallpartner:      1,
	SkillWEBaby:             1,
	SkillWECallparent:       1,
	SkillWECallbaby:         1,
	SkillSMSword:            10,
	SkillSMTwohand:          10,
	SkillSMRecovery:         10,
	SkillSMBash:             10,
	SkillSMProvoke:          10,
	SkillSMMagnum:           10,
	SkillSMEndure:           10,
	SkillSMMovingrecovery:   1,
	SkillSMFatalblow:        1,
	SkillSMAutoberserk:      1,
	SkillKNSpearmastery:     10,
	SkillKNPierce:           10,
	SkillKNBrandishspear:    10,
	SkillKNSpearstab:        10,
	SkillKNSpearboomerang:   5,
	SkillKNTwohandquicken:   10,
	SkillKNAutocounter:      5,
	SkillKNBowlingbash:      10,
	SkillKNChargeatk:        1,
	SkillKNRiding:           1,
	SkillKNCavaliermastery:  5,
	SkillKNOnehand:          1,
	SkillLKSpiralpierce:     5,
	SkillLKHeadcrush:        5,
	SkillLKJointbeat:        10,
	SkillLKAurablade:        5,
	SkillLKParrying:         10,
	SkillLKConcentration:    5,
	SkillLKTensionrelax:     1,
	SkillLKBerserk:          1,
	SkillCRTrust:            10,
	SkillCRAutoguard:        10,
	SkillCRShieldcharge:     5,
	SkillCRShieldboomerang:  5,
	SkillCRReflectshield:    10,
	SkillCRHolycross:        10,
	SkillCRGrandcross:       10,
	SkillCRDevotion:         5,
	SkillCRProvidence:       5,
	SkillCRDefender:         5,
	SkillCRSpearquicken:     10,
	SkillCRShrink:           1,
	SkillPaPressure:         5,
	SkillPaShieldchain:      5,
	SkillPaSacrifice:        5,
	SkillPaGospel:           10,
	SkillMGSrecovery:        10,
	SkillMGSight:            1,
	SkillMGNapalmbeat:       10,
	SkillMGSafetywall:       10,
	SkillMGSoulstrike:       10,
	SkillMGColdbolt:         10,
	SkillMGFrostdiver:       10,
	SkillMGStonecurse:       10,
	SkillMGFireball:         10,
	SkillMGFirewall:         10,
	SkillMGFirebolt:         10,
	SkillMGLightningbolt:    10,
	SkillMGThunderstorm:     10,
	SkillMGEnergycoat:       1,
	SkillWZFirepillar:       10,
	SkillWZSightrasher:      10,
	SkillWZMeteor:           10,
	SkillWZJupitel:          10,
	SkillWZVermilion:        10,
	SkillWZWaterball:        5,
	SkillWZIcewall:          10,
	SkillWZFrostnova:        10,
	SkillWZStormgust:        10,
	SkillWZEarthspike:       5,
	SkillWZHeavendrive:      5,
	SkillWZQuagmire:         5,
	SkillWZEstimation:       1,
	SkillWZSightblaster:     1,
	SkillHWSouldrain:        10,
	SkillHWMagiccrasher:     1,
	SkillHWMagicpower:       10,
	SkillHWNapalmvulcan:     5,
	SkillHWGanbantein:       1,
	SkillHWGravitation:      5,
	SkillSAAdvancedbook:     10,
	SkillSACastcancel:       5,
	SkillSAMagicrod:         5,
	SkillSASpellbreaker:     5,
	SkillSAFreecast:         10,
	SkillSAAutospell:        10,
	SkillSAFlamelauncher:    5,
	SkillSAFrostweapon:      5,
	SkillSALightningloader:  5,
	SkillSASeismicweapon:    5,
	SkillSADragonology:      5,
	SkillSAVolcano:          5,
	SkillSADeluge:           5,
	SkillSAViolentgale:      5,
	SkillSALandprotector:    5,
	SkillSADispell:          5,
	SkillSAAbracadabra:      10,
	SkillSAMonocell:         10,
	SkillSAClasschange:      10,
	SkillSASummonmonster:    10,
	SkillSAReverseorcish:    10,
	SkillSADeath:            10,
	SkillSAFortune:          10,
	SkillSATamingmonster:    10,
	SkillSAQuestion:         10,
	SkillSAGravity:          10,
	SkillSALevelup:          10,
	SkillSAInstantdeath:     10,
	SkillSAFullrecovery:     10,
	SkillSAComa:             10,
	SkillPFHpconversion:     5,
	SkillPFSoulchange:       1,
	SkillPFSoulburn:         5,
	SkillPFMindbreaker:      5,
	SkillPFMemorize:         1,
	SkillPFFogwall:          1,
	SkillPFSpiderweb:        1,
	SkillPFDoublecasting:    5,
	SkillSACreatecon:        1,
	SkillSAElementwater:     1,
	SkillSAElementground:    1,
	SkillSAElementfire:      1,
	SkillSAElementwind:      1,
	SkillALRuwach:           1,
	SkillALPneuma:           1,
	SkillALTeleport:         2,
	SkillALWarp:             4,
	SkillALHeal:             10,
	SkillALIncagi:           10,
	SkillALDecagi:           10,
	SkillALHolywater:        1,
	SkillALCrucis:           10,
	SkillALAngelus:          10,
	SkillALBlessing:         10,
	SkillALCure:             1,
	SkillALDp:               10,
	SkillALDemonbane:        10,
	SkillALHolylight:        1,
	SkillALLResurrection:    4,
	SkillPRMacemastery:      10,
	SkillPRImpositio:        5,
	SkillPRSuffragium:       3,
	SkillPRAspersio:         5,
	SkillPRBenedictio:       5,
	SkillPRSanctuary:        10,
	SkillPRSlowpoison:       4,
	SkillPRStrecovery:       1,
	SkillPRKyrie:            10,
	SkillPRMagnificat:       5,
	SkillPRGloria:           5,
	SkillPRLexdivina:        10,
	SkillPRTurnundead:       10,
	SkillPRLexaeterna:       1,
	SkillPRMagnus:           10,
	SkillPRRedemptio:        1,
	SkillHPAssumptio:        5,
	SkillHPBasilica:         5,
	SkillHPMeditatio:        10,
	SkillHPManarecharge:     5,
	SkillMOIronhand:         10,
	SkillMOSpiritsrecovery:  5,
	SkillMOCallspirits:      5,
	SkillMOAbsorbspirits:    1,
	SkillMOTripleattack:     10,
	SkillMOBodyrelocation:   1,
	SkillMODodge:            10,
	SkillMOInvestigate:      5,
	SkillMOFingeroffensive:  5,
	SkillMOSteelbody:        5,
	SkillMOBladestop:        5,
	SkillMOExplosionspirits: 5,
	SkillMOExtremityfist:    5,
	SkillMOChaincombo:       5,
	SkillMOCombofinish:      5,
	SkillMOKitranslation:    1,
	SkillMOBalkyoung:        1,
	SkillChSoulcollect:      1,
	SkillChPalmstrike:       5,
	SkillChTigerfist:        5,
	SkillChChaincrush:       10,
	SkillMCInccarry:         10,
	SkillMCDiscount:         10,
	SkillMCOvercharge:       10,
	SkillMCPushcart:         10,
	SkillMCIdentify:         1,
	SkillMCVending:          10,
	SkillMCMammonite:        10,
	SkillMCCartrevolution:   1,
	SkillMCChangecart:       1,
	SkillMCLoud:             1,
	SkillMCCartdecorate:     1,
	SkillBSIron:             5,
	SkillBSSteel:            5,
	SkillBSEnchantedstone:   5,
	SkillBSOrideocon:        5,
	SkillBSDagger:           3,
	SkillBSSword:            3,
	SkillBSTwohandsword:     3,
	SkillBSAxe:              3,
	SkillBSMace:             3,
	SkillBSKnuckle:          3,
	SkillBSSpear:            3,
	SkillBSHiltbinding:      1,
	SkillBSFindingore:       1,
	SkillBSWeaponresearch:   10,
	SkillBSRepairweapon:     1,
	SkillBSSkintemper:       5,
	SkillBSHammerfall:       5,
	SkillBSAdrenaline:       5,
	SkillBSWeaponperfect:    5,
	SkillBSOverthrust:       5,
	SkillBSMaximize:         5,
	SkillBSAdrenaline2:      1,
	SkillBSUnfairlytrick:    1,
	SkillBSGreed:            1,
	SkillWSMeltdown:         10,
	SkillWSCreatecoin:       3,
	SkillWSCreatenugget:     3,
	SkillWSCartboost:        1,
	SkillWSSystemcreate:     1,
	SkillWSWeaponrefine:     10,
	SkillWSCarttermination:  10,
	SkillWSOverthrustmax:    5,
	SkillACOwl:              10,
	SkillACVulture:          10,
	SkillACConcentration:    10,
	SkillACDouble:           10,
	SkillACShower:           10,
	SkillACMakingarrow:      1,
	SkillACChargearrow:      1,
	SkillHTPower:            1,
	SkillHTPhantasmic:       1,
	SkillHTSkidtrap:         5,
	SkillHTLandmine:         5,
	SkillHTAnklesnare:       5,
	SkillHTShockwave:        5,
	SkillHTSandman:          5,
	SkillHTFlasher:          5,
	SkillHTFreezingtrap:     5,
	SkillHTBlastmine:        5,
	SkillHTClaymoretrap:     5,
	SkillHTRemovetrap:       1,
	SkillHTTalkiebox:        1,
	SkillHTBeastbane:        10,
	SkillHTFalcon:           1,
	SkillHTSteelcrow:        10,
	SkillHTBlitzbeat:        5,
	SkillHTDetecting:        4,
	SkillHTSpringtrap:       5,
	SkillSNSight:            10,
	SkillSNFalconassault:    5,
	SkillSNSharpshooting:    5,
	SkillSNWindwalk:         10,
	SkillTFDouble:           10,
	SkillTFMiss:             10,
	SkillTFSteal:            10,
	SkillTFHiding:           10,
	SkillTFPoison:           10,
	SkillTFDetoxify:         1,
	SkillTFSprinklesand:     1,
	SkillTFBacksliding:      1,
	SkillTFPickstone:        1,
	SkillTFThrowstone:       1,
	SkillRGSnatcher:         10,
	SkillRGStealcoin:        10,
	SkillRGBackstap:         10,
	SkillRGTunneldrive:      5,
	SkillRGRaid:             5,
	SkillRGStripweapon:      5,
	SkillRGStripshield:      5,
	SkillRGStriparmor:       5,
	SkillRGStriphelm:        5,
	SkillRGIntimidate:       5,
	SkillRGGraffiti:         1,
	SkillRGFlaggraffiti:     5,
	SkillRGCleaner:          1,
	SkillRGGangster:         1,
	SkillRGCompulsion:       5,
	SkillRGPlagiarism:       10,
	SkillRGCloseconfine:     1,
	SkillSTChasewalk:        5,
	SkillSTRejectsword:      5,
	SkillSTPreserve:         1,
	SkillSTFullstrip:        5,
	SkillASRight:            5,
	SkillASLeft:             5,
	SkillASKatar:            10,
	SkillASCloaking:         10,
	SkillASSonicblow:        10,
	SkillASGrimtooth:        5,
	SkillASEnchantpoison:    10,
	SkillASPoisonreact:      10,
	SkillASVenomdust:        10,
	SkillASSplasher:         10,
	SkillASSonicaccel:       1,
	SkillASVenomknife:       1,
	SkillASCKatar:           5,
	SkillASCEdp:             5,
	SkillASCBreaker:         10,
	SkillASCMeteorassault:   10,
	SkillASCCdp:             1,

	SkillAMAxemastery:        10,
	SkillAMLearningpotion:    10,
	SkillAMPharmacy:          10,
	SkillAMDemonstration:     5,
	SkillAMAcidterror:        5,
	SkillAMPotionpitcher:     5,
	SkillAMCannibalize:       5,
	SkillAMSpheremine:        5,
	SkillAMCpWeapon:          5,
	SkillAMCpShield:          5,
	SkillAMCpArmor:           5,
	SkillAMCpHelm:            5,
	SkillAMBioethics:         1,
	SkillAMCallhomun:         1,
	SkillAMRest:              1,
	SkillAMResurrecthomun:    5,
	SkillAMBerserkpitcher:    1,
	SkillCRSlimpitcher:       10,
	SkillCRFullprotection:    5,
	SkillCRAciddemonstration: 10,
	SkillCRCultivation:       2,
	SkillAMTwilight1:         1,
	SkillAMTwilight2:         1,
	SkillAMTwilight3:         1,

	SkillBDAdaptation:      1,
	SkillBDEncore:          1,
	SkillBDLullaby:         1,
	SkillBDRichmankim:      5,
	SkillBDEternalchaos:    1,
	SkillBDDrumbattlefield: 5,
	SkillBDRingnibelungen:  5,
	SkillBDRokisweil:       1,
	SkillBDIntoabyss:       1,
	SkillBDSiegfried:       5,
	SkillBaMusicallesson:   10,
	SkillBaMusicalstrike:   5,
	SkillBaDissonance:      5,
	SkillBaFrostjoke:       5,
	SkillBaWhistle:         10,
	SkillBaAssassincross:   10,
	SkillBaPoembragi:       10,
	SkillBaAppleidun:       10,
	SkillDCDancinglesson:   10,
	SkillDCThrowarrow:      5,
	SkillDCUglydance:       5,
	SkillDCScream:          5,
	SkillDCHumming:         10,
	SkillDCDontforgetme:    10,
	SkillDCFortunekiss:     10,
	SkillDCServiceforyou:   10,
	SkillCGArrowvulcan:     10,
	SkillCGMoonlit:         5,
	SkillCGMarionette:      1,
	SkillCGLongingfreedom:  5,
	SkillCGHermode:         5,
	SkillCGTarotcard:       5,
	SkillBaPangvoice:       1,
	SkillDCWinkcharm:       1,
	SkillCGSpecialsinger:   1,

	SkillTKRun:          10,
	SkillTKReadystorm:   1,
	SkillTKStormkick:    7,
	SkillTKReadydown:    1,
	SkillTKDownkick:     7,
	SkillTKReadyturn:    1,
	SkillTKTurnkick:     7,
	SkillTKReadycounter: 1,
	SkillTKCounter:      7,
	SkillTKDodge:        1,
	SkillTKJumpkick:     7,
	SkillTKHptime:       10,
	SkillTKSptime:       10,
	SkillTKPower:        5,
	SkillTKSevenwind:    7,
	SkillTKHighjump:     5,
	SkillTKMission:      1,

	SkillSGFeel:        3,
	SkillSGSunWarm:     3,
	SkillSGMoonWarm:    3,
	SkillSGStarWarm:    3,
	SkillSGSunComfort:  4,
	SkillSGMoonComfort: 4,
	SkillSGStarComfort: 4,
	SkillSGHate:        3,
	SkillSGSunAnger:    3,
	SkillSGMoonAnger:   3,
	SkillSGStarAnger:   3,
	SkillSGSunBless:    5,
	SkillSGMoonBless:   5,
	SkillSGStarBless:   5,
	SkillSGDevil:       10,
	SkillSGFriend:      3,
	SkillSGKnowledge:   10,
	SkillSGFusion:      1,

	SkillSLAlchemist:   5,
	SkillSLMonk:        5,
	SkillSLStar:        5,
	SkillSLSage:        5,
	SkillSLCrusader:    5,
	SkillSLSupernovice: 5,
	SkillSLKnight:      5,
	SkillSLWizard:      5,
	SkillSLPriest:      5,
	SkillSLBarddancer:  5,
	SkillSLRogue:       5,
	SkillSLAssasin:     5,
	SkillSLBlacksmith:  5,
	SkillSLHunter:      5,
	SkillSLSoullinker:  5,
	SkillSLKaizel:      7,
	SkillSLKaahi:       7,
	SkillSLKaupe:       3,
	SkillSLKaite:       7,
	SkillSLKaina:       7,
	SkillSLStin:        7,
	SkillSLStun:        7,
	SkillSLSma:         10,
	SkillSLSwoo:        7,
	SkillSLSke:         3,
	SkillSLSka:         3,
	SkillSLHigh:        5,

	SkillGSGlittering:    5,
	SkillGSFling:         1,
	SkillGSTripleaction:  1,
	SkillGSBullseye:      1,
	SkillGSMadnesscancel: 1,
	SkillGSAdjustment:    1,
	SkillGSIncreasing:    1,
	SkillGSMagicalbullet: 1,
	SkillGSCracker:       1,
	SkillGSSingleaction:  10,
	SkillGSSnakeeye:      10,
	SkillGSChainaction:   10,
	SkillGSTracking:      10,
	SkillGSDisarm:        5,
	SkillGSPiercingshot:  5,
	SkillGSRapidshower:   10,
	SkillGSDesperado:     10,
	SkillGSGatlingfever:  10,
	SkillGSDust:          10,
	SkillGSFullbuster:    10,
	SkillGSSpreadattack:  10,
	SkillGSGrounddrift:   10,

	SkillNJTobidougu:    10,
	SkillNJSyuriken:     10,
	SkillNJKunai:        5,
	SkillNJHuuma:        5,
	SkillNJZenynage:     10,
	SkillNJTatamigaeshi: 5,
	SkillNJKasumikiri:   10,
	SkillNJShadowjump:   5,
	SkillNJKirikage:     5,
	SkillNJUtsusemi:     5,
	SkillNJBunsinjyutsu: 10,
	SkillNJNinpou:       10,
	SkillNJKouenka:      10,
	SkillNJKaensin:      10,
	SkillNJBakuenryu:    5,
	SkillNJHyousensou:   10,
	SkillNJSuiton:       10,
	SkillNJHyousyouraku: 5,
	SkillNJHuujin:       10,
	SkillNJRaigekisai:   5,
	SkillNJKamaitachi:   5,
	SkillNJNen:          5,
	SkillNJIssen:        10,
}

var superNoviceSkillTree = []uint16{
	SkillSMSword, SkillSMBash, SkillSMProvoke, SkillTFDouble, SkillTFSteal, SkillTFPoison,
	SkillSMRecovery, SkillSMMagnum, SkillSMEndure, SkillTFMiss, SkillTFHiding, SkillTFDetoxify,
	SkillMGStonecurse, SkillMGColdbolt, SkillMGLightningbolt, SkillMGNapalmbeat, SkillMGFirebolt, SkillMGSight,
	SkillMGSrecovery, SkillMGFrostdiver, SkillMGThunderstorm, SkillMGSoulstrike, SkillMGFireball,
	SkillALRuwach, SkillALHeal, SkillALHolywater, SkillALDp, SkillMGSafetywall, SkillMGFirewall,
	SkillACOwl, SkillALTeleport, SkillALCure, SkillALIncagi, SkillALBlessing, SkillALDemonbane, SkillALAngelus,
	SkillACVulture, SkillALWarp, SkillMCInccarry, SkillALDecagi, SkillMCIdentify, SkillALCrucis,
	SkillMCMammonite, SkillACConcentration, SkillALPneuma, SkillMCDiscount, SkillMCOvercharge,
	SkillMCPushcart, SkillMCVending,
}

var magicianSkillTree = []uint16{
	SkillMGStonecurse, SkillMGColdbolt, SkillMGLightningbolt, SkillMGNapalmbeat, SkillMGFirebolt, SkillMGSight, SkillMGSrecovery, SkillMGFrostdiver, SkillMGThunderstorm, SkillMGSoulstrike, SkillMGFireball, SkillMGEnergycoat, SkillMGSafetywall, SkillMGFirewall,
}

var merchantSkillTree = []uint16{
	SkillMCInccarry,
	SkillMCMammonite,
	SkillMCIdentify,
	SkillMCLoud,
	SkillMCDiscount,
	SkillMCPushcart,
	SkillMCChangecart,
	SkillMCCartdecorate,
	SkillMCOvercharge,
	SkillMCVending,
	SkillMCCartrevolution,
}

var blacksmithSkillTree = []uint16{
	SkillBSIron,
	SkillBSHiltbinding,
	SkillBSSkintemper,
	SkillBSHammerfall,
	SkillBSDagger,
	SkillBSSteel,
	SkillBSEnchantedstone,
	SkillBSWeaponresearch,
	SkillBSAdrenaline,
	SkillBSSpear,
	SkillBSSword,
	SkillBSKnuckle,
	SkillBSFindingore,
	SkillBSOrideocon,
	SkillBSRepairweapon,
	SkillBSWeaponperfect,
	SkillBSOverthrust,
	SkillBSTwohandsword,
	SkillBSMace,
	SkillBSMaximize,
	SkillBSAxe,
	SkillBSAdrenaline2,
	SkillBSGreed,
	SkillBSUnfairlytrick,
}

var whitesmithSkillTree = []uint16{
	SkillWSCartboost,
	SkillWSCarttermination,
	SkillWSMeltdown,
	SkillWSOverthrustmax,
	SkillWSWeaponrefine,
}

var alchemistSkillTree = []uint16{
	SkillAMLearningpotion,
	SkillAMSpheremine,
	SkillAMAxemastery,
	SkillAMCpHelm,
	SkillAMBioethics,
	SkillAMTwilight1,
	SkillAMPharmacy,
	SkillAMPotionpitcher,
	SkillAMCpShield,
	SkillAMRest,
	SkillAMBerserkpitcher,
	SkillAMTwilight2,
	SkillAMDemonstration,
	SkillAMCpArmor,
	SkillAMCallhomun,
	SkillAMTwilight3,
	SkillAMAcidterror,
	SkillAMCpWeapon,
	SkillAMResurrecthomun,
	SkillAMCannibalize,
}

var creatorSkillTree = []uint16{
	SkillCRSlimpitcher,
	SkillCRAciddemonstration,
	SkillCRFullprotection,
}

var archerSkillTree = []uint16{
	SkillACDouble,
	SkillACOwl,
	SkillACChargearrow,
	SkillACShower,
	SkillACVulture,
	SkillACMakingarrow,
	SkillACConcentration,
}

var hunterSkillTree = []uint16{
	SkillHTBeastbane,
	SkillHTSkidtrap,
	SkillHTLandmine,
	SkillHTPower,
	SkillHTFalcon,
	SkillHTFlasher,
	SkillHTAnklesnare,
	SkillHTRemovetrap,
	SkillHTPhantasmic,
	SkillHTBlitzbeat,
	SkillHTSandman,
	SkillHTFreezingtrap,
	SkillHTShockwave,
	SkillHTSpringtrap,
	SkillHTDetecting,
	SkillHTSteelcrow,
	SkillHTBlastmine,
	SkillHTTalkiebox,
	SkillHTClaymoretrap,
}

var sniperSkillTree = []uint16{
	SkillSNFalconassault,
	SkillSNSharpshooting,
	SkillSNSight,
	SkillSNWindwalk,
}

var bardSkillTree = []uint16{
	SkillBDAdaptation,
	SkillBaMusicallesson,
	SkillBaDissonance,
	SkillBaPangvoice,
	SkillBDEncore,
	SkillBaMusicalstrike,
	SkillBaWhistle,
	SkillBaAssassincross,
	SkillBaPoembragi,
	SkillBaAppleidun,
	SkillBaFrostjoke,
	SkillBDLullaby,
	SkillBDRokisweil,
	SkillBDSiegfried,
	SkillBDDrumbattlefield,
	SkillBDIntoabyss,
	SkillBDEternalchaos,
	SkillBDRichmankim,
	SkillBDRingnibelungen,
}

var dancerSkillTree = []uint16{
	SkillBDAdaptation,
	SkillDCDancinglesson,
	SkillDCUglydance,
	SkillDCWinkcharm,
	SkillBDEncore,
	SkillDCThrowarrow,
	SkillDCHumming,
	SkillDCDontforgetme,
	SkillDCFortunekiss,
	SkillDCServiceforyou,
	SkillDCScream,
	SkillBDLullaby,
	SkillBDRokisweil,
	SkillBDSiegfried,
	SkillBDDrumbattlefield,
	SkillBDIntoabyss,
	SkillBDEternalchaos,
	SkillBDRichmankim,
	SkillBDRingnibelungen,
}

var clownGypsySkillTree = []uint16{
	SkillCGArrowvulcan,
	SkillCGMoonlit,
	SkillCGMarionette,
	SkillCGHermode,
	SkillCGLongingfreedom,
	SkillCGSpecialsinger,
	SkillCGTarotcard,
}

var thiefSkillTree = []uint16{
	SkillTFDouble,
	SkillTFSteal,
	SkillTFPoison,
	SkillTFSprinklesand,
	SkillTFThrowstone,
	SkillTFMiss,
	SkillTFHiding,
	SkillTFDetoxify,
	SkillTFBacksliding,
	SkillTFPickstone,
}

var assassinSkillTree = []uint16{
	SkillASRight,
	SkillASKatar,
	SkillASCloaking,
	SkillASEnchantpoison,
	SkillASVenomknife,
	SkillASLeft,
	SkillASSonicblow,
	SkillASVenomdust,
	SkillASPoisonreact,
	SkillASSonicaccel,
	SkillASGrimtooth,
	SkillASSplasher,
}

var assassinCrossSkillTree = []uint16{
	SkillASCBreaker,
	SkillASCCdp,
	SkillASCEdp,
	SkillASCKatar,
	SkillASCMeteorassault,
}

var rogueSkillTree = []uint16{
	SkillACVulture,
	SkillRGTunneldrive,
	SkillRGSnatcher,
	SkillRGStriphelm,
	SkillSMSword,
	SkillRGCloseconfine,
	SkillACDouble,
	SkillRGStealcoin,
	SkillRGStripshield,
	SkillRGGangster,
	SkillHTRemovetrap,
	SkillRGBackstap,
	SkillRGStriparmor,
	SkillRGCleaner,
	SkillRGCompulsion,
	SkillRGRaid,
	SkillRGStripweapon,
	SkillRGFlaggraffiti,
	SkillRGIntimidate,
	SkillRGGraffiti,
	SkillRGPlagiarism,
}

var stalkerSkillTree = []uint16{
	SkillSTChasewalk,
	SkillSTFullstrip,
	SkillSTPreserve,
	SkillSTRejectsword,
}

var acolyteSkillTree = []uint16{
	SkillALRuwach,
	SkillALHeal,
	SkillALHolywater,
	SkillALDp,
	SkillALHolylight,
	SkillALTeleport,
	SkillALCure,
	SkillALIncagi,
	SkillALBlessing,
	SkillALDemonbane,
	SkillALAngelus,
	SkillALWarp,
	SkillALDecagi,
	SkillALCrucis,
	SkillALPneuma,
}

var priestSkillTree = []uint16{
	SkillPRKyrie,
	SkillPRMagnificat,
	SkillPRStrecovery,
	SkillMGSrecovery,
	SkillPRLexdivina,
	SkillPRImpositio,
	SkillPRSanctuary,
	SkillPRGloria,
	SkillPRSlowpoison,
	SkillALLResurrection,
	SkillPRLexaeterna,
	SkillPRSuffragium,
	SkillPRAspersio,
	SkillPRBenedictio,
	SkillPRMacemastery,
	SkillPRTurnundead,
	SkillMGSafetywall,
	SkillPRMagnus,
	SkillPRRedemptio,
}

var highPriestSkillTree = []uint16{
	SkillHPAssumptio,
	SkillHPBasilica,
	SkillHPManarecharge,
	SkillHPMeditatio,
}

var monkSkillTree = []uint16{
	SkillMOIronhand,
	SkillMOCallspirits,
	SkillMODodge,
	SkillMOTripleattack,
	SkillMOKitranslation,
	SkillMOAbsorbspirits,
	SkillMOInvestigate,
	SkillMOBladestop,
	SkillMOChaincombo,
	SkillMOBalkyoung,
	SkillMOExplosionspirits,
	SkillMOFingeroffensive,
	SkillMOSpiritsrecovery,
	SkillMOCombofinish,
	SkillMOExtremityfist,
	SkillMOSteelbody,
	SkillMOBodyrelocation,
}

var championSkillTree = []uint16{
	SkillChPalmstrike,
	SkillChSoulcollect,
	SkillChTigerfist,
	SkillChChaincrush,
}

var swordmanSkillTree = []uint16{
	SkillSMSword,
	SkillSMRecovery,
	SkillSMBash,
	SkillSMProvoke,
	SkillSMAutoberserk,
	SkillSMMovingrecovery,
	SkillSMTwohand,
	SkillSMMagnum,
	SkillSMEndure,
	SkillSMFatalblow,
}

var knightSkillTree = []uint16{
	SkillKNTwohandquicken,
	SkillKNAutocounter,
	SkillKNRiding,
	SkillKNSpearmastery,
	SkillKNChargeatk,
	SkillKNBowlingbash,
	SkillKNCavaliermastery,
	SkillKNPierce,
	SkillKNOnehand,
	SkillKNSpearboomerang,
	SkillKNSpearstab,
	SkillKNBrandishspear,
}

var lordKnightSkillTree = []uint16{
	SkillLKBerserk,
	SkillLKTensionrelax,
	SkillLKParrying,
	SkillLKAurablade,
	SkillLKConcentration,
	SkillLKHeadcrush,
	SkillLKSpiralpierce,
	SkillLKJointbeat,
}

var crusaderSkillTree = []uint16{
	SkillCRTrust,
	SkillCRAutoguard,
	SkillKNSpearmastery,
	SkillKNRiding,
	SkillCRShrink,
	SkillALCure,
	SkillCRHolycross,
	SkillCRShieldcharge,
	SkillCRSpearquicken,
	SkillKNCavaliermastery,
	SkillALDp,
	SkillCRGrandcross,
	SkillCRShieldboomerang,
	SkillALDemonbane,
	SkillCRReflectshield,
	SkillCRDefender,
	SkillALHeal,
	SkillCRDevotion,
	SkillCRProvidence,
}

var paladinSkillTree = []uint16{
	SkillPaPressure,
	SkillPaShieldchain,
	SkillPaGospel,
	SkillPaSacrifice,
}

var wizardSkillTree = []uint16{
	SkillWZEstimation,
	SkillWZIcewall,
	SkillWZJupitel,
	SkillWZEarthspike,
	SkillWZSightrasher,
	SkillWZFirepillar,
	SkillWZSightblaster,
	SkillWZFrostnova,
	SkillWZVermilion,
	SkillWZHeavendrive,
	SkillWZMeteor,
	SkillWZWaterball,
	SkillWZQuagmire,
	SkillWZStormgust,
}

var highWizardSkillTree = []uint16{
	SkillHWGanbantein,
	SkillHWMagiccrasher,
	SkillHWSouldrain,
	SkillHWNapalmvulcan,
	SkillHWMagicpower,
	SkillHWGravitation,
}

var sageSkillTree = []uint16{
	SkillSAAdvancedbook,
	SkillWZEstimation,
	SkillSAElementwater,
	SkillSAElementwind,
	SkillSAElementground,
	SkillSAElementfire,
	SkillSACreatecon,
	SkillSADragonology,
	SkillSASeismicweapon,
	SkillSACastcancel,
	SkillSAMagicrod,
	SkillSAFrostweapon,
	SkillSALightningloader,
	SkillSAFlamelauncher,
	SkillWZEarthspike,
	SkillSAFreecast,
	SkillSASpellbreaker,
	SkillSADeluge,
	SkillSAViolentgale,
	SkillSAVolcano,
	SkillWZHeavendrive,
	SkillSAAutospell,
	SkillSADispell,
	SkillSALandprotector,
	SkillSAAbracadabra,
}

var professorSkillTree = []uint16{
	SkillPFSpiderweb,
	SkillPFSoulchange,
	SkillPFFogwall,
	SkillPFHpconversion,
	SkillPFDoublecasting,
	SkillPFMemorize,
	SkillPFSoulburn,
	SkillPFMindbreaker,
}

var taekwonSkillTree = []uint16{
	SkillTKRun,
	SkillTKStormkick,
	SkillTKDownkick,
	SkillTKTurnkick,
	SkillTKCounter,
	SkillTKJumpkick,
	SkillTKHighjump,
	SkillTKReadystorm,
	SkillTKReadydown,
	SkillTKReadyturn,
	SkillTKReadycounter,
	SkillTKDodge,
	SkillTKHptime,
	SkillTKSptime,
	SkillTKPower,
	SkillTKSevenwind,
	SkillTKMission,
}

var starGladiatorSkillTree = []uint16{
	SkillSGFeel,
	SkillSGHate,
	SkillSGDevil,
	SkillSGKnowledge,
	SkillSGSunWarm,
	SkillSGSunComfort,
	SkillSGSunAnger,
	SkillSGSunBless,
	SkillSGFriend,
	SkillSGFusion,
	SkillSGMoonWarm,
	SkillSGMoonComfort,
	SkillSGMoonAnger,
	SkillSGMoonBless,
	SkillSGStarWarm,
	SkillSGStarComfort,
	SkillSGStarAnger,
	SkillSGStarBless,
}

var soulLinkerSkillTree = []uint16{
	SkillSLAlchemist,
	SkillSLStar,
	SkillSLAssasin,
	SkillSLCrusader,
	SkillSLBarddancer,
	SkillSLSupernovice,
	SkillSLBlacksmith,
	SkillSLSoullinker,
	SkillSLRogue,
	SkillSLKnight,
	SkillSLHunter,
	SkillSLHigh,
	SkillSLMonk,
	SkillSLKaupe,
	SkillSLSke,
	SkillSLSage,
	SkillSLKaina,
	SkillSLPriest,
	SkillSLSka,
	SkillSLWizard,
	SkillSLKaite,
	SkillSLKaahi,
	SkillSLKaizel,
	SkillSLSwoo,
	SkillSLStin,
	SkillSLStun,
	SkillSLSma,
}

var gunslingerSkillTree = []uint16{
	SkillGSGlittering,
	SkillGSSingleaction,
	SkillGSCracker,
	SkillGSMagicalbullet,
	SkillGSChainaction,
	SkillGSTracking,
	SkillGSDust,
	SkillGSSpreadattack,
	SkillGSIncreasing,
	SkillGSFling,
	SkillGSRapidshower,
	SkillGSPiercingshot,
	SkillGSFullbuster,
	SkillGSGrounddrift,
	SkillGSMadnesscancel,
	SkillGSTripleaction,
	SkillGSDesperado,
	SkillGSDisarm,
	SkillGSAdjustment,
	SkillGSGatlingfever,
	SkillGSSnakeeye,
	SkillGSBullseye,
}

var ninjaSkillTree = []uint16{
	SkillNJTobidougu,
	SkillNJTatamigaeshi,
	SkillNJNinpou,
	SkillNJSyuriken,
	SkillNJShadowjump,
	SkillNJNen,
	SkillNJKouenka,
	SkillNJHyousensou,
	SkillNJHuujin,
	SkillNJKunai,
	SkillNJKasumikiri,
	SkillNJUtsusemi,
	SkillNJKaensin,
	SkillNJSuiton,
	SkillNJRaigekisai,
	SkillNJHuuma,
	SkillNJKirikage,
	SkillNJBakuenryu,
	SkillNJHyousyouraku,
	SkillNJKamaitachi,
	SkillNJZenynage,
	SkillNJBunsinjyutsu,
	SkillNJIssen,
}

func combinedSkillTree(trees ...[]uint16) []uint16 {
	total := 0
	for _, tree := range trees {
		total += len(tree)
	}
	out := make([]uint16, 0, total)
	for _, tree := range trees {
		out = append(out, tree...)
	}
	return out
}

// SkillTreeGroup describes the skills displayed under one class-level tab.
// Transcendent skills share the second-class group, matching roBrowser's tree.
type SkillTreeGroup struct {
	ClassLevel int
	SkillIDs   []uint16
}

var noviceSkillTree = []uint16{SkillNVBasic, SkillNVFirstaid, SkillNVTrickdead}

var skillTreeGroupsByJob = map[int][][]uint16{
	JobNovice:       {noviceSkillTree},
	JobSwordman:     {swordmanSkillTree},
	JobSwordmanH:    {swordmanSkillTree},
	JobSwordmanB:    {swordmanSkillTree},
	JobKnight:       {swordmanSkillTree, knightSkillTree},
	JobKnight2:      {swordmanSkillTree, knightSkillTree},
	JobKnightH:      {swordmanSkillTree, combinedSkillTree(knightSkillTree, lordKnightSkillTree)},
	JobKnight2H:     {swordmanSkillTree, combinedSkillTree(knightSkillTree, lordKnightSkillTree)},
	JobKnightB:      {swordmanSkillTree, knightSkillTree},
	JobKnight2B:     {swordmanSkillTree, knightSkillTree},
	JobCrusader:     {swordmanSkillTree, crusaderSkillTree},
	JobCrusader2:    {swordmanSkillTree, crusaderSkillTree},
	JobCrusaderH:    {swordmanSkillTree, combinedSkillTree(crusaderSkillTree, paladinSkillTree)},
	JobCrusader2H:   {swordmanSkillTree, combinedSkillTree(crusaderSkillTree, paladinSkillTree)},
	JobCrusaderB:    {swordmanSkillTree, crusaderSkillTree},
	JobCrusader2B:   {swordmanSkillTree, crusaderSkillTree},
	JobMagician:     {magicianSkillTree},
	JobMagicianH:    {magicianSkillTree},
	JobMagicianB:    {magicianSkillTree},
	JobWizard:       {magicianSkillTree, wizardSkillTree},
	JobWizardH:      {magicianSkillTree, combinedSkillTree(wizardSkillTree, highWizardSkillTree)},
	JobWizardB:      {magicianSkillTree, wizardSkillTree},
	JobSage:         {magicianSkillTree, sageSkillTree},
	JobSageH:        {magicianSkillTree, combinedSkillTree(sageSkillTree, professorSkillTree)},
	JobSageB:        {magicianSkillTree, sageSkillTree},
	JobArcher:       {archerSkillTree},
	JobArcherH:      {archerSkillTree},
	JobArcherB:      {archerSkillTree},
	JobHunter:       {archerSkillTree, hunterSkillTree},
	JobHunterH:      {archerSkillTree, combinedSkillTree(hunterSkillTree, sniperSkillTree)},
	JobHunterB:      {archerSkillTree, hunterSkillTree},
	JobBard:         {archerSkillTree, bardSkillTree},
	JobBardH:        {archerSkillTree, combinedSkillTree(bardSkillTree, clownGypsySkillTree)},
	JobBardB:        {archerSkillTree, bardSkillTree},
	JobDancer:       {archerSkillTree, dancerSkillTree},
	JobDancerH:      {archerSkillTree, combinedSkillTree(dancerSkillTree, clownGypsySkillTree)},
	JobDancerB:      {archerSkillTree, dancerSkillTree},
	JobAcolyte:      {acolyteSkillTree},
	JobAcolyteH:     {acolyteSkillTree},
	JobAcolyteB:     {acolyteSkillTree},
	JobPriest:       {acolyteSkillTree, priestSkillTree},
	JobPriestH:      {acolyteSkillTree, combinedSkillTree(priestSkillTree, highPriestSkillTree)},
	JobPriestB:      {acolyteSkillTree, priestSkillTree},
	JobMonk:         {acolyteSkillTree, monkSkillTree},
	JobMonkH:        {acolyteSkillTree, combinedSkillTree(monkSkillTree, championSkillTree)},
	JobMonkB:        {acolyteSkillTree, monkSkillTree},
	JobMerchant:     {merchantSkillTree},
	JobMerchantH:    {merchantSkillTree},
	JobMerchantB:    {merchantSkillTree},
	JobBlacksmith:   {merchantSkillTree, blacksmithSkillTree},
	JobBlacksmithH:  {merchantSkillTree, combinedSkillTree(blacksmithSkillTree, whitesmithSkillTree)},
	JobBlacksmithB:  {merchantSkillTree, blacksmithSkillTree},
	JobAlchemist:    {merchantSkillTree, alchemistSkillTree},
	JobAlchemistH:   {merchantSkillTree, combinedSkillTree(alchemistSkillTree, creatorSkillTree)},
	JobAlchemistB:   {merchantSkillTree, alchemistSkillTree},
	JobThief:        {thiefSkillTree},
	JobThiefH:       {thiefSkillTree},
	JobThiefB:       {thiefSkillTree},
	JobAssassin:     {thiefSkillTree, assassinSkillTree},
	JobAssassinH:    {thiefSkillTree, combinedSkillTree(assassinSkillTree, assassinCrossSkillTree)},
	JobAssassinB:    {thiefSkillTree, assassinSkillTree},
	JobRogue:        {thiefSkillTree, rogueSkillTree},
	JobRogueH:       {thiefSkillTree, combinedSkillTree(rogueSkillTree, stalkerSkillTree)},
	JobRogueB:       {thiefSkillTree, rogueSkillTree},
	JobSuperNovice:  {superNoviceSkillTree},
	JobSuperNoviceB: {superNoviceSkillTree},
	JobTaekwon:      {taekwonSkillTree},
	JobTaekwonB:     {taekwonSkillTree},
	JobStar:         {taekwonSkillTree, starGladiatorSkillTree},
	JobStar2:        {taekwonSkillTree, starGladiatorSkillTree},
	JobStarB:        {taekwonSkillTree, starGladiatorSkillTree},
	JobStar2B:       {taekwonSkillTree, starGladiatorSkillTree},
	JobLinker:       {taekwonSkillTree, soulLinkerSkillTree},
	JobLinkerB:      {taekwonSkillTree, soulLinkerSkillTree},
	JobGunslinger:   {gunslingerSkillTree},
	JobGunslingerB:  {gunslingerSkillTree},
	JobNinja:        {ninjaSkillTree},
	JobNinjaB:       {ninjaSkillTree},
}

// SkillTreeSkillGroups returns the inherited skill tree split by class level.
func SkillTreeSkillGroups(job int) []SkillTreeGroup {
	jobGroups := skillTreeGroupsByJob[job]
	if job == JobNovice {
		jobGroups = nil
	}
	groupCount := len(jobGroups)
	if groupCount == 0 {
		groupCount = 1
	}
	groups := make([]SkillTreeGroup, groupCount)
	for i := range groups {
		groups[i].ClassLevel = i + 1
		if i == 0 {
			groups[i].SkillIDs = append(groups[i].SkillIDs, noviceSkillTree...)
		}
		if i < len(jobGroups) {
			groups[i].SkillIDs = append(groups[i].SkillIDs, jobGroups[i]...)
		}
	}
	return groups
}

func SkillTreeSkillIDs(job int) []uint16 {
	groups := SkillTreeSkillGroups(job)
	total := 0
	for _, group := range groups {
		total += len(group.SkillIDs)
	}
	out := make([]uint16, 0, total)
	for _, group := range groups {
		out = append(out, group.SkillIDs...)
	}
	return out
}

// SkillTreeLayoutJobs returns the client skill-tree tables which contribute
// positions to a class-level tab. Base and transcendent second classes share
// one tab, matching the classic expanded skill window.
func SkillTreeLayoutJobs(job, classLevel int) []int {
	first, second, advanced := skillTreeLineage(job)
	switch classLevel {
	case 1:
		jobs := []int{JobNovice}
		if first != JobNovice {
			jobs = append(jobs, first)
		}
		return jobs
	case 2:
		jobs := make([]int, 0, 2)
		if second >= 0 {
			jobs = append(jobs, second)
		}
		if advanced >= 0 && advanced != second {
			jobs = append(jobs, advanced)
		}
		return jobs
	default:
		return nil
	}
}

func skillTreeLineage(job int) (first, second, advanced int) {
	first, second, advanced = JobNovice, -1, -1
	switch job {
	case JobSwordman, JobSwordmanH, JobSwordmanB:
		first = JobSwordman
	case JobKnight, JobKnight2, JobKnightB, JobKnight2B:
		first, second = JobSwordman, JobKnight
	case JobKnightH, JobKnight2H:
		first, second, advanced = JobSwordman, JobKnight, JobKnightH
	case JobCrusader, JobCrusader2, JobCrusaderB, JobCrusader2B:
		first, second = JobSwordman, JobCrusader
	case JobCrusaderH, JobCrusader2H:
		first, second, advanced = JobSwordman, JobCrusader, JobCrusaderH
	case JobMagician, JobMagicianH, JobMagicianB:
		first = JobMagician
	case JobWizard, JobWizardB:
		first, second = JobMagician, JobWizard
	case JobWizardH:
		first, second, advanced = JobMagician, JobWizard, JobWizardH
	case JobSage, JobSageB:
		first, second = JobMagician, JobSage
	case JobSageH:
		first, second, advanced = JobMagician, JobSage, JobSageH
	case JobArcher, JobArcherH, JobArcherB:
		first = JobArcher
	case JobHunter, JobHunterB:
		first, second = JobArcher, JobHunter
	case JobHunterH:
		first, second, advanced = JobArcher, JobHunter, JobHunterH
	case JobBard, JobBardB:
		first, second = JobArcher, JobBard
	case JobBardH:
		first, second, advanced = JobArcher, JobBard, JobBardH
	case JobDancer, JobDancerB:
		first, second = JobArcher, JobDancer
	case JobDancerH:
		first, second, advanced = JobArcher, JobDancer, JobDancerH
	case JobAcolyte, JobAcolyteH, JobAcolyteB:
		first = JobAcolyte
	case JobPriest, JobPriestB:
		first, second = JobAcolyte, JobPriest
	case JobPriestH:
		first, second, advanced = JobAcolyte, JobPriest, JobPriestH
	case JobMonk, JobMonkB:
		first, second = JobAcolyte, JobMonk
	case JobMonkH:
		first, second, advanced = JobAcolyte, JobMonk, JobMonkH
	case JobMerchant, JobMerchantH, JobMerchantB:
		first = JobMerchant
	case JobBlacksmith, JobBlacksmithB:
		first, second = JobMerchant, JobBlacksmith
	case JobBlacksmithH:
		first, second, advanced = JobMerchant, JobBlacksmith, JobBlacksmithH
	case JobAlchemist, JobAlchemistB:
		first, second = JobMerchant, JobAlchemist
	case JobAlchemistH:
		first, second, advanced = JobMerchant, JobAlchemist, JobAlchemistH
	case JobThief, JobThiefH, JobThiefB:
		first = JobThief
	case JobAssassin, JobAssassinB:
		first, second = JobThief, JobAssassin
	case JobAssassinH:
		first, second, advanced = JobThief, JobAssassin, JobAssassinH
	case JobRogue, JobRogueB:
		first, second = JobThief, JobRogue
	case JobRogueH:
		first, second, advanced = JobThief, JobRogue, JobRogueH
	case JobSuperNovice, JobSuperNoviceB:
		first = JobSuperNovice
	case JobGunslinger, JobGunslingerB:
		first = JobGunslinger
	case JobNinja, JobNinjaB:
		first = JobNinja
	case JobTaekwon, JobTaekwonB:
		first = JobTaekwon
	case JobStar, JobStar2, JobStarB, JobStar2B:
		first, second = JobTaekwon, JobStar
	case JobLinker, JobLinkerB:
		first, second = JobTaekwon, JobLinker
	}
	return first, second, advanced
}

func SkillMaxLevel(skillID uint16) (int, bool) {
	level, ok := SkillMaxLevels[skillID]
	return level, ok && level > 0
}

func SkillRequirementsForJob(job int, skillID uint16) []SkillRequirement {
	if bySkill, ok := SkillRequirementsByJob[job]; ok {
		if requirements, ok := bySkill[skillID]; ok {
			return requirements
		}
	}
	return SkillRequirements[skillID]
}
