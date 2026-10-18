package db

const (
	SkillUnitUsedTraps uint16 = 140
	// Keeps trap RSMs near a one-cell footprint after Goro's normalized map scaling.
	skillUnitTrapScale = 0.15
)

type SkillUnitModelSpec struct {
	ModelPath          string
	FallbackModelPaths []string
	TriggerEffectID    int
	Scale              float64
	FixedFrame         int
	HasFixedFrame      bool
}

func rsmTrapModelSpec(robrPath string, fallbackPaths []string, triggerEffectID int) SkillUnitModelSpec {
	return SkillUnitModelSpec{
		ModelPath:          robrPath,
		FallbackModelPaths: fallbackPaths,
		TriggerEffectID:    triggerEffectID,
		Scale:              skillUnitTrapScale,
		FixedFrame:         3,
		HasFixedFrame:      true,
	}
}

func trapModelSpec(robrPath, classicPath string, triggerEffectID int) SkillUnitModelSpec {
	return rsmTrapModelSpec(robrPath, []string{classicPath}, triggerEffectID)
}

func robrTrapModelSpec(robrPath string, triggerEffectID int) SkillUnitModelSpec {
	return rsmTrapModelSpec(robrPath, nil, triggerEffectID)
}

// Primary paths mirror roBrowser DB/Skills/SkillUnit.js ef_trap_* aliases.
// Legacy fallback paths mirror the 2008 client's CSkill::GetTrapModelName table.
// Paths are relative to data\model\, matching RSW model filenames.
var SkillUnitModels = map[uint16]SkillUnitModelSpec{
	143: trapModelSpec("외부소품\\트랩03_3.rsm", "effect\\trap08.rsm", effectBlastMineBomb), // UNT_BLASTMINE -> ef_trap_03_3
	144: trapModelSpec("외부소품\\트랩02.rsm", "effect\\trap01.rsm", 0),                     // UNT_SKIDTRAP -> ef_trap_02
	145: trapModelSpec("외부소품\\트랩01.rsm", "effect\\trap03.rsm", 0),                     // UNT_ANKLESNARE -> ef_trap_01
	147: trapModelSpec("외부소품\\트랩03.rsm", "effect\\trap02.rsm", 0),                     // UNT_LANDMINE -> ef_trap_03
	148: trapModelSpec("외부소품\\트랩03_6.rsm", "effect\\trap04.rsm", 0),                   // UNT_SHOCKWAVE -> ef_trap_03_6
	149: trapModelSpec("외부소품\\트랩03_4.rsm", "effect\\trap05.rsm", effectSandman),       // UNT_SANDMAN -> ef_trap_03_4
	150: trapModelSpec("외부소품\\트랩03_5.rsm", "effect\\trap06.rsm", effectFlasher),       // UNT_FLASHER -> ef_trap_03_5
	151: trapModelSpec("외부소품\\트랩03_2.rsm", "effect\\trap07.rsm", effectFreezingTrap),  // UNT_FREEZINGTRAP -> ef_trap_03_2
	152: trapModelSpec("외부소품\\트랩04.rsm", "effect\\trap09.rsm", effectClaymore),        // UNT_CLAYMORETRAP -> ef_trap_04
	153: trapModelSpec("외부소품\\트랩05.rsm", "effect\\trap10.rsm", 0),                     // UNT_TALKIEBOX -> ef_trap_05
	210: robrTrapModelSpec("event\\3차트랩_변화01.rsm", 0),                                 // UNT_MAGENTATRAP -> ef_trap_3_magenta
	211: robrTrapModelSpec("event\\3차트랩_변수01.rsm", 0),                                 // UNT_COBALTTRAP -> ef_trap_3_cobalt
	212: robrTrapModelSpec("event\\3차트랩_변지01.rsm", 0),                                 // UNT_MAIZETRAP -> ef_trap_3_maze
	213: robrTrapModelSpec("event\\3차트랩_변풍01.rsm", 0),                                 // UNT_VERDURETRAP -> ef_trap_3_verdure
	214: robrTrapModelSpec("event\\3차트랩_화01.rsm", 0),                                  // UNT_FIRINGTRAP -> ef_trap_3_fire
	215: robrTrapModelSpec("event\\3차트랩_수01.rsm", 0),                                  // UNT_ICEBOUNDTRAP -> ef_trap_3_ice
	216: robrTrapModelSpec("event\\3차트랩_풍01.rsm", 0),                                  // UNT_ELECTRICSHOCKER -> ef_trap_3_shock
	217: robrTrapModelSpec("event\\3차트랩_지01.rsm", 0),                                  // UNT_CLUSTERBOMB -> ef_trap_3_cluster
	229: robrTrapModelSpec("event\\3차트랩_가시01.rsm", 0),                                 // UNT_THORNS_TRAP -> ef_trap_3_thorn
}
