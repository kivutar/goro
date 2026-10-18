package game

import (
	"testing"

	"github.com/kivutar/goro/db"
	"github.com/kivutar/goro/res"
)

func TestLegacySkillUnitTrapRSMModelsLoadFromRealDataWhenConfigured(t *testing.T) {
	manager := realDataManager(t)
	checked := 0
	for unitID, spec := range db.SkillUnitModels {
		if len(spec.FallbackModelPaths) == 0 {
			continue
		}
		checked++
		var rsm *res.RSM
		var loadedPath string
		paths := append([]string{spec.ModelPath}, spec.FallbackModelPaths...)
		for _, modelPath := range paths {
			loaded, err := loadRSMModel(manager, modelPath)
			if err != nil {
				continue
			}
			rsm = loaded
			loadedPath = modelPath
			break
		}
		if rsm == nil {
			t.Fatalf("load trap model unit=%d paths=%v", unitID, paths)
		}
		if len(rsm.Nodes) == 0 {
			t.Fatalf("trap model unit=%d model=%s has no nodes", unitID, loadedPath)
		}
	}
	if checked == 0 {
		t.Fatal("no legacy trap RSM models were checked")
	}
}
