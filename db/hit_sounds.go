package db

func WeaponHitSounds(weaponType int) []string {
	if sounds, ok := weaponHitSoundTable[weaponType]; ok {
		return sounds
	}
	return weaponHitSoundTable[MaxWeaponType]
}

func WeaponAttackSounds(weaponType int) []string {
	if sounds, ok := weaponAttackSoundTable[weaponType]; ok {
		return sounds
	}
	return weaponAttackSoundTable[MaxWeaponType]
}

func JobHitSounds(job int) []string {
	if sounds, ok := jobHitSoundTable[job]; ok {
		return sounds
	}
	return jobHitSoundTable[JobNovice]
}

var weaponHitSoundTable = map[int][]string{
	WeaponNone:         {"_hit_fist1.wav", "_hit_fist2.wav", "_hit_fist3.wav", "_hit_fist4.wav"},
	WeaponShortsword:   {"_hit_sword.wav"},
	WeaponSword:        {"_hit_sword.wav"},
	WeaponTwoHandSword: {"_hit_sword.wav"},
	WeaponSpear:        {"_hit_spear.wav"},
	WeaponTwoHandSpear: {"_hit_spear.wav"},
	WeaponAxe:          {"_hit_axe.wav"},
	WeaponTwoHandAxe:   {"_hit_axe.wav"},
	WeaponMace:         {"_hit_mace.wav"},
	WeaponTwoHandMace:  {"_hit_mace.wav"},
	WeaponRod:          {"_hit_rod.wav"},
	WeaponBow:          {"_hit_arrow.wav"},
	WeaponKnuckle:      {"_HIT_FIST2.wav"},
	WeaponInstrument:   {"_hit_mace.wav"},
	WeaponWhip:         {"_hit_mace.wav"},
	WeaponBook:         {"_hit_mace.wav"},
	WeaponKatar:        {"_hit_mace.wav"},
	WeaponGunHandgun:   {"_hit_권총.wav"},
	WeaponGunRifle:     {"_hit_라이플.wav"},
	WeaponGunGatling:   {"_hit_개틀링한발.wav"},
	WeaponGunShotgun:   {"_hit_샷건.wav"},
	WeaponGunGrenade:   {"_hit_그레네이드런쳐.wav"},
	WeaponShuriken:     {"_hit_mace.wav"},
	WeaponTwoHandRod:   {"_hit_rod.wav"},
	24:                 {"_hit_fist4.wav"},
	25:                 {"_hit_sword.wav"},
	26:                 {"_hit_sword.wav"},
	27:                 {"_hit_axe.wav"},
	28:                 {"_hit_sword.wav"},
	29:                 {"_hit_sword.wav"},
	30:                 {"_hit_sword.wav"},
	MaxWeaponType:      {"_hit_mace.wav"},
}

var weaponAttackSoundTable = map[int][]string{
	WeaponNone:                 {"attack_fist.wav"},
	WeaponShortsword:           {"attack_short_sword.wav", "attack_short_sword_.wav"},
	WeaponSword:                {"attack_sword.wav"},
	WeaponTwoHandSword:         {"attack_twohand_sword.wav"},
	WeaponSpear:                {"attack_spear.wav"},
	WeaponTwoHandSpear:         {"attack_spear.wav"},
	WeaponAxe:                  {"attack_axe.wav"},
	WeaponTwoHandAxe:           {"attack_axe.wav"},
	WeaponMace:                 {"attack_mace.wav"},
	WeaponTwoHandMace:          {"attack_mace.wav"},
	WeaponRod:                  {"attack_rod.wav"},
	WeaponBow:                  {"attack_bow1.wav", "attack_bow2.wav"},
	WeaponKnuckle:              {"attack_fist.wav"},
	WeaponInstrument:           {"attack_mace.wav"},
	WeaponWhip:                 {"attack_whip.wav"},
	WeaponBook:                 {"attack_book.wav"},
	WeaponKatar:                {"attack_katar.wav"},
	WeaponGunHandgun:           nil,
	WeaponGunRifle:             nil,
	WeaponGunGatling:           nil,
	WeaponGunShotgun:           nil,
	WeaponGunGrenade:           nil,
	WeaponShuriken:             {"attack_sword.wav"},
	WeaponTwoHandRod:           {"attack_rod.wav"},
	24:                         {"attack_fist.wav"},
	WeaponShortswordShortsword: {"attack_mace.wav"},
	WeaponSwordSword:           {"attack_mace.wav"},
	WeaponAxeAxe:               {"attack_mace.wav"},
	WeaponShortswordSword:      {"attack_mace.wav"},
	WeaponShortswordAxe:        {"attack_mace.wav"},
	WeaponSwordAxe:             {"attack_mace.wav"},
	MaxWeaponType:              {"attack_fist.wav"},
}

var jobHitSoundTable = map[int][]string{}

func init() {
	setJobHitSounds([]string{"player_clothes.wav"},
		JobNovice, JobMagician, JobAcolyte, JobMerchant, JobPriest, JobWizard,
		JobBlacksmith, JobSage, JobAlchemist, JobSuperNovice, JobLinker,
		JobMarried, JobXmas, JobSummer,
		JobNoviceH, JobMagicianH, JobAcolyteH, JobMerchantH, JobPriestH,
		JobWizardH, JobBlacksmithH, JobSageH, JobAlchemistH, JobSuperNoviceB,
		JobNoviceB, JobMagicianB, JobAcolyteB, JobMerchantB, JobPriestB,
		JobWizardB, JobBlacksmithB, JobSageB, JobAlchemistB)
	setJobHitSounds([]string{"player_metal.wav"},
		JobSwordman, JobKnight, JobKnight2, JobCrusader, JobMonk, JobCrusader2,
		JobStar, JobStar2, JobSwordmanH, JobKnightH, JobKnight2H, JobCrusaderH,
		JobMonkH, JobCrusader2H, JobSwordmanB, JobKnightB, JobKnight2B,
		JobCrusaderB, JobMonkB, JobCrusader2B)
	setJobHitSounds([]string{"player_wooden_male.wav"},
		JobArcher, JobThief, JobHunter, JobAssassin, JobRogue, JobBard,
		JobDancer, JobGunslinger, JobNinja, JobTaekwon, JobArcherH, JobThiefH,
		JobHunterH, JobAssassinH, JobRogueH, JobBardH, JobDancerH, JobArcherB,
		JobThiefB, JobHunterB, JobAssassinB, JobRogueB, JobBardB, JobDancerB)
}

func setJobHitSounds(sounds []string, jobs ...int) {
	for _, job := range jobs {
		jobHitSoundTable[job] = sounds
	}
}
