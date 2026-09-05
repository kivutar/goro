package res

import (
	"testing"

	"github.com/kivutar/goro/db"
)

func TestSkillTreePositionsRealWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	positions, ok := manager.SkillTreePositions(db.JobAlchemist)
	if !ok {
		t.Fatal("alchemist skill-tree positions were not loaded")
	}
	if got := positions[int(db.SkillAMPharmacy)]; got != 7 {
		t.Fatalf("Pharmacy position = %d, want 7", got)
	}
}
