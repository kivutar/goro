package db

import "testing"

func TestSkillAttackRangeMirrorsRobrowserCompanionRows(t *testing.T) {
	tests := []struct {
		name    string
		skillID uint16
		level   int
		want    int
	}{
		{"vanilmirth caprice", SkillHvanCaprice, 5, 9},
		{"filir sbr44", SkillHfliSbr44, 1, 9},
		{"homunculus s needle", SkillMhStahlHorn, 10, 9},
		{"mercenary devotion", SkillMlDevotion, 4, 10},
		{"mercenary lex divina", SkillMerLexdivina, 1, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SkillAttackRange(tt.skillID, tt.level)
			if !ok || got != tt.want {
				t.Fatalf("SkillAttackRange(%d, %d) = %d, %t; want %d, true", tt.skillID, tt.level, got, ok, tt.want)
			}
		})
	}
}

func TestSkillAttackRangeRejectsUnknownOrInvalidLevel(t *testing.T) {
	if _, ok := SkillAttackRange(SkillHvanCaprice, 0); ok {
		t.Fatal("level 0 should not resolve")
	}
	if _, ok := SkillAttackRange(SkillHvanCaprice, 6); ok {
		t.Fatal("level beyond range table should not resolve")
	}
}
