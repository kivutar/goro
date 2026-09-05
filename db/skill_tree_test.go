package db

import (
	"reflect"
	"testing"
)

func TestSkillTreeGroupsClassifyTranscendentSkillsAsSecondClass(t *testing.T) {
	groups := SkillTreeSkillGroups(JobKnightH)
	if len(groups) != 2 {
		t.Fatalf("lord knight groups = %d, want 2", len(groups))
	}
	if groups[0].ClassLevel != 1 || !containsSkillID(groups[0].SkillIDs, SkillNVBasic) || !containsSkillID(groups[0].SkillIDs, SkillSMBash) {
		t.Fatalf("first-class group = %+v, want novice and swordman skills", groups[0])
	}
	if containsSkillID(groups[0].SkillIDs, SkillKNPierce) {
		t.Fatalf("first-class group = %+v, should not contain knight skills", groups[0])
	}
	if groups[1].ClassLevel != 2 || !containsSkillID(groups[1].SkillIDs, SkillKNPierce) || !containsSkillID(groups[1].SkillIDs, SkillLKSpiralpierce) {
		t.Fatalf("second-class group = %+v, want knight and lord knight skills", groups[1])
	}
}

func TestSkillTreeLayoutJobsMergeInheritedClassicTables(t *testing.T) {
	tests := []struct {
		job        int
		classLevel int
		want       []int
	}{
		{JobAlchemistH, 1, []int{JobNovice, JobMerchant}},
		{JobAlchemistH, 2, []int{JobAlchemist, JobAlchemistH}},
		{JobKnight2B, 2, []int{JobKnight}},
		{JobStarB, 1, []int{JobNovice, JobTaekwon}},
		{JobStarB, 2, []int{JobStar}},
		{JobNoviceH, 1, []int{JobNovice}},
	}
	for _, tc := range tests {
		if got := SkillTreeLayoutJobs(tc.job, tc.classLevel); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("SkillTreeLayoutJobs(%d, %d) = %v, want %v", tc.job, tc.classLevel, got, tc.want)
		}
	}
}

func TestWizardSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	wizard := SkillTreeSkillIDs(JobWizard)
	if !containsSkillID(wizard, SkillMGFirebolt) || !containsSkillID(wizard, SkillWZMeteor) {
		t.Fatalf("wizard tree = %v, want magician and wizard skills", wizard)
	}
	if containsSkillID(wizard, SkillHWMagicpower) {
		t.Fatalf("wizard tree = %v, should not include high wizard skills", wizard)
	}

	babyWizard := SkillTreeSkillIDs(JobWizardB)
	if !containsSkillID(babyWizard, SkillWZStormgust) || containsSkillID(babyWizard, SkillHWMagicpower) {
		t.Fatalf("baby wizard tree = %v, want wizard duplicate without high wizard skills", babyWizard)
	}

	highWizard := SkillTreeSkillIDs(JobWizardH)
	if !containsSkillID(highWizard, SkillMGFirebolt) || !containsSkillID(highWizard, SkillWZStormgust) || !containsSkillID(highWizard, SkillHWMagicpower) {
		t.Fatalf("high wizard tree = %v, want magician, wizard, and high wizard skills", highWizard)
	}
}

func TestSageSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	sage := SkillTreeSkillIDs(JobSage)
	for _, skillID := range []uint16{
		SkillMGFirebolt,
		SkillSAAdvancedbook,
		SkillWZEstimation,
		SkillSACreatecon,
		SkillWZEarthspike,
		SkillWZHeavendrive,
		SkillSAAbracadabra,
	} {
		if !containsSkillID(sage, skillID) {
			t.Fatalf("sage tree = %v, missing robr skill %d", sage, skillID)
		}
	}
	if containsSkillID(sage, SkillPFDoublecasting) {
		t.Fatalf("sage tree = %v, should not include professor skills", sage)
	}

	babySage := SkillTreeSkillIDs(JobSageB)
	if !containsSkillID(babySage, SkillSALandprotector) || containsSkillID(babySage, SkillPFFogwall) {
		t.Fatalf("baby sage tree = %v, want sage duplicate without professor skills", babySage)
	}

	professor := SkillTreeSkillIDs(JobSageH)
	if !containsSkillID(professor, SkillMGLightningbolt) || !containsSkillID(professor, SkillSAAbracadabra) || !containsSkillID(professor, SkillPFSpiderweb) || !containsSkillID(professor, SkillPFMindbreaker) {
		t.Fatalf("professor tree = %v, want magician, sage, and professor skills", professor)
	}
}

func TestKnightSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	knight := SkillTreeSkillIDs(JobKnight)
	if !containsSkillID(knight, SkillSMBash) || !containsSkillID(knight, SkillKNPierce) || !containsSkillID(knight, SkillKNChargeatk) {
		t.Fatalf("knight tree = %v, want swordman and knight skills", knight)
	}
	if containsSkillID(knight, SkillLKSpiralpierce) {
		t.Fatalf("knight tree = %v, should not include lord knight skills", knight)
	}

	mountedKnight := SkillTreeSkillIDs(JobKnight2)
	if !containsSkillID(mountedKnight, SkillKNRiding) || !containsSkillID(mountedKnight, SkillKNBrandishspear) {
		t.Fatalf("mounted knight tree = %v, want knight duplicate skills", mountedKnight)
	}

	babyKnight := SkillTreeSkillIDs(JobKnightB)
	if !containsSkillID(babyKnight, SkillKNBowlingbash) || containsSkillID(babyKnight, SkillLKAurablade) {
		t.Fatalf("baby knight tree = %v, want knight duplicate without lord knight skills", babyKnight)
	}

	lordKnight := SkillTreeSkillIDs(JobKnightH)
	if !containsSkillID(lordKnight, SkillSMBash) || !containsSkillID(lordKnight, SkillKNBowlingbash) || !containsSkillID(lordKnight, SkillLKSpiralpierce) {
		t.Fatalf("lord knight tree = %v, want swordman, knight, and lord knight skills", lordKnight)
	}

	mountedLordKnight := SkillTreeSkillIDs(JobKnight2H)
	if !containsSkillID(mountedLordKnight, SkillLKJointbeat) || !containsSkillID(mountedLordKnight, SkillKNOnehand) {
		t.Fatalf("mounted lord knight tree = %v, want lord knight duplicate skills", mountedLordKnight)
	}
}

func TestCrusaderSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	crusader := SkillTreeSkillIDs(JobCrusader)
	for _, skillID := range []uint16{
		SkillSMBash,
		SkillCRTrust,
		SkillKNSpearmastery,
		SkillALCure,
		SkillCRHolycross,
		SkillCRProvidence,
	} {
		if !containsSkillID(crusader, skillID) {
			t.Fatalf("crusader tree = %v, missing robr skill %d", crusader, skillID)
		}
	}
	if containsSkillID(crusader, SkillPaGospel) {
		t.Fatalf("crusader tree = %v, should not include paladin skills", crusader)
	}

	mountedCrusader := SkillTreeSkillIDs(JobCrusader2)
	if !containsSkillID(mountedCrusader, SkillKNRiding) || !containsSkillID(mountedCrusader, SkillKNCavaliermastery) {
		t.Fatalf("mounted crusader tree = %v, want crusader duplicate skills", mountedCrusader)
	}

	babyCrusader := SkillTreeSkillIDs(JobCrusaderB)
	if !containsSkillID(babyCrusader, SkillCRDefender) || containsSkillID(babyCrusader, SkillPaSacrifice) {
		t.Fatalf("baby crusader tree = %v, want crusader duplicate without paladin skills", babyCrusader)
	}

	paladin := SkillTreeSkillIDs(JobCrusaderH)
	if !containsSkillID(paladin, SkillCRGrandcross) || !containsSkillID(paladin, SkillPaPressure) || !containsSkillID(paladin, SkillPaShieldchain) || !containsSkillID(paladin, SkillPaGospel) {
		t.Fatalf("paladin tree = %v, want swordman, crusader, and paladin skills", paladin)
	}

	mountedPaladin := SkillTreeSkillIDs(JobCrusader2H)
	if !containsSkillID(mountedPaladin, SkillPaSacrifice) || !containsSkillID(mountedPaladin, SkillCRSpearquicken) {
		t.Fatalf("mounted paladin tree = %v, want paladin duplicate skills", mountedPaladin)
	}
}

func TestPriestSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	acolyte := SkillTreeSkillIDs(JobAcolyteH)
	if !containsSkillID(acolyte, SkillALHeal) || !containsSkillID(acolyte, SkillALPneuma) || containsSkillID(acolyte, SkillPRKyrie) {
		t.Fatalf("high acolyte tree = %v, want acolyte skills only", acolyte)
	}

	priest := SkillTreeSkillIDs(JobPriest)
	for _, skillID := range []uint16{
		SkillALHeal,
		SkillPRKyrie,
		SkillMGSrecovery,
		SkillALLResurrection,
		SkillMGSafetywall,
		SkillPRRedemptio,
	} {
		if !containsSkillID(priest, skillID) {
			t.Fatalf("priest tree = %v, missing robr skill %d", priest, skillID)
		}
	}
	if containsSkillID(priest, SkillHPAssumptio) {
		t.Fatalf("priest tree = %v, should not include high priest skills", priest)
	}

	babyPriest := SkillTreeSkillIDs(JobPriestB)
	if !containsSkillID(babyPriest, SkillPRMagnus) || containsSkillID(babyPriest, SkillHPMeditatio) {
		t.Fatalf("baby priest tree = %v, want priest duplicate without high priest skills", babyPriest)
	}

	highPriest := SkillTreeSkillIDs(JobPriestH)
	if !containsSkillID(highPriest, SkillALHeal) || !containsSkillID(highPriest, SkillPRMagnus) || !containsSkillID(highPriest, SkillHPAssumptio) || !containsSkillID(highPriest, SkillHPManarecharge) {
		t.Fatalf("high priest tree = %v, want acolyte, priest, and high priest skills", highPriest)
	}
}

func TestMonkSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	monk := SkillTreeSkillIDs(JobMonk)
	for _, skillID := range []uint16{
		SkillALHeal,
		SkillMOIronhand,
		SkillMOCallspirits,
		SkillMOKitranslation,
		SkillMOBalkyoung,
		SkillMOBodyrelocation,
	} {
		if !containsSkillID(monk, skillID) {
			t.Fatalf("monk tree = %v, missing robr skill %d", monk, skillID)
		}
	}
	if containsSkillID(monk, SkillChSoulcollect) {
		t.Fatalf("monk tree = %v, should not include champion skills", monk)
	}

	babyMonk := SkillTreeSkillIDs(JobMonkB)
	if !containsSkillID(babyMonk, SkillMOExtremityfist) || containsSkillID(babyMonk, SkillChChaincrush) {
		t.Fatalf("baby monk tree = %v, want monk duplicate without champion skills", babyMonk)
	}

	champion := SkillTreeSkillIDs(JobMonkH)
	if !containsSkillID(champion, SkillALBlessing) || !containsSkillID(champion, SkillMOCombofinish) || !containsSkillID(champion, SkillChSoulcollect) || !containsSkillID(champion, SkillChChaincrush) {
		t.Fatalf("champion tree = %v, want acolyte, monk, and champion skills", champion)
	}
}

func TestHunterSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	archer := SkillTreeSkillIDs(JobArcherH)
	if !containsSkillID(archer, SkillACDouble) || !containsSkillID(archer, SkillACConcentration) || containsSkillID(archer, SkillHTFalcon) {
		t.Fatalf("high archer tree = %v, want archer skills only", archer)
	}

	hunter := SkillTreeSkillIDs(JobHunter)
	for _, skillID := range []uint16{
		SkillACDouble,
		SkillHTBeastbane,
		SkillHTSkidtrap,
		SkillHTPhantasmic,
		SkillHTClaymoretrap,
	} {
		if !containsSkillID(hunter, skillID) {
			t.Fatalf("hunter tree = %v, missing robr skill %d", hunter, skillID)
		}
	}
	if containsSkillID(hunter, SkillSNSharpshooting) {
		t.Fatalf("hunter tree = %v, should not include sniper skills", hunter)
	}

	babyHunter := SkillTreeSkillIDs(JobHunterB)
	if !containsSkillID(babyHunter, SkillHTBlitzbeat) || containsSkillID(babyHunter, SkillSNWindwalk) {
		t.Fatalf("baby hunter tree = %v, want hunter duplicate without sniper skills", babyHunter)
	}

	sniper := SkillTreeSkillIDs(JobHunterH)
	if !containsSkillID(sniper, SkillACVulture) || !containsSkillID(sniper, SkillHTSteelcrow) || !containsSkillID(sniper, SkillSNFalconassault) || !containsSkillID(sniper, SkillSNWindwalk) {
		t.Fatalf("sniper tree = %v, want archer, hunter, and sniper skills", sniper)
	}
}

func TestBardDancerSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	bard := SkillTreeSkillIDs(JobBard)
	for _, skillID := range []uint16{
		SkillACDouble,
		SkillBDAdaptation,
		SkillBaMusicallesson,
		SkillBaPangvoice,
		SkillBDRingnibelungen,
	} {
		if !containsSkillID(bard, skillID) {
			t.Fatalf("bard tree = %v, missing robr skill %d", bard, skillID)
		}
	}
	if containsSkillID(bard, SkillCGArrowvulcan) || containsSkillID(bard, SkillDCDancinglesson) {
		t.Fatalf("bard tree = %v, should not include clown or dancer skills", bard)
	}

	babyBard := SkillTreeSkillIDs(JobBardB)
	if !containsSkillID(babyBard, SkillBaFrostjoke) || containsSkillID(babyBard, SkillCGSpecialsinger) {
		t.Fatalf("baby bard tree = %v, want bard duplicate without clown skills", babyBard)
	}

	clown := SkillTreeSkillIDs(JobBardH)
	if !containsSkillID(clown, SkillACShower) || !containsSkillID(clown, SkillBaDissonance) || !containsSkillID(clown, SkillCGArrowvulcan) || !containsSkillID(clown, SkillCGSpecialsinger) {
		t.Fatalf("clown tree = %v, want archer, bard, and clown skills", clown)
	}

	dancer := SkillTreeSkillIDs(JobDancer)
	for _, skillID := range []uint16{
		SkillACDouble,
		SkillBDAdaptation,
		SkillDCDancinglesson,
		SkillDCWinkcharm,
		SkillBDRingnibelungen,
	} {
		if !containsSkillID(dancer, skillID) {
			t.Fatalf("dancer tree = %v, missing robr skill %d", dancer, skillID)
		}
	}
	if containsSkillID(dancer, SkillCGArrowvulcan) || containsSkillID(dancer, SkillBaMusicallesson) {
		t.Fatalf("dancer tree = %v, should not include gypsy or bard skills", dancer)
	}

	babyDancer := SkillTreeSkillIDs(JobDancerB)
	if !containsSkillID(babyDancer, SkillDCScream) || containsSkillID(babyDancer, SkillCGSpecialsinger) {
		t.Fatalf("baby dancer tree = %v, want dancer duplicate without gypsy skills", babyDancer)
	}

	gypsy := SkillTreeSkillIDs(JobDancerH)
	if !containsSkillID(gypsy, SkillACShower) || !containsSkillID(gypsy, SkillDCUglydance) || !containsSkillID(gypsy, SkillCGArrowvulcan) || !containsSkillID(gypsy, SkillCGSpecialsinger) {
		t.Fatalf("gypsy tree = %v, want archer, dancer, and gypsy skills", gypsy)
	}
}

func TestAssassinSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	thief := SkillTreeSkillIDs(JobThiefH)
	if !containsSkillID(thief, SkillTFDouble) || !containsSkillID(thief, SkillTFPickstone) || containsSkillID(thief, SkillASSonicblow) {
		t.Fatalf("high thief tree = %v, want thief skills only", thief)
	}

	assassin := SkillTreeSkillIDs(JobAssassin)
	for _, skillID := range []uint16{
		SkillTFDouble,
		SkillASRight,
		SkillASCloaking,
		SkillASVenomknife,
		SkillASSonicaccel,
		SkillASSplasher,
	} {
		if !containsSkillID(assassin, skillID) {
			t.Fatalf("assassin tree = %v, missing robr skill %d", assassin, skillID)
		}
	}
	if containsSkillID(assassin, SkillASCBreaker) {
		t.Fatalf("assassin tree = %v, should not include assassin cross skills", assassin)
	}

	babyAssassin := SkillTreeSkillIDs(JobAssassinB)
	if !containsSkillID(babyAssassin, SkillASGrimtooth) || containsSkillID(babyAssassin, SkillASCEdp) {
		t.Fatalf("baby assassin tree = %v, want assassin duplicate without assassin cross skills", babyAssassin)
	}

	assassinCross := SkillTreeSkillIDs(JobAssassinH)
	if !containsSkillID(assassinCross, SkillTFPoison) || !containsSkillID(assassinCross, SkillASKatar) || !containsSkillID(assassinCross, SkillASCBreaker) || !containsSkillID(assassinCross, SkillASCMeteorassault) {
		t.Fatalf("assassin cross tree = %v, want thief, assassin, and assassin cross skills", assassinCross)
	}
}

func TestRogueSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	rogue := SkillTreeSkillIDs(JobRogue)
	for _, skillID := range []uint16{
		SkillTFSteal,
		SkillACVulture,
		SkillRGTunneldrive,
		SkillSMSword,
		SkillACDouble,
		SkillHTRemovetrap,
		SkillRGPlagiarism,
	} {
		if !containsSkillID(rogue, skillID) {
			t.Fatalf("rogue tree = %v, missing robr skill %d", rogue, skillID)
		}
	}
	if containsSkillID(rogue, SkillSTChasewalk) {
		t.Fatalf("rogue tree = %v, should not include stalker skills", rogue)
	}

	babyRogue := SkillTreeSkillIDs(JobRogueB)
	if !containsSkillID(babyRogue, SkillRGCloseconfine) || containsSkillID(babyRogue, SkillSTFullstrip) {
		t.Fatalf("baby rogue tree = %v, want rogue duplicate without stalker skills", babyRogue)
	}

	stalker := SkillTreeSkillIDs(JobRogueH)
	if !containsSkillID(stalker, SkillTFPoison) || !containsSkillID(stalker, SkillRGIntimidate) || !containsSkillID(stalker, SkillSTChasewalk) || !containsSkillID(stalker, SkillSTPreserve) {
		t.Fatalf("stalker tree = %v, want thief, rogue, and stalker skills", stalker)
	}
	stalkerTrans := stalker[len(SkillTreeSkillIDs(JobRogue)):]
	wantStalkerTrans := []uint16{SkillSTChasewalk, SkillSTFullstrip, SkillSTPreserve, SkillSTRejectsword}
	if !reflect.DeepEqual(stalkerTrans, wantStalkerTrans) {
		t.Fatalf("stalker trans tree = %v, want robr order %v", stalkerTrans, wantStalkerTrans)
	}
}

func TestBlacksmithSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	merchant := SkillTreeSkillIDs(JobMerchantH)
	if !containsSkillID(merchant, SkillMCInccarry) || !containsSkillID(merchant, SkillMCCartdecorate) || containsSkillID(merchant, SkillBSHammerfall) {
		t.Fatalf("high merchant tree = %v, want merchant skills only", merchant)
	}

	blacksmith := SkillTreeSkillIDs(JobBlacksmith)
	for _, skillID := range []uint16{
		SkillMCInccarry,
		SkillBSIron,
		SkillBSHiltbinding,
		SkillBSAdrenaline,
		SkillBSMaximize,
		SkillBSAdrenaline2,
		SkillBSGreed,
		SkillBSUnfairlytrick,
	} {
		if !containsSkillID(blacksmith, skillID) {
			t.Fatalf("blacksmith tree = %v, missing robr skill %d", blacksmith, skillID)
		}
	}
	if containsSkillID(blacksmith, SkillWSMeltdown) {
		t.Fatalf("blacksmith tree = %v, should not include whitesmith skills", blacksmith)
	}

	babyBlacksmith := SkillTreeSkillIDs(JobBlacksmithB)
	if !containsSkillID(babyBlacksmith, SkillBSWeaponperfect) || containsSkillID(babyBlacksmith, SkillWSCarttermination) {
		t.Fatalf("baby blacksmith tree = %v, want blacksmith duplicate without whitesmith skills", babyBlacksmith)
	}

	whitesmith := SkillTreeSkillIDs(JobBlacksmithH)
	if !containsSkillID(whitesmith, SkillMCMammonite) || !containsSkillID(whitesmith, SkillBSWeaponresearch) || !containsSkillID(whitesmith, SkillWSMeltdown) || !containsSkillID(whitesmith, SkillWSWeaponrefine) {
		t.Fatalf("whitesmith tree = %v, want merchant, blacksmith, and whitesmith skills", whitesmith)
	}
}

func TestAlchemistSkillTreeIncludesRobrowserBeforeJobs(t *testing.T) {
	merchant := SkillTreeSkillIDs(JobMerchantH)
	if !containsSkillID(merchant, SkillMCInccarry) || !containsSkillID(merchant, SkillMCCartdecorate) || containsSkillID(merchant, SkillAMPharmacy) {
		t.Fatalf("high merchant tree = %v, want merchant skills only", merchant)
	}

	alchemist := SkillTreeSkillIDs(JobAlchemist)
	for _, skillID := range []uint16{
		SkillMCInccarry,
		SkillAMLearningpotion,
		SkillAMSpheremine,
		SkillAMBioethics,
		SkillAMDemonstration,
		SkillAMResurrecthomun,
		SkillAMCannibalize,
	} {
		if !containsSkillID(alchemist, skillID) {
			t.Fatalf("alchemist tree = %v, missing robr skill %d", alchemist, skillID)
		}
	}
	if containsSkillID(alchemist, SkillCRFullprotection) {
		t.Fatalf("alchemist tree = %v, should not include creator skills", alchemist)
	}

	babyAlchemist := SkillTreeSkillIDs(JobAlchemistB)
	if !containsSkillID(babyAlchemist, SkillAMCpWeapon) || containsSkillID(babyAlchemist, SkillCRSlimpitcher) {
		t.Fatalf("baby alchemist tree = %v, want alchemist duplicate without creator skills", babyAlchemist)
	}

	creator := SkillTreeSkillIDs(JobAlchemistH)
	if !containsSkillID(creator, SkillMCPushcart) || !containsSkillID(creator, SkillAMAcidterror) || !containsSkillID(creator, SkillCRAciddemonstration) || !containsSkillID(creator, SkillCRFullprotection) {
		t.Fatalf("creator tree = %v, want merchant, alchemist, and creator skills", creator)
	}
}

func TestWizardSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillWZFirepillar, []SkillRequirement{{SkillID: SkillMGFirewall, Level: 1}}},
		{SkillWZSightrasher, []SkillRequirement{{SkillID: SkillMGSight, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}}},
		{SkillWZMeteor, []SkillRequirement{{SkillID: SkillMGThunderstorm, Level: 1}, {SkillID: SkillWZSightrasher, Level: 2}}},
		{SkillWZJupitel, []SkillRequirement{{SkillID: SkillMGNapalmbeat, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}}},
		{SkillWZVermilion, []SkillRequirement{{SkillID: SkillMGThunderstorm, Level: 1}, {SkillID: SkillWZJupitel, Level: 5}}},
		{SkillWZWaterball, []SkillRequirement{{SkillID: SkillMGColdbolt, Level: 1}, {SkillID: SkillMGLightningbolt, Level: 1}}},
		{SkillWZIcewall, []SkillRequirement{{SkillID: SkillMGStonecurse, Level: 1}, {SkillID: SkillMGFrostdiver, Level: 1}}},
		{SkillWZFrostnova, []SkillRequirement{{SkillID: SkillWZIcewall, Level: 1}}},
		{SkillWZStormgust, []SkillRequirement{{SkillID: SkillMGFrostdiver, Level: 1}, {SkillID: SkillWZJupitel, Level: 3}}},
		{SkillWZEarthspike, []SkillRequirement{{SkillID: SkillMGStonecurse, Level: 1}}},
		{SkillWZHeavendrive, []SkillRequirement{{SkillID: SkillWZEarthspike, Level: 3}}},
		{SkillWZQuagmire, []SkillRequirement{{SkillID: SkillWZHeavendrive, Level: 1}}},
		{SkillHWSouldrain, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 5}, {SkillID: SkillMGSoulstrike, Level: 7}}},
		{SkillHWMagiccrasher, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 1}}},
		{SkillHWNapalmvulcan, []SkillRequirement{{SkillID: SkillMGNapalmbeat, Level: 5}}},
		{SkillHWGanbantein, []SkillRequirement{{SkillID: SkillWZEstimation, Level: 1}, {SkillID: SkillWZIcewall, Level: 1}}},
		{SkillHWGravitation, []SkillRequirement{{SkillID: SkillWZQuagmire, Level: 1}, {SkillID: SkillHWMagiccrasher, Level: 1}, {SkillID: SkillHWMagicpower, Level: 10}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestSageSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillSACastcancel, []SkillRequirement{{SkillID: SkillSAAdvancedbook, Level: 2}}},
		{SkillSAMagicrod, []SkillRequirement{{SkillID: SkillSAAdvancedbook, Level: 4}}},
		{SkillSASpellbreaker, []SkillRequirement{{SkillID: SkillSAMagicrod, Level: 1}}},
		{SkillSAFreecast, []SkillRequirement{{SkillID: SkillSACastcancel, Level: 1}}},
		{SkillSAAutospell, []SkillRequirement{{SkillID: SkillSAFreecast, Level: 4}}},
		{SkillSAFlamelauncher, []SkillRequirement{{SkillID: SkillMGFirebolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}}},
		{SkillSAFrostweapon, []SkillRequirement{{SkillID: SkillMGColdbolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}}},
		{SkillSALightningloader, []SkillRequirement{{SkillID: SkillMGLightningbolt, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}}},
		{SkillSASeismicweapon, []SkillRequirement{{SkillID: SkillMGStonecurse, Level: 1}, {SkillID: SkillSAAdvancedbook, Level: 5}}},
		{SkillSADragonology, []SkillRequirement{{SkillID: SkillSAAdvancedbook, Level: 9}}},
		{SkillSAVolcano, []SkillRequirement{{SkillID: SkillSAFlamelauncher, Level: 2}}},
		{SkillSADeluge, []SkillRequirement{{SkillID: SkillSAFrostweapon, Level: 2}}},
		{SkillSAViolentgale, []SkillRequirement{{SkillID: SkillSALightningloader, Level: 2}}},
		{SkillSALandprotector, []SkillRequirement{{SkillID: SkillSADeluge, Level: 3}, {SkillID: SkillSAViolentgale, Level: 3}, {SkillID: SkillSAVolcano, Level: 3}}},
		{SkillSADispell, []SkillRequirement{{SkillID: SkillSASpellbreaker, Level: 3}}},
		{SkillSAAbracadabra, []SkillRequirement{{SkillID: SkillSAAutospell, Level: 5}, {SkillID: SkillSADispell, Level: 1}, {SkillID: SkillSALandprotector, Level: 1}}},
		{SkillPFHpconversion, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 1}, {SkillID: SkillSAMagicrod, Level: 1}}},
		{SkillPFSoulchange, []SkillRequirement{{SkillID: SkillSAMagicrod, Level: 3}, {SkillID: SkillSASpellbreaker, Level: 2}}},
		{SkillPFSoulburn, []SkillRequirement{{SkillID: SkillSACastcancel, Level: 5}, {SkillID: SkillSAMagicrod, Level: 3}, {SkillID: SkillSADispell, Level: 3}}},
		{SkillPFMindbreaker, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 3}, {SkillID: SkillPFSoulburn, Level: 2}}},
		{SkillPFMemorize, []SkillRequirement{{SkillID: SkillSAAdvancedbook, Level: 5}, {SkillID: SkillSAFreecast, Level: 5}, {SkillID: SkillSAAutospell, Level: 1}}},
		{SkillPFFogwall, []SkillRequirement{{SkillID: SkillSAViolentgale, Level: 2}, {SkillID: SkillSADeluge, Level: 2}}},
		{SkillPFSpiderweb, []SkillRequirement{{SkillID: SkillSADragonology, Level: 4}}},
		{SkillPFDoublecasting, []SkillRequirement{{SkillID: SkillSAAutospell, Level: 1}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}

	for _, tc := range []struct {
		job     int
		skillID uint16
		want    []SkillRequirement
	}{
		{JobWizard, SkillWZEarthspike, []SkillRequirement{{SkillID: SkillMGStonecurse, Level: 1}}},
		{JobSage, SkillWZEarthspike, []SkillRequirement{{SkillID: SkillSASeismicweapon, Level: 1}}},
		{JobSageH, SkillWZEarthspike, []SkillRequirement{{SkillID: SkillSASeismicweapon, Level: 1}}},
		{JobSageB, SkillWZHeavendrive, []SkillRequirement{{SkillID: SkillWZEarthspike, Level: 1}}},
	} {
		if got := SkillRequirementsForJob(tc.job, tc.skillID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for job %d skill %d = %+v, want %+v", tc.job, tc.skillID, got, tc.want)
		}
	}
}

func TestKnightSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillKNPierce, []SkillRequirement{{SkillID: SkillKNSpearmastery, Level: 1}}},
		{SkillKNBrandishspear, []SkillRequirement{{SkillID: SkillKNRiding, Level: 1}, {SkillID: SkillKNSpearstab, Level: 3}}},
		{SkillKNSpearstab, []SkillRequirement{{SkillID: SkillKNPierce, Level: 5}}},
		{SkillKNSpearboomerang, []SkillRequirement{{SkillID: SkillKNPierce, Level: 3}}},
		{SkillKNTwohandquicken, []SkillRequirement{{SkillID: SkillSMTwohand, Level: 1}}},
		{SkillKNAutocounter, []SkillRequirement{{SkillID: SkillSMTwohand, Level: 1}}},
		{SkillKNBowlingbash, []SkillRequirement{
			{SkillID: SkillSMBash, Level: 10},
			{SkillID: SkillSMMagnum, Level: 3},
			{SkillID: SkillSMTwohand, Level: 5},
			{SkillID: SkillKNTwohandquicken, Level: 10},
			{SkillID: SkillKNAutocounter, Level: 5},
		}},
		{SkillKNRiding, []SkillRequirement{{SkillID: SkillSMEndure, Level: 1}}},
		{SkillKNCavaliermastery, []SkillRequirement{{SkillID: SkillKNRiding, Level: 1}}},
		{SkillKNOnehand, []SkillRequirement{{SkillID: SkillKNTwohandquicken, Level: 10}}},
		{SkillLKSpiralpierce, []SkillRequirement{
			{SkillID: SkillKNSpearmastery, Level: 5},
			{SkillID: SkillKNPierce, Level: 5},
			{SkillID: SkillKNRiding, Level: 1},
			{SkillID: SkillKNSpearstab, Level: 5},
		}},
		{SkillLKHeadcrush, []SkillRequirement{{SkillID: SkillKNSpearmastery, Level: 9}, {SkillID: SkillKNRiding, Level: 1}}},
		{SkillLKJointbeat, []SkillRequirement{{SkillID: SkillKNCavaliermastery, Level: 3}, {SkillID: SkillLKHeadcrush, Level: 3}}},
		{SkillLKAurablade, []SkillRequirement{{SkillID: SkillSMMagnum, Level: 5}, {SkillID: SkillSMTwohand, Level: 5}}},
		{SkillLKParrying, []SkillRequirement{{SkillID: SkillSMProvoke, Level: 5}, {SkillID: SkillSMTwohand, Level: 10}, {SkillID: SkillKNTwohandquicken, Level: 3}}},
		{SkillLKConcentration, []SkillRequirement{{SkillID: SkillSMRecovery, Level: 5}, {SkillID: SkillKNSpearmastery, Level: 5}, {SkillID: SkillKNRiding, Level: 1}}},
		{SkillLKTensionrelax, []SkillRequirement{{SkillID: SkillSMProvoke, Level: 5}, {SkillID: SkillSMRecovery, Level: 10}, {SkillID: SkillSMEndure, Level: 3}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestCrusaderSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		job     int
		skillID uint16
		want    []SkillRequirement
	}{
		{JobAcolyte, SkillALCure, []SkillRequirement{{SkillID: SkillALHeal, Level: 2}}},
		{JobCrusader, SkillALCure, []SkillRequirement{{SkillID: SkillCRTrust, Level: 5}}},
		{JobCrusaderH, SkillALDp, []SkillRequirement{{SkillID: SkillALCure, Level: 1}}},
		{JobCrusader2B, SkillALHeal, []SkillRequirement{{SkillID: SkillCRTrust, Level: 10}, {SkillID: SkillALDemonbane, Level: 5}}},
		{JobCrusader, SkillCRShieldcharge, []SkillRequirement{{SkillID: SkillCRAutoguard, Level: 5}}},
		{JobCrusader, SkillCRShieldboomerang, []SkillRequirement{{SkillID: SkillCRShieldcharge, Level: 3}}},
		{JobCrusader, SkillCRReflectshield, []SkillRequirement{{SkillID: SkillCRShieldboomerang, Level: 3}}},
		{JobCrusader, SkillCRHolycross, []SkillRequirement{{SkillID: SkillCRTrust, Level: 7}}},
		{JobCrusader, SkillCRGrandcross, []SkillRequirement{{SkillID: SkillCRTrust, Level: 10}, {SkillID: SkillCRHolycross, Level: 6}}},
		{JobCrusader, SkillCRDevotion, []SkillRequirement{{SkillID: SkillCRGrandcross, Level: 4}, {SkillID: SkillCRReflectshield, Level: 5}}},
		{JobCrusader, SkillCRProvidence, []SkillRequirement{{SkillID: SkillALDp, Level: 5}, {SkillID: SkillALHeal, Level: 5}}},
		{JobCrusader, SkillCRDefender, []SkillRequirement{{SkillID: SkillCRShieldboomerang, Level: 1}}},
		{JobCrusader, SkillCRSpearquicken, []SkillRequirement{{SkillID: SkillKNSpearmastery, Level: 10}}},
		{JobCrusaderH, SkillPaPressure, []SkillRequirement{{SkillID: SkillSMEndure, Level: 5}, {SkillID: SkillCRTrust, Level: 5}, {SkillID: SkillCRShieldcharge, Level: 2}}},
		{JobCrusaderH, SkillPaShieldchain, []SkillRequirement{{SkillID: SkillCRShieldboomerang, Level: 5}}},
		{JobCrusaderH, SkillPaSacrifice, []SkillRequirement{{SkillID: SkillSMEndure, Level: 1}, {SkillID: SkillCRTrust, Level: 5}, {SkillID: SkillCRDevotion, Level: 3}}},
		{JobCrusaderH, SkillPaGospel, []SkillRequirement{{SkillID: SkillCRTrust, Level: 8}, {SkillID: SkillALDp, Level: 3}, {SkillID: SkillALDemonbane, Level: 5}}},
	} {
		if got := SkillRequirementsForJob(tc.job, tc.skillID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for job %d skill %d = %+v, want %+v", tc.job, tc.skillID, got, tc.want)
		}
	}
}

func TestPriestSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		job     int
		skillID uint16
		want    []SkillRequirement
	}{
		{JobWizard, SkillMGSafetywall, []SkillRequirement{{SkillID: SkillMGNapalmbeat, Level: 7}, {SkillID: SkillMGSoulstrike, Level: 5}}},
		{JobPriest, SkillMGSafetywall, []SkillRequirement{{SkillID: SkillPRSanctuary, Level: 3}, {SkillID: SkillPRAspersio, Level: 4}}},
		{JobPriestH, SkillMGSafetywall, []SkillRequirement{{SkillID: SkillPRSanctuary, Level: 3}, {SkillID: SkillPRAspersio, Level: 4}}},
		{JobPriest, SkillALLResurrection, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 4}, {SkillID: SkillPRStrecovery, Level: 1}}},
		{JobPriest, SkillPRSuffragium, []SkillRequirement{{SkillID: SkillPRImpositio, Level: 2}}},
		{JobPriest, SkillPRAspersio, []SkillRequirement{{SkillID: SkillALHolywater, Level: 1}, {SkillID: SkillPRImpositio, Level: 3}}},
		{JobPriest, SkillPRBenedictio, []SkillRequirement{{SkillID: SkillPRAspersio, Level: 5}, {SkillID: SkillPRGloria, Level: 3}}},
		{JobPriest, SkillPRSanctuary, []SkillRequirement{{SkillID: SkillALHeal, Level: 1}}},
		{JobPriest, SkillPRKyrie, []SkillRequirement{{SkillID: SkillALAngelus, Level: 2}}},
		{JobPriest, SkillPRGloria, []SkillRequirement{{SkillID: SkillPRKyrie, Level: 4}, {SkillID: SkillPRMagnificat, Level: 3}}},
		{JobPriest, SkillPRLexdivina, []SkillRequirement{{SkillID: SkillALRuwach, Level: 1}}},
		{JobPriest, SkillPRTurnundead, []SkillRequirement{{SkillID: SkillALLResurrection, Level: 1}, {SkillID: SkillPRLexdivina, Level: 3}}},
		{JobPriest, SkillPRLexaeterna, []SkillRequirement{{SkillID: SkillPRLexdivina, Level: 5}}},
		{JobPriest, SkillPRMagnus, []SkillRequirement{{SkillID: SkillMGSafetywall, Level: 1}, {SkillID: SkillPRLexaeterna, Level: 1}, {SkillID: SkillPRTurnundead, Level: 3}}},
		{JobPriestH, SkillHPManarecharge, []SkillRequirement{{SkillID: SkillPRMacemastery, Level: 10}, {SkillID: SkillALDemonbane, Level: 10}}},
		{JobPriestH, SkillHPAssumptio, []SkillRequirement{{SkillID: SkillALAngelus, Level: 1}, {SkillID: SkillMGSrecovery, Level: 3}, {SkillID: SkillPRImpositio, Level: 3}}},
		{JobPriestH, SkillHPBasilica, []SkillRequirement{{SkillID: SkillPRGloria, Level: 2}, {SkillID: SkillMGSrecovery, Level: 1}, {SkillID: SkillPRKyrie, Level: 3}}},
		{JobPriestH, SkillHPMeditatio, []SkillRequirement{{SkillID: SkillMGSrecovery, Level: 5}, {SkillID: SkillPRLexdivina, Level: 5}, {SkillID: SkillPRAspersio, Level: 3}}},
	} {
		if got := SkillRequirementsForJob(tc.job, tc.skillID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for job %d skill %d = %+v, want %+v", tc.job, tc.skillID, got, tc.want)
		}
	}
}

func TestMonkSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillMOIronhand, []SkillRequirement{{SkillID: SkillALDemonbane, Level: 10}, {SkillID: SkillALDp, Level: 10}}},
		{SkillMOSpiritsrecovery, []SkillRequirement{{SkillID: SkillMOBladestop, Level: 2}}},
		{SkillMOCallspirits, []SkillRequirement{{SkillID: SkillMOIronhand, Level: 2}}},
		{SkillMOAbsorbspirits, []SkillRequirement{{SkillID: SkillMOCallspirits, Level: 5}}},
		{SkillMOTripleattack, []SkillRequirement{{SkillID: SkillMODodge, Level: 5}}},
		{SkillMOBodyrelocation, []SkillRequirement{{SkillID: SkillMOSpiritsrecovery, Level: 2}, {SkillID: SkillMOExtremityfist, Level: 3}, {SkillID: SkillMOSteelbody, Level: 3}}},
		{SkillMODodge, []SkillRequirement{{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOCallspirits, Level: 5}}},
		{SkillMOInvestigate, []SkillRequirement{{SkillID: SkillMOCallspirits, Level: 5}}},
		{SkillMOFingeroffensive, []SkillRequirement{{SkillID: SkillMOInvestigate, Level: 3}}},
		{SkillMOSteelbody, []SkillRequirement{{SkillID: SkillMOCombofinish, Level: 3}}},
		{SkillMOBladestop, []SkillRequirement{{SkillID: SkillMODodge, Level: 5}}},
		{SkillMOExplosionspirits, []SkillRequirement{{SkillID: SkillMOAbsorbspirits, Level: 1}}},
		{SkillMOExtremityfist, []SkillRequirement{{SkillID: SkillMOExplosionspirits, Level: 3}, {SkillID: SkillMOFingeroffensive, Level: 3}}},
		{SkillMOChaincombo, []SkillRequirement{{SkillID: SkillMOTripleattack, Level: 5}}},
		{SkillMOCombofinish, []SkillRequirement{{SkillID: SkillMOChaincombo, Level: 3}}},
		{SkillChSoulcollect, []SkillRequirement{{SkillID: SkillMOExplosionspirits, Level: 5}}},
		{SkillChPalmstrike, []SkillRequirement{{SkillID: SkillMOIronhand, Level: 7}, {SkillID: SkillMOCallspirits, Level: 5}}},
		{SkillChTigerfist, []SkillRequirement{{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOTripleattack, Level: 5}, {SkillID: SkillMOCombofinish, Level: 3}}},
		{SkillChChaincrush, []SkillRequirement{{SkillID: SkillMOIronhand, Level: 5}, {SkillID: SkillMOCallspirits, Level: 5}, {SkillID: SkillChTigerfist, Level: 2}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestHunterSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillHTPower, []SkillRequirement{{SkillID: SkillACDouble, Level: 10}}},
		{SkillHTAnklesnare, []SkillRequirement{{SkillID: SkillHTSkidtrap, Level: 1}}},
		{SkillHTShockwave, []SkillRequirement{{SkillID: SkillHTAnklesnare, Level: 1}}},
		{SkillHTSandman, []SkillRequirement{{SkillID: SkillHTFlasher, Level: 1}}},
		{SkillHTFlasher, []SkillRequirement{{SkillID: SkillHTSkidtrap, Level: 1}}},
		{SkillHTFreezingtrap, []SkillRequirement{{SkillID: SkillHTFlasher, Level: 1}}},
		{SkillHTBlastmine, []SkillRequirement{{SkillID: SkillHTLandmine, Level: 1}, {SkillID: SkillHTSandman, Level: 1}, {SkillID: SkillHTFreezingtrap, Level: 1}}},
		{SkillHTClaymoretrap, []SkillRequirement{{SkillID: SkillHTShockwave, Level: 1}, {SkillID: SkillHTBlastmine, Level: 1}}},
		{SkillHTRemovetrap, []SkillRequirement{{SkillID: SkillHTLandmine, Level: 1}}},
		{SkillHTTalkiebox, []SkillRequirement{{SkillID: SkillHTRemovetrap, Level: 1}, {SkillID: SkillHTShockwave, Level: 1}}},
		{SkillHTFalcon, []SkillRequirement{{SkillID: SkillHTBeastbane, Level: 1}}},
		{SkillHTSteelcrow, []SkillRequirement{{SkillID: SkillHTBlitzbeat, Level: 5}}},
		{SkillHTBlitzbeat, []SkillRequirement{{SkillID: SkillHTFalcon, Level: 1}}},
		{SkillHTDetecting, []SkillRequirement{{SkillID: SkillACConcentration, Level: 1}, {SkillID: SkillHTFalcon, Level: 1}}},
		{SkillHTSpringtrap, []SkillRequirement{{SkillID: SkillHTFalcon, Level: 1}, {SkillID: SkillHTRemovetrap, Level: 1}}},
		{SkillSNSight, []SkillRequirement{{SkillID: SkillACOwl, Level: 10}, {SkillID: SkillACVulture, Level: 10}, {SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillHTFalcon, Level: 1}}},
		{SkillSNFalconassault, []SkillRequirement{{SkillID: SkillACVulture, Level: 5}, {SkillID: SkillHTFalcon, Level: 1}, {SkillID: SkillHTBlitzbeat, Level: 5}, {SkillID: SkillHTSteelcrow, Level: 3}}},
		{SkillSNSharpshooting, []SkillRequirement{{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACConcentration, Level: 10}}},
		{SkillSNWindwalk, []SkillRequirement{{SkillID: SkillACConcentration, Level: 9}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestBardDancerSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		job     int
		skillID uint16
		want    []SkillRequirement
	}{
		{JobBard, SkillBDEncore, []SkillRequirement{{SkillID: SkillBDAdaptation, Level: 1}}},
		{JobBard, SkillBDRichmankim, []SkillRequirement{{SkillID: SkillBDSiegfried, Level: 3}}},
		{JobBard, SkillBDEternalchaos, []SkillRequirement{{SkillID: SkillBDRokisweil, Level: 1}}},
		{JobBard, SkillBDRingnibelungen, []SkillRequirement{{SkillID: SkillBDDrumbattlefield, Level: 3}}},
		{JobBard, SkillBDIntoabyss, []SkillRequirement{{SkillID: SkillBDLullaby, Level: 1}}},
		{JobBard, SkillBDLullaby, []SkillRequirement{{SkillID: SkillBaWhistle, Level: 10}}},
		{JobDancer, SkillBDLullaby, []SkillRequirement{{SkillID: SkillDCHumming, Level: 10}}},
		{JobBard, SkillBDDrumbattlefield, []SkillRequirement{{SkillID: SkillBaAppleidun, Level: 10}}},
		{JobDancer, SkillBDDrumbattlefield, []SkillRequirement{{SkillID: SkillDCServiceforyou, Level: 10}}},
		{JobBard, SkillBDRokisweil, []SkillRequirement{{SkillID: SkillBaAssassincross, Level: 10}}},
		{JobDancer, SkillBDRokisweil, []SkillRequirement{{SkillID: SkillDCDontforgetme, Level: 10}}},
		{JobBard, SkillBDSiegfried, []SkillRequirement{{SkillID: SkillBaPoembragi, Level: 10}}},
		{JobDancer, SkillBDSiegfried, []SkillRequirement{{SkillID: SkillDCFortunekiss, Level: 10}}},
		{JobBard, SkillBaMusicalstrike, []SkillRequirement{{SkillID: SkillBaMusicallesson, Level: 3}}},
		{JobBard, SkillBaDissonance, []SkillRequirement{{SkillID: SkillBDAdaptation, Level: 1}, {SkillID: SkillBaMusicallesson, Level: 1}}},
		{JobBard, SkillBaFrostjoke, []SkillRequirement{{SkillID: SkillBDEncore, Level: 1}}},
		{JobBard, SkillBaWhistle, []SkillRequirement{{SkillID: SkillBaDissonance, Level: 3}}},
		{JobBard, SkillBaAssassincross, []SkillRequirement{{SkillID: SkillBaDissonance, Level: 3}}},
		{JobBard, SkillBaPoembragi, []SkillRequirement{{SkillID: SkillBaDissonance, Level: 3}}},
		{JobBard, SkillBaAppleidun, []SkillRequirement{{SkillID: SkillBaDissonance, Level: 3}}},
		{JobDancer, SkillDCThrowarrow, []SkillRequirement{{SkillID: SkillDCDancinglesson, Level: 3}}},
		{JobDancer, SkillDCUglydance, []SkillRequirement{{SkillID: SkillBDAdaptation, Level: 1}, {SkillID: SkillDCDancinglesson, Level: 1}}},
		{JobDancer, SkillDCScream, []SkillRequirement{{SkillID: SkillBDEncore, Level: 1}}},
		{JobDancer, SkillDCHumming, []SkillRequirement{{SkillID: SkillDCUglydance, Level: 3}}},
		{JobDancer, SkillDCDontforgetme, []SkillRequirement{{SkillID: SkillDCUglydance, Level: 3}}},
		{JobDancer, SkillDCFortunekiss, []SkillRequirement{{SkillID: SkillDCUglydance, Level: 3}}},
		{JobDancer, SkillDCServiceforyou, []SkillRequirement{{SkillID: SkillDCUglydance, Level: 3}}},
		{JobBardH, SkillCGArrowvulcan, []SkillRequirement{{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACShower, Level: 5}, {SkillID: SkillBaMusicalstrike, Level: 1}}},
		{JobDancerH, SkillCGArrowvulcan, []SkillRequirement{{SkillID: SkillACDouble, Level: 5}, {SkillID: SkillACShower, Level: 5}, {SkillID: SkillDCThrowarrow, Level: 1}}},
		{JobBardH, SkillCGMoonlit, []SkillRequirement{{SkillID: SkillACConcentration, Level: 5}, {SkillID: SkillBaMusicallesson, Level: 7}}},
		{JobDancerH, SkillCGMoonlit, []SkillRequirement{{SkillID: SkillACConcentration, Level: 5}, {SkillID: SkillDCDancinglesson, Level: 7}}},
		{JobBardH, SkillCGMarionette, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaMusicallesson, Level: 5}}},
		{JobDancerH, SkillCGMarionette, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCDancinglesson, Level: 5}}},
		{JobBardH, SkillCGHermode, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaMusicallesson, Level: 10}}},
		{JobDancerH, SkillCGHermode, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCDancinglesson, Level: 10}}},
		{JobBardH, SkillCGTarotcard, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillBaDissonance, Level: 3}}},
		{JobDancerH, SkillCGTarotcard, []SkillRequirement{{SkillID: SkillACConcentration, Level: 10}, {SkillID: SkillDCUglydance, Level: 3}}},
		{JobBardH, SkillCGLongingfreedom, []SkillRequirement{{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillBaDissonance, Level: 3}, {SkillID: SkillBaMusicallesson, Level: 10}}},
		{JobDancerH, SkillCGLongingfreedom, []SkillRequirement{{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillDCUglydance, Level: 3}, {SkillID: SkillDCDancinglesson, Level: 10}}},
		{JobBardH, SkillCGSpecialsinger, []SkillRequirement{{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillBaDissonance, Level: 3}, {SkillID: SkillBaMusicallesson, Level: 10}}},
		{JobDancerH, SkillCGSpecialsinger, []SkillRequirement{{SkillID: SkillCGMarionette, Level: 1}, {SkillID: SkillDCUglydance, Level: 3}, {SkillID: SkillDCDancinglesson, Level: 10}}},
	} {
		if got := SkillRequirementsForJob(tc.job, tc.skillID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for job %d skill %d = %+v, want %+v", tc.job, tc.skillID, got, tc.want)
		}
	}
}

func TestAssassinSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillASLeft, []SkillRequirement{{SkillID: SkillASRight, Level: 2}}},
		{SkillASCloaking, []SkillRequirement{{SkillID: SkillTFHiding, Level: 2}}},
		{SkillASSonicblow, []SkillRequirement{{SkillID: SkillASKatar, Level: 4}}},
		{SkillASGrimtooth, []SkillRequirement{{SkillID: SkillASCloaking, Level: 2}, {SkillID: SkillASSonicblow, Level: 5}}},
		{SkillASEnchantpoison, []SkillRequirement{{SkillID: SkillTFPoison, Level: 1}}},
		{SkillASPoisonreact, []SkillRequirement{{SkillID: SkillASEnchantpoison, Level: 3}}},
		{SkillASVenomdust, []SkillRequirement{{SkillID: SkillASEnchantpoison, Level: 5}}},
		{SkillASSplasher, []SkillRequirement{{SkillID: SkillASVenomdust, Level: 5}, {SkillID: SkillASPoisonreact, Level: 5}}},
		{SkillASCKatar, []SkillRequirement{{SkillID: SkillTFDouble, Level: 5}, {SkillID: SkillASKatar, Level: 7}}},
		{SkillASCEdp, []SkillRequirement{{SkillID: SkillASCCdp, Level: 1}}},
		{SkillASCBreaker, []SkillRequirement{{SkillID: SkillTFDouble, Level: 5}, {SkillID: SkillTFPoison, Level: 5}, {SkillID: SkillASCloaking, Level: 3}, {SkillID: SkillASEnchantpoison, Level: 6}}},
		{SkillASCMeteorassault, []SkillRequirement{{SkillID: SkillASKatar, Level: 5}, {SkillID: SkillASRight, Level: 3}, {SkillID: SkillASSonicblow, Level: 5}, {SkillID: SkillASCBreaker, Level: 1}}},
		{SkillASCCdp, []SkillRequirement{{SkillID: SkillTFPoison, Level: 10}, {SkillID: SkillTFDetoxify, Level: 1}, {SkillID: SkillASEnchantpoison, Level: 5}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestRogueSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		job     int
		skillID uint16
		want    []SkillRequirement
	}{
		{JobArcher, SkillACVulture, []SkillRequirement{{SkillID: SkillACOwl, Level: 3}}},
		{JobRogue, SkillACVulture, []SkillRequirement{}},
		{JobRogueH, SkillACDouble, []SkillRequirement{{SkillID: SkillACVulture, Level: 10}}},
		{JobRogueB, SkillHTRemovetrap, []SkillRequirement{{SkillID: SkillACDouble, Level: 5}}},
		{JobRogue, SkillRGSnatcher, []SkillRequirement{{SkillID: SkillTFSteal, Level: 1}}},
		{JobRogue, SkillRGStealcoin, []SkillRequirement{{SkillID: SkillRGSnatcher, Level: 4}}},
		{JobRogue, SkillRGBackstap, []SkillRequirement{{SkillID: SkillRGStealcoin, Level: 4}}},
		{JobRogue, SkillRGTunneldrive, []SkillRequirement{{SkillID: SkillTFHiding, Level: 1}}},
		{JobRogue, SkillRGRaid, []SkillRequirement{{SkillID: SkillRGTunneldrive, Level: 2}, {SkillID: SkillRGBackstap, Level: 2}}},
		{JobRogue, SkillRGStripweapon, []SkillRequirement{{SkillID: SkillRGStriparmor, Level: 5}}},
		{JobRogue, SkillRGStripshield, []SkillRequirement{{SkillID: SkillRGStriphelm, Level: 5}}},
		{JobRogue, SkillRGStriparmor, []SkillRequirement{{SkillID: SkillRGStripshield, Level: 5}}},
		{JobRogue, SkillRGStriphelm, []SkillRequirement{{SkillID: SkillRGStealcoin, Level: 2}}},
		{JobRogue, SkillRGIntimidate, []SkillRequirement{{SkillID: SkillRGBackstap, Level: 4}, {SkillID: SkillRGRaid, Level: 5}}},
		{JobRogue, SkillRGGraffiti, []SkillRequirement{{SkillID: SkillRGFlaggraffiti, Level: 5}}},
		{JobRogue, SkillRGFlaggraffiti, []SkillRequirement{{SkillID: SkillRGCleaner, Level: 1}}},
		{JobRogue, SkillRGCleaner, []SkillRequirement{{SkillID: SkillRGGangster, Level: 1}}},
		{JobRogue, SkillRGGangster, []SkillRequirement{{SkillID: SkillRGStripshield, Level: 3}}},
		{JobRogue, SkillRGCompulsion, []SkillRequirement{{SkillID: SkillRGGangster, Level: 1}}},
		{JobRogue, SkillRGPlagiarism, []SkillRequirement{{SkillID: SkillRGIntimidate, Level: 5}}},
		{JobRogueH, SkillSTChasewalk, []SkillRequirement{{SkillID: SkillTFHiding, Level: 5}, {SkillID: SkillRGTunneldrive, Level: 3}}},
		{JobRogueH, SkillSTPreserve, []SkillRequirement{{SkillID: SkillRGPlagiarism, Level: 10}}},
		{JobRogueH, SkillSTFullstrip, []SkillRequirement{{SkillID: SkillRGStripweapon, Level: 5}}},
	} {
		if got := SkillRequirementsForJob(tc.job, tc.skillID); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for job %d skill %d = %+v, want %+v", tc.job, tc.skillID, got, tc.want)
		}
	}
}

func TestBlacksmithSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillBSSteel, []SkillRequirement{{SkillID: SkillBSIron, Level: 1}}},
		{SkillBSEnchantedstone, []SkillRequirement{{SkillID: SkillBSIron, Level: 1}}},
		{SkillBSOrideocon, []SkillRequirement{{SkillID: SkillBSEnchantedstone, Level: 1}}},
		{SkillBSSword, []SkillRequirement{{SkillID: SkillBSDagger, Level: 1}}},
		{SkillBSTwohandsword, []SkillRequirement{{SkillID: SkillBSSword, Level: 1}}},
		{SkillBSAxe, []SkillRequirement{{SkillID: SkillBSSword, Level: 2}}},
		{SkillBSMace, []SkillRequirement{{SkillID: SkillBSKnuckle, Level: 1}}},
		{SkillBSKnuckle, []SkillRequirement{{SkillID: SkillBSDagger, Level: 1}}},
		{SkillBSSpear, []SkillRequirement{{SkillID: SkillBSDagger, Level: 2}}},
		{SkillBSFindingore, []SkillRequirement{{SkillID: SkillBSHiltbinding, Level: 1}, {SkillID: SkillBSSteel, Level: 1}}},
		{SkillBSWeaponresearch, []SkillRequirement{{SkillID: SkillBSHiltbinding, Level: 1}}},
		{SkillBSRepairweapon, []SkillRequirement{{SkillID: SkillBSWeaponresearch, Level: 1}}},
		{SkillBSAdrenaline, []SkillRequirement{{SkillID: SkillBSHammerfall, Level: 2}}},
		{SkillBSWeaponperfect, []SkillRequirement{{SkillID: SkillBSWeaponresearch, Level: 2}, {SkillID: SkillBSAdrenaline, Level: 2}}},
		{SkillBSOverthrust, []SkillRequirement{{SkillID: SkillBSAdrenaline, Level: 3}}},
		{SkillBSMaximize, []SkillRequirement{{SkillID: SkillBSWeaponperfect, Level: 3}, {SkillID: SkillBSOverthrust, Level: 2}}},
		{SkillBSAdrenaline2, []SkillRequirement{{SkillID: SkillBSAdrenaline, Level: 5}}},
		{SkillWSMeltdown, []SkillRequirement{{SkillID: SkillBSSkintemper, Level: 3}, {SkillID: SkillBSHiltbinding, Level: 1}, {SkillID: SkillBSWeaponresearch, Level: 5}, {SkillID: SkillBSOverthrust, Level: 3}}},
		{SkillWSCartboost, []SkillRequirement{{SkillID: SkillMCPushcart, Level: 5}, {SkillID: SkillBSHiltbinding, Level: 1}, {SkillID: SkillMCCartrevolution, Level: 1}, {SkillID: SkillMCChangecart, Level: 1}}},
		{SkillWSWeaponrefine, []SkillRequirement{{SkillID: SkillBSWeaponresearch, Level: 10}}},
		{SkillWSCarttermination, []SkillRequirement{{SkillID: SkillMCMammonite, Level: 10}, {SkillID: SkillBSHammerfall, Level: 5}, {SkillID: SkillWSCartboost, Level: 1}}},
		{SkillWSOverthrustmax, []SkillRequirement{{SkillID: SkillBSOverthrust, Level: 5}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestAlchemistSkillRequirementsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    []SkillRequirement
	}{
		{SkillAMPharmacy, []SkillRequirement{{SkillID: SkillAMLearningpotion, Level: 5}}},
		{SkillAMDemonstration, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 4}}},
		{SkillAMAcidterror, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 5}}},
		{SkillAMPotionpitcher, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 3}}},
		{SkillAMCannibalize, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 6}}},
		{SkillAMSpheremine, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 2}}},
		{SkillAMCpWeapon, []SkillRequirement{{SkillID: SkillAMCpArmor, Level: 3}}},
		{SkillAMCpShield, []SkillRequirement{{SkillID: SkillAMCpHelm, Level: 3}}},
		{SkillAMCpArmor, []SkillRequirement{{SkillID: SkillAMCpShield, Level: 3}}},
		{SkillAMCpHelm, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 2}}},
		{SkillAMCallhomun, []SkillRequirement{{SkillID: SkillAMRest, Level: 1}}},
		{SkillAMRest, []SkillRequirement{{SkillID: SkillAMBioethics, Level: 1}}},
		{SkillAMResurrecthomun, []SkillRequirement{{SkillID: SkillAMCallhomun, Level: 1}}},
		{SkillAMTwilight1, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 10}}},
		{SkillAMTwilight2, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 10}}},
		{SkillAMTwilight3, []SkillRequirement{{SkillID: SkillAMPharmacy, Level: 10}}},
		{SkillCRSlimpitcher, []SkillRequirement{{SkillID: SkillAMPotionpitcher, Level: 5}}},
		{SkillCRFullprotection, []SkillRequirement{{SkillID: SkillAMCpWeapon, Level: 5}, {SkillID: SkillAMCpArmor, Level: 5}, {SkillID: SkillAMCpShield, Level: 5}, {SkillID: SkillAMCpHelm, Level: 5}}},
		{SkillCRAciddemonstration, []SkillRequirement{{SkillID: SkillAMDemonstration, Level: 5}, {SkillID: SkillAMAcidterror, Level: 5}}},
	} {
		if got := SkillRequirements[tc.skillID]; !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("requirements for skill %d = %+v, want %+v", tc.skillID, got, tc.want)
		}
	}
}

func TestWeddingSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, skillID := range []uint16{
		SkillWEMale,
		SkillWEFemale,
		SkillWECallpartner,
		SkillWEBaby,
		SkillWECallparent,
		SkillWECallbaby,
	} {
		got, ok := SkillMaxLevel(skillID)
		if !ok || got != 1 {
			t.Fatalf("skill %d max level = %d ok=%t, want 1", skillID, got, ok)
		}
	}
}

func TestWizardSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillWZFirepillar, 10},
		{SkillWZSightrasher, 10},
		{SkillWZFireivy, 0},
		{SkillWZMeteor, 10},
		{SkillWZJupitel, 10},
		{SkillWZVermilion, 10},
		{SkillWZWaterball, 5},
		{SkillWZIcewall, 10},
		{SkillWZFrostnova, 10},
		{SkillWZStormgust, 10},
		{SkillWZEarthspike, 5},
		{SkillWZHeavendrive, 5},
		{SkillWZQuagmire, 5},
		{SkillWZEstimation, 1},
		{SkillWZSightblaster, 1},
		{SkillHWSouldrain, 10},
		{SkillHWMagiccrasher, 1},
		{SkillHWMagicpower, 10},
		{SkillHWNapalmvulcan, 5},
		{SkillHWGanbantein, 1},
		{SkillHWGravitation, 5},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if tc.want == 0 {
			if ok {
				t.Fatalf("skill %d max level = %d ok=%t, want unavailable", tc.skillID, got, ok)
			}
			continue
		}
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestSageSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillSAAdvancedbook, 10},
		{SkillSACastcancel, 5},
		{SkillSAMagicrod, 5},
		{SkillSASpellbreaker, 5},
		{SkillSAFreecast, 10},
		{SkillSAAutospell, 10},
		{SkillSAFlamelauncher, 5},
		{SkillSAFrostweapon, 5},
		{SkillSALightningloader, 5},
		{SkillSASeismicweapon, 5},
		{SkillSADragonology, 5},
		{SkillSAVolcano, 5},
		{SkillSADeluge, 5},
		{SkillSAViolentgale, 5},
		{SkillSALandprotector, 5},
		{SkillSADispell, 5},
		{SkillSAAbracadabra, 10},
		{SkillSAMonocell, 10},
		{SkillSAClasschange, 10},
		{SkillSASummonmonster, 10},
		{SkillSAReverseorcish, 10},
		{SkillSADeath, 10},
		{SkillSAFortune, 10},
		{SkillSATamingmonster, 10},
		{SkillSAQuestion, 10},
		{SkillSAGravity, 10},
		{SkillSALevelup, 10},
		{SkillSAInstantdeath, 10},
		{SkillSAFullrecovery, 10},
		{SkillSAComa, 10},
		{SkillPFHpconversion, 5},
		{SkillPFSoulchange, 1},
		{SkillPFSoulburn, 5},
		{SkillPFMindbreaker, 5},
		{SkillPFMemorize, 1},
		{SkillPFFogwall, 1},
		{SkillPFSpiderweb, 1},
		{SkillPFDoublecasting, 5},
		{SkillSACreatecon, 1},
		{SkillSAElementwater, 1},
		{SkillSAElementground, 1},
		{SkillSAElementfire, 1},
		{SkillSAElementwind, 1},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestKnightSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillKNSpearmastery, 10},
		{SkillKNPierce, 10},
		{SkillKNBrandishspear, 10},
		{SkillKNSpearstab, 10},
		{SkillKNSpearboomerang, 5},
		{SkillKNTwohandquicken, 10},
		{SkillKNAutocounter, 5},
		{SkillKNBowlingbash, 10},
		{SkillKNChargeatk, 1},
		{SkillKNRiding, 1},
		{SkillKNCavaliermastery, 5},
		{SkillKNOnehand, 1},
		{SkillLKSpiralpierce, 5},
		{SkillLKHeadcrush, 5},
		{SkillLKJointbeat, 10},
		{SkillLKAurablade, 5},
		{SkillLKParrying, 10},
		{SkillLKConcentration, 5},
		{SkillLKTensionrelax, 1},
		{SkillLKBerserk, 1},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestCrusaderSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillCRTrust, 10},
		{SkillCRAutoguard, 10},
		{SkillCRShieldcharge, 5},
		{SkillCRShieldboomerang, 5},
		{SkillCRReflectshield, 10},
		{SkillCRHolycross, 10},
		{SkillCRGrandcross, 10},
		{SkillCRDevotion, 5},
		{SkillCRProvidence, 5},
		{SkillCRDefender, 5},
		{SkillCRSpearquicken, 10},
		{SkillCRShrink, 1},
		{SkillPaPressure, 5},
		{SkillPaShieldchain, 5},
		{SkillPaSacrifice, 5},
		{SkillPaGospel, 10},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestPriestSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillALLResurrection, 4},
		{SkillPRMacemastery, 10},
		{SkillPRImpositio, 5},
		{SkillPRSuffragium, 3},
		{SkillPRAspersio, 5},
		{SkillPRBenedictio, 5},
		{SkillPRSanctuary, 10},
		{SkillPRSlowpoison, 4},
		{SkillPRStrecovery, 1},
		{SkillPRKyrie, 10},
		{SkillPRMagnificat, 5},
		{SkillPRGloria, 5},
		{SkillPRLexdivina, 10},
		{SkillPRTurnundead, 10},
		{SkillPRLexaeterna, 1},
		{SkillPRMagnus, 10},
		{SkillPRRedemptio, 1},
		{SkillHPAssumptio, 5},
		{SkillHPBasilica, 5},
		{SkillHPMeditatio, 10},
		{SkillHPManarecharge, 5},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestMonkSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillMOIronhand, 10},
		{SkillMOSpiritsrecovery, 5},
		{SkillMOCallspirits, 5},
		{SkillMOAbsorbspirits, 1},
		{SkillMOTripleattack, 10},
		{SkillMOBodyrelocation, 1},
		{SkillMODodge, 10},
		{SkillMOInvestigate, 5},
		{SkillMOFingeroffensive, 5},
		{SkillMOSteelbody, 5},
		{SkillMOBladestop, 5},
		{SkillMOExplosionspirits, 5},
		{SkillMOExtremityfist, 5},
		{SkillMOChaincombo, 5},
		{SkillMOCombofinish, 5},
		{SkillMOKitranslation, 1},
		{SkillMOBalkyoung, 1},
		{SkillChSoulcollect, 1},
		{SkillChPalmstrike, 5},
		{SkillChTigerfist, 5},
		{SkillChChaincrush, 10},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestHunterSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillHTPower, 1},
		{SkillHTPhantasmic, 1},
		{SkillHTSkidtrap, 5},
		{SkillHTLandmine, 5},
		{SkillHTAnklesnare, 5},
		{SkillHTShockwave, 5},
		{SkillHTSandman, 5},
		{SkillHTFlasher, 5},
		{SkillHTFreezingtrap, 5},
		{SkillHTBlastmine, 5},
		{SkillHTClaymoretrap, 5},
		{SkillHTRemovetrap, 1},
		{SkillHTTalkiebox, 1},
		{SkillHTBeastbane, 10},
		{SkillHTFalcon, 1},
		{SkillHTSteelcrow, 10},
		{SkillHTBlitzbeat, 5},
		{SkillHTDetecting, 4},
		{SkillHTSpringtrap, 5},
		{SkillSNSight, 10},
		{SkillSNFalconassault, 5},
		{SkillSNSharpshooting, 5},
		{SkillSNWindwalk, 10},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestBardDancerSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillBDAdaptation, 1},
		{SkillBDEncore, 1},
		{SkillBDLullaby, 1},
		{SkillBDRichmankim, 5},
		{SkillBDEternalchaos, 1},
		{SkillBDDrumbattlefield, 5},
		{SkillBDRingnibelungen, 5},
		{SkillBDRokisweil, 1},
		{SkillBDIntoabyss, 1},
		{SkillBDSiegfried, 5},
		{SkillBDRagnarok, 0},
		{SkillBaMusicallesson, 10},
		{SkillBaMusicalstrike, 5},
		{SkillBaDissonance, 5},
		{SkillBaFrostjoke, 5},
		{SkillBaWhistle, 10},
		{SkillBaAssassincross, 10},
		{SkillBaPoembragi, 10},
		{SkillBaAppleidun, 10},
		{SkillDCDancinglesson, 10},
		{SkillDCThrowarrow, 5},
		{SkillDCUglydance, 5},
		{SkillDCScream, 5},
		{SkillDCHumming, 10},
		{SkillDCDontforgetme, 10},
		{SkillDCFortunekiss, 10},
		{SkillDCServiceforyou, 10},
		{SkillCGArrowvulcan, 10},
		{SkillCGMoonlit, 5},
		{SkillCGMarionette, 1},
		{SkillCGLongingfreedom, 5},
		{SkillCGHermode, 5},
		{SkillCGTarotcard, 5},
		{SkillBaPangvoice, 1},
		{SkillDCWinkcharm, 1},
		{SkillCGSpecialsinger, 1},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if tc.want == 0 {
			if ok {
				t.Fatalf("skill %d max level = %d ok=%t, want unavailable", tc.skillID, got, ok)
			}
			continue
		}
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestAssassinSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillASRight, 5},
		{SkillASLeft, 5},
		{SkillASKatar, 10},
		{SkillASCloaking, 10},
		{SkillASSonicblow, 10},
		{SkillASGrimtooth, 5},
		{SkillASEnchantpoison, 10},
		{SkillASPoisonreact, 10},
		{SkillASVenomdust, 10},
		{SkillASSplasher, 10},
		{SkillASSonicaccel, 1},
		{SkillASVenomknife, 1},
		{SkillASCKatar, 5},
		{SkillASCEdp, 5},
		{SkillASCBreaker, 10},
		{SkillASCMeteorassault, 10},
		{SkillASCCdp, 1},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestRogueSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillRGSnatcher, 10},
		{SkillRGStealcoin, 10},
		{SkillRGBackstap, 10},
		{SkillRGTunneldrive, 5},
		{SkillRGRaid, 5},
		{SkillRGStripweapon, 5},
		{SkillRGStripshield, 5},
		{SkillRGStriparmor, 5},
		{SkillRGStriphelm, 5},
		{SkillRGIntimidate, 5},
		{SkillRGGraffiti, 1},
		{SkillRGFlaggraffiti, 5},
		{SkillRGCleaner, 1},
		{SkillRGGangster, 1},
		{SkillRGCompulsion, 5},
		{SkillRGPlagiarism, 10},
		{SkillRGCloseconfine, 1},
		{SkillSTChasewalk, 5},
		{SkillSTRejectsword, 5},
		{SkillSTPreserve, 1},
		{SkillSTFullstrip, 5},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestBlacksmithSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillBSIron, 5},
		{SkillBSSteel, 5},
		{SkillBSEnchantedstone, 5},
		{SkillBSOrideocon, 5},
		{SkillBSDagger, 3},
		{SkillBSSword, 3},
		{SkillBSTwohandsword, 3},
		{SkillBSAxe, 3},
		{SkillBSMace, 3},
		{SkillBSKnuckle, 3},
		{SkillBSSpear, 3},
		{SkillBSHiltbinding, 1},
		{SkillBSFindingore, 1},
		{SkillBSWeaponresearch, 10},
		{SkillBSRepairweapon, 1},
		{SkillBSSkintemper, 5},
		{SkillBSHammerfall, 5},
		{SkillBSAdrenaline, 5},
		{SkillBSWeaponperfect, 5},
		{SkillBSOverthrust, 5},
		{SkillBSMaximize, 5},
		{SkillBSAdrenaline2, 1},
		{SkillBSUnfairlytrick, 1},
		{SkillBSGreed, 1},
		{SkillWSMeltdown, 10},
		{SkillWSCreatecoin, 3},
		{SkillWSCreatenugget, 3},
		{SkillWSCartboost, 1},
		{SkillWSSystemcreate, 1},
		{SkillWSWeaponrefine, 10},
		{SkillWSCarttermination, 10},
		{SkillWSOverthrustmax, 5},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func TestAlchemistSkillMaxLevelsMirrorRobrowser(t *testing.T) {
	for _, tc := range []struct {
		skillID uint16
		want    int
	}{
		{SkillAMAxemastery, 10},
		{SkillAMLearningpotion, 10},
		{SkillAMPharmacy, 10},
		{SkillAMDemonstration, 5},
		{SkillAMAcidterror, 5},
		{SkillAMPotionpitcher, 5},
		{SkillAMCannibalize, 5},
		{SkillAMSpheremine, 5},
		{SkillAMCpWeapon, 5},
		{SkillAMCpShield, 5},
		{SkillAMCpArmor, 5},
		{SkillAMCpHelm, 5},
		{SkillAMBioethics, 1},
		{SkillAMCallhomun, 1},
		{SkillAMRest, 1},
		{SkillAMResurrecthomun, 5},
		{SkillAMBerserkpitcher, 1},
		{SkillAMTwilight1, 1},
		{SkillAMTwilight2, 1},
		{SkillAMTwilight3, 1},
		{SkillCRSlimpitcher, 10},
		{SkillCRFullprotection, 5},
		{SkillCRAciddemonstration, 10},
		{SkillCRCultivation, 2},
		{SkillCRAlchemy, 0},
		{SkillCRSynthesispotion, 0},
	} {
		got, ok := SkillMaxLevel(tc.skillID)
		if tc.want == 0 {
			if ok {
				t.Fatalf("skill %d max level = %d ok=%t, want unavailable", tc.skillID, got, ok)
			}
			continue
		}
		if !ok || got != tc.want {
			t.Fatalf("skill %d max level = %d ok=%t, want %d", tc.skillID, got, ok, tc.want)
		}
	}
}

func containsSkillID(skills []uint16, skillID uint16) bool {
	for _, id := range skills {
		if id == skillID {
			return true
		}
	}
	return false
}
